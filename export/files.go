package export

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/client"
	"github.com/thecoooler/blitz-client/internal/glob"
	"github.com/thecoooler/blitz-client/internal/pathx"
	"github.com/thecoooler/blitz-client/internal/rs"
)

// ClientFileType is a category of unpacked client files.
type ClientFileType string

const (
	FileTypeAll     ClientFileType = "all"
	FileTypeConfigs ClientFileType = "configs"
	FileTypeImages  ClientFileType = "images"
	FileTypeModels  ClientFileType = "models"
	FileTypeSounds  ClientFileType = "sounds"
	FileTypeFonts   ClientFileType = "fonts"
	FileTypeShaders ClientFileType = "shaders"
	FileTypeOther   ClientFileType = "other"
)

// ParseClientFileType parses a category name case-insensitively.
func ParseClientFileType(value string) (ClientFileType, error) {
	switch kind := ClientFileType(asciiLower(strings.TrimSpace(value))); kind {
	case FileTypeAll, FileTypeConfigs, FileTypeImages, FileTypeModels, FileTypeSounds, FileTypeFonts, FileTypeShaders, FileTypeOther:
		return kind, nil
	}
	return "", blitz.NewError("FILE_EXPORT_TYPE_INVALID", fmt.Sprintf(
		"unknown file type %s; expected all, configs, images, models, sounds, fonts, shaders, other", rs.Quote(value)))
}

var fileTypesByExtension = func() map[string]ClientFileType {
	out := map[string]ClientFileType{}
	for kind, extensions := range map[ClientFileType]string{
		FileTypeConfigs: "yaml yml xml xsd txt json ini cfg toml csv tab actions style material html htm",
		FileTypeImages:  "webp png jpg jpeg dds pvr tex tga bmp gif svg ktx ktx2",
		FileTypeModels:  "sc2 scg anim animation model mesh heightmap mkm lka",
		FileTypeSounds:  "bnk pck wem wav ogg mp3 flac fsb",
		FileTypeFonts:   "ttf otf fnt woff woff2",
		FileTypeShaders: "sl slh glsl hlsl vert frag metal",
	} {
		for _, ext := range strings.Fields(extensions) {
			out[ext] = kind
		}
	}
	return out
}()

// FileTypeOf returns the category of the unpacked filename; a final .dvpl
// suffix is ignored.
func FileTypeOf(path string) ClientFileType {
	ext, _ := pathx.Extension(unpackedPath(path))
	if kind, ok := fileTypesByExtension[asciiLower(ext)]; ok {
		return kind
	}
	return FileTypeOther
}

// ClientFilesExportOptions configures ExportClientFiles.
type ClientFilesExportOptions struct {
	GameDir string
	// CacheDir overrides the standard packs folder. A missing explicit folder is an error.
	CacheDir string
	// NoCache disables the cache overlay entirely.
	NoCache bool
	// DryRun counts selected stored files without reading, decoding, or writing them.
	DryRun    bool
	OutputDir string
	// Types selects categories; empty or containing All selects every file.
	Types []ClientFileType
	// Exclude holds glob patterns matched against unpacked paths relative to Data/packs.
	Exclude []string
}

// NewClientFilesExportOptions returns options that export every file.
func NewClientFilesExportOptions(outputDir string) ClientFilesExportOptions {
	return ClientFilesExportOptions{OutputDir: outputDir, Types: []ClientFileType{FileTypeAll}, Exclude: []string{}}
}

// ClientFilesExportReport summarizes a file export.
type ClientFilesExportReport struct {
	DryRun              bool             `json:"dry_run"`
	OutputDir           string           `json:"output_dir"`
	ResourceDir         string           `json:"resource_dir"`
	CacheDir            *string          `json:"cache_dir"`
	Types               []ClientFileType `json:"types"`
	Exclude             []string         `json:"exclude"`
	ExcludedFileCount   int              `json:"excluded_file_count"`
	FileCount           int              `json:"file_count"`
	WrittenCount        int              `json:"written_count"`
	UnchangedCount      int              `json:"unchanged_count"`
	CacheFileCount      int              `json:"cache_file_count"`
	DVPLCount           int              `json:"dvpl_count"`
	SkippedSymlinkCount int              `json:"skipped_symlink_count"`
	// SourceBytes is the size of selected source files on disk, including DVPL containers.
	SourceBytes uint64 `json:"source_bytes"`
	// OutputBytes is the decoded size; zero in dry-run mode.
	OutputBytes    uint64  `json:"output_bytes"`
	ElapsedSeconds float64 `json:"elapsed_seconds"`
}

type sourceFile struct {
	path      string
	packed    bool
	fromCache bool
}

// ExportClientFiles merges Data and packs by unpacked relative path; the cache
// wins regardless of modification time. Plain files take precedence over their
// .dvpl sibling within each source tree, and only one selected source is read
// per output file. Existing identical output is kept and unrelated destination
// files are retained. Failure stops the export.
func ExportClientFiles(options ClientFilesExportOptions) (*ClientFilesExportReport, error) {
	return ExportClientFilesWithProgress(options, nil)
}

// ExportClientFilesWithProgress performs the same export, calling progress
// after discovery and after each processed file. A progress error stops the
// export; files already written are retained.
func ExportClientFilesWithProgress(options ClientFilesExportOptions, progress func(*ClientFilesExportReport) error) (*ClientFilesExportReport, error) {
	started := time.Now()
	if progress == nil {
		progress = func(*ClientFilesExportReport) error { return nil }
	}
	if options.NoCache && options.CacheDir != "" {
		return nil, blitz.NewError("FILE_EXPORT_OPTIONS_INVALID", "cache_dir and no_cache cannot be combined")
	}
	exclusions := make([]*glob.Pattern, 0, len(options.Exclude))
	for _, value := range options.Exclude {
		if value == "" {
			return nil, blitz.NewError("FILE_EXPORT_PATTERN_INVALID", "exclude pattern must not be empty")
		}
		pattern, err := glob.New(value)
		if err != nil {
			return nil, blitz.NewError("FILE_EXPORT_PATTERN_INVALID",
				fmt.Sprintf("invalid exclude pattern %s: %v", rs.Quote(value), err))
		}
		exclusions = append(exclusions, pattern)
	}
	game, err := client.ResolveGameDir(options.GameDir)
	if err != nil {
		return nil, err
	}
	resourceDir, err := canonicalDirectory(game.ResourceDir)
	if err != nil {
		return nil, err
	}
	var cacheDir *string
	switch {
	case options.NoCache:
	case options.CacheDir != "":
		path, err := canonicalDirectory(options.CacheDir)
		if err != nil {
			return nil, err
		}
		cacheDir = &path
	case game.CacheDir != nil && isDir(*game.CacheDir):
		path, err := canonicalDirectory(*game.CacheDir)
		if err != nil {
			return nil, err
		}
		cacheDir = &path
	}
	outputDir, err := resolveOutputPath(options.OutputDir)
	if err != nil {
		return nil, err
	}
	sources := []string{resourceDir}
	if cacheDir != nil {
		sources = append(sources, *cacheDir)
	}
	for _, source := range sources {
		if pathx.HasPrefix(outputDir, source) || pathx.HasPrefix(source, outputDir) {
			return nil, blitz.ErrorAt("FILE_EXPORT_OUTPUT_INVALID", "output and source directories must not overlap", outputDir)
		}
	}

	files := map[string]sourceFile{}
	skipped := 0
	if err := collectFiles(resourceDir, options, false, files, &skipped); err != nil {
		return nil, err
	}
	if cacheDir != nil {
		if err := collectFiles(*cacheDir, options, true, files, &skipped); err != nil {
			return nil, err
		}
	}
	beforeExclusions := len(files)
	for relative := range files {
		matchPath := filepath.ToSlash(relative)
		for _, pattern := range exclusions {
			if pattern.Match(matchPath) {
				delete(files, relative)
				break
			}
		}
	}
	relatives := make([]string, 0, len(files))
	for relative := range files {
		relatives = append(relatives, relative)
	}
	slices.SortFunc(relatives, pathx.Compare)

	types := options.Types
	if len(types) == 0 {
		types = []ClientFileType{FileTypeAll}
	}
	exclude := options.Exclude
	if exclude == nil {
		exclude = []string{}
	}
	report := &ClientFilesExportReport{
		DryRun:              options.DryRun,
		OutputDir:           outputDir,
		ResourceDir:         resourceDir,
		CacheDir:            cacheDir,
		Types:               types,
		Exclude:             exclude,
		ExcludedFileCount:   beforeExclusions - len(files),
		FileCount:           len(files),
		SkippedSymlinkCount: skipped,
	}
	if options.DryRun {
		for _, relative := range relatives {
			source := files[relative]
			info, err := os.Stat(source.path)
			if err != nil {
				return nil, blitz.IOError("could not inspect client file", source.path, err)
			}
			report.SourceBytes += uint64(info.Size())
			report.DVPLCount += boolCount(source.packed)
			report.CacheFileCount += boolCount(source.fromCache)
		}
		report.ElapsedSeconds = time.Since(started).Seconds()
		if err := progress(report); err != nil {
			return nil, err
		}
		return report, nil
	}
	if err := os.MkdirAll(report.OutputDir, 0o777); err != nil {
		return nil, blitz.IOError("could not create export directory", report.OutputDir, err)
	}
	if err := progress(report); err != nil {
		return nil, err
	}
	for _, relative := range relatives {
		source := files[relative]
		destination := filepath.Join(report.OutputDir, relative)
		if err := rejectOutputSymlinks(destination, report.OutputDir); err != nil {
			return nil, err
		}
		data, err := os.ReadFile(source.path)
		if err != nil {
			return nil, blitz.IOError("could not read client file", source.path, err)
		}
		report.SourceBytes += uint64(len(data))
		if source.packed {
			decoded, kind, err := client.DecodeClientAsset(data, source.path)
			if err != nil {
				return nil, err
			}
			if !kind.DVPL {
				return nil, blitz.ErrorAt("DVPL_INVALID", "file with .dvpl extension has no DVPL footer", source.path)
			}
			report.DVPLCount++
			data = decoded
		}
		report.OutputBytes += uint64(len(data))
		report.CacheFileCount += boolCount(source.fromCache)
		existing, err := os.ReadFile(destination)
		switch {
		case err == nil && bytes.Equal(existing, data):
			report.UnchangedCount++
		case err != nil && !errors.Is(err, fs.ErrNotExist):
			return nil, blitz.IOError("could not read existing export file", destination, err)
		default:
			if err := os.MkdirAll(filepath.Dir(destination), 0o777); err != nil {
				return nil, blitz.IOError("could not create resource directory", destination, err)
			}
			if err := atomicWrite(destination, data); err != nil {
				return nil, err
			}
			report.WrittenCount++
		}
		report.ElapsedSeconds = time.Since(started).Seconds()
		if err := progress(report); err != nil {
			return nil, err
		}
	}
	report.ElapsedSeconds = time.Since(started).Seconds()
	return report, nil
}

func boolCount(value bool) int {
	if value {
		return 1
	}
	return 0
}

func isPacked(path string) bool {
	ext, ok := pathx.Extension(path)
	return ok && strings.EqualFold(ext, "dvpl")
}

func unpackedPath(path string) string {
	if isPacked(path) {
		return pathx.WithoutExtension(path)
	}
	return path
}

func collectFiles(root string, options ClientFilesExportOptions, fromCache bool, files map[string]sourceFile, skipped *int) error {
	local := map[string]sourceFile{}
	// spellings detects two files that differ only in case on a filesystem
	// that would treat them as one name. A plain file and its .dvpl sibling
	// are the same resource; plain and packed names are tracked separately.
	plainSpellings := map[string]string{}
	packedSpellings := map[string]string{}
	directories := []string{root}
	selectAll := len(options.Types) == 0 || slices.Contains(options.Types, FileTypeAll)
	for len(directories) > 0 {
		directory := directories[len(directories)-1]
		directories = directories[:len(directories)-1]
		entries, err := readDirUnsorted(directory)
		if err != nil {
			return blitz.IOError("could not scan client directory", directory, err)
		}
		for _, entry := range entries {
			path := filepath.Join(directory, entry.Name())
			mode := entry.Type()
			if mode&fs.ModeSymlink != 0 {
				*skipped++
				continue
			}
			if mode.IsDir() {
				directories = append(directories, path)
				continue
			}
			if !mode.IsRegular() {
				continue
			}
			if !selectAll && !slices.Contains(options.Types, FileTypeOf(path)) {
				continue
			}
			packed := isPacked(path)
			relative, _ := filepath.Rel(root, path)
			relative = unpackedPath(relative)
			logical, _ := client.LogicalResourceKey(filepath.ToSlash(relative), false)
			folded, _ := client.LogicalResourceKey(filepath.ToSlash(relative), client.CaseInsensitiveFS())
			spellings := plainSpellings
			if packed {
				spellings = packedSpellings
			}
			if previous, ok := spellings[folded]; ok && previous != logical {
				return blitz.ErrorAt("CLIENT_RESOURCE_COLLISION",
					"client files differ only in case: "+previous+" and "+logical, path)
			}
			spellings[folded] = logical
			if previous, ok := local[relative]; !ok || (previous.packed && !packed) {
				local[relative] = sourceFile{path: path, packed: packed, fromCache: fromCache}
			}
		}
	}
	for relative, source := range local {
		files[relative] = source
	}
	return nil
}

func readDirUnsorted(directory string) ([]fs.DirEntry, error) {
	handle, err := os.Open(directory)
	if err != nil {
		return nil, err
	}
	defer handle.Close()
	return handle.ReadDir(-1)
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func canonicalDirectory(path string) (string, error) {
	if !isDir(path) {
		return "", blitz.ErrorAt("FILE_EXPORT_SOURCE_NOT_FOUND", "source directory does not exist", path)
	}
	canonical, err := pathx.Canonicalize(path)
	if err != nil {
		return "", blitz.IOError("could not resolve source directory", path, err)
	}
	return canonical, nil
}

// resolveOutputPath resolves the existing ancestor before creating anything,
// including symlink aliases.
func resolveOutputPath(path string) (string, error) {
	absolute, err := absoluteKeepingParents(path)
	if err != nil {
		return "", blitz.IOError("could not resolve output path", path, err)
	}
	ancestor := absolute
	var missing []string
	for {
		if _, err := os.Stat(ancestor); err == nil {
			break
		}
		name := filepath.Base(ancestor)
		parent := pathx.Parent(ancestor)
		if name == ".." || parent == "" || parent == ancestor {
			return "", blitz.ErrorAt("FILE_EXPORT_OUTPUT_INVALID", "cannot resolve output directory", path)
		}
		missing = append(missing, name)
		ancestor = parent
	}
	if !isDir(ancestor) {
		return "", blitz.ErrorAt("FILE_EXPORT_OUTPUT_INVALID", "output path has a file in place of a directory", ancestor)
	}
	resolved, err := pathx.Canonicalize(ancestor)
	if err != nil {
		return "", blitz.IOError("could not resolve output directory", ancestor, err)
	}
	for index := len(missing) - 1; index >= 0; index-- {
		resolved = filepath.Join(resolved, missing[index])
	}
	return resolved, nil
}

// absoluteKeepingParents mirrors std::path::absolute: it prefixes the working
// directory and drops "." components but keeps "..".
func absoluteKeepingParents(path string) (string, error) {
	if filepath.Separator != '/' {
		return filepath.Abs(path)
	}
	if !filepath.IsAbs(path) {
		directory, err := os.Getwd()
		if err != nil {
			return "", err
		}
		path = directory + string(filepath.Separator) + path
	}
	var parts []string
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part != "" && part != "." {
			parts = append(parts, part)
		}
	}
	return filepath.FromSlash("/" + strings.Join(parts, "/")), nil
}

func rejectOutputSymlinks(path, root string) error {
	for ancestor := path; ancestor != root; {
		info, err := os.Lstat(ancestor)
		switch {
		case err == nil && info.Mode()&fs.ModeSymlink != 0:
			return blitz.ErrorAt("FILE_EXPORT_OUTPUT_INVALID", "export path contains a symlink", ancestor)
		case err != nil && !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, os.ErrNotExist):
			return blitz.IOError("could not inspect export path", ancestor, err)
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			break
		}
		ancestor = parent
	}
	return nil
}

func asciiLower(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + 'a' - 'A'
		}
		return r
	}, value)
}

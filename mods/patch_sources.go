package mods

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/internal/pathx"
)

// loadPatches resolves manifest patch sources into self-contained,
// per-resource operations. generated substitutes the content of mod.patch.
func loadPatches(modDir string, sources []string, generated *string) ([]ModFile, error) {
	if len(sources) == 0 {
		return nil, nil
	}
	root, err := pathx.Canonicalize(dirOrDot(modDir))
	if err != nil {
		return nil, blitz.IOError("could not resolve mod directory", modDir, err)
	}
	generatedPath := filepath.Join(root, "mod.patch")
	seen := map[string]bool{}
	var operations []ModFile
	for _, source := range sources {
		relative := "."
		if source != "." {
			if relative, err = safeRelativePath(source); err != nil {
				return nil, blitz.NewError("MANIFEST_INVALID_PATCH_PATH", err.Error())
			}
		}
		sourcePath := pathx.Join(root, filepath.FromSlash(relative))
		var files []string
		if err := collectSource(sourcePath, root, generated != nil, &files); err != nil {
			return nil, err
		}
		slices.SortFunc(files, pathx.Compare)
		if len(files) == 0 {
			return nil, blitz.ErrorAt("PATCH_DIRECTORY_EMPTY", "directory contains no .patch or .diff files", sourcePath)
		}
		for _, file := range files {
			isGenerated := generated != nil && file == generatedPath
			canonical := file
			if !isGenerated {
				if canonical, err = checkedPath(file, root); err != nil {
					return nil, err
				}
			}
			if seen[canonical] {
				continue
			}
			seen[canonical] = true
			var text string
			if isGenerated {
				text = *generated
			} else {
				data, err := os.ReadFile(canonical)
				if err == nil && !utf8Valid(data) {
					err = errInvalidUTF8
				}
				if err != nil {
					return nil, blitz.IOError("could not read UTF-8 patch", canonical, err)
				}
				text = string(data)
			}
			relativeName, _ := filepath.Rel(root, canonical)
			relativeName = strings.ReplaceAll(relativeName, `\`, "/")
			sections, err := splitPatch(text)
			if err != nil {
				typed := *blitz.AsError(err)
				typed.Path = canonical
				return nil, &typed
			}
			for _, section := range sections {
				patch, diff := relativeName, section.diff
				operations = append(operations, ModFile{
					Name: section.path, Action: ActionPatch, Patch: &patch, PatchDiff: &diff,
				})
			}
		}
	}
	return operations, nil
}

func checkedPath(p, root string) (string, error) {
	canonical, err := pathx.Canonicalize(p)
	if err != nil {
		return "", blitz.IOError("could not resolve patch source", p, err)
	}
	if !pathx.HasPrefix(canonical, root) {
		return "", blitz.ErrorAt("PATCH_OUTSIDE_MOD_DIR", "patch source escapes the mod directory", p)
	}
	return canonical, nil
}

func collectSource(p, root string, generated bool, files *[]string) error {
	generatedPath := filepath.Join(root, "mod.patch")
	if generated && p == generatedPath {
		if info, err := os.Lstat(p); err == nil && !info.Mode().IsRegular() {
			return blitz.ErrorAt("PATCH_INVALID_SOURCE", "generated mod.patch must be a regular file", p)
		}
		*files = append(*files, p)
		return nil
	}
	canonical, err := checkedPath(p, root)
	if err != nil {
		return err
	}
	switch {
	case isFile(canonical):
		*files = append(*files, canonical)
	case isDir(canonical):
		if err := collectDirectory(canonical, root, files); err != nil {
			return err
		}
		if generated && canonical == root && !slices.Contains(*files, generatedPath) {
			*files = append(*files, generatedPath)
		}
	default:
		return blitz.ErrorAt("PATCH_INVALID_SOURCE", "patch source must be a file or directory", p)
	}
	return nil
}

func collectDirectory(dir, root string, files *[]string) error {
	entries, err := readDirUnsorted(dir)
	if err != nil {
		return blitz.IOError("could not scan patch directory", dir, err)
	}
	for _, entry := range entries {
		entryPath := filepath.Join(dir, entry.Name())
		ext, _ := pathx.Extension(entryPath)
		switch {
		case entry.IsDir():
			if err := collectDirectory(entryPath, root, files); err != nil {
				return err
			}
		case (strings.EqualFold(ext, "patch") || strings.EqualFold(ext, "diff")) && isASCII(ext) && isFile(entryPath):
			canonical, err := checkedPath(entryPath, root)
			if err != nil {
				return err
			}
			*files = append(*files, canonical)
		}
		// Directory symlinks are not followed, avoiding recursive cycles.
	}
	return nil
}

func isASCII(text string) bool {
	for _, c := range text {
		if c > 127 {
			return false
		}
	}
	return true
}

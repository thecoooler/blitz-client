//! Export the effective resource tree, unpacking DVPL without format conversion.
use crate::client::resolve_game_dir;
use crate::resources::{ContainerKind, decode_dvpl};
use crate::{Error, Result, io_error};
use glob::{MatchOptions, Pattern};
use serde::Serialize;
use std::collections::BTreeMap;
use std::fs;
use std::path::{Path, PathBuf};
use std::str::FromStr;
use std::time::Instant;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
#[serde(rename_all = "lowercase")]
pub enum ClientFileType {
    All,
    Configs,
    Images,
    Models,
    Sounds,
    Fonts,
    Shaders,
    Other,
}

impl FromStr for ClientFileType {
    type Err = Error;

    fn from_str(value: &str) -> Result<Self> {
        match value.trim().to_ascii_lowercase().as_str() {
            "all" => Ok(Self::All),
            "configs" => Ok(Self::Configs),
            "images" => Ok(Self::Images),
            "models" => Ok(Self::Models),
            "sounds" => Ok(Self::Sounds),
            "fonts" => Ok(Self::Fonts),
            "shaders" => Ok(Self::Shaders),
            "other" => Ok(Self::Other),
            _ => Err(Error::new(
                "FILE_EXPORT_TYPE_INVALID",
                format!(
                    "unknown file type {value:?}; expected all, configs, images, models, sounds, fonts, shaders, other"
                ),
            )),
        }
    }
}

/// Category of the unpacked filename (a final `.dvpl` suffix is ignored).
pub fn client_file_type(path: &Path) -> ClientFileType {
    let logical = unpacked_path(path);
    let extension = logical
        .extension()
        .and_then(|value| value.to_str())
        .unwrap_or("")
        .to_ascii_lowercase();
    match extension.as_str() {
        "yaml" | "yml" | "xml" | "xsd" | "txt" | "json" | "ini" | "cfg" | "toml" | "csv"
        | "tab" | "actions" | "style" | "material" | "html" | "htm" => ClientFileType::Configs,
        "webp" | "png" | "jpg" | "jpeg" | "dds" | "pvr" | "tex" | "tga" | "bmp" | "gif" | "svg"
        | "ktx" | "ktx2" => ClientFileType::Images,
        "sc2" | "scg" | "anim" | "animation" | "model" | "mesh" | "heightmap" | "mkm" | "lka" => {
            ClientFileType::Models
        }
        "bnk" | "pck" | "wem" | "wav" | "ogg" | "mp3" | "flac" | "fsb" => ClientFileType::Sounds,
        "ttf" | "otf" | "fnt" | "woff" | "woff2" => ClientFileType::Fonts,
        "sl" | "slh" | "glsl" | "hlsl" | "vert" | "frag" | "metal" => ClientFileType::Shaders,
        _ => ClientFileType::Other,
    }
}

#[derive(Debug, Clone)]
pub struct ClientFilesExportOptions {
    pub game_dir: Option<PathBuf>,
    /// Override the standard packs folder. A missing explicit folder is an error.
    pub cache_dir: Option<PathBuf>,
    /// Disable the cache overlay entirely.
    pub no_cache: bool,
    /// Count selected stored files without reading, decoding, or writing them.
    pub dry_run: bool,
    pub output_dir: PathBuf,
    /// Empty or containing All selects every file.
    pub types: Vec<ClientFileType>,
    /// Glob patterns matched against unpacked paths relative to Data/packs.
    pub exclude: Vec<String>,
}

impl ClientFilesExportOptions {
    pub fn new(output_dir: impl Into<PathBuf>) -> Self {
        Self {
            game_dir: None,
            cache_dir: None,
            no_cache: false,
            dry_run: false,
            output_dir: output_dir.into(),
            types: vec![ClientFileType::All],
            exclude: Vec::new(),
        }
    }
}

#[derive(Debug, Serialize)]
pub struct ClientFilesExportReport {
    pub dry_run: bool,
    pub output_dir: PathBuf,
    pub resource_dir: PathBuf,
    pub cache_dir: Option<PathBuf>,
    pub types: Vec<ClientFileType>,
    pub exclude: Vec<String>,
    pub excluded_file_count: usize,
    pub file_count: usize,
    pub written_count: usize,
    pub unchanged_count: usize,
    pub cache_file_count: usize,
    pub dvpl_count: usize,
    pub skipped_symlink_count: usize,
    /// Size of selected source files on disk, including DVPL containers.
    pub source_bytes: u64,
    /// Decoded size; zero in dry-run mode.
    pub output_bytes: u64,
    pub elapsed_seconds: f64,
}

struct SourceFile {
    path: PathBuf,
    packed: bool,
    from_cache: bool,
}

/// Merge Data and packs by unpacked relative path; cache wins regardless of mtime.
/// Plain files take precedence over their .dvpl sibling within each source tree.
/// Only one selected source is read per output file. Existing identical output is
/// kept; unrelated destination files are retained. Failure stops the export.
pub fn export_client_files(options: &ClientFilesExportOptions) -> Result<ClientFilesExportReport> {
    export_client_files_with_progress(options, |_| Ok(()))
}

/// Same export with synchronous progress after discovery and each processed file.
/// A callback error stops the export; files already written are retained.
pub fn export_client_files_with_progress(
    options: &ClientFilesExportOptions,
    mut progress: impl FnMut(&ClientFilesExportReport) -> Result<()>,
) -> Result<ClientFilesExportReport> {
    let started = Instant::now();
    if options.no_cache && options.cache_dir.is_some() {
        return Err(Error::new(
            "FILE_EXPORT_OPTIONS_INVALID",
            "cache_dir and no_cache cannot be combined",
        ));
    }
    let exclusions = options
        .exclude
        .iter()
        .map(|value| {
            if value.is_empty() {
                return Err(Error::new(
                    "FILE_EXPORT_PATTERN_INVALID",
                    "exclude pattern must not be empty",
                ));
            }
            Pattern::new(value).map_err(|error| {
                Error::new(
                    "FILE_EXPORT_PATTERN_INVALID",
                    format!("invalid exclude pattern {value:?}: {error}"),
                )
            })
        })
        .collect::<Result<Vec<_>>>()?;
    let game = resolve_game_dir(options.game_dir.clone())?;
    let resource_dir = canonical_directory(&game.resource_dir)?;
    let cache_dir = if options.no_cache {
        None
    } else if let Some(path) = &options.cache_dir {
        Some(canonical_directory(path)?)
    } else {
        game.cache_dir
            .filter(|path| path.is_dir())
            .map(|path| canonical_directory(&path))
            .transpose()?
    };
    let output_dir = resolve_output_path(&options.output_dir)?;
    for source in std::iter::once(&resource_dir).chain(cache_dir.iter()) {
        if output_dir.starts_with(source) || source.starts_with(&output_dir) {
            return Err(Error::at(
                "FILE_EXPORT_OUTPUT_INVALID",
                "output and source directories must not overlap",
                &output_dir,
            ));
        }
    }
    let mut files = BTreeMap::new();
    let mut skipped_symlink_count = 0;
    collect_files(
        &resource_dir,
        options,
        false,
        &mut files,
        &mut skipped_symlink_count,
    )?;
    if let Some(cache) = &cache_dir {
        collect_files(cache, options, true, &mut files, &mut skipped_symlink_count)?;
    }
    let before_exclusions = files.len();
    let match_options = MatchOptions {
        case_sensitive: true,
        require_literal_separator: true,
        require_literal_leading_dot: false,
    };
    files.retain(|relative, _| {
        !exclusions
            .iter()
            .any(|pattern| pattern.matches_path_with(relative, match_options))
    });
    let mut report = ClientFilesExportReport {
        dry_run: options.dry_run,
        output_dir,
        resource_dir,
        cache_dir,
        types: if options.types.is_empty() {
            vec![ClientFileType::All]
        } else {
            options.types.clone()
        },
        exclude: options.exclude.clone(),
        excluded_file_count: before_exclusions - files.len(),
        file_count: files.len(),
        written_count: 0,
        unchanged_count: 0,
        cache_file_count: 0,
        dvpl_count: 0,
        skipped_symlink_count,
        source_bytes: 0,
        output_bytes: 0,
        elapsed_seconds: 0.0,
    };
    if options.dry_run {
        for source in files.values() {
            let metadata = fs::metadata(&source.path)
                .map_err(|error| io_error("could not inspect client file", &source.path, error))?;
            report.source_bytes += metadata.len();
            report.dvpl_count += usize::from(source.packed);
            report.cache_file_count += usize::from(source.from_cache);
        }
        report.elapsed_seconds = started.elapsed().as_secs_f64();
        progress(&report)?;
        return Ok(report);
    }
    fs::create_dir_all(&report.output_dir).map_err(|error| {
        io_error(
            "could not create export directory",
            &report.output_dir,
            error,
        )
    })?;
    progress(&report)?;
    for (relative, source) in files {
        let destination = report.output_dir.join(relative);
        reject_output_symlinks(&destination, &report.output_dir)?;
        let bytes = fs::read(&source.path)
            .map_err(|error| io_error("could not read client file", &source.path, error))?;
        report.source_bytes += bytes.len() as u64;
        let bytes = if source.packed {
            let (decoded, kind) = decode_dvpl(&bytes).map_err(|mut error| {
                error.path = Some(source.path.clone());
                error
            })?;
            if kind == ContainerKind::Plain {
                return Err(Error::at(
                    "DVPL_INVALID",
                    "file with .dvpl extension has no DVPL footer",
                    &source.path,
                ));
            }
            report.dvpl_count += 1;
            decoded
        } else {
            bytes
        };
        report.output_bytes += bytes.len() as u64;
        report.cache_file_count += usize::from(source.from_cache);
        let existing = match fs::read(&destination) {
            Ok(existing) => Some(existing),
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => None,
            Err(error) => {
                return Err(io_error(
                    "could not read existing export file",
                    &destination,
                    error,
                ));
            }
        };
        if existing.as_deref() == Some(bytes.as_slice()) {
            report.unchanged_count += 1;
        } else {
            fs::create_dir_all(destination.parent().unwrap()).map_err(|error| {
                io_error("could not create resource directory", &destination, error)
            })?;
            super::output::atomic_write(&destination, &bytes)?;
            report.written_count += 1;
        }
        report.elapsed_seconds = started.elapsed().as_secs_f64();
        progress(&report)?;
    }
    report.elapsed_seconds = started.elapsed().as_secs_f64();
    Ok(report)
}

fn is_packed(path: &Path) -> bool {
    path.extension()
        .is_some_and(|value| value.eq_ignore_ascii_case("dvpl"))
}

fn unpacked_path(path: &Path) -> PathBuf {
    if is_packed(path) {
        path.with_extension("")
    } else {
        path.to_path_buf()
    }
}

fn collect_files(
    root: &Path,
    options: &ClientFilesExportOptions,
    from_cache: bool,
    files: &mut BTreeMap<PathBuf, SourceFile>,
    skipped: &mut usize,
) -> Result<()> {
    let mut local = BTreeMap::<PathBuf, SourceFile>::new();
    let mut directories = vec![root.to_path_buf()];
    while let Some(directory) = directories.pop() {
        for entry in fs::read_dir(&directory)
            .map_err(|error| io_error("could not scan client directory", &directory, error))?
        {
            let entry = entry.map_err(|error| {
                io_error("could not read client directory entry", &directory, error)
            })?;
            let kind = entry.file_type().map_err(|error| {
                io_error("could not inspect client entry", &entry.path(), error)
            })?;
            if kind.is_symlink() {
                *skipped += 1;
                continue;
            }
            if kind.is_dir() {
                directories.push(entry.path());
                continue;
            }
            if !kind.is_file() {
                continue;
            }
            let path = entry.path();
            if !options.types.is_empty()
                && !options.types.contains(&ClientFileType::All)
                && !options.types.contains(&client_file_type(&path))
            {
                continue;
            }
            let packed = is_packed(&path);
            let relative = unpacked_path(path.strip_prefix(root).unwrap());
            if local
                .get(&relative)
                .is_none_or(|previous| previous.packed && !packed)
            {
                local.insert(
                    relative,
                    SourceFile {
                        path,
                        packed,
                        from_cache,
                    },
                );
            }
        }
    }
    files.extend(local);
    Ok(())
}

fn canonical_directory(path: &Path) -> Result<PathBuf> {
    if !path.is_dir() {
        return Err(Error::at(
            "FILE_EXPORT_SOURCE_NOT_FOUND",
            "source directory does not exist",
            path,
        ));
    }
    path.canonicalize()
        .map_err(|error| io_error("could not resolve source directory", path, error))
}

// Resolve the existing ancestor before creating anything, including symlink aliases.
fn resolve_output_path(path: &Path) -> Result<PathBuf> {
    let absolute = std::path::absolute(path)
        .map_err(|error| io_error("could not resolve output path", path, error))?;
    let mut ancestor = absolute.as_path();
    let mut missing = Vec::new();
    while !ancestor.exists() {
        missing.push(
            ancestor
                .file_name()
                .ok_or_else(|| {
                    Error::at(
                        "FILE_EXPORT_OUTPUT_INVALID",
                        "cannot resolve output directory",
                        path,
                    )
                })?
                .to_owned(),
        );
        ancestor = ancestor.parent().unwrap();
    }
    if !ancestor.is_dir() {
        return Err(Error::at(
            "FILE_EXPORT_OUTPUT_INVALID",
            "output path has a file in place of a directory",
            ancestor,
        ));
    }
    let mut resolved = ancestor
        .canonicalize()
        .map_err(|error| io_error("could not resolve output directory", ancestor, error))?;
    for name in missing.into_iter().rev() {
        resolved.push(name);
    }
    Ok(resolved)
}

fn reject_output_symlinks(path: &Path, root: &Path) -> Result<()> {
    for ancestor in path.ancestors().take_while(|path| *path != root) {
        match fs::symlink_metadata(ancestor) {
            Ok(metadata) if metadata.file_type().is_symlink() => {
                return Err(Error::at(
                    "FILE_EXPORT_OUTPUT_INVALID",
                    "export path contains a symlink",
                    ancestor,
                ));
            }
            Ok(_) => {}
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => {}
            Err(error) => return Err(io_error("could not inspect export path", ancestor, error)),
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests;

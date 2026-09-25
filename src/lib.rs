mod dvpl;
mod game_dir;
mod git_diff;
mod installer;
mod manifest;
mod patch;

pub use game_dir::{ResolvedGameDir, discover_game_dir, resolve_game_dir};
pub use installer::{
    FileValidation, InstallReport, RemovalReport, ValidationReport, install_mod, remove_mod,
    validate_mod,
};
pub use manifest::{Action, Manifest, ModFile, load_manifest, update_from_git_diff};

use serde::Serialize;
use std::fmt;
use std::path::PathBuf;

/// Stable, serializable error shape for CLI, Tauri IPC, and other Rust callers.
#[derive(Debug, Clone, Serialize)]
pub struct InstallerError {
    pub code: String,
    pub message: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub path: Option<PathBuf>,
}

impl InstallerError {
    pub fn new(code: impl Into<String>, message: impl Into<String>) -> Self {
        Self {
            code: code.into(),
            message: message.into(),
            path: None,
        }
    }

    pub fn at(
        code: impl Into<String>,
        message: impl Into<String>,
        path: impl Into<PathBuf>,
    ) -> Self {
        Self {
            code: code.into(),
            message: message.into(),
            path: Some(path.into()),
        }
    }
}

impl fmt::Display for InstallerError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        if let Some(path) = &self.path {
            write!(f, "{}: {} ({})", self.code, self.message, path.display())
        } else {
            write!(f, "{}: {}", self.code, self.message)
        }
    }
}

impl std::error::Error for InstallerError {}

pub type Result<T> = std::result::Result<T, InstallerError>;

pub(crate) fn io_error(
    context: &str,
    path: &std::path::Path,
    error: std::io::Error,
) -> InstallerError {
    InstallerError::at(
        "IO_ERROR",
        format!("{context}: {error}"),
        path.to_path_buf(),
    )
}

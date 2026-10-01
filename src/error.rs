use serde::Serialize;
use std::fmt;
use std::path::PathBuf;

/// Stable, serializable error shape for CLI, Tauri IPC, and other Rust callers.
#[derive(Debug, Clone, Serialize)]
pub struct Error {
    pub code: String,
    pub message: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub path: Option<PathBuf>,
}

impl Error {
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

impl fmt::Display for Error {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        if let Some(path) = &self.path {
            write!(f, "{}: {} ({})", self.code, self.message, path.display())
        } else {
            write!(f, "{}: {}", self.code, self.message)
        }
    }
}

impl std::error::Error for Error {}

pub type Result<T> = std::result::Result<T, Error>;

#[cfg(any(
    feature = "client",
    feature = "export",
    feature = "mods",
    feature = "replay"
))]
pub(crate) fn io_error(context: &str, path: &std::path::Path, error: std::io::Error) -> Error {
    Error::at(
        "IO_ERROR",
        format!("{context}: {error}"),
        path.to_path_buf(),
    )
}

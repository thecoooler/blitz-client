//! Mod manifests, validation, installation, removal, and Git patch generation.
mod git_diff;
mod installer;
mod manifest;
mod patch;
pub use installer::{
    FileValidation, InstallReport, RemovalReport, ValidationReport, install_mod, remove_mod,
    validate_mod,
};
pub use manifest::{Action, Manifest, ModFile, load_manifest, update_from_git_diff};

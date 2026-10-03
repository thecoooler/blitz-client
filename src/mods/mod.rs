//! Mod manifests, validation, installation, removal, and Git patch generation.
mod discovery;
mod git_diff;
mod installer;
mod manifest;
mod patch;
mod patch_sources;
mod text_match;
pub use discovery::discover_mods;
pub use installer::{
    FileValidation, InstallReport, InstalledModInfo, InstalledModsReport, RemovalReport,
    ValidationReport, install_mod, list_installed_mods, remove_mod, validate_mod,
};
pub use manifest::{Action, Manifest, ModFile, load_manifest, update_from_git_diff};

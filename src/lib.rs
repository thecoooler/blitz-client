//! Client resources, catalogs, exports, mods, and replay analysis.
//! Enable only the modules you need with Cargo features; see the README.
mod error;
#[cfg(any(feature = "client", feature = "replay"))]
mod paths;
pub use error::{Error, Result};
#[cfg(any(feature = "client", feature = "replay"))]
pub use paths::{default_cache_dir, default_replay_dir};
/// Compatibility name for the shared error type.
pub type InstallerError = Error;
#[cfg(any(
    feature = "client",
    feature = "export",
    feature = "mods",
    feature = "replay"
))]
pub(crate) use error::io_error;

#[cfg(feature = "client")]
pub mod client;
#[cfg(feature = "export")]
pub mod export;
#[cfg(feature = "mods")]
pub mod mods;
#[cfg(feature = "replay")]
pub mod replay;
#[cfg(feature = "resources")]
pub mod resources;

// Compatibility with the original imports from the crate root.
#[cfg(feature = "client")]
pub use client::{
    ClientCatalog, ClientMap, ClientPrice, ClientTank, ClientTankMetadata, ResolvedGameDir,
    TankModelFiles, discover_client_catalog, discover_client_tanks, discover_game_dir,
    is_known_nation, list_client_tanks, resolve_game_dir,
};
#[cfg(feature = "export")]
pub use export::{
    ClientExportOptions, ClientExportReport, ClientFileType, ClientFilesExportOptions,
    ClientFilesExportReport, TankExportFailure, TankExportOptions, TankExportReport,
    client_file_type, export_client, export_client_files, export_client_files_with_progress,
    export_tank_catalog,
};
#[cfg(feature = "mods")]
pub use mods::{
    Action, FileValidation, InstallReport, Manifest, ModFile, RemovalReport, ValidationReport,
    install_mod, load_manifest, remove_mod, update_from_git_diff, validate_mod,
};
#[cfg(feature = "replay")]
pub use replay::{
    ReplayAnalysis, ReplayFollowEvent, ReplayFollower, ReplayOptions, ReplayStreamDecoder,
    ReplayStreamEvent, TelemetryDecoder, analyze_replay, find_latest_replay,
};

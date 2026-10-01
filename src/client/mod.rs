//! Installed-client discovery and catalogs; no export or mod installation.
mod assets;
mod catalog;
mod localization;
mod location;
mod tanks;
#[cfg(feature = "export")]
pub(crate) use assets::resolve_client_asset;
pub use catalog::{
    ClientCatalog, ClientMap, ClientPrice, ClientTankMetadata, discover as discover_client_catalog,
};
pub use location::{ResolvedGameDir, discover_game_dir, resolve_game_dir};
pub(crate) use tanks::nations;
pub use tanks::{
    ClientTank, TankModelFiles, discover_client_tanks, is_known_nation, list_client_tanks,
};

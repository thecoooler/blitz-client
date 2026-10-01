//! Read a client catalog without compiling the exporter or installer.
use blitz_client::client::{discover_client_catalog, resolve_game_dir};
use std::path::PathBuf;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let directory = std::env::args_os().nth(1).map(PathBuf::from);
    let client = resolve_game_dir(directory)?;
    let catalog = discover_client_catalog(&client.resource_dir)?;
    println!(
        "maps: {}, tanks: {}, model pairs: {}",
        catalog.maps.len(),
        catalog.tanks.len(),
        catalog.model_tanks.len()
    );
    Ok(())
}

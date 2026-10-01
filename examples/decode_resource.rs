//! Decode a plain or DVPL resource without client discovery.
use blitz_client::resources::decode_dvpl;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let path = std::env::args_os()
        .nth(1)
        .ok_or("usage: decode_resource <resource-file>")?;
    let bytes = std::fs::read(path)?;
    let (decoded, container) = decode_dvpl(&bytes)?;
    println!("container: {container:?}, decoded bytes: {}", decoded.len());
    Ok(())
}

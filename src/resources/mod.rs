//! Byte-level resource codecs, independent of client discovery and export.
pub mod archive;
mod dvpl;
pub mod xml;
#[cfg(any(feature = "export", feature = "mods"))]
pub(crate) use dvpl::crc32;
pub use dvpl::{ContainerKind, decode as decode_dvpl, encode as encode_dvpl};

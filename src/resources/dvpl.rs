use crate::{Error, Result};
use crc32fast::Hasher;

const FOOTER_SIZE: usize = 20;
const MAX_PLAINTEXT_SIZE: usize = 1024 * 1024 * 1024;
const MAGIC: &[u8; 4] = b"DVPL";

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ContainerKind {
    Plain,
    Dvpl(u32),
}

pub fn decode(input: &[u8]) -> Result<(Vec<u8>, ContainerKind)> {
    if input.len() < FOOTER_SIZE || &input[input.len() - 4..] != MAGIC {
        return Ok((input.to_vec(), ContainerKind::Plain));
    }

    let footer = &input[input.len() - FOOTER_SIZE..];
    let plain_size = read_u32(&footer[0..4]) as usize;
    let compressed_size = read_u32(&footer[4..8]) as usize;
    let expected_crc = read_u32(&footer[8..12]);
    let compression = read_u32(&footer[12..16]);
    if plain_size > MAX_PLAINTEXT_SIZE {
        return Err(Error::new(
            "DVPL_SIZE_LIMIT",
            format!("DVPL payload size {plain_size} exceeds the 1 GiB safety limit"),
        ));
    }
    if input.len() - FOOTER_SIZE != compressed_size {
        return Err(Error::new(
            "DVPL_SIZE_MISMATCH",
            format!(
                "DVPL footer declares {compressed_size} payload bytes, found {}",
                input.len() - FOOTER_SIZE
            ),
        ));
    }
    let payload = &input[..input.len() - FOOTER_SIZE];
    let actual_crc = crc32(payload);
    if actual_crc != expected_crc {
        return Err(Error::new(
            "DVPL_CRC_FAILED",
            format!("DVPL payload CRC mismatch: expected {expected_crc:08x}, got {actual_crc:08x}"),
        ));
    }

    let decoded = match compression {
        0 => payload.to_vec(),
        1 | 2 => lz4_flex::block::decompress(payload, plain_size).map_err(|e| {
            Error::new(
                "DVPL_DECOMPRESS_FAILED",
                format!("LZ4 decompression failed: {e}"),
            )
        })?,
        3 => miniz_oxide::inflate::decompress_to_vec_with_limit(payload, plain_size).map_err(
            |e| {
                Error::new(
                    "DVPL_DECOMPRESS_FAILED",
                    format!("DEFLATE decompression failed: {e}"),
                )
            },
        )?,
        other => {
            return Err(Error::new(
                "DVPL_COMPRESSION_UNSUPPORTED",
                format!("unsupported DVPL compression type {other}"),
            ));
        }
    };
    if decoded.len() != plain_size {
        return Err(Error::new(
            "DVPL_SIZE_MISMATCH",
            format!(
                "DVPL declares {plain_size} plaintext bytes, decoded {}",
                decoded.len()
            ),
        ));
    }
    Ok((decoded, ContainerKind::Dvpl(compression)))
}

pub fn encode(plain: &[u8], kind: ContainerKind) -> Result<Vec<u8>> {
    let ContainerKind::Dvpl(compression) = kind else {
        return Ok(plain.to_vec());
    };
    if plain.len() > MAX_PLAINTEXT_SIZE || plain.len() > u32::MAX as usize {
        return Err(Error::new(
            "DVPL_SIZE_LIMIT",
            "patched resource is too large for a DVPL container",
        ));
    }
    let payload = match compression {
        0 => plain.to_vec(),
        1 | 2 => lz4_flex::block::compress(plain),
        3 => miniz_oxide::deflate::compress_to_vec(plain, 6),
        other => {
            return Err(Error::new(
                "DVPL_COMPRESSION_UNSUPPORTED",
                format!("unsupported DVPL compression type {other}"),
            ));
        }
    };
    if payload.len() > u32::MAX as usize {
        return Err(Error::new(
            "DVPL_SIZE_LIMIT",
            "compressed DVPL payload is too large",
        ));
    }

    let mut output = Vec::with_capacity(payload.len() + FOOTER_SIZE);
    output.extend_from_slice(&payload);
    output.extend_from_slice(&(plain.len() as u32).to_le_bytes());
    output.extend_from_slice(&(payload.len() as u32).to_le_bytes());
    output.extend_from_slice(&crc32(&payload).to_le_bytes());
    output.extend_from_slice(&compression.to_le_bytes());
    output.extend_from_slice(MAGIC);
    Ok(output)
}

fn read_u32(bytes: &[u8]) -> u32 {
    u32::from_le_bytes([bytes[0], bytes[1], bytes[2], bytes[3]])
}

pub(crate) fn crc32(bytes: &[u8]) -> u32 {
    let mut hasher = Hasher::new();
    hasher.update(bytes);
    hasher.finalize()
}

use super::{BasePlayerCreate, UpdateArena};
use crate::{Error, Result};
use prost::Message;
use serde::{Serialize, Serializer};

/// A decoded frame, retaining every original payload byte.
#[derive(Debug, Serialize)]
pub struct Packet {
    #[serde(rename = "clock")]
    pub clock_secs: f32,
    pub payload: Payload,
    #[serde(serialize_with = "serialize_hex")]
    pub raw_payload: Vec<u8>,
}

#[derive(Debug, Serialize)]
pub enum Payload {
    BasePlayerCreate {
        author_nickname: String,
        arena_unique_id: u64,
        arena_type_id: u32,
        arguments: Box<BasePlayerCreate>,
    },
    EntityMethod(EntityMethod),
    Unknown {
        packet_type: u32,
    },
}

#[derive(Debug, Serialize)]
pub enum EntityMethod {
    UpdateArena {
        field_number: u64,
        arguments: UpdateArena,
    },
    Unknown {
        sub_type: u32,
    },
}

fn invalid(message: impl Into<String>) -> Error {
    Error::new("REPLAY_PAYLOAD_INVALID", message)
}

impl Payload {
    pub fn new(packet_type: u32, raw_payload: &[u8]) -> Result<Self> {
        let mut reader = Reader::new(raw_payload);
        match packet_type {
            0 => {
                reader.take(10)?;
                let author_nickname = reader.string()?;
                let arena_unique_id = reader.u64()?;
                let arena_type_id = reader.u32()?;
                let length = reader.quirky_length()?;
                let arguments = serde_pickle::from_slice(reader.take(length)?, Default::default())
                    .map_err(|error| {
                        invalid(format!("cannot decode arena context pickle: {error}"))
                    })?;
                Ok(Self::BasePlayerCreate {
                    author_nickname,
                    arena_unique_id,
                    arena_type_id,
                    arguments,
                })
            }
            8 => Ok(Self::EntityMethod(EntityMethod::new(raw_payload)?)),
            _ => Ok(Self::Unknown { packet_type }),
        }
    }
}

impl EntityMethod {
    pub fn new(raw_payload: &[u8]) -> Result<Self> {
        let mut reader = Reader::new(raw_payload);
        reader.u32()?; // Receiver entity ID.
        let sub_type = reader.u32()?;
        if sub_type != 47 {
            return Ok(Self::Unknown { sub_type });
        }
        reader.u32()?; // Method envelope length.
        let field_number = reader.varint()?;
        let length = reader.quirky_length()?;
        let arguments = UpdateArena::decode(reader.take(length)?)
            .map_err(|error| invalid(format!("cannot decode UpdateArena protobuf: {error}")))?;
        Ok(Self::UpdateArena {
            field_number,
            arguments,
        })
    }
}

/// Reads borrowed slices so length fields cannot cause unbounded allocations.
struct Reader<'a> {
    remaining: &'a [u8],
}
impl<'a> Reader<'a> {
    fn new(bytes: &'a [u8]) -> Self {
        Self { remaining: bytes }
    }
    fn take(&mut self, length: usize) -> Result<&'a [u8]> {
        if length > self.remaining.len() {
            return Err(invalid(format!(
                "truncated payload: need {length} bytes, have {}",
                self.remaining.len()
            )));
        }
        let (bytes, remaining) = self.remaining.split_at(length);
        self.remaining = remaining;
        Ok(bytes)
    }
    fn u8(&mut self) -> Result<u8> {
        Ok(self.take(1)?[0])
    }
    fn u16(&mut self) -> Result<u16> {
        Ok(u16::from_le_bytes(self.take(2)?.try_into().unwrap()))
    }
    fn u32(&mut self) -> Result<u32> {
        Ok(u32::from_le_bytes(self.take(4)?.try_into().unwrap()))
    }
    fn u64(&mut self) -> Result<u64> {
        Ok(u64::from_le_bytes(self.take(8)?.try_into().unwrap()))
    }
    fn string(&mut self) -> Result<String> {
        let length = self.u8()? as usize;
        std::str::from_utf8(self.take(length)?)
            .map(str::to_owned)
            .map_err(|error| invalid(format!("invalid payload string: {error}")))
    }
    fn quirky_length(&mut self) -> Result<usize> {
        let first = self.u8()?;
        if first != 0xff {
            return Ok(first as usize);
        }
        let length = self.u16()?;
        if self.u8()? != 0 {
            return Err(invalid("invalid extended-length terminator"));
        }
        Ok(length as usize)
    }
    fn varint(&mut self) -> Result<u64> {
        let mut result = 0;
        for shift in (0..70).step_by(7) {
            let byte = self.u8()?;
            if shift == 63 && byte > 1 {
                return Err(invalid("protobuf varint overflow"));
            }
            result |= u64::from(byte & 0x7f) << shift;
            if byte & 0x80 == 0 {
                return Ok(result);
            }
        }
        Err(invalid("protobuf varint overflow"))
    }
}

fn serialize_hex<S: Serializer>(
    bytes: &[u8],
    serializer: S,
) -> std::result::Result<S::Ok, S::Error> {
    const DIGITS: &[u8; 16] = b"0123456789abcdef";
    let mut output = String::with_capacity(bytes.len() * 2);
    for &byte in bytes {
        output.push(DIGITS[(byte >> 4) as usize] as char);
        output.push(DIGITS[(byte & 15) as usize] as char);
    }
    serializer.serialize_str(&output)
}

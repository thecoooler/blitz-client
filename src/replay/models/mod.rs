//! Local packet models and the protobuf schemas used by replay analysis.
mod arena;
mod context;
mod packet;
mod results;
#[cfg(test)]
mod tests;
use crate::{Error, Result};
pub use arena::{Player as ArenaPlayer, Players as ArenaPlayers, UpdateArena};
pub use context::{BasePlayerCreate, TurboBattlesStatistics};
pub use packet::{EntityMethod, Packet, Payload};
use prost::Message;
pub(super) use results::BattleResults;
use serde::{Deserialize, Serialize};
use serde_bytes::ByteBuf;
use std::io::Read;

#[derive(Debug, Serialize)]
pub(super) struct Data {
    pub client_version: String,
    pub packets: Vec<Packet>,
}

/// battle_results.dat is a pickle tuple: (arena ID, protobuf bytes).
#[derive(Debug, Deserialize)]
#[serde(from = "(u64, ByteBuf)")]
pub(super) struct BattleResultsDat {
    pub buffer: Vec<u8>,
}

impl From<(u64, ByteBuf)> for BattleResultsDat {
    fn from((_, buffer): (u64, ByteBuf)) -> Self {
        Self {
            buffer: buffer.into_vec(),
        }
    }
}

impl BattleResultsDat {
    pub fn from_reader(reader: impl Read) -> Result<Self> {
        serde_pickle::from_reader(reader, Default::default())
            .map_err(|error| Error::new("REPLAY_RESULTS_INVALID", error.to_string()))
    }
}

impl TryFrom<BattleResultsDat> for BattleResults {
    type Error = Error;
    fn try_from(results: BattleResultsDat) -> Result<Self> {
        Self::decode(results.buffer.as_slice())
            .map_err(|error| Error::new("REPLAY_RESULTS_INVALID", error.to_string()))
    }
}

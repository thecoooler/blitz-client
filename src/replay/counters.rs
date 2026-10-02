//! Author-only cumulative battle counters carried by EntityMethod 14.
use super::{Packet, Payload};
use serde::Serialize;

pub const BATTLE_COUNTER_CLIENT_VERSIONS: [&str; 2] = ["26.8.0_ruby_apple", "26.10.0_ruby_apple"];

#[derive(Debug, Clone, Default, Serialize)]
pub struct BattleCounters {
    pub author_nickname: Option<String>,
    pub player_entity_id: Option<u32>,
    /// None until a counter update is received; absence does not imply zero.
    pub damage_blocked: Option<u32>,
    /// Combined assistance from spotting or immobilizing opponents.
    pub damage_assisted: Option<u32>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum BattleCounterKind {
    DamageBlocked,
    DamageAssisted,
}

#[derive(Debug, Clone, Serialize)]
pub struct BattleCounterUpdate {
    pub packet_index: usize,
    pub clock: f32,
    pub entity_id: u32,
    pub ribbon_id: u8,
    pub counter: BattleCounterKind,
    /// Absolute value reported by the server, never a sum of hit estimates.
    pub value: u32,
    pub previous_value: Option<u32>,
    /// None on the first update. May be negative for a counter correction.
    pub delta: Option<i64>,
}

#[derive(Default)]
pub struct BattleCounterDecoder {
    counters: BattleCounters,
}

impl BattleCounterDecoder {
    pub fn supports_version(version: &str) -> bool {
        BATTLE_COUNTER_CLIENT_VERSIONS.contains(&version)
    }

    pub fn counters(&self) -> &BattleCounters {
        &self.counters
    }

    pub fn reset(&mut self) {
        self.counters = BattleCounters::default();
    }

    /// Feed complete packets in stream order. The BasePlayerCreate prefix
    /// identifies the author entity, so vehicle methods cannot affect totals.
    pub fn on_packet(
        &mut self,
        packet: &Packet,
        packet_index: usize,
        version: &str,
    ) -> Option<BattleCounterUpdate> {
        if !Self::supports_version(version) {
            self.reset();
            return None;
        }
        if let Payload::BasePlayerCreate {
            author_nickname, ..
        } = &packet.payload
        {
            self.counters = BattleCounters {
                author_nickname: Some(author_nickname.clone()),
                player_entity_id: read_u32(&packet.raw_payload, 0),
                ..BattleCounters::default()
            };
            return None;
        }
        let Payload::EntityMethod(_) = &packet.payload else {
            return None;
        };
        let raw = &packet.raw_payload;
        if raw.len() != 18
            || !packet.clock_secs.is_finite()
            || read_u32(raw, 4) != Some(14)
            || read_u32(raw, 8) != Some(6)
        {
            return None;
        }
        let entity_id = read_u32(raw, 0)?;
        if self.counters.player_entity_id != Some(entity_id) {
            return None;
        }
        let (counter, total) = match raw[12] {
            5 => (
                BattleCounterKind::DamageBlocked,
                &mut self.counters.damage_blocked,
            ),
            17 => (
                BattleCounterKind::DamageAssisted,
                &mut self.counters.damage_assisted,
            ),
            _ => return None,
        };
        // Body: ribbon ID at +0, three opaque bytes, cumulative u16 at +4.
        // IDs confirmed in XML/item_defs/ribbons.xml; values checked against
        // battle_results.dat in real 26.8 and 26.10 recordings.
        let value = u32::from(u16::from_le_bytes(raw[16..18].try_into().unwrap()));
        let previous_value = total.replace(value);
        Some(BattleCounterUpdate {
            packet_index,
            clock: packet.clock_secs,
            entity_id,
            ribbon_id: raw[12],
            counter,
            value,
            previous_value,
            delta: previous_value.map(|previous| i64::from(value) - i64::from(previous)),
        })
    }
}

fn read_u32(bytes: &[u8], offset: usize) -> Option<u32> {
    Some(u32::from_le_bytes(
        bytes.get(offset..offset + 4)?.try_into().ok()?,
    ))
}

#[cfg(test)]
mod tests;

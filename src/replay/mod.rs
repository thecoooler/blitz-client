//! Complete replay analysis and incremental live packet decoding.
pub use crate::paths::default_replay_dir;
mod analysis;
mod counters;
mod follower;
mod stream;
mod telemetry;
pub use analysis::{ReplayAnalysis, ReplayOptions, analyze_replay, find_latest_replay};
pub use counters::{
    BATTLE_COUNTER_CLIENT_VERSIONS, BattleCounterDecoder, BattleCounterKind, BattleCounterUpdate,
    BattleCounters,
};
pub use follower::{ReplayFollowEvent, ReplayFollower};
pub use stream::{ReplayStreamDecoder, ReplayStreamEvent};
pub use telemetry::TelemetryDecoder;
mod models;
pub use models::{
    ArenaPlayer, ArenaPlayers, BasePlayerCreate, EntityMethod, Packet, Payload,
    TurboBattlesStatistics, UpdateArena,
};
#[cfg(test)]
mod tests;

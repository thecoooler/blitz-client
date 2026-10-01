//! Complete replay analysis and incremental live packet decoding.
mod analysis;
mod follower;
mod stream;
mod telemetry;
pub use analysis::{ReplayAnalysis, ReplayOptions, analyze_replay, find_latest_replay};
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

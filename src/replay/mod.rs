//! Complete replay analysis and incremental live packet decoding.
mod analysis;
mod follower;
mod stream;
mod telemetry;
pub use analysis::{ReplayAnalysis, ReplayOptions, analyze_replay, find_latest_replay};
pub use follower::{ReplayFollowEvent, ReplayFollower};
pub use stream::{ReplayStreamDecoder, ReplayStreamEvent};
pub use telemetry::TelemetryDecoder;
// Packet types occur in the public interface; callers need no direct parser dependency.
pub use wotbreplay_parser::models::data::{
    entity_method::EntityMethod, packet::Packet, payload::Payload,
};
#[cfg(test)]
mod tests;

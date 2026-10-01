//! Library-only live replay consumer. Ctrl+C stops the example.
use blitz_client::replay::ReplayFollower;
use std::io::Write;
use std::time::Duration;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let path = std::env::args_os()
        .nth(1)
        .ok_or("usage: replay_follow <data.replay>")?;
    let mut follower = ReplayFollower::new(path);
    let mut stdout = std::io::stdout().lock();
    loop {
        follower.poll(|event| {
            let line = serde_json::to_vec(&event).map_err(|error| {
                blitz_client::Error::new("JSON_SERIALIZE_FAILED", error.to_string())
            })?;
            stdout
                .write_all(&line)
                .and_then(|_| stdout.write_all(b"\n"))
                .and_then(|_| stdout.flush())
                .map_err(|error| blitz_client::Error::new("OUTPUT_FAILED", error.to_string()))
        })?;
        std::thread::sleep(Duration::from_millis(25));
    }
}

#![cfg(feature = "cli")]

use std::fs::File;
use std::io::{BufRead, BufReader, Write};
use std::path::PathBuf;
use std::process::{Command, Stdio};
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::Duration;

struct TempDir(PathBuf);
impl TempDir {
    fn new() -> Self {
        static NEXT: AtomicU64 = AtomicU64::new(0);
        let path = std::env::temp_dir().join(format!(
            "blitz-replay-cli-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        std::fs::create_dir(&path).unwrap();
        Self(path)
    }
}
impl Drop for TempDir {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.0);
    }
}

#[cfg(target_os = "macos")]
#[test]
fn locate_returns_replay_and_cache_paths_even_before_folder_creation() {
    let temp = TempDir::new();
    let home = temp.0.join("home");
    let game = temp.0.join("game");
    std::fs::create_dir(&game).unwrap();
    let output = Command::new(env!("CARGO_BIN_EXE_blitz-client"))
        .args(["client", "locate", "--game-dir"])
        .arg(&game)
        .arg("--json")
        .env("HOME", &home)
        .output()
        .unwrap();
    assert!(output.status.success());
    let json: serde_json::Value = serde_json::from_slice(&output.stdout).unwrap();
    let replay_dir =
        home.join("Library/Containers/com.tanksblitz/Data/Documents/tb-project/replays");
    assert!(!replay_dir.exists());
    assert_eq!(
        json["replay_dir"],
        serde_json::to_value(replay_dir).unwrap()
    );
    let cache_dir = home.join("Library/Containers/com.tanksblitz/Data/Library/Caches/packs");
    assert!(!cache_dir.exists());
    assert_eq!(json["cache_dir"], serde_json::to_value(cache_dir).unwrap());
    assert_eq!(json["source"], "explicit");
    assert_eq!(
        json["game_dir"],
        serde_json::to_value(game.canonicalize().unwrap()).unwrap()
    );
}

#[test]
fn follow_flushes_before_exit_and_waits_for_partial_packets() {
    let temp = TempDir::new();
    let path = temp.0.join("data.replay");
    let mut child = Command::new(env!("CARGO_BIN_EXE_blitz-client"))
        .args(["replay", "follow"])
        .arg(&path)
        .args(["--poll-ms", "5", "--idle-timeout-ms", "1000", "--json"])
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .unwrap();
    let stdout = child.stdout.take().unwrap();
    let (send, receive) = std::sync::mpsc::channel();
    let reader = std::thread::spawn(move || {
        for line in BufReader::new(stdout).lines() {
            send.send(line.unwrap()).unwrap();
        }
    });
    let mut writer = File::create(&path).unwrap();
    let mut bytes = 0x12345678_u32.to_le_bytes().to_vec();
    bytes.extend_from_slice(&0_u64.to_le_bytes());
    bytes.extend_from_slice(b"\x00\x01v\x00");
    writer.write_all(&bytes).unwrap();
    let header: serde_json::Value =
        serde_json::from_str(&receive.recv_timeout(Duration::from_secs(5)).unwrap()).unwrap();
    assert_eq!(header["event"], "header");
    assert!(child.try_wait().unwrap().is_none());
    let mut packet = 3_u32.to_le_bytes().to_vec();
    packet.extend_from_slice(&99_u32.to_le_bytes());
    packet.extend_from_slice(&0.5_f32.to_le_bytes());
    packet.extend_from_slice(&[1, 2, 3]);
    writer.write_all(&packet[..13]).unwrap();
    assert!(matches!(
        receive.recv_timeout(Duration::from_millis(50)),
        Err(std::sync::mpsc::RecvTimeoutError::Timeout)
    ));
    writer.write_all(&packet[13..]).unwrap();
    let event: serde_json::Value =
        serde_json::from_str(&receive.recv_timeout(Duration::from_secs(5)).unwrap()).unwrap();
    assert_eq!(event["event"], "packet");
    assert_eq!(event["seq"], 0);
    assert_eq!(event["raw_payload"], "010203");
    assert!(child.try_wait().unwrap().is_none());
    let result = child.wait_with_output().unwrap();
    reader.join().unwrap();
    assert!(
        result.status.success(),
        "{}",
        String::from_utf8_lossy(&result.stderr)
    );
    assert!(result.stderr.is_empty());
    assert!(receive.try_recv().is_err());
}

#[test]
fn follow_protects_source_and_keeps_errors_off_stdout() {
    let temp = TempDir::new();
    let path = temp.0.join("data.replay");
    let result = Command::new(env!("CARGO_BIN_EXE_blitz-client"))
        .args(["replay", "follow"])
        .arg(&path)
        .arg("--out")
        .arg(&path)
        .output()
        .unwrap();
    assert!(!result.status.success());
    assert!(!path.exists());
    std::fs::write(&path, [0; 4]).unwrap();
    let result = Command::new(env!("CARGO_BIN_EXE_blitz-client"))
        .args(["replay", "follow"])
        .arg(&path)
        .arg("--json")
        .output()
        .unwrap();
    assert!(!result.status.success());
    assert!(result.stdout.is_empty());
    assert!(String::from_utf8_lossy(&result.stderr).contains("REPLAY_DATA_INVALID"));
    let result = Command::new(env!("CARGO_BIN_EXE_blitz-client"))
        .args(["replay", "follow"])
        .arg(&path)
        .args(["--poll-ms", "0"])
        .output()
        .unwrap();
    assert!(!result.status.success());
    assert_eq!(std::fs::read(&path).unwrap(), [0; 4]);
}

#![cfg(feature = "cli")]

use blitz_client::resources::{ContainerKind, encode_dvpl};
use std::fs;
use std::path::PathBuf;
use std::process::Command;
use std::sync::atomic::{AtomicU64, Ordering};

struct Fixture(PathBuf);
impl Fixture {
    fn new() -> Self {
        static NEXT: AtomicU64 = AtomicU64::new(0);
        let root = std::env::temp_dir().join(format!(
            "blitz-files-cli-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        fs::create_dir_all(root.join("game/Data")).unwrap();
        fs::create_dir_all(root.join("packs")).unwrap();
        Self(root)
    }
}
impl Drop for Fixture {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.0);
    }
}

#[test]
fn cli_exports_selected_effective_files_and_emits_one_json_report() {
    let fixture = Fixture::new();
    let game = fixture.0.join("game");
    let cache = fixture.0.join("packs");
    let output = fixture.0.join("output");
    fs::write(game.join("Data/config.yaml"), b"base").unwrap();
    fs::write(
        cache.join("config.yaml.dvpl"),
        encode_dvpl(b"cached config", ContainerKind::Dvpl(1)).unwrap(),
    )
    .unwrap();
    fs::write(game.join("Data/tank.scg"), b"model").unwrap();
    fs::write(game.join("Data/image.webp"), b"image").unwrap();
    let run = || {
        Command::new(env!("CARGO_BIN_EXE_blitz-client"))
            .args(["client", "files", "export", "--game-dir"])
            .arg(&game)
            .arg("--cache-dir")
            .arg(&cache)
            .arg("--output-dir")
            .arg(&output)
            .args(["--types", "configs,images", "--json"])
            .output()
            .unwrap()
    };
    let result = run();
    assert!(
        result.status.success(),
        "{}",
        String::from_utf8_lossy(&result.stderr)
    );
    assert!(result.stderr.is_empty());
    let report: serde_json::Value = serde_json::from_slice(&result.stdout).unwrap();
    assert_eq!(report["file_count"], 2);
    assert_eq!(report["cache_file_count"], 1);
    assert_eq!(report["written_count"], 2);
    assert_eq!(
        fs::read(output.join("config.yaml")).unwrap(),
        b"cached config"
    );
    assert!(!output.join("config.yaml.dvpl").exists());
    assert!(!output.join("tank.scg").exists());
    let again: serde_json::Value = serde_json::from_slice(&run().stdout).unwrap();
    assert_eq!(again["unchanged_count"], 2);
    assert_eq!(again["written_count"], 0);
}

#[test]
fn invalid_cli_options_fail_before_creating_output() {
    let fixture = Fixture::new();
    let output = fixture.0.join("output");
    for args in [
        vec!["--types", "configs,unknown"],
        vec!["--types", "configs,"],
        vec!["--types"],
        vec!["--cache-dir", ".", "--no-cache"],
        vec!["--unknown"],
    ] {
        let result = Command::new(env!("CARGO_BIN_EXE_blitz-client"))
            .args(["client", "files", "export", "--output-dir"])
            .arg(&output)
            .args(args)
            .arg("--json")
            .output()
            .unwrap();
        assert!(!result.status.success());
        let report: serde_json::Value = serde_json::from_slice(&result.stdout).unwrap();
        assert_eq!(report["error"]["code"], "USAGE");
        assert!(!output.exists());
    }
}

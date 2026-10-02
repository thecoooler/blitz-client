#![cfg(feature = "cli")]

use std::fs;
use std::path::{Path, PathBuf};
use std::process::{Command, Output};
use std::sync::atomic::{AtomicU64, Ordering};

struct Fixture(PathBuf);
impl Fixture {
    fn new() -> Self {
        static NEXT: AtomicU64 = AtomicU64::new(0);
        let root = std::env::temp_dir().join(format!(
            "blitz-mods-cli-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        fs::create_dir_all(root.join("game/Data")).unwrap();
        fs::create_dir_all(root.join("mods")).unwrap();
        fs::write(root.join("game/Data/config.yaml"), "original\n").unwrap();
        Self(root)
    }

    fn manifest(&self, dir: &str, id: &str, find: &str) {
        let dir = self.0.join("mods").join(dir);
        fs::create_dir_all(&dir).unwrap();
        fs::write(
            dir.join("mod.toml"),
            format!(
                "id = {id:?}\nname = {id:?}\n[[files]]\nname = \"config.yaml\"\naction = \"replace\"\nfind = {find:?}\nreplace = \"changed\"\n"
            ),
        )
        .unwrap();
    }

    fn run(&self, path: &Path, json: bool) -> Output {
        let mut command = Command::new(env!("CARGO_BIN_EXE_blitz-client"));
        command
            .args(["mod", "validate"])
            .arg(path)
            .arg("--game-dir")
            .arg(self.0.join("game"));
        if json {
            command.arg("--json");
        }
        command.output().unwrap()
    }
}
impl Drop for Fixture {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.0);
    }
}

#[test]
fn recursively_validates_root_and_nested_mods_without_writing_resources() {
    let fixture = Fixture::new();
    fixture.manifest("", "Root", "original");
    fixture.manifest("group/deep", "Nested", "original");
    let result = fixture.run(&fixture.0.join("mods"), true);
    assert!(
        result.status.success(),
        "{}",
        String::from_utf8_lossy(&result.stderr)
    );
    let value: serde_json::Value = serde_json::from_slice(&result.stdout).unwrap();
    assert_eq!(value["valid"], true);
    assert_eq!(value["mods"].as_array().unwrap().len(), 2);
    assert!(
        value["mods"]
            .as_array()
            .unwrap()
            .iter()
            .all(|item| item["report"]["valid"] == true)
    );
    assert_eq!(
        fs::read_to_string(fixture.0.join("game/Data/config.yaml")).unwrap(),
        "original\n"
    );
    assert!(!fixture.0.join("game/.tanks-blitz-client").exists());
    let text = fixture.run(&fixture.0.join("mods"), false);
    assert!(
        String::from_utf8_lossy(&text.stdout).contains("Validated 2 mod(s): 2 passed, 0 failed")
    );
}

#[test]
fn collection_continues_after_bad_manifest_and_failed_resource() {
    let fixture = Fixture::new();
    fixture.manifest("a-invalid", "Invalid", "original");
    fs::write(fixture.0.join("mods/a-invalid/mod.toml"), "invalid toml").unwrap();
    fixture.manifest("b-failed", "Failed", "missing text");
    fixture.manifest("z-valid", "Valid", "original");
    let result = fixture.run(&fixture.0.join("mods"), true);
    assert_eq!(result.status.code(), Some(4));
    let value: serde_json::Value = serde_json::from_slice(&result.stdout).unwrap();
    assert_eq!(value["valid"], false);
    assert_eq!(value["mods"][0]["error"]["code"], "MANIFEST_INVALID");
    assert_eq!(
        value["mods"][1]["report"]["files"][0]["error_code"],
        "REPLACE_MATCH_COUNT"
    );
    assert_eq!(value["mods"][2]["report"]["valid"], true);
}

#[test]
fn single_mod_and_explicit_manifest_keep_existing_report() {
    let fixture = Fixture::new();
    fixture.manifest("single", "Single", "original");
    for path in [
        fixture.0.join("mods/single"),
        fixture.0.join("mods/single/mod.toml"),
    ] {
        let result = fixture.run(&path, true);
        assert!(result.status.success());
        let value: serde_json::Value = serde_json::from_slice(&result.stdout).unwrap();
        assert_eq!(value["mod_id"], "Single");
        assert_eq!(value["valid"], true);
        assert!(value.get("mods").is_none());
    }
    let result = fixture.run(&fixture.0.join("mods"), true);
    let value: serde_json::Value = serde_json::from_slice(&result.stdout).unwrap();
    assert_eq!(value["mods"].as_array().unwrap().len(), 1);
}

#[test]
fn empty_directory_is_an_error() {
    let fixture = Fixture::new();
    let result = fixture.run(&fixture.0.join("mods"), true);
    assert_eq!(result.status.code(), Some(2));
    let value: serde_json::Value = serde_json::from_slice(&result.stdout).unwrap();
    assert_eq!(value["error"]["code"], "MANIFEST_NOT_FOUND");
}

#[cfg(unix)]
#[test]
fn directory_symlink_cycles_are_not_followed() {
    let fixture = Fixture::new();
    fixture.manifest("single", "Single", "original");
    std::os::unix::fs::symlink(fixture.0.join("mods"), fixture.0.join("mods/loop")).unwrap();
    let result = fixture.run(&fixture.0.join("mods"), true);
    assert!(result.status.success());
    let value: serde_json::Value = serde_json::from_slice(&result.stdout).unwrap();
    assert_eq!(value["mods"].as_array().unwrap().len(), 1);
}

#[test]
fn insert_actions_round_trip_and_compose_with_other_mods() {
    use blitz_client::mods::{install_mod, remove_mod, validate_mod};
    let fixture = Fixture::new();
    let game = Some(fixture.0.join("game"));
    for (id, action, replacement) in [("A", "before", "prefix:"), ("B", "after", ":suffix")] {
        fixture.manifest(id, id, "original");
        let manifest = fixture.0.join("mods").join(id).join("mod.toml");
        let text = fs::read_to_string(&manifest)
            .unwrap()
            .replace("action = \"replace\"", &format!("action = {action:?}"))
            .replace(
                "replace = \"changed\"",
                &format!("replace = {replacement:?}"),
            );
        fs::write(&manifest, text).unwrap();
        assert!(validate_mod(&manifest, game.clone()).unwrap().valid);
        install_mod(&manifest, game.clone()).unwrap();
    }
    let target = fixture.0.join("game/Data/config.yaml");
    assert_eq!(
        fs::read_to_string(&target).unwrap(),
        "prefix:original:suffix\n"
    );
    remove_mod("A", game.clone()).unwrap();
    assert_eq!(fs::read_to_string(&target).unwrap(), "original:suffix\n");
    remove_mod("B", game).unwrap();
    assert_eq!(fs::read_to_string(&target).unwrap(), "original\n");
}

#[test]
fn insert_actions_require_one_exact_match_and_required_fields() {
    use blitz_client::mods::validate_mod;
    let fixture = Fixture::new();
    let game = Some(fixture.0.join("game"));
    for action in ["before", "after"] {
        for find in ["missing", "i"] {
            fixture.manifest(action, action, find);
            let manifest = fixture.0.join("mods").join(action).join("mod.toml");
            let text = fs::read_to_string(&manifest)
                .unwrap()
                .replace("action = \"replace\"", &format!("action = {action:?}"));
            fs::write(&manifest, &text).unwrap();
            let report = validate_mod(&manifest, game.clone()).unwrap();
            assert!(!report.valid);
            assert_eq!(
                report.files[0].error_code.as_deref(),
                Some("REPLACE_MATCH_COUNT")
            );
            fs::write(&manifest, text.replace("replace = \"changed\"", "")).unwrap();
            assert_eq!(
                validate_mod(&manifest, game.clone()).unwrap_err().code,
                "MANIFEST_INVALID_ACTION"
            );
        }
    }
}

#[test]
fn clear_handles_plain_and_dvpl_binary_resources_and_restores_original_bytes() {
    use blitz_client::mods::{install_mod, remove_mod, validate_mod};
    use blitz_client::resources::{ContainerKind, decode_dvpl, encode_dvpl};
    let fixture = Fixture::new();
    let game = Some(fixture.0.join("game"));
    let binary = [0xff, 0x00, 0xfe, 0x80];
    for (index, kind) in [
        ContainerKind::Plain,
        ContainerKind::Dvpl(0),
        ContainerKind::Dvpl(1),
        ContainerKind::Dvpl(2),
    ]
    .into_iter()
    .enumerate()
    {
        let name = format!("image{index}.webp");
        let target = fixture
            .0
            .join("game/Data")
            .join(if kind == ContainerKind::Plain {
                name.clone()
            } else {
                format!("{name}.dvpl")
            });
        let original = encode_dvpl(&binary, kind).unwrap();
        fs::write(&target, &original).unwrap();
        let dir = fixture.0.join("mods").join(format!("Clear{index}"));
        fs::create_dir_all(&dir).unwrap();
        fs::write(dir.join("mod.toml"), format!("id = \"Clear{index}\"\nname = \"Clear\"\n[[files]]\nname = {name:?}\naction = \"clear\"\n")).unwrap();
        assert!(validate_mod(&dir, game.clone()).unwrap().valid);
        assert_eq!(fs::read(&target).unwrap(), original);
        install_mod(&dir, game.clone()).unwrap();
        let (payload, output_kind) = decode_dvpl(&fs::read(&target).unwrap()).unwrap();
        assert!(payload.is_empty());
        assert_eq!(output_kind, kind);
        remove_mod(&format!("Clear{index}"), game.clone()).unwrap();
        assert_eq!(fs::read(&target).unwrap(), original);
    }
}

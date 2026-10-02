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
            dir.join("mod.yaml"),
            format!(
                "id: {id:?}\nname: {id:?}\nfiles:\n  config.yaml:\n    - replace:\n        search: {find:?}\n        replace: \"changed\"\n"
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
    fs::write(fixture.0.join("mods/a-invalid/mod.yaml"), "files: [invalid").unwrap();
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
        fixture.0.join("mods/single/mod.yaml"),
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
        let manifest = fixture.0.join("mods").join(id).join("mod.yaml");
        let text = fs::read_to_string(&manifest)
            .unwrap()
            .replace("- replace:", &format!("- {action}:"))
            .replace("replace: \"changed\"", &format!("replace: {replacement:?}"));
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
            let manifest = fixture.0.join("mods").join(action).join("mod.yaml");
            let text = fs::read_to_string(&manifest)
                .unwrap()
                .replace("- replace:", &format!("- {action}:"));
            fs::write(&manifest, &text).unwrap();
            let report = validate_mod(&manifest, game.clone()).unwrap();
            assert!(!report.valid);
            assert_eq!(
                report.files[0].error_code.as_deref(),
                Some("REPLACE_MATCH_COUNT")
            );
            fs::write(&manifest, text.replace("replace: \"changed\"", "")).unwrap();
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
        fs::write(
            dir.join("mod.yaml"),
            format!("id: Clear{index}\nname: Clear\nfiles:\n  {name:?}:\n    - clear: {{}}\n"),
        )
        .unwrap();
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

#[test]
fn yaml_normalized_replace_preserves_other_bytes_and_replacement() {
    use blitz_client::mods::{install_mod, remove_mod, validate_mod};
    let fixture = Fixture::new();
    let target = fixture.0.join("game/Data/config.yaml");
    let original = "untouched:  \r\n    first: \t\r\n        second:  \r\nlast: \t\r\n";
    fs::write(&target, original).unwrap();
    let manifest = fixture.0.join("mods/mod.yaml");
    fs::write(
        &manifest,
        r#"version: 1
id: Normalized
name: Normalized
files:
  config.yaml:
    - replace:
        ignore-trailing-whitespace: true
        search: |2
              first:
                  second:
        replace: "    replaced:  \n"
"#,
    )
    .unwrap();
    let game = Some(fixture.0.join("game"));
    assert!(validate_mod(&manifest, game.clone()).unwrap().valid);
    let cli = fixture.run(&fixture.0.join("mods"), true);
    assert!(
        cli.status.success(),
        "{}",
        String::from_utf8_lossy(&cli.stdout)
    );
    install_mod(&manifest, game.clone()).unwrap();
    assert_eq!(
        fs::read_to_string(&target).unwrap(),
        "untouched:  \r\n    replaced:  \nlast: \t\r\n"
    );
    remove_mod("Normalized", game).unwrap();
    assert_eq!(fs::read_to_string(&target).unwrap(), original);
}

#[test]
fn normalization_is_opt_in_and_keeps_indentation_and_case() {
    use blitz_client::mods::validate_mod;
    let fixture = Fixture::new();
    let target = fixture.0.join("game/Data/config.yaml");
    let dir = fixture.0.join("mods/single");
    fixture.manifest("single", "Single", "original");
    let manifest = dir.join("mod.yaml");
    let text = "id: Single\nname: Single\nfiles:\n  config.yaml:\n    - replace:\n        search: \"    first:\\n        second:\\n\"\n        replace: ''\n";

    let source = "    first: \r\n        second:\t\r\n";
    fs::write(&target, source).unwrap();
    fs::write(&manifest, &text).unwrap();
    let game = Some(fixture.0.join("game"));
    assert!(!validate_mod(&dir, game.clone()).unwrap().valid);
    fs::write(
        &manifest,
        format!("{text}        ignore-trailing-whitespace: true\n"),
    )
    .unwrap();
    assert!(validate_mod(&dir, game.clone()).unwrap().valid);
    for mismatch in [
        "    first: \r\n    second:\t\r\n",
        "    First: \r\n        second:\t\r\n",
        "    first: \r\n\r\n        second:\t\r\n",
        "    first: \r\n        second:\t\r\n    first:\n        second:\n",
    ] {
        fs::write(&target, mismatch).unwrap();
        assert!(!validate_mod(&dir, game.clone()).unwrap().valid);
    }
}

#[test]
fn yaml_ordered_insertions_preserve_actual_matched_whitespace_and_survive_recomposition() {
    use blitz_client::mods::{install_mod, remove_mod, validate_mod};
    let fixture = Fixture::new();
    let original = "original \t\r\n";
    let target = fixture.0.join("game/Data/config.yaml");
    fs::write(&target, original).unwrap();
    let dir = fixture.0.join("mods/group/deep");
    fs::create_dir_all(&dir).unwrap();
    fs::write(
        dir.join("mod.yml"),
        r#"id: A
name: Insertions
files:
  config.yaml:
    - before:
        ignore-trailing-whitespace: true
        search: "original\n"
        insert: "prefix\n"
    - after:
        ignore-trailing-whitespace: true
        search: "original\n"
        insert: "suffix\n"
"#,
    )
    .unwrap();
    let game = Some(fixture.0.join("game"));
    assert!(validate_mod(&dir, game.clone()).unwrap().valid);
    assert!(fixture.run(&fixture.0.join("mods"), true).status.success());
    assert_eq!(install_mod(&dir, game.clone()).unwrap().targets.len(), 1);
    assert_eq!(
        fs::read_to_string(&target).unwrap(),
        "prefix\noriginal \t\r\nsuffix\n"
    );
    fixture.manifest("B", "B", "suffix");
    install_mod(fixture.0.join("mods/B"), game.clone()).unwrap();
    assert_eq!(
        fs::read_to_string(&target).unwrap(),
        "prefix\noriginal \t\r\nchanged\n"
    );
    remove_mod("B", game.clone()).unwrap();
    assert_eq!(
        fs::read_to_string(&target).unwrap(),
        "prefix\noriginal \t\r\nsuffix\n"
    );
    assert_eq!(remove_mod("A", game).unwrap().restored_files.len(), 1);
    assert_eq!(fs::read_to_string(&target).unwrap(), original);
}

#[test]
fn yaml_rejects_ambiguous_names_invalid_flags_and_unknown_fields() {
    use blitz_client::mods::load_manifest;
    let fixture = Fixture::new();
    let dir = fixture.0.join("mods");
    let manifest = dir.join("mod.yaml");
    let base = "id: Test\nname: Test\nfiles:\n  config.yaml:\n    - replace:\n        search: original\n        replace: changed\n";
    for invalid in [format!("{base}        ignore-trailing-whitespace: yesplease\n"), format!("{base}        ignore-trailing-whitespac: true\n"), base.replace("search: original", "search: '   '" ) + "        ignore-trailing-whitespace: true\n", "id: Test\nname: Test\nfiles:\n  config.yaml:\n    - clear:\n        ignore-trailing-whitespace: true\n".to_owned()] {
        fs::write(&manifest, invalid).unwrap();
        assert!(load_manifest(&dir).is_err());
    }
    fs::write(&manifest, base).unwrap();
    fs::write(dir.join("mod.yml"), base).unwrap();
    assert_eq!(load_manifest(&dir).unwrap_err().code, "MANIFEST_AMBIGUOUS");
    let result = fixture.run(&dir, true);
    assert_eq!(result.status.code(), Some(2));
    let value: serde_json::Value = serde_json::from_slice(&result.stdout).unwrap();
    assert_eq!(value["error"]["code"], "MANIFEST_AMBIGUOUS");
}

#[test]
fn older_installed_state_without_whitespace_flag_can_be_removed() {
    use blitz_client::mods::{install_mod, remove_mod};
    let fixture = Fixture::new();
    fixture.manifest("Legacy", "Legacy", "original");
    let game = Some(fixture.0.join("game"));
    install_mod(fixture.0.join("mods/Legacy"), game.clone()).unwrap();
    let state_path = fixture.0.join("game/.tanks-blitz-client/state.json");
    let mut state: serde_json::Value =
        serde_json::from_slice(&fs::read(&state_path).unwrap()).unwrap();
    for file in state["mods"][0]["files"].as_array_mut().unwrap() {
        file["operation"]
            .as_object_mut()
            .unwrap()
            .remove("ignore_trailing_whitespace");
    }
    fs::write(&state_path, serde_json::to_vec(&state).unwrap()).unwrap();
    remove_mod("Legacy", game).unwrap();
    assert_eq!(
        fs::read_to_string(fixture.0.join("game/Data/config.yaml")).unwrap(),
        "original\n"
    );
}

#[test]
fn yaml_git_update_preserves_manual_operations_and_refreshes_generated_patches() {
    use blitz_client::mods::{load_manifest, update_from_git_diff, validate_mod};
    let fixture = Fixture::new();
    let repo = fixture.0.join("repo");
    fs::create_dir_all(&repo).unwrap();
    let git = |args: &[&str]| {
        let result = Command::new("git")
            .current_dir(&repo)
            .args(args)
            .output()
            .unwrap();
        assert!(
            result.status.success(),
            "{}",
            String::from_utf8_lossy(&result.stderr)
        );
    };
    git(&["init", "--quiet"]);
    fs::write(repo.join("config.yaml"), "original\n").unwrap();
    fs::write(repo.join("mod.yaml"), "name: Original\n").unwrap();
    git(&["add", "config.yaml", "mod.yaml"]);
    git(&[
        "-c",
        "user.name=Test",
        "-c",
        "user.email=test@example.invalid",
        "commit",
        "--quiet",
        "-m",
        "base",
    ]);
    let dir = fixture.0.join("mods");
    let manifest = dir.join("mod.yaml");
    fs::write(&manifest, "version: 1\nid: Git\nname: Git\nfiles:\n  \"manual.yaml\":\n    - replace:\n        ignore-trailing-whitespace: true\n        search: 'manual'\n        replace: 'changed'\n").unwrap();
    fs::write(fixture.0.join("game/Data/manual.yaml"), "manual \n").unwrap();
    fs::write(repo.join("mod.yaml"), "name: Updated\n").unwrap();
    fs::write(repo.join("config.yaml"), "updated\n").unwrap();
    let first = update_from_git_diff(&dir, &repo, None).unwrap();
    assert_eq!(first.files.len(), 2);
    let generated = fs::read_to_string(&manifest).unwrap();
    assert!(generated.contains("\n  config.yaml:\n"));
    assert!(generated.contains("\n  manual.yaml:\n"));
    let old_patch = dir.join(
        first
            .files
            .iter()
            .find_map(|file| file.patch.as_ref())
            .unwrap(),
    );
    assert!(old_patch.is_file());
    assert!(
        validate_mod(&dir, Some(fixture.0.join("game")))
            .unwrap()
            .valid
    );
    fs::write(repo.join("config.yaml"), "updated again\n").unwrap();
    update_from_git_diff(&manifest, &repo, None).unwrap();
    assert!(
        fs::read_to_string(&manifest)
            .unwrap()
            .contains("\n  config.yaml:\n")
    );
    assert!(!old_patch.exists());
    let loaded = load_manifest(&dir).unwrap();
    assert_eq!(loaded.files.len(), 2);
    assert!(
        loaded
            .files
            .iter()
            .find(|file| file.name == "manual.yaml")
            .unwrap()
            .ignore_trailing_whitespace
    );
    assert!(
        validate_mod(&dir, Some(fixture.0.join("game")))
            .unwrap()
            .valid
    );
}

#[test]
fn toml_manifests_are_not_loaded_or_discovered() {
    use blitz_client::mods::load_manifest;
    let fixture = Fixture::new();
    let dir = fixture.0.join("mods");
    let legacy = dir.join("mod.toml");
    fs::write(&legacy, "id = 'Legacy'\nname = 'Legacy'\n").unwrap();
    assert_eq!(
        load_manifest(&legacy).unwrap_err().code,
        "MANIFEST_FORMAT_UNSUPPORTED"
    );
    assert_eq!(load_manifest(&dir).unwrap_err().code, "MANIFEST_NOT_FOUND");
    let result = fixture.run(&dir, true);
    assert_eq!(result.status.code(), Some(2));
    let value: serde_json::Value = serde_json::from_slice(&result.stdout).unwrap();
    assert_eq!(value["error"]["code"], "MANIFEST_NOT_FOUND");
}

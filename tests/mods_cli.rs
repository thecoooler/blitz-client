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
            .env_remove("CLICOLOR_FORCE")
            .env_remove("NO_COLOR")
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
    assert!(String::from_utf8_lossy(&text.stdout).contains("2 mods · 2 passed · 0 failed"));
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
    fs::write(repo.join("extra resource.yaml"), "extra\n").unwrap();
    git(&["add", "config.yaml", "mod.yaml", "extra resource.yaml"]);
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
    fs::write(repo.join("extra resource.yaml"), "updated extra\n").unwrap();
    fs::write(fixture.0.join("game/Data/extra resource.yaml"), "extra\n").unwrap();
    let first = update_from_git_diff(&dir, &repo, None).unwrap();
    assert_eq!(first.files.len(), 3);
    let generated = fs::read_to_string(&manifest).unwrap();
    assert!(generated.contains("\npatch: mod.patch\n"));
    assert!(!generated.contains("\n  config.yaml:\n"));
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
            .contains("\npatch: mod.patch\n")
    );
    assert!(old_patch.exists());
    assert!(
        fs::read_to_string(&old_patch)
            .unwrap()
            .contains("+updated again")
    );
    let loaded = load_manifest(&dir).unwrap();
    assert_eq!(loaded.files.len(), 3);
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

#[test]
fn validation_text_is_compact_and_deduplicates_file_errors() {
    let fixture = Fixture::new();
    fixture.manifest("single", "Single", "original");
    let single = fixture.run(&fixture.0.join("mods/single"), false);
    assert_eq!(
        String::from_utf8(single.stdout).unwrap(),
        "✓ Single (1 file)\n"
    );
    let dir = fixture.0.join("mods/failed");
    fs::create_dir_all(&dir).unwrap();
    fs::write(dir.join("mod.yaml"), "id: Failed\nname: Failed\nfiles:\n  config.yaml:\n    - replace:\n        search: missing\n        replace: first\n    - replace:\n        search: missing\n        replace: second\n").unwrap();
    let result = fixture.run(&fixture.0.join("mods"), false);
    assert_eq!(result.status.code(), Some(4));
    let text = String::from_utf8(result.stdout).unwrap();
    assert!(text.contains("✗ Failed (1 error)\n  config.yaml\n  MOD_CONFLICT:"));
    assert_eq!(text.matches("config.yaml").count(), 1);
    assert!(text.contains("2 mods · 1 passed · 1 failed"));
    assert!(!text.contains("Validation passed"));
    assert!(!text.contains(&fixture.0.display().to_string()));
    assert!(!text.contains('\x1b'));
}

#[test]
fn validation_colors_respect_pipes_no_color_and_json() {
    let fixture = Fixture::new();
    fixture.manifest("single", "Single", "missing");
    let run = |force: bool, no_color: bool, json: bool| {
        let mut command = Command::new(env!("CARGO_BIN_EXE_blitz-client"));
        command
            .args(["mod", "validate"])
            .arg(fixture.0.join("mods/single"))
            .arg("--game-dir")
            .arg(fixture.0.join("game"))
            .env_remove("CLICOLOR_FORCE")
            .env_remove("NO_COLOR");
        if force {
            command.env("CLICOLOR_FORCE", "1");
        }
        if no_color {
            command.env("NO_COLOR", "1");
        }
        if json {
            command.arg("--json");
        }
        command.output().unwrap()
    };
    let plain = run(false, false, false);
    assert!(!plain.stdout.contains(&0x1b));
    let colored = run(true, false, false);
    assert!(
        String::from_utf8(colored.stdout)
            .unwrap()
            .contains("\x1b[31m✗\x1b[0m")
    );
    assert!(!run(true, true, false).stdout.contains(&0x1b));
    let json = run(true, false, true);
    assert!(!json.stdout.contains(&0x1b));
    let report: serde_json::Value = serde_json::from_slice(&json.stdout).unwrap();
    assert_eq!(report["valid"], false);
    fs::write(fixture.0.join("mods/single/mod.yaml"), "files: [invalid").unwrap();
    let error = run(true, false, false);
    assert_eq!(error.status.code(), Some(2));
    assert!(
        String::from_utf8(error.stderr)
            .unwrap()
            .contains("\x1b[31mMANIFEST_INVALID\x1b[0m")
    );
}

#[test]
fn validation_groups_parse_and_resource_failures_below_successes() {
    let fixture = Fixture::new();
    fixture.manifest("a-invalid", "Invalid", "original");
    fs::write(fixture.0.join("mods/a-invalid/mod.yaml"), "files: [invalid").unwrap();
    fixture.manifest("m-failed", "Failed", "missing");
    fixture.manifest("z-valid", "Valid", "original");
    let result = fixture.run(&fixture.0.join("mods"), false);
    assert_eq!(result.status.code(), Some(4));
    let text = String::from_utf8(result.stdout).unwrap();
    let success = text.find("✓ Valid").unwrap();
    let heading = text.find("Failures (2)").unwrap();
    let invalid = text.find("✗ a-invalid").unwrap();
    let failed = text.find("✗ Failed").unwrap();
    let summary = text.find("3 mods · 1 passed · 2 failed").unwrap();
    assert!(success < heading && heading < invalid && invalid < failed && failed < summary);
    assert_eq!(text.matches("MANIFEST_INVALID").count(), 1);
    assert_eq!(text.matches("REPLACE_MATCH_COUNT").count(), 1);
    let json = fixture.run(&fixture.0.join("mods"), true);
    let report: serde_json::Value = serde_json::from_slice(&json.stdout).unwrap();
    assert_eq!(report["mods"][0]["error"]["code"], "MANIFEST_INVALID");
    assert_eq!(report["mods"][2]["report"]["mod_id"], "Valid");
    fixture.manifest("z-valid", "Valid", "missing");
    let all_failed = fixture.run(&fixture.0.join("mods"), false);
    let text = String::from_utf8(all_failed.stdout).unwrap();
    assert!(text.starts_with("Failures (3)\n"));
    assert!(text.contains("3 mods · 0 passed · 3 failed"));
}

#[test]
fn git_patch_with_context_and_repeated_hunks_round_trips() {
    use blitz_client::mods::{install_mod, remove_mod, validate_mod};
    let fixture = Fixture::new();
    let target = fixture.0.join("game/Data/config.yaml");
    let original = "header\ncontext\nold\nend\nseparator\ncontext\nold\nend\nfooter\n";
    fs::write(&target, original).unwrap();
    let dir = fixture.0.join("mods/Patched");
    fs::create_dir_all(dir.join("patches")).unwrap();
    fs::write(dir.join("patches/config.patch"), "diff --git a/config.yaml b/config.yaml\n--- a/config.yaml\n+++ b/config.yaml\n@@ -2,3 +2,3 @@\n context\n-old\n+new\n end\n@@ -6,3 +6,3 @@\n context\n-old\n+new\n end\n").unwrap();
    fs::write(
        dir.join("mod.yaml"),
        "id: Patched\nname: Patched\npatch: patches/config.patch\n",
    )
    .unwrap();
    let game = Some(fixture.0.join("game"));
    assert!(validate_mod(&dir, game.clone()).unwrap().valid);
    assert_eq!(fs::read_to_string(&target).unwrap(), original);
    install_mod(&dir, game.clone()).unwrap();
    assert_eq!(
        fs::read_to_string(&target).unwrap(),
        original.replace("old\n", "new\n")
    );
    remove_mod("Patched", game).unwrap();
    assert_eq!(fs::read_to_string(&target).unwrap(), original);
}

fn resource_patch(path: &str, before: &str, after: &str) -> String {
    format!(
        "diff --git a/{path} b/{path}\n--- a/{path}\n+++ b/{path}\n@@ -1 +1 @@\n-{before}\n+{after}\n"
    )
}

#[test]
fn multifile_patch_runs_before_text_operations_and_survives_source_removal() {
    use blitz_client::mods::{install_mod, remove_mod, validate_mod};
    let fixture = Fixture::new();
    let dir = fixture.0.join("mods/Multi");
    fs::create_dir_all(&dir).unwrap();
    use blitz_client::resources::{ContainerKind, decode_dvpl, encode_dvpl};
    let second = fixture.0.join("game/Data/second.yaml.dvpl");
    let packed = encode_dvpl(b"second\n", ContainerKind::Dvpl(2)).unwrap();
    fs::write(&second, &packed).unwrap();
    fs::write(
        dir.join("mod.patch"),
        resource_patch("config.yaml", "original", "patched")
            + &resource_patch("second.yaml", "second", "changed"),
    )
    .unwrap();
    fs::write(dir.join("mod.yaml"), "id: Multi\nname: Multi\npatch: mod.patch\nfiles:\n  config.yaml:\n    - replace:\n        search: patched\n        replace: final\n").unwrap();
    let game = Some(fixture.0.join("game"));
    assert!(validate_mod(&dir, game.clone()).unwrap().valid);
    install_mod(&dir, game.clone()).unwrap();
    assert_eq!(
        fs::read_to_string(fixture.0.join("game/Data/config.yaml")).unwrap(),
        "final\n"
    );
    assert_eq!(
        decode_dvpl(&fs::read(&second).unwrap()).unwrap().0,
        b"changed\n"
    );
    fs::remove_dir_all(&dir).unwrap();
    remove_mod("Multi", game).unwrap();
    assert_eq!(
        fs::read_to_string(fixture.0.join("game/Data/config.yaml")).unwrap(),
        "original\n"
    );
    assert_eq!(fs::read(second).unwrap(), packed);
}

#[test]
fn patch_lists_keep_order_and_folders_are_recursive_sorted_and_deduplicated() {
    use blitz_client::mods::{load_manifest, validate_mod};
    let fixture = Fixture::new();
    let dir = fixture.0.join("mods/Ordered");
    fs::create_dir_all(dir.join("patches/nested")).unwrap();
    fs::write(
        dir.join("first.patch"),
        resource_patch("config.yaml", "original", "first"),
    )
    .unwrap();
    fs::write(
        dir.join("patches/01.patch"),
        resource_patch("config.yaml", "first", "second"),
    )
    .unwrap();
    fs::write(
        dir.join("patches/nested/02.diff"),
        resource_patch("config.yaml", "second", "third"),
    )
    .unwrap();
    fs::write(dir.join("patches/ignored.txt"), "not a patch").unwrap();
    fs::write(
        dir.join("mod.yaml"),
        "id: Ordered\nname: Ordered\npatch:\n  - first.patch\n  - patches\n  - patches/01.patch\n",
    )
    .unwrap();
    assert_eq!(load_manifest(&dir).unwrap().files.len(), 3);
    assert!(
        validate_mod(&dir, Some(fixture.0.join("game")))
            .unwrap()
            .valid
    );
    fs::write(
        dir.join("mod.yaml"),
        "id: Ordered\nname: Ordered\npatch: [patches, first.patch]\n",
    )
    .unwrap();
    assert!(
        !validate_mod(&dir, Some(fixture.0.join("game")))
            .unwrap()
            .valid
    );
}

#[test]
fn patches_reject_unsupported_changes_and_unsafe_paths() {
    use blitz_client::mods::load_manifest;
    let fixture = Fixture::new();
    let dir = fixture.0.join("mods/Unsafe");
    fs::create_dir_all(&dir).unwrap();
    fs::write(
        dir.join("mod.yaml"),
        "id: Unsafe\nname: Unsafe\npatch: mod.patch\n",
    )
    .unwrap();
    for diff in [
        resource_patch("../outside", "original", "changed"),
        "diff --git a/config.yaml b/config.yaml\nnew file mode 100644\n--- /dev/null\n+++ b/config.yaml\n@@ -0,0 +1 @@\n+new\n".to_owned(),
        "diff --git a/config.yaml b/other.yaml\nrename from config.yaml\nrename to other.yaml\n".to_owned(),
        "diff --git a/config.yaml b/config.yaml\nBinary files a/config.yaml and b/config.yaml differ\n".to_owned(),
    ] {
        fs::write(dir.join("mod.patch"), diff).unwrap();
        assert!(load_manifest(&dir).is_err());
    }
    fs::write(
        dir.join("mod.yaml"),
        "id: Unsafe\nname: Unsafe\npatch: ../outside.patch\n",
    )
    .unwrap();
    assert_eq!(
        load_manifest(&dir).unwrap_err().code,
        "MANIFEST_INVALID_PATCH_PATH"
    );
}

#[test]
fn git_generation_supports_patch_only_mods_and_preserves_manual_sources() {
    use blitz_client::mods::{update_from_git_diff, validate_mod};
    let fixture = Fixture::new();
    let repo = fixture.0.join("repo");
    let dir = fixture.0.join("mods/Generated");
    fs::create_dir_all(&repo).unwrap();
    fs::create_dir_all(&dir).unwrap();
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
    git(&["add", "."]);
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
    // Headers alone are sufficient when creating the first generated patch.
    let manifest = dir.join("mod.yaml");
    fs::write(&manifest, "id: Generated\nname: Generated\n").unwrap();
    fs::write(repo.join("config.yaml"), "generated\n").unwrap();
    update_from_git_diff(&dir, &repo, None).unwrap();
    assert!(
        validate_mod(&dir, Some(fixture.0.join("game")))
            .unwrap()
            .valid
    );
    fs::write(
        dir.join("manual.diff"),
        resource_patch("config.yaml", "generated", "manual"),
    )
    .unwrap();
    fs::write(
        &manifest,
        "id: Generated\nname: Generated\npatch: [mod.patch, manual.diff]\n",
    )
    .unwrap();
    let updated = update_from_git_diff(&dir, &repo, None).unwrap();
    assert_eq!(updated.patch, ["mod.patch", "manual.diff"]);
    assert!(
        validate_mod(&dir, Some(fixture.0.join("game")))
            .unwrap()
            .valid
    );
    let manifest_before = fs::read(&manifest).unwrap();
    let patch_before = fs::read(dir.join("mod.patch")).unwrap();
    git(&["restore", "config.yaml"]);
    assert_eq!(
        update_from_git_diff(&dir, &repo, None).unwrap_err().code,
        "GIT_DIFF_EMPTY"
    );
    assert_eq!(fs::read(&manifest).unwrap(), manifest_before);
    assert_eq!(fs::read(dir.join("mod.patch")).unwrap(), patch_before);
}

#[test]
fn mod_list_reports_saved_metadata_and_reinstall_status() {
    use blitz_client::mods::install_mod;
    let fixture = Fixture::new();
    let run = |args: &[&str]| {
        Command::new(env!("CARGO_BIN_EXE_blitz-client"))
            .args(["mod"])
            .args(args)
            .arg("--game-dir")
            .arg(fixture.0.join("game"))
            .output()
            .unwrap()
    };
    let empty = run(&["list", "--json"]);
    assert!(empty.status.success());
    let report: serde_json::Value = serde_json::from_slice(&empty.stdout).unwrap();
    assert_eq!(report["mods"], serde_json::json!([]));
    assert!(!fixture.0.join("game/.tanks-blitz-client").exists());
    assert_eq!(run(&["list"]).stdout, b"No mods installed\n");
    fixture.manifest("a", "A", "original");
    fs::write(fixture.0.join("mods/a/mod.yaml"), "id: A\nname: First\nfiles:\n  config.yaml:\n    - replace:\n        search: original\n        replace: intermediate\n    - replace:\n        search: intermediate\n        replace: changed\n").unwrap();
    let b = fixture.0.join("mods/b");
    fs::create_dir_all(&b).unwrap();
    fs::write(fixture.0.join("game/Data/second.yaml"), "second\n").unwrap();
    fs::write(
        b.join("mod.yaml"),
        "id: B\nname: Second\nfiles:\n  second.yaml:\n    - clear: {}\n",
    )
    .unwrap();
    install_mod(&b, Some(fixture.0.join("game"))).unwrap();
    install_mod(fixture.0.join("mods/a"), Some(fixture.0.join("game"))).unwrap();
    let reinstall = run(&[
        "install",
        fixture.0.join("mods/a").to_str().unwrap(),
        "--json",
    ]);
    assert!(reinstall.status.success());
    let report: serde_json::Value = serde_json::from_slice(&reinstall.stdout).unwrap();
    assert_eq!(report["installed"], true);
    assert_eq!(report["reinstalled"], true);
    assert_eq!(
        fs::read(fixture.0.join("game/Data/config.yaml")).unwrap(),
        b"changed\n"
    );
    let text = run(&["install", fixture.0.join("mods/a").to_str().unwrap()]);
    assert!(text.status.success());
    assert!(
        String::from_utf8(text.stdout)
            .unwrap()
            .starts_with("Reinstalled A — First\n")
    );
    fs::remove_dir_all(fixture.0.join("mods")).unwrap();
    let listed = run(&["list", "--json"]);
    assert!(listed.status.success());
    let report: serde_json::Value = serde_json::from_slice(&listed.stdout).unwrap();
    assert_eq!(
        report["mods"],
        serde_json::json!([
            {"id": "A", "name": "First", "files": ["config.yaml"]},
            {"id": "B", "name": "Second", "files": ["second.yaml"]}
        ])
    );
    assert_eq!(
        run(&["list"]).stdout,
        "A — First (1 file)\nB — Second (1 file)\n".as_bytes()
    );
    assert_eq!(run(&["list", "unexpected"]).status.code(), Some(2));
}

#[test]
fn mod_list_reports_invalid_state_instead_of_hiding_installed_mods() {
    let fixture = Fixture::new();
    let dir = fixture.0.join("game/.tanks-blitz-client");
    fs::create_dir_all(&dir).unwrap();
    for (text, code) in [
        ("{broken", "STATE_INVALID"),
        ("null", "STATE_INVALID"),
        (r#"{"version":"2","mods":[]}"#, "STATE_INVALID"),
        (r#"{"version":2,"mods":{}}"#, "STATE_INVALID"),
        (
            r#"{"version":999,"mods":[{"id":"A"}]}"#,
            "STATE_VERSION_UNSUPPORTED",
        ),
    ] {
        fs::write(dir.join("state.json"), text).unwrap();
        let output = Command::new(env!("CARGO_BIN_EXE_blitz-client"))
            .args(["mod", "list", "--json", "--game-dir"])
            .arg(fixture.0.join("game"))
            .output()
            .unwrap();
        assert_eq!(output.status.code(), Some(2));
        let report: serde_json::Value = serde_json::from_slice(&output.stdout).unwrap();
        assert_eq!(report["error"]["code"], code);
        assert_eq!(fs::read_to_string(dir.join("state.json")).unwrap(), text);
    }
}

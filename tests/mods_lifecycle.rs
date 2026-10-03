#![cfg(feature = "mods")]

use blitz_client::mods::{install_mod, list_installed_mods, remove_mod, validate_mod};
use std::collections::BTreeMap;
use std::fs;
use std::path::PathBuf;
use std::sync::atomic::{AtomicU64, Ordering};

// Fixed DVPL bytes: expected output does not use the production encoder.
const PACKED_ORIGINAL: &[u8] = &[
    240, 2, 112, 97, 99, 107, 101, 100, 58, 32, 111, 114, 105, 103, 105, 110, 97, 108, 10, 17, 0,
    0, 0, 19, 0, 0, 0, 33, 171, 11, 170, 2, 0, 0, 0, 68, 86, 80, 76,
];
const PACKED_INSTALLED: &[u8] = &[
    240, 3, 112, 97, 99, 107, 101, 100, 58, 32, 105, 110, 115, 116, 97, 108, 108, 101, 100, 10, 18,
    0, 0, 0, 20, 0, 0, 0, 218, 131, 7, 197, 2, 0, 0, 0, 68, 86, 80, 76,
];

struct Scenario {
    root: PathBuf,
    originals: BTreeMap<String, Vec<u8>>,
}

fn patch(path: &str, line: usize, before: &str, after: &str) -> String {
    format!(
        "diff --git a/{path} b/{path}\n--- a/{path}\n+++ b/{path}\n@@ -{line} +{line} @@\n-{before}\n+{after}\n"
    )
}

impl Scenario {
    fn new() -> Self {
        static NEXT: AtomicU64 = AtomicU64::new(0);
        let root = std::env::temp_dir().join(format!(
            "blitz-mod-lifecycle-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        let originals = BTreeMap::from([
            (
                "shared.yaml".into(),
                b"alpha: original\nseparator\nbeta: original\n".to_vec(),
            ),
            ("old-only.yaml".into(), b"old: original\n".to_vec()),
            ("new-only.yaml".into(), b"new: original\n".to_vec()),
            ("packed.yaml.dvpl".into(), PACKED_ORIGINAL.to_vec()),
            ("untouched.bin".into(), vec![0xff, 0x00, 0x80, 0xfe]),
        ]);
        fs::create_dir_all(root.join("game/Data")).unwrap();
        for (name, bytes) in &originals {
            fs::write(root.join("game/Data").join(name), bytes).unwrap();
        }
        let scenario = Self { root, originals };
        // Two versions share an id but change both the content and affected files.
        scenario.write_mod(
            "alpha-v1",
            "Alpha",
            patch("shared.yaml", 1, "alpha: original", "alpha: v1")
                + &patch("old-only.yaml", 1, "old: original", "old: v1"),
        );
        scenario.write_mod(
            "alpha-v2",
            "Alpha",
            patch("shared.yaml", 1, "alpha: original", "alpha: v2")
                + &patch("new-only.yaml", 1, "new: original", "new: v2"),
        );
        scenario.write_mod(
            "beta",
            "Beta",
            patch("shared.yaml", 3, "beta: original", "beta: installed")
                + &patch("packed.yaml", 1, "packed: original", "packed: installed"),
        );
        scenario
    }

    fn write_mod(&self, dir: &str, id: &str, diff: String) {
        let path = self.root.join("mods").join(dir);
        fs::create_dir_all(&path).unwrap();
        fs::write(
            path.join("mod.yaml"),
            format!("id: {id}\nname: {id}\npatch: mod.patch\n"),
        )
        .unwrap();
        fs::write(path.join("mod.patch"), diff).unwrap();
    }

    fn install(&self, dir: &str) {
        let report = install_mod(
            self.root.join("mods").join(dir),
            Some(self.root.join("game")),
        )
        .unwrap();
        assert!(report.installed, "{dir}: {report:?}");
    }

    fn remove(&self, id: &str) {
        assert!(
            remove_mod(id, Some(self.root.join("game")))
                .unwrap()
                .removed
        );
    }

    fn assert_files(&self, alpha: Option<u8>, beta: bool) {
        let expected = BTreeMap::from([
            (
                "shared.yaml",
                format!(
                    "alpha: {}\nseparator\nbeta: {}\n",
                    match alpha {
                        Some(1) => "v1",
                        Some(2) => "v2",
                        None => "original",
                        _ => unreachable!(),
                    },
                    if beta { "installed" } else { "original" }
                ),
            ),
            (
                "old-only.yaml",
                format!(
                    "old: {}\n",
                    if alpha == Some(1) { "v1" } else { "original" }
                ),
            ),
            (
                "new-only.yaml",
                format!(
                    "new: {}\n",
                    if alpha == Some(2) { "v2" } else { "original" }
                ),
            ),
        ]);
        for (name, text) in expected {
            let bytes = fs::read(self.root.join("game/Data").join(name)).unwrap();
            assert_eq!(
                bytes,
                text.as_bytes(),
                "{name}: Alpha={alpha:?}, Beta={beta}"
            );
        }
        assert_eq!(
            fs::read(self.root.join("game/Data/packed.yaml.dvpl")).unwrap(),
            if beta {
                PACKED_INSTALLED
            } else {
                PACKED_ORIGINAL
            },
            "packed.yaml.dvpl: Alpha={alpha:?}, Beta={beta}"
        );
        assert_eq!(
            fs::read(self.root.join("game/Data/untouched.bin")).unwrap(),
            self.originals["untouched.bin"]
        );
        // Complete removal must restore original bytes, including compression.
        if alpha.is_none() && !beta {
            for (name, original) in &self.originals {
                assert_eq!(
                    fs::read(self.root.join("game/Data").join(name)).unwrap(),
                    *original,
                    "{name}"
                );
            }
        }
        let state_path = self.root.join("game/.tanks-blitz-client/state.json");
        if alpha.is_some() || beta {
            assert!(state_path.is_file(), "installed mods must have saved state");
        }
        if state_path.exists() {
            let state: serde_json::Value =
                serde_json::from_slice(&fs::read(state_path).unwrap()).unwrap();
            let mut ids = state["mods"]
                .as_array()
                .unwrap()
                .iter()
                .map(|m| m["id"].as_str().unwrap())
                .collect::<Vec<_>>();
            ids.sort();
            let mut expected_ids = Vec::new();
            if alpha.is_some() {
                expected_ids.push("Alpha");
            }
            if beta {
                expected_ids.push("Beta");
            }
            assert_eq!(ids, expected_ids);
        }
    }
}

impl Drop for Scenario {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.root);
    }
}

#[test]
fn mod_versions_survive_repeated_install_remove_and_recomposition() {
    let scenario = Scenario::new();
    scenario.assert_files(None, false);
    scenario.install("alpha-v1");
    scenario.assert_files(Some(1), false);
    scenario.install("beta");
    scenario.assert_files(Some(1), true);
    scenario.remove("Alpha");
    scenario.assert_files(None, true);
    scenario.install("alpha-v2");
    scenario.assert_files(Some(2), true);
    scenario.remove("Beta");
    scenario.assert_files(Some(2), false);
    scenario.install("beta");
    scenario.assert_files(Some(2), true);
    // Removing source files must not affect subsequent recomposition.
    fs::remove_dir_all(scenario.root.join("mods/alpha-v2")).unwrap();
    scenario.remove("Alpha");
    scenario.assert_files(None, true);
    scenario.install("alpha-v1");
    scenario.assert_files(Some(1), true);
    scenario.remove("Beta");
    scenario.assert_files(Some(1), false);
    scenario.remove("Alpha");
    scenario.assert_files(None, false);
    // Start again in reverse installation order after all snapshots were released.
    scenario.install("beta");
    scenario.assert_files(None, true);
    scenario.install("alpha-v1");
    scenario.assert_files(Some(1), true);
    scenario.remove("Alpha");
    scenario.assert_files(None, true);
    scenario.remove("Beta");
    scenario.assert_files(None, false);
}

#[test]
fn invalid_new_version_does_not_partially_install_or_change_other_mods() {
    let scenario = Scenario::new();
    scenario.install("alpha-v1");
    scenario.install("beta");
    scenario.remove("Alpha");
    scenario.assert_files(None, true);
    let state_path = scenario.root.join("game/.tanks-blitz-client/state.json");
    let state_before = fs::read(&state_path).unwrap();
    // The shared resource can be prepared, but the next resource cannot match.
    scenario.write_mod(
        "alpha-v2",
        "Alpha",
        patch("shared.yaml", 1, "alpha: original", "alpha: v2")
            + &patch("new-only.yaml", 1, "missing context", "new: v2"),
    );
    let error = install_mod(
        scenario.root.join("mods/alpha-v2"),
        Some(scenario.root.join("game")),
    )
    .unwrap_err();
    assert_eq!(error.code, "PATCH_CONTEXT_MISMATCH");
    scenario.assert_files(None, true);
    assert_eq!(fs::read(&state_path).unwrap(), state_before);
    // A failed attempt must not prevent a later installation of the old version.
    scenario.install("alpha-v1");
    scenario.assert_files(Some(1), true);
    scenario.remove("Beta");
    scenario.assert_files(Some(1), false);
    scenario.remove("Alpha");
    scenario.assert_files(None, false);
}

#[test]
fn reinstall_replaces_versions_and_preserves_other_mods() {
    let scenario = Scenario::new();
    assert!(
        list_installed_mods(Some(scenario.root.join("game")))
            .unwrap()
            .mods
            .is_empty()
    );
    scenario.install("beta");
    scenario.install("alpha-v1");
    scenario.assert_files(Some(1), true);
    assert!(
        validate_mod(
            scenario.root.join("mods/alpha-v2"),
            Some(scenario.root.join("game"))
        )
        .unwrap()
        .valid
    );
    for (dir, version) in [
        ("alpha-v2", 2),
        ("alpha-v2", 2),
        ("alpha-v1", 1),
        ("alpha-v2", 2),
    ] {
        let report = install_mod(
            scenario.root.join("mods").join(dir),
            Some(scenario.root.join("game")),
        )
        .unwrap();
        assert!(report.installed && report.reinstalled);
        scenario.assert_files(Some(version), true);
        let listed = list_installed_mods(Some(scenario.root.join("game"))).unwrap();
        assert_eq!(
            listed
                .mods
                .iter()
                .map(|item| item.id.as_str())
                .collect::<Vec<_>>(),
            ["Alpha", "Beta"]
        );
        assert_eq!(
            listed.mods[0].files,
            if version == 1 {
                vec!["old-only.yaml", "shared.yaml"]
            } else {
                vec!["new-only.yaml", "shared.yaml"]
            }
        );
    }
    fs::remove_dir_all(scenario.root.join("mods")).unwrap();
    let listed = list_installed_mods(Some(scenario.root.join("game"))).unwrap();
    assert_eq!(listed.mods.len(), 2);
    assert_eq!(listed.mods[1].name, "Beta");
    assert_eq!(listed.mods[1].files, ["packed.yaml", "shared.yaml"]);
    scenario.remove("Beta");
    scenario.assert_files(Some(2), false);
    scenario.remove("Alpha");
    scenario.assert_files(None, false);
    assert!(
        list_installed_mods(Some(scenario.root.join("game")))
            .unwrap()
            .mods
            .is_empty()
    );
}

#[test]
fn failed_reinstall_preserves_old_version_and_saved_state() {
    let scenario = Scenario::new();
    scenario.install("alpha-v1");
    scenario.install("beta");
    let state_path = scenario.root.join("game/.tanks-blitz-client/state.json");
    let before = fs::read(&state_path).unwrap();
    scenario.write_mod(
        "alpha-v2",
        "Alpha",
        patch("shared.yaml", 1, "alpha: original", "alpha: v2")
            + &patch("new-only.yaml", 1, "missing", "new: v2"),
    );
    let error = install_mod(
        scenario.root.join("mods/alpha-v2"),
        Some(scenario.root.join("game")),
    )
    .unwrap_err();
    assert_eq!(error.code, "PATCH_CONTEXT_MISMATCH");
    scenario.assert_files(Some(1), true);
    assert_eq!(fs::read(state_path).unwrap(), before);
    scenario.remove("Alpha");
    scenario.assert_files(None, true);
    scenario.remove("Beta");
    scenario.assert_files(None, false);
}

#[test]
fn reinstall_rejects_external_changes_to_old_only_files() {
    let scenario = Scenario::new();
    scenario.install("alpha-v1");
    scenario.install("beta");
    let target = scenario.root.join("game/Data/old-only.yaml");
    let original = fs::read(&target).unwrap();
    let state_path = scenario.root.join("game/.tanks-blitz-client/state.json");
    let before = fs::read(&state_path).unwrap();
    fs::write(&target, b"changed outside installer\n").unwrap();
    let error = install_mod(
        scenario.root.join("mods/alpha-v2"),
        Some(scenario.root.join("game")),
    )
    .unwrap_err();
    assert_eq!(error.code, "INSTALLED_FILE_CHANGED");
    assert!(
        !validate_mod(
            scenario.root.join("mods/alpha-v2"),
            Some(scenario.root.join("game"))
        )
        .unwrap()
        .valid
    );
    assert_eq!(fs::read(&target).unwrap(), b"changed outside installer\n");
    assert_eq!(fs::read(state_path).unwrap(), before);
    fs::write(&target, original).unwrap();
    scenario.assert_files(Some(1), true);
}

#[test]
fn dropping_a_patch_required_by_another_mod_keeps_old_installation() {
    let scenario = Scenario::new();
    scenario.install("alpha-v1");
    scenario.write_mod(
        "beta",
        "Beta",
        patch("old-only.yaml", 1, "old: v1", "old: v1 with beta"),
    );
    scenario.install("beta");
    let before = scenario
        .originals
        .keys()
        .map(|name| {
            (
                name.clone(),
                fs::read(scenario.root.join("game/Data").join(name)).unwrap(),
            )
        })
        .collect::<BTreeMap<_, _>>();
    let state_path = scenario.root.join("game/.tanks-blitz-client/state.json");
    let state_before = fs::read(&state_path).unwrap();
    let validation = validate_mod(
        scenario.root.join("mods/alpha-v2"),
        Some(scenario.root.join("game")),
    )
    .unwrap();
    assert!(!validation.valid);
    assert!(
        validation
            .files
            .iter()
            .any(|file| file.path == "old-only.yaml" && !file.valid)
    );
    let error = install_mod(
        scenario.root.join("mods/alpha-v2"),
        Some(scenario.root.join("game")),
    )
    .unwrap_err();
    assert_eq!(error.code, "PATCH_CONTEXT_MISMATCH");
    for (name, bytes) in before {
        assert_eq!(
            fs::read(scenario.root.join("game/Data").join(name)).unwrap(),
            bytes
        );
    }
    assert_eq!(fs::read(state_path).unwrap(), state_before);
    scenario.remove("Beta");
    scenario.assert_files(Some(1), false);
    scenario.remove("Alpha");
    scenario.assert_files(None, false);
}

#[test]
fn discovery_returns_paths_even_for_broken_mods_and_conflicting_manifests() {
    use blitz_client::mods::{discover_mods, load_manifest};
    let scenario = Scenario::new();
    let root = scenario.root.join("mods");
    let empty = scenario.root.join("empty");
    fs::create_dir_all(&empty).unwrap();
    assert!(discover_mods(&empty).unwrap().is_empty());
    let bad = root.join("nested/broken");
    fs::create_dir_all(&bad).unwrap();
    fs::write(bad.join("mod.yaml"), "files: [broken").unwrap();
    fs::write(bad.join("mod.yml"), "id: Other\nname: Other\n").unwrap();
    let paths = discover_mods(&root).unwrap();
    assert_eq!(
        paths,
        [
            root.join("alpha-v1/mod.yaml"),
            root.join("alpha-v2/mod.yaml"),
            root.join("beta/mod.yaml"),
            bad.join("mod.yaml")
        ]
    );
    assert!(load_manifest(paths.last().unwrap()).is_err());
    fs::remove_file(bad.join("mod.yml")).unwrap();
    assert_eq!(
        load_manifest(bad.join("mod.yaml")).unwrap_err().code,
        "MANIFEST_INVALID"
    );
    assert_eq!(load_manifest(&paths[0]).unwrap().id, "Alpha");
    assert!(discover_mods(scenario.root.join("missing")).is_err());
    #[cfg(unix)]
    {
        std::os::unix::fs::symlink(&root, bad.join("cycle")).unwrap();
        assert_eq!(discover_mods(&root).unwrap(), paths);
    }
}

#[test]
fn reinstall_restores_dropped_binary_clear_from_original_bytes() {
    let scenario = Scenario::new();
    scenario.install("beta");
    let dir = scenario.root.join("mods/alpha-v1");
    fs::write(
        dir.join("mod.yaml"),
        "id: Alpha\nname: Alpha\nfiles:\n  untouched.bin:\n    - clear: {}\n",
    )
    .unwrap();
    scenario.install("alpha-v1");
    assert_eq!(
        fs::read(scenario.root.join("game/Data/untouched.bin")).unwrap(),
        b""
    );
    let report = install_mod(
        scenario.root.join("mods/alpha-v2"),
        Some(scenario.root.join("game")),
    )
    .unwrap();
    assert!(report.reinstalled);
    scenario.assert_files(Some(2), true);
    scenario.remove("Alpha");
    scenario.assert_files(None, true);
    scenario.remove("Beta");
    scenario.assert_files(None, false);
}

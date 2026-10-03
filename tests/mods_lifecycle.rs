#![cfg(feature = "mods")]

use blitz_client::mods::{install_mod, remove_mod};
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

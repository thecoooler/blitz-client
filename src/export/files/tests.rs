use super::*;
use crate::resources::encode_dvpl;
use std::sync::atomic::{AtomicU64, Ordering};

struct Fixture(PathBuf);
impl Fixture {
    fn new() -> Self {
        static NEXT: AtomicU64 = AtomicU64::new(0);
        let root = std::env::temp_dir().join(format!(
            "blitz-files-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        fs::create_dir_all(root.join("game/Data")).unwrap();
        fs::create_dir_all(root.join("packs")).unwrap();
        Self(root)
    }

    fn write(&self, relative: &str, bytes: &[u8]) -> PathBuf {
        let path = self.0.join(relative);
        fs::create_dir_all(path.parent().unwrap()).unwrap();
        fs::write(&path, bytes).unwrap();
        path
    }

    fn packed(&self, relative: &str, bytes: &[u8], compression: u32) -> PathBuf {
        self.write(
            relative,
            &encode_dvpl(bytes, ContainerKind::Dvpl(compression)).unwrap(),
        )
    }

    fn options(&self) -> ClientFilesExportOptions {
        let mut options = ClientFilesExportOptions::new(self.0.join("output"));
        options.game_dir = Some(self.0.join("game"));
        options.cache_dir = Some(self.0.join("packs"));
        options
    }

    fn resources(&self) {
        // The shadowed broken installation file must never be decoded.
        self.write("game/Data/UI/battle.yaml.dvpl", b"broken installation copy");
        self.packed("packs/UI/battle.yaml.dvpl", b"cache config", 1);
        self.write("game/Data/Gfx/image.webp", b"old image");
        self.packed("packs/Gfx/image.webp.dvpl", b"new image", 2);
        self.packed("game/Data/3d/tank.sc2.dvpl", b"scene", 3);
        self.write("game/Data/note.txt", b"text");
        self.write("game/Data/twin.yaml.dvpl", b"invalid packed sibling");
        self.write("game/Data/twin.yaml", b"plain config wins");
        self.write("game/Data/unknown.bin", b"unknown bytes");
    }
}
impl Drop for Fixture {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.0);
    }
}

#[test]
fn merges_unpacked_paths_cache_wins_and_repeated_export_keeps_identical_files() {
    let fixture = Fixture::new();
    fixture.resources();
    fixture.write("output/unrelated.txt", b"keep me");
    let mut progress = Vec::new();
    let report = export_client_files_with_progress(&fixture.options(), |report| {
        progress.push(report.written_count + report.unchanged_count);
        Ok(())
    })
    .unwrap();
    assert_eq!(report.file_count, 6);
    assert_eq!(report.written_count, 6);
    assert_eq!(report.cache_file_count, 2);
    assert_eq!(report.dvpl_count, 3);
    assert_eq!(progress, vec![0, 1, 2, 3, 4, 5, 6]);
    for (path, expected) in [
        ("UI/battle.yaml", &b"cache config"[..]),
        ("Gfx/image.webp", &b"new image"[..]),
        ("3d/tank.sc2", &b"scene"[..]),
        ("twin.yaml", &b"plain config wins"[..]),
        ("unknown.bin", &b"unknown bytes"[..]),
        ("unrelated.txt", &b"keep me"[..]),
    ] {
        assert_eq!(fs::read(report.output_dir.join(path)).unwrap(), expected);
        assert!(!report.output_dir.join(format!("{path}.dvpl")).exists());
    }
    let modified = fs::metadata(report.output_dir.join("UI/battle.yaml"))
        .unwrap()
        .modified()
        .unwrap();
    let again = export_client_files(&fixture.options()).unwrap();
    assert_eq!(again.written_count, 0);
    assert_eq!(again.unchanged_count, 6);
    assert_eq!(
        fs::metadata(report.output_dir.join("UI/battle.yaml"))
            .unwrap()
            .modified()
            .unwrap(),
        modified
    );
}

#[test]
fn selects_multiple_categories_by_unpacked_extension() {
    let fixture = Fixture::new();
    fixture.resources();
    let mut options = fixture.options();
    options.types = vec![ClientFileType::Configs, ClientFileType::Models];
    let report = export_client_files(&options).unwrap();
    assert_eq!(report.file_count, 4);
    assert!(!report.output_dir.join("Gfx").exists());
    assert!(!report.output_dir.join("unknown.bin").exists());
    for (file, expected) in [
        ("a.YAML.DVPL", ClientFileType::Configs),
        ("a.actions", ClientFileType::Configs),
        ("a.dds.dvpl", ClientFileType::Images),
        ("a.scg", ClientFileType::Models),
        ("a.animation", ClientFileType::Models),
        ("a.pck.dvpl", ClientFileType::Sounds),
        ("a.ttf", ClientFileType::Fonts),
        ("a.slh", ClientFileType::Shaders),
        ("no-extension", ClientFileType::Other),
    ] {
        assert_eq!(client_file_type(Path::new(file)), expected);
    }
    assert!("unknown".parse::<ClientFileType>().is_err());
}

#[test]
fn invalid_dvpl_preserves_existing_destination_and_reports_source() {
    let fixture = Fixture::new();
    let source = fixture.packed("game/Data/config.yaml.dvpl", b"config", 0);
    let mut bytes = fs::read(&source).unwrap();
    bytes[0] ^= 1;
    fs::write(&source, bytes).unwrap();
    let output = fixture.write("output/config.yaml", b"previous export");
    let error = export_client_files(&fixture.options()).unwrap_err();
    assert_eq!(error.code, "DVPL_CRC_FAILED");
    assert_eq!(error.path.as_ref(), Some(&source.canonicalize().unwrap()));
    assert_eq!(fs::read(&output).unwrap(), b"previous export");
    fs::write(&source, b"invalid footer").unwrap();
    assert_eq!(
        export_client_files(&fixture.options()).unwrap_err().code,
        "DVPL_INVALID"
    );
}

#[test]
fn rejects_source_overlap_and_missing_explicit_cache_before_writing() {
    let fixture = Fixture::new();
    let mut options = fixture.options();
    for output in [
        fixture.0.join("game/Data/nested"),
        fixture.0.clone(),
        fixture.0.join("packs/export"),
    ] {
        options.output_dir = output;
        assert_eq!(
            export_client_files(&options).unwrap_err().code,
            "FILE_EXPORT_OUTPUT_INVALID"
        );
    }
    assert!(!fixture.0.join("game/Data/nested").exists());
    assert!(!fixture.0.join("packs/export").exists());
    let file = fixture.write("not-a-directory", b"keep me");
    options.output_dir = file.join("export");
    assert_eq!(
        export_client_files(&options).unwrap_err().code,
        "FILE_EXPORT_OUTPUT_INVALID"
    );
    assert_eq!(fs::read(file).unwrap(), b"keep me");
    options = fixture.options();
    options.cache_dir = Some(fixture.0.join("missing-packs"));
    assert_eq!(
        export_client_files(&options).unwrap_err().code,
        "FILE_EXPORT_SOURCE_NOT_FOUND"
    );
    options.no_cache = true;
    assert_eq!(
        export_client_files(&options).unwrap_err().code,
        "FILE_EXPORT_OPTIONS_INVALID"
    );
    options.cache_dir = None;
    fixture.write("game/Data/config.yaml", b"installation");
    fixture.write("packs/config.yaml", b"cache");
    let report = export_client_files(&options).unwrap();
    assert!(report.cache_dir.is_none());
    assert_eq!(
        fs::read(report.output_dir.join("config.yaml")).unwrap(),
        b"installation"
    );
}

#[cfg(unix)]
#[test]
fn skips_source_symlinks_and_rejects_output_symlinks_and_source_aliases() {
    use std::os::unix::fs::symlink;
    let fixture = Fixture::new();
    let real = fixture.write("game/Data/UI/config.yaml", b"config");
    symlink(&real, fixture.0.join("game/Data/linked.yaml")).unwrap();
    let report = export_client_files(&fixture.options()).unwrap();
    assert_eq!(report.skipped_symlink_count, 1);
    assert!(!report.output_dir.join("linked.yaml").exists());
    symlink(fixture.0.join("game/Data"), fixture.0.join("alias")).unwrap();
    let mut options = fixture.options();
    options.output_dir = fixture.0.join("alias/nested");
    assert_eq!(
        export_client_files(&options).unwrap_err().code,
        "FILE_EXPORT_OUTPUT_INVALID"
    );
    options = fixture.options();
    fs::remove_dir_all(&options.output_dir).unwrap();
    fs::create_dir_all(&options.output_dir).unwrap();
    let outside = fixture.write("outside/config.yaml", b"untouched");
    symlink(outside.parent().unwrap(), options.output_dir.join("UI")).unwrap();
    assert_eq!(
        export_client_files(&options).unwrap_err().code,
        "FILE_EXPORT_OUTPUT_INVALID"
    );
    assert_eq!(fs::read(outside).unwrap(), b"untouched");
}

#[test]
fn progress_failure_stops_before_writing_next_file() {
    let fixture = Fixture::new();
    fixture.write("game/Data/first.txt", b"first");
    fixture.write("game/Data/second.txt", b"second");
    let options = fixture.options();
    let error = export_client_files_with_progress(&options, |report| {
        if report.written_count == 1 {
            Err(Error::new("CANCELLED", "stop"))
        } else {
            Ok(())
        }
    })
    .unwrap_err();
    assert_eq!(error.code, "CANCELLED");
    assert!(options.output_dir.join("first.txt").exists());
    assert!(!options.output_dir.join("second.txt").exists());
}

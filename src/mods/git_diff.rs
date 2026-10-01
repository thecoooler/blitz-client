use crate::{Error, Result};
use std::path::Path;
use std::process::Command;

pub(super) struct GitChange {
    pub path: String,
    pub diff: String,
}

pub(super) fn collect_changes(repo: &Path, base: &str) -> Result<Vec<GitChange>> {
    if !repo.is_dir() {
        return Err(Error::at(
            "GIT_REPO_NOT_FOUND",
            "repository directory does not exist",
            repo,
        ));
    }
    if base.is_empty() || base.starts_with('-') {
        return Err(Error::new(
            "GIT_REVISION_INVALID",
            "base revision must be a non-empty revision name",
        ));
    }
    let status = run_git(
        repo,
        ["diff", "--name-status", "-z", "--no-renames", base, "--"],
    )?;
    let fields = status
        .split(|byte| *byte == 0)
        .filter(|part| !part.is_empty())
        .collect::<Vec<_>>();
    let mut eligible = Vec::new();
    let mut unsupported = Vec::new();
    let mut index = 0;
    while index + 1 < fields.len() {
        let state = std::str::from_utf8(fields[index]).unwrap_or("");
        let path = std::str::from_utf8(fields[index + 1]).map_err(|_| {
            Error::new(
                "GIT_PATH_NOT_UTF8",
                "repository contains a modified path that is not UTF-8",
            )
        })?;
        if path == "mod.toml" || path.starts_with("patches/") {
            index += 2;
            continue;
        } else if state == "M" {
            eligible.push(path.to_owned());
        } else {
            unsupported.push(format!("{state} {path}"));
        }
        index += 2;
    }
    if !unsupported.is_empty() {
        return Err(Error::new(
            "GIT_DIFF_UNSUPPORTED_CHANGE",
            format!(
                "only modifications to existing tracked files are supported; found {}",
                unsupported.join(", ")
            ),
        ));
    }

    let mut changes = Vec::new();
    for path in eligible {
        if path.ends_with(".dvpl") {
            return Err(Error::at(
                "GIT_DIFF_DVPL_INPUT",
                "generate diffs from unpacked text resources, not compressed .dvpl containers",
                path,
            ));
        }
        let diff = run_git(
            repo,
            [
                "diff",
                "--no-ext-diff",
                "--no-textconv",
                "--no-color",
                "--no-renames",
                "--unified=3",
                base,
                "--",
                &path,
            ],
        )?;
        let diff = String::from_utf8(diff).map_err(|_| {
            Error::at(
                "GIT_DIFF_NOT_UTF8",
                "git produced a non-UTF-8 text diff",
                &path,
            )
        })?;
        if diff.contains("Binary files ") || diff.contains("GIT binary patch") {
            return Err(Error::at(
                "GIT_DIFF_BINARY_FILE",
                "binary changes cannot be converted to a YAML text patch",
                &path,
            ));
        }
        if !diff.contains("@@ ") {
            continue;
        }
        changes.push(GitChange { path, diff });
    }
    Ok(changes)
}

fn run_git<const N: usize>(repo: &Path, args: [&str; N]) -> Result<Vec<u8>> {
    let output = Command::new("git")
        .arg("-C")
        .arg(repo)
        .args(args)
        .output()
        .map_err(|e| Error::new("GIT_UNAVAILABLE", format!("could not start git: {e}")))?;
    if !output.status.success() {
        let detail = String::from_utf8_lossy(&output.stderr).trim().to_owned();
        return Err(Error::new(
            "GIT_COMMAND_FAILED",
            if detail.is_empty() {
                "git diff failed".to_owned()
            } else {
                detail
            },
        ));
    }
    Ok(output.stdout)
}

use super::manifest::safe_relative_path;
use crate::{Error, Result};

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum Kind {
    Context,
    Remove,
    Add,
}

#[derive(Debug, Clone)]
struct HunkLine {
    kind: Kind,
    text: String,
    no_newline: bool,
}

#[derive(Debug, Clone)]
struct Hunk {
    old_start: usize,
    old_len: usize,
    new_len: usize,
    lines: Vec<HunkLine>,
}

pub(super) struct FilePatch {
    pub path: String,
    pub diff: String,
}

/// Split a multi-file Git/unified diff, validating its paths and hunk framing.
pub(super) fn split_patch(diff: &str) -> Result<Vec<FilePatch>> {
    let raw_lines = diff.split_inclusive('\n').collect::<Vec<_>>();
    let lines = raw_lines
        .iter()
        .map(|line| strip_lf(line).trim_end_matches('\r'))
        .collect::<Vec<_>>();
    let mut offsets = vec![0];
    for line in &raw_lines {
        offsets.push(offsets.last().unwrap() + line.len());
    }
    let mut sections = Vec::new();
    let mut start = None;
    let mut paths = None;
    let mut index = 0;
    while index < lines.len() {
        let line = lines[index];
        if line.starts_with("diff --git ") {
            if let Some(begin) = start {
                sections.push(make_file_patch(
                    &diff[offsets[begin]..offsets[index]],
                    paths.take(),
                )?);
            }
            start = Some(index);
        } else if line.starts_with("--- ")
            && lines
                .get(index + 1)
                .is_some_and(|line| line.starts_with("+++ "))
        {
            if paths.is_some() {
                sections.push(make_file_patch(
                    &diff[offsets[start.unwrap()]..offsets[index]],
                    paths.take(),
                )?);
                start = Some(index);
            } else if start.is_none() {
                start = Some(index);
            }
            let old = decode_header_path(&line[4..])?;
            let new = decode_header_path(&lines[index + 1][4..])?;
            paths = Some((old, new));
            index += 1;
        } else if line.starts_with("@@ ") {
            if paths.is_none() {
                return Err(Error::new(
                    "PATCH_INVALID",
                    "hunk has no ---/+++ file headers",
                ));
            }
            let (_, mut old, mut new) = parse_hunk_header(line)?;
            index += 1;
            while old != 0 || new != 0 {
                let row = lines
                    .get(index)
                    .ok_or_else(|| Error::new("PATCH_INVALID", "truncated hunk"))?;
                let (consume_old, consume_new) = match row.as_bytes().first() {
                    Some(b' ') => (1, 1),
                    Some(b'-') => (1, 0),
                    Some(b'+') => (0, 1),
                    Some(b'\\') if row.starts_with("\\ No newline at end of file") => (0, 0),
                    _ => return Err(Error::new("PATCH_INVALID", "unexpected line in hunk")),
                };
                old = old
                    .checked_sub(consume_old)
                    .ok_or_else(|| Error::new("PATCH_INVALID", "too many old lines in hunk"))?;
                new = new
                    .checked_sub(consume_new)
                    .ok_or_else(|| Error::new("PATCH_INVALID", "too many new lines in hunk"))?;
                index += 1;
            }
            continue;
        } else if [
            "new file mode ",
            "deleted file mode ",
            "rename from ",
            "rename to ",
            "copy from ",
            "copy to ",
            "old mode ",
            "new mode ",
            "Binary files ",
            "GIT binary patch",
        ]
        .iter()
        .any(|prefix| line.starts_with(prefix))
        {
            return Err(Error::new(
                "PATCH_UNSUPPORTED_CHANGE",
                "only modifications to existing text resources are supported",
            ));
        }
        index += 1;
    }
    if let Some(begin) = start {
        sections.push(make_file_patch(&diff[offsets[begin]..], paths)?);
    }
    if sections.is_empty() {
        return Err(Error::new("PATCH_EMPTY", "patch contains no file changes"));
    }
    Ok(sections)
}

fn make_file_patch(diff: &str, paths: Option<(String, String)>) -> Result<FilePatch> {
    let (mut old, mut new) =
        paths.ok_or_else(|| Error::new("PATCH_INVALID", "missing ---/+++ file headers"))?;
    if old == "/dev/null" || new == "/dev/null" {
        return Err(Error::new(
            "PATCH_UNSUPPORTED_CHANGE",
            "file additions and deletions are unsupported",
        ));
    }
    if old.starts_with("a/") && new.starts_with("b/") {
        old = old[2..].to_owned();
        new = new[2..].to_owned();
    }
    if old != new {
        return Err(Error::new(
            "PATCH_UNSUPPORTED_CHANGE",
            "file renames are unsupported",
        ));
    }
    let path = safe_relative_path(&new).map_err(|e| Error::new("PATCH_INVALID_TARGET", e))?;
    if new.to_ascii_lowercase().ends_with(".dvpl") {
        return Err(Error::new(
            "PATCH_DVPL_INPUT",
            "patch paths must refer to unpacked resources, without .dvpl",
        ));
    }
    if parse_hunks(diff)?.is_empty() {
        return Err(Error::new("PATCH_EMPTY", "file section contains no hunks"));
    }
    Ok(FilePatch {
        path: path.to_string_lossy().replace('\\', "/"),
        diff: diff.to_owned(),
    })
}

fn decode_header_path(header: &str) -> Result<String> {
    if !header.starts_with('"') {
        return Ok(header.split('\t').next().unwrap().to_owned());
    }
    // Git quotes paths containing special characters using C escapes and octal bytes.
    let mut bytes = Vec::new();
    let raw = header.as_bytes();
    let mut index = 1;
    while index < raw.len() {
        match raw[index] {
            b'"' => {
                return String::from_utf8(bytes)
                    .map_err(|e| Error::new("PATCH_INVALID_TARGET", e.to_string()));
            }
            b'\\' => {
                index += 1;
                let escaped = *raw
                    .get(index)
                    .ok_or_else(|| Error::new("PATCH_INVALID_TARGET", "incomplete quoted path"))?;
                let byte = match escaped {
                    b'0'..=b'7' => {
                        let end = index + 3;
                        let digits = raw
                            .get(index..end)
                            .filter(|digits| digits.iter().all(|byte| (b'0'..=b'7').contains(byte)))
                            .ok_or_else(|| {
                                Error::new("PATCH_INVALID_TARGET", "invalid octal path escape")
                            })?;
                        let value = digits
                            .iter()
                            .fold(0u16, |value, byte| value * 8 + u16::from(byte - b'0'));
                        index += 2;
                        u8::try_from(value).map_err(|_| {
                            Error::new("PATCH_INVALID_TARGET", "invalid octal path escape")
                        })?
                    }
                    b't' => b'\t',
                    b'n' => b'\n',
                    b'r' => b'\r',
                    b'a' => 7,
                    b'b' => 8,
                    b'f' => 12,
                    b'v' => 11,
                    b'\\' | b'"' => escaped,
                    _ => {
                        return Err(Error::new(
                            "PATCH_INVALID_TARGET",
                            "unknown quoted path escape",
                        ));
                    }
                };
                bytes.push(byte);
            }
            byte => bytes.push(byte),
        }
        index += 1;
    }
    Err(Error::new(
        "PATCH_INVALID_TARGET",
        "unterminated quoted path",
    ))
}

/// Apply a single-file unified diff. Context is exact; every hunk must apply.
pub(super) fn apply_unified_diff(source: &str, diff: &str) -> Result<String> {
    if diff
        .lines()
        .filter(|line| line.starts_with("diff --git "))
        .count()
        > 1
    {
        return Err(Error::new(
            "PATCH_MULTIPLE_FILES",
            "each patch operation must reference a diff for exactly one resource",
        ));
    }
    let hunks = parse_hunks(diff)?;
    if hunks.is_empty() {
        return Err(Error::new("PATCH_EMPTY", "unified diff contains no hunks"));
    }

    let newline = if source.contains("\r\n") {
        "\r\n"
    } else {
        "\n"
    };
    let mut lines = split_source_lines(source);
    let mut cursor = 0usize;
    let mut offset: i64 = 0;

    for hunk in hunks {
        let old_lines = hunk
            .lines
            .iter()
            .filter(|line| line.kind != Kind::Add)
            .collect::<Vec<_>>();
        let mut expected = if hunk.old_len == 0 {
            hunk.old_start
        } else {
            hunk.old_start.saturating_sub(1)
        } as i64
            + offset;
        if expected < cursor as i64 {
            expected = cursor as i64;
        }
        let expected = expected.max(0) as usize;

        let start = if old_lines.is_empty() {
            if expected > lines.len() {
                return Err(Error::new(
                    "PATCH_CONTEXT_MISMATCH",
                    format!(
                        "insertion hunk points past end of file at line {}",
                        expected + 1
                    ),
                ));
            }
            expected
        } else {
            let matches = find_matches(&lines, &old_lines, cursor);
            if matches.contains(&expected) {
                expected
            } else if matches.len() == 1 {
                matches[0]
            } else if matches.is_empty() {
                return Err(Error::new(
                    "PATCH_CONTEXT_MISMATCH",
                    format!(
                        "could not match diff hunk near source line {}",
                        expected + 1
                    ),
                ));
            } else {
                return Err(Error::new(
                    "PATCH_CONTEXT_AMBIGUOUS",
                    format!(
                        "diff hunk near line {} matches multiple locations",
                        expected + 1
                    ),
                ));
            }
        };

        let original_start = if hunk.old_len == 0 {
            hunk.old_start as i64
        } else {
            hunk.old_start.saturating_sub(1) as i64
        };
        offset = start as i64 - original_start + hunk.new_len as i64 - hunk.old_len as i64;

        let mut consumed = 0usize;
        let mut replacement = Vec::new();
        let mut source_index = start;
        for line in &hunk.lines {
            match line.kind {
                Kind::Context => {
                    let existing = lines.get(source_index).ok_or_else(|| {
                        Error::new(
                            "PATCH_CONTEXT_MISMATCH",
                            "diff context runs past end of file",
                        )
                    })?;
                    if line_content(existing) != line.text {
                        return Err(Error::new(
                            "PATCH_CONTEXT_MISMATCH",
                            format!("context differs near source line {}", source_index + 1),
                        ));
                    }
                    replacement.push(existing.clone());
                    source_index += 1;
                    consumed += 1;
                }
                Kind::Remove => {
                    let existing = lines.get(source_index).ok_or_else(|| {
                        Error::new(
                            "PATCH_CONTEXT_MISMATCH",
                            "removed line runs past end of file",
                        )
                    })?;
                    if line_content(existing) != line.text {
                        return Err(Error::new(
                            "PATCH_CONTEXT_MISMATCH",
                            format!("removed line differs near source line {}", source_index + 1),
                        ));
                    }
                    source_index += 1;
                    consumed += 1;
                }
                Kind::Add => {
                    let mut added = line.text.clone();
                    if !line.no_newline {
                        added.push_str(newline);
                    }
                    replacement.push(added);
                }
            }
        }
        if consumed != hunk.old_len {
            return Err(Error::new(
                "PATCH_INVALID",
                format!(
                    "hunk declares {} old lines but contains {consumed}",
                    hunk.old_len
                ),
            ));
        }
        if replacement.len() != hunk.new_len {
            return Err(Error::new(
                "PATCH_INVALID",
                format!(
                    "hunk declares {} new lines but contains {}",
                    hunk.new_len,
                    replacement.len()
                ),
            ));
        }

        lines.splice(start..start + consumed, replacement);
        cursor = start + hunk.new_len;
    }

    Ok(lines.concat())
}

fn parse_hunks(diff: &str) -> Result<Vec<Hunk>> {
    let mut hunks = Vec::new();
    let lines = diff.split_inclusive('\n').map(strip_lf).collect::<Vec<_>>();
    let mut index = 0usize;
    while index < lines.len() {
        let line = lines[index];
        if !line.starts_with("@@ ") {
            index += 1;
            continue;
        }
        let (old_start, old_len, new_len) = parse_hunk_header(line)?;
        index += 1;
        let mut body: Vec<HunkLine> = Vec::new();
        while index < lines.len() {
            let row = lines[index];
            if row.starts_with("@@ ") || row.starts_with("diff --git ") {
                break;
            }
            if row.starts_with("\\ No newline at end of file") {
                if let Some(previous) = body.last_mut() {
                    previous.no_newline = true;
                }
                index += 1;
                continue;
            }
            if row.is_empty() {
                return Err(Error::new(
                    "PATCH_INVALID",
                    "empty line in unified diff hunk",
                ));
            }
            let (kind, text) = match row.as_bytes()[0] {
                b' ' => (Kind::Context, &row[1..]),
                b'-' => (Kind::Remove, &row[1..]),
                b'+' => (Kind::Add, &row[1..]),
                _ => break,
            };
            body.push(HunkLine {
                kind,
                text: text.to_owned(),
                no_newline: false,
            });
            index += 1;
        }
        let actual_old = body.iter().filter(|row| row.kind != Kind::Add).count();
        let actual_new = body.iter().filter(|row| row.kind != Kind::Remove).count();
        if actual_old != old_len || actual_new != new_len {
            return Err(Error::new(
                "PATCH_INVALID",
                format!(
                    "hunk header declares -{old_len}/+{new_len} lines but body has -{actual_old}/+{actual_new}"
                ),
            ));
        }
        hunks.push(Hunk {
            old_start,
            old_len,
            new_len,
            lines: body,
        });
    }
    Ok(hunks)
}

fn parse_hunk_header(line: &str) -> Result<(usize, usize, usize)> {
    let end = line[3..]
        .find(" @@")
        .map(|index| index + 3)
        .ok_or_else(|| Error::new("PATCH_INVALID", "malformed unified diff hunk header"))?;
    let mut specs = line[3..end].split_whitespace();
    let old = specs
        .next()
        .and_then(|v| v.strip_prefix('-'))
        .ok_or_else(|| Error::new("PATCH_INVALID", "hunk header has no old range"))?;
    let new = specs
        .next()
        .and_then(|v| v.strip_prefix('+'))
        .ok_or_else(|| Error::new("PATCH_INVALID", "hunk header has no new range"))?;
    let (old_start, old_len) = parse_range(old)?;
    let (_, new_len) = parse_range(new)?;
    Ok((old_start, old_len, new_len))
}

fn parse_range(value: &str) -> Result<(usize, usize)> {
    let mut parts = value.split(',');
    let start = parts
        .next()
        .and_then(|v| v.parse().ok())
        .ok_or_else(|| Error::new("PATCH_INVALID", "invalid hunk line number"))?;
    let len = match parts.next() {
        Some(value) => value
            .parse()
            .map_err(|_| Error::new("PATCH_INVALID", "invalid hunk line count"))?,
        None => 1,
    };
    if parts.next().is_some() {
        return Err(Error::new("PATCH_INVALID", "invalid hunk range"));
    }
    Ok((start, len))
}

fn find_matches(lines: &[String], old_lines: &[&HunkLine], from: usize) -> Vec<usize> {
    if old_lines.is_empty() || old_lines.len() > lines.len() {
        return Vec::new();
    }
    (from..=lines.len() - old_lines.len())
        .filter(|start| {
            old_lines
                .iter()
                .enumerate()
                .all(|(offset, expected)| line_content(&lines[*start + offset]) == expected.text)
        })
        .collect()
}

fn split_source_lines(source: &str) -> Vec<String> {
    source.split_inclusive('\n').map(str::to_owned).collect()
}

fn line_content(line: &str) -> &str {
    line.strip_suffix('\n').unwrap_or(line)
}

fn strip_lf(line: &str) -> &str {
    line.strip_suffix('\n').unwrap_or(line)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn splits_unified_headers_without_confusing_removed_content() {
        let diff = "--- a/one.yaml\n+++ b/one.yaml\n@@ -1 +1 @@\n--- old\n+++ new\n--- a/two.yaml\n+++ b/two.yaml\n@@ -1 +1 @@\n-old\n+new\n";
        let sections = split_patch(diff).unwrap();
        assert_eq!(sections.len(), 2);
        assert_eq!(sections[0].path, "one.yaml");
        assert_eq!(
            apply_unified_diff("-- old\n", &sections[0].diff).unwrap(),
            "++ new\n"
        );
        assert_eq!(sections[1].path, "two.yaml");
    }

    #[test]
    fn decodes_git_quoted_paths_and_preserves_missing_final_newline() {
        let diff = r#"diff --git "a/space \303\251.yaml" "b/space \303\251.yaml"
--- "a/space \303\251.yaml"
+++ "b/space \303\251.yaml"
@@ -1 +1 @@
-old
\ No newline at end of file
+new
\ No newline at end of file
"#;
        let sections = split_patch(diff).unwrap();
        assert_eq!(sections[0].path, "space é.yaml");
        assert_eq!(apply_unified_diff("old", &sections[0].diff).unwrap(), "new");
    }

    #[test]
    fn later_hunks_follow_insertions_when_context_is_repeated() {
        let diff =
            "--- a/file\n+++ b/file\n@@ -1 +1,2 @@\n a\n+inserted\n@@ -3 +4 @@\n-old\n+new\n";
        let section = split_patch(diff).unwrap().remove(0);
        assert_eq!(
            apply_unified_diff("a\nold\nold\n", &section.diff).unwrap(),
            "a\ninserted\nold\nnew\n"
        );
    }
}

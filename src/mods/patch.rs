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

/// Apply a single-file unified diff. Context is exact; every hunk must apply.
pub(super) fn apply_unified_diff(source: &str, diff: &str) -> Result<String> {
    if diff.matches("diff --git ").count() > 1 {
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
        offset = start as i64 - original_start;

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
        let added_count = hunk
            .lines
            .iter()
            .filter(|line| line.kind == Kind::Add)
            .count();
        if added_count != hunk.new_len {
            return Err(Error::new(
                "PATCH_INVALID",
                format!(
                    "hunk declares {} new lines but contains {added_count}",
                    hunk.new_len
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
    let len = parts.next().map(|v| v.parse().ok()).flatten().unwrap_or(1);
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

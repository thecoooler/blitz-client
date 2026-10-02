use std::ops::Range;

/// Match normalized text, but return ranges in the untouched resource.
/// Only spaces/tabs at line ends and CRLF line endings are normalized.
pub(super) fn find_matches(source: &str, find: &str, ignore_trailing: bool) -> Vec<Range<usize>> {
    if !ignore_trailing {
        return source
            .match_indices(find)
            .map(|(start, text)| start..start + text.len())
            .collect();
    }
    let (source_text, spans) = normalize(source);
    let (find_text, _) = normalize(find);
    if find_text.is_empty() {
        return Vec::new();
    }
    // Block searches must retain their original indentation. A trimmed final
    // space is only optional at a line end, never inside another token.
    let line_start = find_text.contains('\n') || find_text.starts_with([' ', '\t']);
    let line_end = find.ends_with([' ', '\t']);
    source_text
        .match_indices(&find_text)
        .filter(|(start, text)| {
            (!line_start || *start == 0 || source_text.as_bytes()[start - 1] == b'\n')
                && (!line_end
                    || start + text.len() == source_text.len()
                    || source_text.as_bytes()[start + text.len()] == b'\n')
        })
        .map(|(start, text)| spans[start].start..spans[start + text.len() - 1].end)
        .collect()
}

fn normalize(text: &str) -> (String, Vec<Range<usize>>) {
    let mut normalized = String::with_capacity(text.len());
    let mut spans = Vec::with_capacity(text.len());
    let mut offset = 0;
    for line in text.split_inclusive('\n') {
        let has_newline = line.ends_with('\n');
        let content = if has_newline {
            line.strip_suffix("\r\n")
                .unwrap_or_else(|| line.strip_suffix('\n').unwrap())
        } else {
            line
        };
        let trimmed = content.trim_end_matches([' ', '\t']);
        normalized.push_str(trimmed);
        for index in 0..trimmed.len() {
            let end = if index + 1 == trimmed.len() {
                content.len()
            } else {
                index + 1
            };
            spans.push(offset + index..offset + end);
        }
        if has_newline {
            normalized.push('\n');
            spans.push(offset + trimmed.len()..offset + line.len());
        }
        offset += line.len();
    }
    (normalized, spans)
}

#[cfg(test)]
mod tests {
    use super::find_matches;

    #[test]
    fn normalized_blocks_keep_leading_and_internal_whitespace() {
        let find = "    key: value\n";
        assert_eq!(find_matches("    key: value \t\r\n", find, true).len(), 1);
        for source in [
            "        key: value\n",
            "    key:  value\n",
            "    Key: value\n",
        ] {
            assert!(find_matches(source, find, true).is_empty());
        }
        assert!(find_matches("key: value", "key: ", true).is_empty());
    }

    #[test]
    fn normalized_ranges_preserve_unicode_and_trailing_bytes_at_eof() {
        let source = "сосед:  \r\n    ключ: значение \t";
        let ranges = find_matches(source, "    ключ: значение", true);
        assert_eq!(ranges.len(), 1);
        assert_eq!(&source[ranges[0].clone()], "    ключ: значение \t");
        assert!(find_matches(source, "    ключ: значение\n", true).is_empty());
    }
}

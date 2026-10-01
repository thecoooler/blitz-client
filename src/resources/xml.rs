#[derive(Debug)]
pub struct XmlNode {
    pub name: String,
    pub text: String,
    pub children: Vec<XmlNode>,
}

pub fn child_text(node: &XmlNode, name: &str) -> Option<String> {
    node.children
        .iter()
        .find(|child| child.name == name)
        .map(|child| child.text.trim().to_owned())
        .filter(|text| !text.is_empty())
}

pub fn parse_xml(data: &[u8]) -> std::result::Result<XmlNode, String> {
    let decoded = String::from_utf8_lossy(data);
    let source = decoded.trim_start_matches('\u{feff}');
    let mut roots = Vec::<XmlNode>::new();
    let mut stack = Vec::<XmlNode>::new();
    let mut cursor = 0;

    while cursor < source.len() {
        let Some(relative_open) = source[cursor..].find('<') else {
            append_xml_text(&mut stack, &source[cursor..])?;
            break;
        };
        let open = cursor + relative_open;
        if open > cursor {
            append_xml_text(&mut stack, &source[cursor..open])?;
        }

        if source[open..].starts_with("<!--") {
            let end = source[open + 4..]
                .find("-->")
                .ok_or_else(|| "unterminated XML comment".to_owned())?;
            cursor = open + 4 + end + 3;
            continue;
        }
        if source[open..].starts_with("<![CDATA[") {
            let end = source[open + 9..]
                .find("]]>")
                .ok_or_else(|| "unterminated XML CDATA section".to_owned())?;
            let content_start = open + 9;
            let content_end = content_start + end;
            if let Some(node) = stack.last_mut() {
                node.text.push_str(&source[content_start..content_end]);
            }
            cursor = content_end + 3;
            continue;
        }
        if source[open..].starts_with("<?") || source[open..].starts_with("<!") {
            cursor = find_xml_tag_end(&source, open)? + 1;
            continue;
        }

        let end = find_xml_tag_end(&source, open)?;
        let mut body = source[open + 1..end].trim();
        if let Some(close) = body.strip_prefix('/') {
            let close_name = close.trim();
            let node = stack
                .pop()
                .ok_or_else(|| format!("unexpected closing XML tag </{close_name}>"))?;
            if node.name != close_name {
                return Err(format!(
                    "mismatched XML closing tag </{close_name}> for <{}>",
                    node.name
                ));
            }
            append_xml_node(&mut roots, &mut stack, node);
        } else {
            let self_closing = body.ends_with('/');
            if self_closing {
                body = body[..body.len() - 1].trim_end();
            }
            let name_end = body.find(char::is_whitespace).unwrap_or(body.len());
            if name_end == 0 {
                return Err("empty XML tag name".to_owned());
            }
            let node = XmlNode {
                name: body[..name_end].to_owned(),
                text: String::new(),
                children: Vec::new(),
            };
            if self_closing {
                append_xml_node(&mut roots, &mut stack, node);
            } else {
                stack.push(node);
            }
        }
        cursor = end + 1;
    }

    if !stack.is_empty() {
        return Err(format!("unclosed XML tag <{}>", stack.last().unwrap().name));
    }
    if roots.len() != 1 {
        return Err(format!(
            "expected one XML root element, found {}",
            roots.len()
        ));
    }
    Ok(roots.remove(0))
}

fn append_xml_node(roots: &mut Vec<XmlNode>, stack: &mut [XmlNode], node: XmlNode) {
    if let Some(parent) = stack.last_mut() {
        parent.children.push(node);
    } else {
        roots.push(node);
    }
}

fn append_xml_text(stack: &mut [XmlNode], text: &str) -> std::result::Result<(), String> {
    let Some(node) = stack.last_mut() else {
        if text.trim().is_empty() {
            return Ok(());
        }
        return Err("text outside the XML root element".to_owned());
    };
    node.text.push_str(&decode_xml_entities(text)?);
    Ok(())
}

fn find_xml_tag_end(source: &str, open: usize) -> std::result::Result<usize, String> {
    let mut quote = None;
    for (offset, ch) in source[open + 1..].char_indices() {
        match (quote, ch) {
            (Some(current), value) if current == value => quote = None,
            (None, '\'' | '"') => quote = Some(ch),
            (None, '>') => return Ok(open + 1 + offset),
            _ => {}
        }
    }
    Err("unterminated XML tag".to_owned())
}

fn decode_xml_entities(text: &str) -> std::result::Result<String, String> {
    let mut result = String::with_capacity(text.len());
    let mut rest = text;
    while let Some(start) = rest.find('&') {
        result.push_str(&rest[..start]);
        let entity_end = rest[start..]
            .find(';')
            .ok_or_else(|| "unterminated XML entity".to_owned())?
            + start;
        let entity = &rest[start + 1..entity_end];
        let decoded = match entity {
            "amp" => '&',
            "lt" => '<',
            "gt" => '>',
            "quot" => '"',
            "apos" => '\'',
            value if value.starts_with("#x") || value.starts_with("#X") => {
                let scalar = u32::from_str_radix(&value[2..], 16)
                    .map_err(|_| format!("invalid XML entity &{value};"))?;
                char::from_u32(scalar).ok_or_else(|| format!("invalid XML scalar in &{value};"))?
            }
            value if value.starts_with('#') => {
                let scalar = value[1..]
                    .parse::<u32>()
                    .map_err(|_| format!("invalid XML entity &{value};"))?;
                char::from_u32(scalar).ok_or_else(|| format!("invalid XML scalar in &{value};"))?
            }
            value => return Err(format!("unsupported XML entity &{value};")),
        };
        result.push(decoded);
        rest = &rest[entity_end + 1..];
    }
    result.push_str(rest);
    Ok(result)
}

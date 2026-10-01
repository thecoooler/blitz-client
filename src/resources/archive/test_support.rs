// Encode real keyed-archive bytes; assertions cross the archive/export
// interfaces rather than constructing their decoded implementation state.
#[derive(Clone)]
pub(crate) enum Value {
    Number(u64),
    Signed(i64),
    Bytes(Vec<u8>),
    Text(&'static str),
    #[cfg(feature = "export")]
    Map(Fields),
    #[cfg(feature = "export")]
    Array(Vec<Value>),
}
pub(crate) type Fields = Vec<(&'static str, Value)>;

pub(crate) fn archive(fields: &Fields) -> Vec<u8> {
    let mut out = b"KA".to_vec();
    out.extend(1u16.to_le_bytes());
    out.extend((fields.len() as u32).to_le_bytes());
    for (key, value) in fields {
        encode(&Value::Text(key), &mut out);
        encode(value, &mut out);
    }
    out
}

pub(crate) fn encode(value: &Value, out: &mut Vec<u8>) {
    match value {
        Value::Number(value) => {
            out.push(10);
            out.extend(value.to_le_bytes());
        }
        Value::Signed(value) => {
            out.push(9);
            out.extend(value.to_le_bytes());
        }
        Value::Bytes(value) => {
            out.push(6);
            out.extend((value.len() as u32).to_le_bytes());
            out.extend(value);
        }
        Value::Text(value) => {
            out.push(4);
            out.extend((value.len() as u32).to_le_bytes());
            out.extend(value.as_bytes());
        }
        #[cfg(feature = "export")]
        Value::Map(fields) => {
            let nested = archive(fields);
            out.push(8);
            out.extend((nested.len() as u32).to_le_bytes());
            out.extend(nested);
        }
        #[cfg(feature = "export")]
        Value::Array(values) => {
            out.push(27);
            out.extend((values.len() as u32).to_le_bytes());
            for value in values {
                encode(value, out);
            }
        }
    }
}

pub(crate) fn scg(groups: &[Fields]) -> Vec<u8> {
    let mut out = b"SCPG".to_vec();
    out.extend(1u32.to_le_bytes());
    out.extend((groups.len() as u32).to_le_bytes());
    out.extend((groups.len() as u32).to_le_bytes());
    for group in groups {
        out.extend(archive(group));
    }
    out
}

pub(crate) fn floats(values: &[f32]) -> Vec<u8> {
    values
        .iter()
        .flat_map(|value| value.to_le_bytes())
        .collect()
}

pub(crate) fn group(
    id: u64,
    format: u64,
    vertex_count: u64,
    vertices: Vec<u8>,
    wide: bool,
    indices: &[u32],
) -> Fields {
    let indices_bytes = indices
        .iter()
        .flat_map(|index| {
            if wide {
                index.to_le_bytes().to_vec()
            } else {
                (*index as u16).to_le_bytes().to_vec()
            }
        })
        .collect();
    vec![
        ("#id", Value::Number(id)),
        ("vertexFormat", Value::Number(format)),
        ("vertexCount", Value::Number(vertex_count)),
        ("vertices", Value::Bytes(vertices)),
        ("rhi_primitiveType", Value::Number(1)),
        ("indexFormat", Value::Number(u64::from(wide))),
        ("indexCount", Value::Number(indices.len() as u64)),
        ("indices", Value::Bytes(indices_bytes)),
    ]
}

pub(crate) fn triangle(id: u64, wide: bool) -> Fields {
    group(
        id,
        1,
        3,
        floats(&[0., 0., 0., 1., 0., 0., 0., 1., 0.]),
        wide,
        &[0, 1, 2],
    )
}

pub(crate) fn replace(fields: &mut Fields, key: &str, value: Value) {
    fields.iter_mut().find(|(name, _)| *name == key).unwrap().1 = value;
}

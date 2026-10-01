use super::KaValue;
use std::collections::BTreeMap;

const ATTRIBUTE_BYTES: [usize; 20] = [
    12, 12, 4, 8, 8, 8, 8, 12, 12, 4, 16, 12, 4, 8, 16, 16, 12, 12, 12, 12,
];
const KNOWN_VERTEX_FORMAT: u64 = (1 << ATTRIBUTE_BYTES.len()) - 1;

/// Owns group identity; structural decoding remains lazy so unused geometry
/// does not impose new requirements on showroom or collision exports.
#[derive(Debug)]
pub struct ScgGeometry {
    groups: BTreeMap<u64, KaValue>,
}

/// Binary layout and index invariants have already been checked. Optional
/// attributes remain optional and floats retain NaNs for caller-owned repairs.
#[derive(Debug)]
pub struct DecodedScgGroup {
    pub positions: Vec<[f32; 3]>,
    pub normals: Option<Vec<[f32; 3]>>,
    pub colors: Option<Vec<[f32; 4]>>,
    pub uvs: Option<Vec<[f32; 2]>>,
    pub tangents: Option<Vec<[f32; 3]>>,
    pub binormals: Option<Vec<[f32; 3]>>,
    pub armor_zones: Option<Vec<f32>>,
    pub indices: Vec<u32>,
}

impl ScgGeometry {
    pub(super) fn from_groups(groups: Vec<KaValue>) -> Result<Self, String> {
        let mut indexed = BTreeMap::new();
        for group in groups {
            let id = group_id(&group)?;
            if indexed.insert(id, group).is_some() {
                return Err(format!("duplicate SCG polygon group id {id}"));
            }
        }
        Ok(Self { groups: indexed })
    }

    pub fn contains_group(&self, id: u64) -> bool {
        self.groups.contains_key(&id)
    }

    pub fn decode(&self, id: u64) -> Result<DecodedScgGroup, String> {
        let group = self
            .groups
            .get(&id)
            .ok_or_else(|| format!("missing SCG geometry group {id}"))?;
        decode_group(group).map_err(|error| format!("SCG group {id}: {error}"))
    }
}

fn group_id(group: &KaValue) -> Result<u64, String> {
    match group.get("#id") {
        Some(KaValue::Bytes(bytes)) if bytes.len() == 8 => {
            Ok(u64::from_le_bytes(bytes.as_slice().try_into().unwrap()))
        }
        Some(value) => value
            .as_u64()
            .ok_or_else(|| format!("unexpected SCG polygon group id: {value:?}")),
        None => Err("SCG polygon group has no #id".to_owned()),
    }
}

fn decode_group(group: &KaValue) -> Result<DecodedScgGroup, String> {
    let format = number(group, "vertexFormat")?;
    if format & !KNOWN_VERTEX_FORMAT != 0 {
        return Err(format!("unsupported vertex format bits: 0x{format:x}"));
    }
    if format & 1 == 0 {
        return Err("lacks positions".to_owned());
    }
    let mut offsets = [None; ATTRIBUTE_BYTES.len()];
    let mut stride = 0;
    for (bit, size) in ATTRIBUTE_BYTES.iter().enumerate() {
        if format & (1 << bit) != 0 {
            offsets[bit] = Some(stride);
            stride += size;
        }
    }
    let vertex_count = count(group, "vertexCount")?;
    let vertices = bytes(group, "vertices")?;
    let expected = stride
        .checked_mul(vertex_count)
        .ok_or_else(|| "vertex buffer size overflow".to_owned())?;
    if vertices.len() != expected {
        return Err("vertex buffer size mismatch".to_owned());
    }
    if number(group, "rhi_primitiveType")? != 1 {
        return Err("not a triangle list".to_owned());
    }

    let index_count = count(group, "indexCount")?;
    let indices = bytes(group, "indices")?;
    let index_width = match number(group, "indexFormat")? {
        0 => 2usize,
        1 => 4usize,
        other => return Err(format!("unknown index format {other}")),
    };
    let expected = index_count
        .checked_mul(index_width)
        .ok_or_else(|| "index buffer size overflow".to_owned())?;
    if indices.len() != expected {
        return Err("index buffer size mismatch".to_owned());
    }
    if !index_count.is_multiple_of(3) {
        return Err("triangle index count is not divisible by 3".to_owned());
    }
    let indices = indices
        .chunks_exact(index_width)
        .map(|index| match index_width {
            2 => u16::from_le_bytes(index.try_into().unwrap()) as u32,
            _ => u32::from_le_bytes(index.try_into().unwrap()),
        })
        .collect::<Vec<_>>();
    if indices
        .iter()
        .any(|index| !usize::try_from(*index).is_ok_and(|index| index < vertex_count))
    {
        return Err("index out of range".to_owned());
    }

    let colors = offsets[2].map(|offset| {
        vertices
            .chunks_exact(stride)
            .map(|vertex| std::array::from_fn(|axis| vertex[offset + axis] as f32 / 255.0))
            .collect()
    });
    Ok(DecodedScgGroup {
        positions: floats(vertices, stride, offsets[0]).unwrap(),
        normals: floats(vertices, stride, offsets[1]),
        colors,
        uvs: floats(vertices, stride, offsets[3]),
        tangents: floats(vertices, stride, offsets[7]),
        binormals: floats(vertices, stride, offsets[8]),
        armor_zones: floats::<1>(vertices, stride, offsets[9])
            .map(|zones| zones.into_iter().map(|zone| zone[0]).collect()),
        indices,
    })
}

fn floats<const N: usize>(
    vertices: &[u8],
    stride: usize,
    offset: Option<usize>,
) -> Option<Vec<[f32; N]>> {
    offset.map(|offset| {
        vertices
            .chunks_exact(stride)
            .map(|vertex| {
                std::array::from_fn(|axis| {
                    let start = offset + axis * 4;
                    f32::from_le_bytes(vertex[start..start + 4].try_into().unwrap())
                })
            })
            .collect()
    })
}

fn number(group: &KaValue, field: &str) -> Result<u64, String> {
    group
        .get(field)
        .and_then(KaValue::as_u64)
        .ok_or_else(|| format!("missing or invalid field {field}"))
}

fn count(group: &KaValue, field: &str) -> Result<usize, String> {
    usize::try_from(number(group, field)?).map_err(|_| format!("{field} exceeds addressable size"))
}

fn bytes<'a>(group: &'a KaValue, field: &str) -> Result<&'a [u8], String> {
    group
        .get(field)
        .and_then(KaValue::as_bytes)
        .ok_or_else(|| format!("missing or invalid byte field {field}"))
}

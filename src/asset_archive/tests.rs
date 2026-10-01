use super::parse_scg;
use crate::{tank_armor, tank_model};
use serde_json::{Value as Json, json};

// Encode real keyed-archive bytes; assertions cross the archive/export
// interfaces rather than constructing their decoded implementation state.
#[derive(Clone)]
enum Value {
    Number(u64),
    Signed(i64),
    Bytes(Vec<u8>),
    Text(&'static str),
    Map(Fields),
    Array(Vec<Value>),
}
type Fields = Vec<(&'static str, Value)>;

fn archive(fields: &Fields) -> Vec<u8> {
    let mut out = b"KA".to_vec();
    out.extend(1u16.to_le_bytes());
    out.extend((fields.len() as u32).to_le_bytes());
    for (key, value) in fields {
        encode(&Value::Text(key), &mut out);
        encode(value, &mut out);
    }
    out
}

fn encode(value: &Value, out: &mut Vec<u8>) {
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
        Value::Map(fields) => {
            let nested = archive(fields);
            out.push(8);
            out.extend((nested.len() as u32).to_le_bytes());
            out.extend(nested);
        }
        Value::Array(values) => {
            out.push(27);
            out.extend((values.len() as u32).to_le_bytes());
            for value in values {
                encode(value, out);
            }
        }
    }
}

fn scg(groups: &[Fields]) -> Vec<u8> {
    let mut out = b"SCPG".to_vec();
    out.extend(1u32.to_le_bytes());
    out.extend((groups.len() as u32).to_le_bytes());
    out.extend((groups.len() as u32).to_le_bytes());
    for group in groups {
        out.extend(archive(group));
    }
    out
}

fn floats(values: &[f32]) -> Vec<u8> {
    values
        .iter()
        .flat_map(|value| value.to_le_bytes())
        .collect()
}

fn group(
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

fn triangle(id: u64, wide: bool) -> Fields {
    group(
        id,
        1,
        3,
        floats(&[0., 0., 0., 1., 0., 0., 0., 1., 0.]),
        wide,
        &[0, 1, 2],
    )
}

fn replace(fields: &mut Fields, key: &str, value: Value) {
    fields.iter_mut().find(|(name, _)| *name == key).unwrap().1 = value;
}

#[test]
fn decodes_both_index_widths_and_optional_streams() {
    for wide in [false, true] {
        let geometry = parse_scg(&scg(&[triangle(7, wide)])).unwrap();
        assert!(geometry.contains_group(7));
        assert!(!geometry.contains_group(8));
        let decoded = geometry.decode(7).unwrap();
        assert_eq!(
            decoded.positions,
            [[0., 0., 0.], [1., 0., 0.], [0., 1., 0.]]
        );
        assert_eq!(decoded.indices, [0, 1, 2]);
        assert!(decoded.normals.is_none());
        assert!(decoded.colors.is_none());
        assert!(decoded.uvs.is_none());
        assert!(decoded.tangents.is_none());
        assert!(decoded.binormals.is_none());
        assert!(decoded.armor_zones.is_none());
        assert!(
            geometry
                .decode(8)
                .unwrap_err()
                .contains("missing SCG geometry group 8")
        );
    }
}

#[test]
fn decodes_packed_attributes_without_applying_repair_policy() {
    let format = 1 | (1 << 1) | (1 << 2) | (1 << 3) | (1 << 4) | (1 << 7) | (1 << 8) | (1 << 9);
    let mut vertices = floats(&[1., 2., 3., f32::NAN, 0., 1.]);
    vertices.extend([0, 64, 128, 255]);
    vertices.extend(floats(&[0.25, 0.75, 7., 8., 1., 0., 0., 0., 1., 0., 3.]));
    let geometry = parse_scg(&scg(&[group(9, format, 1, vertices, false, &[0, 0, 0])])).unwrap();
    let decoded = geometry.decode(9).unwrap();
    assert_eq!(decoded.positions, [[1., 2., 3.]]);
    assert!(decoded.normals.unwrap()[0][0].is_nan());
    assert_eq!(decoded.colors.unwrap(), [[0., 64. / 255., 128. / 255., 1.]]);
    assert_eq!(decoded.uvs.unwrap(), [[0.25, 0.75]]);
    assert_eq!(decoded.tangents.unwrap(), [[1., 0., 0.]]);
    assert_eq!(decoded.binormals.unwrap(), [[0., 1., 0.]]);
    assert_eq!(decoded.armor_zones.unwrap(), [3.]);
}

#[test]
fn validates_group_identity_globally_but_geometry_only_when_selected() {
    let bad = vec![("#id", Value::Number(99))];
    let geometry = parse_scg(&scg(&[triangle(1, false), bad])).unwrap();
    assert_eq!(geometry.decode(1).unwrap().indices, [0, 1, 2]);
    assert!(geometry.decode(99).unwrap_err().contains("vertexFormat"));
    assert!(
        parse_scg(&scg(&[triangle(1, false), triangle(1, true)]))
            .unwrap_err()
            .contains("duplicate")
    );
    let mut no_id = triangle(1, false);
    no_id.retain(|(key, _)| *key != "#id");
    assert!(parse_scg(&scg(&[no_id])).unwrap_err().contains("no #id"));
    for id in [Value::Bytes(7u64.to_le_bytes().to_vec()), Value::Signed(7)] {
        let mut fields = triangle(7, false);
        replace(&mut fields, "#id", id);
        assert!(parse_scg(&scg(&[fields])).unwrap().contains_group(7));
    }
    let mut invalid_id = triangle(1, false);
    replace(&mut invalid_id, "#id", Value::Signed(-1));
    assert!(parse_scg(&scg(&[invalid_id])).is_err());
}

#[test]
fn rejects_invalid_selected_layouts_buffers_and_indices_without_panics() {
    for (key, value, expected) in [
        ("vertexFormat", Value::Number(0), "lacks positions"),
        (
            "vertexFormat",
            Value::Number(1 | (1 << 20)),
            "unsupported vertex format",
        ),
        (
            "vertexFormat",
            Value::Number(1 | (1 << 40)),
            "unsupported vertex format",
        ),
        ("vertexCount", Value::Number(u64::MAX), "size"),
        (
            "vertices",
            Value::Bytes(vec![0; 35]),
            "vertex buffer size mismatch",
        ),
        ("rhi_primitiveType", Value::Number(2), "not a triangle list"),
        ("indexFormat", Value::Number(2), "unknown index format"),
        ("indexCount", Value::Number(u64::MAX), "size"),
        (
            "indices",
            Value::Bytes(vec![0; 5]),
            "index buffer size mismatch",
        ),
        (
            "indices",
            Value::Bytes(vec![0, 0, 1, 0, 3, 0]),
            "index out of range",
        ),
    ] {
        let mut fields = triangle(42, false);
        replace(&mut fields, key, value);
        let error = parse_scg(&scg(&[fields])).unwrap().decode(42).unwrap_err();
        assert!(error.contains("group 42"), "{error}");
        assert!(error.contains(expected), "{key}: {error}");
    }
    let fields = group(42, 1, 3, vec![0; 36], true, &[0, 1, 2, 0]);
    assert!(
        parse_scg(&scg(&[fields]))
            .unwrap()
            .decode(42)
            .unwrap_err()
            .contains("divisible by 3")
    );
}

#[test]
fn preserves_empty_groups_and_rejects_invalid_archive_framing() {
    let geometry = parse_scg(&scg(&[group(1, 1, 0, vec![], false, &[])])).unwrap();
    assert!(geometry.decode(1).unwrap().positions.is_empty());
    assert!(geometry.decode(1).unwrap().indices.is_empty());
    assert!(!parse_scg(&scg(&[])).unwrap().contains_group(1));
    let valid = scg(&[triangle(1, false)]);
    for length in [0, 4, 15, valid.len() - 1] {
        assert!(parse_scg(&valid[..length]).is_err());
    }
    let mut wrong_version = valid.clone();
    wrong_version[4..8].copy_from_slice(&2u32.to_le_bytes());
    assert!(parse_scg(&wrong_version).is_err());
    let mut wrong_count = valid.clone();
    wrong_count[12..16].copy_from_slice(&2u32.to_le_bytes());
    assert!(parse_scg(&wrong_count).is_err());
    let mut trailing = valid;
    trailing.push(0);
    assert!(parse_scg(&trailing).is_err());
}

fn node(name: &'static str, geometry_id: u64) -> Value {
    Value::Map(vec![
        ("name", Value::Text(name)),
        (
            "components",
            Value::Map(vec![
                (
                    "transform",
                    Value::Map(vec![("comp.typename", Value::Text("TransformComponent"))]),
                ),
                (
                    "render",
                    Value::Map(vec![
                        ("comp.typename", Value::Text("RenderComponent")),
                        (
                            "rc.renderObj",
                            Value::Map(vec![(
                                "ro.batches",
                                Value::Map(vec![(
                                    "batch0",
                                    Value::Map(vec![
                                        ("rb.datasource", Value::Number(geometry_id)),
                                        ("rb.nmatname", Value::Number(100)),
                                    ]),
                                )]),
                            )]),
                        ),
                    ]),
                ),
            ]),
        ),
    ])
}

fn scene(nodes: Vec<Value>) -> Vec<u8> {
    let mut out = b"SFV2".to_vec();
    out.extend(1u32.to_le_bytes());
    out.extend((nodes.len() as u32).to_le_bytes());
    out.extend(archive(&vec![]));
    out.extend(0u32.to_le_bytes());
    out.extend(archive(&vec![
        (
            "#dataNodes",
            Value::Array(vec![Value::Map(vec![("#id", Value::Number(100))])]),
        ),
        ("#hierarchy", Value::Array(nodes)),
    ]));
    out
}

fn model(scg: &[u8], include_pbr: bool) -> Result<tank_model::ExportedModel, String> {
    tank_model::export_model(
        &scene(vec![node("hull", 1)]),
        scg,
        "fixture.sc2",
        "fixture.scg",
        "Fixture",
        include_pbr,
        "png",
        90,
        false,
    )
}

fn binary_floats(binary: &[u8], arrays: &Json, name: &str) -> Vec<f32> {
    let offset = arrays[name]["byteOffset"].as_u64().unwrap() as usize;
    let count = arrays[name]["count"].as_u64().unwrap() as usize;
    binary[offset..offset + count * 4]
        .chunks_exact(4)
        .map(|bytes| f32::from_le_bytes(bytes.try_into().unwrap()))
        .collect()
}

#[test]
fn showroom_export_preserves_missing_attribute_repairs_and_unused_geometry() {
    let bytes = scg(&[triangle(1, false), vec![("#id", Value::Number(99))]]);
    let exported = model(&bytes, true).unwrap();
    assert_eq!(exported.vertex_count, 3);
    assert_eq!(exported.triangle_count, 1);
    assert_eq!(exported.generated_uv_groups, 1);
    assert_eq!(exported.metadata["groups"][0]["geometryId"], 1);
    assert_eq!(exported.metadata["groups"][0]["generatedUVs"], true);
    let arrays = &exported.metadata["binaryGeometry"]["arrays"];
    assert_eq!(
        binary_floats(&exported.binary, arrays, "positions"),
        [0., 0., 0., 1., 0., 0., 0., 1., 0.]
    );
    assert_eq!(
        binary_floats(&exported.binary, arrays, "normals"),
        [0., 0., 1., 0., 0., 1., 0., 0., 1.]
    );
    assert_eq!(
        binary_floats(&exported.binary, arrays, "vertexColors"),
        [1.; 12]
    );
    assert!(
        binary_floats(&exported.binary, arrays, "uvs")
            .iter()
            .all(|value| value.is_finite())
    );
    let pbr_arrays = &exported.metadata["pbrBinaryGeometry"]["arrays"];
    for name in ["tangents", "binormals"] {
        let values = binary_floats(exported.pbr_binary.as_ref().unwrap(), pbr_arrays, name);
        assert_eq!(values.len(), 9);
        assert!(values.iter().all(|value| value.is_finite()));
        assert!(values.iter().any(|value| *value != 0.));
    }
}

fn armor_group(id: u64, zones: [f32; 3], degenerate: bool) -> Fields {
    let positions = if degenerate {
        [[0.; 3]; 3]
    } else {
        [[0., 0., 0.], [1., 0., 0.], [0., 1., 0.]]
    };
    let mut vertices = Vec::new();
    for (position, zone) in positions.into_iter().zip(zones) {
        vertices.extend(floats(&position));
        vertices.extend(floats(&[f32::NAN, 0., 0.]));
        vertices.extend(floats(&[zone]));
    }
    group(id, 1 | (1 << 1) | (1 << 9), 3, vertices, true, &[0, 1, 2])
}

const VEHICLE_XML: &[u8] = br#"<root>
<hull><armor><armor_3>50</armor_3></armor><turretPositions><turret>0 0 0</turret></turretPositions><hitTester><collisionModel>hull.sc2</collisionModel></hitTester></hull>
<turrets0><turret><armor><armor_3>40</armor_3></armor><gunPosition>0 0 0</gunPosition><guns><gun><armor><armor_3>20</armor_3></armor><hitTester><collisionModel>gun.sc2</collisionModel></hitTester></gun></guns></turret></turrets0>
</root>"#;

fn armor(groups: &[Fields]) -> Result<tank_armor::ExportedArmor, String> {
    tank_armor::export_armor(
        &scene(vec![node("gun", 2), node("hull", 1)]),
        &scg(groups),
        VEHICLE_XML,
        "usa",
        "Fixture",
        "fixture.sc2",
        "fixture.scg",
        "fixture.xml",
    )
}

#[test]
fn armor_export_preserves_nan_normal_repairs_thickness_and_degenerate_face_skips() {
    let exported = armor(&[
        armor_group(2, [3.; 3], true),
        armor_group(1, [3.; 3], false),
        vec![("#id", Value::Number(99))],
    ])
    .unwrap();
    assert_eq!(exported.face_count, 1);
    assert_eq!(exported.vertex_count, 3);
    assert_eq!(
        exported.mesh["positions"],
        json!([0., 0., 0., 1., 0., 0., 0., 1., 0.])
    );
    assert_eq!(
        exported.mesh["normals"],
        json!([0., 0., 1., 0., 0., 1., 0., 0., 1.])
    );
    assert_eq!(exported.mesh["thicknessMm"], json!([50, 50, 50]));
    assert_eq!(exported.mesh["armorIds"], json!([3, 3, 3]));
    assert_eq!(exported.mesh["parts"]["gun"]["faceCount"], 0);
    assert_eq!(exported.mesh["parts"]["hull"]["faceCount"], 1);
}

#[test]
fn armor_policy_keeps_required_streams_and_zone_validation() {
    let gun = armor_group(2, [3.; 3], false);
    for (hull, expected) in [
        (triangle(1, false), "normal attribute"),
        (
            group(1, 3, 3, vec![0; 72], false, &[0, 1, 2]),
            "armor-zone attribute",
        ),
        (armor_group(1, [3., 4., 3.], false), "mixed armor labels"),
        (armor_group(1, [3.5; 3], false), "non-integer"),
        (armor_group(1, [f32::NAN; 3], false), "non-finite"),
        (armor_group(1, [4.; 3], false), "no XML thickness"),
    ] {
        let error = armor(&[hull, gun.clone()]).unwrap_err();
        assert!(error.contains(expected), "{error}");
    }
}

#[test]
fn preserves_full_width_indices() {
    let fields = group(1, 1, 65_537, vec![0; 65_537 * 12], true, &[0, 65_536, 1]);
    let geometry = parse_scg(&scg(&[fields])).unwrap();
    assert_eq!(geometry.decode(1).unwrap().indices, [0, 65_536, 1]);
}

#[test]
fn showroom_policy_keeps_nonfinite_repairs_and_position_uv_errors() {
    let format = 1 | (1 << 1) | (1 << 3) | (1 << 7) | (1 << 8);
    let mut vertices = Vec::new();
    for (position, uv) in [
        ([0., 0., 0.], [0., 0.]),
        ([1., 0., 0.], [1., 0.]),
        ([0., 1., 0.], [0., 1.]),
    ] {
        vertices.extend(floats(&position));
        vertices.extend(floats(&[f32::NAN, 0., 0.]));
        vertices.extend(floats(&uv));
        vertices.extend(floats(&[f32::NAN, 0., 0., f32::NAN, 0., 0.]));
    }
    let fields = group(1, format, 3, vertices.clone(), false, &[0, 1, 2]);
    let exported = model(&scg(&[fields]), true).unwrap();
    assert_eq!(exported.generated_uv_groups, 0);
    let arrays = &exported.metadata["binaryGeometry"]["arrays"];
    assert_eq!(
        binary_floats(&exported.binary, arrays, "normals"),
        [0., 0., 1., 0., 0., 1., 0., 0., 1.]
    );
    for name in ["tangents", "binormals"] {
        let values = binary_floats(
            exported.pbr_binary.as_ref().unwrap(),
            &exported.metadata["pbrBinaryGeometry"]["arrays"],
            name,
        );
        assert!(values.iter().all(|value| value.is_finite()));
    }
    // First position and first UV in the interleaved stream.
    for offset in [0, 24] {
        let mut invalid = vertices.clone();
        invalid[offset..offset + 4].copy_from_slice(&f32::NAN.to_le_bytes());
        let fields = group(1, format, 3, invalid, false, &[0, 1, 2]);
        assert!(
            model(&scg(&[fields]), true)
                .unwrap_err()
                .contains("non-finite vertex attribute")
        );
    }
}

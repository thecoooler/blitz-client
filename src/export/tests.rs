use super::{armor as tank_armor, model as tank_model};
use crate::resources::archive::test_support::*;
use serde_json::{Value as Json, json};

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

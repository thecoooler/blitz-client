use super::{parse_scg, test_support::*};

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

#[test]
fn preserves_full_width_indices() {
    let fields = group(1, 1, 65_537, vec![0; 65_537 * 12], true, &[0, 65_536, 1]);
    let geometry = parse_scg(&scg(&[fields])).unwrap();
    assert_eq!(geometry.decode(1).unwrap().indices, [0, 65_536, 1]);
}

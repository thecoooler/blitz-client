use crate::asset_archive::{self, KaValue};
use crate::client::{XmlNode, child_text, parse_xml};
use crate::tank_model::{self, Transform};
use serde_json::{Map as JsonMap, Value as Json, json};
use std::collections::{BTreeMap, HashMap, HashSet};

#[derive(Debug)]
pub(crate) struct ExportedArmor {
    pub(crate) values: Json,
    pub(crate) mesh: Json,
    pub(crate) face_count: usize,
    pub(crate) vertex_count: usize,
}

#[derive(Debug)]
struct ScenePart {
    part: &'static str,
    transform: Transform,
}

pub(crate) fn export_armor(
    sc2_data: &[u8],
    scg_data: &[u8],
    xml_data: &[u8],
    nation: &str,
    tank_id: &str,
    sc2_source: &str,
    scg_source: &str,
    xml_source: &str,
) -> Result<ExportedArmor, String> {
    let root = parse_xml(xml_data)?;
    let value_parts = armor_values(&root)?;
    let armor_by_part = armor_thickness(&root)?;
    let mount_offsets = mount_offsets(&root)?;
    let scene = asset_archive::parse_sc2_scene(sc2_data)?;
    let expected_parts = part_collision_models(&root);
    let scene_parts = scene_parts(&scene, &expected_parts)?;
    let groups = asset_archive::parse_scg(scg_data)?;
    let mut groups_by_id = HashMap::new();
    for group in groups {
        let id = asset_archive::group_id(&group.fields)?;
        if groups_by_id.insert(id, group.fields).is_some() {
            return Err("duplicate SCG polygon group IDs".to_owned());
        }
    }
    for geometry_id in scene_parts.keys() {
        if !groups_by_id.contains_key(geometry_id) {
            return Err(format!(
                "SC2 geometry ID {geometry_id} has no SCG polygon group"
            ));
        }
    }

    let mut positions = Vec::<f32>::new();
    let mut normals = Vec::<f32>::new();
    let mut indices = Vec::<u32>::new();
    let mut armor_ids = Vec::<i32>::new();
    let mut thickness_mm = Vec::<i32>::new();
    let mut part_ids = Vec::<u8>::new();
    let mut faces = Vec::<Json>::new();
    let mut parts_meta = JsonMap::<String, Json>::new();
    let part_numeric_id = [("gun", 0u8), ("hull", 1u8), ("turret", 2u8)];

    let mut ordered_parts: Vec<_> = scene_parts.iter().collect();
    ordered_parts.sort_by_key(|(geometry_id, _)| **geometry_id);
    for (geometry_id, part) in ordered_parts {
        let group = groups_by_id.get(geometry_id).unwrap();
        let mut transform = part.transform.clone();
        let offset = mount_offsets
            .get(part.part)
            .ok_or_else(|| format!("missing XML mount position for {}", part.part))?;
        for (axis, value) in offset.iter().enumerate() {
            transform.translation[axis] += value;
        }

        let vertex_format = field_u64(group, "vertexFormat")? as u32;
        if vertex_format & 1 == 0 || vertex_format & (1 << 1) == 0 || vertex_format & (1 << 9) == 0
        {
            return Err(format!(
                "SCG group {geometry_id} lacks position, normal or armor-zone attribute"
            ));
        }
        let stride: usize = (0..20)
            .filter(|bit| vertex_format & (1 << bit) != 0)
            .map(asset_archive::attribute_bytes)
            .sum();
        let vertex_data = field_bytes(group, "vertices")?;
        let vertex_count = field_u64(group, "vertexCount")? as usize;
        let index_count = field_u64(group, "indexCount")? as usize;
        if vertex_data.len() != stride.saturating_mul(vertex_count) {
            return Err(format!(
                "SCG vertex buffer size mismatch in group {geometry_id}"
            ));
        }
        if field_u64(group, "rhi_primitiveType")? != 1 {
            return Err(format!("SCG group {geometry_id} is not a triangle list"));
        }
        let label_offset: usize = (0..9)
            .filter(|bit| vertex_format & (1 << bit) != 0)
            .map(asset_archive::attribute_bytes)
            .sum();
        let mut local_positions = Vec::with_capacity(vertex_count);
        let mut local_normals = Vec::with_capacity(vertex_count);
        let mut zones = Vec::with_capacity(vertex_count);
        for vertex in 0..vertex_count {
            let base = vertex * stride;
            let position = read_vec3(vertex_data, base)?;
            let normal = read_vec3(vertex_data, base + 12)?;
            let zone_value = read_f32(vertex_data, base + label_offset)?;
            if !position.iter().all(|value| value.is_finite()) || !zone_value.is_finite() {
                return Err(format!("non-finite SCG vertex data in group {geometry_id}"));
            }
            let zone = zone_value.round();
            if (zone_value - zone).abs() > 1e-5 || zone < i32::MIN as f32 || zone > i32::MAX as f32
            {
                return Err(format!(
                    "non-integer SCG armor label {zone_value} in group {geometry_id}"
                ));
            }
            local_positions.push(position);
            local_normals.push(normal);
            zones.push(zone as i32);
        }
        let local_indices = decode_indices(group, index_count)?;
        if index_count % 3 != 0 {
            return Err(format!(
                "SCG triangle index count is not divisible by 3 in group {geometry_id}"
            ));
        }
        let numeric_part = part_numeric_id
            .iter()
            .find(|(name, _)| *name == part.part)
            .unwrap()
            .1;
        let first_face = faces.len();
        let first_index = indices.len();
        for triangle in local_indices.chunks_exact(3) {
            if triangle.iter().any(|index| *index as usize >= vertex_count) {
                return Err(format!("SCG index out of range in geometry {geometry_id}"));
            }
            let mut label_iter = triangle.iter().map(|index| zones[*index as usize]);
            let armor_id = label_iter.next().unwrap();
            if label_iter.any(|label| label != armor_id) {
                return Err(format!(
                    "triangle has mixed armor labels in group {geometry_id}"
                ));
            }
            let millimeters = armor_by_part
                .get(part.part)
                .and_then(|values| values.get(&armor_id))
                .copied()
                .ok_or_else(|| {
                    format!(
                        "SCG label armor_{armor_id} has no XML thickness for {}",
                        part.part
                    )
                })?;
            let first_triangle_index = indices.len();
            // The shipped collision meshes contain degenerate triangles whose stored
            // normals are NaN; derive the plane normal instead, or skip the face when
            // the triangle has no area to render.
            let repaired_normal = if triangle.iter().any(|index| {
                !local_normals[*index as usize]
                    .iter()
                    .all(|value| value.is_finite())
            }) {
                match triangle_face_normal(&local_positions, triangle) {
                    Some(normal) => Some(normal),
                    None => continue,
                }
            } else {
                None
            };
            for index in triangle {
                let index = *index as usize;
                positions.extend(tank_model::transform_position(
                    local_positions[index],
                    &transform,
                )?);
                normals.extend(tank_model::transform_normal(
                    repaired_normal.unwrap_or(local_normals[index]),
                    &transform,
                )?);
                indices.push(indices.len() as u32);
                armor_ids.push(armor_id);
                thickness_mm.push(millimeters);
                part_ids.push(numeric_part);
            }
            faces.push(json!({
                "firstIndex": first_triangle_index,
                "firstVertex": first_triangle_index,
                "indexCount": 3,
                "part": part.part,
                "partId": numeric_part,
                "armorId": armor_id,
                "mm": millimeters,
            }));
        }
        let matrix = tank_model::world_matrix(&transform)?;
        parts_meta.insert(
            part.part.to_owned(),
            json!({
                "node": part.transform.node,
                "geometryId": geometry_id,
                "transform": {
                    "translation": transform.translation,
                    "rotationQuaternionXYZW": transform.rotation,
                    "scale": transform.scale,
                    "matrixColumnMajor": matrix,
                },
                "mountOffsetFromVehicleXml": offset,
                "firstFace": first_face,
                "faceCount": faces.len() - first_face,
                "firstIndex": first_index,
                "indexCount": indices.len() - first_index,
            }),
        );
    }

    let values = json!({
        "vehicle": format!("{nation}.{tank_id}"),
        "source": xml_source,
        "parts": value_parts,
    });
    let face_count = faces.len();
    let vertex_count = positions.len() / 3;
    let mesh = json!({
        "formatVersion": 2,
        "tank": format!("{nation}-{tank_id}"),
        "coordinateSpace": "assembled SCG model coordinates; SC2 world transforms plus XML component mount positions applied",
        "primitive": "triangles",
        "source": {
            "scene": sc2_source,
            "geometry": scg_source,
            "vehicleXml": xml_source,
            "provenance": "Decoded from the locally installed World of Tanks Blitz client; collision SCG zone labels joined to XML armor values and XML turret/gun mount positions; no tank-viewer site assets or code used.",
        },
        "attributeEncoding": {
            "positions": "flat xyz floats; triangle soup",
            "normals": "flat xyz floats; triangle soup",
            "indices": "sequential indices; each face uses 3 vertices",
            "partIds": {"0": "gun", "1": "hull", "2": "turret"},
            "armorIds": "XML armor_N suffix carried by SCG per-vertex zone labels",
            "thicknessMm": "XML armor_N thickness repeated per vertex",
        },
        "parts": parts_meta,
        "positions": positions,
        "normals": normals,
        "indices": indices,
        "partIds": part_ids,
        "armorIds": armor_ids,
        "thicknessMm": thickness_mm,
        "faces": faces,
    });
    Ok(ExportedArmor {
        values,
        mesh,
        face_count,
        vertex_count,
    })
}

fn part_collision_models(root: &XmlNode) -> Vec<(&'static str, String)> {
    let hull = root.children.iter().find(|node| node.name == "hull");
    let turret = root
        .children
        .iter()
        .find(|node| node.name == "turrets0")
        .and_then(|node| node.children.first());
    let gun = turret
        .and_then(|node| node.children.iter().find(|child| child.name == "guns"))
        .and_then(|node| node.children.first());
    [("hull", hull), ("turret", turret), ("gun", gun)]
        .into_iter()
        .filter_map(|(part, node)| {
            node.and_then(collision_model_name)
                .map(|model| (part, model))
        })
        .collect()
}

fn collision_model_name(node: &XmlNode) -> Option<String> {
    let model = node
        .children
        .iter()
        .find(|child| child.name == "hitTester")
        .and_then(|hit_tester| child_text(hit_tester, "collisionModel"))?;
    let file = model.rsplit('/').next()?;
    Some(file.split('.').next()?.to_ascii_lowercase())
}

fn scene_parts(
    scene: &KaValue,
    expected: &[(&'static str, String)],
) -> Result<BTreeMap<u64, ScenePart>, String> {
    let mut result = BTreeMap::new();
    let mut scene_names = Vec::new();
    for node in scene
        .get("#hierarchy")
        .and_then(KaValue::as_array)
        .unwrap_or_default()
    {
        let name = node
            .get("name")
            .and_then(KaValue::as_str)
            .unwrap_or_default();
        scene_names.push(name.to_owned());
        let Some((part_name, _)) = expected
            .iter()
            .find(|(_, model)| model.eq_ignore_ascii_case(name))
        else {
            continue;
        };
        let mut geometry_ids = BTreeSetCompat::default();
        let components = node
            .get("components")
            .and_then(KaValue::as_map)
            .ok_or_else(|| format!("SC2 collision node {name} has no components"))?;
        for component in components.values() {
            if component.get("comp.typename").and_then(KaValue::as_str) != Some("RenderComponent") {
                continue;
            }
            let batches = component
                .get("rc.renderObj")
                .and_then(|value| value.get("ro.batches"))
                .and_then(KaValue::as_map);
            for batch in batches.into_iter().flat_map(|map| map.values()) {
                if let Some(id) = batch.get("rb.datasource").and_then(KaValue::as_u64) {
                    geometry_ids.insert(id);
                }
            }
        }
        if geometry_ids.len() != 1 {
            return Err(format!(
                "expected one SCG datasource on SC2 node {name}, got {geometry_ids:?}"
            ));
        }
        let transform = tank_model::node_transform(node)?;
        result.insert(
            *geometry_ids.iter().next().unwrap(),
            ScenePart {
                part: part_name,
                transform,
            },
        );
    }
    let found: HashSet<_> = result.values().map(|part| part.part).collect();
    for required in ["hull", "gun"] {
        if !found.contains(required) {
            let model = expected
                .iter()
                .find(|(part, _)| *part == required)
                .map(|(_, model)| model.as_str())
                .unwrap_or("?");
            return Err(format!(
                "collision scene has no {required} node (expected {model:?}); scene nodes: {scene_names:?}"
            ));
        }
    }
    Ok(result)
}

#[derive(Default, Debug)]
struct BTreeSetCompat(Vec<u64>);

impl BTreeSetCompat {
    fn insert(&mut self, id: u64) {
        if !self.0.contains(&id) {
            self.0.push(id);
        }
    }
    fn len(&self) -> usize {
        self.0.len()
    }
    fn iter(&self) -> std::slice::Iter<'_, u64> {
        self.0.iter()
    }
}

fn armor_values(root: &XmlNode) -> Result<Json, String> {
    let hull = root.children.iter().find(|node| node.name == "hull");
    let turret = root
        .children
        .iter()
        .find(|node| node.name == "turrets0")
        .and_then(|node| node.children.first());
    let gun = turret
        .and_then(|node| node.children.iter().find(|child| child.name == "guns"))
        .and_then(|node| node.children.first());
    let parts = json!({
        "hull": parse_armor_part(hull, "Корпус")?,
        "turret": parse_armor_part(turret, "Башня")?,
        "gun": parse_armor_part(gun, "Орудие")?,
    });
    Ok(parts)
}

fn parse_armor_part(node: Option<&XmlNode>, label: &str) -> Result<Json, String> {
    let node = node.ok_or_else(|| format!("vehicle XML is missing armor part {label}"))?;
    let armor = node
        .children
        .iter()
        .find(|child| child.name == "armor")
        .ok_or_else(|| format!("vehicle XML is missing armor values for {label}"))?;
    let values: Vec<_> = armor
        .children
        .iter()
        .filter_map(|entry| {
            let text = entry.text.trim();
            if text.is_empty() {
                return None;
            }
            let value = text.parse::<f64>().ok()? as i32;
            Some(json!({"id": entry.name, "mm": value}))
        })
        .collect();
    if values.is_empty() {
        return Err(format!("vehicle XML has no armor values for {label}"));
    }
    Ok(json!({
        "label": label,
        "primaryArmor": child_text(node, "primaryArmor").unwrap_or_default().split_whitespace().collect::<Vec<_>>(),
        "values": values,
    }))
}

fn armor_thickness(root: &XmlNode) -> Result<HashMap<&'static str, HashMap<i32, i32>>, String> {
    let hull = root.children.iter().find(|node| node.name == "hull");
    let turret = root
        .children
        .iter()
        .find(|node| node.name == "turrets0")
        .and_then(|node| node.children.first());
    let gun = turret
        .and_then(|node| node.children.iter().find(|child| child.name == "guns"))
        .and_then(|node| node.children.first());
    Ok(HashMap::from([
        (
            "hull",
            parse_thickness(
                hull.and_then(|node| node.children.iter().find(|child| child.name == "armor")),
                "hull",
            )?,
        ),
        (
            "turret",
            parse_thickness(
                turret.and_then(|node| node.children.iter().find(|child| child.name == "armor")),
                "turret",
            )?,
        ),
        (
            "gun",
            parse_thickness(
                gun.and_then(|node| node.children.iter().find(|child| child.name == "armor")),
                "gun",
            )?,
        ),
    ]))
}

fn parse_thickness(node: Option<&XmlNode>, label: &str) -> Result<HashMap<i32, i32>, String> {
    let node = node.ok_or_else(|| format!("XML armor block missing: {label}"))?;
    let mut values = HashMap::new();
    for entry in &node.children {
        let Some(number) = entry.name.strip_prefix("armor_") else {
            continue;
        };
        let id = number
            .parse::<i32>()
            .map_err(|_| format!("invalid armor label {}", entry.name))?;
        let text = entry.text.trim();
        if text.is_empty() {
            return Err(format!(
                "missing numeric thickness for {label}/{}",
                entry.name
            ));
        }
        let thickness = text
            .parse::<f64>()
            .map_err(|_| format!("invalid armor thickness for {label}/{}", entry.name))?
            as i32;
        values.insert(id, thickness);
    }
    if values.is_empty() {
        return Err(format!("empty armor block: {label}"));
    }
    Ok(values)
}

fn mount_offsets(root: &XmlNode) -> Result<HashMap<&'static str, [f32; 3]>, String> {
    let hull = root
        .children
        .iter()
        .find(|node| node.name == "hull")
        .ok_or_else(|| "vehicle XML has no hull".to_owned())?;
    let positions = hull
        .children
        .iter()
        .find(|node| node.name == "turretPositions")
        .ok_or_else(|| "XML mount position missing: hull/turretPositions".to_owned())?;
    let turret = client_position(
        child_text(positions, "turret").as_deref(),
        "hull/turretPositions/turret",
    )?;
    let turret_node = root
        .children
        .iter()
        .find(|node| node.name == "turrets0")
        .and_then(|node| node.children.first())
        .ok_or_else(|| "vehicle turret definition not found in XML".to_owned())?;
    let gun_local = client_position(
        child_text(turret_node, "gunPosition").as_deref(),
        "turret/gunPosition",
    )?;
    Ok(HashMap::from([
        ("hull", [0.0, 0.0, 0.0]),
        ("turret", turret),
        (
            "gun",
            [
                turret[0] + gun_local[0],
                turret[1] + gun_local[1],
                turret[2] + gun_local[2],
            ],
        ),
    ]))
}

fn client_position(value: Option<&str>, label: &str) -> Result<[f32; 3], String> {
    let value = value.ok_or_else(|| format!("XML mount position missing: {label}"))?;
    let numbers: Vec<f32> = value
        .split_whitespace()
        .map(|item| {
            item.parse::<f32>()
                .map_err(|_| format!("invalid XML mount position for {label}: {value:?}"))
        })
        .collect::<Result<_, _>>()?;
    if numbers.len() != 3 || numbers.iter().any(|number| !number.is_finite()) {
        return Err(format!("invalid XML mount position for {label}: {value:?}"));
    }
    Ok([numbers[0], numbers[2], numbers[1]])
}

fn field_u64(map: &KaValue, field: &str) -> Result<u64, String> {
    map.get(field)
        .and_then(KaValue::as_u64)
        .ok_or_else(|| format!("missing/invalid SCG field {field}"))
}

fn field_bytes<'a>(map: &'a KaValue, field: &str) -> Result<&'a [u8], String> {
    map.get(field)
        .and_then(KaValue::as_bytes)
        .ok_or_else(|| format!("missing/invalid SCG byte field {field}"))
}

fn read_f32(bytes: &[u8], offset: usize) -> Result<f32, String> {
    let end = offset
        .checked_add(4)
        .ok_or_else(|| "vertex offset overflow".to_owned())?;
    let bytes = bytes
        .get(offset..end)
        .ok_or_else(|| "vertex attribute outside SCG buffer".to_owned())?;
    Ok(f32::from_le_bytes(bytes.try_into().unwrap()))
}

fn read_vec3(bytes: &[u8], offset: usize) -> Result<[f32; 3], String> {
    Ok([
        read_f32(bytes, offset)?,
        read_f32(bytes, offset + 4)?,
        read_f32(bytes, offset + 8)?,
    ])
}

fn triangle_face_normal(positions: &[[f32; 3]], triangle: &[u32]) -> Option<[f32; 3]> {
    let [a, b, c] = [
        triangle[0] as usize,
        triangle[1] as usize,
        triangle[2] as usize,
    ];
    let (start, second, third) = (positions[a], positions[b], positions[c]);
    let edge1 = [
        second[0] - start[0],
        second[1] - start[1],
        second[2] - start[2],
    ];
    let edge2 = [
        third[0] - start[0],
        third[1] - start[1],
        third[2] - start[2],
    ];
    let normal = [
        edge1[1] * edge2[2] - edge1[2] * edge2[1],
        edge1[2] * edge2[0] - edge1[0] * edge2[2],
        edge1[0] * edge2[1] - edge1[1] * edge2[0],
    ];
    let length = (normal[0] * normal[0] + normal[1] * normal[1] + normal[2] * normal[2]).sqrt();
    if length <= f32::EPSILON {
        return None;
    }
    Some([normal[0] / length, normal[1] / length, normal[2] / length])
}

fn decode_indices(group: &KaValue, count: usize) -> Result<Vec<u32>, String> {
    let bytes = field_bytes(group, "indices")?;
    match field_u64(group, "indexFormat")? {
        0 if bytes.len() == count.saturating_mul(2) => Ok(bytes
            .chunks_exact(2)
            .map(|pair| u16::from_le_bytes([pair[0], pair[1]]) as u32)
            .collect()),
        1 if bytes.len() == count.saturating_mul(4) => Ok(bytes
            .chunks_exact(4)
            .map(|quad| u32::from_le_bytes(quad.try_into().unwrap()))
            .collect()),
        format @ (0 | 1) => Err(format!(
            "SCG index buffer size mismatch for format {format}"
        )),
        format => Err(format!("unknown SCG index format {format}")),
    }
}

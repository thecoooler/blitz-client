use crate::resources::archive::{self as asset_archive, KaValue};
use serde_json::{Map as JsonMap, Value as Json};
use std::collections::{BTreeMap, BTreeSet, HashMap, HashSet};
use std::path::{Path, PathBuf};

type Vec3 = [f32; 3];
type Vec2 = [f32; 2];

#[derive(Debug, Clone)]
pub(super) struct Transform {
    pub(super) translation: Vec3,
    pub(super) rotation: [f32; 4],
    pub(super) scale: Vec3,
    pub(super) node: String,
}

#[derive(Debug, Clone)]
struct MaterialInfo {
    maps: BTreeMap<String, String>,
    alpha_test: bool,
    alpha_threshold: f32,
}

#[derive(Debug, Clone)]
struct Binding {
    geometry_id: u64,
    material_id: u64,
    node: String,
    transform: Transform,
    material: MaterialInfo,
}

#[derive(Debug)]
pub(super) struct ExportedModel {
    pub(super) metadata: Json,
    pub(super) binary: Vec<u8>,
    pub(super) pbr_binary: Option<Vec<u8>>,
    pub(super) textures: Vec<TextureSource>,
    pub(super) vertex_count: usize,
    pub(super) triangle_count: usize,
    pub(super) material_count: usize,
    pub(super) fallback_maps: usize,
    pub(super) generated_uv_groups: usize,
}

#[derive(Debug, Clone)]
pub(super) struct TextureSource {
    pub(super) file: PathBuf,
    pub(super) source: Option<PathBuf>,
    pub(super) role: String,
    pub(super) lossless: bool,
}

pub(super) fn export_model(
    sc2_data: &[u8],
    scg_data: &[u8],
    sc2_source: &str,
    scg_source: &str,
    tank_id: &str,
    include_pbr: bool,
    texture_format: &str,
    webp_quality: u8,
    compress: bool,
) -> Result<ExportedModel, String> {
    if texture_format != "webp" && texture_format != "png" {
        return Err("texture format must be png or webp".to_owned());
    }
    let scene = asset_archive::parse_sc2_scene(sc2_data)?;
    let materials = material_nodes(&scene)?;
    let bindings = visible_batches(&scene, &materials, include_pbr)?;
    if bindings.is_empty() {
        return Err("SC2 scene contains no visible render batches".to_owned());
    }
    let geometry = asset_archive::parse_scg(scg_data)?;
    let missing: Vec<_> = bindings
        .iter()
        .map(|binding| binding.geometry_id)
        .collect::<BTreeSet<_>>()
        .into_iter()
        .filter(|id| !geometry.contains_group(*id))
        .collect();
    if !missing.is_empty() {
        return Err(format!(
            "SC2 batches reference missing SCG geometry groups: {missing:?}"
        ));
    }

    let selected_maps: Vec<BTreeMap<String, String>> = bindings
        .iter()
        .map(|binding| {
            if include_pbr {
                binding.material.maps.clone()
            } else {
                BTreeMap::from([(
                    "baseColor".to_owned(),
                    binding
                        .material
                        .maps
                        .get("baseColor")
                        .cloned()
                        .unwrap_or_else(|| "@fallback/baseColor".to_owned()),
                )])
            }
        })
        .collect();
    let texture_sets: BTreeSet<_> = selected_maps.iter().cloned().collect();
    let texture_ids: BTreeMap<_, _> = texture_sets
        .iter()
        .cloned()
        .enumerate()
        .map(|(index, signature)| (signature, index as u32))
        .collect();

    let mut positions = Vec::<f32>::new();
    let mut normals = Vec::<f32>::new();
    let mut tangents = include_pbr.then(Vec::<f32>::new);
    let mut binormals = include_pbr.then(Vec::<f32>::new);
    let mut uvs = Vec::<f32>::new();
    let mut vertex_colors = Vec::<f32>::new();
    let mut vertex_texture_ids = Vec::<u32>::new();
    let mut indices = Vec::<u32>::new();
    let mut group_ranges = Vec::<Json>::new();
    let mut generated_uv_groups = 0;

    let mut ordered_bindings: Vec<_> = bindings.iter().enumerate().collect();
    ordered_bindings.sort_by(|(_, left), (_, right)| {
        (left.geometry_id, &left.node).cmp(&(right.geometry_id, &right.node))
    });

    for (binding_index, binding) in ordered_bindings {
        let group = geometry.decode(binding.geometry_id)?;
        let vertex_count = group.positions.len();
        let has_uvs = group.uvs.is_some();
        let local_positions = group.positions;
        let mut local_normals = group
            .normals
            .unwrap_or_else(|| vec![[0.0; 3]; vertex_count]);
        let mut local_uvs = group.uvs.unwrap_or_else(|| vec![[0.0; 2]; vertex_count]);
        if !local_positions
            .iter()
            .flatten()
            .chain(local_uvs.iter().flatten())
            .all(|value| value.is_finite())
        {
            return Err(format!(
                "non-finite vertex attribute in SCG group {}",
                binding.geometry_id
            ));
        }
        let local_colors = group.colors.unwrap_or_else(|| vec![[1.0; 4]; vertex_count]);
        let local_tangents = if include_pbr {
            group
                .tangents
                .unwrap_or_else(|| vec![[0.0; 3]; vertex_count])
        } else {
            Vec::new()
        };
        let local_binormals = if include_pbr {
            group
                .binormals
                .unwrap_or_else(|| vec![[0.0; 3]; vertex_count])
        } else {
            Vec::new()
        };
        let local_indices = group.indices;
        if !has_uvs {
            local_uvs = planar_uvs(&local_positions);
            generated_uv_groups += 1;
        }
        let rebuilt_normals = fallback_normals(&local_positions, &local_indices);
        for (index, normal) in local_normals.iter_mut().enumerate() {
            let length = length3(*normal);
            *normal = if length <= 1e-15 || !length.is_finite() {
                rebuilt_normals[index]
            } else {
                scale3(*normal, 1.0 / length)
            };
        }
        let (fallback_tangents, fallback_binormals) = if include_pbr {
            fallback_tangent_basis(&local_positions, &local_normals, &local_uvs, &local_indices)
        } else {
            (Vec::new(), Vec::new())
        };

        let vertex_base = (positions.len() / 3) as u32;
        let first_index = indices.len();
        let texture_id = *texture_ids
            .get(&selected_maps[binding_index])
            .ok_or_else(|| "internal texture set lookup failed".to_owned())?;
        for vertex in 0..vertex_count {
            positions.extend(transform_position(
                local_positions[vertex],
                &binding.transform,
            )?);
            normals.extend(transform_normal(local_normals[vertex], &binding.transform)?);
            if include_pbr {
                let tangent_length = length3(local_tangents[vertex]);
                let tangent = if !tangent_length.is_finite() || tangent_length <= 1e-15 {
                    fallback_tangents[vertex]
                } else {
                    local_tangents[vertex]
                };
                let binormal_length = length3(local_binormals[vertex]);
                let binormal = if !binormal_length.is_finite() || binormal_length <= 1e-15 {
                    fallback_binormals[vertex]
                } else {
                    local_binormals[vertex]
                };
                tangents
                    .as_mut()
                    .unwrap()
                    .extend(transform_direction(tangent, &binding.transform)?);
                binormals
                    .as_mut()
                    .unwrap()
                    .extend(transform_direction(binormal, &binding.transform)?);
            }
            uvs.extend(local_uvs[vertex]);
            vertex_colors.extend(local_colors[vertex]);
            vertex_texture_ids.push(texture_id);
        }
        indices.extend(local_indices.into_iter().map(|index| vertex_base + index));
        group_ranges.push(serde_json::json!({
            "geometryId": binding.geometry_id,
            "node": binding.node,
            "materialId": binding.material_id,
            "textureId": texture_id,
            "firstIndex": first_index,
            "indexCount": indices.len() - first_index,
            "generatedUVs": !has_uvs,
        }));
    }

    let materials_json = build_material_json(
        &texture_sets,
        &bindings,
        &selected_maps,
        texture_format,
        include_pbr,
    );
    let mut textures = Vec::new();
    let mut fallback_maps = 0;
    for material in &materials_json {
        if let Some(maps) = material.get("maps").and_then(Json::as_object) {
            for (role, map) in maps {
                let file = map.get("file").and_then(Json::as_str).unwrap_or_default();
                let fallback = map.get("fallback").and_then(Json::as_bool).unwrap_or(false);
                fallback_maps += usize::from(fallback);
                let material_path = map.get("materialPath").and_then(Json::as_str);
                let source = material_path
                    .map(|path| PathBuf::from(path.replace('\\', "/")))
                    .map(|mut path| {
                        path.set_extension("dx11.dds.dvpl");
                        path
                    });
                textures.push(TextureSource {
                    file: PathBuf::from(file),
                    source,
                    role: role.clone(),
                    lossless: include_pbr && role != "baseColor" && texture_format == "webp",
                });
            }
        }
    }

    let max_texture_id = vertex_texture_ids.iter().copied().max().unwrap_or(0);
    let texture_type = if max_texture_id <= u8::MAX as u32 {
        ("uint8", 1)
    } else if max_texture_id <= u16::MAX as u32 {
        ("uint16", 2)
    } else {
        ("uint32", 4)
    };
    let max_index = indices.iter().copied().max().unwrap_or(0);
    let index_type = if max_index <= u16::MAX as u32 {
        ("uint16", 2)
    } else {
        ("uint32", 4)
    };
    let mut binary = Vec::new();
    let mut arrays = JsonMap::new();
    push_f32_array(&mut binary, &mut arrays, "positions", &positions);
    push_f32_array(&mut binary, &mut arrays, "normals", &normals);
    push_f32_array(&mut binary, &mut arrays, "uvs", &uvs);
    push_f32_array(&mut binary, &mut arrays, "vertexColors", &vertex_colors);
    let texture_offset = align_buffer(&mut binary, texture_type.1);
    for value in &vertex_texture_ids {
        match texture_type.1 {
            1 => binary.push(*value as u8),
            2 => binary.extend_from_slice(&(*value as u16).to_le_bytes()),
            _ => binary.extend_from_slice(&value.to_le_bytes()),
        }
    }
    arrays.insert(
        "textureIds".to_owned(),
        array_descriptor(texture_type.0, texture_offset, vertex_texture_ids.len()),
    );
    let index_offset = align_buffer(&mut binary, index_type.1);
    for value in &indices {
        if index_type.1 == 2 {
            binary.extend_from_slice(&(*value as u16).to_le_bytes());
        } else {
            binary.extend_from_slice(&value.to_le_bytes());
        }
    }
    arrays.insert(
        "indices".to_owned(),
        array_descriptor(index_type.0, index_offset, indices.len()),
    );

    let (pbr_binary, pbr_descriptors) = if include_pbr {
        let mut data = Vec::new();
        let mut descriptors = JsonMap::new();
        push_f32_array(
            &mut data,
            &mut descriptors,
            "tangents",
            tangents.as_deref().unwrap_or(&[]),
        );
        push_f32_array(
            &mut data,
            &mut descriptors,
            "binormals",
            binormals.as_deref().unwrap_or(&[]),
        );
        (Some(data), Some(descriptors))
    } else {
        (None, None)
    };

    let mut texture_id_json = JsonMap::new();
    for (signature, id) in &texture_ids {
        let maps: JsonMap<String, Json> = signature
            .iter()
            .map(|(role, path)| (role.clone(), Json::String(path.clone())))
            .collect();
        texture_id_json.insert(id.to_string(), Json::Object(maps));
    }
    let mut attribute_encoding = serde_json::json!({
        "positions": "flat xyz floats",
        "normals": "flat xyz floats",
        "uvs": "flat uv pairs from SCG TEXCOORD0; planar UVs generated where the client group has no UV stream",
        "vertexColors": "flat rgba floats from SCG vertex color; default white when absent",
        "textureIds": texture_id_json,
    });
    if include_pbr {
        attribute_encoding["tangents"] =
            Json::String("flat xyz floats from SCG TANGENT".to_owned());
        attribute_encoding["binormals"] =
            Json::String("flat xyz floats from SCG BINORMAL".to_owned());
    }
    let source_name = Path::new(sc2_source)
        .file_name()
        .and_then(|name| name.to_str())
        .unwrap_or(tank_id)
        .trim_end_matches(".sc2.dvpl")
        .trim_end_matches(".sc2");
    let mut metadata = serde_json::json!({
        "formatVersion": 5,
        "pbrEnabled": include_pbr,
        "textureFormat": texture_format,
        "webpQuality": if texture_format == "webp" { Json::from(webp_quality) } else { Json::Null },
        "pbrWebpLossless": include_pbr && texture_format == "webp",
        "tank": source_name,
        "coordinateSpace": "SC2 world/model coordinates; render-node world transforms applied",
        "primitive": "triangles",
        "source": {
            "scene": sc2_source,
            "geometry": scg_source,
            "provenance": "Decoded from the locally installed World of Tanks Blitz client; no tank-viewer site assets or code used.",
            "omittedSceneNodes": ["crash state", "HP_* marker helpers"],
        },
        "materials": materials_json,
        "attributeEncoding": attribute_encoding,
        "groups": group_ranges,
        "generatedUvGroupCount": generated_uv_groups,
    });
    metadata["binaryGeometry"] = serde_json::json!({
        "file": if compress { "model.bin.br" } else { "model.bin" },
        "byteLength": binary.len(),
        "arrays": arrays,
    });
    if include_pbr {
        let pbr = pbr_binary.as_ref().unwrap();
        metadata["pbrBinaryGeometry"] = serde_json::json!({
            "file": if compress { "model.pbr.bin.br" } else { "model.pbr.bin" },
            "byteLength": pbr.len(),
            "arrays": pbr_descriptors.unwrap(),
        });
    }

    Ok(ExportedModel {
        metadata,
        binary,
        pbr_binary,
        textures,
        vertex_count: positions.len() / 3,
        triangle_count: indices.len() / 3,
        material_count: materials_json.len(),
        fallback_maps,
        generated_uv_groups,
    })
}

fn material_nodes(scene: &KaValue) -> Result<HashMap<u64, KaValue>, String> {
    let mut result = HashMap::new();
    for node in scene
        .get("#dataNodes")
        .and_then(KaValue::as_array)
        .unwrap_or_default()
    {
        let Some(id) = node.get("#id") else { continue };
        let id = match id {
            KaValue::Bytes(bytes) if bytes.len() == 8 => {
                u64::from_le_bytes(bytes.as_slice().try_into().unwrap())
            }
            value => value
                .as_u64()
                .ok_or_else(|| format!("unexpected SC2 data-node id: {value:?}"))?,
        };
        result.insert(id, node.clone());
    }
    Ok(result)
}

fn visible_batches(
    scene: &KaValue,
    materials: &HashMap<u64, KaValue>,
    include_pbr: bool,
) -> Result<Vec<Binding>, String> {
    fn visit(
        nodes: &[KaValue],
        parents: &[String],
        materials: &HashMap<u64, KaValue>,
        include_pbr: bool,
        seen: &mut HashSet<String>,
        result: &mut Vec<Binding>,
    ) -> Result<(), String> {
        for node in nodes {
            let name = node
                .get("name")
                .and_then(KaValue::as_str)
                .unwrap_or("unnamed")
                .to_owned();
            let mut path = parents.to_vec();
            path.push(name.clone());
            let lowered: Vec<_> = path.iter().map(|item| item.to_ascii_lowercase()).collect();
            if lowered
                .iter()
                .any(|part| part.contains("crash") || part.starts_with("hp_"))
            {
                continue;
            }
            let transform = node_transform(node)?;
            if let Some(components) = node.get("components").and_then(KaValue::as_map) {
                for component in components.values() {
                    if component.get("comp.typename").and_then(KaValue::as_str)
                        != Some("RenderComponent")
                    {
                        continue;
                    }
                    let batches = component
                        .get("rc.renderObj")
                        .and_then(|value| value.get("ro.batches"))
                        .and_then(KaValue::as_map);
                    let Some(batches) = batches else { continue };
                    for batch in batches.values() {
                        let Some(geometry) = batch.get("rb.datasource").and_then(KaValue::as_u64)
                        else {
                            continue;
                        };
                        let material_id = batch
                            .get("rb.nmatname")
                            .and_then(KaValue::as_u64)
                            .ok_or_else(|| "SC2 render batch has invalid material id".to_owned())?;
                        let material = material_info(material_id, materials)?;
                        let signature = serde_json::to_string(&(
                            geometry,
                            &material.maps,
                            material.alpha_test,
                            material.alpha_threshold,
                            &transform.node,
                            &transform.translation,
                            &transform.rotation,
                            &transform.scale,
                        ))
                        .map_err(|error| error.to_string())?;
                        if !seen.insert(signature) {
                            continue;
                        }
                        let maps = if include_pbr {
                            material.maps.clone()
                        } else {
                            BTreeMap::from([(
                                "baseColor".to_owned(),
                                material
                                    .maps
                                    .get("baseColor")
                                    .cloned()
                                    .unwrap_or_else(|| "@fallback/baseColor".to_owned()),
                            )])
                        };
                        result.push(Binding {
                            geometry_id: geometry,
                            material_id,
                            node: format!("/{}", path.join("/")),
                            transform: transform.clone(),
                            material: MaterialInfo { maps, ..material },
                        });
                    }
                }
            }
            if let Some(children) = node.get("#hierarchy").and_then(KaValue::as_array) {
                visit(children, &path, materials, include_pbr, seen, result)?;
            }
        }
        Ok(())
    }

    let mut result = Vec::new();
    let mut seen = HashSet::new();
    let nodes = scene
        .get("#hierarchy")
        .and_then(KaValue::as_array)
        .unwrap_or_default();
    visit(nodes, &[], materials, include_pbr, &mut seen, &mut result)?;
    Ok(result)
}

pub(super) fn node_transform(node: &KaValue) -> Result<Transform, String> {
    let name = node
        .get("name")
        .and_then(KaValue::as_str)
        .unwrap_or("unnamed")
        .to_owned();
    let components = node
        .get("components")
        .and_then(KaValue::as_map)
        .ok_or_else(|| format!("node {name:?} has no component map"))?;
    let matches: Vec<_> = components
        .values()
        .filter(|component| {
            component.get("comp.typename").and_then(KaValue::as_str) == Some("TransformComponent")
        })
        .collect();
    if matches.len() != 1 {
        return Err(format!(
            "expected one transform component on node {name:?}, got {}",
            matches.len()
        ));
    }
    let component = matches[0];
    Ok(Transform {
        translation: vector::<3>(
            component.get("tc.worldTranslation"),
            [0.0, 0.0, 0.0],
            &name,
            "tc.worldTranslation",
        )?,
        rotation: vector::<4>(
            component.get("tc.worldRotation"),
            [0.0, 0.0, 0.0, 1.0],
            &name,
            "tc.worldRotation",
        )?,
        scale: vector::<3>(
            component.get("tc.worldScale"),
            [1.0, 1.0, 1.0],
            &name,
            "tc.worldScale",
        )?,
        node: name,
    })
}

fn vector<const N: usize>(
    value: Option<&KaValue>,
    default: [f32; N],
    node: &str,
    key: &str,
) -> Result<[f32; N], String> {
    let Some(value) = value else {
        return Ok(default);
    };
    let items = value
        .as_array()
        .ok_or_else(|| format!("invalid {key} on node {node:?}"))?;
    if items.len() != N {
        return Err(format!("invalid {key} on node {node:?}"));
    }
    let mut result = [0.0; N];
    for (index, item) in items.iter().enumerate() {
        let number = item
            .as_f64()
            .ok_or_else(|| format!("invalid {key} on node {node:?}"))? as f32;
        if !number.is_finite() {
            return Err(format!("non-finite {key} on node {node:?}"));
        }
        result[index] = number;
    }
    Ok(result)
}

fn material_info(id: u64, materials: &HashMap<u64, KaValue>) -> Result<MaterialInfo, String> {
    let instance = materials
        .get(&id)
        .ok_or_else(|| format!("SC2 render batch references missing material id {id}"))?;
    let parent_id = instance.get("parentMaterialKey").and_then(KaValue::as_u64);
    let parent = parent_id
        .and_then(|parent| materials.get(&parent))
        .unwrap_or(instance);
    let mut configs: Vec<_> = parent
        .as_map()
        .into_iter()
        .flat_map(|map| map.iter())
        .filter(|(key, value)| key.starts_with("configArchive_") && value.as_map().is_some())
        .map(|(_, value)| value)
        .collect();
    if configs.is_empty() && parent.get("textures").and_then(KaValue::as_map).is_some() {
        configs.push(parent);
    }
    configs.sort_by_key(|config| {
        if config.get("configName").and_then(KaValue::as_str) == Some("Default") {
            0
        } else {
            1
        }
    });
    // Older client assets carry only the pre-PBR slot names, so each role accepts
    // candidate slots in priority order.
    let slots: [(&str, &[&str]); 4] = [
        ("baseColor", &["baseColorMap", "albedo"]),
        ("normal", &["baseNormalMap", "normalmap"]),
        ("roughnessMetallic", &["baseRMMap"]),
        ("misc", &["miscMap"]),
    ];
    let mut selected: Option<(usize, BTreeMap<String, String>, &KaValue)> = None;
    for config in configs {
        let mut maps = BTreeMap::new();
        let textures = config.get("textures");
        for (role, candidates) in slots {
            let path = candidates.iter().find_map(|slot| {
                textures
                    .and_then(|value| value.get(slot))
                    .and_then(KaValue::as_str)
                    .filter(|path| !path.is_empty())
            });
            if let Some(path) = path {
                maps.insert(role.to_owned(), path.to_owned());
            }
        }
        let available = maps.len();
        if available == slots.len() {
            selected = Some((available, maps, config));
            break;
        }
        if selected
            .as_ref()
            .is_none_or(|(best, _, _)| available > *best)
        {
            selected = Some((available, maps, config));
        }
    }
    let Some((_, maps, config)) = selected else {
        return Ok(MaterialInfo {
            maps: slots
                .into_iter()
                .map(|(role, _)| (role.to_owned(), format!("@fallback/{role}")))
                .collect(),
            alpha_test: false,
            alpha_threshold: 0.35,
        });
    };
    let threshold = config
        .get("properties")
        .and_then(|value| value.get("alphatestThreshold"))
        .and_then(KaValue::as_bytes)
        .and_then(|bytes| bytes.get(bytes.len().saturating_sub(4)..))
        .filter(|bytes| bytes.len() == 4)
        .map(|bytes| f32::from_le_bytes(bytes.try_into().unwrap()))
        .filter(|value| value.is_finite())
        .unwrap_or(0.35);
    let alpha_test = config
        .get("enabledPresets")
        .and_then(|value| value.get("AlphaTest"))
        .is_some_and(KaValue::as_bool);
    let mut maps = maps;
    for (role, _) in slots {
        maps.entry(role.to_owned())
            .or_insert_with(|| format!("@fallback/{role}"));
    }
    Ok(MaterialInfo {
        maps,
        alpha_test,
        alpha_threshold: threshold,
    })
}

fn build_material_json(
    texture_sets: &BTreeSet<BTreeMap<String, String>>,
    bindings: &[Binding],
    selected_maps: &[BTreeMap<String, String>],
    texture_format: &str,
    include_pbr: bool,
) -> Vec<Json> {
    texture_sets
        .iter()
        .enumerate()
        .map(|(id, maps)| {
            let binding_index = selected_maps
                .iter()
                .position(|selected| selected == maps)
                .unwrap_or(0);
            let binding = &bindings[binding_index];
            let mut map_json = JsonMap::new();
            for (role, path) in maps {
                let fallback = path.starts_with("@fallback/");
                let folder = if role == "baseColor" { "base" } else { "pbr" };
                let file = if fallback {
                    format!("textures/{folder}/__fallback_{role}.png")
                } else {
                    let basename = Path::new(path)
                        .file_name()
                        .and_then(|name| name.to_str())
                        .unwrap_or(path);
                    let stem = Path::new(basename)
                        .file_stem()
                        .and_then(|name| name.to_str())
                        .unwrap_or(basename);
                    format!("textures/{folder}/{stem}.{texture_format}")
                };
                map_json.insert(role.clone(), serde_json::json!({
                    "materialPath": if fallback { Json::Null } else { Json::String(path.clone()) },
                    "file": file,
                    "fallback": fallback,
                }));
            }
            let mut item = serde_json::json!({
                "id": id,
                "maps": map_json,
                "alphaTest": binding.material.alpha_test,
                "alphaThreshold": binding.material.alpha_threshold,
            });
            if include_pbr {
                item["normalEncoding"] = Json::String("DXT5nm AG".to_owned());
                item["roughnessMetallicEncoding"] =
                    Json::String("DXT5nm AG (roughness, metallic)".to_owned());
                item["miscEncoding"] =
                    Json::String("DXT5nm AG (ambient occlusion, emissive)".to_owned());
            }
            item
        })
        .collect()
}

fn planar_uvs(positions: &[Vec3]) -> Vec<Vec2> {
    let mut minimum = [f32::INFINITY; 3];
    let mut maximum = [f32::NEG_INFINITY; 3];
    for position in positions {
        for axis in 0..3 {
            minimum[axis] = minimum[axis].min(position[axis]);
            maximum[axis] = maximum[axis].max(position[axis]);
        }
    }
    let mut axes = [0, 1, 2];
    axes.sort_by(|left, right| {
        (maximum[*right] - minimum[*right]).total_cmp(&(maximum[*left] - minimum[*left]))
    });
    let spans = [
        (maximum[axes[0]] - minimum[axes[0]]).max(1e-12),
        (maximum[axes[1]] - minimum[axes[1]]).max(1e-12),
    ];
    positions
        .iter()
        .map(|position| {
            [
                (position[axes[0]] - minimum[axes[0]]) / spans[0],
                (position[axes[1]] - minimum[axes[1]]) / spans[1],
            ]
        })
        .collect()
}

fn fallback_normals(positions: &[Vec3], indices: &[u32]) -> Vec<Vec3> {
    let mut sums = vec![[0.0; 3]; positions.len()];
    for triangle in indices.chunks_exact(3) {
        let a = positions[triangle[0] as usize];
        let b = positions[triangle[1] as usize];
        let c = positions[triangle[2] as usize];
        let face = cross3(sub3(b, a), sub3(c, a));
        for index in triangle {
            for axis in 0..3 {
                sums[*index as usize][axis] += face[axis];
            }
        }
    }
    sums.into_iter()
        .map(|normal| {
            let length = length3(normal);
            if length > 1e-15 {
                scale3(normal, 1.0 / length)
            } else {
                [0.0, 0.0, 1.0]
            }
        })
        .collect()
}

fn fallback_tangent_basis(
    positions: &[Vec3],
    normals: &[Vec3],
    uvs: &[Vec2],
    indices: &[u32],
) -> (Vec<Vec3>, Vec<Vec3>) {
    let mut tangent_sums = vec![[0.0; 3]; positions.len()];
    let mut binormal_sums = vec![[0.0; 3]; positions.len()];
    for triangle in indices.chunks_exact(3) {
        let [ia, ib, ic] = [
            triangle[0] as usize,
            triangle[1] as usize,
            triangle[2] as usize,
        ];
        let ab = sub3(positions[ib], positions[ia]);
        let ac = sub3(positions[ic], positions[ia]);
        let uv_ab = [uvs[ib][0] - uvs[ia][0], uvs[ib][1] - uvs[ia][1]];
        let uv_ac = [uvs[ic][0] - uvs[ia][0], uvs[ic][1] - uvs[ia][1]];
        let determinant = uv_ab[0] * uv_ac[1] - uv_ab[1] * uv_ac[0];
        if determinant.abs() <= 1e-12 {
            continue;
        }
        let inverse = determinant.recip();
        let tangent = scale3(sub3(scale3(ab, uv_ac[1]), scale3(ac, uv_ab[1])), inverse);
        let binormal = scale3(sub3(scale3(ac, uv_ab[0]), scale3(ab, uv_ac[0])), inverse);
        for index in [ia, ib, ic] {
            tangent_sums[index] = add3(tangent_sums[index], tangent);
            binormal_sums[index] = add3(binormal_sums[index], binormal);
        }
    }
    let mut tangents = Vec::with_capacity(positions.len());
    let mut binormals = Vec::with_capacity(positions.len());
    for (index, normal) in normals.iter().enumerate() {
        let mut tangent = sub3(
            tangent_sums[index],
            scale3(*normal, dot3(*normal, tangent_sums[index])),
        );
        let mut tangent_length = length3(tangent);
        if tangent_length <= 1e-15 {
            let axis = if normal[2].abs() < 0.9 {
                [0.0, 0.0, 1.0]
            } else {
                [0.0, 1.0, 0.0]
            };
            tangent = cross3(axis, *normal);
            tangent_length = length3(tangent).max(1.0);
        }
        tangent = scale3(tangent, tangent_length.recip());
        let cross = cross3(*normal, tangent);
        let handedness = if dot3(cross, binormal_sums[index]) < 0.0 {
            -1.0
        } else {
            1.0
        };
        tangents.push(tangent);
        binormals.push(scale3(cross, handedness));
    }
    (tangents, binormals)
}

pub(super) fn transform_position(position: Vec3, transform: &Transform) -> Result<Vec3, String> {
    let scaled = [
        position[0] * transform.scale[0],
        position[1] * transform.scale[1],
        position[2] * transform.scale[2],
    ];
    let rotated = rotate(scaled, transform.rotation, &transform.node)?;
    Ok(add3(rotated, transform.translation))
}

pub(super) fn transform_normal(normal: Vec3, transform: &Transform) -> Result<Vec3, String> {
    if transform.scale.iter().any(|scale| scale.abs() <= 1e-15) {
        return Err(format!("zero scale on {}", transform.node));
    }
    let inverse_scaled = [
        normal[0] / transform.scale[0],
        normal[1] / transform.scale[1],
        normal[2] / transform.scale[2],
    ];
    let rotated = rotate(inverse_scaled, transform.rotation, &transform.node)?;
    let length = length3(rotated);
    if length <= 1e-15 {
        return Err(format!("zero-length normal on {}", transform.node));
    }
    Ok(scale3(rotated, length.recip()))
}

fn transform_direction(direction: Vec3, transform: &Transform) -> Result<Vec3, String> {
    let origin = transform_position([0.0; 3], transform)?;
    let endpoint = transform_position(direction, transform)?;
    let result = sub3(endpoint, origin);
    let length = length3(result);
    if length <= 1e-15 || !length.is_finite() {
        return Err(format!("invalid tangent direction on {}", transform.node));
    }
    Ok(scale3(result, length.recip()))
}

fn rotate(value: Vec3, quaternion: [f32; 4], node: &str) -> Result<Vec3, String> {
    let qnorm = quaternion
        .iter()
        .map(|component| component * component)
        .sum::<f32>()
        .sqrt();
    if qnorm <= 1e-15 {
        return Err(format!("zero-length quaternion on {node}"));
    }
    let [qx, qy, qz, qw] = quaternion.map(|component| component / qnorm);
    let [x, y, z] = value;
    let tx = 2.0 * (qy * z - qz * y);
    let ty = 2.0 * (qz * x - qx * z);
    let tz = 2.0 * (qx * y - qy * x);
    Ok([
        x + qw * tx + qy * tz - qz * ty,
        y + qw * ty + qz * tx - qx * tz,
        z + qw * tz + qx * ty - qy * tx,
    ])
}

pub(super) fn world_matrix(transform: &Transform) -> Result<[f32; 16], String> {
    let qnorm = transform
        .rotation
        .iter()
        .map(|component| component * component)
        .sum::<f32>()
        .sqrt();
    if qnorm <= 1e-15 {
        return Err(format!("zero-length quaternion on {}", transform.node));
    }
    let [qx, qy, qz, qw] = transform.rotation.map(|component| component / qnorm);
    let [sx, sy, sz] = transform.scale;
    let [tx, ty, tz] = transform.translation;
    Ok([
        (1.0 - 2.0 * (qy * qy + qz * qz)) * sx,
        2.0 * (qx * qy + qz * qw) * sx,
        2.0 * (qx * qz - qy * qw) * sx,
        0.0,
        2.0 * (qx * qy - qz * qw) * sy,
        (1.0 - 2.0 * (qx * qx + qz * qz)) * sy,
        2.0 * (qy * qz + qx * qw) * sy,
        0.0,
        2.0 * (qx * qz + qy * qw) * sz,
        2.0 * (qy * qz - qx * qw) * sz,
        (1.0 - 2.0 * (qx * qx + qy * qy)) * sz,
        0.0,
        tx,
        ty,
        tz,
        1.0,
    ])
}

fn add3(a: Vec3, b: Vec3) -> Vec3 {
    [a[0] + b[0], a[1] + b[1], a[2] + b[2]]
}
fn sub3(a: Vec3, b: Vec3) -> Vec3 {
    [a[0] - b[0], a[1] - b[1], a[2] - b[2]]
}
fn scale3(value: Vec3, scale: f32) -> Vec3 {
    [value[0] * scale, value[1] * scale, value[2] * scale]
}
fn dot3(a: Vec3, b: Vec3) -> f32 {
    a[0] * b[0] + a[1] * b[1] + a[2] * b[2]
}
fn cross3(a: Vec3, b: Vec3) -> Vec3 {
    [
        a[1] * b[2] - a[2] * b[1],
        a[2] * b[0] - a[0] * b[2],
        a[0] * b[1] - a[1] * b[0],
    ]
}
fn length3(value: Vec3) -> f32 {
    dot3(value, value).sqrt()
}

fn align_buffer(buffer: &mut Vec<u8>, alignment: usize) -> usize {
    let padding = (alignment - buffer.len() % alignment) % alignment;
    buffer.resize(buffer.len() + padding, 0);
    buffer.len()
}

fn array_descriptor(kind: &str, offset: usize, count: usize) -> Json {
    serde_json::json!({ "type": kind, "byteOffset": offset, "count": count })
}

fn push_f32_array(
    buffer: &mut Vec<u8>,
    descriptors: &mut JsonMap<String, Json>,
    name: &str,
    values: &[f32],
) {
    let offset = align_buffer(buffer, std::mem::align_of::<f32>());
    for value in values {
        buffer.extend_from_slice(&value.to_le_bytes());
    }
    descriptors.insert(
        name.to_owned(),
        array_descriptor("float32", offset, values.len()),
    );
}

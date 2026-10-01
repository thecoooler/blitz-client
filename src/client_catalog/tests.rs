use crate::{discover_client_catalog, discover_client_tanks, list_client_tanks};
use serde_json::json;
use std::fs;
use std::path::PathBuf;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::time::{SystemTime, UNIX_EPOCH};

static FIXTURE_SEQUENCE: AtomicUsize = AtomicUsize::new(0);

struct ClientFixture {
    root: PathBuf,
    data: PathBuf,
}

impl ClientFixture {
    fn new() -> Self {
        let tick = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_nanos();
        let sequence = FIXTURE_SEQUENCE.fetch_add(1, Ordering::Relaxed);
        let root = std::env::temp_dir().join(format!(
            "blitz-catalog-test-{}-{tick}-{sequence}",
            std::process::id()
        ));
        let data = root.join("Data");
        fs::create_dir_all(data.join("XML/item_defs/vehicles")).unwrap();
        Self { root, data }
    }

    fn write(&self, relative: &str, bytes: impl AsRef<[u8]>) -> PathBuf {
        let path = self.data.join(relative);
        fs::create_dir_all(path.parent().unwrap()).unwrap();
        fs::write(&path, bytes).unwrap();
        path
    }

    fn write_dvpl(&self, relative: &str, bytes: impl AsRef<[u8]>) -> PathBuf {
        let encoded =
            crate::dvpl::encode(bytes.as_ref(), crate::dvpl::ContainerKind::Dvpl(2)).unwrap();
        self.write(&format!("{relative}.dvpl"), encoded)
    }

    fn list(&self, nation: &str, entries: &str) -> PathBuf {
        self.write(
            &format!("XML/item_defs/vehicles/{nation}/list.xml"),
            format!("<root>{entries}</root>"),
        )
    }

    fn pair(&self, folder: &str, id: &str) {
        // Discovery checks existence, not the contents of scene/geometry files.
        self.write(&format!("3d/Tanks/{folder}/{id}.sc2"), []);
        self.write(&format!("3d/Tanks/{folder}/{id}.scg"), []);
    }

    fn maps(&self) {
        self.write(
            "maps.yaml",
            "maps:\n    arena:\n        id: 7\n        localName: NAME\n        frame: 4\n        availableModes: [1, 2]\n        availableInTrainingRoom: true\n",
        );
    }
}

impl Drop for ClientFixture {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.root);
    }
}

fn complete_tank(name: &str, id: &str) -> String {
    format!(
        "<{name}>\
         <id>{id}</id><userString>#vehicles:{name}</userString>\
         <shortUserString>#short:{name}</shortUserString>\
         <description>#description:{name}</description>\
         <price>100<gold/></price><sellPrice>50</sellPrice>\
         <notInShop>true</notInShop><enrichmentPermanentCost>20</enrichmentPermanentCost>\
         <tags>lightTank regular</tags><level>3</level>\
         <combatRole><default>support</default></combatRole>\
         <configurationModes>default special</configurationModes>\
         <leasing>30.09.2026 12:34</leasing></{name}>"
    )
}

#[test]
fn models_do_not_require_catalog_fields_or_optional_display_fields() {
    let fixture = ClientFixture::new();
    let list_path = fixture.list(
        "usa",
        "<Sparse_Tank/><Empty_Fields><userString> </userString><level/></Empty_Fields><Unpaired/>",
    );
    fixture.pair("USA", "Sparse_Tank");
    fixture.pair("USA", "Empty_Fields");

    let models = discover_client_tanks(&fixture.data).unwrap();
    assert_eq!(models.len(), 2);
    assert_eq!(models[0].id, "Sparse_Tank");
    assert_eq!(models[0].name, "Sparse Tank");
    assert_eq!(models[1].name, "Empty Fields");
    for model in models {
        assert_eq!(model.level, None);
        assert_eq!(model.user_string, None);
    }
    let error = discover_client_catalog(&fixture.data).unwrap_err();
    assert_eq!(error.code, "CLIENT_XML_INVALID");
    assert_eq!(error.path, Some(list_path));
}

#[test]
fn unmodeled_invalid_metadata_does_not_block_model_listing() {
    let fixture = ClientFixture::new();
    let list_path = fixture.list(
        "usa",
        &(complete_tank("Modeled", "42")
            + "<Unmodeled><userString>unknown</userString><id>43</id><price>invalid</price></Unmodeled>"),
    );
    fixture.pair("USA", "Modeled");

    let models = list_client_tanks(Some(fixture.data.clone())).unwrap();
    assert_eq!(models.len(), 1);
    assert_eq!(models[0].id, "Modeled");
    assert_eq!(models[0].name, "Modeled");
    assert_eq!(models[0].user_string.as_deref(), Some("#vehicles:Modeled"));
    let error = discover_client_catalog(&fixture.data).unwrap_err();
    assert_eq!(error.code, "CLIENT_XML_INVALID");
    assert_eq!(error.path, Some(list_path));
    assert!(error.message.contains("invalid price"));
}

#[test]
fn only_complete_pairs_are_discovered_from_plain_and_dvpl_resources() {
    for compressed_list in [false, true] {
        let fixture = ClientFixture::new();
        let xml = "<root><First><userString>#first</userString><level>4</level></First><Second/><Scene_Only/><Geometry_Only/></root>";
        let strings = "\"#first\": \"Первый\"\n";
        if compressed_list {
            fixture.write_dvpl("XML/item_defs/vehicles/usa/list.xml", xml);
            fixture.write("Strings/ru.yaml", strings);
        } else {
            fixture.write("XML/item_defs/vehicles/usa/list.xml", xml);
            fixture.write_dvpl("Strings/ru.yaml", strings);
        }
        let first_scene = fixture.write("3d/Tanks/USA/First.sc2", []);
        let first_geometry = fixture.write_dvpl("3d/Tanks/USA/First.scg", []);
        let second_scene = fixture.write_dvpl("3d/Tanks/USA/Second.sc2", []);
        let second_geometry = fixture.write("3d/Tanks/USA/Second.scg", []);
        fixture.write("3d/Tanks/USA/Scene_Only.sc2", []);
        fixture.write("3d/Tanks/USA/Geometry_Only.scg", []);

        let models = discover_client_tanks(&fixture.data).unwrap();
        assert_eq!(models.len(), 2);
        assert_eq!(models[0].name, "Первый");
        assert_eq!(models[0].level.as_deref(), Some("4"));
        assert_eq!(models[0].model_files.scene, first_scene);
        assert_eq!(models[0].model_files.geometry, first_geometry);
        assert_eq!(models[1].model_files.scene, second_scene);
        assert_eq!(models[1].model_files.geometry, second_geometry);
    }
}

#[test]
fn catalog_preserves_schema_localization_and_distinct_model_ids() {
    let fixture = ClientFixture::new();
    fixture.list("usa", &complete_tank("Model_Key", "42"));
    fixture.pair("USA", "Model_Key");
    fixture.maps();
    fixture.write_dvpl(
        "Strings/ru.yaml",
        "\"#vehicles:Model_Key\": \"Танк\"\n\"#short:Model_Key\": \"Т\"\n\"#description:Model_Key\": \"Описание\"\n\"#maps:arena:NAME\": \"Карта\"\n",
    );

    let catalog = discover_client_catalog(&fixture.data).unwrap();
    assert_eq!(catalog.tanks.len(), 1);
    assert_eq!(catalog.maps.len(), 1);
    assert_eq!(
        serde_json::to_value(&catalog.tanks[0]).unwrap(),
        json!({
            "id": "42", "userString": "Танк", "shortUserString": "Т",
            "description": "Описание", "price": {"value": 100, "is_gold": true},
            "sellPrice": {"value": 50, "is_gold": false}, "notInShop": true,
            "enrichmentPermanentCost": 20, "tags": ["lightTank", "regular"],
            "level": 3, "nation": "usa", "combatRole": {"default": "support"},
            "configurationModes": ["default", "special"], "leasing": "2026-09-30T12:34:00"
        })
    );
    assert_eq!(
        serde_json::to_value(&catalog.maps[0]).unwrap(),
        json!({
            "id": 7, "localName": "Карта", "availableInTrainingRoom": true,
            "availableModes": [1, 2], "spriteFrame": 4
        })
    );
    let models = discover_client_tanks(&fixture.data).unwrap();
    assert_eq!(models.len(), 1);
    assert_eq!(models[0].id, "Model_Key");
    assert_eq!(models[0].name, "Танк");
    assert_eq!(
        models[0].user_string.as_deref(),
        Some("#vehicles:Model_Key")
    );
    assert_eq!(
        serde_json::to_value(&catalog.model_tanks).unwrap(),
        serde_json::to_value(models).unwrap()
    );
}

#[test]
fn models_preserve_legacy_nation_order_and_xml_order() {
    let fixture = ClientFixture::new();
    fixture.write(
        "available_nations.yaml",
        "available_nations:\n  - ussr\n  - usa\n  - unknown\n",
    );
    fixture.list(
        "ussr",
        &(complete_tank("RU_Second", "1") + &complete_tank("RU_First", "2")),
    );
    fixture.list(
        "usa",
        &(complete_tank("US_Second", "3") + &complete_tank("US_First", "4")),
    );
    fixture.list("unknown", &complete_tank("Unknown", "5"));
    for (folder, id) in [
        ("USSR", "RU_Second"),
        ("USSR", "RU_First"),
        ("USA", "US_Second"),
        ("USA", "US_First"),
        ("unknown", "Unknown"),
    ] {
        fixture.pair(folder, id);
    }
    fixture.maps();

    let catalog = discover_client_catalog(&fixture.data).unwrap();
    assert_eq!(
        catalog
            .tanks
            .iter()
            .map(|tank| tank.id.as_str())
            .collect::<Vec<_>>(),
        ["1", "2", "3", "4", "5"]
    );
    let models = discover_client_tanks(&fixture.data).unwrap();
    assert_eq!(
        models
            .iter()
            .map(|tank| tank.id.as_str())
            .collect::<Vec<_>>(),
        ["US_Second", "US_First", "RU_Second", "RU_First"]
    );
    assert_eq!(
        serde_json::to_value(catalog.model_tanks).unwrap(),
        serde_json::to_value(models).unwrap()
    );
}

#[test]
fn full_catalog_still_rejects_invalid_optional_and_required_metadata() {
    let valid = complete_tank("Tank", "42");
    for (from, to) in [
        ("<id>42</id>", ""),
        ("<userString>#vehicles:Tank</userString>", ""),
        ("<price>100<gold/></price>", "<price>bad</price>"),
        (
            "<enrichmentPermanentCost>20</enrichmentPermanentCost>",
            "<enrichmentPermanentCost>bad</enrichmentPermanentCost>",
        ),
        ("<level>3</level>", "<level>bad</level>"),
        ("<tags>lightTank regular</tags>", "<tags/>"),
        (
            "<configurationModes>default special</configurationModes>",
            "<configurationModes/>",
        ),
        ("<sellPrice>50</sellPrice>", "<sellPrice>bad</sellPrice>"),
        (
            "<leasing>30.09.2026 12:34</leasing>",
            "<leasing>bad</leasing>",
        ),
    ] {
        let fixture = ClientFixture::new();
        let list_path = fixture.list("usa", &valid.replace(from, to));
        fixture.pair("USA", "Tank");
        assert_eq!(
            discover_client_tanks(&fixture.data).unwrap().len(),
            1,
            "model discovery for {from}"
        );
        let error = discover_client_catalog(&fixture.data).unwrap_err();
        assert_eq!(error.code, "CLIENT_XML_INVALID", "full catalog for {from}");
        assert_eq!(error.path, Some(list_path));
    }
}

#[test]
fn model_discovery_keeps_xml_and_dvpl_errors() {
    let fixture = ClientFixture::new();
    let path = fixture.write("XML/item_defs/vehicles/usa/list.xml", "<root><Tank></root>");
    for error in [
        discover_client_tanks(&fixture.data).unwrap_err(),
        discover_client_catalog(&fixture.data).unwrap_err(),
    ] {
        assert_eq!(error.code, "CLIENT_XML_INVALID");
        assert_eq!(error.path.as_ref(), Some(&path));
    }

    let fixture = ClientFixture::new();
    let path = fixture.write_dvpl("XML/item_defs/vehicles/usa/list.xml", "<root/>");
    let mut bytes = fs::read(&path).unwrap();
    bytes[0] ^= 0xff;
    fs::write(&path, bytes).unwrap();
    for error in [
        discover_client_tanks(&fixture.data).unwrap_err(),
        discover_client_catalog(&fixture.data).unwrap_err(),
    ] {
        assert_eq!(error.code, "DVPL_CRC_FAILED");
        assert_eq!(error.path.as_ref(), Some(&path));
    }
}

#[test]
fn model_discovery_does_not_require_maps() {
    let fixture = ClientFixture::new();
    fixture.list("usa", &complete_tank("Tank", "42"));
    fixture.pair("USA", "Tank");
    assert_eq!(discover_client_tanks(&fixture.data).unwrap().len(), 1);
    let error = discover_client_catalog(&fixture.data).unwrap_err();
    assert_eq!(error.code, "CLIENT_DATA_INVALID");
    assert_eq!(error.path, Some(fixture.data.clone()));
    fixture.maps();
    assert_eq!(
        discover_client_catalog(&fixture.data).unwrap().tanks.len(),
        1
    );
}

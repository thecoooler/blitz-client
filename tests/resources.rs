#![cfg(feature = "resources")]

use blitz_client::resources::{ContainerKind, decode_dvpl, encode_dvpl};

#[test]
fn codecs_are_reusable_without_client_discovery() {
    let plain = b"resource payload, resource payload, resource payload";
    for kind in [
        ContainerKind::Plain,
        ContainerKind::Dvpl(0),
        ContainerKind::Dvpl(1),
        ContainerKind::Dvpl(2),
        ContainerKind::Dvpl(3),
    ] {
        let encoded = encode_dvpl(plain, kind).unwrap();
        let (decoded, detected) = decode_dvpl(&encoded).unwrap();
        assert_eq!(decoded, plain);
        assert_eq!(detected, kind);
        if kind != ContainerKind::Plain {
            let mut damaged = encoded;
            damaged[0] ^= 1;
            let error: blitz_client::InstallerError = decode_dvpl(&damaged).unwrap_err();
            assert_eq!(error.code, "DVPL_CRC_FAILED");
        }
    }
}

#[test]
fn xml_parser_is_reusable_without_client_catalog() {
    use blitz_client::resources::xml::{child_text, parse_xml};
    let root = parse_xml(b"<root><name>Tanks &amp; Blitz</name></root>").unwrap();
    assert_eq!(child_text(&root, "name").as_deref(), Some("Tanks & Blitz"));
}

package xml

import "testing"

func TestParserIsReusableWithoutClientCatalog(t *testing.T) {
	root, err := Parse([]byte("<root><name>Tanks &amp; Blitz</name></root>"))
	if err != nil {
		t.Fatal(err)
	}
	if text, ok := ChildText(root, "name"); !ok || text != "Tanks & Blitz" {
		t.Fatalf("%q %v", text, ok)
	}
}

func TestParsesCommentsCDATAAndRejectsMalformedInput(t *testing.T) {
	root, err := Parse([]byte("\xef\xbb\xbf<?xml version=\"1.0\"?><!-- c --><a x='>'><b/><c><![CDATA[<x>]]> &#x41;&#66;</c></a>"))
	if err != nil {
		t.Fatal(err)
	}
	if len(root.Children) != 2 || root.Children[1].Text != "<x> AB" {
		t.Fatalf("%+v", root.Children)
	}
	for _, input := range []string{"<a>", "<a></b>", "</a>", "<a/><b/>", "text", "<a>&bad;</a>", "<a>&#xD800;</a>", "<>", "<a"} {
		if _, err := Parse([]byte(input)); err == nil {
			t.Fatalf("%q parsed", input)
		}
	}
}

func TestAttributesAndRepeatedChildren(t *testing.T) {
	root, err := Parse([]byte(`<root><items path="common.xml"/><items path='mode&amp;test.xml'/><factor>1</factor><factor>2</factor><empty/></root>`))
	if err != nil {
		t.Fatal(err)
	}
	if len(root.Children) != 5 || root.Children[0].Attributes["path"] != "common.xml" || root.Children[1].Attributes["path"] != "mode&test.xml" || root.Children[2].Text != "1" || root.Children[3].Text != "2" || root.Children[4].Text != "" {
		t.Fatalf("attributes/order/scalars lost: %+v", root)
	}
}

func TestMalformedAttributesDoNotFailParsing(t *testing.T) {
	root, err := Parse([]byte(`<root><a x=1 y="&nbsp;&" x="2"/><b =broken path="kept.xml"/><c/></root>`))
	if err != nil {
		t.Fatal(err)
	}
	if len(root.Children) != 3 || root.Children[0].Attributes["x"] != "1" || root.Children[0].Attributes["y"] != "\u00a0&" || root.Children[1].Attributes["path"] != "kept.xml" {
		t.Fatalf("permissive attributes lost: %+v", root.Children[0])
	}
}

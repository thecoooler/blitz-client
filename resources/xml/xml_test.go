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

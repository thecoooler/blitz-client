package mods

import "testing"

func TestNormalizedBlocksKeepLeadingAndInternalWhitespace(t *testing.T) {
	find := "    key: value\n"
	if len(findMatches("    key: value \t\r\n", find, true)) != 1 {
		t.Fatal("normalized match")
	}
	for _, source := range []string{"        key: value\n", "    key:  value\n", "    Key: value\n"} {
		if len(findMatches(source, find, true)) != 0 {
			t.Fatal(source)
		}
	}
	if len(findMatches("key: value", "key: ", true)) != 0 {
		t.Fatal("trailing space inside token")
	}
}

func TestNormalizedRangesPreserveUnicodeAndTrailingBytesAtEOF(t *testing.T) {
	source := "сосед:  \r\n    ключ: значение \t"
	ranges := findMatches(source, "    ключ: значение", true)
	if len(ranges) != 1 || source[ranges[0].start:ranges[0].end] != "    ключ: значение \t" {
		t.Fatal(ranges)
	}
	if len(findMatches(source, "    ключ: значение\n", true)) != 0 {
		t.Fatal("matched newline at EOF")
	}
}

func TestExactMatchesDoNotOverlap(t *testing.T) {
	if got := findMatches("aaaa", "aa", false); len(got) != 2 {
		t.Fatal(got)
	}
}

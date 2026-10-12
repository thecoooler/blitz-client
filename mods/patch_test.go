package mods

import "testing"

func TestSplitsUnifiedHeadersWithoutConfusingRemovedContent(t *testing.T) {
	diff := "--- a/one.yaml\n+++ b/one.yaml\n@@ -1 +1 @@\n--- old\n+++ new\n--- a/two.yaml\n+++ b/two.yaml\n@@ -1 +1 @@\n-old\n+new\n"
	sections, err := splitPatch(diff)
	if err != nil {
		t.Fatal(err)
	}
	if len(sections) != 2 || sections[0].path != "one.yaml" || sections[1].path != "two.yaml" {
		t.Fatalf("%+v", sections)
	}
	if got, err := applyUnifiedDiff("-- old\n", sections[0].diff); err != nil || got != "++ new\n" {
		t.Fatal(got, err)
	}
}

func TestDecodesGitQuotedPathsAndPreservesMissingFinalNewline(t *testing.T) {
	diff := `diff --git "a/space \303\251.yaml" "b/space \303\251.yaml"
--- "a/space \303\251.yaml"
+++ "b/space \303\251.yaml"
@@ -1 +1 @@
-old
\ No newline at end of file
+new
\ No newline at end of file
`
	sections, err := splitPatch(diff)
	if err != nil {
		t.Fatal(err)
	}
	if sections[0].path != "space é.yaml" {
		t.Fatal(sections[0].path)
	}
	if got, err := applyUnifiedDiff("old", sections[0].diff); err != nil || got != "new" {
		t.Fatal(got, err)
	}
}

func TestLaterHunksFollowInsertionsWhenContextIsRepeated(t *testing.T) {
	diff := "--- a/file\n+++ b/file\n@@ -1 +1,2 @@\n a\n+inserted\n@@ -3 +4 @@\n-old\n+new\n"
	sections, err := splitPatch(diff)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := applyUnifiedDiff("a\nold\nold\n", sections[0].diff); err != nil || got != "a\ninserted\nold\nnew\n" {
		t.Fatalf("%q %v", got, err)
	}
}

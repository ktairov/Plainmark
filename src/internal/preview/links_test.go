package preview

import "testing"

func TestResolveLinks(t *testing.T) {
	for _, tc := range []struct{ href, want, anchor string }{
		{"../Welcome.md", "Welcome.md", ""}, {"/Folder/Linked.md#раздел", "Folder/Linked.md", "раздел"}, {"../Some%20note.md?x=1#heading", "Some note.md", "heading"}, {"#here", "", "here"},
	} {
		p, a, _, err := ResolveLink("Folder/Linked.md", tc.href)
		if err != nil || p != tc.want || a != tc.anchor {
			t.Fatalf("%s -> %q %q %v", tc.href, p, a, err)
		}
	}
	for _, href := range []string{"../../outside.md", "javascript:alert(1)", "//other/file.md", "broken%xx.md"} {
		if _, _, _, err := ResolveLink("Folder/Linked.md", href); err == nil {
			t.Fatal("accepted", href)
		}
	}
}

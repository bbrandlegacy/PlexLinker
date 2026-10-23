package browser

import "testing"

func TestParseHTMLDirectorySupportsNestedMarkupAndEntities(t *testing.T) {
	t.Parallel()
	html := `<table>
<tr><td><a href="Season%201/"><strong>Season &amp; One/</strong></a></td><td>2026-10-22 12:30</td></tr>
<tr><td><a class="file" href="Episode%20One.mkv"><span>Episode &amp; One.mkv</span></a></td><td>2026-10-23 09:15</td></tr>
<tr><td><a href="notes.txt">notes.txt</a></td></tr>
</table>`
	listing, err := ParseHTMLDirectory(html, "Shows")
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Items) != 2 {
		t.Fatalf("items = %#v", listing.Items)
	}
	if listing.Items[0].Name != "Season & One" || listing.Items[0].Path != "Shows/Season 1" || listing.Items[0].Type != TypeDirectory {
		t.Fatalf("directory = %#v", listing.Items[0])
	}
	if listing.Items[1].Name != "Episode & One.mkv" || listing.Items[1].Path != "Shows/Episode One.mkv" || listing.Items[1].ModifiedTime.IsZero() {
		t.Fatalf("file = %#v", listing.Items[1])
	}
}

func TestParseHTMLDirectorySupportsPreListings(t *testing.T) {
	t.Parallel()
	html := `<pre><a href="../">Parent Directory</a>
<a href="Movie.2026.mp4">Movie.2026.mp4</a> 23-Oct-2026 10:20
<a href="Music/">Music/</a></pre>`
	listing, err := ParseHTMLDirectoryWithSort(html, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Items) != 2 || listing.Items[0].Type != TypeFile || listing.Items[1].Type != TypeDirectory {
		t.Fatalf("items = %#v", listing.Items)
	}
}

func TestParseHTMLDirectoryRejectsExternalAndTraversalLinks(t *testing.T) {
	t.Parallel()
	html := `<a href="//evil.example/file.mkv">evil</a><a href="https://evil.example/file.mkv">evil2</a><a href="../../escape.mkv">escape</a><a href="safe.mkv">safe.mkv</a>`
	listing, err := ParseHTMLDirectory(html, "shows")
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Items) != 1 || listing.Items[0].Name != "safe.mkv" {
		t.Fatalf("items = %#v", listing.Items)
	}
}

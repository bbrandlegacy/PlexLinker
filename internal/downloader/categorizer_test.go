package downloader

import "testing"

func TestGetCategoryFromFilename(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want MediaCategory
	}{
		{"Show.S01E01.1080p.mkv", CategoryTVShows},
		{"Show.S01.Complete.mkv", CategoryTVShows},
		{"Show Season 2.mkv", CategoryTVShows},
		{"Show.2x03.mkv", CategoryTVShows},
		{"Show.Episode 7.mkv", CategoryTVShows},
		{"Movie.2026.1080p.mkv", CategoryMovies},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := GetCategory(test.name); got != test.want {
				t.Fatalf("GetCategory(%q) = %q, want %q", test.name, got, test.want)
			}
		})
	}
}

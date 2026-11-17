package vibe

import "testing"

func TestNormalizeSearch(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{name: "case and whitespace", query: "  DaRude\t Sandstorm  ", want: "darude sandstorm"},
		{name: "dangling separators", query: "darude sandstorm - ,", want: "darude sandstorm"},
		{name: "trailing comma", query: "darude sandstorm,", want: "darude sandstorm"},
		{name: "paste characters", query: "darude\u200b sandstorm\ufeff", want: "darude sandstorm"},
		{name: "artist punctuation", query: "P!nk - So What", want: "p!nk - so what"},
		{name: "slash and qualifier", query: "AC/DC live", want: "ac/dc live"},
		{name: "symbols", query: "C++ A#", want: "c++ a#"},
		{name: "attached punctuation", query: "Help!", want: "help!"},
		{name: "quoted phrase", query: "\"Song -\" remix", want: "\"song -\" remix"},
		{name: "punctuation artist", query: "!!!", want: "!!!"},
		{name: "version", query: "Song (live)", want: "song (live)"},
		{name: "official audio", query: "03 Greedo - I Can't Control Myself Ft. OHGEESY (Official Audio)", want: "03 greedo - i can't control myself ft. ohgeesy"},
		{name: "featured artist after label", query: "24kGoldn - Mood (Official Video) ft. iann dior", want: "24kgoldn - mood ft. iann dior"},
		{name: "remix preserved", query: "71Digits x Flo Rida - Low (LUNAX Remix) [Official Music Video]", want: "71digits x flo rida - low (lunax remix)"},
		{name: "quality label", query: "AC/DC - Back In Black (Official 4K Video)", want: "ac/dc - back in black"},
		{name: "vevo and official", query: "Artist - Song (VEVO) [ Official ] -", want: "artist - song"},
		{name: "version preserved", query: "2Pac ft. Dr. Dre - California Love (Official Video) [Full Length Version]", want: "2pac ft. dr. dre - california love [full length version]"},
		{name: "remaster preserved", query: "1979 (Remastered 2012)", want: "1979 (remastered 2012)"},
		{name: "mixed label preserved", query: "A$AP Rocky - L$D (Explicit - Official Video)", want: "a$ap rocky - l$d (explicit - official video)"},
		{name: "lyrics preserved", query: "Song (Official Lyric Video)", want: "song (official lyric video)"},
		{name: "quoted metadata preserved", query: "\"Song (Official)\"", want: "\"song (official)\""},
		{name: "unbracketed title preserved", query: "Official by ArtistVEVO", want: "official by artistvevo"},
		{name: "metadata only preserved", query: "(Official)", want: "(official)"},
		{name: "entities", query: "Earth &amp; Fire", want: "earth & fire"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeSearch(tt.query)
			if got != tt.want {
				t.Errorf("NormalizeSearch() = %q, want %q", got, tt.want)
			}
			if NormalizeSearch(got) != got {
				t.Errorf("NormalizeSearch() is not idempotent for %q", got)
			}
		})
	}
}

package main

import (
	"testing"
)

func BenchmarkCleanHTML(b *testing.B) {
	sampleHTML := `<div class="synopsis">Hello &quot;world&quot;! <br/> This is a <b>test</b> of HTML cleaning with &amp; special &lt;entities&gt; and ’quotes’.</div>`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cleanHTML(sampleHTML)
	}
}

func BenchmarkParseEpisodeNumber(b *testing.B) {
	eps := []string{"12", "12.5", "S01E05", "Movie", "Episode 24"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, ep := range eps {
			_ = parseEpisodeNumber(ep)
		}
	}
}

func BenchmarkParseJikanDuration(b *testing.B) {
	durations := []string{"24 min per ep", "1 hr 45 min", "2 hours", "45 s", "24m"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, d := range durations {
			_ = parseJikanDuration(d)
		}
	}
}

func BenchmarkFormatMediaTitle(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = formatMediaTitle("Sousou no Frieren", "12")
		_ = formatMediaTitle("Yani Neko", "S01E02")
	}
}

func BenchmarkFormatTime(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = formatTime(3725.5)
		_ = formatTime(145.0)
	}
}

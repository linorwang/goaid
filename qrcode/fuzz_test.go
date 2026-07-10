package qrcode_test

import (
	"testing"

	"github.com/linorwang/goaid/qrcode"
)

func FuzzGenerate(f *testing.F) {
	f.Add("https://example.com")
	f.Add("企业二维码")
	f.Add("")
	f.Fuzz(func(t *testing.T, content string) {
		if len(content) > 1024 {
			t.Skip()
		}
		_, _ = qrcode.Generate(content)
		_, _ = qrcode.Generate(content, qrcode.WithFormat(qrcode.FormatSVG))
	})
}

func BenchmarkGeneratePNG256(b *testing.B) {
	for b.Loop() {
		_, err := qrcode.Generate("https://example.com/benchmark")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateSVG512(b *testing.B) {
	for b.Loop() {
		_, err := qrcode.Generate(
			"https://example.com/benchmark",
			qrcode.WithFormat(qrcode.FormatSVG),
			qrcode.WithSize(512),
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

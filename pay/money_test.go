package pay

import "testing"

func TestYuanFromFen(t *testing.T) {
	tests := map[int64]string{
		1:     "0.01",
		10:    "0.10",
		1234:  "12.34",
		-1234: "-12.34",
	}
	for input, want := range tests {
		if got := YuanFromFen(input); got != want {
			t.Fatalf("YuanFromFen(%d) = %q, want %q", input, got, want)
		}
	}
}

func TestFenFromYuan(t *testing.T) {
	tests := map[string]int64{
		"0.01":  1,
		"0.1":   10,
		"12":    1200,
		"12.34": 1234,
	}
	for input, want := range tests {
		got, err := FenFromYuan(input)
		if err != nil {
			t.Fatalf("FenFromYuan(%q) returned error: %v", input, err)
		}
		if got != want {
			t.Fatalf("FenFromYuan(%q) = %d, want %d", input, got, want)
		}
	}
}

func TestFenFromYuanRejectsInvalidAmount(t *testing.T) {
	if _, err := FenFromYuan("1.001"); err == nil {
		t.Fatal("expected error")
	}
}

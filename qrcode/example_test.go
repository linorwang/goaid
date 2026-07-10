package qrcode_test

import (
	"fmt"

	"github.com/linorwang/goaid/qrcode"
)

func ExampleGenerate() {
	data, err := qrcode.Generate("https://example.com", qrcode.WithSize(256))
	fmt.Println(err == nil, len(data) > 0)
	// Output: true true
}

func ExampleWIFI() {
	payload, err := qrcode.WIFI(qrcode.WIFIConfig{
		SSID:       "Office-WIFI",
		Password:   "12345678",
		Encryption: qrcode.WPA2,
	})
	fmt.Println(payload, err)
	// Output: WIFI:T:WPA;S:Office-WIFI;P:12345678;H:false;; <nil>
}

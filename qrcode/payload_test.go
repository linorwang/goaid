package qrcode_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/linorwang/goaid/qrcode"
)

func TestStructuredPayloads(t *testing.T) {
	t.Parallel()

	urlPayload, err := qrcode.URL("https://example.com/path?q=二维码")
	if err != nil {
		t.Fatalf("URL() error = %v", err)
	}
	if !strings.HasPrefix(urlPayload, "https://example.com/path?") {
		t.Fatalf("URL() = %q", urlPayload)
	}

	wifi, err := qrcode.WIFI(qrcode.WIFIConfig{
		SSID:       `Office;WiFi`,
		Password:   `p,a:ss\\word`,
		Encryption: qrcode.WPA2,
		Hidden:     true,
	})
	if err != nil {
		t.Fatalf("WIFI() error = %v", err)
	}
	if wifi != `WIFI:T:WPA;S:Office\;WiFi;P:p\,a\:ss\\\\word;H:true;;` {
		t.Fatalf("WIFI() = %q", wifi)
	}

	vcard, err := qrcode.VCard(qrcode.VCardConfig{
		Name:    "张三",
		Phone:   "+86 13800000000",
		Email:   "zhangsan@example.com",
		Company: "Example, Inc.",
	})
	if err != nil {
		t.Fatalf("VCard() error = %v", err)
	}
	for _, expected := range []string{"BEGIN:VCARD\r\n", "VERSION:3.0\r\n", "FN:张三\r\n", "ORG:Example\\, Inc.\r\n", "END:VCARD\r\n"} {
		if !strings.Contains(vcard, expected) {
			t.Fatalf("VCard() missing %q:\n%s", expected, vcard)
		}
	}

	email, err := qrcode.Email(qrcode.EmailConfig{Address: "support@example.com", Subject: "问题反馈", Body: "您好"})
	if err != nil {
		t.Fatalf("Email() error = %v", err)
	}
	if !strings.HasPrefix(email, "mailto:support@example.com?") || !strings.Contains(email, "subject=") || !strings.Contains(email, "body=") {
		t.Fatalf("Email() = %q", email)
	}

	phone, err := qrcode.Phone("+86 13800000000")
	if err != nil {
		t.Fatalf("Phone() error = %v", err)
	}
	if phone != "tel:+86%2013800000000" {
		t.Fatalf("Phone() = %q", phone)
	}

	sms, err := qrcode.SMS("13800000000", "验证码为 123456")
	if err != nil {
		t.Fatalf("SMS() error = %v", err)
	}
	if !strings.HasPrefix(sms, "sms:13800000000?body=") {
		t.Fatalf("SMS() = %q", sms)
	}

	geo, err := qrcode.Geo(39.9042, 116.4074)
	if err != nil {
		t.Fatalf("Geo() error = %v", err)
	}
	if geo != "geo:39.9042,116.4074" {
		t.Fatalf("Geo() = %q", geo)
	}
}

func TestStructuredPayloadValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		call func() error
	}{
		{name: "relative URL", call: func() error { _, err := qrcode.URL("/relative"); return err }},
		{name: "empty SSID", call: func() error { _, err := qrcode.WiFi(qrcode.WiFiConfig{}); return err }},
		{name: "bad encryption", call: func() error { _, err := qrcode.WiFi(qrcode.WiFiConfig{SSID: "x", Encryption: "UNKNOWN"}); return err }},
		{name: "empty vcard", call: func() error { _, err := qrcode.VCard(qrcode.VCardConfig{}); return err }},
		{name: "bad email", call: func() error { _, err := qrcode.Email(qrcode.EmailConfig{Address: "not-email"}); return err }},
		{name: "bad phone", call: func() error { _, err := qrcode.Phone("call-me"); return err }},
		{name: "bad geo", call: func() error { _, err := qrcode.Geo(91, 0); return err }},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := test.call(); !errors.Is(err, qrcode.ErrInvalidPayload) {
				t.Fatalf("error = %v, want ErrInvalidPayload", err)
			}
		})
	}
}

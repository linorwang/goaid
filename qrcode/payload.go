package qrcode

import (
	"fmt"
	"net/mail"
	neturl "net/url"
	"strconv"
	"strings"
	"unicode"
)

// WiFiEncryption is the authentication value used by the Wi-Fi QR payload.
type WiFiEncryption string

const (
	Open WiFiEncryption = "nopass"
	WEP  WiFiEncryption = "WEP"
	WPA  WiFiEncryption = "WPA"
	WPA2 WiFiEncryption = "WPA"
)

type WiFiConfig struct {
	SSID       string
	Password   string
	Encryption WiFiEncryption
	Hidden     bool
}

// WIFIConfig is kept as an acronym-compatible alias for examples using WIFI.
type WIFIConfig = WiFiConfig

type VCardConfig struct {
	Name    string
	Phone   string
	Email   string
	Company string
	Title   string
	URL     string
	Address string
}

type EmailConfig struct {
	Address string
	Subject string
	Body    string
}

// URL validates an absolute URL and returns its canonical string form.
func URL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || containsControl(rawURL) {
		return "", fmt.Errorf("%w: URL is empty or contains control characters", ErrInvalidPayload)
	}
	parsed, err := neturl.ParseRequestURI(rawURL)
	if err != nil || !parsed.IsAbs() {
		return "", fmt.Errorf("%w: invalid absolute URL %q", ErrInvalidPayload, rawURL)
	}
	if (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host == "" {
		return "", fmt.Errorf("%w: URL host is empty", ErrInvalidPayload)
	}
	return parsed.String(), nil
}

func WiFi(config WiFiConfig) (string, error) {
	if config.SSID == "" || containsControl(config.SSID) {
		return "", fmt.Errorf("%w: Wi-Fi SSID is empty or invalid", ErrInvalidPayload)
	}
	encryption := config.Encryption
	if encryption == "" {
		encryption = WPA
	}
	if encryption != Open && encryption != WEP && encryption != WPA {
		return "", fmt.Errorf("%w: unsupported Wi-Fi encryption %q", ErrInvalidPayload, encryption)
	}
	if encryption != Open && config.Password == "" {
		return "", fmt.Errorf("%w: Wi-Fi password is required", ErrInvalidPayload)
	}
	if containsControl(config.Password) {
		return "", fmt.Errorf("%w: Wi-Fi password contains invalid control characters", ErrInvalidPayload)
	}
	return fmt.Sprintf(
		"WIFI:T:%s;S:%s;P:%s;H:%t;;",
		encryption,
		escapeWiFi(config.SSID),
		escapeWiFi(config.Password),
		config.Hidden,
	), nil
}

func WIFI(config WIFIConfig) (string, error) {
	return WiFi(config)
}

func VCard(config VCardConfig) (string, error) {
	config.Name = strings.TrimSpace(config.Name)
	if config.Name == "" || containsControl(config.Name) {
		return "", fmt.Errorf("%w: vCard name is empty or invalid", ErrInvalidPayload)
	}
	if config.Email != "" {
		if err := validateEmail(config.Email); err != nil {
			return "", err
		}
	}
	if config.URL != "" {
		if _, err := URL(config.URL); err != nil {
			return "", fmt.Errorf("%w: vCard URL: %v", ErrInvalidPayload, err)
		}
	}
	fields := []string{
		"BEGIN:VCARD",
		"VERSION:3.0",
		"FN:" + escapeVCard(config.Name),
	}
	if config.Company != "" {
		fields = append(fields, "ORG:"+escapeVCard(config.Company))
	}
	if config.Title != "" {
		fields = append(fields, "TITLE:"+escapeVCard(config.Title))
	}
	if config.Phone != "" {
		if err := validatePhone(config.Phone); err != nil {
			return "", err
		}
		fields = append(fields, "TEL;TYPE=CELL:"+escapeVCard(config.Phone))
	}
	if config.Email != "" {
		fields = append(fields, "EMAIL:"+escapeVCard(config.Email))
	}
	if config.URL != "" {
		fields = append(fields, "URL:"+escapeVCard(config.URL))
	}
	if config.Address != "" {
		fields = append(fields, "ADR;TYPE=WORK:;;"+escapeVCard(config.Address)+";;;;")
	}
	fields = append(fields, "END:VCARD")
	return strings.Join(fields, "\r\n") + "\r\n", nil
}

func Email(config EmailConfig) (string, error) {
	address := strings.TrimSpace(config.Address)
	if err := validateEmail(address); err != nil {
		return "", err
	}
	query := make(neturl.Values)
	if config.Subject != "" {
		query.Set("subject", config.Subject)
	}
	if config.Body != "" {
		query.Set("body", config.Body)
	}
	uri := neturl.URL{Scheme: "mailto", Opaque: address, RawQuery: query.Encode()}
	return uri.String(), nil
}

func Phone(number string) (string, error) {
	number = strings.TrimSpace(number)
	if err := validatePhone(number); err != nil {
		return "", err
	}
	return "tel:" + neturl.PathEscape(number), nil
}

func SMS(number, message string) (string, error) {
	number = strings.TrimSpace(number)
	if err := validatePhone(number); err != nil {
		return "", err
	}
	query := make(neturl.Values)
	if message != "" {
		query.Set("body", message)
	}
	uri := neturl.URL{Scheme: "sms", Opaque: number, RawQuery: query.Encode()}
	return uri.String(), nil
}

func Geo(latitude, longitude float64) (string, error) {
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		return "", fmt.Errorf("%w: coordinates out of range", ErrInvalidPayload)
	}
	return "geo:" + strconv.FormatFloat(latitude, 'f', -1, 64) + "," + strconv.FormatFloat(longitude, 'f', -1, 64), nil
}

func validateEmail(value string) error {
	if value == "" || containsControl(value) {
		return fmt.Errorf("%w: email address is empty or invalid", ErrInvalidPayload)
	}
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value {
		return fmt.Errorf("%w: invalid email address %q", ErrInvalidPayload, value)
	}
	return nil
}

func validatePhone(value string) error {
	if value == "" || containsControl(value) {
		return fmt.Errorf("%w: phone number is empty or invalid", ErrInvalidPayload)
	}
	for _, character := range value {
		if unicode.IsDigit(character) || strings.ContainsRune("+()- .#*", character) {
			continue
		}
		return fmt.Errorf("%w: phone number contains unsupported character %q", ErrInvalidPayload, character)
	}
	return nil
}

func escapeWiFi(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for _, character := range value {
		if strings.ContainsRune(`\\;,\":`, character) {
			builder.WriteByte('\\')
		}
		builder.WriteRune(character)
	}
	return builder.String()
}

func escapeVCard(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"\r\n", "\\n",
		"\n", "\\n",
		"\r", "\\n",
		";", "\\;",
		",", "\\,",
	)
	return replacer.Replace(value)
}

func containsControl(value string) bool {
	for _, character := range value {
		if unicode.IsControl(character) {
			return true
		}
	}
	return false
}

func containsControlExceptNewline(value string) bool {
	for _, character := range value {
		if unicode.IsControl(character) && character != '\n' && character != '\r' && character != '\t' {
			return true
		}
	}
	return false
}

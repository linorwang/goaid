package pay

import "strings"

func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return defaultValue
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func int64Ptr(value int64) *int64 {
	return &value
}

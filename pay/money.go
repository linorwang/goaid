package pay

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	defaultCurrencyCNY      = "CNY"
	defaultUnionCurrencyCNY = "156"
)

// YuanFromFen formats an amount in cents as a yuan amount string for Alipay.
func YuanFromFen(fen int64) string {
	sign := ""
	if fen < 0 {
		sign = "-"
		fen = -fen
	}
	return fmt.Sprintf("%s%d.%02d", sign, fen/100, fen%100)
}

// FenFromYuan parses a yuan amount string such as "12.34" into cents.
func FenFromYuan(amount string) (int64, error) {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return 0, fmt.Errorf("amount cannot be empty")
	}
	if strings.HasPrefix(amount, "-") {
		return 0, fmt.Errorf("amount cannot be negative")
	}

	parts := strings.Split(amount, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid amount %q", amount)
	}
	if parts[0] == "" {
		return 0, fmt.Errorf("invalid amount %q", amount)
	}

	yuan, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid yuan amount %q: %w", amount, err)
	}

	var cents int64
	if len(parts) == 2 {
		if len(parts[1]) > 2 {
			return 0, fmt.Errorf("amount supports at most two decimals")
		}
		fraction := parts[1]
		for len(fraction) < 2 {
			fraction += "0"
		}
		if fraction != "" {
			cents, err = strconv.ParseInt(fraction, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid cent amount %q: %w", amount, err)
			}
		}
	}

	return yuan*100 + cents, nil
}

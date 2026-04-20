package domain

import (
	"fmt"
	"strings"
)

func ParseBusyLevel(s string) (BusyLevel, error) {
	switch BusyLevel(strings.ToLower(strings.TrimSpace(s))) {
	case BusyQuiet:
		return BusyQuiet, nil
	case BusyModerate:
		return BusyModerate, nil
	case BusyBusy:
		return BusyBusy, nil
	case BusyClosed:
		return BusyClosed, nil
	default:
		return "", fmt.Errorf("invalid busy_level")
	}
}

func ParseStatusSource(s string) (StatusSource, error) {
	switch StatusSource(strings.ToLower(strings.TrimSpace(s))) {
	case SourceMerchant:
		return SourceMerchant, nil
	case SourceOperator:
		return SourceOperator, nil
	case SourceIntegration:
		return SourceIntegration, nil
	case SourceCrowd:
		return SourceCrowd, nil
	default:
		return "", fmt.Errorf("invalid source")
	}
}

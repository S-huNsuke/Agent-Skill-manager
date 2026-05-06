package catalog

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	ReasonUnsupportedSchemaMajor = "unsupported_schema_major"
	ReasonClientTooOld           = "client_too_old"
	ReasonInvalidSchemaVersion   = "invalid_schema_version"
	ReasonInvalidClientVersion   = "invalid_client_version"
)

type CompatibilityResult struct {
	Compatible bool
	ReasonCode string
	Reason     string
}

func CheckCompatibility(c Catalog, clientVersion string) CompatibilityResult {
	schemaMajor, err := majorComponent(c.SchemaVersion)
	if err != nil {
		return CompatibilityResult{
			Compatible: false,
			ReasonCode: ReasonInvalidSchemaVersion,
			Reason:     fmt.Sprintf("catalog schema version %q is invalid", c.SchemaVersion),
		}
	}
	if schemaMajor != supportedSchemaMajor {
		return CompatibilityResult{
			Compatible: false,
			ReasonCode: ReasonUnsupportedSchemaMajor,
			Reason:     fmt.Sprintf("catalog schema major version %d is unsupported", schemaMajor),
		}
	}

	if c.MinClientVersion == "" {
		return CompatibilityResult{Compatible: true}
	}

	comparison, err := compareVersions(clientVersion, c.MinClientVersion)
	if err != nil {
		return CompatibilityResult{
			Compatible: false,
			ReasonCode: ReasonInvalidClientVersion,
			Reason:     fmt.Sprintf("client version %q or minimum version %q is invalid", clientVersion, c.MinClientVersion),
		}
	}
	if comparison < 0 {
		return CompatibilityResult{
			Compatible: false,
			ReasonCode: ReasonClientTooOld,
			Reason:     fmt.Sprintf("client version %s is below required minimum %s", clientVersion, c.MinClientVersion),
		}
	}

	return CompatibilityResult{Compatible: true}
}

func majorComponent(version string) (int, error) {
	parts := strings.Split(version, ".")
	if len(parts) == 0 || parts[0] == "" {
		return 0, fmt.Errorf("missing major version")
	}
	return strconv.Atoi(parts[0])
}

func compareVersions(left, right string) (int, error) {
	leftParts, err := parseVersion(left)
	if err != nil {
		return 0, err
	}
	rightParts, err := parseVersion(right)
	if err != nil {
		return 0, err
	}

	width := len(leftParts)
	if len(rightParts) > width {
		width = len(rightParts)
	}

	for i := 0; i < width; i++ {
		leftValue := 0
		if i < len(leftParts) {
			leftValue = leftParts[i]
		}
		rightValue := 0
		if i < len(rightParts) {
			rightValue = rightParts[i]
		}
		switch {
		case leftValue < rightValue:
			return -1, nil
		case leftValue > rightValue:
			return 1, nil
		}
	}

	return 0, nil
}

func parseVersion(version string) ([]int, error) {
	parts := strings.Split(version, ".")
	values := make([]int, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			return nil, fmt.Errorf("invalid version %q", version)
		}
		value, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid version %q", version)
		}
		values = append(values, value)
	}
	return values, nil
}

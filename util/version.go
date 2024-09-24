package util

import (
	"strconv"
	"strings"
)

func SerializeAgentVersion(version []int32) *string {
	if len(version) == 0 {
		return nil
	}

	var builder strings.Builder
	for i, v := range version {
		if i > 0 {
			builder.WriteByte('.')
		}
		builder.WriteString(strconv.FormatInt(int64(v), 10))
	}
	result := builder.String()
	return &result
}

func DeserializeAgentVersion(version *string) []int32 {
	if version == nil {
		return nil
	}

	parts := strings.Split(*version, ".")
	if len(parts) < 3 || len(parts) > 4 {
		return nil
	}

	result := make([]int32, len(parts))
	for i, part := range parts {
		num, err := strconv.ParseInt(part, 10, 32)
		if err != nil {
			return nil
		}
		result[i] = int32(num)
	}

	return result
}

package util

import "fmt"

func TokenizeCommand(commandStr string) ([]string, error) {
	var command []string
	var current string
	inQuotes := false
	escapeNext := false

	for _, r := range commandStr {
		if escapeNext {
			current += string(r)
			escapeNext = false
			continue
		}
		switch r {
		case '\\':
			escapeNext = true
		case '"':
			inQuotes = !inQuotes
		case ' ':
			if !inQuotes {
				if current != "" {
					command = append(command, current)
					current = ""
				}
				continue
			}
			current += string(r)
		default:
			current += string(r)
		}
	}
	if inQuotes {
		return nil, fmt.Errorf("unclosed quotes in KUBE_MIGRATE_COMMAND")
	}
	if current != "" {
		command = append(command, current)
	}

	return command, nil
}

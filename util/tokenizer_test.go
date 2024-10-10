package util

import (
	"reflect"
	"testing"
)

func TestTokenizeCommand(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{
			name:    "Simple command",
			input:   "ls -l",
			want:    []string{"ls", "-l"},
			wantErr: false,
		},
		{
			name:    "Command with quotes",
			input:   `echo "Hello, World!"`,
			want:    []string{"echo", "Hello, World!"},
			wantErr: false,
		},
		{
			name:    "Command with escaped quotes",
			input:   `echo "Hello, \"World\"!"`,
			want:    []string{"echo", `Hello, "World"!`},
			wantErr: false,
		},
		{
			name:    "Command with multiple spaces",
			input:   "grep   -i   pattern",
			want:    []string{"grep", "-i", "pattern"},
			wantErr: false,
		},
		{
			name:    "Command with escaped spaces",
			input:   `echo Hello\ World`,
			want:    []string{"echo", "Hello World"},
			wantErr: false,
		},
		{
			name:    "Unclosed quotes",
			input:   `echo "Hello, World!`,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Empty command",
			input:   "",
			want:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TokenizeCommand(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("TokenizeCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TokenizeCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

package sshc

import (
	"testing"
)

func TestExpandPath(t *testing.T) {
	t.Setenv("HOME", "/home/testuser")
	tests := []struct {
		path string
		base string
		want string
	}{
		{"path/to/key", "path/to", "path/to/path/to/key"},
		{"./path/to/key", "path/to", "path/to/path/to/key"},
		{"/path/to/key", "path/to", "/path/to/key"},
		{"~/path/to/key", "path/to", "/home/testuser/path/to/key"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := expandPath(tt.path, tt.base)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %v want %v", got, tt.want)
			}
		})
	}
}

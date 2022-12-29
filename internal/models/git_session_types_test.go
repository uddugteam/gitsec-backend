package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitSessionTypeFromString(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		expected  GitSessionType
		expectErr bool
	}{
		{
			name:      "git-receive-pack",
			input:     "git-receive-pack",
			expected:  GitSessionReceivePack,
			expectErr: false,
		},
		{
			name:      "git-upload-pack",
			input:     "git-upload-pack",
			expected:  GitSessionUploadPack,
			expectErr: false,
		},
		{
			name:      "invalid type",
			input:     "invalid-type",
			expected:  GitSessionUnsupported,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := GitSessionTypeFromString(tc.input)
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestGitSessionType_String(t *testing.T) {
	tests := []struct {
		name string
		s    GitSessionType
		want string
	}{
		{
			name: "Test GitSessionReceivePack string representation",
			s:    GitSessionReceivePack,
			want: "git-receive-pack",
		},
		{
			name: "Test GitSessionUploadPack string representation",
			s:    GitSessionUploadPack,
			want: "git-upload-pack",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Errorf("GitSessionType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

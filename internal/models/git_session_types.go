package models

import (
	"fmt"
	"strings"
)

// GitSessionType represents available Git Session Types
type GitSessionType int

const (
	// GitSessionReceivePack is the type of session that allows
	// clients to receive pack data from the server.
	GitSessionReceivePack GitSessionType = iota
	// GitSessionUploadPack is the type of session that allows
	// clients to send pack data to the server.
	GitSessionUploadPack
	// GitSessionUnsupported is the type of session that is not supported.
	GitSessionUnsupported
)

// gitSessionTypes is slice of GitSessionType
// string representations
var gitSessionTypes = [...]string{
	GitSessionReceivePack: "git-receive-pack",
	GitSessionUploadPack:  "git-upload-pack",
}

// String returns the GitSessionType as a string
func (s GitSessionType) String() string {
	return gitSessionTypes[s]
}

// GitSessionTypeFromString returns a new GitSessionType
// enum from the given string
func GitSessionTypeFromString(s string) (GitSessionType, error) {
	for i, r := range gitSessionTypes {
		if strings.ToLower(s) == r {
			return GitSessionType(i), nil
		}
	}
	return GitSessionUnsupported, fmt.Errorf("invalid git session type value %q", s)
}

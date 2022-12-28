package models

import (
	"fmt"
	"strings"
)

// GitSessionType represent available Git Session Types
type GitSessionType int

const (
	GitSessionReceivePack GitSessionType = iota
	GitSessionUploadPack
	GitSessionUnsupported
)

// gitSessionTypes is slice of GitSessionType
// string representations
var gitSessionTypes = [...]string{
	GitSessionReceivePack: "git-receive-pack",
	GitSessionUploadPack:  "git-upload-pack",
}

// String return GitSessionType enum as a string
func (s GitSessionType) String() string {
	return gitSessionTypes[s]
}

// GitSessionTypeFromString return new GitSessionType
// enum from given string
func GitSessionTypeFromString(s string) (GitSessionType, error) {
	for i, r := range gitSessionTypes {
		if strings.ToLower(s) == r {
			return GitSessionType(i), nil
		}
	}
	return GitSessionUnsupported, fmt.Errorf("invalid git session type value %q", s)
}

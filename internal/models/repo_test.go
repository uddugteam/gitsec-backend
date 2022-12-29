package models

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testRepoName     = "test-repo"
	testRepoBasePath = "/tmp"
)

func TestNewRepo(t *testing.T) {
	defer func() {
		if err := os.RemoveAll(fmt.Sprintf("%s/%s", testRepoBasePath, testRepoName)); err != nil {
			t.Fatalf("failed to remove test repo: %v", err)
		}
	}()

	repo, err := NewRepo(testRepoName, testRepoBasePath)
	require.NoError(t, err)
	assert.Equal(t, testRepoName, repo.Name)
	assert.Equal(t, testRepoBasePath, repo.BasePath)
	assert.NotNil(t, repo.fileSystem)
	assert.NotNil(t, repo.server)
	assert.NotNil(t, repo.endpoint)
}

func TestRepo_FullPath(t *testing.T) {
	repo, err := NewRepo(testRepoName, testRepoBasePath)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%s/%s/", testRepoBasePath, testRepoName), repo.FullPath())
}

func TestRepo_NewUploadPackSession(t *testing.T) {
	repo, err := NewRepo(testRepoName, testRepoBasePath)
	require.NoError(t, err)
	s, err := repo.NewUploadPackSession()
	require.NoError(t, err)
	assert.NotNil(t, s)
}

func TestRepo_NewReceivePackSession(t *testing.T) {
	repo, err := NewRepo(testRepoName, testRepoBasePath)
	require.NoError(t, err)
	s, err := repo.NewReceivePackSession()
	require.NoError(t, err)
	assert.NotNil(t, s)
}

func TestRepo_isRepoFSExists(t *testing.T) {
	repo, err := NewRepo(testRepoName, testRepoBasePath)
	require.NoError(t, err)
	assert.True(t, repo.isRepoFSExists())
}

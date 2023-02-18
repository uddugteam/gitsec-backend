package models

import (
	"errors"
	"fmt"
	"io"

	"github.com/go-git/go-git/v5/plumbing/object"
)

type RepoMetadata struct {
	Name         string      `json:"name"`
	ExternalUrl  string      `json:"external_url"`
	Description  string      `json:"description"`
	Tree         []*RepoFile `json:"content"`
	Commit       string      `json:"commit"`
	Timestamp    int64       `json:"timestamp"`
	CommitsCount int         `json:"commits_count"`
}

func (m *RepoMetadata) FillContent(tree *object.Tree) error {
	if err := tree.Files().ForEach(func(f *object.File) error {

		content, err := f.Contents()
		if err != nil {
			return fmt.Errorf("failed to retrieve content: %w", err)
		}

		m.Tree = append(m.Tree, &RepoFile{
			Name:    f.Name,
			Content: content,
			Hash:    f.Hash.String(),
		})

		return nil
	}); err != nil {
		if errors.Is(err, io.EOF) {
			fmt.Println("EOF")
		} else {
			return fmt.Errorf("failed to iter tree: %w", err)
		}
	}

	return nil
}

func (m *RepoMetadata) FillCommit(repo *Repo) error {
	for _, f := range m.Tree {
		commit, err := repo.FileLastCommit(f.Name)
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println("EOF")
				continue
			} else {
				return fmt.Errorf("failed to retrieve file last commit: %w", err)
			}
		}

		f.Author = commit.Author.Name
		f.Commit = commit.Hash.String()
		f.Timestamp = commit.Author.When.Unix()
	}
	return nil
}

type RepoFile struct {
	Name      string `json:"name"`
	Hash      string `json:"hash"`
	Author    string `json:"author"`
	Commit    string `json:"commit"`
	Timestamp int64  `json:"timestamp"`
	Content   string `json:"-"`
}

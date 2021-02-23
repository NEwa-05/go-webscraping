package main

import "time"

// Gist represents a GitHub's gist.
type Gist struct {
	ID          *string                   `json:"id,omitempty"`
	Description *string                   `json:"description,omitempty"`
	Public      *bool                     `json:"public,omitempty"`
	Owner       *User                     `json:"owner,omitempty"`
	Files       map[GistFilename]GistFile `json:"files,omitempty"`
	Comments    *int                      `json:"comments,omitempty"`
	HTMLURL     *string                   `json:"html_url,omitempty"`
	GitPullURL  *string                   `json:"git_pull_url,omitempty"`
	GitPushURL  *string                   `json:"git_push_url,omitempty"`
	CreatedAt   *time.Time                `json:"created_at,omitempty"`
	UpdatedAt   *time.Time                `json:"updated_at,omitempty"`
	NodeID      *string                   `json:"node_id,omitempty"`
}

// GistFilename represents filename on a gist.
type GistFilename string

// GistFile represents a file on a gist.
type GistFile struct {
	Size     *int    `json:"size,omitempty"`
	Filename *string `json:"filename,omitempty"`
	Language *string `json:"language,omitempty"`
	Type     *string `json:"type,omitempty"`
	RawURL   *string `json:"raw_url,omitempty"`
	Content  *string `json:"content,omitempty"`
}

// User represents a GitHub user.
type User struct {
	Login *string `json:"login,omitempty"`
}

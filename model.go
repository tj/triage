package triage

import (
	"context"

	"github.com/google/go-github/v28/github"
	"github.com/tj/go-tea"
	"github.com/tj/go-tea/input"
	"github.com/tj/go-tea/options"
)

// Page is the page the user is viewing.
type Page int

// Pages available.
const (
	PageNotifications Page = iota
	PageNotification
	PageLabels
	PageComment
)

// Model is the application model.
type Model struct {
	// active page
	Page

	// notifications page
	Notifications        []*github.Notification
	NotificationsScrollY int
	Selected             int
	Searching            bool
	SearchInput          input.Model

	// notification page
	Notification        *github.Notification
	NotificationScrollY int
	Labels              []*github.Label
	Issue               *github.Issue
	Comments            []*github.IssueComment
	LoadingIssue        bool
	LoadingLabels       bool
	LoadingComments     bool

	// labels page
	LabelOptions options.Model
	RepoLabels   []*github.Label

	// comment
	CommentInput input.Model

	// shared
	MarkingAsRead bool
	Unsubscribing bool
	Unwatching    bool
	Loading       bool
	Width         int
	Height        int
}

// Init function.
func Init(ctx context.Context) (tea.Model, tea.Cmd) {
	return Model{
		Page:    PageNotifications,
		Loading: true,
	}, GetDimensions
}

package triage

import (
	"strings"

	"github.com/google/go-github/v28/github"
)

// filterNotifications using the given text.
func filterNotifications(notifications []*github.Notification, s string) (filtered []*github.Notification) {
	for _, n := range notifications {
		if !strings.Contains(n.Repository.GetFullName(), s) {
			continue
		}
		filtered = append(filtered, n)
	}
	return
}

// ownerRepo returns the owner and repo.
func ownerRepo(n *github.Notification) (owner, repo string) {
	repository := n.GetRepository()
	owner = repository.GetOwner().GetLogin()
	repo = repository.GetName()
	return
}

// min returns the minimum of two ints.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two ints.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

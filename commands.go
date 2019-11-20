package triage

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/AstromechZA/terminfo"
	"github.com/google/go-github/v28/github"
	"github.com/pkg/browser"
	"github.com/tj/go-tea"
)

// GetDimensions requests the terminal dimensions.
func GetDimensions(ctx context.Context) tea.Msg {
	w, h, err := terminfo.GetStdoutDimensions()
	if err != nil {
		return err
	}

	// pty may be allocated with 0x0,
	// trap SIGWINCH for Docker etc
	if w == 0 && h == 0 {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGWINCH)
		<-ch
		return GetDimensions(ctx)
	}

	return GotDimensions{w, h}
}

// LoadNotifications loads the notifications.
func LoadNotifications(ctx context.Context) tea.Msg {
	gh := MustClientFromContext(ctx)

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	// TODO: pagination
	options := &github.NotificationListOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var filtered []*github.Notification

	// fetch
	notifications, _, err := gh.Activity.ListNotifications(ctx, options)
	if err != nil {
		return fmt.Errorf("fetching notifications: %w", err)
	}

	// filter
	for _, n := range notifications {
		// ignore releases
		if n.GetSubject().GetType() == "Release" {
			continue
		}

		filtered = append(filtered, n)
	}

	return NotificationsLoaded{filtered}
}

// LoadNotification loads a notification's issue, labels, and comments.
func LoadNotification(n *github.Notification) tea.Cmd {
	return LoadNotificationIssue(n)
}

// LoadNotificationIssue loads a notification's issue.
func LoadNotificationIssue(n *github.Notification) tea.Cmd {
	return func(ctx context.Context) tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()

		issue, err := getIssue(ctx, n)
		if err != nil {
			return fmt.Errorf("fetching issue: %w", err)
		}

		return NotificationIssueLoaded{issue}
	}
}

// LoadNotificationLabels loads a notification's labels.
func LoadNotificationLabels(n *github.Notification, issue *github.Issue) tea.Cmd {
	return func(ctx context.Context) tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()

		labels, err := getIssueLabels(ctx, n, issue.GetNumber())
		if err != nil {
			return fmt.Errorf("fetching issue labels: %w", err)
		}

		return NotificationLabelsLoaded{labels}
	}
}

// LoadNotificationComments loads a notification's comments.
func LoadNotificationComments(issue *github.Issue) tea.Cmd {
	return func(ctx context.Context) tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()

		comments, err := getIssueComments(ctx, issue)
		if err != nil {
			return fmt.Errorf("fetching issue comments: %w", err)
		}

		return NotificationCommentsLoaded{comments}
	}
}

// LoadRepoLabels loads a repo's labels.
func LoadRepoLabels(n *github.Notification) tea.Cmd {
	return func(ctx context.Context) tea.Msg {
		gh := MustClientFromContext(ctx)

		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()

		owner, repo := ownerRepo(n)
		labels, _, err := gh.Issues.ListLabels(ctx, owner, repo, nil)
		if err != nil {
			return fmt.Errorf("fetching repo labels: %w", err)
		}

		return LabelsLoaded{labels}
	}
}

// UpdateNotificationLabels updates an issue's labels.
func UpdateNotificationLabels(n *github.Notification, issue *github.Issue, labels []string) tea.Cmd {
	return func(ctx context.Context) tea.Msg {
		gh := MustClientFromContext(ctx)

		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()

		owner, repo := ownerRepo(n)

		if len(labels) == 0 {
			_, err := gh.Issues.RemoveLabelsForIssue(ctx, owner, repo, issue.GetNumber())
			if err != nil {
				return fmt.Errorf("removing labels: %w", err)
			}
		} else {
			_, _, err := gh.Issues.ReplaceLabelsForIssue(ctx, owner, repo, issue.GetNumber(), labels)
			if err != nil {
				return fmt.Errorf("replacing labels: %w", err)
			}
		}

		return NotificationLabelsUpdated{}
	}
}

// UpdateNotificationPriority updates an issue's priority by name.
func UpdateNotificationPriority(n *github.Notification, issue *github.Issue, name string) tea.Cmd {
	return func(ctx context.Context) tea.Msg {
		gh := MustClientFromContext(ctx)
		config := MustConfigFromContext(ctx)

		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()

		owner, repo := ownerRepo(n)

		// find label
		var priority Priority
		for _, p := range config.Priorities {
			if p.Name == name {
				priority = p
				break
			}
		}

		// strip leading # from the color
		color := strings.Replace(priority.Color, "#", "", 1)

		// create the label
		desc := fmt.Sprintf("%s priority issue.", priority.Name)
		_, _, err := gh.Issues.CreateLabel(ctx, owner, repo, &github.Label{
			Name:        &priority.Label,
			Color:       &color,
			Description: &desc,
		})

		// ignore error if it already exists
		if err != nil && !isAlreadyExists(err) {
			return fmt.Errorf("creating priority label: %w", err)
		}

		// remove any priority labels
		for _, p := range config.Priorities {
			_, err := gh.Issues.RemoveLabelForIssue(ctx, owner, repo, issue.GetNumber(), p.Label)
			if err != nil && !isNotFound(err) {
				return fmt.Errorf("error removing label %q: %w", p.Label, err)
			}
		}

		// assign the label
		_, _, err = gh.Issues.AddLabelsToIssue(ctx, owner, repo, issue.GetNumber(), []string{priority.Label})
		if err != nil {
			return fmt.Errorf("assigning priority label: %w", err)
		}

		return NotificationPriorityUpdated{}
	}
}

// AddComment adds a comment to an issue.
func AddComment(n *github.Notification, issue *github.Issue, comment string) tea.Cmd {
	return func(ctx context.Context) tea.Msg {
		gh := MustClientFromContext(ctx)

		ctx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()

		owner, repo := ownerRepo(n)
		_, _, err := gh.Issues.CreateComment(ctx, owner, repo, issue.GetNumber(), &github.IssueComment{
			Body: &comment,
		})

		if err != nil {
			return fmt.Errorf("creating comment: %w", err)
		}

		return CommentAdded{}
	}
}

// MarkAsRead marks an issue as read.
func MarkAsRead(n *github.Notification) tea.Cmd {
	return func(ctx context.Context) tea.Msg {
		gh := MustClientFromContext(ctx)

		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()

		_, err := gh.Activity.MarkThreadRead(ctx, n.GetID())
		if err != nil {
			return fmt.Errorf("marking thread as read: %w", err)
		}

		return MarkedAsRead{n}
	}
}

// Unsubscribe unsubscribes from the issue, and marks it as read.
func Unsubscribe(n *github.Notification) tea.Cmd {
	return func(ctx context.Context) tea.Msg {
		gh := MustClientFromContext(ctx)

		ctx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()

		_, err := gh.Activity.DeleteThreadSubscription(ctx, n.GetID())
		if err != nil {
			return fmt.Errorf("removing thread subscription: %w", err)
		}

		_, err = gh.Activity.MarkThreadRead(ctx, n.GetID())
		if err != nil {
			return fmt.Errorf("marking thread as read: %w", err)
		}

		return Unsubscribed{n}
	}
}

// Unwatch unwatches the repository, and marks it as read.
func Unwatch(owner, repo string) tea.Cmd {
	return func(ctx context.Context) tea.Msg {
		gh := MustClientFromContext(ctx)

		ctx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()

		_, err := gh.Activity.DeleteRepositorySubscription(ctx, owner, repo)
		if err != nil {
			return fmt.Errorf("unwatching repository: %w", err)
		}

		return Unwatched{
			Owner: owner,
			Repo:  repo,
		}
	}
}

// OpenInBrowser opens the in the browser.
func OpenInBrowser(n *github.Notification) tea.Cmd {
	return func(ctx context.Context) tea.Msg {
		gh := MustClientFromContext(ctx)

		url := n.Subject.GetURL()

		req, err := gh.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()

		var v github.Issue
		_, err = gh.Do(ctx, req, &v)
		if err != nil {
			return fmt.Errorf("fetching issue url: %w", err)
		}

		return browser.OpenURL(v.GetHTMLURL())
	}
}

// getIssue returns the issue for the notification.
func getIssue(ctx context.Context, n *github.Notification) (*github.Issue, error) {
	gh := MustClientFromContext(ctx)
	url := n.Subject.GetURL()

	req, err := gh.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var v github.Issue
	_, err = gh.Do(ctx, req, &v)
	return &v, err
}

// getIssueLabels returns the labels for the issue.
func getIssueLabels(ctx context.Context, n *github.Notification, number int) ([]*github.Label, error) {
	gh := MustClientFromContext(ctx)
	owner, repo := ownerRepo(n)
	labels, _, err := gh.Issues.ListLabelsByIssue(ctx, owner, repo, number, nil)
	return labels, err
}

// getIssueComments returns the comments for an issue.
func getIssueComments(ctx context.Context, issue *github.Issue) (v []*github.IssueComment, err error) {
	gh := MustClientFromContext(ctx)
	url := issue.GetCommentsURL()

	req, err := gh.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	_, err = gh.Do(ctx, req, &v)
	return
}

// isNotFound returns true the error is a 404.
func isNotFound(err error) bool {
	res, ok := err.(*github.ErrorResponse)
	if !ok {
		return false
	}

	return res.Response.StatusCode == 404
}

// isAlreadyExists returns true is already_exists.
func isAlreadyExists(err error) bool {
	res, ok := err.(*github.ErrorResponse)
	if !ok {
		return false
	}

	for _, e := range res.Errors {
		if e.Code == "already_exists" {
			return true
		}
	}

	return false
}

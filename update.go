package triage

import (
	"context"

	"github.com/tj/go-tea/input"
	"github.com/tj/go-tea/option"

	"github.com/google/go-github/v28/github"
	"github.com/tj/go-tea"
	"github.com/tj/go-tea/options"
	"github.com/tj/go-terminput"
)

// listItemHeight is the number of rows a list item consumes.
var listItemHeight = 4

// GotDimensions msg.
type GotDimensions struct {
	Width  int
	Height int
}

// NotificationLabelsUpdated msg.
type NotificationLabelsUpdated struct{}

// NotificationPriorityUpdated msg.
type NotificationPriorityUpdated struct{}

// CommentAdded msg.
type CommentAdded struct{}

// LabelsLoaded msg.
type LabelsLoaded struct {
	Labels []*github.Label
}

// NotificationsLoaded msg.
type NotificationsLoaded struct {
	Notifications []*github.Notification
}

// NotificationIssueLoaded msg.
type NotificationIssueLoaded struct {
	Issue *github.Issue
}

// NotificationLabelsLoaded msg.
type NotificationLabelsLoaded struct {
	Labels []*github.Label
}

// NotificationCommentsLoaded msg.
type NotificationCommentsLoaded struct {
	Comments []*github.IssueComment
}

// MarkedAsRead msg.
type MarkedAsRead struct {
	*github.Notification
}

// Unsubscribed msg.
type Unsubscribed struct {
	*github.Notification
}

// Unwatched msg.
type Unwatched struct {
	Owner string
	Repo  string
}

// Update function.
func Update(ctx context.Context, msg tea.Msg, model tea.Model) (tea.Model, tea.Cmd) {
	m := model.(Model)
	config := MustConfigFromContext(ctx)

	// filter so that selection calculations
	// take the search text into account
	notifications := filterNotifications(m.Notifications, m.SearchInput.Value)

	// dimensions
	if v, ok := msg.(GotDimensions); ok {
		m.Width = v.Width
		m.Height = v.Height
		return m, LoadNotifications
	}

	// comment
	if m.Page == PageComment {
		switch msg := msg.(type) {
		case *terminput.KeyboardInput:
			switch msg.Key() {
			case terminput.KeyEscape:
				m.CommentInput = input.Model{}
				m.Page = PageNotification
				return m, nil
			case terminput.KeyEnter:
				comment := m.CommentInput.Value
				m.CommentInput = input.Model{}
				m.Page = PageNotification
				return m, AddComment(m.Notification, m.Issue, comment)
			default:
				m.CommentInput = input.Update(msg, m.CommentInput)
			}
			return m, nil
		}
	}

	// labels
	if m.Page == PageLabels {
		switch msg := msg.(type) {
		case LabelsLoaded:
			m.RepoLabels = filterPriorityLabels(msg.Labels, config.Priorities)
			m.Loading = false
			return m, LoadNotificationLabels(m.Notification, m.Issue)
		case NotificationLabelsLoaded:
			m.LabelOptions = options.Model{
				Options:  labelNames(m.RepoLabels),
				Selected: labelsSelected(m.RepoLabels, msg.Labels),
			}
			m.LoadingLabels = false
			return m, nil
		case NotificationLabelsUpdated:
			m.Page = PageNotification
			m.LoadingLabels = true
			return m, LoadNotificationLabels(m.Notification, m.Issue)
		case *terminput.KeyboardInput:
			switch msg.Key() {
			case terminput.KeyEnter:
				m.Page = PageNotification
				labels := m.LabelOptions.Value()
				return m, UpdateNotificationLabels(m.Notification, m.Issue, labels)
			case terminput.KeyEscape:
				m.LabelOptions = options.Model{}
				m.Page = PageNotification
				return m, nil
			default:
				m.LabelOptions = options.Update(msg, m.LabelOptions)
				return m, nil
			}
		}
	}

	// priorities
	if m.Page == PagePriorities {
		switch msg := msg.(type) {
		case NotificationPriorityUpdated:
			m.Page = PageNotification
			m.LoadingLabels = true
			return m, LoadNotificationLabels(m.Notification, m.Issue)
		case *terminput.KeyboardInput:
			switch msg.Key() {
			case terminput.KeyEnter:
				m.Page = PageNotification
				name := m.PriorityOptions.Value()
				return m, UpdateNotificationPriority(m.Notification, m.Issue, name)
			case terminput.KeyEscape:
				m.LabelOptions = options.Model{}
				m.Page = PageNotification
				return m, nil
			default:
				m.PriorityOptions = option.Update(msg, m.PriorityOptions)
				return m, nil
			}
		}
	}

	// notification
	if m.Page == PageNotification {
		switch msg := msg.(type) {
		case CommentAdded:
			m.LoadingComments = true
			return m, LoadNotificationComments(m.Issue)
		case NotificationIssueLoaded:
			m.Issue = msg.Issue
			m.LoadingIssue = false
			return m, tea.Batch(
				LoadNotificationLabels(m.Notification, msg.Issue),
				LoadNotificationComments(msg.Issue),
			)
		case NotificationLabelsLoaded:
			m.LoadingLabels = false
			m.Labels = msg.Labels
			return m, nil
		case NotificationCommentsLoaded:
			m.LoadingComments = false
			m.Comments = msg.Comments
			return m, nil
		case *terminput.KeyboardInput:
			switch msg.Key() {
			case terminput.KeyLeft:
				m.Page = PageNotifications
				m.NotificationScrollY = 0
				return m, nil
			case terminput.KeyUp:
				if m.NotificationScrollY > 0 {
					m.NotificationScrollY -= m.Height / 4
				} else {
					m.NotificationScrollY = 0
				}
				return m, nil
			case terminput.KeyDown:
				m.NotificationScrollY += m.Height / 4
				return m, nil
			case terminput.KeyBackspace:
				m.Page = PageNotifications
				m.MarkingAsRead = true
				return m, MarkAsRead(m.Notification)
			case terminput.KeyRune:
				switch r := msg.Rune(); r {
				case 'R':
					m.Labels = nil
					m.Comments = nil
					return loadNotification(m, m.Notification)
				case 'r':
					m.MarkingAsRead = true
					return m, MarkAsRead(m.Notification)
				case 'u':
					m.Unsubscribing = true
					return m, Unsubscribe(m.Notification)
				case 'o':
					return m, OpenInBrowser(m.Notification)
				case 'l':
					m.Page = PageLabels
					m.Loading = true
					m.LoadingLabels = true
					return m, LoadRepoLabels(m.Notification)
				case 'p':
					var o option.Model
					m.Page = PagePriorities
					for _, p := range config.Priorities {
						o.Options = append(o.Options, p.Name)
					}
					m.PriorityOptions = o
					return m, nil
				case 'c':
					m.Page = PageComment
					return m, nil
				}
			}
		}
	}

	// notifications
	if m.Page == PageNotifications {
		// searching
		if m.Searching {
			switch msg := msg.(type) {
			case *terminput.KeyboardInput:
				switch msg.Key() {
				case terminput.KeyEscape:
					m.Searching = false
					m.SearchInput.Value = ""
					return m, nil
				case terminput.KeyEnter, terminput.KeyDown:
					m.Searching = false
					return m, nil
				default:
					m.Selected = 0
					m.SearchInput = input.Update(msg, m.SearchInput)
					m.NotificationsScrollY = 0
					return m, nil
				}
			default:
				m.SearchInput = input.Update(msg, m.SearchInput)
				return m, nil
			}
		}

		// listing
		switch msg := msg.(type) {
		case NotificationsLoaded:
			m.Notifications = msg.Notifications
			m.Loading = false
			return m, nil
		case *terminput.KeyboardInput:
			if len(notifications) == 0 {
				return m, tea.Quit
			}
			switch msg.Key() {
			case terminput.KeyUp:
				if m.Selected > 0 {
					m.Selected--
				} else if m.SearchInput.Value != "" {
					m.Searching = true
				}
				m.NotificationsScrollY = scrollNotifications(m, notifications, 1)
				return m, nil
			case terminput.KeyDown:
				if m.Selected < len(notifications)-1 {
					m.Selected++
				}
				m.NotificationsScrollY = scrollNotifications(m, notifications, -1)
				return m, nil
			case terminput.KeyEnter, terminput.KeyRight:
				n := notifications[m.Selected]
				m.Page = PageNotification
				m.NotificationScrollY = 0
				m.Issue = nil
				m.Labels = nil
				m.Comments = nil
				return loadNotification(m, n)
			case terminput.KeyBackspace:
				n := notifications[m.Selected]
				m.MarkingAsRead = true
				return m, MarkAsRead(n)
			case terminput.KeyRune:
				switch r := msg.Rune(); r {
				case 'R':
					m.Loading = true
					return m, LoadNotifications
				case 'r':
					n := notifications[m.Selected]
					m.MarkingAsRead = true
					return m, MarkAsRead(n)
				case 'u':
					n := notifications[m.Selected]
					m.Unsubscribing = true
					return m, Unsubscribe(n)
				case 'U':
					n := m.Notifications[m.Selected]
					owner, repo := ownerRepo(n)
					var cmds []tea.Cmd
					cmds = append(cmds, Unwatch(owner, repo))
					for _, n := range getNotificationsByRepo(m.Notifications, owner, repo) {
						cmds = append(cmds, MarkAsRead(n))
					}
					return m, tea.Batch(cmds...)
				case 'o':
					n := notifications[m.Selected]
					return m, OpenInBrowser(n)
				case '/':
					m.Searching = true
					return m, nil
				}
			}
		}
	}

	// shared messages
	if m.Page == PageNotification || m.Page == PageNotifications {
		switch msg := msg.(type) {
		case Unsubscribed:
			m.Page = PageNotifications
			m.Notifications = removeNotification(m.Notifications, msg.GetID())
			m.Selected = min(m.Selected, len(notifications)-1)
			m.Unsubscribing = false
			m.NotificationScrollY = 0
			return m, nil
		case MarkedAsRead:
			m.Page = PageNotifications
			m.Notifications = removeNotification(m.Notifications, msg.GetID())
			m.Selected = min(m.Selected, len(notifications)-1)
			m.MarkingAsRead = false
			m.NotificationScrollY = 0
			return m, nil
		case Unwatched:
			m.Page = PageNotifications
			m.Unwatching = false
			m.Selected = min(m.Selected, len(notifications)-1)
			m.NotificationScrollY = 0
			return m, nil
		}
	}

	// global messages
	switch msg := msg.(type) {
	case *terminput.KeyboardInput:
		switch msg.Key() {
		case terminput.KeyEscape:
			return m, tea.Quit
		case terminput.KeyRune:
			switch msg.Rune() {
			case 'q':
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

// loadNotification loads the notification.
func loadNotification(m Model, n *github.Notification) (Model, tea.Cmd) {
	m.Notification = n
	m.LoadingIssue = true
	m.LoadingLabels = true
	m.LoadingComments = true
	return m, LoadNotification(n)
}

// labelNames returns label names, filtering priorities.
func labelNames(labels []*github.Label) (names []string) {
	for _, l := range labels {
		names = append(names, l.GetName())
	}
	return
}

// labelsSelected returns the indexes of selected labels.
func labelsSelected(labels []*github.Label, selected []*github.Label) (indexes []int) {
	for i, l := range labels {
		for _, s := range selected {
			if s.GetID() == l.GetID() {
				indexes = append(indexes, i)
				break
			}
		}
	}
	return
}

// scrollNotifications returns the scroll position based on the current selection.
func scrollNotifications(m Model, notifications []*github.Notification, direction int) int {
	selectedHeight := m.Selected * listItemHeight
	listHeight := len(notifications)*listItemHeight + 2
	padding := m.Height / 2

	if m.Searching {
		listHeight += 2
	}

	// start of the list, scroll after threshold
	if selectedHeight < padding {
		return 0
	}

	// end of the list scrolling down, stop scrolling
	if direction < 0 && selectedHeight > listHeight-m.Height {
		return listHeight - m.Height
	}

	// end of the list scrolling up, scroll after threshold
	if direction > 0 && selectedHeight > listHeight-padding {
		return listHeight - m.Height
	}

	return selectedHeight - padding
}

// getNotificationsByRepo returns notifications by owner and repo name.
func getNotificationsByRepo(notifications []*github.Notification, owner, repo string) (filtered []*github.Notification) {
	for _, n := range notifications {
		o, r := ownerRepo(n)
		if o == owner && r == repo {
			filtered = append(filtered, n)
		}
	}
	return
}

// removeNotification by id if present.
func removeNotification(notifications []*github.Notification, id string) []*github.Notification {
	i := getNotificationIndex(notifications, id)
	if i == -1 {
		return notifications
	}
	return append(notifications[:i], notifications[i+1:]...)
}

// getNotificationIndex returns the index of a notification, or -1.
func getNotificationIndex(notifications []*github.Notification, id string) int {
	index := -1
	for i, n := range notifications {
		if n.GetID() == id {
			index = i
		}
	}
	return index
}

// filterPriorityLabels returns priorities filtered from labels.
func filterPriorityLabels(labels []*github.Label, priorities []Priority) (filtered []*github.Label) {
loop:
	for _, l := range labels {
		for _, p := range priorities {
			if l.GetName() == p.Label {
				continue loop
			}
		}

		filtered = append(filtered, l)
	}
	return
}

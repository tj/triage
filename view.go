package triage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/kr/text"
	"github.com/tj/go-tea/input"

	"github.com/aybabtme/rgbterm"
	"github.com/dustin/go-humanize"
	"github.com/kyokomi/emoji"
	"github.com/tj/go-css/csshex"
	"github.com/tj/go-tea"
	"github.com/tj/go-tea/option"
	"github.com/tj/go-tea/options"
	"github.com/tj/go-tea/shortcut"
	"github.com/tj/go-termd"

	"github.com/tj/triage/internal/colors"
)

// defaultTheme is the default code syntax highlighting theme.
var defaultTheme = termd.SyntaxTheme{
	"comment": termd.Style{
		Color: "#323232",
	},
	"literal": termd.Style{
		Color: "#555555",
	},
	"name": termd.Style{
		Color: "#777777",
	},
	"name.function": termd.Style{
		Color: "#444444",
	},
	"literal.string": termd.Style{
		Color: "#333333",
	},
}

// key is a shortcut key.
type key struct {
	Key  string
	Help string
}

// View function.
func View(ctx context.Context, model tea.Model) string {
	switch m := model.(Model); m.Page {
	case PageNotifications:
		return viewNotifications(ctx, m)
	case PageNotification:
		return viewNotification(ctx, m)
	case PageComment:
		return viewComment(ctx, m)
	case PageLabels:
		return viewLabels(ctx, m)
	case PagePriorities:
		return viewPriorities(ctx, m)
	default:
		panic("unhandled page")
	}
}

// viewNotifications page.
func viewNotifications(ctx context.Context, m Model) string {
	w := new(bytes.Buffer)

	// loading
	if m.Loading {
		return loading(m)
	}

	// no notifications
	if len(m.Notifications) == 0 {
		return centered(m, "Looks like you're all done 😊")
	}

	// padding
	defer padding(w)()

	// search focused
	if m.Searching {
		fmt.Fprintf(w, "  Searching: %s\r\n\r\n", input.View(m.SearchInput))
	}

	// search blurred
	if !m.Searching && m.SearchInput.Value != "" {
		fmt.Fprintf(w, "  Searching: %s\r\n\r\n", m.SearchInput.Value)
	}

	// sort by updated time asc
	sort.Slice(m.Notifications, func(i, j int) bool {
		a := m.Notifications[i]
		b := m.Notifications[j]
		return a.GetUpdatedAt().After(b.GetUpdatedAt())
	})

	// filter
	filtered := filterNotifications(m.Notifications, m.SearchInput.Value)

	// notifications
	for i, n := range filtered {
		// title
		if m.Selected == i {
			fmt.Fprintf(w, "  * %s\r\n", colors.Bold(n.Repository.GetFullName()))
		} else {
			fmt.Fprintf(w, "    %s\r\n", colors.Bold(n.Repository.GetFullName()))
		}

		// marking as read
		if m.MarkingAsRead && m.Selected == i {
			fmt.Fprintf(w, "    \033[32mMarking as read.\033[0m\r\n\r\n\r\n")
			continue
		}

		// unsubscribing
		if m.Unsubscribing && m.Selected == i {
			fmt.Fprintf(w, "    \033[32mUnsubscribing.\033[0m\r\n\r\n\r\n")
			continue
		}

		// unwatching
		if m.Unwatching && m.Selected == i {
			fmt.Fprintf(w, "    \033[32mUnwatching.\033[0m\r\n\r\n\r\n")
			continue
		}

		// subject
		fmt.Fprintf(w, "    %s\r\n", n.Subject.GetTitle())

		// updated time
		fmt.Fprintf(w, "    Updated %s (%s)\r\n", humanize.Time(n.GetUpdatedAt()), n.GetReason())
		fmt.Fprintf(w, "\r\n")
	}
	fmt.Fprintf(w, "\r\n")

	// viewport
	var offset int
	if m.Searching || m.SearchInput.Value != "" {
		offset = 3
	}
	s := viewport(w.String(), m.NotificationsScrollY, m.Height, offset)

	// menu
	if m.Searching {
		s = menu(s, m,
			shortcut.Key{"Esc", "Abort"},
			shortcut.Key{"Enter", "Save"})
	} else {
		s = menu(s, m,
			shortcut.Key{"q", "Quit"},
			shortcut.Key{"→", "View"},
			shortcut.Key{"↑↓", "Scroll"},
			shortcut.Key{"r", "Mark read"},
			shortcut.Key{"u", "Unsubscribe"},
			shortcut.Key{"U", "Unwatch"},
			shortcut.Key{"R", "Refresh"},
			shortcut.Key{"/", "Search"})
	}

	return s
}

// viewNotification page.
func viewNotification(ctx context.Context, m Model) string {
	config := MustConfigFromContext(ctx)
	theme := config.Theme.Code

	w := new(bytes.Buffer)

	n := m.Notification
	issue := m.Issue
	labels := m.Labels
	comments := m.Comments

	// padding
	defer padding(w)()

	// header
	fmt.Fprintf(w, "    %s\r\n", colors.Bold(n.Repository.GetFullName()))
	fmt.Fprintf(w, "    %s\r\n", n.Subject.GetTitle())
	if issue == nil {
		fmt.Fprintf(w, "\r\n")
	} else {
		fmt.Fprintf(w, "    Opened %s by @%s\r\n", humanize.Time(issue.GetCreatedAt()), issue.GetUser().GetLogin())
	}

	// pending
	switch {
	case m.LoadingIssue:
		fmt.Fprintf(w, "\r\n%s\r\n\r\n", hr())
		fmt.Fprintf(w, "    Loading\r\n")
		return w.String()
	case m.MarkingAsRead:
		fmt.Fprintf(w, "\r\n%s\r\n\r\n", hr())
		fmt.Fprintf(w, "    Marking as read\r\n")
		return w.String()
	case m.Unsubscribing:
		fmt.Fprintf(w, "\r\n%s\r\n\r\n", hr())
		fmt.Fprintf(w, "    Unsubscribing\r\n")
		return w.String()
	case m.Unwatching:
		fmt.Fprintf(w, "\r\n%s\r\n\r\n", hr())
		fmt.Fprintf(w, "    Unwatching\r\n")
		return w.String()
	}

	// labels
	if len(labels) > 0 {
		fmt.Fprintf(w, "    ")
		for _, l := range labels {
			r, g, b, ok := csshex.Parse(l.GetColor())
			if !ok {
				continue
			}
			name := fmt.Sprintf(" %s ", l.GetName())
			name = rgbterm.BgString(name, r, g, b)
			name = rgbterm.FgString(name, 0, 0, 0)
			emoji.Fprintf(w, "%s ", name)
		}
		fmt.Fprintf(w, "\r\n")
	}

	// body
	fmt.Fprintf(w, "\r\n%s\r\n\r\n", hr())
	if body := issue.GetBody(); body == "" {
		fmt.Fprintf(w, "    No description provided.\r\n")
	} else {
		fmt.Fprintf(w, "%s", text.Indent(markdownText(body, theme), "    "))
	}

	// comments
	fmt.Fprintf(w, "\r\n")
	fmt.Fprintf(w, "%s\r\n", hr())
	for i, c := range comments {
		fmt.Fprintf(w, "\r\n")
		fmt.Fprintf(w, "    %s %s\r\n\r\n", colors.Bold("@"+c.GetUser().GetLogin()), humanize.Time(c.GetCreatedAt()))
		fmt.Fprintf(w, "%s", text.Indent(markdownText(c.GetBody(), theme), "    "))
		if i < len(comments)-1 {
			fmt.Fprintf(w, "%s\r\n", hr())
		}
	}
	fmt.Fprintf(w, "\r\n")

	// viewport
	offset := 7
	if len(labels) > 0 {
		offset = 8
	}
	s := viewport(w.String(), m.NotificationScrollY, m.Height, offset)

	// menu
	s = menu(s, m,
		shortcut.Key{"q", "Quit"},
		shortcut.Key{"←", "Back"},
		shortcut.Key{"↑↓", "Scroll"},
		shortcut.Key{"r", "Mark read"},
		shortcut.Key{"u", "Unsubscribe"},
		shortcut.Key{"c", "Comment"},
		shortcut.Key{"l", "Labels"},
		shortcut.Key{"p", "Priority"},
		shortcut.Key{"o", "Open"},
		shortcut.Key{"R", "Refresh"})

	return s
}

// viewLabels page.
func viewLabels(ctx context.Context, m Model) string {
	w := new(bytes.Buffer)

	// loading
	if m.Loading || m.LoadingLabels {
		return loading(m)
	}

	// padding
	defer padding(w)()

	fmt.Fprintf(w, "  Press space to select labels:\r\n\r\n")
	fmt.Fprintf(w, "%s", options.View(m.LabelOptions))

	return menu(w.String(), m,
		shortcut.Key{"Esc", "Abort"},
		shortcut.Key{"Space", "Toggle"},
		shortcut.Key{"Enter", "Save"})
}

// viewPriorities page.
func viewPriorities(ctx context.Context, m Model) string {
	w := new(bytes.Buffer)

	// padding
	defer padding(w)()

	fmt.Fprintf(w, "  Select a priority:\r\n\r\n")
	fmt.Fprintf(w, "%s", option.View(m.PriorityOptions))

	return menu(w.String(), m,
		shortcut.Key{"Esc", "Abort"},
		shortcut.Key{"Space", "Toggle"},
		shortcut.Key{"Enter", "Save"})
}

// viewComment page.
func viewComment(ctx context.Context, m Model) string {
	w := new(bytes.Buffer)

	// padding
	defer padding(w)()

	fmt.Fprintf(w, "  Press enter to save your comment:\r\n\r\n")
	fmt.Fprintf(w, "  %s", input.View(m.CommentInput))

	return menu(w.String(), m,
		shortcut.Key{"Esc", "Abort"},
		shortcut.Key{"Enter", "Save"})
}

// loading indicator.
func loading(m Model) string {
	if m.Height == 0 {
		return ""
	}
	return centered(m, "Loading")
}

// centered text.
func centered(m Model, s string) string {
	y := strings.Repeat("\r\n", (m.Height/2)-1)
	x := strings.Repeat(" ", (m.Width/2)-(len(s)/2))
	return y + x + s
}

// menu view.
func menu(s string, m Model, keys ...shortcut.Key) string {
	// TODO: refactor this stuff using a nicer box model
	if m.Height == 0 {
		return ""
	}
	lines := strings.Split(s, "\r\n")
	for i := len(lines); i < m.Height; i++ {
		lines = append(lines, "")
	}
	lines[len(lines)-2] = strings.Repeat(" ", m.Width)
	lines[len(lines)-1] = shortcut.View(shortcut.Model{keys})
	return strings.Join(lines, "\r\n")
}

// viewport returns a view into the lines of text, providing
// the scroll offset, height of the viewport, and offset
// which retains N lines behaving like a "sticky" header.
func viewport(s string, scroll, height, offset int) string {
	lines := strings.Split(s, "\r\n")

	// offset
	leading := lines[:offset]
	lines = lines[offset:]

	// view
	from := scroll
	to := scroll + height - offset
	lines = append(leading, bounded(lines, from, to)...)

	return strings.Join(lines, "\r\n")
}

// bounded slice.
func bounded(s []string, from, to int) []string {
	from = max(0, min(from, len(s)))
	to = max(0, min(to, len(s)))
	return s[from:to]
}

// markdownText helper.
func markdownText(s string, theme *termd.SyntaxTheme) string {
	var md termd.Compiler

	if theme == nil {
		md.SyntaxHighlighter = defaultTheme
	} else {
		md.SyntaxHighlighter = *theme
	}

	// blackfriday's markdown parser only supports a single rune as the linebreak
	s = strings.Replace(s, "\r\n", "\n", -1)

	// tabs -> spaces
	s = strings.Replace(s, "\t", "  ", -1)

	// compile and apply emoji support
	s = md.Compile(s)
	s = strings.Replace(s, "\n", "\r\n", -1)
	s = emoji.Sprintf("%s", s)
	return s
}

// padding util.
func padding(w io.Writer) func() {
	fmt.Fprintf(w, "\r\n")
	return func() {
		fmt.Fprintf(w, "\r\n")
	}
}

// hr is a horizontal rule.
func hr() string {
	return fmt.Sprintf("    \033[38;5;102m%s\033[0m", strings.Repeat("─", 90))
}

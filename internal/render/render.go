package render

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/Olyxz16/ytcli/internal/model"
	"github.com/Olyxz16/ytcli/internal/store"
)

var (
	titleStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	labelStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	idStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Bold(true)
	resolvedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	unresolvedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	headerStyle     = lipgloss.NewStyle().Bold(true).Underline(true)
)

// OutputMode controls how data is rendered.
type OutputMode string

const (
	OutputTable    OutputMode = "table"
	OutputJSON     OutputMode = "json"
	OutputWide     OutputMode = "wide"
	OutputMarkdown OutputMode = "markdown"
)

// JSON marshals v to indented JSON and writes to stdout.
func JSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// JSONQuiet prints minimal JSON for agents.
func JSONQuiet(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	return enc.Encode(v)
}

// IssueList renders a list of issues.
func IssueList(issues []model.Issue, mode OutputMode) error {
	switch mode {
	case OutputJSON:
		return JSON(issues)
	default:
		return issueListTable(issues)
	}
}

func issueListTable(issues []model.Issue) error {
	if len(issues) == 0 {
		fmt.Println("No issues found.")
		return nil
	}

	// Determine column widths
	maxID := 10
	maxSummary := 30
	for _, i := range issues {
		if len(i.IDReadable) > maxID {
			maxID = len(i.IDReadable)
		}
		if len(i.Summary) > maxSummary {
			maxSummary = len(i.Summary)
		}
	}

	fmt.Println(headerStyle.Render(fmt.Sprintf(
		"%-*s  %-*s  %s",
		maxID, "ID",
		maxSummary, "Summary",
		"State",
	)))

	for _, i := range issues {
		state := "Open"
		stateCol := unresolvedStyle
		if i.Resolved != nil {
			state = "Resolved"
			stateCol = resolvedStyle
		}
		fmt.Printf(
			"%s  %s  %s\n",
			idStyle.Render(fmt.Sprintf("%-*s", maxID, i.IDReadable)),
			truncate(i.Summary, maxSummary),
			stateCol.Render(state),
		)
	}
	return nil
}

// IssueDetail renders a single issue in detail.
func IssueDetail(issue *model.Issue, mode OutputMode) error {
	switch mode {
	case OutputJSON:
		return JSON(issue)
	default:
		return issueDetailText(issue)
	}
}

func issueDetailText(issue *model.Issue) error {
	fmt.Println(titleStyle.Render(issue.IDReadable + ": " + issue.Summary))
	fmt.Println()

	fmt.Printf("%s %s\n", labelStyle.Render("Project:"), issue.Project.Name)
	if issue.Reporter != nil {
		fmt.Printf("%s %s\n", labelStyle.Render("Reporter:"), issue.Reporter.DisplayName())
	}
	if issue.Updater != nil {
		fmt.Printf("%s %s\n", labelStyle.Render("Updated by:"), issue.Updater.DisplayName())
	}
	fmt.Printf("%s %s\n", labelStyle.Render("Created:"), formatTime(issue.Created))
	fmt.Printf("%s %s\n", labelStyle.Render("Updated:"), formatTime(issue.Updated))
	if issue.Resolved != nil {
		fmt.Printf("%s %s\n", labelStyle.Render("Resolved:"), formatTime(*issue.Resolved))
	}
	fmt.Printf("%s %d\n", labelStyle.Render("Votes:"), issue.Votes)
	fmt.Printf("%s %d\n", labelStyle.Render("Comments:"), issue.CommentsCount)

	if len(issue.Tags) > 0 {
		var tags []string
		for _, t := range issue.Tags {
			tags = append(tags, t.Name)
		}
		fmt.Printf("%s %s\n", labelStyle.Render("Tags:"), strings.Join(tags, ", "))
	}

	if len(issue.CustomFields) > 0 {
		fmt.Println()
		fmt.Println(headerStyle.Render("Custom Fields"))
		for _, cf := range issue.CustomFields {
			fmt.Printf("  %s %s\n", labelStyle.Render(cf.Name+":"), cf.StringValue())
		}
	}

	if issue.Description != "" {
		fmt.Println()
		fmt.Println(headerStyle.Render("Description"))
		rendered, err := RenderMarkdown(issue.Description)
		if err == nil {
			fmt.Println(rendered)
		} else {
			fmt.Println(issue.Description)
		}
	}

	if len(issue.Comments) > 0 {
		fmt.Println()
		fmt.Println(headerStyle.Render("Comments"))
		for _, c := range issue.Comments {
			fmt.Printf("  %s %s\n", labelStyle.Render(c.Author.DisplayName()+":"), c.Text)
		}
	}

	return nil
}

// CommentList renders a list of comments.
func CommentList(comments []model.Comment, mode OutputMode) error {
	switch mode {
	case OutputJSON:
		return JSON(comments)
	default:
		for _, c := range comments {
			fmt.Printf(
				"%s %s\n  %s\n\n",
				labelStyle.Render(c.Author.DisplayName()),
				labelStyle.Render(formatTime(c.Created)),
				c.Text,
			)
		}
		return nil
	}
}

// ProjectList renders a list of projects.
func ProjectList(projects []model.Project, mode OutputMode) error {
	switch mode {
	case OutputJSON:
		return JSON(projects)
	default:
		for _, p := range projects {
			fmt.Printf("%s  %s\n", idStyle.Render(p.ShortName), p.Name)
		}
		return nil
	}
}

// User renders a user.
func User(user *model.User, mode OutputMode) error {
	switch mode {
	case OutputJSON:
		return JSON(user)
	default:
		fmt.Printf("%s %s (%s)\n", labelStyle.Render("User:"), user.DisplayName(), user.Login)
		if user.Email != "" {
			fmt.Printf("%s %s\n", labelStyle.Render("Email:"), user.Email)
		}
		return nil
	}
}

// CommandResult renders command execution output.
func CommandResult(result *model.CommandResult, mode OutputMode) error {
	switch mode {
	case OutputJSON:
		return JSON(result)
	default:
		fmt.Printf("Command: %s\n", result.Query)
		if len(result.Issues) > 0 {
			fmt.Println("Applied to:")
			for _, i := range result.Issues {
				fmt.Printf("  %s %s\n", idStyle.Render(i.IDReadable), i.Summary)
			}
		}
		return nil
	}
}

// WorkItemList renders work items.
func WorkItemList(items []model.WorkItem, mode OutputMode) error {
	switch mode {
	case OutputJSON:
		return JSON(items)
	default:
		for _, wi := range items {
			dur := ""
			if wi.Duration != nil {
				dur = wi.Duration.Presentation
			}
			fmt.Printf("%s  %s  %s\n", labelStyle.Render(formatTime(wi.Date)), dur, wi.Text)
		}
		return nil
	}
}

// LocalIssueList renders a list of local issues.
func LocalIssueList(issues []store.LocalIssue, mode OutputMode) error {
	switch mode {
	case OutputJSON:
		return JSON(issues)
	default:
		return localIssueListTable(issues)
	}
}

func localIssueListTable(issues []store.LocalIssue) error {
	if len(issues) == 0 {
		fmt.Println("No issues found.")
		return nil
	}

	maxID := 10
	maxSummary := 30
	for _, i := range issues {
		displayID := fmt.Sprintf("#%d", i.ID)
		if i.RemoteID != nil {
			displayID = *i.RemoteID
		}
		if len(displayID) > maxID {
			maxID = len(displayID)
		}
		if len(i.Summary) > maxSummary {
			maxSummary = len(i.Summary)
		}
	}

	fmt.Println(headerStyle.Render(fmt.Sprintf(
		"%-*s  %-*s  %-6s  %s",
		maxID, "ID",
		maxSummary, "Summary",
		"State",
		"Sync",
	)))

	for _, i := range issues {
		displayID := fmt.Sprintf("#%d", i.ID)
		if i.RemoteID != nil {
			displayID = *i.RemoteID
		}

		stateCol := unresolvedStyle
		if i.State == "Done" || i.State == "Closed" || i.State == "Resolved" {
			stateCol = resolvedStyle
		}

		syncIndicator := ""
		switch i.SyncStatus {
		case "local":
			syncIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Render("local")
		case "modified":
			syncIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render("modified")
		case "synced":
			syncIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("synced")
		case "conflict":
			syncIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("CONFLICT")
		}

		fmt.Printf(
			"%s  %s  %s  %s\n",
			idStyle.Render(fmt.Sprintf("%-*s", maxID, displayID)),
			truncate(i.Summary, maxSummary),
			stateCol.Render(fmt.Sprintf("%-6s", i.State)),
			syncIndicator,
		)
	}
	return nil
}

// LocalIssueDetail renders a single local issue in detail.
func LocalIssueDetail(issue *store.LocalIssue, mode OutputMode) error {
	switch mode {
	case OutputJSON:
		return JSON(issue)
	default:
		return localIssueDetailText(issue)
	}
}

func localIssueDetailText(issue *store.LocalIssue) error {
	displayID := fmt.Sprintf("#%d", issue.ID)
	if issue.RemoteID != nil {
		displayID = *issue.RemoteID
	}
	fmt.Println(titleStyle.Render(displayID + ": " + issue.Summary))
	fmt.Println()

	fmt.Printf("%s %s\n", labelStyle.Render("State:"), issue.State)
	fmt.Printf("%s %s\n", labelStyle.Render("Priority:"), issue.Priority)
	if issue.Assignee != nil {
		fmt.Printf("%s %s\n", labelStyle.Render("Assignee:"), *issue.Assignee)
	}
	fmt.Printf("%s %s\n", labelStyle.Render("Created:"), issue.CreatedAt.Format("2006-01-02 15:04"))
	fmt.Printf("%s %s\n", labelStyle.Render("Updated:"), issue.UpdatedAt.Format("2006-01-02 15:04"))
	fmt.Printf("%s %s\n", labelStyle.Render("Sync:"), issue.SyncStatus)

	if len(issue.Tags) > 0 {
		fmt.Printf("%s %s\n", labelStyle.Render("Tags:"), strings.Join(issue.Tags, ", "))
	}
	if issue.Comments > 0 {
		fmt.Printf("%s %d\n", labelStyle.Render("Comments:"), issue.Comments)
	}

	if issue.Description != "" {
		fmt.Println()
		fmt.Println(headerStyle.Render("Description"))
		rendered, err := RenderMarkdown(issue.Description)
		if err == nil {
			fmt.Println(rendered)
		} else {
			fmt.Println(issue.Description)
		}
	}

	return nil
}

// LocalCommentList renders a list of local comments.
func LocalCommentList(comments []store.LocalComment, mode OutputMode) error {
	switch mode {
	case OutputJSON:
		return JSON(comments)
	default:
		for _, c := range comments {
			fmt.Printf(
				"%s %s\n  %s\n\n",
				labelStyle.Render(c.Author),
				labelStyle.Render(c.CreatedAt.Format("2006-01-02 15:04")),
				c.Text,
			)
		}
		return nil
	}
}

// Header renders a section header.
func Header(text string) string {
	return headerStyle.Render(text)
}

// Error renders a structured error.
func Error(err error, mode OutputMode) {
	switch mode {
	case OutputJSON:
		_ = JSON(map[string]interface{}{
			"error":   err.Error(),
			"message": err.Error(),
		})
	default:
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
	}
}

func formatTime(ms int64) string {
	t := time.UnixMilli(ms)
	return t.Format("2006-01-02 15:04")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// RenderMarkdown renders markdown text to terminal-friendly output.
func RenderMarkdown(text string) (string, error) {
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(120),
	)
	if err != nil {
		return "", err
	}
	return r.Render(text)
}

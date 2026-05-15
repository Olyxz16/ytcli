package tui

import (
	"database/sql"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/glamour"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Olyxz16/tkt/internal/config"
	"github.com/Olyxz16/tkt/internal/local"
	"github.com/Olyxz16/tkt/internal/service"
	"github.com/Olyxz16/tkt/internal/store"
)

// App is the TUI application.
type App struct {
	service *service.Service
	cfg     *config.MergedConfig
	providerErr string
}

// NewApp creates a new TUI app.
func NewApp(svc *service.Service, cfg *config.MergedConfig, providerErr string) *App {
	return &App{service: svc, cfg: cfg, providerErr: providerErr}
}

// Run starts the TUI.
func (a *App) Run() error {
	if !store.IsInitialized() {
		return fmt.Errorf("no local project found. Run: tkt init")
	}

	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	localCfg, _, err := config.LoadLocal()
	if err != nil {
		return err
	}
	schema := local.GetSchema(localCfg)

	m := newModel(a.service, a.cfg, a.providerErr, db, schema)
	prog := tea.NewProgram(m, tea.WithAltScreen())
	_, err = prog.Run()
	return err
}

type viewMode int

const (
	viewList viewMode = iota
	viewBoard
)

type focusArea int

const (
	focusList focusArea = iota
	focusDetail
)

type model struct {
	service *service.Service
	cfg     *config.MergedConfig
	db      *sql.DB
	schema  config.LocalSchema
	providerErr string

	issues       []store.LocalIssue
	filtered     []int
	selected     int
	filterText   string
	filterInput  textinput.Model
	filterActive bool
	viewMode     viewMode
	focus        focusArea

	width  int
	height int

	status string

	showComments    bool
	commentsIssueID int64
	comments        []store.LocalComment
	commentsErr     string

	paletteActive bool
	paletteInput  textinput.Model
	paletteItems  []paletteItem
	paletteIndex  int

	formActive  bool
	formType    string
	formIssueID int64
	formMsg     string
	formInput   textinput.Model
	formArea    textarea.Model
	md        *glamour.TermRenderer
}

type paletteItem struct {
	Label      string
	Cmd        string
	NeedsInput bool
}

type issuesMsg struct {
	Issues []store.LocalIssue
}

type errMsg struct {
	Err error
}

type commentsMsg struct {
	IssueID  int64
	Comments []store.LocalComment
	Err      error
}

func newModel(svc *service.Service, cfg *config.MergedConfig, providerErr string, db *sql.DB, schema config.LocalSchema) model {
	filter := textinput.New()
	filter.Placeholder = "filter"
	filter.Prompt = "/ "
	filter.CharLimit = 200
	filter.Width = 30

	palette := textinput.New()
	palette.Placeholder = "command"
	palette.Prompt = ": "
	palette.CharLimit = 200
	palette.Width = 40

	formInput := textinput.New()
	formInput.CharLimit = 200
	formInput.Width = 60

	formArea := textarea.New()
	formArea.CharLimit = 2000
	formArea.SetHeight(5)
	formArea.SetWidth(60)
	formArea.ShowLineNumbers = false
	formArea.Prompt = ""

	md, _ := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(78),
	)

	return model{
		service:       svc,
		cfg:           cfg,
		db:            db,
		schema:        schema,
		providerErr:   providerErr,
		filterInput:   filter,
		paletteInput:  palette,
		viewMode:      viewList,
		focus:         focusList,
		paletteItems:  defaultPaletteItems(schema),
		status:        "Ready",
		md:            md,
		paletteIndex:  0,
		selected:      0,
		formInput:     formInput,
		formArea:      formArea,
		showComments:  true,
	}
}

func (m model) Init() tea.Cmd {
	return m.refreshIssuesCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case issuesMsg:
		m.issues = msg.Issues
		m.applyFilter()
		return m, m.loadSelectedCommentsCmd()
	case commentsMsg:
		m.commentsIssueID = msg.IssueID
		m.commentsErr = ""
		if msg.Err != nil {
			m.comments = nil
			m.commentsErr = msg.Err.Error()
			m.status = msg.Err.Error()
			return m, nil
		}
		m.comments = msg.Comments
		return m, nil
	case errMsg:
		m.status = msg.Err.Error()
		return m, nil
	case tea.KeyMsg:
		if m.formActive {
			return m.handleFormKeys(msg)
		}
		if m.paletteActive {
			return m.handlePaletteKeys(msg)
		}
		if m.filterActive {
			return m.handleFilterKeys(msg)
		}
		return m.handleMainKeys(msg)
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	content := ""
	if m.viewMode == viewBoard {
		content = m.viewBoard()
	} else {
		content = m.viewList()
	}
	status := m.viewStatusBar()
	if m.paletteActive {
		return content + "\n" + m.viewPalette() + "\n" + status
	}
	if m.formActive {
		return content + "\n" + m.viewForm() + "\n" + status
	}
	return content + "\n" + status
}

func (m model) refreshIssuesCmd() tea.Cmd {
	return func() tea.Msg {
		issues, err := store.ListIssues(m.db, "", 1000)
		if err != nil {
			return errMsg{Err: err}
		}
		_ = store.LoadIssueTags(m.db, issues)
		_ = store.LoadIssueComments(m.db, issues)
		return issuesMsg{Issues: issues}
	}
}

func (m *model) applyFilter() {
	if m.filterText == "" {
		m.filtered = m.filtered[:0]
		for i := range m.issues {
			m.filtered = append(m.filtered, i)
		}
		if m.selected >= len(m.filtered) {
			m.selected = len(m.filtered) - 1
		}
		if m.selected < 0 {
			m.selected = 0
		}
		return
	}
	query := strings.ToLower(m.filterText)
	m.filtered = m.filtered[:0]
	for i, issue := range m.issues {
		if strings.Contains(strings.ToLower(issue.Summary), query) ||
			strings.Contains(strings.ToLower(issue.Description), query) {
			m.filtered = append(m.filtered, i)
			continue
		}
		for _, tag := range issue.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				m.filtered = append(m.filtered, i)
				break
			}
		}
	}
	if m.selected >= len(m.filtered) {
		m.selected = len(m.filtered) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
}

func (m model) handleMainKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "/":
		m.filterActive = true
		m.filterInput.SetValue("")
		m.filterInput.CursorEnd()
		m.filterInput.Focus()
		return m, nil
	case ":":
		m.paletteActive = true
		m.paletteIndex = 0
		m.paletteInput.SetValue("")
		m.paletteInput.CursorEnd()
		m.paletteInput.Focus()
		return m, nil
	case "j", "down":
		prevID := m.selectedIssueID()
		m.moveSelection(1)
		return m, m.loadCommentsForSelection(prevID)
	case "k", "up":
		prevID := m.selectedIssueID()
		m.moveSelection(-1)
		return m, m.loadCommentsForSelection(prevID)
	case "g":
		prevID := m.selectedIssueID()
		m.selected = 0
		return m, m.loadCommentsForSelection(prevID)
	case "G":
		prevID := m.selectedIssueID()
		if len(m.filtered) > 0 {
			m.selected = len(m.filtered) - 1
		}
		return m, m.loadCommentsForSelection(prevID)
	case "tab":
		if m.focus == focusList {
			m.focus = focusDetail
		} else {
			m.focus = focusList
		}
		return m, nil
	case "C":
		m.showComments = !m.showComments
		if m.showComments {
			return m, m.loadSelectedCommentsCmd()
		}
		return m, nil
	case "b":
		if m.viewMode == viewBoard {
			m.viewMode = viewList
		} else {
			m.viewMode = viewBoard
		}
		return m, nil
	case "e":
		issue := m.selectedIssue()
		if issue == nil {
			return m, nil
		}
		if issue.SyncStatus != "local" {
			m.status = "Editing synced issues is not supported in TUI yet"
			return m, nil
		}
		m.startFormSummary(*issue)
		return m, nil
	case "c":
		issue := m.selectedIssue()
		if issue == nil {
			return m, nil
		}
		if issue.SyncStatus != "local" {
			m.status = "Commenting on synced issues is not supported in TUI yet"
			return m, nil
		}
		m.startFormComment(*issue)
		return m, nil
	case "E":
		issue := m.selectedIssue()
		if issue == nil {
			return m, nil
		}
		if issue.SyncStatus != "local" {
			m.status = "Editing synced issues is not supported in TUI yet"
			return m, nil
		}
		m.startFormDescription(*issue)
		return m, nil
	}
	return m, nil
}

func (m model) handleFilterKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	switch msg.String() {
	case "enter":
		m.filterText = strings.TrimSpace(m.filterInput.Value())
		m.filterActive = false
		m.filterInput.Blur()
		m.applyFilter()
		return m, nil
	case "esc":
		m.filterActive = false
		m.filterInput.Blur()
		return m, nil
	}
	return m, cmd
}

func (m model) handlePaletteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.paletteInput, cmd = m.paletteInput.Update(msg)
	switch msg.String() {
	case "esc":
		m.paletteActive = false
		m.paletteInput.Blur()
		return m, nil
	case "enter":
		text := strings.TrimSpace(m.paletteInput.Value())
		if text == "" {
			items := m.filteredPaletteItems()
			if len(items) > 0 && m.paletteIndex < len(items) {
				item := items[m.paletteIndex]
				if item.NeedsInput {
					m.paletteInput.SetValue(item.Cmd + " ")
					m.paletteInput.CursorEnd()
					return m, nil
				}
			}
		}
		m.paletteActive = false
		m.paletteInput.Blur()
		return m, m.runPaletteCommand(text)
	case "j", "down":
		if m.paletteIndex < len(m.filteredPaletteItems())-1 {
			m.paletteIndex++
		}
		return m, nil
	case "k", "up":
		if m.paletteIndex > 0 {
			m.paletteIndex--
		}
		return m, nil
	}
	return m, cmd
}

func (m model) handleFormKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.formType == "summary" {
		var cmd tea.Cmd
		m.formInput, cmd = m.formInput.Update(msg)
		switch msg.String() {
		case "enter":
			value := strings.TrimSpace(m.formInput.Value())
			m.endForm()
			if value == "" {
				return m, nil
			}
			return m, m.updateIssueFieldCmd(m.formIssueID, map[string]interface{}{"summary": value})
		case "esc":
			m.endForm()
			return m, nil
		}
		return m, cmd
	}
	if m.formType == "description" || m.formType == "comment" {
		var cmd tea.Cmd
		m.formArea, cmd = m.formArea.Update(msg)
		switch msg.String() {
		case "ctrl+s":
			value := strings.TrimSpace(m.formArea.Value())
			formType := m.formType
			issueID := m.formIssueID
			m.endForm()
			if value == "" {
				return m, nil
			}
			if formType == "comment" {
				return m, m.addCommentCmd(issueID, value)
			}
			return m, m.updateIssueFieldCmd(issueID, map[string]interface{}{"description": value})
		case "esc":
			m.endForm()
			return m, nil
		}
		return m, cmd
	}
	return m, nil
}

func (m *model) moveSelection(delta int) {
	if len(m.filtered) == 0 {
		m.selected = 0
		return
	}
	m.selected += delta
	if m.selected < 0 {
		m.selected = 0
	}
	if m.selected >= len(m.filtered) {
		m.selected = len(m.filtered) - 1
	}
}

func (m model) selectedIssue() *store.LocalIssue {
	if len(m.filtered) == 0 || m.selected < 0 || m.selected >= len(m.filtered) {
		return nil
	}
	idx := m.filtered[m.selected]
	if idx < 0 || idx >= len(m.issues) {
		return nil
	}
	return &m.issues[idx]
}

func (m model) selectedIssueID() int64 {
	issue := m.selectedIssue()
	if issue == nil {
		return 0
	}
	return issue.ID
}

func (m model) viewList() string {
	listW := listWidth(m.width)
	panelHeight := m.height - 2 - m.overlayHeight()
	if panelHeight < 4 {
		panelHeight = 4
	}
	left := m.renderList(listW, panelHeight)
	right := m.renderDetail(detailWidth(m.width), panelHeight)
	return lipglossJoinHorizontal(left, right)
}

func (m model) renderList(width, height int) string {
	items := make([]string, 0, len(m.filtered)+2)
	if m.filterActive {
		items = append(items, inputStyle.Render(m.filterInput.View()))
	}
	for i, idx := range m.filtered {
		issue := m.issues[idx]
		id := local.FormatID(&issue)
		line := fmt.Sprintf("%s %s", id, issue.Summary)
		if issue.SyncStatus == "conflict" {
			line = listConflictStyle.Render(line)
		} else if issue.SyncStatus == "modified" {
			line = listModifiedStyle.Render(line)
		} else if issue.SyncStatus == "synced" {
			line = listSyncedStyle.Render(line)
		}
		if i == m.selected {
			line = listSelectedStyle.Render(line)
		} else {
			line = listItemStyle.Render(line)
		}
		items = append(items, line)
	}
	if len(items) == 0 {
		items = append(items, listHelpStyle.Render("No issues"))
	}
	content := strings.Join(items, "\n")
	return listStyle.Width(width).Height(height).Render(content)
}

func (m model) renderDetail(width, height int) string {
	issue := m.selectedIssue()
	if issue == nil {
		content := detailTitleStyle.Render("No issue selected")
		return detailStyle.Width(width).Height(height).Render(content)
	}

	meta := fmt.Sprintf("%s · %s · %s", issue.State, issue.Priority, issue.SyncStatus)
	meta = detailMetaStyle.Render(meta)

	assignee := "Unassigned"
	if issue.Assignee != nil && *issue.Assignee != "" {
		assignee = *issue.Assignee
	}
	labels := []string{
		detailLabelStyle.Render("Assignee:") + " " + detailValueStyle.Render(assignee),
		detailLabelStyle.Render("Tags:") + " " + detailValueStyle.Render(strings.Join(issue.Tags, ", ")),
		detailLabelStyle.Render("Comments:") + " " + detailValueStyle.Render(fmt.Sprintf("%d", issue.Comments)),
	}

	md := issue.Description
	if md == "" {
		md = "(no description)"
	}
	if strings.TrimSpace(md) == "" {
		md = "(no description)"
	}
	body := md
	if m.md != nil {
		if rendered, err := m.md.Render(md); err == nil {
			body = rendered
		}
	}

	commentsSection := ""
	if m.showComments {
		commentsSection = "\n\n" + commentSectionTitleStyle.Render("Comments")
		if issue.Comments == 0 {
			commentsSection += "\n" + commentEmptyStyle.Render("(no comments)")
		} else if m.commentsIssueID != issue.ID {
			commentsSection += "\n" + commentEmptyStyle.Render("Loading comments...")
		} else if m.commentsErr != "" {
			commentsSection += "\n" + commentEmptyStyle.Render(m.commentsErr)
		} else {
			for _, comment := range m.comments {
				author := strings.TrimSpace(comment.Author)
				if author == "" {
					author = "Unknown"
				}
				header := fmt.Sprintf("%s · %s", author, comment.CreatedAt.Format("2006-01-02 15:04"))
				commentsSection += "\n" + commentMetaStyle.Render(header)
				commentsSection += "\n" + commentBodyStyle.Render(comment.Text)
				commentsSection += "\n" + commentDividerStyle.Render(strings.Repeat("-", 20))
			}
		}
	}

	content := detailTitleStyle.Render(issue.Summary) + "\n" + meta + "\n" + strings.Join(labels, "\n") + "\n\n" + body + commentsSection
	return detailStyle.Width(width).Height(height).Render(content)
}

func (m model) viewBoard() string {
	columns := m.schema.States
	if len(columns) == 0 {
		columns = []string{"Open"}
	}
	colWidth := boardColumnWidth(m.width-2, len(columns))
	var cols []string
	for _, state := range columns {
		items := m.issuesByState(state)
		content := boardColumnHeader.Render(fmt.Sprintf("%s (%d)", state, len(items)))
		for _, issue := range items {
			line := fmt.Sprintf("%s %s", local.FormatID(&issue), issue.Summary)
			content += "\n" + boardItemStyle.Render(line)
		}
		height := m.height - 2 - m.overlayHeight()
		if height < 4 {
			height = 4
		}
		cols = append(cols, boardColumnStyle.Width(colWidth).Height(height).Render(content))
	}
	return lipglossJoinHorizontal(cols...)
}

func (m model) issuesByState(state string) []store.LocalIssue {
	var out []store.LocalIssue
	for _, idx := range m.filtered {
		issue := m.issues[idx]
		if issue.State == state {
			out = append(out, issue)
		}
	}
	return out
}

func (m model) viewStatusBar() string {
	filter := ""
	if m.filterText != "" {
		filter = fmt.Sprintf("filter: %s", m.filterText)
	}
	left := statusKeyStyle.Render("TKT") + statusValStyle.Render(fmt.Sprintf("%d issues", len(m.filtered)))
	if filter != "" {
		left += statusValStyle.Render(filter)
	}
	providerStatus := m.providerStatusText()
	if providerStatus != "" {
		left += statusValStyle.Render(providerStatus)
	}
	keys := "q quit  / filter  : command  e summary  E desc  c comment  C comments  b board"
	if m.filterActive {
		keys = "enter apply  esc cancel"
	}
	if m.paletteActive {
		keys = "enter run  esc cancel  j/k move"
	}
	if m.formActive {
		keys = "ctrl+s save  esc cancel"
	}
	right := statusValStyle.Render(m.status)
	return statusBarStyle.Render(left + "  " + keys + "  " + right)
}

func (m model) providerStatusText() string {
	if m.cfg == nil || m.cfg.ProviderURL == "" {
		return "provider: offline"
	}
	if m.providerErr != "" || m.service == nil {
		return "provider: auth required"
	}
	return "provider: connected"
}

func (m *model) startFormSummary(issue store.LocalIssue) {
	m.formActive = true
	m.formType = "summary"
	m.formIssueID = issue.ID
	m.formInput.SetValue(issue.Summary)
	m.formInput.CursorEnd()
	m.formInput.Focus()
	m.formArea.Blur()
	m.formMsg = "Edit summary"
}

func (m *model) startFormDescription(issue store.LocalIssue) {
	m.formActive = true
	m.formType = "description"
	m.formIssueID = issue.ID
	m.formArea.SetValue(issue.Description)
	m.formArea.CursorEnd()
	m.formArea.Focus()
	m.formInput.Blur()
	m.formMsg = "Edit description (ctrl+s to save)"
}

func (m *model) startFormComment(issue store.LocalIssue) {
	m.formActive = true
	m.formType = "comment"
	m.formIssueID = issue.ID
	m.formArea.SetValue("")
	m.formArea.CursorEnd()
	m.formArea.Focus()
	m.formInput.Blur()
	m.formMsg = "Add comment (ctrl+s to save)"
}

func (m *model) endForm() {
	m.formActive = false
	m.formType = ""
	m.formMsg = ""
	m.formIssueID = 0
	m.formInput.Blur()
	m.formArea.Blur()
}

func (m model) viewForm() string {
	if !m.formActive {
		return ""
	}
	width := m.width - 2
	content := detailTitleStyle.Render(m.formMsg) + "\n"
	if m.formType == "summary" {
		content += inputStyle.Render(m.formInput.View())
	} else {
		content += m.formArea.View()
	}
	return detailStyle.Width(width).Render(content)
}

func (m model) viewPalette() string {
	items := m.filteredPaletteItems()
	maxItems := paletteMaxItems
	if maxItems <= 0 {
		maxItems = 5
	}
	start := 0
	if len(items) > maxItems {
		if m.paletteIndex > maxItems/2 {
			start = m.paletteIndex - maxItems/2
		}
		if start+maxItems > len(items) {
			start = len(items) - maxItems
		}
		if start < 0 {
			start = 0
		}
		items = items[start : start+maxItems]
	}
	lines := make([]string, 0, len(items)+1)
	for i, item := range items {
		line := item.Label
		if i+start == m.paletteIndex {
			line = paletteMatchStyle.Render(line)
		}
		lines = append(lines, line)
	}
	lines = append(lines, inputStyle.Render(m.paletteInput.View()))
	return paletteStyle.Height(paletteOverlayHeight).Render(strings.Join(lines, "\n"))
}

func (m model) overlayHeight() int {
	if m.formActive || m.paletteActive {
		return paletteOverlayHeight
	}
	if m.filterActive {
		return 1
	}
	return 0
}

func (m model) runPaletteCommand(text string) tea.Cmd {
	cmd := strings.TrimSpace(text)
	if cmd == "" {
		items := m.filteredPaletteItems()
		if len(items) == 0 || m.paletteIndex >= len(items) {
			return nil
		}
		cmd = items[m.paletteIndex].Cmd
	}
	return m.runCommand(cmd)
}

func (m model) filteredPaletteItems() []paletteItem {
	query := strings.ToLower(strings.TrimSpace(m.paletteInput.Value()))
	if query == "" {
		return m.paletteItems
	}
	var out []paletteItem
	for _, item := range m.paletteItems {
		if strings.Contains(strings.ToLower(item.Label), query) || strings.Contains(strings.ToLower(item.Cmd), query) {
			out = append(out, item)
		}
	}
	return out
}

func (m model) runCommand(cmd string) tea.Cmd {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return nil
	}
	switch fields[0] {
	case "sync":
		return m.refreshIssuesCmd()
	case "state":
		if len(fields) < 2 {
			return errorCmd("usage: :state <state>")
		}
		issue := m.selectedIssue()
		if issue == nil {
			return errorCmd("no issue selected")
		}
		if issue.SyncStatus != "local" {
			return errorCmd("editing synced issues is not supported in TUI yet")
		}
		state := strings.Join(fields[1:], " ")
		return m.updateIssueFieldCmd(issue.ID, map[string]interface{}{"state": state})
	case "priority":
		if len(fields) < 2 {
			return errorCmd("usage: :priority <priority>")
		}
		issue := m.selectedIssue()
		if issue == nil {
			return errorCmd("no issue selected")
		}
		if issue.SyncStatus != "local" {
			return errorCmd("editing synced issues is not supported in TUI yet")
		}
		prio := strings.Join(fields[1:], " ")
		return m.updateIssueFieldCmd(issue.ID, map[string]interface{}{"priority": prio})
	case "assign":
		if len(fields) < 2 {
			return errorCmd("usage: :assign <login>")
		}
		issue := m.selectedIssue()
		if issue == nil {
			return errorCmd("no issue selected")
		}
		if issue.SyncStatus != "local" {
			return errorCmd("editing synced issues is not supported in TUI yet")
		}
		assignee := strings.TrimPrefix(fields[1], "@")
		if assignee == "me" {
			return errorCmd("assign me is not supported locally")
		}
		return m.updateIssueFieldCmd(issue.ID, map[string]interface{}{"assignee": assignee})
	case "tag":
		if len(fields) < 2 {
			return errorCmd("usage: :tag <name>")
		}
		issue := m.selectedIssue()
		if issue == nil {
			return errorCmd("no issue selected")
		}
		tag := strings.Join(fields[1:], " ")
		if issue.SyncStatus != "local" {
			return errorCmd("tagging synced issues is not supported in TUI yet")
		}
		return m.addTagCmd(issue.ID, tag)
	case "untag":
		if len(fields) < 2 {
			return errorCmd("usage: :untag <name>")
		}
		issue := m.selectedIssue()
		if issue == nil {
			return errorCmd("no issue selected")
		}
		tag := strings.Join(fields[1:], " ")
		if issue.SyncStatus != "local" {
			return errorCmd("tag removal for synced issues is not supported in TUI yet")
		}
		return m.removeTagCmd(issue.ID, tag)
	case "done":
		issue := m.selectedIssue()
		if issue == nil {
			return errorCmd("no issue selected")
		}
		if issue.SyncStatus != "local" {
			return errorCmd("editing synced issues is not supported in TUI yet")
		}
		doneState := "Done"
		if len(m.schema.DoneStates) > 0 {
			doneState = m.schema.DoneStates[0]
		}
		return m.updateIssueFieldCmd(issue.ID, map[string]interface{}{"state": doneState})
	case "open":
		issue := m.selectedIssue()
		if issue == nil {
			return errorCmd("no issue selected")
		}
		if issue.ProviderRef == "" || m.cfg == nil || m.cfg.ProviderURL == "" {
			return errorCmd("no remote link for issue")
		}
		return openBrowserCmd(fmt.Sprintf("%s/issue/%s", m.cfg.ProviderURL, issue.ProviderRef))
	case "refresh":
		return m.refreshIssuesCmd()
	case "summary":
		if len(fields) < 2 {
			return errorCmd("usage: :summary <text>")
		}
		issue := m.selectedIssue()
		if issue == nil {
			return errorCmd("no issue selected")
		}
		if issue.SyncStatus != "local" {
			return errorCmd("editing synced issues is not supported in TUI yet")
		}
		value := strings.Join(fields[1:], " ")
		return m.updateIssueFieldCmd(issue.ID, map[string]interface{}{"summary": value})
	case "comment":
		if len(fields) < 2 {
			return errorCmd("usage: :comment <text>")
		}
		issue := m.selectedIssue()
		if issue == nil {
			return errorCmd("no issue selected")
		}
		if issue.SyncStatus != "local" {
			return errorCmd("commenting on synced issues is not supported in TUI yet")
		}
		value := strings.Join(fields[1:], " ")
		return m.addCommentCmd(issue.ID, value)
	case "help":
		return nil
	default:
		return errorCmd(fmt.Sprintf("unknown command: %s", fields[0]))
	}
}

func (m model) updateIssueFieldCmd(id int64, updates map[string]interface{}) tea.Cmd {
	return func() tea.Msg {
		_, err := store.UpdateIssue(m.db, id, updates)
		if err != nil {
			return errMsg{Err: err}
		}
		return issuesMsg{Issues: m.reloadIssues()}
	}
}

func (m model) addTagCmd(id int64, tag string) tea.Cmd {
	return func() tea.Msg {
		if err := store.AddIssueTag(m.db, id, tag); err != nil {
			return errMsg{Err: err}
		}
		return issuesMsg{Issues: m.reloadIssues()}
	}
}

func (m model) addCommentCmd(id int64, text string) tea.Cmd {
	return func() tea.Msg {
		_, err := store.CreateComment(m.db, id, text)
		if err != nil {
			return errMsg{Err: err}
		}
		return issuesMsg{Issues: m.reloadIssues()}
	}
}

func (m model) removeTagCmd(id int64, tag string) tea.Cmd {
	return func() tea.Msg {
		if err := store.RemoveIssueTag(m.db, id, tag); err != nil {
			return errMsg{Err: err}
		}
		return issuesMsg{Issues: m.reloadIssues()}
	}
}

func (m model) reloadIssues() []store.LocalIssue {
	issues, err := store.ListIssues(m.db, "", 1000)
	if err != nil {
		m.status = err.Error()
		return m.issues
	}
	_ = store.LoadIssueTags(m.db, issues)
	_ = store.LoadIssueComments(m.db, issues)
	return issues
}

func (m model) loadCommentsCmd(issueID int64) tea.Cmd {
	if issueID == 0 {
		return nil
	}
	return func() tea.Msg {
		comments, err := store.ListComments(m.db, issueID)
		if err != nil {
			return commentsMsg{IssueID: issueID, Err: err}
		}
		return commentsMsg{IssueID: issueID, Comments: comments}
	}
}

func (m model) loadSelectedCommentsCmd() tea.Cmd {
	if !m.showComments {
		return nil
	}
	issue := m.selectedIssue()
	if issue == nil {
		return nil
	}
	return m.loadCommentsCmd(issue.ID)
}

func (m model) loadCommentsForSelection(prevID int64) tea.Cmd {
	if !m.showComments {
		return nil
	}
	issue := m.selectedIssue()
	if issue == nil {
		return nil
	}
	if issue.ID == prevID {
		return nil
	}
	return m.loadCommentsCmd(issue.ID)
}

func defaultPaletteItems(schema config.LocalSchema) []paletteItem {
	items := []paletteItem{
		{Label: "refresh", Cmd: "refresh"},
		{Label: "done", Cmd: "done"},
		{Label: "assign me", Cmd: "assign", NeedsInput: true},
		{Label: "tag <name>", Cmd: "tag", NeedsInput: true},
		{Label: "untag <name>", Cmd: "untag", NeedsInput: true},
		{Label: "open", Cmd: "open"},
		{Label: "summary <text>", Cmd: "summary", NeedsInput: true},
		{Label: "comment <text>", Cmd: "comment", NeedsInput: true},
	}
	for _, state := range schema.States {
		items = append(items, paletteItem{Label: "state " + state, Cmd: "state " + state})
	}
	for _, prio := range schema.Priorities {
		items = append(items, paletteItem{Label: "priority " + prio, Cmd: "priority " + prio})
	}
	return items
}

func openBrowserCmd(url string) tea.Cmd {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	return tea.ExecProcess(exec.Command(cmd, args...), func(err error) tea.Msg {
		if err != nil {
			return errMsg{Err: err}
		}
		return nil
	})
}

func errorCmd(msg string) tea.Cmd {
	return func() tea.Msg {
		return errMsg{Err: fmt.Errorf("%s", msg)}
	}
}

func lipglossJoinHorizontal(parts ...string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

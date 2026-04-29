package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kaysush-twilio/identity-explorer/internal/dynamo"
	"github.com/kaysush-twilio/identity-explorer/internal/models"
)

// AppState represents the current state of the application
type AppState int

const (
	StateModeSelection AppState = iota
	StateProfileInput
	StateIdentifierInput
	StateCountInput
	StateLoading
	StateProfileResult
	StateIdentifierResult
	StateProfileSelection
	StateCountResult
	StateError
)

// QueryMode represents the type of query being performed
type QueryMode int

const (
	ModeQueryProfile QueryMode = iota
	ModeQueryIdentifier
	ModeCountProfiles
	ModeUnset QueryMode = -1
)

// Config holds CLI options to prefill the UI
type Config struct {
	Mode       string // "profile" or "identifier"
	AccountID  string
	StoreID    string
	ProfileID  string
	IDType     string
	IDValue    string
}

// Model represents the application state
type Model struct {
	state          AppState
	mode           QueryMode
	err            error
	width          int
	height         int

	// Mode selection
	modeOptions    []string
	selectedMode   int

	// Input fields for profile query
	profileInputs  []textinput.Model
	profileFocused int

	// Input fields for identifier query
	identifierInputs  []textinput.Model
	identifierFocused int

	// Input fields for count query
	countInputs  []textinput.Model
	countFocused int

	// Loading state
	spinner    spinner.Model
	loadingMsg string

	// Results
	queryResult      *models.QueryResult
	mappingsTable    table.Model
	mergesTable      table.Model
	focusedTable     int // 0 = mappings, 1 = merges

	// Profile selection (for identifier query with multiple matches)
	profileMatches   []string
	selectedProfile  int

	// Count results
	countResults     []dynamo.ShardCountResult
	countTotal       int
	countComplete    bool

	// Error viewport for scrolling long errors
	errorViewport viewport.Model
}

// Message types
type queryResultMsg struct {
	result *models.QueryResult
}

type errorMsg struct {
	err error
}

type profilesFoundMsg struct {
	profiles []string
}

type countShardResultMsg struct {
	result dynamo.ShardCountResult
}

// NewModel creates a new application model
func NewModel() Model {
	return NewModelWithConfig(Config{})
}

// NewModelWithConfig creates a new application model with prefilled values
func NewModelWithConfig(cfg Config) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))

	// Profile inputs: AccountID, StoreID, ProfileID
	profileInputs := make([]textinput.Model, 3)
	for i := range profileInputs {
		ti := textinput.New()
		ti.CharLimit = 156
		ti.Width = 50
		profileInputs[i] = ti
	}
	profileInputs[0].Placeholder = "Account ID (e.g., AC...)"
	profileInputs[1].Placeholder = "Store ID"
	profileInputs[2].Placeholder = "Profile ID"

	// Prefill profile inputs from config
	if cfg.AccountID != "" {
		profileInputs[0].SetValue(cfg.AccountID)
	}
	if cfg.StoreID != "" {
		profileInputs[1].SetValue(cfg.StoreID)
	}
	if cfg.ProfileID != "" {
		profileInputs[2].SetValue(cfg.ProfileID)
	}

	// Identifier inputs: AccountID, StoreID, IDType, IDValue
	identifierInputs := make([]textinput.Model, 4)
	for i := range identifierInputs {
		ti := textinput.New()
		ti.CharLimit = 256
		ti.Width = 50
		identifierInputs[i] = ti
	}
	identifierInputs[0].Placeholder = "Account ID (e.g., AC...)"
	identifierInputs[1].Placeholder = "Store ID"
	identifierInputs[2].Placeholder = "ID Type (e.g., email, phone)"
	identifierInputs[3].Placeholder = "ID Value (e.g., user@example.com)"

	// Prefill identifier inputs from config
	if cfg.AccountID != "" {
		identifierInputs[0].SetValue(cfg.AccountID)
	}
	if cfg.StoreID != "" {
		identifierInputs[1].SetValue(cfg.StoreID)
	}
	if cfg.IDType != "" {
		identifierInputs[2].SetValue(cfg.IDType)
	}
	if cfg.IDValue != "" {
		identifierInputs[3].SetValue(cfg.IDValue)
	}

	// Count inputs: StoreID only
	countInputs := make([]textinput.Model, 1)
	countInputs[0] = textinput.New()
	countInputs[0].CharLimit = 156
	countInputs[0].Width = 50
	countInputs[0].Placeholder = "Store ID"
	if cfg.StoreID != "" {
		countInputs[0].SetValue(cfg.StoreID)
	}

	// Determine initial state based on config
	initialState := StateModeSelection
	initialMode := ModeUnset
	selectedMode := 0

	switch strings.ToLower(cfg.Mode) {
	case "profile":
		initialState = StateProfileInput
		initialMode = ModeQueryProfile
		selectedMode = 0
		profileInputs[0].Focus()
	case "identifier":
		initialState = StateIdentifierInput
		initialMode = ModeQueryIdentifier
		selectedMode = 1
		identifierInputs[0].Focus()
	case "count":
		initialState = StateCountInput
		initialMode = ModeCountProfiles
		selectedMode = 2
		countInputs[0].Focus()
	default:
		profileInputs[0].Focus()
	}

	// Error viewport
	vp := viewport.New(80, 10)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#EF4444")).
		Padding(1)

	return Model{
		state:            initialState,
		mode:             initialMode,
		modeOptions:      []string{"Query Profile", "Query Identifier", "Count Store Profiles"},
		selectedMode:     selectedMode,
		profileInputs:    profileInputs,
		identifierInputs: identifierInputs,
		countInputs:      countInputs,
		spinner:          s,
		errorViewport:    vp,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle global keys first
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.state != StateModeSelection {
				m.state = StateModeSelection
				m.err = nil
				return m, nil
			}
			return m, tea.Quit
		}

		// For input states, handle navigation keys but let other keys pass through to inputs
		switch m.state {
		case StateModeSelection:
			return m.handleModeSelection(msg)
		case StateProfileInput:
			return m.handleProfileInput(msg)
		case StateIdentifierInput:
			return m.handleIdentifierInput(msg)
		case StateCountInput:
			return m.handleCountInput(msg)
		case StateProfileSelection:
			return m.handleProfileSelection(msg)
		case StateProfileResult, StateIdentifierResult:
			return m.handleResultNavigation(msg)
		case StateCountResult:
			if msg.String() == "q" || msg.String() == "esc" {
				m.state = StateModeSelection
				m.countResults = nil
				m.countTotal = 0
				m.countComplete = false
			}
			return m, nil
		case StateError:
			// Press any key to go back from error
			if msg.String() == "q" {
				m.state = StateModeSelection
				m.err = nil
			}
			return m, nil
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case queryResultMsg:
		m.queryResult = msg.result
		if msg.result.Error != nil {
			m.err = msg.result.Error
			m.state = StateError
		} else {
			m.state = StateProfileResult
			m.mappingsTable = m.createMappingsTable(msg.result.Mappings)
			m.mergesTable = m.createMergesTable(msg.result.Merges)
		}
		return m, nil

	case profilesFoundMsg:
		if len(msg.profiles) == 0 {
			m.err = fmt.Errorf("no profiles found for the given identifier")
			m.state = StateError
			return m, nil
		}
		if len(msg.profiles) == 1 {
			// Single match, query directly
			m.profileMatches = msg.profiles
			return m, m.queryProfileData(msg.profiles[0])
		}
		// Multiple matches, show selection
		m.profileMatches = msg.profiles
		m.selectedProfile = 0
		m.state = StateProfileSelection
		return m, nil

	case errorMsg:
		m.err = msg.err
		m.state = StateError
		return m, nil

	case countShardResultMsg:
		m.countResults = append(m.countResults, msg.result)
		if msg.result.Error == nil {
			m.countTotal += msg.result.Count
		}
		// Mark complete when all 11 shards have been queried
		if len(m.countResults) >= 11 {
			m.countComplete = true
		}
		return m, nil
	}

	// Handle input updates based on state
	switch m.state {
	case StateProfileInput:
		return m.updateProfileInputs(msg)
	case StateIdentifierInput:
		return m.updateIdentifierInputs(msg)
	case StateCountInput:
		return m.updateCountInputs(msg)
	case StateProfileResult, StateIdentifierResult:
		return m.updateResultView(msg)
	}

	return m, nil
}


func (m Model) handleModeSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedMode > 0 {
			m.selectedMode--
		}
	case "down", "j":
		if m.selectedMode < len(m.modeOptions)-1 {
			m.selectedMode++
		}
	case "enter":
		m.mode = QueryMode(m.selectedMode)
		switch m.mode {
		case ModeQueryProfile:
			m.state = StateProfileInput
			m.profileInputs[0].Focus()
		case ModeQueryIdentifier:
			m.state = StateIdentifierInput
			m.identifierInputs[0].Focus()
		case ModeCountProfiles:
			m.state = StateCountInput
			m.countInputs[0].Focus()
		}
		return m, textinput.Blink
	}
	return m, nil
}

func (m Model) handleProfileInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.profileInputs[m.profileFocused].Blur()
		m.profileFocused = (m.profileFocused + 1) % len(m.profileInputs)
		m.profileInputs[m.profileFocused].Focus()
		return m, textinput.Blink
	case "shift+tab":
		m.profileInputs[m.profileFocused].Blur()
		m.profileFocused--
		if m.profileFocused < 0 {
			m.profileFocused = len(m.profileInputs) - 1
		}
		m.profileInputs[m.profileFocused].Focus()
		return m, textinput.Blink
	case "enter":
		// Validate inputs
		accountID := strings.TrimSpace(m.profileInputs[0].Value())
		storeID := strings.TrimSpace(m.profileInputs[1].Value())
		profileID := strings.TrimSpace(m.profileInputs[2].Value())

		if accountID == "" || storeID == "" || profileID == "" {
			m.err = fmt.Errorf("all fields are required")
			m.state = StateError
			return m, nil
		}

		m.state = StateLoading
		m.loadingMsg = "Querying profile data..."
		return m, m.queryProfileData(profileID)
	}

	// Forward other key events to the focused text input
	var cmd tea.Cmd
	m.profileInputs[m.profileFocused], cmd = m.profileInputs[m.profileFocused].Update(msg)
	return m, cmd
}

func (m Model) handleIdentifierInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.identifierInputs[m.identifierFocused].Blur()
		m.identifierFocused = (m.identifierFocused + 1) % len(m.identifierInputs)
		m.identifierInputs[m.identifierFocused].Focus()
		return m, textinput.Blink
	case "shift+tab":
		m.identifierInputs[m.identifierFocused].Blur()
		m.identifierFocused--
		if m.identifierFocused < 0 {
			m.identifierFocused = len(m.identifierInputs) - 1
		}
		m.identifierInputs[m.identifierFocused].Focus()
		return m, textinput.Blink
	case "enter":
		// Validate inputs
		accountID := strings.TrimSpace(m.identifierInputs[0].Value())
		storeID := strings.TrimSpace(m.identifierInputs[1].Value())
		idType := strings.TrimSpace(m.identifierInputs[2].Value())
		idValue := strings.TrimSpace(m.identifierInputs[3].Value())

		if accountID == "" || storeID == "" || idType == "" || idValue == "" {
			m.err = fmt.Errorf("all fields are required")
			m.state = StateError
			return m, nil
		}

		m.state = StateLoading
		m.loadingMsg = "Looking up profiles..."
		return m, m.queryIdentifier()
	}

	// Forward other key events to the focused text input
	var cmd tea.Cmd
	m.identifierInputs[m.identifierFocused], cmd = m.identifierInputs[m.identifierFocused].Update(msg)
	return m, cmd
}

func (m Model) handleProfileSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedProfile > 0 {
			m.selectedProfile--
		}
	case "down", "j":
		if m.selectedProfile < len(m.profileMatches)-1 {
			m.selectedProfile++
		}
	case "enter":
		m.state = StateLoading
		m.loadingMsg = "Querying profile data..."
		return m, m.queryProfileData(m.profileMatches[m.selectedProfile])
	}
	return m, nil
}

func (m Model) handleCountInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		storeID := strings.TrimSpace(m.countInputs[0].Value())
		if storeID == "" {
			m.err = fmt.Errorf("Store ID is required")
			m.state = StateError
			return m, nil
		}

		// Reset count state
		m.countResults = nil
		m.countTotal = 0
		m.countComplete = false
		m.state = StateCountResult
		return m, m.queryStoreProfileCount(storeID)
	}

	// Forward other key events to the focused text input
	var cmd tea.Cmd
	m.countInputs[m.countFocused], cmd = m.countInputs[m.countFocused].Update(msg)
	return m, cmd
}

func (m Model) handleResultNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		// Toggle between mappings and merges tables
		m.focusedTable = (m.focusedTable + 1) % 2
		if m.focusedTable == 0 {
			m.mappingsTable.Focus()
			m.mergesTable.Blur()
		} else {
			m.mappingsTable.Blur()
			m.mergesTable.Focus()
		}
		return m, nil
	}

	// Forward navigation to the focused table
	var cmd tea.Cmd
	if m.focusedTable == 0 {
		m.mappingsTable, cmd = m.mappingsTable.Update(msg)
	} else {
		m.mergesTable, cmd = m.mergesTable.Update(msg)
	}
	return m, cmd
}

func (m Model) updateProfileInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, len(m.profileInputs))
	for i := range m.profileInputs {
		m.profileInputs[i], cmds[i] = m.profileInputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m Model) updateIdentifierInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, len(m.identifierInputs))
	for i := range m.identifierInputs {
		m.identifierInputs[i], cmds[i] = m.identifierInputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m Model) updateCountInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, len(m.countInputs))
	for i := range m.countInputs {
		m.countInputs[i], cmds[i] = m.countInputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m Model) updateResultView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.focusedTable == 0 {
		m.mappingsTable, cmd = m.mappingsTable.Update(msg)
	} else {
		m.mergesTable, cmd = m.mergesTable.Update(msg)
	}
	return m, cmd
}

func (m Model) queryProfileData(profileID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		client, err := dynamo.NewClient(ctx)
		if err != nil {
			return errorMsg{err: fmt.Errorf("failed to create DynamoDB client: %w", err)}
		}

		var accountID, storeID string
		if m.mode == ModeQueryProfile {
			accountID = strings.TrimSpace(m.profileInputs[0].Value())
			storeID = strings.TrimSpace(m.profileInputs[1].Value())
		} else {
			accountID = strings.TrimSpace(m.identifierInputs[0].Value())
			storeID = strings.TrimSpace(m.identifierInputs[1].Value())
		}

		// Step 1: Query merges to get canonical link and all merged profile IDs
		merges, canonicalLink, err := client.QueryAllMergesForProfile(ctx, accountID, storeID, profileID)
		if err != nil {
			return queryResultMsg{result: &models.QueryResult{ProfileID: profileID, Error: err}}
		}

		// Step 2: Build list of all profile IDs to query mappings for
		// This includes the canonical profile + all merged profiles
		profileIDsToQuery := []string{profileID}
		if canonicalLink != nil {
			// Add canonical profile if different from input
			if canonicalLink.CanonicalProfileID != profileID {
				profileIDsToQuery = append(profileIDsToQuery, canonicalLink.CanonicalProfileID)
			}
			// Add all merged profiles
			for _, mergedID := range canonicalLink.MergedProfileIDs {
				if mergedID != "" && mergedID != profileID {
					profileIDsToQuery = append(profileIDsToQuery, mergedID)
				}
			}
		}

		// Step 3: Query mappings for all profiles in parallel
		mappings, err := client.QueryMappingsForMultipleProfiles(ctx, storeID, profileIDsToQuery)
		if err != nil {
			return queryResultMsg{result: &models.QueryResult{ProfileID: profileID, Merges: merges, CanonicalLink: canonicalLink, Error: err}}
		}

		return queryResultMsg{
			result: &models.QueryResult{
				ProfileID:     profileID,
				Mappings:      mappings,
				Merges:        merges,
				CanonicalLink: canonicalLink,
			},
		}
	}
}

func (m Model) queryIdentifier() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		client, err := dynamo.NewClient(ctx)
		if err != nil {
			return errorMsg{err: fmt.Errorf("failed to create DynamoDB client: %w", err)}
		}

		storeID := strings.TrimSpace(m.identifierInputs[1].Value())
		idType := strings.TrimSpace(m.identifierInputs[2].Value())
		idValue := strings.TrimSpace(m.identifierInputs[3].Value())

		profiles, err := client.QueryProfileIDsByIdentifier(ctx, storeID, idType, idValue)
		if err != nil {
			return errorMsg{err: fmt.Errorf("failed to query profiles: %w", err)}
		}

		return profilesFoundMsg{profiles: profiles}
	}
}

func (m Model) queryStoreProfileCount(storeID string) tea.Cmd {
	// Generate all shard values
	shards := make([]string, 11)
	shards[0] = storeID
	for i := 0; i < 10; i++ {
		shards[i+1] = fmt.Sprintf("%s#%d", storeID, i)
	}

	// Create a shared client and semaphore for rate limiting
	ctx := context.Background()
	client, err := dynamo.NewClient(ctx)
	if err != nil {
		return func() tea.Msg {
			return errorMsg{err: fmt.Errorf("failed to create DynamoDB client: %w", err)}
		}
	}

	// Rate limit to 3 concurrent queries to avoid hitting DynamoDB too hard
	semaphore := make(chan struct{}, 3)

	// Create commands for each shard query
	cmds := make([]tea.Cmd, len(shards))
	for i, shard := range shards {
		s := shard // capture for closure
		cmds[i] = func() tea.Msg {
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			count, err := client.CountProfilesInStoreByShard(ctx, s)
			return countShardResultMsg{
				result: dynamo.ShardCountResult{
					Shard: s,
					Count: count,
					Error: err,
				},
			}
		}
	}

	return tea.Batch(cmds...)
}

func (m Model) createMappingsTable(mappings []models.Mapping) table.Model {
	columns := []table.Column{
		{Title: "Profile ID", Width: 38},
		{Title: "ID Type", Width: 12},
		{Title: "ID Value", Width: 35},
		{Title: "Unique", Width: 6},
	}

	rows := make([]table.Row, len(mappings))
	for i, mapping := range mappings {
		unique := "No"
		if mapping.IsUnique {
			unique = "Yes"
		}
		rows[i] = table.Row{
			mapping.ProfileID,
			mapping.IDType,
			mapping.IDValue,
			unique,
		}
	}

	// Calculate height - show more rows, up to 15
	height := len(rows) + 1
	if height > 15 {
		height = 15
	}
	if height < 3 {
		height = 3
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7C3AED")).
		Bold(false)
	t.SetStyles(s)

	return t
}

func (m Model) createMergesTable(merges []models.Merge) table.Model {
	columns := []table.Column{
		{Title: "Merge From", Width: 38},
		{Title: "Merge To", Width: 38},
		{Title: "Canonical (CPID)", Width: 38},
		{Title: "Reason", Width: 10},
	}

	rows := make([]table.Row, len(merges))
	for i, merge := range merges {
		rows[i] = table.Row{
			merge.MergeFrom,
			merge.MergeTo,
			merge.CanonicalProfileID,
			merge.Reason,
		}
	}

	// Calculate height - show more rows, up to 15
	height := len(rows) + 1
	if height > 15 {
		height = 15
	}
	if height < 3 {
		height = 3
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(height),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#10B981")).
		BorderBottom(true).
		Bold(true)
	t.SetStyles(s)

	return t
}

// View renders the UI
func (m Model) View() string {
	var b strings.Builder

	title := TitleStyle.Render("Identity Explorer")
	b.WriteString(title + "\n\n")

	switch m.state {
	case StateModeSelection:
		b.WriteString(m.renderModeSelection())
	case StateProfileInput:
		b.WriteString(m.renderProfileInput())
	case StateIdentifierInput:
		b.WriteString(m.renderIdentifierInput())
	case StateCountInput:
		b.WriteString(m.renderCountInput())
	case StateLoading:
		b.WriteString(m.renderLoading())
	case StateProfileResult, StateIdentifierResult:
		b.WriteString(m.renderResults())
	case StateProfileSelection:
		b.WriteString(m.renderProfileSelection())
	case StateCountResult:
		b.WriteString(m.renderCountResult())
	case StateError:
		b.WriteString(m.renderError())
	}

	b.WriteString("\n" + HelpStyle.Render("Press 'q' or 'esc' to go back • 'ctrl+c' to quit"))

	return b.String()
}

func (m Model) renderModeSelection() string {
	var b strings.Builder
	b.WriteString(SubtitleStyle.Render("Select a query mode:") + "\n\n")

	for i, option := range m.modeOptions {
		cursor := "  "
		style := UnselectedStyle
		if i == m.selectedMode {
			cursor = "> "
			style = SelectedStyle
		}
		b.WriteString(cursor + style.Render(option) + "\n")
	}

	return b.String()
}

func (m Model) renderProfileInput() string {
	var b strings.Builder
	b.WriteString(SubtitleStyle.Render("Query Profile - Enter details:") + "\n\n")

	labels := []string{"Account ID:", "Store ID:", "Profile ID:"}
	for i, input := range m.profileInputs {
		b.WriteString(InputLabelStyle.Render(labels[i]) + "\n")
		if i == m.profileFocused {
			b.WriteString(FocusedInputStyle.Render(input.View()) + "\n\n")
		} else {
			b.WriteString(InputStyle.Render(input.View()) + "\n\n")
		}
	}

	b.WriteString(HelpStyle.Render("Tab/Arrow keys to navigate • Enter to submit"))
	return b.String()
}

func (m Model) renderIdentifierInput() string {
	var b strings.Builder
	b.WriteString(SubtitleStyle.Render("Query Identifier - Enter details:") + "\n\n")

	labels := []string{"Account ID:", "Store ID:", "ID Type:", "ID Value:"}
	for i, input := range m.identifierInputs {
		b.WriteString(InputLabelStyle.Render(labels[i]) + "\n")
		if i == m.identifierFocused {
			b.WriteString(FocusedInputStyle.Render(input.View()) + "\n\n")
		} else {
			b.WriteString(InputStyle.Render(input.View()) + "\n\n")
		}
	}

	b.WriteString(HelpStyle.Render("Tab/Arrow keys to navigate • Enter to submit"))
	return b.String()
}

func (m Model) renderCountInput() string {
	var b strings.Builder
	b.WriteString(SubtitleStyle.Render("Count Store Profiles - Enter Store ID:") + "\n\n")
	b.WriteString(SubtitleStyle.Render("This will query 11 shards (storeId, storeId#0-9) in parallel.") + "\n\n")

	b.WriteString(InputLabelStyle.Render("Store ID:") + "\n")
	b.WriteString(FocusedInputStyle.Render(m.countInputs[0].View()) + "\n\n")

	b.WriteString(HelpStyle.Render("Enter to submit"))
	return b.String()
}

func (m Model) renderCountResult() string {
	var b strings.Builder

	storeID := strings.TrimSpace(m.countInputs[0].Value())
	b.WriteString(SubtitleStyle.Render(fmt.Sprintf("Profile Count for Store: %s", storeID)) + "\n\n")

	// Show spinner if still counting
	if !m.countComplete {
		b.WriteString(m.spinner.View() + " Counting profiles across shards...\n\n")
	}

	// Show intermediate results
	if len(m.countResults) > 0 {
		b.WriteString(InputLabelStyle.Render("Shard Results:") + "\n")

		// Sort results by shard name for consistent display
		sortedResults := make([]dynamo.ShardCountResult, len(m.countResults))
		copy(sortedResults, m.countResults)
		sortShardResults(sortedResults, storeID)

		var firstError error
		for _, result := range sortedResults {
			status := "✓"
			countStr := fmt.Sprintf("%d", result.Count)
			if result.Error != nil {
				status = "✗"
				countStr = "error"
				if firstError == nil {
					firstError = result.Error
				}
			}
			b.WriteString(fmt.Sprintf("  %s %s: %s\n", status, result.Shard, countStr))
		}

		// Show first error details if any
		if firstError != nil {
			b.WriteString("\n" + ErrorStyle.Render("Error details:") + "\n")
			errMsg := fmt.Sprintf("%v", firstError)
			b.WriteString(wrapText(errMsg, 80) + "\n")
		}
		b.WriteString("\n")
	}

	// Show running total
	b.WriteString(SelectedStyle.Render(fmt.Sprintf("Running Total: %d profiles", m.countTotal)) + "\n")
	b.WriteString(SubtitleStyle.Render(fmt.Sprintf("Shards queried: %d/11", len(m.countResults))) + "\n")

	if m.countComplete {
		b.WriteString("\n" + WarningStyle.Render(fmt.Sprintf("Final Count: %d profiles", m.countTotal)) + "\n")
	}

	return b.String()
}

func sortShardResults(results []dynamo.ShardCountResult, storeID string) {
	// Sort by: storeID first, then storeID#0, storeID#1, ..., storeID#9
	shardOrder := func(shard string) int {
		if shard == storeID {
			return -1 // Base storeID comes first
		}
		// Extract shard number from storeID#N
		if strings.HasPrefix(shard, storeID+"#") {
			suffix := strings.TrimPrefix(shard, storeID+"#")
			if n, err := fmt.Sscanf(suffix, "%d", new(int)); err == nil && n == 1 {
				var num int
				fmt.Sscanf(suffix, "%d", &num)
				return num
			}
		}
		return 100 // Unknown format, sort to end
	}

	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if shardOrder(results[i].Shard) > shardOrder(results[j].Shard) {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

func (m Model) renderLoading() string {
	return fmt.Sprintf("\n%s %s\n", m.spinner.View(), m.loadingMsg)
}

func (m Model) renderResults() string {
	var b strings.Builder

	if m.queryResult == nil {
		return "No results"
	}

	// Profile info header
	profileInfo := fmt.Sprintf("Profile ID: %s", m.queryResult.ProfileID)
	b.WriteString(SelectedStyle.Render(profileInfo) + "\n\n")

	// Canonical link info
	if m.queryResult.CanonicalLink != nil {
		b.WriteString(WarningStyle.Render("Canonical Profile: "+m.queryResult.CanonicalLink.CanonicalProfileID) + "\n")
		if len(m.queryResult.CanonicalLink.MergedProfileIDs) > 0 {
			mergeCount := len(m.queryResult.CanonicalLink.MergedProfileIDs)
			b.WriteString(SubtitleStyle.Render(fmt.Sprintf("Merged Profiles: %d profiles merged into this canonical", mergeCount)) + "\n")
		}
		b.WriteString("\n")
	}

	// Mappings table
	mappingsTitle := "Mappings"
	if m.focusedTable == 0 {
		mappingsTitle = "> Mappings"
	}
	b.WriteString(TitleStyle.Render(mappingsTitle) + fmt.Sprintf(" (%d items)\n", len(m.queryResult.Mappings)))
	if len(m.queryResult.Mappings) == 0 {
		b.WriteString(SubtitleStyle.Render("  No mappings found") + "\n\n")
	} else {
		b.WriteString(m.mappingsTable.View() + "\n\n")
	}

	// Merges table
	mergesTitle := "Merges"
	if m.focusedTable == 1 {
		mergesTitle = "> Merges"
	}
	b.WriteString(TitleStyle.Render(mergesTitle) + fmt.Sprintf(" (%d items)\n", len(m.queryResult.Merges)))
	if len(m.queryResult.Merges) == 0 {
		b.WriteString(SubtitleStyle.Render("  No merges found") + "\n")
	} else {
		b.WriteString(m.mergesTable.View() + "\n")
	}

	b.WriteString("\n" + HelpStyle.Render("Tab to switch tables • Arrow keys to scroll • q/esc to go back"))

	return b.String()
}

func (m Model) renderProfileSelection() string {
	var b strings.Builder
	b.WriteString(SubtitleStyle.Render(fmt.Sprintf("Found %d profiles matching the identifier:", len(m.profileMatches))) + "\n\n")

	for i, profile := range m.profileMatches {
		cursor := "  "
		style := UnselectedStyle
		if i == m.selectedProfile {
			cursor = "> "
			style = SelectedStyle
		}
		b.WriteString(cursor + style.Render(profile) + "\n")
	}

	b.WriteString("\n" + HelpStyle.Render("Use arrow keys to select • Enter to view profile"))
	return b.String()
}

func (m Model) renderError() string {
	var b strings.Builder
	b.WriteString(ErrorStyle.Render("Error occurred:") + "\n\n")

	// Show full error message with word wrapping
	errMsg := fmt.Sprintf("%+v", m.err)

	// Wrap long lines for readability
	maxWidth := 80
	if m.width > 20 {
		maxWidth = m.width - 10
	}
	wrapped := wrapText(errMsg, maxWidth)

	b.WriteString(lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#EF4444")).
		Padding(1).
		Width(maxWidth).
		Render(wrapped))

	b.WriteString("\n\n" + HelpStyle.Render("Press 'q' or 'esc' to go back"))
	return b.String()
}

// wrapText wraps text to the specified width
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}

		for len(line) > width {
			// Find last space before width
			breakPoint := width
			for j := width; j > 0; j-- {
				if line[j] == ' ' {
					breakPoint = j
					break
				}
			}
			result.WriteString(line[:breakPoint])
			result.WriteString("\n")
			line = strings.TrimLeft(line[breakPoint:], " ")
		}
		result.WriteString(line)
	}

	return result.String()
}

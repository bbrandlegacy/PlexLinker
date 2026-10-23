package v2tui

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"plexlinker/internal/browser"
	"plexlinker/internal/config"
	"plexlinker/internal/downloads"
	"plexlinker/internal/storage"
)

type Tab int

const (
	TabRemote Tab = iota
	TabSelection
	TabDestination
	TabDownloads
	TabSettings
)

type selectedFile struct{ URL, Name string }

type Model struct {
	services           Services
	tab                Tab
	width, height      int
	loading            bool
	status             string
	remotePath         string
	remoteItems        []browser.DirectoryItem
	remoteCursor       int
	selected           map[string]selectedFile
	volumes            []storage.Volume
	destinationPath    string
	destinationEntries []storage.Entry
	destinationCursor  int
	destinationInput   textinput.Model
	destinationPrompt  string
	jobs               []downloads.Job
	downloadCursor     int
	progress           progress.Model
	settingsInputs     []textinput.Model
	settingsCursor     int
}

type remoteLoadedMsg struct {
	listing *browser.DirectoryListing
	err     error
}
type recursiveSelectedMsg struct {
	files []selectedFile
	err   error
}
type volumesLoadedMsg struct {
	volumes []storage.Volume
	err     error
}
type directoryLoadedMsg struct {
	path    string
	entries []storage.Entry
	err     error
}
type jobsLoadedMsg struct {
	jobs []downloads.Job
	err  error
}
type submittedMsg struct {
	count int
	err   error
}
type actionMsg struct {
	message string
	err     error
}
type tickMsg time.Time

func New(services Services) *Model {
	status := services.StartupMessage
	if status == "" {
		status = "Connecting…"
	}
	m := &Model{services: services, selected: make(map[string]selectedFile), status: status, loading: true, progress: progress.New(progress.WithDefaultBlend(), progress.WithoutPercentage())}
	m.destinationInput = textinput.New()
	m.destinationInput.SetWidth(70)
	m.destinationInput.Prompt = ""
	m.initializeSettings()
	return m
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.loadRemote(""), m.loadVolumes(), m.loadJobs(), tick())
}

func (m *Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.progress.SetWidth(max(8, min(28, msg.Width/4)))
	case tea.KeyPressMsg:
		if m.tab == TabDestination && m.destinationPrompt != "" {
			return m.handleDestinationPrompt(msg)
		}
		if m.tab == TabSettings {
			return m.handleSettingsKey(msg)
		}
		return m.handleKey(msg)
	case remoteLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.status = "Remote error: " + msg.err.Error()
		} else {
			m.remotePath, m.remoteItems, m.remoteCursor = msg.listing.CurrentPath, msg.listing.Items, 0
			m.status = fmt.Sprintf("Loaded %d remote items", len(m.remoteItems))
		}
	case recursiveSelectedMsg:
		m.loading = false
		for _, file := range msg.files {
			m.selected[file.URL] = file
		}
		if msg.err != nil {
			m.status = fmt.Sprintf("Recursive scan selected %d files with errors: %v", len(msg.files), msg.err)
		} else {
			m.status = fmt.Sprintf("Recursively selected %d files", len(msg.files))
		}
	case volumesLoadedMsg:
		if msg.err != nil {
			m.status = "Volume error: " + msg.err.Error()
		} else {
			m.volumes = msg.volumes
			if m.services.Config != nil {
				for _, favorite := range m.services.Config.FavoritePaths {
					found := false
					for _, volume := range m.volumes {
						if volume.Path == favorite {
							found = true
							break
						}
					}
					if !found {
						m.volumes = append([]storage.Volume{{Path: favorite, Label: "Favorite", Kind: storage.VolumeNetwork, Available: true}}, m.volumes...)
					}
				}
			}
		}
	case directoryLoadedMsg:
		if msg.err != nil {
			m.status = "Destination error: " + msg.err.Error()
		} else {
			directories := make([]storage.Entry, 0, len(msg.entries))
			for _, entry := range msg.entries {
				if entry.IsDir {
					directories = append(directories, entry)
				}
			}
			m.destinationPath, m.destinationEntries, m.destinationCursor = msg.path, directories, 0
			m.status = "Browsing " + msg.path
		}
	case jobsLoadedMsg:
		if msg.err != nil {
			m.status = "aria2 unavailable: " + msg.err.Error()
		} else {
			m.jobs = msg.jobs
		}
	case submittedMsg:
		m.loading = false
		if msg.err != nil {
			m.status = fmt.Sprintf("Submitted %d with errors: %v", msg.count, msg.err)
		} else {
			m.status = fmt.Sprintf("Queued %d downloads", msg.count)
			m.selected = make(map[string]selectedFile)
		}
		return m, m.loadJobs()
	case actionMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
		} else {
			m.status = msg.message
		}
		return m, m.loadJobs()
	case tickMsg:
		return m, tea.Batch(m.loadJobs(), tick())
	}
	return m, nil
}

func (m *Model) handleKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "1":
		m.tab = TabRemote
	case "2":
		m.tab = TabSelection
	case "3":
		m.tab = TabDestination
	case "4":
		m.tab = TabDownloads
		return m, m.loadJobs()
	case "5":
		m.tab = TabSettings
		m.focusSetting(0)
	case "tab":
		m.tab = (m.tab + 1) % 5
	case "shift+tab":
		m.tab = (m.tab + 4) % 5
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "enter", " ":
		return m, m.activate()
	case "backspace", "h":
		return m, m.back()
	case "a":
		m.selectAll()
	case "s":
		if m.tab == TabRemote {
			m.loading = true
			m.status = "Scanning remote directories…"
			return m, m.selectRecursive(m.remotePath)
		}
	case "c":
		m.selected = make(map[string]selectedFile)
		m.status = "Selection cleared"
	case "d":
		return m, m.submitSelected()
	case "g":
		if m.tab == TabDestination {
			m.openDestinationPrompt("go to path", m.destinationPath)
			return m, nil
		}
	case "n":
		if m.tab == TabDestination {
			m.openDestinationPrompt("create directory", m.destinationPath)
			return m, nil
		}
	case "f":
		if m.tab == TabDestination {
			return m, m.favoriteDestination()
		}
	case "r":
		if m.tab == TabRemote {
			return m, m.loadRemote(m.remotePath)
		}
		if m.tab == TabDownloads {
			return m, m.loadJobs()
		}
	case "p":
		return m, m.togglePause()
	case "x":
		return m, m.removeJob()
	}
	return m, nil
}

func (m *Model) move(delta int) {
	length, cursor := 0, &m.remoteCursor
	switch m.tab {
	case TabRemote:
		length = len(m.remoteItems)
	case TabSelection:
		return
	case TabDestination:
		if m.destinationPath == "" {
			length = len(m.volumes)
		} else {
			length = len(m.destinationEntries)
		}
		cursor = &m.destinationCursor
	case TabDownloads:
		length = len(m.jobs)
		cursor = &m.downloadCursor
	case TabSettings:
		return
	}
	if length == 0 {
		*cursor = 0
		return
	}
	*cursor = (*cursor + delta + length) % length
}

func (m *Model) activate() tea.Cmd {
	switch m.tab {
	case TabRemote:
		if len(m.remoteItems) == 0 {
			return nil
		}
		item := m.remoteItems[m.remoteCursor]
		if item.Type == browser.TypeDirectory {
			m.loading = true
			return m.loadRemote(item.Path)
		}
		url := m.services.Remote.FullURL(item.Path)
		if _, exists := m.selected[url]; exists {
			delete(m.selected, url)
			m.status = "Deselected " + item.Name
		} else {
			m.selected[url] = selectedFile{URL: url, Name: item.Name}
			m.status = "Selected " + item.Name
		}
	case TabDestination:
		if m.destinationPath == "" {
			if len(m.volumes) == 0 {
				return nil
			}
			return m.loadDirectory(m.volumes[m.destinationCursor].Path)
		}
		if len(m.destinationEntries) == 0 {
			return nil
		}
		entry := m.destinationEntries[m.destinationCursor]
		if entry.IsDir {
			return m.loadDirectory(entry.Path)
		}
	case TabDownloads:
		return m.togglePause()
	}
	return nil
}

func (m *Model) back() tea.Cmd {
	if m.tab == TabRemote && m.remotePath != "" {
		return m.loadRemote(browser.GetParentPath(m.remotePath))
	}
	if m.tab == TabDestination && m.destinationPath != "" {
		parent := filepath.Dir(m.destinationPath)
		if parent == m.destinationPath || parent == "." {
			m.destinationPath, m.destinationEntries, m.destinationCursor = "", nil, 0
			return nil
		}
		return m.loadDirectory(parent)
	}
	return nil
}

func (m *Model) openDestinationPrompt(mode, value string) {
	m.destinationPrompt = mode
	m.destinationInput.SetValue(value)
	m.destinationInput.CursorEnd()
	m.destinationInput.Focus()
}

func (m *Model) handleDestinationPrompt(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.destinationPrompt = ""
		m.destinationInput.Blur()
		return m, nil
	case "enter":
		path, mode := strings.TrimSpace(m.destinationInput.Value()), m.destinationPrompt
		m.destinationPrompt = ""
		m.destinationInput.Blur()
		if path == "" {
			m.status = "Destination path cannot be empty"
			return m, nil
		}
		if mode == "create directory" {
			return m, func() tea.Msg {
				if err := m.services.Directories.CreateDirectory(path); err != nil {
					return directoryLoadedMsg{path: path, err: err}
				}
				entries, err := m.services.Directories.List(context.Background(), path)
				return directoryLoadedMsg{path: path, entries: entries, err: err}
			}
		}
		return m, m.loadDirectory(path)
	}
	var command tea.Cmd
	m.destinationInput, command = m.destinationInput.Update(key)
	return m, command
}

func (m *Model) favoriteDestination() tea.Cmd {
	if m.destinationPath == "" {
		m.status = "Browse to a destination before adding a favorite"
		return nil
	}
	return func() tea.Msg {
		cfg := m.services.Config
		if cfg == nil || m.services.SaveConfig == nil {
			return actionMsg{err: fmt.Errorf("configuration saving is unavailable")}
		}
		for _, path := range cfg.FavoritePaths {
			if path == m.destinationPath {
				return actionMsg{message: "Destination is already a favorite"}
			}
		}
		cfg.FavoritePaths = append(cfg.FavoritePaths, m.destinationPath)
		if err := m.services.SaveConfig(cfg); err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{message: "Added destination favorite " + m.destinationPath}
	}
}

func (m *Model) selectAll() {
	if m.tab != TabRemote {
		return
	}
	for _, item := range m.remoteItems {
		if item.Type == browser.TypeFile {
			url := m.services.Remote.FullURL(item.Path)
			m.selected[url] = selectedFile{URL: url, Name: item.Name}
		}
	}
	m.status = fmt.Sprintf("Selected %d files", len(m.selected))
}

func (m *Model) selectRecursive(root string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		queue := []string{root}
		visited := make(map[string]bool)
		files := make([]selectedFile, 0)
		for len(queue) > 0 {
			path := queue[0]
			queue = queue[1:]
			if visited[path] {
				continue
			}
			visited[path] = true
			if len(visited) > 10000 {
				return recursiveSelectedMsg{files: files, err: fmt.Errorf("scan stopped after 10,000 directories")}
			}
			listing, err := m.services.Remote.List(ctx, path)
			if err != nil {
				return recursiveSelectedMsg{files: files, err: fmt.Errorf("scan %q: %w", path, err)}
			}
			for _, item := range listing.Items {
				if item.Type == browser.TypeDirectory {
					queue = append(queue, item.Path)
					continue
				}
				files = append(files, selectedFile{URL: m.services.Remote.FullURL(item.Path), Name: item.Name})
			}
		}
		return recursiveSelectedMsg{files: files}
	}
}

func (m *Model) submitSelected() tea.Cmd {
	if len(m.selected) == 0 {
		m.status = "Select files first"
		return nil
	}
	if m.destinationPath == "" {
		m.status = "Choose a destination first"
		m.tab = TabDestination
		return nil
	}
	requests := make([]downloads.Request, 0, len(m.selected))
	for _, file := range m.selected {
		requests = append(requests, downloads.Request{URL: file.URL, Filename: file.Name, Destination: m.destinationPath, Username: m.services.Username, Password: m.services.Password})
	}
	sort.Slice(requests, func(i, j int) bool { return requests[i].URL < requests[j].URL })
	m.loading = true
	return func() tea.Msg {
		if err := m.services.Directories.IsWritable(m.destinationPath); err != nil {
			return submittedMsg{err: err}
		}
		results, err := m.services.Downloads.Submit(context.Background(), requests)
		return submittedMsg{count: len(results), err: err}
	}
}

func (m *Model) togglePause() tea.Cmd {
	if m.tab != TabDownloads || len(m.jobs) == 0 {
		return nil
	}
	job := m.jobs[m.downloadCursor]
	return func() tea.Msg {
		var err error
		message := "Paused " + job.Name
		if job.State == downloads.StatePaused {
			err = m.services.Downloads.Resume(context.Background(), job.GID)
			message = "Resumed " + job.Name
		} else {
			err = m.services.Downloads.Pause(context.Background(), job.GID)
		}
		return actionMsg{message: message, err: err}
	}
}
func (m *Model) removeJob() tea.Cmd {
	if m.tab != TabDownloads || len(m.jobs) == 0 {
		return nil
	}
	job := m.jobs[m.downloadCursor]
	return func() tea.Msg {
		return actionMsg{message: "Removed " + job.Name, err: m.services.Downloads.Remove(context.Background(), job.GID, true)}
	}
}

func (m *Model) loadRemote(path string) tea.Cmd {
	return func() tea.Msg {
		listing, err := m.services.Remote.List(context.Background(), path)
		return remoteLoadedMsg{listing, err}
	}
}
func (m *Model) loadVolumes() tea.Cmd {
	return func() tea.Msg {
		volumes, err := m.services.Volumes.ListVolumes(context.Background())
		return volumesLoadedMsg{volumes, err}
	}
}
func (m *Model) loadDirectory(path string) tea.Cmd {
	return func() tea.Msg {
		entries, err := m.services.Directories.List(context.Background(), path)
		return directoryLoadedMsg{path, entries, err}
	}
}
func (m *Model) loadJobs() tea.Cmd {
	return func() tea.Msg {
		jobs, err := m.services.Downloads.Snapshot(context.Background())
		return jobsLoadedMsg{jobs, err}
	}
}
func tick() tea.Cmd { return tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) }) }

var settingLabels = []string{"Remote base URL", "HTTP username", "HTTP password", "aria2 RPC endpoint", "aria2 RPC secret", "Max concurrent downloads"}

func (m *Model) initializeSettings() {
	values := []string{"", "", "", "http://127.0.0.1:6800/jsonrpc", "", "3"}
	if cfg := m.services.Config; cfg != nil {
		values = []string{cfg.BaseURL, cfg.Username, cfg.Password, cfg.Aria2Endpoint, cfg.Aria2Secret, strconv.Itoa(cfg.MaxConcurrentDownloads)}
	}
	m.settingsInputs = make([]textinput.Model, len(values))
	for index, value := range values {
		input := textinput.New()
		input.SetValue(value)
		input.SetWidth(60)
		input.Prompt = ""
		if index == 2 || index == 4 {
			input.EchoMode = textinput.EchoPassword
			input.EchoCharacter = '•'
		}
		m.settingsInputs[index] = input
	}
}

func (m *Model) focusSetting(index int) {
	if len(m.settingsInputs) == 0 {
		return
	}
	m.settingsCursor = (index + len(m.settingsInputs)) % len(m.settingsInputs)
	for i := range m.settingsInputs {
		if i == m.settingsCursor {
			m.settingsInputs[i].Focus()
		} else {
			m.settingsInputs[i].Blur()
		}
	}
}

func (m *Model) handleSettingsKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		for i := range m.settingsInputs {
			m.settingsInputs[i].Blur()
		}
		m.tab = TabRemote
		return m, nil
	case "tab", "down":
		m.focusSetting(m.settingsCursor + 1)
		return m, nil
	case "shift+tab", "up":
		m.focusSetting(m.settingsCursor - 1)
		return m, nil
	case "ctrl+s":
		return m, m.saveSettings()
	}
	var command tea.Cmd
	m.settingsInputs[m.settingsCursor], command = m.settingsInputs[m.settingsCursor].Update(key)
	return m, command
}

func (m *Model) saveSettings() tea.Cmd {
	return func() tea.Msg {
		cfg := m.services.Config
		if cfg == nil {
			cfg = config.NewConfig()
		}
		cfg.BaseURL = m.settingsInputs[0].Value()
		cfg.Username = m.settingsInputs[1].Value()
		cfg.Password = m.settingsInputs[2].Value()
		cfg.Aria2Endpoint = m.settingsInputs[3].Value()
		cfg.Aria2Secret = m.settingsInputs[4].Value()
		concurrent, err := strconv.Atoi(m.settingsInputs[5].Value())
		if err != nil || concurrent < 1 {
			return actionMsg{err: fmt.Errorf("max concurrent downloads must be a positive number")}
		}
		cfg.MaxConcurrentDownloads = concurrent
		if err := cfg.ValidateV2(); err != nil {
			return actionMsg{err: err}
		}
		if m.services.SaveConfig == nil {
			return actionMsg{err: fmt.Errorf("configuration saving is unavailable")}
		}
		if err := m.services.SaveConfig(cfg); err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{message: "Settings saved; restart PlexLinker to reconnect with changed endpoints"}
	}
}

var (
	accent        = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	muted         = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("24")).Bold(true)
	good          = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
)

func (m *Model) View() tea.View {
	var b strings.Builder
	b.WriteString(accent.Render("PlexLinker v2"))
	b.WriteString("  ")
	for index, label := range []string{"1 Remote", "2 Selection", "3 Destination", "4 Downloads", "5 Settings"} {
		style := muted
		if int(m.tab) == index {
			style = accent
		}
		b.WriteString(style.Render("[" + label + "]"))
		b.WriteString(" ")
	}
	b.WriteString("\n")
	b.WriteString(muted.Render(strings.Repeat("─", max(1, m.width))))
	b.WriteString("\n")
	switch m.tab {
	case TabRemote:
		m.viewRemote(&b)
	case TabSelection:
		m.viewSelection(&b)
	case TabDestination:
		m.viewDestination(&b)
	case TabDownloads:
		m.viewDownloads(&b)
	case TabSettings:
		m.viewSettings(&b)
	}
	b.WriteString("\n")
	b.WriteString(muted.Render(strings.Repeat("─", max(1, m.width))))
	b.WriteString("\n")
	b.WriteString(m.status)
	if m.loading {
		b.WriteString("  …")
	}
	b.WriteString("\n")
	if m.tab == TabSettings {
		b.WriteString(muted.Render("Tab/↑/↓ fields • Ctrl+S save • Esc return"))
	} else {
		b.WriteString(muted.Render("Tab switch • ↑/↓ navigate • Enter select/open • a current • s recursive • d queue • q quit"))
	}
	view := tea.NewView(b.String())
	view.AltScreen = true
	return view
}

func (m *Model) viewRemote(b *strings.Builder) {
	b.WriteString("Remote: " + displayPath(m.remotePath) + fmt.Sprintf("  (%d selected)\n", len(m.selected)))
	for i, item := range m.remoteItems {
		marker := "  "
		if _, ok := m.selected[m.services.Remote.FullURL(item.Path)]; ok {
			marker = good.Render("✓ ")
		}
		icon := "▸"
		if item.Type == browser.TypeFile {
			icon = "·"
		}
		m.row(b, i == m.remoteCursor, marker+icon+" "+item.Name)
	}
}
func (m *Model) viewSelection(b *strings.Builder) {
	b.WriteString(fmt.Sprintf("Selection: %d files\n", len(m.selected)))
	files := make([]selectedFile, 0, len(m.selected))
	for _, f := range m.selected {
		files = append(files, f)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	for _, file := range files {
		b.WriteString("  " + file.Name + "\n")
	}
}
func (m *Model) viewDestination(b *strings.Builder) {
	if m.destinationPath == "" {
		b.WriteString("Choose a drive or mount\n")
		for i, v := range m.volumes {
			label := v.Path
			if v.Label != "" {
				label += "  " + v.Label
			}
			label += fmt.Sprintf("  %s free", formatBytes(v.FreeBytes))
			m.row(b, i == m.destinationCursor, label)
		}
		if m.destinationPrompt != "" {
			b.WriteString("\n" + accent.Render(m.destinationPrompt) + "\n  " + m.destinationInput.View() + "\n")
		} else {
			b.WriteString("\n" + muted.Render("g go to path/UNC") + "\n")
		}
		return
	}
	b.WriteString("Destination: " + good.Render(m.destinationPath) + "\n")
	for i, e := range m.destinationEntries {
		if e.IsDir {
			m.row(b, i == m.destinationCursor, "▸ "+e.Name)
		}
	}
	if m.destinationPrompt != "" {
		b.WriteString("\n" + accent.Render(m.destinationPrompt) + "\n  " + m.destinationInput.View() + "\n")
	} else {
		b.WriteString("\n" + muted.Render("g go to path/UNC • n create directory • f favorite current destination") + "\n")
	}
}
func (m *Model) viewDownloads(b *strings.Builder) {
	b.WriteString(fmt.Sprintf("aria2 jobs: %d\n", len(m.jobs)))
	for i, j := range m.jobs {
		text := fmt.Sprintf("%-9s %s %5.1f%% %8s/s  %s", j.State, m.progress.ViewAs(j.Progress()), j.Progress()*100, formatBytes(j.BytesPerSecond), j.Name)
		m.row(b, i == m.downloadCursor, text)
	}
}
func (m *Model) viewSettings(b *strings.Builder) {
	b.WriteString("Settings\n")
	for index := range m.settingsInputs {
		label := muted.Render(settingLabels[index])
		if index == m.settingsCursor {
			label = accent.Render(settingLabels[index])
		}
		b.WriteString(label + "\n  " + m.settingsInputs[index].View() + "\n")
	}
}
func (m *Model) row(b *strings.Builder, active bool, text string) {
	limit := m.width - 2
	if limit > 3 && len(text) > limit {
		text = text[:limit-1] + "…"
	}
	if active {
		b.WriteString(selectedStyle.Render(text))
	} else {
		b.WriteString(text)
	}
	b.WriteString("\n")
}
func displayPath(path string) string {
	if path == "" {
		return "/"
	}
	return path
}
func formatBytes(value uint64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	amount := float64(value)
	unit := 0
	for amount >= 1024 && unit < len(units)-1 {
		amount /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d %s", value, units[unit])
	}
	return fmt.Sprintf("%.1f %s", amount, units[unit])
}

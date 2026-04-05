// tui/model.go
package tui

import (
	"bytes"
	"text/template"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lazychat/conversation"
	"lazychat/provider"
	"lazychat/store"
)

const sidebarWidth = 25

type focus int

const (
	focusSidebar focus = iota
	focusChat
	focusUsage
	focusModelPicker
	focusSkillPicker
	focusPager
)

type usageResultMsg struct {
	info provider.RateLimitInfo
	err  error
}

type resizeMsg int

type Model struct {
	sidebar          sidebar
	chat             chat
	usage            usageView
	modelPicker      modelPicker
	skillPicker      skillPicker
	pager            pager
	statusbar        statusBar
	store            *store.Store
	providers        []provider.Provider
	active           provider.Provider
	focus            focus
	width            int
	height           int
	activeConv       *conversation.Conversation
	showUsagePreview bool
	ready            bool
	resizeSeq        int
	skills           []store.Skill
}

func (m Model) isFixedMode(mode string) bool {
	for _, s := range m.skills {
		if s.Mode == mode {
			return true
		}
	}
	return false
}

func (m Model) fixedAfterIdx(convs []conversation.Conversation) int {
	for i, c := range convs {
		if !m.isFixedMode(c.Mode) {
			return i
		}
	}
	return len(convs)
}

func NewModel(s *store.Store, providers ...provider.Provider) Model {
	sb := newSidebar()
	ch := newChat()
	ch.inputHistory = s.LoadHistory()
	uv := newUsageView()
	mp := newModelPicker(providers)

	convs, _ := s.ListMeta()

	cfg, _ := s.LoadConfig()
	skills := cfg.Skills
	skp := newSkillPicker(skills)

	// Filter to only include non-skill conversations in sidebar
	var filtered []conversation.Conversation
	for _, c := range convs {
		isFixed := false
		for _, skill := range skills {
			if c.Mode == skill.Mode {
				isFixed = true
				break
			}
		}
		if !isFixed {
			filtered = append(filtered, c)
		}
	}
	sb.conversations = filtered
	sb.skills = skills

	active := providers[0]

	// Restore last selected model from config
	if cfg.Provider != "" {
		for _, p := range providers {
			if p.Name() == cfg.Provider {
				p.SetModel(cfg.Model)
				active = p
				mp.current = modelEntry{provider: cfg.Provider, model: cfg.Model}
				// Set picker cursor to the saved model
				for i, e := range mp.entries {
					if e.provider == cfg.Provider && e.model == cfg.Model {
						mp.selected = i
						break
					}
				}
				break
			}
		}
	}

	sb2 := statusBar{
		provider: active.Name(),
		model:    active.GetModel(),
	}

	ch.activeModel = active.GetModel()
	pg := newPager()

	m := Model{
		sidebar:     sb,
		chat:        ch,
		usage:       uv,
		modelPicker: mp,
		skillPicker: skp,
		pager:       pg,
		statusbar:   sb2,
		store:       s,
		providers:   providers,
		active:      active,
		focus:       focusSkillPicker,
		skills:      skills,
	}

	m.sidebar.focused = false
	m.chat.focused = false
	m.chat.inputFocused = false

	// Start with the first conversation selected
	if len(m.sidebar.conversations) > 0 {
		m.sidebar.selected = 0
		full, err := s.Load(m.sidebar.conversations[0].ID)
		if err == nil {
			m.sidebar.conversations[0].Messages = full.Messages
		} else {
			m.sidebar.conversations[0].Messages = []conversation.Message{}
		}
		m.activeConv = &m.sidebar.conversations[0]
		m.chat.messages = m.activeConv.Messages
		m.chat.selectedMsg = -1
		// Chat is already blurred by the initial focusSkillPicker settings above
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.updateSizes()
		m.resizeSeq++
		seq := m.resizeSeq
		return m, tea.Tick(50*time.Millisecond, func(time.Time) tea.Msg {
			return resizeMsg(seq)
		})

	case resizeMsg:
		if int(msg) == m.resizeSeq {
			m.chat.refreshViewport()
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.focus == focusModelPicker || m.focus == focusSkillPicker || m.focus == focusUsage {
				m.focusToSidebar()
				return m, nil
			}
		case "ctrl+k":
			if m.focus == focusSkillPicker {
				m.focusToSidebar()
				return m, nil
			}
			m.focus = focusSkillPicker
			m.sidebar.focused = false
			m.chat.focused = false
			m.chat.inputFocused = false
			m.chat.input.Blur()
			return m, nil
		case "tab":
			if m.focus == focusUsage || m.focus == focusModelPicker || m.focus == focusSkillPicker || m.focus == focusPager {
				m.focusToSidebar()
			} else if m.activeConv != nil {
				if m.focus == focusSidebar {
					m.focusToChat()
				} else {
					m.focusToSidebar()
				}
			}
			return m, nil
		case "h":
			if m.focus == focusChat && !m.chat.inputFocused {
				m.focusToSidebar()
				return m, nil
			}
			if m.focus == focusUsage || m.focus == focusModelPicker || m.focus == focusSkillPicker {
				m.focusToSidebar()
				return m, nil
			}
		case "m":
			if m.focus == focusModelPicker {
				m.focusToSidebar()
				return m, nil
			}
			if m.focus == focusSidebar || (m.focus == focusChat && !m.chat.inputFocused) {
				m.focus = focusModelPicker
				m.sidebar.focused = false
				m.chat.focused = false
				m.chat.inputFocused = false
				m.chat.input.Blur()
				return m, nil
			}
		case "r":
			if m.focus == focusUsage && !m.usage.loading {
				m.usage.loading = true
				m.usage.err = ""
				m.statusbar.activity = netFetchingUsage
				m.statusbar.spinnerFrame = 0
				g := m.active
				return m, tea.Batch(
					func() tea.Msg {
						info, err := g.FetchUsage()
						return usageResultMsg{info: info, err: err}
					},
					spinnerTick(),
				)
			}
		case "q":
			if m.focus != focusPager && !(m.focus == focusChat && m.chat.inputFocused) {
				return m, tea.Quit
			}
		}

	case openPagerMsg:
		rightWidth := m.width - sidebarWidth - 4
		panelHeight := m.height - 3
		m.pager.open(msg.content, rightWidth-2, panelHeight)
		m.focus = focusPager
		m.chat.focused = false
		m.chat.inputFocused = false
		m.chat.input.Blur()
		m.sidebar.focused = false
		return m, nil

	case pagerCloseMsg:
		m.focusToChat()
		return m, nil

	case clipboardMsg:
		if msg.err != nil {
			m.statusbar.flashMsg = "Copy failed: " + msg.err.Error()
		} else {
			m.statusbar.flashMsg = "Copied to clipboard!"
		}
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearStatusMsg{}
		})

	case clearStatusMsg:
		m.statusbar.flashMsg = ""
		return m, nil

	case spinnerTickMsg:
		if m.statusbar.activity != netIdle {
			m.statusbar.spinnerFrame++
			return m, spinnerTick()
		}
		return m, nil

	case escEmptyChatMsg:
		m.focusToSidebar()
		return m, nil

	case previewUsageMsg:
		m.syncActiveConv()
		m.activeConv = nil
		m.usage.info = m.active.GetRateLimit()
		m.usage.providerName = m.active.Name()
		m.usage.model = m.active.GetModel()
		m.showUsagePreview = true
		return m, nil

	case previewConvMsg:
		idx := int(msg)
		if idx >= 0 && idx < len(m.sidebar.conversations) {
			m.syncActiveConv()
			m.loadConvMessages(idx)
			m.activeConv = &m.sidebar.conversations[idx]
			m.chat.messages = m.activeConv.Messages
			m.chat.err = ""
			m.chat.selectedMsg = -1
			m.chat.refreshViewport()
			m.chat.viewport.GotoBottom()
			m.showUsagePreview = false
		}
		return m, nil

	case selectConvInputMsg:
		idx := int(msg)
		if idx >= 0 && idx < len(m.sidebar.conversations) {
			m.syncActiveConv()
			m.loadConvMessages(idx)
			m.activeConv = &m.sidebar.conversations[idx]
			m.chat.messages = m.activeConv.Messages
			m.chat.err = ""
			m.chat.selectedMsg = -1
			m.chat.refreshViewport()
			m.chat.viewport.GotoBottom()
			m.showUsagePreview = false
			m.focus = focusChat
			m.sidebar.focused = false
			m.chat.focused = true
			m.chat.inputFocused = true
			m.chat.input.Focus()
			m.chat.resizeInput()
		}
		return m, nil

	case selectConvMsg:
		idx := int(msg)
		if idx >= 0 && idx < len(m.sidebar.conversations) {
			m.syncActiveConv()
			m.loadConvMessages(idx)
			m.activeConv = &m.sidebar.conversations[idx]
			m.chat.messages = m.activeConv.Messages
			m.chat.err = ""
			m.showUsagePreview = false
			if len(m.activeConv.Messages) == 0 {
				m.chat.selectedMsg = -1
				m.focus = focusChat
				m.sidebar.focused = false
				m.chat.focused = true
				m.chat.inputFocused = true
				m.chat.input.Focus()
				m.chat.resizeInput()
			} else {
				m.chat.selectedMsg = len(m.activeConv.Messages) - 1
				m.focusToChat()
			}
			m.chat.refreshViewport()
			m.chat.viewport.GotoBottom()
		}
		return m, nil

	case newConvMsg:
		// Reuse existing "[new chat]" if one exists
		for i, c := range m.sidebar.conversations {
			if c.Title == "[new chat]" && !m.isFixedMode(c.Mode) {
				m.syncActiveConv()
				m.loadConvMessages(i)
				m.sidebar.selected = i
				m.activeConv = &m.sidebar.conversations[i]
				m.chat.messages = m.activeConv.Messages
				m.chat.selectedMsg = -1
				m.chat.err = ""
				m.chat.refreshViewport()
				m.showUsagePreview = false
				m.focus = focusChat
				m.sidebar.focused = false
				m.chat.focused = true
				m.chat.inputFocused = true
				m.chat.input.Focus()
				m.chat.resizeInput()
				return m, nil
			}
		}
		conv := conversation.New("[new chat]")
		// Insert after fixed skill conversations
		insertIdx := m.fixedAfterIdx(m.sidebar.conversations)
		m.sidebar.conversations = append(m.sidebar.conversations[:insertIdx],
			append([]conversation.Conversation{conv}, m.sidebar.conversations[insertIdx:]...)...)
		m.sidebar.selected = insertIdx
		m.activeConv = &m.sidebar.conversations[insertIdx]
		m.chat.messages = nil
		m.chat.selectedMsg = -1
		m.chat.err = ""
		m.chat.refreshViewport()
		m.showUsagePreview = false
		m.focus = focusChat
		m.sidebar.focused = false
		m.chat.focused = true
		m.chat.inputFocused = true
		m.chat.input.Focus()
		m.chat.resizeInput()
		return m, nil

	case modelChangedMsg:
		p, model := msg.provider, msg.model
		for _, prov := range m.providers {
			if prov.Name() == p {
				prov.SetModel(model)
				m.active = prov
				break
			}
		}
		m.statusbar.provider = p
		m.statusbar.model = model
		m.chat.activeModel = model
		m.store.SaveConfig(store.Config{Provider: p, Model: model, Skills: m.skills})
		m.focusToSidebar()
		return m, nil

	case skillSelectedMsg:
		if msg.isNew {
			return m.Update(newConvMsg{})
		}

		// Find the conversation with this skill mode and select it if in sidebar
		for i, c := range m.sidebar.conversations {
			if c.Mode == msg.skill.Mode {
				m.sidebar.selected = i
				return m.Update(selectConvInputMsg(i))
			}
		}

		// Not in sidebar, load from store or create
		all, _ := m.store.ListMeta()
		var skillConv *conversation.Conversation
		for i := range all {
			if all[i].Mode == msg.skill.Mode {
				all[i].Title = msg.skill.Title // Sync title with listing
				skillConv = &all[i]
				break
			}
		}

		if skillConv == nil {
			tc := conversation.New(msg.skill.Title)
			tc.Mode = msg.skill.Mode
			skillConv = &tc
		}

		// Load messages
		full, _ := m.store.Load(skillConv.ID)
		skillConv.Messages = full.Messages

		// Add to sidebar at the top (or after other skills if we want)
		m.sidebar.conversations = append([]conversation.Conversation{*skillConv}, m.sidebar.conversations...)
		m.sidebar.selected = 0
		return m.Update(selectConvInputMsg(0))

	case selectUsageMsg:
		m.syncActiveConv()
		m.activeConv = nil
		m.usage.info = m.active.GetRateLimit()
		m.usage.providerName = m.active.Name()
		m.usage.model = m.active.GetModel()
		m.usage.loading = true
		m.usage.err = ""
		m.focus = focusUsage
		m.sidebar.focused = false
		m.chat.focused = false
		m.chat.inputFocused = false
		m.chat.input.Blur()
		m.statusbar.activity = netFetchingUsage
		m.statusbar.spinnerFrame = 0
		g := m.active
		return m, tea.Batch(
			func() tea.Msg {
				info, err := g.FetchUsage()
				return usageResultMsg{info: info, err: err}
			},
			spinnerTick(),
		)

	case usageResultMsg:
		m.usage.loading = false
		m.usage.info = msg.info
		m.statusbar.activity = netIdle
		if msg.err != nil {
			m.usage.err = msg.err.Error()
		}
		return m, nil

	case deleteConvMsg:
		idx := int(msg)
		if idx >= 0 && idx < len(m.sidebar.conversations) {
			conv := m.sidebar.conversations[idx]
			m.store.Delete(conv.ID)
			m.sidebar.conversations = append(
				m.sidebar.conversations[:idx],
				m.sidebar.conversations[idx+1:]...,
			)
			if m.activeConv != nil && m.activeConv.ID == conv.ID {
				m.activeConv = nil
				m.chat.messages = nil
				m.chat.refreshViewport()
			}
			if m.sidebar.selected >= len(m.sidebar.conversations) && m.sidebar.selected > 0 {
				m.sidebar.selected--
			}
		}
		return m, nil

	case deleteMsgMsg:
		idx := int(msg)
		if m.activeConv != nil && idx >= 0 && idx < len(m.activeConv.Messages) {
			m.activeConv.Messages = append(m.activeConv.Messages[:idx], m.activeConv.Messages[idx+1:]...)
			m.chat.messages = m.activeConv.Messages
			if m.chat.selectedMsg >= len(m.chat.messages) {
				m.chat.selectedMsg = len(m.chat.messages) - 1
			}
			m.chat.refreshViewport()
			m.store.Save(*m.activeConv)
		}
		return m, nil

	case sendMsg:
		if m.activeConv == nil {
			return m, nil
		}
		m.store.SaveHistory(m.chat.inputHistory)

		// If currently streaming, finalize the in-progress assistant message
		if m.chat.streaming {
			m.chat.streaming = false
			m.chat.streamCh = nil
			if m.activeConv != nil {
				m.activeConv.Messages = m.chat.messages
				m.store.Save(*m.activeConv)
			}
		}

		skillMode := m.activeConv.Mode
		isSkill := m.isFixedMode(skillMode)

		// If a skill chat already has messages, archive it and create a fresh one
		if isSkill && len(m.activeConv.Messages) > 0 {
			// Auto-title the old conversation from its first user message
			isSkillTitle := false
			for _, s := range m.skills {
				if m.activeConv.Title == s.Title {
					isSkillTitle = true
					break
				}
			}
			if isSkillTitle {
				for _, mm := range m.activeConv.Messages {
					if mm.Role == "user" {
						title := mm.Content
						if len(title) > 30 {
							title = title[:30] + "..."
						}
						m.activeConv.Title = title
						break
					}
				}
			}
			archivedID := m.activeConv.ID
			m.activeConv.Mode = ""
			for i := range m.sidebar.conversations {
				if m.sidebar.conversations[i].ID == archivedID {
					m.sidebar.conversations[i].Title = m.activeConv.Title
					m.sidebar.conversations[i].Mode = ""
					break
				}
			}
			m.store.Save(*m.activeConv)

			// Create a fresh skill conversation
			var newSkill store.Skill
			for _, s := range m.skills {
				if s.Mode == skillMode {
					newSkill = s
					break
				}
			}
			tc := conversation.New(newSkill.Title)
			tc.Mode = skillMode

			// Remove archived from its fixed position, insert new skill, then
			// place archived after all fixed items to keep ordering clean.
			newConvs := make([]conversation.Conversation, 0, len(m.sidebar.conversations))
			var archived conversation.Conversation
			for _, c := range m.sidebar.conversations {
				if c.ID == archivedID {
					archived = c
				} else {
					newConvs = append(newConvs, c)
				}
			}

			// Find correct insertion index based on config order
			insertIdx := 0
			for i, s := range m.skills {
				if s.Mode == skillMode {
					insertIdx = i
					break
				}
			}

			newConvs = append(newConvs[:insertIdx],
				append([]conversation.Conversation{tc}, newConvs[insertIdx:]...)...)
			afterFixed := m.fixedAfterIdx(newConvs)
			newConvs = append(newConvs[:afterFixed],
				append([]conversation.Conversation{archived}, newConvs[afterFixed:]...)...)
			m.sidebar.conversations = newConvs
			m.sidebar.selected = insertIdx
			m.activeConv = &m.sidebar.conversations[insertIdx]
			m.chat.messages = nil
			m.chat.selectedMsg = -1
			m.chat.err = ""
			m.showUsagePreview = false
		}

		// If non-skill chat already has messages, spawn a new conversation
		if !isSkill && len(m.activeConv.Messages) > 0 {
			m.syncActiveConv()
			conv := conversation.New("[new chat]")
			insertIdx := m.fixedAfterIdx(m.sidebar.conversations)
			m.sidebar.conversations = append(m.sidebar.conversations[:insertIdx],
				append([]conversation.Conversation{conv}, m.sidebar.conversations[insertIdx:]...)...)
			m.sidebar.selected = insertIdx
			m.activeConv = &m.sidebar.conversations[insertIdx]
			m.chat.messages = nil
			m.chat.selectedMsg = -1
			m.chat.err = ""
			m.showUsagePreview = false
		}

		userMsg := conversation.Message{Role: "user", Content: string(msg)}
		m.activeConv.Messages = append(m.activeConv.Messages, userMsg)

		// Auto-title from first message
		if len(m.activeConv.Messages) == 1 && m.activeConv.Title == "[new chat]" {
			title := string(msg)
			if len(title) > 30 {
				title = title[:30] + "..."
			}
			m.activeConv.Title = title
			for i := range m.sidebar.conversations {
				if m.sidebar.conversations[i].ID == m.activeConv.ID {
					m.sidebar.conversations[i].Title = title
					break
				}
			}
		}

		assistantMsg := conversation.Message{Role: "assistant", Content: "", Model: m.active.GetModel()}
		m.activeConv.Messages = append(m.activeConv.Messages, assistantMsg)

		m.store.Save(*m.activeConv)

		m.chat.messages = m.activeConv.Messages
		m.chat.streaming = true // set before refresh so "thinking..." shows immediately
		m.chat.refreshViewport()
		m.chat.viewport.GotoBottom()

		// Build messages to send to the LLM (exclude reasoning messages)
		var chatMsgs []conversation.Message
		if isSkill {
			var skill store.Skill
			for _, s := range m.skills {
				if s.Mode == skillMode {
					skill = s
					break
				}
			}

			tmpl, err := template.New("prompt").Parse(skill.Prompt)
			if err == nil {
				var buf bytes.Buffer
				err = tmpl.Execute(&buf, struct{ Input string }{Input: string(msg)})
				if err == nil {
					chatMsgs = []conversation.Message{{Role: "user", Content: buf.String()}}
				}
			}

			if chatMsgs == nil {
				// Fallback if template fails
				chatMsgs = []conversation.Message{{Role: "user", Content: string(msg)}}
			}
		} else {
			for _, m := range m.activeConv.Messages[:len(m.activeConv.Messages)-1] {
				if !m.Reasoning {
					chatMsgs = append(chatMsgs, m)
				}
			}
		}

		ch := m.active.StreamChat(chatMsgs)
		m.chat.streamCh = ch
		m.chat.err = ""
		m.statusbar.activity = netSending
		m.statusbar.tokenCount = 0
		m.statusbar.spinnerFrame = 0
		m.statusbar.lastError = ""

		return m, tea.Batch(waitForStream(ch), spinnerTick())

	case reasoningTokenMsg:
		m.statusbar.activity = netStreaming
		m.statusbar.tokenCount++
		var cmd tea.Cmd
		m.chat, cmd = m.chat.Update(msg)
		if m.activeConv != nil {
			m.activeConv.Messages = m.chat.messages
		}
		return m, cmd

	case tokenMsg:
		m.statusbar.activity = netStreaming
		m.statusbar.tokenCount++
		var cmd tea.Cmd
		m.chat, cmd = m.chat.Update(msg)
		if m.activeConv != nil {
			m.activeConv.Messages = m.chat.messages
		}
		return m, cmd

	case streamDoneMsg:
		m.chat, _ = m.chat.Update(msg)
		m.statusbar.activity = netIdle
		if m.activeConv != nil {
			m.activeConv.Messages = m.chat.messages
			m.store.Save(*m.activeConv)
		}
		// Focus on the assistant's response
		if len(m.chat.messages) > 0 {
			m.chat.selectedMsg = len(m.chat.messages) - 1
			m.chat.inputFocused = false
			m.chat.input.Blur()
			m.chat.refreshViewport()
			m.chat.viewport.GotoBottom()
		}
		return m, nil

	case streamErrMsg:
		m.chat, _ = m.chat.Update(msg)
		m.statusbar.activity = netError
		m.statusbar.lastError = msg.err.Error()
		if m.activeConv != nil {
			m.activeConv.Messages = m.chat.messages
			m.store.Save(*m.activeConv)
		}
		return m, nil
	}

	var cmd tea.Cmd
	if m.focus == focusPager {
		m.pager, cmd = m.pager.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}
	if m.focus == focusModelPicker {
		m.modelPicker, cmd = m.modelPicker.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}
	if m.focus == focusSkillPicker {
		m.skillPicker, cmd = m.skillPicker.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	m.sidebar, cmd = m.sidebar.Update(msg)
	cmds = append(cmds, cmd)

	m.chat, cmd = m.chat.Update(msg)
	cmds = append(cmds, cmd)

	if m.activeConv != nil {
		m.activeConv.Messages = m.chat.messages
	}

	return m, tea.Batch(cmds...)
}

// syncActiveConv saves the current active conversation's messages back to the
// sidebar slice and persists to disk. Call before switching to a different conversation.
func (m *Model) syncActiveConv() {
	if m.activeConv == nil {
		return
	}
	if len(m.chat.messages) == 0 {
		return // nothing to save
	}
	m.activeConv.Messages = m.chat.messages
	m.store.Save(*m.activeConv)
}

// loadConvMessages lazily loads a conversation's messages from disk if not already loaded.
func (m *Model) loadConvMessages(idx int) {
	conv := &m.sidebar.conversations[idx]
	if conv.Messages != nil {
		return // already loaded
	}
	full, err := m.store.Load(conv.ID)
	if err != nil {
		conv.Messages = []conversation.Message{}
		return
	}
	conv.Messages = full.Messages
}

func (m *Model) focusToChat() {
	m.focus = focusChat
	m.sidebar.focused = false
	m.chat.focused = true
	m.chat.inputFocused = false
	m.chat.input.Blur()
}

func (m *Model) focusToSidebar() {
	m.focus = focusSidebar
	m.sidebar.focused = true
	m.chat.focused = false
	m.chat.inputFocused = false
	m.chat.input.Blur()
}

func (m *Model) updateSizes() {
	rightWidth := m.width - sidebarWidth - 4
	rightHeight := m.height - 3 // reserve 1 row for status bar
	m.sidebar.width = sidebarWidth
	m.sidebar.height = m.height - 1
	m.statusbar.width = m.width
	m.chat.setSize(rightWidth, rightHeight)
	m.usage.setSize(rightWidth, rightHeight)
	m.modelPicker.setSize(rightWidth, rightHeight)
	m.skillPicker.setSize(rightWidth, rightHeight)
	m.pager.setSize(rightWidth-2, rightHeight)
}

func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	sbStyle := blurredBorder.Padding(1, 1)
	chStyle := blurredBorder.Padding(0, 1)

	if m.focus == focusSidebar {
		sbStyle = focusedBorder.Padding(1, 1)
	} else {
		chStyle = focusedBorder.Padding(0, 1)
	}

	panelHeight := m.height - 3

	sidebarView := sbStyle.
		Width(sidebarWidth).
		Height(panelHeight).
		Render(m.sidebar.View())

	rightWidth := m.width - sidebarWidth - 4
	var rightView string
	switch {
	case m.focus == focusUsage, m.focus == focusSidebar && m.showUsagePreview:
		rightView = chStyle.Width(rightWidth).Height(panelHeight).Render(m.usage.View())
	case m.focus == focusModelPicker:
		rightView = chStyle.Width(rightWidth).Height(panelHeight).Render(m.modelPicker.View())
	case m.focus == focusSkillPicker:
		rightView = chStyle.Width(rightWidth).Height(panelHeight).Render(m.skillPicker.View())
	case m.focus == focusPager:
		rightView = chStyle.Width(rightWidth).Height(panelHeight).Render(m.pager.View())
	default:
		rightView = chStyle.Width(rightWidth).Height(panelHeight).Render(m.chat.View())
	}

	top := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, rightView)
	return lipgloss.JoinVertical(lipgloss.Left, top, m.statusbar.View())
}

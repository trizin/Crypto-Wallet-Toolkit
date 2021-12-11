package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ListItem struct {
	title string
	desc  string
	id    string
}

func (i ListItem) Title() string       { return i.title }
func (i ListItem) Description() string { return i.desc }
func (i ListItem) FilterValue() string { return i.title }

var (
	docStyle            = lipgloss.NewStyle().Margin(1, 2)
	focusedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	cursorStyle         = focusedStyle.Copy()
	noStyle             = lipgloss.NewStyle()
	helpStyle           = blurredStyle.Copy()
	cursorModeHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	titleStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("#9763e6")).Bold(true)

	focusedButton = focusedStyle.Copy().Render("[ Submit ]")
	blurredButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))
)

type UI struct {
	list  list.Model
	input textinput.Model

	choice     ListItem
	state      string
	inputText  string
	walletData WalletData
	output     string
	title      string

	multiInput []textinput.Model
	focusIndex int
}

func (m UI) Init() tea.Cmd {
	return nil
}

func (m UI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			return m, tea.Quit

		case "tab", "shift+tab", "up", "down":
			if m.state == "sign_transaction" || m.state == "keystore_access" {
				s := msg.String()

				// Cycle indexes
				if s == "up" || s == "shift+tab" {
					m.focusIndex--
				} else {
					m.focusIndex++
				}

				if m.focusIndex > len(m.multiInput) {
					m.focusIndex = 0
				} else if m.focusIndex < 0 {
					m.focusIndex = len(m.multiInput)
				}

				cmds := make([]tea.Cmd, len(m.multiInput))
				for i := 0; i <= len(m.multiInput)-1; i++ {
					if i == m.focusIndex {
						// Set focused state
						cmds[i] = m.multiInput[i].Focus()
						m.multiInput[i].PromptStyle = focusedStyle
						m.multiInput[i].TextStyle = focusedStyle
						continue
					}
					// Remove focused state
					m.multiInput[i].Blur()
					m.multiInput[i].PromptStyle = noStyle
					m.multiInput[i].TextStyle = noStyle
				}

				return m, tea.Batch(cmds...)
			}

		case "enter":

			if m.state == "new_wallet" || m.state == "get_info_wallet" || m.state == "output" {
				m.setState("main")
			} else if m.state == "sign_transaction" {
				if m.focusIndex == len(m.multiInput) {
					nonce, _ := strconv.Atoi(m.multiInput[0].Value())
					toAddress := m.multiInput[1].Value()
					value, _ := strconv.ParseFloat(m.multiInput[2].Value(), 64)
					gasLimit, _ := strconv.Atoi(m.multiInput[3].Value())
					gasPrice, _ := strconv.ParseFloat(m.multiInput[4].Value(), 64)
					data := m.multiInput[5].Value()

					signedTransaction := m.walletData.signTransaction(nonce, toAddress, value, gasLimit, gasPrice, data)

					m.title = "Signed Transaction Hash"
					m.output = signedTransaction
					m.setState("output")

					m.setMultiInputView()
				}
			} else if m.state == "keystore_access" {
				path := m.multiInput[0].Value()
				password := m.multiInput[1].Value()

				walletData := loadKeystore(path, password)
				m.walletData = walletData
				m.setState("main")
				m.list.SetItems(getControlWalletItems())
				m.list.Title = m.walletData.PublicKey

			} else if m.state == "pk" {
				privateKey := m.input.Value()
				m.walletData = getWalletFromPK(privateKey)
				m.setState("main")
				m.list.SetItems(getControlWalletItems())
				m.input = getText()
				m.list.Title = m.walletData.PublicKey
			} else if m.state == "sign_message" {
				message := m.input.Value()
				signedMessage := m.walletData.signMessage(message)
				m.title = "Signed Message"
				m.output = signedMessage
				m.setState("output")
				m.input = getText()
			} else if m.state == "save_keystore" {
				password := m.input.Value()
				keystoreFile := m.walletData.createKeystore(password)
				m.title = "Keystore file saved"
				m.output = "Path: " + keystoreFile
				m.setState("output")
				m.input = getText()
			} else if m.state == "main" || m.state == "access_wallet" {
				item, ok := m.list.SelectedItem().(ListItem)

				m.setState(item.id)
				switch item.id {
				case "sign_transaction":
					m.setMultiInputView()
				case "keystore_access":
					m.setMultiInputViewKeystoreFile()
				case "access_wallet":
					m.list.SetItems(getAccessWalletItems())
					m.list.Title = "Access Wallet"
				case "new_wallet":
					walletData := generateWallet()
					m.walletData = walletData
					m.setState("main")
					m.list.SetItems(getControlWalletItems())
					m.input = getText()
					m.list.Title = m.walletData.PublicKey
				case "public_key":
					m.output = dispalWalletPublicKey(m.walletData)
					m.title = "Public Key"
					m.setState("output")
				case "private_key":
					m.output = displayWalletPrivateKey(m.walletData)
					m.title = "Private Key"
					m.setState("output")
				case "pk":
					m.title = "Private Key"
				case "sign_message":
					m.title = "Message to sign"
				case "save_keystore":
					m.title = "Keystore Password"
				}

				if m.state == "quit" {
					m.list.SetItems(getMainItems())
					m.setState("main")
					m.list.Title = "✨✨✨"
				}

				if ok {
					m.choice = item
				}
			}
		}

	case tea.WindowSizeMsg:
		top, right, bottom, left := docStyle.GetMargin()
		m.list.SetSize(msg.Width-left-right, msg.Height-top-bottom)
	}

	var cmd tea.Cmd

	if m.state == "main" || m.state == "access_wallet" {
		m.list, cmd = m.list.Update(msg)
	}

	if m.state == "pk" || m.state == "sign_message" || m.state == "save_keystore" || m.state == "keystore_access" {
		m.input, cmd = m.input.Update(msg)
	}

	if m.state == "sign_transaction" || m.state == "keystore_access" {
		cmd = m.updateInputs(msg)
	}

	return m, cmd
}

func getMainItems() []list.Item {
	items := []list.Item{
		ListItem{title: "New Wallet", desc: "Create a new wallet", id: "new_wallet"},
		ListItem{title: "Access Wallet", desc: "Access an existing wallet", id: "access_wallet"},
	}
	return items
}

func getAccessWalletItems() []list.Item {
	items := []list.Item{
		ListItem{title: "Private Key", desc: "Access your wallet using your private key", id: "pk"},
		ListItem{title: "Keystore File", desc: "Access a wallet using your keystore file", id: "keystore_access"},
		ListItem{title: "Quit", desc: "Quit to main menu", id: "quit"},
	}
	return items
}

func getControlWalletItems() []list.Item {
	items := []list.Item{
		ListItem{title: "Public Key", desc: "Display public key and QR", id: "public_key"},
		ListItem{title: "Private Key", desc: "Display private key and QR", id: "private_key"},
		ListItem{title: "Save Keystore", desc: "Save the wallet to a keystore file", id: "save_keystore"},
		ListItem{title: "Sign Message", desc: "Sign a message with the private key", id: "sign_message"},
		ListItem{title: "Sign Transaction", desc: "Sign a transaction with the private key", id: "sign_transaction"},
		ListItem{title: "Quit", desc: "Quit to main menu", id: "quit"},
	}
	return items
}

func getText() textinput.Model {
	ti := textinput.NewModel()
	ti.Placeholder = "Private Key"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50
	return ti
}

func getUI() UI {
	m := UI{list: list.NewModel(getMainItems(), list.NewDefaultDelegate(), 0, 0), input: getText(), state: "main"}
	return m
}

func (m *UI) setState(state string) {
	m.state = state
}

func dispalWalletPublicKey(walletData WalletData) string {
	return fmt.Sprintf(
		"%s\n%s",
		walletData.PublicKeyQR.ToSmallString(false),
		"Public Key: "+walletData.PublicKey,
	)
}
func displayWalletPrivateKey(walletData WalletData) string {
	return fmt.Sprintf(
		"%s\n%s",
		walletData.PrivateKeyQR.ToSmallString(false),
		"Private Key: "+walletData.PrivateKey,
	)
}

func (m *UI) setMultiInputView() {
	m.multiInput = make([]textinput.Model, 6)

	var t textinput.Model
	for i := range m.multiInput {
		t = textinput.NewModel()
		t.CursorStyle = cursorStyle

		switch i {
		case 0:
			t.Prompt = "Nonce: "
			t.Placeholder = "5"
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 1:
			t.Placeholder = "0x"
			t.CharLimit = 42
			t.Prompt = "To Address: "
		case 2:
			t.Prompt = "Value in ETH: "
			t.Placeholder = "0.01"
			t.CharLimit = 20
		case 3:
			t.Prompt = "Gas Limit: "
			t.Placeholder = "70000"
			t.CharLimit = 20
		case 4:
			t.Prompt = "Gas Price in GWEI: "
			t.Placeholder = "120"
			t.CharLimit = 20
		case 5:
			t.Prompt = "Data: "
			t.Placeholder = "0x"
		}

		m.multiInput[i] = t
	}
}

func (m *UI) setMultiInputViewKeystoreFile() {
	m.multiInput = make([]textinput.Model, 2)

	var t textinput.Model
	for i := range m.multiInput {
		t = textinput.NewModel()
		t.CursorStyle = cursorStyle

		switch i {
		case 0:
			t.Prompt = "Keystore File Path: "
			t.Placeholder = "./0x.keystore"
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 1:
			t.Placeholder = "secret"
			t.Prompt = "Password: "
			t.EchoCharacter = '⚬'
			t.EchoMode = textinput.EchoPassword
		}

		m.multiInput[i] = t
	}
}

func (m *UI) updateInputs(msg tea.Msg) tea.Cmd {
	var cmds = make([]tea.Cmd, len(m.multiInput))

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	for i := range m.multiInput {
		m.multiInput[i], cmds[i] = m.multiInput[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m UI) View() string {

	if m.choice.title != "" {
		switch m.state {

		case "sign_transaction":
			var b strings.Builder
			for i := range m.multiInput {
				b.WriteString(m.multiInput[i].View())
				if i < len(m.multiInput)-1 {
					b.WriteRune('\n')
				}
			}

			button := &blurredButton
			if m.focusIndex == len(m.multiInput) {
				button = &focusedButton
			}
			fmt.Fprintf(&b, "\n\n%s\n\n", *button)

			return b.String()

		case "keystore_access":
			var b strings.Builder
			for i := range m.multiInput {
				b.WriteString(m.multiInput[i].View())
				if i < len(m.multiInput)-1 {
					b.WriteRune('\n')
				}
			}

			button := &blurredButton
			if m.focusIndex == len(m.multiInput) {
				button = &focusedButton
			}
			fmt.Fprintf(&b, "\n\n%s\n\n", *button)

			return b.String()

		case "save_keystore", "pk", "sign_message":
			return docStyle.Render(fmt.Sprintf(
				"%s\n%s\n%s",
				titleStyle.Render(m.title),
				m.input.View(),
				blurredStyle.Render("Press ctrl+c to quit"),
			))

		case "output":
			in := fmt.Sprintf(
				"%s\n%s\n%s",
				titleStyle.Render(m.title),
				docStyle.Render(m.output),
				blurredStyle.Render("Press enter to continue"),
			)

			return docStyle.Render(in)
		}
	}

	return docStyle.Render(m.list.View())
}

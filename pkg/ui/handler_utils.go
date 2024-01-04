package ui

import (
	"ethtui/pkg/eth"
	"math/big"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

func requestProvider(m *UI) {
	setInputState(m, "Set Provider", "Enter provider URL", "update_provider")
}

func moveIndex(m UI, s string) (UI, []tea.Cmd) {
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

	return m, cmds
}

func sendERC20Tokens(m UI) (string, error) {
	toAddress := m.multiInput[0].Value()
	amount := m.multiInput[1].Value()
	contractAddress := m.multiInput[2].Value()

	// convert amount to big.Int
	amountInt, _ := strconv.Atoi(amount)
	amountWei := big.NewInt(int64(amountInt))
	amountWei = amountWei.Mul(amountWei, big.NewInt(1e18))

	return eth.TransferERC20Tokens(
		m.walletData,
		contractAddress,
		toAddress,
		amountWei,
		m.provider,
	)
}

func signTransaction(m UI) (string, error) {
	nonce, _ := strconv.Atoi(m.multiInput[0].Value())
	toAddress := m.multiInput[1].Value()
	value, _ := strconv.ParseFloat(m.multiInput[2].Value(), 64)
	gasLimit, _ := strconv.Atoi(m.multiInput[3].Value())
	gasPrice, _ := strconv.ParseFloat(m.multiInput[4].Value(), 64)
	data := m.multiInput[5].Value()
	chainId, _ := strconv.Atoi(m.multiInput[6].Value())
	gasTipCap, _ := strconv.ParseFloat(m.multiInput[7].Value(), 64)

	signedTransaction, err := m.walletData.SignTransaction(uint64(nonce), toAddress, value, gasLimit, gasPrice, data, int64(chainId), gasTipCap)
	if err != nil {
		return "", err
	}

	return signedTransaction, nil
}

func setOutputState(m *UI, title string, output string) {
	m.setState("output")
	m.title = title
	m.output = output
}

func setInputState(m *UI, title string, placeholder string, instate string) {
	m.setState("input")
	m.setInState(instate)
	m.title = title
	m.input = getText(placeholder)
}

func loadWalletState(m *UI, walletData eth.WalletData) {
	m.walletData = walletData
	m.loadListItems(
		getControlWalletItems(*m),
		m.walletData.PublicKey,
	)
}

func quitToMainMenu(m *UI) {
	m.list.SetItems(getMainItems())
	m.resetListCursor()
	m.setState("main")
	m.setListTitle("✨✨✨")
}

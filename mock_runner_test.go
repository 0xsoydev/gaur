package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

// MockCommandRunner implements CommandRunner for testing.
type MockCommandRunner struct {
	RunFunc         func(name string, args ...string) ([]byte, error)
	RunWithInputFunc func(input string, name string, args ...string) ([]byte, error)
	InteractiveFunc func(onExit func(error) tea.Msg, name string, args ...string) tea.Cmd
}

func (m *MockCommandRunner) Run(name string, args ...string) ([]byte, error) {
	if m.RunFunc != nil {
		return m.RunFunc(name, args...)
	}
	return nil, nil
}

func (m *MockCommandRunner) RunWithInput(input string, name string, args ...string) ([]byte, error) {
	if m.RunWithInputFunc != nil {
		return m.RunWithInputFunc(input, name, args...)
	}
	return nil, nil
}

func (m *MockCommandRunner) Interactive(onExit func(error) tea.Msg, name string, args ...string) tea.Cmd {
	if m.InteractiveFunc != nil {
		return m.InteractiveFunc(onExit, name, args...)
	}
	return nil
}

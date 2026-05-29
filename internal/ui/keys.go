package ui

import (
	"gioui.org/io/event"
	"gioui.org/io/key"
)

// KeyHandler routes function keys to App callbacks.
type KeyHandler struct {
	OnHelp       func()
	OnAttenuator func()
	OnSelectPage func(letter byte) // F3..F10 → 'A'..'H'
	OnToggleLog  func()             // F11
	OnSnapshot   func()             // F12
	OnEscape     func()
	OnSpace      func()
}

// Filters returns the set of key.Filter values to register each frame.
// Focus: nil so events fire regardless of which widget has focus.
func (h *KeyHandler) Filters() []event.Filter {
	return []event.Filter{
		key.Filter{Name: key.NameF1},
		key.Filter{Name: key.NameF2},
		key.Filter{Name: key.NameF3},
		key.Filter{Name: key.NameF4},
		key.Filter{Name: key.NameF5},
		key.Filter{Name: key.NameF6},
		key.Filter{Name: key.NameF7},
		key.Filter{Name: key.NameF8},
		key.Filter{Name: key.NameF9},
		key.Filter{Name: key.NameF10},
		key.Filter{Name: key.NameF11},
		key.Filter{Name: key.NameF12},
		key.Filter{Name: key.NameEscape},
		key.Filter{Name: key.NameSpace},
	}
}

// Handle dispatches one key event.
func (h *KeyHandler) Handle(e key.Event) {
	if e.State != key.Press {
		return
	}
	switch e.Name {
	case key.NameF1:
		if h.OnHelp != nil {
			h.OnHelp()
		}
	case key.NameF2:
		if h.OnAttenuator != nil {
			h.OnAttenuator()
		}
	case key.NameF3, key.NameF4, key.NameF5, key.NameF6,
		key.NameF7, key.NameF8, key.NameF9, key.NameF10:
		if h.OnSelectPage != nil {
			letter := byte('A') + byte(fkeyIndex(e.Name))
			h.OnSelectPage(letter)
		}
	case key.NameF11:
		if h.OnToggleLog != nil {
			h.OnToggleLog()
		}
	case key.NameF12:
		if h.OnSnapshot != nil {
			h.OnSnapshot()
		}
	case key.NameEscape:
		if h.OnEscape != nil {
			h.OnEscape()
		}
	case key.NameSpace:
		if h.OnSpace != nil {
			h.OnSpace()
		}
	}
}

func fkeyIndex(name key.Name) int {
	switch name {
	case key.NameF3:
		return 0
	case key.NameF4:
		return 1
	case key.NameF5:
		return 2
	case key.NameF6:
		return 3
	case key.NameF7:
		return 4
	case key.NameF8:
		return 5
	case key.NameF9:
		return 6
	case key.NameF10:
		return 7
	}
	return -1
}

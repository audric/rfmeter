package ui

import "testing"

func TestMenuToggle(t *testing.T) {
	var m Menu
	if m.open {
		t.Fatal("menu should start closed")
	}
	m.toggle()
	if !m.open {
		t.Fatal("toggle should open the menu")
	}
	m.toggle()
	if m.open {
		t.Fatal("toggle should close the menu")
	}
}

func TestMenuSelectClosesAndFires(t *testing.T) {
	var aboutFired, exitFired bool
	m := Menu{
		OnAbout: func() { aboutFired = true },
		OnExit:  func() { exitFired = true },
	}

	m.open = true
	m.selectAbout()
	if m.open {
		t.Error("selecting About should close the menu")
	}
	if !aboutFired {
		t.Error("selecting About should fire OnAbout")
	}

	m.open = true
	m.selectExit()
	if m.open {
		t.Error("selecting Exit should close the menu")
	}
	if !exitFired {
		t.Error("selecting Exit should fire OnExit")
	}
}

// Selecting an item with no callback wired must not panic.
func TestMenuSelectNilCallbacks(t *testing.T) {
	var m Menu
	m.open = true
	m.selectAbout()
	m.open = true
	m.selectExit()
}

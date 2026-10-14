// Package pick shows interactive lists in the terminal. It is the only package that uses huh.
package pick

import (
	"errors"
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

// maxHeight is the number of choices visible at once; the list scrolls past it.
const maxHeight = 15

// ErrCancelled means the user left the list with esc or ctrl+c.
var ErrCancelled = errors.New("cancelled")

// Choice is one line of a list.
type Choice struct {
	Label    string
	Selected bool // checked when the list opens (Many only)
}

// Picker asks the user to choose among lines.
type Picker interface {
	// Many shows a checklist and returns the indexes of the checked choices.
	Many(title string, choices []Choice) ([]int, error)
	// One shows a single-choice list and returns the index of the chosen one.
	One(title string, choices []Choice) (int, error)
}

// Huh implements Picker with charmbracelet/huh.
type Huh struct {
	In  io.Reader
	Out io.Writer
}

func (h Huh) Many(title string, choices []Choice) ([]int, error) {
	var picked []int
	field := manyField(title, choices, &picked)
	err := h.run(huh.NewForm(huh.NewGroup(keys{filterField: field, all: true})))
	return picked, err
}

// manyField builds the checklist; picked receives the checked indexes.
func manyField(title string, choices []Choice, picked *[]int) *huh.MultiSelect[int] {
	field := huh.NewMultiSelect[int]().
		Options(options(choices)...).
		Value(picked).
		Filterable(true).
		Height(min(len(choices), maxHeight) + 2) // + title and filter lines
	counter := func() string {
		return fmt.Sprintf("%s  %d of %d selected", title, len(*picked), len(choices))
	}
	// huh only computes a title func once the bindings change: show it from the start too.
	field.Title(counter()).TitleFunc(counter, picked)
	return field
}

func (h Huh) One(title string, choices []Choice) (int, error) {
	var picked int
	field := huh.NewSelect[int]().
		Title(title).
		Options(options(choices)...).
		Value(&picked).
		Height(min(len(choices), maxHeight) + 1) // + title line
	err := h.run(huh.NewForm(huh.NewGroup(keys{filterField: field})))
	return picked, err
}

func (h Huh) run(form *huh.Form) error {
	err := form.WithKeyMap(keymap()).WithInput(h.In).WithOutput(h.Out).WithShowHelp(true).Run()
	if errors.Is(err, huh.ErrUserAborted) {
		return ErrCancelled
	}
	return err
}

// clearFilter is the help text of huh's clear-filter key; keys looks for it.
const clearFilter = "clear filter"

// keymap is huh's default with gitix's help texts. "a" and esc are handled by keys.
func keymap() *huh.KeyMap {
	k := huh.NewDefaultKeyMap()
	k.MultiSelect.ClearFilter = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", clearFilter), key.WithDisabled())
	k.Select.ClearFilter = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", clearFilter), key.WithDisabled())
	k.MultiSelect.Toggle = key.NewBinding(key.WithKeys(" ", "x"), key.WithHelp("space", "toggle"))
	k.MultiSelect.Submit = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm"))
	k.MultiSelect.SelectAll = key.NewBinding(key.WithKeys("ctrl+a"), key.WithHelp("a", "select all"))
	k.MultiSelect.SelectNone = key.NewBinding(key.WithKeys("ctrl+a"), key.WithHelp("a", "select none"), key.WithDisabled())
	k.Select.Submit = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm"))
	return k
}

// options turns choices into huh options whose value is the choice index.
func options(choices []Choice) []huh.Option[int] {
	opts := make([]huh.Option[int], len(choices))
	for i, c := range choices {
		opts[i] = huh.NewOption(c.Label, i).Selected(c.Selected)
	}
	return opts
}

// filterField is a huh field that can filter its options, like Select and MultiSelect.
type filterField interface {
	huh.Field
	GetFiltering() bool
}

// keys adds gitix's shortcuts to a field: "a" selects all or none, and esc clears
// the filter if one is applied, else cancels. Both are ignored while typing a filter.
type keys struct {
	filterField
	all bool // map "a" to select all / none (MultiSelect only)
}

var cancel = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel"))

// filterApplied reports whether esc would clear a filter: huh enables its
// clear-filter key only when a filter is applied and not being typed.
func (k keys) filterApplied() bool {
	for _, b := range k.filterField.KeyBinds() {
		if b.Enabled() && b.Help().Desc == clearFilter {
			return true
		}
	}
	return false
}

// KeyBinds adds "esc cancel" to the help when esc cancels.
func (k keys) KeyBinds() []key.Binding {
	if k.GetFiltering() || k.filterApplied() {
		return k.filterField.KeyBinds()
	}
	return append(k.filterField.KeyBinds(), cancel)
}

func (k keys) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if press, ok := msg.(tea.KeyMsg); ok && !k.GetFiltering() {
		switch {
		case press.Type == tea.KeyEsc && !k.filterApplied():
			// The form only aborts on ctrl+c: send it one.
			return k, func() tea.Msg { return tea.KeyMsg{Type: tea.KeyCtrlC} }
		case k.all && press.String() == "a":
			msg = tea.KeyMsg{Type: tea.KeyCtrlA}
		}
	}
	_, cmd := k.filterField.Update(msg) // the field returns itself; keep the wrapper
	return k, cmd
}

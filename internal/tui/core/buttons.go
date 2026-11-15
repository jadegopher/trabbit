package core

import (
	"sort"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"github.com/samber/lo"
)

const (
	KeyShiftTab   = "shift+tab"
	KeyCtrlC      = "ctrl+c"
	KeyCtrlA      = "ctrl+a"
	KeyCtrlShiftA = "ctrl+shift+a"

	KeyTab          = "tab"
	KeyEnter        = "enter"
	KeyUp           = "up"
	KeyDown         = "down"
	KeyLeft         = "left"
	KeyRight        = "right"
	KeyBackspace    = "backspace"
	KeyEsc          = "esc"
	KeyQ            = "q"
	KeyA            = "a"
	KeyD            = "d"
	KeyE            = "e"
	KeyF            = "f"
	KeyG            = "g"
	KeyP            = "p"
	KeyR            = "r"
	KeyV            = "v"
	KeyCtrlR        = "ctrl+r"
	KeyOpenBracket  = "["
	KeyCloseBracket = "]"
	KeyB            = "b"
	KeyU            = "u"
	KeyShiftLeft    = "shift+left"
	KeyShiftRight   = "shift+right"
	KeyAny          = "any"

	KeyCtrlB           = "ctrl+b"
	KeyCtrlP           = "ctrl+p"
	KeyCtrlQ           = "ctrl+q"
	KeyCtrlEnter       = "ctrl+enter"
	KeyF5              = "f5"
	KeyF2              = "f2"
	KeyF3              = "f3"
	KeySuperEnter      = "super+enter"
	KeyMetaEnter       = "meta+enter"
	KeyAltEnter        = "alt+enter"
	KeyCtrlG           = "ctrl+g"
	KeyCtrlY           = "ctrl+y"
	KeySuperB          = "super+b"
	KeyMetaB           = "meta+b"
	KeyCtrlE           = "ctrl+e"
	KeyCtrlShiftC      = "ctrl+shift+c"
	KeyCtrlV           = "ctrl+v"
	KeyCtrlX           = "ctrl+x"
	KeyCtrlZ           = "ctrl+z"
	KeyCtrlShiftZ      = "ctrl+shift+z"
	KeySuperC          = "super+c"
	KeyMetaC           = "meta+c"
	KeySuperV          = "super+v"
	KeyMetaV           = "meta+v"
	KeySuperX          = "super+x"
	KeyMetaX           = "meta+x"
	KeySuperZ          = "super+z"
	KeyMetaZ           = "meta+z"
	KeySuperShiftZ     = "shift+super+z"
	KeyMetaShiftZ      = "shift+meta+z"
	KeyShiftHome       = "shift+home"
	KeyShiftEnd        = "shift+end"
	KeyCtrlShiftHome   = "ctrl+shift+home"
	KeyCtrlShiftEnd    = "ctrl+shift+end"
	KeySuperShiftLeft  = "shift+super+left"
	KeySuperShiftRight = "shift+super+right"
	KeySuperShiftUp    = "shift+super+up"
	KeySuperShiftDown  = "shift+super+down"
	KeyMetaShiftLeft   = "shift+meta+left"
	KeyMetaShiftRight  = "shift+meta+right"
	KeyMetaShiftUp     = "shift+meta+up"
	KeyMetaShiftDown   = "shift+meta+down"
	KeyHome            = "home"
	KeyEnd             = "end"
	KeyCtrlHome        = "ctrl+home"
	KeyCtrlEnd         = "ctrl+end"
	KeySuperLeft       = "super+left"
	KeySuperRight      = "super+right"
	KeySuperUp         = "super+up"
	KeySuperDown       = "super+down"
	KeyAltLess         = "alt+<"
	KeyAltGreater      = "alt+>"
	KeyPgUp            = "pgup"
	KeyPgDown          = "pgdown"
	KeyH               = "h"
	KeyL               = "l"
	KeySpace           = "space"
	KeyY               = "y"
	KeyUpperY          = "Y"
	KeyN               = "n"
	KeyUpperN          = "N"
)

const (
	symbolTab        = "⇥"
	symbolShiftTab   = "⇥"
	symbolUp         = "↑"
	symbolDown       = "↓"
	symbolLeft       = "←"
	symbolRight      = "→"
	symbolEnter      = "↵"
	symbolEsc        = "⎋"
	symbolCtrlA      = "^A"
	symbolCtrlShiftA = "^⇧A"
	symbolCtrlC      = "^C"
	symbolBackspace  = "⌫"
)

var KeySymbols = map[string]string{
	KeyTab:        symbolTab,
	KeyShiftTab:   symbolShiftTab,
	KeyUp:         symbolUp,
	KeyDown:       symbolDown,
	KeyLeft:       symbolLeft,
	KeyRight:      symbolRight,
	KeyEnter:      symbolEnter,
	KeyCtrlA:      symbolCtrlA,
	KeyCtrlShiftA: symbolCtrlShiftA,
	KeyCtrlC:      symbolCtrlC,
	KeyEsc:        symbolEsc,
	KeyBackspace:  symbolBackspace,
	KeySuperEnter: "⌘↵",
	KeyF5:         "F5",
	KeyF2:         "F2",
	KeyF3:         "F3",
}

const (
	HelpSectionNav = "navigate"
)

type Shortcut = key.Binding

func NewShortcut(help string, keys ...string) Shortcut {
	keys = lo.Uniq(keys)
	keys = sortKeys(keys)

	var symbols []string
	for _, k := range keys {
		symbols = append(symbols, GetSymbol(k))
	}

	return key.NewBinding(
		key.WithKeys(keys...),
		key.WithHelp(strings.Join(lo.Uniq(symbols), " "), help),
	)
}

func NavigateUp() Shortcut {
	return NewShortcut(HelpSectionNav, KeyUp, KeyShiftTab)
}

func NavigateDown() Shortcut {
	return NewShortcut(HelpSectionNav, KeyDown, KeyTab)
}

func NavigateLeft() Shortcut {
	return NewShortcut(HelpSectionNav, KeyLeft, KeyShiftTab)
}

func NavigateRight() Shortcut {
	return NewShortcut(HelpSectionNav, KeyRight, KeyTab)
}

func Submit() Shortcut {
	return NewShortcut("submit", KeyEnter)
}

func Cancel() Shortcut {
	return NewShortcut("cancel", KeyEsc, KeyCtrlQ)
}

func Esc() Shortcut {
	return NewShortcut("cancel", KeyEsc)
}

func Yes() Shortcut {
	return NewShortcut("select", KeyUpperY, KeyY)
}

func No() Shortcut {
	return NewShortcut("select", KeyUpperN, KeyN)
}

func Clean() Shortcut {
	return NewShortcut("clean focused", KeyCtrlA)
}

func CleanAll() Shortcut {
	return NewShortcut("clean all", KeyCtrlShiftA)
}

func Undo() Shortcut {
	shortcut := NewShortcut("undo", KeyCtrlZ)
	shortcut.SetKeys(KeyCtrlZ, KeySuperZ, KeyMetaZ)
	return shortcut
}

func Redo() Shortcut {
	shortcut := NewShortcut("redo", KeyCtrlShiftZ)
	shortcut.SetKeys(KeyCtrlShiftZ, KeySuperShiftZ, KeyMetaShiftZ, KeyCtrlY)
	return shortcut
}

func Copy() Shortcut {
	return NewShortcut("copy", KeyCtrlShiftC, KeySuperC, KeyMetaC)
}

func Cut() Shortcut {
	return NewShortcut("cut", KeyCtrlX, KeySuperX, KeyMetaX)
}

func Paste() Shortcut {
	return NewShortcut("paste", KeyCtrlV, KeySuperV, KeyMetaV)
}

func SelectAll() Shortcut {
	return NewShortcut("select all", KeyCtrlA)
}

func GetSymbol(key string) string {
	if symbol, ok := KeySymbols[key]; ok {
		return symbol
	}
	return key
}

// prepareHelp processes key bindings into a structured format for rendering.
// It solves three primary orchestration problems:
// 1. Grouping: Merges multiple bindings with the same help description into a single one (e.g., "↑ ↓: navigate").
// 2. Deduplication: Ensures that identical keys or symbols only appear once.
// 3. Consistency: Enforces a logical sorting order and format.
func prepareHelp(shortcuts ...key.Binding) []key.Binding {
	if len(shortcuts) == 0 {
		return nil
	}

	groups := lo.GroupBy(
		shortcuts, func(b key.Binding) string {
			h := b.Help()
			return h.Desc
		},
	)

	var helpOrder []string
	for _, s := range shortcuts {
		h := s.Help()
		if h.Desc != "" && !lo.Contains(helpOrder, h.Desc) {
			helpOrder = append(helpOrder, h.Desc)
		}
	}

	var merged []key.Binding
	for _, desc := range helpOrder {
		group := groups[desc]
		if len(group) == 1 {
			// Preserve explicit help that omits terminal-dependent aliases.
			merged = append(merged, group[0])
			continue
		}

		var allKeys []string
		for _, b := range group {
			allKeys = append(allKeys, b.Keys()...)
		}

		merged = append(merged, NewShortcut(desc, allKeys...))
	}

	return merged
}

func sortKeys(keys []string) []string {
	priority := map[string]int{
		KeyTab:      10,
		KeyShiftTab: 11,
		KeyUp:       20,
		KeyDown:     21,
		KeyLeft:     22,
		KeyRight:    23,
		KeyEnter:    30,
		KeyEsc:      40,
		KeyCtrlA:    50,
		KeyCtrlC:    51,
	}

	sort.Slice(
		keys, func(i, j int) bool {
			pI, okI := priority[keys[i]]
			if !okI {
				pI = 1000
			}
			pJ, okJ := priority[keys[j]]
			if !okJ {
				pJ = 1000
			}
			if pI != pJ {
				return pI < pJ
			}
			return keys[i] < keys[j]
		},
	)

	return keys
}

// Shortcuts adapts a list of shared bindings for the theme help renderer.
type Shortcuts []Shortcut

func (s Shortcuts) ShortHelp() []key.Binding { return s }

func (s Shortcuts) FullHelp() [][]key.Binding { return [][]key.Binding{s} }

func NextPage() Shortcut {
	return NewShortcut("Next", KeyRight)
}

func PreviousPage() Shortcut {
	return NewShortcut("Prev", KeyLeft)
}

func GoToPage() Shortcut {
	return NewShortcut("GoTo", KeyG)
}

func Toggle() Shortcut {
	return NewShortcut("toggle", KeySpace, KeyLeft, KeyRight)
}

func NavigateChoice() Shortcut {
	return NewShortcut("navigate", KeyTab, KeyLeft, KeyRight)
}

func CloseAny() Shortcut {
	return NewShortcut("close", KeyAny)
}

func LineStart() Shortcut {
	return NewShortcut("line start", KeyHome, KeySuperLeft)
}

func LineEnd() Shortcut {
	return NewShortcut("line end", KeyEnd, KeyCtrlE, KeySuperRight)
}

func DocumentStart() Shortcut {
	return NewShortcut("document start", KeyCtrlHome, KeySuperUp, KeyAltLess)
}

func DocumentEnd() Shortcut {
	return NewShortcut("document end", KeyCtrlEnd, KeySuperDown, KeyAltGreater)
}

func Publish() Shortcut {
	shortcut := NewShortcut("publish", KeyF5)
	shortcut.SetKeys(KeyF5, KeyCtrlEnter)
	return shortcut
}

func AddKeyValue() Shortcut {
	return NewShortcut("add pair", KeyF2)
}

func DeleteKeyValue() Shortcut {
	return NewShortcut("delete pair", KeyF3)
}

func Quit() Shortcut {
	return NewShortcut("quit", KeyCtrlC)
}

func ShowHelp() Shortcut {
	return NewShortcut("help", KeyCtrlG)
}

// ClearText uses a free editor binding; Command alternatives collide with IDE actions.
func ClearText() Shortcut {
	return NewShortcut("clear", KeyCtrlR)
}

func Beautify() Shortcut {
	shortcut := NewShortcut("beautify", KeyCtrlB)
	shortcut.SetKeys(KeyCtrlB, KeySuperB, KeyMetaB)
	return shortcut
}

func SwitchPanel() Shortcut {
	return NewShortcut("panel", KeyTab, KeyShiftTab)
}

func Exit() Shortcut {
	return NewShortcut("exit", KeyEsc, KeyCtrlQ)
}

func ResetFilter() Shortcut {
	return NewShortcut("reset filter", KeyCtrlR)
}

func AddQueue() Shortcut {
	return NewShortcut("Add q", KeyA)
}

func DeleteQueue() Shortcut {
	return NewShortcut("Delete q", KeyD)
}

func PurgeQueue() Shortcut {
	return NewShortcut("Purge q", KeyE)
}

func PublishQueue() Shortcut {
	return NewShortcut("Publish", KeyP)
}

func Filter() Shortcut {
	return NewShortcut("Filter q", KeyF)
}

func ViewSettings() Shortcut {
	return NewShortcut("View", KeyV)
}

func SwitchFocus() Shortcut {
	return NewShortcut("Switch Focus", KeyTab, KeyShiftTab)
}

func SwitchSubtab() Shortcut {
	return NewShortcut("Subtab", KeyOpenBracket, KeyCloseBracket)
}

func AddBinding() Shortcut { return NewShortcut("Add binding", KeyB) }

func Unbind() Shortcut { return NewShortcut("Unbind", KeyU) }

func TableScroll() Shortcut { return NewShortcut("Scroll", KeyUp, KeyDown) }

func TablePageScroll() Shortcut { return NewShortcut("Scroll page", KeyPgUp, KeyPgDown) }

func TableHorizontalScroll() Shortcut {
	return NewShortcut("Columns", KeyShiftLeft, KeyShiftRight)
}

func RefreshSettings() Shortcut {
	return NewShortcut("refresh", KeyR)
}

func SelectPrevious() Shortcut {
	return NewShortcut("select", KeyLeft, KeyH)
}

func SelectNext() Shortcut {
	return NewShortcut("select", KeyRight, KeyL, KeySpace, KeyEnter)
}

// TabNumber keeps numbered tab labels and input matching on the same binding.
func TabNumber(number int, label string) Shortcut {
	return NewShortcut(label, strconv.Itoa(number))
}

func DeleteLineEnd(binding Shortcut) Shortcut {
	return NewShortcut("delete line end", binding.Keys()...)
}

func DeleteLineStart(binding Shortcut) Shortcut {
	return NewShortcut("delete line start", binding.Keys()...)
}

func SelectLineStart() Shortcut {
	return NewShortcut("select to line start", KeyShiftHome, KeySuperShiftLeft, KeyMetaShiftLeft)
}

func SelectLineEnd() Shortcut {
	return NewShortcut("select to line end", KeyShiftEnd, KeySuperShiftRight, KeyMetaShiftRight)
}

func SelectDocumentStart() Shortcut {
	return NewShortcut("select to document start", KeyCtrlShiftHome, KeySuperShiftUp, KeyMetaShiftUp)
}

func SelectDocumentEnd() Shortcut {
	return NewShortcut("select to document end", KeyCtrlShiftEnd, KeySuperShiftDown, KeyMetaShiftDown)
}

// ShortcutReference gives a readable reference without exposing protocol aliases.
// Meta and Super remain accepted input, but both are displayed as Command.
func ShortcutReference(binding Shortcut) Shortcut {
	var labels []string
	for _, chord := range binding.Keys() {
		parts := strings.Split(chord, "+")
		for i, part := range parts {
			switch part {
			case "ctrl":
				parts[i] = "Ctrl"
			case "super", "meta":
				parts[i] = "Cmd"
			case "shift":
				parts[i] = "Shift"
			case "alt":
				parts[i] = "Alt"
			case KeyLeft, KeyRight, KeyUp, KeyDown:
				parts[i] = GetSymbol(part)
			default:
				if len(part) == 1 {
					parts[i] = strings.ToUpper(part)
				} else if part != "" {
					parts[i] = strings.ToUpper(part[:1]) + part[1:]
				}
			}
		}
		labels = append(labels, strings.Join(parts, "+"))
	}
	binding.SetHelp(strings.Join(lo.Uniq(labels), " / "), binding.Help().Desc)
	return binding
}

// WithoutShortcuts removes keys intercepted by an owning form from editor help and input.
func WithoutShortcuts(binding Shortcut, intercepted ...Shortcut) Shortcut {
	var keys []string
	for _, candidate := range binding.Keys() {
		blocked := false
		for _, shortcut := range intercepted {
			blocked = blocked || lo.Contains(shortcut.Keys(), candidate)
		}
		if !blocked {
			keys = append(keys, candidate)
		}
	}
	return NewShortcut(binding.Help().Desc, keys...)
}

// SelectText combines the editor's live directional selection bindings.
func SelectText(bindings ...Shortcut) Shortcut {
	var keys []string
	for _, binding := range bindings {
		if binding.Enabled() {
			keys = append(keys, binding.Keys()...)
		}
	}
	return NewShortcut("select text", keys...)
}

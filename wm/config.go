package wm

import (
	"strings"

	"github.com/BurntSushi/xgbutil/ewmh"

	"github.com/BurntSushi/wingo/logger"
	"github.com/BurntSushi/wingo/wini"
)

type Configuration struct {
	mouse                 map[string][]mouseCommand
	key                   map[string][]keyCommand
	Ffm                   bool
	Workspaces            []string
	AlwaysFloating        []string
	ConfirmKey, CancelKey string
	BackspaceKey          string
	TabKey, RevTabKey     string
}

// newConfig
func newConfig() *Configuration {
	return &Configuration{
		mouse:          map[string][]mouseCommand{},
		key:            map[string][]keyCommand{},
		Ffm:            false,
		Workspaces:     []string{"1", "2", "3", "4"},
		AlwaysFloating: []string{},
		ConfirmKey:     "Return",
		CancelKey:      "Escape",
		BackspaceKey:   "BackSpace",
		TabKey:         "Tab",
		RevTabKey:      "ISO_Left_Tab",
	}
}

// loadConfig reads all configuration files and loads them into the
// a single config value.
//
// Most of this code is incredibly boring.
func loadConfig() (*Configuration, error) {
	conf := newConfig() // globally defined in wingo.go

	type confFile struct {
		fpath       string
		loadSection func(*Configuration, *wini.Data, string)
	}
	cfiles := []confFile{
		{"config/mouse.wini", (*Configuration).loadMouseConfigSection},
		{"config/key.wini", (*Configuration).loadKeyConfigSection},
		{"config/options.wini", (*Configuration).loadOptionsConfigSection},
	}
	for _, cfile := range cfiles {
		cdata, err := wini.Parse(cfile.fpath)
		if err != nil {
			return nil, err
		}
		for _, section := range cdata.Sections() {
			cfile.loadSection(conf, cdata, section)
		}
	}
	return conf, nil
}

// loadMouseConfigSection does two things:
// 1) Inspects the section name to infer the identifier. In general, the
// "mouse" prefix is removed, and whatever remains is the identifier. There
// are two special cases: "MouseBorders*" turns into "borders_*" and
// "MouseFull*" turns into "full_*".
// 2) Constructs a "mouseCommand" for *every* value.
//
// The idents are used for attaching mouse commands to the corresponding
// frames. (See the mouseCommand methods.)
func (conf *Configuration) loadMouseConfigSection(
	cdata *wini.Data, section string) {

	ident := ""
	switch {
	case len(section) > 7 && section[:7] == "borders":
		ident = "borders_" + section[7:]
	case len(section) > 4 && section[:4] == "full":
		ident = "full_" + section[4:]
	default:
		ident = section
	}

	for _, key := range cdata.Keys(section) {
		mouseStr := key.Name()
		for _, cmd := range key.Strings() {
			if _, ok := conf.mouse[ident]; !ok {
				conf.mouse[ident] = make([]mouseCommand, 0)
			}

			if err := gribbleEnv.Check(cmd); err != nil {
				logger.Warning.Printf(
					"Could not parse command '%s' because: %s", cmd, err)
			} else {
				down, justMouseStr := isDown(mouseStr)
				mcmd := mouseCommand{
					cmdStr:    cmd,
					cmdName:   gribbleEnv.CommandName(cmd),
					down:      down,
					buttonStr: justMouseStr,
				}
				conf.mouse[ident] = append(conf.mouse[ident], mcmd)
			}
		}
	}
}

func (conf *Configuration) loadKeyConfigSection(
	cdata *wini.Data, section string) {

	for _, key := range cdata.Keys(section) {
		keyStr := key.Name()
		for _, cmd := range key.Strings() {
			if _, ok := conf.key[section]; !ok {
				conf.key[section] = make([]keyCommand, 0)
			}

			if err := gribbleEnv.Check(cmd); err != nil {
				logger.Warning.Printf(
					"Could not parse command '%s' because: %s", cmd, err)
			} else {
				down, justKeyStr := isDown(keyStr)
				kcmd := keyCommand{
					cmdStr:  cmd,
					cmdName: gribbleEnv.CommandName(cmd),
					down:    down,
					keyStr:  justKeyStr,
				}
				conf.key[section] = append(conf.key[section], kcmd)
			}
		}
	}
}

func (conf *Configuration) loadOptionsConfigSection(
	cdata *wini.Data, section string) {

	for _, key := range cdata.Keys(section) {
		option := key.Name()
		switch option {
		case "workspaces":
			if workspaces, ok := getLastString(key); ok {
				conf.Workspaces = strings.Split(workspaces, " ")
			}
		case "always_floating":
			if alwaysFloating, ok := getLastString(key); ok {
				conf.AlwaysFloating = strings.Split(alwaysFloating, " ")
			}
		case "focus_follows_mouse":
			setBool(key, &conf.Ffm)
		case "cancel":
			setString(key, &conf.CancelKey)
		case "confirm":
			setString(key, &conf.ConfirmKey)
		}
	}
}

// strToDirection converts a string representation of a mouse direction
// to an xgbutil.ewmh constant value. It is case insensitive.
func strToDirection(s string) uint32 {
	switch strings.ToLower(s) {
	case "top":
		return ewmh.SizeTop
	case "bottom":
		return ewmh.SizeBottom
	case "left":
		return ewmh.SizeLeft
	case "right":
		return ewmh.SizeRight
	case "topleft":
		return ewmh.SizeTopLeft
	case "topright":
		return ewmh.SizeTopRight
	case "bottomleft":
		return ewmh.SizeBottomLeft
	case "bottomright":
		return ewmh.SizeBottomRight
	}
	return ewmh.Infer
}

// isDown takes a key/mouse combination, and looks for the keyword "up".
// If "up" exists, isDown returns false. Otherwise, true.
// It also returns the key/mouse string without "up" or "down".
func isDown(keyStr string) (bool, string) {
	spacei := strings.Index(keyStr, " ")
	down := true
	if spacei > -1 {
		if strings.ToLower(keyStr[spacei+1:]) == "up" {
			down = false
		}
		keyStr = keyStr[:spacei]
	}
	return down, keyStr
}
package main

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/xgbutil/ewmh"
	"github.com/BurntSushi/xgbutil/xwindow"

	"github.com/BurntSushi/wingo/logger"
)

func (wingo *wingoState) ewmhSupportingWmCheck() {
	supportingWin := xwindow.Must(xwindow.Create(X, wingo.root.Id))
	ewmh.SupportingWmCheckSet(X, wingo.root.Id, supportingWin.Id)
	ewmh.SupportingWmCheckSet(X, supportingWin.Id, supportingWin.Id)
	ewmh.WmNameSet(X, supportingWin.Id, "Wingo")
}

func (wingo *wingoState) ewmhDesktopNames() {
	if wingo == nil || wingo.heads == nil {
		return // still starting up
	}

	names := make([]string, len(wingo.heads.Workspaces()))
	for i, wrk := range wingo.heads.Workspaces() {
		if len(strings.TrimSpace(wrk.String())) == 0 {
			names[i] = fmt.Sprintf("Default workspace %d", i)
		} else {
			names[i] = wrk.String()
		}
	}
	ewmh.DesktopNamesSet(X, names)
}

// ewmhWorkarea is responsible for syncing _NET_WORKAREA with the current
// workspace state.
// Since multiple workspaces can be viewable at one time, this property
// doesn't make much sense. So I'm not going to implement it until it's obvious
// that I have to.
func (wingo *wingoState) ewmhWorkarea() {
}

// ewmhDesktopGeometry is another totally useless property. Christ.
func (wingo *wingoState) ewmhDesktopGeometry() {
	rootGeom, err := wingo.root.Geometry()
	if err != nil {
		logger.Error.Printf("Could not get ROOT window geometry: %s", err)
		panic("")
	}

	ewmh.DesktopGeometrySet(X,
		&ewmh.DesktopGeometry{
			Width:  rootGeom.Width(),
			Height: rootGeom.Height(),
		})
}

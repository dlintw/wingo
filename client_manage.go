package main

import (
	"github.com/BurntSushi/xgb/xproto"

	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/ewmh"
	"github.com/BurntSushi/xgbutil/icccm"
	"github.com/BurntSushi/xgbutil/xwindow"

	"github.com/BurntSushi/wingo/focus"
	"github.com/BurntSushi/wingo/frame"
	"github.com/BurntSushi/wingo/logger"
	"github.com/BurntSushi/wingo/stack"
)

func newClient(X *xgbutil.XUtil, id xproto.Window) *client {
	X.Grab()
	defer X.Ungrab()

	if client := wingo.findManagedClient(id); client != nil {
		logger.Message.Printf("Already managing client: %s", client)
		return nil
	}

	win := xwindow.New(X, id)
	if _, err := win.Geometry(); err != nil {
		logger.Warning.Printf("Could not manage client %d because: %s", id, err)
		return nil
	}

	c := &client{
		X:           X,
		win:         win,
		name:        "N/A",
		state:       frame.Inactive,
		layer:       stack.LayerDefault,
		maximized:   false,
		iconified:   false,
		unmapIgnore: 0,
		floating:    false,
	}

	c.manage()
	if !c.iconified {
		c.Map()
		if c.primaryType == clientTypeNormal {
			focus.Focus(c)
		}
	}
	return c
}

func (c *client) manage() {
	c.refreshName()
	logger.Message.Printf("Managing new client: %s", c)

	c.fetchXProperties()
	c.setPrimaryType()
	c.setInitialLayer()

	// Determine whether the client should start iconified or not.
	c.iconified = c.nhints.Flags&icccm.HintState > 0 &&
		c.hints.InitialState == icccm.StateIconic

	// newClientFrames sets c.frame.
	c.frames = c.newClientFrames()
	c.states = c.newClientStates()
	c.prompts = c.newClientPrompts()

	wingo.add(c)
	focus.InitialAdd(c)
	stack.Raise(c)
	wingo.workspace().Add(c)
	c.attachEventCallbacks()
}

func (c *client) fetchXProperties() {
	var err error

	c.hints, err = icccm.WmHintsGet(c.X, c.Id())
	if err != nil {
		logger.Warning.Println(err)
		logger.Message.Printf("Using reasonable defaults for WM_HINTS for %X",
			c.Id())
		c.hints = &icccm.Hints{
			Flags:        icccm.HintInput | icccm.HintState,
			Input:        1,
			InitialState: icccm.StateNormal,
		}
	}

	c.nhints, err = icccm.WmNormalHintsGet(c.X, c.Id())
	if err != nil {
		logger.Warning.Println(err)
		logger.Message.Printf("Using reasonable defaults for WM_NORMAL_HINTS "+
			"for %X", c.Id())
		c.nhints = &icccm.NormalHints{}
	}

	c.protocols, err = icccm.WmProtocolsGet(c.X, c.Id())
	if err != nil {
		logger.Warning.Printf(
			"Window %X does not have WM_PROTOCOLS set.", c.Id())
	}

	c.winTypes, err = ewmh.WmWindowTypeGet(c.X, c.Id())
	if err != nil {
		logger.Warning.Printf("Could not find window type for window %X, "+
			"using 'normal'.", c.Id())
		c.winTypes = []string{"_NET_WM_WINDOW_TYPE_NORMAL"}
	}

	trans, _ := icccm.WmTransientForGet(c.X, c.Id())
	if trans == 0 {
		for _, c2 := range wingo.clients {
			if c2.transient(c) {
				c.transientFor = c2
				break
			}
		}
	} else if transCli := wingo.findManagedClient(trans); transCli != nil {
		c.transientFor = transCli
	}
}

func (c *client) setPrimaryType() {
	switch {
	case c.hasType("_NET_WM_WINDOW_TYPE_DESKTOP"):
		c.primaryType = clientTypeDesktop
	case c.hasType("_NET_WM_WINDOW_TYPE_DOCK"):
		c.primaryType = clientTypeDock
	default:
		c.primaryType = clientTypeNormal
	}
}

func (c *client) setInitialLayer() {
	switch c.primaryType {
	case clientTypeDesktop:
		c.layer = stack.LayerDesktop
	case clientTypeDock:
		c.layer = stack.LayerDock
	case clientTypeNormal:
		c.layer = stack.LayerDefault
	default:
		panic("Unimplemented client type.")
	}
}

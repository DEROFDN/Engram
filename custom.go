// Copyright 2023-2024 DERO Foundation. All rights reserved.
// Use of this source code in any form is governed by RESEARCH license.
// license can be found in the LICENSE file.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY
// EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL
// THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
// PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT,
// STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF
// THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

type returnEntry struct {
	widget.Entry
	OnReturn func()
}

// NewReturnEntry creates a new single line entry widget that executes a function when the
// return key is pressed
func NewReturnEntry() *returnEntry {
	entry := &returnEntry{}
	entry.ExtendBaseWidget(entry)
	return entry
}

func (e *returnEntry) TypedKey(key *fyne.KeyEvent) {
	switch key.Name {
	case fyne.KeyReturn:
		e.OnReturn()
	default:
		e.Entry.TypedKey(key)
	}
}

var _ fyne.Draggable = (*iframe)(nil)

type iframe struct {
	widget.BaseWidget
}

func (o *iframe) CreateRenderer() fyne.WidgetRenderer {
	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(ui.MaxWidth*0.99, ui.MaxHeight*0.99))
	o.ExtendBaseWidget(o)
	return &iframeRenderer{
		rect: rect,
	}
}

func (o *iframe) MinSize() fyne.Size {
	o.ExtendBaseWidget(o)
	return o.BaseWidget.MinSize()
}

func (o *iframe) Tapped(e *fyne.PointEvent) {

}

func (o *iframe) TappedSecondary(e *fyne.PointEvent) {

}

func (o *iframe) Dragged(e *fyne.DragEvent) {
	if engram.Disk != nil {
		if nav.PosX == 0 && nav.PosY == 0 {
			nav.PosX = e.Position.X
			nav.PosY = e.Position.Y
		}
		nav.CurX = e.Position.X
		nav.CurY = e.Position.Y
	}
}

func (o *iframe) DragEnd() {
	/*
		if engram.Disk != nil {
			if nav.CurX > nav.PosX+30 {
				if session.Domain == "app.wallet" {
					session.Window.SetContent(layoutTransition())
					session.Window.SetContent(layoutIdentity())
				} else if session.Domain == "app.cyberdeck" {
					session.Window.SetContent(layoutTransition())
					session.Window.SetContent(layoutDashboard())
				}
			} else if nav.CurX < nav.PosX-30 {
				if session.Domain == "app.wallet" {
					session.Window.SetContent(layoutTransition())
					session.Window.SetContent(layoutCyberdeck())
				} else if session.Domain == "app.Identity" {
					session.Window.SetContent(layoutTransition())
					session.Window.SetContent(layoutDashboard())
				}
			} else if nav.CurY > nav.PosY+30 {
				if session.Domain == "app.wallet" {
					session.Window.SetContent(layoutTransition())
					session.Window.SetContent(layoutTransfers())
				} else if session.Domain == "app.messages" {
					session.Window.SetContent(layoutTransition())
					session.Window.SetContent(layoutDashboard())
				} else if session.Domain == "app.messages.contact" {
					session.Window.SetContent(layoutTransition())
					session.Window.SetContent(layoutMessages())
				}
			} else if nav.CurY < nav.PosY-30 {
				if session.Domain == "app.wallet" {
					session.Window.SetContent(layoutTransition())
					session.Window.SetContent(layoutMessages())
				} else if session.Domain == "app.transfers" {
					session.Window.SetContent(layoutTransition())
					session.Window.SetContent(layoutDashboard())
				}
			}

			nav.PosX = 0
			nav.PosY = 0
		}
	*/
	if engram.Disk != nil {
		if nav.CurY > nav.PosY+30 {
			if session.Domain == appMessages {
				session.Window.SetContent(layoutTransition())
				session.Window.SetContent(layoutMessages())
			} else if session.Domain == appMessagesContact {
				session.Window.SetContent(layoutTransition())
				session.Window.SetContent(layoutPM())
			} else if session.Domain == appIdentity {
				session.Window.SetContent(layoutTransition())
				session.Window.SetContent(layoutIdentity())
			}
		}
	}
}

var _ fyne.WidgetRenderer = (*iframeRenderer)(nil)

type iframeRenderer struct {
	rect *canvas.Rectangle
}

func (o *iframeRenderer) BackgroundColor() color.Color {
	return color.Transparent
}

func (o *iframeRenderer) Destroy() {
}

func (o *iframeRenderer) Layout(size fyne.Size) {
	o.rect.Resize(size)
}

func (o *iframeRenderer) MinSize() fyne.Size {
	return o.rect.MinSize()
}

func (o *iframeRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{o.rect}
}

func (o *iframeRenderer) Refresh() {
	o.rect.Refresh()
}

type mobileEntry struct {
	widget.Entry
	OnFocusLost   func()
	OnFocusGained func()
}

// NewMobileEntry creates a new single line entry widget with more options for mobile devices
func NewMobileEntry() *mobileEntry {
	entry := &mobileEntry{}
	entry.ExtendBaseWidget(entry)
	return entry
}

func (o *mobileEntry) FocusGained() {
	o.Entry.FocusGained()
	o.OnFocusGained()
}

type contextMenuButton struct {
	widget.Button
	menu *fyne.Menu
}

func (o *contextMenuButton) Tapped(e *fyne.PointEvent) {
	widget.ShowPopUpMenuAtPosition(
		o.menu,
		fyne.CurrentApp().Driver().CanvasForObject(o),
		e.AbsolutePosition,
	)
}

// NewContextMenuButton creates a new button widget with a dropdown menu
func NewContextMenuButton(label string, image fyne.Resource, menu *fyne.Menu) *contextMenuButton {
	o := &contextMenuButton{menu: menu}
	o.Text = label
	o.SetIcon(image)
	o.ExtendBaseWidget(o)
	return o
}

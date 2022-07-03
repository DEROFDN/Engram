// Copyright 2021-2022 DERO Foundation. All rights reserved.
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
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// AnimatedGif widget shows a Gif image with many frames.
type AnimatedGif struct {
	widget.BaseWidget
	min fyne.Size

	src               *gif.GIF
	dst               *canvas.Image
	remaining         int
	stopping, running bool
	runLock           sync.RWMutex
}

type ImageButton struct {
	widget.BaseWidget
	Image             *canvas.Image
	Res               fyne.Resource
	OnTapped          func()
	OnTappedSecondary func()
}

type ImageButtonRenderer struct {
	ImageButton *ImageButton
	Object      *canvas.Image
}

type returnEntry struct {
	widget.Entry
}

type tapRect struct {
	canvas.Rectangle
	Content fyne.CanvasObject
	Object  fyne.CanvasObject
	Fill    color.Color
}

type tapRectRenderer struct {
	e *tapRect
}

// NewAnimatedGif creates a new widget loaded to show the specified image.
// If there is an error loading the image it will be returned in the error value.

// CreateRenderer loads the widget renderer for this widget. This is an internal requirement for Fyne.
func (g *AnimatedGif) CreateRenderer() fyne.WidgetRenderer {
	return &gifRenderer{gif: g}
}

// MinSize returns the minimum size that this GIF can occupy.
// Because gif images are measured in pixels we cannot use the dimensions, so this defaults to 0x0.
// You can set a minimum size if required using SetMinSize.
func (g *AnimatedGif) MinSize() fyne.Size {
	return g.min
}

// SetMinSize sets the smallest possible size that this AnimatedGif should be drawn at.
// Be careful not to set this based on pixel sizes as that will vary based on output device.
func (g *AnimatedGif) SetMinSize(min fyne.Size) {
	g.min = min
}

func newGif(f fyne.Resource) (*AnimatedGif, error) {

	ret := &AnimatedGif{}
	ret.ExtendBaseWidget(ret)
	ret.dst = &canvas.Image{}
	ret.dst.FillMode = canvas.ImageFillContain

	return ret, ret.Load(f)
}

func (g *AnimatedGif) Load(f fyne.Resource) (err error) {
	if f == nil {
		return
	}

	data := f.Content()
	g.CreateRenderer()
	g.dst.Image = nil
	g.dst.Refresh()

	img, err := gif.DecodeAll(bytes.NewReader(data))

	w := new(bytes.Buffer)
	err = gif.EncodeAll(w, img)
	if err != nil {
		return
	}

	buffer, err := gif.DecodeAll(w)
	if err != nil {
		return
	}

	g.src = buffer
	g.dst.Image = buffer.Image[0]
	g.dst.Refresh()
	g.Refresh()

	return nil
}

// Start begins the animation. The speed of the transition is controlled by the loaded gif file.
func (g *AnimatedGif) Start() {
	if g.isRunning() {
		return
	}
	g.runLock.Lock()
	g.running = true
	g.runLock.Unlock()

	buffer := image.NewNRGBA(g.dst.Image.Bounds())
	//draw.Draw(buffer, g.dst.Image.Bounds(), g.src.Image[0], image.Point{}, draw.Src)
	g.dst.Image = buffer
	g.dst.Refresh()

	go func() {
		g.remaining = -1

		for g.remaining != 0 {
			for c, srcImg := range g.src.Image {
				if g.isStopping() {
					break
				}
				draw.Draw(buffer, g.dst.Image.Bounds(), srcImg, image.Point{}, draw.Src)
				g.dst.Refresh()

				time.Sleep(time.Millisecond * time.Duration(g.src.Delay[c]) * 0)
			}
		}

		g.running = false
	}()
}

// Stop will request that the animation stops running, the last frame will remain visible
func (g *AnimatedGif) Stop() {
	g.runLock.Lock()
	g.stopping = true
	g.runLock.Unlock()
}

func (g *AnimatedGif) isStopping() bool {
	g.runLock.RLock()
	defer g.runLock.RUnlock()
	return g.stopping
}

func (g *AnimatedGif) isRunning() bool {
	g.runLock.RLock()
	defer g.runLock.RUnlock()
	return g.running
}

type gifRenderer struct {
	gif *AnimatedGif
}

func (g *gifRenderer) Destroy() {
	g.gif.Stop()
}

func (g *gifRenderer) Layout(size fyne.Size) {
	g.gif.dst.Resize(size)
}

func (g *gifRenderer) MinSize() fyne.Size {
	return g.gif.MinSize()
}

func (g *gifRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{g.gif.dst}
}

func (g *gifRenderer) Refresh() {
	g.gif.dst.Refresh()
}

func newImageButton(r fyne.Resource, tapFunc func()) *ImageButton {
	img := canvas.NewImageFromResource(r)
	img.ScaleMode = canvas.ImageScaleSmooth
	img.FillMode = canvas.ImageFillOriginal
	w := &ImageButton{Image: img, Res: r}
	w.OnTapped = tapFunc
	w.OnTappedSecondary = tapFunc
	w.SetMinSize(fyne.NewSize(50, 50))
	w.ExtendBaseWidget(w)
	return w
}

func (o *ImageButton) SetMinSize(s fyne.Size) {
	o.Image.SetMinSize(s)
	return
}

func (o *ImageButton) Tapped(ev *fyne.PointEvent) {
	o.OnTapped()
}

func (o *ImageButton) TappedSecondary(ev *fyne.PointEvent) {
	o.OnTapped()
}

func (o *ImageButton) CreateRenderer() fyne.WidgetRenderer {
	img := canvas.NewImageFromResource(o.Res)
	img.SetMinSize(fyne.NewSize(50, 50))
	img.FillMode = canvas.ImageFillOriginal
	return &ImageButtonRenderer{
		ImageButton: o,
		Object:      img,
	}
}

func (r *ImageButtonRenderer) Layout(fyne.Size) {
	r.Object.ScaleMode = canvas.ImageScalePixels
	r.Object.SetMinSize(fyne.NewSize(50, 50))
	r.Object.Resize(fyne.NewSize(50, 50))
}

func (r *ImageButtonRenderer) MinSize() fyne.Size {
	size := r.Object.MinSize()
	return size
}

func (r *ImageButtonRenderer) Refresh() {
	r.Layout(r.ImageButton.MinSize())
	r.Object.Resource = r.ImageButton.Res
	canvas.Refresh(r.ImageButton)
	canvas.Refresh(r.Object)
}

func (r *ImageButtonRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.Object}
}

func (r *ImageButtonRenderer) Destroy() {

}

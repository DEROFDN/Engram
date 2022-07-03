//	Copyright 2021-2022 DERO Foundation. All rights reserved.

package main

import (
	"fyne.io/fyne/v2"
)

// Declare conformity with Layout interface
var _ fyne.Layout = (*Screen)(nil)

type Screen struct {
}

// NewScreen creates a new Screen instance
func NewScreen() fyne.Layout {
	return &Screen{}
}

func (s *Screen) MinSize(objects []fyne.CanvasObject) fyne.Size {
	w, h := float32(MIN_WIDTH), float32(MIN_HEIGHT)

	return fyne.NewSize(w, h)
}

func (s *Screen) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	topLeft := fyne.NewPos(0, 0)
	for _, child := range objects {
		child.Resize(size)
		child.Move(topLeft)
	}
}

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
	"fyne.io/fyne/v2/theme"
)

type eTheme struct{}

func (eTheme) Color(c fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	switch c {
	case theme.ColorNameBackground:
		return color.NRGBA{
			R: 21,
			G: 23,
			B: 30,
			A: 0xff,
		}
	case theme.ColorNameHyperlink:
		return color.NRGBA{
			R: 235,
			G: 235,
			B: 235,
			A: 0x99,
		}
	case theme.ColorNameButton:
		return color.NRGBA{
			R: 19,
			G: 202,
			B: 105,
			A: 0x75,
		}
	case theme.ColorNameDisabledButton:
		return color.NRGBA{
			R: 19,
			G: 202,
			B: 105,
			A: 0x22,
		}
	case theme.ColorNameDisabled:
		return color.NRGBA{
			R: 164,
			G: 164,
			B: 164,
			A: 0x42,
		}
	case theme.ColorNameError:
		return color.NRGBA{
			R: 0xf4,
			G: 0x43,
			B: 0x36,
			A: 0xff,
		}
	case theme.ColorNameFocus:
		return color.NRGBA{
			R: 19,
			G: 202,
			B: 105,
			A: 0x88,
		}
	case theme.ColorNameForeground:
		return color.NRGBA{
			R: 208,
			G: 208,
			B: 208,
			A: 0xff,
		}
	case theme.ColorNameHover:
		return color.NRGBA{
			R: 19,
			G: 202,
			B: 105,
			A: 0x99,
		}
	case theme.ColorNameMenuBackground:
		return color.NRGBA{
			R: 31,
			G: 33,
			B: 40,
			A: 0xee,
		}
	case theme.ColorNameInputBackground:
		return color.Alpha16{
			A: 0x0,
		}
	case theme.ColorNameSeparator:
		return color.NRGBA{
			R: 0x88,
			G: 0x88,
			B: 0x88,
			A: 0x35,
		}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{
			R: 0x88,
			G: 0x88,
			B: 0x88,
			A: 0xff,
		}
	case theme.ColorNamePressed:
		return color.NRGBA{
			R: 208,
			G: 208,
			B: 208,
			A: 0x19,
		}
	case theme.ColorNamePrimary:
		return color.NRGBA{
			R: 208,
			G: 208,
			B: 208,
			A: 0xff,
		}
	case theme.ColorNameScrollBar:
		return color.NRGBA{
			R: 19,
			G: 202,
			B: 105,
			A: 0x99,
		}
	case theme.ColorNameShadow:
		return color.Alpha16{
			0x19,
		}
	default:
		return theme.DefaultTheme().Color(c, v)
	}
}

func (eTheme) Font(s fyne.TextStyle) fyne.Resource {
	if s.Monospace {
		return resourceRegularTtf
		//return resourceGoNotoCurrentTtf
	}
	if s.Bold {
		if s.Italic {
			return resourceBoldItalicTtf
			//return resourceGoNotoCurrentTtf
		}
		return resourceBoldTtf
		//return resourceNotoSansBoldTtf
	}
	if s.Italic {
		return resourceItalicTtf
		//return resourceGoNotoCurrentTtf
	}
	return resourceRegularTtf
	//return resourceGoNotoCurrentTtf
}

func (eTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

func (eTheme) Size(s fyne.ThemeSizeName) float32 {
	switch s {
	case theme.SizeNameCaptionText:
		return 11
	case theme.SizeNameInlineIcon:
		return 20
	case theme.SizeNamePadding:
		return 4
	case theme.SizeNameScrollBar:
		return 16
	case theme.SizeNameScrollBarSmall:
		return 3
	case theme.SizeNameSeparatorThickness:
		return 1
	case theme.SizeNameText:
		return 15
	case theme.SizeNameInputBorder:
		return 2
	default:
		return theme.DefaultTheme().Size(s)
	}
}

type eTheme2 struct{}

func (eTheme2) Color(c fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	switch c {
	case theme.ColorNameBackground:
		return color.NRGBA{
			R: 21,
			G: 23,
			B: 30,
			A: 0xff,
		}
	case theme.ColorNameHyperlink:
		return color.NRGBA{
			R: 235,
			G: 235,
			B: 235,
			A: 0x99,
		}
	case theme.ColorNameButton:
		return color.NRGBA{
			R: 19,
			G: 202,
			B: 105,
			A: 0x75,
		}
	case theme.ColorNameDisabledButton:
		return color.NRGBA{
			R: 19,
			G: 202,
			B: 105,
			A: 0x22,
		}
	case theme.ColorNameDisabled:
		return color.NRGBA{
			R: 164,
			G: 164,
			B: 164,
			A: 0x42,
		}
	case theme.ColorNameError:
		return color.NRGBA{
			R: 0xf4,
			G: 0x43,
			B: 0x36,
			A: 0xff,
		}
	case theme.ColorNameFocus:
		return color.NRGBA{
			R: 19,
			G: 202,
			B: 105,
			A: 0x88,
		}
	case theme.ColorNameForeground:
		return color.NRGBA{
			R: 208,
			G: 208,
			B: 208,
			A: 0xff,
		}
	case theme.ColorNameHover:
		return color.NRGBA{
			R: 19,
			G: 202,
			B: 105,
			A: 0x99,
		}
	case theme.ColorNameMenuBackground:
		return color.NRGBA{
			R: 21,
			G: 23,
			B: 30,
			A: 0xee,
		}
	case theme.ColorNameInputBackground:
		return color.Alpha16{
			A: 0x0,
		}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{
			R: 0x88,
			G: 0x88,
			B: 0x88,
			A: 0xff,
		}
	case theme.ColorNamePressed:
		return color.NRGBA{
			R: 208,
			G: 208,
			B: 208,
			A: 0x19,
		}
	case theme.ColorNamePrimary:
		return color.NRGBA{
			R: 208,
			G: 208,
			B: 208,
			A: 0xff,
		}
	case theme.ColorNameScrollBar:
		return color.NRGBA{
			R: 19,
			G: 202,
			B: 105,
			A: 0x99,
		}
	case theme.ColorNameShadow:
		return color.Alpha16{
			0x19,
		}
	default:
		return theme.DefaultTheme().Color(c, v)
	}
}

func (eTheme2) Font(s fyne.TextStyle) fyne.Resource {
	if s.Monospace {
		//return resourceRegularTtf
		return resourceGoNotoCurrentTtf
	}
	if s.Bold {
		if s.Italic {
			//return resourceBoldItalicTtf
			return resourceGoNotoCurrentTtf
		}
		return resourceBoldTtf
		//return resourceNotoSansBoldTtf
	}
	if s.Italic {
		//return resourceItalicTtf
		return resourceGoNotoCurrentTtf
	}
	//return resourceRegularTtf
	return resourceGoNotoCurrentTtf
}

func (eTheme2) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

func (eTheme2) Size(s fyne.ThemeSizeName) float32 {
	switch s {
	case theme.SizeNameCaptionText:
		return 11
	case theme.SizeNameInlineIcon:
		return 20
	case theme.SizeNamePadding:
		return 4
	case theme.SizeNameScrollBar:
		return 16
	case theme.SizeNameScrollBarSmall:
		return 3
	case theme.SizeNameSeparatorThickness:
		return 1
	case theme.SizeNameText:
		return 15
	case theme.SizeNameInputBorder:
		return 2
	default:
		return theme.DefaultTheme().Size(s)
	}
}

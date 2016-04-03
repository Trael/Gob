package main

import (
	"errors"
	"fmt"
	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/event/touch"
	"golang.org/x/net/html"
	"image"
	"image/color"
	"image/draw"
	"os"
)

var (
	background = color.RGBA{0xE0, 0xE0, 0xE0, 0xFF}
	//	background = color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}

	NoStyles     = errors.New("No styles to apply")
	NotAnElement = errors.New("Not an element node")
)

type Viewport struct {
	// The size of the viewport
	Size size.Event

	// The whole, source image to be displayed in the viewport. It will be clipped
	// and displayed in the viewport according to the Size and Cursor
	Content *image.RGBA

	// The location of the image to be displayed into the viewpart.
	Cursor image.Point
}
type Page struct {
	//*html.Node
	Body *HTMLElement
}

func realWalkBody(n *HTMLElement, callback func(e *HTMLElement)) {
	if n == nil {
		return
	}
	if n.Type == html.ElementNode {
		callback(n)
	}
	for _, c := range n.Children {
		if val, ok := c.(*HTMLElement); ok {
			realWalkBody(val, callback)
		}
	}
}
func (p Page) WalkBody(callback func(*HTMLElement)) {
	if p.Body == nil {
		panic("Nothing to walk")
	}
	realWalkBody(p.Body, callback)
}

func paintWindow(s screen.Screen, w screen.Window, v *Viewport, page *Page, sty Stylesheet) {
	viewport := v.Size.Bounds()

	// Fill the window background with gray
	w.Fill(viewport, background, screen.Src)

	if v.Content != nil {
		b, err := s.NewBuffer(v.Size.Size())
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			return
		}
		defer b.Release()
		//fmt.Printf("%s", v.Size.Size())
		draw.Draw(b.RGBA(), viewport, v.Content, v.Cursor, draw.Src)
		//page.Body.Render(b.RGBA())
		w.Upload(image.Point{0, 0}, b, viewport)
	} else {
		fmt.Fprintf(os.Stderr, "No body to render!\n")
	}
	w.Publish()
}

func main() {
	driver.Main(func(s screen.Screen) {
		w, err := s.NewWindow(nil)
		if err != nil {
			panic(err)
		}
		defer w.Release()

		f, err := os.Open("test.html")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not open test.html\n")
			return
		}
		parsedhtml, sty := parseHTML(f)
		f.Close()
		parsedhtml.WalkBody(func(n *HTMLElement) {
			for _, rule := range sty {
				if rule.Matches(n) {
					n.AddStyle(rule)
				}
			}
		})

		var v Viewport
		v.Content = parsedhtml.Body.Render(v.Size.Size().X)
		for {
			switch e := w.NextEvent().(type) {
			case lifecycle.Event:
				if e.To == lifecycle.StageDead {
					return
				}
			case key.Event:
				switch e.Code {
				case key.CodeEscape:
					return
				case key.CodeDownArrow:
					if e.Direction == key.DirPress {
						scrollSize := v.Size.Size().Y / 2
						v.Cursor.Y += scrollSize
						if v.Cursor.Y > v.Content.Bounds().Max.Y {
							v.Cursor.Y = v.Content.Bounds().Max.Y - 10
						}
						paintWindow(s, w, &v, parsedhtml, sty)
					}
				case key.CodeUpArrow:
					if e.Direction == key.DirPress {
						scrollSize := v.Size.Size().Y / 2
						v.Cursor.Y -= scrollSize
						if v.Cursor.Y < 0 {
							v.Cursor.Y = 0
						}
						paintWindow(s, w, &v, parsedhtml, sty)
					}
				default:
					fmt.Printf("Unknown key: %s", e.Code)
				}
			case paint.Event:
				paintWindow(s, w, &v, parsedhtml, sty)
			case size.Event:
				v.Size = e
				v.Content = parsedhtml.Body.Render(e.Size().X)
			case touch.Event:
				fmt.Printf("Touch event!")
			default:
				//	fmt.Printf("%s\n", e)
			}
		}
	})
}
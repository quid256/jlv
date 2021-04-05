package main

import (
	"encoding/json"
	"strings"

	"github.com/gdamore/tcell"
)

type jsonView struct {
	// The location that the log will be displayed in (inclusive of endpoints)
	frame rect

	// The scrolling amount of the log
	ind int

	bufferLines []string
}

func (jv *jsonView) setEntry(entry logEntry) {
	b, err := json.MarshalIndent(entry.rest, "", "  ")
	if err != nil {
		b = []byte("[error] Failed to marshal json entry!")
	}

	jv.bufferLines = strings.Split(string(b), "\n")
	jv.ind = 0
}

func (jv *jsonView) scroll(screen tcell.Screen, amnt int) {
	jv.ind += amnt

	if jv.ind+jv.frame.height()-2 > len(jv.bufferLines) {
		jv.ind = len(jv.bufferLines) - jv.frame.height() + 2
	}

	if jv.ind < 0 {
		jv.ind = 0
	}

	jv.draw(screen)
}

func (jv *jsonView) resize(screen tcell.Screen, newFrame rect) {
	jv.frame = newFrame
	jv.draw(screen)
}

func (jv *jsonView) draw(screen tcell.Screen) {
	drawFrame(screen, jv.frame, "json")

	contentFrame := jv.frame.inset()
	for x := contentFrame.x0; x <= contentFrame.x1; x++ {
		for y := contentFrame.y0; y <= contentFrame.y1; y++ {
			screen.SetContent(x, y, ' ', nil, tcell.StyleDefault)
		}
	}

	for i := jv.ind; i < len(jv.bufferLines) && (i-jv.ind) < contentFrame.height(); i++ {
		if contentFrame.y0+(i-jv.ind) > contentFrame.y1 {
			break
		}

		for j, char := range jv.bufferLines[i] {
			if (contentFrame.x0 + j) > contentFrame.x1 {
				break
			}
			screen.SetContent(contentFrame.x0+j, contentFrame.y0+(i-jv.ind), char, nil, tcell.StyleDefault)
		}
	}
}

package main

import (
	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/gdamore/tcell"
)

type queryView struct {
	// The location that the query will be displayed in (inclusive of endpoints)
	frame rect

	query string

	cursor       int
	invalidQuery bool
	validQuery   bool
}

func (qv *queryView) typeRune(screen tcell.Screen, r rune) {
	qv.query = qv.query[:qv.cursor] + string(r) + qv.query[qv.cursor:]
	qv.validQuery = false
	qv.invalidQuery = false

	qv.draw(screen)
	qv.navigate(screen, +1)
}

func (qv *queryView) deleteRune(screen tcell.Screen) {
	qv.validQuery = false
	qv.invalidQuery = false

	if qv.cursor > 0 {
		qv.query = qv.query[:qv.cursor-1] + qv.query[qv.cursor:]

		qv.navigate(screen, -1)
		qv.draw(screen)
	}
}

func (qv *queryView) clear(screen tcell.Screen) {
	qv.query = ""
	qv.validQuery = false
	qv.invalidQuery = false
	qv.cursor = 0

	qv.draw(screen)
	qv.showCursor(screen)
}

func (qv *queryView) navigate(screen tcell.Screen, amnt int) {
	qv.cursor += amnt
	if qv.cursor < 0 {
		qv.cursor = 0
	}
	if qv.cursor > len(qv.query) {
		qv.cursor = len(qv.query)
	}

	qv.showCursor(screen)
}

func (qv *queryView) submit(screen tcell.Screen) (prog *vm.Program) {
	prog, err := expr.Compile(qv.query, expr.AsBool())

	if qv.query == "" {
		qv.invalidQuery = false
		qv.validQuery = false
	} else if err == nil {
		qv.invalidQuery = false
		qv.validQuery = true
	} else {
		qv.invalidQuery = true
		qv.validQuery = false
	}

	qv.draw(screen)
	screen.HideCursor()

	return
}

func (qv *queryView) leave(screen tcell.Screen) {
	screen.HideCursor()
}

func (qv *queryView) start(screen tcell.Screen) {
	qv.cursor = len(qv.query)
	qv.showCursor(screen)
}

func (qv *queryView) showCursor(screen tcell.Screen) {
	screen.ShowCursor(qv.frame.x0+2+qv.cursor, qv.frame.y0)
}

func (qv *queryView) resize(screen tcell.Screen, newFrame rect) {
	qv.frame = newFrame
	qv.draw(screen)
}

func (qv *queryView) draw(screen tcell.Screen) {
	width, _ := qv.frame.dims()

	fullQueryString := []rune("? " + qv.query)

	var j int

	for j = 0; j < len(fullQueryString) && j < width; j++ {
		style := tcell.StyleDefault
		r := fullQueryString[j]

		if j == 0 {
			if qv.invalidQuery {
				style = style.Bold(true).Foreground(tcell.GetColor("#BF616A"))
				r = 'X'
			} else if qv.validQuery {
				style = style.Bold(true).Foreground(tcell.GetColor("#A3BE8C"))
				r = '✓'
			} else {
				style = style.Bold(true).Foreground(tcell.GetColor("#8FBCBB"))
				r = '?'
			}
		}

		screen.SetContent(qv.frame.x0+j, qv.frame.y0, r, nil, style)
	}

	for j < width {
		screen.SetContent(qv.frame.x0+j, qv.frame.y0, ' ', nil, tcell.StyleDefault)
		j++
	}
}

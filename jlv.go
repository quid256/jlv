package main

import (
	"fmt"
	"math"
	"os"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/gdamore/tcell"
)

var logRatio = 0.5

const (
	runeTL = '╭'
	runeTR = '╮'
	runeBL = '╰'
	runeBR = '╯'
	runeH  = '─'
	runeV  = '│'
)

type logViewer struct {
	screen tcell.Screen

	file []logEntry

	lv logView
	jv jsonView
	qv queryView

	editMode bool
}

func (l *logViewer) init(filename string) {
	l.file = parseJSONLogFile(filename)

	l.lv = logView{
		file:         l.file,
		starredLines: make(map[int]struct{}),
	}
	l.jv = jsonView{file: l.file}
	l.qv = queryView{}
}

func (l *logViewer) handleKey(key *tcell.EventKey) (shouldQuit bool) {
	if l.editMode {
		switch key.Key() {
		case tcell.KeyRune:
			l.qv.typeRune(l.screen, key.Rune())
		case tcell.KeyLeft:
			l.qv.navigate(l.screen, -1)
		case tcell.KeyRight:
			l.qv.navigate(l.screen, +1)
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			l.qv.deleteRune(l.screen)
		case tcell.KeyCtrlU:
			l.qv.clear(l.screen)
		case tcell.KeyEnter:
			l.editMode = false
			prog := l.qv.submit(l.screen)
			l.executeProgram(prog)
		case tcell.KeyEsc:
			l.editMode = false
			l.qv.leave(l.screen)
		}
	} else {
		switch key.Key() {
		case tcell.KeyRune:
			r := key.Rune()
			switch r {
			case 'j':
				l.lv.scroll(l.screen, +1)
			case 'k':
				l.lv.scroll(l.screen, -1)
			case 'J':
				l.lv.scroll(l.screen, l.lv.frame.inset().height())
			case 'K':
				l.lv.scroll(l.screen, -l.lv.frame.inset().height())
			case 'G':
				l.lv.scroll(l.screen, len(l.file))
			case 'g':
				l.lv.scroll(l.screen, -len(l.file))
			case 'n':
				l.lv.nextMatch(l.screen)
			case 'N':
				l.lv.prevMatch(l.screen)
			case 'h':
				logRatio -= 0.05
				if logRatio < 0.2 {
					logRatio = 0.2
				}
				l.resize()
			case 'l':
				logRatio += 0.05
				if logRatio > 0.8 {
					logRatio = 0.8
				}
				l.resize()
			case '/', ':':
				l.editMode = true
				l.qv.start(l.screen)
			case 'q':
				return true
			}

		case tcell.KeyDown:
			l.lv.scroll(l.screen, +1)
		case tcell.KeyUp:
			l.lv.scroll(l.screen, -1)
		}

		l.jv.setEntry(l.screen, l.lv.getEntry())
	}

	l.screen.Show()

	return false
}

func (l *logViewer) executeProgram(prog *vm.Program) {
	defer l.lv.draw(l.screen)

	l.lv.starredLines = make(map[int]struct{})

	if prog == nil {
		return
	}

	for i, line := range l.file {
		result, err := expr.Run(prog, line.rest)
		if err == nil && result.(bool) {
			l.lv.starredLines[i] = struct{}{}
		}
	}
}

func (l *logViewer) resize() {
	l.screen.Clear()

	width, height := l.screen.Size()

	dividerX := int(math.Ceil(logRatio*float64(width))) - 1

	l.lv.resize(l.screen, rect{0, 0, dividerX, height - 2})
	l.jv.resize(l.screen, rect{dividerX + 1, 0, width - 1, height - 2})
	l.qv.resize(l.screen, rect{0, height - 1, width, height - 1})

	l.screen.Sync()
}

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Must provide exactly one argument (the log file), provided %d", len(os.Args))
		return
	}
	if _, err := os.Stat(os.Args[1]); err != nil {
		fmt.Printf("Unable to locate file %s", os.Args[1])
		return
	}

	screen, err := tcell.NewScreen()
	checkErr(err, "Unable to construct screen: %v")

	err = screen.Init()
	checkErr(err, "Unable to initialize screen: %v")
	defer screen.Fini()

	viewer := &logViewer{screen: screen}
	viewer.init(os.Args[1])

	for {
		ev := screen.PollEvent()
		if ev == nil {
			// Screen has been finalized, need to exit event loop
			break
		}

		switch event := ev.(type) {
		case *tcell.EventError:
			checkErr(event, "t-cell event loop error: %v")
		case *tcell.EventInterrupt:
			screen.Sync()
		case *tcell.EventKey:
			if event.Key() == tcell.KeyCtrlC {
				return
			}

			if shouldQuit := viewer.handleKey(event); shouldQuit {
				return
			}
		case *tcell.EventResize:
			viewer.resize()
		}
	}
}

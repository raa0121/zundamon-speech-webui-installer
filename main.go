package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type MyMainWindow struct {
	*walk.MainWindow
	cfg *config

	gitPathLE    *walk.LineEdit
	pythonPathLE *walk.LineEdit
	logTE        *walk.TextEdit
	checkButton  *walk.PushButton
	setupButton  *walk.PushButton
	startButton  *walk.PushButton
	stopButton   *walk.PushButton
}

var (
	configDir string
	repoDir   string

	ctx = context.Background()

	httpClient = &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   time.Second,
			ResponseHeaderTimeout: time.Second,
			IdleConnTimeout:       time.Second,
		},
	}
)

func (mw *MyMainWindow) log(mes string) {
	slog.Info(mes)
	mw.Synchronize(func() {
		mw.logTE.AppendText(mes + "\r\n")
	})
}

func main() {
	var cfg config
	err := cfg.load()
	if err != nil {
		log.Fatal(err)
	}
	repoDir = filepath.Join(configDir, "zundamon")
	mw := new(MyMainWindow)
	mw.cfg = &cfg
	go func() {
		time.Sleep(500 * time.Millisecond)
		mw.Synchronize(func() {
			mw.setupLogger("app")
			mw.logTE.SetMaxLength(300000000)
		})
	}()
	MainWindow{
		AssignTo: &mw.MainWindow,
		Title:    "zundamon-speech-webui installer",
		MinSize:  Size{Width: 300, Height: 300},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 3},
				Children: []Widget{
					Label{
						Text: "git.exe のパス",
					},
					LineEdit{
						Text:     mw.cfg.GitPath,
						Enabled:  false,
						AssignTo: &mw.gitPathLE,
					},
					PushButton{
						Text:      "Open",
						OnClicked: mw.openGitPathAction,
					},
					Label{
						Text: "python.exe のパス",
					},
					LineEdit{
						Text:     mw.cfg.PythonPath,
						Enabled:  false,
						AssignTo: &mw.pythonPathLE,
					},
					PushButton{
						Text:      "Open",
						OnClicked: mw.openPythonPathAction,
					},
				},
			},
			Composite{
				Layout: Grid{Columns: 4},
				Children: []Widget{
					PushButton{
						Text:      "Check",
						OnClicked: mw.checkAction,
						AssignTo:  &mw.checkButton,
						Enabled:   !mw.cfg.IsCheck,
					},
					PushButton{
						Text:      "SetUp",
						OnClicked: mw.setupAction,
						AssignTo:  &mw.setupButton,
					},
					PushButton{
						Text:      "Start",
						OnClicked: mw.startAction,
						AssignTo:  &mw.startButton,
						Enabled:   !isStart,
					},
					PushButton{
						Text:      "Stop",
						OnClicked: mw.stopAction,
						AssignTo:  &mw.stopButton,
						Enabled:   isStart,
					},
				},
			},
			Composite{
				Layout: Grid{Columns: 1},
				Children: []Widget{
					Label{
						Text: "ログ",
					},
					TextEdit{
						AssignTo: &mw.logTE,
						ReadOnly: true,
						VScroll:  true,
					},
				},
			},
		},
	}.Run()
}

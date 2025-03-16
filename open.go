package main

import (
	"log"

	"github.com/lxn/walk"
)

func (mw *MyMainWindow) openGitPathAction() {
	if err := mw.openGitPath(); err != nil {
		log.Print(err)
	}
}

func (mw *MyMainWindow) openGitPath() error {
	dlg := new(walk.FileDialog)
	dlg.FilePath = mw.cfg.GitPath
	dlg.Filter = "git.exe|git.exe"
	dlg.Title = "git.exe を開く"
	if ok, err := dlg.ShowOpen(mw); err != nil {
		return err
	} else if !ok {
		return nil
	}
	mw.cfg.GitPath = dlg.FilePath
	err := mw.gitPathLE.SetText(dlg.FilePath)
	if err != nil {
		return err
	}
	return nil
}

func (mw *MyMainWindow) openPythonPathAction() {
	if err := mw.openPythonPath(); err != nil {
		log.Print(err)
	}
}

func (mw *MyMainWindow) openPythonPath() error {
	dlg := new(walk.FileDialog)
	dlg.FilePath = mw.cfg.PythonPath
	dlg.Filter = "python.exe|python.exe;python3.exe"
	dlg.Title = "python.exe を開く"
	if ok, err := dlg.ShowOpen(mw); err != nil {
		return err
	} else if !ok {
		return nil
	}
	mw.cfg.PythonPath = dlg.FilePath
	err := mw.pythonPathLE.SetText(dlg.FilePath)
	if err != nil {
		return err
	}
	return nil
}

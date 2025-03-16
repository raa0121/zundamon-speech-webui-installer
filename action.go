package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	mainProcess *exec.Cmd
	isStart     bool
)

func (mw *MyMainWindow) checkAction() {
	if err := mw.check(); err != nil {
		slog.Error(err.Error())
	}
}

func (mw *MyMainWindow) check() error {
	err := mw.checkGitLFS()
	if err != nil {
		return err
	}
	err = mw.checkPythonVersion()
	if err != nil {
		return err
	}
	err = mw.checkCmake()
	if err != nil {
		return err
	}
	mw.cfg.IsCheck = true
	mw.cfg.save()
	mw.checkButton.SetEnabled(false)
	return nil
}

func (mw *MyMainWindow) checkGitLFS() error {
	mw.log("git-lfs の存在を確認中...")

	_, err := exec.LookPath("git-lfs")
	if err == nil {
		mw.log("git-lfs は PATH に存在します。")
		return nil
	}

	dir := filepath.Dir(mw.cfg.GitPath)
	lfsInGitDir := filepath.Join(dir, "git-lfs")
	if _, err := os.Stat(lfsInGitDir); err == nil {
		mw.log("git-lfs は Git のディレクトリ内に存在します。")
		return nil
	}

	mw.log("エラー: git-lfs が見つかりません。")
	return fmt.Errorf("git-lfs is not installed or not found in the same directory as Git")
}

func (mw *MyMainWindow) checkPythonVersion() error {
	mw.log("Python のバージョンを確認中...")

	cmd := exec.Command(mw.cfg.PythonPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		mw.log("エラー: Python のバージョン確認に失敗しました。")
		return fmt.Errorf("failed to execute Python: %v", err)
	}

	versionStr := strings.TrimPrefix(strings.TrimSpace(string(output)), "Python ")
	mw.log("検出された Python バージョン: " + versionStr)

	versionParts := strings.Split(versionStr, ".")
	if len(versionParts) < 2 {
		mw.log("エラー: Python のバージョンフォーマットが不正です。")
		return fmt.Errorf("invalid Python version format: %s", versionStr)
	}

	major, err := strconv.Atoi(versionParts[0])
	if err != nil {
		mw.log("エラー: Python のメジャーバージョンが不正です。")
		return fmt.Errorf("invalid Python major version: %s", versionParts[0])
	}
	minor, err := strconv.Atoi(versionParts[1])
	if err != nil {
		mw.log("エラー: Python のマイナーバージョンが不正です。")
		return fmt.Errorf("invalid Python minor version: %s", versionParts[1])
	}

	if major < 3 || (major == 3 && minor < 9) {
		mw.log(fmt.Sprintf("エラー: Python 3.9 以上が必要ですが、%s が検出されました。", versionStr))
		return fmt.Errorf("Python 3.9 or higher is required, but found %s", versionStr)
	}

	if major > 3 || (major == 3 && minor >= 12) {
		mw.log("エラー: Python 3.12 以上はサポートされていません")
		return fmt.Errorf("Python 3.12 以上はサポートされていません")
	}

	mw.log("Python のバージョンチェック完了。問題ありません。")
	return nil
}

func (mw *MyMainWindow) checkCmake() error {
	_, err := exec.LookPath("cmake")
	if err != nil {
		slog.Error("CMake が PATH に見つかりません")
		mw.logToUI("CMake が PATH に見つかりません")
		return err
	}
	mw.log("CMake がインストールされています")
	return nil
}

func (mw *MyMainWindow) setupAction() {
	if err := mw.setup(); err != nil {
		slog.Error(err.Error())
	}
}

func (mw *MyMainWindow) setup() error {
	modelsDir := filepath.Join(configDir, "models")
	fineTunedModelsPath := filepath.Join(configDir, "zundamon_GPT-SoVITS")
	go func() {
		var wg sync.WaitGroup
		mw.log("セットアップを開始します...")

		steps := []struct {
			message string
			action  func() error
		}{
			{
				"リポジトリを更新中...",
				func() error { return mw.updateOrCloneRepo(repoDir, "https://github.com/zunzun999/zundamon-speech-webui") },
			},
			{"Python環境をセットアップ中...", mw.setupPythonEnv},
			{
				"モデルを取得中...",
				func() error { return mw.updateOrCloneRepo(modelsDir, "https://huggingface.co/lj1995/GPT-SoVITS") },
			},
			{"プリトレーニング済みモデルをコピー中...", mw.copyPretrainedModels},
			{"G2PWモデルをダウンロード中...", mw.downloadAndExtractG2PWModel},
			{
				"ファインチューニング済みモデルを取得中...",
				func() error { return mw.updateOrCloneRepo(fineTunedModelsPath, "https://huggingface.co/zunzunpj/zundamon_GPT-SoVITS") },
			},
			{"ファインチューニング済みモデルをコピー中...", mw.copyFineTunedModels},
		}

		for _, step := range steps {
			wg.Add(1)
			go func(stepMessage string, stepAction func() error) {
				defer wg.Done()

				mw.log(stepMessage)

				if err := stepAction(); err != nil {
					mw.log("エラー: " + err.Error())
				}
			}(step.message, step.action)

			// 各ステップが終わるまで待機
			wg.Wait()
		}

		mw.log("セットアップ完了")
	}()
	return nil
}

func (mw *MyMainWindow) setupLogger(name string) (*os.File, error) {
	filePath := filepath.Join(configDir, name+time.Now().Format("20060102_15405")+".log")
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		slog.Error("ログファイルを開けません", "error", err)
		return nil, err
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(file, nil)))
	return file, nil
}

func (mw *MyMainWindow) updateOrCloneRepo(path, repoURL string) error {
	if _, err := os.Stat(path); err == nil {
		if _, err := mw.execCommand(path, mw.cfg.GitPath, "pull", "--rebase", "origin", "main"); err != nil {
			return err
		}
		_, err = mw.execCommand(path, mw.cfg.GitPath, "lfs", "pull")
		return err
	}
	_, err := mw.execCommand(configDir, mw.cfg.GitPath, "clone", repoURL, path)
	return err

}

func (mw *MyMainWindow) setupPythonEnv() error {
	venvPath := filepath.Join(repoDir, ".venv")
	if _, err := os.Stat(venvPath); err != nil {
		_, err := mw.execCommand(repoDir, mw.cfg.PythonPath, "-m", "venv", venvPath)
		if err != nil {
			return err
		}
	}
	venvPython := filepath.Join(venvPath, "Scripts", "python.exe")
	return mw.installPythonDependencies(venvPython)
}

func (mw *MyMainWindow) installPythonDependencies(pythonPath string) error {
	_, err := mw.execCommand(repoDir, pythonPath, "-m", "pip", "install", "-r", "requirements.txt")
	if err != nil {
		return err
	}
	_, err = mw.execCommand(
		repoDir,
		pythonPath,
		"-m",
		"pip",
		"install",
		"torch==2.1.2",
		"torchvision==0.16.2",
		"torchaudio==2.1.2",
		"--index-url",
		"https://download.pytorch.org/whl/cu121",
	)
	if err != nil {
		return err
	}
	return nil
}

func (mw *MyMainWindow) copyPretrainedModels() error {
	mw.log("Copy pretrained_models...")
	modelsDir := filepath.Join(configDir, "models")
	destPath := filepath.Join(repoDir, "GPT-SoVITS", "GPT_SoVITS", "pretrained_models")

	// 既存のディレクトリを削除
	if _, err := os.Stat(destPath); err == nil {
		if err := os.RemoveAll(destPath); err != nil {
			return fmt.Errorf("failed to remove existing pretrained_models: %w", err)
		}
	}

	// コピー実行
	return os.CopyFS(destPath, os.DirFS(modelsDir))
}

func (mw *MyMainWindow) downloadAndExtractG2PWModel() error {
	zipPath := filepath.Join(configDir, "G2PWModel_1.1.zip")
	modelDir := filepath.Join(configDir, "G2PWModel_1.1")
	// ファイルが存在しない場合はダウンロード
	if _, err := os.Stat(zipPath); err != nil {
		mw.log("Download G2PWModel...")
		if err := DownloadFile(zipPath, "https://paddlespeech.bj.bcebos.com/Parakeet/released_models/g2p/G2PWModel_1.1.zip"); err != nil {
			return err
		}
	}
	mw.log("Unzip G2PWModel...")
	if err := os.RemoveAll(modelDir); err != nil {
		return fmt.Errorf("failed to remove existing model directory: %w", err)
	}
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Mode().IsDir() {
			continue
		}
		if err := saveUnZipFile(configDir, *f); err != nil {
			return err
		}
	}

	destPath := filepath.Join(repoDir, "GPT-SoVITS", "GPT_SoVITS", "text", "G2PWModel")
	// 既存のディレクトリを削除
	if _, err := os.Stat(destPath); err == nil {
		if err := os.RemoveAll(destPath); err != nil {
			return fmt.Errorf("failed to remove existing G2PWModel: %w", err)
		}
	}
	return os.CopyFS(destPath, os.DirFS(modelDir))
}

func (mw *MyMainWindow) copyFineTunedModels() error {
	mw.log("Copy fine-tune models...")
	fineTunedModelsPath := filepath.Join(configDir, "zundamon_GPT-SoVITS")
	srcGPT := filepath.Join(fineTunedModelsPath, "GPT_weights_v2")
	destGPT := filepath.Join(repoDir, "GPT-SoVITS", "GPT_weights_v2")
	srcSoVITS := filepath.Join(fineTunedModelsPath, "SoVITS_weights_v2")
	destSoVITS := filepath.Join(repoDir, "GPT-SoVITS", "SoVITS_weights_v2")

	// 既存のディレクトリを削除
	if err := os.RemoveAll(destGPT); err != nil {
		return fmt.Errorf("failed to remove existing GPT_weights_v2: %w", err)
	}
	if err := os.RemoveAll(destSoVITS); err != nil {
		return fmt.Errorf("failed to remove existing SoVITS_weights_v2: %w", err)
	}

	// コピー実行
	if err := os.CopyFS(destGPT, os.DirFS(srcGPT)); err != nil {
		return err
	}
	return os.CopyFS(destSoVITS, os.DirFS(srcSoVITS))
}

func saveUnZipFile(destDir string, f zip.File) error {
	// 展開先のパスを設定する
	destPath := filepath.Join(destDir, f.Name)
	// 子孫ディレクトリがあれば作成する
	if err := os.MkdirAll(filepath.Dir(destPath), f.Mode()); err != nil {
		return err
	}
	// Zipファイルを開く
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	// 展開先ファイルを作成する
	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()
	// 展開先ファイルに書き込む
	if _, err := io.Copy(destFile, rc); err != nil {
		return err
	}

	return nil
}

func (mw *MyMainWindow) execCommand(dir string, name string, arg ...string) (*exec.Cmd, error) {
	cmd, err := mw.ShellExecWithArgs(ctx, name, arg, dir, 15*time.Minute)
	if err != nil {
		return nil, err
	}
	return cmd, nil
}

func (mw *MyMainWindow) startAction() {
	if err := mw.start(); err != nil {
		slog.Error(err.Error())
	}
}

func (mw *MyMainWindow) start() error {
	var err error
	mw.startButton.SetEnabled(false)
	mw.stopButton.SetEnabled(true)
	isStart = true
	go func () {
		venvStreamlitPath := filepath.Join(repoDir, ".venv", "Scripts", "streamlit.exe")
		mainProcess, err = mw.execCommand(filepath.Join(repoDir, "GPT-SoVITS"), venvStreamlitPath, "run", "zundamon_webui.py")
		if err != nil {
			return
		}
	}()
	return nil
}

func (mw *MyMainWindow) stopAction() {
	if err := mw.stop(); err != nil {
		slog.Error(err.Error())
	}
}

func (mw *MyMainWindow) stop() error {
	if mainProcess == nil {
		mw.log("プロセス情報が取得できませんでした")
		return nil
	}
	if err := mw.killProcessGroup(mainProcess); err != nil {
		return fmt.Errorf("failed to kill process: %v", err)
	}
	mw.startButton.SetEnabled(true)
	mw.stopButton.SetEnabled(false)
	mw.log("プロセスを終了しました")
	return nil
}

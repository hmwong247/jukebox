package ytdlp

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/url"
	"os/exec"
)

const (
	CMD_YTDLP = "./third_party/yt-dlp/yt-dlp_linux"
	CMD_JQ    = "jq"
)

var (
	YTDLP_OPT_STDOUT = []string{"-x", "-f", "bestaudio", "-o", "-"}

	// YTDLP_OPT_THUMBNAIL = []string{"--skip-download", "--write-thumbnail"}
	YTDLP_OPT_INFOJSON = []string{"--skip-download", "-j"}
	JQ_OPT             = []string{"{fulltitle: .fulltitle, uploader: .uploader, thumbnail: .thumbnail, duration: .duration}"}
)

type InfoJson struct {
	ID        int
	FullTitle string
	Uploader  string
	Thumbnail string
	Duration  int
}

func DownloadAudio(rawURL string) ([]byte, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		slog.Info("url parse failed", "err", err, "url", parsedURL)
		return []byte{}, err
	}

	opt := append(YTDLP_OPT_STDOUT, parsedURL.String())
	cmd := exec.Command(CMD_YTDLP, opt...)
	stdout, err := cmd.StdoutPipe()

	slog.Debug("final cmd", "cmd", cmd)

	if errors.Is(cmd.Err, exec.ErrDot) {
		cmd.Err = nil
	}
	var exitStatus int
	err = cmd.Start()
	if err != nil {
		slog.Info("cmd start err", "err", err)
		return []byte{}, err
	}

	audioBytes, err := io.ReadAll(stdout)
	if err != nil {
		slog.Info("io read err", "err", err)
		return []byte{}, err
	}

	if err := cmd.Wait(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			slog.Info("cmd wait err", "exitError", exitError)
			return []byte{}, err
		}
	}

	slog.Debug("exited", "exitStatus", exitStatus)

	return audioBytes, nil
}

func DownloadThumbnail(rawURL string) ([]byte, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		slog.Info("url parse failed", "err", err, "url", parsedURL)
		return []byte{}, err
	}

	return []byte{}, nil
}

func DownloadInfoJson(rawURL string) (InfoJson, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		slog.Info("url parse failed", "err", err, "url", parsedURL)
		return InfoJson{}, err
	}

	opt := append(YTDLP_OPT_INFOJSON, parsedURL.String())
	cmd1 := exec.Command(CMD_YTDLP, opt...)
	cmd2 := exec.Command(CMD_JQ, JQ_OPT...)
	slog.Debug("final cmd", "cmd1", cmd1, "cmd2", cmd2)

	// command piping
	cmd2.Stdin, err = cmd1.StdoutPipe()
	if err != nil {
		slog.Error("cmd1.Stdoutpipe err", "err", err)
		return InfoJson{}, err
	}
	finalStdout, err := cmd2.StdoutPipe()
	if err != nil {
		slog.Error("cmd2.Stdoutpipe err", "err", err)
		return InfoJson{}, err
	}

	// start execute cmd
	err = cmd1.Start()
	if err != nil {
		slog.Info("cmd1 start err", "err", err)
		return InfoJson{}, err
	}
	err = cmd2.Start()
	if err != nil {
		slog.Info("cmd2 start err", "err", err)
		return InfoJson{}, err
	}
	if err := cmd1.Wait(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			slog.Info("cmd1 wait err", "exitError", exitError)
			return InfoJson{}, err
		}
	}

	jsonBytes, err := io.ReadAll(finalStdout)
	if err != nil {
		slog.Error("finalStdout io error", "err", err)
		return InfoJson{}, err
	}

	if err := cmd2.Wait(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			slog.Info("cmd2 wait err", "exitError", exitError)
			return InfoJson{}, err
		}
	}

	slog.Debug("cmd exited")

	infoJson := InfoJson{}
	if err := json.Unmarshal(jsonBytes, &infoJson); err != nil {
		slog.Debug("jsontest error", "err", err)
		return InfoJson{}, err
	}
	slog.Debug("infoJson", "json", infoJson)

	return infoJson, nil
	// return []byte("ok"), nil
}

package ytdlp

import (
	"errors"
	"io"
	"log/slog"
	"net/url"
	"os/exec"
)

const (
	CMD_PREFIX = "./third_party/yt-dlp/yt-dlp_linux"
)

var (
	YTDLP_OPT_STDOUT = []string{"-x", "-f", "bestaudio", "-o", "-"}
)

func CmdStart(rawURL string) ([]byte, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		slog.Info("url parse failed", "err", err, "url", parsedURL)
		return []byte{}, err
	}

	opt := append(YTDLP_OPT_STDOUT, parsedURL.String())
	cmd := exec.Command(CMD_PREFIX, opt...)
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

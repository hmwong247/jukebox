package ytdlp

import (
	"encoding/json"
	"errors"
	"fmt"
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
	FullTitle string
	Uploader  string
	Thumbnail string
	Duration  int
}

func DownloadAudio(rawURL string) ([]byte, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		errStr := fmt.Sprintf("url parse failed, err: %v, url: %v", err, parsedURL)
		newErr := errors.New(errStr)
		return []byte{}, newErr
	}

	opt := append(YTDLP_OPT_STDOUT, parsedURL.String())
	cmd := exec.Command(CMD_YTDLP, opt...)
	stdout, err := cmd.StdoutPipe()

	slog.Debug("final cmd", "cmd", cmd)

	if errors.Is(cmd.Err, exec.ErrDot) {
		cmd.Err = nil
	}
	err = cmd.Start()
	if err != nil {
		errStr := fmt.Sprintf("cmd start error: %v", err)
		newErr := errors.New(errStr)
		return []byte{}, newErr
	}

	audioBytes, err := io.ReadAll(stdout)
	if err != nil {
		errStr := fmt.Sprintf("io read error: %v", err)
		newErr := errors.New(errStr)
		return []byte{}, newErr
	}

	if err := cmd.Wait(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			errStr := fmt.Sprintf("cmd wait error: %v", exitError)
			newErr := errors.New(errStr)
			return []byte{}, newErr
		}
	}

	return audioBytes, nil
}

func DownloadThumbnail(rawURL string) ([]byte, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		errStr := fmt.Sprintf("url parse failed, err: %v, url: %v", err, parsedURL)
		newErr := errors.New(errStr)
		return []byte{}, newErr
	}

	return []byte{}, nil
}

func DownloadInfoJson(rawURL string) (InfoJson, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		errStr := fmt.Sprintf("url parse failed, err: %v, url: %v", err, parsedURL)
		newErr := errors.New(errStr)
		return InfoJson{}, newErr
	}

	opt := append(YTDLP_OPT_INFOJSON, parsedURL.String())
	cmd1 := exec.Command(CMD_YTDLP, opt...)
	cmd2 := exec.Command(CMD_JQ, JQ_OPT...)
	slog.Debug("final cmd", "cmd1", cmd1, "cmd2", cmd2)

	// command piping
	cmd2.Stdin, err = cmd1.StdoutPipe()
	if err != nil {
		errStr := fmt.Sprintf("cmd1.Stdoutpipe error, err: %v", err)
		newErr := errors.New(errStr)
		return InfoJson{}, newErr
	}
	finalStdout, err := cmd2.StdoutPipe()
	if err != nil {
		errStr := fmt.Sprintf("cmd2.Stdoutpipe error, err: %v", err)
		newErr := errors.New(errStr)
		return InfoJson{}, newErr
	}

	// start execute cmd
	err = cmd1.Start()
	if err != nil {
		errStr := fmt.Sprintf("cmd1 start error, err: %v", err)
		newErr := errors.New(errStr)
		return InfoJson{}, newErr
	}
	err = cmd2.Start()
	if err != nil {
		errStr := fmt.Sprintf("cmd2 start error, err: %v", err)
		newErr := errors.New(errStr)
		return InfoJson{}, newErr
	}
	if err := cmd1.Wait(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			cmd2.Wait()
			errStr := fmt.Sprintf("cmd1 wait error, exitError: %v", exitError)
			newErr := errors.New(errStr)
			return InfoJson{}, newErr
		}
	}

	jsonBytes, err := io.ReadAll(finalStdout)
	if err != nil {
		errStr := fmt.Sprintf("finalStdout io error, err: %v", err)
		newErr := errors.New(errStr)
		return InfoJson{}, newErr
	}

	if err := cmd2.Wait(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			errStr := fmt.Sprintf("cmd2 wait error, exitError: %v", exitError)
			newErr := errors.New(errStr)
			return InfoJson{}, newErr
		}
	}

	slog.Debug("cmd exited")

	infoJson := InfoJson{}
	if err := json.Unmarshal(jsonBytes, &infoJson); err != nil {
		errStr := fmt.Sprintf("json unmarshal error, err: %v", err)
		newErr := errors.New(errStr)
		return InfoJson{}, newErr
	}
	// slog.Debug("infoJson", "json", infoJson)

	return infoJson, nil
}

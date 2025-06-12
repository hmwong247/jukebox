package ytdlp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/url"
	"os"
)

var (
	YTDLPY_SOCKET_PATH = func() string {
		path := os.Getenv("YTDLPY_SOCKET_PATH")
		if path == "" {
			slog.Error("YTDLPY_SOCKET_PATH not found")
			return ""

		}
		return path
	}()
)

type InfoJson struct {
	FullTitle string
	Uploader  string
	Thumbnail string
	Duration  int
}

/*
	Downloading with embedded YTDLP in python
*/

func connectUDS(ctx context.Context, endpoint string) (*net.UnixConn, error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		errf := fmt.Errorf("[UDS] recieved ctx with no specified timeout")
		return nil, errf
	}
	unixSocketAddr, err := net.ResolveUnixAddr("unix", endpoint)
	if err != nil {
		errf := fmt.Errorf("[UDS] address resolve error, err:%v", err)
		return &net.UnixConn{}, errf
	}
	conn, err := net.DialUnix("unix", nil, unixSocketAddr)
	if err != nil {
		errf := fmt.Errorf("[USD] connection error, err: %v", err)
		return &net.UnixConn{}, errf
	}
	// enable socket timeout
	conn.SetDeadline(deadline)

	return conn, nil
}

type RPCInfoJsonRequest struct {
	Type string
	URL  string
}

func DownloadInfoJson(ctx context.Context, rawURL string) (InfoJson, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		errf := fmt.Errorf("url parse failed, err: %v, url: %v", err, parsedURL)
		return InfoJson{}, errf
	}

	conn, err := connectUDS(ctx, YTDLPY_SOCKET_PATH)
	slog.Debug("connectUDS done")
	if err != nil {
		return InfoJson{}, err
	}
	defer conn.Close()

	request := RPCInfoJsonRequest{
		Type: "json",
		URL:  rawURL,
	}
	requestJson, err := json.Marshal(request)
	if err != nil {
		errf := fmt.Errorf("requestJson parse error, err: %v, url: %v", err, parsedURL)
		return InfoJson{}, errf
	}
	conn.Write(requestJson)
	conn.CloseWrite()

	jsonBytes, err := io.ReadAll(conn)
	if err != nil {
		errf := fmt.Errorf("[UDS] read error, err:%v", err)
		return InfoJson{}, errf
	}
	// slog.Debug("[UDS] recv: ", "jsonBytes", jsonBytes)

	infoJson := InfoJson{}
	if err := json.Unmarshal(jsonBytes, &infoJson); err != nil {
		errf := fmt.Errorf("json unmarshal error, err: %v", err)
		return InfoJson{}, errf
	}
	// slog.Debug("infoJson", "json", infoJson)

	return infoJson, nil
}

func DownloadAudio(ctx context.Context, rawURL string) ([]byte, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		errf := fmt.Errorf("url parse failed, err: %v, url: %v", err, parsedURL)
		return []byte{}, errf
	}

	conn, err := connectUDS(ctx, YTDLPY_SOCKET_PATH)
	if err != nil {
		return []byte{}, err
	}
	defer conn.Close()

	request := RPCInfoJsonRequest{
		Type: "audio",
		URL:  rawURL,
	}
	requestJson, err := json.Marshal(request)
	if err != nil {
		errf := fmt.Errorf("requestJson parse error, err: %v, url: %v", err, parsedURL)
		return []byte{}, errf
	}
	conn.Write(requestJson)
	conn.CloseWrite()

	audioBytes, err := io.ReadAll(conn)
	if err != nil {
		errf := fmt.Errorf("[UDS] read error, err:%v", err)
		return []byte{}, errf
	}

	return audioBytes, nil
}

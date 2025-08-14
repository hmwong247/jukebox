package ytdlp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"

	"github.com/rs/zerolog/log"
)

var (
	YTDLPY_SOCKET_PATH = func() string {
		path := os.Getenv("YTDLPY_SOCKET_PATH")
		if path == "" {
			log.Error().Msg("YTDLPY_SOCKET_PATH not found")
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
	Err       string // error from ytdlpy
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
	log.Debug().Bytes("jsonBytes", jsonBytes).Msg("[UDS] recv: ")

	infoJson := InfoJson{}
	if err := json.Unmarshal(jsonBytes, &infoJson); err != nil {
		errf := fmt.Errorf("json unmarshal error, err: %v", err)
		return InfoJson{}, errf
	}
	// slog.Debug("infoJson", "json", infoJson)
	if infoJson.Err != "" {
		log.Debug().Str("error", infoJson.Err).Msg("Failed to parse infoJson")
		errf := fmt.Errorf("ytdlpy error: %v", infoJson.Err)
		return InfoJson{}, errf
	}

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

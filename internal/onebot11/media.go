package onebot11

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AlonaBot/AlonaBot/sdk/errcode"
	msg "github.com/AlonaBot/AlonaBot/sdk/message"
)

func (i *instance) mediaFile(ctx context.Context, source msg.MediaSource) (file string, cleanup func() error, err error) {
	cleanup = func() error { return nil }
	switch {
	case source.HTTPSURL != "":
		return source.HTTPSURL, cleanup, nil
	case source.InlineBase64 != "":
		return "base64://" + source.InlineBase64, cleanup, nil
	case source.PlatformFile != "":
		file, err = i.platformFileURL(ctx, source.PlatformFile)
		return file, cleanup, err
	case source.MediaRef != "":
		if i.config.MediaDir == nil {
			file, _, err = i.core.MediaURL(ctx, source.MediaRef, i.config.MediaURLBase)
			return
		}
		spool, createErr := os.CreateTemp(i.config.MediaDir.Local, "onebot-*")
		if createErr != nil {
			return "", cleanup, createErr
		}
		cleanup = func() error { return os.Remove(spool.Name()) }
		defer func() {
			err = errors.Join(err, spool.Close())
			if err != nil {
				err = errors.Join(err, cleanup())
				cleanup = func() error { return nil }
			}
		}()
		var offset uint64
		for {
			data, next, eof, readErr := i.core.ReadMedia(ctx, source.MediaRef, offset, 1<<20)
			if readErr != nil {
				return "", cleanup, readErr
			}
			if _, err = spool.Write(data); err != nil {
				return
			}
			offset = next
			if eof {
				break
			}
		}
		file = (&url.URL{Scheme: "file", Path: path.Join(i.config.MediaDir.Remote, filepath.Base(spool.Name()))}).String()
		return
	default:
		return "", cleanup, errcode.InvalidArgument.New("unsupported media source")
	}
}

// ReadPlatformFile 按调用上下文读取平台 URL，支持取消与未实现 Range 的服务。
func (i *instance) ReadPlatformFile(ctx context.Context, fileID string, offset uint64, length uint32) ([]byte, uint64, bool, error) {
	fileID, err := i.platformFileURL(ctx, fileID)
	if err != nil {
		return nil, offset, false, err
	}
	endpoint, err := url.Parse(fileID)
	if err != nil || (endpoint.Scheme != "https" && endpoint.Scheme != "http") {
		return nil, offset, false, errcode.Unsupported.New("platform media has no HTTP URL")
	}
	if length == 0 || length > 1<<20 {
		return nil, offset, false, errcode.InvalidArgument.New("invalid media read length")
	}
	request, err := http.NewRequestWithContext(ctx, "GET", fileID, nil)
	if err != nil {
		return nil, offset, false, err
	}
	request.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+uint64(length)))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, offset, false, err
	}
	defer response.Body.Close()
	if response.StatusCode == 416 {
		return nil, offset, true, nil
	}
	if response.StatusCode != 200 && response.StatusCode != 206 {
		return nil, offset, false, fmt.Errorf("platform media HTTP %d", response.StatusCode)
	}
	if response.StatusCode == 206 {
		start := fmt.Sprintf("bytes %d-", offset)
		if len(response.Header.Get("Content-Range")) < len(start) || response.Header.Get("Content-Range")[:len(start)] != start {
			return nil, offset, false, errors.New("invalid platform media content range")
		}
	} else if offset > 0 {
		if _, err = io.CopyN(io.Discard, response.Body, int64(offset)); err != nil {
			if errors.Is(err, io.EOF) {
				return nil, offset, true, nil
			}
			return nil, offset, false, err
		}
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, int64(length)+1))
	if err != nil {
		return nil, offset, false, err
	}
	eof := len(data) <= int(length)
	if !eof {
		data = data[:length]
	}
	return data, offset + uint64(len(data)), eof, nil
}

// validateInline 在写平台请求前报告损坏的内联来源。
func validateInline(source msg.MediaSource) error {
	if source.InlineBase64 != "" {
		if _, err := base64.StdEncoding.DecodeString(source.InlineBase64); err != nil {
			return fmt.Errorf("invalid inline media: %w", err)
		}
	}
	return nil
}

func targetParams(to string) (map[string]any, error) {
	adapterID, chat, thread, err := msg.ParseSID(to)
	if err != nil {
		return nil, err
	}
	if adapterID != AdapterID || thread != "" {
		return nil, errcode.Unsupported.New("unsupported onebot target")
	}
	for _, kind := range []string{"group", "private"} {
		prefix := kind + ":"
		if len(chat) > len(prefix) && chat[:len(prefix)] == prefix {
			id := chat[len(prefix):]
			if number, err := strconv.ParseInt(id, 10, 64); err != nil || number <= 0 {
				return nil, errcode.InvalidArgument.New("invalid target ID")
			}
			key := "user_id"
			if kind == "group" {
				key = "group_id"
			}
			return map[string]any{"message_type": kind, key: json.Number(id)}, nil
		}
	}
	return nil, errcode.InvalidArgument.New("invalid onebot chat")
}

const fileTokenPrefix = "onebot-file:"

// platformFile 将来源会话与实现端文件元数据留在 Core 登记的私有描述符中。
type platformFile struct {
	Session string
	ID      string
	Data    map[string]json.RawMessage
}

func (i *instance) platformFileURL(ctx context.Context, token string) (string, error) {
	if !strings.HasPrefix(token, fileTokenPrefix) {
		return token, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(token, fileTokenPrefix))
	if err != nil {
		return "", err
	}
	var file platformFile
	if err = json.Unmarshal(raw, &file); err != nil {
		return "", err
	}
	params, err := targetParams(file.Session)
	if err != nil {
		return "", err
	}
	action := "get_private_file_url"
	if params["message_type"] == "group" {
		action = "get_group_file_url"
	}
	delete(params, "message_type")
	params["file_id"] = file.ID
	for _, key := range []string{"busid", "file_hash"} {
		if value, ok := file.Data[key]; ok {
			params[key] = value
		}
	}
	var result struct {
		URL string `json:"url"`
	}
	err = i.callAPI(ctx, action, params, &result, true)
	if err != nil {
		return "", err
	}
	if result.URL == "" {
		return "", errors.New("file URL response has no URL")
	}
	return result.URL, nil
}

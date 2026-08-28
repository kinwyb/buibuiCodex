package wxcom

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kinwyb/buibuiCodex/core/types"
)

// 媒体文件处理

// SendMedia 主动发送媒体资源
func (c *Channel) SendMedia(ctx context.Context, mediaType string, reqID string, chatID string, mediaID string) error {
	if !c.IsRunning() {
		return ErrNotConnected
	}
	cmd := WsCmdSendMsg
	if reqID != "" {
		cmd = WsCmdResponse
	} else {
		reqID = generateReqID(WsCmdSendMsg)
	}
	body := map[string]any{
		"chatid":  chatID,
		"msgtype": mediaType,
		mediaType: map[string]any{
			"media_id": mediaID,
		},
	}
	_, err := c.wsManager.SendReply(ctx, reqID, body, cmd)
	return err
}

// DownloadFile 下载并解密文件
// url: 文件下载地址
// aesKey: Base64编码的AES密钥 (来自消息中的 aeskey)
// 返回: 解密后的文件数据, 文件名, 错误
func (c *Channel) DownloadFile(ctx context.Context, fileURL, aesKey string) ([]byte, string, error) {
	if !c.IsRunning() {
		return nil, "", ErrNotConnected
	}

	timeout := time.Duration(c.config.RequestTimeout) * time.Millisecond
	downloader := NewFileDownloader(timeout)

	return downloader.DownloadFile(ctx, fileURL, aesKey)
}

// DownloadImage 下载并解密图片
// 从 bus.Media 中获取 URL 和 aeskey
func (c *Channel) DownloadImage(ctx context.Context, media *types.Media) ([]byte, string, error) {
	if media == nil || media.URL == "" {
		return nil, "", fmt.Errorf("media URL is empty")
	}

	aesKey := ""
	if media.Metadata != nil {
		if key, ok := media.Metadata["aeskey"].(string); ok {
			aesKey = key
		}
	}

	return c.DownloadFile(ctx, media.URL, aesKey)
}

// UploadMedia 上传临时素材到企业微信
// mediaType: "file" / "image" / "voice" / "video"
// 返回: media_id（3天内有效），错误
func (c *Channel) UploadMedia(ctx context.Context, mediaType, filename string, data []byte) (string, error) {
	if !c.IsRunning() {
		return "", ErrNotConnected
	}

	totalSize := len(data)
	if totalSize < 5 {
		return "", fmt.Errorf("file too small: %d bytes, minimum 5", totalSize)
	}

	// 分片大小 512KB，最多 100 分片
	const chunkSize = 512 * 1024
	totalChunks := (totalSize + chunkSize - 1) / chunkSize
	if totalChunks > 100 {
		return "", fmt.Errorf("file too large: %d chunks, maximum 100", totalChunks)
	}

	// 计算 MD5
	md5 := fmt.Sprintf("%x", computeMD5(data))

	var mediaID string
	var lastErr error

	// 最多重试 3 次（上传会话 30 分钟有效，分片幂等可重复上传）
	for attempt := 1; attempt <= 3; attempt++ {
		if attempt > 1 {
			// 断线后重连，等连接稳定再重试
			time.Sleep(2 * time.Second)
			if !c.IsRunning() {
				return "", fmt.Errorf("connection lost, upload aborted: %w", lastErr)
			}
		}

		// 1. 初始化上传
		uploadID, err := c.uploadMediaInit(ctx, mediaType, filename, totalSize, totalChunks, md5)
		if err != nil {
			lastErr = fmt.Errorf("upload init failed: %w", err)
			if attempt < 3 {
				c.logger.Warn("Upload init failed, will retry", "attempt", attempt, "error", err)
				continue
			}
			return "", lastErr
		}

		// 2. 分片上传
		uploadFailed := false
		for i := range totalChunks {
			start := i * chunkSize
			end := start + chunkSize
			if end > totalSize {
				end = totalSize
			}
			chunkData := data[start:end]
			if err := c.uploadMediaChunk(ctx, uploadID, i, chunkData); err != nil {
				lastErr = fmt.Errorf("upload chunk %d failed: %w", i, err)
				if attempt < 3 {
					c.logger.Warn("Upload chunk failed, will retry from init", "attempt", attempt, "chunk", i, "error", err)
				}
				uploadFailed = true
				break
			}
		}
		if uploadFailed {
			continue
		}

		// 3. 完成上传
		mediaID, err = c.uploadMediaFinish(ctx, uploadID)
		if err != nil {
			lastErr = fmt.Errorf("upload finish failed: %w", err)
			if attempt < 3 {
				c.logger.Warn("Upload finish failed, will retry", "attempt", attempt, "error", err)
				continue
			}
			return "", lastErr
		}

		return mediaID, nil
	}

	return "", lastErr
}

// uploadMediaInit 上传初始化
func (c *Channel) uploadMediaInit(ctx context.Context, mediaType, filename string, totalSize, totalChunks int, md5 string) (string, error) {
	body := map[string]any{
		"type":         mediaType,
		"filename":     filename,
		"total_size":   totalSize,
		"total_chunks": totalChunks,
		"md5":          md5,
	}

	frame, err := c.wsManager.SendCommand(ctx, WsCmdUploadMediaInit, body, 30*time.Second)
	if err != nil {
		return "", err
	}

	if frame.ErrCode != 0 {
		return "", fmt.Errorf("upload init error: %d %s", frame.ErrCode, frame.ErrMsg)
	}

	if uploadID, ok := getStringFromMap(frame.Body, "upload_id"); ok {
		return uploadID, nil
	}
	return "", fmt.Errorf("upload_id not found in response")
}

// uploadMediaChunk 上传分片
func (c *Channel) uploadMediaChunk(ctx context.Context, uploadID string, chunkIndex int, data []byte) error {
	body := map[string]any{
		"upload_id":   uploadID,
		"chunk_index": chunkIndex,
		"base64_data": base64.StdEncoding.EncodeToString(data),
	}

	frame, err := c.wsManager.SendCommand(ctx, WsCmdUploadMediaChunk, body, 30*time.Second)
	if err != nil {
		return err
	}

	if frame.ErrCode != 0 {
		return fmt.Errorf("upload chunk %d error: %d %s", chunkIndex, frame.ErrCode, frame.ErrMsg)
	}

	return nil
}

// uploadMediaFinish 完成上传
func (c *Channel) uploadMediaFinish(ctx context.Context, uploadID string) (string, error) {
	body := map[string]any{
		"upload_id": uploadID,
	}

	frame, err := c.wsManager.SendCommand(ctx, WsCmdUploadMediaFinish, body, 30*time.Second)
	if err != nil {
		return "", err
	}

	if frame.ErrCode != 0 {
		return "", fmt.Errorf("upload finish error: %d %s", frame.ErrCode, frame.ErrMsg)
	}

	if mediaID, ok := getStringFromMap(frame.Body, "media_id"); ok {
		return mediaID, nil
	}
	return "", fmt.Errorf("media_id not found in response")
}

// processMedia 下载并解密媒体文件，填充 Media 字段
func (c *Channel) processMedia(ctx context.Context, inbound *types.InputMessage) {
	// 倒序遍历以支持安全删除 document 项
	for i := len(inbound.Media) - 1; i >= 0; i-- {
		media := &inbound.Media[i]
		switch media.Type {
		case "image", "audio":
			data, filename, err := c.DownloadImage(ctx, media)
			if err != nil {
				c.logger.Warn("Failed to download media", "type", media.Type, "error", err)
				continue
			}
			media.Base64 = base64.StdEncoding.EncodeToString(data)
			media.URL = "" // 清空 URL，避免 OpenAI 接口优先使用已失效的临时链接
			media.MimeType = DetectMimeType(data)
			if media.Metadata == nil {
				media.Metadata = make(map[string]any)
			}
			media.Metadata["filename"] = filename

		case "document":
			aesKey := ""
			if media.Metadata != nil {
				if key, ok := media.Metadata["aeskey"].(string); ok {
					aesKey = key
				}
			}
			data, filename, err := c.DownloadFile(ctx, media.URL, aesKey)
			if err != nil {
				c.logger.Warn("Failed to download document", "error", err)
				continue
			}
			tmpPathName := aesKey
			text, filePath, ext := extractText(data, filename, tmpPathName)
			if text != "" {
				media.Base64 = base64.StdEncoding.EncodeToString([]byte(text))
				media.URL = ""
				media.MimeType = ext
			} else {
				media.URL = filePath
				media.MimeType = ext
			}
			media.Metadata["filename"] = filename
		}
	}
}

// 常见纯文本扩展名
var textExtensions = map[string]bool{
	".txt":  true,
	".csv":  true,
	".json": true,
	".md":   true,
	".xml":  true,
	".yaml": true,
	".yml":  true,
	".log":  true,
	".ini":  true,
	".conf": true,
	".cfg":  true,
	".env":  true,
	".sh":   true,
	".html": true,
	".htm":  true,
	".sql":  true,
	".js":   true,
	".ts":   true,
	".py":   true,
	".go":   true,
	".java": true,
	".c":    true,
	".h":    true,
	".css":  true,
}

// extractText 从文件数据中提取文本内容
func extractText(data []byte, filename string, tmpPathName string) (string, string, string) {
	ext := strings.ToLower(filepath.Ext(filename))

	// 纯文本文件：直接返回
	if textExtensions[ext] {
		if t := strings.TrimSpace(string(data)); t != "" {
			return t, "", ext
		}
		return "(空文件)", "", ext
	}

	tmpFile := filepath.Join(TempFilePath, time.Now().Format("20060102"), tmpPathName)

	if _, err := os.Stat(tmpFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			me := os.MkdirAll(tmpFile, os.ModePerm)
			if me != nil {
				return fmt.Sprintf("临时文件目录创建失败: %s", me.Error()), "", tmpFile
			}
		}
	}
	// ZIP 格式：docx / xlsx / pptx
	if ext == ".zip" {
		return extractZipText(data, ext, tmpFile), "", ext
	}
	tmp := filepath.Join(tmpFile, filename)
	err := os.WriteFile(tmp, data, os.ModePerm)
	if err != nil {
		return fmt.Sprintf("文件下载失败：%s", err.Error()), "", ext
	}
	return "", tmp, ext
}

// extractZipText 从 ZIP 文件中提取文本
func extractZipText(data []byte, filename, tmpFilePath string) string {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Sprintf("(ZIP 解析失败: %s)", err.Error())
	}
	sb := strings.Builder{}
	sb.WriteString(filename + "这个zip压缩包中存在以下这些文件:\n")
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			continue
		}
		xmlData, _ := io.ReadAll(rc)
		rc.Close()
		tmp := filepath.Join(tmpFilePath, f.Name)
		_ = os.WriteFile(tmp, xmlData, f.Mode())
		sb.WriteString(tmp + "\n")
	}
	return sb.String()
}

// computeMD5 计算数据的 MD5 值
func computeMD5(data []byte) []byte {
	h := md5.New()
	h.Write(data)
	return h.Sum(nil)
}

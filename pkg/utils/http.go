package utils

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// HTTPClient HTTP客户端封装
type HTTPClient struct {
	client  *http.Client
	timeout time.Duration
}

// NewHTTPClient 创建新的HTTP客户端
func NewHTTPClient(timeout time.Duration, skipVerify bool) *HTTPClient {
	// 创建传输配置
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: skipVerify,
		},
	}

	// 创建HTTP客户端
	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	return &HTTPClient{
		client:  client,
		timeout: timeout,
	}
}

// DefaultHTTPClient 默认HTTP客户端（超时10秒，验证TLS）
func DefaultHTTPClient() *HTTPClient {
	return NewHTTPClient(10*time.Second, false)
}

// Get 发送GET请求
func (c *HTTPClient) Get(url string, headers map[string]string) (*http.Response, error) {
	// 创建请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 添加请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 发送请求
	return c.client.Do(req)
}

// GetWithContext 带上下文的GET请求
func (c *HTTPClient) GetWithContext(ctx context.Context, url string, headers map[string]string) (*http.Response, error) {
	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 添加请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 发送请求
	return c.client.Do(req)
}

// DownloadFile 下载文件
func (c *HTTPClient) DownloadFile(url, destPath string) error {
	// 确保目标目录存在
	dir := filepath.Dir(destPath)
	if err := CreateDirIfNotExist(dir); err != nil {
		return err
	}

	// 创建请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载文件失败，状态码: %d", resp.StatusCode)
	}

	// 创建目标文件
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %v", err)
	}
	defer out.Close()

	// 复制内容到文件
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("写入文件内容失败: %v", err)
	}

	return nil
}

// DownloadFileWithProgress 带进度的文件下载
func (c *HTTPClient) DownloadFileWithProgress(url, destPath string, progress func(current, total int64)) error {
	// 确保目标目录存在
	dir := filepath.Dir(destPath)
	if err := CreateDirIfNotExist(dir); err != nil {
		return err
	}

	// 创建请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载文件失败，状态码: %d", resp.StatusCode)
	}

	// 获取文件大小
	contentLength := resp.ContentLength

	// 创建目标文件
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %v", err)
	}
	defer out.Close()

	// 创建进度读取器
	progressReader := &ProgressReader{
		Reader:   resp.Body,
		Total:    contentLength,
		Progress: progress,
	}

	// 复制内容到文件
	_, err = io.Copy(out, progressReader)
	if err != nil {
		return fmt.Errorf("写入文件内容失败: %v", err)
	}

	return nil
}

// ProgressReader 进度读取器
type ProgressReader struct {
	Reader   io.Reader
	Total    int64
	Current  int64
	Progress func(current, total int64)
}

// Read 实现io.Reader接口
func (r *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	if n > 0 {
		r.Current += int64(n)
		if r.Progress != nil {
			r.Progress(r.Current, r.Total)
		}
	}
	return
}

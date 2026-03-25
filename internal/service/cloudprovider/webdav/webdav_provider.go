package webdav

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// WebDavConfig WebDav 配置
type WebDavConfig struct {
	Server   string
	Username string
	Password string
}

// XML 解析结构
type MultiStatus struct {
	XMLName  xml.Name  `xml:"DAV: multistatus"`
	Response []Response `xml:"DAV: response"`
}

type Response struct {
	Href     string   `xml:"DAV: href"`
	Propstat Propstat `xml:"DAV: propstat"`
}

type Propstat struct {
	Status string `xml:"DAV: status"`
	Prop   Prop   `xml:"DAV: prop"`
}

type Prop struct {
	IsCollection string `xml:"DAV: iscollection"`
	DisplayName  string `xml:"DAV: displayname"`
}

// WebDavProvider WebDav 云存储提供商
type WebDavProvider struct {
	config WebDavConfig
	client *http.Client
}

// NewWebDavProvider 创建 WebDav 提供商
func NewWebDavProvider(config WebDavConfig) (*WebDavProvider, error) {
	if config.Server == "" {
		return nil, fmt.Errorf("WebDav 服务器地址不能为空")
	}

	// 确保服务器地址包含协议方案
	if !strings.HasPrefix(config.Server, "http://") && !strings.HasPrefix(config.Server, "https://") {
		config.Server = "http://" + config.Server
	}

	// 确保服务器地址以 / 结尾
	if !strings.HasSuffix(config.Server, "/") {
		config.Server += "/"
	}

	return &WebDavProvider{
		config: config,
		client: &http.Client{},
	}, nil
}

// setBasicAuth 设置 HTTP Basic Auth 头部
func (p *WebDavProvider) setBasicAuth(req *http.Request) {
	if p.config.Username != "" && p.config.Password != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(p.config.Username + ":" + p.config.Password))
		req.Header.Set("Authorization", "Basic "+auth)
	}
}

// UploadFile 上传文件到 WebDav
func (p *WebDavProvider) UploadFile(ctx context.Context, cloudPath, localPath string) error {
	// 打开本地文件
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %w", err)
	}
	defer file.Close()

	// 构建 WebDav URL
	webdavURL := p.config.Server + cloudPath

	// 创建 PUT 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, webdavURL, file)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置 Basic Auth
	p.setBasicAuth(req)

	// 发送请求
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("上传文件失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("上传文件失败，状态码: %d", resp.StatusCode)
	}

	return nil
}

// DownloadFile 从 WebDav 下载文件
func (p *WebDavProvider) DownloadFile(ctx context.Context, cloudPath, localPath string) error {
	// 构建 WebDav URL
	webdavURL := p.config.Server + cloudPath

	// 创建 GET 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, webdavURL, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置 Basic Auth
	p.setBasicAuth(req)

	// 发送请求
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("下载文件失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载文件失败，状态码: %d", resp.StatusCode)
	}

	// 确保本地目录存在
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("创建本地目录失败: %w", err)
	}

	// 创建本地文件
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("创建本地文件失败: %w", err)
	}
	defer file.Close()

	// 复制内容
	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// ListObjects 列出 WebDav 目录下的对象
func (p *WebDavProvider) ListObjects(ctx context.Context, prefix string) ([]string, error) {
	// 构建 WebDav URL
	webdavURL := p.config.Server + prefix

	// 创建 PROPFIND 请求
	req, err := http.NewRequestWithContext(ctx, "PROPFIND", webdavURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置 Basic Auth
	p.setBasicAuth(req)

	// 设置深度
	req.Header.Set("Depth", "1")

	// 发送请求
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("列出对象失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusMultiStatus {
		return nil, fmt.Errorf("列出对象失败，状态码: %d", resp.StatusCode)
	}

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析 XML
	var multiStatus MultiStatus
	err = xml.Unmarshal(body, &multiStatus)
	if err != nil {
		return nil, fmt.Errorf("解析 XML 失败: %w", err)
	}

	var result []string

	// 处理每个响应
	for _, response := range multiStatus.Response {
		// 解析 href
		parsedHref, err := url.Parse(response.Href)
		if err != nil {
			continue
		}

		// 提取路径部分
		path := parsedHref.Path

		// 从服务器地址中提取基础路径
		baseURL, _ := url.Parse(p.config.Server)
		basePath := baseURL.Path

		// 移除基础路径前缀
		relativePath := strings.TrimPrefix(path, basePath)

		// 跳过目录本身和空路径
		if relativePath == "" || relativePath == "." {
			continue
		}

		// 检查是否是集合（目录），如果是则跳过
		if response.Propstat.Prop.IsCollection == "1" {
			continue
		}

		// 添加到结果
		result = append(result, relativePath)
	}

	return result, nil
}

// DeleteObject 删除 WebDav 上的对象
func (p *WebDavProvider) DeleteObject(ctx context.Context, key string) error {
	// 构建 WebDav URL
	webdavURL := p.config.Server + key

	// 创建 DELETE 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, webdavURL, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置 Basic Auth
	p.setBasicAuth(req)

	// 发送请求
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("删除对象失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("删除对象失败，状态码: %d", resp.StatusCode)
	}

	return nil
}

// TestConnection 测试 WebDav 连接
func (p *WebDavProvider) TestConnection(ctx context.Context) error {
	// 构建 WebDav URL
	webdavURL := p.config.Server

	// 创建 OPTIONS 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodOptions, webdavURL, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置 Basic Auth
	p.setBasicAuth(req)

	// 发送请求
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("连接测试失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("连接测试失败，状态码: %d", resp.StatusCode)
	}

	return nil
}

// EnsureDir 确保 WebDav 目录存在
func (p *WebDavProvider) EnsureDir(ctx context.Context, path string) error {
	// WebDav 会在上传文件时自动创建目录，所以这里可以为空实现
	return nil
}

// GetCloudPath 获取 WebDav 云路径
func (p *WebDavProvider) GetCloudPath(userID, subPath string) string {
	// WebDav 路径直接使用 userID 和 subPath
	return fmt.Sprintf("%s/%s", userID, subPath)
}

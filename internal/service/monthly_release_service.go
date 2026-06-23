package service

import (
	"context"
	"fmt"
	"io"
	"lunabox/internal/applog"
	"lunabox/internal/utils"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MonthlyReleaseService 每月游戏发售列表服务
type MonthlyReleaseService struct {
	ctx    context.Context
	getter *utils.MonthlyInfoGetter
}

func NewMonthlyReleaseService() *MonthlyReleaseService {
	return &MonthlyReleaseService{
		getter: utils.NewMonthlyInfoGetter(),
	}
}

func (s *MonthlyReleaseService) Init(ctx context.Context) {
	s.ctx = ctx
}

// FetchMonthlyReleases 获取指定年月的 Getchu 游戏发售列表
// year: 年份（如 2026），month: 月份（1-12）
func (s *MonthlyReleaseService) FetchMonthlyReleases(year, month int, age string) (utils.MonthlyReleaseResult, error) {
	applog.InfoLogSaveAppLog("MonthlyReleaseService: fetching releases for %d/%02d", year, month)
	result, err := s.getter.FetchMonthlyReleases(year, month, age)
	if err != nil {
		applog.LogErrorf(s.ctx, "MonthlyReleaseService: failed to fetch releases: %v", err)
		return utils.MonthlyReleaseResult{}, err
	}

	// 准备临时图片文件夹（每次进入页面都清空）
	tempDir, err := s.prepareTempDir()
	if err != nil {
		applog.LogErrorf(s.ctx, "MonthlyReleaseService: failed to prepare temp dir: %v", err)
		return result, nil // 返回不带本地图片的结果
	}

	// 并发下载所有封面图片
	s.downloadAllCovers(result.Groups, tempDir)

	return result, nil
}

// prepareTempDir 准备临时图片目录，清除旧文件
func (s *MonthlyReleaseService) prepareTempDir() (string, error) {
	dataDir, err := utils.GetDataDir()
	if err != nil {
		return "", err
	}
	tempDir := filepath.Join(dataDir, "monthly", "temp")

	// 清除旧文件夹
	// os.RemoveAll(tempDir)
	// 重新创建
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		err = os.MkdirAll(tempDir, os.ModePerm)
		if err != nil {
			return "", err
		}
	}

	applog.InfoLogSaveAppLog("MonthlyReleaseService: temp dir prepared: %s", tempDir)
	return tempDir, nil
}

// downloadAllCovers 并发下载所有游戏封面图片
func (s *MonthlyReleaseService) downloadAllCovers(groups []utils.MonthlyReleaseGroup, tempDir string) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // 最大 5 并发

	for i := range groups {
		for j := range groups[i].Games {
			game := &groups[i].Games[j]
			if game.CoverURL == "" {
				continue
			}
			wg.Add(1)
			sem <- struct{}{}
			go func(g *utils.MonthlyReleaseGame) {
				defer wg.Done()
				defer func() { <-sem }()

				localPath := filepath.Join(tempDir, g.GetchuID+".jpg")
				err := s.downloadCoverImage(g.CoverURL, localPath)
				if err != nil {
					applog.InfoLogSaveAppLog("MonthlyReleaseService: failed to download cover [%s]: %v", g.GetchuID, err)
					return
				}
				// 替换为本地路径
				g.CoverURL = localPath
			}(game)
		}
	}
	wg.Wait()
}

// downloadCoverImage 下载单个封面图片，带 Referer 绕过防盗链
func (s *MonthlyReleaseService) downloadCoverImage(imageURL, localPath string) error {
	client := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Referer", determineReferer(imageURL))

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create file failed: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(localPath)
		return fmt.Errorf("write file failed: %w", err)
	}

	return nil
}

// FetchGetchuImages 获取 Getchu 游戏截图并下载到临时文件夹，返回本地路径列表
func (s *MonthlyReleaseService) FetchGetchuImages(imageUrls []string) ([]string, error) {
	if len(imageUrls) == 0 {
		return []string{}, nil
	}

	// 使用与封面图相同的临时目录
	tempDir, err := s.prepareTempDir()
	if err != nil {
		return []string{}, err
	}

	// 下载所有截图到临时目录
	var localPaths []string
	for _, imageUrl := range imageUrls {
		if imageUrl == "" {
			continue
		}
		// 使用时间戳和索引生成唯一文件名，避免多张图片覆盖
		localPath := filepath.Join(tempDir, fmt.Sprintf("%s.jpg", uuid.New().String()))
		err := s.downloadCoverImage(imageUrl, localPath)
		if err != nil {
			applog.InfoLogSaveAppLog("MonthlyReleaseService: failed to download image [%s]: %v", imageUrl, err)
			continue
		}
		applog.InfoLogSaveAppLog("FetchGetchuImages: downloaded [%s] to [%s]", imageUrl, localPath)
		localPaths = append(localPaths, localPath)
	}

	return localPaths, nil
}

// ClearGetchuTempImages 清空 Getchu 截图临时文件夹
func (s *MonthlyReleaseService) ClearGetchuTempImages() error {
	tempDir, err := s.prepareTempDir()
	if err != nil {
		return err
	}
	return os.RemoveAll(tempDir)
}

package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"

	// 添加 GoQuery 导入
	"github.com/gocolly/colly/v2" // 添加 Colly 导入
)

type SaveInfoGetter struct {
	client  *http.Client
	timeout time.Duration
}

func NewSaveInfoGetter() *SaveInfoGetter {
	return &SaveInfoGetter{
		client:  &http.Client{},
		timeout: 10 * time.Second,
	}
}

type potentialGame struct {
	Title string
	Link  string
}

func (b SaveInfoGetter) FetchSeiyaSave(name string) error {
	url, err := b.FetchSeiyaSaveUrl(name)
	// dataDir, err := GetDataDir()
	// coversDestDir := filepath.Join(dataDir, "1.zip")
	zipPath := `C:\temp\projects\lunabox\build\bin\1.zip`
	err = downloadFile(url, zipPath)
	destDir := `C:\temp\projects\lunabox\build\bin\extracted`
	target := `C:\temp\projects\lunabox\build\bin\target`
	if err := extractZip(zipPath, destDir); err != nil {
		return fmt.Errorf("解压失败: %v", err)
	}
	err = overrideFiles(destDir, target)
	if err != nil {
		fmt.Println("FetchSeiyaSave错误：", err)
	}
	return err
}

func (b SaveInfoGetter) FetchSeiyaSaveUrl(name string) (string, error) {
	var url string = "https://seiya-saiga.com/save.html"
	c := CreateCollector("*seiya-saiga.com")

	var potentialGames []struct {
		Title string
		Link  string
	}
	gameName := ""
	gameLink := ""

	// 处理搜索结果页面中的游戏条目
	c.OnHTML("div > table > tbody > tr > th table tbody tr", func(e *colly.HTMLElement) {
		titles := e.ChildTexts("td b")
		title := ""
		if len(titles) > 0 {
			title, _ = shiftJISToUTF8(titles[0])
		}

		link := e.ChildAttr("a", "href")
		// th := e.ChildText("th")
		// // log.Print("OnHTML 网页列表 ：", e.Text)
		// if th != "" {
		// 	fmt.Println("th:", th)
		// }

		if link == "" || title == "" || strings.Contains(link, "#") || !strings.Contains(link, "save/") {
			return
		}
		fmt.Println("title:", title)

		potentialGames = append(potentialGames, struct {
			Title string
			Link  string
		}{
			Title: title,
			Link:  e.Request.AbsoluteURL(link),
		})
	})

	// 在访问完搜索页面后进行过滤和处理
	c.OnScraped(func(r *colly.Response) {
		for _, gameFound := range potentialGames {
			// 应用过滤条件
			if gameFound.Link == "" || !strings.Contains(gameFound.Title, name) {
				continue
			}
			gameName = gameFound.Title
			gameLink = gameFound.Link
			return
		}
	})

	// 错误处理
	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Request error: %s with error: %s\n", r.Request.URL, err)
	})

	// 访问构建的 URL
	err := c.Visit(url)
	if err != nil {
		return "", err
	}

	// 等待收集完成
	c.Wait()
	fmt.Printf("找到游戏‘%s’的存档%s\n", gameName, gameLink)

	return gameLink, err
}

func shiftJISToUTF8(shiftJISData string) (string, error) {
	// 创建 Shift-JIS 解码器
	decoder := japanese.ShiftJIS.NewDecoder()

	// 转换编码
	reader := transform.NewReader(strings.NewReader(shiftJISData), decoder)
	utf8Data, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(utf8Data), nil
}

func downloadFile(url, localPath string) error {
	// 发起 HTTP 请求
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("无法访问 URL: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	}

	// 创建本地文件
	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("无法创建本地文件: %v", err)
	}
	defer out.Close()

	// 将响应内容写入文件
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	fmt.Printf("文件已成功下载到: %s\n", localPath)
	return nil
}

// overrideFiles 将 sourcePath 中的所有文件覆盖到 targetPath 中
// 支持任意目标路径结构，仅通过文件名匹配
func overrideFiles(sourcePath string, targetPath string) error {
	overwrittenCount := 0 // 记录覆盖的文件数量

	// 遍历 sourcePath 中的所有文件
	err := filepath.Walk(sourcePath, func(srcFilePath string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("访问文件失败: %v", err)
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 获取源文件名
		srcFileName := filepath.Base(srcFilePath)

		// 在目标路径中查找同名文件
		dstFilePath, err := findFileInTarget(srcFileName, targetPath)
		if err != nil {
			return fmt.Errorf("查找目标文件失败: %v", err)
		}

		// 如果未找到匹配文件，跳过
		if dstFilePath == "" {
			fmt.Printf("未找到匹配文件，跳过: %s\n", srcFileName)
			return nil
		}

		// 执行覆盖操作
		if err := copyFile(srcFilePath, dstFilePath); err != nil {
			return fmt.Errorf("复制文件失败: %v", err)
		}

		// 增加覆盖计数
		overwrittenCount++
		fmt.Printf("已覆盖文件: %s -> %s\n", srcFilePath, dstFilePath)
		return nil
	})

	// 如果遍历过程中发生错误，直接返回
	if err != nil {
		return err
	}

	// 如果没有覆盖任何文件，返回错误
	if overwrittenCount == 0 {
		return fmt.Errorf("未覆盖任何文件，请检查源路径或目标路径是否正确")
	}

	fmt.Printf("总共覆盖了 %d 个文件。\n", overwrittenCount)
	return nil
}

// findFileInTarget 在目标路径中递归查找与指定文件名匹配的文件
func findFileInTarget(fileName, targetPath string) (string, error) {
	var matchedPath string

	// 遍历目标路径
	err := filepath.Walk(targetPath, func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 对比文件名
		if filepath.Base(currentPath) == fileName {
			matchedPath = currentPath
			return filepath.SkipDir // 找到后立即停止遍历当前目录
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("遍历目标路径失败: %v", err)
	}

	return matchedPath, nil
}

// copyFile 将源文件复制到目标文件（覆盖）
func copyFile(srcPath, dstPath string) error {
	// 打开源文件
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %v", err)
	}
	defer srcFile.Close()

	// 创建目标文件的父目录
	if err := os.MkdirAll(filepath.Dir(dstPath), os.ModePerm); err != nil {
		return fmt.Errorf("创建目标目录失败: %v", err)
	}

	// 检查文件扩展名是否为 .txt
	if filepath.Ext(srcPath) == ".txt" {
		// 读取源文件内容
		data, err := ioutil.ReadAll(srcFile)
		if err != nil {
			return fmt.Errorf("读取源文件失败: %v", err)
		}

		// 将 Shift-JIS 编码转换为 UTF-8
		utf8Data, err := shiftJISToUTF8(string(data))
		if err != nil {
			return fmt.Errorf("编码转换失败: %v", err)
		}

		// 创建目标文件并写入转换后的内容
		dstFile, err := os.Create(dstPath)
		if err != nil {
			return fmt.Errorf("创建目标文件失败: %v", err)
		}
		defer dstFile.Close()

		if _, err := dstFile.WriteString(utf8Data); err != nil {
			return fmt.Errorf("写入目标文件失败: %v", err)
		}
	} else {
		// 非 .txt 文件直接复制
		dstFile, err := os.Create(dstPath)
		if err != nil {
			return fmt.Errorf("创建目标文件失败: %v", err)
		}
		defer dstFile.Close()

		// 复制内容
		if _, err := io.Copy(dstFile, srcFile); err != nil {
			return fmt.Errorf("复制文件内容失败: %v", err)
		}
	}

	return nil
}

func extractZip(zipPath, destDir string) error {
	// 打开 ZIP 文件
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("无法打开 ZIP 文件: %v", err)
	}
	defer r.Close()

	// 确保目标目录存在
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return fmt.Errorf("创建目标目录失败: %v", err)
	}

	// 遍历 ZIP 文件中的每个文件
	for _, f := range r.File {
		// 构造目标文件路径
		filename := ""
		filename, err = shiftJISToUTF8(f.Name)
		targetPath := filepath.Join(destDir, filename)

		// 如果是目录，直接创建
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, os.ModePerm); err != nil {
				return fmt.Errorf("创建目录失败: %v", err)
			}
			continue
		}

		// 如果是文件，解压并写入
		if err := extractFile(f, targetPath); err != nil {
			return fmt.Errorf("解压文件失败: %v", err)
		}
	}

	fmt.Printf("ZIP 文件已成功解压到: %s\n", destDir)
	return nil
}

// extractFile 解压单个文件
func extractFile(file *zip.File, targetPath string) error {
	// 创建目标文件的父目录
	if err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
		return fmt.Errorf("创建父目录失败: %v", err)
	}

	// 创建目标文件
	dstFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}
	defer dstFile.Close()

	// 打开源文件
	srcFile, err := file.Open()
	if err != nil {
		return fmt.Errorf("打开 ZIP 中的文件失败: %v", err)
	}
	defer srcFile.Close()

	// 复制内容
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("复制文件内容失败: %v", err)
	}

	return nil
}

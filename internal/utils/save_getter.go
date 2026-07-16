package utils

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"lunabox/internal/applog"
	"lunabox/internal/enums"
	"lunabox/internal/models"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"

	// 添加 GoQuery 导入
	"github.com/PuerkitoBio/goquery"
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

// ExtractFilename 从 URL 中提取文件名
func ExtractFilename(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("URL 解析失败: %v", err)
	}

	// 尝试从查询参数中获取文件名
	query := parsedURL.Query()
	filename := query.Get("filename")

	// 如果查询参数中没有文件名，则从路径中提取
	if filename == "" {
		filename = path.Base(parsedURL.Path)
	}
	filename = "klb_savedata_" + filename

	return filename, nil
}

func (b SaveInfoGetter) FetchSeiyaSave(name string, folder string, isOverride bool) (string, error) {
	url, err := b.FetchSeiyaSaveUrl(name)
	if err != nil {
		return "", fmt.Errorf("获取存档链接失败 err:%v", err)
	}
	fileName, err := ExtractFilename(url)
	if err != nil {
		return "", fmt.Errorf("获取文件名失败 err:%v", err)
	}
	// dataDir, err := GetDataDir()
	// coversDestDir := filepath.Join(dataDir, "1.zip")
	zipPath := folder + "\\" + fileName
	err = downloadFile(url, zipPath)
	if err != nil {
		return "", fmt.Errorf("下载文件失败 err:%v", err)
	}

	target := folder

	if isOverride {
		extractedDir := filepath.Join(os.TempDir() + "extracted")
		err = extractZip(zipPath, extractedDir)
		if err != nil {
			return "", fmt.Errorf("解压失败: %v", err)
		}
		err = OverrideFiles(extractedDir, target, "")
		if err != nil {
			return "", fmt.Errorf("FetchSeiyaSave错误：%v", err)
		}
		err = os.Remove(zipPath)
		err = os.Remove(extractedDir)
		return "", err
	}

	return zipPath, err
}

func (b SaveInfoGetter) DownloadFileInFolder(url string) (string, error) {
	folder, err := os.UserHomeDir()
	folder = filepath.Join(folder, "Downloads")

	fileName, err := ExtractFilename(url)
	if err != nil {
		return "", fmt.Errorf("获取文件名失败 err:%v", err)
	}
	// dataDir, err := GetDataDir()
	// coversDestDir := filepath.Join(dataDir, "1.zip")
	zipPath := folder + "\\" + fileName
	err = downloadFile(url, zipPath)
	if err != nil {
		os.Remove(zipPath)
		return "", fmt.Errorf("下载文件失败 err:%v", err)
	}

	return zipPath, err
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
	mainTitle, _, _ := getTitles(name)
	firstName := mainTitle

	// 处理搜索结果页面中的游戏条目
	c.OnHTML("div > table > tbody > tr > th table tbody tr", func(e *colly.HTMLElement) {
		titles := e.ChildTexts("td b")
		title := ""
		if len(titles) > 0 {
			title, _ = shiftJISToUTF8(titles[0])
		}

		link := e.ChildAttr("a", "href")

		if link == "" || title == "" || strings.Contains(link, "#") || !strings.Contains(link, "save/") {
			return
		}

		if !strings.Contains(title, firstName) {
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

		gameFound, s := searchNameByRegexSimilarity(potentialGames, name, false, []string{"セット", "PSV", "PS4", "PSP", "Android"}, func(t1 struct {
			Title string
			Link  string
		}) string {
			return t1.Title
		})
		if gameFound != nil && s > 0.7 {
			gameName = gameFound.Title
			gameLink = gameFound.Link
		}
	})

	// 错误处理
	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Request error: %s with error: %s\n", r.Request.URL, err)
	})

	// 访问构建的 URL
	err := c.Visit(url)
	if err != nil {
		return "", fmt.Errorf("找不到游戏%s的存档,err:%v", name, err)
	}

	// 等待收集完成
	c.Wait()
	fmt.Printf("找到游戏‘%s’的存档%s\n", gameName, gameLink)
	if gameLink == "" {
		return "", fmt.Errorf("找不到游戏%s的存档", name)
	}

	return gameLink, err
}

type SaveResult struct {
	GameName    string `json:"game_name"`
	ArchiveName string `json:"archive_name"`
	SavePath    string `json:"save_path"`
	SourcePath  string `json:"source_path"`
	GameID      string `json:"game_id"`
	GamePath    string `json:"game_path"`
}

func (b SaveInfoGetter) DownloadSavesForGames(games []models.Game, isOverride bool) ([]SaveResult, error) {
	var results []SaveResult
	names := []string{}
	for _, game := range games {
		names = append(names, game.SearchName)
	}
	savesMap, err := b.FetchSeiyaSaveUrlMap(names)
	if err != nil {
		return results, err
	}
	for _, game := range games {
		saveLink, ok := savesMap[game.SearchName]
		if !ok {
			continue
		}
		fileName, err := ExtractFilename(saveLink)
		if err != nil {
			continue
		}
		saveTargetPath := filepath.Join(filepath.Dir(game.Path), fileName)
		err = downloadFile(saveLink, saveTargetPath)
		if err != nil || !isOverride {
			continue
		}

		extractedDir := filepath.Join(os.TempDir(), "extracted", game.ID)
		err = extractZip(saveTargetPath, extractedDir)
		entries, err := os.ReadDir(extractedDir)
		foundIndex := -1
		for i, entry := range entries {
			if entry.IsDir() {
				foundIndex = i
				break
			}
		}
		if foundIndex == -1 {
			fmt.Println("未找到存档文件夹", len(entries))
			os.RemoveAll(extractedDir)
			continue
		}
		eTarget := filepath.Join(extractedDir, entries[foundIndex].Name())
		target := getSavePathFromReadme(eTarget, &game)
		fmt.Println("存档说明书中位置：", target)
		if target == "" {
			target = game.SavePath
		}
		if target == "" {
			newGame, err := SearchSave(game)
			if err == nil {
				target = newGame.SavePath
				if target == "" {
					continue
				}
			}
		}
		fmt.Printf("存档位置：%s\n", target)

		entries, err = os.ReadDir(eTarget)
		foundIndex = -1
		for i, entry := range entries {
			if entry.IsDir() {
				foundIndex = i
				break
			}
		}

		sourcePath := eTarget
		if foundIndex != -1 {
			sourcePath = filepath.Join(eTarget, entries[foundIndex].Name())
		}

		results = append(results, SaveResult{
			GameName:    game.Name,
			ArchiveName: fileName,
			SavePath:    target,
			SourcePath:  sourcePath,
			GameID:      game.ID,
			GamePath:    game.Path,
		})
	}
	return results, nil
}

func (b SaveInfoGetter) OverrideSaves(results []SaveResult) error {
	for _, result := range results {
		fmt.Println("copying", result.SourcePath, "to", result.SavePath)
		OverrideFiles(result.SourcePath, result.SavePath, filepath.Dir(result.GamePath))
		extractedDir := filepath.Join(os.TempDir(), "extracted", result.GameID)
		os.RemoveAll(extractedDir)
	}
	return nil
}

func getSavePathFromReadme(eTarget string, game *models.Game) string {
	readmePath := filepath.Join(eTarget, "説明書.txt")
	entries, err := os.ReadDir(eTarget)
	saveFileName := ""
	fullpath := ""
	for _, entry := range entries {
		ext := filepath.Ext(entry.Name())
		if ext != ".txt" {
			saveFileName = entry.Name()
		}
	}
	if saveFileName == "" {
		return ""
	}
	data, err := os.ReadFile(readmePath)
	if err != nil {
		sjisReadme, _ := utf8ToShiftJIS("説明書.txt")
		readmePath = filepath.Join(eTarget, sjisReadme)
		data, err = os.ReadFile(readmePath)
		if err != nil {
			return ""
		}
	}
	decoder := japanese.ShiftJIS.NewDecoder()
	reader := transform.NewReader(bytes.NewReader(data), decoder)
	utf8Content, err := io.ReadAll(reader)
	if err != nil {
		return filepath.Dir(game.Path)
	}
	fmt.Println("ds16")
	lines := strings.Split(string(utf8Content), "\n")
	if entries != nil && len(entries) > 0 {
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "C:\\Users\\") {
				fullPathEnd := strings.IndexAny(line, "　\r\n")
				if fullPathEnd == -1 {
					fullPathEnd = len(line)
				}
				fullPath := line[:fullPathEnd]

				start := strings.Index(fullPath, "\\Users\\") + len("\\Users\\")
				end := strings.Index(fullPath[start:], "\\")
				if end == -1 {
					continue
				}
				relPath := fullPath[start+end+1:]
				homeDir, err := os.UserHomeDir()
				if err != nil {
					continue
				}
				fullpath = filepath.Join(homeDir, relPath)
				base := filepath.Base(fullpath)
				if saveFileName == base {
					fullpath = filepath.Dir(fullpath)
				}

				return fullpath
			}
		}
	}

	if fullpath == "" {
		gameDir := filepath.Dir(game.Path)
		filepath.Walk(gameDir, func(path string, info fs.FileInfo, err error) error {
			baseName := filepath.Base(path)
			if !info.IsDir() && baseName == saveFileName {
				fullpath = filepath.Dir(path)
				return errors.New("")
			}

			return nil
		})
	}

	return fullpath
}

func utf8ToShiftJIS(utf8Data string) (string, error) {
	encoder := japanese.ShiftJIS.NewEncoder()
	reader := transform.NewReader(strings.NewReader(utf8Data), encoder)
	sjisData, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(sjisData), nil
}

func (b SaveInfoGetter) FetchSeiyaSaveUrlMap(names []string) (map[string]string, error) {
	var url string = "https://seiya-saiga.com/save.html"
	c := CreateCollector("*seiya-saiga.com")

	var potentialGames []struct {
		Title string
		Link  string
	}
	results := make(map[string]string)

	// 处理搜索结果页面中的游戏条目
	c.OnHTML("div > table > tbody > tr > th table tbody tr", func(e *colly.HTMLElement) {
		titles := e.ChildTexts("td b")
		title := ""
		if len(titles) > 0 {
			title, _ = shiftJISToUTF8(titles[0])
		}

		link := e.ChildAttr("a", "href")

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
		for _, name := range names {
			gameFound, s := searchNameByRegexSimilarity(potentialGames, name, false, []string{"セット", "PSV", "PS4", "PSP", "Android"}, func(t1 struct {
				Title string
				Link  string
			}) string {
				return t1.Title
			})
			if gameFound != nil && s > 0.7 {
				results[name] = gameFound.Link
			}
		}

	})

	// 错误处理
	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Request error: %s with error: %s\n", r.Request.URL, err)
	})

	// 访问构建的 URL
	err := c.Visit(url)
	if err != nil {
		return results, fmt.Errorf("找不到游戏存档,err:%v", results, err)
	}

	// 等待收集完成
	c.Wait()

	return results, err
}

func (b SaveInfoGetter) FetchSeiyaGuide(name string, fetchType int) (models.GuideContent, error) {
	guide := models.GuideContent{}
	var url string = "https://seiya-saiga.com/game/kouryaku.html"
	c := CreateCollector("*seiya-saiga.com")

	var potentialGames []struct {
		Title string
		Link  string
	}
	gameName := ""
	gameLink := ""
	mainTitle, subtitle, _ := getTitles(name)
	firstName := mainTitle
	var sm float32 = 0
	if fetchType == 1 {
		firstName = getGameNameAlternative(firstName)
	} else if fetchType == 2 && subtitle != "" {
		firstName = subtitle
	}

	// 处理搜索结果页面中的游戏条目
	c.OnHTML("div > table > tbody > tr > th table tbody tr", func(e *colly.HTMLElement) {
		titles := e.ChildTexts("td b")
		title := ""
		if len(titles) > 0 {
			title, _ = shiftJISToUTF8(titles[0])
		}

		link := "https://seiya-saiga.com/game/" + e.ChildAttr("a", "href")

		if link == "" || title == "" || strings.Contains(link, "#") || !strings.Contains(link, "game/") {
			return
		}

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
		gameFound, s := searchNameByRegexSimilarity(potentialGames, name, false, []string{"セット", "PSV", "PS4", "PSP", "Android"}, func(t1 struct {
			Title string
			Link  string
		}) string {
			return t1.Title
		})
		if gameFound != nil && s > 0.7 {
			gameName = gameFound.Title
			gameLink = gameFound.Link
			sm = s
		}

	})

	// 错误处理
	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Request error: %s with error: %s\n", r.Request.URL, err)
	})

	// 访问构建的 URL
	err := c.Visit(url)
	if err != nil {
		return guide, fmt.Errorf("找不到游戏%s的攻略,err:%v", name, err)
	}

	// 等待收集完成
	c.Wait()
	if gameLink == "" {
		return guide, fmt.Errorf("找不到游戏%s的攻略", name)
	}

	fmt.Printf("找到游戏‘%s’的攻略%s, s:%0.2f\n", gameName, gameLink, sm)
	guide, err = b.FetchSeiyaGuideContent(gameLink)

	return guide, err
}

func (b SaveInfoGetter) FetchSeiyaGuideContent(link string) (models.GuideContent, error) {
	guide := models.GuideContent{Link: link}
	guide.Source = enums.Seiya
	var err error = nil
	c := CreateCollector("*seiya-saiga.com")

	c.OnResponse(func(r *colly.Response) {
		decoder := japanese.EUCJP.NewDecoder()

		// 转换编码
		reader := transform.NewReader(bytes.NewReader(r.Body), decoder)
		utf8Data, err := io.ReadAll(reader)
		if err != nil {
			fmt.Println("Error decoding content:", err)
		}
		decodedText := string(utf8Data)
		// decodedBody, _ := decodeJapaneseContent(r.Body, r.Headers.Get("Content-Type"))
		if !isValidJapaneseText(decodedText) {
			fmt.Println("✅ euc-jp解码无效")
			decoder = japanese.ShiftJIS.NewDecoder()
			reader = transform.NewReader(bytes.NewReader(r.Body), decoder)
			utf8Data, err = io.ReadAll(reader)

		}
		r.Body = utf8Data

	})

	c.OnHTML("body > div > table > tbody > tr", func(e *colly.HTMLElement) {
		// oHtml, err := o.DOM.Html()
		// if err != nil {
		// 	return
		// }
		// doc, err := eucToUTF8(oHtml)

		// if err != nil {
		// 	fmt.Println("Error loading HTML 00:", err)
		// }
		// doc = "<tr>" + doc + "</tr>"
		// // fmt.Println("开始获取游戏攻略内容:" + doc)
		// eDoc, err := goquery.NewDocumentFromReader(strings.NewReader(doc))
		// if err != nil {
		// 	fmt.Println("Error loading HTML:", err)
		// }

		// // e := eDoc.Find()
		// s0, err := eDoc.Html()
		// // s0, err := eDoc.Find("th").Html()

		// fmt.Printf("FetchSeiyaGuideContent 30:\n%s\n", s0)
		s := e.DOM.Find("th > table").Eq(3)
		// fmt.Printf("FetchSeiyaGuideContent 20:\n%s\n", s.Text())
		s.Find("tr").Each(func(i int, ss *goquery.Selection) {
			labels := ss.Find("label").Contents()
			fonts := ss.Find("font").Contents()
			bs := ss.Find("b").Contents()

			if labels.Length() == 0 && fonts.Length() == 0 && bs.Length() == 0 {
				ss.Remove()
			}
		})

		g, _ := s.Html()
		fmt.Printf("FetchSeiyaGuideContent 00:\n%s\n", g)
		g = fmt.Sprintf(`<table border="1" bordercolor="#66ccff" bgcolor="#ffffff" height="40" width="800" cellspacing="0">
        %s
		</table>`, g)
		// guide.Content, err = shiftJISToUTF8(hml)
		guide.Content = g
		s2 := e.DOM.Find("th > table").Eq(2)
		guide.SaveLink = s2.Find("td:contains('セーブ') a").AttrOr("href", "")
		s2.Find("tr").Each(func(i int, ss *goquery.Selection) {
			trText := ss.Text()
			if i == 0 || i == 1 || strings.Contains(trText, "攻略リンク") {
				ss.Remove()
			}
		})
		text, _ := s2.Html()
		// guide.Text, err = eucToUTF8(text)
		guide.Text = strings.ReplaceAll(strings.ReplaceAll(text, "��", ""), "フルコンプセーブ", "")

		// text, _ := shiftJISToUTF8(e.DOM.Text())
		// fmt.Printf("FetchSeiyaGuideContent:\n%s\n", guide.Text)
		// guide.Text, err = shiftJISToUTF8(e.DOM.Text())

		name := e.DOM.Find("table").Eq(0).Find("th").Text()
		guide.Name = name
		fmt.Printf("FetchSeiyaGuideContent name: %s\n", name)

	})
	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Request error: %s with error: %s\n", r.Request.URL, err)
	})

	// 访问构建的 URL
	err = c.Visit(link)
	if err != nil {
		return guide, fmt.Errorf("找不到游戏%s的攻略,err:%v", link, err)
	}

	// 等待收集完成
	c.Wait()

	return guide, err

}

func shiftJISToUTF8(shiftJISData string) (string, error) {
	// 创建 Shift-JIS 解码器
	decoder := japanese.ShiftJIS.NewDecoder()

	// 转换编码
	reader := transform.NewReader(strings.NewReader(shiftJISData), decoder)
	utf8Data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(utf8Data), nil
}

func eucToUTF8(shiftJISData string) (string, error) {
	// 创建 Shift-JIS 解码器
	decoder := japanese.EUCJP.NewDecoder()

	// 转换编码
	reader := transform.NewReader(strings.NewReader(shiftJISData), decoder)
	utf8Data, err := io.ReadAll(reader)
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
func OverrideFiles(sourcePath string, targetPath string, exePath string) error {
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
		// if err != nil {
		// 	return fmt.Errorf("查找目标文件失败: %v", err)
		// }
		if dstFilePath == "" && exePath != "" {
			dstFilePath, err = findFileInTarget(srcFileName, exePath)
		}

		// 如果未找到匹配文件，跳过
		if dstFilePath == "" {
			fmt.Printf("未找到匹配文件，直接复制: %s\n", srcFileName)
			dstFilePath = filepath.Join(targetPath, srcFileName)
			err = copyFile(srcFilePath, dstFilePath)
			return err
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
		data, err := io.ReadAll(srcFile)
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

func SearchSave(game models.Game) (models.Game, error) {
	if game.SavePath != "" {
		return game, nil
	}
	brandMap := map[string]string{
		"FrontWing":      "フロントウイング",
		"ETERNAL":        "WillPuls",
		"シルキーズSAKURA":    "SilkysSakura",
		"でぼの巣製作所":        "DebonosuWorks",
		"アストロノーツ・シリウス":   "Sirius",
		"かえるそふと":         "Kaeru Soft",
		"ミルフィーユ":         "mille-feuille",
		"GrayZone":       "My Games",
		"monoceros+黒":    "モノセロスプラス黒",
		"コンプリーツ":         "Complets",
		"calcite":        "skdata",
		"シルキーズプラスWASABI": "SilkysPlus",
		"シルキーズプラス":       "SilkysPlus",
		"えどわ～る":          "edoire",
	}
	newGame, engine, exeName, folderName, err := SearchGamePathSave(game)
	applog.InfoLogSaveAppLog("engine: %s, exeName: %s, folderName: %s", engine, exeName, folderName)

	// if err != nil && engine == "" && newGame.SavePath == "" {
	// 	return game, err
	// }
	if err == nil && newGame.SavePath != "" {
		return newGame, nil
	}
	brand := game.Company
	if brandMap[brand] != "" {
		brand = brandMap[brand]
	}
	gameName := game.Name
	savePath := searchSpSave(exeName, gameName, folderName, brand, engine, filepath.Dir(game.Path))
	applog.InfoLogSaveAppLog("SearchSave searchSpSave savePath: %s", savePath)
	if savePath != "" {
		newGame.SavePath = savePath
		return newGame, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return game, err
	}

	// searchPath := filepath.Join(homeDir, "Saved Games")
	// savePath = searchFolderSave(searchPath, exeName, gameName, brand, engine)
	// if savePath != "" {
	// 	newGame.SavePath = savePath
	// 	return newGame, nil
	// }
	searchPath := filepath.Join(homeDir, "Documents", "My Games")
	savePath = searchFolderSave(searchPath, exeName, gameName, folderName, brand, engine, false)

	applog.InfoLogSaveAppLog("SearchSave My Games savePath: %s", savePath)
	if savePath != "" {
		newGame.SavePath = savePath
		return newGame, nil
	}
	searchPath = filepath.Join(homeDir, "Documents")
	savePath = searchFolderSave(searchPath, exeName, gameName, folderName, brand, engine, false)
	applog.InfoLogSaveAppLog("SearchSave Documents savePath: %s", savePath)
	if savePath != "" {
		newGame.SavePath = savePath
		return newGame, nil
	}
	searchPath = filepath.Join(homeDir, "AppData", "Roaming")
	savePath = searchFolderSave(searchPath, exeName, gameName, folderName, brand, engine, false)
	applog.InfoLogSaveAppLog("SearchSave Roaming savePath: %s", savePath)
	if savePath != "" {
		newGame.SavePath = savePath
		return newGame, nil
	}

	searchPath = filepath.Join(homeDir, "AppData", "Local")
	savePath = searchFolderSave(searchPath, exeName, gameName, folderName, brand, engine, false)
	applog.InfoLogSaveAppLog("SearchSave Local savePath: %s", savePath)
	if savePath != "" {
		newGame.SavePath = savePath
		return newGame, nil
	}
	searchPath = filepath.Join(homeDir, "AppData", "LocalLow")
	savePath = searchFolderSave(searchPath, exeName, gameName, folderName, brand, engine, false)
	applog.InfoLogSaveAppLog("SearchSave LocalLow savePath: %s", savePath)
	if savePath != "" {
		newGame.SavePath = savePath
		return newGame, nil
	}
	searchPath = filepath.Join(homeDir, "Documents")
	savePath = searchGameSave(searchPath, exeName, gameName, folderName, engine, 0.5)
	applog.InfoLogSaveAppLog("SearchSave Documents savePath: %s", savePath)
	if savePath != "" {
		newGame.SavePath = savePath
		return newGame, nil
	}
	if savePath == "" {
		return newGame, errors.New("no save path found")
	}

	return newGame, err
}

func searchFolderSave(searchPath, exeName, gameName, folderName, brand, engine string, mustInFolder bool) string {
	savePath := searchFolderSaveFolder(searchPath, exeName, gameName, folderName, brand, engine, mustInFolder)
	if savePath != "" {
		return savePath
	}
	savePath = searchGameSave(searchPath, exeName, gameName, folderName, engine, 0.5)
	applog.InfoLogSaveAppLog("SearchSave folder searchGameSave savePath: %s, gameName: %s, folderName: %s, exeName: %s, engine: %s, searchPath: %s", savePath, gameName, folderName, exeName, engine, searchPath)
	return savePath
}

func searchFolderSaveFolder(searchPath, exeName, gameName, folderName, brand, engine string, mustInFolder bool) string {
	brandEntries, err := os.ReadDir(searchPath)
	if err != nil {
		return ""
	}

	brandEntry, s := searchNameByRegexSimilarity(brandEntries, brand, false, []string{}, func(entry os.DirEntry) string {
		return entry.Name()
	})
	if brandEntry != nil {
		brandFound := strings.ToLower((*brandEntry).Name())
		bfc := len([]rune(brandFound))
		// bc := len([]rune(brand))
		re := regexp.MustCompile(`^[\x00-\x7F]*$`)
		if bfc < 6 && re.MatchString(brand) && re.MatchString(brandFound) &&
			!strings.Contains(strings.ToLower(brand), brandFound) {
			s -= 0.2
		}
	}
	if brandEntry == nil || s < 0.7 {
		return ""
	} else {

	}
	gameEntry := searchGameSave(filepath.Join(searchPath, (*brandEntry).Name()), exeName, gameName, folderName, engine, s)
	if gameEntry == "" {
		return ""
	}

	return gameEntry

}

/**
 * 返回的是完整路径
 */
func searchGameSave(searchPath, exeName, gameName, folderName, engine string, lastSimilarity float32) string {
	fmt.Printf("searchGameSave folder:%s, s:%.4f\n", searchPath, lastSimilarity)
	gameEntries, err := os.ReadDir(searchPath)
	if err != nil {
		return ""
	}

	gameEntry, s := searchNameByRegexSimilarity(gameEntries, gameName, false, []string{}, func(entry os.DirEntry) string {
		return entry.Name()
	})
	if gameEntry != nil {
		gameEntryName := (*gameEntry).Name()
		if s > 0.7 || strings.Contains(gameEntryName, gameName) || strings.Contains(gameName, gameEntryName) {
			return filepath.Join(searchPath, (*gameEntry).Name())
		}

	}
	gameEntry, s = searchNameByRegexSimilarity(gameEntries, folderName, false, []string{}, func(entry os.DirEntry) string {
		return entry.Name()
	})
	if gameEntry != nil {
		if s > 0.7 {
			return filepath.Join(searchPath, (*gameEntry).Name())
		}

	}
	eName := strings.TrimSuffix(exeName, filepath.Ext(exeName))
	gameEntry, s = searchNameByRegexSimilarity(gameEntries, eName, false, []string{}, func(entry os.DirEntry) string {
		return entry.Name()
	})
	if gameEntry != nil && s > 0.6 || len(gameEntries) < 10 {
		if s < 0.1 {
			return ""
		} else if lastSimilarity < 0.84 {
			if s > 0.8 || (s > 0.7 && lastSimilarity > 0.5) {
				return filepath.Join(searchPath, (*gameEntry).Name())
			}
		} else {
			if s > 0.5 && len(gameEntries) < 10 || s > 0.6 {
				return filepath.Join(searchPath, (*gameEntry).Name())
			}
		}
	}

	return ""
}

func searchSpSave(exeName, gameName, folderName, brand, engine, exePath string) string {
	rs := ""

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	if engine == "willplus" {
		rs = filepath.Join(homeDir, "Saved Games", "WillPlus")
		path := searchGameSave(rs, exeName, gameName, folderName, engine, 0.8)
		if path != "" {
			return path
		}
		rs = filepath.Join(homeDir, "AppData", "Roaming", "WillPlus")
		return searchFolderSave(rs, exeName, gameName, folderName, brand, engine, true)
	}
	if engine == "unity" {
		rs = filepath.Join(homeDir, "AppData", "LocalLow")
		return searchFolderSave(rs, exeName, gameName, folderName, brand, engine, true)
	}
	if engine == "gxp" {
		rs = filepath.Join(homeDir, "Documents", "Astronauts Sirius")
		return searchGameSave(rs, exeName, gameName, folderName, engine, 1)
	}
	if engine == "odn" {
		rs = filepath.Join(exePath, "data")
		return rs
	}
	if engine == "npf" {
		return exePath
	}
	if engine == "fpk" {
		return exePath
	}
	if engine == "DefaultCompany" {
		rs = filepath.Join(homeDir, "AppData", "LocalLow", "DefaultCompany")
		return searchGameSave(rs, exeName, gameName, folderName, engine, 1)
	}
	lowBrand := strings.ToLower(brand)
	if strings.Contains(lowBrand, "anim") && !strings.Contains(lowBrand, "anime") {
		return exePath
	}
	if brand == "玉藻ソフト" {
		return filepath.Join(homeDir, "Documents", "My Games")
	}

	return ""
}

func SearchGamePathSave(game models.Game) (models.Game, string, string, string, error) {
	var err error = nil

	var engine string
	var exeName string = filepath.Base(game.Path)
	fns := []string{"datasu.ksd", "main.ev"}
	exts := []string{".sav", ".gdb"}
	pexts := []string{".sav", ".gdb", ".bin", ".dat"}
	folders := []string{"save", "userdata", "savedata", "sav", "セーブデータ"}
	foldersExclude := []string{"saveloadimage", "menu", "thumsave", "quicksavedata"}
	var foundSaveFile string
	var foundPath string
	folderPath := filepath.Dir(game.Path)
	folderName := filepath.Base(folderPath)
	err = filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 忽略无法访问的路径
		}

		lowPath := strings.ToLower(path)

		fmt.Printf("search save folder:%s\n", lowPath)
		for _, f := range foldersExclude {
			if strings.Contains(lowPath, f) {
				return nil
			}
		}
		if !info.IsDir() {
			// 检查文件扩展名是否为视频格式
			ext := strings.ToLower(filepath.Ext(path))
			oFileName := filepath.Base(path)
			fileName := strings.ToLower(oFileName)
			fName := strings.TrimSuffix(fileName, ext)
			if strings.Contains(path, "AdvHD.exe") {
				engine = "willplus"
				return errors.New("found")
			}
			if strings.Contains(path, ".odn") {
				engine = "odn"
				return errors.New("found")
			}
			if strings.Contains(path, ".npf") {
				engine = "npf"
				return errors.New("found")
			}
			if strings.Contains(path, ".gxp") {
				engine = "gxp"
				return errors.New("found")
			}
			if strings.Contains(path, ".fpk") {
				engine = "fpk"
				return errors.New("found")
			}
			if strings.Contains(path, "player_win_x86.pdb") {
				engine = "DefaultCompany"
				return errors.New("found")
			}
			for _, e := range exts {
				if ext == e {
					foundSaveFile = path
					return nil
				}
			}
			fmt.Printf("search save filename:%s\n", fileName)
			if fName == "sav" || strings.Contains(fName, "save") {
				if ArrayContains(pexts, ext) {
					foundSaveFile = path
					return errors.New("found")
				}

			}
			if strings.Contains(path, "xp3") {
				engine = "krkr2"
				return nil
				//krkr2
			}
			if strings.Contains(path, "UnityCrashHandler64") {
				engine = "unity"
				return nil
			}

			if strings.Contains(path, "\\BGI.") {
				engine = "bgi"
				foundSaveFile = path
				return errors.New("found")
			}
			for _, fn := range fns {
				if strings.Contains(path, fn) {
					foundSaveFile = path
					return errors.New("found")
				}
			}

		} else {
			for _, f := range folders {
				if strings.Contains(lowPath, f) {
					foundPath = path
					return errors.New("found")
				}
			}
		}
		return nil
	})
	if err != nil && err.Error() == "found" {
		err = nil
	}
	if foundSaveFile != "" && foundPath == "" {
		foundPath = filepath.Dir(foundSaveFile)
	}
	if foundPath == "" {
		return game, engine, exeName, folderName, errors.New("no save path found")
	}
	fmt.Printf("search save foundpath:%s\n", foundPath)
	game.SavePath = foundPath
	return game, engine, exeName, folderName, err
}

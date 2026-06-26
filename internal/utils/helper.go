package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"lunabox/internal/models"
	"lunabox/internal/vo"
	"regexp"
	"sort"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/hbollon/go-edlib"
)

var (
	ExcludeExeKeywords []string = []string{
		"courier_i", "courier", "acmp", "curl", "unitycrashhandler64", "krkr", "krkrconf", "krkrfont", "krkrlt", "krkrrel", "krkrsign", "krkrtpc", "tcwfcomp",
		"settings", "setting", "python", "protect", "instx86", "instx64", "install", "inst", "config2", "autorun", "supporttools", "filechecker",
		"uninstall_x86", "uninst64", "uninst32", "uninst", "unins003", "unins002", "unins001", "unins000", "uinst", "bhvc", "unitycrashhandler32",
		"vcredist_x86", "vcredist_x64", "vc_redist.x86", "updchk", "upgrade", "uninstx86", "uninstx64", "uninstcl", "uninstaller", "tracelog",
		"unins", "setup", "config", "patch", "update", "crashpad", "ファイル破損チェックツール", "システム詳細設定", "エンジン設定",
		"vc_redist", "dxwebsetup", "directx", "vcredist", "dotnet", "_uninst", "セーブデータ場所設定ツール", "システム設定",
		"redistributable", "installer", "launcher_helper", "crashreporter", "ファイル破損チェック", "セーブデータフォルダを開く",
		"updater", "uninstall", "删除", "卸载", "syscfg", "ihs", "configure", "セーブファイル設定", "セーブデータフォルダ・開く",
	}
)

func MergeStrings(tagStr1, tagStr2 string) string {
	if tagStr1 == "" {
		return tagStr2
	}
	if tagStr2 == "" {
		return tagStr1
	}
	// 分割字符串为数组
	tags1 := strings.Split(tagStr1, ",")
	tags2 := strings.Split(tagStr2, ",")

	// 使用map去重
	tagSet := make(map[string]struct{})

	for _, tags := range [][]string{tags1, tags2} {
		for _, tag := range tags {
			trimmedTag := strings.TrimSpace(tag)
			if trimmedTag != "" {
				tagSet[trimmedTag] = struct{}{}
			}
		}
	}

	// 转换为结果数组
	uniqueTags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		uniqueTags = append(uniqueTags, tag)
	}

	// 重新组合为字符串
	return strings.Join(uniqueTags, ",")
}

func PrintJsonLog(title string, data interface{}) {
	jsonStr, _ := json.Marshal(data)
	fmt.Printf(title + " json: " + string(jsonStr))
}

func GetReqEntity(g *models.Game) vo.MetadataRequest {
	return vo.MetadataRequest{
		ID:       g.SourceID,
		Source:   g.SourceType,
		DbGameId: g.ID,
	}
}

func ArrayToMap[T1 any, T2 comparable](slice []T1, fn func(t1 T1) T2) map[T2][]T1 {
	result := make(map[T2][]T1)
	for _, item := range slice {
		var key T2 = fn(item)
		result[key] = append(result[key], item)
	}
	return result
}

func MapToArray[T1 any, T2 comparable](m map[T2][]T1) []T1 {
	var result []T1 = []T1{}
	for _, item := range m {
		for _, item2 := range item {
			result = append(result, item2)
		}
	}
	return result
}

func JoinString[T any](slice []T, divider string, fn func(t T) string) string {
	var rs = ""
	for i, item := range slice {
		if i > 0 {
			rs += divider
		}
		rs += fn(item)
	}
	return rs
}

func FindMatch[T1 any, T2 any](slice1 []T1, slice2 []T2, fn func(t1 T1, t2 T2) bool) *T1 {
	if len(slice2) == 0 && len(slice1) > 0 {
		return &slice1[0]
	}
	for _, item1 := range slice1 {
		for _, item2 := range slice2 {
			if fn(item1, item2) {
				return &item1
			}
		}
	}
	return nil
}

func Find[T1 any](slice1 []T1, fn func(t1 T1) bool) *T1 {
	for _, item1 := range slice1 {
		if fn(item1) {
			return &item1
		}
	}
	return nil
}

func Contains[T1 any](slice1 []T1, fn func(t1 T1) bool) bool {
	for _, item1 := range slice1 {
		if fn(item1) {
			return true
		}
	}
	return false
}

func Filter[T1 any](slice1 []T1, fn func(t1 T1) bool) []T1 {
	var rs []T1 = []T1{}
	for _, item1 := range slice1 {
		if fn(item1) {
			rs = append(rs, item1)
		}
	}
	return rs
}

func MapArray[T1 any, T2 any](slice1 []T1, fn func(t1 T1) T2) []T2 {
	var result []T2
	for _, item1 := range slice1 {
		result = append(result, fn(item1))
	}
	return result
}

func setupTestContext() context.Context {
	// 创建一个简单的context，避免调用Wails runtime
	return context.WithValue(context.Background(), "test_mode", true)
}

func isValidDateFormat(v string) bool {
	// 定义正则表达式模式：YYYY年M月D日
	pattern := `^\d{4}年(0?[1-9]|1[0-2])月(0?[1-9]|[12]\d|3[01])日$`

	// 编译正则表达式
	re := regexp.MustCompile(pattern)

	// 使用正则表达式匹配字符串
	return re.MatchString(v)
}

func generateSearchRegex(searchName string) string {
	// 检查是否包含有效的分隔符（除了纯空格）
	hasValidSeparator := regexp.MustCompile(`[－\-~～]`).MatchString(searchName)

	if !hasValidSeparator {
		// 没有有效分隔符，整个字符串作为主标题处理
		mainTitle := regexp.QuoteMeta(strings.TrimSpace(searchName))
		return fmt.Sprintf(`^.*?%s.*$`, mainTitle)
	}

	// 有有效分隔符，尝试分离主标题和副标题
	// 使用非空格分隔符进行分割
	separatorPattern := `[－\-~～]+`
	parts := regexp.MustCompile(separatorPattern).Split(searchName, -1)

	if len(parts) >= 2 {
		// 成功分离出主标题和副标题
		mainTitle := strings.TrimSpace(parts[0])
		subTitle := strings.TrimSpace(parts[1])

		if mainTitle != "" && subTitle != "" {
			// 都不为空，构建成对匹配
			mainQuoted := regexp.QuoteMeta(mainTitle)
			subQuoted := regexp.QuoteMeta(subTitle)
			// 修复：使用单个+而不是++
			sepPattern := `[－\-~～\s]*` // 改为*表示可选的分隔符

			return fmt.Sprintf(
				`^.*?%s%s%s.*$`,
				mainQuoted,
				sepPattern,
				subQuoted,
			)
		}
	}

	// 默认情况：整个字符串作为主标题
	mainTitle := regexp.QuoteMeta(strings.TrimSpace(searchName))
	return fmt.Sprintf(`^.*?%s.*$`, mainTitle)
}

/**
 * 截取最后一个数字字符，包括全角
 */
func extractLastNumberFromString(titleN string) (t string, n string) {
	title := strings.TrimSpace(titleN)
	var rsTitle string = ""
	var rsStr = ""

	if len(title) == 0 {
		return title, ""
	}

	// 将字符串转换为 rune 数组以正确处理多字节字符
	runes := []rune(title)
	lastChar := runes[len(runes)-1]
	lastCharStr := string(lastChar)

	// fmt.Printf("extractLastNumberFromString title:%s char:%c\n", title, lastChar)

	// 使用 regexp2 支持 \u 转义
	digitPattern := regexp2.MustCompile(`^[\u0030-\u0039\uFF10-\uFF19弐壱参]$`, 0)

	match, _ := digitPattern.MatchString(lastCharStr)
	if match {
		rsStr = lastCharStr
	}

	if rsStr == "" {
		return title, "" // 没有找到数字
	}

	// 截取最后一个字符之前的部分
	rsTitle = string(runes[:len(runes)-1])

	fmt.Printf("extractLastNumberFromString num found title:%s char:%c\n", rsTitle, lastChar)
	return rsTitle, rsStr
}

func getTitles(searchName string) (mainT string, subT string, number string) {
	return getTitlesNum(searchName, false)

}

func getTitlesNum(searchName string, onlyNum bool) (mainT string, subT string, number string) {
	// 检查是否包含有效的分隔符（除了纯空格）
	// num := -1
	// numStr := ""
	hasValidSeparator := regexp.MustCompile(`[－\-~～　＝ ・！：─―_!「\[]`).MatchString(searchName)

	mainTitle := ""

	if !hasValidSeparator {
		// 没有有效分隔符，整个字符串作为主标题处理
		mainTitle = regexp.QuoteMeta(strings.TrimSpace(searchName))
		mainTitle, numStr := extractLastNumberFromString(mainTitle)
		return handleMainTitle(mainTitle), "", numStr
	}

	// 有有效分隔符，尝试分离主标题和副标题
	// 使用非空格分隔符进行分割
	hasNonEnglish := regexp.MustCompile(`[^a-zA-Z0-9 ]`).MatchString(searchName)
	separatorPattern := `[－\-~～_]+`
	if hasNonEnglish {
		// separatorPattern = `[－・「\-~ ～]+`
		separatorPattern = `[－\-~～＝： ・_!─―「]+`
	}
	parts := regexp.MustCompile(separatorPattern).Split(searchName, -1)
	// fmt.Printf("getTitlesNum 11 :%s\n", strings.Join(parts, ","))
	if len(parts) > 1 {
		mainTitle = strings.TrimSpace(parts[0])

		// fmt.Printf("getTitlesNum 01:%s\n", mainTitle)
	}
	if mainTitle == "" {
		mainTitle = searchName
	}
	if len(parts) < 2 {
		hasNonEnglish = regexp.MustCompile(`[^a-zA-Z0-9 ]`).MatchString(mainTitle)
		if onlyNum || hasNonEnglish {
			separatorPattern = `[－\-~～　＝ ！・─「\[]+`
			// fmt.Printf("getTitlesNum 02:\n")
		} else {
			separatorPattern = `[－\-~～　！\[]+`
			// fmt.Printf("getTitlesNum 03:\n")
		}

		parts = regexp.MustCompile(separatorPattern).Split(searchName, -1)

	}
	// fmt.Printf("getTitlesNum 12 :%s\n", strings.Join(parts, ","))

	if len(parts) >= 2 {
		// 成功分离出主标题和副标题
		mainTitle = strings.TrimSpace(parts[0])
		subTitle := strings.TrimSpace(parts[1])
		numStr := ""
		if mainTitle != "" && subTitle != "" {
			for i, txt := range parts {
				if i == 0 {
					mainTitle, numStr = extractLastNumberFromString(txt)
					// fmt.Printf("getTitlesNum 04:\n")
					if numStr != "" {

						return handleMainTitle(mainTitle), subTitle, numStr
					}
				} else if i == 1 {
					subTitle, numStr = extractLastNumberFromString(txt)

					if numStr != "" {
						fmt.Printf("getTitlesNum 05:\n")
						return handleMainTitle(mainTitle), subTitle, numStr
					}
				} else {
					_, numStr = extractLastNumberFromString(txt)
					if numStr != "" {
						fmt.Printf("getTitlesNum 06:\n")
						return handleMainTitle(mainTitle), subTitle, numStr
					}
				}

			}
			if re, err := regexp.MatchString(`^[a-zA-Z]{1,4}$`, mainTitle); err == nil && re && subTitle != "" {
				mainTitle = subTitle
			}
			return handleMainTitle(mainTitle), subTitle, numStr
		}
		if mainTitle == "" && subTitle != "" {
			return handleMainTitle(subTitle), "", ""
		}
	}

	// 默认情况：整个字符串作为主标题
	mainTitle = regexp.QuoteMeta(strings.TrimSpace(searchName))
	mainTitle, numStr := extractLastNumberFromString(mainTitle)

	return handleMainTitle(mainTitle), "", numStr
}

func handleMainTitle(mainTitle string) string {
	title := mainTitle
	lowTitle := strings.ToLower(mainTitle)
	if strings.Contains(lowTitle, "chapter") {
		parts := strings.Split(lowTitle, "chapter")
		if len(parts) > 0 {
			title = strings.TrimSpace(parts[0])
		}
	}
	return title
}

func stringCharSplit(str string, split string) []string {
	rs := []string{}
	text := []rune{}
	// rnSplit := []rune(split)
	for _, r := range str {
		if strings.Contains(split, string(r)) {
			if len(text) > 0 {
				rs = append(rs, string(text))
			}
		} else {
			text = append(text, r)
		}
	}
	return rs

}

func searchByRegex[T1 any](slice1 []T1, pattern string, searchName string, excludeWords []string, fn func(t1 T1) string) []T1 {
	if len(slice1) == 0 {
		return nil
	}
	re := regexp.MustCompile(pattern)
	var result []T1 = []T1{}
	for _, item := range slice1 {
		target := fn(item)
		fmt.Printf("searchByRegex 01 target:%s, data:%v\n", target, item)
		if re.MatchString(target) {
			isExcluded := false
			for _, word := range excludeWords {
				if strings.Contains(target, word) {
					isExcluded = true
					break
				}
			}
			fmt.Printf("target:%s, isexcluded:%v, data:%v\n", target, isExcluded, item)
			if !isExcluded {
				result = append(result, item)
			}
		} else if strings.Contains(target, strings.ReplaceAll(searchName, "？", "")) {
			fmt.Printf("searchByRegex 02 target:%s\n", target)
			isExcluded := false
			for _, word := range excludeWords {
				if strings.Contains(target, word) {
					isExcluded = true
					break
				}
			}
			fmt.Printf("target:%s, 02 isexcluded:%v\n", target, isExcluded)
			if !isExcluded {
				result = append(result, item)
			}
		}
	}
	if len(result) == 0 {
		// sort.Slice(slice1, func(i, j int) bool {
		// 	return len(fn(slice1[i])) < len(fn(slice1[j]))
		// })
		// result = []T1{}
		// result = append(result, slice1[0])
		return slice1
	}
	return result
}

// func getTransmittedMainTitle(searchName string) string {
// 	runes := []rune(searchName)
// 	titleRunes := []rune{}
// 	lastLetterType := 0 //0半角英文字母 1全角英字 2半角数字 3全角数字 4汉字 5片假名 6平假名 7半角或全角控股 8其他
// 	for i := 0; i < len(runes); i++ {
// 		if i >= 4 {

// 		} else {
// 			titleRunes = append(titleRunes, runes[i])
// 		}

// 	}
// 	return string(titleRunes)
// }

// ... existing code ...

func getTransmittedMainTitle(searchName string) string {
	runes := []rune(searchName)
	titleRunes := []rune{}
	lastLetterType := 0 //0半角英文字母 1全角英文字 2半角数字 3全角数字 4汉字 5片假名 6平假名 7半角或全角空格 8其他

	for i := 0; i < len(runes); i++ {
		currentChar := runes[i]
		currentType := getCharType(currentChar)

		if len(titleRunes) > 10 {
			if len(titleRunes) > 0 {
				lastChar := titleRunes[len(titleRunes)-1]
				lastType := getCharType(lastChar)

				if (lastType == 0 && currentType == 7) || (lastType == 7 && currentType == 0) {
					break
				}
			}
		}

		if len(titleRunes) > 5 && lastLetterType != 0 && currentType != lastLetterType {
			break
		}

		titleRunes = append(titleRunes, currentChar)
		if currentType != 7 {
			lastLetterType = currentType
		}
	}

	return string(titleRunes)
}

func getCharType(r rune) int {
	if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
		return 0 // 半角英文字母
	}
	if (r >= 'ａ' && r <= 'ｚ') || (r >= 'Ａ' && r <= 'Ｚ') {
		return 1 // 全角英文字（实际使用中半角更常见）
	}
	if r >= '0' && r <= '9' {
		return 2 // 半角数字
	}
	if r >= '０' && r <= '９' {
		return 3 // 全角数字
	}
	if r >= '\u4e00' && r <= '\u9fff' {
		return 4 // 汉字
	}
	if r >= '\u30a0' && r <= '\u30ff' {
		return 5 // 片假名
	}
	if r >= '\u3040' && r <= '\u309f' {
		return 6 // 平假名
	}
	if r == ' ' || r == '\u3000' {
		return 7 // 半角或全角空格
	}
	return 8 // 其他
}

// ... existing code ...

func getGameNameAlternative(searchName string) string {

	var name = searchName
	re := regexp.MustCompile(`（.*?）`)
	name = re.ReplaceAllString(name, "")
	for strings.HasSuffix(name, "〇") {
		name = strings.TrimSuffix(name, "〇")
	}
	for strings.HasSuffix(name, "！") {
		name = strings.TrimSuffix(name, "！")
	}

	// fmt.Printf("try00 IsCamelCase\n")
	if IsCamelCase(name) {
		// fmt.Printf("try IsCamelCase\n")
		return CamelCaseToSpaces(name)
	}
	if strings.Contains(name, "／") {
		name = strings.ReplaceAll(name, "／", "/")
	}
	if strings.Contains(name, "１") {
		name = strings.ReplaceAll(name, "１", "1")
	}
	if strings.Contains(name, "２") {
		name = strings.ReplaceAll(name, "２", "2")
	}
	if strings.Contains(name, "３") {
		name = strings.ReplaceAll(name, "３", "3")
	}
	if strings.Contains(name, "４") {
		name = strings.ReplaceAll(name, "４", "4")
	}
	if strings.Contains(name, "５") {
		name = strings.ReplaceAll(name, "５", "5")
	}
	if strings.Contains(name, "６") {
		name = strings.ReplaceAll(name, "６", "6")
	}
	if strings.Contains(name, "７") {
		name = strings.ReplaceAll(name, "７", "7")
	}
	if strings.Contains(name, "８") {
		name = strings.ReplaceAll(name, "８", "8")
	}
	if strings.Contains(name, "９") {
		name = strings.ReplaceAll(name, "９", "9")
	}
	if strings.Contains(name, "０") {
		name = strings.ReplaceAll(name, "０", "0")
	}
	if strings.Contains(name, "Ａ") {
		name = strings.ReplaceAll(name, "Ａ", "A")
	}
	if strings.Contains(name, "Ｂ") {
		name = strings.ReplaceAll(name, "Ｂ", "B")
	}
	if strings.Contains(name, "Ｃ") {
		name = strings.ReplaceAll(name, "Ｃ", "C")
	}
	if strings.Contains(name, "Ｄ") {
		name = strings.ReplaceAll(name, "Ｄ", "D")
	}
	if strings.Contains(name, "Ｅ") {
		name = strings.ReplaceAll(name, "Ｅ", "E")
	}
	if strings.Contains(name, "Ｆ") {
		name = strings.ReplaceAll(name, "Ｆ", "F")
	}
	if strings.Contains(name, "Ｇ") {
		name = strings.ReplaceAll(name, "Ｇ", "G")
	}
	if strings.Contains(name, "Ｈ") {
		name = strings.ReplaceAll(name, "Ｈ", "H")
	}
	if strings.Contains(name, "Ｉ") {
		name = strings.ReplaceAll(name, "Ｉ", "I")
	}
	if strings.Contains(name, "Ｊ") {
		name = strings.ReplaceAll(name, "Ｊ", "J")
	}
	if strings.Contains(name, "Ｋ") {
		name = strings.ReplaceAll(name, "Ｋ", "K")
	}
	if strings.Contains(name, "Ｌ") {
		name = strings.ReplaceAll(name, "Ｌ", "L")
	}
	if strings.Contains(name, "Ｍ") {
		name = strings.ReplaceAll(name, "Ｍ", "M")
	}
	if strings.Contains(name, "М") {
		name = strings.ReplaceAll(name, "М", "M")
	}
	if strings.Contains(name, "Ｎ") {
		name = strings.ReplaceAll(name, "Ｎ", "N")
	}
	if strings.Contains(name, "Ｏ") {
		name = strings.ReplaceAll(name, "Ｏ", "O")
	}
	if strings.Contains(name, "Ｐ") {
		name = strings.ReplaceAll(name, "Ｐ", "P")
	}
	if strings.Contains(name, "Ｑ") {
		name = strings.ReplaceAll(name, "Ｑ", "Q")
	}
	if strings.Contains(name, "Ｒ") {
		name = strings.ReplaceAll(name, "Ｒ", "R")
	}

	if strings.Contains(name, "Ｓ") {
		name = strings.ReplaceAll(name, "Ｓ", "S")
	}
	if strings.Contains(name, "Ｔ") {
		name = strings.ReplaceAll(name, "Ｔ", "T")
	}
	if strings.Contains(name, "Ｕ") {
		name = strings.ReplaceAll(name, "Ｕ", "U")
	}
	if strings.Contains(name, "Ｖ") {
		name = strings.ReplaceAll(name, "Ｖ", "V")
	}
	if strings.Contains(name, "Ｘ") {
		name = strings.ReplaceAll(name, "Ｘ", "X")
	}

	if strings.Contains(name, "Ｗ") {
		name = strings.ReplaceAll(name, "Ｗ", "W")
	}
	if strings.Contains(name, "Ｙ") {
		name = strings.ReplaceAll(name, "Ｙ", "Y")
	}
	if strings.Contains(name, "Ｚ") {
		name = strings.ReplaceAll(name, "Ｚ", "Z")
	}
	if strings.Contains(name, "∧") {
		name = strings.ReplaceAll(name, "∧", "A")
	}
	if strings.Contains(name, "’") {
		name = strings.ReplaceAll(name, "’", "'")
	}
	if strings.Contains(name, "ａ") {
		name = strings.ReplaceAll(name, "ａ", "a")
	}
	if strings.Contains(name, "ｂ") {
		name = strings.ReplaceAll(name, "ｂ", "b")
	}
	if strings.Contains(name, "ｃ") {
		name = strings.ReplaceAll(name, "ｃ", "c")
	}
	if strings.Contains(name, "ｄ") {
		name = strings.ReplaceAll(name, "ｄ", "d")
	}
	if strings.Contains(name, "ｅ") {
		name = strings.ReplaceAll(name, "ｅ", "e")
	}
	if strings.Contains(name, "ｆ") {
		name = strings.ReplaceAll(name, "ｆ", "f")
	}
	if strings.Contains(name, "ｇ") {
		name = strings.ReplaceAll(name, "ｇ", "g")
	}
	if strings.Contains(name, "ｈ") {
		name = strings.ReplaceAll(name, "ｈ", "h")
	}
	if strings.Contains(name, "ｉ") {
		name = strings.ReplaceAll(name, "ｉ", "i")
	}
	if strings.Contains(name, "ｊ") {
		name = strings.ReplaceAll(name, "ｊ", "j")
	}
	if strings.Contains(name, "ｋ") {
		name = strings.ReplaceAll(name, "ｋ", "k")
	}
	if strings.Contains(name, "ｌ") {
		name = strings.ReplaceAll(name, "ｌ", "l")
	}
	if strings.Contains(name, "ｍ") {
		name = strings.ReplaceAll(name, "ｍ", "m")
	}
	if strings.Contains(name, "ｎ") {
		name = strings.ReplaceAll(name, "ｎ", "n")
	}
	if strings.Contains(name, "ｏ") {
		name = strings.ReplaceAll(name, "ｏ", "o")
	}
	if strings.Contains(name, "ｐ") {
		name = strings.ReplaceAll(name, "ｐ", "p")
	}
	if strings.Contains(name, "ｑ") {
		name = strings.ReplaceAll(name, "ｑ", "q")
	}
	if strings.Contains(name, "ｒ") {
		name = strings.ReplaceAll(name, "ｒ", "r")
	}
	if strings.Contains(name, "ｓ") {
		name = strings.ReplaceAll(name, "ｓ", "s")
	}
	if strings.Contains(name, "ｔ") {
		name = strings.ReplaceAll(name, "ｔ", "t")
	}
	if strings.Contains(name, "ｕ") {
		name = strings.ReplaceAll(name, "ｕ", "u")
	}
	if strings.Contains(name, "ｖ") {
		name = strings.ReplaceAll(name, "ｖ", "v")
	}
	if strings.Contains(name, "ｗ") {
		name = strings.ReplaceAll(name, "ｗ", "w")
	}
	if strings.Contains(name, "ｘ") {
		name = strings.ReplaceAll(name, "ｘ", "x")
	}
	if strings.Contains(name, "ｙ") {
		name = strings.ReplaceAll(name, "ｙ", "y")
	}
	if strings.Contains(name, "ｚ") {
		name = strings.ReplaceAll(name, "ｚ", "z")
	}

	if strings.Contains(name, "III") {
		name = strings.ReplaceAll(name, "III", "3")
	}
	if strings.Contains(name, "II") {
		name = strings.ReplaceAll(name, "II", "2")
	}
	if strings.Contains(name, "Ⅱ") {
		name = strings.ReplaceAll(name, "Ⅱ", "2")
	}

	if strings.Contains(name, "＋") {
		name = strings.ReplaceAll(name, "＋", "+")
	}
	if strings.Contains(name, "＊") {
		name = strings.ReplaceAll(name, "＊", "*")
	}
	// if strings.Contains(name, "学") {
	// 	name = strings.ReplaceAll(name, "学", "學")
	// }
	if strings.Contains(name, "师") {
		name = strings.ReplaceAll(name, "师", "師")
	}

	name = strings.TrimSuffix(name, "％")
	// fmt.Printf("getGameNameAlternative:%s\n", name)
	return name
}

func searchNameByRegex[T1 any](slice1 []T1, searchNameO string, excludeWords []string, fn func(t1 T1) string) *T1 {
	found, _ := searchNameByRegexSimilarity(slice1, searchNameO, true, excludeWords, fn)
	return found
}

func searchNameByRegexSimilarity[T1 any](slice1 []T1, searchNameO string, hasLog bool, excludeWords []string, fn func(t1 T1) string) (*T1, float32) {
	searchName := getGameNameAlternative(searchNameO)
	searchName = strings.ToLower(searchName)
	mainTitle, _, num := getTitlesNum(searchName, true)
	// if num != "" {
	// 	num = getGameNameAlternative(num)
	// }
	fmt.Printf("searchNameByRegex 00:%s, num:%s, alnum:%s\n", searchName, num, getGameNameAlternative(num))
	type Result struct {
		similarity float32
		Value      T1
	}
	if len(slice1) == 0 {
		return nil, 0
	}
	var results []Result
	for _, item := range slice1 {
		oname := fn(item)
		oname2 := getGameNameAlternative(oname)
		name := strings.ToLower(oname2)
		// fmt.Printf("searchNameByRegex 10 name:%s, oname:%s, oname2:%s, searchName:%s\n", name, oname, oname2, searchName)
		if name == searchName {
			return &item, 1
		}
		containsExcludeWords := false
		for _, word := range excludeWords {
			if strings.Contains(name, strings.ToLower(word)) {
				containsExcludeWords = true
				break
			}
		}
		if containsExcludeWords {
			continue
		}
		similarity := edlib.JaroWinklerSimilarity(searchName, name)
		if num != "" {
			if strings.Contains(name, num) || strings.Contains(name, getGameNameAlternative(num)) {
				fmt.Printf("num is same : %s, name:%s\n", num, name)
				similarity += 0.1
			}
		}
		gMainTitle, _, gameNum := getTitlesNum(name, true)
		if num == "" {

		}
		if gameNum != "" {
			// fmt.Printf("num cannot find 00 : %v, al:%v\n", strings.Contains(name, gameNum), strings.Contains(name, getGameNameAlternative(gameNum)))
			if !strings.Contains(searchName, gameNum) && !strings.Contains(searchName, getGameNameAlternative(gameNum)) {
				fmt.Printf("num cannot find : %s, name:%s\n", num, name)
				similarity -= 0.1
			}
		}
		similarity += 0.2 * edlib.JaroWinklerSimilarity(mainTitle, gMainTitle)
		if strings.Contains(searchName, "シナリオ") && strings.Contains(name, "シナリオ") {
			similarity += 0.1
		}
		if !strings.Contains(searchName, "シナリオ") && strings.Contains(name, "シナリオ") {
			similarity -= 0.1
		}
		if strings.Contains(searchName, "DLC") && strings.Contains(name, "DLC") {
			similarity += 0.1
		}
		if strings.Contains(searchName, "アフター") && strings.Contains(name, "アフター") {
			similarity += 0.1
		}
		if !strings.Contains(searchName, "アフター") && strings.Contains(name, "アフター") {
			similarity -= 0.1
		}
		if IsEnglishWords(name) && IsEnglishWords(searchName) && (!ContainsAtLeastOneWordInStr(name, searchNameO) && !ContainsAtLeastOneWordInStr(searchName, name)) {
			similarity -= 0.4
		}
		if true || hasLog {
			fmt.Printf("searchNameByRegex title: %s, similarity:%0.4f, num:%s, searchname: %s \n", name, similarity, gameNum, searchNameO)
		}
		results = append(results, Result{similarity: similarity, Value: item})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].similarity > results[j].similarity
	})
	if len(results) == 0 {
		return nil, 0
	}
	return &results[0].Value, results[0].similarity
}

func searchNameByRegex2[T1 any](slice1 []T1, searchName string, excludeWords []string, fn func(t1 T1) string) *T1 {
	mainTitle, _, num := getTitles(searchName)
	fmt.Printf("searchName:%s, mainTitle:%s\n", searchName, mainTitle)
	result := searchByRegex(slice1, generateSearchRegex(searchName), searchName, excludeWords, fn)

	switch len(result) {
	case 0:
		return nil
	case 1:
		return &result[0]
	default:
		sort.Slice(result, func(i, j int) bool {
			return len(fn(result[i])) < len(fn(result[j]))
		})
		for _, item := range result {
			name := fn(item)
			fmt.Printf("searchNameByRegex 03 name:%s, mainTitle:%s, num:%s\n", name, mainTitle, getGameNameAlternative(num))
			if strings.Contains(name, mainTitle) && (num == "" || strings.Contains(name, num) ||
				strings.Contains(name, getGameNameAlternative(num))) {
				return &item
			}
		}
	}
	return &result[0]
}

func ArrayContains(arr []string, str string) bool {
	for _, item := range arr {
		if item == str {
			return true
		}
	}
	return false
}

func removeAllChar(str string, char string) string {
	rs := str
	for _, c := range char {
		rs = strings.ReplaceAll(rs, string(c), "")
	}
	return rs
}

func RemoveString(tagStr1, tagStr2 string) string {
	if tagStr1 == "" {
		return ""
	}
	if tagStr2 == "" {
		return tagStr1
	}
	tags1 := strings.Split(tagStr1, ",")
	tags2 := strings.Split(tagStr2, ",")

	tagSet := make(map[string]bool)
	for _, tag := range tags1 {
		trimmedTag := strings.TrimSpace(tag)
		if trimmedTag != "" {
			tagSet[trimmedTag] = true
		}
	}
	for _, tag := range tags2 {
		trimmedTag := strings.TrimSpace(tag)
		if trimmedTag != "" {
			delete(tagSet, trimmedTag)
		}
	}

	var uniqueTags []string
	for tag := range tagSet {
		uniqueTags = append(uniqueTags, tag)
	}

	return strings.Join(uniqueTags, ",")
}

func IsCamelCase(s string) bool {
	if len(s) == 0 {
		return false
	}
	hasInvalidChar := regexp.MustCompile(`[^a-zA-Z]`).MatchString(s)
	if hasInvalidChar {
		return false
	}

	// 检查是否以小写字母开头
	if s[0] < 'A' || s[0] > 'Z' {
		// fmt.Println("字符串不以小写字母开头")
		return false
	}
	// ab := s[0]f
	// 检查是否包含大写字母
	hasUpperCase := false
	hasLowerCase := false
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			hasUpperCase = true
		}
		if c >= 'a' && c <= 'z' {
			hasLowerCase = true
		}
		// 检查是否包含非字母字符
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			// fmt.Println("字符串包含非字母字符")
			return false
		}
	}
	return hasUpperCase && hasLowerCase
}

func CamelCaseToSpaces(s string) string {
	if len(s) == 0 {
		return ""
	}
	var result []rune
	for i, c := range s {
		if i > 0 && c >= 'A' && c <= 'Z' {
			result = append(result, ' ')
		}
		result = append(result, c)
	}
	return string(result)
}

func IsEnglishWords(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c == ' ' || c == ',') {
			return false
		}
	}
	return true
}

func ContainsAtLeastOneWordInStr(str string, words string) bool {
	if words == "" || str == "" {
		return false
	}
	ws := words
	ws = getGameNameAlternative(ws)
	ws = CamelCaseToSpaces(ws)
	ws = strings.ToLower(ws)
	ws = strings.ReplaceAll(ws, "_", ",")
	ws = strings.ReplaceAll(ws, " ", ",")
	wordList := strings.Split(ws, ",")
	for _, word := range wordList {
		trimmedWord := strings.TrimSpace(word)
		if trimmedWord != "" && strings.Contains(str, trimmedWord) {
			return true
		}
	}
	return false
}

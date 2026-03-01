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

func getMainTitle(searchName string) string {
	// 检查是否包含有效的分隔符（除了纯空格）
	hasValidSeparator := regexp.MustCompile(`[－\-~～]`).MatchString(searchName)

	if !hasValidSeparator {
		// 没有有效分隔符，整个字符串作为主标题处理
		mainTitle := regexp.QuoteMeta(strings.TrimSpace(searchName))
		return mainTitle
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
			return mainTitle
		}
	}

	// 默认情况：整个字符串作为主标题
	mainTitle := regexp.QuoteMeta(strings.TrimSpace(searchName))
	return mainTitle
}

func searchByRegex[T1 any](slice1 []T1, pattern string, searchName string, excludeWords []string, fn func(t1 T1) string) []T1 {
	if len(slice1) == 0 {
		return nil
	}
	re := regexp.MustCompile(pattern)
	var result []T1 = []T1{}
	for _, item := range slice1 {
		target := fn(item)
		fmt.Printf("searchByRegex 01 target:%s\n", target)
		if re.MatchString(target) {
			isExcluded := false
			for _, word := range excludeWords {
				if strings.Contains(target, word) {
					isExcluded = true
					break
				}
			}
			fmt.Printf("target:%s, isexcluded:%v\n", target, isExcluded)
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
		sort.Slice(slice1, func(i, j int) bool {
			return len(fn(slice1[i])) < len(fn(slice1[j]))
		})
		result = []T1{}
		result = append(result, slice1[0])
		return result
	}
	return result
}

func searchNameByRegex[T1 any](slice1 []T1, searchName string, excludeWords []string, fn func(t1 T1) string) *T1 {
	mainTitle := getMainTitle(searchName)
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
			if strings.Contains(fn(item), mainTitle) {
				return &item
			}
		}
	}
	return &result[0]
}

package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"lunabox/internal/models"
	"lunabox/internal/vo"
	"strings"
)

func MergeStrings(tagStr1, tagStr2 string) string {
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

func MapToArray[T1 any, T2 comparable](m map[T2][]T1, fn func(t1 T1) T2) []T1 {
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

func setupTestContext() context.Context {
	// 创建一个简单的context，避免调用Wails runtime
	return context.WithValue(context.Background(), "test_mode", true)
}

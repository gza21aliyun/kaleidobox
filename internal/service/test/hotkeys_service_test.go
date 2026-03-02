package test

import (
	"lunabox/internal/enums"
	"lunabox/internal/service"
	"testing"
	"time"
)

func TestHotkeyService_BasicInitialization(t *testing.T) {
	// 测试基本初始化
	hotkeyService := service.NewHotkeyService()
	
	if hotkeyService == nil {
		t.Fatal("NewHotkeyService should not return nil")
	}
	
	// 验证初始状态
	mappings := hotkeyService.GetKeyMappings()
	if len(mappings) != 0 {
		t.Error("New service should have empty key mappings")
	}
}

func TestHotkeyService_KeyMappingOperations(t *testing.T) {
	hotkeyService := service.NewHotkeyService()
	
	// 测试添加按键映射
	sourceKey := "circle"
	targetKey := "a"
	modifiers := []enums.ModifierKey{enums.ModifierCtrl}
	
	hotkeyService.AddKeyMapping(sourceKey, targetKey, service.MappingTypeDirect, modifiers)
	
	// 验证映射已添加
	mappings := hotkeyService.GetKeyMappings()
	if len(mappings) != 1 {
		t.Fatalf("Expected 1 mapping, got %d", len(mappings))
	}
	
	mapping, exists := mappings[sourceKey]
	if !exists {
		t.Fatal("Mapping should exist for source key")
	}
	
	if mapping.SourceKey != sourceKey {
		t.Errorf("Expected source key %s, got %s", sourceKey, mapping.SourceKey)
	}
	
	if mapping.TargetKey != targetKey {
		t.Errorf("Expected target key %s, got %s", targetKey, mapping.TargetKey)
	}
	
	if mapping.MappingType != service.MappingTypeDirect {
		t.Errorf("Expected mapping type direct, got %s", mapping.MappingType)
	}
	
	if len(mapping.Modifiers) != 1 || mapping.Modifiers[0] != enums.ModifierCtrl {
		t.Error("Modifiers not set correctly")
	}
	
	if !mapping.IsEnabled {
		t.Error("Mapping should be enabled by default")
	}
	
	// 测试禁用映射
	hotkeyService.DisableKeyMapping(sourceKey)
	mappings = hotkeyService.GetKeyMappings()
	if mappings[sourceKey].IsEnabled {
		t.Error("Mapping should be disabled")
	}
	
	// 测试启用映射
	hotkeyService.EnableKeyMapping(sourceKey)
	mappings = hotkeyService.GetKeyMappings()
	if !mappings[sourceKey].IsEnabled {
		t.Error("Mapping should be enabled")
	}
	
	// 测试移除映射
	hotkeyService.RemoveKeyMapping(sourceKey)
	mappings = hotkeyService.GetKeyMappings()
	if len(mappings) != 0 {
		t.Error("Mapping should be removed")
	}
}

func TestHotkeyService_MultipleMappings(t *testing.T) {
	hotkeyService := service.NewHotkeyService()
	
	// 添加多个映射
	mappingsToAdd := []struct {
		sourceKey   string
		targetKey   string
		mappingType service.MappingType
	}{
		{"circle", "a", service.MappingTypeDirect},
		{"triangle", "b", service.MappingTypeDirect},
		{"cross", "enter", service.MappingTypeRelease},
		{"square", "esc", service.MappingTypeDirect},
	}
	
	for _, m := range mappingsToAdd {
		hotkeyService.AddKeyMapping(m.sourceKey, m.targetKey, m.mappingType, nil)
	}
	
	// 验证所有映射都已添加
	mappings := hotkeyService.GetKeyMappings()
	if len(mappings) != len(mappingsToAdd) {
		t.Errorf("Expected %d mappings, got %d", len(mappingsToAdd), len(mappings))
	}
	
	// 验证每个映射的属性
	for _, expected := range mappingsToAdd {
		actual, exists := mappings[expected.sourceKey]
		if !exists {
			t.Errorf("Mapping for %s should exist", expected.sourceKey)
			continue
		}
		
		if actual.TargetKey != expected.targetKey {
			t.Errorf("For %s: expected target %s, got %s", 
				expected.sourceKey, expected.targetKey, actual.TargetKey)
		}
		
		if actual.MappingType != expected.mappingType {
			t.Errorf("For %s: expected type %s, got %s", 
				expected.sourceKey, expected.mappingType, actual.MappingType)
		}
	}
	
	// 测试批量操作
	for _, m := range mappingsToAdd {
		hotkeyService.DisableKeyMapping(m.sourceKey)
	}
	
	mappings = hotkeyService.GetKeyMappings()
	for _, m := range mappingsToAdd {
		if mappings[m.sourceKey].IsEnabled {
			t.Errorf("Mapping %s should be disabled", m.sourceKey)
		}
	}
	
	// 清理所有映射
	for _, m := range mappingsToAdd {
		hotkeyService.RemoveKeyMapping(m.sourceKey)
	}
	
	mappings = hotkeyService.GetKeyMappings()
	if len(mappings) != 0 {
		t.Error("All mappings should be removed")
	}
}

func TestHotkeyService_GameContext(t *testing.T) {
	hotkeyService := service.NewHotkeyService()
	
	// 注意：由于缺少SetServices方法，我们无法完全测试游戏上下文功能
	// 这里主要是测试服务的基本结构
	
	// 验证服务可以正常创建和使用
	if hotkeyService == nil {
		t.Fatal("HotkeyService should be creatable")
	}
	
	mappings := hotkeyService.GetKeyMappings()
	if mappings == nil {
		t.Error("GetKeyMappings should return a valid map")
	}
}

func TestHotkeyService_ModifierKeys(t *testing.T) {
	hotkeyService := service.NewHotkeyService()
	
	// 测试多种修饰键组合
	testCases := []struct {
		name       string
		modifiers  []enums.ModifierKey
		expected   []string
	}{
		{
			name:      "Single Ctrl",
			modifiers: []enums.ModifierKey{enums.ModifierCtrl},
			expected:  []string{"ctrl"},
		},
		{
			name:      "Ctrl + Shift",
			modifiers: []enums.ModifierKey{enums.ModifierCtrl, enums.ModifierShift},
			expected:  []string{"ctrl", "shift"},
		},
		{
			name:      "All modifiers",
			modifiers: []enums.ModifierKey{enums.ModifierCtrl, enums.ModifierShift, enums.ModifierAlt},
			expected:  []string{"ctrl", "shift", "alt"},
		},
		{
			name:      "No modifiers",
			modifiers: []enums.ModifierKey{},
			expected:  []string{},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sourceKey := "test_source_" + tc.name
			targetKey := "test_target"
			
			hotkeyService.AddKeyMapping(sourceKey, targetKey, service.MappingTypeDirect, tc.modifiers)
			
			mappings := hotkeyService.GetKeyMappings()
			mapping := mappings[sourceKey]
			
			if len(mapping.Modifiers) != len(tc.modifiers) {
				t.Errorf("Expected %d modifiers, got %d", len(tc.modifiers), len(mapping.Modifiers))
			}
			
			// 验证修饰键内容
			for i, expectedMod := range tc.modifiers {
				if i < len(mapping.Modifiers) && mapping.Modifiers[i] != expectedMod {
					t.Errorf("Modifier at index %d: expected %v, got %v", 
						i, expectedMod, mapping.Modifiers[i])
				}
			}
			
			// 清理
			hotkeyService.RemoveKeyMapping(sourceKey)
		})
	}
}

func TestHotkeyService_MappingTypes(t *testing.T) {
	hotkeyService := service.NewHotkeyService()
	
	// 测试不同的映射类型
	testCases := []struct {
		name        string
		mappingType service.MappingType
		description string
	}{
		{
			name:        "Direct Mapping",
			mappingType: service.MappingTypeDirect,
			description: "Should map press/release directly",
		},
		{
			name:        "Release Mapping",
			mappingType: service.MappingTypeRelease,
			description: "Should only trigger on release",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sourceKey := "source_" + tc.name
			targetKey := "target_" + tc.name
			
			hotkeyService.AddKeyMapping(sourceKey, targetKey, tc.mappingType, nil)
			
			mappings := hotkeyService.GetKeyMappings()
			mapping := mappings[sourceKey]
			
			if mapping.MappingType != tc.mappingType {
				t.Errorf("Expected mapping type %s, got %s", tc.mappingType, mapping.MappingType)
			}
			
			// 清理
			hotkeyService.RemoveKeyMapping(sourceKey)
		})
	}
}

func TestHotkeyService_ConcurrentOperations(t *testing.T) {
	hotkeyService := service.NewHotkeyService()
	
	// 测试并发安全性
	done := make(chan bool)
	errorChan := make(chan string, 100)
	
	// 并发添加映射
	for i := 0; i < 10; i++ {
		go func(id int) {
			sourceKey := "concurrent_source_" + string(rune('a'+id))
			targetKey := "concurrent_target_" + string(rune('A'+id))
			
			hotkeyService.AddKeyMapping(sourceKey, targetKey, service.MappingTypeDirect, nil)
			
			// 验证添加成功
			mappings := hotkeyService.GetKeyMappings()
			if _, exists := mappings[sourceKey]; !exists {
				errorChan <- "Mapping not found after concurrent addition"
			}
			
			done <- true
		}(i)
	}
	
	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case err := <-errorChan:
			t.Error(err)
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out")
		}
	}
	
	// 验证所有映射都存在
	mappings := hotkeyService.GetKeyMappings()
	expectedCount := 10
	if len(mappings) != expectedCount {
		t.Errorf("Expected %d mappings after concurrent operations, got %d", expectedCount, len(mappings))
	}
	
	// 清理所有映射
	for key := range mappings {
		hotkeyService.RemoveKeyMapping(key)
	}
	
	finalMappings := hotkeyService.GetKeyMappings()
	if len(finalMappings) != 0 {
		t.Error("All mappings should be removed after cleanup")
	}
}

func TestHotkeyService_StateTracking(t *testing.T) {
	hotkeyService := service.NewHotkeyService()
	
	// 测试按键状态跟踪机制
	// 注意：由于checkKeyState使用robotgo.KeyTap进行检测，
	// 在测试环境中可能无法真实反映按键状态
	// 这里主要是测试状态跟踪的数据结构
	
	// 验证初始状态下没有按键被跟踪
	mappings := hotkeyService.GetKeyMappings()
	if len(mappings) != 0 {
		t.Error("Should start with no mappings")
	}
	
	// 添加一个映射来测试状态跟踪
	hotkeyService.AddKeyMapping("test_source", "test_target", service.MappingTypeDirect, nil)
	
	mappings = hotkeyService.GetKeyMappings()
	if len(mappings) != 1 {
		t.Error("Should have one mapping after addition")
	}
	
	// 验证映射属性
	mapping := mappings["test_source"]
	if mapping == nil {
		t.Fatal("Mapping should exist")
	}
	
	if mapping.SourceKey != "test_source" {
		t.Error("Source key mismatch")
	}
	
	if mapping.TargetKey != "test_target" {
		t.Error("Target key mismatch")
	}
	
	if mapping.MappingType != service.MappingTypeDirect {
		t.Error("Mapping type mismatch")
	}
	
	// 清理
	hotkeyService.RemoveKeyMapping("test_source")
	mappings = hotkeyService.GetKeyMappings()
	if len(mappings) != 0 {
		t.Error("Mapping should be removed")
	}
}

func TestHotkeyService_ServiceIntegration(t *testing.T) {
	// 测试与其他服务的集成
	hotkeyService := service.NewHotkeyService()
	
	// 注意：由于类型不匹配，这里只是验证服务可以创建
	// 实际的集成测试需要正确的服务实例
	
	if hotkeyService == nil {
		t.Fatal("HotkeyService should be creatable")
	}
}


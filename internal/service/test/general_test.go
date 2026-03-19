package test

import (
	"fmt"
	"lunabox/internal/applog"
	"lunabox/internal/service"
	"lunabox/internal/utils"
	"testing"
)

func TestProcmon(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()
	// config := appconf.AppConfig{}

	services := createServices(t)

	t.Run("import success", func(t *testing.T) {
		applog.SetMode(applog.ModeCLI)
		services.StartService.DetectProcessSavePath(10552)
	})
}

func TestCamel(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()
	// config := appconf.AppConfig{}

	s := createServices(t)
	searchName := "MagicalMarriageLunatics"

	t.Run("import success", func(t *testing.T) {
		applog.SetMode(applog.ModeCLI)
		if s.config.Language == "en" {

		}
		rs := ""
		if utils.IsCamelCase(searchName) {
			fmt.Println("camel")
			rs = utils.CamelCaseToSpaces(searchName)
		} else {
			fmt.Println("not camel")
		}
		if rs != "Magical Marriage Lunatics" {
			t.Errorf("err rs=%s", rs)
		}

	})
}

func TestDbRestore(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()
	// config := appconf.AppConfig{}

	s := createServices(t)
	s.config.PendingDBRestore = `C:\temp\projects\lunabox\build\bin\lunabox_full_2026-03-19T18-32-31.zip`

	t.Run("restore success", func(t *testing.T) {
		applog.SetMode(applog.ModeCLI)
		service.ExecuteDBRestore(&s.config)

	})
}

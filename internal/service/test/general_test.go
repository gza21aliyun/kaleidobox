package test

import (
	"lunabox/internal/applog"
	"testing"
)

func TestProcmon(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()
	// config := appconf.AppConfig{}

	services := createServices(t)

	t.Run("import success", func(t *testing.T) {
		applog.SetMode(applog.ModeCLI)
		services.StartService.DetectProcessSavePath(26496)
	})
}

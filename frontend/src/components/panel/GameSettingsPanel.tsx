import type { appconf } from "../../../wailsjs/go/models";
import { toast } from "react-hot-toast";
import { SelectGameExecutable } from "../../../wailsjs/go/service/GameService";
import { BetterButton } from "../ui/BetterButton";
import { BetterSwitch } from "../ui/BetterSwitch";
import { useTranslation } from "react-i18next";

interface GameSettingsPanelProps {
  formData: appconf.AppConfig;
  onChange: (data: appconf.AppConfig) => void;
}

export function GameSettingsPanel({ formData, onChange }: GameSettingsPanelProps) {
  const { t } = useTranslation();
  const handleSelectLocaleEmulatorPath = async () => {
    try {
      const path = await SelectGameExecutable();
      if (path) {
        onChange({ ...formData, locale_emulator_path: path } as appconf.AppConfig);
      }
    }
    catch (error) {
      console.error("Failed to select Locale Emulator:", error);
      toast.error(t("gameSettings.localeEmulatorSelectError"));
    }
  };

  const handleSelectMagpiePath = async () => {
    try {
      const path = await SelectGameExecutable();
      if (path) {
        onChange({ ...formData, magpie_path: path } as appconf.AppConfig);
      }
    }
    catch (error) {
      console.error("Failed to select Magpie:", error);
      toast.error(t("gameSettings.magpieSelectError"));
    }
  };
  return (
    <>
      <div className="flex items-center justify-between p-2">
        <div className="flex-1">
          <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">
            {t("gameSettings.recordActiveTimeOnly")}
          </label>
          <p className="text-xs text-brand-500 dark:text-brand-400 mt-1">
            {t("gameSettings.recordActiveTimeOnlyDescription")}
          </p>
        </div>
        <BetterSwitch
          id="record_active_time_only"
          checked={formData.record_active_time_only || false}
          onCheckedChange={checked =>
            onChange({ ...formData, record_active_time_only: checked } as appconf.AppConfig)}
        />
      </div>

      <div className="mt-4 p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-700 rounded-lg">
        <div className="flex items-start gap-2">
          <span className="i-mdi-alert text-amber-600 dark:text-amber-400 text-lg mt-0.5" />
          <div className="text-xs text-amber-700 dark:text-amber-300">
            <p className="font-medium mb-1">{t("gameSettings.note")}：</p>
            <ul className="list-disc list-inside space-y-1 ml-2">
              <li>{t("gameSettings.activeTimeNote1")}</li>
              <li>{t("gameSettings.activeTimeNote2")}</li>
              <li>{t("gameSettings.activeTimeNote3")}</li>
            </ul>
          </div>
        </div>
      </div>

      {/* 自动进程检测 */}
      <div className="mt-6 border-t border-brand-200 dark:border-brand-700 pt-6">
        <div className="flex items-center justify-between p-2">
          <div className="flex-1">
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">
              {t("gameSettings.autoProcessDetection")}
            </label>
            <p className="text-xs text-brand-500 dark:text-brand-400 mt-1">
              {t("gameSettings.autoProcessDetectionDescription")}
            </p>
          </div>
          <BetterSwitch
            id="auto_detect_game_process"
            checked={formData.auto_detect_game_process ?? true}
            onCheckedChange={checked =>
              onChange({ ...formData, auto_detect_game_process: checked } as appconf.AppConfig)}
          />
        </div>
      </div>

      {/* Locale Emulator 配置 */}
      <div className="mt-6 border-t border-brand-200 dark:border-brand-700 pt-6">
        <h3 className="text-sm font-semibold text-brand-900 dark:text-white mb-4">{t("gameSettings.gameLaunchTools")}</h3>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-2">
              {t("gameSettings.localeEmulatorPath")}
            </label>
            <div className="flex gap-2">
              <input
                type="text"
                value={formData.locale_emulator_path || ""}
                onChange={e => onChange({ ...formData, locale_emulator_path: e.target.value } as appconf.AppConfig)}
                placeholder={t("gameSettings.localeEmulatorPathPlaceholder")}
                className="glass-input flex-1 px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
              />
              <BetterButton onClick={handleSelectLocaleEmulatorPath} icon="i-mdi-file">
                {t("common.select")}
              </BetterButton>
            </div>
            <p className="mt-1 text-xs text-brand-500">
              {t("gameSettings.localeEmulatorPathDescription")}
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-2">
              {t("gameSettings.magpiePath")}
            </label>
            <div className="flex gap-2">
              <input
                type="text"
                value={formData.magpie_path || ""}
                onChange={e => onChange({ ...formData, magpie_path: e.target.value } as appconf.AppConfig)}
                placeholder={t("gameSettings.magpiePathPlaceholder")}
                className="glass-input flex-1 px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
              />
              <BetterButton onClick={handleSelectMagpiePath} icon="i-mdi-file">
                {t("common.select")}
              </BetterButton>
            </div>
            <p className="mt-1 text-xs text-brand-500">
              {t("gameSettings.magpiePathDescription")}
            </p>
          </div>
        </div>
      </div>
    </>
  );
}

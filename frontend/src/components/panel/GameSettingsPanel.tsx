import type { appconf } from "../../../wailsjs/go/models";
import { toast } from "react-hot-toast";
import { SelectGameExecutable } from "../../../wailsjs/go/service/GameService";
import { BetterButton } from "../ui/BetterButton";
import { BetterSwitch } from "../ui/BetterSwitch";
import { useTranslation } from "react-i18next";
import { useState, useEffect, useRef } from "react";

interface GameSettingsPanelProps {
  formData: appconf.AppConfig;
  onChange: (data: appconf.AppConfig) => void;
}

function getKeyName(code: string): string {
  if (code.startsWith("Key")) return code.slice(3);
  if (code.startsWith("Digit")) return code.slice(5);
  switch (code) {
    case "ControlLeft": case "ControlRight": return "Ctrl";
    case "ShiftLeft": case "ShiftRight": return "Shift";
    case "AltLeft": case "AltRight": return "Alt";
    case "MetaLeft": case "MetaRight": return "Win";
    case "F1": case "F2": case "F3": case "F4": case "F5": case "F6":
    case "F7": case "F8": case "F9": case "F10": case "F11": case "F12":
      return code;
    default: return code;
  }
}

export function GameSettingsPanel({ formData, onChange }: GameSettingsPanelProps) {
  const { t } = useTranslation();
  const [recordingHotkey, setRecordingHotkey] = useState(false);
  const hotkeyRecorderRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleGlobalClick = (e: MouseEvent) => {
      if (hotkeyRecorderRef.current && !hotkeyRecorderRef.current.contains(e.target as Node)) {
        setRecordingHotkey(false);
      }
    };
    if (recordingHotkey) {
      document.addEventListener("click", handleGlobalClick);
    }
    return () => document.removeEventListener("click", handleGlobalClick);
  }, [recordingHotkey]);
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
          <div className="flex flex-col gap-1.5 pl-6">
            <label className="text-sm font-medium text-brand-700 dark:text-brand-300">{t("gameSettings.detectTime")}</label>
            <input
              type="number"
              name="detect_time"
              value={formData.detect_time || 60}
              onChange={e => onChange({ ...formData, detect_time: Number.parseInt(e.target.value) || 0 } as appconf.AppConfig)}
              className="glass-input w-24 px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-neutral-500 dark:bg-brand-700 dark:text-white"
            />
          </div>
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

          <div>
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-2">
              Magpie 缩放快捷键
            </label>
            <div className="flex gap-2">
              <div
                ref={hotkeyRecorderRef}
                tabIndex={0}
                onClick={() => setRecordingHotkey(true)}
                onKeyDown={(e) => {
                  if (!recordingHotkey) return;
                  e.preventDefault();
                  e.stopPropagation();
                  const mods: string[] = [];
                  let mainKey = "";
                  if (e.ctrlKey) mods.push("Ctrl");
                  if (e.shiftKey) mods.push("Shift");
                  if (e.altKey) mods.push("Alt");
                  if (e.metaKey) mods.push("Win");
                  const keyName = getKeyName(e.code);
                  if (!["Ctrl", "Shift", "Alt", "Win"].includes(keyName)) {
                    mainKey = keyName;
                  }
                  if (mainKey) {
                    let finalKeys = [...new Set([...mods, mainKey])];
                    onChange({ ...formData, magpie_hotkey: finalKeys.join("+") } as appconf.AppConfig);
                    setRecordingHotkey(false);
                  }
                }}
                className={`glass-input flex-1 px-3 py-2 border rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white outline-none select-none transition-all cursor-text ${
                  recordingHotkey
                    ? "ring-2 ring-blue-500 border-blue-400"
                    : "border-brand-300 dark:border-brand-600 focus:ring-2 focus:ring-neutral-500"
                }`}
              >
                <span className={recordingHotkey ? "text-blue-500 italic" : ""}>
                  {recordingHotkey
                    ? "按快捷键组合..."
                    : formData.magpie_hotkey || "Win+Shift+A"}
                </span>
              </div>
              <BetterButton
                onClick={() => {
                  onChange({ ...formData, magpie_hotkey: "Win+Shift+A" } as appconf.AppConfig);
                }}
                size="sm"
              >
                重置
              </BetterButton>
            </div>
            <p className="mt-1 text-xs text-brand-500">
              点击输入框后直接按下要设置的快捷键组合，需与 Magpie 中设置的快捷键一致
            </p>
          </div>
        </div>
      </div>
    </>
  );
}

import type { appconf, models } from "../../../wailsjs/go/models";
import { BetterSelect } from "../ui/BetterSelect";
import { BetterSwitch } from "../ui/BetterSwitch";
import { useTranslation } from 'react-i18next';
import { SelectFile } from "../../../wailsjs/go/service/GameService";
import { GetAvailableDisplays } from "../../../wailsjs/go/service/StartService";
import { useEffect, useState } from "react";

interface BetterSelectOption {
  value: string;
  label: string;
}

const COMMON_TIMEZONES: { value: string; tKey: string }[] = [
  { value: "Asia/Shanghai", tKey: "timezone.names.shanghai" },
  { value: "Asia/Tokyo", tKey: "timezone.names.tokyo" },
  { value: "Asia/Seoul", tKey: "timezone.names.seoul" },
  { value: "Asia/Hong_Kong", tKey: "timezone.names.hongkong" },
  { value: "Asia/Taipei", tKey: "timezone.names.taipei" },
  { value: "Asia/Singapore", tKey: "timezone.names.singapore" },
  { value: "Asia/Bangkok", tKey: "timezone.names.bangkok" },
  { value: "Asia/Dubai", tKey: "timezone.names.dubai" },
  { value: "Europe/London", tKey: "timezone.names.london" },
  { value: "Europe/Paris", tKey: "timezone.names.paris" },
  { value: "Europe/Berlin", tKey: "timezone.names.berlin" },
  { value: "Europe/Moscow", tKey: "timezone.names.moscow" },
  { value: "America/New_York", tKey: "timezone.names.newyork" },
  { value: "America/Chicago", tKey: "timezone.names.chicago" },
  { value: "America/Denver", tKey: "timezone.names.denver" },
  { value: "America/Los_Angeles", tKey: "timezone.names.losangeles" },
  { value: "America/Sao_Paulo", tKey: "timezone.names.saopaulo" },
  { value: "Australia/Sydney", tKey: "timezone.names.sydney" },
  { value: "Pacific/Auckland", tKey: "timezone.names.auckland" },
  { value: "UTC", tKey: "timezone.names.utc" },
];

interface BasicSettingsProps {
  formData: appconf.AppConfig;
  onChange: (data: appconf.AppConfig) => void;
}

export function BasicSettingsPanel({ formData, onChange }: BasicSettingsProps) {
  const { t, i18n } = useTranslation();
  const [displays, setDisplays] = useState<models.MonitorInfo[]>([]);

  useEffect(() => {
    GetAvailableDisplays().then(setDisplays).catch(console.error);
  }, []);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const { name, value, type } = e.target;
    const newValue = type === "checkbox" ? (e.target as HTMLInputElement).checked : value;
    onChange({ ...formData, [name]: newValue } as appconf.AppConfig);
  };

  const handleFfmpegSelect = async () => {
    const result = await SelectFile("FFmpeg.exe", "*.exe");
    if (result) {
      onChange({ ...formData, ffmpeg_path: result } as appconf.AppConfig);
    }
  };

  const displayOptions: BetterSelectOption[] = [
    { value: "", label: t("basic.defaultDisplay") },
    ...displays.map(d => ({
      value: d.device_name,
      label: `${d.device_name}${d.is_primary ? ` (${t("basic.primaryDisplay")})` : ""}`
    }))
  ];

  return (
    <>
      <div className="space-y-2">
        <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">{t("basic.bangumiAccessToken")}</label>
        <input
          type="text"
          name="access_token"
          value={formData.access_token || ""}
          onChange={handleChange}
          className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-neutral-500 dark:bg-brand-700 dark:text-white"
        />
        <p className="text-xs text-brand-500 dark:text-brand-400">{t("basic.bangumiTokenHint")}</p>
      </div>

      <div className="space-y-2">
        <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">{t("basic.vndbAccessToken")}</label>
        <input
          type="text"
          name="vndb_access_token"
          value={formData.vndb_access_token || ""}
          onChange={handleChange}
          className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-neutral-500 dark:bg-brand-700 dark:text-white"
        />
      </div>

      <div className="flex items-center justify-between p-2">
        <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">
          {t('basic.searchChineseFirst')}
        </label>
        <BetterSwitch
          id="search_cn"
          checked={formData.search_cn || false}
          onCheckedChange={checked => onChange({ ...formData, search_cn: checked } as appconf.AppConfig)}
        />
      </div>

      <div className="flex items-center justify-between p-2">
        <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">
          {t('basic.autoDownloadImages')}
        </label>
        <BetterSwitch
          id="auto_download_images"
          checked={formData.auto_download_images || false}
          onCheckedChange={checked => onChange({ ...formData, auto_download_images: checked } as appconf.AppConfig)}
        />
      </div>

      <div className="flex items-center justify-between p-2">
        <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">
          {t('basic.moreLogs')}
        </label>
        <BetterSwitch
          id="log_all"
          checked={formData.log_all || false}
          onCheckedChange={checked => onChange({ ...formData, log_all: checked } as appconf.AppConfig)}
        />
      </div>

      <div className="flex items-center justify-between p-2">
        <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">
          {t("basic.eroscapeUseMirror")}
        </label>
        <BetterSwitch
          id="eroscape_use_mirror"
          checked={formData.eroscape_use_mirror || false}
          onCheckedChange={checked => onChange({ ...formData, eroscape_use_mirror: checked } as appconf.AppConfig)}
        />
      </div>

      <div className="flex items-center justify-between p-2">
        <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">
          {t("basic.hideOldMatchingBtns")}
        </label>
        <BetterSwitch
          id="hide_old_matching_btns"
          checked={formData.hide_old_matching_btns || false}
          onCheckedChange={checked => onChange({ ...formData, hide_old_matching_btns: checked } as appconf.AppConfig)}
        />
      </div>


      <div className="space-y-2">
        <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">{t("common.language")}</label>
        <BetterSelect
          name="language"
          value={formData.language}
          onChange={value => {
            onChange({ ...formData, language: value } as appconf.AppConfig)
            i18n.changeLanguage(value);
          }}
          options={[
            { value: "zh-CN", label: "简体中文" },
            { value: "en-US", label: "English" },
            { value: "ja-JP", label: "日本語" },

          ]}
        />
      </div>

      <div className="space-y-2">
        <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">{t("basic.timezone")}</label>
        <span className="text-xs text-brand-500 dark:text-brand-400">{t("basic.timezoneHint")}</span>
        <BetterSelect
          name="timezone"
          value={formData.time_zone || "Asia/Shanghai"}
          onChange={value => onChange({ ...formData, time_zone: value } as appconf.AppConfig)}
          options={COMMON_TIMEZONES.map(o => ({ value: o.value, label: t(o.tKey) }))}
          placeholder={t("common.pleaseSelect")}
        />
      </div>

      <div className="space-y-2">
        <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">FFmpeg Path</label>
        <div className="flex space-x-2">
          <input
            type="text"
            name="ffmpeg_path"
            value={formData.ffmpeg_path || ""}
            onChange={handleChange}
            className="glass-input flex-1 px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-neutral-500 dark:bg-brand-700 dark:text-white"
            placeholder="Path to ffmpeg.exe"
          />
          <button
            onClick={handleFfmpegSelect}
            className="px-3 py-2 bg-brand-600 text-white rounded-md hover:bg-brand-700 focus:outline-none focus:ring-2 focus:ring-brand-500"
          >
            Browse
          </button>
        </div>
      </div>

      <div className="flex items-center justify-between p-2">
        <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">
          {t("basic.minimizeToTray")}
        </label>
        <BetterSwitch
          id="close_to_tray"
          checked={formData.close_to_tray || false}
          onCheckedChange={checked => onChange({ ...formData, close_to_tray: checked } as appconf.AppConfig)}
        />
      </div>

      <div className="flex items-center justify-between p-2">
        <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">
          {t('basic.newFolderChooser')}
        </label>
        <BetterSwitch
          id="new_folder_chooser"
          checked={formData.new_folder_chooser || false}
          onCheckedChange={checked => onChange({ ...formData, new_folder_chooser: checked } as appconf.AppConfig)}
        />
      </div>

      <div className="flex items-center justify-between p-2">
        <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">
          {t('basic.deleteRequiresConfirm')}
        </label>
        <BetterSwitch
          id="delete_confirm"
          checked={formData.delete_confirm || false}
          onCheckedChange={checked => onChange({ ...formData, delete_confirm: checked } as appconf.AppConfig)}
        />
      </div>

      <div className="space-y-2">
        <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">{t("basic.displayName")}</label>
        <BetterSelect
          name="display_name"
          value={formData.display_name || ""}
          onChange={value => onChange({ ...formData, display_name: value } as appconf.AppConfig)}
          options={displayOptions}
          placeholder={t("common.pleaseSelect")}
        />
      </div>
    </>
  );
}

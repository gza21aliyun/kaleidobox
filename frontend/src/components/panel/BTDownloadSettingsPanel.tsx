import type { appconf } from "../../../wailsjs/go/models";
import { useTranslation } from "react-i18next";

interface BTDownloadSettingsPanelProps {
  formData: appconf.AppConfig;
  onChange: (data: appconf.AppConfig) => void;
}

export function BTDownloadSettingsPanel({ formData, onChange }: BTDownloadSettingsPanelProps) {
  const { t } = useTranslation();

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value, type } = e.target;
    const newValue = type === "number" ? Number(value) : value;
    onChange({ ...formData, [name]: newValue } as appconf.AppConfig);
  };

  return (
    <>
      {/* RSS URL 配置 */}
      <div className="space-y-2">
        <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">
          {t("btDownload.rssUrl")}
        </label>
        <input
          type="text"
          name="rss_url"
          value={formData.rss_url || ""}
          onChange={handleChange}
          placeholder="https://sukebei.nyaa.si/?page=rss&c=1_3&f=0&q=%search_key"
          className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-neutral-500 dark:bg-brand-700 dark:text-white"
        />
        <p className="text-xs text-brand-500 dark:text-brand-400">
          {t("btDownload.rssUrlHint")}
        </p>
      </div>

      {/* qBittorrent 配置 */}
      <div className="mt-6 border-t border-brand-200 dark:border-brand-700 pt-6">
        <h3 className="text-sm font-semibold text-brand-900 dark:text-white mb-4">
          {t("btDownload.qbittorrentSettings")}
        </h3>

        <div className="space-y-4">
          {/* 服务器地址 */}
          <div>
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-2">
              {t("btDownload.qbServer")}
            </label>
            <input
              type="text"
              name="qb_server"
              value={formData.qb_server || ""}
              onChange={handleChange}
              placeholder="192.168.1.100"
              className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-neutral-500 dark:bg-brand-700 dark:text-white"
            />
          </div>

          {/* 端口 */}
          <div>
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-2">
              {t("btDownload.qbPort")}
            </label>
            <input
              type="number"
              name="qb_port"
              value={formData.qb_port || 8080}
              onChange={handleChange}
              placeholder="8080"
              className="glass-input w-32 px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-neutral-500 dark:bg-brand-700 dark:text-white"
            />
          </div>

          {/* 用户名 */}
          <div>
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-2">
              {t("btDownload.qbUser")}
            </label>
            <input
              type="text"
              name="qb_user"
              value={formData.qb_user || ""}
              onChange={handleChange}
              placeholder="admin"
              className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-neutral-500 dark:bg-brand-700 dark:text-white"
            />
          </div>

          {/* 密码 */}
          <div>
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-2">
              {t("btDownload.qbPassword")}
            </label>
            <input
              type="password"
              name="qb_password"
              value={formData.qb_password || ""}
              onChange={handleChange}
              placeholder="••••••••"
              className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-neutral-500 dark:bg-brand-700 dark:text-white"
            />
          </div>

          {/* 下载目录 */}
          <div>
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-2">
              {t("btDownload.qbDownloadFolder")}
            </label>
            <input
              type="text"
              name="qb_download_folder"
              value={formData.qb_download_folder || ""}
              onChange={handleChange}
              placeholder={t("btDownload.downloadFolderPlaceholder") || "/downloads"}
              className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-neutral-500 dark:bg-brand-700 dark:text-white"
            />
            <p className="mt-1 text-xs text-brand-500 dark:text-brand-400">
              {t("btDownload.downloadFolderHint")}
            </p>
          </div>
        </div>
      </div>
    </>
  );
}

import type { appconf } from "../../../wailsjs/go/models";
import { useState } from "react";
import toast from "react-hot-toast";
import { GetOneDriveAuthURL, SetupCloudBackup, StartOneDriveAuth, TestOneDriveConnection, TestS3Connection, TestWebDavConnection } from "../../../wailsjs/go/service/BackupService";
import { GetAppConfig } from "../../../wailsjs/go/service/ConfigService";
import { BrowserOpenURL } from "../../../wailsjs/runtime";
import { PasswordInputModal } from "../modal/PasswordInputModal";
import { BetterSelect } from "../ui/BetterSelect";
import { BetterSwitch } from "../ui/BetterSwitch";
import i18next from "../../i18n/i18n";
const t = i18next.t;

interface CloudBackupSettingsProps {
  formData: appconf.AppConfig;
  onChange: (data: appconf.AppConfig) => void;
}

export function CloudBackupSettingsPanel({ formData, onChange }: CloudBackupSettingsProps) {
  const [testingS3, setTestingS3] = useState(false);
  const [testingOneDrive, setTestingOneDrive] = useState(false);
  const [testingWebDav, setTestingWebDav] = useState(false);
  const [authorizingOneDrive, setAuthorizingOneDrive] = useState(false);
  const [showPasswordModal, setShowPasswordModal] = useState(false);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    onChange({ ...formData, [name]: value } as appconf.AppConfig);
  };

  const handleSetupBackupPassword = async (password: string, confirmPassword: string) => {
    if (password.length < 6) {
      toast.error(t("cloudBackup.passwordMinLength"));
      return;
    }

    if (password !== confirmPassword) {
      toast.error(t("cloudBackup.passwordMismatch"));
      return;
    }

    try {
      const userID = await SetupCloudBackup(password);
      toast.success(t("cloudBackup.passwordSetupSuccess", { userId: userID.substring(0, 8) }));

      // 重新加载配置以更新界面
      const updatedConfig = await GetAppConfig();
      onChange(updatedConfig);
    }
    catch (err: any) {
      toast.error(t("cloudBackup.setupFailed", { error: err }));
    }
  };

  const handleTestS3 = async () => {
    setTestingS3(true);
    try {
      await TestS3Connection(formData);
      toast.success(t("cloudBackup.s3TestSuccess"));
    }
    catch (err: any) {
      toast.error(t("cloudBackup.s3TestFailed", { error: err }));
    }
    finally {
      setTestingS3(false);
    }
  };

  const handleTestOneDrive = async () => {
    setTestingOneDrive(true);
    try {
      await TestOneDriveConnection(formData);
      toast.success(t("cloudBackup.oneDriveTestSuccess"));
    }
    catch (err: any) {
      toast.error(t("cloudBackup.oneDriveTestFailed", { error: err }));
    }
    finally {
      setTestingOneDrive(false);
    }
  };

  const handleTestWebDav = async () => {
    setTestingWebDav(true);
    try {
      await TestWebDavConnection(formData);
      toast.success(t("cloudBackup.webDavTestSuccess"));
    }
    catch (err: any) {
      toast.error(t("cloudBackup.webDavTestFailed", { error: err }));
    }
    finally {
      setTestingWebDav(false);
    }
  };

  const handleOneDriveAuth = async () => {
    setAuthorizingOneDrive(true);
    try {
      const authURL = await GetOneDriveAuthURL();
      BrowserOpenURL(authURL);
      const refreshToken = await StartOneDriveAuth();
      onChange({ ...formData, onedrive_refresh_token: refreshToken } as appconf.AppConfig);
      toast.success(t("cloudBackup.oneDriveAuthSuccess"));
    }
    catch (err: any) {
      toast.error(t("cloudBackup.authFailed", { error: err }));
    }
    finally {
      setAuthorizingOneDrive(false);
    }
  };

  return (
    <div className="space-y-4">
      <div className="glass-card flex items-center justify-between p-4 bg-brand-50 dark:bg-brand-800/50 rounded-lg">
        <div className="flex flex-col">
          <label htmlFor="cloud_backup_enabled" className="text-sm font-medium text-brand-700 dark:text-brand-300 cursor-pointer">
            {t("cloudBackup.enableCloudBackup")}
          </label>
          <p className="text-xs text-brand-500 dark:text-brand-400 mt-1">{t("cloudBackup.syncToCloudStorage")}</p>
        </div>
        <BetterSwitch
          id="cloud_backup_enabled"
          checked={formData.cloud_backup_enabled || false}
          onCheckedChange={checked => onChange({ ...formData, cloud_backup_enabled: checked } as appconf.AppConfig)}
        />
      </div>

      <div className="space-y-2">
        <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">{t("cloudBackup.cloudStorageProvider")}</label>
        <BetterSelect
          value={formData.cloud_backup_provider || "s3"}
          onChange={value => onChange({ ...formData, cloud_backup_provider: value } as appconf.AppConfig)}
          options={[
            { value: "s3", label: t("cloudBackup.s3CompatibleStorage") },
            { value: "onedrive", label: "OneDrive" },
            { value: "webdav", label: "WebDav" },
          ]}
        />
      </div>

      {/* S3 配置 */}
      {formData.cloud_backup_provider === "s3" && (
        <div className="glass-card space-y-4 p-4 bg-brand-100 dark:bg-brand-800 rounded-lg">
          <h3 className="text-sm font-medium text-brand-800 dark:text-brand-200">{t("cloudBackup.s3Config")}</h3>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">{t("cloudBackup.backupPassword")}</label>
            {formData.backup_password
              ? (
                  <div className="space-y-2">
                    <div className="px-3 py-2 bg-brand-100 dark:bg-brand-700 border border-brand-300 dark:border-brand-600 rounded-md text-brand-600 dark:text-brand-300">
                      ********
                    </div>
                    <p className="text-xs text-brand-500 dark:text-brand-400">
                      <span className="text-success-600 dark:text-success-400">
                        ✓ {t("cloudBackup.passwordSet", { userId: formData.backup_user_id?.substring(0, 8) })}
                      </span>
                    </p>
                  </div>
                )
              : (
                  <div className="space-y-2">
                    <button
                      type="button"
                      onClick={() => setShowPasswordModal(true)}
                      className="glass-btn-neutral w-full px-4 py-2 bg-brand-600 hover:bg-brand-700 text-white rounded-md transition-colors flex items-center justify-center gap-2"
                    >
                      <span className="i-mdi-lock-plus text-lg" />
                      {t("cloudBackup.setupBackupPassword")}
                    </button>
                    <p className="text-xs text-brand-500 dark:text-brand-400">
                      {t("cloudBackup.usedForUserId")}
                    </p>
                    <p className="text-xs text-warning-600 dark:text-warning-400">
                      {t("cloudBackup.passwordImportant")}
                    </p>
                  </div>
                )}
          </div>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">{t("cloudBackup.s3Endpoint")}</label>
            <input type="text" name="s3_endpoint" value={formData.s3_endpoint || ""} onChange={handleChange} placeholder="https://s3.example.com" className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-neutral-500 dark:bg-brand-700 dark:text-white" />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">{t("cloudBackup.s3Region")}</label>
              <input type="text" name="s3_region" value={formData.s3_region || ""} onChange={handleChange} placeholder="us-east-1" className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-neutral-500 dark:bg-brand-700 dark:text-white" />
            </div>
            <div className="space-y-2">
              <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">{t("cloudBackup.s3Bucket")}</label>
              <input type="text" name="s3_bucket" value={formData.s3_bucket || ""} onChange={handleChange} placeholder="kaleidobox-backup" className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-neutral-500 dark:bg-brand-700 dark:text-white" />
            </div>
          </div>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">{t("cloudBackup.accessKey")}</label>
            <input type="text" name="s3_access_key" value={formData.s3_access_key || ""} onChange={handleChange} className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-neutral-500 dark:bg-brand-700 dark:text-white" />
          </div>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">{t("cloudBackup.secretKey")}</label>
            <input type="password" name="s3_secret_key" value={formData.s3_secret_key || ""} onChange={handleChange} className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-neutral-500 dark:bg-brand-700 dark:text-white" />
          </div>
          <div className="flex justify-end">
            <button type="button" onClick={handleTestS3} disabled={testingS3} className="glass-btn-neutral px-3 py-1.5 text-sm bg-brand-100 text-brand-700 rounded-md hover:bg-brand-200 dark:bg-brand-700 dark:text-brand-300 dark:hover:bg-brand-600 disabled:opacity-50">
              {testingS3 ? t("cloudBackup.testing") : t("cloudBackup.testConnection")}
            </button>
          </div>
        </div>
      )}

      {/* OneDrive 配置 */}
      {formData.cloud_backup_provider === "onedrive" && (
        <div className="space-y-4 p-4 bg-brand-100 dark:bg-brand-800 rounded-lg">
          <h3 className="text-sm font-medium text-brand-800 dark:text-brand-200">{t("cloudBackup.oneDriveConfig")}</h3>

          <div className="p-3 bg-brand-100 dark:bg-brand-700 rounded-md border border-brand-300 dark:border-brand-600">
            <div className="flex items-start gap-2">
              <span className="i-mdi-information-outline text-lg text-warning-500 dark:text-brand-400 mt-0.5 flex-shrink-0" />
              <div className="text-xs text-brand-600 dark:text-brand-400 space-y-1">
                <p className="font-medium">{t("cloudBackup.note")}：</p>
                <ul className="list-disc list-inside space-y-0.5 pl-2">
                  <li>{t("cloudBackup.oneDriveNote1")}</li>
                  <li>{t("cloudBackup.oneDriveNote2")}</li>
                  <li>{t("cloudBackup.oneDriveNote3")}</li>
                  <li>{t("cloudBackup.oneDriveNote4")}</li>
                  <li>{t("cloudBackup.oneDriveNote5")}</li>
                </ul>
              </div>
            </div>
          </div>

          {/* Client ID 配置 */}
          <div className="space-y-2">
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">
              {t("cloudBackup.clientId")}
              {(!formData.onedrive_client_id || formData.onedrive_client_id === "26fcab6e-41ea-49ff-8ec9-063983cae3ef")
                && <span className="ml-2 text-xs text-brand-500 dark:text-brand-400">({t("cloudBackup.usingDefault")})</span>}
            </label>
            <input
              type="text"
              name="onedrive_client_id"
              value={formData.onedrive_client_id || ""}
              onChange={handleChange}
              placeholder="26fcab6e-41ea-49ff-8ec9-063983cae3ef (默认)"
              className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-neutral-500 dark:bg-brand-700 dark:text-white font-mono text-sm"
            />
            <p className="text-xs text-brand-500 dark:text-brand-400">
              {t("cloudBackup.customClientIdNote")}
              {" "}
              <a href="https://portal.azure.com/#view/Microsoft_AAD_RegisteredApps/ApplicationsListBlade" target="_blank" rel="noopener noreferrer" className="underline hover:text-brand-600 dark:hover:text-brand-300">Microsoft Entra</a>
              {" "}
              {t("cloudBackup.registerApp")}
            </p>
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">{t("cloudBackup.authStatus")}</label>
            {formData.onedrive_refresh_token
              ? (
                  <div className="flex items-center gap-2">
                    <span className="text-success-600 dark:text-success-400 flex items-center gap-1">
                      <span className="i-mdi-check-circle text-lg" />
                      {t("cloudBackup.authorized")}
                    </span>
                    <button type="button" onClick={() => onChange({ ...formData, onedrive_refresh_token: "" } as appconf.AppConfig)} className="px-2 py-1 text-xs text-error-600 hover:bg-error-100 dark:hover:bg-error-900 rounded">
                      {t("cloudBackup.revokeAuth")}
                    </button>
                  </div>
                )
              : (
                  <div className="space-y-3">
                    <button type="button" onClick={handleOneDriveAuth} disabled={authorizingOneDrive} className="glass-btn-neutral px-3 py-1.5 text-sm bg-neutral-600 text-white rounded-md hover:bg-neutral-700 disabled:opacity-50 flex items-center gap-2">
                      {authorizingOneDrive
                        ? (
                            <>
                              <span className="i-mdi-loading animate-spin" />
                              {t("cloudBackup.waitingForAuth")}
                            </>
                          )
                        : (
                            <>
                              <span className="i-mdi-microsoft" />
                              {t("cloudBackup.authorizeOneDrive")}
                            </>
                          )}
                    </button>
                    {authorizingOneDrive && <p className="text-xs text-brand-500 dark:text-brand-400">{t("cloudBackup.completeAuthInBrowser")}</p>}
                  </div>
                )}
          </div>
          {formData.onedrive_refresh_token && (
            <div className="flex justify-end">
              <button type="button" onClick={handleTestOneDrive} disabled={testingOneDrive} className="glass-btn-neutral px-3 py-1.5 text-sm bg-brand-100 text-brand-700 rounded-md hover:bg-brand-200 dark:bg-brand-700 dark:text-brand-300 dark:hover:bg-brand-600 disabled:opacity-50">
                {testingOneDrive ? t("cloudBackup.testing") : t("cloudBackup.testConnection")}
              </button>
            </div>
          )}
        </div>
      )}

      {/* WebDav 配置 */}
      {formData.cloud_backup_provider === "webdav" && (
        <div className="glass-card space-y-4 p-4 bg-brand-100 dark:bg-brand-800 rounded-lg">
          <h3 className="text-sm font-medium text-brand-800 dark:text-brand-200">{t("cloudBackup.webDavConfig")}</h3>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">{t("cloudBackup.backupPassword")}</label>
            {formData.backup_password
              ? (
                  <div className="space-y-2">
                    <div className="px-3 py-2 bg-brand-100 dark:bg-brand-700 border border-brand-300 dark:border-brand-600 rounded-md text-brand-600 dark:text-brand-300">
                      ********
                    </div>
                    <p className="text-xs text-brand-500 dark:text-brand-400">
                      <span className="text-success-600 dark:text-success-400">
                        ✓ {t("cloudBackup.passwordSet", { userId: formData.backup_user_id?.substring(0, 8) })}
                      </span>
                    </p>
                  </div>
                )
              : (
                  <div className="space-y-2">
                    <button
                      type="button"
                      onClick={() => setShowPasswordModal(true)}
                      className="glass-btn-neutral w-full px-4 py-2 bg-brand-600 hover:bg-brand-700 text-white rounded-md transition-colors flex items-center justify-center gap-2"
                    >
                      <span className="i-mdi-lock-plus text-lg" />
                      {t("cloudBackup.setupBackupPassword")}
                    </button>
                    <p className="text-xs text-brand-500 dark:text-brand-400">
                      {t("cloudBackup.usedForUserId")}
                    </p>
                    <p className="text-xs text-warning-600 dark:text-warning-400">
                      {t("cloudBackup.passwordImportant")}
                    </p>
                  </div>
                )}
          </div>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">{t("cloudBackup.webDavServer")}</label>
            <input type="text" name="webdav_server" value={formData.webdav_server || ""} onChange={handleChange} placeholder="https://webdav.example.com" className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-neutral-500 dark:bg-brand-700 dark:text-white" />
          </div>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">{t("cloudBackup.webDavUsername")}</label>
            <input type="text" name="webdav_username" value={formData.webdav_username || ""} onChange={handleChange} className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-neutral-500 dark:bg-brand-700 dark:text-white" />
          </div>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">{t("cloudBackup.webDavPassword")}</label>
            <input type="password" name="webdav_password" value={formData.webdav_password || ""} onChange={handleChange} className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-neutral-500 dark:bg-brand-700 dark:text-white" />
          </div>
          <div className="flex justify-end">
            <button type="button" onClick={handleTestWebDav} disabled={testingWebDav} className="glass-btn-neutral px-3 py-1.5 text-sm bg-brand-100 text-brand-700 rounded-md hover:bg-brand-200 dark:bg-brand-700 dark:text-brand-300 dark:hover:bg-brand-600 disabled:opacity-50">
              {testingWebDav ? t("cloudBackup.testing") : t("cloudBackup.testConnection")}
            </button>
          </div>
        </div>
      )}

      <div className="space-y-2">
        <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">{t("cloudBackup.retentionCount")}</label>
        <input
          type="number"
          value={formData.cloud_backup_retention || 5}
          onChange={e => onChange({ ...formData, cloud_backup_retention: Number.parseInt(e.target.value) || 20 } as appconf.AppConfig)}
          min={1}
          max={100}
          className="glass-input w-32 px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-neutral-500 dark:bg-brand-700 dark:text-white"
        />
        <p className="text-xs text-brand-500 dark:text-brand-400">{t("cloudBackup.retentionDescription")}</p>
      </div>

      <PasswordInputModal
        isOpen={showPasswordModal}
        onClose={() => setShowPasswordModal(false)}
        onConfirm={handleSetupBackupPassword}
      />
    </div>
  );
}

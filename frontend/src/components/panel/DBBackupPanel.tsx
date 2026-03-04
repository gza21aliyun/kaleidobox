import type { vo } from "../../../wailsjs/go/models";
import { useEffect, useState } from "react";
import toast from "react-hot-toast";
import {
  CreateAndUploadDBBackup,
  DeleteDBBackup,
  GetCloudDBBackups,
  GetDBBackups,
  ScheduleDBRestore,
  ScheduleDBRestoreFromCloud,
  UploadDBBackupToCloud,
} from "../../../wailsjs/go/service/BackupService";
import { SafeQuit } from "../../../wailsjs/go/service/ConfigService";
import { useAppStore } from "../../store";
import { formatFileSize } from "../../utils/size";
import { formatLocalDateTime } from "../../utils/time";
import { ConfirmModal } from "../modal/ConfirmModal";
import i18next from "../../i18n/i18n";
const t = i18next.t;

export function DBBackupPanel() {
  const { config } = useAppStore();
  const [dbBackups, setDbBackups] = useState<vo.DBBackupStatus | null>(null);
  const [cloudDBBackups, setCloudDBBackups] = useState<vo.CloudBackupItem[]>([]);
  const [isBackingUp, setIsBackingUp] = useState(false);
  const [restoringBackup, setRestoringBackup] = useState<string | null>(null);
  const [uploadingBackup, setUploadingBackup] = useState<string | null>(null);
  const [loadingLocal, setLoadingLocal] = useState(true);
  const [loadingCloud, setLoadingCloud] = useState(false);

  // 确认弹窗状态
  const [confirmConfig, setConfirmConfig] = useState<{
    isOpen: boolean;
    title: string;
    message: string;
    type: "danger" | "info";
    onConfirm: () => void;
  }>({
    isOpen: false,
    title: "",
    message: "",
    type: "info",
    onConfirm: () => {},
  });

  const cloudProvider = config?.cloud_backup_provider;

  // 检查云备份是否真正可用
  const cloudEnabled = (() => {
    if (!config?.cloud_backup_enabled) {
      return false;
    }
    // 如果是OneDrive，需要检查是否已授权
    if (cloudProvider === "onedrive") {
      return !!config?.onedrive_refresh_token;
    }
    // S3或其他provider需要backup_user_id
    if (cloudProvider === "s3") {
      return !!config?.backup_user_id;
    }
    return false;
  })();

  const loadDBBackups = async () => {
    setLoadingLocal(true);
    try {
      const backups = await GetDBBackups();
      setDbBackups(backups);
    }
    catch (err) {
      console.error("Failed to load DB backups:", err);
    }
    finally {
      setLoadingLocal(false);
    }
  };

  const loadCloudDBBackups = async () => {
    setLoadingCloud(true);
    try {
      const backups = await GetCloudDBBackups();
      setCloudDBBackups(backups || []);
    }
    catch (err) {
      console.error("Failed to load cloud DB backups:", err);
      setCloudDBBackups([]);
    }
    finally {
      setLoadingCloud(false);
    }
  };

  const handleCreateBackup = async () => {
    if (isBackingUp)
      return;
    setIsBackingUp(true);
    try {
      await CreateAndUploadDBBackup();
      await loadDBBackups();
      if (cloudEnabled)
        await loadCloudDBBackups();
      toast.success(cloudEnabled && config?.auto_upload_db_to_cloud ? t("dbBackup.backupSuccessAndUploaded" ) : t("dbBackup.backupSuccess"));
    }
    catch (err: any) {
      if (err.toString().includes("本地备份成功")) {
        await loadDBBackups();
        toast.success(t("dbBackup.localBackupSuccess"));
        toast.error(err.toString());
      }
      else {
        toast.error(t("dbBackup.backupFailed", { error: err }));
      }
    }
    finally {
      setIsBackingUp(false);
    }
  };

  const handleRestoreDB = async (backupPath: string) => {
    setConfirmConfig({
      isOpen: true,
      title: t("dbBackup.restoreDatabase"),
      message: t("dbBackup.confirmRestore"),
      type: "info",
      onConfirm: async () => {
        setRestoringBackup(backupPath);
        try {
          await ScheduleDBRestore(backupPath);
          toast.success(t("dbBackup.restoreScheduled"));
          setTimeout(() => SafeQuit(), 1500);
        }
        catch (err: any) {
          toast.error(t("dbBackup.scheduleRestoreFailed", { error: err }));
          setRestoringBackup(null);
        }
      },
    });
  };

  const handleDeleteDBBackup = async (backupPath: string) => {
    setConfirmConfig({
      isOpen: true,
      title: t("dbBackup.deleteBackup"),
      message: t("dbBackup.confirmDelete"),
      type: "danger",
      onConfirm: async () => {
        try {
          await DeleteDBBackup(backupPath);
          await loadDBBackups();
          toast.success(t("dbBackup.backupDeleted"));
        }
        catch (err: any) {
          toast.error(t("dbBackup.deleteFailed", { error: err }));
        }
      },
    });
  };

  const handleUploadDBBackup = async (backupPath: string) => {
    setUploadingBackup(backupPath);
    try {
      await UploadDBBackupToCloud(backupPath);
      await loadCloudDBBackups();
      toast.success(t("dbBackup.uploadedToCloud"));
    }
    catch (err: any) {
      toast.error(t("dbBackup.uploadFailed", { error: err }));
    }
    finally {
      setUploadingBackup(null);
    }
  };

  const handleRestoreFromCloud = async (cloudKey: string) => {
    setConfirmConfig({
      isOpen: true,
      title: t("dbBackup.restoreFromCloud"),
      message: t("dbBackup.confirmRestoreFromCloud"),
      type: "info",
      onConfirm: async () => {
        setRestoringBackup(cloudKey);
        try {
          await ScheduleDBRestoreFromCloud(cloudKey);
          toast.success(t("dbBackup.restoreScheduled"));
          setTimeout(() => SafeQuit(), 1500);
        }
        catch (err: any) {
          toast.error(t("dbBackup.scheduleRestoreFailed", { error: err }));
          setRestoringBackup(null);
        }
      },
    });
  };

  useEffect(() => {
    loadDBBackups();
  }, []);

  // 云存储提供商变化时自动刷新云备份列表
  useEffect(() => {
    if (cloudEnabled) {
      loadCloudDBBackups();
    }
    else {
      setCloudDBBackups([]);
    }
  }, [cloudEnabled, cloudProvider]);

  const isDisabled = restoringBackup !== null || uploadingBackup !== null || isBackingUp;

  return (
    <div className="space-y-6">
      {/* 备份操作区 */}
      <div className="glass-card bg-white dark:bg-brand-800 p-6 rounded-lg shadow-sm">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-lg font-semibold text-brand-900 dark:text-white">{t("dbBackup.databaseBackup")}</h3>
            <p className="text-sm text-brand-500 dark:text-brand-400 mt-1">
              {t("dbBackup.backupDescription")}
            </p>
          </div>
          <button
            onClick={handleCreateBackup}
            disabled={isDisabled}
            className="glass-btn-neutral px-4 py-2 bg-neutral-600 text-white rounded-md hover:bg-neutral-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
          >
            {isBackingUp && <div className="i-mdi-loading animate-spin" />}
            {isBackingUp ? t("dbBackup.backingUp") : t("dbBackup.backupNow")}
          </button>
        </div>
        {dbBackups?.last_backup_time && (
          <p className="text-xs text-brand-500 dark:text-brand-400">
            {t("dbBackup.lastBackup")}:
            {" "}
            {formatLocalDateTime(dbBackups.last_backup_time, config?.time_zone)}
          </p>
        )}
      </div>

      {/* 本地备份列表 */}
      <div className="glass-card bg-white dark:bg-brand-800 p-6 rounded-lg shadow-sm">
        <div className="flex items-center gap-2 mb-4">
          <h3 className="text-lg font-semibold text-brand-900 dark:text-white">{t("dbBackup.localBackups")}</h3>
          {config?.auto_backup_db && (
            <span className="px-2 py-0.5 text-xs font-medium bg-success-100 text-success-700 dark:bg-success-900/30 dark:text-success-400 rounded-full flex items-center gap-1">
              <div className="i-mdi-shield-check text-sm" />
              {t("dbBackup.autoBackupEnabled")}
            </span>
          )}
        </div>
        {loadingLocal
          ? (
              <div className="flex justify-center py-8">
                <div className="i-mdi-loading animate-spin text-2xl text-brand-500" />
              </div>
            )
          : dbBackups?.backups && dbBackups.backups.length > 0
            ? (
                <div className="space-y-3">
                  {dbBackups.backups.map(backup => (
                    <div
                      key={backup.path}
                      className="data-glass:bg-white/1 data-glass:dark:bg-black/1 flex items-center justify-between p-4 bg-brand-50 dark:bg-brand-700 rounded-lg"
                    >
                      <div className="flex items-center gap-4">
                        <div className="i-mdi-database text-2xl text-brand-500" />
                        <div>
                          <div className="font-medium text-brand-900 dark:text-white">
                            {formatLocalDateTime(backup.created_at, config?.time_zone)}
                          </div>
                          <div className="text-sm text-brand-500">
                            {t("dbBackup.size")}:
                            {formatFileSize(backup.size)}
                          </div>
                        </div>
                      </div>
                      <div className="flex gap-2">
                        {cloudEnabled && (
                          <button
                            onClick={() => handleUploadDBBackup(backup.path)}
                            disabled={isDisabled}
                            title={t("dbBackup.uploadToCloud")}
                            className="p-2 text-neutral-600 hover:bg-neutral-100 dark:hover:bg-neutral-900 rounded transition-colors disabled:opacity-50"
                          >
                            {uploadingBackup === backup.path
                              ? (
                                  <div className="i-mdi-loading text-xl animate-spin" />
                                )
                              : (
                                  <div className="i-mdi-cloud-upload text-xl" />
                                )}
                          </button>
                        )}
                        <button
                          onClick={() => handleRestoreDB(backup.path)}
                          disabled={isDisabled}
                          title={t("dbBackup.restoreThisBackup")}
                          className="p-2 text-success-600 hover:bg-success-100 dark:hover:bg-success-900 rounded transition-colors disabled:opacity-50"
                        >
                          {restoringBackup === backup.path
                            ? (
                                <div className="i-mdi-loading text-xl animate-spin" />
                              )
                            : (
                                <div className="i-mdi-backup-restore text-xl" />
                              )}
                        </button>
                        <button
                          onClick={() => handleDeleteDBBackup(backup.path)}
                          disabled={isDisabled}
                          title={t("dbBackup.deleteBackup")}
                          className="p-2 text-error-600 hover:bg-error-100 dark:hover:bg-error-900 rounded transition-colors disabled:opacity-50"
                        >
                          <div className="i-mdi-delete text-xl" />
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )
            : (
                <div className="text-center py-8 text-brand-500">{t("dbBackup.noLocalBackups")}</div>
              )}
      </div>

      {/* 云端备份列表 */}
      {cloudEnabled && (
        <div className="glass-card bg-white dark:bg-brand-800 p-6 rounded-lg shadow-sm">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-brand-900 dark:text-white flex items-center gap-2">
              <div className="i-mdi-cloud text-xl text-neutral-500" />
              {t("dbBackup.cloudBackups")}
            </h3>
            <button
              onClick={loadCloudDBBackups}
              disabled={loadingCloud || isDisabled}
              title={t("dbBackup.refreshCloudBackups")}
              className="p-2 text-brand-600 hover:bg-brand-100 dark:hover:bg-brand-700 rounded transition-colors disabled:opacity-50"
            >
              <div className={`i-mdi-refresh text-xl ${loadingCloud ? "animate-spin" : ""}`} />
            </button>
          </div>
          {loadingCloud
            ? (
                <div className="flex justify-center py-8">
                  <div className="i-mdi-loading animate-spin text-2xl text-brand-500" />
                </div>
              )
            : cloudDBBackups.length > 0
              ? (
                  <div className="space-y-3">
                    {cloudDBBackups.map(backup => (
                      <div
                        key={backup.key}
                        className="data-glass:bg-white/1 data-glass:dark:bg-black/1 flex items-center justify-between p-4 bg-neutral-50 dark:bg-neutral-900/30 rounded-lg"
                      >
                        <div className="flex items-center gap-4">
                          <div className="i-mdi-cloud-check text-2xl text-neutral-500" />
                          <div>
                            <div className="font-medium text-brand-900 dark:text-white">
                              {backup.name || formatLocalDateTime(backup.created_at, config?.time_zone)}
                            </div>
                            <div className="text-sm text-brand-500">
                              {formatLocalDateTime(backup.created_at, config?.time_zone)}
                            </div>
                          </div>
                        </div>
                        <div className="flex gap-2">
                          <button
                            onClick={() => handleRestoreFromCloud(backup.key)}
                            disabled={isDisabled}
                            title={t("dbBackup.restoreFromCloud")}
                            className="p-2 text-success-600 hover:bg-success-100 dark:hover:bg-success-900 rounded transition-colors disabled:opacity-50"
                          >
                            {restoringBackup === backup.key
                              ? (
                                  <div className="i-mdi-loading text-xl animate-spin" />
                                )
                              : (
                                  <div className="i-mdi-cloud-download text-xl" />
                                )}
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                )
              : (
                  <div className="text-center py-8 text-brand-500">{t("dbBackup.noCloudBackups")}</div>
                )}
        </div>
      )}

      {/* 云备份未配置提示 */}
      {!cloudEnabled && (
        <div className="bg-brand-50 dark:bg-brand-800 p-4 rounded-lg border border-brand-200 dark:border-brand-700">
          <div className="flex items-center gap-3">
            <div className="i-mdi-cloud-off-outline text-2xl text-brand-400" />
            <div>
              <div className="font-medium text-brand-700 dark:text-brand-300">{t("dbBackup.cloudBackupNotEnabled")}</div>
              <div className="text-sm text-brand-500">{t("dbBackup.cloudBackupHint")}</div>
            </div>
          </div>
        </div>
      )}

      <ConfirmModal
        isOpen={confirmConfig.isOpen}
        title={confirmConfig.title}
        message={confirmConfig.message}
        type={confirmConfig.type}
        onClose={() => setConfirmConfig({ ...confirmConfig, isOpen: false })}
        onConfirm={confirmConfig.onConfirm}
      />
    </div>
  );
}

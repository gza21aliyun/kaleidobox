import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { createRoute } from "@tanstack/react-router";
import { Route as rootRoute } from "./__root";
import { ListDownloadedFiles, ExtractItem, ExtractFolder, MountISO, InstallGame, OpenFolder, DeleteItem, DeleteExtractedFolder } from "../../wailsjs/go/service/DownloadedFilesService";
import { GameSearchModal } from "../components/modal/GameSearchModal";
import type { service } from "../../wailsjs/go/models";
import { OpenLocalPath } from "../../wailsjs/go/service/GameService";

interface DownloadedFile extends service.DownloadedFile {
  selected: boolean;
}

export default function DownloadedFiles() {
  const { t } = useTranslation();
  const [items, setItems] = useState<DownloadedFile[]>([]);
  const [selectedItems, setSelectedItems] = useState<string[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [showExtract, setShowExtract] = useState(true);
  const [showInstall, setShowInstall] = useState(true);
  const [showImport, setShowImport] = useState(true);
  const [md5AsFolder, setMd5AsFolder] = useState(false);
  const [showConfirmModal, setShowConfirmModal] = useState(false);
  const [confirmModalType, setConfirmModalType] = useState<string>("");
  const [confirmModalItem, setConfirmModalItem] = useState<DownloadedFile | null>(null);
  const [installMethod, setInstallMethod] = useState<string>("name");
  const [errorMessage, setErrorMessage] = useState<string>("");
  const [searchModalItem, setSearchModalItem] = useState<DownloadedFile | null>(null);
  const [isExecuting, setIsExecuting] = useState(false);
  const [currentExecutingName, setCurrentExecutingName] = useState("");

  const loadItems = async () => {
    setIsLoading(true);
    try {
      const result = await ListDownloadedFiles();
      setItems(result.map(item => ({ ...item, selected: false })));
      setSelectedItems([]);
    } catch (err) {
      console.error("Failed to load items:", err);
      setErrorMessage(err instanceof Error ? err.message : "加载失败");
    }
    setIsLoading(false);
  };

  useEffect(() => {
    loadItems();
  }, []);

  const handleSelectAll = () => {
    if (selectedItems.length === items.length) {
      setItems(items.map(item => ({ ...item, selected: false })));
      setSelectedItems([]);
    } else {
      setItems(items.map(item => ({ ...item, selected: true })));
      setSelectedItems(items.map(item => item.id));
    }
  };

  const handleSelectItem = (itemId: string) => {
    const item = items.find(i => i.id === itemId);
    if (!item) return;

    const newSelected = item.selected
      ? selectedItems.filter(id => id !== itemId)
      : [...selectedItems, itemId];

    setItems(items.map(i => ({ ...i, selected: newSelected.includes(i.id) })));
    setSelectedItems(newSelected);
  };

  const handleClearAll = () => {
    setItems(items.map(item => ({ ...item, selected: false })));
    setSelectedItems([]);
  };

  const handleExecute = async () => {
    setIsExecuting(true);
    setCurrentExecutingName("");
    try {
      for (const itemId of selectedItems) {
        const item = items.find(i => i.id === itemId);
        if (!item) continue;

        if (showExtract && !item.is_extracted) {
          setCurrentExecutingName(item.name);
          await handleExtract(item);
        }
        if (showInstall && !item.is_installed) {
          setCurrentExecutingName(item.name);
          try {
            await InstallGame(item.path, md5AsFolder ? "md5" : "name");
          } catch (err) {
            console.error("Install failed:", err);
            setErrorMessage(err instanceof Error ? err.message : "安装失败");
          }
        }
        if (showImport && !item.is_imported) {
          setCurrentExecutingName(item.name);
          await handleImport(item);
        }
      }
      await loadItems();
    } finally {
      setIsExecuting(false);
      setCurrentExecutingName("");
    }
  };

  const handleExtract = async (item: DownloadedFile) => {
    setCurrentExecutingName(item.name);
    setIsExecuting(true);
    try {
      if (item.is_folder) {
        await ExtractFolder(item.path);
      } else {
        await ExtractItem(item.path);
      }
      // 解压成功后更新状态
      setItems(items.map(i =>
        i.id === item.id ? { ...i, is_extracted: true } : i
      ));
    } catch (err) {
      console.error("Extract failed:", err);
      setErrorMessage(err instanceof Error ? err.message : "解压失败");
    } finally {
      setIsExecuting(false);
      setCurrentExecutingName("");
    }
  };

  const handleMount = async (item: DownloadedFile) => {
    if (!item.iso_file_path) return;
    try {
      await MountISO(item.iso_file_path);
    } catch (err) {
      console.error("Mount failed:", err);
      setErrorMessage(err instanceof Error ? err.message : "装载失败");
    }
  };

  const handleInstall = async (item: DownloadedFile) => {
    if (item.iso_count === 1) {
      setConfirmModalType("install_iso");
      setConfirmModalItem(item);
      setShowConfirmModal(true);
    } else if (item.iso_count === 0) {
      setConfirmModalType("install");
      setConfirmModalItem(item);
      setShowConfirmModal(true);
    }
  };

  const handleConfirmInstall = async () => {
    if (!confirmModalItem) return;

    try {
      await InstallGame(confirmModalItem.path, md5AsFolder ? "md5" : installMethod);
    } catch (err) {
      console.error("Install failed:", err);
      setErrorMessage(err instanceof Error ? err.message : "安装失败");
    }

    setShowConfirmModal(false);
    setConfirmModalItem(null);
    await loadItems();
  };

  const handleImport = async (item: DownloadedFile) => {
    console.log("Import:", item.name);
  };

  const handleOpen = async (item: DownloadedFile) => {
    try {
      await OpenLocalPath(item.path);
    } catch (err) {
      console.error("Open failed:", err);
    }
  };

  const handleOpenSearch = (item: DownloadedFile) => {
    setSearchModalItem(item);
  };

  const handleDelete = async (item: DownloadedFile) => {
    if (!confirm(t("downloadedFiles.confirmDelete", { name: item.name }))) {
      return;
    }
    try {
      await DeleteItem(item.path);
      await loadItems();
    } catch (err) {
      console.error("Delete failed:", err);
      setErrorMessage(err instanceof Error ? err.message : "删除失败");
    }
  };

  const handleDeleteExtracted = async (item: DownloadedFile) => {
    try {
      const result = await DeleteExtractedFolder(item.path, item.name);
      if (result && result.has_archive && result.archive_item) {
        // 有同名压缩包，更新当前单元的信息（但保持 id 和 selected 状态）
        setItems(items.map(i =>
          i.id === item.id
            ? {
                ...i,
                name: result.archive_item.name,
                path: result.archive_item.path,
                is_folder: false,
                is_extracted: false,
                size: result.archive_item.size,
                inner_items: result.archive_item.inner_items,
                has_numeric_name: result.archive_item.has_numeric_name,
                contains_iso: result.archive_item.contains_iso,
                iso_count: result.archive_item.iso_count,
                iso_file_path: result.archive_item.iso_file_path,
                // 保持选中状态
              }
            : i
        ));
      } else {
        // 没有同名压缩包，只更新状态
        setItems(items.map(i =>
          i.id === item.id ? { ...i, is_extracted: false } : i
        ));
      }
    } catch (err) {
      console.error("Delete extracted folder failed:", err);
      setErrorMessage(err instanceof Error ? err.message : "删除解压文件夹失败");
    }
  };

  const getDisplayTitle = (item: DownloadedFile) => {
    if (item.has_numeric_name && item.longest_zip_name) {
      return item.longest_zip_name;
    }
    const ext = item.is_folder ? "" : (item.name.match(/\.[^.]+$/)?.[0] || "");
    return item.name.replace(new RegExp(`${ext}$`), "");
  };

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return bytes + " B";
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB";
    if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + " MB";
    return (bytes / (1024 * 1024 * 1024)).toFixed(1) + " GB";
  };

  const canInstall = (item: DownloadedFile) => {
    return item.is_folder && item.iso_count <= 1;
  };

  const canMount = (item: DownloadedFile) => {
    return item.is_folder && item.iso_count === 1;
  };

  const isDownloadingItem = (item: DownloadedFile) => {
    return item.is_downloading;
  };

  return (
    <div className="p-6 h-full overflow-auto">
      {errorMessage && (
        <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          {errorMessage}
          <button onClick={() => setErrorMessage("")} className="float-right">×</button>
        </div>
      )}

      <div className="flex flex-col h-full">
        {/* 页面标题 */}
        <h1 className="text-2xl font-bold text-brand-900 dark:text-white mb-2">
          {t("nav.downloadedFiles")}
        </h1>

        {/* 提示信息 */}
        <p className="text-sm text-brand-500 dark:text-brand-400 mb-4">
          {t("downloadedFiles.tempStateHint")}
        </p>

        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <button
              onClick={handleSelectAll}
              className="px-3 py-1.5 text-sm bg-brand-100 hover:bg-brand-200 dark:bg-brand-700 dark:hover:bg-brand-600 rounded-md"
            >
              {t("downloadedFiles.selectAll")}
            </button>
            <button
              onClick={handleClearAll}
              className="px-3 py-1.5 text-sm bg-brand-100 hover:bg-brand-200 dark:bg-brand-700 dark:hover:bg-brand-600 rounded-md"
            >
              {t("downloadedFiles.clear")}
            </button>
          </div>

          <div className="flex items-center gap-3">
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={showExtract}
                onChange={(e) => setShowExtract(e.target.checked)}
                className="rounded border-brand-300 text-brand-600 focus:ring-neutral-500"
              />
              {t("downloadedFiles.extract")}
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={showInstall}
                onChange={(e) => setShowInstall(e.target.checked)}
                className="rounded border-brand-300 text-brand-600 focus:ring-neutral-500"
              />
              {t("downloadedFiles.install")}
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={showImport}
                onChange={(e) => setShowImport(e.target.checked)}
                className="rounded border-brand-300 text-brand-600 focus:ring-neutral-500"
              />
              {t("downloadedFiles.import")}
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={md5AsFolder}
                onChange={(e) => setMd5AsFolder(e.target.checked)}
                className="rounded border-brand-300 text-brand-600 focus:ring-neutral-500"
              />
              {t("downloadedFiles.md5AsFolder")}
            </label>
            <button
              onClick={() => {
                selectedItems.forEach(id => {
                  const item = items.find(i => i.id === id);
                  if (item) handleDelete(item);
                });
              }}
              disabled={selectedItems.length === 0}
              className="px-3 py-1.5 text-sm bg-red-500 hover:bg-red-600 text-white rounded-md disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {t("downloadedFiles.delete")}
            </button>
            <button
              onClick={handleExecute}
              disabled={selectedItems.length === 0 || isExecuting}
              className="px-3 py-1.5 text-sm bg-brand-600 hover:bg-brand-700 text-white rounded-md disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {t("downloadedFiles.execute")}
            </button>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={loadItems}
              disabled={isLoading}
              className="p-2 text-sm bg-brand-100 hover:bg-brand-200 dark:bg-brand-700 dark:hover:bg-brand-600 rounded-md disabled:opacity-50"
              title={t("downloadedFiles.refresh")}
            >
              <div className="i-mdi-refresh text-lg" />
            </button>
          </div>
        </div>

        <div className="flex-1 overflow-auto space-y-2 relative">
          {isExecuting && (
            <div className="absolute inset-0 z-10 flex items-center justify-center bg-black/30 backdrop-blur-sm">
              <div className="flex flex-col items-center gap-2 bg-white dark:bg-brand-800 p-4 rounded-lg shadow-lg">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-brand-600"></div>
                <span className="text-sm text-brand-600">{t("downloadedFiles.executing")}</span>
                {currentExecutingName && (
                  <span className="text-xs text-brand-500 dark:text-brand-400 max-w-xs truncate" title={currentExecutingName}>
                    {currentExecutingName}
                  </span>
                )}
              </div>
            </div>
          )}

          {isLoading ? (
            <div className="flex items-center justify-center h-32">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-brand-600"></div>
            </div>
          ) : items.length === 0 ? (
            <div className="text-center py-12 text-brand-500">
              {t("downloadedFiles.empty")}
            </div>
          ) : (
            items.map(item => (
              <div
                key={item.id}
                className={`p-4 border rounded-lg ${
                  item.selected
                    ? "border-brand-500 bg-brand-50 dark:bg-brand-700"
                    : "border-brand-200 dark:border-brand-700 bg-white dark:bg-brand-800"
                }`}
              >
                <div className="flex items-start gap-3">
                  <input
                    type="checkbox"
                    checked={item.selected}
                    onChange={() => handleSelectItem(item.id)}
                    className="mt-1 rounded border-brand-300 text-brand-600 focus:ring-neutral-500"
                  />

                  <div className="flex-1 min-w-0 overflow-hidden">
                    <div className="flex items-center gap-2 min-w-0">
                      <div className={`text-lg ${item.is_folder ? "i-mdi-folder" : "i-mdi-archive"} flex-shrink-0`} />
                      <span className="font-medium truncate min-w-0" title={getDisplayTitle(item)}>{getDisplayTitle(item)}</span>
                      {item.has_numeric_name && (
                        <span className="text-xs text-brand-500 flex-shrink-0">({item.name})</span>
                      )}
                    </div>

                    {/* 第二行：类型、大小、下载状态 + 按钮栏 */}
                    <div className="flex items-center justify-between gap-2 mt-1">
                      <div className="flex items-center gap-3 text-xs text-brand-500">
                        <span>{item.is_folder ? t("downloadedFiles.folder") : t("downloadedFiles.archive")}</span>
                        <span>{formatSize(item.size)}</span>
                        {item.is_downloading && (
                          <span className="text-orange-500 flex items-center gap-1">
                            <div className="i-mdi-download animate-pulse" />
                            {t("downloadedFiles.downloading")}
                          </span>
                        )}
                      </div>

                      {/* 按钮栏 */}
                      <div className="flex gap-1 flex-wrap">
                      {/* 解压按钮 */}
                      {item.is_downloading ? (
                        <button
                          disabled
                          className="px-2 py-1 text-xs bg-brand-100 dark:bg-brand-700 text-brand-400 rounded cursor-not-allowed"
                        >
                          {t("downloadedFiles.extract")}
                        </button>
                      ) : item.is_extracted ? (
                        <>
                          <button
                            disabled
                            className="px-2 py-1 text-xs bg-brand-100 dark:bg-brand-700 text-brand-400 rounded cursor-not-allowed"
                          >
                            {t("downloadedFiles.extracted")}
                          </button>
                          <button
                            onClick={() => handleDeleteExtracted(item)}
                            disabled={isExecuting}
                            className="px-2 py-1 text-xs bg-red-400 hover:bg-red-500 text-white rounded disabled:opacity-50"
                            title={t("downloadedFiles.deleteExtracted")}
                          >
                            {t("downloadedFiles.deleteExtracted")}
                          </button>
                        </>
                      ) : (
                        <button
                          onClick={() => handleExtract(item)}
                          disabled={isExecuting}
                          className="px-2 py-1 text-xs bg-blue-500 hover:bg-blue-600 text-white rounded disabled:opacity-50"
                        >
                          {t("downloadedFiles.extract")}
                        </button>
                      )}

                      {/* 安装按钮 */}
                      {item.is_downloading ? (
                        <button
                          disabled
                          className="px-2 py-1 text-xs bg-brand-100 dark:bg-brand-700 text-brand-400 rounded cursor-not-allowed"
                        >
                          {t("downloadedFiles.install")}
                        </button>
                      ) : !canInstall(item) ? (
                        <button
                          disabled
                          className="px-2 py-1 text-xs bg-brand-100 dark:bg-brand-700 text-brand-400 rounded cursor-not-allowed"
                        >
                          {t("downloadedFiles.install")}
                        </button>
                      ) : item.is_installed ? (
                        <button
                          disabled
                          className="px-2 py-1 text-xs bg-brand-100 dark:bg-brand-700 text-brand-400 rounded cursor-not-allowed"
                        >
                          {t("downloadedFiles.installed")}
                        </button>
                      ) : (
                        <button
                          onClick={() => handleInstall(item)}
                          className="px-2 py-1 text-xs bg-green-500 hover:bg-green-600 text-white rounded"
                        >
                          {t("downloadedFiles.install")}
                        </button>
                      )}

                      {/* 导入按钮 */}
                      {item.is_downloading ? (
                        <button
                          disabled
                          className="px-2 py-1 text-xs bg-brand-100 dark:bg-brand-700 text-brand-400 rounded cursor-not-allowed"
                        >
                          {t("downloadedFiles.import")}
                        </button>
                      ) : item.is_imported ? (
                        <button
                          disabled
                          className="px-2 py-1 text-xs bg-brand-100 dark:bg-brand-700 text-brand-400 rounded cursor-not-allowed"
                        >
                          {t("downloadedFiles.imported")}
                        </button>
                      ) : !item.is_installed ? (
                        <button
                          disabled
                          className="px-2 py-1 text-xs bg-brand-100 dark:bg-brand-700 text-brand-400 rounded cursor-not-allowed"
                        >
                          {t("downloadedFiles.import")}
                        </button>
                      ) : (
                        <button
                          onClick={() => handleImport(item)}
                          className="px-2 py-1 text-xs bg-purple-500 hover:bg-purple-600 text-white rounded"
                        >
                          {t("downloadedFiles.import")}
                        </button>
                      )}

                      {/* 打开按钮 */}
                      <button
                        onClick={() => handleOpen(item)}
                        className="px-2 py-1 text-xs bg-brand-500 hover:bg-brand-600 text-white rounded"
                      >
                        {t("downloadedFiles.open")}
                      </button>

                      {/* 搜索按钮 */}
                      <button
                        onClick={() => handleOpenSearch(item)}
                        className="px-2 py-1 text-xs bg-brand-100 hover:bg-brand-200 dark:bg-brand-700 dark:hover:bg-brand-600 text-brand-700 dark:text-brand-300 rounded"
                      >
                        {t("downloadedFiles.search")}
                      </button>

                      {/* 装载按钮 */}
                      {!item.is_downloading && canMount(item) && (
                        <button
                          onClick={() => handleMount(item)}
                          className="px-2 py-1 text-xs bg-orange-500 hover:bg-orange-600 text-white rounded"
                        >
                          {t("downloadedFiles.mount")}
                        </button>
                      )}

                      {/* 删除按钮 */}
                      <button
                        onClick={() => handleDelete(item)}
                        className="px-2 py-1 text-xs bg-red-500 hover:bg-red-600 text-white rounded"
                      >
                        {t("downloadedFiles.delete")}
                      </button>
                    </div>
                    </div>

                    {/* inner_items 在第三行 */}
                    {item.inner_items.length > 0 && (
                      <div className="flex flex-wrap gap-1 text-xs text-brand-400 mt-1">
                        {item.inner_items.map((inner, idx) => (
                          <span key={idx} className="px-1.5 py-0.5 bg-brand-100 dark:bg-brand-700 rounded">
                            {inner}
                          </span>
                        ))}
                        {item.inner_items.length > 5 && (
                          <span className="px-1.5 py-0.5 text-brand-500">+{item.inner_items.length - 5}</span>
                        )}
                      </div>
                    )}
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      {showConfirmModal && confirmModalItem && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-brand-800 rounded-lg p-6 w-96">
            <h3 className="text-lg font-semibold mb-4">
              {confirmModalType === "install_iso"
                ? t("downloadedFiles.installIsoTitle")
                : t("downloadedFiles.installTitle")}
            </h3>
            {confirmModalType === "install_iso" && (
              <p className="mb-4 text-brand-600">
                {t("downloadedFiles.installIsoMessage")}
              </p>
            )}
            <div className="space-y-3 mb-4">
              <label className="flex items-center gap-2">
                <input
                  type="radio"
                  name="installMethod"
                  value="name"
                  checked={installMethod === "name"}
                  onChange={(e) => setInstallMethod(e.target.value)}
                />
                {t("downloadedFiles.installByName")}
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="radio"
                  name="installMethod"
                  value="uuid"
                  checked={installMethod === "uuid"}
                  onChange={(e) => setInstallMethod(e.target.value)}
                />
                {t("downloadedFiles.installByUuid")}
              </label>
            </div>
            <div className="flex justify-end gap-2">
              <button
                onClick={() => {
                  setShowConfirmModal(false);
                  setConfirmModalItem(null);
                }}
                className="px-4 py-2 bg-brand-200 dark:bg-brand-700 rounded hover:bg-brand-300 dark:hover:bg-brand-600"
              >
                {t("common.cancel")}
              </button>
              <button
                onClick={handleConfirmInstall}
                className="px-4 py-2 bg-brand-600 text-white rounded hover:bg-brand-700"
              >
                {t("downloadedFiles.confirmInstall")}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 游戏搜索弹窗 */}
      {searchModalItem && (
        <GameSearchModal
          itemName={getDisplayTitle(searchModalItem)}
          onClose={() => setSearchModalItem(null)}
        />
      )}
    </div>
  );
}

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/downloaded_files",
  component: DownloadedFiles,
});

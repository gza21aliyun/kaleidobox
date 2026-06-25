import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { createRoute } from "@tanstack/react-router";
import { Route as rootRoute } from "./__root";
import { ListDownloadedFiles, ExtractItem, ExtractFolder, MountISO, InstallGame, OpenFolder, DeleteItem, DeleteExtractedFolder, DeleteInstalledGame, ExtractGameNameFromDLSite, CreateDownloadedFile, RefreshDownloadedFile, ExtractArchivesInFolder, SaveImportedID, ScanFolderForExecutables, GetGameNameMD5 } from "../../wailsjs/go/service/DownloadedFilesService";
import { GameSearchModal } from "../components/modal/GameSearchModal";
import { BatchImportModal } from "../components/modal/BatchImportModal";
import type { service, models } from "../../wailsjs/go/models";
import { OpenLocalPath } from "../../wailsjs/go/service/GameService";
import { useAppStore } from "../store";
import { vo } from "../../wailsjs/go/models";

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
  const [directIsoInstall, setDirectIsoInstall] = useState(false);
  const [showConfirmModal, setShowConfirmModal] = useState(false);
  const [confirmModalType, setConfirmModalType] = useState<string>("");
  const [confirmModalItem, setConfirmModalItem] = useState<DownloadedFile | null>(null);
  const [installMethod, setInstallMethod] = useState<string>("name");
  const [errorMessage, setErrorMessage] = useState<string>("");
  const [searchModalItem, setSearchModalItem] = useState<DownloadedFile | null>(null);
  const [isExecuting, setIsExecuting] = useState(false);
  const [currentExecutingName, setCurrentExecutingName] = useState("");
  const [batchImportModalItem, setBatchImportModalItem] = useState<DownloadedFile | null>(null);
  const [batchImportCandidates, setBatchImportCandidates] = useState<vo.BatchImportCandidate[]>([]);
  const [runGameModalItem, setRunGameModalItem] = useState<DownloadedFile | null>(null);
  const [runGameExecutables, setRunGameExecutables] = useState<string[]>([]);
  
  const config = useAppStore(state => state.config);
  const fetchConfig = useAppStore(state => state.fetchConfig);

  useEffect(() => {
    fetchConfig();
  }, [fetchConfig]);

  useEffect(() => {
    if (config && !config.seven_zip_path) {
      setErrorMessage(t("downloadedFiles.noSevenZipPath"));
    }
  }, [config, t]);

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

        // 解压步骤
        if (showExtract && !item.is_extracted) {
          setCurrentExecutingName(item.name);
          await handleExtract(item);
        }
        
        // 安装步骤：需要已解压，且如果镜像文件直接安装开关关闭，则有iso的单元不执行安装
        const hasIso = item.iso_items && item.iso_items.length > 0;
        const shouldInstall = showInstall && !item.is_installed && (directIsoInstall || !hasIso);
        
        if (shouldInstall) {
          setCurrentExecutingName(item.name);
          try {
            await InstallGame(item, md5AsFolder ? "md5" : "name");
          } catch (err) {
            console.error("Install failed:", err);
            setErrorMessage(err instanceof Error ? err.message : "安装失败");
          }
        }
        
        // 导入步骤：需要已安装，且如果镜像文件直接安装开关关闭，则有iso的单元不执行导入
        const shouldImport = showImport && !item.is_imported && item.is_installed && (directIsoInstall || !hasIso);
        
        if (shouldImport) {
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
      if (item.type == 1) {
        await ExtractArchivesInFolder(item.path);
        const extractedFolder = await RefreshDownloadedFile(item);
        setItems(items.map(i =>
          i.id === item.id ? { ...i, is_extracted: true, extracted_paths: extractedFolder.extracted_paths, 
            iso_items: extractedFolder.iso_items, extracted_game_path: extractedFolder.extracted_game_path, inner_items: extractedFolder.inner_items } : i
        ));
      } else {
        await ExtractItem(item.path);
        const arc = await RefreshDownloadedFile(item);
        console.log("file extracted", arc);
        setItems(items.map(i =>
          i.id === item.id ? { ...i, is_extracted: true, inner_items: arc.inner_items, iso_items: arc.iso_items, 
            extracted_game_path: arc.extracted_game_path, extracted_paths: arc.extracted_paths } : i
        ));
      }
    } catch (err) {
      console.error("Extract failed:", err);
      setErrorMessage(err instanceof Error ? err.message : "解压失败");
    } finally {
      setIsExecuting(false);
      setCurrentExecutingName("");
    }
  };

  const handleMount = async (item: DownloadedFile) => {
    if (item.iso_items.length < 1) return;
    try {
      await MountISO(item.iso_items[0]);
    } catch (err) {
      console.error("Mount failed:", err);
      setErrorMessage(err instanceof Error ? err.message : "装载失败");
    }
  };

  const handleDirectMount = async (iso_path: string) => {
    try {
      await MountISO(iso_path);
    } catch (err) {
      console.error("Mount failed:", err);
      setErrorMessage(err instanceof Error ? err.message : "装载失败：" + err);
    }
  };

  const handleInstall = async (item: DownloadedFile) => {
    if (item.iso_items.length === 1) {
      setConfirmModalType("install_iso");
      setConfirmModalItem(item);
      setShowConfirmModal(true);
    } else if (item.iso_items.length === 0) {
      setConfirmModalType("install");
      setConfirmModalItem(item);
      setShowConfirmModal(true);
    }
  };

  const handleConfirmInstall = async () => {
    if (!confirmModalItem) return;

    // 立即显示执行中动画
    setIsExecuting(true);
    setCurrentExecutingName(confirmModalItem.name);
    // 关闭确认弹窗
    setShowConfirmModal(false);
    setConfirmModalItem(null);

    try {
      const installedPath = await InstallGame(confirmModalItem, md5AsFolder ? "md5" : installMethod);
      // 更新单元状态，包含安装路径
      setItems(items.map(i =>
        i.id === confirmModalItem.id 
          ? { ...i, is_installed: true, installed_path: installedPath } 
          : i
      ));
    } catch (err) {
      console.error("Install failed:", err);
      setErrorMessage(err instanceof Error ? err.message : "安装失败");
    } finally {
      // 隐藏执行中动画
      setIsExecuting(false);
      setCurrentExecutingName("");
    }


    // await loadItems();
  };

  const handleImport = async (item: DownloadedFile) => {
    console.log("Import:", item.name);
    
    // 准备导入数据
    const config = useAppStore.getState().config;
    const installFolder = config?.game_install_folder;
    
    if (!installFolder) {
      setErrorMessage("游戏安装文件夹未配置");
      return;
    }
    
    // 游戏名
    const gameName = item.game_name || item.name;
    
    // 尝试两种安装路径：游戏名 或 游戏名MD5
    const pathByName = `${installFolder}/${gameName}`;
    const md5Name = await GetGameNameMD5(gameName);
    const pathByMD5 = `${installFolder}/${md5Name}`;
    
    // 扫描安装文件夹下的游戏目录查找可执行文件
    let executables: string[] = [];
    let selectedExe = "";
    let installedGamePath = "";
    
    // 先尝试游戏名路径
    try {
      const result = await ScanFolderForExecutables(pathByName) as [string[], string];
      if (result && result[0] && result[0].length > 0) {
        executables = result[0] || [];
        selectedExe = result[1] || "";
        installedGamePath = pathByName;
      }
    } catch (err) {
      console.error("扫描游戏名路径失败:", err);
    }
    
    // 如果没找到，尝试MD5路径
    if (executables.length === 0) {
      try {
        const result = await ScanFolderForExecutables(pathByMD5) as [string[], string];
        if (result && result[0] && result[0].length > 0) {
          executables = result[0] || [];
          selectedExe = result[1] || "";
          installedGamePath = pathByMD5;
        }
      } catch (err) {
        console.error("扫描MD5路径失败:", err);
      }
    }
    
    // 如果都没找到，使用游戏名路径（让用户手动选择）
    if (installedGamePath === "") {
      installedGamePath = pathByName;
    }
    
    // 创建 BatchImportCandidate
    const candidate = new vo.BatchImportCandidate({
      folder_path: installedGamePath,
      folder_name: gameName,
      executables: executables,
      selected_exe: selectedExe,
      search_name: gameName,
      is_selected: true,
      match_status: "pending",
    });
    
    setBatchImportCandidates([candidate]);
    setBatchImportModalItem(item);
  };
  
  const handleBatchImportComplete = async (importedId?: string) => {
    if (batchImportModalItem && importedId) {
      // 保存导入ID到文件
      try {
        await SaveImportedID(batchImportModalItem.path, importedId);
      } catch (err) {
        console.error("保存导入ID失败:", err);
      }
      
      // 更新单元状态
      setItems(items.map(i => 
        i.id === batchImportModalItem.id 
          ? { ...i, is_imported: true, imported_id: importedId as any } 
          : i
      ));
    }
    setBatchImportModalItem(null);
    setBatchImportCandidates([]);
  };
  
  const handleOpenGame = (item: DownloadedFile) => {
    // 跳转到游戏详情页
    const importedId = (item as any).imported_id;
    if (importedId) {
      window.location.hash = `/game/${importedId}`;
    }
  };
  
  const handleRunGame = async (item: DownloadedFile) => {
    const config = useAppStore.getState().config;
    const installFolder = config?.game_install_folder;
    
    if (!installFolder) {
      setErrorMessage("游戏安装文件夹未配置");
      return;
    }
    
    const gameName = item.game_name || item.name;
    
    // 尝试两种安装路径
    const pathByName = `${installFolder}/${gameName}`;
    const md5Name = await GetGameNameMD5(gameName);
    const pathByMD5 = `${installFolder}/${md5Name}`;
    
    // 扫描查找可执行文件
    let executables: string[] = [];
    let foundPath = "";
    
    try {
      const result = await ScanFolderForExecutables(pathByName) as [string[], string];
      if (result && result[0] && result[0].length > 0) {
        executables = result[0];
        foundPath = pathByName;
      }
    } catch (err) {}
    
    if (executables.length === 0) {
      try {
        const result = await ScanFolderForExecutables(pathByMD5) as [string[], string];
        if (result && result[0] && result[0].length > 0) {
          executables = result[0];
          foundPath = pathByMD5;
        }
      } catch (err) {}
    }
    
    if (executables.length === 0) {
      setErrorMessage("未找到可执行文件");
      return;
    }
    
    // 如果只有一个exe，直接运行
    if (executables.length === 1) {
      try {
        await OpenLocalPath(`${foundPath}/${executables[0]}`);
      } catch (err) {
        console.error("运行游戏失败:", err);
        setErrorMessage("运行游戏失败");
      }
      return;
    }
    
    // 多个exe，弹窗让用户选择
    setRunGameExecutables(executables);
    setRunGameModalItem(item);
  };
  
  const handleSelectExecutable = async (exeName: string) => {
    if (!runGameModalItem) return;
    
    const config = useAppStore.getState().config;
    const installFolder = config?.game_install_folder;
    const gameName = runGameModalItem.game_name || runGameModalItem.name;
    
    const pathByName = `${installFolder}/${gameName}`;
    const md5Name = await GetGameNameMD5(gameName);
    const pathByMD5 = `${installFolder}/${md5Name}`;
    
    // 检查哪个路径存在
    let foundPath = pathByName;
    try {
      await ScanFolderForExecutables(pathByName);
    } catch {
      foundPath = pathByMD5;
    }
    
    try {
      await OpenLocalPath(`${foundPath}/${exeName}`);
    } catch (err) {
      console.error("运行游戏失败:", err);
      setErrorMessage("运行游戏失败");
    }
    
    setRunGameModalItem(null);
    setRunGameExecutables([]);
  };

  const handleOpen = async (item: DownloadedFile) => {
    try {
      await OpenLocalPath(item.path);
    } catch (err) {
      console.error("Open failed:", err);
    }
  };

  const handleOpenInstalled = async (item: DownloadedFile) => {
    try {
      const config = useAppStore.getState().config;
      if (config?.game_install_folder) {
        const baseName = item.name.replace(/\.[^.]+$/, "");
        console.log("handleOpenInstalled baseName:", baseName);
        console.log("handleOpenInstalled itemname:", item.name);
        const extractedGameName = await ExtractGameNameFromDLSite(baseName);
        const installPath = `${config.game_install_folder}/${extractedGameName}`;
        console.log("handleOpenInstalled installPath:", installPath);
        await OpenLocalPath(installPath);
      }
    } catch (err) {
      console.error("Open installed folder failed:", err);
    }
  };


  const handleOpenSearch = (item: DownloadedFile) => {
    setSearchModalItem(item);
  };

  const handleDelete = async (item: DownloadedFile) => {
    if (!confirm(t("downloadedFiles.confirmDelete", { name: item.name }))) {
      return;
    }
    setIsExecuting(true);
    setCurrentExecutingName(item.name);
    try {
      await DeleteItem(item.path);
      await loadItems();
    } catch (err) {
      console.error("Delete failed:", err);
      setErrorMessage(err instanceof Error ? err.message : "删除失败");
    }
    setIsExecuting(false);
  };

  const handleDeleteExtracted = async (item: DownloadedFile) => {
    setIsExecuting(true);
    setCurrentExecutingName(item.name);
    try {
      const result = await DeleteExtractedFolder(item.path, item.name, item.extracted_paths || []);
      if (result && result.has_archive && result.archive_item) {
        // 有同名压缩包，更新当前单元的信息（但保持 id 和 selected 状态）
        const archiveItem = result.archive_item;
        setItems(items.map(i =>
          i.id === item.id
            ? {
                ...i,
                name: archiveItem.name,
                path: archiveItem.path,
                extracted_game_path: "",
                extracted_paths: [],
                is_folder: false,
                is_extracted: false,
                size: archiveItem.size,
                inner_items: archiveItem.inner_items,
                iso_items: archiveItem.iso_items,
                has_numeric_name: archiveItem.has_numeric_name,
              }
            : i
        ));
      } else {
        // 没有同名压缩包，只更新状态
        setItems(items.map(i =>
          i.id === item.id ? { ...i, is_extracted: false, extracted_game_path: "", extracted_paths: [] } : i
        ));
      }
    } catch (err) {
      console.error("Delete extracted folder failed:", err);
      setErrorMessage(err instanceof Error ? err.message : "删除解压文件夹失败");
    }
    setIsExecuting(false);
  };

  const handleDeleteInstalled = async (item: DownloadedFile) => {
    setIsExecuting(true);
    setCurrentExecutingName(item.name);
    try {
      await DeleteInstalledGame(item);
      // 删除成功后更新单元状态
      setItems(items.map(i =>
        i.id === item.id ? { ...i, is_installed: false, installed_path: "" } : i
      ));
    } catch (err) {
      console.error("Delete installed game failed:", err);
      setErrorMessage(err instanceof Error ? err.message : "删除安装失败");
    }
    setIsExecuting(false);
  };


  const formatSize = (bytes: number) => {
    if (bytes < 1024) return bytes + " B";
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB";
    if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + " MB";
    return (bytes / (1024 * 1024 * 1024)).toFixed(1) + " GB";
  };

  const canInstall = (item: DownloadedFile) => {
    return item.is_extracted && item.iso_items.length <= 1;
  };

  const canMount = (item: DownloadedFile) => {
    return item.iso_items.length === 1 && !item.is_installed;
  };

  const isDownloadingItem = (item: DownloadedFile) => {
    return item.is_downloading;
  };

  return (
    <div className="p-6 h-full overflow-auto relative">
      {/* 执行中动画 - 覆盖整个页面内容 */}
      {isExecuting && (
        <div className="absolute inset-0 z-10 flex items-center justify-center bg-black/70">
          <div className="flex flex-col items-center gap-3 px-6 py-4">
            <div className="animate-spin rounded-full h-10 w-10 border-4 border-white/30 border-t-white"></div>
            <span className="text-base text-white/90 font-medium">{t("downloadedFiles.executing")}</span>
            {currentExecutingName && (
              <span className="text-sm text-white/70 max-w-md text-center break-all" title={currentExecutingName}>
                {currentExecutingName}
              </span>
            )}
          </div>
        </div>
      )}

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
                checked={md5AsFolder}
                onChange={(e) => setMd5AsFolder(e.target.checked)}
                className="rounded border-brand-300 text-brand-600 focus:ring-neutral-500"
              />
              {t("downloadedFiles.md5AsFolder")}
            </label>
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
                checked={directIsoInstall}
                onChange={(e) => setDirectIsoInstall(e.target.checked)}
                className="rounded border-brand-300 text-brand-600 focus:ring-neutral-500"
              />
              {t("downloadedFiles.directIsoInstall")}
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

        <div className="flex-1 overflow-auto space-y-2">
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
                      <div className={`text-lg ${item.type == 1 ? "i-mdi-folder" : "i-mdi-archive"} flex-shrink-0`} />
                      <span className="font-medium truncate min-w-0" title={item.name}>{item.name}</span>
                    </div>

                    <div className="flex items-center gap-2 min-w-0">
                      <span className="text-xs truncate min-w-0" title={item.game_name}>游戏名： {item.game_name}</span>
                    </div>
                    {item.extracted_game_path && (
                      <div className="flex items-center gap-2 min-w-0">
                        <span className="text-xs truncate min-w-0" title={item.extracted_game_path}>游戏解压目录： {item.extracted_game_path}</span>
                      </div>
                    )}
                    {item.installed_path && (
                      <div className="flex items-center gap-2 min-w-0">
                        <span className="text-xs truncate min-w-0" title={item.installed_path}>游戏安装目录： {(item as any).installed_path}</span>
                      </div>
                    )}
                    {/* {item.iso_items.length > 0 && (
                      <div className="flex items-center gap-2 min-w-0">
                        <span className="text-xs truncate min-w-0" title={item.iso_items[0]}>Iso： {item.iso_items[0]}</span>
                      </div>
                    )} */}

                    {/* 第二行：类型、大小、下载状态 + 按钮栏 */}
                    <div className="flex items-center justify-between gap-2 mt-1">
                      <div className="flex items-center gap-3 text-xs text-brand-500">
                        <span>{item.type == 1 ? t("downloadedFiles.folder") : item.type ==0 ? t("downloadedFiles.no_need_extract") : t("downloadedFiles.archive")}</span>
                        <span>{formatSize(item.size)}</span>
                        <span>{!item.is_extracted ? "" : "isos:" + item.iso_items.length}</span>
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
                          {item.type != 0 && !item.is_installed && (
                            <button
                              onClick={() => handleDeleteExtracted(item)}
                              disabled={isExecuting}
                              className="px-2 py-1 text-xs bg-red-400 hover:bg-red-500 text-white rounded disabled:opacity-50"
                              title={t("downloadedFiles.deleteExtracted")}
                            >
                              {t("downloadedFiles.deleteExtracted")}
                            </button> )}
                          
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

                      {/* 打开游戏按钮 - 只有导入后显示 */}
                      {item.is_imported && (item as any).imported_id && (
                        <button
                          onClick={() => handleOpenGame(item)}
                          className="px-2 py-1 text-xs bg-blue-500 hover:bg-blue-600 text-white rounded"
                        >
                          {t("downloadedFiles.openGame")}
                        </button>
                      )}

                      {/* 打开安装按钮 */}
                      {item.is_installed && (
                        <button
                          onClick={() => handleOpenInstalled(item)}
                          className="px-2 py-1 text-xs bg-blue-500 hover:bg-blue-600 text-white rounded"
                        >
                          {t("downloadedFiles.openInstall")}
                        </button>
                      )}

                      {/* 运行游戏按钮 */}
                      {item.is_installed && (
                        <button
                          onClick={() => handleRunGame(item)}
                          className="px-2 py-1 text-xs bg-green-500 hover:bg-green-600 text-white rounded"
                        >
                          {t("downloadedFiles.runGame")}
                        </button>
                      )}

                      {/* 打开下载按钮 */}
                      <button
                        onClick={() => handleOpen(item)}
                        className="px-2 py-1 text-xs bg-brand-500 hover:bg-brand-600 text-white rounded"
                      >
                        {t("downloadedFiles.openDownload")}
                      </button>

                      {/* 删除安装按钮 */}
                      {item.is_installed && (
                        <button
                          onClick={() => handleDeleteInstalled(item)}
                          disabled={isExecuting}
                          className="px-2 py-1 text-xs bg-red-400 hover:bg-red-500 text-white rounded disabled:opacity-50"
                          title={t("downloadedFiles.deleteInstalled")}
                        >
                          {t("downloadedFiles.deleteInstalled")}
                        </button>
                      )}

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
                    {(item.inner_items.length > 0 || item.iso_items.length > 0) && (
                      <div className="flex flex-wrap gap-1 text-xs text-brand-400 mt-1">
                        {item.inner_items.map((inner, idx) => (
                          <span key={idx} className="px-1.5 py-0.5 bg-brand-100 dark:bg-brand-700 rounded">
                            {inner}
                          </span>
                        ))}
                        {item.iso_items.map((iso_path, idx) => (
                          <button 
                            key={idx} className="px-1.5 py-0.5 bg-green-100 dark:bg-green-700 rounded"
                            title={`装载${iso_path.split("\\").pop()}`}
                            onClick={() => handleDirectMount(iso_path)}
                            disabled={item.is_installed}
                          >
                            {iso_path.split("\\").pop()}
                          </button>
                        ))}
                        {/* {item.inner_items.length > 5 && (
                          <span className="px-1.5 py-0.5 text-brand-500">+{item.inner_items.length - 5}</span>
                        )} */}
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
                  value="md5"
                  checked={installMethod === "md5"}
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
          itemName={searchModalItem.game_name}
          onClose={() => setSearchModalItem(null)}
        />
      )}

      {/* 批量导入弹窗 */}
      {batchImportModalItem && (
        <BatchImportModal
          isOpen={true}
          onClose={() => {
            setBatchImportModalItem(null);
            setBatchImportCandidates([]);
          }}
          onImportComplete={() => {
            // 导入完成后刷新
            loadItems();
          }}
          onOpenUpdate={(games) => {
            if (games && games.length > 0) {
              const importedId = games[0].id;
              handleBatchImportComplete(importedId);
            }
          }}
          preloadedCandidates={batchImportCandidates}
          preloadedStep="preview"
        />
      )}

      {/* 运行游戏选择弹窗 */}
      {runGameModalItem && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-brand-800 rounded-lg p-6 w-96">
            <h3 className="text-lg font-semibold mb-4">
              {t("downloadedFiles.selectExecutable")}
            </h3>
            <div className="space-y-2 max-h-64 overflow-y-auto">
              {runGameExecutables.map((exe, idx) => (
                <button
                  key={idx}
                  onClick={() => handleSelectExecutable(exe)}
                  className="w-full px-4 py-2 text-left bg-brand-100 dark:bg-brand-700 hover:bg-brand-200 dark:hover:bg-brand-600 rounded"
                >
                  {exe}
                </button>
              ))}
            </div>
            <div className="mt-4 flex justify-end">
              <button
                onClick={() => {
                  setRunGameModalItem(null);
                  setRunGameExecutables([]);
                }}
                className="px-4 py-2 bg-brand-200 dark:bg-brand-700 rounded hover:bg-brand-300 dark:hover:bg-brand-600"
              >
                {t("common.cancel")}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/downloaded_files",
  component: DownloadedFiles,
});

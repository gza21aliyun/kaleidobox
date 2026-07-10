import { useState, useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { createRoute } from "@tanstack/react-router";
import { Route as rootRoute } from "./__root";
import { ListDownloadedFiles, ExtractItem, StartGameTemp, MountISO, InstallGame, DeleteItem, DeleteExtractedFolder, DeleteInstalledGame, RefreshDownloadedFile, ExtractArchivesInFolder, SaveImportedID, ScanFolderForExecutables, UpdateGameName, DownloadSaves, SaveDownloadedFileInfo, OverwriteInstall } from "../../wailsjs/go/service/DownloadedFilesService";
import { LocalSearchModal } from "../components/modal/LocalSearchModal";
import { BatchImportModal } from "../components/modal/BatchImportModal";
import type { service, models } from "../../wailsjs/go/models";
import { OpenLocalPath, DeleteGame, GetGamesByIdsStr } from "../../wailsjs/go/service/GameService";
import { useAppStore } from "../store";
import { vo } from "../../wailsjs/go/models";
import toast from "react-hot-toast";
import { useNavigate } from "@tanstack/react-router";
import { BatchUpdateModal } from "../components/modal/BatchUpdateModal";
import { ConfirmModal } from "../components/modal/ConfirmModal";
import { BetterSelect } from "../components/ui/BetterSelect";
import { parseTime } from "../utils/time";
import { GameInfoModal } from "../components/modal/GameInfoModal";

type DownloadedFile = Omit<service.DownloadedFile, "convertValues"> & {
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
  const [showDownloadSave, setShowDownloadSave] = useState(false);
  const [showOverrideSave, setShowOverrideSave] = useState(false);
  const [md5AsFolder, setMd5AsFolder] = useState(false);
  const [directIsoInstall, setDirectIsoInstall] = useState(false);
  const [showHelp, setShowHelp] = useState(false);
  const [showConfirmModal, setShowConfirmModal] = useState(false);
  const [confirmModalType, setConfirmModalType] = useState<string>("");
  const [confirmModalItem, setConfirmModalItem] = useState<DownloadedFile | null>(null);
  const [installMethod, setInstallMethod] = useState<string>("name");
  const [errorMessage, setErrorMessage] = useState<string>("");
  const [searchModalItem, setSearchModalItem] = useState<DownloadedFile | null>(null);
  const [isExecuting, setIsExecuting] = useState(false);
  const [currentExecutingName, setCurrentExecutingName] = useState("");
  const [currentExecutingTask, setCurrentExecutingTask] = useState("");
  const [scrollPosition, setScrollPosition] = useState(0);
  const listContainerRef = useRef<HTMLDivElement>(null);
  const [batchImportModalItems, setBatchImportModalItems] = useState<DownloadedFile[]>([]);
  const [batchImportCandidates, setBatchImportCandidates] = useState<vo.BatchImportCandidate[]>([]);
  const [runGameModalItem, setRunGameModalItem] = useState<DownloadedFile | null>(null);
  const [runGameExecutables, setRunGameExecutables] = useState<string[]>([]);
  const [isBatchUpdateOpen, setIsBatchUpdateOpen] = useState(false);
  const [overwriteLayers, setOverwriteLayers] = useState(1);
  const [importedGames, setImportedGames] = useState<models.Game[]>([]);
  const [gameEntity, setGameEntity] = useState<models.GameEntity | null>(null);
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);

  const [searchQuery, setSearchQuery] = useState<string>("");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [typeFilter, setTypeFilter] = useState<string>("all");
  const [sortBy, setSortBy] = useState<string>("name");
  const [sortOrder, setSortOrder] = useState<string>("asc");

  const navigate = useNavigate();
  
  const config = useAppStore(state => state.config);
  const fetchConfig = useAppStore(state => state.fetchConfig);
  const fetchGames = useAppStore(state => state.fetchGames);

  useEffect(() => {
    fetchConfig();
  }, [fetchConfig]);

  useEffect(() => {
    if (config && !config.seven_zip_path) {
      setErrorMessage(t("downloadedFiles.noSevenZipPath"));
    }
  }, [config, t]);

  const loadItems = async () => {
    if (listContainerRef.current) {
      setScrollPosition(listContainerRef.current.scrollTop);
    }
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

  useEffect(() => {
    if (!isLoading && listContainerRef.current) {
      listContainerRef.current.scrollTop = scrollPosition;
    }
  }, [items, isLoading, scrollPosition]);

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
      const itemsToImport: DownloadedFile[] = []
      const itemsImported: DownloadedFile[] = []
      for (const itemId of selectedItems) {
        const item = items.find(i => i.id === itemId);
        if (!item) continue;

        // 解压步骤
        if (showExtract && item.status == 1) {
          setCurrentExecutingTask(t("downloadedFiles.taskExtract"));
          setCurrentExecutingName(item.name);
          await handleExtract(item);
          toast.success("解压完成status：" + item.status)
        }
        
        // 安装步骤：需要已解压，且如果镜像文件直接安装开关关闭，则有iso的单元不执行安装
        const hasIso = item.iso_items && item.iso_items.length > 0;
        const shouldInstall = showInstall && item.status == 2 && item.extracted_game_path && (directIsoInstall || !hasIso);
        
        if (shouldInstall) {
          setIsExecuting(true);
          setCurrentExecutingTask(t("downloadedFiles.taskInstall"));
          setCurrentExecutingName(item.name);
          try {
            const installed_path = await InstallGame(item as unknown as service.DownloadedFile, md5AsFolder ? "md5" : "name");
            item.installed_path = installed_path;
            item.status = 3;
          } catch (err) {
            console.error("Install failed:", err);
            setErrorMessage(err instanceof Error ? err.message : "安装失败");
          }
        }
        
        // 导入步骤：需要已安装
        const shouldImport = showImport && item.status == 3;
        if (shouldImport) {
          itemsToImport.push(item);
        }
        if (item.status == 4 && item.imported_id) {
          itemsImported.push(item);
        }
        
        
      }
      if (showImport && itemsToImport.length > 0) {
        setCurrentExecutingTask(t("downloadedFiles.taskImport"));
        await handleImport(itemsToImport);
      }
      if (showDownloadSave && itemsImported.length > 0) {
        const games = await GetGamesByIdsStr(itemsImported.map(item => item.imported_id!).join(","));
        await DownloadSaves(games, showOverrideSave);
      }
      await loadItems();
    } finally {
      setIsExecuting(false);
      setCurrentExecutingName("");
      setCurrentExecutingTask("");
    }
  };

  const handleExtract = async (item: DownloadedFile) => {
    setCurrentExecutingTask(t("downloadedFiles.taskExtract"));
    setCurrentExecutingName(item.name);
    setIsExecuting(true);
    try {
      if (item.type == 1) {
        await ExtractArchivesInFolder(item.path);
        const extractedFolder = await RefreshDownloadedFile(item as unknown as service.DownloadedFile);
        setItems(items.map(i =>
          i.id === item.id ? { ...i, status: 2, extracted_paths: extractedFolder.extracted_paths, 
            iso_items: extractedFolder.iso_items, extracted_game_path: extractedFolder.extracted_game_path, inner_items: extractedFolder.inner_items } : i
        ));
        item.status = 2;
        item.extracted_paths = extractedFolder.extracted_paths;
        item.extracted_game_path = extractedFolder.extracted_game_path;
        item.inner_items = extractedFolder.inner_items;
        item.iso_items = extractedFolder.iso_items;
      } else {
        await ExtractItem(item.path);
        const arc = await RefreshDownloadedFile(item as unknown as service.DownloadedFile);
        console.log("file extracted", arc);
        setItems(items.map(i =>
          i.id === item.id ? { ...i, status: 2, inner_items: arc.inner_items, iso_items: arc.iso_items, path: arc.path,
            extracted_game_path: arc.extracted_game_path, extracted_paths: arc.extracted_paths } : i
        ));
        item.path = arc.path;
        item.status = 2;
        item.inner_items = arc.inner_items;
        item.iso_items = arc.iso_items;
        item.extracted_game_path = arc.extracted_game_path;
        item.extracted_paths = arc.extracted_paths;
        
      }
    } catch (err) {
      console.error("Extract failed:", err);
      setErrorMessage(err instanceof Error ? err.message : "解压失败");
    } finally {
      setIsExecuting(false);
      setCurrentExecutingName("");
      setCurrentExecutingTask("");
    }
  };

  const handleMount = async (item: DownloadedFile) => {
    if (item.iso_items.length < 1) return;
    try {
      await MountISO(item.iso_items[0]);
      toast.success("已装载" + item.iso_items[0]);
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
    setCurrentExecutingTask(t("downloadedFiles.taskInstall"));
    setCurrentExecutingName(confirmModalItem.name);
    // 关闭确认弹窗
    setShowConfirmModal(false);
    setConfirmModalItem(null);

    try {
      const installedPath = await InstallGame(confirmModalItem as unknown as service.DownloadedFile, md5AsFolder ? "md5" : installMethod);
      // 更新单元状态，包含安装路径
      setItems(items.map(i =>
        i.id === confirmModalItem.id 
          ? { ...i, status: 3, installed_path: installedPath } 
          : i
      ));
      confirmModalItem.installed_path = installedPath;
      confirmModalItem.status = 3;
    } catch (err) {
      console.error("Install failed:", err);
      setErrorMessage(err instanceof Error ? err.message : `安装失败${err}`);
    } finally {
      // 隐藏执行中动画
      setIsExecuting(false);
      setCurrentExecutingName("");
      setCurrentExecutingTask("");
    }


    // await loadItems();
  };

  const handleOverwrite = async (item: DownloadedFile) => {
    setConfirmModalType("overwrite");
    setConfirmModalItem(item);
    setShowConfirmModal(true);
  };

  const handleConfirmOverwrite = async () => {
    if (!confirmModalItem) return;

    setIsExecuting(true);
    setCurrentExecutingTask(t("downloadedFiles.taskInstall"));
    setCurrentExecutingName(confirmModalItem.name);
    setShowConfirmModal(false);
    setConfirmModalItem(null);

    try {
      if (!confirmModalItem.imported_id) {
        throw new Error("未找到导入ID");
      }
      const games = await GetGamesByIdsStr(confirmModalItem.imported_id);
      if (games.length === 0) {
        throw new Error("未找到关联的游戏");
      }
      const exePath = games[0].path;
      if (!exePath) {
        throw new Error("游戏执行路径为空");
      }
      await OverwriteInstall(confirmModalItem as unknown as service.DownloadedFile, exePath, overwriteLayers);
      toast.success(t('downloadedFiles.overwriteSuccess') || '覆盖安装成功');
    } catch (err) {
      console.error("Overwrite failed:", err);
      setErrorMessage(err instanceof Error ? err.message : `覆盖安装失败${err}`);
    } finally {
      setIsExecuting(false);
      setCurrentExecutingName("");
      setCurrentExecutingTask("");
    }
  };

  const handleImport = async (items: DownloadedFile[]) => {
    console.log("Import:");
    const candidates : vo.BatchImportCandidate[] = [];
    for (const item of items) {
      // 准备导入
      if (!item.installed_path) continue;
      
      
      // 扫描安装文件夹下的游戏目录查找可执行文件
      let executables: string[] = [];
      let selectedExe = "";
      
      // 先尝试游戏名路径
      try {
        const result = await ScanFolderForExecutables(item.installed_path || "");
        if (result && result.length > 0) {
          executables = result || [];
          selectedExe = result[0] || "";
        }
      } catch (err) {
        console.error("扫描游戏名路径失败:", err);
      }
      
      
      
      // 创建 BatchImportCandidate
      const candidate = new vo.BatchImportCandidate({
        folder_path: item.installed_path,
        folder_name: item.installed_path!.split("/").pop(),
        executables: executables,
        selected_exe: selectedExe,
        search_name: item.game_name,
        is_selected: true,
        match_status: "pending",
      });
      candidates.push(candidate);
    }
    
    
    
    setBatchImportCandidates(candidates);
    setBatchImportModalItems(items);
  };
  
  const handleBatchImportComplete = async (games: models.Game[]) => {
    if (games.length === 0 || batchImportModalItems.length === 0) {
      setBatchImportModalItems([]);
      setBatchImportCandidates([]);
      return;
    }
    for (const game of games) {
      const found = batchImportModalItems.find(i => game.path.includes(i.installed_path!))
      if (found) {
        // toast.success('找到导入');
        found.imported_id = game.id;
        found.status = 4;
        await SaveImportedID(found.path, game.id);
        setItems(items.map(i =>
          i.id === found.id 
            ? { ...i, imported_id: game.id, status: 4 } 
            : i
        ));
        
      } else {
        toast.error('未找到导入' + game.path);
      }
    }
    if (showDownloadSave) {
      await DownloadSaves(games, showOverrideSave);
    }
    await fetchGames();
    if (games.length != 1 || batchImportModalItems.length != 1) {
      await loadItems();
    } else {
      const item = await RefreshDownloadedFile(batchImportModalItems[0] as unknown as service.DownloadedFile)
      setItems(items.map(i =>
        i.id === item.id 
          ? { ...i, imported_id: item.imported_id, status: item.status } 
          : i
      ));
    }
    
    setBatchImportModalItems([]);
    setBatchImportCandidates([]);
  };
  
  const handleOpenGame = (item: DownloadedFile) => {
    // 跳转到游戏详情页
    const importedId = item.imported_id;
    if (importedId) {
      navigate({ to: "/game/$gameId", params: { gameId: importedId } });
    }
  };
  
  const handleRunGame = async (item: DownloadedFile) => {
    if (!item.installed_path) {
      setErrorMessage("游戏未安装");
      return;
    }
    
    
    const gameName = item.game_name || item.name;
    
    // 扫描查找可执行文件
    let executables: string[] = [];
    let foundPath = "";
    
    try {
      const result = await ScanFolderForExecutables(item.installed_path);
      // toast.success(`找到可执行文件：${result.join(", ")}`);
      if (result && result.length > 0) {
        executables = result;
        foundPath = item.installed_path;
      }
    } catch (err) {
      toast.error(`扫描可执行文件失败：${err}`);
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
  
  const handleSelectExecutable = async (exePath: string) => {
    if (!runGameModalItem) return;
    
    
    try {
      await ScanFolderForExecutables(exePath);
    } catch {
    }
    
    try {
      await StartGameTemp(exePath);
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

  const handleGameNameChange = async (item: DownloadedFile, newName: string) => {
    setItems(items.map(i =>
      i.id === item.id ? { ...i, game_name: newName } : i
    ));
    // 保存到 download.klb
    try {
      await UpdateGameName(item.path, newName);
    } catch (err) {
      console.error("保存游戏名失败:", err);
    }
  };

  const handleOpenInstalled = async (item: DownloadedFile) => {
    try {
      if (item.installed_path) {
        console.log("handleOpenInstalled installPath:", item.installed_path);
        await OpenLocalPath(item.installed_path);
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
    setCurrentExecutingTask(t("downloadedFiles.taskDelete"));
    setCurrentExecutingName(item.name);
    try {
      await DeleteItem(item.path);
      await loadItems();
    } catch (err) {
      console.error("Delete failed:", err);
      setErrorMessage(err instanceof Error ? err.message : "删除失败");
    } finally {
      setIsExecuting(false);
      setCurrentExecutingName("");
      setCurrentExecutingTask("");
    }
  };

  const handleDeleteExtracted = async (item: DownloadedFile) => {
    setIsExecuting(true);
    setCurrentExecutingTask(t("downloadedFiles.taskDelete"));
    setCurrentExecutingName(item.name);
    try {
      const result = await DeleteExtractedFolder(item.path, item.base_name || '', item.extracted_paths || []);
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
                status: 1,
                size: archiveItem.size,
                inner_items: archiveItem.inner_items,
                iso_items: archiveItem.iso_items,
                has_numeric_name: archiveItem.has_numeric_name,
                game_name: archiveItem.game_name,
              }
            : i
        ));
      } else {
        // 没有同名压缩包，只更新状态
        setItems(items.map(i =>
          i.id === item.id ? { ...i, status: 1, extracted_game_path: "", extracted_paths: [] } : i
        ));
      }
    } catch (err) {
      console.error("Delete extracted folder failed:", err);
      setErrorMessage(err instanceof Error ? err.message : `删除解压文件夹失败:${err}`);
    } finally {
      setIsExecuting(false);
      setCurrentExecutingName("");
      setCurrentExecutingTask("");
    }
  };

  const handleDeleteInstalled = async (item: DownloadedFile) => {
    setIsExecuting(true);
    setCurrentExecutingTask(t("downloadedFiles.taskDelete"));
    setCurrentExecutingName(item.name);
    try {
      await DeleteInstalledGame(item as unknown as service.DownloadedFile);
      // 删除成功后更新单元状态
      setItems(items.map(i =>
        i.id === item.id ? { ...i, status: 2, installed_path: "" } : i
      ));
    } catch (err) {
      console.error("Delete installed game failed:", err);
      setErrorMessage(err instanceof Error ? err.message : "删除安装失败");
    } finally {
      setIsExecuting(false);
      setCurrentExecutingName("");
      setCurrentExecutingTask("");
    }
  };


  // const confirmDeleteGame = async () => {
  //     if (!confirmModalItem)
  //       return;
  //     try {
  //       await DeleteGame(confirmModalItem.imported_id || '');
  //       toast.success(t('common.deleteSuccess'));
  //       fetchGames();
  //       setConfirmModalItem(null);
  //     }
  //     catch (error) {
  //       console.error("Failed to delete game:", error);
  //       toast.error(t('common.deleteFailed'));
  //     }
  //   };

  const handleDeleteImported = async (item: DownloadedFile) => {
    if (!confirm(`确定要删除“${item.game_name}”的导入吗？`)) {
      return;
    }
    // setConfirmModalItem(item)
    try {
      await DeleteGame(item.imported_id || '');
      // await loadItems();
      const game = await RefreshDownloadedFile(item as unknown as service.DownloadedFile)
      setItems(items.map(i =>
        i.id === item.id ? { ...i, ...game } : i
      ));

    } catch (err) {
      console.error("Delete failed:", err);
    }
  }


  const handleForceCompleteDownload = async (item: DownloadedFile) => {
    const updatedItem = { ...item, status: 1 };
    setItems(prevItems => prevItems.map(i =>
      i.id === item.id ? updatedItem : i
    ));
    SaveDownloadedFileInfo(item.path, updatedItem as unknown as service.DownloadedFile).catch(err => {
      console.error("Failed to save downloaded file info:", err);
    });
    toast.success(t('downloadedFiles.forceComplete') || '已强制完成下载');
  }

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return bytes + " B";
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB";
    if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + " MB";
    return (bytes / (1024 * 1024 * 1024)).toFixed(1) + " GB";
  };

  const canInstall = (item: DownloadedFile) => {
    return item.status == 2 && item.iso_items.length <= 1;
  };

  const canMount = (item: DownloadedFile) => {
    return item.iso_items.length === 1 && item.status == 2;
  };

  const filterItems = useMemo(() => { 
    return items.filter(item => {
                if (searchQuery && !item.name.toLowerCase().includes(searchQuery.toLowerCase())) {
                  return false;
                }
                if (statusFilter === "downloaded") {
                  return item.status === 1;
                }
                if (statusFilter === "extracted") {
                  return item.status === 2;
                }
                if (statusFilter === "installed") {
                  return item.status === 3;
                }
                if (statusFilter === "imported") {
                  return item.status === 4;
                }
                if (typeFilter !== "all" && String(item.type) !== typeFilter) {
                  return false;
                }
                return true;
              })
  }, [items, searchQuery, statusFilter, typeFilter]);


  return (
    <div className="p-6 h-full overflow-auto relative">
      {/* 执行中动画 - 覆盖整个页面内容 */}
      {isExecuting && (
        <div className="absolute inset-0 z-10 flex items-center justify-center bg-black/70">
          <div className="flex flex-col items-center gap-3 px-6 py-4">
            <div className="animate-spin rounded-full h-10 w-10 border-4 border-white/30 border-t-white"></div>
            <span className="text-base text-white/90 font-medium">{t("downloadedFiles.executing")}</span>
            {currentExecutingTask && (
              <span className="text-sm text-white/70">{t("downloadedFiles.executingTask")}: {currentExecutingTask}</span>
            )}
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
        <div className="flex items-center mb-2">
          <h1 className="text-2xl font-bold text-brand-900 dark:text-white mr-2">
            {t("nav.downloadedFiles")}({filterItems.length})
          </h1>
          <div className="flex items-center gap-1">
            <button
              onClick={() => setShowHelp(!showHelp)}
              className="text-brand-600 dark:text-brand-400 hover:text-brand-800 dark:hover:text-brand-300"
              title={showHelp ? "收起说明" : "使用说明"}
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </button>
            <button
              onClick={() => setShowHelp(!showHelp)}
              className="text-brand-600 dark:text-brand-400 hover:text-brand-800 dark:hover:text-brand-300"
              title={showHelp ? "收起说明" : "展开说明"}
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d={showHelp ? "M5 15l7-7 7 7" : "M19 9l-7 7-7-7"} />
              </svg>
            </button>
          </div>
        </div>

        {/* 提示信息 */}
        {showHelp && (
          <p className="text-sm text-brand-500 dark:text-brand-400 mb-4">
            本页面目的是快速批量处理已下载的游戏。<br/>
            根据下载游戏数据摆放方式主要分为文件夹不需解压、文件夹含压缩包、单独压缩包。<br/>
            第四类型为安装文件夹，只在安装文件夹找到，下载文件夹没关联上，通常是直接装载镜像用官方安装程序安装的时候出现，下载文件夹和安装文件夹会作为两条独立记录出现。这时候直接导入安装文件夹后删除下载文件夹即可。<br/>
            下载存档时会下载klb_savedata_xxx的文件到游戏安装目录。删除解压或删除记录时注意要手动把已装载到虚拟光驱的弹出，否则会删除失败。<br/>
            这页面修改的临时信息如游戏名会存在游戏下载目录的download.klb文件中,删除解压时如果类型是单独压缩包会一并清理。<br/>
            装载的时候可能用到win官方或第三方的软件，暂无法完整跟踪全流程，请自己留意盘符变化和处理弹出。
          </p>
        )}

        <div className="flex items-center justify-between mb-4">
          <input
            type="text"
            placeholder={t("downloadedFiles.search")}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="px-3 py-2 text-sm border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-neutral-500 w-64"
          />
          <div className="flex items-center gap-2">
            <span className="text-sm text-brand-500 dark:text-brand-400 whitespace-nowrap">{t("downloadedFiles.statusLabel")}</span>
            <BetterSelect
              value={statusFilter}
              onChange={setStatusFilter}
              options={[
                { value: "all", label: t("downloadedFiles.status.all") },
                { value: "downloaded", label: t("downloadedFiles.status.downloaded") },
                { value: "extracted", label: t("downloadedFiles.status.extracted") },
                { value: "installed", label: t("downloadedFiles.status.installed") },
                { value: "imported", label: t("downloadedFiles.status.imported") },
              ]}
              className="min-w-[120px]"
            />
            <span className="text-sm text-brand-500 dark:text-brand-400 whitespace-nowrap">{t("downloadedFiles.typeLabel")}</span>
            <BetterSelect
              value={typeFilter}
              onChange={setTypeFilter}
              options={[
                { value: "all", label: t("downloadedFiles.type.all") },
                { value: "0", label: t("downloadedFiles.type.folder") },
                { value: "1", label: t("downloadedFiles.type.archive_folder") },
                { value: "2", label: t("downloadedFiles.type.archive") },
                { value: "3", label: t("downloadedFiles.type.installed_folder") },
              ]}
              className="min-w-[150px]"
            />
            <span className="text-sm text-brand-500 dark:text-brand-400 whitespace-nowrap">{t("downloadedFiles.sortLabel")}</span>
            <BetterSelect
              value={sortBy}
              onChange={setSortBy}
              options={[
                { value: "name", label: t("downloadedFiles.sort.name") },
                { value: "game_name", label: t("downloadedFiles.sort.game_name") },
                { value: "time", label: t("downloadedFiles.sort.time") },
              ]}
              className="min-w-[120px]"
            />
            <button
              onClick={() => setSortOrder(sortOrder === "asc" ? "desc" : "asc")}
              className="px-3 py-2 text-sm bg-brand-100 hover:bg-brand-200 dark:bg-brand-700 dark:hover:bg-brand-600 rounded-md"
            >
              {sortOrder === "asc" ? "↑" : "↓"}
            </button>
          </div>
        </div>

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
            
            
          </div>

          <div className="flex items-center gap-2">
            <label className="flex items-center gap-2 text-sm" title="如果已解压下载文件夹里有且只有一个镜像，会直接解压到安装目录。如不打开且文件夹里有镜像，将不执行安装。">
              <input
                type="checkbox"
                checked={directIsoInstall}
                onChange={(e) => setDirectIsoInstall(e.target.checked)}
                className="rounded border-brand-300 text-brand-600 focus:ring-neutral-500"
              />
              安装镜像
            </label>
            <label className="flex items-center gap-2 text-sm" title="以游戏名生成MD5，安装时会以此命名游戏安装文件夹名，用于只允许英数字路径名的游戏，选否时将使用游戏名。">
              <input
                type="checkbox"
                checked={md5AsFolder}
                onChange={(e) => setMd5AsFolder(e.target.checked)}
                className="rounded border-brand-300 text-brand-600 focus:ring-neutral-500"
              />
              MD5文件夹名
            </label>
            <label className="flex items-center gap-2 text-sm" title="解压压缩包到同路径的同名文件夹">
              <input
                type="checkbox"
                checked={showExtract}
                onChange={(e) => setShowExtract(e.target.checked)}
                className="rounded border-brand-300 text-brand-600 focus:ring-neutral-500"
              />
              {t("downloadedFiles.extract")}
            </label>
            <label className="flex items-center gap-2 text-sm" title="将游戏解压目录复制到设置好的游戏安装文件夹下">
              <input
                type="checkbox"
                checked={showInstall}
                onChange={(e) => setShowInstall(e.target.checked)}
                className="rounded border-brand-300 text-brand-600 focus:ring-neutral-500"
              />
              {t("downloadedFiles.install")}
            </label>
            <label className="flex items-center gap-2 text-sm"
              title="批量导入游戏选中并执行后，将在完成前一步后为选中的已安装游戏准备好数据打开批量导入弹窗直接跳到选择导入阶段"
            >
              <input
                type="checkbox"
                checked={showImport}
                onChange={(e) => setShowImport(e.target.checked)}
                className="rounded border-brand-300 text-brand-600 focus:ring-neutral-500"
              />
              {t("downloadedFiles.import")}
            </label>

            <label className="flex items-center gap-2 text-sm"
              title="开启后将在导入后自动下载游戏存档到游戏执行文件的同一目录下"
            >
              <input
                type="checkbox"
                checked={showDownloadSave}
                onChange={(e) => setShowDownloadSave(e.target.checked)}
                className="rounded border-brand-300 text-brand-600 focus:ring-neutral-500"
              />
              下载存档
            </label>

            <label className="flex items-center gap-2 text-sm"
              title="在下载存档后会搜索游戏存放存档位置解压存档覆盖到该位置，但很多游戏在运行游戏前并没生成存档文件夹"
            >
              <input
                type="checkbox"
                checked={showOverrideSave}
                onChange={(e) => setShowOverrideSave(e.target.checked)}
                className="rounded border-brand-300 text-brand-600 focus:ring-neutral-500"
              />
              覆盖存档
            </label>

            
            
            
            <button
              onClick={handleExecute}
              disabled={selectedItems.length === 0 || isExecuting}
              className="px-3 py-1.5 text-sm bg-brand-600 hover:bg-brand-700 text-white rounded-md disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {t("downloadedFiles.execute")}
            </button>

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
              onClick={loadItems}
              disabled={isLoading}
              className="p-2 text-sm bg-brand-100 hover:bg-brand-200 dark:bg-brand-700 dark:hover:bg-brand-600 rounded-md disabled:opacity-50"
              title={t("downloadedFiles.refresh")}
            >
              <div className="i-mdi-refresh text-lg" />
            </button>
          </div>
        </div>

        <div className="flex-1 overflow-auto space-y-2" ref={listContainerRef}>
          {isLoading ? (
            <div className="flex items-center justify-center h-32">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-brand-600"></div>
            </div>
          ) : items.length === 0 ? (
            <div className="text-center py-12 text-brand-500">
              {t("downloadedFiles.empty")}
            </div>
          ) : (
            filterItems
              .sort((a, b) => {
                let comparison = 0;
                if (sortBy === "name") {
                  comparison = a.name.localeCompare(b.name);
                } else if (sortBy === "game_name") {
                  comparison = (a.game_name || "").localeCompare(b.game_name || "");
                } else if (sortBy === "time") {
                  comparison = parseTime(a.time).getTime() - parseTime(b.time).getTime();
                }
                return sortOrder === "asc" ? comparison : -comparison;
              })
              .map(item => (
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
                      <span className="text-xs">游戏名：</span>
                      <input
                        type="text"
                        disabled={item.type == 2 && item.status < 2 || item.status < 1}
                        value={item.game_name || ""}
                        onChange={(e) => handleGameNameChange(item, e.target.value)}
                        className="flex-1 text-xs px-1 py-0.5 border border-brand-300 dark:border-brand-600 rounded bg-transparent dark:bg-brand-700 min-w-0"
                      />
                    </div>
                    {item.extracted_game_path && (
                      <div className="flex items-center gap-2 min-w-0">
                        <span className="text-xs truncate min-w-0" title={item.extracted_game_path}>游戏解压目录： {item.extracted_game_path}</span>
                      </div>
                    )}
                    {item.installed_path && (
                      <button
                        onClick={() => handleOpenInstalled(item)}
                        className="flex items-center gap-2 min-w-0 hover:text-brand-500 text-left"
                      >
                        <span className="text-xs truncate min-w-0" title={item.installed_path}>游戏安装目录： {item.installed_path}</span>
                      </button>
                    )}
                    {/* {item.iso_items.length > 0 && (
                      <div className="flex items-center gap-2 min-w-0">
                        <span className="text-xs truncate min-w-0" title={item.iso_items[0]}>Iso： {item.iso_items[0]}</span>
                      </div>
                    )} */}

                    {/* 第二行：类型、大小、下载状态 + 按钮栏 */}
                    <div className="flex items-center justify-between gap-2 mt-1">
                      <div className="flex items-center gap-3 text-xs text-brand-500">
                        <span>类型：{item.type == 1 ? '文件夹含压缩包' : item.type ==0 ? '文件夹无需解压' : item.type == 3 ? '安装文件夹' : '单独压缩包'}</span>
                        <span>{item.size == 0 ? '' : formatSize(item.size)}</span>
                        {/* <span>{!item.is_extracted ? "" : "镜像数目:" + item.iso_items.length}</span> */}
                        <span>{parseTime(item.time).toLocaleDateString()}</span>
                        {item.status == 0 && (
                          <span className="text-orange-500 flex items-center gap-1">
                            <div className="i-mdi-download animate-pulse" />
                            {t("downloadedFiles.downloading")}
                          </span>
                        )}
                      </div>

                      {/* 按钮栏 */}
                      <div className="flex gap-1 flex-wrap">
                      {/* 解压按钮 */}
                      {item.status == 0 ? (
                        <>
                          <button
                            disabled
                            className="px-2 py-1 text-xs bg-brand-100 dark:bg-brand-700 text-brand-400 rounded cursor-not-allowed"
                          >
                            {t("downloadedFiles.extract")}
                          </button>
                          {item.type != 2 && (
                            <button
                              onClick={() => handleForceCompleteDownload(item)}
                              className="px-2 py-1 text-xs bg-yellow-500 hover:bg-yellow-600 text-white rounded"
                            >
                              {t("downloadedFiles.forceComplete")}
                            </button>
                          )}
                        </>
                      ) : item.status == 2 ? (
                        <>
                          <button
                            disabled
                            className="px-2 py-1 text-xs bg-brand-100 dark:bg-brand-700 text-brand-400 rounded cursor-not-allowed"
                          >
                            {t("downloadedFiles.extracted")}
                          </button>
                          {item.type != 0 && (
                            <button
                              onClick={() => handleDeleteExtracted(item)}
                              disabled={isExecuting}
                              className="px-2 py-1 text-xs bg-red-400 hover:bg-red-500 text-white rounded disabled:opacity-50"
                              title={t("downloadedFiles.deleteExtracted")}
                            >
                              {t("downloadedFiles.deleteExtracted")}
                            </button> )}
                          
                        </>
                      ) : item.status == 1 ? (
                        <button
                          onClick={() => handleExtract(item)}
                          disabled={isExecuting}
                          className="px-2 py-1 text-xs bg-blue-500 hover:bg-blue-600 text-white rounded disabled:opacity-50"
                        >
                          {t("downloadedFiles.extract")}
                        </button>
                      ) : (<></>)}

                      {/* 安装按钮 */}
                      {item.status == 0 ? (
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
                      ) : item.status == 3 ? (
                        <button
                          disabled
                          className="px-2 py-1 text-xs bg-brand-100 dark:bg-brand-700 text-brand-400 rounded cursor-not-allowed"
                        >
                          {t("downloadedFiles.installed")}
                        </button>
                      ) : item.status == 2 ? (
                        <button
                          onClick={() => handleInstall(item)}
                          className="px-2 py-1 text-xs bg-green-500 hover:bg-green-600 text-white rounded"
                        >
                          {t("downloadedFiles.install")}
                        </button>
                      ) : (<></>)}

                      {/* 导入按钮 */}
                      {item.status == 0 ? (
                        <button
                          disabled
                          className="px-2 py-1 text-xs bg-brand-100 dark:bg-brand-700 text-brand-400 rounded cursor-not-allowed"
                        >
                          {t("downloadedFiles.import")}
                        </button>
                      ) : item.status == 4 ? (
                        <button
                          disabled
                          className="px-2 py-1 text-xs bg-brand-100 dark:bg-brand-700 text-brand-400 rounded cursor-not-allowed"
                        >
                          {t("downloadedFiles.imported")}
                        </button>
                      ) : item.status < 3 ? (
                        <button
                          disabled
                          className="px-2 py-1 text-xs bg-brand-100 dark:bg-brand-700 text-brand-400 rounded cursor-not-allowed"
                        >
                          {t("downloadedFiles.import")}
                        </button>
                      ) : (
                        <button
                          onClick={() => handleImport([item])}
                          className="px-2 py-1 text-xs bg-purple-500 hover:bg-purple-600 text-white rounded"
                        >
                          {t("downloadedFiles.import")}
                        </button>
                      )}

                      {/* 打开游戏按钮 - 只有导入后显示 */}
                      {item.status == 4 && item.imported_id && (
                        <button
                          onClick={() => handleOpenGame(item)}
                          className="px-2 py-1 text-xs bg-blue-500 hover:bg-blue-600 text-white rounded"
                        >
                          {t("downloadedFiles.openGame")}
                        </button>
                      )}

                      {/* 运行游戏按钮 */}
                      {item.status >= 3 && (
                        <button
                          onClick={() => handleRunGame(item)}
                          className="px-2 py-1 text-xs bg-green-500 hover:bg-green-600 text-white rounded"
                        >
                          {t("downloadedFiles.runGame")}
                        </button>
                      )}

                      {/* 打开下载按钮 */}
                      {item.type != 3 && (
                        <button
                          onClick={() => handleOpen(item)}
                          className="px-2 py-1 text-xs bg-brand-500 hover:bg-brand-600 text-white rounded"
                        >
                          {t("downloadedFiles.openDownload")}
                        </button>
                      )}
                      

                      {/* 删除安装按钮 */}
                      {item.status == 3 && item.type != 3 && (
                        <button
                          onClick={() => handleDeleteInstalled(item)}
                          disabled={isExecuting}
                          className="px-2 py-1 text-xs bg-red-400 hover:bg-red-500 text-white rounded disabled:opacity-50"
                          title={t("downloadedFiles.deleteInstalled")}
                        >
                          {t("downloadedFiles.deleteInstalled")}
                        </button>
                      )}

                      {item.status == 4 && item.imported_id && (
                        <>
                          <button
                            onClick={() => handleDeleteImported(item)}
                            disabled={isExecuting}
                            className="px-2 py-1 text-xs bg-red-400 hover:bg-red-500 text-white rounded disabled:opacity-50"
                            title="删除导入"
                          >
                            删除导入
                          </button>
                          <button
                            onClick={() => handleOverwrite(item)}
                            disabled={isExecuting}
                            className="px-2 py-1 text-xs bg-orange-500 hover:bg-orange-600 text-white rounded disabled:opacity-50"
                            title={t("downloadedFiles.overwrite")}
                          >
                            {t("downloadedFiles.overwrite")}
                          </button>
                        </>
                      )}

                      {/* 搜索按钮 - 已导入的单元隐藏 */}
                      {item.status < 4 && (
                        <button
                          onClick={() => handleOpenSearch(item)}
                          className="px-2 py-1 text-xs bg-brand-100 hover:bg-brand-200 dark:bg-brand-700 dark:hover:bg-brand-600 text-brand-700 dark:text-brand-300 rounded"
                        >
                          {t("downloadedFiles.search")}
                        </button>
                      )}

                      {/* 装载按钮 */}
                      {canMount(item) && (
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
                            title={`装载${iso_path}`}
                            onClick={() => handleDirectMount(iso_path)}
                            disabled={item.status >= 3}
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
                : confirmModalType === "overwrite"
                ? t("downloadedFiles.overwriteTitle")
                : t("downloadedFiles.installTitle")}
            </h3>
            {confirmModalType === "install_iso" && (
              <p className="mb-4 text-brand-600">
                {t("downloadedFiles.installIsoMessage")}
              </p>
            )}
            {confirmModalType === "overwrite" && (
              <>
                {confirmModalItem.iso_items && confirmModalItem.iso_items.length > 0 && (
                  <p className="mb-4 text-orange-500">
                    {t("downloadedFiles.overwriteIsoMessage")}
                  </p>
                )}
                <div className="mb-4">
                  <label className="block mb-2">
                    {t("downloadedFiles.overwriteLayers")}
                  </label>
                  <input
                    type="number"
                    min="0"
                    max="10"
                    value={overwriteLayers}
                    onChange={(e) => setOverwriteLayers(Math.max(0, parseInt(e.target.value) || 0))}
                    className="w-full px-3 py-2 border rounded"
                  />
                </div>
              </>
            )}
            {confirmModalType !== "overwrite" && (
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
            )}
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
                onClick={confirmModalType === "overwrite" ? handleConfirmOverwrite : handleConfirmInstall}
                className="px-4 py-2 bg-brand-600 text-white rounded hover:bg-brand-700"
              >
                {confirmModalType === "overwrite"
                  ? t("downloadedFiles.confirmOverwrite")
                  : t("downloadedFiles.confirmInstall")}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 游戏搜索弹窗 */}
      {searchModalItem && (
        <LocalSearchModal
          itemName={searchModalItem.game_name}
          onOpenInfo={(ge)=> {setGameEntity(ge)}}
          type={searchModalItem.type}
          status={searchModalItem.status}
          onChoose={(game) => {
            const installedPath = game.path.replace(/[/\\][^/\\]+$/, '');
            const updatedItem = { ...searchModalItem, status: 4, imported_id: game.id, installed_path: installedPath };
            setItems(prevItems => prevItems.map(i =>
              i.id === searchModalItem.id ? updatedItem : i
            ));
            SaveDownloadedFileInfo(searchModalItem.path, updatedItem as unknown as service.DownloadedFile).catch(err => {
              console.error("Failed to save downloaded file info:", err);
            });
            setSearchModalItem(null);
            toast.success(t('common.associateSuccess') || '关联成功');
          }}
          onClose={() => setSearchModalItem(null)}
        />
      )}

      {gameEntity && (
        <GameInfoModal
          gameEntity={gameEntity}
          onClose={() => {setGameEntity(null)}}
        />
      )}

      {/* 批量导入弹窗 */}
      {batchImportModalItems.length > 0 && (
        <BatchImportModal
          isOpen={true}
          onClose={() => {
            setBatchImportModalItems([]);
            setBatchImportCandidates([]);
          }}
          onImportComplete={(games, isOpenUpdate) => {
            // 导入完成后刷新
            // loadItems();
            if (games && games.length > 0) {
              handleBatchImportComplete(games);
              if (isOpenUpdate) {
                setImportedGames(importedGames)
                setIsBatchUpdateOpen(true);
              }
              
            }


          }}
          preloadedCandidates={batchImportCandidates}
          preloadedStep="preview"
        />
      )}



      {/* <ConfirmModal
              isOpen={isDeleteModalOpen}
              title={t('common.deleteGame')}
              message={t('game.modals.deleteMessage', { name: confirmModalItem?.game_name })}
              confirmText={t('common.confirmDelete')}
              type="danger"
              onClose={() => setIsDeleteModalOpen(false)}
              onConfirm={confirmDeleteGame}
            /> */}

      <BatchUpdateModal
              isOpen={isBatchUpdateOpen}
              onClose={() => setIsBatchUpdateOpen(false)}
              onUpdateComplete={() => {
                setImportedGames([])
              }}
              games={importedGames}
            />

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
                  {exe.split("\\").pop()}
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

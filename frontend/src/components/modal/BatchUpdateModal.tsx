import { models, service } from "../../../wailsjs/go/models";
import { useRef, useState, useEffect } from "react"; // 添加useEffect
import { createPortal } from "react-dom";
import toast from "react-hot-toast";
import { enums, vo } from "../../../wailsjs/go/models";
import { EventsOff, EventsOn, EventsOnce, EventsOffAll, EventsOnMultiple } from "../../../wailsjs/runtime";

import { FetchMetadata, FetchMetadataByName, UpdateGamesBackground, FillGame } from "../../../wailsjs/go/service/GameService";
import {
  CancelTask
} from "../../../wailsjs/go/service/TaskService";
import { BetterSelect } from "../ui/BetterSelect";
import { BetterSwitch } from "../ui/BetterSwitch";

interface BatchUpdateModalProps {
  isOpen: boolean;
  onClose: () => void;
  onUpdateComplete: () => void;
  games: models.Game[];
}


export function BatchUpdateModal({ isOpen, onClose, onUpdateComplete, games }: BatchUpdateModalProps) {
//   const [step, setStep] = useState<Step>("select");
//   const [libraryPath, setLibraryPath] = useState("");
  const [candidates, setCandidates] = useState<models.Game[]>(games);
//   const [importResult, setImportResult] = useState<service.ImportResult | null>(null);
//   const [isLoading, setIsLoading] = useState(false);
//   const [matchProgress, setMatchProgress] = useState({ current: 0, total: 0, gameName: "" });
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [updatedIds, setUpdatedIds] = useState<string[]>([]);
  const [matchedIds, setMatchedIds] = useState<string[]>([]);
  const [failedIds, setFailedIds] = useState<string[]>([]);
  const [source, setSource] = useState<enums.SourceType>(enums.SourceType.BANGUMI);
//   const [isDropdownOpen, setIsDropdownOpen] = useState(false);
//   const [manualData, setManualData] = useState<GameFetchedData[]>([]);
  const [manualData, setManualData] = useState<vo.GameMetadataFromWebVO[]>([]);

  // 用于中断匹配过程的标志
  const abortMatchRef = useRef(false);

  // 手动选择弹窗状态
  const [showManualSelect, setShowManualSelect] = useState(false);
  const [manualSelectIndex, setManualSelectIndex] = useState<number | null>(null);
  const [manualMatches, setManualMatches] = useState<vo.GameMetadataFromWebVO[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [manualId, setManualId] = useState("");
  const [manualSource, setManualSource] = useState<enums.SourceType>(enums.SourceType.BANGUMI);
  const taskId = useRef("");
  const [isOverwrite, setIsOverWrite]  = useState(false);
  const [shouldLoadStaffs, setShouldLoadStaffs] = useState(false);
  const [shouldLoadCharacters, setShouldLoadCharacters] = useState(false);
  const [shouldLoadImages, setShouldLoadImages] = useState(false);
  const [itemIdMatching, setItemIdMatching] = useState("")

  // Move this useEffect to the top level, right after all useState declarations
    useEffect(() => {
    const unlistenTaskUpdate = EventsOn("game_updates", (data: any) => {
        console.log("Received task update:", data);
        // setCurrentTask(task);

        // 将接收到的数据转换为Task对象
        // 注意：使用正确的语法从data对象获取值
        const task : models.TaskNotice = new models.TaskNotice(data);
        
        // console.log("task:", realTask);
        if (task.status === enums.TaskStatus.STARTED && task.item_id === "") {
            taskId.current = task.id;
            // console.log("set taskId:", task.id);
        }
        // console.log("received taskid:" + task.id + " current taskid:" + taskId + ", item_id:" + task.item_id +  " item_status:" + task.item_status)
        if (task.id === taskId.current) {
            if (task.item_status === enums.TaskStatus.INITIAL && task.item_id !== "") {
                setItemIdMatching(task.item_id)
            }
            if (task.item_status === enums.TaskStatus.COMPLETED && task.item_id !== "") {
                const newGame : models.Game = task.item_data as models.Game;
                console.log("newGame:", newGame)
                const games = [...candidates]
                const oldGame = games.find(game => game.id === task.item_id)
                if (oldGame) {
                    const index = games.indexOf(oldGame)
                    games[index] = newGame
                }
                setCandidates(games)
                setUpdatedIds([...updatedIds, task.item_id])

            }
            if (task.item_status === enums.TaskStatus.ERROR && task.item_id !== "") {
                setFailedIds([...failedIds, task.item_id])
            }
            if (task.item_id == "") {
                setItemIdMatching("")
            }
        }
        
        
        
    });

    return () => {
        if (unlistenTaskUpdate) {
        unlistenTaskUpdate(); // 取消事件监听
        }
    };
    }, [updatedIds, failedIds]);

  // 当isOpen变为true时，重置候选游戏列表
  useEffect(() => {
    if (isOpen) {
      setCandidates(games);
      // 重置相关状态
      setSelectedIds(games.map(game => game.id)); // 默认选中所有游戏
      setUpdatedIds([]);
      setMatchedIds([]);
      setFailedIds([]);
    }
  }, [isOpen, games]); // 当isOpen或games变化时执行

  if (!isOpen)
    return null;

    


  const isMatched = (c: models.Game, source: enums.SourceType) => {
        // if (updatedIds.includes(c.id)) return true;
        if (c.source_type === source) {
            return true;
        }
        if (source === enums.SourceType.BANGUMI && c.bangumi_id && c.bangumi_id.length > 0) {
            return true;
        }
        if (source === enums.SourceType.DMM && c.dmm_id && c.dmm_id.length > 0) {
            return true;
        }
        if (source === enums.SourceType.EROSCAPE && c.eroscape_id && c.eroscape_id.length > 0) {
            return true;
        }
        if (source === enums.SourceType.YMGAL && c.ymgal_id && c.ymgal_id.length > 0) {
            return true;
        }
        return false;
    }

  const cancelUpdate = () => { 
    if (taskId.current !== "") {
        CancelTask(taskId.current)
    }
    
  };
  const handleUpdate = () => {
    const uuid = crypto.randomUUID();
    // setTaskId(uuid);
    // console.log("create uuid  :" + uuid + " current taskId:" + taskId)
    const req = new vo.MetadataRequest({
      id: uuid,
      source: source,
      isOverwrite: isOverwrite,
      should_fetch_staffs: shouldLoadStaffs,
      should_fetch_charactors: shouldLoadCharacters,
      should_fetch_images: shouldLoadImages,
    });
    UpdateGamesBackground(candidates.filter(c => selectedIds.includes(c.id)), req, uuid)
    // console.log("create uuid :" + uuid)
    
    
  };

  const handleImport = async () => {
    // setStep("importing");
    // setIsLoading(true);

    // try {
    //   // 转换为后端需要的格式
    //   const importCandidates: vo.BatchImportCandidate[] = candidates
    //     .filter(c => c.isSelected)
    //     .map((c) => {
    //       const candidate = new vo.BatchImportCandidate({
    //         folder_path: c.folderPath,
    //         folder_name: c.folderName,
    //         executables: c.executables,
    //         selected_exe: c.selectedExe,
    //         search_name: c.searchName,
    //         is_selected: c.isSelected,
    //         match_status: c.matchStatus,
    //       });
    //       if (c.matchedGame) {
    //         candidate.matched_game = c.matchedGame;
    //       }
    //       if (c.matchSource) {
    //         candidate.match_source = c.matchSource;
    //       }
    //       return candidate;
    //     });

    //   const result = await BatchImportGames(importCandidates);
    //   setImportResult(result);
    //   setStep("result");

    //   if (result.success > 0) {
    //     toast.success(`成功导入 ${result.success} 个游戏`);
    //     onUpdateComplete();
    //   }
    // }
    // catch (error) {
    //   console.error("Failed to import:", error);
    //   toast.error("导入失败");
    //   setStep("preview");
    // }
    // finally {
    //   setIsLoading(false);
    // }
  };

  const toggleCandidate = (id: string) => {
    selectedIds.includes(id)
      ? setSelectedIds(selectedIds.filter((id) => id !== id))
      : setSelectedIds([...selectedIds, id]);
  };

  const updateSearchName = (index: number, name: string) => {
    const updated = [...candidates];
    updated[index].name = name;
    setCandidates(updated);
  };

  const openManualSelect = async (index: number, id: string) => {
    setManualSelectIndex(index);
    const found = manualData.filter((m) => m.Game.id === id)
    setManualMatches(found || []);
    setShowManualSelect(true);
    setManualId("");

    // 如果没有缓存的匹配结果，重新搜索
    if (found.length === 0) {
      setIsSearching(true);
      try {
        const results = await FetchMetadataByName(candidates[index].name);
        setManualMatches(results || []);
      }
      catch (error) {
        console.error("Failed to search:", error);
      }
      finally {
        setIsSearching(false);
      }
    }
  };

  const selectManualMatch = (game: models.Game, source: enums.SourceType) => {
    if (manualSelectIndex !== null) {
      const updated = [...candidates];
      const oldGame = updated[manualSelectIndex];
      if (source === enums.SourceType.DMM) {
        oldGame.dmm_id = game.dmm_id
      }
      if (source === enums.SourceType.EROSCAPE) {
        oldGame.eroscape_id = game.eroscape_id
      }
      if (source === enums.SourceType.BANGUMI) {
        oldGame.bangumi_id = game.bangumi_id
      }
      if (source === enums.SourceType.YMGAL) {
        oldGame.ymgal_id = game.ymgal_id
      }
      oldGame.images = game.images
      oldGame.tags = game.tags
      oldGame.release_at = game.release_at
      oldGame.staffs = game.staffs
      oldGame.summary = game.summary
      oldGame.name = game.name
      oldGame.charactors = game.charactors
      setCandidates(updated);
    }
    setShowManualSelect(false);
    setManualSelectIndex(null);
  };


  const handleSearchById = async () => {
    if (!manualId || manualSelectIndex === null)
      return;
    setIsSearching(true);
    
    
    try {
      const request = new vo.MetadataRequest({
        source: manualSource,
        id: manualId,
      });
      const game = await FetchMetadata(request);
      if (game && game.name) {
        selectManualMatch(game, manualSource);
      }
      else {
        toast.error("未找到游戏");
      }
    }
    catch (error) {
      console.error("Failed to fetch by ID:", error);
      toast.error("获取失败");
    }
    finally {
      setIsSearching(false);
    }
  };

  const resetAndClose = () => {
    // 中断正在进行的匹配
    abortMatchRef.current = true;

    // setStep("select");
    // setLibraryPath("");
    setCandidates([]);
    // setImportResult(null);
    // setMatchProgress({ current: 0, total: 0, gameName: "" });
    setShowManualSelect(false);
    setManualSelectIndex(null);
    onClose();
  };

  const selectedCount = selectedIds.length;
  // 已匹配包括自动匹配和手动匹配
  const matchedCount = matchedIds.length;
  const updatedCount = updatedIds.length;
  const notFoundCount = candidates.filter(c => !isMatched(c, source)).length;
  const pendingCount = candidates.filter(c => selectedIds.includes(c.id) && !updatedIds.includes(c.id)).length;

  return createPortal(
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <div className="w-full max-w-4xl max-h-[90vh] rounded-xl bg-white shadow-2xl dark:bg-brand-800 flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-brand-200 dark:border-brand-700">
            <div className="flex items-center gap-3">
                <div className="flex items-center gap-3">
                            <div className="i-mdi-folder-multiple text-3xl text-blue-500" />
                            <h2 className="text-2xl font-bold text-brand-900 dark:text-white">
                            批量更新游戏库
                            </h2>
                        </div>
                        <div>
            </div>
          
            <div className="min-w-[150px] rounded-lg bg-blue-50 dark:bg-blue-900/20 p-3"> {/* 调整内边距 */}
              <div className="flex items-center gap-2"> {/* 水平布局放置标签和选择器 */}
                <span className="text-sm text-blue-700 dark:text-blue-300 whitespace-nowrap">数据源:</span> {/* 标签移到左侧 */}
                <BetterSelect
                  value={source}
                  onChange={(value) => {
                    if (source !== value as enums.SourceType) {
                        setSource(value as enums.SourceType)
                        setUpdatedIds([])
                        setItemIdMatching("")
                        setFailedIds([])
                        taskId.current = ""
                    }
                    
                  }}
                  options={[
                    { value: enums.SourceType.BANGUMI, label: "Bangumi" },
                    { value: enums.SourceType.VNDB, label: "VNDB" },
                    { value: enums.SourceType.YMGAL, label: "月幕Gal" },
                    { value: enums.SourceType.DMM, label: "DMM" },
                    { value: enums.SourceType.EROSCAPE, label: "EroScape" },
                  ]}
                  className="min-w-[200px] flex-1" // 设置最小宽度并允许伸缩
                />
                {/* 更新选项开关 */}
                <div className="glass-card bg-brand-50 dark:bg-brand-800/30 rounded-lg p-4">
                  <div className="grid grid-cols-2 gap-3">
                    <div className="flex items-center justify-between p-2 bg-white dark:bg-brand-700/50 rounded-lg">
                      <label className="text-sm font-medium text-brand-700 dark:text-brand-300 truncate">
                        覆盖数据
                      </label>
                      <BetterSwitch
                        id="overwrite_switch"
                        checked={isOverwrite}
                        onCheckedChange={(checked) => {
                          setIsOverWrite(checked);
                        }}
                      />
                    </div>

                    <div className="flex items-center justify-between p-2 bg-white dark:bg-brand-700/50 rounded-lg">
                      <label className="text-sm font-medium text-brand-700 dark:text-brand-300 truncate">
                        制作人员
                      </label>
                      <BetterSwitch
                        id="load_staffs_switch"
                        checked={shouldLoadStaffs}
                        onCheckedChange={(checked) => {
                          setShouldLoadStaffs(checked);
                        }}
                      />
                    </div>

                    <div className="flex items-center justify-between p-2 bg-white dark:bg-brand-700/50 rounded-lg">
                      <label className="text-sm font-medium text-brand-700 dark:text-brand-300 truncate">
                        角色信息
                      </label>
                      <BetterSwitch
                        id="load_characters_switch"
                        checked={shouldLoadCharacters}
                        onCheckedChange={(checked) => {
                          setShouldLoadCharacters(checked);
                        }}
                      />
                    </div>

                    <div className="flex items-center justify-between p-2 bg-white dark:bg-brand-700/50 rounded-lg">
                      <label className="text-sm font-medium text-brand-700 dark:text-brand-300 truncate">
                        图片资源
                      </label>
                      <BetterSwitch
                        id="load_images_switch"
                        checked={shouldLoadImages}
                        onCheckedChange={(checked) => {
                          setShouldLoadImages(checked);
                        }}
                      />
                    </div>
                  </div>
                </div>
              </div>
            </div>
                
          </div>
          <button
            onClick={resetAndClose}
            className="i-mdi-close text-2xl text-brand-500 p-1 rounded-lg
                            hover:bg-brand-100 hover:text-brand-700 focus:outline-none
                            dark:text-brand-400 dark:hover:bg-brand-700 dark:hover:text-brand-200"
          />
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">


          {/* Step: Preview & Match */}
          {true && (
            <div className="space-y-4">
              {/* Summary */}
              <div className="flex gap-4">
                <div className="flex-1 rounded-lg bg-neutral-50 dark:bg-neutral-900/20 p-4 text-center">
                  <div className="text-3xl font-bold text-neutral-600 dark:text-neutral-400">
                    {candidates.length}
                  </div>
                  <div className="text-sm text-neutral-700 dark:text-neutral-300">
                    游戏数
                  </div>
                </div>
                <div className="flex-1 rounded-lg bg-success-50 dark:bg-success-900/20 p-4 text-center">
                  <div className="text-3xl font-bold text-success-600 dark:text-success-400">
                    {updatedCount}
                  </div>
                  <div className="text-sm text-success-700 dark:text-success-300">
                    已更新
                  </div>
                </div>
                {notFoundCount > 0 && (
                  <div className="flex-1 rounded-lg bg-orange-50 dark:bg-orange-900/20 p-4 text-center">
                    <div className="text-3xl font-bold text-orange-600 dark:text-orange-400">
                      {notFoundCount}
                    </div>
                    <div className="text-sm text-orange-700 dark:text-orange-300">
                      未匹配
                    </div>
                  </div>
                )}
                {pendingCount > 0 && (
                  <div className="flex-1 rounded-lg bg-gray-50 dark:bg-gray-900/20 p-4 text-center">
                    <div className="text-3xl font-bold text-gray-600 dark:text-gray-400">
                      {pendingCount}
                    </div>
                    <div className="text-sm text-gray-700 dark:text-gray-300">
                      待更新
                    </div>
                  </div>
                )}
              </div>

              {/* Candidate List */}
              <div className="max-h-[400px] overflow-y-auto rounded-lg border border-brand-200 dark:border-brand-700">
                {candidates.length === 0
                  ? (
                      <div className="p-8 text-center text-brand-400">
                        未导入游戏
                      </div>
                    )
                  : (
                      <table className="w-full">
                        <thead className="top-0 bg-brand-50 dark:bg-brand-700">
                          <tr>
                            <th className="px-3 py-2 text-left text-sm font-medium text-brand-600 dark:text-brand-300 w-10">
                              <input
                                type="checkbox"
                                checked={candidates.every(c => selectedIds.includes(c.id))}
                                onChange={(e) => {
                                  const isAllSelected = candidates.length == selectedIds.length;
                                  const ids = isAllSelected ? [] : candidates.map(c => c.id);
                                  setSelectedIds(ids);
                                }}
                              />
                            </th>
                            <th className="px-3 py-2 text-left text-sm font-medium text-brand-600 dark:text-brand-300">
                              名称
                            </th>
                            <th className="px-3 py-2 text-left text-sm font-medium text-brand-600 dark:text-brand-300">
                              匹配情况
                            </th>
                            <th className="px-3 py-2 text-center text-sm font-medium text-brand-600 dark:text-brand-300 w-32">
                              更新状态
                            </th>
                            <th className="px-3 py-2 text-center text-sm font-medium text-brand-600 dark:text-brand-300 w-20">
                              操作
                            </th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-brand-100 dark:divide-brand-700">
                          {candidates.map((candidate, index) => (
                            <tr
                              key={index}
                              className={`${!selectedIds.includes(candidate.id) ? "opacity-50" : "hover:bg-brand-50 dark:hover:bg-brand-750"}`}
                            >
                              <td className="px-3 py-2">
                                <input
                                  type="checkbox"
                                  checked={selectedIds.includes(candidate.id)}
                                  onChange={() => toggleCandidate(candidate.id)}
                                />
                              </td>
                              <td className="px-3 py-2">
                                <input
                                  type="text"
                                  value={candidate.name}
                                  onChange={e => updateSearchName(index, e.target.value)}
                                  className="w-full bg-transparent border-b border-transparent hover:border-brand-300 focus:border-neutral-500 focus:outline-none text-sm text-brand-900 dark:text-white"
                                />
                              </td>
                              <td className="px-3 py-2">
                                {(() => {
                                    var text: string = "";
                                    text += candidate.source_type + ":" + candidate.source_id;
                                    if (candidate.source_type !== enums.SourceType.DMM && candidate.dmm_id && candidate.dmm_id !== "") {
                                        text += ",DMM:" + candidate.dmm_id;
                                    }
                                    if (candidate.source_type !== enums.SourceType.BANGUMI && candidate.bangumi_id && candidate.bangumi_id !== "") {
                                        text += ",Bangumi:" + candidate.bangumi_id;
                                    }
                                    if (candidate.source_type !== enums.SourceType.EROSCAPE && candidate.eroscape_id && candidate.eroscape_id !== "") {
                                        text += ",批评空间:" + candidate.eroscape_id;
                                    }
                                    if (candidate.source_type !== enums.SourceType.YMGAL && candidate.ymgal_id && candidate.ymgal_id !== "") {
                                        text += ",YMGal:" + candidate.ymgal_id;
                                    }
                                    return text;
                                })()}
                              </td>
                              <td className="px-3 py-2 text-center">
                                {
                                    itemIdMatching == candidate.id ? (
                                        <span className="inline-flex items-center rounded-full bg-blue-100 px-2 py-1 text-xs text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">
                                            <div className="i-mdi-sync mr-1 animate-spin" />
                                            更新中
                                        </span>
                                    ) : failedIds.includes(candidate.id) ? (
                                        <span className="inline-flex items-center rounded-full bg-red-100 px-2 py-1 text-xs text-red-700 dark:bg-red-900/30 dark:text-red-400">
                                            <div className="i-mdi-alert-circle mr-1" />
                                            已出错
                                        </span>
                                    ) : updatedIds.includes(candidate.id) ? (
                                        <span className="inline-flex items-center rounded-full bg-success-100 px-2 py-1 text-xs text-success-700 dark:bg-success-900/30 dark:text-success-400">
                                            <div className="i-mdi-check-circle mr-1" />
                                            已更新
                                        </span>
                                    ) : !isMatched(candidate, source) ? ( 
                                        <span className="inline-flex items-center rounded-full bg-orange-100 px-2 py-1 text-xs text-orange-700 dark:bg-orange-900/30 dark:text-orange-400">
                                            <div className="i-mdi-alert-circle mr-1" />
                                            未找到
                                        </span>
                                    ) : (
                                        <span className="inline-flex items-center rounded-full bg-gray-100 px-2 py-1 text-xs text-gray-700 dark:bg-gray-900/30 dark:text-gray-400">
                                            <div className="i-mdi-clock-outline mr-1" />
                                            已匹配
                                        </span>
                                    )
                                }

                              </td>
                              <td className="px-3 py-2 text-center">
                                <button
                                  onClick={() => openManualSelect(index, candidate.id)}
                                  className="text-neutral-500 hover:text-neutral-700 text-sm"
                                  title="手动选择"
                                >
                                  <div className="i-mdi-pencil text-lg" />
                                </button>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    )}
              </div>

              {/* Actions */}
              <div className="flex justify-between">
                <div className="flex gap-3">
                  {pendingCount > 0 && (
                    <button
                      onClick={handleUpdate}
                      className="rounded-lg px-5 py-2.5 text-sm font-medium text-white bg-neutral-600 hover:bg-neutral-700"
                    >
                      开始更新元数据
                    </button>
                  )}
                  {/* <button
                    onClick={handleImport}
                    disabled={selectedCount === 0}
                    className="rounded-lg px-5 py-2.5 text-sm font-medium text-white disabled:opacity-50 bg-success-600 hover:bg-success-700"
                  >
                    导入
                    {" "}
                    {selectedCount}
                    {" "}
                    个游戏
                  </button> */}
                  <button
                    onClick={cancelUpdate}
                    disabled={selectedCount === 0 && taskId.current !== ""}
                    className="rounded-lg px-5 py-2.5 text-sm font-medium text-white disabled:opacity-50 bg-success-600 hover:bg-success-700"
                  >
                    取消更新
                  </button>
                </div>
              </div>
            </div>
          )}



        </div>
      </div>

      {/* Manual Select Modal */}
      {showManualSelect && manualSelectIndex !== null && (
        <div className="fixed inset-0 z-60 flex items-center justify-center bg-black/50">
          <div className="w-full max-w-2xl max-h-[80vh] rounded-xl bg-white shadow-2xl dark:bg-brand-800 flex flex-col">
            <div className="flex items-center justify-between p-4 border-b border-brand-200 dark:border-brand-700">
              <h3 className="text-lg font-bold text-brand-900 dark:text-white">
                手动选择:
                {" "}
                {candidates[manualSelectIndex].name}
              </h3>
              <button
                onClick={() => setShowManualSelect(false)}
                className="i-mdi-close text-xl text-brand-500 hover:text-brand-700"
              />
            </div>

            <div className="flex-1 overflow-y-auto p-4 space-y-4">
              {isSearching ? (
                <div className="py-8 text-center">
                  <div className="i-mdi-loading animate-spin text-3xl mx-auto mb-2 text-neutral-500" />
                  <p className="text-brand-400">搜索中...</p>
                </div>
              ) : (
                <>
                  {/* 匹配结果 */}
                  <div className="flex flex-wrap gap-3">
                    {manualMatches.filter(m => m.Game).map((match, idx) => (
                      <div
                        key={idx}
                        onClick={() => selectManualMatch(match.Game!, match.Source)}
                        className="w-36 cursor-pointer rounded-lg border border-brand-200 p-2 transition hover:border-neutral-500 hover:shadow-md dark:border-brand-700"
                      >
                        <div className="aspect-[3/4] w-full overflow-hidden rounded-md bg-brand-200 dark:bg-brand-700">
                          {match.Game!.cover_url
                            ? (
                                <img src={match.Game!.cover_url} alt={match.Game!.name} className="h-full w-full object-cover" referrerPolicy="no-referrer" />
                              )
                            : (
                                <div className="flex h-full items-center justify-center text-brand-400">
                                  <div className="i-mdi-image-off text-3xl" />
                                </div>
                              )}
                        </div>
                        <h4 className="mt-1 truncate text-xs font-bold text-brand-900 dark:text-white" title={match.Game!.name}>
                          {match.Game!.name}
                        </h4>
                        <p className="text-xs text-brand-400">{match.Source}</p>
                      </div>
                    ))}
                  </div>

                  {manualMatches.length === 0 && (
                    <p className="text-center text-brand-400 py-4">未找到匹配结果</p>
                  )}

                  {/* 手动输入ID */}
                  <div className="border-t border-brand-200 dark:border-brand-700 pt-4 mt-4">
                    <p className="text-sm text-brand-500 mb-3">通过 ID 查找:</p>
                    <div className="flex gap-2">
                      <BetterSelect
                        value={manualSource}
                        onChange={value => setManualSource(value as enums.SourceType)}
                        options={[
                          { value: enums.SourceType.BANGUMI, label: "Bangumi" },
                          { value: enums.SourceType.VNDB, label: "VNDB" },
                          { value: enums.SourceType.YMGAL, label: "月幕gal" },
                        ]}
                        className="w-32"
                      />
                      <input
                        type="text"
                        value={manualId}
                        onChange={e => setManualId(e.target.value)}
                        placeholder="输入 ID"
                        className="flex-1 rounded border border-brand-300 bg-brand-50 px-3 py-1.5 text-sm dark:border-brand-600 dark:bg-brand-700"
                      />
                      <button
                        onClick={handleSearchById}
                        disabled={!manualId || isSearching}
                        className="rounded bg-neutral-500 px-4 py-1.5 text-sm text-white hover:bg-neutral-600 disabled:opacity-50"
                      >
                        查找
                      </button>
                    </div>
                  </div>

                  
                </>
              )}
            </div>
          </div>
        </div>
      )}
    </div>,
    document.body,
  );
}

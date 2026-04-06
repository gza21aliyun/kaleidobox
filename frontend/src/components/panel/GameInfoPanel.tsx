import { appconf, enums, models } from "../../../wailsjs/go/models";
import { useNavigate } from "@tanstack/react-router";
import { GetWorksMapByGameId, CountWorks, GetWorksByGameId } from "../../../wailsjs/go/service/WorkService";
import { tagMapForEach, workMapForEach, charactorsForEach } from "../utils/Utility";
import { GetTagListByString } from "../../../wailsjs/go/service/TagService";
import { DeleteTagForGame, AddTagsForGames } from "../../../wailsjs/go/service/GameService";
import { useEffect, useState } from "react";
import { useAppStore } from "../../store";
import { useTranslation } from "react-i18next";
import { AddTagModal } from "../modal/AddTagModal";
import toast from "react-hot-toast";

interface GameEditFormProps {
  game: models.Game;
  config?: appconf.AppConfig;
  onTagTaps: (tag: string) => void;
  updateGame: (g: models.Game) => void;
}

export function GameInfoPanel({ 
    game, config, onTagTaps, updateGame }: GameEditFormProps) { 
        const navigate = useNavigate();
        const { t } = useTranslation();
        const { updateGameInGames } = useAppStore();
        const [worksMap, setWorksMap] = useState<Map<enums.StaffRole, models.Work[]>>(new Map())
        const [tagsMap, setTagsMap] = useState<Map<string, models.Tag[]>>(new Map())
        const [showAddTagModal, setShowAddTagModal] = useState(false);
        const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
        const [tagToDelete, setTagToDelete] = useState<models.Tag | null>(null);


        useEffect(() => { 
            console.log("01 GetWorksMapByGameId", game.id)
            GetWorksByGameId(game.id).then((res) => { 
                console.log("01 GetWorksByGameId", game.id)
                console.log("work res:", res)
                var array: models.Work[] = res || [];
                var m = new Map<enums.StaffRole, models.Work[]>();
                for (let i = 0; i < array.length; i++) {
                    const existingItems = m.get(array[i].role) || [];
                    m.set(array[i].role, [...existingItems, array[i]]);
                }
                setWorksMap(m)
            
            })
            console.log("gameTag", game.tags)
            GetTagListByString(game.tags).then((res) => {
                console.log("tag res:", res)
                var array: models.Tag[] = res || []
                var m = new Map<string, models.Tag[]>();
                for (let i = 0; i < array.length; i++) {
                    const existingItems = m.get(array[i].category) || [];
                    m.set(array[i].category, [...existingItems, array[i]]);
                }
                console.log("tag map:", m)
                setTagsMap(m)
            })
            
            return () => { 
                setWorksMap(new Map())
                setTagsMap(new Map())
            
            }
        }, [game])
        const handleTagClick = (tag: models.Tag) => {
                // 直接执行搜索功能
                navigate({ to: '/library', search: 
                    { tags: tag.name } 
                });
            };

        const handleDeleteTag = (tag: models.Tag) => {
                setTagToDelete(tag);
                setShowDeleteConfirm(true);
            };

        const confirmDeleteTag = async () => {
                if (tagToDelete) {
                    try {
                        const newGame = await DeleteTagForGame(game, tagToDelete.name);
                        updateGame(newGame);
                        updateGameInGames(newGame);
                        // 刷新标签列表
                        GetTagListByString(newGame.tags).then((res) => {
                            console.log("tag res:", res)
                            var array: models.Tag[] = res || []
                            var m = new Map<string, models.Tag[]>();
                            for (let i = 0; i < array.length; i++) {
                                const existingItems = m.get(array[i].category) || [];
                                m.set(array[i].category, [...existingItems, array[i]]);
                            }
                            setTagsMap(m)
                        });
                        setShowDeleteConfirm(false);
                        setTagToDelete(null);
                    } catch (error) {
                        console.error("Error deleting tag:", error);
                    }
                }
            };

        const cancelDeleteTag = () => {
                setShowDeleteConfirm(false);
                setTagToDelete(null);
            };

        const handleStaffClick = (work: models.Work) => {
                console.log(`跳转到员工: `, work);
                if (work.staff_id != "") {
                    navigate({ to: `/staff/${work.staff_id}` });
                }
            };

        const handleOpenAddTagModal = () => {
                setShowAddTagModal(true);
            };

        const handleAddTag = (gs: models.Game[]) => {
                if (true) {
                    try {
                        updateGame(gs[0]);
                        toast.success(t('game.toasts.tagAddSuccess'));
                        // 刷新标签列表
                        GetTagListByString(gs[0].tags).then((res) => {
                            console.log("tag res:", res)
                            var array: models.Tag[] = res || []
                            var m = new Map<string, models.Tag[]>();
                            for (let i = 0; i < array.length; i++) {
                                const existingItems = m.get(array[i].category) || [];
                                m.set(array[i].category, [...existingItems, array[i]]);
                            }
                            setTagsMap(m)
                        });
                        setShowAddTagModal(false);
                    } catch (error) {
                        console.error("Error adding tag:", error);
                    }
                }
            };
        
        return ( 
            <div> 
                {/* 在这里插入worksMap展示内容 */}
                <div className="mt-6 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
                    <h3 className="text-lg font-semibold mb-3 text-brand-900 dark:text-white">{t('gameInfo.staffInfo')}</h3>
                    {worksMap && worksMap.size > 0 ? (
                        <div className="space-y-4">
                            {workMapForEach(worksMap,t,(role, works) => (
                                <div key={role} className="border-l-4 border-brand-500 pl-4">
                                    <h4 className="font-medium text-brand-800 dark:text-brand-200 capitalize">
                                        {role.toString() || role.replace(/([A-Z])/g, ' $1').trim()}
                                    </h4>
                                    {works && works.length > 0 ? (
                                        <ul className="mt-2 flex flex-wrap gap-2">
                                            {works.map((work: models.Work, index: number) => (
                                                <li key={index} 
                                                onClick={() => handleStaffClick(work)}
                                                className="text-sm text-brand-600 dark:text-brand-400 bg-brand-50 dark:bg-brand-900/50 px-2 py-1 rounded">
                                                    {work.role === enums.StaffRole.CV 
                                                        ? `${work.staff_name || ''} (${work.charactor_name || t('gameInfo.unknownCharacter')})`
                                                        : work.staff_name || t('gameInfo.staff', { index: index + 1 })
                                                    }
                                                </li>
                                            ))}
                                        </ul>
                                    ) : (
                                        <p className="text-sm text-brand-500 dark:text-brand-400 italic mt-1">{t('gameInfo.noData')}</p>
                                    )}
                                </div>
                            ))}
                        </div>
                    ) : (
                        <p className="text-brand-600 dark:text-brand-400 text-sm">{t('gameInfo.noStaffInfo')}</p>
                    )}
                </div>




                <div className="mt-4">
                    <div className="flex items-center justify-between">
                        
                        <div className="font-semibold text-brand-900 dark:text-white">{t('gameInfo.categoryTags')}</div>
                        <button
                            onClick={handleOpenAddTagModal}
                            className="flex items-center px-2 py-1 bg-brand-500 text-white rounded-lg hover:bg-brand-600 transition-colors text-sm"
                            >
                            <div className="i-mdi-plus text-sm"></div>
                        </button>
                    </div>
                    <div className="space-y-4">
                        {tagsMap && tagsMap.size > 0 ? (
                            
                            tagMapForEach(tagsMap, (category, tags) => (
                                <div key={category} className="border-l-4 border-brand-500 pl-4">
                                    <h4 className="font-medium text-brand-800 dark:text-brand-200 capitalize">
                                        {category}
                                    </h4>
                                    <ul className="mt-2 flex flex-wrap gap-2">
                                        {tags.map((tag: models.Tag, index: number) => (
                                            <li
                                                key={index}
                                                className="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium bg-[#e0e000] text-brand-800 dark:bg-[#e0e000] dark:text-brand-200 hover:bg-[#d0d000] dark:hover:bg-[#d0d000] transition-colors cursor-pointer"
                                                onClick={() => {
                                                    console.log(`Clicked tag: ${tag.name}`);
                                                    handleTagClick(tag);
                                                }}
                                            >
                                                {tag.name}
                                                {!tag.block_modify && (
                                                    <button
                                                        className="ml-2 text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
                                                        onClick={(e) => {
                                                            e.stopPropagation(); // 阻止事件冒泡
                                                            handleDeleteTag(tag);
                                                        }}
                                                    >
                                                        <div className="i-mdi-close text-xs"></div>
                                                    </button>
                                                )}
                                            </li>
                                        ))}
                                    </ul>
                                </div>
                            ), t)
                        ) : (
                            <p className="text-brand-600 dark:text-brand-400 text-sm">{t('gameInfo.noCategoryTags')}</p>
                        )}
                    </div>
                </div>



                {/* 删除标签确认弹窗 */}
                {showDeleteConfirm && tagToDelete && (
                    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
                        <div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full">
                            <div className="flex justify-between items-center mb-4">
                                <h3 className="text-xl font-semibold text-brand-900 dark:text-white">{t('gameInfo.confirmDelete')}</h3>
                                <button 
                                    onClick={cancelDeleteTag}
                                    className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
                                >
                                    ×
                                </button>
                            </div>
                            <div className="space-y-4">
                                <p className="text-brand-900 dark:text-white">{t('gameInfo.confirmDeleteTag', { tag: tagToDelete.name })}</p>
                                <div className="flex gap-4 justify-end mt-6">
                                    <button
                                        onClick={cancelDeleteTag}
                                        className="px-4 py-2 bg-gray-500 text-white rounded hover:bg-gray-600 transition-colors"
                                    >
                                        {t('gameInfo.cancel')}
                                    </button>
                                    <button
                                        onClick={confirmDeleteTag}
                                        className="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600 transition-colors"
                                    >
                                        {t('gameInfo.delete')}
                                    </button>
                                </div>
                            </div>
                        </div>
                    </div>
                )}

                {/* 添加标签弹窗 */}
                <AddTagModal
                    isOpen={showAddTagModal}
                    onClose={() => setShowAddTagModal(false)}
                    onConfirm={handleAddTag}
                    games={[game]}
                />

            </div>
        );
}
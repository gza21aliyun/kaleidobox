
import { appconf, enums, models } from "../../../wailsjs/go/models";
import { useNavigate } from "@tanstack/react-router";
import { GetWorksMapByGameId, CountWorks, GetWorksByGameId } from "../../../wailsjs/go/service/WorkService";
import { charactorsForEach, getCharactorIds } from "../utils/Utility";
import { GetGamesByRelatedGames, GetGamesByBrand, AddRelatedGames, DeleteRelatedGame } from "../../../wailsjs/go/service/GameService";
import { FetchImages } from "../../../wailsjs/go/service/ImageService";
import { useEffect, useState, useRef } from "react";
import { useTranslation } from "react-i18next";
import { GameCard } from "../card/GameCard";
import { ImageBackupCard, ImageCard } from "../card/ImageCard";
import { arrayFind, arrayMapString, joinString } from "../utils/Utility";
import { useAppStore } from "../../store";

interface GameEditFormProps {
  game: models.Game;
  config?: appconf.AppConfig;
  onTagTaps: (tag: string) => void;
  updateGame: (g: models.Game) => void;
}

const CharactorImages = ({ workId }: { workId: string }) => {
  const { t } = useTranslation();
  const [images, setImages] = useState<models.ImageBackup[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (workId) {
      FetchImages(workId, 3, 1, true).then((res) => {
        setImages(res ?? []);
        setLoading(false);
      }).catch((err) => {
        console.error("Failed to fetch works images:", err);
        setLoading(false);
      });
    }
  }, [workId]);

  if (loading) {
    return null;
  }

  if (images.length === 0) {
    return null;
  }

  return (
    <div className="mt-3">
      <div className="flex flex-wrap gap-3">
        {images.map((imageBackup, idx) => (
          <div key={idx} className="w-54 h-40 flex-shrink-0">
            <ImageBackupCard imageBackup={imageBackup} />
          </div>
        ))}
      </div>
    </div>
  );
};

export function GameIntroPanel({ 
    game, config, onTagTaps, updateGame }: GameEditFormProps) { 
        const navigate = useNavigate();
        const { t } = useTranslation();
        const [worksMap, setWorksMap] = useState<Map<enums.StaffRole, models.Work[]>>(new Map())
        const textareaRef = useRef<HTMLTextAreaElement>(null);
        const [relatedGames, setRelatedGames] = useState<models.Game[]>([])
        const [brandGames, setBrandGames] = useState<models.Game[]>([])        
        const { updateGameInGames } = useAppStore()

        // 检查URL参数中是否有选中的游戏
        useEffect(() => {
            console.log("URL search:", window.location.search);
            const urlParams = new URLSearchParams(window.location.search);
            const selectedGameIds = urlParams.get('selectedGameIds');
            console.log("selectedGameIds = ", selectedGameIds);
            
            // 尝试直接从 URL 字符串中提取 selectedGameIds
            if (!selectedGameIds) {
                const searchStr = window.location.search;
                const match = searchStr.match(/selectedGameIds=([^&]+)/);
                if (match) {
                    const extractedIds = decodeURIComponent(match[1]);
                    console.log("Extracted selectedGameIds:", extractedIds);
                    processSelectedGames(extractedIds);
                }
            } else {
                processSelectedGames(selectedGameIds);
            }
        }, [game, window.location.search]);
        
        // 处理选中的游戏
        const processSelectedGames = (selectedGameIds: string) => {
            console.log("Processing selectedGameIds:", selectedGameIds);
            // 清除URL参数
            const newUrl = new URL(window.location.href);
            newUrl.searchParams.delete('selectedGameIds');
            window.history.replaceState({}, '', newUrl.toString());
            
            // 调用AddRelatedGames添加关联游戏
            const gameIds = selectedGameIds.split(',');
            console.log("gameIds = ", selectedGameIds);
            if (gameIds.length > 0) {
                // 构建游戏对象数组
                console.log("selectedGames = ", selectedGameIds);
                console.log("game = ", game);
                AddRelatedGames(gameIds, game).then((res) => {
                    console.log("AddRelatedGames response = ", res);
                    // 更新关联游戏列表
                    if (res) {
                        updateGame(res);
                        updateGameInGames(res);

                    }
                }).catch((err) => {
                    console.error("Failed to add related games:", err);
                });
            }
        };

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
            GetGamesByBrand(game.company).then((res) => { 
                setBrandGames(res || [])
            })
            console.log("game.related_games = ", game.related_games)
            GetGamesByRelatedGames(game.related_games).then((res) => { 
                console.log("GetGamesByRelatedGames:", res)
                setRelatedGames(res || [])
            })
            
            return () => { 
                setWorksMap(new Map())
            
            }
        }, [game])

        useEffect(() => {
            if (textareaRef.current) {
            // 重置高度以获取准确的scrollHeight
            textareaRef.current.style.height = 'auto';
            // 设置为内容的实际高度
            textareaRef.current.style.height = textareaRef.current.scrollHeight + 'px';
            }
            
        }, [game?.summary || ""]); // 当summary变化时重新计算高度


        
        return ( 
            <div> 

                <div className="mt-4">
                    <div className="font-semibold mb-2 text-brand-900 dark:text-white">{t('gameIntro.summary')}</div>
                    <div className="bg-gradient-to-br from-brand-50 to-brand-100 dark:from-brand-900/30 dark:to-brand-800/20 rounded-xl p-5 border border-brand-200 dark:border-brand-700 shadow-sm">
                        <textarea
                            ref={textareaRef}
                            value={game.summary || t('gameIntro.noSummary')}
                            disabled={true}
                            className="w-full bg-transparent border-0 outline-none text-brand-600 dark:text-brand-400 text-sm leading-relaxed resize-none"
                            style={{
                                height: 'auto',
                                overflow: 'hidden',
                                resize: 'none',
                                padding: 0,
                                margin: 0
                            }}
                            />
                    </div>
                </div>

                {/* 角色 */}
                <div className="mt-4">
                    <div className="font-semibold mb-2 text-brand-900 dark:text-white">{t('gameIntro.characters')}</div>
                    <div className="space-y-4">
                        {true ? (
                            
                            // ... existing code ...
                            charactorsForEach(worksMap, (charactor) => (
                                <div key={`${charactor.charactor_name}-${charactor.staff_name}`} 
                                    className="bg-white dark:bg-brand-800/30 rounded-lg p-4 border border-brand-200 dark:border-brand-700 hover:shadow-md transition-shadow">
                                    
                                    <div className="flex flex-col xl:flex-row gap-6 items-start">
                                        {/* 角色图片 */}
                                        {charactor.charactor_image && (
                                            <div className="flex-shrink-0" style={{ minWidth: '200px' }}>
                                                <ImageCard 
                                                    url={charactor.charactor_image} 
                                                    alt={charactor.charactor_name}
                                                    className="w-[200px] h-[300px] object-contain rounded-lg shadow-md"
                                                    style={{ objectFit: 'cover', objectPosition: 'center top' }}
                                                />
                                            </div>
                                        )}
                                        
                                        {/* 角色信息 */}
                                        <div className="flex-1 min-w-[400px]">
                                            <div className="flex flex-wrap items-center gap-4 mb-4">
                                                <h3 className="font-bold text-brand-900 dark:text-white text-xl">
                                                    <button
                                                        onClick={() => {
                                                            var characterIds = getCharactorIds(worksMap)
                                                            if (charactor.charactor_id) {
                                                                navigate({ 
                                                                    to: `/charactor/${charactor.charactor_id}`, 
                                                                    search: { characterIds },
                                                                });
                                                            }
                                                        }}
                                                        className="text-brand-700 dark:text-brand-300 hover:text-brand-900 dark:hover:text-white font-medium underline-offset-2 hover:underline transition-colors"
                                                    >
                                                        {charactor.charactor_name}
                                                    </button>
                                                </h3>
                                                
                                                {charactor.staff_name && (
                                                    <div>
                                                        <span className="text-sm text-brand-600 dark:text-brand-400">{t('gameIntro.cv')}：</span>
                                                        <button
                                                            onClick={() => {
                                                                if (charactor.staff_id) {
                                                                    navigate({ to: `/staff/${charactor.staff_id}` });
                                                                }
                                                            }}
                                                            className="text-sm text-brand-700 dark:text-brand-300 hover:text-brand-900 dark:hover:text-white font-medium underline-offset-2 hover:underline transition-colors"
                                                        >
                                                            {charactor.staff_name}
                                                        </button>
                                                    </div>
                                                )}
                                            </div>
                                            
                                            {/* 角色简介 */}
                                            {charactor.work_summary && (
                                                <div className="bg-brand-50 dark:bg-brand-900/20 rounded-lg p-4">
                                                    <p className="text-sm text-brand-700 dark:text-brand-300 whitespace-pre-wrap leading-relaxed">
                                                        {charactor.work_summary.replace('\n\n', '\n')}
                                                    </p>
                                                </div>
                                            )}
                                        </div>

                                        {/* 角色截图 */}
                                        {charactor.id && (
                                            <div className="w-full xl:w-auto xl:flex-shrink-0 xl:max-w-[700px]" style={{ minWidth: '100px' }}> 
                                                <CharactorImages workId={charactor.id} />
                                            </div>
                                        )}
                                    </div>
                                </div>
                                
                            ))
// ... existing code ...
                        ) : (
                            <p className="text-brand-600 dark:text-brand-400 text-sm">{t('gameIntro.noCategories')}</p>
                        )}
                    </div>
                </div>

                {/* 关联游戏 */}
                <div className="mt-4">
                    <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">{t('gameIntro.relatedGames')}</h3>
                    <div className="grid grid-cols-[repeat(auto-fill,minmax(8.75rem,1fr))] gap-3">
                        {relatedGames.filter((g) => g.id != game.id).map((g) => (
                            <GameCard key={g.id} 
                                game={g}
                                filteredGameIdsStr={arrayMapString(relatedGames, (g) => g.id)}
                                onDelete={(g) => {
                                    DeleteRelatedGame(g, game).then((res) => {
                                        // 删除成功，刷新页面
                                        updateGame(res)
                                        updateGameInGames(res)
                                    });

                                }}
                                />
                        ))}
                        <div 
                            className="glass-card relative flex w-full flex-col overflow-hidden rounded-xl border border-brand-100 bg-white shadow-sm transition-all duration-300 hover:shadow-xl dark:border-brand-700 dark:bg-brand-800 flex items-center justify-center cursor-pointer"
                            onClick={() => {
                                // 导航到游戏库页面，开启选择模式
                                const currentPath = window.location.href;
                                console.log("Navigate to library with returnPath:", currentPath);
                                // 对 returnPath 进行 URL 编码，确保特殊字符被正确处理
                                const encodedReturnPath = encodeURIComponent(currentPath);
                                console.log("Encoded returnPath:", encodedReturnPath);
                                navigate({ 
                                    to: `/library`, 
                                    search: { 
                                        selectMode: true, 
                                        returnPath: encodedReturnPath 
                                    }
                                });
                            }}
                        >
                            <div className="flex flex-col items-center justify-center gap-2 p-4">
                                <div className="i-mdi-plus-circle text-4xl text-brand-500 dark:text-brand-400" />
                                <span className="text-sm font-medium text-brand-900 dark:text-white">{t('gameIntro.addRelatedGame')}</span>
                            </div>
                        </div>
                    </div>
                </div>

                {/* 捅品牌游戏 */}
                { brandGames.length > 0 && (
                    <div className="mt-4">
                        <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">同品牌游戏</h3>
                        <div className="grid grid-cols-[repeat(auto-fill,minmax(8.75rem,1fr))] gap-3">
                            {brandGames.filter((g) => g.id != game.id).map((g) => (
                                <GameCard key={g.id} 
                                    game={g}
                                    filteredGameIdsStr={arrayMapString(brandGames, (g) => g.id)}
                                    />
                            ))}
                        </div>
                    </div>
                )}



                



                
            </div>
        );
}
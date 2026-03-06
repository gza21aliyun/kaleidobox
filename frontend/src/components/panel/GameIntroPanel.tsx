
import { appconf, enums, models } from "../../../wailsjs/go/models";
import { useNavigate } from "@tanstack/react-router";
import { GetWorksMapByGameId, CountWorks, GetWorksByGameId } from "../../../wailsjs/go/service/WorkService";
import { tagMapForEach, workMapForEach, charactorsForEach, getCharactors } from "../utils/Utility";
import { GetGamesByRelatedGames } from "../../../wailsjs/go/service/GameService";
import { useEffect, useState, useRef } from "react";
import { useTranslation } from "react-i18next";
import { GameCard } from "../card/GameCard";

interface GameEditFormProps {
  game: models.Game;
  config?: appconf.AppConfig;
  onTagTaps: (tag: string) => void;
}

export function GameIntroPanel({ 
    game, config, onTagTaps }: GameEditFormProps) { 
        const navigate = useNavigate();
        const { t } = useTranslation();
        const [worksMap, setWorksMap] = useState<Map<enums.StaffRole, models.Work[]>>(new Map())
        const textareaRef = useRef<HTMLTextAreaElement>(null);
        const [relatedGames, setRelatedGames] = useState<models.Game[]>([])



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

                {/* 角色 */}
                <div className="mt-4">
                    <div className="font-semibold mb-2 text-brand-900 dark:text-white">{t('gameIntro.characters')}</div>
                    <div className="flex flex-wrap gap-3">
                        {true ? (
                            
                            charactorsForEach(worksMap, (charactor) => (
                                <div key={`${charactor.charactor_name}-${charactor.staff_name}`} 
                                    className="flex items-center gap-3 bg-white dark:bg-brand-800/30 rounded-lg p-3 border border-brand-200 dark:border-brand-700 min-w-[280px] hover:shadow-md transition-shadow">
                                    {/* 角色图片 */}
                                    {charactor.charactor_image && (
                                        <div className="flex-shrink-0">
                                            <img 
                                                src={charactor.charactor_image} 
                                                alt={charactor.charactor_name}
                                                className="w-26 h-40 object-cover rounded-lg"
                                                style={{ objectFit: 'cover', objectPosition: 'center top' }}
                                            />
                                        </div>
                                    )}
                                    
                                    {/* 角色信息 */}
                                    <div className="flex-1 min-w-0">
                                        <h3 className="font-bold text-brand-900 dark:text-white truncate">
                                            <button
                                                    onClick={() => {
                                                        var characterIds = getCharactors(worksMap)
                                                        if (charactor.charactor_id) {
                                                            navigate({ 
                                                                to: `/charactor/${charactor.charactor_id}`, 
                                                                // params: { charactorId: charactor.charactor_id } 
                                                                search: { characterIds },
                                                            });
                                                        }
                                                    }}
                                                    className="text-sm text-brand-700 dark:text-brand-300 hover:text-brand-900 dark:hover:text-white font-medium underline-offset-2 hover:underline transition-colors"
                                                >
                                                    {charactor.charactor_name}
                                                </button>
                                            {/* {charactor.charactor_name} */}
                                        </h3>
                                        
                                        {charactor.staff_name && (
                                            <div className="mt-1">
                                                <span className="text-xs text-brand-600 dark:text-brand-400">{t('gameIntro.cv')}：</span>
                                                <button
                                                    onClick={() => {
                                                        if (charactor.staff_id) {
                                                            navigate({ to: '/staff/$staffId', params: { staffId: charactor.staff_id } });
                                                        }
                                                    }}
                                                    className="text-sm text-brand-700 dark:text-brand-300 hover:text-brand-900 dark:hover:text-white font-medium underline-offset-2 hover:underline transition-colors"
                                                >
                                                    {charactor.staff_name}
                                                </button>
                                            </div>
                                        )}
                                    </div>
                                    
                                </div>
                                
                            ))
                        ) : (
                            <p className="text-brand-600 dark:text-brand-400 text-sm">{t('gameIntro.noCategories')}</p>
                        )}
                    </div>
                </div>

                {/* 关联游戏 */}
                { relatedGames.length > 0 && (
                    <div className="mt-4">
                        <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">{t('gameIntro.relatedGames')}</h3>
                        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4 mt-2">
                            {relatedGames.map((game) => (
                                <GameCard key={game.id} game={game} />
                            ))}
                        </div>
                    </div>
                )}



                



                
            </div>
        );
}
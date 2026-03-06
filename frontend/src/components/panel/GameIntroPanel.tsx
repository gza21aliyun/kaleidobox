
import { appconf, enums, models } from "../../../wailsjs/go/models";
import { useNavigate } from "@tanstack/react-router";
import { GetWorksMapByGameId, CountWorks, GetWorksByGameId } from "../../../wailsjs/go/service/WorkService";
import { tagMapForEach, workMapForEach, charactorsForEach, getCharactors } from "../utils/Utility";
import { GetGamesByRelatedGames } from "../../../wailsjs/go/service/GameService";
import { FetchImages } from "../../../wailsjs/go/service/ImageService";
import { useEffect, useState, useRef } from "react";
import { useTranslation } from "react-i18next";
import { GameCard } from "../card/GameCard";
import { ImageBackupCard } from "../card/ImageCard";

interface GameEditFormProps {
  game: models.Game;
  config?: appconf.AppConfig;
  onTagTaps: (tag: string) => void;
}

const CharactorImages = ({ workId }: { workId: string }) => {
  const { t } = useTranslation();
  const [images, setImages] = useState<models.ImageBackup[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (workId) {
      FetchImages(workId, 3, 1).then((res) => {
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
      <h4 className="text-sm font-medium text-brand-600 dark:text-brand-400 mb-2">{t('gameIntro.characterImages')}</h4>
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
            GetGamesByRelatedGames(game.related_games).then((res) => { 
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
                    <div className="space-y-4">
                        {true ? (
                            
                            charactorsForEach(worksMap, (charactor) => (
                                <div key={`${charactor.charactor_name}-${charactor.staff_name}`} 
                                    className="bg-white dark:bg-brand-800/30 rounded-lg p-4 border border-brand-200 dark:border-brand-700 hover:shadow-md transition-shadow">
                                    <div className="grid grid-cols-1 md:grid-cols-12 gap-4">
                                        {/* 角色图片 */}
                                        {charactor.charactor_image && (
                                            <div className="md:col-span-2">
                                                <img 
                                                    src={charactor.charactor_image} 
                                                    alt={charactor.charactor_name}
                                                    className="w-full h-40 object-cover rounded-lg"
                                                    style={{ objectFit: 'cover', objectPosition: 'center top' }}
                                                />
                                            </div>
                                        )}
                                        
                                        {/* 角色信息 */}
                                        <div className="md:col-span-10">
                                            <div className="flex flex-wrap items-center gap-4 mb-3">
                                                <h3 className="font-bold text-brand-900 dark:text-white text-lg">
                                                    <button
                                                            onClick={() => {
                                                                var characterIds = getCharactors(worksMap)
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
                                                <div className="mb-4">
                                                    <h4 className="text-sm font-medium text-brand-600 dark:text-brand-400 mb-1">{t('gameIntro.characterSummary')}</h4>
                                                    <p className="text-sm text-brand-700 dark:text-brand-300 whitespace-pre-wrap leading-relaxed">
                                                        {charactor.work_summary}
                                                    </p>
                                                </div>
                                            )}
                                            
                                            {/* 角色图片 */}
                                            {charactor.id && (
                                                <CharactorImages workId={charactor.id} />
                                            )}
                                        </div>
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
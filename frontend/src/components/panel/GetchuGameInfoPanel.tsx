import { models, enums } from "../../../wailsjs/go/models";
import { FetchMetadata } from "../../../wailsjs/go/service/GameService";
import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { ImageCard } from "../card/ImageCard";
import { workMapForEach, charactorsForEach } from "../utils/Utility";
import toast from "react-hot-toast";

interface GetchuGameInfoPanelProps {
  getchuId: string;
  gameName: string;
  company: string;
  coverURL: string;
  onClose: () => void;
}

export function GetchuGameInfoPanel({ getchuId, gameName, company, coverURL, onClose }: GetchuGameInfoPanelProps) {
  const { t } = useTranslation();
  const [game, setGame] = useState<models.Game | null>(null);
  const [worksMap, setWorksMap] = useState<Map<enums.StaffRole, models.Work[]>>(new Map());
  const [images, setImages] = useState<string[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [imgError, setImgError] = useState(false);

  useEffect(() => {
    fetchGameInfo();
  }, [getchuId]);

  const fetchGameInfo = async () => {
    if (!getchuId) return;
    
    setIsLoading(true);
    try {
      const request = {
        id: getchuId,
        source: "getchu",
        should_fetch_charactors: true,
        should_fetch_staffs: true,
        should_fetch_images: true,
        is_overwrite: false,
      };

      const fetchedGame = await FetchMetadata(request as any);
      setGame(fetchedGame);

      if (fetchedGame.images) {
        setImages(fetchedGame.images.split(",").filter((img: string) => img.trim() !== ""));
      }

      const workMap = new Map<enums.StaffRole, models.Work[]>();
      setWorksMap(workMap);
    } catch (error) {
      console.error("Failed to fetch game info:", error);
      toast.error(t("gameInfo.fetchFailed") || "获取游戏信息失败");
    } finally {
      setIsLoading(false);
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-10">
        <div className="flex flex-col items-center gap-4">
          <div className="i-mdi-loading text-5xl text-primary-500 animate-spin" />
          <p className="text-brand-500 dark:text-brand-400">{t("gameInfo.loading") || "加载中"}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex gap-6">
        {coverURL && (
          <div className="flex-shrink-0 w-48 h-64">
            {coverURL && !imgError ? (
              <img
                src={coverURL}
                alt={gameName}
                className="w-full h-full object-cover rounded-lg shadow-md"
                onError={() => setImgError(true)}
              />
            ) : (
              <div className="w-full h-full flex items-center justify-center bg-brand-100 dark:bg-brand-700 rounded-lg">
                <div className="i-mdi-image-off text-4xl text-brand-400" />
              </div>
            )}
          </div>
        )}
        <div className="flex-1">
          <h2 className="text-2xl font-bold text-brand-900 dark:text-white mb-2">{gameName}</h2>
          <p className="text-brand-600 dark:text-brand-400">{company}</p>
          {game && game.release_at && (
            <p className="text-sm text-brand-500 dark:text-brand-400 mt-2">
              {t("gameInfo.releaseDate") || "发售日期"}: {new Date(game.release_at.toString()).toLocaleDateString()}
            </p>
          )}
        </div>
      </div>

      <div className="bg-gradient-to-br from-brand-50 to-brand-100 dark:from-brand-900/30 dark:to-brand-800/20 rounded-xl p-5 border border-brand-200 dark:border-brand-700">
        <h3 className="text-lg font-semibold text-brand-900 dark:text-white mb-3">{t("gameIntro.summary") || "简介"}</h3>
        <p className="text-sm text-brand-700 dark:text-brand-300 leading-relaxed whitespace-pre-wrap">
          {game?.summary || t("gameIntro.noSummary") || "暂无简介"}
        </p>
      </div>

      <div>
        <h3 className="text-lg font-semibold text-brand-900 dark:text-white mb-3">{t("gameIntro.characters") || "角色"}</h3>
        <div className="space-y-4">
          {worksMap.has(enums.StaffRole.CHARACTOR) || worksMap.has(enums.StaffRole.CV) ? (
            charactorsForEach(worksMap, (charactor) => (
              <div
                key={`${charactor.charactor_name}-${charactor.staff_name}`}
                className="bg-white dark:bg-brand-800/30 rounded-lg p-4 border border-brand-200 dark:border-brand-700"
              >
                <div className="flex flex-col xl:flex-row gap-4">
                  {charactor.charactor_image && (
                    <div className="flex-shrink-0 w-32 h-48">
                      <ImageCard
                        url={charactor.charactor_image}
                        alt={charactor.charactor_name}
                        className="w-full h-full object-cover rounded-lg"
                      />
                    </div>
                  )}
                  <div className="flex-1">
                    <div className="flex items-center gap-4 mb-2">
                      <h4 className="font-bold text-brand-900 dark:text-white">
                        {charactor.charactor_name || t("gameInfo.unknownCharacter") || "未知角色"}
                      </h4>
                      {charactor.staff_name && (
                        <span className="text-sm text-brand-600 dark:text-brand-400">
                          {t("gameIntro.cv") || "CV"}: {charactor.staff_name}
                        </span>
                      )}
                    </div>
                    {charactor.work_summary && (
                      <p className="text-sm text-brand-600 dark:text-brand-400 whitespace-pre-wrap">
                        {charactor.work_summary}
                      </p>
                    )}
                  </div>
                </div>
              </div>
            ))
          ) : (
            <p className="text-brand-600 dark:text-brand-400 text-sm">{t("gameIntro.noCategories") || "暂无角色信息"}</p>
          )}
        </div>
      </div>

      <div>
        <h3 className="text-lg font-semibold text-brand-900 dark:text-white mb-3">{t("gameGallery.gallery") || "截图"}</h3>
        {images.length > 0 ? (
          <div className="grid grid-cols-2 xl:grid-cols-3 gap-3">
            {images.map((img, index) => (
              <ImageCard
                key={index}
                url={img}
                alt={`${gameName} ${index + 1}`}
                className="w-full aspect-video object-cover rounded-lg"
              />
            ))}
          </div>
        ) : (
          <p className="text-brand-600 dark:text-brand-400 text-sm">{t("gameGallery.noScreenshots") || "暂无截图"}</p>
        )}
      </div>

      <div className="mt-6 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
        <h3 className="text-lg font-semibold mb-3 text-brand-900 dark:text-white">{t("gameInfo.staffInfo") || "制作人员"}</h3>
        {worksMap.size > 0 ? (
          <div className="space-y-4">
            {workMapForEach(worksMap, t, (role, works) => (
              <div key={role} className="border-l-4 border-brand-500 pl-4">
                <h4 className="font-medium text-brand-800 dark:text-brand-200 capitalize">
                  {role}
                </h4>
                {works && works.length > 0 ? (
                  <ul className="mt-2 flex flex-wrap gap-2">
                    {works.map((work: models.Work, index: number) => (
                      <li
                        key={index}
                        className="text-sm text-brand-600 dark:text-brand-400 bg-brand-50 dark:bg-brand-900/50 px-2 py-1 rounded"
                      >
                        {work.role === enums.StaffRole.CV
                          ? `${work.staff_name || ''} (${work.charactor_name || t('gameInfo.unknownCharacter') || '未知角色'})`
                          : work.staff_name || t('gameInfo.staff', { index: index + 1 }) || `工作人员 ${index + 1}`
                        }
                      </li>
                    ))}
                  </ul>
                ) : (
                  <p className="text-sm text-brand-500 dark:text-brand-400 italic mt-1">{t("gameInfo.noData") || "暂无数据"}</p>
                )}
              </div>
            ))}
          </div>
        ) : (
          <p className="text-brand-600 dark:text-brand-400 text-sm">{t("gameInfo.noStaffInfo") || "暂无制作人员信息"}</p>
        )}
      </div>
    </div>
  );
}
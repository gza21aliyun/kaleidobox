import type { models, utils } from "../../../wailsjs/go/models";
import { useCallback, useEffect, useState } from "react";
import toast from "react-hot-toast";
import { useTranslation } from "react-i18next";
import { FetchGameGuide,DownloadFileInDownload, DownloadSaveInDownload } from "../../../wailsjs/go/service/BackupService";
import { GetGameByID } from "../../../wailsjs/go/service/GameService";
import { OpenBrowser } from '../../../wailsjs/go/service/ImportService';
import { BetterButton } from "../ui/BetterButton";
import { OverrideSaveModal } from "../modal/OverrideSaveModal";
import { OverrideSaves } from "../../../wailsjs/go/service/DownloadedFilesService";

interface GuidePanelProps {
  game: models.Game;
}

export function GuidePanel({ game }: GuidePanelProps) {
  const { t } = useTranslation();
  const [guideContent, setGuideContent] = useState<models.GuideContent | null>(null);
  const [loading, setLoading] = useState(true);
  const [isOverrideSaveModalOpen, setIsOverrideSaveModalOpen] = useState(false);
  const [overrideSaveResults, setOverrideSaveResults] = useState<utils.SaveResult[]>([]);

  const loadGuideContent = useCallback(async () => {
    setLoading(true);
    try {
      if (game) {
        const data = await FetchGameGuide(game);
        setGuideContent(data);
      }
    }
    catch (err) {
      console.error("Failed to load guide content:", err);
      toast.error(t('guide.loadError'));
    }
    finally {
      setLoading(false);
    }
  }, [game]);

  useEffect(() => {
    loadGuideContent();
  }, [game]);

  const downloadSave = async () => { 
    if (!guideContent || !guideContent.SaveLink || guideContent.SaveLink === "") {
      
      return;
    }
    try {
      // await DownloadFileInDownload(guideContent?.SaveLink ?? "");
      const rs = await DownloadSaveInDownload(guideContent?.SaveLink ?? "", game);
      setOverrideSaveResults([rs]);
      setIsOverrideSaveModalOpen(true);
      toast.success(t('guide.downloadSuccess'));
    } catch (err) {
      console.error("Failed to download", err);
      toast.error(t('guide.downloadError'));
    }
  };

  const handleConfirmOverrideSaves = async (selectedResults: utils.SaveResult[]) => {
      const ids = selectedResults.map((i)=>i.game_id)
      try {
        await OverrideSaves(selectedResults);
        toast.success(t("downloadedFiles.overrideSaveSuccess") || '覆盖存档成功');
      } catch (err) {
        console.error("Override saves failed:", err);
      } finally {
        setOverrideSaveResults([]);
      }
    };

  return (
    <div className="space-y-6">
      <div className="glass-card bg-white dark:bg-brand-800 p-6 rounded-lg shadow-sm">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h3 className="text-xl font-semibold text-brand-900 dark:text-white">{t('common.guide')}</h3>
            <p className="text-sm text-brand-500 dark:text-brand-400 mt-1">
              {t('guide.description')}
            </p>
          </div>
        </div>

        {loading
          ? (
              <div className="flex flex-col items-center justify-center py-12">
                <div className="i-mdi-loading animate-spin text-3xl text-brand-500 mb-4" />
                <p className="text-brand-500 dark:text-brand-400">{t('common.loading')}</p>
              </div>
            )
          : guideContent
            ? (
                <div className="space-y-6">
                  {/* 攻略标题 */}
                  <div className="border-b border-brand-200 dark:border-brand-700 pb-4">
                    <h2 className="text-2xl font-bold text-brand-900 dark:text-white">{guideContent.Name}</h2>
                  </div>
                  
                  {/* 操作按钮 */}
                  {guideContent.SaveLink || guideContent.Link ? (
                    <div className="flex flex-wrap gap-3">
                      {guideContent.SaveLink && (
                        <BetterButton 
                          onClick={downloadSave}
                          icon="i-mdi-folder-download"
                          title={t('guide.downloadSave')}
                          className="bg-brand-600 hover:bg-brand-700 text-white"
                        />
                      )}
                      {guideContent.Link && (
                        <BetterButton 
                          onClick={() => OpenBrowser(guideContent.Link)}
                          icon="i-mdi-folder-open"
                          title={t('guide.openWeb')}
                          className="bg-neutral-600 hover:bg-neutral-700 text-white"
                        />
                      )}
                    </div>
                  ) : null}
                  
                  {/* 攻略文本 */}
                  {guideContent.Text && (
                    <div className="glass-card bg-brand-50 dark:bg-brand-900/50 p-4 rounded-lg">
                      <table
                   dangerouslySetInnerHTML={{ __html: guideContent.Text }}
                  >
                  </table>
                    </div>
                  )}
                  
                  {/* 攻略内容 */}
                  {guideContent.Content && (
                    <div className="prose dark:prose-invert max-w-none">
                      <div dangerouslySetInnerHTML={{ __html: guideContent.Content }} />
                    </div>
                  )}
                </div>
              )
            : (
                <div className="flex flex-col items-center justify-center py-12 text-center">
                  <div className="i-mdi-information-outline text-4xl text-brand-400 mb-4" />
                  <p className="text-brand-500 dark:text-brand-400 text-lg">{t('guide.noContent')}</p>
                </div>
              )}
      </div>

      {/* 覆盖存档确认弹窗 */}
            <OverrideSaveModal
              isOpen={isOverrideSaveModalOpen}
              results={overrideSaveResults}
              onClose={() => {
                setIsOverrideSaveModalOpen(false);
                setOverrideSaveResults([]);
              }}
              onConfirm={handleConfirmOverrideSaves}
            />
    </div>
  );
}

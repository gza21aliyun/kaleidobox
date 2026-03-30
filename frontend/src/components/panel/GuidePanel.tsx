import type { models } from "../../../wailsjs/go/models";
import { useCallback, useEffect, useState } from "react";
import toast from "react-hot-toast";
import { useTranslation } from "react-i18next";
import { FetchGameGuide,DownloadFileInDownload } from "../../../wailsjs/go/service/BackupService";
import { GetGameByID } from "../../../wailsjs/go/service/GameService";
import { OpenBrowser } from '../../../wailsjs/go/service/ImportService';
import { BetterButton } from "../ui/BetterButton";

interface GuidePanelProps {
  game: models.Game;
}

export function GuidePanel({ game }: GuidePanelProps) {
  const { t } = useTranslation();
  const [guideContent, setGuideContent] = useState<models.GuideContent | null>(null);
  const [loading, setLoading] = useState(true);

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
      await DownloadFileInDownload(guideContent?.SaveLink ?? "");
      toast.success(t('guide.downloadSuccess'));
    } catch (err) {
      console.error("Failed to download", err);
      toast.error(t('guide.downloadError'));
    }
  };

  return (
    <div className="space-y-6">
      <div className="glass-card bg-white dark:bg-brand-800 p-6 rounded-lg shadow-sm">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-lg font-semibold text-brand-900 dark:text-white">{t('common.guide')}</h3>
            <p className="text-sm text-brand-500 dark:text-brand-400 mt-1">
              {t('guide.description')}
            </p>
          </div>
        </div>

        {loading
          ? (
              <div className="flex justify-center py-8">
                <div className="i-mdi-loading animate-spin text-2xl text-brand-500" />
              </div>
            )
          : guideContent
            ? (
                <div className="prose dark:prose-invert max-w-none">
                  <div><h6><b>{guideContent.Name}</b></h6></div>
                  <br/>
                  {guideContent.SaveLink && (
                    <BetterButton onClick={() => { 
                      downloadSave()
                    }} icon="i-mdi-folder-download" title="下载存档"/>
                  )}
                  {guideContent.Link && (
                    <BetterButton onClick={() => { 
                      OpenBrowser(guideContent.Link)
                    }}
                    icon="i-mdi-folder-open" title="打开网页"/>
                  )}
                  <br/>
                  <table
                   dangerouslySetInnerHTML={{ __html: guideContent.Text }}
                  >
                    {/* <p className="text-sm text-brand-700 dark:text-brand-300 whitespace-pre-wrap leading-relaxed">
                                                        {guideContent.Text.replace('\n\n', '\n')}
                                                    </p> */}
                  </table>
                  <br/>
                  <div dangerouslySetInnerHTML={{ __html: guideContent.Content }} />
                  {/* <table border={1} bordercolor={"#66ccff"} bgcolor="#ffffff" height="40" cellspacing="0" dangerouslySetInnerHTML={{ __html: guideContent.Content }} /> */}
                  {/* <div>{guideContent.Text}</div> */}
                </div>
              )
            : (
                <div className="text-center py-8 text-brand-500">{t('guide.noContent')}</div>
              )}
      </div>
    </div>
  );
}

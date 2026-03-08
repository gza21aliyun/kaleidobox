import { models, enums } from "../../../wailsjs/go/models";
import { 
  GetGlobalHotkeys, UpdateHotkey, 
  AddHotkey } from "../../../wailsjs/go/service/HotkeyService";
import { OpenLocalPath } from "../../../wailsjs/go/service/GameService";
import { FetchImages } from "../../../wailsjs/go/service/ImageService";
import { useState, useEffect } from "react";
import { ScreenshotHotkeyModal } from "../modal/ScreenshotHotkeyModal";
import { ImageBackupCard } from "../card/ImageCard";
import { arrayMapString } from "../utils/Utility";
import { useTranslation } from 'react-i18next';
import { BetterButton } from "../ui/BetterButton";
import { toast } from "react-hot-toast";
import { arrayContains } from "../utils/Utility";
// import { 
//   DeviceType, 
//   HotkeyActionType, 
//   ModifierKey, 
//   Hotkey 
// } from "../utils/Hotkeys";

interface GameGalleryPanelProps {
  game: models.Game;
}

export function GameGalleryPanel({ game }: GameGalleryPanelProps) {
  const { t } = useTranslation();
  // 截图相关状态
  const [screenshotHotkey, setScreenshotHotkey] = useState<models.Hotkey | null>(null);
  const [isHotkeyModalOpen, setIsHotkeyModalOpen] = useState(false);
  const [screenshots, setScreenshots] = useState<models.ImageBackup[]>([]);
  const [images, setImages] = useState<models.ImageBackup[]>([]);
  const [loading, setLoading] = useState(true);

  // 检查是否有图片且过滤条件满足
  const hasImages = images && images.length > 0;
  
  // 加载截图快捷键配置
  useEffect(() => {
    loadScreenshotHotkey();
    loadScreenshots();
  }, [game.id]);

  const loadScreenshotHotkey = async () => {
    try {
      // 获取所有全局快捷键
      const hotkeys = await GetGlobalHotkeys();
      const screenshotHotkey = hotkeys.find(h => h.action_type === enums.HotkeyActionType.SCREENSHOT);
      setScreenshotHotkey(screenshotHotkey || null);
    } catch (error) {
      console.error("加载截图快捷键失败:", error);
    }
  };

  const loadScreenshots = async () => {
    try {
      setLoading(true);
      // 这里应该调用获取游戏截图的API
      // 暂时使用空数组
      let rs = await FetchImages(game.id, 0, 3);
      console.log("获取游戏截图成功:", rs);
      setScreenshots(rs ?? []);
      let imgs = await FetchImages(game.id, 0, 2);
      setImages(imgs ?? []);
    } catch (error) {
      console.error("加载截图失败:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleSaveHotkey = async (hotkey: models.Hotkey) => {
    try {
      if (screenshotHotkey) {
        // 更新现有快捷键
        await UpdateHotkey(new models.Hotkey({
          ...hotkey,
          id: screenshotHotkey.id,
          // game_id: "global",
          updated_at: new Date()
        }));
      } else {
        // 创建新快捷键
        await AddHotkey(hotkey);
      }
      setScreenshotHotkey(hotkey);
    } catch (error) {
      console.error("保存快捷键失败:", error);
      throw error;
    }
  };


  // 处理图片数组，过滤封面图和特定后缀的图片
  const galleryImages = hasImages
    ? images
        .filter(img => !img.url.endsWith("pl.jpg"))
        .filter(img => img.url.trim() !== "") // 过滤空字符串
    : [];

    const imagesPath = `images/${game.id}/`;
    console.log("galleryImages:", galleryImages)

    const hasFolder = arrayContains(images, (img) => img.local_path !== "")

  // 如果没有任何内容显示，不显示面板
  // if (!screenshotHotkey && galleryImages.length === 0) {
  //   return null;
  // }

  return (
    <div className="flex flex-col gap-4">
      {/* 截图板块 */}
      {true && (
        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <div className="font-semibold text-brand-900 dark:text-white">{t('gameGallery.screenshots')}</div>
            <div className="flex items-center gap-2">
              {screenshotHotkey && (
                <span className="px-2 py-1 bg-brand-100 text-brand-700 text-sm rounded dark:bg-brand-900/30 dark:text-brand-300">
                  {screenshotHotkey.device_type === enums.DeviceType.KEYBOARD && screenshotHotkey.modifiers?.length > 0
                    ? `${screenshotHotkey.device_type}   ${screenshotHotkey.modifiers.join(" + ")} + ${screenshotHotkey.name}`
                    : `${screenshotHotkey.device_type}   ${screenshotHotkey.name}`
                  }
                </span>
              )}
              <button
                onClick={() => setIsHotkeyModalOpen(true)}
                className="flex items-center gap-1 px-3 py-1 text-sm font-medium text-brand-700 bg-brand-100 hover:bg-brand-200 rounded-lg dark:text-brand-300 dark:bg-brand-900/30 dark:hover:bg-brand-800 transition-colors"
              >
                <div className="i-mdi-keyboard-settings text-base" />
                {screenshotHotkey ? t('gameGallery.editHotkey') : t('gameGallery.setHotkey')}
              </button>
            </div>
          </div>
          
          {loading ? (
            <div className="text-brand-600 dark:text-brand-400 text-sm">{t('gameGallery.loading')}</div>
          ) : screenshots.length > 0 ? (
            <div className="grid grid-cols-3 gap-2">
              {screenshots.map((screenshot, index) => (
                <ImageBackupCard
                  imageBackup={screenshot}
                  key={screenshot.url}
                  />
              ))}
            </div>
          ) : (
            <div className="text-brand-600 dark:text-brand-400 text-sm italic">
              {screenshotHotkey 
                ? t('gameGallery.useHotkey', { 
                    hotkey: screenshotHotkey.device_type === enums.DeviceType.KEYBOARD && screenshotHotkey.modifiers?.length > 0
                      ? `${screenshotHotkey.modifiers.join(" + ")} + ${screenshotHotkey.key_code}`
                      : screenshotHotkey.key_code
                  })
                : t('gameGallery.noScreenshots')}
            </div>
          )}
        </div>
      )}

      {/* 画廊板块 */}
      {galleryImages.length > 0 && (
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-2">
                <span className="text-lg font-bold">{t('gameGallery.gallery')}{`(${images.length})`}</span>
                {hasFolder && (
                    <BetterButton
                      onClick={async () => {
                        try {
                          await OpenLocalPath(imagesPath);
                        }
                        catch {
                          toast.error(t('gameGallery.openPathError'));
                        }
                      }}
                      disabled={!game.path}
                      icon="i-mdi-folder-open"
                      title={t('gameGallery.openLocation')}
                    />
                )}
            </div>
          <div className="grid grid-cols-3 gap-2">
            {galleryImages.map((image, index) => (
                <ImageBackupCard
                  imageBackup={image}
                  key={image.url}
                  />
              ))}
          </div>
        </div>
      )}

      {/* 截图快捷键设置模态框 */}
      <ScreenshotHotkeyModal
        isOpen={isHotkeyModalOpen}
        onClose={() => setIsHotkeyModalOpen(false)}
        onSave={handleSaveHotkey}
        currentHotkey={screenshotHotkey || undefined}
      />
    </div>
  );
}
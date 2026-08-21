import { useState } from "react";
import { toast } from "react-hot-toast";
import { useTranslation } from "react-i18next";
import { CreateOrUpdateImageUrl, PickLocalImage } from "../../../wailsjs/go/service/ImageService";

interface SingleImageFieldProps {
  label: string;
  value: string;
  onChange: (url: string) => void;
  previewClassName?: string;
  /** 图片所属实体的 ID（charactorId/staffId/workId/gameId） */
  subjectId?: string;
  /** 0=游戏 1=人物 2=工作人员 3=作品 */
  subjectType?: number;
  /** 0=主图 1=图集 2=官图 3=截图 */
  imageType?: number;
  /** 关联的游戏 ID，用于图片存放路径 */
  gameId?: string;
}

export function SingleImageField({
  label,
  value,
  onChange,
  previewClassName = "w-28 h-36",
  subjectId = "",
  subjectType = 0,
  imageType = 0,
  gameId = "",
}: SingleImageFieldProps) {
  const { t } = useTranslation();
  const [urlInputOpen, setUrlInputOpen] = useState(false);
  const [urlInput, setUrlInput] = useState("");
  const [picking, setPicking] = useState(false);
  const [registering, setRegistering] = useState(false);

  // 从本地文件系统选择图片：后端弹文件对话框，复制到 images 目录，返回本地路径
  const handlePickFile = async () => {
    setPicking(true);
    try {
      const path = await PickLocalImage(subjectId, subjectType, imageType, gameId);
      if (path) {
        onChange(path);
      }
    } catch (err) {
      console.error("PickLocalImage failed:", err);
      toast.error(t("common.error"));
    } finally {
      setPicking(false);
    }
  };

  // 填写图片网址：注册到 image_backups，然后写入字段
  const handleUrlSubmit = async () => {
    const url = urlInput.trim();
    if (!url) {
      setUrlInputOpen(false);
      return;
    }
    setRegistering(true);
    try {
      await CreateOrUpdateImageUrl(url);
      onChange(url);
      setUrlInputOpen(false);
      setUrlInput("");
    } catch (err) {
      console.error("CreateOrUpdateImageUrl failed:", err);
      toast.error(t("common.error"));
    } finally {
      setRegistering(false);
    }
  };

  const btnClass =
    "inline-flex items-center gap-1 px-2.5 py-1.5 text-xs font-medium rounded-md border border-brand-300 dark:border-brand-600 text-brand-700 dark:text-brand-300 hover:bg-brand-50 dark:hover:bg-brand-700/50 transition-colors disabled:opacity-50";

  return (
    <div>
      <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1">
        {label}
      </label>
      <div className="flex gap-3 items-start">
        {/* 预览 */}
        <div className={previewClassName + " flex-shrink-0 rounded-md overflow-hidden border border-brand-200 dark:border-brand-600"}>
          {value ? (
            <img
              src={value}
              alt={label}
              className="w-full h-full object-cover"
              onError={(e) => {
                const el = e.currentTarget as HTMLImageElement;
                el.style.display = "none";
                const parent = el.parentElement;
                if (parent) {
                  parent.className += " flex items-center justify-center bg-brand-100 dark:bg-brand-700";
                  parent.innerHTML = '<div class="i-mdi-image-broken text-2xl text-brand-400"></div>';
                }
              }}
            />
          ) : (
            <div className="w-full h-full flex items-center justify-center bg-brand-100 dark:bg-brand-700">
              <div className="i-mdi-image-outline text-3xl text-brand-400" />
            </div>
          )}
        </div>

        {/* 按钮 + 网址输入 */}
        <div className="flex-1 space-y-2">
          <div className="flex gap-2 flex-wrap">
            <button
              type="button"
              onClick={handlePickFile}
              disabled={picking}
              className={btnClass}
            >
              <div className={picking ? "i-mdi-loading animate-spin" : "i-mdi-folder-open-outline"} />
              <span>{t("imageField.selectImage")}</span>
            </button>
            <button
              type="button"
              onClick={() => {
                setUrlInput(value || "");
                setUrlInputOpen(!urlInputOpen);
              }}
              className={btnClass}
            >
              <div className="i-mdi-link-variant" />
              <span>{t("imageField.fillUrl")}</span>
            </button>
            {value && (
              <button
                type="button"
                onClick={() => onChange("")}
                className={btnClass + " text-red-500 border-red-300 dark:border-red-700 hover:bg-red-50 dark:hover:bg-red-700/30"}
              >
                <div className="i-mdi-delete-outline" />
                <span>{t("common.delete")}</span>
              </button>
            )}
          </div>

          {urlInputOpen && (
            <div className="flex gap-2">
              <input
                type="text"
                value={urlInput}
                onChange={(e) => setUrlInput(e.target.value)}
                placeholder={t("imageField.urlPlaceholder")}
                className="flex-1 px-3 py-1.5 text-sm border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    handleUrlSubmit();
                  }
                }}
                autoFocus
              />
              <button
                type="button"
                onClick={handleUrlSubmit}
                disabled={registering}
                className="px-3 py-1.5 text-sm font-medium text-white bg-neutral-600 hover:bg-neutral-700 rounded-md transition-colors disabled:opacity-50"
              >
                {registering ? t("common.loading") : t("common.ok")}
              </button>
            </div>
          )}

          {value && !urlInputOpen && (
            <div className="text-xs text-brand-400 dark:text-brand-500 truncate" title={value}>
              {value}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

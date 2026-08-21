import { useState } from "react";
import { toast } from "react-hot-toast";
import { useTranslation } from "react-i18next";
import { CreateOrUpdateImageUrl, PickLocalImage } from "../../../wailsjs/go/service/ImageService";

interface MultiImageFieldProps {
  label: string;
  /** 以逗号分隔的图片 URL 字符串 */
  value: string;
  onChange: (value: string) => void;
  /** 单个缩略图尺寸 */
  thumbClassName?: string;
  /** 图片所属实体的 ID */
  subjectId?: string;
  /** 0=游戏 1=人物 2=工作人员 3=作品 */
  subjectType?: number;
  /** 0=主图 1=图集 2=官图 3=截图 */
  imageType?: number;
  /** 关联的游戏 ID，用于图片存放路径 */
  gameId?: string;
}

// 将逗号分隔的字符串拆分为 URL 数组（自动去除空值与首尾空格）
const splitUrls = (raw: string): string[] =>
  (raw || "")
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);

const joinUrls = (urls: string[]): string => urls.join(",");

export function MultiImageField({
  label,
  value,
  onChange,
  thumbClassName = "w-28 h-36",
  subjectId = "",
  subjectType = 0,
  imageType = 1,
  gameId = "",
}: MultiImageFieldProps) {
  const { t } = useTranslation();
  const [urlInputOpen, setUrlInputOpen] = useState(false);
  const [urlInput, setUrlInput] = useState("");
  const [picking, setPicking] = useState(false);
  const [registering, setRegistering] = useState(false);

  const urls = splitUrls(value);

  const commit = (next: string[]) => onChange(joinUrls(next));

  const handlePickFile = async () => {
    setPicking(true);
    try {
      const path = await PickLocalImage(subjectId, subjectType, imageType, gameId);
      if (path) {
        commit([...urls, path]);
      }
    } catch (err) {
      console.error("PickLocalImage failed:", err);
      toast.error(t("common.error"));
    } finally {
      setPicking(false);
    }
  };

  const handleUrlSubmit = async () => {
    const url = urlInput.trim();
    if (!url) {
      setUrlInputOpen(false);
      return;
    }
    setRegistering(true);
    try {
      await CreateOrUpdateImageUrl(url);
      commit([...urls, url]);
      setUrlInputOpen(false);
      setUrlInput("");
    } catch (err) {
      console.error("CreateOrUpdateImageUrl failed:", err);
      toast.error(t("common.error"));
    } finally {
      setRegistering(false);
    }
  };

  const handleDelete = (idx: number) => {
    commit(urls.filter((_, i) => i !== idx));
  };

  const btnClass =
    "inline-flex items-center gap-1 px-2.5 py-1.5 text-xs font-medium rounded-md border border-brand-300 dark:border-brand-600 text-brand-700 dark:text-brand-300 hover:bg-brand-50 dark:hover:bg-brand-700/50 transition-colors disabled:opacity-50";

  return (
    <div>
      <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1">
        {label}
      </label>

      <div className="flex gap-2 flex-wrap mb-3">
        <button
          type="button"
          onClick={handlePickFile}
          disabled={picking}
          className={btnClass}
        >
          <div className={picking ? "i-mdi-loading animate-spin" : "i-mdi-folder-open-outline"} />
          <span>{t("imageField.selectAddImage")}</span>
        </button>
        <button
          type="button"
          onClick={() => {
            setUrlInput("");
            setUrlInputOpen(!urlInputOpen);
          }}
          className={btnClass}
        >
          <div className="i-mdi-link-variant-plus" />
          <span>{t("imageField.fillAddUrl")}</span>
        </button>
      </div>

      {urlInputOpen && (
        <div className="flex gap-2 mb-3">
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

      {urls.length === 0 ? (
        <div className="text-xs text-brand-400 dark:text-brand-500">
          {t("imageField.noImages")}
        </div>
      ) : (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(7rem,1fr))] gap-3">
          {urls.map((url, idx) => (
            <div
              key={url + idx}
              className={"group relative rounded-md overflow-hidden border border-brand-200 dark:border-brand-600 " + thumbClassName}
            >
              <img
                src={url}
                alt={url}
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
              <button
                type="button"
                onClick={() => handleDelete(idx)}
                className="absolute top-1 right-1 p-1 rounded-full bg-black/60 text-white opacity-0 group-hover:opacity-100 transition-opacity hover:bg-red-600"
                title={t("common.delete")}
              >
                <div className="i-mdi-close text-sm" />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

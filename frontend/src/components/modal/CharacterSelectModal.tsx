import { useEffect, useRef, useState } from "react";
import { toast } from "react-hot-toast";
import { useTranslation } from "react-i18next";
import { enums, models } from "../../../wailsjs/go/models";
import {
  SearchCharactors,
  CreateCharactor,
  UpdateCharactor,
  GetCharactorById,
} from "../../../wailsjs/go/service/CharactorService";
import { SingleImageField } from "../field/SingleImageField";
import { MultiImageField } from "../field/MultiImageField";

interface CharacterSelectModalProps {
  currentCharactorId?: string;
  currentWorkName?: string;
  onSelect: (charactor: models.Charactor, workName: string) => void;
  onClose: () => void;
}

const SEARCH_LIMIT = 50;

const emptyCharactor = (): models.Charactor => ({
  id: crypto.randomUUID(),
  name: "",
  other_names: "",
  image_path: "",
  images: "",
  source_charactor_id: "",
  source_type: "" as enums.SourceType,
  game_ids: "",
  summary: "",
  gender: 0,
  measurements: "",
  height: "",
  sort: 0,
});

// 将 newName 合并进 comma 分隔的 names 字符串（去重）
const mergeName = (existing: string, newName: string): string => {
  if (!newName) return existing || "";
  const parts = (existing || "")
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
  if (!parts.includes(newName)) {
    parts.push(newName);
  }
  return parts.join(",");
};

export function CharacterSelectModal({
  currentCharactorId,
  currentWorkName,
  onSelect,
  onClose,
}: CharacterSelectModalProps) {
  const { t } = useTranslation();
  const [searchResults, setSearchResults] = useState<models.Charactor[]>([]);
  const [loading, setLoading] = useState(true);
  const [keyword, setKeyword] = useState("");
  const [selected, setSelected] = useState<models.Charactor>(emptyCharactor());
  const [workName, setWorkName] = useState(currentWorkName || "");
  const [isNew, setIsNew] = useState(false);
  const [saving, setSaving] = useState(false);

  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const reqIdRef = useRef(0);

  const runSearch = (kw: string) => {
    const reqId = ++reqIdRef.current;
    setLoading(true);
    SearchCharactors(kw, SEARCH_LIMIT)
      .then((res) => {
        if (reqId !== reqIdRef.current) return;
        const list = res || [];
        setSearchResults(list);
        setSelected((prev) => {
          if (prev.id) return prev;
          return list.length > 0 ? { ...list[0] } : emptyCharactor();
        });
      })
      .catch((err) => {
        console.error("SearchCharactors failed:", err);
        if (reqId === reqIdRef.current) {
          toast.error(t("common.error"));
          setSearchResults([]);
        }
      })
      .finally(() => {
        if (reqId === reqIdRef.current) setLoading(false);
      });
  };

  useEffect(() => {
    const init = async () => {
      if (currentCharactorId) {
        try {
          const cur = await GetCharactorById(currentCharactorId);
          if (cur && cur.id) {
            setSelected({ ...cur });
            setIsNew(false);
            if (!currentWorkName) setWorkName(cur.name);
          } else {
            setIsNew(true);
          }
        } catch (err) {
          console.error("GetCharactorById failed:", err);
          setIsNew(true);
        }
      } else {
        setIsNew(true);
      }
      runSearch("");
    };
    init();
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
      reqIdRef.current++;
    };
  }, [currentCharactorId]);

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      runSearch(keyword.trim());
    }, 300);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [keyword]);

  const update = <K extends keyof models.Charactor>(
    key: K,
    value: models.Charactor[K]
  ) => {
    setSelected((prev) => ({ ...prev, [key]: value }));
  };

  const handleSelect = (c: models.Charactor) => {
    setSelected({ ...c });
    setIsNew(false);
    // 如果 workName 为空或是上一个选中项的名字，自动填充为新选中项的名字
    setWorkName((prev) => prev || c.name);
  };

  const handleAddNew = () => {
    setSelected(emptyCharactor());
    setIsNew(true);
    setKeyword("");
  };

  // 仅保存角色记录，不影响 work
  const handleSave = async () => {
    if (!selected.name.trim()) {
      toast.error(t("characterSelect.nameRequired"));
      return;
    }
    setSaving(true);
    try {
      const saved = { ...selected };
      if (isNew || !saved.id) {
        if (!saved.id) saved.id = crypto.randomUUID();
        await CreateCharactor(saved);
        toast.success(t("common.createSuccess"));
        setSearchResults((prev) => [saved, ...prev]);
      } else {
        await UpdateCharactor(saved);
        toast.success(t("common.success"));
        setSearchResults((prev) =>
          prev.map((it) => (it.id === saved.id ? saved : it))
        );
      }
      setSelected({ ...saved });
      onClose();
    } catch (err) {
      console.error("Save charactor failed:", err);
      toast.error(t("common.error"));
    } finally {
      setSaving(false);
    }
  };

  // 保存角色记录 + 将作品名应用到 work
  const handleApply = async () => {
    if (!selected.name.trim()) {
      toast.error(t("characterSelect.nameRequired"));
      return;
    }
    if (!workName.trim()) {
      toast.error(t("characterSelect.workNameRequired"));
      return;
    }
    setSaving(true);
    try {
      const saved = { ...selected };
      // 将作品名合并进 other_names（自动记录所有不同名字）
      saved.other_names = mergeName(saved.other_names, workName.trim());
      if (isNew || !saved.id) {
        if (!saved.id) saved.id = crypto.randomUUID();
        await CreateCharactor(saved);
        toast.success(t("common.createSuccess"));
        setSearchResults((prev) => [saved, ...prev]);
      } else {
        await UpdateCharactor(saved);
        toast.success(t("common.success"));
        setSearchResults((prev) =>
          prev.map((it) => (it.id === saved.id ? saved : it))
        );
      }
      setSelected({ ...saved });
      onSelect(saved, workName.trim());
      onClose();
    } catch (err) {
      console.error("Apply charactor failed:", err);
      toast.error(t("common.error"));
    } finally {
      setSaving(false);
    }
  };

  const inputClass =
    "w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none";
  const labelClass =
    "block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1";
  const readonlyClass =
    "w-full px-3 py-2 border border-brand-200 dark:border-brand-600 rounded-md bg-brand-50 dark:bg-brand-700/40 text-brand-500 dark:text-brand-400 cursor-not-allowed";

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
      <div className="relative bg-white dark:bg-brand-800 rounded-lg shadow-xl w-full max-w-5xl mx-4 flex flex-col max-h-[90vh]">
        <div className="flex justify-between items-center p-4 border-b border-brand-200 dark:border-brand-700">
          <h2 className="text-lg font-semibold text-brand-900 dark:text-white">
            {t("characterSelect.title")}
          </h2>
          <button
            onClick={onClose}
            className="p-1 rounded-lg hover:bg-brand-100 dark:hover:bg-brand-700 text-brand-500"
          >
            <div className="i-mdi-close text-xl" />
          </button>
        </div>

        <div className="flex-1 flex overflow-hidden">
          {/* 左侧：搜索栏 + 新增按钮 + 搜索结果 */}
          <div className="w-2/5 border-r border-brand-200 dark:border-brand-700 flex flex-col">
            <div className="p-3 border-b border-brand-200 dark:border-brand-700 space-y-2">
              <div className="relative">
                <div className="i-mdi-magnify absolute left-2 top-1/2 -translate-y-1/2 text-brand-400" />
                <input
                  type="text"
                  value={keyword}
                  onChange={(e) => setKeyword(e.target.value)}
                  placeholder={t("characterSelect.searchPlaceholder")}
                  className="w-full pl-8 pr-3 py-2 text-sm border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
                />
              </div>
              <button
                onClick={handleAddNew}
                className="w-full inline-flex items-center justify-center gap-1 px-3 py-2 text-sm font-medium text-white bg-brand-600 hover:bg-brand-700 dark:bg-brand-500 dark:hover:bg-brand-400 rounded-md transition-colors"
              >
                <div className="i-mdi-plus" />
                <span>{t("characterSelect.addNew")}</span>
              </button>
            </div>

            <div className="flex-1 overflow-y-auto">
              {loading ? (
                <div className="p-4 text-sm text-brand-500 dark:text-brand-400 text-center">
                  {t("common.loading")}
                </div>
              ) : searchResults.length === 0 ? (
                <div className="p-4 text-sm text-brand-500 dark:text-brand-400 text-center">
                  {t("characterSelect.noResults")}
                </div>
              ) : (
                <ul className="divide-y divide-brand-100 dark:divide-brand-700">
                  {searchResults.map((c) => {
                    const active = c.id === selected.id && !isNew;
                    return (
                      <li key={c.id}>
                        <button
                          type="button"
                          onClick={() => handleSelect(c)}
                          className={
                            "w-full flex items-center gap-3 px-3 py-2 text-left hover:bg-brand-50 dark:hover:bg-brand-700/50 transition-colors " +
                            (active
                              ? "bg-brand-100 dark:bg-brand-700/70 border-l-4 border-brand-500"
                              : "border-l-4 border-transparent")
                          }
                        >
                          {c.image_path ? (
                            <img
                              src={c.image_path}
                              alt={c.name}
                              className="w-10 h-12 object-cover rounded flex-shrink-0"
                              onError={(e) => {
                                (e.currentTarget as HTMLImageElement).style.display = "none";
                              }}
                            />
                          ) : (
                            <div className="w-10 h-12 bg-brand-100 dark:bg-brand-700 rounded flex items-center justify-center flex-shrink-0">
                              <div className="i-mdi-account text-brand-400" />
                            </div>
                          )}
                          <div className="flex-1 min-w-0">
                            <div className="text-sm font-medium text-brand-900 dark:text-white truncate">
                              {c.name}
                            </div>
                            {c.other_names && c.other_names !== c.name && (
                              <div className="text-xs text-brand-500 dark:text-brand-400 truncate">
                                {c.other_names}
                              </div>
                            )}
                          </div>
                          {c.id === currentCharactorId && (
                            <span className="text-xs px-1.5 py-0.5 rounded bg-brand-200 dark:bg-brand-600 text-brand-700 dark:text-brand-200 flex-shrink-0">
                              {t("characterSelect.current")}
                            </span>
                          )}
                        </button>
                      </li>
                    );
                  })}
                </ul>
              )}
            </div>

            {!loading && searchResults.length >= SEARCH_LIMIT && (
              <div className="px-3 py-2 text-xs text-brand-400 text-center border-t border-brand-100 dark:border-brand-700">
                {t("characterSelect.resultLimitHint")}
              </div>
            )}
          </div>

          {/* 右侧：编辑栏 */}
          <div className="flex-1 flex flex-col overflow-hidden">
            <div className="px-4 py-3 border-b border-brand-200 dark:border-brand-700">
              <h3 className="text-sm font-semibold text-brand-900 dark:text-white">
                {isNew
                  ? t("characterSelect.creatingNew")
                  : selected.id
                  ? t("characterSelect.editing")
                  : t("characterSelect.selectOrCreateHint")}
              </h3>
            </div>
            <form
              onSubmit={(e) => {
                e.preventDefault();
                handleApply();
              }}
              className="flex-1 overflow-y-auto p-4 space-y-3"
            >
              {!selected.id && !isNew && (
                <div className="p-3 text-sm text-brand-500 dark:text-brand-400 bg-brand-50 dark:bg-brand-700/30 rounded-md text-center">
                  {t("characterSelect.selectOrCreateHint")}
                </div>
              )}

              <div className="grid grid-cols-1 gap-3">
                <div>
                  <label className={labelClass}>{t("characterSelect.name")}</label>
                  <input
                    type="text"
                    value={selected.name || ""}
                    onChange={(e) => update("name", e.target.value)}
                    className={inputClass}
                  />
                </div>
                <div>
                  <label className={labelClass}>{t("characterSelect.workName")}</label>
                  <input
                    type="text"
                    value={workName}
                    onChange={(e) => setWorkName(e.target.value)}
                    placeholder={t("characterSelect.workNamePlaceholder")}
                    className={inputClass + " border-brand-400 dark:border-brand-500"}
                  />
                </div>
              </div>

              <div>
                <label className={labelClass}>{t("characterSelect.otherNames")}</label>
                <input
                  type="text"
                  value={selected.other_names || ""}
                  readOnly
                  placeholder={t("characterSelect.otherNamesAutoManaged")}
                  className={readonlyClass}
                />
                <p className="mt-1 text-xs text-brand-400 dark:text-brand-500">
                  {t("characterSelect.otherNamesAutoManaged")}
                </p>
              </div>

              <div>
                <label className={labelClass}>{t("characterSelect.appearedGames")}</label>
                <div className={readonlyClass}>
                  {selected.game_ids ? (
                    <div className="flex flex-wrap gap-2">
                      {selected.game_ids.split(",").map((gameId, idx) => (
                        <span
                          key={idx}
                          className="inline-flex items-center px-2 py-1 text-xs font-medium rounded-md bg-brand-100 dark:bg-brand-700 text-brand-700 dark:text-brand-300"
                        >
                          {gameId.trim()}
                        </span>
                      ))}
                    </div>
                  ) : (
                    <span className="text-sm text-brand-400 dark:text-brand-500">
                      {t("characterSelect.noAppearedGames")}
                    </span>
                  )}
                </div>
              </div>

              <SingleImageField
                label={t("characterSelect.imagePath")}
                value={selected.image_path || ""}
                onChange={(url) => update("image_path", url)}
                subjectId={selected.id || ""}
                subjectType={1}
                imageType={0}
              />

              <MultiImageField
                label={t("characterSelect.images")}
                value={selected.images || ""}
                onChange={(v) => update("images", v)}
                subjectId={selected.id || ""}
                subjectType={1}
                imageType={1}
              />

              <div>
                <label className={labelClass}>{t("characterSelect.summary")}</label>
                <textarea
                  value={selected.summary || ""}
                  onChange={(e) => update("summary", e.target.value)}
                  rows={4}
                  className={inputClass + " resize-y"}
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className={labelClass}>{t("characterSelect.gender")}</label>
                  <select
                    value={selected.gender ?? 0}
                    onChange={(e) => update("gender", Number(e.target.value))}
                    className={inputClass}
                  >
                    <option value={0}>{t("common.unknown")}</option>
                    <option value={1}>{t("common.male")}</option>
                    <option value={2}>{t("common.female")}</option>
                  </select>
                </div>
                <div>
                  <label className={labelClass}>{t("characterSelect.sort")}</label>
                  <input
                    type="number"
                    value={selected.sort ?? 0}
                    onChange={(e) => update("sort", Number(e.target.value))}
                    className={inputClass}
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className={labelClass}>{t("characterSelect.measurements")}</label>
                  <input
                    type="text"
                    value={selected.measurements || ""}
                    onChange={(e) => update("measurements", e.target.value)}
                    className={inputClass}
                  />
                </div>
                <div>
                  <label className={labelClass}>{t("characterSelect.height")}</label>
                  <input
                    type="text"
                    value={selected.height || ""}
                    onChange={(e) => update("height", e.target.value)}
                    className={inputClass}
                  />
                </div>
              </div>

              <div className="flex justify-end gap-3 pt-4 border-t border-brand-200 dark:border-brand-700 sticky bottom-0 bg-white dark:bg-brand-800">
                <button
                  type="button"
                  onClick={onClose}
                  className="px-4 py-2 text-brand-600 dark:text-brand-400 hover:bg-brand-100 dark:hover:bg-brand-700 rounded-md transition-colors"
                >
                  {t("common.cancel")}
                </button>
                <button
                  type="button"
                  onClick={handleSave}
                  disabled={saving || (!selected.id && !isNew)}
                  className="px-4 py-2 text-brand-700 dark:text-brand-300 border border-brand-300 dark:border-brand-600 hover:bg-brand-50 dark:hover:bg-brand-700 rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {t("common.save")}
                </button>
                <button
                  type="submit"
                  disabled={saving || (!selected.id && !isNew)}
                  className="px-4 py-2 bg-neutral-600 text-white rounded-md hover:bg-neutral-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {saving ? t("common.loading") : t("common.apply")}
                </button>
              </div>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
}

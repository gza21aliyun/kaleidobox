import { useEffect, useRef, useState } from "react";
import { toast } from "react-hot-toast";
import { useTranslation } from "react-i18next";
import { enums, models } from "../../../wailsjs/go/models";
import {
  SearchStaffs,
  CreateStaff,
  UpdateStaff,
  GetStaffById,
} from "../../../wailsjs/go/service/StaffService";
import { SingleImageField } from "../field/SingleImageField";

interface StaffSelectModalProps {
  currentStaffId?: string;
  currentWorkName?: string;
  onSelect: (staff: models.Staff, workName: string) => void;
  onClose: () => void;
}

const SEARCH_LIMIT = 50;

const emptyStaff = (): models.Staff => ({
  id: crypto.randomUUID(),
  name: "",
  other_names: "",
  roles: "",
  source_staff_id: "",
  source_type: "" as enums.SourceType,
  game_ids: "",
  summary: "",
  gender: 0,
  image: "",
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

export function StaffSelectModal({
  currentStaffId,
  currentWorkName,
  onSelect,
  onClose,
}: StaffSelectModalProps) {
  const { t } = useTranslation();
  const [searchResults, setSearchResults] = useState<models.Staff[]>([]);
  const [loading, setLoading] = useState(true);
  const [keyword, setKeyword] = useState("");
  const [selected, setSelected] = useState<models.Staff>(emptyStaff());
  const [workName, setWorkName] = useState(currentWorkName || "");
  const [isNew, setIsNew] = useState(false);
  const [saving, setSaving] = useState(false);

  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const reqIdRef = useRef(0);

  const runSearch = (kw: string) => {
    const reqId = ++reqIdRef.current;
    setLoading(true);
    SearchStaffs(kw, SEARCH_LIMIT)
      .then((res) => {
        if (reqId !== reqIdRef.current) return;
        const list = res || [];
        setSearchResults(list);
        setSelected((prev) => {
          if (prev.id) return prev;
          return list.length > 0 ? { ...list[0] } : emptyStaff();
        });
      })
      .catch((err) => {
        console.error("SearchStaffs failed:", err);
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
      if (currentStaffId) {
        try {
          const cur = await GetStaffById(currentStaffId);
          if (cur && cur.id) {
            setSelected({ ...cur });
            setIsNew(false);
            if (!currentWorkName) setWorkName(cur.name);
          } else {
            setIsNew(true);
          }
        } catch (err) {
          console.error("GetStaffById failed:", err);
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
  }, [currentStaffId]);

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      runSearch(keyword.trim());
    }, 300);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [keyword]);

  const update = <K extends keyof models.Staff>(key: K, value: models.Staff[K]) => {
    setSelected((prev) => ({ ...prev, [key]: value }));
  };

  const handleSelect = (s: models.Staff) => {
    setSelected({ ...s });
    setIsNew(false);
    setWorkName((prev) => prev || s.name);
  };

  const handleAddNew = () => {
    setSelected(emptyStaff());
    setIsNew(true);
    setKeyword("");
  };

  // 仅保存人员记录，不影响 work
  const handleSave = async () => {
    if (!selected.name.trim()) {
      toast.error(t("staffSelect.nameRequired"));
      return;
    }
    setSaving(true);
    try {
      const saved = { ...selected };
      if (isNew || !saved.id) {
        if (!saved.id) saved.id = crypto.randomUUID();
        await CreateStaff(saved);
        toast.success(t("common.createSuccess"));
        setSearchResults((prev) => [saved, ...prev]);
      } else {
        await UpdateStaff(saved);
        toast.success(t("common.success"));
        setSearchResults((prev) =>
          prev.map((it) => (it.id === saved.id ? saved : it))
        );
      }
      setSelected({ ...saved });
      onClose();
    } catch (err) {
      console.error("Save staff failed:", err);
      toast.error(t("common.error"));
    } finally {
      setSaving(false);
    }
  };

  // 保存人员记录 + 将作品名应用到 work
  const handleApply = async () => {
    if (!selected.name.trim()) {
      toast.error(t("staffSelect.nameRequired"));
      return;
    }
    if (!workName.trim()) {
      toast.error(t("staffSelect.workNameRequired"));
      return;
    }
    setSaving(true);
    try {
      const saved = { ...selected };
      saved.other_names = mergeName(saved.other_names, workName.trim());
      if (isNew || !saved.id) {
        if (!saved.id) saved.id = crypto.randomUUID();
        await CreateStaff(saved);
        toast.success(t("common.createSuccess"));
        setSearchResults((prev) => [saved, ...prev]);
      } else {
        await UpdateStaff(saved);
        toast.success(t("common.success"));
        setSearchResults((prev) =>
          prev.map((it) => (it.id === saved.id ? saved : it))
        );
      }
      setSelected({ ...saved });
      onSelect(saved, workName.trim());
      onClose();
    } catch (err) {
      console.error("Apply staff failed:", err);
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
            {t("staffSelect.title")}
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
                  placeholder={t("staffSelect.searchPlaceholder")}
                  className="w-full pl-8 pr-3 py-2 text-sm border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
                />
              </div>
              <button
                onClick={handleAddNew}
                className="w-full inline-flex items-center justify-center gap-1 px-3 py-2 text-sm font-medium text-white bg-brand-600 hover:bg-brand-700 dark:bg-brand-500 dark:hover:bg-brand-400 rounded-md transition-colors"
              >
                <div className="i-mdi-plus" />
                <span>{t("staffSelect.addNew")}</span>
              </button>
            </div>

            <div className="flex-1 overflow-y-auto">
              {loading ? (
                <div className="p-4 text-sm text-brand-500 dark:text-brand-400 text-center">
                  {t("common.loading")}
                </div>
              ) : searchResults.length === 0 ? (
                <div className="p-4 text-sm text-brand-500 dark:text-brand-400 text-center">
                  {t("staffSelect.noResults")}
                </div>
              ) : (
                <ul className="divide-y divide-brand-100 dark:divide-brand-700">
                  {searchResults.map((s) => {
                    const active = s.id === selected.id && !isNew;
                    return (
                      <li key={s.id}>
                        <button
                          type="button"
                          onClick={() => handleSelect(s)}
                          className={
                            "w-full flex items-center gap-3 px-3 py-2 text-left hover:bg-brand-50 dark:hover:bg-brand-700/50 transition-colors " +
                            (active
                              ? "bg-brand-100 dark:bg-brand-700/70 border-l-4 border-brand-500"
                              : "border-l-4 border-transparent")
                          }
                        >
                          {s.image ? (
                            <img
                              src={s.image}
                              alt={s.name}
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
                              {s.name}
                            </div>
                            {s.roles && (
                              <div className="text-xs text-brand-500 dark:text-brand-400 truncate">
                                {s.roles}
                              </div>
                            )}
                          </div>
                          {s.id === currentStaffId && (
                            <span className="text-xs px-1.5 py-0.5 rounded bg-brand-200 dark:bg-brand-600 text-brand-700 dark:text-brand-200 flex-shrink-0">
                              {t("staffSelect.current")}
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
                {t("staffSelect.resultLimitHint")}
              </div>
            )}
          </div>

          {/* 右侧：编辑栏 */}
          <div className="flex-1 flex flex-col overflow-hidden">
            <div className="px-4 py-3 border-b border-brand-200 dark:border-brand-700">
              <h3 className="text-sm font-semibold text-brand-900 dark:text-white">
                {isNew
                  ? t("staffSelect.creatingNew")
                  : selected.id
                  ? t("staffSelect.editing")
                  : t("staffSelect.selectOrCreateHint")}
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
                  {t("staffSelect.selectOrCreateHint")}
                </div>
              )}

              <div className="grid grid-cols-1 gap-3">
                <div>
                  <label className={labelClass}>{t("staffSelect.name")}</label>
                  <input
                    type="text"
                    value={selected.name || ""}
                    onChange={(e) => update("name", e.target.value)}
                    className={inputClass}
                  />
                </div>
                <div>
                  <label className={labelClass}>{t("staffSelect.workName")}</label>
                  <input
                    type="text"
                    value={workName}
                    onChange={(e) => setWorkName(e.target.value)}
                    placeholder={t("staffSelect.workNamePlaceholder")}
                    className={inputClass + " border-brand-400 dark:border-brand-500"}
                  />
                </div>
              </div>

              <div>
                <label className={labelClass}>{t("staffSelect.otherNames")}</label>
                <input
                  type="text"
                  value={selected.other_names || ""}
                  readOnly
                  placeholder={t("staffSelect.otherNamesAutoManaged")}
                  className={readonlyClass}
                />
                <p className="mt-1 text-xs text-brand-400 dark:text-brand-500">
                  {t("staffSelect.otherNamesAutoManaged")}
                </p>
              </div>

              <div>
                <label className={labelClass}>{t("staffSelect.appearedGames")}</label>
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
                      {t("staffSelect.noAppearedGames")}
                    </span>
                  )}
                </div>
              </div>

              <SingleImageField
                label={t("staffSelect.image")}
                value={selected.image || ""}
                onChange={(url) => update("image", url)}
                subjectId={selected.id || ""}
                subjectType={2}
                imageType={0}
              />

              <div>
                <label className={labelClass}>{t("staffSelect.roles")}</label>
                <input
                  type="text"
                  value={selected.roles || ""}
                  onChange={(e) => update("roles", e.target.value)}
                  placeholder={t("staffSelect.rolesPlaceholder")}
                  className={inputClass}
                />
              </div>

              <div>
                <label className={labelClass}>{t("staffSelect.summary")}</label>
                <textarea
                  value={selected.summary || ""}
                  onChange={(e) => update("summary", e.target.value)}
                  rows={4}
                  className={inputClass + " resize-y"}
                />
              </div>

              <div className="grid grid-cols-1 gap-3">
                <div>
                  <label className={labelClass}>{t("staffSelect.gender")}</label>
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

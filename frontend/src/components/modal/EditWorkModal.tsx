import { useState } from "react";
import { toast } from "react-hot-toast";
import { useTranslation } from "react-i18next";
import { models } from "../../../wailsjs/go/models";
import { UpdateWork, DeleteWork } from "../../../wailsjs/go/service/WorkService";
import { CharacterSelectModal } from "./CharacterSelectModal";
import { StaffSelectModal } from "./StaffSelectModal";
import { SingleImageField } from "../field/SingleImageField";
import { MultiImageField } from "../field/MultiImageField";

interface EditWorkModalProps {
  work: models.Work;
  onClose: () => void;
  onSuccess: () => void;
}

export function EditWorkModal({ work, onClose, onSuccess }: EditWorkModalProps) {
  const { t } = useTranslation();
  const [form, setForm] = useState<models.Work>(() => ({ ...work }));
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [charSelectOpen, setCharSelectOpen] = useState(false);
  const [staffSelectOpen, setStaffSelectOpen] = useState(false);

  const update = <K extends keyof models.Work>(key: K, value: models.Work[K]) => {
    setForm(prev => ({ ...prev, [key]: value }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      await UpdateWork(form);
      toast.success(t('common.success'));
      onSuccess();
      onClose();
    } catch (err) {
      console.error("Failed to update work:", err);
      toast.error(t('common.error'));
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleCharactorSelect = (c: models.Charactor, workName: string) => {
    setForm(prev => ({
      ...prev,
      charactor_id: c.id,
      charactor_name: workName,
      charactor_image: c.image_path || prev.charactor_image,
      measurements: c.measurements || prev.measurements,
      height: c.height || prev.height,
      sort: c.sort ?? prev.sort,
    }));
  };

  const handleStaffSelect = (s: models.Staff, workName: string) => {
    setForm(prev => ({
      ...prev,
      staff_id: s.id,
      staff_name: workName,
    }));
  };

  const handleDelete = async () => {
    if (!window.confirm(t('gameIntro.deleteConfirm'))) {
      return;
    }
    setIsSubmitting(true);
    try {
      await DeleteWork(form);
      toast.success(t('common.success'));
      onSuccess();
      onClose();
    } catch (err) {
      console.error("Failed to delete work:", err);
      toast.error(t('common.error'));
    } finally {
      setIsSubmitting(false);
    }
  };

  const inputClass =
    "w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none";
  const labelClass =
    "block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1";
  const selectBtnClass =
    "w-full flex items-center justify-between gap-2 px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white hover:bg-brand-50 dark:hover:bg-brand-600/50 transition-colors text-left";

  return (
    <>
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
        <div className="relative bg-white dark:bg-brand-800 rounded-lg shadow-xl w-full max-w-2xl mx-4 flex flex-col max-h-[90vh]">
          <div className="flex justify-between items-center p-4 border-b border-brand-200 dark:border-brand-700">
            <h2 className="text-lg font-semibold text-brand-900 dark:text-white">
              {t('gameIntro.editCharacter')}
            </h2>
            <button
              onClick={onClose}
              className="p-1 rounded-lg hover:bg-brand-100 dark:hover:bg-brand-700 text-brand-500"
            >
              <div className="i-mdi-close text-xl" />
            </button>
          </div>

          <form onSubmit={handleSubmit} className="flex-1 overflow-y-auto p-6 space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className={labelClass}>{t('gameIntro.characters')}</label>
                <button
                  type="button"
                  onClick={() => setCharSelectOpen(true)}
                  className={selectBtnClass}
                >
                  <span className={form.charactor_name ? "" : "text-brand-400"}>
                    {form.charactor_name || t('characterSelect.selectPlaceholder')}
                  </span>
                  <div className="i-mdi-chevron-right text-brand-400" />
                </button>
              </div>

              <div>
                <label className={labelClass}>{t('gameIntro.cv')}</label>
                <button
                  type="button"
                  onClick={() => setStaffSelectOpen(true)}
                  className={selectBtnClass}
                >
                  <span className={form.staff_name ? "" : "text-brand-400"}>
                    {form.staff_name || t('staffSelect.selectPlaceholder')}
                  </span>
                  <div className="i-mdi-chevron-right text-brand-400" />
                </button>
              </div>

              <div>
                <label className={labelClass}>{t('gameIntro.measurements')}</label>
                <input
                  type="text"
                  value={form.measurements || ''}
                  onChange={e => update('measurements', e.target.value)}
                  className={inputClass}
                />
              </div>

              <div>
                <label className={labelClass}>{t('gameIntro.height')}</label>
                <input
                  type="text"
                  value={form.height || ''}
                  onChange={e => update('height', e.target.value)}
                  className={inputClass}
                />
              </div>

              <div>
                <label className={labelClass}>{t('gameIntro.sort')}</label>
                <input
                  type="number"
                  value={form.sort ?? 0}
                  onChange={e => update('sort', Number(e.target.value))}
                  className={inputClass}
                />
              </div>

              <div className="md:col-span-2">
                <SingleImageField
                  label={t('gameIntro.characterImage')}
                  value={form.charactor_image || ''}
                  onChange={url => update('charactor_image', url)}
                  subjectId={form.id || ''}
                  subjectType={3}
                  imageType={0}
                  gameId={form.game_id || ''}
                />
              </div>

              <div className="md:col-span-2">
                <MultiImageField
                  label={t('gameIntro.workImages')}
                  value={form.images || ''}
                  onChange={v => update('images', v)}
                  subjectId={form.id || ''}
                  subjectType={3}
                  imageType={1}
                  gameId={form.game_id || ''}
                />
              </div>

              <div className="md:col-span-2">
                <label className={labelClass}>{t('gameIntro.characterSummary')}</label>
                <textarea
                  value={form.work_summary || ''}
                  onChange={e => update('work_summary', e.target.value)}
                  rows={5}
                  className={inputClass + " resize-y"}
                />
              </div>
            </div>

            <div className="flex justify-between items-center pt-4 border-t border-brand-200 dark:border-brand-700">
              <button
                type="button"
                onClick={handleDelete}
                disabled={isSubmitting}
                className="px-4 py-2 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-700/30 rounded-md transition-colors disabled:opacity-50"
              >
                {t('common.delete')}
              </button>
              <div className="flex gap-3">
                <button
                  type="button"
                  onClick={onClose}
                  className="px-4 py-2 text-brand-600 dark:text-brand-400 hover:bg-brand-100 dark:hover:bg-brand-700 rounded-md transition-colors"
                >
                  {t('common.cancel')}
                </button>
                <button
                  type="submit"
                  disabled={isSubmitting}
                  className="px-4 py-2 bg-neutral-600 text-white rounded-md hover:bg-neutral-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {isSubmitting ? t('common.loading') : t('common.save')}
                </button>
              </div>
            </div>
          </form>
        </div>
      </div>

      {charSelectOpen && (
        <CharacterSelectModal
          currentCharactorId={form.charactor_id || undefined}
          currentWorkName={form.charactor_name || undefined}
          onSelect={handleCharactorSelect}
          onClose={() => setCharSelectOpen(false)}
        />
      )}

      {staffSelectOpen && (
        <StaffSelectModal
          currentStaffId={form.staff_id || undefined}
          currentWorkName={form.staff_name || undefined}
          onSelect={handleStaffSelect}
          onClose={() => setStaffSelectOpen(false)}
        />
      )}
    </>
  );
}

import type { appconf } from "../../../wailsjs/go/models";
import { SelectFile } from "../../../wailsjs/go/service/GameService";
import { useTranslation } from "react-i18next";

interface VMPanelProps {
  formData: appconf.AppConfig;
  onChange: (newData: appconf.AppConfig) => void;
}

export function VMPanel({ formData, onChange }: VMPanelProps) {
  const { t } = useTranslation();

  const handleVmrunPathChange = async () => {
    try {
      const selection = await SelectFile("vmrun.exe", "*.exe");
      if (selection) {
        onChange({ ...formData, vmrun_path: selection });
      }
    } catch (error) {
      console.error(t("gameEdit.selectFileFailed"), error);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <div className="flex items-center gap-2 mb-1">
          <span className="i-mdi-file-executable text-brand-600 dark:text-brand-400" />
          <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">
            {t("gameEdit.vmrunPathLabel")}
          </label>
        </div>
        <div className="flex gap-2">
          <input
            type="text"
            value={formData.vmrun_path || ""}
            onChange={(e) => onChange({ ...formData, vmrun_path: e.target.value })}
            placeholder={t("gameEdit.vmrunPathPlaceholder")}
            className="glass-input flex-1 px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
          />
          <button
            type="button"
            onClick={handleVmrunPathChange}
            className="glass-btn-neutral px-4 py-2 bg-brand-100 dark:bg-brand-700 text-brand-700 dark:text-brand-300 rounded-md hover:bg-brand-200 dark:hover:bg-brand-600 transition-colors"
          >
            {t("gameEdit.select")}
          </button>
        </div>
      </div>
    </div>
  );
}

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
      const selection = await SelectFile("Vmrun.exe", "*.exe");
      if (selection) {
        onChange({ ...formData, vmrun_path: selection });
      }
    } catch (error) {
      console.error("选择文件失败:", error);
    }
  };

  const handleVmPathChange = async () => {
    try {
      const selection = await SelectFile("虚拟机配置文件", "*.vmx");
      if (selection) {
        onChange({ ...formData, vm_path: selection });
      }
    } catch (error) {
      console.error("选择文件失败:", error);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <div className="flex items-center gap-2 mb-1">
          <span className="i-mdi-file-executable text-brand-600 dark:text-brand-400" />
          <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">
            vmrun.exe 路径
          </label>
        </div>
        <div className="flex gap-2">
          <input
            type="text"
            value={formData.vmrun_path || ""}
            onChange={(e) => onChange({ ...formData, vmrun_path: e.target.value })}
            placeholder="Vmrun.exe 的完整路径"
            className="glass-input flex-1 px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
          />
          <button
            type="button"
            onClick={handleVmrunPathChange}
            className="glass-btn-neutral px-4 py-2 bg-brand-100 dark:bg-brand-700 text-brand-700 dark:text-brand-300 rounded-md hover:bg-brand-200 dark:hover:bg-brand-600 transition-colors"
          >
            选择
          </button>
        </div>
      </div>

      <div>
        <div className="flex items-center gap-2 mb-1">
          <span className="i-mdi-file-code text-brand-600 dark:text-brand-400" />
          <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">
            虚拟机配置文件路径
          </label>
        </div>
        <div className="flex gap-2">
          <input
            type="text"
            value={formData.vm_path || ""}
            onChange={(e) => onChange({ ...formData, vm_path: e.target.value })}
            placeholder="虚拟机 .vmx 文件的完整路径"
            className="glass-input flex-1 px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
          />
          <button
            type="button"
            onClick={handleVmPathChange}
            className="glass-btn-neutral px-4 py-2 bg-brand-100 dark:bg-brand-700 text-brand-700 dark:text-brand-300 rounded-md hover:bg-brand-200 dark:hover:bg-brand-600 transition-colors"
          >
            选择
          </button>
        </div>
      </div>

      <div>
        <div className="flex items-center gap-2 mb-1">
          <span className="i-mdi-monitor text-brand-600 dark:text-brand-400" />
          <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">
            虚拟机名称
          </label>
        </div>
        <input
          type="text"
          value={formData.vm_name || ""}
          onChange={(e) => onChange({ ...formData, vm_name: e.target.value })}
          placeholder="虚拟机的名称"
          className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
        />
      </div>

      <div>
        <div className="flex items-center gap-2 mb-1">
          <span className="i-mdi-account text-brand-600 dark:text-brand-400" />
          <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">
            虚拟机用户名
          </label>
        </div>
        <input
          type="text"
          value={formData.vm_user_name || ""}
          onChange={(e) => onChange({ ...formData, vm_user_name: e.target.value })}
          placeholder="虚拟机登录用户名"
          className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
        />
      </div>

      <div>
        <div className="flex items-center gap-2 mb-1">
          <span className="i-mdi-lock text-brand-600 dark:text-brand-400" />
          <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">
            虚拟机密码
          </label>
        </div>
        <input
          type="password"
          value={formData.vm_pass || ""}
          onChange={(e) => onChange({ ...formData, vm_pass: e.target.value })}
          placeholder="虚拟机登录密码"
          className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
        />
      </div>
    </div>
  );
}

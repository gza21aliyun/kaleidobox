import type { appconf } from "../../../wailsjs/go/models";
import { LanguageSwitcher } from "../ui/LanguageSwitcher";

interface LanguageSettingsPanelProps {
  formData: appconf.AppConfig;
  onChange: (newData: appconf.AppConfig) => void;
}

export function LanguageSettingsPanel({ formData, onChange }: LanguageSettingsPanelProps) {
  return (
    <div className="space-y-6">
      <div className="bg-white dark:bg-gray-800 rounded-lg p-6 shadow-sm">
        <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
          语言设置
        </h3>
        
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                应用语言
              </label>
              <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                选择应用界面显示的语言
              </p>
            </div>
            <LanguageSwitcher />
          </div>
        </div>
      </div>
      
      <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
        <div className="flex">
          <div className="flex-shrink-0">
            <div className="i-mdi-information text-blue-400 dark:text-blue-300 text-lg" />
          </div>
          <div className="ml-3">
            <h3 className="text-sm font-medium text-blue-800 dark:text-blue-200">
              语言切换说明
            </h3>
            <div className="mt-2 text-sm text-blue-700 dark:text-blue-300">
              <ul className="list-disc list-inside space-y-1">
                <li>语言设置会自动保存并在下次启动时生效</li>
                <li>支持中文、英文、日文三种语言</li>
                <li>部分游戏数据可能仍显示原始语言</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
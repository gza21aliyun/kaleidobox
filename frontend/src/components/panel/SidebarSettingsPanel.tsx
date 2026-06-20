import type { appconf } from "../../../wailsjs/go/models";
import { BetterSwitch } from "../ui/BetterSwitch";
import { useTranslation } from 'react-i18next';

interface SidebarSettingsProps {
  formData: appconf.AppConfig;
  onChange: (data: appconf.AppConfig) => void;
}

// 可配置显示/隐藏的侧边栏导航项
const CONFIGURABLE_ITEMS = [
  { key: "task", labelKey: "sidebar.task" },
  { key: "charactor_list", labelKey: "sidebar.characters" },
  { key: "tag_list", labelKey: "sidebar.tags" },
  { key: "category_list", labelKey: "sidebar.categories" },
  { key: "stats", labelKey: "sidebar.stats" },
  { key: "favorites", labelKey: "sidebar.favorites" },
  { key: "virtual_machines", labelKey: "sidebar.virtualMachines" },
  { key: "joystick", labelKey: "sidebar.joystick" },
  { key: "github", labelKey: "sidebar.github" },
];

export function SidebarSettingsPanel({ formData, onChange }: SidebarSettingsProps) {
  const { t } = useTranslation();

  // 解析当前配置
  const visibleItems = formData.sidebar_visible_items
    ? formData.sidebar_visible_items.split(",").filter(Boolean)
    : [];

  // 检查某项是否可见
  const isVisible = (key: string): boolean => {
    // 如果配置为空字符串，则所有项都隐藏
    if (!formData.sidebar_visible_items) {
      return false;
    }
    return visibleItems.includes(key);
  };

  // 切换某项的可见性
  const toggleItem = (key: string, checked: boolean) => {
    let newVisibleItems: string[];

    if (checked) {
      // 添加项
      newVisibleItems = [...visibleItems, key];
    } else {
      // 移除项
      newVisibleItems = visibleItems.filter(item => item !== key);
    }

    // 更新配置
    onChange({
      ...formData,
      sidebar_visible_items: newVisibleItems.join(",")
    } as appconf.AppConfig);
  };

  return (
    <div className="space-y-4">
      <p className="text-sm text-brand-600 dark:text-brand-400 mb-4">
        {t('sidebar.settingsHint')}
      </p>
      {CONFIGURABLE_ITEMS.map(item => (
        <div key={item.key} className="flex items-center justify-between p-2">
          <label className="block text-sm font-medium text-brand-700 dark:text-brand-300">
            {t(item.labelKey)}
          </label>
          <BetterSwitch
            id={`sidebar_${item.key}`}
            checked={isVisible(item.key)}
            onCheckedChange={checked => toggleItem(item.key, checked)}
          />
        </div>
      ))}
    </div>
  );
}
import { Link } from "@tanstack/react-router";
import { useTranslation } from 'react-i18next';
import { BrowserOpenURL } from "../../../wailsjs/runtime/runtime";
import { useAppStore } from "../../store";
import { enums } from "../../../wailsjs/go/models";

interface SideBarProps {
  bgEnabled?: boolean;
  bgOpacity?: number;
}

export function SideBar({ bgEnabled = false, bgOpacity = 0.85 }: SideBarProps) {
  const { t } = useTranslation();
  const { isSidebarOpen, toggleSidebar, tasks } = useAppStore();

  const navItems = [
    { to: "/", label: t('nav.home'), icon: "i-mdi-home" },
    { to: "/library", label: t('nav.library'), icon: "i-mdi-gamepad-variant" },
    { to: "/task", label: t('nav.task'), icon: "i-mdi-clipboard-list" },
    { to: "/charactor_list", label: t('nav.charactors'), icon: "i-mdi-account-group" },
    { to: "/tag_list", label: t('nav.tags'), icon: "i-mdi-tag-multiple" },
    { to: "/brands", label: "品牌", icon: "i-mdi-store" },
    { to: "/series", label: "系列", icon: "i-mdi-format-list-bulleted-type" },
    { to: "/stats", label: t('nav.stats'), icon: "i-mdi-chart-bar" },
    { to: "/categories", label: t('nav.categories'), icon: "i-mdi-format-list-bulleted" },
    { to: "/virtual_machines", label: t('nav.virtualMachines'), icon: "i-mdi-laptop" },
  ];

  const getTaskText = () => {
    var c = 0
    if (tasks && tasks.length > 0) {
      tasks.forEach(task => {
        if (task.status == enums.TaskStatus.STARTED) {
          c += task.total - task.completed
        }
      });
    }
    if (c > 0) {
      return <span className="text-red-500">({c})</span>
    }
    return (<></>)
  };

  // 根据是否启用背景图来决定样式
  const sidebarBgClass = bgEnabled
    ? "border-r border-white/20 dark:border-white/10"
    : "bg-white dark:bg-brand-800 border-r border-brand-200 dark:border-brand-700";

  const sidebarStyle = bgEnabled
    ? { backgroundColor: `rgba(var(--sidebar-bg-rgb), ${bgOpacity})` }
    : undefined;

  return (
    <aside
      className={`flex flex-col transition-all duration-300 ${sidebarBgClass} ${
        isSidebarOpen ? "w-64" : "w-16"
      }`}
      style={sidebarStyle}
    >
      <div className={`flex items-center h-16 ${bgEnabled ? "border-white/20 dark:border-white/10" : "border-brand-200 dark:border-brand-700"} ${isSidebarOpen ? "justify-between px-4" : "justify-center"}`}>
        {isSidebarOpen && (
          <div className="flex items-center gap-1 select-none">
            <img src="/appicon.png" className="w-8 h-8 dark:hidden pointer-events-none" draggable="false" />
            <img src="/appicon-dark.png" className="w-8 h-8 hidden dark:block pointer-events-none" draggable="false" />
            {/* <span className="text-xl font-bold pointer-events-none">LunaBox</span> */}
            <img src="/topbar-title-dark.png" className="h-6 dark:hidden pointer-events-none " />
            <img src="/topbar-title.png" className="h-6 hidden dark:block pointer-events-none " />
          </div>
        )}
        <button
          onClick={toggleSidebar}
          className="p-2 rounded hover:bg-brand-100 dark:hover:bg-brand-700 focus:outline-none select-none data-glass:hover:bg-white/10 data-glass:hover:dark:bg-black/10"
          onDragStart={e => e.preventDefault()}
        >
          <div className="i-mdi-menu text-xl pointer-events-none" />
        </button>
      </div>

      <nav className="flex-1 py-4">
        <ul className="space-y-2 px-2">
          {navItems.map(item => (
            <li key={item.to}>
              <Link
                to={item.to}
                className={`flex items-center p-2 rounded hover:bg-brand-100 dark:hover:bg-brand-700 text-brand-700 dark:text-brand-300 no-underline [&.active]:bg-brand-200 [&.active]:text-brand-900 dark:[&.active]:bg-brand-700 dark:[&.active]:text-brand-100 select-none data-glass:hover:bg-white/10 data-glass:hover:dark:bg-black/10 data-glass:[&.active]:bg-white/20 data-glass:[&.active]:dark:bg-black/20 ${isSidebarOpen ? "" : "justify-center"}`}
                onDragStart={e => e.preventDefault()}
              >
                <div className={`${item.icon} text-xl pointer-events-none`} />
                {isSidebarOpen && <span className="ml-3 pointer-events-none">{item.label}{item.to === "/task" && (<>{getTaskText()}</>)}</span>}
              </Link>
            </li>
          ))}
        </ul>
      </nav>

      <div className={`p-4 ${bgEnabled ? "border-white/20 dark:border-white/10" : "border-brand-200 dark:border-brand-700"} flex ${isSidebarOpen ? "flex-row items-center justify-end gap-1" : "flex-col items-center gap-2"}`}>
        <div
          onClick={() => BrowserOpenURL("https://github.com/gza21aliyun/kaleidobox")}
          className="flex items-center p-2 rounded hover:bg-brand-100 dark:hover:bg-brand-700 text-brand-700 dark:text-brand-300 cursor-pointer select-none data-glass:hover:bg-white/10 data-glass:hover:dark:bg:black/10"
          title="GitHub"
          onDragStart={e => e.preventDefault()}
        >
          <div className="i-mdi-github text-xl pointer-events-none" />
        </div>
        <Link
          to="/joystick"
          className="flex items-center p-2 rounded hover:bg-brand-100 dark:hover:bg-brand-700 text-brand-700 dark:text-brand-300 no-underline [&.active]:bg-brand-200 [&.active]:text-brand-900 dark:[&.active]:bg-brand-700 dark:[&.active]:text-brand-100 select-none data-glass:hover:bg-white/10 data-glass:hover:dark:bg:black/10 data-glass:[&.active]:bg-white/20 data-glass:[&.active]:dark:bg:black/20"
          title="手柄设置"
          onDragStart={e => e.preventDefault()}
        >
          <div className="i-mdi-controller-classic text-xl pointer-events-none" />
        </Link>
        <Link
          to="/settings"
          className="flex items-center p-2 rounded hover:bg-brand-100 dark:hover:bg-brand-700 text-brand-700 dark:text-brand-300 no-underline [&.active]:bg-brand-200 [&.active]:text-brand-900 dark:[&.active]:bg-brand-700 dark:[&.active]:text-brand-100 select-none data-glass:hover:bg-white/10 data-glass:hover:dark:bg:black/10 data-glass:[&.active]:bg-white/20 data-glass:[&.active]:dark:bg:black/20"
          title={t('nav.settings')}
          onDragStart={e => e.preventDefault()}
        >
          <div className="i-mdi-cog text-xl pointer-events-none" />
        </Link>
      </div>
    </aside>
  );
}

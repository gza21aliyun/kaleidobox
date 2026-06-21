// 可配置显示/隐藏的侧边栏导航项
// 添加新页面时只需在这里添加一项

// 主导航区域的配置项
export const CONFIGURABLE_NAV_ITEMS = [
  { to: "/task", key: "task", labelKey: "nav.task", icon: "i-mdi-clipboard-list" },
  { to: "/charactor_list", key: "charactor_list", labelKey: "nav.charactors", icon: "i-mdi-gamepad-variant" },
  { to: "/tag_list", key: "tag_list", labelKey: "nav.tags", icon: "i-mdi-tag-multiple" },
  { to: "/category_list", key: "category_list", labelKey: "nav.categoryList", icon: "i-mdi-folder-multiple-outline" },
  { to: "/stats", key: "stats", labelKey: "nav.stats", icon: "i-mdi-chart-bar" },
  { to: "/monthly_releases", key: "monthly_releases", labelKey: "nav.monthlyReleases", icon: "i-mdi-calendar-month" },
  { to: "/favorites", key: "favorites", labelKey: "nav.favorites", icon: "i-mdi-format-list-bulleted" },
  { to: "/virtual_machines", key: "virtual_machines", labelKey: "nav.virtualMachines", icon: "i-mdi-laptop" },
];

// 底部区域的配置项
export const CONFIGURABLE_BOTTOM_ITEMS = [
  { key: "github", labelKey: "sidebar.github", icon: "i-mdi-github", url: "https://github.com/gza21aliyun/kaleidobox" },
  { key: "joystick", labelKey: "sidebar.joystick", icon: "i-mdi-controller-classic", to: "/joystick" },
];

// 设置面板使用的配置（只包含 key 和 labelKey）
export const CONFIGURABLE_ITEMS = [
  ...CONFIGURABLE_NAV_ITEMS.map(({ key, labelKey }) => ({ key, labelKey })),
  ...CONFIGURABLE_BOTTOM_ITEMS.map(({ key, labelKey }) => ({ key, labelKey })),
];

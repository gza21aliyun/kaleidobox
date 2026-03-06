import { create } from "zustand";

import { appconf, models, vo } from "../wailsjs/go/models";

import { GetAppConfig, UpdateAppConfig } from "../wailsjs/go/service/ConfigService";
import { GetGames, GetGamesByPage } from "../wailsjs/go/service/GameService";
import { GetHomePageData } from "../wailsjs/go/service/HomeService";

type AISummaryCache = {
  [dimension: string]: string;
};

type AppState = {
  isSidebarOpen: boolean;
  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;
  homeData: vo.HomePageData | null;
  config: appconf.AppConfig | null;
  isLoading: boolean;
  fetchHomeData: () => Promise<void>;
  fetchConfig: () => Promise<void>;
  updateConfig: (config: appconf.AppConfig) => Promise<void>;
  // 游戏列表全局状态
  games: models.Game[];
  gamesLoading: boolean;
  fetchGames: () => Promise<void>;
  // AI Summary 缓存
  aiSummaryCache: AISummaryCache;
  setAISummary: (dimension: string, summary: string) => void;
  getAISummary: (dimension: string) => string | undefined;
  setGames: (games: models.Game[]) => void;
  // page: number;
};

export const useAppStore = create<AppState>((set, get) => ({
  isSidebarOpen: true,
  toggleSidebar: () => {
    const newState = !get().isSidebarOpen;
    set({ isSidebarOpen: newState });
    // 保存到配置
    const config = get().config;
    if (config) {
      const newConfig = { ...config, sidebar_open: newState };
      UpdateAppConfig(newConfig).catch(console.error);
    }
  },
  setSidebarOpen: (open: boolean) => set({ isSidebarOpen: open }),
  homeData: null,
  config: null,
  // page: 1,
  isLoading: false,
  games: [],
  gamesLoading: false,
  fetchHomeData: async () => {
    set({ isLoading: true });
    try {
      const data = await GetHomePageData();
      set({ homeData: data });
    }
    catch (error) {
      console.error("Failed to fetch home data:", error);
    }
    finally {
      set({ isLoading: false });
    }
  },
  fetchConfig: async () => {
    try {
      const config = await GetAppConfig();
      set({ config, isSidebarOpen: config.sidebar_open });
    }
    catch (error) {
      console.error("Failed to fetch config:", error);
    }
  },
  updateConfig: async (config: appconf.AppConfig) => {
    try {
      await UpdateAppConfig(config);
      set({ config });
    }
    catch (error) {
      console.error("Failed to update config:", error);
    }
  },
  // 游戏列表管理
  // fetchGames: async () => {
  //   set({ gamesLoading: true });
  //   try {
  //     const result = await GetGames();
  //     set({ games: result || [] });
  //   }
  //   catch (error) {
  //     console.error("Failed to fetch games:", error);
  //   }
  //   finally {
  //     set({ gamesLoading: false });
  //   }
  // },
  fetchGames: async () => {
    set({ gamesLoading: true });
    var page = 1;
    var pageSize = 10;
    try {
      var gameList: models.Game[] = [];
      for (;;) { 
        const result = await GetGamesByPage(page, pageSize);        
        
        if (page == 1) {
          gameList = result || [];
        } else {
          gameList = gameList.concat(result || []);
        }
        set({ games: gameList });
        page++;
        if (result?.length < pageSize) {
          break;
        }
        
      }
      
    }
    catch (error) {
      console.error("Failed to fetch games:", error);
    }
    finally {
      set({ gamesLoading: false });
    }
  },
  // AI Summary 缓存
  aiSummaryCache: {},
  setAISummary: (dimension: string, summary: string) => {
    set(state => ({
      aiSummaryCache: { ...state.aiSummaryCache, [dimension]: summary },
    }));
  },
  getAISummary: () => {
    return undefined; // 这个方法不需要，直接用 selector 访问
  },
  setGames: (gamesToSet: models.Game[]) => {
    set({ games: gamesToSet || [] });
  },
}));

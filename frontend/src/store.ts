import { create } from "zustand";

import { appconf, models, vo, enums } from "../wailsjs/go/models";

import { GetAppConfig, UpdateAppConfig } from "../wailsjs/go/service/ConfigService";
import { GetGames, GetGamesByPage, GetGameByID, GetSimpleGamesByPage, GetGamesSendFront } from "../wailsjs/go/service/GameService";
import { GetHomePageData } from "../wailsjs/go/service/HomeService";
import { ListTags } from "../wailsjs/go/service/TagService";
import { EventsOn } from "../wailsjs/runtime/runtime";
import { arrayFind, arrayToMap } from "./components/utils/Utility";
import { g } from "@unocss/preset-wind3/dist/rules-Dd5IWQsx.mjs";

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
  updateGameInGames: (game: models.Game) => void;
  updateGamesInGames: (games: models.Game[]) => void;
  loadGamesData: () => void;
  // 任务列表全局状态
  tasks: models.TaskNotice[];
  setTasks: (tasks: models.TaskNotice[]) => void;
  // 标签加载状态
  tagsLoaded: Map<string, models.Tag[]>;
  setTagsLoaded: (tags: Map<string, models.Tag[]>) => void;
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
  updateGameInGames: (game: models.Game) => {
    const data = [...get().games];
    const foundGame = arrayFind(data, g => g.id === game.id);
    if (foundGame) {
      const index = data.indexOf(foundGame);
      if (index >= 0) {
        data[index] = game;
        set({ games: data });
      }
    }
    
  },
  updateGamesInGames: (games: models.Game[]) => {
    const data = [...get().games];
    games.forEach(game => {
      const foundGame = arrayFind(data, g => g.id === game.id);
      if (foundGame) {
        const index = data.indexOf(foundGame);
        if (index >= 0) {
          data[index] = game;
        }
      }
    });
    set({ games: data });
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
    var pageSize = 100;
    try {
      var gameList: models.Game[] = [];
      for (;;) { 
        const result = await GetGamesByPage(page, pageSize);        
        
        if (page == 1) {
          gameList = result || [];
          set({ games: gameList });
        } else {
          gameList = gameList.concat(result || []);
        }
        
        page++;
        if (result?.length < pageSize) {
          set({ games: gameList });
          break;
        }
        
      }
      ListTags().then(tags => {
              const map = arrayToMap(tags, tag => tag.category);
              console.log("loadgames tags", map);
              set({ tagsLoaded: map });
      
            });
      
      set({ gamesLoading: false });
      // get().loadGamesData()
      
    }
    catch (error) {
      console.error("Failed to fetch games:", error);
      set({ gamesLoading: false });
    }
    // finally {
    //   set({ gamesLoading: false });
    // }
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
  // 任务列表全局状态
  tasks: [],
  setTasks: (tasksToSet: models.TaskNotice[]) => {
    set({ tasks: tasksToSet || [] });
  },
  // 标签加载状态
  tagsLoaded: new Map(),
  setTagsLoaded: (tags: Map<string, models.Tag[]>) => {
    set({ tagsLoaded: tags });
  },
  loadGamesData: async () => { 
    // for (const game of get().games) {
    //   // const g = await GetGameByID(game.id);
    //   // if (g) {
    //   //   get().updateGameInGames(g);
    //   // }

    //   GetGameByID(game.id).then((g) => {
    //     if (g) {
    //       get().updateGameInGames(g);
    //     }
    //   });
    // }
    GetGamesSendFront(get().games.length)
  },
}));

// 全局事件监听器，确保在任何页面都能接收到游戏更新和任务更新
const unlistenTaskUpdate = EventsOn("game_updates", (data: any) => {
  const task: models.TaskNotice = new models.TaskNotice(data);
  
  // 处理游戏更新
  if (task.item_status === enums.TaskStatus.COMPLETED && task.item_id !== "" && task.type == enums.TaskType.GAMES) {
    const newGame: models.Game = task.item_data as models.Game;
    console.log("newGame:", newGame);
    
    const currentGames = useAppStore.getState().games;
    const newGames = [...currentGames];
    const game = arrayFind(newGames, (it) => it.id === task.item_id);
    
    if (game) {
      const index = newGames.indexOf(game);
      newGames[index] = newGame;
      useAppStore.getState().setGames(newGames);
    } else {
      // 如果游戏不在当前列表中，添加它
      useAppStore.getState().setGames([...currentGames, newGame]);
    }
  } else if (task.type === enums.TaskType.REFRESH_GAMES) { 
    const newGames: models.Game[] = task.item_data as models.Game[];
    useAppStore.getState().updateGamesInGames(newGames)
  }
  
  // 处理任务列表更新
  const currentTasks = useAppStore.getState().tasks;
  const existingTaskIndex = currentTasks.findIndex((t) => t.id === task.id);
  if (existingTaskIndex !== -1) {
    const updatedTasks = [...currentTasks];
    updatedTasks[existingTaskIndex] = task;
    useAppStore.getState().setTasks(updatedTasks);
  } else {
    useAppStore.getState().setTasks([...currentTasks, task]);
  }
  
  // 当视频路径搜索任务完成时，刷新游戏列表以确保所有更新都被包含
  // if (task.status === enums.TaskStatus.COMPLETED && task.item_id == "" && task.type === enums.TaskType.VIDEO_PATHS) {
  //   useAppStore.getState().fetchGames().catch(console.error);
  // }
});

// 在应用退出时取消事件监听
if (typeof window !== 'undefined') {
  window.addEventListener('beforeunload', () => {
    if (unlistenTaskUpdate) {
      unlistenTaskUpdate();
    }
  });
}

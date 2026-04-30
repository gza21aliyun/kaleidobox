import { enums } from "../../wailsjs/go/models";

export const statusOptions = [
  { label: "全部状态", value: "", tKey: "common.filter.allStatus" },
  { label: "未开始", value: enums.GameStatus.NOT_STARTED, tKey: "library.gameStatus.not_started" },
  { label: "游玩中", value: enums.GameStatus.PLAYING, tKey: "library.gameStatus.playing" },
  { label: "已通关", value: enums.GameStatus.COMPLETED, tKey: "library.gameStatus.completed" },
  { label: "搁置", value: enums.GameStatus.ON_HOLD, tKey: "library.gameStatus.on_hold" },
];

export const sortOptions = [
  { label: "名称", value: "name", tKey: "library.sortOptions.name" },
  { label: "添加时间", value: "created_at", tKey: "library.sortOptions.createdAt" },
  { label: "发售日期", value: "release_at", tKey: "library.sortOptions.releaseAt" },
  { label: "开发商", value: "company", tKey: "library.sortOptions.company" },
  { label: "最后游玩", value: "last_played", tKey: "library.sortOptions.lastPlayed" },
  { label: "游戏时间", value: "play_time", tKey: "library.sortOptions.playTime" },
];

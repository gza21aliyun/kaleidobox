import { useEffect, useState } from "react";
import type { vo } from "../../../wailsjs/go/models";
import { enums } from "../../../wailsjs/go/models";
import { GetGameStats, GetGameEndDate } from "../../../wailsjs/go/service/StatsService";
import { formatDurationSimple, formatLastDateText } from "../../utils/time";

interface GameStatsProps {
  game_id: string;
}

export default function GameStats({ game_id }: GameStatsProps) {
  const [stats, setStats] = useState<vo.GameDetailStats | null>(null);

  useEffect(() => {
    const loadStats = async () => {
      try {
        const statsData = await GetGameStats({
          game_id,
          dimension: enums.Period.ALL,
          start_date: "",
          end_date: "",
        });
            
        const end_date = await GetGameEndDate(game_id);
        if (statsData) {
          statsData.end_date = formatLastDateText(end_date) ?? ""
        }      
        setStats(statsData);  
      }
      catch (error) {
        console.error("Failed to load game stats:", error);
      }
    };

    loadStats();
  }, [game_id]);

  if (!stats || stats.total_play_time <= 0) {
    return null;
  }

  return (
    <div className="absolute right-1 bottom-16 z-10 flex flex-col rounded-md bg-black/30 px-2 py-1 text-xs text-white/90 backdrop-blur-sm shadow-lg"> 
      <div className="flex items-center whitespace-nowrap">
        玩过{formatDurationSimple(stats.total_play_time)}
      </div>
      {stats.end_date && (
        <div className="mt-0.5 flex items-center whitespace-nowrap border-t border-white/20 pt-0.5">
          {stats.end_date}玩过
        </div>
      )}
    </div>
  );
}

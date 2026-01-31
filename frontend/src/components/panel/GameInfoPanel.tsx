import type { appconf, models } from "../../../wailsjs/go/models";
import { toast } from "react-hot-toast";
import { useNavigate } from "@tanstack/react-router";
import { BetterSelect } from "../ui/BetterSelect";
import { BetterSwitch } from "../ui/BetterSwitch";

interface GameEditFormProps {
  game: models.Game;
  config?: appconf.AppConfig;
  onTagTaps: (tag: string) => void;
}

export function GameInfoPanel({ 
    game, config, onTagTaps }: GameEditFormProps) { 
        const navigate = useNavigate();
        const handleTagClick = (tag: string) => {
            // 跳转到Library页面并传递标签参数
            // window.location.href = `/library?tags=${encodeURIComponent(tag)}`;
            // 或者如果你使用React Router的navigate功能
                navigate({ to: '/library', search: { tags: tag } });
            };
        return ( 
            <div> 
                
                <div className="mt-4">
                    <div className="font-semibold mb-2 text-brand-900 dark:text-white">类型标签</div>
                    <div className="flex flex-wrap gap-2">
                    {game.meta_tags ? (
                        game.meta_tags.split(',').map((tag, index) => (
                        <button
                            key={index}
                            className="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium bg-[#eeeeee] text-brand-800 dark:bg-[#eeeeee] dark:text-brand-200 hover:bg-[#e0e0e0] dark:hover:bg-[#e0e0e0] transition-colors cursor-pointer"
                            onClick={() => {
                            // 在这里添加点击标签时的处理逻辑
                            console.log(`Clicked tag: ${tag.trim()}`);
                            }}
                        >
                            {tag.trim()}
                        </button>
                        ))
                    ) : (
                        <p className="text-brand-600 dark:text-brand-400 text-sm">-</p>
                    )}
                    </div>
                </div>

                <div className="mt-4">
                    <div className="font-semibold mb-2 text-brand-900 dark:text-white">标签</div>
                    <div className="flex flex-wrap gap-2">
                    {game.tags ? (
                        game.tags.split(',').map((tag, index) => (
                        <button
                            key={index}
                            className="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium bg-[#e0e000] text-brand-800 dark:bg-[#e0e000] dark:text-brand-200 hover:bg-[#d0d000] dark:hover:bg-[#d0d000] transition-colors cursor-pointer"
                            onClick={() => {
                            // 在这里添加点击标签时的处理逻辑
                            handleTagClick(tag.trim());
                            console.log(`Clicked tag: ${tag.trim()}`);
                            }}
                        >
                            {tag.trim()}
                        </button>
                        ))
                    ) : (
                        <p className="text-brand-600 dark:text-brand-400 text-sm">-</p>
                    )}
                    </div>
                </div>
            </div>
        );
}
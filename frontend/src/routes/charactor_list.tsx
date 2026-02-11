import { models } from "../../wailsjs/go/models";
import { createRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { ListCharactors } from "../../wailsjs/go/service/CharactorService";
import { Route as rootRoute } from "./__root";
import { useNavigate } from "@tanstack/react-router";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/charactor_list",
  component: CharactorListPage,
});

function CharactorListPage() {
  const navigate = useNavigate();
  const [charactors, setCharactors] = useState<models.Charactor[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadCharactors();
  }, []);

  const loadCharactors = async () => {
    try {
      setLoading(true);
      setError(null);
      const result = await ListCharactors();
      setCharactors(result || []);
    } catch (err) {
      console.error("Failed to load charactors:", err);
      setError("无法加载角色列表");
    } finally {
      setLoading(false);
    }
  };

  const handleCharactorClick = (charactorId: string) => {
    if (charactorId) {
      navigate({ to: '/charactor/$charactorId', params: { charactorId } });
    }
  };

  const handleStaffClick = (staffId: string) => {
    if (staffId) {
      navigate({ to: '/staff/$staffId', params: { staffId } });
    }
  };

  if (loading) {
    return (
      <div className="p-6">
        <div className="mb-6">
          <h1 className="text-2xl font-bold text-brand-900 dark:text-white">角色列表</h1>
          <p className="text-brand-600 dark:text-brand-400">加载中...</p>
        </div>
        <div className="animate-pulse">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {[...Array(8)].map((_, index) => (
              <div key={index} className="flex items-center gap-3 bg-white dark:bg-brand-800/30 rounded-lg p-3 border border-brand-200 dark:border-brand-700 min-w-[280px]">
                <div className="flex-shrink-0">
                  <div className="w-26 h-40 bg-gray-200 dark:bg-gray-700 rounded-lg"></div>
                </div>
                <div className="flex-1 min-w-0">
                  <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-3/4 mb-2"></div>
                  <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded w-1/2"></div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <div className="mb-6">
          <h1 className="text-2xl font-bold text-brand-900 dark:text-white">角色列表</h1>
        </div>
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-6">
          <div className="flex items-center">
            <div className="i-mdi-alert-circle text-red-500 text-xl mr-3"></div>
            <div>
              <h3 className="font-medium text-red-800 dark:text-red-200">加载失败</h3>
              <p className="text-red-600 dark:text-red-400 mt-1">{error}</p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-brand-900 dark:text-white">角色列表</h1>
        <p className="text-brand-600 dark:text-brand-400">
          共找到 {charactors.length} 个角色
        </p>
      </div>

      {charactors.length === 0 ? (
        <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-6">
          <div className="flex items-center">
            <div className="i-mdi-information text-yellow-500 text-xl mr-3"></div>
            <div>
              <h3 className="font-medium text-yellow-800 dark:text-yellow-200">暂无角色</h3>
              <p className="text-yellow-600 dark:text-yellow-400 mt-1">还没有添加任何角色信息</p>
            </div>
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {charactors.map((charactor) => (
            <div 
              key={charactor.id} 
              className="flex items-center gap-3 bg-white dark:bg-brand-800/30 rounded-lg p-3 border border-brand-200 dark:border-brand-700 min-w-[280px] hover:shadow-md transition-shadow hover:border-brand-300 dark:hover:border-brand-600"
            >
              {/* 角色图片 */}
              {charactor.image_path && (
                <div className="flex-shrink-0">
                  <img 
                    src={charactor.image_path} 
                    alt={charactor.name}
                    className="w-26 h-40 object-cover rounded-lg"
                    style={{ objectFit: 'cover', objectPosition: 'center top' }}
                    onError={(e) => {
                      const target = e.target as HTMLImageElement;
                      target.style.display = 'none';
                    }}
                  />
                </div>
              )}
              
              {/* 角色信息 */}
              <div className="flex-1 min-w-0">
                <h3 className="font-bold text-brand-900 dark:text-white truncate">
                  <button
                    onClick={() => handleCharactorClick(charactor.id)}
                    className="text-sm text-brand-700 dark:text-brand-300 hover:text-brand-900 dark:hover:text-white font-medium underline-offset-2 hover:underline transition-colors text-left"
                  >
                    {charactor.name}
                  </button>
                </h3>
                
                {charactor.other_names && (
                  <div className="mt-1">
                    <span className="text-xs text-brand-600 dark:text-brand-400">别名：</span>
                    <span className="text-xs text-brand-700 dark:text-brand-300">
                      {charactor.other_names}
                    </span>
                  </div>
                )}
                
                
                
                {charactor.game_ids && (
                  <div className="mt-1">
                    <span className="text-xs text-brand-600 dark:text-brand-400">游戏数：</span>
                    <span className="text-xs text-brand-700 dark:text-brand-300">
                      {charactor.game_ids.split(',').length} 个
                    </span>
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
import { appconf, enums, models } from "../../wailsjs/go/models";
import { createRoute, useParams } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { GetCharactorById } from "../../wailsjs/go/service/CharactorService";
import { GetWorkGamesByCharactorId } from "../../wailsjs/go/service/GameService";
import { Route as rootRoute } from "./__root";
import { useNavigate } from "@tanstack/react-router";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/charactor/$charactorId",
  component: CharactorPage,
});

function CharactorPage() {
  const navigate = useNavigate();
  const { charactorId } = Route.useParams();
  const [charactor, setCharactor] = useState<models.Charactor | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [games, setGames] = useState<models.WorkGame[]>([]);

  useEffect(() => {
    console.log("进入charactorPage");
    if (charactorId) {
      loadCharactorDetails();
    }
  }, [charactorId]);

  const loadCharactorDetails = async () => {
    try {
      setLoading(true);
      setError(null);
      const result = await GetCharactorById(charactorId);
      setCharactor(result);
      const loadedGames = await GetWorkGamesByCharactorId(result.id);
      setGames(loadedGames);
    } catch (err) {
      console.error("Failed to load charactor details:", err);
      setError("无法加载角色信息");
    } finally {
      setLoading(false);
    }
  };

  const handleGameClick = (gameId: string) => {
    if (gameId != "") {
      navigate({ to: `/game/${gameId}` });
    }
  };

  const handleStaffClick = (staffId: string) => {
    if (staffId != "") {
      navigate({ to: `/staff/${staffId}` });
    }
  };

  if (loading) {
    return (
      <div className="p-6">
        <button
          onClick={() => window.history.back()}
          className="flex rounded-md items-center text-brand-600 hover:text-brand-900 dark:text-brand-400 dark:hover:text-brand-200 transition-colors"
        >
          <div className="i-mdi-arrow-left text-2xl mr-1" />
          <span>返回</span>
        </button>
        <div className="animate-pulse">
          <div className="h-8 bg-gray-200 rounded w-1/4 mb-6"></div>
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
            <div className="space-y-4">
              <div className="h-8 bg-gray-200 rounded w-1/3"></div>
              <div className="h-4 bg-gray-200 rounded w-1/2"></div>
              <div className="h-4 bg-gray-200 rounded w-2/3"></div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <button
          onClick={() => window.history.back()}
          className="flex rounded-md items-center text-brand-600 hover:text-brand-900 dark:text-brand-400 dark:hover:text-brand-200 transition-colors"
        >
          <div className="i-mdi-arrow-left text-2xl mr-1" />
          <span>返回</span>
        </button>
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

  if (!charactor) {
    return (
      <div className="p-6">
        <button
          onClick={() => window.history.back()}
          className="flex rounded-md items-center text-brand-600 hover:text-brand-900 dark:text-brand-400 dark:hover:text-brand-200 transition-colors"
        >
          <div className="i-mdi-arrow-left text-2xl mr-1" />
          <span>返回</span>
        </button>
        <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-6">
          <div className="flex items-center">
            <div className="i-mdi-information text-yellow-500 text-xl mr-3"></div>
            <div>
              <h3 className="font-medium text-yellow-800 dark:text-yellow-200">未找到角色</h3>
              <p className="text-yellow-600 dark:text-yellow-400 mt-1">指定的角色不存在</p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6">
      <button
        onClick={() => window.history.back()}
        className="flex rounded-md items-center text-brand-600 hover:text-brand-900 dark:text-brand-400 dark:hover:text-brand-200 transition-colors"
      >
        <div className="i-mdi-arrow-left text-2xl mr-1" />
        <span>返回</span>
      </button>
      <br />
      <br />
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-brand-900 dark:text-white mb-2">角色详情</h1>
        <p className="text-brand-600 dark:text-brand-400">查看角色详细信息</p>
      </div>

      <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
        <div className="p-6">
          {/* 头部信息 */}
          <div className="flex items-start justify-between mb-6 pb-6 border-b border-gray-200 dark:border-gray-700">
            <div>
              <h2 className="text-3xl font-bold text-brand-900 dark:text-white">
                {charactor.name}
              </h2>
              {charactor.other_names && (
                <p className="text-brand-600 dark:text-brand-400 mt-2 text-lg">
                  别名: {charactor.other_names}
                </p>
              )}
            </div>
            <div className="flex gap-2">
              <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-brand-100 text-brand-800 dark:bg-brand-900/30 dark:text-brand-200">
                {charactor.source_type}
              </span>
            </div>
          </div>

          

          {/* 基本信息卡片 */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">

            {/* 角色图片 */}
            {charactor.image_path && (
                <div className="bg-gray-50 dark:bg-gray-700/50 rounded-lg p-4 mb-6">
                <img
                    src={charactor.image_path}
                    alt={`${charactor.name} 图片`}
                    className="w-32 h-68 object-cover rounded mx-auto"
                />
                </div>
            )}
            <div className="bg-gray-50 dark:bg-gray-700/50 rounded-lg p-4">
              <h3 className="font-semibold text-brand-900 dark:text-white mb-3">基本信息</h3>
              <div className="space-y-3">
                <div className="flex justify-between">
                  <span className="text-brand-600 dark:text-brand-400">性别:</span>
                  <span className="text-brand-900 dark:text-white font-medium">
                    {charactor.gender === 1 ? '男' : charactor.gender === 2 ? '女' : '未知'}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-brand-600 dark:text-brand-400">来源:</span>
                  <span className="text-brand-900 dark:text-white font-mono text-sm">
                    {charactor.source_type}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-brand-600 dark:text-brand-400">来源ID:</span>
                  <span className="text-brand-900 dark:text-white font-mono text-sm">
                    {charactor.source_charactor_id || '无'}
                  </span>
                </div>
              </div>
            </div>

          </div>

          {/* 简介部分 */}
          {charactor.summary && (
            <div className="mb-6">
              <h3 className="font-semibold text-brand-900 dark:text-white mb-3 text-lg">角色简介</h3>
              <div className="bg-gray-50 dark:bg-gray-700/50 rounded-lg p-4">
                <p className="text-brand-700 dark:text-brand-300 whitespace-pre-wrap leading-relaxed">
                  {charactor.summary}
                </p>
              </div>
            </div>
          )}

          {/* 参与游戏 */}
          {charactor.game_ids && (
            <div>
              <h3 className="font-semibold text-brand-900 dark:text-white mb-3 text-lg">参与项目</h3>
              <div className="bg-gray-50 dark:bg-gray-700/50 rounded-lg p-4">
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-gray-200 dark:border-gray-600">
                        <th className="text-left py-2 px-3 font-medium text-brand-900 dark:text-white">封面</th>
                        <th className="text-left py-2 px-3 font-medium text-brand-900 dark:text-white">游戏名称</th>
                        <th className="text-left py-2 px-3 font-medium text-brand-900 dark:text-white">游戏介绍</th>
                        <th className="text-left py-2 px-3 font-medium text-brand-900 dark:text-white">来源</th>
                        <th className="text-left py-2 px-3 font-medium text-brand-900 dark:text-white">角色图</th>
                        <th className="text-left py-2 px-3 font-medium text-brand-900 dark:text-white">声优</th>
                        <th className="text-left py-2 px-3 font-medium text-brand-900 dark:text-white">角色经历</th>
                      </tr>
                    </thead>
                    <tbody>
                      {games.map((work, index) => (
                        <tr
                          key={index}
                          className="border-b border-gray-100 dark:border-gray-700 hover:bg-gray-100 dark:hover:bg-gray-600/50 cursor-pointer"
                          onClick={() => {
                            console.log(`跳转到游戏: ${work.game.name?.trim()}`);
                            handleGameClick(work.game.id);
                          }}
                        >
                          <td className="py-3 px-3">
                            {work.game.cover_url && (
                              <img
                                src={work.game.cover_url}
                                alt={`${work.game.name?.trim() || '游戏'} 封面`}
                                className="w-16 h-24 object-cover rounded"
                              />
                            )}
                          </td>
                          <td className="py-3 px-3 font-medium text-brand-900 dark:text-white w-1/10">
                            {work.game.name?.trim() || '未知游戏'}
                          </td>
                          
                          <td className="py-3 px-3 text-brand-600 dark:text-brand-400 font-mono text-xs w-2/6">
                            {work.game.summary || '-'}
                          </td>
                          <td className="py-3 px-3">
                            <span className="text-xs px-2 py-1 rounded bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300">
                              {work.work.source_type || '未知'}
                            </span>
                          </td>
                          <td className="py-3 px-3">
                            {work.work.images && (
                              <img
                                src={work.work.images}
                                alt="角色图片"
                                className="w-12 h-24 object-cover rounded"
                              />
                            )}
                          </td>
                          <td className="py-3 px-3 w-1/15">
                            <span 
                                onClick={(e) => {
                                    e.stopPropagation()
                                    handleStaffClick(work.work.staff_id)
                                }}
                                className="inline-flex items-center px-3 py-3 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-200">
                              {work.work.staff_name || '-'}
                            </span>
                          </td>
                          <td className="py-3 px-3 text-brand-600 dark:text-brand-400 font-mono text-xs w-2/6">
                            {work.work.work_summary || '-'}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
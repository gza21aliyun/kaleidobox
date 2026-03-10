import { useTranslation } from 'react-i18next';
import { appconf, enums, models } from "../../wailsjs/go/models";
import { createRoute, useParams } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { GetStaffById } from "../../wailsjs/go/service/StaffService";
import { GetWorkGamesByStaffId } from "../../wailsjs/go/service/GameService";
import { Route as rootRoute } from "./__root";
import { useNavigate } from "@tanstack/react-router";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/staff/$staffId",
  component: StaffPage,
});

function StaffPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { staffId } = Route.useParams();
  const [staff, setStaff] = useState<models.Staff | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [games, setGames] = useState<models.WorkGame[]>([])

  useEffect(() => {
    console.log("进入staffPage")
    if (staffId) {
      loadStaffDetails();
    }
  }, [staffId]);

  const loadStaffDetails = async () => {
    try {
      setLoading(true);
      setError(null);
      const result = await GetStaffById(staffId);
      setStaff(result);
      const loadedGames = await GetWorkGamesByStaffId(result.id)
      setGames(loadedGames)
    } catch (err) {
      console.error("Failed to load staff details:", err);
      setError(t('staff.toasts.loadStaffFailed'));
    } finally {
      setLoading(false);
    }
  };

  const handleGameClick = (gameId: string) => {
                if (gameId != "") {
                    navigate({ to: `/game/${gameId}` });
                }
                // navigate({ to: `/staff/${staffId}` });
            };

  const handleCharactorClick = (charactorId: string) => {
      if (charactorId != "") {
          navigate({ to: `/charactor/${charactorId}` });
      }
      // navigate({ to: `/staff/${staffId}` });
  };

  if (loading) {
    return (
      <div className="p-6">
        <button
        onClick={() => window.history.back()}
        className="flex rounded-md items-center text-brand-600 hover:text-brand-900 dark:text-brand-400 dark:hover:text-brand-200 transition-colors"
      >
        <div className="i-mdi-arrow-left text-2xl mr-1" />
        <span>{t('staff.buttons.back')}</span>
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
        <span>{t('staff.buttons.back')}</span>
      </button>
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-6">
          <div className="flex items-center">
            <div className="i-mdi-alert-circle text-red-500 text-xl mr-3"></div>
            <div>
              <h3 className="font-medium text-red-800 dark:text-red-200">{t('staff.errors.loadFailed')}</h3>
              <p className="text-red-600 dark:text-red-400 mt-1">{error}</p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (!staff) {
    return (
      <div className="p-6">
        <button
        onClick={() => window.history.back()}
        className="flex rounded-md items-center text-brand-600 hover:text-brand-900 dark:text-brand-400 dark:hover:text-brand-200 transition-colors"
      >
        <div className="i-mdi-arrow-left text-2xl mr-1" />
        <span>{t('staff.buttons.back')}</span>
      </button>
        <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-6">
          <div className="flex items-center">
            <div className="i-mdi-information text-yellow-500 text-xl mr-3"></div>
            <div>
              <h3 className="font-medium text-yellow-800 dark:text-yellow-200">{t('staff.errors.notFound')}</h3>
              <p className="text-yellow-600 dark:text-yellow-400 mt-1">{t('staff.errors.notFoundMessage')}</p>
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
        <span>{t('staff.buttons.back')}</span>
      </button>
      <br/>
      <br/>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-brand-900 dark:text-white mb-2">{t('staff.titles.staffDetails')}</h1>
        <p className="text-brand-600 dark:text-brand-400">{t('staff.titles.viewDetails')}</p>
      </div>

      <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
        <div className="p-6">
          {/* 头部信息 */}
          <div className="flex items-start justify-between mb-6 pb-6 border-b border-gray-200 dark:border-gray-700">
            <div>
              <h2 className="text-3xl font-bold text-brand-900 dark:text-white">
                {staff.name}
              </h2>
              {staff.other_names && (
                <p className="text-brand-600 dark:text-brand-400 mt-2 text-lg">
                  {t('staff.labels.alias')}: {staff.other_names}
                </p>
              )}
            </div>
            <div className="flex gap-2">
              <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-brand-100 text-brand-800 dark:bg-brand-900/30 dark:text-brand-200">
                {staff.source_type}
              </span>
            </div>
          </div>

          {/* { staff && staff.gender } */}

          {/* 基本信息卡片 */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">

            {staff?.image && (
                                
                                
                <div className="bg-gray-50 dark:bg-gray-700/50 rounded-lg p-4">   
                    <img 
                                    src={staff.image} 
                                    className="w-16 h-32 object-cover rounded"
                                />
                </div>
            )}

            <div className="bg-gray-50 dark:bg-gray-700/50 rounded-lg p-4">
              <h3 className="font-semibold text-brand-900 dark:text-white mb-3">{t('staff.labels.basicInfo')}</h3>
              <div className="space-y-3">
                <div className="flex justify-between">
                  <span className="text-brand-600 dark:text-brand-400">{t('staff.labels.gender')}:</span>
                  <span className="text-brand-900 dark:text-white font-medium">
                    {staff.gender === 1 ? t('common.male') : staff.gender === 2 ? t('common.female') : t('common.unknown')}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-brand-600 dark:text-brand-400">{t('staff.labels.source')}:</span>
                  <span className="text-brand-900 dark:text-white font-mono text-sm">
                    {staff.source_type}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-brand-600 dark:text-brand-400">{t('staff.labels.sourceId')}:</span>
                  <span className="text-brand-900 dark:text-white font-mono text-sm">
                    {staff.source_staff_id || t('common.none')}
                  </span>
                </div>
              </div>
            </div>


            <div className="bg-gray-50 dark:bg-gray-700/50 rounded-lg p-4">
              <h3 className="font-semibold text-brand-900 dark:text-white mb-3">{t('staff.labels.careerInfo')}</h3>
              <div>
                <span className="text-brand-600 dark:text-brand-400"></span>
                <div className="mt-3">
                  {staff.roles ? (
                    <div className="flex flex-wrap gap-2">
                      {staff.roles.split(',').map((role, index) => (
                        <span 
                          key={index}
                          className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-200"
                        >
                          {role.trim()}
                        </span>
                      ))}
                    </div>
                  ) : (
                    <span className="text-brand-500 dark:text-brand-400 italic"></span>
                  )}
                </div>
              </div>
            </div>
          </div>

          {/* 简介部分 */}
          {staff.summary && (
            <div className="mb-6">
              <h3 className="font-semibold text-brand-900 dark:text-white mb-3 text-lg">{t('staff.labels.summary')}</h3>
              <div className="bg-gray-50 dark:bg-gray-700/50 rounded-lg p-4">
                <p className="text-brand-700 dark:text-brand-300 whitespace-pre-wrap leading-relaxed">
                  {staff.summary}
                </p>
              </div>
            </div>
          )}

          {/* 参与游戏 */}
          {staff.game_ids && (
            <div>
                <h3 className="font-semibold text-brand-900 dark:text-white mb-3 text-lg">{t('staff.labels.participatedWorks')}</h3>
                <div className="bg-gray-50 dark:bg-gray-700/50 rounded-lg p-4">
                    <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                        <thead>
                        <tr className="border-b border-gray-200 dark:border-gray-600">
                            <th className="text-left py-2 px-3 font-medium text-brand-900 dark:text-white">{t('staff.tableHeaders.cover')}</th>
                            <th className="text-left py-2 px-3 font-medium text-brand-900 dark:text-white">{t('staff.tableHeaders.gameName')}</th>
                            <th className="text-left py-2 px-3 font-medium text-brand-900 dark:text-white">{t('staff.tableHeaders.role')}</th>
                            <th className="text-left py-2 px-3 font-medium text-brand-900 dark:text-white">{t('staff.tableHeaders.gameSummary')}</th>
                            <th className="text-left py-2 px-3 font-medium text-brand-900 dark:text-white">{t('staff.tableHeaders.sourceType')}</th>
                            {staff && staff.roles.includes(enums.StaffRole.CV) && 
                            (<th className="text-left py-2 px-3 font-medium text-brand-900 dark:text-white">{t('staff.tableHeaders.characterImage')}</th>)}
                            {staff && staff.roles.includes(enums.StaffRole.CV) && 
                            (<th className="text-left py-2 px-3 font-medium text-brand-900 dark:text-white">{t('staff.tableHeaders.characterName')}</th>)}
                            {staff && staff.roles.includes(enums.StaffRole.CV) && 
                            (<th className="text-left py-2 px-3 font-medium text-brand-900 dark:text-white">{t('staff.tableHeaders.characterSummary')}</th>)}
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
                                    alt={`${work.game.name?.trim() || t('common.game')} ${t('common.cover')}`} 
                                    className="w-16 h-24 object-cover rounded"
                                />
                                )}
                            </td>
                            <td className="py-3 px-3 font-medium text-brand-900 dark:text-white w-1/10">
                                {work.game.name?.trim() || `${t('common.unknown')}${t('common.game')}`}
                            </td>
                            <td className="py-3 px-3 w-1/20">
                                <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-200">
                                {work.work.role || '-'}
                                </span>
                            </td>
                            
                            <td className="py-3 px-3 text-brand-600 dark:text-brand-400 font-mono text-xs w-2/6">
                                {work.game.summary || '-'}
                            </td>
                            <td className="py-3 px-3">
                                <span className="text-xs px-2 py-1 rounded bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300">
                                {work.work.source_type || t('common.unknown')}
                                </span>
                            </td>
                            {staff && staff.roles.includes(enums.StaffRole.CV) && (
                                <td className="py-3 px-3">
                                {work.work.charactor_image && (
                                <img 
                                    src={work.work.charactor_image} 
                                    className="w-12 h-24 object-cover rounded"
                                />
                                )}
                            </td>
                            )}
                            {staff && staff.roles.includes(enums.StaffRole.CV) && (
                                <td className="py-3 px-3 text-brand-700 dark:text-brand-300">
                                  <button
                                    onClick={(e) => {
                                      e.stopPropagation(); // 阻止事件冒泡
                                      handleCharactorClick(work.work.charactor_id);
                                    }}
                                    className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 underline"
                                  >
                                    {work.work.charactor_name || '-'}
                                  </button>
                                    
                                </td>
                            )}
                            {staff && staff.roles.includes(enums.StaffRole.CV) && (
                                <td className="py-3 px-3 text-brand-600 dark:text-brand-400 font-mono text-xs w-2/7">
                                    <div className="whitespace-normal break-words">
                                        {work.work.work_summary || '-'}
                                    </div>
                                </td>
                            )}
                            
                            
                            
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
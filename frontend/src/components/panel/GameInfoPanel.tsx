import { appconf, enums, models } from "../../../wailsjs/go/models";
import { toast } from "react-hot-toast";
import { useNavigate } from "@tanstack/react-router";
import { GetWorksMapByGameId, CountWorks, GetWorksByGameId } from "../../../wailsjs/go/service/WorkService";
import { ListStaffs } from "../../../wailsjs/go/service/StaffService";
import { BetterSelect } from "../ui/BetterSelect";
import { BetterSwitch } from "../ui/BetterSwitch";
import { useEffect, useState } from "react";

interface GameEditFormProps {
  game: models.Game;
  config?: appconf.AppConfig;
  onTagTaps: (tag: string) => void;
}

export function GameInfoPanel({ 
    game, config, onTagTaps }: GameEditFormProps) { 
        const navigate = useNavigate();
        const [worksMap, setWorksMap] = useState<Map<enums.StaffRole, models.Work[]>>(new Map())

        useEffect(() => { 
            console.log("01 GetWorksMapByGameId", game.id)
            // GetWorksMapByGameId(game.id).then((res) => { 
            //     var newWorksMap = new Map(Object.entries(res) as [enums.StaffRole, models.Work[]][])
                
            //     console.log("02 GetWorksMapByGameId", game.id)
            //     console.log("newWorksMap:", res)
            //     setWorksMap(newWorksMap)
            
            // })
            GetWorksByGameId(game.id).then((res) => { 
                console.log("01 GetWorksByGameId", game.id)
                console.log("res:", res)
                var array: models.Work[] = res;
                var m = new Map<enums.StaffRole, models.Work[]>();
                for (let i = 0; i < array.length; i++) {
                    const existingItems = m.get(array[i].role) || [];
                    m.set(array[i].role, [...existingItems, array[i]]);
                }
                setWorksMap(m)
                // setWorks(res)
            
            })
            ListStaffs().then((res) => { 
                console.log("staffs:", res)
            })
            // CountWorks().then((res) => { 
            //     console.log("03 CountWorks", res)
            // })
            return () => { 
                setWorksMap(new Map())
            }
        }, [game])
        const handleTagClick = (tag: string) => {
            // 跳转到Library页面并传递标签参数
            // window.location.href = `/library?tags=${encodeURIComponent(tag)}`;
            // 或者如果你使用React Router的navigate功能
                navigate({ to: '/library', search: { tags: tag } });
            };

        const handleStaffClick = (work: models.Work) => {
            // 跳转到Library页面并传递标签参数
            // window.location.href = `/library?tags=${encodeURIComponent(tag)}`;
            // 或者如果你使用React Router的navigate功能
                console.log(`跳转到员工: `, work);
                if (work.staff_id != "") {
                    navigate({ to: `/staff/${work.staff_id}` });
                }
                // navigate({ to: `/staff/${staffId}` });
            };
        
        return ( 
            <div> 
                {/* 在这里插入worksMap展示内容 */}
                <div className="mt-6 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
                    <h3 className="text-lg font-semibold mb-3 text-brand-900 dark:text-white">工作人员信息</h3>
                    {worksMap && worksMap.size > 0 ? (
                        <div className="space-y-4">
                            {Array.from(worksMap.entries()).map(([role, works]) => (
                                <div key={role} className="border-l-4 border-brand-500 pl-4">
                                    <h4 className="font-medium text-brand-800 dark:text-brand-200 capitalize">
                                        {role.replace(/([A-Z])/g, ' $1').trim()} {/* 将驼峰命名转换为可读格式 */}
                                    </h4>
                                    {works && works.length > 0 ? (
                                        <ul className="mt-2 flex flex-wrap gap-2">
                                            {works.map((work: models.Work, index: number) => (
                                                <li key={index} 
                                                onClick={() => handleStaffClick(work)}
                                                className="text-sm text-brand-600 dark:text-brand-400 bg-brand-50 dark:bg-brand-900/50 px-2 py-1 rounded">
                                                    {work.role === enums.StaffRole.CV 
                                                        ? `${work.charactor_name || '未知角色'} (${work.staff_name || '未知声优'})`
                                                        : work.staff_name || work.charactor_name || `工作人员 ${index + 1}`
                                                    }
                                                </li>
                                            ))}
                                        </ul>
                                    ) : (
                                        <p className="text-sm text-brand-500 dark:text-brand-400 italic mt-1">暂无数据</p>
                                    )}
                                </div>
                            ))}
                        </div>
                    ) : (
                        <p className="text-brand-600 dark:text-brand-400 text-sm">暂无工作人员信息</p>
                    )}
                </div>
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
                {/* <div>{"cover:" + game.cover_url + "\n images:" + game.images}</div> */}

                { game.images.length > 0 && (
                    <div className="flex flex-col gap-2">
                        <div className="font-semibold mb-2 text-brand-900 dark:text-white">类型标签</div>
                        <div className="grid grid-cols-3 gap-2">
                            {game.images.split(",").filter(img => img != game.cover_url && !img.endsWith("pl.jpg")).map((image, index) => (
                                <img
                                    key={index}
                                    src={image}
                                />
                            ))}
                        </div>
                    </div>
                )

                }
            </div>
        );
}
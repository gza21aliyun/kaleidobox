import { enums, models } from "../../../wailsjs/go/models";



export const TAG_CATEGORY = {
    TagCategoryBrand        : "品牌",
	TagCategoryGenre        : "游戏类型",
	TagCategoryPlatform     : "平台",
	TagCategoryPublisher    : "发行",
	TagCategoryPlayerNumber : "玩家人数",
	TagCategoryOther        : "其他",
	TagCategoryCustom       : "自定义",    
	TagCategorySeries       : "系列",
} as const;
export function arrayToMap<T>(ar: T[], keyFn: (key: T) => string): Map<string, T[]>{
    const map = new Map<string, T[]>();
    ar.forEach(item => {
        const key = keyFn(item);
        if (!map.has(key)) {
            map.set(key, []);
        }
        map.get(key)?.push(item);
    });
    return map;
}

export function mapToArray<T>(map: Map<string, T[]>): T[]{
    const ar: T[] = [];
    map.forEach(value => {
        ar.push(...value);
    });
    return ar;
}

export function checkMapItem<T>(map: Map<string, T[]>, fn: (item: T) => boolean): boolean {
    map.forEach(value => {
        for (const item of value) {
            if (fn(item)) {
                return true;
            }
        }
    });
    return false;
}

export function findMapItem<T>(map: Map<string, T[]>, fn: (item: T) => boolean): T | null {
    map.forEach(value => {
        for (const item of value) {
            if (fn(item)) {
                return item;
            }
        }
    });
    return null;
}

export function arrayContains<T>(arr: T[], fn: (item: T) => boolean): boolean {
    for (const item of arr) {
        if (fn(item)) {
            return true;
        }
    }
    return false;
}

export function arrayFind<T>(arr: T[], fn: (item: T) => boolean): T | null {
    for (const item of arr) {
        if (fn(item)) {
            return item;
        }
    }
    return null;
}

export function getMapFromArrayMap(array: models.Tag[], map: Map<string, models.Tag[]>): Map<string, models.Tag[]> { 
    const newMap = new Map<string, models.Tag[]>();
    const newArray = mapToArray(map)

    for (const item of array) { 

        const existingItem = arrayFind(newArray, (it) => it.name === item.name);
        if (existingItem) {
            const key = item.category;
            if (!newMap.has(key)) {
                newMap.set(key, []);
            }
            newMap.get(key)?.push(item);
        } else continue;
        
    }
    return newMap;
}


export function tagMapForEach(map: Map<string, models.Tag[]>, fn: (key: string, 
    tags: models.Tag[]) => JSX.Element) : JSX.Element[] {
        var result: JSX.Element[] = [];            
        const handle = (key: string) => {
            // console.log("tagMapForEach key: " + key + ",has: " + map.has(key))
            if (map.has(key)) {
                result.push(fn(key, map.get(key)!));
            }
        };
        handle(TAG_CATEGORY.TagCategoryBrand);
        handle(TAG_CATEGORY.TagCategoryGenre);
        handle(TAG_CATEGORY.TagCategoryPlatform);
        handle(TAG_CATEGORY.TagCategoryPlayerNumber);
        handle(TAG_CATEGORY.TagCategorySeries);
        handle(TAG_CATEGORY.TagCategoryCustom);
        handle(TAG_CATEGORY.TagCategoryOther);
        Array.from(map.entries()).map(([key, tags]) => { 
            if (key != TAG_CATEGORY.TagCategoryOther && key != TAG_CATEGORY.TagCategoryCustom && 
                key != TAG_CATEGORY.TagCategorySeries && key != TAG_CATEGORY.TagCategoryPlatform && 
                key != TAG_CATEGORY.TagCategoryPlayerNumber && key != TAG_CATEGORY.TagCategoryGenre && key != TAG_CATEGORY.TagCategoryBrand
            ) {
                handle(key);
            }
        });
        return result;
}

export function workMapForEach(map: Map<enums.StaffRole, models.Work[]>, fn: (key: enums.StaffRole, 
    tags: models.Work[]) => JSX.Element) : JSX.Element[] {
        var result: JSX.Element[] = [];            
        const handle = (role: enums.StaffRole) => {
            if (map.has(role)) {
                result.push(fn(role, map.get(role)!));
            }
        };
        handle(enums.StaffRole.DIRECTOR);
        handle(enums.StaffRole.CHARA_DESIGN);
        handle(enums.StaffRole.SCENEARIO);
        handle(enums.StaffRole.ART);
        handle(enums.StaffRole.STAFF);
        handle(enums.StaffRole.CV);
        handle(enums.StaffRole.COMPOSER);
        handle(enums.StaffRole.SINGER);


        return result;
}

export function charactorsForEach(map: Map<enums.StaffRole, models.Work[]>, fn: ( 
    work: models.Work) => JSX.Element) : JSX.Element[] {
        var result: JSX.Element[] = [];        
        var charactors: models.Work[] = []   
        const handle = (role: enums.StaffRole) => {
            if (map.has(role)) {
                map.get(role)!.forEach((work) => {
                    if (work.charactor_name != "")
                        charactors.push(work);
                        // result.push(fn(work));
                })
            }
        };
        handle(enums.StaffRole.CV);
        handle(enums.StaffRole.CHARACTOR);
        charactors = charactors.sort((a, b) => -a.images.localeCompare(b.images));
        charactors.forEach((charactor) => {
            result.push(fn(charactor)); 
        })


        return result;
}

/**
 * 从文件路径中提取文件夹路径
 * @param filePath 完整文件路径
 * @returns 文件夹路径
 */
export function getFolderPath(filePath: string): string {
  // 找到最后一个斜杠或反斜杠的位置
  const lastSlashIndex = Math.max(
    filePath.lastIndexOf("/"),
    filePath.lastIndexOf("\\")
  );

  // 如果没有找到斜杠，返回当前目录
  if (lastSlashIndex === -1) {
    return ".";
  }

  // 截取文件夹路径
  return filePath.substring(0, lastSlashIndex);
}
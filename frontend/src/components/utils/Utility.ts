import { models } from "../../../wailsjs/go/models";


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
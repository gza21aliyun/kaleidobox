import { models, enums } from '../../wailsjs/go/models';
import { GetGamesByTag } from '../../wailsjs/go/service/GameService';
import { GetGamesByCategory } from '../../wailsjs/go/service/CategoryService';
import { GetWorksByStaffIdAndRole } from '../../wailsjs/go/service/WorkService';

export type CategoryType =
  | 'favorite'
  | 'brand'
  | 'series'
  | 'genre'
  | 'chara_design'
  | 'sceneario'
  | 'parent1'
  | 'parent2';

export interface CategoryGameSpec {
  type: CategoryType;
  id: string;
  name: string;
  dirPath?: string;
  original?: unknown;
}

export function getGamesFromIds(
  allGames: models.Game[],
  gameIds: string[],
): models.Game[] {
  const idSet = new Set(gameIds);
  return allGames.filter(g => idSet.has(g.id));
}

export function getGameIdsByDirPath(
  allGames: models.Game[],
  dirPath: string,
): string[] {
  const prefix = dirPath + '/';
  return allGames
    .filter(g => {
      if (!g.path) return false;
      const normalized = g.path.replace(/\\/g, '/');
      return normalized === dirPath || normalized.startsWith(prefix);
    })
    .map(g => g.id);
}

export async function getGameIdsForCategory(
  spec: CategoryGameSpec,
  allGames: models.Game[],
): Promise<string[]> {
  const { type, id, name, dirPath, original } = spec;

  if (type === 'parent1' || type === 'parent2') {
    if (!dirPath) return [];
    return getGameIdsByDirPath(allGames, dirPath);
  }

  if (type === 'favorite') {
    const result = await GetGamesByCategory(id);
    return (result || []).map((g: models.Game) => g.id);
  }

  if (type === 'chara_design' || type === 'sceneario') {
    const staffModel = original as unknown as models.Staff;
    const role = type === 'chara_design' ? enums.StaffRole.CHARA_DESIGN : enums.StaffRole.SCENEARIO;
    const works: models.Work[] = await GetWorksByStaffIdAndRole(staffModel.id, role);
    return works.map((w: models.Work) => w.game_id).filter((gid: string | undefined): gid is string => !!gid);
  }

  const result = await GetGamesByTag(name);
  return (result || []).map((g: models.Game) => g.id);
}

export async function getGamesForCategory(
  spec: CategoryGameSpec,
  allGames: models.Game[],
): Promise<models.Game[]> {
  const { type, id, name, dirPath, original } = spec;

  if (type === 'parent1' || type === 'parent2') {
    if (!dirPath) return [];
    const prefix = dirPath + '/';
    return allGames.filter(g => {
      if (!g.path) return false;
      const normalized = g.path.replace(/\\/g, '/');
      return normalized === dirPath || normalized.startsWith(prefix);
    });
  }

  if (type === 'favorite') {
    const result = await GetGamesByCategory(id);
    return result || [];
  }

  if (type === 'chara_design' || type === 'sceneario') {
    const staffModel = original as unknown as models.Staff;
    const role = type === 'chara_design' ? enums.StaffRole.CHARA_DESIGN : enums.StaffRole.SCENEARIO;
    const works: models.Work[] = await GetWorksByStaffIdAndRole(staffModel.id, role);
    const gameIds = works.map((w: models.Work) => w.game_id).filter((gid: string | undefined): gid is string => !!gid);
    return getGamesFromIds(allGames, gameIds);
  }

  const result = await GetGamesByTag(name);
  return result || [];
}

import { enums } from "../../wailsjs/go/models";

// source_list 使用稳定的 ascii 标识作为 key（格式 "dmm:xxx;getchu:yyy"，同源只允许一个）。
// 大多数来源的枚举 value 本身即 ascii，可直接复用；
// 只有 Eroscape 的 value 是中文"批评空间"，这里统一映射为 "eroscape"（与后端 enums.SourceType.Key() 保持一致）。
export function sourceKey(sourceType: string | enums.SourceType | undefined | null): string {
  if (!sourceType) return "";
  if (sourceType === enums.SourceType.EROSCAPE) return "eroscape";
  return sourceType as string;
}

// 从 source_list 中获取指定来源的 ID，不存在返回 ""
export function getSourceId(sourceList: string | undefined | null, sourceType: string | enums.SourceType): string {
  if (!sourceList) return "";
  const key = sourceKey(sourceType);
  if (!key) return "";
  for (const part of sourceList.split(";")) {
    if (!part) continue;
    const idx = part.indexOf(":");
    if (idx < 0) continue;
    if (part.slice(0, idx) === key) return part.slice(idx + 1);
  }
  return "";
}

// 设置指定来源的 ID（同源只允许一个，覆盖旧值；id 为空则移除该来源），返回新的 source_list
export function setSourceId(
  sourceList: string | undefined | null,
  sourceType: string | enums.SourceType,
  id: string,
): string {
  const key = sourceKey(sourceType);
  if (!key) return sourceList || "";
  const parts = (sourceList || "")
    .split(";")
    .filter((p) => {
      if (!p) return false;
      const idx = p.indexOf(":");
      return idx < 0 || p.slice(0, idx) !== key;
    });
  if (id && id.length > 0) parts.push(`${key}:${id}`);
  return parts.join(";");
}

// 是否包含指定来源
export function hasSource(sourceList: string | undefined | null, sourceType: string | enums.SourceType): boolean {
  return getSourceId(sourceList, sourceType).length > 0;
}

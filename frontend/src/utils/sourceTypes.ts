import { enums } from "../../wailsjs/go/models";
import { sourceKey } from "./sourceList";

export interface SourceOption {
  value: enums.SourceType;
  // i18n 键：sourceType.<ascii key>，与后端 enums.SourceType.Key() / 前端 sourceKey() 保持一致
  labelKey: string;
}

function toOption(value: enums.SourceType): SourceOption {
  return { value, labelKey: `sourceType.${sourceKey(value)}` };
}

// 直接从前端的 enums.SourceType（由 Go 侧 AllSourceTypes 生成）派生，
// 新增来源时只要在 Go 枚举里加一项，这里及所有下拉/列表会自动生效，无需再手写列表。
const ALL: SourceOption[] = (Object.values(enums.SourceType) as enums.SourceType[]).map(toOption);

// 全部来源（含 local 伪来源）
export const ALL_SOURCES: SourceOption[] = ALL;

// 可抓取元数据的来源（排除 local）
export const METADATA_SOURCES: SourceOption[] = ALL.filter(o => o.value !== enums.SourceType.LOCAL);

// 用于「仅匹配该来源」判断的外部来源（在 METADATA_SOURCES 基础上再排除基线 VNDB）
export const EXTERNAL_SOURCES: SourceOption[] = METADATA_SOURCES.filter(o => o.value !== enums.SourceType.VNDB);

// 纯枚举值数组，便于 hasSource / some 等场景使用
export const EXTERNAL_SOURCE_VALUES: enums.SourceType[] = EXTERNAL_SOURCES.map(o => o.value);

// 精确提取「真正需要 i18n 的可见 UI 文案」，排除 JSX 注释/比较/内部 error/测试组件。
// 用法： node scripts/i18n_real_ui.mjs
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
const __dirname = path.dirname(fileURLToPath(import.meta.url));
const { hardcoded } = JSON.parse(fs.readFileSync(path.join(__dirname, 'i18n_audit_result.json'), 'utf8'));

const SKIP_FILE = /I18nTestComponent|Utility\.ts$/;
const cjk = /[\u4e00-\u9fff]/;
const actionable = [];
for (const h of hardcoded) {
  const t = h.text;
  if (SKIP_FILE.test(h.file)) continue;
  // 纯 JSX 注释 {/* ... */}
  if (/^\{?\/\*.*\*\/\}?$/.test(t)) continue;
  // 行尾 // 注释（中文只在注释里）
  const slash = t.indexOf('//');
  if (slash >= 0 && !cjk.test(t.slice(0, slash))) continue;
  // console / throw new Error / 纯比较
  if (/console\.|throw new Error/.test(t)) continue;
  // 时区/来源等 label 数据表：保留但标记（此处按数据源 label 仍视为可 i18n，但排除纯枚举 value 行）
  actionable.push(h);
}

// 按文件分组
const byFile = {};
for (const h of actionable) (byFile[h.file] ||= []).push(h);
const files = Object.keys(byFile).sort((a, b) => byFile[b].length - byFile[a].length);
let out = `真正可见 UI 硬编码行数: ${actionable.length}，涉及 ${files.length} 个文件\n\n`;
for (const f of files) {
  out += `## ${f}  (${byFile[f].length})\n`;
  for (const h of byFile[f]) out += `  ${h.line}: ${h.text}\n`;
  out += '\n';
}
fs.writeFileSync(path.join(__dirname, '_real_ui.txt'), out, 'utf8');
console.log(`actionable=${actionable.length} files=${files.length}`);

// i18n 审计脚本：检测三类问题
// 1) 前端引用了但三语翻译文件里都缺的键；
// 2) 前端引用了但某些语言缺的键 / 语言之间互缺的键；
// 3) 疑似硬编码中文（未走 t()）的文案（启发式，需人工确认）。
// 用法： node scripts/i18n_audit.mjs
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const SRC = path.resolve(__dirname, '..', 'src');
const LOCALE_DIR = path.join(SRC, 'i18n', 'locales');
const LANGS = ['zh-CN', 'en-US', 'ja-JP'];

function flatten(obj, prefix = '', out = {}) {
  for (const k of Object.keys(obj)) {
    const v = obj[k];
    const np = prefix ? `${prefix}.${k}` : k;
    if (v && typeof v === 'object' && !Array.isArray(v)) flatten(v, np, out);
    else out[np] = v;
  }
  return out;
}

const locales = {};
const flat = {};
for (const lang of LANGS) {
  const raw = JSON.parse(fs.readFileSync(path.join(LOCALE_DIR, `${lang}.json`), 'utf8'));
  locales[lang] = raw;
  flat[lang] = flatten(raw);
}
const keySets = Object.fromEntries(LANGS.map(l => [l, new Set(Object.keys(flat[l]))]));
const union = new Set([...LANGS.flatMap(l => Object.keys(flat[l]))]);
const intersection = [...union].filter(k => LANGS.every(l => keySets[l].has(k)));

// ---- 扫描源码，提取 t() 引用的键 ----
function* walk(dir) {
  for (const e of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, e.name);
    if (e.isDirectory()) {
      if (p.includes(`${path.sep}i18n${path.sep}locales`)) continue;
      yield* walk(p);
    } else if (/\.(ts|tsx)$/.test(e.name)) {
      yield p;
    }
  }
}

const used = new Set();
const dynamicRefs = [];
const fileOfKey = {};
const tRe = /(?:^|[^\w$.])t\(\s*(['"`])([^'"`\n]*)\1/g;
const i18nRe = /i18next\.t\(\s*(['"`])([^'"`\n]*)\1/g;
const transRe = /i18nKey\s*=\s*(['"])([^'"]+)\1/g;

for (const file of walk(SRC)) {
  const code = fs.readFileSync(file, 'utf8');
  for (const re of [tRe, i18nRe, transRe]) {
    re.lastIndex = 0;
    let m;
    while ((m = re.exec(code))) {
      const key = m[2].trim();
      if (!key) continue;
      if (key.includes('${')) { dynamicRefs.push({ file, key }); continue; }
      used.add(key);
      (fileOfKey[key] ||= new Set()).add(path.relative(SRC, file));
    }
  }
}

// ---- 问题 1：引用了但三语全缺 ----
const missingEverywhere = [...used].filter(k => !union.has(k)).sort();

// ---- 问题 2a：引用了但某些语言缺 ----
const usedMissingByLang = {};
for (const lang of LANGS) {
  usedMissingByLang[lang] = [...used].filter(k => union.has(k) && !keySets[lang].has(k)).sort();
}

// ---- 问题 2b：语言之间互缺（与是否使用无关）----
const crossLang = {};
for (const lang of LANGS) {
  const others = LANGS.filter(l => l !== lang);
  crossLang[lang] = [...keySets[lang]].filter(k => others.some(o => !keySets[o].has(k))).sort();
}

// ---- 问题：翻译文件里有但从未被引用（死键）----
const unused = [...union].filter(k => !used.has(k) && !k.includes('.' ) === false ? !used.has(k) : !used.has(k));
const deadKeys = [...union].filter(k => !used.has(k)).sort();

// ---- 问题 3：硬编码中文（启发式）----
const cjk = /[\u4e00-\u9fff]/;
const hardcoded = [];
for (const file of walk(SRC)) {
  const lines = fs.readFileSync(file, 'utf8').split('\n');
  lines.forEach((line, i) => {
    if (!cjk.test(line)) return;
    const trimmed = line.trim();
    if (/^(\/\/|\/\*|\*)/.test(trimmed)) return;              // 注释
    if (/console\.(log|error|warn|info)/.test(line)) return;  // 日志
    if (/\bt\(\s*['"`][^'"`]*['"`]\s*[,)]/.test(line) === false && cjk.test(line)) {
      // 该行含中文且没有明显的 t('...') 包裹
      const hasT = /(?:^|[^\w$.])t\(\s*['"`]/.test(line) || /i18next\.t\(/.test(line) || /i18nKey=/.test(line);
      if (!hasT) {
        hardcoded.push({ file: path.relative(SRC, file), line: i + 1, text: trimmed.slice(0, 120) });
      }
    }
  });
}

// ---- 输出 ----
function list(title, arr) {
  console.log(`\n### ${title} (${arr.length})`);
  arr.slice(0, 500).forEach(x => console.log(typeof x === 'string' ? x : JSON.stringify(x)));
  if (arr.length > 500) console.log(`... 其余 ${arr.length - 500} 条省略`);
}

console.log('===== i18n 审计汇总 =====');
console.log(`三语共同键: ${intersection.length}, 全部键(并集): ${union.size}, 前端引用静态键: ${used.size}`);
console.log(`问题1 引用但三语全缺: ${missingEverywhere.length}`);
console.log(`问题2a 引用但按语言缺: ${LANGS.map(l => `${l}=${usedMissingByLang[l].length}`).join(', ')}`);
console.log(`问题2b 语言互缺: ${LANGS.map(l => `${l}=${crossLang[l].length}`).join(', ')}`);
console.log(`死键(有翻译但未被引用): ${deadKeys.length}`);
console.log(`问题3 疑似硬编码中文行: ${hardcoded.length}`);

list('问题1: 引用了但三语全缺', missingEverywhere.map(k => `${k}  <= ${[...(fileOfKey[k] || [])].join(',')}`));
for (const lang of LANGS) list(`问题2a: ${lang} 引用但缺失`, usedMissingByLang[lang]);
for (const lang of LANGS) list(`问题2b: ${lang} 相对其它语言多出/缺失`, crossLang[lang]);
list('死键(翻译有但代码未引用) 前500', deadKeys);
list('问题3: 疑似硬编码中文(需人工确认)', hardcoded);

fs.writeFileSync(path.join(__dirname, 'i18n_audit_result.json'), JSON.stringify({
  summary: { intersection: intersection.length, union: union.size, used: used.size },
  missingEverywhere, usedMissingByLang, crossLang, deadKeys, hardcoded, dynamicRefs,
}, null, 2), 'utf8');
console.log('\n完整结果已写入 scripts/i18n_audit_result.json');

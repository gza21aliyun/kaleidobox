// i18n 修复 patch2：把源文件转换中新增的翻译键补进三种语言文件（仅补缺失，不覆盖已有值）。
// 用法： node scripts/i18n_apply_fix2.mjs
// 安全网：脚本会读取 scripts/_keys.txt，确保其中每个键都在 PATCHES 中定义，否则报错退出。
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const LOCALE_DIR = path.join(__dirname, '..', 'src', 'i18n', 'locales');
const LANGS = ['zh-CN', 'en-US', 'ja-JP'];

// key -> { zh, en, ja }
const PATCHES = {
  // ===== update (UpdateDialog) =====
  'update.availableUpdate': { zh: '检测到 {{name}} 有可用的更新', en: 'An update is available for {{name}}', ja: '{{name}} のアップデートが利用可能です' },
  'update.currentVersion': { zh: '当前版本', en: 'Current version', ja: '現在のバージョン' },
  'update.releaseDate': { zh: '发布日期', en: 'Release date', ja: 'リリース日' },
  'update.changelog': { zh: '更新日志', en: 'Changelog', ja: '変更履歴' },
  'update.downloadGithub': { zh: '从 GitHub 下载', en: 'Download from GitHub', ja: 'GitHub からダウンロード' },
  'update.downloadGitee': { zh: '从 Gitee 下载', en: 'Download from Gitee', ja: 'Gitee からダウンロード' },
  'update.skipVersion': { zh: '跳过此版本', en: 'Skip this version', ja: 'このバージョンをスキップ' },

  // ===== updateSettings =====
  'updateSettings.checkFailed': { zh: '检查更新失败', en: 'Failed to check for updates', ja: '更新の確認に失敗しました' },
  'updateSettings.modifyFailed': { zh: '修改失败：{{err}}', en: 'Modification failed: {{err}}', ja: '変更に失敗しました：{{err}}' },

  // ===== video =====
  'video.notExist': { zh: '视频不存在', en: 'Video does not exist', ja: '動画が存在しません' },
  'video.streamFailed': { zh: '获取视频流失败', en: 'Failed to load video stream', ja: '動画ストリームの取得に失敗しました' },
  'video.collapseList': { zh: '收起列表', en: 'Collapse list', ja: 'リストを折りたたむ' },
  'video.expandList': { zh: '展开列表', en: 'Expand list', ja: 'リストを展開' },
  'video.list': { zh: '视频列表', en: 'Video list', ja: '動画リスト' },
  'video.restore': { zh: '还原', en: 'Restore', ja: '元に戻す' },
  'video.maximize': { zh: '最大化', en: 'Maximize', ja: '最大化' },
  'video.loading': { zh: '加载中...', en: 'Loading...', ja: '読み込み中...' },
  'video.prev': { zh: '上一个', en: 'Previous', ja: '前へ' },
  'video.next': { zh: '下一个', en: 'Next', ja: '次へ' },
  'video.noVideo': { zh: '暂无视频', en: 'No videos', ja: '動画がありません' },

  // ===== brandGames / seriesGames =====
  'brandGames.loadFailed': { zh: '加载品牌游戏失败', en: 'Failed to load brand games', ja: 'ブランドゲームの読み込みに失敗しました' },
  'brandGames.notFound': { zh: '未找到品牌', en: 'Brand not found', ja: 'ブランドが見つかりません' },
  'brandGames.emptyBrand': { zh: '该品牌下暂无游戏', en: 'No games for this brand', ja: 'このブランドにはゲームがありません' },
  'seriesGames.loadFailed': { zh: '加载系列游戏失败', en: 'Failed to load series games', ja: 'シリーズゲームの読み込みに失敗しました' },
  'seriesGames.notFound': { zh: '未找到系列', en: 'Series not found', ja: 'シリーズが見つかりません' },
  'seriesGames.emptySeries': { zh: '该系列下暂无游戏', en: 'No games for this series', ja: 'このシリーズにはゲームがありません' },

  // ===== search =====
  'search.game': { zh: '搜索游戏', en: 'Search games', ja: 'ゲームを検索' },
  'search.placeholder': { zh: '搜索...', en: 'Search...', ja: '検索...' },

  // ===== charactor =====
  'charactor.images': { zh: '角色图片', en: 'Character images', ja: 'キャラクター画像' },
  'charactor.height': { zh: '身高', en: 'Height', ja: '身長' },
  'charactor.measurements': { zh: '三围', en: 'Measurements', ja: 'スリーサイズ' },

  // ===== game =====
  'game.searchVideo': { zh: '搜索视频', en: 'Search video', ja: '動画を検索' },
  'game.playedFor': { zh: '玩了 {{duration}}', en: 'Played for {{duration}}', ja: '{{duration}} プレイ' },
  'game.playedOn': { zh: '上次游玩：{{date}}', en: 'Last played: {{date}}', ja: '最終プレイ：{{date}}' },

  // ===== common =====
  'common.character': { zh: '角色', en: 'Character', ja: 'キャラクター' },
  'common.search': { zh: '搜索', en: 'Search', ja: '検索' },

  // ===== settings / sourceType / monthly / filter / category =====
  'settings.vmSettings': { zh: '虚拟机设置', en: 'Virtual Machine Settings', ja: '仮想マシン設定' },
  'sourceType.vm': { zh: '虚拟机', en: 'Virtual Machine', ja: '仮想マシン' },
  'sourceType.duplicated': { zh: '重复', en: 'Duplicated', ja: '重複' },
  'monthly.searchDownloadFolder': { zh: '搜索下载文件夹', en: 'Search download folder', ja: 'ダウンロードフォルダを検索' },
  'monthly.imported': { zh: '已导入', en: 'Imported', ja: 'インポート済み' },
  'gameIntro.sameBrandGames': { zh: '同品牌游戏', en: 'Games from the same brand', ja: '同ブランドのゲーム' },
  'filter.toggleIntersectionMode': { zh: '切换交集模式或合集模式', en: 'Toggle intersection or union mode', ja: '積集合/和集合モードを切り替え' },
  'category.clickTimes': { zh: '(点击{{count}}次)', en: '({{count}} clicks)', ja: '（{{count}}回クリック）' },

  // ===== imageCard / localSearch / tagGroup / template / gameGallery =====
  'imageCard.setAsCover': { zh: '设为封面', en: 'Set as cover', ja: '表紙に設定' },
  'imageCard.fullscreenMagpie': { zh: '全屏显示，注意该功能会修改Magpie裁剪设置', en: 'Fullscreen (note: this changes Magpie crop settings)', ja: '全画面表示（この機能はMagpieのトリミング設定を変更します）' },
  'localSearch.launchGame': { zh: '启动游戏', en: 'Launch game', ja: 'ゲームを起動' },
  'localSearch.openGame': { zh: '打开游戏', en: 'Open game', ja: 'ゲームを開く' },
  'tagGroup.adultContent': { zh: '成人内容', en: 'Adult content', ja: '成人向けコンテンツ' },
  'tagGroup.spoilerWarning': { zh: '剧透预警', en: 'Spoiler warning', ja: 'ネタバレ注意' },
  'template.preview': { zh: '模板预览', en: 'Template preview', ja: 'テンプレートのプレビュー' },
  'gameGallery.waitProcessAssign': { zh: '等待处理分配', en: 'Waiting to assign', ja: '割り当て待ち' },

  // ===== updateSettingsPanel already above; basic =====
  'basic.hideOldMatchingBtns': { zh: '隐藏旧版匹配按钮', en: 'Hide legacy matching buttons', ja: '旧バージョンのマッチングボタンを隠す' },
  'basic.deleteRequiresConfirm': { zh: '删除操作需要确认', en: 'Deletion requires confirmation', ja: '削除操作には確認が必要' },

  // ===== screenshotHotkey =====
  'screenshotHotkey.desc': { zh: '截图快捷键', en: 'Screenshot hotkey', ja: 'スクリーンショットショートカット' },

  // ===== gameEdit =====
  'gameEdit.saveDirFoundSuccess': { zh: '已找到存档目录', en: 'Save directory found', ja: 'セーブディレクトリを検出しました' },
  'gameEdit.saveDirSearchFailed': { zh: '搜索存档目录失败', en: 'Failed to search save directory', ja: 'セーブディレクトリの検索に失敗しました' },
  'gameEdit.videoFile': { zh: '视频文件', en: 'Video file', ja: '動画ファイル' },

  // ===== batchImport =====
  'batchImport.scanDirFailed': { zh: '扫描目录失败', en: 'Failed to scan directory', ja: 'ディレクトリのスキャンに失敗しました' },
  'batchImport.selectDirFailed': { zh: '选择目录失败', en: 'Failed to select directory', ja: 'ディレクトリの選択に失敗しました' },
  'batchImport.importSuccessCount': { zh: '成功导入 {{count}} 个游戏', en: 'Successfully imported {{count}} game(s)', ja: '{{count}} 本のゲームをインポートしました' },
  'batchImport.fetchFailed': { zh: '获取失败', en: 'Fetch failed', ja: '取得に失敗しました' },
  'batchImport.noGameFolderDetected': { zh: '未检测到游戏文件夹', en: 'No game folder detected', ja: 'ゲームフォルダーが検出されませんでした' },
  'batchImport.rateLimitHint': { zh: '为避免触发限流，匹配速度可能较慢，请耐心等待', en: 'To avoid rate limiting, matching may be slow. Please wait.', ja: 'レート制限を避けるため、マッチングが遅くなる場合があります。お待ちください' },

  // ===== rootDrop =====
  'rootDrop.dropGameFolderTitle': { zh: '拖放游戏文件夹', en: 'Drop game folder', ja: 'ゲームフォルダーをドロップ' },
  'rootDrop.dropGameFolderHint': { zh: '松开以导入该游戏文件夹', en: 'Release to import this game folder', ja: 'このゲームフォルダーをインポートするには離してください' },
  'rootDrop.dropLnkTitle': { zh: '拖放快捷方式', en: 'Drop shortcut', ja: 'ショートカットをドロップ' },
  'rootDrop.dropLnkHint': { zh: '松开以添加该快捷方式指向的游戏', en: 'Release to add the game targeted by this shortcut', ja: 'このショートカットが指すゲームを追加するには離してください' },

  // ===== downloadedFiles =====
  'downloadedFiles.saveNotFound': { zh: '未找到存档', en: 'Save not found', ja: 'セーブデータが見つかりません' },
  'downloadedFiles.execAnimHint': { zh: '执行存档动画处理', en: 'Run save animation processing', ja: 'セーブデータのアニメーション処理を実行' },
  'downloadedFiles.execAnim': { zh: '执行动画', en: 'Run animation', ja: 'アニメーション実行' },
  'downloadedFiles.saveOverwritten': { zh: '存档已覆盖', en: 'Save overwritten', ja: 'セーブデータを上書きしました' },
  'downloadedFiles.saveDownloaded': { zh: '存档已下载', en: 'Save downloaded', ja: 'セーブデータをダウンロードしました' },

  // ===== batchUpdate =====
  'batchUpdate.updateFailed': { zh: '更新失败', en: 'Update failed', ja: '更新に失敗しました' },
  'batchUpdate.fetchFailed': { zh: '获取失败', en: 'Fetch failed', ja: '取得に失敗しました' },
  'batchUpdate.scrapeSource': { zh: '搜刮源', en: 'Scrape source', ja: 'スクレイプソース' },
  'batchUpdate.overwriteHint': { zh: '是否覆盖', en: 'Whether to overwrite', ja: '上書きするかどうか' },
  'batchUpdate.loadTagsHint': { zh: '是否加载标签', en: 'Whether to load tags', ja: 'タグを読み込むかどうか' },
  'batchUpdate.loadStaffHint': { zh: '是否加载制作人员', en: 'Whether to load staff', ja: 'スタッフを読み込むかどうか' },
  'batchUpdate.loadCharsHint': { zh: '是否加载角色', en: 'Whether to load characters', ja: 'キャラクターを読み込むかどうか' },
  'batchUpdate.loadGalleryHint': { zh: '是否加载画廊', en: 'Whether to load gallery', ja: 'ギャラリーを読み込むかどうか' },
  'batchUpdate.rematchHint': { zh: '否的话只会根据id重新获取数据，是的话会根据名字重新搜索', en: 'If off, only refetch by id; if on, re-search by name', ja: 'オフの場合はIDのみで再取得、オンの場合は名前でも再検索' },
  'batchUpdate.unionSearchHint': { zh: '会自动根据次要源补充数据，需要该源的ID，暂时只有批评空间和bangumi能同时获取其他源ID', en: 'Automatically supplements data from secondary sources (requires that source id). Currently only Eroscape and Bangumi can fetch other source ids together.', ja: '副次ソースから自動的にデータを補完します（そのソースのIDが必要）。現時点では批評空間とBangumiのみ他ソースのIDを同時に取得できます' },
  'batchUpdate.selectMatchedHint': { zh: '选取已匹配指定源的游戏', en: 'Select games matched to the specified source', ja: '指定ソースにマッチしたゲームを選択' },
  'batchUpdate.selectUpdatedHint': { zh: '选取已更新', en: 'Select updated', ja: '更新済みを選択' },
  'batchUpdate.selectUnmatchedHint': { zh: '选取未匹配指定源的游戏', en: 'Select games not matched to the specified source', ja: '指定ソースに未マッチのゲームを選択' },
  'batchUpdate.selectEmptyHint': { zh: '选取空匹配', en: 'Select empty matches', ja: '空マッチを選択' },
  'batchUpdate.selectOnlyMatchedHint': { zh: '选取仅匹配指定源的游戏', en: 'Select games matched only to the specified source', ja: '指定ソースのみにマッチしたゲームを選択' },
  'batchUpdate.onlyMatched': { zh: '仅匹配', en: 'Only matched', ja: 'マッチのみ' },
  'batchUpdate.openDetail': { zh: '打开游戏详情', en: 'Open game details', ja: 'ゲーム詳細を開く' },

  // ===== stats.period =====
  'stats.period.week': { zh: '本周', en: 'This week', ja: '今週' },
  'stats.period.month': { zh: '本月', en: 'This month', ja: '今月' },
  'stats.period.all': { zh: '全部', en: 'All', ja: 'すべて' },

  // ===== keyMapping =====
  'keyMapping.disabled': { zh: '关闭手柄映射', en: 'Disable gamepad mapping', ja: 'コントローラー割り当てを無効化' },
  'keyMapping.mappingDeleted': { zh: '映射已删除', en: 'Mapping deleted', ja: '割り当てを削除しました' },
  'keyMapping.deleteMappingFailed': { zh: '删除映射失败', en: 'Failed to delete mapping', ja: '割り当ての削除に失敗しました' },
  'keyMapping.clearedAll': { zh: '已清除所有按键映射，请点击保存按钮确认更改', en: 'All key mappings cleared, click Save to confirm', ja: 'すべての割り当てをクリアしました。保存ボタンで確定してください' },
  'keyMapping.allMappingsSaved': { zh: '所有映射已保存', en: 'All mappings saved', ja: 'すべての割り当てを保存しました' },
  'keyMapping.saveMappingsFailed': { zh: '保存映射失败', en: 'Failed to save mappings', ja: '割り当ての保存に失敗しました' },
  'keyMapping.clear': { zh: '清除', en: 'Clear', ja: 'クリア' },
  'keyMapping.screenshot': { zh: '截图', en: 'Screenshot', ja: 'スクリーンショット' },
  'keyMapping.notSet': { zh: '未设置', en: 'Not set', ja: '未設定' },
  'keyMapping.deleteMapping': { zh: '删除映射', en: 'Delete mapping', ja: '割り当てを削除' },
  'keyMapping.settingsTitle': { zh: '设置按键映射', en: 'Set button mapping', ja: 'ボタン割り当て設定' },
  'keyMapping.pressToSet': { zh: '请按下手柄上的按键来设置映射', en: 'Press a button on the gamepad to set the mapping', ja: 'コントローラーのボタンを押して割り当てを設定してください' },
  'keyMapping.currentButton': { zh: '当前按钮', en: 'Current button', ja: '現在のボタン' },
  'keyMapping.listenStatus': { zh: '监听状态', en: 'Listening status', ja: 'リスニング状態' },
  'keyMapping.listening': { zh: '正在监听...请按下手柄按键', en: 'Listening... press a gamepad button', ja: 'リスニング中...コントローラーのボタンを押してください' },
  'keyMapping.ready': { zh: '准备就绪', en: 'Ready', ja: '準備完了' },

  // ===== touchMapping / reviews =====
  'touchMapping.arrowKeys': { zh: '方向键', en: 'Arrow keys', ja: '方向キー' },
  'reviews.shortComment': { zh: '吐槽', en: 'Short comment', ja: '感想' },
  'reviews.comment': { zh: '评论', en: 'Review', ja: 'レビュー' },

  // ===== time (utils/time.ts) =====
  'time.hoursMinutes': { zh: '{{hours}}小时{{minutes}}分钟', en: '{{hours}}h {{minutes}}m', ja: '{{hours}}時間{{minutes}}分' },
  'time.hours': { zh: '{{hours}}小时', en: '{{hours}}h', ja: '{{hours}}時間' },
  'time.minutes': { zh: '{{minutes}}分钟', en: '{{minutes}}m', ja: '{{minutes}}分' },
  'time.future': { zh: '未来', en: 'Upcoming', ja: '未来' },
  'time.today': { zh: '今天', en: 'Today', ja: '今日' },
  'time.yesterday': { zh: '昨天', en: 'Yesterday', ja: '昨日' },
  'time.daysAgo': { zh: '{{count}}天前', en: '{{count}} days ago', ja: '{{count}}日前' },
  'time.weeksAgo': { zh: '{{count}}周前', en: '{{count}} weeks ago', ja: '{{count}}週間前' },
  'time.monthsAgo': { zh: '{{count}}个月前', en: '{{count}} months ago', ja: '{{count}}か月前' },
  'time.yearsAgo': { zh: '{{count}}年前', en: '{{count}} years ago', ja: '{{count}}年前' },

  // ===== timezone.names =====
  'timezone.names.shanghai': { zh: '中国标准时间 (UTC+8)', en: 'China Standard Time (UTC+8)', ja: '中国標準時 (UTC+8)' },
  'timezone.names.tokyo': { zh: '日本标准时间 (UTC+9)', en: 'Japan Standard Time (UTC+9)', ja: '日本標準時 (UTC+9)' },
  'timezone.names.seoul': { zh: '韩国标准时间 (UTC+9)', en: 'Korea Standard Time (UTC+9)', ja: '韓国標準時 (UTC+9)' },
  'timezone.names.hongkong': { zh: '香港时间 (UTC+8)', en: 'Hong Kong Time (UTC+8)', ja: '香港時間 (UTC+8)' },
  'timezone.names.taipei': { zh: '台北时间 (UTC+8)', en: 'Taipei Time (UTC+8)', ja: '台北時間 (UTC+8)' },
  'timezone.names.singapore': { zh: '新加坡时间 (UTC+8)', en: 'Singapore Time (UTC+8)', ja: 'シンガポール時間 (UTC+8)' },
  'timezone.names.bangkok': { zh: '曼谷时间 (UTC+7)', en: 'Bangkok Time (UTC+7)', ja: 'バンコク時間 (UTC+7)' },
  'timezone.names.dubai': { zh: '迪拜时间 (UTC+4)', en: 'Dubai Time (UTC+4)', ja: 'ドバイ時間 (UTC+4)' },
  'timezone.names.london': { zh: '伦敦时间 (UTC+0)', en: 'London Time (UTC+0)', ja: 'ロンドン時間 (UTC+0)' },
  'timezone.names.paris': { zh: '巴黎时间 (UTC+1)', en: 'Paris Time (UTC+1)', ja: 'パリ時間 (UTC+1)' },
  'timezone.names.berlin': { zh: '柏林时间 (UTC+1)', en: 'Berlin Time (UTC+1)', ja: 'ベルリン時間 (UTC+1)' },
  'timezone.names.moscow': { zh: '莫斯科时间 (UTC+3)', en: 'Moscow Time (UTC+3)', ja: 'モスクワ時間 (UTC+3)' },
  'timezone.names.newyork': { zh: '纽约时间 (UTC-5)', en: 'New York Time (UTC-5)', ja: 'ニューヨーク時間 (UTC-5)' },
  'timezone.names.chicago': { zh: '芝加哥时间 (UTC-6)', en: 'Chicago Time (UTC-6)', ja: 'シカゴ時間 (UTC-6)' },
  'timezone.names.denver': { zh: '丹佛时间 (UTC-7)', en: 'Denver Time (UTC-7)', ja: 'デンバー時間 (UTC-7)' },
  'timezone.names.losangeles': { zh: '洛杉矶时间 (UTC-8)', en: 'Los Angeles Time (UTC-8)', ja: 'ロサンゼルス時間 (UTC-8)' },
  'timezone.names.saopaulo': { zh: '圣保罗时间 (UTC-3)', en: 'Sao Paulo Time (UTC-3)', ja: 'サンパウロ時間 (UTC-3)' },
  'timezone.names.sydney': { zh: '悉尼时间 (UTC+10)', en: 'Sydney Time (UTC+10)', ja: 'シドニー時間 (UTC+10)' },
  'timezone.names.auckland': { zh: '奥克兰时间 (UTC+12)', en: 'Auckland Time (UTC+12)', ja: 'オークランド時間 (UTC+12)' },
  'timezone.names.utc': { zh: '世界协调时间 (UTC)', en: 'Coordinated Universal Time (UTC)', ja: '協定世界時 (UTC)' },
};

const LANG_KEY = { 'zh-CN': 'zh', 'en-US': 'en', 'ja-JP': 'ja' };

function has(obj, parts) {
  let cur = obj;
  for (const p of parts) {
    if (cur && typeof cur === 'object' && p in cur) cur = cur[p];
    else return false;
  }
  return true;
}
function deepSet(obj, parts, val) {
  let cur = obj;
  for (let i = 0; i < parts.length - 1; i++) {
    const p = parts[i];
    if (!cur[p] || typeof cur[p] !== 'object') cur[p] = {};
    cur = cur[p];
  }
  cur[parts[parts.length - 1]] = val;
}

// 安全网：校验 _keys.txt 中每个键都已定义
const required = fs.readFileSync(path.join(__dirname, '_keys.txt'), 'utf8').split(/\r?\n/).map(s => s.trim()).filter(Boolean).filter(k => k !== 'gameInfo.');
const missing = required.filter(k => !(k in PATCHES));
if (missing.length) {
  console.error('PATCHES 缺少以下键：\n' + missing.join('\n'));
  process.exit(1);
}

let totalSet = 0;
for (const lang of LANGS) {
  const file = path.join(LOCALE_DIR, `${lang}.json`);
  const json = JSON.parse(fs.readFileSync(file, 'utf8'));
  const langKey = LANG_KEY[lang];
  let setCount = 0;
  for (const [key, tr] of Object.entries(PATCHES)) {
    const val = tr[langKey];
    if (val == null) continue;
    const parts = key.split('.');
    if (has(json, parts)) continue; // 已有则不覆盖
    deepSet(json, parts, val);
    setCount++;
  }
  fs.writeFileSync(file, JSON.stringify(json, null, 2) + '\n', 'utf8');
  console.log(`${lang}: +${setCount} keys`);
  totalSet += setCount;
}
console.log(`DONE required=${required.length} patches=${Object.keys(PATCHES).length} totalSet=${totalSet}`);

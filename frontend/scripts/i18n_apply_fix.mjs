// 应用 i18n 修复：把缺失的翻译键补进三种语言文件（仅补缺失，不覆盖已有值）。
// 用法： node scripts/i18n_apply_fix.mjs
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const LOCALE_DIR = path.join(__dirname, '..', 'src', 'i18n', 'locales');
const LANGS = ['zh-CN', 'en-US', 'ja-JP'];

// key -> { zh, en, ja }（null 表示该语言本就有值或不需补）
const PATCHES = {
  // ===== 三语全缺（真实键）=====
  'category.messages.noCategories': { zh: '暂无收藏夹', en: 'No categories yet', ja: '收藏夹がありません' },
  'category.modals.addToCategory.title': { zh: '添加到收藏夹', en: 'Add to Category', ja: '收藏夹に追加' },
  'common.complete': { zh: '完成', en: 'Complete', ja: '完了' },
  'common.saving': { zh: '保存中...', en: 'Saving...', ja: '保存中...' },
  'game.toasts.loadFailed': { zh: '加载游戏失败', en: 'Failed to load game', ja: 'ゲームの読み込みに失敗しました' },
  'game.toasts.videoUpdated': { zh: '视频已更新', en: 'Video updated', ja: '動画を更新しました' },
  'import.modals.dragDrop.error': { zh: '导入失败', en: 'Import failed', ja: 'インポートに失敗しました' },

  // ===== en-US 缺（zh 已有），同时 ja 也缺 =====
  'downloadedFiles.status.downloading': { en: 'Downloading', ja: 'ダウンロード中' },
  'downloadedFiles.status.not_downloading': { en: 'Not Downloading', ja: 'ダウンロード中以外' },
  'gameEdit.noVm': { en: 'Do not use VM', ja: '仮想マシンを使用しない' },
  'gameEdit.vmId': { en: 'Virtual Machine', ja: '仮想マシン' },
  'gameEdit.vmIdHint': { en: 'Select the VM to run this game', ja: 'このゲームを実行する仮想マシンを選択してください' },
  'reviews.type': { en: 'Type', ja: 'タイプ' },

  // ===== ja-JP 缺（zh + en 已有）=====
  'characterSelect.addNew': { ja: 'キャラクターを追加' },
  'characterSelect.appearedGames': { ja: '出演ゲーム' },
  'characterSelect.creatingNew': { ja: 'キャラクターを新規作成' },
  'characterSelect.current': { ja: '現在' },
  'characterSelect.editing': { ja: 'キャラクター編集' },
  'characterSelect.gender': { ja: '性別' },
  'characterSelect.height': { ja: '身長' },
  'characterSelect.imagePath': { ja: 'メイン画像 URL' },
  'characterSelect.images': { ja: '画像 URL' },
  'characterSelect.imagesPlaceholder': { ja: '複数の URL はカンマで区切ります' },
  'characterSelect.measurements': { ja: 'スリーサイズ' },
  'characterSelect.name': { ja: 'キャラクター名' },
  'characterSelect.nameRequired': { ja: 'キャラクター名を入力してください' },
  'characterSelect.noAppearedGames': { ja: '出演ゲームなし' },
  'characterSelect.noResults': { ja: 'キャラクターが見つかりません' },
  'characterSelect.otherNames': { ja: '別名' },
  'characterSelect.otherNamesAutoManaged': { ja: 'このフィールドは作品間のすべての別名を自動的に記録し、手動で編集できません' },
  'characterSelect.resultLimitHint': { ja: '結果が多すぎます。キーワードを入力して絞り込んでください' },
  'characterSelect.searchPlaceholder': { ja: '名前または別名で検索' },
  'characterSelect.selectOrCreateHint': { ja: '左側からキャラクターを選択するか、「キャラクターを追加」をクリックしてください' },
  'characterSelect.selectPlaceholder': { ja: 'キャラクターを選択してください' },
  'characterSelect.sort': { ja: '並び順' },
  'characterSelect.summary': { ja: '概要' },
  'characterSelect.title': { ja: 'キャラクターを選択' },
  'characterSelect.workName': { ja: '作品内のキャラクター名' },
  'characterSelect.workNamePlaceholder': { ja: 'この作品でのキャラクター名（未入力ならキャラクター名を使用）' },
  'characterSelect.workNameRequired': { ja: '作品内のキャラクター名を入力してください' },
  'common.associate': { ja: '関連付け' },
  'common.associateSuccess': { ja: '関連付けに成功しました' },
  'common.createFailed': { ja: '作成に失敗しました' },
  'common.createSuccess': { ja: '作成に成功しました' },
  'gameIntro.characterImage': { ja: 'キャラクター画像' },
  'gameIntro.deleteConfirm': { ja: 'このキャラクター設定を削除してもよろしいですか？' },
  'gameIntro.editCharacter': { ja: 'キャラクター編集' },
  'gameIntro.height': { ja: '身長' },
  'gameIntro.measurements': { ja: 'スリーサイズ' },
  'gameIntro.sort': { ja: '並び順' },
  'gameIntro.workImages': { ja: 'キャラクターのスクリーンショット' },
  'gameIntro.workImagesPlaceholder': { ja: '複数のURLはカンマで区切ります' },
  'imageField.fillAddUrl': { ja: '新しいURLを入力' },
  'imageField.fillUrl': { ja: 'URLを入力' },
  'imageField.noImages': { ja: '画像がありません' },
  'imageField.selectAddImage': { ja: '新しい画像を選択' },
  'imageField.selectImage': { ja: '画像を選択' },
  'imageField.urlPlaceholder': { ja: '画像URLを入力' },
  'staffSelect.addNew': { ja: 'スタッフを追加' },
  'staffSelect.appearedGames': { ja: '出演ゲーム' },
  'staffSelect.creatingNew': { ja: 'スタッフを新規作成' },
  'staffSelect.current': { ja: '現在' },
  'staffSelect.editing': { ja: 'スタッフ編集' },
  'staffSelect.gender': { ja: '性別' },
  'staffSelect.image': { ja: '画像 URL' },
  'staffSelect.name': { ja: 'スタッフ名' },
  'staffSelect.nameRequired': { ja: 'スタッフ名を入力してください' },
  'staffSelect.noAppearedGames': { ja: '出演ゲームなし' },
  'staffSelect.noResults': { ja: 'スタッフが見つかりません' },
  'staffSelect.otherNames': { ja: '別名' },
  'staffSelect.otherNamesAutoManaged': { ja: 'このフィールドは作品間のすべての別名を自動的に記録し、手動で編集できません' },
  'staffSelect.resultLimitHint': { ja: '結果が多すぎます。キーワードを入力して絞り込んでください' },
  'staffSelect.roles': { ja: '担当' },
  'staffSelect.rolesPlaceholder': { ja: '複数の担当はカンマで区切ります' },
  'staffSelect.searchPlaceholder': { ja: '名前または別名で検索' },
  'staffSelect.selectOrCreateHint': { ja: '左側からスタッフを選択するか、「スタッフを追加」をクリックしてください' },
  'staffSelect.selectPlaceholder': { ja: 'スタッフを選択してください' },
  'staffSelect.summary': { ja: '概要' },
  'staffSelect.title': { ja: 'スタッフを選択' },
  'staffSelect.workName': { ja: '作品内のCV名' },
  'staffSelect.workNamePlaceholder': { ja: 'この作品でのCV名（未入力ならスタッフ名を使用）' },
  'staffSelect.workNameRequired': { ja: '作品内のCV名を入力してください' },
  'touchMapping.refresh': { ja: '更新' },
  'touchMapping.toastRefreshed': { ja: 'ボタン設定を更新しました' },
  'vm.add': { ja: '仮想マシンを追加' },
  'vm.addSuccess': { ja: '仮想マシンを追加しました' },
  'vm.addTitle': { ja: '仮想マシンを追加' },
  'vm.deleteConfirm': { ja: 'この仮想マシンを削除してもよろしいですか？' },
  'vm.deleteError': { ja: '仮想マシンの削除に失敗しました' },
  'vm.deleteSuccess': { ja: '仮想マシンを削除しました' },
  'vm.editTitle': { ja: '仮想マシンを編集' },
  'vm.host': { ja: 'ホスト' },
  'vm.hostPassword': { ja: 'ホストパスワード' },
  'vm.hostPasswordPlaceholder': { ja: 'ESXホストパスワードを入力' },
  'vm.hostUrl': { ja: 'ホスト URL' },
  'vm.hostUrlPlaceholder': { ja: 'ESXホストURLを入力' },
  'vm.hostUser': { ja: 'ホストユーザー名' },
  'vm.hostUserPlaceholder': { ja: 'ESXホストユーザー名を入力' },
  'vm.listVms': { ja: '仮想マシン一覧' },
  'vm.loadError': { ja: '仮想マシンの読み込みに失敗しました' },
  'vm.loading': { ja: '読み込み中...' },
  'vm.name': { ja: '名前' },
  'vm.namePlaceholder': { ja: '仮想マシン名を入力' },
  'vm.password': { ja: 'パスワード' },
  'vm.passwordPlaceholder': { ja: '仮想マシンパスワードを入力' },
  'vm.path': { ja: '仮想マシンパス' },
  'vm.pathPlaceholder': { ja: '仮想マシンの .vmx ファイルパスを入力' },
  'vm.saveError': { ja: '仮想マシンの保存に失敗しました' },
  'vm.title': { ja: '仮想マシン管理' },
  'vm.type': { ja: 'タイプ' },
  'vm.typeEsx': { ja: 'ESX' },
  'vm.typeWorkstation': { ja: 'Workstation' },
  'vm.updateSuccess': { ja: '仮想マシンを更新しました' },
  'vm.userName': { ja: 'ユーザー名' },
  'vm.userNamePlaceholder': { ja: '仮想マシンユーザー名を入力' },
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
console.log(`DONE total=${totalSet}`);

import { useTranslation } from 'react-i18next';

export const I18nTestComponent = () => {
  const { t, i18n } = useTranslation();

  return (
    <div className="p-4 bg-white dark:bg-gray-800 rounded-lg shadow">
      <h3 className="text-lg font-semibold mb-4">多语言测试</h3>
      
      <div className="space-y-3">
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            当前语言:
          </label>
          <div className="text-sm text-gray-900 dark:text-white">
            {i18n.language}
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            测试翻译:
          </label>
          <div className="space-y-1 text-sm">
            <div>游戏状态 - 未开始: {t('gameStatus.not_started')}</div>
            <div>周期 - 日: {t('period.day')}</div>
            <div>提示类型 - 默认: {t('promptType.DEFAULT_SYSTEM')}</div>
            <div>来源类型 - 本地: {t('sourceType.local')}</div>
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            语言切换:
          </label>
          <div className="flex gap-2">
            <button
              onClick={() => i18n.changeLanguage('zh-CN')}
              className={`px-3 py-1 text-sm rounded ${
                i18n.language === 'zh-CN'
                  ? 'bg-blue-500 text-white'
                  : 'bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-200'
              }`}
            >
              中文
            </button>
            <button
              onClick={() => i18n.changeLanguage('en-US')}
              className={`px-3 py-1 text-sm rounded ${
                i18n.language === 'en-US'
                  ? 'bg-blue-500 text-white'
                  : 'bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-200'
              }`}
            >
              English
            </button>
            <button
              onClick={() => i18n.changeLanguage('ja-JP')}
              className={`px-3 py-1 text-sm rounded ${
                i18n.language === 'ja-JP'
                  ? 'bg-blue-500 text-white'
                  : 'bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-200'
              }`}
            >
              日本語
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
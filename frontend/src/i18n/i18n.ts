import i18next from 'i18next';
import { initReactI18next } from 'react-i18next';
import zhCN from './locales/zh-CN.json';
import enUS from './locales/en-US.json';
import jaJP from './locales/ja-JP.json';

// 获取浏览器语言设置
const getBrowserLanguage = () => {
  const lang = navigator.language || navigator.languages?.[0] || 'zh-CN';
  if (lang.startsWith('zh')) return 'zh-CN';
  if (lang.startsWith('ja')) return 'ja-JP';
  return 'en-US';
};

// 从localStorage获取用户选择的语言
const getUserLanguage = () => {
  return localStorage.getItem('lunabox-language') || getBrowserLanguage();
};

const resources = {
  'zh-CN': {
    translation: zhCN
  },
  'en-US': {
    translation: enUS
  },
  'ja-JP': {
    translation: jaJP
  }
};

i18next
  .use(initReactI18next)
  .init({
    resources,
    lng: getUserLanguage(),
    fallbackLng: 'en-US',
    interpolation: {
      escapeValue: false
    },
    react: {
      useSuspense: false
    }
  });

// 监听语言变化并保存到localStorage
i18next.on('languageChanged', (lng) => {
  localStorage.setItem('lunabox-language', lng);
});

export default i18next;
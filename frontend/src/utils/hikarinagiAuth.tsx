import toast from "react-hot-toast";
import { StartHikarinagiAuth } from "../../wailsjs/go/service/ConfigService";

// Hikarinagi 未授权错误标记（与后端 ErrHikarinagiNotAuthorized 对应）
export const HIKARINAGI_AUTH_MARKER = "HIKARINAGI_NOT_AUTHORIZED";

export function isHikarinagiAuthError(error: unknown): boolean {
  return String(error ?? "").includes(HIKARINAGI_AUTH_MARKER);
}

type TranslateFn = (key: string, opts?: any) => string;

// 全局去重：未授权提示弹窗同一时间只弹一个（批量任务会对每个游戏报错，避免刷屏）
let authPromptActive = false;
const AUTH_TOAST_ID = 'hikarinagi-auth';
const AUTH_TOAST_DURATION = 30000;

/**
 * 识别 Hikarinagi 未授权错误并弹出"去授权"引导。
 * 授权成功后自动执行 onAuthorized（通常是重试原操作）。
 * 返回是否为未授权错误（调用方据此决定是否再展示原始错误）。
 */
export function handleHikarinagiAuthError(
  error: unknown,
  t: TranslateFn,
  onAuthorized?: () => void,
): boolean {
  if (!isHikarinagiAuthError(error)) {
    return false;
  }
  if (authPromptActive) {
    return true;
  }
  authPromptActive = true;
  // 提示到期自动关闭后允许再次弹出
  setTimeout(() => { authPromptActive = false; }, AUTH_TOAST_DURATION);
  toast(
    (toastInstance) => (
      <div className="flex items-center gap-3">
        <span className="text-sm">{t('basic.hikarinagiAuthRequired')}</span>
        <button
          onClick={() => {
            toast.dismiss(toastInstance.id);
            authPromptActive = false;
            toast.promise(StartHikarinagiAuth(), {
              loading: t('basic.hikarinagiAuthorizing'),
              success: () => {
                onAuthorized?.();
                return t('basic.hikarinagiAuthSuccess');
              },
              error: (err: unknown) => t('basic.hikarinagiAuthFailed', { error: err }),
            });
          }}
          className="px-3 py-1 bg-sky-600 text-white text-sm rounded hover:bg-sky-700 whitespace-nowrap"
        >
          {t('basic.hikarinagiAuthBtn')}
        </button>
      </div>
    ),
    { duration: AUTH_TOAST_DURATION, id: AUTH_TOAST_ID },
  );
  return true;
}

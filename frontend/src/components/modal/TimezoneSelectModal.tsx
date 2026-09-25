import { useState } from "react";
import { useTranslation } from 'react-i18next';
import { BetterSelect } from "../ui/BetterSelect";

interface TimezoneSelectModalProps {
  isOpen: boolean;
  onConfirm: (timezone: string) => void;
}

// 常见时区列表
const COMMON_TIMEZONES = [
  { value: "Asia/Shanghai", tKey: "timezone.names.shanghai" },
  { value: "Asia/Tokyo", tKey: "timezone.names.tokyo" },
  { value: "Asia/Seoul", tKey: "timezone.names.seoul" },
  { value: "Asia/Singapore", tKey: "timezone.names.singapore" },
  { value: "Asia/Bangkok", tKey: "timezone.names.bangkok" },
  { value: "Asia/Dubai", tKey: "timezone.names.dubai" },
  { value: "Europe/London", tKey: "timezone.names.london" },
  { value: "Europe/Paris", tKey: "timezone.names.paris" },
  { value: "Europe/Berlin", tKey: "timezone.names.berlin" },
  { value: "Europe/Moscow", tKey: "timezone.names.moscow" },
  { value: "America/New_York", tKey: "timezone.names.newyork" },
  { value: "America/Chicago", tKey: "timezone.names.chicago" },
  { value: "America/Denver", tKey: "timezone.names.denver" },
  { value: "America/Los_Angeles", tKey: "timezone.names.losangeles" },
  { value: "America/Sao_Paulo", tKey: "timezone.names.saopaulo" },
  { value: "Australia/Sydney", tKey: "timezone.names.sydney" },
  { value: "Pacific/Auckland", tKey: "timezone.names.auckland" },
  { value: "UTC", tKey: "timezone.names.utc" },
];

export function TimezoneSelectModal({ isOpen, onConfirm }: TimezoneSelectModalProps) {
  const { t } = useTranslation();
  // 尝试从浏览器获取当前时区
  const browserTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  const [selectedTimezone, setSelectedTimezone] = useState(browserTimezone || "Asia/Shanghai");

  if (!isOpen)
    return null;

  const handleConfirm = () => {
    onConfirm(selectedTimezone);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
      <div className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-brand-800 border border-brand-200 dark:border-brand-700">
        <div className="flex items-start gap-4 mb-6">
          <div className="p-2 rounded-full bg-primary-100 text-primary-600 dark:bg-primary-900/30 dark:text-primary-400">
            <div className="i-mdi-earth text-2xl" />
          </div>
          <div className="flex-1">
            <h3 className="text-xl font-bold text-brand-900 dark:text-white mb-2">{t('timezone.modals.selectTimezone.title')}</h3>
            <p className="text-brand-600 dark:text-brand-400 text-sm leading-relaxed">
              {t('timezone.modals.selectTimezone.description')}
            </p>
          </div>
        </div>

        <div className="mb-6">
          <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-2">
            {t('timezone.labels.currentDetectedTimezone')}
            {" "}
            <span className="font-semibold text-primary-600 dark:text-primary-400">{browserTimezone || t('timezone.labels.unknownTimezone')}</span>
          </label>
          <BetterSelect
            value={selectedTimezone}
            onChange={setSelectedTimezone}
            options={COMMON_TIMEZONES.map(o => ({ value: o.value, label: t(o.tKey) }))}
            placeholder={t('timezone.placeholders.selectTimezone')}
            className="w-full"
          />
        </div>

        <div className="flex justify-end gap-3">
          <button
            type="button"
            onClick={handleConfirm}
            className="px-6 py-2.5 text-sm font-medium text-white bg-primary-600 hover:bg-primary-700 rounded-lg transition-colors shadow-sm shadow-primary-200 dark:shadow-none"
          >
            {t('timezone.buttons.confirmAndRestart')}
          </button>
        </div>
      </div>
    </div>
  );
}

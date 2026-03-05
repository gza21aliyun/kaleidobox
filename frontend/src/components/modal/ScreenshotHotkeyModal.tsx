import { createPortal } from "react-dom";
import { useState, useEffect, useRef } from "react";
import { useTranslation } from 'react-i18next';
import { arrayMapString } from "../utils/Utility";
import { enums, models, vo } from "../../../wailsjs/go/models";
import { MonitorKeySetting, CancelMonitorKeySetting } from "../../../wailsjs/go/service/HotkeyService";
import { formatLocalDate } from "../../utils/time";
// import { 
//   DeviceType, 
//   HotkeyActionType, 
//   ModifierKey, 
// } from "../utils/Hotkeys";
// import { HotkeyService } from "../../../wailsjs/go/service/HotkeyService";

interface ScreenshotHotkeyModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (hotkey: models.Hotkey) => void;
  currentHotkey?: models.Hotkey;
}

export function ScreenshotHotkeyModal({
  isOpen,
  onClose,
  onSave,
  currentHotkey
}: ScreenshotHotkeyModalProps) {
  const { t } = useTranslation();
  const [selectedDeviceType, setSelectedDeviceType] = useState<enums.DeviceType>(enums.DeviceType.KEYBOARD);
  const [keyCode, setKeyCode] = useState("");
  const [modifiers, setModifiers] = useState<string[]>([]);
  const [isListening, setIsListening] = useState(false);
  const [error, setError] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  // 支持的设备类型
  const deviceTypes = [
    { type: enums.DeviceType.KEYBOARD, name: "键盘", icon: "i-mdi-keyboard" },
    { type: enums.DeviceType.DUALSENSE, name: "DualSense", icon: "i-mdi-controller-playstation" },
    { type: enums.DeviceType.DUALSHOCK4, name: "DualShock 4", icon: "i-mdi-controller-playstation" },
    { type: enums.DeviceType.JOYCON, name: "Joy-Con", icon: "i-mdi-controller-nintendo" },
    { type: enums.DeviceType.XINPUT, name: "XInput", icon: "i-mdi-controller-xbox" }
  ];

  // 修饰键选项（仅键盘支持）
  const modifierOptions = [
    { key: enums.ModifierKey.CTRL, label: "Ctrl" },
    { key: enums.ModifierKey.SHIFT, label: "Shift" },
    { key: enums.ModifierKey.ALT, label: "Alt" },
    { key: enums.ModifierKey.WIN, label: "Win" }
  ];

  useEffect(() => {
    if (isOpen) {
      // 重置状态
      if (currentHotkey) {
        setSelectedDeviceType(currentHotkey.device_type);
        setKeyCode(currentHotkey.key_code);
        setModifiers(currentHotkey.modifiers || []);
      } else {
        setSelectedDeviceType(enums.DeviceType.KEYBOARD);
        setKeyCode("");
        setModifiers([]);
      }
      setError("");
      setIsListening(false);
    }
  }, [isOpen, currentHotkey]);

  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (!isListening || selectedDeviceType !== enums.DeviceType.KEYBOARD) return;
      
      e.preventDefault();
      
      // 获取按下的键
      const key = e.key.toUpperCase();
      let displayKey = key;
      
      // 特殊键处理
      switch (e.code) {
        case "Space":
          displayKey = "SPACE";
          break;
        case "Enter":
          displayKey = "ENTER";
          break;
        case "Escape":
          displayKey = "ESC";
          break;
        case "Tab":
          displayKey = "TAB";
          break;
        case "Backspace":
          displayKey = "BACKSPACE";
          break;
        case "ArrowUp":
          displayKey = "↑";
          break;
        case "ArrowDown":
          displayKey = "↓";
          break;
        case "ArrowLeft":
          displayKey = "←";
          break;
        case "ArrowRight":
          displayKey = "→";
          break;
        default:
          // 保持字母和数字的原始形式
          break;
      }
      
      setKeyCode(displayKey);
      setIsListening(false);
    };

    

    const handleKeyUp = (e: KeyboardEvent) => {
      if (!isListening || selectedDeviceType !== enums.DeviceType.KEYBOARD) return;
      // 不处理keyup事件
    };

    if (isListening && selectedDeviceType === enums.DeviceType.KEYBOARD) {
      window.addEventListener("keydown", handleKeyDown);
      window.addEventListener("keyup", handleKeyUp);
    }

    return () => {
      window.removeEventListener("keydown", handleKeyDown);
      window.removeEventListener("keyup", handleKeyUp);
    };
  }, [isListening, selectedDeviceType, isOpen]);

  const setListening = async (isListen: boolean) => {
    if (isListen) {
      setIsListening(true);
      const key = await MonitorKeySetting();
      setKeyCode(key.key_code);
      setIsListening(false);
    } else {
      setIsListening(false);
      CancelMonitorKeySetting()
    }
      
      
      
  }

  const handleSave = async () => {
    if (!keyCode.trim()) {
      setError(t('hotkey.errors.pleaseEnterKey'));
      return;
    }

    try {
        // const hotkey: models.Hotkey = new models.Hotkey{
        //     id = "",
        // }
      const hotkey: models.Hotkey = new models.Hotkey({
        id: currentHotkey?.id || `screenshot_${Date.now()}`,
        game_id: "global",
        name: t('hotkey.names.screenshot'),
        device_type: selectedDeviceType,
        key_code: keyCode,
        modifiers: selectedDeviceType === enums.DeviceType.KEYBOARD ? modifiers : [],
        action_type: enums.HotkeyActionType.SCREENSHOT,
        action_params: "",
        is_enabled: true,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        
      });

      await onSave(hotkey);
      onClose();
    } catch (err) {
      setError("保存失败: " + (err as Error).message);
    }
  };

  const toggleModifier = (modifier: string) => {
    if (selectedDeviceType !== enums.DeviceType.KEYBOARD) return;
    
    setModifiers(prev => 
      prev.includes(modifier) 
        ? prev.filter(m => m !== modifier)
        : [...prev, modifier]
    );
  };

  if (!isOpen) return null;

  return createPortal(
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
      <div className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-brand-800 border border-brand-200 dark:border-brand-700">
        <div className="flex items-start gap-4 mb-6">
          <div className="p-2 rounded-full bg-brand-100 text-brand-600 dark:bg-brand-900/30 dark:text-brand-400">
            <div className="i-mdi-camera text-2xl" />
          </div>
          <div className="flex-1">
            <h3 className="text-xl font-bold text-brand-900 dark:text-white mb-2">
              {t('hotkey.modals.screenshot.title')}
            </h3>
            <p className="text-brand-600 dark:text-brand-400 text-sm leading-relaxed">
              {t('hotkey.modals.screenshot.description')}
            </p>
          </div>
        </div>

        {/* 设备类型选择 */}
        <div className="mb-4">
          <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-2">
            {t('hotkey.labels.deviceType')}
          </label>
          <div className="grid grid-cols-2 gap-2">
            {deviceTypes.map(device => (
              <button
                key={device.type}
                onClick={() => setSelectedDeviceType(device.type)}
                className={`flex items-center gap-2 p-3 rounded-lg border transition-colors ${
                  selectedDeviceType === device.type
                    ? "border-brand-500 bg-brand-50 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300 dark:border-brand-400"
                    : "border-brand-200 hover:border-brand-300 dark:border-brand-700 dark:hover:border-brand-600"
                }`}
              >
                <div className={`${device.icon} text-lg`} />
                <span className="text-sm">{device.name}</span>
              </button>
            ))}
          </div>
        </div>

        {/* 修饰键选择（仅键盘） */}
        {selectedDeviceType === enums.DeviceType.KEYBOARD && (
          <div className="mb-4">
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-2">
              {t('hotkey.labels.modifiers')}（可选）
            </label>
            <div className="flex flex-wrap gap-2">
              {modifierOptions.map(modifier => (
                <button
                  key={modifier.key}
                  onClick={() => toggleModifier(modifier.key)}
                  className={`px-3 py-1 text-sm rounded ${
                    modifiers.includes(modifier.key)
                      ? "bg-brand-500 text-white"
                      : "bg-brand-100 text-brand-700 hover:bg-brand-200 dark:bg-brand-700 dark:text-brand-300 dark:hover:bg-brand-600"
                  }`}
                >
                  {modifier.label}
                </button>
              ))}
            </div>
          </div>
        )}

        {/* 按键输入 */}
        <div className="mb-6">
          <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-2">
            {t('hotkey.labels.key')}
            {selectedDeviceType === enums.DeviceType.KEYBOARD && `（${t('hotkey.labels.combinationSupported')}）`}
            {selectedDeviceType !== enums.DeviceType.KEYBOARD && `（${t('hotkey.labels.singleKeyOnly')}）`}
          </label>
          <div className="relative">
            <input
              ref={inputRef}
              type="text"
              value={keyCode}
              readOnly
              placeholder={
                isListening 
                  ? t('hotkey.placeholders.pressKey') 
                  : (selectedDeviceType === enums.DeviceType.KEYBOARD
                      ? t('hotkey.placeholders.clickToSetOrType') 
                      : t('hotkey.placeholders.connectControllerFirst'))
              }
              className="w-full px-4 py-3 bg-brand-50 border border-brand-200 rounded-lg text-brand-900 dark:bg-brand-900/50 dark:border-brand-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-brand-500"
            />
            <button
              type="button"
              onClick={() => setListening(!isListening)}
              disabled={isListening}
              className={`absolute right-2 top-1/2 -translate-y-1/2 px-3 py-1 text-xs font-medium rounded ${
                isListening
                  ? "bg-red-500 text-white"
                  : "bg-brand-100 text-brand-700 hover:bg-brand-200 dark:bg-brand-700 dark:text-brand-300 dark:hover:bg-brand-600"
              }`}
            >
              {isListening ? t('hotkey.buttons.stopListening') : t('hotkey.buttons.setKey')}
            </button>
          </div>
          
          {/* 显示当前组合键 */}
          {selectedDeviceType === enums.DeviceType.KEYBOARD && modifiers.length > 0 && keyCode && (
            <div className="mt-2 text-sm text-brand-600 dark:text-brand-400">
              {t('hotkey.labels.currentCombination')}: {modifiers.join(" + ")} + {keyCode}
            </div>
          )}
        </div>

        {/* 错误信息 */}
        {error && (
          <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 dark:bg-red-900/20 dark:border-red-800 dark:text-red-300">
            {error}
          </div>
        )}

        {/* 操作按钮 */}
        <div className="flex justify-end gap-3">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-brand-700 hover:bg-brand-100 rounded-lg dark:text-brand-300 dark:hover:bg-brand-700 transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={handleSave}
            className="px-4 py-2 text-sm font-medium text-white bg-brand-600 hover:bg-brand-700 rounded-lg transition-colors shadow-sm shadow-brand-200 dark:shadow-none"
          >
            {t('common.save')}
          </button>
        </div>
      </div>
    </div>,
    document.body,
  );
}
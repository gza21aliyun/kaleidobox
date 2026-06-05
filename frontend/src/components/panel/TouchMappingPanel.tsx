import { useState, useEffect, useCallback, forwardRef, useImperativeHandle } from 'react';
import { toast } from "react-hot-toast";
import { models, enums, service } from '../../../wailsjs/go/models';
import { BetterButton } from '../ui/BetterButton';
import i18next from "../../i18n/i18n";
const t = i18next.t;
import {
  GetHotkeysByGameID,
  UpdateHotkey,
  AddHotkey,
  DeleteHotkey,
  GetGlobalHotkeys,
  StartTouchEditMode,
  StopTouchEditMode,
  UpdateTouchEditModeButtons,
} from '../../../wailsjs/go/service/HotkeyService';

interface TouchMappingPanelProps {
  gameId: string;
}

export interface TouchMappingPanelRef {
  saveAll: () => Promise<void>;
}

interface TouchButton {
  id: string;
  hotkeyID: string;
  name: string;
  actionParams: string; // virtual key code or name
  keyCode: string;      // "x:N;y:N" format
  x: number;
  y: number;
  gameId: string;       // "global" or game specific id
  isNew: boolean;
  actionType: enums.HotkeyActionType; // 功能键类型，默认 CUSTOM
}

// 功能键选项
const ACTION_KEYS = [
  { actionType: enums.HotkeyActionType.SCREENSHOT, name: 'SCREENSHOT', label: '截图', icon: '📷' },
];

// 常见虚拟键 - 名称和代码映射
const VIRTUAL_KEYS = [
  { name: 'ENTER', label: 'Enter' },
  { name: 'CONTROL', label: 'Ctrl' },
  { name: 'SHIFT', label: 'Shift' },
  { name: 'ALT', label: 'Alt' },
  { name: 'SPACE', label: 'Space' },
  { name: 'ESC', label: 'Esc' },
  { name: 'TAB', label: 'Tab' },
  { name: 'F1', label: 'F1' },
  { name: 'F2', label: 'F2' },
  { name: 'F3', label: 'F3' },
  { name: 'F4', label: 'F4' },
  { name: 'F5', label: 'F5' },
  { name: 'F6', label: 'F6' },
  { name: 'F7', label: 'F7' },
  { name: 'F8', label: 'F8' },
  { name: 'F9', label: 'F9' },
  { name: 'F10', label: 'F10' },
  { name: 'F11', label: 'F11' },
  { name: 'F12', label: 'F12' },
  { name: 'A', label: 'A' },
  { name: 'B', label: 'B' },
  { name: 'C', label: 'C' },
  { name: 'D', label: 'D' },
  { name: 'E', label: 'E' },
  { name: 'F', label: 'F' },
  { name: 'G', label: 'G' },
  { name: 'H', label: 'H' },
  { name: 'I', label: 'I' },
  { name: 'J', label: 'J' },
  { name: 'K', label: 'K' },
  { name: 'L', label: 'L' },
  { name: 'M', label: 'M' },
  { name: 'N', label: 'N' },
  { name: 'O', label: 'O' },
  { name: 'P', label: 'P' },
  { name: 'Q', label: 'Q' },
  { name: 'R', label: 'R' },
  { name: 'S', label: 'S' },
  { name: 'T', label: 'T' },
  { name: 'U', label: 'U' },
  { name: 'V', label: 'V' },
  { name: 'W', label: 'W' },
  { name: 'X', label: 'X' },
  { name: 'Y', label: 'Y' },
  { name: 'Z', label: 'Z' },
  { name: '0', label: '0' },
  { name: '1', label: '1' },
  { name: '2', label: '2' },
  { name: '3', label: '3' },
  { name: '4', label: '4' },
  { name: '5', label: '5' },
  { name: '6', label: '6' },
  { name: '7', label: '7' },
  { name: '8', label: '8' },
  { name: '9', label: '9' },
];

// 新增按钮默认坐标（组件外常量，避免渲染时重建导致 useCallback 依赖变化）
const DEFAULT_TOUCH_X = 1700;
const DEFAULT_TOUCH_Y = 340;

export const TouchMappingPanel = forwardRef<TouchMappingPanelRef, TouchMappingPanelProps>(({ gameId }, ref) => {
  const [touchButtons, setTouchButtons] = useState<TouchButton[]>([]);
  const [loading, setLoading] = useState(false);
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [waitingForKey, setWaitingForKey] = useState(false);
  const [editMode, setEditMode] = useState(false);
  const [selectedActionKey, setSelectedActionKey] = useState<typeof ACTION_KEYS[0] | null>(null);

  // 从 hotkeys 解析坐标
  const parseKeyCodeToXY = (keyCode: string): { x: number; y: number } => {
    let x = 0, y = 0;
    const parts = keyCode.split(';').map((p) => p.trim());
    for (const p of parts) {
      if (p.startsWith('x:')) {
        x = parseInt(p.slice(2), 10) || 0;
      } else if (p.startsWith('y:')) {
        y = parseInt(p.slice(2), 10) || 0;
      }
    }
    return { x, y };
  };

  // 加载触摸按钮配置（只加载该 gameId 的按钮，不合并全局）
  const loadTouchButtons = useCallback(async () => {
    try {
      setLoading(true);
      // 只获取该 gameId 的触摸按钮
      const hotkeys = await GetHotkeysByGameID(gameId);
      const touchButtons = hotkeys.filter(
        (h) => h.device_type === enums.DeviceType.TOUCH
      );

      // 转换为 TouchButton 数组
      const buttons: TouchButton[] = touchButtons.map((h) => {
        const pos = parseKeyCodeToXY(h.key_code);
        return {
          id: h.id,
          hotkeyID: h.id,
          name: h.name,
          actionParams: h.action_params,
          keyCode: h.key_code,
          x: pos.x,
          y: pos.y,
          gameId: h.game_id,
          isNew: false,
          actionType: h.action_type || enums.HotkeyActionType.CUSTOM,
        };
      });

      setTouchButtons(buttons);
      console.log(`已加载触摸按钮: ${buttons.length} 个 - [${buttons.map(b => b.name).join(', ')}]`);
    } catch (err) {
      console.error('加载触摸按钮失败:', err);
      toast.error(t('touchMapping.toastLoadFailed'));
    } finally {
      setLoading(false);
    }
  }, [gameId]);

  useEffect(() => {
    loadTouchButtons();
  }, [loadTouchButtons]);

  // 键盘事件处理 - 监听新增按钮时用户按下的按键
  useEffect(() => {
    if (waitingForKey && !selectedActionKey) {
      const handleKeyDown = (e: KeyboardEvent) => {
        e.preventDefault();
        const keyPressed = e.key;
        const keyCode = e.code;

        // 将按键转换为标准化名称（用于 action_params 和显示）
        let keyName = keyPressed;
        if (keyPressed === 'Enter') keyName = 'ENTER';
        else if (keyPressed === ' ') keyName = 'SPACE';
        else if (keyPressed === 'Escape') keyName = 'ESC';
        else if (keyPressed === 'Tab') keyName = 'TAB';
        else if (keyPressed === 'Control') keyName = 'CONTROL';
        else if (keyPressed === 'Shift') keyName = 'SHIFT';
        else if (keyPressed === 'Alt') keyName = 'ALT';
        else if (keyPressed.length === 1) keyName = keyPressed.toUpperCase();
        else keyName = keyPressed.toUpperCase().replace('KEY', '');

        // 创建新按钮
        const btn: TouchButton = {
          id: crypto.randomUUID(),
          hotkeyID: '',
          name: keyName,
          actionParams: keyName,
          keyCode: `x:${DEFAULT_TOUCH_X};y:${DEFAULT_TOUCH_Y}`,
          x: DEFAULT_TOUCH_X,
          y: DEFAULT_TOUCH_Y,
          gameId: gameId,
          isNew: true,
          actionType: enums.HotkeyActionType.CUSTOM,
        };
        setTouchButtons((prev) => [...prev, btn]);

        setWaitingForKey(false);
        setShowAddDialog(false);
        toast.success(t('touchMapping.toastButtonAdded', { buttonName: keyName }));
      };

      window.addEventListener('keydown', handleKeyDown);
      return () => window.removeEventListener('keydown', handleKeyDown);
    }
  }, [waitingForKey, gameId, selectedActionKey]);

  // 新增按钮（改为打开监听按键弹窗 - 与 KeyMappingPanel 风格一致）
  const handleOpenAddDialog = () => {
    setShowAddDialog(true);
    setWaitingForKey(true);
    setSelectedActionKey(null);
  };

  // 取消新增按钮
  const handleCancelAdd = () => {
    setShowAddDialog(false);
    setWaitingForKey(false);
    setSelectedActionKey(null);
  };

  // 选择功能键
  const handleSelectActionKey = (actionKey: typeof ACTION_KEYS[0]) => {
    const btn: TouchButton = {
      id: crypto.randomUUID(),
      hotkeyID: '',
      name: actionKey.label,
      actionParams: actionKey.name,
      keyCode: `x:${DEFAULT_TOUCH_X};y:${DEFAULT_TOUCH_Y}`,
      x: DEFAULT_TOUCH_X,
      y: DEFAULT_TOUCH_Y,
      gameId: gameId,
      isNew: true,
      actionType: actionKey.actionType,
    };
    setTouchButtons((prev) => [...prev, btn]);
    setSelectedActionKey(null);
    setShowAddDialog(false);
    toast.success(t('touchMapping.toastButtonAdded', { buttonName: actionKey.label }));
  };

  // 载入默认按钮（Enter + Ctrl）— 清除原有按钮后仅存入内存
  const handleLoadDefaults = useCallback(() => {
    const defaultButtons: TouchButton[] = [
      {
        id: crypto.randomUUID(),
        hotkeyID: '',
        name: 'ENTER',
        actionParams: 'ENTER',
        keyCode: `x:${DEFAULT_TOUCH_X};y:${DEFAULT_TOUCH_Y}`,
        x: DEFAULT_TOUCH_X,
        y: DEFAULT_TOUCH_Y,
        gameId: gameId,
        isNew: true,
        actionType: enums.HotkeyActionType.CUSTOM,
      },
      {
        id: crypto.randomUUID(),
        hotkeyID: '',
        name: 'CONTROL',
        actionParams: 'CONTROL',
        keyCode: `x:${DEFAULT_TOUCH_X};y:${DEFAULT_TOUCH_Y + 200}`,
        x: DEFAULT_TOUCH_X,
        y: DEFAULT_TOUCH_Y + 200,
        gameId: gameId,
        isNew: true,
        actionType: enums.HotkeyActionType.CUSTOM,
      },
    ];
    setTouchButtons(defaultButtons);
    toast.success(t('touchMapping.toastDefaultsLoaded'));
  }, [gameId]);

  // 载入全局配置（仅游戏页面可用）— 清除原有按钮后仅存入内存，game_id 改为本游戏
  const handleLoadGlobal = useCallback(async () => {
    try {
      setLoading(true);
      const globalHotkeys = await GetGlobalHotkeys();
      const globalTouch = globalHotkeys.filter(
        (h) => h.device_type === enums.DeviceType.TOUCH
      );

      if (globalTouch.length === 0) {
        toast.error(t('touchMapping.toastNoGlobalButtons'));
        return;
      }

      const buttons: TouchButton[] = globalTouch.map((h) => {
        const pos = parseKeyCodeToXY(h.key_code);
        return {
          id: crypto.randomUUID(),
          hotkeyID: '',
          name: h.name,
          actionParams: h.action_params,
          keyCode: h.key_code,
          x: pos.x,
          y: pos.y,
          gameId: gameId,
          isNew: true,
          actionType: h.action_type || enums.HotkeyActionType.CUSTOM,
        };
      });

      setTouchButtons(buttons);
      toast.success(t('touchMapping.toastGlobalLoaded', { count: buttons.length }));
    } catch (err) {
      console.error('载入全局按钮失败:', err);
      toast.error(t('touchMapping.toastLoadFailed'));
    } finally {
      setLoading(false);
    }
  }, [gameId]);

  // 删除按钮（仅从内存移除）
  const handleDeleteButton = (id: string) => {
    setTouchButtons((prev) => prev.filter((b) => b.id !== id));
  };

  // 更新按钮 X/Y 坐标（编辑模式下用）
  const handleUpdatePosition = (id: string, x: number, y: number) => {
    setTouchButtons((prev) =>
      prev.map((b) =>
        b.id === id
          ? { ...b, x, y, keyCode: `x:${x};y:${y}` }
          : b
      )
    );
  };

  // 虚拟键名称 -> 虚拟键代码（十进制）
  const virtualKeyNameToCode = (name: string): number => {
    const map: { [key: string]: number } = {
      ENTER: 0x0d, CONTROL: 0x11, SHIFT: 0x10, ALT: 0x12,
      SPACE: 0x20, ESC: 0x1b, TAB: 0x09,
      F1: 0x70, F2: 0x71, F3: 0x72, F4: 0x73, F5: 0x74, F6: 0x75, F7: 0x76, F8: 0x77, F9: 0x78, F10: 0x79, F11: 0x7a, F12: 0x7b,
      A: 0x41, B: 0x42, C: 0x43, D: 0x44, E: 0x45, F: 0x46, G: 0x47, H: 0x48, I: 0x49, J: 0x4a, K: 0x4b, L: 0x4c, M: 0x4d,
      N: 0x4e, O: 0x4f, P: 0x50, Q: 0x51, R: 0x52, S: 0x53, T: 0x54, U: 0x55, V: 0x56, W: 0x57, X: 0x58, Y: 0x59, Z: 0x5a,
      '0': 0x30, '1': 0x31, '2': 0x32, '3': 0x33, '4': 0x34, '5': 0x35, '6': 0x36, '7': 0x37, '8': 0x38, '9': 0x39,
    };
    return map[name] ?? 0x0d;
  };

  // 将 TouchButton[] 转换为后端 TouchButtonInfo[]
  const touchButtonsToBackend = (buttons: TouchButton[]): service.TouchButtonInfo[] => {
    return buttons.map((btn, idx) => ({
      Index: idx,
      Name: btn.name,
      VirtualKey: virtualKeyNameToCode(btn.actionParams),
      X: btn.x,
      Y: btn.y,
      ActionType: btn.actionType,
    }));
  };

  // 编辑模式下同步按钮列表到后端 overlay
  const syncButtonsToBackend = useCallback(async (buttons: TouchButton[]) => {
    if (!editMode) return;
    try {
      const backendButtons = touchButtonsToBackend(buttons);
      await UpdateTouchEditModeButtons(backendButtons);
    } catch (err) {
      console.error('同步按钮到后端失败:', err);
    }
  }, [editMode]);

  // 监听 touchButtons 变化，编辑模式下同步到后端 overlay
  useEffect(() => {
    syncButtonsToBackend(touchButtons);
  }, [touchButtons, syncButtonsToBackend]);

  // 启动编辑模式 - 通过 Go 后端调用，让按钮可拖动
  const handleStartEditMode = async () => {
    if (touchButtons.length === 0) {
      toast.error(t('touchMapping.toastNoButtonsToEdit'));
      return;
    }
    try {
      // 转换为后端需要的结构
      const buttons = touchButtons.map((btn, idx) => ({
        Index: idx,
        Name: btn.name,
        VirtualKey: virtualKeyNameToCode(btn.actionParams),
        X: btn.x,
        Y: btn.y,
        ActionType: btn.actionType,
      }));
      await StartTouchEditMode(buttons);
      setEditMode(true);
      toast.success(t('touchMapping.toastEditModeStarted'));
    } catch (err) {
      console.error('启动编辑模式失败:', err);
      toast.error(t('touchMapping.toastStartEditFailed'));
    }
  };

  // 退出编辑模式 - 从后端获取更新后的位置
  const handleStopEditMode = async () => {
    try {
      const positions = await StopTouchEditMode();
      setEditMode(false);
      // 用后端返回的位置更新列表
      setTouchButtons((prev) => {
        const next = [...prev];
        positions.forEach((pos) => {
          if (pos.Index >= 0 && pos.Index < next.length) {
            next[pos.Index] = {
              ...next[pos.Index],
              x: pos.X,
              y: pos.Y,
              keyCode: `x:${pos.X};y:${pos.Y}`,
            };
          }
        });
        return next;
      });
      toast.success(t('touchMapping.toastEditModeFinished'));
    } catch (err) {
      console.error('退出编辑模式失败:', err);
      toast.error(t('touchMapping.toastStopEditFailed'));
    }
  };

  // 保存所有触摸按钮到数据库（只保存到该 gameId）
  const handleSaveAll = useCallback(async () => {
    try {
      // 1. 获取该 gameId 的现有触摸按钮
      const existingHotkeys = await GetHotkeysByGameID(gameId);
      const existingTouchHotkeys = existingHotkeys.filter(
        (h) => h.device_type === enums.DeviceType.TOUCH
      );

      // 2. 删除该 gameId 的现有触摸按钮
      const deletedNames = existingTouchHotkeys.map((h) => h.name).join(', ');
      console.log(`[保存触摸按钮] 删除现有按钮: [${deletedNames}] (共${existingTouchHotkeys.length}个)`);
      for (const hk of existingTouchHotkeys) {
        await DeleteHotkey(hk.id);
      }

      // 3. 把当前内存中的按钮转换为 Hotkey 对象，全部使用新的 UUID
      const addedNames: string[] = [];
      for (const btn of touchButtons) {
        const hotkey = new models.Hotkey({
          id: crypto.randomUUID(), // 强制使用新的 UUID，避免主键冲突
          game_id: gameId,
          name: btn.name,
          device_type: enums.DeviceType.TOUCH,
          key_code: btn.keyCode,
          modifiers: [],
          action_type: btn.actionType,
          action_params: btn.actionParams,
          is_enabled: true,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        });

        await AddHotkey(hotkey);
        addedNames.push(btn.name);
      }
      console.log(`[保存触摸按钮] 添加按钮: [${addedNames.join(', ')}] (共${touchButtons.length}个)`);

      toast.success(t('touchMapping.toastSavedAll'));
    } catch (err) {
      console.error('保存触摸按钮失败:', err);
      toast.error(t('touchMapping.toastSaveFailed'));
    }
  }, [gameId, touchButtons]);

  // 暴露保存方法给父组件
  useImperativeHandle(
    ref,
    () => ({
      saveAll: handleSaveAll,
    }),
    [handleSaveAll]
  );



  return (
    <div className="w-full h-full p-8 overflow-y-auto">
      {/* 操作按钮 */}
      <div className="flex flex-wrap gap-3 mb-6">
        {!editMode && (
          <>
            <BetterButton
              onClick={handleOpenAddDialog}
              icon="i-mdi-plus"
              variant="ghost"
              className="!bg-blue-600 !hover:bg-blue-700 !text-white !dark:bg-blue-500 !dark:hover:bg-blue-600 !dark:text-white px-6 py-2 rounded-lg shadow-md"
            >
              {t('touchMapping.addButton')}
            </BetterButton>

            {/* 载入默认：Enter + Ctrl */}
            <BetterButton
              onClick={handleLoadDefaults}
              icon="i-mdi-restore"
              variant="ghost"
              className="!bg-indigo-600 !hover:bg-indigo-700 !text-white !dark:bg-indigo-500 !dark:hover:bg-indigo-600 !dark:text-white px-6 py-2 rounded-lg shadow-md"
            >
              {t('touchMapping.loadDefaults')}
            </BetterButton>

            {/* 载入全局：仅在游戏页面显示（gameId 不是 'global'） */}
            {gameId !== 'global' && (
              <BetterButton
                onClick={handleLoadGlobal}
                icon="i-mdi-upload"
                variant="ghost"
                className="!bg-teal-600 !hover:bg-teal-700 !text-white !dark:bg-teal-500 !dark:hover:bg-teal-600 !dark:text-white px-6 py-2 rounded-lg shadow-md"
              >
                {t('touchMapping.loadGlobal')}
              </BetterButton>
            )}
          </>
        )}

        {/* 编辑模式按钮：只在按钮列表不为空时显示 */}
        {touchButtons.length > 0 && (
          !editMode ? (
            <BetterButton
              onClick={handleStartEditMode}
              icon="i-mdi-pencil"
              variant="ghost"
              className="!bg-yellow-600 !hover:bg-yellow-700 !text-white !dark:bg-yellow-500 !dark:hover:bg-yellow-600 !dark:text-white px-6 py-2 rounded-lg shadow-md"
            >
              {t('touchMapping.startEditMode')}
            </BetterButton>
          ) : (
            <BetterButton
              onClick={handleStopEditMode}
              icon="i-mdi-check"
              variant="ghost"
              className="!bg-green-600 !hover:bg-green-700 !text-white !dark:bg-green-500 !dark:hover:bg-green-600 !dark:text-white px-6 py-2 rounded-lg shadow-md"
            >
              {t('touchMapping.finishEdit')}
            </BetterButton>
          )
        )}
      </div>

      {/* 按钮列表 */}
      <div className="bg-white rounded-xl shadow-lg overflow-hidden dark:bg-brand-800">
        <table className="w-full">
          <thead className="bg-gray-100 dark:bg-brand-700">
            <tr>
              <th className="px-4 py-3 text-left text-sm font-semibold text-gray-700 dark:text-gray-200">
                {t('touchMapping.colName')}
              </th>
              <th className="px-4 py-3 text-left text-sm font-semibold text-gray-700 dark:text-gray-200">
                {t('touchMapping.colMappedKey')}
              </th>
              <th className="px-4 py-3 text-left text-sm font-semibold text-gray-700 dark:text-gray-200">
                {t('touchMapping.colX')}
              </th>
              <th className="px-4 py-3 text-left text-sm font-semibold text-gray-700 dark:text-gray-200">
                {t('touchMapping.colY')}
              </th>
              <th className="px-4 py-3 text-left text-sm font-semibold text-gray-700 dark:text-gray-200">
                {t('touchMapping.colAction')}
              </th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-gray-500">
                  {t('common.loading')}
                </td>
              </tr>
            ) : touchButtons.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-gray-500">
                  {t('touchMapping.noTouchButtons')}
                </td>
              </tr>
            ) : (
              touchButtons.map((btn) => (
                <tr
                  key={btn.id}
                  className="border-t border-gray-200 dark:border-brand-700 hover:bg-gray-50 dark:hover:bg-brand-700/50"
                >
                  <td className="px-4 py-3 text-gray-800 dark:text-gray-200">
                    {btn.actionType === enums.HotkeyActionType.SCREENSHOT && '📷'}
                    {btn.name}
                    {btn.isNew && (
                      <span className="ml-2 px-2 py-0.5 text-xs bg-blue-600 text-white rounded dark:bg-blue-500 dark:text-white font-medium">
                        {t('touchMapping.isNew')}
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-gray-800 dark:text-gray-200">
                    {btn.actionType === enums.HotkeyActionType.SCREENSHOT ? '截图' : btn.actionParams}
                  </td>
                  <td className="px-4 py-3 text-gray-800 dark:text-gray-200">
                    <input
                      type="number"
                      value={btn.x}
                      onChange={(e) =>
                        handleUpdatePosition(
                          btn.id,
                          parseInt(e.target.value) || 0,
                          btn.y
                        )
                      }
                      className="w-24 px-2 py-1 border border-gray-300 rounded text-sm dark:bg-brand-900 dark:border-brand-600 dark:text-gray-200"
                    />
                  </td>
                  <td className="px-4 py-3 text-gray-800 dark:text-gray-200">
                    <input
                      type="number"
                      value={btn.y}
                      onChange={(e) =>
                        handleUpdatePosition(
                          btn.id,
                          btn.x,
                          parseInt(e.target.value) || 0
                        )
                      }
                      className="w-24 px-2 py-1 border border-gray-300 rounded text-sm dark:bg-brand-900 dark:border-brand-600 dark:text-gray-200"
                    />
                  </td>
                  <td className="px-4 py-3">
                    {!editMode && (
                      <button
                        onClick={() => handleDeleteButton(btn.id)}
                        className="text-red-500 hover:text-red-600 text-sm font-medium"
                      >
                        {t('touchMapping.deleteButton')}
                      </button>
                    )}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* 提示 */}
      <div className="mt-6 p-4 bg-blue-50 border border-blue-200 rounded-lg text-sm text-blue-700 dark:bg-brand-900/50 dark:border-brand-700 dark:text-blue-300">
        <div className="font-semibold mb-1">{t('touchMapping.hintTitle')}</div>
        <ul className="list-disc pl-5 space-y-1">
          <li>{t('touchMapping.hintMemoryOnly')}</li>
          <li>{t('touchMapping.hintEditMode')}</li>
          <li>{t('touchMapping.hintOnlyInGame')}</li>
        </ul>
      </div>

      {/* 新增按钮弹窗 - 参考 KeyMappingPanel 风格 */}
      {showAddDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
          <div className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-brand-800 border border-brand-200 dark:border-brand-700">
            <div className="flex items-start gap-4 mb-6">
              <div className="p-2 rounded-full bg-brand-100 text-brand-600 dark:bg-brand-900/30 dark:text-brand-400">
                <div className="i-mdi-gesture-tap-button text-2xl" />
              </div>
              <div className="flex-1">
                <h3 className="text-xl font-bold text-brand-900 dark:text-white mb-2">
                  {t('touchMapping.addButtonDialogTitle')}
                </h3>
                <p className="text-brand-600 dark:text-brand-400 text-sm leading-relaxed">
                  {t('touchMapping.addButtonDialogHint')}
                </p>
              </div>
            </div>

            {/* 功能键选项 */}
            <div className="mb-6">
              <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-2">
                {t('touchMapping.functionKeys')}
              </label>
              <div className="grid grid-cols-2 gap-2">
                {ACTION_KEYS.map((actionKey) => (
                  <button
                    key={actionKey.name}
                    onClick={() => handleSelectActionKey(actionKey)}
                    className="flex items-center justify-center gap-2 p-3 bg-brand-50 hover:bg-brand-100 border border-brand-200 rounded-lg transition-colors dark:bg-brand-900/50 dark:border-brand-700 dark:hover:bg-brand-800"
                  >
                    <span className="text-xl">{actionKey.icon}</span>
                    <span className="text-sm text-brand-700 dark:text-brand-300">{actionKey.label}</span>
                  </button>
                ))}
              </div>
            </div>

            {/* 分隔线 */}
            <div className="border-t border-brand-200 dark:border-brand-700 my-4"></div>

            <div className="mb-6">
              <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-2">
                {t('touchMapping.orPressKey')}
              </label>
              <div className="p-4 bg-brand-50 border border-brand-200 rounded-lg text-brand-900 dark:bg-brand-900/50 dark:border-brand-700 dark:text-white text-center">
                {waitingForKey ? t('touchMapping.listening') : t('touchMapping.ready')}
              </div>
            </div>

            <div className="flex justify-end gap-3">
              <button
                onClick={handleCancelAdd}
                className="px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100 rounded-lg dark:text-gray-200 dark:hover:bg-brand-700"
              >
                {t('common.cancel')}
              </button>
            </div>
          </div>
        </div>
      )}

    </div>
  );
});

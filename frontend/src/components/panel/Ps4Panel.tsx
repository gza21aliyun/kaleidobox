import { useState, useEffect } from 'react';
import { BetterButton } from '../ui/BetterButton';
import { toast } from "react-hot-toast";
import { models, enums } from '../../../wailsjs/go/models';
import { GetHotkeysByGameID, UpdateHotkey, AddHotkey, DeleteHotkey } from '../../../wailsjs/go/service/HotkeyService';
import i18next from "../../i18n/i18n";
const t = i18next.t;

interface Ps4PanelProps {
  gameId: string;
}

interface KeyMapping {
  button: string;
  position: { x: number; y: number };
  label: string;
}

export function Ps4Panel({ gameId }: Ps4PanelProps) {
  const [hotkeys, setHotkeys] = useState<models.Hotkey[]>([]);
  const [loading, setLoading] = useState(true);
  const [showMappingDialog, setShowMappingDialog] = useState(false);
  const [currentMappingButton, setCurrentMappingButton] = useState<string | null>(null);
  const [newKeyCode, setNewKeyCode] = useState('');
  const [waitingForKey, setWaitingForKey] = useState(false);

  // PS4手柄按键位置定义
  const buttonMappings: KeyMapping[] = [
    { button: 'square', position: { x: 52, y: 17 }, label: 'A' },
    { button: 'cross', position: { x: 58, y: 25 }, label: 'S' },
    { button: 'circle', position: { x: 65, y: 17 }, label: 'D' },
    { button: 'triangle', position: { x: 58, y: 10 }, label: 'W' },
    { button: 'l1', position: { x: 15, y: -5 }, label: 'Q' },
    { button: 'r1', position: { x: 60, y: -5 }, label: 'E' },
    { button: 'l2', position: { x: 10, y: -15 }, label: 'Z' },
    { button: 'r2', position: { x: 65, y: -15 }, label: 'C' },
    { button: 'l3', position: { x: 23, y: 40 }, label: '1' },
    { button: 'r3', position: { x: 49, y: 40 }, label: '3' },
    { button: 'share', position: { x: 22, y: 5 }, label: 'V' },
    { button: 'options', position: { x: 50, y: 5 }, label: 'M' },
    { button: 'ps', position: { x: 37, y: 30 }, label: 'B' },
    { button: 'touchpad', position: { x: 37, y: 13 }, label: 'N' },
    { button: 'l3-left', position: { x: 15, y: 40 }, label: '←' },
    { button: 'l3-up', position: { x: 23, y: 29 }, label: '↑' },
    { button: 'l3-right', position: { x: 31, y: 40 }, label: '→' },
    { button: 'l3-down', position: { x: 23, y: 51 }, label: '↓' },
    { button: 'r3-left', position: { x: 41, y: 40 }, label: 'J' },
    { button: 'r3-up', position: { x: 49, y: 29 }, label: 'I' },
    { button: 'r3-right', position: { x: 57, y: 40 }, label: 'L' },
    { button: 'r3-down', position: { x: 49, y: 51 }, label: 'K' },
    { button: 'dpad_up', position: { x: 13, y: 10 }, label: 'T' },
    { button: 'dpad_down', position: { x: 13, y: 25 }, label: 'G' },
    { button: 'dpad_left', position: { x: 8, y: 17 }, label: 'F' },
    { button: 'dpad_right', position: { x: 19, y: 17 }, label: 'H' }
  ];

  // 加载游戏的快捷键配置
  const loadHotkeys = async () => {
    try {
      setLoading(true);
      const gameHotkeys = await GetHotkeysByGameID(gameId);
      setHotkeys(gameHotkeys);
    } catch (error) {
      console.error('Failed to load hotkeys:', error);
      toast.error(t('ps4.loadHotkeysFailed'));
    } finally {
      setLoading(false);
    }
  };

  // 组件初始化时加载数据
  useEffect(() => {
    if (gameId) {
      loadHotkeys();
    }
  }, [gameId]);

  // 查找指定按钮的映射
  const findMapping = (button: string) => {
    return hotkeys.find(hotkey => 
      hotkey.device_type === enums.DeviceType.DUALSHOCK4 && 
      hotkey.key_code === button
    );
  };

  // 处理按钮点击
  const handleButtonClick = (button: string) => {
    setCurrentMappingButton(button);
    setNewKeyCode('');
    setShowMappingDialog(true);
  };

  // 处理按键捕获
  const handleKeyDown = (e: KeyboardEvent) => {
    if (waitingForKey && currentMappingButton) {
      e.preventDefault();
      const key = e.code.replace('Key', '').toLowerCase();
      setNewKeyCode(key);
      setWaitingForKey(false);
      
      // 更新或创建映射
      saveMapping(currentMappingButton, key);
    }
  };

  // 保存映射配置
  const saveMapping = async (button: string, targetKey: string) => {
    try {
      const existingHotkey = findMapping(button);
      
      if (existingHotkey) {
        // 更新现有映射
        const updatedHotkey = new models.Hotkey({
          ...existingHotkey,
          action_params: targetKey,
          updated_at: new Date()
        });
        await UpdateHotkey(updatedHotkey);
        setHotkeys(prev => prev.map(h => 
          h.id === existingHotkey.id ? updatedHotkey : h
        ));
      } else {
        // 创建新映射
        const newHotkey = new models.Hotkey({
          id: Date.now().toString(),
          game_id: gameId,
          name: `PS4 ${button}`,
          device_type: enums.DeviceType.DUALSHOCK4,
          key_code: button,
          modifiers: [],
          action_type: enums.HotkeyActionType.CUSTOM,
          action_params: { target_key: targetKey },
          is_enabled: true,
          created_at: new Date(),
          updated_at: new Date()
        });
        const createdHotkey = await AddHotkey(newHotkey);
        setHotkeys(prev => [...prev, newHotkey]);
      }
      
      setShowMappingDialog(false);
      toast.success(t('ps4.mappingSaved', { button, targetKey }));
    } catch (error) {
      console.error('Failed to save mapping:', error);
      toast.error(t('ps4.saveMappingFailed'));
    }
  };

  // 删除映射
  const deleteMapping = async (hotkeyId: string) => {
    try {
      await DeleteHotkey(hotkeyId);
      setHotkeys(prev => prev.filter(h => h.id !== hotkeyId));
      toast.success('映射已删除');
    } catch (error) {
      console.error('Failed to delete mapping:', error);
      toast.error('删除映射失败');
    }
  };

  // 开始等待按键输入
  const startKeyCapture = () => {
    setWaitingForKey(true);
    setNewKeyCode('');
  };

  // 取消映射设置
  const cancelMapping = () => {
    setShowMappingDialog(false);
    setWaitingForKey(false);
    setCurrentMappingButton(null);
  };

  useEffect(() => {
    if (showMappingDialog) {
      window.addEventListener('keydown', handleKeyDown);
      return () => {
        window.removeEventListener('keydown', handleKeyDown);
      };
    }
  }, [showMappingDialog, waitingForKey, currentMappingButton]);

  if (loading) {
    return (
      <div className="ps4-panel flex items-center justify-center h-full">
        <div className="text-gray-500">{t('common.loading')}</div>
      </div>
    );
  }

  return (
    <div className="ps4-panel relative w-full h-full min-h-[700px]">
      {/* 手柄图片和按钮映射容器 - 固定大小并居中 */}
      <div className="absolute inset-0 flex items-center justify-center">
        <div className="relative w-[700px] h-[500px]">
          {/* 手柄背景图 - 位于按钮层下方 */}
          <img 
            src="/ds4.png" 
            alt="DS4 Controller"
            className="absolute inset-0 w-[500px] h-[312px] object-contain"
          />

          {/* 按钮映射层 - 位于图片上方 */}
          {buttonMappings.map((mapping) => {
            const hotkey = findMapping(mapping.button);
            return (
              <div key={mapping.button} className="absolute" 
                   style={{
                     left: `${mapping.position.x}%`,
                     top: `${mapping.position.y}%`,
                     transform: 'translate(-50%, -50%)'
                   }}>
                <button
                  className="w-12 h-12 rounded-full bg-blue-500 hover:bg-blue-600 transition-colors flex items-center justify-center text-white font-bold shadow-lg hover:scale-110 transform"
                  onClick={() => handleButtonClick(mapping.button)}
                  title={hotkey ? `${mapping.label} → ${hotkey.action_params || '未设置'}` : mapping.label}
                >
                  {hotkey ? hotkey.action_params?.toUpperCase() || mapping.label : mapping.label}
                </button>
                
                {/* 删除按钮 */}
                {hotkey && (
                  <button
                    className="absolute -top-2 -right-2 w-6 h-6 bg-red-500 hover:bg-red-600 rounded-full text-white text-xs flex items-center justify-center shadow-md"
                    onClick={(e) => {
                      e.stopPropagation();
                      deleteMapping(hotkey.id);
                    }}
                    title="删除映射"
                  >
                    ×
                  </button>
                )}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
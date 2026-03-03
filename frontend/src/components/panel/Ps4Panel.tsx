import { useState, useEffect } from 'react';
import { BetterButton } from '../ui/BetterButton';
import { toast } from "react-hot-toast";
import { models, enums } from '../../../wailsjs/go/models';
import { GetHotkeysByGameID, UpdateHotkey, AddHotkey, DeleteHotkey } from '../../../wailsjs/go/service/HotkeyService';

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
    { button: 'square', position: { x: 25, y: 35 }, label: '□' },
    { button: 'cross', position: { x: 35, y: 45 }, label: '✕' },
    { button: 'circle', position: { x: 45, y: 35 }, label: '○' },
    { button: 'triangle', position: { x: 35, y: 25 }, label: '△' },
    { button: 'l1', position: { x: 15, y: 15 }, label: 'L1' },
    { button: 'r1', position: { x: 75, y: 15 }, label: 'R1' },
    { button: 'l2', position: { x: 10, y: 5 }, label: 'L2' },
    { button: 'r2', position: { x: 80, y: 5 }, label: 'R2' },
    { button: 'share', position: { x: 20, y: 25 }, label: 'SHARE' },
    { button: 'options', position: { x: 70, y: 25 }, label: 'OPTIONS' },
    { button: 'ps', position: { x: 45, y: 50 }, label: 'PS' },
    { button: 'touchpad', position: { x: 40, y: 30 }, label: 'TOUCHPAD' },
    { button: 'left_stick', position: { x: 25, y: 60 }, label: 'LS' },
    { button: 'right_stick', position: { x: 65, y: 60 }, label: 'RS' },
    { button: 'dpad_up', position: { x: 20, y: 75 }, label: '↑' },
    { button: 'dpad_down', position: { x: 20, y: 85 }, label: '↓' },
    { button: 'dpad_left', position: { x: 15, y: 80 }, label: '←' },
    { button: 'dpad_right', position: { x: 25, y: 80 }, label: '→' }
  ];

  // 加载游戏的快捷键配置
  const loadHotkeys = async () => {
    try {
      setLoading(true);
      const gameHotkeys = await GetHotkeysByGameID(gameId);
      setHotkeys(gameHotkeys);
    } catch (error) {
      console.error('Failed to load hotkeys:', error);
      toast.error('加载快捷键配置失败');
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
      toast.success(`已将 ${button} 映射到 ${targetKey}`);
    } catch (error) {
      console.error('Failed to save mapping:', error);
      toast.error('保存映射配置失败');
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
        <div className="text-gray-500">加载中...</div>
      </div>
    );
  }

  return (
    <div className="ps4-panel relative w-full h-full min-h-[500px]">
      {/* 手柄背景图 */}
      <div className="absolute inset-0 flex items-center justify-center">
        <div className="relative w-80 h-64 bg-gray-800 rounded-2xl border-4 border-gray-700 shadow-xl">
          {/* 手柄主体形状 */}
          <div className="absolute inset-4 bg-gray-900 rounded-xl"></div>
          
          {/* 左右握把 */}
          <div className="absolute left-2 top-8 w-12 h-32 bg-gray-700 rounded-l-lg"></div>
          <div className="absolute right-2 top-8 w-12 h-32 bg-gray-700 rounded-r-lg"></div>
          
          {/* 中央凹槽 */}
          <div className="absolute left-1/2 top-1/2 transform -translate-x-1/2 -translate-y-1/2 w-24 h-16 bg-gray-700 rounded-full"></div>
        </div>
      </div>

      {/* 按钮映射层 */}
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

      {/* 映射设置对话框 */}
      {showMappingDialog && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-md mx-4">
            <div className="p-6">
              <div className="flex justify-between items-center mb-4">
                <h3 className="text-lg font-semibold">
                  设置 {currentMappingButton} 按键映射
                </h3>
                <button
                  onClick={cancelMapping}
                  className="text-gray-400 hover:text-gray-600"
                >
                  ×
                </button>
              </div>
              
              <div className="space-y-4">
                <p className="text-gray-600">
                  请按下您想要映射到 <strong>{currentMappingButton}</strong> 的键盘按键
                </p>
                
                <div className="p-4 bg-gray-100 rounded-lg">
                  <div className="text-center">
                    {waitingForKey ? (
                      <div className="animate-pulse">
                        <div className="text-2xl">⌨️</div>
                        <p className="mt-2 text-gray-500">正在等待按键输入...</p>
                      </div>
                    ) : (
                      <div>
                        <input
                          type="text"
                          value={newKeyCode}
                          onChange={(e) => setNewKeyCode(e.target.value)}
                          placeholder="或手动输入按键代码"
                          className="w-full px-3 py-2 border border-gray-300 rounded-md text-center focus:outline-none focus:ring-2 focus:ring-blue-500"
                        />
                        <BetterButton 
                          onClick={startKeyCapture} 
                          className="mt-2 w-full"
                          variant="secondary"
                        >
                          点击开始按键捕获
                        </BetterButton>
                      </div>
                    )}
                  </div>
                </div>

                <div className="flex justify-end space-x-2 pt-4">
                  <BetterButton onClick={cancelMapping}>
                    取消
                  </BetterButton>
                  <BetterButton 
                    onClick={() => {
                      if (newKeyCode && currentMappingButton) {
                        saveMapping(currentMappingButton, newKeyCode);
                      }
                    }}
                    disabled={!newKeyCode}
                  >
                    确认映射
                  </BetterButton>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
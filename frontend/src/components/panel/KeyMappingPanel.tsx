import { useState, useEffect } from 'react';
import { BetterButton } from '../ui/BetterButton';
import { toast } from "react-hot-toast";
import { models, enums } from '../../../wailsjs/go/models';
import { GetHotkeysByGameID, UpdateHotkey, AddHotkey, DeleteHotkey, GetGlobalHotkeys } from '../../../wailsjs/go/service/HotkeyService';
import { GetAppConfig, UpdateAppConfig } from '../../../wailsjs/go/service/ConfigService';
import i18next from "../../i18n/i18n";
const t = i18next.t;

interface Ps4PanelProps {
  gameId: string;
}

interface KeyMapping {
  button: string;
  buttonLabel: string;
  mappingLabel: string;
  mappingKey: string;
  position: { x: number; y: number };
}

interface KeyMap {
  label: string;
  keyCode: string;
}

export function KeyMappingPanel({ gameId }: Ps4PanelProps) {
  const [hotkeys, setHotkeys] = useState<models.Hotkey[]>([]);
  const [loading, setLoading] = useState(true);
  const [showMappingDialog, setShowMappingDialog] = useState(false);
  const [currentMappingButton, setCurrentMappingButton] = useState<string | null>(null);
  const [newKeyCode, setNewKeyCode] = useState('');
  const [waitingForKey, setWaitingForKey] = useState(false);
  const [selectedDeviceType, setSelectedDeviceType] = useState<enums.DeviceType | null>(null);
  const [showDeviceDropdown, setShowDeviceDropdown] = useState(false);

  const keyboardMap: KeyMap[] = [
    { keyCode: 'ctrl', label: 'Control' }

  ]

  

  // PS4手柄按键位置定义
  const ds4Mappings: KeyMapping[] = [
    { button: 'square', buttonLabel: 'Square', mappingLabel: 'A', mappingKey: 'KeyA', position: { x: 52, y: 17 } },
    { button: 'cross', buttonLabel: 'Cross', mappingLabel: 'S', mappingKey: 'KeyS', position: { x: 58, y: 25 } },
    { button: 'circle', buttonLabel: 'Circle', mappingLabel: 'D', mappingKey: 'KeyD', position: { x: 65, y: 17 } },
    { button: 'triangle', buttonLabel: 'Triangle', mappingLabel: 'W', mappingKey: 'KeyW', position: { x: 58, y: 10 } },
    { button: 'l1', buttonLabel: 'L1', mappingLabel: 'Q', mappingKey: 'KeyQ', position: { x: 15, y: -5 } },
    { button: 'r1', buttonLabel: 'R1', mappingLabel: 'E', mappingKey: 'KeyE', position: { x: 60, y: -5 } },
    { button: 'l2', buttonLabel: 'L2', mappingLabel: 'Z', mappingKey: 'KeyZ', position: { x: 10, y: -15 } },
    { button: 'r2', buttonLabel: 'R2', mappingLabel: 'C', mappingKey: 'KeyC', position: { x: 65, y: -15 } },
    { button: 'l3', buttonLabel: 'L3', mappingLabel: '1', mappingKey: 'Digit1', position: { x: 23, y: 40 } },
    { button: 'r3', buttonLabel: 'R3', mappingLabel: '3', mappingKey: 'Digit3', position: { x: 49, y: 40 } },
    { button: 'share', buttonLabel: 'Share', mappingLabel: 'V', mappingKey: 'KeyV', position: { x: 22, y: 5 } },
    { button: 'options', buttonLabel: 'Options', mappingLabel: 'M', mappingKey: 'KeyM', position: { x: 50, y: 5 } },
    { button: 'ps', buttonLabel: 'PS', mappingLabel: 'B', mappingKey: 'KeyB', position: { x: 37, y: 30 } },
    { button: 'touchpad', buttonLabel: 'Touchpad', mappingLabel: 'N', mappingKey: 'KeyN', position: { x: 37, y: 13 } },
    { button: 'l3-left', buttonLabel: 'L3 Left', mappingLabel: '←', mappingKey: 'ArrowLeft', position: { x: 15, y: 40 } },
    { button: 'l3-up', buttonLabel: 'L3 Up', mappingLabel: '↑', mappingKey: 'ArrowUp', position: { x: 23, y: 29 } },
    { button: 'l3-right', buttonLabel: 'L3 Right', mappingLabel: '→', mappingKey: 'ArrowRight', position: { x: 31, y: 40 } },
    { button: 'l3-down', buttonLabel: 'L3 Down', mappingLabel: '↓', mappingKey: 'ArrowDown', position: { x: 23, y: 51 } },
    { button: 'r3-left', buttonLabel: 'R3 Left', mappingLabel: 'J', mappingKey: 'KeyJ', position: { x: 41, y: 40 } },
    { button: 'r3-up', buttonLabel: 'R3 Up', mappingLabel: 'I', mappingKey: 'KeyI', position: { x: 49, y: 29 } },
    { button: 'r3-right', buttonLabel: 'R3 Right', mappingLabel: 'L', mappingKey: 'KeyL', position: { x: 57, y: 40 } },
    { button: 'r3-down', buttonLabel: 'R3 Down', mappingLabel: 'K', mappingKey: 'KeyK', position: { x: 49, y: 51 } },
    { button: 'dpad_up', buttonLabel: 'D-Pad Up', mappingLabel: 'T', mappingKey: 'KeyT', position: { x: 13, y: 10 } },
    { button: 'dpad_down', buttonLabel: 'D-Pad Down', mappingLabel: 'G', mappingKey: 'KeyG', position: { x: 13, y: 25 } },
    { button: 'dpad_left', buttonLabel: 'D-Pad Left', mappingLabel: 'F', mappingKey: 'KeyF', position: { x: 8, y: 17 } },
    { button: 'dpad_right', buttonLabel: 'D-Pad Right', mappingLabel: 'H', mappingKey: 'KeyH', position: { x: 19, y: 17 } }
  ];

  // Joy-Con手柄按键位置定义
  const joyconMappings: KeyMapping[] = [
    { button: 'a', buttonLabel: 'A', mappingLabel: 'A', mappingKey: 'KeyA', position: { x: 65, y: 30 } },
    { button: 'b', buttonLabel: 'B', mappingLabel: 'B', mappingKey: 'KeyB', position: { x: 70, y: 20 } },
    { button: 'x', buttonLabel: 'X', mappingLabel: 'X', mappingKey: 'KeyX', position: { x: 60, y: 20 } },
    { button: 'y', buttonLabel: 'Y', mappingLabel: 'Y', mappingKey: 'KeyY', position: { x: 65, y: 10 } },
    { button: 'l', buttonLabel: 'L', mappingLabel: 'L', mappingKey: 'KeyL', position: { x: 15, y: 15 } },
    { button: 'r', buttonLabel: 'R', mappingLabel: 'R', mappingKey: 'KeyR', position: { x: 85, y: 15 } },
    { button: 'zl', buttonLabel: 'ZL', mappingLabel: 'Q', mappingKey: 'KeyQ', position: { x: 10, y: 25 } },
    { button: 'zr', buttonLabel: 'ZR', mappingLabel: 'E', mappingKey: 'KeyE', position: { x: 90, y: 25 } },
    { button: 'minus', buttonLabel: 'Minus', mappingLabel: '-', mappingKey: 'Minus', position: { x: 25, y: 5 } },
    { button: 'plus', buttonLabel: 'Plus', mappingLabel: '+', mappingKey: 'Equal', position: { x: 75, y: 5 } },
    { button: 'home', buttonLabel: 'Home', mappingLabel: 'Home', mappingKey: 'Home', position: { x: 50, y: 5 } },
    { button: 'l-stick', buttonLabel: 'L-Stick', mappingLabel: '1', mappingKey: 'Digit1', position: { x: 25, y: 40 } },
    { button: 'r-stick', buttonLabel: 'R-Stick', mappingLabel: '3', mappingKey: 'Digit3', position: { x: 75, y: 40 } },
    { button: 'l-stick-left', buttonLabel: 'L-Stick Left', mappingLabel: '←', mappingKey: 'ArrowLeft', position: { x: 15, y: 40 } },
    { button: 'l-stick-up', buttonLabel: 'L-Stick Up', mappingLabel: '↑', mappingKey: 'ArrowUp', position: { x: 25, y: 30 } },
    { button: 'l-stick-right', buttonLabel: 'L-Stick Right', mappingLabel: '→', mappingKey: 'ArrowRight', position: { x: 35, y: 40 } },
    { button: 'l-stick-down', buttonLabel: 'L-Stick Down', mappingLabel: '↓', mappingKey: 'ArrowDown', position: { x: 25, y: 50 } },
    { button: 'r-stick-left', buttonLabel: 'R-Stick Left', mappingLabel: 'J', mappingKey: 'KeyJ', position: { x: 65, y: 40 } },
    { button: 'r-stick-up', buttonLabel: 'R-Stick Up', mappingLabel: 'I', mappingKey: 'KeyI', position: { x: 75, y: 30 } },
    { button: 'r-stick-right', buttonLabel: 'R-Stick Right', mappingLabel: 'L', mappingKey: 'KeyL', position: { x: 85, y: 40 } },
    { button: 'r-stick-down', buttonLabel: 'R-Stick Down', mappingLabel: 'K', mappingKey: 'KeyK', position: { x: 75, y: 50 } }
  ];

  // XInput手柄按键位置定义
  const xinputMappings: KeyMapping[] = [
    { button: 'a', buttonLabel: 'A', mappingLabel: 'A', mappingKey: 'KeyA', position: { x: 65, y: 30 } },
    { button: 'b', buttonLabel: 'B', mappingLabel: 'B', mappingKey: 'KeyB', position: { x: 70, y: 20 } },
    { button: 'x', buttonLabel: 'X', mappingLabel: 'X', mappingKey: 'KeyX', position: { x: 60, y: 20 } },
    { button: 'y', buttonLabel: 'Y', mappingLabel: 'Y', mappingKey: 'KeyY', position: { x: 65, y: 10 } },
    { button: 'lb', buttonLabel: 'LB', mappingLabel: 'Q', mappingKey: 'KeyQ', position: { x: 15, y: -5 } },
    { button: 'rb', buttonLabel: 'RB', mappingLabel: 'E', mappingKey: 'KeyE', position: { x: 85, y: -5 } },
    { button: 'lt', buttonLabel: 'LT', mappingLabel: 'Z', mappingKey: 'KeyZ', position: { x: 15, y: 5 } },
    { button: 'rt', buttonLabel: 'RT', mappingLabel: 'C', mappingKey: 'KeyC', position: { x: 85, y: 5 } },
    { button: 'ls', buttonLabel: 'LS', mappingLabel: '1', mappingKey: 'Digit1', position: { x: 25, y: 40 } },
    { button: 'rs', buttonLabel: 'RS', mappingLabel: '3', mappingKey: 'Digit3', position: { x: 75, y: 40 } },
    { button: 'back', buttonLabel: 'Back', mappingLabel: 'V', mappingKey: 'KeyV', position: { x: 35, y: 15 } },
    { button: 'start', buttonLabel: 'Start', mappingLabel: 'M', mappingKey: 'KeyM', position: { x: 65, y: 15 } },
    { button: 'ls-left', buttonLabel: 'LS Left', mappingLabel: '←', mappingKey: 'ArrowLeft', position: { x: 15, y: 40 } },
    { button: 'ls-up', buttonLabel: 'LS Up', mappingLabel: '↑', mappingKey: 'ArrowUp', position: { x: 25, y: 30 } },
    { button: 'ls-right', buttonLabel: 'LS Right', mappingLabel: '→', mappingKey: 'ArrowRight', position: { x: 35, y: 40 } },
    { button: 'ls-down', buttonLabel: 'LS Down', mappingLabel: '↓', mappingKey: 'ArrowDown', position: { x: 25, y: 50 } },
    { button: 'rs-left', buttonLabel: 'RS Left', mappingLabel: 'J', mappingKey: 'KeyJ', position: { x: 65, y: 40 } },
    { button: 'rs-up', buttonLabel: 'RS Up', mappingLabel: 'I', mappingKey: 'KeyI', position: { x: 75, y: 30 } },
    { button: 'rs-right', buttonLabel: 'RS Right', mappingLabel: 'L', mappingKey: 'KeyL', position: { x: 85, y: 40 } },
    { button: 'rs-down', buttonLabel: 'RS Down', mappingLabel: 'K', mappingKey: 'KeyK', position: { x: 75, y: 50 } },
    { button: 'dpad_up', buttonLabel: 'D-Pad Up', mappingLabel: 'T', mappingKey: 'KeyT', position: { x: 15, y: 20 } },
    { button: 'dpad_down', buttonLabel: 'D-Pad Down', mappingLabel: 'G', mappingKey: 'KeyG', position: { x: 15, y: 30 } },
    { button: 'dpad_left', buttonLabel: 'D-Pad Left', mappingLabel: 'F', mappingKey: 'KeyF', position: { x: 10, y: 25 } },
    { button: 'dpad_right', buttonLabel: 'D-Pad Right', mappingLabel: 'H', mappingKey: 'KeyH', position: { x: 20, y: 25 } }
  ];

  // 设备类型选项（过滤掉键盘）
  const deviceTypes = [
    { type: null as any, name: "关闭手柄映射", mapping: [] as KeyMapping[], img: "", imgWidth: 0, imgHeight: 0 },
    { type: enums.DeviceType.DUALSENSE, name: "DualSense", mapping: ds4Mappings, img: "/ds4.png", imgWidth: 500, imgHeight: 312 },
    { type: enums.DeviceType.DUALSHOCK4, name: "DualShock 4", mapping: ds4Mappings, img: "/ds4.png", imgWidth: 500, imgHeight: 312 },
    { type: enums.DeviceType.JOYCON, name: "Joy-Con", mapping: joyconMappings, img: "/joycon.jpg", imgWidth: 500, imgHeight: 500 },
    { type: enums.DeviceType.XINPUT, name: "XInput", mapping: xinputMappings, img: "/xinput.png", imgWidth: 500, imgHeight: 500 }
  ];

  // 加载游戏的快捷键配置
  const loadHotkeys = async () => {
    try {
      setLoading(true);
      // 加载游戏特定的按键映射
      const gameHotkeys = await GetHotkeysByGameID(gameId);
      // 加载全局按键映射
      const globalHotkeys = await GetGlobalHotkeys();
      
      // 合并映射，游戏映射优先级高于全局
      const mergedHotkeys = [...globalHotkeys];
      const currentDeviceType = selectedDeviceType || enums.DeviceType.DUALSHOCK4;
      
      // 用游戏特定映射覆盖全局映射
      gameHotkeys.forEach(gameHotkey => {
        const index = mergedHotkeys.findIndex(
          hotkey => hotkey.device_type === currentDeviceType && 
                    hotkey.key_code === gameHotkey.key_code
        );
        if (index !== -1) {
          mergedHotkeys[index] = gameHotkey;
        } else {
          mergedHotkeys.push(gameHotkey);
        }
      });
      
      setHotkeys(mergedHotkeys);
    } catch (error) {
      console.error('Failed to load hotkeys:', error);
      toast.error(t('ps4.loadHotkeysFailed'));
    } finally {
      setLoading(false);
    }
  };

  // 从配置中加载手柄类型
  const loadJoystickConfig = async () => {
    try {
      console.log('Loading joystick config...');
      const config = await GetAppConfig();
      console.log('Loaded config:', config);
      console.log('Joystick type from config:', config.joystick_type);
      if (config.joystick_type) {
        // 查找对应的设备类型
        console.log('Device types:', deviceTypes);
        const device = deviceTypes.find(d => {
          if (d.type === null) return false;
          console.log('Comparing:', d.type, 'vs', config.joystick_type);
          // 直接比较枚举值和配置值
          return d.type === config.joystick_type;
        });
        console.log('Found device:', device);
        if (device) {
          setSelectedDeviceType(device.type);
          console.log('Set selected device type:', device.type);
        } else {
          setSelectedDeviceType(null); // 关闭手柄映射
          console.log('Set selected device type to null');
        }
      } else {
        setSelectedDeviceType(null); // 关闭手柄映射
        console.log('No joystick type in config, set to null');
      }
    } catch (error) {
      console.error('Failed to load joystick config:', error);
    } finally {
      // 无论是否成功加载配置，都设置loading为false
      setLoading(false);
    }
  };

  // 保存手柄类型到配置
  const saveJoystickConfig = async (deviceType: enums.DeviceType | null) => {
    try {
      console.log('Saving joystick config with deviceType:', deviceType);
      const config = await GetAppConfig();
      console.log('Config before update:', config);
      console.log('Updating config with joystick_type:', deviceType?.toString());
      config.joystick_type = deviceType?.toString() || '';
      console.log('Config after update:', config);
      await UpdateAppConfig(config);
      console.log('Joystick config saved successfully');
    } catch (error) {
      console.error('Failed to save joystick config:', error);
    }
  };

  // 组件初始化时加载数据
  useEffect(() => {
    loadJoystickConfig();
  }, []);

  // 设备类型变化时保存配置
  // useEffect(() => {
  //   saveJoystickConfig(selectedDeviceType);
  // }, [selectedDeviceType]);

  // 游戏ID或设备类型变化时加载热键
  useEffect(() => {
    if (gameId && selectedDeviceType) {
      loadHotkeys();
    }
  }, [gameId, selectedDeviceType]);

  // 查找指定按钮的映射
  const findMapping = (button: string) => {
    const currentDeviceType = selectedDeviceType || enums.DeviceType.DUALSHOCK4;
    return hotkeys.find(hotkey => 
      hotkey.device_type === currentDeviceType && 
      hotkey.key_code === button
    );
  };

  // 处理按钮点击
  const handleButtonClick = (button: string) => {
    setCurrentMappingButton(button);
    setNewKeyCode('');
    setShowMappingDialog(true);
    setWaitingForKey(true);
  };

  // 键盘事件处理
  useEffect(() => {
    if (waitingForKey) {
      const handleKeyDown = (e: KeyboardEvent) => {
        e.preventDefault();
        const keyCode = e.key;
        console.log('Key pressed:', keyCode);
        setNewKeyCode(keyCode);
        if (currentMappingButton) {
          console.log('Updating mapping for button:', currentMappingButton, 'with key:', keyCode);
          updateMapping(currentMappingButton, keyCode);
        }
        setWaitingForKey(false);
        setShowMappingDialog(false);
      };

      window.addEventListener('keydown', handleKeyDown);
      return () => window.removeEventListener('keydown', handleKeyDown);
    }
  }, [waitingForKey, currentMappingButton]);

  // 更新映射配置（仅更新状态，不保存到数据库）
  const updateMapping = (button: string, targetKey: string) => {
    try {
      console.log('updateMapping called with button:', button, 'targetKey:', targetKey);
      const existingHotkey = findMapping(button);
      console.log('Existing hotkey:', existingHotkey);
      
      if (existingHotkey) {
        // 更新现有映射
        var foundMapping = keyboardMap.find((mapping) => mapping.label === targetKey)
        const updatedHotkey = new models.Hotkey({
          ...existingHotkey,
          name: targetKey,
          action_params: foundMapping?.keyCode ?? targetKey,
          action_type: enums.HotkeyActionType.CUSTOM,
          device_type: selectedDeviceType || enums.DeviceType.DUALSHOCK4,
          updated_at: new Date().toISOString()
        });
        console.log('Updated hotkey:', updatedHotkey);
        setHotkeys(prev => {
          const newHotkeys = prev.map(h => 
            h.id === existingHotkey.id ? updatedHotkey : h
          );
          console.log('New hotkeys after update:', newHotkeys);
          return newHotkeys;
        });
      } else {
        // 创建新映射
        const newHotkey = new models.Hotkey({
          id: Date.now().toString(),
          game_id: gameId,
          name: targetKey,
          device_type: selectedDeviceType || enums.DeviceType.DUALSHOCK4,
          key_code: button,
          modifiers: [],
          action_type: enums.HotkeyActionType.CUSTOM,
          action_params: targetKey,
          is_enabled: true,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString()
        });
        console.log('New hotkey to add:', newHotkey);
        setHotkeys(prev => {
          const newHotkeys = [...prev, newHotkey];
          console.log('New hotkeys after add:', newHotkeys);
          return newHotkeys;
        });
      }
      
      toast.success(t('ps4.mappingSaved', { button, targetKey }));
    } catch (error) {
      console.error('Failed to update mapping:', error);
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

  // 取消映射设置
  const cancelMapping = () => {
    setShowMappingDialog(false);
    setWaitingForKey(false);
    setCurrentMappingButton(null);
  };

  // 保存所有映射
  const saveAllMappings = async () => {
    try {
      saveJoystickConfig(selectedDeviceType);
      if (selectedDeviceType === null) { 
        return
      }
      // 1. 获取所有游戏特定的映射
      const gameHotkeys = await GetHotkeysByGameID(gameId);
      
      // 2. 删除所有游戏特定的映射
      for (const hotkey of gameHotkeys) {
        await DeleteHotkey(hotkey.id);
      }
      
      // 3. 获取全局映射
      const globalHotkeys = await GetGlobalHotkeys();
      
      // 4. 处理要添加的映射
      const currentDeviceType = selectedDeviceType || enums.DeviceType.DUALSHOCK4;
      const mappingsToAdd = hotkeys.filter(hotkey => {
        if (hotkey.device_type !== currentDeviceType) return false;
        return true;
      });
      
      // 5. 添加新的映射
      for (const mapping of mappingsToAdd) {
        // 检查全局是否有对应按键的设定
        const globalHotkey = globalHotkeys.find(
          gk => gk.device_type === currentDeviceType && 
                gk.key_code === mapping.key_code
        );
        
        if (!globalHotkey) {
          // 如果全局没有，保存为全局映射
          const globalMapping = new models.Hotkey({
            ...mapping,
            id: Date.now().toString(),
            game_id: 'global',
            device_type: currentDeviceType
          });
          await AddHotkey(globalMapping);
        } else if (globalHotkey.action_params !== mapping.action_params) {
          // 如果全局有且不同，保存为游戏特定映射
          const gameMapping = new models.Hotkey({
            ...mapping,
            id: Date.now().toString(),
            game_id: gameId,
            device_type: currentDeviceType
          });
          await AddHotkey(gameMapping);
        }
        // 如果全局有且相同，不需要保存
      }
      
      toast.success('所有映射已保存');
    } catch (error) {
      console.error('Failed to save all mappings:', error);
      toast.error('保存映射失败');
    }
  };

  if (loading) {
    return (
      <div className="ps4-panel flex items-center justify-center h-full">
        <div className="text-gray-500">{t('common.loading')}</div>
      </div>
    );
  }

  // 获取当前选中设备的信息
  const currentDevice = deviceTypes.find(device => device.type === selectedDeviceType);

  return (
    <div className="ps4-panel relative w-full h-full min-h-[700px]">
      {/* 手柄图片和按钮映射容器 - 动态显示 */}
      {currentDevice && (
        <div className="absolute inset-0 flex items-center justify-center">
          <div className="relative w-[700px] h-[500px]">
            {/* 手柄背景图 - 位于按钮层下方 */}
            <img 
              src={currentDevice.img} 
              alt={`${currentDevice.name} Controller`}
              className="absolute inset-0" 
              style={{ 
                width: `${currentDevice.imgWidth}px`, 
                height: `${currentDevice.imgHeight}px`,
                objectFit: 'contain'
              }}
            />

            {/* 按钮映射层 - 位于图片上方 */}
            {currentDevice.mapping.map((mapping) => {
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
                    title={hotkey ? `${mapping.buttonLabel} → ${hotkey.action_type === enums.HotkeyActionType.SCREENSHOT ? '截图' : (hotkey.name || '未设置')}` : mapping.buttonLabel}
                  >
                    {hotkey ? (hotkey.action_type === enums.HotkeyActionType.SCREENSHOT ? '📷' : (hotkey.name || mapping.mappingLabel)) : mapping.mappingLabel}
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
      )}
      
      {/* 保存按钮和设备选择 */}
      <div className="absolute top-4 left-4 flex flex-col gap-2">
        <BetterButton 
          onClick={saveAllMappings}
          icon="i-mdi-content-save"
          className="px-6 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg shadow-md transition-colors"
        >
          保存所有映射
        </BetterButton>
        
        {/* 手柄选择 */}
        <div className="relative">
          <div className="text-sm font-medium text-gray-700 mb-1">手柄选择</div>
          <button
            onClick={() => setShowDeviceDropdown(!showDeviceDropdown)}
            className="px-6 py-2 bg-gray-100 hover:bg-gray-200 text-gray-800 rounded-lg shadow-md transition-colors w-full flex justify-between items-center"
          >
            <span>{deviceTypes.find(d => d.type === selectedDeviceType)?.name || "请选择手柄"}</span>
            <span className="i-mdi-chevron-down text-sm"></span>
          </button>
          
          {/* 下拉菜单 */}
          {showDeviceDropdown && (
            <div className="absolute top-full left-0 mt-1 w-full bg-white rounded-lg shadow-lg border border-gray-200 z-10">
              {deviceTypes.map(device => (
                <button
                  key={device.type || 'disabled'}
                  onClick={() => {
                    setSelectedDeviceType(device.type);
                    setShowDeviceDropdown(false);
                  }}
                  className={`w-full text-left px-4 py-2 hover:bg-gray-100 transition-colors ${
                    selectedDeviceType === device.type ? "bg-blue-50 text-blue-700" : ""
                  }`}
                >
                  {device.name}
                </button>
              ))}
            </div>
          )}
        </div>
      </div>
      
      {/* 映射设置弹窗 */}
      {showMappingDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
          <div className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-brand-800 border border-brand-200 dark:border-brand-700">
            <div className="flex items-start gap-4 mb-6">
              <div className="p-2 rounded-full bg-brand-100 text-brand-600 dark:bg-brand-900/30 dark:text-brand-400">
                <div className="i-mdi-controller-playstation text-2xl" />
              </div>
              <div className="flex-1">
                <h3 className="text-xl font-bold text-brand-900 dark:text-white mb-2">
                  设置按键映射
                </h3>
                <p className="text-brand-600 dark:text-brand-400 text-sm leading-relaxed">
                  请按下手柄上的按键来设置映射
                </p>
              </div>
            </div>
            
            <div className="mb-6">
              <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-2">
                当前按钮
              </label>
              <div className="p-3 bg-brand-50 border border-brand-200 rounded-lg text-brand-900 dark:bg-brand-900/50 dark:border-brand-700 dark:text-white">
                {currentDevice?.mapping.find(m => m.button === currentMappingButton)?.buttonLabel || currentMappingButton}
              </div>
            </div>
            
            <div className="mb-6">
              <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-2">
                监听状态
              </label>
              <div className="p-4 bg-brand-50 border border-brand-200 rounded-lg text-brand-900 dark:bg-brand-900/50 dark:border-brand-700 dark:text-white text-center">
                {waitingForKey ? "正在监听...请按下手柄按键" : "准备就绪"}
              </div>
            </div>
            
            <div className="flex justify-end">
              <button
                onClick={cancelMapping}
                className="px-4 py-2 text-sm font-medium text-brand-700 hover:bg-brand-100 rounded-lg dark:text-brand-300 dark:hover:bg-brand-700 transition-colors"
              >
                取消
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
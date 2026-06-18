import { useState, useEffect, useRef, useCallback } from 'react';
import { toast } from "react-hot-toast";
import { models, enums } from '../../../wailsjs/go/models';
import { GetHotkeysByGameID, UpdateHotkey, AddHotkey, DeleteHotkey, GetGlobalHotkeys } from '../../../wailsjs/go/service/HotkeyService';
import { GetAppConfig, UpdateAppConfig } from '../../../wailsjs/go/service/ConfigService';
import i18next from "../../i18n/i18n";
import { TouchMappingPanel, TouchMappingPanelRef } from './TouchMappingPanel';
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
  const touchMappingRef = useRef<TouchMappingPanelRef>(null);
  const [editMode, setEditMode] = useState(false);

  // 键盘映射
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
    { button: 'l3-left', buttonLabel: 'L3 Left', mappingLabel: '←', mappingKey: 'LEFT', position: { x: 15, y: 40 } },
    { button: 'l3-up', buttonLabel: 'L3 Up', mappingLabel: '↑', mappingKey: 'UP', position: { x: 23, y: 29 } },
    { button: 'l3-right', buttonLabel: 'L3 Right', mappingLabel: '→', mappingKey: 'RIGHT', position: { x: 31, y: 40 } },
    { button: 'l3-down', buttonLabel: 'L3 Down', mappingLabel: '↓', mappingKey: 'DOWN', position: { x: 23, y: 51 } },
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
    { button: 'a', buttonLabel: 'A', mappingLabel: 'D', mappingKey: 'KeyD', position: { x: 58, y: 40 } },
    { button: 'b', buttonLabel: 'B', mappingLabel: 'S', mappingKey: 'KeyS', position: { x: 53, y: 48 } },
    { button: 'x', buttonLabel: 'X', mappingLabel: 'W', mappingKey: 'KeyW', position: { x: 53, y: 32 } },
    { button: 'y', buttonLabel: 'Y', mappingLabel: 'A', mappingKey: 'KeyA', position: { x: 48, y: 40 } },
    { button: 'l', buttonLabel: 'L', mappingLabel: 'Q', mappingKey: 'KeyQ', position: { x: 15, y: 15 } },
    { button: 'r', buttonLabel: 'R', mappingLabel: 'E', mappingKey: 'KeyE', position: { x: 55, y: 15 } },
    { button: 'zl', buttonLabel: 'ZL', mappingLabel: 'Z', mappingKey: 'KeyZ', position: { x: 10, y: 5 } },
    { button: 'zr', buttonLabel: 'ZR', mappingLabel: 'C', mappingKey: 'KeyC', position: { x: 60, y: 5 } },
    { button: 'minus', buttonLabel: 'Minus', mappingLabel: 'V', mappingKey: 'KeyV', position: { x: 25, y: 25 } },
    { button: 'plus', buttonLabel: 'Plus', mappingLabel: 'M', mappingKey: 'KeyM', position: { x: 44, y: 25 } },
    { button: 'home', buttonLabel: 'Home', mappingLabel: 'B', mappingKey: 'KeyB', position: { x: 40, y: 42 } },
    { button: 'l-stick', buttonLabel: 'L-Stick', mappingLabel: '1', mappingKey: 'Digit1', position: { x: 17, y: 40 } },
    { button: 'r-stick', buttonLabel: 'R-Stick', mappingLabel: '3', mappingKey: 'Digit3', position: { x: 45, y: 59 } },
    { button: 'l-stick-left', buttonLabel: 'L-Stick Left', mappingLabel: '←', mappingKey: 'ArrowLeft', position: { x: 7, y: 40 } },
    { button: 'l-stick-up', buttonLabel: 'L-Stick Up', mappingLabel: '↑', mappingKey: 'ArrowUp', position: { x: 17, y: 30 } },
    { button: 'l-stick-right', buttonLabel: 'L-Stick Right', mappingLabel: '→', mappingKey: 'ArrowRight', position: { x: 25, y: 40 } },
    { button: 'l-stick-down', buttonLabel: 'L-Stick Down', mappingLabel: '↓', mappingKey: 'ArrowDown', position: { x: 17, y: 50 } },
    { button: 'r-stick-left', buttonLabel: 'R-Stick Left', mappingLabel: 'J', mappingKey: 'KeyJ', position: { x: 38, y: 60 } },
    { button: 'r-stick-up', buttonLabel: 'R-Stick Up', mappingLabel: 'I', mappingKey: 'KeyI', position: { x: 45, y: 50 } },
    { button: 'r-stick-right', buttonLabel: 'R-Stick Right', mappingLabel: 'L', mappingKey: 'KeyL', position: { x: 52, y: 60 } },
    { button: 'r-stick-down', buttonLabel: 'R-Stick Down', mappingLabel: 'K', mappingKey: 'KeyK', position: { x: 45, y: 70 } },
    { button: 'dpad_up', buttonLabel: 'D-Pad Up', mappingLabel: 'T', mappingKey: 'KeyT', position: { x: 26, y: 53 } },
    { button: 'dpad_down', buttonLabel: 'D-Pad Down', mappingLabel: 'G', mappingKey: 'KeyG', position: { x: 26, y: 68 } },
    { button: 'dpad_left', buttonLabel: 'D-Pad Left', mappingLabel: 'F', mappingKey: 'KeyF', position: { x: 21, y: 60 } },
    { button: 'dpad_right', buttonLabel: 'D-Pad Right', mappingLabel: 'H', mappingKey: 'KeyH', position: { x: 31, y: 60 } }
  ];

  // XInput手柄按键位置定义
  const xinputMappings: KeyMapping[] = [
    { button: 'a', buttonLabel: 'A', mappingLabel: 'S', mappingKey: 'KeyS', position: { x: 55, y: 43 } },
    { button: 'b', buttonLabel: 'B', mappingLabel: 'D', mappingKey: 'KeyD', position: { x: 60, y: 35 } },
    { button: 'x', buttonLabel: 'X', mappingLabel: 'A', mappingKey: 'KeyA', position: { x: 50, y: 35 } },
    { button: 'y', buttonLabel: 'Y', mappingLabel: 'W', mappingKey: 'KeyW', position: { x: 55, y: 27 } },
    { button: 'lb', buttonLabel: 'LB', mappingLabel: 'Q', mappingKey: 'KeyQ', position: { x: 15, y: 12 } },
    { button: 'rb', buttonLabel: 'RB', mappingLabel: 'E', mappingKey: 'KeyE', position: { x: 55, y: 12 } },
    { button: 'lt', buttonLabel: 'LT', mappingLabel: 'Z', mappingKey: 'KeyZ', position: { x: 15, y: 2 } },
    { button: 'rt', buttonLabel: 'RT', mappingLabel: 'C', mappingKey: 'KeyC', position: { x: 55, y: 2 } },
    { button: 'ls', buttonLabel: 'LS', mappingLabel: '1', mappingKey: 'Digit1', position: { x: 17, y: 32 } },
    { button: 'rs', buttonLabel: 'RS', mappingLabel: '3', mappingKey: 'Digit3', position: { x: 45, y: 53 } },
    { button: 'back', buttonLabel: 'Back', mappingLabel: 'V', mappingKey: 'KeyV', position: { x: 30, y: 30 } },
    { button: 'start', buttonLabel: 'Start', mappingLabel: 'M', mappingKey: 'KeyM', position: { x: 42, y: 30 } },
    { button: 'ls-left', buttonLabel: 'LS Left', mappingLabel: '←', mappingKey: 'ArrowLeft', position: { x: 9, y: 33 } },
    { button: 'ls-up', buttonLabel: 'LS Up', mappingLabel: '↑', mappingKey: 'ArrowUp', position: { x: 17, y: 23 } },
    { button: 'ls-right', buttonLabel: 'LS Right', mappingLabel: '→', mappingKey: 'ArrowRight', position: { x: 23, y: 33 } },
    { button: 'ls-down', buttonLabel: 'LS Down', mappingLabel: '↓', mappingKey: 'ArrowDown', position: { x: 17, y: 43 } },
    { button: 'rs-left', buttonLabel: 'RS Left', mappingLabel: 'J', mappingKey: 'KeyJ', position: { x: 37, y: 53 } },
    { button: 'rs-up', buttonLabel: 'RS Up', mappingLabel: 'I', mappingKey: 'KeyI', position: { x: 45, y: 43 } },
    { button: 'rs-right', buttonLabel: 'RS Right', mappingLabel: 'L', mappingKey: 'KeyL', position: { x: 53, y: 53 } },
    { button: 'rs-down', buttonLabel: 'RS Down', mappingLabel: 'K', mappingKey: 'KeyK', position: { x: 45, y: 63 } },
    { button: 'dpad_up', buttonLabel: 'D-Pad Up', mappingLabel: 'T', mappingKey: 'KeyT', position: { x: 26, y: 53 } },
    { button: 'dpad_down', buttonLabel: 'D-Pad Down', mappingLabel: 'G', mappingKey: 'KeyG', position: { x: 26, y: 68 } },
    { button: 'dpad_left', buttonLabel: 'D-Pad Left', mappingLabel: 'F', mappingKey: 'KeyF', position: { x: 21, y: 60 } },
    { button: 'dpad_right', buttonLabel: 'D-Pad Right', mappingLabel: 'H', mappingKey: 'KeyH', position: { x: 31, y: 60 } }
  ];

  // 设备类型选项（过滤掉键盘）
  const deviceTypes = [
    { type: null as any, name: "关闭手柄映射", mapping: [] as KeyMapping[], img: "", imgWidth: 0, imgHeight: 0 },
    { type: enums.DeviceType.DUALSENSE, name: "DualSense", mapping: ds4Mappings, img: "/ds4.png", imgWidth: 500, imgHeight: 312 },
    { type: enums.DeviceType.DUALSHOCK4, name: "DualShock 4", mapping: ds4Mappings, img: "/ds4.png", imgWidth: 500, imgHeight: 312 },
    { type: enums.DeviceType.JOYCON, name: "Joy-Con", mapping: joyconMappings, img: "/joycon.jpg", imgWidth: 500, imgHeight: 500 },
    { type: enums.DeviceType.XINPUT, name: "XInput", mapping: xinputMappings, img: "/xinput.png", imgWidth: 500, imgHeight: 500 },
    { type: enums.DeviceType.TOUCH, name: t('ps4.touchDeviceName'), mapping: [] as KeyMapping[], img: "", imgWidth: 0, imgHeight: 0 },
  ];

  // 获取当前选中设备的信息
  const isTouchDevice = selectedDeviceType === enums.DeviceType.TOUCH;

  // 监听触摸面板的编辑模式变化
  useEffect(() => {
    if (isTouchDevice && touchMappingRef.current) {
      setEditMode(touchMappingRef.current.editMode);
    }
  }, [isTouchDevice, selectedDeviceType]);

  // 加载游戏的快捷键配置
  const loadHotkeys = async () => {
    try {
      setLoading(true);
      // 加载游戏特定的按键映射
      const gameHotkeys = await GetHotkeysByGameID(gameId);
      // 加载全局按键映射
      // const globalHotkeys = await GetGlobalHotkeys();
      
      // 合并映射，游戏映射优先级高于全局
      // const mergedHotkeys = [...globalHotkeys];
      
      const mergedHotkeys: models.Hotkey[] = [];
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
          modifiers: '',
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

  // 从数据库重新加载按键列表
  const handleRefresh = async () => {
    await loadHotkeys();
  };

  // 清除当前设备类型的所有按键（仅内存，不影响数据库）
  const handleClear = useCallback(async () => {
    if (isTouchDevice) {
      touchMappingRef.current?.clearAll();
    } else {
      const currentDeviceType = selectedDeviceType || enums.DeviceType.DUALSHOCK4;
      setHotkeys((prev) =>
        prev.filter((h) => h.device_type !== currentDeviceType)
      );
    }
    toast.success('已清除所有按键映射，请点击保存按钮确认更改');
  }, [isTouchDevice, selectedDeviceType]);

  // 载入默认配置（覆盖内存中原有的按键列表）
  const handleLoadDefaults = async () => {
    try {
      setLoading(true);
      const currentDeviceType = selectedDeviceType || enums.DeviceType.DUALSHOCK4;
      const device = deviceTypes.find((d) => d.type === currentDeviceType);
      const defaultMappings = device?.mapping || [];

      if (defaultMappings.length === 0) {
        toast.error(t('keyMapping.noDefaults'));
        return;
      }

      const newHotkeys = defaultMappings.map((m) =>
        new models.Hotkey({
          id: crypto.randomUUID(),
          game_id: gameId,
          name: m.mappingLabel,
          device_type: currentDeviceType,
          key_code: m.button,
          modifiers: '',
          action_type: enums.HotkeyActionType.CUSTOM,
          action_params: m.mappingKey,
          is_enabled: true,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        })
      );

      setHotkeys(newHotkeys);
      toast.success(t('keyMapping.defaultsLoaded', { count: newHotkeys.length }));
    } catch (err) {
      console.error('载入默认失败:', err);
      toast.error(t('keyMapping.loadFailed'));
    } finally {
      setLoading(false);
    }
  };

  // 载入全局配置（覆盖内存中原有的按键列表）
  const handleLoadGlobal = async () => {
    try {
      setLoading(true);
      const globalHotkeys = await GetGlobalHotkeys();
      const currentDeviceType = selectedDeviceType || enums.DeviceType.DUALSHOCK4;
      const globalMappings = globalHotkeys.filter(
        (h) => h.device_type === currentDeviceType
      );

      if (globalMappings.length === 0) {
        toast.error(t('ps4.noGlobalMappings'));
        return;
      }

      // 使用新的 ID 创建新的映射对象
      const newHotkeys = globalMappings.map((h) =>
        new models.Hotkey({
          ...h,
          id: crypto.randomUUID(),
          game_id: gameId,
        })
      );

      setHotkeys(newHotkeys);
      toast.success(t('ps4.globalLoaded', { count: newHotkeys.length }));
    } catch (err) {
      console.error('载入全局失败:', err);
      toast.error(t('ps4.loadFailed'));
    } finally {
      setLoading(false);
    }
  };

  // 保存所有映射
  const saveAllMappings = async () => {
    try {
      saveJoystickConfig(selectedDeviceType);
      if (selectedDeviceType === null) { 
        return
      }
      
      const currentDeviceType = selectedDeviceType || enums.DeviceType.DUALSHOCK4;
      
      // 1. 获取所有游戏特定的映射
      const gameHotkeys = await GetHotkeysByGameID(gameId);
      
      // 2. 只删除同gameId和同设备类型的映射
      const deviceGameHotkeys = gameHotkeys.filter(
        hotkey => hotkey.device_type === currentDeviceType
      );
      for (const hotkey of deviceGameHotkeys) {
        await DeleteHotkey(hotkey.id);
      }
      
      // 3. 获取全局映射
      const globalHotkeys = await GetGlobalHotkeys();
      
      // 4. 处理要添加的映射（只处理当前设备类型）
      const mappingsToAdd = hotkeys.filter(hotkey => {
        return hotkey.device_type === currentDeviceType;
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
            id: crypto.randomUUID(),
            game_id: 'global',
            device_type: currentDeviceType
          });
          await AddHotkey(globalMapping);
        } else if (globalHotkey.action_params !== mapping.action_params) {
          // 如果全局有且不同，保存为游戏特定映射
          const gameMapping = new models.Hotkey({
            ...mapping,
            id: crypto.randomUUID(),
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

  // 统一的按钮样式
  const buttonClass = "px-4 py-2 rounded-lg shadow-md transition-colors font-medium text-sm";

  return (
    <div className="ps4-panel flex flex-col w-full h-full min-h-[700px]">
      {/* 顶部标题栏 */}
      <div className="px-6 pt-6 pb-4 flex items-center justify-between gap-4 border-b border-gray-200 dark:border-brand-700">
        <h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
          {t('keyMapping.title')}
        </h2>

        {/* 设备选择 */}
        <div className="relative">
          <button
            onClick={() => setShowDeviceDropdown(!showDeviceDropdown)}
            className={`px-4 py-2 bg-gray-100 hover:bg-gray-200 text-gray-800 rounded-lg shadow-sm transition-colors flex items-center gap-2 dark:bg-brand-700 dark:text-gray-200 dark:hover:bg-brand-600 ${buttonClass}`}
          >
            <span>{deviceTypes.find(d => d.type === selectedDeviceType)?.name || t('keyMapping.deviceSelectorPlaceholder')}</span>
            <span className="i-mdi-chevron-down text-sm"></span>
          </button>

          {showDeviceDropdown && (
            <div className="absolute top-full right-0 mt-2 w-48 bg-white rounded-lg shadow-lg border border-gray-200 z-20 dark:bg-brand-800 dark:border-brand-700">
              {deviceTypes.map(device => (
                <button
                  key={device.type || 'disabled'}
                  onClick={() => {
                    setSelectedDeviceType(device.type);
                    setShowDeviceDropdown(false);
                  }}
                  className={`w-full text-left px-4 py-2 hover:bg-gray-100 transition-colors dark:hover:bg-brand-700 ${
                    selectedDeviceType === device.type ? "bg-blue-50 text-blue-700 dark:bg-brand-700/50 dark:text-blue-300" : "text-gray-800 dark:text-gray-200"
                  }`}
                >
                  {device.name}
                </button>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* 统一操作按钮栏 */}
      <div className="px-6 py-4 flex flex-wrap items-center gap-3 border-b border-gray-200 dark:border-brand-700">
        {selectedDeviceType !== null && isTouchDevice ? (
          // 触摸设备按钮
          <>
            {!editMode && (
              <>
                <button
                  onClick={() => touchMappingRef.current?.openAddDialog()}
                  className={`${buttonClass} bg-blue-600 hover:bg-blue-700 text-white`}
                >
                  <span className="i-mdi-plus mr-2"></span>
                  {t('touchMapping.addButton')}
                </button>
                <button
                  onClick={() => touchMappingRef.current?.loadDefaults()}
                  className={`${buttonClass} bg-indigo-600 hover:bg-indigo-700 text-white`}
                >
                  <span className="i-mdi-restore mr-2"></span>
                  {t('touchMapping.loadDefaults')}
                </button>
                {gameId !== 'global' && (
                  <button
                    onClick={() => touchMappingRef.current?.loadGlobal()}
                    className={`${buttonClass} bg-teal-600 hover:bg-teal-700 text-white`}
                  >
                    <span className="i-mdi-upload mr-2"></span>
                    {t('touchMapping.loadGlobal')}
                  </button>
                )}
                <button
                  onClick={() => touchMappingRef.current?.refresh()}
                  className={`${buttonClass} bg-blue-600 hover:bg-blue-700 text-white`}
                >
                  <span className="i-mdi-refresh mr-2"></span>
                  {t('touchMapping.refresh')}
                </button>
                <button
                  onClick={handleClear}
                  className={`${buttonClass} bg-red-600 hover:bg-red-700 text-white`}
                >
                  <span className="i-mdi-delete-sweep mr-2"></span>
                  清除
                </button>
              </>
            )}
            {/* 编辑模式按钮 */}
            {!editMode ? (
              <button
                onClick={() => {
                  touchMappingRef.current?.startEditMode();
                  setEditMode(true);
                }}
                className={`${buttonClass} bg-yellow-600 hover:bg-yellow-700 text-white`}
              >
                <span className="i-mdi-pencil mr-2"></span>
                {t('touchMapping.startEditMode')}
              </button>
            ) : (
              <button
                onClick={() => {
                  touchMappingRef.current?.stopEditMode();
                  setEditMode(false);
                }}
                className={`${buttonClass} bg-green-600 hover:bg-green-700 text-white`}
              >
                <span className="i-mdi-check mr-2"></span>
                {t('touchMapping.finishEdit')}
              </button>
            )}
          </>
        ) : (
          // 非触摸设备按钮（只有选择了设备类型时才显示）
          selectedDeviceType !== null && (
            <>
              <button
                onClick={handleLoadDefaults}
                className={`${buttonClass} bg-indigo-600 hover:bg-indigo-700 text-white`}
              >
                <span className="i-mdi-restore mr-2"></span>
                {t('keyMapping.loadDefaults')}
              </button>
              {gameId !== 'global' && (
                <button
                  onClick={handleLoadGlobal}
                  className={`${buttonClass} bg-teal-600 hover:bg-teal-700 text-white`}
                >
                  <span className="i-mdi-upload mr-2"></span>
                  {t('keyMapping.loadGlobal')}
                </button>
              )}
              <button
                onClick={handleRefresh}
                className={`${buttonClass} bg-blue-600 hover:bg-blue-700 text-white`}
              >
                <span className="i-mdi-refresh mr-2"></span>
                {t('keyMapping.refresh')}
              </button>
              <button
                onClick={handleClear}
                className={`${buttonClass} bg-red-600 hover:bg-red-700 text-white`}
              >
                <span className="i-mdi-delete-sweep mr-2"></span>
                清除
              </button>
            </>
          )
        )}
        {/* 保存按钮始终显示 - 包括选择关闭手柄映射时 */}
        {!(isTouchDevice && editMode) && (
          <>
            <div className="flex-1"></div>
            <button
              onClick={async () => {
                // 先保存设备类型到配置（所有设备类型都需要）
                saveJoystickConfig(selectedDeviceType);
                if (isTouchDevice) {
                  await touchMappingRef.current?.saveAll();
                } else {
                  saveAllMappings();
                }
              }}
              className={`${buttonClass} bg-green-600 hover:bg-green-700 text-white`}
            >
              <span className="i-mdi-content-save mr-2"></span>
              {t('keyMapping.saveAllMappings')}
            </button>
          </>
        )}
      </div>

      {/* 内容区：触摸按钮或手柄映射 - 只有选择了设备类型时才显示 */}
      {selectedDeviceType !== null && (
        <div className="flex-1 relative overflow-hidden">
          {isTouchDevice ? (
            <TouchMappingPanel ref={touchMappingRef} gameId={gameId} />
          ) : (
          <>
            {/* 手柄图片和按钮映射容器 - 动态显示 */}
            {currentDevice && (
              <div className="absolute inset-0 flex items-center justify-center mt-50">
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
                          {hotkey ? (hotkey.action_type === enums.HotkeyActionType.SCREENSHOT ? '📷' : (hotkey.name || mapping.mappingLabel)) : mapping.buttonLabel}
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
          </>
        )}
        </div>
      )}

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
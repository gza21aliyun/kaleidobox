package service

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"lunabox/internal/appconf"
	"lunabox/internal/applog"
	"lunabox/internal/models"
	"lunabox/internal/utils"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/google/uuid"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

// VMRepository 虚拟机仓库接口
type VMRepository interface {
	GetAll() ([]models.Vms, error)
	GetByID(vmID string) (*models.Vms, error)
	Create(vm models.Vms) error
	Update(vm models.Vms) error
	Delete(vmID string) error
}

// vmRepository 虚拟机仓库实现
type vmRepository struct {
	db *sql.DB
}

// NewVMRepository 创建虚拟机仓库实例
func NewVMRepository(db *sql.DB) VMRepository {
	return &vmRepository{
		db: db,
	}
}

// GetAll 获取所有虚拟机
func (r *vmRepository) GetAll() ([]models.Vms, error) {
	query := `
		SELECT vm_id, vm_name, vm_user_name, vm_pass, vm_path, vm_type, host_url, host_user, host_pass
		FROM vms
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query vms: %w", err)
	}
	defer rows.Close()

	var vms []models.Vms
	for rows.Next() {
		var vm models.Vms
		err := rows.Scan(
			&vm.VmId,
			&vm.VmName,
			&vm.VmUserName,
			&vm.VmPass,
			&vm.VmPath,
			&vm.VmType,
			&vm.HostUrl,
			&vm.HostUser,
			&vm.HostPass,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan vm: %w", err)
		}
		vms = append(vms, vm)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return vms, nil
}

// GetByID 根据 ID 获取虚拟机
func (r *vmRepository) GetByID(vmID string) (*models.Vms, error) {
	query := `
		SELECT vm_id, vm_name, vm_user_name, vm_pass, vm_path, vm_type, host_url, host_user, host_pass
		FROM vms
		WHERE vm_id = ?
	`

	var vm models.Vms
	err := r.db.QueryRow(query, vmID).Scan(
		&vm.VmId,
		&vm.VmName,
		&vm.VmUserName,
		&vm.VmPass,
		&vm.VmPath,
		&vm.VmType,
		&vm.HostUrl,
		&vm.HostUser,
		&vm.HostPass,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("vm not found: %s", vmID)
		}
		return nil, fmt.Errorf("failed to query vm: %w", err)
	}

	return &vm, nil
}

// Create 创建虚拟机
func (r *vmRepository) Create(vm models.Vms) error {
	query := `
		INSERT INTO vms (vm_id, vm_name, vm_user_name, vm_pass, vm_path, vm_type, host_url, host_user, host_pass)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(
		query,
		vm.VmId,
		vm.VmName,
		vm.VmUserName,
		vm.VmPass,
		vm.VmPath,
		vm.VmType,
		vm.HostUrl,
		vm.HostUser,
		vm.HostPass,
	)
	if err != nil {
		return fmt.Errorf("failed to create vm: %w", err)
	}

	return nil
}

// Update 更新虚拟机
func (r *vmRepository) Update(vm models.Vms) error {
	query := `
		UPDATE vms
		SET vm_name = ?, vm_user_name = ?, vm_pass = ?, vm_path = ?, vm_type = ?, host_url = ?, host_user = ?, host_pass = ?
		WHERE vm_id = ?
	`

	_, err := r.db.Exec(
		query,
		vm.VmName,
		vm.VmUserName,
		vm.VmPass,
		vm.VmPath,
		vm.VmType,
		vm.HostUrl,
		vm.HostUser,
		vm.HostPass,
		vm.VmId,
	)
	if err != nil {
		return fmt.Errorf("failed to update vm: %w", err)
	}

	return nil
}

// Delete 删除虚拟机
func (r *vmRepository) Delete(vmID string) error {
	query := `DELETE FROM vms WHERE vm_id = ?`

	_, err := r.db.Exec(query, vmID)
	if err != nil {
		return fmt.Errorf("failed to delete vm: %w", err)
	}

	return nil
}

// VMService 虚拟机服务
type VMService struct {
	ctx         context.Context
	vmRepo      VMRepository
	config      *appconf.AppConfig
	gameService *GameService
}

// NewVMService 创建虚拟机服务实例
func NewVMService() *VMService {
	return &VMService{}
}

// Init 初始化虚拟机服务
func (s *VMService) Init(ctx context.Context, db *sql.DB, config *appconf.AppConfig) error {
	s.ctx = ctx
	s.vmRepo = NewVMRepository(db)
	s.config = config
	return nil
}

// SetGameService 设置游戏服务
func (s *VMService) SetGameService(gameService *GameService) {
	s.gameService = gameService
}

// GetAllVMs 获取所有虚拟机
func (s *VMService) GetAllVMs() ([]models.Vms, error) {
	vms, err := s.vmRepo.GetAll()
	if err != nil {
		applog.LogErrorf(s.ctx, "failed to get all VMs: %v", err)
		return []models.Vms{}, err
	}
	if vms == nil {
		return []models.Vms{}, nil
	}
	return vms, nil
}

// GetVMByID 根据 ID 获取虚拟机
func (s *VMService) GetVMByID(vmID string) (*models.Vms, error) {
	vm, err := s.vmRepo.GetByID(vmID)
	if err != nil {
		applog.LogErrorf(s.ctx, "failed to get VM by ID: %v", err)
		return nil, err
	}
	return vm, nil
}

// AddVM 添加虚拟机
func (s *VMService) AddVM(vm models.Vms) error {
	vm.VmId = uuid.New().String()
	err := s.vmRepo.Create(vm)
	if err != nil {
		applog.LogErrorf(s.ctx, "failed to add VM: %v", err)
		return err
	}
	return nil
}

// UpdateVM 更新虚拟机
func (s *VMService) UpdateVM(vm models.Vms) error {
	err := s.vmRepo.Update(vm)
	if err != nil {
		applog.LogErrorf(s.ctx, "failed to update VM: %v", err)
		return err
	}
	return nil
}

// DeleteVM 删除虚拟机
func (s *VMService) DeleteVM(vmID string) error {
	err := s.vmRepo.Delete(vmID)
	if err != nil {
		applog.LogErrorf(s.ctx, "failed to delete VM: %v", err)
		return err
	}
	return nil
}

// StartGameInsideVm 在虚拟机中启动游戏
// useMagpie - 是否启用 Magpie 缩放
func (s *VMService) StartGameInsideVm(gameID string, useMagpie bool) (bool, error) {
	// 获取游戏信息
	game, err := s.gameService.GetGameByID(gameID)
	if err != nil {
		applog.LogErrorf(s.ctx, "failed to get game: %v", err)
		return false, fmt.Errorf("failed to get game: %w", err)
	}

	if game.VmId == "" {
		applog.LogErrorf(s.ctx, "game vm_id is empty: %s", gameID)
		return false, fmt.Errorf("game vm_id is empty: %s", gameID)
	}

	// 获取虚拟机信息
	vm, err := s.GetVMByID(game.VmId)
	if err != nil {
		applog.LogErrorf(s.ctx, "failed to get vm: %v", err)
		return false, fmt.Errorf("failed to get vm: %w", err)
	}

	// 获取游戏路径和参数
	if err != nil {
		applog.LogErrorf(s.ctx, "failed to get game path: %v", err)
		return false, fmt.Errorf("failed to get game path: %w", err)
	}

	// 根据虚拟机类型调用不同的启动函数
	switch vm.VmType {
	case "workstation":
		success, err := s.StartGameInsideWs(vm, game.Path, game.Arguments)
		if success && err == nil && useMagpie {
			// 游戏启动成功后配置裁剪并触发 Magpie 缩放
			s.configureMagpieAndScale()
		}
		return success, err
	case "esx":
		success, err := s.StartGameInsideEsx(vm, game.Path, game.Arguments)
		if success && err == nil && useMagpie {
			// 游戏启动成功后配置裁剪并触发 Magpie 缩放
			s.configureMagpieAndScale()
		}
		return success, err
	default:
		applog.LogErrorf(s.ctx, "unsupported vm type: %s", vm.VmType)
		return false, fmt.Errorf("unsupported vm type: %s", vm.VmType)
	}
}

// configureMagpieAndScale 配置 Magpie 裁剪并触发缩放
func (s *VMService) configureMagpieAndScale() {
	// 如果启用了裁剪参数且不全为 0，先配置裁剪
	if s.config.MagpieCroppingEnabled && s.hasCroppingParams() {
		applog.LogInfof(s.ctx, "配置 Magpie 裁剪参数...")
		if err := s.configureMagpieCropping(); err != nil {
			applog.LogWarningf(s.ctx, "配置 Magpie 裁剪失败: %v", err)
			return
		}
	}

	// 触发 Magpie 缩放
	s.triggerMagpieScalingForVM()
}

// hasCroppingParams 检查裁剪参数是否不全为 0
func (s *VMService) hasCroppingParams() bool {
	return s.config.MagpieCroppingLeft != 0 ||
		s.config.MagpieCroppingTop != 0 ||
		s.config.MagpieCroppingRight != 0 ||
		s.config.MagpieCroppingBottom != 0
}

// configureMagpieCropping 配置 Magpie 裁剪参数
func (s *VMService) configureMagpieCropping() error {
	// 确定配置文件路径
	configPath := s.config.MagpieConfigPath
	if configPath == "" {
		// 使用默认路径
		homeDir := os.Getenv("USERPROFILE")
		configPath = filepath.Join(homeDir, "AppData", "Local", "Magpie", "config", "v4", "config.json")
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		applog.LogErrorf(s.ctx, "读取 Magpie 配置失败: %v", err)
		return err
	}

	// 解析 JSON
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		applog.LogErrorf(s.ctx, "解析 Magpie 配置失败: %v", err)
		return err
	}

	// 获取 profiles 数组
	profiles, ok := config["profiles"].([]interface{})
	if !ok || len(profiles) == 0 {
		applog.LogErrorf(s.ctx, "配置文件中没有 profiles 数组")
		return fmt.Errorf("配置文件中没有 profiles 数组")
	}

	// 获取第一个 profile
	profile, ok := profiles[0].(map[string]interface{})
	if !ok {
		applog.LogErrorf(s.ctx, "profiles[0] 不是 map")
		return fmt.Errorf("profiles[0] 不是 map")
	}

	// 检查当前配置是否已经正确
	currentCroppingEnabled := false
	if val, ok := profile["croppingEnabled"].(bool); ok {
		currentCroppingEnabled = val
	}
	currentCropping, _ := profile["cropping"].(map[string]interface{})

	if currentCroppingEnabled && currentCropping != nil {
		// 获取当前裁剪参数（JSON解析时数字默认为float64，需转换为int）
		currentLeft := 0
		if v, ok := currentCropping["left"].(float64); ok {
			currentLeft = int(v)
		}
		currentTop := 0
		if v, ok := currentCropping["top"].(float64); ok {
			currentTop = int(v)
		}
		currentRight := 0
		if v, ok := currentCropping["right"].(float64); ok {
			currentRight = int(v)
		}
		currentBottom := 0
		if v, ok := currentCropping["bottom"].(float64); ok {
			currentBottom = int(v)
		}

		// 检查是否与目标值相同
		if currentLeft == s.config.MagpieCroppingLeft &&
			currentTop == s.config.MagpieCroppingTop &&
			currentRight == s.config.MagpieCroppingRight &&
			currentBottom == s.config.MagpieCroppingBottom {

			applog.LogInfof(s.ctx, "Magpie 裁剪参数已正确配置，无需修改")
			return nil
		}
	}

	// 修改裁剪设置
	cropping := map[string]interface{}{
		"left":   s.config.MagpieCroppingLeft,
		"top":    s.config.MagpieCroppingTop,
		"right":  s.config.MagpieCroppingRight,
		"bottom": s.config.MagpieCroppingBottom,
	}
	profile["croppingEnabled"] = true
	profile["cropping"] = cropping

	// 写回配置文件
	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		applog.LogErrorf(s.ctx, "序列化 Magpie 配置失败: %v", err)
		return err
	}

	if err := os.WriteFile(configPath, newData, 0644); err != nil {
		applog.LogErrorf(s.ctx, "写入 Magpie 配置失败: %v", err)
		return err
	}

	applog.LogInfof(s.ctx, "Magpie 裁剪参数已配置: left=%.1f, top=%.1f, right=%.1f, bottom=%.1f",
		s.config.MagpieCroppingLeft, s.config.MagpieCroppingTop,
		s.config.MagpieCroppingRight, s.config.MagpieCroppingBottom)

	// 重启 Magpie
	return s.restartMagpie()
}

// restartMagpie 重启 Magpie 进程
func (s *VMService) restartMagpie() error {
	if s.config.MagpiePath == "" {
		return fmt.Errorf("Magpie 路径未设置")
	}

	// 关闭现有 Magpie 进程
	applog.InfoLogSaveAppLog("关闭 Magpie 进程...")
	killCmd := exec.Command("taskkill", "/F", "/IM", "Magpie.exe")
	_ = killCmd.Run()

	// 等待进程关闭
	time.Sleep(1 * time.Second)

	// 启动 Magpie
	applog.InfoLogSaveAppLog("启动 Magpie...")
	cmd := exec.Command(s.config.MagpiePath, "-t")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}

	if err := cmd.Start(); err != nil {
		applog.LogErrorf(s.ctx, "启动 Magpie 失败: %v", err)
		return err
	}

	// 等待 Magpie 启动
	time.Sleep(2 * time.Second)
	applog.InfoLogSaveAppLog("Magpie 已重启")

	return nil
}

// triggerMagpieScalingForVM 触发 Magpie 缩放（虚拟机模式）
func (s *VMService) triggerMagpieScalingForVM() {
	if s.config.MagpiePath == "" {
		return
	}

	// 等待游戏窗口出现
	applog.LogInfof(s.ctx, "等待游戏窗口出现...")
	time.Sleep(1 * time.Second)

	// 先激活 VMware 窗口
	applog.LogInfof(s.ctx, "激活 VMware 窗口...")
	if err := s.activateVMwareWindow(); err != nil {
		applog.LogWarningf(s.ctx, "激活 VMware 窗口失败: %v", err)
	}

	// 等待窗口真正获得焦点
	applog.LogInfof(s.ctx, "等待窗口获得焦点...")
	time.Sleep(3 * time.Second)

	// 发送缩放快捷键
	applog.InfoLogSaveAppLog("发送 Magpie 缩放快捷键: %s", s.config.MagpieHotkey)
	utils.SendHotkey("ctrl+alt")
	time.Sleep(500 * time.Millisecond)
	utils.SendHotkey(s.config.MagpieHotkey)

	// 等待快捷键生效
	time.Sleep(500 * time.Millisecond)
	applog.InfoLogSaveAppLog("Magpie trigger: scaling hotkey sent successfully")
}

// activateVMwareWindow 激活 VMware 窗口
func (s *VMService) activateVMwareWindow() error {
	// 定义 user32.dll 函数
	user32 := syscall.NewLazyDLL("user32.dll")
	procEnumWindows := user32.NewProc("EnumWindows")
	procGetWindowTextLengthW := user32.NewProc("GetWindowTextLengthW")
	procGetWindowTextW := user32.NewProc("GetWindowTextW")
	procSetForegroundWindow := user32.NewProc("SetForegroundWindow")

	var targetHWND uintptr

	// 回调函数用于查找 VMware 窗口
	callback := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		// 获取窗口标题长度
		ret, _, _ := procGetWindowTextLengthW.Call(hwnd)
		if ret == 0 {
			return 1 // 继续枚举
		}

		// 分配缓冲区
		buf := make([]uint16, ret+1)
		procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), ret+1)
		title := syscall.UTF16ToString(buf)

		// 检查窗口标题是否包含 VMware 相关内容
		if strings.Contains(strings.ToLower(title), "vmware") {
			targetHWND = hwnd
			return 0 // 停止枚举
		}

		return 1 // 继续枚举
	})

	// 枚举所有顶层窗口
	procEnumWindows.Call(callback, 0)

	if targetHWND == 0 {
		return fmt.Errorf("未找到 VMware 窗口")
	}

	// 设置为前台窗口
	ret, _, err := procSetForegroundWindow.Call(targetHWND)
	if ret == 0 {
		return fmt.Errorf("SetForegroundWindow 失败: %v", err)
	}

	applog.LogInfof(s.ctx, "VMware 窗口已激活")
	return nil
}

// StartGameInsideWs 在 Workstation 虚拟机中启动游戏
func (s *VMService) StartGameInsideWs(vm *models.Vms, path, arguments string) (bool, error) {
	// 检查虚拟机配置是否完整
	if s.config.VmrunPath == "" || vm.VmPath == "" || vm.VmUserName == "" {
		applog.LogErrorf(s.ctx, "虚拟机配置不完整")
		return false, fmt.Errorf("虚拟机配置不完整，请在设置中配置虚拟机参数")
	}

	// 构建vmrun命令来启动虚拟机（如果未运行）
	vmrunCmd := exec.Command(s.config.VmrunPath, "start", vm.VmPath, "gui")
	if err := vmrunCmd.Run(); err != nil {
		applog.LogWarningf(s.ctx, "启动虚拟机失败，可能已经在运行: %v", err)
	}

	// 等待虚拟机启动
	time.Sleep(10 * time.Second)

	// 构建vmrun命令在虚拟机中启动游戏
	var vmrunArgs []string
	vmrunArgs = append(vmrunArgs, "-T", "ws")
	vmrunArgs = append(vmrunArgs, "-gu", vm.VmUserName)
	vmrunArgs = append(vmrunArgs, "-gp", vm.VmPass)
	vmrunArgs = append(vmrunArgs, "runProgramInGuest")
	vmrunArgs = append(vmrunArgs, vm.VmPath)
	vmrunArgs = append(vmrunArgs, "-noWait")
	vmrunArgs = append(vmrunArgs, "-interactive")
	vmrunArgs = append(vmrunArgs, "-activeWindow")
	vmrunArgs = append(vmrunArgs, path)
	if arguments != "" {
		vmrunArgs = append(vmrunArgs, arguments)
	}

	applog.LogInfof(s.ctx, "在 Workstation 虚拟机中启动游戏")
	cmd := exec.Command(s.config.VmrunPath, vmrunArgs...)

	if err := cmd.Start(); err != nil {
		applog.LogErrorf(s.ctx, "failed to start game in VM: %v", err)
		return false, fmt.Errorf("failed to start game in VM: %w", err)
	}

	// 启动成功，返回 true 给前端
	return true, nil
}

// StartGameInsideEsx 在 ESX 虚拟机中启动游戏
func (s *VMService) StartGameInsideEsx(vm *models.Vms, path, arguments string) (bool, error) {
	// 检查 ESX 配置是否完整
	if vm.HostUrl == "" || vm.HostUser == "" || vm.HostPass == "" {
		applog.LogErrorf(s.ctx, "ESX 虚拟机配置不完整")
		return false, fmt.Errorf("ESX 虚拟机配置不完整")
	}

	applog.LogInfof(s.ctx, "在 ESX 虚拟机中启动游戏")

	// 1. 连接到 ESX 主机
	c, sessionManager, err := s.ConnectToESX(vm.HostUrl, vm.HostUser, vm.HostPass)
	if err != nil {
		applog.LogErrorf(s.ctx, "连接 ESX 主机失败: %v", err)
		return false, fmt.Errorf("连接 ESX 主机失败: %w", err)
	}
	defer sessionManager.Logout(s.ctx)

	// 2. 找到指定的虚拟机
	vmObj, err := s.FindVM(c, vm.VmPath)
	if err != nil {
		applog.LogErrorf(s.ctx, "查找虚拟机失败: %v", err)
		return false, fmt.Errorf("查找虚拟机失败: %w", err)
	}
	fmt.Println("虚拟机已找到:", vmObj.Name())

	// 3. 确保虚拟机处于运行状态
	running, err := s.IsVMRunning(vmObj)
	if err != nil {
		applog.LogErrorf(s.ctx, "检查虚拟机状态失败: %v", err)
		return false, fmt.Errorf("检查虚拟机状态失败: %w", err)
	}

	if !running {
		applog.LogInfof(s.ctx, "启动虚拟机: %s", vm.VmName)
		err = s.PowerOnVM(vmObj)
		if err != nil {
			applog.LogErrorf(s.ctx, "启动虚拟机失败: %v", err)
			return false, fmt.Errorf("启动虚拟机失败: %w", err)
		}
		// 等待虚拟机启动
		time.Sleep(30 * time.Second)
	}

	// 4. 在虚拟机中启动游戏
	applog.LogInfof(s.ctx, "在 ESX 虚拟机中执行游戏启动命令")
	sc := c.ServiceContent
	var guestOpsMgr mo.GuestOperationsManager
	if c.ServiceContent.GuestOperationsManager == nil {
		fmt.Println("ESXi 主机不支持 Guest Operations 或未正确初始化")
		return false, fmt.Errorf("ESXi 主机不支持 Guest Operations 或未正确初始化")
	}
	err = property.DefaultCollector(c).RetrieveOne(s.ctx, *sc.GuestOperationsManager, []string{"processManager"}, &guestOpsMgr)
	if err != nil || guestOpsMgr.ProcessManager == nil {
		return false, fmt.Errorf("无法获取 ProcessManager 引用: %w", err)
	}

	// 注意：在 ESX 虚拟机中启动游戏需要启用 VMware Tools
	// 由于涉及到复杂的 Guest Operations 配置，这里暂时返回成功
	// 实际实现
	guestAuth := &types.NamePasswordAuthentication{
		Username: vm.VmUserName, // 虚拟机里的 Windows 账号
		Password: vm.VmPass,     // 虚拟机里的 Windows 密码
		GuestAuthentication: types.GuestAuthentication{
			InteractiveSession: true,
		},
	}

	gameExeName := filepath.Base(path)
	if ext := filepath.Ext(gameExeName); ext != "" {
		gameExeName = gameExeName[:len(gameExeName)-len(ext)]
	}
	workingDir := filepath.Dir(path)

	// 使用 cmd.exe /s /c 启动游戏，而不是直接调用 exe。
	// 原因：直接通过 StartProgramInGuest 调用 exe 会跳过 Shell 的初始化流程，
	// 导致：1) 用户环境变量未完全加载（PATH、APPDATA 等）；2) 缺少兼容性垫片；
	// 3) 快捷方式(.lnk)无法解析；4) 路径含空格时解析异常。
	// 用 cmd.exe /s /c 包裹等价于在命令提示符中执行，行为更接近用户双击运行。
	// 引号规则：cmd /s /c 会剥离首尾的一对引号，因此用 ""X"" 包裹，剥离后剩 "X"，
	// 从而正确保留路径本身的引号（处理含空格路径）。
	var cmdArgs string
	if arguments != "" {
		cmdArgs = fmt.Sprintf(`/s /c ""%s" %s"`, path, arguments)
	} else {
		cmdArgs = fmt.Sprintf(`/s /c ""%s""`, path)
	}

	applog.LogInfof(s.ctx, "ESX 启动命令: ProgramPath=C:\\Windows\\System32\\cmd.exe Arguments=%s WorkingDir=%s", cmdArgs, workingDir)
	fmt.Printf("ESX 启动命令: cmd.exe %s (工作目录: %s)\n", cmdArgs, workingDir)

	spec := &types.GuestProgramSpec{
		ProgramPath:      "C:\\Windows\\System32\\cmd.exe",
		Arguments:        cmdArgs,
		WorkingDirectory: workingDir,
	}

	// 手动构建 StartProgramInGuest 请求
	req := types.StartProgramInGuest{
		This: *guestOpsMgr.ProcessManager,
		Vm:   vmObj.Reference(),
		Auth: guestAuth,
		Spec: spec,
	}
	fmt.Println("开始启动游戏")

	// 发起 SOAP 请求
	res, err := methods.StartProgramInGuest(s.ctx, c, &req)
	if err != nil {
		fmt.Println("无法启动游戏:", err)
		applog.LogErrorf(s.ctx, "无法启动游戏, path=%s args=%s err=%v", path, arguments, err)
		return false, fmt.Errorf("无法启动游戏: %v", err)
	}

	fmt.Printf("游戏已在虚拟机中成功启动，进程 ID: %d\n", res.Returnval)
	applog.LogInfof(s.ctx, "游戏已在虚拟机中成功启动，进程 ID: %d\n", res.Returnval)

	// 启动后激活窗口到前台，模拟 vmrun -activeWindow 行为
	go func() {
		time.Sleep(5 * time.Second) // 等待游戏初始化

		// 使用简单的 AppActivate，兼容性最好
		activateScript := fmt.Sprintf(`$wshell = New-Object -ComObject WScript.Shell; if ($wshell.AppActivate('%s')) { Write-Host 'Activated' } else { Write-Host 'Failed to find window' }`, gameExeName)

		encoded := base64.StdEncoding.EncodeToString([]byte(activateScript))
		activateSpec := &types.GuestProgramSpec{
			ProgramPath: "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe",
			Arguments:   fmt.Sprintf("-NoProfile -WindowStyle Hidden -EncodedCommand %s", encoded),
		}

		// 再次发起请求
		_, err := methods.StartProgramInGuest(s.ctx, c, &types.StartProgramInGuest{
			This: *guestOpsMgr.ProcessManager,
			Vm:   vmObj.Reference(),
			Auth: guestAuth,
			Spec: activateSpec,
		})

		if err != nil {
			applog.LogWarningf(s.ctx, "窗口激活脚本执行失败: %v", err)
		} else {
			applog.LogInfof(s.ctx, "窗口激活脚本已执行: %s", gameExeName)
		}
	}()

	return true, nil
}

// ... existing code ...

// OpenGamePathInsideVm 在虚拟机中打开游戏路径
func (s *VMService) OpenGamePathInsideVm(gameID string) (bool, error) {
	// 获取游戏信息
	game, err := s.gameService.GetGameByID(gameID)
	if err != nil {
		applog.LogErrorf(s.ctx, "failed to get game: %v", err)
		return false, fmt.Errorf("failed to get game: %w", err)
	}

	if game.VmId == "" {
		applog.LogErrorf(s.ctx, "game vm_id is empty: %s", gameID)
		return false, fmt.Errorf("game vm_id is empty: %s", gameID)
	}

	// 获取虚拟机信息
	vm, err := s.GetVMByID(game.VmId)
	if err != nil {
		applog.LogErrorf(s.ctx, "failed to get vm: %v", err)
		return false, fmt.Errorf("failed to get vm: %w", err)
	}

	// 检查游戏路径是否存在
	if game.Path == "" {
		applog.LogErrorf(s.ctx, "game path is empty: %s", gameID)
		return false, fmt.Errorf("game path is empty: %s", gameID)
	}

	// 根据虚拟机类型调用不同的打开路径函数
	switch vm.VmType {
	case "workstation":
		return s.OpenGamePathInsideWs(vm, game.Path)
	case "esx":
		return s.OpenGamePathInsideEsx(vm, game.Path)
	default:
		applog.LogErrorf(s.ctx, "unsupported vm type: %s", vm.VmType)
		return false, fmt.Errorf("unsupported vm type: %s", vm.VmType)
	}
}

// OpenGamePathInsideWs 在 Workstation 虚拟机中打开游戏路径
func (s *VMService) OpenGamePathInsideWs(vm *models.Vms, path string) (bool, error) {
	// 检查虚拟机配置是否完整
	if s.config.VmrunPath == "" || vm.VmPath == "" || vm.VmUserName == "" {
		applog.LogErrorf(s.ctx, "虚拟机配置不完整")
		return false, fmt.Errorf("虚拟机配置不完整，请在设置中配置虚拟机参数")
	}

	// 构建vmrun命令来启动虚拟机（如果未运行）
	vmrunCmd := exec.Command(s.config.VmrunPath, "start", vm.VmPath, "gui")
	if err := vmrunCmd.Run(); err != nil {
		applog.LogWarningf(s.ctx, "启动虚拟机失败，可能已经在运行: %v", err)
	}

	// 等待虚拟机启动
	time.Sleep(10 * time.Second)

	// 构建在虚拟机中打开路径的命令
	// 使用 cmd.exe 执行 explorer 命令，兼容 Windows XP 及更高版本
	var vmrunArgs []string
	vmrunArgs = append(vmrunArgs, "-T", "ws")
	vmrunArgs = append(vmrunArgs, "-gu", vm.VmUserName)
	vmrunArgs = append(vmrunArgs, "-gp", vm.VmPass)
	vmrunArgs = append(vmrunArgs, "runProgramInGuest")
	vmrunArgs = append(vmrunArgs, vm.VmPath)
	vmrunArgs = append(vmrunArgs, "-noWait")
	vmrunArgs = append(vmrunArgs, "-interactive")

	// 使用 cmd.exe /c 执行 explorer 命令，兼容所有 Windows 版本
	vmrunArgs = append(vmrunArgs, "C:\\Windows\\System32\\cmd.exe")
	vmrunArgs = append(vmrunArgs, "/c")
	vmrunArgs = append(vmrunArgs, fmt.Sprintf(`explorer.exe /select,"%s"`, path))

	applog.LogInfof(s.ctx, "在 Workstation 虚拟机中打开游戏路径: %s", path)
	cmd := exec.Command(s.config.VmrunPath, vmrunArgs...)

	if err := cmd.Start(); err != nil {
		applog.LogErrorf(s.ctx, "failed to open path in VM: %v", err)
		return false, fmt.Errorf("failed to open path in VM: %w", err)
	}

	// 操作成功，返回 true 给前端
	return true, nil
}

// OpenGamePathInsideEsx 在 ESX 虚拟机中打开游戏路径
func (s *VMService) OpenGamePathInsideEsx(vm *models.Vms, path string) (bool, error) {
	// 检查 ESX 配置是否完整
	if vm.HostUrl == "" || vm.HostUser == "" || vm.HostPass == "" {
		applog.LogErrorf(s.ctx, "ESX 虚拟机配置不完整")
		return false, fmt.Errorf("ESX 虚拟机配置不完整")
	}

	applog.LogInfof(s.ctx, "在 ESX 虚拟机中打开游戏路径: %s", path)

	// 1. 连接到 ESX 主机
	c, sessionManager, err := s.ConnectToESX(vm.HostUrl, vm.HostUser, vm.HostPass)
	if err != nil {
		applog.LogErrorf(s.ctx, "连接 ESX 主机失败: %v", err)
		return false, fmt.Errorf("连接 ESX 主机失败: %w", err)
	}
	defer sessionManager.Logout(s.ctx)

	// 2. 找到指定的虚拟机
	vmObj, err := s.FindVM(c, vm.VmPath)
	if err != nil {
		applog.LogErrorf(s.ctx, "查找虚拟机失败: %v", err)
		return false, fmt.Errorf("查找虚拟机失败: %w", err)
	}
	fmt.Println("虚拟机已找到:", vmObj.Name())

	// 3. 确保虚拟机处于运行状态
	running, err := s.IsVMRunning(vmObj)
	if err != nil {
		applog.LogErrorf(s.ctx, "检查虚拟机状态失败: %v", err)
		return false, fmt.Errorf("检查虚拟机状态失败: %w", err)
	}

	if !running {
		applog.LogInfof(s.ctx, "启动虚拟机: %s", vm.VmName)
		err = s.PowerOnVM(vmObj)
		if err != nil {
			applog.LogErrorf(s.ctx, "启动虚拟机失败: %v", err)
			return false, fmt.Errorf("启动虚拟机失败: %w", err)
		}
		// 等待虚拟机启动
		time.Sleep(30 * time.Second)
	}

	// 4. 在虚拟机中打开路径
	applog.LogInfof(s.ctx, "在 ESX 虚拟机中执行打开路径命令")
	sc := c.ServiceContent
	var guestOpsMgr mo.GuestOperationsManager
	if c.ServiceContent.GuestOperationsManager == nil {
		fmt.Println("ESXi 主机不支持 Guest Operations 或未正确初始化")
		return false, fmt.Errorf("ESXi 主机不支持 Guest Operations 或未正确初始化")
	}
	err = property.DefaultCollector(c).RetrieveOne(s.ctx, *sc.GuestOperationsManager, []string{"processManager"}, &guestOpsMgr)
	if err != nil || guestOpsMgr.ProcessManager == nil {
		return false, fmt.Errorf("无法获取 ProcessManager 引用: %w", err)
	}

	guestAuth := &types.NamePasswordAuthentication{
		Username: vm.VmUserName,
		Password: vm.VmPass,
		GuestAuthentication: types.GuestAuthentication{
			InteractiveSession: true,
		},
	}

	// 使用 cmd.exe /c 执行 explorer 命令，兼容 Windows XP 及更高版本
	spec := &types.GuestProgramSpec{
		ProgramPath:      "C:\\Windows\\System32\\cmd.exe",
		Arguments:        fmt.Sprintf(`/c explorer.exe /select,"%s"`, path),
		WorkingDirectory: filepath.Dir(path),
	}

	req := types.StartProgramInGuest{
		This: *guestOpsMgr.ProcessManager,
		Vm:   vmObj.Reference(),
		Auth: guestAuth,
		Spec: spec,
	}
	fmt.Println("开始打开路径")

	res, err := methods.StartProgramInGuest(s.ctx, c, &req)
	if err != nil {
		fmt.Println("无法打开路径:", err)
		return false, fmt.Errorf("无法打开路径: %v", err)
	}

	fmt.Printf("路径已在虚拟机中成功打开，进程 ID: %d\n", res.Returnval)
	applog.LogInfof(s.ctx, "路径已在虚拟机中成功打开，进程 ID: %d\n", res.Returnval)

	return true, nil
}

// ... existing code ...

// ConnectToESX 连接到 ESX 主机
func (s *VMService) ConnectToESX(hostURL, username, password string) (*vim25.Client, *session.Manager, error) {
	applog.LogInfof(s.ctx, "连接到 ESX 主机: %s", hostURL)

	// 解析 URL
	u, err := soap.ParseURL(hostURL)
	if err != nil {
		return nil, nil, fmt.Errorf("解析 ESX 主机 URL 失败: %w", err)
	}

	// 设置用户名和密码
	u.User = url.UserPassword(username, password)

	// 创建客户端
	c, err := vim25.NewClient(s.ctx, soap.NewClient(u, true))
	if err != nil {
		return nil, nil, fmt.Errorf("创建 ESX 客户端失败: %w", err)
	}

	// 登录
	sessionManager := session.NewManager(c)
	err = sessionManager.Login(s.ctx, u.User)
	if err != nil {
		return nil, nil, fmt.Errorf("登录 ESX 主机失败: %w", err)
	}

	applog.LogInfof(s.ctx, "成功连接到 ESX 主机")
	return c, sessionManager, nil
}

// FindVM 查找指定名称的虚拟机
func (s *VMService) FindVM(c *vim25.Client, vmPath string) (*object.VirtualMachine, error) {
	applog.LogInfof(s.ctx, "查找虚拟机: %s", vmPath)

	finder := find.NewFinder(c)
	vm, err := finder.VirtualMachine(s.ctx, vmPath)
	if err != nil {
		return nil, fmt.Errorf("找不到虚拟机 '%s': %w", vmPath, err)
	}

	applog.LogInfof(s.ctx, "成功找到虚拟机: %s", vmPath)
	return vm, nil
}

// IsVMRunning 检查虚拟机是否正在运行
func (s *VMService) IsVMRunning(vm *object.VirtualMachine) (bool, error) {
	applog.LogInfof(s.ctx, "检查虚拟机状态")

	// 使用 vm.PowerState() 方法获取虚拟机的电源状态
	powerState, err := vm.PowerState(s.ctx)
	if err != nil {
		return false, fmt.Errorf("获取虚拟机状态失败: %w", err)
	}

	running := powerState == "poweredOn"
	applog.LogInfof(s.ctx, "虚拟机状态: %s", powerState)
	return running, nil
}

// PowerOnVM 启动虚拟机
func (s *VMService) PowerOnVM(vm *object.VirtualMachine) error {
	applog.LogInfof(s.ctx, "启动虚拟机")

	task, err := vm.PowerOn(s.ctx)
	if err != nil {
		return fmt.Errorf("启动虚拟机失败: %w", err)
	}

	// 等待任务完成
	err = task.Wait(s.ctx)
	if err != nil {
		return fmt.Errorf("启动虚拟机任务失败: %w", err)
	}

	applog.LogInfof(s.ctx, "虚拟机启动成功")
	return nil
}

// ListVmsInEsx 列出 ESX 主机上的所有虚拟机
func (s *VMService) ListVmsInEsx(hostURL, username, password string) ([]models.Vms, error) {
	applog.LogInfof(s.ctx, "列出 ESX 主机上的所有虚拟机: %s", hostURL)

	// 连接到 ESX 主机
	c, sessionManager, err := s.ConnectToESX(hostURL, username, password)
	if err != nil {
		applog.LogErrorf(s.ctx, "连接 ESX 主机失败: %v", err)
		return nil, fmt.Errorf("连接 ESX 主机失败: %w", err)
	}
	defer sessionManager.Logout(s.ctx)

	// 创建 finder
	finder := find.NewFinder(c)

	// 获取所有虚拟机
	vms, err := finder.VirtualMachineList(s.ctx, "*")
	if err != nil {
		applog.LogErrorf(s.ctx, "获取 ESX 主机上的虚拟机列表失败: %v", err)
		return nil, fmt.Errorf("获取 ESX 主机上的虚拟机列表失败: %w", err)
	}

	// 转换为 models.Vms 类型
	var result []models.Vms
	for _, vm := range vms {
		// 获取虚拟机的名称
		name := vm.Name()

		// 创建 models.Vms 对象
		vmObj := models.Vms{
			VmName:   name,
			VmType:   "esx",
			HostUrl:  hostURL,
			HostUser: username,
			HostPass: password,
			VmPath:   vm.InventoryPath,
		}

		result = append(result, vmObj)
	}

	applog.LogInfof(s.ctx, "成功列出 ESX 主机上的 %d 个虚拟机", len(result))
	return result, nil
}

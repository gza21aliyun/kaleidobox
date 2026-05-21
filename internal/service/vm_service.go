package service

import (
	"context"
	"database/sql"
	"fmt"
	"lunabox/internal/appconf"
	"lunabox/internal/applog"
	"lunabox/internal/models"
	"net/url"
	"os/exec"
	"path/filepath"
	"time"

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
func (s *VMService) StartGameInsideVm(gameID string) (bool, error) {
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
		return s.StartGameInsideWs(vm, game.Path, game.Arguments)
	case "esx":
		return s.StartGameInsideEsx(vm, game.Path, game.Arguments)
	default:
		applog.LogErrorf(s.ctx, "unsupported vm type: %s", vm.VmType)
		return false, fmt.Errorf("unsupported vm type: %s", vm.VmType)
	}
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

	spec := &types.GuestProgramSpec{
		ProgramPath:      path,
		Arguments:        arguments,
		WorkingDirectory: filepath.Dir(path),
	}
	// 关键：获取 ServiceContent 中的 GuestOperationsManager 引用
	// c 是 *vim25.Client，它内部有一个 ServiceContent 字段
	// 获取 ServiceContent

	// 手动构建 StartProgramInGuest 请求
	req := types.StartProgramInGuest{
		This: *guestOpsMgr.ProcessManager, // 关键点：使用 ProcessManager 的引用
		Vm:   vmObj.Reference(),           // 虚拟机引用
		Auth: guestAuth,
		Spec: spec,
	}
	fmt.Println("开始启动游戏")

	// 发起 SOAP 请求
	res, err := methods.StartProgramInGuest(s.ctx, c, &req)
	if err != nil {
		fmt.Println("无法启动游戏:", err)
		return false, fmt.Errorf("无法启动游戏: %v", err)
	}
	//需要添加更多代码，包括获取 GuestOperationsManager 并执行命令
	// 获取 Service Content 以找到 GuestOperationsManager

	fmt.Printf("游戏已在虚拟机中成功启动，进程 ID: %d\n", res.Returnval)
	applog.LogInfof(s.ctx, "游戏已在虚拟机中成功启动，进程 ID: %d\n", res.Returnval)

	// go func() {
	// 	time.Sleep(5 * time.Second) // 等待游戏初始化

	// 	gameExeName := filepath.Base(path)
	// 	if ext := filepath.Ext(gameExeName); ext != "" {
	// 		gameExeName = gameExeName[:len(gameExeName)-len(ext)]
	// 	}

	// 	// 使用简单的 AppActivate，兼容性最好
	// 	activateScript := fmt.Sprintf(`$wshell = New-Object -ComObject WScript.Shell; if ($wshell.AppActivate('%s')) { Write-Host 'Activated' } else { Write-Host 'Failed to find window' }`, gameExeName)

	// 	encoded := base64.StdEncoding.EncodeToString([]byte(activateScript))
	// 	activateSpec := &types.GuestProgramSpec{
	// 		ProgramPath: "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe",
	// 		Arguments:   fmt.Sprintf("-NoProfile -WindowStyle Hidden -EncodedCommand %s", encoded),
	// 	}

	// 	// 再次发起请求
	// 	_, err := methods.StartProgramInGuest(s.ctx, c, &types.StartProgramInGuest{
	// 		This: *guestOpsMgr.ProcessManager,
	// 		Vm:   vmObj.Reference(),
	// 		Auth: guestAuth,
	// 		Spec: activateSpec,
	// 	})

	// 	if err != nil {
	// 		applog.LogWarningf(s.ctx, "窗口激活脚本执行失败: %v", err)
	// 	}
	// }()

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

package utils

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"lunabox/internal/applog"
)

// 用于跟踪正在运行的ffmpeg进程
var ffmpegProcessMap = make(map[string]*exec.Cmd)
var ffmpegProcessMutex sync.Mutex

// getSupportedHardwareEncoder 检测支持的GPU硬编码器
func GetSupportedHardwareEncoder(ffmpegPath string) string {
	// 检查Intel QSV（优先检查，因为用户使用的是Intel核显）
	cmd := exec.Command(ffmpegPath, "-encoders")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,       // 隐藏窗口
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW 标志
	}
	output, err := cmd.CombinedOutput()
	if err == nil && strings.Contains(string(output), "h264_qsv") {
		return "h264_qsv"
	}

	// 检查NVIDIA CUDA
	cmd = exec.Command(ffmpegPath, "-encoders")
	output, err = cmd.CombinedOutput()
	if err == nil && strings.Contains(string(output), "h264_nvenc") {
		return "h264_nvenc"
	}

	// 检查AMD VCE
	cmd = exec.Command(ffmpegPath, "-encoders")
	output, err = cmd.CombinedOutput()
	if err == nil && strings.Contains(string(output), "h264_amf") {
		return "h264_amf"
	}

	// 检查Apple VideoToolbox (macOS)
	cmd = exec.Command(ffmpegPath, "-encoders")
	output, err = cmd.CombinedOutput()
	if err == nil && strings.Contains(string(output), "h264_videotoolbox") {
		return "h264_videotoolbox"
	}

	// 默认使用CPU编码
	return "h264"
}

// ConvertVideoWithFFmpeg 使用ffmpeg转换视频
func ConvertVideoWithFFmpeg(ctx context.Context, ffmpegPath, inputPath, outputPath, gameID string) error {
	// 首先尝试使用硬件编码
	encoder := GetSupportedHardwareEncoder(ffmpegPath)
	applog.LogInfof(ctx, "VideoStreamHandler: using encoder: %s", encoder)

	// 构建ffmpeg命令参数
	args := []string{
		"-y",
		"-i", inputPath,
		"-f", "mp4",
		"-vcodec", encoder,
		"-acodec", "aac",
		"-movflags", "frag_keyframe+empty_moov+faststart",
		outputPath,
		"-hide_banner",
	}

	// 如果使用硬件编码，添加相应的硬件加速参数
	if encoder != "h264" {
		switch encoder {
		case "h264_nvenc":
			// NVIDIA CUDA
			args = append([]string{"-hwaccel", "cuda"}, args...)
		case "h264_qsv":
			// Intel QSV
			args = append([]string{"-hwaccel", "qsv", "-hwaccel_output_format", "qsv"}, args...)
		case "h264_amf":
			// AMD VCE
			args = append([]string{"-hwaccel", "amf"}, args...)
		case "h264_videotoolbox":
			// Apple VideoToolbox
			args = append([]string{"-hwaccel", "videotoolbox"}, args...)
		}
	}

	// 创建ffmpeg命令
	cmd := exec.Command(ffmpegPath, args...)

	// 捕获标准错误输出
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,       // 隐藏窗口
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW 标志
	}

	// 将进程添加到映射中
	ffmpegProcessMutex.Lock()
	ffmpegProcessMap[gameID] = cmd
	ffmpegProcessMutex.Unlock()
	applog.LogInfof(ctx, "VideoStreamHandler: added ffmpeg process to map for gameID: %s", gameID)

	applog.LogInfof(ctx, "VideoStreamHandler: starting ffmpeg conversion with encoder: %s", encoder)
	err := cmd.Run()

	// 转换完成后从映射中删除进程
	ffmpegProcessMutex.Lock()
	delete(ffmpegProcessMap, gameID)
	ffmpegProcessMutex.Unlock()
	applog.LogInfof(ctx, "VideoStreamHandler: removed ffmpeg process from map for gameID: %s", gameID)

	// 如果硬件编码成功，直接返回
	if err == nil {
		applog.LogInfof(ctx, "VideoStreamHandler: hardware encoding succeeded")
		return nil
	}

	// 硬件编码失败，记录详细错误信息
	applog.LogWarningf(ctx, "VideoStreamHandler: hardware encoding failed, falling back to CPU encoding: %v", err)
	applog.LogWarningf(ctx, "VideoStreamHandler: hardware encoding error details: %s", stderr.String())

	// 构建CPU编码的ffmpeg命令参数
	cpuArgs := []string{
		"-y",
		"-i", inputPath,
		"-f", "mp4",
		"-vcodec", "h264",
		"-acodec", "aac",
		"-movflags", "frag_keyframe+empty_moov+faststart",
		outputPath,
		"-hide_banner",
	}

	// 创建ffmpeg命令
	cpuCmd := exec.Command(ffmpegPath, cpuArgs...)

	// 捕获标准错误输出
	var cpuStderr bytes.Buffer
	cpuCmd.Stderr = &cpuStderr

	// 将进程添加到映射中
	ffmpegProcessMutex.Lock()
	ffmpegProcessMap[gameID] = cpuCmd
	ffmpegProcessMutex.Unlock()
	applog.LogInfof(ctx, "VideoStreamHandler: added ffmpeg process to map for gameID: %s", gameID)

	applog.LogInfof(ctx, "VideoStreamHandler: starting ffmpeg conversion with CPU encoder")
	cpuCmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,       // 隐藏窗口
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW 标志
	}
	err = cpuCmd.Run()

	// 转换完成后从映射中删除进程
	ffmpegProcessMutex.Lock()
	delete(ffmpegProcessMap, gameID)
	ffmpegProcessMutex.Unlock()
	applog.LogInfof(ctx, "VideoStreamHandler: removed ffmpeg process from map for gameID: %s", gameID)

	if err != nil {
		applog.LogErrorf(ctx, "VideoStreamHandler: CPU encoding also failed: %v", err)
		applog.LogErrorf(ctx, "VideoStreamHandler: CPU encoding error details: %s", cpuStderr.String())
	} else {
		applog.LogInfof(ctx, "VideoStreamHandler: CPU encoding succeeded")
	}

	return err
}

// GetFFmpegProcessMap 获取ffmpeg进程映射
func GetFFmpegProcessMap() map[string]*exec.Cmd {
	return ffmpegProcessMap
}

// GetFFmpegProcessMutex 获取ffmpeg进程互斥锁
func GetFFmpegProcessMutex() *sync.Mutex {
	return &ffmpegProcessMutex
}

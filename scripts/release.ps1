# KaleidoBox Release Script
# 用法:
#   .\scripts\release.ps1                     # 版本号取自 sync/version.json，完整流程
#   .\scripts\release.ps1 -Version 1.4.0      # 指定版本号
#   .\scripts\release.ps1 -SkipBuild          # 跳过构建，直接用已有产物发布
#   .\scripts\release.ps1 -DryRun             # 只校验和预览，不构建/不推送/不发布
#
# 流程: 前置校验 -> 构建 portable (GUI+CLI+ZIP) -> 打 tag 并推送 -> 创建 GitHub Release 并上传产物
# 依赖: MSYS2 UCRT64 (go/wails)、git 凭据管理器中已存储 github.com 的 PAT

param(
    [string]$Version = "",
    [switch]$SkipBuild,
    [switch]$DryRun
)

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $RepoRoot

$RepoSlug = "gza21aliyun/kaleidobox"
$Branch = "kaleidobox"

# ---------- 0. 读取版本号 ----------
if (-not $Version) {
    $vj = Get-Content "sync/version.json" -Raw -Encoding UTF8 | ConvertFrom-Json
    $Version = $vj.version
}
$Tag = $Version   # 本仓库 tag 惯例: 不带 v 前缀 (1.2.0 / 1.3.1 ...)

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "KaleidoBox Release $Version" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

# ---------- 1. 前置校验 ----------
Write-Host "[1/4] 前置校验..." -ForegroundColor Yellow

# 版本一致性
$vjRaw = Get-Content "sync/version.json" -Raw -Encoding UTF8
if ($vjRaw -notmatch """version"":\s*""$Version""") { throw "sync/version.json 中版本与 $Version 不一致" }
$changelog = Get-Content "CHANGELOG.md" -Raw -Encoding UTF8
if ($changelog -notmatch "##\s*$([regex]::Escape($Version))") { throw "CHANGELOG.md 中缺少 $Version 章节" }

# 分支与工作区
$currentBranch = (git rev-parse --abbrev-ref HEAD).Trim()
if ($currentBranch -ne $Branch) { throw "当前分支为 $currentBranch，需要在 $Branch 上发布" }
$dirty = git status --porcelain | Where-Object { $_ -notmatch '^\?\?' }
if ($dirty) { throw "工作区有未提交的修改，请先提交:`n$dirty" }

# 构建产物检查 (SkipBuild 时)
$exePath = "build\bin\kaleidobox-portable.exe"
$zipPath = "build\bin\KaleidoBox-Portable-$Version.zip"
$cliPath = "build\bin\kaleidobox-cli.exe"
if ($SkipBuild) {
    foreach ($f in @($exePath, $cliPath)) {
        if (-not (Test-Path $f)) { throw "缺少 $f，无法跳过构建" }
    }
}

if ($DryRun) {
    Write-Host "(DryRun) 校验通过。将执行: 构建 -> tag $Tag -> 推送 $Branch 与 tag -> GitHub Release" -ForegroundColor Green
    return
}

# ---------- 2. 构建 ----------
Write-Host "[2/4] 构建 portable 版本 (GUI + CLI + ZIP)..." -ForegroundColor Yellow
if (-not $SkipBuild) {
    # MSYS2 UCRT64 工具链 (go 需要显式 GOROOT)
    $env:GOROOT = "C:\msys64\ucrt64\lib\go"
    $env:PATH = "C:\msys64\ucrt64\bin;C:\msys64\home\$env:USERNAME\go\bin;$env:PATH"

    & cmd /c "scripts\build.bat portable $Version"
    if ($LASTEXITCODE -ne 0) { throw "构建失败" }
}
if (-not (Test-Path $exePath)) { throw "构建产物缺失: $exePath" }
if (-not (Test-Path $cliPath)) { throw "构建产物缺失: $cliPath" }

# 打包 portable ZIP (PowerShell 完成，避免 build.bat 在部分环境下打包失败)
if (-not (Test-Path $zipPath)) {
    Write-Host "打包 $zipPath ..."
    $pkgDir = "build\bin\KaleidoBox-Portable-$Version"
    if (Test-Path $pkgDir) { Remove-Item $pkgDir -Recurse -Force }
    New-Item $pkgDir -ItemType Directory | Out-Null
    foreach ($d in @("backups", "covers", "backgrounds", "logs")) { New-Item "$pkgDir\$d" -ItemType Directory | Out-Null }
    Copy-Item $exePath "$pkgDir\KaleidoBox.exe"
    Copy-Item $cliPath "$pkgDir\kaleidocli.exe"
    @"
KaleidoBox Portable v$Version

This package contains:
  - KaleidoBox.exe  : GUI version (Double-click to launch)
  - kaleidocli.exe  : CLI version (Use in terminal)

CLI Usage:
  kaleidocli list
  kaleidocli start <game-id>
  kaleidocli help
"@ | Set-Content "$pkgDir\README.txt" -Encoding UTF8
    # AV 可能短暂锁定新复制的 exe，重试 3 次
    for ($i = 1; $i -le 3; $i++) {
        try {
            if (Test-Path $zipPath) { Remove-Item $zipPath -Force }
            Compress-Archive -Path $pkgDir -DestinationPath $zipPath -Force -ErrorAction Stop
            break
        } catch {
            if ($i -eq 3) { throw "打包 ZIP 失败: $_" }
            Write-Host "打包被占用，2 秒后重试 ($i/3)..."
            Start-Sleep -Seconds 2
        }
    }
    Remove-Item $pkgDir -Recurse -Force
}
Write-Host "构建完成: $exePath / $zipPath" -ForegroundColor Green

# ---------- 3. 打 tag 并推送 ----------
Write-Host "[3/4] 打 tag 并推送..." -ForegroundColor Yellow

# PS5.1 在 EAP=Stop 下会把 git 的 stderr 进度输出包装成终止错误，函数内临时降级处理
function Invoke-GitSafe {
    param([string[]]$GitArgs)
    $ErrorActionPreference = "Continue"
    try { $out = & git @GitArgs 2>&1 | ForEach-Object { "$_" }; $code = $LASTEXITCODE }
    finally { $ErrorActionPreference = "Stop" }
    Write-Host ($out -join "`n")
    if ($code -ne 0) { throw "git $($GitArgs -join ' ') 失败" }
}

Invoke-GitSafe @("push", "origin", $Branch)

git tag -f $Tag
if ($LASTEXITCODE -ne 0) { throw "打 tag 失败" }
Invoke-GitSafe @("push", "--force", "origin", "refs/tags/${Tag}:refs/tags/${Tag}")
Write-Host "已推送 $Branch 与 tag $Tag" -ForegroundColor Green

# ---------- 4. 创建 GitHub Release ----------
Write-Host "[4/4] 创建 GitHub Release..." -ForegroundColor Yellow

# 从凭据管理器获取 PAT (临时文件 + cmd 重定向; .NET 直写 stdin 在部分上下文会被 git 判定缺少 protocol 字段)
$tmpCredIn = Join-Path $env:TEMP "kaleidobox_cred_in.txt"
[System.IO.File]::WriteAllBytes($tmpCredIn, [System.Text.Encoding]::ASCII.GetBytes("protocol=https`r`nhost=github.com`r`n`r`n"))
$psi = New-Object System.Diagnostics.ProcessStartInfo
$psi.FileName = "cmd.exe"
$psi.Arguments = "/c git credential fill < `"$tmpCredIn`""
$psi.UseShellExecute = $false
$psi.RedirectStandardOutput = $true
$psi.RedirectStandardError = $true
$proc = [System.Diagnostics.Process]::Start($psi)
$credOut = $proc.StandardOutput.ReadToEnd()
$proc.WaitForExit()
$errText = $proc.StandardError.ReadToEnd()
Remove-Item $tmpCredIn -Force -ErrorAction SilentlyContinue
if ($proc.ExitCode -ne 0) { throw "git credential fill 失败: $errText" }
$token = ($credOut -split "`n" | Where-Object { $_ -like "password=*" }) -replace "^password=", ""
if (-not $token) { throw "未能从 git 凭据管理器获取 github.com 的 token: $errText" }

$headers = @{
    Authorization = "token $token"
    "User-Agent"  = "kaleidobox-release"
    Accept        = "application/vnd.github+json"
}

# 提取 CHANGELOG 中该版本的章节作为 release notes
$start = $changelog.IndexOf("## $Version")
if ($start -lt 0) { throw "CHANGELOG.md 中未找到 $Version 章节" }
$next = $changelog.IndexOf("`n## ", $start + 1)
if ($next -gt 0) { $body = $changelog.Substring($start, $next - $start).Trim() } else { $body = $changelog.Substring($start).Trim() }

$commitSha = (git rev-parse HEAD).Trim()
$releasePayload = @{
    tag_name         = $Tag
    target_commitish = $commitSha
    name             = $Version
    body             = $body
    draft            = $false
    prerelease       = $false
} | ConvertTo-Json

# 已存在同 tag release 则复用，否则新建
$release = $null
$existing = Invoke-RestMethod -Uri "https://api.github.com/repos/$RepoSlug/releases/tags/$Tag" -Headers $headers -ErrorAction SilentlyContinue
if ($existing -and $existing.id) {
    Write-Host "Release $Tag 已存在 (id=$($existing.id))，更新说明..."
    $updatePayload = @{ name = $Version; body = $body; draft = $false; prerelease = $false } | ConvertTo-Json
    $release = Invoke-RestMethod -Method Patch -Uri "https://api.github.com/repos/$RepoSlug/releases/$($existing.id)" -Headers $headers -Body $updatePayload -ContentType "application/json"
} else {
    $release = Invoke-RestMethod -Method Post -Uri "https://api.github.com/repos/$RepoSlug/releases" -Headers $headers -Body $releasePayload -ContentType "application/json"
}

# 上传产物 (重名先删旧资产)
$assets = @(
    @{ Path = $exePath; Name = "KaleidoBox.exe" },
    @{ Path = $zipPath; Name = "KaleidoBox-Portable-$Version.zip" }
)
foreach ($a in $assets) {
    $old = $release.assets | Where-Object { $_.name -eq $a.Name }
    if ($old) {
        Write-Host "删除旧资产 $($a.Name)..."
        Invoke-RestMethod -Method Delete -Uri "https://api.github.com/repos/$RepoSlug/releases/assets/$($old.id)" -Headers $headers | Out-Null
    }
    $size = [math]::Round((Get-Item $a.Path).Length / 1MB, 1)
    Write-Host "上传 $($a.Name) ($size MB)..."
    Invoke-RestMethod -Method Post `
        -Uri "https://uploads.github.com/repos/$RepoSlug/releases/$($release.id)/assets?name=$($a.Name)" `
        -Headers $headers -ContentType "application/octet-stream" -InFile $a.Path | Out-Null
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "Release $Version 发布完成!" -ForegroundColor Green
Write-Host "https://github.com/$RepoSlug/releases/tag/$Tag" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green

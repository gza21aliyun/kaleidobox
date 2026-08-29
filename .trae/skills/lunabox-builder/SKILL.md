---
name: "kaleidobox-builder"
description: "Builds and compiles the KaleidoBox Wails Go+React project in the MSYS2 UCRT64 environment. Invoke when user wants to build, compile, package, or run wails commands for the KaleidoBox project."
---

# KaleidoBox Builder

This skill provides the correct MSYS2 UCRT64 build environment for the KaleidoBox Wails project located at `C:\temp\projects\lunabox`.

## Project Information

- **Project Type**: Wails v2 application (Go backend + React frontend)
- **Project Path**: `C:\temp\projects\lunabox`
- **Operating System**: Windows
- **Build Toolchain**: MSYS2 UCRT64

## Critical Paths

| Component | Windows Path | MSYS2 Path |
|-----------|-------------|------------|
| MSYS2 Installation | `C:\msys64` | `/` |
| UCRT64 Shell | `C:\msys64\ucrt64.exe` | - |
| Bash | `C:\msys64\usr\bin\bash.exe` | `/usr/bin/bash.exe` |
| Go Installation | `C:\msys64\ucrt64\lib\go` | `/ucrt64/lib/go` |
| Go Executable | `C:\msys64\ucrt64\bin\go.exe` | `/ucrt64/bin/go.exe` |
| Wails CLI | `C:\msys64\home\osk666\go\bin\wails.exe` | `/home/osk666/go/bin/wails` |

## Required Environment Variables

Before running any Go or Wails command inside MSYS2 bash, you MUST set:

```bash
export GOROOT=/ucrt64/lib/go
export PATH="/ucrt64/bin:/usr/local/bin:/usr/bin:/bin:/home/osk666/go/bin:$PATH"
```

## Build Commands

### Full Wails Build (Production)

```bash
C:\msys64\usr\bin\bash.exe -lc 'export GOROOT=/ucrt64/lib/go && export PATH="/ucrt64/bin:/usr/local/bin:/usr/bin:/bin:/home/osk666/go/bin:$PATH" && cd /c/temp/projects/lunabox && wails build 2>&1 | tee /c/temp/projects/lunabox/build.log'
```

### Wails Dev Server

```bash
C:\msys64\usr\bin\bash.exe -lc 'export GOROOT=/ucrt64/lib/go && export PATH="/ucrt64/bin:/usr/local/bin:/usr/bin:/bin:/home/osk666/go/bin:$PATH" && cd /c/temp/projects/lunabox && wails dev'
```

### Verify Environment

```bash
C:\msys64\usr\bin\bash.exe -lc 'export GOROOT=/ucrt64/lib/go && export PATH="/ucrt64/bin:/usr/local/bin:/usr/bin:/bin:/home/osk666/go/bin:$PATH" && which wails && wails version && which go && go version'
```

## Output Artifacts

After a successful build, the binaries are located at:

- **Production binary**: `build/bin/KaleidoBox.exe`
- **Development binary**: `build/bin/KaleidoBox-dev.exe`

## Important Rules

1. **Always set `GOROOT` first**: The `/ucrt64/bin/go.exe` binary is trimmed and will fail with `cannot find GOROOT directory` if `GOROOT` is not set.
2. **Use MSYS2 bash for builds**: Do not run `wails build` directly in the default PowerShell/Command Prompt environment; the PATH and GOROOT settings will not be correct.
3. **Path conversion**: When using MSYS2 bash, Windows paths like `C:\temp\projects\lunabox` become `/c/temp/projects/lunabox`.
4. **Do not assume tools are in PATH**: Always explicitly export the PATH and GOROOT before running any build command.

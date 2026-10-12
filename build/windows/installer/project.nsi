Unicode true
!include "wails_tools.nsh"

VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion "${INFO_PRODUCTVERSION}.0"
VIAddVersionKey "CompanyName" "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion" "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright" "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName" "${INFO_PRODUCTNAME}"

ManifestDPIAware true
!include "MUI.nsh"
!define MUI_ABORTWARNING
!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_FINISHPAGE_RUN "$INSTDIR\DownTube.exe"
!define MUI_FINISHPAGE_RUN_TEXT "启动 DownTube"
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "SimpChinese"

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\bin\DownTube-Setup.exe"
!ifdef WAILS_INSTALL_SCOPE
  !if "${WAILS_INSTALL_SCOPE}" == "user"
    InstallDir "$LOCALAPPDATA\Programs\${INFO_PRODUCTNAME}"
  !else
    InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
  !endif
!else
  InstallDir "$LOCALAPPDATA\Programs\${INFO_PRODUCTNAME}"
!endif
ShowInstDetails show

Function .onInit
  !insertmacro wails.checkArchitecture
FunctionEnd

Section
  !insertmacro wails.setShellContext

  ; ---- 覆盖安装：先结束正在运行的旧版本 ----
  ; 应用内自动更新时旧版本会自行退出，这里兜底等待它完全退出；
  ; 手动安装时若程序正开着，先请求优雅关闭，超时后强制结束，
  ; 否则下面的 File 会因 DownTube.exe 被占用而写入失败。
  nsExec::Exec 'taskkill /IM "${PRODUCT_EXECUTABLE}"'
  Pop $0
  StrCpy $1 0
  dt_wait_loop:
    Sleep 500
    nsExec::ExecToStack 'cmd /C tasklist /FI "IMAGENAME eq ${PRODUCT_EXECUTABLE}" /NH | find /I "${PRODUCT_EXECUTABLE}"'
    Pop $0
    Pop $2
    StrCmp $0 "0" 0 dt_wait_done
    IntOp $1 $1 + 1
    IntCmp $1 16 dt_wait_timeout dt_wait_loop dt_wait_loop
  dt_wait_timeout:
    nsExec::Exec 'taskkill /F /IM "${PRODUCT_EXECUTABLE}"'
    Pop $0
    Sleep 800
  dt_wait_done:

  !insertmacro wails.webview2runtime
  SetOutPath $INSTDIR
  !insertmacro wails.files
  File "..\..\..\tools\yt-dlp.exe"
  CreateDirectory "$SMPROGRAMS\DownTube"
  CreateShortcut "$SMPROGRAMS\DownTube\DownTube.lnk" "$INSTDIR\DownTube.exe"
  CreateShortcut "$DESKTOP\DownTube.lnk" "$INSTDIR\DownTube.exe"
  !insertmacro wails.writeUninstaller

  ; ---- 静默安装（应用内自动更新）完成后自动重启新版 ----
  ; 经 explorer 启动，避免新版继承安装程序的管理员权限；
  ; 交互安装不自动启动，由完成页的「启动 DownTube」勾选处理。
  IfSilent dt_launch_app dt_section_end
  dt_launch_app:
    Exec '"$WINDIR\explorer.exe" "$INSTDIR\DownTube.exe"'
  dt_section_end:
SectionEnd

Section "uninstall"
  !insertmacro wails.setShellContext
  RMDir /r "$AppData\DownTube.exe"
  RMDir /r "$INSTDIR"
  Delete "$SMPROGRAMS\DownTube\DownTube.lnk"
  RMDir "$SMPROGRAMS\DownTube"
  Delete "$DESKTOP\DownTube.lnk"
  !insertmacro wails.deleteUninstaller
SectionEnd

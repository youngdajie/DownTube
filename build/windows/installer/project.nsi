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
  !insertmacro wails.webview2runtime
  SetOutPath $INSTDIR
  !insertmacro wails.files
  File "..\..\..\tools\yt-dlp.exe"
  CreateDirectory "$SMPROGRAMS\DownTube"
  CreateShortcut "$SMPROGRAMS\DownTube\DownTube.lnk" "$INSTDIR\DownTube.exe"
  CreateShortcut "$DESKTOP\DownTube.lnk" "$INSTDIR\DownTube.exe"
  !insertmacro wails.writeUninstaller
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

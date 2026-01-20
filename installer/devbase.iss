; DevBase Windows Installer Script
; This script creates a Windows installer that automatically adds DevBase to PATH

#define MyAppName "DevBase"
#define MyAppVersion "1.7.16"
#define MyAppPublisher "Maleesha Pramud"
#define MyAppURL "https://github.com/maleesha-pramud/devbase"
#define MyAppExeName "devbase.exe"

[Setup]
; NOTE: The value of AppId uniquely identifies this application
AppId={{8F9A2E3C-1B4D-4E5F-9A6B-7C8D9E0F1A2B}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}/issues
AppUpdatesURL={#MyAppURL}/releases
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
AllowNoIcons=yes
LicenseFile=..\LICENSE
OutputDir=..\dist
OutputBaseFilename=devbase-windows-installer
; SetupIconFile is optional - comment out if icon doesn't exist
; SetupIconFile=..\assets\icon.ico
Compression=lzma
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog
ChangesEnvironment=yes
; UninstallDisplayIcon={app}\{#MyAppExeName}

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked
Name: "addtopath"; Description: "Add to PATH (recommended)"; GroupDescription: "System Integration:"; Flags: checkedonce

[Files]
Source: "..\devbase.exe"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{group}\{cm:UninstallProgram,{#MyAppName}}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[Code]
const
    EnvironmentKey = 'SYSTEM\CurrentControlSet\Control\Session Manager\Environment';

procedure EnvAddPath(Path: string);
var
    Paths: string;
begin
    { Retrieve current path (use empty string if entry not exists) }
    if not RegQueryStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', Paths) then
        Paths := '';

    { Skip if already in path }
    if Pos(';' + Uppercase(Path) + ';', ';' + Uppercase(Paths) + ';') > 0 then exit;

    { App string to the end of the path variable }
    if Paths = '' then
        Paths := Path
    else
        Paths := Paths + ';' + Path;

    { Overwrite (or create if missing) path environment variable }
    if RegWriteStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', Paths) then
        Log(Format('The [%s] added to PATH: [%s]', [Path, Paths]))
    else
        Log(Format('Error while adding the [%s] to PATH: [%s]', [Path, Paths]));
end;

procedure EnvRemovePath(Path: string);
var
    Paths: string;
    P: Integer;
begin
    { Skip if registry entry not exists }
    if not RegQueryStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', Paths) then
        exit;

    { Skip if already not in path }
    P := Pos(';' + Uppercase(Path) + ';', ';' + Uppercase(Paths) + ';');
    if P = 0 then exit;

    { Remove from path }
    Delete(Paths, P - 1, Length(Path) + 1);

    { Overwrite path environment variable }
    if RegWriteStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', Paths) then
        Log(Format('The [%s] removed from PATH: [%s]', [Path, Paths]))
    else
        Log(Format('Error while removing the [%s] from PATH: [%s]', [Path, Paths]));
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
    if CurStep = ssPostInstall then
    begin
        if WizardIsTaskSelected('addtopath') then
            EnvAddPath(ExpandConstant('{app}'));
    end;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
    if CurUninstallStep = usPostUninstall then
        EnvRemovePath(ExpandConstant('{app}'));
end;

[Run]
Filename: "{cmd}"; Parameters: "/C echo Installation complete! Please restart your terminal to use 'devbase' command."; Flags: postinstall nowait skipifsilent runhidden

[UninstallDelete]
Type: filesandordirs; Name: "{%USERPROFILE}\devbase.db"

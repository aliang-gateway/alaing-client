# Build MSI installer for Windows using WiX Toolset
# This script should be run on Windows with WiX installed

param(
    [string]$Version = "1.0.0",
    [string]$OutputDir = "."
)

$ErrorActionPreference = "Stop"

$BINARY_NAME = "aliang.exe"
$SERVICE_NAME = "aliang"
$MANUFACTURER = "Aliang"
$UPGRADE_CODE = "A1B2C3D4-E5F6-7890-ABCD-EF1234567890"  # Should be generated once per product
$ICON_FILE = "desktop-logo.ico"

Write-Host "=== Building Aliang MSI Installer ===" -ForegroundColor Cyan
Write-Host "Version: $Version"
Write-Host "Output: $OutputDir"

# Check if WiX is installed
$wixPath = $null
if (Test-Path "C:\Program Files (x86)\WiX Toolset v3.11\bin\candle.exe") {
    $wixPath = "C:\Program Files (x86)\WiX Toolset v3.11\bin"
} elseif (Test-Path "C:\Program Files (x86)\WiX Toolset v3\bin\candle.exe") {
    $wixPath = "C:\Program Files (x86)\WiX Toolset v3\bin"
} elseif (Get-Command candle.exe -ErrorAction SilentlyContinue) {
    $wixPath = Split-Path (Get-Command candle.exe).Source
}

if (-not $wixPath) {
    Write-Host "WiX Toolset not found. Installing via NuGet..." -ForegroundColor Yellow
    # Download WiX via NuGet
    $nugetExe = "$env:TEMP\nuget.exe"
    if (-not (Test-Path $nugetExe)) {
        Invoke-WebRequest -Uri "https://dist.nuget.org/win-x86-commandline/latest/nuget.exe" -OutFile $nugetExe
    }

    $wixPath = "$env:TEMP\wix"
    New-Item -ItemType Directory -Force -Path $wixPath | Out-Null
    & $nugetExe install WiX -Version 3.11.2 -OutputDirectory $wixPath -NoCache

    $wixPath = "$wixPath\wix\WiX.Toolset.3.11.2\tools"
}

Write-Host "Using WiX from: $wixPath" -ForegroundColor Green

# Create build directory
$buildDir = "$env:TEMP\aliang-msi-build"
$sourceDir = "$buildDir\source"
$payloadDir = "$buildDir\payload"

# Clean build directory
Remove-Item -Path $buildDir -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $sourceDir | Out-Null
New-Item -ItemType Directory -Force -Path $payloadDir | Out-Null

# Copy binary and icon
Write-Host "Copying binary and icon..." -ForegroundColor Cyan
Copy-Item ".\dist\$BINARY_NAME" -Destination "$payloadDir\" -Force
if (Test-Path ".\$ICON_FILE") {
    Copy-Item ".\$ICON_FILE" -Destination "$payloadDir\" -Force
} else {
    Write-Host "Warning: Icon file $ICON_FILE not found, shortcuts will use default icon" -ForegroundColor Yellow
}

# Create WiX source file
Write-Host "Creating WiX source..." -ForegroundColor Cyan

$wxsContent = @"
<?xml version="1.0" encoding="UTF-8"?>
<Wix xmlns="http://schemas.microsoft.com/wix/2006/wi">
    <Product Id="*" Name="Aliang" Language="1033" Version="$Version" Manufacturer="$MANUFACTURER" UpgradeCode="$UPGRADE_CODE">
        <Package InstallerVersion="200" Compressed="yes" InstallScope="perMachine" Description="Aliang Gateway Proxy Client" Manufacturer="$MANUFACTURER"/>

        <MajorUpgrade DowngradeErrorMessage="A newer version of [ProductName] is already installed."/>
        <MediaTemplate EmbedCab="yes"/>

        <!-- Icon Definition -->
        <Icon Id="AliangIcon" SourceFile="$payloadDir\$ICON_FILE"/>
        <Property Id="ARPPRODUCTICON" Value="AliangIcon"/>

        <!-- Directory Structure -->
        <Directory Id="TARGETDIR" Name="SourceDir">
            <Directory Id="ProgramFilesFolder">
                <Directory Id="INSTALLFOLDER" Name="Aliang">
                    <Component Id="MainBinary" Guid="*">
                        <File Id="Aliangexe" Source="$payloadDir\$BINARY_NAME" KeyPath="yes"/>
                        <RegistryValue Root="HKLM" Key="Software\Aliang" Name="InstallDir" Type="string" Value="[INSTALLFOLDER]" KeyPath="yes"/>
                    </Component>
                </Directory>
            </Directory>
            <Directory Id="CommonAppDataFolder" Name="CommonAppData">
                <Directory Id="AliangData" Name="Aliang">
                    <Component Id="DataDirectory" Guid="*">
                        <CreateFolder/>
                        <RegistryValue Root="HKLM" Key="Software\Aliang" Name="DataDir" Type="string" Value="[AliangData]" KeyPath="yes"/>
                    </Component>
                </Directory>
            </Directory>
        </Directory>

        <!-- Features -->
        <Feature Id="ProductFeature" Title="Aliang" Level="1">
            <ComponentRef Id="MainBinary"/>
            <ComponentRef Id="DataDirectory"/>
            <ComponentRef Id="StartMenuShortcut"/>
            <ComponentRef Id="DesktopShortcut"/>
        </Feature>

        <!-- Environment Variables -->
        <Environment Id="ALIANG_DATA_DIR" Name="ALIANG_DATA_DIR" Value="[AliangData]" Permanent="yes" Part="last" Action="set" System="yes"/>
        <Environment Id="ALIANG_LOG_DIR" Name="ALIANG_LOG_DIR" Value="[AliangData]\logs" Permanent="yes" Part="last" Action="set" System="yes"/>
        <Environment Id="ALIANG_SOCKET_PATH" Name="ALIANG_SOCKET_PATH" Value="%PROGRAMDATA%\Aliang\aliang-core.sock" Permanent="yes" Part="last" Action="set" System="yes"/>

        <!-- Start Menu Shortcuts -->
        <Directory Id="ProgramMenuFolder">
            <Directory Id="ApplicationProgramsFolder" Name="Aliang">
                <Component Id="StartMenuShortcut" Guid="*">
                    <Shortcut Id="ApplicationStartMenuShortcut"
                              Name="Aliang"
                              Description="Aliang Gateway Proxy Client"
                              Target="[INSTALLFOLDER]$BINARY_NAME"
                              WorkingDirectory="INSTALLFOLDER"
                              Icon="AliangIcon"/>
                    <RemoveFolder Id="CleanUpShortCut" On="uninstall"/>
                    <RegistryValue Root="HKCU" Key="Software\Aliang" Name="installed" Type="integer" Value="1" KeyPath="yes"/>
                </Component>
            </Directory>
        </Directory>

        <!-- Desktop Shortcut -->
        <Directory Id="DesktopFolder" Name="Desktop">
            <Component Id="DesktopShortcut" Guid="*">
                <Shortcut Id="DesktopShortcut"
                          Name="Aliang"
                          Description="Aliang Gateway Proxy Client"
                          Target="[INSTALLFOLDER]$BINARY_NAME"
                          WorkingDirectory="INSTALLFOLDER"
                          Icon="AliangIcon"/>
                <RegistryValue Root="HKCU" Key="Software\Aliang" Name="DesktopShortcut" Type="integer" Value="1" KeyPath="yes"/>
            </Component>
        </Directory>

        <!-- Service Registration Custom Action -->
        <CustomAction Id="RegisterService"
                      Directory="INSTALLFOLDER"
                      ExeCommand="[INSTALLFOLDER]$BINARY_NAME service install --system-wide"
                      Return="ignore"
                      Execute="deferred"
                      Impersonate="no"/>
        <CustomAction Id="UnregisterService"
                      Directory="INSTALLFOLDER"
                      ExeCommand="[INSTALLFOLDER]$BINARY_NAME service uninstall --system-wide"
                      Return="ignore"
                      Execute="deferred"
                      Impersonate="no"/>

        <!-- Install Execute Sequence -->
        <InstallExecuteSequence>
            <Custom Action="RegisterService" After="InstallFinalize">NOT Installed</Custom>
            <Custom Action="UnregisterService" Before="RemoveFiles">REMOVE="ALL"</Custom>
        </InstallExecuteSequence>
    </Product>
</Wix>
"@

$wxsPath = "$sourceDir\aliang.wxs"
$wxsContent | Out-File -FilePath $wxsPath -Encoding UTF8

# Build MSI using WiX
Write-Host "Building MSI..." -ForegroundColor Cyan
Push-Location $sourceDir

try {
    # Compile WiX source
    Write-Host "Compiling WiX source..." -ForegroundColor Yellow
    & "$wixPath\candle.exe" -nologo -ext WixUIExtension -out "$sourceDir\aliang.wixobj" "$wxsPath"

    # Link/Combine into MSI
    Write-Host "Linking into MSI..." -ForegroundColor Yellow
    & "$wixPath\light.exe" -nologo -ext WixUIExtension -o "$OutputDir\aliang-$Version.msi" "$sourceDir\aliang.wixobj"

    Write-Host "MSI created successfully!" -ForegroundColor Green
    Write-Host "Output: $OutputDir\aliang-$Version.msi" -ForegroundColor Cyan
}
catch {
    Write-Host "Error building MSI: $_" -ForegroundColor Red
    throw
}
finally {
    Pop-Location
}

# Cleanup
Remove-Item -Path "$buildDir" -Recurse -Force -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "=== Build Complete ===" -ForegroundColor Cyan
Write-Host "MSI Installer: $OutputDir\aliang-$Version.msi"
Write-Host ""
Write-Host "Note: To install, run: msiexec /i aliang-$Version.msi"

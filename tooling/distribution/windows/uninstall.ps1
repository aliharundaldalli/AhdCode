# Removes the per-user AhdCode installation. Projects, databases, the local
# registry and build caches live outside the installation root and are kept.
$ErrorActionPreference = 'Stop'
$root = Join-Path $env:LOCALAPPDATA 'AhdCode'
if (!(Test-Path (Join-Path $root '.ahdcode-install'))) { throw 'No AhdCode-owned installation found.' }
if ((Get-Item $root).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'Refusing a linked installation root.' }

$answer = Read-Host 'Remove AhdCode installed files? Projects, databases and caches stay untouched. Type YES'
if ($answer -cne 'YES') { exit }

# Edit the user PATH in place: drop only AhdCode's own bin folder and keep every
# other entry exactly as it was, including empty and unexpanded ones.
$bin = (Join-Path $root 'bin')
$normalize = {
    param($value)
    if ($null -eq $value) { return '' }
    $trimmed = $value.Trim().Trim('"').Replace('/', '\').TrimEnd('\')
    try { [Environment]::ExpandEnvironmentVariables($trimmed).TrimEnd('\') } catch { $trimmed }
}
$target = & $normalize $bin
$path = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($null -ne $path) {
    $kept = @($path -split ';' | Where-Object { (& $normalize $_) -ine $target })
    $updated = ($kept -join ';')
    if ($updated -cne $path) { [Environment]::SetEnvironmentVariable('Path', $updated, 'User') }
}

Remove-Item 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\AhdCode' -Recurse -ErrorAction SilentlyContinue
Remove-Item -LiteralPath $root -Recurse -Force

# Tell running programs the environment changed, so a terminal opened after this
# no longer lists the removed folder.
if (-not ('AhdCodeEnv' -as [type])) {
    Add-Type -Name AhdCodeEnv -Namespace Win32 -MemberDefinition @'
[System.Runtime.InteropServices.DllImport("user32.dll", SetLastError = true, CharSet = System.Runtime.InteropServices.CharSet.Auto)]
public static extern System.IntPtr SendMessageTimeout(System.IntPtr hWnd, uint Msg, System.IntPtr wParam, string lParam, uint fuFlags, uint uTimeout, out System.UIntPtr lpdwResult);
'@
}
$result = [UIntPtr]::Zero
[void][Win32.AhdCodeEnv]::SendMessageTimeout([IntPtr]0xffff, 0x1A, [IntPtr]::Zero, 'Environment', 2, 5000, [ref]$result)

Write-Host 'AhdCode removed. User projects, databases, registry data and caches were preserved.'
Write-Host 'Open a new terminal for the PATH change to take effect.'

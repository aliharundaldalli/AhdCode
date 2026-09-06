$ErrorActionPreference = 'Stop'
$root = Join-Path $env:LOCALAPPDATA 'AhdCode'
if (!(Test-Path (Join-Path $root '.ahdcode-install'))) { throw 'No AhdCode-owned installation found.' }
if ((Get-Item $root).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'Refusing a linked installation root.' }
$answer = Read-Host 'Remove AhdCode installed files? Projects, databases and caches stay untouched. Type YES'
if ($answer -cne 'YES') { exit }
$bin = Join-Path $root 'bin'
$path = [Environment]::GetEnvironmentVariable('Path', 'User')
$parts = @($path.Split(';') | Where-Object { $_.TrimEnd('\') -ine $bin })
[Environment]::SetEnvironmentVariable('Path', ($parts -join ';'), 'User')
Remove-Item 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\AhdCode' -Recurse -ErrorAction SilentlyContinue
Remove-Item -LiteralPath $root -Recurse -Force
Write-Host 'AhdCode removed. User projects, databases, registry data and caches were preserved.'

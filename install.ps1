# Installs Blup on Windows (PowerShell):
#   irm https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.ps1 | iex
# Optional: $env:BLUP_INSTALL_DIR (default %LOCALAPPDATA%\Programs\blup), $env:BLUP_VERSION.
$ErrorActionPreference = "Stop"
$repo = "gabriellsalesx/blup"
$dir = if ($env:BLUP_INSTALL_DIR) { $env:BLUP_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\blup" }

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
  "AMD64" { "amd64" }
  "ARM64" { "arm64" }
  default { throw "blup: unsupported processor / processador não suportado: $env:PROCESSOR_ARCHITECTURE" }
}

$version = $env:BLUP_VERSION
if (-not $version) {
  $version = (Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest").tag_name
}
$version = $version.TrimStart("v")

$asset = "blup_${version}_windows_${arch}.zip"
$base = "https://github.com/$repo/releases/download/v$version"
$tmp = Join-Path ([IO.Path]::GetTempPath()) ("blup-" + [Guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
  Write-Host "Downloading Blup $version (windows/$arch)..."
  Invoke-WebRequest "$base/$asset" -OutFile (Join-Path $tmp $asset)
  Invoke-WebRequest "$base/checksums.txt" -OutFile (Join-Path $tmp "checksums.txt")

  $line = Get-Content (Join-Path $tmp "checksums.txt") | Where-Object { $_ -match " $([regex]::Escape($asset))$" }
  $want = ($line -split "\s+")[0]
  $got = (Get-FileHash (Join-Path $tmp $asset) -Algorithm SHA256).Hash.ToLower()
  if (-not $want -or $want -ne $got) { throw "blup: checksum mismatch, nothing installed / verificação falhou, nada foi instalado" }

  Expand-Archive (Join-Path $tmp $asset) -DestinationPath $tmp
  New-Item -ItemType Directory -Force -Path $dir | Out-Null
  Copy-Item (Join-Path $tmp "blup.exe") (Join-Path $dir "blup.exe") -Force
} finally {
  Remove-Item -Recurse -Force $tmp
}

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (($userPath -split ";") -notcontains $dir) {
  [Environment]::SetEnvironmentVariable("Path", "$userPath;$dir", "User")
  Write-Host "Added $dir to your PATH. Open a new terminal. / Adicionado ao PATH. Abra um terminal novo."
}
Write-Host "Blup $version installed. Run: blup"

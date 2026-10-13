# Install gitix on Windows (PowerShell):
#   irm https://raw.githubusercontent.com/hbeanjarah/gitix/main/install.ps1 | iex
# Options (environment variables):
#   GITIX_VERSION      tag to install, e.g. v0.1.0 (default: newest release, pre-releases included)
#   GITIX_INSTALL_DIR  where to put gitix.exe (default: %LOCALAPPDATA%\Programs\gitix)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue" # makes Invoke-WebRequest much faster on Windows PowerShell
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$repo = "hbeanjarah/gitix"
$dir = if ($env:GITIX_INSTALL_DIR) { $env:GITIX_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\gitix" }

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { "amd64" }
    "ARM64" { "arm64" }
    default { throw "gitix install: unsupported architecture $env:PROCESSOR_ARCHITECTURE" }
}

if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Write-Warning "git is not installed; gitix needs git 2.23 or newer (https://git-scm.com/downloads)."
}

$tag = $env:GITIX_VERSION
if (-not $tag) {
    $releases = Invoke-RestMethod "https://api.github.com/repos/$repo/releases?per_page=1"
    $tag = @($releases)[0].tag_name
    if (-not $tag) { throw "gitix install: cannot find the latest release" }
}
$version = $tag.TrimStart("v")
$archive = "gitix_${version}_windows_${arch}.zip"
$base = "https://github.com/$repo/releases/download/$tag"

$tmp = Join-Path ([IO.Path]::GetTempPath()) ("gitix-" + [Guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
    Write-Host "Downloading gitix $tag (windows/$arch)..."
    $zip = Join-Path $tmp $archive
    Invoke-WebRequest "$base/$archive" -OutFile $zip -UseBasicParsing
    $sums = (Invoke-WebRequest "$base/checksums.txt" -UseBasicParsing).Content
    if ($sums -is [byte[]]) { $sums = [Text.Encoding]::UTF8.GetString($sums) }

    $line = $sums -split "`n" | Where-Object { $_.Trim().EndsWith(" $archive") } | Select-Object -First 1
    $expected = if ($line) { ($line -split "\s+")[0] } else { "" }
    $actual = (Get-FileHash $zip -Algorithm SHA256).Hash
    if (-not $expected -or $expected.ToLower() -ne $actual.ToLower()) {
        throw "gitix install: checksum mismatch for $archive"
    }

    Expand-Archive $zip -DestinationPath $tmp -Force
    New-Item -ItemType Directory -Path $dir -Force | Out-Null
    Copy-Item (Join-Path $tmp "gitix.exe") (Join-Path $dir "gitix.exe") -Force
} finally {
    Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
}

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (-not $userPath) { $userPath = "" }
if (($userPath -split ";") -notcontains $dir) {
    [Environment]::SetEnvironmentVariable("Path", ($userPath.TrimEnd(";") + ";" + $dir).TrimStart(";"), "User")
}
if (($env:Path -split ";") -notcontains $dir) {
    $env:Path = $env:Path.TrimEnd(";") + ";" + $dir
}

Write-Host "Installed $(& (Join-Path $dir 'gitix.exe') --version) in $dir"
Write-Host "Run: gitix --help (open a new terminal if the command is not found)"

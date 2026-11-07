# Install goh on Windows:
#   irm https://github.com/Sanidhya14321/goharness/releases/latest/download/install.ps1 | iex
# Environment: GOH_VERSION, GOH_BASE_URL, GOH_INSTALL_DIR (default %LOCALAPPDATA%\Programs\goh)
$ErrorActionPreference = 'Stop'
$repo = 'Sanidhya14321/goharness'
$version = if ($env:GOH_VERSION) { $env:GOH_VERSION } else { 'latest' }
if ($env:GOH_BASE_URL) { $base = $env:GOH_BASE_URL }
elseif ($version -eq 'latest') { $base = "https://github.com/$repo/releases/latest/download" }
else { $base = "https://github.com/$repo/releases/download/$version" }
$dest = if ($env:GOH_INSTALL_DIR) { $env:GOH_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'Programs\goh' }

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    'AMD64' { 'amd64' }
    'ARM64' { 'arm64' }
    default { throw "goh install: unsupported architecture $($env:PROCESSOR_ARCHITECTURE)" }
}
$artifact = "goh-windows-$arch.exe"
$tmp = Join-Path ([IO.Path]::GetTempPath()) ([Guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
    Write-Host "Downloading $artifact from $base"
    Invoke-WebRequest -UseBasicParsing "$base/$artifact" -OutFile (Join-Path $tmp $artifact)
    Invoke-WebRequest -UseBasicParsing "$base/SHA256SUMS" -OutFile (Join-Path $tmp 'SHA256SUMS')
    $entries = @(Get-Content (Join-Path $tmp 'SHA256SUMS') | ForEach-Object {
        $parts = $_ -split '\s+', 2
        if ($parts.Count -eq 2 -and ($parts[1] -eq $artifact -or $parts[1] -eq "*$artifact")) { $parts[0] }
    })
    if ($entries.Count -ne 1) { throw "goh install: expected one checksum for $artifact, found $($entries.Count)" }
    $actual = (Get-FileHash -Algorithm SHA256 (Join-Path $tmp $artifact)).Hash.ToLowerInvariant()
    if ($actual -ne $entries[0].ToLowerInvariant()) {
        throw "goh install: checksum mismatch for $artifact (expected $($entries[0]), got $actual); nothing was installed"
    }
    # Check the new binary runs before replacing an existing installation.
    $newVersion = & (Join-Path $tmp $artifact) version 2>&1
    if ($LASTEXITCODE -ne 0) { throw "goh install: downloaded goh does not run: $newVersion; existing installation left unchanged" }
    New-Item -ItemType Directory -Force -Path $dest | Out-Null
    $target = Join-Path $dest 'goh.exe'
    Move-Item -Force (Join-Path $tmp $artifact) $target
    Write-Host "Installed goh $newVersion to $target (sha256 $actual)"
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if (-not (($userPath -split ';') -contains $dest)) {
        Write-Host "Add $dest to your Path, for example:"
        Write-Host "  [Environment]::SetEnvironmentVariable('Path', `"$dest;`" + [Environment]::GetEnvironmentVariable('Path','User'), 'User')"
    }
} finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}

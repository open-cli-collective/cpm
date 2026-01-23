$ErrorActionPreference = 'Stop'

$packageName = 'cpm'
$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"

$url64 = "https://github.com/open-cli-collective/cpm/releases/download/v$($env:chocolateyPackageVersion)/cpm_$($env:chocolateyPackageVersion)_windows_amd64.zip"

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  checksum64     = '$checksum64$'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs

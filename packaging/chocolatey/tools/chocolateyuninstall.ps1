$ErrorActionPreference = 'Stop'

$packageName = 'cpm'
$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"

Remove-Item -Path "$toolsDir\cpm.exe" -Force -ErrorAction SilentlyContinue

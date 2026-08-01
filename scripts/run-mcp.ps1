# Launch stratabench-mcp on Windows (PowerShell).
$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $Root
$Bin = Join-Path $Root "bin\stratabench-mcp.exe"
if (Test-Path $Bin) {
    & $Bin
    exit $LASTEXITCODE
}
go run ./cmd/stratabench-mcp

$ErrorActionPreference = 'Stop'

$date = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$commit = "${env:GITHUB_SHA}".Substring(0, 7)
$base = "qrrun_${env:VERSION}_windows_${env:GOARCH}"

New-Item -ItemType Directory -Path dist -Force | Out-Null

$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
go build -trimpath -ldflags "-s -w -X 'main.version=${env:VERSION}' -X 'main.commit=${commit}' -X 'main.date=${date}'" -o "dist/${base}.exe" ./cmd/qrrun

$productVersion = [regex]::Match($env:VERSION, '^[vV]?(\d+\.\d+\.\d+)').Groups[1].Value
if ([string]::IsNullOrWhiteSpace($productVersion)) {
  throw "Tag '$env:VERSION' must start with SemVer core (e.g. v1.2.3) for MSI versioning"
}

candle -nologo -arch $env:WIX_ARCH -dBinarySource="dist/${base}.exe" -dProductVersion=$productVersion -out "dist/${base}.wixobj" packaging/windows/qrrun.wxs
light -nologo -ext WixUIExtension -out "dist/${base}.msi" "dist/${base}.wixobj"

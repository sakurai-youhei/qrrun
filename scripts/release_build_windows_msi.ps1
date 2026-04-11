$ErrorActionPreference = 'Stop'

$date = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$commit = "${env:GITHUB_SHA}".Substring(0, 7)
$base = "qrrun_${env:VERSION}_windows_${env:GOARCH}"

New-Item -ItemType Directory -Path dist -Force | Out-Null

$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
go build -trimpath -ldflags "-s -w -X 'main.version=${env:VERSION}' -X 'main.commit=${commit}' -X 'main.date=${date}'" -o "dist/${base}.exe" ./cmd/qrrun

$match = [regex]::Match($env:VERSION, '^[vV]?(\d+)\.(\d+)\.(\d+)(?:-[A-Za-z]+\.(\d+))?')
if (-not $match.Success) {
  throw "Tag '$env:VERSION' must start with SemVer core (e.g. v1.2.3) for MSI versioning"
}

$major = [int]$match.Groups[1].Value
$minor = [int]$match.Groups[2].Value
$patch = [int]$match.Groups[3].Value
$build = 0
if ($match.Groups[4].Success) {
  $build = [int]$match.Groups[4].Value
}

$productVersion = "$major.$minor.$patch.$build"
$wxsTemplate = Get-Content packaging/windows/qrrun.wxs -Raw
$wxsRendered = $wxsTemplate.Replace('$(var.ProductVersion)', $productVersion).Replace('$(var.BinarySource)', "dist/${base}.exe")
$wxsPath = "dist/${base}.wxs"
Set-Content -Path $wxsPath -Value $wxsRendered -NoNewline

candle -nologo -arch $env:WIX_ARCH -out "dist/${base}.wixobj" $wxsPath
light -nologo -ext WixUIExtension -out "dist/${base}.msi" "dist/${base}.wixobj"

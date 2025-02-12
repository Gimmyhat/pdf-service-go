param(
    [string]$PprofPort,
    [string]$ServerPort
)

$env:PPROF_PORT = $PprofPort
$env:SERVER_PORT = $ServerPort

$process = Start-Process -NoNewWindow -FilePath '.\pdf-service.exe' -PassThru
$process | Format-List Id, Name, Path 
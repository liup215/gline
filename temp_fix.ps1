$f = 'C:\Users\22569\Workspace\gline\internal\ui\tui_state.go'
$c = [System.IO.File]::ReadAllText($f)
$old = [regex]::Escape('failed"' + "`r`nbreak")
$c = $c -replace $old, ('failed"' + "`r`n`t`t`tbreak")
[System.IO.File]::WriteAllText($f, $c)
Write-Host 'done'
# Update all chat JSON files in ./.util/chats to add message_number fields
$chatDir = ".util/chats"
$chatFiles = Get-ChildItem -Path "$chatDir" -Filter *.json
foreach ($file in $chatFiles) {
    $filePath = $file.FullName
    try {
        $json = Get-Content $filePath -Raw | ConvertFrom-Json
        if ($json.messages) {
            $newMessages = @()
            for ($i = 0; $i -lt $json.messages.Count; $i++) {
                $msg = $json.messages[$i] | ConvertTo-Json | ConvertFrom-Json -AsHashtable
                $msg["message_number"] = $i
                $newMessages += [PSCustomObject]$msg
            }
            $json.messages = $newMessages
            $json | ConvertTo-Json -Depth 10 | Set-Content $filePath
            Write-Host "Updated: $filePath"
        }
    } catch {
        Write-Host "Failed: $filePath"
        Write-Host "Error: $_"
    }
} 
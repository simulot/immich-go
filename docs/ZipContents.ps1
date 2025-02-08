param(
    [Parameter(Mandatory = $true, HelpMessage = "Specify the path to the folder containing ZIP files.")]
    [string]$Path,

    [Parameter(Mandatory = $true, HelpMessage = "Specify the path to the output text file.")]
    [string]$OutputFile
)

# Load the required .NET assembly for ZIP processing
Add-Type -AssemblyName System.IO.Compression.FileSystem

# Validate the provided path
if (-Not (Test-Path -Path $Path)) {
    Write-Host "Error: The specified path does not exist." -ForegroundColor Red
    exit 1
}

# Prepare the output file
"ZIP File Contents" | Out-File -FilePath $OutputFile -Encoding UTF8
Add-Content -Path $OutputFile -Value "`n"

# Process each ZIP file in the specified directory
Get-ChildItem -Path $Path -Filter "*.zip" | ForEach-Object {
    $zipFile = $_.FullName
    Add-Content -Path $OutputFile -Value "Contents of: $zipFile"
    Add-Content -Path $OutputFile -Value "`n"

    # Read and process entries in the ZIP file
    [System.IO.Compression.ZipFile]::OpenRead($zipFile).Entries | ForEach-Object {
        $fileLength = "{0,12}" -f $_.Length 
        $fileDate = $_.LastWriteTime.ToString("yyyy-MM-ddTHH:mm:ssK") # RFC 3339 format
        $filePath = $_.FullName

        # Write formatted line to the output file
        "$fileLength | $fileDate | $filePath" | Add-Content -Path $OutputFile
    }
    Add-Content -Path $OutputFile -Value "`n"
}

Write-Host "The output has been saved to $OutputFile" -ForegroundColor Green

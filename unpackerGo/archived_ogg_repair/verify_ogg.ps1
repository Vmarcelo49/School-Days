# OGG File Verification Script
# This script verifies that extracted OGG files are properly formatted

param(
    [string]$FilePath = "test_output_corrected\SD_BGM\SDBGM01_INT.OGG"
)

function Test-OggHeader {
    param([string]$Path)
    
    if (-not (Test-Path $Path)) {
        Write-Host "File not found: $Path" -ForegroundColor Red
        return $false
    }
    
    $bytes = [System.IO.File]::ReadAllBytes($Path)
    
    # Check for OggS signature
    if ($bytes.Length -lt 4) {
        Write-Host "File too small to be valid OGG" -ForegroundColor Red
        return $false
    }
    
    $signature = [System.Text.Encoding]::ASCII.GetString($bytes[0..3])
    
    if ($signature -eq "OggS") {
        Write-Host "✓ Valid OGG signature found" -ForegroundColor Green
        
        # Check basic OGG page structure
        if ($bytes.Length -ge 27) {
            $version = $bytes[4]
            $headerType = $bytes[5]
            $pageSegments = $bytes[26]
            
            Write-Host "  Version: $version" -ForegroundColor Cyan
            Write-Host "  Header Type: 0x$($headerType.ToString('X2'))" -ForegroundColor Cyan
            Write-Host "  Page Segments: $pageSegments" -ForegroundColor Cyan
            
            # Look for Vorbis signature
            $vorbisFound = $false
            for ($i = 0; $i -le $bytes.Length - 6; $i++) {
                $testStr = [System.Text.Encoding]::ASCII.GetString($bytes[$i..($i+5)])
                if ($testStr -eq "vorbis") {
                    Write-Host "  ✓ Vorbis signature found at offset $i" -ForegroundColor Green
                    $vorbisFound = $true
                    break
                }
            }
            
            if (-not $vorbisFound) {
                Write-Host "  ⚠ No Vorbis signature found" -ForegroundColor Yellow
            }
        }
        
        return $true
    } else {
        Write-Host "✗ Invalid signature: Expected 'OggS', got '$signature'" -ForegroundColor Red
        Write-Host "First 16 bytes: $([System.BitConverter]::ToString($bytes[0..15]))" -ForegroundColor Yellow
        return $false
    }
}

Write-Host "Verifying OGG file: $FilePath" -ForegroundColor White
Write-Host "============================================" -ForegroundColor White

$result = Test-OggHeader -Path $FilePath

if ($result) {
    Write-Host "`n✓ File appears to be a valid OGG file!" -ForegroundColor Green
    $fileInfo = Get-Item $FilePath
    Write-Host "File size: $($fileInfo.Length) bytes" -ForegroundColor Cyan
} else {
    Write-Host "`n✗ File does not appear to be a valid OGG file!" -ForegroundColor Red
}

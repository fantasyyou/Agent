param(
    [ValidateSet("Prepare", "Preflight", "Smoke", "Train", "Evaluate", "All")]
    [string]$Stage = "All",
    [switch]$Resume
)

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$Python = Join-Path $ProjectRoot ".venv\Scripts\python.exe"
if (-not (Test-Path $Python)) {
    throw "没有找到 .venv，请先运行 .\setup_windows.ps1"
}
Set-Location $ProjectRoot

function Invoke-Prepare { & $Python scripts\prepare_data.py --config config.json; if ($LASTEXITCODE -ne 0) { throw "数据准备失败" } }
function Invoke-Preflight { & $Python scripts\preflight.py --config config.json; if ($LASTEXITCODE -ne 0) { throw "环境预检失败" } }
function Invoke-Smoke { & $Python scripts\train_lora.py --config config.json --smoke; if ($LASTEXITCODE -ne 0) { throw "冒烟训练失败" } }
function Invoke-Train {
    $Arguments = @("scripts\train_lora.py", "--config", "config.json")
    if ($Resume) { $Arguments += "--resume" }
    & $Python @Arguments
    if ($LASTEXITCODE -ne 0) { throw "正式训练失败" }
}
function Invoke-Evaluate { & $Python scripts\evaluate.py --config config.json; if ($LASTEXITCODE -ne 0) { throw "评测失败" } }

switch ($Stage) {
    "Prepare" { Invoke-Prepare }
    "Preflight" { Invoke-Preflight }
    "Smoke" { Invoke-Prepare; Invoke-Preflight; Invoke-Smoke }
    "Train" { Invoke-Prepare; Invoke-Preflight; Invoke-Train }
    "Evaluate" { Invoke-Preflight; Invoke-Evaluate }
    "All" { Invoke-Prepare; Invoke-Preflight; Invoke-Smoke; Invoke-Train; Invoke-Evaluate }
}

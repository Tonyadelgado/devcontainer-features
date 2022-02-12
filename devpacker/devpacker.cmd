@echo off
if EXIST "%~dp0\devpacker-windows-%processor_architecture%.exe" (
    "%~dp0\devpacker-windows-%processor_architecture%.exe" %*
) else (
    "%~dp0\dist\devpacker-windows-%processor_architecture%.exe" %*
)

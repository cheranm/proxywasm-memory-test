@echo off

REM Set the path to hey.exe

echo HEY_PATH is set to: %HEY_PATH%

REM Run the command 5 times with a 20-second pause between executions
for /L %%i in (1,1,5) do (
    echo Running iteration %%i
    %HEY_PATH% -z 12s -c 10 -q 120 http://localhost:8080
    timeout /t 20 /nobreak
)

echo Test completed.

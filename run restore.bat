@echo off

go run sfa/archive.go sfa/index.go sfa/main.go sfa/restore.go --pass "pass:test" -v restore archive output

pause

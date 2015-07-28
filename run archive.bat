@echo off

go run sfa/archive.go sfa/index.go sfa/main.go sfa/restore.go --plainindex --pass "pass:test" archive -v . archive

pause

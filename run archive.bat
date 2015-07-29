@echo off

go run sfa/archive.go sfa/index.go sfa/main.go sfa/restore.go --noindexenc --noindexzip --pass "pass:test" -v archive . archive

pause

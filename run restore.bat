@echo off

go run sfa/archive.go sfa/index.go sfa/main.go sfa/restore.go --password "test" --verbose restore archive output

pause

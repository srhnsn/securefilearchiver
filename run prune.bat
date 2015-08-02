@echo off

go run sfa/archive.go sfa/index.go sfa/main.go sfa/restore.go --password "test" --noindexenc --noindexzip --verbose index --prune 1d --gc archive

pause

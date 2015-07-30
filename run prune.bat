@echo off

go run sfa/archive.go sfa/index.go sfa/main.go sfa/restore.go --noindexenc --noindexzip --password "test" -v index --prune 1d --gc archive

pause

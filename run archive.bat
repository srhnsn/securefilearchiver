@echo off

go run sfa/archive.go sfa/index.go sfa/main.go sfa/restore.go --noindexenc --noindexzip --password "test" -v archive --exclude-file test/exclude.txt . archive

pause

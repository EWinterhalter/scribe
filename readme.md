CLI-a tool for backups on Go

CLI is a Go tool using Cobra designed to automate data backup.

Implemented:
- Flag support
- S3 cloud support. Yandex and MinIO cloud were used for testing.
- Configuration via env variables
- Go routines for parallel data processing and minimizing downtime.

Further improvements:
- Add Bubble tea\lip gloss for the outer shell
- Add the possibility of backups on a schedule\trigger
- Package it in docker for more convenience of configuration and automated use
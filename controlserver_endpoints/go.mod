module endpoints

go 1.22.5

replace tpm_sync => ../tpm_sync

require tpm_sync v0.0.0-00010101000000-000000000000

require (
	github.com/beevik/ntp v1.4.3 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
)

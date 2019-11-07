package mode

type Mode int

const (
	MODE_TEST Mode = iota
	MODE_PROD
	MODE_DEV
)

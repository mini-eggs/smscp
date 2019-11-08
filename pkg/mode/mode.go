package mode

type Mode int

const (
	ModeTest Mode = iota
	ModeDev
	ModeProd
)

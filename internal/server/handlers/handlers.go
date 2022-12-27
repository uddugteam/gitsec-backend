package handlers

type Handlers struct {
	dir string
}

func NewHandlers(dir string) *Handlers {
	return &Handlers{dir: dir}
}

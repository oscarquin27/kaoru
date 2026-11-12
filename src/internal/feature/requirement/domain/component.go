package domain

// ComponentRef es un componente del catálogo visto desde el REQ que lo toca.
type ComponentRef struct {
	Id   string
	Code string
	Name string
	Kind string
}

package usecase

import (
	"context"
)

// FindBuild identifica un BUILD por su código. Es la lectura que usan otros features
// para registrar algo sobre un BUILD del catálogo (un RUN, un despliegue, un paquete).
type FindBuild struct {
	d Deps
}

// FindBuildInput es la entrada de FindBuild.
type FindBuildInput struct {
	Code string
}

// BuildRef es lo que identifica a un BUILD: quién es, de qué sistema y qué artefacto.
type BuildRef struct {
	Id        string
	Code      string
	SystemKey string
	Version   string
	Checksum  string
}

// Execute devuelve ErrBuildNotFound si el BUILD no existe.
func (uc *FindBuild) Execute(ctx context.Context, in FindBuildInput) (BuildRef, error) {
	build, err := findBuild(ctx, uc.d.Repository, in.Code)
	if err != nil {
		return BuildRef{}, err
	}
	s := build.Snapshot()
	return BuildRef{Id: s.Id, Code: s.Code.String(), SystemKey: s.System.Key, Version: s.Version, Checksum: s.Checksum}, nil
}

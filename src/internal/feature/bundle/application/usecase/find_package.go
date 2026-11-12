package usecase

import (
	"context"
)

// FindPackage identifica un PKG por su código. Es la lectura que usan otros features
// para enlazar algo a un PKG (un documento, un comentario).
type FindPackage struct {
	d Deps
}

// FindPackageInput es la entrada de FindPackage.
type FindPackageInput struct {
	Code string
}

// PackageRef es lo que identifica a un PKG: quién es, de qué sistema y en qué estado está.
type PackageRef struct {
	Id        string
	Code      string
	Title     string
	SystemKey string
	StateKey  string
}

// Execute devuelve ErrPackageNotFound si el PKG no existe.
func (uc *FindPackage) Execute(ctx context.Context, in FindPackageInput) (PackageRef, error) {
	pkg, err := findPackage(ctx, uc.d.Repository, in.Code)
	if err != nil {
		return PackageRef{}, err
	}
	s := pkg.Snapshot()
	return PackageRef{Id: s.Id, Code: s.Code.String(), Title: s.Title, SystemKey: s.System.Key, StateKey: s.State.Key}, nil
}

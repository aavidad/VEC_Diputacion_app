package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

func nuevaCapacidadReservaCargaPrueba() (ports.TokenReservaCargaDocumental, string, error) {
	token, err := ports.NuevoTokenReservaCargaDocumental()
	if err != nil {
		return ports.TokenReservaCargaDocumental{}, "", err
	}
	huella, err := token.HuellaSHA256()
	return token, huella, err
}

func TestRepositorioCargaPruebaExigeTokenExactoAlAbandonar(t *testing.T) {
	propio, err := ports.NuevoTokenReservaCargaDocumental()
	if err != nil {
		t.Fatal(err)
	}
	huellaPropia, err := propio.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	ajeno, err := ports.NuevoTokenReservaCargaDocumental()
	if err != nil {
		t.Fatal(err)
	}
	repositorio := &repositorioCargaPrueba{huellaTokenSHA256: huellaPropia}
	if err := repositorio.AbandonarReserva(context.Background(), ajeno); !errors.Is(
		err, ports.ErrReservaCargaDocumentalInvalida,
	) {
		t.Fatalf("el doble acepto una capacidad ajena: %v", err)
	}
	if repositorio.abandonada || repositorio.huellaTokenSHA256 != huellaPropia {
		t.Fatal("la capacidad ajena altero el doble de repositorio")
	}
	if err := repositorio.AbandonarReserva(context.Background(), propio); err != nil {
		t.Fatalf("la capacidad original no pudo abandonar: %v", err)
	}
}

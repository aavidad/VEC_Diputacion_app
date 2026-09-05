package bootstrap

import (
	"context"
	"testing"
	postgresidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
)

func TestSeudonimosSesionDesarrolloSeparaPropositosYConservaCuenta(t *testing.T) {
	s := &seudonimizadorSesionDesarrollo{derivador: nuevoDerivadorIdempotenciaPrueba(t, 2, 1)}
	ids := postgresidentidad.IdentificadoresAlta{EspacioIdentidad: espacioIdentidadSesionDesarrollo,
		AsercionID: "mismo", SesionID: "mismo", SujetoID: "mismo", CuentaID: "mismo"}
	a, err := s.SeudonimizarAlta(context.Background(), ids)
	if err != nil {
		t.Fatal(err)
	}
	vistos := map[[32]byte]bool{}
	for _, h := range [][32]byte{a.AsercionIDHMAC, a.SesionIDHMAC, a.SujetoIDHMAC, a.CuentaIDHMAC} {
		if h == [32]byte{} || vistos[h] {
			t.Fatal("huella vacía o propósito repetido")
		}
		vistos[h] = true
	}
	ids.AsercionID, ids.SesionID = "otra-asercion", "otra-sesion"
	b, err := s.SeudonimizarAlta(context.Background(), ids)
	if err != nil || a.CuentaIDHMAC != b.CuentaIDHMAC || a.SujetoIDHMAC != b.SujetoIDHMAC || a.AsercionIDHMAC == b.AsercionIDHMAC || a.SesionIDHMAC == b.SesionIDHMAC {
		t.Fatal("identidad inestable o sesión reutilizada")
	}
	if a.ClaveVersion != 2 || a.ClaveID != "vec.identidad.desarrollo.g2" || a.CuentaOrdinariaIDHMAC != [32]byte{} {
		t.Fatal("coordenadas de desarrollo incorrectas")
	}
}

func TestSeudonimosSesionDesarrolloRechazaOrigenAjenoYDatosIncompletos(t *testing.T) {
	s := &seudonimizadorSesionDesarrollo{derivador: nuevoDerivadorIdempotenciaPrueba(t, 2, 1)}
	base := postgresidentidad.IdentificadoresAlta{EspacioIdentidad: espacioIdentidadSesionDesarrollo, AsercionID: "a", SesionID: "s", SujetoID: "p", CuentaID: "c"}
	for _, alterar := range []func(*postgresidentidad.IdentificadoresAlta){
		func(v *postgresidentidad.IdentificadoresAlta) { v.EspacioIdentidad = "https://otro.invalid" },
		func(v *postgresidentidad.IdentificadoresAlta) { v.CuentaOrdinariaID = "privilegiada" },
		func(v *postgresidentidad.IdentificadoresAlta) { v.AsercionID = "" },
		func(v *postgresidentidad.IdentificadoresAlta) { v.SesionID = "sesion\x00cuenta" },
	} {
		ids := base
		alterar(&ids)
		if _, err := s.SeudonimizarAlta(context.Background(), ids); err == nil {
			t.Fatal("entrada ajena aceptada")
		}
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := s.SeudonimizarAlta(ctx, base); err == nil {
		t.Fatal("cancelación ignorada")
	}
}

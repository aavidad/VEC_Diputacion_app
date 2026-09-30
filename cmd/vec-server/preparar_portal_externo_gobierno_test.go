package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/bootstrap"
)

func TestPrepararPortalExternoDistinguePropuestaDePublicacion(t *testing.T) {
	var salida bytes.Buffer
	var recibidas bootstrap.OpcionesPreparacionPortalExterno
	preparar := func(_ context.Context, _ config.Config, o bootstrap.OpcionesPreparacionPortalExterno) (bootstrap.ResumenPreparacionPortalExterno, error) {
		recibidas = o
		return bootstrap.ResumenPreparacionPortalExterno{PendientePublicacion: o.HuellaAprobacionSHA256 == "", HuellaAprobacionSHA256: strings.Repeat("a", 64), PreimagenSHA256: strings.Repeat("b", 64)}, nil
	}
	args := []string{"--material-externo", "/material-externo", "--consumidores", "usuarios_preferencias"}
	if err := ejecutarPreparacionPortalExterno(t.Context(), args, &salida, config.Config{}, preparar); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(salida.String(), "preparacion_portal_externo=pendiente_publicacion") || strings.Contains(salida.String(), "=correcta") {
		t.Fatal("la propuesta se presenta como publicada")
	}
	salida.Reset()
	args = append(args, "--aprobacion-sha256", strings.Repeat("a", 64), "--preimagen-sha256", strings.Repeat("b", 64), "--rotar-raiz-preimagen-sha256", strings.Repeat("c", 64))
	if err := ejecutarPreparacionPortalExterno(t.Context(), args, &salida, config.Config{}, preparar); err != nil {
		t.Fatal(err)
	}
	if recibidas.HuellaAprobacionSHA256 != strings.Repeat("a", 64) || recibidas.PreimagenSHA256 != strings.Repeat("b", 64) || recibidas.RotarRaizPreimagenSHA256 != strings.Repeat("c", 64) || !strings.Contains(salida.String(), "preparacion_portal_externo=correcta") {
		t.Fatal("la aprobación explícita no llegó a la preparación")
	}
}

func TestPrepararPortalExternoRechazaAprobacionSinPreimagen(t *testing.T) {
	for _, bandera := range []string{"--aprobacion-sha256", "--preimagen-sha256"} {
		llamada := false
		f := func(context.Context, config.Config, bootstrap.OpcionesPreparacionPortalExterno) (bootstrap.ResumenPreparacionPortalExterno, error) {
			llamada = true
			return bootstrap.ResumenPreparacionPortalExterno{}, nil
		}
		err := ejecutarPreparacionPortalExterno(t.Context(), []string{"--material-externo", "/externo", "--consumidores", "usuarios_preferencias", bandera, strings.Repeat("a", 64)}, &bytes.Buffer{}, config.Config{}, f)
		if !errors.Is(err, errArgumentosPreparacion) || llamada {
			t.Fatal("se aceptó media aprobación")
		}
	}
}

package ports

import (
	"context"
	"testing"
	"time"
)

func TestEmisorBolsaReutilizaElCanonDelVerificadorExistente(t *testing.T) {
	ctx := context.Background()
	base := instanteBolsaPrueba()
	ahora := base.Add(3 * time.Minute)
	emisor, err := NuevoEmisorEvidenciaIntegracionBolsa(autoridadRespuestaBolsaPrueba, claveRespuestaBolsaV1Prueba, selladorRespuestaBolsaPrueba())
	if err != nil {
		t.Fatal(err)
	}
	verificador := verificadorRespuestaBolsaPrueba(claveRespuestaBolsaV1Prueba)
	solicitud := solicitudDisponibilidadPrueba(t, base)
	disponibilidad, err := emisor.FirmarDisponibilidad(ctx, solicitud, resultadoDisponibilidadPrueba(t, base), ahora)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := verificador.VerificarDisponibilidad(ctx, solicitud, disponibilidad, ahora); err != nil {
		t.Fatal(err)
	}
	comandoOrden := comandoOrdenPrueba(t, base)
	orden, err := emisor.FirmarOrden(ctx, comandoOrden, reciboOrdenPrueba(t, base), ahora)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := verificador.VerificarReciboOrden(ctx, comandoOrden, orden, ahora); err != nil {
		t.Fatal(err)
	}
	comando := comandoLlamamientoPrueba(t, base, selladorRespuestaBolsaPrueba())
	llamamiento, err := emisor.FirmarLlamamiento(ctx, comando, reciboLlamamientoPrueba(t, comando, base), ahora)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := verificador.VerificarReciboLlamamiento(ctx, comando, llamamiento, ahora); err != nil {
		t.Fatal(err)
	}
	llamamiento.ReciboRef = "recibo:alterado"
	if _, _, err := verificador.VerificarReciboLlamamiento(ctx, comando, llamamiento, ahora); err == nil {
		t.Fatal("se aceptó un recibo alterado después de firmar")
	}
}

func TestEmisorBolsaNoDevuelveSelloProvisionalAlFallar(t *testing.T) {
	base := instanteBolsaPrueba()
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	sellador := selladorRespuestaBolsaPrueba()
	sellador.cancelar = cancelar
	emisor, err := NuevoEmisorEvidenciaIntegracionBolsa(autoridadRespuestaBolsaPrueba, claveRespuestaBolsaV1Prueba, sellador)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := emisor.FirmarDisponibilidad(ctx, solicitudDisponibilidadPrueba(t, base), resultadoDisponibilidadPrueba(t, base), base.Add(3*time.Minute))
	if err == nil || resultado != (ResultadoDisponibilidadBolsa{}) {
		t.Fatal("cancelación devolvió una respuesta utilizable")
	}
	if _, err := NuevoEmisorEvidenciaIntegracionBolsa("", claveRespuestaBolsaV1Prueba, sellador); err == nil {
		t.Fatal("autoridad vacía aceptada")
	}
}

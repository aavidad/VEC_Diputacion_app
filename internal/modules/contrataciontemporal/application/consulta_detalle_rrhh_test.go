package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestConsultaDetalleRRHHDevuelveProyeccionValidadaYClonada(t *testing.T) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	servicio, err := NuevoServicioConsultaDetalleRRHH(
		entorno.autoridad, entorno.autorizador, entorno.sesion, entorno.reloj,
	)
	if err != nil {
		t.Fatal(err)
	}
	obtenido, err := servicio.Consultar(context.Background(), entorno.detalle)
	if err != nil {
		t.Fatalf("consultar: %v", err)
	}
	if entorno.sesion.llamadasDetalle != 1 ||
		obtenido.Resumen.Validar() != nil ||
		obtenido.Lectura.ExpedienteRef() != entorno.detalle.ExpedienteRef() {
		t.Fatalf("detalle inesperado: %#v", obtenido)
	}
	obtenido.Hitos[0].AccionClave = "accion.alterada"
	if entorno.sesion.detalle.Hitos[0].AccionClave == "accion.alterada" {
		t.Fatal("el detalle comparte la colección del adaptador")
	}
}

func TestConsultaDetalleRRHHNoSerializaCamposPersonalesNiTextoLibre(t *testing.T) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	serializado, err := json.Marshal(entorno.sesion.detalle)
	if err != nil {
		t.Fatal(err)
	}
	contenido := string(serializado)
	for _, prohibido := range []string{
		"actor_ref", "observaciones", "contacto",
		"actor:rrhh:001", "contacto:rrhh:001", "Necesidad temporal.",
	} {
		if strings.Contains(contenido, prohibido) {
			t.Fatalf("el detalle filtra %q: %s", prohibido, contenido)
		}
	}
}

func TestConsultaDetalleRRHHNoEsOraculo(t *testing.T) {
	t.Parallel()
	for _, origen := range []string{"autorizador", "sesion"} {
		origen := origen
		t.Run(origen, func(t *testing.T) {
			t.Parallel()
			entorno := nuevoEntornoConsultaRRHH(t)
			if origen == "autorizador" {
				entorno.autorizador.errDetalle =
					ports.ErrConsultaRRHHNoObservable
			} else {
				entorno.sesion.errDetalle =
					ports.ErrConsultaRRHHNoObservable
			}
			servicio, err := NuevoServicioConsultaDetalleRRHH(
				entorno.autoridad, entorno.autorizador,
				entorno.sesion, entorno.reloj,
			)
			if err != nil {
				t.Fatal(err)
			}
			_, err = servicio.Consultar(context.Background(), entorno.detalle)
			if !errors.Is(err, ErrConsultaRRHHNoObservable) {
				t.Fatalf("resultado observable: %v", err)
			}
		})
	}
}

func TestConsultaDetalleRRHHRechazaResultadoDeOtroExpediente(t *testing.T) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	entorno.sesion.detalle.Resumen.ExpedienteRef = "expediente:rrhh:ajeno"
	servicio, err := NuevoServicioConsultaDetalleRRHH(
		entorno.autoridad, entorno.autorizador, entorno.sesion, entorno.reloj,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = servicio.Consultar(
		context.Background(), entorno.detalle,
	); !errors.Is(err, ErrResultadoConsultaRRHHNoConfiable) {
		t.Fatalf("resultado ajeno aceptado: %v", err)
	}
}

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
		entorno.autoridad, entorno.emisor, entorno.sesion, entorno.reloj,
	)
	if err != nil {
		t.Fatal(err)
	}
	obtenido, err := servicio.Consultar(context.Background(), entorno.detalle)
	if err != nil {
		t.Fatalf("consultar: %v", err)
	}
	if entorno.emision.motivos.llamadasDetalle != 1 ||
		entorno.emision.motivos.llamadasCuadro != 0 ||
		entorno.emision.correlaciones.llamadas != 1 ||
		entorno.emision.reloj.llamadas != 2 ||
		entorno.emision.detalle.llamadas != 1 ||
		entorno.emision.cuadro.llamadas != 0 ||
		entorno.sesion.llamadasDetalle != 1 ||
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
	for _, origen := range []string{"ausente", "ajeno"} {
		origen := origen
		t.Run(origen, func(t *testing.T) {
			t.Parallel()
			entorno := nuevoEntornoConsultaRRHH(t)
			entorno.sesion.errDetalle = ports.ErrConsultaRRHHNoObservable
			servicio, err := NuevoServicioConsultaDetalleRRHH(
				entorno.autoridad, entorno.emisor,
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
		entorno.autoridad, entorno.emisor, entorno.sesion, entorno.reloj,
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

func TestConsultaResumenSeguimientoNoExportaDetalleYConservaConsultaNominal(t *testing.T) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	servicio, err := NuevoServicioConsultaDetalleRRHH(entorno.autoridad, entorno.emisor, entorno.sesion, entorno.reloj)
	if err != nil {
		t.Fatal(err)
	}
	resumen, err := servicio.ConsultarResumenSeguimiento(context.Background(), entorno.detalle,
		entorno.sesion.detalle.Resumen.OrganizacionRef, entorno.sesion.detalle.Resumen.UnidadRef)
	if err != nil {
		t.Fatal(err)
	}
	if resumen.ExpedienteRef != entorno.detalle.ExpedienteRef() ||
		resumen.VersionExpediente != entorno.sesion.detalle.Resumen.Version ||
		entorno.sesion.llamadasDetalle != 1 || entorno.emision.detalle.llamadas != 1 {
		t.Fatal("la proyeccion no procede de la consulta nominal validada")
	}
	b, err := json.Marshal(resumen)
	if err != nil {
		t.Fatal(err)
	}
	var campos map[string]any
	if err := json.Unmarshal(b, &campos); err != nil {
		t.Fatal(err)
	}
	if len(campos) != 2 || campos["expediente_ref"] == nil || campos["version_expediente"] == nil {
		t.Fatalf("la proyeccion exporta otros campos: %s", b)
	}
}

func TestConsultaResumenSeguimientoDeniegaCruces(t *testing.T) {
	t.Parallel()
	for _, caso := range []struct {
		nombre string
		org    string
		unidad string
	}{
		{nombre: "organizacion", org: "organizacion:ajena"},
		{nombre: "unidad", unidad: "unidad:ajena"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			entorno := nuevoEntornoConsultaRRHH(t)
			servicio, err := NuevoServicioConsultaDetalleRRHH(entorno.autoridad, entorno.emisor, entorno.sesion, entorno.reloj)
			if err != nil {
				t.Fatal(err)
			}
			org, unidad := entorno.sesion.detalle.Resumen.OrganizacionRef, entorno.sesion.detalle.Resumen.UnidadRef
			if caso.org != "" {
				org = caso.org
			}
			if caso.unidad != "" {
				unidad = caso.unidad
			}
			resumen, err := servicio.ConsultarResumenSeguimiento(context.Background(), entorno.detalle, org, unidad)
			if !errors.Is(err, ErrConsultaRRHHNoObservable) || resumen != (ResumenConsultaSeguimientoRRHH{}) {
				t.Fatalf("cruce publicado: %+v, %v", resumen, err)
			}
		})
	}
}

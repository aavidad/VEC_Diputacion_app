package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func solicitudReanudacionSeleccionPGPrueba(t *testing.T) ports.SolicitudReservaEjecucionSeleccionLlamamiento {
	t.Helper()
	base := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	emisor, err := ports.NuevoEmisorContextoPeticionIntegracionBolsa(
		"autoridad:contratacion-temporal", clavePeticionSeleccionO6Prueba, selladorContextoSeleccionO6Prueba{})
	if err != nil {
		t.Fatal(err)
	}
	necesidad := referenciaSeleccionO6Prueba("necesidad:sintetica", 'a')
	bolsa := referenciaSeleccionO6Prueba("bolsa:sintetica", 'b')
	politica := referenciaSeleccionO6Prueba("politica:sintetica", 'c')
	crear := func(operacion string, recurso ports.ReferenciaVersionadaIntegracionBolsa) ports.ContextoPeticionIntegracionBolsa {
		c, err := emisor.Emitir(context.Background(), ports.DatosContextoPeticionIntegracionBolsa{
			OperacionRef: operacion, OrganizacionRef: "org:sintetica", ExpedienteRef: "exp:sintetico",
			VersionExpediente: 6, CorrelacionRef: "correlacion:sintetica", ContratoVersion: 1,
			AutoridadSolicitante: "autoridad:contratacion-temporal",
			Autorizacion:         referenciaSeleccionO6Prueba("intencion:sintetica", 'd'),
			Accion:               referenciaSeleccionO6Prueba("accion:"+operacion, 'e'), Recurso: recurso,
			Finalidad:    referenciaSeleccionO6Prueba("finalidad:sintetica", 'f'),
			SolicitadaEn: base, ValidaHasta: base.Add(time.Minute),
		}, base)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	consulta, err := ports.NuevaConsultaTerminalAutorizada(claveEjecucionSeleccionO6Prueba, crear("consulta", necesidad), base)
	if err != nil {
		t.Fatal(err)
	}
	s, err := ports.NuevaSolicitudReservaEjecucionSeleccionLlamamiento(consulta,
		ports.ComandoPrepararOrdenBolsa{Contexto: crear("orden", bolsa), Necesidad: necesidad,
			Bolsa: bolsa, Politica: politica, MaximoPosiciones: 3}, 2, base)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestReanudacionSeleccionPGEstadoExactoYFalloSinAutoridad(t *testing.T) {
	s := solicitudReanudacionSeleccionPGPrueba(t)
	b, err := codificarSolicitudSeleccionO6(s)
	if err != nil {
		t.Fatal(err)
	}
	f := filaEjecucionSeleccionO6{Situacion: "propietaria", SolicitudJSON: string(b),
		ReservaRef: prefijoReservaSeleccionO6 + strings.Repeat("c", 64), Efecto: "preparar_orden"}
	r, err := estadoReanudacionSeleccionDesdeFila(f, s)
	if err != nil || r.Solicitud != s || r.ReservaRef != f.ReservaRef ||
		r.EfectoPosible != ports.EfectoPrepararOrdenSeleccionLlamamiento ||
		r.Situacion != ports.EjecucionSeleccionLlamamientoPropietaria {
		t.Fatal("estado reanudado divergente")
	}
	for _, cambiar := range []func(*filaEjecucionSeleccionO6){
		func(x *filaEjecucionSeleccionO6) { x.Situacion = "ocupada" },
		func(x *filaEjecucionSeleccionO6) { x.ReservaRef = "" },
		func(x *filaEjecucionSeleccionO6) { x.Efecto = "" },
		func(x *filaEjecucionSeleccionO6) { x.Efecto = "solicitar_llamamiento" },
		func(x *filaEjecucionSeleccionO6) { x.ReciboJSON = "{}" },
		func(x *filaEjecucionSeleccionO6) { x.ArtefactoJSON = "{}" },
		func(x *filaEjecucionSeleccionO6) { x.SolicitudJSON = "{}" },
	} {
		otra := f
		cambiar(&otra)
		if r, err := estadoReanudacionSeleccionDesdeFila(otra, s); err == nil || r != (ports.EstadoEjecucionSeleccionLlamamiento{}) {
			t.Fatal("aceptó respuesta divergente")
		}
	}
	pool := &iniciadorEjecucionSeleccionO6Prueba{}
	a := &EjecucionesSeleccionLlamamientoPostgreSQL{pool: pool}
	if r, err := a.ReanudarPreparacionOrden(context.Background(), s, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}); err == nil || r != (ports.EstadoEjecucionSeleccionLlamamiento{}) || pool.inicios != 0 {
		t.Fatal("material vacío alcanzó transacción")
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := a.ReanudarPreparacionOrden(ctx, s, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}); !errors.Is(err, context.Canceled) || pool.inicios != 0 {
		t.Fatal("cancelación alcanzó transacción")
	}
}

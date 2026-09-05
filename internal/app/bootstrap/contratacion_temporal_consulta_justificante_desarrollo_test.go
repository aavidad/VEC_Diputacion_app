package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func escenarioConsultaJustificantePrueba(t *testing.T) (context.Context, *proveedorConsultaJustificanteRespuestaDesarrollo, *autorizacionComunicacionDesarrolloPrueba, ports.SolicitudResolverLlamamiento) {
	t.Helper()
	ctx, base, a, respuesta := escenarioRespuestaRecibidaPrueba(t)
	s := ports.SolicitudResolverLlamamiento{ClaveIdempotencia: "22222222-2222-4222-8222-222222222222",
		OrganizacionRef: respuesta.OrganizacionRef, ExpedienteRef: respuesta.ExpedienteRef,
		LlamamientoRef: respuesta.LlamamientoRef, ComunicacionRef: respuesta.ComunicacionRef,
		VersionEsperada: 2, Respuesta: respuesta.Respuesta, PruebaRespuestaRef: "justificante:sintetico"}
	capacidad, _ := base.soporte.capacidadValida(ctx)
	capacidad.ruta = httpinterno.RutaResolucionComunicacionLlamamiento
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	ctx = context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{}, preparacionLlamamientoDesarrollo{expediente: expedientePuenteBolsaPrueba(t)})
	ctx = context.WithValue(ctx, claveConsultaJustificanteRespuestaDesarrollo{}, s)
	return ctx, &proveedorConsultaJustificanteRespuestaDesarrollo{soporte: base.soporte, autorizador: a, reloj: base.reloj}, a, s
}

func TestConsultaJustificanteDesarrolloExigeAccionPropiaYMaterialLigado(t *testing.T) {
	ctx, _, _, s := escenarioConsultaJustificantePrueba(t)
	r, err := postgresct.RecursoConsultaJustificanteRespuestaRecibida(s)
	if err != nil {
		t.Fatal(err)
	}
	d := dominiovec.DatosSolicitudAutorizacionLigadaV3{Finalidad: "gestionar_contratacion_temporal",
		ReferenciaMotivo: motivoConsultaJustificanteRespuestaDesarrollo(), Accion: postgresct.AccionConsultaJustificanteRespuestaRecibida, Recurso: r}
	if !solicitudAutorizacionLlamamientoDesarrolloValida(ctx, httpinterno.RutaResolucionComunicacionLlamamiento, d) {
		t.Fatal("consulta ligada rechazada")
	}
	for _, cambiar := range []func(*dominiovec.DatosSolicitudAutorizacionLigadaV3){
		func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Accion = postgresct.AccionRegistroRespuestaRecibida
		},
		func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.ReferenciaMotivo = motivoLlamamientoDesarrollo(true)
		},
		func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) { d.Recurso.Referencia += "-otro" },
		func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Atributos = map[string]string{"material_sha256": strings.Repeat("b", 64)}
		},
	} {
		otro := d
		cambiar(&otro)
		if solicitudAutorizacionLlamamientoDesarrolloValida(ctx, httpinterno.RutaResolucionComunicacionLlamamiento, otro) {
			t.Fatal("permiso de otro material o efecto admitido")
		}
	}
	if solicitudAutorizacionLlamamientoDesarrolloValida(ctx, httpinterno.RutaRegistroRespuestaRecibida, d) {
		t.Fatal("consulta concedida por la ruta de declaración")
	}
}

func TestConsultaJustificanteDesarrolloReacreditaIdentidadYPeticion(t *testing.T) {
	ctx, p, a, s := escenarioConsultaJustificantePrueba(t)
	for intento := 1; intento <= 2; intento++ {
		if _, err := p.AutorizarConsultaJustificanteRespuestaRecibida(ctx, s); !errors.Is(err, ports.ErrOperacionRespuestaRecibidaDenegada) ||
			a.llamadas != intento || a.accion != postgresct.AccionConsultaJustificanteRespuestaRecibida {
			t.Fatal("lectura omitió autorización fresca o aceptó material vacío")
		}
	}
	if _, err := p.AutorizarConsultaJustificanteRespuestaRecibida(context.Background(), s); !errors.Is(err, ports.ErrOperacionRespuestaRecibidaDenegada) || a.llamadas != 2 {
		t.Fatal("referencias sustituyeron identidad")
	}
	s.PruebaRespuestaRef += "-otro"
	if _, err := p.AutorizarConsultaJustificanteRespuestaRecibida(ctx, s); !errors.Is(err, ports.ErrOperacionRespuestaRecibidaDenegada) || a.llamadas != 2 {
		t.Fatal("justificante ajeno alcanzó autorización")
	}
}

type lectorExpedienteConsultaJustificantePrueba struct {
	expediente ports.ExpedienteParaSeleccion
	llamadas   int
}

func (l *lectorExpedienteConsultaJustificantePrueba) LeerExpedienteParaSeleccion(context.Context, string, string, uint64) (ports.ExpedienteParaSeleccion, error) {
	l.llamadas++
	return l.expediente, nil
}

type lectorJustificanteDenegadoPrueba struct{ llamadas int }

func (l *lectorJustificanteDenegadoPrueba) ConsultarJustificanteRespuestaRecibida(context.Context, ports.SolicitudResolverLlamamiento) (ports.JustificanteRespuestaRecibida, error) {
	l.llamadas++
	return ports.JustificanteRespuestaRecibida{}, ports.ErrOperacionRespuestaRecibidaDenegada
}

func TestConsultaJustificanteDesarrolloResolverNoOcultaDenegacion(t *testing.T) {
	ctx, p, _, s := escenarioConsultaJustificantePrueba(t)
	le := &lectorExpedienteConsultaJustificantePrueba{expediente: expedientePuenteBolsaPrueba(t)}
	lj := &lectorJustificanteDenegadoPrueba{}
	e := &ejecutorComunicacionLlamamientoDesarrollo{soporte: p.soporte, lector: le, lectorJustificante: lj}
	if _, err := e.Resolver(context.Background(), s); !errors.Is(err, application.ErrComunicacionLlamamientoDenegada) || le.llamadas != 0 || lj.llamadas != 0 {
		t.Fatal("resolución leyó sin identidad")
	}
	r, err := e.Resolver(ctx, s)
	if !errors.Is(err, application.ErrComunicacionLlamamientoDenegada) || r != (ports.ResultadoResolucionLlamamiento{}) ||
		le.llamadas != 1 || lj.llamadas != 1 {
		t.Fatal("denegación de justificante convertida en validación pendiente o recibo")
	}
}

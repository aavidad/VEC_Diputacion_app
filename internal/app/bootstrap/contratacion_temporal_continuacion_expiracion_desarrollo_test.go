package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Tras una expiración no hay justificante: consultarlo sería un error.
type lectorJustificanteProhibidoPrueba struct{ t *testing.T }

func (l lectorJustificanteProhibidoPrueba) ConsultarJustificanteRespuestaRecibida(context.Context, ports.SolicitudResolverLlamamiento) (ports.JustificanteRespuestaRecibida, error) {
	l.t.Fatal("la continuación tras expiración consultó un justificante")
	return ports.JustificanteRespuestaRecibida{}, nil
}

func escenarioContinuacionExpiracionPrueba(t *testing.T) escenarioContinuacionCoordinador {
	t.Helper()
	_, _, l := escenarioRevisionManualPrueba(t)
	p, ctx, _, seleccion, _, repo, _ := escenarioAceptacionPuentePrueba(t)
	if seleccion.OperacionRef != l.justificante.Seleccion.OperacionRef {
		t.Fatal("fixtures de apertura distintos")
	}
	s := l.solicitud
	s.Respuesta, s.PruebaRespuestaRef = ports.RespuestaLlamamientoExpirada, ""
	politica := ports.ReferenciaGobernadaComunicacionLlamamiento{Referencia: reglas.CatalogoBolsa + ":3:b08.sin_respuesta_baja",
		Version: 3, HuellaSHA256: strings.Repeat("ab", 32)}
	s.CriterioValidacionRef = politica.Referencia
	local := l.local
	local.Solicitud, local.Politica, local.EstadoPlazo = s, politica, ports.PlazoLlamamientoExpirado
	local.RespuestaHasta = local.ResueltaEn.Add(-time.Hour)
	local.IntencionSiguiente = ports.IntencionOutboxSiguienteCandidato{
		Solicitud: s, ResolucionRef: local.ResolucionRef, LlamamientoRef: s.LlamamientoRef,
		ClaveIdempotencia: s.ClaveIdempotencia, VersionEsperada: 2, VersionResultante: 3,
		IntencionRef: "intencion:9a1c7a52-3f7e-4b8e-9d55-2f0c5e1b7a11", ComandoOpacoRef: "comando:expiracion-prueba",
		Estado: ports.OutboxSiguienteCandidatoPendiente, ActualizadaEn: local.ResueltaEn}
	if local.ValidarPara(s) != nil {
		t.Fatal("resolución de expiración inválida")
	}
	sc := ports.SolicitudContinuarLlamamiento{ClaveIdempotencia: "34444444-3333-4333-8333-333333333333",
		OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
		ResolucionRef: local.ResolucionRef, IntencionRef: local.IntencionSiguiente.IntencionRef}
	sel := l.justificante.Seleccion
	a := ports.AntecedenteContinuacionLlamamiento{Resolucion: local, ComandoSiguienteRef: local.IntencionSiguiente.ComandoOpacoRef,
		ComandoSiguiente: ports.ComandoSiguienteLlamamiento{Esquema: "vec.contratacion-temporal.siguiente-candidato.intencion.v1",
			ComandoRef: local.IntencionSiguiente.ComandoOpacoRef, IntencionRef: sc.IntencionRef,
			OrganizacionRef: sc.OrganizacionRef, ExpedienteRef: sc.ExpedienteRef, LlamamientoRef: s.LlamamientoRef,
			SeleccionClave: "11111111-1111-4111-8111-111111111111"}, Seleccion: &sel}
	if err := a.ValidarPara(sc); err != nil {
		t.Fatal("antecedente de expiración inválido", err)
	}
	capacidad, _ := p.alta.soporte.capacidadValida(ctx)
	capacidad.ruta = httpinterno.RutaContinuacionLlamamiento
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	autoridad := &autorizacionContinuacionPrueba{t: t, puente: p}
	p.autorizadorSiguiente, p.autorizadorRenuncia = autoridad, autoridad
	registro := &registroContinuacionCoordinadorPrueba{t: t, antecedente: a, bolsa: repo,
		proveedor: &proveedorContinuacionLlamamientoDesarrollo{soporte: p.alta.soporte, autorizador: autoridad, reloj: p.reloj}}
	lector := &lectorExpedienteConsultaJustificantePrueba{expediente: expedientePuenteBolsaPrueba(t)}
	e := &ejecutorComunicacionLlamamientoDesarrollo{soporte: p.alta.soporte, lector: lector,
		lectorJustificante: lectorJustificanteProhibidoPrueba{t}, continuaciones: registro, continuador: p}
	return escenarioContinuacionCoordinador{ctx: ctx, ejecutor: e, solicitud: sc, registro: registro, lector: lector, autoridad: autoridad}
}

// La expiración confirmada cierra en Bolsa el llamamiento «sin respuesta» con
// el permiso de no aceptación y abre el siguiente, como tras una renuncia.
func TestContinuacionLlamamientoDesarrolloTrasExpiracionCierraYAbreSiguiente(t *testing.T) {
	f := escenarioContinuacionExpiracionPrueba(t)
	r, err := f.ejecutor.Continuar(f.ctx, f.solicitud)
	if err != nil || r.ValidarPara(f.solicitud) != nil || r.Estado != "confirmado" {
		t.Fatal("continuación tras expiración", err)
	}
	if !reflect.DeepEqual(f.autoridad.etapas, []string{"ct:consulta", "bolsa:sin_respuesta", "bolsa", "ct:confirmacion"}) {
		t.Fatal("permisos omitidos o desordenados", f.autoridad.etapas)
	}
	terminal, existe := f.registro.bolsa.filas[r.ReciboBolsa.TerminalOperacionRef]
	if !existe || terminal.Registro.Tipo != puertosbolsa.TipoExpiracionRRHHDesarrollo ||
		terminal.Registro.EstadoLlamamiento != dominiobolsa.EstadoLlamamientoExpirado ||
		terminal.Registro.Resolucion.PoliticaRef != f.registro.antecedente.Resolucion.Politica.Referencia ||
		len(f.registro.bolsa.filas) != 4 || r.LlamamientoAnteriorRef != f.registro.antecedente.Resolucion.Solicitud.LlamamientoRef {
		t.Fatal("llamamiento no cerrado sin respuesta o siguiente no abierto")
	}
	// Reintento: no vuelve a cerrar ni pide su permiso; recupera el siguiente.
	f.registro.bolsa.reloj.instante = f.registro.bolsa.reloj.instante.Add(time.Minute)
	replay, err := f.ejecutor.Continuar(f.ctx, f.solicitud)
	if err != nil || replay.Estado != "replay_confirmado" || replay.ReciboRef != r.ReciboRef || replay.ReciboBolsa != r.ReciboBolsa ||
		!reflect.DeepEqual(f.autoridad.etapas[4:], []string{"ct:consulta", "bolsa", "ct:confirmacion"}) || len(f.registro.bolsa.filas) != 4 {
		t.Fatal("replay tras expiración", err, f.autoridad.etapas)
	}
}

func TestContinuacionLlamamientoDesarrolloTrasExpiracionSoloRecuperacionNoCierra(t *testing.T) {
	f := escenarioContinuacionExpiracionPrueba(t)
	f.lector.expediente.VersionActual = 7
	r, err := f.ejecutor.Continuar(f.ctx, f.solicitud)
	if !errors.Is(err, ports.ErrOperacionContinuacionNoDisponible) || r != (ports.ResultadoContinuacionLlamamiento{}) ||
		len(f.registro.bolsa.filas) != 2 {
		t.Fatal("cerró o abrió en Bolsa después de la propuesta", err)
	}
}

// El permiso de no aceptación en la ruta de continuación solo vale para el
// terminal exacto de una expiración; nunca tras una renuncia.
func TestContinuacionLlamamientoDesarrolloPermisoSinRespuestaLigado(t *testing.T) {
	for _, expiracion := range []bool{true, false} {
		var f escenarioContinuacionCoordinador
		if expiracion {
			f = escenarioContinuacionExpiracionPrueba(t)
		} else {
			f = escenarioContinuacionCoordinadorPrueba(t)
		}
		if _, err := f.ejecutor.Continuar(f.ctx, f.solicitud); err != nil {
			t.Fatal(err)
		}
		i := 0
		for i < len(f.autoridad.etapas) && f.autoridad.etapas[i] != "bolsa" {
			i++
		}
		ctx := f.autoridad.contextos[i]
		l, _ := ctx.Value(claveContinuacionLlamamientoDesarrollo{}).(continuacionLigadaDesarrollo)
		recurso := dominiovec.RecursoAutorizable{Referencia: terminalContinuacionDesarrollo(l), ModuloID: "bolsa", Tipo: "integracion_llamamientos_bolsa",
			Ambitos:   map[string]string{"categoria_ref": "categoria:desarrollo:c2", "unidad_ref": unidadCoberturaContratacionTemporalDesarrollo},
			Atributos: map[string]string{"necesidad_ref": l.seleccion.Necesidad.Referencia, "contenido_sha256": strings.Repeat("c", 64)}}
		d := dominiovec.DatosSolicitudAutorizacionLigadaV3{Accion: puertosbolsa.AccionRenunciarLlamamientoRRHHDesarrollo,
			Finalidad: "gestionar_contratacion_temporal", ReferenciaMotivo: motivoRenunciaBolsaDesarrollo(), Recurso: recurso}
		if solicitudAutorizacionLlamamientoDesarrolloValida(ctx, httpinterno.RutaContinuacionLlamamiento, d) != expiracion {
			t.Fatal("permiso de no aceptación mal ligado", expiracion)
		}
		if expiracion {
			d.Recurso.Referencia = operacionSiguienteDesarrollo(l.solicitud)
			if solicitudAutorizacionLlamamientoDesarrolloValida(ctx, httpinterno.RutaContinuacionLlamamiento, d) {
				t.Fatal("permiso de no aceptación sobre otra operación")
			}
		}
	}
}

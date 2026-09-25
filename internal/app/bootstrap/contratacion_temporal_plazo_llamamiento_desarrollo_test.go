package bootstrap

import (
	"errors"
	"strings"
	"testing"
	"time"

	calendariosdomain "vec-diputacion-granada/internal/modules/calendarios/domain"
	calendariosports "vec-diputacion-granada/internal/modules/calendarios/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

func reglasPlazoEjemploPrueba(t *testing.T, consulta calendariosports.ConsultaCalendarios) reglasPlazoLlamamientoDesarrollo {
	t.Helper()
	compuestas, err := nuevasReglasEjemploDesarrollo(configuracionDesarrolloReglasEjemplo(rutaReglasBolsaEjemploPrueba, ""),
		consulta, relojPresentacionReglasEjemplo)
	if err != nil {
		t.Fatal(err)
	}
	return reglasPlazoLlamamientoDesarrollo{resolutor: compuestas.bolsa}
}

func calendarioUnDiaHabilPrueba(t *testing.T) *consultaCalendariosReglasPrueba {
	t.Helper()
	dia, err := calendariosdomain.ParsearFechaCivil("2026-09-29")
	if err != nil {
		t.Fatal(err)
	}
	fin, err := dia.FinEnMadrid()
	if err != nil {
		t.Fatal(err)
	}
	return &consultaCalendariosReglasPrueba{resultado: calendariosports.ResultadoCalculoPlazo{
		ResultadoPlazo: calendariosdomain.ResultadoPlazo{Vencimiento: dia, VenceAntesDe: fin},
	}}
}

// El vencimiento sale del catálogo (b05: un día hábil desde el contacto
// efectivo) y del calendario oficial; las reglas de resolución, de b07-b09.
func TestReglasPlazoLlamamientoDesdeElCatalogoDeEjemplo(t *testing.T) {
	calendario := calendarioUnDiaHabilPrueba(t)
	r := reglasPlazoEjemploPrueba(t, calendario)
	contacto := time.Date(2026, 9, 28, 9, 30, 0, 0, time.UTC)
	plazo, err := r.PlazoRespuesta(t.Context(), contacto)
	if err != nil {
		t.Fatal(err)
	}
	if !plazo.RespuestaHasta.Equal(time.Date(2026, 9, 29, 22, 0, 0, 0, time.UTC)) || plazo.UltimoDia != "2026-09-29" ||
		calendario.recibida.NotificadoEn != contacto || calendario.recibida.Cantidad != 1 ||
		!strings.HasSuffix(plazo.Politica.Referencia, ":"+reglas.BolsaPlazoRespuesta) ||
		!strings.HasSuffix(plazo.CriterioRespuesta.Referencia, ":"+reglas.BolsaFueraDePlazo) ||
		!strings.HasSuffix(plazo.CriterioExpiracion.Referencia, ":"+reglas.BolsaSinRespuestaBaja) ||
		plazo.TratamientoFueraDePlazo != domain.TratamientoFueraDePlazoExigeCausaJustificada ||
		plazo.ConfirmacionExpiracion != domain.ConfirmacionExpiracionRRHH || !plazo.ReglaEjemplo ||
		plazo.ValidarDesde(contacto) != nil {
		t.Fatalf("plazo inesperado: %+v", plazo)
	}
	calendario.err = calendariosdomain.ErrCalendarioNoCubre
	if _, err := r.PlazoRespuesta(t.Context(), contacto); !errors.Is(err, ports.ErrReglasPlazoNoDisponibles) {
		t.Fatalf("sin calendario no se supone un vencimiento: %v", err)
	}
	if _, err := (reglasPlazoLlamamientoDesarrollo{}).PlazoRespuesta(t.Context(), contacto); !errors.Is(err, ports.ErrReglasPlazoNoDisponibles) {
		t.Fatalf("sin catálogo no hay plazo: %v", err)
	}
}

func TestPoliticaDeResolucionDesdeHistoricaOCatalogo(t *testing.T) {
	r := reglasPlazoEjemploPrueba(t, calendarioUnDiaHabilPrueba(t))
	respuesta, expiracion, _, err := r.criterios(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	conReglas := &soporteAltaContratacionTemporalDesarrollo{reglasPlazo: r}
	sinReglas := &soporteAltaContratacionTemporalDesarrollo{reglasPlazo: reglasPlazoLlamamientoDesarrollo{}}
	s := ports.SolicitudResolverLlamamiento{Respuesta: ports.RespuestaLlamamientoAceptada, CriterioValidacionRef: criterioRevisionManualDesarrollo}
	for _, caso := range []struct {
		nombre    string
		soporte   *soporteAltaContratacionTemporalDesarrollo
		respuesta ports.RespuestaLlamamiento
		criterio  string
		esperada  ports.ReferenciaGobernadaComunicacionLlamamiento
		admitida  bool
	}{
		{"histórica sin catálogo", sinReglas, ports.RespuestaLlamamientoAceptada, criterioRevisionManualDesarrollo, politicaManualDesarrollo(), true},
		{"histórica no expira", conReglas, ports.RespuestaLlamamientoExpirada, criterioRevisionManualDesarrollo, politicaManualDesarrollo(), false},
		{"respuesta con catálogo", conReglas, ports.RespuestaLlamamientoRenunciada, respuesta.referencia.Referencia, respuesta.referencia, true},
		{"expiración con catálogo", conReglas, ports.RespuestaLlamamientoExpirada, expiracion.referencia.Referencia, expiracion.referencia, true},
		{"criterio cruzado", conReglas, ports.RespuestaLlamamientoExpirada, respuesta.referencia.Referencia, expiracion.referencia, false},
		{"catálogo ausente", sinReglas, ports.RespuestaLlamamientoAceptada, respuesta.referencia.Referencia, ports.ReferenciaGobernadaComunicacionLlamamiento{}, false},
	} {
		s.Respuesta, s.CriterioValidacionRef = caso.respuesta, caso.criterio
		politica, err := caso.soporte.politicaResolucionDesarrollo(t.Context(), s)
		admitida := err == nil
		if admitida != caso.admitida || (admitida && politica != caso.esperada) {
			t.Errorf("%s: %+v %v", caso.nombre, politica, admitida)
		}
	}
	historica := politicaManualDesarrollo()
	if !politicaResolucionAdmitidaDesarrollo(historica.Referencia, historica.Version, historica.HuellaSHA256) ||
		!politicaResolucionAdmitidaDesarrollo(respuesta.referencia.Referencia, respuesta.referencia.Version, respuesta.referencia.HuellaSHA256) ||
		politicaResolucionAdmitidaDesarrollo(historica.Referencia, 2, historica.HuellaSHA256) ||
		politicaResolucionAdmitidaDesarrollo("otro.catalogo:1:b07.fuera_de_plazo", 1, respuesta.referencia.HuellaSHA256) {
		t.Fatal("antecedentes con política no admitida")
	}
}

// VEC propone y RRHH confirma: sin la confirmación expresa no hay efecto; con
// ella se registra la expiración bajo la regla del catálogo, sin llamar a Bolsa.
func TestConfirmacionExpiracionExigeRRHHYReglaDelCatalogo(t *testing.T) {
	r := reglasPlazoEjemploPrueba(t, calendarioUnDiaHabilPrueba(t))
	_, expiracion, _, err := r.criterios(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	expediente := expedientePuenteBolsaPrueba(t)
	s := ports.SolicitudResolverLlamamiento{ClaveIdempotencia: "31111111-1111-4111-8111-111111111111",
		OrganizacionRef: expediente.Fiscalizado.OrganizacionRef, ExpedienteRef: expediente.Fiscalizado.Referencia,
		LlamamientoRef: "llamamiento:expiracion", ComunicacionRef: "comunicacion:expiracion", VersionEsperada: 2,
		Respuesta: ports.RespuestaLlamamientoExpirada}
	resuelta := time.Date(2026, 9, 30, 8, 0, 0, 0, time.UTC)
	local := ports.ResultadoResolucionLlamamiento{Solicitud: s, Politica: expiracion.referencia,
		EvaluacionPlazoRef: "evaluacion:expiracion", EstadoPlazo: ports.PlazoLlamamientoExpirado,
		ResolucionRef: "resolucion:expiracion", ReciboLocalRef: "recibo:expiracion", AuditoriaRef: "auditoria:expiracion",
		VersionResultante: 3, ResueltaEn: resuelta, Estado: ports.ResultadoComunicacionLlamamientoConfirmado,
		RespuestaHasta: time.Date(2026, 9, 29, 22, 0, 0, 0, time.UTC)}
	servicio := &servicioRevisionManualPrueba{local: local}
	aceptador := &aceptadorRevisionManualPrueba{}
	e := &ejecutorComunicacionLlamamientoDesarrollo{soporte: &soporteAltaContratacionTemporalDesarrollo{reglasPlazo: r},
		servicio: servicio, aceptador: aceptador}
	if _, err := e.confirmarExpiracion(t.Context(), s, expediente); !errors.Is(err, application.ErrValidacionRespuestaLlamamientoPendiente) || servicio.llamadas != 0 {
		t.Fatalf("sin confirmación de RRHH no hay efecto: %v", err)
	}
	s.RevisionRespuestaRRHH, s.RevisionPlazoRRHH, s.CriterioValidacionRef = true, true, criterioRevisionManualDesarrollo
	if _, err := e.confirmarExpiracion(t.Context(), s, expediente); !errors.Is(err, application.ErrComunicacionLlamamientoDenegada) || servicio.llamadas != 0 {
		t.Fatalf("la política histórica no admite expirar: %v", err)
	}
	s.CriterioValidacionRef = expiracion.referencia.Referencia
	local.Solicitud = s
	local.IntencionSiguiente = ports.IntencionOutboxSiguienteCandidato{Solicitud: s, ResolucionRef: local.ResolucionRef,
		LlamamientoRef: s.LlamamientoRef, ClaveIdempotencia: s.ClaveIdempotencia, VersionEsperada: 2, VersionResultante: 3,
		IntencionRef: "intencion:siguiente", ComandoOpacoRef: "comando:siguiente", Estado: ports.OutboxSiguienteCandidatoPendiente,
		ActualizadaEn: resuelta}
	servicio.local = local
	obtenido, err := e.confirmarExpiracion(t.Context(), s, expediente)
	if err != nil || obtenido.EstadoPlazo != ports.PlazoLlamamientoExpirado || obtenido.IntencionSiguiente.Estado != ports.OutboxSiguienteCandidatoPendiente ||
		servicio.llamadas != 1 || aceptador.llamadas != 0 {
		t.Fatalf("expiración confirmada inesperada: %+v %v", obtenido, err)
	}
	servicio.local.Politica = politicaManualDesarrollo()
	if _, err := e.confirmarExpiracion(t.Context(), s, expediente); !errors.Is(err, application.ErrResultadoComunicacionLlamamientoNoConfiable) {
		t.Fatalf("recibo con otra política aceptado: %v", err)
	}
}

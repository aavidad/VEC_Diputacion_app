package application

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type resolutorContextoRecuperacionCoberturaPrueba struct {
	origen       ports.ResolutorContextoAutorizacionAltaV3
	solicitud    ports.SolicitudResolverContextoAutorizacionAltaV3
	organizacion string
	reloj        cobertura.RelojGobiernoOperacionCobertura
	err          error
}

func (r *resolutorContextoRecuperacionCoberturaPrueba) ResolverContextoRecuperacionResultadoCobertura(
	ctx context.Context,
) (ports.ContextoRecuperacionResultadoCobertura, error) {
	if err := ctx.Err(); err != nil {
		return ports.ContextoRecuperacionResultadoCobertura{}, err
	}
	if r.err != nil {
		return ports.ContextoRecuperacionResultadoCobertura{}, r.err
	}
	ahora, err := r.reloj.AhoraGobiernoOperacionCobertura(ctx)
	if err != nil {
		return ports.ContextoRecuperacionResultadoCobertura{}, err
	}
	contexto, err := r.origen.ResolverContextoAutorizacionAltaV3(
		ctx,
		r.solicitud,
	)
	if err != nil {
		return ports.ContextoRecuperacionResultadoCobertura{}, err
	}
	return ports.NuevoContextoRecuperacionResultadoCobertura(
		r.solicitud,
		contexto,
		r.organizacion,
		ahora,
	)
}

type autorizadorLecturaResultadoCoberturaPrueba struct {
	mu                   sync.Mutex
	resultado            ports.ResultadoAutorizacionLecturaResultadoCobertura
	err                  error
	errores              map[int]error
	organizacionAdmitida string
	llamadas             int
	datos                []ports.DatosSolicitudLecturaResultadoCobertura
	cancelar             context.CancelFunc
}

func (a *autorizadorLecturaResultadoCoberturaPrueba) AutorizarLecturaResultadoCobertura(
	ctx context.Context,
	solicitud ports.SolicitudLecturaResultadoCobertura,
) (ports.ResultadoAutorizacionLecturaResultadoCobertura, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	datos, err := solicitud.Datos()
	a.mu.Lock()
	defer a.mu.Unlock()
	a.llamadas++
	if err != nil {
		return 0, err
	}
	a.datos = append(a.datos, datos)
	errAutorizacion := a.err
	if errLlamada, existe := a.errores[a.llamadas]; existe {
		errAutorizacion = errLlamada
	}
	if a.cancelar != nil {
		a.cancelar()
	}
	if a.organizacionAdmitida != "" &&
		datos.OrganizacionRef != a.organizacionAdmitida {
		return ports.AutorizacionLecturaResultadoCoberturaDenegada, nil
	}
	if errAutorizacion != nil {
		return 0, errAutorizacion
	}
	if a.resultado == 0 {
		return ports.AutorizacionLecturaResultadoCoberturaConcedida, nil
	}
	return a.resultado, nil
}

func (a *autorizadorLecturaResultadoCoberturaPrueba) total() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.llamadas
}

func (a *autorizadorLecturaResultadoCoberturaPrueba) ultima() (
	ports.DatosSolicitudLecturaResultadoCobertura,
	bool,
) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.datos) == 0 {
		return ports.DatosSolicitudLecturaResultadoCobertura{}, false
	}
	return a.datos[len(a.datos)-1], true
}

type selladorAmbitoConsultaCoberturaPrueba struct {
	mu       sync.Mutex
	marca    string
	err      error
	bytes    []byte
	llamadas int
}

func (s *selladorAmbitoConsultaCoberturaPrueba) SellarAmbitoOperacionDecisionCobertura(
	ctx context.Context,
	preimagen cobertura.PreimagenAmbitoRecuperacionOperacionDecisionCobertura,
) (ports.ColeccionSellosHMAC, error) {
	if err := ctx.Err(); err != nil {
		return ports.ColeccionSellosHMAC{}, err
	}
	contenido, err := preimagen.Bytes()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.llamadas++
	s.bytes = append([]byte(nil), contenido...)
	if err != nil {
		return ports.ColeccionSellosHMAC{}, err
	}
	if s.err != nil {
		return ports.ColeccionSellosHMAC{}, s.err
	}
	marca := s.marca
	if marca == "" {
		marca = "a"
	}
	return ports.NuevaColeccionSellosHMAC(
		"hmac-sha256:vec.contratacion-temporal."+
			"cobertura-decision.ambito/v1:"+strings.Repeat(marca, 64),
		nil,
	)
}

type lectorResultadoHistoricoCoberturaPrueba struct {
	mu             sync.Mutex
	evidencia      *cobertura.DatosEvidenciaPersistidaResultadoOperacionDecisionCobertura
	noObservable   bool
	contradictorio bool
	err            error
	cancelar       context.CancelFunc
	consultas      int
	efectos        int
}

func (l *lectorResultadoHistoricoCoberturaPrueba) EjecutarLecturaResultadoHistoricoTCB(
	ctx context.Context,
	callback func(
		cobertura.SesionLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
	) error,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return callback(l)
}

func (l *lectorResultadoHistoricoCoberturaPrueba) LeerResultadoHistoricoTCB(
	ctx context.Context,
	consulta cobertura.ConsultaLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
) (cobertura.DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB, error) {
	if err := ctx.Err(); err != nil {
		return cobertura.DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB{},
			err
	}
	if _, err := consulta.DatosLectura(); err != nil {
		return cobertura.DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB{},
			err
	}
	l.mu.Lock()
	l.consultas++
	evidencia, noObservable, contradictorio, errForzado, cancelar :=
		l.evidencia, l.noObservable, l.contradictorio, l.err, l.cancelar
	l.mu.Unlock()
	if cancelar != nil {
		cancelar()
	}
	if errForzado != nil {
		return cobertura.DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB{},
			errForzado
	}
	if contradictorio {
		return cobertura.DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB{
			ObservadaEn: time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC),
			Recibo: cobertura.ReciboOperacionDecisionCobertura{
				ReciboRef: "recibo_contradictorio_resultado_01",
			},
		}, nil
	}
	if noObservable {
		instante := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
		if evidencia != nil {
			instante = evidencia.ObservadaEn
		}
		return cobertura.DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB{
			ObservadaEn: instante,
		}, nil
	}
	if evidencia == nil {
		return cobertura.DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB{},
			nil
	}
	return cobertura.DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB{
		Encontrado:  true,
		Reserva:     evidencia.Reserva,
		Recibo:      evidencia.Recibo,
		ObservadaEn: evidencia.ObservadaEn,
	}, nil
}

func (l *lectorResultadoHistoricoCoberturaPrueba) producirEfecto() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.efectos++
}

func (l *lectorResultadoHistoricoCoberturaPrueba) totales() (int, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.consultas, l.efectos
}

func contextoRecuperacionDesdeDecisionPrueba(
	escenario *escenarioConfirmacionCobertura,
) *resolutorContextoRecuperacionCoberturaPrueba {
	return &resolutorContextoRecuperacionCoberturaPrueba{
		origen: escenario.base.contextos,
		solicitud: ports.SolicitudResolverContextoAutorizacionAltaV3{
			AutenticacionRef: escenario.solicitud.AutenticacionRef,
			SesionRef:        escenario.solicitud.SesionRef,
			PerfilRef:        escenario.solicitud.PerfilRef,
		},
		organizacion: escenario.solicitud.OrganizacionRef,
		reloj:        escenario.base.reloj,
	}
}

func solicitudConsultaDesdeDecisionPrueba(
	escenario *escenarioConfirmacionCobertura,
) SolicitudConsultaResultadoCobertura {
	return SolicitudConsultaResultadoCobertura{
		ClaveIdempotencia: escenario.solicitud.ClaveIdempotencia,
		ExpedienteRef:     escenario.solicitud.ExpedienteRef,
	}
}

func evidenciaHistoricaConfirmacionPrueba(
	t *testing.T,
	idempotencia *idempotenciaConfirmacionPrueba,
	recibo cobertura.ReciboOperacionDecisionCobertura,
) cobertura.DatosEvidenciaPersistidaResultadoOperacionDecisionCobertura {
	t.Helper()
	datosSolicitud, err := idempotencia.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datos := idempotencia.datos
	return cobertura.DatosEvidenciaPersistidaResultadoOperacionDecisionCobertura{
		Reserva: cobertura.DatosReservaTerminalOperacionDecisionCobertura{
			OrganizacionRef:        datosSolicitud.OrganizacionRef,
			ExpedienteRef:          datosSolicitud.ExpedienteRef,
			VersionExpediente:      datosSolicitud.VersionExpediente,
			ReservaRef:             datos.ReservaRef,
			ReciboRef:              datos.ReciboRef,
			ActuacionRef:           datos.ActuacionRef,
			AuditoriaRef:           datos.AuditoriaRef,
			EventoRef:              datos.EventoRef,
			CorrelacionVECRef:      datos.CorrelacionVECRef,
			DecisionVECRef:         datos.DecisionVECRef,
			AmbitoIdempotenciaHMAC: datos.AmbitoIdempotenciaHMAC,
			HuellaSemanticaHMAC:    datos.HuellaSemanticaHMAC,
			RevisionCercado:        datos.RevisionCercado,
			ObservadaEnDB:          datos.ObservadaEnDB,
		},
		Recibo:      recibo,
		ObservadaEn: recibo.ConfirmadaEn,
	}
}

func nuevoServicioConsultaResultadoPrueba(
	t *testing.T,
	contextos ports.ResolutorContextoRecuperacionResultadoCobertura,
	accesos ports.AutorizadorLecturaResultadoCobertura,
	sellador cobertura.SelladorAmbitoOperacionDecisionCobertura,
	reloj cobertura.RelojGobiernoOperacionCobertura,
	ejecutor cobertura.EjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
) *ServicioConsultaResultadoCobertura {
	t.Helper()
	lector, err :=
		cobertura.NuevoLectorResultadoHistoricoOperacionDecisionCoberturaTCB(
			ejecutor,
		)
	if err != nil {
		t.Fatal(err)
	}
	servicio, err := NuevoServicioConsultaResultadoCobertura(
		contextos,
		accesos,
		sellador,
		reloj,
		lector,
	)
	if err != nil {
		t.Fatal(err)
	}
	return servicio
}

func comprobarAutorizacionConsultaResultado(
	t *testing.T,
	accesos *autorizadorLecturaResultadoCoberturaPrueba,
	organizacionRef string,
	expedienteRef string,
) {
	t.Helper()
	datos, existe := accesos.ultima()
	if !existe ||
		datos.OrganizacionRef != organizacionRef ||
		datos.ExpedienteRef != expedienteRef ||
		datos.Accion != ports.AccionConsultarResultadoCobertura ||
		datos.Finalidad != ports.FinalidadRecuperarResultadoCobertura {
		t.Fatalf("autorización de lectura no ligada: %#v", datos)
	}
}

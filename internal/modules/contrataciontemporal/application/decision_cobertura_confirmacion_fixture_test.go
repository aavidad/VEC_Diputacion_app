package application

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type resolutorMotivoConfirmacionPrueba struct {
	mu       sync.Mutex
	llamadas int
	err      error
}

func (r *resolutorMotivoConfirmacionPrueba) ResolverClave(
	_ context.Context,
	_ domain.ClaveCatalogo,
	_ time.Time,
) (cobertura.ResolucionMotivoDecisionCobertura, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llamadas++
	if r.err != nil {
		return cobertura.ResolucionMotivoDecisionCobertura{}, r.err
	}
	return cobertura.ResolucionMotivoDecisionCobertura{},
		errors.New("motivo inesperado en escenario sin desviación")
}

type selladorConfirmacionPrueba struct {
	mu       sync.Mutex
	llamadas int
	err      error
	invalido bool
}

func (s *selladorConfirmacionPrueba) SellarOperacionDecisionCobertura(
	ctx context.Context,
	_ cobertura.PreimagenesOperacionDecisionCobertura,
) (cobertura.SellosOperacionDecisionCobertura, error) {
	if err := ctx.Err(); err != nil {
		return cobertura.SellosOperacionDecisionCobertura{}, err
	}
	s.mu.Lock()
	s.llamadas++
	errForzado := s.err
	invalido := s.invalido
	s.mu.Unlock()
	if errForzado != nil {
		return cobertura.SellosOperacionDecisionCobertura{}, errForzado
	}
	if invalido {
		return cobertura.SellosOperacionDecisionCobertura{}, nil
	}
	ambito, err := ports.NuevaColeccionSellosHMAC(
		"hmac-sha256:vec.contratacion-temporal."+
			"cobertura-decision.ambito/v1:"+strings.Repeat("a", 64),
		nil,
	)
	if err != nil {
		return cobertura.SellosOperacionDecisionCobertura{}, err
	}
	semantica, err := ports.NuevaColeccionSellosHMAC(
		"hmac-sha256:vec.contratacion-temporal."+
			"cobertura-decision.semantica/v1:"+strings.Repeat("b", 64),
		nil,
	)
	return cobertura.SellosOperacionDecisionCobertura{
		AmbitosIdempotenciaHMAC: ambito,
		HuellasSemanticasHMAC:   semantica,
	}, err
}

func (s *selladorConfirmacionPrueba) total() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.llamadas
}

type modoIdempotenciaConfirmacionPrueba string

const (
	idempotenciaPropietaria modoIdempotenciaConfirmacionPrueba = "propietaria"
	idempotenciaOcupada     modoIdempotenciaConfirmacionPrueba = "ocupada"
	idempotenciaReplay      modoIdempotenciaConfirmacionPrueba = "replay"
	idempotenciaExclusion   modoIdempotenciaConfirmacionPrueba = "exclusion"
)

type idempotenciaConfirmacionPrueba struct {
	mu              sync.Mutex
	modo            modoIdempotenciaConfirmacionPrueba
	expediente      domain.Expediente
	analisisRef     string
	analisisHuella  string
	reloj           *relojCoberturaAplicacionPrueba
	consultas       int
	reservas        int
	consulta        cobertura.SolicitudConsultarOperacionDecisionCoberturaConfirmada
	solicitud       cobertura.SolicitudReservarOperacionDecisionCobertura
	datos           cobertura.DatosReservaPropietariaOperacionDecisionCobertura
	replay          cobertura.PreparacionOperacionDecisionCobertura
	cancelarReserva context.CancelFunc
}

func (i *idempotenciaConfirmacionPrueba) ConsultarOperacionDecisionCoberturaConfirmada(
	ctx context.Context,
	solicitud cobertura.SolicitudConsultarOperacionDecisionCoberturaConfirmada,
) (cobertura.PreparacionOperacionDecisionCobertura, bool, error) {
	if err := ctx.Err(); err != nil {
		return cobertura.PreparacionOperacionDecisionCobertura{}, false, err
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.consultas++
	i.consulta = solicitud
	if i.modo == idempotenciaReplay {
		return i.replay, true, nil
	}
	return cobertura.PreparacionOperacionDecisionCobertura{}, false, nil
}

func (i *idempotenciaConfirmacionPrueba) ReservarOReapropiarOperacionDecisionCobertura(
	ctx context.Context,
	solicitud cobertura.SolicitudReservarOperacionDecisionCobertura,
) (cobertura.PreparacionOperacionDecisionCobertura, error) {
	if err := ctx.Err(); err != nil {
		return cobertura.PreparacionOperacionDecisionCobertura{}, err
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.reservas++
	if i.cancelarReserva != nil {
		i.cancelarReserva()
	}
	datosSolicitud, err := solicitud.Datos()
	if err != nil {
		return cobertura.PreparacionOperacionDecisionCobertura{}, err
	}
	if i.modo == idempotenciaOcupada ||
		(i.modo == idempotenciaExclusion && i.reservas > 1) {
		return cobertura.NuevaPreparacionOperacionDecisionCoberturaOcupada(
			solicitud,
			datosSolicitud.AmbitoIdempotenciaHMAC,
			datosSolicitud.HuellaSemanticaHMAC,
		)
	}
	ahora := i.reloj.Ahora()
	expediente := i.expediente.Clonar()
	datos := cobertura.DatosReservaPropietariaOperacionDecisionCobertura{
		ReservaRef:              "reserva_confirmacion_cobertura_012345",
		ReciboRef:               "recibo_confirmacion_cobertura_012345",
		ActuacionRef:            "actuacion_confirmacion_cobertura_012345",
		AuditoriaRef:            "auditoria_confirmacion_cobertura_012345",
		EventoRef:               "evento_confirmacion_cobertura_012345",
		CorrelacionVECRef:       "correlacion_11111111111111111111111111111111",
		DecisionVECRef:          "dec_11111111111111111111111111111111",
		AnalisisRef:             i.analisisRef,
		AnalisisHuellaSHA256:    i.analisisHuella,
		TokenPropietarioSHA256:  datosSolicitud.TokenPropietarioSHA256,
		AmbitoIdempotenciaHMAC:  datosSolicitud.AmbitoIdempotenciaHMAC,
		HuellaSemanticaHMAC:     datosSolicitud.HuellaSemanticaHMAC,
		AgregadoAnterior:        &expediente,
		RevisionCercadoAnterior: 0,
		RevisionCercado:         1,
		ObservadaEnDB:           ahora,
		PropiedadHasta:          ahora.Add(5 * time.Second),
	}
	preparacion, err :=
		cobertura.NuevaPreparacionOperacionDecisionCoberturaPropietaria(
			solicitud,
			datos,
		)
	if err == nil {
		i.solicitud = solicitud
		i.datos = datos
	}
	return preparacion, err
}

func (i *idempotenciaConfirmacionPrueba) instalarReplay(
	t *testing.T,
	recibo cobertura.ReciboOperacionDecisionCobertura,
) {
	t.Helper()
	i.mu.Lock()
	defer i.mu.Unlock()
	datosSolicitud, err := i.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	terminal, err :=
		cobertura.RehidratarReservaTerminalOperacionDecisionCobertura(
			i.consulta,
			cobertura.DatosReservaTerminalOperacionDecisionCobertura{
				OrganizacionRef:   datosSolicitud.OrganizacionRef,
				ExpedienteRef:     datosSolicitud.ExpedienteRef,
				VersionExpediente: datosSolicitud.VersionExpediente,
				ReservaRef:        i.datos.ReservaRef, ReciboRef: i.datos.ReciboRef,
				ActuacionRef:           i.datos.ActuacionRef,
				AuditoriaRef:           i.datos.AuditoriaRef,
				EventoRef:              i.datos.EventoRef,
				CorrelacionVECRef:      i.datos.CorrelacionVECRef,
				DecisionVECRef:         i.datos.DecisionVECRef,
				AmbitoIdempotenciaHMAC: i.datos.AmbitoIdempotenciaHMAC,
				HuellaSemanticaHMAC:    i.datos.HuellaSemanticaHMAC,
				RevisionCercado:        i.datos.RevisionCercado,
				ObservadaEnDB:          i.datos.ObservadaEnDB,
			},
		)
	if err != nil {
		t.Fatal(err)
	}
	i.replay, err =
		cobertura.NuevaPreparacionOperacionDecisionCoberturaConfirmada(
			i.consulta,
			terminal,
			recibo,
		)
	if err != nil {
		t.Fatal(err)
	}
	i.modo = idempotenciaReplay
}

type preparadorVECConfirmacionPrueba struct {
	mu        sync.Mutex
	reloj     *relojCoberturaAplicacionPrueba
	conceder  bool
	llamadas  int
	candidata puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3
}

func (p *preparadorVECConfirmacionPrueba) PrepararRegistroCompuestoSolicitudLigadaV3(
	ctx context.Context,
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	resultado dominiovec.ResultadoContextoActorRegistradoV2,
	generador puertosvec.GeneradorReferenciaDecisionAutorizacion,
) (
	dominiovec.DecisionAutorizacionLigadaV3,
	puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3,
	error,
) {
	if err := ctx.Err(); err != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{},
			puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{},
			err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.llamadas++
	decisionRef, err := generador.NuevaReferenciaDecisionAutorizacion()
	if err != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{},
			puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{},
			err
	}
	decision, candidata, err := candidataConfirmacionCoberturaPrueba(
		solicitud,
		resultado,
		decisionRef,
		p.reloj.Ahora(),
		p.conceder,
	)
	if err == nil {
		p.candidata = candidata
	}
	return decision, candidata, err
}

type transaccionConfirmacionPrueba struct {
	mu           sync.Mutex
	idempotencia *idempotenciaConfirmacionPrueba
	vec          *preparadorVECConfirmacionPrueba
	reloj        *relojCoberturaAplicacionPrueba
	aplicada     *cobertura.ResultadoAplicadoOperacionDecisionCobertura
	ambigua      bool
	errorRetorno error
	cancelar     context.CancelFunc
	llamadas     int
}

func (t *transaccionConfirmacionPrueba) ConfirmarOperacionDecisionCobertura(
	_ context.Context,
	orden cobertura.OrdenOperacionDecisionCobertura,
) (cobertura.ResultadoConfirmacionOperacionDecisionCobertura, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.llamadas++
	if t.ambigua {
		if t.cancelar != nil {
			t.cancelar()
		}
		return cobertura.ResultadoConfirmacionOperacionDecisionCobertura{},
			errors.New("respuesta perdida")
	}
	recibo, err := reciboConfirmacionCoberturaPrueba(
		t.idempotencia,
		t.vec,
		t.reloj.Ahora(),
		t.aplicada,
	)
	if err != nil {
		return cobertura.ResultadoConfirmacionOperacionDecisionCobertura{}, err
	}
	resultado, err :=
		cobertura.NuevaResultadoConfirmacionOperacionDecisionCobertura(
			orden,
			recibo,
		)
	if t.cancelar != nil {
		t.cancelar()
	}
	if err == nil && t.errorRetorno != nil {
		return resultado, t.errorRetorno
	}
	return resultado, err
}

type reconciliadorConfirmacionPrueba struct {
	mu           sync.Mutex
	reloj        *relojCoberturaAplicacionPrueba
	idempotencia *idempotenciaConfirmacionPrueba
	vec          *preparadorVECConfirmacionPrueba
	aplicada     *cobertura.ResultadoAplicadoOperacionDecisionCobertura
	confirmar    bool
	cancelar     context.CancelFunc
	llamadas     int
}

func (i *idempotenciaConfirmacionPrueba) totales() (int, int) {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.consultas, i.reservas
}

func (p *preparadorVECConfirmacionPrueba) total() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.llamadas
}

func (t *transaccionConfirmacionPrueba) total() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.llamadas
}

func (r *reconciliadorConfirmacionPrueba) total() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.llamadas
}

func (r *reconciliadorConfirmacionPrueba) ReconciliarResultadoAmbiguoOperacionDecisionCobertura(
	_ context.Context,
	solicitud cobertura.SolicitudReconciliacionOperacionDecisionCobertura,
) (cobertura.ResultadoReconciliacionOperacionDecisionCobertura, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llamadas++
	if r.confirmar {
		recibo, err := reciboConfirmacionCoberturaPrueba(
			r.idempotencia,
			r.vec,
			r.reloj.Ahora(),
			r.aplicada,
		)
		if err != nil {
			return cobertura.ResultadoReconciliacionOperacionDecisionCobertura{},
				err
		}
		resultado, err :=
			cobertura.NuevaResultadoReconciliacionConfirmadaOperacionDecisionCobertura(
				solicitud,
				recibo,
				r.reloj.Ahora(),
			)
		if r.cancelar != nil {
			r.cancelar()
		}
		return resultado, err
	}
	return cobertura.NuevaResultadoReconciliacionNoConcluyenteOperacionDecisionCobertura(
		solicitud,
		r.reloj.Ahora(),
	)
}

func reciboConfirmacionCoberturaPrueba(
	idempotencia *idempotenciaConfirmacionPrueba,
	vec *preparadorVECConfirmacionPrueba,
	confirmadaEn time.Time,
	aplicada *cobertura.ResultadoAplicadoOperacionDecisionCobertura,
) (cobertura.ReciboOperacionDecisionCobertura, error) {
	idempotencia.mu.Lock()
	datosReserva := idempotencia.datos
	idempotencia.mu.Unlock()
	vec.mu.Lock()
	resumen, err := vec.candidata.Resumen()
	conceder := vec.conceder
	vec.mu.Unlock()
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{}, err
	}
	datosVEC, err := resumen.Datos()
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{}, err
	}
	recibo := cobertura.ReciboOperacionDecisionCobertura{
		ReciboRef: datosReserva.ReciboRef, ReservaRef: datosReserva.ReservaRef,
		AuditoriaRef:            datosReserva.AuditoriaRef,
		CorrelacionVECRef:       datosReserva.CorrelacionVECRef,
		DecisionVECRef:          datosVEC.DecisionRef,
		DecisionVECHuellaSHA256: datosVEC.DecisionHuellaSHA256,
		CodigoProbatorioVEC:     datosVEC.CodigoProbatorio,
		ConcedidaVEC:            datosVEC.Concedida,
		RevisionCercado:         datosReserva.RevisionCercado,
		AmbitoIdempotenciaHMAC:  datosReserva.AmbitoIdempotenciaHMAC,
		HuellaSemanticaHMAC:     datosReserva.HuellaSemanticaHMAC,
		ConfirmadaEn:            confirmadaEn,
	}
	if conceder && aplicada != nil {
		copia := *aplicada
		recibo.Aplicada = &copia
	} else {
		recibo.DenegadaVEC =
			&cobertura.ResultadoDenegadoVECOperacionDecisionCobertura{}
	}
	return recibo, nil
}

type escenarioConfirmacionCobertura struct {
	base          *escenarioPresentacionCobertura
	motivos       *resolutorMotivoConfirmacionPrueba
	sellador      *selladorConfirmacionPrueba
	idempotencia  *idempotenciaConfirmacionPrueba
	vec           *preparadorVECConfirmacionPrueba
	transaccion   *transaccionConfirmacionPrueba
	reconciliador *reconciliadorConfirmacionPrueba
	servicio      *ServicioConfirmacionDecisionCobertura
	solicitud     SolicitudDecidirCobertura
}

func nuevoEscenarioConfirmacionCobertura(
	t *testing.T,
	conceder bool,
) *escenarioConfirmacionCobertura {
	return nuevoEscenarioConfirmacionCoberturaConVias(t, conceder, 1)
}

func nuevoEscenarioConfirmacionCoberturaConVias(
	t *testing.T,
	conceder bool,
	cantidadVias int,
) *escenarioConfirmacionCobertura {
	t.Helper()
	base := nuevoEscenarioPresentacionCobertura(
		t,
		viasPresentacionCoberturaPrueba(cantidadVias),
	)
	presentacion, err := base.servicio.Proponer(
		context.Background(),
		base.solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitudO3, _ := cobertura.NuevaSolicitudInstantaneaAnalisisDurableO3(
		base.expediente.OrganizacionRef,
		base.expediente.Referencia,
		base.expediente.Version,
	)
	instantanea, err := cobertura.ObtenerInstantaneaAnalisisDurableO3(
		context.Background(),
		base.analisis,
		solicitudO3,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, analisisRef, analisisHuella, err :=
		instantanea.DesplegarPara(solicitudO3)
	if err != nil {
		t.Fatal(err)
	}
	escenario := &escenarioConfirmacionCobertura{
		base:     base,
		motivos:  &resolutorMotivoConfirmacionPrueba{},
		sellador: &selladorConfirmacionPrueba{},
		idempotencia: &idempotenciaConfirmacionPrueba{
			modo:           idempotenciaPropietaria,
			expediente:     base.expediente,
			analisisRef:    analisisRef,
			analisisHuella: analisisHuella,
			reloj:          base.global.entorno.reloj,
		},
		vec: &preparadorVECConfirmacionPrueba{
			reloj:    base.global.entorno.reloj,
			conceder: conceder,
		},
		solicitud: SolicitudDecidirCobertura{
			AutenticacionRef:   base.solicitud.AutenticacionRef,
			SesionRef:          base.solicitud.SesionRef,
			PerfilRef:          base.solicitud.PerfilRef,
			OrganizacionRef:    base.solicitud.OrganizacionRef,
			ExpedienteRef:      base.solicitud.ExpedienteRef,
			VersionEsperada:    base.solicitud.VersionEsperada,
			ClaveIdempotencia:  "12345678-1234-4234-8234-123456789abc",
			IdentidadSemantica: presentacion.IdentidadSemantica,
			ViaElegida:         presentacion.ViaRecomendada,
		},
	}
	escenario.transaccion = &transaccionConfirmacionPrueba{
		idempotencia: escenario.idempotencia,
		vec:          escenario.vec,
		reloj:        base.global.entorno.reloj,
	}
	if conceder {
		aplicada, err := resultadoAplicadoEsperadoConfirmacionCobertura(
			t,
			base,
			analisisRef,
			analisisHuella,
			escenario.idempotencia,
		)
		if err != nil {
			t.Fatal(err)
		}
		escenario.transaccion.aplicada = &aplicada
	}
	escenario.reconciliador = &reconciliadorConfirmacionPrueba{
		reloj:        base.global.entorno.reloj,
		idempotencia: escenario.idempotencia,
		vec:          escenario.vec,
		aplicada:     escenario.transaccion.aplicada,
	}
	escenario.servicio, err = NuevoServicioConfirmacionDecisionCobertura(
		base.contextos,
		escenario.motivos,
		escenario.sellador,
		escenario.idempotencia,
		base.analisis,
		base.reloj,
		base.gobierno,
		base.global.preparador,
		escenario.vec,
		escenario.transaccion,
		escenario.reconciliador,
	)
	if err != nil {
		t.Fatal(err)
	}
	return escenario
}

func resultadoAplicadoEsperadoConfirmacionCobertura(
	t *testing.T,
	base *escenarioPresentacionCobertura,
	analisisRef string,
	analisisHuella string,
	idempotencia *idempotenciaConfirmacionPrueba,
) (cobertura.ResultadoAplicadoOperacionDecisionCobertura, error) {
	t.Helper()
	instanteInicial := base.global.entorno.reloj.Ahora()
	base.global.generador.mu.Lock()
	siguienteInicial := base.global.generador.siguiente
	base.global.generador.mu.Unlock()
	defer func() {
		base.global.generador.mu.Lock()
		base.global.generador.siguiente = siguienteInicial
		base.global.generador.mu.Unlock()
		base.global.entorno.reloj.fijar(instanteInicial)
	}()
	if base.expediente.Analisis == nil {
		return cobertura.ResultadoAplicadoOperacionDecisionCobertura{},
			errors.New("expediente de prueba sin análisis")
	}
	datosGlobales, err := nuevosDatosPreparacionGlobalCobertura(
		analisisRef,
		analisisHuella,
		base.global.catalogo,
		base.global.politica,
		base.expediente.OrganizacionRef,
		base.expediente.Referencia,
		base.expediente.Version,
		base.expediente.Analisis.CategoriaRef,
		base.expediente.Analisis.Periodo,
	)
	if err != nil {
		return cobertura.ResultadoAplicadoOperacionDecisionCobertura{}, err
	}
	preparacion, err := base.global.preparador.Preparar(
		context.Background(),
		datosGlobales,
	)
	if err != nil {
		return cobertura.ResultadoAplicadoOperacionDecisionCobertura{}, err
	}
	instanteDecision := base.global.entorno.reloj.Ahora()
	datosPropuesta, err := preparacion.DatosCrearPropuestaEn(instanteDecision)
	if err != nil {
		return cobertura.ResultadoAplicadoOperacionDecisionCobertura{}, err
	}
	propuesta, err := domain.CrearPropuestaDecisionCobertura(datosPropuesta)
	if err != nil {
		return cobertura.ResultadoAplicadoOperacionDecisionCobertura{}, err
	}
	datosVinculo, err := base.contextos.contexto.Vinculo.Datos()
	if err != nil {
		return cobertura.ResultadoAplicadoOperacionDecisionCobertura{}, err
	}
	siguiente, err := base.expediente.RegistrarDecisionCoberturaGobernada(
		base.expediente.Version,
		domain.DatosAdoptarDecisionCobertura{
			PerfilRef:  datosVinculo.PerfilActivoRef,
			ViaElegida: propuesta.ViaPropuesta(),
		},
		propuesta,
		domain.DatosActuacion{
			AccionClave:   domain.AccionDecidirCoberturaGobernada,
			ActorRef:      datosVinculo.PrincipalID,
			UnidadRef:     "unidad_rrhh_presentacion_cobertura_01",
			ReciboRef:     "recibo_confirmacion_cobertura_012345",
			RealizadaEn:   instanteDecision,
			FaseDestino:   "decision_cobertura",
			EstadoDestino: domain.EstadoEnCurso,
		},
	)
	if err != nil {
		return cobertura.ResultadoAplicadoOperacionDecisionCobertura{}, err
	}
	publicacion := siguiente.DecisionesCobertura[len(siguiente.DecisionesCobertura)-1]
	return cobertura.ResultadoAplicadoOperacionDecisionCobertura{
		DecisionCoberturaRef:    publicacion.Referencia,
		DecisionCoberturaHuella: publicacion.HuellaSHA256,
		VersionResultante:       siguiente.Version,
		EventoRef:               "evento_confirmacion_cobertura_012345",
		ActuacionRef:            "actuacion_confirmacion_cobertura_012345",
	}, nil
}

func candidataConfirmacionCoberturaPrueba(
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	resultado dominiovec.ResultadoContextoActorRegistradoV2,
	decisionRef string,
	instante time.Time,
	conceder bool,
) (
	dominiovec.DecisionAutorizacionLigadaV3,
	puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3,
	error,
) {
	datosSolicitud, err := solicitud.Datos()
	if err != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{},
			puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{}, err
	}
	datosVinculo, err := datosSolicitud.VinculoAutenticacionActor.Datos()
	if err != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{},
			puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{}, err
	}
	accionRol := datosSolicitud.Accion
	if !conceder {
		accionRol = "contratacion_temporal.cobertura.accion_ajena"
	}
	version := dominiovec.VersionRol{
		RolID: "tecnico_rrhh_confirmacion", Version: 1,
		Nombre: "Técnico de RRHH",
		Estado: dominiovec.EstadoVersionRolPublicada,
		Concesiones: []dominiovec.ConcesionRol{{
			Accion:         accionRol,
			ModuloID:       datosSolicitud.Recurso.ModuloID,
			TipoRecurso:    datosSolicitud.Recurso.Tipo,
			Finalidades:    []string{datosSolicitud.Finalidad},
			GarantiaMinima: dominiovec.AuthAssuranceSubstantial,
		}},
		PublicadaPor: "responsable_seguridad_confirmacion",
		PublicadaEn:  instante.Add(-24 * time.Hour),
	}
	ambitos := make([]dominiovec.AmbitoPerfil, 0, len(datosSolicitud.Recurso.Ambitos))
	for clave, valor := range datosSolicitud.Recurso.Ambitos {
		ambitos = append(ambitos, dominiovec.AmbitoPerfil{
			Clave: clave, Valores: []string{valor},
		})
	}
	huellaCatalogo, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{},
			puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{}, err
	}
	instantanea := dominiovec.InstantaneaAutorizacion{
		AsignacionPerfil: dominiovec.AsignacionPerfil{
			AsignacionID: "asignacion_rrhh_confirmacion_012345",
			Version:      1, PerfilActivoRef: datosVinculo.PerfilActivoRef,
			PrincipalID:   datosVinculo.PrincipalID,
			VersionRolRef: version.Referencia(),
			Estado:        dominiovec.EstadoAsignacionPerfilActiva,
			Ambitos:       ambitos,
			VigenteDesde:  instante.Add(-time.Hour),
			VigenteHasta:  instante.Add(time.Hour),
			EmitidaPor:    "administrador_identidades_confirmacion",
			EmitidaEn:     instante.Add(-2 * time.Hour),
		},
		VersionRol: version,
		ControlVigenciaVersionRol: dominiovec.ControlVigenciaVersionRol{
			VersionRolRef: version.Referencia(), Revision: 1,
			Estado:         dominiovec.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor,
			ActualizadoEn:  version.PublicadaEn,
		},
		RevisionCatalogoPoliticas:     1,
		CatalogoPoliticasHuellaSHA256: huellaCatalogo,
	}
	evidencia, err := dominiovec.NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud,
		instantanea,
		decisionRef,
		instante,
		instante.Add(2*time.Second),
	)
	if err != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{},
			puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{}, err
	}
	decision, err := dominiovec.NuevaDecisionAutorizacionLigadaV3(
		solicitud,
		evidencia,
	)
	if err != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{},
			puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{}, err
	}
	candidata, err :=
		puertosvec.NuevaCandidataRegistroDecisionAutorizacionLigadaV3(
			solicitud,
			decision,
			datosSolicitud.ReferenciaMotivo,
			resultado,
		)
	return decision, candidata, err
}

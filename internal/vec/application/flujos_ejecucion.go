package application

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrDependenciaEjecucionFlujosRequerida = errors.New("vec: dependencia de ejecucion de flujos requerida")
	ErrOrdenEjecucionFlujoInvalida         = errors.New("vec: orden de ejecucion de flujo invalida")
)

type ServicioEjecucionFlujos struct {
	definiciones    ports.ConsultaDefinicionesFlujo
	instancias      ports.ConsultaInstanciasFlujo
	repositorio     ports.RepositorioInstanciasFlujo
	evaluador       ports.EvaluadorReglasFlujo
	decisiones      ports.RegistroDecisionesReglaFlujo
	aprobaciones    ports.VerificadorAprobacionesFlujo
	autorizador     ports.Autorizador
	identificadores ports.GeneradorIDInstanciaFlujo
	reloj           ports.Reloj
}

func NuevoServicioEjecucionFlujos(
	definiciones ports.ConsultaDefinicionesFlujo,
	instancias ports.ConsultaInstanciasFlujo,
	repositorio ports.RepositorioInstanciasFlujo,
	evaluador ports.EvaluadorReglasFlujo,
	decisiones ports.RegistroDecisionesReglaFlujo,
	aprobaciones ports.VerificadorAprobacionesFlujo,
	autorizador ports.Autorizador,
	identificadores ports.GeneradorIDInstanciaFlujo,
	reloj ports.Reloj,
) (*ServicioEjecucionFlujos, error) {
	if definiciones == nil || instancias == nil || repositorio == nil || evaluador == nil || decisiones == nil ||
		aprobaciones == nil || autorizador == nil || identificadores == nil || reloj == nil {
		return nil, ErrDependenciaEjecucionFlujosRequerida
	}
	return &ServicioEjecucionFlujos{
		definiciones:    definiciones,
		instancias:      instancias,
		repositorio:     repositorio,
		evaluador:       evaluador,
		decisiones:      decisiones,
		aprobaciones:    aprobaciones,
		autorizador:     autorizador,
		identificadores: identificadores,
		reloj:           reloj,
	}, nil
}

type OrdenIniciarInstanciaFlujo struct {
	Principal      domain.Principal
	PerfilActivo   string
	Finalidad      string
	DefinicionID   string
	Version        int
	EntidadRef     string
	Motivo         string
	CorrelacionRef string
}

func (s *ServicioEjecucionFlujos) IniciarInstancia(
	ctx context.Context,
	orden OrdenIniciarInstanciaFlujo,
) (domain.InstanciaFlujo, error) {
	if err := validarContextoEjecucionFlujo(ctx, orden.Principal, orden.PerfilActivo, orden.Finalidad, orden.Motivo, orden.CorrelacionRef); err != nil {
		return domain.InstanciaFlujo{}, err
	}
	if orden.Version < 1 || orden.DefinicionID != strings.TrimSpace(orden.DefinicionID) ||
		orden.EntidadRef != strings.TrimSpace(orden.EntidadRef) || orden.EntidadRef == "" {
		return domain.InstanciaFlujo{}, ErrOrdenEjecucionFlujoInvalida
	}
	definicion, err := s.definiciones.ObtenerDefinicionFlujo(ctx, orden.DefinicionID, orden.Version)
	if err != nil {
		return domain.InstanciaFlujo{}, err
	}
	if definicion.Estado != domain.EstadoDefinicionFlujoPublicada {
		return domain.InstanciaFlujo{}, domain.ErrDefinicionFlujoNoPublicada
	}
	if !orden.Principal.AuthAssurance.Cumple(definicion.GarantiaInicio) {
		return domain.InstanciaFlujo{}, domain.ErrGarantiaInsuficiente
	}
	decision, err := exigirDecisionAutorizacion(
		ctx,
		s.autorizador,
		s.reloj,
		orden.Principal,
		orden.PerfilActivo,
		definicion.AccionInicio,
		domain.RecursoAutorizable{
			Referencia: orden.EntidadRef,
			ModuloID:   definicion.ModuloID,
			Tipo:       definicion.TipoEntidad,
			Atributos: map[string]string{
				"definicion_ref": definicion.Referencia(),
				"estado_inicial": definicion.EstadoInicial,
			},
		},
		orden.Finalidad,
		orden.CorrelacionRef,
		orden.Motivo,
		usoCamposDecisionNoAplicables,
	)
	if err != nil {
		return domain.InstanciaFlujo{}, err
	}
	instante := s.reloj.Ahora().UTC()
	if instante.IsZero() || !decision.VigenteEn(instante) {
		return domain.InstanciaFlujo{}, domain.ErrDecisionAutorizacionInvalida
	}
	identificador, err := s.identificadores.NuevoIDInstanciaFlujo()
	if err != nil {
		return domain.InstanciaFlujo{}, fmt.Errorf("crear identificador de instancia: %w", err)
	}
	instancia, err := domain.IniciarInstanciaFlujo(
		definicion,
		identificador,
		orden.EntidadRef,
		orden.Principal.ID,
		instante,
	)
	if err != nil {
		return domain.InstanciaFlujo{}, err
	}
	traza, evento, err := evidenciaInicioInstanciaFlujo(
		definicion,
		instancia,
		orden.Principal,
		orden.PerfilActivo,
		decision.DecisionRef,
		orden.Finalidad,
		orden.Motivo,
		orden.CorrelacionRef,
	)
	if err != nil {
		return domain.InstanciaFlujo{}, err
	}
	if err := s.repositorio.ConfirmarInicioInstanciaFlujo(ctx, instancia, traza, evento); err != nil {
		return domain.InstanciaFlujo{}, fmt.Errorf("confirmar inicio de flujo: %w", err)
	}
	return instancia, nil
}

type OrdenAplicarTransicionFlujo struct {
	Principal       domain.Principal
	PerfilActivo    string
	Finalidad       string
	InstanciaID     string
	TransicionClave string
	AprobacionRef   string
	Motivo          string
	CorrelacionRef  string
}

func (s *ServicioEjecucionFlujos) AplicarTransicion(
	ctx context.Context,
	orden OrdenAplicarTransicionFlujo,
) (domain.InstanciaFlujo, error) {
	if err := validarContextoEjecucionFlujo(ctx, orden.Principal, orden.PerfilActivo, orden.Finalidad, orden.Motivo, orden.CorrelacionRef); err != nil {
		return domain.InstanciaFlujo{}, err
	}
	if orden.InstanciaID != strings.TrimSpace(orden.InstanciaID) || orden.InstanciaID == "" ||
		orden.TransicionClave != strings.TrimSpace(orden.TransicionClave) || orden.TransicionClave == "" ||
		orden.AprobacionRef != strings.TrimSpace(orden.AprobacionRef) {
		return domain.InstanciaFlujo{}, ErrOrdenEjecucionFlujoInvalida
	}
	instancia, err := s.instancias.ObtenerInstanciaFlujo(ctx, orden.InstanciaID)
	if err != nil {
		return domain.InstanciaFlujo{}, err
	}
	definicion, err := s.definiciones.ObtenerDefinicionFlujoPorReferencia(ctx, instancia.DefinicionRef)
	if err != nil {
		return domain.InstanciaFlujo{}, err
	}
	transicion, err := definicion.ObtenerTransicion(orden.TransicionClave, instancia.EstadoActual)
	if err != nil {
		return domain.InstanciaFlujo{}, err
	}
	if !orden.Principal.AuthAssurance.Cumple(transicion.GarantiaMinima) {
		return domain.InstanciaFlujo{}, domain.ErrGarantiaInsuficiente
	}
	decisionAutorizacion, err := exigirDecisionAutorizacion(
		ctx,
		s.autorizador,
		s.reloj,
		orden.Principal,
		orden.PerfilActivo,
		transicion.Accion,
		domain.RecursoAutorizable{
			Referencia: instancia.ID,
			ModuloID:   definicion.ModuloID,
			Tipo:       "instancia_flujo",
			Atributos: map[string]string{
				"definicion_ref": definicion.Referencia(),
				"entidad_ref":    instancia.EntidadRef,
				"estado":         instancia.EstadoActual,
				"revision":       strconv.Itoa(instancia.Revision),
				"transicion":     transicion.Clave,
			},
		},
		orden.Finalidad,
		orden.CorrelacionRef,
		orden.Motivo,
		usoCamposDecisionNoAplicables,
	)
	if err != nil {
		return domain.InstanciaFlujo{}, err
	}
	decisionRegla, err := s.evaluador.EvaluarReglaFlujo(ctx, ports.SolicitudEvaluarReglaFlujo{
		Definicion:     definicion,
		Instancia:      instancia,
		Transicion:     transicion,
		ActorID:        orden.Principal.ID,
		Finalidad:      orden.Finalidad,
		Motivo:         orden.Motivo,
		CorrelacionRef: orden.CorrelacionRef,
	})
	if err != nil {
		return domain.InstanciaFlujo{}, fmt.Errorf("evaluar regla de flujo: %w", err)
	}
	instante := s.reloj.Ahora().UTC()
	if err := validarDecisionReglaEmitida(definicion, instancia, transicion, decisionRegla, orden, instante); err != nil {
		return domain.InstanciaFlujo{}, err
	}
	trazaRegla, eventoRegla, err := evidenciaDecisionReglaFlujo(
		definicion,
		decisionRegla,
		orden.Principal,
		orden.PerfilActivo,
		decisionAutorizacion.DecisionRef,
		orden.Motivo,
	)
	if err != nil {
		return domain.InstanciaFlujo{}, err
	}
	if err := s.decisiones.RegistrarDecisionReglaFlujo(ctx, decisionRegla, trazaRegla, eventoRegla); err != nil {
		return domain.InstanciaFlujo{}, fmt.Errorf("registrar decision de regla: %w", err)
	}
	if !decisionRegla.Concedida {
		return domain.InstanciaFlujo{}, domain.ErrReglaFlujoDenegada
	}

	aprobacion, err := s.verificarAprobacion(ctx, orden, definicion, instancia, transicion, decisionRegla, instante)
	if err != nil {
		return domain.InstanciaFlujo{}, err
	}
	instante = s.reloj.Ahora().UTC()
	if instante.IsZero() || !decisionAutorizacion.VigenteEn(instante) || !decisionRegla.VigenteEn(instante) ||
		(aprobacion != nil && !aprobacion.VigenteEn(instante)) {
		return domain.InstanciaFlujo{}, domain.ErrDecisionReglaInvalida
	}
	huellaAnterior, err := instancia.HuellaSHA256()
	if err != nil {
		return domain.InstanciaFlujo{}, err
	}
	aprobacionRef := ""
	if aprobacion != nil {
		aprobacionRef = aprobacion.AprobacionRef
	}
	actualizada, cambio, err := instancia.AplicarTransicion(
		definicion,
		transicion.Clave,
		decisionRegla,
		decisionAutorizacion.DecisionRef,
		aprobacionRef,
		orden.Principal.ID,
		orden.Finalidad,
		orden.Motivo,
		orden.CorrelacionRef,
		instante,
	)
	if err != nil {
		return domain.InstanciaFlujo{}, err
	}
	traza, evento, err := evidenciaTransicionInstanciaFlujo(
		definicion,
		actualizada,
		cambio,
		decisionRegla,
		aprobacion,
		orden.Principal,
		orden.PerfilActivo,
		orden.Finalidad,
	)
	if err != nil {
		return domain.InstanciaFlujo{}, err
	}
	if err := s.repositorio.ConfirmarTransicionInstanciaFlujo(
		ctx, huellaAnterior, actualizada, cambio, decisionRegla, aprobacion, traza, evento,
	); err != nil {
		return domain.InstanciaFlujo{}, fmt.Errorf("confirmar transicion de flujo: %w", err)
	}
	return actualizada, nil
}

func (s *ServicioEjecucionFlujos) verificarAprobacion(
	ctx context.Context,
	orden OrdenAplicarTransicionFlujo,
	definicion domain.DefinicionFlujo,
	instancia domain.InstanciaFlujo,
	transicion domain.TransicionFlujoConfigurable,
	decision domain.DecisionReglaFlujo,
	instante time.Time,
) (*domain.EvidenciaAprobacionFlujo, error) {
	if transicion.RequiereAprobacion && orden.AprobacionRef == "" {
		return nil, domain.ErrAprobacionFlujoRequerida
	}
	if orden.AprobacionRef == "" {
		return nil, nil
	}
	evidencia, err := s.aprobaciones.VerificarAprobacionFlujo(ctx, ports.SolicitudVerificarAprobacionFlujo{
		ReferenciaAprobacion: orden.AprobacionRef,
		SolicitanteID:        orden.Principal.ID,
		Definicion:           definicion,
		Instancia:            instancia,
		Transicion:           transicion,
		DecisionRegla:        decision,
		Finalidad:            orden.Finalidad,
		CorrelacionRef:       orden.CorrelacionRef,
	})
	if err != nil {
		return nil, errors.Join(ports.ErrAprobacionFlujoNoVerificada, err)
	}
	huellaDefinicion, err := definicion.HuellaContenidoSHA256()
	if err != nil || evidencia.Validar() != nil || !evidencia.VigenteEn(instante) ||
		evidencia.AprobacionRef != orden.AprobacionRef || evidencia.SolicitanteID != orden.Principal.ID ||
		evidencia.DefinicionRef != definicion.Referencia() ||
		evidencia.DefinicionContenidoHuellaSHA256 != huellaDefinicion || evidencia.InstanciaRef != instancia.ID ||
		evidencia.InstanciaRevision != instancia.Revision || evidencia.EstadoOrigen != instancia.EstadoActual ||
		evidencia.TransicionClave != transicion.Clave || evidencia.DecisionReglaRef != decision.DecisionRef ||
		evidencia.AprobadaEn.Before(decision.EvaluadaEn) || !evidencia.Garantia.Cumple(transicion.GarantiaMinima) {
		return nil, errors.Join(ports.ErrAprobacionFlujoNoVerificada, domain.ErrAprobacionFlujoInvalida)
	}
	return &evidencia, nil
}

func validarDecisionReglaEmitida(
	definicion domain.DefinicionFlujo,
	instancia domain.InstanciaFlujo,
	transicion domain.TransicionFlujoConfigurable,
	decision domain.DecisionReglaFlujo,
	orden OrdenAplicarTransicionFlujo,
	instante time.Time,
) error {
	huellaDefinicion, err := definicion.HuellaContenidoSHA256()
	ultimaActualizacion := instancia.CreadaEn
	if !instancia.ActualizadaEn.IsZero() {
		ultimaActualizacion = instancia.ActualizadaEn
	}
	if err != nil || decision.Validar() != nil || instante.IsZero() || !decision.VigenteEn(instante) ||
		decision.EvaluadaEn.Before(ultimaActualizacion) || decision.DefinicionRef != definicion.Referencia() ||
		decision.DefinicionContenidoHuellaSHA256 != huellaDefinicion || decision.InstanciaRef != instancia.ID ||
		decision.InstanciaRevision != instancia.Revision || decision.EstadoOrigen != instancia.EstadoActual ||
		decision.TransicionClave != transicion.Clave || decision.ReglaRef != transicion.ReglaRef ||
		decision.ActorID != orden.Principal.ID || decision.Finalidad != strings.TrimSpace(orden.Finalidad) ||
		decision.CorrelacionRef != strings.TrimSpace(orden.CorrelacionRef) {
		return domain.ErrDecisionReglaInvalida
	}
	return nil
}

func validarContextoEjecucionFlujo(
	ctx context.Context,
	principal domain.Principal,
	perfilActivo, finalidad, motivo, correlacionRef string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := principal.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(perfilActivo) == "" || strings.TrimSpace(finalidad) == "" ||
		strings.TrimSpace(motivo) == "" || strings.TrimSpace(correlacionRef) == "" {
		return ErrOrdenEjecucionFlujoInvalida
	}
	return nil
}

func evidenciaInicioInstanciaFlujo(
	definicion domain.DefinicionFlujo,
	instancia domain.InstanciaFlujo,
	principal domain.Principal,
	perfilActivo, autorizacionRef, finalidad, motivo, correlacionRef string,
) (domain.AuditEntry, domain.Event, error) {
	huella, err := instancia.HuellaSHA256()
	if err != nil {
		return domain.AuditEntry{}, domain.Event{}, err
	}
	traza := domain.AuditEntry{
		ActorID:          instancia.CreadaPor,
		ActorProfile:     strings.TrimSpace(perfilActivo),
		ActorRoles:       append([]string(nil), principal.Roles...),
		AuthMethod:       principal.AuthMethod,
		AuthAssurance:    principal.AuthAssurance,
		AuthorizationRef: strings.TrimSpace(autorizacionRef),
		Purpose:          strings.TrimSpace(finalidad),
		Action:           domain.AccionInstanciaFlujoIniciada,
		ModuleID:         definicion.ModuloID,
		SubjectRef:       instancia.ID,
		ObjectVersion:    instancia.Revision,
		RuleRef:          definicion.Referencia(),
		Reason:           strings.TrimSpace(motivo),
		Result:           "correcto",
		AfterHash:        huella,
		CorrelationRef:   strings.TrimSpace(correlacionRef),
		OccurredAt:       instancia.CreadaEn,
		Metadata: map[string]string{
			"definicion_ref": definicion.Referencia(),
			"entidad_ref":    instancia.EntidadRef,
			"estado":         instancia.EstadoActual,
		},
	}
	evento := domain.Event{
		Type:       domain.AccionInstanciaFlujoIniciada,
		ModuleID:   definicion.ModuloID,
		SubjectRef: instancia.ID,
		ActorID:    instancia.CreadaPor,
		OccurredAt: instancia.CreadaEn,
		Payload: map[string]string{
			"instancia_ref":  instancia.ID,
			"definicion_ref": definicion.Referencia(),
			"entidad_ref":    instancia.EntidadRef,
			"estado":         instancia.EstadoActual,
			"revision":       strconv.Itoa(instancia.Revision),
			"huella_sha256":  huella,
		},
	}
	return traza, evento, nil
}

func evidenciaDecisionReglaFlujo(
	definicion domain.DefinicionFlujo,
	decision domain.DecisionReglaFlujo,
	principal domain.Principal,
	perfilActivo, autorizacionRef, motivo string,
) (domain.AuditEntry, domain.Event, error) {
	huella, err := decision.HuellaSHA256()
	if err != nil {
		return domain.AuditEntry{}, domain.Event{}, err
	}
	resultado := "denegada"
	if decision.Concedida {
		resultado = "concedida"
	}
	traza := domain.AuditEntry{
		ActorID:          decision.ActorID,
		ActorProfile:     strings.TrimSpace(perfilActivo),
		ActorRoles:       append([]string(nil), principal.Roles...),
		AuthMethod:       principal.AuthMethod,
		AuthAssurance:    principal.AuthAssurance,
		AuthorizationRef: strings.TrimSpace(autorizacionRef),
		Purpose:          decision.Finalidad,
		Action:           domain.AccionDecisionReglaFlujoRegistrada,
		ModuleID:         definicion.ModuloID,
		SubjectRef:       decision.InstanciaRef,
		ObjectVersion:    decision.InstanciaRevision,
		RuleRef:          decision.ReglaRef,
		Reason:           strings.TrimSpace(motivo),
		Result:           resultado,
		AfterHash:        huella,
		CorrelationRef:   decision.CorrelacionRef,
		OccurredAt:       decision.EvaluadaEn,
		Metadata: map[string]string{
			"decision_ref":  decision.DecisionRef,
			"transicion":    decision.TransicionClave,
			"estado_origen": decision.EstadoOrigen,
			"codigo":        decision.Codigo,
		},
	}
	evento := domain.Event{
		Type:       domain.AccionDecisionReglaFlujoRegistrada,
		ModuleID:   definicion.ModuloID,
		SubjectRef: decision.InstanciaRef,
		ActorID:    decision.ActorID,
		OccurredAt: decision.EvaluadaEn,
		Payload: map[string]string{
			"decision_ref":  decision.DecisionRef,
			"regla_ref":     decision.ReglaRef,
			"transicion":    decision.TransicionClave,
			"resultado":     resultado,
			"huella_sha256": huella,
		},
	}
	return traza, evento, nil
}

func evidenciaTransicionInstanciaFlujo(
	definicion domain.DefinicionFlujo,
	instancia domain.InstanciaFlujo,
	cambio domain.CambioEstadoFlujo,
	decision domain.DecisionReglaFlujo,
	aprobacion *domain.EvidenciaAprobacionFlujo,
	principal domain.Principal,
	perfilActivo, finalidad string,
) (domain.AuditEntry, domain.Event, error) {
	huellaAprobacion := ""
	if aprobacion != nil {
		var err error
		huellaAprobacion, err = aprobacion.HuellaSHA256()
		if err != nil {
			return domain.AuditEntry{}, domain.Event{}, err
		}
	}
	traza := domain.AuditEntry{
		ActorID:          instancia.ActualizadaPor,
		ActorProfile:     strings.TrimSpace(perfilActivo),
		ActorRoles:       append([]string(nil), principal.Roles...),
		AuthMethod:       principal.AuthMethod,
		AuthAssurance:    principal.AuthAssurance,
		AuthorizationRef: instancia.UltimaAutorizacionRef,
		Purpose:          strings.TrimSpace(finalidad),
		Action:           domain.AccionInstanciaFlujoTransicionada,
		ModuleID:         definicion.ModuloID,
		SubjectRef:       instancia.ID,
		ObjectVersion:    instancia.Revision,
		RuleRef:          decision.ReglaRef,
		Reason:           instancia.UltimoMotivo,
		Result:           "correcto",
		BeforeHash:       cambio.HuellaAnterior,
		AfterHash:        cambio.HuellaPosterior,
		CorrelationRef:   instancia.UltimaCorrelacionRef,
		OccurredAt:       instancia.ActualizadaEn,
		Metadata: map[string]string{
			"decision_ref":             decision.DecisionRef,
			"transicion":               cambio.TransicionClave,
			"estado_anterior":          cambio.EstadoAnterior,
			"estado_posterior":         cambio.EstadoPosterior,
			"aprobacion_ref":           instancia.UltimaAprobacionRef,
			"aprobacion_huella_sha256": huellaAprobacion,
		},
	}
	evento := domain.Event{
		Type:       domain.AccionInstanciaFlujoTransicionada,
		ModuleID:   definicion.ModuloID,
		SubjectRef: instancia.ID,
		ActorID:    instancia.ActualizadaPor,
		OccurredAt: instancia.ActualizadaEn,
		Payload: map[string]string{
			"instancia_ref":    instancia.ID,
			"transicion":       cambio.TransicionClave,
			"estado_anterior":  cambio.EstadoAnterior,
			"estado_posterior": cambio.EstadoPosterior,
			"revision":         strconv.Itoa(instancia.Revision),
			"huella_sha256":    cambio.HuellaPosterior,
		},
	}
	return traza, evento, nil
}

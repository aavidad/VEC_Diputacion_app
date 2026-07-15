package memory

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func (s *Store) ConfirmarAltaBorradorFlujo(
	ctx context.Context,
	definicion domain.DefinicionFlujo,
	traza domain.AuditEntry,
	evento domain.Event,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	canonica, err := definicion.ClonarCanonico()
	if err != nil {
		return err
	}
	if canonica.Estado != domain.EstadoDefinicionFlujoBorrador || canonica.Revision != 1 ||
		!evidenciaDefinicionFlujoValida(canonica, traza, evento, domain.AccionDefinicionFlujoBorradorCreada, "") {
		return domain.ErrDefinicionFlujoInvalida
	}
	clave := claveDefinicionFlujo(canonica.ID, canonica.Version)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, existe := s.definicionesFlujo[clave]; existe {
		return ports.ErrVersionDefinicionFlujoYaExiste
	}
	if canonica.Version > 1 {
		anterior, existe := s.definicionesFlujo[claveDefinicionFlujo(canonica.ID, canonica.Version-1)]
		if !existe ||
			(anterior.Estado != domain.EstadoDefinicionFlujoPublicada && anterior.Estado != domain.EstadoDefinicionFlujoRetirada) ||
			canonica.VersionAnteriorRef != anterior.Referencia() {
			return ports.ErrSecuenciaDefinicionFlujoInvalida
		}
	}
	s.definicionesFlujo[clave] = canonica
	s.confirmarEvidenciaFlujoBloqueado(traza, evento)
	return nil
}

func (s *Store) ConfirmarActualizacionBorradorFlujo(
	ctx context.Context,
	huellaAnterior string,
	definicion domain.DefinicionFlujo,
	traza domain.AuditEntry,
	evento domain.Event,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	canonica, err := definicion.ClonarCanonico()
	if err != nil {
		return err
	}
	if canonica.Estado != domain.EstadoDefinicionFlujoBorrador || canonica.Revision < 2 ||
		strings.TrimSpace(huellaAnterior) == "" ||
		!evidenciaDefinicionFlujoValida(canonica, traza, evento, domain.AccionDefinicionFlujoBorradorActualizada, huellaAnterior) {
		return domain.ErrDefinicionFlujoInvalida
	}
	clave := claveDefinicionFlujo(canonica.ID, canonica.Version)
	s.mu.Lock()
	defer s.mu.Unlock()
	actual, existe := s.definicionesFlujo[clave]
	if !existe {
		return ports.ErrDefinicionFlujoNoEncontrada
	}
	huellaActual, err := actual.HuellaSHA256()
	if err != nil || huellaActual != huellaAnterior || actual.Estado != domain.EstadoDefinicionFlujoBorrador ||
		canonica.Revision != actual.Revision+1 || !identidadVersionFlujoIgual(actual, canonica) {
		return ports.ErrRevisionDefinicionFlujoConflicto
	}
	s.definicionesFlujo[clave] = canonica
	s.confirmarEvidenciaFlujoBloqueado(traza, evento)
	return nil
}

func (s *Store) ConfirmarPublicacionFlujo(
	ctx context.Context,
	huellaBorrador string,
	definicion domain.DefinicionFlujo,
	traza domain.AuditEntry,
	evento domain.Event,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	canonica, err := definicion.ClonarCanonico()
	if err != nil {
		return err
	}
	if canonica.Estado != domain.EstadoDefinicionFlujoPublicada || strings.TrimSpace(huellaBorrador) == "" ||
		!evidenciaDefinicionFlujoValida(canonica, traza, evento, domain.AccionDefinicionFlujoPublicada, huellaBorrador) {
		return domain.ErrDefinicionFlujoInvalida
	}
	base := canonica
	base.Estado = domain.EstadoDefinicionFlujoBorrador
	base.PublicadaPor = ""
	base.PublicadaEn = time.Time{}
	base.AprobacionRef = ""
	base.MotivoPublicacion = ""
	huellaBase, err := base.HuellaSHA256()
	if err != nil || huellaBase != huellaBorrador {
		return ports.ErrRevisionDefinicionFlujoConflicto
	}

	clave := claveDefinicionFlujo(canonica.ID, canonica.Version)
	s.mu.Lock()
	defer s.mu.Unlock()
	actual, existe := s.definicionesFlujo[clave]
	if !existe {
		return ports.ErrDefinicionFlujoNoEncontrada
	}
	huellaActual, err := actual.HuellaSHA256()
	if err != nil || actual.Estado != domain.EstadoDefinicionFlujoBorrador || huellaActual != huellaBorrador {
		return ports.ErrRevisionDefinicionFlujoConflicto
	}
	if !s.referenciasCatalogoDefinicionValidasBloqueado(canonica) {
		return domain.ErrDefinicionFlujoInvalida
	}
	s.definicionesFlujo[clave] = canonica
	s.confirmarEvidenciaFlujoBloqueado(traza, evento)
	return nil
}

func (s *Store) ConfirmarRetiradaFlujo(
	ctx context.Context,
	huellaPublicada string,
	definicion domain.DefinicionFlujo,
	traza domain.AuditEntry,
	evento domain.Event,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	canonica, err := definicion.ClonarCanonico()
	if err != nil {
		return err
	}
	if canonica.Estado != domain.EstadoDefinicionFlujoRetirada || strings.TrimSpace(huellaPublicada) == "" ||
		!evidenciaDefinicionFlujoValida(canonica, traza, evento, domain.AccionDefinicionFlujoRetirada, huellaPublicada) {
		return domain.ErrDefinicionFlujoInvalida
	}
	base := canonica
	base.Estado = domain.EstadoDefinicionFlujoPublicada
	base.RetiradaPor = ""
	base.RetiradaEn = time.Time{}
	base.RetiradaAprobacionRef = ""
	base.MotivoRetirada = ""
	huellaBase, err := base.HuellaSHA256()
	if err != nil || huellaBase != huellaPublicada {
		return ports.ErrRevisionDefinicionFlujoConflicto
	}

	clave := claveDefinicionFlujo(canonica.ID, canonica.Version)
	s.mu.Lock()
	defer s.mu.Unlock()
	actual, existe := s.definicionesFlujo[clave]
	if !existe {
		return ports.ErrDefinicionFlujoNoEncontrada
	}
	huellaActual, err := actual.HuellaSHA256()
	if err != nil || actual.Estado != domain.EstadoDefinicionFlujoPublicada || huellaActual != huellaPublicada {
		return ports.ErrRevisionDefinicionFlujoConflicto
	}
	s.definicionesFlujo[clave] = canonica
	s.confirmarEvidenciaFlujoBloqueado(traza, evento)
	return nil
}

func (s *Store) ObtenerDefinicionFlujo(ctx context.Context, id string, version int) (domain.DefinicionFlujo, error) {
	if err := ctx.Err(); err != nil {
		return domain.DefinicionFlujo{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	definicion, existe := s.definicionesFlujo[claveDefinicionFlujo(id, version)]
	if !existe {
		return domain.DefinicionFlujo{}, ports.ErrDefinicionFlujoNoEncontrada
	}
	return definicion.ClonarCanonico()
}

func (s *Store) ObtenerDefinicionFlujoPorReferencia(ctx context.Context, referencia string) (domain.DefinicionFlujo, error) {
	if err := ctx.Err(); err != nil {
		return domain.DefinicionFlujo{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	definicion, existe := s.definicionesFlujo[strings.TrimSpace(referencia)]
	if !existe {
		return domain.DefinicionFlujo{}, ports.ErrDefinicionFlujoNoEncontrada
	}
	return definicion.ClonarCanonico()
}

func (s *Store) ListarVersionesDefinicionFlujo(ctx context.Context, id string) ([]domain.DefinicionFlujo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ports.ErrDefinicionFlujoNoEncontrada
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	resultado := make([]domain.DefinicionFlujo, 0)
	for _, definicion := range s.definicionesFlujo {
		if definicion.ID != id {
			continue
		}
		clon, err := definicion.ClonarCanonico()
		if err != nil {
			return nil, err
		}
		resultado = append(resultado, clon)
	}
	if len(resultado) == 0 {
		return nil, ports.ErrDefinicionFlujoNoEncontrada
	}
	sort.Slice(resultado, func(i, j int) bool { return resultado[i].Version < resultado[j].Version })
	return resultado, nil
}

func (s *Store) NuevoIDInstanciaFlujo() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.secuenciaInstancias++
	return fmt.Sprintf("instancia-flujo-%06d", s.secuenciaInstancias), nil
}

func (s *Store) ConfirmarInicioInstanciaFlujo(
	ctx context.Context,
	instancia domain.InstanciaFlujo,
	traza domain.AuditEntry,
	evento domain.Event,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := instancia.Validar(); err != nil {
		return err
	}
	if instancia.Revision != 1 {
		return domain.ErrInstanciaFlujoInvalida
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	definicion, existe := s.definicionesFlujo[instancia.DefinicionRef]
	if !existe {
		return ports.ErrDefinicionFlujoNoEncontrada
	}
	huellaContenido, err := definicion.HuellaContenidoSHA256()
	if err != nil || definicion.Estado != domain.EstadoDefinicionFlujoPublicada ||
		instancia.TipoEntidad != definicion.TipoEntidad || instancia.EstadoActual != definicion.EstadoInicial ||
		instancia.DefinicionContenidoHuellaSHA256 != huellaContenido ||
		!evidenciaInicioInstanciaFlujoValida(definicion, instancia, traza, evento) {
		return domain.ErrInstanciaFlujoInvalida
	}
	if _, existe := s.instanciasFlujo[instancia.ID]; existe {
		return ports.ErrInstanciaFlujoYaExiste
	}
	claveEntidad := claveInstanciaFlujoPorEntidad(instancia.DefinicionRef, instancia.EntidadRef)
	if _, existe := s.instanciasPorEntidad[claveEntidad]; existe {
		return ports.ErrEntidadConInstanciaFlujo
	}
	s.instanciasFlujo[instancia.ID] = instancia
	s.instanciasPorEntidad[claveEntidad] = instancia.ID
	s.confirmarEvidenciaFlujoBloqueado(traza, evento)
	return nil
}

func (s *Store) ObtenerInstanciaFlujo(ctx context.Context, id string) (domain.InstanciaFlujo, error) {
	if err := ctx.Err(); err != nil {
		return domain.InstanciaFlujo{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	instancia, existe := s.instanciasFlujo[strings.TrimSpace(id)]
	if !existe {
		return domain.InstanciaFlujo{}, ports.ErrInstanciaFlujoNoEncontrada
	}
	return instancia, nil
}

func (s *Store) RegistrarDecisionReglaFlujo(
	ctx context.Context,
	decision domain.DecisionReglaFlujo,
	traza domain.AuditEntry,
	evento domain.Event,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := decision.Validar(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, existe := s.decisionesReglaFlujo[decision.DecisionRef]; existe {
		return ports.ErrDecisionReglaFlujoYaExiste
	}
	definicion, existe := s.definicionesFlujo[decision.DefinicionRef]
	if !existe {
		return ports.ErrDefinicionFlujoNoEncontrada
	}
	instancia, existe := s.instanciasFlujo[decision.InstanciaRef]
	if !existe {
		return ports.ErrInstanciaFlujoNoEncontrada
	}
	huellaContenido, err := definicion.HuellaContenidoSHA256()
	transicion, errTransicion := transicionConfiguradaFlujo(definicion, decision.TransicionClave, decision.EstadoOrigen)
	instanteRevision, revisionValida := s.instanteRevisionInstanciaFlujoBloqueado(definicion, instancia, decision)
	if err != nil || errTransicion != nil || decision.DefinicionContenidoHuellaSHA256 != huellaContenido ||
		!revisionValida || decision.EvaluadaEn.Before(instanteRevision) ||
		decision.DefinicionRef != instancia.DefinicionRef || transicion.ReglaRef != decision.ReglaRef ||
		!evidenciaDecisionReglaFlujoValida(definicion, decision, traza, evento) {
		return domain.ErrDecisionReglaInvalida
	}
	s.decisionesReglaFlujo[decision.DecisionRef] = decision
	s.confirmarEvidenciaFlujoBloqueado(traza, evento)
	return nil
}

func transicionConfiguradaFlujo(
	definicion domain.DefinicionFlujo,
	clave, estadoOrigen string,
) (domain.TransicionFlujoConfigurable, error) {
	for _, transicion := range definicion.Transiciones {
		if transicion.Clave == clave && transicion.AdmiteOrigen(estadoOrigen) {
			return transicion, nil
		}
	}
	return domain.TransicionFlujoConfigurable{}, domain.ErrTransicionFlujoInvalida
}

func (s *Store) instanteRevisionInstanciaFlujoBloqueado(
	definicion domain.DefinicionFlujo,
	instancia domain.InstanciaFlujo,
	decision domain.DecisionReglaFlujo,
) (time.Time, bool) {
	if decision.InstanciaRevision > instancia.Revision {
		return time.Time{}, false
	}
	if decision.InstanciaRevision == instancia.Revision {
		if decision.EstadoOrigen != instancia.EstadoActual {
			return time.Time{}, false
		}
		if instancia.Revision == 1 {
			return instancia.CreadaEn, true
		}
		return instancia.ActualizadaEn, true
	}
	if decision.InstanciaRevision == 1 {
		return instancia.CreadaEn, decision.EstadoOrigen == definicion.EstadoInicial
	}
	for _, entrada := range s.audit {
		if entrada.SubjectRef == instancia.ID && entrada.Action == domain.AccionInstanciaFlujoTransicionada &&
			entrada.ObjectVersion == decision.InstanciaRevision && entrada.Metadata["estado_posterior"] == decision.EstadoOrigen {
			return entrada.OccurredAt, true
		}
	}
	return time.Time{}, false
}

func (s *Store) ObtenerDecisionReglaFlujo(ctx context.Context, referencia string) (domain.DecisionReglaFlujo, error) {
	if err := ctx.Err(); err != nil {
		return domain.DecisionReglaFlujo{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	decision, existe := s.decisionesReglaFlujo[strings.TrimSpace(referencia)]
	if !existe {
		return domain.DecisionReglaFlujo{}, ports.ErrDecisionReglaFlujoNoEncontrada
	}
	return decision, nil
}

func (s *Store) ConfirmarTransicionInstanciaFlujo(
	ctx context.Context,
	huellaAnterior string,
	actualizada domain.InstanciaFlujo,
	cambio domain.CambioEstadoFlujo,
	decision domain.DecisionReglaFlujo,
	aprobacion *domain.EvidenciaAprobacionFlujo,
	traza domain.AuditEntry,
	evento domain.Event,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := actualizada.Validar(); err != nil {
		return err
	}
	if err := decision.Validar(); err != nil || !decision.Concedida || strings.TrimSpace(huellaAnterior) == "" {
		return domain.ErrDecisionReglaInvalida
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	anterior, existe := s.instanciasFlujo[actualizada.ID]
	if !existe {
		return ports.ErrInstanciaFlujoNoEncontrada
	}
	huellaActual, err := anterior.HuellaSHA256()
	if err != nil || huellaActual != huellaAnterior || actualizada.Revision != anterior.Revision+1 ||
		!identidadInstanciaFlujoIgual(anterior, actualizada) {
		return ports.ErrRevisionInstanciaFlujoConflicto
	}
	definicion, existe := s.definicionesFlujo[actualizada.DefinicionRef]
	if !existe {
		return ports.ErrDefinicionFlujoNoEncontrada
	}
	definicionUtilizable := definicion.Estado == domain.EstadoDefinicionFlujoPublicada ||
		(definicion.Estado == domain.EstadoDefinicionFlujoRetirada && definicion.PermiteFinalizacionTrasRetirada)
	huellaDefinicion, errDefinicion := definicion.HuellaContenidoSHA256()
	transicion, errTransicion := definicion.ObtenerTransicion(decision.TransicionClave, anterior.EstadoActual)
	if !definicionUtilizable {
		return domain.ErrDefinicionFlujoNoPublicada
	}
	registrada, existe := s.decisionesReglaFlujo[decision.DecisionRef]
	if !existe {
		return ports.ErrDecisionReglaFlujoNoEncontrada
	}
	huellaRegistrada, errRegistrada := registrada.HuellaSHA256()
	huellaDecision, errDecision := decision.HuellaSHA256()
	if errDefinicion != nil || errRegistrada != nil || errDecision != nil || errTransicion != nil ||
		huellaRegistrada != huellaDecision ||
		anterior.DefinicionContenidoHuellaSHA256 != huellaDefinicion || transicion.Hacia != actualizada.EstadoActual ||
		transicion.ReglaRef != decision.ReglaRef ||
		decision.InstanciaRevision != anterior.Revision || decision.EstadoOrigen != anterior.EstadoActual ||
		decision.ActorID != actualizada.ActualizadaPor || decision.CorrelacionRef != actualizada.UltimaCorrelacionRef ||
		!decision.VigenteEn(actualizada.ActualizadaEn) ||
		!aprobacionTransicionFlujoValida(actualizada, transicion, decision, aprobacion) ||
		!cambioEstadoFlujoValido(anterior, actualizada, cambio, decision) ||
		!evidenciaTransicionInstanciaFlujoValida(definicion, actualizada, cambio, decision, aprobacion, traza, evento) {
		return domain.ErrInstanciaFlujoInvalida
	}
	s.instanciasFlujo[actualizada.ID] = actualizada
	s.confirmarEvidenciaFlujoBloqueado(traza, evento)
	return nil
}

func (s *Store) confirmarEvidenciaFlujoBloqueado(traza domain.AuditEntry, evento domain.Event) {
	trazaConfirmada := s.appendAuditLocked(traza)
	evento.Payload = cloneStringMap(evento.Payload)
	if evento.Payload == nil {
		evento.Payload = map[string]string{}
	}
	evento.Payload["auditoria_ref"] = trazaConfirmada.ID
	s.appendEventLocked(evento)
}

func evidenciaDefinicionFlujoValida(
	definicion domain.DefinicionFlujo,
	traza domain.AuditEntry,
	evento domain.Event,
	accion, huellaAnterior string,
) bool {
	huella, err := definicion.HuellaSHA256()
	huellaContenido, errContenido := definicion.HuellaContenidoSHA256()
	if err != nil || errContenido != nil {
		return false
	}
	actor, fecha, regla, motivo := definicion.CreadaPor, definicion.CreadaEn, definicion.FuenteRef, definicion.MotivoCreacion
	switch accion {
	case domain.AccionDefinicionFlujoBorradorActualizada:
		actor, fecha, motivo = definicion.UltimaModificacionPor, definicion.UltimaModificacionEn, definicion.MotivoModificacion
	case domain.AccionDefinicionFlujoPublicada:
		actor, fecha, regla, motivo = definicion.PublicadaPor, definicion.PublicadaEn, definicion.AprobacionRef, definicion.MotivoPublicacion
	case domain.AccionDefinicionFlujoRetirada:
		actor, fecha, regla, motivo = definicion.RetiradaPor, definicion.RetiradaEn, definicion.RetiradaAprobacionRef, definicion.MotivoRetirada
	case domain.AccionDefinicionFlujoBorradorCreada:
	default:
		return false
	}
	return traza.ActorID == actor && strings.TrimSpace(traza.ActorProfile) != "" &&
		traza.AuthMethod.Valido() && traza.AuthAssurance.Valida() && strings.TrimSpace(traza.AuthorizationRef) != "" &&
		strings.TrimSpace(traza.Purpose) != "" && traza.Action == accion && traza.ModuleID == definicion.ModuloID &&
		traza.SubjectRef == definicion.Referencia() && traza.ObjectVersion == definicion.Version && traza.RuleRef == regla &&
		traza.Result == "correcto" && traza.BeforeHash == huellaAnterior && traza.AfterHash == huella &&
		traza.Reason == motivo && traza.Metadata["revision"] == strconv.Itoa(definicion.Revision) &&
		traza.Metadata["huella_contenido_sha256"] == huellaContenido && strings.TrimSpace(traza.CorrelationRef) != "" &&
		traza.OccurredAt.Equal(fecha) && evento.Type == accion && evento.ModuleID == definicion.ModuloID &&
		evento.SubjectRef == definicion.Referencia() && evento.ActorID == actor && evento.OccurredAt.Equal(fecha) &&
		evento.Payload["definicion_id"] == definicion.ID &&
		evento.Payload["definicion_version"] == strconv.Itoa(definicion.Version) &&
		evento.Payload["definicion_revision"] == strconv.Itoa(definicion.Revision) &&
		evento.Payload["estado"] == string(definicion.Estado) && evento.Payload["huella_sha256"] == huella &&
		evento.Payload["huella_contenido_sha256"] == huellaContenido
}

func evidenciaInicioInstanciaFlujoValida(
	definicion domain.DefinicionFlujo,
	instancia domain.InstanciaFlujo,
	traza domain.AuditEntry,
	evento domain.Event,
) bool {
	huella, err := instancia.HuellaSHA256()
	if err != nil {
		return false
	}
	return traza.ActorID == instancia.CreadaPor && strings.TrimSpace(traza.ActorProfile) != "" &&
		traza.AuthMethod.Valido() && traza.AuthAssurance.Valida() && strings.TrimSpace(traza.AuthorizationRef) != "" &&
		strings.TrimSpace(traza.Purpose) != "" && traza.Action == domain.AccionInstanciaFlujoIniciada &&
		traza.ModuleID == definicion.ModuloID && traza.SubjectRef == instancia.ID &&
		traza.ObjectVersion == instancia.Revision && traza.RuleRef == definicion.Referencia() &&
		traza.Result == "correcto" && traza.BeforeHash == "" && traza.AfterHash == huella &&
		strings.TrimSpace(traza.Reason) != "" && traza.CorrelationRef != "" && traza.OccurredAt.Equal(instancia.CreadaEn) &&
		traza.Metadata["definicion_ref"] == definicion.Referencia() &&
		traza.Metadata["entidad_ref"] == instancia.EntidadRef && traza.Metadata["estado"] == instancia.EstadoActual &&
		evento.Type == domain.AccionInstanciaFlujoIniciada && evento.ModuleID == definicion.ModuloID &&
		evento.SubjectRef == instancia.ID && evento.ActorID == instancia.CreadaPor && evento.OccurredAt.Equal(instancia.CreadaEn) &&
		evento.Payload["instancia_ref"] == instancia.ID && evento.Payload["definicion_ref"] == definicion.Referencia() &&
		evento.Payload["entidad_ref"] == instancia.EntidadRef && evento.Payload["estado"] == instancia.EstadoActual &&
		evento.Payload["revision"] == strconv.Itoa(instancia.Revision) && evento.Payload["huella_sha256"] == huella
}

func evidenciaDecisionReglaFlujoValida(
	definicion domain.DefinicionFlujo,
	decision domain.DecisionReglaFlujo,
	traza domain.AuditEntry,
	evento domain.Event,
) bool {
	huella, err := decision.HuellaSHA256()
	if err != nil {
		return false
	}
	resultado := "denegada"
	if decision.Concedida {
		resultado = "concedida"
	}
	return traza.ActorID == decision.ActorID && strings.TrimSpace(traza.ActorProfile) != "" &&
		traza.AuthMethod.Valido() && traza.AuthAssurance.Valida() && strings.TrimSpace(traza.AuthorizationRef) != "" &&
		traza.Purpose == decision.Finalidad && traza.Action == domain.AccionDecisionReglaFlujoRegistrada &&
		traza.ModuleID == definicion.ModuloID && traza.SubjectRef == decision.InstanciaRef &&
		traza.ObjectVersion == decision.InstanciaRevision && traza.RuleRef == decision.ReglaRef &&
		traza.Result == resultado && traza.BeforeHash == "" && traza.AfterHash == huella &&
		strings.TrimSpace(traza.Reason) != "" && traza.CorrelationRef == decision.CorrelacionRef &&
		traza.OccurredAt.Equal(decision.EvaluadaEn) && traza.Metadata["decision_ref"] == decision.DecisionRef &&
		traza.Metadata["transicion"] == decision.TransicionClave && traza.Metadata["estado_origen"] == decision.EstadoOrigen &&
		traza.Metadata["codigo"] == decision.Codigo && evento.Type == domain.AccionDecisionReglaFlujoRegistrada &&
		evento.ModuleID == definicion.ModuloID && evento.SubjectRef == decision.InstanciaRef &&
		evento.ActorID == decision.ActorID && evento.OccurredAt.Equal(decision.EvaluadaEn) &&
		evento.Payload["decision_ref"] == decision.DecisionRef && evento.Payload["regla_ref"] == decision.ReglaRef &&
		evento.Payload["transicion"] == decision.TransicionClave && evento.Payload["resultado"] == resultado &&
		evento.Payload["huella_sha256"] == huella
}

func evidenciaTransicionInstanciaFlujoValida(
	definicion domain.DefinicionFlujo,
	instancia domain.InstanciaFlujo,
	cambio domain.CambioEstadoFlujo,
	decision domain.DecisionReglaFlujo,
	aprobacion *domain.EvidenciaAprobacionFlujo,
	traza domain.AuditEntry,
	evento domain.Event,
) bool {
	huellaAprobacion := ""
	if aprobacion != nil {
		var err error
		huellaAprobacion, err = aprobacion.HuellaSHA256()
		if err != nil {
			return false
		}
	}
	return traza.ActorID == instancia.ActualizadaPor && strings.TrimSpace(traza.ActorProfile) != "" &&
		traza.AuthMethod.Valido() && traza.AuthAssurance.Valida() &&
		traza.AuthorizationRef == instancia.UltimaAutorizacionRef && traza.Purpose == decision.Finalidad &&
		traza.Action == domain.AccionInstanciaFlujoTransicionada && traza.ModuleID == definicion.ModuloID &&
		traza.SubjectRef == instancia.ID && traza.ObjectVersion == instancia.Revision && traza.RuleRef == decision.ReglaRef &&
		traza.Result == "correcto" && traza.BeforeHash == cambio.HuellaAnterior && traza.AfterHash == cambio.HuellaPosterior &&
		traza.Reason == instancia.UltimoMotivo && traza.CorrelationRef == instancia.UltimaCorrelacionRef &&
		traza.OccurredAt.Equal(instancia.ActualizadaEn) && traza.Metadata["decision_ref"] == decision.DecisionRef &&
		traza.Metadata["transicion"] == cambio.TransicionClave && traza.Metadata["estado_anterior"] == cambio.EstadoAnterior &&
		traza.Metadata["estado_posterior"] == cambio.EstadoPosterior &&
		traza.Metadata["aprobacion_ref"] == instancia.UltimaAprobacionRef &&
		traza.Metadata["aprobacion_huella_sha256"] == huellaAprobacion &&
		evento.Type == domain.AccionInstanciaFlujoTransicionada && evento.ModuleID == definicion.ModuloID &&
		evento.SubjectRef == instancia.ID && evento.ActorID == instancia.ActualizadaPor &&
		evento.OccurredAt.Equal(instancia.ActualizadaEn) && evento.Payload["instancia_ref"] == instancia.ID &&
		evento.Payload["transicion"] == cambio.TransicionClave && evento.Payload["estado_anterior"] == cambio.EstadoAnterior &&
		evento.Payload["estado_posterior"] == cambio.EstadoPosterior &&
		evento.Payload["revision"] == strconv.Itoa(instancia.Revision) && evento.Payload["huella_sha256"] == cambio.HuellaPosterior
}

func aprobacionTransicionFlujoValida(
	instancia domain.InstanciaFlujo,
	transicion domain.TransicionFlujoConfigurable,
	decision domain.DecisionReglaFlujo,
	aprobacion *domain.EvidenciaAprobacionFlujo,
) bool {
	if instancia.UltimaAprobacionRef == "" {
		return aprobacion == nil && !transicion.RequiereAprobacion
	}
	if aprobacion == nil || aprobacion.Validar() != nil {
		return false
	}
	return aprobacion.AprobacionRef == instancia.UltimaAprobacionRef &&
		aprobacion.VigenteEn(instancia.ActualizadaEn) && !aprobacion.AprobadaEn.Before(decision.EvaluadaEn) &&
		aprobacion.Garantia.Cumple(transicion.GarantiaMinima) &&
		aprobacion.SolicitanteID == instancia.ActualizadaPor && aprobacion.DefinicionRef == instancia.DefinicionRef &&
		aprobacion.DefinicionContenidoHuellaSHA256 == instancia.DefinicionContenidoHuellaSHA256 &&
		aprobacion.InstanciaRef == instancia.ID && aprobacion.InstanciaRevision == decision.InstanciaRevision &&
		aprobacion.EstadoOrigen == decision.EstadoOrigen && aprobacion.TransicionClave == decision.TransicionClave &&
		aprobacion.DecisionReglaRef == decision.DecisionRef
}

func (s *Store) referenciasCatalogoDefinicionValidasBloqueado(definicion domain.DefinicionFlujo) bool {
	cache := make(map[string]domain.CatalogoConfigurable)
	for _, estado := range definicion.Estados {
		clave := claveCatalogo(estado.Catalogo.CatalogoID, estado.Catalogo.CatalogoVersion)
		catalogo, existe := cache[clave]
		if !existe {
			catalogo, existe = s.catalogos[clave]
			if !existe || catalogo.Estado != domain.EstadoCatalogoPublicado {
				return false
			}
			huella, err := catalogo.HuellaContenidoSHA256()
			if err != nil || huella != estado.Catalogo.CatalogoHuellaSHA256 {
				return false
			}
			cache[clave] = catalogo
		}
		encontrada := false
		for _, entrada := range catalogo.Entradas {
			if entrada.Clave == estado.Clave && entrada.Clave == estado.Catalogo.EntradaClave {
				encontrada = true
				break
			}
		}
		if !encontrada {
			return false
		}
	}
	return true
}

func cambioEstadoFlujoValido(
	anterior, posterior domain.InstanciaFlujo,
	cambio domain.CambioEstadoFlujo,
	decision domain.DecisionReglaFlujo,
) bool {
	huellaAnterior, errAnterior := anterior.HuellaSHA256()
	huellaPosterior, errPosterior := posterior.HuellaSHA256()
	return errAnterior == nil && errPosterior == nil && cambio.InstanciaRef == anterior.ID &&
		cambio.RevisionAnterior == anterior.Revision && cambio.RevisionPosterior == posterior.Revision &&
		cambio.EstadoAnterior == anterior.EstadoActual && cambio.EstadoPosterior == posterior.EstadoActual &&
		cambio.TransicionClave == posterior.UltimaTransicionClave && cambio.DecisionReglaRef == decision.DecisionRef &&
		cambio.AutorizacionRef == posterior.UltimaAutorizacionRef && cambio.HuellaAnterior == huellaAnterior &&
		cambio.HuellaPosterior == huellaPosterior
}

func identidadVersionFlujoIgual(anterior, posterior domain.DefinicionFlujo) bool {
	return anterior.ID == posterior.ID && anterior.Version == posterior.Version &&
		anterior.VersionAnteriorRef == posterior.VersionAnteriorRef && anterior.ModuloID == posterior.ModuloID &&
		anterior.TipoEntidad == posterior.TipoEntidad && anterior.CreadaPor == posterior.CreadaPor &&
		anterior.CreadaEn.Equal(posterior.CreadaEn) && anterior.MotivoCreacion == posterior.MotivoCreacion
}

func identidadInstanciaFlujoIgual(anterior, posterior domain.InstanciaFlujo) bool {
	return anterior.ID == posterior.ID && anterior.TipoEntidad == posterior.TipoEntidad &&
		anterior.EntidadRef == posterior.EntidadRef && anterior.DefinicionRef == posterior.DefinicionRef &&
		anterior.DefinicionContenidoHuellaSHA256 == posterior.DefinicionContenidoHuellaSHA256 &&
		anterior.CreadaPor == posterior.CreadaPor && anterior.CreadaEn.Equal(posterior.CreadaEn)
}

func claveDefinicionFlujo(id string, version int) string {
	return strings.TrimSpace(id) + ":" + strconv.Itoa(version)
}

func claveInstanciaFlujoPorEntidad(definicionRef, entidadRef string) string {
	return strings.TrimSpace(definicionRef) + "\x00" + strings.TrimSpace(entidadRef)
}

var _ ports.ConsultaDefinicionesFlujo = (*Store)(nil)
var _ ports.RepositorioGobiernoFlujos = (*Store)(nil)
var _ ports.ConsultaInstanciasFlujo = (*Store)(nil)
var _ ports.RepositorioInstanciasFlujo = (*Store)(nil)
var _ ports.RegistroDecisionesReglaFlujo = (*Store)(nil)
var _ ports.GeneradorIDInstanciaFlujo = (*Store)(nil)

package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const vigenciaMaximaReservaCotejoMemoria = 10 * time.Minute

func (s *Store) ConfirmarAltaBorradorPoliticaCotejo(
	ctx context.Context,
	politica domain.PoliticaCotejo,
	traza domain.AuditEntry,
	evento domain.Event,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	canonica, err := politica.ClonarCanonica()
	if err != nil {
		return err
	}
	if canonica.Estado != domain.EstadoPoliticaCotejoBorrador || canonica.Revision != 1 ||
		!evidenciaPoliticaCotejoValida(canonica, traza, evento, domain.AccionPoliticaCotejoBorradorCreada, "") {
		return domain.ErrPoliticaCotejoInvalida
	}
	clave := clavePoliticaCotejo(canonica.ID, canonica.Version)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, existe := s.politicasCotejo[clave]; existe {
		return ports.ErrVersionPoliticaCotejoYaExiste
	}
	if canonica.Version > 1 {
		anterior, existe := s.politicasCotejo[clavePoliticaCotejo(canonica.ID, canonica.Version-1)]
		if !existe || (anterior.Estado != domain.EstadoPoliticaCotejoPublicada && anterior.Estado != domain.EstadoPoliticaCotejoRetirada) ||
			canonica.VersionAnteriorRef != anterior.Referencia() {
			return ports.ErrSecuenciaPoliticaCotejoInvalida
		}
	}
	s.politicasCotejo[clave] = canonica
	s.confirmarEvidenciaCotejoBloqueado(traza, evento)
	return nil
}

func (s *Store) ConfirmarActualizacionBorradorPoliticaCotejo(
	ctx context.Context,
	huellaAnterior string,
	politica domain.PoliticaCotejo,
	traza domain.AuditEntry,
	evento domain.Event,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	canonica, err := politica.ClonarCanonica()
	if err != nil {
		return err
	}
	if canonica.Estado != domain.EstadoPoliticaCotejoBorrador || canonica.Revision < 2 ||
		strings.TrimSpace(huellaAnterior) == "" ||
		!evidenciaPoliticaCotejoValida(canonica, traza, evento, domain.AccionPoliticaCotejoBorradorActualizada, huellaAnterior) {
		return domain.ErrPoliticaCotejoInvalida
	}
	clave := clavePoliticaCotejo(canonica.ID, canonica.Version)
	s.mu.Lock()
	defer s.mu.Unlock()
	actual, existe := s.politicasCotejo[clave]
	if !existe {
		return ports.ErrPoliticaCotejoNoEncontrada
	}
	huellaActual, err := actual.HuellaSHA256()
	if err != nil || actual.Estado != domain.EstadoPoliticaCotejoBorrador || huellaActual != huellaAnterior ||
		canonica.Revision != actual.Revision+1 || !identidadPoliticaCotejoIgual(actual, canonica) {
		return ports.ErrRevisionPoliticaCotejoConflicto
	}
	s.politicasCotejo[clave] = canonica
	s.confirmarEvidenciaCotejoBloqueado(traza, evento)
	return nil
}

func (s *Store) ConfirmarPublicacionPoliticaCotejo(
	ctx context.Context,
	huellaAnterior string,
	politica domain.PoliticaCotejo,
	traza domain.AuditEntry,
	evento domain.Event,
) error {
	return s.confirmarTransicionPoliticaCotejo(ctx, huellaAnterior, politica, traza, evento,
		domain.EstadoPoliticaCotejoBorrador, domain.EstadoPoliticaCotejoPublicada,
		domain.AccionPoliticaCotejoPublicada)
}

func (s *Store) ConfirmarRetiradaPoliticaCotejo(
	ctx context.Context,
	huellaAnterior string,
	politica domain.PoliticaCotejo,
	traza domain.AuditEntry,
	evento domain.Event,
) error {
	return s.confirmarTransicionPoliticaCotejo(ctx, huellaAnterior, politica, traza, evento,
		domain.EstadoPoliticaCotejoPublicada, domain.EstadoPoliticaCotejoRetirada,
		domain.AccionPoliticaCotejoRetirada)
}

func (s *Store) confirmarTransicionPoliticaCotejo(
	ctx context.Context,
	huellaAnterior string,
	politica domain.PoliticaCotejo,
	traza domain.AuditEntry,
	evento domain.Event,
	estadoAnterior, estadoNuevo domain.EstadoPoliticaCotejo,
	accion string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	canonica, err := politica.ClonarCanonica()
	if err != nil {
		return err
	}
	if canonica.Estado != estadoNuevo || strings.TrimSpace(huellaAnterior) == "" ||
		!evidenciaPoliticaCotejoValida(canonica, traza, evento, accion, huellaAnterior) {
		return domain.ErrPoliticaCotejoInvalida
	}
	clave := clavePoliticaCotejo(canonica.ID, canonica.Version)
	s.mu.Lock()
	defer s.mu.Unlock()
	actual, existe := s.politicasCotejo[clave]
	if !existe {
		return ports.ErrPoliticaCotejoNoEncontrada
	}
	huellaActual, err := actual.HuellaSHA256()
	if err != nil || actual.Estado != estadoAnterior || huellaActual != huellaAnterior ||
		canonica.Revision != actual.Revision+1 || !identidadPoliticaCotejoIgual(actual, canonica) {
		return ports.ErrRevisionPoliticaCotejoConflicto
	}
	var esperada domain.PoliticaCotejo
	if estadoNuevo == domain.EstadoPoliticaCotejoPublicada {
		esperada, err = actual.Publicar(canonica.PublicadaPor, canonica.AprobacionRef,
			canonica.MotivoPublicacion, canonica.PublicadaEn)
	} else {
		esperada, err = actual.Retirar(canonica.RetiradaPor, canonica.RetiradaAprobacionRef,
			canonica.MotivoRetirada, canonica.RetiradaEn)
	}
	if err != nil || !huellasPoliticaCotejoIguales(esperada, canonica) {
		return ports.ErrRevisionPoliticaCotejoConflicto
	}
	s.politicasCotejo[clave] = canonica
	s.confirmarEvidenciaCotejoBloqueado(traza, evento)
	return nil
}

func (s *Store) ObtenerPoliticaCotejo(ctx context.Context, id string, version int) (domain.PoliticaCotejo, error) {
	if err := ctx.Err(); err != nil {
		return domain.PoliticaCotejo{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	politica, existe := s.politicasCotejo[clavePoliticaCotejo(id, version)]
	if !existe {
		return domain.PoliticaCotejo{}, ports.ErrPoliticaCotejoNoEncontrada
	}
	return politica.ClonarCanonica()
}

func (s *Store) ListarVersionesPoliticaCotejo(ctx context.Context, id string) ([]domain.PoliticaCotejo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ports.ErrPoliticaCotejoNoEncontrada
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	resultado := make([]domain.PoliticaCotejo, 0)
	for _, politica := range s.politicasCotejo {
		if politica.ID != id {
			continue
		}
		clon, err := politica.ClonarCanonica()
		if err != nil {
			return nil, err
		}
		resultado = append(resultado, clon)
	}
	if len(resultado) == 0 {
		return nil, ports.ErrPoliticaCotejoNoEncontrada
	}
	sort.Slice(resultado, func(i, j int) bool { return resultado[i].Version < resultado[j].Version })
	return resultado, nil
}

func (s *Store) ReservarEmisionCodigoCotejo(
	ctx context.Context,
	solicitud ports.SolicitudReservarEmisionCodigoCotejo,
) (ports.ReservaEmisionCodigoCotejo, error) {
	if err := ctx.Err(); err != nil {
		return ports.ReservaEmisionCodigoCotejo{}, err
	}
	if !solicitudReservaCotejoValida(solicitud) {
		return ports.ReservaEmisionCodigoCotejo{}, ports.ErrClaveIdempotenciaCotejoInvalida
	}
	claveAmbito := claveAmbitoCotejo(solicitud.PrincipalID, solicitud.ClaveIdempotencia)
	instante := solicitud.SolicitadaEn.UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return ports.ReservaEmisionCodigoCotejo{}, err
	}
	if existente, existe := s.reservasCotejo[claveAmbito]; existe {
		if !huellasIguales(existente.HuellaSolicitudHMAC, solicitud.HuellaSolicitudHMAC) ||
			existente.Documento != solicitud.Documento || existente.Politica != solicitud.Politica {
			return ports.ReservaEmisionCodigoCotejo{}, ports.ErrClaveIdempotenciaCotejoReutilizada
		}
		switch existente.Estado {
		case estadoReservaCotejoConfirmada:
			codigo, err := existente.Codigo.ClonarCanonico()
			if err != nil {
				return ports.ReservaEmisionCodigoCotejo{}, ports.ErrReservaCodigoCotejoNoValida
			}
			return ports.ReservaEmisionCodigoCotejo{Repetida: true, Codigo: codigo}, nil
		case estadoReservaCotejoActiva:
			if instante.Before(existente.ExpiraEn) {
				return ports.ReservaEmisionCodigoCotejo{}, ports.ErrEmisionCodigoCotejoEnCurso
			}
			delete(s.reservasCotejoPorHuellaToken, existente.HuellaTokenSHA256)
		case estadoReservaCotejoAbandonada:
		default:
			return ports.ReservaEmisionCodigoCotejo{}, ports.ErrReservaCodigoCotejoNoValida
		}
	}
	token, err := ports.NuevoTokenReservaEmisionCodigoCotejo()
	if err != nil {
		return ports.ReservaEmisionCodigoCotejo{}, ports.ErrReservaCodigoCotejoNoValida
	}
	huellaToken, err := token.HuellaSHA256()
	if err != nil {
		return ports.ReservaEmisionCodigoCotejo{}, ports.ErrReservaCodigoCotejoNoValida
	}
	reserva := reservaEmisionCodigoCotejo{
		ClaveAmbito:         claveAmbito,
		PrincipalID:         strings.TrimSpace(solicitud.PrincipalID),
		HuellaSolicitudHMAC: solicitud.HuellaSolicitudHMAC,
		Documento:           solicitud.Documento,
		Politica:            solicitud.Politica,
		HuellaTokenSHA256:   huellaToken,
		Estado:              estadoReservaCotejoActiva,
		SolicitadaEn:        instante,
		ExpiraEn:            solicitud.ExpiraEn.UTC(),
	}
	s.reservasCotejo[claveAmbito] = reserva
	s.reservasCotejoPorHuellaToken[huellaToken] = claveAmbito
	return ports.ReservaEmisionCodigoCotejo{Token: token}, nil
}

func (s *Store) ConfirmarReservaCodigoCotejo(
	ctx context.Context,
	token ports.TokenReservaEmisionCodigoCotejo,
	huellaSolicitudHMAC string,
	confirmadaEn time.Time,
	codigo domain.CodigoCotejo,
	traza domain.AuditEntry,
	evento domain.Event,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	huellaToken, err := token.HuellaSHA256()
	if err != nil {
		return ports.ErrReservaCodigoCotejoNoValida
	}
	canonico, err := codigo.ClonarCanonico()
	if err != nil {
		return err
	}
	if !huellaHMACSHA256Valida(huellaSolicitudHMAC) || confirmadaEn.IsZero() ||
		canonico.Estado != domain.EstadoCodigoCotejoReservado || canonico.Revision != 1 ||
		!evidenciaCodigoCotejoValida(canonico, traza, evento, domain.AccionCodigoCotejoReservado, "", confirmadaEn.UTC()) {
		return ports.ErrReservaCodigoCotejoNoValida
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	claveAmbito, existe := s.reservasCotejoPorHuellaToken[huellaToken]
	if !existe {
		return ports.ErrReservaCodigoCotejoNoValida
	}
	reserva, existe := s.reservasCotejo[claveAmbito]
	if !existe || reserva.Estado != estadoReservaCotejoActiva ||
		!token.CoincideConHuellaSHA256(reserva.HuellaTokenSHA256) ||
		!huellasIguales(reserva.HuellaSolicitudHMAC, huellaSolicitudHMAC) ||
		reserva.PrincipalID != canonico.ReservadoPor || reserva.Documento != canonico.Documento ||
		reserva.Politica != canonico.Politica.Referencia || !confirmadaEn.UTC().Before(reserva.ExpiraEn) ||
		!canonico.ReservadoEn.Equal(reserva.SolicitadaEn) ||
		!s.vinculacionCodigoCotejoValidaBloqueado(canonico, true) ||
		!s.aplicacionPoliticaCotejoVigenteBloqueada(canonico.Politica) {
		return ports.ErrReservaCodigoCotejoNoValida
	}
	if _, existe := s.codigosCotejo[canonico.ID]; existe {
		return ports.ErrCodigoCotejoYaExiste
	}
	claveDocumento := claveDocumentoLogico(canonico.Documento)
	if _, existe := s.cotejoPorDocumento[claveDocumento]; existe {
		return ports.ErrDocumentoConCodigoCotejo
	}
	if _, existe := s.cotejoPorIndice[canonico.IndiceCodigoHMAC]; existe {
		return ports.ErrIndiceCodigoCotejoYaExiste
	}
	s.codigosCotejo[canonico.ID] = canonico
	s.cotejoPorDocumento[claveDocumento] = canonico.ID
	s.cotejoPorIndice[canonico.IndiceCodigoHMAC] = canonico.ID
	s.confirmarEvidenciaCotejoBloqueado(traza, evento)
	reserva.Estado = estadoReservaCotejoConfirmada
	reserva.Codigo = canonico
	s.reservasCotejo[claveAmbito] = reserva
	delete(s.reservasCotejoPorHuellaToken, huellaToken)
	return nil
}

func (s *Store) AbandonarReservaCodigoCotejo(ctx context.Context, token ports.TokenReservaEmisionCodigoCotejo) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	huellaToken, err := token.HuellaSHA256()
	if err != nil {
		return ports.ErrReservaCodigoCotejoNoValida
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	claveAmbito, existe := s.reservasCotejoPorHuellaToken[huellaToken]
	if !existe {
		return ports.ErrReservaCodigoCotejoNoValida
	}
	reserva, existe := s.reservasCotejo[claveAmbito]
	if !existe || reserva.Estado != estadoReservaCotejoActiva ||
		!token.CoincideConHuellaSHA256(reserva.HuellaTokenSHA256) {
		return ports.ErrReservaCodigoCotejoNoValida
	}
	reserva.Estado = estadoReservaCotejoAbandonada
	s.reservasCotejo[claveAmbito] = reserva
	delete(s.reservasCotejoPorHuellaToken, huellaToken)
	return nil
}

func (s *Store) ObtenerCodigoCotejo(ctx context.Context, id string) (domain.CodigoCotejo, error) {
	if err := ctx.Err(); err != nil {
		return domain.CodigoCotejo{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	codigo, existe := s.codigosCotejo[strings.TrimSpace(id)]
	if !existe {
		return domain.CodigoCotejo{}, ports.ErrCodigoCotejoNoEncontrado
	}
	return codigo.ClonarCanonico()
}

func (s *Store) ObtenerCodigoCotejoPorDocumento(ctx context.Context, referencia domain.ReferenciaDocumento) (domain.CodigoCotejo, error) {
	if err := ctx.Err(); err != nil {
		return domain.CodigoCotejo{}, err
	}
	if referencia.Validar() != nil {
		return domain.CodigoCotejo{}, ports.ErrCodigoCotejoNoEncontrado
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, existe := s.cotejoPorDocumento[claveDocumentoLogico(referencia)]
	if !existe {
		return domain.CodigoCotejo{}, ports.ErrCodigoCotejoNoEncontrado
	}
	return s.codigosCotejo[id].ClonarCanonico()
}

func (s *Store) BuscarCodigoCotejoPorIndices(ctx context.Context, indices []string) (domain.CodigoCotejo, error) {
	if err := ctx.Err(); err != nil {
		return domain.CodigoCotejo{}, err
	}
	canonicos, validos := indicesCotejoMemoriaValidos(indices)
	if !validos {
		return domain.CodigoCotejo{}, ports.ErrMaterialCodigoCotejoInvalido
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	coincidencias := make(map[string]struct{})
	for _, indice := range canonicos {
		if id, existe := s.cotejoPorIndice[indice]; existe {
			coincidencias[id] = struct{}{}
		}
	}
	if len(coincidencias) == 0 {
		return domain.CodigoCotejo{}, ports.ErrCodigoCotejoNoEncontrado
	}
	if len(coincidencias) != 1 {
		return domain.CodigoCotejo{}, ports.ErrIndicesCodigoCotejoAmbiguos
	}
	for id := range coincidencias {
		return s.codigosCotejo[id].ClonarCanonico()
	}
	return domain.CodigoCotejo{}, ports.ErrCodigoCotejoNoEncontrado
}

func (s *Store) ConfirmarActivacionCodigoCotejo(ctx context.Context, huellaAnterior string, codigo domain.CodigoCotejo, traza domain.AuditEntry, evento domain.Event) error {
	return s.confirmarTransicionCodigoCotejo(ctx, huellaAnterior, codigo, traza, evento,
		domain.EstadoCodigoCotejoReservado, domain.EstadoCodigoCotejoActivo, domain.AccionCodigoCotejoActivado)
}

func (s *Store) ConfirmarRetiradaCodigoCotejo(ctx context.Context, huellaAnterior string, codigo domain.CodigoCotejo, traza domain.AuditEntry, evento domain.Event) error {
	return s.confirmarTransicionCodigoCotejo(ctx, huellaAnterior, codigo, traza, evento,
		domain.EstadoCodigoCotejoActivo, domain.EstadoCodigoCotejoRetirado, domain.AccionCodigoCotejoRetirado)
}

func (s *Store) ConfirmarSustitucionCodigoCotejo(ctx context.Context, huellaAnterior string, codigo domain.CodigoCotejo, traza domain.AuditEntry, evento domain.Event) error {
	return s.confirmarTransicionCodigoCotejo(ctx, huellaAnterior, codigo, traza, evento,
		domain.EstadoCodigoCotejoActivo, domain.EstadoCodigoCotejoSustituido, domain.AccionCodigoCotejoSustituido)
}

func (s *Store) confirmarTransicionCodigoCotejo(
	ctx context.Context,
	huellaAnterior string,
	codigo domain.CodigoCotejo,
	traza domain.AuditEntry,
	evento domain.Event,
	estadoAnterior, estadoNuevo domain.EstadoCodigoCotejo,
	accion string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	canonico, err := codigo.ClonarCanonico()
	if err != nil {
		return err
	}
	instante := instanteTransicionCodigoCotejo(canonico)
	if canonico.Estado != estadoNuevo || strings.TrimSpace(huellaAnterior) == "" || instante.IsZero() ||
		!evidenciaCodigoCotejoValida(canonico, traza, evento, accion, huellaAnterior, instante) {
		return domain.ErrCodigoCotejoInvalido
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	actual, existe := s.codigosCotejo[canonico.ID]
	if !existe {
		return ports.ErrCodigoCotejoNoEncontrado
	}
	huellaActual, err := actual.HuellaEstadoSHA256()
	if err != nil || actual.Estado != estadoAnterior || huellaActual != huellaAnterior ||
		canonico.Revision != actual.Revision+1 {
		return ports.ErrRevisionCodigoCotejoConflicto
	}
	var esperada domain.CodigoCotejo
	switch estadoNuevo {
	case domain.EstadoCodigoCotejoActivo:
		if !s.vinculacionCodigoCotejoValidaBloqueado(canonico, false) ||
			!s.aplicacionPoliticaCotejoVigenteBloqueada(canonico.Politica) ||
			!s.versionEmitidaCodigoCotejoValidaBloqueada(canonico) {
			return domain.ErrEvidenciaEmisionInvalida
		}
		esperada, err = actual.Activar(canonico.ActivadoPor, canonico.ActivacionRef, canonico.MotivoActivacion,
			domain.EvidenciaEmisionDocumento{
				Documento:      canonico.Documento,
				VersionEmitida: *canonico.VersionEmitida,
				Apta:           true,
				EvidenciaRef:   canonico.EvidenciaEmisionRef,
			}, canonico.ActivadoEn)
	case domain.EstadoCodigoCotejoRetirado:
		esperada, err = actual.Retirar(canonico.RetiradoPor, canonico.RetiradaRef,
			canonico.MotivoRetirada, canonico.RetiradoEn)
	case domain.EstadoCodigoCotejoSustituido:
		esperada, err = actual.Sustituir(canonico.RetiradoPor, canonico.RetiradaRef,
			canonico.MotivoRetirada, canonico.SustituidoPorRef, canonico.RetiradoEn)
		if err == nil {
			sustitutoID := strings.TrimPrefix(canonico.SustituidoPorRef, "cotejo:")
			sustituto, existe := s.codigosCotejo[sustitutoID]
			if !existe || sustituto.Referencia() != canonico.SustituidoPorRef || !sustituto.DisponibleEn(canonico.RetiradoEn) {
				return domain.ErrTransicionCodigoCotejo
			}
		}
	default:
		return domain.ErrTransicionCodigoCotejo
	}
	if err != nil || !huellasCodigoCotejoIguales(esperada, canonico) {
		return ports.ErrRevisionCodigoCotejoConflicto
	}
	s.codigosCotejo[canonico.ID] = canonico
	s.confirmarEvidenciaCotejoBloqueado(traza, evento)
	return nil
}

func (s *Store) RegistrarConsultaCotejo(ctx context.Context, traza domain.AuditEntry, evento domain.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !evidenciaConsultaCotejoValida(traza, evento) {
		return domain.ErrCodigoCotejoInvalido
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if traza.SubjectRef != "cotejo:consulta-no-resuelta" {
		id := strings.TrimPrefix(traza.SubjectRef, "cotejo:")
		codigo, existe := s.codigosCotejo[id]
		if !existe || codigo.Referencia() != traza.SubjectRef || codigo.Revision != traza.ObjectVersion {
			return ports.ErrCodigoCotejoNoEncontrado
		}
	}
	s.confirmarEvidenciaCotejoBloqueado(traza, evento)
	return nil
}

func (s *Store) confirmarEvidenciaCotejoBloqueado(traza domain.AuditEntry, evento domain.Event) {
	trazaConfirmada := s.appendAuditLocked(traza)
	evento.Payload = cloneStringMap(evento.Payload)
	if evento.Payload == nil {
		evento.Payload = map[string]string{}
	}
	evento.Payload["auditoria_ref"] = trazaConfirmada.ID
	s.appendEventLocked(evento)
}

func evidenciaPoliticaCotejoValida(
	politica domain.PoliticaCotejo,
	traza domain.AuditEntry,
	evento domain.Event,
	accion, huellaAnterior string,
) bool {
	huella, err := politica.HuellaSHA256()
	if err != nil {
		return false
	}
	actor, instante, regla, motivo := politica.CreadaPor, politica.CreadaEn, politica.FuenteRef, politica.MotivoCreacion
	switch accion {
	case domain.AccionPoliticaCotejoBorradorActualizada:
		actor, instante, motivo = politica.ActualizadaPor, politica.ActualizadaEn, politica.MotivoActualizacion
	case domain.AccionPoliticaCotejoPublicada:
		actor, instante, regla, motivo = politica.PublicadaPor, politica.PublicadaEn, politica.AprobacionRef, politica.MotivoPublicacion
	case domain.AccionPoliticaCotejoRetirada:
		actor, instante, regla, motivo = politica.RetiradaPor, politica.RetiradaEn, politica.RetiradaAprobacionRef, politica.MotivoRetirada
	case domain.AccionPoliticaCotejoBorradorCreada:
	default:
		return false
	}
	return traza.ActorID == actor && strings.TrimSpace(traza.ActorProfile) != "" && traza.AuthMethod.Valido() &&
		traza.AuthAssurance.Valida() && strings.TrimSpace(traza.AuthorizationRef) != "" && strings.TrimSpace(traza.Purpose) != "" &&
		traza.Action == accion && traza.ModuleID == "documentos" && traza.SubjectRef == politica.Referencia() &&
		traza.ObjectVersion == politica.Revision && traza.RuleRef == regla && traza.Reason == motivo && traza.Result == "correcto" &&
		traza.BeforeHash == huellaAnterior && traza.AfterHash == huella && strings.TrimSpace(traza.CorrelationRef) != "" &&
		traza.OccurredAt.Equal(instante) && metadatosCotejoSinSecretos(traza.Metadata) &&
		evento.Type == accion && evento.ModuleID == "documentos" && evento.SubjectRef == politica.Referencia() &&
		evento.ActorID == actor && evento.OccurredAt.Equal(instante) && evento.Payload["politica_id"] == politica.ID &&
		evento.Payload["politica_version"] == strconv.Itoa(politica.Version) &&
		evento.Payload["revision"] == strconv.Itoa(politica.Revision) && evento.Payload["estado"] == string(politica.Estado) &&
		evento.Payload["huella_sha256"] == huella && metadatosCotejoSinSecretos(evento.Payload)
}

func evidenciaCodigoCotejoValida(
	codigo domain.CodigoCotejo,
	traza domain.AuditEntry,
	evento domain.Event,
	accion, huellaAnterior string,
	instante time.Time,
) bool {
	huella, err := codigo.HuellaEstadoSHA256()
	if err != nil {
		return false
	}
	actor := codigo.ReservadoPor
	motivo := codigo.MotivoReserva
	regla := codigo.Politica.Referencia.ID + ":" + strconv.Itoa(codigo.Politica.Referencia.Version)
	switch accion {
	case domain.AccionCodigoCotejoActivado:
		actor, motivo, regla = codigo.ActivadoPor, codigo.MotivoActivacion, codigo.EvidenciaEmisionRef
	case domain.AccionCodigoCotejoRetirado, domain.AccionCodigoCotejoSustituido:
		actor, motivo, regla = codigo.RetiradoPor, codigo.MotivoRetirada, codigo.RetiradaRef
	case domain.AccionCodigoCotejoReservado:
	default:
		return false
	}
	return traza.ActorID == actor && strings.TrimSpace(traza.ActorProfile) != "" && traza.AuthMethod.Valido() &&
		traza.AuthAssurance.Valida() && strings.TrimSpace(traza.AuthorizationRef) != "" && strings.TrimSpace(traza.Purpose) != "" &&
		traza.Action == accion && traza.ModuleID == codigo.ModuloID && traza.SubjectRef == codigo.Referencia() &&
		traza.ObjectVersion == codigo.Revision && traza.ExpedienteRef == codigo.ExpedienteRef &&
		traza.DocumentRef == claveDocumentoLogico(codigo.Documento) && traza.RuleRef == regla && traza.Reason == motivo &&
		traza.Result == "correcto" && traza.BeforeHash == huellaAnterior && traza.AfterHash == huella &&
		traza.CorrelationRef == codigo.CorrelacionRef && traza.OccurredAt.Equal(instante) && metadatosCotejoSinSecretos(traza.Metadata) &&
		evento.Type == accion && evento.ModuleID == codigo.ModuloID && evento.SubjectRef == codigo.Referencia() &&
		evento.ActorID == actor && evento.OccurredAt.Equal(instante) && evento.Payload["codigo_ref"] == codigo.Referencia() &&
		evento.Payload["documento_ref"] == claveDocumentoLogico(codigo.Documento) &&
		evento.Payload["revision"] == strconv.Itoa(codigo.Revision) && evento.Payload["estado"] == string(codigo.Estado) &&
		evento.Payload["huella_estado"] == huella && metadatosCotejoSinSecretos(evento.Payload)
}

func evidenciaConsultaCotejoValida(traza domain.AuditEntry, evento domain.Event) bool {
	if traza.Action != domain.AccionConsultaPublicaCotejo && traza.Action != domain.AccionConsultaProtegidaCotejo {
		return false
	}
	publica := traza.Action == domain.AccionConsultaPublicaCotejo
	if publica {
		if traza.ActorID != "publico-anonimo" || traza.ActorProfile != "publico" ||
			traza.Purpose != "verificacion_documental_publica" || traza.AuthMethod != "" || traza.AuthAssurance != "" ||
			traza.AuthorizationRef != "" || traza.ExpedienteRef != "" || traza.DocumentRef != "" {
			return false
		}
	} else if strings.TrimSpace(traza.ActorID) == "" || strings.TrimSpace(traza.ActorProfile) == "" ||
		strings.TrimSpace(traza.Purpose) == "" || !traza.AuthMethod.Valido() || !traza.AuthAssurance.Valida() {
		return false
	}
	return strings.TrimSpace(traza.SubjectRef) != "" && strings.TrimSpace(traza.Result) != "" &&
		strings.TrimSpace(traza.CorrelationRef) != "" && !traza.OccurredAt.IsZero() && traza.BeforeHash == "" && traza.AfterHash == "" &&
		metadatosCotejoSinSecretos(traza.Metadata) && evento.Type == traza.Action && evento.ModuleID == traza.ModuleID &&
		evento.SubjectRef == traza.SubjectRef && evento.ActorID == traza.ActorID && evento.OccurredAt.Equal(traza.OccurredAt) &&
		evento.Payload["resultado_consulta"] == traza.Result && metadatosCotejoSinSecretos(evento.Payload)
}

func metadatosCotejoSinSecretos(metadatos map[string]string) bool {
	for clave, valor := range metadatos {
		texto := strings.ToLower(clave + "\x00" + valor)
		if strings.Contains(texto, "hmac-sha256") || strings.Contains(texto, "proteccion_ref") ||
			strings.Contains(texto, "secreto_cotejo") || strings.Contains(texto, "valor_csv") {
			return false
		}
	}
	return true
}

// vinculacionCodigoCotejoValidaBloqueado repite en el limite de persistencia
// las comprobaciones realizadas por aplicacion. En un adaptador SQL estas
// condiciones deben formar parte de la misma transaccion que agregado,
// auditoria y outbox, para cerrar cambios concurrentes de politica/documento.
// El llamador debe mantener s.mu.
func (s *Store) vinculacionCodigoCotejoValidaBloqueado(codigo domain.CodigoCotejo, paraReserva bool) bool {
	documento, existe := s.documentosLogicos[claveDocumentoLogico(codigo.Documento)]
	if !existe || documento.Validar() != nil || documento.Referencia() != codigo.Documento ||
		documento.ModuloID != codigo.ModuloID || documento.TipoDocumental != codigo.TipoDocumental ||
		documento.Clasificacion != codigo.Clasificacion || documento.ENI.Organo != codigo.Organo {
		return false
	}
	expediente := ""
	coincidencias := 0
	for _, relacion := range documento.Relaciones {
		if relacion.Tipo == domain.TipoRelacionExpediente && relacion.Rol == "principal" {
			expediente = relacion.Referencia
			coincidencias++
		}
	}
	if coincidencias != 1 || expediente != codigo.ExpedienteRef {
		return false
	}
	if !paraReserva {
		return estadoDocumentoPermiteActivacionCotejoMemoria(documento.Estado, codigo.Politica)
	}
	switch documento.Estado {
	case domain.EstadoDocumentoLogicoBorrador, domain.EstadoDocumentoLogicoEnRevision,
		domain.EstadoDocumentoLogicoCerrado, domain.EstadoDocumentoLogicoPendienteFirma:
		return true
	default:
		return false
	}
}

// aplicacionPoliticaCotejoVigenteBloqueada impide confirmar una instantanea
// inventada o una politica retirada entre la autorizacion y el commit.
// El llamador debe mantener s.mu.
func (s *Store) aplicacionPoliticaCotejoVigenteBloqueada(aplicacion domain.AplicacionPoliticaCotejo) bool {
	politica, existe := s.politicasCotejo[clavePoliticaCotejo(aplicacion.Referencia.ID, aplicacion.Referencia.Version)]
	if !existe || politica.Estado != domain.EstadoPoliticaCotejoPublicada {
		return false
	}
	vigente, err := politica.Aplicacion()
	return err == nil && reflect.DeepEqual(vigente, aplicacion)
}

// versionEmitidaCodigoCotejoValidaBloqueada enlaza la activacion con los bytes
// exactos ya aceptados por el repositorio documental. Firma, sellado temporal y
// registro proceden de la evidencia interna fijada en el agregado.
// El llamador debe mantener s.mu.
func (s *Store) versionEmitidaCodigoCotejoValidaBloqueada(codigo domain.CodigoCotejo) bool {
	if codigo.VersionEmitida == nil {
		return false
	}
	documento, existe := s.documentosLogicos[claveDocumentoLogico(codigo.Documento)]
	if !existe {
		return false
	}
	representacion, existe := s.representaciones[codigo.VersionEmitida.RepresentacionID]
	if !existe || representacion.ValidarPertenencia(documento) != nil ||
		representacion.EstadoTecnico != domain.EstadoRepresentacionDisponible ||
		(representacion.EstadoAntivirus != domain.EstadoAntivirusLimpio &&
			representacion.EstadoAntivirus != domain.EstadoAntivirusNoAplica) ||
		representacion.Tipo == domain.TipoRepresentacionTrabajo ||
		(codigo.Politica.RequiereFirma && representacion.Tipo != domain.TipoRepresentacionFirma &&
			representacion.Tipo != domain.TipoRepresentacionPreservacion) {
		return false
	}
	version := codigo.VersionEmitida
	return version.RepresentacionID == representacion.ID &&
		version.ReferenciaContenido == representacion.ReferenciaContenido &&
		version.HuellaContenidoSHA256 == representacion.HuellaContenidoSHA256 &&
		version.MIME == representacion.MIME && version.Tamano == representacion.Tamano &&
		!version.EmitidaEn.Before(representacion.GeneradaEn)
}

func estadoDocumentoPermiteActivacionCotejoMemoria(estado domain.EstadoDocumentoLogico, politica domain.AplicacionPoliticaCotejo) bool {
	if politica.RequiereRegistro {
		return estado == domain.EstadoDocumentoLogicoRegistrado
	}
	if politica.RequiereFirma {
		return estado == domain.EstadoDocumentoLogicoFirmado || estado == domain.EstadoDocumentoLogicoRegistrado
	}
	return estado == domain.EstadoDocumentoLogicoCerrado || estado == domain.EstadoDocumentoLogicoFirmado ||
		estado == domain.EstadoDocumentoLogicoRegistrado
}

func solicitudReservaCotejoValida(solicitud ports.SolicitudReservarEmisionCodigoCotejo) bool {
	clave := solicitud.ClaveIdempotencia
	principal := solicitud.PrincipalID
	return clave == strings.TrimSpace(clave) && len(clave) >= longitudMinimaClaveIdempotencia &&
		len(clave) <= longitudMaximaClaveIdempotencia && textoMemoriaValido(clave) &&
		principal == strings.TrimSpace(principal) && principal != "" && len(principal) <= 512 && textoMemoriaValido(principal) &&
		huellaHMACSHA256Valida(solicitud.HuellaSolicitudHMAC) && solicitud.Documento.Validar() == nil &&
		solicitud.Politica.Validar() == nil && !solicitud.SolicitadaEn.IsZero() && !solicitud.ExpiraEn.IsZero() &&
		solicitud.ExpiraEn.After(solicitud.SolicitadaEn) &&
		solicitud.ExpiraEn.Sub(solicitud.SolicitadaEn) <= vigenciaMaximaReservaCotejoMemoria
}

func indicesCotejoMemoriaValidos(indices []string) ([]string, bool) {
	if len(indices) == 0 || len(indices) > 16 {
		return nil, false
	}
	canonicos := append([]string(nil), indices...)
	sort.Strings(canonicos)
	for indice, valor := range canonicos {
		if !huellaHMACSHA256Valida(valor) || (indice > 0 && valor == canonicos[indice-1]) {
			return nil, false
		}
	}
	return canonicos, true
}

func claveAmbitoCotejo(principalID, clave string) string {
	suma := sha256.Sum256([]byte(strings.TrimSpace(principalID) + "\x00" + clave))
	return hex.EncodeToString(suma[:])
}

func clavePoliticaCotejo(id string, version int) string {
	return strings.TrimSpace(id) + ":" + strconv.Itoa(version)
}

func identidadPoliticaCotejoIgual(anterior, posterior domain.PoliticaCotejo) bool {
	return anterior.ID == posterior.ID && anterior.Version == posterior.Version &&
		anterior.VersionAnteriorRef == posterior.VersionAnteriorRef && anterior.CreadaPor == posterior.CreadaPor &&
		anterior.CreadaEn.Equal(posterior.CreadaEn) && anterior.MotivoCreacion == posterior.MotivoCreacion
}

func huellasPoliticaCotejoIguales(primera, segunda domain.PoliticaCotejo) bool {
	huellaPrimera, errPrimera := primera.HuellaSHA256()
	huellaSegunda, errSegunda := segunda.HuellaSHA256()
	return errPrimera == nil && errSegunda == nil && huellaPrimera == huellaSegunda
}

func huellasCodigoCotejoIguales(primero, segundo domain.CodigoCotejo) bool {
	huellaPrimera, errPrimera := primero.HuellaEstadoSHA256()
	huellaSegunda, errSegunda := segundo.HuellaEstadoSHA256()
	return errPrimera == nil && errSegunda == nil && huellaPrimera == huellaSegunda
}

func instanteTransicionCodigoCotejo(codigo domain.CodigoCotejo) time.Time {
	switch codigo.Estado {
	case domain.EstadoCodigoCotejoActivo:
		return codigo.ActivadoEn
	case domain.EstadoCodigoCotejoRetirado, domain.EstadoCodigoCotejoSustituido:
		return codigo.RetiradoEn
	default:
		return time.Time{}
	}
}

var _ ports.CatalogoPoliticasCotejo = (*Store)(nil)
var _ ports.RepositorioGobiernoPoliticasCotejo = (*Store)(nil)
var _ ports.RepositorioCodigosCotejo = (*Store)(nil)

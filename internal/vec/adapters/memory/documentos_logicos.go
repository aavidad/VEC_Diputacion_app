package memory

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	longitudMinimaClaveIdempotencia = 16
	longitudMaximaClaveIdempotencia = 200
	vigenciaMaximaReservaDocumento  = 10 * time.Minute
)

func (s *Store) ReservarGeneracion(ctx context.Context, solicitud ports.SolicitudReservarGeneracionDocumento) (ports.ReservaGeneracionDocumento, error) {
	if err := ctx.Err(); err != nil {
		return ports.ReservaGeneracionDocumento{}, err
	}
	if !solicitudReservaDocumentoValida(solicitud) {
		return ports.ReservaGeneracionDocumento{}, ports.ErrClaveIdempotenciaInvalida
	}
	principalID := strings.TrimSpace(solicitud.PrincipalID)
	claveAmbito := claveAmbitoIdempotenciaDocumento(principalID, solicitud.ClaveIdempotencia)
	instante := solicitud.SolicitadaEn.UTC()

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return ports.ReservaGeneracionDocumento{}, err
	}
	if existente, existe := s.reservasDocumentales[claveAmbito]; existe {
		if !huellasIguales(existente.HuellaSolicitudHMAC, solicitud.HuellaSolicitudHMAC) {
			return ports.ReservaGeneracionDocumento{}, ports.ErrClaveIdempotenciaReutilizada
		}
		switch existente.Estado {
		case estadoReservaDocumentalConfirmada:
			resultado, err := existente.Resultado.ClonarCanonico()
			if err != nil {
				return ports.ReservaGeneracionDocumento{}, ports.ErrReservaDocumentoNoValida
			}
			resultado.Repetida = true
			return ports.ReservaGeneracionDocumento{Repetida: true, Resultado: resultado}, nil
		case estadoReservaDocumentalActiva:
			if instante.Before(existente.ExpiraEn) {
				return ports.ReservaGeneracionDocumento{}, ports.ErrGeneracionDocumentoEnCurso
			}
			if err := ctx.Err(); err != nil {
				return ports.ReservaGeneracionDocumento{}, err
			}
			delete(s.reservasPorToken, existente.Token)
		case estadoReservaDocumentalAbandonada:
			// Una operacion fallida puede repetirse con la misma huella, pero la
			// clave permanece ligada para impedir cambiar su significado.
		default:
			return ports.ReservaGeneracionDocumento{}, ports.ErrReservaDocumentoNoValida
		}
	}

	if err := ctx.Err(); err != nil {
		return ports.ReservaGeneracionDocumento{}, err
	}
	s.secuenciaReservas++
	token := fmt.Sprintf("reserva-documental-%012d", s.secuenciaReservas)
	reserva := reservaGeneracionDocumento{
		ClaveAmbito:         claveAmbito,
		PrincipalID:         principalID,
		HuellaSolicitudHMAC: solicitud.HuellaSolicitudHMAC,
		Token:               token,
		Estado:              estadoReservaDocumentalActiva,
		ExpiraEn:            solicitud.ExpiraEn.UTC(),
	}
	s.reservasDocumentales[claveAmbito] = reserva
	s.reservasPorToken[token] = claveAmbito
	return ports.ReservaGeneracionDocumento{Token: token}, nil
}

func (s *Store) ConfirmarGeneracionLogica(
	ctx context.Context,
	token string,
	huellaSolicitudHMAC string,
	confirmadaEn time.Time,
	resultado domain.ResultadoGeneracionDocumento,
	traza domain.AuditEntry,
	evento domain.Event,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(token) == "" || !huellaHMACSHA256Valida(huellaSolicitudHMAC) || confirmadaEn.IsZero() || resultado.Repetida {
		return ports.ErrReservaDocumentoNoValida
	}
	canonico, err := resultado.ClonarCanonico()
	if err != nil {
		return err
	}
	if !trazaDocumentoLogicoValida(canonico, traza, evento, confirmadaEn.UTC()) {
		return domain.ErrDocumentoLogicoInvalido
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	claveAmbito, existe := s.reservasPorToken[token]
	if !existe {
		return ports.ErrReservaDocumentoNoValida
	}
	reserva, existe := s.reservasDocumentales[claveAmbito]
	if !existe || reserva.Estado != estadoReservaDocumentalActiva || reserva.Token != token ||
		reserva.PrincipalID != canonico.Documento.CreadoPor ||
		!huellasIguales(reserva.HuellaSolicitudHMAC, huellaSolicitudHMAC) ||
		!confirmadaEn.UTC().Before(reserva.ExpiraEn) {
		return ports.ErrReservaDocumentoNoValida
	}

	claveDocumento := claveDocumentoLogico(canonico.Documento.Referencia())
	if _, existe := s.documentosLogicos[claveDocumento]; existe {
		return ports.ErrDocumentoYaExiste
	}
	for _, representacion := range canonico.Representaciones {
		if _, existe := s.representaciones[representacion.ID]; existe {
			return ports.ErrDocumentoYaExiste
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	s.documentosLogicos[claveDocumento] = canonico.Documento
	for _, representacion := range canonico.Representaciones {
		s.representaciones[representacion.ID] = representacion
	}
	trazaConfirmada := s.appendAuditLocked(traza)
	evento.Payload = cloneStringMap(evento.Payload)
	if evento.Payload == nil {
		evento.Payload = map[string]string{}
	}
	evento.Payload["auditoria_ref"] = trazaConfirmada.ID
	s.appendEventLocked(evento)

	reserva.Estado = estadoReservaDocumentalConfirmada
	reserva.Resultado = canonico
	s.reservasDocumentales[claveAmbito] = reserva
	delete(s.reservasPorToken, token)
	return nil
}

func (s *Store) AbandonarGeneracion(ctx context.Context, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return ports.ErrReservaDocumentoNoValida
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	claveAmbito, existe := s.reservasPorToken[token]
	if !existe {
		return ports.ErrReservaDocumentoNoValida
	}
	reserva, existe := s.reservasDocumentales[claveAmbito]
	if !existe || reserva.Estado != estadoReservaDocumentalActiva || reserva.Token != token {
		return ports.ErrReservaDocumentoNoValida
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	reserva.Estado = estadoReservaDocumentalAbandonada
	s.reservasDocumentales[claveAmbito] = reserva
	delete(s.reservasPorToken, token)
	return nil
}

func (s *Store) ObtenerDocumentoLogico(ctx context.Context, referencia domain.ReferenciaDocumento) (domain.DocumentoLogico, error) {
	if err := ctx.Err(); err != nil {
		return domain.DocumentoLogico{}, err
	}
	if err := referencia.Validar(); err != nil {
		return domain.DocumentoLogico{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	documento, existe := s.documentosLogicos[claveDocumentoLogico(referencia)]
	if !existe {
		return domain.DocumentoLogico{}, ports.ErrDocumentoLogicoNoEncontrado
	}
	return documento.ClonarCanonico()
}

func (s *Store) ListarRepresentacionesDocumento(ctx context.Context, referencia domain.ReferenciaDocumento) ([]domain.RepresentacionDocumento, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := referencia.Validar(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	resultado := make([]domain.RepresentacionDocumento, 0)
	for _, representacion := range s.representaciones {
		if representacion.Documento == referencia {
			resultado = append(resultado, representacion)
		}
	}
	if len(resultado) == 0 {
		if _, existe := s.documentosLogicos[claveDocumentoLogico(referencia)]; !existe {
			return nil, ports.ErrDocumentoLogicoNoEncontrado
		}
	}
	sort.Slice(resultado, func(i, j int) bool { return resultado[i].ID < resultado[j].ID })
	return append([]domain.RepresentacionDocumento(nil), resultado...), nil
}

func solicitudReservaDocumentoValida(solicitud ports.SolicitudReservarGeneracionDocumento) bool {
	clave := solicitud.ClaveIdempotencia
	principalID := solicitud.PrincipalID
	return clave == strings.TrimSpace(clave) && len(clave) >= longitudMinimaClaveIdempotencia &&
		len(clave) <= longitudMaximaClaveIdempotencia && textoMemoriaValido(clave) &&
		principalID == strings.TrimSpace(principalID) && principalID != "" && len(principalID) <= 512 &&
		textoMemoriaValido(principalID) && huellaHMACSHA256Valida(solicitud.HuellaSolicitudHMAC) &&
		!solicitud.SolicitadaEn.IsZero() && !solicitud.ExpiraEn.IsZero() &&
		solicitud.ExpiraEn.After(solicitud.SolicitadaEn) &&
		solicitud.ExpiraEn.Sub(solicitud.SolicitadaEn) <= vigenciaMaximaReservaDocumento
}

func claveAmbitoIdempotenciaDocumento(principalID, clave string) string {
	suma := sha256.Sum256([]byte(principalID + "\x00" + clave))
	return hex.EncodeToString(suma[:])
}

func claveDocumentoLogico(referencia domain.ReferenciaDocumento) string {
	return referencia.ID + ":" + strconv.Itoa(referencia.Version)
}

func huellasIguales(primera, segunda string) bool {
	return len(primera) == len(segunda) && subtle.ConstantTimeCompare([]byte(primera), []byte(segunda)) == 1
}

func huellaHMACSHA256Valida(valor string) bool {
	partes := strings.Split(valor, ":")
	if len(partes) != 3 || partes[0] != "hmac-sha256" || partes[1] == "" || len(partes[1]) > 128 || !textoMemoriaValido(partes[1]) {
		return false
	}
	if len(partes[2]) != 64 {
		return false
	}
	for _, caracter := range partes[2] {
		if (caracter < '0' || caracter > '9') && (caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}

func textoMemoriaValido(valor string) bool {
	if !utf8.ValidString(valor) {
		return false
	}
	for _, caracter := range valor {
		if caracter < 0x20 || caracter == 0x7f {
			return false
		}
	}
	return true
}

func trazaDocumentoLogicoValida(resultado domain.ResultadoGeneracionDocumento, traza domain.AuditEntry, evento domain.Event, confirmadaEn time.Time) bool {
	documento := resultado.Documento
	referencia := claveDocumentoLogico(documento.Referencia())
	referenciaPlantilla := documento.Plantilla.ID + ":" + strconv.Itoa(documento.Plantilla.Version)
	return traza.ActorID == documento.CreadoPor && strings.TrimSpace(traza.ActorProfile) != "" &&
		traza.AuthMethod.Valido() && traza.AuthAssurance.Valida() &&
		strings.TrimSpace(traza.AuthorizationRef) != "" && strings.TrimSpace(traza.Purpose) != "" &&
		traza.Action == "vec.documento.logico.generado" && traza.ModuleID == documento.ModuloID &&
		traza.SubjectRef == referencia && traza.ObjectVersion == documento.Version &&
		traza.DocumentRef == referencia && traza.RuleRef == referenciaPlantilla &&
		strings.TrimSpace(traza.Reason) == documento.Motivo && traza.Result == "correcto" &&
		traza.BeforeHash == "" && traza.AfterHash == documento.HuellaFuenteHMAC &&
		traza.CorrelationRef == documento.CorrelacionRef && traza.OccurredAt.Equal(confirmadaEn) &&
		evento.Type == "vec.documento.logico.generado" && evento.ModuleID == documento.ModuloID &&
		evento.SubjectRef == referencia && evento.ActorID == documento.CreadoPor && evento.OccurredAt.Equal(confirmadaEn) &&
		evento.Payload["documento_ref"] == documento.ID &&
		evento.Payload["documento_version"] == strconv.Itoa(documento.Version) &&
		evento.Payload["huella_fuente_hmac"] == documento.HuellaFuenteHMAC &&
		evento.Payload["representaciones"] == strconv.Itoa(len(resultado.Representaciones))
}

var _ ports.RepositorioDocumentosLogicos = (*Store)(nil)

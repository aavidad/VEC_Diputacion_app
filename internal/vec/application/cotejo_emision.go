package application

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	eventoCodigoCotejoReservado  = domain.AccionCodigoCotejoReservado
	eventoCodigoCotejoActivado   = domain.AccionCodigoCotejoActivado
	eventoCodigoCotejoRetirado   = domain.AccionCodigoCotejoRetirado
	eventoCodigoCotejoSustituido = domain.AccionCodigoCotejoSustituido
)

type OrdenReservarCodigoCotejo struct {
	Principal         domain.Principal
	PerfilActivo      string
	RepresentadoRef   string
	Finalidad         string
	ClaveIdempotencia string
	Documento         domain.ReferenciaDocumento
	PoliticaID        string
	PoliticaVersion   int
	Motivo            string
	CorrelacionRef    string
}

// ResultadoReservaCodigoCotejo mantiene el secreto fuera de cualquier JSON.
// Solo el adaptador que prepara el sello PDF/QR puede llamar a Revelar.
type ResultadoReservaCodigoCotejo struct {
	Codigo   domain.CodigoCotejo       `json:"codigo"`
	Secreto  ports.SecretoCodigoCotejo `json:"-"`
	Repetida bool                      `json:"repetida"`
}

func (s *ServicioCotejo) ReservarCodigoCotejo(ctx context.Context, orden OrdenReservarCodigoCotejo) (resultado ResultadoReservaCodigoCotejo, err error) {
	if err := ctx.Err(); err != nil {
		return ResultadoReservaCodigoCotejo{}, err
	}
	if err := validarContextoCotejo(orden.Principal, orden.PerfilActivo, orden.Finalidad,
		orden.Motivo, orden.CorrelacionRef, domain.AuthAssuranceHigh); err != nil {
		return ResultadoReservaCodigoCotejo{}, err
	}
	if orden.Documento.Validar() != nil || strings.TrimSpace(orden.PoliticaID) == "" || orden.PoliticaVersion < 1 ||
		orden.ClaveIdempotencia != strings.TrimSpace(orden.ClaveIdempotencia) || orden.ClaveIdempotencia == "" {
		return ResultadoReservaCodigoCotejo{}, ErrOrdenCotejoInvalida
	}
	documento, err := s.documentos.ObtenerDocumentoLogico(ctx, orden.Documento)
	if err != nil {
		return ResultadoReservaCodigoCotejo{}, err
	}
	if !estadoDocumentoPermiteReservaCotejo(documento.Estado) {
		return ResultadoReservaCodigoCotejo{}, domain.ErrDocumentoLogicoInvalido
	}
	politica, err := s.politicas.ObtenerPoliticaCotejo(ctx, strings.TrimSpace(orden.PoliticaID), orden.PoliticaVersion)
	if err != nil {
		return ResultadoReservaCodigoCotejo{}, err
	}
	if !politica.Admite(documento) {
		return ResultadoReservaCodigoCotejo{}, domain.ErrDocumentoNoAdmitidoPorCotejo
	}
	aplicacionPolitica, err := politica.Aplicacion()
	if err != nil {
		return ResultadoReservaCodigoCotejo{}, err
	}
	if !orden.Principal.AuthAssurance.Cumple(aplicacionPolitica.GarantiaMinima) {
		return ResultadoReservaCodigoCotejo{}, domain.ErrGarantiaInsuficiente
	}
	expedienteRef, err := referenciaExpedientePrincipal(documento.Relaciones)
	if err != nil {
		return ResultadoReservaCodigoCotejo{}, err
	}
	decision, err := exigirDecisionAutorizacion(ctx, s.autorizador, s.reloj,
		orden.Principal, orden.PerfilActivo, AccionReservarCodigoCotejo,
		recursoDocumentoParaCotejo(documento, expedienteRef, aplicacionPolitica.Referencia),
		orden.Finalidad, orden.CorrelacionRef, orden.Motivo, usoCamposDecisionNoAplicables)
	if err != nil {
		return ResultadoReservaCodigoCotejo{}, err
	}
	huellaSolicitud, err := s.selladorSolicitud.SellarSolicitudCotejo(ctx,
		solicitudReservaCodigoCotejoCanonica(orden, documento, expedienteRef, aplicacionPolitica.Referencia))
	if err != nil {
		return ResultadoReservaCodigoCotejo{}, fmt.Errorf("sellar solicitud de cotejo: %w", err)
	}
	if !huellaHMACCotejoValida(huellaSolicitud) {
		return ResultadoReservaCodigoCotejo{}, ErrResultadoCotejoInvalido
	}

	instanteReserva := s.reloj.Ahora().UTC()
	if instanteReserva.IsZero() {
		return ResultadoReservaCodigoCotejo{}, ErrResultadoCotejoInvalido
	}
	reserva, err := s.codigos.ReservarEmisionCodigoCotejo(ctx, ports.SolicitudReservarEmisionCodigoCotejo{
		ClaveIdempotencia:   orden.ClaveIdempotencia,
		PrincipalID:         strings.TrimSpace(orden.Principal.ID),
		HuellaSolicitudHMAC: huellaSolicitud,
		Documento:           documento.Referencia(),
		Politica:            aplicacionPolitica.Referencia,
		SolicitadaEn:        instanteReserva,
		ExpiraEn:            instanteReserva.Add(vigenciaReservaOperacion),
	})
	if err != nil {
		return ResultadoReservaCodigoCotejo{}, err
	}
	if reserva.Repetida {
		return s.recuperarReservaCodigoCotejo(ctx, orden, decision, documento, aplicacionPolitica, reserva)
	}
	if strings.TrimSpace(reserva.Token) == "" {
		return ResultadoReservaCodigoCotejo{}, ports.ErrReservaCodigoCotejoNoValida
	}

	confirmada := false
	var custodia ports.CustodiaCodigoCotejo
	defer func() {
		if confirmada {
			return
		}
		ctxLimpieza, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancelar()
		_ = s.codigos.AbandonarReservaCodigoCotejo(ctxLimpieza, reserva.Token)
		// No se reinterpreta la decision interactiva de reserva para borrar en
		// Vault/KMS. Hasta inyectar un worker interno con cuenta privilegiada y
		// una decision PDP exacta de eliminar_huerfano, la custodia queda
		// aislada para reconciliacion y este flujo no puede eliminarla.
	}()

	generado, err := s.generadorValor.GenerarValorCodigoCotejo(ctx)
	if err != nil {
		return ResultadoReservaCodigoCotejo{}, fmt.Errorf("generar codigo de cotejo: %w", err)
	}
	if generado.Secreto.Validar() != nil || generado.EntropiaBits < minimoEntropiaCotejoAplicacion ||
		strings.TrimSpace(generado.VersionGenerador) == "" || generado.VersionGenerador != strings.TrimSpace(generado.VersionGenerador) {
		return ResultadoReservaCodigoCotejo{}, ports.ErrMaterialCodigoCotejoInvalido
	}
	indice, err := s.selladorIndice.SellarIndiceCodigoCotejo(ctx, generado.Secreto)
	if err != nil {
		return ResultadoReservaCodigoCotejo{}, fmt.Errorf("indexar codigo de cotejo: %w", err)
	}
	if !huellaHMACCotejoValida(indice) {
		return ResultadoReservaCodigoCotejo{}, ErrResultadoCotejoInvalido
	}
	codigoID, err := s.generadorID.NuevoIDCodigoCotejo()
	if err != nil {
		return ResultadoReservaCodigoCotejo{}, fmt.Errorf("generar identificador de cotejo: %w", err)
	}
	codigoID = strings.TrimSpace(codigoID)
	if (domain.ReferenciaDocumento{ID: codigoID, Version: 1}).Validar() != nil {
		return ResultadoReservaCodigoCotejo{}, ErrResultadoCotejoInvalido
	}
	codigoRef := "cotejo:" + codigoID
	claveCustodia := "custodia-cotejo:" + codigoID
	recursoProteccion := recursoCustodiaCodigoCotejo(
		codigoRef, documento, expedienteRef, aplicacionPolitica.Referencia,
		ports.AccionProtegerCodigoCotejo,
		map[string]string{"clave_idempotencia": claveCustodia, "indice_codigo_hmac": indice},
	)
	decisionProteccion, err := exigirDecisionAutorizacion(
		ctx, s.autorizador, s.reloj, orden.Principal, orden.PerfilActivo,
		ports.AccionProtegerCodigoCotejo, recursoProteccion, orden.Finalidad,
		orden.CorrelacionRef, orden.Motivo, usoCamposDecisionNoAplicables,
	)
	if err != nil {
		return ResultadoReservaCodigoCotejo{}, err
	}
	if decisionProteccion.DecisionRef == decision.DecisionRef {
		return ResultadoReservaCodigoCotejo{}, errors.Join(domain.ErrAutorizacionDenegada, domain.ErrDecisionAutorizacionInvalida)
	}
	instanteProteccion := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	contextoProteccion, err := ports.NuevoContextoProtegerCodigoCotejo(
		decisionProteccion, recursoProteccion, codigoRef, documento.Clasificacion,
		claveCustodia, indice, instanteProteccion,
	)
	if err != nil {
		return ResultadoReservaCodigoCotejo{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	custodia, err = s.protector.ProtegerCodigoCotejo(ctx, ports.SolicitudProtegerCodigoCotejo{
		Contexto:          contextoProteccion,
		ClaveIdempotencia: claveCustodia,
		Secreto:           generado.Secreto,
		IndiceCodigoHMAC:  indice,
	})
	if err != nil {
		return ResultadoReservaCodigoCotejo{}, fmt.Errorf("proteger codigo de cotejo: %w", err)
	}
	if !custodiaCodigoCotejoValida(custodia) {
		return ResultadoReservaCodigoCotejo{}, ErrResultadoCotejoInvalido
	}

	codigo := domain.CodigoCotejo{
		ID:               codigoID,
		Revision:         1,
		Documento:        documento.Referencia(),
		ModuloID:         documento.ModuloID,
		TipoDocumental:   documento.TipoDocumental,
		Clasificacion:    documento.Clasificacion,
		Organo:           documento.ENI.Organo,
		ExpedienteRef:    expedienteRef,
		IndiceCodigoHMAC: indice,
		ProteccionRef:    custodia.ProteccionRef,
		VersionGenerador: generado.VersionGenerador,
		EntropiaBits:     generado.EntropiaBits,
		Politica:         aplicacionPolitica,
		Estado:           domain.EstadoCodigoCotejoReservado,
		ReservadoPor:     strings.TrimSpace(orden.Principal.ID),
		ReservadoEn:      instanteReserva,
		ReservaExpiraEn:  instanteReserva.AddDate(0, 0, aplicacionPolitica.DiasPlazoActivacion),
		MotivoReserva:    strings.TrimSpace(orden.Motivo),
		CorrelacionRef:   strings.TrimSpace(orden.CorrelacionRef),
	}
	canonico, err := codigo.ClonarCanonico()
	if err != nil {
		return ResultadoReservaCodigoCotejo{}, err
	}
	huellaNueva, err := canonico.HuellaEstadoSHA256()
	if err != nil {
		return ResultadoReservaCodigoCotejo{}, err
	}
	confirmadaEn := s.reloj.Ahora().UTC()
	if confirmadaEn.IsZero() || confirmadaEn.Before(instanteReserva) {
		return ResultadoReservaCodigoCotejo{}, ErrResultadoCotejoInvalido
	}
	traza := trazaCodigoCotejo(orden.Principal, orden.PerfilActivo, orden.RepresentadoRef,
		decision.DecisionRef, orden.Finalidad, eventoCodigoCotejoReservado, canonico,
		orden.Motivo, orden.CorrelacionRef, "", huellaNueva, confirmadaEn)
	traza.Metadata["conector_custodia"] = custodia.ConectorID
	evento := eventoCodigoCotejo(eventoCodigoCotejoReservado, canonico, orden.Principal.ID, huellaNueva, confirmadaEn)
	if err := s.codigos.ConfirmarReservaCodigoCotejo(ctx, reserva.Token, huellaSolicitud,
		confirmadaEn, canonico, traza, evento); err != nil {
		return ResultadoReservaCodigoCotejo{}, err
	}
	confirmada = true
	return ResultadoReservaCodigoCotejo{
		Codigo:  canonico,
		Secreto: generado.Secreto,
	}, nil
}

func (s *ServicioCotejo) recuperarReservaCodigoCotejo(
	ctx context.Context,
	orden OrdenReservarCodigoCotejo,
	decisionReserva domain.DecisionAutorizacion,
	documento domain.DocumentoLogico,
	politica domain.AplicacionPoliticaCotejo,
	reserva ports.ReservaEmisionCodigoCotejo,
) (ResultadoReservaCodigoCotejo, error) {
	if reserva.Token != "" || !reserva.Repetida || reserva.Codigo.Validar() != nil ||
		reserva.Codigo.Documento != documento.Referencia() ||
		!reflect.DeepEqual(reserva.Codigo.Politica, politica) ||
		(reserva.Codigo.Estado != domain.EstadoCodigoCotejoReservado && reserva.Codigo.Estado != domain.EstadoCodigoCotejoActivo) {
		return ResultadoReservaCodigoCotejo{}, ports.ErrReservaCodigoCotejoNoValida
	}
	recursoRecuperacion := recursoCustodiaCodigoCotejo(
		reserva.Codigo.Referencia(), documento, reserva.Codigo.ExpedienteRef,
		politica.Referencia, ports.AccionRecuperarCodigoCotejo,
		map[string]string{
			"proteccion_ref":     reserva.Codigo.ProteccionRef,
			"indice_codigo_hmac": reserva.Codigo.IndiceCodigoHMAC,
		},
	)
	decisionRecuperacion, err := exigirDecisionAutorizacion(
		ctx, s.autorizador, s.reloj, orden.Principal, orden.PerfilActivo,
		ports.AccionRecuperarCodigoCotejo, recursoRecuperacion, orden.Finalidad,
		orden.CorrelacionRef, orden.Motivo, usoCamposDecisionNoAplicables,
	)
	if err != nil {
		return ResultadoReservaCodigoCotejo{}, err
	}
	if decisionRecuperacion.DecisionRef == decisionReserva.DecisionRef {
		return ResultadoReservaCodigoCotejo{}, errors.Join(domain.ErrAutorizacionDenegada, domain.ErrDecisionAutorizacionInvalida)
	}
	contexto, err := ports.NuevoContextoRecuperarCodigoCotejo(
		decisionRecuperacion, recursoRecuperacion, reserva.Codigo.Referencia(),
		documento.Clasificacion, reserva.Codigo.ProteccionRef,
		reserva.Codigo.IndiceCodigoHMAC, s.reloj.Ahora().UTC().Truncate(time.Microsecond),
	)
	if err != nil {
		return ResultadoReservaCodigoCotejo{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	recuperacion, err := s.protector.RecuperarCodigoCotejo(ctx, ports.SolicitudRecuperarCodigoCotejo{
		Contexto:                 contexto,
		ProteccionRef:            reserva.Codigo.ProteccionRef,
		IndiceCodigoHMACEsperado: reserva.Codigo.IndiceCodigoHMAC,
	})
	if err != nil {
		return ResultadoReservaCodigoCotejo{}, fmt.Errorf("recuperar codigo de cotejo: %w", err)
	}
	if recuperacion.Secreto.Validar() != nil || strings.TrimSpace(recuperacion.ConectorID) == "" ||
		strings.TrimSpace(recuperacion.EvidenciaRef) == "" {
		return ResultadoReservaCodigoCotejo{}, ErrResultadoCotejoInvalido
	}
	indices, err := s.selladorIndice.SellarIndicesConsultaCodigoCotejo(ctx, recuperacion.Secreto)
	if err != nil {
		return ResultadoReservaCodigoCotejo{}, fmt.Errorf("verificar codigo recuperado: %w", err)
	}
	if !contieneHuellaCotejoConstante(indices, reserva.Codigo.IndiceCodigoHMAC) {
		return ResultadoReservaCodigoCotejo{}, errors.Join(ErrResultadoCotejoInvalido, ports.ErrValorCodigoCotejoNoDisponible)
	}
	return ResultadoReservaCodigoCotejo{
		Codigo:   reserva.Codigo,
		Secreto:  recuperacion.Secreto,
		Repetida: true,
	}, nil
}

func estadoDocumentoPermiteReservaCotejo(estado domain.EstadoDocumentoLogico) bool {
	switch estado {
	case domain.EstadoDocumentoLogicoBorrador, domain.EstadoDocumentoLogicoEnRevision,
		domain.EstadoDocumentoLogicoCerrado, domain.EstadoDocumentoLogicoPendienteFirma:
		return true
	default:
		return false
	}
}

func recursoDocumentoParaCotejo(
	documento domain.DocumentoLogico,
	expedienteRef string,
	politica domain.ReferenciaPoliticaCotejo,
) domain.RecursoAutorizable {
	return domain.RecursoAutorizable{
		Referencia: referenciaDocumentoCotejo(documento.Referencia()),
		ModuloID:   documento.ModuloID,
		Tipo:       "documento",
		Ambitos: map[string]string{
			"expediente":    expedienteRef,
			"clasificacion": documento.Clasificacion,
		},
		Atributos: map[string]string{
			"tipo_documental": documento.TipoDocumental,
			"politica_ref":    politica.ID + ":" + strconv.Itoa(politica.Version),
		},
	}
}

func solicitudReservaCodigoCotejoCanonica(
	orden OrdenReservarCodigoCotejo,
	documento domain.DocumentoLogico,
	expedienteRef string,
	politica domain.ReferenciaPoliticaCotejo,
) []byte {
	var salida strings.Builder
	escribirCampoCanonico(&salida, "esquema", "reserva-codigo-cotejo-v1")
	escribirCampoCanonico(&salida, "principal_id", strings.TrimSpace(orden.Principal.ID))
	escribirCampoCanonico(&salida, "perfil_activo", strings.TrimSpace(orden.PerfilActivo))
	escribirCampoCanonico(&salida, "representado_ref", strings.TrimSpace(orden.RepresentadoRef))
	escribirCampoCanonico(&salida, "metodo_autenticacion", string(orden.Principal.AuthMethod))
	escribirCampoCanonico(&salida, "garantia_autenticacion", string(orden.Principal.AuthAssurance))
	escribirCampoCanonico(&salida, "finalidad", strings.TrimSpace(orden.Finalidad))
	escribirCampoCanonico(&salida, "documento_id", documento.ID)
	escribirCampoCanonico(&salida, "documento_version", strconv.Itoa(documento.Version))
	escribirCampoCanonico(&salida, "huella_fuente", documento.HuellaFuenteHMAC)
	escribirCampoCanonico(&salida, "modulo_id", documento.ModuloID)
	escribirCampoCanonico(&salida, "tipo_documental", documento.TipoDocumental)
	escribirCampoCanonico(&salida, "clasificacion", documento.Clasificacion)
	escribirCampoCanonico(&salida, "expediente_ref", expedienteRef)
	escribirCampoCanonico(&salida, "politica_id", politica.ID)
	escribirCampoCanonico(&salida, "politica_version", strconv.Itoa(politica.Version))
	escribirCampoCanonico(&salida, "politica_huella", politica.HuellaSHA256)
	escribirCampoCanonico(&salida, "motivo", strings.TrimSpace(orden.Motivo))
	escribirCampoCanonico(&salida, "correlacion_ref", strings.TrimSpace(orden.CorrelacionRef))
	return []byte(salida.String())
}

func recursoCustodiaCodigoCotejo(
	codigoRef string,
	documento domain.DocumentoLogico,
	expedienteRef string,
	politica domain.ReferenciaPoliticaCotejo,
	accion string,
	atributosOperacion map[string]string,
) domain.RecursoAutorizable {
	atributos := map[string]string{
		"operacion_custodia": accion,
		"documento_ref":      referenciaDocumentoCotejo(documento.Referencia()),
		"politica_ref":       politica.ID + ":" + strconv.Itoa(politica.Version),
	}
	for clave, valor := range atributosOperacion {
		atributos[clave] = valor
	}
	return domain.RecursoAutorizable{
		Referencia: strings.TrimSpace(codigoRef),
		ModuloID:   documento.ModuloID,
		Tipo:       "codigo_cotejo",
		Ambitos: map[string]string{
			"expediente":    expedienteRef,
			"clasificacion": documento.Clasificacion,
		},
		Atributos: atributos,
	}
}

func custodiaCodigoCotejoValida(custodia ports.CustodiaCodigoCotejo) bool {
	return strings.TrimSpace(custodia.ProteccionRef) != "" && custodia.ProteccionRef == strings.TrimSpace(custodia.ProteccionRef) &&
		strings.TrimSpace(custodia.ConectorID) != "" && custodia.ConectorID == strings.TrimSpace(custodia.ConectorID) &&
		strings.TrimSpace(custodia.EvidenciaRef) != "" && custodia.EvidenciaRef == strings.TrimSpace(custodia.EvidenciaRef)
}

func huellaHMACCotejoValida(valor string) bool {
	partes := strings.Split(valor, ":")
	if len(partes) != 3 || partes[0] != "hmac-sha256" || !claveHMACCotejoValida.MatchString(partes[1]) || len(partes[2]) != 64 {
		return false
	}
	for _, caracter := range partes[2] {
		if (caracter < '0' || caracter > '9') && (caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}

var claveHMACCotejoValida = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,127}$`)

func contieneHuellaCotejoConstante(indices []string, esperada string) bool {
	if len(indices) == 0 || len(indices) > 16 || !huellaHMACCotejoValida(esperada) {
		return false
	}
	canonicos := append([]string(nil), indices...)
	sort.Strings(canonicos)
	coincidencias := 0
	for indice, candidata := range canonicos {
		if !huellaHMACCotejoValida(candidata) || (indice > 0 && candidata == canonicos[indice-1]) {
			return false
		}
		if len(candidata) == len(esperada) {
			coincidencias += subtle.ConstantTimeCompare([]byte(candidata), []byte(esperada))
		}
	}
	return coincidencias == 1
}

func trazaCodigoCotejo(
	principal domain.Principal,
	perfilActivo, representadoRef, autorizacionRef, finalidad, accion string,
	codigo domain.CodigoCotejo,
	motivo, correlacionRef, huellaAnterior, huellaNueva string,
	instante time.Time,
) domain.AuditEntry {
	return domain.AuditEntry{
		ActorID:              strings.TrimSpace(principal.ID),
		ActorProfile:         strings.TrimSpace(perfilActivo),
		ActorRoles:           append([]string(nil), principal.Roles...),
		RepresentedSubjectID: strings.TrimSpace(representadoRef),
		AuthMethod:           principal.AuthMethod,
		AuthAssurance:        principal.AuthAssurance,
		AuthorizationRef:     strings.TrimSpace(autorizacionRef),
		Purpose:              strings.TrimSpace(finalidad),
		Action:               accion,
		ModuleID:             codigo.ModuloID,
		SubjectRef:           codigo.Referencia(),
		ObjectVersion:        codigo.Revision,
		ExpedienteRef:        codigo.ExpedienteRef,
		DocumentRef:          referenciaDocumentoCotejo(codigo.Documento),
		RuleRef:              codigo.Politica.Referencia.ID + ":" + strconv.Itoa(codigo.Politica.Referencia.Version),
		Reason:               strings.TrimSpace(motivo),
		Result:               "correcto",
		BeforeHash:           huellaAnterior,
		AfterHash:            huellaNueva,
		CorrelationRef:       strings.TrimSpace(correlacionRef),
		Metadata: map[string]string{
			"estado":            string(codigo.Estado),
			"clasificacion":     codigo.Clasificacion,
			"tipo_documental":   codigo.TipoDocumental,
			"politica_huella":   codigo.Politica.Referencia.HuellaSHA256,
			"entropia_bits":     strconv.Itoa(codigo.EntropiaBits),
			"version_generador": codigo.VersionGenerador,
		},
		OccurredAt: instante.UTC(),
	}
}

func eventoCodigoCotejo(tipo string, codigo domain.CodigoCotejo, actorID, huella string, instante time.Time) domain.Event {
	return domain.Event{
		Type:       tipo,
		ModuleID:   codigo.ModuloID,
		SubjectRef: codigo.Referencia(),
		ActorID:    strings.TrimSpace(actorID),
		OccurredAt: instante.UTC(),
		Payload: map[string]string{
			"codigo_ref":       codigo.Referencia(),
			"documento_ref":    referenciaDocumentoCotejo(codigo.Documento),
			"revision":         strconv.Itoa(codigo.Revision),
			"estado":           string(codigo.Estado),
			"huella_estado":    huella,
			"politica_id":      codigo.Politica.Referencia.ID,
			"politica_version": strconv.Itoa(codigo.Politica.Referencia.Version),
		},
	}
}

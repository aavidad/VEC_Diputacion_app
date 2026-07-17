package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

const (
	esquemaResultadoConsultaAutoridadV1 = "vec.fuentes-autoridad.consulta-resultado.v1"
	esquemaEntradaAuditoriaConsultaV1   = "vec.fuentes-autoridad.auditoria-consulta.v1"
	esquemaCompromisoReciboConsultaV1   = "vec.fuentes-autoridad.recibo-consulta.v1"
	formatoInstanteAuditoriaConsultaV1  = "2006-01-02T15:04:05.000000Z"
)

// DatosReciboConsultaInternaFuenteAutoridad es una proyeccion interna. La
// entrada confirmada queda minimizada y su copia defensiva permite comprobar
// el registro firmado que ya existe tras el COMMIT, no solo una referencia.
type DatosReciboConsultaInternaFuenteAutoridad struct {
	bloqueoSerializacionGobiernoFuenteAutoridad
	TransaccionRef                 string
	Selector                       SelectorVersionFuenteAutoridad
	Resultado                      ResultadoConsultaFuenteAutoridad
	Estado                         ReferenciaEstadoFuenteAutoridad
	DecisionRef                    string
	HuellaDecisionSHA256           string
	AuditoriaRef                   string
	AuditoriaSecuencia             int64
	AuditoriaAlgoritmoIntegridad   string
	AuditoriaEncadenadoAnteriorRef string
	AuditoriaFirmaRef              string
	AuditoriaConfirmada            domain.AuditEntry
	AuditoriaHuellaEntradaSHA256   string
	HuellaCompromisoReciboSHA256   string
	ConfirmadaEn                   time.Time
}

// ReciboConsultaInternaFuenteAutoridad acredita que la decision se consumio,
// la auditoria firmada y encadenada se registro y, si existia, que snapshot
// exacto se leyo. Solo puede salir de la barrera despues de su COMMIT.
type ReciboConsultaInternaFuenteAutoridad struct {
	bloqueoSerializacionGobiernoFuenteAutoridad
	datos *DatosReciboConsultaInternaFuenteAutoridad
}

// PrepararAuditoriaResultadoConsultaInternaFuenteAutoridad fija el outcome y
// el snapshot exacto en la entrada que la barrera debe encadenar y persistir.
// ID, secuencia y envolvente de integridad siguen vacios hasta el registro.
func PrepararAuditoriaResultadoConsultaInternaFuenteAutoridad(
	solicitud SolicitudConsultaInternaGobernadaFuenteAutoridad,
	resultado ResultadoConsultaFuenteAutoridad,
	estado ReferenciaEstadoFuenteAutoridad,
) (domain.AuditEntry, error) {
	selector, errSelector := solicitud.Selector()
	autorizacion, errAutorizacion := solicitud.Autorizacion()
	datosAutorizacion, errDatos := autorizacion.Datos()
	auditoria, errAuditoria := solicitud.Auditoria()
	if errSelector != nil || errAutorizacion != nil || errDatos != nil || errAuditoria != nil ||
		validarOutcomeConsultaAutoridad(selector, resultado, estado) != nil {
		return domain.AuditEntry{}, ErrReciboConsultaFuenteAutoridadInvalido
	}
	huellaResultado, err := huellaResultadoConsultaAutoridadV1(
		selector, resultado, estado, datosAutorizacion.HuellaDecisionSHA256,
	)
	if err != nil {
		return domain.AuditEntry{}, ErrReciboConsultaFuenteAutoridadInvalido
	}
	auditoria.Result = string(resultado)
	auditoria.AfterHash = huellaResultado
	return clonarAuditoriaFuenteAutoridad(auditoria), nil
}

func NuevoReciboConsultaInternaFuenteAutoridad(
	solicitud SolicitudConsultaInternaGobernadaFuenteAutoridad,
	datos DatosReciboConsultaInternaFuenteAutoridad,
) (ReciboConsultaInternaFuenteAutoridad, error) {
	// Las huellas son calculadas por este puerto sobre DTO V1 congeladas; una
	// huella declarada por el adaptador nunca se acepta como autoridad.
	if datos.AuditoriaHuellaEntradaSHA256 != "" || datos.HuellaCompromisoReciboSHA256 != "" {
		return ReciboConsultaInternaFuenteAutoridad{}, ErrReciboConsultaFuenteAutoridadInvalido
	}
	huellaAuditoria, err := huellaEntradaAuditoriaConsultaV1(datos.AuditoriaConfirmada)
	if err != nil {
		return ReciboConsultaInternaFuenteAutoridad{}, ErrReciboConsultaFuenteAutoridadInvalido
	}
	datos.AuditoriaHuellaEntradaSHA256 = huellaAuditoria
	huellaCompromiso, err := huellaCompromisoReciboConsultaV1(datos, huellaAuditoria)
	if err != nil {
		return ReciboConsultaInternaFuenteAutoridad{}, ErrReciboConsultaFuenteAutoridadInvalido
	}
	datos.HuellaCompromisoReciboSHA256 = huellaCompromiso
	if validarDatosReciboConsultaAutoridad(solicitud, datos) != nil {
		return ReciboConsultaInternaFuenteAutoridad{}, ErrReciboConsultaFuenteAutoridadInvalido
	}
	copia := clonarDatosReciboConsultaAutoridad(datos)
	return ReciboConsultaInternaFuenteAutoridad{datos: &copia}, nil
}

func (r ReciboConsultaInternaFuenteAutoridad) Datos() (
	DatosReciboConsultaInternaFuenteAutoridad,
	error,
) {
	if r.datos == nil || validarHuellaPropiaReciboConsultaAutoridad(*r.datos) != nil {
		return DatosReciboConsultaInternaFuenteAutoridad{}, ErrReciboConsultaFuenteAutoridadInvalido
	}
	return clonarDatosReciboConsultaAutoridad(*r.datos), nil
}

func (r ReciboConsultaInternaFuenteAutoridad) ValidarPara(
	solicitud SolicitudConsultaInternaGobernadaFuenteAutoridad,
) error {
	datos, err := r.Datos()
	if err != nil || validarDatosReciboConsultaAutoridad(solicitud, datos) != nil {
		return ErrReciboConsultaFuenteAutoridadInvalido
	}
	return nil
}

func validarDatosReciboConsultaAutoridad(
	solicitud SolicitudConsultaInternaGobernadaFuenteAutoridad,
	datos DatosReciboConsultaInternaFuenteAutoridad,
) error {
	selector, errSelector := solicitud.Selector()
	autorizacion, errAutorizacion := solicitud.Autorizacion()
	datosAutorizacion, errDatos := autorizacion.Datos()
	solicitadaEn, errInstante := solicitud.SolicitadaEn()
	if errSelector != nil || errAutorizacion != nil || errDatos != nil || errInstante != nil ||
		!referenciaPuertoAutoridadValida(datos.TransaccionRef) || datos.Selector != selector ||
		validarOutcomeConsultaAutoridad(selector, datos.Resultado, datos.Estado) != nil ||
		datos.DecisionRef != datosAutorizacion.Decision.DecisionRef ||
		datos.HuellaDecisionSHA256 != datosAutorizacion.HuellaDecisionSHA256 ||
		!instantePuertoAutoridadCanonico(datos.ConfirmadaEn) || !datos.ConfirmadaEn.After(solicitadaEn) ||
		autorizacion.ValidarEn(datos.ConfirmadaEn) != nil ||
		validarAuditoriaConfirmadaConsultaAutoridad(solicitud, datos) != nil ||
		validarHuellaPropiaReciboConsultaAutoridad(datos) != nil {
		return ErrReciboConsultaFuenteAutoridadInvalido
	}
	return nil
}

func validarOutcomeConsultaAutoridad(
	selector SelectorVersionFuenteAutoridad,
	resultado ResultadoConsultaFuenteAutoridad,
	estado ReferenciaEstadoFuenteAutoridad,
) error {
	if selector.Validar() != nil || !resultado.Valido() {
		return ErrReciboConsultaFuenteAutoridadInvalido
	}
	if resultado == ResultadoConsultaFuenteEncontrada {
		if estado.Validar() != nil || estado.Fuente.FuenteID != selector.FuenteID ||
			estado.Fuente.Version != selector.Version {
			return ErrReciboConsultaFuenteAutoridadInvalido
		}
		return nil
	}
	if estado != (ReferenciaEstadoFuenteAutoridad{}) {
		return ErrReciboConsultaFuenteAutoridadInvalido
	}
	return nil
}

func validarAuditoriaConfirmadaConsultaAutoridad(
	solicitud SolicitudConsultaInternaGobernadaFuenteAutoridad,
	datos DatosReciboConsultaInternaFuenteAutoridad,
) error {
	esperada, err := PrepararAuditoriaResultadoConsultaInternaFuenteAutoridad(
		solicitud, datos.Resultado, datos.Estado,
	)
	confirmada := clonarAuditoriaFuenteAutoridad(datos.AuditoriaConfirmada)
	if err != nil || validarEnvolventeAuditoriaConsultaAutoridad(datos) != nil {
		return ErrReciboConsultaFuenteAutoridadInvalido
	}
	confirmada.ID, confirmada.Seq = "", 0
	confirmada.IntegrityAlgorithm, confirmada.PrevSignature, confirmada.Signature = "", "", ""
	if !auditoriasConsultaAutoridadIguales(confirmada, esperada) {
		return ErrReciboConsultaFuenteAutoridadInvalido
	}
	return nil
}

func validarEnvolventeAuditoriaConsultaAutoridad(
	datos DatosReciboConsultaInternaFuenteAutoridad,
) error {
	confirmada := datos.AuditoriaConfirmada
	if !referenciaPuertoAutoridadValida(confirmada.ID) || confirmada.Seq <= 0 ||
		!algoritmoIntegridadAuditoriaConsultaValido(confirmada.IntegrityAlgorithm) ||
		!referenciaPuertoAutoridadValida(confirmada.Signature) ||
		(confirmada.Seq == 1 && confirmada.PrevSignature != "") ||
		(confirmada.Seq > 1 && !referenciaPuertoAutoridadValida(confirmada.PrevSignature)) ||
		confirmada.Signature == confirmada.PrevSignature ||
		datos.AuditoriaRef != confirmada.ID || datos.AuditoriaSecuencia != confirmada.Seq ||
		datos.AuditoriaAlgoritmoIntegridad != confirmada.IntegrityAlgorithm ||
		datos.AuditoriaEncadenadoAnteriorRef != confirmada.PrevSignature ||
		datos.AuditoriaFirmaRef != confirmada.Signature ||
		!referenciasPuertoAutoridadDistintas(datos.TransaccionRef, datos.DecisionRef, datos.AuditoriaRef) {
		return ErrReciboConsultaFuenteAutoridadInvalido
	}
	return nil
}

func validarHuellaPropiaReciboConsultaAutoridad(datos DatosReciboConsultaInternaFuenteAutoridad) error {
	huellaAuditoria, errAuditoria := huellaEntradaAuditoriaConsultaV1(datos.AuditoriaConfirmada)
	huellaCompromiso, errCompromiso := huellaCompromisoReciboConsultaV1(datos, huellaAuditoria)
	if validarOutcomeConsultaAutoridad(datos.Selector, datos.Resultado, datos.Estado) != nil ||
		validarEnvolventeAuditoriaConsultaAutoridad(datos) != nil ||
		!huellaSHA256PuertoAutoridadValida(datos.HuellaDecisionSHA256) ||
		!instantePuertoAutoridadCanonico(datos.ConfirmadaEn) ||
		errAuditoria != nil || errCompromiso != nil ||
		datos.AuditoriaHuellaEntradaSHA256 != huellaAuditoria ||
		datos.HuellaCompromisoReciboSHA256 != huellaCompromiso ||
		!huellaSHA256PuertoAutoridadValida(huellaAuditoria) ||
		!huellaSHA256PuertoAutoridadValida(huellaCompromiso) {
		return ErrReciboConsultaFuenteAutoridadInvalido
	}
	return nil
}

func (ReciboConsultaInternaFuenteAutoridad) String() string {
	return "[RECIBO-CONSULTA-INTERNA-FUENTE-AUTORIDAD-OPACO]"
}
func (r ReciboConsultaInternaFuenteAutoridad) GoString() string { return r.String() }
func (r ReciboConsultaInternaFuenteAutoridad) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (r ReciboConsultaInternaFuenteAutoridad) LogValue() slog.Value {
	return slog.StringValue(r.String())
}

type metadatoAuditoriaConsultaV1 struct {
	Clave string `json:"clave"`
	Valor string `json:"valor"`
}

// entradaAuditoriaConsultaCanonicaV1 es una DTO congelada. No se serializa el
// AuditEntry vivo y los mapas se convierten antes en listas ordenadas.
type entradaAuditoriaConsultaCanonicaV1 struct {
	Esquema              string                        `json:"esquema"`
	ID                   string                        `json:"id"`
	Secuencia            int64                         `json:"secuencia"`
	ActorID              string                        `json:"actor_id"`
	ActorProfile         string                        `json:"actor_profile"`
	ActorRoles           []string                      `json:"actor_roles"`
	RepresentedSubjectID string                        `json:"represented_subject_id"`
	AuthMethod           string                        `json:"auth_method"`
	AuthAssurance        string                        `json:"auth_assurance"`
	AuthorizationRef     string                        `json:"authorization_ref"`
	Purpose              string                        `json:"purpose"`
	Action               string                        `json:"action"`
	ModuleID             string                        `json:"module_id"`
	SubjectRef           string                        `json:"subject_ref"`
	ObjectVersion        int                           `json:"object_version"`
	ExpedienteRef        string                        `json:"expediente_ref"`
	DocumentRef          string                        `json:"document_ref"`
	RuleRef              string                        `json:"rule_ref"`
	Reason               string                        `json:"reason"`
	Result               string                        `json:"result"`
	BeforeHash           string                        `json:"before_hash"`
	AfterHash            string                        `json:"after_hash"`
	CorrelationRef       string                        `json:"correlation_ref"`
	Metadata             []metadatoAuditoriaConsultaV1 `json:"metadata"`
	OccurredAt           string                        `json:"occurred_at"`
	IntegrityAlgorithm   string                        `json:"integrity_algorithm"`
	PrevSignature        string                        `json:"prev_signature"`
	Signature            string                        `json:"signature"`
}

type estadoResultadoConsultaCanonicoV1 struct {
	FuenteID             string `json:"fuente_id"`
	Version              uint64 `json:"version"`
	HuellaContenido      string `json:"huella_contenido_sha256"`
	Revision             uint64 `json:"revision"`
	Estado               string `json:"estado"`
	HuellaHistoriaSHA256 string `json:"huella_historia_sha256"`
	HuellaEstadoSHA256   string `json:"huella_estado_sha256"`
}

type resultadoConsultaAutoridadCanonicoV1 struct {
	Esquema              string                            `json:"esquema"`
	SelectorFuenteID     string                            `json:"selector_fuente_id"`
	SelectorVersion      uint64                            `json:"selector_version"`
	Resultado            string                            `json:"resultado"`
	Estado               estadoResultadoConsultaCanonicoV1 `json:"estado"`
	HuellaDecisionSHA256 string                            `json:"huella_decision_sha256"`
}

type compromisoReciboConsultaCanonicoV1 struct {
	Esquema                      string                            `json:"esquema"`
	TransaccionRef               string                            `json:"transaccion_ref"`
	SelectorFuenteID             string                            `json:"selector_fuente_id"`
	SelectorVersion              uint64                            `json:"selector_version"`
	Resultado                    string                            `json:"resultado"`
	Estado                       estadoResultadoConsultaCanonicoV1 `json:"estado"`
	DecisionRef                  string                            `json:"decision_ref"`
	HuellaDecisionSHA256         string                            `json:"huella_decision_sha256"`
	AuditoriaRef                 string                            `json:"auditoria_ref"`
	AuditoriaSecuencia           int64                             `json:"auditoria_secuencia"`
	AuditoriaHuellaEntradaSHA256 string                            `json:"auditoria_huella_entrada_sha256"`
	ConfirmadaEn                 string                            `json:"confirmada_en"`
}

func huellaEntradaAuditoriaConsultaV1(entrada domain.AuditEntry) (string, error) {
	if !instantePuertoAutoridadCanonico(entrada.OccurredAt) {
		return "", ErrReciboConsultaFuenteAutoridadInvalido
	}
	claves := make([]string, 0, len(entrada.Metadata))
	for clave := range entrada.Metadata {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	metadatos := make([]metadatoAuditoriaConsultaV1, 0, len(claves))
	for _, clave := range claves {
		metadatos = append(metadatos, metadatoAuditoriaConsultaV1{Clave: clave, Valor: entrada.Metadata[clave]})
	}
	roles := append([]string(nil), entrada.ActorRoles...)
	sort.Strings(roles)
	dto := entradaAuditoriaConsultaCanonicaV1{
		Esquema: esquemaEntradaAuditoriaConsultaV1, ID: entrada.ID, Secuencia: entrada.Seq,
		ActorID: entrada.ActorID, ActorProfile: entrada.ActorProfile,
		ActorRoles: roles, RepresentedSubjectID: entrada.RepresentedSubjectID,
		AuthMethod: string(entrada.AuthMethod), AuthAssurance: string(entrada.AuthAssurance),
		AuthorizationRef: entrada.AuthorizationRef, Purpose: entrada.Purpose,
		Action: entrada.Action, ModuleID: entrada.ModuleID, SubjectRef: entrada.SubjectRef,
		ObjectVersion: entrada.ObjectVersion, ExpedienteRef: entrada.ExpedienteRef,
		DocumentRef: entrada.DocumentRef, RuleRef: entrada.RuleRef,
		Reason: entrada.Reason, Result: entrada.Result,
		BeforeHash: entrada.BeforeHash, AfterHash: entrada.AfterHash,
		CorrelationRef: entrada.CorrelationRef, Metadata: metadatos,
		OccurredAt:         entrada.OccurredAt.Format(formatoInstanteAuditoriaConsultaV1),
		IntegrityAlgorithm: entrada.IntegrityAlgorithm,
		PrevSignature:      entrada.PrevSignature, Signature: entrada.Signature,
	}
	return huellaDTOConsultaAutoridadV1(dto)
}

func algoritmoIntegridadAuditoriaConsultaValido(valor string) bool {
	if valor == "" || len(valor) > 128 {
		return false
	}
	tieneLetra := false
	for indice := 0; indice < len(valor); indice++ {
		caracter := valor[indice]
		if caracter >= 'a' && caracter <= 'z' {
			tieneLetra = true
			continue
		}
		if caracter >= '0' && caracter <= '9' {
			continue
		}
		switch caracter {
		case '-', '_', '.', ':', '/', '+':
			continue
		default:
			return false
		}
	}
	return tieneLetra
}

func huellaResultadoConsultaAutoridadV1(
	selector SelectorVersionFuenteAutoridad,
	resultado ResultadoConsultaFuenteAutoridad,
	estado ReferenciaEstadoFuenteAutoridad,
	huellaDecision string,
) (string, error) {
	if validarOutcomeConsultaAutoridad(selector, resultado, estado) != nil ||
		!huellaSHA256PuertoAutoridadValida(huellaDecision) {
		return "", ErrReciboConsultaFuenteAutoridadInvalido
	}
	return huellaDTOConsultaAutoridadV1(resultadoConsultaAutoridadCanonicoV1{
		Esquema:          esquemaResultadoConsultaAutoridadV1,
		SelectorFuenteID: selector.FuenteID, SelectorVersion: selector.Version,
		Resultado: string(resultado), Estado: estadoResultadoConsultaV1(estado),
		HuellaDecisionSHA256: huellaDecision,
	})
}

func huellaCompromisoReciboConsultaV1(
	datos DatosReciboConsultaInternaFuenteAutoridad,
	huellaAuditoria string,
) (string, error) {
	if !huellaSHA256PuertoAutoridadValida(datos.HuellaDecisionSHA256) ||
		!huellaSHA256PuertoAutoridadValida(huellaAuditoria) ||
		!instantePuertoAutoridadCanonico(datos.ConfirmadaEn) {
		return "", ErrReciboConsultaFuenteAutoridadInvalido
	}
	return huellaDTOConsultaAutoridadV1(compromisoReciboConsultaCanonicoV1{
		Esquema: esquemaCompromisoReciboConsultaV1, TransaccionRef: datos.TransaccionRef,
		SelectorFuenteID: datos.Selector.FuenteID, SelectorVersion: datos.Selector.Version,
		Resultado: string(datos.Resultado), Estado: estadoResultadoConsultaV1(datos.Estado),
		DecisionRef: datos.DecisionRef, HuellaDecisionSHA256: datos.HuellaDecisionSHA256,
		AuditoriaRef: datos.AuditoriaRef, AuditoriaSecuencia: datos.AuditoriaSecuencia,
		AuditoriaHuellaEntradaSHA256: huellaAuditoria,
		ConfirmadaEn:                 datos.ConfirmadaEn.Format(formatoInstanteAuditoriaConsultaV1),
	})
}

func estadoResultadoConsultaV1(estado ReferenciaEstadoFuenteAutoridad) estadoResultadoConsultaCanonicoV1 {
	return estadoResultadoConsultaCanonicoV1{
		FuenteID: estado.Fuente.FuenteID, Version: estado.Fuente.Version,
		HuellaContenido: estado.Fuente.HuellaContenidoSHA256,
		Revision:        estado.Revision, Estado: string(estado.Estado),
		HuellaHistoriaSHA256: estado.HuellaHistoriaSHA256,
		HuellaEstadoSHA256:   estado.HuellaEstadoSHA256,
	}
}

func huellaDTOConsultaAutoridadV1(dto any) (string, error) {
	canonico, err := json.Marshal(dto)
	if err != nil || len(canonico) == 0 {
		return "", ErrReciboConsultaFuenteAutoridadInvalido
	}
	suma := sha256.Sum256(canonico)
	huella := hex.EncodeToString(suma[:])
	if !huellaSHA256PuertoAutoridadValida(huella) {
		return "", ErrReciboConsultaFuenteAutoridadInvalido
	}
	return huella, nil
}

func auditoriasConsultaAutoridadIguales(primera, segunda domain.AuditEntry) bool {
	return primera.ID == segunda.ID && primera.Seq == segunda.Seq &&
		primera.ActorID == segunda.ActorID && primera.ActorProfile == segunda.ActorProfile &&
		cadenasAutoridadIguales(primera.ActorRoles, segunda.ActorRoles) &&
		primera.RepresentedSubjectID == segunda.RepresentedSubjectID &&
		primera.AuthMethod == segunda.AuthMethod && primera.AuthAssurance == segunda.AuthAssurance &&
		primera.AuthorizationRef == segunda.AuthorizationRef && primera.Purpose == segunda.Purpose &&
		primera.Action == segunda.Action && primera.ModuleID == segunda.ModuleID &&
		primera.SubjectRef == segunda.SubjectRef && primera.ObjectVersion == segunda.ObjectVersion &&
		primera.ExpedienteRef == segunda.ExpedienteRef && primera.DocumentRef == segunda.DocumentRef &&
		primera.RuleRef == segunda.RuleRef && primera.Reason == segunda.Reason &&
		primera.Result == segunda.Result && primera.BeforeHash == segunda.BeforeHash &&
		primera.AfterHash == segunda.AfterHash && primera.CorrelationRef == segunda.CorrelationRef &&
		mapaCadenasAutoridadIgual(primera.Metadata, segunda.Metadata) &&
		primera.OccurredAt.Equal(segunda.OccurredAt) &&
		primera.IntegrityAlgorithm == segunda.IntegrityAlgorithm &&
		primera.PrevSignature == segunda.PrevSignature && primera.Signature == segunda.Signature
}

func clonarDatosReciboConsultaAutoridad(
	datos DatosReciboConsultaInternaFuenteAutoridad,
) DatosReciboConsultaInternaFuenteAutoridad {
	datos.AuditoriaConfirmada = clonarAuditoriaFuenteAutoridad(datos.AuditoriaConfirmada)
	return datos
}

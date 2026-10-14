package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
)

const (
	AccionAnexarOriginalC8  = "cronos.justificacion.anexar"
	AccionRevisarOriginalC8 = "cronos.justificacion.revisar"
)

// MaterialJustificacionOriginalC8 contiene solo referencias comprobadas por
// las fuentes de Cronos y Documentos. Ambos efectos consumen su propia
// decisión V3 y cotejan estos mismos bytes dentro de la transacción.
type MaterialJustificacionOriginalC8 struct {
	ActorRef            string `json:"actor_ref"`
	PerfilRef           string `json:"perfil_ref"`
	EmpleadoRef         string `json:"empleado_ref"`
	EmpleadoObjetivoRef string `json:"empleado_objetivo_ref"`
	SolicitudRef        string `json:"solicitud_ref"`
	ClaveOperacion      string `json:"clave_operacion"`
	VersionEsperada     string `json:"version_esperada"`
	DocumentoID         string `json:"documento_id"`
	DocumentoVersion    string `json:"documento_version"`
	DocumentoSHA256     string `json:"documento_sha256"`
	ReciboDocumentoRef  string `json:"recibo_documento_ref"`
	CustodioID          string `json:"custodio_id"`
	PersonaTitularRef   string `json:"persona_titular_ref"`
	PersonaAportanteRef string `json:"persona_aportante_ref"`
	ContextoRef         string `json:"contexto_ref"`
	Accion              string `json:"accion"`
	Decision            string `json:"decision"`
	MotivoRef           string `json:"motivo_ref"`
	ZonaHoraria         string `json:"zona_horaria"`
}

func (m MaterialJustificacionOriginalC8) Canonico() ([]byte, error) {
	version, err := strconv.ParseInt(m.VersionEsperada, 10, 64)
	if err != nil || version < 0 || strconv.FormatInt(version, 10) != m.VersionEsperada ||
		!referenciaIdentidadMarcaje(m.ActorRef, "per_") || !referenciaIdentidadMarcaje(m.PerfilRef, "prf_") ||
		!referenciaIdentidadMarcaje(m.EmpleadoRef, "emp_") || !referenciaIdentidadMarcaje(m.EmpleadoObjetivoRef, "emp_") ||
		!SolicitudPermisoRefValida(m.SolicitudRef) || !RefDocumentoJustificacionValida(m.ClaveOperacion) ||
		!RefDocumentoJustificacionValida(m.DocumentoID) || !HuellaEfectosValida(m.DocumentoSHA256) ||
		!reciboDocumentalC8.MatchString(m.ReciboDocumentoRef) || !tecnicoJustificacion.MatchString(m.CustodioID) ||
		!referenciaIdentidadMarcaje(m.PersonaTitularRef, "per_") || m.PersonaAportanteRef != m.PersonaTitularRef ||
		!referenciaIdentidadMarcaje(m.ContextoRef, "rca_") ||
		(m.ZonaHoraria != "Europe/Madrid" && m.ZonaHoraria != "Atlantic/Canary") {
		return nil, ErrJustificacionInvalida
	}
	docVersion, err := strconv.ParseUint(m.DocumentoVersion, 10, 64)
	if err != nil || docVersion == 0 || docVersion > 999999999999999999 || strconv.FormatUint(docVersion, 10) != m.DocumentoVersion {
		return nil, ErrJustificacionInvalida
	}
	switch m.Accion {
	case AccionAnexarOriginalC8:
		if m.Decision != "" || m.MotivoRef != "" || m.ActorRef != m.PersonaTitularRef || m.EmpleadoRef != m.EmpleadoObjetivoRef {
			return nil, ErrJustificacionInvalida
		}
	case AccionRevisarOriginalC8:
		if m.ActorRef == m.PersonaTitularRef || m.EmpleadoRef == m.EmpleadoObjetivoRef ||
			(m.Decision != string(JustificacionAceptada) && m.Decision != string(JustificacionRechazada)) ||
			!referenciaMarcaje(m.MotivoRef) {
			return nil, ErrJustificacionInvalida
		}
	default:
		return nil, ErrJustificacionInvalida
	}
	resultado, err := json.Marshal(m)
	if err != nil || len(resultado) > 4096 {
		return nil, ErrJustificacionInvalida
	}
	return resultado, nil
}

func (m MaterialJustificacionOriginalC8) Huella() (string, error) {
	b, err := m.Canonico()
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

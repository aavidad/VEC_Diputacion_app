package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	transaccionbolsa "vec-diputacion-granada/internal/modules/bolsa/internal/transaccion"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const formatoInstanteMicrosegundo = "2006-01-02T15:04:05.000000Z"

type pruebaDecisionPostgreSQL struct {
	EsquemaHuella        string `json:"esquema_huella"`
	DecisionRef          string `json:"decision_ref"`
	HuellaDecisionSHA256 string `json:"huella_decision_sha256"`
	VerificadaEn         string `json:"verificada_en"`
	SujetoRef            string `json:"sujeto_ref"`
}

type contextoRecursoPostgreSQL struct {
	Ambitos   map[string]string `json:"ambitos"`
	Atributos map[string]string `json:"atributos"`
}

func serializarPruebaYRecurso(
	contexto puertosbolsa.ContextoOperacionBaremacion,
	instante time.Time,
	huellaEfecto string,
) ([]byte, []byte, []byte, error) {
	prueba, err := transaccionbolsa.ExtraerPruebaDecisionAutorizacion(
		contexto, instante, huellaEfecto,
	)
	if err != nil {
		return nil, nil, nil, puertosbolsa.ErrAutorizacionBaremacionInvalida
	}
	proyeccion := contexto.Proyeccion()
	pruebaJSON, err := json.Marshal(pruebaDecisionPostgreSQL{
		EsquemaHuella: prueba.EsquemaHuella, DecisionRef: prueba.Uso.DecisionRef,
		HuellaDecisionSHA256: prueba.Uso.HuellaDecisionSHA256,
		VerificadaEn:         prueba.VerificadaEn.UTC().Format(formatoInstanteMicrosegundo),
		SujetoRef:            proyeccion.SujetoRef,
	})
	if err != nil {
		return nil, nil, nil, puertosbolsa.ErrAutorizacionBaremacionInvalida
	}
	recursoJSON, err := json.Marshal(contextoRecursoPostgreSQL{
		Ambitos:   map[string]string{"sujeto_ref": proyeccion.SujetoRef},
		Atributos: map[string]string{},
	})
	if err != nil {
		return nil, nil, nil, puertosbolsa.ErrAutorizacionBaremacionInvalida
	}
	huellaRecurso := sha256.Sum256(recursoJSON)
	if hex.EncodeToString(huellaRecurso[:]) != prueba.Decision.ContextoRecursoHuellaSHA256 {
		return nil, nil, nil, puertosbolsa.ErrAutorizacionBaremacionInvalida
	}
	return pruebaJSON, append([]byte(nil), prueba.RepresentacionCanonica...), recursoJSON, nil
}

func decodificarJSONEstricto(contenido []byte, destino any) error {
	if len(contenido) == 0 || destino == nil {
		return puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	var resto any
	if err := decodificador.Decode(&resto); !errors.Is(err, io.EOF) {
		return puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	return nil
}

func construirVersion(
	baremacionRef, numeroTexto, huella string,
	agregadoCanonico []byte,
	confirmadaEn time.Time,
) (puertosbolsa.VersionBaremacion, error) {
	numero, err := parsearUint64Positivo(numeroTexto)
	if err != nil || confirmadaEn.IsZero() {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	var agregado dominiobolsa.BaremacionMerito
	if decodificarJSONEstricto(agregadoCanonico, &agregado) != nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	version := puertosbolsa.VersionBaremacion{
		Referencia: puertosbolsa.ReferenciaVersionBaremacion{
			BaremacionMeritoRef: baremacionRef,
			Numero:              numero,
			HuellaEstadoSHA256:  huella,
		},
		Agregado: agregado, ConfirmadaEn: confirmadaEn.UTC(),
	}
	if version.Validar() != nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	return version.Clonar()
}

func parsearUint64Positivo(valor string) (uint64, error) {
	if valor == "" || valor[0] == '0' {
		return 0, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	numero, err := strconv.ParseUint(valor, 10, 64)
	if err != nil || numero == 0 || strconv.FormatUint(numero, 10) != valor {
		return 0, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	return numero, nil
}

type registroAuditoriaPostgreSQL struct {
	Referencia                     string                                       `json:"referencia"`
	Secuencia                      uint64                                       `json:"secuencia"`
	PrincipalRef                   string                                       `json:"principal_ref"`
	SujetoRef                      string                                       `json:"sujeto_ref"`
	PerfilActorClave               string                                       `json:"perfil_actor_clave"`
	MetodoAutenticacion            dominiovec.AuthMethod                        `json:"metodo_autenticacion"`
	NivelAutenticacion             dominiovec.AuthAssurance                     `json:"nivel_autenticacion"`
	GarantiaMinima                 dominiovec.AuthAssurance                     `json:"garantia_minima"`
	AutenticacionRef               string                                       `json:"autenticacion_ref"`
	AutorizacionRef                string                                       `json:"autorizacion_ref"`
	AccionAutorizada               puertosbolsa.AccionOperacionBaremacion       `json:"accion_autorizada"`
	ClaseRecursoAutorizada         puertosbolsa.ClaseRecursoOperacionBaremacion `json:"clase_recurso_autorizada"`
	RecursoAutorizadoRef           string                                       `json:"recurso_autorizado_ref"`
	CamposPermitidos               []string                                     `json:"campos_permitidos"`
	FinalidadClave                 string                                       `json:"finalidad_clave"`
	CorrelacionRef                 string                                       `json:"correlacion_ref"`
	Modulo                         string                                       `json:"modulo"`
	Accion                         puertosbolsa.AccionAuditoriaBaremacion       `json:"accion"`
	ClaseCambio                    puertosbolsa.ClaseCambioBaremacion           `json:"clase_cambio"`
	ProcesoRef                     string                                       `json:"proceso_ref"`
	SolicitudRef                   string                                       `json:"solicitud_ref"`
	BaremacionMeritoRef            string                                       `json:"baremacion_merito_ref"`
	DecisionRef                    string                                       `json:"decision_ref"`
	ManifiestoProbatorioRef        string                                       `json:"manifiesto_probatorio_ref"`
	HuellaManifiestoSHA256         string                                       `json:"huella_manifiesto_sha256"`
	DocumentoFirmadoCustodiadoRef  string                                       `json:"documento_firmado_custodiado_ref"`
	EvidenciaCustodiaFirmadoRef    string                                       `json:"evidencia_custodia_firmado_ref"`
	EvidenciaRetencionFirmadoRef   string                                       `json:"evidencia_retencion_firmado_ref"`
	VersionAnterior                uint64                                       `json:"version_anterior"`
	VersionNueva                   uint64                                       `json:"version_nueva"`
	HuellaAnteriorSHA256           string                                       `json:"huella_anterior_sha256"`
	HuellaNuevaSHA256              string                                       `json:"huella_nueva_sha256"`
	MotivoClave                    string                                       `json:"motivo_clave"`
	Motivo                         string                                       `json:"motivo"`
	HuellaSolicitudHMAC            string                                       `json:"huella_solicitud_hmac"`
	Resultado                      string                                       `json:"resultado"`
	SolicitadaConfirmacionEn       time.Time                                    `json:"solicitada_confirmacion_en"`
	SolicitadaConfirmacionCanonica string                                       `json:"solicitada_confirmacion_canonica"`
	RegistradaEn                   time.Time                                    `json:"registrada_en"`
	HuellaAnteriorAuditoriaSHA256  string                                       `json:"huella_anterior_auditoria_sha256"`
	HuellaRegistroSHA256           string                                       `json:"huella_registro_sha256"`
}

func (r registroAuditoriaPostgreSQL) dominio() (puertosbolsa.RegistroAuditoriaBaremacion, error) {
	solicitada, err := time.Parse(time.RFC3339Nano, r.SolicitadaConfirmacionCanonica)
	// PostgreSQL materializa timestamptz a microsegundos, pero la preimagen
	// conserva los nanosegundos sellados. Ambas deben representar el mismo
	// instante tras aplicar exactamente la precision del motor.
	if err != nil || !solicitada.Round(time.Microsecond).Equal(r.SolicitadaConfirmacionEn) {
		return puertosbolsa.RegistroAuditoriaBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	resultado := puertosbolsa.RegistroAuditoriaBaremacion{
		Referencia: r.Referencia, Secuencia: r.Secuencia, PrincipalRef: r.PrincipalRef,
		SujetoRef: r.SujetoRef, PerfilActorClave: r.PerfilActorClave,
		MetodoAutenticacion: r.MetodoAutenticacion, NivelAutenticacion: r.NivelAutenticacion,
		GarantiaMinima: r.GarantiaMinima, AutenticacionRef: r.AutenticacionRef,
		AutorizacionRef: r.AutorizacionRef, AccionAutorizada: r.AccionAutorizada,
		ClaseRecursoAutorizada: r.ClaseRecursoAutorizada,
		RecursoAutorizadoRef:   r.RecursoAutorizadoRef,
		CamposPermitidos:       append([]string(nil), r.CamposPermitidos...),
		FinalidadClave:         r.FinalidadClave, CorrelacionRef: r.CorrelacionRef,
		Modulo: r.Modulo, Accion: r.Accion, ClaseCambio: r.ClaseCambio,
		ProcesoRef: r.ProcesoRef, SolicitudRef: r.SolicitudRef,
		BaremacionMeritoRef: r.BaremacionMeritoRef, DecisionRef: r.DecisionRef,
		ManifiestoProbatorioRef:       r.ManifiestoProbatorioRef,
		HuellaManifiestoSHA256:        r.HuellaManifiestoSHA256,
		DocumentoFirmadoCustodiadoRef: r.DocumentoFirmadoCustodiadoRef,
		EvidenciaCustodiaFirmadoRef:   r.EvidenciaCustodiaFirmadoRef,
		EvidenciaRetencionFirmadoRef:  r.EvidenciaRetencionFirmadoRef,
		VersionAnterior:               r.VersionAnterior, VersionNueva: r.VersionNueva,
		HuellaAnteriorSHA256: r.HuellaAnteriorSHA256, HuellaNuevaSHA256: r.HuellaNuevaSHA256,
		MotivoClave: r.MotivoClave, Motivo: r.Motivo, HuellaSolicitudHMAC: r.HuellaSolicitudHMAC,
		Resultado: r.Resultado, SolicitadaConfirmacionEn: solicitada.UTC(),
		RegistradaEn:                  r.RegistradaEn.UTC(),
		HuellaAnteriorAuditoriaSHA256: r.HuellaAnteriorAuditoriaSHA256,
		HuellaRegistroSHA256:          r.HuellaRegistroSHA256,
	}
	if resultado.Validar() != nil || transaccionbolsa.HuellaAuditoria(resultado) != resultado.HuellaRegistroSHA256 {
		return puertosbolsa.RegistroAuditoriaBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	return resultado, nil
}

type eventoOutboxPostgreSQL struct {
	Referencia                   string                                    `json:"referencia"`
	Secuencia                    uint64                                    `json:"secuencia"`
	Tipo                         puertosbolsa.TipoEventoOutboxBaremacion   `json:"tipo"`
	Estado                       puertosbolsa.EstadoEventoOutboxBaremacion `json:"estado"`
	Modulo                       string                                    `json:"modulo"`
	ProcesoRef                   string                                    `json:"proceso_ref"`
	SolicitudRef                 string                                    `json:"solicitud_ref"`
	BaremacionMeritoRef          string                                    `json:"baremacion_merito_ref"`
	DecisionRef                  string                                    `json:"decision_ref"`
	ManifiestoProbatorioRef      string                                    `json:"manifiesto_probatorio_ref"`
	HuellaManifiestoSHA256       string                                    `json:"huella_manifiesto_sha256"`
	DocumentoFirmadoRef          string                                    `json:"documento_firmado_ref"`
	EvidenciaCustodiaFirmadoRef  string                                    `json:"evidencia_custodia_firmado_ref"`
	EvidenciaRetencionFirmadoRef string                                    `json:"evidencia_retencion_firmado_ref"`
	SujetoRef                    string                                    `json:"sujeto_ref"`
	PrincipalRef                 string                                    `json:"principal_ref"`
	VersionNueva                 uint64                                    `json:"version_nueva"`
	HuellaNuevaSHA256            string                                    `json:"huella_nueva_sha256"`
	AuditoriaRef                 string                                    `json:"auditoria_ref"`
	HuellaAuditoriaSHA256        string                                    `json:"huella_auditoria_sha256"`
	CorrelacionRef               string                                    `json:"correlacion_ref"`
	RegistradaEn                 time.Time                                 `json:"registrada_en"`
	HuellaEventoAnteriorSHA256   string                                    `json:"huella_evento_anterior_sha256"`
	HuellaRegistroSHA256         string                                    `json:"huella_registro_sha256"`
}

func (e eventoOutboxPostgreSQL) dominio() (puertosbolsa.EventoOutboxBaremacion, error) {
	resultado := puertosbolsa.EventoOutboxBaremacion{
		Referencia: e.Referencia, Secuencia: e.Secuencia, Tipo: e.Tipo, Estado: e.Estado,
		Modulo: e.Modulo, ProcesoRef: e.ProcesoRef, SolicitudRef: e.SolicitudRef,
		BaremacionMeritoRef: e.BaremacionMeritoRef, DecisionRef: e.DecisionRef,
		ManifiestoProbatorioRef:      e.ManifiestoProbatorioRef,
		HuellaManifiestoSHA256:       e.HuellaManifiestoSHA256,
		DocumentoFirmadoRef:          e.DocumentoFirmadoRef,
		EvidenciaCustodiaFirmadoRef:  e.EvidenciaCustodiaFirmadoRef,
		EvidenciaRetencionFirmadoRef: e.EvidenciaRetencionFirmadoRef,
		SujetoRef:                    e.SujetoRef, PrincipalRef: e.PrincipalRef,
		VersionNueva: e.VersionNueva, HuellaNuevaSHA256: e.HuellaNuevaSHA256,
		AuditoriaRef: e.AuditoriaRef, HuellaAuditoriaSHA256: e.HuellaAuditoriaSHA256,
		CorrelacionRef: e.CorrelacionRef, RegistradoEn: e.RegistradaEn.UTC(),
		HuellaEventoAnteriorSHA256: e.HuellaEventoAnteriorSHA256,
		HuellaRegistroSHA256:       e.HuellaRegistroSHA256,
	}
	if resultado.Validar() != nil || transaccionbolsa.HuellaEvento(resultado) != resultado.HuellaRegistroSHA256 {
		return puertosbolsa.EventoOutboxBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	return resultado, nil
}

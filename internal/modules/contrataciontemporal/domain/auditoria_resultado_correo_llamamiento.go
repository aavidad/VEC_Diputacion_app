package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type DatosAuditoriaResultadoCorreoLlamamiento struct {
	ActorID, ActorProfile, VersionRolRef, AuthMethod, AuthAssurance string
	Correlacion                                                     vecdomain.ReferenciaCorrelacionAutorizacionV2
	Solicitud                                                       SolicitudRegistrarResultadoCorreoLlamamiento
	OcurridoEn                                                      time.Time
}
type AuditoriaResultadoCorreoLlamamiento struct {
	datos DatosAuditoriaResultadoCorreoLlamamiento
}

func NuevaAuditoriaResultadoCorreoLlamamiento(d DatosAuditoriaResultadoCorreoLlamamiento) (AuditoriaResultadoCorreoLlamamiento, error) {
	if d.Solicitud.Validar() != nil || !seudonimoHMACAuditoriaValido(d.ActorID) || !ReferenciaOpacaValida(d.ActorProfile) || !ReferenciaOpacaValida(d.VersionRolRef) || d.Correlacion.Validar() != nil || d.AuthMethod == "" || d.AuthAssurance == "" || !InstanteUTCCanonico(d.OcurridoEn) {
		return AuditoriaResultadoCorreoLlamamiento{}, ErrResultadoCorreoLlamamientoNoConfiable
	}
	return AuditoriaResultadoCorreoLlamamiento{d}, nil
}

func seudonimoHMACAuditoriaValido(valor string) bool {
	partes := strings.Split(valor, ":")
	if len(partes) != 3 || partes[0] != "hmac-sha256" || partes[1] == "" || len(partes[2]) != 64 || strings.ToLower(partes[2]) != partes[2] {
		return false
	}
	_, err := hex.DecodeString(partes[2])
	return err == nil
}
func (a AuditoriaResultadoCorreoLlamamiento) SerializarCanonico() ([]byte, error) {
	if _, e := NuevaAuditoriaResultadoCorreoLlamamiento(a.datos); e != nil {
		return nil, e
	}
	d := a.datos
	correlacion, err := d.Correlacion.ValorCanonico()
	if err != nil {
		return nil, ErrResultadoCorreoLlamamientoNoConfiable
	}
	return json.Marshal(struct {
		ID             string   `json:"id"`
		Seq            uint64   `json:"seq"`
		Signature      string   `json:"signature"`
		ActorID        string   `json:"actor_id"`
		ActorProfile   string   `json:"actor_profile"`
		ActorRoles     []string `json:"actor_roles"`
		AuthMethod     string   `json:"auth_method"`
		AuthAssurance  string   `json:"auth_assurance"`
		Purpose        string   `json:"purpose"`
		Action         string   `json:"action"`
		ModuleID       string   `json:"module_id"`
		SubjectRef     string   `json:"subject_ref"`
		ObjectVersion  uint64   `json:"object_version"`
		ExpedienteRef  string   `json:"expediente_ref"`
		Result         string   `json:"result"`
		CorrelationRef string   `json:"correlation_ref"`
		OccurredAt     string   `json:"occurred_at"`
	}{"", 0, "", d.ActorID, d.ActorProfile, []string{d.VersionRolRef}, d.AuthMethod, d.AuthAssurance, "gestionar_contratacion_temporal", "contratacion_temporal.llamamiento.correo.registrar_resultado", "vec.module.contratacion_temporal", d.Solicitud.IntentoRef, 2, d.Solicitud.ExpedienteRef, "accepted", correlacion, d.OcurridoEn.Format("2006-01-02T15:04:05.000000Z")})
}
func (a AuditoriaResultadoCorreoLlamamiento) HuellaSHA256() (string, error) {
	b, e := a.SerializarCanonico()
	if e != nil {
		return "", e
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

// CorrelacionRef solo expone la referencia que debe quedar comprometida por la
// autorización final. La auditoría sigue siendo el único sitio que conserva el
// resto de la identidad seudonimizada y del contexto atestado.
func (a AuditoriaResultadoCorreoLlamamiento) CorrelacionRef() (string, error) {
	if _, err := NuevaAuditoriaResultadoCorreoLlamamiento(a.datos); err != nil {
		return "", err
	}
	return a.datos.Correlacion.ValorCanonico()
}

// CorrelacionPara transporta la capacidad nominal original, acuñada por la
// autoridad común. Nunca reconstruye una capacidad a partir del JSON Audit17.
func (a AuditoriaResultadoCorreoLlamamiento) CorrelacionPara(s SolicitudRegistrarResultadoCorreoLlamamiento) (vecdomain.ReferenciaCorrelacionAutorizacionV2, error) {
	if _, err := NuevaAuditoriaResultadoCorreoLlamamiento(a.datos); err != nil || a.datos.Solicitud != s {
		return vecdomain.ReferenciaCorrelacionAutorizacionV2{}, ErrResultadoCorreoLlamamientoNoConfiable
	}
	return a.datos.Correlacion, nil
}

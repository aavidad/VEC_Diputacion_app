package main

import (
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
)

type envoltura struct {
	Estado           string           `json:"estado"`
	Codigo           *string          `json:"codigo"`
	Recibo           *reciboUnidad    `json:"recibo"`
	Replay           bool             `json:"replay"`
	AuditoriaIntento auditoriaIntento `json:"auditoria_intento"`
}
type auditoriaIntento struct {
	AuditoriaRef   string `json:"auditoria_ref"`
	Secuencia      uint64 `json:"secuencia"`
	HuellaSHA256   string `json:"huella_sha256"`
	CorrelacionRef string `json:"correlacion_ref"`
	RegistradaEn   string `json:"registrada_en"`
}
type reciboUnidad struct {
	Esquema               string       `json:"esquema"`
	Version               uint64       `json:"version"`
	ReciboRef             string       `json:"recibo_ref"`
	OperacionRef          string       `json:"operacion_ref"`
	PlanSHA256            string       `json:"plan_sha256"`
	FuenteRef             string       `json:"fuente_ref"`
	FuenteVersion         uint64       `json:"fuente_version"`
	FuenteSHA256          string       `json:"fuente_sha256"`
	PreimagenSHA256       string       `json:"preimagen_sha256"`
	ConfiguracionSHA256   string       `json:"configuracion_sha256"`
	AprobacionRef         string       `json:"aprobacion_ref"`
	AlcanceFuente         string       `json:"alcance_fuente"`
	RegistradaEn          string       `json:"registrada_en"`
	Unidad                nodoHistoria `json:"unidad"`
	ReciboSHA256          string       `json:"recibo_sha256"`
	AuditoriaRef          string       `json:"auditoria_ref"`
	AuditoriaSecuencia    uint64       `json:"auditoria_secuencia"`
	AuditoriaHuellaSHA256 string       `json:"auditoria_huella_sha256"`
	AuditoriaRegistradaEn string       `json:"auditoria_registrada_en"`
}

// Columnas exactas de Personal10. Se conserva la fila devuelta por su dueño.
type nodoHistoria struct {
	NodoRef              string            `json:"nodo_ref"`
	Revision             uint64            `json:"revision"`
	OrganismoRef         string            `json:"organismo_ref"`
	UnidadRef            string            `json:"unidad_ref"`
	Clase                string            `json:"clase"`
	CatalogoRef          string            `json:"catalogo_ref"`
	CatalogoVersion      uint64            `json:"catalogo_version"`
	CatalogoRevision     uint64            `json:"catalogo_revision"`
	CatalogoEntradaClave string            `json:"catalogo_entrada_clave"`
	Denominacion         string            `json:"denominacion"`
	CentroPadreRef       *string           `json:"centro_padre_ref"`
	Retirado             bool              `json:"retirado"`
	VigenteDesde         domain.FechaCivil `json:"vigente_desde"`
	VigenteHasta         domain.FechaCivil `json:"vigente_hasta"`
	ConocidoDesde        string            `json:"conocido_desde"`
	FuenteRef            string            `json:"fuente_ref"`
	ActoRef              string            `json:"acto_ref"`
	HuellaFuenteSHA256   string            `json:"huella_fuente_sha256"`
}

var errEnvoltura = errors.New("envoltura_invalida")

func validarEnvoltura(b []byte, p domain.PlanUnidadInicialAdminV1, sha string) (envoltura, error) {
	var e envoltura
	if len(b) == 0 || len(b) > limiteDocumento || decodificarEstricto(b, &e) != nil {
		return e, errEnvoltura
	}
	a := e.AuditoriaIntento
	if !refHex(a.AuditoriaRef, "aud_v3_ui_") || a.Secuencia == 0 || a.Secuencia > 1<<53-1 || !hashValido(a.HuellaSHA256) || !refHex(a.CorrelacionRef, "correlacion_") || !fechaRecibo(a.RegistradaEn) {
		return e, errEnvoltura
	}
	switch e.Estado {
	case "permitido":
		if e.Codigo != nil || e.Recibo == nil || !reciboValido(e.Recibo, p, sha) {
			return e, errEnvoltura
		}
	case "denegado":
		if e.Codigo == nil || *e.Codigo != "unidad_rechazada" || e.Recibo != nil || e.Replay {
			return e, errEnvoltura
		}
	case "error":
		if e.Codigo == nil || *e.Codigo != "unidad_no_disponible" || e.Recibo != nil || e.Replay {
			return e, errEnvoltura
		}
	default:
		return e, errEnvoltura
	}
	return e, nil
}
func reciboValido(r *reciboUnidad, p domain.PlanUnidadInicialAdminV1, sha string) bool {
	if r.Esquema != "vec.personal.unidad-inicial.v1" || r.Version != 1 || !refHex(r.ReciboRef, "recibo_unidad:") || r.OperacionRef != p.OperacionRef || r.PlanSHA256 != sha ||
		r.FuenteRef != p.Fuente.Referencia || r.FuenteVersion != 1 || r.FuenteSHA256 != p.Fuente.HuellaSHA256 || !hashValido(r.PreimagenSHA256) || !hashValido(r.ConfiguracionSHA256) ||
		r.AprobacionRef == "" || len(r.AprobacionRef) > 128 || r.AlcanceFuente != p.AlcanceFuente || !fechaRecibo(r.RegistradaEn) || !hashValido(r.ReciboSHA256) ||
		!refHex(r.AuditoriaRef, "aud_v3_u_") || r.AuditoriaSecuencia == 0 || r.AuditoriaSecuencia > 1<<53-1 || !hashValido(r.AuditoriaHuellaSHA256) || !fechaRecibo(r.AuditoriaRegistradaEn) {
		return false
	}
	u, v := r.Unidad, p.Unidad
	return u.NodoRef == v.NodoRef && u.Revision == 1 && u.OrganismoRef == v.OrganizacionRef && u.UnidadRef == v.UnidadRef && u.Clase == v.Clase && u.CatalogoRef == v.CatalogoRef && u.CatalogoVersion == 1 && u.CatalogoRevision == 1 && u.CatalogoEntradaClave == v.CatalogoEntradaClave && u.Denominacion == v.Denominacion && u.CentroPadreRef == nil && !u.Retirado && u.VigenteDesde == v.VigenteDesde && u.VigenteHasta == v.VigenteHasta && fechaRecibo(u.ConocidoDesde) && u.FuenteRef == p.Fuente.Referencia && u.ActoRef == p.ActoTecnicoRef && u.HuellaFuenteSHA256 == p.Fuente.HuellaSHA256
}
func fechaRecibo(v string) bool {
	t, e := time.Parse(time.RFC3339Nano, v)
	if e != nil || t.IsZero() || t.Year() < 1 || t.Year() > 9999 {
		return false
	}
	_, offset := t.Zone()
	return offset == 0
}
func hashValido(v string) bool {
	if len(v) != 64 {
		return false
	}
	for _, c := range v {
		if !(c >= 'a' && c <= 'f' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}
func refHex(v, p string) bool {
	if !strings.HasPrefix(v, p) || len(v) != len(p)+32 {
		return false
	}
	for _, c := range v[len(p):] {
		if !(c >= 'a' && c <= 'f' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

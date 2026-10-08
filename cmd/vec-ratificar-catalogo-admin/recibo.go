package main

import (
	"errors"
	"strconv"
	"strings"
)

var errRecibo = errors.New("recibo_invalido")

type resultadoRatificacion struct {
	Estado string
	Replay bool
	Acuse  []byte
}

type respuestaRatificacion struct {
	Estado           string              `json:"estado"`
	Codigo           string              `json:"codigo"`
	Recibo           *reciboRatificacion `json:"recibo"`
	Replay           bool                `json:"replay"`
	AuditoriaIntento auditoriaIntento    `json:"auditoria_intento"`
}

type auditoriaIntento struct {
	AuditoriaRef   string `json:"auditoria_ref"`
	Secuencia      uint64 `json:"secuencia"`
	HuellaSHA256   string `json:"huella_sha256"`
	CorrelacionRef string `json:"correlacion_ref"`
	RegistradaEn   string `json:"registrada_en"`
}

type reciboRatificacion struct {
	Esquema            string `json:"esquema"`
	OperacionRef       string `json:"operacion_ref"`
	PlanSHA            string `json:"plan_sha256"`
	PreimagenSHA       string `json:"preimagen_sha256"`
	RolRef             string `json:"rol_ref"`
	RolSHA             string `json:"rol_sha256"`
	ControlRevision    uint64 `json:"control_revision"`
	ControlSHA         string `json:"control_sha256"`
	CatalogoPrevioSHA  string `json:"catalogo_previo_sha256"`
	DescriptoresSHA    string `json:"descriptores_sha256"`
	CatalogoPosterSHA  string `json:"catalogo_posterior_sha256"`
	VigenteDesde       string `json:"vigente_desde"`
	AprobacionRef      string `json:"aprobacion_ref"`
	AprobacionSHA      string `json:"aprobacion_sha256"`
	OperadorLogin      string `json:"operador_login"`
	AuditoriaRef       string `json:"auditoria_ref"`
	AuditoriaSecuencia uint64 `json:"auditoria_secuencia"`
	AuditoriaSHA       string `json:"auditoria_huella_sha256"`
	ConfirmadoEn       string `json:"confirmado_en"`
}

func validarRespuesta(b []byte, p documentoPlan, planSHA, descriptoresSHA string, aprobacion aprobacionPrivada, operador string) (resultadoRatificacion, error) {
	var res respuestaRatificacion
	if len(b) < 1 || len(b) > limiteDocumento || !hashValido(planSHA) ||
		!hashValido(descriptoresSHA) || decodificarEstricto(b, &res) != nil {
		return resultadoRatificacion{}, errRecibo
	}
	a := res.AuditoriaIntento
	if !refHex(a.AuditoriaRef, "aud_v3_rcai_") || a.Secuencia == 0 ||
		!hashValido(a.HuellaSHA256) || !refHex(a.CorrelacionRef, "correlacion_") {
		return resultadoRatificacion{}, errRecibo
	}
	registrada, err := instanteUTC(a.RegistradaEn)
	if err != nil {
		return resultadoRatificacion{}, errors.Join(errRecibo, err)
	}
	switch res.Estado {
	case "denegado":
		if res.Codigo != "ratificacion_rechazada" || res.Recibo != nil || res.Replay {
			return resultadoRatificacion{}, errRecibo
		}
		return resultadoRatificacion{Estado: res.Estado}, nil
	case "error":
		if res.Codigo != "ratificacion_no_disponible" || res.Recibo != nil || res.Replay {
			return resultadoRatificacion{}, errRecibo
		}
		return resultadoRatificacion{Estado: res.Estado}, nil
	case "permitido":
		if res.Recibo == nil || res.Replay && res.Codigo != "ratificacion_replay" ||
			!res.Replay && res.Codigo != "ratificacion_registrada" {
			return resultadoRatificacion{}, errRecibo
		}
	default:
		return resultadoRatificacion{}, errRecibo
	}
	r := res.Recibo
	if r.Esquema != "vec.admin.ratificacion-catalogo.recibo.v1" ||
		r.OperacionRef != p.OperacionRef || r.PlanSHA != planSHA || r.PreimagenSHA != p.PreimagenSHA ||
		r.RolRef != p.RolRef || r.RolSHA != p.RolSHA || r.ControlRevision == 0 ||
		p.ControlRevision.String() != strconv.FormatUint(r.ControlRevision, 10) ||
		r.ControlSHA != p.ControlSHA || r.CatalogoPrevioSHA != p.CatalogoSHA ||
		r.DescriptoresSHA != descriptoresSHA || !hashValido(r.CatalogoPosterSHA) ||
		r.CatalogoPosterSHA == r.CatalogoPrevioSHA || r.AprobacionRef != aprobacion.AprobacionRef ||
		r.AprobacionSHA != aprobacion.AprobacionSHA256 || r.OperadorLogin != operador ||
		r.AuditoriaRef != "aud_v3_rca_"+planSHA[:32] || r.AuditoriaSecuencia == 0 ||
		!hashValido(r.AuditoriaSHA) || strings.TrimSpace(r.OperadorLogin) == "" {
		return resultadoRatificacion{}, errRecibo
	}
	vigente, err := instanteUTC(r.VigenteDesde)
	if err != nil {
		return resultadoRatificacion{}, errors.Join(errRecibo, err)
	}
	confirmado, err := instanteUTC(r.ConfirmadoEn)
	if err != nil {
		return resultadoRatificacion{}, errors.Join(errRecibo, err)
	}
	if confirmado.Before(vigente) || registrada.Before(confirmado) {
		return resultadoRatificacion{}, errRecibo
	}
	return resultadoRatificacion{Estado: res.Estado, Replay: res.Replay, Acuse: b}, nil
}

func refHex(valor, prefijo string) bool {
	if !strings.HasPrefix(valor, prefijo) || len(valor) != len(prefijo)+32 {
		return false
	}
	for _, c := range valor[len(prefijo):] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

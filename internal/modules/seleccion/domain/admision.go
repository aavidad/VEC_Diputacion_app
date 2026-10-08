package domain

import (
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	meritos "vec-diputacion-granada/internal/modules/meritos/domain"
)

const AlcanceAdmisionPreparacion = "preparacion_sintetica"

var ErrAdmisionPreparacion = errors.New("seleccion.admision.entrada_invalida")

type BasesAdmision struct {
	Referencia   string `json:"referencia"`
	Version      string `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

// ContextoSolicitudLocal conserva metadatos de S3, nunca un registro oficial.
type ContextoSolicitudLocal struct {
	HuellaArchivoOriginalSHA256 string `json:"huella_archivo_original_sha256"`
	IdentificadorPublico        string `json:"identificador_publico"`
	Version                     string `json:"version"`
	HuellaSHA256                string `json:"huella_sha256"`
	Estado                      string `json:"estado"`
}

type ReferenciaHechoAdmision struct {
	Referencia string `json:"referencia"`
	Version    int    `json:"version"`
}

type RequisitoAdmision struct {
	TituloPropuesto string                    `json:"titulo_propuesto"`
	Referencia      string                    `json:"referencia"`
	Version         string                    `json:"version"`
	Representacion  string                    `json:"representacion"`
	ReglaRef        string                    `json:"regla_ref,omitempty"`
	ReglaVersion    string                    `json:"regla_version,omitempty"`
	HitoRef         string                    `json:"hito_ref"`
	HitoFecha       string                    `json:"hito_fecha"`
	HechosEsperados []ReferenciaHechoAdmision `json:"hechos_esperados"`
}

// SoporteAdmision minimiza los hechos aportados. Su estado es una afirmación
// del paquete sintético; no acredita correspondencia, documento ni autoridad.
type SoporteAdmision struct {
	ReferenciaHechoAdmision
	FuenteRef           string `json:"fuente_ref"`
	FuenteVersion       string `json:"fuente_version"`
	EstadoAportado      string `json:"estado_aportado"`
	EvidenciasAportadas int    `json:"evidencias_aportadas"`
	Desde               string `json:"desde"`
	Hasta               string `json:"hasta,omitempty"`
}

type RevisionRequisitoAdmision struct {
	Requisito       RequisitoAdmision `json:"requisito"`
	Estado          string            `json:"estado"`
	Causas          []string          `json:"causas"`
	Soportes        []SoporteAdmision `json:"soportes"`
	AccionPropuesta string            `json:"accion_propuesta"`
}

type PreparacionAdmision struct {
	Esquema              string                      `json:"esquema"`
	PreparacionRef       string                      `json:"preparacion_ref"`
	Revision             int                         `json:"revision"`
	Alcance              string                      `json:"alcance"`
	Estado               string                      `json:"estado"`
	UniversoRequisitos   string                      `json:"universo_requisitos"`
	Bases                BasesAdmision               `json:"bases"`
	SolicitudContexto    *ContextoSolicitudLocal     `json:"solicitud_contexto,omitempty"`
	VersionPaqueteHechos string                      `json:"version_paquete_hechos,omitempty"`
	Requisitos           []RevisionRequisitoAdmision `json:"requisitos"`
	Pendientes           []string                    `json:"pendientes"`
	Persistido           bool                        `json:"persistido"`
	AdmisionOficial      bool                        `json:"admision_oficial"`
}

func ValidarBasesAdmision(b BasesAdmision, s *ContextoSolicitudLocal) error {
	if !meritos.ReferenciaValida(b.Referencia) || !meritos.ReferenciaValida(b.Version) || !shaAdmision(b.HuellaSHA256) {
		return ErrAdmisionPreparacion
	}
	if s != nil && (!shaAdmision(s.HuellaArchivoOriginalSHA256) || !meritos.ReferenciaValida(s.IdentificadorPublico) || !meritos.ReferenciaValida(s.Version) ||
		!shaAdmision(s.HuellaSHA256) || s.Estado != "sin_presentar" || s.Version != b.Version || s.HuellaSHA256 != b.HuellaSHA256) {
		return ErrAdmisionPreparacion
	}
	return nil
}

// RevisarRequisitoAdmision prepara carencias y revisión. No interpreta texto,
// equivalencias, puntuaciones o reglas gobernadas que este corte no ejecuta.
// La ausencia y cualquier estado de hecho permanecen pendientes de decisión.
func RevisarRequisitoAdmision(r RequisitoAdmision, soportes []SoporteAdmision) (RevisionRequisitoAdmision, error) {
	cero := RevisionRequisitoAdmision{}
	if !tituloAdmisionValido(r.TituloPropuesto) || !meritos.ReferenciaValida(r.Referencia) || !meritos.ReferenciaValida(r.Version) ||
		!meritos.ReferenciaValida(r.HitoRef) || !fechaAdmision(r.HitoFecha) || len(r.HechosEsperados) > 32 ||
		(r.Representacion != "texto_libre" && r.Representacion != "estructurada") ||
		(r.ReglaRef == "") != (r.ReglaVersion == "") ||
		(r.ReglaRef != "" && (!meritos.ReferenciaValida(r.ReglaRef) || !meritos.ReferenciaValida(r.ReglaVersion))) ||
		(r.Representacion == "texto_libre" && r.ReglaRef != "") {
		return cero, ErrAdmisionPreparacion
	}
	esperados := map[string]int{}
	for _, h := range r.HechosEsperados {
		if !meritos.ReferenciaValida(h.Referencia) || h.Version < 1 || esperados[h.Referencia] != 0 {
			return cero, ErrAdmisionPreparacion
		}
		esperados[h.Referencia] = h.Version
	}
	causas := []string{"seleccion.admision.causa.fuentes_no_verificadas"}
	if r.Representacion == "texto_libre" {
		causas = append(causas, "seleccion.admision.causa.texto_libre")
	} else if r.ReglaRef == "" {
		causas = append(causas, "seleccion.admision.causa.regla_ausente")
	} else {
		causas = append(causas, "seleccion.admision.causa.evaluador_pendiente")
	}
	aportados := map[string]bool{}
	completar := len(esperados) == 0
	for _, h := range soportes {
		if h.EstadoAportado == "rechazado" {
			causas = append(causas, "seleccion.admision.causa.rechazo_aplicabilidad_pendiente")
			break
		}
	}
	for _, h := range soportes {
		if h.Desde > r.HitoFecha || (h.Hasta != "" && h.Hasta < r.HitoFecha) {
			causas = append(causas, "seleccion.admision.causa.vigencia_aplicabilidad_pendiente")
			break
		}
	}
	for _, h := range soportes {
		if esperados[h.Referencia] != h.Version || aportados[h.Referencia] ||
			!meritos.ReferenciaValida(h.FuenteRef) || !meritos.ReferenciaValida(h.FuenteVersion) ||
			!fechaAdmision(h.Desde) || (h.Hasta != "" && (!fechaAdmision(h.Hasta) || h.Hasta < h.Desde)) ||
			h.EvidenciasAportadas < 0 || h.EvidenciasAportadas > 32 ||
			(h.EstadoAportado != "declarado" && h.EstadoAportado != "pendiente" && h.EstadoAportado != "acreditado" && h.EstadoAportado != "rechazado") {
			return cero, ErrAdmisionPreparacion
		}
		aportados[h.Referencia] = true
		if h.EstadoAportado != "acreditado" || h.EvidenciasAportadas == 0 {
			completar = true
		}
	}
	if len(aportados) < len(esperados) || len(esperados) == 0 {
		causas = append(causas, "seleccion.admision.causa.dato_no_aportado")
		completar = true
	}
	for _, h := range soportes {
		if h.EstadoAportado != "acreditado" || h.EvidenciasAportadas == 0 {
			causas = append(causas, "seleccion.admision.causa.acreditacion_pendiente")
			break
		}
	}
	accion := "seleccion.admision.accion.preparar_revision"
	if completar {
		accion = "seleccion.admision.accion.preparar_aportacion"
	}
	r.HechosEsperados = append([]ReferenciaHechoAdmision{}, r.HechosEsperados...)
	return RevisionRequisitoAdmision{r, "pendiente", causas, append([]SoporteAdmision{}, soportes...), accion}, nil
}

func shaAdmision(s string) bool {
	b, err := hex.DecodeString(s)
	return len(s) == 64 && err == nil && len(b) == 32 && strings.ToLower(s) == s
}

func fechaAdmision(s string) bool {
	d, err := time.Parse(time.DateOnly, s)
	return err == nil && d.Year() > 0
}

func tituloAdmisionValido(s string) bool {
	return s != "" && len(s) <= 256 && utf8.ValidString(s) && strings.TrimSpace(s) == s && !strings.ContainsFunc(s, unicode.IsControl)
}

package httpinterno

import (
	"context"
	"net/http"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// RutaVinculoCategoriaRPTB2 consulta (GET) y registra (POST) el vínculo CT154
// entre un expediente y una categoría RPT publicada. Organización, catálogo,
// módulo, actor y perfiles los pone el servidor; el canal sólo trae la intención.
const RutaVinculoCategoriaRPTB2 = "/api/vec/contratacion-temporal/incorporacion-personal-b2/vinculo-categoria-rpt/v1"

// EjecutorVinculoCategoriaRPTB2 lo implementa la composición con el servicio
// ServicioVinculoCategoriaRPT ya existente.
type EjecutorVinculoCategoriaRPTB2 interface {
	ConsultarVinculo(context.Context, string) (ports.LecturaVinculoCategoriaRPT, error)
	RegistrarVinculo(context.Context, EntradaVinculoCategoriaRPTB2) (ports.ReciboVinculoCategoriaRPT, error)
}

// EntradaVinculoCategoriaRPTB2 lleva las versiones esperadas (expediente,
// análisis y revisión del vínculo) y la clave de idempotencia, como el resto
// de actos CT: un reintento con la misma clave y el mismo material devuelve
// el recibo original.
type EntradaVinculoCategoriaRPTB2 struct {
	ExpedienteRef             string `json:"expediente_ref"`
	VersionExpedienteEsperada uint64 `json:"version_expediente_esperada"`
	AnalisisVersion           uint64 `json:"analisis_version"`
	AnalisisReciboRef         string `json:"analisis_recibo_ref"`
	AnalisisHuellaSHA256      string `json:"analisis_huella_sha256"`
	CategoriaRef              string `json:"categoria_ref"`
	CatalogoVersion           uint64 `json:"catalogo_version"`
	CatalogoHuellaSHA256      string `json:"catalogo_huella_sha256"`
	CategoriaID               string `json:"categoria_id"`
	FuenteRef                 string `json:"fuente_ref"`
	MotivoRef                 string `json:"motivo_ref"`
	AprobacionRef             string `json:"aprobacion_ref"`
	RevisionEsperada          uint64 `json:"revision_esperada"`
	// AnteriorReciboRef va vacío en la primera revisión: el lector estricto
	// del canal B2 no admite null.
	AnteriorReciboRef string `json:"anterior_recibo_ref"`
	ClaveIdempotencia string `json:"clave_idempotencia"`
}

var camposEntradaVinculoCategoriaRPTB2 = []string{"expediente_ref", "version_expediente_esperada", "analisis_version", "analisis_recibo_ref", "analisis_huella_sha256", "categoria_ref", "catalogo_version", "catalogo_huella_sha256", "categoria_id", "fuente_ref", "motivo_ref", "aprobacion_ref", "revision_esperada", "anterior_recibo_ref", "clave_idempotencia"}

// Validar es una defensa del canal; el registro completo lo vuelve a validar
// ports.RegistroVinculoCategoriaRPT y, después, la transacción CT154.
func (e EntradaVinculoCategoriaRPTB2) Validar() error {
	if !domain.ReferenciaOpacaValida(e.ExpedienteRef) || !versionHTTPB2(e.VersionExpedienteEsperada) || !versionHTTPB2(e.AnalisisVersion) ||
		!domain.ReferenciaOpacaValida(e.AnalisisReciboRef) || !patronSHAHTTPB2.MatchString(e.AnalisisHuellaSHA256) ||
		!patronSHAHTTPB2.MatchString(e.CatalogoHuellaSHA256) || e.CatalogoVersion == 0 || e.CategoriaRef != e.CategoriaID ||
		!patronClaveHTTPB2.MatchString(e.CategoriaID) || !domain.ReferenciaOpacaValida(e.FuenteRef) ||
		!domain.ReferenciaOpacaValida(e.MotivoRef) || !domain.ReferenciaOpacaValida(e.AprobacionRef) ||
		!patronUUIDHTTPB2.MatchString(e.ClaveIdempotencia) || (e.RevisionEsperada == 0) != (e.AnteriorReciboRef == "") ||
		(e.AnteriorReciboRef != "" && !domain.ReferenciaOpacaValida(e.AnteriorReciboRef)) {
		return ErrPeticionIncorporacionPersonalB2
	}
	return nil
}

// ReciboVinculoCategoriaRPTB2HTTP omite las referencias internas de decisión,
// auditoría y consumo V3; quedan en PostgreSQL junto al recibo.
type ReciboVinculoCategoriaRPTB2HTTP struct {
	ExpedienteRef                string                          `json:"expediente_ref"`
	ReciboRef                    string                          `json:"recibo_ref"`
	RegistradoEn                 string                          `json:"registrado_en"`
	Revision                     uint64                          `json:"revision"`
	Prospectivo                  bool                            `json:"prospectivo"`
	AcreditaProcedenciaHistorica bool                            `json:"acredita_procedencia_historica"`
	Vinculo                      ports.EstadoVinculoCategoriaRPT `json:"vinculo"`
}

// NuevoManejadorVinculoCategoriaRPTB2 se monta en la ruta exacta detrás de la
// frontera B2; la identidad y los perfiles nominales ya van en el contexto.
func NuevoManejadorVinculoCategoriaRPTB2(e EjecutorVinculoCategoriaRPTB2) (http.Handler, error) {
	if dependenciaNula(e) {
		return nil, ErrManejadorIncorporacionPersonalB2
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rutaHTTPVinculoRPTExacta(r) {
			errorHTTPB2(w, r, 400, "peticion_no_valida")
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			w.Header().Set("Allow", "GET, POST")
			errorHTTPB2(w, r, 405, "metodo_no_permitido")
			return
		}
		if r.Context().Err() != nil {
			errorOperacionHTTPB2(w, r, r.Context().Err())
			return
		}
		if !cabecerasPropuestaFormalizacionPermitidas(r) || !acceptCompatibleJSON(r.Header) {
			errorHTTPB2(w, r, 400, "peticion_no_valida")
			return
		}
		if r.Method == http.MethodGet {
			exp, err := leerConsultaIncorporacionEjercicioV2(r)
			if err != nil || !referenciaConsultaHTTPB2(exp) {
				errorHTTPB2(w, r, 400, "peticion_no_valida")
				return
			}
			out, err := e.ConsultarVinculo(r.Context(), exp)
			if r.Context().Err() != nil {
				errorOperacionHTTPB2(w, r, r.Context().Err())
				return
			}
			if err != nil || out.ExpedienteRef != exp || out.Analisis.Validar() != nil || (out.Vinculo != nil && out.Vinculo.Validar() != nil) {
				if err == nil {
					err = ErrManejadorIncorporacionPersonalB2
				}
				errorOperacionHTTPB2(w, r, err)
				return
			}
			responderJSONCobertura(w, r, 200, struct {
				Data ports.LecturaVinculoCategoriaRPT `json:"data"`
			}{out})
			return
		}
		var entrada EntradaVinculoCategoriaRPTB2
		err := leerCuerpoHTTPB2(w, r, &entrada, camposEntradaVinculoCategoriaRPTB2)
		if r.Context().Err() != nil {
			errorOperacionHTTPB2(w, r, r.Context().Err())
			return
		}
		if err != nil {
			errorHTTPB2(w, r, 400, "peticion_no_valida")
			return
		}
		if entrada.Validar() != nil {
			errorHTTPB2(w, r, 422, "contenido_no_valido")
			return
		}
		recibo, err := e.RegistrarVinculo(r.Context(), entrada)
		if r.Context().Err() != nil {
			errorOperacionHTTPB2(w, r, r.Context().Err())
			return
		}
		if err != nil {
			if !reflect.ValueOf(recibo).IsZero() {
				err = ErrManejadorIncorporacionPersonalB2
			}
			errorOperacionHTTPB2(w, r, err)
			return
		}
		if recibo.Revision != entrada.RevisionEsperada+1 || recibo.Vinculo.CategoriaID != entrada.CategoriaID || recibo.Vinculo.Validar() != nil {
			errorOperacionHTTPB2(w, r, ErrManejadorIncorporacionPersonalB2)
			return
		}
		responderJSONCobertura(w, r, 200, struct {
			Data ReciboVinculoCategoriaRPTB2HTTP `json:"data"`
		}{ReciboVinculoCategoriaRPTB2HTTP{ExpedienteRef: entrada.ExpedienteRef, ReciboRef: recibo.ReciboRef,
			RegistradoEn: recibo.RegistradoEn.UTC().Format(time.RFC3339Nano), Revision: recibo.Revision,
			Prospectivo: recibo.Prospectivo, AcreditaProcedenciaHistorica: recibo.AcreditaProcedenciaHistorica, Vinculo: recibo.Vinculo}})
	}), nil
}

func rutaHTTPVinculoRPTExacta(r *http.Request) bool {
	return r != nil && r.URL != nil && r.URL.Path == RutaVinculoCategoriaRPTB2 && r.URL.RawPath == "" && r.URL.Scheme == "" && r.URL.Host == "" && r.URL.User == nil && r.URL.Opaque == "" && r.URL.Fragment == "" && r.URL.RawFragment == "" && !r.URL.ForceQuery && (r.Method == http.MethodGet || r.URL.RawQuery == "") && r.URL.EscapedPath() == r.URL.Path
}

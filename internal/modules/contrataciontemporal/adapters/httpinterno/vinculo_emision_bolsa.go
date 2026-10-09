package httpinterno

import (
	"context"
	"errors"
	"io"
	"net/http"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const RutaVinculosEmisionBolsa = "/api/vec/contratacion-temporal/expedientes/vinculos-bolsa"

type EjecutorVinculoEmisionBolsa interface {
	Vincular(context.Context, ports.SolicitudVinculoEmisionBolsa) (ports.ReciboVinculoEmisionBolsa, error)
}

type manejadorVinculoEmisionBolsa struct {
	autoridad AutoridadCanalSeguimiento
	ejecutor  EjecutorVinculoEmisionBolsa
}

func NuevoManejadorVinculoEmisionBolsa(a AutoridadCanalSeguimiento, e EjecutorVinculoEmisionBolsa) (http.Handler, error) {
	if dependenciaNula(a) || dependenciaNula(e) {
		return nil, ports.ErrVinculoEmisionBolsaNoDisponible
	}
	return &manejadorVinculoEmisionBolsa{autoridad: a, ejecutor: e}, nil
}

func (h *manejadorVinculoEmisionBolsa) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || r.URL == nil || r.URL.Path != RutaVinculosEmisionBolsa || r.URL.RawQuery != "" || r.URL.ForceQuery {
		responderErrorVinculoEmisionBolsa(w, r, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodPost {
		responderErrorVinculoEmisionBolsa(w, r, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if !tipoContenidoJSON(r.Header) || cabeceraCoberturaProhibida(r.Header) {
		responderErrorVinculoEmisionBolsa(w, r, http.StatusBadRequest, "peticion_no_permitida")
		return
	}
	contenido, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 8*1024))
	if err != nil || len(contenido) == 0 || len(contenido) > 8*1024 {
		responderErrorVinculoEmisionBolsa(w, r, http.StatusUnprocessableEntity, "contenido_no_valido")
		return
	}
	canal, err := h.autoridad.ResolverContextoCanalSeguimiento(r.Context())
	if err != nil || !canal.Valido() {
		responderErrorVinculoEmisionBolsa(w, r, http.StatusForbidden, "acceso_denegado", err)
		return
	}
	var entrada struct {
		ExpedienteRef     string `json:"expediente_ref"`
		VersionEsperada   uint64 `json:"version_esperada"`
		BolsaRef          string `json:"bolsa_ref"`
		LlamamientoRef    string `json:"llamamiento_ref"`
		ReciboEmisionRef  string `json:"recibo_emision_ref"`
		ClaveIdempotencia string `json:"clave_idempotencia"`
	}
	if decodificarCuerpoSeguimiento(contenido, &entrada) != nil {
		responderErrorVinculoEmisionBolsa(w, r, http.StatusUnprocessableEntity, "contenido_no_valido")
		return
	}
	s := ports.SolicitudVinculoEmisionBolsa{OrganizacionRef: canal.OrganizacionRef,
		ExpedienteRef: entrada.ExpedienteRef, VersionEsperada: entrada.VersionEsperada,
		BolsaRef: entrada.BolsaRef, LlamamientoRef: entrada.LlamamientoRef,
		ReciboEmisionRef: entrada.ReciboEmisionRef, ClaveIdempotencia: entrada.ClaveIdempotencia}
	if s.Validar() != nil {
		responderErrorVinculoEmisionBolsa(w, r, http.StatusUnprocessableEntity, "contenido_no_valido")
		return
	}
	recibo, err := h.ejecutor.Vincular(r.Context(), s)
	if err != nil {
		estado, codigo := estadoErrorVinculoEmisionBolsa(err)
		responderErrorVinculoEmisionBolsa(w, r, estado, codigo, err)
		return
	}
	if recibo.ValidarPara(s) != nil {
		responderErrorVinculoEmisionBolsa(w, r, http.StatusBadGateway, "resultado_no_confiable")
		return
	}
	estado := http.StatusCreated
	if recibo.Reutilizado {
		estado = http.StatusOK
	}
	responderJSONCobertura(w, r, estado, map[string]any{"data": recibo})
}

func estadoErrorVinculoEmisionBolsa(err error) (int, string) {
	switch {
	case errors.Is(err, ports.ErrVinculoEmisionBolsaInvalido):
		return http.StatusUnprocessableEntity, "contenido_no_valido"
	case errors.Is(err, ports.ErrVinculoEmisionBolsaConflicto):
		return http.StatusConflict, "vinculo_en_conflicto"
	case errors.Is(err, ports.ErrAutorizacionDenegada):
		return http.StatusForbidden, "acceso_denegado"
	default:
		return http.StatusServiceUnavailable, "servicio_no_disponible"
	}
}

func responderErrorVinculoEmisionBolsa(w http.ResponseWriter, r *http.Request, estado int, codigo string, causas ...error) {
	responderJSONCobertura(w, r, estado, envoltorioErrorCobertura{Error: detalleErrorCobertura{
		Codigo: codigo, ClaveI18n: "api.contratacion_temporal.vinculo_bolsa.error." + codigo,
		CorrelacionRef: nuevaCorrelacionCobertura()}}, causas...)
}

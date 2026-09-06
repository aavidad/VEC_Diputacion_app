package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// La composición registra esta operación solo con un ejecutor que confirma
// el recibo del módulo propietario. Abrir un llamamiento no envía un aviso.
type EjecutorContinuacionLlamamiento interface {
	Continuar(context.Context, ports.SolicitudContinuarLlamamiento) (ports.ResultadoContinuacionLlamamiento, error)
}

type continuacionLlamamientoJSON struct {
	ClaveIdempotencia string `json:"clave_idempotencia"`
	OrganizacionRef   string `json:"organizacion_ref"`
	ExpedienteRef     string `json:"expediente_ref"`
	ResolucionRef     string `json:"resolucion_ref"`
	IntencionRef      string `json:"intencion_ref"`
}

type continuacionLlamamientoSalidaJSON struct {
	Esquema                string `json:"esquema"`
	OrganizacionRef        string `json:"organizacion_ref"`
	ExpedienteRef          string `json:"expediente_ref"`
	ResolucionRef          string `json:"resolucion_ref"`
	IntencionRef           string `json:"intencion_ref"`
	LlamamientoAnteriorRef string `json:"llamamiento_anterior_ref"`
	LlamamientoRef         string `json:"llamamiento_ref"`
	VersionLlamamiento     uint64 `json:"version_llamamiento"`
	ReciboBolsaRef         string `json:"recibo_bolsa_ref"`
	ReciboRef              string `json:"recibo_ref"`
	AuditoriaRef           string `json:"auditoria_ref"`
	ConfirmadaEn           string `json:"confirmada_en"`
	EstadoIntencion        string `json:"estado_intencion"`
	EstadoLocal            string `json:"estado_local"`
}

func (h *manejadorComunicacionLlamamiento) continuar(w http.ResponseWriter, r *http.Request) {
	e, disponible := h.ejecutor.(EjecutorContinuacionLlamamiento)
	if !disponible || dependenciaNula(e) {
		responderErrorComunicacionLlamamiento(w, errorServicioComunicacionLlamamientoNoDisponible)
		return
	}
	var entrada continuacionLlamamientoJSON
	if err := decodificarComunicacionLlamamiento(w, r, &entrada); err != nil {
		responderErrorComunicacionLlamamiento(w, errorEntradaComunicacionLlamamiento(err))
		return
	}
	s := ports.SolicitudContinuarLlamamiento{ClaveIdempotencia: entrada.ClaveIdempotencia,
		OrganizacionRef: entrada.OrganizacionRef, ExpedienteRef: entrada.ExpedienteRef,
		ResolucionRef: entrada.ResolucionRef, IntencionRef: entrada.IntencionRef}
	if s.Validar() != nil {
		responderErrorComunicacionLlamamiento(w, errorContenidoComunicacionLlamamientoInvalido)
		return
	}
	resultado, err := e.Continuar(r.Context(), s)
	if r.Context().Err() != nil {
		err = r.Context().Err()
	}
	if err != nil {
		problema := clasificarErrorComunicacionLlamamientoHTTP(err)
		switch {
		case errors.Is(err, ports.ErrOperacionContinuacionInvalida):
			problema = errorContenidoComunicacionLlamamientoInvalido
		case errors.Is(err, ports.ErrOperacionContinuacionDenegada):
			problema = errorAccesoComunicacionLlamamientoDenegado
		case errors.Is(err, ports.ErrOperacionContinuacionConflicto):
			problema = errorClaveComunicacionLlamamientoReutilizada
		}
		responderErrorComunicacionLlamamiento(w, problema)
		return
	}
	if resultado.ValidarPara(s) != nil {
		responderErrorComunicacionLlamamiento(w, errorResultadoComunicacionLlamamientoNoConfiable)
		return
	}
	salida := continuacionLlamamientoSalidaJSON{
		Esquema:         "vec.contratacion-temporal.continuacion-llamamiento.v1",
		OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
		ResolucionRef: s.ResolucionRef, IntencionRef: s.IntencionRef,
		LlamamientoAnteriorRef: resultado.LlamamientoAnteriorRef,
		LlamamientoRef:         resultado.ReciboBolsa.LlamamientoRef, VersionLlamamiento: 1,
		ReciboBolsaRef: resultado.ReciboBolsa.ReciboRef, ReciboRef: resultado.ReciboRef,
		AuditoriaRef: resultado.AuditoriaRef, ConfirmadaEn: resultado.ConfirmadaEn.Format(time.RFC3339Nano),
		EstadoIntencion: "despachada", EstadoLocal: resultado.Estado,
	}
	estado := http.StatusCreated
	if resultado.Estado == "replay_confirmado" {
		estado = http.StatusOK
	}
	responderJSONCobertura(w, estado, struct {
		Data continuacionLlamamientoSalidaJSON `json:"data"`
	}{salida})
}

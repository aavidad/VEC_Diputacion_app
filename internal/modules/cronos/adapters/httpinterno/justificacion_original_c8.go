package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

const (
	RutaAportarOriginalC8 = "/api/interna/cronos/permisos/justificantes"
	RutaRevisarOriginalC8 = "/api/interna/cronos/permisos/justificantes/revisiones"
)

type CasoUsoOriginalC8 interface {
	Aportar(context.Context, ports.OrdenOriginalC8, ports.PeticionAnexoOriginalC8) (ports.ReciboOriginalC8, error)
	Revisar(context.Context, ports.OrdenOriginalC8, ports.PeticionRevisionOriginalC8V2) (ports.ReciboOriginalC8, error)
}

type ResolverOriginalC8 interface {
	ResolverOriginal(*http.Request) (ports.OrdenOriginalC8, error)
}

type ManejadorOriginalC8 struct {
	casoUso  CasoUsoOriginalC8
	resolver ResolverOriginalC8
}

func NuevoManejadorOriginalC8(c CasoUsoOriginalC8, r ResolverOriginalC8) (*ManejadorOriginalC8, error) {
	if dependenciaCronosHTTPNula(c) || dependenciaCronosHTTPNula(r) {
		return nil, ports.ErrJustificacionNoDisponible
	}
	return &ManejadorOriginalC8{c, r}, nil
}

func (m *ManejadorOriginalC8) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cabecerasJSON(w)
	if r == nil || r.URL == nil || r.URL.RawPath != "" || r.URL.RawQuery != "" ||
		(r.URL.Path != RutaAportarOriginalC8 && r.URL.Path != RutaRevisarOriginalC8) {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	if !soloMetodo(w, r, http.MethodPost) {
		return
	}
	if m == nil || dependenciaCronosHTTPNula(m.casoUso) || dependenciaCronosHTTPNula(m.resolver) {
		responderErrorAccesoCronos(w, ports.ErrJustificacionNoDisponible)
		return
	}
	cadenas := map[string]int{"solicitud_ref": 160, "clave_operacion": 128, "version_esperada": 20}
	obligatorios := []string{"solicitud_ref", "clave_operacion", "version_esperada"}
	anexo := r.URL.Path == RutaAportarOriginalC8
	if anexo {
		cadenas["documento_id"], cadenas["documento_version"] = 128, 20
		obligatorios = append(obligatorios, "documento_id", "documento_version")
	} else {
		cadenas["decision"], cadenas["motivo_ref"] = 16, 160
		obligatorios = append(obligatorios, "decision", "motivo_ref")
	}
	v, err := decodificarCadenasAcotadas(w, r, obligatorios, cadenas)
	if err != nil {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	version, err := strconv.ParseInt(v["version_esperada"], 10, 64)
	if err != nil || version < 0 || strconv.FormatInt(version, 10) != v["version_esperada"] {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	orden, err := m.resolver.ResolverOriginal(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	var recibo ports.ReciboOriginalC8
	if anexo {
		docVersion, e := strconv.ParseUint(v["documento_version"], 10, 64)
		if e != nil || docVersion == 0 || strconv.FormatUint(docVersion, 10) != v["documento_version"] {
			errorJSON(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		recibo, err = m.casoUso.Aportar(r.Context(), orden, ports.PeticionAnexoOriginalC8{
			SolicitudRef: v["solicitud_ref"], ClaveOperacion: v["clave_operacion"], VersionEsperada: version,
			DocumentoID: v["documento_id"], DocumentoVersion: docVersion,
		})
	} else {
		recibo, err = m.casoUso.Revisar(r.Context(), orden, ports.PeticionRevisionOriginalC8V2{
			SolicitudRef: v["solicitud_ref"], ClaveOperacion: v["clave_operacion"], VersionEsperada: version,
			Decision: domain.EstadoJustificacion(v["decision"]), MotivoRef: v["motivo_ref"],
		})
	}
	if err != nil {
		responderErrorOriginalC8(w, err)
		return
	}
	if recibo.SolicitudRef != v["solicitud_ref"] || recibo.Version != version+1 || recibo.ReciboRef == "" ||
		recibo.DocumentoID == "" || recibo.DocumentoVersion == 0 || recibo.SHA256 == "" ||
		recibo.InstanteUTC.IsZero() || recibo.InstanteUTC.Location() != time.UTC {
		responderErrorAccesoCronos(w, ports.ErrJustificacionNoDisponible)
		return
	}
	if anexo && (recibo.DocumentoID != v["documento_id"] || recibo.DocumentoVersion != mustUintC8(v["documento_version"]) ||
		recibo.Estado != domain.JustificacionPendiente) {
		responderErrorAccesoCronos(w, ports.ErrJustificacionNoDisponible)
		return
	}
	if !anexo && recibo.Estado != domain.EstadoJustificacion(v["decision"]) {
		responderErrorAccesoCronos(w, ports.ErrJustificacionNoDisponible)
		return
	}
	if !recibo.Replay {
		w.WriteHeader(http.StatusCreated)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"recibo": map[string]any{
		"solicitud_ref": recibo.SolicitudRef, "documento_id": recibo.DocumentoID,
		"documento_version": recibo.DocumentoVersion, "sha256": recibo.SHA256,
		"estado": recibo.Estado, "version": recibo.Version, "recibo_ref": recibo.ReciboRef,
		"instante_utc": recibo.InstanteUTC, "replay": recibo.Replay,
	}})
}

func mustUintC8(v string) uint64 { n, _ := strconv.ParseUint(v, 10, 64); return n }

func responderErrorOriginalC8(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrJustificacionInvalida):
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
	case errors.Is(err, domain.ErrJustificacionConflicto):
		errorJSON(w, http.StatusConflict, "estado_cambiado")
	case errors.Is(err, ports.ErrResolucionNoCompetente):
		errorJSON(w, http.StatusForbidden, "no_competente")
	default:
		responderErrorAccesoCronos(w, err)
	}
}

package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

const (
	RutaAportarJustificanteC8 = "/api/interna/cronos/permisos/justificantes"
	RutaRevisarJustificanteC8 = "/api/interna/cronos/permisos/justificantes/revisiones"
)

type CasoUsoJustificantesC8 interface {
	Aportar(context.Context, ports.OrdenJustificacion, ports.PeticionAporteOriginal) (ports.ResultadoAnexoJustificacion, error)
	Revisar(context.Context, ports.OrdenJustificacion, ports.PeticionRevisionOriginal) (ports.ReciboJustificacion, error)
}

type ResolverJustificantesC8 interface {
	ResolverJustificacion(*http.Request) (ports.OrdenJustificacion, error)
}

type ManejadorJustificantesC8 struct {
	resolver ResolverJustificantesC8
	casoUso  CasoUsoJustificantesC8
}

func NuevoManejadorJustificantesC8(c CasoUsoJustificantesC8, r ResolverJustificantesC8) (*ManejadorJustificantesC8, error) {
	if dependenciaCronosHTTPNula(c) || dependenciaCronosHTTPNula(r) {
		return nil, ports.ErrJustificacionNoDisponible
	}
	return &ManejadorJustificantesC8{r, c}, nil
}

func (m *ManejadorJustificantesC8) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cabecerasJSON(w)
	if r == nil || r.URL == nil || r.URL.RawPath != "" || r.URL.RawQuery != "" ||
		(r.URL.Path != RutaAportarJustificanteC8 && r.URL.Path != RutaRevisarJustificanteC8) {
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
	campos := map[string]int{"solicitud_ref": 160, "clave_operacion": 128, "version_esperada": 20}
	obligatorios := []string{"solicitud_ref", "clave_operacion", "version_esperada"}
	if r.URL.Path == RutaAportarJustificanteC8 {
		campos["documento_id"] = 128
		campos["documento_version"] = 20
		campos["sha256"] = 64
		obligatorios = append(obligatorios, "documento_id", "documento_version", "sha256")
	} else {
		campos["decision"] = 16
		campos["motivo_ref"] = 160
		obligatorios = append(obligatorios, "decision", "motivo_ref")
	}
	v, err := decodificarCadenasAcotadas(w, r, obligatorios, campos)
	if err != nil {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	version, err := strconv.ParseInt(v["version_esperada"], 10, 64)
	if err != nil || version < 0 || strconv.FormatInt(version, 10) != v["version_esperada"] {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	orden, err := m.resolver.ResolverJustificacion(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	if r.URL.Path == RutaAportarJustificanteC8 {
		docVersion, e := strconv.ParseUint(v["documento_version"], 10, 64)
		if e != nil || docVersion == 0 || strconv.FormatUint(docVersion, 10) != v["documento_version"] {
			errorJSON(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		res, e := m.casoUso.Aportar(r.Context(), orden, ports.PeticionAporteOriginal{
			SolicitudRef: v["solicitud_ref"], ClaveOperacion: v["clave_operacion"],
			VersionEsperada: version, DocumentoID: v["documento_id"],
			DocumentoVersion: docVersion, SHA256: v["sha256"],
		})
		if e != nil && !errors.Is(e, ports.ErrEnlaceJustificacionPendiente) {
			responderErrorJustificantesC8(w, e)
			return
		}
		if res.EnlacePendiente {
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{"enlace_pendiente": true})
			return
		}
		if res.ReciboCronos == nil {
			responderErrorAccesoCronos(w, ports.ErrJustificacionNoDisponible)
			return
		}
		responderReciboC8(w, *res.ReciboCronos)
		return
	}
	res, err := m.casoUso.Revisar(r.Context(), orden, ports.PeticionRevisionOriginal{
		SolicitudRef: v["solicitud_ref"], ClaveOperacion: v["clave_operacion"],
		VersionEsperada: version, Decision: domain.EstadoJustificacion(v["decision"]), MotivoRef: v["motivo_ref"],
	})
	if err != nil {
		responderErrorJustificantesC8(w, err)
		return
	}
	responderReciboC8(w, res)
}

func responderReciboC8(w http.ResponseWriter, r ports.ReciboJustificacion) {
	if r.ReciboRef == "" || r.FechaUTC.IsZero() || r.Registro == nil {
		responderErrorAccesoCronos(w, ports.ErrJustificacionNoDisponible)
		return
	}
	if !r.Replay {
		w.WriteHeader(http.StatusCreated)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"recibo": map[string]any{
		"solicitud_ref": r.Justificacion.Vinculo.SolicitudRef, "documento_id": r.Registro.Documento.ID,
		"documento_version": r.Registro.Documento.Version, "sha256": r.Registro.Documento.SHA256,
		"estado": r.Justificacion.Estado, "version": r.Justificacion.Version,
		"recibo_ref": r.ReciboRef, "instante_utc": r.FechaUTC, "replay": r.Replay,
	}})
}

func responderErrorJustificantesC8(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrJustificacionInvalida):
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
	case errors.Is(err, domain.ErrJustificacionConflicto):
		errorJSON(w, http.StatusConflict, "estado_cambiado")
	default:
		responderErrorAccesoCronos(w, err)
	}
}

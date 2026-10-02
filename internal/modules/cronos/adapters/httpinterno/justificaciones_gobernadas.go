package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

const (
	RutaConsultarJustificacion = "/api/interna/cronos/justificaciones/consulta"
	RutaAnexarJustificacion    = "/api/interna/cronos/justificaciones/anexos"
	RutaRevisarJustificacion   = "/api/interna/cronos/justificaciones/revisiones"
	maximoJSONJustificacion    = 8192
)

// La orden sale de la identidad registrada en la frontera interna. Ningún
// campo del cuerpo HTTP selecciona actor, perfil, empleado ni política.
type ResolverOrdenJustificacion interface {
	ResolverOrdenJustificacion(*http.Request) (ports.OrdenJustificacion, error)
}

type CasoUsoJustificacionHTTP interface {
	Consultar(context.Context, ports.OrdenJustificacion, string) (ports.PreparacionJustificacion, error)
	Anexar(context.Context, ports.OrdenJustificacion, ports.PeticionAnexoJustificacion) (ports.ResultadoAnexoJustificacion, error)
	Revisar(context.Context, ports.OrdenJustificacion, ports.PeticionRevisionJustificacion) (ports.ReciboJustificacion, error)
}

type ManejadorJustificaciones struct {
	resolver ResolverOrdenJustificacion
	casoUso  CasoUsoJustificacionHTTP
}

func NuevoManejadorJustificaciones(casoUso CasoUsoJustificacionHTTP, resolver ResolverOrdenJustificacion) (*ManejadorJustificaciones, error) {
	if dependenciaCronosHTTPNula(casoUso) || dependenciaCronosHTTPNula(resolver) {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &ManejadorJustificaciones{resolver: resolver, casoUso: casoUso}, nil
}

func (m *ManejadorJustificaciones) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cabecerasJSON(w)
	if m == nil || r == nil || r.URL == nil || r.URL.RawPath != "" {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	switch r.URL.Path {
	case RutaConsultarJustificacion:
		m.consultar(w, r)
	case RutaAnexarJustificacion:
		m.anexar(w, r)
	case RutaRevisarJustificacion:
		m.revisar(w, r)
	default:
		errorJSON(w, http.StatusNotFound, "no_disponible")
	}
}

func (m *ManejadorJustificaciones) consultar(w http.ResponseWriter, r *http.Request) {
	if !soloMetodo(w, r, http.MethodGet) {
		return
	}
	ref, ok := referenciaConsultaJustificacion(r.URL.RawQuery)
	if !ok {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	orden, err := m.resolver.ResolverOrdenJustificacion(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	preparacion, err := m.casoUso.Consultar(r.Context(), orden, ref)
	if err != nil {
		responderErrorJustificacion(w, err)
		return
	}
	// Proyección acotada: los motivos y referencias documentales completos
	// permanecen en el servicio y se consultan con facultades nominales.
	actual := map[string]any(nil)
	if preparacion.Actual != nil {
		actual = map[string]any{"version": preparacion.Actual.Version, "estado": preparacion.Actual.Estado,
			"documento_id": preparacion.Actual.Vinculo.Documento.ID, "documento_version": preparacion.Actual.Vinculo.Documento.Version}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"solicitud_ref": preparacion.Solicitud.SolicitudRef, "version": preparacion.Solicitud.Version,
		"estado": preparacion.Solicitud.Estado, "justificante_exigido": preparacion.Solicitud.JustificanteExigido,
		"politica_ref": preparacion.Politica.Referencia, "politica_version": preparacion.Politica.Version,
		"motivos_ref": preparacion.Politica.MotivosRef, "actual": actual,
	})
}

func (m *ManejadorJustificaciones) anexar(w http.ResponseWriter, r *http.Request) {
	if !soloMetodo(w, r, http.MethodPost) {
		return
	}
	if r.URL.RawQuery != "" {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	var cuerpo struct {
		SolicitudRef    string                        `json:"solicitud_ref"`
		ClaveOperacion  string                        `json:"clave_operacion"`
		VersionEsperada int64                         `json:"version_esperada"`
		Documento       domain.DocumentoJustificacion `json:"documento"`
	}
	if !decodificarJustificacion(w, r, &cuerpo) || cuerpo.SolicitudRef == "" || cuerpo.ClaveOperacion == "" || cuerpo.VersionEsperada < 0 || cuerpo.Documento.Validar() != nil {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	orden, err := m.resolver.ResolverOrdenJustificacion(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	resultado, err := m.casoUso.Anexar(r.Context(), orden, ports.PeticionAnexoJustificacion{
		SolicitudRef: cuerpo.SolicitudRef, ClaveOperacion: cuerpo.ClaveOperacion,
		VersionEsperada: cuerpo.VersionEsperada, Documento: cuerpo.Documento,
	})
	if err != nil {
		if errors.Is(err, ports.ErrEnlaceJustificacionPendiente) && resultado.EnlacePendiente &&
			resultado.Documento != nil && resultado.Documento.Validar() == nil && resultado.ReciboCronos == nil {
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(proyectarAnexoJustificacion(resultado))
			return
		}
		responderErrorJustificacion(w, err)
		return
	}
	if resultado.Documento == nil || resultado.Documento.Validar() != nil || (resultado.EnlacePendiente && resultado.ReciboCronos != nil) ||
		(!resultado.EnlacePendiente && (resultado.ReciboCronos == nil || !reciboJustificacionHTTPValido(*resultado.ReciboCronos))) {
		responderErrorAccesoCronos(w, ports.ErrDependenciaNoDisponible)
		return
	}
	if resultado.EnlacePendiente {
		w.WriteHeader(http.StatusAccepted)
	} else if !resultado.ReciboCronos.Replay {
		w.WriteHeader(http.StatusCreated)
	}
	_ = json.NewEncoder(w).Encode(proyectarAnexoJustificacion(resultado))
}

func (m *ManejadorJustificaciones) revisar(w http.ResponseWriter, r *http.Request) {
	if !soloMetodo(w, r, http.MethodPost) {
		return
	}
	if r.URL.RawQuery != "" {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	var cuerpo struct {
		SolicitudRef    string                     `json:"solicitud_ref"`
		ClaveOperacion  string                     `json:"clave_operacion"`
		VersionEsperada int64                      `json:"version_esperada"`
		Decision        domain.EstadoJustificacion `json:"decision"`
		MotivoRef       string                     `json:"motivo_ref"`
	}
	if !decodificarJustificacion(w, r, &cuerpo) || cuerpo.SolicitudRef == "" || cuerpo.ClaveOperacion == "" || cuerpo.VersionEsperada < 1 ||
		(cuerpo.Decision != domain.JustificacionAceptada && cuerpo.Decision != domain.JustificacionRechazada) || cuerpo.MotivoRef == "" {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	orden, err := m.resolver.ResolverOrdenJustificacion(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	preparacion, err := m.casoUso.Consultar(r.Context(), orden, cuerpo.SolicitudRef)
	if err != nil {
		responderErrorJustificacion(w, err)
		return
	}
	if preparacion.Actual == nil || preparacion.Actual.Vinculo.SolicitudRef != cuerpo.SolicitudRef {
		errorJSON(w, http.StatusConflict, "conflicto")
		return
	}
	recibo, err := m.casoUso.Revisar(r.Context(), orden, ports.PeticionRevisionJustificacion{
		SolicitudRef: cuerpo.SolicitudRef, ClaveOperacion: cuerpo.ClaveOperacion,
		VersionEsperada: cuerpo.VersionEsperada, Vinculo: preparacion.Actual.Vinculo,
		Decision: cuerpo.Decision, MotivoRef: cuerpo.MotivoRef,
	})
	if err != nil {
		responderErrorJustificacion(w, err)
		return
	}
	if !reciboJustificacionHTTPValido(recibo) {
		responderErrorAccesoCronos(w, ports.ErrDependenciaNoDisponible)
		return
	}
	if !recibo.Replay {
		w.WriteHeader(http.StatusCreated)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"recibo": proyectarReciboJustificacion(recibo)})
}

func proyectarAnexoJustificacion(r ports.ResultadoAnexoJustificacion) map[string]any {
	salida := map[string]any{"enlace_pendiente": r.EnlacePendiente}
	if r.Documento != nil {
		salida["documento"] = map[string]any{"id": r.Documento.ID, "version": r.Documento.Version,
			"sha256": r.Documento.SHA256}
	}
	if r.ReciboCronos != nil {
		salida["recibo_cronos"] = proyectarReciboJustificacion(*r.ReciboCronos)
	}
	return salida
}

func proyectarReciboJustificacion(r ports.ReciboJustificacion) map[string]any {
	return map[string]any{"recibo_ref": r.ReciboRef, "fecha_utc": r.FechaUTC,
		"version": r.Justificacion.Version, "estado": r.Justificacion.Estado, "replay": r.Replay}
}

func reciboJustificacionHTTPValido(r ports.ReciboJustificacion) bool {
	return r.ReciboRef != "" && !r.FechaUTC.IsZero() && r.Justificacion.Version > 0
}

func responderErrorJustificacion(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrJustificacionInvalida):
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
	case errors.Is(err, domain.ErrJustificacionConflicto):
		errorJSON(w, http.StatusConflict, "conflicto")
	default:
		responderErrorAccesoCronos(w, err)
	}
}

func referenciaConsultaJustificacion(raw string) (string, bool) {
	if len(raw) == 0 || len(raw) > 160 {
		return "", false
	}
	v, err := url.ParseQuery(raw)
	if err != nil || len(v) != 1 || len(v["solicitud_ref"]) != 1 {
		return "", false
	}
	ref := v.Get("solicitud_ref")
	return ref, domain.SolicitudPermisoRefValida(ref)
}

func decodificarJustificacion(w http.ResponseWriter, r *http.Request, destino any) bool {
	if r.Header.Get("Content-Type") != "application/json" || r.Body == nil {
		return false
	}
	defer r.Body.Close()
	contenido, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maximoJSONJustificacion))
	if err != nil || len(contenido) == 0 || !utf8.Valid(contenido) {
		return false
	}
	claves := json.NewDecoder(bytes.NewReader(contenido))
	claves.UseNumber()
	if recorrerJSONJustificacion(claves, 0) != nil {
		return false
	}
	if _, err := claves.Token(); !errors.Is(err, io.EOF) {
		return false
	}
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	return dec.Decode(destino) == nil
}

// La entrada C8 contiene sólo objetos y escalares; rechazamos listas,
// profundidad y claves repetidas antes de convertirla al DTO tipado.
func recorrerJSONJustificacion(dec *json.Decoder, profundidad int) error {
	if profundidad > 3 {
		return domain.ErrJustificacionInvalida
	}
	t, err := dec.Token()
	if err != nil {
		return err
	}
	d, compuesto := t.(json.Delim)
	if !compuesto {
		return nil
	}
	if d != '{' {
		return domain.ErrJustificacionInvalida
	}
	claves := make(map[string]struct{}, 16)
	for dec.More() {
		t, err := dec.Token()
		clave, valida := t.(string)
		if err != nil || !valida || len(claves) >= 24 {
			return domain.ErrJustificacionInvalida
		}
		for _, b := range []byte(clave) {
			if (b < 'a' || b > 'z') && (b < '0' || b > '9') && b != '_' {
				return domain.ErrJustificacionInvalida
			}
		}
		if _, existe := claves[clave]; existe {
			return domain.ErrJustificacionInvalida
		}
		claves[clave] = struct{}{}
		if err := recorrerJSONJustificacion(dec, profundidad+1); err != nil {
			return err
		}
	}
	cierre, err := dec.Token()
	if err != nil || cierre != json.Delim('}') {
		return domain.ErrJustificacionInvalida
	}
	return nil
}

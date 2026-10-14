package httpinterno

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

const RutaInformesCronosC12 = "/api/interna/cronos/informes/"

type ResolverInformeC12 interface {
	ResolverInformeC12(*http.Request, string) (ports.OrdenInformeC12, error)
}
type ManejadorInformeC12 struct {
	resolver ResolverInformeC12
	caso     ports.CasoUsoInformeC12
}

func NuevoManejadorInformeC12(caso ports.CasoUsoInformeC12, resolver ResolverInformeC12) (*ManejadorInformeC12, error) {
	if dependenciaCronosHTTPNula(caso) || dependenciaCronosHTTPNula(resolver) {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &ManejadorInformeC12{resolver, caso}, nil
}
func (m *ManejadorInformeC12) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if m == nil || m.caso == nil || m.resolver == nil || r == nil || r.URL == nil {
		errorJSON(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	equipo, nombre, ok := rutaInformeC12(r.URL)
	if !ok {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		errorJSON(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	partes := strings.Split(nombre, ".")
	if len(partes) != 2 {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	tipo, formato := domain.TipoInformeCronos(partes[0]), domain.FormatoInformeCronos(partes[1])
	desde, hasta, clave, err := parametrosInformeC12(w, r)
	if err != nil {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	orden, err := m.resolver.ResolverInformeC12(r, equipo)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	salida, err := m.caso.ExportarInforme(r.Context(), orden, ports.PeticionInformeC12{EquipoRef: equipo, Tipo: tipo, Formato: formato, Desde: desde, Hasta: hasta, ClaveOperacion: clave, Idioma: r.Header.Get("Accept-Language")})
	if err != nil {
		if errors.Is(err, ports.ErrInformeC12Invalido) {
			errorJSON(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		responderErrorAccesoCronos(w, err)
		return
	}
	if len(salida.Contenido) == 0 || salida.Recibo.ReciboRef == "" {
		errorJSON(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	switch salida.Formato {
	case domain.InformePDF:
		w.Header().Set("Content-Type", "application/pdf")
	case domain.InformeCSV:
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	default:
		errorJSON(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename=cronos-informe."+string(salida.Formato))
	w.Header().Set("X-VEC-Recibo-Ref", salida.Recibo.ReciboRef)
	_, _ = w.Write(salida.Contenido)
}
func rutaInformeC12(u *url.URL) (string, string, bool) {
	if u == nil || u.RawPath != "" {
		return "", "", false
	}
	if strings.HasPrefix(u.Path, RutaInformesCronosC12) {
		n := strings.TrimPrefix(u.Path, RutaInformesCronosC12)
		return "", n, n != "" && !strings.Contains(n, "/")
	}
	if !strings.HasPrefix(u.Path, RutaEquipoCronosC10) {
		return "", "", false
	}
	resto := strings.TrimPrefix(u.Path, RutaEquipoCronosC10)
	trozos := strings.Split(resto, "/")
	if len(trozos) != 3 || trozos[0] == "" || len(trozos[0]) > 128 || strings.ContainsAny(trozos[0], ".%\\") || trozos[1] != "informes" || trozos[2] == "" {
		return "", "", false
	}
	return trozos[0], trozos[2], true
}
func parametrosInformeC12(w http.ResponseWriter, r *http.Request) (string, string, string, error) {
	if r.URL.RawQuery != "" || r.Body == nil || r.ContentLength > 2048 {
		return "", "", "", ports.ErrInformeC12Invalido
	}
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return "", "", "", ports.ErrInformeC12Invalido
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2048)
	var p struct {
		Desde          string `json:"desde"`
		Hasta          string `json:"hasta"`
		ClaveOperacion string `json:"clave_operacion"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if dec.Decode(&p) != nil || dec.Decode(new(any)) != io.EOF {
		return "", "", "", ports.ErrInformeC12Invalido
	}
	desde, hasta, clave := p.Desde, p.Hasta, p.ClaveOperacion
	if !fechaCivilCanonica(desde) || !fechaCivilCanonica(hasta) || desde > hasta || clave == "" || len(clave) > 128 {
		return "", "", "", ports.ErrInformeC12Invalido
	}
	return desde, hasta, clave, nil
}

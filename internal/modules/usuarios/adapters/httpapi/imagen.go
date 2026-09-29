package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"vec-diputacion-granada/internal/modules/usuarios/application"
	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
)

// Rutas exactas de «Mi imagen», una por superficie. GET devuelve la elección
// y, si la hay, la foto del titular; POST la cambia con clave idempotente.
const RutaMiImagen = "/api/vec/usuarios/mi-imagen"
const RutaMiImagenAreaPersonal = "/api/vec/usuarios/area-personal/mi-imagen"

// La foto viaja en base64 dentro del JSON. El límite admite el fichero máximo
// codificado más los campos de la operación; nada mayor se lee.
var (
	limiteFotoBase64     = int64(base64.StdEncoding.EncodedLen(ports.TamanoMaximoFotoImagen))
	limitePeticionImagen = limiteFotoBase64 + 4<<10
	limiteSinFotoImagen  = int64(4 << 10)
	camposPeticionImagen = []string{"operacion", "version_esperada", "catalogo_version_ref", "clave_operacion", "eleccion"}
	camposEleccionImagen = []string{"modo", "paleta", "icono"}
)

type ResolverOrdenImagen interface {
	ResolverOrdenImagen(context.Context) (ports.OrdenImagen, error)
}

type ManejadorImagen struct {
	servicio *application.ServicioImagen
	orden    ResolverOrdenImagen
	auditor  AuditorDenegacion
	ruta     string
}

func NuevoManejadorImagenEnRuta(servicio *application.ServicioImagen, orden ResolverOrdenImagen, auditor AuditorDenegacion, ruta string) (*ManejadorImagen, error) {
	if servicio == nil || orden == nil || auditor == nil || (ruta != RutaMiImagen && ruta != RutaMiImagenAreaPersonal) {
		return nil, ports.ErrImagenNoDisponible
	}
	return &ManejadorImagen{servicio: servicio, orden: orden, auditor: auditor, ruta: ruta}, nil
}

type peticionImagenHTTP struct {
	Operacion          string                `json:"operacion"`
	VersionEsperada    uint64                `json:"version_esperada"`
	CatalogoVersionRef string                `json:"catalogo_version_ref"`
	ClaveOperacion     string                `json:"clave_operacion"`
	Eleccion           domain.EleccionImagen `json:"eleccion"`
	FotoBase64         json.RawMessage       `json:"foto_base64"`
}

// camposImagenValidos exige exactamente los campos de cada operación.
func camposImagenValidos(b []byte) bool {
	var top map[string]json.RawMessage
	if json.Unmarshal(b, &top) != nil {
		return false
	}
	var operacion string
	if len(top["operacion"]) == 0 || json.Unmarshal(top["operacion"], &operacion) != nil {
		return false
	}
	requeridos := camposPeticionImagen
	switch operacion {
	case ports.OperacionElegirImagen:
	case ports.OperacionSubirFotoImagen:
		requeridos = append(append([]string(nil), camposPeticionImagen...), "foto_base64")
	default:
		return false
	}
	if len(top) != len(requeridos) {
		return false
	}
	for _, clave := range requeridos {
		if len(top[clave]) == 0 || bytes.Equal(top[clave], []byte("null")) {
			return false
		}
	}
	var eleccion map[string]json.RawMessage
	if json.Unmarshal(top["eleccion"], &eleccion) != nil || len(eleccion) != len(camposEleccionImagen) {
		return false
	}
	for _, clave := range camposEleccionImagen {
		if len(eleccion[clave]) == 0 || bytes.Equal(eleccion[clave], []byte("null")) {
			return false
		}
	}
	return true
}

// decodificarFoto comprueba el tamaño codificado antes de reservar el búfer
// de los bytes; solo admite base64 estándar sin espacios ni prefijos.
func decodificarFoto(crudo json.RawMessage) ([]byte, error) {
	var texto string
	if json.Unmarshal(crudo, &texto) != nil || texto == "" {
		return nil, ports.ErrImagenPeticionInvalida
	}
	if int64(len(texto)) > limiteFotoBase64 {
		return nil, ports.ErrImagenFotoGrande
	}
	foto, err := base64.StdEncoding.Strict().DecodeString(texto)
	if err != nil || len(foto) == 0 {
		return nil, ports.ErrImagenFotoNoAdmitida
	}
	return foto, nil
}

func (m *ManejadorImagen) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if m == nil || m.servicio == nil || m.orden == nil || r == nil || r.URL == nil {
		responderErrorImagen(w, ports.ErrImagenNoDisponible)
		return
	}
	if r.URL.Path != m.ruta || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.URL.ForceQuery || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 {
		responderErrorImagen(w, ports.ErrImagenPeticionInvalida)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		responder(w, http.StatusMethodNotAllowed, map[string]any{"error": map[string]string{"codigo": "metodo_no_permitido", "clave_i18n": "api.usuarios.imagen.error.metodo_no_permitido"}})
		return
	}
	orden, err := m.orden.ResolverOrdenImagen(r.Context())
	if err != nil {
		m.responderError(w, r, err)
		return
	}
	if r.Method == http.MethodGet {
		if r.ContentLength > 0 || r.Body != nil && r.ContentLength != 0 {
			responderErrorImagen(w, ports.ErrImagenPeticionInvalida)
			return
		}
		vista, err := m.servicio.Consultar(r.Context(), orden)
		if err != nil {
			m.responderError(w, r, err)
			return
		}
		responder(w, http.StatusOK, map[string]any{"data": vista})
		return
	}
	if r.ContentLength > limitePeticionImagen {
		responderErrorImagen(w, ports.ErrImagenFotoGrande)
		return
	}
	if r.Body == nil || r.ContentLength < 0 || !esJSON(r.Header.Get("Content-Type")) {
		responderErrorImagen(w, ports.ErrImagenPeticionInvalida)
		return
	}
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limitePeticionImagen))
	if err != nil {
		var demasiado *http.MaxBytesError
		if errors.As(err, &demasiado) {
			responderErrorImagen(w, ports.ErrImagenFotoGrande)
			return
		}
		responderErrorImagen(w, ports.ErrImagenPeticionInvalida)
		return
	}
	defer clear(b)
	if !jsonSinClavesDuplicadas(b) || !camposImagenValidos(b) {
		responderErrorImagen(w, ports.ErrImagenPeticionInvalida)
		return
	}
	var peticion peticionImagenHTTP
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if dec.Decode(&peticion) != nil || dec.Decode(&struct{}{}) != io.EOF {
		responderErrorImagen(w, ports.ErrImagenPeticionInvalida)
		return
	}
	defer clear(peticion.FotoBase64)
	if peticion.Operacion == ports.OperacionElegirImagen && int64(len(b)) > limiteSinFotoImagen {
		responderErrorImagen(w, ports.ErrImagenPeticionInvalida)
		return
	}
	var foto []byte
	if peticion.Operacion == ports.OperacionSubirFotoImagen {
		if foto, err = decodificarFoto(peticion.FotoBase64); err != nil {
			responderErrorImagen(w, err)
			return
		}
		defer clear(foto)
	}
	recibo, err := m.servicio.Guardar(r.Context(), orden, ports.PeticionImagen{
		Operacion: peticion.Operacion, VersionEsperada: peticion.VersionEsperada, CatalogoVersionRef: peticion.CatalogoVersionRef,
		ClaveOperacion: peticion.ClaveOperacion, Eleccion: peticion.Eleccion, Foto: foto,
	})
	if err != nil {
		m.responderError(w, r, err)
		return
	}
	estado := http.StatusCreated
	if recibo.Replay {
		estado = http.StatusOK
	}
	responder(w, estado, map[string]any{"data": recibo})
}

func (m *ManejadorImagen) responderError(w http.ResponseWriter, r *http.Request, err error) {
	estado, _ := clasificarErrorImagen(err)
	if estado == http.StatusUnauthorized || estado == http.StatusForbidden {
		if m.auditor.AuditarDenegacionPreferencias(r.Context(), estado) != nil {
			responderErrorImagen(w, ports.ErrImagenNoDisponible)
			return
		}
	}
	responderErrorImagen(w, err)
}

func clasificarErrorImagen(err error) (int, string) {
	switch {
	case errors.Is(err, ports.ErrImagenNoAutenticado):
		return http.StatusUnauthorized, "no_autenticado"
	case errors.Is(err, ports.ErrImagenProhibido):
		return http.StatusForbidden, "prohibido"
	case errors.Is(err, ports.ErrImagenConflicto):
		return http.StatusConflict, "conflicto"
	case errors.Is(err, ports.ErrImagenFotoGrande):
		return http.StatusRequestEntityTooLarge, "foto_grande"
	case errors.Is(err, ports.ErrImagenFotoNoAdmitida):
		return http.StatusUnprocessableEntity, "foto_no_admitida"
	case errors.Is(err, ports.ErrImagenPeticionInvalida):
		return http.StatusUnprocessableEntity, "peticion_invalida"
	}
	return http.StatusServiceUnavailable, "no_disponible"
}

func responderErrorImagen(w http.ResponseWriter, err error) {
	estado, codigo := clasificarErrorImagen(err)
	responder(w, estado, map[string]any{"error": map[string]string{"codigo": codigo, "clave_i18n": "api.usuarios.imagen.error." + codigo}})
}

package httpinterno

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

const RutaExportacionServiciosPropios = "/api/interna/personal/mi-ficha/servicios/exportaciones"
const limiteCuerpoExportacionServiciosPropios = 4096

type ExportadorServiciosPropios interface {
	Exportar(context.Context, domain.SolicitudExportacionServiciosPropios) (ports.ResultadoExportacionServiciosPropios, error)
}
type ManejadorExportacionServiciosPropios struct {
	actor      ResolutorActorFichaPropia
	exportador ExportadorServiciosPropios
	registro   ports.RegistroIntentosExportacionServiciosPropios
}

func NuevoManejadorExportacionServiciosPropios(a ResolutorActorFichaPropia, e ExportadorServiciosPropios, r ports.RegistroIntentosExportacionServiciosPropios) (*ManejadorExportacionServiciosPropios, error) {
	if nuloRelacionesDietas(a) || nuloRelacionesDietas(e) || nuloRelacionesDietas(r) {
		return nil, domain.ErrExportacionServiciosPropiosNoDisponible
	}
	return &ManejadorExportacionServiciosPropios{a, e, r}, nil
}
func (m *ManejadorExportacionServiciosPropios) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if w == nil {
		return
	}
	for _, k := range []string{"Content-Disposition", "X-Content-SHA256", "X-Recibo-Ref"} {
		w.Header().Del(k)
	}
	if m == nil || r == nil || r.URL == nil || nuloRelacionesDietas(m.actor) || nuloRelacionesDietas(m.exportador) || nuloRelacionesDietas(m.registro) {
		responderFichaPropia(w, http.StatusServiceUnavailable, "no_disponible", nil)
		return
	}
	actor, err := m.actor.ResolverActorFichaPropia(r.Context())
	if err != nil || actor.Validar() != nil {
		responderFichaPropia(w, http.StatusServiceUnavailable, "no_disponible", nil)
		return
	}
	if m.registro.VerificarRegistroExportacionServiciosPropios(r.Context()) != nil {
		m.rechazar(w, r, http.StatusServiceUnavailable, "no_disponible", "no_disponible")
		return
	}
	if r.URL.Path != RutaExportacionServiciosPropios || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.URL.ForceQuery {
		m.rechazar(w, r, http.StatusNotFound, "no_encontrada", "entrada_invalida")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		m.rechazar(w, r, http.StatusMethodNotAllowed, "metodo_no_permitido", "entrada_invalida")
		return
	}
	if cabeceraLibreRelacionesDietas(r.Header) || r.Header.Get("Content-Encoding") != "" || len(r.TransferEncoding) != 0 || r.ContentLength > limiteCuerpoExportacionServiciosPropios {
		m.rechazar(w, r, http.StatusBadRequest, "peticion_invalida", "entrada_invalida")
		return
	}
	tipo, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || tipo != "application/json" || len(params) > 1 || params["charset"] != "" && !strings.EqualFold(params["charset"], "utf-8") || len(params) == 1 && params["charset"] == "" {
		m.rechazar(w, r, http.StatusBadRequest, "peticion_invalida", "entrada_invalida")
		return
	}
	entrada, err := decodificarExportacionServiciosPropios(r.Body)
	if err != nil {
		m.rechazar(w, r, http.StatusBadRequest, "peticion_invalida", "entrada_invalida")
		return
	}
	entrada.Actor = actor
	resultado, err := m.exportador.Exportar(r.Context(), entrada)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		estado, codigo := http.StatusServiceUnavailable, "no_disponible"
		if errors.Is(err, domain.ErrExportacionServiciosPropiosInvalida) {
			estado, codigo = http.StatusBadRequest, "peticion_invalida"
		}
		if errors.Is(err, domain.ErrExportacionServiciosPropiosDenegada) {
			estado, codigo = http.StatusForbidden, "acceso_denegado"
		}
		// El servicio ya registró su fallo después de cerrar la transacción.
		responderFichaPropia(w, estado, codigo, nil)
		return
	}
	h := sha256.Sum256(resultado.ContenidoCSV)
	if len(resultado.ContenidoCSV) == 0 || len(resultado.ContenidoCSV) > domain.LimiteBytesExportacionServiciosPropios || hex.EncodeToString(h[:]) != resultado.ContenidoSHA256 || resultado.Evidencia.ReciboRef != entrada.ReciboRef || resultado.Corte.VigenteEn != entrada.Corte.VigenteEn || !resultado.Corte.ConocidoEn.Equal(entrada.Corte.ConocidoEn) {
		m.rechazar(w, r, http.StatusServiceUnavailable, "no_disponible", "no_disponible")
		return
	}
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": resultado.NombreArchivo})
	// El servicio/serializer validan el nombre procedente del catálogo; este
	// límite de transporte también evita rutas o cabeceras aportadas por un doble.
	if disposition == "" || strings.ContainsAny(resultado.NombreArchivo, "\r\n\x00/\\") || !strings.HasSuffix(resultado.NombreArchivo, ".csv") {
		m.rechazar(w, r, http.StatusServiceUnavailable, "no_disponible", "no_disponible")
		return
	}
	for _, k := range []string{"Set-Cookie", "Access-Control-Allow-Origin", "Access-Control-Allow-Credentials", "Location", "Content-Encoding"} {
		w.Header().Del(k)
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", disposition)
	w.Header().Set("Content-Length", strconv.Itoa(len(resultado.ContenidoCSV)))
	w.Header().Set("X-Content-SHA256", resultado.ContenidoSHA256)
	w.Header().Set("X-Recibo-Ref", resultado.Evidencia.ReciboRef)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; frame-ancestors 'none'")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resultado.ContenidoCSV)
}
func (m *ManejadorExportacionServiciosPropios) rechazar(w http.ResponseWriter, r *http.Request, estado int, codigo, motivo string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), plazoarranque.Ampliar(2*time.Second))
	defer cancel()
	if m.registro.RegistrarIntentoExportacionServiciosPropios(ctx, ports.IntentoFichaPropia{Motivo: motivo}) != nil {
		estado, codigo = http.StatusServiceUnavailable, "no_disponible"
	}
	responderFichaPropia(w, estado, codigo, nil)
}
func decodificarExportacionServiciosPropios(body io.Reader) (domain.SolicitudExportacionServiciosPropios, error) {
	var cero domain.SolicitudExportacionServiciosPropios
	if body == nil {
		return cero, domain.ErrExportacionServiciosPropiosInvalida
	}
	b, err := io.ReadAll(io.LimitReader(body, limiteCuerpoExportacionServiciosPropios+1))
	if err != nil || len(b) == 0 || len(b) > limiteCuerpoExportacionServiciosPropios || !utf8.Valid(b) || jsonUnicoOrganizacionHistorica(b) != nil {
		return cero, domain.ErrExportacionServiciosPropiosInvalida
	}
	var top, corte map[string]json.RawMessage
	if json.Unmarshal(b, &top) != nil || len(top) != 3 || top["recibo_ref"] == nil || top["idioma"] == nil || json.Unmarshal(top["corte"], &corte) != nil || len(corte) != 2 || corte["vigente_en"] == nil || corte["conocido_en"] == nil {
		return cero, domain.ErrExportacionServiciosPropiosInvalida
	}
	for _, v := range top {
		if bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return cero, domain.ErrExportacionServiciosPropiosInvalida
		}
	}
	for _, v := range corte {
		if bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return cero, domain.ErrExportacionServiciosPropiosInvalida
		}
	}
	var dto struct {
		ReciboRef string                 `json:"recibo_ref"`
		Corte     domain.CorteEmpleadoB2 `json:"corte"`
		Idioma    string                 `json:"idioma"`
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&dto) != nil || d.Decode(new(any)) != io.EOF || dto.Corte.Validar() != nil {
		return cero, domain.ErrExportacionServiciosPropiosInvalida
	}
	return domain.SolicitudExportacionServiciosPropios{ReciboRef: dto.ReciboRef, Corte: dto.Corte, Idioma: dto.Idioma}, nil
}

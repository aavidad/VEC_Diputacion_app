package meritoshttp

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
	"time"

	"vec-diputacion-granada/internal/modules/meritos/application"
	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const RutaDeclaracionesCartera = "/api/vec/meritos/cartera/declaraciones"

// La composición aporta la sesión nominal y el motivo gobernado. El cuerpo
// nunca contiene persona, perfil, estado, revisión ni evidencias acreditadas.
type ProveedorDeclaracionCartera interface {
	PrepararDeclaracionCartera(*http.Request) (application.Solicitud, error)
}

type DeclaradorCartera interface {
	Declarar(context.Context, application.Solicitud) (ports.Recibo, error)
}

type HandlerDeclaracionesCartera struct {
	proveedor ProveedorDeclaracionCartera
	servicio  DeclaradorCartera
	reloj     vecports.Reloj
}

type entradaDeclaracionCartera struct {
	Tipo         string `json:"tipo"`
	Denominacion string `json:"denominacion"`
	Desde        string `json:"fecha_desde"`
	Hasta        string `json:"fecha_hasta,omitempty"`
	Horas        *int   `json:"horas,omitempty"`
	DeclaradaEn  string `json:"declarada_en"`
	Clave        string `json:"clave_idempotencia"`
}

func NuevoHandlerDeclaracionesCartera(p ProveedorDeclaracionCartera, s DeclaradorCartera, r vecports.Reloj) (*HandlerDeclaracionesCartera, error) {
	if dependenciaConsultaNula(p) || dependenciaConsultaNula(s) || dependenciaConsultaNula(r) {
		return nil, ports.ErrCarteraNoDisponible
	}
	return &HandlerDeclaracionesCartera{proveedor: p, servicio: s, reloj: r}, nil
}

func (h *HandlerDeclaracionesCartera) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if r == nil || r.URL == nil || r.URL.Path != RutaDeclaracionesCartera || r.URL.EscapedPath() != RutaDeclaracionesCartera {
		responderErrorCartera(w, http.StatusNotFound, "meritos.error.ruta_no_encontrada")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorCartera(w, http.StatusMethodNotAllowed, "meritos.error.metodo_no_admitido")
		return
	}
	if h == nil || dependenciaConsultaNula(h.proveedor) || dependenciaConsultaNula(h.servicio) || dependenciaConsultaNula(h.reloj) {
		responderErrorCartera(w, http.StatusServiceUnavailable, "meritos.error.cartera_no_disponible")
		return
	}
	solicitud, err := h.proveedor.PrepararDeclaracionCartera(r)
	if err != nil {
		responderFalloCartera(w, err)
		return
	}
	operacionCtx, cancelar := context.WithTimeout(context.WithoutCancel(r.Context()), 15*time.Second)
	defer cancelar()
	entrada, err := leerDeclaracionCartera(w, r)
	declarada, errFecha := time.Parse(time.RFC3339Nano, entrada.DeclaradaEn)
	if err != nil || errFecha != nil || declarada.After(h.reloj.Ahora()) || r.URL.RawQuery != "" || r.URL.ForceQuery {
		// El servicio registra el rechazo bajo la misma identidad nominal.
		_, fallo := h.servicio.Declarar(operacionCtx, solicitud)
		if !errors.Is(fallo, application.ErrSolicitud) {
			responderFalloCartera(w, fallo)
			return
		}
		responderErrorCartera(w, http.StatusBadRequest, "meritos.error.solicitud_invalida")
		return
	}
	// Referencia estable y opaca para reintentar incluso tras reiniciar. La clave
	// aleatoria identifica la operación; la persona procede solo de la sesión.
	digest := sha256.Sum256([]byte("vec.meritos.cartera.declaracion.v1:" + solicitud.Contexto.Contexto.PersonaRef + ":" + entrada.Clave))
	hechoRef := "ref:" + hex.EncodeToString(digest[:])
	solicitud.ClaveIdempotencia, solicitud.VersionEsperada = entrada.Clave, 0
	solicitud.FechaCorte = declarada.UTC().Format(time.DateOnly)
	solicitud.Hecho = domain.Hecho{Referencia: hechoRef, PersonaRef: solicitud.Contexto.Contexto.PersonaRef,
		Version: 1, Tipo: entrada.Tipo, ConceptoRef: "tipo:" + entrada.Tipo, Denominacion: entrada.Denominacion, Horas: entrada.Horas,
		Procedencia: domain.Procedencia{FuenteRef: "meritos:declaracion-personal", Version: "1", HechoOrigenRef: hechoRef, CapturadaEn: entrada.DeclaradaEn},
		Vigencia:    domain.Vigencia{Desde: entrada.Desde, Hasta: entrada.Hasta}, Estado: domain.Declarado, Evidencias: []vec.ReferenciaDocumento{}}
	recibo, err := h.servicio.Declarar(operacionCtx, solicitud)
	if err != nil {
		switch {
		case errors.Is(err, application.ErrSolicitud):
			responderErrorCartera(w, http.StatusUnprocessableEntity, "meritos.error.solicitud_invalida")
		case errors.Is(err, ports.ErrConflictoVersion), errors.Is(err, ports.ErrClaveReutilizada):
			responderErrorCartera(w, http.StatusConflict, "meritos.error.conflicto_version")
		case errors.Is(err, vec.ErrAutorizacionDenegada):
			responderErrorCartera(w, http.StatusForbidden, "meritos.error.cartera_denegada")
		default:
			responderFalloCartera(w, err)
		}
		return
	}
	contenido, err := json.Marshal(struct {
		Data ports.Recibo `json:"data"`
	}{recibo})
	if err != nil {
		responderErrorCartera(w, http.StatusServiceUnavailable, "meritos.error.cartera_no_disponible")
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(contenido)
}

func leerDeclaracionCartera(w http.ResponseWriter, r *http.Request) (entradaDeclaracionCartera, error) {
	var entrada entradaDeclaracionCartera
	tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || tipo != "application/json" || r.Body == nil {
		return entrada, application.ErrSolicitud
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 16<<10))
	if err != nil {
		return entrada, application.ErrSolicitud
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&entrada) != nil || decoder.Decode(new(any)) != io.EOF {
		return entradaDeclaracionCartera{}, application.ErrSolicitud
	}
	return entrada, nil
}

package httpinterno

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	bolsapuertos "vec-diputacion-granada/internal/modules/bolsa/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
)

const RutaJustificanteRenunciaRRHH = "/api/vec/bolsa/justificantes-renuncia/rrhh"
const limiteJustificanteRenunciaRRHH = 16 << 20

var selectorJustificanteRenunciaRRHH = regexp.MustCompile(`^[A-Za-z0-9:_-]{1,256}$`)

type IdentidadJustificanteRenunciaRRHH struct {
	Actor          vec.ResultadoContextoActorRegistradoV2
	Vinculo        vec.VinculoAutenticacionActorV2
	CorrelacionRef string
}

type ProveedorIdentidadJustificanteRenunciaRRHH interface {
	PrepararIdentidadJustificanteRenunciaRRHH(*http.Request) (IdentidadJustificanteRenunciaRRHH, error)
}

type ConsultorJustificanteRenunciaRRHH interface {
	AbrirJustificanteRenunciaRRHH(context.Context, IdentidadJustificanteRenunciaRRHH, string, string) (io.ReadCloser, error)
}

type handlerJustificanteRenunciaRRHH struct {
	identidad ProveedorIdentidadJustificanteRenunciaRRHH
	consulta  ConsultorJustificanteRenunciaRRHH
}

func NuevoHandlerJustificanteRenunciaRRHHNoDisponible() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		cabecerasJustificanteRenunciaRRHH(w)
		responderOperacion(w, http.StatusServiceUnavailable, "servicio_no_disponible")
	})
}

func NuevoHandlerJustificanteRenunciaRRHH(identidad ProveedorIdentidadJustificanteRenunciaRRHH,
	consulta ConsultorJustificanteRenunciaRRHH) (http.Handler, error) {
	if identidad == nil || consulta == nil {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	return &handlerJustificanteRenunciaRRHH{identidad: identidad, consulta: consulta}, nil
}

func cabecerasJustificanteRenunciaRRHH(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "private, no-store, max-age=0")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
}

func (h *handlerJustificanteRenunciaRRHH) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cabecerasJustificanteRenunciaRRHH(w)
	if r == nil || r.URL == nil || r.URL.Path != RutaJustificanteRenunciaRRHH ||
		r.URL.RawPath != "" || r.URL.EscapedPath() != RutaJustificanteRenunciaRRHH || r.URL.ForceQuery {
		responderOperacion(w, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		responderOperacion(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if h == nil || h.identidad == nil || h.consulta == nil {
		responderOperacion(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	identidad, err := h.identidad.PrepararIdentidadJustificanteRenunciaRRHH(r)
	if err != nil || identidad.Actor.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.Actor) != nil ||
		!vec.ReferenciaCorrelacionAutorizacionV2Valida(identidad.CorrelacionRef) {
		responderOperacion(w, http.StatusForbidden, "acceso_denegado")
		return
	}
	if r.ContentLength != 0 || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 {
		responderOperacion(w, http.StatusForbidden, "acceso_denegado")
		return
	}
	valores, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(valores) != 2 || len(valores["bolsa"]) != 1 || len(valores["participacion"]) != 1 ||
		!selectorJustificanteRenunciaRRHH.MatchString(valores.Get("bolsa")) ||
		!selectorJustificanteRenunciaRRHH.MatchString(valores.Get("participacion")) {
		responderOperacion(w, http.StatusForbidden, "acceso_denegado")
		return
	}
	original, err := h.consulta.AbrirJustificanteRenunciaRRHH(r.Context(), identidad,
		valores.Get("bolsa"), valores.Get("participacion"))
	if err != nil || original == nil {
		if original != nil {
			if cierre := original.Close(); cierre != nil {
				slog.Warn("justificante_renuncia_rrhh_cierre_fallido")
			}
		}
		estado := http.StatusServiceUnavailable
		if errors.Is(err, docports.ErrSubidaLocalDenegada) || errors.Is(err, docports.ErrAccesoDenegado) || errors.Is(err, docports.ErrNoEncontrado) ||
			errors.Is(err, bolsapuertos.ErrJustificanteRenunciaDenegado) || errors.Is(err, vec.ErrAutorizacionDenegada) {
			estado = http.StatusForbidden
		}
		codigo := "servicio_no_disponible"
		if estado == http.StatusForbidden {
			codigo = "acceso_denegado"
		}
		responderOperacion(w, estado, codigo)
		return
	}
	contenido, errorLectura := io.ReadAll(io.LimitReader(original, limiteJustificanteRenunciaRRHH+1))
	defer clear(contenido)
	errorCierre := original.Close()
	if errorLectura != nil || errorCierre != nil || len(contenido) == 0 ||
		len(contenido) > limiteJustificanteRenunciaRRHH || !bytes.HasPrefix(contenido, []byte("%PDF-")) {
		responderOperacion(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="justificante.pdf"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(contenido); err != nil {
		slog.Warn("justificante_renuncia_rrhh_respuesta_interrumpida")
	}
}

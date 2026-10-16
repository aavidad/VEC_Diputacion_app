package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// La ruta documental tiene sesión y permiso propios. El rechazo de frontera
// conserva únicamente ruta y motivo, en la bitácora común de CT210/CT222.
func componerRaizConJustificanteRenunciaRRHH(raiz http.Handler, ruta *vechttp.RutaExacta,
	auditoria vecports.RegistradorAuditoriaFronteraRutaExacta) http.Handler {
	if ruta == nil || ruta.Manejador == nil {
		return raiz
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil || r.URL.Path != ruta.Ruta {
			raiz.ServeHTTP(w, r)
			return
		}
		salida := &respuestaAuditadaJustificanteRRHH{ResponseWriter: w, peticion: r,
			ruta: ruta.Ruta, auditoria: auditoria}
		ruta.Manejador.ServeHTTP(salida, r)
	})
}

type respuestaAuditadaJustificanteRRHH struct {
	http.ResponseWriter
	peticion       *http.Request
	ruta           string
	auditoria      vecports.RegistradorAuditoriaFronteraRutaExacta
	escrita        bool
	falloAuditoria bool
	alRechazar     func(int) error
}

func (w *respuestaAuditadaJustificanteRRHH) WriteHeader(estado int) {
	if w.escrita {
		return
	}
	w.escrita = true
	if estado >= 400 {
		if err := w.registrar(estado); err != nil {
			// Nunca registrar el error original: puede incluir un DSN o datos.
			causa := "registro_fallido"
			switch {
			case errors.Is(err, vecports.ErrIntentoAuditoriaNoDisponible):
				causa = "dependencia_ausente"
			case errors.Is(err, context.DeadlineExceeded):
				causa = "plazo_agotado"
			case errors.Is(err, context.Canceled):
				causa = "peticion_cancelada"
			}
			slog.Warn("justificante_frontera_auditoria_no_disponible", "etapa", "registrar_rechazo", "causa", causa)
			w.falloAuditoria = true
			w.Header().Del("Content-Length")
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.ResponseWriter.WriteHeader(http.StatusServiceUnavailable)
			if _, err := w.ResponseWriter.Write([]byte(`{"error":{"codigo":"servicio_no_disponible"}}`)); err != nil {
				slog.Warn("justificante_frontera_respuesta_interrumpida")
			}
			return
		}
	}
	w.ResponseWriter.WriteHeader(estado)
}

func (w *respuestaAuditadaJustificanteRRHH) Write(b []byte) (int, error) {
	if !w.escrita {
		w.WriteHeader(http.StatusOK)
	}
	if w.falloAuditoria {
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}

func (w *respuestaAuditadaJustificanteRRHH) registrar(estado int) error {
	if w.alRechazar != nil {
		return w.alRechazar(estado)
	}
	if w.auditoria == nil || w.peticion == nil {
		return vecports.ErrIntentoAuditoriaNoDisponible
	}
	ctx, cancelar := context.WithTimeout(context.WithoutCancel(w.peticion.Context()), 5*time.Second)
	defer cancelar()
	c, err := vec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return err
	}
	ref, err := c.ValorCanonico()
	if err != nil {
		return err
	}
	motivo := vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado
	if estado == http.StatusUnauthorized {
		motivo = vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida
	} else if estado >= 500 {
		motivo = vecports.MotivoAuditoriaFronteraRutaExactaServicioNoDisponible
	}
	o := vecports.OrdenAuditoriaFronteraRutaExacta{CorrelacionRef: "corr_" + strings.TrimPrefix(ref, "correlacion_"),
		Motivo: motivo, Superficie: vecports.SuperficieAuditoriaFronteraRutaExactaDocumentosBolsaRRHH, Ruta: w.ruta}
	if err := o.Validar(); err != nil {
		return err
	}
	return w.auditoria.RegistrarAuditoriaFronteraRutaExacta(ctx, o)
}

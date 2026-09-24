package interna

import (
	"context"
	"errors"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/app/composicion/internagobierno"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// enrutadorPersonalB2 conserva el manejador CT y añade cuatro
// superficies B2 explícitas. La consulta de la ficha valida su emp_ref en el
// adaptador; ninguna ruta desconocida entra al módulo.
type enrutadorPersonalB2 struct {
	ct, ficha, vacantes, alta, hecho http.Handler
	autoridad                        httpapi.AutoridadRutasExactas
	auditoria                        vecports.RegistradorAuditoriaFronteraRutaExacta
}

func nuevoEnrutadorPersonalB2(ct, ficha, vacantes, alta, hecho http.Handler, autoridad httpapi.AutoridadRutasExactas, auditoria vecports.RegistradorAuditoriaFronteraRutaExacta) (http.Handler, error) {
	if manejadorNulo(ct) || manejadorNulo(ficha) || manejadorNulo(vacantes) || manejadorNulo(alta) || manejadorNulo(hecho) ||
		interfazNulaIdentidadOffline(autoridad) || interfazNulaIdentidadOffline(auditoria) {
		return nil, ErrAPIInternaNoDisponible
	}
	return &enrutadorPersonalB2{ct, ficha, vacantes, alta, hecho, autoridad, auditoria}, nil
}

func (e *enrutadorPersonalB2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e == nil || r == nil || r.URL == nil {
		responderPuenteSeguimiento(w, http.StatusServiceUnavailable)
		return
	}
	if r.URL.Path == httpct.RutaConsultaSeguimientoV2 {
		e.ct.ServeHTTP(w, r)
		return
	}
	if !internagobierno.RutaInternaGobernada(r.URL.Path) || r.URL.RawPath != "" ||
		r.URL.Opaque != "" || r.URL.EscapedPath() != r.URL.Path {
		responderPuenteSeguimiento(w, http.StatusNotFound)
		return
	}
	if err := e.autoridad.AutorizarRutaExacta(r.Context(), r.URL.Path); err != nil {
		estado := http.StatusServiceUnavailable
		motivo := vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado
		switch {
		case errors.Is(err, httpapi.ErrAutenticacionRutaExactaRequerida):
			estado, motivo = http.StatusUnauthorized, vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida
		case errors.Is(err, httpapi.ErrAccesoRutaExactaDenegado):
			estado = http.StatusForbidden
		}
		if estado != http.StatusServiceUnavailable {
			rutaAuditoria := r.URL.Path
			if r.URL.Path != httpapi.RutaVacantesEmpleadoB2 && r.URL.Path != "/api/vec/personal/empleados" && r.URL.Path != "/api/vec/personal/hechos" {
				rutaAuditoria = "/api/vec/personal/empleados/{emp_ref}"
			}
			orden := vecports.OrdenAuditoriaFronteraRutaExacta{
				CorrelacionRef: correlacionDenegacionSeguimiento(), Motivo: motivo,
				Superficie: vecports.SuperficieAuditoriaFronteraRutaExactaPersonal,
				Ruta:       rutaAuditoria,
			}
			ctx, cancelar := context.WithTimeout(context.WithoutCancel(r.Context()), 250*time.Millisecond)
			defer cancelar()
			if orden.Validar() != nil || e.auditoria.RegistrarAuditoriaFronteraRutaExacta(ctx, orden) != nil {
				estado = http.StatusServiceUnavailable
			}
		}
		responderPuenteSeguimiento(w, estado)
		return
	}
	if r.URL.Path == httpapi.RutaVacantesEmpleadoB2 {
		e.vacantes.ServeHTTP(w, r)
		return
	}
	if r.URL.Path == "/api/vec/personal/empleados" {
		e.alta.ServeHTTP(w, r)
		return
	}
	if r.URL.Path == "/api/vec/personal/hechos" {
		e.hecho.ServeHTTP(w, r)
		return
	}
	e.ficha.ServeHTTP(w, r)
}

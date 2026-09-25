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

// enrutadorPersonalB2 conserva el manejador CT y añade seis
// superficies B2 explícitas. La consulta de la ficha valida su emp_ref en el
// adaptador; ninguna ruta desconocida entra al módulo.
type enrutadorPersonalB2 struct {
	ct, ficha, vacantes, empleados, alta, hecho, catalogos http.Handler
	autoridad                                              httpapi.AutoridadRutasExactas
	auditoria                                              vecports.RegistradorAuditoriaFronteraRutaExacta
	personalActivo                                         bool
}

func nuevoEnrutadorPersonalB2(ct, ficha, vacantes, empleados, alta, hecho, catalogos http.Handler, autoridad httpapi.AutoridadRutasExactas, auditoria vecports.RegistradorAuditoriaFronteraRutaExacta) (http.Handler, error) {
	if manejadorNulo(ct) || interfazNulaIdentidadOffline(autoridad) || interfazNulaIdentidadOffline(auditoria) {
		return nil, ErrAPIInternaNoDisponible
	}
	presentes := 0
	for _, h := range []http.Handler{ficha, vacantes, empleados, alta, hecho, catalogos} {
		if !manejadorNulo(h) {
			presentes++
		}
	}
	if presentes != 0 && presentes != 6 {
		return nil, ErrAPIInternaNoDisponible
	}
	return &enrutadorPersonalB2{ct, ficha, vacantes, empleados, alta, hecho, catalogos, autoridad, auditoria, presentes == 6}, nil
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
	if !e.personalActivo {
		responderPuenteSeguimiento(w, http.StatusNotFound)
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
			rutaAuditoria := httpapi.RutaAuditoriaRegistroEmpleadoB2(r.URL.Path)
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
	if r.URL.Path == httpapi.RutaEmpleadosOrganismoB2 {
		e.empleados.ServeHTTP(w, r)
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
	if r.URL.Path == httpapi.RutaCatalogosRegistroEmpleadoB2 {
		e.catalogos.ServeHTTP(w, r)
		return
	}
	e.ficha.ServeHTTP(w, r)
}

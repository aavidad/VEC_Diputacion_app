package interna

import (
	"context"
	"net/http"

	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	vecapp "vec-diputacion-granada/internal/vec/application"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// extractorAsercionInstitucional pertenece al proveedor de la pasarela. No
// interpreta cabeceras, Bearer, cookies ni PEM reenviado en esta composición.
// Su contrato de transporte y la implementación institucional siguen
// pendientes; ningún adaptador de desarrollo satisface el arranque.
type extractorAsercionInstitucional interface {
	ExtraerAsercionProtegida(*http.Request) ([]byte, error)
}

type proveedoresLecturaCT struct {
	identidad      *httpseguridad.ServicioIdentidad
	extractor      extractorAsercionInstitucional
	servicioVEC    *vecapp.Service
	autoridadRutas httpapi.AutoridadRutasExactas
	auditoriaRutas vecports.RegistradorAuditoriaFronteraRutaExacta
	actor          contextoActorLecturaCT
	ambitos        resolutorAmbitoConsultaRRHHRegistrado
	consultas      dependenciasLecturasRRHH
	recursos       []recursoCerrableAplicacionInterna
}

// obtenerProveedoresLecturaCT es la única entrada de infraestructura para la
// raíz productiva. No existen todavía verificador/evaluador institucionales,
// contrato de transporte ni seudonimizador HSM; por ello no construye dobles
// ni devuelve una composición parcialmente operativa.
func obtenerProveedoresLecturaCT(context.Context, Configuracion) (proveedoresLecturaCT, error) {
	return proveedoresLecturaCT{}, &ErrorDependenciasFaltantes{
		faltantes: append([]Dependencia(nil), dependenciasLecturaCT[:]...),
	}
}

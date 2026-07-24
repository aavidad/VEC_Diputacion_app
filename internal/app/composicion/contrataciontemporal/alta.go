// Package contrataciontemporal compone la vertical interna de alta sin
// convertir configuración, HTTP o PostgreSQL en autoridad de dominio.
package contrataciontemporal

import (
	"errors"
	"net/http"
	"reflect"

	"github.com/jackc/pgx/v5/pgxpool"

	adaptadorhttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	adaptadorpostgres "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var ErrComposicionAltaNoDisponible = errors.New(
	"composicion contratacion temporal: alta no disponible",
)

// DependenciaAlta identifica una frontera obligatoria sin revelar su
// configuración, credenciales o estado operativo.
type DependenciaAlta string

const (
	DependenciaAutoridadCanal DependenciaAlta = "autoridad_canal"
	DependenciaContextoActor  DependenciaAlta = "contexto_actor"
	DependenciaFlujo          DependenciaAlta = "flujo"
	DependenciaHuellaHMAC     DependenciaAlta = "huella_hmac"
	DependenciaAmbitoHMAC     DependenciaAlta = "ambito_hmac"
	DependenciaMotivos        DependenciaAlta = "motivos"
	DependenciaCorrelaciones  DependenciaAlta = "correlaciones"
	DependenciaReferencias    DependenciaAlta = "referencias"
	DependenciaPDPV3          DependenciaAlta = "pdp_v3"
	DependenciaMaterialVEC    DependenciaAlta = "material_vec_ad_3"
	DependenciaReloj          DependenciaAlta = "reloj"
	DependenciaPostgreSQL     DependenciaAlta = "postgresql_altas"
)

var ordenDependenciasAlta = [...]DependenciaAlta{
	DependenciaAutoridadCanal,
	DependenciaContextoActor,
	DependenciaFlujo,
	DependenciaHuellaHMAC,
	DependenciaAmbitoHMAC,
	DependenciaMotivos,
	DependenciaCorrelaciones,
	DependenciaReferencias,
	DependenciaPDPV3,
	DependenciaMaterialVEC,
	DependenciaReloj,
	DependenciaPostgreSQL,
}

// DependenciasAlta recibe capacidades ya construidas por sus conectores
// propietarios. No acepta DSN, claves HMAC, cookies ni cabeceras de identidad.
type DependenciasAlta struct {
	AutoridadCanal adaptadorhttp.AutoridadContextoCanal
	Contextos      ports.ResolutorContextoAutorizacionAltaV3
	Flujos         ports.ResolutorFlujoAlta
	Huellas        ports.DerivadorHuellaAlta
	Ambitos        ports.SelladorAmbitoIdempotencia
	Motivos        ports.ResolutorMotivoAutorizacionAltaV3
	Correlaciones  puertosvec.GeneradorReferenciasAutorizacionV2
	Referencias    ports.GeneradorReferenciasAlta
	Autorizador    puertosvec.AutorizadorSolicitudLigadaV3
	Material       ports.ProveedorMaterialConfirmacionAlta
	Reloj          ports.Reloj
	PoolAltas      *pgxpool.Pool
}

// ErrorDependenciasAltaFaltantes conserva un inventario tipado y redactado.
// No debe serializarse como respuesta HTTP.
type ErrorDependenciasAltaFaltantes struct {
	faltantes []DependenciaAlta
}

func (e *ErrorDependenciasAltaFaltantes) Error() string {
	return ErrComposicionAltaNoDisponible.Error()
}

func (e *ErrorDependenciasAltaFaltantes) Unwrap() error {
	return ErrComposicionAltaNoDisponible
}

func (e *ErrorDependenciasAltaFaltantes) Faltantes() []DependenciaAlta {
	if e == nil {
		return nil
	}
	return append([]DependenciaAlta(nil), e.faltantes...)
}

func (e *ErrorDependenciasAltaFaltantes) Falta(
	dependencia DependenciaAlta,
) bool {
	if e == nil {
		return false
	}
	for _, faltante := range e.faltantes {
		if faltante == dependencia {
			return true
		}
	}
	return false
}

// NuevaAPIAlta construye una lista positiva con una única ruta. PostgreSQL
// estabiliza la candidatura y confirma el agregado con la misma identidad de
// ejecución, pero ambos adaptadores siguen separados por puertos.
func NuevaAPIAlta(dependencias DependenciasAlta) (http.Handler, error) {
	if faltantes := dependencias.faltantes(); len(faltantes) != 0 {
		return nil, &ErrorDependenciasAltaFaltantes{faltantes: faltantes}
	}
	candidaturas, err :=
		adaptadorpostgres.NuevoResolutorCandidaturaAltaPostgreSQL(
			dependencias.PoolAltas,
		)
	if err != nil {
		return nil, ErrComposicionAltaNoDisponible
	}
	transaccion, err := adaptadorpostgres.NuevoTransaccionAltasPostgreSQL(
		dependencias.PoolAltas,
		dependencias.Material,
	)
	if err != nil {
		return nil, ErrComposicionAltaNoDisponible
	}
	servicio, err := application.NuevoServicioRegistroSolicitud(
		dependencias.Contextos,
		dependencias.Flujos,
		dependencias.Huellas,
		dependencias.Ambitos,
		dependencias.Motivos,
		dependencias.Correlaciones,
		dependencias.Referencias,
		candidaturas,
		adaptadorpostgres.NuevoProyectorEfectoAltaV2(),
		dependencias.Autorizador,
		dependencias.Reloj,
		transaccion,
	)
	if err != nil {
		return nil, ErrComposicionAltaNoDisponible
	}
	manejador, err := adaptadorhttp.NuevoManejadorAlta(
		dependencias.AutoridadCanal,
		servicio,
		dependencias.Reloj,
	)
	if err != nil {
		return nil, ErrComposicionAltaNoDisponible
	}
	mux := http.NewServeMux()
	mux.Handle(adaptadorhttp.RutaAltaSolicitudes, manejador)
	return mux, nil
}

func (d DependenciasAlta) faltantes() []DependenciaAlta {
	faltantes := make([]DependenciaAlta, 0, len(ordenDependenciasAlta))
	comprobar := func(nombre DependenciaAlta, valor any) {
		if dependenciaNula(valor) {
			faltantes = append(faltantes, nombre)
		}
	}
	comprobar(DependenciaAutoridadCanal, d.AutoridadCanal)
	comprobar(DependenciaContextoActor, d.Contextos)
	comprobar(DependenciaFlujo, d.Flujos)
	comprobar(DependenciaHuellaHMAC, d.Huellas)
	comprobar(DependenciaAmbitoHMAC, d.Ambitos)
	comprobar(DependenciaMotivos, d.Motivos)
	comprobar(DependenciaCorrelaciones, d.Correlaciones)
	comprobar(DependenciaReferencias, d.Referencias)
	comprobar(DependenciaPDPV3, d.Autorizador)
	comprobar(DependenciaMaterialVEC, d.Material)
	comprobar(DependenciaReloj, d.Reloj)
	comprobar(DependenciaPostgreSQL, d.PoolAltas)
	return faltantes
}

func dependenciaNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

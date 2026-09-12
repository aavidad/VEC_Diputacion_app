package contrataciontemporal

import (
	"errors"
	"net/http"
	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
)

var ErrRutasContratacionTemporalInvalidas = errors.New(
	"composicion interna: rutas de contratacion temporal invalidas",
)

// DependenciasRutas solo acepta casos de uso y autoridades ya constituidos.
// La identidad corporativa, PostgreSQL y los proveedores criptograficos
// pertenecen a fronteras anteriores de la raiz de composicion.
type DependenciasRutas struct {
	IncorporacionV2                 *inc.ServidorV2PostgreSQL
	AutoridadAlta                   httpinterno.AutoridadContextoCanal
	EjecutorAlta                    httpinterno.EjecutorAlta
	Reloj                           ports.Reloj
	AutoridadAnalisis               httpinterno.AutoridadContextoCanalAnalisisRRHH
	EjecutorAnalisis                httpinterno.EjecutorAnalisisRRHH
	AutoridadCobertura              httpinterno.AutoridadContextoCanalCobertura
	Presentador                     httpinterno.PresentadorPropuestaCobertura
	Decisor                         httpinterno.EjecutorDecisionCobertura
	ConsultorResultado              httpinterno.ConsultorResultadoCobertura
	ConsultorCuadroRRHH             httpinterno.ConsultorCuadroRRHH
	ConsultorDetalleRRHH            httpinterno.ConsultorDetalleRRHH
	BorradorRRHH                    ports.RenderizadorBorradorRRHH
	BorradorRRHHDOCX                httpinterno.RenderizadorBorradorRRHHDOCX
	EjecutorSeleccion               httpinterno.EjecutorSeleccionLlamamiento
	AutoridadPropuestaFormalizacion httpinterno.AutoridadServidorPropuestaFormalizacion
	EjecutorPropuestaFormalizacion  httpinterno.EjecutorPropuestaFormalizacion
	AutoridadCierreAdministrativo   httpinterno.AutoridadServidorCierreAdministrativo
	EjecutorCierreAdministrativo    httpinterno.EjecutorCierreAdministrativo
	AutoridadAsignacion             httpinterno.AutoridadContextoCanalAsignacion
	EjecutorAsignacion              httpinterno.EjecutorAsignacion
	AutoridadInformeJuridico        httpinterno.AutoridadContextoCanalInformeJuridico
	EjecutorInformeJuridico         httpinterno.EjecutorInformeJuridico
	AutoridadFiscalizacion          httpinterno.AutoridadContextoCanalFiscalizacion
	EjecutorFiscalizacion           httpinterno.EjecutorFiscalizacion
	// Subsanacion se registra sólo cuando la composición ha aportado las dos
	// dependencias reales. La ausencia no publica una ruta parcialmente capaz.
	AutoridadSubsanacionReparos httpinterno.AutoridadContextoCanalSubsanacionReparos
	EjecutorSubsanacionReparos  httpinterno.EjecutorSubsanacionReparos
}

// NuevasRutas construye el conjunto de forma atomica. No devuelve una API
// parcial si falta una dependencia o falla un adaptador.
func NuevasRutas(
	dependencias DependenciasRutas,
) ([]httpapi.RutaExacta, error) {
	alta, err := httpinterno.NuevoManejadorAlta(
		dependencias.AutoridadAlta,
		dependencias.EjecutorAlta,
		dependencias.Reloj,
	)
	if err != nil {
		return nil, ErrRutasContratacionTemporalInvalidas
	}
	cobertura, err := httpinterno.NuevoManejadorCobertura(
		dependencias.AutoridadCobertura,
		dependencias.Presentador,
		dependencias.Decisor,
	)
	if err != nil {
		return nil, ErrRutasContratacionTemporalInvalidas
	}
	resultado, err := httpinterno.NuevoManejadorResultadoCobertura(
		dependencias.ConsultorResultado,
	)
	if err != nil {
		return nil, ErrRutasContratacionTemporalInvalidas
	}
	analisis, err := httpinterno.NuevoManejadorAnalisisRRHH(
		dependencias.AutoridadAnalisis,
		dependencias.EjecutorAnalisis,
	)
	if err != nil {
		return nil, ErrRutasContratacionTemporalInvalidas
	}
	seleccion, err := httpinterno.NuevoManejadorSeleccionLlamamiento(
		dependencias.EjecutorSeleccion,
	)
	if err != nil {
		return nil, ErrRutasContratacionTemporalInvalidas
	}
	propuesta, err := httpinterno.NuevoManejadorPropuestaFormalizacion(
		dependencias.AutoridadPropuestaFormalizacion,
		dependencias.EjecutorPropuestaFormalizacion,
	)
	if err != nil {
		return nil, ErrRutasContratacionTemporalInvalidas
	}
	cierreAdministrativo, err := httpinterno.NuevoManejadorCierreAdministrativo(
		dependencias.AutoridadCierreAdministrativo,
		dependencias.EjecutorCierreAdministrativo,
	)
	if err != nil {
		return nil, ErrRutasContratacionTemporalInvalidas
	}
	cuadroRRHH, err := httpinterno.NuevoManejadorConsultaCuadroRRHH(
		dependencias.ConsultorCuadroRRHH,
	)
	if err != nil {
		return nil, ErrRutasContratacionTemporalInvalidas
	}
	var detalleRRHH http.Handler
	if dependencias.BorradorRRHHDOCX != nil {
		detalleRRHH, err = httpinterno.NuevoManejadorConsultaDetalleRRHHConDOCX(
			dependencias.ConsultorDetalleRRHH, dependencias.BorradorRRHH,
			dependencias.BorradorRRHHDOCX,
		)
	} else {
		var renderizadoresBorrador []ports.RenderizadorBorradorRRHH
		if dependencias.BorradorRRHH != nil {
			renderizadoresBorrador = append(renderizadoresBorrador, dependencias.BorradorRRHH)
		}
		detalleRRHH, err = httpinterno.NuevoManejadorConsultaDetalleRRHH(
			dependencias.ConsultorDetalleRRHH, renderizadoresBorrador...,
		)
	}
	if err != nil {
		return nil, ErrRutasContratacionTemporalInvalidas
	}
	asignacion, err := httpinterno.NuevoManejadorAsignacion(
		dependencias.AutoridadAsignacion,
		dependencias.EjecutorAsignacion,
	)
	if err != nil {
		return nil, ErrRutasContratacionTemporalInvalidas
	}
	informeJuridico, err := httpinterno.NuevoManejadorInformeJuridico(
		dependencias.AutoridadInformeJuridico,
		dependencias.EjecutorInformeJuridico,
	)
	if err != nil {
		return nil, ErrRutasContratacionTemporalInvalidas
	}
	fiscalizacion, err := httpinterno.NuevoManejadorFiscalizacion(
		dependencias.AutoridadFiscalizacion,
		dependencias.EjecutorFiscalizacion,
	)
	if err != nil {
		return nil, ErrRutasContratacionTemporalInvalidas
	}
	var subsanacion http.Handler
	if dependencias.AutoridadSubsanacionReparos != nil || dependencias.EjecutorSubsanacionReparos != nil {
		subsanacion, err = httpinterno.NuevoManejadorSubsanacionReparos(
			dependencias.AutoridadSubsanacionReparos,
			dependencias.EjecutorSubsanacionReparos,
		)
		if err != nil {
			return nil, ErrRutasContratacionTemporalInvalidas
		}
	}
	rutas := []httpapi.RutaExacta{
		{
			Ruta:      httpinterno.RutaAltaSolicitudes,
			Manejador: alta,
		},
		{
			Ruta:      httpinterno.RutaPropuestaCobertura,
			Manejador: cobertura,
		},
		{
			Ruta:      httpinterno.RutaDecisionCobertura,
			Manejador: cobertura,
		},
		{
			Ruta:      httpinterno.RutaRectificacionCobertura,
			Manejador: cobertura,
		},
		{
			Ruta:      httpinterno.RutaResultadoCobertura,
			Manejador: resultado,
		},
		{
			Ruta:      httpinterno.RutaRegistroAnalisisRRHH,
			Manejador: analisis,
		},
		{
			Ruta:      httpinterno.RutaRectificacionAnalisisRRHH,
			Manejador: analisis,
		},
		{
			Ruta:      httpinterno.RutaSeleccionLlamamiento,
			Manejador: seleccion,
		},
		{
			Ruta:      httpinterno.RutaPropuestaFormalizacion,
			Manejador: propuesta,
		},
		{
			Ruta:      httpinterno.RutaCerrarAdministrativamente,
			Manejador: cierreAdministrativo,
		},
		{
			Ruta:      httpinterno.RutaReabrirExcepcionalmente,
			Manejador: cierreAdministrativo,
		},
		{
			Ruta:      httpinterno.RutaPreparacionesInformeJuridico,
			Manejador: informeJuridico,
		},
		{
			Ruta:      httpinterno.RutaResultadosFiscalizacion,
			Manejador: fiscalizacion,
		},
		{
			Ruta:      httpinterno.RutaConsultaCuadroRRHH,
			Manejador: cuadroRRHH,
		},
		{
			Ruta:      httpinterno.RutaConsultaDetalleRRHH,
			Manejador: detalleRRHH,
		},
		{
			Ruta:      httpinterno.RutaAsignaciones,
			Manejador: asignacion,
		},
		{
			Ruta:      httpinterno.RutaReasignaciones,
			Manejador: asignacion,
		},
	}
	if subsanacion != nil {
		rutas = append(rutas, httpapi.RutaExacta{
			Ruta: httpinterno.RutaSubsanacionReparos, Manejador: subsanacion,
		})
	}
	if dependencias.IncorporacionV2 != nil {
		ruta, err := NuevaRutaIncorporacionV2(dependencias.IncorporacionV2)
		if err != nil {
			return nil, ErrRutasContratacionTemporalInvalidas
		}
		rutas = append(rutas, ruta)
	}
	return rutas, nil
}

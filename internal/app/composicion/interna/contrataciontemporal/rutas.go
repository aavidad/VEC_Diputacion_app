package contrataciontemporal

import (
	"errors"

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
	detalleRRHH, err := httpinterno.NuevoManejadorConsultaDetalleRRHH(
		dependencias.ConsultorDetalleRRHH,
	)
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
	return []httpapi.RutaExacta{
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
	}, nil
}

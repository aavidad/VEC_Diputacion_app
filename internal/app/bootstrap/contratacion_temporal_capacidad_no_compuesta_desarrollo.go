package bootstrap

import (
	"context"
	"encoding/json"
	"io"
	"reflect"
	"sync"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
)

const claveI18nCapacidadNoCompuestaContratacionTemporal = "api.vec.ruta_exacta.error.servicio_no_disponible"

var rutasCapacidadNoCompuestaContratacionTemporal = map[string]struct{}{
	httpinterno.RutaRectificacionAnalisisRRHH: {},
	httpinterno.RutaSeleccionLlamamiento:      {},
	httpinterno.RutaPropuestaFormalizacion:    {},
	httpinterno.RutaCerrarAdministrativamente: {},
	httpinterno.RutaReabrirExcepcionalmente:   {},
	httpinterno.RutaConsultaCuadroRRHH:        {},
	httpinterno.RutaConsultaDetalleRRHH:       {},
	httpinterno.RutaReasignaciones:            {},
}

// capacidadNoCompuestaContratacionTemporalDesarrollo es la única dependencia
// compartida por las rutas registradas que todavía no tienen un servicio real.
// La autoridad exterior la consulta antes de que HTTP lea el cuerpo.
type capacidadNoCompuestaContratacionTemporalDesarrollo struct {
	mu       sync.Mutex
	registro io.Writer
}

var (
	_ httpinterno.AutoridadContextoCanalAnalisisRRHH      = (*capacidadNoCompuestaContratacionTemporalDesarrollo)(nil)
	_ httpinterno.EjecutorAnalisisRRHH                    = (*capacidadNoCompuestaContratacionTemporalDesarrollo)(nil)
	_ httpinterno.EjecutorSeleccionLlamamiento            = (*capacidadNoCompuestaContratacionTemporalDesarrollo)(nil)
	_ httpinterno.AutoridadServidorPropuestaFormalizacion = (*capacidadNoCompuestaContratacionTemporalDesarrollo)(nil)
	_ httpinterno.EjecutorPropuestaFormalizacion          = (*capacidadNoCompuestaContratacionTemporalDesarrollo)(nil)
	_ httpinterno.AutoridadServidorCierreAdministrativo   = (*capacidadNoCompuestaContratacionTemporalDesarrollo)(nil)
	_ httpinterno.EjecutorCierreAdministrativo            = (*capacidadNoCompuestaContratacionTemporalDesarrollo)(nil)
	_ httpinterno.ConsultorCuadroRRHH                     = (*consultorCuadroNoCompuestoContratacionTemporalDesarrollo)(nil)
	_ httpinterno.ConsultorDetalleRRHH                    = (*consultorDetalleNoCompuestoContratacionTemporalDesarrollo)(nil)
)

func nuevaCapacidadNoCompuestaContratacionTemporalDesarrollo(
	registro io.Writer,
) (*capacidadNoCompuestaContratacionTemporalDesarrollo, error) {
	if dependenciaEsNulaContratacionTemporalDesarrollo(registro) {
		return nil, ErrComposicionDesarrolloIncompleta
	}
	return &capacidadNoCompuestaContratacionTemporalDesarrollo{registro: registro}, nil
}

func (c *capacidadNoCompuestaContratacionTemporalDesarrollo) esRuta(ruta string) bool {
	if c == nil {
		return false
	}
	_, existe := rutasCapacidadNoCompuestaContratacionTemporal[ruta]
	return existe
}

func (c *capacidadNoCompuestaContratacionTemporalDesarrollo) denegarRuta(
	ctx context.Context,
	ruta string,
) error {
	if c == nil || dependenciaEsNulaContratacionTemporalDesarrollo(c.registro) ||
		ctx == nil || ctx.Err() != nil || !c.esRuta(ruta) {
		return vechttp.ErrAutoridadRutaExactaNoDisponible
	}
	entrada := struct {
		Modulo    string `json:"modulo"`
		Ruta      string `json:"ruta"`
		Resultado string `json:"resultado"`
		Motivo    string `json:"motivo"`
		ClaveI18n string `json:"clave_i18n"`
	}{
		Modulo:    "contratacion_temporal",
		Ruta:      ruta,
		Resultado: "denegado",
		Motivo:    "servicio_no_disponible",
		ClaveI18n: claveI18nCapacidadNoCompuestaContratacionTemporal,
	}
	c.mu.Lock()
	_ = json.NewEncoder(c.registro).Encode(entrada)
	c.mu.Unlock()
	return vechttp.ErrAutoridadRutaExactaNoDisponible
}

func dependenciaEsNulaContratacionTemporalDesarrollo(dependencia any) bool {
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

func (*capacidadNoCompuestaContratacionTemporalDesarrollo) ResolverContextoCanalAnalisisRRHH(
	context.Context,
) (httpinterno.ContextoCanalAnalisisRRHH, error) {
	return httpinterno.ContextoCanalAnalisisRRHH{}, httpinterno.ErrContextoCanalNoDisponible
}

func (*capacidadNoCompuestaContratacionTemporalDesarrollo) Registrar(
	context.Context,
	application.SolicitudRegistrarAnalisis,
) (ports.ReciboOperacionAnalisis, error) {
	return ports.ReciboOperacionAnalisis{}, application.ErrDependenciaOperacionAnalisisNoDisponible
}

func (*capacidadNoCompuestaContratacionTemporalDesarrollo) Rectificar(
	context.Context,
	application.SolicitudRectificarAnalisis,
) (ports.ReciboOperacionAnalisis, error) {
	return ports.ReciboOperacionAnalisis{}, application.ErrDependenciaOperacionAnalisisNoDisponible
}

func (*capacidadNoCompuestaContratacionTemporalDesarrollo) SeleccionarYLlamarParaAdaptador(
	context.Context,
	application.SolicitudSeleccionLlamamiento,
) (application.DatosReciboSeleccionLlamamientoParaAdaptador, error) {
	return application.DatosReciboSeleccionLlamamientoParaAdaptador{},
		application.ErrServicioSeleccionLlamamientoInvalido
}

func (*capacidadNoCompuestaContratacionTemporalDesarrollo) ResolverContextoPropuestaFormalizacion(
	context.Context,
) (httpinterno.ContextoServidorPropuestaFormalizacion, error) {
	return httpinterno.ContextoServidorPropuestaFormalizacion{},
		application.ErrPropuestaFormalizacionNoDisponible
}

func (*capacidadNoCompuestaContratacionTemporalDesarrollo) PrepararYConfirmar(
	context.Context,
	ports.SolicitudPropuestaFormalizacion,
) (ports.ResultadoPropuestaFormalizacion, error) {
	return ports.ResultadoPropuestaFormalizacion{},
		application.ErrPropuestaFormalizacionNoDisponible
}

func (*capacidadNoCompuestaContratacionTemporalDesarrollo) ResolverOrganizacionCierreAdministrativo(
	context.Context,
) (string, error) {
	return "", application.ErrCierreAdministrativoNoDisponible
}

func (*capacidadNoCompuestaContratacionTemporalDesarrollo) Cerrar(
	context.Context,
	application.SolicitudCerrarAdministrativamente,
) (ports.ResultadoCierreAdministrativo, error) {
	return ports.ResultadoCierreAdministrativo{}, application.ErrCierreAdministrativoNoDisponible
}

func (*capacidadNoCompuestaContratacionTemporalDesarrollo) ReabrirExcepcionalmente(
	context.Context,
	application.SolicitudReabrirExcepcionalmente,
) (ports.ResultadoCierreAdministrativo, error) {
	return ports.ResultadoCierreAdministrativo{}, application.ErrCierreAdministrativoNoDisponible
}

type consultorCuadroNoCompuestoContratacionTemporalDesarrollo struct {
	*capacidadNoCompuestaContratacionTemporalDesarrollo
}

func (*consultorCuadroNoCompuestoContratacionTemporalDesarrollo) Consultar(
	context.Context,
	ports.SolicitudCuadroRRHH,
) (ports.PaginaCuadroRRHH, error) {
	return ports.PaginaCuadroRRHH{}, application.ErrConsultaRRHHNoDisponible
}

type consultorDetalleNoCompuestoContratacionTemporalDesarrollo struct {
	*capacidadNoCompuestaContratacionTemporalDesarrollo
}

func (*consultorDetalleNoCompuestoContratacionTemporalDesarrollo) Consultar(
	context.Context,
	ports.SolicitudDetalleRRHH,
) (ports.DetalleExpedienteRRHH, error) {
	return ports.DetalleExpedienteRRHH{}, application.ErrConsultaRRHHNoDisponible
}

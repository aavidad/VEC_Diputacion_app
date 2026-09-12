package bootstrap

import (
	"context"
	"reflect"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"

	"github.com/jackc/pgx/v5/pgxpool"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// ConfiguracionAnotacionAdministrativaDesarrollo recibe fronteras ya
// constituidas por la raíz. No abre DSN, publica políticas ni crea identidad.
type ConfiguracionAnotacionAdministrativaDesarrollo struct {
	EjecutorCT    *pgxpool.Pool
	Contextos     ct.ResolutorContextoAutorizacionAltaV3
	Solicitudes   ct.ResolutorSolicitudPersonalAnotacionAdministrativa
	Ambitos       ct.SelladorAmbitoAnotacionAdministrativa
	Huellas       ct.DerivadorHuellaAnotacionAdministrativa
	Politicas     ct.ResolutorPoliticaAnotacionAdministrativa
	Correlaciones vp.GeneradorReferenciasAutorizacionV2
	Autorizador   vp.AutorizadorSolicitudLigadaV3
	Reloj         ct.Reloj
	Proveedor     postgresct.ProveedorMaterialAnotacionAdministrativa
	Detalle       *appct.ServicioConsultaDetalleRRHH
	Lector        lectorMaterialAnotacionAdministrativaDesarrollo
}

// NuevoServicioAnotacionAdministrativaDesarrollo compone solamente cuando la
// autoridad V3, la política publicada y el consumidor durable ya existen.
func NuevoServicioAnotacionAdministrativaDesarrollo(c ConfiguracionAnotacionAdministrativaDesarrollo) (*appct.ServicioAnotacionesAdministrativas, error) {
	if c.EjecutorCT == nil || dependenciaBootstrapNula(c.Contextos) ||
		dependenciaBootstrapNula(c.Solicitudes) || dependenciaBootstrapNula(c.Ambitos) ||
		dependenciaBootstrapNula(c.Huellas) || dependenciaBootstrapNula(c.Politicas) ||
		dependenciaBootstrapNula(c.Correlaciones) || dependenciaBootstrapNula(c.Autorizador) ||
		dependenciaBootstrapNula(c.Reloj) || dependenciaBootstrapNula(c.Proveedor) || c.Detalle == nil || dependenciaBootstrapNula(c.Lector) {
		return nil, appct.ErrServicioAnotacionesAdministrativasInvalido
	}
	preparador, err := postgresct.NuevoPreparadorAnotacionAdministrativaPostgreSQL(c.EjecutorCT)
	if err != nil {
		return nil, appct.ErrServicioAnotacionesAdministrativasInvalido
	}
	transaccion, err := postgresct.NuevaTransaccionAnotacionesAdministrativasPostgreSQL(c.EjecutorCT, c.Proveedor)
	if err != nil {
		return nil, appct.ErrServicioAnotacionesAdministrativasInvalido
	}
	servicio, err := appct.NuevoServicioAnotacionesAdministrativas(c.Contextos, c.Solicitudes, c.Ambitos, c.Huellas, preparador, c.Politicas, c.Correlaciones, c.Autorizador, c.Reloj, transaccion)
	if err != nil {
		return nil, err
	}
	return servicio.ConRecuperacionAutorizada(&fuenteLecturaAnotacionAdministrativaDesarrollo{detalle: c.Detalle, lector: c.Lector})
}

type lectorMaterialAnotacionAdministrativaDesarrollo interface {
	RecuperarMaterialAnotacionAdministrativa(context.Context, ct.SolicitudRecuperarMaterialAnotacionAdministrativa) (ct.MaterialAnotacionAdministrativa, error)
}

type fuenteLecturaAnotacionAdministrativaDesarrollo struct {
	detalle *appct.ServicioConsultaDetalleRRHH
	lector  lectorMaterialAnotacionAdministrativaDesarrollo
}

func (f *fuenteLecturaAnotacionAdministrativaDesarrollo) RecuperarMaterialAnotacionAdministrativaAutorizada(ctx context.Context, s ct.SolicitudRecuperarMaterialAnotacionAdministrativa) (ct.MaterialAnotacionAdministrativa, error) {
	if ctx == nil || f == nil || f.detalle == nil || dependenciaBootstrapNula(f.lector) || s.Contexto.Vinculo.ValidarPara(s.Contexto.Resultado) != nil {
		return ct.MaterialAnotacionAdministrativa{}, ct.ErrPersistenciaAnotacionAdministrativaNoDisponible
	}
	consulta, err := ct.NuevaSolicitudDetalleRRHH(s.ExpedienteRef, 0)
	if err != nil {
		return ct.MaterialAnotacionAdministrativa{}, ct.ErrPreparacionAnotacionAdministrativaInvalida
	}
	detalle, err := f.detalle.Consultar(ctx, consulta)
	if err != nil || detalle.Resumen.ExpedienteRef != s.ExpedienteRef || detalle.Resumen.OrganizacionRef != s.OrganizacionRef {
		return ct.MaterialAnotacionAdministrativa{}, ct.ErrAutorizacionDenegada
	}
	return f.lector.RecuperarMaterialAnotacionAdministrativa(ctx, s)
}

// ejecutorAnotacionAdministrativaDesarrollo adapta el canal confiable al caso
// de uso. La recuperación nunca incorpora observaciones o versión desde HTTP.
type ejecutorAnotacionAdministrativaDesarrollo struct {
	servicio *appct.ServicioAnotacionesAdministrativas
}

func (e *ejecutorAnotacionAdministrativaDesarrollo) RegistrarAnotacionAdministrativa(ctx context.Context, s appct.SolicitudRegistrarAnotacionAdministrativa) (ct.ReciboAnotacionAdministrativa, error) {
	if e == nil || e.servicio == nil {
		return ct.ReciboAnotacionAdministrativa{}, ct.ErrPersistenciaAnotacionAdministrativaNoDisponible
	}
	return e.servicio.RegistrarAnotacionAdministrativa(ctx, s)
}
func (e *ejecutorAnotacionAdministrativaDesarrollo) RecuperarAnotacionAdministrativa(ctx context.Context, exp, clave string, canal httpinterno.ContextoCanalAnotacionAdministrativa) (ct.ReciboAnotacionAdministrativa, error) {
	if e == nil || e.servicio == nil {
		return ct.ReciboAnotacionAdministrativa{}, ct.ErrPersistenciaAnotacionAdministrativaNoDisponible
	}
	return e.servicio.RecuperarAnotacionAdministrativa(ctx, appct.SolicitudRegistrarAnotacionAdministrativa{AutenticacionRef: canal.AutenticacionRef, SesionRef: canal.SesionRef, PerfilRef: canal.PerfilRef, OrganizacionRef: canal.OrganizacionRef, ExpedienteRef: exp, ClaveIdempotencia: clave})
}

func NuevoEjecutorAnotacionAdministrativaDesarrollo(c ConfiguracionAnotacionAdministrativaDesarrollo) (httpinterno.EjecutorAnotacionAdministrativa, error) {
	s, err := NuevoServicioAnotacionAdministrativaDesarrollo(c)
	if err != nil {
		return nil, err
	}
	return &ejecutorAnotacionAdministrativaDesarrollo{servicio: s}, nil
}

func dependenciaBootstrapNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	default:
		return false
	}
}

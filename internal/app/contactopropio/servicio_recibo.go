package contactopropio

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/usuarios"
	"vec-diputacion-granada/internal/shared/i18n"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// DependenciasConsultaRecibo es opt-in. El emisor y el perfil de consumidor
// corresponden a recibos propios, no al lector de correo para llamamientos.
type DependenciasConsultaRecibo struct {
	PoolConsulta *pgxpool.Pool
	Emisor       *confianzaatestacion.EmisorMaterialAutorizacionAtestadaV3
	Motivo       domain.ReferenciaEntradaCatalogo
}

type ServicioConRecibos struct {
	*Servicio
	consulta      ports.ConsultorReciboContactoUsuario
	emisorRecibos *confianzaatestacion.EmisorMaterialAutorizacionAtestadaV3
	motivoRecibos domain.ReferenciaEntradaCatalogo
}

// NuevoServicioConRecibos conserva el constructor anterior sin exigir nuevas
// dependencias al guardado. No monta una ruta de consulta si falta su autoridad.
func NuevoServicioConRecibos(s *Servicio, d DependenciasConsultaRecibo) (*ServicioConRecibos, error) {
	if s == nil || s.dependencias.Identidad == nil || dependenciaContactoPropioNula(s.registro) || d.PoolConsulta == nil || d.Emisor == nil || d.Motivo.Validar() != nil {
		return nil, ErrContactoPropioNoDisponible
	}
	consultor, err := postgres.NuevoConsultorReciboContactoUsuarioPostgreSQL(d.PoolConsulta)
	if err != nil {
		return nil, ErrContactoPropioNoDisponible
	}
	return &ServicioConRecibos{Servicio: s, consulta: consultor, emisorRecibos: d.Emisor, motivoRecibos: d.Motivo}, nil
}

func (s *ServicioConRecibos) ConsultarRecibo(ctx context.Context, version uint64) (ports.ResultadoConsultaReciboContactoUsuario, error) {
	vacio := ports.ResultadoConsultaReciboContactoUsuario{}
	if version == 0 || version > 1<<53-1 {
		return vacio, ErrContactoPropioInvalido
	}
	if s == nil || s.Servicio == nil || ctx == nil || ctx.Err() != nil || s.dependencias.Identidad == nil || dependenciaContactoPropioNula(s.consulta) || s.emisorRecibos == nil {
		return vacio, ErrContactoPropioNoDisponible
	}
	d := s.dependencias
	cuenta, audit, err := d.Identidad.ExtraerCapsulaIdentidadPeticion(ctx)
	if err != nil || audit.CuentaPrivilegiada() || audit.Superficie() != httpseguridad.SuperficieExternaPersonal || cuenta.Garantia != domain.AuthAssuranceHigh || (cuenta.Metodo != domain.AuthMethodCertificate && cuenta.Metodo != domain.AuthMethodDNIe) {
		return vacio, ErrContactoPropioNoDisponible
	}
	vinculo, resultado, err := domain.CrearVinculoAutenticacionActorV2ConResultado(ctx, d.Revalidador, domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: audit.AutenticacionRef(), SesionRef: audit.SesionRef()}, d.Resolutor, domain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: d.PerfilPropioRef}, d.Reloj)
	if err != nil {
		return vacio, ErrContactoPropioNoDisponible
	}
	correlacion, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, d.GeneradorCorrelacion)
	if err != nil {
		return vacio, ErrContactoPropioNoDisponible
	}
	correlacionRef, err := correlacion.ValorCanonico()
	if err != nil {
		return vacio, ErrContactoPropioNoDisponible
	}
	recurso := domain.RecursoAutorizable{Referencia: resultado.Contexto.PersonaRef, ModuloID: usuarios.ModuleID, Tipo: "contacto_usuario", Ambitos: clonarAmbitos(d.AmbitosRecurso)}
	preparador := preparadorAuditoria{fuente: d.FuenteAutorizacion, seudonimizador: d.Seudonimizador, reloj: d.Reloj, correlacion: correlacionRef, recurso: recurso}
	servicio, err := application.NuevoServicioConsultaReciboContactoUsuario(preparador, s.emisorRecibos, s.consulta)
	if err != nil {
		return vacio, ErrContactoPropioNoDisponible
	}
	r, err := servicio.Consultar(ctx, ports.SolicitudConsultaReciboContactoUsuario{ContextoActor: resultado.Contexto, Version: version, Recurso: recurso, ResultadoContexto: resultado, SolicitudBase: domain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: vinculo, ReferenciaMotivo: s.motivoRecibos, Accion: application.AccionConsultarContactoUsuario, Recurso: recurso, Finalidad: application.FinalidadConsultaReciboContactoUsuario, Correlacion: correlacion}})
	if err != nil {
		return vacio, ErrContactoPropioNoDisponible
	}
	return r, nil
}

func NuevasRutasConRecibos(s *ServicioConRecibos, catalogo *i18n.Catalog) ([]httpapi.RutaExacta, error) {
	if s == nil || dependenciaContactoPropioNula(s.consulta) || s.emisorRecibos == nil {
		return nil, ErrManejadorContactoPropioInvalido
	}
	rutas, err := NuevasRutas(s.Servicio, catalogo)
	if err != nil {
		return nil, err
	}
	manejador, err := NuevoManejadorRecibo(s, catalogo)
	if err != nil {
		return nil, err
	}
	return append(rutas, httpapi.RutaExacta{Ruta: RutaReciboContactoPropio, Manejador: manejador}), nil
}

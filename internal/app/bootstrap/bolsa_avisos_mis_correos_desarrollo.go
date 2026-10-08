package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	usuariospg "vec-diputacion-granada/internal/modules/usuarios/adapters/postgres"
	usuariosseguridad "vec-diputacion-granada/internal/modules/usuarios/adapters/seguridad"
	usuariosapp "vec-diputacion-granada/internal/modules/usuarios/application"
	"vec-diputacion-granada/internal/modules/usuarios/canonico"
	usuariosports "vec-diputacion-granada/internal/modules/usuarios/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// VEC_BOLSA_AVISOS_MIS_CORREOS_ENABLED (B59) hace que el aviso de un
// llamamiento vaya al correo activo de «Mis correos» cuando la persona
// candidata lo tenga, y al del alta en otro caso, dejando constancia de la
// fuente. Exige «Mis correos», B-BACK y las SQL AD3-109, ContextoActor 000010,
// Bolsa 000059 y Usuarios 000008. Literal true/false; otro valor no arranca.
const envBolsaAvisosMisCorreosDesarrollo = "VEC_BOLSA_AVISOS_MIS_CORREOS_ENABLED"

var errComposicionAvisosMisCorreos = errors.New("bootstrap: avisos con Mis correos no disponibles")

// descriptorMaterialCorreoAvisosDesarrollo es la audiencia de AD3-109, con su
// dominio y prefijo de clave propios bajo el gobierno V3 común.
func descriptorMaterialCorreoAvisosDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia:        usuariosports.AudienciaCorreoAvisosLlamamientoInterna,
		Dominio:          "vec.usuarios.correos.avisos-llamamiento.interna.desarrollo.capacidad-v3",
		Prefijo:          "clave:capacidad:usuarios-correos-avisos-llamamiento-interna:",
		ProveedorNominal: "proveedor-material-usuarios-correos-avisos-llamamiento-interna",
	}
}

// abrirPoolCorreoAvisosDesarrollo abre un pool pequeño con el LOGIN ejecutor
// interno de Usuarios (el mismo DSN que «Mis correos» de RRHH).
func abrirPoolCorreoAvisosDesarrollo(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error) {
	c, err := leerConfiguracionUsuariosPreferenciasDesarrollo(cfg, core.SuperficieAutenticacionInternaCorporativaV1)
	if err != nil {
		return nil, errComposicionAvisosMisCorreos
	}
	pool, _, err := abrirPoolUsuariosPreferencias(ctx, c.DSNUsuarios, rolEjecutorPreferencias(string(core.SuperficieAutenticacionInternaCorporativaV1)))
	if err != nil {
		return nil, errComposicionAvisosMisCorreos
	}
	return pool, nil
}

// preflightSQLCorreoAvisosDesarrollo comprueba, antes de publicar la
// audiencia, que Usuarios 000008 está instalada y es ejecutable.
func preflightSQLCorreoAvisosDesarrollo(cfg config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), plazoarranque.Ampliar(20*time.Second))
	defer cancel()
	pool, err := abrirPoolCorreoAvisosDesarrollo(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()
	if _, err := usuariospg.NuevoRegistroCorreoAvisosPostgreSQL(ctx, pool); err != nil {
		return errComposicionAvisosMisCorreos
	}
	return nil
}

// componerAvisosMisCorreosBolsaDesarrollo activa B59 sobre la emisión ya
// compuesta. Sin el selector no hace nada y devuelve el cierre recibido; con
// él, devuelve ese cierre precedido del cierre de su propio pool. Nunca
// devuelve un cierre nulo, tampoco con error.
func componerAvisosMisCorreosBolsaDesarrollo(ctx context.Context, cfg config.Config, emision *aplicacionbolsa.ServicioEmisionLlamamiento,
	repositorio interface {
		puertosbolsa.LectorCandidatoParticipacion
		ActivarFuentesCorreo(context.Context) error
	},
	pdp vecports.AutorizadorSolicitudLigadaV3, material *proveedorMaterialAltaContratacionTemporalDesarrollo, kms *emisorKMSDesarrollo,
	cerrarPrevio func(),
) (func(), error) {
	if cerrarPrevio == nil {
		cerrarPrevio = func() {}
	}
	activo, err := selectorCapacidadRRHHDesarrollo(cfg, envBolsaAvisosMisCorreosDesarrollo)
	if err != nil || !activo {
		return cerrarPrevio, err
	}
	if ctx == nil || emision == nil || repositorio == nil || pdp == nil || material == nil || kms == nil {
		return cerrarPrevio, errComposicionAvisosMisCorreos
	}
	if err := repositorio.ActivarFuentesCorreo(ctx); err != nil {
		return cerrarPrevio, errComposicionAvisosMisCorreos
	}
	emisor, err := nuevoEmisorMaterialRenovableCTDesarrollo(pdp, material)
	if err != nil {
		return cerrarPrevio, errComposicionAvisosMisCorreos
	}
	fuente, err := nuevaFuenteClavesCorreosDesarrollo(kms)
	if err != nil {
		return cerrarPrevio, errComposicionAvisosMisCorreos
	}
	cripto, err := usuariosseguridad.NuevoAdaptadorCorreos(fuente, time.Now)
	if err != nil {
		return cerrarPrevio, errComposicionAvisosMisCorreos
	}
	pool, err := abrirPoolCorreoAvisosDesarrollo(ctx, cfg)
	if err != nil {
		return cerrarPrevio, err
	}
	registro, err := usuariospg.NuevoRegistroCorreoAvisosPostgreSQL(ctx, pool)
	if err != nil {
		pool.Close()
		return cerrarPrevio, errComposicionAvisosMisCorreos
	}
	servicio, err := usuariosapp.NuevoServicioCorreoAvisos(registro, cripto)
	if err != nil {
		pool.Close()
		return cerrarPrevio, errComposicionAvisosMisCorreos
	}
	if err := emision.EstablecerCorreoAvisosPersona(repositorio, &puenteCorreoAvisosBolsaDesarrollo{servicio: servicio, emisor: emisor}); err != nil {
		pool.Close()
		return cerrarPrevio, errComposicionAvisosMisCorreos
	}
	return func() { pool.Close(); cerrarPrevio() }, nil
}

// puenteCorreoAvisosBolsaDesarrollo es el adaptador entre el puerto de
// Bolsa y el caso de uso de Usuarios. Bolsa no importa Usuarios.
type puenteCorreoAvisosBolsaDesarrollo struct {
	servicio *usuariosapp.ServicioCorreoAvisos
	emisor   emisorPreferenciasUsuarios
}

var _ puertosbolsa.FuenteCorreoAvisoPersona = (*puenteCorreoAvisosBolsaDesarrollo)(nil)

func (p *puenteCorreoAvisosBolsaDesarrollo) ConCorreoAvisoPersona(ctx context.Context, s puertosbolsa.SolicitudCorreoAvisoPersona, usar func(string)) (bool, string, error) {
	if p == nil || p.servicio == nil || p.emisor == nil {
		return false, "", usuariosports.ErrCorreosNoDisponible
	}
	orden := usuariosports.OrdenCorreoAvisos{Proveedor: &proveedorV3CorreoAvisosDesarrollo{emisor: p.emisor, solicitud: s}}
	r, err := p.servicio.ConCorreoActivoAvisos(ctx, orden, usuariosports.SolicitudCorreoAvisos{
		BolsaRef: s.BolsaRef, UnidadRef: s.UnidadRef, AmbitoRef: s.AmbitoRef, LlamamientoRef: s.LlamamientoRef, CandidatoRef: s.CandidatoRef,
	}, usar)
	return r.Encontrado, r.CorreoRef, err
}

// proveedorV3CorreoAvisosDesarrollo emite la V3 de la lectura con la
// identidad de quien emite el llamamiento y el PDP de Bolsa: la misma
// concesión que la emisión, con la audiencia propia de AD3-109.
type proveedorV3CorreoAvisosDesarrollo struct {
	emisor    emisorPreferenciasUsuarios
	solicitud puertosbolsa.SolicitudCorreoAvisoPersona
}

func (p *proveedorV3CorreoAvisosDesarrollo) ProveerMaterialCorreoAvisos(ctx context.Context, m usuariosports.MaterialCorreoAvisos, material []byte) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacia := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if p == nil || p.emisor == nil || ctx == nil || ctx.Err() != nil {
		return vacia, usuariosports.ErrCorreosNoDisponible
	}
	s := p.solicitud
	if s.ResultadoContexto.Validar() != nil || s.Vinculo.ValidarPara(s.ResultadoContexto) != nil || !core.ReferenciaMotivoAutorizacionV2Valida(s.MotivoAutorizacion) {
		return vacia, usuariosports.ErrCorreosProhibido
	}
	datos, err := s.Vinculo.Datos()
	if err != nil || datos.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 {
		return vacia, usuariosports.ErrCorreosProhibido
	}
	recurso, preimagen, err := canonico.RecursoCorreoAvisos(m)
	if err != nil || !bytes.Equal(preimagen, material) || m.BolsaRef != s.BolsaRef || m.CandidatoRef != s.CandidatoRef || m.LlamamientoRef != s.LlamamientoRef {
		return vacia, usuariosports.ErrCorreosProhibido
	}
	correlacion, err := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacia, usuariosports.ErrCorreosNoDisponible
	}
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: s.Vinculo, ReferenciaMotivo: s.MotivoAutorizacion, Accion: usuariosports.AccionV3CorreoAvisosLlamamiento,
		Recurso: recurso, Finalidad: usuariosports.FinalidadCorreoAvisosLlamamiento, Correlacion: correlacion,
	})
	if err != nil {
		return vacia, usuariosports.ErrCorreosProhibido
	}
	decision, confirmacion, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, s.ResultadoContexto)
	if errors.Is(err, core.ErrAutorizacionDenegada) {
		return vacia, usuariosports.ErrCorreosProhibido
	}
	if err != nil || exportador == nil || decision.ValidarPara(solicitud) != nil {
		return vacia, usuariosports.ErrCorreosNoDisponible
	}
	exportado, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, s.ResultadoContexto, s.MotivoAutorizacion, exportado, usuariosports.AudienciaCorreoAvisosLlamamientoInterna) {
		return vacia, usuariosports.ErrCorreosProhibido
	}
	return exportado, nil
}

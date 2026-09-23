package contactopropio

import (
	"context"

	"vec-diputacion-granada/internal/modules/usuarios"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// VersionPropia consulta el selector durable con una concesión distinta de
// leer el correo y sin aceptar sujeto o versión de HTTP.
func (s *ServicioConRecibos) VersionPropia(ctx context.Context) (ports.ResultadoVersionContactoUsuario, error) {
	vacio := ports.ResultadoVersionContactoUsuario{}
	if s == nil || s.Servicio == nil || ctx == nil || ctx.Err() != nil || dependenciaContactoPropioNula(s.consultaVersion) || dependenciaContactoPropioNula(s.emisorVersion) || s.motivoVersion.Validar() != nil {
		return vacio, ErrContactoPropioNoDisponible
	}
	d := s.dependencias
	var vinculo domain.VinculoAutenticacionActorV2
	var resultado domain.ResultadoContextoActorRegistradoV2
	var err error
	if !dependenciaContactoPropioNula(d.Sesion) {
		vinculo, resultado, err = d.Sesion.ResolverContactoPropio(ctx)
	} else {
		cuenta, audit, e := d.Identidad.ExtraerCapsulaIdentidadPeticion(ctx)
		if e != nil || audit.CuentaPrivilegiada() || audit.Superficie() != httpseguridad.SuperficieExternaPersonal || cuenta.Garantia != domain.AuthAssuranceHigh || (cuenta.Metodo != domain.AuthMethodCertificate && cuenta.Metodo != domain.AuthMethodDNIe) {
			return vacio, ErrContactoPropioNoDisponible
		}
		vinculo, resultado, err = domain.CrearVinculoAutenticacionActorV2ConResultado(ctx, d.Revalidador, domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: audit.AutenticacionRef(), SesionRef: audit.SesionRef()}, d.Resolutor, domain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: d.PerfilPropioRef}, d.Reloj)
	}
	if err != nil || vinculo.ValidarPara(resultado) != nil || resultado.Contexto.PerfilActivoRef != d.PerfilPropioRef {
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
	sujeto := resultado.Contexto.PersonaRef
	recurso := domain.RecursoAutorizable{Referencia: sujeto, ModuloID: usuarios.ModuleID, Tipo: "contacto_usuario", Ambitos: clonarAmbitos(d.AmbitosRecurso)}
	preparador := preparadorAuditoria{fuente: d.FuenteAutorizacion, seudonimizador: d.Seudonimizador, reloj: d.Reloj, correlacion: correlacionRef, recurso: recurso}
	servicio, err := application.NuevoServicioVersionContactoUsuario(preparador, s.emisorVersion, s.consultaVersion)
	if err != nil {
		return vacio, ErrContactoPropioNoDisponible
	}
	r, err := servicio.Consultar(ctx, ports.SolicitudVersionContactoUsuario{Clase: ports.VersionContactoPropia, SujetoRef: sujeto, ContextoActor: resultado.Contexto, Recurso: recurso, ResultadoContexto: resultado, SolicitudBase: domain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: vinculo, ReferenciaMotivo: s.motivoVersion, Accion: application.AccionVersionContactoPropia, Recurso: recurso, Finalidad: application.FinalidadVersionContactoPropia, Correlacion: correlacion}})
	if err != nil {
		return vacio, ErrContactoPropioNoDisponible
	}
	return r, nil
}

package bootstrap

import (
	"context"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	bolsaapp "vec-diputacion-granada/internal/modules/bolsa/application"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

type perfilPreparacionBasesV3 struct {
	soporte                                                          *soporteAltaContratacionTemporalDesarrollo
	plantilla                                                        core.InstantaneaAutorizacion
	accion, ruta, audiencia, actoAsignacion, actoControl, actoSesion string
	motivo                                                           core.ReferenciaEntradaCatalogo
	ambito                                                           bolsa.AmbitoOrganizativoConvocatoria
	provision                                                        provisionPreparacionBasesV3
}

func (p *perfilPreparacionBasesV3) perfilRef() string {
	if p == nil {
		return ""
	}
	return p.plantilla.AsignacionPerfil.PerfilActivoRef
}

func nuevoPerfilPreparacionBasesV3(base *soporteAltaContratacionTemporalDesarrollo,
	c configuracionPreparacionBasesV3, guardar bool, ahora time.Time) (*perfilPreparacionBasesV3, error) {
	if !c.valida() {
		return nil, errMontajePreparacionBasesV3
	}
	i := 1
	nombre, motivo, provision := "consultar", c.MotivoConsultar, c.Consultar
	if guardar {
		i, nombre, motivo, provision = 0, "guardar", c.MotivoGuardar, c.Guardar
	}
	etiqueta := "seleccion-preparacion-bases-" + nombre + "-v3"
	d := discriminadorContextoSinteticoDesarrollo{perfil: "perfil-" + etiqueta, vinculo: "vinculo-" + etiqueta,
		procedencia: "procedencia", registro: "registro-contexto-" + etiqueta,
		autenticacion: "autenticacion-" + etiqueta, asercion: "asercion-" + etiqueta, sesion: "sesion-" + etiqueta,
		controlSesion: "control-sesion-" + etiqueta, politicaGarantia: "politica-garantia-" + etiqueta}
	soporte, perfil, err := nuevoSoportePlantillasCTDesdeBaseDesarrollo(base, ahora, d)
	if err != nil {
		return nil, errMontajePreparacionBasesV3
	}
	a, err := c.Ambito.dominio()
	if err != nil {
		return nil, errMontajePreparacionBasesV3
	}
	ambitos := []core.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{a.OrganizacionRef()}}}
	if a.UnidadGestionRef() != "" {
		ambitos = append(ambitos, core.AmbitoPerfil{Clave: "unidad_gestion_ref", Valores: []string{a.UnidadGestionRef()}})
	}
	par := paresPreparacionBasesHTTPV3()[i]
	rol := nombre + "_preparacion_bases_bolsa"
	plantilla, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(soporte.contexto.Resultado.Contexto.PersonaRef,
		perfil, ahora, rol, rol, etiqueta,
		[]core.ConcesionRol{{Accion: par.accion, ModuloID: "bolsa", TipoRecurso: bolsaports.TipoRecursoPreparacionBases,
			Finalidades: []string{bolsaports.FinalidadPreparacionBases}, CamposPermitidos: bolsaapp.CamposPreparacionBasesV3(par.accion), GarantiaMinima: core.AuthAssuranceHigh}}, ambitos)
	if err != nil {
		return nil, errMontajePreparacionBasesV3
	}
	prefijo := "acto:seleccion:preparacion-bases:" + nombre + ":"
	return &perfilPreparacionBasesV3{soporte: soporte, plantilla: plantilla, accion: par.accion, ruta: par.ruta,
		audiencia: DescriptoresMaterialPreparacionBasesV3()[i].Audiencia, motivo: motivo, ambito: a,
		actoAsignacion: prefijo + "asignacion:v1", actoControl: prefijo + "control-rol:v1", actoSesion: prefijo + "sesion:v1", provision: provision}, nil
}

// Sólo arranque/CLI. Si ya existe una asignación propia se conserva también
// cuando está revocada o restringida: el PDP deniega cada petición en vivo.
func asegurarPerfilPreparacionBasesV3(ctx context.Context, gobierno *pgxpool.Pool, p *perfilPreparacionBasesV3, ahora time.Time) error {
	if ctx == nil || ctx.Err() != nil || gobierno == nil || p == nil || p.soporte == nil || p.plantilla.Validar() != nil || !p.provision.valida() {
		return errMontajePreparacionBasesV3
	}
	autoridad := autoridadPostgreSQLDesarrollo{pool: gobierno, vinculo: p.soporte.contexto.Vinculo,
		prefijoBloqueo: "vec:seleccion:preparacion-bases:autorizacion:", actoAsignacion: p.actoAsignacion,
		actoControlRol: p.actoControl, actoSesion: p.actoSesion, exigirOrigenOperativo: true}
	actual, existe, err := leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, gobierno, p.perfilRef())
	if err != nil {
		return errMontajePreparacionBasesV3
	}
	if !existe {
		if p.provision.AprobacionRef != "" {
			return errMontajePreparacionBasesV3
		}
		autoridad.soloInicial = true
		i, err := autoridad.prepararInstantanea(ctx, p.plantilla, true)
		if err != nil || i.AsignacionPerfil.Version != 1 || autoridad.publicarInstantanea(ctx, i) != nil {
			return errMontajePreparacionBasesV3
		}
		return nil
	}
	if actual.instantanea.Validar() != nil || actual.actoAsignacion != p.actoAsignacion || actual.actoControl != p.actoControl ||
		actual.instantanea.VersionRol.RolID != p.plantilla.VersionRol.RolID ||
		actual.instantanea.AsignacionPerfil.PrincipalID != p.plantilla.AsignacionPerfil.PrincipalID ||
		actual.instantanea.AsignacionPerfil.PerfilActivoRef != p.perfilRef() {
		return errMontajePreparacionBasesV3
	}
	if p.provision.AprobacionRef == "" {
		return nil
	}
	if !preimagenPreparacionBasesProvisionable(actual, p, ahora) {
		return errMontajePreparacionBasesV3
	}
	objetivo, err := autoridad.prepararInstantanea(ctx, p.plantilla, false)
	if err != nil || objetivo.AsignacionPerfil.Version != actual.instantanea.AsignacionPerfil.Version+1 ||
		objetivo.AsignacionPerfil.AsignacionID != actual.instantanea.AsignacionPerfil.AsignacionID ||
		autoridad.publicarInstantaneaDesdePreimagen(ctx, objetivo, actual.instantanea) != nil {
		return errMontajePreparacionBasesV3
	}
	return nil
}

func preimagenPreparacionBasesProvisionable(a instantaneaPublicadaDesarrollo, p *perfilPreparacionBasesV3, ahora time.Time) bool {
	if p == nil || p.provision.AprobacionRef == "" || !p.provision.valida() {
		return false
	}
	h, err := a.instantanea.AsignacionPerfil.HuellaSHA256()
	return err == nil && h == p.provision.PreimagenSHA256 && origenOperativoPublicadoCTDesarrollo(a, p.actoAsignacion, ahora) &&
		a.actoControl == p.actoControl && a.instantanea.VersionRol.RolID == p.plantilla.VersionRol.RolID &&
		a.instantanea.AsignacionPerfil.PrincipalID == p.plantilla.AsignacionPerfil.PrincipalID &&
		a.instantanea.AsignacionPerfil.PerfilActivoRef == p.perfilRef() &&
		reflect.DeepEqual(a.instantanea.VersionRol.Concesiones, p.plantilla.VersionRol.Concesiones) &&
		reflect.DeepEqual(a.instantanea.AsignacionPerfil.Ambitos, p.plantilla.AsignacionPerfil.Ambitos)
}

func DescriptoresMaterialPreparacionBasesV3() [2]descriptorMaterialConsumidorV3Desarrollo {
	return [2]descriptorMaterialConsumidorV3Desarrollo{
		{Audiencia: bolsaapp.AudienciaGuardarPreparacionBasesV3, Dominio: "vec.bolsa.preparacion-bases.guardar.capacidad-v3", Prefijo: "clave:capacidad:bolsa-preparacion-bases-guardar:", ProveedorNominal: "proveedor-material-bolsa-preparacion-bases-guardar"},
		{Audiencia: bolsaapp.AudienciaConsultarPreparacionBasesV3, Dominio: "vec.bolsa.preparacion-bases.consultar.capacidad-v3", Prefijo: "clave:capacidad:bolsa-preparacion-bases-consultar:", ProveedorNominal: "proveedor-material-bolsa-preparacion-bases-consultar"},
	}
}

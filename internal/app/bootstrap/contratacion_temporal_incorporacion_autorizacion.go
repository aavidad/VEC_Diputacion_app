package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"maps"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	alta "vec-diputacion-granada/internal/modules/personal/adapters/contrataciontemporal"
	"vec-diputacion-granada/internal/modules/personal/adapters/fuenteejercicio"
	lectura "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// La petición coteja el recurso contra configuración y planes confiables.
// No prepara ni publica asignaciones: el PDP consume permisos provisionados
// previamente y PostgreSQL los revalida dentro de la transacción del efecto.
type autoridadOperacionesIncorporacionV2 struct {
	soporte                                  *soporteAltaContratacionTemporalDesarrollo
	consultas                                *autoridadConsultasRRHHDesarrollo
	delegado                                 vp.AutorizadorSolicitudLigadaV3
	referencias                              ReferenciasCTIncorporacionDesarrollo
	planes                                   inc.FuentePlanesPreparacionV2
	personal                                 []byte
	ternaPersonal                            fuenteejercicio.TernaEsperada
	motivoAlta, motivoLectura, motivoDetalle core.ReferenciaEntradaCatalogo
	reloj                                    ct.Reloj
	nominales                                *perfilesNominalesIncorporacion
}

func nuevaAutoridadOperacionesIncorporacionV2(s *soporteAltaContratacionTemporalDesarrollo, consultas *autoridadConsultasRRHHDesarrollo, delegado vp.AutorizadorSolicitudLigadaV3, c archivoIncorporacionV2, planes inc.FuentePlanesPreparacionV2, personal []byte, detalle core.ReferenciaEntradaCatalogo, reloj ct.Reloj, nominales ...*perfilesNominalesIncorporacion) (*autoridadOperacionesIncorporacionV2, error) {
	if s == nil || consultas == nil || consultas.soporte != s || s.sello == nil || !c.Referencias.valida() ||
		dependenciaEsNulaContratacionTemporalDesarrollo(delegado) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(planes) || dependenciaEsNulaContratacionTemporalDesarrollo(reloj) {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	var perfiles *perfilesNominalesIncorporacion
	if len(nominales) > 1 {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	contextoBase := s.contexto
	if len(nominales) == 1 {
		perfiles = nominales[0]
		if perfiles == nil || perfiles.soporte != s || perfiles.consultas != consultas || perfiles.detalle == nil || perfiles.alta == nil || perfiles.ct == nil {
			return nil, ct.ErrComposicionIncorporacionAplicacion
		}
		contextoBase = perfiles.ct.contexto
	}
	v, err := contextoBase.Vinculo.Datos()
	if err != nil || v.PrincipalID != c.Referencias.PrincipalV3Ref || v.PerfilActivoRef != c.Referencias.PerfilV3Ref ||
		!core.ReferenciaMotivoAutorizacionV2Valida(c.MotivoAlta) || !core.ReferenciaMotivoAutorizacionV2Valida(c.MotivoLectura) ||
		c.MotivoAlta.CatalogoID != c.MotivoLectura.CatalogoID || !core.ReferenciaMotivoAutorizacionV2Valida(detalle) {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	return &autoridadOperacionesIncorporacionV2{s, consultas, delegado, c.Referencias, planes, bytes.Clone(personal), c.TernaPersonal, c.MotivoAlta, c.MotivoLectura, detalle, reloj, perfiles}, nil
}

func (a *autoridadOperacionesIncorporacionV2) ExigirSolicitudLigadaV3(ctx context.Context, solicitud core.SolicitudAutorizacionLigadaV3, resultado core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	vacia, confirmacion := core.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}
	if a == nil || ctx == nil || a.soporte == nil || a.soporte.sello == nil || ctx.Value(claveIncorporacionV2Desarrollo{}) != a.soporte.sello {
		return vacia, confirmacion, ct.ErrAutorizacionDenegada
	}
	if err := ctx.Err(); err != nil {
		return vacia, confirmacion, err
	}
	var contexto ct.ContextoAutorizacionAltaV3
	if a.nominales == nil {
		var err error
		contexto, err = a.consultas.contextoConsultaRRHHDesarrollo(ctx)
		if err != nil {
			return vacia, confirmacion, err
		}
	}
	datos, err := solicitud.Datos()
	var perfil *perfilFijoCTDesarrollo
	if a.nominales != nil {
		switch datos.Accion {
		case ct.AccionConsultarDetalleRRHH:
			perfil = a.nominales.detalle
		case alta.AccionAltaEjercicio:
			perfil = a.nominales.alta
		case lectura.Accion, ct.AccionConfirmarIncorporacion:
			perfil = a.nominales.ct
		default:
			return vacia, confirmacion, ct.ErrAutorizacionDenegada
		}
		contexto, err = a.nominales.resolver(ctx, perfil)
		if err != nil {
			return vacia, confirmacion, err
		}
	}
	perfilEsperado := a.referencias.PerfilV3Ref
	if perfil != nil {
		perfilEsperado = perfil.perfilRef()
	}
	v, ev := datos.VinculoAutenticacionActor.Datos()
	if err != nil || ev != nil || v.PrincipalID != a.referencias.PrincipalV3Ref || v.PerfilActivoRef != perfilEsperado ||
		datos.VinculoAutenticacionActor.ValidarPara(resultado) != nil {
		return vacia, confirmacion, ct.ErrAutorizacionDenegada
	}
	// La autoridad de aplicación registra otra captura legítima del mismo
	// actor. Conservamos su recibo y sólo cotejamos la identidad y sesión de
	// esta petición, sin equiparar los identificadores/huellas de ambas capturas.
	propio := ct.ContextoAutorizacionAltaV3{Vinculo: datos.VinculoAutenticacionActor, Resultado: resultado}
	base, err := contexto.Vinculo.Datos()
	if err != nil {
		return vacia, confirmacion, ct.ErrAutorizacionDenegada
	}
	autenticacion, esperada := v.Autenticacion(), base.Autenticacion()
	ahora := a.reloj.Ahora()
	if autenticacion.SesionRevalidadaEn.Before(esperada.SesionRevalidadaEn) || autenticacion.SesionRevalidadaEn.After(ahora) {
		return vacia, confirmacion, ct.ErrAutorizacionDenegada
	}
	esperada.SesionRevalidadaEn = autenticacion.SesionRevalidadaEn
	conservaActor := a.consultas.contextoConsultaRRHHConservaActor(propio)
	if perfil != nil {
		conservaActor = mismoContextoEsperadoRegistradoDesarrollo(perfil.contextoEsperadoRegistrado, resultado)
	}
	if autenticacion != esperada || !conservaActor || propio.ValidarPara(ct.SolicitudResolverContextoAutorizacionAltaV3{AutenticacionRef: base.AutenticacionRef, SesionRef: base.SesionRef, PerfilRef: base.PerfilActivoRef}, ahora) != nil {
		return vacia, confirmacion, ct.ErrAutorizacionDenegada
	}
	if err := a.validarSolicitud(ctx, datos, propio); err != nil {
		return vacia, confirmacion, ct.ErrAutorizacionDenegada
	}
	if perfil != nil {
		if err := a.nominales.consumir(ctx, perfil); err != nil {
			return vacia, confirmacion, err
		}
	}
	return a.delegado.ExigirSolicitudLigadaV3(ctx, solicitud, resultado)
}

func (a *autoridadOperacionesIncorporacionV2) validarSolicitud(ctx context.Context, d core.DatosSolicitudAutorizacionLigadaV3, contexto ct.ContextoAutorizacionAltaV3) error {
	fallo := ct.ErrAutorizacionDenegada
	r := d.Recurso
	exp := r.Referencia
	if d.Accion == alta.AccionAltaEjercicio || d.Accion == lectura.Accion {
		exp = r.Atributos["expediente_ref"]
	}
	var modulo, tipo, finalidad string
	var motivo core.ReferenciaEntradaCatalogo
	ambitos := map[string]string{"organizacion_ref": a.referencias.OrganizacionRef}
	var p inc.PlanPreparacionDurableV2
	if d.Accion != ct.AccionConsultarDetalleRRHH {
		var err error
		p, err = a.planes.ResolverPlan(ctx, a.referencias.OrganizacionRef, exp)
		if err != nil {
			return errors.Join(fallo, err)
		}
		if p.OrganizacionRef != a.referencias.OrganizacionRef || p.UnidadRef != a.referencias.UnidadRef || p.SolicitudPersonal.ExpedienteRef != exp {
			return fallo
		}
	}
	switch d.Accion {
	case ct.AccionConsultarDetalleRRHH:
		modulo, tipo, finalidad = ct.ModuloContratacion, ct.TipoRecursoExpediente, ct.FinalidadConsultarDetalleRRHH
		motivo = a.motivoDetalle
		if a.nominales == nil {
			if !a.soporte.solicitudAutorizacionConsultaRRHHDesarrolloValida(httpinterno.RutaConsultaDetalleRRHH, d) {
				return fallo
			}
		} else if !dom.ReferenciaOpacaValida(exp) || len(r.Atributos) != 2 || r.Atributos["consulta_dominio"] != ct.DominioHuellaConsultaDetalleRRHH || !huellaSHA256ValidaContratacionTemporalDesarrollo(r.Atributos["consulta_huella_sha256"]) {
			return fallo
		}
		ambitos["clase_ambito"], ambitos["ambito_ref"] = string(ct.AmbitoOrganizacionRRHH), a.referencias.OrganizacionRef
	case alta.AccionAltaEjercicio:
		modulo, tipo, finalidad = "personal", alta.TipoRecursoAltaEjercicio, alta.FinalidadAltaEjercicio
		motivo = a.motivoAlta
		if p.FuentePersonal != a.ternaPersonal {
			return fallo
		}
		fuente, err := fuenteejercicio.NuevaFuenteEjercicio(a.personal, a.ternaPersonal)
		if err != nil {
			return errors.Join(fallo, err)
		}
		vinculo, err := fuente.Resolver(ctx, p.SolicitudPersonal)
		if err != nil {
			return errors.Join(fallo, err)
		}
		// El recurso completo se reconstruye con el plan/fuente privados, nunca
		// con un centro o material proporcionados por la solicitud de autorización.
		esperado, err := alta.RecursoAltaEjercicio(alta.MaterialAlta{Preparacion: alta.PreparacionAlta{Solicitud: p.SolicitudPersonal, Fuente: p.FuentePersonal, Vinculo: vinculo}, OrganizacionRef: a.referencias.OrganizacionRef, ActorRef: a.referencias.PrincipalV3Ref, PerfilRef: contexto.Resultado.Contexto.PerfilActivoRef})
		if err != nil {
			return errors.Join(fallo, err)
		}
		if r.Referencia != esperado.Referencia || !maps.Equal(r.Atributos, esperado.Atributos) {
			return fallo
		}
		ambitos["centro_ref"] = vinculo.CentroRef
	case lectura.Accion:
		modulo, tipo, finalidad = "personal", lectura.TipoRecursoV2, lectura.Finalidad
		motivo = a.motivoLectura
		selector := lectura.Selector{OrganizacionRef: a.referencias.OrganizacionRef, SolicitudRef: p.SolicitudPersonal.SolicitudRef, ExpedienteRef: p.SolicitudPersonal.ExpedienteRef, VersionExpediente: p.SolicitudPersonal.VersionExpediente,
			ResultadoRef: r.Atributos["resultado_ref"], ReciboRef: r.Atributos["recibo_ref"], RelacionRef: r.Atributos["relacion_ref"], OcupacionRef: r.Atributos["ocupacion_ref"], MaterialSHA256: r.Atributos["material_sha256"]}
		material, err := lectura.NuevoMaterialV2(selector, a.referencias.UnidadRef, contexto, a.reloj.Ahora())
		if err != nil {
			return errors.Join(fallo, err)
		}
		esperado, err := material.Recurso()
		if err != nil {
			return errors.Join(fallo, err)
		}
		if r.Referencia != esperado.Referencia || !maps.Equal(r.Atributos, esperado.Atributos) {
			return fallo
		}
		ambitos["unidad_ref"] = a.referencias.UnidadRef
	case ct.AccionConfirmarIncorporacion:
		modulo, tipo, finalidad = ct.ModuloContratacion, ct.TipoRecursoConfirmacionIncorporacionV2, ct.FinalidadConfirmarIncorporacion
		motivo = p.MotivoV3
		cor, err := d.Correlacion.ValorCanonico()
		if err != nil {
			return errors.Join(fallo, err)
		}
		if len(r.Atributos) != 8 || !huellaSHA256ValidaContratacionTemporalDesarrollo(r.Atributos["material_sha256"]) || !dom.ReferenciaOpacaValida(r.Atributos["correlacion_seguimiento_ref"]) || motivo.CatalogoID != a.motivoAlta.CatalogoID || r.Atributos["principal_v3_ref"] != a.referencias.PrincipalV3Ref || r.Atributos["perfil_v3_ref"] != a.referencias.PerfilV3Ref || r.Atributos["actor_seguimiento_ref"] != a.referencias.ActorRef || r.Atributos["correlacion_v3_ref"] != cor || r.Atributos["motivo_v3_ref"] != motivo.EntradaClave || r.Atributos["tipo_validacion"] != "ejercicio_sintetico" {
			return fallo
		}
		ambitos["unidad_ref"] = a.referencias.UnidadRef
	default:
		return fallo
	}
	if r.ModuloID != modulo || r.Tipo != tipo || d.Finalidad != finalidad || d.ReferenciaMotivo != motivo || !maps.Equal(r.Ambitos, ambitos) {
		return fallo
	}
	return nil
}

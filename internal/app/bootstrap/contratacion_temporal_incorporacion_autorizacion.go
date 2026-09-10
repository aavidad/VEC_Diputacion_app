package bootstrap

import (
	"bytes"
	"context"
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

// El opt-in sintético declara cuatro permisos nominales. Lectura y CT
// comparten asignación exacta porque se consumen juntos en la transacción. Los planes sellados sólo acotan expedientes y ámbitos;
// nunca aportan acciones, roles ni concesiones. El PDP y los consumidores
// PostgreSQL siguen comprobando la asignación publicada y su vigencia.
// No se modifica el perfil registrado ni se amplían los roles de otras rutas.
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
}

func nuevaAutoridadOperacionesIncorporacionV2(s *soporteAltaContratacionTemporalDesarrollo, consultas *autoridadConsultasRRHHDesarrollo, delegado vp.AutorizadorSolicitudLigadaV3, c archivoIncorporacionV2, planes inc.FuentePlanesPreparacionV2, personal []byte, detalle core.ReferenciaEntradaCatalogo, reloj ct.Reloj) (*autoridadOperacionesIncorporacionV2, error) {
	if s == nil || consultas == nil || consultas.soporte != s || s.sello == nil || !c.Referencias.valida() ||
		dependenciaEsNulaContratacionTemporalDesarrollo(s.autoridadAsignaciones) || dependenciaEsNulaContratacionTemporalDesarrollo(delegado) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(planes) || dependenciaEsNulaContratacionTemporalDesarrollo(reloj) {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	v, err := s.contexto.Vinculo.Datos()
	if err != nil || v.PrincipalID != c.Referencias.PrincipalV3Ref || v.PerfilActivoRef != c.Referencias.PerfilV3Ref ||
		!core.ReferenciaMotivoAutorizacionV2Valida(c.MotivoAlta) || !core.ReferenciaMotivoAutorizacionV2Valida(c.MotivoLectura) ||
		c.MotivoAlta.CatalogoID != c.MotivoLectura.CatalogoID || !core.ReferenciaMotivoAutorizacionV2Valida(detalle) {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	return &autoridadOperacionesIncorporacionV2{s, consultas, delegado, c.Referencias, planes, bytes.Clone(personal), c.TernaPersonal, c.MotivoAlta, c.MotivoLectura, detalle, reloj}, nil
}

func (a *autoridadOperacionesIncorporacionV2) ExigirSolicitudLigadaV3(ctx context.Context, solicitud core.SolicitudAutorizacionLigadaV3, resultado core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	vacia, confirmacion := core.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}
	if a == nil || ctx == nil || a.soporte == nil || a.soporte.sello == nil || ctx.Value(claveIncorporacionV2Desarrollo{}) != a.soporte.sello {
		return vacia, confirmacion, ct.ErrAutorizacionDenegada
	}
	if err := ctx.Err(); err != nil {
		return vacia, confirmacion, err
	}
	contexto, err := a.consultas.contextoConsultaRRHHDesarrollo(ctx)
	if err != nil {
		return vacia, confirmacion, ct.ErrAutorizacionDenegada
	}
	datos, err := solicitud.Datos()
	v, ev := datos.VinculoAutenticacionActor.Datos()
	if err != nil || ev != nil || v.PrincipalID != a.referencias.PrincipalV3Ref || v.PerfilActivoRef != a.referencias.PerfilV3Ref ||
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
	if autenticacion != esperada || !a.consultas.contextoConsultaRRHHConservaActor(propio) || propio.ValidarPara(ct.SolicitudResolverContextoAutorizacionAltaV3{AutenticacionRef: base.AutenticacionRef, SesionRef: base.SesionRef, PerfilRef: base.PerfilActivoRef}, ahora) != nil {
		return vacia, confirmacion, ct.ErrAutorizacionDenegada
	}
	instantanea, err := a.instantanea(ctx, datos, propio)
	if err != nil {
		return vacia, confirmacion, ct.ErrAutorizacionDenegada
	}
	autoridad := a.soporte.autoridadAsignaciones
	preparada, err := autoridad.PrepararInstantanea(ctx, instantanea)
	if err != nil {
		return vacia, confirmacion, ct.ErrAutorizacionDenegada
	}
	if err = autoridad.PublicarInstantanea(ctx, preparada); err != nil {
		return vacia, confirmacion, ct.ErrAutorizacionDenegada
	}
	// Misma fuente PostgreSQL: una carrera con otra operación da obsolescencia,
	// nunca un permiso ficticio ni un reintento que oculte efectos anteriores.
	return a.delegado.ExigirSolicitudLigadaV3(ctx, solicitud, resultado)
}

func (a *autoridadOperacionesIncorporacionV2) instantanea(ctx context.Context, d core.DatosSolicitudAutorizacionLigadaV3, contexto ct.ContextoAutorizacionAltaV3) (core.InstantaneaAutorizacion, error) {
	cero, fallo := core.InstantaneaAutorizacion{}, ct.ErrAutorizacionDenegada
	r := d.Recurso
	exp := r.Referencia
	if d.Accion == alta.AccionAltaEjercicio || d.Accion == lectura.Accion {
		exp = r.Atributos["expediente_ref"]
	}
	p, err := a.planes.ResolverPlan(ctx, a.referencias.OrganizacionRef, exp)
	if err != nil || p.OrganizacionRef != a.referencias.OrganizacionRef || p.UnidadRef != a.referencias.UnidadRef || p.SolicitudPersonal.ExpedienteRef != exp {
		return cero, fallo
	}
	var rol, modulo, tipo, finalidad string
	var motivo core.ReferenciaEntradaCatalogo
	ambitos := map[string]string{"organizacion_ref": a.referencias.OrganizacionRef}
	switch d.Accion {
	case ct.AccionConsultarDetalleRRHH:
		rol, modulo, tipo, finalidad = "incorporacion_detalle_desarrollo_v2", ct.ModuloContratacion, ct.TipoRecursoExpediente, ct.FinalidadConsultarDetalleRRHH
		motivo = a.motivoDetalle
		if !a.soporte.solicitudAutorizacionConsultaRRHHDesarrolloValida(httpinterno.RutaConsultaDetalleRRHH, d) {
			return cero, fallo
		}
		ambitos["clase_ambito"], ambitos["ambito_ref"] = string(ct.AmbitoOrganizacionRRHH), a.referencias.OrganizacionRef
	case alta.AccionAltaEjercicio:
		rol, modulo, tipo, finalidad = "incorporacion_alta_personal_desarrollo_v2", "personal", alta.TipoRecursoAltaEjercicio, alta.FinalidadAltaEjercicio
		motivo = a.motivoAlta
		if p.FuentePersonal != a.ternaPersonal {
			return cero, fallo
		}
		fuente, err := fuenteejercicio.NuevaFuenteEjercicio(a.personal, a.ternaPersonal)
		if err != nil {
			return cero, fallo
		}
		vinculo, err := fuente.Resolver(ctx, p.SolicitudPersonal)
		if err != nil {
			return cero, fallo
		}
		// El recurso completo se reconstruye con el plan/fuente privados, nunca
		// con un centro o material proporcionados por la solicitud de autorización.
		esperado, err := alta.RecursoAltaEjercicio(alta.MaterialAlta{Preparacion: alta.PreparacionAlta{Solicitud: p.SolicitudPersonal, Fuente: p.FuentePersonal, Vinculo: vinculo}, OrganizacionRef: a.referencias.OrganizacionRef, ActorRef: a.referencias.PrincipalV3Ref, PerfilRef: a.referencias.PerfilV3Ref})
		if err != nil || r.Referencia != esperado.Referencia || !maps.Equal(r.Atributos, esperado.Atributos) {
			return cero, fallo
		}
		ambitos["centro_ref"] = vinculo.CentroRef
	case lectura.Accion:
		rol, modulo, tipo, finalidad = "incorporacion_personal_ct_desarrollo_v2", "personal", lectura.TipoRecursoV2, lectura.Finalidad
		motivo = a.motivoLectura
		selector := lectura.Selector{OrganizacionRef: a.referencias.OrganizacionRef, SolicitudRef: p.SolicitudPersonal.SolicitudRef, ExpedienteRef: p.SolicitudPersonal.ExpedienteRef, VersionExpediente: p.SolicitudPersonal.VersionExpediente,
			ResultadoRef: r.Atributos["resultado_ref"], ReciboRef: r.Atributos["recibo_ref"], RelacionRef: r.Atributos["relacion_ref"], OcupacionRef: r.Atributos["ocupacion_ref"], MaterialSHA256: r.Atributos["material_sha256"]}
		material, err := lectura.NuevoMaterialV2(selector, a.referencias.UnidadRef, contexto, a.reloj.Ahora())
		if err != nil {
			return cero, fallo
		}
		esperado, err := material.Recurso()
		if err != nil || r.Referencia != esperado.Referencia || !maps.Equal(r.Atributos, esperado.Atributos) {
			return cero, fallo
		}
		ambitos["unidad_ref"] = a.referencias.UnidadRef
	case ct.AccionConfirmarIncorporacion:
		rol, modulo, tipo, finalidad = "incorporacion_personal_ct_desarrollo_v2", ct.ModuloContratacion, ct.TipoRecursoConfirmacionIncorporacionV2, ct.FinalidadConfirmarIncorporacion
		motivo = p.MotivoV3
		cor, err := d.Correlacion.ValorCanonico()
		if err != nil || len(r.Atributos) != 8 || !huellaSHA256ValidaContratacionTemporalDesarrollo(r.Atributos["material_sha256"]) || !dom.ReferenciaOpacaValida(r.Atributos["correlacion_seguimiento_ref"]) || motivo.CatalogoID != a.motivoAlta.CatalogoID || r.Atributos["principal_v3_ref"] != a.referencias.PrincipalV3Ref || r.Atributos["perfil_v3_ref"] != a.referencias.PerfilV3Ref || r.Atributos["actor_seguimiento_ref"] != a.referencias.ActorRef || r.Atributos["correlacion_v3_ref"] != cor || r.Atributos["motivo_v3_ref"] != motivo.EntradaClave || r.Atributos["tipo_validacion"] != "ejercicio_sintetico" {
			return cero, fallo
		}
		ambitos["unidad_ref"] = a.referencias.UnidadRef
	default:
		return cero, fallo
	}
	if r.ModuloID != modulo || r.Tipo != tipo || d.Finalidad != finalidad || d.ReferenciaMotivo != motivo || !maps.Equal(r.Ambitos, ambitos) {
		return cero, fallo
	}
	// Orden fijo para un documento/huella estables; no se copian ámbitos libres.
	perfil := []core.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{a.referencias.OrganizacionRef}}}
	for _, k := range []string{"clase_ambito", "ambito_ref", "centro_ref", "unidad_ref"} {
		if v, ok := ambitos[k]; ok {
			perfil = append(perfil, core.AmbitoPerfil{Clave: k, Valores: []string{v}})
		}
	}
	concesiones := []core.ConcesionRol{{Accion: d.Accion, ModuloID: modulo, TipoRecurso: tipo, Finalidades: []string{finalidad}, GarantiaMinima: core.AuthAssuranceHigh}}
	// CT se emite antes de la lectura dentro del escritor. Una asignación nueva
	// para esa lectura invalidaría CT antes de consumir ambos permisos. Sólo
	// este par comparte ámbitos/rol; cada decisión, audiencia y motivo es propio.
	if d.Accion == lectura.Accion || d.Accion == ct.AccionConfirmarIncorporacion {
		concesiones = []core.ConcesionRol{
			{Accion: lectura.Accion, ModuloID: "personal", TipoRecurso: lectura.TipoRecursoV2, Finalidades: []string{lectura.Finalidad}, GarantiaMinima: core.AuthAssuranceHigh},
			{Accion: ct.AccionConfirmarIncorporacion, ModuloID: ct.ModuloContratacion, TipoRecurso: ct.TipoRecursoConfirmacionIncorporacionV2, Finalidades: []string{ct.FinalidadConfirmarIncorporacion}, GarantiaMinima: core.AuthAssuranceHigh},
		}
	}
	return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(a.referencias.PrincipalV3Ref, a.referencias.PerfilV3Ref, a.reloj.Ahora(), rol, "Permiso nominal de incorporación de ejercicio", rol, concesiones, perfil)
}

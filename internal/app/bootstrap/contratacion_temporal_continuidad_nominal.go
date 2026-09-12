package bootstrap

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	comp "vec-diputacion-granada/internal/app/composicion/interna/contrataciontemporal"
	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	pgct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	segct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/seguridad"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	pgvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	seg "vec-diputacion-granada/internal/vec/adapters/seguridad"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	appvec "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// Opt-in privado. Las instantáneas son publicaciones técnicas explícitas del
// ejercicio, nunca una política jurídica ni una concesión inferida del cargo.
// La terna de política de anotación es la versión inmutable de su propio rol.
type archivoContinuidadNominal struct {
	Anotacion archivoOperacionContinuidadNominal `json:"anotacion"`
	Cierre    archivoOperacionContinuidadNominal `json:"cierre"`
}
type archivoOperacionContinuidadNominal struct {
	Instantanea core.InstantaneaAutorizacion    `json:"instantanea"`
	Motivo      core.ReferenciaEntradaCatalogo  `json:"motivo"`
	Capacidad   archivoCapacidadIncorporacionV2 `json:"capacidad"`
}
type continuidadNominalDesarrollo struct {
	anotacion, cierre *autoridadContinuidadNominal
	pool              *pgxpool.Pool
	detalle           *appct.ServicioConsultaDetalleRRHH
}
type claveRutaContinuidadNominal struct{}

func rutaContinuidadNominal(r string) bool {
	return r == httpct.RutaAnotacionesAdministrativas || r == httpct.RutaRecuperacionAnotacionesAdministrativas || r == httpct.RutaCerrarAdministrativamenteSinCese || r == httpct.RutaPreparacionCierreSinCese
}

// fuenteRolContinuidadNominal no permite que una carrera con lectura cambie
// silenciosamente la terna autorizadora. El PDP vuelve a leer esta fuente; el
// registro y el consumidor comparan la asignación vigente en su transacción.
type fuenteRolContinuidadNominal struct {
	fuente      vp.FuenteAutorizacion
	referencias ReferenciasCTIncorporacionDesarrollo
	rol         core.VersionRol
	asignacion  core.AsignacionPerfil
}

func (f *fuenteRolContinuidadNominal) ObtenerInstantaneaAutorizacion(ctx context.Context, principal, perfil string) (core.InstantaneaAutorizacion, error) {
	var cero core.InstantaneaAutorizacion
	if ctx == nil || ctx.Err() != nil || f == nil || dependenciaBootstrapNula(f.fuente) || principal != f.referencias.PrincipalV3Ref || perfil != f.referencias.PerfilV3Ref {
		return cero, ct.ErrAutorizacionDenegada
	}
	i, e := f.fuente.ObtenerInstantaneaAutorizacion(ctx, principal, perfil)
	h, eh := i.VersionRol.HuellaSHA256()
	esperado, ee := f.rol.HuellaSHA256()
	if e != nil || eh != nil || ee != nil || i.Validar() != nil || i.AsignacionPerfil.PrincipalID != principal || i.AsignacionPerfil.PerfilActivoRef != perfil || i.VersionRol.Referencia() != f.rol.Referencia() || h != esperado {
		return cero, ct.ErrAutorizacionDenegada
	}

	esperadaAsignacion := f.asignacion
	esperadaAsignacion.AsignacionID = i.AsignacionPerfil.AsignacionID
	esperadaAsignacion.Version = i.AsignacionPerfil.Version
	ha, ea := esperadaAsignacion.HuellaSHA256()
	hb, eb := i.AsignacionPerfil.HuellaSHA256()
	if ea != nil || eb != nil || ha != hb {
		return cero, ct.ErrAutorizacionDenegada
	}
	return i, nil
}

type autoridadContinuidadNominal struct {
	soporte                            *soporteAltaContratacionTemporalDesarrollo
	consultas                          *autoridadConsultasRRHHDesarrollo
	referencias                        ReferenciasCTIncorporacionDesarrollo
	planes                             inc.FuentePlanesPreparacionV2
	configuracion                      archivoOperacionContinuidadNominal
	accion, finalidad, tipo, audiencia string
	fuente                             *fuenteRolContinuidadNominal
	pdp                                vp.AutorizadorSolicitudLigadaV3
	material                           *proveedorMaterialAltaContratacionTemporalDesarrollo
	emisor                             *confianza.EmisorCapacidadesAtestacionAutorizacionV3
	reloj                              ct.Reloj
}

func validarOperacionContinuidadNominal(c archivoOperacionContinuidadNominal, refs ReferenciasCTIncorporacionDesarrollo, accion, finalidad, tipo string, ahora time.Time) error {
	i := c.Instantanea
	a := i.AsignacionPerfil
	if !refs.valida() || i.Validar() != nil || len(i.Politicas) != 0 || !core.ReferenciaMotivoAutorizacionV2Valida(c.Motivo) || !dom.ReferenciaOpacaValida(i.VersionRol.RolID) ||
		a.PrincipalID != refs.PrincipalV3Ref || a.PerfilActivoRef != refs.PerfilV3Ref || a.Estado != core.EstadoAsignacionPerfilActiva || ahora.Before(a.VigenteDesde) || !ahora.Before(a.VigenteHasta) ||
		i.VersionRol.Estado != core.EstadoVersionRolPublicada || i.ControlVigenciaVersionRol.Estado != core.EstadoControlVigenciaVersionRolHabilitada || len(i.VersionRol.Concesiones) != 1 {
		return ct.ErrAutorizacionDenegada
	}
	crol := i.VersionRol.Concesiones[0]
	if crol.Accion != accion || crol.ModuloID != ct.ModuloContratacion || crol.TipoRecurso != tipo || len(crol.Finalidades) != 1 || crol.Finalidades[0] != finalidad || crol.GarantiaMinima != core.AuthAssuranceHigh {
		return ct.ErrAutorizacionDenegada
	}
	ambitos := map[string][]string{}
	for _, a := range a.Ambitos {
		ambitos[a.Clave] = a.Valores
	}
	org := ambitos["organizacion_ref"]
	if len(org) != 1 || org[0] != refs.OrganizacionRef || len(ambitos["expediente_ref"]) == 0 {
		return ct.ErrAutorizacionDenegada
	}
	claves := []string{"organizacion_ref", "expediente_ref", "fase_previa", "estado_previo"}
	if accion == ct.AccionAutorizacionCerrarAdministrativamente {
		claves = []string{"organizacion_ref", "expediente_ref", "seguimiento_ref"}
	}
	if len(ambitos) != len(claves) {
		return ct.ErrAutorizacionDenegada
	}
	for _, k := range claves {
		if len(ambitos[k]) == 0 {
			return ct.ErrAutorizacionDenegada
		}
	}
	return nil
}

func cargarContinuidadNominal(raiz *os.Root, c *archivoContinuidadNominal, refs ReferenciasCTIncorporacionDesarrollo, pools map[string]*pgxpool.Pool, alta *dependenciasAltaContratacionTemporalDesarrollo, consultas dependenciasConsultasRRHHDesarrollo, planes inc.FuentePlanesPreparacionV2, detalle *appct.ServicioConsultaDetalleRRHH, reloj relojContratacionTemporalDesarrollo) (*continuidadNominalDesarrollo, error) {
	if c == nil {
		return nil, nil
	}
	fallo := ct.ErrComposicionIncorporacionAplicacion
	if alta == nil || alta.soporte == nil || dependenciaBootstrapNula(alta.soporte.autoridadAsignaciones) || consultas.materialDetalle == nil || consultas.autoridad == nil || detalle == nil {
		return nil, fallo
	}
	if c.Anotacion.Instantanea.VersionRol.RolID == c.Cierre.Instantanea.VersionRol.RolID || c.Anotacion.Capacidad.ClaveID == c.Cierre.Capacidad.ClaveID || c.Anotacion.Capacidad.File == c.Cierre.Capacidad.File {
		return nil, fallo
	}
	fuente, e := pgvec.NuevoAlmacenAutorizacion(pools["fuente_autorizacion"])
	if e != nil {
		return nil, fallo
	}
	registro, e := pgvec.NuevoAlmacenAutorizacion(alta.postgresql.registroAutorizacion)
	if e != nil {
		return nil, fallo
	}
	resultado := &continuidadNominalDesarrollo{pool: pools["registro_ct"], detalle: detalle}
	for n, op := range []archivoOperacionContinuidadNominal{c.Anotacion, c.Cierre} {
		accion, finalidad, tipo, audiencia := string(dom.AccionRegistrarAnotacionAdministrativa), ct.FinalidadRegistrarAnotacionAdministrativa, ct.TipoRecursoAnotacionAdministrativa, pgct.AudienciaAnotacionAdministrativaV1
		if n == 1 {
			accion, finalidad, tipo, audiencia = ct.AccionAutorizacionCerrarAdministrativamente, ct.FinalidadAutorizacionCerrarAdministrativamente, ct.TipoRecursoCierreAdministrativo, pgct.AudienciaCierreAdministrativoSinCese
		}
		if validarOperacionContinuidadNominal(op, refs, accion, finalidad, tipo, reloj.Ahora()) != nil {
			return nil, fallo
		}
		lector := &fuenteRolContinuidadNominal{fuente, refs, op.Instantanea.VersionRol, op.Instantanea.AsignacionPerfil}
		motivos, e := pgvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(pools["motivos_autorizacion"], op.Motivo.CatalogoID)
		if e != nil {
			return nil, fallo
		}
		pdp, e := appvec.NuevoServicioAutorizacionSolicitudLigadaV3(lector, registro, registro, motivos, reloj, seg.GeneradorReferenciasCriptograficas{}, appvec.ConfiguracionServicioAutorizacion{})
		if e != nil {
			return nil, fallo
		}
		capacidad := op.Capacidad
		secreto, e := leerArchivoIncorporacionV2(raiz, capacidad.File, 256)
		if e != nil {
			return nil, fallo
		}
		h := sha256.Sum256(secreto)
		if hex.EncodeToString(h[:]) != capacidad.SHA256 {
			borrarBytes(secreto)
			return nil, fallo
		}
		clave, e := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(capacidad.ClaveID, capacidad.Version, secreto, capacidad.EmisorID, audiencia, confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, capacidad.Desde, capacidad.Hasta, time.Time{}, capacidad.RevisionGobierno, capacidad.HuellaGobierno)
		borrarBytes(secreto)
		if e != nil || reloj.Ahora().Before(capacidad.Desde) || !reloj.Ahora().Before(capacidad.Hasta) {
			return nil, fallo
		}
		emisor, e := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, reloj)
		if e != nil {
			return nil, fallo
		}
		a := &autoridadContinuidadNominal{alta.soporte, consultas.autoridad, refs, planes, op, accion, finalidad, tipo, audiencia, lector, pdp, consultas.materialDetalle, emisor, reloj}
		if n == 0 {
			resultado.anotacion = a
		} else {
			resultado.cierre = a
		}
	}
	return resultado, nil
}

func (a *autoridadContinuidadNominal) contexto(ctx context.Context) (ct.ContextoAutorizacionAltaV3, error) {
	var cero ct.ContextoAutorizacionAltaV3
	if a == nil || ctx == nil || a.soporte == nil || a.consultas == nil || ctx.Value(claveIncorporacionV2Desarrollo{}) != a.soporte.sello {
		return cero, ct.ErrAutorizacionDenegada
	}
	ruta, _ := ctx.Value(claveRutaContinuidadNominal{}).(string)
	if !rutaContinuidadNominal(ruta) || (a.accion == ct.AccionAutorizacionCerrarAdministrativamente) != (ruta == httpct.RutaCerrarAdministrativamenteSinCese || ruta == httpct.RutaPreparacionCierreSinCese) {
		return cero, ct.ErrAutorizacionDenegada
	}
	c, e := a.consultas.contextoConsultaRRHHDesarrollo(ctx)
	if e != nil {
		return cero, e
	}
	v, e := c.Vinculo.Datos()
	if e != nil || v.PrincipalID != a.referencias.PrincipalV3Ref || v.PerfilActivoRef != a.referencias.PerfilV3Ref {
		return cero, ct.ErrAutorizacionDenegada
	}
	return c, nil
}
func (a *autoridadContinuidadNominal) ResolverContextoCanalAnotacionAdministrativa(ctx context.Context) (httpct.ContextoCanalAnotacionAdministrativa, error) {
	c, e := a.contexto(ctx)
	if e != nil {
		return httpct.ContextoCanalAnotacionAdministrativa{}, e
	}
	v, e := c.Vinculo.Datos()
	if e != nil {
		return httpct.ContextoCanalAnotacionAdministrativa{}, e
	}
	return httpct.ContextoCanalAnotacionAdministrativa{AutenticacionRef: v.AutenticacionRef, SesionRef: v.SesionRef, PerfilRef: v.PerfilActivoRef, OrganizacionRef: a.referencias.OrganizacionRef}, nil
}
func (a *autoridadContinuidadNominal) ResolverContextoAutorizacionAltaV3(ctx context.Context, s ct.SolicitudResolverContextoAutorizacionAltaV3) (ct.ContextoAutorizacionAltaV3, error) {
	c, e := a.contexto(ctx)
	if e != nil {
		return ct.ContextoAutorizacionAltaV3{}, e
	}
	if c.ValidarPara(s, a.reloj.Ahora()) != nil {
		return ct.ContextoAutorizacionAltaV3{}, ct.ErrAutorizacionDenegada
	}
	return c, nil
}
func (a *autoridadContinuidadNominal) ResolverOrganizacionCierreAdministrativo(ctx context.Context) (string, error) {
	if _, e := a.contexto(ctx); e != nil {
		return "", e
	}
	return a.referencias.OrganizacionRef, nil
}
func (a *autoridadContinuidadNominal) ResolverSolicitudPersonalAnotacionAdministrativa(ctx context.Context, org, exp string) (string, error) {
	if _, e := a.contexto(ctx); e != nil {
		return "", e
	}
	if org != a.referencias.OrganizacionRef {
		return "", ct.ErrAutorizacionDenegada
	}
	p, e := a.planes.ResolverPlan(ctx, org, exp)
	if e != nil || p.OrganizacionRef != org || p.SolicitudPersonal.ExpedienteRef != exp || p.UnidadRef != a.referencias.UnidadRef {
		return "", ct.ErrAutorizacionDenegada
	}
	return p.SolicitudPersonal.SolicitudRef, nil
}
func (a *autoridadContinuidadNominal) publicar(ctx context.Context) (core.InstantaneaAutorizacion, error) {
	var cero core.InstantaneaAutorizacion
	if _, e := a.contexto(ctx); e != nil {
		return cero, e
	}
	if validarOperacionContinuidadNominal(a.configuracion, a.referencias, a.accion, a.finalidad, a.tipo, a.reloj.Ahora()) != nil {
		return cero, ct.ErrAutorizacionDenegada
	}
	preparada, e := a.soporte.autoridadAsignaciones.PrepararInstantanea(ctx, clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(a.configuracion.Instantanea))
	if e != nil {
		return cero, e
	}
	if e = a.soporte.autoridadAsignaciones.PublicarInstantanea(ctx, preparada); e != nil {
		return cero, e
	}
	return a.fuente.ObtenerInstantaneaAutorizacion(ctx, a.referencias.PrincipalV3Ref, a.referencias.PerfilV3Ref)
}
func (a *autoridadContinuidadNominal) ResolverPoliticaAnotacionAdministrativa(ctx context.Context, s ct.SolicitudResolverPoliticaAnotacionAdministrativa) (ct.PoliticaAnotacionAdministrativa, error) {
	var cero ct.PoliticaAnotacionAdministrativa
	if a == nil || s.Material.Validar() != nil || s.Material.OrganizacionRef != a.referencias.OrganizacionRef || s.Material.ActorRef != a.referencias.PrincipalV3Ref || s.Material.PerfilRef != a.referencias.PerfilV3Ref {
		return cero, ct.ErrAutorizacionDenegada
	}
	i, e := a.publicar(ctx)
	if e != nil {
		return cero, e
	}
	h, e := i.VersionRol.HuellaSHA256()
	if e != nil {
		return cero, e
	}
	hasta := s.Instante.Add(15 * time.Second)
	if i.AsignacionPerfil.VigenteHasta.Before(hasta) {
		hasta = i.AsignacionPerfil.VigenteHasta
	}
	c, e := a.contexto(ctx)
	if e != nil {
		return cero, e
	}
	v, e := c.Vinculo.Datos()
	if e != nil {
		return cero, e
	}
	if c.Resultado.Contexto.Instantanea.VigenteHasta.Before(hasta) {
		hasta = c.Resultado.Contexto.Instantanea.VigenteHasta
	}
	if v.SesionValidaHasta.Before(hasta) {
		hasta = v.SesionValidaHasta
	}
	p := ct.PoliticaAnotacionAdministrativa{MotivoAutorizacion: a.configuracion.Motivo, DefinicionRef: i.VersionRol.RolID, DefinicionVersion: uint64(i.VersionRol.Version), DefinicionHuellaSHA256: h, EvaluadaEn: s.Instante, ValidaHasta: hasta}
	if p.ValidarPara(s, s.Instante) != nil {
		return cero, ct.ErrAutorizacionDenegada
	}
	return p, nil
}
func (a *autoridadContinuidadNominal) ExigirSolicitudLigadaV3(ctx context.Context, s core.SolicitudAutorizacionLigadaV3, resultado core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	cero, conf := core.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}
	if a == nil || ctx == nil {
		return cero, conf, ct.ErrAutorizacionDenegada
	}
	if a.accion == ct.AccionAutorizacionCerrarAdministrativamente && ctx.Value(claveRutaContinuidadNominal{}) != httpct.RutaCerrarAdministrativamenteSinCese {
		return cero, conf, ct.ErrAutorizacionDenegada
	}
	c, e := a.contexto(ctx)
	if e != nil {
		return cero, conf, e
	}
	d, e := s.Datos()
	if e != nil || !d.VinculoAutenticacionActor.CoincideExactamenteCon(c.Vinculo) || d.VinculoAutenticacionActor.ValidarPara(resultado) != nil || d.Accion != a.accion || d.Finalidad != a.finalidad || d.ReferenciaMotivo != a.configuracion.Motivo || d.Recurso.ModuloID != ct.ModuloContratacion || d.Recurso.Tipo != a.tipo || d.Recurso.Ambitos["organizacion_ref"] != a.referencias.OrganizacionRef {
		return cero, conf, ct.ErrAutorizacionDenegada
	}
	exp := d.Recurso.Ambitos["expediente_ref"]
	if _, e = a.ResolverSolicitudPersonalAnotacionAdministrativa(ctx, a.referencias.OrganizacionRef, exp); e != nil {
		return cero, conf, e
	}
	i, e := a.publicar(ctx)
	if e != nil {
		return cero, conf, e
	}
	for _, amb := range i.AsignacionPerfil.Ambitos {
		v, ok := d.Recurso.Ambitos[amb.Clave]
		admite := false
		for _, valor := range amb.Valores {
			admite = admite || valor == v
		}
		if !ok || !admite {
			return cero, conf, ct.ErrAutorizacionDenegada
		}
	}
	if len(d.Recurso.Ambitos) != len(i.AsignacionPerfil.Ambitos) {
		return cero, conf, ct.ErrAutorizacionDenegada
	}
	if a.accion == string(dom.AccionRegistrarAnotacionAdministrativa) {
		h, e := i.VersionRol.HuellaSHA256()
		if e != nil || d.Recurso.Atributos["politica_ref"] != i.VersionRol.RolID || d.Recurso.Atributos["politica_version"] != strconv.Itoa(i.VersionRol.Version) || d.Recurso.Atributos["politica_huella_sha256"] != h {
			return cero, conf, ct.ErrAutorizacionDenegada
		}
	}
	return a.pdp.ExigirSolicitudLigadaV3(ctx, s, resultado)
}
func (a *autoridadContinuidadNominal) exportar(ctx context.Context, c ct.ContextoAutorizacionAltaV3, s core.SolicitudAutorizacionLigadaV3, d core.DecisionAutorizacionLigadaV3, confirmacion vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var cero vp.ExportacionMaterialConsumoAutorizacionAtestadaV3
	actual, e := a.contexto(ctx)
	if e != nil || !actual.Vinculo.CoincideExactamenteCon(c.Vinculo) {
		return cero, ct.ErrAutorizacionDenegada
	}
	orden, e := vp.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, d, a.configuracion.Motivo, c.Resultado)
	if e != nil || confirmacion.ValidarPara(orden) != nil || !confirmacion.DentroDeVentanaEn(a.reloj.Ahora()) {
		return cero, ct.ErrAutorizacionDenegada
	}
	at, e := a.material.atestador.Atestar(ctx, d, a.configuracion.Motivo, c.Resultado)
	if e != nil {
		return cero, e
	}
	prueba, e := a.material.confianza.Verificar(ctx, s, d, a.configuracion.Motivo, c.Resultado, at)
	if e != nil {
		return cero, e
	}
	cap, e := a.emisor.Emitir(ctx, s, d, a.configuracion.Motivo, c.Resultado, at, prueba)
	if e != nil {
		return cero, e
	}
	m, e := confianza.NuevoMaterialConsumoAutorizacionAtestadaV3(s, d, a.configuracion.Motivo, c.Resultado, at, prueba, cap, a.material.raiz)
	if e != nil {
		return cero, e
	}
	return m.ExportarMaterialParaConsumidor()
}
func (a *autoridadContinuidadNominal) ProveerMaterialAnotacionAdministrativa(ctx context.Context, o ct.OrdenConfirmarAnotacionAdministrativa) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	d, e := o.Evidencia.SolicitudV3.Datos()
	if e != nil {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, e
	}
	r := ct.RecursoAutorizacionAnotacionAdministrativa(o.Preparacion, o.Politica)
	if d.Accion != a.accion || d.Finalidad != a.finalidad || d.ReferenciaMotivo != a.configuracion.Motivo || d.Recurso.Referencia != r.Referencia || d.Recurso.ModuloID != r.ModuloID || d.Recurso.Tipo != r.Tipo || !maps.Equal(d.Recurso.Ambitos, r.Ambitos) || !maps.Equal(d.Recurso.Atributos, r.Atributos) {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ct.ErrAutorizacionDenegada
	}
	return a.exportar(ctx, o.Evidencia.Contexto, o.Evidencia.SolicitudV3, o.Evidencia.DecisionV3, o.Evidencia.ConfirmacionV3)
}
func (a *autoridadContinuidadNominal) AutorizarCierreAdministrativo(ctx context.Context, s ct.SolicitudTransaccionCierreAdministrativo) (pgct.AutorizacionCierreAdministrativo, error) {
	var cero pgct.AutorizacionCierreAdministrativo
	c, e := a.contexto(ctx)
	if e != nil {
		return cero, e
	}
	if s.Validar() != nil || s.Operacion != ct.OperacionCerrarAdministrativamenteSinCese || s.OrganizacionRef != a.referencias.OrganizacionRef {
		return cero, ct.ErrAutorizacionDenegada
	}
	cor, e := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seg.GeneradorReferenciasCriptograficas{})
	if e != nil {
		return cero, e
	}
	corV3, e := cor.ValorCanonico()
	if e != nil {
		return cero, e
	}
	var nonce [32]byte
	if _, e = rand.Read(nonce[:]); e != nil {
		return cero, e
	}
	corCT := "ref:" + hex.EncodeToString(nonce[:])
	r := core.RecursoAutorizable{Referencia: s.SeguimientoRef, ModuloID: ct.ModuloContratacion, Tipo: ct.TipoRecursoCierreAdministrativo, Ambitos: map[string]string{"organizacion_ref": s.OrganizacionRef, "expediente_ref": s.ExpedienteRef, "seguimiento_ref": s.SeguimientoRef}, Atributos: map[string]string{"operacion": string(s.Operacion), "version_esperada": strconv.FormatUint(s.VersionEsperada, 10), "transicion_clave": string(s.TransicionClave), "motivo_clave": string(s.MotivoClave), "principal_v3_ref": a.referencias.PrincipalV3Ref, "actor_seguimiento_ref": a.referencias.ActorRef, "correlacion_v3_ref": corV3, "correlacion_seguimiento_ref": corCT}}
	solicitud, e := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: c.Vinculo, Accion: a.accion, Finalidad: a.finalidad, Recurso: r, ReferenciaMotivo: a.configuracion.Motivo, Correlacion: cor})
	if e != nil {
		return cero, e
	}
	d, confirmacion, e := a.ExigirSolicitudLigadaV3(ctx, solicitud, c.Resultado)
	if e != nil {
		return cero, e
	}
	x, e := a.exportar(ctx, c, solicitud, d, confirmacion)
	if e != nil {
		return cero, e
	}
	return pgct.AutorizacionCierreAdministrativo{Contexto: c, Solicitud: solicitud, Decision: d, Confirmacion: confirmacion, Motivo: a.configuracion.Motivo, ActorRef: a.referencias.ActorRef, PerfilRef: a.referencias.PerfilV3Ref, UnidadRef: a.referencias.UnidadRef, CorrelacionRef: corCT, Exportacion: x}, nil
}

func (c *continuidadNominalDesarrollo) rutas(derivador *derivadorIdentidadOperacionDesarrollo) ([]httpapi.RutaExacta, error) {
	if c == nil {
		return nil, nil
	}
	aa, ra, e := configuracionesHMACAltaContratacionTemporalDesarrollo(derivador, ct.DominioAmbitoIdempotenciaAnotacionAdministrativa, true)
	if e != nil {
		return nil, e
	}
	ah, rh, e := configuracionesHMACAltaContratacionTemporalDesarrollo(derivador, ct.DominioHuellaPeticionAnotacionAdministrativa, false)
	if e != nil {
		return nil, e
	}
	sellos, e := segct.NuevaAutoridadSellosAnotacionAdministrativaHMAC(aa, ra, ah, rh)
	if e != nil {
		return nil, e
	}
	lector, e := pgct.NuevoLectorMaterialAnotacionAdministrativaPostgreSQL(c.pool)
	if e != nil {
		return nil, e
	}
	ejecutor, e := NuevoEjecutorAnotacionAdministrativaDesarrollo(ConfiguracionAnotacionAdministrativaDesarrollo{EjecutorCT: c.pool, Contextos: c.anotacion, Solicitudes: c.anotacion, Ambitos: sellos, Huellas: sellos, Politicas: c.anotacion, Correlaciones: seg.GeneradorReferenciasCriptograficas{}, Autorizador: c.anotacion, Reloj: c.anotacion.reloj, Proveedor: c.anotacion, Detalle: c.detalle, Lector: lector})
	if e != nil {
		return nil, e
	}
	rutas, e := comp.NuevasRutasAnotacionAdministrativa(c.anotacion, ejecutor)
	if e != nil {
		return nil, e
	}
	servicio, e := NuevoServicioCierreAdministrativoSinCeseDesarrollo(ConfiguracionCierreAdministrativoSinCeseDesarrollo{EjecutorCT: c.pool, Proveedor: c.cierre})
	if e != nil {
		return nil, e
	}
	cierre, e := comp.NuevaRutaCierreAdministrativoSinCese(c.cierre, servicio)
	if e != nil {
		return nil, e
	}
	rutas = append(rutas, cierre)
	lectorCierre, e := pgct.NuevoLectorPreparacionCierreAdministrativoPostgreSQL(c.pool)
	if e != nil {
		return nil, e
	}
	preparacion, e := httpct.NuevoManejadorPreparacionCierreSinCese(c.cierre, &lectorPreparacionCierreNominal{c.cierre, c.detalle, lectorCierre})
	if e != nil {
		return nil, e
	}
	rutas = append(rutas, httpapi.RutaExacta{Ruta: httpct.RutaPreparacionCierreSinCese, Manejador: preparacion})
	for n := range rutas {
		h := rutas[n].Manejador
		ruta := rutas[n].Ruta
		rutas[n].Manejador = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capacidad, ok := c.anotacion.soporte.capacidadValida(r.Context())
			if ok && capacidad.ruta == ruta {
				ctx, e := contextoDetalleIncorporacionV2Desarrollo(r.Context(), c.anotacion.soporte)
				if e == nil {
					r = r.WithContext(context.WithValue(ctx, claveRutaContinuidadNominal{}, ruta))
				}
			}
			h.ServeHTTP(w, r)
		})
	}
	return rutas, nil
}

// El lector privado nunca recibe una solicitud hasta que la consulta V3
// nominal acredita el acceso al expediente exacto de esta petición mTLS.
type lectorPreparacionCierreNominal struct {
	autoridad *autoridadContinuidadNominal
	detalle   httpct.ConsultorDetalleRRHH
	lector    ct.LectorPreparacionCierreAdministrativo
}

func (l *lectorPreparacionCierreNominal) ConsultarPreparacionCierreAdministrativo(ctx context.Context, s ct.SolicitudPreparacionCierreAdministrativo) (ct.PreparacionCierreAdministrativo, error) {
	var cero ct.PreparacionCierreAdministrativo
	if ctx == nil || l == nil || l.autoridad == nil || dependenciaBootstrapNula(l.detalle) || dependenciaBootstrapNula(l.lector) || s.Validar() != nil {
		return cero, ct.ErrConsultaPreparacionCierreAdministrativoInvalida
	}
	if ctx.Value(claveRutaContinuidadNominal{}) != httpct.RutaPreparacionCierreSinCese {
		return cero, ct.ErrAutorizacionDenegada
	}
	org, e := l.autoridad.ResolverOrganizacionCierreAdministrativo(ctx)
	if e != nil || org != s.OrganizacionRef {
		return cero, ct.ErrAutorizacionDenegada
	}
	consulta, e := ct.NuevaSolicitudDetalleRRHH(s.ExpedienteRef, 0)
	if e != nil {
		return cero, e
	}
	detalle, e := l.detalle.Consultar(ctx, consulta)
	if e != nil {
		return cero, e
	}
	if detalle.Resumen.ExpedienteRef != s.ExpedienteRef || detalle.Resumen.OrganizacionRef != org {
		return cero, ct.ErrAutorizacionDenegada
	}
	p, e := l.lector.ConsultarPreparacionCierreAdministrativo(ctx, s)
	if e != nil {
		return cero, e
	}
	if p.ValidarPara(s) != nil {
		return cero, ct.ErrConsultaPreparacionCierreAdministrativoNoDisponible
	}
	return p, nil
}

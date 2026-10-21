package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	appbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	plantillashttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpapi/plantillascatalogo"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctpostgres "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	pgvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	seg "vec-diputacion-granada/internal/vec/adapters/seguridad"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	appvec "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type selectorConfiableIdentidadDocumentalCT func(context.Context, string) (bp.SelectorAceptacionNativaCT, error)
type lectorAsignacionIdentidadDocumentalCT func(context.Context, string) (instantaneaPublicadaDesarrollo, bool, error)

type preparacionIdentidadDocumentalCT struct {
	contexto context.Context
	selector bp.SelectorAceptacionNativaCT
	actor    bp.ActorConfiablePersonaAceptacionCT
}

type marcaSesionIdentidadDocumentalCT struct {
	propietario *proveedorIdentidadDocumentalCT
	selector    bp.SelectorAceptacionNativaCT
	perfilRef   string
	version     uint64
	actor       *bp.ActorConfiablePersonaAceptacionCT
}
type claveSesionIdentidadDocumentalCT struct{}

type proveedorIdentidadDocumentalCT struct {
	soporte   *soporteAltaContratacionTemporalDesarrollo
	sesion    *proveedorSesionConsultaRRHHDesarrollo
	config    archivoIdentidadDocumentalCT
	lector    lectorAsignacionIdentidadDocumentalCT
	selector  selectorConfiableIdentidadDocumentalCT
	pdp       vecports.AutorizadorSolicitudLigadaV3
	emisor    *confianza.EmisorCapacidadesAtestacionAutorizacionV3
	material  *proveedorMaterialAltaContratacionTemporalDesarrollo
	poolBolsa *pgxpool.Pool
	reloj     ct.Reloj
}

func rutaIdentidadDocumentalCT(ruta, metodo string) bool {
	if metodo != http.MethodPost {
		return false
	}
	switch ruta {
	case httpct.RutaOriginalFirmableCT, httpct.RutaPropuestaFormalizacion,
		httpct.RutaResolucionFormalizacion, plantillashttp.RutaBorradores:
		return true
	}
	return false
}

func fronteraIdentidadDocumentalCTValida(f fronteraSeguridadComunDesarrollo) bool {
	if f.metodo != http.MethodPost || f.superficie != superficieInternaSeguridadComunDesarrollo {
		return false
	}
	switch f.ruta {
	case httpct.RutaOriginalFirmableCT:
		return f.descriptor.Clave == "ct-original-firmable-preparar" &&
			f.descriptor.ClaveCapacidad == ct.AccionConsultarDetalleRRHH &&
			f.descriptor.ClavePolitica == clavePoliticaContratacionTemporalDesarrollo
	case httpct.RutaPropuestaFormalizacion:
		return f.descriptor.Clave == "ct-formalizacion-proponer" &&
			f.descriptor.ClaveCapacidad == ctpostgres.AccionPropuestaFormalizacion &&
			f.descriptor.ClavePolitica == clavePoliticaContratacionTemporalDesarrollo
	case httpct.RutaResolucionFormalizacion:
		return f.descriptor.Clave == "ct-formalizacion-resolver" &&
			f.descriptor.ClaveCapacidad == ctpostgres.AccionResolucionFormalizacion &&
			f.descriptor.ClavePolitica == clavePoliticaContratacionTemporalDesarrollo
	case plantillashttp.RutaBorradores:
		return f.descriptor.Clave == "ct-plantillas-documental-descargar" &&
			f.descriptor.ClaveCapacidad == "contratacion_temporal.plantillas_documentos.documental_descargar" &&
			f.descriptor.ClavePolitica == "ct-plantillas-documental-v3"
	}
	return false
}

// Compone una capacidad nueva y el PDP común con fuentes ya gobernadas. No
// publica roles, asignaciones, claves ni actos al arrancar.
func nuevoProveedorIdentidadDocumentalCT(raiz *os.Root, c archivoIdentidadDocumentalCT,
	soporte *soporteAltaContratacionTemporalDesarrollo, sesion *proveedorSesionConsultaRRHHDesarrollo,
	lector lectorAsignacionIdentidadDocumentalCT, selector selectorConfiableIdentidadDocumentalCT,
	pools map[string]*pgxpool.Pool, registro *pgxpool.Pool,
	material *proveedorMaterialAltaContratacionTemporalDesarrollo, reloj ct.Reloj,
) (*proveedorIdentidadDocumentalCT, error) {
	if raiz == nil || !c.valida() || soporte == nil || sesion == nil || sesion.soporte != soporte ||
		sesion.canalIntervencion != nil ||
		lector == nil || selector == nil || registro == nil || material == nil || material.atestador == nil ||
		material.confianza == nil || reloj == nil || pools["bolsa"] == nil ||
		pools["fuente_autorizacion"] == nil || pools["motivos_autorizacion"] == nil ||
		registro.Config().ConnConfig.User == pools["bolsa"].Config().ConnConfig.User ||
		registro.Config().ConnConfig.User == pools["fuente_autorizacion"].Config().ConnConfig.User ||
		registro.Config().ConnConfig.User == pools["motivos_autorizacion"].Config().ConnConfig.User ||
		soporte.principalID != c.PrincipalID || soporte.certificadoSHA256 != c.CertificadoSHA256 ||
		sesion.base.Contexto.Instantanea.CuentaRef != c.CuentaRef || sesion.base.Contexto.Principal.ID != c.PrincipalID {
		return nil, errIdentidadDocumentalCTNoDisponible
	}
	fuente, err := pgvec.NuevoAlmacenAutorizacion(pools["fuente_autorizacion"])
	if err != nil {
		return nil, errIdentidadDocumentalCTNoDisponible
	}
	decisiones, err := pgvec.NuevoAlmacenAutorizacion(registro)
	if err != nil {
		return nil, errIdentidadDocumentalCTNoDisponible
	}
	motivos, err := pgvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(pools["motivos_autorizacion"], c.Motivo.CatalogoID)
	if err != nil {
		return nil, errIdentidadDocumentalCTNoDisponible
	}
	pdp, err := appvec.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, decisiones, decisiones, motivos,
		reloj, seg.GeneradorReferenciasCriptograficas{}, appvec.ConfiguracionServicioAutorizacion{})
	if err != nil {
		return nil, errIdentidadDocumentalCTNoDisponible
	}
	secreto, err := leerArchivoIncorporacionV2(raiz, c.Capacidad.File, 256)
	if err != nil {
		return nil, errIdentidadDocumentalCTNoDisponible
	}
	defer borrarBytes(secreto)
	h := sha256.Sum256(secreto)
	if hex.EncodeToString(h[:]) != c.Capacidad.SHA256 {
		return nil, errIdentidadDocumentalCTNoDisponible
	}
	clave, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(c.Capacidad.ClaveID,
		c.Capacidad.Version, secreto, c.Capacidad.EmisorID, bp.AudienciaIdentidadDocumentoAceptacionCT,
		confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, c.Capacidad.Desde, c.Capacidad.Hasta,
		time.Time{}, c.Capacidad.RevisionGobierno, c.Capacidad.HuellaGobierno)
	if err != nil || reloj.Ahora().Before(c.Capacidad.Desde) || !reloj.Ahora().Before(c.Capacidad.Hasta) {
		return nil, errIdentidadDocumentalCTNoDisponible
	}
	emisor, err := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, reloj)
	if err != nil {
		return nil, errIdentidadDocumentalCTNoDisponible
	}
	a := &proveedorIdentidadDocumentalCT{soporte: soporte, sesion: sesion, config: c, lector: lector,
		selector: selector, pdp: pdp, emisor: emisor, material: material, poolBolsa: pools["bolsa"], reloj: reloj}
	sesion.identidadDocumental = a
	return a, nil
}

func (a *proveedorIdentidadDocumentalCT) Preparar(ctx context.Context, expedienteRef string) (preparacionIdentidadDocumentalCT, error) {
	var cero preparacionIdentidadDocumentalCT
	if a == nil || ctx == nil || a.soporte == nil || a.sesion == nil || a.lector == nil || a.selector == nil ||
		expedienteRef == "" {
		return cero, errIdentidadDocumentalCTNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	canal, valido := a.soporte.capacidadValida(ctx)
	if !valido || !rutaIdentidadDocumentalCT(canal.ruta, canal.metodo) ||
		canal.principal.ID != a.config.PrincipalID ||
		canal.principal.Attributes["certificate_sha256"] != a.config.CertificadoSHA256 {
		return cero, ct.ErrAutorizacionDenegada
	}
	frontera, existe := fronteraSeguridadComunDesdeContexto(ctx)
	if !existe || !a.sesion.fronteras.mismaInstancia(frontera.catalogo) ||
		frontera.ruta != canal.ruta || frontera.metodo != canal.metodo ||
		!fronteraIdentidadDocumentalCTValida(frontera) {
		return cero, ct.ErrAutorizacionDenegada
	}
	selector, err := a.selector(ctx, expedienteRef)
	if err != nil || selector.ExpedienteRef != expedienteRef || selector.UnidadRef != a.config.UnidadRef {
		return cero, ct.ErrAutorizacionDenegada
	}
	publicada, existe, err := a.lector(ctx, a.config.PerfilActivoRef)
	if err != nil || !existe || !a.asignacionAdmitida(publicada) {
		return cero, ct.ErrAutorizacionDenegada
	}
	marca := &marcaSesionIdentidadDocumentalCT{propietario: a, selector: selector,
		perfilRef: a.config.PerfilActivoRef, version: uint64(publicada.instantanea.AsignacionPerfil.Version)}
	marcado := context.WithValue(ctx, claveSesionIdentidadDocumentalCT{}, marca)
	actor, err := a.sesion.ResolverContexto(marcado)
	if err != nil || actor.Resultado.Contexto.PerfilActivoRef != marca.perfilRef ||
		actor.Resultado.Contexto.Instantanea.PerfilVersion != marca.version ||
		actor.Resultado.Contexto.Instantanea.CuentaRef != a.config.CuentaRef ||
		actor.Resultado.Contexto.Principal.ID != a.config.PrincipalID ||
		actor.Vinculo.ValidarPara(actor.Resultado) != nil {
		return cero, ct.ErrAutorizacionDenegada
	}
	q := bp.SolicitudIdentidadDocumentoAceptacionCT{Selector: selector,
		ActorConfiable: bp.ActorConfiablePersonaAceptacionCT{Vinculo: actor.Vinculo, Resultado: actor.Resultado}}
	if _, err := appbolsa.PrepararIdentidadDocumentoAceptacionCT(q); err != nil {
		return cero, ct.ErrAutorizacionDenegada
	}
	marca.actor = &q.ActorConfiable
	return preparacionIdentidadDocumentalCT{contexto: marcado, selector: selector, actor: q.ActorConfiable}, nil
}

func (a *proveedorIdentidadDocumentalCT) asignacionAdmitida(p instantaneaPublicadaDesarrollo) bool {
	i := p.instantanea
	ahora := a.reloj.Ahora()
	return i.Validar() == nil && i.AsignacionPerfil.PerfilActivoRef == a.config.PerfilActivoRef &&
		i.AsignacionPerfil.PrincipalID == a.config.PrincipalID &&
		i.AsignacionPerfil.VersionRolRef == a.config.VersionRolRef &&
		p.actoAsignacion == a.config.ActoAsignacionRef &&
		i.AsignacionPerfil.Estado == core.EstadoAsignacionPerfilActiva && i.AsignacionPerfil.VigenteEn(ahora) &&
		i.VersionRol.Referencia() == a.config.VersionRolRef &&
		i.VersionRol.Estado == core.EstadoVersionRolPublicada &&
		i.VersionRol.RolID == "identidad_documental_ct_gestion_dev" &&
		len(i.VersionRol.Concesiones) == 1 && concesionIdentidadDocumentalCTValida(i.VersionRol.Concesiones[0]) &&
		i.ControlVigenciaVersionRol.Estado == core.EstadoControlVigenciaVersionRolHabilitada &&
		len(i.AsignacionPerfil.Ambitos) == 1 && i.AsignacionPerfil.Ambitos[0].Clave == "unidad_ref" &&
		len(i.AsignacionPerfil.Ambitos[0].Valores) == 1 &&
		i.AsignacionPerfil.Ambitos[0].Valores[0] == a.config.UnidadRef
}

// Solo acepta la solicitud construida con el selector CT y el actor de esta
// misma petición. El PDP vuelve a leer la asignación bajo su propio control.
func (a *proveedorIdentidadDocumentalCT) ExigirSolicitudLigadaV3(ctx context.Context,
	s core.SolicitudAutorizacionLigadaV3, resultado core.ResultadoContextoActorRegistradoV2,
) (core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	var d core.DecisionAutorizacionLigadaV3
	var c vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	marca, ok := a.marcaValida(ctx)
	if !ok || a.pdp == nil || marca.actor == nil {
		return d, c, ct.ErrAutorizacionDenegada
	}
	datos, err := s.Datos()
	if err != nil || datos.Accion != bp.AccionIdentidadDocumentoAceptacionCT ||
		datos.Finalidad != bp.FinalidadIdentidadDocumentoAceptacionCT ||
		!datos.VinculoAutenticacionActor.CoincideExactamenteCon(marca.actor.Vinculo) ||
		!bytes.Equal(resultado.RepresentacionCanonica, marca.actor.Resultado.RepresentacionCanonica) ||
		!reflect.DeepEqual(resultado.Contexto, marca.actor.Resultado.Contexto) {
		return d, c, ct.ErrAutorizacionDenegada
	}
	p, err := appbolsa.PrepararIdentidadDocumentoAceptacionCT(bp.SolicitudIdentidadDocumentoAceptacionCT{
		Selector: marca.selector, ActorConfiable: *marca.actor})
	if err != nil || !reflect.DeepEqual(datos.Recurso, p.Recurso) {
		return d, c, ct.ErrAutorizacionDenegada
	}
	return a.pdp.ExigirSolicitudLigadaV3(ctx, s, resultado)
}

func (a *proveedorIdentidadDocumentalCT) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context,
	s core.SolicitudAutorizacionLigadaV3, r core.ResultadoContextoActorRegistradoV2,
) (core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	var d core.DecisionAutorizacionLigadaV3
	var c vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	marca, ok := a.marcaValida(ctx)
	if !ok || marca.actor == nil || a.material == nil || a.emisor == nil {
		return d, c, nil, ct.ErrAutorizacionDenegada
	}
	confianzaActual, err := a.material.instantaneaConfianza(ctx)
	if err != nil {
		return d, c, nil, errors.Join(errIdentidadDocumentalCTNoDisponible, err)
	}
	emisor, err := confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(a, a.material.atestador, confianzaActual, a.emisor)
	if err != nil {
		return d, c, nil, errors.Join(errIdentidadDocumentalCTNoDisponible, err)
	}
	return emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, s, r)
}

func (a *proveedorIdentidadDocumentalCT) marcaValida(ctx context.Context) (*marcaSesionIdentidadDocumentalCT, bool) {
	if a == nil || ctx == nil || ctx.Err() != nil || a.soporte == nil || a.sesion == nil {
		return nil, false
	}
	m, ok := ctx.Value(claveSesionIdentidadDocumentalCT{}).(*marcaSesionIdentidadDocumentalCT)
	canal, valido := a.soporte.capacidadValida(ctx)
	frontera, existe := fronteraSeguridadComunDesdeContexto(ctx)
	return m, ok && m != nil && m.propietario == a && m.perfilRef == a.config.PerfilActivoRef &&
		m.selector.UnidadRef == a.config.UnidadRef && valido && rutaIdentidadDocumentalCT(canal.ruta, canal.metodo) &&
		existe && a.sesion.fronteras.mismaInstancia(frontera.catalogo) &&
		frontera.ruta == canal.ruta && frontera.metodo == canal.metodo && fronteraIdentidadDocumentalCTValida(frontera) &&
		canal.principal.ID == a.config.PrincipalID &&
		canal.principal.Attributes["certificate_sha256"] == a.config.CertificadoSHA256
}

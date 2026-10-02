package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"reflect"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// Arranque del permiso V3 de Mi Bolsa (Área personal del candidato).
//
// El arranque solo publica el permiso si el perfil no tiene ninguno, o si lo
// que hay es exactamente la consulta que publicaba el binario anterior y se
// sube al portal contra esa preimagen exacta. Con el permiso exacto no escribe
// nada. En cualquier otro caso (revocado, restringido, rol retirado o de otra
// forma) no escribe, avisa en el registro con la huella de la asignación
// vigente y deja Mi Bolsa denegada. Restaurarlo exige la aprobación expresa
// del operador ligada a esa huella exacta, y solo sobre un permiso activo, en
// vigor y puesto por este mismo circuito con los ámbitos del candidato.

// estadoPerfilMiBolsaDesarrollo resume, sin datos personales, qué ha hecho el
// arranque con el permiso de Mi Bolsa.
type estadoPerfilMiBolsaDesarrollo string

const (
	perfilMiBolsaPublicado          estadoPerfilMiBolsaDesarrollo = "publicado"
	perfilMiBolsaVigente            estadoPerfilMiBolsaDesarrollo = "vigente"
	perfilMiBolsaProvisionado       estadoPerfilMiBolsaDesarrollo = "provisionado"
	perfilMiBolsaPendienteProvision estadoPerfilMiBolsaDesarrollo = "pendiente_provision"
)

// aprobacionProvisionMiBolsaDesarrollo liga la aprobación del operador a la
// asignación vigente exacta que autoriza sustituir.
type aprobacionProvisionMiBolsaDesarrollo struct {
	referencia string
	preimagen  string
}

func aprobacionProvisionMiBolsaDesdeConfig(cfg config.Config) aprobacionProvisionMiBolsaDesarrollo {
	referencia, preimagen := cfg.BolsaProvisionMiBolsa()
	return aprobacionProvisionMiBolsaDesarrollo{referencia: referencia, preimagen: preimagen}
}

func (a aprobacionProvisionMiBolsaDesarrollo) aprueba(huella string) bool {
	return a.referencia != "" && a.preimagen != "" && huella != "" && a.preimagen == huella
}

// instantaneaVigenteMiBolsaDesarrollo es la instantánea vigente del perfil tal
// como está en PostgreSQL, con la procedencia de sus punteros actuales.
type instantaneaVigenteMiBolsaDesarrollo struct {
	instantanea    dominiovec.InstantaneaAutorizacion
	huella         string
	actoAsignacion string
	actualizadaPor string
	actoControl    string
}

// leerInstantaneaVigenteMiBolsaDesarrollo lee, en una transacción de solo
// lectura, la asignación vigente del perfil, su versión de rol, el control
// actual de esa versión y el catálogo de políticas. No escribe nada. Devuelve
// encontrada=true con error si hay puntero pero sus documentos no cuadran con
// sus huellas.
func leerInstantaneaVigenteMiBolsaDesarrollo(
	ctx context.Context, pool *pgxpool.Pool, perfilRef string,
) (instantaneaVigenteMiBolsaDesarrollo, bool, error) {
	vacia := instantaneaVigenteMiBolsaDesarrollo{}
	if ctx == nil || ctx.Err() != nil || pool == nil || perfilRef == "" {
		return vacia, false, falloPostgreSQLCTDesarrollo(nil)
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return vacia, false, falloPostgreSQLCTDesarrollo(err)
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE `+rolPropietarioAutorizacionPostgreSQLDesarrollo); err != nil {
		return vacia, false, falloPostgreSQLCTDesarrollo(err)
	}
	var documentoAsignacion, documentoRol, documentoControl []byte
	var asignacionRef, huellaRol, huellaControl, revisionCatalogo, huellaCatalogo string
	var vigente instantaneaVigenteMiBolsaDesarrollo
	err = tx.QueryRow(ctx, `
		SELECT asignacion.asignacion_ref, asignacion.documento, rol.documento, control.documento,
		       asignacion.huella_sha256, rol.huella_sha256, control.huella_sha256,
		       catalogo.revision::text, catalogo.huella_sha256,
		       puntero.acto_ref, puntero.actualizada_por, control_actual.acto_ref
		  FROM vec_autorizacion.asignacion_perfil_actual AS puntero
		  JOIN vec_autorizacion.asignacion_perfil AS asignacion
		    ON asignacion.perfil_activo_ref=puntero.perfil_activo_ref
		   AND asignacion.asignacion_ref=puntero.asignacion_ref
		  JOIN vec_autorizacion.version_rol AS rol
		    ON rol.version_rol_ref=asignacion.version_rol_ref
		  JOIN vec_autorizacion.control_vigencia_version_rol_actual AS control_actual
		    ON control_actual.version_rol_ref=rol.version_rol_ref
		  JOIN vec_autorizacion.control_vigencia_version_rol AS control
		    ON control.version_rol_ref=control_actual.version_rol_ref
		   AND control.revision=control_actual.revision
		  JOIN vec_autorizacion.control_catalogo_politicas AS catalogo ON catalogo.control_id=true
		 WHERE puntero.perfil_activo_ref=$1`, perfilRef,
	).Scan(&asignacionRef, &documentoAsignacion, &documentoRol, &documentoControl,
		&vigente.huella, &huellaRol, &huellaControl, &revisionCatalogo, &huellaCatalogo,
		&vigente.actoAsignacion, &vigente.actualizadaPor, &vigente.actoControl)
	if errors.Is(err, pgx.ErrNoRows) {
		return vacia, false, nil
	}
	if err != nil {
		return vacia, false, falloPostgreSQLCTDesarrollo(err)
	}
	// Desde aquí hay puntero: la huella leída sirve para el aviso aunque los
	// documentos no cuadren.
	fallida := instantaneaVigenteMiBolsaDesarrollo{huella: vigente.huella}
	i := &vigente.instantanea
	revision, errRevision := strconv.ParseUint(revisionCatalogo, 10, 64)
	if errRevision != nil || json.Unmarshal(documentoAsignacion, &i.AsignacionPerfil) != nil ||
		json.Unmarshal(documentoRol, &i.VersionRol) != nil ||
		json.Unmarshal(documentoControl, &i.ControlVigenciaVersionRol) != nil {
		return fallida, true, falloPostgreSQLCTDesarrollo(nil)
	}
	i.RevisionCatalogoPoliticas, i.CatalogoPoliticasHuellaSHA256 = revision, huellaCatalogo
	calculadaAsignacion, errAsignacion := i.AsignacionPerfil.HuellaSHA256()
	calculadaRol, errRol := i.VersionRol.HuellaSHA256()
	calculadaControl, errControl := i.ControlVigenciaVersionRol.HuellaSHA256()
	// Los documentos han de ser exactamente los que acreditan sus huellas.
	if errAsignacion != nil || errRol != nil || errControl != nil || i.Validar() != nil ||
		calculadaAsignacion != vigente.huella || calculadaRol != huellaRol || calculadaControl != huellaControl ||
		i.AsignacionPerfil.Referencia() != asignacionRef || i.AsignacionPerfil.PerfilActivoRef != perfilRef {
		return fallida, true, falloPostgreSQLCTDesarrollo(nil)
	}
	if err = tx.Commit(ctx); err != nil {
		return fallida, true, falloPostgreSQLCTDesarrollo(err)
	}
	return vigente, true, nil
}

// instantaneaMiBolsaEsperada es la semilla del perfil con el identificador y
// las versiones que fija la base. No cambia nada más.
func instantaneaMiBolsaEsperada(semilla, publicada dominiovec.InstantaneaAutorizacion) dominiovec.InstantaneaAutorizacion {
	esperada := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(semilla)
	esperada.VersionRol.Version = publicada.VersionRol.Version
	esperada.AsignacionPerfil.VersionRolRef = esperada.VersionRol.Referencia()
	esperada.ControlVigenciaVersionRol.VersionRolRef = esperada.VersionRol.Referencia()
	esperada.AsignacionPerfil.AsignacionID = publicada.AsignacionPerfil.AsignacionID
	esperada.AsignacionPerfil.Version = publicada.AsignacionPerfil.Version
	return esperada
}

// operativaDeEsteCircuitoMiBolsa: la asignación vigente está activa y en
// vigor, su rol habilitado, y la pusieron los actos y el emisor de este
// arranque, para el mismo principal y perfil.
func operativaDeEsteCircuitoMiBolsa(
	vigente instantaneaVigenteMiBolsaDesarrollo, semilla dominiovec.InstantaneaAutorizacion,
	autoridad *autoridadPostgreSQLDesarrollo, ahora time.Time,
) bool {
	p := vigente.instantanea
	return autoridad != nil && p.Validar() == nil && semilla.Validar() == nil &&
		vigente.actoAsignacion == autoridad.actoAsignacion && vigente.actoControl == autoridad.actoControlRol &&
		vigente.actualizadaPor == semilla.AsignacionPerfil.EmitidaPor &&
		p.AsignacionPerfil.EmitidaPor == semilla.AsignacionPerfil.EmitidaPor &&
		p.AsignacionPerfil.Estado == dominiovec.EstadoAsignacionPerfilActiva && p.AsignacionPerfil.VigenteEn(ahora) &&
		p.ControlVigenciaVersionRol.Estado == dominiovec.EstadoControlVigenciaVersionRolHabilitada &&
		p.AsignacionPerfil.PrincipalID == semilla.AsignacionPerfil.PrincipalID &&
		p.AsignacionPerfil.PerfilActivoRef == semilla.AsignacionPerfil.PerfilActivoRef &&
		p.RevisionCatalogoPoliticas == semilla.RevisionCatalogoPoliticas &&
		p.CatalogoPoliticasHuellaSHA256 == semilla.CatalogoPoliticasHuellaSHA256
}

// miBolsaVigenteExacta: lo publicado es exactamente la semilla de este
// arranque (mismo rol y concesiones, ámbitos, vigencia y control) y sigue
// operativo. Entonces no hay nada que escribir.
func miBolsaVigenteExacta(
	vigente instantaneaVigenteMiBolsaDesarrollo, semilla dominiovec.InstantaneaAutorizacion,
	autoridad *autoridadPostgreSQLDesarrollo, ahora time.Time,
) (dominiovec.InstantaneaAutorizacion, bool) {
	if !operativaDeEsteCircuitoMiBolsa(vigente, semilla, autoridad, ahora) {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	esperada := instantaneaMiBolsaEsperada(semilla, vigente.instantanea)
	if esperada.Validar() != nil || !mismasHuellasInstantaneaMiBolsa(esperada, vigente.instantanea) {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	return esperada, true
}

// restaurableMiBolsa reconoce el único estado que una aprobación puede
// sustituir: el permiso operativo de este circuito, con un rol de Mi Bolsa de
// otra forma (otras concesiones) y exactamente los ámbitos y la vigencia de la
// semilla. Nunca una revocación, una restricción ni un rol retirado.
func restaurableMiBolsa(
	vigente instantaneaVigenteMiBolsaDesarrollo, semilla dominiovec.InstantaneaAutorizacion,
	autoridad *autoridadPostgreSQLDesarrollo, ahora time.Time,
) bool {
	p := vigente.instantanea
	if !operativaDeEsteCircuitoMiBolsa(vigente, semilla, autoridad, ahora) ||
		!rolPropioMiBolsaDesarrollo(p.VersionRol.RolID) || p.VersionRol.PublicadaPor != semilla.VersionRol.PublicadaPor ||
		!p.AsignacionPerfil.VigenteDesde.Equal(semilla.AsignacionPerfil.VigenteDesde) ||
		!p.AsignacionPerfil.VigenteHasta.Equal(semilla.AsignacionPerfil.VigenteHasta) {
		return false
	}
	return reflect.DeepEqual(p.AsignacionPerfil.Ambitos, semilla.AsignacionPerfil.Ambitos)
}

func rolPropioMiBolsaDesarrollo(rolID string) bool {
	return rolID == rolConsultaMiBolsaDesarrollo || rolID == rolPortalMiBolsaDesarrollo
}

func mismasHuellasInstantaneaMiBolsa(a, b dominiovec.InstantaneaAutorizacion) bool {
	ha, errA := a.AsignacionPerfil.HuellaSHA256()
	hb, errB := b.AsignacionPerfil.HuellaSHA256()
	ra, errRA := a.VersionRol.HuellaSHA256()
	rb, errRB := b.VersionRol.HuellaSHA256()
	ca, errCA := a.ControlVigenciaVersionRol.HuellaSHA256()
	cb, errCB := b.ControlVigenciaVersionRol.HuellaSHA256()
	return errA == nil && errB == nil && errRA == nil && errRB == nil && errCA == nil && errCB == nil &&
		ha == hb && ra == rb && ca == cb &&
		a.AsignacionPerfil.Referencia() == b.AsignacionPerfil.Referencia() &&
		a.VersionRol.Referencia() == b.VersionRol.Referencia() &&
		a.RevisionCatalogoPoliticas == b.RevisionCatalogoPoliticas &&
		a.CatalogoPoliticasHuellaSHA256 == b.CatalogoPoliticasHuellaSHA256
}

// asegurarPerfilMiBolsaDesarrollo sustituye a la publicación ciega de
// arranque. Devuelve la instantánea consumible y el estado; con
// «pendiente_provision» la instantánea va vacía y Mi Bolsa se deniega. Un
// error solo indica que la base no está disponible o que la publicación
// inicial falló.
func asegurarPerfilMiBolsaDesarrollo(
	ctx context.Context, autoridad *autoridadPostgreSQLDesarrollo,
	identidad *identidadCandidatoBolsaDesarrollo, ahora time.Time, portal bool,
	aprobacion aprobacionProvisionMiBolsaDesarrollo,
) (dominiovec.InstantaneaAutorizacion, estadoPerfilMiBolsaDesarrollo, error) {
	vacia := dominiovec.InstantaneaAutorizacion{}
	if ctx == nil || autoridad == nil || !autoridad.validaConfiguracion() || identidad == nil {
		return vacia, "", errMiBolsaNoDisponible
	}
	semilla, err := nuevaInstantaneaMiBolsaDesarrollo(identidad, ahora, portal)
	if err != nil {
		return vacia, "", errMiBolsaNoDisponible
	}
	perfilRef := semilla.AsignacionPerfil.PerfilActivoRef
	vigente, encontrada, err := leerInstantaneaVigenteMiBolsaDesarrollo(ctx, autoridad.pool, perfilRef)
	if err != nil && !encontrada {
		return vacia, "", err
	}
	if !encontrada {
		// Solo alta: si otro arranque la crea entre medias, el CAS de
		// soloInicial rechaza en vez de sobrescribir.
		inicial := *autoridad
		inicial.soloInicial = true
		preparada, errAlta := publicarPerfilMiBolsaDesarrollo(ctx, &inicial, identidad, ahora, portal, false)
		if errAlta == nil {
			return preparada, perfilMiBolsaPublicado, nil
		}
		// Otro arranque pudo crearla a la vez: se vuelve a leer y se sigue
		// como con un permiso existente, sin escribir. Si sigue sin haber
		// permiso (por ejemplo, el rol compartido está retirado), el alta no
		// se hizo y el arranque se detiene como antes.
		vigente, encontrada, err = leerInstantaneaVigenteMiBolsaDesarrollo(ctx, autoridad.pool, perfilRef)
		if !encontrada {
			if err == nil {
				err = errAlta
			}
			return vacia, "", err
		}
	}
	if err == nil {
		if exacta, ok := miBolsaVigenteExacta(vigente, semilla, autoridad, ahora); ok {
			return exacta, perfilMiBolsaVigente, nil
		}
		// Consulta del binario anterior → portal, solo contra la preimagen
		// exacta de la consulta; cualquier otro estado lo rechaza el CAS.
		if portal && operativaDeEsteCircuitoMiBolsa(vigente, semilla, autoridad, ahora) &&
			vigente.instantanea.VersionRol.RolID == rolConsultaMiBolsaDesarrollo {
			if preparada, err := publicarPerfilMiBolsaDesarrollo(ctx, autoridad, identidad, ahora, portal, false); err == nil {
				return preparada, perfilMiBolsaPublicado, nil
			}
		}
		if aprobacion.aprueba(vigente.huella) && restaurableMiBolsa(vigente, semilla, autoridad, ahora) {
			return provisionarMiBolsaDesarrollo(ctx, autoridad, semilla, vigente, aprobacion)
		}
		if portal {
			anterior, err := nuevaInstantaneaMiBolsaDesarrollo(identidad, ahora, true, false)
			if err == nil {
				if exacta, ok := miBolsaVigenteExacta(vigente, anterior, autoridad, ahora); ok {
					slog.Warn("solicitud documental Mi Bolsa pendiente de provisión; consulta anterior conservada",
						"perfil_ref", perfilRef, "asignacion_vigente_huella_sha256", vigente.huella, "estado", string(perfilMiBolsaPendienteProvision))
					return exacta, perfilMiBolsaVigente, nil
				}
			}
		}
	}
	slog.Warn("Mi Bolsa sin asignación consumible: sus peticiones se deniegan hasta la provisión",
		"perfil_ref", perfilRef, "asignacion_vigente_huella_sha256", vigente.huella,
		"estado", string(perfilMiBolsaPendienteProvision))
	return vacia, perfilMiBolsaPendienteProvision, nil
}

// provisionarMiBolsaDesarrollo publica la semilla contra la preimagen exacta
// aprobada (CAS bajo bloqueo): si la asignación, su rol o su control cambian
// entre la lectura y la publicación, no se escribe nada.
func provisionarMiBolsaDesarrollo(
	ctx context.Context, autoridad *autoridadPostgreSQLDesarrollo, semilla dominiovec.InstantaneaAutorizacion,
	vigente instantaneaVigenteMiBolsaDesarrollo, aprobacion aprobacionProvisionMiBolsaDesarrollo,
) (dominiovec.InstantaneaAutorizacion, estadoPerfilMiBolsaDesarrollo, error) {
	vacia := dominiovec.InstantaneaAutorizacion{}
	preparada, err := autoridad.prepararInstantanea(ctx, semilla, false)
	if err != nil || preparada.Validar() != nil ||
		preparada.AsignacionPerfil.Version != vigente.instantanea.AsignacionPerfil.Version+1 {
		return vacia, "", errMiBolsaNoDisponible
	}
	esperada := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(semilla)
	esperada.VersionRol.Version = preparada.VersionRol.Version
	esperada.AsignacionPerfil.VersionRolRef = esperada.VersionRol.Referencia()
	esperada.ControlVigenciaVersionRol.VersionRolRef = esperada.VersionRol.Referencia()
	esperada.AsignacionPerfil.Version = preparada.AsignacionPerfil.Version
	// El identificador lo fija la base (ver publicarPerfilMiBolsaDesarrollo).
	esperada.AsignacionPerfil.AsignacionID = preparada.AsignacionPerfil.AsignacionID
	if preparada.AsignacionPerfil.AsignacionID != vigente.instantanea.AsignacionPerfil.AsignacionID ||
		!reflect.DeepEqual(preparada, esperada) ||
		autoridad.publicarInstantaneaDesdePreimagen(ctx, preparada, vigente.instantanea) != nil {
		return vacia, "", errMiBolsaNoDisponible
	}
	huella, _ := preparada.AsignacionPerfil.HuellaSHA256()
	slog.Info("provisión de Mi Bolsa aplicada",
		"perfil_ref", preparada.AsignacionPerfil.PerfilActivoRef, "aprobacion_ref", aprobacion.referencia,
		"preimagen_huella_sha256", vigente.huella,
		"version_previa", vigente.instantanea.AsignacionPerfil.Version, "version", preparada.AsignacionPerfil.Version,
		"asignacion_huella_sha256", huella, "estado", string(perfilMiBolsaProvisionado))
	return preparada, perfilMiBolsaProvisionado, nil
}

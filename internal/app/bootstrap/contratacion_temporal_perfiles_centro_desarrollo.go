package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// Perfiles V3 de las personas del centro en la composición de desarrollo.
//
// El perfil general de cada persona (el de siempre, con los ámbitos de su
// organización y su centro) ya no se prepara ni se publica por petición: las
// lecturas y las escrituras que usan esos ámbitos (contexto, bandeja,
// operaciones, incorporaciones y consulta de las opciones de cancelación)
// consumen la asignación publicada tal cual. Si no coincide exactamente con la
// del centro (revocada, restringida, retirada o de otra forma), se deniega.
//
// La cancelación del expediente necesita una asignación ligada a ese
// expediente y a su fase. Se ejecuta con un perfil propio de la misma persona
// (misma cuenta y persona; perfil, vínculo y sesión distintos) que sigue
// publicándose por petición, pero solo encima de una asignación operativa
// publicada por este mismo circuito: una revocación nunca se reactiva.
const (
	rolCancelacionCentroDesarrollo            = "cancelacion_centro_desarrollo"
	actoControlRolCancelacionCentroDesarrollo = "acto:ct:cancelacion-centro:control-rol:v1"
	actoAsignacionCancelacionCentroDesarrollo = "acto:ct:cancelacion-centro:asignacion:v1"
	actoSesionCancelacionCentroDesarrollo     = "acto:ct:cancelacion-centro:sesion:v1"
	// Actos con los que la composición de desarrollo ha publicado siempre el
	// perfil general del centro (autoridadComun de Contratación temporal).
	actoControlRolCTDesarrollo = "acto:ct:desarrollo:control-rol:v1"
	actoAsignacionCTDesarrollo = "acto:ct:desarrollo:asignacion:v1"
)

var errPerfilCentroNoConsumible = errors.New("perfil del centro sin asignación publicada consumible")

// instantaneaPublicadaDesarrollo es la instantánea vigente de un perfil tal
// como está en PostgreSQL, con la procedencia de sus punteros actuales.
type instantaneaPublicadaDesarrollo struct {
	instantanea    dominiovec.InstantaneaAutorizacion
	actoAsignacion string
	actualizadaPor string
	actoControl    string
}

// leerInstantaneaPublicadaPostgreSQLDesarrollo lee, en una transacción de solo
// lectura y sin bloqueos, la asignación vigente del perfil, su versión de rol,
// el control actual de esa versión y el catálogo de políticas. No escribe
// nada: el registro V3 vuelve a comprobarlo todo bajo bloqueo al conceder.
func leerInstantaneaPublicadaPostgreSQLDesarrollo(
	ctx context.Context, pool *pgxpool.Pool, perfilRef string,
) (instantaneaPublicadaDesarrollo, bool, error) {
	vacia := instantaneaPublicadaDesarrollo{}
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
	var asignacionRef, huellaAsignacion, huellaRol, huellaControl, revisionCatalogo, huellaCatalogo string
	var publicada instantaneaPublicadaDesarrollo
	err = tx.QueryRow(ctx, `
		SELECT asignacion.asignacion_ref, asignacion.documento, rol.documento, control.documento,
		       asignacion.huella_sha256, rol.huella_sha256, control.huella_sha256,
		       catalogo.revision::text, catalogo.huella_sha256,
		       vigente.acto_ref, vigente.actualizada_por, control_actual.acto_ref
		  FROM vec_autorizacion.asignacion_perfil_actual AS vigente
		  JOIN vec_autorizacion.asignacion_perfil AS asignacion
		    ON asignacion.perfil_activo_ref=vigente.perfil_activo_ref
		   AND asignacion.asignacion_ref=vigente.asignacion_ref
		  JOIN vec_autorizacion.version_rol AS rol
		    ON rol.version_rol_ref=asignacion.version_rol_ref
		  JOIN vec_autorizacion.control_vigencia_version_rol_actual AS control_actual
		    ON control_actual.version_rol_ref=rol.version_rol_ref
		  JOIN vec_autorizacion.control_vigencia_version_rol AS control
		    ON control.version_rol_ref=control_actual.version_rol_ref
		   AND control.revision=control_actual.revision
		  JOIN vec_autorizacion.control_catalogo_politicas AS catalogo ON catalogo.control_id=true
		 WHERE vigente.perfil_activo_ref=$1`, perfilRef,
	).Scan(&asignacionRef, &documentoAsignacion, &documentoRol, &documentoControl,
		&huellaAsignacion, &huellaRol, &huellaControl, &revisionCatalogo, &huellaCatalogo,
		&publicada.actoAsignacion, &publicada.actualizadaPor, &publicada.actoControl)
	if errors.Is(err, pgx.ErrNoRows) {
		return vacia, false, nil
	}
	if err != nil {
		return vacia, false, falloPostgreSQLCTDesarrollo(err)
	}
	i := &publicada.instantanea
	revision, errRevision := strconv.ParseUint(revisionCatalogo, 10, 64)
	if errRevision != nil || json.Unmarshal(documentoAsignacion, &i.AsignacionPerfil) != nil ||
		json.Unmarshal(documentoRol, &i.VersionRol) != nil ||
		json.Unmarshal(documentoControl, &i.ControlVigenciaVersionRol) != nil {
		return vacia, true, falloPostgreSQLCTDesarrollo(nil)
	}
	i.RevisionCatalogoPoliticas, i.CatalogoPoliticasHuellaSHA256 = revision, huellaCatalogo
	calculadaAsignacion, errAsignacion := i.AsignacionPerfil.HuellaSHA256()
	calculadaRol, errRol := i.VersionRol.HuellaSHA256()
	calculadaControl, errControl := i.ControlVigenciaVersionRol.HuellaSHA256()
	// Los documentos han de ser exactamente los que acreditan sus huellas.
	if errAsignacion != nil || errRol != nil || errControl != nil || i.Validar() != nil ||
		calculadaAsignacion != huellaAsignacion || calculadaRol != huellaRol || calculadaControl != huellaControl ||
		i.AsignacionPerfil.Referencia() != asignacionRef || i.AsignacionPerfil.PerfilActivoRef != perfilRef {
		return vacia, true, falloPostgreSQLCTDesarrollo(nil)
	}
	if err = tx.Commit(ctx); err != nil {
		return vacia, true, falloPostgreSQLCTDesarrollo(err)
	}
	return publicada, true, nil
}

// instantaneaEsperadaDesdePlantilla es la plantilla del perfil con el
// identificador y las versiones que fija la base. No cambia nada más.
func instantaneaEsperadaDesdePlantilla(
	plantilla dominiovec.InstantaneaAutorizacion, publicada dominiovec.InstantaneaAutorizacion,
) dominiovec.InstantaneaAutorizacion {
	esperada := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(plantilla)
	esperada.VersionRol.Version = publicada.VersionRol.Version
	esperada.AsignacionPerfil.VersionRolRef = esperada.VersionRol.Referencia()
	esperada.ControlVigenciaVersionRol.VersionRolRef = esperada.VersionRol.Referencia()
	esperada.AsignacionPerfil.AsignacionID = publicada.AsignacionPerfil.AsignacionID
	esperada.AsignacionPerfil.Version = publicada.AsignacionPerfil.Version
	return esperada
}

// instantaneaConsumible devuelve la instantánea que se entrega al PDP si, y
// solo si, la publicada es exactamente la plantilla del perfil (mismo rol y
// concesiones, mismos ámbitos, mismo control) y sigue operativa en ahora.
func instantaneaConsumible(
	publicada instantaneaPublicadaDesarrollo, plantilla dominiovec.InstantaneaAutorizacion, ahora time.Time,
) (dominiovec.InstantaneaAutorizacion, bool) {
	p := publicada.instantanea
	if plantilla.Validar() != nil || p.Validar() != nil ||
		p.AsignacionPerfil.Estado != dominiovec.EstadoAsignacionPerfilActiva || !p.AsignacionPerfil.VigenteEn(ahora) ||
		p.ControlVigenciaVersionRol.Estado != dominiovec.EstadoControlVigenciaVersionRolHabilitada ||
		p.RevisionCatalogoPoliticas != plantilla.RevisionCatalogoPoliticas ||
		p.CatalogoPoliticasHuellaSHA256 != plantilla.CatalogoPoliticasHuellaSHA256 {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	esperada := instantaneaEsperadaDesdePlantilla(plantilla, p)
	if esperada.Validar() != nil || !mismasHuellasInstantaneaDesarrollo(esperada, p) {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	return esperada, true
}

func mismasHuellasInstantaneaDesarrollo(a, b dominiovec.InstantaneaAutorizacion) bool {
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

// consumidorInstantaneaPublicadaDesarrollo lo implementa la autoridad
// PostgreSQL; un soporte que solo consume nunca prepara ni publica.
type consumidorInstantaneaPublicadaDesarrollo interface {
	consumirInstantaneaPublicada(context.Context, dominiovec.InstantaneaAutorizacion) (dominiovec.InstantaneaAutorizacion, error)
}

func (a *autoridadPostgreSQLContratacionTemporalDesarrollo) consumirInstantaneaPublicada(
	ctx context.Context, plantilla dominiovec.InstantaneaAutorizacion,
) (dominiovec.InstantaneaAutorizacion, error) {
	if a == nil || a.soporte == nil || dependenciaEsNulaContratacionTemporalDesarrollo(a.soporte.reloj) {
		return dominiovec.InstantaneaAutorizacion{}, errPerfilCentroNoConsumible
	}
	publicada, encontrada, err := leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, a.pool, plantilla.AsignacionPerfil.PerfilActivoRef)
	if err != nil {
		return dominiovec.InstantaneaAutorizacion{}, err
	}
	if !encontrada {
		return dominiovec.InstantaneaAutorizacion{}, errPerfilCentroNoConsumible
	}
	consumible, ok := instantaneaConsumible(publicada, plantilla, a.soporte.reloj.Ahora())
	if !ok {
		return dominiovec.InstantaneaAutorizacion{}, errPerfilCentroNoConsumible
	}
	return consumible, nil
}

// instantaneaConsumidaPublicada es la fuente de los soportes que solo
// consumen: la plantilla fija del perfil, contrastada con la publicada.
func (s *soporteAltaContratacionTemporalDesarrollo) instantaneaConsumidaPublicada(
	ctx context.Context,
) (dominiovec.InstantaneaAutorizacion, bool) {
	s.mu.Lock()
	plantilla := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantanea)
	consumidor, ok := s.autoridadAsignaciones.(consumidorInstantaneaPublicadaDesarrollo)
	s.mu.Unlock()
	if !ok || dependenciaEsNulaContratacionTemporalDesarrollo(consumidor) || plantilla.Validar() != nil {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	consumida, err := consumidor.consumirInstantaneaPublicada(ctx, plantilla)
	if err == nil {
		err = consumida.Validar()
	}
	if err != nil {
		// Se deniega. El registro permite distinguir una asignación que
		// espera provisión de una base no disponible, sin datos personales.
		causa := "asignacion_no_consumible"
		if !errors.Is(err, errPerfilCentroNoConsumible) {
			causa = causaFalloPostgreSQLCTDesarrollo(err)
		}
		// Como mucho un aviso por minuto y perfil: la denegación es constante
		// mientras el perfil espera la provisión.
		ahora := time.Now()
		s.mu.Lock()
		avisar := s.avisoNoConsumibleEn.IsZero() || ahora.Sub(s.avisoNoConsumibleEn) >= time.Minute
		if avisar {
			s.avisoNoConsumibleEn = ahora
		}
		s.mu.Unlock()
		if avisar {
			slog.Warn("lectura del centro denegada: asignación publicada no consumible",
				"perfil_ref", plantilla.AsignacionPerfil.PerfilActivoRef, "causa", causa)
		}
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	return consumida, true
}

// aprobacionProvisionCentroDesarrollo liga la aprobación del operador a las
// asignaciones vigentes exactas que autoriza sustituir.
type aprobacionProvisionCentroDesarrollo struct {
	referencia  string
	preimagenes map[string]bool
}

func (a aprobacionProvisionCentroDesarrollo) valida() bool {
	return a.referencia != "" && len(a.preimagenes) != 0
}

// perfilCentroOperativo: el arranque dejó el perfil general consumible.
func (e estadoPerfilCentroDesarrollo) perfilCentroOperativo() bool {
	return e == perfilCentroPublicadoInicial || e == perfilCentroVigente || e == perfilCentroProvisionado
}

// estadoPerfilCentroDesarrollo resume, sin datos personales, qué ha hecho el
// arranque con el perfil general de una persona del centro.
type estadoPerfilCentroDesarrollo string

const (
	perfilCentroPublicadoInicial   estadoPerfilCentroDesarrollo = "publicado_inicial"
	perfilCentroVigente            estadoPerfilCentroDesarrollo = "vigente"
	perfilCentroProvisionado       estadoPerfilCentroDesarrollo = "provisionado"
	perfilCentroPendienteProvision estadoPerfilCentroDesarrollo = "pendiente_provision"
)

// asegurarPerfilCentroConsumibleDesarrollo sustituye a la publicación de
// arranque del perfil general del centro. Solo publica si el perfil no tiene
// asignación (publicación inicial transaccional). Con una asignación exacta no
// escribe nada. En cualquier otro caso tampoco escribe, salvo que el operador
// haya aprobado expresamente la provisión de esa asignación exacta (referencia
// de aprobación y huella de la asignación vigente, que el aviso muestra) y
// sea el estrechamiento por petición que dejaba el binario anterior:
// activa, en vigor, con su rol habilitado, publicada por este mismo circuito,
// del mismo rol y con los ámbitos de la plantilla. Entonces se publica la
// plantilla contra esa preimagen exacta (CAS bajo bloqueo). Una asignación
// revocada, restringida o retirada nunca se toca: las lecturas se deniegan.
func asegurarPerfilCentroConsumibleDesarrollo(
	ctx context.Context, pool *pgxpool.Pool, soporte *soporteAltaContratacionTemporalDesarrollo, aprobacion aprobacionProvisionCentroDesarrollo,
) (estadoPerfilCentroDesarrollo, error) {
	if ctx == nil || pool == nil || soporte == nil || dependenciaEsNulaContratacionTemporalDesarrollo(soporte.reloj) || soporte.instantanea.Validar() != nil {
		return "", falloPostgreSQLCTDesarrollo(nil)
	}
	plantilla := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(soporte.instantanea)
	autoridad := &autoridadPostgreSQLContratacionTemporalDesarrollo{pool: pool, soporte: soporte}
	publicada, encontrada, err := leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, pool, plantilla.AsignacionPerfil.PerfilActivoRef)
	if err != nil && !encontrada {
		return "", err
	}
	if !encontrada {
		comun := autoridad.autoridadComun()
		comun.soloInicial = true
		preparada, err := comun.prepararInstantanea(ctx, plantilla, true)
		if err != nil || preparada.AsignacionPerfil.Version != 1 {
			return "", falloPostgreSQLCTDesarrollo(err)
		}
		if err := comun.publicarInstantanea(ctx, preparada); err != nil {
			return "", err
		}
		return perfilCentroPublicadoInicial, nil
	}
	ahora := soporte.reloj.Ahora()
	if err == nil {
		if _, exacta := instantaneaConsumible(publicada, plantilla, ahora); exacta {
			return perfilCentroVigente, nil
		}
	}
	huellaVigente, errHuella := publicada.instantanea.AsignacionPerfil.HuellaSHA256()
	if err != nil || errHuella != nil || !aprobacion.valida() || !aprobacion.preimagenes[huellaVigente] ||
		!preimagenAutoproducidaCentroDesarrollo(publicada, plantilla, ahora) {
		slog.Warn("perfil del centro sin asignación consumible: sus peticiones se deniegan hasta la provisión",
			"perfil_ref", plantilla.AsignacionPerfil.PerfilActivoRef, "asignacion_vigente_huella_sha256", huellaVigente,
			"estado", string(perfilCentroPendienteProvision))
		return perfilCentroPendienteProvision, nil
	}
	comun := autoridad.autoridadComun()
	preparada, err := comun.prepararInstantanea(ctx, plantilla, false)
	if err != nil || preparada.AsignacionPerfil.Version != publicada.instantanea.AsignacionPerfil.Version+1 {
		return "", falloPostgreSQLCTDesarrollo(err)
	}
	if err := comun.publicarInstantaneaDesdePreimagen(ctx, preparada, publicada.instantanea); err != nil {
		return "", err
	}
	huella, _ := preparada.AsignacionPerfil.HuellaSHA256()
	slog.Info("provisión del perfil del centro aplicada",
		"perfil_ref", preparada.AsignacionPerfil.PerfilActivoRef, "aprobacion_ref", aprobacion.referencia,
		"preimagen_huella_sha256", huellaVigente,
		"version_previa", publicada.instantanea.AsignacionPerfil.Version, "version", preparada.AsignacionPerfil.Version,
		"asignacion_huella_sha256", huella, "estado", string(perfilCentroProvisionado))
	return perfilCentroProvisionado, nil
}

// preimagenAutoproducidaCentroDesarrollo reconoce el estrechamiento por
// petición que publicaba el binario anterior sobre el perfil general: nunca
// una revocación, una restricción de otro acto ni un rol retirado.
func preimagenAutoproducidaCentroDesarrollo(
	publicada instantaneaPublicadaDesarrollo, plantilla dominiovec.InstantaneaAutorizacion, ahora time.Time,
) bool {
	p := publicada.instantanea
	if p.Validar() != nil || plantilla.Validar() != nil ||
		publicada.actoAsignacion != actoAsignacionCTDesarrollo || publicada.actoControl != actoControlRolCTDesarrollo ||
		publicada.actualizadaPor != p.AsignacionPerfil.EmitidaPor ||
		p.AsignacionPerfil.EmitidaPor != plantilla.AsignacionPerfil.EmitidaPor ||
		p.AsignacionPerfil.Estado != dominiovec.EstadoAsignacionPerfilActiva || !p.AsignacionPerfil.VigenteEn(ahora) ||
		p.ControlVigenciaVersionRol.Estado != dominiovec.EstadoControlVigenciaVersionRolHabilitada ||
		p.AsignacionPerfil.PrincipalID != plantilla.AsignacionPerfil.PrincipalID ||
		p.AsignacionPerfil.PerfilActivoRef != plantilla.AsignacionPerfil.PerfilActivoRef ||
		p.VersionRol.RolID != plantilla.VersionRol.RolID ||
		p.VersionRol.PublicadaPor != plantilla.VersionRol.PublicadaPor ||
		p.RevisionCatalogoPoliticas != plantilla.RevisionCatalogoPoliticas ||
		p.CatalogoPoliticasHuellaSHA256 != plantilla.CatalogoPoliticasHuellaSHA256 {
		return false
	}
	// Estrechamiento: conserva cada ámbito de la plantilla con los mismos
	// valores (organización y centro) y como mucho añade otros.
	actuales := make(map[string][]string, len(p.AsignacionPerfil.Ambitos))
	for _, a := range p.AsignacionPerfil.Ambitos {
		actuales[a.Clave] = a.Valores
	}
	for _, a := range plantilla.AsignacionPerfil.Ambitos {
		valores, ok := actuales[a.Clave]
		if !ok || len(valores) != len(a.Valores) {
			return false
		}
		for i := range valores {
			if valores[i] != a.Valores[i] {
				return false
			}
		}
	}
	return true
}

// asegurarPerfilCancelacionCentroDesarrollo publica la asignación inicial del
// perfil de cancelación solo si no existe. Si existe no la toca: las
// cancelaciones posteriores la estrechan por petición con la guarda de origen
// operativo, y una revocación la deja cerrada.
func asegurarPerfilCancelacionCentroDesarrollo(
	ctx context.Context, pool *pgxpool.Pool, soporte *soporteAltaContratacionTemporalDesarrollo,
) error {
	if ctx == nil || pool == nil || soporte == nil || !soporte.perfilCancelacionCentro || soporte.instantanea.Validar() != nil {
		return falloPostgreSQLCTDesarrollo(nil)
	}
	plantilla := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(soporte.instantanea)
	_, encontrada, err := leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, pool, plantilla.AsignacionPerfil.PerfilActivoRef)
	if encontrada {
		return nil
	}
	if err != nil {
		return err
	}
	comun := (&autoridadPostgreSQLContratacionTemporalDesarrollo{pool: pool, soporte: soporte}).autoridadComun()
	comun.soloInicial = true
	preparada, err := comun.prepararInstantanea(ctx, plantilla, true)
	if err != nil || preparada.AsignacionPerfil.Version != 1 {
		return falloPostgreSQLCTDesarrollo(err)
	}
	return comun.publicarInstantanea(ctx, preparada)
}

// discriminadorContextoCancelacionCentroDesarrollo separa el perfil de
// cancelación del perfil general de la misma persona del centro.
func discriminadorContextoCancelacionCentroDesarrollo() discriminadorContextoSinteticoDesarrollo {
	return discriminadorContextoSinteticoDesarrollo{
		perfil: "cancelacion-centro-perfil", vinculo: "cancelacion-centro-vinculo",
		// La cuenta, la persona y la procedencia son las del perfil general.
		procedencia: "procedencia", registro: "cancelacion-centro-registro-contexto",
		autenticacion: "cancelacion-centro-autenticacion", asercion: "cancelacion-centro-asercion",
		sesion: "cancelacion-centro-sesion", controlSesion: "cancelacion-centro-control-sesion",
		politicaGarantia: "cancelacion-centro-politica-garantia",
	}
}

// nuevoPerfilCancelacionCentroDesarrollo compone el perfil propio con el que
// una persona del centro cancela: misma cuenta, persona y certificado que su
// perfil general; perfil, vínculo y sesión distintos; un rol con la única
// concesión de cancelar y los ámbitos iniciales de su organización y centro.
func nuevoPerfilCancelacionCentroDesarrollo(
	general *soporteAltaContratacionTemporalDesarrollo, principal dominiovec.Principal,
	actor domain.ActorPeticionCentro, concesiones []dominiovec.ConcesionRol, reloj relojContratacionTemporalDesarrollo,
) (*perfilCentroDesarrollo, error) {
	if general == nil || general.sello == nil || len(concesiones) != 1 ||
		concesiones[0].Accion != string(domain.AccionCancelarExpediente) {
		return nil, ports.ErrPeticionCentroNoDisponible
	}
	ahora := reloj.Ahora()
	contexto, err := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(
		principal, ahora, discriminadorContextoCancelacionCentroDesarrollo())
	if err != nil {
		return nil, err
	}
	v, err := contexto.Vinculo.Datos()
	base := general.contexto.Resultado.Contexto
	if err != nil || v.PerfilActivoRef == base.PerfilActivoRef || v.PrincipalID != actor.ActorRef ||
		contexto.Resultado.Contexto.Instantanea.CuentaRef != base.Instantanea.CuentaRef ||
		contexto.Resultado.Contexto.PersonaRef != base.PersonaRef {
		return nil, ports.ErrPeticionCentroNoDisponible
	}
	s := &soporteAltaContratacionTemporalDesarrollo{sello: general.sello, principalID: general.principalID,
		certificadoSHA256: general.certificadoSHA256, contexto: contexto, reloj: reloj,
		peticionesCentro: true, perfilCancelacionCentro: true, motivo: motivoPeticionCentroDesarrollo(),
		registroDecisionesAnalisis: general.registroDecisionesAnalisis,
		instantaneasPorSolicitud:   make(map[string]dominiovec.InstantaneaAutorizacion), concesiones: make(map[string]struct{})}
	if autoridad, ok := general.autoridadAsignaciones.(*autoridadPostgreSQLContratacionTemporalDesarrollo); ok && autoridad != nil {
		s.autoridadAsignaciones = &autoridadPostgreSQLContratacionTemporalDesarrollo{pool: autoridad.pool, soporte: s}
	} else {
		return nil, ports.ErrPeticionCentroNoDisponible
	}
	s.instantanea, err = nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef, ahora,
		rolCancelacionCentroDesarrollo, "Cancelación de expedientes por el centro de desarrollo",
		"cancelacion-centro-desarrollo-"+principal.ID, concesiones,
		[]dominiovec.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
			{Clave: "centro_ref", Valores: []string{actor.CentroRef}}})
	if err != nil {
		return nil, err
	}
	autorizador, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(s, s, s, s, reloj,
		seguridadvec.GeneradorReferenciasCriptograficas{}, aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second})
	if err != nil {
		return nil, err
	}
	actorCancelacion := actor
	actorCancelacion.PerfilRef = v.PerfilActivoRef
	if actorCancelacion.Validar() != nil {
		return nil, domain.ErrPeticionCentroInvalida
	}
	return &perfilCentroDesarrollo{soporte: s, autorizador: autorizador, actor: actorCancelacion}, nil
}

// publicarContextoCancelacionCentroDesarrollo registra el contexto del perfil
// de cancelación con una operación propia, que incluye el perfil para no
// colisionar con el contexto general de la misma persona.
func publicarContextoCancelacionCentroDesarrollo(
	ctx context.Context, pool *pgxpool.Pool, soporte *soporteAltaContratacionTemporalDesarrollo,
) error {
	if soporte == nil || !soporte.perfilCancelacionCentro {
		return falloPostgreSQLCTDesarrollo(nil)
	}
	v, err := soporte.contexto.Vinculo.Datos()
	if err != nil {
		return falloPostgreSQLCTDesarrollo(err)
	}
	operacion := referenciaAltaContratacionTemporalDesarrollo("oca_",
		v.PrincipalID+"\x00"+v.PerfilActivoRef+"\x00registro-contexto-cancelacion-centro")
	return publicarResultadoContextoPostgreSQLDesarrollo(ctx, pool, soporte.contexto.Resultado, operacion)
}

package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	postgres "vec-diputacion-granada/internal/vec/adapters/postgres"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	app "vec-diputacion-granada/internal/vec/application"
	vec "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ConfiguracionRPTUsosFixtureV3 llega de la composición privada del ensayo.
// No contiene DSN ni claves. Las pruebas de READY habilitan comprobar el clon;
// el operador conserva fuera de Git los documentos que acreditan esas referencias.
type ConfiguracionRPTUsosFixtureV3 struct {
	Activar                                    bool
	Host                                       string
	Puerto                                     uint16
	Base                                       string
	ClonHuellaSHA256                           string
	ReadyClonHuellaSHA256                      string
	ReadyH6Ref, ReadyAD132Ref, ReadySeisRPTRef string
	AprobacionRef, PreimagenAsignacionSHA256   string
	Descriptor                                 ports.DescriptorCatalogoRPT
	Motivo                                     vec.ReferenciaEntradaCatalogo
}

// RPTUsosFixtureV3 conserva una sola composición nominal del ejercicio.
// No se registra en vec-server ni añade rutas HTTP. El contexto usado por el
// ejercicio es sintético; no acredita autenticación mTLS de una persona.
type RPTUsosFixtureV3 struct {
	pools      [3]*pgxpool.Pool
	cerrar     sync.Once
	gestor     *postgres.GestorUsosCategoriaRPTPostgreSQL
	emisor     *confianza.EmisorMaterialAutorizacionAtestadaV3
	contexto   ctports.ContextoAutorizacionAltaV3
	descriptor ports.DescriptorCatalogoRPT
	motivo     vec.ReferenciaEntradaCatalogo
}

func (f *RPTUsosFixtureV3) Cerrar() {
	if f == nil {
		return
	}
	f.cerrar.Do(func() {
		for _, p := range f.pools {
			if p != nil {
				p.Close()
			}
		}
	})
}

func (f *RPTUsosFixtureV3) Preparador() ports.PreparadorUsosCategoriaRPT {
	if f == nil {
		return nil
	}
	return f.gestor
}
func (f *RPTUsosFixtureV3) Gestor() ports.GestorUsosCategoriaRPT {
	if f == nil {
		return nil
	}
	return f.gestor
}

// NuevoRPTUsosFixtureV3 sólo se invoca desde el canal privado autorizado del
// clon. El validador de motivos es un puerto nominal ya compuesto: su pool es
// el de la autoridad existente de motivos, nunca el pool de gobierno RPT.
func NuevoRPTUsosFixtureV3(ctx context.Context, cfg config.Config, c ConfiguracionRPTUsosFixtureV3,
	validadorMotivos ports.ValidadorReferenciaMotivoAutorizacionV2) (_ *RPTUsosFixtureV3, errFinal error) {
	if ctx == nil || ctx.Err() != nil || dependenciaAutorizacionComunDesarrolloNula(validadorMotivos) ||
		!vec.ReferenciaMotivoAutorizacionV2Valida(c.Motivo) {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	dsnE, dsnG, err := cfg.ContratacionTemporalPostgreSQL.DSNSeparados()
	if err != nil {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	dsnR, err := cfg.ContratacionTemporalPostgreSQL.DSNRegistroAutorizacionSeparado()
	if err != nil {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	dsns := [3]string{dsnG, dsnR, dsnE}
	var configuraciones [3]*pgxpool.Config
	for i, dsn := range dsns {
		configuraciones[i], err = pgxpool.ParseConfig(dsn)
		if err != nil {
			return nil, ErrSeguridadComunDesarrolloDenegada
		}
	}
	admision := admisionRPTUsosFixture{activado: c.Activar, host: c.Host, puerto: c.Puerto, base: c.Base,
		huellaClonEsperada: c.ClonHuellaSHA256, huellaClonReady: c.ReadyClonHuellaSHA256,
		readyH6: c.ReadyH6Ref, readyAD132: c.ReadyAD132Ref, readySeisRPT: c.ReadySeisRPTRef,
		aprobacionRef: c.AprobacionRef, preimagenAsignacionHuellaSHA256: c.PreimagenAsignacionSHA256}
	if err := admision.validar(cfg, configuraciones); err != nil {
		return nil, err
	}
	reloj := relojContratacionTemporalDesarrollo{}
	if err := validadorMotivos.ValidarReferenciaMotivoAutorizacionV2(ctx, c.Motivo, reloj.Ahora()); err != nil {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	// La composición existente comprueba CA, certificado y asociación nominal
	// privados, y entrega el derivador gobernado. No genera ni copia material.
	seguridad, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	defer seguridad.derivadorIdempotencia.borrar()
	contexto, plantilla, err := contextoYPlantillaRPTUsosFixture(seguridad, c.Descriptor, reloj.Ahora())
	if err != nil {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	f := &RPTUsosFixtureV3{contexto: contexto, descriptor: c.Descriptor, motivo: c.Motivo}
	defer func() {
		if errFinal != nil {
			f.Cerrar()
		}
	}()
	roles := [3]string{rolGobiernoPostgreSQLContratacionTemporalDesarrollo, rolRegistroAutorizacionPostgreSQLContratacionTemporalDesarrollo, rolEjecucionPostgreSQLContratacionTemporalDesarrollo}
	usuarios := make(map[string]bool, len(roles))
	for i, rol := range roles {
		var usuario string
		f.pools[i], usuario, err = abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, dsns[i], "vec-rpt-usos-fixture", rol)
		if err != nil || usuario != configuraciones[i].ConnConfig.User || usuarios[usuario] {
			return nil, ErrSeguridadComunDesarrolloDenegada
		}
		usuarios[usuario] = true
	}
	if err := comprobarDependenciasRPTUsosFixture(ctx, f.pools[2]); err != nil {
		return nil, err
	}
	f.gestor, err = postgres.NuevoGestorUsosCategoriaRPTPostgreSQL(f.pools[2], c.Descriptor, "contratacion_temporal")
	if err != nil {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	publicada, encontrada, err := leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, f.pools[0], plantilla.AsignacionPerfil.PerfilActivoRef)
	if err != nil || admitirPreimagenRPTUsosFixture(publicada, encontrada, plantilla, c.PreimagenAsignacionSHA256, reloj.Ahora()) != nil {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	base, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(seguridad.derivadorIdempotencia, reloj.Ahora())
	if err != nil {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	defer base.borrarCopiasEfimeras()
	material, err := derivarMaterialConsumidorV3Desarrollo(base, descriptorMaterialRPTUsosFixture())
	if err != nil {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	defer borrarBytes(material.claveHMAC)
	if err := publicarGobiernoAtestacionContratacionTemporalDesarrollo(ctx, f.pools[0], &material); err != nil {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	a := autoridadPostgreSQLDesarrollo{pool: f.pools[0], vinculo: contexto.Vinculo, prefijoBloqueo: "vec:rpt:usos:fixture:autorizacion:",
		actoControlRol: actoControlRPTUsosFixture, actoAsignacion: actoAsignacionRPTUsosFixture, actoSesion: actoSesionRPTUsosFixture, exigirOrigenOperativo: true}
	if err := publicarResultadoContextoPostgreSQLDesarrollo(ctx, f.pools[0], contexto.Resultado,
		referenciaAltaContratacionTemporalDesarrollo("oca_", contexto.Resultado.RegistroContextoRef+"\x00rpt-usos-fixture")); err != nil {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	if err := provisionarPerfilRPTUsosFixture(ctx, a, plantilla, c.PreimagenAsignacionSHA256, reloj.Ahora()); err != nil {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	slog.Info("rpt_usos_fixture_provision_consumible", "aprobacion_ref", c.AprobacionRef,
		"preimagen_sha256", c.PreimagenAsignacionSHA256)
	registro, err := postgres.NuevoAlmacenAutorizacion(f.pools[1])
	if err != nil {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	servicio, err := app.NuevoServicioAutorizacionSolicitudLigadaV3(&fuentePerfilRPTUsosFixture{pool: f.pools[0], plantilla: plantilla, reloj: reloj},
		registro, registro, validadorMotivos, reloj, generadorRPTUsosFixture{}, app.ConfiguracionServicioAutorizacion{})
	if err != nil {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	p, err := nuevoProveedorMaterialAutorizacionBaseDesarrollo(material, reloj)
	if err != nil {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	f.emisor, err = confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(servicio, p.atestador, p.confianza, p.emisor)
	if err != nil {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	return f, nil
}

func (f *RPTUsosFixtureV3) Emitir(ctx context.Context, p ports.PreparacionAutorizacionUsoCategoriaRPT) (vec.SolicitudAutorizacionLigadaV3, ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var solicitud vec.SolicitudAutorizacionLigadaV3
	var material ports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if f == nil || f.emisor == nil || ctx == nil || ctx.Err() != nil ||
		p.AudienciaConsumo != audienciaUsosCategoriasRPTFixture || p.Finalidad != finalidadRPTUsosFixture ||
		(p.Accion != accionReservaRPTUsosFixture && p.Accion != accionConfirmacionRPTUsosFixture && p.Accion != accionCancelacionRPTUsosFixture) ||
		p.Recurso.Validar() != nil || p.Recurso.ModuloID != f.descriptor.ModuloID || p.Recurso.Tipo != "uso_categoria" ||
		len(p.Recurso.Ambitos) != 3 || p.Recurso.Ambitos["catalogo_id"] != f.descriptor.CatalogoID ||
		p.Recurso.Ambitos["modulo_id"] != f.descriptor.ModuloID || p.Recurso.Ambitos["consumidor"] != "contratacion_temporal" ||
		len(p.Recurso.Atributos) != 1 || !huellaRPTUsosFixture.MatchString(p.Recurso.Atributos["material_sha256"]) {
		return solicitud, material, ErrSeguridadComunDesarrolloDenegada
	}
	correlacion, err := vec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, generadorRPTUsosFixture{})
	if err != nil {
		return solicitud, material, ErrSeguridadComunDesarrolloDenegada
	}
	solicitud, err = vec.NuevaSolicitudAutorizacionLigadaV3(vec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: f.contexto.Vinculo, ReferenciaMotivo: f.motivo, Accion: p.Accion, Recurso: p.Recurso, Finalidad: p.Finalidad, Correlacion: correlacion})
	if err != nil {
		return vec.SolicitudAutorizacionLigadaV3{}, material, ErrSeguridadComunDesarrolloDenegada
	}
	_, _, exportador, err := f.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, f.contexto.Resultado)
	if err != nil || dependenciaAutorizacionComunDesarrolloNula(exportador) {
		return vec.SolicitudAutorizacionLigadaV3{}, material, ErrSeguridadComunDesarrolloDenegada
	}
	material, err = exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		return vec.SolicitudAutorizacionLigadaV3{}, ports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ErrSeguridadComunDesarrolloDenegada
	}
	return solicitud, material, nil
}

type generadorRPTUsosFixture struct{}

func (generadorRPTUsosFixture) NuevaReferenciaDecisionAutorizacion() (string, error) {
	return referenciaAleatoriaRPTUsosFixture("decision:rpt:fixture:")
}
func (generadorRPTUsosFixture) NuevaReferenciaCorrelacionAutorizacionV2(ctx context.Context) (string, error) {
	if ctx == nil || ctx.Err() != nil {
		return "", ErrSeguridadComunDesarrolloDenegada
	}
	return referenciaAleatoriaRPTUsosFixture("correlacion_")
}
func referenciaAleatoriaRPTUsosFixture(prefijo string) (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", ErrSeguridadComunDesarrolloDenegada
	}
	return prefijo + hex.EncodeToString(b[:]), nil
}

// PlanPerfilRPTUsosFixtureV3 contiene sólo compromisos de la plantilla inicial
// que el operador puede aprobar. No es una autorización ni publica nada.
type PlanPerfilRPTUsosFixtureV3 struct {
	PrincipalRef           string
	PerfilRef              string
	AsignacionHuellaSHA256 string
}

func PlanificarPerfilRPTUsosFixtureV3(cfg config.Config, descriptor ports.DescriptorCatalogoRPT) (PlanPerfilRPTUsosFixtureV3, error) {
	if !cfg.DevelopmentEnabledByDoubleKey() {
		return PlanPerfilRPTUsosFixtureV3{}, ErrSeguridadComunDesarrolloDenegada
	}
	seguridad, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		return PlanPerfilRPTUsosFixtureV3{}, ErrSeguridadComunDesarrolloDenegada
	}
	defer seguridad.derivadorIdempotencia.borrar()
	contexto, plantilla, err := contextoYPlantillaRPTUsosFixture(seguridad, descriptor, relojContratacionTemporalDesarrollo{}.Ahora())
	if err != nil {
		return PlanPerfilRPTUsosFixtureV3{}, ErrSeguridadComunDesarrolloDenegada
	}
	huella, err := plantilla.AsignacionPerfil.HuellaSHA256()
	if err != nil {
		return PlanPerfilRPTUsosFixtureV3{}, ErrSeguridadComunDesarrolloDenegada
	}
	return PlanPerfilRPTUsosFixtureV3{PrincipalRef: contexto.Resultado.Contexto.Principal.ID,
		PerfilRef: contexto.Resultado.Contexto.PerfilActivoRef, AsignacionHuellaSHA256: huella}, nil
}

func contextoYPlantillaRPTUsosFixture(seguridad *ComposicionSeguridadDesarrollo, descriptor ports.DescriptorCatalogoRPT, ahora time.Time) (ctports.ContextoAutorizacionAltaV3, vec.InstantaneaAutorizacion, error) {
	if seguridad == nil {
		return ctports.ContextoAutorizacionAltaV3{}, vec.InstantaneaAutorizacion{}, ErrSeguridadComunDesarrolloDenegada
	}
	identidades, ok := seguridad.identidad.(*resolvedorIdentidadDesarrollo)
	if !ok {
		return ctports.ContextoAutorizacionAltaV3{}, vec.InstantaneaAutorizacion{}, ErrSeguridadComunDesarrolloDenegada
	}
	principal, ok := identidades.principalConRolUnico(rolTecnicoRRHHContratacionTemporalDesarrollo)
	if !ok {
		return ctports.ContextoAutorizacionAltaV3{}, vec.InstantaneaAutorizacion{}, ErrSeguridadComunDesarrolloDenegada
	}
	d := discriminadorContextoSinteticoDesarrollo{perfil: "rpt-usos-fixture-perfil", vinculo: "rpt-usos-fixture-vinculo",
		// Cuenta y persona conservan su procedencia común; perfil, vínculo,
		// registro y sesión son propios de RPT, como los perfiles fijos existentes.
		procedencia: "procedencia", registro: "rpt-usos-fixture-registro", autenticacion: "rpt-usos-fixture-autenticacion",
		asercion: "rpt-usos-fixture-asercion", sesion: "rpt-usos-fixture-sesion", controlSesion: "rpt-usos-fixture-control-sesion", politicaGarantia: "rpt-usos-fixture-politica-garantia"}
	contexto, err := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(principal, ahora, d)
	if err != nil {
		return ctports.ContextoAutorizacionAltaV3{}, vec.InstantaneaAutorizacion{}, ErrSeguridadComunDesarrolloDenegada
	}
	plantilla, err := plantillaRPTUsosFixture(contexto.Resultado.Contexto.Principal.ID, contexto.Resultado.Contexto.PerfilActivoRef, descriptor, ahora)
	return contexto, plantilla, err
}

// Preflight de sólo lectura: nunca crea LOGIN, pertenencias, permisos ni SQL.
func comprobarDependenciasRPTUsosFixture(ctx context.Context, pool *pgxpool.Pool) error {
	var valida bool
	err := pool.QueryRow(ctx, `SELECT
 (SELECT count(*) FROM pg_catalog.pg_auth_members WHERE member=session_user::regrole)=1
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles r ON r.oid=m.roleid
 WHERE m.member=session_user::regrole AND r.rolname='vec_contratacion_temporal_ejecutor'
 AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member='vec_contratacion_temporal_ejecutor'::regrole)
 AND COALESCE((SELECT bool_and(pg_catalog.to_regprocedure(f) IS NOT NULL
 AND pg_catalog.has_function_privilege(session_user,pg_catalog.to_regprocedure(f),'EXECUTE'))
 FROM unnest($1::text[]) f),false)`, funcionesRPTUsosFixture()).Scan(&valida)
	if err != nil || !valida {
		return ErrSeguridadComunDesarrolloDenegada
	}
	return nil
}

func funcionesRPTUsosFixture() []string {
	var r []string
	for _, nombre := range []string{"listar_categorias_habilitadas_rpt_v3_atestada", "leer_publicacion_categoria_rpt_v3_atestada",
		"consultar_uso_categoria_rpt_v3_atestada", "reservar_uso_categoria_rpt_v3_atestada", "confirmar_uso_categoria_rpt_v3_atestada", "cancelar_uso_categoria_rpt_v3_atestada"} {
		r = append(r, "vec_autorizacion_atestada_v3."+nombre+"(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)")
	}
	return r
}

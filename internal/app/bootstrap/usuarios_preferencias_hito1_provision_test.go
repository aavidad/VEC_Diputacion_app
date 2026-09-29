package bootstrap

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	usuariosports "vec-diputacion-granada/internal/modules/usuarios/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

// Este harness sólo se ejecuta en el clon desechable HITO1, nunca en la
// puerta ordinaria. La etiqueta maestra es sintética y no acredita identidad
// real. El guion privado fija los DSN y el hash de este archivo revisado.
const (
	marcaProvisionHito1    = "HITO1_PREF_508A_APLICAR_REVISADO"
	instanteProvisionHito1 = "2026-09-29T04:00:00Z"
	identificadorHito1     = "7690574644148150322"
)

type cuentaProvisionPreferenciasHito1 struct {
	configuracion configuracionUsuariosPreferenciasDesarrollo
	cuenta        cuentaUsuariosPreferenciasDesarrollo
	personaRef    string
	contexto      core.ResultadoContextoActorRegistradoV2
	vinculo       core.VinculoAutenticacionActorV2
	operacionRef  string
	instantanea   core.InstantaneaAutorizacion
}

func referenciaProvisionPreferenciasHito1(prefijo, material string) string {
	return referenciaAltaContratacionTemporalDesarrollo(prefijo, "hito1-usuarios-preferencias-508a\x00"+material)
}

func huellaProvisionPreferenciasHito1(material string) string {
	suma := sha256.Sum256([]byte("hito1-usuarios-preferencias-508a\x00" + material))
	return hex.EncodeToString(suma[:])
}

func construirCuentaProvisionPreferenciasHito1(c configuracionUsuariosPreferenciasDesarrollo, ahora time.Time) (cuentaProvisionPreferenciasHito1, error) {
	vacia := cuentaProvisionPreferenciasHito1{}
	if len(c.Cuentas) != 1 || (c.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 && c.Superficie != core.SuperficieAutenticacionExternaPersonalV1) {
		return vacia, errComposicionUsuariosPreferencias
	}
	cuentaConfig := c.Cuentas[0]
	base := string(c.Superficie) + "\x00" + cuentaConfig.CuentaRef + "\x00" + cuentaConfig.PerfilRef + "\x00" + cuentaConfig.CertificadoSHA256
	personaRef := referenciaProvisionPreferenciasHito1("per_", base+"\x00persona")
	vinculoRef := referenciaProvisionPreferenciasHito1("vca_", base+"\x00vinculo")
	procedencia := core.AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef:          referenciaProvisionPreferenciasHito1("prc_", base+"\x00procedencia"),
		ProcedenciaVersion:      1,
		ProcedenciaHuellaSHA256: huellaProvisionPreferenciasHito1(base + "\x00procedencia"),
		ProcedenciaAutoridad:    core.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	desde, hasta := ahora, ahora.Add(14*24*time.Hour)
	resueltoEn := ahora.Add(3 * time.Minute)
	cuenta := core.CuentaAutenticadaContextoActor{CuentaRef: cuentaConfig.CuentaRef, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh}
	instantanea := core.InstantaneaContextoActor{
		VinculoRef: vinculoRef, VinculoVersion: 1,
		CuentaRef: cuentaConfig.CuentaRef, CuentaVersion: 1,
		PersonaRef: personaRef, PersonaVersion: 1,
		PerfilActivoRef: cuentaConfig.PerfilRef, PerfilVersion: 1,
		Estado:       core.EstadoVinculoContextoActorActivo,
		VigenteDesde: desde, VigenteHasta: hasta,
		Vinculos: []core.VinculoReferenciaContextoActor{},
	}
	actor, err := core.NuevoContextoActor(cuenta, instantanea, resueltoEn)
	if err != nil {
		return vacia, errComposicionUsuariosPreferencias
	}
	manifiesto := core.ManifiestoProcedenciaContextoActorV1{
		Esquema:           core.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: core.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta:            core.ProcedenciaCuentaContextoActorV1{CuentaRef: cuentaConfig.CuentaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: procedencia},
		Persona:           core.ProcedenciaPersonaContextoActorV1{PersonaRef: personaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: procedencia},
		Perfil:            core.ProcedenciaPerfilContextoActorV1{PerfilRef: cuentaConfig.PerfilRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: procedencia},
		Contexto:          core.ProcedenciaVinculoContextoActorV1{VinculoRef: vinculoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: procedencia},
		Vinculos:          []core.ProcedenciaVinculoReferenciaContextoActorV1{},
	}
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		return vacia, errComposicionUsuariosPreferencias
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		return vacia, errComposicionUsuariosPreferencias
	}
	manifiestoCanon, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		return vacia, errComposicionUsuariosPreferencias
	}
	manifiestoHuella, err := core.HuellaSHA256ManifiestoProcedenciaContextoActorV1(manifiestoCanon)
	if err != nil {
		return vacia, errComposicionUsuariosPreferencias
	}
	resultado := core.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: referenciaProvisionPreferenciasHito1("rca_", base+"\x00registro-contexto"),
		Contexto:            actor, RepresentacionCanonica: canon, HuellaSHA256: huella,
		ManifiestoProcedenciaCanonico: manifiestoCanon, ManifiestoProcedenciaHuellaSHA256: manifiestoHuella,
		AutoridadEfectiva:      core.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo: resueltoEn,
	}
	autenticacion := core.AutenticacionRevalidadaV1{
		AutenticacionRef:          referenciaProvisionPreferenciasHito1("aut_", base+"\x00autenticacion"),
		AutenticacionHuellaSHA256: huellaProvisionPreferenciasHito1(base + "\x00autenticacion"),
		AsercionRef:               referenciaProvisionPreferenciasHito1("ase_", base+"\x00asercion"),
		SesionRef:                 referenciaProvisionPreferenciasHito1("ses_", base+"\x00sesion"),
		ControlSesionRef:          referenciaProvisionPreferenciasHito1("cse_", base+"\x00control-sesion"),
		ControlSesionRevision:     1,
		ControlSesionHuellaSHA256: huellaProvisionPreferenciasHito1(base + "\x00control-sesion"),
		CuentaRef:                 cuentaConfig.CuentaRef, CuentaOrdinariaRef: cuentaConfig.CuentaRef,
		Superficie: c.Superficie, MetodoObservado: core.AuthMethodCertificate, GarantiaObservada: core.AuthAssuranceHigh,
		PoliticaGarantiaRef:          referenciaProvisionPreferenciasHito1("pga_", base+"\x00politica-garantia"),
		PoliticaGarantiaHuellaSHA256: huellaProvisionPreferenciasHito1(base + "\x00politica-garantia"),
		AutenticacionVerificadaEn:    desde,
		SesionEmitidaEn:              desde.Add(time.Minute),
		SesionRevalidadaEn:           desde.Add(2 * time.Minute),
		SesionValidaHasta:            desde.Add(24 * time.Hour),
	}
	vinculo, resultadoClonado, err := core.CrearVinculoAutenticacionActorV2ConResultado(
		context.Background(),
		revalidadorAutenticacionAltaContratacionTemporalDesarrollo{valor: autenticacion},
		core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef},
		resolutorContextoAltaContratacionTemporalDesarrollo{valor: resultado},
		core.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: cuentaConfig.PerfilRef},
		relojRutasDietas{},
	)
	if err != nil || resultadoClonado.Validar() != nil || vinculo.ValidarPara(resultadoClonado) != nil {
		return vacia, errComposicionUsuariosPreferencias
	}
	return cuentaProvisionPreferenciasHito1{configuracion: c, cuenta: cuentaConfig, personaRef: personaRef,
		contexto: resultadoClonado, vinculo: vinculo,
		operacionRef: referenciaProvisionPreferenciasHito1("oca_", base+"\x00publicar-contexto")}, nil
}

func instantaneaProvisionPreferenciasHito1(c cuentaProvisionPreferenciasHito1, ahora time.Time, revision uint64, huellaCatalogo string) (core.InstantaneaAutorizacion, error) {
	vacia := core.InstantaneaAutorizacion{}
	rolID := "usuarios_preferencias_hito1_interna"
	if c.configuracion.Superficie == core.SuperficieAutenticacionExternaPersonalV1 {
		rolID = "usuarios_preferencias_hito1_externa"
	}
	rol := core.VersionRol{
		RolID: rolID, Version: 1, Nombre: "Preferencias HITO1 sintéticas", Estado: core.EstadoVersionRolPublicada,
		Concesiones: []core.ConcesionRol{
			{Accion: usuariosports.AccionConsultarPreferencias, ModuloID: "usuarios", TipoRecurso: usuariosports.TipoRecursoPreferencias,
				Finalidades: []string{usuariosports.FinalidadPreferenciasPropias}, GarantiaMinima: core.AuthAssuranceHigh,
				CamposPermitidos: []string{"catalogo", "valores", "version"}},
			{Accion: usuariosports.AccionActualizarPreferencias, ModuloID: "usuarios", TipoRecurso: usuariosports.TipoRecursoPreferencias,
				Finalidades: []string{usuariosports.FinalidadPreferenciasPropias}, GarantiaMinima: core.AuthAssuranceHigh,
				CamposPermitidos: []string{"valores", "version"}},
		},
		PublicadaPor: "seguridad:hito1:sintetica", PublicadaEn: ahora.Add(4 * time.Minute),
	}
	control := core.ControlVigenciaVersionRol{
		VersionRolRef: rol.Referencia(), Revision: 1, Estado: core.EstadoControlVigenciaVersionRolHabilitada,
		ActualizadoPor: "seguridad:hito1:sintetica", ActualizadoEn: ahora.Add(5 * time.Minute),
	}
	asignacion := core.AsignacionPerfil{
		AsignacionID: rolID, Version: 1, PerfilActivoRef: c.cuenta.PerfilRef, PrincipalID: c.personaRef,
		VersionRolRef: rol.Referencia(), Estado: core.EstadoAsignacionPerfilActiva,
		Ambitos:      []core.AmbitoPerfil{{Clave: "persona_ref", Valores: []string{c.personaRef}}},
		VigenteDesde: ahora.Add(6 * time.Minute), VigenteHasta: ahora.Add(14 * 24 * time.Hour),
		EmitidaPor: "identidad:hito1:sintetica", EmitidaEn: ahora.Add(6 * time.Minute),
	}
	i := core.InstantaneaAutorizacion{AsignacionPerfil: asignacion, VersionRol: rol, ControlVigenciaVersionRol: control,
		RevisionCatalogoPoliticas: revision, CatalogoPoliticasHuellaSHA256: huellaCatalogo}
	if i.Validar() != nil {
		return vacia, errComposicionUsuariosPreferencias
	}
	return i, nil
}

func contarContextoProvisionPreferenciasHito1(ctx context.Context, pool *pgxpool.Pool, c cuentaProvisionPreferenciasHito1) (int, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	if err != nil {
		return 0, errComposicionUsuariosPreferencias
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE vec_contexto_actor_v1_propietario`); err != nil {
		return 0, errComposicionUsuariosPreferencias
	}
	manifiesto, err := core.RehidratarManifiestoProcedenciaContextoActorV1(c.contexto.ManifiestoProcedenciaCanonico)
	if err != nil {
		return 0, errComposicionUsuariosPreferencias
	}
	prc := manifiesto.Cuenta.ProcedenciaRef
	vca := c.contexto.Contexto.Instantanea.VinculoRef
	const consulta = `SELECT
 (SELECT count(*) FROM vec_contexto_actor_v1.procedencias WHERE procedencia_ref=$1),
 (SELECT count(*) FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones WHERE cuenta_ref=$2),
 (SELECT count(*) FROM vec_contexto_actor_v1.proyeccion_cuenta_actual WHERE cuenta_ref=$2),
 (SELECT count(*) FROM vec_contexto_actor_v1.persona_versiones WHERE persona_ref=$3),
 (SELECT count(*) FROM vec_contexto_actor_v1.persona_actual WHERE persona_ref=$3),
 (SELECT count(*) FROM vec_contexto_actor_v1.perfil_versiones WHERE perfil_ref=$4),
 (SELECT count(*) FROM vec_contexto_actor_v1.perfil_actual WHERE perfil_ref=$4),
 (SELECT count(*) FROM vec_contexto_actor_v1.vinculo_contexto_versiones WHERE vinculo_ref=$5),
 (SELECT count(*) FROM vec_contexto_actor_v1.vinculo_contexto_actual WHERE vinculo_ref=$5),
 (SELECT count(*) FROM vec_contexto_actor_v1.registros_contexto WHERE operacion_ref=$6)`
	var n [10]int
	err = tx.QueryRow(ctx, consulta, prc, c.cuenta.CuentaRef, c.personaRef, c.cuenta.PerfilRef, vca, c.operacionRef).Scan(
		&n[0], &n[1], &n[2], &n[3], &n[4], &n[5], &n[6], &n[7], &n[8], &n[9])
	if err != nil {
		return 0, errComposicionUsuariosPreferencias
	}
	suma := 0
	for _, valor := range n {
		if valor > 1 {
			return 0, errComposicionUsuariosPreferencias
		}
		suma += valor
	}
	return suma, nil
}

func contarAutorizacionProvisionPreferenciasHito1(ctx context.Context, pool *pgxpool.Pool, c cuentaProvisionPreferenciasHito1) (int, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	if err != nil {
		return 0, errComposicionUsuariosPreferencias
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE vec_autorizacion_propietario`); err != nil {
		return 0, errComposicionUsuariosPreferencias
	}
	datos, err := c.vinculo.Datos()
	if err != nil {
		return 0, errComposicionUsuariosPreferencias
	}
	rol := c.instantanea.VersionRol.Referencia()
	asignacion := c.instantanea.AsignacionPerfil.Referencia()
	const consulta = `SELECT
 (SELECT count(*) FROM vec_autorizacion.version_rol WHERE version_rol_ref=$1),
 (SELECT count(*) FROM vec_autorizacion.control_vigencia_version_rol WHERE version_rol_ref=$1),
 (SELECT count(*) FROM vec_autorizacion.control_vigencia_version_rol_actual WHERE version_rol_ref=$1),
 (SELECT count(*) FROM vec_autorizacion.asignacion_perfil WHERE asignacion_ref=$2),
 (SELECT count(*) FROM vec_autorizacion.asignacion_perfil_actual WHERE perfil_activo_ref=$3),
 (SELECT count(*) FROM vec_autorizacion.sesion_autenticacion_v1 WHERE sesion_ref=$4),
 (SELECT count(*) FROM vec_autorizacion.control_sesion_v1 WHERE control_sesion_ref=$5),
 (SELECT count(*) FROM vec_autorizacion.control_sesion_actual_v1 WHERE sesion_ref=$4)`
	var n [8]int
	err = tx.QueryRow(ctx, consulta, rol, asignacion, c.cuenta.PerfilRef, datos.SesionRef, datos.ControlSesionRef).Scan(
		&n[0], &n[1], &n[2], &n[3], &n[4], &n[5], &n[6], &n[7])
	if err != nil {
		return 0, errComposicionUsuariosPreferencias
	}
	suma := 0
	for _, valor := range n {
		if valor > 1 {
			return 0, errComposicionUsuariosPreferencias
		}
		suma += valor
	}
	return suma, nil
}

func catalogoPoliticasProvisionPreferenciasHito1(ctx context.Context, pool *pgxpool.Pool) (uint64, string, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	if err != nil {
		return 0, "", errComposicionUsuariosPreferencias
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE vec_autorizacion_propietario`); err != nil {
		return 0, "", errComposicionUsuariosPreferencias
	}
	var revision uint64
	var huella string
	var numero int
	err = tx.QueryRow(ctx, `SELECT revision,huella_sha256,
 (SELECT count(*) FROM vec_autorizacion.politica_restrictiva_actual)
 FROM vec_autorizacion.control_catalogo_politicas WHERE control_id=true`).Scan(&revision, &huella, &numero)
	if err != nil || revision != 1 || numero != 0 {
		return 0, "", errComposicionUsuariosPreferencias
	}
	esperada, err := core.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil || huella != esperada {
		return 0, "", errComposicionUsuariosPreferencias
	}
	return revision, huella, nil
}

func dsnProvisionPreferenciasHito1(dsn string) (*pgxpool.Config, error) {
	c, err := pgxpool.ParseConfig(dsn)
	if err != nil || c == nil || c.ConnConfig == nil || c.ConnConfig.Host != "127.0.0.1" ||
		c.ConnConfig.Port != 55441 || c.ConnConfig.Database != "postgres" ||
		len(c.ConnConfig.Fallbacks) != 0 || validarTLSPostgreSQLBorradores(&c.ConnConfig.Config, false) != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	return c, nil
}

func leerCertificadoProvisionPreferenciasHito1(ruta string) (*x509.Certificate, error) {
	contenido, err := leerFicheroMaterialSeguro(ruta, 64<<10)
	if err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	defer borrarBytes(contenido)
	certificado, err := decodificarCertificadoUnico(contenido)
	if err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	return certificado, nil
}

func directorioProvisionPreferenciasHito1(ruta string) bool {
	if ruta == "" || !filepath.IsAbs(ruta) || filepath.Clean(ruta) != ruta || validarArbolMaterialDesarrollo(ruta) != nil {
		return false
	}
	resuelta, err := filepath.EvalSymlinks(ruta)
	if err != nil || resuelta != ruta {
		return false
	}
	info, err := os.Lstat(ruta)
	if err != nil || info.Mode().Perm() != 0o700 {
		return false
	}
	datos, ok := info.Sys().(*syscall.Stat_t)
	return ok && datos.Uid == uint32(os.Getuid())
}

func configuracionesProvisionPreferenciasHito1(materialDir string) ([]configuracionUsuariosPreferenciasDesarrollo, error) {
	cfg := config.Config{DevelopmentMaterialDir: materialDir}
	interna, err := leerConfiguracionUsuariosPreferenciasDesarrollo(cfg, core.SuperficieAutenticacionInternaCorporativaV1)
	if err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	externa, err := leerConfiguracionUsuariosPreferenciasDesarrollo(cfg, core.SuperficieAutenticacionExternaPersonalV1)
	if err != nil || !configuracionesPreferenciasSeparadas(interna, externa) || len(interna.Cuentas) != 1 || len(externa.Cuentas) != 1 {
		return nil, errComposicionUsuariosPreferencias
	}
	cliente, err := leerCertificadoProvisionPreferenciasHito1(filepath.Join(materialDir, "mtls", "cliente.crt"))
	if err != nil {
		return nil, err
	}
	intervencion, err := leerCertificadoProvisionPreferenciasHito1(filepath.Join(materialDir, "mtls", "intervencion.crt"))
	if err != nil {
		return nil, err
	}
	identidadInterna, err := cargarIdentidadDesarrollo(filepath.Join(materialDir, "identidad", "identidad.json"), cliente, "tecnico_rrhh")
	if err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	identidadExterna, err := cargarIdentidadDesarrollo(filepath.Join(materialDir, "identidad", "intervencion.json"), intervencion, "intervencion")
	if err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	resolvedor, err := nuevoResolvedorIdentidadDesarrollo(identidadInterna, identidadExterna)
	if err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	if _, err = cuentasPreferenciasAcreditadas(resolvedor, interna); err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	if _, err = cuentasPreferenciasAcreditadas(resolvedor, externa); err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	return []configuracionUsuariosPreferenciasDesarrollo{interna, externa}, nil
}

func verificarClavesPreferenciasHito1(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	if err != nil {
		return errComposicionUsuariosPreferencias
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario`); err != nil {
		return errComposicionUsuariosPreferencias
	}
	var total, distintas int
	err = tx.QueryRow(ctx, `SELECT count(*),count(DISTINCT audiencia_consumo)
 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
 WHERE audiencia_consumo LIKE 'vec_usuarios.preferencias.%'`).Scan(&total, &distintas)
	if err != nil || total != 4 || distintas != 4 {
		return errComposicionUsuariosPreferencias
	}
	for _, d := range descriptoresMaterialPreferenciasUsuariosDesarrollo() {
		var n int
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM vec_autorizacion_atestada_v3.clave_capacidad_version
 WHERE audiencia_consumo=$1 AND valida_desde<=clock_timestamp() AND valida_hasta>clock_timestamp()`, d.Audiencia).Scan(&n); err != nil || n != 1 {
			return errComposicionUsuariosPreferencias
		}
	}
	return nil
}

func estadoProvisionPreferenciasHito1(ctx context.Context, pool *pgxpool.Pool, cuentas []cuentaProvisionPreferenciasHito1) ([][2]int, error) {
	estados := make([][2]int, len(cuentas))
	for i, cuenta := range cuentas {
		contextos, err := contarContextoProvisionPreferenciasHito1(ctx, pool, cuenta)
		if err != nil {
			return nil, errComposicionUsuariosPreferencias
		}
		autorizaciones, err := contarAutorizacionProvisionPreferenciasHito1(ctx, pool, cuenta)
		if err != nil || (contextos != 0 && contextos != 10) || (autorizaciones != 0 && autorizaciones != 8) || (contextos == 0 && autorizaciones != 0) {
			return nil, errComposicionUsuariosPreferencias
		}
		estados[i] = [2]int{contextos, autorizaciones}
	}
	return estados, nil
}

func verificarDestinoProvisionPreferenciasHito1(ctx context.Context, admin, gobierno *pgxpool.Pool, identificadorEsperado string) error {
	if admin == nil || gobierno == nil || identificadorEsperado == "" {
		return errComposicionUsuariosPreferencias
	}
	var version int
	var base, id, usuarioAdmin string
	var superusuario bool
	err := admin.QueryRow(ctx, `SELECT current_setting('server_version_num')::int,current_database()::text,
 (SELECT system_identifier::text FROM pg_catalog.pg_control_system()),session_user::text,
 (SELECT rolsuper FROM pg_catalog.pg_roles WHERE rolname=session_user)`).Scan(&version, &base, &id, &usuarioAdmin, &superusuario)
	if err != nil || version/10000 != 18 || base != "postgres" || id != identificadorEsperado || !superusuario {
		return errComposicionUsuariosPreferencias
	}
	usuarioGobierno, err := comprobarIdentidadPostgreSQLContratacionTemporalDesarrollo(ctx, gobierno, rolGobiernoPostgreSQLContratacionTemporalDesarrollo)
	if err != nil || usuarioGobierno == usuarioAdmin {
		return errComposicionUsuariosPreferencias
	}
	adminTopologia, err := acreditarTopologiaPostgreSQLPreferenciasUsuarios(ctx, admin)
	if err != nil || cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, gobierno, adminTopologia) != nil {
		return errComposicionUsuariosPreferencias
	}
	var puedeContexto, puedeAutorizacion, puedeAD3 bool
	err = gobierno.QueryRow(ctx, `SELECT pg_has_role(session_user,'vec_contexto_actor_v1_propietario','SET'),
 pg_has_role(session_user,'vec_autorizacion_propietario','SET'),
 pg_has_role(session_user,'vec_autorizacion_atestada_v3_propietario','SET')`).Scan(&puedeContexto, &puedeAutorizacion, &puedeAD3)
	if err != nil || !puedeContexto || !puedeAutorizacion || !puedeAD3 {
		return errComposicionUsuariosPreferencias
	}
	return nil
}

func verificarLoginsUsuariosProvisionPreferenciasHito1(ctx context.Context, gobierno *pgxpool.Pool, configuraciones []configuracionUsuariosPreferenciasDesarrollo) error {
	if len(configuraciones) != 2 || gobierno == nil {
		return errComposicionUsuariosPreferencias
	}
	topologia, err := acreditarTopologiaPostgreSQLPreferenciasUsuarios(ctx, gobierno)
	if err != nil {
		return errComposicionUsuariosPreferencias
	}
	var gobiernoLogin string
	if err = gobierno.QueryRow(ctx, `SELECT session_user::text`).Scan(&gobiernoLogin); err != nil {
		return errComposicionUsuariosPreferencias
	}
	logins := map[string]bool{gobiernoLogin: true}
	for _, c := range configuraciones {
		for _, entrada := range []struct{ dsn, rol string }{
			{c.DSNUsuarios, rolEjecutorPreferencias(string(c.Superficie))},
			{c.DSNUsuariosFrontera, rolRegistradorPreferencias(string(c.Superficie))},
		} {
			if _, err = dsnProvisionPreferenciasHito1(entrada.dsn); err != nil {
				return errComposicionUsuariosPreferencias
			}
			pool, login, e := abrirPoolUsuariosPreferencias(ctx, entrada.dsn, entrada.rol)
			if e != nil {
				return errComposicionUsuariosPreferencias
			}
			coincide := cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, pool, topologia) == nil
			pool.Close()
			if !coincide || login == "" || logins[login] {
				return errComposicionUsuariosPreferencias
			}
			logins[login] = true
		}
	}
	return nil
}

func TestMaterialProvisionPreferenciasHito1SinPermisosAjenos(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond).Add(-10 * time.Minute)
	huellaCatalogo, err := core.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, superficie := range []core.SuperficieAutenticacionActorV1{
		core.SuperficieAutenticacionInternaCorporativaV1, core.SuperficieAutenticacionExternaPersonalV1,
	} {
		c := configuracionUsuariosPreferenciasDesarrollo{Superficie: superficie,
			Cuentas: []cuentaUsuariosPreferenciasDesarrollo{{cuentaRutasDietasDesarrollo: cuentaRutasDietasDesarrollo{
				CuentaRef: "cta_0123456789abcdefghijklmnop", PerfilRef: "prf_0123456789abcdefghijklmnop",
				CertificadoSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			}}},
		}
		cuenta, err := construirCuentaProvisionPreferenciasHito1(c, ahora)
		if err != nil {
			t.Fatalf("Contexto sintético %s: %v", superficie, err)
		}
		i, err := instantaneaProvisionPreferenciasHito1(cuenta, ahora, 1, huellaCatalogo)
		if err != nil || i.Validar() != nil || len(i.VersionRol.Concesiones) != 2 || len(i.AsignacionPerfil.Ambitos) != 1 ||
			i.AsignacionPerfil.Ambitos[0].Clave != "persona_ref" || i.AsignacionPerfil.Ambitos[0].Valores[0] != cuenta.personaRef {
			t.Fatalf("concesión sintética %s inválida: %v", superficie, err)
		}
		for j, accion := range []string{usuariosports.AccionConsultarPreferencias, usuariosports.AccionActualizarPreferencias} {
			concesion := i.VersionRol.Concesiones[j]
			if concesion.Accion != accion || concesion.ModuloID != "usuarios" || concesion.TipoRecurso != usuariosports.TipoRecursoPreferencias ||
				len(concesion.Finalidades) != 1 || concesion.Finalidades[0] != usuariosports.FinalidadPreferenciasPropias || len(concesion.Obligaciones) != 0 {
				t.Fatalf("concesión %s ajena", accion)
			}
		}
	}
}

func TestDirectorioProvisionPreferenciasHito1ExigePrivacidad(t *testing.T) {
	directorio := t.TempDir()
	if err := os.Chmod(directorio, 0o700); err != nil {
		t.Fatal(err)
	}
	if !directorioProvisionPreferenciasHito1(directorio) || directorioProvisionPreferenciasHito1("relativo") {
		t.Fatal("directorio privado válido o ruta relativa mal clasificados")
	}
	enlace := filepath.Join(t.TempDir(), "enlace")
	if err := os.Symlink(directorio, enlace); err != nil {
		t.Fatal(err)
	}
	if directorioProvisionPreferenciasHito1(enlace) {
		t.Fatal("enlace a material privado aceptado")
	}
	if err := os.Chmod(directorio, 0o755); err != nil {
		t.Fatal(err)
	}
	if directorioProvisionPreferenciasHito1(directorio) {
		t.Fatal("directorio legible por terceros aceptado")
	}
}

// TestProvisionarPreferenciasHito1 no forma parte del producto. Requiere el
// marcador deliberado del guion privado revisado; sin él siempre se omite.
func TestProvisionarPreferenciasHito1(t *testing.T) {
	if os.Getenv("VEC_PREF_HITO1_PROVISIONAR") != marcaProvisionHito1 {
		t.Skip("provisión privada HITO1 deshabilitada")
	}
	ahora, err := time.Parse(time.RFC3339, instanteProvisionHito1)
	if err != nil || time.Now().UTC().Before(ahora.Add(7*time.Minute)) || !time.Now().UTC().Before(ahora.Add(24*time.Hour)) {
		t.Fatal("ventana sintética HITO1 caducada o inválida")
	}
	materialDir := os.Getenv("VEC_PREF_HITO1_MATERIAL_DIR")
	if !directorioProvisionPreferenciasHito1(materialDir) {
		t.Fatal("directorio privado HITO1 no acreditado")
	}
	configuraciones, err := configuracionesProvisionPreferenciasHito1(materialDir)
	if err != nil {
		t.Fatal("identidades sintéticas HITO1 no acreditadas")
	}
	adminDSN := os.Getenv("VEC_PREF_HITO1_ADMIN_DSN")
	gobiernoDSN := os.Getenv("VEC_PREF_HITO1_GOBIERNO_DSN")
	adminConfig, err := dsnProvisionPreferenciasHito1(adminDSN)
	if err != nil {
		t.Fatal("DSN DBA fuera del clon TLS exacto")
	}
	if _, err = dsnProvisionPreferenciasHito1(gobiernoDSN); err != nil {
		t.Fatal("DSN gobierno fuera del clon TLS exacto")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	admin, err := pgxpool.NewWithConfig(ctx, adminConfig)
	if err != nil {
		t.Fatal("DBA del clon no disponible")
	}
	defer admin.Close()
	gobierno, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, gobiernoDSN,
		"vec-hito1-usuarios-preferencias-provision", rolGobiernoPostgreSQLContratacionTemporalDesarrollo)
	if err != nil {
		t.Fatal("LOGIN de gobierno no acreditado")
	}
	defer gobierno.Close()
	if verificarDestinoProvisionPreferenciasHito1(ctx, admin, gobierno, identificadorHito1) != nil ||
		verificarLoginsUsuariosProvisionPreferenciasHito1(ctx, gobierno, configuraciones) != nil ||
		verificarClavesPreferenciasHito1(ctx, gobierno) != nil {
		t.Fatal("preflight del clon, roles o audiencias V3 rechazado")
	}
	revision, huellaCatalogo, err := catalogoPoliticasProvisionPreferenciasHito1(ctx, gobierno)
	if err != nil {
		t.Fatal("catálogo de políticas fuera de la preimagen vacía HITO1")
	}
	cuentas := make([]cuentaProvisionPreferenciasHito1, 2)
	for i, c := range configuraciones {
		cuentas[i], err = construirCuentaProvisionPreferenciasHito1(c, ahora)
		if err != nil {
			t.Fatal("Contexto V2 sintético inválido")
		}
		cuentas[i].instantanea, err = instantaneaProvisionPreferenciasHito1(cuentas[i], ahora, revision, huellaCatalogo)
		if err != nil {
			t.Fatal("asignación V3 sintética inválida")
		}
	}
	if cuentas[0].personaRef == cuentas[1].personaRef || cuentas[0].cuenta.CuentaRef == cuentas[1].cuenta.CuentaRef ||
		cuentas[0].cuenta.PerfilRef == cuentas[1].cuenta.PerfilRef {
		t.Fatal("identidades sintéticas de ambos portales mezcladas")
	}
	estados, err := estadoProvisionPreferenciasHito1(ctx, gobierno, cuentas)
	if err != nil {
		t.Fatal("preimagen parcial o ajena; no se publica")
	}
	// Las dos autoridades publican en transacciones propias. Se prevalidan las
	// dos superficies antes del primer efecto. Un fallo posterior se detiene:
	// la repetición sólo acepta estado ausente o postimagen exacta.
	for i, c := range cuentas {
		if err = publicarResultadoContextoPostgreSQLDesarrollo(ctx, gobierno, c.contexto, c.operacionRef); err != nil {
			t.Fatalf("Contexto sintético %d no publicado o postimagen distinta", i+1)
		}
		publicador := autoridadPostgreSQLDesarrollo{
			pool: gobierno, vinculo: c.vinculo,
			prefijoBloqueo: "vec:hito1:usuarios:preferencias:",
			actoControlRol: "acto:hito1:usuarios:preferencias:control:" + string(c.configuracion.Superficie),
			actoAsignacion: "acto:hito1:usuarios:preferencias:asignacion:" + string(c.configuracion.Superficie),
			actoSesion:     "acto:hito1:usuarios:preferencias:sesion:" + string(c.configuracion.Superficie),
			soloInicial:    estados[i][1] == 0,
		}
		instantanea := c.instantanea
		if estados[i][1] == 0 {
			instantanea, err = publicador.prepararInstantanea(ctx, instantanea, true)
			if err != nil || instantanea.Validar() != nil || instantanea.VersionRol.Referencia() != c.instantanea.VersionRol.Referencia() ||
				instantanea.AsignacionPerfil.Referencia() != c.instantanea.AsignacionPerfil.Referencia() {
				t.Fatalf("preparación de concesión %d divergente", i+1)
			}
		}
		if err = publicador.PublicarInstantanea(ctx, instantanea); err != nil {
			t.Fatalf("concesión sintética %d no publicada o postimagen distinta", i+1)
		}
	}
	post, err := estadoProvisionPreferenciasHito1(ctx, gobierno, cuentas)
	if err != nil || post[0] != [2]int{10, 8} || post[1] != [2]int{10, 8} ||
		verificarClavesPreferenciasHito1(ctx, gobierno) != nil {
		t.Fatal("postimagen HITO1 o gobierno V3 divergente")
	}
	t.Log("recibo sintético HITO1: dos contextos y dos asignaciones propias; claves V3 conservadas")
}

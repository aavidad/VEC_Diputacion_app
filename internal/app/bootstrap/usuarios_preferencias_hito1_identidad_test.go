package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	postgresidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	core "vec-diputacion-granada/internal/vec/domain"
)

const (
	marcaIdentidadPreferenciasHito1 = "HITO1_PREF_IDENTIDAD_APLICAR_REVISADO"
	loginProvisionadorHito1         = "vec_hito1_pref_provisionador"
)

type identidadConciliadaPreferenciasHito1 struct {
	Superficie  string `json:"superficie"`
	CuentaNueva string `json:"cuenta_ref_nueva"`
}

type resultadoConciliacionPreferenciasHito1 struct {
	Version int                                    `json:"version"`
	Cuentas []identidadConciliadaPreferenciasHito1 `json:"cuentas"`
}

func leerConfiguracionOriginalPreferenciasHito1(ruta string, superficie core.SuperficieAutenticacionActorV1) (configuracionUsuariosPreferenciasDesarrollo, error) {
	vacia := configuracionUsuariosPreferenciasDesarrollo{}
	contenido, err := leerFicheroMaterialSeguro(ruta, 128<<10)
	if err != nil || validarClavesJSONUnicas(contenido) != nil {
		return vacia, errComposicionUsuariosPreferencias
	}
	defer borrarBytes(contenido)
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	var c configuracionUsuariosPreferenciasDesarrollo
	var extra any
	if dec.Decode(&c) != nil || !errors.Is(dec.Decode(&extra), io.EOF) || c.Version != 1 ||
		c.Autoridad != AutoridadNoAutoritativa || c.Superficie != superficie || len(c.Cuentas) != 1 {
		return vacia, errComposicionUsuariosPreferencias
	}
	return c, nil
}

func configuracionesOriginalesPreferenciasHito1(materialDir, copiaInterna, copiaExterna string) ([]configuracionUsuariosPreferenciasDesarrollo, error) {
	if !directorioProvisionPreferenciasHito1(materialDir) || !filepath.IsAbs(copiaInterna) || !filepath.IsAbs(copiaExterna) ||
		filepath.Dir(copiaInterna) != filepath.Join(materialDir, "identidad") || filepath.Dir(copiaExterna) != filepath.Join(materialDir, "identidad") ||
		copiaInterna == copiaExterna {
		return nil, errComposicionUsuariosPreferencias
	}
	interna, err := leerConfiguracionOriginalPreferenciasHito1(copiaInterna, core.SuperficieAutenticacionInternaCorporativaV1)
	if err != nil {
		return nil, err
	}
	externa, err := leerConfiguracionOriginalPreferenciasHito1(copiaExterna, core.SuperficieAutenticacionExternaPersonalV1)
	if err != nil || !configuracionesPreferenciasSeparadas(interna, externa) {
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

func seudonimosCuentaPreferenciasHito1(ctx context.Context, derivador *derivadorIdentidadOperacionDesarrollo,
	c configuracionUsuariosPreferenciasDesarrollo, cuentaID string,
) (postgresidentidad.SeudonimosAlta, error) {
	vacio := postgresidentidad.SeudonimosAlta{}
	if derivador == nil || !derivador.valido() || len(c.Cuentas) != 1 || cuentaID == "" {
		return vacio, errComposicionUsuariosPreferencias
	}
	base := string(c.Superficie) + "-preferencias-hito1"
	resultado, err := (&seudonimizadorSesionDesarrollo{derivador: derivador}).SeudonimizarAlta(ctx, postgresidentidad.IdentificadoresAlta{
		EspacioIdentidad: espacioIdentidadSesionDesarrollo,
		AsercionID:       base + "-asercion",
		SesionID:         base + "-sesion",
		SujetoID:         c.Cuentas[0].Sujeto,
		CuentaID:         cuentaID,
	})
	if err != nil || resultado.Esquema != postgresidentidad.EsquemaHMACSHA256V1 || resultado.DominioRef != dominioIdentidadSesionDesarrollo ||
		resultado.ClaveVersion == 0 || resultado.ClaveVersion > uint64(1<<63-1) {
		return vacio, errComposicionUsuariosPreferencias
	}
	return resultado, nil
}

func operacionProvisionPreferenciasHito1(c configuracionUsuariosPreferenciasDesarrollo, fase, cuentaRef string) string {
	return referenciaProvisionPreferenciasHito1("opr_", string(c.Superficie)+"\x00"+c.Cuentas[0].CertificadoSHA256+"\x00"+fase+"\x00"+cuentaRef)
}

// El único DDL del subcorte crea un LOGIN temporal propio del clon. Ninguna
// tabla de datos se escribe fuera de las funciones/autoridades gobernadas.
func crearOComprobarLoginProvisionadorHito1(ctx context.Context, admin *pgxpool.Pool) error {
	if admin == nil {
		return errComposicionUsuariosPreferencias
	}
	tx, err := admin.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return errComposicionUsuariosPreferencias
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:hito1:preferencias:login-provisionador',0))`); err != nil {
		return errComposicionUsuariosPreferencias
	}
	var existe bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=$1)`, loginProvisionadorHito1).Scan(&existe); err != nil {
		return errComposicionUsuariosPreferencias
	}
	if !existe {
		if _, err = tx.Exec(ctx, `CREATE ROLE vec_hito1_pref_provisionador LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS`); err != nil {
			return errComposicionUsuariosPreferencias
		}
		if _, err = tx.Exec(ctx, `GRANT vec_identidad_sesiones_v1_provisionador TO vec_hito1_pref_provisionador WITH INHERIT TRUE, SET FALSE, ADMIN FALSE`); err != nil {
			return errComposicionUsuariosPreferencias
		}
	}
	var valido bool
	err = tx.QueryRow(ctx, `SELECT r.rolcanlogin AND r.rolinherit AND NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
 AND (SELECT count(*)=1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid)
 AND (SELECT count(*)=1 FROM pg_catalog.pg_auth_members m WHERE m.roleid='vec_identidad_sesiones_v1_provisionador'::regrole)
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
 WHERE m.member=r.oid AND g.rolname='vec_identidad_sesiones_v1_provisionador'
 AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members superior WHERE superior.member=g.oid))
 FROM pg_catalog.pg_roles r WHERE r.rolname=$1`, loginProvisionadorHito1).Scan(&valido)
	if err != nil || !valido || tx.Commit(ctx) != nil {
		return errComposicionUsuariosPreferencias
	}
	return nil
}

func comprobarLoginProvisionadorHito1(ctx context.Context, pool *pgxpool.Pool, topologia topologiaPostgreSQLPreferenciasUsuarios) error {
	if pool == nil || cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, pool, topologia) != nil {
		return errComposicionUsuariosPreferencias
	}
	var valido bool
	err := pool.QueryRow(ctx, `SELECT session_user=current_user AND session_user=$1
 AND has_schema_privilege(session_user,'vec_identidad_sesiones_v1','USAGE')
 AND has_function_privilege(session_user,'vec_identidad_sesiones_v1.provisionar_cuenta_v1(text,text,text,text,bigint,bytea,bytea,boolean,bytea)','EXECUTE')
 AND has_function_privilege(session_user,'vec_identidad_sesiones_v1.registrar_alias_hmac_cuenta_v1(text,text,text,text,text,bigint,bytea,bytea)','EXECUTE')
 AND (SELECT count(*)=1 FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole)
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole
 AND m.roleid='vec_identidad_sesiones_v1_provisionador'::regrole
 AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)`, loginProvisionadorHito1).Scan(&valido)
	if err != nil || !valido {
		return errComposicionUsuariosPreferencias
	}
	return nil
}

func cuentaIdentidadActivaHito1(ctx context.Context, admin *pgxpool.Pool, cuentaRef string) error {
	var valida bool
	err := admin.QueryRow(ctx, `SELECT count(*)=1 AND bool_and(NOT c.cuenta_privilegiada AND c.cuenta_ordinaria_ref IS NULL AND e.estado='activa' AND a.revision=1)
 FROM vec_identidad_sesiones_v1.cuenta c
 JOIN vec_identidad_sesiones_v1.estado_cuenta_actual a USING(cuenta_ref)
 JOIN vec_identidad_sesiones_v1.estado_cuenta e ON e.cuenta_ref=a.cuenta_ref AND e.revision=a.revision
 WHERE c.cuenta_ref=$1`, cuentaRef).Scan(&valida)
	if err != nil || !valida {
		return errComposicionUsuariosPreferencias
	}
	return nil
}

func provisionarYRegistrarAliasIdentidadHito1(ctx context.Context, pool, admin *pgxpool.Pool,
	derivador *derivadorIdentidadOperacionDesarrollo, c configuracionUsuariosPreferenciasDesarrollo,
) (string, error) {
	if pool == nil || admin == nil || len(c.Cuentas) != 1 || derivador == nil {
		return "", errComposicionUsuariosPreferencias
	}
	cuentaAnterior := c.Cuentas[0].CuentaRef
	primera, err := seudonimosCuentaPreferenciasHito1(ctx, derivador, c, "desarrollo:"+cuentaAnterior)
	if err != nil {
		return "", errComposicionUsuariosPreferencias
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return "", errComposicionUsuariosPreferencias
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL search_path = pg_catalog`); err != nil {
		return "", errComposicionUsuariosPreferencias
	}
	operacionCuenta := operacionProvisionPreferenciasHito1(c, "provision-cuenta", cuentaAnterior)
	var cuentaNueva string
	err = tx.QueryRow(ctx, `SELECT cuenta_ref FROM vec_identidad_sesiones_v1.provisionar_cuenta_v1(
 $1,$2,$3,$4,$5,$6,$7,false,NULL::bytea)`, operacionCuenta, primera.Esquema,
		primera.DominioRef, primera.ClaveID, int64(primera.ClaveVersion), primera.CuentaIDHMAC[:], primera.SujetoIDHMAC[:]).Scan(&cuentaNueva)
	if err != nil || (core.CuentaAutenticadaContextoActor{CuentaRef: cuentaNueva, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh}).Validar() != nil || cuentaNueva == cuentaAnterior {
		return "", errComposicionUsuariosPreferencias
	}
	segunda, err := seudonimosCuentaPreferenciasHito1(ctx, derivador, c, "desarrollo:"+cuentaNueva)
	if err != nil || primera.Esquema != segunda.Esquema || primera.DominioRef != segunda.DominioRef ||
		primera.ClaveID != segunda.ClaveID || primera.ClaveVersion != segunda.ClaveVersion ||
		primera.SujetoIDHMAC != segunda.SujetoIDHMAC || primera.CuentaIDHMAC == segunda.CuentaIDHMAC {
		return "", errComposicionUsuariosPreferencias
	}
	operacionAlias := operacionProvisionPreferenciasHito1(c, "alias-cuenta-canonica", cuentaNueva)
	var confirmada string
	err = tx.QueryRow(ctx, `SELECT vec_identidad_sesiones_v1.registrar_alias_hmac_cuenta_v1(
 $1,$2,$3,$4,$5,$6,$7,$8)`, operacionAlias, cuentaNueva, segunda.Esquema,
		segunda.DominioRef, segunda.ClaveID, int64(segunda.ClaveVersion), segunda.CuentaIDHMAC[:], segunda.SujetoIDHMAC[:]).Scan(&confirmada)
	if err != nil || confirmada != cuentaNueva {
		return "", errComposicionUsuariosPreferencias
	}
	if tx.Commit(ctx) != nil || cuentaIdentidadActivaHito1(ctx, admin, cuentaNueva) != nil {
		return "", errComposicionUsuariosPreferencias
	}
	var aliasCount int
	err = admin.QueryRow(ctx, `SELECT count(*) FROM vec_identidad_sesiones_v1.alias_hmac_cuenta
 WHERE cuenta_ref=$1 AND esquema_hmac=$2 AND dominio_hmac_ref=$3 AND clave_hmac_id=$4 AND clave_hmac_version=$5
 AND sujeto_id_hmac=$6 AND cuenta_id_hmac IN ($7,$8)`, cuentaNueva, segunda.Esquema, segunda.DominioRef,
		segunda.ClaveID, int64(segunda.ClaveVersion), segunda.SujetoIDHMAC[:], primera.CuentaIDHMAC[:], segunda.CuentaIDHMAC[:]).Scan(&aliasCount)
	if err != nil || aliasCount != 2 {
		return "", errComposicionUsuariosPreferencias
	}
	return cuentaNueva, nil
}

func nuevoResultadoCuentaConciliadaHito1(anterior cuentaProvisionPreferenciasHito1, nueva string) (core.ResultadoContextoActorRegistradoV2, string, error) {
	vacio := core.ResultadoContextoActorRegistradoV2{}
	if nueva == "" || nueva == anterior.cuenta.CuentaRef || anterior.contexto.Validar() != nil {
		return vacio, "", errComposicionUsuariosPreferencias
	}
	base := string(anterior.configuracion.Superficie) + "\x00" + anterior.cuenta.CuentaRef + "\x00" + nueva + "\x00" + anterior.cuenta.PerfilRef
	vinculoNuevo := referenciaProvisionPreferenciasHito1("vca_", base+"\x00cuenta-conciliada")
	instantanea := anterior.contexto.Contexto.Instantanea
	instantanea.CuentaRef = nueva
	instantanea.VinculoRef = vinculoNuevo
	cuenta := core.CuentaAutenticadaContextoActor{CuentaRef: nueva, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh}
	actor, err := core.NuevoContextoActor(cuenta, instantanea, anterior.contexto.ResueltoEnAutoritativo)
	if err != nil || actor.PersonaRef != anterior.personaRef || actor.PerfilActivoRef != anterior.cuenta.PerfilRef {
		return vacio, "", errComposicionUsuariosPreferencias
	}
	manifiesto, err := core.RehidratarManifiestoProcedenciaContextoActorV1(anterior.contexto.ManifiestoProcedenciaCanonico)
	if err != nil {
		return vacio, "", errComposicionUsuariosPreferencias
	}
	manifiesto.Cuenta.CuentaRef = nueva
	manifiesto.Contexto.VinculoRef = vinculoNuevo
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		return vacio, "", errComposicionUsuariosPreferencias
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		return vacio, "", errComposicionUsuariosPreferencias
	}
	manifiestoCanon, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		return vacio, "", errComposicionUsuariosPreferencias
	}
	manifiestoHuella, err := core.HuellaSHA256ManifiestoProcedenciaContextoActorV1(manifiestoCanon)
	if err != nil {
		return vacio, "", errComposicionUsuariosPreferencias
	}
	resultado := core.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: referenciaProvisionPreferenciasHito1("rca_", base+"\x00registro-conciliado"),
		Contexto:            actor, RepresentacionCanonica: canon, HuellaSHA256: huella,
		ManifiestoProcedenciaCanonico: manifiestoCanon, ManifiestoProcedenciaHuellaSHA256: manifiestoHuella,
		AutoridadEfectiva:      core.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo: anterior.contexto.ResueltoEnAutoritativo,
	}
	operacion := referenciaProvisionPreferenciasHito1("oca_", base+"\x00publicar-conciliado")
	if resultado.Validar() != nil {
		return vacio, "", errComposicionUsuariosPreferencias
	}
	return resultado, operacion, nil
}

func estadoContextoConciliadoHito1(ctx context.Context, admin *pgxpool.Pool, r core.ResultadoContextoActorRegistradoV2, operacion string) (int, error) {
	if admin == nil || r.Validar() != nil || operacion == "" {
		return 0, errComposicionUsuariosPreferencias
	}
	var n [5]int
	err := admin.QueryRow(ctx, `SELECT
 (SELECT count(*) FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones WHERE cuenta_ref=$1),
 (SELECT count(*) FROM vec_contexto_actor_v1.proyeccion_cuenta_actual WHERE cuenta_ref=$1),
 (SELECT count(*) FROM vec_contexto_actor_v1.vinculo_contexto_versiones WHERE vinculo_ref=$2),
 (SELECT count(*) FROM vec_contexto_actor_v1.vinculo_contexto_actual WHERE vinculo_ref=$2),
 (SELECT count(*) FROM vec_contexto_actor_v1.registros_contexto WHERE operacion_ref=$3)`,
		r.Contexto.Instantanea.CuentaRef, r.Contexto.Instantanea.VinculoRef, operacion).Scan(&n[0], &n[1], &n[2], &n[3], &n[4])
	if err != nil {
		return 0, errComposicionUsuariosPreferencias
	}
	suma := 0
	for _, v := range n {
		if v > 1 {
			return 0, errComposicionUsuariosPreferencias
		}
		suma += v
	}
	if suma != 0 && suma != 5 {
		return 0, errComposicionUsuariosPreferencias
	}
	return suma, nil
}

func escribirResultadoConciliacionHito1(ruta, materialDir string, resultado resultadoConciliacionPreferenciasHito1) error {
	if !directorioProvisionPreferenciasHito1(materialDir) || !filepath.IsAbs(ruta) ||
		filepath.Dir(ruta) != filepath.Join(materialDir, "identidad") ||
		filepath.Base(ruta) != "resultado-identidad-preferencias-hito1.json" || resultado.Version != 1 || len(resultado.Cuentas) != 2 ||
		resultado.Cuentas[0].Superficie != string(core.SuperficieAutenticacionInternaCorporativaV1) ||
		resultado.Cuentas[1].Superficie != string(core.SuperficieAutenticacionExternaPersonalV1) ||
		resultado.Cuentas[0].CuentaNueva == resultado.Cuentas[1].CuentaNueva {
		return errComposicionUsuariosPreferencias
	}
	for _, c := range resultado.Cuentas {
		if (core.CuentaAutenticadaContextoActor{CuentaRef: c.CuentaNueva, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh}).Validar() != nil {
			return errComposicionUsuariosPreferencias
		}
	}
	contenido, err := json.Marshal(resultado)
	if err != nil {
		return errComposicionUsuariosPreferencias
	}
	if _, err = os.Lstat(ruta); err == nil {
		actual, e := leerFicheroMaterialSeguro(ruta, 4096)
		if e != nil || !bytes.Equal(actual, contenido) {
			return errComposicionUsuariosPreferencias
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return errComposicionUsuariosPreferencias
	}
	fichero, err := os.OpenFile(ruta, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return errComposicionUsuariosPreferencias
	}
	n, escrito := fichero.Write(contenido)
	sinc := fichero.Sync()
	cierre := fichero.Close()
	if escrito != nil || sinc != nil || cierre != nil || n != len(contenido) {
		_ = os.Remove(ruta)
		return errComposicionUsuariosPreferencias
	}
	return nil
}

func TestContextoConciliadoHito1MantienePersonaYPerfil(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond).Add(-10 * time.Minute)
	c := configuracionUsuariosPreferenciasDesarrollo{Superficie: core.SuperficieAutenticacionInternaCorporativaV1,
		Cuentas: []cuentaUsuariosPreferenciasDesarrollo{{cuentaRutasDietasDesarrollo: cuentaRutasDietasDesarrollo{
			CuentaRef: "cta_0123456789abcdefghijklmnop", PerfilRef: "prf_0123456789abcdefghijklmnop",
			CertificadoSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		}}},
	}
	previa, err := construirCuentaProvisionPreferenciasHito1(c, ahora)
	if err != nil {
		t.Fatal(err)
	}
	nueva, operacion, err := nuevoResultadoCuentaConciliadaHito1(previa, "cta_abcdef0123456789abcdefghijkl")
	if err != nil || nueva.Validar() != nil || operacion == "" || nueva.Contexto.PersonaRef != previa.personaRef ||
		nueva.Contexto.PerfilActivoRef != previa.cuenta.PerfilRef ||
		nueva.Contexto.Instantanea.CuentaRef == previa.cuenta.CuentaRef ||
		nueva.Contexto.Instantanea.VinculoRef == previa.contexto.Contexto.Instantanea.VinculoRef {
		t.Fatalf("conciliación alteró Persona/Perfil o reutilizó vínculo: %v", err)
	}
}

func TestProvisionarIdentidadPreferenciasHito1(t *testing.T) {
	if os.Getenv("VEC_PREF_HITO1_IDENTIDAD_PROVISIONAR") != marcaIdentidadPreferenciasHito1 {
		t.Skip("conciliación privada HITO1 deshabilitada")
	}
	materialDir := os.Getenv("VEC_PREF_HITO1_MATERIAL_DIR")
	if !directorioProvisionPreferenciasHito1(materialDir) {
		t.Fatal("directorio HITO1 no acreditado")
	}
	ahora, err := time.Parse(time.RFC3339, instanteProvisionHito1)
	if err != nil || !time.Now().UTC().Before(ahora.Add(24*time.Hour)) {
		t.Fatal("ventana de la preimagen HITO1 caducada")
	}
	configuraciones, err := configuracionesOriginalesPreferenciasHito1(materialDir,
		os.Getenv("VEC_PREF_HITO1_COPIA_INTERNA"), os.Getenv("VEC_PREF_HITO1_COPIA_EXTERNA"))
	if err != nil {
		t.Fatal("copias de identidad HITO1 no acreditadas")
	}
	adminDSN, gobiernoDSN, provisionadorDSN := os.Getenv("VEC_PREF_HITO1_ADMIN_DSN"), os.Getenv("VEC_PREF_HITO1_GOBIERNO_DSN"), os.Getenv("VEC_PREF_HITO1_PROVISIONADOR_DSN")
	adminConfig, err := dsnProvisionPreferenciasHito1(adminDSN)
	if err != nil {
		t.Fatal("DBA fuera del clon TLS exacto")
	}
	if _, err = dsnProvisionPreferenciasHito1(gobiernoDSN); err != nil {
		t.Fatal("gobierno fuera del clon TLS exacto")
	}
	provisionadorConfig, err := dsnProvisionPreferenciasHito1(provisionadorDSN)
	if err != nil || provisionadorConfig.ConnConfig.User != loginProvisionadorHito1 ||
		adminConfig.ConnConfig.User == loginProvisionadorHito1 {
		t.Fatal("LOGIN provisionador fuera del perfil exclusivo")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelar()
	admin, err := pgxpool.NewWithConfig(ctx, adminConfig)
	if err != nil {
		t.Fatal("DBA HITO1 no disponible")
	}
	defer admin.Close()
	gobierno, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, gobiernoDSN,
		"vec-hito1-pref-identidad-gobierno", rolGobiernoPostgreSQLContratacionTemporalDesarrollo)
	if err != nil {
		t.Fatal("gobierno HITO1 no disponible")
	}
	defer gobierno.Close()
	if verificarDestinoProvisionPreferenciasHito1(ctx, admin, gobierno, identificadorHito1) != nil ||
		verificarClavesPreferenciasHito1(ctx, gobierno) != nil {
		t.Fatal("clon o cuatro audiencias V3 no acreditados")
	}
	material, err := cargarMaterialIdempotenciaDesarrollo(materialDir, filepath.Join(materialDir, "idempotencia", "configuracion.json"))
	if err != nil {
		t.Fatal("material HMAC HITO1 no disponible")
	}
	derivador, err := nuevoDerivadorIdentidadOperacionDesarrollo(&material)
	if err != nil {
		material.borrar()
		t.Fatal("derivador HMAC HITO1 no disponible")
	}
	defer derivador.borrar()
	anteriores := make([]cuentaProvisionPreferenciasHito1, 2)
	for i, c := range configuraciones {
		anteriores[i], err = construirCuentaProvisionPreferenciasHito1(c, ahora)
		if err != nil {
			t.Fatal("preimagen Contexto sintético inválida")
		}
		var cuenta, estado, alias int
		err = admin.QueryRow(ctx, `SELECT
 (SELECT count(*) FROM vec_identidad_sesiones_v1.cuenta WHERE cuenta_ref=$1),
 (SELECT count(*) FROM vec_identidad_sesiones_v1.estado_cuenta_actual WHERE cuenta_ref=$1),
 (SELECT count(*) FROM vec_identidad_sesiones_v1.alias_hmac_cuenta WHERE cuenta_ref=$1)`, c.Cuentas[0].CuentaRef).Scan(&cuenta, &estado, &alias)
		if err != nil || cuenta != 0 || estado != 0 || alias != 0 {
			t.Fatal("cuenta antigua de Identidad presente o deriva de preimagen")
		}
	}
	estados, err := estadoProvisionPreferenciasHito1(ctx, gobierno, anteriores)
	if err != nil || estados[0] != [2]int{10, 8} || estados[1] != [2]int{10, 8} {
		t.Fatal("Contexto/V3 anteriores no conservados exactos")
	}
	// Paso administrativo explícito del clon: sin LOGIN miembro provisionador
	// no se permite llamar funciones de Identidad con el gobierno CT.
	if crearOComprobarLoginProvisionadorHito1(ctx, admin) != nil {
		t.Fatal("LOGIN provisionador HITO1 no creado/acreditado")
	}
	provisionador, err := pgxpool.NewWithConfig(ctx, provisionadorConfig)
	if err != nil {
		t.Fatal("LOGIN provisionador HITO1 no disponible")
	}
	defer provisionador.Close()
	topologia, err := acreditarTopologiaPostgreSQLPreferenciasUsuarios(ctx, admin)
	if err != nil || comprobarLoginProvisionadorHito1(ctx, provisionador, topologia) != nil {
		t.Fatal("ACL, identidad o topología del provisionador rechazadas")
	}
	resultado := resultadoConciliacionPreferenciasHito1{Version: 1, Cuentas: make([]identidadConciliadaPreferenciasHito1, 2)}
	for i, c := range configuraciones {
		cuentaNueva, e := provisionarYRegistrarAliasIdentidadHito1(ctx, provisionador, admin, derivador, c)
		if e != nil {
			t.Fatalf("cuenta y alias de superficie %d no provisionados", i+1)
		}
		r, operacion, e := nuevoResultadoCuentaConciliadaHito1(anteriores[i], cuentaNueva)
		if e != nil {
			t.Fatalf("Contexto conciliado de superficie %d inválido", i+1)
		}
		previo, e := estadoContextoConciliadoHito1(ctx, admin, r, operacion)
		if e != nil || (previo != 0 && previo != 5) {
			t.Fatalf("preimagen Contexto de superficie %d parcial", i+1)
		}
		if e = publicarResultadoContextoPostgreSQLDesarrollo(ctx, gobierno, r, operacion); e != nil {
			t.Fatalf("Contexto de superficie %d no publicado o replay divergente", i+1)
		}
		posterior, e := estadoContextoConciliadoHito1(ctx, admin, r, operacion)
		if e != nil || posterior != 5 || cuentaIdentidadActivaHito1(ctx, admin, cuentaNueva) != nil {
			t.Fatalf("postimagen de superficie %d divergente", i+1)
		}
		resultado.Cuentas[i] = identidadConciliadaPreferenciasHito1{Superficie: string(c.Superficie), CuentaNueva: cuentaNueva}
	}
	if resultado.Cuentas[0].CuentaNueva == resultado.Cuentas[1].CuentaNueva ||
		verificarClavesPreferenciasHito1(ctx, gobierno) != nil {
		t.Fatal("conciliación mezcló cuentas o cambió audiencias V3")
	}
	if escribirResultadoConciliacionHito1(os.Getenv("VEC_PREF_HITO1_IDENTIDAD_RESULTADO"), materialDir, resultado) != nil {
		t.Fatal("resultado privado no escrito o replay divergente")
	}
	t.Log("recibo sintético HITO1 Identidad: dos cuentas y alias canónicos, dos vínculos Contexto; V3 conservada")
}

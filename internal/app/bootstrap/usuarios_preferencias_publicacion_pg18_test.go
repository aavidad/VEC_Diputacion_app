package bootstrap

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type destinoEnsayoPreferenciasPG18 struct {
	base, direccion, inicio string
	puerto, version         int
}

func (d destinoEnsayoPreferenciasPG18) valido() bool {
	return strings.HasPrefix(d.base, "vec_pref_lote_") && d.direccion != "" && d.inicio != "" && d.puerto > 0 && d.version/10000 == 18
}

func mismoDestinoEnsayoPreferenciasPG18(adminDSN, gobiernoDSN string, admin, gobierno destinoEnsayoPreferenciasPG18) bool {
	adminConfig, errA := pgxpool.ParseConfig(adminDSN)
	gobiernoConfig, errG := pgxpool.ParseConfig(gobiernoDSN)
	if errA != nil || errG != nil || adminConfig == nil || gobiernoConfig == nil || adminConfig.ConnConfig == nil || gobiernoConfig.ConnConfig == nil {
		return false
	}
	a, g := adminConfig.ConnConfig, gobiernoConfig.ConnConfig
	return a.Host != "" && a.Host == g.Host && a.Port != 0 && a.Port == g.Port &&
		a.Database == admin.base && g.Database == gobierno.base && a.Database == g.Database &&
		len(a.Fallbacks) == 0 && len(g.Fallbacks) == 0 && admin.valido() && gobierno.valido() && admin == gobierno
}

func destinoObservadoEnsayoPreferenciasPG18(ctx context.Context, pool *pgxpool.Pool) (destinoEnsayoPreferenciasPG18, error) {
	var destino destinoEnsayoPreferenciasPG18
	if pool == nil {
		return destino, errComposicionUsuariosPreferencias
	}
	err := pool.QueryRow(ctx, `SELECT current_database()::text, COALESCE(inet_server_addr()::text,''),
 COALESCE(inet_server_port(),0), current_setting('server_version_num')::int,
	extract(epoch FROM pg_postmaster_start_time())::text`).Scan(&destino.base, &destino.direccion, &destino.puerto, &destino.version, &destino.inicio)
	return destino, err
}

func TestEnsayoPG18RechazaDestinosDistintosAntesDePublicar(t *testing.T) {
	comun := destinoEnsayoPreferenciasPG18{base: "vec_pref_lote_prueba", direccion: "127.0.0.1", puerto: 55432, version: 180004, inicio: "2026-09-29 00:00:00+00"}
	admin := "postgres://administrador@127.0.0.1:55432/vec_pref_lote_prueba?sslmode=require"
	gobierno := "postgres://gobierno@127.0.0.1:55432/vec_pref_lote_prueba?sslmode=require"
	if !mismoDestinoEnsayoPreferenciasPG18(admin, gobierno, comun, comun) {
		t.Fatal("dos LOGIN de una misma base efímera rechazados")
	}
	for _, caso := range []struct {
		nombre, dsn string
		observado   destinoEnsayoPreferenciasPG18
	}{
		{"otra_base_mismo_cluster", "postgres://gobierno@127.0.0.1:55432/vec_pref_lote_otra?sslmode=require", destinoEnsayoPreferenciasPG18{base: "vec_pref_lote_otra", direccion: comun.direccion, puerto: comun.puerto, version: comun.version, inicio: comun.inicio}},
		{"mismo_nombre_otro_servidor", gobierno, destinoEnsayoPreferenciasPG18{base: comun.base, direccion: "127.0.0.2", puerto: comun.puerto, version: comun.version, inicio: comun.inicio}},
		{"otro_host_configurado", "postgres://gobierno@127.0.0.2:55432/vec_pref_lote_prueba?sslmode=require", comun},
		{"base_no_efimera", "postgres://gobierno@127.0.0.1:55432/vec_principal?sslmode=require", destinoEnsayoPreferenciasPG18{base: "vec_principal", direccion: comun.direccion, puerto: comun.puerto, version: comun.version, inicio: comun.inicio}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if mismoDestinoEnsayoPreferenciasPG18(admin, caso.dsn, comun, caso.observado) {
				t.Fatal("destino no acreditado aceptado")
			}
		})
	}
}

// Exige una base PostgreSQL 18 efímera llamada vec_pref_lote_* con la
// preimagen canónica AD3 y AD3-106 ya instaladas. No utiliza el clon privado
// conservado ni crea roles o migraciones. El runner externo aporta dos LOGIN:
// DBA de esa base desechable y gobierno nominal de vec-server.
func TestPreferenciasLoteGobiernoPostgreSQL18(t *testing.T) {
	if os.Getenv("VEC_PREF_LOTE_PG_DESECHABLE") != "si" {
		t.Skip("requiere PostgreSQL 18 desechable exclusivo")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	adminDSN, gobiernoDSN := os.Getenv("VEC_PREF_LOTE_ADMIN_DSN"), os.Getenv("VEC_PREF_LOTE_GOBIERNO_DSN")
	adminConfig, err := pgxpool.ParseConfig(adminDSN)
	gobiernoConfig, errGobierno := pgxpool.ParseConfig(gobiernoDSN)
	if err != nil || errGobierno != nil || adminConfig == nil || gobiernoConfig == nil || adminConfig.ConnConfig == nil || gobiernoConfig.ConnConfig == nil ||
		adminConfig.ConnConfig.Host == "" || adminConfig.ConnConfig.Host != gobiernoConfig.ConnConfig.Host ||
		adminConfig.ConnConfig.Port == 0 || adminConfig.ConnConfig.Port != gobiernoConfig.ConnConfig.Port ||
		!strings.HasPrefix(adminConfig.ConnConfig.Database, "vec_pref_lote_") || adminConfig.ConnConfig.Database != gobiernoConfig.ConnConfig.Database ||
		len(adminConfig.ConnConfig.Fallbacks) != 0 || len(gobiernoConfig.ConnConfig.Fallbacks) != 0 {
		t.Fatal("DSN de ensayo no apuntan a un mismo destino efímero")
	}
	admin, err := pgxpool.NewWithConfig(ctx, adminConfig)
	if err != nil {
		t.Fatal("DBA efímero no disponible")
	}
	defer admin.Close()
	adminObservado, err := destinoObservadoEnsayoPreferenciasPG18(ctx, admin)
	if err != nil || !adminObservado.valido() || adminObservado.base != adminConfig.ConnConfig.Database {
		t.Fatal("base DBA no acreditada como efímera PostgreSQL18")
	}
	gobierno, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, gobiernoDSN, "vec-pref-lote-pg18", rolGobiernoPostgreSQLContratacionTemporalDesarrollo)
	if err != nil {
		t.Fatal("LOGIN de gobierno nominal no acreditado")
	}
	defer gobierno.Close()
	gobiernoObservado, err := destinoObservadoEnsayoPreferenciasPG18(ctx, gobierno)
	if err != nil || !mismoDestinoEnsayoPreferenciasPG18(adminDSN, gobiernoDSN, adminObservado, gobiernoObservado) {
		t.Fatal("gobierno no conecta al servidor y base efímeros acreditados")
	}
	const contarUsuarios = `SELECT count(*) FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE audiencia_consumo LIKE 'vec_usuarios.preferencias.%'`
	var antes int
	if err = admin.QueryRow(ctx, contarUsuarios).Scan(&antes); err != nil || antes != 0 {
		t.Fatalf("la base efímera ya contiene claves Usuarios: %d %v", antes, err)
	}
	if err = admin.QueryRow(ctx, `SELECT count(*) FROM vec_autorizacion_atestada_v3.clave_capacidad_version`).Scan(&antes); err != nil || antes != 0 {
		t.Fatalf("el gobierno efímero no está vacío: %d %v", antes, err)
	}
	material := materialRenovableCTPrueba(t, time.Now().UTC().Truncate(time.Microsecond).Add(-24*time.Hour))
	if err = publicarGobiernoAtestacionContratacionTemporalDesarrollo(ctx, gobierno, &material); err != nil {
		t.Fatalf("clave CT base no publicable: %v", err)
	}
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptoresMaterialPreferenciasUsuariosDesarrollo())
	if err != nil {
		t.Fatal(err)
	}
	const huellaFunciones = `SELECT md5(string_agg(pg_get_functiondef(p.oid),E'\n' ORDER BY p.oid::regprocedure::text))
 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname IN ('consumir_decision_mutacion_v3_interna','texto_tecnico_valido')`
	var funcionesAntes, funcionesDespues string
	if err = admin.QueryRow(ctx, huellaFunciones).Scan(&funcionesAntes); err != nil || funcionesAntes == "" {
		t.Fatalf("funciones previas ausentes: %v", err)
	}
	const restriccion = `ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT vec_pref_lote_fallo_tercera
 CHECK (audiencia_consumo <> 'vec_usuarios.preferencias.consultar.externa_personal.v1')`
	const quitar = `ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT vec_pref_lote_fallo_tercera`
	if _, err = admin.Exec(ctx, restriccion); err != nil {
		t.Fatalf("no se pudo provocar rechazo de tercera audiencia: %v", err)
	}
	defer func() {
		limpiar, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelar()
		_, _ = admin.Exec(limpiar, quitar)
	}()
	if _, err = publicarMaterialPreferenciasUsuariosEnLote(ctx, gobierno, material, relojContratacionTemporalDesarrollo{}, catalogo); err == nil {
		t.Fatal("tercera publicación rechazada aceptó lote")
	}
	var trasFallo int
	if err = admin.QueryRow(ctx, contarUsuarios).Scan(&trasFallo); err != nil || trasFallo != 0 {
		t.Fatalf("ROLLBACK dejó %d claves Usuarios: %v", trasFallo, err)
	}
	if err = admin.QueryRow(ctx, huellaFunciones).Scan(&funcionesDespues); err != nil || funcionesDespues != funcionesAntes {
		t.Fatalf("fallo cambió funciones anteriores: %v", err)
	}
	if _, err = admin.Exec(ctx, quitar); err != nil {
		t.Fatalf("retirar restricción efímera: %v", err)
	}
	proveedores, err := publicarMaterialPreferenciasUsuariosEnLote(ctx, gobierno, material, relojContratacionTemporalDesarrollo{}, catalogo)
	if err != nil {
		t.Fatalf("cuatro publicaciones atómicas: %v", err)
	}
	for i, p := range proveedores {
		if p == nil {
			t.Fatalf("proveedor %d no construido tras COMMIT", i)
		}
	}
	var confirmadas int
	if err = admin.QueryRow(ctx, contarUsuarios).Scan(&confirmadas); err != nil || confirmadas != 4 {
		t.Fatalf("COMMIT confirmó %d claves Usuarios: %v", confirmadas, err)
	}
	filas, err := admin.Query(ctx, `SELECT audiencia_consumo,count(*) FROM vec_autorizacion_atestada_v3.clave_capacidad_version
 WHERE audiencia_consumo LIKE 'vec_usuarios.preferencias.%' GROUP BY audiencia_consumo`)
	if err != nil {
		t.Fatal(err)
	}
	observadas := map[string]int64{}
	for filas.Next() {
		var audiencia string
		var n int64
		if err = filas.Scan(&audiencia, &n); err != nil {
			filas.Close()
			t.Fatal(err)
		}
		observadas[audiencia] = n
	}
	err = filas.Err()
	filas.Close()
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range descriptoresMaterialPreferenciasUsuariosDesarrollo() {
		if observadas[d.Audiencia] != 1 {
			t.Fatalf("audiencia %s publicada %d veces", d.Audiencia, observadas[d.Audiencia])
		}
	}
	if err = admin.QueryRow(ctx, huellaFunciones).Scan(&funcionesDespues); err != nil || funcionesDespues != funcionesAntes {
		t.Fatalf("éxito alteró funciones previas: %v", err)
	}
}

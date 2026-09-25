package internactproveedores

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// El binario exige el estado exacto posterior a AD3-69: cuatro funciones del
// preflight, ni una más ni una menos. aprovisionar.py debe cotejar lo mismo.
func TestManifiestoPreflightExigeExactamenteV1YV2(t *testing.T) {
	esperadas := []string{
		"vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(jsonb)",
		"vec_autorizacion_atestada_v3.leer_configuracion_interna_v1(jsonb)",
		"vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2(text,jsonb)",
		"vec_autorizacion_atestada_v3.leer_configuracion_interna_v2(text,jsonb)",
	}
	obtenidas := funcionesEsperadasPerfil(perfilPool{rol: "vec_autorizacion_atestada_v3_preflight_interno"})
	if !slices.Equal(obtenidas, esperadas) {
		t.Fatalf("manifiesto del preflight: %v", obtenidas)
	}
	fuente, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "deploy", "principal", "composicion_interna", "aprovisionar.py"))
	if err != nil {
		t.Fatal(err)
	}
	bloque := regexp.MustCompile(`(?s)"gobierno_v3": \[(.*?)\]`).FindSubmatch(fuente)
	if bloque == nil {
		t.Fatal("aprovisionar.py sin lista gobierno_v3")
	}
	var python []string
	for _, m := range regexp.MustCompile(`"([^"]+)"`).FindAllSubmatch(bloque[1], -1) {
		python = append(python, string(m[1]))
	}
	if !slices.Equal(python, esperadas) {
		t.Fatalf("aprovisionar.py y el binario discrepan: %v", python)
	}
	// La función que acredita el pool (privilegio EXECUTE, sin llamarla) es la
	// lectura v1, presente en ambos manifiestos mientras siga instalada.
	if !slices.Contains(esperadas, perfilPreflight.funcion) {
		t.Fatal("la función acreditada del pool no está en el manifiesto")
	}
}

// CT y B2 leen el gobierno con las v2 de AD3-69 y consumidor literal: ningún
// camino de lectura usa v1, cuya comparación revision_gobierno/checkpoint
// rechazaba claves recién publicadas (consenso B2 R3–R5).
func TestLecturaGobiernoCTUsaV2ConConsumidorLiteral(t *testing.T) {
	if sqlLeerConfiguracionCT != `SELECT vec_autorizacion_atestada_v3.leer_configuracion_interna_v2('ct',$1::jsonb)` {
		t.Fatalf("lectura CT: %s", sqlLeerConfiguracionCT)
	}
	for _, sql := range []string{sqlLeerConfiguracionCT, sqlComprobarMaterialB2, sqlLeerConfiguracionB2} {
		if strings.Contains(sql, "_v1(") || !strings.Contains(sql, "_interna_v2('") {
			t.Fatalf("lectura de gobierno fuera de v2 con consumidor literal: %s", sql)
		}
	}
}

func TestTLSVerificadoRechazaRutaDeFallbackSinVerificacion(t *testing.T) {
	cfg := &pgconn.Config{Host: "db.interno", TLSConfig: &tls.Config{ServerName: "db.interno", MinVersion: tls.VersionTLS12}}
	if !tlsVerificado(cfg) {
		t.Fatal("configuracion TLS verificada rechazada")
	}
	cfg.Fallbacks = []*pgconn.FallbackConfig{{Host: "db.interno"}}
	if tlsVerificado(cfg) {
		t.Fatal("fallback sin TLS admitido")
	}
	cfg.Fallbacks[0].TLSConfig = &tls.Config{ServerName: "db.interno", InsecureSkipVerify: true}
	if tlsVerificado(cfg) {
		t.Fatal("fallback sin verificacion admitido")
	}
	cfg.Fallbacks = nil
	cfg.TLSConfig.ServerName = "otro.interno"
	if tlsVerificado(cfg) {
		t.Fatal("nombre TLS distinto admitido")
	}
}

// Consulta exacta de vec-interno para CT contra un gobierno publicado por el
// publicador real de vec-server sin tocar el checkpoint (lo prepara
// cmd/vec-preparar-material-interno/probar_postgresql_pg18.sh, fase
// «publicar»). La v2('ct') acepta de inmediato; la v1 anterior lo rechazaba.
func TestLecturaCTV2ContraGobiernoPublicadoPostgreSQL18(t *testing.T) {
	ruta, dsn := os.Getenv("VEC_AD369_MATERIAL_CT"), os.Getenv("VEC_T4_PG_PREFLIGHT_DSN")
	if os.Getenv("VEC_T4_PG_DESECHABLE") != "si" || ruta == "" || dsn == "" {
		t.Skip("requiere PostgreSQL 18.4 desechable (probar_postgresql_pg18.sh)")
	}
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	var caso struct {
		Material json.RawMessage `json:"material"`
		Esperada struct {
			Revision  string `json:"revision"`
			Secuencia uint64 `json:"secuencia"`
		} `json:"esperada"`
	}
	if err := json.Unmarshal(contenido, &caso); err != nil || caso.Esperada.Revision == "" {
		t.Fatal("material CT del ensayo ilegible")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal("preflight no disponible")
	}
	defer pool.Close()
	respuesta, err := consultaLecturaGobiernoV3(pool, sqlLeerConfiguracionCT)(ctx, caso.Material)
	if err != nil {
		t.Fatal("vec-interno CT rechazó claves recién publicadas con v2('ct')")
	}
	var leida struct {
		Revision  string `json:"revision"`
		Secuencia uint64 `json:"secuencia"`
	}
	if err := json.Unmarshal(respuesta, &leida); err != nil || leida != caso.Esperada {
		t.Fatalf("configuración leída %+v", leida)
	}
	const v1 = `SELECT vec_autorizacion_atestada_v3.leer_configuracion_interna_v1($1::jsonb)`
	if _, err := consultaLecturaGobiernoV3(pool, v1)(ctx, caso.Material); err == nil {
		t.Fatal("v1 aceptó CT con el checkpoint por debajo: el ensayo no reproduce el desfase")
	}
}

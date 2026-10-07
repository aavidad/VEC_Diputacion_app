package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	postgresqlcompartido "vec-diputacion-granada/internal/shared/postgresql"
)

// Las fábricas de pools acreditados O4-05 vuelven a validar pool.Config() en
// cada préstamo y exigen, entre otras cosas, que no haya trazador. Esta prueba
// fija que la configuración del pool ya creado sigue siendo válida: cualquier
// instrumentación añadida en la composición (por ejemplo, un trazador de
// telemetría) haría fallar BeginTx con «consulta RRHH no disponible».
func TestFabricasPoolO405ConservanConfiguracionAcreditada(t *testing.T) {
	t.Parallel()
	const socket = "postgres:///?host=/tmp/vec-ct-o405-socket-inexistente&port=5432&sslmode=disable&user="
	modo := modoTLSAcreditacionPoolO405SocketUnixPrueba

	const login = "vec_ct_rrhh_prueba_01"
	configConsultas, err := pgxpool.ParseConfig(socket + login)
	if err != nil || !endurecerTLSFabricaPoolO405(&configConsultas.ConnConfig.Config, modo) {
		t.Fatalf("configuración de consultas inválida: %v", err)
	}
	postgresqlcompartido.FijarTamanoPool(configConsultas, socket+login, 4)
	if configConsultas.ConnConfig.Tracer != nil || !configuracionPoolAcreditacionO405Valida(configConsultas, modo) {
		t.Fatal("la configuración de consultas RRHH perdió la acreditación O4-05")
	}

	configuracion, err := pgxpool.ParseConfig(socket + loginResolutorMotivosRRHHPrueba)
	if err != nil || !endurecerTLSFabricaPoolO405(&configuracion.ConnConfig.Config, modo) ||
		!configuracionPoolAcreditacionO405Valida(configuracion, modo) {
		t.Fatal("configuración de partida no acreditada")
	}
	if configuracion.ConnConfig.Tracer != nil {
		t.Fatal("la configuración de resolución de motivos añadió trazador")
	}
	if consultas, err := nuevoPoolConsultasRRHHPostgreSQL(context.Background(), socket+login, login, modo); consultas != nil || err == nil {
		t.Fatal("consulta RRHH entregó pool sin preflight")
	}
	if origen, err := crearOrigenPoolResolucionMotivosRRHHPostgreSQL(context.Background(), configuracion); origen != nil || err == nil {
		t.Fatal("resolución de motivos entregó pool sin preflight")
	}
}

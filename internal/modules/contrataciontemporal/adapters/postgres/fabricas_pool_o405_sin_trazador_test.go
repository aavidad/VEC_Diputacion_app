package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
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
	consultas, err := nuevoPoolConsultasRRHHPostgreSQL(context.Background(), socket+login, login, modo)
	if err != nil {
		t.Fatalf("crear pool de consultas: %v", err)
	}
	defer consultas.Cerrar()
	if c := consultas.pool.Config(); c.ConnConfig.Tracer != nil || !configuracionPoolAcreditacionO405Valida(c, modo) {
		t.Fatal("el pool de consultas RRHH perdió la acreditación O4-05")
	}

	configuracion, err := pgxpool.ParseConfig(socket + loginResolutorMotivosRRHHPrueba)
	if err != nil || !endurecerTLSFabricaPoolO405(&configuracion.ConnConfig.Config, modo) ||
		!configuracionPoolAcreditacionO405Valida(configuracion, modo) {
		t.Fatal("configuración de partida no acreditada")
	}
	origen, err := crearOrigenPoolResolucionMotivosRRHHPostgreSQL(context.Background(), configuracion)
	if err != nil {
		t.Fatalf("crear pool de motivos: %v", err)
	}
	defer origen.Cerrar()
	if c := origen.Configuracion(); c.ConnConfig.Tracer != nil || !configuracionPoolAcreditacionO405Valida(c, modo) {
		t.Fatal("el pool de resolución de motivos perdió la acreditación O4-05")
	}
}

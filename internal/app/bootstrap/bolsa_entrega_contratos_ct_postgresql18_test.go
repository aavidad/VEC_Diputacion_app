package bootstrap

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
)

// TestEntregaContratosCTBolsaPostgreSQL18 recorre el relevo con adaptadores
// reales: CT113 con el rol ejecutor de CT y el inbox B13 con el de Bolsa.
// Requiere un PG18.4 desechable con ambas migraciones y al menos una
// incorporación CT cuyo llamamiento conozca Bolsa. Se ejecuta a mano:
// VEC_B13_PG18_CT_DSN y VEC_B13_PG18_BOLSA_DSN con roles de login distintos.
func TestEntregaContratosCTBolsaPostgreSQL18(t *testing.T) {
	dsnCT, dsnBolsa := os.Getenv("VEC_B13_PG18_CT_DSN"), os.Getenv("VEC_B13_PG18_BOLSA_DSN")
	if dsnCT == "" || dsnBolsa == "" {
		t.Skip("solo se ejecuta contra un PostgreSQL 18.4 desechable de B13")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	ct, err := pgxpool.New(ctx, dsnCT)
	if err != nil {
		t.Fatal(err)
	}
	defer ct.Close()
	bolsa, err := pgxpool.New(ctx, dsnBolsa)
	if err != nil {
		t.Fatal(err)
	}
	defer bolsa.Close()
	lector, _ := postgresct.NuevoLectorPublicacionContratosBolsaPostgreSQL(ct)
	buzon, _ := postgresbolsa.NuevoBuzonContratosParticipacionPostgreSQL(bolsa)
	receptor, _ := aplicacionbolsa.NuevoServicioRecepcionContratos(buzon)
	relevo := &entregaContratosCTBolsa{lector: lector, receptor: receptor, lote: 1}
	primera, err := relevo.entregar(ctx)
	if err != nil || primera.nuevos == 0 || primera.rechazados != 0 {
		t.Fatalf("primera pasada=%+v err=%v", primera, err)
	}
	segunda, err := relevo.entregar(ctx)
	// La segunda pasada parte del cursor exacto: nada nuevo ni repetido.
	if err != nil || segunda.nuevos != 0 || segunda.reentregas != 0 || segunda.rechazados != 0 {
		t.Fatalf("segunda pasada=%+v err=%v", segunda, err)
	}
	cursor, hay, err := receptor.Cursor(ctx)
	if err != nil || !hay || cursor.OrigenRef == "" {
		t.Fatalf("cursor=%+v hay=%v err=%v", cursor, hay, err)
	}
	t.Logf("B13 PG18: %d incorporaciones entregadas; la segunda pasada parte del cursor (posición %d)", primera.nuevos, cursor.Posicion)
}

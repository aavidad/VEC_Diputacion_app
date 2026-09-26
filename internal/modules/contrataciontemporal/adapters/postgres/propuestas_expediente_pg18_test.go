package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Prueba de contrato Go↔SQL de la consulta de propuestas de CT128. Solo se
// ejecuta dentro del ensayo desechable (probar_ct128_propuesta_sucesor_pg18.sh
// con VEC_CT128_GO=1), tras la cadena completa: la propuesta de A sustituida
// por su no incorporación y la de B vigente, con GINPIX confirmado.
func TestPropuestasExpedientePostgreSQLContratoGoSQL(t *testing.T) {
	dsn := os.Getenv("VEC_CT128_PG_DSN")
	if dsn == "" {
		t.Skip("solo en el ensayo PostgreSQL 18 desechable de CT128")
	}
	org, exp := os.Getenv("VEC_CT128_ORG"), os.Getenv("VEC_CT128_EXP")
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repo, err := NuevoRepositorioOperacionSeguimientoPostgreSQL(pool)
	if err != nil {
		t.Fatal(err)
	}
	estado, err := repo.ConsultarIncorporacionAcreditada(ctx, org, exp)
	if err != nil {
		t.Fatal(err)
	}
	p := estado.Propuestas
	if len(p) != 2 || p[0].Vigente || p[0].VersionResultante != 7 || p[0].Sustitucion == nil ||
		p[0].Sustitucion.MotivoClave != "no_presentado" || !p[1].Vigente ||
		// La de B sigue a la no incorporación confirmada de A (con cuatro
		// ojos, al menos propuesta y confirmación después de la versión 7).
		p[1].VersionResultante <= p[0].VersionResultante+2 ||
		p[1].Sustitucion != nil || estado.GINPIX == nil || estado.NoIncorporacion == nil {
		t.Fatalf("propuestas: %+v", estado)
	}
	if _, err := repo.ConsultarIncorporacionAcreditada(ctx, "organizacion:otra:ct128", exp); err == nil {
		t.Fatal("consulta con otra organización admitida")
	}
}

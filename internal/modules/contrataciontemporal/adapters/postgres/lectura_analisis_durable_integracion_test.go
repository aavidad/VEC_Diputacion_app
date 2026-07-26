package postgres

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

func TestLectorExpedienteAnalisisDurableO3PostgreSQLReal(
	t *testing.T,
) {
	dsn := os.Getenv("VEC_O4_LECTOR_DSN")
	if dsn == "" {
		t.Skip("PostgreSQL efímero O4 no solicitado")
	}
	version, err := strconv.ParseUint(
		os.Getenv("VEC_O4_LECTOR_VERSION"),
		10,
		64,
	)
	if err != nil {
		t.Fatalf("versión de integración inválida: %v", err)
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	lector, err := NuevoLectorExpedienteAnalisisDurableO3PostgreSQL(pool)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := cobertura.NuevaSolicitudInstantaneaAnalisisDurableO3(
		"organizacion:dipgra",
		"expediente:ct:o205:alta_valida",
		version,
	)
	if err != nil {
		t.Fatal(err)
	}
	instantanea, err := cobertura.ObtenerInstantaneaAnalisisDurableO3(
		ctx,
		lector,
		solicitud,
	)
	if err != nil {
		t.Fatalf("obtener instantánea real: %v", err)
	}
	expediente, reciboRef, huella, err := instantanea.DesplegarPara(solicitud)
	if err != nil ||
		expediente.Version != version ||
		expediente.Referencia != "expediente:ct:o205:alta_valida" ||
		expediente.Analisis == nil ||
		reciboRef == "" ||
		len(huella) != 64 {
		t.Fatalf(
			"instantánea real inesperada: version=%d recibo=%q huella=%q err=%v",
			expediente.Version,
			reciboRef,
			huella,
			err,
		)
	}
}

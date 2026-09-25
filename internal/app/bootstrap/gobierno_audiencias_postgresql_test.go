package bootstrap

import (
	"context"
	"os"
	"testing"
	"time"
)

// Reproduce en PostgreSQL 18.4 desechable (AD3 1/2 reales) la secuencia de
// arranques de vec-server del 25/09: un binario anterior publica la clave base
// CT y las ocho de Cronos empleado; el siguiente arranca con resolución y
// notificaciones de Cronos. Cada consumidor nuevo pasa a ser el último puntero
// de emisión, y el gobierno sigue siendo propio en cada publicación posterior
// y en un reinicio idempotente.
func TestGobiernoCTPublicaConsumidoresNuevosSinVolverseAjenoPostgreSQL18(t *testing.T) {
	if os.Getenv("VEC_GOBV3_PG_DESECHABLE") != "si" {
		t.Skip("requiere PostgreSQL 18.4 desechable con AD3 1/2")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelar()
	gobierno, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, os.Getenv("VEC_GOBV3_PG_DSN"),
		"vec-ct-desarrollo-gobierno", rolGobiernoPostgreSQLContratacionTemporalDesarrollo)
	if err != nil {
		t.Fatalf("pool de gobierno rechazado: %v", err)
	}
	defer gobierno.Close()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	material := materialRenovableCTPrueba(t, ahora.Add(-24*time.Hour))
	if err := publicarGobiernoAtestacionContratacionTemporalDesarrollo(ctx, gobierno, &material); err != nil {
		t.Fatalf("publicación base CT: %v", err)
	}
	anterior := descriptoresMaterialCronosDesarrollo()
	nuevos := append(append(append(append(descriptoresMaterialCronosResolucionDesarrollo(),
		descriptoresMaterialCronosNotificacionesDesarrollo()...),
		descriptoresMaterialDocumentosDesarrollo()...),
		descriptorMaterialFichaPropiaPersonalDesarrollo()),
		descriptorMaterialMiBolsaDesarrollo())
	todos := append(append(descriptoresMaterialAutorizacionContratacionTemporalDesarrollo(), anterior...), nuevos...)
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(todos)
	if err != nil {
		t.Fatal(err)
	}
	publicar := func(etapa string, descriptores []descriptorMaterialConsumidorV3Desarrollo) {
		t.Helper()
		for _, d := range descriptores {
			if _, err := nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material,
				relojContratacionTemporalDesarrollo{}, catalogo, d.Audiencia); err != nil {
				t.Fatalf("%s: publicación de %s: %v", etapa, d.Audiencia, err)
			}
		}
	}
	contar := func() (n int64) {
		t.Helper()
		if err := gobierno.QueryRow(ctx, `SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3.clave_capacidad_version`).Scan(&n); err == nil {
			return n
		}
		// El LOGIN de gobierno no lee sin SET ROLE: se cuenta en una transacción.
		tx, err := gobierno.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(context.Background())
		if _, err := tx.Exec(ctx, `SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario`); err != nil {
			t.Fatal(err)
		}
		if err := tx.QueryRow(ctx, `SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3.clave_capacidad_version`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	publicar("binario anterior", anterior)
	publicar("binario anterior, reinicio", anterior)
	publicar("binario nuevo", append(append([]descriptorMaterialConsumidorV3Desarrollo{}, anterior...), nuevos...))
	tras := contar()
	if want := int64(1 + len(anterior) + len(nuevos)); tras != want {
		t.Fatalf("claves publicadas: %d, esperadas %d", tras, want)
	}
	publicar("binario nuevo, reinicio", append(append([]descriptorMaterialConsumidorV3Desarrollo{}, anterior...), nuevos...))
	if contar() != tras {
		t.Fatal("el reinicio no fue idempotente")
	}
	// La renovación diaria parte del gobierno ya ampliado.
	if _, err := renovarConfiguracionConfianzaCTDesarrollo(ctx, gobierno, material, ahora); err != nil {
		t.Fatalf("renovación tras ampliar consumidores: %v", err)
	}
}

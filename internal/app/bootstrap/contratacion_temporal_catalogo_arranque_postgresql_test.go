package bootstrap

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	postgrescontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// Arranques reales contra PostgreSQL 18 desechable con el volcado sintético
// de la principal y CT-000126 instalada (guion
// probar_ct126_numeracion_y_gobierno_cobertura_pg18.sh). Publica como el
// gobernador real y lee como el ejecutor real. La fase 1 hace dos arranques
// sin catálogo, un cambio de catálogo y su rearranque; tras reiniciar
// PostgreSQL, la fase 2 comprueba que nada se republica y vuelve a las vías
// de siempre.
func TestArranqueCatalogoCoberturaYNumeracionPostgreSQL(t *testing.T) {
	fase := os.Getenv("VEC_CT126_PG_FASE")
	if fase != "1" && fase != "2" {
		t.Skip("requiere PostgreSQL 18 desechable con el volcado sintético y CT-000126")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelar()
	abrir := func(variable string) *pgxpool.Pool {
		t.Helper()
		pool, err := pgxpool.New(ctx, os.Getenv(variable))
		if err != nil {
			t.Fatalf("pool %s no disponible", variable)
		}
		t.Cleanup(pool.Close)
		return pool
	}
	gobierno, ejecucion, administracion := abrir("VEC_CT126_PG_DSN_GOBIERNO"),
		abrir("VEC_CT126_PG_DSN_EJECUCION"), abrir("VEC_CT126_PG_DSN_ADMIN")
	eventos := func() int64 {
		t.Helper()
		var total, checkpoint int64
		if err := administracion.QueryRow(ctx, `SELECT
		  (SELECT count(*) FROM vec_contratacion_temporal.gobi_o404b_evento),
		  (SELECT ultima_secuencia FROM vec_contratacion_temporal.gobi_o404b_checkpoint)`).Scan(&total, &checkpoint); err != nil {
			t.Fatal(err)
		}
		if total != checkpoint {
			t.Fatalf("eventos %d y secuencia %d no coinciden", total, checkpoint)
		}
		return total
	}
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	reloj := relojContratacionTemporalDesarrollo{}
	resolutor, err := postgrescontratacion.NuevoResolutorGobiernoCoberturaO404BPostgreSQL(ejecucion)
	if err != nil {
		t.Fatal(err)
	}
	arrancar := func(opciones *opcionesAnalisisCTDesarrollo) {
		t.Helper()
		soporte.opcionesCatalogo = opciones
		if err := publicarGobiernoCoberturaPostgreSQLContratacionTemporalDesarrollo(ctx, gobierno, soporte); err != nil {
			t.Fatalf("publicaciones fijas: %v", err)
		}
		if err := sincronizarGobiernoCoberturaCatalogoPostgreSQLCT(ctx, gobierno, ejecucion, soporte, reloj); err != nil {
			t.Fatalf("sincronización del catálogo: %v", err)
		}
		if err := publicarNumeracionExpedientesCT(ctx, gobierno, opciones.numeracionVigente()); err != nil {
			t.Fatalf("numeración: %v", err)
		}
	}
	vigente := func(accion domain.ClaveCatalogo) string {
		t.Helper()
		huella, err := huellaActuacionVigenteCoberturaCT(ctx, resolutor, reloj, accion)
		if err != nil {
			t.Fatal(err)
		}
		return huella
	}
	otras := &opcionesAnalisisCTDesarrollo{
		viasCobertura: append(viasCoberturaPredeterminadasCT()[:2:2], viaCoberturaCT{
			Clave: "bolsa_otra_categoria", Procedencia: "bolsa", Etiqueta: "Bolsa de otra categoría",
			Comprobaciones: []domain.ClaveCatalogo{"existe_bolsa_afin"},
		}),
		numeracion: &numeracionExpedientesCT{Prefijo: "CTEMP-", Digitos: 4, FuenteRef: "vec.contratacion_temporal.reglas:1:c16.numeracion"},
	}
	deseadoOtras, err := gobiernoCoberturaDeseadoParaCatalogoCT(soporte, otras.viasCoberturaVigentes())
	if err != nil {
		t.Fatal(err)
	}
	deseadoSiempre, err := gobiernoCoberturaDeseadoParaCatalogoCT(soporte, viasCoberturaPredeterminadasCT())
	if err != nil {
		t.Fatal(err)
	}
	numero := func() string {
		t.Helper()
		var valor string
		if err := ejecucion.QueryRow(ctx, "SELECT vec_contratacion_temporal.siguiente_numero_visible_v1(2033)").Scan(&valor); err != nil {
			t.Fatal(err)
		}
		return valor
	}
	switch fase {
	case "1":
		inicial := eventos()
		for arranque := 1; arranque <= 2; arranque++ {
			arrancar(nil)
			if total := eventos(); total != inicial {
				t.Fatalf("arranque %d sin catálogo publicó %d eventos", arranque, total-inicial)
			}
		}
		if vigente(domain.AccionDecidirCoberturaGobernada) != deseadoSiempre.actuaciones[0].HuellaSHA256 {
			t.Fatal("sin catálogo no rigen las vías de siempre")
		}
		if valor := numero(); valor != "2033/CT-000001" {
			t.Fatalf("formato de siempre alterado: %s", valor)
		}
		arrancar(otras)
		if total := eventos(); total != inicial+2 {
			t.Fatalf("el cambio de catálogo publicó %d eventos", total-inicial)
		}
		arrancar(otras)
		if total := eventos(); total != inicial+2 {
			t.Fatalf("el rearranque con el mismo catálogo publicó %d eventos más", total-inicial-2)
		}
		for indice, accion := range []domain.ClaveCatalogo{domain.AccionDecidirCoberturaGobernada, domain.AccionRectificarCoberturaGobernada} {
			if vigente(accion) != deseadoOtras.actuaciones[indice].HuellaSHA256 {
				t.Fatalf("la acción %s no apunta al catálogo nuevo", accion)
			}
		}
		if valor := numero(); valor != "2033/CTEMP-0002" {
			t.Fatalf("numeración del catálogo no aplicada: %s", valor)
		}
		t.Log("eventos " + strconv.FormatInt(inicial, 10) + " -> " + strconv.FormatInt(eventos(), 10))
	case "2":
		antes := eventos()
		arrancar(otras)
		if total := eventos(); total != antes {
			t.Fatalf("tras reiniciar PostgreSQL se republicó (%d eventos)", total-antes)
		}
		if vigente(domain.AccionRectificarCoberturaGobernada) != deseadoOtras.actuaciones[1].HuellaSHA256 {
			t.Fatal("el puntero no sobrevivió al reinicio")
		}
		if valor := numero(); valor != "2033/CTEMP-0003" {
			t.Fatalf("numeración tras el reinicio: %s", valor)
		}
		arrancar(nil)
		if total := eventos(); total != antes+2 {
			t.Fatalf("volver a las vías de siempre publicó %d eventos", total-antes)
		}
		if vigente(domain.AccionDecidirCoberturaGobernada) != deseadoSiempre.actuaciones[0].HuellaSHA256 {
			t.Fatal("no se volvió a las vías de siempre")
		}
		arrancar(nil)
		if total := eventos(); total != antes+2 {
			t.Fatal("el rearranque sin catálogo republicó")
		}
		if valor := numero(); valor != "2033/CT-000004" {
			t.Fatalf("formato de siempre no restablecido: %s", valor)
		}
	}
}

package bootstrap

import (
	"context"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// Entradas de «motivos_seguimiento_cese_ct» v1 tal como las publicó el
// binario anterior (cese, cierre, modificación y consulta) y tal como están ya
// en las bases de ensayo y en la principal. Una versión publicada no admite
// otras entradas: si este catálogo cambia, el arranque siguiente se detiene
// con «cese, cierre y modificacion de desarrollo no disponibles».
var entradasPublicadasSeguimientoCeseV1 = []string{
	"motivo_63b6c3989d28a30b798bae4aba3ed217",
	"motivo_8906db3719462f51e24b438c13ab3635",
	"motivo_bcaf41e10e5b0c55d8956f07c1803138",
	"motivo_f7a3d187c2a71af63888934b766191ef",
}

func claveCatalogoMotivos(m dominiovec.ReferenciaEntradaCatalogo) string {
	return m.CatalogoID + "\x00" + m.CatalogoHuellaSHA256
}

// Con y sin incorporación acreditada, el catálogo del seguimiento de cese
// conserva exactamente su versión 1 publicada; las rutas de CT124 van en un
// catálogo propio que solo se publica con el selector encendido.
func TestCatalogoMotivosSeguimientoCeseV1Inmutable(t *testing.T) {
	for _, acreditada := range []bool{false, true} {
		grupos := motivosSeguimientoCesePorCatalogo(acreditada)
		var seguimiento, acreditadas []string
		for _, grupo := range grupos {
			for _, m := range grupo {
				if claveCatalogoMotivos(m) != claveCatalogoMotivos(grupo[0]) || m.CatalogoVersion != grupo[0].CatalogoVersion {
					t.Fatalf("grupo con catálogos mezclados: %+v", grupo)
				}
				switch m.CatalogoID {
				case "motivos_seguimiento_cese_ct":
					seguimiento = append(seguimiento, m.EntradaClave)
				case "motivos_incorporacion_acreditada_ct":
					acreditadas = append(acreditadas, m.EntradaClave)
				default:
					t.Fatalf("catálogo inesperado %s", m.CatalogoID)
				}
			}
		}
		slices.Sort(seguimiento)
		if !slices.Equal(seguimiento, entradasPublicadasSeguimientoCeseV1) {
			t.Fatalf("acreditada=%v: motivos_seguimiento_cese_ct v1 cambió: %v", acreditada, seguimiento)
		}
		if acreditada && len(acreditadas) != 2 || !acreditada && len(acreditadas) != 0 {
			t.Fatalf("acreditada=%v: %d motivos de CT124", acreditada, len(acreditadas))
		}
	}
	for _, ruta := range []string{httpinterno.RutaConfirmacionesGINPIX, httpinterno.RutaNoIncorporaciones} {
		if motivoSeguimientoCeseDesarrollo(ruta).CatalogoID != "motivos_incorporacion_acreditada_ct" {
			t.Fatalf("%s no usa el catálogo de CT124", ruta)
		}
	}
}

// Reproduce el fallo del clon de ensayo del 26/09/2026: con
// «motivos_seguimiento_cese_ct» v1 ya publicada por el binario anterior (cuatro
// entradas), el arranque con todos los selectores encendidos (seguimiento de
// cese e incorporación acreditada) publicaba seis entradas en esa misma
// versión y la función SQL rechazaba el replay. Se ejecuta contra PostgreSQL
// 18 desechable con el volcado sintético y AD3-87/88, Bolsa 041/042 y
// CT122…126 instaladas (probar_huecos_rrhh_conjunto_pg18.sh), como el
// gobernador y el ejecutor reales: compone el soporte del seguimiento dos
// veces sin y dos con incorporación acreditada, comprueba las migraciones de
// CT124 y que el catálogo antiguo sigue intacto.
func TestArranqueSeguimientoCeseIncorporacionAcreditadaPostgreSQL(t *testing.T) {
	if os.Getenv("VEC_ARRANQUE_T2_PG_DESECHABLE") != "1" {
		t.Skip("requiere PostgreSQL 18 desechable con el volcado sintético y las migraciones de los huecos de RRHH")
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
	gobierno, ejecucion, administracion := abrir("VEC_ARRANQUE_T2_PG_DSN_GOBIERNO"),
		abrir("VEC_ARRANQUE_T2_PG_DSN_EJECUCION"), abrir("VEC_ARRANQUE_T2_PG_DSN_ADMIN")
	entradas := func(catalogo string) []string {
		t.Helper()
		filas, err := administracion.Query(ctx, `SELECT entrada_clave FROM vec_autorizacion.motivo_v2_entrada
		  WHERE catalogo_id=$1 AND catalogo_version=1 ORDER BY entrada_clave`, catalogo)
		if err != nil {
			t.Fatal(err)
		}
		var claves []string
		for filas.Next() {
			var c string
			if err := filas.Scan(&c); err != nil {
				t.Fatal(err)
			}
			claves = append(claves, c)
		}
		if filas.Err() != nil {
			t.Fatal(filas.Err())
		}
		return claves
	}

	// Estado previo: el binario anterior publicó las cuatro entradas.
	desde, _, _ := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(time.Now())
	var anteriores []dominiovec.ReferenciaEntradaCatalogo
	for _, ruta := range []string{httpinterno.RutaCesesNombramiento, httpinterno.RutaCierresExpediente,
		httpinterno.RutaModificacionesNombramiento, httpinterno.RutaSeguimientoCese} {
		anteriores = append(anteriores, motivoSeguimientoCeseDesarrollo(ruta))
	}
	if err := publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, gobierno, anteriores, desde); err != nil {
		t.Fatalf("publicación del binario anterior: %v", err)
	}
	// La causa del fallo: añadir entradas a la versión ya publicada se rechaza.
	seis := append(slices.Clone(anteriores), dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: anteriores[0].CatalogoID, CatalogoVersion: anteriores[0].CatalogoVersion,
		CatalogoHuellaSHA256: anteriores[0].CatalogoHuellaSHA256,
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "seguimiento-cese-ct:"+httpinterno.RutaConfirmacionesGINPIX),
	})
	if err := publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, gobierno, seis, desde); err == nil {
		t.Fatal("añadir entradas a motivos_seguimiento_cese_ct v1 debía rechazarse")
	}

	if err := comprobarMigracionesIncorporacionAcreditadaDesarrollo(ctx, ejecucion); err != nil {
		t.Fatalf("migraciones de la incorporación acreditada: %v", err)
	}
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	alta := &dependenciasAltaContratacionTemporalDesarrollo{soporte: soporte}
	alta.postgresql.gobierno = gobierno
	for _, acreditada := range []bool{false, false, true, true} {
		if err := configurarSoporteSeguimientoCeseDesarrollo(ctx, alta, relojContratacionTemporalDesarrollo{}, acreditada); err != nil {
			t.Fatalf("arranque con acreditada=%v: %v", acreditada, err)
		}
		if _, valida := soporte.instantaneaSeguimientoCese(); !valida {
			t.Fatalf("instantánea inválida con acreditada=%v", acreditada)
		}
	}
	if got := entradas("motivos_seguimiento_cese_ct"); !slices.Equal(got, entradasPublicadasSeguimientoCeseV1) {
		t.Fatalf("motivos_seguimiento_cese_ct alterado: %v", got)
	}
	if got := entradas("motivos_incorporacion_acreditada_ct"); len(got) != 2 {
		t.Fatalf("motivos de CT124 publicados: %v", got)
	}
}

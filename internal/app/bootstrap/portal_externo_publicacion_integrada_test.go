package bootstrap

import (
	"bytes"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// Una copia desechable de la principal post-HZ37 con AD263/AD265 y los LOGIN
// nominales. No aplica SQL ni cambia infraestructura desde la prueba.
func TestPreparacionConfianzaExternaPostgreSQL(t *testing.T) {
	if os.Getenv("VEC_HZ38_PG_DESECHABLE") != "1" {
		t.Skip("requiere copia PostgreSQL desechable post-HZ37/AD263/AD265")
	}
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	externo := generarMaterialPortalExternoPrueba(t)
	destino := externo.cfg.DevelopmentMaterialDir
	dsn, err := os.ReadFile(os.Getenv("VEC_HZ38_PUBLICADOR_DSN_FICHERO"))
	if err != nil {
		t.Fatal(err)
	}
	defer clear(dsn)
	opciones := OpcionesPreparacionPortalExterno{Destino: destino, Consumidores: append([]string(nil), consumidoresPortalExternoV3...), PublicadorConfianzaDSN: strings.TrimSpace(string(dsn))}
	inicio := time.Now()
	resumen, err := PrepararMaterialPortalExterno(t.Context(), cfg, opciones)
	if err != nil {
		t.Fatalf("preparar: %v", err)
	}
	t.Logf("preparación=%s claves=%d", time.Since(inicio), resumen.Claves)
	inv, err := leerInventarioV3PortalExterno(destino)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(inv.Configuracion.Revision, "confianza:atestacion:externo:") || !strings.HasPrefix(inv.Raiz.ClaveID, "clave:atestacion:externo:") {
		t.Fatal("coordenadas internas en inventario externo")
	}
	antes, err := os.ReadFile(filepath.Join(destino, inventarioMaterialV3PortalExterno))
	if err != nil {
		t.Fatal(err)
	}
	inicio = time.Now()
	repetida, err := PrepararMaterialPortalExterno(t.Context(), cfg, opciones)
	if err != nil {
		t.Fatalf("repetir: %v", err)
	}
	despues, err := os.ReadFile(filepath.Join(destino, inventarioMaterialV3PortalExterno))
	if err != nil {
		t.Fatal(err)
	}
	if resumen.Configuracion != repetida.Configuracion || !bytes.Equal(antes, despues) {
		t.Fatal("replay cambió configuración o inventario")
	}
	t.Logf("replay=%s", time.Since(inicio))
	dsnLectura, err := os.ReadFile(os.Getenv("VEC_HZ38_PREFLIGHT_DSN_FICHERO"))
	if err != nil {
		t.Fatal(err)
	}
	defer clear(dsnLectura)
	pool, err := abrirPoolPreflightV3PortalExterno(t.Context(), strings.TrimSpace(string(dsnLectura)))
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	defer pool.Close()
	for _, consumidor := range opciones.Consumidores {
		inicio = time.Now()
		leida, err := leerConfiguracionV3PortalExterno(t.Context(), pool, inv, consumidor, inv.Configuracion)
		if err != nil {
			t.Fatalf("leer %s: %v", consumidor, err)
		}
		if leida.Revision != inv.Configuracion.Revision || leida.HuellaSHA256 != inv.Configuracion.HuellaSHA256 {
			t.Fatalf("configuración distinta %s", consumidor)
		}
		proveedores, err := nuevosProveedoresV3PortalExterno(t.Context(), destino, pool, consumidor, relojContratacionTemporalDesarrollo{})
		if err != nil {
			t.Fatalf("proveedores %s: %v", consumidor, err)
		}
		if len(proveedores) != len(audienciasConsumidorPortalExternoV3(consumidor)) {
			t.Fatalf("proveedores incompletos %s", consumidor)
		}
		t.Logf("lector + proveedores %s=%s", consumidor, time.Since(inicio))
	}
	// Acortar una publicación por la misma fachada, con las mismas claves,
	// permite comprobar la renovación de un proveedor ya vivo, sin reiniciarlo.
	publicador, err := abrirPoolPublicadorConfianzaExterna(t.Context(), strings.TrimSpace(string(dsn)))
	if err != nil {
		t.Fatal(err)
	}
	defer publicador.Close()
	completos := map[string][]materialAtestacionContratacionTemporalDesarrollo{}
	defer func() {
		for _, m := range completos {
			borrarMaterialesV3PortalExterno(m)
		}
	}()
	for _, c := range consumidoresPortalExternoV3 {
		m, err := materialesConsumidorV3PortalExterno(destino, inv, c, inv.Configuracion)
		if err != nil {
			t.Fatal(err)
		}
		completos[c] = m
	}
	tx, err := publicador.BeginTx(t.Context(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(t.Context())
	var estadoBytes []byte
	if err := tx.QueryRow(t.Context(), `SELECT vec_autorizacion_atestada_v3.gestionar_publicacion_externa_v1(NULL::jsonb)`).Scan(&estadoBytes); err != nil {
		t.Fatal(err)
	}
	var estado estadoPublicacionExternaV3
	if err := json.Unmarshal(estadoBytes, &estado); err != nil {
		t.Fatal(err)
	}
	var actual materialPublicoExternoV3
	if err := json.Unmarshal(estado.Actual, &actual); err != nil {
		t.Fatal(err)
	}
	sort.Slice(actual.Claves, func(i, j int) bool { return actual.Claves[i].Audiencia < actual.Claves[j].Audiencia })
	lista := ordenarMaterialExternoV3(completos)
	expira := time.Now().UTC().Truncate(time.Microsecond).Add(time.Second)
	if err := coordinarMaterialExternoV3(lista, "confianza:atestacion:externo:"+numeroDecimal64(estado.SecuenciaSiguiente), estado.SecuenciaSiguiente,
		time.Now().UTC().Add(-time.Second).Truncate(time.Microsecond), expira, lista[0].validaDesde, lista[0].validaHasta, actual.Claves); err != nil {
		t.Fatal(err)
	}
	propuesta, err := serializarMaterialExternoV3(lista)
	if err != nil {
		t.Fatal(err)
	}
	var publicada []byte
	if err := tx.QueryRow(t.Context(), `SELECT vec_autorizacion_atestada_v3.gestionar_publicacion_externa_v1($1::jsonb)`, propuesta).Scan(&publicada); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	vivos, err := nuevosProveedoresV3PortalExterno(t.Context(), destino, pool, "mi_bolsa", relojContratacionTemporalDesarrollo{})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Until(expira) + time.Millisecond)
	if _, err := PrepararMaterialPortalExterno(t.Context(), cfg, opciones); err != nil {
		t.Fatalf("renovación: %v", err)
	}
	for _, p := range vivos {
		if _, err := p.fuenteConfianza.instantanea(t.Context()); err != nil {
			t.Fatalf("proveedor vivo después de renovar: %v", err)
		}
	}
	renovado, err := leerInventarioV3PortalExterno(destino)
	if err != nil {
		t.Fatal(err)
	}
	for c, claves := range inv.Consumidores {
		for i, k := range claves {
			r := renovado.Consumidores[c][i]
			if k != r {
				t.Fatalf("la renovación cambió clave %s", c)
			}
		}
	}
	if renovado.Configuracion.Secuencia <= inv.Configuracion.Secuencia {
		t.Fatal("la renovación no avanzó configuración")
	}
	t.Log("renovación con las mismas23claves y proveedor vivo: OK")

}

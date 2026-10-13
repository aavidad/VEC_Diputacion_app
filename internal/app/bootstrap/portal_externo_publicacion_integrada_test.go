package bootstrap

import (
	"bytes"
	"os"
	"path/filepath"
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
	opciones := OpcionesPreparacionPortalExterno{Destino: destino, Consumidores: []string{"mi_bolsa", "portal_candidato", "preferencias_bolsas", "inscripcion_bolsa", "meritos_cartera"}, PublicadorConfianzaDSN: strings.TrimSpace(string(dsn))}
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
}

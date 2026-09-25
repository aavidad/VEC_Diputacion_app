package almacen_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/almacen"
	"vec-diputacion-granada/internal/vec/ports"
)

type relojRegistroPrueba struct{}

func (relojRegistroPrueba) Ahora() time.Time { return time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC) }

func TestRegistroSeleccionaFicherosOS3PorConfiguracion(t *testing.T) {
	registro := almacen.NuevoRegistroConectoresAlmacen()
	if err := almacen.RegistrarFicheros(registro, "ficheros-local", relojRegistroPrueba{}); err != nil {
		t.Fatal(err)
	}
	if err := almacen.RegistrarS3Compatible(registro, "s3-documental"); err != nil {
		t.Fatal(err)
	}
	if got := registro.Listar(); !reflect.DeepEqual(got, []string{"ficheros-local", "s3-documental"}) {
		t.Fatalf("conectores: %v", got)
	}
	dir := filepath.Join(t.TempDir(), "originales")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	conector, err := registro.Crear(context.Background(), "ficheros-local", almacen.ConfiguracionConectorAlmacen{
		"directorio": dir, "tamano_maximo": "1048576", "retencion_minima_dias": "3650",
	}, ports.RequisitosAlmacenObjetos{EscrituraEnFlujo: true, LecturaEnFlujo: true, IntegridadSHA256: true, Retencion: true})
	if err != nil || conector == nil {
		t.Fatalf("crear ficheros: %v", err)
	}
	for nombre, cfg := range map[string]almacen.ConfiguracionConectorAlmacen{
		"clave desconocida":  {"directorio": dir, "tamano_maximo": "10", "retencion_minima_dias": "1", "ruta_cliente": "/tmp"},
		"sin retención":      {"directorio": dir, "tamano_maximo": "10"},
		"relativo":           {"directorio": "originales", "tamano_maximo": "10", "retencion_minima_dias": "1"},
		"retención negativa": {"directorio": dir, "tamano_maximo": "10", "retencion_minima_dias": "-1"},
	} {
		if _, err := registro.Crear(context.Background(), "ficheros-local", cfg, ports.RequisitosAlmacenObjetos{}); err == nil {
			t.Fatalf("%s: configuración aceptada", nombre)
		}
	}
	if err := almacen.RegistrarFicheros(registro, "otro", nil); err == nil {
		t.Fatal("sin reloj aceptado")
	}
}

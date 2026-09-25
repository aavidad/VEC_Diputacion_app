package xlsconvoca_test

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	memoria "vec-diputacion-granada/internal/modules/bolsa/adapters/memory"
	"vec-diputacion-granada/internal/modules/bolsa/adapters/xlsconvoca"
	aplicacion "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

// Los libros convoca_v2_* y convoca_v1_hoja_real_* salen del generador de
// paquetes sintéticos (scripts/paquete_ejemplo, --cabeceras convoca y vec).
func TestLectorAceptaFormatoRealConvocaV2YElV1SinRechazos(t *testing.T) {
	casos := []struct {
		fixture string
		esquema dominio.EsquemaExportacion
		formato dominio.FormatoCabeceras
		hoja    string
	}{
		{"convoca_v2_resumen.xls", dominio.EsquemaResumenPersona, dominio.FormatoConvocaV2, "grupo de méritos (Tribunal) (1)"},
		{"convoca_v2_detalle.xls", dominio.EsquemaDetalleMerito, dominio.FormatoConvocaV2, "méritos (1)"},
		{"convoca_v1_hoja_real_resumen.xls", dominio.EsquemaResumenPersona, dominio.FormatoConvocaV1, "grupo de méritos (Tribunal) (1)"},
		{"convoca_v1_hoja_real_detalle.xls", dominio.EsquemaDetalleMerito, dominio.FormatoConvocaV1, "méritos (1)"},
		{"convoca_v2_nfd.xls", dominio.EsquemaResumenPersona, dominio.FormatoConvocaV2, "grupo de méritos (Tribunal) (1)"},
	}
	for _, caso := range casos {
		t.Run(caso.fixture, func(t *testing.T) {
			hoja, staging := decodificarYValidar(t, leerFixture(t, caso.fixture))
			if _, formato, err := dominio.DetectarFormato(hoja.Cabeceras); err != nil ||
				hoja.Esquema != caso.esquema || formato != caso.formato || hoja.NombreHoja != caso.hoja {
				t.Fatalf("formato inesperado: %s %s %q %v", hoja.Esquema, formato, hoja.NombreHoja, err)
			}
			if staging.Rechazadas != 0 || len(staging.Incidencias) != 0 || len(staging.Aceptadas) == 0 ||
				len(staging.Aceptadas) != staging.FilasLeidas {
				t.Fatalf("rechazos inesperados: %#v", staging)
			}
			for _, fila := range staging.Aceptadas {
				if !strings.HasPrefix(fila.Identidad.Documento, "***") {
					t.Fatalf("documento enmascarado no mapeado: %#v", fila.Identidad)
				}
			}
		})
	}
}

func TestLectorRechazaHojaYFormatoConvocaNoAcreditados(t *testing.T) {
	lector := xlsconvoca.NuevoLector()
	for _, fixture := range []string{
		"convoca_v2_hoja_sufijo_2.xls",
		"convoca_v2_hoja_ajena.xls",
		"convoca_v2_hoja_cruzada.xls",
		"convoca_v1_hoja_cruzada.xls",
		"convoca_mezcla_formatos.xls",
	} {
		t.Run(fixture, func(t *testing.T) {
			hoja, err := lector.Decodificar(context.Background(), bytes.NewReader(leerFixture(t, fixture)))
			if !errors.Is(err, dominio.ErrEsquemaExportacionDesconocido) || hoja.Esquema != "" || hoja.Filas != nil {
				t.Fatalf("exportacion no acreditada aceptada: %v %#v", err, hoja)
			}
		})
	}
}

func TestVerticalConvocaV2ImportaActaSinRechazos(t *testing.T) {
	contenido := leerFixture(t, "convoca_v2_resumen.xls")
	repositorio := memoria.NuevoRepositorioImportacionesConvoca()
	servicio, err := aplicacion.NuevoServicio(
		xlsconvoca.NuevoLector(), repositorio,
		func() time.Time { return time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatalf("componer vertical: %v", err)
	}
	resultado, err := servicio.Importar(context.Background(), aplicacion.SolicitudImportacion{
		CategoriaRef: "categoria:rpt:administrativo", BolsaRef: "bolsa:administrativo:2026-09-25",
		NombreFichero: "resumen-real-sintetico.xls", FicheroCustodiadoRef: "almacen:objeto:convoca:fixture-v2",
		ActorRef: "actor:rrhh:fixture-v2", Contenido: contenido,
	})
	if err != nil {
		t.Fatalf("importar v2: %v", err)
	}
	if resultado.Acta.Esquema != dominio.EsquemaResumenPersona || resultado.Acta.FilasRechazadas != 0 ||
		resultado.Acta.FilasAceptadas == 0 || resultado.Acta.FilasAceptadas != resultado.Acta.FilasLeidas {
		t.Fatalf("acta v2 inesperada: %#v", resultado.Acta)
	}
}

// Recorre un paquete completo del generador cuando se indica su directorio:
// VEC_CONVOCA_PAQUETE_DIR=<salida>/convoca go test ./internal/modules/bolsa/adapters/xlsconvoca/
func TestLectorPaqueteGeneradoCompletoSinRechazos(t *testing.T) {
	raiz := os.Getenv("VEC_CONVOCA_PAQUETE_DIR")
	if raiz == "" {
		t.Skip("VEC_CONVOCA_PAQUETE_DIR no definido")
	}
	ficheros := 0
	err := filepath.WalkDir(raiz, func(ruta string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(ruta) != ".xls" {
			return err
		}
		contenido, err := os.ReadFile(ruta)
		if err != nil {
			return err
		}
		ficheros++
		_, staging := decodificarYValidar(t, contenido)
		if staging.Rechazadas != 0 || len(staging.Aceptadas) == 0 {
			t.Errorf("%s: aceptadas=%d rechazadas=%d incidencias=%#v",
				ruta, len(staging.Aceptadas), staging.Rechazadas, staging.Incidencias)
		}
		return nil
	})
	if err != nil || ficheros == 0 {
		t.Fatalf("recorrer paquete: ficheros=%d error=%v", ficheros, err)
	}
}

func decodificarYValidar(t *testing.T, contenido []byte) (dominio.HojaStaging, dominio.ResultadoStaging) {
	t.Helper()
	hoja, err := xlsconvoca.NuevoLector().Decodificar(context.Background(), bytes.NewReader(contenido))
	if err != nil {
		t.Fatalf("decodificar: %v", err)
	}
	staging, err := dominio.ValidarHoja(hoja)
	if err != nil {
		t.Fatalf("validar staging: %v", err)
	}
	return hoja, staging
}

package xlsconvoca_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
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

func TestLectorDetectaLosDosEsquemasRealesConFixturesSinteticos(t *testing.T) {
	casos := []struct {
		fixture    string
		esquema    dominio.EsquemaExportacion
		aceptadas  int
		rechazadas int
	}{
		{"resumen.xls", dominio.EsquemaResumenPersona, 2, 1},
		{"detalle.xls", dominio.EsquemaDetalleMerito, 2, 1},
	}
	lector := xlsconvoca.NuevoLector()
	for _, caso := range casos {
		t.Run(caso.fixture, func(t *testing.T) {
			contenido := leerFixture(t, caso.fixture)
			hoja, err := lector.Decodificar(context.Background(), bytes.NewReader(contenido))
			if err != nil {
				t.Fatalf("decodificar: %v", err)
			}
			if hoja.Esquema != caso.esquema || len(hoja.Cabeceras) != caso.esquema.NumeroColumnas() {
				t.Fatalf("esquema inesperado: %s %#v", hoja.Esquema, hoja.Cabeceras)
			}
			staging, err := dominio.ValidarHoja(hoja)
			if err != nil {
				t.Fatalf("validar staging: %v", err)
			}
			if len(staging.Aceptadas) != caso.aceptadas || staging.Rechazadas != caso.rechazadas {
				t.Fatalf("resultado inesperado: aceptadas=%d rechazadas=%d incidencias=%#v",
					len(staging.Aceptadas), staging.Rechazadas, staging.Incidencias)
			}
		})
	}
}

func TestLectorConservaTipoFormulaParaQueStagingLaRechace(t *testing.T) {
	contenido := leerFixture(t, "formula.xls")
	hoja, err := xlsconvoca.NuevoLector().Decodificar(context.Background(), bytes.NewReader(contenido))
	if err != nil {
		t.Fatalf("decodificar formula: %v", err)
	}
	staging, err := dominio.ValidarHoja(hoja)
	if err != nil {
		t.Fatalf("validar formula: %v", err)
	}
	if len(staging.Aceptadas) != 0 || staging.Rechazadas != 1 ||
		!hayIncidencia(staging.Incidencias, 2, "Primer Apellido", "formula_prohibida") {
		t.Fatalf("formula no rechazada: %#v", staging)
	}
}

func TestLectorRechazaCabeceraDesconocidaFormatoYLimites(t *testing.T) {
	lector := xlsconvoca.NuevoLector()
	contenido := leerFixture(t, "cabecera_desconocida.xls")
	if _, err := lector.Decodificar(context.Background(), bytes.NewReader(contenido)); !errors.Is(err, dominio.ErrEsquemaExportacionDesconocido) {
		t.Fatalf("cabecera desconocida aceptada: %v", err)
	}
	for nombre, datos := range map[string][]byte{
		"texto":    []byte("esto no es un XLS"),
		"truncado": {0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1},
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := lector.Decodificar(context.Background(), bytes.NewReader(datos)); !errors.Is(err, xlsconvoca.ErrXLSInvalido) {
				t.Fatalf("formato hostil aceptado: %v", err)
			}
		})
	}
	demasiadoGrande := make([]byte, aplicacion.MaximoBytesExportacion+1)
	copy(demasiadoGrande, []byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1})
	if _, err := lector.Decodificar(context.Background(), bytes.NewReader(demasiadoGrande)); !errors.Is(err, xlsconvoca.ErrLimiteXLSExcedido) {
		t.Fatalf("limite no aplicado: %v", err)
	}
}

func TestVerticalRealXLSStagingActaEIdempotencia(t *testing.T) {
	contenido := leerFixture(t, "resumen.xls")
	repositorio := memoria.NuevoRepositorioImportacionesConvoca()
	servicio, err := aplicacion.NuevoServicio(
		xlsconvoca.NuevoLector(), repositorio,
		func() time.Time { return time.Date(2026, 7, 18, 14, 0, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatalf("componer vertical: %v", err)
	}
	solicitud := aplicacion.SolicitudImportacion{
		NombreFichero: "resumen-sintetico.xls",
		ActorRef:      "actor:rrhh:fixture-t17", Contenido: contenido,
	}
	primero, err := servicio.Importar(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("importar XLS: %v", err)
	}
	segundo, err := servicio.Importar(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("reimportar XLS: %v", err)
	}
	suma := sha256.Sum256(contenido)
	huella := hex.EncodeToString(suma[:])
	if primero.Reutilizada || !segundo.Reutilizada || repositorio.NumeroLotes() != 1 ||
		primero.Acta.HuellaFicheroSHA256 != huella || primero.Acta.FilasAceptadas != 2 ||
		primero.Acta.FilasRechazadas != 1 || primero.Acta.FilasLeidas != 3 ||
		primero.Acta.Procedencia.HabilitaActosConEfectos ||
		!primero.Acta.Procedencia.RequiereConfirmacionRegistro ||
		primero.Acta.Procedencia.UsoPuntosAutobaremacion != dominio.UsoAutobaremoHistorico {
		t.Fatalf("vertical incompleta: primero=%#v segundo=%#v lotes=%d",
			primero, segundo, repositorio.NumeroLotes())
	}
	lote, existe, err := repositorio.ObtenerPorHuella(context.Background(), huella)
	if err != nil || !existe || len(lote.Aceptadas) != 2 || lote.Aceptadas[0].Identidad.Documento != "***0001**" {
		t.Fatalf("staging no persistido: existe=%v error=%v lote=%#v", existe, err, lote)
	}
	acta := fmt.Sprint(primero.Acta.Incidencias)
	for _, valorFila := range []string{"Carla", "Prueba", "***0003**", "99"} {
		if strings.Contains(acta, valorFila) {
			t.Fatalf("acta filtra valor de fila rechazada %q: %s", valorFila, acta)
		}
	}
}

func leerFixture(t *testing.T, nombre string) []byte {
	t.Helper()
	ruta := filepath.Join("testdata", "xls_sinteticos", nombre)
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("leer fixture %s: %v", ruta, err)
	}
	return contenido
}

func hayIncidencia(incidencias []dominio.Incidencia, fila int, campo, codigo string) bool {
	for _, actual := range incidencias {
		if actual.Fila == fila && actual.Campo == campo && actual.Codigo == codigo {
			return true
		}
	}
	return false
}

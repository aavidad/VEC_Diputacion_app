package rptpublica

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
)

func TestFuenteProyectaSoloCamposPublicosYExigeHuella(t *testing.T) {
	fuente, err := NuevaFuente(filepath.Join("..", "..", "..", "..", "..", "data", "catalogos", "rpt", "v1.rpt-2026.json"))
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := fuente.ObtenerRPTPublica(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if catalogo.Fuente.HuellaSHA256 != HuellaRPT2026 || len(catalogo.Categorias) != 145 || len(catalogo.Puestos) != 842 || catalogo.Resumen != (domain.ResumenRPTPublica{Puestos: 842, Dotacion: 1714, Categorias: 145, Centros: 41}) {
		t.Fatalf("proyeccion inesperada: %+v", catalogo.Fuente)
	}
	var secretaria domain.PuestoRPTPublico
	for _, puesto := range catalogo.Puestos {
		if puesto.Codigo == "430-101-001" {
			secretaria = puesto
			break
		}
	}
	if secretaria.Denominacion != "SECRETARIA DE GRUPO" || secretaria.Centro != "GABINETE DE PRESIDENCIA" || secretaria.Dotacion != 3 || secretaria.Grupos == nil {
		t.Fatalf("puesto publico incompleto: %+v", secretaria)
	}
	primera := catalogo.Categorias[0]
	if primera.Clave == "" || primera.Denominacion == "" || len(primera.Grupos) == 0 || len(primera.Escalas) == 0 {
		t.Fatalf("fila publica incompleta: %+v", primera)
	}
	var sinEscalas domain.CategoriaRPTPublica
	for _, categoria := range catalogo.Categorias {
		if categoria.Clave == "aux-tec-sup-informatica" {
			sinEscalas = categoria
			break
		}
	}
	if sinEscalas.Escalas == nil || len(sinEscalas.Escalas) != 0 {
		t.Fatalf("escalas vacías no normalizadas: %#v", sinEscalas.Escalas)
	}
	serializada, err := json.Marshal(sinEscalas)
	if err != nil || string(serializada) == "" || string(serializada) == "null" || !strings.Contains(string(serializada), `"escalas":[]`) {
		t.Fatalf("salida JSON de escalas vacías = %q, %v", serializada, err)
	}
	falsa, err := NuevaFuente(filepath.Join(t.TempDir(), "rpt.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := falsa.ObtenerRPTPublica(context.Background()); !errors.Is(err, domain.ErrRPTPublicaNoDisponible) {
		t.Fatalf("fuente ausente = %v", err)
	}
}

func TestFuenteRechazaTamanoYTipoNoRegular(t *testing.T) {
	directorio := t.TempDir()
	sobredimensionada := filepath.Join(directorio, "demasiado-grande.json")
	if err := os.WriteFile(sobredimensionada, make([]byte, (8<<20)+1), 0o600); err != nil {
		t.Fatal(err)
	}
	fuente, err := NuevaFuente(sobredimensionada)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fuente.ObtenerRPTPublica(context.Background()); !errors.Is(err, domain.ErrRPTPublicaNoDisponible) {
		t.Fatalf("fuente sobredimensionada = %v", err)
	}

	destinoNoRegular := filepath.Join(directorio, "directorio")
	if err := os.Mkdir(destinoNoRegular, 0o700); err != nil {
		t.Fatal(err)
	}
	enlace := filepath.Join(directorio, "enlace-no-regular.json")
	if err := os.Symlink(destinoNoRegular, enlace); err != nil {
		t.Skipf("no se puede crear enlace simbolico: %v", err)
	}
	fuente, err = NuevaFuente(enlace)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fuente.ObtenerRPTPublica(context.Background()); !errors.Is(err, domain.ErrRPTPublicaNoDisponible) {
		t.Fatalf("enlace a tipo no regular = %v", err)
	}
}

func TestFuenteConservaEnMemoriaElCatalogoValidadoMientrasNoCambieElFichero(t *testing.T) {
	original, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "data", "catalogos", "rpt", "v1.rpt-2026.json"))
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "rpt.json")
	if err := os.WriteFile(ruta, original, 0o600); err != nil {
		t.Fatal(err)
	}
	fuente, err := NuevaFuente(ruta)
	if err != nil {
		t.Fatal(err)
	}
	primero, err := fuente.ObtenerRPTPublica(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(ruta)
	if err != nil {
		t.Fatal(err)
	}
	// Mismo tamaño y misma fecha con otro contenido: si se releyera, la huella
	// fallaría. Que siga respondiendo prueba que no se vuelve a leer.
	if err := os.WriteFile(ruta, make([]byte, len(original)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(ruta, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	segundo, err := fuente.ObtenerRPTPublica(context.Background())
	if err != nil {
		t.Fatalf("catálogo en memoria = %v", err)
	}
	if len(segundo.Puestos) != len(primero.Puestos) || segundo.Fuente != primero.Fuente {
		t.Fatalf("copia en memoria distinta")
	}
	// Cada consulta recibe su copia: modificarla no altera la memoria.
	segundo.Puestos[0].Denominacion = "ALTERADA"
	segundo.Categorias[0].Grupos[0] = "ALTERADO"
	tercero, err := fuente.ObtenerRPTPublica(context.Background())
	if err != nil || tercero.Puestos[0].Denominacion != primero.Puestos[0].Denominacion ||
		tercero.Categorias[0].Grupos[0] != primero.Categorias[0].Grupos[0] {
		t.Fatalf("la memoria compartió datos mutables: %v", err)
	}
	// Otra fecha de modificación obliga a validar de nuevo: el contenido falso
	// ya no se acepta.
	otra := info.ModTime().Add(time.Second)
	if err := os.Chtimes(ruta, otra, otra); err != nil {
		t.Fatal(err)
	}
	if _, err := fuente.ObtenerRPTPublica(context.Background()); !errors.Is(err, domain.ErrRPTPublicaNoDisponible) {
		t.Fatalf("fichero cambiado = %v", err)
	}
	// Retirada la fuente, falla cerrada aunque hubiera memoria.
	if err := os.Remove(ruta); err != nil {
		t.Fatal(err)
	}
	if _, err := fuente.ObtenerRPTPublica(context.Background()); !errors.Is(err, domain.ErrRPTPublicaNoDisponible) {
		t.Fatalf("fuente retirada = %v", err)
	}
}

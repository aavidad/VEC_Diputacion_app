package gobiernoreglasbaremo

import (
	"bufio"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestArquitecturaNoAcoplaGobiernoAInfraestructura(t *testing.T) {
	_, actual, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no se pudo localizar el paquete")
	}
	directorio := filepath.Dir(actual)
	archivos, err := filepath.Glob(filepath.Join(directorio, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, archivo := range archivos {
		if strings.HasSuffix(archivo, "_test.go") {
			continue
		}
		fuente, err := parser.ParseFile(token.NewFileSet(), archivo, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parsear %s: %v", archivo, err)
		}
		for _, declaracion := range fuente.Imports {
			ruta, err := strconv.Unquote(declaracion.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(ruta, "/adapters/") ||
				strings.Contains(ruta, "/adapters") ||
				strings.HasSuffix(ruta, "/ports") ||
				ruta == "database/sql" || ruta == "net/http" ||
				strings.Contains(ruta, "postgres") ||
				strings.Contains(ruta, "pgx") {
				t.Fatalf("dependencia de infraestructura en %s: %s", filepath.Base(archivo), ruta)
			}
		}
		if lineasArchivo(t, archivo) > 800 {
			t.Fatalf("%s supera 800 lineas", filepath.Base(archivo))
		}
	}
}

func lineasArchivo(t *testing.T, ruta string) int {
	t.Helper()
	archivo, err := os.Open(ruta)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = archivo.Close() }()
	escaner := bufio.NewScanner(archivo)
	lineas := 0
	for escaner.Scan() {
		lineas++
	}
	if err := escaner.Err(); err != nil {
		t.Fatal(err)
	}
	return lineas
}

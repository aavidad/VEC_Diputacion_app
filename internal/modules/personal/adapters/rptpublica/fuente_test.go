package rptpublica

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
		if puesto.Codigo == "430-101-001" { secretaria = puesto; break }
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

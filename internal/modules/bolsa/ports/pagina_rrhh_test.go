package ports

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestFiltroPaginaCandidatosRRHHConservaEstadosYCorte(t *testing.T) {
	base := FiltroPaginaCandidatosRRHH{BolsaRef: "bolsa:constituida:prueba", Limite: 100, Corte: time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)}
	for _, estado := range []string{"", "disponible", "no_disponible", "trabajando", "pendiente_incorporacion", "renuncia", "excluido", "disponible_desde", "en_revision"} {
		f := base
		f.Estado = estado
		if err := f.Validar(); err != nil {
			t.Fatalf("estado vigente %q rechazado: %v", estado, err)
		}
	}
	for nombre, alterar := range map[string]func(*FiltroPaginaCandidatosRRHH){
		"estado ajeno": func(f *FiltroPaginaCandidatosRRHH) { f.Estado = "renuncia_rrhh" },
		"limite":       func(f *FiltroPaginaCandidatosRRHH) { f.Limite = 101 },
		"texto":        func(f *FiltroPaginaCandidatosRRHH) { f.Texto = strings.Repeat("a", 101) },
		"cursor":       func(f *FiltroPaginaCandidatosRRHH) { f.Cursor = "\n" },
		"bolsa":        func(f *FiltroPaginaCandidatosRRHH) { f.BolsaRef = "bolsa/otra" },
		"corte":        func(f *FiltroPaginaCandidatosRRHH) { f.Corte = f.Corte.In(time.FixedZone("local", 3600)) },
	} {
		f := base
		alterar(&f)
		if err := f.Validar(); !errors.Is(err, ErrPaginaCandidatosRRHHInvalida) {
			t.Fatalf("%s admitido: %v", nombre, err)
		}
	}
}

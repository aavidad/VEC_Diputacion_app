package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestPoliticaSubsanacionExigeMotivoCompatibleConPublicacion(t *testing.T) {
	politica := configuracionPoliticaSubsanacionReparosDesarrollo{
		DefinicionRef: "politica:subsanacion:sintetica", DefinicionVersion: 1,
		DefinicionHuellaSHA256: strings.Repeat("a", 64),
		MotivoAutorizacion: vecdomain.ReferenciaEntradaCatalogo{
			CatalogoID: "motivos_subsanacion_prueba", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("b", 64),
			EntradaClave:         "motivo_" + strings.Repeat("c", 32),
		},
	}
	for _, caso := range []struct {
		nombre, clave string
		valida        bool
	}{
		{"motivo publicable", politica.MotivoAutorizacion.EntradaClave, true},
		{"clave generica no publicable", "subsanacion_sintetica", false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			candidata := politica
			candidata.MotivoAutorizacion.EntradaClave = caso.clave
			if err := candidata.MotivoAutorizacion.Validar(); err != nil {
				t.Fatal(err)
			}
			contenido, err := json.Marshal(candidata)
			if err != nil {
				t.Fatal(err)
			}
			ruta := filepath.Join(t.TempDir(), "politica.json")
			if err := os.WriteFile(ruta, contenido, 0600); err != nil {
				t.Fatal(err)
			}
			recibida, err := cargarConfiguracionPoliticaSubsanacionReparosDesarrollo(config.Config{ContratacionTemporalSubsanacionPoliticaFile: ruta})
			if (err == nil) != caso.valida {
				t.Fatalf("valida=%v: %v", caso.valida, err)
			}
			if caso.valida && recibida != candidata {
				t.Fatal("politica alterada durante carga")
			}
		})
	}
}

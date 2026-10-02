package ensayofisicopg

import (
	"archive/tar"
	"context"
	"os"
	"path/filepath"
	"testing"

	puertos "vec-diputacion-granada/internal/modules/administracion/ports/ensayofisicopg"
)

func TestMaterialInteriorConservaArchivoOriginalYRechazaEscape(t *testing.T) {
	ctx := context.Background()
	c := Configuracion{LimiteExtraidoBytes: 4096, LimiteEntradas: 10}
	for _, malo := range []bool{false, true} {
		t.Run(map[bool]string{false: "admitido", true: "traversal_rechazado"}[malo], func(t *testing.T) {
			nombre := "contenido/config.json"
			if malo {
				nombre = "contenido/../escape"
			}
			a := tarMuestra(t, []*tar.Header{{Name: "contenido/", Typeflag: tar.TypeDir, Mode: 0700}, {Name: nombre, Typeflag: tar.TypeReg, Mode: 0600, Size: 10}})
			destino := t.TempDir()
			bytes, err := os.ReadFile(a.Ruta)
			if err != nil || os.WriteFile(filepath.Join(destino, "contenido"), bytes, 0600) != nil {
				t.Fatal("preparar material")
			}
			comp := puertos.Componente{ID: "fisica:material", Tipo: "material", RutaInterna: "/componentes/0001/contenido"}
			hash, _, err := huellaContenido(ctx, filepath.Join(destino, "contenido"))
			if err != nil {
				t.Fatal(err)
			}
			err = prepararMaterial(ctx, destino, c, &cuentaTar{}, &comp)
			if malo {
				if err == nil {
					t.Fatal("escape material aceptado")
				}
				if _, e := os.Stat(filepath.Join(destino, "material")); !os.IsNotExist(e) {
					t.Fatal("extrajo antes de validar")
				}
				return
			}
			if err != nil || comp.RutaMaterial != "/componentes/0001/material/contenido" {
				t.Fatal("material no preparado", err, comp)
			}
			despues, _, err := huellaContenido(ctx, filepath.Join(destino, "contenido"))
			if err != nil || despues != hash {
				t.Fatal("archivo de material original modificado")
			}
			if _, err := os.Stat(filepath.Join(destino, "material", "contenido", "config.json")); err != nil {
				t.Fatal("archivo material ausente")
			}
		})
	}
}

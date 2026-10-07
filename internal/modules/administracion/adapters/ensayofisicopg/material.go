package ensayofisicopg

import (
	"context"
	"os"
	"path/filepath"

	puertos "vec-diputacion-granada/internal/modules/administracion/ports/ensayofisicopg"
)

// prepararMaterial admite un TAR interior sólo para el tipo material gobernado.
// La huella original sigue siendo la del archivo regular; su árbol se monta aparte
// bajo el mismo componente y comparte límites acumulados de todo el conjunto.
func prepararMaterial(ctx context.Context, destino string, c Configuracion, cuenta *cuentaTar, componente *puertos.Componente) error {
	if componente.Tipo != "material" {
		return nil
	}
	origen := filepath.Join(destino, "contenido")
	i, err := os.Stat(origen)
	if err != nil {
		return errEntrada
	}
	if i.IsDir() {
		componente.RutaMaterial = componente.RutaInterna
		return nil
	}
	if !i.Mode().IsRegular() || revisarTar(ctx, origen, c, cuenta) != nil {
		return errEntrada
	}
	material := filepath.Join(destino, "material")
	if os.Mkdir(material, 0700) != nil || extraerTar(ctx, origen, material) != nil {
		return errEntrada
	}
	componente.RutaMaterial = filepath.ToSlash(filepath.Join(filepath.Dir(componente.RutaInterna), "material", "contenido"))
	return nil
}

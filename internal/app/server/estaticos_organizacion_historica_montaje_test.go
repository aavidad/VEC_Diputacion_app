package server

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// La pantalla de organización solo puede ofrecer las pestañas de histórico e
// importación cuando sus rutas están compuestas en la raíz. La bandera web
// MONTAJE_ORGANIZACION_HISTORICA y la composición Go deben coincidir: activar
// la bandera sin montar la ruta publicaría una pantalla que abre en error, y
// montar la ruta sin la bandera dejaría la capacidad inalcanzable.
func TestBanderaWebOrganizacionHistoricaCoincideConComposicion(t *testing.T) {
	raiz := filepath.Join("..", "..", "..")
	fuente, err := os.ReadFile(filepath.Join(raiz, "web", "static", "portal-empleado", "organizacion", "historico.js"))
	if err != nil {
		t.Fatal(err)
	}
	bandera := regexp.MustCompile(`MONTAJE_ORGANIZACION_HISTORICA = Object\.freeze\(\{ consulta: (true|false), importacion: (true|false) \}\);`).
		FindSubmatch(fuente)
	if bandera == nil {
		t.Fatal("historico.js no declara MONTAJE_ORGANIZACION_HISTORICA con el formato esperado")
	}
	compuestos := map[string]bool{}
	for _, dir := range []string{filepath.Join(raiz, "cmd"), filepath.Join(raiz, "internal")} {
		err := filepath.WalkDir(dir, func(ruta string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			// El paquete que define los constructores no cuenta como composición.
			if d.IsDir() || !strings.HasSuffix(ruta, ".go") || strings.HasSuffix(ruta, "_test.go") ||
				strings.Contains(filepath.ToSlash(ruta), "internal/vec/adapters/httpapi/") {
				return nil
			}
			codigo, err := os.ReadFile(ruta)
			if err != nil {
				return err
			}
			for _, constructor := range []string{"NewHandlerOrganizacionHistoricaPersonal(", "NewHandlerImportacionOrganizacionPersonal("} {
				if strings.Contains(string(codigo), constructor) {
					compuestos[constructor] = true
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for nombre, caso := range map[string]struct {
		bandera     string
		constructor string
	}{
		"consulta":    {string(bandera[1]), "NewHandlerOrganizacionHistoricaPersonal("},
		"importacion": {string(bandera[2]), "NewHandlerImportacionOrganizacionPersonal("},
	} {
		if (caso.bandera == "true") != compuestos[caso.constructor] {
			t.Errorf("%s: bandera web %s pero composición en la raíz %v", nombre, caso.bandera, compuestos[caso.constructor])
		}
	}
}

package separacionportales

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// variableDirectorioMaterial es la raíz del material del proceso. Es la
// única ruta que el externo no tiene que llevar dentro de sí misma.
const variableDirectorioMaterial = "VEC_DEVELOPMENT_MATERIAL_DIR"

// catalogosPublicosExterno son los ficheros que el proceso externo puede
// leer fuera de su material: catálogos públicos, sin claves ni datos de
// personas. Deben ser JSON y no estar dentro del material de otro portal.
var catalogosPublicosExterno = map[string]struct{}{
	"VEC_BOLSA_PUBLIC_SOURCE_PATH":     {},
	"VEC_BOLSA_CATEGORIES_SOURCE_PATH": {},
}

// rutasSinPrefijo son rutas que el proceso lee aunque no empiecen por VEC_:
// las heredadas de Bolsa y las de confianza TLS que leen pgx y Go por su
// cuenta. En el externo también deben quedar dentro de su material.
var rutasSinPrefijo = map[string]struct{}{
	"BOLSA_DATA_DIR": {}, "BOLSA_DATA_PATH": {},
	"PGSSLROOTCERT": {}, "PGSSLCRL": {}, "PGSSLCRLDIR": {},
	"SSL_CERT_FILE": {}, "SSL_CERT_DIR": {},
}

// esVariableRuta reconoce por el nombre las variables que apuntan a un
// fichero o a un directorio.
func esVariableRuta(nombre string) bool {
	if _, ok := rutasSinPrefijo[nombre]; ok {
		return true
	}
	if !strings.HasPrefix(nombre, "VEC_") {
		return false
	}
	for _, sufijo := range []string{"_PATH", "_DIR", "_FILE"} {
		if strings.HasSuffix(nombre, sufijo) {
			return true
		}
	}
	return false
}

// comprobarRutaExterno exige que toda ruta que recibe el proceso externo
// esté dentro de su propio material, salvo los catálogos públicos admitidos.
// Así el externo no puede leer ficheros del interno aunque tenga permiso de
// sistema para abrirlos. Una ruta relativa o con «..» se rechaza: su destino
// depende del directorio de trabajo.
func comprobarRutaExterno(nombre, valor, material string) error {
	if !esVariableRuta(nombre) || nombre == variableDirectorioMaterial {
		return nil
	}
	valor = strings.TrimSpace(valor)
	if _, publico := catalogosPublicosExterno[nombre]; publico {
		return comprobarCatalogoPublico(nombre, valor)
	}
	if material == "" || !filepath.IsAbs(material) {
		return rechazo("ruta del portal externo sin directorio de material propio", nombre)
	}
	if !rutaLimpiaAbsoluta(valor) || !dentroDe(filepath.Clean(material), valor) {
		return rechazo("ruta del portal externo fuera de su material", nombre)
	}
	return nil
}

// comprobarCatalogoPublico admite un fichero JSON fuera del material, pero no
// uno que esté, directamente o por un enlace, dentro de un material marcado
// como de otro portal. Una ruta relativa se resuelve desde el directorio de
// trabajo, como hace el cargador del catálogo.
func comprobarCatalogoPublico(nombre, valor string) error {
	if valor == "" || filepath.Clean(valor) != valor || valor == ".." || strings.HasPrefix(valor, ".."+string(filepath.Separator)) {
		return rechazo("catalogo publico con una ruta no valida", nombre)
	}
	if !strings.HasSuffix(valor, ".json") {
		return rechazo("catalogo publico que no es JSON", nombre)
	}
	absoluta, err := filepath.Abs(valor)
	if err == nil {
		absoluta, err = filepath.EvalSymlinks(absoluta)
	}
	if err != nil {
		return rechazo("no se puede comprobar el catalogo publico", nombre)
	}
	if info, errInfo := os.Stat(absoluta); errInfo != nil || !info.Mode().IsRegular() {
		return rechazo("catalogo publico que no es un fichero normal", nombre)
	}
	for directorio := filepath.Dir(absoluta); ; directorio = filepath.Dir(directorio) {
		_, err := os.Lstat(filepath.Join(directorio, FicheroMarcaPortal))
		if err == nil {
			if marca, errMarca := leerMarca(directorio); errMarca != nil || marca != PortalExterno {
				return rechazo("catalogo publico dentro del material de otro portal", nombre)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return rechazo("no se puede comprobar el catalogo publico", nombre)
		}
		if filepath.Dir(directorio) == directorio {
			return nil
		}
	}
}

// rutaLimpiaAbsoluta indica si la ruta es absoluta y no tiene «..», «.» ni
// separadores repetidos.
func rutaLimpiaAbsoluta(ruta string) bool {
	return filepath.IsAbs(ruta) && filepath.Clean(ruta) == ruta
}

// dentroDe indica si ruta es raiz o está debajo de ella; ambas limpias.
func dentroDe(raiz, ruta string) bool {
	rel, err := filepath.Rel(raiz, ruta)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

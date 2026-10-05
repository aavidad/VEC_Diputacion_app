// Package temascss traduce paquetes de color a hojas locales reproducibles.
// Generar no instala, publica ni concede autoridad administrativa.
package temascss

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"vec-diputacion-granada/internal/modules/administracion/domain/temas"
)

// Hoja vincula el recurso generado con el contenido canónico del paquete.
type Hoja struct {
	Contenido       []byte
	PaqueteSHA256   string
	ContenidoSHA256 string
}

// Generar vuelve a validar el material y limita la salida a colores opacos.
// La política debe proceder de configuración confiable, nunca del paquete.
// El consumidor debe suspender los atributos de tema y modo de color antiguos
// mientras aplica esta hoja para evitar mezclar autoridades visuales.
func Generar(paquete temas.Paquete, politica temas.Politica) (Hoja, error) {
	normalizado, err := paquete.Normalizar(politica)
	if err != nil {
		return Hoja{}, err
	}
	canonico, err := normalizado.Canonico()
	if err != nil {
		return Hoja{}, err
	}
	huellaPaquete := huella(canonico)
	tokens := append([]string(nil), politica.Tokens...)
	sort.Strings(tokens)
	lineas := []string{"@media (forced-colors: none) {"}
	for _, variante := range []struct {
		nombre, esquema string
		colores         map[string]string
	}{
		{"clara", "light", normalizado.Variantes.Clara},
		{"oscura", "dark", normalizado.Variantes.Oscura},
	} {
		lineas = append(lineas,
			"  body[data-paquete-tema=\""+huellaPaquete+"\"][data-variante-tema=\""+variante.nombre+"\"]:not([data-contraste=\"true\"]):not(.alto-contraste) {",
			"    color-scheme: "+variante.esquema+";")
		for _, token := range tokens {
			lineas = append(lineas, "    "+token+": "+variante.colores[token]+";")
		}
		lineas = append(lineas, "  }")
	}
	lineas = append(lineas, "}")
	contenido := []byte(strings.Join(lineas, "\n") + "\n")
	return Hoja{Contenido: contenido, PaqueteSHA256: huellaPaquete, ContenidoSHA256: huella(contenido)}, nil
}

func huella(contenido []byte) string {
	h := sha256.Sum256(contenido)
	return hex.EncodeToString(h[:])
}

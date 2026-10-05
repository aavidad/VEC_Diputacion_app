// Package informeperiodo escribe en CSV la selección de informes de Dietas.
//
// Todos los rótulos proceden de un catálogo por idioma. Los importes salen en
// céntimos enteros para que la hoja de cálculo no dependa del separador
// decimal. No incluye referencias opacas de persona ni de unidad.
package informeperiodo

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/language"
	app "vec-diputacion-granada/internal/modules/dietas/application/informeperiodo"
)

// ErrCatalogoInvalido indica un catálogo mal formado o incompleto para la selección.
var ErrCatalogoInvalido = errors.New("catalogo_csv_informe_dietas_invalido")

// ErrSalidaNoDisponible indica que no se pudo componer el CSV.
var ErrSalidaNoDisponible = errors.New("csv_informe_dietas_no_disponible")

const limiteCatalogo = 65536

// CatalogoCSV contiene los textos de la muestra; no es una política de exportación.
type CatalogoCSV struct {
	Esquema      string            `json:"esquema"`
	Referencia   string            `json:"referencia"`
	Version      string            `json:"version"`
	Idioma       string            `json:"idioma"`
	FormatoFecha string            `json:"formato_fecha"`
	Cabeceras    map[string]string `json:"cabeceras"`
	CamposFecha  map[string]string `json:"campos_fecha"`
	Conceptos    map[string]string `json:"conceptos"`
	Situaciones  map[string]string `json:"situaciones"`
	Sintetico    string            `json:"sintetico"`
}

var columnasFijas = [...]string{"referencia", "version_comision", "persona", "unidad", "situacion"}

// CargarCatalogo valida el esquema cerrado del catálogo CSV.
func CargarCatalogo(r io.Reader) (CatalogoCSV, error) {
	var c CatalogoCSV
	raw, err := io.ReadAll(io.LimitReader(r, limiteCatalogo+1))
	if err != nil || len(raw) > limiteCatalogo {
		return c, ErrCatalogoInvalido
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(&c) != nil || dec.Decode(new(any)) != io.EOF {
		return CatalogoCSV{}, ErrCatalogoInvalido
	}
	version, err := strconv.Atoi(c.Version)
	idioma, errIdioma := language.Parse(c.Idioma)
	if err != nil || errIdioma != nil || version < 1 || strconv.Itoa(version) != c.Version ||
		idioma.String() != c.Idioma || c.Esquema != "dietas-informes-csv-v1" || !texto(c.Referencia, 512) ||
		!texto(c.FormatoFecha, 32) || !texto(c.Sintetico, 2048) ||
		len(c.Cabeceras) != len(columnasFijas)+2 || len(c.CamposFecha) != len(app.CamposFecha) ||
		!mapaTextos(c.Conceptos) || !mapaTextos(c.Situaciones) {
		return CatalogoCSV{}, ErrCatalogoInvalido
	}
	for _, clave := range append(columnasFijas[:], "importe_incluido_centimos", "aviso") {
		if !texto(c.Cabeceras[clave], 128) {
			return CatalogoCSV{}, ErrCatalogoInvalido
		}
	}
	for _, clave := range app.CamposFecha {
		if !texto(c.CamposFecha[clave], 128) {
			return CatalogoCSV{}, ErrCatalogoInvalido
		}
	}
	return c, nil
}

// Escribir devuelve el CSV completo o un error sin bytes parciales. Cada
// concepto incluido y cada situación seleccionada debe tener su rótulo.
func Escribir(c CatalogoCSV, inf app.Informe) ([]byte, error) {
	cabeceras := make([]string, 0, len(columnasFijas)+len(inf.Conceptos)+3)
	for _, clave := range columnasFijas {
		cabeceras = append(cabeceras, seguro(c.Cabeceras[clave]))
	}
	cabeceras = append(cabeceras, seguro(c.CamposFecha[inf.CampoFecha]))
	for _, clave := range inf.Conceptos {
		rotulo, ok := c.Conceptos[clave]
		if !ok {
			return nil, ErrCatalogoInvalido
		}
		cabeceras = append(cabeceras, seguro(rotulo))
	}
	cabeceras = append(cabeceras, seguro(c.Cabeceras["importe_incluido_centimos"]), seguro(c.Cabeceras["aviso"]))
	if cabeceras[len(columnasFijas)] == "" {
		return nil, ErrCatalogoInvalido
	}
	var salida bytes.Buffer
	w := csv.NewWriter(&salida)
	if w.Write(cabeceras) != nil {
		return nil, ErrSalidaNoDisponible
	}
	for _, f := range inf.Filas {
		situacion, ok := c.Situaciones[f.Situacion]
		if !ok || len(f.ConceptosCentimos) != len(inf.Conceptos) {
			return nil, ErrCatalogoInvalido
		}
		fila := []string{seguro(f.Referencia), strconv.Itoa(f.VersionComision), seguro(f.Persona),
			seguro(f.Unidad), seguro(situacion), seguro(f.Fecha.Format(c.FormatoFecha))}
		// Los números los produce el servidor: no son fórmulas y siguen siendo operables.
		for _, v := range f.ConceptosCentimos {
			fila = append(fila, strconv.FormatInt(v, 10))
		}
		fila = append(fila, strconv.FormatInt(f.ImporteIncluidoCentimos, 10), seguro(c.Sintetico))
		if w.Write(fila) != nil {
			return nil, ErrSalidaNoDisponible
		}
	}
	w.Flush()
	if w.Error() != nil {
		return nil, ErrSalidaNoDisponible
	}
	return salida.Bytes(), nil
}

func mapaTextos(m map[string]string) bool {
	if len(m) == 0 || len(m) > 20 {
		return false
	}
	for clave, v := range m {
		if clave == "" || !texto(v, 128) {
			return false
		}
	}
	return true
}

func texto(s string, max int) bool { return strings.TrimSpace(s) != "" && len(s) <= max }

// seguro neutraliza celdas que una hoja de cálculo interpretaría como fórmula.
func seguro(s string) string {
	limpia := strings.TrimLeftFunc(s, func(r rune) bool { return unicode.IsSpace(r) || unicode.In(r, unicode.Cf) })
	if len(limpia) > 0 && strings.ContainsRune("=+-@", rune(limpia[0])) {
		return "'" + s
	}
	return s
}

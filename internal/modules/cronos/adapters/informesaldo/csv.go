package informesaldo

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/language"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

// CatalogoCSV contiene los textos de la muestra local; no es una política de exportación.
type CatalogoCSV struct {
	Esquema      string            `json:"esquema"`
	Referencia   string            `json:"referencia"`
	Version      string            `json:"version"`
	Idioma       string            `json:"idioma"`
	FormatoFecha string            `json:"formato_fecha"`
	Cabeceras    map[string]string `json:"cabeceras"`
	Estados      map[string]string `json:"estados"`
	Desconocido  string            `json:"desconocido"`
	Sintetico    string            `json:"sintetico"`
}

var columnasSaldoCSV = [...]string{"persona_ejemplo", "desde", "hasta", "previstos_minutos", "trabajados_minutos", "saldo_minutos", "estado", "aviso"}

// PrepararCSVEjemploSintetico prepara el saldo recibido, sin recalcularlo ni
// consultar fuentes. Todas las cantidades son minutos enteros; nil no es cero.
func PrepararCSVEjemploSintetico(ctx context.Context, datos io.Reader, saldo ports.SaldoExportable, nombre string) ([]byte, error) {
	if ctx == nil || dependenciaNula(datos) {
		return nil, ports.ErrExportacionSaldoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !texto(nombre, 128) {
		return nil, ports.ErrExportacionSaldoInvalida
	}
	raw, err := io.ReadAll(io.LimitReader(datos, 65537))
	if err != nil || len(raw) > 65536 {
		return nil, ports.ErrExportacionSaldoInvalida
	}
	var c CatalogoCSV
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(&c) != nil || dec.Decode(new(any)) != io.EOF || validarCatalogoSaldoCSV(c) != nil {
		return nil, ports.ErrExportacionSaldoInvalida
	}
	desde, hasta, err := validarSaldoExportable(saldo)
	if err != nil {
		return nil, err
	}
	cabeceras := make([]string, len(columnasSaldoCSV))
	for i, clave := range columnasSaldoCSV {
		cabeceras[i] = textoSeguroCSV(c.Cabeceras[clave])
	}
	cantidad := func(v *int64) string {
		if v == nil {
			return textoSeguroCSV(c.Desconocido)
		}
		// Sólo números producidos por el servidor quedan exentos: no son fórmulas.
		return strconv.FormatInt(*v, 10)
	}
	r := saldo.Resumen
	fila := []string{textoSeguroCSV(nombre), textoSeguroCSV(desde.Format(c.FormatoFecha)), textoSeguroCSV(hasta.Format(c.FormatoFecha)),
		cantidad(r.PrevistosMinutos), strconv.FormatInt(r.TrabajadosMinutos, 10), cantidad(r.SaldoMinutos),
		textoSeguroCSV(c.Estados[r.Estado]), textoSeguroCSV(c.Sintetico)}
	var salida bytes.Buffer
	w := csv.NewWriter(&salida)
	if err := w.Write(cabeceras); err != nil {
		return nil, ports.ErrExportacionSaldoNoDisponible
	}
	if err := w.Write(fila); err != nil {
		return nil, ports.ErrExportacionSaldoNoDisponible
	}
	w.Flush()
	if w.Error() != nil {
		return nil, ports.ErrExportacionSaldoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append([]byte(nil), salida.Bytes()...), nil
}

func validarCatalogoSaldoCSV(c CatalogoCSV) error {
	version, err := strconv.ParseInt(c.Version, 10, 64)
	idioma, errIdioma := language.Parse(c.Idioma)
	if err != nil || errIdioma != nil || version < 1 || strconv.FormatInt(version, 10) != c.Version || idioma.String() != c.Idioma ||
		c.Esquema != "cronos-saldo-csv-v1" || !texto(c.Referencia, 512) || !texto(c.FormatoFecha, 32) ||
		!texto(c.Desconocido, 256) || !texto(c.Sintetico, 2048) || len(c.Cabeceras) != len(columnasSaldoCSV) || len(c.Estados) != 3 {
		return ports.ErrExportacionSaldoInvalida
	}
	for _, clave := range columnasSaldoCSV {
		if !texto(c.Cabeceras[clave], 128) {
			return ports.ErrExportacionSaldoInvalida
		}
	}
	for _, clave := range []string{ports.EstadoSaldoDisponible, ports.EstadoSaldoNoDisponible, ports.EstadoSaldoIncompleto} {
		if !texto(c.Estados[clave], 256) {
			return ports.ErrExportacionSaldoInvalida
		}
	}
	return nil
}

func textoSeguroCSV(s string) string {
	limpia := strings.TrimLeftFunc(s, func(r rune) bool { return unicode.IsSpace(r) || unicode.In(r, unicode.Cf) })
	if len(limpia) > 0 && strings.ContainsRune("=+-@", rune(limpia[0])) {
		return "'" + s
	}
	return s
}

package informemovimientos

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/language"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

// CatalogoCSV conserva todos los textos de la salida en datos por idioma.
type CatalogoCSV struct {
	Referencia         string            `json:"referencia"`
	Version            string            `json:"version"`
	Idioma             string            `json:"idioma"`
	FormatoFecha       string            `json:"formato_fecha"`
	FormatoHora        string            `json:"formato_hora"`
	Cabeceras          map[string]string `json:"cabeceras"`
	Contexto           string            `json:"contexto"`
	Marcaje            string            `json:"marcaje"`
	Sintetico          string            `json:"sintetico"`
	FuenteCompleta     string            `json:"fuente_completa"`
	FuenteIncompleta   string            `json:"fuente_incompleta"`
	OrigenSinVerificar string            `json:"origen_sin_verificar"`
	Movimientos        map[string]string `json:"movimientos"`
	Origenes           map[string]string `json:"origenes"`
}

var clavesCabecerasCSV = [...]string{
	"registro", "fecha", "hora", "movimiento", "origen", "desde", "hasta",
	"zona_horaria", "estado_fuente", "aviso",
}

// PrepararCSV recibe el mismo ejemplo cerrado que el PDF de movimientos. La
// primera fila de datos conserva el periodo y la completitud incluso sin hechos.
func PrepararCSV(ctx context.Context, catalogo io.Reader, e EjemploSintetico) ([]byte, error) {
	if ctx == nil || nulo(catalogo) || ctx.Err() != nil {
		return nil, ErrEjemploNoDisponible
	}
	var c CatalogoCSV
	if err := leerJSON(catalogo, &c); err != nil {
		return nil, err
	}
	if err := validarCatalogoCSV(c); err != nil {
		return nil, err
	}
	zona, err := time.LoadLocation(e.ZonaHoraria)
	if err != nil || !texto(e.ZonaHoraria, 64) || e.ZonaHoraria == "Local" || !e.Demo || !texto(e.Nombre, 128) || e.Completo == nil || e.Marcajes == nil || len(e.Marcajes) > 10000 {
		return nil, ErrEjemploInvalido
	}
	desde, hasta, err := periodo(e.Periodo, zona)
	if err != nil {
		return nil, err
	}
	fin := hasta.AddDate(0, 0, 1)
	ordenados := append([]ports.MarcajeDia(nil), e.Marcajes...)
	for _, m := range ordenados {
		if m.InstanteUTC.Location() != time.UTC || m.InstanteUTC.IsZero() || m.InstanteUTC.Nanosecond()%1000 != 0 || m.InstanteUTC.Before(desde) || !m.InstanteUTC.Before(fin) || c.Movimientos[string(m.Movimiento)] == "" {
			return nil, ErrEjemploInvalido
		}
		if m.Origen != nil && c.Origenes[*m.Origen] == "" {
			return nil, ErrEjemploInvalido
		}
	}
	sort.SliceStable(ordenados, func(i, j int) bool { return ordenados[i].InstanteUTC.Before(ordenados[j].InstanteUTC) })
	var salida bytes.Buffer
	w := csv.NewWriter(&salida)
	cabeceras := make([]string, 0, len(clavesCabecerasCSV))
	for _, clave := range clavesCabecerasCSV {
		cabeceras = append(cabeceras, c.Cabeceras[clave])
	}
	if err := escribirFilaCSV(w, cabeceras); err != nil {
		return nil, err
	}
	fuente := c.FuenteIncompleta
	if *e.Completo {
		fuente = c.FuenteCompleta
	}
	if err := escribirFilaCSV(w, []string{c.Contexto, "", "", "", "", desde.Format(c.FormatoFecha), hasta.Format(c.FormatoFecha), e.ZonaHoraria, fuente, c.Sintetico}); err != nil {
		return nil, err
	}
	for _, m := range ordenados {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		at := m.InstanteUTC.In(zona)
		origen := c.OrigenSinVerificar
		if m.Origen != nil {
			origen = c.Origenes[*m.Origen]
		}
		if err := escribirFilaCSV(w, []string{c.Marcaje, at.Format(c.FormatoFecha), at.Format(c.FormatoHora), c.Movimientos[string(m.Movimiento)], origen, "", "", "", "", ""}); err != nil {
			return nil, err
		}
		if salida.Len() > 2*1024*1024 {
			return nil, ErrEjemploInvalido
		}
	}
	w.Flush()
	if w.Error() != nil || salida.Len() > 2*1024*1024 || ctx.Err() != nil {
		return nil, ErrEjemploNoDisponible
	}
	return append([]byte(nil), salida.Bytes()...), nil
}

func validarCatalogoCSV(c CatalogoCSV) error {
	v, err := strconv.ParseInt(c.Version, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrEjemploInvalido, err)
	}
	if v < 1 || strconv.FormatInt(v, 10) != c.Version {
		return ErrEjemploInvalido
	}
	idioma, err := language.Parse(c.Idioma)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrEjemploInvalido, err)
	}
	if idioma.String() != c.Idioma || !texto(c.Referencia, 512) || !texto(c.FormatoFecha, 32) || !texto(c.FormatoHora, 64) || !mapa(c.Cabeceras, clavesCabecerasCSV[:]) {
		return ErrEjemploInvalido
	}
	base, confianza := idioma.Base()
	if confianza == language.No || c.Referencia != "cronos-informe-movimientos-csv-"+base.String() {
		return ErrEjemploInvalido
	}
	if !strings.Contains(c.FormatoHora, "-07:00") && !strings.Contains(c.FormatoHora, "Z07:00") {
		return ErrEjemploInvalido
	}
	for _, s := range c.Cabeceras {
		if !texto(s, 128) {
			return ErrEjemploInvalido
		}
	}
	for _, s := range []string{c.Contexto, c.Marcaje, c.Sintetico, c.FuenteCompleta, c.FuenteIncompleta, c.OrigenSinVerificar} {
		if !texto(s, 2048) {
			return ErrEjemploInvalido
		}
	}
	if !mapa(c.Movimientos, []string{string(domain.PunchEntry), string(domain.PunchExit), string(domain.PunchPauseStart), string(domain.PunchPauseEnd)}) || !mapa(c.Origenes, []string{"terminal", "remoto"}) {
		return ErrEjemploInvalido
	}
	return nil
}

func escribirFilaCSV(w *csv.Writer, fila []string) error {
	copia := make([]string, len(fila))
	for i, s := range fila {
		limpia := strings.TrimLeftFunc(s, func(r rune) bool { return unicode.IsSpace(r) || unicode.In(r, unicode.Cf) })
		if len(limpia) > 0 && strings.ContainsRune("=+-@", rune(limpia[0])) {
			copia[i] = "'" + s
		} else {
			copia[i] = s
		}
	}
	return w.Write(copia)
}

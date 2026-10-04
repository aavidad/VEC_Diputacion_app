package informepermisos

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/language"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

// CatalogoCSV solo da textos a la muestra local. No es una política de exportación.
type CatalogoCSV struct {
	Esquema      string            `json:"esquema"`
	Referencia   string            `json:"referencia"`
	Version      string            `json:"version"`
	Idioma       string            `json:"idioma"`
	FormatoCorte string            `json:"formato_corte"`
	ZonaHoraria  string            `json:"zona_horaria"`
	Cabeceras    map[string]string `json:"cabeceras"`
	Unidades     map[string]string `json:"unidades"`
	Computos     map[string]string `json:"computos"`
	Estados      map[string]string `json:"estados"`
	Desconocido  string            `json:"desconocido"`
	Contexto     string            `json:"contexto"`
	Permiso      string            `json:"permiso"`
	Sintetico    string            `json:"sintetico"`
}

type EjemploSinteticoCSV struct {
	Demo    bool
	Nombre  string
	Resumen ports.ResumenPermisosInforme
}

var columnasContextoPermisosCSV = [...]string{"registro", "persona_ejemplo", "ejercicio", "corte", "aviso"}

// PrepararCSVEjemploSintetico usa una proyección sin referencias. Ninguna
// columna excluida llega a las cabeceras ni a las filas, incluso si el ejemplo
// original contenía otros valores.
func PrepararCSVEjemploSintetico(ctx context.Context, datos io.Reader, e EjemploSinteticoCSV) ([]byte, error) {
	if ctx == nil || nulo(datos) {
		return nil, ports.ErrExportacionPermisosNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !e.Demo || !texto(e.Nombre, 128) {
		return nil, ports.ErrExportacionPermisosInvalida
	}
	raw, err := io.ReadAll(io.LimitReader(datos, 65537))
	if err != nil || len(raw) > 65536 || validarJSONSinDuplicados(raw) != nil {
		return nil, ports.ErrExportacionPermisosInvalida
	}
	var c CatalogoCSV
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(&c) != nil || dec.Decode(new(any)) != io.EOF || validarCatalogoPermisosCSV(c) != nil {
		return nil, ports.ErrExportacionPermisosInvalida
	}
	r := e.Resumen
	if r.Ejercicio < 1 || r.Ejercicio > 9999 || r.CorteUTC.IsZero() || r.CorteUTC.Location() != time.UTC || r.CorteUTC.Nanosecond()%1000 != 0 || len(r.Filas) > 256 {
		return nil, ports.ErrExportacionPermisosInvalida
	}
	permitidos, err := camposInformePermisosPermitidos(r.CamposPermitidos)
	if err != nil {
		return nil, err
	}
	zona, err := time.LoadLocation(c.ZonaHoraria)
	if err != nil {
		return nil, ports.ErrExportacionPermisosInvalida
	}
	columnas := make([]string, 0, len(columnasContextoPermisosCSV)+len(permitidos))
	for _, clave := range columnasContextoPermisosCSV {
		columnas = append(columnas, textoSeguroPermisosCSV(c.Cabeceras[clave]))
	}
	seleccion := make([]string, 0, len(permitidos))
	for _, clave := range clavesCamposInformePermisos {
		if permitidos[clave] {
			seleccion = append(seleccion, clave)
			columnas = append(columnas, textoSeguroPermisosCSV(c.Cabeceras[clave]))
		}
	}
	var salida bytes.Buffer
	escritor := csv.NewWriter(&salida)
	if escritor.Write(columnas) != nil {
		return nil, ports.ErrExportacionPermisosNoDisponible
	}
	contexto := []string{textoSeguroPermisosCSV(c.Contexto), textoSeguroPermisosCSV(e.Nombre), strconv.Itoa(r.Ejercicio), textoSeguroPermisosCSV(r.CorteUTC.In(zona).Format(c.FormatoCorte)), textoSeguroPermisosCSV(c.Sintetico)}
	contexto = append(contexto, make([]string, len(seleccion))...)
	if escritor.Write(contexto) != nil {
		return nil, ports.ErrExportacionPermisosNoDisponible
	}
	for _, f := range r.Filas {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := validarFilaInformePermisos(f, permitidos); err != nil {
			return nil, err
		}
		fila := []string{textoSeguroPermisosCSV(c.Permiso), "", "", "", ""}
		for _, clave := range seleccion {
			valor := valorInformePermisosCSV(f, clave, c)
			if clave == "pendiente_resolver" || clave == "concedido" || clave == "restante" {
				fila = append(fila, valor)
			} else {
				fila = append(fila, textoSeguroPermisosCSV(valor))
			}
		}
		if escritor.Write(fila) != nil || salida.Len() > 2*1024*1024 {
			return nil, ports.ErrExportacionPermisosNoDisponible
		}
	}
	escritor.Flush()
	if escritor.Error() != nil || salida.Len() == 0 || salida.Len() > 2*1024*1024 {
		return nil, ports.ErrExportacionPermisosNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append([]byte(nil), salida.Bytes()...), nil
}

func validarCatalogoPermisosCSV(c CatalogoCSV) error {
	version, err := strconv.ParseInt(c.Version, 10, 64)
	idioma, errIdioma := language.Parse(c.Idioma)
	if err != nil || errIdioma != nil || version < 1 || strconv.FormatInt(version, 10) != c.Version || idioma.String() != c.Idioma ||
		c.Esquema != "cronos-permisos-csv-v1" || !texto(c.Referencia, 512) || !texto(c.FormatoCorte, 64) || !texto(c.ZonaHoraria, 64) ||
		!texto(c.Desconocido, 256) || !texto(c.Contexto, 256) || !texto(c.Permiso, 256) || !texto(c.Sintetico, 2048) ||
		len(c.Cabeceras) != len(columnasContextoPermisosCSV)+len(clavesCamposInformePermisos) {
		return ports.ErrExportacionPermisosInvalida
	}
	for _, clave := range columnasContextoPermisosCSV {
		if !texto(c.Cabeceras[clave], 128) {
			return ports.ErrExportacionPermisosInvalida
		}
	}
	for _, clave := range clavesCamposInformePermisos {
		if !texto(c.Cabeceras[clave], 128) {
			return ports.ErrExportacionPermisosInvalida
		}
	}
	if !mapaValido(c.Unidades, []string{string(domain.LeaveUnitDay), string(domain.LeaveUnitHour)}) ||
		!mapaValido(c.Computos, []string{string(domain.ComputoLaborables), string(domain.ComputoNaturales)}) ||
		!mapaValido(c.Estados, []string{ports.ConciliacionPermisosConfirmada, ports.ConciliacionPermisosPendiente}) {
		return ports.ErrExportacionPermisosInvalida
	}
	return nil
}

func valorInformePermisosCSV(f ports.FilaInformePermisos, clave string, c CatalogoCSV) string {
	switch clave {
	case "etiqueta":
		return f.Etiqueta
	case "unidad":
		return c.Unidades[string(f.Unidad)]
	case "computo":
		return c.Computos[string(f.Computo)]
	case "pendiente_resolver":
		return cantidadInformePermisosCSV(f.PendienteResolver, c.Desconocido)
	case "concedido":
		return cantidadInformePermisosCSV(f.Concedido, c.Desconocido)
	case "restante":
		return cantidadInformePermisosCSV(f.Restante, c.Desconocido)
	case "conciliacion":
		return c.Estados[f.Conciliacion]
	default:
		return ""
	}
}

func cantidadInformePermisosCSV(v *int64, desconocido string) string {
	if v == nil {
		return textoSeguroPermisosCSV(desconocido)
	}
	// Solo los enteros validados se mantienen como números, incluido el cero.
	return strconv.FormatInt(*v, 10)
}

func textoSeguroPermisosCSV(s string) string {
	limpia := strings.TrimLeftFunc(s, func(r rune) bool { return unicode.IsSpace(r) || unicode.In(r, unicode.Cf) })
	if len(limpia) > 0 && strings.ContainsRune("=+-@", rune(limpia[0])) {
		return "'" + s
	}
	return s
}

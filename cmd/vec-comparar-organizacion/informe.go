package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	personal "vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/shared/i18n"
	"vec-diputacion-granada/web"
)

type filaCambio struct {
	Nombre, Tipo, ID string
	Campos           []valorCambio
	Traza            string
}
type valorCambio struct{ Campo, Antes, Despues string }
type coleccionVista struct {
	Clave, Nombre, Estado, CoberturaAntes, CoberturaDespues string
	Antes, Despues                                          int
	Cambios                                                 []filaCambio
	NoVerificables                                          []string
}
type corteVista struct{ Fecha, Conocido, Organismo, Unidad, RPT, Plantilla string }
type vista struct {
	Idioma             string
	Textos             map[string]string
	CSS                template.CSS
	Antes, Despues     corteVista
	Colecciones        []coleccionVista
	Huella, Manifiesto string
}

func informeHTML(w io.Writer, e entrada, s salida, c *i18n.Catalog) error {
	css, err := web.CSSComparacionOrganizacion()
	if err != nil {
		return err
	}
	datos, err := json.MarshalIndent(s.Manifiesto, "", "  ")
	if err != nil {
		return err
	}
	// CSS is exclusively the embedded, repository-owned common theme and layout.
	v := vista{Idioma: s.Idioma, Textos: s.Mensajes, CSS: template.CSS(css + estiloInforme), Antes: corte(e.Antes.Selector, s.Idioma, c), Despues: corte(e.Despues.Selector, s.Idioma, c), Huella: s.HuellaSHA256, Manifiesto: string(datos)} // #nosec G203 -- trusted embedded CSS; no input is converted to a safe template type.
	conjuntos := []struct {
		clave                            string
		resultado                        personal.ColeccionComparacionOrganizacion
		antes, despues                   any
		coberturaAntes, coberturaDespues string
	}{
		{"unidades", s.Manifiesto.Comparacion.Unidades, e.Antes.Unidades, e.Despues.Unidades, e.Antes.Cobertura.Unidades, e.Despues.Cobertura.Unidades},
		{"puestos_tipo", s.Manifiesto.Comparacion.PuestosTipo, e.Antes.PuestosTipo, e.Despues.PuestosTipo, e.Antes.Cobertura.PuestosTipo, e.Despues.Cobertura.PuestosTipo},
		{"dotaciones", s.Manifiesto.Comparacion.Dotaciones, e.Antes.Dotaciones, e.Despues.Dotaciones, e.Antes.Cobertura.Dotaciones, e.Despues.Cobertura.Dotaciones},
		{"plazas", s.Manifiesto.Comparacion.Plazas, e.Antes.Plazas, e.Despues.Plazas, e.Antes.Cobertura.Plazas, e.Despues.Cobertura.Plazas},
		{"puestos_individuales", s.Manifiesto.Comparacion.PuestosIndividuales, e.Antes.PuestosIndividuales, e.Despues.PuestosIndividuales, e.Antes.Cobertura.PuestosIndividuales, e.Despues.Cobertura.PuestosIndividuales},
		{"vinculos", s.Manifiesto.Comparacion.Vinculos, e.Antes.Vinculos, e.Despues.Vinculos, e.Antes.Cobertura.Vinculos, e.Despues.Cobertura.Vinculos},
	}
	for _, conjunto := range conjuntos {
		a, err := indexar(conjunto.antes)
		if err != nil {
			return err
		}
		d, err := indexar(conjunto.despues)
		if err != nil {
			return err
		}
		cv := coleccionVista{Clave: conjunto.clave, Nombre: texto(c, s.Idioma, conjunto.clave), Estado: texto(c, s.Idioma, conjunto.resultado.Estado), Antes: conjunto.resultado.Antes, Despues: conjunto.resultado.Despues, CoberturaAntes: texto(c, s.Idioma, conjunto.coberturaAntes), CoberturaDespues: texto(c, s.Idioma, conjunto.coberturaDespues), NoVerificables: conjunto.resultado.NoVerificables}
		for _, cambio := range conjunto.resultado.Cambios {
			aa, dd := a[cambio.ID], d[cambio.ID]
			fila := filaCambio{Nombre: nombre(aa, dd, cv.Nombre), Tipo: texto(c, s.Idioma, cambio.Tipo), ID: cambio.ID}
			campos := cambio.Campos
			if len(campos) == 0 {
				campos = camposBasicos(aa, dd)
			}
			for _, campo := range campos {
				fila.Campos = append(fila.Campos, valorCambio{texto(c, s.Idioma, campo), valor(aa, campo, s.Idioma, c), valor(dd, campo, s.Idioma, c)})
			}
			trazas, err := json.MarshalIndent(struct {
				Antes, Despues *personal.TrazaOrganizacionHistorica
			}{cambio.TrazaAntes, cambio.TrazaDespues}, "", "  ")
			if err != nil {
				return err
			}
			fila.Traza = string(trazas)
			cv.Cambios = append(cv.Cambios, fila)
		}
		v.Colecciones = append(v.Colecciones, cv)
	}
	return template.Must(template.New("informe").Funcs(template.FuncMap{"n": func(n int) string { return numero(n, s.Mensajes["separador_miles"]) }}).Parse(plantillaInforme)).Execute(w, v)
}

func corte(s personal.SelectorOrganizacionHistorica, idioma string, c *i18n.Catalog) corteVista {
	fecha := string(s.VigenteEn)
	if t, err := time.Parse("2006-01-02", fecha); err == nil {
		if formato, ok := c.Message(idioma, "formato_fecha"); ok {
			fecha = t.Format(formato)
		}
	}
	zona, _ := time.LoadLocation("Europe/Madrid")
	conocido := s.ConocidoEn.In(zona).Format(texto(c, idioma, "formato_instante"))
	return corteVista{fecha, conocido, s.OrganismoRef, s.UnidadClave, s.VersionRPTRef, s.VersionPlantillaRef}
}
func indexar(elementos any) (map[string]map[string]any, error) {
	datos, err := json.Marshal(elementos)
	if err != nil {
		return nil, err
	}
	var filas []map[string]any
	decoder := json.NewDecoder(bytes.NewReader(datos))
	decoder.UseNumber()
	if err = decoder.Decode(&filas); err != nil {
		return nil, err
	}
	resultado := map[string]map[string]any{}
	for _, fila := range filas {
		traza, ok := fila["traza"].(map[string]any)
		if !ok {
			continue
		}
		id, _ := traza["id"].(string)
		resultado[id] = fila
	}
	return resultado, nil
}
func nombre(a, d map[string]any, alternativo string) string {
	for _, fila := range []map[string]any{d, a} {
		for _, k := range []string{"denominacion", "etiqueta", "codigo_fuente"} {
			if v, ok := fila[k].(string); ok && v != "" {
				return v
			}
		}
	}
	return alternativo
}
func camposBasicos(a, d map[string]any) []string {
	m := map[string]bool{}
	for _, fila := range []map[string]any{a, d} {
		for k := range fila {
			if k != "traza" {
				m[k] = true
			}
		}
	}
	campos := make([]string, 0, len(m))
	for k := range m {
		campos = append(campos, k)
	}
	sort.Strings(campos)
	return campos
}
func valor(fila map[string]any, campo, idioma string, c *i18n.Catalog) string {
	if fila == nil {
		return texto(c, idioma, "no_aportado")
	}
	var dato any = fila
	for _, parte := range strings.Split(campo, ".") {
		mapa, ok := dato.(map[string]any)
		if !ok {
			return texto(c, idioma, "no_aportado")
		}
		dato = mapa[parte]
	}
	if campo == "traza" {
		return texto(c, idioma, "traza")
	}
	if dato == nil || dato == "" {
		return texto(c, idioma, "no_aportado")
	}
	if s, ok := dato.(string); ok {
		if campo == "estado_estructural" || campo == "tipo" {
			return texto(c, idioma, s)
		}
		return s
	}
	if n, ok := dato.(json.Number); ok {
		return numeroTexto(n.String(), texto(c, idioma, "separador_miles"))
	}
	datos, _ := json.Marshal(dato)
	return string(datos)
}

func numero(n int, separador string) string {
	return numeroTexto(strconv.Itoa(n), separador)
}
func numeroTexto(s, separador string) string {
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + separador + s[i:]
	}
	return s
}

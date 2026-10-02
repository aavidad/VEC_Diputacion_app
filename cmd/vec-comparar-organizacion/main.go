// vec-comparar-organizacion compares synthetic preparations or explicit nominal queries.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"

	personal "vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/shared/i18n"
	"vec-diputacion-granada/web"
)

const esquemaEntrada = "vec.personal.comparacion-organizacion.preparacion.v1"
const esquemaEntradaPaginada = "vec.personal.comparacion-organizacion.paginas-sinteticas.v1"
const esquemaManifiesto = "vec.personal.comparacion-organizacion.manifiesto.v1"
const limiteEntrada = 8 << 20

type entrada struct {
	Esquema        string                                      `json:"esquema"`
	Sintetico      bool                                        `json:"sintetico"`
	Idioma         string                                      `json:"idioma"`
	Formato        string                                      `json:"formato"`
	Antes          personal.InstantaneaComparacionOrganizacion `json:"antes"`
	Despues        personal.InstantaneaComparacionOrganizacion `json:"despues"`
	AntesPaginas   []personal.PaginaInstantaneaOrganizacion    `json:"antes_paginas,omitempty"`
	DespuesPaginas []personal.PaginaInstantaneaOrganizacion    `json:"despues_paginas,omitempty"`
}

type manifiesto struct {
	Antes          personal.InstantaneaComparacionOrganizacion `json:"antes"`
	Despues        personal.InstantaneaComparacionOrganizacion `json:"despues"`
	Esquema        string                                      `json:"esquema"`
	Sintetico      bool                                        `json:"sintetico"`
	EntradaSHA256  string                                      `json:"entrada_sha256"`
	Comparacion    personal.ComparacionOrganizacionHistorica   `json:"comparacion"`
	AntesPaginas   []personal.PaginaInstantaneaOrganizacion    `json:"antes_paginas,omitempty"`
	DespuesPaginas []personal.PaginaInstantaneaOrganizacion    `json:"despues_paginas,omitempty"`
}

type salida struct {
	Manifiesto   manifiesto        `json:"manifiesto"`
	HuellaSHA256 string            `json:"huella_sha256"`
	Idioma       string            `json:"idioma"`
	Mensajes     map[string]string `json:"mensajes"`
}

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }

func ejecutar(args []string, input io.Reader, output, errors io.Writer) int {
	if len(args) == 2 && args[0] == "-consulta-nominal" {
		return ejecutarConsultaNominal(args[1], input, output, errors, nuevaConsultaNominal)
	}
	catalogo, mensajes, err := web.CatalogoComparacionOrganizacion()
	if err != nil {
		return codigoErrorSalida(err)
	}
	idioma := catalogo.DefaultLocale()
	fallo := func(key string) int {
		if _, err := fmt.Fprintln(errors, catalogo.T(idioma, key)); err != nil {
			return codigoErrorSalida(err)
		}
		return 1
	}
	if len(args) != 0 {
		return fallo("error_entrada")
	}
	datos, err := io.ReadAll(io.LimitReader(input, limiteEntrada+1))
	if err != nil || len(datos) > limiteEntrada || !json.Valid(datos) || clavesUnicas(datos) != nil {
		return fallo("error_entrada")
	}
	var e entrada
	decoder := json.NewDecoder(bytes.NewReader(datos))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&e); err != nil {
		return fallo("error_entrada")
	}
	if e.Idioma != "" && len(mensajes[e.Idioma]) > 0 {
		idioma = e.Idioma
	}
	if (e.Esquema != esquemaEntrada && e.Esquema != esquemaEntradaPaginada) || !e.Sintetico || (e.Formato != "json" && e.Formato != "html") || (e.Idioma != "" && len(mensajes[e.Idioma]) == 0) {
		return fallo("error_entrada")
	}
	if e.Esquema == esquemaEntradaPaginada {
		if !reflect.DeepEqual(e.Antes, personal.InstantaneaComparacionOrganizacion{}) || !reflect.DeepEqual(e.Despues, personal.InstantaneaComparacionOrganizacion{}) {
			return fallo("error_entrada")
		}
		e.Antes, err = personal.ReunirPaginasOrganizacionHistorica(e.AntesPaginas)
		if err != nil {
			return fallo("error_entrada")
		}
		e.Despues, err = personal.ReunirPaginasOrganizacionHistorica(e.DespuesPaginas)
		if err != nil {
			return fallo("error_entrada")
		}
	} else if len(e.AntesPaginas) != 0 || len(e.DespuesPaginas) != 0 {
		return fallo("error_entrada")
	}
	resultado, err := personal.CompararOrganizacionHistorica(e.Antes, e.Despues)
	if err != nil {
		return fallo("error_entrada")
	}
	material := struct {
		Esquema        string                                      `json:"esquema"`
		Sintetico      bool                                        `json:"sintetico"`
		Antes          personal.InstantaneaComparacionOrganizacion `json:"antes"`
		Despues        personal.InstantaneaComparacionOrganizacion `json:"despues"`
		AntesPaginas   []personal.PaginaInstantaneaOrganizacion    `json:"antes_paginas,omitempty"`
		DespuesPaginas []personal.PaginaInstantaneaOrganizacion    `json:"despues_paginas,omitempty"`
	}{e.Esquema, e.Sintetico, e.Antes, e.Despues, e.AntesPaginas, e.DespuesPaginas}
	canonical, err := json.Marshal(material)
	if err != nil {
		return fallo("error_entrada")
	}
	m := manifiesto{Esquema: esquemaManifiesto, Sintetico: true, EntradaSHA256: huella(canonical), Comparacion: resultado, Antes: e.Antes, Despues: e.Despues, AntesPaginas: e.AntesPaginas, DespuesPaginas: e.DespuesPaginas}
	canonical, err = json.Marshal(m)
	if err != nil {
		return fallo("error_salida")
	}
	s := salida{m, huella(canonical), idioma, mensajes[idioma]}
	var b bytes.Buffer
	if e.Formato == "json" {
		encoder := json.NewEncoder(&b)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(s)
	} else {
		err = informeHTML(&b, e, s, catalogo)
	}
	if err != nil {
		return fallo("error_salida")
	}
	if _, err = output.Write(b.Bytes()); err != nil {
		return fallo("error_salida")
	}
	return 0
}

func huella(datos []byte) string { h := sha256.Sum256(datos); return hex.EncodeToString(h[:]) }

// Reject ambiguous duplicate object members, including nested ones. The standard
// decoder otherwise silently keeps the last value. Recursion is explicitly bounded.
func clavesUnicas(datos []byte) error {
	d := json.NewDecoder(bytes.NewReader(datos))
	var recorrer func(int) error
	recorrer = func(profundidad int) error {
		if profundidad > 64 {
			return io.ErrUnexpectedEOF
		}
		token, err := d.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		if delim == '{' {
			vistos := map[string]bool{}
			for d.More() {
				k, err := d.Token()
				if err != nil {
					return err
				}
				clave, ok := k.(string)
				if !ok || vistos[clave] {
					return io.ErrUnexpectedEOF
				}
				vistos[clave] = true
				if err = recorrer(profundidad + 1); err != nil {
					return err
				}
			}
		} else if delim == '[' {
			for d.More() {
				if err := recorrer(profundidad + 1); err != nil {
					return err
				}
			}
		}
		_, err = d.Token()
		return err
	}
	if err := recorrer(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func texto(c *i18n.Catalog, idioma, key string) string { return c.T(idioma, key) }

// Un error de salida se comunica al invocante mediante el estado del proceso.
// No incorpora valores recibidos ni el detalle técnico a un mensaje de usuario.
func codigoErrorSalida(err error) int {
	switch err {
	case nil:
		return 0
	default:
		return 2
	}
}

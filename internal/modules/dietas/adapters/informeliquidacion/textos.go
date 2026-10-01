package informeliquidacion

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"
)

var ErrTextos = errors.New("dietas_informe_textos_invalidos")

// Textos se inyecta desde un catálogo de datos del idioma elegido.
// Los mapas traducen códigos del dominio; ninguna clave se imprime como rótulo.
type Textos struct {
	Idioma    string            `json:"idioma"`
	Formato   Formato           `json:"formato"`
	Rotulos   Rotulos           `json:"rotulos"`
	Conceptos map[string]string `json:"conceptos"`
	Tipos     map[string]string `json:"tipos"`
	Motivos   map[string]string `json:"motivos"`
}

type Formato struct {
	Decimal    string `json:"decimal"`
	Agrupacion string `json:"agrupacion"`
	Moneda     string `json:"moneda"`
	Fecha      string `json:"fecha"`
}

type Rotulos struct {
	Titulo             string `json:"titulo"`
	Estado             string `json:"estado"`
	Limite             string `json:"limite"`
	Comision           string `json:"comision"`
	Grupo              string `json:"grupo"`
	ReferenciaComision string `json:"referencia_comision"`
	Version            string `json:"version"`
	Conceptos          string `json:"conceptos"`
	Concepto           string `json:"concepto"`
	Inicial            string `json:"inicial"`
	Propuesto          string `json:"propuesto"`
	Diferencia         string `json:"diferencia"`
	Motivo             string `json:"motivo"`
	Total              string `json:"total"`
	SinReduccion       string `json:"sin_reduccion"`
	Detalle            string `json:"detalle"`
	Catalogo           string `json:"catalogo"`
	Tarifa             string `json:"tarifa"`
	Fuentes            string `json:"fuentes"`
	DocumentoHuella    string `json:"documento_huella"`
	CatalogoHuella     string `json:"catalogo_huella"`
	PreparacionHuella  string `json:"preparacion_huella"`
	HuellaLimite       string `json:"huella_limite"`
	Reglas             string `json:"reglas"`
	Linea              string `json:"linea"`
	Regla              string `json:"regla"`
	TablaAyuda         string `json:"tabla_ayuda"`
}

var idiomaValido = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)

func CargarTextos(datos []byte) (Textos, error) {
	var t Textos
	if len(datos) > 65536 {
		return t, ErrTextos
	}
	d := json.NewDecoder(bytes.NewReader(datos))
	d.DisallowUnknownFields()
	if d.Decode(&t) != nil {
		return Textos{}, ErrTextos
	}
	var extra any
	if d.Decode(&extra) != io.EOF || t.validar() != nil {
		return Textos{}, ErrTextos
	}
	return t, nil
}

func (t Textos) validar() error {
	if !idiomaValido.MatchString(t.Idioma) || len(t.Idioma) > 35 || !separador(t.Formato.Decimal) || !separador(t.Formato.Agrupacion) || t.Formato.Decimal == t.Formato.Agrupacion || !texto(t.Formato.Moneda) || len(t.Formato.Moneda) > 12 || !texto(t.Formato.Fecha) || len(t.Formato.Fecha) > 32 {
		return ErrTextos
	}
	r := t.Rotulos
	for _, s := range []string{r.Titulo, r.Estado, r.Limite, r.Comision, r.Grupo, r.ReferenciaComision, r.Version, r.Conceptos, r.Concepto, r.Inicial, r.Propuesto, r.Diferencia, r.Motivo, r.Total, r.SinReduccion, r.Detalle, r.Catalogo, r.Tarifa, r.Fuentes, r.DocumentoHuella, r.CatalogoHuella, r.PreparacionHuella, r.HuellaLimite, r.Reglas, r.Linea, r.Regla, r.TablaAyuda} {
		if !texto(s) {
			return ErrTextos
		}
	}
	for _, dic := range []map[string]string{t.Conceptos, t.Tipos, t.Motivos} {
		if len(dic) == 0 || len(dic) > 128 {
			return ErrTextos
		}
		for k, s := range dic {
			if len(k) > 128 || !texto(k) || !texto(s) {
				return ErrTextos
			}
		}
	}
	return nil
}

func texto(s string) bool {
	return strings.TrimSpace(s) != "" && len(s) <= 1000 && !strings.ContainsAny(s, "\x00\r\n")
}
func separador(s string) bool { return s == "," || s == "." || s == " " || s == "\u00a0" }

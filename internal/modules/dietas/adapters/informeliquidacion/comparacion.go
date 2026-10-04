package informeliquidacion

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/dietas/application/preparacionliquidacion"
)

var ErrComparacionInforme = errors.New("dietas_informe_comparacion_invalida")

// TextosComparacion es un catálogo de datos; las claves fijan la semántica y
// el renderizador el orden de presentación.
type TextosComparacion struct {
	Idioma  string            `json:"idioma"`
	Formato Formato           `json:"formato"`
	Rotulos map[string]string `json:"rotulos"`
	Motivos map[string]string `json:"motivos"`
}

var rotulosComparacion = [...]string{
	"titulo", "estado", "limite", "importes", "rechazos", "tabla_ayuda", "linea", "original",
	"reconocido_antes", "reconocido_ahora", "cambio_reconocido", "rechazado_antes",
	"rechazado_ahora", "cambio_rechazado", "total", "criterios", "regla_antes",
	"regla_ahora", "motivo_antes", "motivo_ahora", "sin_motivo", "catalogos",
	"catalogos_distintos", "catalogos_iguales", "anterior", "propuesta", "referencia",
	"version", "huella_catalogo", "huella_propuesta", "huella_documento", "comision",
	"comision_version", "huella_limite",
}

// CargarTextosComparacion rechaza claves repetidas y cualquier estructura de
// catálogo que pueda cambiar el significado de una columna.
func CargarTextosComparacion(datos []byte) (TextosComparacion, error) {
	var t TextosComparacion
	if len(datos) == 0 || len(datos) > 65536 || !utf8.Valid(datos) {
		return t, ErrTextos
	}
	if err := objetosSinDuplicados(datos); err != nil {
		return t, ErrTextos
	}
	d := json.NewDecoder(bytes.NewReader(datos))
	d.DisallowUnknownFields()
	if d.Decode(&t) != nil || d.Decode(new(any)) != io.EOF || t.validar() != nil {
		return TextosComparacion{}, ErrTextos
	}
	return t, nil
}

func objetosSinDuplicados(datos []byte) error {
	d := json.NewDecoder(bytes.NewReader(datos))
	var valor func(int) error
	valor = func(profundidad int) error {
		if profundidad > 4 {
			return ErrTextos
		}
		tok, err := d.Token()
		if err != nil {
			return fmt.Errorf("%w: %w", ErrTextos, err)
		}
		switch tok := tok.(type) {
		case string:
			return nil
		case json.Delim:
			if tok != '{' {
				return ErrTextos
			}
			vistos := map[string]bool{}
			for d.More() {
				clave, err := d.Token()
				if err != nil {
					return fmt.Errorf("%w: %w", ErrTextos, err)
				}
				nombre, ok := clave.(string)
				if !ok || vistos[strings.ToLower(nombre)] {
					return ErrTextos
				}
				vistos[strings.ToLower(nombre)] = true
				if err := valor(profundidad + 1); err != nil {
					return err
				}
			}
			fin, err := d.Token()
			if err != nil {
				return fmt.Errorf("%w: %w", ErrTextos, err)
			}
			if fin != json.Delim('}') {
				return ErrTextos
			}
			return nil
		default:
			return ErrTextos
		}
	}
	if err := valor(0); err != nil {
		return err
	}
	_, err := d.Token()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTextos, err)
	}
	return ErrTextos
}

func (t TextosComparacion) validar() error {
	if !idiomaValido.MatchString(t.Idioma) || len(t.Idioma) > 35 ||
		!separador(t.Formato.Decimal) || !separador(t.Formato.Agrupacion) ||
		t.Formato.Decimal == t.Formato.Agrupacion || !texto(t.Formato.Moneda) ||
		len(t.Formato.Moneda) > 12 || !texto(t.Formato.Fecha) || len(t.Formato.Fecha) > 32 ||
		len(t.Rotulos) != len(rotulosComparacion) || len(t.Motivos) == 0 || len(t.Motivos) > 128 {
		return ErrTextos
	}
	for _, clave := range rotulosComparacion {
		if !texto(t.Rotulos[clave]) {
			return ErrTextos
		}
	}
	for clave, valor := range t.Motivos {
		if !texto(clave) || !texto(valor) {
			return ErrTextos
		}
	}
	return nil
}

type filaComparacionVista struct {
	Numero                                                       int
	Original, ReconocidoAntes, ReconocidoAhora, CambioReconocido string
	RechazadoAntes, RechazadoAhora, CambioRechazado              string
	ReglaAntes, ReglaAhora, MotivoAntes, MotivoAhora             string
}

type catalogoComparacionVista struct {
	Rotulo, Referencia, Version, CatalogoSHA, PropuestaSHA string
}

type vistaComparacion struct {
	Textos                                              TextosComparacion
	Tema, Estilos                                       template.CSS
	ComisionRef, Version, DocumentoSHA, CatalogosEstado string
	Filas                                               []filaComparacionVista
	Total                                               filaComparacionVista
	Catalogos                                           []catalogoComparacionVista
}

//go:embed comparacion.gohtml
var plantillaComparacion string

var informeComparacion = template.Must(template.New("comparacion").Parse(plantillaComparacion))

// RenderizarComparacion muestra el resultado ya verificado por Comparar. No
// vuelve a calcular importes ni atribuye aprobación a las huellas.
func (r *Renderizador) RenderizarComparacion(c *preparacionliquidacion.ComparacionLiquidacion, t TextosComparacion) ([]byte, error) {
	if r == nil || r.tema == "" {
		return nil, ErrTema
	}
	if c == nil || c.Esquema != preparacionliquidacion.EsquemaComparacionLiquidacion ||
		c.Procedencia != "comparacion_local_sin_registrar" || c.Liquidable ||
		len(c.Lineas) == 0 || len(c.Lineas) > 10000 {
		return nil, ErrComparacionInforme
	}
	if err := t.validar(); err != nil {
		return nil, err
	}
	motivo := func(codigo string) (string, error) {
		if codigo == "" {
			return t.Rotulos["sin_motivo"], nil
		}
		valor := t.Motivos[codigo]
		if valor == "" {
			return "", ErrTextos
		}
		return valor, nil
	}
	fila := func(numero int, i preparacionliquidacion.ImportesComparacion) filaComparacionVista {
		return filaComparacionVista{Numero: numero,
			Original:         moneda(i.OriginalCentimos, t.Formato),
			ReconocidoAntes:  moneda(i.ReconocidoAnteriorCentimos, t.Formato),
			ReconocidoAhora:  moneda(i.ReconocidoPropuestoCentimos, t.Formato),
			CambioReconocido: moneda(i.DiferenciaReconocidoCentimos, t.Formato),
			RechazadoAntes:   moneda(i.RechazadoAnteriorCentimos, t.Formato),
			RechazadoAhora:   moneda(i.RechazadoPropuestoCentimos, t.Formato),
			CambioRechazado:  moneda(i.DiferenciaRechazadoCentimos, t.Formato)}
	}
	v := vistaComparacion{Textos: t, Tema: r.tema, Estilos: template.CSS(estilos),
		ComisionRef: c.ComisionRef, Version: strconv.FormatInt(c.ComisionVersion, 10),
		DocumentoSHA: c.DocumentoSHA256, Total: fila(0, c.Totales),
		Catalogos: []catalogoComparacionVista{
			{t.Rotulos["anterior"], c.Anterior.CatalogoRef, c.Anterior.CatalogoVersion, c.Anterior.CatalogoSHA256, c.Anterior.SnapshotSHA256},
			{t.Rotulos["propuesta"], c.Propuesta.CatalogoRef, c.Propuesta.CatalogoVersion, c.Propuesta.CatalogoSHA256, c.Propuesta.SnapshotSHA256},
		}}
	if c.CatalogosDistintos {
		v.CatalogosEstado = t.Rotulos["catalogos_distintos"]
	} else {
		v.CatalogosEstado = t.Rotulos["catalogos_iguales"]
	}
	for _, linea := range c.Lineas {
		anterior, err := motivo(linea.MotivoAnteriorCodigo)
		if err != nil {
			return nil, err
		}
		propuesto, err := motivo(linea.MotivoPropuestoCodigo)
		if err != nil {
			return nil, err
		}
		f := fila(linea.Indice+1, linea.ImportesComparacion)
		f.ReglaAntes, f.ReglaAhora = linea.ReglaAnteriorRef, linea.ReglaPropuestaRef
		f.MotivoAntes, f.MotivoAhora = anterior, propuesto
		v.Filas = append(v.Filas, f)
	}
	var b bytes.Buffer
	if err := informeComparacion.Execute(&b, v); err != nil {
		return nil, ErrComparacionInforme
	}
	return b.Bytes(), nil
}

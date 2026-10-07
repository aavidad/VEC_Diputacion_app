package validadorautofirma

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// EsquemaDictamenV2SHA256 fija el esquema publicado por GrxFirma 0.0.107.
const EsquemaDictamenV2SHA256 = "e57e5cc0f47ad1ffdde30ff5ae8df3e0f1f93671f4714416ddce425e6142db6e"

//go:embed testdata/dictamen-verificacion-v2.schema.json
var esquemaV2JSON []byte

// esquemaV2 implementa solo las palabras clave presentes en el esquema fijado.
// No es un validador JSON Schema de uso general ni admite esquemas externos.
type esquemaV2 struct {
	Referencia   string                `json:"$ref"`
	Tipo         string                `json:"type"`
	Constante    json.RawMessage       `json:"const"`
	Enum         []json.RawMessage     `json:"enum"`
	Propiedades  map[string]*esquemaV2 `json:"properties"`
	Requeridas   []string              `json:"required"`
	Adicionales  *bool                 `json:"additionalProperties"`
	Elementos    *esquemaV2            `json:"items"`
	MinElementos *int                  `json:"minItems"`
	MaxElementos *int                  `json:"maxItems"`
	Minimo       *int64                `json:"minimum"`
	Maximo       *int64                `json:"maximum"`
	MaxLongitud  *int                  `json:"maxLength"`
	Patron       string                `json:"pattern"`
	Formato      string                `json:"format"`
	Definiciones map[string]*esquemaV2 `json:"$defs"`
	expresion    *regexp.Regexp
}

var esquemaPublicadoV2, errEsquemaPublicadoV2 = cargarEsquemaV2()

func cargarEsquemaV2() (*esquemaV2, error) {
	h := sha256.Sum256(esquemaV2JSON)
	if hex.EncodeToString(h[:]) != EsquemaDictamenV2SHA256 {
		return nil, errEstructuraRespuesta
	}
	var s esquemaV2
	if err := json.Unmarshal(esquemaV2JSON, &s); err != nil {
		return nil, err
	}
	var preparar func(*esquemaV2) error
	preparar = func(n *esquemaV2) error {
		if n == nil {
			return nil
		}
		if n.Patron != "" {
			var err error
			n.expresion, err = regexp.Compile(n.Patron)
			if err != nil {
				return err
			}
		}
		for _, p := range n.Propiedades {
			if err := preparar(p); err != nil {
				return err
			}
		}
		for _, p := range n.Definiciones {
			if err := preparar(p); err != nil {
				return err
			}
		}
		return preparar(n.Elementos)
	}
	if err := preparar(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

func valorJSON(raw []byte) (any, error) {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var v any
	if err := d.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

func coincideJSON(v any, raw json.RawMessage) (bool, error) {
	esperado, err := valorJSON(raw)
	if err != nil {
		return false, err
	}
	// Las constantes y enumeraciones del esquema son escalares.
	switch x := esperado.(type) {
	case nil:
		return v == nil, nil
	case string:
		actual, ok := v.(string)
		return ok && x == actual, nil
	case json.Number:
		actual, ok := v.(json.Number)
		if !ok {
			return false, nil
		}
		a, err := actual.Int64()
		if err != nil {
			return false, err
		}
		b, err := x.Int64()
		if err != nil {
			return false, err
		}
		return a == b, nil
	default:
		return false, nil
	}
}

func cumpleEsquemaV2(v any, n *esquemaV2, profundidad int) error {
	if n == nil || esquemaPublicadoV2 == nil || profundidad > maximaProfundidad {
		return errEstructuraRespuesta
	}
	if n.Referencia != "" {
		const prefijo = "#/$defs/"
		if !strings.HasPrefix(n.Referencia, prefijo) {
			return errEstructuraRespuesta
		}
		return cumpleEsquemaV2(v, esquemaPublicadoV2.Definiciones[strings.TrimPrefix(n.Referencia, prefijo)], profundidad+1)
	}
	if len(n.Constante) != 0 {
		coincide, err := coincideJSON(v, n.Constante)
		if err != nil {
			return err
		}
		if !coincide {
			return errEstructuraRespuesta
		}
	}
	if len(n.Enum) != 0 {
		coincide := false
		for _, e := range n.Enum {
			igual, err := coincideJSON(v, e)
			if err != nil {
				return err
			}
			if igual {
				coincide = true
				break
			}
		}
		if !coincide {
			return errEstructuraRespuesta
		}
	}
	switch n.Tipo {
	case "object":
		objeto, ok := v.(map[string]any)
		if !ok {
			return errEstructuraRespuesta
		}
		for _, requerida := range n.Requeridas {
			if _, ok := objeto[requerida]; !ok {
				return errEstructuraRespuesta
			}
		}
		for clave, valor := range objeto {
			propiedad, existe := n.Propiedades[clave]
			if !existe {
				if n.Adicionales != nil && !*n.Adicionales {
					return errEstructuraRespuesta
				}
				continue
			}
			if err := cumpleEsquemaV2(valor, propiedad, profundidad+1); err != nil {
				return err
			}
		}
	case "array":
		lista, ok := v.([]any)
		if !ok {
			return errEstructuraRespuesta
		}
		if n.MinElementos != nil && len(lista) < *n.MinElementos || n.MaxElementos != nil && len(lista) > *n.MaxElementos {
			return errEstructuraRespuesta
		}
		for _, e := range lista {
			if err := cumpleEsquemaV2(e, n.Elementos, profundidad+1); err != nil {
				return err
			}
		}
	case "string":
		s, ok := v.(string)
		if !ok {
			return errEstructuraRespuesta
		}
		if n.MaxLongitud != nil && utf8.RuneCountInString(s) > *n.MaxLongitud || n.expresion != nil && !n.expresion.MatchString(s) {
			return errEstructuraRespuesta
		}
		if n.Formato == "date-time" {
			if _, err := time.Parse(time.RFC3339Nano, s); err != nil {
				return err
			}
		}
	case "integer":
		numero, ok := v.(json.Number)
		if !ok {
			return errEstructuraRespuesta
		}
		i, err := numero.Int64()
		if err != nil {
			return err
		}
		if n.Minimo != nil && i < *n.Minimo || n.Maximo != nil && i > *n.Maximo {
			return errEstructuraRespuesta
		}
	case "boolean":
		if _, ok := v.(bool); !ok {
			return errEstructuraRespuesta
		}
	case "": // const y enum (incluido null) ya comprobados.
	default:
		return errEstructuraRespuesta
	}
	return nil
}

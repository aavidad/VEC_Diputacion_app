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

var esquemaPublicadoV2 = cargarEsquemaV2()

func cargarEsquemaV2() *esquemaV2 {
	h := sha256.Sum256(esquemaV2JSON)
	if hex.EncodeToString(h[:]) != EsquemaDictamenV2SHA256 {
		return nil
	}
	var s esquemaV2
	if json.Unmarshal(esquemaV2JSON, &s) != nil {
		return nil
	}
	var preparar func(*esquemaV2) bool
	preparar = func(n *esquemaV2) bool {
		if n == nil {
			return true
		}
		if n.Patron != "" {
			var err error
			n.expresion, err = regexp.Compile(n.Patron)
			if err != nil {
				return false
			}
		}
		for _, p := range n.Propiedades {
			if !preparar(p) {
				return false
			}
		}
		for _, p := range n.Definiciones {
			if !preparar(p) {
				return false
			}
		}
		return preparar(n.Elementos)
	}
	if !preparar(&s) {
		return nil
	}
	return &s
}

func valorJSON(raw []byte) (any, bool) {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var v any
	if d.Decode(&v) != nil {
		return nil, false
	}
	return v, true
}

func coincideJSON(v any, raw json.RawMessage) bool {
	esperado, ok := valorJSON(raw)
	if !ok {
		return false
	}
	// Las constantes y enumeraciones del esquema son escalares.
	switch x := esperado.(type) {
	case nil:
		return v == nil
	case string:
		actual, ok := v.(string)
		return ok && x == actual
	case json.Number:
		actual, ok := v.(json.Number)
		a, ea := actual.Int64()
		b, eb := x.Int64()
		return ok && ea == nil && eb == nil && a == b
	default:
		return false
	}
}

func cumpleEsquemaV2(v any, n *esquemaV2, profundidad int) bool {
	if n == nil || esquemaPublicadoV2 == nil || profundidad > maximaProfundidad {
		return false
	}
	if n.Referencia != "" {
		const prefijo = "#/$defs/"
		if !strings.HasPrefix(n.Referencia, prefijo) {
			return false
		}
		return cumpleEsquemaV2(v, esquemaPublicadoV2.Definiciones[strings.TrimPrefix(n.Referencia, prefijo)], profundidad+1)
	}
	if len(n.Constante) != 0 && !coincideJSON(v, n.Constante) {
		return false
	}
	if len(n.Enum) != 0 {
		coincide := false
		for _, e := range n.Enum {
			if coincideJSON(v, e) {
				coincide = true
				break
			}
		}
		if !coincide {
			return false
		}
	}
	switch n.Tipo {
	case "object":
		objeto, ok := v.(map[string]any)
		if !ok {
			return false
		}
		for _, requerida := range n.Requeridas {
			if _, ok := objeto[requerida]; !ok {
				return false
			}
		}
		for clave, valor := range objeto {
			propiedad, existe := n.Propiedades[clave]
			if !existe {
				if n.Adicionales != nil && !*n.Adicionales {
					return false
				}
				continue
			}
			if !cumpleEsquemaV2(valor, propiedad, profundidad+1) {
				return false
			}
		}
	case "array":
		lista, ok := v.([]any)
		if !ok {
			return false
		}
		if n.MinElementos != nil && len(lista) < *n.MinElementos || n.MaxElementos != nil && len(lista) > *n.MaxElementos {
			return false
		}
		for _, e := range lista {
			if !cumpleEsquemaV2(e, n.Elementos, profundidad+1) {
				return false
			}
		}
	case "string":
		s, ok := v.(string)
		if !ok {
			return false
		}
		if n.MaxLongitud != nil && utf8.RuneCountInString(s) > *n.MaxLongitud || n.expresion != nil && !n.expresion.MatchString(s) {
			return false
		}
		if n.Formato == "date-time" {
			if _, err := time.Parse(time.RFC3339Nano, s); err != nil {
				return false
			}
		}
	case "integer":
		numero, ok := v.(json.Number)
		if !ok {
			return false
		}
		i, err := numero.Int64()
		if err != nil {
			return false
		}
		if n.Minimo != nil && i < *n.Minimo || n.Maximo != nil && i > *n.Maximo {
			return false
		}
	case "boolean":
		if _, ok := v.(bool); !ok {
			return false
		}
	case "": // const y enum (incluido null) ya comprobados.
	default:
		return false
	}
	return true
}

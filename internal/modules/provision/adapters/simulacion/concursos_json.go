// Package simulacion traduce JSON local; no obtiene ni guarda datos de personas.
package simulacion

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"vec-diputacion-granada/internal/modules/provision/domain"
	"vec-diputacion-granada/internal/modules/provision/ports"
)

const MaximoBytes int64 = 2 << 20

//go:embed ejemplos/*.json
var ejemplosFS embed.FS

type Ejemplo struct {
	Referencia    string               `json:"referencia"`
	Configuracion domain.Configuracion `json:"configuracion"`
	Entrada       domain.Entrada       `json:"entrada"`
}

type SobreResultado struct {
	SchemaVersion string           `json:"schema_version"`
	Alcance       string           `json:"alcance"`
	Resultado     domain.Resultado `json:"resultado"`
}

func Sobre(r domain.Resultado) SobreResultado {
	return SobreResultado{"provision.simulacion.v1", "simulacion", r}
}

// Ejemplos vuelve a decodificar los datos públicos en cada llamada; editar una
// configuración recibida no puede modificar el ejemplo de otro consumidor.
func Ejemplos() ([]Ejemplo, error) {
	entradas, err := ejemplosFS.ReadDir("ejemplos")
	if err != nil {
		return nil, err
	}
	out := make([]Ejemplo, 0, len(entradas))
	for _, f := range entradas {
		datos, err := ejemplosFS.ReadFile("ejemplos/" + f.Name())
		if err != nil {
			return nil, err
		}
		var x Ejemplo
		if err := Decodificar(bytes.NewReader(datos), &x); err != nil {
			return nil, err
		}
		if err := domain.ValidarConfiguracion(x.Configuracion); err != nil {
			return nil, err
		}
		if err := domain.ValidarEntrada(x.Entrada); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, nil
}
func DecodificarPeticion(r io.Reader) (ports.PeticionSimulacion, error) {
	var p ports.PeticionSimulacion
	err := Decodificar(r, &p)
	return p, err
}

// Decodificar cierra tamaño, campos desconocidos, claves repetidas, profundidad
// y documentos concatenados. Los tipos shared cierran sus formas numéricas.
func Decodificar(r io.Reader, destino any) error {
	datos, err := io.ReadAll(io.LimitReader(r, MaximoBytes+1))
	if err != nil {
		return &domain.Error{Codigo: "lectura_json_fallida", Campo: "documento"}
	}
	if int64(len(datos)) > MaximoBytes {
		return &domain.Error{Codigo: "json_excesivo", Campo: "documento"}
	}
	tokens := json.NewDecoder(bytes.NewReader(datos))
	tokens.UseNumber()
	if err := validarTokens(tokens, 0); err != nil {
		return &domain.Error{Codigo: "json_invalido", Campo: "documento"}
	}
	if _, err := tokens.Token(); !errors.Is(err, io.EOF) {
		return &domain.Error{Codigo: "json_invalido", Campo: "documento"}
	}
	tipo := reflect.TypeOf(destino)
	if tipo == nil || tipo.Kind() != reflect.Pointer || reflect.ValueOf(destino).IsNil() {
		return &domain.Error{Codigo: "json_destino_invalido", Campo: "documento"}
	}
	if err := validarForma(datos, tipo.Elem()); err != nil {
		return &domain.Error{Codigo: "json_contrato_invalido", Campo: "documento"}
	}
	decoder := json.NewDecoder(bytes.NewReader(datos))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destino); err != nil {
		return &domain.Error{Codigo: "json_contrato_invalido", Campo: "documento"}
	}
	return nil
}
func validarTokens(d *json.Decoder, profundidad int) error {
	if profundidad > 24 {
		return errors.New("json_profundo")
	}
	tok, err := d.Token()
	if err != nil {
		return err
	}
	delim, esDelim := tok.(json.Delim)
	if !esDelim {
		return nil
	}
	switch delim {
	case '{':
		vistas := map[string]bool{}
		for d.More() {
			k, err := d.Token()
			if err != nil {
				return err
			}
			clave, ok := k.(string)
			if !ok || vistas[clave] {
				return errors.New("json_duplicado")
			}
			vistas[clave] = true
			if len(vistas) > 10000 {
				return errors.New("json_excesivo")
			}
			if err := validarTokens(d, profundidad+1); err != nil {
				return err
			}
		}
		cierre, err := d.Token()
		if err != nil || cierre != json.Delim('}') {
			return errors.New("json_invalido")
		}
	case '[':
		n := 0
		for d.More() {
			n++
			if n > 10000 {
				return errors.New("json_excesivo")
			}
			if err := validarTokens(d, profundidad+1); err != nil {
				return err
			}
		}
		cierre, err := d.Token()
		if err != nil || cierre != json.Delim(']') {
			return errors.New("json_invalido")
		}
	default:
		return errors.New("json_invalido")
	}
	return nil
}

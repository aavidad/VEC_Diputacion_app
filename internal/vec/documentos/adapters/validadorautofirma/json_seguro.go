package validadorautofirma

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
)

// sinClavesDuplicadas recorre el documento por tokens y exige un unico valor
// de nivel superior, profundidad acotada y claves unicas por objeto tras el
// plegado que aplica encoding/json al emparejar campos.
func sinClavesDuplicadas(contenido []byte) bool {
	lector := json.NewDecoder(bytes.NewReader(contenido))
	if recorrerValor(lector, 0) != nil {
		return false
	}
	_, err := lector.Token()
	return err == io.EOF
}

// validarJSONUnico conserva las causas de parseo hasta la frontera del cliente.
func validarJSONUnico(contenido []byte) error {
	lector := json.NewDecoder(bytes.NewReader(contenido))
	if err := recorrerValor(lector, 0); err != nil {
		return err
	}
	_, err := lector.Token()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%w: %w", errEstructuraRespuesta, err)
	}
	return errEstructuraRespuesta
}

// errEstructuraRespuesta es la causa cerrada de una respuesta cuyo recorrido
// por tokens no es admisible; cuando procede de encoding/json la conserva.
var errEstructuraRespuesta = errors.New("validadorautofirma: estructura de respuesta no admisible")

func recorrerValor(lector *json.Decoder, profundidad int) error {
	token, err := lector.Token()
	if err != nil {
		return fmt.Errorf("%w: %w", errEstructuraRespuesta, err)
	}
	delimitador, compuesto := token.(json.Delim)
	if !compuesto {
		return nil
	}
	if profundidad >= maximaProfundidad {
		return errEstructuraRespuesta
	}
	switch delimitador {
	case '{':
		vistas := make(map[string]struct{})
		for lector.More() {
			token, err := lector.Token()
			if err != nil {
				return fmt.Errorf("%w: %w", errEstructuraRespuesta, err)
			}
			clave, esClave := token.(string)
			if !esClave {
				return errEstructuraRespuesta
			}
			plegada := plegarClave(clave)
			if _, repetida := vistas[plegada]; repetida {
				return errEstructuraRespuesta
			}
			vistas[plegada] = struct{}{}
			if err := recorrerValor(lector, profundidad+1); err != nil {
				return err
			}
		}
	case '[':
		for lector.More() {
			if err := recorrerValor(lector, profundidad+1); err != nil {
				return err
			}
		}
	default:
		return errEstructuraRespuesta
	}
	if _, err = lector.Token(); err != nil {
		return fmt.Errorf("%w: %w", errEstructuraRespuesta, err)
	}
	return nil
}

// plegarClave reproduce el plegado sin distincion de mayusculas con el que
// encoding/json asocia claves a campos, para detectar duplicados equivalentes.
func plegarClave(clave string) string {
	return strings.Map(func(r rune) rune { return unicode.ToUpper(unicode.ToLower(r)) }, clave)
}

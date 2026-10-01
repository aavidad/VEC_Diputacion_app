// Package informemovimientos prepara únicamente ejemplos sintéticos de fichajes.
// No lee registros reales ni implementa un puerto de exportación autorizado.
package informemovimientos

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/language"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

var (
	ErrEjemploInvalido     = errors.New("cronos_informe_movimientos_ejemplo_invalido")
	ErrEjemploNoDisponible = errors.New("cronos_informe_movimientos_ejemplo_no_disponible")
)

type Catalogo struct {
	Referencia         string            `json:"referencia"`
	Version            string            `json:"version"`
	Idioma             string            `json:"idioma"`
	FormatoFecha       string            `json:"formato_fecha"`
	FormatoHora        string            `json:"formato_hora"`
	Titulo             string            `json:"titulo"`
	Sintetico          string            `json:"sintetico"`
	NombreSintetico    string            `json:"nombre_sintetico"`
	Periodo            string            `json:"periodo"`
	Zona               string            `json:"zona"`
	FuenteCompleta     string            `json:"fuente_completa"`
	FuenteIncompleta   string            `json:"fuente_incompleta"`
	Fila               string            `json:"fila"`
	OrigenSinVerificar string            `json:"origen_sin_verificar"`
	Vacio              string            `json:"vacio"`
	Limite             string            `json:"limite"`
	Movimientos        map[string]string `json:"movimientos"`
	Origenes           map[string]string `json:"origenes"`
}

// EjemploSintetico es entrada de ensayo; no es una fuente nominal ni auditada.
// MarcajeDia conserva solo los campos ya proyectados desde FuenteSaldo.Marcajes.
// Completo y Marcajes deben estar presentes, incluso para una lista vacía.
type EjemploSintetico struct {
	Demo        bool                       `json:"demo"`
	Nombre      string                     `json:"nombre"`
	Periodo     ports.PeriodoConsultaSaldo `json:"periodo"`
	ZonaHoraria string                     `json:"zona_horaria"`
	Completo    *bool                      `json:"completo"`
	Marcajes    []ports.MarcajeDia         `json:"marcajes"`
}

func LeerEjemploSintetico(datos io.Reader) (EjemploSintetico, error) {
	var e EjemploSintetico
	if err := leerJSON(datos, &e); err != nil {
		return EjemploSintetico{}, err
	}
	return e, nil
}

// Comparte los límites de catálogo/fixture de los CLI de Cronos: 64 KiB.
func leerJSON(datos io.Reader, destino any) error {
	if nulo(datos) {
		return ErrEjemploInvalido
	}
	raw, err := io.ReadAll(io.LimitReader(datos, 65537))
	if err != nil || len(raw) > 65536 || !utf8.Valid(raw) || validarClaves(raw) != nil {
		return ErrEjemploInvalido
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil || d.Decode(new(any)) != io.EOF {
		return ErrEjemploInvalido
	}
	return nil
}

// Las claves de este esquema son ASCII. Rechazar otras evita alias Unicode
// que encoding/json plegaría al mismo campo; las variantes ASCII se comparan
// sin mayúsculas. El límite de profundidad ya se usa en informes.
func validarClaves(raw []byte) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	var valor func(int) error
	valor = func(profundidad int) error {
		if profundidad > 8 {
			return ErrEjemploInvalido
		}
		t, err := d.Token()
		if err != nil {
			return ErrEjemploInvalido
		}
		delimitador, compuesto := t.(json.Delim)
		if !compuesto {
			return nil
		}
		switch delimitador {
		case '{':
			vistos := map[string]bool{}
			for d.More() {
				t, err := d.Token()
				k, ok := t.(string)
				if err != nil || !ok {
					return ErrEjemploInvalido
				}
				for _, caracter := range k {
					if caracter > unicode.MaxASCII {
						return ErrEjemploInvalido
					}
				}
				if vistos[strings.ToLower(k)] {
					return ErrEjemploInvalido
				}
				vistos[strings.ToLower(k)] = true
				if valor(profundidad+1) != nil {
					return ErrEjemploInvalido
				}
			}
			t, err := d.Token()
			if err != nil || t != json.Delim('}') {
				return ErrEjemploInvalido
			}
		case '[':
			for d.More() {
				if valor(profundidad+1) != nil {
					return ErrEjemploInvalido
				}
			}
			t, err := d.Token()
			if err != nil || t != json.Delim(']') {
				return ErrEjemploInvalido
			}
		default:
			return ErrEjemploInvalido
		}
		return nil
	}
	if valor(0) != nil {
		return ErrEjemploInvalido
	}
	if _, err := d.Token(); err != io.EOF {
		return ErrEjemploInvalido
	}
	return nil
}

func validarCatalogo(c Catalogo) (int64, error) {
	version, err := strconv.ParseInt(c.Version, 10, 64)
	if err != nil || version < 1 || strconv.FormatInt(version, 10) != c.Version {
		return 0, ErrEjemploInvalido
	}
	idioma, err := language.Parse(c.Idioma)
	if err != nil || idioma.String() != c.Idioma || !texto(c.Referencia, 512) || !texto(c.FormatoFecha, 32) || !texto(c.FormatoHora, 64) {
		return 0, ErrEjemploInvalido
	}
	// La hora local repetida en un cambio horario necesita su desfase numérico.
	if !strings.Contains(c.FormatoHora, "-07:00") && !strings.Contains(c.FormatoHora, "Z07:00") {
		return 0, ErrEjemploInvalido
	}
	for _, v := range []string{c.Titulo, c.Sintetico, c.NombreSintetico, c.Periodo, c.Zona, c.FuenteCompleta, c.FuenteIncompleta, c.Fila, c.OrigenSinVerificar, c.Vacio, c.Limite} {
		if !texto(v, 2048) {
			return 0, ErrEjemploInvalido
		}
	}
	if !mapa(c.Movimientos, []string{string(domain.PunchEntry), string(domain.PunchExit), string(domain.PunchPauseStart), string(domain.PunchPauseEnd)}) || !mapa(c.Origenes, []string{"terminal", "remoto"}) {
		return 0, ErrEjemploInvalido
	}
	for _, p := range []struct {
		texto  string
		claves []string
	}{
		{c.NombreSintetico, []string{"nombre"}}, {c.Periodo, []string{"desde", "hasta"}}, {c.Zona, []string{"zona"}}, {c.Fila, []string{"fecha", "hora", "movimiento", "origen"}},
	} {
		restante := p.texto
		for _, k := range p.claves {
			marcador := "{{" + k + "}}"
			if strings.Count(restante, marcador) != 1 {
				return 0, ErrEjemploInvalido
			}
			restante = strings.ReplaceAll(restante, marcador, "")
		}
		if strings.Contains(restante, "{{") || strings.Contains(restante, "}}") {
			return 0, ErrEjemploInvalido
		}
	}
	return version, nil
}
func mapa(m map[string]string, claves []string) bool {
	if len(m) != len(claves) {
		return false
	}
	for _, k := range claves {
		if !texto(m[k], 256) {
			return false
		}
	}
	return true
}
func texto(s string, max int) bool {
	if strings.TrimSpace(s) == "" || len(s) > max || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
func nulo(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return r.IsNil()
	}
	return false
}

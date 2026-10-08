// Package paquetestemas lee dos documentos locales separados; no lee recursos
// referenciados por ellos ni escribe archivos.
package paquetestemas

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/administracion/domain/temas"
)

var ErrEntrada = errors.New("temas.paquetes.error.entrada")
var ErrHuella = errors.New("temas.paquetes.error.huella_politica")
var claveJSON = regexp.MustCompile(`^[a-z_-][a-z0-9_.-]{0,127}$`)

type PoliticaValidada struct {
	datos  temas.Politica
	huella string
}

// Copia devuelve datos independientes para otro consumidor técnico. No concede
// autoridad; el valor cero continúa siendo una política inválida.
func (p PoliticaValidada) Copia() temas.Politica {
	c := p.datos
	c.Tokens = append([]string(nil), c.Tokens...)
	c.IdiomasRequeridos = append([]string(nil), c.IdiomasRequeridos...)
	c.Contrastes = append([]temas.ParContraste(nil), c.Contrastes...)
	c.ValoresFijos = make(map[string]string, len(p.datos.ValoresFijos))
	for k, v := range p.datos.ValoresFijos {
		c.ValoresFijos[k] = v
	}
	return c
}

type ContrasteCalculado struct {
	Variante    string  `json:"variante"`
	PrimerPlano string  `json:"primer_plano"`
	Fondo       string  `json:"fondo"`
	Razon       float64 `json:"razon"`
	Minimo      float64 `json:"minimo"`
}

type Resultado struct {
	Estado         string               `json:"estado"`
	OriginalSHA256 string               `json:"original_sha256"`
	CanonicoSHA256 string               `json:"canonico_sha256"`
	PoliticaSHA256 string               `json:"politica_sha256"`
	Material       temas.Paquete        `json:"material"`
	Contrastes     []ContrasteCalculado `json:"contrastes"`
}

func huella(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func leer(r io.Reader) ([]byte, error) {
	if r == nil {
		return nil, ErrEntrada
	}
	b, err := io.ReadAll(io.LimitReader(r, temas.MaxBytes+1))
	if err != nil || len(b) == 0 || len(b) > temas.MaxBytes || !utf8.Valid(b) {
		return nil, ErrEntrada
	}
	return b, nil
}

// caminar limita anidamiento y detecta claves repetidas incluso dentro de mapas.
func caminar(d *json.Decoder, profundidad int) error {
	if profundidad > temas.MaxProfundidad {
		return ErrEntrada
	}
	t, err := d.Token()
	if err != nil {
		return ErrEntrada
	}
	delim, esDelim := t.(json.Delim)
	if !esDelim {
		return nil
	}
	switch delim {
	case '{':
		claves := map[string]bool{}
		for d.More() {
			t, err := d.Token()
			if err != nil {
				return ErrEntrada
			}
			k, ok := t.(string)
			if !ok || !claveJSON.MatchString(k) || claves[k] {
				return ErrEntrada
			}
			claves[k] = true
			if caminar(d, profundidad+1) != nil {
				return ErrEntrada
			}
		}
	case '[':
		for d.More() {
			if caminar(d, profundidad+1) != nil {
				return ErrEntrada
			}
		}
	default:
		return ErrEntrada
	}
	_, err = d.Token()
	if err != nil {
		return ErrEntrada
	}
	return nil
}

func decodificar(b []byte, destino any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	if caminar(d, 1) != nil {
		return ErrEntrada
	}
	if _, err := d.Token(); err != io.EOF {
		return ErrEntrada
	}
	d = json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil {
		return ErrEntrada
	}
	return nil
}

// LeerPolitica coteja los bytes exactos con una huella elegida por configuración
// de confianza. El paquete no puede escoger ni proporcionar esa huella.
func LeerPolitica(r io.Reader, esperada string) (PoliticaValidada, error) {
	var vacia PoliticaValidada
	expected, err := hex.DecodeString(esperada)
	if err != nil || len(expected) != sha256.Size || esperada != hex.EncodeToString(expected) {
		return vacia, ErrHuella
	}
	b, err := leer(r)
	if err != nil {
		return vacia, err
	}
	if huella(b) != esperada {
		return vacia, ErrHuella
	}
	var p temas.Politica
	if decodificar(b, &p) != nil {
		return vacia, ErrEntrada
	}
	if p.Validar() != nil {
		return vacia, temas.ErrPolitica
	}
	return PoliticaValidada{datos: p, huella: esperada}, nil
}

func Preparar(r io.Reader, politica PoliticaValidada) (Resultado, error) {
	if politica.huella == "" || politica.datos.Validar() != nil {
		return Resultado{}, temas.ErrPolitica
	}
	b, err := leer(r)
	if err != nil {
		return Resultado{}, err
	}
	if len(b) > politica.datos.Limites.Bytes {
		return Resultado{}, ErrEntrada
	}
	var p temas.Paquete
	if decodificar(b, &p) != nil {
		return Resultado{}, ErrEntrada
	}
	normal, err := p.Normalizar(politica.datos)
	if err != nil {
		return Resultado{}, err
	}
	canonico, err := normal.Canonico()
	if err != nil {
		return Resultado{}, temas.ErrPaquete
	}
	informe := make([]ContrasteCalculado, 0, 2*len(politica.datos.Contrastes))
	for _, variante := range []struct {
		nombre  string
		colores map[string]string
	}{{"clara", normal.Variantes.Clara}, {"oscura", normal.Variantes.Oscura}} {
		for _, par := range politica.datos.Contrastes {
			informe = append(informe, ContrasteCalculado{variante.nombre, par.PrimerPlano, par.Fondo, temas.Contraste(variante.colores[par.PrimerPlano], variante.colores[par.Fondo]), par.Minimo})
		}
	}
	return Resultado{Estado: "validado_sin_instalar", OriginalSHA256: huella(b), CanonicoSHA256: huella(canonico), PoliticaSHA256: politica.huella, Material: normal, Contrastes: informe}, nil
}

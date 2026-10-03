// Package catalogoincidencias resuelve textos fijos del canal técnico.
// No clasifica incidencias ni concede permisos; esas reglas siguen en dominio.
package catalogoincidencias

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unicode"
	"unicode/utf8"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/web"
)

// MaxBytes conserva el límite de los catálogos y configuración de la CLI.
const MaxBytes = 1 << 20

// Catalogo mantiene una instantánea inmutable validada; un valor cero no sirve.
type Catalogo struct {
	plantillas map[domain.CodigoIncidenciaTecnica]string
}

func (c *Catalogo) Valido() bool {
	return c != nil && len(c.plantillas) == len(domain.CodigosIncidenciaTecnica())
}

func (c *Catalogo) Plantilla(codigo domain.CodigoIncidenciaTecnica) (string, bool) {
	if c == nil {
		return "", false
	}
	p, ok := c.plantillas[codigo]
	return p, ok
}

var predeterminado = sync.OnceValues(func() (*Catalogo, error) { return PorIdioma("") })

func Predeterminado() (*Catalogo, error) { return predeterminado() }

func PorIdioma(idioma string) (*Catalogo, error) {
	b, err := web.TextosIncidenciasTecnicas(idioma)
	if err != nil {
		return nil, os.ErrInvalid
	}
	return Cargar(bytes.NewReader(b))
}

// DesdeArchivo solo lee el archivo regular elegido por Sistemas. Nunca
// incorpora rutas ni errores del sistema al diagnóstico.
func DesdeArchivo(ruta string) (*Catalogo, error) {
	f, err := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0) // #nosec G304 -- explicit local catalogue, no leaf symlink, regular file, bounded read.
	if err != nil {
		return nil, os.ErrInvalid
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > MaxBytes {
		return nil, os.ErrInvalid
	}
	return Cargar(f)
}

// Cargar exige esquema, versión y los trece códigos exactos. No acepta
// sustituciones, controles, valores vacíos ni claves JSON ambiguas.
func Cargar(r io.Reader) (*Catalogo, error) {
	if r == nil {
		return nil, os.ErrInvalid
	}
	b, err := io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if err != nil || len(b) == 0 || len(b) > MaxBytes || !utf8.Valid(b) {
		return nil, os.ErrInvalid
	}
	defer clear(b)
	objeto, err := objetoExacto(b, []string{"esquema", "version_catalogo", "plantillas"})
	if err != nil {
		return nil, os.ErrInvalid
	}
	var esquema string
	var version string
	if json.Unmarshal(objeto["esquema"], &esquema) != nil || esquema != "1" ||
		json.Unmarshal(objeto["version_catalogo"], &version) != nil || version != strconv.Itoa(domain.VersionCatalogoIncidenciasTecnicas) {
		return nil, os.ErrInvalid
	}
	codigos := domain.CodigosIncidenciaTecnica()
	claves := make([]string, len(codigos))
	for i, codigo := range codigos {
		claves[i] = string(codigo)
	}
	plantillas, err := objetoExacto(objeto["plantillas"], claves)
	if err != nil {
		return nil, os.ErrInvalid
	}
	c := &Catalogo{plantillas: make(map[domain.CodigoIncidenciaTecnica]string, len(codigos))}
	for _, codigo := range codigos {
		var texto string
		if json.Unmarshal(plantillas[string(codigo)], &texto) != nil || texto == "" ||
			strings.TrimSpace(texto) != texto || strings.ContainsAny(texto, "%{}<>") ||
			strings.ContainsFunc(texto, unicode.IsControl) || strings.ContainsRune(texto, utf8.RuneError) {
			return nil, os.ErrInvalid
		}
		c.plantillas[codigo] = texto
	}
	return c, nil
}

func objetoExacto(b []byte, claves []string) (map[string]json.RawMessage, error) {
	d := json.NewDecoder(bytes.NewReader(b))
	t, err := d.Token()
	if err != nil || t != json.Delim('{') {
		return nil, os.ErrInvalid
	}
	campos := make(map[string]json.RawMessage, len(claves))
	for d.More() {
		t, err := d.Token()
		k, ok := t.(string)
		if err != nil || !ok {
			return nil, os.ErrInvalid
		}
		permitida := false
		for _, clave := range claves {
			permitida = permitida || clave == k
		}
		if _, repetida := campos[k]; !permitida || repetida {
			return nil, os.ErrInvalid
		}
		var v json.RawMessage
		if d.Decode(&v) != nil {
			return nil, os.ErrInvalid
		}
		campos[k] = v
	}
	t, err = d.Token()
	if err != nil || t != json.Delim('}') || len(campos) != len(claves) || d.Decode(new(any)) != io.EOF {
		return nil, os.ErrInvalid
	}
	return campos, nil
}

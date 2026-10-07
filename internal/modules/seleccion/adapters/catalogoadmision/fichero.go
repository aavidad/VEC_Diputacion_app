// Package catalogoadmision lee de un fichero de configuración los catálogos
// versionados de motivos de exclusión y plazo de subsanación. RRHH los cambia
// sin tocar código: cada cambio es una versión nueva en el mismo fichero.
package catalogoadmision

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"syscall"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

const maximoFichero = 256 * 1024

var (
	ErrConfiguracion = errors.New("seleccion.catalogo_admision.configuracion_invalida")
	ErrNoEncontrado  = errors.New("seleccion.catalogo_admision.version_no_encontrada")
)

type fichero struct {
	Catalogos []domain.CatalogoAdmision `json:"catalogos"`
}

// Fichero conserva en memoria las versiones validadas al cargar.
type Fichero struct {
	versiones map[[2]string]domain.CatalogoAdmision
}

var _ ports.CatalogoAdmision = (*Fichero)(nil)

// Cargar lee nombre dentro de dir sin seguir rutas fuera de esa raíz. Falla
// si el fichero no es regular, supera el límite, tiene campos desconocidos,
// alguna versión inválida o una referencia y versión repetidas.
func Cargar(dir, nombre string) (*Fichero, error) {
	raiz, err := os.OpenRoot(dir)
	if err != nil {
		return nil, ErrConfiguracion
	}
	defer raiz.Close()
	f, err := raiz.OpenFile(nombre, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, ErrConfiguracion
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maximoFichero {
		return nil, ErrConfiguracion
	}
	raw, err := io.ReadAll(io.LimitReader(f, maximoFichero+1))
	if err != nil || len(raw) > maximoFichero {
		return nil, ErrConfiguracion
	}
	return Leer(raw)
}

// Leer valida el contenido ya leído; Cargar la usa tras abrir el fichero.
func Leer(raw []byte) (*Fichero, error) {
	var contenido fichero
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&contenido) != nil || d.More() || len(contenido.Catalogos) == 0 || len(contenido.Catalogos) > 256 {
		return nil, ErrConfiguracion
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, ErrConfiguracion
	}
	out := &Fichero{versiones: make(map[[2]string]domain.CatalogoAdmision, len(contenido.Catalogos))}
	for _, c := range contenido.Catalogos {
		clave := [2]string{c.Referencia, c.Version}
		if _, repetida := out.versiones[clave]; repetida || c.Validar() != nil {
			return nil, ErrConfiguracion
		}
		c.Motivos = append([]domain.MotivoExclusion{}, c.Motivos...)
		out.versiones[clave] = c
	}
	return out, nil
}

func (f *Fichero) CatalogoAdmision(ctx context.Context, referencia, version string) (domain.CatalogoAdmision, error) {
	if f == nil || ctx == nil {
		return domain.CatalogoAdmision{}, ErrConfiguracion
	}
	if err := ctx.Err(); err != nil {
		return domain.CatalogoAdmision{}, err
	}
	c, ok := f.versiones[[2]string{referencia, version}]
	if !ok {
		return domain.CatalogoAdmision{}, ErrNoEncontrado
	}
	c.Motivos = append([]domain.MotivoExclusion{}, c.Motivos...)
	return c, nil
}

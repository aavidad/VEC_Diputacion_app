// Package catalogo lee el catálogo versionado de convocatorias de Selección
// (plazo, turnos, requisitos estructurados, baremo y formato del
// justificante). Las reglas se cambian editando el catálogo, nunca el código.
package catalogo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"time"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
	"vec-diputacion-granada/internal/shared/baremacion"
)

const (
	// IdentificadorCatalogo del catálogo de convocatorias de Selección.
	IdentificadorCatalogo = "vec.seleccion.convocatorias"
	// FuentePaqueteEjemplo marca las reglas inventadas y retirables.
	FuentePaqueteEjemplo = "paquete:ejemplo:vec:v1"
	tamanoMaximo         = 1 << 20
)

// ErrCatalogoInvalido impide arrancar con un catálogo ilegible o incoherente.
var ErrCatalogoInvalido = errors.New("seleccion: catalogo de convocatorias invalido")

type archivo struct {
	VersionEsquema int `json:"version_esquema"`
	Fuente         struct {
		Revision      string    `json:"revision"`
		ActualizadaEn time.Time `json:"actualizada_en"`
		Demostracion  bool      `json:"demostracion"`
		Aviso         string    `json:"aviso"`
	} `json:"fuente"`
	Catalogo struct {
		ID             string `json:"id"`
		Version        int    `json:"version"`
		Revision       int    `json:"revision"`
		ModuloID       string `json:"modulo_id"`
		Nombre         string `json:"nombre"`
		FuenteRef      string `json:"fuente_ref"`
		MotivoCreacion string `json:"motivo_creacion"`
	} `json:"catalogo"`
	Convocatorias []convocatoriaArchivo `json:"convocatorias"`
}

type meritoArchivo struct {
	Clave           string `json:"clave"`
	Titulo          string `json:"titulo"`
	Unidad          string `json:"unidad"`
	PuntosPorUnidad string `json:"puntos_por_unidad"`
	Maximo          string `json:"maximo"`
}

type convocatoriaArchivo struct {
	Ref         string    `json:"convocatoria_ref"`
	Titulo      string    `json:"titulo"`
	PublicadaEn time.Time `json:"publicada_en"`
	Plazo       struct {
		AbreEn   time.Time `json:"abre_en"`
		CierraEn time.Time `json:"cierra_en"`
	} `json:"plazo"`
	FechaReferencia string             `json:"fecha_referencia"`
	Turnos          []domain.Turno     `json:"turnos"`
	Requisitos      []domain.Requisito `json:"requisitos"`
	Baremo          struct {
		Maximo   string `json:"maximo"`
		Redondeo string `json:"redondeo"`
		Grupos   []struct {
			Clave   string          `json:"clave"`
			Titulo  string          `json:"titulo"`
			Maximo  string          `json:"maximo"`
			Meritos []meritoArchivo `json:"meritos"`
		} `json:"grupos"`
	} `json:"baremo"`
	Numeracion domain.Numeracion `json:"numeracion"`
}

// Fichero es la fuente de convocatorias leída de un fichero JSON. Se lee y
// valida al construirse: un catálogo roto impide arrancar.
type Fichero struct {
	convocatorias []domain.Convocatoria
}

var _ ports.FuenteConvocatorias = (*Fichero)(nil)

// NuevoFichero lee y valida el catálogo.
func NuevoFichero(ruta string) (*Fichero, error) {
	f, err := os.Open(ruta)
	if err != nil {
		return nil, errors.Join(ErrCatalogoInvalido, err)
	}
	defer f.Close()
	bruto, err := io.ReadAll(io.LimitReader(f, tamanoMaximo+1))
	if err != nil || len(bruto) > tamanoMaximo {
		return nil, errors.Join(ErrCatalogoInvalido, err)
	}
	return Parsear(bruto)
}

// Parsear valida el documento del catálogo.
func Parsear(bruto []byte) (*Fichero, error) {
	var a archivo
	decodificador := json.NewDecoder(bytes.NewReader(bruto))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&a); err != nil || decodificador.More() {
		return nil, errors.Join(ErrCatalogoInvalido, err)
	}
	if a.VersionEsquema != 1 || a.Catalogo.ID != IdentificadorCatalogo || a.Catalogo.ModuloID != "seleccion" || a.Catalogo.Version < 1 ||
		a.Catalogo.Revision < 1 || a.Fuente.Revision == "" || len(a.Convocatorias) == 0 || len(a.Convocatorias) > 200 {
		return nil, ErrCatalogoInvalido
	}
	// Un paquete de ejemplo debe declararse como demostración.
	ejemplo := a.Catalogo.FuenteRef == FuentePaqueteEjemplo
	if ejemplo != a.Fuente.Demostracion {
		return nil, ErrCatalogoInvalido
	}
	resultado := &Fichero{convocatorias: make([]domain.Convocatoria, 0, len(a.Convocatorias))}
	vistas := map[string]bool{}
	for _, c := range a.Convocatorias {
		convocatoria, err := c.aDominio(ejemplo)
		if err != nil || vistas[convocatoria.Ref] {
			return nil, errors.Join(ErrCatalogoInvalido, err)
		}
		vistas[convocatoria.Ref] = true
		resultado.convocatorias = append(resultado.convocatorias, convocatoria)
	}
	return resultado, nil
}

func (c convocatoriaArchivo) aDominio(ejemplo bool) (domain.Convocatoria, error) {
	d := domain.Convocatoria{Ref: c.Ref, Titulo: c.Titulo, AbreEn: c.Plazo.AbreEn.UTC(), CierraEn: c.Plazo.CierraEn.UTC(),
		FechaReferencia: c.FechaReferencia, Turnos: c.Turnos, Requisitos: c.Requisitos, Numeracion: c.Numeracion,
		MarcaEjemplo: ejemplo, PublicadaEn: c.PublicadaEn.UTC()}
	var err error
	if d.Baremo.Maximo, err = domain.ParsearPuntos(c.Baremo.Maximo); err != nil {
		return domain.Convocatoria{}, err
	}
	d.Baremo.Redondeo = baremacion.ModoRedondeo(c.Baremo.Redondeo)
	for _, g := range c.Baremo.Grupos {
		grupo := domain.GrupoBaremo{Clave: g.Clave, Titulo: g.Titulo}
		if grupo.Maximo, err = domain.ParsearPuntos(g.Maximo); err != nil {
			return domain.Convocatoria{}, err
		}
		for _, m := range g.Meritos {
			merito := domain.MeritoBaremo{Clave: m.Clave, Titulo: m.Titulo, Unidad: m.Unidad}
			if merito.PuntosPorUnidad, err = domain.ParsearPuntos(m.PuntosPorUnidad); err != nil {
				return domain.Convocatoria{}, err
			}
			if merito.Maximo, err = domain.ParsearPuntos(m.Maximo); err != nil {
				return domain.Convocatoria{}, err
			}
			grupo.Meritos = append(grupo.Meritos, merito)
		}
		d.Baremo.Grupos = append(d.Baremo.Grupos, grupo)
	}
	return d, d.Validar()
}

// Convocatorias devuelve una copia de las convocatorias del catálogo.
func (f *Fichero) Convocatorias(ctx context.Context) ([]domain.Convocatoria, error) {
	if f == nil || ctx == nil {
		return nil, ports.ErrNoDisponible
	}
	return append([]domain.Convocatoria(nil), f.convocatorias...), nil
}

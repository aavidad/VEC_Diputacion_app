package inscripcion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"sort"
	"time"

	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// Solicitud conjunta (F1-41): una sola presentación y un solo recibo para
// varias categorías de una convocatoria cuya modalidad sellada es conjunta.
// Cada categoría conserva su propia solicitud, autorizada por separado, para
// que RRHH la revise e incorpore como una solicitud por categoría.

const EsquemaMaterialPresentacionConjunta = "vec.bolsa.inscripcion.presentar_conjunta.v1"

const maximoCategoriasConjunta = 128

// categoriaConjunta es la misma forma que exige la SQL (B146/BC13): se rechaza
// aquí, antes de pedir una concesión por categoría.
var categoriaConjunta = regexp.MustCompile(`^[a-z][a-z0-9_.:-]{2,127}$`)

type PresentacionConjunta struct {
	ConvocatoriaRef   string
	CategoriasRefs    []string
	CatalogoVersion   uint64
	ClaveIdempotencia string
	Declaraciones     []Declaracion
}

// Validar exige categorías distintas; el orden lo fija el material canónico.
func (p PresentacionConjunta) Validar() error {
	if len(p.CategoriasRefs) == 0 || len(p.CategoriasRefs) > maximoCategoriasConjunta {
		return ErrSolicitudInvalida
	}
	vistas := make(map[string]struct{}, len(p.CategoriasRefs))
	for _, categoria := range p.CategoriasRefs {
		if _, repetida := vistas[categoria]; repetida {
			return ErrSolicitudInvalida
		}
		vistas[categoria] = struct{}{}
		if !categoriaConjunta.MatchString(categoria) || p.PorCategoria(categoria).Validar() != nil {
			return ErrSolicitudInvalida
		}
	}
	return nil
}

// PorCategoria es la solicitud de una categoría dentro de la conjunta: mismo
// recurso que la solicitud separada, ligado al material conjunto.
func (p PresentacionConjunta) PorCategoria(categoria string) Presentacion {
	return Presentacion{ConvocatoriaRef: p.ConvocatoriaRef, CategoriaRef: categoria,
		CatalogoVersion: p.CatalogoVersion, ClaveIdempotencia: p.ClaveIdempotencia,
		Declaraciones: p.Declaraciones}
}

// Categorias devuelve las categorías en el orden canónico del material.
func (p PresentacionConjunta) Categorias() []string {
	categorias := append([]string{}, p.CategoriasRefs...)
	sort.Strings(categorias)
	return categorias
}

func MaterialPresentacionConjunta(p PresentacionConjunta) ([]byte, string, error) {
	if err := p.Validar(); err != nil {
		return nil, "", err
	}
	declaraciones := append([]Declaracion{}, p.Declaraciones...)
	sort.Slice(declaraciones, func(i, j int) bool { return declaraciones[i].RequisitoCodigo < declaraciones[j].RequisitoCodigo })
	contenido, err := json.Marshal(struct {
		Esquema           string        `json:"esquema"`
		ConvocatoriaRef   string        `json:"convocatoria_ref"`
		CategoriasRefs    []string      `json:"categorias_refs"`
		CatalogoVersion   uint64        `json:"catalogo_version"`
		ClaveIdempotencia string        `json:"clave_idempotencia"`
		Declaraciones     []Declaracion `json:"declaraciones"`
	}{EsquemaMaterialPresentacionConjunta, p.ConvocatoriaRef, p.Categorias(), p.CatalogoVersion,
		p.ClaveIdempotencia, declaraciones})
	if err != nil {
		return nil, "", ErrSolicitudInvalida
	}
	huella := sha256.Sum256(contenido)
	return contenido, hex.EncodeToString(huella[:]), nil
}

// PresentacionConjuntaDesdeMaterial reconstruye el comando y solo lo acepta si
// vuelve a producir exactamente los mismos bytes.
func PresentacionConjuntaDesdeMaterial(material []byte) (PresentacionConjunta, bool) {
	p, err := presentacionConjuntaDesdeMaterial(material)
	return p, err == nil
}

func presentacionConjuntaDesdeMaterial(material []byte) (PresentacionConjunta, error) {
	var v struct {
		Esquema           string        `json:"esquema"`
		ConvocatoriaRef   string        `json:"convocatoria_ref"`
		CategoriasRefs    []string      `json:"categorias_refs"`
		CatalogoVersion   uint64        `json:"catalogo_version"`
		ClaveIdempotencia string        `json:"clave_idempotencia"`
		Declaraciones     []Declaracion `json:"declaraciones"`
	}
	if len(material) == 0 || len(material) > 64*1024 || json.Unmarshal(material, &v) != nil ||
		v.Esquema != EsquemaMaterialPresentacionConjunta {
		return PresentacionConjunta{}, ErrSolicitudInvalida
	}
	p := PresentacionConjunta{ConvocatoriaRef: v.ConvocatoriaRef, CategoriasRefs: v.CategoriasRefs,
		CatalogoVersion: v.CatalogoVersion, ClaveIdempotencia: v.ClaveIdempotencia, Declaraciones: v.Declaraciones}
	esperado, _, err := MaterialPresentacionConjunta(p)
	if err != nil {
		return PresentacionConjunta{}, err
	}
	if !bytes.Equal(esperado, material) {
		return PresentacionConjunta{}, ErrSolicitudInvalida
	}
	return p, nil
}

// SolicitudDeCategoria localiza la categoría cuya solicitud es ref.
func (p PresentacionConjunta) SolicitudDeCategoria(personaRef, ref string) (Presentacion, bool) {
	for _, categoria := range p.Categorias() {
		porCategoria := p.PorCategoria(categoria)
		if esperada, err := ReferenciaSolicitud(personaRef, porCategoria); err == nil && esperada == ref {
			return porCategoria, true
		}
	}
	return Presentacion{}, false
}

// EscrituraAutorizada es la concesión V3 de una categoría y su recurso exacto.
type EscrituraAutorizada struct {
	Material        puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
	RecursoCanonico []byte
}

// EscriturasConjuntasValidas exige una concesión por categoría, en el orden
// canónico, ligada a su solicitud y a su recurso.
func (a Actor) EscriturasConjuntasValidas(p PresentacionConjunta) bool {
	return a.validarEscriturasConjuntas(p) == nil
}

func (a Actor) validarEscriturasConjuntas(p PresentacionConjunta) error {
	categorias := p.Categorias()
	if !a.Valido() || a.MaterialEscritura != nil || len(a.RecursoEscrituraCanonico) != 0 ||
		p.Validar() != nil || len(a.EscriturasConjuntas) != len(categorias) {
		return ErrAccesoDenegado
	}
	for i, categoria := range categorias {
		e := a.EscriturasConjuntas[i]
		ref, err := ReferenciaSolicitud(a.PersonaRef, p.PorCategoria(categoria))
		if err != nil {
			return err
		}
		if e.Material.ValidarEstructura() != nil || len(e.RecursoCanonico) == 0 || len(e.RecursoCanonico) > 8192 {
			return ErrAccesoDenegado
		}
		huella := sha256.Sum256(e.RecursoCanonico)
		resumen := e.Material.ResumenCapacidad()
		if resumen.EfectoRef() != ref || resumen.EfectoHuellaSHA256() != hex.EncodeToString(huella[:]) ||
			resumen.Operacion() != AccionPresentar {
			return ErrAccesoDenegado
		}
	}
	return nil
}

type CategoriaReciboConjunto struct {
	CategoriaRef   string `json:"categoria_ref"`
	Categoria      string `json:"categoria"`
	SolicitudRef   string `json:"solicitud_ref"`
	ReciboRef      string `json:"recibo_ref"`
	DeclaracionRef string `json:"declaracion_ref"`
}

type ReciboConjunto struct {
	PresentacionRef string                    `json:"presentacion_ref"`
	ReciboRef       string                    `json:"recibo_ref"`
	ConvocatoriaRef string                    `json:"convocatoria_ref"`
	BasesRef        string                    `json:"bases_ref"`
	Estado          string                    `json:"estado"`
	Version         uint64                    `json:"version"`
	RegistradaEn    time.Time                 `json:"registrada_en"`
	Repetida        bool                      `json:"repetida"`
	Categorias      []CategoriaReciboConjunto `json:"categorias"`
}

// Validar coteja el recibo con la presentación pedida, categoría a categoría.
func (r ReciboConjunto) Validar(personaRef string, p PresentacionConjunta) error {
	categorias := p.Categorias()
	if !referenciaOpaca.MatchString(r.PresentacionRef) || !referenciaOpaca.MatchString(r.ReciboRef) ||
		r.ConvocatoriaRef != p.ConvocatoriaRef || r.BasesRef == "" || r.Estado != EstadoPendiente ||
		r.Version != 1 || r.RegistradaEn.IsZero() || len(r.Categorias) != len(categorias) {
		return ErrSolicitudInvalida
	}
	for i, c := range r.Categorias {
		ref, err := ReferenciaSolicitud(personaRef, p.PorCategoria(categorias[i]))
		if err != nil || c.CategoriaRef != categorias[i] || c.SolicitudRef != ref ||
			c.Categoria == "" || len(c.Categoria) > 2048 ||
			!referenciaOpaca.MatchString(c.ReciboRef) || !referenciaOpaca.MatchString(c.DeclaracionRef) {
			return ErrSolicitudInvalida
		}
	}
	return nil
}

func (s *Servicio) PresentarConjunta(ctx context.Context, actor Actor, p PresentacionConjunta) (ReciboConjunto, error) {
	if s == nil || s.repositorio == nil {
		return ReciboConjunto{}, ErrNoDisponible
	}
	if p.Validar() != nil || !actor.EscriturasConjuntasValidas(p) {
		return ReciboConjunto{}, ErrSolicitudInvalida
	}
	r, err := s.repositorio.PresentarConjunta(ctx, actor, p)
	if err != nil {
		return ReciboConjunto{}, err
	}
	if r.Validar(actor.PersonaRef, p) != nil {
		return ReciboConjunto{}, ErrNoDisponible
	}
	return r, nil
}

// Package conservacion resuelve la política de conservación documental desde
// un catálogo versionado local, sin depender de otra aplicación. El catálogo
// v1 es PROVISIONAL: sus plazos son valores de desarrollo rotulados como tales
// hasta que RRHH fije los reales (dudas.md, pregunta 60). Cuando exista una
// autoridad documental corporativa bastará con otro adaptador del mismo puerto.
package conservacion

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/vec/ports"
)

//go:embed politicas_conservacion.v1.json
var catalogoV1 []byte

var (
	ErrCatalogoInvalido  = errors.New("vec: catálogo de conservación documental inválido")
	ErrTipoNoCatalogado  = errors.New("vec: tipo documental sin política de conservación")
	claveCatalogo        = regexp.MustCompile(`^[a-z][a-z0-9_.]{2,127}$`)
	maximoEntradas       = 256
	maximoPlazoAnios     = 100
	dominioReferencias   = "vec.documentos.conservacion.v1"
	catalogoEsperado     = "vec.documentos.conservacion"
	versionCatalogoLocal = uint64(1)
)

type entradaJSON struct {
	Tipo          string `json:"tipo"`
	Procedimiento string `json:"procedimiento"`
	Serie         string `json:"serie"`
	BaseJuridica  string `json:"base_juridica"`
	PlazoAnios    int    `json:"plazo_anios"`
	Proteccion    string `json:"proteccion"`
}

type catalogoJSON struct {
	Catalogo     string        `json:"catalogo"`
	Version      uint64        `json:"version"`
	Provisional  bool          `json:"provisional"`
	Rotulo       string        `json:"rotulo"`
	VigenteDesde time.Time     `json:"vigente_desde"`
	VigenteHasta time.Time     `json:"vigente_hasta"`
	Politicas    []entradaJSON `json:"politicas"`
}

type entrada struct {
	tipo, tipoRef, procedimientoRef, serieRef, baseRef, politicaRef string
	huella                                                          [sha256.Size]byte
	plazoAnios                                                      int
}

// Catalogo es inmutable tras construirse; es seguro entre goroutines.
type Catalogo struct {
	version      uint64
	provisional  bool
	vigenteDesde time.Time
	vigenteHasta time.Time
	porTipoRef   map[string]entrada
	porTipo      map[string]string
	reloj        ports.Reloj
}

var _ ports.ResolutorPoliticaConservacionDocumental = (*Catalogo)(nil)

// NuevoCatalogoProvisional carga el catálogo v1 versionado en Git.
func NuevoCatalogoProvisional(reloj ports.Reloj) (*Catalogo, error) {
	return NuevoCatalogo(catalogoV1, reloj)
}

// NuevoCatalogo valida estrictamente un catálogo: campos cerrados, claves
// técnicas, plazos acotados, vigencia en UTC y tipos no repetidos.
func NuevoCatalogo(raw []byte, reloj ports.Reloj) (*Catalogo, error) {
	if reloj == nil || len(raw) == 0 || len(raw) > 256<<10 {
		return nil, ErrCatalogoInvalido
	}
	var c catalogoJSON
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var extra any
	if dec.Decode(&c) != nil || !errors.Is(dec.Decode(&extra), io.EOF) ||
		c.Catalogo != catalogoEsperado || c.Version != versionCatalogoLocal || c.Rotulo == "" ||
		len(c.Politicas) == 0 || len(c.Politicas) > maximoEntradas ||
		c.VigenteDesde.Location() != time.UTC || c.VigenteHasta.Location() != time.UTC ||
		c.VigenteDesde.Nanosecond()%1000 != 0 || c.VigenteHasta.Nanosecond()%1000 != 0 ||
		!c.VigenteDesde.Before(c.VigenteHasta) {
		return nil, ErrCatalogoInvalido
	}
	cat := &Catalogo{version: c.Version, provisional: c.Provisional, vigenteDesde: c.VigenteDesde,
		vigenteHasta: c.VigenteHasta, porTipoRef: map[string]entrada{}, porTipo: map[string]string{}, reloj: reloj}
	for _, e := range c.Politicas {
		// Solo protección ordinaria: un bloqueo exige referencia de la orden
		// que lo impone, que no puede proceder de un catálogo estático.
		if !claveCatalogo.MatchString(e.Tipo) || !claveCatalogo.MatchString(e.Procedimiento) ||
			!claveCatalogo.MatchString(e.Serie) || !claveCatalogo.MatchString(e.BaseJuridica) ||
			e.PlazoAnios < 1 || e.PlazoAnios > maximoPlazoAnios ||
			e.Proteccion != string(ports.ProteccionPoliticaConservacionDocumentalOrdinaria) {
			return nil, ErrCatalogoInvalido
		}
		if _, repetido := cat.porTipo[e.Tipo]; repetido {
			return nil, ErrCatalogoInvalido
		}
		canon, err := json.Marshal(struct {
			Catalogo string      `json:"catalogo"`
			Version  uint64      `json:"version"`
			Desde    time.Time   `json:"vigente_desde"`
			Hasta    time.Time   `json:"vigente_hasta"`
			Entrada  entradaJSON `json:"entrada"`
		}{c.Catalogo, c.Version, c.VigenteDesde, c.VigenteHasta, e})
		if err != nil {
			return nil, ErrCatalogoInvalido
		}
		en := entrada{
			tipo: e.Tipo, tipoRef: referencia("tipo", e.Tipo), procedimientoRef: referencia("procedimiento", e.Procedimiento),
			serieRef: referencia("serie", e.Serie), baseRef: referencia("base_juridica", e.BaseJuridica),
			politicaRef: referencia("politica", e.Tipo+"\x00v1"), huella: sha256.Sum256(canon), plazoAnios: e.PlazoAnios,
		}
		cat.porTipo[e.Tipo] = en.tipoRef
		cat.porTipoRef[en.tipoRef] = en
	}
	return cat, nil
}

func referencia(campo, valor string) string {
	suma := sha256.Sum256([]byte(dominioReferencias + "\x00" + campo + "\x00" + valor))
	return "ref:" + hex.EncodeToString(suma[:])
}

// Provisional indica que los plazos no son una política aprobada por RRHH.
func (c *Catalogo) Provisional() bool { return c != nil && c.provisional }

// TipoDocumentalRef devuelve la referencia opaca de un tipo catalogado.
func (c *Catalogo) TipoDocumentalRef(tipo string) (string, error) {
	if c == nil {
		return "", ErrCatalogoInvalido
	}
	ref, ok := c.porTipo[tipo]
	if !ok {
		return "", ErrTipoNoCatalogado
	}
	return ref, nil
}

// ClaveTipo devuelve la clave estable de un tipo catalogado a partir de su
// referencia opaca, para presentarlo sin exponer la referencia.
func (c *Catalogo) ClaveTipo(tipoRef string) (string, bool) {
	if c == nil {
		return "", false
	}
	e, ok := c.porTipoRef[tipoRef]
	return e.tipo, ok
}

// SolicitudPara construye la solicitud exacta que un módulo productor debe
// pasar a Documentos para un tipo catalogado y un expediente opaco.
func (c *Catalogo) SolicitudPara(tipo, expedienteRef string) (ports.SolicitudPoliticaConservacionDocumental, error) {
	ref, err := c.TipoDocumentalRef(tipo)
	if err != nil {
		return ports.SolicitudPoliticaConservacionDocumental{}, err
	}
	e := c.porTipoRef[ref]
	return ports.NuevaSolicitudPoliticaConservacionDocumental(e.procedimientoRef, e.serieRef, e.tipoRef, expedienteRef,
		e.politicaRef, c.version, e.huella[:], e.baseRef, c.vigenteDesde, c.vigenteHasta)
}

// BuscarPoliticasConservacionDocumental devuelve cero o una coincidencia
// exacta. La fecha de conservación se calcula desde el instante de la
// resolución con el plazo del catálogo.
func (c *Catalogo) BuscarPoliticasConservacionDocumental(ctx context.Context, s ports.SolicitudPoliticaConservacionDocumental) ([]ports.PoliticaConservacionDocumental, error) {
	if c == nil || c.reloj == nil || ctx == nil {
		return nil, ErrCatalogoInvalido
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.Validar() != nil {
		return nil, ports.ErrSolicitudPoliticaConservacionDocumentalInvalida
	}
	e, ok := c.porTipoRef[s.TipoDocumentalRef()]
	if !ok || s.ProcedimientoRef() != e.procedimientoRef || s.SerieDocumentalRef() != e.serieRef ||
		s.PoliticaRef() != e.politicaRef || s.VersionPolitica() != c.version ||
		!bytes.Equal(s.HuellaPoliticaSHA256(), e.huella[:]) || s.BaseJuridicaRef() != e.baseRef ||
		!s.VigenteDesde().Equal(c.vigenteDesde) || !s.VigenteHasta().Equal(c.vigenteHasta) {
		return nil, nil
	}
	hasta := c.reloj.Ahora().UTC().Truncate(time.Microsecond).AddDate(e.plazoAnios, 0, 0)
	// Un catálogo provisional nunca se presenta como política aprobada.
	estado := ports.EstadoPoliticaConservacionDocumentalAprobada
	if c.provisional {
		estado = ports.EstadoPoliticaConservacionDocumentalProvisional
	}
	politica, err := ports.NuevaPoliticaConservacionDocumental(s, hasta,
		ports.ProteccionPoliticaConservacionDocumentalOrdinaria, "", estado, time.Time{})
	if err != nil {
		return nil, err
	}
	return []ports.PoliticaConservacionDocumental{politica}, nil
}

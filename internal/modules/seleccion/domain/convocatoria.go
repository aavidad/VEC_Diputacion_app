package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/shared/baremacion"
)

var (
	// ErrConvocatoriaInvalida: la definición publicada no cumple el contrato.
	ErrConvocatoriaInvalida = errors.New("seleccion: convocatoria invalida")
	// ErrSolicitudInvalida: el contenido de la solicitud no es admisible para
	// la convocatoria (campo vacío, requisito o mérito desconocido…).
	ErrSolicitudInvalida = errors.New("seleccion: solicitud invalida")
)

var (
	convocatoriaRefValida = regexp.MustCompile(`^[a-z0-9][a-z0-9:._-]{2,190}$`)
	claveValida           = regexp.MustCompile(`^[a-z0-9_]{1,64}$`)
	fechaCivilValida      = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
)

// Turno de acceso admitido por la convocatoria (libre, discapacidad…).
type Turno struct {
	Clave    string `json:"clave"`
	Etiqueta string `json:"etiqueta"`
}

// Requisito de acceso estructurado. ImpidePresentar dice si, declarado
// «no_cumple», impide presentar la solicitud; «pendiente» nunca lo impide.
type Requisito struct {
	Clave           string `json:"clave"`
	Titulo          string `json:"titulo"`
	Descripcion     string `json:"descripcion"`
	Obligatorio     bool   `json:"obligatorio"`
	ImpidePresentar bool   `json:"impide_presentar"`
}

// MeritoBaremo puntúa una unidad declarada (mes, curso, hora…).
type MeritoBaremo struct {
	Clave           string
	Titulo          string
	Unidad          string
	PuntosPorUnidad baremacion.Puntos
	Maximo          baremacion.Puntos
}

// GrupoBaremo acota la suma de sus méritos.
type GrupoBaremo struct {
	Clave   string
	Titulo  string
	Maximo  baremacion.Puntos
	Meritos []MeritoBaremo
}

// Baremo de autobaremación de la convocatoria: orientativo, lo recalcula
// siempre el servidor. Redondeo lo fija la convocatoria de forma expresa.
type Baremo struct {
	Maximo   baremacion.Puntos
	Redondeo baremacion.ModoRedondeo
	Grupos   []GrupoBaremo
}

// Numeracion del justificante interno de presentación: Patron lleva
// «{anio}» y «{numero}»; Ancho es el relleno con ceros del número.
type Numeracion struct {
	Patron string `json:"patron"`
	Ancho  int    `json:"ancho"`
}

// Convocatoria es la definición gobernada que Selección publica: plazo de
// presentación, turnos, requisitos, fecha de referencia de los requisitos,
// baremo y numeración. MarcaEjemplo indica un paquete de ejemplo no aprobado.
type Convocatoria struct {
	Ref             string
	Titulo          string
	AbreEn          time.Time
	CierraEn        time.Time
	FechaReferencia string
	Turnos          []Turno
	Requisitos      []Requisito
	Baremo          Baremo
	Numeracion      Numeracion
	MarcaEjemplo    bool
	// PublicadaEn es el instante fijo de la publicación en el catálogo, no
	// el del arranque: repetir el arranque no cambia la huella.
	PublicadaEn time.Time
}

// ConvocatoriaPublicada es la versión vigente en la base con su plazo
// evaluado por el reloj de la base.
type ConvocatoriaPublicada struct {
	Convocatoria
	Version int
	Abierta bool
}

func textoValido(texto string, maximo int) bool {
	return texto != "" && strings.TrimSpace(texto) == texto && utf8.ValidString(texto) && len(texto) <= maximo
}

// Validar exige una definición completa y coherente.
func (c Convocatoria) Validar() error {
	if !convocatoriaRefValida.MatchString(c.Ref) || !textoValido(c.Titulo, 512) || c.AbreEn.IsZero() || !c.CierraEn.After(c.AbreEn) ||
		c.PublicadaEn.IsZero() || !fechaCivilValida.MatchString(c.FechaReferencia) || len(c.Turnos) == 0 || len(c.Turnos) > 16 ||
		len(c.Requisitos) > 64 || !strings.Contains(c.Numeracion.Patron, "{anio}") || !strings.Contains(c.Numeracion.Patron, "{numero}") ||
		len(c.Numeracion.Patron) < 8 || len(c.Numeracion.Patron) > 40 || c.Numeracion.Ancho < 4 || c.Numeracion.Ancho > 9 {
		return ErrConvocatoriaInvalida
	}
	if _, err := time.Parse(time.DateOnly, c.FechaReferencia); err != nil {
		return ErrConvocatoriaInvalida
	}
	vistas := map[string]bool{}
	for _, t := range c.Turnos {
		if !claveValida.MatchString(t.Clave) || !textoValido(t.Etiqueta, 200) || vistas["t:"+t.Clave] {
			return ErrConvocatoriaInvalida
		}
		vistas["t:"+t.Clave] = true
	}
	for _, r := range c.Requisitos {
		if !claveValida.MatchString(r.Clave) || !textoValido(r.Titulo, 300) || len(r.Descripcion) > 2000 || vistas["r:"+r.Clave] ||
			(r.ImpidePresentar && !r.Obligatorio) {
			return ErrConvocatoriaInvalida
		}
		vistas["r:"+r.Clave] = true
	}
	return c.Baremo.Validar()
}

// Validar exige grupos y méritos con claves únicas y máximos coherentes.
func (b Baremo) Validar() error {
	if !b.Redondeo.EsValido() || !b.Maximo.EsValido() || len(b.Grupos) > 32 {
		return ErrConvocatoriaInvalida
	}
	grupos := map[string]bool{}
	for _, g := range b.Grupos {
		if !claveValida.MatchString(g.Clave) || !textoValido(g.Titulo, 300) || grupos[g.Clave] || !g.Maximo.EsValido() ||
			len(g.Meritos) == 0 || len(g.Meritos) > 64 {
			return ErrConvocatoriaInvalida
		}
		grupos[g.Clave] = true
		meritos := map[string]bool{}
		for _, m := range g.Meritos {
			if !claveValida.MatchString(m.Clave) || !textoValido(m.Titulo, 300) || !claveValida.MatchString(m.Unidad) || meritos[m.Clave] ||
				!m.PuntosPorUnidad.EsValido() || !m.Maximo.EsValido() {
				return ErrConvocatoriaInvalida
			}
			meritos[m.Clave] = true
		}
	}
	return nil
}

// Turno devuelve el turno admitido con esa clave.
func (c Convocatoria) Turno(clave string) (Turno, bool) {
	for _, t := range c.Turnos {
		if t.Clave == clave {
			return t, true
		}
	}
	return Turno{}, false
}

// Requisito devuelve el requisito con esa clave.
func (c Convocatoria) Requisito(clave string) (Requisito, bool) {
	for _, r := range c.Requisitos {
		if r.Clave == clave {
			return r, true
		}
	}
	return Requisito{}, false
}

type meritoCanonico struct {
	Clave           string `json:"clave"`
	Titulo          string `json:"titulo"`
	Unidad          string `json:"unidad"`
	PuntosPorUnidad string `json:"puntos_por_unidad"`
	Maximo          string `json:"maximo"`
}

type grupoCanonico struct {
	Clave   string           `json:"clave"`
	Titulo  string           `json:"titulo"`
	Maximo  string           `json:"maximo"`
	Meritos []meritoCanonico `json:"meritos"`
}

type baremoCanonico struct {
	Maximo   string          `json:"maximo"`
	Redondeo string          `json:"redondeo"`
	Grupos   []grupoCanonico `json:"grupos"`
}

// ContenidoCanonico es el documento de reglas que se publica (turnos,
// requisitos, baremo, numeración y fecha de referencia), en orden estable.
type ContenidoCanonico struct {
	Turnos          []Turno        `json:"turnos"`
	Requisitos      []Requisito    `json:"requisitos"`
	Baremo          baremoCanonico `json:"baremo"`
	Numeracion      Numeracion     `json:"numeracion"`
	FechaReferencia string         `json:"fecha_referencia"`
	MarcaEjemplo    bool           `json:"marca_ejemplo"`
}

// Contenido devuelve el documento de reglas canónico de la convocatoria.
func (c Convocatoria) Contenido() ContenidoCanonico {
	b := baremoCanonico{Maximo: FormatearPuntos(c.Baremo.Maximo), Redondeo: string(c.Baremo.Redondeo), Grupos: []grupoCanonico{}}
	for _, g := range c.Baremo.Grupos {
		gc := grupoCanonico{Clave: g.Clave, Titulo: g.Titulo, Maximo: FormatearPuntos(g.Maximo), Meritos: []meritoCanonico{}}
		for _, m := range g.Meritos {
			gc.Meritos = append(gc.Meritos, meritoCanonico{Clave: m.Clave, Titulo: m.Titulo, Unidad: m.Unidad,
				PuntosPorUnidad: FormatearPuntos(m.PuntosPorUnidad), Maximo: FormatearPuntos(m.Maximo)})
		}
		b.Grupos = append(b.Grupos, gc)
	}
	requisitos := append([]Requisito{}, c.Requisitos...)
	return ContenidoCanonico{Turnos: append([]Turno{}, c.Turnos...), Requisitos: requisitos, Baremo: b,
		Numeracion: c.Numeracion, FechaReferencia: c.FechaReferencia, MarcaEjemplo: c.MarcaEjemplo}
}

// ConvocatoriaDesdeContenido reconstruye una convocatoria a partir de su
// documento de reglas publicado.
func ConvocatoriaDesdeContenido(ref, titulo string, abre, cierra, publicada time.Time, contenido []byte) (Convocatoria, error) {
	var cc ContenidoCanonico
	decodificador := json.NewDecoder(strings.NewReader(string(contenido)))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&cc); err != nil {
		return Convocatoria{}, ErrConvocatoriaInvalida
	}
	c := Convocatoria{Ref: ref, Titulo: titulo, AbreEn: abre.UTC(), CierraEn: cierra.UTC(), PublicadaEn: publicada.UTC(),
		FechaReferencia: cc.FechaReferencia, Turnos: cc.Turnos, Requisitos: cc.Requisitos, Numeracion: cc.Numeracion, MarcaEjemplo: cc.MarcaEjemplo}
	var err error
	if c.Baremo.Maximo, err = ParsearPuntos(cc.Baremo.Maximo); err != nil {
		return Convocatoria{}, ErrConvocatoriaInvalida
	}
	c.Baremo.Redondeo = baremacion.ModoRedondeo(cc.Baremo.Redondeo)
	for _, g := range cc.Baremo.Grupos {
		grupo := GrupoBaremo{Clave: g.Clave, Titulo: g.Titulo}
		if grupo.Maximo, err = ParsearPuntos(g.Maximo); err != nil {
			return Convocatoria{}, ErrConvocatoriaInvalida
		}
		for _, m := range g.Meritos {
			merito := MeritoBaremo{Clave: m.Clave, Titulo: m.Titulo, Unidad: m.Unidad}
			if merito.PuntosPorUnidad, err = ParsearPuntos(m.PuntosPorUnidad); err != nil {
				return Convocatoria{}, ErrConvocatoriaInvalida
			}
			if merito.Maximo, err = ParsearPuntos(m.Maximo); err != nil {
				return Convocatoria{}, ErrConvocatoriaInvalida
			}
			grupo.Meritos = append(grupo.Meritos, merito)
		}
		c.Baremo.Grupos = append(c.Baremo.Grupos, grupo)
	}
	if err := c.Validar(); err != nil {
		return Convocatoria{}, err
	}
	return c, nil
}

// ContenidoJSON serializa el documento de reglas.
func (c Convocatoria) ContenidoJSON() ([]byte, error) {
	if err := c.Validar(); err != nil {
		return nil, err
	}
	return json.Marshal(c.Contenido())
}

// HuellaSHA256 identifica la publicación completa: referencia, título,
// plazo, instante de publicación y reglas. El mismo catálogo da siempre la
// misma huella, de modo que republicar al arrancar reutiliza la versión.
func (c Convocatoria) HuellaSHA256() (string, error) {
	contenido, err := c.ContenidoJSON()
	if err != nil {
		return "", err
	}
	documento, err := json.Marshal(struct {
		Ref         string          `json:"convocatoria_ref"`
		Titulo      string          `json:"titulo"`
		AbreEn      string          `json:"abre_en"`
		CierraEn    string          `json:"cierra_en"`
		PublicadaEn string          `json:"publicada_en"`
		Contenido   json.RawMessage `json:"contenido"`
	}{c.Ref, c.Titulo, instanteCanonico(c.AbreEn), instanteCanonico(c.CierraEn), instanteCanonico(c.PublicadaEn), contenido})
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(documento)
	return hex.EncodeToString(suma[:]), nil
}

func instanteCanonico(t time.Time) string {
	return t.UTC().Truncate(time.Microsecond).Format("2006-01-02T15:04:05.000000Z")
}

// AbiertaEn dice si el plazo está abierto en el instante dado. El plazo que
// decide es el de la base; esto solo orienta a la interfaz.
func (c Convocatoria) AbiertaEn(t time.Time) bool {
	return !t.Before(c.AbreEn) && !t.After(c.CierraEn)
}

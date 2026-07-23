package domain

import (
	"errors"
	"sort"
	"time"
)

const (
	maximoViasCobertura                 = 64
	maximoComprobacionesPorViaCobertura = 32
	maximoComprobacionesCatalogo        = 512
)

// ErrPublicacionCatalogoEnConflicto indica que una clave referencia+versión ya
// existe con un resumen de contenido diferente.
var ErrPublicacionCatalogoEnConflicto = errors.New(
	"contratacion temporal: publicacion de catalogo en conflicto",
)

// VigenciaCatalogoCobertura representa un intervalo [Desde, Hasta). Los
// instantes presentes usan UTC, precisión máxima de microsegundo y el intervalo
// 0001-01-01T00:00:00.000001Z..9999-12-31T23:59:59.999999Z para que el estado
// sea transportable mediante JSON/RFC 3339. Exclusivamente
// Hasta == time.Time{} significa ausencia de fecha final; cero es inválido en
// PublicadoEn, Desde y consultas.
type VigenciaCatalogoCobertura struct {
	Desde time.Time `json:"desde"`
	Hasta time.Time `json:"hasta"`
}

func (v VigenciaCatalogoCobertura) Validar() error {
	if !instanteCatalogoCoberturaValido(v.Desde) {
		return ErrDatoInvalido
	}
	if !v.Hasta.IsZero() &&
		(!instanteCatalogoCoberturaValido(v.Hasta) ||
			!v.Hasta.After(v.Desde)) {
		return ErrDatoInvalido
	}
	return nil
}

func (v VigenciaCatalogoCobertura) contiene(instante time.Time) bool {
	return instanteCatalogoCoberturaValido(instante) &&
		!instante.Before(v.Desde) &&
		(v.Hasta.IsZero() || instante.Before(v.Hasta))
}

// ProcedenciaComprobacionCobertura identifica la definición gobernada de la
// fuente que deberá consultar la capa de aplicación. No contiene credenciales
// ni autoriza por sí sola el acceso al sistema de procedencia.
type ProcedenciaComprobacionCobertura struct {
	Clave               ClaveCatalogo `json:"clave"`
	DefinicionFuenteRef string        `json:"definicion_fuente_ref"`
}

func (p ProcedenciaComprobacionCobertura) Validar() error {
	if !p.Clave.Valida() || !referenciaValida(p.DefinicionFuenteRef) {
		return ErrDatoInvalido
	}
	return nil
}

// ComprobacionExigibleCobertura define una comprobación de una vía. La
// obligatoriedad es propia de cada vía, aunque una misma clave pueda
// reutilizarse en otras vías con la misma procedencia.
type ComprobacionExigibleCobertura struct {
	Clave       ClaveCatalogo                    `json:"clave"`
	Orden       uint16                           `json:"orden"`
	Obligatoria bool                             `json:"obligatoria"`
	Procedencia ProcedenciaComprobacionCobertura `json:"procedencia"`
}

func (c ComprobacionExigibleCobertura) Validar() error {
	if !c.Clave.Valida() || c.Orden == 0 || c.Procedencia.Validar() != nil {
		return ErrDatoInvalido
	}
	return nil
}

// DefinicionViaCobertura es una opción funcional gobernada. Su clave no está
// limitada a una lista compilada: cada publicación decide qué vías existen.
type DefinicionViaCobertura struct {
	Clave          ClaveCatalogo                   `json:"clave"`
	Orden          uint16                          `json:"orden"`
	Comprobaciones []ComprobacionExigibleCobertura `json:"comprobaciones"`
}

func (d DefinicionViaCobertura) Validar() error {
	if !d.Clave.Valida() || d.Orden == 0 || len(d.Comprobaciones) == 0 ||
		len(d.Comprobaciones) > maximoComprobacionesPorViaCobertura {
		return ErrDatoInvalido
	}
	claves := make(map[ClaveCatalogo]struct{}, len(d.Comprobaciones))
	ordenes := make(map[uint16]struct{}, len(d.Comprobaciones))
	for _, comprobacion := range d.Comprobaciones {
		if comprobacion.Validar() != nil {
			return ErrDatoInvalido
		}
		if _, repetida := claves[comprobacion.Clave]; repetida {
			return ErrDatoInvalido
		}
		if _, repetido := ordenes[comprobacion.Orden]; repetido {
			return ErrDatoInvalido
		}
		claves[comprobacion.Clave] = struct{}{}
		ordenes[comprobacion.Orden] = struct{}{}
	}
	return nil
}

func (d DefinicionViaCobertura) clonar() DefinicionViaCobertura {
	d.Comprobaciones = append(
		[]ComprobacionExigibleCobertura(nil),
		d.Comprobaciones...,
	)
	return d
}

// IdentidadCatalogoViasCobertura identifica exactamente una publicación. La
// referencia y versión forman la clave durable; la huella debe coincidir para
// que una repetición sea el mismo contenido.
type IdentidadCatalogoViasCobertura struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

func (i IdentidadCatalogoViasCobertura) Validar() error {
	if !referenciaValida(i.Referencia) || i.Version == 0 ||
		!huellaCatalogoValida(i.HuellaSHA256) {
		return ErrDatoInvalido
	}
	return nil
}

func (i IdentidadCatalogoViasCobertura) MismaClaveVersion(
	otra IdentidadCatalogoViasCobertura,
) bool {
	return i.Validar() == nil && otra.Validar() == nil &&
		i.Referencia == otra.Referencia && i.Version == otra.Version
}

func (i IdentidadCatalogoViasCobertura) CoincideExactamente(
	otra IdentidadCatalogoViasCobertura,
) bool {
	return i.MismaClaveVersion(otra) && i.HuellaSHA256 == otra.HuellaSHA256
}

// ValidarReintentoPublicacionCatalogoCobertura resuelve una colisión que la
// persistencia durable ya ha detectado sobre UNIQUE(referencia, version).
// Acepta exclusivamente la repetición exacta y rechaza otro contenido.
//
// Esta función no reserva la clave ni finge unicidad global. El adaptador
// durable deberá aplicar historia de solo adición, impedir UPDATE/DELETE y
// atestar la publicación mediante la capacidad VEC y sus ACL. El resumen
// SHA-256 por sí solo no prueba origen, autenticidad ni autorización.
func ValidarReintentoPublicacionCatalogoCobertura(
	registrada IdentidadCatalogoViasCobertura,
	propuesta IdentidadCatalogoViasCobertura,
) error {
	if registrada.Validar() != nil || propuesta.Validar() != nil ||
		!registrada.MismaClaveVersion(propuesta) {
		return ErrDatoInvalido
	}
	if !registrada.CoincideExactamente(propuesta) {
		return ErrPublicacionCatalogoEnConflicto
	}
	return nil
}

// BorradorCatalogoViasCobertura contiene los datos funcionales que una capa
// autorizada puede someter a publicación. El dominio valida y calcula un
// resumen del contenido, pero no sustituye autorización, atestación VEC,
// auditoría, ACL ni persistencia durable de solo adición.
type BorradorCatalogoViasCobertura struct {
	Referencia     string                    `json:"referencia"`
	Version        uint64                    `json:"version"`
	PublicadoEn    time.Time                 `json:"publicado_en"`
	Vigencia       VigenciaCatalogoCobertura `json:"vigencia"`
	ProcedenciaRef string                    `json:"procedencia_ref"`
	Vias           []DefinicionViaCobertura  `json:"vias"`
}

// PublicacionCatalogoViasCobertura es el estado transportable de una
// publicación. Restaurarlo vuelve a calcular la huella antes de aceptarlo.
type PublicacionCatalogoViasCobertura struct {
	Referencia     string                       `json:"referencia"`
	Version        uint64                       `json:"version"`
	HuellaSHA256   string                       `json:"huella_sha256"`
	Canon          CanonHuellaCatalogoCobertura `json:"canon"`
	PublicadoEn    time.Time                    `json:"publicado_en"`
	Vigencia       VigenciaCatalogoCobertura    `json:"vigencia"`
	ProcedenciaRef string                       `json:"procedencia_ref"`
	Vias           []DefinicionViaCobertura     `json:"vias"`
}

// CatalogoViasCobertura conserva una publicación inmutable dentro del proceso.
// Los métodos de acceso a colecciones entregan copias defensivas. La
// inmutabilidad durable corresponde al adaptador de persistencia append-only.
type CatalogoViasCobertura struct {
	publicacion PublicacionCatalogoViasCobertura
}

// PublicarCatalogoViasCobertura valida, ordena y resume una nueva publicación.
// Añadir una vía o comprobación requiere datos nuevos, no recompilar el núcleo.
func PublicarCatalogoViasCobertura(
	borrador BorradorCatalogoViasCobertura,
) (CatalogoViasCobertura, error) {
	normalizado, err := normalizarBorradorCatalogo(borrador)
	if err != nil {
		return CatalogoViasCobertura{}, err
	}
	publicacion := PublicacionCatalogoViasCobertura{
		Referencia: normalizado.Referencia, Version: normalizado.Version,
		Canon:       CanonHuellaCatalogoCoberturaV1(),
		PublicadoEn: normalizado.PublicadoEn, Vigencia: normalizado.Vigencia,
		ProcedenciaRef: normalizado.ProcedenciaRef, Vias: normalizado.Vias,
	}
	publicacion.HuellaSHA256, err = calcularHuellaCatalogo(publicacion)
	if err != nil || !huellaCatalogoValida(publicacion.HuellaSHA256) {
		return CatalogoViasCobertura{}, ErrDatoInvalido
	}
	return CatalogoViasCobertura{publicacion: publicacion}, nil
}

// RestaurarCatalogoViasCobertura rechaza estados adulterados o no canónicos.
func RestaurarCatalogoViasCobertura(
	publicacion PublicacionCatalogoViasCobertura,
) (CatalogoViasCobertura, error) {
	if !publicacion.Canon.Valido() ||
		!huellaCatalogoValida(publicacion.HuellaSHA256) {
		return CatalogoViasCobertura{}, ErrDatoInvalido
	}
	borrador := BorradorCatalogoViasCobertura{
		Referencia: publicacion.Referencia, Version: publicacion.Version,
		PublicadoEn: publicacion.PublicadoEn, Vigencia: publicacion.Vigencia,
		ProcedenciaRef: publicacion.ProcedenciaRef, Vias: publicacion.Vias,
	}
	restaurado, err := PublicarCatalogoViasCobertura(borrador)
	if err != nil ||
		!restaurado.Identidad().CoincideExactamente(IdentidadCatalogoViasCobertura{
			Referencia: publicacion.Referencia, Version: publicacion.Version,
			HuellaSHA256: publicacion.HuellaSHA256,
		}) {
		return CatalogoViasCobertura{}, ErrDatoInvalido
	}
	return restaurado, nil
}

func (c CatalogoViasCobertura) Validar() error {
	_, err := RestaurarCatalogoViasCobertura(c.Publicacion())
	return err
}

func (c CatalogoViasCobertura) Referencia() string {
	return c.publicacion.Referencia
}

func (c CatalogoViasCobertura) Version() uint64 {
	return c.publicacion.Version
}

func (c CatalogoViasCobertura) HuellaSHA256() string {
	return c.publicacion.HuellaSHA256
}

func (c CatalogoViasCobertura) Identidad() IdentidadCatalogoViasCobertura {
	return IdentidadCatalogoViasCobertura{
		Referencia: c.publicacion.Referencia, Version: c.publicacion.Version,
		HuellaSHA256: c.publicacion.HuellaSHA256,
	}
}

func (c CatalogoViasCobertura) Canon() CanonHuellaCatalogoCobertura {
	return c.publicacion.Canon
}

func (c CatalogoViasCobertura) PublicadoEn() time.Time {
	return c.publicacion.PublicadoEn
}

func (c CatalogoViasCobertura) Vigencia() VigenciaCatalogoCobertura {
	return c.publicacion.Vigencia
}

func (c CatalogoViasCobertura) ProcedenciaRef() string {
	return c.publicacion.ProcedenciaRef
}

func (c CatalogoViasCobertura) VigenteEn(instante time.Time) bool {
	return c.Validar() == nil && c.publicacion.Vigencia.contiene(instante)
}

func (c CatalogoViasCobertura) Vias() []DefinicionViaCobertura {
	return clonarViasCobertura(c.publicacion.Vias)
}

func (c CatalogoViasCobertura) Via(
	clave ClaveCatalogo,
) (DefinicionViaCobertura, bool) {
	if !clave.Valida() || c.Validar() != nil {
		return DefinicionViaCobertura{}, false
	}
	for _, via := range c.publicacion.Vias {
		if via.Clave == clave {
			return via.clonar(), true
		}
	}
	return DefinicionViaCobertura{}, false
}

func (c CatalogoViasCobertura) Publicacion() PublicacionCatalogoViasCobertura {
	publicacion := c.publicacion
	publicacion.Vias = clonarViasCobertura(publicacion.Vias)
	return publicacion
}

func normalizarBorradorCatalogo(
	borrador BorradorCatalogoViasCobertura,
) (BorradorCatalogoViasCobertura, error) {
	if !referenciaValida(borrador.Referencia) || borrador.Version == 0 ||
		!instanteCatalogoCoberturaValido(borrador.PublicadoEn) ||
		borrador.Vigencia.Validar() != nil ||
		!referenciaValida(borrador.ProcedenciaRef) ||
		len(borrador.Vias) == 0 || len(borrador.Vias) > maximoViasCobertura {
		return BorradorCatalogoViasCobertura{}, ErrDatoInvalido
	}
	normalizado := borrador
	normalizado.Vias = clonarViasCobertura(borrador.Vias)
	claves := make(map[ClaveCatalogo]struct{}, len(normalizado.Vias))
	ordenes := make(map[uint16]struct{}, len(normalizado.Vias))
	procedencias := make(map[ClaveCatalogo]ProcedenciaComprobacionCobertura)
	totalComprobaciones := 0
	for indice := range normalizado.Vias {
		via := &normalizado.Vias[indice]
		if via.Validar() != nil {
			return BorradorCatalogoViasCobertura{}, ErrDatoInvalido
		}
		if _, repetida := claves[via.Clave]; repetida {
			return BorradorCatalogoViasCobertura{}, ErrDatoInvalido
		}
		if _, repetido := ordenes[via.Orden]; repetido {
			return BorradorCatalogoViasCobertura{}, ErrDatoInvalido
		}
		claves[via.Clave] = struct{}{}
		ordenes[via.Orden] = struct{}{}
		totalComprobaciones += len(via.Comprobaciones)
		for _, comprobacion := range via.Comprobaciones {
			anterior, reutilizada := procedencias[comprobacion.Clave]
			if reutilizada && anterior != comprobacion.Procedencia {
				return BorradorCatalogoViasCobertura{}, ErrDatoInvalido
			}
			procedencias[comprobacion.Clave] = comprobacion.Procedencia
		}
		sort.Slice(via.Comprobaciones, func(i, j int) bool {
			return via.Comprobaciones[i].Orden < via.Comprobaciones[j].Orden
		})
	}
	if totalComprobaciones > maximoComprobacionesCatalogo {
		return BorradorCatalogoViasCobertura{}, ErrDatoInvalido
	}
	sort.Slice(normalizado.Vias, func(i, j int) bool {
		return normalizado.Vias[i].Orden < normalizado.Vias[j].Orden
	})
	return normalizado, nil
}

func clonarViasCobertura(
	vias []DefinicionViaCobertura,
) []DefinicionViaCobertura {
	clon := make([]DefinicionViaCobertura, len(vias))
	for indice, via := range vias {
		clon[indice] = via.clonar()
	}
	return clon
}

func instanteCatalogoCoberturaValido(valor time.Time) bool {
	return !valor.IsZero() && valor.Location() == time.UTC &&
		valor.Equal(valor.Truncate(time.Microsecond)) &&
		valor.Year() >= 1 && valor.Year() <= 9999
}

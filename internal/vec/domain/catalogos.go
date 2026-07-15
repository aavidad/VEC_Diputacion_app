package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrCatalogoConfigurableInvalido = errors.New("vec: catalogo configurable invalido")
	ErrEntradaCatalogoInvalida      = errors.New("vec: entrada de catalogo invalida")
	ErrEntradaCatalogoDuplicada     = errors.New("vec: entrada de catalogo duplicada")
	ErrEntradaCatalogoNoVigente     = errors.New("vec: entrada de catalogo no vigente")
	ErrCatalogoNoPublicado          = errors.New("vec: catalogo no publicado")
	ErrTransicionCatalogoInvalida   = errors.New("vec: transicion de catalogo invalida")
)

// ReferenciaEntradaCatalogo fija tanto la version como la huella del catalogo.
// Una definicion de flujo nunca resuelve estados contra «la ultima version».
type ReferenciaEntradaCatalogo struct {
	CatalogoID           string `json:"catalogo_id"`
	CatalogoVersion      int    `json:"catalogo_version"`
	CatalogoHuellaSHA256 string `json:"catalogo_huella_sha256"`
	EntradaClave         string `json:"entrada_clave"`
}

func (r ReferenciaEntradaCatalogo) Validar() error {
	if !esClaveDocumentalCanonica(r.CatalogoID) || r.CatalogoVersion < 1 ||
		!esSHA256(r.CatalogoHuellaSHA256) || !esClaveDocumentalCanonica(r.EntradaClave) {
		return ErrEntradaCatalogoInvalida
	}
	return nil
}

func (r ReferenciaEntradaCatalogo) Referencia() string {
	return r.CatalogoID + ":" + strconv.Itoa(r.CatalogoVersion) + ":" + r.EntradaClave
}

const (
	maximoEntradasCatalogo         = 10_000
	maximoAtributosEntradaCatalogo = 128
	maximoCaracteresEtiqueta       = 512
	maximoCaracteresDescripcion    = 8 * 1024
	maximoCaracteresAtributo       = 8 * 1024
	maximoBytesCatalogo            = 16 * 1024 * 1024
)

// EntradaCatalogoConfigurable es un valor de negocio administrable sin
// recompilar. Atributos permite ampliar metadatos sencillos; las estructuras
// complejas deben apuntar a una definicion o regla versionada independiente.
type EntradaCatalogoConfigurable struct {
	Clave        string            `json:"clave"`
	Etiqueta     string            `json:"etiqueta"`
	Descripcion  string            `json:"descripcion,omitempty"`
	Orden        int               `json:"orden"`
	VigenteDesde time.Time         `json:"vigente_desde"`
	VigenteHasta time.Time         `json:"vigente_hasta,omitempty"`
	Atributos    map[string]string `json:"atributos,omitempty"`
}

func (e EntradaCatalogoConfigurable) Validar() error {
	if !esClaveDocumentalCanonica(e.Clave) || !textoAcotadoCatalogo(e.Etiqueta, maximoCaracteresEtiqueta, true) ||
		!textoAcotadoCatalogo(e.Descripcion, maximoCaracteresDescripcion, false) || e.Orden < 0 ||
		e.VigenteDesde.IsZero() || (!e.VigenteHasta.IsZero() && !e.VigenteHasta.After(e.VigenteDesde)) ||
		len(e.Atributos) > maximoAtributosEntradaCatalogo {
		return ErrEntradaCatalogoInvalida
	}
	for clave, valor := range e.Atributos {
		if !esClaveDocumentalCanonica(clave) || !textoAcotadoCatalogo(valor, maximoCaracteresAtributo, true) {
			return ErrEntradaCatalogoInvalida
		}
	}
	return nil
}

func (e EntradaCatalogoConfigurable) VigenteEn(instante time.Time) bool {
	instante = instante.UTC()
	return !instante.Before(e.VigenteDesde.UTC()) && (e.VigenteHasta.IsZero() || instante.Before(e.VigenteHasta.UTC()))
}

type EstadoCatalogoConfigurable string

const (
	EstadoCatalogoBorrador  EstadoCatalogoConfigurable = "borrador"
	EstadoCatalogoPublicado EstadoCatalogoConfigurable = "publicado"
	EstadoCatalogoRetirado  EstadoCatalogoConfigurable = "retirado"
)

const (
	AccionCatalogoBorradorCreado      = "vec.catalogos.borrador.creado"
	AccionCatalogoBorradorActualizado = "vec.catalogos.borrador.actualizado"
	AccionCatalogoPublicado           = "vec.catalogos.publicado"
	AccionCatalogoRetirado            = "vec.catalogos.retirado"
)

func (e EstadoCatalogoConfigurable) Valido() bool {
	return e == EstadoCatalogoBorrador || e == EstadoCatalogoPublicado || e == EstadoCatalogoRetirado
}

// CatalogoConfigurable es una instantanea completa e inmutable al publicarse.
// Agregar «cosa cuatro» crea una nueva version desde la aplicacion; ningun
// consumidor depende implicitamente de la ultima version.
type CatalogoConfigurable struct {
	ID                    string                        `json:"id"`
	Version               int                           `json:"version"`
	Revision              int                           `json:"revision"`
	VersionAnteriorRef    string                        `json:"version_anterior_ref,omitempty"`
	ModuloID              string                        `json:"modulo_id"`
	Nombre                string                        `json:"nombre"`
	Descripcion           string                        `json:"descripcion,omitempty"`
	FuenteRef             string                        `json:"fuente_ref"`
	MotivoCreacion        string                        `json:"motivo_creacion"`
	Entradas              []EntradaCatalogoConfigurable `json:"entradas"`
	Estado                EstadoCatalogoConfigurable    `json:"estado"`
	CreadoPor             string                        `json:"creado_por"`
	CreadoEn              time.Time                     `json:"creado_en"`
	UltimaModificacionPor string                        `json:"ultima_modificacion_por,omitempty"`
	UltimaModificacionEn  time.Time                     `json:"ultima_modificacion_en,omitempty"`
	MotivoModificacion    string                        `json:"motivo_modificacion,omitempty"`
	PublicadoPor          string                        `json:"publicado_por,omitempty"`
	PublicadoEn           time.Time                     `json:"publicado_en,omitempty"`
	AprobacionRef         string                        `json:"aprobacion_ref,omitempty"`
	MotivoPublicacion     string                        `json:"motivo_publicacion,omitempty"`
	RetiradoPor           string                        `json:"retirado_por,omitempty"`
	RetiradoEn            time.Time                     `json:"retirado_en,omitempty"`
	RetiradaAprobacionRef string                        `json:"retirada_aprobacion_ref,omitempty"`
	MotivoRetirada        string                        `json:"motivo_retirada,omitempty"`
}

func (c CatalogoConfigurable) Referencia() string {
	return strings.TrimSpace(c.ID) + ":" + strconv.Itoa(c.Version)
}

func (c CatalogoConfigurable) Validar() error {
	if !esClaveDocumentalCanonica(c.ID) || c.Version < 1 || c.Revision < 1 || !esClaveDocumentalCanonica(c.ModuloID) ||
		!textoAcotadoCatalogo(c.Nombre, maximoCaracteresEtiqueta, true) ||
		!textoAcotadoCatalogo(c.Descripcion, maximoCaracteresDescripcion, false) ||
		!textoAcotadoCatalogo(c.FuenteRef, maximoCaracteresReferenciaDocumental, true) ||
		!textoAcotadoCatalogo(c.MotivoCreacion, maximoCaracteresDescripcion, true) ||
		len(c.Entradas) > maximoEntradasCatalogo || !c.Estado.Valido() ||
		!referenciaDocumentalValida(c.CreadoPor) || c.CreadoEn.IsZero() {
		return ErrCatalogoConfigurableInvalido
	}
	if (c.Version == 1 && c.VersionAnteriorRef != "") ||
		(c.Version > 1 && c.VersionAnteriorRef != c.ID+":"+strconv.Itoa(c.Version-1)) {
		return ErrCatalogoConfigurableInvalido
	}
	if c.Revision == 1 {
		if c.UltimaModificacionPor != "" || !c.UltimaModificacionEn.IsZero() || c.MotivoModificacion != "" {
			return ErrCatalogoConfigurableInvalido
		}
	} else if !referenciaDocumentalValida(c.UltimaModificacionPor) || c.UltimaModificacionEn.IsZero() ||
		c.UltimaModificacionEn.Before(c.CreadoEn) ||
		!textoAcotadoCatalogo(c.MotivoModificacion, maximoCaracteresDescripcion, true) {
		return ErrCatalogoConfigurableInvalido
	}
	totalBytes := len(c.ID) + len(c.VersionAnteriorRef) + len(c.ModuloID) + len(c.Nombre) + len(c.Descripcion) +
		len(c.FuenteRef) + len(c.MotivoCreacion) + len(c.UltimaModificacionPor) + len(c.MotivoModificacion)
	vistas := make(map[string]struct{}, len(c.Entradas))
	for _, entrada := range c.Entradas {
		if err := entrada.Validar(); err != nil {
			return err
		}
		if _, existe := vistas[entrada.Clave]; existe {
			return ErrEntradaCatalogoDuplicada
		}
		vistas[entrada.Clave] = struct{}{}
		totalBytes += len(entrada.Clave) + len(entrada.Etiqueta) + len(entrada.Descripcion)
		for clave, valor := range entrada.Atributos {
			totalBytes += len(clave) + len(valor)
		}
		if totalBytes > maximoBytesCatalogo {
			return ErrCatalogoConfigurableInvalido
		}
	}
	switch c.Estado {
	case EstadoCatalogoBorrador:
		if c.PublicadoPor != "" || !c.PublicadoEn.IsZero() || c.AprobacionRef != "" || c.MotivoPublicacion != "" ||
			c.RetiradoPor != "" || !c.RetiradoEn.IsZero() || c.RetiradaAprobacionRef != "" || c.MotivoRetirada != "" {
			return ErrCatalogoConfigurableInvalido
		}
	case EstadoCatalogoPublicado:
		if len(c.Entradas) == 0 || !datosPublicacionCatalogoValidos(c) ||
			c.RetiradoPor != "" || !c.RetiradoEn.IsZero() || c.RetiradaAprobacionRef != "" || c.MotivoRetirada != "" {
			return ErrCatalogoConfigurableInvalido
		}
	case EstadoCatalogoRetirado:
		if len(c.Entradas) == 0 || !datosPublicacionCatalogoValidos(c) ||
			!referenciaDocumentalValida(c.RetiradoPor) || c.RetiradoEn.IsZero() || c.RetiradoEn.Before(c.PublicadoEn) ||
			!textoAcotadoCatalogo(c.RetiradaAprobacionRef, maximoCaracteresReferenciaDocumental, true) ||
			!textoAcotadoCatalogo(c.MotivoRetirada, maximoCaracteresDescripcion, true) {
			return ErrCatalogoConfigurableInvalido
		}
	}
	return nil
}

func (c CatalogoConfigurable) ClonarCanonico() (CatalogoConfigurable, error) {
	canonico := c
	canonico.CreadoEn = c.CreadoEn.UTC()
	canonico.UltimaModificacionEn = fechaCatalogoUTC(c.UltimaModificacionEn)
	canonico.PublicadoEn = fechaCatalogoUTC(c.PublicadoEn)
	canonico.RetiradoEn = fechaCatalogoUTC(c.RetiradoEn)
	canonico.Entradas = make([]EntradaCatalogoConfigurable, len(c.Entradas))
	for indice, entrada := range c.Entradas {
		canonico.Entradas[indice] = entrada
		canonico.Entradas[indice].VigenteDesde = entrada.VigenteDesde.UTC()
		canonico.Entradas[indice].VigenteHasta = fechaCatalogoUTC(entrada.VigenteHasta)
		canonico.Entradas[indice].Atributos = clonarAtributosCatalogo(entrada.Atributos)
	}
	sort.Slice(canonico.Entradas, func(i, j int) bool {
		if canonico.Entradas[i].Orden != canonico.Entradas[j].Orden {
			return canonico.Entradas[i].Orden < canonico.Entradas[j].Orden
		}
		return canonico.Entradas[i].Clave < canonico.Entradas[j].Clave
	})
	if err := canonico.Validar(); err != nil {
		return CatalogoConfigurable{}, err
	}
	return canonico, nil
}

func (c CatalogoConfigurable) HuellaSHA256() (string, error) {
	canonico, err := c.ClonarCanonico()
	if err != nil {
		return "", err
	}
	contenido, err := json.Marshal(canonico)
	if err != nil {
		return "", ErrCatalogoConfigurableInvalido
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

// HuellaContenidoSHA256 identifica la semantica inmutable de la version. No
// cambia al publicar o retirar, a diferencia de HuellaSHA256, que evidencia la
// instantanea completa de gobierno.
func (c CatalogoConfigurable) HuellaContenidoSHA256() (string, error) {
	canonico, err := c.ClonarCanonico()
	if err != nil {
		return "", err
	}
	canonico.Estado = ""
	canonico.CreadoPor = ""
	canonico.CreadoEn = time.Time{}
	canonico.MotivoCreacion = ""
	canonico.UltimaModificacionPor = ""
	canonico.UltimaModificacionEn = time.Time{}
	canonico.MotivoModificacion = ""
	canonico.PublicadoPor = ""
	canonico.PublicadoEn = time.Time{}
	canonico.AprobacionRef = ""
	canonico.MotivoPublicacion = ""
	canonico.RetiradoPor = ""
	canonico.RetiradoEn = time.Time{}
	canonico.RetiradaAprobacionRef = ""
	canonico.MotivoRetirada = ""
	contenido, err := json.Marshal(canonico)
	if err != nil {
		return "", ErrCatalogoConfigurableInvalido
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func (c CatalogoConfigurable) Publicar(actorID, aprobacionRef, motivo string, instante time.Time) (CatalogoConfigurable, error) {
	if err := c.Validar(); err != nil {
		return CatalogoConfigurable{}, err
	}
	actorID = strings.TrimSpace(actorID)
	if c.Estado != EstadoCatalogoBorrador || len(c.Entradas) == 0 || !referenciaDocumentalValida(actorID) ||
		actorID == c.CreadoPor || actorID == c.UltimaModificacionPor ||
		!textoAcotadoCatalogo(aprobacionRef, maximoCaracteresReferenciaDocumental, true) ||
		!textoAcotadoCatalogo(motivo, maximoCaracteresDescripcion, true) || instante.IsZero() || instante.Before(c.CreadoEn) {
		return CatalogoConfigurable{}, ErrTransicionCatalogoInvalida
	}
	publicado := c
	publicado.Estado = EstadoCatalogoPublicado
	publicado.PublicadoPor = actorID
	publicado.PublicadoEn = instante.UTC()
	publicado.AprobacionRef = strings.TrimSpace(aprobacionRef)
	publicado.MotivoPublicacion = strings.TrimSpace(motivo)
	if err := publicado.Validar(); err != nil {
		return CatalogoConfigurable{}, err
	}
	return publicado.ClonarCanonico()
}

func (c CatalogoConfigurable) Retirar(actorID, aprobacionRef, motivo string, instante time.Time) (CatalogoConfigurable, error) {
	if err := c.Validar(); err != nil {
		return CatalogoConfigurable{}, err
	}
	actorID = strings.TrimSpace(actorID)
	if c.Estado != EstadoCatalogoPublicado || !referenciaDocumentalValida(actorID) || actorID == c.PublicadoPor ||
		!textoAcotadoCatalogo(aprobacionRef, maximoCaracteresReferenciaDocumental, true) ||
		!textoAcotadoCatalogo(motivo, maximoCaracteresDescripcion, true) || instante.IsZero() || instante.Before(c.PublicadoEn) {
		return CatalogoConfigurable{}, ErrTransicionCatalogoInvalida
	}
	retirado := c
	retirado.Estado = EstadoCatalogoRetirado
	retirado.RetiradoPor = actorID
	retirado.RetiradoEn = instante.UTC()
	retirado.RetiradaAprobacionRef = strings.TrimSpace(aprobacionRef)
	retirado.MotivoRetirada = strings.TrimSpace(motivo)
	if err := retirado.Validar(); err != nil {
		return CatalogoConfigurable{}, err
	}
	return retirado.ClonarCanonico()
}

func (c CatalogoConfigurable) NuevaVersion(version int, creadorID, fuenteRef, motivo string, instante time.Time) (CatalogoConfigurable, error) {
	if err := c.Validar(); err != nil {
		return CatalogoConfigurable{}, err
	}
	creadorID = strings.TrimSpace(creadorID)
	if c.Estado == EstadoCatalogoBorrador || version != c.Version+1 || !referenciaDocumentalValida(creadorID) ||
		!textoAcotadoCatalogo(fuenteRef, maximoCaracteresReferenciaDocumental, true) ||
		!textoAcotadoCatalogo(motivo, maximoCaracteresDescripcion, true) || instante.IsZero() || instante.Before(c.CreadoEn) {
		return CatalogoConfigurable{}, ErrTransicionCatalogoInvalida
	}
	nueva := CatalogoConfigurable{
		ID:                 c.ID,
		Version:            version,
		Revision:           1,
		VersionAnteriorRef: c.Referencia(),
		ModuloID:           c.ModuloID,
		Nombre:             c.Nombre,
		Descripcion:        c.Descripcion,
		FuenteRef:          strings.TrimSpace(fuenteRef),
		MotivoCreacion:     strings.TrimSpace(motivo),
		Entradas:           c.Entradas,
		Estado:             EstadoCatalogoBorrador,
		CreadoPor:          creadorID,
		CreadoEn:           instante.UTC(),
	}
	return nueva.ClonarCanonico()
}

func (c CatalogoConfigurable) ActualizarBorrador(
	revisionEsperada int,
	actorID, nombre, descripcion, fuenteRef, motivo string,
	entradas []EntradaCatalogoConfigurable,
	instante time.Time,
) (CatalogoConfigurable, error) {
	if err := c.Validar(); err != nil {
		return CatalogoConfigurable{}, err
	}
	actorID = strings.TrimSpace(actorID)
	if c.Estado != EstadoCatalogoBorrador || revisionEsperada != c.Revision || !referenciaDocumentalValida(actorID) ||
		!textoAcotadoCatalogo(nombre, maximoCaracteresEtiqueta, true) ||
		!textoAcotadoCatalogo(descripcion, maximoCaracteresDescripcion, false) ||
		!textoAcotadoCatalogo(fuenteRef, maximoCaracteresReferenciaDocumental, true) ||
		!textoAcotadoCatalogo(motivo, maximoCaracteresDescripcion, true) || instante.IsZero() ||
		instante.Before(c.CreadoEn) || (!c.UltimaModificacionEn.IsZero() && instante.Before(c.UltimaModificacionEn)) {
		return CatalogoConfigurable{}, ErrTransicionCatalogoInvalida
	}
	actualizado := c
	actualizado.Revision++
	actualizado.Nombre = strings.TrimSpace(nombre)
	actualizado.Descripcion = strings.TrimSpace(descripcion)
	actualizado.FuenteRef = strings.TrimSpace(fuenteRef)
	actualizado.Entradas = append([]EntradaCatalogoConfigurable(nil), entradas...)
	actualizado.UltimaModificacionPor = actorID
	actualizado.UltimaModificacionEn = instante.UTC()
	actualizado.MotivoModificacion = strings.TrimSpace(motivo)
	return actualizado.ClonarCanonico()
}

func (c CatalogoConfigurable) ObtenerEntradaVigente(clave string, instante time.Time) (EntradaCatalogoConfigurable, error) {
	if err := c.Validar(); err != nil {
		return EntradaCatalogoConfigurable{}, err
	}
	if c.Estado != EstadoCatalogoPublicado {
		return EntradaCatalogoConfigurable{}, ErrCatalogoNoPublicado
	}
	if clave != strings.TrimSpace(clave) {
		return EntradaCatalogoConfigurable{}, ErrEntradaCatalogoNoVigente
	}
	for _, entrada := range c.Entradas {
		if entrada.Clave == clave {
			if !entrada.VigenteEn(instante) {
				return EntradaCatalogoConfigurable{}, ErrEntradaCatalogoNoVigente
			}
			entrada.Atributos = clonarAtributosCatalogo(entrada.Atributos)
			return entrada, nil
		}
	}
	return EntradaCatalogoConfigurable{}, ErrEntradaCatalogoNoVigente
}

func datosPublicacionCatalogoValidos(c CatalogoConfigurable) bool {
	return referenciaDocumentalValida(c.PublicadoPor) && !c.PublicadoEn.IsZero() && !c.PublicadoEn.Before(c.CreadoEn) &&
		textoAcotadoCatalogo(c.AprobacionRef, maximoCaracteresReferenciaDocumental, true) &&
		textoAcotadoCatalogo(c.MotivoPublicacion, maximoCaracteresDescripcion, true)
}

func textoAcotadoCatalogo(valor string, maximo int, obligatorio bool) bool {
	if valor != strings.TrimSpace(valor) || len(valor) > maximo || !textoDocumentalValido(valor) {
		return false
	}
	return !obligatorio || valor != ""
}

func clonarAtributosCatalogo(atributos map[string]string) map[string]string {
	if atributos == nil {
		return nil
	}
	clon := make(map[string]string, len(atributos))
	for clave, valor := range atributos {
		clon[clave] = valor
	}
	return clon
}

func fechaCatalogoUTC(fecha time.Time) time.Time {
	if fecha.IsZero() {
		return time.Time{}
	}
	return fecha.UTC()
}

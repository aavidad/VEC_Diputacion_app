package domain

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrVersionConvocatoriaGobernadaInvalida = errors.New("bolsa: version gobernada de convocatoria invalida")
	ErrTransicionGobiernoConvocatoria       = errors.New("bolsa: transicion de gobierno de convocatoria invalida")
)

const (
	AccionBorradorConvocatoriaCreado      = "bolsa.convocatoria.borrador.creado"
	AccionBorradorConvocatoriaActualizado = "bolsa.convocatoria.borrador.actualizado"
	AccionConvocatoriaPublicada           = "bolsa.convocatoria.publicada"
	AccionConvocatoriaSustituida          = "bolsa.convocatoria.sustituida"
	AccionConvocatoriaRetirada            = "bolsa.convocatoria.retirada"
)

// ReferenciaConfiguracionConvocatoria fija identidad, version y contenido.
// Nunca representa «la ultima version» de una dependencia.
type ReferenciaConfiguracionConvocatoria struct {
	ID                    string `json:"id"`
	Version               int    `json:"version"`
	HuellaContenidoSHA256 string `json:"huella_contenido_sha256"`
}

func (r ReferenciaConfiguracionConvocatoria) Validar() error {
	if !referenciaOpacaValida(r.ID) || r.Version < 1 || !huellaSHA256Valida(r.HuellaContenidoSHA256) {
		return ErrVersionConvocatoriaGobernadaInvalida
	}
	return nil
}

type ReferenciaDocumentoOficialConvocatoria struct {
	Rol                   string `json:"rol"`
	PublicacionRef        string `json:"publicacion_ref"`
	DocumentoRef          string `json:"documento_ref"`
	VersionDocumento      int    `json:"version_documento"`
	RepresentacionRef     string `json:"representacion_ref"`
	HuellaContenidoSHA256 string `json:"huella_contenido_sha256"`
	FirmaValidadaRef      string `json:"firma_validada_ref"`
	ReciboCustodiaRef     string `json:"recibo_custodia_ref"`
}

func (r ReferenciaDocumentoOficialConvocatoria) Validar() error {
	if !claveNegocioValida(r.Rol) || !referenciaConvocatoriaValida(r.PublicacionRef) ||
		!referenciaOpacaValida(r.DocumentoRef) || r.VersionDocumento < 1 ||
		!referenciaOpacaValida(r.RepresentacionRef) || !huellaSHA256Valida(r.HuellaContenidoSHA256) ||
		!referenciaOpacaValida(r.FirmaValidadaRef) || !referenciaOpacaValida(r.ReciboCustodiaRef) {
		return ErrVersionConvocatoriaGobernadaInvalida
	}
	return nil
}

type ConfiguracionFijadaConvocatoria struct {
	Catalogos        ReferenciaConfiguracionConvocatoria      `json:"catalogos"`
	Calendario       ReferenciaConfiguracionConvocatoria      `json:"calendario"`
	ReglasBaremacion ReferenciaConfiguracionConvocatoria      `json:"reglas_baremacion"`
	FlujoProceso     ReferenciaConfiguracionConvocatoria      `json:"flujo_proceso"`
	FlujoSolicitud   ReferenciaConfiguracionConvocatoria      `json:"flujo_solicitud"`
	Documentos       []ReferenciaDocumentoOficialConvocatoria `json:"documentos"`
}

func (c ConfiguracionFijadaConvocatoria) ClonarCanonicaPara(
	contenido ContenidoPublicableConvocatoria,
) (ConfiguracionFijadaConvocatoria, error) {
	clon := c
	clon.Documentos = append([]ReferenciaDocumentoOficialConvocatoria(nil), c.Documentos...)
	sort.Slice(clon.Documentos, func(i, j int) bool {
		return clon.Documentos[i].PublicacionRef < clon.Documentos[j].PublicacionRef
	})
	if err := clon.ValidarPara(contenido); err != nil {
		return ConfiguracionFijadaConvocatoria{}, err
	}
	return clon, nil
}

func (c ConfiguracionFijadaConvocatoria) ValidarPara(contenido ContenidoPublicableConvocatoria) error {
	if c.Catalogos.Validar() != nil || c.Calendario.Validar() != nil ||
		c.ReglasBaremacion.Validar() != nil || c.FlujoProceso.Validar() != nil ||
		c.FlujoSolicitud.Validar() != nil || len(c.Documentos) == 0 ||
		len(c.Documentos) != len(contenido.Documentos) {
		return ErrVersionConvocatoriaGobernadaInvalida
	}
	publicos := make(map[string]DocumentoPublicableConvocatoria, len(contenido.Documentos))
	for _, documento := range contenido.Documentos {
		publicos[documento.Referencia] = documento
	}
	publicaciones := make(map[string]struct{}, len(c.Documentos))
	representaciones := make(map[string]struct{}, len(c.Documentos))
	for _, documento := range c.Documentos {
		publico, existe := publicos[documento.PublicacionRef]
		if documento.Validar() != nil || !existe || documento.Rol != publico.Tipo {
			return ErrVersionConvocatoriaGobernadaInvalida
		}
		if _, repetida := publicaciones[documento.PublicacionRef]; repetida {
			return ErrVersionConvocatoriaGobernadaInvalida
		}
		if _, repetida := representaciones[documento.RepresentacionRef]; repetida {
			return ErrVersionConvocatoriaGobernadaInvalida
		}
		publicaciones[documento.PublicacionRef] = struct{}{}
		representaciones[documento.RepresentacionRef] = struct{}{}
	}
	return nil
}

type DocumentoPublicableConvocatoria struct {
	Referencia  string `json:"referencia"`
	Tipo        string `json:"tipo"`
	Orden       int    `json:"orden"`
	Titulo      string `json:"titulo"`
	Descripcion string `json:"descripcion"`
	Formato     string `json:"formato"`
	URL         string `json:"url"`
}

// ContenidoPublicableConvocatoria no contiene fase ni marcas de publicacion.
// Esas piezas proceden, respectivamente, del flujo y del acto de gobierno.
type ContenidoPublicableConvocatoria struct {
	IdentificadorPublico string                            `json:"identificador_publico"`
	Tipo                 string                            `json:"tipo"`
	CatalogoCategorias   ReferenciaCatalogoCategorias      `json:"catalogo_categorias"`
	Categorias           []string                          `json:"categorias"`
	Titulo               string                            `json:"titulo"`
	Resumen              string                            `json:"resumen"`
	Descripcion          string                            `json:"descripcion"`
	Plazos               []PlazoConvocatoria               `json:"plazos"`
	Requisitos           []RequisitoConvocatoria           `json:"requisitos"`
	Documentos           []DocumentoPublicableConvocatoria `json:"documentos"`
	Ayuda                []AyudaConvocatoria               `json:"ayuda"`
}

func (c ContenidoPublicableConvocatoria) Validar() error {
	if !patronIdentificadorPublico.MatchString(c.IdentificadorPublico) ||
		!claveCatalogoConvocatoriaValida(c.Tipo) || !c.CatalogoCategorias.Valida() ||
		!textoConvocatoriaValido(c.Titulo, 180, false) ||
		!textoConvocatoriaValido(c.Resumen, 500, false) ||
		!textoConvocatoriaValido(c.Descripcion, 12000, true) ||
		len(c.Categorias) == 0 || len(c.Categorias) > maximoCategoriasConvocatoria ||
		len(c.Plazos) == 0 || len(c.Plazos) > maximoPlazosConvocatoria ||
		len(c.Requisitos) > maximoRequisitosConvocatoria || len(c.Documentos) == 0 ||
		len(c.Documentos) > maximoDocumentosConvocatoria || len(c.Ayuda) > maximoAyudasConvocatoria ||
		!clavesCatalogoUnicas(c.Categorias) || !plazosValidos(c.Plazos) ||
		!requisitosValidos(c.Requisitos) || !documentosPublicablesValidos(c.Documentos) ||
		!ayudasValidas(c.Ayuda) {
		return ErrVersionConvocatoriaGobernadaInvalida
	}
	return nil
}

func (c ContenidoPublicableConvocatoria) ClonarCanonico() (ContenidoPublicableConvocatoria, error) {
	clon := c
	clon.Categorias = append([]string(nil), c.Categorias...)
	clon.Plazos = append([]PlazoConvocatoria(nil), c.Plazos...)
	clon.Requisitos = append([]RequisitoConvocatoria(nil), c.Requisitos...)
	clon.Documentos = append([]DocumentoPublicableConvocatoria(nil), c.Documentos...)
	clon.Ayuda = append([]AyudaConvocatoria(nil), c.Ayuda...)
	sort.Strings(clon.Categorias)
	sort.Slice(clon.Plazos, func(i, j int) bool { return clon.Plazos[i].Referencia < clon.Plazos[j].Referencia })
	sort.Slice(clon.Requisitos, func(i, j int) bool {
		if clon.Requisitos[i].Orden == clon.Requisitos[j].Orden {
			return clon.Requisitos[i].Referencia < clon.Requisitos[j].Referencia
		}
		return clon.Requisitos[i].Orden < clon.Requisitos[j].Orden
	})
	sort.Slice(clon.Documentos, func(i, j int) bool {
		if clon.Documentos[i].Orden == clon.Documentos[j].Orden {
			return clon.Documentos[i].Referencia < clon.Documentos[j].Referencia
		}
		return clon.Documentos[i].Orden < clon.Documentos[j].Orden
	})
	sort.Slice(clon.Ayuda, func(i, j int) bool {
		if clon.Ayuda[i].Orden == clon.Ayuda[j].Orden {
			return clon.Ayuda[i].Referencia < clon.Ayuda[j].Referencia
		}
		return clon.Ayuda[i].Orden < clon.Ayuda[j].Orden
	})
	if err := clon.Validar(); err != nil {
		return ContenidoPublicableConvocatoria{}, err
	}
	return clon, nil
}

func documentosPublicablesValidos(documentos []DocumentoPublicableConvocatoria) bool {
	vistos := make(map[string]struct{}, len(documentos))
	ordenes := make(map[int]struct{}, len(documentos))
	for _, documento := range documentos {
		if !referenciaUnica(documento.Referencia, vistos) || documento.Orden < 1 ||
			!ordenUnico(documento.Orden, ordenes) || !claveCatalogoConvocatoriaValida(documento.Tipo) ||
			!claveCatalogoConvocatoriaValida(documento.Formato) ||
			!textoConvocatoriaValido(documento.Titulo, 180, false) ||
			!textoConvocatoriaValido(documento.Descripcion, 1000, true) ||
			!urlDocumentoPublicoValida(documento.URL) {
			return false
		}
	}
	return true
}

func referenciaVersionConvocatoria(id string, secuencia int) string {
	return strings.TrimSpace(id) + "#" + strconv.Itoa(secuencia)
}

func instanteConvocatoriaCanonico(instante time.Time) time.Time {
	if instante.IsZero() {
		return time.Time{}
	}
	return instante.UTC().Truncate(time.Microsecond)
}

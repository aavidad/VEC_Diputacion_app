package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrCatalogoNoEncontrado              = errors.New("vec: catalogo configurable no encontrado")
	ErrVersionCatalogoYaExiste           = errors.New("vec: version de catalogo ya existe")
	ErrRevisionCatalogoEnConflicto       = errors.New("vec: revision de catalogo en conflicto")
	ErrSecuenciaCatalogoInvalida         = errors.New("vec: secuencia de catalogo invalida")
	ErrLimitesConsultaCatalogosInvalidos = errors.New(
		"vec: limites de consulta de catalogos invalidos",
	)
)

const (
	AccionCrearCatalogoConfigurable      = "vec.catalogos.crear"
	AccionActualizarCatalogoConfigurable = "vec.catalogos.actualizar"
	AccionPublicarCatalogoConfigurable   = "vec.catalogos.publicar"
	AccionRetirarCatalogoConfigurable    = "vec.catalogos.retirar"
)

type ConsultaCatalogosConfigurables interface {
	ObtenerCatalogo(context.Context, string, int) (domain.CatalogoConfigurable, error)
	ListarVersionesCatalogo(context.Context, string) ([]domain.CatalogoConfigurable, error)
}

const (
	// Estos máximos son deliberadamente menores que los límites unitarios del
	// dominio. Una resolución interactiva no debe materializar el catálogo
	// completo de gobierno ni convertir sus límites documentales en memoria.
	MaximoVersionesConsultaCatalogosAcotada = 64
	MaximoEntradasConsultaCatalogosAcotada  = 4_096
	MaximoAtributosConsultaCatalogosAcotada = 8_192
	MaximoBytesConsultaCatalogosAcotada     = 4 << 20
)

// LimitesConsultaCatalogosAcotada obliga al adaptador a limitar antes de
// clonar. Cada consumidor puede solicitar un presupuesto menor, nunca mayor.
type LimitesConsultaCatalogosAcotada struct {
	Versiones        int
	Entradas         int
	Atributos        int
	BytesAproximados int
}

func (l LimitesConsultaCatalogosAcotada) Validar() error {
	if l.Versiones < 1 ||
		l.Versiones > MaximoVersionesConsultaCatalogosAcotada ||
		l.Entradas < 1 ||
		l.Entradas > MaximoEntradasConsultaCatalogosAcotada ||
		l.Atributos < 1 ||
		l.Atributos > MaximoAtributosConsultaCatalogosAcotada ||
		l.BytesAproximados < 1 ||
		l.BytesAproximados > MaximoBytesConsultaCatalogosAcotada {
		return ErrLimitesConsultaCatalogosInvalidos
	}
	return nil
}

type ConsumoConsultaCatalogosAcotada struct {
	Versiones        int
	Entradas         int
	Atributos        int
	BytesAproximados int
}

func (c ConsumoConsultaCatalogosAcotada) Agregar(
	otro ConsumoConsultaCatalogosAcotada,
	limites LimitesConsultaCatalogosAcotada,
) (ConsumoConsultaCatalogosAcotada, bool) {
	if limites.Validar() != nil ||
		!sumaConsultaCatalogosSegura(c.Versiones, otro.Versiones, limites.Versiones) ||
		!sumaConsultaCatalogosSegura(c.Entradas, otro.Entradas, limites.Entradas) ||
		!sumaConsultaCatalogosSegura(c.Atributos, otro.Atributos, limites.Atributos) ||
		!sumaConsultaCatalogosSegura(
			c.BytesAproximados,
			otro.BytesAproximados,
			limites.BytesAproximados,
		) {
		return ConsumoConsultaCatalogosAcotada{}, false
	}
	return ConsumoConsultaCatalogosAcotada{
		Versiones:        c.Versiones + otro.Versiones,
		Entradas:         c.Entradas + otro.Entradas,
		Atributos:        c.Atributos + otro.Atributos,
		BytesAproximados: c.BytesAproximados + otro.BytesAproximados,
	}, true
}

// MedirCatalogoConfigurable calcula un presupuesto conservador sin clonar. La
// parte fija aproxima cabeceras de slices, mapas, cadenas, enteros y fechas.
func MedirCatalogoConfigurable(
	catalogo domain.CatalogoConfigurable,
) (ConsumoConsultaCatalogosAcotada, bool) {
	consumo := ConsumoConsultaCatalogosAcotada{
		Versiones: 1,
		Entradas:  len(catalogo.Entradas),
	}
	if consumo.Entradas > MaximoEntradasConsultaCatalogosAcotada {
		return ConsumoConsultaCatalogosAcotada{}, false
	}
	bytes := 256
	for _, texto := range []string{
		catalogo.ID,
		catalogo.VersionAnteriorRef,
		catalogo.ModuloID,
		catalogo.Nombre,
		catalogo.Descripcion,
		catalogo.FuenteRef,
		catalogo.MotivoCreacion,
		catalogo.CreadoPor,
		catalogo.UltimaModificacionPor,
		catalogo.MotivoModificacion,
		catalogo.PublicadoPor,
		catalogo.AprobacionRef,
		catalogo.MotivoPublicacion,
		catalogo.RetiradoPor,
		catalogo.RetiradaAprobacionRef,
		catalogo.MotivoRetirada,
	} {
		if !sumarBytesConsultaCatalogos(&bytes, len(texto)) {
			return ConsumoConsultaCatalogosAcotada{}, false
		}
	}
	for _, entrada := range catalogo.Entradas {
		if consumo.Atributos >
			MaximoAtributosConsultaCatalogosAcotada-len(entrada.Atributos) {
			return ConsumoConsultaCatalogosAcotada{}, false
		}
		consumo.Atributos += len(entrada.Atributos)
		if !sumarBytesConsultaCatalogos(&bytes, 96) ||
			!sumarBytesConsultaCatalogos(&bytes, len(entrada.Clave)) ||
			!sumarBytesConsultaCatalogos(&bytes, len(entrada.Etiqueta)) ||
			!sumarBytesConsultaCatalogos(&bytes, len(entrada.Descripcion)) {
			return ConsumoConsultaCatalogosAcotada{}, false
		}
		for clave, valor := range entrada.Atributos {
			if !sumarBytesConsultaCatalogos(&bytes, 32) ||
				!sumarBytesConsultaCatalogos(&bytes, len(clave)) ||
				!sumarBytesConsultaCatalogos(&bytes, len(valor)) {
				return ConsumoConsultaCatalogosAcotada{}, false
			}
		}
	}
	consumo.BytesAproximados = bytes
	return consumo, true
}

type ResultadoConsultaCatalogosAcotada struct {
	Catalogos []domain.CatalogoConfigurable
	Truncado  bool
}

type ResultadoConsultaCatalogoAcotado struct {
	Catalogo domain.CatalogoConfigurable
	Truncado bool
}

// ConsultaCatalogosConfigurablesAcotada es independiente del puerto histórico:
// añadirla no obliga a modificar implementadores que no sirven esta consulta.
type ConsultaCatalogosConfigurablesAcotada interface {
	ObtenerCatalogoAcotado(
		context.Context,
		string,
		int,
		LimitesConsultaCatalogosAcotada,
	) (ResultadoConsultaCatalogoAcotado, error)
	ListarVersionesCatalogoAcotado(
		context.Context,
		string,
		LimitesConsultaCatalogosAcotada,
	) (ResultadoConsultaCatalogosAcotada, error)
}

func sumaConsultaCatalogosSegura(actual, incremento, maximo int) bool {
	return actual >= 0 && incremento >= 0 &&
		actual <= maximo && incremento <= maximo-actual
}

func sumarBytesConsultaCatalogos(actual *int, incremento int) bool {
	if incremento < 0 ||
		*actual > MaximoBytesConsultaCatalogosAcotada-incremento {
		return false
	}
	*actual += incremento
	return true
}

// MetadatosFuenteCatalogos describe el adaptador de lectura sin revelar la
// procedencia interna de cada entrada ni los actores del gobierno del catalogo.
// En produccion estos datos pueden proceder del repositorio publicado; el
// adaptador de fichero los toma del envoltorio DEMO versionado. La huella de
// procedencia declarada no equivale a una firma verificable del paquete.
type MetadatosFuenteCatalogos struct {
	Revision      string
	ActualizadaEn time.Time
	Demostracion  bool
	Aviso         string
}

type ConsultaMetadatosFuenteCatalogos interface {
	ObtenerMetadatosFuenteCatalogos(context.Context) (MetadatosFuenteCatalogos, error)
}

// RepositorioGobiernoCatalogos confirma cada cambio junto con su auditoria y
// outbox. Una version publicada o retirada nunca se sobrescribe.
type RepositorioGobiernoCatalogos interface {
	ConfirmarAltaBorradorCatalogo(context.Context, domain.CatalogoConfigurable, domain.AuditEntry, domain.Event, EvidenciaUsoDecisionAutorizacion) error
	ConfirmarActualizacionBorradorCatalogo(context.Context, string, domain.CatalogoConfigurable, domain.AuditEntry, domain.Event, EvidenciaUsoDecisionAutorizacion) error
	ConfirmarPublicacionCatalogo(context.Context, string, domain.CatalogoConfigurable, domain.AuditEntry, domain.Event, EvidenciaUsoDecisionAutorizacion) error
	ConfirmarRetiradaCatalogo(context.Context, string, domain.CatalogoConfigurable, domain.AuditEntry, domain.Event, EvidenciaUsoDecisionAutorizacion) error
}

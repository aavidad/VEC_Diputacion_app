package catalogosvec

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var _ ports.VerificadorCatalogosImportacionOrganizacion = (*VerificadorCatalogosImportacionOrganizacion)(nil)

var ErrCatalogosImportacionOrganizacionNoDisponibles = errors.New("personal: catalogos de importacion de organizacion no disponibles")

const idCatalogoClasificacionesImportacion = "categorias-profesionales"

// VerificadorCatalogosImportacionOrganizacion lee únicamente los catálogos
// comunes de Personal. La selección de versión, revisión y huella llega en el
// manifiesto y se coteja completa antes de usar una entrada.
type VerificadorCatalogosImportacionOrganizacion struct {
	consulta vecports.ConsultaCatalogosConfigurables
}

func NuevoVerificadorCatalogosImportacionOrganizacion(consulta vecports.ConsultaCatalogosConfigurables) (*VerificadorCatalogosImportacionOrganizacion, error) {
	if dependenciaCatalogosNula(consulta) {
		return nil, ErrCatalogosImportacionOrganizacionNoDisponibles
	}
	return &VerificadorCatalogosImportacionOrganizacion{consulta: consulta}, nil
}

// ValidarReferencias también se usa durante la conciliación. Por ello admite
// borradores exactos; la publicación requiere además la política de fuentes,
// acto y custodia del caso de uso y las guardas durables del repositorio.
func (v *VerificadorCatalogosImportacionOrganizacion) ValidarReferencias(ctx context.Context, unidades, clasificaciones domain.ReferenciaCatalogoImportacion) error {
	_, _, err := v.leer(ctx, unidades, clasificaciones)
	return err
}

func (v *VerificadorCatalogosImportacionOrganizacion) ValidarHechos(ctx context.Context, unidades, clasificaciones domain.ReferenciaCatalogoImportacion, hechos []domain.HechoImportacionOrganizacion) error {
	if len(hechos) == 0 || len(hechos) > 3000 {
		return ErrCatalogosImportacionOrganizacionNoDisponibles
	}
	cu, cc, err := v.leer(ctx, unidades, clasificaciones)
	if err != nil {
		return err
	}
	clavesUnidad := make(map[string]vecdomain.EntradaCatalogoConfigurable, len(cu.Entradas))
	for _, entrada := range cu.Entradas {
		clavesUnidad[entrada.Clave] = entrada
	}
	clavesClasificacion := make(map[string]vecdomain.EntradaCatalogoConfigurable, len(cc.Entradas))
	for _, entrada := range cc.Entradas {
		clavesClasificacion[entrada.Clave] = entrada
	}
	// El nodo aporta el enlace explícito entre su referencia técnica y la
	// clave del catálogo. No se deriva del texto ni del prefijo de la referencia.
	nodos := make(map[string]string)
	for _, hecho := range hechos {
		if hecho.Clase != "nodo" {
			continue
		}
		if hecho.UnidadRef == "" || hecho.CatalogoEntradaClave == "" {
			return ErrCatalogosImportacionOrganizacionNoDisponibles
		}
		if anterior, existe := nodos[hecho.UnidadRef]; existe && anterior != hecho.CatalogoEntradaClave {
			return ErrCatalogosImportacionOrganizacionNoDisponibles
		}
		nodos[hecho.UnidadRef] = hecho.CatalogoEntradaClave
	}
	for _, hecho := range hechos {
		if err := ctx.Err(); err != nil {
			return err
		}
		claveUnidad := hecho.UnidadRef
		if hecho.Clase == "nodo" {
			claveUnidad = hecho.CatalogoEntradaClave
		} else if clave, existe := nodos[hecho.UnidadRef]; existe {
			claveUnidad = clave
		} else {
			return ErrCatalogosImportacionOrganizacionNoDisponibles
		}
		entrada, existe := clavesUnidad[claveUnidad]
		if !existe || !intervaloImportacionCubierto(entrada, hecho.VigenteDesde, hecho.VigenteHasta) {
			return ErrCatalogosImportacionOrganizacionNoDisponibles
		}
		if hecho.Clase == "nodo" && hecho.TipoUnidad != entrada.Atributos["tipo"] {
			return ErrCatalogosImportacionOrganizacionNoDisponibles
		}
		if hecho.ClasificacionRef != "" {
			clasificacion, existe := clavesClasificacion[hecho.ClasificacionRef]
			if !existe || !intervaloImportacionCubierto(clasificacion, hecho.VigenteDesde, hecho.VigenteHasta) {
				return ErrCatalogosImportacionOrganizacionNoDisponibles
			}
		}
	}
	return ctx.Err()
}

func (v *VerificadorCatalogosImportacionOrganizacion) leer(ctx context.Context, unidades, clasificaciones domain.ReferenciaCatalogoImportacion) (vecdomain.CatalogoConfigurable, vecdomain.CatalogoConfigurable, error) {
	var vacio vecdomain.CatalogoConfigurable
	if v == nil || dependenciaCatalogosNula(v.consulta) || ctx == nil ||
		unidades.Validar() != nil || clasificaciones.Validar() != nil ||
		unidades.ID != ports.IDCatalogoOrganizacion || clasificaciones.ID != idCatalogoClasificacionesImportacion {
		return vacio, vacio, ErrCatalogosImportacionOrganizacionNoDisponibles
	}
	if err := ctx.Err(); err != nil {
		return vacio, vacio, err
	}
	cu, err := v.leerUno(ctx, unidades, 1000, true)
	if err != nil {
		return vacio, vacio, err
	}
	cc, err := v.leerUno(ctx, clasificaciones, 1000, false)
	if err != nil {
		return vacio, vacio, err
	}
	return cu, cc, nil
}

func (v *VerificadorCatalogosImportacionOrganizacion) leerUno(ctx context.Context, referencia domain.ReferenciaCatalogoImportacion, maxEntradas int, estructura bool) (vecdomain.CatalogoConfigurable, error) {
	var vacio vecdomain.CatalogoConfigurable
	lectura, err := v.consulta.ObtenerCatalogo(ctx, referencia.ID, referencia.Version)
	if err != nil {
		if ctx.Err() != nil {
			return vacio, ctx.Err()
		}
		return vacio, ErrCatalogosImportacionOrganizacionNoDisponibles
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	medida, ok := vecports.MedirCatalogoConfigurable(lectura)
	if !ok || medida.Entradas == 0 || medida.Entradas > maxEntradas || medida.BytesAproximados > 4<<20 || medida.Atributos > 4096 {
		return vacio, ErrCatalogosImportacionOrganizacionNoDisponibles
	}
	canonico, err := lectura.ClonarCanonico()
	if err != nil || canonico.ModuloID != "personal" || canonico.ID != referencia.ID || canonico.Version != referencia.Version ||
		canonico.Revision != referencia.Revision ||
		(canonico.Estado != vecdomain.EstadoCatalogoBorrador && canonico.Estado != vecdomain.EstadoCatalogoPublicado) {
		return vacio, ErrCatalogosImportacionOrganizacionNoDisponibles
	}
	huella, err := canonico.HuellaSHA256()
	if err != nil || !huellasCatalogoIguales(huella, referencia.HuellaSHA256) {
		return vacio, ErrCatalogosImportacionOrganizacionNoDisponibles
	}
	if estructura {
		if domain.ValidarEstructuraOrganizativa(canonico) != nil {
			return vacio, ErrCatalogosImportacionOrganizacionNoDisponibles
		}
	} else {
		for _, entrada := range canonico.Entradas {
			if _, err := proyectarCategoriaProfesional(entrada); err != nil {
				return vacio, ErrCatalogosImportacionOrganizacionNoDisponibles
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	return canonico, nil
}

func intervaloImportacionCubierto(entrada vecdomain.EntradaCatalogoConfigurable, desde, hasta domain.FechaCivil) bool {
	if desde.Validar() != nil || hasta != "" && hasta.Validar() != nil {
		return false
	}
	inicio, err := time.Parse("2006-01-02", desde.Texto())
	if err != nil || desde.Texto() < entrada.VigenteDesde.UTC().Format("2006-01-02") {
		return false
	}
	if hasta == "" {
		return entrada.VigenteHasta.IsZero()
	}
	fin, err := time.Parse("2006-01-02", hasta.Texto())
	return err == nil && fin.After(inicio) && (entrada.VigenteHasta.IsZero() || hasta.Texto() <= entrada.VigenteHasta.UTC().Format("2006-01-02"))
}

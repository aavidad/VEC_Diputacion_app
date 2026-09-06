package catalogosvec

import (
	"context"
	"errors"
	"strconv"
	"strings"

	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var (
	_                                         personalports.ConsultaEstructuraOrganizativa = (*ConsultaEstructuraOrganizativa)(nil)
	ErrConsultaEstructuraOrganizativaInvalida                                              = errors.New("personal: consulta de estructura organizativa invalida")
	ErrEstructuraOrganizativaNoDisponible                                                  = errors.New("personal: estructura organizativa no disponible")
)

const maximoUnidadesEstructuraOrganizativa = 1000

type ConsultaEstructuraOrganizativa struct {
	fuente  vecports.ConsultaCatalogosConfigurables
	id      string
	version int
}

func NuevaConsultaEstructuraOrganizativa(fuente vecports.ConsultaCatalogosConfigurables, id string, version int) (*ConsultaEstructuraOrganizativa, error) {
	if dependenciaCatalogosNula(fuente) || strings.TrimSpace(id) != id || id == "" || version < 1 {
		return nil, ErrConsultaEstructuraOrganizativaInvalida
	}
	return &ConsultaEstructuraOrganizativa{fuente: fuente, id: id, version: version}, nil
}

func (c *ConsultaEstructuraOrganizativa) Obtener(ctx context.Context) (personalports.EstructuraOrganizativaConsultable, error) {
	if c == nil || dependenciaCatalogosNula(c.fuente) || ctx == nil || c.id == "" || c.version < 1 {
		return personalports.EstructuraOrganizativaConsultable{}, ErrConsultaEstructuraOrganizativaInvalida
	}
	if err := ctx.Err(); err != nil {
		return personalports.EstructuraOrganizativaConsultable{}, err
	}
	catalogo, err := c.fuente.ObtenerCatalogo(ctx, c.id, c.version)
	if err != nil {
		return personalports.EstructuraOrganizativaConsultable{}, err
	}
	if err := ctx.Err(); err != nil {
		return personalports.EstructuraOrganizativaConsultable{}, err
	}
	if len(catalogo.Entradas) > maximoUnidadesEstructuraOrganizativa {
		return personalports.EstructuraOrganizativaConsultable{}, ErrEstructuraOrganizativaNoDisponible
	}
	canonico, err := catalogo.ClonarCanonico()
	if err != nil || canonico.ID != c.id || canonico.Version != c.version ||
		(canonico.Estado != vecdomain.EstadoCatalogoBorrador && canonico.Estado != vecdomain.EstadoCatalogoPublicado) {
		return personalports.EstructuraOrganizativaConsultable{}, ErrEstructuraOrganizativaNoDisponible
	}
	huella, err := canonico.HuellaSHA256()
	if err != nil {
		return personalports.EstructuraOrganizativaConsultable{}, ErrEstructuraOrganizativaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return personalports.EstructuraOrganizativaConsultable{}, err
	}
	resultado := personalports.EstructuraOrganizativaConsultable{Esquema: personalports.EsquemaEstructuraOrganizativa, CatalogoID: canonico.ID, CatalogoVersion: canonico.Version, CatalogoHuellaSHA256: huella, Estado: string(canonico.Estado), FuenteRef: canonico.FuenteRef, Descripcion: canonico.Descripcion, Unidades: make([]personalports.UnidadEstructuraOrganizativa, 0, len(canonico.Entradas))}
	claves := make(map[string]struct{}, len(canonico.Entradas))
	padres := make(map[string]string, len(canonico.Entradas))
	for _, entrada := range canonico.Entradas {
		if err := ctx.Err(); err != nil {
			return personalports.EstructuraOrganizativaConsultable{}, err
		}
		tipo := entrada.Atributos["tipo"]
		switch tipo {
		case "delegacion", "centro", "puesto_responsabilidad":
		default:
			return personalports.EstructuraOrganizativaConsultable{}, ErrEstructuraOrganizativaNoDisponible
		}
		if entrada.Clave == "" || entrada.Etiqueta == "" || entrada.Etiqueta != strings.TrimSpace(entrada.Etiqueta) {
			return personalports.EstructuraOrganizativaConsultable{}, ErrEstructuraOrganizativaNoDisponible
		}
		if _, ok := claves[entrada.Clave]; ok {
			return personalports.EstructuraOrganizativaConsultable{}, ErrEstructuraOrganizativaNoDisponible
		}
		claves[entrada.Clave] = struct{}{}
		padre := entrada.Atributos["adscripcion_clave"]
		if padre == entrada.Clave {
			return personalports.EstructuraOrganizativaConsultable{}, ErrEstructuraOrganizativaNoDisponible
		}
		if padre != "" {
			padres[entrada.Clave] = padre
		}
		unidad := personalports.UnidadEstructuraOrganizativa{Clave: entrada.Clave, Etiqueta: entrada.Etiqueta, Tipo: tipo, AdscripcionClave: padre, CodigoFuente: entrada.Atributos["codigo_fuente"]}
		if pagina := entrada.Atributos["pagina_fuente"]; pagina != "" {
			n, err := strconv.Atoi(pagina)
			if err != nil || n < 1 || n > 10000 {
				return personalports.EstructuraOrganizativaConsultable{}, ErrEstructuraOrganizativaNoDisponible
			}
			unidad.PaginaFuente = n
		}
		resultado.Unidades = append(resultado.Unidades, unidad)
	}
	for clave, padre := range padres {
		if _, ok := claves[padre]; !ok {
			return personalports.EstructuraOrganizativaConsultable{}, ErrEstructuraOrganizativaNoDisponible
		}
		vistos := map[string]bool{}
		for actual := clave; actual != ""; actual = padres[actual] {
			if vistos[actual] {
				return personalports.EstructuraOrganizativaConsultable{}, ErrEstructuraOrganizativaNoDisponible
			}
			vistos[actual] = true
		}
	}
	if err := ctx.Err(); err != nil {
		return personalports.EstructuraOrganizativaConsultable{}, err
	}
	return resultado, nil
}

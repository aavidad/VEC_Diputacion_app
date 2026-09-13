package bootstrap

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	atributoClaveI18nMotivoRectificacionAnalisisDesarrollo = "clave_i18n"
	prefijoClaveI18nMotivoRectificacionAnalisisDesarrollo  = "contratacion_temporal.analisis.rectificacion."
	catalogoIDMotivosRectificacionAnalisisDesarrollo       = "motivos_rectificacion_analisis"
	moduloIDMotivosRectificacionAnalisisDesarrollo         = "contratacion_temporal"
)

// fuenteMotivosRectificacionAnalisisDesarrollo lee únicamente una publicación
// de catálogo que composición haya identificado. Una fuente ausente, truncada
// o no verificable no ofrece motivos: la rectificación permanece denegada.
type fuenteMotivosRectificacionAnalisisDesarrollo struct {
	consulta   vecports.ConsultaCatalogosConfigurablesAcotada
	catalogoID string
	moduloID   string
	reloj      ports.Reloj
}

func nuevaFuenteMotivosRectificacionAnalisisDesarrollo(
	consulta vecports.ConsultaCatalogosConfigurablesAcotada,
	catalogoID string,
	moduloID string,
	reloj ports.Reloj,
) fuenteMotivosRectificacionAnalisisDesarrollo {
	return fuenteMotivosRectificacionAnalisisDesarrollo{
		consulta: consulta, catalogoID: catalogoID, moduloID: moduloID, reloj: reloj,
	}
}

// nuevaFuenteMotivosRectificacionAnalisisDesarrolloConfigurada compone solo
// una fuente DEMO explícitamente declarada. El adaptador valida el paquete al
// arrancar; su ausencia mantiene la rectificación sin motivos publicados.
func nuevaFuenteMotivosRectificacionAnalisisDesarrolloConfigurada(
	cfg config.Config,
	reloj ports.Reloj,
) (fuenteMotivosRectificacionAnalisisDesarrollo, error) {
	ruta := strings.TrimSpace(cfg.CTAnalisisMotivosSourcePath)
	if ruta == "" {
		return fuenteMotivosRectificacionAnalisisDesarrollo{}, nil
	}
	consulta, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		return fuenteMotivosRectificacionAnalisisDesarrollo{}, err
	}
	return nuevaFuenteMotivosRectificacionAnalisisDesarrollo(
		consulta,
		catalogoIDMotivosRectificacionAnalisisDesarrollo,
		moduloIDMotivosRectificacionAnalisisDesarrollo,
		reloj,
	), nil
}

func (f fuenteMotivosRectificacionAnalisisDesarrollo) opciones(
	ctx context.Context,
) []opcionClaveCatalogosAltaContratacionTemporalDesarrollo {
	catalogo, instante, _, encontrado := f.catalogoVigente(ctx)
	if !encontrado {
		return []opcionClaveCatalogosAltaContratacionTemporalDesarrollo{}
	}
	return opcionesMotivosRectificacionAnalisis(catalogo, instante)
}

// resolverMotivo fija la entrada que se utilizará en la autorización. No usa
// las opciones de presentación: vuelve a resolver el catálogo publicado para
// que una clave recibida del cliente nunca equivalga por sí sola a un motivo.
func (f fuenteMotivosRectificacionAnalisisDesarrollo) resolverMotivo(
	ctx context.Context,
	clave domain.ClaveCatalogo,
) (ports.MotivoRectificacionGobernado, error) {
	vacio := ports.MotivoRectificacionGobernado{}
	if !clave.Valida() {
		return vacio, ports.ErrPoliticaOperacionAnalisisNoDisponible
	}
	catalogo, instante, limiteCatalogo, encontrado := f.catalogoVigente(ctx)
	if !encontrado {
		return vacio, ports.ErrPoliticaOperacionAnalisisNoDisponible
	}
	for _, entrada := range catalogo.Entradas {
		claveI18n, existe := entrada.Atributos[atributoClaveI18nMotivoRectificacionAnalisisDesarrollo]
		if entrada.Clave != string(clave) || !entrada.VigenteEn(instante) ||
			!existe || !domain.ClaveCatalogo(claveI18n).Valida() ||
			!strings.HasPrefix(claveI18n, prefijoClaveI18nMotivoRectificacionAnalisisDesarrollo) {
			continue
		}
		huella, err := catalogo.HuellaSHA256()
		if err != nil {
			return vacio, ports.ErrPoliticaOperacionAnalisisNoDisponible
		}
		motivo := ports.MotivoRectificacionGobernado{
			ReferenciaCatalogo: vecdomain.ReferenciaEntradaCatalogo{
				CatalogoID: catalogo.ID, CatalogoVersion: catalogo.Version,
				CatalogoHuellaSHA256: huella, EntradaClave: entrada.Clave,
			},
			ClaveMensajeI18N: domain.ClaveCatalogo(claveI18n),
			VigenteDesde:     maximoInstanteMotivoRectificacionAnalisis(entrada.VigenteDesde, catalogo.PublicadoEn),
			VigenteHasta: limiteVigenciaMotivoRectificacionAnalisis(
				entrada.VigenteHasta,
				limiteCatalogo,
			),
		}
		if motivo.ValidarPara(clave) == nil {
			return motivo, nil
		}
	}
	return vacio, ports.ErrPoliticaOperacionAnalisisNoDisponible
}

func (f fuenteMotivosRectificacionAnalisisDesarrollo) catalogoVigente(
	ctx context.Context,
) (vecdomain.CatalogoConfigurable, time.Time, time.Time, bool) {
	if !f.configuracionValida() || ctx == nil || ctx.Err() != nil {
		return vecdomain.CatalogoConfigurable{}, time.Time{}, time.Time{}, false
	}
	limites := limitesMotivosRectificacionAnalisisDesarrollo()
	resultado, err := f.consulta.ListarVersionesCatalogoAcotado(ctx, f.catalogoID, limites)
	if err != nil || ctx.Err() != nil || resultado.Truncado ||
		len(resultado.Catalogos) == 0 || len(resultado.Catalogos) > limites.Versiones {
		return vecdomain.CatalogoConfigurable{}, time.Time{}, time.Time{}, false
	}
	versiones := make([]vecdomain.CatalogoConfigurable, len(resultado.Catalogos))
	var consumo vecports.ConsumoConsultaCatalogosAcotada
	for indice := range resultado.Catalogos {
		medida, medible := vecports.MedirCatalogoConfigurable(resultado.Catalogos[indice])
		siguiente, cabe := consumo.Agregar(medida, limites)
		if !medible || !cabe || ctx.Err() != nil {
			return vecdomain.CatalogoConfigurable{}, time.Time{}, time.Time{}, false
		}
		catalogo, err := resultado.Catalogos[indice].ClonarCanonico()
		if err != nil || catalogo.ID != f.catalogoID || catalogo.ModuloID != f.moduloID {
			return vecdomain.CatalogoConfigurable{}, time.Time{}, time.Time{}, false
		}
		versiones[indice] = catalogo
		consumo = siguiente
	}
	sort.Slice(versiones, func(primera, segunda int) bool {
		return versiones[primera].Version < versiones[segunda].Version
	})
	if !historialMotivosRectificacionAnalisisValido(versiones) {
		return vecdomain.CatalogoConfigurable{}, time.Time{}, time.Time{}, false
	}
	instante := f.reloj.Ahora()
	if !domain.InstanteUTCCanonico(instante) {
		return vecdomain.CatalogoConfigurable{}, time.Time{}, time.Time{}, false
	}
	catalogo, encontrado := catalogoMotivosRectificacionAnalisisVigente(versiones, instante)
	if !encontrado {
		return vecdomain.CatalogoConfigurable{}, time.Time{}, time.Time{}, false
	}
	return catalogo, instante,
		limiteCatalogoMotivosRectificacionAnalisis(versiones, catalogo, instante), true
}

func limiteCatalogoMotivosRectificacionAnalisis(
	versiones []vecdomain.CatalogoConfigurable,
	catalogo vecdomain.CatalogoConfigurable,
	instante time.Time,
) time.Time {
	limite := time.Time{}
	if catalogo.Estado == vecdomain.EstadoCatalogoRetirado {
		limite = catalogo.RetiradoEn
	}
	for _, posterior := range versiones {
		if posterior.Version <= catalogo.Version ||
			posterior.Estado == vecdomain.EstadoCatalogoBorrador ||
			!posterior.PublicadoEn.After(instante) {
			continue
		}
		limite = limiteVigenciaMotivoRectificacionAnalisis(limite, posterior.PublicadoEn)
	}
	return limite
}

func maximoInstanteMotivoRectificacionAnalisis(primero, segundo time.Time) time.Time {
	if primero.After(segundo) {
		return primero
	}
	return segundo
}

func limiteVigenciaMotivoRectificacionAnalisis(primero, segundo time.Time) time.Time {
	if primero.IsZero() || (!segundo.IsZero() && segundo.Before(primero)) {
		return segundo
	}
	return primero
}

func (f fuenteMotivosRectificacionAnalisisDesarrollo) configuracionValida() bool {
	if dependenciaMotivosRectificacionAnalisisNula(f.consulta) ||
		dependenciaMotivosRectificacionAnalisisNula(f.reloj) ||
		f.catalogoID != strings.TrimSpace(f.catalogoID) ||
		f.moduloID != strings.TrimSpace(f.moduloID) {
		return false
	}
	centinela := vecdomain.CatalogoConfigurable{
		ID: f.catalogoID, Version: 1, Revision: 1, ModuloID: f.moduloID,
		Nombre: "Motivos de rectificación de análisis", FuenteRef: "catalogo_configurable",
		MotivoCreacion: "Composición de lectura.", Estado: vecdomain.EstadoCatalogoBorrador,
		CreadoPor: "composicion_aplicacion", CreadoEn: time.Unix(0, 0).UTC(),
	}
	return centinela.Validar() == nil
}

func limitesMotivosRectificacionAnalisisDesarrollo() vecports.LimitesConsultaCatalogosAcotada {
	return vecports.LimitesConsultaCatalogosAcotada{
		Versiones: 32, Entradas: 100, Atributos: 200,
		BytesAproximados: 128 << 10,
	}
}

func historialMotivosRectificacionAnalisisValido(
	versiones []vecdomain.CatalogoConfigurable,
) bool {
	for indice := range versiones {
		actual := versiones[indice]
		if actual.Validar() != nil || actual.Version != indice+1 {
			return false
		}
		if indice == 0 {
			continue
		}
		anterior := versiones[indice-1]
		if actual.VersionAnteriorRef != anterior.Referencia() ||
			actual.CreadoEn.Before(anterior.CreadoEn) ||
			anterior.Estado == vecdomain.EstadoCatalogoBorrador ||
			actual.CreadoEn.Before(anterior.PublicadoEn) ||
			(actual.Estado != vecdomain.EstadoCatalogoBorrador &&
				actual.PublicadoEn.Before(anterior.PublicadoEn)) {
			return false
		}
	}
	return true
}

func catalogoMotivosRectificacionAnalisisVigente(
	versiones []vecdomain.CatalogoConfigurable,
	instante time.Time,
) (vecdomain.CatalogoConfigurable, bool) {
	for indice := len(versiones) - 1; indice >= 0; indice-- {
		catalogo := versiones[indice]
		if catalogo.Estado == vecdomain.EstadoCatalogoBorrador ||
			catalogo.PublicadoEn.After(instante) {
			continue
		}
		if catalogo.Estado == vecdomain.EstadoCatalogoPublicado ||
			(catalogo.Estado == vecdomain.EstadoCatalogoRetirado &&
				catalogo.RetiradoEn.After(instante)) {
			return catalogo, true
		}
		return vecdomain.CatalogoConfigurable{}, false
	}
	return vecdomain.CatalogoConfigurable{}, false
}

func opcionesMotivosRectificacionAnalisis(
	catalogo vecdomain.CatalogoConfigurable,
	instante time.Time,
) []opcionClaveCatalogosAltaContratacionTemporalDesarrollo {
	opciones := make([]opcionClaveCatalogosAltaContratacionTemporalDesarrollo, 0)
	for _, entrada := range catalogo.Entradas {
		claveI18n, existe := entrada.Atributos[atributoClaveI18nMotivoRectificacionAnalisisDesarrollo]
		if !entrada.VigenteEn(instante) || !existe ||
			!domain.ClaveCatalogo(entrada.Clave).Valida() ||
			!domain.ClaveCatalogo(claveI18n).Valida() ||
			!strings.HasPrefix(claveI18n, prefijoClaveI18nMotivoRectificacionAnalisisDesarrollo) {
			continue
		}
		opciones = append(opciones,
			opcionClaveCatalogosAltaContratacionTemporalDesarrollo{
				Clave: entrada.Clave, Etiqueta: entrada.Etiqueta,
			},
		)
	}
	if len(opciones) == 0 || len(opciones) > 100 {
		return []opcionClaveCatalogosAltaContratacionTemporalDesarrollo{}
	}
	return opciones
}

func dependenciaMotivosRectificacionAnalisisNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"

	bolsapersonal "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const superficieInternaSeguridadComunDesarrollo = "interna-corporativa"
const superficieExternaPersonalSeguridadComunDesarrollo = "externa-personal"

var ErrSeguridadComunDesarrolloDenegada = errors.New("bootstrap: seguridad común denegada")

// descriptorFronteraComunDesarrollo declara una frontera de composición. No
// concede acceso. DetalleColeccion describe exclusivamente colección/{id}.
type descriptorFronteraComunDesarrollo struct {
	Clave              string
	Superficie         string
	Metodo             string
	Ruta               string
	PerfilesActivosRef []string
	ClavePolitica      string
	ClaveCapacidad     string
	DetalleColeccion   bool
	PlantillaDetalle   []string
}

type catalogoFronterasComunDesarrollo struct {
	identidad *identidadCatalogoFronterasComunDesarrollo
	porClave  map[string]descriptorFronteraComunDesarrollo
}

// La identidad no representa un permiso. Impide que un contexto fabricado con
// claves iguales pero procedente de otro catálogo pueda cruzar la composición.
// token evita el tamaño cero: Go puede reutilizar direcciones de objetos de
// tamaño cero, lo que destruiría la separación por identidad de instancia.
type identidadCatalogoFronterasComunDesarrollo struct{ token byte }

// nuevoCatalogoFronterasComunDesarrollo toma una copia de las declaraciones y
// rechaza toda ambigüedad. Un catálogo vacío es válido para un ensamblaje sin
// módulos, pero una petición contra él queda denegada.
func nuevoCatalogoFronterasComunDesarrollo(
	descriptores []descriptorFronteraComunDesarrollo,
) (catalogoFronterasComunDesarrollo, error) {
	catalogo := catalogoFronterasComunDesarrollo{
		identidad: &identidadCatalogoFronterasComunDesarrollo{token: 1},
		porClave:  make(map[string]descriptorFronteraComunDesarrollo, len(descriptores)),
	}
	for _, descriptor := range descriptores {
		if !descriptor.valida() {
			return catalogoFronterasComunDesarrollo{}, ErrSeguridadComunDesarrolloDenegada
		}
		if _, existe := catalogo.porClave[descriptor.Clave]; existe {
			return catalogoFronterasComunDesarrollo{}, ErrSeguridadComunDesarrolloDenegada
		}
		for _, previo := range catalogo.porClave {
			if colisionanFronterasComunDesarrollo(previo, descriptor) {
				return catalogoFronterasComunDesarrollo{}, ErrSeguridadComunDesarrolloDenegada
			}
		}
		descriptor.PerfilesActivosRef = append([]string(nil), descriptor.PerfilesActivosRef...)
		catalogo.porClave[descriptor.Clave] = descriptor
	}
	return catalogo, nil
}

func (c catalogoFronterasComunDesarrollo) mismaInstancia(otro catalogoFronterasComunDesarrollo) bool {
	return c.identidad != nil && c.identidad == otro.identidad
}

func (d descriptorFronteraComunDesarrollo) valida() bool {
	if d.Clave == "" || (d.Superficie != superficieInternaSeguridadComunDesarrollo &&
		!(d.Superficie == superficieExternaPersonalSeguridadComunDesarrollo && len(bolsapersonal.AccionPortalEn(d.Metodo, d.Ruta)) != 0)) ||
		d.Ruta == "" || d.ClavePolitica == "" || d.ClaveCapacidad == "" ||
		!claveCatalogoComunValida(d.Clave) || !claveCatalogoComunValida(d.ClavePolitica) ||
		!claveCatalogoComunValida(d.ClaveCapacidad) || !rutaCatalogoComunValida(d.Ruta) ||
		len(d.PerfilesActivosRef) == 0 || (d.DetalleColeccion && len(d.PlantillaDetalle) != 0) {
		return false
	}
	vistos := make(map[string]struct{}, len(d.PerfilesActivosRef))
	for _, perfil := range d.PerfilesActivosRef {
		if !perfilActivoSeguridadComunValido(perfil) {
			return false
		}
		if _, duplicado := vistos[perfil]; duplicado {
			return false
		}
		vistos[perfil] = struct{}{}
	}
	for _, segmento := range d.PlantillaDetalle {
		if segmento == "" || strings.ContainsAny(segmento, "/?# \t\r\n") {
			return false
		}
	}
	switch d.Metodo {
	case http.MethodGet, http.MethodPost, http.MethodHead, http.MethodPut,
		http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func (d descriptorFronteraComunDesarrollo) admitePerfil(perfil string) bool {
	for _, permitido := range d.PerfilesActivosRef {
		if permitido == perfil {
			return true
		}
	}
	return false
}

func claveCatalogoComunValida(clave string) bool {
	return clave != "" && !strings.ContainsAny(clave, " \t\r\n*?#")
}

func rutaCatalogoComunValida(ruta string) bool {
	return strings.HasPrefix(ruta, "/") && ruta != "/" &&
		!strings.ContainsAny(ruta, "?# \t\r\n") && !strings.HasSuffix(ruta, "/") &&
		!strings.Contains(ruta, "//")
}

func perfilActivoSeguridadComunValido(perfil string) bool {
	return strings.HasPrefix(perfil, "prf_") && len(perfil) > len("prf_") &&
		!strings.ContainsAny(perfil, " \t\r\n*/")
}

func colisionanFronterasComunDesarrollo(a, b descriptorFronteraComunDesarrollo) bool {
	if a.Superficie != b.Superficie || a.Metodo != b.Metodo {
		return false
	}
	segmentosA, variablesA := patronFronteraComunDesarrollo(a)
	segmentosB, variablesB := patronFronteraComunDesarrollo(b)
	if len(segmentosA) != len(segmentosB) {
		return false
	}
	for i := range segmentosA {
		if !variablesA[i] && !variablesB[i] && segmentosA[i] != segmentosB[i] {
			return false
		}
	}
	return true
}

// Ruta conserva segmentos literales; solo '*' en PlantillaDetalle y el
// segmento de DetalleColeccion aceptan cualquier valor no vacío. El catálogo
// ya validó rutas y plantillas antes de comparar sus intersecciones.
func patronFronteraComunDesarrollo(d descriptorFronteraComunDesarrollo) ([]string, []bool) {
	segmentos := strings.Split(strings.TrimPrefix(d.Ruta, "/"), "/")
	variables := make([]bool, len(segmentos))
	if d.DetalleColeccion {
		return append(segmentos, "*"), append(variables, true)
	}
	for _, segmento := range d.PlantillaDetalle {
		segmentos = append(segmentos, segmento)
		variables = append(variables, segmento == "*")
	}
	return segmentos, variables
}

func (d descriptorFronteraComunDesarrollo) coincide(metodo, ruta string) bool {
	if d.Metodo != metodo {
		return false
	}
	if !d.DetalleColeccion {
		if len(d.PlantillaDetalle) == 0 {
			return d.Ruta == ruta
		}
		resto, presente := strings.CutPrefix(ruta, d.Ruta+"/")
		if !presente {
			return false
		}
		segmentos := strings.Split(resto, "/")
		if len(segmentos) != len(d.PlantillaDetalle) {
			return false
		}
		for i, esperado := range d.PlantillaDetalle {
			if segmentos[i] == "" || (esperado != "*" && segmentos[i] != esperado) {
				return false
			}
		}
		return true
	}
	detalle, presente := strings.CutPrefix(ruta, d.Ruta+"/")
	return presente && detalle != "" && !strings.Contains(detalle, "/")
}

func (c catalogoFronterasComunDesarrollo) resolver(
	metodo, ruta string,
) (descriptorFronteraComunDesarrollo, bool) {
	for _, descriptor := range c.porClave {
		if descriptor.coincide(metodo, ruta) {
			descriptor.PerfilesActivosRef = append([]string(nil), descriptor.PerfilesActivosRef...)
			return descriptor, true
		}
	}
	return descriptorFronteraComunDesarrollo{}, false
}

type contextoSeguridadComunDesarrollo struct {
	Vinculo   vecdomain.VinculoAutenticacionActorV2
	Resultado vecdomain.ResultadoContextoActorRegistradoV2
}

type claveFronteraSeguridadComunDesarrollo struct{}

type fronteraSeguridadComunDesarrollo struct {
	metodo, ruta, superficie string
	catalogo                 catalogoFronterasComunDesarrollo
	descriptor               descriptorFronteraComunDesarrollo
}

func fronteraSeguridadComunDesdeContexto(
	ctx context.Context,
) (fronteraSeguridadComunDesarrollo, bool) {
	if ctx == nil {
		return fronteraSeguridadComunDesarrollo{}, false
	}
	frontera, ok := ctx.Value(claveFronteraSeguridadComunDesarrollo{}).(fronteraSeguridadComunDesarrollo)
	if !ok {
		return fronteraSeguridadComunDesarrollo{}, false
	}
	descriptor, ok := frontera.catalogo.resolver(frontera.metodo, frontera.ruta)
	if !ok || descriptor.Superficie != frontera.superficie {
		return fronteraSeguridadComunDesarrollo{}, false
	}
	frontera.descriptor = descriptor
	return frontera, true
}

type resolutorContextoSeguridadComunDesarrollo interface {
	ResolverContexto(context.Context) (contextoSeguridadComunDesarrollo, error)
}

type seguridadComunDesarrollo struct {
	sesion resolutorContextoSeguridadComunDesarrollo
	reloj  vecports.Reloj
}

func nuevaSeguridadComunDesarrollo(
	sesion resolutorContextoSeguridadComunDesarrollo,
	reloj vecports.Reloj,
) (*seguridadComunDesarrollo, error) {
	if dependenciaAutorizacionComunDesarrolloNula(sesion) || dependenciaAutorizacionComunDesarrolloNula(reloj) {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	return &seguridadComunDesarrollo{sesion: sesion, reloj: reloj}, nil
}

type claveCacheSeguridadComunDesarrollo struct{}

type cacheSeguridadComunDesarrollo struct {
	unaVez   sync.Once
	contexto contextoSeguridadComunDesarrollo
	err      error
}

func (s *seguridadComunDesarrollo) ResolverContexto(
	ctx context.Context,
) (contextoSeguridadComunDesarrollo, error) {
	if s == nil || dependenciaAutorizacionComunDesarrolloNula(s.sesion) || dependenciaAutorizacionComunDesarrolloNula(s.reloj) ||
		ctx == nil || ctx.Err() != nil {
		return contextoSeguridadComunDesarrollo{}, ErrSeguridadComunDesarrolloDenegada
	}
	frontera, ok := fronteraSeguridadComunDesdeContexto(ctx)
	if !ok {
		return contextoSeguridadComunDesarrollo{}, ErrSeguridadComunDesarrolloDenegada
	}
	cache, ok := ctx.Value(claveCacheSeguridadComunDesarrollo{}).(*cacheSeguridadComunDesarrollo)
	if !ok || cache == nil {
		cache = &cacheSeguridadComunDesarrollo{}
	}
	cache.unaVez.Do(func() { cache.contexto, cache.err = s.sesion.ResolverContexto(ctx) })
	if cache.err != nil {
		if errors.Is(cache.err, ct.ErrConsultaRRHHNoDisponible) {
			return contextoSeguridadComunDesarrollo{}, ct.ErrConsultaRRHHNoDisponible
		}
		return contextoSeguridadComunDesarrollo{}, ErrSeguridadComunDesarrolloDenegada
	}
	if cache.contexto.Resultado.Validar() != nil ||
		cache.contexto.Vinculo.ValidarPara(cache.contexto.Resultado) != nil ||
		!cache.contexto.Vinculo.VigenteEn(s.reloj.Ahora(), cache.contexto.Resultado) ||
		!frontera.descriptor.admitePerfil(cache.contexto.Resultado.Contexto.PerfilActivoRef) {
		return contextoSeguridadComunDesarrollo{}, ErrSeguridadComunDesarrolloDenegada
	}
	resultado, err := cache.contexto.Resultado.Clonar()
	if err != nil {
		return contextoSeguridadComunDesarrollo{}, ErrSeguridadComunDesarrolloDenegada
	}
	return contextoSeguridadComunDesarrollo{Vinculo: cache.contexto.Vinculo, Resultado: resultado}, nil
}

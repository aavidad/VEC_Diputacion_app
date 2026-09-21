package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const superficieInternaSeguridadComunDesarrollo = "interna-corporativa"

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
	if d.Clave == "" || d.Superficie != superficieInternaSeguridadComunDesarrollo ||
		d.Ruta == "" || d.ClavePolitica == "" || d.ClaveCapacidad == "" ||
		!claveCatalogoComunValida(d.Clave) || !claveCatalogoComunValida(d.ClavePolitica) ||
		!claveCatalogoComunValida(d.ClaveCapacidad) || !rutaCatalogoComunValida(d.Ruta) ||
		len(d.PerfilesActivosRef) == 0 {
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
	if a.DetalleColeccion == b.DetalleColeccion && a.Ruta == b.Ruta {
		return true
	}
	if a.DetalleColeccion && !b.DetalleColeccion && rutaEsDetalleCatalogoComun(a.Ruta, b.Ruta) {
		return true
	}
	return b.DetalleColeccion && !a.DetalleColeccion && rutaEsDetalleCatalogoComun(b.Ruta, a.Ruta)
}

func rutaEsDetalleCatalogoComun(coleccion, ruta string) bool {
	detalle, ok := strings.CutPrefix(ruta, coleccion+"/")
	return ok && detalle != "" && !strings.Contains(detalle, "/")
}

func (d descriptorFronteraComunDesarrollo) coincide(metodo, ruta string) bool {
	if d.Metodo != metodo {
		return false
	}
	if !d.DetalleColeccion {
		return d.Ruta == ruta
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
	if !ok || frontera.superficie != superficieInternaSeguridadComunDesarrollo {
		return fronteraSeguridadComunDesarrollo{}, false
	}
	descriptor, ok := frontera.catalogo.resolver(frontera.metodo, frontera.ruta)
	if !ok {
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
	if cache.err != nil || cache.contexto.Resultado.Validar() != nil ||
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

// Package gobiernov3lector consume publicaciones de confianza V3 sin publicar
// ni modificar el gobierno. La fuente comprueba ACL, puntero y revocaciones.
package gobiernov3lector

import (
	"context"
	"errors"
	"sync"
	"time"

	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

var ErrGobiernoNoDisponible = errors.New("gobierno V3 no disponible")

type Reloj interface{ Ahora() time.Time }

// Publicacion contiene exclusivamente los datos públicos de una revisión.
type Publicacion struct {
	Revision     string
	Secuencia    uint64
	HuellaSHA256 string
	PublicadaEn  time.Time
	ExpiraEn     time.Time
}

// Fuente debe devolver el puntero vigente y comprobarlo junto con la revisión
// anterior y sus revocaciones bajo una única instantánea del gobierno.
type Fuente func(context.Context, Publicacion) (Publicacion, error)

type Lector struct {
	mu       sync.Mutex
	anterior Publicacion
	raiz     confianza.RaizPublicaAtestacionAutorizacionV3
	reloj    Reloj
	fuente   Fuente
	actual   *confianza.ServicioConfianzaAtestacionAutorizacionV3
}

func Nuevo(anterior Publicacion, raiz confianza.RaizPublicaAtestacionAutorizacionV3, reloj Reloj, fuente Fuente) (*Lector, error) {
	if reloj == nil || fuente == nil || configurar(anterior, raiz) == nil {
		return nil, ErrGobiernoNoDisponible
	}
	return &Lector{anterior: anterior, raiz: raiz, reloj: reloj, fuente: fuente}, nil
}

// Instantanea lee el gobierno en cada uso, también antes del vencimiento: una
// revocación o cambio de puntero debe cortar la siguiente operación.
func (l *Lector) Instantanea(ctx context.Context) (*confianza.ServicioConfianzaAtestacionAutorizacionV3, error) {
	servicio, _, err := l.Leer(ctx)
	return servicio, err
}

// Leer entrega servicio y coordenadas de la misma lectura validada.
func (l *Lector) Leer(ctx context.Context) (*confianza.ServicioConfianzaAtestacionAutorizacionV3, Publicacion, error) {
	if l == nil || ctx == nil || ctx.Err() != nil {
		return nil, Publicacion{}, ErrGobiernoNoDisponible
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	actual, err := l.fuente(ctx, l.anterior)
	if err != nil || ctx.Err() != nil || actual.Secuencia < l.anterior.Secuencia ||
		(actual.Secuencia == l.anterior.Secuencia && !mismaPublicacion(actual, l.anterior)) {
		return nil, Publicacion{}, ErrGobiernoNoDisponible
	}
	config := configurar(actual, l.raiz)
	ahora := l.reloj.Ahora().UTC()
	if config == nil || ahora.Before(actual.PublicadaEn) || !ahora.Before(actual.ExpiraEn) || ctx.Err() != nil {
		return nil, Publicacion{}, ErrGobiernoNoDisponible
	}
	if l.actual != nil && mismaPublicacion(actual, l.anterior) {
		return l.actual, actual, nil
	}
	servicio, err := confianza.NuevoServicioConfianzaAtestacionAutorizacionV3(*config, l.reloj)
	if err != nil || ctx.Err() != nil {
		return nil, Publicacion{}, ErrGobiernoNoDisponible
	}
	l.anterior = actual
	l.actual = servicio
	return servicio, actual, nil
}

func (l *Lector) Verificar(ctx context.Context, solicitud core.SolicitudAutorizacionLigadaV3, decision core.DecisionAutorizacionLigadaV3, motivo core.ReferenciaEntradaCatalogo, resultado core.ResultadoContextoActorRegistradoV2, atestacion vp.AtestacionAutorizacionV3) (confianza.PruebaConfianzaAtestacionAutorizacionV3, error) {
	servicio, err := l.Instantanea(ctx)
	if err != nil {
		return confianza.PruebaConfianzaAtestacionAutorizacionV3{}, err
	}
	return servicio.Verificar(ctx, solicitud, decision, motivo, resultado, atestacion)
}

func mismaPublicacion(a, b Publicacion) bool {
	return a.Revision == b.Revision && a.Secuencia == b.Secuencia && a.HuellaSHA256 == b.HuellaSHA256 &&
		a.PublicadaEn.Equal(b.PublicadaEn) && a.ExpiraEn.Equal(b.ExpiraEn)
}

func configurar(p Publicacion, raiz confianza.RaizPublicaAtestacionAutorizacionV3) *confianza.ConfiguracionConfianzaAtestacionAutorizacionV3 {
	config, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(p.Revision, p.Secuencia, p.PublicadaEn, p.ExpiraEn, raiz)
	if err != nil || config.ValidarHuellaSHA256Esperada(p.HuellaSHA256) != nil {
		return nil
	}
	return &config
}

package internagobierno

import (
	"context"
	"strings"

	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type claveContextoPersonalB2 struct{}
type contextoPersonalB2Sellado struct {
	fuente *FuenteF1
	valor  ct.ContextoAutorizacionAltaV3
}

// VincularContextoPersonalB2 crea una sola prueba F1, después de mTLS. La
// clave contextual es privada de esta composición y no procede de HTTP.
func (f *FuenteF1) VincularContextoPersonalB2(ctx context.Context) (context.Context, error) {
	if f == nil || ctx == nil || ctx.Err() != nil || ctx.Value(claveContextoPersonalB2{}) != nil {
		return nil, ErrGobiernoInternoNoDisponible
	}
	resuelta, err := f.ResolverContexto(ctx)
	if err != nil || resuelta.Resultado.Validar() != nil ||
		resuelta.Vinculo.ValidarPara(resuelta.Resultado) != nil ||
		!resuelta.Vinculo.VigenteEn(f.reloj.Ahora(), resuelta.Resultado) {
		return nil, ErrGobiernoInternoNoDisponible
	}
	clon, err := resuelta.Resultado.Clonar()
	if err != nil {
		return nil, ErrGobiernoInternoNoDisponible
	}
	resuelta.Resultado = clon
	return context.WithValue(ctx, claveContextoPersonalB2{}, contextoPersonalB2Sellado{fuente: f, valor: resuelta}), nil
}

// ContextoVinculadoPersonalB2 devuelve la misma evidencia F1 a la aplicación
// y al emisor V3. Revalida su ventana, sin producir otro registro de contexto.
func (f *FuenteF1) ContextoVinculadoPersonalB2(ctx context.Context) (ct.ContextoAutorizacionAltaV3, error) {
	var vacio ct.ContextoAutorizacionAltaV3
	if f == nil || f.reloj == nil || ctx == nil || ctx.Err() != nil {
		return vacio, ErrGobiernoInternoNoDisponible
	}
	sello, ok := ctx.Value(claveContextoPersonalB2{}).(contextoPersonalB2Sellado)
	if !ok || sello.fuente != f {
		return vacio, ErrGobiernoInternoNoDisponible
	}
	resuelta := sello.valor
	if resuelta.Resultado.Validar() != nil ||
		resuelta.Vinculo.ValidarPara(resuelta.Resultado) != nil ||
		!resuelta.Vinculo.VigenteEn(f.reloj.Ahora(), resuelta.Resultado) {
		return vacio, ErrGobiernoInternoNoDisponible
	}
	clon, err := resuelta.Resultado.Clonar()
	if err != nil {
		return vacio, ErrGobiernoInternoNoDisponible
	}
	resuelta.Resultado = clon
	return resuelta, nil
}

// AutoridadRutaSeguimiento limita la frontera HTTP a las rutas internas
// declaradas. Comprueba sesión, vínculo F1 y vínculo corporativo vivos antes
// del caso de uso; Personal consume el
// resultado ya sellado por el puente. Cada concesión concreta se evalúa en
// V3 y se consume en PostgreSQL.
type AutoridadRutaSeguimiento struct{ fuente *FuenteF1 }

func NuevaAutoridadRutaSeguimiento(fuente *FuenteF1) (*AutoridadRutaSeguimiento, error) {
	if fuente == nil || fuente.identidad == nil || fuente.revalidador == nil || fuente.resolutor == nil ||
		fuente.corporativo == nil {
		return nil, ErrGobiernoInternoNoDisponible
	}
	return &AutoridadRutaSeguimiento{fuente: fuente}, nil
}

func (a *AutoridadRutaSeguimiento) AutorizarRutaExacta(ctx context.Context, ruta string) error {
	if a == nil || a.fuente == nil || ctx == nil || ctx.Err() != nil || !RutaInternaGobernada(ruta) {
		return httpapi.ErrAccesoRutaExactaDenegado
	}
	if ruta != httpct.RutaConsultaSeguimientoV2 {
		// La prueba F1 ya está sellada por el puente tras mTLS. Validarla aquí
		// evita otra resolución y deniega invocaciones directas del router.
		if _, err := a.fuente.ContextoVinculadoPersonalB2(ctx); err != nil {
			return httpapi.ErrAccesoRutaExactaDenegado
		}
		return nil
	}
	if _, err := a.fuente.ResolverContexto(ctx); err != nil {
		return httpapi.ErrAccesoRutaExactaDenegado
	}
	return nil
}

// ResolverContextoPersonalB2 asocia la organización nominal del servidor al
// único contexto F1 creado para la lectura HTTP. Ningún selector del cliente
// interviene en esta asociación.
func (f *FuenteF1) ResolverContextoPersonalB2(ctx context.Context) (vecdomain.ContextoActor, string, error) {
	var vacio vecdomain.ContextoActor
	if f == nil || ctx == nil || ctx.Err() != nil {
		return vacio, "", ErrGobiernoInternoNoDisponible
	}
	resuelta, err := f.ContextoVinculadoPersonalB2(ctx)
	if err != nil {
		return vacio, "", ErrGobiernoInternoNoDisponible
	}
	actor, err := resuelta.Resultado.Contexto.Clonar()
	if err != nil {
		return vacio, "", ErrGobiernoInternoNoDisponible
	}
	nominal, existe := f.porCuenta[actor.Instantanea.CuentaRef]
	if !existe || nominal.PerfilActivoRef != actor.PerfilActivoRef {
		return vacio, "", ErrGobiernoInternoNoDisponible
	}
	return actor, nominal.OrganizacionRef, nil
}

// RutaInternaGobernada es la lista positiva del canal interno. La ficha
// admite una sola referencia de empleado; la decisión sobre ese empleado se
// realiza después en V3, nunca a partir del identificador de la URL.
func RutaInternaGobernada(ruta string) bool {
	if ruta == httpct.RutaConsultaSeguimientoV2 || ruta == httpapi.RutaVacantesEmpleadoB2 ||
		ruta == httpapi.RutaEmpleadosOrganismoB2 ||
		ruta == "/api/vec/personal/empleados" || ruta == "/api/vec/personal/hechos" ||
		ruta == httpapi.RutaCatalogosRegistroEmpleadoB2 {
		return true
	}
	if !strings.HasPrefix(ruta, httpapi.PrefijoFichaEmpleadoB2) {
		return false
	}
	referencia := strings.TrimPrefix(ruta, httpapi.PrefijoFichaEmpleadoB2)
	return !strings.Contains(referencia, "/") && personaldomain.ReferenciaEmpleadoValida(referencia)
}

var _ httpapi.AutoridadRutasExactas = (*AutoridadRutaSeguimiento)(nil)

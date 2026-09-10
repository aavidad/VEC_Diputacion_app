package incorporacionejercicio

import (
	"bytes"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	puente "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/personalincorporacion"
	pgct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	lector "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
	pgpersonal "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// ConfiguracionServidorV2PostgreSQL recibe autoridades ya gobernadas, nunca
// decisiones, identidad HTTP o permisos derivados de una bandera de UI. El
// propietario del runtime abre/verifica/cierra los pools y monta las fuentes
// selladas. Esta fábrica no instala SQL ni publica gobierno al arrancar.
type ConfiguracionServidorV2PostgreSQL struct {
	FuenteAutoridad FuentePeticionAutoridad
	Revalidador     vp.RevalidadorAutenticacionActorV1
	Resolutor       core.ResolutorContextoActorRegistradoV2
	Cadena          *CadenaAutorizacionAplicacion
	Correlador      GeneradorCorrelacionAutoridad
	Preparacion     ConfiguracionPreparacionDurableV2PostgreSQL
	AltaPersonal    *pgxpool.Pool
	RegistroCT      *pgxpool.Pool
}

// ServidorV2PostgreSQL sólo retiene configuración inmutable. No retiene
// peticiones, autorizaciones, órdenes, recibos ni claves de idempotencia.
type ServidorV2PostgreSQL struct {
	c ConfiguracionServidorV2PostgreSQL
}

func NuevoServidorV2PostgreSQL(c ConfiguracionServidorV2PostgreSQL) (*ServidorV2PostgreSQL, error) {
	f := ct.ErrComposicionIncorporacionAplicacion
	for _, d := range []any{c.FuenteAutoridad, c.Revalidador, c.Resolutor, c.Correlador} {
		if nulo(d) {
			return nil, f
		}
	}
	// No admitir una autoridad reutilizada entre peticiones.
	if c.Cadena == nil || !c.Cadena.valida() || c.Preparacion.Autoridad != nil {
		return nil, f
	}
	x := c.Preparacion
	if validarConfiguracionPreparacionV2PostgreSQL(x) != nil {
		return nil, f
	}
	if _, err := NuevaFuentePlanesPreparacionV2(x.Planes, x.TernaPlanes); err != nil {
		return nil, f
	}
	ps := x.Pools
	vistos := map[*pgxpool.Pool]bool{}
	for _, p := range []*pgxpool.Pool{c.AltaPersonal, c.RegistroCT, ps.InicialCT, ps.LocalizadorCT, ps.LocalizadorPersonal, ps.LecturaPersonal,
		ps.Historia.RegistroCT, ps.Historia.Autenticacion, ps.Historia.Contexto, ps.Historia.Evaluacion, ps.Historia.Concesion} {
		if p == nil || vistos[p] {
			return nil, f
		}
		vistos[p] = true
	}
	c.Preparacion.Planes = bytes.Clone(c.Preparacion.Planes)
	c.Preparacion.FuentePersonal = bytes.Clone(c.Preparacion.FuentePersonal)
	return &ServidorV2PostgreSQL{c: c}, nil
}

// PeticionV2PostgreSQL pertenece a una sola petición autenticada. No se exporta
// un constructor que permita introducir AutoridadAplicacion desde el canal.
type PeticionV2PostgreSQL struct {
	servidor   *ServidorV2PostgreSQL
	autoridad  *AutoridadAplicacion
	preparador *PreparadorDurableV2
}

func (s *ServidorV2PostgreSQL) NuevaPeticion(ctx context.Context) (*PeticionV2PostgreSQL, error) {
	if s == nil || ctx == nil {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c := s.c
	a, err := NuevaAutoridadAplicacion(ctx, c.FuenteAutoridad, c.Revalidador, c.Resolutor, c.Cadena, c.Correlador, c.Preparacion.Reloj)
	if err != nil {
		return nil, errorAutoridadPreparacion(ctx, err)
	}
	x := c.Preparacion
	x.Autoridad = a
	p, err := NuevoPreparadorDurableV2PostgreSQL(x)
	if err != nil {
		return nil, fallo(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &PeticionV2PostgreSQL{s, a, p}, nil
}

func (p *PeticionV2PostgreSQL) Consultar(ctx context.Context, exp string) (ct.ProyeccionIncorporacionAplicacionV2, error) {
	if p == nil || p.preparador == nil {
		return ct.ProyeccionIncorporacionAplicacionV2{}, ct.ErrComposicionIncorporacionAplicacion
	}
	// GET no construye un consumidor de alta ni depende de la fuente actual
	// para restaurar un original. El único proveedor durable valida la salida.
	return p.preparador.Consultar(ctx, exp)
}

func (p *PeticionV2PostgreSQL) Confirmar(ctx context.Context, i ct.IntencionIncorporacionAplicacionV2) (ct.ReciboIncorporacionAplicacionV2, error) {
	var cero ct.ReciboIncorporacionAplicacionV2
	if p == nil || p.servidor == nil || p.autoridad == nil || p.preparador == nil || ctx == nil {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if i.Validar() != nil {
		return cero, ct.ErrIntencionIncorporacionAplicacion
	}
	s, err := p.servicioConfirmacion()
	if err != nil {
		return cero, fallo(ctx, err)
	}
	return s.Confirmar(ctx, i)
}

func (p *PeticionV2PostgreSQL) servicioConfirmacion() (*Servicio, error) {
	c := p.servidor.c
	reloj := c.Preparacion.Reloj
	// Se reutilizan las instancias lectoras propietarias de esta petición; no
	// se repiten localizadores/restauradores ni se reconstruye su orden.
	l := p.preparador.c.LectorPersonal.(*lector.ConsumidorV2)
	r := p.preparador.c.Inicial.(*pgct.ResolverRaizIncorporacionV2PostgreSQL)
	txAlta, err := pgpersonal.NuevaTransaccionAltaEjercicioPostgreSQL(c.AltaPersonal, reloj)
	if err != nil {
		return nil, err
	}
	b, err := puente.NuevoV2(l, reloj)
	if err != nil {
		return nil, err
	}
	txCT, err := pgct.NuevaTransaccionRegistroIncorporacionV2PostgreSQL(c.RegistroCT, p.autoridad, r, p.preparador.c.Restaurador, reloj)
	if err != nil {
		return nil, err
	}
	confirmador, err := appct.NuevoServicioConfirmacionIncorporacionV2(p.autoridad, b, txCT, reloj)
	if err != nil {
		return nil, err
	}
	return Nuevo(Configuracion{Preparador: p.preparador, FuentePersonal: c.Preparacion.FuentePersonal, TernaPersonal: c.Preparacion.TernaPersonal,
		ProveedorAlta: p.autoridad, TransaccionAlta: txAlta, LectorPersonal: l, Confirmador: confirmador, Reloj: reloj})
}

var _ ct.ServicioIncorporacionAplicacionV2 = (*PeticionV2PostgreSQL)(nil)

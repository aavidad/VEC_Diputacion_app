package incorporacionejercicio

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func configuracionServidorV2Prueba(t *testing.T, c *casoPreparacionV2) ConfiguracionServidorV2PostgreSQL {
	t.Helper()
	x := configuracionPGPreparacionPrueba(t, c.plan)
	x.Autoridad = nil
	x.Reloj = c.a.reloj
	x.FuentePersonal, x.TernaPersonal = c.app.s.c.FuentePersonal, c.plan.FuentePersonal
	f := c.a.fuente
	f.p.PreparacionCT = c.a.a.peticion.PreparacionCT
	return ConfiguracionServidorV2PostgreSQL{FuenteAutoridad: f, Revalidador: c.a.reval, Resolutor: autoridadContextoDoble{c.a.a.contexto.Resultado},
		Cadena: c.a.a.cadena, Correlador: c.a.gen, Preparacion: x, AltaPersonal: new(pgxpool.Pool), RegistroCT: new(pgxpool.Pool)}
}

func TestIncorporacionV2EnsamblajePeticionYServicioReal(t *testing.T) {
	c := nuevoCasoPreparacionV2(t)
	cfg := configuracionServidorV2Prueba(t, c)
	s, err := NuevoServidorV2PostgreSQL(cfg)
	registroV2Exigir(t, err)
	clear(cfg.Preparacion.Planes) // El servidor ya posee copia del plan.
	p, err := s.NuevaPeticion(context.Background())
	registroV2Exigir(t, err)
	otra, err := s.NuevaPeticion(context.Background())
	registroV2Exigir(t, err)
	if p == otra || p.autoridad == otra.autoridad || p.preparador == otra.preparador {
		t.Fatal("estado nominal compartido entre peticiones")
	}
	servicio, err := p.servicioConfirmacion()
	registroV2Exigir(t, err)
	if servicio.c.Preparador != p.preparador || servicio.c.ProveedorAlta != p.autoridad || servicio.c.LectorPersonal != p.preparador.c.LectorPersonal {
		t.Fatal("no reutiliza las piezas nominales de la petición")
	}
	if c.a.store.registros != 0 || len(c.pasos) != 0 {
		t.Fatal("construir el servidor emitió permisos o consultó negocio")
	}
	// Pools inertes sólo para construir. Esta prueba NO instala ni ejecuta PG.
}

func TestIncorporacionV2EnsamblajeConsultaSinFuenteActual(t *testing.T) {
	c := nuevoCasoPreparacionV2(t)
	cfg := configuracionServidorV2Prueba(t, c)
	cfg.Preparacion.FuentePersonal = nil
	s, err := NuevoServidorV2PostgreSQL(cfg)
	registroV2Exigir(t, err)
	p, err := s.NuevaPeticion(context.Background())
	registroV2Exigir(t, err)
	// Sólo IO sustituido por dobles propietarios del preparador ya existente.
	// Conserva plan real sellado y la autoridad nueva de esta petición.
	p.preparador.c.Detalle = c.p.c.Detalle
	p.preparador.c.LocalizadorCT = c.p.c.LocalizadorCT
	p.preparador.c.Restaurador = c.p.c.Restaurador
	original, err := c.app.s.Confirmar(context.Background(), c.app.i)
	registroV2Exigir(t, err)
	c.confirmada = true
	r, err := p.Consultar(context.Background(), c.plan.SolicitudPersonal.ExpedienteRef)
	registroV2Exigir(t, err)
	if r.Recibo == nil || r.Preparacion != nil || !reflect.DeepEqual(*r.Recibo, original) {
		t.Fatal("no restaura original sin fuente actual")
	}
	if _, err := p.servicioConfirmacion(); err == nil {
		t.Fatal("admite alta sin fuente sellada")
	}
	if !reflect.DeepEqual(c.pasos, []string{"detalle", "ct81", "restaurar"}) {
		t.Fatalf("consultas inesperadas: %v", c.pasos)
	}
}

func TestIncorporacionV2EnsamblajeConfiguracionCerrada(t *testing.T) {
	c := nuevoCasoPreparacionV2(t)
	for _, nombre := range []string{"cadena", "identidad", "pool", "pool_compartido", "autoridad_compartida", "plan"} {
		t.Run(nombre, func(t *testing.T) {
			x := configuracionServidorV2Prueba(t, c)
			switch nombre {
			case "cadena":
				x.Cadena = nil
			case "identidad":
				x.Revalidador = nil
			case "pool":
				x.AltaPersonal = nil
			case "pool_compartido":
				x.RegistroCT = x.Preparacion.Pools.LocalizadorCT
			case "autoridad_compartida":
				x.Preparacion.Autoridad = c.a.a
			case "plan":
				x.Preparacion.Planes = []byte("{}")
			}
			s, err := NuevoServidorV2PostgreSQL(x)
			if s != nil || !errors.Is(err, ct.ErrComposicionIncorporacionAplicacion) {
				t.Fatal("configuración inválida aceptada")
			}
		})
	}
	x := configuracionServidorV2Prueba(t, c)
	x.FuenteAutoridad = autoridadFuenteDoble{fallo: ct.ErrDenegadaIncorporacionAplicacion}
	s, err := NuevoServidorV2PostgreSQL(x)
	registroV2Exigir(t, err)
	if p, err := s.NuevaPeticion(context.Background()); p != nil || !errors.Is(err, ct.ErrDenegadaIncorporacionAplicacion) {
		t.Fatal("autoridad denegada aceptada")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if p, err := s.NuevaPeticion(ctx); p != nil || !errors.Is(err, context.Canceled) {
		t.Fatal("cancelación perdida")
	}
}

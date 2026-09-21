package bootstrap

import (
	"context"
	"testing"
	"time"

	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type fuenteAutorizacionComunPrueba struct{}

func (fuenteAutorizacionComunPrueba) ObtenerInstantaneaAutorizacion(context.Context, string, string) (dominiovec.InstantaneaAutorizacion, error) {
	return dominiovec.InstantaneaAutorizacion{}, nil
}

type concesionesAutorizacionComunPrueba struct{}

func (concesionesAutorizacionComunPrueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Context, puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	return time.Now(), nil
}

type denegacionesAutorizacionComunPrueba struct{}

func (denegacionesAutorizacionComunPrueba) RegistrarDenegacionAutorizacionLigadaV3(context.Context, puertosvec.OrdenRegistroDenegacionAutorizacionLigadaV3) error {
	return nil
}

type motivosAutorizacionComunPrueba struct{}

func (motivosAutorizacionComunPrueba) ValidarReferenciaMotivoAutorizacionV2(context.Context, dominiovec.ReferenciaEntradaCatalogo, time.Time) error {
	return nil
}
func politicaComunPrueba(t *testing.T) politicaAutorizacionSolicitudLigadaV3Desarrollo {
	t.Helper()
	p, e := nuevaPoliticaAutorizacionSolicitudLigadaV3Desarrollo(fuenteAutorizacionComunPrueba{}, concesionesAutorizacionComunPrueba{}, denegacionesAutorizacionComunPrueba{}, motivosAutorizacionComunPrueba{})
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func TestCatalogoAutorizacionComunLigaAccionFronteraYPolitica(t *testing.T) {
	f := fronteraComunPrueba("cronos-crear", "POST", "/api/vec/cronos/partes", false)
	cs, e := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{f})
	if e != nil {
		t.Fatal(e)
	}
	p := politicaComunPrueba(t)
	c, e := nuevoCatalogoAutorizacionComunDesarrollo(cs, []descriptorAutorizacionComunDesarrollo{{Accion: "cronos.crear", ClavePolitica: "politica_modulo", ClaveCapacidad: "capacidad_modulo", Fronteras: []string{"cronos-crear"}, Politica: p}})
	if e != nil {
		t.Fatal(e)
	}
	if _, ok := c.politicaPara("cronos.crear", "cronos-crear", "politica_modulo", "capacidad_modulo"); !ok {
		t.Fatal("acción nominal no ligada")
	}
	if _, ok := c.politicaPara("cronos.crear", "cronos-crear", "otra", "capacidad_modulo"); ok {
		t.Fatal("clave política cruzada admitida")
	}
	if _, ok := c.politicaPara("otra", "cronos-crear", "politica_modulo", "capacidad_modulo"); ok {
		t.Fatal("acción cruzada admitida")
	}
	if _, ok := c.politicaPara("cronos.crear", "cronos-crear", "politica_modulo", "otra-capacidad"); ok {
		t.Fatal("capacidad cruzada admitida")
	}
	for _, d := range []descriptorAutorizacionComunDesarrollo{{Accion: "sin-politica", Fronteras: []string{"cronos-crear"}, Politica: p}, {Accion: "cruzada", ClavePolitica: "otra", ClaveCapacidad: "capacidad_modulo", Fronteras: []string{"cronos-crear"}, Politica: p}, {Accion: "capacidad-cruzada", ClavePolitica: "politica_modulo", ClaveCapacidad: "otra", Fronteras: []string{"cronos-crear"}, Politica: p}} {
		if _, e := nuevoCatalogoAutorizacionComunDesarrollo(cs, []descriptorAutorizacionComunDesarrollo{d}); e == nil {
			t.Fatal("descriptor autorización inválido admitido")
		}
	}
}

func TestCatalogoAutorizacionComunRechazaCatalogoFalsoConMismasClaves(t *testing.T) {
	d := fronteraComunPrueba("cronos-crear", "POST", "/api/vec/cronos/partes", false)
	canonico, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{d})
	if err != nil {
		t.Fatal(err)
	}
	falso, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{d})
	if err != nil {
		t.Fatal(err)
	}
	c, err := nuevoCatalogoAutorizacionComunDesarrollo(canonico, []descriptorAutorizacionComunDesarrollo{{Accion: "cronos.crear", ClavePolitica: "politica_modulo", ClaveCapacidad: "capacidad_modulo", Fronteras: []string{"cronos-crear"}, Politica: politicaComunPrueba(t)}})
	if err != nil {
		t.Fatal(err)
	}
	if !c.aceptaCatalogoFronteras(canonico) || c.aceptaCatalogoFronteras(falso) {
		t.Fatal("catálogo de fronteras sustituto fue aceptado")
	}
}

// Los cuatro puertos no son una vía de recuperación de una frontera inválida:
// el autorizador debe cortar antes de pedir instantánea, motivo o registro.
type puertosAutorizacionComunContadoresPrueba struct {
	instantaneas, concesiones, denegaciones, motivos int
}

func (p *puertosAutorizacionComunContadoresPrueba) ObtenerInstantaneaAutorizacion(context.Context, string, string) (dominiovec.InstantaneaAutorizacion, error) {
	p.instantaneas++
	return dominiovec.InstantaneaAutorizacion{}, nil
}

func (p *puertosAutorizacionComunContadoresPrueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Context, puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	p.concesiones++
	return time.Time{}, nil
}

func (p *puertosAutorizacionComunContadoresPrueba) RegistrarDenegacionAutorizacionLigadaV3(context.Context, puertosvec.OrdenRegistroDenegacionAutorizacionLigadaV3) error {
	p.denegaciones++
	return nil
}

func (p *puertosAutorizacionComunContadoresPrueba) ValidarReferenciaMotivoAutorizacionV2(context.Context, dominiovec.ReferenciaEntradaCatalogo, time.Time) error {
	p.motivos++
	return nil
}

func TestAutorizadorComunDeniegaFronteraAjenaOPerfilCruzadoAntesDeLosPuertos(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	perfil := soporte.contexto.Resultado.Contexto.PerfilActivoRef
	solicitud := solicitudAutorizacionComunValidaPrueba(t, soporte)

	construir := func(perfilDeclarado string) (*autorizadorComunDesarrollo, catalogoFronterasComunDesarrollo, *puertosAutorizacionComunContadoresPrueba) {
		t.Helper()
		frontera := fronteraComunPrueba("modulo-accion", "POST", "/api/vec/modulo/accion", false)
		frontera.PerfilesActivosRef = []string{perfilDeclarado}
		fronteras, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{frontera})
		if err != nil {
			t.Fatal(err)
		}
		puertos := &puertosAutorizacionComunContadoresPrueba{}
		politica, err := nuevaPoliticaAutorizacionSolicitudLigadaV3Desarrollo(puertos, puertos, puertos, puertos)
		if err != nil {
			t.Fatal(err)
		}
		catalogo, err := nuevoCatalogoAutorizacionComunDesarrollo(fronteras, []descriptorAutorizacionComunDesarrollo{{
			Accion: "modulo.accion", ClavePolitica: "politica_modulo", ClaveCapacidad: "capacidad_modulo",
			Fronteras: []string{"modulo-accion"}, Politica: politica,
		}})
		if err != nil {
			t.Fatal(err)
		}
		autorizador, err := nuevoAutorizadorComunDesarrollo(catalogo, soporte.reloj, seguridadvec.GeneradorReferenciasCriptograficas{}, aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: time.Minute})
		if err != nil {
			t.Fatal(err)
		}
		return autorizador, fronteras, puertos
	}
	contextoFrontera := func(fronteras catalogoFronterasComunDesarrollo) context.Context {
		d, ok := fronteras.resolver("POST", "/api/vec/modulo/accion")
		if !ok {
			t.Fatal("frontera de prueba ausente")
		}
		return context.WithValue(context.Background(), claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
			metodo: "POST", ruta: "/api/vec/modulo/accion", superficie: superficieInternaSeguridadComunDesarrollo,
			catalogo: fronteras, descriptor: d,
		})
	}
	for _, caso := range []struct {
		nombre string
		usar   func(*autorizadorComunDesarrollo, context.Context) error
		crear  func() (*autorizadorComunDesarrollo, context.Context, *puertosAutorizacionComunContadoresPrueba)
	}{
		{
			nombre: "catalogo-ajeno",
			usar: func(a *autorizadorComunDesarrollo, ctx context.Context) error {
				_, _, err := a.ExigirSolicitudLigadaV3(ctx, solicitud, soporte.contexto.Resultado)
				return err
			},
			crear: func() (*autorizadorComunDesarrollo, context.Context, *puertosAutorizacionComunContadoresPrueba) {
				a, _, puertos := construir(perfil)
				_, ajeno, _ := construir(perfil)
				return a, contextoFrontera(ajeno), puertos
			},
		},
		{
			nombre: "perfil-cruzado",
			usar: func(a *autorizadorComunDesarrollo, ctx context.Context) error {
				_, _, err := a.PrepararRegistroCompuestoSolicitudLigadaV3(ctx, solicitud, soporte.contexto.Resultado, seguridadvec.GeneradorReferenciasCriptograficas{})
				return err
			},
			crear: func() (*autorizadorComunDesarrollo, context.Context, *puertosAutorizacionComunContadoresPrueba) {
				a, fronteras, puertos := construir("prf_perfil_cruzado_0123456789abcdef")
				return a, contextoFrontera(fronteras), puertos
			},
		},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			autorizador, ctx, puertos := caso.crear()
			if err := caso.usar(autorizador, ctx); err == nil {
				t.Fatal("frontera cruzada concedida")
			}
			if puertos.instantaneas != 0 || puertos.concesiones != 0 || puertos.denegaciones != 0 || puertos.motivos != 0 {
				t.Fatalf("una frontera denegada alcanzó puertos: %+v", puertos)
			}
		})
	}
}

func solicitudAutorizacionComunValidaPrueba(t *testing.T, soporte *soporteAltaContratacionTemporalDesarrollo) dominiovec.SolicitudAutorizacionLigadaV3 {
	t.Helper()
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: soporte.contexto.Vinculo,
		ReferenciaMotivo:          soporte.motivo,
		Accion:                    "modulo.accion",
		Finalidad:                 "gestionar_modulo",
		Correlacion:               correlacion,
		Recurso: dominiovec.RecursoAutorizable{
			Referencia: "recurso:modulo:0123456789abcdef",
			ModuloID:   "modulo",
			Tipo:       "accion",
			Ambitos:    map[string]string{"organizacion_ref": "organizacion:prueba"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return solicitud
}

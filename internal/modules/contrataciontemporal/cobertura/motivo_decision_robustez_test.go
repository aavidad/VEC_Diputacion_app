package cobertura

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func TestResolutorMotivoCoberturaRechazaNulosTipadosYContexto(
	t *testing.T,
) {
	catalogo := catalogoMotivoCoberturaPublicado(t, nil)
	var consultaNula *consultaMotivoCoberturaPrueba
	if resolutor, err := NuevoResolutorMotivoDecisionCobertura(
		consultaNula,
		catalogo.ID,
		catalogo.ModuloID,
	); resolutor != nil ||
		!errors.Is(err, ErrConfiguracionResolutorMotivoDecisionCobertura) {
		t.Fatalf("consulta nula tipada: %#v, %v", resolutor, err)
	}
	resolutor := resolutorMotivoCoberturaPrueba(t, catalogo)
	motivo := motivoCoberturaPrueba(t, catalogo)
	if _, err := resolutor.Resolver(
		nil,
		motivo,
		instanteMotivoCoberturaPrueba,
	); !errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) {
		t.Fatalf("contexto nulo: %v", err)
	}
	var contextoPuntero *contextoPunteroPanicoMotivoCobertura
	var contextoMapa contextoMapaPanicoMotivoCobertura
	for nombre, contextoNulo := range map[string]context.Context{
		"puntero nulo tipado": contextoPuntero,
		"mapa nulo tipado":    contextoMapa,
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := resolutor.Resolver(
				contextoNulo,
				motivo,
				instanteMotivoCoberturaPrueba,
			); !errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) {
				t.Fatalf("contexto nulo tipado: %v", err)
			}
		})
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := resolutor.Resolver(
		ctx,
		motivo,
		instanteMotivoCoberturaPrueba,
	); !errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) ||
		!errors.Is(err, context.Canceled) {
		t.Fatalf("contexto cancelado: %v", err)
	}
	var resolutorNulo *ResolutorMotivoDecisionCobertura
	if _, err := resolutorNulo.Resolver(
		context.Background(),
		motivo,
		instanteMotivoCoberturaPrueba,
	); !errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) {
		t.Fatalf("resolutor nulo: %v", err)
	}
}

type contextoPunteroPanicoMotivoCobertura struct{}

func (*contextoPunteroPanicoMotivoCobertura) Deadline() (time.Time, bool) {
	panic("Deadline no debe invocarse")
}

func (*contextoPunteroPanicoMotivoCobertura) Done() <-chan struct{} {
	panic("Done no debe invocarse")
}

func (*contextoPunteroPanicoMotivoCobertura) Err() error {
	panic("Err no debe invocarse")
}

func (*contextoPunteroPanicoMotivoCobertura) Value(any) any {
	panic("Value no debe invocarse")
}

type contextoMapaPanicoMotivoCobertura map[string]any

func (contextoMapaPanicoMotivoCobertura) Deadline() (time.Time, bool) {
	panic("Deadline no debe invocarse")
}

func (contextoMapaPanicoMotivoCobertura) Done() <-chan struct{} {
	panic("Done no debe invocarse")
}

func (contextoMapaPanicoMotivoCobertura) Err() error {
	panic("Err no debe invocarse")
}

func (contextoMapaPanicoMotivoCobertura) Value(any) any {
	panic("Value no debe invocarse")
}

func TestResolutorMotivoCoberturaRechazaInstanteNoCanonico(t *testing.T) {
	catalogo := catalogoMotivoCoberturaPublicado(t, nil)
	resolutor := resolutorMotivoCoberturaPrueba(t, catalogo)
	motivo := motivoCoberturaPrueba(t, catalogo)
	casos := map[string]time.Time{
		"cero":         {},
		"zona local":   instanteMotivoCoberturaPrueba.In(time.FixedZone("X", 3600)),
		"submicro":     instanteMotivoCoberturaPrueba.Add(time.Nanosecond),
		"año cero":     time.Date(0, 12, 31, 0, 0, 0, 0, time.UTC),
		"año diez mil": time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	for nombre, instante := range casos {
		t.Run(nombre, func(t *testing.T) {
			if _, err := resolutor.Resolver(
				context.Background(),
				motivo,
				instante,
			); !errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) {
				t.Fatalf("instante inválido aceptado: %v", err)
			}
		})
	}
}

func TestResolutorMotivoCoberturaNoDelegaCatalogoElegidoPorCliente(
	t *testing.T,
) {
	catalogo := catalogoMotivoCoberturaPublicado(t, nil)
	consulta := &consultaContadoraMotivoCobertura{
		consultaMotivoCoberturaPrueba: consultaMotivoCoberturaPrueba{
			catalogo: catalogo,
		},
	}
	resolutor, err := NuevoResolutorMotivoDecisionCobertura(
		consulta,
		catalogo.ID,
		catalogo.ModuloID,
	)
	if err != nil {
		t.Fatal(err)
	}
	motivo := motivoCoberturaPrueba(t, catalogo)
	motivo.ReferenciaCatalogo.CatalogoID = "catalogo_elegido_cliente"
	if _, err := resolutor.Resolver(
		context.Background(),
		motivo,
		instanteMotivoCoberturaPrueba,
	); !errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) {
		t.Fatalf("catálogo del cliente aceptado: %v", err)
	}
	if consulta.llamadas.Load() != 0 {
		t.Fatal("se consultó un catálogo no fijado por composición")
	}
}

func TestResolutorMotivoCoberturaAcotaCatalogoAntesDeClonar(t *testing.T) {
	catalogo := catalogoMotivoCoberturaPublicado(t, nil)
	catalogo.Entradas = make([]dominiovec.EntradaCatalogoConfigurable, 10_001)
	consulta := &consultaCrudaMotivoCobertura{catalogo: catalogo}
	resolutor, err := NuevoResolutorMotivoDecisionCobertura(
		consulta,
		catalogo.ID,
		catalogo.ModuloID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolutor.Resolver(
		context.Background(),
		domain.MotivoGobernadoDecisionCobertura{
			ReferenciaCatalogo: motivoCoberturaPrueba(
				t,
				catalogoMotivoCoberturaPublicado(t, nil),
			).ReferenciaCatalogo,
			ClaveI18n: "cobertura.motivo.rectificacion",
		},
		instanteMotivoCoberturaPrueba,
	); !errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) {
		t.Fatalf("catálogo sobredimensionado aceptado: %v", err)
	}
}

func TestResolverClaveMotivoCoberturaRechazaProveedorHostilSinFallback(
	t *testing.T,
) {
	v1 := historialPublicadoMotivoCobertura(t, 1)[0]
	casos := map[string]*consultaHistorialMotivoCobertura{
		"truncamiento declarado": {
			versiones: []dominiovec.CatalogoConfigurable{v1},
			truncado:  true,
		},
		"demasiadas versiones sin declarar": {
			versiones: make(
				[]dominiovec.CatalogoConfigurable,
				limitesConsultaMotivosDecisionCobertura().Versiones+1,
			),
		},
	}
	for nombre, consulta := range casos {
		t.Run(nombre, func(t *testing.T) {
			resolutor := nuevoResolutorClaveMotivoCobertura(t, consulta)
			if _, err := resolutor.ResolverClave(
				context.Background(),
				claveMotivoCoberturaPrueba,
				instanteMotivoCoberturaPrueba,
			); !errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) {
				t.Fatalf("respuesta hostil aceptada: %v", err)
			}
			if consulta.llamadasSinCota.Load() != 0 ||
				consulta.llamadasObtener.Load() != 0 ||
				consulta.lecturasSinCota.Load() != 0 {
				t.Fatalf(
					"fallback: lista=%d exacta=%d; exactas acotadas=%d",
					consulta.llamadasSinCota.Load(),
					consulta.lecturasSinCota.Load(),
					consulta.llamadasObtener.Load(),
				)
			}
		})
	}
}

func TestResolverClaveMotivoCoberturaRevalidaPresupuestoHostil(
	t *testing.T,
) {
	v1 := historialPublicadoMotivoCobertura(t, 1)[0]
	demasiadasEntradas := v1
	demasiadasEntradas.Entradas = make(
		[]dominiovec.EntradaCatalogoConfigurable,
		limitesConsultaMotivosDecisionCobertura().Entradas+1,
	)
	demasiadosAtributos := v1
	demasiadosAtributos.Entradas = append(
		[]dominiovec.EntradaCatalogoConfigurable(nil),
		v1.Entradas...,
	)
	demasiadosAtributos.Entradas[0].Atributos = make(
		map[string]string,
		limitesConsultaMotivosDecisionCobertura().Atributos+1,
	)
	for indice := 0; indice <= limitesConsultaMotivosDecisionCobertura().Atributos; indice++ {
		demasiadosAtributos.Entradas[0].Atributos["atributo_"+strconv.Itoa(indice)] = "valor"
	}
	demasiadosBytes := v1
	demasiadosBytes.Descripcion = strings.Repeat(
		"x",
		limitesConsultaMotivosDecisionCobertura().BytesAproximados+1,
	)
	casos := map[string]dominiovec.CatalogoConfigurable{
		"entradas":  demasiadasEntradas,
		"atributos": demasiadosAtributos,
		"bytes":     demasiadosBytes,
	}
	for nombre, catalogo := range casos {
		t.Run(nombre, func(t *testing.T) {
			consulta := &consultaHistorialMotivoCobertura{
				versiones: []dominiovec.CatalogoConfigurable{catalogo},
			}
			resolutor := nuevoResolutorClaveMotivoCobertura(t, consulta)
			if _, err := resolutor.ResolverClave(
				context.Background(),
				claveMotivoCoberturaPrueba,
				instanteMotivoCoberturaPrueba,
			); !errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) {
				t.Fatalf("presupuesto hostil aceptado: %v", err)
			}
			if consulta.llamadasObtener.Load() != 0 {
				t.Fatal("el catálogo hostil llegó a la lectura exacta")
			}
		})
	}
}

func TestResolverClaveMotivoCoberturaRechazaLecturaExactaHostil(
	t *testing.T,
) {
	v1 := historialPublicadoMotivoCobertura(t, 1)[0]
	sobredimensionada := v1
	sobredimensionada.Descripcion = strings.Repeat(
		"x",
		limitesConsultaMotivosDecisionCobertura().BytesAproximados+1,
	)
	for nombre, consulta := range map[string]*consultaHistorialMotivoCobertura{
		"truncamiento declarado": {
			versiones:      []dominiovec.CatalogoConfigurable{v1},
			exactaTruncada: true,
		},
		"presupuesto ocultado": {
			versiones: []dominiovec.CatalogoConfigurable{v1},
			exactas: map[int]dominiovec.CatalogoConfigurable{
				1: sobredimensionada,
			},
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			resolutor := nuevoResolutorClaveMotivoCobertura(t, consulta)
			if _, err := resolutor.ResolverClave(
				context.Background(),
				claveMotivoCoberturaPrueba,
				instanteMotivoCoberturaPrueba,
			); !errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) {
				t.Fatalf("lectura exacta hostil aceptada: %v", err)
			}
			if consulta.llamadasObtener.Load() != 1 ||
				consulta.lecturasSinCota.Load() != 0 {
				t.Fatalf(
					"lecturas exactas: acotada=%d ilimitada=%d",
					consulta.llamadasObtener.Load(),
					consulta.lecturasSinCota.Load(),
				)
			}
		})
	}
}

func TestResolutorMotivoCoberturaCopiaAntesDeConservar(t *testing.T) {
	catalogo := catalogoMotivoCoberturaPublicado(t, nil)
	consulta := &consultaMotivoCoberturaPrueba{catalogo: catalogo}
	resolutor, err := NuevoResolutorMotivoDecisionCobertura(
		consulta,
		catalogo.ID,
		catalogo.ModuloID,
	)
	if err != nil {
		t.Fatal(err)
	}
	motivo := motivoCoberturaPrueba(t, catalogo)
	resolucion, err := resolutor.Resolver(
		context.Background(),
		motivo,
		instanteMotivoCoberturaPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	consulta.catalogo.Entradas[0].Atributos[atributoClaveI18nMotivoDecisionCobertura] =
		"cobertura.motivo.mutado"
	motivo.ClaveI18n = "cobertura.motivo.mutado"
	conservado, err := resolucion.Motivo()
	if err != nil ||
		conservado.ClaveI18n != "cobertura.motivo.rectificacion" {
		t.Fatalf("resolución compartió estado mutable: %#v, %v", conservado, err)
	}
}

func TestResolutorMotivoCoberturaToleraCambioConcurrenteSinFalsoExito(
	t *testing.T,
) {
	valido := catalogoMotivoCoberturaPublicado(t, nil)
	invalido := catalogoMotivoCoberturaPublicado(
		t,
		map[string]string{
			atributoClaveI18nMotivoDecisionCobertura: "cobertura.motivo.distinto",
		},
	)
	consulta := &consultaConcurrenteMotivoCobertura{catalogo: valido}
	resolutor, err := NuevoResolutorMotivoDecisionCobertura(
		consulta,
		valido.ID,
		valido.ModuloID,
	)
	if err != nil {
		t.Fatal(err)
	}
	motivo := motivoCoberturaPrueba(t, valido)
	var inesperados atomic.Int32
	var fallos atomic.Int32
	var grupo sync.WaitGroup
	grupo.Add(9)
	go func() {
		defer grupo.Done()
		for indice := 0; indice < 500; indice++ {
			if indice%2 == 0 {
				consulta.cambiar(valido)
			} else {
				consulta.cambiar(invalido)
			}
		}
	}()
	for range 8 {
		go func() {
			defer grupo.Done()
			for range 100 {
				_, err := resolutor.Resolver(
					context.Background(),
					motivo,
					instanteMotivoCoberturaPrueba,
				)
				if err != nil &&
					!errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) {
					inesperados.Add(1)
				}
				if errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) {
					fallos.Add(1)
				}
			}
		}()
	}
	grupo.Wait()
	if inesperados.Load() != 0 {
		t.Fatalf("resultados no cerrados: %d", inesperados.Load())
	}
	if consulta.invalidosDevueltos.Load() == 0 ||
		fallos.Load() != consulta.invalidosDevueltos.Load() {
		t.Fatalf(
			"falso éxito concurrente: inválidos=%d fallos=%d",
			consulta.invalidosDevueltos.Load(),
			fallos.Load(),
		)
	}
}

type consultaContadoraMotivoCobertura struct {
	consultaMotivoCoberturaPrueba
	llamadas atomic.Int32
}

func (c *consultaContadoraMotivoCobertura) ObtenerCatalogo(
	ctx context.Context,
	id string,
	version int,
) (dominiovec.CatalogoConfigurable, error) {
	c.llamadas.Add(1)
	return c.consultaMotivoCoberturaPrueba.ObtenerCatalogo(ctx, id, version)
}

type consultaCrudaMotivoCobertura struct {
	catalogo dominiovec.CatalogoConfigurable
}

func (c *consultaCrudaMotivoCobertura) ObtenerCatalogo(
	context.Context,
	string,
	int,
) (dominiovec.CatalogoConfigurable, error) {
	return c.catalogo, nil
}

func (c *consultaCrudaMotivoCobertura) ObtenerCatalogoAcotado(
	context.Context,
	string,
	int,
	puertosvec.LimitesConsultaCatalogosAcotada,
) (puertosvec.ResultadoConsultaCatalogoAcotado, error) {
	return puertosvec.ResultadoConsultaCatalogoAcotado{
		Catalogo: c.catalogo,
	}, nil
}

func (c *consultaCrudaMotivoCobertura) ListarVersionesCatalogo(
	context.Context,
	string,
) ([]dominiovec.CatalogoConfigurable, error) {
	return nil, nil
}

func (c *consultaCrudaMotivoCobertura) ListarVersionesCatalogoAcotado(
	context.Context,
	string,
	puertosvec.LimitesConsultaCatalogosAcotada,
) (puertosvec.ResultadoConsultaCatalogosAcotada, error) {
	return puertosvec.ResultadoConsultaCatalogosAcotada{}, nil
}

type consultaConcurrenteMotivoCobertura struct {
	mu                 sync.RWMutex
	catalogo           dominiovec.CatalogoConfigurable
	invalidosDevueltos atomic.Int32
}

func (c *consultaConcurrenteMotivoCobertura) cambiar(
	catalogo dominiovec.CatalogoConfigurable,
) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.catalogo = catalogo
}

func (c *consultaConcurrenteMotivoCobertura) ObtenerCatalogo(
	_ context.Context,
	_ string,
	_ int,
) (dominiovec.CatalogoConfigurable, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	clon, err := c.catalogo.ClonarCanonico()
	if err == nil &&
		clon.Entradas[0].Atributos[atributoClaveI18nMotivoDecisionCobertura] !=
			"cobertura.motivo.rectificacion" {
		c.invalidosDevueltos.Add(1)
	}
	return clon, err
}

func (c *consultaConcurrenteMotivoCobertura) ObtenerCatalogoAcotado(
	ctx context.Context,
	id string,
	version int,
	_ puertosvec.LimitesConsultaCatalogosAcotada,
) (puertosvec.ResultadoConsultaCatalogoAcotado, error) {
	catalogo, err := c.ObtenerCatalogo(ctx, id, version)
	return puertosvec.ResultadoConsultaCatalogoAcotado{
		Catalogo: catalogo,
	}, err
}

func (c *consultaConcurrenteMotivoCobertura) ListarVersionesCatalogo(
	context.Context,
	string,
) ([]dominiovec.CatalogoConfigurable, error) {
	return nil, nil
}

func (c *consultaConcurrenteMotivoCobertura) ListarVersionesCatalogoAcotado(
	context.Context,
	string,
	puertosvec.LimitesConsultaCatalogosAcotada,
) (puertosvec.ResultadoConsultaCatalogosAcotada, error) {
	return puertosvec.ResultadoConsultaCatalogosAcotada{}, nil
}

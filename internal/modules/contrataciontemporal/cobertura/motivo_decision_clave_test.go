package cobertura

import (
	"context"
	"errors"
	"math/rand"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	catalogoMotivosCoberturaPrueba = "motivos_cobertura"
	moduloMotivosCoberturaPrueba   = "contratacion_temporal"
	claveMotivoCoberturaPrueba     = "rectificacion_decision"
	claveI18nMotivoCoberturaPrueba = "cobertura.motivo.rectificacion"
)

type consultaHistorialMotivoCobertura struct {
	versiones       []dominiovec.CatalogoConfigurable
	exactas         map[int]dominiovec.CatalogoConfigurable
	errListar       error
	errObtener      error
	alListar        func(context.Context)
	truncado        bool
	exactaTruncada  bool
	llamadasListar  atomic.Int32
	llamadasSinCota atomic.Int32
	llamadasObtener atomic.Int32
	lecturasSinCota atomic.Int32
}

func (c *consultaHistorialMotivoCobertura) ListarVersionesCatalogo(
	ctx context.Context,
	_ string,
) ([]dominiovec.CatalogoConfigurable, error) {
	c.llamadasSinCota.Add(1)
	if c.alListar != nil {
		c.alListar(ctx)
	}
	if c.errListar != nil {
		return nil, c.errListar
	}
	return c.versiones, nil
}

func (c *consultaHistorialMotivoCobertura) ListarVersionesCatalogoAcotado(
	ctx context.Context,
	_ string,
	_ puertosvec.LimitesConsultaCatalogosAcotada,
) (puertosvec.ResultadoConsultaCatalogosAcotada, error) {
	c.llamadasListar.Add(1)
	if c.alListar != nil {
		c.alListar(ctx)
	}
	if c.errListar != nil {
		return puertosvec.ResultadoConsultaCatalogosAcotada{}, c.errListar
	}
	return puertosvec.ResultadoConsultaCatalogosAcotada{
		Catalogos: c.versiones,
		Truncado:  c.truncado,
	}, nil
}

func (c *consultaHistorialMotivoCobertura) ObtenerCatalogo(
	_ context.Context,
	id string,
	version int,
) (dominiovec.CatalogoConfigurable, error) {
	c.lecturasSinCota.Add(1)
	if c.errObtener != nil {
		return dominiovec.CatalogoConfigurable{}, c.errObtener
	}
	if id != catalogoMotivosCoberturaPrueba {
		return dominiovec.CatalogoConfigurable{},
			puertosvec.ErrCatalogoNoEncontrado
	}
	if exacta, existe := c.exactas[version]; existe {
		return exacta, nil
	}
	for _, catalogo := range c.versiones {
		if catalogo.Version == version {
			return catalogo, nil
		}
	}
	return dominiovec.CatalogoConfigurable{},
		puertosvec.ErrCatalogoNoEncontrado
}

func (c *consultaHistorialMotivoCobertura) ObtenerCatalogoAcotado(
	_ context.Context,
	id string,
	version int,
	_ puertosvec.LimitesConsultaCatalogosAcotada,
) (puertosvec.ResultadoConsultaCatalogoAcotado, error) {
	c.llamadasObtener.Add(1)
	if c.errObtener != nil {
		return puertosvec.ResultadoConsultaCatalogoAcotado{}, c.errObtener
	}
	if id != catalogoMotivosCoberturaPrueba {
		return puertosvec.ResultadoConsultaCatalogoAcotado{},
			puertosvec.ErrCatalogoNoEncontrado
	}
	if exacta, existe := c.exactas[version]; existe {
		return puertosvec.ResultadoConsultaCatalogoAcotado{
			Catalogo: exacta,
			Truncado: c.exactaTruncada,
		}, nil
	}
	for _, catalogo := range c.versiones {
		if catalogo.Version == version {
			return puertosvec.ResultadoConsultaCatalogoAcotado{
				Catalogo: catalogo,
				Truncado: c.exactaTruncada,
			}, nil
		}
	}
	return puertosvec.ResultadoConsultaCatalogoAcotado{},
		puertosvec.ErrCatalogoNoEncontrado
}

func TestResolverClaveMotivoCoberturaSeleccionaVersionPublicadaMayor(
	t *testing.T,
) {
	versiones := historialPublicadoMotivoCobertura(t, 3)
	rand.New(rand.NewSource(83)).Shuffle(
		len(versiones),
		func(i, j int) { versiones[i], versiones[j] = versiones[j], versiones[i] },
	)
	ordenEntregado := []int{
		versiones[0].Version,
		versiones[1].Version,
		versiones[2].Version,
	}
	consulta := &consultaHistorialMotivoCobertura{versiones: versiones}
	resolutor := nuevoResolutorClaveMotivoCobertura(t, consulta)

	resolucion, err := resolutor.ResolverClave(
		context.Background(),
		domain.ClaveCatalogo(claveMotivoCoberturaPrueba),
		instanteMotivoCoberturaPrueba,
	)
	if err != nil {
		t.Fatalf("resolver historial desordenado: %v", err)
	}
	motivo, err := resolucion.Motivo()
	if err != nil {
		t.Fatal(err)
	}
	esperado := versionesPorNumero(versiones)[3]
	huella, err := esperado.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	if motivo.ReferenciaCatalogo.CatalogoID != esperado.ID ||
		motivo.ReferenciaCatalogo.CatalogoVersion != 3 ||
		motivo.ReferenciaCatalogo.CatalogoHuellaSHA256 != huella ||
		motivo.ReferenciaCatalogo.EntradaClave != claveMotivoCoberturaPrueba ||
		motivo.ClaveI18n != claveI18nMotivoCoberturaPrueba {
		t.Fatalf("motivo derivado inesperado: %#v", motivo)
	}
	if consulta.llamadasListar.Load() != 1 ||
		consulta.llamadasObtener.Load() != 1 {
		t.Fatalf(
			"lecturas inesperadas: lista=%d exacta=%d",
			consulta.llamadasListar.Load(),
			consulta.llamadasObtener.Load(),
		)
	}
	for indice, version := range ordenEntregado {
		if versiones[indice].Version != version {
			t.Fatal("el resolutor ordenó el slice propiedad del adaptador")
		}
	}
}

func TestResolverClaveMotivoCoberturaIgnoraBorradorPosterior(t *testing.T) {
	v1 := historialPublicadoMotivoCobertura(t, 1)[0]
	v2, err := v1.NuevaVersion(
		2,
		"actor_creador_motivo_v2",
		"fuente_motivo_v2",
		"Nueva versión en preparación.",
		instanteMotivoCoberturaPrueba.Add(-90*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	consulta := &consultaHistorialMotivoCobertura{
		versiones: []dominiovec.CatalogoConfigurable{v2, v1},
	}
	resolutor := nuevoResolutorClaveMotivoCobertura(t, consulta)
	resolucion, err := resolutor.ResolverClave(
		context.Background(),
		claveMotivoCoberturaPrueba,
		instanteMotivoCoberturaPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	motivo, err := resolucion.Motivo()
	if err != nil || motivo.ReferenciaCatalogo.CatalogoVersion != 1 {
		t.Fatalf("el borrador desplazó a v1: %#v, %v", motivo, err)
	}
}

func TestResolverClaveMotivoCoberturaEligeV2Publicada(t *testing.T) {
	versiones := historialPublicadoMotivoCobertura(t, 2)
	consulta := &consultaHistorialMotivoCobertura{versiones: versiones}
	resolutor := nuevoResolutorClaveMotivoCobertura(t, consulta)
	resolucion, err := resolutor.ResolverClave(
		context.Background(),
		claveMotivoCoberturaPrueba,
		instanteMotivoCoberturaPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	motivo, err := resolucion.Motivo()
	if err != nil || motivo.ReferenciaCatalogo.CatalogoVersion != 2 {
		t.Fatalf("v2 publicada no elegida: %#v, %v", motivo, err)
	}
}

func TestResolverClaveMotivoCoberturaNoRetrocedeTrasRetirada(t *testing.T) {
	versiones := historialPublicadoMotivoCobertura(t, 2)
	v2 := versiones[1]
	retirada, err := v2.Retirar(
		"actor_retirada_motivo_v2",
		"aprobacion_retirada_motivo_v2",
		"Retirada vigente.",
		instanteMotivoCoberturaPrueba.Add(-time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	consulta := &consultaHistorialMotivoCobertura{
		versiones: []dominiovec.CatalogoConfigurable{versiones[0], retirada},
	}
	resolutor := nuevoResolutorClaveMotivoCobertura(t, consulta)
	if _, err := resolutor.ResolverClave(
		context.Background(),
		claveMotivoCoberturaPrueba,
		instanteMotivoCoberturaPrueba,
	); !errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) {
		t.Fatalf("retroceso a v1 tras retirar v2: %v", err)
	}
	if consulta.llamadasObtener.Load() != 0 {
		t.Fatal("una retirada efectiva llegó a la relectura exacta")
	}
}

func TestResolverClaveMotivoCoberturaConservaConsultaHistoricaPreviaARetirada(
	t *testing.T,
) {
	versiones := historialPublicadoMotivoCobertura(t, 2)
	retirada, err := versiones[1].Retirar(
		"actor_retirada_motivo_v2",
		"aprobacion_retirada_motivo_v2",
		"Retirada posterior.",
		instanteMotivoCoberturaPrueba.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	consulta := &consultaHistorialMotivoCobertura{
		versiones: []dominiovec.CatalogoConfigurable{versiones[0], retirada},
	}
	resolutor := nuevoResolutorClaveMotivoCobertura(t, consulta)
	resolucion, err := resolutor.ResolverClave(
		context.Background(),
		claveMotivoCoberturaPrueba,
		instanteMotivoCoberturaPrueba,
	)
	if err != nil {
		t.Fatalf("retirada posterior invalidó el instante: %v", err)
	}
	motivo, err := resolucion.Motivo()
	if err != nil || motivo.ReferenciaCatalogo.CatalogoVersion != 2 {
		t.Fatalf("versión histórica inesperada: %#v, %v", motivo, err)
	}
}

func TestResolverClaveMotivoCoberturaRechazaHistorialAnomalo(t *testing.T) {
	versiones := historialPublicadoMotivoCobertura(t, 3)
	v2 := versiones[1]
	v2.ModuloID = "otro_modulo"
	incoherente := versiones[1]
	incoherente.CreadoEn = versiones[0].CreadoEn.Add(-time.Minute)
	casos := map[string][]dominiovec.CatalogoConfigurable{
		"duplicado":   {versiones[0], versiones[1], versiones[1]},
		"hueco":       {versiones[0], versiones[2]},
		"otro modulo": {versiones[0], v2},
		"cronologia":  {versiones[0], incoherente},
	}
	for nombre, historial := range casos {
		t.Run(nombre, func(t *testing.T) {
			consulta := &consultaHistorialMotivoCobertura{versiones: historial}
			resolutor := nuevoResolutorClaveMotivoCobertura(t, consulta)
			if _, err := resolutor.ResolverClave(
				context.Background(),
				claveMotivoCoberturaPrueba,
				instanteMotivoCoberturaPrueba,
			); !errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) {
				t.Fatalf("historial anómalo aceptado: %v", err)
			}
			if consulta.llamadasObtener.Load() != 0 {
				t.Fatal("un historial anómalo llegó a la relectura exacta")
			}
		})
	}
}

func TestResolverClaveMotivoCoberturaRechazaPublicacionSoloFutura(
	t *testing.T,
) {
	borrador := borradorVersionUnoMotivoCobertura(
		t,
		instanteMotivoCoberturaPrueba.Add(-time.Hour),
	)
	futura, err := borrador.Publicar(
		"actor_publicador_motivo_v1",
		"aprobacion_publicacion_motivo_v1",
		"Publicación futura.",
		instanteMotivoCoberturaPrueba.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	consulta := &consultaHistorialMotivoCobertura{
		versiones: []dominiovec.CatalogoConfigurable{futura},
	}
	resolutor := nuevoResolutorClaveMotivoCobertura(t, consulta)
	if _, err := resolutor.ResolverClave(
		context.Background(),
		claveMotivoCoberturaPrueba,
		instanteMotivoCoberturaPrueba,
	); !errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) {
		t.Fatalf("publicación futura aceptada: %v", err)
	}
}

func TestResolverClaveMotivoCoberturaPublicacionFuturaNoDesplazaHistorica(
	t *testing.T,
) {
	v1 := historialPublicadoMotivoCobertura(t, 1)[0]
	v2, err := v1.NuevaVersion(
		2,
		"actor_creador_motivo_v2",
		"fuente_motivo_v2",
		"Nueva versión futura.",
		instanteMotivoCoberturaPrueba.Add(-time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	v2, err = v2.Publicar(
		"actor_publicador_motivo_v2",
		"aprobacion_publicacion_motivo_v2",
		"Publicación futura.",
		instanteMotivoCoberturaPrueba.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	consulta := &consultaHistorialMotivoCobertura{
		versiones: []dominiovec.CatalogoConfigurable{v2, v1},
	}
	resolutor := nuevoResolutorClaveMotivoCobertura(t, consulta)
	resolucion, err := resolutor.ResolverClave(
		context.Background(),
		claveMotivoCoberturaPrueba,
		instanteMotivoCoberturaPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	motivo, err := resolucion.Motivo()
	if err != nil || motivo.ReferenciaCatalogo.CatalogoVersion != 1 {
		t.Fatalf("la publicación futura desplazó a v1: %#v, %v", motivo, err)
	}
}

func TestResolverClaveMotivoCoberturaRevalidaHuellaYClaveI18n(
	t *testing.T,
) {
	v1 := historialPublicadoMotivoCobertura(t, 1)[0]
	mutada := v1
	mutada.Entradas = append(
		[]dominiovec.EntradaCatalogoConfigurable(nil),
		v1.Entradas...,
	)
	mutada.Entradas[0].Atributos = map[string]string{
		atributoClaveI18nMotivoDecisionCobertura: "cobertura.motivo.mutado",
	}
	casos := map[string]*consultaHistorialMotivoCobertura{
		"huella mutada entre lecturas": {
			versiones: []dominiovec.CatalogoConfigurable{v1},
			exactas:   map[int]dominiovec.CatalogoConfigurable{1: mutada},
		},
		"atributo ausente": {
			versiones: []dominiovec.CatalogoConfigurable{
				catalogoSinAtributoI18nMotivoCobertura(v1),
			},
		},
		"atributo inválido": {
			versiones: []dominiovec.CatalogoConfigurable{
				catalogoConI18nMotivoCobertura(v1, "Clave no canónica"),
			},
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
				t.Fatalf("catálogo no confiable aceptado: %v", err)
			}
		})
	}
}

func TestResolverClaveMotivoCoberturaExigeEntradaVigente(t *testing.T) {
	v1 := historialPublicadoMotivoCobertura(t, 1)[0]
	caducada := v1
	caducada.Entradas = append(
		[]dominiovec.EntradaCatalogoConfigurable(nil),
		v1.Entradas...,
	)
	caducada.Entradas[0].VigenteHasta = instanteMotivoCoberturaPrueba
	casos := map[string]struct {
		catalogo dominiovec.CatalogoConfigurable
		clave    domain.ClaveCatalogo
	}{
		"caducada": {
			catalogo: caducada,
			clave:    claveMotivoCoberturaPrueba,
		},
		"ausente": {
			catalogo: v1,
			clave:    "motivo_no_publicado",
		},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			consulta := &consultaHistorialMotivoCobertura{
				versiones: []dominiovec.CatalogoConfigurable{caso.catalogo},
			}
			resolutor := nuevoResolutorClaveMotivoCobertura(t, consulta)
			if _, err := resolutor.ResolverClave(
				context.Background(),
				caso.clave,
				instanteMotivoCoberturaPrueba,
			); !errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) {
				t.Fatalf("entrada no vigente aceptada: %v", err)
			}
		})
	}
}

func TestResolverClaveMotivoCoberturaCancelaYAcota(t *testing.T) {
	v1 := historialPublicadoMotivoCobertura(t, 1)[0]
	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	consulta := &consultaHistorialMotivoCobertura{
		versiones: []dominiovec.CatalogoConfigurable{v1},
	}
	resolutor := nuevoResolutorClaveMotivoCobertura(t, consulta)
	if _, err := resolutor.ResolverClave(
		ctxCancelado,
		claveMotivoCoberturaPrueba,
		instanteMotivoCoberturaPrueba,
	); !errors.Is(err, context.Canceled) ||
		!errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) {
		t.Fatalf("cancelación previa no propagada de forma saneada: %v", err)
	}
	if consulta.llamadasListar.Load() != 0 {
		t.Fatal("se consultó el catálogo con contexto ya cancelado")
	}

	ctxDurante, cancelarDurante := context.WithCancel(context.Background())
	consultaDurante := &consultaHistorialMotivoCobertura{
		versiones: []dominiovec.CatalogoConfigurable{v1},
		alListar: func(context.Context) {
			cancelarDurante()
		},
	}
	resolutor = nuevoResolutorClaveMotivoCobertura(t, consultaDurante)
	if _, err := resolutor.ResolverClave(
		ctxDurante,
		claveMotivoCoberturaPrueba,
		instanteMotivoCoberturaPrueba,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación durante listado no propagada: %v", err)
	}

	demasiadas := make(
		[]dominiovec.CatalogoConfigurable,
		limitesConsultaMotivosDecisionCobertura().Versiones+1,
	)
	consultaAcotada := &consultaHistorialMotivoCobertura{versiones: demasiadas}
	resolutor = nuevoResolutorClaveMotivoCobertura(t, consultaAcotada)
	if _, err := resolutor.ResolverClave(
		context.Background(),
		claveMotivoCoberturaPrueba,
		instanteMotivoCoberturaPrueba,
	); !errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) {
		t.Fatalf("listado sobredimensionado aceptado: %v", err)
	}
}

func TestResolverClaveMotivoCoberturaSaneaFallosYNulos(t *testing.T) {
	v1 := historialPublicadoMotivoCobertura(t, 1)[0]
	consulta := &consultaHistorialMotivoCobertura{
		versiones: []dominiovec.CatalogoConfigurable{v1},
		errListar: errors.New("DNI 12345678Z: detalle privado"),
	}
	resolutor := nuevoResolutorClaveMotivoCobertura(t, consulta)
	if _, err := resolutor.ResolverClave(
		context.Background(),
		claveMotivoCoberturaPrueba,
		instanteMotivoCoberturaPrueba,
	); !errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) ||
		strings.Contains(err.Error(), "12345678Z") {
		t.Fatalf("fallo del proveedor expuesto: %v", err)
	}
	for nombre, ejecutar := range map[string]func() error{
		"contexto nulo": func() error {
			_, err := resolutor.ResolverClave(
				nil,
				claveMotivoCoberturaPrueba,
				instanteMotivoCoberturaPrueba,
			)
			return err
		},
		"clave inválida": func() error {
			_, err := resolutor.ResolverClave(
				context.Background(),
				" Clave elegida por cliente ",
				instanteMotivoCoberturaPrueba,
			)
			return err
		},
		"resolutor nulo": func() error {
			var nulo *ResolutorMotivoDecisionCobertura
			_, err := nulo.ResolverClave(
				context.Background(),
				claveMotivoCoberturaPrueba,
				instanteMotivoCoberturaPrueba,
			)
			return err
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			if err := ejecutar(); !errors.Is(
				err,
				ErrMotivoDecisionCoberturaNoConfiable,
			) {
				t.Fatalf("entrada inválida aceptada: %v", err)
			}
		})
	}
}

func TestNuevoResolutorClaveMotivoCoberturaFijaModulo(t *testing.T) {
	v1 := historialPublicadoMotivoCobertura(t, 1)[0]
	consulta := &consultaHistorialMotivoCobertura{
		versiones: []dominiovec.CatalogoConfigurable{v1},
	}
	for nombre, modulo := range map[string]string{
		"vacío":        "",
		"con espacios": " contratacion_temporal",
		"no canónico":  "Contratacion Temporal",
	} {
		t.Run(nombre, func(t *testing.T) {
			resolutor, err := NuevoResolutorMotivoDecisionCobertura(
				consulta,
				catalogoMotivosCoberturaPrueba,
				modulo,
			)
			if resolutor != nil ||
				!errors.Is(err, ErrConfiguracionResolutorMotivoDecisionCobertura) {
				t.Fatalf("módulo inválido aceptado: %#v, %v", resolutor, err)
			}
		})
	}
}

func nuevoResolutorClaveMotivoCobertura(
	t *testing.T,
	consulta puertosvec.ConsultaCatalogosConfigurablesAcotada,
) *ResolutorMotivoDecisionCobertura {
	t.Helper()
	resolutor, err := NuevoResolutorMotivoDecisionCobertura(
		consulta,
		catalogoMotivosCoberturaPrueba,
		moduloMotivosCoberturaPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	return resolutor
}

func historialPublicadoMotivoCobertura(
	t *testing.T,
	cantidad int,
) []dominiovec.CatalogoConfigurable {
	t.Helper()
	if cantidad < 1 {
		t.Fatal("el historial de prueba requiere alguna versión")
	}
	creadoEn := instanteMotivoCoberturaPrueba.Add(-12 * time.Hour)
	borrador := borradorVersionUnoMotivoCobertura(t, creadoEn)
	publicado, err := borrador.Publicar(
		"actor_publicador_motivo_v1",
		"aprobacion_publicacion_motivo_v1",
		"Publicación gobernada de v1.",
		creadoEn.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	versiones := []dominiovec.CatalogoConfigurable{publicado}
	for version := 2; version <= cantidad; version++ {
		instanteCreacion := creadoEn.Add(time.Duration(version*2-2) * time.Hour)
		borrador, err = publicado.NuevaVersion(
			version,
			"actor_creador_motivo_v"+numeroVersionMotivoCobertura(version),
			"fuente_motivo_v"+numeroVersionMotivoCobertura(version),
			"Nueva versión gobernada.",
			instanteCreacion,
		)
		if err != nil {
			t.Fatal(err)
		}
		publicado, err = borrador.Publicar(
			"actor_publicador_motivo_v"+numeroVersionMotivoCobertura(version),
			"aprobacion_publicacion_motivo_v"+numeroVersionMotivoCobertura(version),
			"Publicación gobernada.",
			instanteCreacion.Add(time.Hour),
		)
		if err != nil {
			t.Fatal(err)
		}
		versiones = append(versiones, publicado)
	}
	return versiones
}

func borradorVersionUnoMotivoCobertura(
	t *testing.T,
	creadoEn time.Time,
) dominiovec.CatalogoConfigurable {
	t.Helper()
	borrador := dominiovec.CatalogoConfigurable{
		ID:             catalogoMotivosCoberturaPrueba,
		Version:        1,
		Revision:       1,
		ModuloID:       moduloMotivosCoberturaPrueba,
		Nombre:         "Motivos de cobertura",
		FuenteRef:      "politica_motivos_cobertura",
		MotivoCreacion: "Primera versión gobernada.",
		Entradas: []dominiovec.EntradaCatalogoConfigurable{{
			Clave:        claveMotivoCoberturaPrueba,
			Etiqueta:     "Rectificación",
			Orden:        10,
			VigenteDesde: creadoEn,
			Atributos: map[string]string{
				atributoClaveI18nMotivoDecisionCobertura: claveI18nMotivoCoberturaPrueba,
			},
		}},
		Estado:    dominiovec.EstadoCatalogoBorrador,
		CreadoPor: "actor_creador_motivo_v1",
		CreadoEn:  creadoEn,
	}
	if err := borrador.Validar(); err != nil {
		t.Fatalf("borrador v1: %v", err)
	}
	return borrador
}

func catalogoSinAtributoI18nMotivoCobertura(
	catalogo dominiovec.CatalogoConfigurable,
) dominiovec.CatalogoConfigurable {
	return catalogoConI18nMotivoCobertura(catalogo, "")
}

func catalogoConI18nMotivoCobertura(
	catalogo dominiovec.CatalogoConfigurable,
	claveI18n string,
) dominiovec.CatalogoConfigurable {
	catalogo.Entradas = append(
		[]dominiovec.EntradaCatalogoConfigurable(nil),
		catalogo.Entradas...,
	)
	if claveI18n == "" {
		catalogo.Entradas[0].Atributos = nil
	} else {
		catalogo.Entradas[0].Atributos = map[string]string{
			atributoClaveI18nMotivoDecisionCobertura: claveI18n,
		}
	}
	return catalogo
}

func versionesPorNumero(
	versiones []dominiovec.CatalogoConfigurable,
) map[int]dominiovec.CatalogoConfigurable {
	resultado := make(map[int]dominiovec.CatalogoConfigurable, len(versiones))
	for _, catalogo := range versiones {
		resultado[catalogo.Version] = catalogo
	}
	return resultado
}

func numeroVersionMotivoCobertura(version int) string {
	return strconv.Itoa(version)
}

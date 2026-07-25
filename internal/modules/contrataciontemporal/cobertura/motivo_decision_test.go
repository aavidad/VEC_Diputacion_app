package cobertura

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var instanteMotivoCoberturaPrueba = time.Date(
	2026, 7, 25, 10, 0, 0, 0, time.UTC,
)

type consultaMotivoCoberturaPrueba struct {
	catalogo dominiovec.CatalogoConfigurable
	err      error
}

func (c *consultaMotivoCoberturaPrueba) ObtenerCatalogo(
	_ context.Context,
	id string,
	version int,
) (dominiovec.CatalogoConfigurable, error) {
	if c.err != nil {
		return dominiovec.CatalogoConfigurable{}, c.err
	}
	if id != c.catalogo.ID || version != c.catalogo.Version {
		return dominiovec.CatalogoConfigurable{},
			puertosvec.ErrCatalogoNoEncontrado
	}
	return c.catalogo.ClonarCanonico()
}

func (c *consultaMotivoCoberturaPrueba) ListarVersionesCatalogo(
	context.Context,
	string,
) ([]dominiovec.CatalogoConfigurable, error) {
	clon, err := c.catalogo.ClonarCanonico()
	if err != nil {
		return nil, err
	}
	return []dominiovec.CatalogoConfigurable{clon}, nil
}

func TestResolutorMotivoCoberturaExigePublicacionExactaEI18n(
	t *testing.T,
) {
	catalogo := catalogoMotivoCoberturaPublicado(t, nil)
	motivo := motivoCoberturaPrueba(t, catalogo)
	resolutor := resolutorMotivoCoberturaPrueba(t, catalogo)
	resolucion, err := resolutor.Resolver(
		context.Background(),
		motivo,
		instanteMotivoCoberturaPrueba,
	)
	if err != nil {
		t.Fatalf("resolver motivo publicado: %v", err)
	}
	copia, err := resolucion.Motivo()
	if err != nil || copia != motivo {
		t.Fatalf("copia nominal inesperada: %#v, %v", copia, err)
	}
	resueltaEn, err := resolucion.ResueltaEn()
	if err != nil || !resueltaEn.Equal(instanteMotivoCoberturaPrueba) {
		t.Fatalf("instante de resolución: %v, %v", resueltaEn, err)
	}
	if strings.Contains(resolucion.String(), motivo.ReferenciaCatalogo.EntradaClave) {
		t.Fatal("String expone la referencia resuelta")
	}
	assertResolucionMotivoCoberturaRedactada(t, resolucion)
}

func TestResolutorMotivoCoberturaRechazaEstadoNoPublicado(t *testing.T) {
	borrador := borradorMotivoCoberturaPrueba(t, nil)
	publicado := catalogoMotivoCoberturaPublicado(t, nil)
	retirado, err := publicado.Retirar(
		"actor_retirada_motivo_01",
		"aprobacion_retirada_motivo_01",
		"Retirada gobernada.",
		instanteMotivoCoberturaPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	futuro, err := borrador.Publicar(
		"actor_publicador_motivo_01",
		"aprobacion_publicacion_motivo_01",
		"Publicación futura gobernada.",
		instanteMotivoCoberturaPrueba.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	for nombre, catalogo := range map[string]dominiovec.CatalogoConfigurable{
		"borrador":           borrador,
		"retirado":           retirado,
		"publicación futura": futuro,
	} {
		t.Run(nombre, func(t *testing.T) {
			resolutor := resolutorMotivoCoberturaPrueba(t, catalogo)
			motivo := motivoCoberturaPrueba(t, catalogo)
			if _, err := resolutor.Resolver(
				context.Background(),
				motivo,
				instanteMotivoCoberturaPrueba,
			); !errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) {
				t.Fatalf("catálogo no publicado aceptado: %v", err)
			}
		})
	}
}

func TestResolutorMotivoCoberturaRechazaHuellaEntradaYVigencia(
	t *testing.T,
) {
	catalogo := catalogoMotivoCoberturaPublicado(t, nil)
	resolutor := resolutorMotivoCoberturaPrueba(t, catalogo)
	base := motivoCoberturaPrueba(t, catalogo)
	casos := []struct {
		nombre string
		motivo domain.MotivoGobernadoDecisionCobertura
	}{
		{
			nombre: "huella falsa",
			motivo: func() domain.MotivoGobernadoDecisionCobertura {
				m := base
				m.ReferenciaCatalogo.CatalogoHuellaSHA256 =
					strings.Repeat("9", 64)
				return m
			}(),
		},
		{
			nombre: "entrada ausente",
			motivo: func() domain.MotivoGobernadoDecisionCobertura {
				m := base
				m.ReferenciaCatalogo.EntradaClave = "motivo_ausente_01"
				return m
			}(),
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := resolutor.Resolver(
				context.Background(),
				caso.motivo,
				instanteMotivoCoberturaPrueba,
			); !errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) {
				t.Fatalf("referencia falsa aceptada: %v", err)
			}
		})
	}

	caducado := catalogoMotivoCoberturaPublicado(
		t,
		map[string]string{
			atributoClaveI18nMotivoDecisionCobertura: "cobertura.motivo.rectificacion",
		},
	)
	caducado.Entradas[0].VigenteHasta = instanteMotivoCoberturaPrueba
	caducado, err := caducado.ClonarCanonico()
	if err != nil {
		t.Fatal(err)
	}
	resolutor = resolutorMotivoCoberturaPrueba(t, caducado)
	if _, err := resolutor.Resolver(
		context.Background(),
		motivoCoberturaPrueba(t, caducado),
		instanteMotivoCoberturaPrueba,
	); !errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) {
		t.Fatalf("entrada caducada aceptada: %v", err)
	}
}

func TestResolutorMotivoCoberturaExigeAtributoI18nExacto(t *testing.T) {
	casos := []struct {
		nombre    string
		atributos map[string]string
	}{
		{"ausente", map[string]string{"otro_atributo": "valor"}},
		{
			"distinto",
			map[string]string{
				atributoClaveI18nMotivoDecisionCobertura: "cobertura.motivo.otro",
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			catalogo := catalogoMotivoCoberturaPublicado(
				t,
				caso.atributos,
			)
			resolutor := resolutorMotivoCoberturaPrueba(t, catalogo)
			motivo := motivoCoberturaPrueba(t, catalogo)
			if _, err := resolutor.Resolver(
				context.Background(),
				motivo,
				instanteMotivoCoberturaPrueba,
			); !errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) {
				t.Fatalf("i18n no ligada aceptada: %v", err)
			}
		})
	}
}

func TestResolutorMotivoCoberturaFallaCerradoSinFiltrarProveedor(
	t *testing.T,
) {
	catalogo := catalogoMotivoCoberturaPublicado(t, nil)
	consulta := &consultaMotivoCoberturaPrueba{
		catalogo: catalogo,
		err:      errors.New("DNI 12345678Z: detalle privado"),
	}
	resolutor, err := NuevoResolutorMotivoDecisionCobertura(
		consulta,
		catalogo.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = resolutor.Resolver(
		context.Background(),
		motivoCoberturaPrueba(t, catalogo),
		instanteMotivoCoberturaPrueba,
	)
	if !errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) ||
		strings.Contains(err.Error(), "12345678Z") {
		t.Fatalf("error privado expuesto: %v", err)
	}
}

func resolutorMotivoCoberturaPrueba(
	t *testing.T,
	catalogo dominiovec.CatalogoConfigurable,
) *ResolutorMotivoDecisionCobertura {
	t.Helper()
	resolutor, err := NuevoResolutorMotivoDecisionCobertura(
		&consultaMotivoCoberturaPrueba{catalogo: catalogo},
		catalogo.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	return resolutor
}

func motivoCoberturaPrueba(
	t *testing.T,
	catalogo dominiovec.CatalogoConfigurable,
) domain.MotivoGobernadoDecisionCobertura {
	t.Helper()
	huella, err := catalogo.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	return domain.MotivoGobernadoDecisionCobertura{
		ReferenciaCatalogo: dominiovec.ReferenciaEntradaCatalogo{
			CatalogoID: catalogo.ID, CatalogoVersion: catalogo.Version,
			CatalogoHuellaSHA256: huella,
			EntradaClave:         catalogo.Entradas[0].Clave,
		},
		ClaveI18n: "cobertura.motivo.rectificacion",
	}
}

func catalogoMotivoCoberturaPublicado(
	t *testing.T,
	atributos map[string]string,
) dominiovec.CatalogoConfigurable {
	t.Helper()
	borrador := borradorMotivoCoberturaPrueba(t, atributos)
	publicado, err := borrador.Publicar(
		"actor_publicador_motivo_01",
		"aprobacion_publicacion_motivo_01",
		"Publicación gobernada.",
		instanteMotivoCoberturaPrueba.Add(-time.Hour),
	)
	if err != nil {
		t.Fatalf("publicar catálogo: %v", err)
	}
	return publicado
}

func borradorMotivoCoberturaPrueba(
	t *testing.T,
	atributos map[string]string,
) dominiovec.CatalogoConfigurable {
	t.Helper()
	if atributos == nil {
		atributos = map[string]string{
			atributoClaveI18nMotivoDecisionCobertura: "cobertura.motivo.rectificacion",
		}
	}
	borrador := dominiovec.CatalogoConfigurable{
		ID: "motivos_cobertura", Version: 7, Revision: 1,
		VersionAnteriorRef: "motivos_cobertura:6",
		ModuloID:           "contratacion_temporal", Nombre: "Motivos de cobertura",
		FuenteRef:      "politica_motivos_cobertura",
		MotivoCreacion: "Versión gobernada.",
		Entradas: []dominiovec.EntradaCatalogoConfigurable{{
			Clave: "rectificacion_decision", Etiqueta: "Rectificación",
			Orden: 10, VigenteDesde: instanteMotivoCoberturaPrueba.Add(-2 * time.Hour),
			Atributos: atributos,
		}},
		Estado:    dominiovec.EstadoCatalogoBorrador,
		CreadoPor: "actor_creador_motivo_01",
		CreadoEn:  instanteMotivoCoberturaPrueba.Add(-2 * time.Hour),
	}
	if err := borrador.Validar(); err != nil {
		t.Fatalf("catálogo de prueba: %v", err)
	}
	return borrador
}

func assertResolucionMotivoCoberturaRedactada(
	t *testing.T,
	resolucion ResolucionMotivoDecisionCobertura,
) {
	t.Helper()
	for nombre, representacion := range map[string]string{
		"String":   resolucion.String(),
		"GoString": fmt.Sprintf("%#v", resolucion),
		"Format":   fmt.Sprintf("%+v", resolucion),
		"LogValue": resolucion.LogValue().String(),
	} {
		if representacion != redaccionResolucionMotivoCobertura {
			t.Fatalf("%s no redactado: %q", nombre, representacion)
		}
	}
	texto, err := resolucion.MarshalText()
	if err != nil || string(texto) != redaccionResolucionMotivoCobertura {
		t.Fatalf("texto no redactado: %q, %v", texto, err)
	}
	contenido, err := json.Marshal(resolucion)
	if err != nil ||
		string(contenido) != `"`+redaccionResolucionMotivoCobertura+`"` {
		t.Fatalf("JSON no redactado: %s, %v", contenido, err)
	}
	registro := slog.GroupValue(slog.Any("motivo", resolucion))
	if strings.Contains(
		registro.String(),
		resolucion.motivo.ReferenciaCatalogo.EntradaClave,
	) {
		t.Fatal("LogValue expone referencia")
	}
}

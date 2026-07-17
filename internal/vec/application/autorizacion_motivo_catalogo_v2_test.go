package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type consultaCatalogosMotivoAutorizacionV2Prueba struct {
	catalogo domain.CatalogoConfigurable
	err      error
}

func (c *consultaCatalogosMotivoAutorizacionV2Prueba) ObtenerCatalogo(
	_ context.Context,
	id string,
	version int,
) (domain.CatalogoConfigurable, error) {
	if c.err != nil {
		return domain.CatalogoConfigurable{}, c.err
	}
	if id != c.catalogo.ID || version != c.catalogo.Version {
		return domain.CatalogoConfigurable{}, ports.ErrCatalogoNoEncontrado
	}
	return c.catalogo.ClonarCanonico()
}

func (c *consultaCatalogosMotivoAutorizacionV2Prueba) ListarVersionesCatalogo(
	context.Context,
	string,
) ([]domain.CatalogoConfigurable, error) {
	return []domain.CatalogoConfigurable{c.catalogo}, nil
}

func TestValidadorReferenciaMotivoCatalogoV2ResuelveDocumentoPublicadoExacto(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	catalogo := catalogoMotivosAutorizacionV2Prueba(t, ahora)
	referencia := referenciaCatalogoMotivoAutorizacionV2Prueba(t, catalogo, claveMotivoAutorizacionV2Prueba)
	validador, err := NuevoValidadorReferenciaMotivoCatalogoV2(
		&consultaCatalogosMotivoAutorizacionV2Prueba{catalogo: catalogo},
		catalogo.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := validador.ValidarReferenciaMotivoAutorizacionV2(
		context.Background(),
		referencia,
		ahora,
	); err != nil {
		t.Fatalf("referencia publicada exacta rechazada: %v", err)
	}

	for nombre, mutar := range map[string]func(*domain.ReferenciaEntradaCatalogo){
		"huella fabricada": func(r *domain.ReferenciaEntradaCatalogo) {
			r.CatalogoHuellaSHA256 = "9999999999999999999999999999999999999999999999999999999999999999"
		},
		"entrada inexistente con PII": func(r *domain.ReferenciaEntradaCatalogo) {
			r.EntradaClave = "dni_12345678z"
		},
		"entrada opaca inexistente": func(r *domain.ReferenciaEntradaCatalogo) {
			r.EntradaClave = claveMotivoAutorizacionV2Alternativa
		},
		"otro catalogo": func(r *domain.ReferenciaEntradaCatalogo) { r.CatalogoID = "personas" },
		"otra version":  func(r *domain.ReferenciaEntradaCatalogo) { r.CatalogoVersion++ },
	} {
		t.Run(nombre, func(t *testing.T) {
			alterada := referencia
			mutar(&alterada)
			if err := validador.ValidarReferenciaMotivoAutorizacionV2(
				context.Background(),
				alterada,
				ahora,
			); !errors.Is(err, domain.ErrSolicitudAutorizacionInvalida) {
				t.Fatalf("referencia no resuelta aceptada: %v", err)
			}
		})
	}
}

func TestValidadorReferenciaMotivoCatalogoV2RechazaEstadoOVigenciaNoUtilizable(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	borrador := borradorCatalogoMotivosAutorizacionV2Prueba(t, ahora)
	publicado := catalogoMotivosAutorizacionV2Prueba(t, ahora)
	retirado, err := publicado.Retirar(
		"responsable_rrhh",
		"retirada_motivos_autorizacion_v2",
		"Sustitucion del catalogo",
		ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	publicadoFuturo, err := borrador.Publicar(
		"responsable_seguridad",
		"aprobacion_motivos_autorizacion_v2_futura",
		"Publicacion programada",
		ahora.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	entradaFutura := publicado
	entradaFutura.Entradas = append([]domain.EntradaCatalogoConfigurable(nil), publicado.Entradas...)
	entradaFutura.Entradas[0].VigenteDesde = ahora.Add(time.Hour)
	entradaFutura, err = entradaFutura.ClonarCanonico()
	if err != nil {
		t.Fatal(err)
	}

	for nombre, catalogo := range map[string]domain.CatalogoConfigurable{
		"borrador":           borrador,
		"retirado":           retirado,
		"publicado despues":  publicadoFuturo,
		"entrada no vigente": entradaFutura,
	} {
		t.Run(nombre, func(t *testing.T) {
			validador, err := NuevoValidadorReferenciaMotivoCatalogoV2(
				&consultaCatalogosMotivoAutorizacionV2Prueba{catalogo: catalogo},
				catalogo.ID,
			)
			if err != nil {
				t.Fatal(err)
			}
			referencia := referenciaCatalogoMotivoAutorizacionV2Prueba(
				t,
				catalogo,
				claveMotivoAutorizacionV2Prueba,
			)
			if err := validador.ValidarReferenciaMotivoAutorizacionV2(
				context.Background(),
				referencia,
				ahora,
			); !errors.Is(err, domain.ErrSolicitudAutorizacionInvalida) {
				t.Fatalf("catalogo no utilizable aceptado: %v", err)
			}
		})
	}
}

func TestValidadorReferenciaMotivoCatalogoV2FallaCerradoConContextoONulo(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	catalogo := catalogoMotivosAutorizacionV2Prueba(t, ahora)
	var consultaNula *consultaCatalogosMotivoAutorizacionV2Prueba
	if validador, err := NuevoValidadorReferenciaMotivoCatalogoV2(consultaNula, catalogo.ID); validador != nil || !errors.Is(err, domain.ErrConfiguracionAccesoInvalida) {
		t.Fatalf("consulta nula tipada aceptada: validador=%v err=%v", validador, err)
	}
	validador, err := NuevoValidadorReferenciaMotivoCatalogoV2(
		&consultaCatalogosMotivoAutorizacionV2Prueba{catalogo: catalogo},
		catalogo.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	referencia := referenciaCatalogoMotivoAutorizacionV2Prueba(t, catalogo, claveMotivoAutorizacionV2Prueba)
	if err := validador.ValidarReferenciaMotivoAutorizacionV2(ctx, referencia, ahora); !errors.Is(err, domain.ErrSolicitudAutorizacionInvalida) || !errors.Is(err, context.Canceled) {
		t.Fatalf("contexto cancelado no fallo cerrado: %v", err)
	}
}

func catalogoMotivosAutorizacionV2Prueba(
	t *testing.T,
	ahora time.Time,
) domain.CatalogoConfigurable {
	t.Helper()
	borrador := borradorCatalogoMotivosAutorizacionV2Prueba(t, ahora)
	publicado, err := borrador.Publicar(
		"responsable_seguridad",
		"aprobacion_motivos_autorizacion_v2",
		"Publicacion del catalogo gobernado",
		ahora.Add(-time.Hour),
	)
	if err != nil {
		t.Fatalf("publicar catalogo de prueba: %v", err)
	}
	return publicado
}

func borradorCatalogoMotivosAutorizacionV2Prueba(
	t *testing.T,
	ahora time.Time,
) domain.CatalogoConfigurable {
	t.Helper()
	borrador := domain.CatalogoConfigurable{
		ID: "motivos_autorizacion", Version: 2, Revision: 1,
		VersionAnteriorRef: "motivos_autorizacion:1",
		ModuloID:           "nucleo", Nombre: "Motivos de autorizacion",
		FuenteRef: "politica_motivos_autorizacion", MotivoCreacion: "Nueva version gobernada",
		Entradas: []domain.EntradaCatalogoConfigurable{
			{Clave: claveMotivoAutorizacionV2Prueba, Etiqueta: "Consulta administrativa", Orden: 10, VigenteDesde: ahora.Add(-2 * time.Hour)},
		},
		Estado: domain.EstadoCatalogoBorrador, CreadoPor: "responsable_catalogos", CreadoEn: ahora.Add(-2 * time.Hour),
	}
	if err := borrador.Validar(); err != nil {
		t.Fatalf("catalogo borrador de prueba invalido: %v", err)
	}
	return borrador
}

func referenciaCatalogoMotivoAutorizacionV2Prueba(
	t *testing.T,
	catalogo domain.CatalogoConfigurable,
	clave string,
) domain.ReferenciaEntradaCatalogo {
	t.Helper()
	huella, err := catalogo.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	return domain.ReferenciaEntradaCatalogo{
		CatalogoID: catalogo.ID, CatalogoVersion: catalogo.Version,
		CatalogoHuellaSHA256: huella, EntradaClave: clave,
	}
}

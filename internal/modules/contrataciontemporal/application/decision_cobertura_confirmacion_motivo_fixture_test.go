package application

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	claveMotivoConfirmacionPrueba  = "rectificacion_decision"
	moduloMotivoConfirmacionPrueba = "contratacion_temporal"
)

type consultaMotivoConfirmacionPrueba struct {
	catalogo dominiovec.CatalogoConfigurable
}

func (c consultaMotivoConfirmacionPrueba) ObtenerCatalogoAcotado(
	ctx context.Context,
	id string,
	version int,
	limites puertosvec.LimitesConsultaCatalogosAcotada,
) (puertosvec.ResultadoConsultaCatalogoAcotado, error) {
	if err := ctx.Err(); err != nil {
		return puertosvec.ResultadoConsultaCatalogoAcotado{}, err
	}
	if limites.Validar() != nil ||
		id != c.catalogo.ID ||
		version != c.catalogo.Version {
		return puertosvec.ResultadoConsultaCatalogoAcotado{},
			puertosvec.ErrCatalogoNoEncontrado
	}
	clon, err := c.catalogo.ClonarCanonico()
	return puertosvec.ResultadoConsultaCatalogoAcotado{
		Catalogo: clon,
	}, err
}

func (c consultaMotivoConfirmacionPrueba) ListarVersionesCatalogoAcotado(
	ctx context.Context,
	id string,
	limites puertosvec.LimitesConsultaCatalogosAcotada,
) (puertosvec.ResultadoConsultaCatalogosAcotada, error) {
	if err := ctx.Err(); err != nil {
		return puertosvec.ResultadoConsultaCatalogosAcotada{}, err
	}
	if limites.Validar() != nil || id != c.catalogo.ID {
		return puertosvec.ResultadoConsultaCatalogosAcotada{},
			puertosvec.ErrCatalogoNoEncontrado
	}
	clon, err := c.catalogo.ClonarCanonico()
	return puertosvec.ResultadoConsultaCatalogosAcotada{
		Catalogos: []dominiovec.CatalogoConfigurable{clon},
	}, err
}

type resolutorSecuenciaMotivoConfirmacionPrueba struct {
	mu          sync.Mutex
	resolutores []*cobertura.ResolutorMotivoDecisionCobertura
	err         error
	invalida    bool
	errores     map[int]error
	invalidas   map[int]bool
	llamadas    int
}

func (r *resolutorSecuenciaMotivoConfirmacionPrueba) ResolverClave(
	ctx context.Context,
	clave domain.ClaveCatalogo,
	instante time.Time,
) (cobertura.ResolucionMotivoDecisionCobertura, error) {
	r.mu.Lock()
	indice := r.llamadas
	r.llamadas++
	errForzado := r.err
	invalida := r.invalida
	if errLlamada, existe := r.errores[indice+1]; existe {
		errForzado = errLlamada
	}
	if invalidaLlamada, existe := r.invalidas[indice+1]; existe {
		invalida = invalidaLlamada
	}
	if indice >= len(r.resolutores) {
		indice = len(r.resolutores) - 1
	}
	var resolutor *cobertura.ResolutorMotivoDecisionCobertura
	if indice >= 0 {
		resolutor = r.resolutores[indice]
	}
	r.mu.Unlock()
	if errForzado != nil {
		return cobertura.ResolucionMotivoDecisionCobertura{}, errForzado
	}
	if invalida {
		return cobertura.ResolucionMotivoDecisionCobertura{}, nil
	}
	if resolutor == nil {
		return cobertura.ResolucionMotivoDecisionCobertura{},
			errors.New("resolutor de motivo de prueba ausente")
	}
	return resolutor.ResolverClave(ctx, clave, instante)
}

func (r *resolutorSecuenciaMotivoConfirmacionPrueba) total() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.llamadas
}

func nuevoResolutorMotivoConfirmacionPrueba(
	t *testing.T,
	id string,
	instante time.Time,
) *cobertura.ResolutorMotivoDecisionCobertura {
	t.Helper()
	borrador := dominiovec.CatalogoConfigurable{
		ID:             id,
		Version:        1,
		Revision:       1,
		ModuloID:       moduloMotivoConfirmacionPrueba,
		Nombre:         "Motivos funcionales de cobertura",
		FuenteRef:      "politica_motivos_confirmacion_cobertura",
		MotivoCreacion: "Catálogo gobernado de prueba.",
		Entradas: []dominiovec.EntradaCatalogoConfigurable{{
			Clave:        claveMotivoConfirmacionPrueba,
			Etiqueta:     "Rectificación de decisión",
			Orden:        1,
			VigenteDesde: instante.Add(-2 * time.Hour),
			Atributos: map[string]string{
				"clave_i18n": "contratacion_temporal.cobertura.motivo.rectificacion",
			},
		}},
		Estado:    dominiovec.EstadoCatalogoBorrador,
		CreadoPor: "actor_gobierno_motivos_cobertura_01",
		CreadoEn:  instante.Add(-2 * time.Hour),
	}
	if err := borrador.Validar(); err != nil {
		t.Fatal(err)
	}
	publicado, err := borrador.Publicar(
		"actor_publicador_motivos_cobertura_01",
		"aprobacion_motivos_cobertura_01",
		"Publicación gobernada para la decisión.",
		instante.Add(-time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	resolutor, err := cobertura.NuevoResolutorMotivoDecisionCobertura(
		consultaMotivoConfirmacionPrueba{catalogo: publicado},
		publicado.ID,
		publicado.ModuloID,
	)
	if err != nil {
		t.Fatal(err)
	}
	return resolutor
}

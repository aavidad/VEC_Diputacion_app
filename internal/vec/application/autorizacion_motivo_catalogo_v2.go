package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ValidadorReferenciaMotivoCatalogoV2 relee la version exacta del catalogo
// configurado. Una ReferenciaEntradaCatalogo rellenada por el llamador nunca
// basta: deben coincidir documento publicado, huella y entrada vigente.
type ValidadorReferenciaMotivoCatalogoV2 struct {
	consulta   ports.ConsultaCatalogosConfigurables
	catalogoID string
}

func NuevoValidadorReferenciaMotivoCatalogoV2(
	consulta ports.ConsultaCatalogosConfigurables,
	catalogoID string,
) (*ValidadorReferenciaMotivoCatalogoV2, error) {
	centinela := domain.ReferenciaEntradaCatalogo{
		CatalogoID: catalogoID, CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("a", 64),
		EntradaClave:         "motivo_00000000000000000000000000000001",
	}
	if dependenciaAutorizacionNula(consulta) || centinela.Validar() != nil {
		return nil, domain.ErrConfiguracionAccesoInvalida
	}
	return &ValidadorReferenciaMotivoCatalogoV2{consulta: consulta, catalogoID: catalogoID}, nil
}

func (v *ValidadorReferenciaMotivoCatalogoV2) ValidarReferenciaMotivoAutorizacionV2(
	ctx context.Context,
	referencia domain.ReferenciaEntradaCatalogo,
	instante time.Time,
) error {
	if v == nil || dependenciaAutorizacionNula(v.consulta) || ctx == nil ||
		referencia.CatalogoID != v.catalogoID || !domain.ReferenciaMotivoAutorizacionV2Valida(referencia) ||
		instante.IsZero() || instante.Location() != time.UTC || instante.Nanosecond()%1_000 != 0 {
		return domain.ErrSolicitudAutorizacionInvalida
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(domain.ErrSolicitudAutorizacionInvalida, err)
	}
	catalogoVivo, err := v.consulta.ObtenerCatalogo(
		ctx,
		referencia.CatalogoID,
		referencia.CatalogoVersion,
	)
	if err != nil || ctx.Err() != nil {
		return errors.Join(domain.ErrSolicitudAutorizacionInvalida, err, ctx.Err())
	}
	// Se toma una unica instantanea defensiva. Huella y busqueda de entrada
	// operan sobre este clon, nunca sobre el slice vivo devuelto por el adaptador.
	catalogo, err := catalogoVivo.ClonarCanonico()
	if err != nil ||
		catalogo.ID != referencia.CatalogoID || catalogo.Version != referencia.CatalogoVersion ||
		catalogo.Estado != domain.EstadoCatalogoPublicado || catalogo.PublicadoEn.After(instante) {
		return errors.Join(domain.ErrSolicitudAutorizacionInvalida, err)
	}
	huella, err := catalogo.HuellaSHA256()
	if err != nil || huella != referencia.CatalogoHuellaSHA256 {
		return errors.Join(domain.ErrSolicitudAutorizacionInvalida, err)
	}
	for _, entrada := range catalogo.Entradas {
		if entrada.Clave == referencia.EntradaClave && entrada.VigenteEn(instante) {
			return nil
		}
	}
	return domain.ErrSolicitudAutorizacionInvalida
}

var _ ports.ValidadorReferenciaMotivoAutorizacionV2 = (*ValidadorReferenciaMotivoCatalogoV2)(nil)

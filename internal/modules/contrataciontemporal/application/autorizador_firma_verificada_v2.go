package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// AutorizadorNominalFirmaV2 conserva la selección del plan que recibió el
// PDP. El registro consume esa misma copia y nunca vuelve a resolver el plan.
type AutorizadorNominalFirmaV2 struct {
	plan   ports.FuenteDescriptorFirmaV2
	emisor ports.EmisorMaterialFirmaVerificadaV2
}

func NuevoAutorizadorNominalFirmaV2(plan ports.FuenteDescriptorFirmaV2, emisor ports.EmisorMaterialFirmaVerificadaV2) (*AutorizadorNominalFirmaV2, error) {
	if nula(plan) || nula(emisor) {
		return nil, ports.ErrCompetenciaFirmanteNoDisponible
	}
	return &AutorizadorNominalFirmaV2{plan, emisor}, nil
}

func (a *AutorizadorNominalFirmaV2) ObtenerPerfilActivoOperadorFirmaV2(ctx context.Context) (string, error) {
	if a == nil || ctx == nil || nula(a.emisor) {
		return "", ports.ErrFirmaDocumentoDenegada
	}
	return a.emisor.ObtenerPerfilActivoOperadorFirmaV2(ctx)
}

func (a *AutorizadorNominalFirmaV2) AutorizarFirmaVerificadaV2(ctx context.Context, m ports.MaterialFirmaVerificadaV2) (ports.CapacidadFirmaVerificadaV2, error) {
	var cero ports.CapacidadFirmaVerificadaV2
	if a == nil || ctx == nil || nula(a.plan) || nula(a.emisor) {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	m.EvidenciaFirmasCanonica = bytes.Clone(m.EvidenciaFirmasCanonica)
	if m.Validar() != nil {
		return cero, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	perfil, err := a.emisor.ObtenerPerfilActivoOperadorFirmaV2(ctx)
	if err != nil || perfil != m.PerfilActivoOperadorRef {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	d, err := a.plan.DescriptorFirmaV2(ctx, m)
	if err != nil {
		return cero, ports.ErrCompetenciaFirmanteNoDisponible
	}
	descriptor, err := CanonicoDescriptorFirmaVerificadaV2(m, d)
	if err != nil {
		return cero, err
	}
	defer clear(descriptor)
	ambitos, err := a.emisor.ObtenerAmbitosOperadorFirmaV2(ctx)
	if err != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	r, err := RecursoFirmaVerificadaV2ConAmbitos(m, descriptor, ambitos)
	if err != nil {
		return cero, err
	}
	material, err := a.emisor.AutorizarMaterialFirmaVerificadaV2(ctx, m, r)
	if err != nil {
		return cero, err
	}
	if err = ctx.Err(); err != nil {
		return cero, err
	}
	h := sha256.Sum256(descriptor)
	c := ports.TransportarMaterialFirmaVerificadaV2ConDescriptor(material, descriptor, hex.EncodeToString(h[:]))
	if ValidarCapacidadFirmaVerificadaV2(c, m) != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	return c, nil
}

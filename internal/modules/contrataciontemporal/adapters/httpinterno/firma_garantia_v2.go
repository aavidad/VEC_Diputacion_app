package httpinterno

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// La raíz obtiene la garantía observada de la sesión registrada y revalidada.
// No procede del nombre del cargo, certificado del PDF ni JSON del navegador.
type FuenteGarantiaAutenticacionFirmaV2 interface {
	GarantiaAutenticacionFirmaV2(context.Context) (string, error)
}

type FuenteGarantiaActoFirmaV2 interface {
	GarantiaActoFirmaV2(context.Context, ports.MaterialFirmaVerificadaV2) (string, error)
}

func garantiaAutenticacionFirmaV2(ctx context.Context, fuente any) (string, error) {
	f, presente := fuente.(FuenteGarantiaAutenticacionFirmaV2)
	if !presente {
		return "", nil // Conservar el contrato de las vías corporativas previas.
	}
	garantia, err := f.GarantiaAutenticacionFirmaV2(ctx)
	if err != nil {
		return "", err
	}
	if garantia != "" && garantia != "sustancial" {
		return "", ports.ErrFirmaDocumentoDenegada
	}
	return garantia, nil
}

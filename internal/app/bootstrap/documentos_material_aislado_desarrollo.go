package bootstrap

import (
	"context"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

type materialDocumentosAisladoDesarrollo struct {
	material *proveedorMaterialAltaContratacionTemporalDesarrollo
	propio    bool
	*gobiernoTecnicoAisladoDesarrollo
}

func (m *materialDocumentosAisladoDesarrollo) cerrar() {
	if m != nil && m.propio {
		m.gobiernoTecnicoAisladoDesarrollo.cerrar()
	}
}

// abrirMaterialDocumentosAisladoDesarrollo publica solo la audiencia V3 que
// consume Documentos. No prepara perfiles, acciones ni material funcional CT.
func abrirMaterialDocumentosAisladoDesarrollo(cfg config.Config,
	derivador *derivadorIdentidadOperacionDesarrollo,
	compartido *gobiernoTecnicoAisladoDesarrollo,
) (*materialDocumentosAisladoDesarrollo, error) {
	activo, err := cfg.DocumentosDesarrolloActivo()
	if err != nil || !activo {
		return nil, err
	}
	gobierno := compartido
	propio := gobierno == nil
	if propio {
		gobierno, err = abrirGobiernoTecnicoAisladoDesarrollo(cfg, derivador)
		if err != nil {
			return nil, err
		}
	}
	ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(15*time.Second))
	defer cancelar()
	descriptores := descriptoresMaterialDocumentosDesarrollo()
	if len(descriptores) != 1 {
		if propio {
			gobierno.cerrar()
		}
		return nil, errMaterialAutorizacionComunDesarrolloNoDisponible
	}
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptores)
	if err != nil {
		if propio {
			gobierno.cerrar()
		}
		return nil, err
	}
	material, err := nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno.gobierno, gobierno.base,
		relojContratacionTemporalDesarrollo{}, catalogo, descriptores[0].Audiencia)
	if propio {
		gobierno.base.borrarCopiasEfimeras()
	}
	if err != nil {
		if propio {
			gobierno.cerrar()
		}
		return nil, err
	}
	return &materialDocumentosAisladoDesarrollo{material: material, propio: propio, gobiernoTecnicoAisladoDesarrollo: gobierno}, nil
}

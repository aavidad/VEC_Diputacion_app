package seguridad

import (
	"context"
	"encoding/json"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	esquemaAmbitoFiscalizacionV1   = "vec.contratacion-temporal.fiscalizacion.ambito.v1"
	esquemaPeticionFiscalizacionV1 = "vec.contratacion-temporal.fiscalizacion.peticion.v1"
)

// AutoridadSellosFiscalizacionHMAC mantiene alineados los dominios de
// idempotencia y material sin exponer la clave del navegador a PostgreSQL.
type AutoridadSellosFiscalizacionHMAC struct {
	ambitos *llaveroHMAC
	huellas *llaveroHMAC
}

func NuevaAutoridadSellosFiscalizacionHMAC(
	activaAmbito ConfiguracionSelladorHMAC,
	retenidasAmbito []ConfiguracionSelladorHMAC,
	activaHuella ConfiguracionSelladorHMAC,
	retenidasHuella []ConfiguracionSelladorHMAC,
) (*AutoridadSellosFiscalizacionHMAC, error) {
	ambitos, err := nuevoLlaveroHMAC(
		ports.DominioAmbitoIdempotenciaFiscalizacion,
		activaAmbito,
		retenidasAmbito,
	)
	if err != nil {
		return nil, ErrSelladoAltaNoDisponible
	}
	huellas, err := nuevoLlaveroHMAC(
		ports.DominioHuellaPeticionFiscalizacion,
		activaHuella,
		retenidasHuella,
	)
	if err != nil || !llaverosFiscalizacionAlineados(ambitos, huellas) {
		return nil, ErrSelladoAltaNoDisponible
	}
	return &AutoridadSellosFiscalizacionHMAC{
		ambitos: ambitos,
		huellas: huellas,
	}, nil
}

func llaverosFiscalizacionAlineados(
	ambitos *llaveroHMAC,
	huellas *llaveroHMAC,
) bool {
	if ambitos == nil || huellas == nil ||
		ambitos.dominio != ports.DominioAmbitoIdempotenciaFiscalizacion ||
		huellas.dominio != ports.DominioHuellaPeticionFiscalizacion ||
		len(ambitos.generaciones) != len(huellas.generaciones) {
		return false
	}
	for indice := range ambitos.generaciones {
		generacionAmbito := generacionReferencia(
			ambitos.generaciones[indice].referenciaClave,
			ambitos.dominio,
		)
		generacionHuella := generacionReferencia(
			huellas.generaciones[indice].referenciaClave,
			huellas.dominio,
		)
		if generacionAmbito == 0 || generacionAmbito != generacionHuella {
			return false
		}
	}
	return true
}

func (a *AutoridadSellosFiscalizacionHMAC) SellarAmbitoFiscalizacion(
	ctx context.Context,
	solicitud ports.SolicitudSellarAmbitoIdempotencia,
) (ports.ColeccionSellosHMAC, error) {
	if ctx == nil || a == nil || a.ambitos == nil || a.huellas == nil ||
		solicitud.Validar() != nil ||
		!llaverosFiscalizacionAlineados(a.ambitos, a.huellas) {
		return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.ColeccionSellosHMAC{}, err
	}
	contenido, err := json.Marshal(struct {
		Esquema           string `json:"esquema"`
		ClaveIdempotencia string `json:"clave_idempotencia"`
		OrganizacionRef   string `json:"organizacion_ref"`
		ActorRef          string `json:"actor_ref"`
		PerfilRef         string `json:"perfil_ref"`
	}{
		Esquema:           esquemaAmbitoFiscalizacionV1,
		ClaveIdempotencia: solicitud.ClaveIdempotencia,
		OrganizacionRef:   solicitud.OrganizacionRef,
		ActorRef:          solicitud.ActorRef,
		PerfilRef:         solicitud.PerfilRef,
	})
	if err != nil {
		return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
	}
	defer borrar(contenido)
	coleccion, err := a.ambitos.sellar(ctx, contenido)
	if err != nil {
		return ports.ColeccionSellosHMAC{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.ColeccionSellosHMAC{}, err
	}
	return coleccion, nil
}

func (a *AutoridadSellosFiscalizacionHMAC) DerivarHuellaFiscalizacion(
	ctx context.Context,
	material ports.MaterialHuellaFiscalizacion,
) (ports.ColeccionSellosHMAC, error) {
	if ctx == nil || a == nil || a.ambitos == nil || a.huellas == nil ||
		material.Validar() != nil ||
		!llaverosFiscalizacionAlineados(a.ambitos, a.huellas) {
		return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.ColeccionSellosHMAC{}, err
	}
	contenido, err := json.Marshal(struct {
		Esquema           string `json:"esquema"`
		OrganizacionRef   string `json:"organizacion_ref"`
		ExpedienteRef     string `json:"expediente_ref"`
		VersionExpediente uint64 `json:"version_expediente"`
		ActorRef          string `json:"actor_ref"`
		PerfilRef         string `json:"perfil_ref"`
		Resultado         string `json:"resultado"`
		Observaciones     string `json:"observaciones"`
	}{
		Esquema:           esquemaPeticionFiscalizacionV1,
		OrganizacionRef:   material.OrganizacionRef,
		ExpedienteRef:     material.ExpedienteRef,
		VersionExpediente: material.VersionExpediente,
		ActorRef:          material.ActorRef,
		PerfilRef:         material.PerfilRef,
		Resultado:         string(material.Resultado),
		Observaciones:     material.Observaciones,
	})
	if err != nil {
		return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
	}
	defer borrar(contenido)
	coleccion, err := a.huellas.sellar(ctx, contenido)
	if err != nil {
		return ports.ColeccionSellosHMAC{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.ColeccionSellosHMAC{}, err
	}
	return coleccion, nil
}

var (
	_ ports.SelladorAmbitoFiscalizacion  = (*AutoridadSellosFiscalizacionHMAC)(nil)
	_ ports.DerivadorHuellaFiscalizacion = (*AutoridadSellosFiscalizacionHMAC)(nil)
)

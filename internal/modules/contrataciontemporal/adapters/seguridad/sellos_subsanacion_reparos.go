package seguridad

import (
	"context"
	"encoding/json"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	esquemaAmbitoSubsanacionReparoV1   = "vec.contratacion-temporal.subsanacion-reparo.ambito.v1"
	esquemaPeticionSubsanacionReparoV1 = "vec.contratacion-temporal.subsanacion-reparo.peticion.v1"
)

// AutoridadSellosSubsanacionReparoHMAC mantiene alineados los dominios de
// idempotencia y material sin exponer la clave del navegador a PostgreSQL.
type AutoridadSellosSubsanacionReparoHMAC struct {
	ambitos *llaveroHMAC
	huellas *llaveroHMAC
}

func NuevaAutoridadSellosSubsanacionReparoHMAC(
	activaAmbito ConfiguracionSelladorHMAC,
	retenidasAmbito []ConfiguracionSelladorHMAC,
	activaHuella ConfiguracionSelladorHMAC,
	retenidasHuella []ConfiguracionSelladorHMAC,
) (*AutoridadSellosSubsanacionReparoHMAC, error) {
	ambitos, err := nuevoLlaveroHMAC(
		ports.DominioAmbitoIdempotenciaSubsanacionReparo,
		activaAmbito,
		retenidasAmbito,
	)
	if err != nil {
		return nil, ErrSelladoAltaNoDisponible
	}
	huellas, err := nuevoLlaveroHMAC(
		ports.DominioHuellaPeticionSubsanacionReparo,
		activaHuella,
		retenidasHuella,
	)
	if err != nil || !llaverosSubsanacionReparoAlineados(ambitos, huellas) {
		return nil, ErrSelladoAltaNoDisponible
	}
	return &AutoridadSellosSubsanacionReparoHMAC{
		ambitos: ambitos,
		huellas: huellas,
	}, nil
}

func llaverosSubsanacionReparoAlineados(
	ambitos *llaveroHMAC,
	huellas *llaveroHMAC,
) bool {
	if ambitos == nil || huellas == nil ||
		ambitos.dominio != ports.DominioAmbitoIdempotenciaSubsanacionReparo ||
		huellas.dominio != ports.DominioHuellaPeticionSubsanacionReparo ||
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

func (a *AutoridadSellosSubsanacionReparoHMAC) SellarAmbitoSubsanacionReparo(
	ctx context.Context,
	solicitud ports.SolicitudSellarAmbitoIdempotencia,
) (ports.ColeccionSellosHMAC, error) {
	if ctx == nil || a == nil || a.ambitos == nil || a.huellas == nil ||
		solicitud.Validar() != nil ||
		!llaverosSubsanacionReparoAlineados(a.ambitos, a.huellas) {
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
		Esquema:           esquemaAmbitoSubsanacionReparoV1,
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

func (a *AutoridadSellosSubsanacionReparoHMAC) DerivarHuellaSubsanacionReparo(
	ctx context.Context,
	material ports.MaterialSubsanacionReparo,
) (ports.ColeccionSellosHMAC, error) {
	if ctx == nil || a == nil || a.ambitos == nil || a.huellas == nil ||
		!material.Valido() ||
		!llaverosSubsanacionReparoAlineados(a.ambitos, a.huellas) {
		return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.ColeccionSellosHMAC{}, err
	}
	contenido, err := json.Marshal(struct {
		Esquema         string `json:"esquema"`
		OrganizacionRef string `json:"organizacion_ref"`
		ExpedienteRef   string `json:"expediente_ref"`
		VersionEsperada uint64 `json:"version_esperada"`
		Observaciones   string `json:"observaciones"`
		ActorRef        string `json:"actor_ref"`
		PerfilRef       string `json:"perfil_ref"`
	}{
		Esquema:         esquemaPeticionSubsanacionReparoV1,
		OrganizacionRef: material.OrganizacionRef,
		ExpedienteRef:   material.ExpedienteRef,
		VersionEsperada: material.VersionEsperada,
		Observaciones:   material.Observaciones,
		ActorRef:        material.ActorRef,
		PerfilRef:       material.PerfilRef,
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
	_ ports.SelladorAmbitoSubsanacionReparo  = (*AutoridadSellosSubsanacionReparoHMAC)(nil)
	_ ports.DerivadorHuellaSubsanacionReparo = (*AutoridadSellosSubsanacionReparoHMAC)(nil)
)

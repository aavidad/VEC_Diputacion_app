package seguridad

import (
	"context"
	"encoding/json"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	esquemaAmbitoInformeJuridicoV1   = "vec.contratacion-temporal.informe-juridico.ambito.v1"
	esquemaPeticionInformeJuridicoV1 = "vec.contratacion-temporal.informe-juridico.peticion.v1"
)

// AutoridadSellosInformeJuridicoHMAC mantiene alineadas las generaciones de
// los dos dominios de clave y evita que la clave idempotente llegue a SQL.
type AutoridadSellosInformeJuridicoHMAC struct {
	ambitos *llaveroHMAC
	huellas *llaveroHMAC
}

func NuevaAutoridadSellosInformeJuridicoHMAC(
	activaAmbito ConfiguracionSelladorHMAC,
	retenidasAmbito []ConfiguracionSelladorHMAC,
	activaHuella ConfiguracionSelladorHMAC,
	retenidasHuella []ConfiguracionSelladorHMAC,
) (*AutoridadSellosInformeJuridicoHMAC, error) {
	ambitos, err := nuevoLlaveroHMAC(
		ports.DominioAmbitoIdempotenciaInformeJuridico,
		activaAmbito,
		retenidasAmbito,
	)
	if err != nil {
		return nil, ErrSelladoAltaNoDisponible
	}
	huellas, err := nuevoLlaveroHMAC(
		ports.DominioHuellaPeticionInformeJuridico,
		activaHuella,
		retenidasHuella,
	)
	if err != nil || !llaverosInformeJuridicoAlineados(ambitos, huellas) {
		return nil, ErrSelladoAltaNoDisponible
	}
	return &AutoridadSellosInformeJuridicoHMAC{
		ambitos: ambitos,
		huellas: huellas,
	}, nil
}

func llaverosInformeJuridicoAlineados(
	ambitos *llaveroHMAC,
	huellas *llaveroHMAC,
) bool {
	if ambitos == nil || huellas == nil ||
		ambitos.dominio != ports.DominioAmbitoIdempotenciaInformeJuridico ||
		huellas.dominio != ports.DominioHuellaPeticionInformeJuridico ||
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

func (a *AutoridadSellosInformeJuridicoHMAC) SellarAmbitoInformeJuridico(
	ctx context.Context,
	solicitud ports.SolicitudSellarAmbitoIdempotencia,
) (ports.ColeccionSellosHMAC, error) {
	if ctx == nil || a == nil || a.ambitos == nil || a.huellas == nil ||
		solicitud.Validar() != nil ||
		!llaverosInformeJuridicoAlineados(a.ambitos, a.huellas) {
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
		Esquema:           esquemaAmbitoInformeJuridicoV1,
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

func (a *AutoridadSellosInformeJuridicoHMAC) DerivarHuellaInformeJuridico(
	ctx context.Context,
	material ports.MaterialHuellaInformeJuridico,
) (ports.ColeccionSellosHMAC, error) {
	if ctx == nil || a == nil || a.ambitos == nil || a.huellas == nil ||
		material.Validar() != nil ||
		!llaverosInformeJuridicoAlineados(a.ambitos, a.huellas) {
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
	}{
		Esquema:           esquemaPeticionInformeJuridicoV1,
		OrganizacionRef:   material.OrganizacionRef,
		ExpedienteRef:     material.ExpedienteRef,
		VersionExpediente: material.VersionExpediente,
		ActorRef:          material.ActorRef,
		PerfilRef:         material.PerfilRef,
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
	_ ports.SelladorAmbitoInformeJuridico  = (*AutoridadSellosInformeJuridicoHMAC)(nil)
	_ ports.DerivadorHuellaInformeJuridico = (*AutoridadSellosInformeJuridicoHMAC)(nil)
)

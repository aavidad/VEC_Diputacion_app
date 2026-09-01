package seguridad

import (
	"context"
	"encoding/json"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	esquemaAmbitoAsignacionV1   = "vec.contratacion-temporal.asignacion.ambito.v1"
	esquemaPeticionAsignacionV1 = "vec.contratacion-temporal.asignacion.peticion.v1"
)

// AutoridadSellosAsignacionHMAC deriva conjuntamente el ámbito idempotente y
// la huella funcional de una asignación. Sus dos llaveros son inmutables,
// conservan la misma historia generacional y separan los dominios de clave.
type AutoridadSellosAsignacionHMAC struct {
	ambitos *llaveroHMAC
	huellas *llaveroHMAC
}

// NuevaAutoridadSellosAsignacionHMAC construye una única autoridad a partir
// de conectores opacos ya configurados. Cada posición retenida debe representar
// la misma generación en ambos dominios.
func NuevaAutoridadSellosAsignacionHMAC(
	activaAmbito ConfiguracionSelladorHMAC,
	retenidasAmbito []ConfiguracionSelladorHMAC,
	activaHuella ConfiguracionSelladorHMAC,
	retenidasHuella []ConfiguracionSelladorHMAC,
) (*AutoridadSellosAsignacionHMAC, error) {
	ambitos, err := nuevoLlaveroHMAC(
		ports.DominioAmbitoIdempotenciaAsignacion,
		activaAmbito,
		retenidasAmbito,
	)
	if err != nil {
		return nil, ErrSelladoAltaNoDisponible
	}
	huellas, err := nuevoLlaveroHMAC(
		ports.DominioHuellaPeticionAsignacion,
		activaHuella,
		retenidasHuella,
	)
	if err != nil || !llaverosAsignacionAlineados(ambitos, huellas) {
		return nil, ErrSelladoAltaNoDisponible
	}
	return &AutoridadSellosAsignacionHMAC{
		ambitos: ambitos,
		huellas: huellas,
	}, nil
}

func llaverosAsignacionAlineados(
	ambitos *llaveroHMAC,
	huellas *llaveroHMAC,
) bool {
	if ambitos == nil || huellas == nil ||
		ambitos.dominio != ports.DominioAmbitoIdempotenciaAsignacion ||
		huellas.dominio != ports.DominioHuellaPeticionAsignacion ||
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

func (a *AutoridadSellosAsignacionHMAC) SellarAmbitoAsignacion(
	ctx context.Context,
	solicitud ports.SolicitudSellarAmbitoIdempotencia,
) (ports.ColeccionSellosHMAC, error) {
	if ctx == nil || a == nil || a.ambitos == nil || a.huellas == nil ||
		solicitud.Validar() != nil ||
		!llaverosAsignacionAlineados(a.ambitos, a.huellas) {
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
		Esquema:           esquemaAmbitoAsignacionV1,
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

func (a *AutoridadSellosAsignacionHMAC) DerivarHuellaAsignacion(
	ctx context.Context,
	material ports.MaterialHuellaAsignacion,
) (ports.ColeccionSellosHMAC, error) {
	if ctx == nil || a == nil || a.ambitos == nil || a.huellas == nil ||
		material.Validar() != nil ||
		!llaverosAsignacionAlineados(a.ambitos, a.huellas) {
		return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.ColeccionSellosHMAC{}, err
	}
	contenido, err := json.Marshal(struct {
		Esquema                 string                        `json:"esquema"`
		Operacion               ports.TipoOperacionAsignacion `json:"operacion"`
		OrganizacionRef         string                        `json:"organizacion_ref"`
		ExpedienteRef           string                        `json:"expediente_ref"`
		VersionExpediente       uint64                        `json:"version_expediente"`
		ActorRef                string                        `json:"actor_ref"`
		PerfilRef               string                        `json:"perfil_ref"`
		UnidadRef               string                        `json:"unidad_ref"`
		ResponsableRef          string                        `json:"responsable_ref"`
		MotivoReasignacionClave domain.ClaveCatalogo          `json:"motivo_reasignacion_clave"`
		Observaciones           string                        `json:"observaciones"`
	}{
		Esquema:                 esquemaPeticionAsignacionV1,
		Operacion:               material.Operacion,
		OrganizacionRef:         material.OrganizacionRef,
		ExpedienteRef:           material.ExpedienteRef,
		VersionExpediente:       material.VersionExpediente,
		ActorRef:                material.ActorRef,
		PerfilRef:               material.PerfilRef,
		UnidadRef:               material.UnidadRef,
		ResponsableRef:          material.ResponsableRef,
		MotivoReasignacionClave: material.MotivoReasignacionClave,
		Observaciones:           material.Observaciones,
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
	_ ports.SelladorAmbitoAsignacion  = (*AutoridadSellosAsignacionHMAC)(nil)
	_ ports.DerivadorHuellaAsignacion = (*AutoridadSellosAsignacionHMAC)(nil)
)

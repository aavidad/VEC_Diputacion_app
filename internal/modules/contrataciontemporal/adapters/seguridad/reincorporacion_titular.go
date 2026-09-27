package seguridad

import (
	"context"
	"encoding/json"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// AutoridadSellosReincorporacionTitularHMAC mantiene dominios y generaciones
// propios del retorno del titular. Reutiliza el conector HMAC de CT, sin
// copiar ni exponer sus claves a la aplicación.
type AutoridadSellosReincorporacionTitularHMAC struct {
	ambitos, huellas *llaveroHMAC
}

func NuevaAutoridadSellosReincorporacionTitularHMAC(
	activaAmbito, activaHuella ConfiguracionSelladorHMAC,
	retenidasAmbito, retenidasHuella []ConfiguracionSelladorHMAC,
) (*AutoridadSellosReincorporacionTitularHMAC, error) {
	ambitos, err := nuevoLlaveroHMAC(ports.DominioAmbitoReincorporacionTitular, activaAmbito, retenidasAmbito)
	if err != nil {
		return nil, err
	}
	huellas, err := nuevoLlaveroHMAC(ports.DominioHuellaReincorporacionTitular, activaHuella, retenidasHuella)
	if err != nil || !llaverosAlineados(ambitos, huellas) {
		return nil, ErrSelladoAltaNoDisponible
	}
	return &AutoridadSellosReincorporacionTitularHMAC{ambitos: ambitos, huellas: huellas}, nil
}

func (a *AutoridadSellosReincorporacionTitularHMAC) SellarAmbitoReincorporacionTitular(
	ctx context.Context, s ports.SolicitudSellarAmbitoIdempotencia,
) (ports.ColeccionSellosHMAC, error) {
	if a == nil || a.ambitos == nil || ctx == nil || s.Validar() != nil {
		return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
	}
	contenido, err := json.Marshal(struct {
		Esquema           string `json:"esquema"`
		Operacion         string `json:"operacion"`
		ClaveIdempotencia string `json:"clave_idempotencia"`
		OrganizacionRef   string `json:"organizacion_ref"`
		ActorRef          string `json:"actor_ref"`
		PerfilRef         string `json:"perfil_ref"`
	}{esquemaAmbitoOperacionSeguimientoV1, ports.OperacionRegistrarReincorporacionTitular,
		s.ClaveIdempotencia, s.OrganizacionRef, s.ActorRef, s.PerfilRef})
	if err != nil {
		return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
	}
	defer borrar(contenido)
	return a.ambitos.sellar(ctx, contenido)
}

func (a *AutoridadSellosReincorporacionTitularHMAC) DerivarHuellaReincorporacionTitular(
	ctx context.Context, material []byte,
) (ports.ColeccionSellosHMAC, error) {
	if a == nil || a.huellas == nil || ctx == nil || len(material) == 0 || len(material) > 16*1024 {
		return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
	}
	return a.huellas.sellar(ctx, material)
}

var _ ports.SelladorReincorporacionTitular = (*AutoridadSellosReincorporacionTitularHMAC)(nil)

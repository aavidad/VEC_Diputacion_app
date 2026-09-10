package seguridad

import (
	"context"
	"encoding/json"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	esquemaAmbitoAnotacionAdministrativaV1   = "vec.contratacion-temporal.anotacion-administrativa.ambito.v1"
	esquemaPeticionAnotacionAdministrativaV1 = "vec.contratacion-temporal.anotacion-administrativa.peticion.v1"
)

// AutoridadSellosAnotacionAdministrativaHMAC conserva llaveros separados y
// generaciones alineadas para localizar una anotación sin revelar sus claves.
type AutoridadSellosAnotacionAdministrativaHMAC struct{ ambitos, huellas *llaveroHMAC }

func NuevaAutoridadSellosAnotacionAdministrativaHMAC(activaAmbito ConfiguracionSelladorHMAC, retenidasAmbito []ConfiguracionSelladorHMAC, activaHuella ConfiguracionSelladorHMAC, retenidasHuella []ConfiguracionSelladorHMAC) (*AutoridadSellosAnotacionAdministrativaHMAC, error) {
	ambitos, err := nuevoLlaveroHMAC(ports.DominioAmbitoIdempotenciaAnotacionAdministrativa, activaAmbito, retenidasAmbito)
	if err != nil {
		return nil, ErrSelladoAltaNoDisponible
	}
	huellas, err := nuevoLlaveroHMAC(ports.DominioHuellaPeticionAnotacionAdministrativa, activaHuella, retenidasHuella)
	if err != nil || !llaverosAnotacionAdministrativaAlineados(ambitos, huellas) {
		return nil, ErrSelladoAltaNoDisponible
	}
	return &AutoridadSellosAnotacionAdministrativaHMAC{ambitos: ambitos, huellas: huellas}, nil
}

func llaverosAnotacionAdministrativaAlineados(ambitos, huellas *llaveroHMAC) bool {
	if ambitos == nil || huellas == nil || ambitos.dominio != ports.DominioAmbitoIdempotenciaAnotacionAdministrativa || huellas.dominio != ports.DominioHuellaPeticionAnotacionAdministrativa || len(ambitos.generaciones) != len(huellas.generaciones) {
		return false
	}
	for i := range ambitos.generaciones {
		if a, b := generacionReferencia(ambitos.generaciones[i].referenciaClave, ambitos.dominio), generacionReferencia(huellas.generaciones[i].referenciaClave, huellas.dominio); a == 0 || a != b {
			return false
		}
	}
	return true
}

func (a *AutoridadSellosAnotacionAdministrativaHMAC) SellarAmbitoAnotacionAdministrativa(ctx context.Context, s ports.SolicitudSellarAmbitoIdempotencia) (ports.ColeccionSellosHMAC, error) {
	if ctx == nil || a == nil || a.ambitos == nil || a.huellas == nil || s.Validar() != nil || !llaverosAnotacionAdministrativaAlineados(a.ambitos, a.huellas) {
		return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.ColeccionSellosHMAC{}, err
	}
	b, err := json.Marshal(struct {
		Esquema      string `json:"esquema"`
		Clave        string `json:"clave_idempotencia"`
		Organizacion string `json:"organizacion_ref"`
		Actor        string `json:"actor_ref"`
		Perfil       string `json:"perfil_ref"`
	}{esquemaAmbitoAnotacionAdministrativaV1, s.ClaveIdempotencia, s.OrganizacionRef, s.ActorRef, s.PerfilRef})
	if err != nil {
		return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
	}
	defer borrar(b)
	coleccion, err := a.ambitos.sellar(ctx, b)
	if err != nil {
		return ports.ColeccionSellosHMAC{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.ColeccionSellosHMAC{}, err
	}
	return coleccion, nil
}

func (a *AutoridadSellosAnotacionAdministrativaHMAC) DerivarHuellaAnotacionAdministrativa(ctx context.Context, m ports.MaterialAnotacionAdministrativa) (ports.ColeccionSellosHMAC, error) {
	if ctx == nil || a == nil || a.ambitos == nil || a.huellas == nil || m.Validar() != nil || !llaverosAnotacionAdministrativaAlineados(a.ambitos, a.huellas) {
		return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.ColeccionSellosHMAC{}, err
	}
	b, err := json.Marshal(struct {
		Esquema       string `json:"esquema"`
		Organizacion  string `json:"organizacion_ref"`
		Expediente    string `json:"expediente_ref"`
		Solicitud     string `json:"solicitud_personal_ref"`
		Version       uint64 `json:"version_esperada"`
		Clave         string `json:"clave_idempotencia"`
		Actor         string `json:"actor_ref"`
		Perfil        string `json:"perfil_ref"`
		Observaciones string `json:"observaciones"`
	}{esquemaPeticionAnotacionAdministrativaV1, m.OrganizacionRef, m.ExpedienteRef, m.SolicitudPersonalRef, m.VersionEsperada, m.ClaveIdempotencia, m.ActorRef, m.PerfilRef, m.Observaciones})
	if err != nil {
		return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
	}
	defer borrar(b)
	coleccion, err := a.huellas.sellar(ctx, b)
	if err != nil {
		return ports.ColeccionSellosHMAC{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.ColeccionSellosHMAC{}, err
	}
	return coleccion, nil
}

var _ ports.SelladorAmbitoAnotacionAdministrativa = (*AutoridadSellosAnotacionAdministrativaHMAC)(nil)
var _ ports.DerivadorHuellaAnotacionAdministrativa = (*AutoridadSellosAnotacionAdministrativaHMAC)(nil)

package seguridad

import (
	"context"
	"encoding/json"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const esquemaAmbitoOperacionSeguimientoV1 = "vec.contratacion-temporal.seguimiento.ambito.v1"

// ConfiguracionLlaverosSeguimiento agrupa las generaciones HMAC de ámbito y
// de petición de una operación de seguimiento (cese, cierre, modificación).
type ConfiguracionLlaverosSeguimiento struct {
	Operacion                        string
	ActivaAmbito, ActivaHuella       ConfiguracionSelladorHMAC
	RetenidasAmbito, RetenidasHuella []ConfiguracionSelladorHMAC
}

type llaverosSeguimiento struct{ ambitos, huellas *llaveroHMAC }

// AutoridadSellosSeguimientoHMAC sella idempotencia y petición de las tres
// operaciones con dominios separados; la clave nunca sale del conector.
type AutoridadSellosSeguimientoHMAC struct {
	operaciones map[string]llaverosSeguimiento
}

func NuevaAutoridadSellosSeguimientoHMAC(configuraciones ...ConfiguracionLlaverosSeguimiento) (*AutoridadSellosSeguimientoHMAC, error) {
	a := &AutoridadSellosSeguimientoHMAC{operaciones: make(map[string]llaverosSeguimiento, len(configuraciones))}
	for _, c := range configuraciones {
		dominioAmbito, dominioHuella, ok := ports.DominiosHMACOperacionSeguimiento(c.Operacion)
		if _, repetida := a.operaciones[c.Operacion]; !ok || repetida {
			return nil, ErrSelladoAltaNoDisponible
		}
		ambitos, err := nuevoLlaveroHMAC(dominioAmbito, c.ActivaAmbito, c.RetenidasAmbito)
		if err != nil {
			return nil, ErrSelladoAltaNoDisponible
		}
		huellas, err := nuevoLlaveroHMAC(dominioHuella, c.ActivaHuella, c.RetenidasHuella)
		if err != nil || !llaverosAlineados(ambitos, huellas) {
			return nil, ErrSelladoAltaNoDisponible
		}
		a.operaciones[c.Operacion] = llaverosSeguimiento{ambitos: ambitos, huellas: huellas}
	}
	if len(a.operaciones) == 0 {
		return nil, ErrSelladoAltaNoDisponible
	}
	return a, nil
}

func llaverosAlineados(ambitos, huellas *llaveroHMAC) bool {
	if ambitos == nil || huellas == nil || len(ambitos.generaciones) != len(huellas.generaciones) {
		return false
	}
	for i := range ambitos.generaciones {
		ga := generacionReferencia(ambitos.generaciones[i].referenciaClave, ambitos.dominio)
		gh := generacionReferencia(huellas.generaciones[i].referenciaClave, huellas.dominio)
		if ga == 0 || ga != gh {
			return false
		}
	}
	return true
}

func (a *AutoridadSellosSeguimientoHMAC) llaveros(operacion string) (llaverosSeguimiento, bool) {
	if a == nil {
		return llaverosSeguimiento{}, false
	}
	l, ok := a.operaciones[operacion]
	return l, ok && l.ambitos != nil && l.huellas != nil
}

func (a *AutoridadSellosSeguimientoHMAC) SellarAmbitoOperacionSeguimiento(ctx context.Context, operacion string, s ports.SolicitudSellarAmbitoIdempotencia) (ports.ColeccionSellosHMAC, error) {
	l, ok := a.llaveros(operacion)
	if ctx == nil || !ok || s.Validar() != nil {
		return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
	}
	contenido, err := json.Marshal(struct {
		Esquema           string `json:"esquema"`
		Operacion         string `json:"operacion"`
		ClaveIdempotencia string `json:"clave_idempotencia"`
		OrganizacionRef   string `json:"organizacion_ref"`
		ActorRef          string `json:"actor_ref"`
		PerfilRef         string `json:"perfil_ref"`
	}{esquemaAmbitoOperacionSeguimientoV1, operacion, s.ClaveIdempotencia, s.OrganizacionRef, s.ActorRef, s.PerfilRef})
	if err != nil {
		return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
	}
	defer borrar(contenido)
	return l.ambitos.sellar(ctx, contenido)
}

// DerivarHuellaOperacionSeguimiento sella la intención ya canonizada por la
// aplicación (JSON de campos fijos); la misma intención da la misma huella.
func (a *AutoridadSellosSeguimientoHMAC) DerivarHuellaOperacionSeguimiento(ctx context.Context, operacion string, material []byte) (ports.ColeccionSellosHMAC, error) {
	l, ok := a.llaveros(operacion)
	if ctx == nil || !ok || len(material) == 0 || len(material) > 16*1024 {
		return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
	}
	return l.huellas.sellar(ctx, material)
}

// GenerarReferenciasSeguimiento reutiliza el CSPRNG de la autoridad de
// referencias; los prefijos no contienen identidad funcional.
func (g *GeneradorReferenciasAltaCriptografico) GenerarReferenciasSeguimiento(ctx context.Context) (ports.ReferenciasEfectoSeguimiento, error) {
	if !generadorValido(g) || ctx == nil {
		return ports.ReferenciasEfectoSeguimiento{}, ErrGeneracionReferenciaAlta
	}
	prefijos := []string{"reserva:ct-seguimiento:", "recibo:ct-seguimiento:", "evento:ct-seguimiento:"}
	valores := make([]string, len(prefijos))
	for i, prefijo := range prefijos {
		v, err := g.generar(ctx, prefijo)
		if err != nil {
			return ports.ReferenciasEfectoSeguimiento{}, err
		}
		valores[i] = v
	}
	r := ports.ReferenciasEfectoSeguimiento{ReservaRef: valores[0], ReciboRef: valores[1], EventoRef: valores[2]}
	if !r.Validas() {
		return ports.ReferenciasEfectoSeguimiento{}, ErrGeneracionReferenciaAlta
	}
	return r, nil
}

var (
	_ ports.SelladorOperacionSeguimiento    = (*AutoridadSellosSeguimientoHMAC)(nil)
	_ ports.GeneradorReferenciasSeguimiento = (*GeneradorReferenciasAltaCriptografico)(nil)
)

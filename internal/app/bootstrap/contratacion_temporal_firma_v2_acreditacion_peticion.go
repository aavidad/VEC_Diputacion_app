package bootstrap

import (
	"context"
	"sync"
)

type claveAcreditacionCertificadoFirmaCT struct{}

type claveVinculoCertificadoFirmaCT struct {
	huella, persona, cuenta, perfil string
}

type acreditacionCertificadoPorPeticionCT struct {
	mu       sync.Mutex
	vinculos map[claveVinculoCertificadoFirmaCT]bool
}

// Comparte sólo la prueba del certificado dentro de una petición. Cada acto
// conserva su decisión V3 y el cotejo transaccional de sus fuentes vigentes.
func acreditarCertificadoUnaVezCT(ctx context.Context, huella string, s seleccionNominalFirmaCT, acreditar func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c, ok := ctx.Value(claveAcreditacionCertificadoFirmaCT{}).(*acreditacionCertificadoPorPeticionCT)
	if !ok || c == nil {
		return acreditar()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	k := claveVinculoCertificadoFirmaCT{huella, s.PersonaRef, s.CuentaRef, s.PerfilRef}
	if c.vinculos[k] {
		return nil
	}
	if err := acreditar(); err != nil {
		return err
	}
	if c.vinculos == nil {
		c.vinculos = make(map[claveVinculoCertificadoFirmaCT]bool)
	}
	c.vinculos[k] = true
	return nil
}

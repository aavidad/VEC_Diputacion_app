package config

import (
	"strings"
	"testing"
)

func TestBolsaProvisionMiBolsaSoloConDobleLlaveYFormaValida(t *testing.T) {
	desarrollo := Config{ExecutionProfile: ExecutionProfileDevelopment, AuthMode: AuthModeDevelopment, DevelopmentGuard: DevelopmentGuardAcknowledgement}
	huella := strings.Repeat("ab", 32)
	for _, caso := range []struct {
		nombre             string
		base               Config
		aprobacion, imagen string
		valida             bool
	}{
		{"completa", desarrollo, " aprobacion:bolsa:20260929 ", huella, true},
		{"sin doble llave", Config{}, "aprobacion:bolsa:20260929", huella, false},
		{"sin huella", desarrollo, "aprobacion:bolsa:20260929", "", false},
		{"huella en mayúsculas", desarrollo, "aprobacion:bolsa:20260929", strings.ToUpper(huella), false},
		{"varias huellas", desarrollo, "aprobacion:bolsa:20260929", huella + "," + huella, false},
		{"referencia corta", desarrollo, "ap", huella, false},
		{"referencia con espacios", desarrollo, "aprobacion bolsa", huella, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			c := caso.base
			c.BolsaAprobacionProvisionMiBolsa, c.BolsaPreimagenProvisionMiBolsa = caso.aprobacion, caso.imagen
			ref, imagen := c.BolsaProvisionMiBolsa()
			if (ref != "" && imagen != "") != caso.valida || (ref == "") != (imagen == "") {
				t.Fatalf("aprobación %q %q", ref, imagen)
			}
		})
	}
}

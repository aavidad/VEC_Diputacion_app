package separacionportales

import (
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

func TestSemillaCorreoNominalSoloPerteneceAlExterno(t *testing.T) {
	for _, portal := range []Portal{PortalExterno, PortalInterno} {
		material := materialSintetico(t, portal)
		escribir(t, material, config.DevelopmentExternalMailSeedRelativePath, strings.Repeat("s", 32))
		err := ComprobarMaterial(portal, material)
		if portal == PortalExterno {
			if err != nil {
				t.Fatalf("semilla propia rechazada: %v", err)
			}
		} else if got := motivo(t, err); got != config.DevelopmentExternalMailSeedRelativePath {
			t.Fatalf("semilla externa admitida por el interno: %s", got)
		}
	}
}

func TestSemillaCorreoExternaNoAbrePrefijosNiKMS(t *testing.T) {
	for _, relativa := range []string{
		"usuarios/otra-semilla.bin", "usuarios/correos-externos-semilla.bin.bak", "usuarios/correos/semilla-externa.bin",
		"kms/clave-maestra.bin", "kms/atestacion-ed25519.key", "kms/revalidacion-ed25519.key", "tsa/clave-hmac.bin",
	} {
		material := materialSintetico(t, PortalExterno)
		escribir(t, material, config.DevelopmentExternalMailSeedRelativePath, strings.Repeat("s", 32))
		escribir(t, material, relativa, "material sintetico")
		if got := motivo(t, ComprobarMaterial(PortalExterno, material)); got != relativa {
			t.Fatalf("material ajeno admitido con la semilla nominal: %s", got)
		}
	}
}

func TestSeparacionComparaSemillaCorreoExternaConKMSInterno(t *testing.T) {
	interno, externo := materialSintetico(t, PortalInterno), materialSintetico(t, PortalExterno)
	declararCAInternaPrueba(t, interno, externo)
	escribir(t, interno, config.DevelopmentKMSSecretRelativePath, strings.Repeat("i", 32))
	escribir(t, externo, config.DevelopmentExternalMailSeedRelativePath, strings.Repeat("e", 32))
	if _, err := ComprobarSeparacion(Proceso{Material: interno}, Proceso{Material: externo}); err != nil {
		t.Fatalf("semillas separadas rechazadas: %v", err)
	}
	escribir(t, externo, config.DevelopmentExternalMailSeedRelativePath, strings.Repeat("i", 32))
	_, err := ComprobarSeparacion(Proceso{Material: interno}, Proceso{Material: externo})
	if got := motivo(t, err); got != config.DevelopmentKMSSecretRelativePath+" = "+config.DevelopmentExternalMailSeedRelativePath {
		t.Fatalf("la copia del KMS interno no se identifica por su contenido: %s", got)
	}
}

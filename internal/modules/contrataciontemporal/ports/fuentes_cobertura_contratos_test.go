package ports

import (
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"testing"
)

func identidadCoberturaContratoPrueba(
	t *testing.T,
	autoridad string,
	backend string,
	rol RolAutoridadFuenteAnalisis,
	semillaTexto string,
) IdentidadAutoridadFuenteAnalisis {
	t.Helper()
	semilla := sha256.Sum256([]byte(semillaTexto))
	privada := ed25519.NewKeyFromSeed(semilla[:])
	identidad, err := NuevaIdentidadAutoridadFuenteAnalisis(
		autoridad,
		backend,
		privada.Public().(ed25519.PublicKey),
		rol,
	)
	if err != nil {
		t.Fatal(err)
	}
	return identidad
}

func TestIdentidadAutoridadCoberturaEsInmutableYRedactada(t *testing.T) {
	identidad := identidadCoberturaContratoPrueba(
		t,
		"autoridad_fuente_cobertura_012345",
		"backend_fuente_cobertura_012345",
		RolFuenteCobertura,
		"fuente",
	)
	clave := identidad.ClavePruebaEd25519()
	original := clave[0]
	clave[0] ^= 0xff
	if identidad.ClavePruebaEd25519()[0] != original {
		t.Fatal("la proyeccion de clave permite mutar la identidad")
	}
	const redaccion = "[IDENTIDAD-AUTORIDAD-FUENTE-ANALISIS-REDACTADA]"
	if identidad.String() != redaccion ||
		fmt.Sprintf("%+v", identidad) != redaccion ||
		identidad.LogValue().Kind() != slog.KindString ||
		identidad.LogValue().String() != redaccion {
		t.Fatal("la identidad no se formatea de forma redactada")
	}
}

func TestAutoridadesCoberturaDebenSepararFunciones(t *testing.T) {
	fuente := identidadCoberturaContratoPrueba(
		t,
		"autoridad_fuente_cobertura_012345",
		"backend_fuente_cobertura_012345",
		RolFuenteCobertura,
		"fuente",
	)
	verificador := identidadCoberturaContratoPrueba(
		t,
		"autoridad_verificador_cobertura_01",
		"backend_verificador_cobertura_01",
		RolVerificadorCobertura,
		"verificador",
	)
	publicador := identidadCoberturaContratoPrueba(
		t,
		"autoridad_publicador_cobertura_01",
		"backend_publicador_cobertura_01",
		RolPublicadorCatalogoCobertura,
		"publicador",
	)
	if !AutoridadesFuenteAnalisisSeparadas(
		fuente,
		verificador,
		publicador,
	) {
		t.Fatal("tres autoridades independientes deben aceptarse")
	}
	suplantada := identidadCoberturaContratoPrueba(
		t,
		fuente.AutoridadRef(),
		fuente.BackendRef(),
		RolVerificadorCobertura,
		"fuente",
	)
	if AutoridadesFuenteAnalisisSeparadas(
		fuente,
		suplantada,
		publicador,
	) {
		t.Fatal("la reutilizacion de autoridad debe rechazarse")
	}
}

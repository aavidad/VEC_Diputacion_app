package bootstrap

import (
	"context"
	"strings"
	"testing"

	postgresidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
)

func TestSeudonimosPresentacionCertificadoSeparaPropositosYSostieneEpoch(t *testing.T) {
	s := &seudonimizadorSesionDesarrollo{derivador: nuevoDerivadorIdempotenciaPrueba(t, 2, 1)}
	ids := postgresidentidad.IdentificadoresPresentacionCertificado{
		EspacioIdentidad: espacioIdentidadSesionDesarrollo,
		AsercionID:       "aser-actual", SesionIDAfirmada: "sesion-actual",
		SujetoID: "persona-actual", CuentaID: "cuenta-actual",
		CertificadoSHA256: "sha256:" + strings.Repeat("a", 64),
		CASHA256:          "sha256:" + strings.Repeat("b", 64), NonceID: "aser-actual",
	}
	a, err := s.SeudonimizarPresentacionCertificado(context.Background(), ids)
	if err != nil || a.ClaveVersion != 2 || a.ClaveID != "vec.identidad.desarrollo.g2" {
		t.Fatalf("epoch y coordenadas no fijados: %v", err)
	}
	vistos := map[[32]byte]bool{}
	for _, huella := range [][32]byte{a.AsercionIDHMAC, a.SesionIDHMAC, a.SujetoIDHMAC, a.CuentaIDHMAC, a.CertificadoDERHMAC, a.CAHMAC, a.NonceHMAC} {
		if huella == [32]byte{} || vistos[huella] {
			t.Fatal("propósitos HMAC colisionaron")
		}
		vistos[huella] = true
	}
	ids.NonceID = "otro-nonce"
	b, err := s.SeudonimizarPresentacionCertificado(context.Background(), ids)
	if err != nil || a.CertificadoDERHMAC != b.CertificadoDERHMAC || a.NonceHMAC == b.NonceHMAC {
		t.Fatalf("identidad de certificado o nonce inestable: %v", err)
	}
}

func TestSeudonimosPresentacionCertificadoEmiteSoloEpochActivo(t *testing.T) {
	s := &seudonimizadorSesionDesarrollo{derivador: nuevoDerivadorIdempotenciaPrueba(t, 2, 1)}
	ids := postgresidentidad.IdentificadoresPresentacionCertificado{
		EspacioIdentidad: espacioIdentidadSesionDesarrollo,
		AsercionID:       "a", SesionIDAfirmada: "s", SujetoID: "p", CuentaID: "c",
		CertificadoSHA256: "sha256:" + strings.Repeat("a", 64),
		CASHA256:          "sha256:" + strings.Repeat("b", 64), NonceID: "n",
	}
	primera, err := s.SeudonimizarPresentacionCertificado(context.Background(), ids)
	if err != nil || primera.ClaveVersion != 2 {
		t.Fatalf("epoch activo no identificado: %v", err)
	}
	rotado := &seudonimizadorSesionDesarrollo{derivador: nuevoDerivadorIdempotenciaPrueba(t, 3, 2)}
	segunda, err := rotado.SeudonimizarPresentacionCertificado(context.Background(), ids)
	if err != nil || segunda.ClaveVersion != 3 || primera.CertificadoDERHMAC == segunda.CertificadoDERHMAC {
		t.Fatalf("rotacion de clave se confundio con epoch anterior: %v", err)
	}
}

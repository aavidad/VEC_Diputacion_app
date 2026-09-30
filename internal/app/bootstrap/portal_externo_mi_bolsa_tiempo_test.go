package bootstrap

import (
	"crypto/x509"
	"testing"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"
)

func TestMiBolsaExteriorCompruebaVigenciaDespuesDeResolver(t *testing.T) {
	ahora := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora,
		"per_candidato_tiempo_1234567890123456", "prf_candidato_tiempo_1234567890123456", core.AuthMethodCertificate, core.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	entrada := resultado.ResueltoEnAutoritativo.Add(-time.Microsecond)
	salida := resultado.ResueltoEnAutoritativo.Add(time.Microsecond)
	certificado := &x509.Certificate{NotBefore: entrada.Add(-time.Minute), NotAfter: salida.Add(time.Hour)}
	identidad := &identidadCandidatoBolsaDesarrollo{verificadoEn: entrada.Add(-time.Minute), validoHasta: salida.Add(time.Hour)}
	if vigenciaMiBolsaPortalExterno(certificado, identidad, vinculo, resultado, entrada) {
		t.Fatal("aceptó el vínculo antes de su resolución autoritativa")
	}
	if !vigenciaMiBolsaPortalExterno(certificado, identidad, vinculo, resultado, salida) {
		t.Fatal("denegó el vínculo vigente al terminar la resolución")
	}
}

func TestMiBolsaExteriorDeniegaCaducidadDuranteResolucion(t *testing.T) {
	ahora := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora,
		"per_candidato_tiempo_1234567890123456", "prf_candidato_tiempo_1234567890123456", core.AuthMethodCertificate, core.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	entrada := resultado.ResueltoEnAutoritativo.Add(-time.Microsecond)
	salida := resultado.ResueltoEnAutoritativo.Add(time.Microsecond)
	for _, caso := range []string{"certificado", "identidad", "sesion"} {
		t.Run(caso, func(t *testing.T) {
			certificado := &x509.Certificate{NotBefore: entrada.Add(-time.Hour), NotAfter: ahora.Add(time.Hour)}
			identidad := &identidadCandidatoBolsaDesarrollo{verificadoEn: entrada.Add(-time.Hour), validoHasta: ahora.Add(time.Hour)}
			comprobadaEn := salida
			switch caso {
			case "certificado":
				certificado.NotAfter = resultado.ResueltoEnAutoritativo
				if !entrada.Before(certificado.NotAfter) {
					t.Fatal("el certificado ya estaba caducado en la entrada")
				}
			case "identidad":
				identidad.validoHasta = resultado.ResueltoEnAutoritativo
				if !entrada.Before(identidad.validoHasta) {
					t.Fatal("la identidad ya estaba caducada en la entrada")
				}
			case "sesion":
				datos, err := vinculo.Datos()
				if err != nil {
					t.Fatal(err)
				}
				comprobadaEn = datos.SesionValidaHasta
			}
			if vigenciaMiBolsaPortalExterno(certificado, identidad, vinculo, resultado, comprobadaEn) {
				t.Fatal("aceptó una credencial caducada al terminar la resolución")
			}
		})
	}
}

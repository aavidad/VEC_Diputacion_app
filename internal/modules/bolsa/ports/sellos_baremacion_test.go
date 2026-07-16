package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
)

func TestMaterialCanonicoHMACSelloBaremacionCongelaVerificacionHistoricaV2(t *testing.T) {
	representacion, err := NuevaCargaProtegida([]byte("representacion-canonica-vector-v1"))
	if err != nil {
		t.Fatal(err)
	}
	verificacion := SolicitudVerificarSelloBaremacion{
		Finalidad:              FinalidadSelloManifiestoProbatorioBaremacionV2,
		RepresentacionCanonica: representacion,
		SelloHMAC:              "hmac-sha256:vector_1:" + huellaPruebaPuertos("a"),
	}
	material, err := verificacion.MaterialCanonicoHMAC()
	if err != nil {
		t.Fatal(err)
	}
	esperado := []byte("manifiesto_probatorio_baremacion_v2\x00representacion-canonica-vector-v1")
	if !bytes.Equal(material.Revelar(), esperado) {
		t.Fatalf("preimagen contractual alterada: %x", material.Revelar())
	}
	suma := sha256.Sum256(material.Revelar())
	const vectorSHA256 = "c608fb5a3b1981835edfd3539c8d74c0080f2aecc88fd30dc39481b307211b20"
	if obtenida := hex.EncodeToString(suma[:]); obtenida != vectorSHA256 {
		t.Fatalf("vector de preimagen alterado: obtenido=%s esperado=%s", obtenida, vectorSHA256)
	}

	if _, err := (SolicitudSellarSelloBaremacion{
		Finalidad:              FinalidadSelloManifiestoProbatorioBaremacionV2,
		RepresentacionCanonica: representacion,
	}).MaterialCanonicoHMAC(); !errors.Is(err, ErrSolicitudBaremacionInvalida) {
		t.Fatalf("el productor admitio la finalidad historica V2: %v", err)
	}
}

func TestSelladoVigenteYVerificacionCompartenPreimagenSinDowngrade(t *testing.T) {
	representacion, err := NuevaCargaProtegida([]byte("representacion-canonica-vigente"))
	if err != nil {
		t.Fatal(err)
	}
	for _, finalidad := range []FinalidadSelloBaremacion{
		FinalidadSelloReservaBaremacion,
		FinalidadSelloConfirmacionBaremacionV2,
		FinalidadSelloSobreProbatorioConfirmacionBaremacionV3,
		FinalidadSelloManifiestoProbatorioBaremacionV3,
	} {
		productor := SolicitudSellarSelloBaremacion{
			Finalidad: finalidad, RepresentacionCanonica: representacion,
		}
		materialProductor, err := productor.MaterialCanonicoHMAC()
		if err != nil {
			t.Fatalf("finalidad vigente %s rechazada: %v", finalidad, err)
		}
		verificador := SolicitudVerificarSelloBaremacion{
			Finalidad: finalidad, RepresentacionCanonica: representacion,
			SelloHMAC: "hmac-sha256:vigente_1:" + huellaPruebaPuertos("b"),
		}
		materialVerificador, err := verificador.MaterialCanonicoHMAC()
		if err != nil || !bytes.Equal(materialProductor.Revelar(), materialVerificador.Revelar()) {
			t.Fatalf("preimagen vigente %s divergente: %v", finalidad, err)
		}
	}
	for _, finalidad := range []FinalidadSelloBaremacion{
		FinalidadSelloConfirmacionBaremacion,
		FinalidadSelloSobreProbatorioConfirmacionBaremacionV2,
		FinalidadSelloManifiestoProbatorioBaremacionV2,
	} {
		if err := (SolicitudSellarSelloBaremacion{
			Finalidad: finalidad, RepresentacionCanonica: representacion,
		}).Validar(); !errors.Is(err, ErrSolicitudBaremacionInvalida) {
			t.Fatalf("finalidad retirada %s admitida para sellado: %v", finalidad, err)
		}
		if err := (SolicitudVerificarSelloBaremacion{
			Finalidad: finalidad, RepresentacionCanonica: representacion,
			SelloHMAC: "hmac-sha256:historica_1:" + huellaPruebaPuertos("c"),
		}).Validar(); err != nil {
			t.Fatalf("finalidad historica %s no verificable: %v", finalidad, err)
		}
	}
}

func TestMaterialCanonicoHMACSelloBaremacionRechazaContratoIncompleto(t *testing.T) {
	representacion, err := NuevaCargaProtegida([]byte("representacion-canonica"))
	if err != nil {
		t.Fatal(err)
	}
	for _, solicitud := range []SolicitudSellarSelloBaremacion{
		{RepresentacionCanonica: representacion},
		{Finalidad: FinalidadSelloReservaBaremacion},
		{Finalidad: "finalidad_ajena", RepresentacionCanonica: representacion},
	} {
		if _, err := solicitud.MaterialCanonicoHMAC(); !errors.Is(err, ErrSolicitudBaremacionInvalida) {
			t.Fatalf("contrato incompleto admitido: %+v / %v", solicitud, err)
		}
	}
}

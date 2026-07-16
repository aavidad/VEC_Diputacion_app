package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
)

func TestMaterialCanonicoHMACSelloBaremacionCongelaFormulaContractual(t *testing.T) {
	representacion, err := NuevaCargaProtegida([]byte("representacion-canonica-vector-v1"))
	if err != nil {
		t.Fatal(err)
	}
	solicitud := SolicitudSellarSelloBaremacion{
		Finalidad:              FinalidadSelloManifiestoProbatorioBaremacionV2,
		RepresentacionCanonica: representacion,
	}
	material, err := solicitud.MaterialCanonicoHMAC()
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

	verificacion := SolicitudVerificarSelloBaremacion{
		Finalidad:              solicitud.Finalidad,
		RepresentacionCanonica: solicitud.RepresentacionCanonica,
		SelloHMAC:              "hmac-sha256:vector_1:" + huellaPruebaPuertos("a"),
	}
	materialVerificacion, err := verificacion.MaterialCanonicoHMAC()
	if err != nil || !bytes.Equal(material.Revelar(), materialVerificacion.Revelar()) {
		t.Fatalf("productor y verificador divergen: %v", err)
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

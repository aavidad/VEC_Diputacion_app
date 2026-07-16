package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestSobreNominalV3NoMutaElDominioLegadoV2(t *testing.T) {
	legado := intentoNominalConfirmacionV2ValidoPrueba(t, 0x61)
	vigente := IntentoNominalConfirmacionBaremacionV3{
		IdentificadorOperacion: legado.IdentificadorOperacion,
		Confirmacion:           legado.Confirmacion,
	}
	canonicoV2, err := RepresentacionCanonicaSobreProbatorioConfirmacionBaremacionV2(legado)
	if err != nil {
		t.Fatal(err)
	}
	canonicoV3, err := RepresentacionCanonicaSobreProbatorioConfirmacionBaremacionV3(vigente)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(canonicoV2.Revelar(), canonicoV3.Revelar()) {
		t.Fatal("V3 descendio al dominio nominal V2 retirado")
	}
	suma := sha256.Sum256(canonicoV3.Revelar())
	const vectorV3 = "d102035c537a8a6a96e8d02c853334c86296a006a1e4a319810ec48ed28ffb73"
	if obtenida := hex.EncodeToString(suma[:]); obtenida != vectorV3 {
		t.Fatalf("vector nominal V3 inesperado: %s", obtenida)
	}
	peticion := SolicitudVerificarSelloBaremacion{
		Finalidad:              FinalidadSelloSobreProbatorioConfirmacionBaremacionV3,
		RepresentacionCanonica: canonicoV3,
		SelloHMAC:              vigente.Confirmacion.HuellaSolicitudHMAC,
	}
	if err := peticion.Validar(); err != nil {
		t.Fatalf("dominio nominal V3 no verificable: %v", err)
	}
}

func TestIntentoNominalV3ClonaSinCompartirAgregado(t *testing.T) {
	legado := intentoNominalConfirmacionV2ValidoPrueba(t, 0x62)
	vigente := IntentoNominalConfirmacionBaremacionV3{
		IdentificadorOperacion: legado.IdentificadorOperacion,
		Confirmacion:           legado.Confirmacion,
	}
	clon, err := vigente.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	clon.Confirmacion.Agregado.EvidenciasIniciales[0].Referencia.DocumentoRef = "alterada"
	if vigente.Confirmacion.Agregado.EvidenciasIniciales[0].Referencia.DocumentoRef == "alterada" {
		t.Fatal("el sobre V3 comparte el agregado mutable")
	}
}

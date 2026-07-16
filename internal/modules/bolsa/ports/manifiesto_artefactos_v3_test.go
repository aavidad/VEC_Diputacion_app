package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
)

func TestArtefactosCanonicosManifiestoProbatorioV3SonExactosYDefensivos(t *testing.T) {
	manifiesto := manifiestoProbatorioValidoPrueba(t, contenidoDecisionValidoPrueba(t))
	artefactos, err := ArtefactosCanonicosManifiestoProbatorioBaremacion(manifiesto)
	if err != nil {
		t.Fatal(err)
	}
	contenido := artefactos.ContenidoSinHuella.Revelar()
	sumaContenido := sha256.Sum256(contenido)
	if obtenida := hex.EncodeToString(sumaContenido[:]); obtenida != manifiesto.HuellaManifiestoSHA256 {
		t.Fatalf("la preimagen SHA no reproduce la huella: %s", obtenida)
	}
	representacion, err := RepresentacionCanonicaManifiestoProbatorioBaremacion(manifiesto)
	if err != nil || !bytes.Equal(artefactos.RepresentacionSellada.Revelar(), representacion.Revelar()) {
		t.Fatalf("representacion durable divergente: %v", err)
	}
	preimagen, err := (SolicitudSellarSelloBaremacion{
		Finalidad:              FinalidadSelloManifiestoProbatorioBaremacionV3,
		RepresentacionCanonica: representacion,
	}).MaterialCanonicoHMAC()
	if err != nil || !bytes.Equal(artefactos.PreimagenHMAC.Revelar(), preimagen.Revelar()) {
		t.Fatalf("preimagen HMAC durable divergente: %v", err)
	}
	for _, vector := range []struct {
		nombre   string
		carga    CargaProtegida
		huella   string
		longitud int
	}{
		{"contenido", artefactos.ContenidoSinHuella, "6d4ae3b15120684ae1600d4462a83b4f7f3141921a52c445999e1c798c505d57", 6270},
		{"representacion", artefactos.RepresentacionSellada, "c8fb92a5f3b3dae996fed88b507fb1cd454d877ce08698e84edba1d50e3608af", 6393},
		{"preimagen", artefactos.PreimagenHMAC, "522a5e709f22a722314ad9e76d4a4cf362fba0562a91f650e91353f0abaf3d15", 6429},
	} {
		valor := vector.carga.Revelar()
		suma := sha256.Sum256(valor)
		if obtenida := hex.EncodeToString(suma[:]); obtenida != vector.huella || len(valor) != vector.longitud {
			t.Fatalf("vector %s alterado: sha256=%s longitud=%d", vector.nombre, obtenida, len(valor))
		}
	}

	primeraCopia := artefactos.ContenidoSinHuella.Revelar()
	segundaCopia := artefactos.ContenidoSinHuella.Revelar()
	primeraCopia[0] ^= 0xff
	if bytes.Equal(primeraCopia, segundaCopia) || !bytes.Equal(segundaCopia, artefactos.ContenidoSinHuella.Revelar()) {
		t.Fatal("CargaProtegida no entrego copias defensivas independientes")
	}
}

func TestArtefactosCanonicosManifiestoProbatorioV3RechazanContenidoAdulterado(t *testing.T) {
	manifiesto := manifiestoProbatorioValidoPrueba(t, contenidoDecisionValidoPrueba(t))
	manifiesto.Autorizaciones[0].RecursoRef += ":adulterado"
	if _, err := ArtefactosCanonicosManifiestoProbatorioBaremacion(manifiesto); !errors.Is(err, ErrSolicitudBaremacionInvalida) {
		t.Fatalf("manifiesto adulterado admitido: %v", err)
	}
}

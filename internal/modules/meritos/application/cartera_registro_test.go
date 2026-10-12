package application

import (
	"regexp"
	"strings"
	"testing"
)

func TestCarteraSeparaReferenciasDeLecturaEIntentoComun(t *testing.T) {
	lectura, err := identificadorRegistroCartera(false)
	if err != nil || !regexp.MustCompile(`^lectura_[0-9a-f]{32}$`).MatchString(lectura) {
		t.Fatalf("lectura positiva: %q / %v", lectura, err)
	}
	intento, err := identificadorRegistroCartera(true)
	if err != nil || !regexp.MustCompile(`^intento_[0-9a-f]{32}$`).MatchString(intento) {
		t.Fatalf("intento nominal común: %q / %v", intento, err)
	}
	if lectura == intento {
		t.Fatal("lectura e intento comparten referencia")
	}
}

func TestRetiradaCarteraMantieneAmbitoPersonalYLigaVersionAlContexto(t *testing.T) {
	q := SolicitudRetiradaCartera{HechoRef: "hecho:propio", HechoVersionEsperada: 1}
	persona, huella := "per_aaaaaaaaaaaaaaaaaaaaaa", strings.Repeat("a", 64)
	recurso := recursoRetiradaCartera(q, persona, huella)
	if recurso.Validar() != nil || len(recurso.Ambitos) != 1 || recurso.Ambitos["persona_ref"] != persona ||
		len(recurso.Atributos) != 4 || recurso.Atributos["hecho_version_esperada"] != "1" ||
		recurso.Atributos["disponibilidad_version_esperada"] != "0" ||
		recurso.Atributos["huella_comando_sha256"] != huella {
		t.Fatalf("ámbito o atributos incompatibles: %#v", recurso)
	}
	original, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	q.HechoVersionEsperada = 2
	otraVersion, err := recursoRetiradaCartera(q, persona, huella).HuellaContextoAutorizacionSHA256()
	if err != nil || otraVersion == original {
		t.Fatalf("versión sin vínculo criptográfico: %q %q %v", original, otraVersion, err)
	}
	q.HechoVersionEsperada = 1
	otraHuella, err := recursoRetiradaCartera(q, persona, strings.Repeat("b", 64)).HuellaContextoAutorizacionSHA256()
	if err != nil || otraHuella == original {
		t.Fatalf("comando sin vínculo criptográfico: %q %q %v", original, otraHuella, err)
	}
}

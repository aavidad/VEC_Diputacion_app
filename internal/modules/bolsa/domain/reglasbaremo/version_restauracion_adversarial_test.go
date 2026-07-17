package reglasbaremo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestRestaurarVersionGobernadaRechazaReferenciaContenidoCorrupta(t *testing.T) {
	borrador := nuevaVersionGobiernoPrueba(t, instanteBaseGobiernoPrueba)
	canonico, _ := borrador.RepresentacionCanonica()
	referencia, _ := borrador.ReferenciaContenido()
	alterado := sustituirUnaVez(
		t, canonico,
		`"huella_sha256":"`+referencia.HuellaSHA256()+`"`,
		`"huella_sha256":"`+huellaDistintaGobierno(referencia.HuellaSHA256())+`"`,
	)
	_, err := RestaurarVersionGobernadaReglasBaremo(alterado)
	if !errors.Is(err, ErrGobiernoVinculoInexacto) {
		t.Fatalf("referencia de contenido corrupta aceptada: %v", err)
	}
}

func TestRestaurarVersionGobernadaRechazaVinculosDeEvidenciaCorruptos(t *testing.T) {
	publicada := versionesGobernadasRestauracionPrueba(t)["publicada"]
	canonico, _ := publicada.RepresentacionCanonica()
	material := publicada.publicacion.aprobacion.vinculo
	alterado := sustituirUnaVez(
		t, canonico,
		`"huella_estado_sha256":"`+material.huellaEstadoSHA256+`"`,
		`"huella_estado_sha256":"`+huellaDistintaGobierno(material.huellaEstadoSHA256)+`"`,
	)
	_, err := RestaurarVersionGobernadaReglasBaremo(alterado)
	if !errors.Is(err, ErrGobiernoEvidenciaInvalida) {
		t.Fatalf("vinculo de aprobacion corrupto aceptado: %v", err)
	}
}

func TestRestaurarVersionGobernadaRechazaOrdenNoCanonicoDeEvidencias(t *testing.T) {
	versiones := versionesGobernadasRestauracionPrueba(t)
	publicada, _ := versiones["publicada"].RepresentacionCanonica()
	firmantesInvertidos := bytes.Replace(
		publicada,
		[]byte(`"firmantes":["per_11111111111111111111111111111111","per_22222222222222222222222222222222"]`),
		[]byte(`"firmantes":["per_22222222222222222222222222222222","per_11111111111111111111111111111111"]`),
		1,
	)
	if bytes.Equal(firmantesInvertidos, publicada) {
		t.Fatal("no se localizaron los firmantes canonicos")
	}
	_, err := RestaurarVersionGobernadaReglasBaremo(firmantesInvertidos)
	if !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("orden no canonico de firmantes aceptado: %v", err)
	}

	activa := versiones["activa"]
	canonico, _ := activa.RepresentacionCanonica()
	dependencias := activa.activacion.dependencias.dependencias
	if len(dependencias) < 2 {
		t.Fatal("la prueba necesita dos dependencias")
	}
	primera, _ := json.Marshal(materialReferenciaGobierno(dependencias[0]))
	segunda, _ := json.Marshal(materialReferenciaGobierno(dependencias[1]))
	fragmento := append(append(append([]byte(nil), primera...), ','), segunda...)
	invertido := append(append(append([]byte(nil), segunda...), ','), primera...)
	alterado := bytes.Replace(canonico, fragmento, invertido, 1)
	if bytes.Equal(alterado, canonico) {
		t.Fatal("no se localizaron las dependencias canonicas")
	}
	_, err = RestaurarVersionGobernadaReglasBaremo(alterado)
	if !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("orden no canonico de dependencias aceptado: %v", err)
	}
}

func TestRestaurarVersionGobernadaRechazaCorrupcionDeTransicion(t *testing.T) {
	sustituida := versionesGobernadasRestauracionPrueba(t)["sustituida"]
	canonico, _ := sustituida.RepresentacionCanonica()
	alterado := sustituirUnaVez(
		t, canonico, `"estado":"sustituida"`, `"estado":"retirada"`,
	)
	_, err := RestaurarVersionGobernadaReglasBaremo(alterado)
	if !errors.Is(err, ErrGobiernoInvarianteQuebrada) {
		t.Fatalf("transicion incompatible aceptada: %v", err)
	}
}

func TestRestaurarVersionGobernadaLimitaColeccionesDeEvidencias(t *testing.T) {
	versiones := versionesGobernadasRestauracionPrueba(t)
	publicada, _ := versiones["publicada"].RepresentacionCanonica()
	firmantes := make([]string, maximoFirmantesAprobacionReglasBaremo+1)
	for indice := range firmantes {
		firmantes[indice] = fmt.Sprintf("per_%032x", indice)
	}
	firmantesJSON, _ := json.Marshal(firmantes)
	sobredimensionada := bytes.Replace(
		publicada,
		[]byte(`["per_11111111111111111111111111111111","per_22222222222222222222222222222222"]`),
		firmantesJSON,
		1,
	)
	_, err := RestaurarVersionGobernadaReglasBaremo(sobredimensionada)
	if !errors.Is(err, ErrGobiernoEvidenciaInvalida) {
		t.Fatalf("firmantes sobredimensionados aceptados: %v", err)
	}

	activa := versiones["activa"]
	canonico, _ := activa.RepresentacionCanonica()
	referencia := materialReferenciaGobierno(activa.activacion.dependencias.dependencias[0])
	referenciaJSON, _ := json.Marshal(referencia)
	listaExcesiva := `[` + strings.Repeat(
		string(referenciaJSON)+`,`, maximoDependenciasReglasBaremo,
	) + string(referenciaJSON) + `]`
	materialOriginal, _ := json.Marshal(
		materialesReferenciasGobierno(activa.activacion.dependencias.dependencias),
	)
	sobredimensionada = bytes.Replace(canonico, materialOriginal, []byte(listaExcesiva), 1)
	if bytes.Equal(sobredimensionada, canonico) {
		t.Fatal("no se localizo la coleccion de dependencias")
	}
	_, err = RestaurarVersionGobernadaReglasBaremo(sobredimensionada)
	if !errors.Is(err, ErrGobiernoEvidenciaInvalida) {
		t.Fatalf("dependencias sobredimensionadas aceptadas: %v", err)
	}
}

func huellaDistintaGobierno(original string) string {
	if original[0] == '0' {
		return "1" + original[1:]
	}
	return "0" + original[1:]
}

func jsonIndentacionGobierno(destino *bytes.Buffer, origen []byte) error {
	return json.Indent(destino, origen, "", "  ")
}

func materialesReferenciasGobierno(
	referencias []ReferenciaVersionada,
) []materialReferenciaGobiernoReglas {
	resultado := make([]materialReferenciaGobiernoReglas, len(referencias))
	for indice := range referencias {
		resultado[indice] = materialReferenciaGobierno(referencias[indice])
	}
	return resultado
}

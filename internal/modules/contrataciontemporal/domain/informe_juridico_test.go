package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestInformeJuridicoConstruyeCanonRestauraYVerificaHuella(t *testing.T) {
	datos := datosInformeJuridicoPrueba()
	borrador, err := NuevoBorradorInformeJuridico(datos)
	if err != nil {
		t.Fatalf("construir borrador: %v", err)
	}
	material, err := borrador.SerializarCanonico()
	if err != nil {
		t.Fatalf("serializar borrador: %v", err)
	}
	suma := sha256.Sum256(material)
	huella := hex.EncodeToString(suma[:])
	const huellaEsperada = "bfd05972f66408600f6d5a1dbc49a55c35a2cb7eee0595cf1cf12a1bfcc3d30f"
	if huella != huellaEsperada || huella != borrador.HuellaSHA256() ||
		!borrador.VerificarHuellaSHA256(huella) {
		t.Fatal("la huella no es el SHA-256 exacto del canon")
	}
	if borrador.VerificarHuellaSHA256(strings.Repeat("f", 64)) {
		t.Fatal("se acepto una huella distinta")
	}

	codificado, err := json.Marshal(borrador.Estado())
	if err != nil {
		t.Fatalf("serializar estado: %v", err)
	}
	var estado EstadoBorradorInformeJuridico
	if err := json.Unmarshal(codificado, &estado); err != nil {
		t.Fatalf("restaurar estado JSON: %v", err)
	}
	restaurado, err := RestaurarBorradorInformeJuridico(estado)
	if err != nil {
		t.Fatalf("verificar y restaurar borrador: %v", err)
	}
	materialRestaurado, _ := restaurado.SerializarCanonico()
	if !reflect.DeepEqual(material, materialRestaurado) {
		t.Fatal("la restauracion altero el canon")
	}
}

func TestInformeJuridicoCanonOrdenadoYCopiasDefensivas(t *testing.T) {
	datos := datosInformeJuridicoPrueba()
	datos.ReferenciasNormativas[0], datos.ReferenciasNormativas[1] =
		datos.ReferenciasNormativas[1], datos.ReferenciasNormativas[0]
	datos.Anexos[0], datos.Anexos[1] = datos.Anexos[1], datos.Anexos[0]
	borrador, err := NuevoBorradorInformeJuridico(datos)
	if err != nil {
		t.Fatalf("construir borrador desordenado: %v", err)
	}
	datos.ReferenciasNormativas[0].NormaRef = "norma:alterada"
	datos.Anexos[0].DocumentoRef = "documento:alterado"

	estado := borrador.Estado()
	if estado.ReferenciasNormativas[0].NormaRef != "norma:empleo-publico:trebep" ||
		estado.Anexos[0].DocumentoRef != "documento:analisis-rrhh:01" {
		t.Fatal("el constructor no ordeno o compartio las entradas")
	}
	estado.ReferenciasNormativas[0].NormaRef = "norma:otra"
	estado.Anexos[0].DocumentoRef = "documento:otro"
	materialAntes, _ := borrador.SerializarCanonico()
	materialAntes[0] ^= 0xff
	materialDespues, _ := borrador.SerializarCanonico()
	if borrador.Estado().ReferenciasNormativas[0].NormaRef != "norma:empleo-publico:trebep" ||
		borrador.Estado().Anexos[0].DocumentoRef != "documento:analisis-rrhh:01" ||
		materialAntes[0] == materialDespues[0] {
		t.Fatal("una salida comparte memoria con el borrador")
	}

	ordenado, err := NuevoBorradorInformeJuridico(datosInformeJuridicoPrueba())
	if err != nil {
		t.Fatalf("construir borrador ordenado: %v", err)
	}
	canonOrdenado, _ := ordenado.SerializarCanonico()
	if !reflect.DeepEqual(materialDespues, canonOrdenado) ||
		borrador.HuellaSHA256() != ordenado.HuellaSHA256() {
		t.Fatal("el orden de entrada cambio el canon")
	}
}

func TestInformeJuridicoRechazaVersionesReferenciasHuellasYDuplicados(t *testing.T) {
	casos := map[string]func(*DatosBorradorInformeJuridico){
		"esquema":            func(d *DatosBorradorInformeJuridico) { d.Canon.Esquema = "otro" },
		"version esquema":    func(d *DatosBorradorInformeJuridico) { d.Canon.VersionEsquema++ },
		"expediente":         func(d *DatosBorradorInformeJuridico) { d.ExpedienteRef = "dato personal" },
		"version expediente": func(d *DatosBorradorInformeJuridico) { d.VersionEsperadaExpediente = 0 },
		"plantilla":          func(d *DatosBorradorInformeJuridico) { d.Plantilla.PlantillaRef = "" },
		"version plantilla":  func(d *DatosBorradorInformeJuridico) { d.Plantilla.Version = 0 },
		"huella plantilla":   func(d *DatosBorradorInformeJuridico) { d.Plantilla.HuellaSHA256 = strings.Repeat("0", 64) },
		"sin normativa":      func(d *DatosBorradorInformeJuridico) { d.ReferenciasNormativas = nil },
		"version normativa":  func(d *DatosBorradorInformeJuridico) { d.ReferenciasNormativas[0].Version = 0 },
		"huella normativa":   func(d *DatosBorradorInformeJuridico) { d.ReferenciasNormativas[0].HuellaSHA256 = "AA" },
		"normativa duplicada": func(d *DatosBorradorInformeJuridico) {
			d.ReferenciasNormativas[1].NormaRef = d.ReferenciasNormativas[0].NormaRef
		},
		"anexo":           func(d *DatosBorradorInformeJuridico) { d.Anexos[0].DocumentoRef = "x" },
		"huella anexo":    func(d *DatosBorradorInformeJuridico) { d.Anexos[0].HuellaSHA256 = strings.Repeat("0", 64) },
		"anexo duplicado": func(d *DatosBorradorInformeJuridico) { d.Anexos[1].DocumentoRef = d.Anexos[0].DocumentoRef },
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			datos := datosInformeJuridicoPrueba()
			alterar(&datos)
			if _, err := NuevoBorradorInformeJuridico(datos); !errors.Is(err, ErrBorradorInformeJuridicoInvalido) {
				t.Fatalf("se acepto el caso invalido: %v", err)
			}
		})
	}
}

func TestInformeJuridicoValidaLimitesAntesDeCopiar(t *testing.T) {
	datos := datosInformeJuridicoPrueba()
	datos.ReferenciasNormativas = make(
		[]ReferenciaNormativaInformeJuridico,
		maximoReferenciasNormativasInformeJuridico+1,
	)
	if _, err := NuevoBorradorInformeJuridico(datos); !errors.Is(err, ErrBorradorInformeJuridicoInvalido) {
		t.Fatalf("se acepto cardinalidad normativa excesiva: %v", err)
	}
	datos = datosInformeJuridicoPrueba()
	datos.Anexos = make([]AnexoDocumentalInformeJuridico, maximoAnexosInformeJuridico+1)
	if _, err := NuevoBorradorInformeJuridico(datos); !errors.Is(err, ErrBorradorInformeJuridicoInvalido) {
		t.Fatalf("se acepto cardinalidad de anexos excesiva: %v", err)
	}
	datos = datosInformeJuridicoPrueba()
	datos.VersionEsperadaExpediente = maximoEnteroSeguroInformeJuridico + 1
	if _, err := NuevoBorradorInformeJuridico(datos); !errors.Is(err, ErrBorradorInformeJuridicoInvalido) {
		t.Fatalf("se acepto una version no interoperable: %v", err)
	}
	datos = datosInformeJuridicoPrueba()
	datos.ReferenciasNormativas = make(
		[]ReferenciaNormativaInformeJuridico,
		maximoReferenciasNormativasInformeJuridico,
	)
	datos.Anexos = make([]AnexoDocumentalInformeJuridico, maximoAnexosInformeJuridico)
	for indice := range datos.ReferenciasNormativas {
		prefijo := fmt.Sprintf("norma:%02d:", indice)
		datos.ReferenciasNormativas[indice] = ReferenciaNormativaInformeJuridico{
			NormaRef:     prefijo + strings.Repeat("n", 159-len(prefijo)),
			Version:      1,
			HuellaSHA256: strings.Repeat("a", 64),
		}
		prefijo = fmt.Sprintf("anexo:%02d:", indice)
		datos.Anexos[indice] = AnexoDocumentalInformeJuridico{
			DocumentoRef: prefijo + strings.Repeat("d", 159-len(prefijo)),
			HuellaSHA256: strings.Repeat("b", 64),
		}
	}
	if _, err := NuevoBorradorInformeJuridico(datos); !errors.Is(err, ErrBorradorInformeJuridicoInvalido) {
		t.Fatalf("se acepto un tamano agregado excesivo: %v", err)
	}
}

func TestInformeJuridicoRechazaRestauracionAdulterada(t *testing.T) {
	borrador, err := NuevoBorradorInformeJuridico(datosInformeJuridicoPrueba())
	if err != nil {
		t.Fatalf("construir borrador: %v", err)
	}
	casos := map[string]func(*EstadoBorradorInformeJuridico){
		"expediente": func(e *EstadoBorradorInformeJuridico) { e.ExpedienteRef = "expediente:ct:alterado" },
		"plantilla":  func(e *EstadoBorradorInformeJuridico) { e.Plantilla.Version++ },
		"normativa":  func(e *EstadoBorradorInformeJuridico) { e.ReferenciasNormativas[0].Version++ },
		"anexo":      func(e *EstadoBorradorInformeJuridico) { e.Anexos[0].DocumentoRef = "documento:alterado:01" },
		"huella":     func(e *EstadoBorradorInformeJuridico) { e.HuellaSHA256 = strings.Repeat("f", 64) },
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			estado := borrador.Estado()
			alterar(&estado)
			if _, err := RestaurarBorradorInformeJuridico(estado); !errors.Is(err, ErrBorradorInformeJuridicoInvalido) {
				t.Fatalf("se restauro material adulterado: %v", err)
			}
		})
	}
}

func datosInformeJuridicoPrueba() DatosBorradorInformeJuridico {
	return DatosBorradorInformeJuridico{
		Canon:                     CanonBorradorInformeJuridicoV1(),
		ExpedienteRef:             "expediente:contratacion-temporal:01",
		VersionEsperadaExpediente: 7,
		Plantilla: ReferenciaPlantillaInformeJuridico{
			PlantillaRef: "plantilla:informe-juridico:rrhh",
			Version:      3,
			HuellaSHA256: strings.Repeat("a", 64),
		},
		ReferenciasNormativas: []ReferenciaNormativaInformeJuridico{
			{NormaRef: "norma:empleo-publico:trebep", Version: 5, HuellaSHA256: strings.Repeat("b", 64)},
			{NormaRef: "norma:procedimiento:ley-39-2015", Version: 2, HuellaSHA256: strings.Repeat("c", 64)},
		},
		Anexos: []AnexoDocumentalInformeJuridico{
			{DocumentoRef: "documento:analisis-rrhh:01", HuellaSHA256: strings.Repeat("d", 64)},
			{DocumentoRef: "documento:cobertura:01", HuellaSHA256: strings.Repeat("e", 64)},
		},
	}
}

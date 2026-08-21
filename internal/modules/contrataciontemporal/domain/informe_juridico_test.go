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
	const huellaEsperada = "4cb476f1a8b82c0622e41413d1675e4324e4f120d7815cc1eb1ab1346e70d7c6"
	if huella != huellaEsperada || huella != borrador.HuellaSHA256() ||
		!borrador.VerificarHuellaSHA256(huella) {
		t.Fatalf("la huella no es el SHA-256 exacto del canon: %s", huella)
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
	if estado.Anexos[0].VersionDocumento != datos.Anexos[0].VersionDocumento {
		t.Fatal("el snapshot no conservo la version documental")
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
	datos.Anexos[0].VersionDocumento = 1

	estado := borrador.Estado()
	if estado.ReferenciasNormativas[0].NormaRef != "norma:empleo-publico:trebep" ||
		estado.Anexos[0].DocumentoRef != "documento:analisis-rrhh:01" ||
		estado.Anexos[0].VersionDocumento != 11 {
		t.Fatal("el constructor no ordeno o compartio las entradas")
	}
	estado.ReferenciasNormativas[0].NormaRef = "norma:otra"
	estado.Anexos[0].DocumentoRef = "documento:otro"
	estado.Anexos[0].VersionDocumento = 2
	materialAntes, _ := borrador.SerializarCanonico()
	materialAntes[0] ^= 0xff
	materialDespues, _ := borrador.SerializarCanonico()
	if borrador.Estado().ReferenciasNormativas[0].NormaRef != "norma:empleo-publico:trebep" ||
		borrador.Estado().Anexos[0].DocumentoRef != "documento:analisis-rrhh:01" ||
		borrador.Estado().Anexos[0].VersionDocumento != 11 ||
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

	versiones := datosInformeJuridicoPrueba()
	versiones.Anexos = []AnexoDocumentalInformeJuridico{
		{DocumentoRef: "documento:versionado:01", VersionDocumento: maximoEnteroSeguroInformeJuridico, HuellaSHA256: strings.Repeat("d", 64)},
		{DocumentoRef: "documento:versionado:01", VersionDocumento: 1, HuellaSHA256: strings.Repeat("e", 64)},
	}
	versionado, err := NuevoBorradorInformeJuridico(versiones)
	if err != nil {
		t.Fatalf("construir anexos con versiones distintas: %v", err)
	}
	estadoVersionado := versionado.Estado()
	if estadoVersionado.Anexos[0].VersionDocumento != 1 ||
		estadoVersionado.Anexos[1].VersionDocumento != maximoEnteroSeguroInformeJuridico {
		t.Fatal("los anexos no se ordenaron por referencia y version documental")
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
		"anexo":         func(d *DatosBorradorInformeJuridico) { d.Anexos[0].DocumentoRef = "x" },
		"version anexo": func(d *DatosBorradorInformeJuridico) { d.Anexos[0].VersionDocumento = 0 },
		"huella anexo":  func(d *DatosBorradorInformeJuridico) { d.Anexos[0].HuellaSHA256 = strings.Repeat("0", 64) },
		"anexo duplicado": func(d *DatosBorradorInformeJuridico) {
			d.Anexos[1].DocumentoRef = d.Anexos[0].DocumentoRef
			d.Anexos[1].VersionDocumento = d.Anexos[0].VersionDocumento
		},
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
	datosMaximos := datosInformeJuridicoEnLimitesMaximos(t)
	if _, err := NuevoBorradorInformeJuridico(datosMaximos); err != nil {
		t.Fatalf("se rechazo el tamano maximo: %v", err)
	}

	datosVersionExpediente := datosInformeJuridicoPrueba()
	datosVersionExpediente.VersionEsperadaExpediente = maximoEnteroSeguroInformeJuridico + 1
	if _, err := NuevoBorradorInformeJuridico(datosVersionExpediente); !errors.Is(err, ErrBorradorInformeJuridicoInvalido) {
		t.Fatalf("se acepto una version no interoperable: %v", err)
	}
	datosVersionDocumento := datosInformeJuridicoPrueba()
	datosVersionDocumento.Anexos[0].VersionDocumento = maximoEnteroSeguroInformeJuridico + 1
	if _, err := NuevoBorradorInformeJuridico(datosVersionDocumento); !errors.Is(err, ErrBorradorInformeJuridicoInvalido) {
		t.Fatalf("se acepto una version documental no interoperable: %v", err)
	}

	cardinalidadNormativa := datosInformeJuridicoConNormativas(
		maximoReferenciasNormativasInformeJuridico + 1,
	)
	cardinalidadAnexos := datosInformeJuridicoConAnexos(maximoAnexosInformeJuridico + 1)
	validarFixtureCardinalidadInformeJuridico(t, cardinalidadNormativa)
	validarFixtureCardinalidadInformeJuridico(t, cardinalidadAnexos)

	maximoNormativas := cardinalidadNormativa
	maximoNormativas.ReferenciasNormativas =
		maximoNormativas.ReferenciasNormativas[:maximoReferenciasNormativasInformeJuridico]
	if _, err := NuevoBorradorInformeJuridico(maximoNormativas); err != nil {
		t.Fatalf("se rechazo la cardinalidad normativa maxima: %v", err)
	}
	maximoAnexos := cardinalidadAnexos
	maximoAnexos.Anexos = maximoAnexos.Anexos[:maximoAnexosInformeJuridico]
	if _, err := NuevoBorradorInformeJuridico(maximoAnexos); err != nil {
		t.Fatalf("se rechazo la cardinalidad maxima de anexos: %v", err)
	}

	casosCardinalidad := map[string]DatosBorradorInformeJuridico{
		"cardinalidad normativa maxima mas uno": cardinalidadNormativa,
		"cardinalidad anexos maxima mas uno":    cardinalidadAnexos,
	}
	for nombre, caso := range casosCardinalidad {
		t.Run(nombre, func(t *testing.T) {
			var errRutaTemprana error
			asignaciones := testing.AllocsPerRun(1_000, func() {
				_, errRutaTemprana = NuevoBorradorInformeJuridico(caso)
			})
			if !errors.Is(errRutaTemprana, ErrBorradorInformeJuridicoInvalido) {
				t.Fatalf("se acepto la cardinalidad excedida: %v", errRutaTemprana)
			}
			if asignaciones != 0 {
				t.Fatalf("la guardia de cardinalidad copio o reservo memoria: %.0f asignaciones", asignaciones)
			}
		})
	}

	tamanoExcesivo := datosMaximos
	tamanoExcesivo.ReferenciasNormativas = append(
		[]ReferenciaNormativaInformeJuridico(nil), datosMaximos.ReferenciasNormativas...,
	)
	tamanoExcesivo.ReferenciasNormativas[0].NormaRef += "x"
	var errTamanoExcesivo error
	asignaciones := testing.AllocsPerRun(1_000, func() {
		_, errTamanoExcesivo = NuevoBorradorInformeJuridico(tamanoExcesivo)
	})
	if !errors.Is(errTamanoExcesivo, ErrBorradorInformeJuridicoInvalido) {
		t.Fatalf("se acepto el tamano maximo mas uno: %v", errTamanoExcesivo)
	}
	if asignaciones != 0 {
		t.Fatalf("la guardia de tamano copio o reservo memoria: %.0f asignaciones", asignaciones)
	}
}

func datosInformeJuridicoConNormativas(cantidad int) DatosBorradorInformeJuridico {
	datos := datosInformeJuridicoPrueba()
	datos.ReferenciasNormativas = make([]ReferenciaNormativaInformeJuridico, cantidad)
	for indice := range datos.ReferenciasNormativas {
		datos.ReferenciasNormativas[indice] = ReferenciaNormativaInformeJuridico{
			NormaRef:     fmt.Sprintf("norma:cardinalidad:%02d", indice),
			Version:      uint64(indice + 1),
			HuellaSHA256: fmt.Sprintf("%064x", indice+1),
		}
	}
	return datos
}

func datosInformeJuridicoConAnexos(cantidad int) DatosBorradorInformeJuridico {
	datos := datosInformeJuridicoPrueba()
	datos.Anexos = make([]AnexoDocumentalInformeJuridico, cantidad)
	for indice := range datos.Anexos {
		datos.Anexos[indice] = AnexoDocumentalInformeJuridico{
			DocumentoRef:     fmt.Sprintf("documento:cardinalidad:%02d", indice),
			VersionDocumento: uint64(indice + 1),
			HuellaSHA256:     fmt.Sprintf("%064x", indice+1),
		}
	}
	return datos
}

func validarFixtureCardinalidadInformeJuridico(t *testing.T, datos DatosBorradorInformeJuridico) {
	t.Helper()
	if datos.Canon != CanonBorradorInformeJuridicoV1() ||
		!referenciaValida(datos.ExpedienteRef) ||
		!versionInformeJuridicoValida(datos.VersionEsperadaExpediente) ||
		!referenciaPlantillaInformeJuridicoValida(datos.Plantilla) {
		t.Fatal("el fixture de cardinalidad contiene datos base invalidos")
	}
	for _, referencia := range datos.ReferenciasNormativas {
		if !referenciaNormativaInformeJuridicoValida(referencia) {
			t.Fatal("el fixture de cardinalidad contiene una normativa invalida")
		}
	}
	for _, anexo := range datos.Anexos {
		if !anexoDocumentalInformeJuridicoValido(anexo) {
			t.Fatal("el fixture de cardinalidad contiene un anexo invalido")
		}
	}
	if tieneDuplicadosInformeJuridico(datos) {
		t.Fatal("el fixture de cardinalidad contiene entradas duplicadas")
	}
	if bytes := bytesReferenciasInformeJuridicoPrueba(datos); bytes >= maximoBytesReferenciasInformeJuridico/2 {
		t.Fatalf("el fixture no aisla la cardinalidad: consume %d bytes", bytes)
	}
}

func bytesReferenciasInformeJuridicoPrueba(datos DatosBorradorInformeJuridico) int {
	total := len(datos.ExpedienteRef) + len(datos.Plantilla.PlantillaRef) +
		len(datos.Plantilla.HuellaSHA256)
	for _, referencia := range datos.ReferenciasNormativas {
		total += len(referencia.NormaRef) + len(referencia.HuellaSHA256)
	}
	for _, anexo := range datos.Anexos {
		total += len(anexo.DocumentoRef) + len(anexo.HuellaSHA256)
	}
	return total
}

func datosInformeJuridicoEnLimitesMaximos(t *testing.T) DatosBorradorInformeJuridico {
	t.Helper()
	datos := datosInformeJuridicoPrueba()
	datos.ReferenciasNormativas = make(
		[]ReferenciaNormativaInformeJuridico,
		maximoReferenciasNormativasInformeJuridico,
	)
	datos.Anexos = make([]AnexoDocumentalInformeJuridico, maximoAnexosInformeJuridico)

	const bytesHuellas = (maximoReferenciasNormativasInformeJuridico + maximoAnexosInformeJuridico) * 64
	bytesFijos := len(datos.ExpedienteRef) + len(datos.Plantilla.PlantillaRef) +
		len(datos.Plantilla.HuellaSHA256) + bytesHuellas
	entradas := len(datos.ReferenciasNormativas) + len(datos.Anexos)
	bytesReferencias := maximoBytesReferenciasInformeJuridico - bytesFijos
	longitudBase, restantes := bytesReferencias/entradas, bytesReferencias%entradas

	longitud := func(indice int) int {
		if indice < restantes {
			return longitudBase + 1
		}
		return longitudBase
	}
	for indice := range datos.ReferenciasNormativas {
		prefijo := fmt.Sprintf("norma:%02d:", indice)
		datos.ReferenciasNormativas[indice] = ReferenciaNormativaInformeJuridico{
			NormaRef:     prefijo + strings.Repeat("n", longitud(indice)-len(prefijo)),
			Version:      uint64(indice + 1),
			HuellaSHA256: strings.Repeat("a", 64),
		}
	}
	for indice := range datos.Anexos {
		posicion := len(datos.ReferenciasNormativas) + indice
		prefijo := fmt.Sprintf("anexo:%02d:", indice)
		datos.Anexos[indice] = AnexoDocumentalInformeJuridico{
			DocumentoRef:     prefijo + strings.Repeat("d", longitud(posicion)-len(prefijo)),
			VersionDocumento: uint64(indice + 1),
			HuellaSHA256:     strings.Repeat("b", 64),
		}
	}
	return datos
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
		"version anexo": func(e *EstadoBorradorInformeJuridico) {
			e.Anexos[0].VersionDocumento++
		},
		"huella": func(e *EstadoBorradorInformeJuridico) { e.HuellaSHA256 = strings.Repeat("f", 64) },
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
			{DocumentoRef: "documento:analisis-rrhh:01", VersionDocumento: 11, HuellaSHA256: strings.Repeat("d", 64)},
			{DocumentoRef: "documento:cobertura:01", VersionDocumento: 7, HuellaSHA256: strings.Repeat("e", 64)},
		},
	}
}

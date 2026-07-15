package ports

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPreimagenRecursoAutorizacionV4EsCompletaCanonicaYDefensiva(t *testing.T) {
	escenario, preimagen := nuevaPreimagenRecursoAutorizacionV4Prueba(t)
	if preimagen.Validar() != nil {
		t.Fatal("preimagen valida rechazada")
	}
	recurso, err := preimagen.RecursoCanonico()
	if err != nil || !reflect.DeepEqual(recurso, escenario.expectativa.Recurso) {
		t.Fatalf("recurso canonico distinto: %#v, %v", recurso, err)
	}
	huellaContexto, _ := preimagen.HuellaContextoRecursoSHA256()
	huellaEsperada, _ := escenario.expectativa.Recurso.HuellaContextoAutorizacionSHA256()
	huellaAmbitos, _ := preimagen.HuellaAmbitosSHA256()
	if huellaContexto != huellaEsperada || huellaAmbitos != huellaMapaAutorizacionEjecucionDocumentalV4(
		"vec.documentos.autorizacion-ejecucion.ambitos.v4",
		escenario.expectativa.Recurso.Ambitos,
	) {
		t.Fatal("las huellas recomputadas no corresponden al recurso")
	}
	serializacion, err := preimagen.SerializacionCanonicaParaPersistencia()
	huella, errHuella := preimagen.HuellaSHA256()
	if err != nil || errHuella != nil || len(serializacion) == 0 ||
		binary.BigEndian.Uint64(serializacion[len(serializacion)-8:]) != uint64(len(serializacion)) ||
		huella != huellaBytesFormatoDocumental(serializacion) {
		t.Fatalf("salida persistible invalida: %v, %v", err, errHuella)
	}
	rehidratada, err := InterpretarPreimagenRecursoAutorizacionEjecucionDocumentalV4(
		serializacion, huella,
	)
	if err != nil || rehidratada.Validar() != nil {
		t.Fatalf("rehidratar preimagen: %v", err)
	}
	recursoRehidratado, _ := rehidratada.RecursoCanonico()
	if !reflect.DeepEqual(recursoRehidratado, escenario.expectativa.Recurso) {
		t.Fatal("la rehidratacion perdio el recurso")
	}

	// Todas las salidas son copias profundas.
	recurso.Ambitos["organizacion"] = "otra"
	recurso.Atributos[AtributoAutorizacionDocumentalEfectoRef] = "efecto:otro"
	serializacion[0] ^= 0xff
	segunda, _ := preimagen.RecursoCanonico()
	segundaSerializacion, _ := preimagen.SerializacionCanonicaParaPersistencia()
	if !reflect.DeepEqual(segunda, escenario.expectativa.Recurso) ||
		huellaBytesFormatoDocumental(segundaSerializacion) != huella {
		t.Fatal("un accessor expuso un alias mutable")
	}

	if _, err := json.Marshal(preimagen); !errors.Is(
		err, ErrSerializacionGeneralPreimagenRecursoAutorizacionV4Prohibida,
	) {
		t.Fatalf("JSON no fue bloqueado: %v", err)
	}
	texto := fmt.Sprintf("%v|%+v|%#v", preimagen, preimagen, preimagen)
	if strings.Contains(texto, escenario.expectativa.Recurso.Referencia) ||
		strings.Contains(texto, escenario.expectativa.EfectoRef) {
		t.Fatalf("el formato filtro la preimagen: %s", texto)
	}

	var cero PreimagenRecursoAutorizacionEjecucionDocumentalV4
	if cero.Validar() == nil {
		t.Fatal("la preimagen cero fue aceptada")
	}
}

func TestPreimagenRecursoAutorizacionV4DetectaManipulacionDeTodoElRecurso(t *testing.T) {
	_, original := nuevaPreimagenRecursoAutorizacionV4Prueba(t)
	casos := []struct {
		nombre string
		mutar  func(*PreimagenRecursoAutorizacionEjecucionDocumentalV4)
	}{
		{"referencia", func(p *PreimagenRecursoAutorizacionEjecucionDocumentalV4) {
			p.recurso.Referencia = "recurso:otro"
		}},
		{"modulo", func(p *PreimagenRecursoAutorizacionEjecucionDocumentalV4) {
			p.recurso.ModuloID = "otro"
		}},
		{"tipo", func(p *PreimagenRecursoAutorizacionEjecucionDocumentalV4) {
			p.recurso.Tipo = "otro"
		}},
		{"ambito", func(p *PreimagenRecursoAutorizacionEjecucionDocumentalV4) {
			p.recurso.Ambitos["organizacion"] = "otra"
		}},
		{"plan", func(p *PreimagenRecursoAutorizacionEjecucionDocumentalV4) {
			p.recurso.Atributos[AtributoAutorizacionDocumentalHuellaPlanSHA256] =
				huellaPrueba('f')
		}},
		{"efecto", func(p *PreimagenRecursoAutorizacionEjecucionDocumentalV4) {
			p.recurso.Atributos[AtributoAutorizacionDocumentalEfectoRef] = "efecto:otro"
		}},
		{"atributo adicional", func(p *PreimagenRecursoAutorizacionEjecucionDocumentalV4) {
			p.recurso.Atributos["ampliacion"] = "si"
		}},
		{"huella contexto", func(p *PreimagenRecursoAutorizacionEjecucionDocumentalV4) {
			p.huellaContextoSHA256 = huellaPrueba('1')
		}},
		{"huella ambitos", func(p *PreimagenRecursoAutorizacionEjecucionDocumentalV4) {
			p.huellaAmbitosSHA256 = huellaPrueba('2')
		}},
		{"bytes", func(p *PreimagenRecursoAutorizacionEjecucionDocumentalV4) {
			p.serializacionCanonica[0] ^= 0xff
		}},
		{"huella bytes", func(p *PreimagenRecursoAutorizacionEjecucionDocumentalV4) {
			p.huellaSHA256 = huellaPrueba('3')
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			manipulada := original
			manipulada.recurso = clonarRecursoAutorizacionEjecucionDocumentalV4(original.recurso)
			manipulada.serializacionCanonica = append([]byte(nil), original.serializacionCanonica...)
			caso.mutar(&manipulada)
			if !errors.Is(
				manipulada.Validar(),
				ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida,
			) {
				t.Fatal("preimagen manipulada aceptada")
			}
		})
	}
	if original.Validar() != nil {
		t.Fatal("una copia manipulada altero el original")
	}
}

func TestParserPreimagenRecursoAutorizacionV4RechazaFormatoHostilAntesDeAsignar(t *testing.T) {
	_, preimagen := nuevaPreimagenRecursoAutorizacionV4Prueba(t)
	canonico, _ := preimagen.SerializacionCanonicaParaPersistencia()
	prefijo := len(esquemaPreimagenRecursoAutorizacionEjecucionDocumentalV4) + 1
	posicionPrimerCampo := prefijo + 2
	posicionPrimerMapa := posicionPrimerCampo
	for indice := 0; indice < 3; indice++ {
		longitud := binary.BigEndian.Uint64(canonico[posicionPrimerMapa : posicionPrimerMapa+8])
		posicionPrimerMapa += 8 + int(longitud)
	}

	longitudEnorme := append([]byte(nil), canonico...)
	binary.BigEndian.PutUint64(
		longitudEnorme[posicionPrimerCampo:posicionPrimerCampo+8],
		^uint64(0),
	)
	cantidadEnorme := append([]byte(nil), canonico...)
	binary.BigEndian.PutUint32(
		cantidadEnorme[posicionPrimerMapa:posicionPrimerMapa+4],
		^uint32(0),
	)
	versionDistinta := append([]byte(nil), canonico...)
	binary.BigEndian.PutUint16(versionDistinta[prefijo:prefijo+2], 2)
	ordenMapaNoCanonico := intercambiarPrimerasEntradasMapaPreimagenPrueba(
		t, canonico, posicionPrimerMapa,
	)
	truncada := append([]byte(nil), canonico[:len(canonico)-1]...)
	conExtra := append([]byte(nil), canonico[:len(canonico)-8]...)
	conExtra = append(conExtra, 0xff)
	var total [8]byte
	binary.BigEndian.PutUint64(total[:], uint64(len(conExtra)+8))
	conExtra = append(conExtra, total[:]...)

	casos := []struct {
		nombre    string
		contenido []byte
		huella    string
	}{
		{"longitud enorme", longitudEnorme, huellaBytesFormatoDocumental(longitudEnorme)},
		{"cantidad enorme", cantidadEnorme, huellaBytesFormatoDocumental(cantidadEnorme)},
		{"version distinta", versionDistinta, huellaBytesFormatoDocumental(versionDistinta)},
		{"orden de mapa no canonico", ordenMapaNoCanonico, huellaBytesFormatoDocumental(ordenMapaNoCanonico)},
		{"truncada", truncada, huellaBytesFormatoDocumental(truncada)},
		{"contenido extra", conExtra, huellaBytesFormatoDocumental(conExtra)},
		{"huella ausente", canonico, ""},
		{"huella ajena", canonico, huellaPrueba('9')},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			interpretada, err := InterpretarPreimagenRecursoAutorizacionEjecucionDocumentalV4(
				caso.contenido, caso.huella,
			)
			if !errors.Is(err, ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida) ||
				interpretada.Validar() == nil {
				t.Fatalf("formato hostil aceptado: %v", err)
			}
		})
	}
}

func intercambiarPrimerasEntradasMapaPreimagenPrueba(
	t *testing.T,
	canonico []byte,
	posicionCantidad int,
) []byte {
	t.Helper()
	if binary.BigEndian.Uint32(canonico[posicionCantidad:posicionCantidad+4]) < 2 {
		t.Fatal("el vector necesita al menos dos entradas de ambito")
	}
	primera := posicionCantidad + 4
	finPrimera := finEntradaMapaPreimagenPrueba(t, canonico, primera)
	finSegunda := finEntradaMapaPreimagenPrueba(t, canonico, finPrimera)
	resultado := make([]byte, 0, len(canonico))
	resultado = append(resultado, canonico[:primera]...)
	resultado = append(resultado, canonico[finPrimera:finSegunda]...)
	resultado = append(resultado, canonico[primera:finPrimera]...)
	resultado = append(resultado, canonico[finSegunda:]...)
	return resultado
}

func finEntradaMapaPreimagenPrueba(t *testing.T, contenido []byte, posicion int) int {
	t.Helper()
	for indice := 0; indice < 2; indice++ {
		if posicion > len(contenido)-8 {
			t.Fatal("entrada de mapa truncada en el vector")
		}
		longitud := binary.BigEndian.Uint64(contenido[posicion : posicion+8])
		posicion += 8
		if longitud > uint64(len(contenido)-posicion) {
			t.Fatal("longitud de entrada de mapa invalida en el vector")
		}
		posicion += int(longitud)
	}
	return posicion
}

func nuevaPreimagenRecursoAutorizacionV4Prueba(
	t *testing.T,
) (
	escenarioAutorizacionEjecucionDocumentalV4,
	PreimagenRecursoAutorizacionEjecucionDocumentalV4,
) {
	t.Helper()
	escenario := nuevoEscenarioAutorizacionEjecucionDocumentalV4(t)
	vinculo, err := NuevaSolicitudVinculadaAutorizacionEjecucionDocumentalV4(
		escenario.evidencia,
		escenario.expectativa,
		escenario.vinculadaEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := vinculo.PrepararSolicitudAplicacionEn(
		escenario.vinculadaEn.Add(time.Microsecond),
	)
	if err != nil {
		t.Fatal(err)
	}
	preimagen, err := solicitud.PreimagenRecursoParaEvidenciaDurable()
	if err != nil {
		t.Fatal(err)
	}
	return escenario, preimagen
}

func TestPreimagenRecursoAutorizacionV4SerializacionNoDependeDelOrdenDeMapas(t *testing.T) {
	_, original := nuevaPreimagenRecursoAutorizacionV4Prueba(t)
	reordenado := original.recurso
	reordenado.Ambitos = map[string]string{
		"procedimiento": original.recurso.Ambitos["procedimiento"],
		"organizacion":  original.recurso.Ambitos["organizacion"],
	}
	reordenado.Atributos = map[string]string{
		AtributoAutorizacionDocumentalHuellaPlanSHA256: original.recurso.Atributos[AtributoAutorizacionDocumentalHuellaPlanSHA256],
		AtributoAutorizacionDocumentalEfectoRef:        original.recurso.Atributos[AtributoAutorizacionDocumentalEfectoRef],
	}
	segunda, err := nuevaPreimagenRecursoAutorizacionEjecucionDocumentalV4(reordenado)
	if err != nil || !bytes.Equal(original.serializacionCanonica, segunda.serializacionCanonica) ||
		original.huellaSHA256 != segunda.huellaSHA256 {
		t.Fatalf("el orden de insercion altero la forma canonica: %v", err)
	}
}

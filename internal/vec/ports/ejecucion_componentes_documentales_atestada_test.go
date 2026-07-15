package ports

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestCompromisoEjecucionDocumentalLigaTodosLosEjes(t *testing.T) {
	escenario := nuevoEscenarioAtestacionEjecucionDocumental(t)
	compromiso := nuevoCompromisoAtestacionEjecucionDocumental(
		t, escenario, OperacionVerificacionSemanticaDocumental, escenario.semantico,
		"operacion:semantica:1", retoAtestacionDocumental(3), huellaAtestacionDocumental('9'), 128,
	)
	if compromiso.Validar() != nil || compromiso.DescriptorPerfil() != escenario.descriptorPerfil ||
		compromiso.DescriptorComponente() != escenario.semantico ||
		compromiso.HuellaDocumentoSHA256() != huellaAtestacionDocumental('9') ||
		compromiso.TamanoDocumento() != 128 || compromiso.LimiteBytes() != escenario.limite {
		t.Fatalf("compromiso valido rechazado: %#v", compromiso)
	}
	if !strings.Contains(compromiso.String(), "NOMINAL-NO-AUTORITATIVO") ||
		strings.Contains(compromiso.String(), "CON-CAPACIDAD") {
		t.Fatalf("String ambiguo para un valor nominal: %s", compromiso.String())
	}
	huella, err := compromiso.HuellaSHA256()
	if err != nil || len(huella) != 64 {
		t.Fatalf("huella de compromiso invalida: %q, %v", huella, err)
	}

	mutaciones := []struct {
		nombre string
		mutar  func(*CompromisoEjecucionComponenteDocumental)
	}{
		{"operacion", func(c *CompromisoEjecucionComponenteDocumental) { c.operacion = OperacionRenderizadoDocumental }},
		{"publicacion", func(c *CompromisoEjecucionComponenteDocumental) {
			c.descriptorPerfil.publicacionRef = "publicacion:otra:2"
		}},
		{"revision", func(c *CompromisoEjecucionComponenteDocumental) {
			c.descriptorPerfil.revision = domain.RevisionCatalogoFormatosDocumentales{}
		}},
		{"situacion", func(c *CompromisoEjecucionComponenteDocumental) {
			c.situacionOperativa = escenario.revocada
		}},
		{"descriptor", func(c *CompromisoEjecucionComponenteDocumental) {
			c.descriptorComponente.referencia = "atestacion:otra"
		}},
		{"reserva", func(c *CompromisoEjecucionComponenteDocumental) {
			c.reservaRef = "reserva:documental:otra"
		}},
		{"vinculo activacion", func(c *CompromisoEjecucionComponenteDocumental) {
			c.vinculoActivacion.ReservaRef = "reserva:documental:otra"
		}},
		{"efecto", func(c *CompromisoEjecucionComponenteDocumental) {
			c.efectoRef = "efecto:documental:otro"
		}},
		{"plan", func(c *CompromisoEjecucionComponenteDocumental) {
			c.huellaPlanSHA256 = huellaAtestacionDocumental('0')
		}},
		{"secuencia cercado", func(c *CompromisoEjecucionComponenteDocumental) {
			c.secuenciaCercado++
		}},
		{"vinculo cercado", func(c *CompromisoEjecucionComponenteDocumental) {
			c.huellaVinculoSHA256 = huellaAtestacionDocumental('0')
		}},
		{"manifiesto", func(c *CompromisoEjecucionComponenteDocumental) {
			c.manifiesto = ManifiestoEjecucionDocumentalV3{}
		}},
		{"consumo", func(c *CompromisoEjecucionComponenteDocumental) {
			c.consumoDecision.EfectoRef = "efecto:documental:otro"
		}},
		{"orden despacho consumida", func(c *CompromisoEjecucionComponenteDocumental) {
			c.ordenDespachoConsumida = OrdenDespachoDocumentalV3ConsumidaNominal{}
		}},
		{"consumo CAS despacho", func(c *CompromisoEjecucionComponenteDocumental) {
			c.ordenDespachoConsumida.estado.versionConsumoCAS++
		}},
		{"reto", func(c *CompromisoEjecucionComponenteDocumental) { c.reto = [32]byte{} }},
		{"hmac", func(c *CompromisoEjecucionComponenteDocumental) {
			c.huellaContenidoNeutralHMAC = "sha256:" + huellaAtestacionDocumental('1')
		}},
		{"hash", func(c *CompromisoEjecucionComponenteDocumental) {
			c.huellaDocumentoSHA256 = huellaAtestacionDocumental('8')
		}},
		{"tamano", func(c *CompromisoEjecucionComponenteDocumental) { c.tamanoDocumento++ }},
		{"limite", func(c *CompromisoEjecucionComponenteDocumental) { c.limiteBytes++ }},
		{"expiracion", func(c *CompromisoEjecucionComponenteDocumental) {
			c.expiraEn = c.expiraEn.Add(time.Second)
		}},
	}
	for _, prueba := range mutaciones {
		t.Run("manipulacion_"+prueba.nombre, func(t *testing.T) {
			alterado := compromiso
			prueba.mutar(&alterado)
			if alterado.Validar() == nil {
				t.Fatal("la manipulacion mantuvo autoridad")
			}
		})
	}
	if (CompromisoEjecucionComponenteDocumental{}).Validar() == nil {
		t.Fatal("el compromiso cero obtuvo autoridad")
	}
}

func TestCompromisoEjecucionDocumentalRechazaRolPIIComodinYModosAmbiguos(t *testing.T) {
	escenario := nuevoEscenarioAtestacionEjecucionDocumental(t)
	hmac := hmacAtestacionDocumental()
	semanticoAjeno := descriptorComponenteDocumentalPrueba(
		t, escenario.semantico.Consulta(), "verificador-semantico-ajeno", 4,
		"dominio:semantica:ajena", "d", escenario.limite,
	)
	casos := []struct {
		nombre       string
		operacionRef string
		reto         [32]byte
		operacion    OperacionComponenteDocumental
		componente   DescriptorComponenteDocumentalAtestado
		huella       string
		tamano       uint64
		limite       uint64
	}{
		{"reto nulo", "operacion:render:1", [32]byte{}, OperacionRenderizadoDocumental, escenario.render, "", 0, escenario.limite},
		{"rol ajeno", "operacion:render:1", retoAtestacionDocumental(1), OperacionRenderizadoDocumental, escenario.estructural, "", 0, escenario.limite},
		{"pii evidente", "operacion:dni:12345678z", retoAtestacionDocumental(1), OperacionRenderizadoDocumental, escenario.render, "", 0, escenario.limite},
		{"comodin", "operacion:*", retoAtestacionDocumental(1), OperacionRenderizadoDocumental, escenario.render, "", 0, escenario.limite},
		{"render con salida propuesta", "operacion:render:1", retoAtestacionDocumental(1), OperacionRenderizadoDocumental, escenario.render, huellaAtestacionDocumental('9'), 128, escenario.limite},
		{"verificacion sin salida", "operacion:estructura:1", retoAtestacionDocumental(2), OperacionValidacionEstructuralDocumental, escenario.estructural, "", 0, escenario.limite},
		{"sobre limite", "operacion:estructura:1", retoAtestacionDocumental(2), OperacionValidacionEstructuralDocumental, escenario.estructural, huellaAtestacionDocumental('9'), escenario.limite + 1, escenario.limite},
		{"operacion abierta", "operacion:libre:1", retoAtestacionDocumental(4), OperacionComponenteDocumental("libre"), escenario.estructural, huellaAtestacionDocumental('9'), 128, escenario.limite},
		{"semantico ajeno al plan", "operacion:semantica:ajena", retoAtestacionDocumental(5), OperacionVerificacionSemanticaDocumental, semanticoAjeno, huellaAtestacionDocumental('9'), 128, escenario.limite},
	}
	for _, prueba := range casos {
		t.Run(prueba.nombre, func(t *testing.T) {
			_, err := NuevoCompromisoEjecucionComponenteDocumental(
				prueba.operacionRef, prueba.reto, prueba.operacion, escenario.descriptorPerfil,
				escenario.vigente, prueba.componente, escenario.ordenDespachoConsumida,
				"borrador:documental:1", hmac,
				prueba.huella, prueba.tamano, prueba.limite,
				escenario.expiraEn.Sub(escenario.emitidoEn),
			)
			if !errors.Is(err, ErrCompromisoEjecucionDocumentalInvalido) {
				t.Fatalf("entrada hostil aceptada: %v", err)
			}
		})
	}
}

func TestCompromisoEjecucionDocumentalRechazaSituacionCruzadaRevocadaYVentanaAbierta(t *testing.T) {
	escenario := nuevoEscenarioAtestacionEjecucionDocumental(t)
	casos := []struct {
		nombre    string
		situacion domain.SituacionOperativaPerfilDocumental
		vigencia  time.Duration
	}{
		{"revocada", escenario.revocada, 5 * time.Minute},
		{"publicacion cruzada", escenario.vigenteAjena, 5 * time.Minute},
		{"ventana mayor de diez minutos", escenario.vigente, 10*time.Minute + time.Nanosecond},
		{"ventana nula", escenario.vigente, 0},
		{"ventana invertida", escenario.vigente, -time.Second},
	}
	for _, prueba := range casos {
		t.Run(prueba.nombre, func(t *testing.T) {
			_, err := NuevoCompromisoEjecucionComponenteDocumental(
				"operacion:render:1", retoAtestacionDocumental(1),
				OperacionRenderizadoDocumental, escenario.descriptorPerfil,
				prueba.situacion, escenario.render, escenario.ordenDespachoConsumida,
				"borrador:documental:1",
				hmacAtestacionDocumental(), "", 0, escenario.limite,
				prueba.vigencia,
			)
			if !errors.Is(err, ErrCompromisoEjecucionDocumentalInvalido) {
				t.Fatalf("situacion/ventana no autorizante aceptada: %v", err)
			}
		})
	}
}

func TestCompromisoEjecucionDocumentalRechazaOrdenDespachoDeOtraReservaAunqueRecalculeDigest(t *testing.T) {
	escenario := nuevoEscenarioAtestacionEjecucionDocumental(t)
	compromiso := nuevoCompromisoAtestacionEjecucionDocumental(
		t, escenario, OperacionRenderizadoDocumental, escenario.render,
		"operacion:render:1", retoAtestacionDocumental(1), "", 0,
	)
	compromiso.ordenDespachoConsumida.solicitud.vinculo.ReservaRef = "reserva:documental:otra"
	compromiso.huellaCompromisoSHA256 = compromiso.calcularHuella()
	if compromiso.Validar() == nil {
		t.Fatal("una orden cruzada con otra reserva autorizo el compromiso")
	}
}

func TestSobreCOSEEsOpacoLimitadoYCopiadoDefensivamente(t *testing.T) {
	original := bytes.Repeat([]byte{0xa1}, 32)
	sobre, err := NuevoSobreReciboEjecucionDocumentalCrudo(original)
	if err != nil || sobre.Validar() != nil {
		t.Fatalf("sobre valido rechazado: %v", err)
	}
	original[0] = 0xff
	primera, err := sobre.COSESign1()
	if err != nil || primera[0] != 0xa1 {
		t.Fatal("el constructor retuvo el buffer del llamador")
	}
	primera[0] = 0xee
	segunda, _ := sobre.COSESign1()
	if segunda[0] != 0xa1 {
		t.Fatal("el getter expuso el buffer autoritativo")
	}
	for _, contenido := range [][]byte{
		nil,
		bytes.Repeat([]byte{0}, minimoBytesSobreCOSEDocumental),
		bytes.Repeat([]byte{1}, minimoBytesSobreCOSEDocumental-1),
		bytes.Repeat([]byte{1}, maximoBytesSobreCOSEDocumental+1),
	} {
		if _, err := NuevoSobreReciboEjecucionDocumentalCrudo(contenido); !errors.Is(
			err, ErrSobreReciboEjecucionDocumentalInvalido,
		) {
			t.Fatalf("sobre invalido aceptado, longitud=%d: %v", len(contenido), err)
		}
	}
}

func TestCompromisoYSobreCOSESeRedactanYNoSeSerializan(t *testing.T) {
	escenario := nuevoEscenarioAtestacionEjecucionDocumental(t)
	compromiso := nuevoCompromisoAtestacionEjecucionDocumental(
		t, escenario, OperacionRenderizadoDocumental, escenario.render,
		"operacion:render:secreto", retoAtestacionDocumental(1), "", 0,
	)
	sobre := sobreAtestacionDocumental(t, 7)
	identidad, err := NuevaIdentidadEjecucionComponenteDocumental(
		"carga:trabajo:redactada", "instancia:proceso:redactada",
		"dominio:aislamiento:redactado", "clave:firma:redactada",
		strings.Repeat("8", 64), strings.Repeat("9", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	texto := fmt.Sprintf("%v|%+v|%#v|%s", compromiso, compromiso, compromiso, compromiso)
	if strings.Contains(texto, "token:cercado:documental:1") ||
		strings.Contains(texto, "clave:atestacion:cercado") ||
		strings.Contains(texto, "operacion:render:secreto") {
		t.Fatalf("el compromiso filtro su capacidad: %s", texto)
	}
	textoSobre := fmt.Sprintf("%v|%+v|%#v|%s", sobre, sobre, sobre, sobre)
	if strings.Contains(textoSobre, "070707") {
		t.Fatalf("el sobre COSE filtro bytes: %s", textoSobre)
	}
	textoIdentidad := fmt.Sprintf("%v|%+v|%#v|%s", identidad, identidad, identidad, identidad)
	if strings.Contains(textoIdentidad, identidad.ClaveFirmaRef()) ||
		strings.Contains(textoIdentidad, identidad.CargaTrabajoRef()) {
		t.Fatalf("la identidad filtro referencias de confianza: %s", textoIdentidad)
	}
	for nombre, valor := range map[string]any{
		"compromiso": compromiso,
		"sobre":      sobre,
		"identidad":  identidad,
	} {
		if _, err := json.Marshal(valor); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
			t.Fatalf("%s se serializo por JSON: %v", nombre, err)
		}
		if slog.Any("valor", valor).Value.Resolve().Kind() != slog.KindString {
			t.Fatalf("%s no se redacto en slog", nombre)
		}
		binario, ok := valor.(interface{ MarshalBinary() ([]byte, error) })
		if !ok {
			t.Fatalf("%s no bloquea binario", nombre)
		}
		if _, err := binario.MarshalBinary(); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
			t.Fatalf("%s se serializo como binario: %v", nombre, err)
		}
		texto, ok := valor.(interface{ MarshalText() ([]byte, error) })
		if !ok {
			t.Fatalf("%s no bloquea texto", nombre)
		}
		if _, err := texto.MarshalText(); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
			t.Fatalf("%s se serializo como texto: %v", nombre, err)
		}
	}
	if _, err := compromiso.MarshalText(); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("compromiso se serializo como texto: %v", err)
	}
	if _, err := sobre.MarshalText(); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("sobre se serializo como texto: %v", err)
	}
	var compromisoRestaurado CompromisoEjecucionComponenteDocumental
	if err := json.Unmarshal([]byte(`{}`), &compromisoRestaurado); !errors.Is(
		err, ErrSerializacionSecretoDocumentalV3,
	) {
		t.Fatalf("compromiso se restauro por JSON: %v", err)
	}
	var sobreRestaurado SobreReciboEjecucionDocumentalCrudo
	if err := json.Unmarshal([]byte(`{}`), &sobreRestaurado); !errors.Is(
		err, ErrSerializacionSecretoDocumentalV3,
	) {
		t.Fatalf("sobre se restauro por JSON: %v", err)
	}
}

func TestCompromisoRechazaTiempoNoCanonicoAunqueRecalculeHuellas(t *testing.T) {
	escenario := nuevoEscenarioAtestacionEjecucionDocumental(t)
	compromiso := nuevoCompromisoAtestacionEjecucionDocumental(
		t, escenario, OperacionRenderizadoDocumental, escenario.render,
		"operacion:render:tiempo", retoAtestacionDocumental(1), "", 0,
	)
	compromiso.emitidoEn = compromiso.emitidoEn.Add(time.Nanosecond)
	compromiso.expiraEn = compromiso.expiraEn.Add(time.Nanosecond)
	compromiso.ordenDespachoConsumida.estado.consumidaEn = compromiso.emitidoEn
	compromiso.huellaCompromisoSHA256 = compromiso.calcularHuella()
	if compromiso.Validar() == nil {
		t.Fatal("un instante no persistible a microsegundos conservo autoridad")
	}
}

func TestRecibosVerificadosExigenMedicionYSegregacionRuntime(t *testing.T) {
	escenario := nuevoEscenarioAtestacionEjecucionDocumental(t)
	sobreRender := sobreAtestacionDocumental(t, 1)
	sobreEstructura := sobreAtestacionDocumental(t, 2)
	sobreSemantica := sobreAtestacionDocumental(t, 3)
	huellaSalida := huellaAtestacionDocumental('9')
	render := nuevoCompromisoAtestacionEjecucionDocumental(
		t, escenario, OperacionRenderizadoDocumental, escenario.render,
		"operacion:render:1", retoAtestacionDocumental(1), "", 0,
	)
	estructura := nuevoCompromisoAtestacionEjecucionDocumental(
		t, escenario, OperacionValidacionEstructuralDocumental, escenario.estructural,
		"operacion:estructura:1", retoAtestacionDocumental(2), huellaSalida, 128,
	)
	semantica := nuevoCompromisoAtestacionEjecucionDocumental(
		t, escenario, OperacionVerificacionSemanticaDocumental, escenario.semantico,
		"operacion:semantica:1", retoAtestacionDocumental(3), huellaSalida, 128,
	)
	identidadRender := identidadAtestacionDocumental(t, escenario.render, "render", "proceso:render:1", 'd')
	identidadEstructura := identidadAtestacionDocumental(t, escenario.estructural, "estructura", "proceso:estructura:1", 'e')
	identidadSemantica := identidadAtestacionDocumental(t, escenario.semantico, "semantica", "proceso:semantica:1", 'f')
	instante := time.Date(2026, time.July, 15, 16, 59, 0, 0, time.UTC)
	reciboRender := nuevoReciboAtestacionDocumental(
		t, render, sobreRender, "recibo:render:1", ResultadoRenderizadoDocumentalCorrecto,
		huellaSalida, 128, identidadRender, instante,
	)
	reciboEstructura := nuevoReciboAtestacionDocumental(
		t, estructura, sobreEstructura, "recibo:estructura:1", ResultadoEstructuraDocumentalConforme,
		huellaSalida, 128, identidadEstructura, instante,
	)
	reciboSemantica := nuevoReciboAtestacionDocumental(
		t, semantica, sobreSemantica, "recibo:semantica:1", ResultadoSemanticaDocumentalEquivalente,
		huellaSalida, 128, identidadSemantica, instante,
	)
	if !reciboRender.IndependienteDe(reciboEstructura) ||
		!reciboRender.IndependienteDe(reciboSemantica) ||
		!reciboEstructura.IndependienteDe(reciboSemantica) {
		t.Fatal("tres procesos atestados distintos no se reconocieron como independientes")
	}
	datosRecibo, err := reciboRender.Datos()
	if err != nil {
		t.Fatal(err)
	}
	_ = map[DatosReciboEjecucionComponenteDocumentalNominal]struct{}{datosRecibo: {}}
	tipoDatos := reflect.TypeOf(datosRecibo)
	for _, tipoProhibido := range []reflect.Type{
		reflect.TypeOf(CompromisoEjecucionComponenteDocumental{}),
		reflect.TypeOf(TokenCercadoEjecucionDocumentalV3Nominal{}),
		reflect.TypeOf(SobreReciboEjecucionDocumentalCrudo{}),
		reflect.TypeOf([]byte(nil)),
	} {
		for indice := 0; indice < tipoDatos.NumField(); indice++ {
			if tipoDatos.Field(indice).Type == tipoProhibido {
				t.Fatalf("la evidencia expuso capacidad o bytes secretos: %v", tipoProhibido)
			}
		}
	}
	secretoCercado := escenario.tokenCercado.valor
	if secretoCercado == "" || strings.Contains(fmt.Sprintf("%#v", reciboRender), secretoCercado) ||
		strings.Contains(fmt.Sprintf("%#v", datosRecibo), secretoCercado) {
		t.Fatal("el recibo o su proyeccion retuvieron el valor secreto de cercado")
	}
	if reciboRender.ValidarContra(render, sobreRender) != nil {
		t.Fatal("el recibo correcto no quedo ligado al compromiso y COSE exactos")
	}
	if reciboRender.ValidarContra(render, sobreEstructura) == nil {
		t.Fatal("el recibo acepto un COSE diferente")
	}

	mismaInstancia := identidadAtestacionDocumental(
		t, escenario.semantico, "semantica-otra", identidadRender.InstanciaProcesoRef(), 'a',
	)
	reciboMismaInstancia := nuevoReciboAtestacionDocumental(
		t, semantica, sobreSemantica, "recibo:semantica:2", ResultadoSemanticaDocumentalEquivalente,
		huellaSalida, 128, mismaInstancia, instante,
	)
	if reciboRender.IndependienteDe(reciboMismaInstancia) {
		t.Fatal("el mismo proceso parecio una barrera independiente")
	}
	mismaCarga, err := NuevaIdentidadEjecucionComponenteDocumental(
		identidadRender.CargaTrabajoRef(), "proceso:semantica:carga-compartida",
		escenario.semantico.DominioConfianzaRef(), "clave:semantica:carga-compartida",
		huellaAtestacionDocumental('a'),
		escenario.semantico.Componente().HuellaArtefactoSHA256(),
	)
	if err != nil {
		t.Fatal(err)
	}
	reciboMismaCarga := nuevoReciboAtestacionDocumental(
		t, semantica, sobreSemantica, "recibo:semantica:carga-compartida",
		ResultadoSemanticaDocumentalEquivalente, huellaSalida, 128, mismaCarga, instante,
	)
	if reciboRender.IndependienteDe(reciboMismaCarga) {
		t.Fatal("la misma carga de trabajo parecio independiente")
	}

	mismaClave, err := NuevaIdentidadEjecucionComponenteDocumental(
		"carga:semantica:otra", "proceso:semantica:otra",
		escenario.semantico.DominioConfianzaRef(), "clave:semantica:otra",
		identidadRender.HuellaClaveFirmaSHA256(),
		escenario.semantico.Componente().HuellaArtefactoSHA256(),
	)
	if err != nil {
		t.Fatal(err)
	}
	reciboMismaClave := nuevoReciboAtestacionDocumental(
		t, semantica, sobreSemantica, "recibo:semantica:3", ResultadoSemanticaDocumentalEquivalente,
		huellaSalida, 128, mismaClave, instante,
	)
	if reciboRender.IndependienteDe(reciboMismaClave) {
		t.Fatal("la misma clave de carga de trabajo parecio independiente")
	}
}

func TestRecibosIndependientesPuedenCompartirAutoridadDeAtestacion(t *testing.T) {
	escenario := nuevoEscenarioAtestacionEjecucionDocumental(t)
	semantico := escenario.semantico
	semantico.atestacionBrokerRef = escenario.render.AtestacionBrokerRef()
	semantico.huellaAtestacionBrokerSHA256 = escenario.render.HuellaAtestacionBrokerSHA256()
	semantico.digestDeclaracion = semantico.calcularDigestDeclaracion()
	if semantico.Validar() != nil || !escenario.render.IndependienteDe(semantico) {
		t.Fatal("compartir autoridad de atestacion altero la segregacion del ejecutor")
	}
	perfil := escenario.descriptorPerfil.Perfil()
	consulta := ConsultaFormatoDocumental{
		Identidad: perfil.Identidad(), PerfilRef: perfil.Referencia(),
		DigestPerfilSHA256: perfil.DigestSHA256(),
		RevisionCatalogo:   escenario.descriptorPerfil.Revision(),
	}
	manifiesto, err := NuevoManifiestoEjecucionDocumentalV3(
		consulta, escenario.descriptorPerfil, escenario.vigente, escenario.render,
		escenario.estructural, semantico, "borrador:documental:1", "efecto:documental:1",
		hmacAtestacionDocumental(), escenario.limite,
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaPlan, _ := manifiesto.HuellaSHA256()
	consumo := escenario.consumoDecision
	consumo.HuellaPlanSHA256 = huellaPlan
	vinculoActivacion := escenario.vinculoActivacion
	vinculoActivacion.Manifiesto = manifiesto
	vinculoActivacion.ConsumoDecision = consumo
	vinculoActivacion.OrdenConsumoDurableV4Ref = consumo.EfectoRef
	if vinculoActivacion.Validar() != nil {
		t.Fatal("vinculo compartido invalido")
	}
	token := nuevoTokenCercadoEjecucionDocumentalV3Prueba(
		t, "token:cercado:documental:compartido", 13, vinculoActivacion,
		"clave:atestacion:cercado", 1, "evidencia:cercado:compartido",
	)
	ordenDespachoConsumida := ordenDespachoDocumentalV3ConsumidaNominalPrueba(
		t, vinculoActivacion, token, escenario.emitidoEn, escenario.expiraEn, "compartida",
	)
	escenario.semantico = semantico
	escenario.manifiesto = manifiesto
	escenario.consumoDecision = consumo
	escenario.vinculoActivacion = vinculoActivacion
	escenario.tokenCercado = token
	escenario.ordenDespachoConsumida = ordenDespachoConsumida
	huellaSalida := huellaAtestacionDocumental('9')
	compromisoRender := nuevoCompromisoAtestacionEjecucionDocumental(
		t, escenario, OperacionRenderizadoDocumental, escenario.render,
		"operacion:render:compartido", retoAtestacionDocumental(5), "", 0,
	)
	compromisoSemantico := nuevoCompromisoAtestacionEjecucionDocumental(
		t, escenario, OperacionVerificacionSemanticaDocumental, semantico,
		"operacion:semantica:compartida", retoAtestacionDocumental(6), huellaSalida, 128,
	)
	reciboRender := nuevoReciboAtestacionDocumental(
		t, compromisoRender, sobreAtestacionDocumental(t, 5), "recibo:render:compartido",
		ResultadoRenderizadoDocumentalCorrecto, huellaSalida, 128,
		identidadAtestacionDocumental(t, escenario.render, "render-compartido", "proceso:render:compartido", 'd'),
		escenario.emitidoEn.Add(time.Minute),
	)
	reciboSemantico := nuevoReciboAtestacionDocumental(
		t, compromisoSemantico, sobreAtestacionDocumental(t, 6), "recibo:semantica:compartida",
		ResultadoSemanticaDocumentalEquivalente, huellaSalida, 128,
		identidadAtestacionDocumental(t, semantico, "semantica-compartida", "proceso:semantica:compartida", 'f'),
		escenario.emitidoEn.Add(time.Minute),
	)
	if !reciboRender.IndependienteDe(reciboSemantico) {
		t.Fatal("una autoridad comun se confundio con una carga de trabajo comun")
	}
}

func TestReciboVerificadoRechazaResultadoMedicionYManipulacion(t *testing.T) {
	escenario := nuevoEscenarioAtestacionEjecucionDocumental(t)
	compromiso := nuevoCompromisoAtestacionEjecucionDocumental(
		t, escenario, OperacionVerificacionSemanticaDocumental, escenario.semantico,
		"operacion:semantica:1", retoAtestacionDocumental(3), huellaAtestacionDocumental('9'), 128,
	)
	sobre := sobreAtestacionDocumental(t, 3)
	identidad := identidadAtestacionDocumental(t, escenario.semantico, "semantica", "proceso:semantica:1", 'f')
	instante := time.Date(2026, time.July, 15, 16, 59, 0, 0, time.UTC)
	if _, err := NuevoReciboEjecucionComponenteDocumentalNominal(
		compromiso, sobre, "recibo:semantica:1", ResultadoEstructuraDocumentalConforme,
		huellaAtestacionDocumental('9'), 128, identidad, instante,
	); !errors.Is(err, ErrReciboEjecucionDocumentalInvalido) {
		t.Fatalf("resultado de otro rol aceptado: %v", err)
	}
	if _, err := NuevoReciboEjecucionComponenteDocumentalNominal(
		compromiso, sobre, "recibo:semantica:expirado", ResultadoSemanticaDocumentalEquivalente,
		huellaAtestacionDocumental('9'), 128, identidad, escenario.expiraEn,
	); !errors.Is(err, ErrReciboEjecucionDocumentalInvalido) {
		t.Fatalf("recibo emitido en el borde exclusivo de expiracion aceptado: %v", err)
	}
	medicionAjena, err := NuevaIdentidadEjecucionComponenteDocumental(
		"carga:semantica:1", "proceso:semantica:1", escenario.semantico.DominioConfianzaRef(),
		"clave:semantica:1", huellaAtestacionDocumental('f'), huellaAtestacionDocumental('0'),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NuevoReciboEjecucionComponenteDocumentalNominal(
		compromiso, sobre, "recibo:semantica:1", ResultadoSemanticaDocumentalEquivalente,
		huellaAtestacionDocumental('9'), 128, medicionAjena, instante,
	); !errors.Is(err, ErrReciboEjecucionDocumentalInvalido) {
		t.Fatalf("medicion ajena al artefacto aceptada: %v", err)
	}
	recibo := nuevoReciboAtestacionDocumental(
		t, compromiso, sobre, "recibo:semantica:1", ResultadoSemanticaDocumentalEquivalente,
		huellaAtestacionDocumental('9'), 128, identidad, instante,
	)
	datosRecibo, err := recibo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	textoDatos := fmt.Sprintf("%v|%+v|%#v", datosRecibo, datosRecibo, datosRecibo)
	if strings.Contains(textoDatos, datosRecibo.HuellaContenidoNeutralHMAC) {
		t.Fatal("el DTO de recibo filtro la HMAC")
	}
	if slog.Any("datos", datosRecibo).Value.Resolve().Kind() != slog.KindString {
		t.Fatal("el DTO de recibo no se redacto en slog")
	}
	if _, err := json.Marshal(datosRecibo); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("DTO de recibo serializado por JSON: %v", err)
	}
	if _, err := datosRecibo.MarshalText(); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("DTO de recibo serializado como texto: %v", err)
	}
	if _, err := datosRecibo.MarshalBinary(); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("DTO de recibo serializado como binario: %v", err)
	}
	recibo.tamanoSalida++
	if recibo.Validar() == nil {
		t.Fatal("un recibo manipulado conservo autoridad")
	}
	if (ReciboEjecucionComponenteDocumentalNominal{}).Validar() == nil ||
		(IdentidadEjecucionComponenteDocumental{}).Validar() == nil {
		t.Fatal("un valor cero obtuvo semantica positiva")
	}
}

type escenarioAtestacionEjecucionDocumental struct {
	descriptorPerfil       DescriptorPerfilDocumental
	vigente                domain.SituacionOperativaPerfilDocumental
	revocada               domain.SituacionOperativaPerfilDocumental
	vigenteAjena           domain.SituacionOperativaPerfilDocumental
	render                 DescriptorComponenteDocumentalAtestado
	estructural            DescriptorComponenteDocumentalAtestado
	semantico              DescriptorComponenteDocumentalAtestado
	reservaRef             string
	manifiesto             ManifiestoEjecucionDocumentalV3
	consumoDecision        ConsumoDecisionEjecucionDocumentalV3
	vinculoActivacion      VinculoEstableActivacionDocumentalV3
	tokenCercado           TokenCercadoEjecucionDocumentalV3Nominal
	ordenDespachoConsumida OrdenDespachoDocumentalV3ConsumidaNominal
	limite                 uint64
	emitidoEn              time.Time
	expiraEn               time.Time
}

func nuevoEscenarioAtestacionEjecucionDocumental(
	t *testing.T,
) escenarioAtestacionEjecucionDocumental {
	t.Helper()
	perfil, revision := valoresPerfilDocumentalGobernadoPrueba(t)
	descriptorPerfil, err := NuevoDescriptorPerfilDocumental(
		"descriptor:pdfa4:2", "publicacion:pdfa4:2", perfil, revision,
	)
	if err != nil {
		t.Fatal(err)
	}
	vigente, err := domain.NuevaSituacionOperativaPerfilDocumental(
		descriptorPerfil.PublicacionRef(), perfil, revision, 7,
		domain.EstadoPublicacionPerfilVigente,
	)
	if err != nil {
		t.Fatal(err)
	}
	revocada, err := domain.NuevaSituacionOperativaPerfilDocumental(
		descriptorPerfil.PublicacionRef(), perfil, revision, 8,
		domain.EstadoPublicacionPerfilRevocada,
	)
	if err != nil {
		t.Fatal(err)
	}
	vigenteAjena, err := domain.NuevaSituacionOperativaPerfilDocumental(
		"publicacion:pdfa4:ajena", perfil, revision, 1,
		domain.EstadoPublicacionPerfilVigente,
	)
	if err != nil {
		t.Fatal(err)
	}
	limite := uint64(8 * 1024 * 1024)
	emitidoEn := time.Date(2026, time.July, 15, 16, 55, 0, 0, time.UTC)
	render := descriptorComponenteDocumentalPrueba(
		t, consultaComponenteDocumentalPrueba(perfil, revision, domain.RolComponenteRenderizador),
		"renderizador-pdfa", 1, "dominio:renderizador", "1", limite,
	)
	estructural := descriptorComponenteDocumentalPrueba(
		t, consultaComponenteDocumentalPrueba(perfil, revision, domain.RolComponenteValidadorEstructural),
		"validador-estructural-pdfa", 2, "dominio:estructura", "4", limite,
	)
	semantico := descriptorComponenteDocumentalPrueba(
		t, consultaComponenteDocumentalPrueba(perfil, revision, domain.RolComponenteVerificadorSemantico),
		"verificador-semantico-pdfa", 3, "dominio:semantica", "a", limite,
	)
	consulta := ConsultaFormatoDocumental{
		Identidad: perfil.Identidad(), PerfilRef: perfil.Referencia(),
		DigestPerfilSHA256: perfil.DigestSHA256(), RevisionCatalogo: revision,
	}
	manifiesto, err := NuevoManifiestoEjecucionDocumentalV3(
		consulta, descriptorPerfil, vigente, render, estructural, semantico,
		"borrador:documental:1", "efecto:documental:1", hmacAtestacionDocumental(), limite,
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaPlan, err := manifiesto.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	consumo := ConsumoDecisionEjecucionDocumentalV3{
		DecisionRef: "decision:documental:1", EfectoRef: "efecto:documental:1",
		EsquemaHuellaDecision: EsquemaHuellaDecisionAutorizacionReforzadaV1,
		HuellaDecisionSHA256:  huellaAtestacionDocumental('e'), HuellaPlanSHA256: huellaPlan,
	}
	reservaRef := "reserva:documental:1"
	solicitudActivar := SolicitudActivarEjecucionDocumentalV3{
		ReservaRef:             reservaRef,
		IndiceIdempotenciaHMAC: "hmac-sha256:indice-activacion:" + huellaAtestacionDocumental('b'),
		HuellaSolicitudHMAC:    "hmac-sha256:solicitud-activacion:" + huellaAtestacionDocumental('c'),
		Manifiesto:             manifiesto, ConsumoDecision: consumo,
		OrdenConsumoDurableV4Ref: consumo.EfectoRef,
		ActivadaEn:               emitidoEn.Add(-time.Second),
	}
	vinculoActivacion, err := solicitudActivar.VinculoEstable()
	if err != nil {
		t.Fatal(err)
	}
	token := nuevoTokenCercadoEjecucionDocumentalV3Prueba(
		t, "token:cercado:documental:1", 11, vinculoActivacion,
		"clave:atestacion:cercado", 1, "evidencia:cercado:1",
	)
	expiraEn := emitidoEn.Add(5 * time.Minute)
	ordenDespachoConsumida := ordenDespachoDocumentalV3ConsumidaNominalPrueba(
		t, vinculoActivacion, token, emitidoEn, expiraEn, "principal",
	)
	return escenarioAtestacionEjecucionDocumental{
		descriptorPerfil: descriptorPerfil,
		vigente:          vigente, revocada: revocada, vigenteAjena: vigenteAjena,
		render: render, estructural: estructural, semantico: semantico,
		reservaRef: reservaRef, manifiesto: manifiesto,
		consumoDecision: consumo, vinculoActivacion: vinculoActivacion, tokenCercado: token,
		ordenDespachoConsumida: ordenDespachoConsumida,
		limite:                 limite, emitidoEn: emitidoEn, expiraEn: expiraEn,
	}
}

func nuevoCompromisoAtestacionEjecucionDocumental(
	t *testing.T,
	escenario escenarioAtestacionEjecucionDocumental,
	operacion OperacionComponenteDocumental,
	componente DescriptorComponenteDocumentalAtestado,
	operacionRef string,
	reto [32]byte,
	huellaDocumento string,
	tamano uint64,
) CompromisoEjecucionComponenteDocumental {
	t.Helper()
	compromiso, err := NuevoCompromisoEjecucionComponenteDocumental(
		operacionRef, reto, operacion, escenario.descriptorPerfil, escenario.vigente, componente,
		escenario.ordenDespachoConsumida,
		"borrador:documental:1", hmacAtestacionDocumental(), huellaDocumento,
		tamano, escenario.limite, escenario.expiraEn.Sub(escenario.emitidoEn),
	)
	if err != nil {
		t.Fatal(err)
	}
	return compromiso
}

func identidadAtestacionDocumental(
	t *testing.T,
	componente DescriptorComponenteDocumentalAtestado,
	sufijo, procesoRef string,
	semillaClave byte,
) IdentidadEjecucionComponenteDocumental {
	t.Helper()
	identidad, err := NuevaIdentidadEjecucionComponenteDocumental(
		"carga:"+sufijo, procesoRef, componente.DominioConfianzaRef(), "clave:"+sufijo,
		huellaAtestacionDocumental(semillaClave),
		componente.Componente().HuellaArtefactoSHA256(),
	)
	if err != nil {
		t.Fatal(err)
	}
	return identidad
}

func nuevoReciboAtestacionDocumental(
	t *testing.T,
	compromiso CompromisoEjecucionComponenteDocumental,
	sobre SobreReciboEjecucionDocumentalCrudo,
	reciboRef string,
	resultado ResultadoEjecucionComponenteDocumental,
	huella string,
	tamano uint64,
	identidad IdentidadEjecucionComponenteDocumental,
	instante time.Time,
) ReciboEjecucionComponenteDocumentalNominal {
	t.Helper()
	recibo, err := NuevoReciboEjecucionComponenteDocumentalNominal(
		compromiso, sobre, reciboRef, resultado, huella, tamano, identidad, instante,
	)
	if err != nil {
		t.Fatal(err)
	}
	return recibo
}

func sobreAtestacionDocumental(t *testing.T, semilla byte) SobreReciboEjecucionDocumentalCrudo {
	t.Helper()
	sobre, err := NuevoSobreReciboEjecucionDocumentalCrudo(bytes.Repeat([]byte{semilla}, 32))
	if err != nil {
		t.Fatal(err)
	}
	return sobre
}

func retoAtestacionDocumental(semilla byte) [32]byte {
	var reto [32]byte
	for indice := range reto {
		reto[indice] = semilla
	}
	return reto
}

type verificadorOrdenDespachoDocumentalV3Prueba struct {
	sufijo       string
	comprobadaEn time.Time
}

func (v *verificadorOrdenDespachoDocumentalV3Prueba) VerificarOrdenDespachoDocumentalV3(
	ctx context.Context,
	solicitud SolicitudComprobarOrdenDespachoDocumentalV3,
) (ResultadoCrudoVerificacionOrdenDespachoDocumentalV3, error) {
	if v == nil || ctx == nil {
		return ResultadoCrudoVerificacionOrdenDespachoDocumentalV3{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	select {
	case <-ctx.Done():
		return ResultadoCrudoVerificacionOrdenDespachoDocumentalV3{}, ctx.Err()
	default:
	}
	material, err := solicitud.MaterialCrudo()
	if err != nil {
		return ResultadoCrudoVerificacionOrdenDespachoDocumentalV3{}, err
	}
	cercado, inicio, reclamacion, err := material.Pruebas()
	if err != nil {
		return ResultadoCrudoVerificacionOrdenDespachoDocumentalV3{}, err
	}
	for _, prueba := range []PruebaCrudaAtestacionDespachoDocumentalV3{
		cercado, inicio, reclamacion,
	} {
		_, _, _, claveRef, revision, errPerfil := prueba.Perfil()
		mensaje, errMensaje := prueba.MensajeCanonico()
		sobre, errSobre := prueba.SobreCriptografico()
		if errPerfil != nil || errMensaje != nil || errSobre != nil ||
			!hmac.Equal(sobre, firmarAtestacionDocumentalV3Prueba(mensaje, claveRef, revision)) {
			return ResultadoCrudoVerificacionOrdenDespachoDocumentalV3{}, ErrOrdenDespachoDocumentalV3Invalida
		}
	}
	mensaje, err := material.MensajeCanonico()
	if err != nil {
		return ResultadoCrudoVerificacionOrdenDespachoDocumentalV3{}, err
	}
	_, _, _, claveRef, revision, _ := cercado.Perfil()
	atestacionResultado := firmarAtestacionDocumentalV3Prueba(mensaje, claveRef, revision)
	return NuevoResultadoCrudoVerificacionOrdenDespachoDocumentalV3(
		solicitud, "comprobacion:kms:"+v.sufijo, "evidencia:kms:"+v.sufijo,
		huellaBytesFormatoDocumental(atestacionResultado), v.comprobadaEn,
	)
}

type consumidorOrdenDespachoDocumentalV3Prueba struct {
	sufijo             string
	consumidaEn        time.Time
	intentosCAS        int
	estadosPersistidos int
}

func (c *consumidorOrdenDespachoDocumentalV3Prueba) ReleerYConsumirOrdenDespachoDocumentalV3(
	ctx context.Context,
	solicitud SolicitudComprobarOrdenDespachoDocumentalV3,
	resultado ResultadoCrudoVerificacionOrdenDespachoDocumentalV3,
) (EstadoCrudoOrdenDespachoDocumentalV3, error) {
	if c == nil || ctx == nil {
		return EstadoCrudoOrdenDespachoDocumentalV3{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	select {
	case <-ctx.Done():
		return EstadoCrudoOrdenDespachoDocumentalV3{}, ctx.Err()
	default:
	}
	// Representa la barrera que el adaptador debe ejecutar dentro de la misma
	// transaccion, antes de tocar la fila reclamada.
	if resultado.ValidarPara(solicitud) != nil {
		return EstadoCrudoOrdenDespachoDocumentalV3{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	c.intentosCAS++
	estado, err := NuevoEstadoCrudoOrdenDespachoDocumentalV3(
		solicitud, resultado, "estado:consumido:"+c.sufijo, "auditoria:consumo:"+c.sufijo,
		"consumo:despacho:"+c.sufijo, "outbox:consumo:"+c.sufijo, 3, c.consumidaEn,
	)
	if err == nil {
		c.estadosPersistidos++
	}
	return estado, err
}

func ordenDespachoDocumentalV3ConsumidaNominalPrueba(
	t *testing.T,
	vinculo VinculoEstableActivacionDocumentalV3,
	token TokenCercadoEjecucionDocumentalV3Nominal,
	comprobadaEn, expiraEn time.Time,
	sufijo string,
) OrdenDespachoDocumentalV3ConsumidaNominal {
	t.Helper()
	inicio := comprobadaEn.Add(-2 * time.Second)
	solicitudInicio := SolicitudIniciarEfectoDocumentalV3{
		VinculoActivacion: vinculo, Token: token, IniciadoEn: inicio,
	}
	inicioRef := "inicio:" + sufijo
	auditoriaInicioRef := "auditoria:inicio:" + sufijo
	outboxInicioRef := "outbox:inicio:" + sufijo
	evidenciaInicioRef := "atestacion:inicio:" + sufijo
	mensajeInicio, err := MensajeCanonicoAtestacionInicioEfectoDocumentalV3(
		solicitudInicio, inicioRef, 1, auditoriaInicioRef, outboxInicioRef,
		evidenciaInicioRef,
	)
	if err != nil {
		t.Fatal(err)
	}
	pruebaInicio, err := NuevaPruebaCrudaAtestacionDespachoDocumentalV3(
		AlgoritmoSelloEvidenciaHMACSHA256V3, AudienciaAtestacionInicioEfectoV3,
		ContextoAtestacionInicioEfectoV3, token.claveAtestacionRef,
		token.revisionClave, evidenciaInicioRef, mensajeInicio,
		firmarAtestacionDocumentalV3Prueba(mensajeInicio, token.claveAtestacionRef, token.revisionClave),
	)
	if err != nil {
		t.Fatal(err)
	}
	recibo, err := NuevoReciboInicioEfectoDocumentalV3Nominal(
		solicitudInicio, inicioRef, 1, auditoriaInicioRef, outboxInicioRef,
		pruebaInicio,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitudReclamar := SolicitudReclamarOrdenDespachoDocumentalV3{
		ReclamacionRef: "reclamacion:despacho:" + sufijo,
		InicioRef:      "inicio:" + sufijo,
		OutboxRef:      "outbox:inicio:" + sufijo,
		ConsumidorRef:  "consumidor:despacho:" + sufijo,
		ReclamadaEn:    comprobadaEn.Add(-time.Second),
		ExpiraEn:       expiraEn,
	}
	auditoriaReclamacionRef := "auditoria:reclamacion:" + sufijo
	evidenciaReclamacionRef := "atestacion:reclamacion:" + sufijo
	mensajeReclamacion, err := MensajeCanonicoAtestacionReclamacionDespachoDocumentalV3(
		recibo, solicitudReclamar, 2, auditoriaReclamacionRef, evidenciaReclamacionRef,
	)
	if err != nil {
		t.Fatal(err)
	}
	pruebaReclamacion, err := NuevaPruebaCrudaAtestacionDespachoDocumentalV3(
		AlgoritmoSelloEvidenciaHMACSHA256V3, AudienciaAtestacionReclamacionV3,
		ContextoAtestacionReclamacionV3, token.claveAtestacionRef,
		token.revisionClave, evidenciaReclamacionRef, mensajeReclamacion,
		firmarAtestacionDocumentalV3Prueba(
			mensajeReclamacion, token.claveAtestacionRef, token.revisionClave,
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	orden, err := NuevaOrdenDespachoDocumentalV3Nominal(
		recibo, solicitudReclamar, 2, auditoriaReclamacionRef, pruebaReclamacion,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitudComprobar, err := NuevaSolicitudComprobarOrdenDespachoDocumentalV3(
		orden, vinculo, token,
	)
	if err != nil {
		t.Fatal(err)
	}
	verificador := &verificadorOrdenDespachoDocumentalV3Prueba{
		sufijo: sufijo, comprobadaEn: comprobadaEn,
	}
	resultado, err := verificador.VerificarOrdenDespachoDocumentalV3(
		context.Background(), solicitudComprobar,
	)
	if err != nil {
		t.Fatal(err)
	}
	consumidor := &consumidorOrdenDespachoDocumentalV3Prueba{
		sufijo: sufijo, consumidaEn: comprobadaEn,
	}
	estado, err := consumidor.ReleerYConsumirOrdenDespachoDocumentalV3(
		context.Background(), solicitudComprobar, resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	ordenConsumida, err := NuevaOrdenDespachoDocumentalV3ConsumidaNominal(
		solicitudComprobar, resultado, estado,
	)
	if err != nil {
		t.Fatal(err)
	}
	return ordenConsumida
}

func hmacAtestacionDocumental() string {
	return "hmac-sha256:clave-documental:" + huellaAtestacionDocumental('a')
}

func huellaAtestacionDocumental(semilla byte) string {
	return strings.Repeat(string(semilla), 64)
}

func claveGestionadaAtestacionDocumentalV3Prueba(referencia string, revision uint64) []byte {
	huella := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", referencia, revision)))
	return huella[:]
}

func firmarAtestacionDocumentalV3Prueba(
	mensaje []byte,
	claveRef string,
	revision uint64,
) []byte {
	firmante := hmac.New(sha256.New, claveGestionadaAtestacionDocumentalV3Prueba(claveRef, revision))
	_, _ = firmante.Write(mensaje)
	return firmante.Sum(nil)
}

func nuevoTokenCercadoEjecucionDocumentalV3Prueba(
	t *testing.T,
	valor string,
	secuencia uint64,
	vinculo VinculoEstableActivacionDocumentalV3,
	claveRef string,
	revision uint64,
	evidenciaRef string,
) TokenCercadoEjecucionDocumentalV3Nominal {
	t.Helper()
	provisional, err := NuevoTokenCercadoEjecucionDocumentalV3Nominal(
		valor, secuencia, vinculo, claveRef, revision, bytes.Repeat([]byte{1}, sha256.Size),
		evidenciaRef,
	)
	if err != nil {
		t.Fatal(err)
	}
	mensaje := serializarAtestacionTokenCercadoDocumentalV3(vinculo, provisional)
	mac := firmarAtestacionDocumentalV3Prueba(mensaje, claveRef, revision)
	token, err := NuevoTokenCercadoEjecucionDocumentalV3Nominal(
		valor, secuencia, vinculo, claveRef, revision, mac, evidenciaRef,
	)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

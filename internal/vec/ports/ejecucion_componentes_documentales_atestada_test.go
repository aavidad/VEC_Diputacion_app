package ports

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
		{"token", func(c *CompromisoEjecucionComponenteDocumental) {
			c.tokenCercado = TokenCercadoEjecucionDocumentalV3{}
		}},
		{"verificacion token", func(c *CompromisoEjecucionComponenteDocumental) {
			c.verificacionToken = ResultadoVerificacionTokenCercadoDocumentalV3{}
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
				escenario.vigente, prueba.componente, escenario.reservaRef,
				escenario.manifiesto, escenario.consumoDecision, escenario.tokenCercado,
				escenario.verificacionToken,
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
				prueba.situacion, escenario.render, escenario.reservaRef,
				escenario.manifiesto, escenario.consumoDecision, escenario.tokenCercado,
				escenario.verificacionToken,
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

func TestCompromisoEjecucionDocumentalRechazaTokenDeOtraReservaAunqueRecalculeDigest(t *testing.T) {
	escenario := nuevoEscenarioAtestacionEjecucionDocumental(t)
	compromiso := nuevoCompromisoAtestacionEjecucionDocumental(
		t, escenario, OperacionRenderizadoDocumental, escenario.render,
		"operacion:render:1", retoAtestacionDocumental(1), "", 0,
	)
	tokenAjeno, err := NuevoTokenCercadoEjecucionDocumentalV3(
		"token:cercado:documental:otro", 12, "reserva:documental:otra",
		escenario.manifiesto, escenario.consumoDecision, "clave:atestacion:cercado",
		bytes.Repeat([]byte{0x6b}, 32), "evidencia:cercado:otra",
	)
	if err != nil {
		t.Fatal(err)
	}
	compromiso.tokenCercado = tokenAjeno
	compromiso.secuenciaCercado = tokenAjeno.Secuencia()
	compromiso.huellaVinculoSHA256 = tokenAjeno.HuellaVinculoSHA256()
	compromiso.huellaCompromisoSHA256 = compromiso.calcularHuella()
	if compromiso.Validar() == nil {
		t.Fatal("un token valido para otra reserva autorizo el compromiso")
	}
}

func TestSobreCOSEEsOpacoLimitadoYCopiadoDefensivamente(t *testing.T) {
	original := bytes.Repeat([]byte{0xa1}, 32)
	sobre, err := NuevoSobreReciboEjecucionDocumental(original)
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
		if _, err := NuevoSobreReciboEjecucionDocumental(contenido); !errors.Is(
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
	for nombre, valor := range map[string]any{
		"compromiso": compromiso,
		"sobre":      sobre,
	} {
		if _, err := json.Marshal(valor); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
			t.Fatalf("%s se serializo por JSON: %v", nombre, err)
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
	var sobreRestaurado SobreReciboEjecucionDocumental
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
	compromiso.verificacionToken.verificadaEn = compromiso.emitidoEn
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
	_ = map[DatosReciboEjecucionComponenteDocumentalVerificado]struct{}{datosRecibo: {}}
	tipoDatos := reflect.TypeOf(datosRecibo)
	for _, tipoProhibido := range []reflect.Type{
		reflect.TypeOf(CompromisoEjecucionComponenteDocumental{}),
		reflect.TypeOf(TokenCercadoEjecucionDocumentalV3{}),
		reflect.TypeOf(SobreReciboEjecucionDocumental{}),
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
	token, err := NuevoTokenCercadoEjecucionDocumentalV3(
		"token:cercado:documental:compartido", 13, escenario.reservaRef, manifiesto,
		consumo, "clave:atestacion:cercado", bytes.Repeat([]byte{0x5c}, 32),
		"evidencia:cercado:compartido",
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitudVerificacion, err := NuevaSolicitudVerificacionTokenCercadoDocumentalV3(
		escenario.reservaRef, manifiesto, consumo, token,
	)
	if err != nil {
		t.Fatal(err)
	}
	verificacionToken, err := NuevoResultadoVerificacionTokenCercadoDocumentalV3(
		solicitudVerificacion, "verificacion:cercado:compartida", escenario.emitidoEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	escenario.semantico = semantico
	escenario.manifiesto = manifiesto
	escenario.consumoDecision = consumo
	escenario.tokenCercado = token
	escenario.verificacionToken = verificacionToken
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
	if _, err := NuevoReciboEjecucionComponenteDocumentalVerificado(
		compromiso, sobre, "recibo:semantica:1", ResultadoEstructuraDocumentalConforme,
		huellaAtestacionDocumental('9'), 128, identidad, instante,
	); !errors.Is(err, ErrReciboEjecucionDocumentalInvalido) {
		t.Fatalf("resultado de otro rol aceptado: %v", err)
	}
	if _, err := NuevoReciboEjecucionComponenteDocumentalVerificado(
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
	if _, err := NuevoReciboEjecucionComponenteDocumentalVerificado(
		compromiso, sobre, "recibo:semantica:1", ResultadoSemanticaDocumentalEquivalente,
		huellaAtestacionDocumental('9'), 128, medicionAjena, instante,
	); !errors.Is(err, ErrReciboEjecucionDocumentalInvalido) {
		t.Fatalf("medicion ajena al artefacto aceptada: %v", err)
	}
	recibo := nuevoReciboAtestacionDocumental(
		t, compromiso, sobre, "recibo:semantica:1", ResultadoSemanticaDocumentalEquivalente,
		huellaAtestacionDocumental('9'), 128, identidad, instante,
	)
	recibo.tamanoSalida++
	if recibo.Validar() == nil {
		t.Fatal("un recibo manipulado conservo autoridad")
	}
	if (ReciboEjecucionComponenteDocumentalVerificado{}).Validar() == nil ||
		(IdentidadEjecucionComponenteDocumental{}).Validar() == nil {
		t.Fatal("un valor cero obtuvo semantica positiva")
	}
}

type escenarioAtestacionEjecucionDocumental struct {
	descriptorPerfil  DescriptorPerfilDocumental
	vigente           domain.SituacionOperativaPerfilDocumental
	revocada          domain.SituacionOperativaPerfilDocumental
	vigenteAjena      domain.SituacionOperativaPerfilDocumental
	render            DescriptorComponenteDocumentalAtestado
	estructural       DescriptorComponenteDocumentalAtestado
	semantico         DescriptorComponenteDocumentalAtestado
	reservaRef        string
	manifiesto        ManifiestoEjecucionDocumentalV3
	consumoDecision   ConsumoDecisionEjecucionDocumentalV3
	tokenCercado      TokenCercadoEjecucionDocumentalV3
	verificacionToken ResultadoVerificacionTokenCercadoDocumentalV3
	limite            uint64
	emitidoEn         time.Time
	expiraEn          time.Time
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
	token, err := NuevoTokenCercadoEjecucionDocumentalV3(
		"token:cercado:documental:1", 11, reservaRef, manifiesto, consumo,
		"clave:atestacion:cercado", bytes.Repeat([]byte{0x7a}, 32), "evidencia:cercado:1",
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitudVerificacion, err := NuevaSolicitudVerificacionTokenCercadoDocumentalV3(
		reservaRef, manifiesto, consumo, token,
	)
	if err != nil {
		t.Fatal(err)
	}
	verificacionToken, err := NuevoResultadoVerificacionTokenCercadoDocumentalV3(
		solicitudVerificacion, "verificacion:cercado:documental:1", emitidoEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	return escenarioAtestacionEjecucionDocumental{
		descriptorPerfil: descriptorPerfil,
		vigente:          vigente, revocada: revocada, vigenteAjena: vigenteAjena,
		render: render, estructural: estructural, semantico: semantico,
		reservaRef: reservaRef, manifiesto: manifiesto,
		consumoDecision: consumo, tokenCercado: token,
		verificacionToken: verificacionToken,
		limite:            limite, emitidoEn: emitidoEn, expiraEn: emitidoEn.Add(5 * time.Minute),
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
		escenario.reservaRef, escenario.manifiesto, escenario.consumoDecision, escenario.tokenCercado,
		escenario.verificacionToken,
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
	sobre SobreReciboEjecucionDocumental,
	reciboRef string,
	resultado ResultadoEjecucionComponenteDocumental,
	huella string,
	tamano uint64,
	identidad IdentidadEjecucionComponenteDocumental,
	instante time.Time,
) ReciboEjecucionComponenteDocumentalVerificado {
	t.Helper()
	recibo, err := NuevoReciboEjecucionComponenteDocumentalVerificado(
		compromiso, sobre, reciboRef, resultado, huella, tamano, identidad, instante,
	)
	if err != nil {
		t.Fatal(err)
	}
	return recibo
}

func sobreAtestacionDocumental(t *testing.T, semilla byte) SobreReciboEjecucionDocumental {
	t.Helper()
	sobre, err := NuevoSobreReciboEjecucionDocumental(bytes.Repeat([]byte{semilla}, 32))
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

func hmacAtestacionDocumental() string {
	return "hmac-sha256:clave-documental:" + huellaAtestacionDocumental('a')
}

func huellaAtestacionDocumental(semilla byte) string {
	return strings.Repeat(string(semilla), 64)
}

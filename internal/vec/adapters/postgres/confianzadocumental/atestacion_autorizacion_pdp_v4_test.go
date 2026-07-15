package confianzadocumental

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/veraison/go-cose"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

type escenarioAtestacionAutorizacionPDPV4 struct {
	escenario escenarioAutoridadInternaEjecucionDocumentalV4
	vinculo   ports.SolicitudVinculadaAutorizacionEjecucionDocumentalV4
	material  materialFirmaCOSEPrueba
	servicio  *Servicio
	cabecera  domain.CabeceraAtestacionAutorizacionV1
	payload   []byte
	solicitud SolicitudVerificacionCOSESign1
	sobre     ports.SobreCriptograficoDocumentalCrudoV4
}

type relojContadorAtestacionPDP struct {
	instante time.Time
	llamadas atomic.Int32
}

type relojSecuenciaAtestacionPDP struct {
	instantes []time.Time
	indice    atomic.Int32
}

func (r *relojSecuenciaAtestacionPDP) Ahora() time.Time {
	indice := int(r.indice.Add(1)) - 1
	if indice < 0 || indice >= len(r.instantes) {
		return time.Time{}
	}
	return r.instantes[indice]
}

func (r *relojContadorAtestacionPDP) Ahora() time.Time {
	r.llamadas.Add(1)
	return r.instante
}

func TestServicioEmiteAutoridadSoloTrasVECAD1COSEPDPExacto(t *testing.T) {
	caso := nuevoEscenarioAtestacionAutorizacionPDPV4(t)
	reloj := &relojContadorAtestacionPDP{instante: caso.escenario.emitidaEn}
	caso.servicio.reloj = reloj

	autoridad, err := caso.servicio.EmitirAutoridadInternaEjecucionDocumentalV4(
		context.Background(), caso.vinculo, caso.cabecera, caso.sobre,
	)
	if err != nil {
		t.Fatalf("emitir autoridad atestada: %v", err)
	}
	if reloj.llamadas.Load() != 2 {
		t.Fatalf("se consulto %d veces el reloj interno; se esperaban dos", reloj.llamadas.Load())
	}
	if autoridad.ValidarEn(caso.escenario.emitidaEn) != nil ||
		!autoridad.emitidaEn.Equal(caso.escenario.emitidaEn) ||
		!autoridad.pruebaPDP.verificadaEn.Equal(autoridad.emitidaEn) ||
		autoridad.cabeceraPDP != caso.cabecera ||
		autoridad.pruebaPDP.huellaPayloadSHA256 != huellaBytesDocumentales(caso.payload) {
		t.Fatal("la autoridad no conservo instante, cabecera y payload exactos")
	}

	texto := fmt.Sprintf("%v|%+v|%#v", autoridad, autoridad, autoridad)
	var salida bytes.Buffer
	slog.New(slog.NewTextHandler(&salida, nil)).Info("autoridad", "valor", autoridad)
	for _, sensible := range []string{
		string(caso.material.claveID), caso.cabecera.Suite,
		caso.cabecera.Audiencia, caso.escenario.decision.PrincipalID,
	} {
		if strings.Contains(texto, sensible) || strings.Contains(salida.String(), sensible) {
			t.Fatalf("fmt/slog expuso %q", sensible)
		}
	}
}

func TestServicioDeniegaRetrocesoDelRelojDuranteLaVerificacionPDP(t *testing.T) {
	base := nuevoEscenarioAtestacionAutorizacionPDPV4(t)
	reloj := &relojSecuenciaAtestacionPDP{instantes: []time.Time{
		base.escenario.emitidaEn.Add(time.Microsecond),
		base.escenario.emitidaEn,
	}}
	base.servicio.reloj = reloj
	autoridad, err := base.servicio.EmitirAutoridadInternaEjecucionDocumentalV4(
		context.Background(), base.vinculo, base.cabecera, base.sobre,
	)
	comprobarDenegacionAtestacionPDP(t, autoridad, err)
}

func TestEvidenciaDurablePDPConservaBytesCopiasYPermiteReverificar(t *testing.T) {
	base := nuevoEscenarioAtestacionAutorizacionPDPV4(t)
	autoridad, err := base.servicio.EmitirAutoridadInternaEjecucionDocumentalV4(
		context.Background(), base.vinculo, base.cabecera, base.sobre,
	)
	if err != nil {
		t.Fatal(err)
	}
	instanteAplicacion := base.escenario.emitidaEn.Add(time.Microsecond)
	solicitud, evidencia, err := autoridad.PrepararAplicacionExactaConEvidenciaEn(
		base.escenario.decision.DecisionRef,
		base.escenario.expectativa.HuellaPlanSHA256,
		base.escenario.expectativa.EfectoRef,
		instanteAplicacion,
	)
	if err != nil || evidencia.Validar() != nil {
		t.Fatalf("preparar solicitud con evidencia durable: %v", err)
	}
	if solicitud.ValidarContraEn(
		base.escenario.decision.DecisionRef,
		base.escenario.expectativa.HuellaPlanSHA256,
		base.escenario.expectativa.EfectoRef,
		instanteAplicacion,
	) != nil {
		t.Fatal("la solicitud conjunta perdio su terna")
	}

	metadatos, err := evidencia.Metadatos()
	if err != nil || metadatos.DecisionRef != base.escenario.decision.DecisionRef ||
		metadatos.HuellaPlanSHA256 != base.escenario.expectativa.HuellaPlanSHA256 ||
		metadatos.EfectoRef != base.escenario.expectativa.EfectoRef ||
		metadatos.HuellaSolicitudVinculadaSHA256 != autoridad.huellaVinculoSHA256 ||
		metadatos.Suite != base.cabecera.Suite || metadatos.ClaveID != base.cabecera.ClaveID ||
		metadatos.AudienciaDespliegue != base.cabecera.Audiencia ||
		metadatos.AlgoritmoCOSE != AlgoritmoCOSEDocumentalEdDSA ||
		metadatos.AudienciaCOSE != AudienciaCOSEAtestacionAutorizacionPDP ||
		metadatos.EstadoConfianza != EstadoConfianzaClaveDocumentalActiva ||
		!huellaSHA256DocumentalValida(metadatos.HuellaPreimagenRecursoSHA256) ||
		metadatos.HuellaContextoRecursoSHA256 !=
			base.escenario.decision.ContextoRecursoHuellaSHA256 ||
		!huellaSHA256DocumentalValida(metadatos.HuellaAmbitosRecursoSHA256) {
		t.Fatalf("metadatos durables incompletos: %v, %v", metadatos, err)
	}
	preimagenBytes, err := evidencia.PreimagenRecursoCanonica()
	if err != nil {
		t.Fatal(err)
	}
	preimagen, err := ports.InterpretarPreimagenRecursoAutorizacionEjecucionDocumentalV4(
		preimagenBytes,
		metadatos.HuellaPreimagenRecursoSHA256,
	)
	if err != nil {
		t.Fatalf("interpretar preimagen durable: %v", err)
	}
	recurso, err := preimagen.RecursoCanonico()
	if err != nil || !reflect.DeepEqual(recurso, base.escenario.expectativa.Recurso) {
		t.Fatalf("preimagen durable no conserva el recurso exacto: %#v, %v", recurso, err)
	}
	payload, err := evidencia.PayloadVECAD1()
	if err != nil || !bytes.Equal(payload, base.payload) {
		t.Fatalf("payload durable distinto: %v", err)
	}
	sobreBytes, err := evidencia.SobreCOSESign1()
	if err != nil {
		t.Fatal(err)
	}
	sobreOriginal, _ := base.sobre.COSESign1()
	if !bytes.Equal(sobreBytes, sobreOriginal) {
		t.Fatal("sobre durable distinto del verificado")
	}
	canonico, err := evidencia.SerializacionCanonicaParaPersistencia()
	if err != nil || len(canonico) < 8 ||
		binary.BigEndian.Uint64(canonico[len(canonico)-8:]) != uint64(len(canonico)) {
		t.Fatalf("serializacion durable no canonica: %v", err)
	}
	huella, err := evidencia.HuellaSHA256()
	if err != nil || huella != huellaBytesDocumentales(canonico) ||
		huella != metadatos.HuellaEvidenciaDurableSHA256 {
		t.Fatalf("huella durable incoherente: %q, %v", huella, err)
	}

	// Cada salida binaria es una copia; modificarla no altera la evidencia ni
	// la autoridad que conserva su propia copia.
	payload[0] ^= 0xff
	sobreBytes[0] ^= 0xff
	preimagenBytes[0] ^= 0xff
	canonico[0] ^= 0xff
	payloadSegundo, _ := evidencia.PayloadVECAD1()
	sobreSegundo, _ := evidencia.SobreCOSESign1()
	canonicoSegundo, _ := evidencia.SerializacionCanonicaParaPersistencia()
	preimagenSegunda, _ := evidencia.PreimagenRecursoCanonica()
	if !bytes.Equal(payloadSegundo, base.payload) || !bytes.Equal(sobreSegundo, sobreOriginal) ||
		huellaBytesDocumentales(preimagenSegunda) != metadatos.HuellaPreimagenRecursoSHA256 ||
		huellaBytesDocumentales(canonicoSegundo) != huella || autoridad.ValidarEn(instanteAplicacion) != nil {
		t.Fatal("un accessor expuso un alias mutable")
	}

	// Reinicio real: primero se interpreta el unico registro persistido y solo
	// despues se extraen payload y sobre para la reverificacion independiente.
	rehidratada, err := interpretarEvidenciaHistoricaAtestacionPDPV4(canonicoSegundo, huella)
	if err != nil || rehidratada.Validar() != nil {
		t.Fatalf("rehidratar evidencia historica: %v", err)
	}
	payloadRehidratado, err := rehidratada.PayloadVECAD1()
	if err != nil {
		t.Fatal(err)
	}
	sobreRehidratadoBytes, err := rehidratada.SobreCOSESign1()
	if err != nil {
		t.Fatal(err)
	}
	sobreRehidratado, err := ports.NuevoSobreCriptograficoDocumentalCrudoV4(
		sobreRehidratadoBytes,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitudRehidratada, err := NuevaSolicitudVerificacionCOSESign1(
		payloadRehidratado, AudienciaCOSEAtestacionAutorizacionPDP,
	)
	if err != nil {
		t.Fatal(err)
	}
	prueba, err := base.servicio.VerificarCOSESign1(
		context.Background(), solicitudRehidratada, sobreRehidratado,
	)
	if err != nil || prueba.ValidarPara(solicitudRehidratada, sobreRehidratado) != nil {
		t.Fatalf("la evidencia persistible no permitio reverificar: %v", err)
	}

	for nombre, valor := range map[string]any{"evidencia": evidencia, "metadatos": metadatos} {
		if _, err := json.Marshal(valor); !errors.Is(
			err, ErrSerializacionGeneralEvidenciaDurableAtestacionPDPV4Prohibida,
		) {
			t.Fatalf("%s se serializo por JSON: %v", nombre, err)
		}
		serializadorTexto, okTexto := valor.(interface{ MarshalText() ([]byte, error) })
		serializadorBinario, okBinario := valor.(interface{ MarshalBinary() ([]byte, error) })
		if !okTexto || !okBinario {
			t.Fatalf("%s no bloquea todos los serializadores generales", nombre)
		}
		if _, err := serializadorTexto.MarshalText(); !errors.Is(
			err, ErrSerializacionGeneralEvidenciaDurableAtestacionPDPV4Prohibida,
		) {
			t.Fatalf("%s se serializo como texto: %v", nombre, err)
		}
		if _, err := serializadorBinario.MarshalBinary(); !errors.Is(
			err, ErrSerializacionGeneralEvidenciaDurableAtestacionPDPV4Prohibida,
		) {
			t.Fatalf("%s se serializo como binario generico: %v", nombre, err)
		}
		texto := fmt.Sprintf("%v|%+v|%#v", valor, valor, valor)
		if strings.Contains(texto, base.cabecera.ClaveID) ||
			strings.Contains(texto, base.escenario.decision.PrincipalID) {
			t.Fatalf("%s filtro datos: %s", nombre, texto)
		}
	}
}

func TestEvidenciaDurablePDPDetectaTodaManipulacionYNoAlteraAutoridad(t *testing.T) {
	base := nuevoEscenarioAtestacionAutorizacionPDPV4(t)
	autoridad, err := base.servicio.EmitirAutoridadInternaEjecucionDocumentalV4(
		context.Background(), base.vinculo, base.cabecera, base.sobre,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, evidencia, err := autoridad.PrepararAplicacionExactaConEvidenciaEn(
		base.escenario.decision.DecisionRef,
		base.escenario.expectativa.HuellaPlanSHA256,
		base.escenario.expectativa.EfectoRef,
		base.escenario.emitidaEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre string
		mutar  func(*EvidenciaDurableAtestacionAutorizacionPDPV4)
	}{
		{"decision", func(e *EvidenciaDurableAtestacionAutorizacionPDPV4) { e.metadatos.DecisionRef = "decision:otra" }},
		{"plan", func(e *EvidenciaDurableAtestacionAutorizacionPDPV4) {
			e.metadatos.HuellaPlanSHA256 = huellaInternaPrueba('9')
		}},
		{"efecto", func(e *EvidenciaDurableAtestacionAutorizacionPDPV4) { e.metadatos.EfectoRef = "efecto:otro" }},
		{"vinculo", func(e *EvidenciaDurableAtestacionAutorizacionPDPV4) {
			e.metadatos.HuellaSolicitudVinculadaSHA256 = huellaInternaPrueba('8')
		}},
		{"estado", func(e *EvidenciaDurableAtestacionAutorizacionPDPV4) {
			e.metadatos.EstadoConfianza = EstadoConfianzaClaveDocumentalRevocada
		}},
		{"preimagen", func(e *EvidenciaDurableAtestacionAutorizacionPDPV4) {
			e.preimagenRecurso[0] ^= 0xff
		}},
		{"huella preimagen", func(e *EvidenciaDurableAtestacionAutorizacionPDPV4) {
			e.metadatos.HuellaPreimagenRecursoSHA256 = huellaInternaPrueba('6')
		}},
		{"huella contexto recurso", func(e *EvidenciaDurableAtestacionAutorizacionPDPV4) {
			e.metadatos.HuellaContextoRecursoSHA256 = huellaInternaPrueba('5')
		}},
		{"huella ambitos recurso", func(e *EvidenciaDurableAtestacionAutorizacionPDPV4) {
			e.metadatos.HuellaAmbitosRecursoSHA256 = huellaInternaPrueba('4')
		}},
		{"payload", func(e *EvidenciaDurableAtestacionAutorizacionPDPV4) { e.payloadVECAD1[0] ^= 0xff }},
		{"sobre", func(e *EvidenciaDurableAtestacionAutorizacionPDPV4) { e.sobreCOSESign1[0] ^= 0xff }},
		{"canonico", func(e *EvidenciaDurableAtestacionAutorizacionPDPV4) { e.serializacionCanonica[0] ^= 0xff }},
		{"huella", func(e *EvidenciaDurableAtestacionAutorizacionPDPV4) {
			e.metadatos.HuellaEvidenciaDurableSHA256 = huellaInternaPrueba('7')
		}},
	}
	for _, prueba := range casos {
		t.Run(prueba.nombre, func(t *testing.T) {
			manipulada, err := evidencia.clonar()
			if err != nil {
				t.Fatal(err)
			}
			prueba.mutar(&manipulada)
			if !errors.Is(manipulada.Validar(), ErrEvidenciaDurableAtestacionPDPV4Invalida) {
				t.Fatal("evidencia manipulada aceptada")
			}
		})
	}
	if autoridad.ValidarEn(base.escenario.emitidaEn) != nil {
		t.Fatal("la copia manipulada altero la autoridad")
	}
}

func TestParserHistoricoEvidenciaPDPRechazaFormatoHostilEstricto(t *testing.T) {
	base := nuevoEscenarioAtestacionAutorizacionPDPV4(t)
	autoridad, err := base.servicio.EmitirAutoridadInternaEjecucionDocumentalV4(
		context.Background(), base.vinculo, base.cabecera, base.sobre,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, evidencia, err := autoridad.PrepararAplicacionExactaConEvidenciaEn(
		base.escenario.decision.DecisionRef,
		base.escenario.expectativa.HuellaPlanSHA256,
		base.escenario.expectativa.EfectoRef,
		base.escenario.emitidaEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	canonico, _ := evidencia.SerializacionCanonicaParaPersistencia()
	huella, _ := evidencia.HuellaSHA256()

	truncado := append([]byte(nil), canonico[:len(canonico)-1]...)
	longitudEnorme := append([]byte(nil), canonico...)
	posicionPrimeraLongitud := len(esquemaEvidenciaDurableAtestacionPDPV4) + 1 + 2
	binary.BigEndian.PutUint64(
		longitudEnorme[posicionPrimeraLongitud:posicionPrimeraLongitud+8],
		^uint64(0),
	)
	versionDistinta := append([]byte(nil), canonico...)
	versionDistinta[len(esquemaEvidenciaDurableAtestacionPDPV4)+1] = 0
	versionDistinta[len(esquemaEvidenciaDurableAtestacionPDPV4)+2] = 2

	conExtra := append([]byte(nil), canonico[:len(canonico)-8]...)
	var longitudCampo [8]byte
	binary.BigEndian.PutUint64(longitudCampo[:], 1)
	conExtra = append(conExtra, longitudCampo[:]...)
	conExtra = append(conExtra, 'x')
	var longitudTotal [8]byte
	binary.BigEndian.PutUint64(longitudTotal[:], uint64(len(conExtra)+8))
	conExtra = append(conExtra, longitudTotal[:]...)

	mutacionSinActualizarHuella := append([]byte(nil), canonico...)
	mutacionSinActualizarHuella[len(mutacionSinActualizarHuella)/2] ^= 0x01
	casos := []struct {
		nombre    string
		contenido []byte
		huella    string
	}{
		{"truncado", truncado, huellaBytesDocumentales(truncado)},
		{"longitud enorme", longitudEnorme, huellaBytesDocumentales(longitudEnorme)},
		{"version distinta", versionDistinta, huellaBytesDocumentales(versionDistinta)},
		{"campo extra", conExtra, huellaBytesDocumentales(conExtra)},
		{"mutacion sin huella", mutacionSinActualizarHuella, huella},
		{"huella ausente", canonico, ""},
	}
	for _, prueba := range casos {
		t.Run(prueba.nombre, func(t *testing.T) {
			if reconstruida, err := interpretarEvidenciaHistoricaAtestacionPDPV4(
				prueba.contenido, prueba.huella,
			); !errors.Is(err, ErrEvidenciaDurableAtestacionPDPV4Invalida) ||
				reconstruida.Validar() == nil {
				t.Fatalf("formato hostil aceptado: %v", err)
			}
		})
	}
}

func TestParserHistoricoPDPNoConfundeIntegridadConAutenticidad(t *testing.T) {
	base := nuevoEscenarioAtestacionAutorizacionPDPV4(t)
	autoridad, err := base.servicio.EmitirAutoridadInternaEjecucionDocumentalV4(
		context.Background(), base.vinculo, base.cabecera, base.sobre,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, evidencia, err := autoridad.PrepararAplicacionExactaConEvidenciaEn(
		base.escenario.decision.DecisionRef,
		base.escenario.expectativa.HuellaPlanSHA256,
		base.escenario.expectativa.EfectoRef,
		base.escenario.emitidaEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	canonico, _ := evidencia.SerializacionCanonicaParaPersistencia()
	metadatos, _ := evidencia.Metadatos()
	manipulado := append([]byte(nil), canonico...)
	referencia := []byte(metadatos.RevisionConfianza)
	posicion := bytes.Index(manipulado, referencia)
	if posicion < 0 {
		t.Fatal("no se encontro la revision de confianza en el registro canonico")
	}
	manipulado[posicion+len(referencia)-1] = 'x'
	recalculada := huellaBytesDocumentales(manipulado)
	reconstruida, err := interpretarEvidenciaHistoricaAtestacionPDPV4(
		manipulado, recalculada,
	)
	if err != nil || reconstruida.Validar() != nil {
		t.Fatalf("el caso debe seguir siendo integridad sintactica valida: %v", err)
	}
	if reconstruida.coincideConAutoridad(autoridad) {
		t.Fatal("una huella local recalculada se convirtio en autoridad")
	}
	// Esta aceptacion sintactica es intencionada: solo el registro historico
	// autenticado y el cotejo del payload firmado pueden cerrar autenticidad.
}

func TestEvidenciaDurablePDPRechazaReescrituraCanonicaDelRecursoLigadoALaFirma(t *testing.T) {
	base := nuevoEscenarioAtestacionAutorizacionPDPV4(t)
	autoridad, err := base.servicio.EmitirAutoridadInternaEjecucionDocumentalV4(
		context.Background(), base.vinculo, base.cabecera, base.sobre,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, evidencia, err := autoridad.PrepararAplicacionExactaConEvidenciaEn(
		base.escenario.decision.DecisionRef,
		base.escenario.expectativa.HuellaPlanSHA256,
		base.escenario.expectativa.EfectoRef,
		base.escenario.emitidaEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre           string
		original         string
		reemplazo        string
		actualizarPlan   bool
		actualizarEfecto bool
	}{
		{
			"referencia",
			base.escenario.expectativa.Recurso.Referencia,
			base.escenario.expectativa.Recurso.Referencia[:len(base.escenario.expectativa.Recurso.Referencia)-1] + "x",
			false,
			false,
		},
		{
			"ambito",
			base.escenario.expectativa.Recurso.Ambitos["organizacion"],
			"diputacion_granadx",
			false,
			false,
		},
		{
			"plan",
			base.escenario.expectativa.HuellaPlanSHA256,
			huellaInternaPrueba('b'),
			true,
			false,
		},
		{
			"efecto",
			base.escenario.expectativa.EfectoRef,
			base.escenario.expectativa.EfectoRef[:len(base.escenario.expectativa.EfectoRef)-1] + "x",
			false,
			true,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			manipulada, err := evidencia.clonar()
			if err != nil {
				t.Fatal(err)
			}
			if len(caso.original) != len(caso.reemplazo) ||
				bytes.Count(manipulada.preimagenRecurso, []byte(caso.original)) != 1 {
				t.Fatal("vector de mutacion no univoco o de longitud distinta")
			}
			manipulada.preimagenRecurso = bytes.Replace(
				manipulada.preimagenRecurso,
				[]byte(caso.original),
				[]byte(caso.reemplazo),
				1,
			)
			huellaPreimagen := huellaBytesDocumentales(manipulada.preimagenRecurso)
			preimagen, err := ports.InterpretarPreimagenRecursoAutorizacionEjecucionDocumentalV4(
				manipulada.preimagenRecurso,
				huellaPreimagen,
			)
			if err != nil {
				t.Fatalf("la preimagen adversarial debe seguir siendo canonica: %v", err)
			}
			huellaContexto, _ := preimagen.HuellaContextoRecursoSHA256()
			huellaAmbitos, _ := preimagen.HuellaAmbitosSHA256()
			manipulada.metadatos.HuellaPreimagenRecursoSHA256 = huellaPreimagen
			manipulada.metadatos.HuellaContextoRecursoSHA256 = huellaContexto
			manipulada.metadatos.HuellaAmbitosRecursoSHA256 = huellaAmbitos
			if caso.actualizarPlan {
				manipulada.metadatos.HuellaPlanSHA256 = caso.reemplazo
			}
			if caso.actualizarEfecto {
				manipulada.metadatos.EfectoRef = caso.reemplazo
			}
			manipulada.serializacionCanonica = manipulada.calcularSerializacionCanonica()
			manipulada.metadatos.HuellaEvidenciaDurableSHA256 = huellaBytesDocumentales(
				manipulada.serializacionCanonica,
			)
			if !errors.Is(
				manipulada.Validar(),
				ErrEvidenciaDurableAtestacionPDPV4Invalida,
			) {
				t.Fatal("la preimagen reescrita se atribuyo al payload firmado")
			}
		})
	}
}

func TestServicioDeniegaDecisionActorCamposPlanOEfectoDistintosDeLaFirma(t *testing.T) {
	base := nuevoEscenarioAtestacionAutorizacionPDPV4(t)
	casos := []struct {
		nombre string
		mutar  func(*escenarioAutoridadInternaEjecucionDocumentalV4)
	}{
		{
			"decision",
			func(e *escenarioAutoridadInternaEjecucionDocumentalV4) {
				e.decision.DecisionRef = "decision:documental:v4:otra"
			},
		},
		{
			"actor",
			func(e *escenarioAutoridadInternaEjecucionDocumentalV4) {
				vinculo, err := pruebasvec.NuevoVinculoGenerico(
					e.decision.EmitidaEn.Add(time.Microsecond),
				)
				if err != nil {
					t.Fatalf("crear actor alternativo: %v", err)
				}
				e.decision.VinculoAutenticacionActor = vinculo
			},
		},
		{
			"campos",
			func(e *escenarioAutoridadInternaEjecucionDocumentalV4) {
				e.decision.CamposPermitidos = []string{"documento.otro"}
			},
		},
		{
			"plan",
			func(e *escenarioAutoridadInternaEjecucionDocumentalV4) {
				e.expectativa.Recurso.Atributos[ports.AtributoAutorizacionDocumentalHuellaPlanSHA256] =
					huellaInternaPrueba('b')
				huella, err := e.expectativa.Recurso.HuellaContextoAutorizacionSHA256()
				if err != nil {
					t.Fatalf("recalcular recurso con otro plan: %v", err)
				}
				e.decision.ContextoRecursoHuellaSHA256 = huella
			},
		},
		{
			"efecto",
			func(e *escenarioAutoridadInternaEjecucionDocumentalV4) {
				e.expectativa.Recurso.Atributos[ports.AtributoAutorizacionDocumentalEfectoRef] =
					"efecto:documental:v4:otro"
				huella, err := e.expectativa.Recurso.HuellaContextoAutorizacionSHA256()
				if err != nil {
					t.Fatalf("recalcular recurso con otro efecto: %v", err)
				}
				e.decision.ContextoRecursoHuellaSHA256 = huella
			},
		},
	}

	for _, prueba := range casos {
		t.Run(prueba.nombre, func(t *testing.T) {
			escenario := nuevoEscenarioAutoridadInternaEjecucionDocumentalV4(t)
			prueba.mutar(&escenario)
			reconstruirEscenarioAutorizacionPDPV4(t, &escenario)
			vinculo, err := ports.NuevaSolicitudVinculadaAutorizacionEjecucionDocumentalV4(
				escenario.evidencia, escenario.expectativa, escenario.emitidaEn,
			)
			if err != nil {
				t.Fatalf("la mutacion de prueba no produjo un vinculo valido: %v", err)
			}
			autoridad, err := base.servicio.EmitirAutoridadInternaEjecucionDocumentalV4(
				context.Background(), vinculo, base.cabecera, base.sobre,
			)
			comprobarDenegacionAtestacionPDP(t, autoridad, err)
		})
	}
}

func TestServicioDeniegaPayloadAudienciasSuiteKidClaveYAlgoritmoDistintos(t *testing.T) {
	base := nuevoEscenarioAtestacionAutorizacionPDPV4(t)

	payloadAlterado := append([]byte(nil), base.payload...)
	payloadAlterado[len(payloadAlterado)/2] ^= 0x01
	sobrePayloadAlterado := firmarSobreCOSEPrueba(
		t, base.material, payloadAlterado, base.solicitud, nil, nil,
	)

	solicitudOtraAudiencia := nuevaSolicitudCOSEPrueba(
		t, base.payload, AudienciaCOSEEvidenciaDocumental,
	)
	sobreOtraAudiencia := firmarSobreCOSEPrueba(
		t, base.material, base.payload, solicitudOtraAudiencia, nil, nil,
	)
	sobreOtroKid := firmarSobreCOSEPrueba(
		t, base.material, base.payload, base.solicitud,
		func(mensaje *cose.Sign1Message) {
			mensaje.Headers.Protected[cose.HeaderLabelKeyID] = []byte("clave:pdp:otro-kid")
		},
		nil,
	)
	otraClave := generarMaterialFirmaCOSEPrueba(
		t, AlgoritmoCOSEDocumentalEdDSA, base.material.claveID,
	)
	sobreOtraClave := firmarSobreCOSEPrueba(
		t, otraClave, base.payload, base.solicitud, nil, nil,
	)
	otroAlgoritmo := generarMaterialFirmaCOSEPrueba(
		t, AlgoritmoCOSEDocumentalES256, []byte("clave:pdp:es256-no-aprobada"),
	)
	sobreOtroAlgoritmo := firmarSobreCOSEPrueba(
		t, otroAlgoritmo, base.payload, base.solicitud, nil, nil,
	)

	cabeceraSuiteSeparada := base.cabecera
	cabeceraSuiteSeparada.Suite = "VEC-AD-ED25519-1"
	payloadSuiteSeparada, err := domain.SerializarMensajeAtestacionAutorizacionV1(
		cabeceraSuiteSeparada, base.escenario.decision,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitudSuiteSeparada := nuevaSolicitudCOSEPrueba(
		t, payloadSuiteSeparada, AudienciaCOSEAtestacionAutorizacionPDP,
	)
	sobreSuiteSeparada := firmarSobreCOSEPrueba(
		t, base.material, payloadSuiteSeparada, solicitudSuiteSeparada, nil, nil,
	)

	cabeceraOtroEntorno := base.cabecera
	cabeceraOtroEntorno.Audiencia = "vec-diputacion/produccion/vec/autorizacion"
	payloadOtroEntorno, err := domain.SerializarMensajeAtestacionAutorizacionV1(
		cabeceraOtroEntorno, base.escenario.decision,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitudOtroEntorno := nuevaSolicitudCOSEPrueba(
		t, payloadOtroEntorno, AudienciaCOSEAtestacionAutorizacionPDP,
	)
	sobreOtroEntorno := firmarSobreCOSEPrueba(
		t, base.material, payloadOtroEntorno, solicitudOtroEntorno, nil, nil,
	)

	casos := []struct {
		nombre   string
		cabecera domain.CabeceraAtestacionAutorizacionV1
		sobre    ports.SobreCriptograficoDocumentalCrudoV4
	}{
		{"payload", base.cabecera, sobrePayloadAlterado},
		{"firma valida para otra audiencia COSE", base.cabecera, sobreOtraAudiencia},
		{"suite de firma separada", cabeceraSuiteSeparada, sobreSuiteSeparada},
		{"audiencia de otro despliegue", cabeceraOtroEntorno, sobreOtroEntorno},
		{"kid", base.cabecera, sobreOtroKid},
		{"clave", base.cabecera, sobreOtraClave},
		{"algoritmo", base.cabecera, sobreOtroAlgoritmo},
	}
	for _, prueba := range casos {
		t.Run(prueba.nombre, func(t *testing.T) {
			autoridad, err := base.servicio.EmitirAutoridadInternaEjecucionDocumentalV4(
				context.Background(), base.vinculo, prueba.cabecera, prueba.sobre,
			)
			comprobarDenegacionAtestacionPDP(t, autoridad, err)
		})
	}
}

func TestServicioDeniegaFirmaValidaDeOtraRaizPDPConfiable(t *testing.T) {
	base := nuevoEscenarioAtestacionAutorizacionPDPV4(t)
	otroMaterial := generarMaterialFirmaCOSEPrueba(
		t, AlgoritmoCOSEDocumentalEdDSA, []byte("clave:pdp:cose:ed25519:2"),
	)
	raizPrincipal := nuevaRaizCOSEPrueba(
		t, base.material, EstadoConfianzaClaveDocumentalActiva,
		AudienciaCOSEAtestacionAutorizacionPDP, base.escenario.emitidaEn,
	)
	raizAlternativa := nuevaRaizCOSEPrueba(
		t, otroMaterial, EstadoConfianzaClaveDocumentalActiva,
		AudienciaCOSEAtestacionAutorizacionPDP, base.escenario.emitidaEn,
	)
	configuracion, err := nuevaConfiguracionConfianzaFijada(
		"confianza:pdp:rotacion", base.escenario.emitidaEn.Add(-time.Hour),
		base.escenario.emitidaEn.Add(time.Hour), raizPrincipal, raizAlternativa,
	)
	if err != nil {
		t.Fatal(err)
	}
	servicio, err := nuevoServicioConReloj(
		configuracion,
		&relojConfianzaDocumentalFijo{ahora: base.escenario.emitidaEn},
	)
	if err != nil {
		t.Fatal(err)
	}
	// La segunda raiz puede verificar una firma PDP real, pero el payload dice
	// que la cabecera selecciono la primera. Esa mezcla debe fallar despues de
	// la verificacion criptografica al cotejar kid/cabecera.
	sobreAlternativo := firmarSobreCOSEPrueba(
		t, otroMaterial, base.payload, base.solicitud, nil, nil,
	)
	autoridad, err := servicio.EmitirAutoridadInternaEjecucionDocumentalV4(
		context.Background(), base.vinculo, base.cabecera, sobreAlternativo,
	)
	comprobarDenegacionAtestacionPDP(t, autoridad, err)
}

func TestServicioDeniegaRevocacionCaducidadValorCeroYCancelacion(t *testing.T) {
	base := nuevoEscenarioAtestacionAutorizacionPDPV4(t)

	servicioRevocado, _ := nuevoServicioCOSEPrueba(
		t, base.escenario.emitidaEn, base.material,
		EstadoConfianzaClaveDocumentalRevocada,
		AudienciaCOSEAtestacionAutorizacionPDP,
	)
	autoridad, err := servicioRevocado.EmitirAutoridadInternaEjecucionDocumentalV4(
		context.Background(), base.vinculo, base.cabecera, base.sobre,
	)
	comprobarDenegacionAtestacionPDP(t, autoridad, err)

	servicioDecisionCaducada := *base.servicio
	servicioDecisionCaducada.reloj = &relojConfianzaDocumentalFijo{
		ahora: base.escenario.decision.ValidaHasta,
	}
	autoridad, err = servicioDecisionCaducada.EmitirAutoridadInternaEjecucionDocumentalV4(
		context.Background(), base.vinculo, base.cabecera, base.sobre,
	)
	comprobarDenegacionAtestacionPDP(t, autoridad, err)

	raiz := nuevaRaizCOSEPrueba(
		t, base.material, EstadoConfianzaClaveDocumentalActiva,
		AudienciaCOSEAtestacionAutorizacionPDP, base.escenario.emitidaEn,
	)
	configuracionCaducada, err := nuevaConfiguracionConfianzaFijada(
		"confianza:pdp:caducada", base.escenario.emitidaEn.Add(-time.Hour),
		base.escenario.emitidaEn, raiz,
	)
	if err != nil {
		t.Fatal(err)
	}
	servicioCaducado, err := nuevoServicioConReloj(
		configuracionCaducada,
		&relojConfianzaDocumentalFijo{ahora: base.escenario.emitidaEn},
	)
	if err != nil {
		t.Fatal(err)
	}
	autoridad, err = servicioCaducado.EmitirAutoridadInternaEjecucionDocumentalV4(
		context.Background(), base.vinculo, base.cabecera, base.sobre,
	)
	comprobarDenegacionAtestacionPDP(t, autoridad, err)

	var servicioNulo *Servicio
	autoridad, err = servicioNulo.EmitirAutoridadInternaEjecucionDocumentalV4(
		context.Background(), base.vinculo, base.cabecera, base.sobre,
	)
	comprobarDenegacionAtestacionPDP(t, autoridad, err)
	autoridad, err = (&Servicio{}).EmitirAutoridadInternaEjecucionDocumentalV4(
		context.Background(), base.vinculo, base.cabecera, base.sobre,
	)
	comprobarDenegacionAtestacionPDP(t, autoridad, err)

	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	autoridad, err = base.servicio.EmitirAutoridadInternaEjecucionDocumentalV4(
		ctx, base.vinculo, base.cabecera, base.sobre,
	)
	comprobarDenegacionAtestacionPDP(t, autoridad, err)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("la cancelacion no se conservo: %v", err)
	}
}

func TestAutoridadRespetaCaducidadDeRaizYConfiguracion(t *testing.T) {
	casos := []struct {
		nombre          string
		limiteRaiz      func(time.Time) time.Time
		limiteConfianza func(time.Time) time.Time
	}{
		{
			"raiz",
			func(ahora time.Time) time.Time { return ahora.Add(time.Microsecond) },
			func(ahora time.Time) time.Time { return ahora.Add(time.Hour) },
		},
		{
			"configuracion",
			func(ahora time.Time) time.Time { return ahora.Add(time.Hour) },
			func(ahora time.Time) time.Time { return ahora.Add(time.Microsecond) },
		},
	}
	for _, prueba := range casos {
		t.Run(prueba.nombre, func(t *testing.T) {
			base := nuevoEscenarioAtestacionAutorizacionPDPV4(t)
			limiteRaiz := prueba.limiteRaiz(base.escenario.emitidaEn)
			limiteConfianza := prueba.limiteConfianza(base.escenario.emitidaEn)
			limiteEfectivo := limiteRaiz
			if limiteConfianza.Before(limiteEfectivo) {
				limiteEfectivo = limiteConfianza
			}
			raiz, err := nuevaRaizPublicaFijadaAtestacionPDP(
				base.material.claveID, base.material.algoritmoDocumental, base.material.publica,
				suiteAtestacionAutorizacionPDPCOSEEdDSAV1,
				audienciaDespliegueAtestacionPDPPrueba,
				EstadoConfianzaClaveDocumentalActiva,
				base.escenario.emitidaEn.Add(-time.Hour), limiteRaiz, time.Time{},
			)
			if err != nil {
				t.Fatal(err)
			}
			configuracion, err := nuevaConfiguracionConfianzaFijada(
				"confianza:pdp:limite:"+prueba.nombre,
				base.escenario.emitidaEn.Add(-time.Hour), limiteConfianza, raiz,
			)
			if err != nil {
				t.Fatal(err)
			}
			servicio, err := nuevoServicioConReloj(
				configuracion,
				&relojConfianzaDocumentalFijo{ahora: base.escenario.emitidaEn},
			)
			if err != nil {
				t.Fatal(err)
			}
			autoridad, err := servicio.EmitirAutoridadInternaEjecucionDocumentalV4(
				context.Background(), base.vinculo, base.cabecera, base.sobre,
			)
			if err != nil {
				t.Fatalf("emitir justo antes del limite: %v", err)
			}
			if autoridad.ValidarEn(base.escenario.emitidaEn) != nil {
				t.Fatal("autoridad invalida antes del limite")
			}
			if err := autoridad.ValidarEn(limiteEfectivo); !errors.Is(err, domain.ErrAutorizacionDenegada) {
				t.Fatalf("autoridad valida en limite superior exclusivo: %v", err)
			}
			if _, err := autoridad.PrepararAplicacionExactaEn(
				base.escenario.decision.DecisionRef,
				base.escenario.expectativa.HuellaPlanSHA256,
				base.escenario.expectativa.EfectoRef,
				limiteEfectivo,
			); !errors.Is(err, domain.ErrAutorizacionDenegada) {
				t.Fatalf("se preparo efecto en el limite de confianza: %v", err)
			}
		})
	}
}

func TestRaizPDPExigePerfilDedicadoKidASCIIYAudienciaDeDespliegue(t *testing.T) {
	ahora := instanteConfianzaDocumentalPrueba
	material := generarMaterialFirmaCOSEPrueba(
		t, AlgoritmoCOSEDocumentalEdDSA, []byte("clave:pdp:perfil"),
	)
	if _, err := nuevaRaizPublicaFijada(
		material.claveID, material.algoritmoDocumental, material.publica,
		AudienciaCOSEAtestacionAutorizacionPDP, EstadoConfianzaClaveDocumentalActiva,
		ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{},
	); !errors.Is(err, ErrConfiguracionConfianzaDocumentalInvalida) {
		t.Fatalf("raiz PDP generica sin perfil aceptada: %v", err)
	}

	casos := []struct {
		nombre    string
		claveID   []byte
		suite     string
		audiencia string
	}{
		{"kid binario", []byte{0xff, 0x01}, suiteAtestacionAutorizacionPDPCOSEEdDSAV1, audienciaDespliegueAtestacionPDPPrueba},
		{"suite de firma separada", material.claveID, "VEC-AD-ED25519-1", audienciaDespliegueAtestacionPDPPrueba},
		{"suite libre", material.claveID, "VEC-AD-COSE-OTRA-1", audienciaDespliegueAtestacionPDPPrueba},
		{"audiencia sin entorno", material.claveID, suiteAtestacionAutorizacionPDPCOSEEdDSAV1, "atestacion_autorizacion_pdp"},
		{"audiencia con comodin", material.claveID, suiteAtestacionAutorizacionPDPCOSEEdDSAV1, "vec-diputacion/*/vec/autorizacion"},
	}
	for _, prueba := range casos {
		t.Run(prueba.nombre, func(t *testing.T) {
			if _, err := nuevaRaizPublicaFijadaAtestacionPDP(
				prueba.claveID, material.algoritmoDocumental, material.publica,
				prueba.suite, prueba.audiencia, EstadoConfianzaClaveDocumentalActiva,
				ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{},
			); !errors.Is(err, ErrConfiguracionConfianzaDocumentalInvalida) {
				t.Fatalf("perfil PDP invalido aceptado: %v", err)
			}
		})
	}

	materialES256 := generarMaterialFirmaCOSEPrueba(
		t, AlgoritmoCOSEDocumentalES256, []byte("clave:pdp:es256"),
	)
	if _, err := nuevaRaizPublicaFijadaAtestacionPDP(
		materialES256.claveID, materialES256.algoritmoDocumental, materialES256.publica,
		"VEC-AD-COSE-ES256-1", audienciaDespliegueAtestacionPDPPrueba,
		EstadoConfianzaClaveDocumentalActiva,
		ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{},
	); !errors.Is(err, ErrConfiguracionConfianzaDocumentalInvalida) {
		t.Fatalf("suite PDP ES256 no aprobada fue aceptada: %v", err)
	}

	raizPDP, err := nuevaRaizPublicaFijadaAtestacionPDP(
		material.claveID, material.algoritmoDocumental, material.publica,
		suiteAtestacionAutorizacionPDPCOSEEdDSAV1,
		audienciaDespliegueAtestacionPDPPrueba,
		EstadoConfianzaClaveDocumentalActiva,
		ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{},
	)
	if err != nil {
		t.Fatal(err)
	}
	raizMismoMaterialOtroProtocolo, err := nuevaRaizPublicaFijada(
		[]byte("clave:evidencia:alias-pdp"), material.algoritmoDocumental, material.publica,
		AudienciaCOSEEvidenciaDocumental, EstadoConfianzaClaveDocumentalActiva,
		ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := nuevaConfiguracionConfianzaFijada(
		"confianza:pdp:reutilizada", ahora.Add(-time.Hour), ahora.Add(time.Hour),
		raizPDP, raizMismoMaterialOtroProtocolo,
	); !errors.Is(err, ErrConfiguracionConfianzaDocumentalInvalida) {
		t.Fatalf("material PDP reutilizado por otro protocolo: %v", err)
	}

	raizOtroEntorno, err := nuevaRaizPublicaFijadaAtestacionPDP(
		material.claveID, material.algoritmoDocumental, material.publica,
		suiteAtestacionAutorizacionPDPCOSEEdDSAV1,
		"vec-diputacion/produccion/vec/autorizacion",
		EstadoConfianzaClaveDocumentalActiva,
		ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{},
	)
	if err != nil {
		t.Fatal(err)
	}
	configuracionPruebas, err := nuevaConfiguracionConfianzaFijada(
		"confianza:pdp:perfil", ahora.Add(-time.Hour), ahora.Add(time.Hour), raizPDP,
	)
	if err != nil {
		t.Fatal(err)
	}
	configuracionProduccion, err := nuevaConfiguracionConfianzaFijada(
		"confianza:pdp:perfil", ahora.Add(-time.Hour), ahora.Add(time.Hour), raizOtroEntorno,
	)
	if err != nil {
		t.Fatal(err)
	}
	if configuracionPruebas.huellaSHA256 == configuracionProduccion.huellaSHA256 {
		t.Fatal("la huella de confianza no comprometio la audiencia de despliegue")
	}
}

func nuevoEscenarioAtestacionAutorizacionPDPV4(
	t *testing.T,
) escenarioAtestacionAutorizacionPDPV4 {
	t.Helper()
	escenario := nuevoEscenarioAutoridadInternaEjecucionDocumentalV4(t)
	vinculo, err := ports.NuevaSolicitudVinculadaAutorizacionEjecucionDocumentalV4(
		escenario.evidencia, escenario.expectativa, escenario.emitidaEn,
	)
	if err != nil {
		t.Fatalf("crear vinculo PDP: %v", err)
	}
	material := generarMaterialFirmaCOSEPrueba(
		t, AlgoritmoCOSEDocumentalEdDSA, []byte("clave:pdp:cose:ed25519:1"),
	)
	servicio, _ := nuevoServicioCOSEPrueba(
		t, escenario.emitidaEn, material, EstadoConfianzaClaveDocumentalActiva,
		AudienciaCOSEAtestacionAutorizacionPDP,
	)
	cabecera := domain.CabeceraAtestacionAutorizacionV1{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV1,
		Suite:          suiteAtestacionAutorizacionPDPCOSEEdDSAV1,
		ClaveID:        string(material.claveID),
		Audiencia:      audienciaDespliegueAtestacionPDPPrueba,
	}
	payload, err := domain.SerializarMensajeAtestacionAutorizacionV1(
		cabecera, escenario.decision,
	)
	if err != nil {
		t.Fatalf("serializar VEC-AD-1 PDP: %v", err)
	}
	solicitud := nuevaSolicitudCOSEPrueba(
		t, payload, AudienciaCOSEAtestacionAutorizacionPDP,
	)
	sobre := firmarSobreCOSEPrueba(t, material, payload, solicitud, nil, nil)
	return escenarioAtestacionAutorizacionPDPV4{
		escenario: escenario, vinculo: vinculo, material: material, servicio: servicio,
		cabecera: cabecera, payload: payload, solicitud: solicitud, sobre: sobre,
	}
}

func reconstruirEscenarioAutorizacionPDPV4(
	t *testing.T,
	escenario *escenarioAutoridadInternaEjecucionDocumentalV4,
) {
	t.Helper()
	datosAnteriores, err := escenario.evidencia.Datos()
	if err != nil {
		t.Fatalf("leer evidencia anterior: %v", err)
	}
	datosVinculo, err := escenario.decision.VinculoAutenticacionActor.Datos()
	if err != nil {
		t.Fatalf("leer actor mutado: %v", err)
	}
	escenario.expectativa.DecisionEsperada = clonarDecisionInterna(escenario.decision)
	escenario.expectativa.PrincipalID = escenario.decision.PrincipalID
	escenario.expectativa.PerfilActivoRef = escenario.decision.PerfilActivoRef
	escenario.expectativa.AutenticacionRef = datosVinculo.AutenticacionRef
	escenario.expectativa.SesionRef = datosVinculo.SesionRef
	escenario.expectativa.ControlSesionRef = datosVinculo.ControlSesionRef
	escenario.expectativa.ControlSesionRevision = datosVinculo.ControlSesionRevision
	escenario.expectativa.ControlSesionHuellaSHA256 = datosVinculo.ControlSesionHuellaSHA256
	escenario.expectativa.ContextoActorRef = datosVinculo.ContextoActorRef
	escenario.expectativa.ContextoActorVersion = datosVinculo.ContextoActorVersion
	escenario.expectativa.ContextoActorHuellaSHA256 = datosVinculo.ContextoActorHuellaSHA256
	escenario.expectativa.Finalidad = escenario.decision.Finalidad
	escenario.expectativa.CorrelacionRef = escenario.decision.CorrelacionRef
	escenario.expectativa.EfectoRef = escenario.expectativa.Recurso.Atributos[ports.AtributoAutorizacionDocumentalEfectoRef]
	escenario.expectativa.HuellaPlanSHA256 = escenario.expectativa.Recurso.Atributos[ports.AtributoAutorizacionDocumentalHuellaPlanSHA256]
	escenario.expectativa.CamposPermitidosEsperados = append(
		[]string(nil), escenario.decision.CamposPermitidos...,
	)
	escenario.expectativa.ObligacionesEsperadas = append(
		[]string(nil), escenario.decision.Obligaciones...,
	)
	escenario.evidencia, err = ports.NuevaEvidenciaUsoDecisionAutorizacion(
		escenario.decision, datosAnteriores.VerificadaEn,
	)
	if err != nil {
		t.Fatalf("reconstruir evidencia mutada: %v", err)
	}
}

func comprobarDenegacionAtestacionPDP(
	t *testing.T,
	autoridad AutoridadInternaEjecucionDocumentalV4,
	err error,
) {
	t.Helper()
	if err == nil || !errors.Is(err, domain.ErrAutorizacionDenegada) ||
		!errors.Is(err, ErrAutoridadInternaEjecucionDocumentalV4Invalida) ||
		autoridad.ValidarEn(instanteConfianzaDocumentalPrueba) == nil {
		t.Fatalf("se esperaba denegacion cerrada; autoridad=%v error=%v", autoridad, err)
	}
}

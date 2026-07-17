package domain

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
)

func TestParserAtestacionDenegacionAutorizacionV1AceptaVectorNegativoOpaco(t *testing.T) {
	referencia := referenciaMotivoAtestacionAutorizacionV2Prueba()
	decision := decisionAtestacionDenegacionAutorizacionV1Prueba(t, referencia)
	cabecera := cabeceraAtestacionDenegacionAutorizacionV1Prueba()
	mensaje, err := SerializarMensajeAtestacionDenegacionAutorizacionV1(cabecera, decision, referencia)
	if err != nil {
		t.Fatalf("crear vector VEC-AD-D-1: %v", err)
	}
	original := append([]byte(nil), mensaje...)
	proyeccion, err := ParsearMensajeAtestacionDenegacionAutorizacionV1NoAutoritativo(mensaje)
	if err != nil {
		t.Fatalf("parsear vector negativo: %v", err)
	}
	if recibida, err := proyeccion.Cabecera(); err != nil || recibida != cabecera {
		t.Fatalf("cabecera nominal = %#v, err=%v", recibida, err)
	}
	if recibida, err := proyeccion.DecisionRef(); err != nil || recibida != decision.DecisionRef {
		t.Fatalf("decision_ref = %q, err=%v", recibida, err)
	}
	if recibida, err := proyeccion.SolicitudHuellaSHA256(); err != nil || recibida != decision.SolicitudHuellaSHA256 {
		t.Fatalf("huella solicitud = %q, err=%v", recibida, err)
	}
	if recibida, err := proyeccion.MotivoHuellaSHA256(); err != nil || recibida != decision.MotivoHuellaSHA256 {
		t.Fatalf("huella motivo = %q, err=%v", recibida, err)
	}
	if !bytes.Equal(mensaje, original) {
		t.Fatal("el parser altero el buffer")
	}
	for indice := range mensaje {
		mensaje[indice] = 0
	}
	if recibida, err := proyeccion.DecisionRef(); err != nil || recibida != decision.DecisionRef {
		t.Fatalf("la proyeccion conserva alias del buffer: %q, err=%v", recibida, err)
	}
}

func TestParserAtestacionDenegacionAutorizacionV1AceptaDenegacionTempranaSinGarantia(t *testing.T) {
	referencia := referenciaMotivoAtestacionAutorizacionV2Prueba()
	decision := decisionAtestacionDenegacionAutorizacionV1Prueba(t, referencia)
	decision.GarantiaMinima = ""
	mensaje, err := SerializarMensajeAtestacionDenegacionAutorizacionV1(
		cabeceraAtestacionDenegacionAutorizacionV1Prueba(),
		decision,
		referencia,
	)
	if err != nil {
		t.Fatalf("crear denegacion temprana: %v", err)
	}
	if _, err := ParsearMensajeAtestacionDenegacionAutorizacionV1NoAutoritativo(
		mensaje,
	); err != nil {
		t.Fatalf("parsear denegacion temprana: %v", err)
	}
}

func TestParserAtestacionDenegacionAutorizacionV1NoExponeDatosNiAutoridad(t *testing.T) {
	tipo := reflect.TypeOf(ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa{})
	for indice := 0; indice < tipo.NumField(); indice++ {
		campo := tipo.Field(indice)
		if campo.PkgPath == "" {
			t.Fatalf("campo fabricable fuera del dominio: %s", campo.Name)
		}
		if campo.Type == reflect.TypeOf(DecisionAutorizacion{}) ||
			campo.Type == reflect.TypeOf(VinculoAutenticacionActorV1{}) {
			t.Fatalf("autoridad reconstruida en %s", campo.Name)
		}
	}
	if _, existe := tipo.MethodByName("Datos"); existe {
		t.Fatal("la denegacion expone datos completos")
	}
	if _, existe := tipo.MethodByName("MensajeCanonico"); existe {
		t.Fatal("la denegacion expone el payload sensible")
	}
}

func TestParserAtestacionDenegacionAutorizacionV1BloqueaCodecsYRedacta(t *testing.T) {
	proyeccion := proyeccionAtestacionDenegacionAutorizacionV1Prueba(t)
	exigirProyeccionAtestacionCodecsBloqueadosPrueba(
		t,
		proyeccion,
		ErrSerializacionProyeccionAtestacionDenegacionAutorizacionV1Prohibida,
	)
	var destino ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa
	exigirDeserializacionProyeccionAtestacionBloqueadaPrueba(
		t,
		&destino,
		ErrSerializacionProyeccionAtestacionDenegacionAutorizacionV1Prohibida,
	)
	principal := proyeccion.datos.PrincipalID
	sesion := proyeccion.datos.VinculoAutenticacionActor.SesionRef
	entrada := proyeccion.motivo.entradaClave
	texto := fmt.Sprintf("%v|%+v|%#v|%s|%q", proyeccion, proyeccion, proyeccion, proyeccion, proyeccion)
	if strings.Contains(texto, principal) || strings.Contains(texto, sesion) || strings.Contains(texto, entrada) ||
		!strings.Contains(texto, representacionRedactadaProyeccionAtestacionDenegacionAutorizacionV1) {
		t.Fatalf("formato no redactado: %s", texto)
	}
	var registro bytes.Buffer
	slog.New(slog.NewTextHandler(&registro, nil)).Info("prueba", "denegacion", proyeccion)
	if strings.Contains(registro.String(), principal) || strings.Contains(registro.String(), sesion) {
		t.Fatalf("slog filtro datos sensibles: %s", registro.String())
	}
}

func TestParserAtestacionDenegacionAutorizacionV1RechazaClaseYCruces(t *testing.T) {
	proyeccion := proyeccionAtestacionDenegacionAutorizacionV1Prueba(t)
	casos := []struct {
		nombre string
		mutar  func(*datosDecisionAtestacionAutorizacionV2NoAutoritativos, *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos)
	}{
		{"concesion", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.Concedida, d.Codigo = true, "concedida"
		}},
		{"codigo_concesion", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.Codigo = "concedida"
		}},
		{"codigo_vacio", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.Codigo = ""
		}},
		{"codigo_comodin", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.Codigo = "denegacion_*"
		}},
		{"garantia_no_gobernada", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.GarantiaMinima = AuthAssurance("garantia_inventada")
		}},
		{"principal_vinculo", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.PrincipalID = "per_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		}},
		{"politica_aplicada", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.PoliticasHuellasSHA256[d.PoliticasRefs[0]] = strings.Repeat("8", 64)
		}},
		{"motivo", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.MotivoHuellaSHA256 = strings.Repeat("9", 64)
		}},
		{"referencia_motivo", func(_ *datosDecisionAtestacionAutorizacionV2NoAutoritativos, m *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			m.entradaClave = "motivo_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			mensaje := mensajeAtestacionDenegacionAutorizacionV1MutadoPrueba(t, proyeccion, caso.mutar)
			if _, err := ParsearMensajeAtestacionDenegacionAutorizacionV1NoAutoritativo(mensaje); !errors.Is(err, ErrParseoAtestacionDenegacionAutorizacionV1Invalido) {
				t.Fatalf("denegacion incoherente aceptada: %v", err)
			}
		})
	}
}

func TestParserAtestacionDenegacionAutorizacionV1RechazaMutacionesTruncadoYSobrante(t *testing.T) {
	proyeccion := proyeccionAtestacionDenegacionAutorizacionV1Prueba(t)
	mensaje := mensajeAtestacionDenegacionAutorizacionV1DesdeProyeccionPrueba(t, proyeccion)
	casos := map[string][]byte{
		"nil":      nil,
		"vacio":    {},
		"sobrante": append(append([]byte(nil), mensaje...), 0),
		"dominio":  mutarByteAtestacionPrueba(mensaje, 0, mensaje[0]^0xff),
		"longitud": mutarByteAtestacionPrueba(mensaje, len(mensaje)-1, mensaje[len(mensaje)-1]^1),
		"booleano": mutarByteAtestacionPrueba(
			mensaje,
			posicionConcedidaAtestacionSolicitudLigadaPrueba(mensaje, EsquemaMensajeAtestacionDenegacionAutorizacionV1),
			2,
		),
		"desbordado": make([]byte, TamanoMaximoMensajeAtestacionDenegacionAutorizacionV1+1),
	}
	for nombre, contenido := range casos {
		t.Run(nombre, func(t *testing.T) {
			if _, err := ParsearMensajeAtestacionDenegacionAutorizacionV1NoAutoritativo(contenido); !errors.Is(err, ErrParseoAtestacionDenegacionAutorizacionV1Invalido) {
				t.Fatalf("mutacion aceptada: %v", err)
			}
		})
	}
	for longitud := 0; longitud < len(mensaje); longitud++ {
		if _, err := ParsearMensajeAtestacionDenegacionAutorizacionV1NoAutoritativo(mensaje[:longitud]); !errors.Is(err, ErrParseoAtestacionDenegacionAutorizacionV1Invalido) {
			t.Fatalf("truncado en %d aceptado: %v", longitud, err)
		}
	}

	// Un contador hostil se rechaza antes de reservar la coleccion. Se crea a
	// traves del escritor sin semantica para no depender de offsets internos.
	listaDesbordada := mensajeAtestacionDenegacionAutorizacionV1MutadoPrueba(
		t,
		proyeccion,
		func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.Obligaciones = make([]string, maximoElementosAutorizacion+1)
		},
	)
	if _, err := ParsearMensajeAtestacionDenegacionAutorizacionV1NoAutoritativo(listaDesbordada); !errors.Is(err, ErrParseoAtestacionDenegacionAutorizacionV1Invalido) {
		t.Fatalf("contador desbordado aceptado: %v", err)
	}

	longitudSuite := len(EsquemaMensajeAtestacionDenegacionAutorizacionV1) + 1 + 2
	textoDesbordado := append([]byte(nil), mensaje...)
	binary.BigEndian.PutUint32(textoDesbordado[longitudSuite:longitudSuite+4], ^uint32(0))
	if _, err := ParsearMensajeAtestacionDenegacionAutorizacionV1NoAutoritativo(textoDesbordado); !errors.Is(err, ErrParseoAtestacionDenegacionAutorizacionV1Invalido) {
		t.Fatalf("texto desbordado aceptado: %v", err)
	}
}

func TestParserAtestacionDenegacionAutorizacionV1AceptaExactamenteElTecho(t *testing.T) {
	decision, referencia := decisionAtestacionDenegacionAutorizacionV1ConTamanoObjetivo(
		t,
		TamanoMaximoMensajeAtestacionDenegacionAutorizacionV1,
	)
	mensaje, err := SerializarMensajeAtestacionDenegacionAutorizacionV1(
		cabeceraAtestacionDenegacionAutorizacionV1Prueba(), decision, referencia,
	)
	if err != nil || len(mensaje) != TamanoMaximoMensajeAtestacionDenegacionAutorizacionV1 {
		t.Fatalf("crear borde exacto: len=%d err=%v", len(mensaje), err)
	}
	if _, err := ParsearMensajeAtestacionDenegacionAutorizacionV1NoAutoritativo(mensaje); err != nil {
		t.Fatalf("borde exacto rechazado: %v", err)
	}
}

func TestParserAtestacionDenegacionAutorizacionV1ValorCeroYDominioSeparado(t *testing.T) {
	var cero ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa
	if _, err := cero.Cabecera(); !errors.Is(err, ErrParseoAtestacionDenegacionAutorizacionV1Invalido) {
		t.Fatalf("valor cero aceptado: %v", err)
	}
	if _, err := cero.MotivoHuellaSHA256(); !errors.Is(err, ErrParseoAtestacionDenegacionAutorizacionV1Invalido) {
		t.Fatalf("compromiso del valor cero aceptado: %v", err)
	}
	referencia := referenciaMotivoAtestacionAutorizacionV2Prueba()
	decision := decisionAtestacionAutorizacionV2Prueba(t, referencia)
	mensaje, err := SerializarMensajeAtestacionAutorizacionV2(
		cabeceraAtestacionAutorizacionV2Prueba(), decision, referencia,
	)
	if err != nil {
		t.Fatalf("crear concesion: %v", err)
	}
	if _, err := ParsearMensajeAtestacionDenegacionAutorizacionV1NoAutoritativo(mensaje); !errors.Is(err, ErrParseoAtestacionDenegacionAutorizacionV1Invalido) {
		t.Fatalf("VEC-AD-2 aceptado como VEC-AD-D-1: %v", err)
	}
}

func FuzzParsearMensajeAtestacionDenegacionAutorizacionV1NoAutoritativoNoEntraEnPanico(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte(EsquemaMensajeAtestacionDenegacionAutorizacionV1))
	f.Fuzz(func(t *testing.T, contenido []byte) {
		_, _ = ParsearMensajeAtestacionDenegacionAutorizacionV1NoAutoritativo(contenido)
	})
}

func proyeccionAtestacionDenegacionAutorizacionV1Prueba(
	t *testing.T,
) ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa {
	t.Helper()
	referencia := referenciaMotivoAtestacionAutorizacionV2Prueba()
	mensaje, err := SerializarMensajeAtestacionDenegacionAutorizacionV1(
		cabeceraAtestacionDenegacionAutorizacionV1Prueba(),
		decisionAtestacionDenegacionAutorizacionV1Prueba(t, referencia),
		referencia,
	)
	if err != nil {
		t.Fatalf("crear vector negativo: %v", err)
	}
	proyeccion, err := ParsearMensajeAtestacionDenegacionAutorizacionV1NoAutoritativo(mensaje)
	if err != nil {
		t.Fatalf("parsear fixture negativo: %v", err)
	}
	return proyeccion
}

func mensajeAtestacionDenegacionAutorizacionV1DesdeProyeccionPrueba(
	t *testing.T,
	p ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa,
) []byte {
	t.Helper()
	mensaje, err := serializarMensajeAtestacionSolicitudLigadaNoAutoritativo(
		EsquemaMensajeAtestacionDenegacionAutorizacionV1,
		p.cabecera.FormatoVersion,
		p.cabecera.Suite,
		p.cabecera.ClaveID,
		p.cabecera.Audiencia,
		*p.datos,
		*p.motivo,
		TamanoMaximoMensajeAtestacionDenegacionAutorizacionV1,
	)
	if err != nil {
		t.Fatalf("serializar denegacion de prueba: %v", err)
	}
	return mensaje
}

func mensajeAtestacionDenegacionAutorizacionV1MutadoPrueba(
	t *testing.T,
	p ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa,
	mutar func(*datosDecisionAtestacionAutorizacionV2NoAutoritativos, *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos),
) []byte {
	t.Helper()
	datos := clonarDatosAtestacionSolicitudLigadaNoAutoritativos(*p.datos)
	motivo := *p.motivo
	mutar(&datos, &motivo)
	mensaje, err := serializarMensajeAtestacionSolicitudLigadaNoAutoritativo(
		EsquemaMensajeAtestacionDenegacionAutorizacionV1,
		p.cabecera.FormatoVersion,
		p.cabecera.Suite,
		p.cabecera.ClaveID,
		p.cabecera.Audiencia,
		datos,
		motivo,
		TamanoMaximoMensajeAtestacionDenegacionAutorizacionV1,
	)
	if err != nil {
		t.Fatalf("crear denegacion mutada: %v", err)
	}
	return mensaje
}

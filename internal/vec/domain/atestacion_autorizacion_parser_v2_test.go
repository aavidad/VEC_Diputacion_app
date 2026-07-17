package domain

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestParserAtestacionAutorizacionV2AceptaVectorYSoloExponeCompromisos(t *testing.T) {
	referencia := referenciaMotivoAtestacionAutorizacionV2Prueba()
	decision := decisionAtestacionAutorizacionV2Prueba(t, referencia)
	cabecera := cabeceraAtestacionAutorizacionV2Prueba()
	mensaje, err := SerializarMensajeAtestacionAutorizacionV2(cabecera, decision, referencia)
	if err != nil {
		t.Fatalf("crear vector VEC-AD-2: %v", err)
	}
	original := append([]byte(nil), mensaje...)

	proyeccion, err := ParsearMensajeAtestacionAutorizacionV2NoAutoritativo(mensaje)
	if err != nil {
		t.Fatalf("parsear vector contractual: %v", err)
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
		t.Fatal("el parser modifico el buffer del llamador")
	}

	// La proyeccion no conserva el buffer: cambiarlo despues no altera sus datos.
	for indice := range mensaje {
		mensaje[indice] ^= 0xff
	}
	if recibida, err := proyeccion.DecisionRef(); err != nil || recibida != decision.DecisionRef {
		t.Fatalf("alias con el buffer de entrada: %q, err=%v", recibida, err)
	}
}

func TestParserAtestacionAutorizacionV2EsNominalOpacoYEsquemaCerrado(t *testing.T) {
	tipoProyeccion := reflect.TypeOf(ProyeccionAtestacionAutorizacionV2NoAutoritativa{})
	tipoDecision := reflect.TypeOf(DecisionAutorizacion{})
	tipoVinculo := reflect.TypeOf(VinculoAutenticacionActorV1{})
	for indice := 0; indice < tipoProyeccion.NumField(); indice++ {
		campo := tipoProyeccion.Field(indice)
		if campo.PkgPath == "" {
			t.Fatalf("campo fabricable fuera del dominio: %s", campo.Name)
		}
		if campo.Type == tipoDecision || campo.Type == tipoVinculo {
			t.Fatalf("la proyeccion reconstruye autoridad en %s", campo.Name)
		}
	}
	if _, existe := tipoProyeccion.MethodByName("Datos"); existe {
		t.Fatal("la proyeccion expone los datos completos")
	}
	if _, existe := tipoProyeccion.MethodByName("MensajeCanonico"); existe {
		t.Fatal("la proyeccion expone de nuevo el payload con datos personales")
	}

	tipoDatos := reflect.TypeOf(datosDecisionAtestacionAutorizacionV2NoAutoritativos{})
	if tipoDatos.NumField() != len(camposDecisionAtestacionAutorizacionV2) || tipoDatos.NumField() != 35 {
		t.Fatalf("parser=%d contrato=%d; se exigen 35 campos", tipoDatos.NumField(), len(camposDecisionAtestacionAutorizacionV2))
	}
	for indice, esperado := range camposDecisionAtestacionAutorizacionV2 {
		campoParser := tipoDatos.Field(indice)
		if campoParser.Name != esperado.nombreGo {
			t.Fatalf("campo %d del parser = %s; esperado %s", indice, campoParser.Name, esperado.nombreGo)
		}
		campoDecision := tipoDecision.Field(indice)
		if campoParser.Name == "VinculoAutenticacionActor" {
			if campoParser.Type != reflect.TypeOf(DatosVinculoAutenticacionActorV1{}) {
				t.Fatalf("el parser reconstruye vinculo vivo: %v", campoParser.Type)
			}
			continue
		}
		if campoParser.Type != campoDecision.Type {
			t.Fatalf("tipo distinto en %s: %v frente a %v", campoParser.Name, campoParser.Type, campoDecision.Type)
		}
	}
	if reflect.TypeOf(DatosVinculoAutenticacionActorV1{}).NumField() != 25 {
		t.Fatal("el bloque de autenticacion ya no contiene exactamente 25 datos")
	}
	if reflect.TypeOf(datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos{}).NumField() != 4 {
		t.Fatal("el motivo ya no contiene exactamente cuatro coordenadas")
	}
}

func TestParserAtestacionAutorizacionV2BloqueaCodecsYRedacta(t *testing.T) {
	referencia := referenciaMotivoAtestacionAutorizacionV2Prueba()
	decision := decisionAtestacionAutorizacionV2Prueba(t, referencia)
	mensaje, err := SerializarMensajeAtestacionAutorizacionV2(
		cabeceraAtestacionAutorizacionV2Prueba(), decision, referencia,
	)
	if err != nil {
		t.Fatalf("crear vector: %v", err)
	}
	proyeccion, err := ParsearMensajeAtestacionAutorizacionV2NoAutoritativo(mensaje)
	if err != nil {
		t.Fatalf("parsear: %v", err)
	}
	exigirProyeccionAtestacionCodecsBloqueadosPrueba(
		t,
		proyeccion,
		ErrSerializacionProyeccionAtestacionAutorizacionV2Prohibida,
	)
	var destino ProyeccionAtestacionAutorizacionV2NoAutoritativa
	exigirDeserializacionProyeccionAtestacionBloqueadaPrueba(
		t,
		&destino,
		ErrSerializacionProyeccionAtestacionAutorizacionV2Prohibida,
	)

	datosVinculo, _ := decision.VinculoAutenticacionActor.Datos()
	texto := fmt.Sprintf("%v|%+v|%#v|%s|%q", proyeccion, proyeccion, proyeccion, proyeccion, proyeccion)
	if strings.Contains(texto, decision.PrincipalID) || strings.Contains(texto, datosVinculo.SesionRef) ||
		strings.Contains(texto, referencia.EntradaClave) ||
		!strings.Contains(texto, representacionRedactadaProyeccionAtestacionAutorizacionV2) {
		t.Fatalf("formato no redactado: %s", texto)
	}
	var registro bytes.Buffer
	slog.New(slog.NewTextHandler(&registro, nil)).Info("prueba", "proyeccion", proyeccion)
	if strings.Contains(registro.String(), decision.PrincipalID) || strings.Contains(registro.String(), datosVinculo.SesionRef) {
		t.Fatalf("slog filtro datos personales: %s", registro.String())
	}
}

func TestParserAtestacionAutorizacionV2RechazaCadaCruceSemantico(t *testing.T) {
	proyeccion := proyeccionAtestacionAutorizacionV2Prueba(t)
	casos := []struct {
		nombre string
		mutar  func(*datosDecisionAtestacionAutorizacionV2NoAutoritativos, *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos)
	}{
		{"clase_denegacion", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.Concedida, d.Codigo = false, "denegada"
		}},
		{"principal_vinculo", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.PrincipalID = "per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		}},
		{"perfil_vinculo", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.PerfilActivoRef = "prf_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		}},
		{"correlacion", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.CorrelacionRef = "correlacion_declarada"
		}},
		{"esquema_solicitud", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.EsquemaHuellaSolicitud = EsquemaMensajeAtestacionAutorizacionV1
		}},
		{"huella_solicitud_nula", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.SolicitudHuellaSHA256 = strings.Repeat("0", 64)
		}},
		{"esquema_motivo", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.EsquemaHuellaMotivo = EsquemaMensajeAtestacionAutorizacionV1
		}},
		{"huella_motivo_referencia", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.MotivoHuellaSHA256 = strings.Repeat("1", 64)
		}},
		{"rol_control", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.ControlVigenciaVersionRolRef = d.VersionRolRef + ":otra"
		}},
		{"garantia_vinculo", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.GarantiaMinima = AuthAssurance("invalida")
		}},
		{"emision_antes_revalidacion", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.EmitidaEn = d.VinculoAutenticacionActor.SesionRevalidadaEn.Add(-time.Microsecond)
		}},
		{"vigencia_despues_sesion", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.ValidaHasta = d.VinculoAutenticacionActor.SesionValidaHasta.Add(time.Microsecond)
		}},
		{"vigencia_excesiva", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.ValidaHasta = d.EmitidaEn.Add(VigenciaMaximaDecisionAutorizacion + time.Microsecond)
		}},
		{"instante_fuera_rango", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.EmitidaEn = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
		}},
		{"catalogo_politicas", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.CatalogoPoliticasHuellaSHA256 = strings.Repeat("2", 64)
		}},
		{"politica_aplicada_evaluada", func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.PoliticasHuellasSHA256[d.PoliticasRefs[0]] = strings.Repeat("3", 64)
		}},
		{"motivo_catalogo", func(_ *datosDecisionAtestacionAutorizacionV2NoAutoritativos, m *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			m.catalogoID += "_otro"
		}},
		{"motivo_version", func(_ *datosDecisionAtestacionAutorizacionV2NoAutoritativos, m *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			m.catalogoVersion = math.MaxInt32 + 1
		}},
		{"motivo_version_cero", func(_ *datosDecisionAtestacionAutorizacionV2NoAutoritativos, m *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			m.catalogoVersion = 0
		}},
		{"motivo_huella_catalogo", func(_ *datosDecisionAtestacionAutorizacionV2NoAutoritativos, m *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			m.catalogoHuellaSHA256 = strings.Repeat("4", 64)
		}},
		{"motivo_clave", func(_ *datosDecisionAtestacionAutorizacionV2NoAutoritativos, m *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			m.entradaClave = "texto_libre"
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			mensaje := mensajeAtestacionAutorizacionV2MutadoPrueba(t, proyeccion, caso.mutar)
			if _, err := ParsearMensajeAtestacionAutorizacionV2NoAutoritativo(mensaje); !errors.Is(err, ErrParseoAtestacionAutorizacionV2Invalido) {
				t.Fatalf("cruce invalido aceptado: %v", err)
			}
		})
	}
}

func TestParserAtestacionAutorizacionV2RechazaMutacionesYLímitesAntesDeReservar(t *testing.T) {
	proyeccion := proyeccionAtestacionAutorizacionV2Prueba(t)
	mensaje := mensajeAtestacionAutorizacionV2DesdeProyeccionPrueba(t, proyeccion)

	casos := map[string][]byte{
		"nil":            nil,
		"vacio":          {},
		"sobrante":       append(append([]byte(nil), mensaje...), 0),
		"dominio":        mutarByteAtestacionPrueba(mensaje, 0, mensaje[0]^0xff),
		"longitud_total": mutarByteAtestacionPrueba(mensaje, len(mensaje)-1, mensaje[len(mensaje)-1]^1),
		"booleano_no_binario": mutarByteAtestacionPrueba(
			mensaje, posicionConcedidaAtestacionSolicitudLigadaPrueba(mensaje, EsquemaMensajeAtestacionAutorizacionV2), 2,
		),
	}
	// La longitud de Suite se lee antes de reservar su contenido.
	longitudSuite := len(EsquemaMensajeAtestacionAutorizacionV2) + 1 + 2
	versionInvalida := append([]byte(nil), mensaje...)
	binary.BigEndian.PutUint16(versionInvalida[len(EsquemaMensajeAtestacionAutorizacionV2)+1:], 1)
	casos["version_invalida"] = versionInvalida
	suiteDesbordada := append([]byte(nil), mensaje...)
	binary.BigEndian.PutUint32(suiteDesbordada[longitudSuite:longitudSuite+4], math.MaxUint32)
	casos["texto_desbordado"] = suiteDesbordada
	utf8Invalido := append([]byte(nil), mensaje...)
	utf8Invalido[longitudSuite+4] = 0xff
	casos["texto_utf8_invalido"] = utf8Invalido
	suiteConComodin := append([]byte(nil), mensaje...)
	suiteConComodin[longitudSuite+4] = '*'
	casos["suite_no_canonica"] = suiteConComodin
	casos["mensaje_desbordado"] = make([]byte, TamanoMaximoMensajeAtestacionAutorizacionV2+1)

	for nombre, contenido := range casos {
		t.Run(nombre, func(t *testing.T) {
			if _, err := ParsearMensajeAtestacionAutorizacionV2NoAutoritativo(contenido); !errors.Is(err, ErrParseoAtestacionAutorizacionV2Invalido) {
				t.Fatalf("mutacion aceptada: %v", err)
			}
		})
	}

	for longitud := 0; longitud < len(mensaje); longitud++ {
		if _, err := ParsearMensajeAtestacionAutorizacionV2NoAutoritativo(mensaje[:longitud]); !errors.Is(err, ErrParseoAtestacionAutorizacionV2Invalido) {
			t.Fatalf("truncado en %d aceptado: %v", longitud, err)
		}
	}

	listaDesbordada := mensajeAtestacionAutorizacionV2MutadoPrueba(
		t,
		proyeccion,
		func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.CamposPermitidos = make([]string, maximoElementosAutorizacion+1)
		},
	)
	if _, err := ParsearMensajeAtestacionAutorizacionV2NoAutoritativo(listaDesbordada); !errors.Is(err, ErrParseoAtestacionAutorizacionV2Invalido) {
		t.Fatalf("conteo de lista desbordado aceptado: %v", err)
	}
	mapaDesbordado := mensajeAtestacionAutorizacionV2MutadoPrueba(
		t,
		proyeccion,
		func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.PoliticasEvaluadasHuellasSHA256 = make(map[string]string, maximoElementosAutorizacion+1)
			for indice := 0; indice <= maximoElementosAutorizacion; indice++ {
				d.PoliticasEvaluadasHuellasSHA256[fmt.Sprintf("politica:%04d", indice)] = strings.Repeat("a", 64)
			}
		},
	)
	if _, err := ParsearMensajeAtestacionAutorizacionV2NoAutoritativo(mapaDesbordado); !errors.Is(err, ErrParseoAtestacionAutorizacionV2Invalido) {
		t.Fatalf("conteo de mapa desbordado aceptado: %v", err)
	}
	listaNoCanonica := mensajeAtestacionAutorizacionV2MutadoPrueba(
		t,
		proyeccion,
		func(d *datosDecisionAtestacionAutorizacionV2NoAutoritativos, _ *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos) {
			d.CamposPermitidos = []string{"z", "a"}
		},
	)
	if _, err := ParsearMensajeAtestacionAutorizacionV2NoAutoritativo(listaNoCanonica); !errors.Is(err, ErrParseoAtestacionAutorizacionV2Invalido) {
		t.Fatalf("lista fuera de orden aceptada: %v", err)
	}

	mapaNoCanonico := invertirPrimerMapaAtestacionSolicitudLigadaPrueba(
		t,
		mensaje,
		proyeccion.datos.PoliticasEvaluadasRefs,
		proyeccion.datos.PoliticasEvaluadasHuellasSHA256,
	)
	if _, err := ParsearMensajeAtestacionAutorizacionV2NoAutoritativo(mapaNoCanonico); !errors.Is(err, ErrParseoAtestacionAutorizacionV2Invalido) {
		t.Fatalf("mapa fuera de orden aceptado: %v", err)
	}
}

func TestParserAtestacionAutorizacionV2ProyeccionesAdmitenLecturaConcurrente(t *testing.T) {
	concesion := proyeccionAtestacionAutorizacionV2Prueba(t)
	denegacion := proyeccionAtestacionDenegacionAutorizacionV1Prueba(t)
	const lectores = 64
	errores := make(chan error, lectores*2)
	var grupo sync.WaitGroup
	for indice := 0; indice < lectores; indice++ {
		grupo.Add(2)
		go func() {
			defer grupo.Done()
			_, err := concesion.MotivoHuellaSHA256()
			errores <- err
		}()
		go func() {
			defer grupo.Done()
			_, err := denegacion.SolicitudHuellaSHA256()
			errores <- err
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatalf("lectura concurrente: %v", err)
		}
	}
}

func TestParserAtestacionAutorizacionV2AceptaExactamenteElTecho(t *testing.T) {
	decision, referencia := decisionAtestacionAutorizacionV2ConTamanoObjetivo(
		t,
		TamanoMaximoMensajeAtestacionAutorizacionV2,
	)
	mensaje, err := SerializarMensajeAtestacionAutorizacionV2(
		cabeceraAtestacionAutorizacionV2Prueba(), decision, referencia,
	)
	if err != nil || len(mensaje) != TamanoMaximoMensajeAtestacionAutorizacionV2 {
		t.Fatalf("crear borde exacto: len=%d err=%v", len(mensaje), err)
	}
	if _, err := ParsearMensajeAtestacionAutorizacionV2NoAutoritativo(mensaje); err != nil {
		t.Fatalf("borde exacto rechazado: %v", err)
	}
}

func TestParserAtestacionAutorizacionV2ValorCeroYDominioSeparado(t *testing.T) {
	var cero ProyeccionAtestacionAutorizacionV2NoAutoritativa
	if _, err := cero.Cabecera(); !errors.Is(err, ErrParseoAtestacionAutorizacionV2Invalido) {
		t.Fatalf("cabecera del valor cero aceptada: %v", err)
	}
	if _, err := cero.DecisionRef(); !errors.Is(err, ErrParseoAtestacionAutorizacionV2Invalido) {
		t.Fatalf("decision del valor cero aceptada: %v", err)
	}
	referencia := referenciaMotivoAtestacionAutorizacionV2Prueba()
	denegacion := decisionAtestacionDenegacionAutorizacionV1Prueba(t, referencia)
	mensaje, err := SerializarMensajeAtestacionDenegacionAutorizacionV1(
		cabeceraAtestacionDenegacionAutorizacionV1Prueba(), denegacion, referencia,
	)
	if err != nil {
		t.Fatalf("crear dominio negativo: %v", err)
	}
	if _, err := ParsearMensajeAtestacionAutorizacionV2NoAutoritativo(mensaje); !errors.Is(err, ErrParseoAtestacionAutorizacionV2Invalido) {
		t.Fatalf("VEC-AD-D-1 aceptado como VEC-AD-2: %v", err)
	}
}

func FuzzParsearMensajeAtestacionAutorizacionV2NoAutoritativoNoEntraEnPanico(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte(EsquemaMensajeAtestacionAutorizacionV2))
	f.Fuzz(func(t *testing.T, contenido []byte) {
		_, _ = ParsearMensajeAtestacionAutorizacionV2NoAutoritativo(contenido)
	})
}

func proyeccionAtestacionAutorizacionV2Prueba(t *testing.T) ProyeccionAtestacionAutorizacionV2NoAutoritativa {
	t.Helper()
	referencia := referenciaMotivoAtestacionAutorizacionV2Prueba()
	mensaje, err := SerializarMensajeAtestacionAutorizacionV2(
		cabeceraAtestacionAutorizacionV2Prueba(),
		decisionAtestacionAutorizacionV2Prueba(t, referencia),
		referencia,
	)
	if err != nil {
		t.Fatalf("crear vector: %v", err)
	}
	proyeccion, err := ParsearMensajeAtestacionAutorizacionV2NoAutoritativo(mensaje)
	if err != nil {
		t.Fatalf("parsear fixture: %v", err)
	}
	return proyeccion
}

func mensajeAtestacionAutorizacionV2DesdeProyeccionPrueba(
	t *testing.T,
	p ProyeccionAtestacionAutorizacionV2NoAutoritativa,
) []byte {
	t.Helper()
	mensaje, err := serializarMensajeAtestacionSolicitudLigadaNoAutoritativo(
		EsquemaMensajeAtestacionAutorizacionV2,
		p.cabecera.FormatoVersion,
		p.cabecera.Suite,
		p.cabecera.ClaveID,
		p.cabecera.Audiencia,
		*p.datos,
		*p.motivo,
		TamanoMaximoMensajeAtestacionAutorizacionV2,
	)
	if err != nil {
		t.Fatalf("serializar datos de prueba: %v", err)
	}
	return mensaje
}

func mensajeAtestacionAutorizacionV2MutadoPrueba(
	t *testing.T,
	p ProyeccionAtestacionAutorizacionV2NoAutoritativa,
	mutar func(*datosDecisionAtestacionAutorizacionV2NoAutoritativos, *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos),
) []byte {
	t.Helper()
	datos := clonarDatosAtestacionSolicitudLigadaNoAutoritativos(*p.datos)
	motivo := *p.motivo
	mutar(&datos, &motivo)
	mensaje, err := serializarMensajeAtestacionSolicitudLigadaNoAutoritativo(
		EsquemaMensajeAtestacionAutorizacionV2,
		p.cabecera.FormatoVersion,
		p.cabecera.Suite,
		p.cabecera.ClaveID,
		p.cabecera.Audiencia,
		datos,
		motivo,
		TamanoMaximoMensajeAtestacionAutorizacionV2,
	)
	if err != nil {
		t.Fatalf("crear mensaje mutado: %v", err)
	}
	return mensaje
}

func posicionConcedidaAtestacionSolicitudLigadaPrueba(mensaje []byte, esquema string) int {
	posicion := len(esquema) + 1 + 2
	for indice := 0; indice < 4; indice++ { // suite, clave, audiencia y decision_ref
		longitud := int(binary.BigEndian.Uint32(mensaje[posicion : posicion+4]))
		posicion += 4 + longitud
	}
	return posicion
}

func mutarByteAtestacionPrueba(origen []byte, posicion int, valor byte) []byte {
	copia := append([]byte(nil), origen...)
	copia[posicion] = valor
	return copia
}

func invertirPrimerMapaAtestacionSolicitudLigadaPrueba(
	t *testing.T,
	mensaje []byte,
	referencias []string,
	huellas map[string]string,
) []byte {
	t.Helper()
	if len(referencias) < 2 {
		t.Fatal("se requieren dos entradas para probar orden de mapa")
	}
	primera := codificarEntradaMapaAtestacionPrueba(referencias[0], huellas[referencias[0]])
	segunda := codificarEntradaMapaAtestacionPrueba(referencias[1], huellas[referencias[1]])
	patron := make([]byte, 4, 4+len(primera)+len(segunda))
	binary.BigEndian.PutUint32(patron, 2)
	patron = append(patron, primera...)
	patron = append(patron, segunda...)
	posicion := bytes.Index(mensaje, patron)
	if posicion < 0 {
		t.Fatal("no se encontro el mapa canonico en el vector")
	}

	reemplazo := make([]byte, 4, len(patron))
	binary.BigEndian.PutUint32(reemplazo, 2)
	reemplazo = append(reemplazo, segunda...)
	reemplazo = append(reemplazo, primera...)
	mutado := append([]byte(nil), mensaje...)
	copy(mutado[posicion:posicion+len(reemplazo)], reemplazo)
	return mutado
}

func codificarEntradaMapaAtestacionPrueba(clave, valor string) []byte {
	resultado := make([]byte, 4, 8+len(clave)+len(valor))
	binary.BigEndian.PutUint32(resultado, uint32(len(clave)))
	resultado = append(resultado, clave...)
	inicioValor := len(resultado)
	resultado = append(resultado, make([]byte, 4)...)
	binary.BigEndian.PutUint32(resultado[inicioValor:inicioValor+4], uint32(len(valor)))
	resultado = append(resultado, valor...)
	return resultado
}

func exigirProyeccionAtestacionCodecsBloqueadosPrueba(t *testing.T, valor any, esperado error) {
	t.Helper()
	if _, err := json.Marshal(valor); !errors.Is(err, esperado) {
		t.Fatalf("JSON no bloqueado: %v", err)
	}
	if _, err := xml.Marshal(valor); !errors.Is(err, esperado) {
		t.Fatalf("XML no bloqueado: %v", err)
	}
	var destino bytes.Buffer
	if err := gob.NewEncoder(&destino).Encode(valor); !errors.Is(err, esperado) {
		t.Fatalf("gob no bloqueado: %v", err)
	}
	texto, ok := valor.(interface{ MarshalText() ([]byte, error) })
	if !ok {
		t.Fatal("falta bloqueo de texto")
	}
	if _, err := texto.MarshalText(); !errors.Is(err, esperado) {
		t.Fatalf("texto no bloqueado: %v", err)
	}
	binario, ok := valor.(interface{ MarshalBinary() ([]byte, error) })
	if !ok {
		t.Fatal("falta bloqueo binario")
	}
	if _, err := binario.MarshalBinary(); !errors.Is(err, esperado) {
		t.Fatalf("binario no bloqueado: %v", err)
	}
	cbor, ok := valor.(interface{ MarshalCBOR() ([]byte, error) })
	if !ok {
		t.Fatal("falta bloqueo CBOR")
	}
	if _, err := cbor.MarshalCBOR(); !errors.Is(err, esperado) {
		t.Fatalf("CBOR no bloqueado: %v", err)
	}
	yaml, ok := valor.(interface{ MarshalYAML() (any, error) })
	if !ok {
		t.Fatal("falta bloqueo YAML")
	}
	if _, err := yaml.MarshalYAML(); !errors.Is(err, esperado) {
		t.Fatalf("YAML no bloqueado: %v", err)
	}
}

func exigirDeserializacionProyeccionAtestacionBloqueadaPrueba(t *testing.T, destino any, esperado error) {
	t.Helper()
	if err := json.Unmarshal([]byte(`{}`), destino); !errors.Is(err, esperado) {
		t.Fatalf("Unmarshal JSON no bloqueado: %v", err)
	}
	if err := xml.Unmarshal([]byte(`<proyeccion/>`), destino); !errors.Is(err, esperado) {
		t.Fatalf("Unmarshal XML no bloqueado: %v", err)
	}
	texto, ok := destino.(interface{ UnmarshalText([]byte) error })
	if !ok || !errors.Is(texto.UnmarshalText(nil), esperado) {
		t.Fatal("Unmarshal texto no bloqueado")
	}
	binario, ok := destino.(interface{ UnmarshalBinary([]byte) error })
	if !ok || !errors.Is(binario.UnmarshalBinary(nil), esperado) {
		t.Fatal("Unmarshal binario no bloqueado")
	}
	gobDecoder, ok := destino.(interface{ GobDecode([]byte) error })
	if !ok || !errors.Is(gobDecoder.GobDecode(nil), esperado) {
		t.Fatal("GobDecode no bloqueado")
	}
	cbor, ok := destino.(interface{ UnmarshalCBOR([]byte) error })
	if !ok || !errors.Is(cbor.UnmarshalCBOR([]byte{0xa0}), esperado) {
		t.Fatal("Unmarshal CBOR no bloqueado")
	}
	yaml, ok := destino.(interface{ UnmarshalYAML(func(any) error) error })
	if !ok || !errors.Is(yaml.UnmarshalYAML(func(any) error { return nil }), esperado) {
		t.Fatal("Unmarshal YAML no bloqueado")
	}
}

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
	"testing"
)

func TestParserHistoricoAtestacionAutorizacionV1ProyectaTodoSinCrearAutoridad(t *testing.T) {
	cabecera := cabeceraAtestacionAutorizacionV1Prueba()
	decision := decisionAtestacionAutorizacionV1Prueba(t)
	mensaje, err := SerializarMensajeAtestacionAutorizacionV1(cabecera, decision)
	if err != nil {
		t.Fatalf("crear VEC-AD-1: %v", err)
	}
	original := append([]byte(nil), mensaje...)

	proyeccion, err := ParsearMensajeAtestacionAutorizacionV1NoAutoritativo(mensaje)
	if err != nil {
		t.Fatalf("parsear vector emitido por el serializador contractual: %v", err)
	}
	cabeceraLeida, err := proyeccion.Cabecera()
	if err != nil || cabeceraLeida != cabecera {
		t.Fatalf("cabecera nominal distinta: %#v err=%v", cabeceraLeida, err)
	}
	datos, err := proyeccion.Datos()
	if err != nil {
		t.Fatalf("extraer datos nominales: %v", err)
	}
	esperados := datosDecisionHistoricaAtestacionAutorizacionV1Prueba(t, decision)
	if !reflect.DeepEqual(datos, esperados) {
		t.Fatal("la proyeccion historica no contiene exactamente todos los datos de decision")
	}
	canonico, err := serializarProyeccionHistoricaAtestacionAutorizacionV1(cabeceraLeida, datos)
	if err != nil || !bytes.Equal(canonico, original) {
		t.Fatalf("reserializacion historica no identica: err=%v", err)
	}
	if !bytes.Equal(mensaje, original) {
		t.Fatal("el parser modifico el buffer del llamador")
	}

	// El resultado no conserva alias del buffer ni de sus colecciones.
	mensaje[0] ^= 0xff
	datos.PoliticasEvaluadasRefs[0] = "politica:alterada:v1"
	datos.PoliticasEvaluadasHuellasSHA256[decision.PoliticasEvaluadasRefs[0]] = huellaAtestacionPrueba('9')
	datos.PoliticasRefs[0] = "politica:alterada:v2"
	datos.PoliticasHuellasSHA256[decision.PoliticasRefs[0]] = huellaAtestacionPrueba('8')
	datos.CamposPermitidos[0] = "alterado"
	datos.Obligaciones[0] = "alterada"
	segundaLectura, err := proyeccion.Datos()
	if err != nil || !reflect.DeepEqual(segundaLectura, esperados) {
		t.Fatalf("las copias del llamador alteraron la proyeccion: err=%v", err)
	}
}

func TestParserHistoricoAtestacionAutorizacionV1TipoNominalNoContieneCapacidades(t *testing.T) {
	tipoDatos := reflect.TypeOf(DatosDecisionHistoricaAtestacionAutorizacionV1{})
	if tipoDatos.NumField() != 31 {
		t.Fatalf("la proyeccion enumera %d campos; esperados los 30 datos mas el bloque de vinculo", tipoDatos.NumField())
	}
	tipoVinculo := reflect.TypeOf(VinculoAutenticacionActorV1{})
	tipoDecision := reflect.TypeOf(DecisionAutorizacion{})
	for indice := 0; indice < tipoDatos.NumField(); indice++ {
		campo := tipoDatos.Field(indice)
		if campo.Type == tipoVinculo || campo.Type == tipoDecision {
			t.Fatalf("la proyeccion nominal contiene autoridad reconstruible en %s", campo.Name)
		}
	}
	campoVinculo, existe := tipoDatos.FieldByName("VinculoAutenticacionActor")
	if !existe || campoVinculo.Type != reflect.TypeOf(DatosVinculoAutenticacionActorV1{}) {
		t.Fatalf("bloque de vinculo = %v; se exige solo DatosVinculoAutenticacionActorV1", campoVinculo.Type)
	}
	tipoProyeccion := reflect.TypeOf(ProyeccionHistoricaAtestacionAutorizacionV1{})
	for indice := 0; indice < tipoProyeccion.NumField(); indice++ {
		if tipoProyeccion.Field(indice).PkgPath == "" {
			t.Fatalf("campo fabricable fuera del paquete: %s", tipoProyeccion.Field(indice).Name)
		}
	}
}

func TestParserHistoricoAtestacionAutorizacionV1MantieneEsquemaCerradoDeTreintaDatosYBloque(t *testing.T) {
	esquema := []string{
		"decision_ref", "concedida", "codigo", "principal_id", "perfil_activo_ref",
		"accion", "recurso_ref", "modulo_id", "tipo_recurso", "contexto_recurso_huella_sha256",
		"finalidad", "correlacion_ref", "vinculo_autenticacion_actor", "asignacion_ref",
		"asignacion_huella_sha256", "version_rol_ref", "version_rol_huella_sha256",
		"control_vigencia_version_rol_ref", "control_vigencia_version_rol_revision",
		"control_vigencia_version_rol_huella_sha256", "revision_catalogo_politicas",
		"catalogo_politicas_huella_sha256", "politicas_evaluadas_refs",
		"politicas_evaluadas_huellas_sha256", "politicas_refs", "politicas_huellas_sha256",
		"garantia_minima", "campos_permitidos", "obligaciones", "emitida_en", "valida_hasta",
	}
	tipoHistorico := reflect.TypeOf(DatosDecisionHistoricaAtestacionAutorizacionV1{})
	tipoDecision := reflect.TypeOf(DecisionAutorizacion{})
	if tipoHistorico.NumField() != len(esquema) || len(esquema)-1 != 30 {
		t.Fatalf("deriva del parser historico: historico=%d datos_sin_bloque=%d",
			tipoHistorico.NumField(), len(esquema)-1)
	}
	for indice, etiquetaEsperada := range esquema {
		campoHistorico := tipoHistorico.Field(indice)
		campoDecision, existe := campoDecisionAtestacionAutorizacionV1(tipoDecision, etiquetaEsperada)
		if !existe {
			t.Fatalf("DecisionAutorizacion ya no contiene el dato historico %q", etiquetaEsperada)
		}
		etiquetaDecision := strings.Split(campoDecision.Tag.Get("json"), ",")[0]
		if etiquetaDecision != etiquetaEsperada || campoHistorico.Name != campoDecision.Name {
			t.Fatalf("campo %d: decision=%s/%q historico=%s esperado=%q",
				indice, campoDecision.Name, etiquetaDecision, campoHistorico.Name, etiquetaEsperada)
		}
		if etiquetaEsperada == "vinculo_autenticacion_actor" {
			if campoHistorico.Type != reflect.TypeOf(DatosVinculoAutenticacionActorV1{}) {
				t.Fatalf("el bloque historico reconstruye un tipo indebido: %v", campoHistorico.Type)
			}
			continue
		}
		if campoHistorico.Type != campoDecision.Type {
			t.Fatalf("tipo historico distinto en %s: %v frente a %v",
				campoDecision.Name, campoHistorico.Type, campoDecision.Type)
		}
	}
	tipoCabecera := reflect.TypeOf(CabeceraAtestacionAutorizacionV1{})
	if tipoCabecera.NumField() != 4 || EsquemaMensajeAtestacionAutorizacionV1 == "" ||
		VersionFormatoAtestacionAutorizacionV1 != 1 {
		t.Fatal("la cabecera, el dominio o la version historica cambiaron sin version nueva")
	}
}

func TestParserHistoricoAtestacionAutorizacionV1BloqueaLogsYSerializadoresGenerales(t *testing.T) {
	mensaje, err := SerializarMensajeAtestacionAutorizacionV1(
		cabeceraAtestacionAutorizacionV1Prueba(),
		decisionAtestacionAutorizacionV1Prueba(t),
	)
	if err != nil {
		t.Fatalf("crear mensaje: %v", err)
	}
	proyeccion, err := ParsearMensajeAtestacionAutorizacionV1NoAutoritativo(mensaje)
	if err != nil {
		t.Fatalf("parsear: %v", err)
	}
	datos, err := proyeccion.Datos()
	if err != nil {
		t.Fatalf("datos: %v", err)
	}

	serializables := []struct {
		nombre string
		valor  any
	}{
		{"proyeccion", proyeccion},
		{"datos", datos},
	}
	for _, caso := range serializables {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := json.Marshal(caso.valor); !errors.Is(err, ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida) {
				t.Fatalf("JSON no bloqueado: %v", err)
			}
			if _, err := xml.Marshal(caso.valor); !errors.Is(err, ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida) {
				t.Fatalf("XML no bloqueado: %v", err)
			}
			var destino bytes.Buffer
			if err := gob.NewEncoder(&destino).Encode(caso.valor); !errors.Is(err, ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida) {
				t.Fatalf("gob no bloqueado: %v", err)
			}
		})
	}
	if _, err := proyeccion.MarshalText(); !errors.Is(err, ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida) {
		t.Fatalf("texto de proyeccion no bloqueado: %v", err)
	}
	if _, err := proyeccion.MarshalBinary(); !errors.Is(err, ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida) {
		t.Fatalf("binario de proyeccion no bloqueado: %v", err)
	}
	if _, err := datos.MarshalText(); !errors.Is(err, ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida) {
		t.Fatalf("texto de datos no bloqueado: %v", err)
	}
	if _, err := datos.MarshalBinary(); !errors.Is(err, ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida) {
		t.Fatalf("binario de datos no bloqueado: %v", err)
	}
	if _, err := datos.MarshalCBOR(); !errors.Is(err, ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida) {
		t.Fatalf("CBOR no bloqueado: %v", err)
	}
	if _, err := proyeccion.MarshalCBOR(); !errors.Is(err, ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida) {
		t.Fatalf("CBOR de proyeccion no bloqueado: %v", err)
	}
	if _, err := proyeccion.MarshalYAML(); !errors.Is(err, ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida) {
		t.Fatalf("YAML no bloqueado: %v", err)
	}
	if _, err := datos.MarshalYAML(); !errors.Is(err, ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida) {
		t.Fatalf("YAML de datos no bloqueado: %v", err)
	}

	texto := fmt.Sprintf("%v|%+v|%#v|%s|%q", proyeccion, proyeccion, proyeccion, datos, datos)
	if strings.Contains(texto, datos.PrincipalID) || strings.Contains(texto, datos.DecisionRef) ||
		strings.Contains(texto, datos.VinculoAutenticacionActor.SesionRef) ||
		!strings.Contains(texto, representacionRedactadaProyeccionHistoricaAtestacionAutorizacionV1) {
		t.Fatalf("formato no redactado: %s", texto)
	}
	var registro bytes.Buffer
	slog.New(slog.NewTextHandler(&registro, nil)).Info("prueba", "proyeccion", proyeccion, "datos", datos)
	if strings.Contains(registro.String(), datos.PrincipalID) || strings.Contains(registro.String(), datos.VinculoAutenticacionActor.SesionRef) {
		t.Fatalf("slog filtro datos personales: %s", registro.String())
	}
}

func TestParserHistoricoAtestacionAutorizacionV1RechazaValorCeroYCadaCampo(t *testing.T) {
	var cero ProyeccionHistoricaAtestacionAutorizacionV1
	if _, err := cero.Cabecera(); !errors.Is(err, ErrParseoHistoricoAtestacionAutorizacionV1Invalido) {
		t.Fatalf("cabecera de valor cero aceptada: %v", err)
	}
	if _, err := cero.Datos(); !errors.Is(err, ErrParseoHistoricoAtestacionAutorizacionV1Invalido) {
		t.Fatalf("datos de valor cero aceptados: %v", err)
	}
	if _, err := ParsearMensajeAtestacionAutorizacionV1NoAutoritativo(nil); !errors.Is(err, ErrParseoHistoricoAtestacionAutorizacionV1Invalido) {
		t.Fatalf("mensaje cero aceptado: %v", err)
	}

	mensaje, err := SerializarMensajeAtestacionAutorizacionV1(
		cabeceraAtestacionAutorizacionV1Prueba(),
		decisionAtestacionAutorizacionV1Prueba(t),
	)
	if err != nil {
		t.Fatalf("crear mensaje: %v", err)
	}
	plano := analizarPlanoVECAD1Prueba(t, mensaje)
	if len(plano.campos) != 59 {
		t.Fatalf("matriz adversaria incompleta: %d campos atomicos/compuestos", len(plano.campos))
	}
	for _, campo := range plano.campos {
		t.Run(campo.nombre, func(t *testing.T) {
			mutado := append([]byte(nil), mensaje...)
			switch campo.tipo {
			case tipoCampoVECADTextoPrueba:
				if campo.contenidoInicio >= campo.fin {
					t.Fatal("fixture textual vacio")
				}
				mutado[campo.contenidoInicio] = 0
			case tipoCampoVECADBooleanoPrueba:
				mutado[campo.inicio] = 2
			case tipoCampoVECADUint16Prueba, tipoCampoVECADUint64Prueba:
				clear(mutado[campo.inicio:campo.fin])
			case tipoCampoVECADInstantePrueba:
				binary.BigEndian.PutUint64(mutado[campo.inicio:campo.fin], uint64(math.MaxInt64))
			case tipoCampoVECADListaPrueba, tipoCampoVECADMapaPrueba:
				binary.BigEndian.PutUint32(mutado[campo.inicio:campo.fin], uint32(maximoElementosAutorizacion+1))
			default:
				t.Fatalf("tipo de campo desconocido: %d", campo.tipo)
			}
			if _, err := ParsearMensajeAtestacionAutorizacionV1NoAutoritativo(mutado); !errors.Is(err, ErrParseoHistoricoAtestacionAutorizacionV1Invalido) {
				t.Fatalf("campo adulterado aceptado: %v", err)
			}
		})
	}
}

func TestParserHistoricoAtestacionAutorizacionV1RechazaFronterasYFormaNoCanonica(t *testing.T) {
	decision := decisionAtestacionAutorizacionV1Prueba(t)
	// Dos entradas en ambas colecciones de politicas permiten probar tambien
	// los campos que normalmente contienen una sola politica aplicable.
	decision.PoliticasRefs = append([]string(nil), decision.PoliticasEvaluadasRefs...)
	decision.PoliticasHuellasSHA256 = map[string]string{}
	for _, referencia := range decision.PoliticasRefs {
		decision.PoliticasHuellasSHA256[referencia] = decision.PoliticasEvaluadasHuellasSHA256[referencia]
	}
	mensaje, err := SerializarMensajeAtestacionAutorizacionV1(
		cabeceraAtestacionAutorizacionV1Prueba(), decision,
	)
	if err != nil {
		t.Fatalf("crear mensaje con colecciones dobles: %v", err)
	}
	plano := analizarPlanoVECAD1Prueba(t, mensaje)

	casos := map[string][]byte{}
	esquema := append([]byte(nil), mensaje...)
	esquema[0] ^= 1
	casos["esquema"] = esquema
	separador := append([]byte(nil), mensaje...)
	separador[len(EsquemaMensajeAtestacionAutorizacionV1)] = 1
	casos["separador"] = separador
	version := append([]byte(nil), mensaje...)
	campoVersion := plano.porNombre["cabecera.formato_version"]
	binary.BigEndian.PutUint16(version[campoVersion.inicio:campoVersion.fin], VersionFormatoAtestacionAutorizacionV1+1)
	casos["version"] = version
	utf8Invalido := append([]byte(nil), mensaje...)
	campoSuite := plano.porNombre["cabecera.suite"]
	utf8Invalido[campoSuite.contenidoInicio] = 0xff
	casos["utf8"] = utf8Invalido
	longitudTexto := append([]byte(nil), mensaje...)
	binary.BigEndian.PutUint32(longitudTexto[campoSuite.inicio:campoSuite.inicio+4], math.MaxUint32)
	casos["longitud_texto_sin_reserva"] = longitudTexto
	texto513 := append([]byte(nil), mensaje...)
	binary.BigEndian.PutUint32(texto513[campoSuite.inicio:campoSuite.inicio+4], 513)
	casos["texto_513_sin_reserva"] = texto513
	longitudFinal := append([]byte(nil), mensaje...)
	binary.BigEndian.PutUint64(longitudFinal[len(longitudFinal)-8:], uint64(len(longitudFinal)-1))
	casos["longitud_final"] = longitudFinal
	casos["trailing"] = append(append([]byte(nil), mensaje...), 0)
	casos["sobredimension"] = make([]byte, TamanoMaximoMensajeAtestacionAutorizacionV1+1)

	for _, nombre := range []string{
		"decision.politicas_evaluadas_refs",
		"decision.politicas_evaluadas_huellas_sha256",
		"decision.politicas_refs",
		"decision.politicas_huellas_sha256",
		"decision.campos_permitidos",
		"decision.obligaciones",
	} {
		entradas := plano.entradasColeccion[nombre]
		if len(entradas) != 2 {
			t.Fatalf("%s tiene %d entradas; se esperaban dos", nombre, len(entradas))
		}
		casos[nombre+"_desorden"] = reemplazarEntradasColeccionVECAD1Prueba(mensaje, entradas, []int{1, 0})
		casos[nombre+"_duplicado"] = reemplazarEntradasColeccionVECAD1Prueba(mensaje, entradas, []int{0, 0})
	}

	// La forma binaria puede seguir siendo canonica y, aun asi, describir una
	// semantica incoherente. Ambas adulteraciones deben cerrarse despues del
	// parseo estructural y antes de devolver la proyeccion.
	mapaEvaluadoDesigual := append([]byte(nil), mensaje...)
	entradaEvaluada := plano.entradasColeccion["decision.politicas_evaluadas_huellas_sha256"][0]
	valorInicio := inicioValorEntradaMapaVECAD1Prueba(t, mapaEvaluadoDesigual, entradaEvaluada)
	mapaEvaluadoDesigual[valorInicio] = '9'
	casos["mapa_evaluado_huella_distinta"] = mapaEvaluadoDesigual

	politicaExterna := "politica:externa:v9"
	aplicadaExterna := reemplazarIntervaloVECAD1Prueba(
		mensaje,
		plano.entradasColeccion["decision.politicas_refs"][0],
		codificarTextoVECAD1Prueba(politicaExterna),
	)
	planoAplicadaExterna := analizarPlanoVECAD1Prueba(t, aplicadaExterna)
	huellaExterna := decision.PoliticasHuellasSHA256[decision.PoliticasRefs[0]]
	aplicadaExterna = reemplazarIntervaloVECAD1Prueba(
		aplicadaExterna,
		planoAplicadaExterna.entradasColeccion["decision.politicas_huellas_sha256"][0],
		append(codificarTextoVECAD1Prueba(politicaExterna), codificarTextoVECAD1Prueba(huellaExterna)...),
	)
	casos["politica_aplicada_fuera_de_evaluadas"] = aplicadaExterna

	for nombre, contenido := range casos {
		t.Run(nombre, func(t *testing.T) {
			if _, err := ParsearMensajeAtestacionAutorizacionV1NoAutoritativo(contenido); !errors.Is(err, ErrParseoHistoricoAtestacionAutorizacionV1Invalido) ||
				!errors.Is(err, ErrMensajeAtestacionAutorizacionInvalido) {
				t.Fatalf("forma adversaria aceptada o error no clasificable: %v", err)
			}
		})
	}

	// Cada prefijo truncado debe cerrarse sin panico ni aceptacion parcial.
	for longitud := 0; longitud < len(mensaje); longitud++ {
		if _, err := ParsearMensajeAtestacionAutorizacionV1NoAutoritativo(mensaje[:longitud]); !errors.Is(err, ErrParseoHistoricoAtestacionAutorizacionV1Invalido) {
			t.Fatalf("truncado en %d aceptado: %v", longitud, err)
		}
	}
}

func FuzzParsearMensajeAtestacionAutorizacionV1NoAutoritativoNoEntraEnPanico(f *testing.F) {
	f.Add([]byte(nil))
	f.Add([]byte(EsquemaMensajeAtestacionAutorizacionV1))
	f.Add(append([]byte(EsquemaMensajeAtestacionAutorizacionV1), 0, 0, 1))
	f.Add([]byte{0, 1, 2, 3, 0xff, 0xff, 0xff, 0xff})
	f.Add(bytes.Repeat([]byte{0xff}, 1024))
	f.Fuzz(func(t *testing.T, contenido []byte) {
		_, _ = ParsearMensajeAtestacionAutorizacionV1NoAutoritativo(contenido)
	})
}

func datosDecisionHistoricaAtestacionAutorizacionV1Prueba(
	t *testing.T,
	d DecisionAutorizacion,
) DatosDecisionHistoricaAtestacionAutorizacionV1 {
	t.Helper()
	vinculo, err := d.VinculoAutenticacionActor.Datos()
	if err != nil {
		t.Fatalf("extraer datos de vinculo: %v", err)
	}
	return DatosDecisionHistoricaAtestacionAutorizacionV1{
		DecisionRef:                           d.DecisionRef,
		Concedida:                             d.Concedida,
		Codigo:                                d.Codigo,
		PrincipalID:                           d.PrincipalID,
		PerfilActivoRef:                       d.PerfilActivoRef,
		Accion:                                d.Accion,
		RecursoRef:                            d.RecursoRef,
		ModuloID:                              d.ModuloID,
		TipoRecurso:                           d.TipoRecurso,
		ContextoRecursoHuellaSHA256:           d.ContextoRecursoHuellaSHA256,
		Finalidad:                             d.Finalidad,
		CorrelacionRef:                        d.CorrelacionRef,
		VinculoAutenticacionActor:             vinculo,
		AsignacionRef:                         d.AsignacionRef,
		AsignacionHuellaSHA256:                d.AsignacionHuellaSHA256,
		VersionRolRef:                         d.VersionRolRef,
		VersionRolHuellaSHA256:                d.VersionRolHuellaSHA256,
		ControlVigenciaVersionRolRef:          d.ControlVigenciaVersionRolRef,
		ControlVigenciaVersionRolRevision:     d.ControlVigenciaVersionRolRevision,
		ControlVigenciaVersionRolHuellaSHA256: d.ControlVigenciaVersionRolHuellaSHA256,
		RevisionCatalogoPoliticas:             d.RevisionCatalogoPoliticas,
		CatalogoPoliticasHuellaSHA256:         d.CatalogoPoliticasHuellaSHA256,
		PoliticasEvaluadasRefs:                append([]string(nil), d.PoliticasEvaluadasRefs...),
		PoliticasEvaluadasHuellasSHA256:       clonarMapaTextoHistoricoAtestacionAutorizacionV1(d.PoliticasEvaluadasHuellasSHA256),
		PoliticasRefs:                         append([]string(nil), d.PoliticasRefs...),
		PoliticasHuellasSHA256:                clonarMapaTextoHistoricoAtestacionAutorizacionV1(d.PoliticasHuellasSHA256),
		GarantiaMinima:                        d.GarantiaMinima,
		CamposPermitidos:                      append([]string(nil), d.CamposPermitidos...),
		Obligaciones:                          append([]string(nil), d.Obligaciones...),
		EmitidaEn:                             d.EmitidaEn,
		ValidaHasta:                           d.ValidaHasta,
	}
}

type tipoCampoVECAD1Prueba uint8

const (
	tipoCampoVECADTextoPrueba tipoCampoVECAD1Prueba = iota + 1
	tipoCampoVECADBooleanoPrueba
	tipoCampoVECADUint16Prueba
	tipoCampoVECADUint64Prueba
	tipoCampoVECADInstantePrueba
	tipoCampoVECADListaPrueba
	tipoCampoVECADMapaPrueba
)

type campoVECAD1Prueba struct {
	nombre          string
	tipo            tipoCampoVECAD1Prueba
	inicio          int
	contenidoInicio int
	fin             int
}

type intervaloVECAD1Prueba struct {
	inicio int
	fin    int
}

type planoVECAD1Prueba struct {
	campos            []campoVECAD1Prueba
	porNombre         map[string]campoVECAD1Prueba
	entradasColeccion map[string][]intervaloVECAD1Prueba
}

func analizarPlanoVECAD1Prueba(t *testing.T, contenido []byte) planoVECAD1Prueba {
	t.Helper()
	posicion := len(EsquemaMensajeAtestacionAutorizacionV1) + 1
	plano := planoVECAD1Prueba{
		porNombre:         map[string]campoVECAD1Prueba{},
		entradasColeccion: map[string][]intervaloVECAD1Prueba{},
	}
	registrar := func(campo campoVECAD1Prueba) {
		plano.campos = append(plano.campos, campo)
		plano.porNombre[campo.nombre] = campo
	}
	exigir := func(cantidad int) {
		t.Helper()
		if cantidad < 0 || posicion > len(contenido)-cantidad {
			t.Fatalf("fixture VEC-AD-1 truncado en %d al pedir %d", posicion, cantidad)
		}
	}
	u16 := func(nombre string) {
		exigir(2)
		registrar(campoVECAD1Prueba{nombre: nombre, tipo: tipoCampoVECADUint16Prueba, inicio: posicion, fin: posicion + 2})
		posicion += 2
	}
	u64 := func(nombre string, instante bool) {
		exigir(8)
		tipo := tipoCampoVECADUint64Prueba
		if instante {
			tipo = tipoCampoVECADInstantePrueba
		}
		registrar(campoVECAD1Prueba{nombre: nombre, tipo: tipo, inicio: posicion, fin: posicion + 8})
		posicion += 8
	}
	booleano := func(nombre string) {
		exigir(1)
		registrar(campoVECAD1Prueba{nombre: nombre, tipo: tipoCampoVECADBooleanoPrueba, inicio: posicion, fin: posicion + 1})
		posicion++
	}
	leerTexto := func() intervaloVECAD1Prueba {
		exigir(4)
		inicio := posicion
		longitud := int(binary.BigEndian.Uint32(contenido[posicion : posicion+4]))
		posicion += 4
		exigir(longitud)
		posicion += longitud
		return intervaloVECAD1Prueba{inicio: inicio, fin: posicion}
	}
	texto := func(nombre string) {
		intervalo := leerTexto()
		registrar(campoVECAD1Prueba{
			nombre: nombre, tipo: tipoCampoVECADTextoPrueba,
			inicio: intervalo.inicio, contenidoInicio: intervalo.inicio + 4, fin: intervalo.fin,
		})
	}
	lista := func(nombre string) {
		exigir(4)
		inicio := posicion
		cantidad := binary.BigEndian.Uint32(contenido[posicion : posicion+4])
		posicion += 4
		registrar(campoVECAD1Prueba{nombre: nombre, tipo: tipoCampoVECADListaPrueba, inicio: inicio, fin: inicio + 4})
		for indice := uint32(0); indice < cantidad; indice++ {
			plano.entradasColeccion[nombre] = append(plano.entradasColeccion[nombre], leerTexto())
		}
	}
	mapa := func(nombre string) {
		exigir(4)
		inicio := posicion
		cantidad := binary.BigEndian.Uint32(contenido[posicion : posicion+4])
		posicion += 4
		registrar(campoVECAD1Prueba{nombre: nombre, tipo: tipoCampoVECADMapaPrueba, inicio: inicio, fin: inicio + 4})
		for indice := uint32(0); indice < cantidad; indice++ {
			entradaInicio := posicion
			_ = leerTexto()
			_ = leerTexto()
			plano.entradasColeccion[nombre] = append(plano.entradasColeccion[nombre], intervaloVECAD1Prueba{inicio: entradaInicio, fin: posicion})
		}
	}

	u16("cabecera.formato_version")
	texto("cabecera.suite")
	texto("cabecera.clave_id")
	texto("cabecera.audiencia")
	texto("decision.decision_ref")
	booleano("decision.concedida")
	texto("decision.codigo")
	texto("decision.principal_id")
	texto("decision.perfil_activo_ref")
	texto("decision.accion")
	texto("decision.recurso_ref")
	texto("decision.modulo_id")
	texto("decision.tipo_recurso")
	texto("decision.contexto_recurso_huella_sha256")
	texto("decision.finalidad")
	texto("decision.correlacion_ref")
	u16("vinculo.bloque_version")
	texto("vinculo.autenticacion_ref")
	texto("vinculo.autenticacion_huella_sha256")
	texto("vinculo.asercion_ref")
	texto("vinculo.sesion_ref")
	texto("vinculo.control_sesion_ref")
	u64("vinculo.control_sesion_revision", false)
	texto("vinculo.control_sesion_huella_sha256")
	texto("vinculo.cuenta_ref")
	texto("vinculo.cuenta_ordinaria_ref")
	texto("vinculo.principal_id")
	texto("vinculo.perfil_activo_ref")
	booleano("vinculo.cuenta_privilegiada")
	texto("vinculo.superficie")
	texto("vinculo.metodo_observado")
	texto("vinculo.garantia_observada")
	texto("vinculo.politica_garantia_ref")
	texto("vinculo.politica_garantia_huella_sha256")
	u64("vinculo.autenticacion_verificada_en", true)
	u64("vinculo.sesion_emitida_en", true)
	u64("vinculo.sesion_valida_hasta", true)
	u64("vinculo.sesion_revalidada_en", true)
	texto("vinculo.contexto_actor_ref")
	u64("vinculo.contexto_actor_version", false)
	texto("vinculo.contexto_actor_huella_sha256")
	texto("decision.asignacion_ref")
	texto("decision.asignacion_huella_sha256")
	texto("decision.version_rol_ref")
	texto("decision.version_rol_huella_sha256")
	texto("decision.control_vigencia_version_rol_ref")
	u64("decision.control_vigencia_version_rol_revision", false)
	texto("decision.control_vigencia_version_rol_huella_sha256")
	u64("decision.revision_catalogo_politicas", false)
	texto("decision.catalogo_politicas_huella_sha256")
	lista("decision.politicas_evaluadas_refs")
	mapa("decision.politicas_evaluadas_huellas_sha256")
	lista("decision.politicas_refs")
	mapa("decision.politicas_huellas_sha256")
	texto("decision.garantia_minima")
	lista("decision.campos_permitidos")
	lista("decision.obligaciones")
	u64("decision.emitida_en", true)
	u64("decision.valida_hasta", true)
	exigir(8)
	posicion += 8
	if posicion != len(contenido) {
		t.Fatalf("plano consumio %d de %d bytes", posicion, len(contenido))
	}
	return plano
}

func reemplazarEntradasColeccionVECAD1Prueba(
	original []byte,
	entradas []intervaloVECAD1Prueba,
	orden []int,
) []byte {
	inicio := entradas[0].inicio
	fin := entradas[len(entradas)-1].fin
	resultado := make([]byte, 0, len(original))
	resultado = append(resultado, original[:inicio]...)
	for _, indice := range orden {
		resultado = append(resultado, original[entradas[indice].inicio:entradas[indice].fin]...)
	}
	resultado = append(resultado, original[fin:]...)
	binary.BigEndian.PutUint64(resultado[len(resultado)-8:], uint64(len(resultado)))
	return resultado
}

func inicioValorEntradaMapaVECAD1Prueba(
	t *testing.T,
	contenido []byte,
	entrada intervaloVECAD1Prueba,
) int {
	t.Helper()
	if entrada.inicio < 0 || entrada.fin > len(contenido) || entrada.fin-entrada.inicio < 8 {
		t.Fatal("intervalo de mapa invalido")
	}
	longitudClave := int(binary.BigEndian.Uint32(contenido[entrada.inicio : entrada.inicio+4]))
	inicioLongitudValor := entrada.inicio + 4 + longitudClave
	if inicioLongitudValor > entrada.fin-4 {
		t.Fatal("clave de mapa truncada")
	}
	longitudValor := int(binary.BigEndian.Uint32(contenido[inicioLongitudValor : inicioLongitudValor+4]))
	inicioValor := inicioLongitudValor + 4
	if longitudValor == 0 || inicioValor > entrada.fin-longitudValor {
		t.Fatal("valor de mapa truncado o vacio")
	}
	return inicioValor
}

func codificarTextoVECAD1Prueba(valor string) []byte {
	resultado := make([]byte, 4, 4+len(valor))
	binary.BigEndian.PutUint32(resultado, uint32(len(valor)))
	return append(resultado, []byte(valor)...)
}

func reemplazarIntervaloVECAD1Prueba(
	original []byte,
	intervalo intervaloVECAD1Prueba,
	reemplazo []byte,
) []byte {
	resultado := make([]byte, 0, len(original)-(intervalo.fin-intervalo.inicio)+len(reemplazo))
	resultado = append(resultado, original[:intervalo.inicio]...)
	resultado = append(resultado, reemplazo...)
	resultado = append(resultado, original[intervalo.fin:]...)
	binary.BigEndian.PutUint64(resultado[len(resultado)-8:], uint64(len(resultado)))
	return resultado
}

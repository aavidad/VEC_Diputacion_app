package ports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

type escenarioEjecucionDocumentalV3Prueba struct {
	instante               time.Time
	manifiesto             ManifiestoEjecucionDocumentalV3
	preparar               SolicitudPrepararEjecucionDocumentalV3
	consumo                ConsumoDecisionEjecucionDocumentalV3
	vinculoActivacion      VinculoEstableActivacionDocumentalV3
	token                  TokenCercadoEjecucionDocumentalV3Nominal
	verificacionCercado    MetadatosComprobacionTokenCercadoDocumentalV3Nominal
	ordenDespachoConsumida OrdenDespachoDocumentalV3ConsumidaNominal
	resultado              ResultadoEfectoRenderizadoDocumentalV3Crudo
	recibos                RecibosEjecucionDocumentalV3
	evidencia              DatosEvidenciaRenderizadoDocumentalV3
	sello                  SelloEvidenciaDocumentalV3Nominal
	verificacion           MetadatosComprobacionEvidenciaDocumentalV3Nominal
	confirmacion           SolicitudConfirmarEjecucionDocumentalV3
}

func TestArquitecturaDocumentalV3SoloExponeComandosConsumidosNominales(t *testing.T) {
	prohibido := "Comprobada" + "TCB"
	promotoresProhibidos := map[string]struct{}{
		"NuevoComprobadorOrdenDespachoDocumentalV3" + "TCB": {},
		"nuevoComprobadorOrdenDespachoDocumentalV3" + "TCB": {},
	}
	nombresAutoritativosProhibidos := map[string]struct{}{
		"DatosReciboInicioEfectoDocumentalV3":            {},
		"ReciboInicioEfectoDocumentalV3":                 {},
		"MetadatosComprobacionTokenCercadoDocumentalV3":  {},
		"VerificadorTokensCercadoDocumentalV3":           {},
		"ComprobadorCriptograficoRecibosDocumentalesTCB": {},
		"TokenCercadoEjecucionDocumentalV3":              {},
		"NuevoTokenCercadoEjecucionDocumentalV3":         {},
		"PreparacionEjecucionDocumentalV3":               {},
		"ActivacionEjecucionDocumentalV3":                {},
		"InstantaneaEjecucionDocumentalV3":               {},
		"ResultadoConsultaEfectoDocumentalV3":            {},
		"ResultadoEfectoRenderizadoDocumentalV3":         {},
		"DatosSelloEvidenciaDocumentalV3":                {},
		"SelloEvidenciaDocumentalV3":                     {},
		"NuevoSelloEvidenciaDocumentalV3":                {},
		"SobreAtestacionReconciliacionDocumentalV3":      {},
		"NuevoSobreAtestacionReconciliacionDocumentalV3": {},
		"SobreReciboEjecucionDocumental":                 {},
		"NuevoSobreReciboEjecucionDocumental":            {},
		"PreparacionEntradaNeutralDocumental":            {},
		"PrepararEntradaNeutralDocumental":               {},
		"EntradaNeutralDocumental":                       {},
		"NuevaEntradaNeutralDocumental":                  {},
		"PreparacionEscrituraAlmacenDocumentalV4":        {},
		"PrepararEscrituraAlmacenDocumentalV4":           {},
	}
	for _, nombre := range []string{
		"ejecuciones_documentales_v3.go",
		"ejecucion_componentes_documentales_atestada.go",
		"materializacion_documental_v4.go",
	} {
		ruta := filepath.Clean(nombre)
		archivo, err := parser.ParseFile(token.NewFileSet(), ruta, nil, 0)
		if err != nil {
			t.Fatalf("parsear %s: %v", ruta, err)
		}
		ast.Inspect(archivo, func(nodo ast.Node) bool {
			identificador, ok := nodo.(*ast.Ident)
			if !ok {
				return true
			}
			if strings.Contains(identificador.Name, prohibido) {
				t.Errorf("%s expone identificador autoritativo prohibido %s", ruta, identificador.Name)
			}
			if ast.IsExported(identificador.Name) && strings.Contains(identificador.Name, "TCB") {
				t.Errorf("%s expone TCB componible desde ports en %s", ruta, identificador.Name)
			}
			if _, existe := promotoresProhibidos[identificador.Name]; existe {
				t.Errorf("%s conserva promotor componible desde puertos: %s", ruta, identificador.Name)
			}
			if _, existe := nombresAutoritativosProhibidos[identificador.Name]; existe {
				t.Errorf("%s conserva nombre publico ambiguo: %s", ruta, identificador.Name)
			}
			return true
		})
	}

	for _, valor := range []any{
		SolicitudMarcarEjecucionDocumentalV3Indeterminada{},
		SolicitudConfirmarEjecucionDocumentalV3{},
		SolicitudConsultarEfectoDocumentalV3{},
		CompromisoEjecucionComponenteDocumental{},
		VinculoEjecucionEscrituraAlmacenDocumental{},
	} {
		tipo := reflect.TypeOf(valor)
		for indice := 0; indice < tipo.NumField(); indice++ {
			campo := tipo.Field(indice)
			if strings.Contains(campo.Name, prohibido) || strings.Contains(campo.Type.String(), prohibido) {
				t.Fatalf("%s conserva capacidad autoritativa en %s", tipo.Name(), campo.Name)
			}
		}
	}
	if !strings.Contains((OrdenDespachoDocumentalV3ConsumidaNominal{}).String(), "NO-AUTORITATIVA") {
		t.Fatal("el comando consumido no declara explicitamente su caracter nominal")
	}
}

func TestValidadoresDocumentalesConValorCeroFallanCerradoSinPanic(t *testing.T) {
	casos := []struct {
		nombre  string
		validar func() error
	}{
		{"preparacion", func() error {
			return (PreparacionEjecucionDocumentalV3Nominal{}).ValidarContra(
				SolicitudPrepararEjecucionDocumentalV3{},
			)
		}},
		{"consumo decision", func() error {
			return (ConsumoDecisionEjecucionDocumentalV3{}).ValidarContra(
				ManifiestoEjecucionDocumentalV3{},
			)
		}},
		{"token", func() error {
			return (TokenCercadoEjecucionDocumentalV3Nominal{}).ValidarPara(
				VinculoEstableActivacionDocumentalV3{},
			)
		}},
		{"metadatos token", func() error {
			return (MetadatosComprobacionTokenCercadoDocumentalV3Nominal{}).ValidarPara(
				SolicitudVerificacionTokenCercadoDocumentalV3{},
			)
		}},
		{"activacion", func() error {
			return (ActivacionEjecucionDocumentalV3Nominal{}).ValidarContra(
				SolicitudActivarEjecucionDocumentalV3{},
			)
		}},
		{"recibo inicio", func() error {
			return (ReciboInicioEfectoDocumentalV3Nominal{}).ValidarContra(
				SolicitudIniciarEfectoDocumentalV3{},
			)
		}},
		{"resultado KMS", func() error {
			return (ResultadoCrudoVerificacionOrdenDespachoDocumentalV3{}).ValidarPara(
				SolicitudComprobarOrdenDespachoDocumentalV3{},
			)
		}},
		{"estado consumo", func() error {
			return (EstadoCrudoOrdenDespachoDocumentalV3{}).ValidarPara(
				SolicitudComprobarOrdenDespachoDocumentalV3{},
				ResultadoCrudoVerificacionOrdenDespachoDocumentalV3{},
			)
		}},
		{"orden consumida", func() error {
			return (OrdenDespachoDocumentalV3ConsumidaNominal{}).ValidarEn(time.Time{})
		}},
		{"resultado renderizado", func() error {
			return (ResultadoEfectoRenderizadoDocumentalV3Crudo{}).ValidarContra(
				ManifiestoEjecucionDocumentalV3{},
			)
		}},
		{"recibos componentes", func() error {
			return (RecibosEjecucionDocumentalV3{}).ValidarContra(
				ManifiestoEjecucionDocumentalV3{}, ResultadoEfectoRenderizadoDocumentalV3Crudo{},
				"", ConsumoDecisionEjecucionDocumentalV3{},
				OrdenDespachoDocumentalV3ConsumidaNominal{}, time.Time{},
			)
		}},
		{"sello evidencia", func() error {
			return (SelloEvidenciaDocumentalV3Nominal{}).ValidarPara(
				SolicitudFirmaEvidenciaRenderizadoDocumentalV3{},
			)
		}},
		{"metadatos evidencia", func() error {
			return (MetadatosComprobacionEvidenciaDocumentalV3Nominal{}).ValidarPara(
				SolicitudVerificacionEvidenciaDocumentalV3{},
			)
		}},
		{"consulta reconciliacion", func() error {
			return (ResultadoConsultaEfectoDocumentalV3Crudo{}).ValidarContra(
				SolicitudConsultarEfectoDocumentalV3{},
			)
		}},
		{"metadatos reconciliacion", func() error {
			return (MetadatosComprobacionReconciliacionDocumentalV3Nominal{}).ValidarPara(
				SolicitudVerificacionReconciliacionDocumentalV3{},
			)
		}},
		{"recibo componente", func() error {
			return (ReciboEjecucionComponenteDocumentalNominal{}).ValidarContra(
				CompromisoEjecucionComponenteDocumental{}, SobreReciboEjecucionDocumentalCrudo{},
			)
		}},
		{"salida observada", func() error {
			return (SalidaObservadaDocumental{}).ValidarContraSolicitudEscribirObjeto(
				SolicitudEscribirObjeto{},
			)
		}},
		{"vinculo escritura", func() error {
			return (VinculoEjecucionEscrituraAlmacenDocumental{}).ValidarContra(
				OrdenDespachoDocumentalV3ConsumidaNominal{},
			)
		}},
		{"declaracion y salida", func() error {
			return (DeclaracionEscrituraAlmacenDocumental{}).ValidarContraSalida(
				SalidaObservadaDocumental{},
			)
		}},
		{"declaracion y ejecucion", func() error {
			return (DeclaracionEscrituraAlmacenDocumental{}).ValidarContraEjecucion(
				OrdenDespachoDocumentalV3ConsumidaNominal{}, SalidaObservadaDocumental{},
			)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			defer func() {
				if recuperado := recover(); recuperado != nil {
					t.Fatalf("valor cero provoco panic: %v", recuperado)
				}
			}()
			if err := caso.validar(); err == nil {
				t.Fatal("valor cero aceptado")
			}
		})
	}
}

func TestFabricasDocumentalesDeEfectoNoAceptanTokenNiMetadatosAutocreados(t *testing.T) {
	prohibidos := map[reflect.Type]struct{}{
		reflect.TypeOf(TokenCercadoEjecucionDocumentalV3Nominal{}):               {},
		reflect.TypeOf(MetadatosComprobacionTokenCercadoDocumentalV3Nominal{}):   {},
		reflect.TypeOf(MetadatosComprobacionEvidenciaDocumentalV3Nominal{}):      {},
		reflect.TypeOf(MetadatosComprobacionReconciliacionDocumentalV3Nominal{}): {},
	}
	for nombre, fabrica := range map[string]any{
		"orden consumida nominal": NuevaOrdenDespachoDocumentalV3ConsumidaNominal,
		"compromiso componente":   NuevoCompromisoEjecucionComponenteDocumental,
		"vinculo materializacion": NuevoVinculoEjecucionEscrituraAlmacenDocumental,
	} {
		tipo := reflect.TypeOf(fabrica)
		for indice := 0; indice < tipo.NumIn(); indice++ {
			if _, prohibido := prohibidos[tipo.In(indice)]; prohibido {
				t.Fatalf("%s acepta autoridad publica autocreable en argumento %d", nombre, indice)
			}
		}
	}
}

func TestOrdenDespachoConsumidaNominalLigaCASYFallaCerrada(t *testing.T) {
	escenario := nuevoEscenarioEjecucionDocumentalV3Prueba(t)
	orden := escenario.ordenDespachoConsumida
	datos, err := orden.DatosOrden()
	if err != nil || orden.ValidarEn(orden.estado.consumidaEn) != nil {
		t.Fatalf("orden nominal consumida valida rechazada: %v", err)
	}
	if orden.ValidarEn(orden.estado.consumidaEn.Add(-time.Microsecond)) == nil {
		t.Fatal("orden utilizable antes del consumo CAS")
	}
	if orden.ValidarEn(datos.ExpiraEn) == nil {
		t.Fatal("orden utilizable en el borde exclusivo de expiracion")
	}

	mutaciones := map[string]func(*OrdenDespachoDocumentalV3ConsumidaNominal){
		"replay version consumo": func(o *OrdenDespachoDocumentalV3ConsumidaNominal) {
			o.estado.versionConsumoCAS = o.estado.versionReclamacionCAS
		},
		"consumo cruzado": func(o *OrdenDespachoDocumentalV3ConsumidaNominal) {
			o.estado.consumoRef = o.resultado.comprobacionRef
		},
		"resultado KMS": func(o *OrdenDespachoDocumentalV3ConsumidaNominal) {
			o.resultado.huellaAtestacionSHA256 = strings.Repeat("0", 64)
		},
		"secuencia cercado": func(o *OrdenDespachoDocumentalV3ConsumidaNominal) {
			o.solicitud.orden.datos.ReciboInicio.SecuenciaCercado++
		},
		"vinculo estable": func(o *OrdenDespachoDocumentalV3ConsumidaNominal) {
			o.solicitud.vinculo.ReservaRef = "reserva:documental:v3:otra"
		},
	}
	for nombre, mutar := range mutaciones {
		t.Run(nombre, func(t *testing.T) {
			alterada := clonarOrdenDespachoDocumentalV3ConsumidaNominal(orden)
			mutar(&alterada)
			alterada.huellaConsumo = huellaCanonicaFormatoDocumental([]string{
				"vec.documentos.orden-despacho-consumida-nominal.v3", alterada.solicitud.huella,
				alterada.resultado.huellaSHA256(), alterada.estado.huellaSHA256(),
			})
			if alterada.ValidarEn(orden.estado.consumidaEn) == nil {
				t.Fatal("la manipulacion nominal con huella recalculada conservo validez")
			}
		})
	}

	texto := fmt.Sprintf("%v|%+v|%#v", orden, orden, orden)
	if strings.Contains(texto, orden.estado.consumoRef) ||
		strings.Contains(texto, orden.resultado.comprobacionRef) {
		t.Fatal("el comando nominal filtro referencias internas")
	}
	if _, err := json.Marshal(orden); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("orden nominal serializada por JSON: %v", err)
	}
	if _, err := orden.MarshalText(); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("orden nominal serializada como texto: %v", err)
	}
	if _, err := orden.MarshalBinary(); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("orden nominal serializada como binario: %v", err)
	}
}

func TestVerificadorOrdenDespachoDocumentalV3LeeMaterialCriptograficoReal(t *testing.T) {
	escenario := nuevoEscenarioEjecucionDocumentalV3Prueba(t)
	solicitud := escenario.ordenDespachoConsumida.solicitud
	verificador := &verificadorOrdenDespachoDocumentalV3Prueba{
		sufijo: "adversarial", comprobadaEn: escenario.ordenDespachoConsumida.estado.consumidaEn,
	}
	resultadoCrudo, err := verificador.VerificarOrdenDespachoDocumentalV3(
		context.Background(), solicitud,
	)
	if err != nil {
		t.Fatalf("material criptografico valido rechazado: %v", err)
	}
	datosCrudos, err := resultadoCrudo.DatosCrudos()
	if err != nil || datosCrudos.ComprobacionRef != resultadoCrudo.comprobacionRef ||
		datosCrudos.EvidenciaOperacionRef != resultadoCrudo.evidenciaOperacionRef ||
		datosCrudos.ClaveGestionadaRef != resultadoCrudo.claveGestionadaRef ||
		datosCrudos.RevisionClaveGestionada != resultadoCrudo.revisionClaveGestionada ||
		datosCrudos.Algoritmo != AlgoritmoSelloEvidenciaHMACSHA256V3 ||
		datosCrudos.Audiencia != AudienciaComprobacionOrdenDespachoV3 ||
		datosCrudos.Contexto != ContextoComprobacionOrdenDespachoV3 ||
		datosCrudos.HuellaResultadoCrudoSHA256 != resultadoCrudo.huellaSHA256() {
		t.Fatalf("el DTO externo perdio material KMS: %+v, %v", datosCrudos, err)
	}
	datosCrudos.ComprobacionRef = "comprobacion:alterada:solo-en-copia"
	datosCrudosNuevos, err := resultadoCrudo.DatosCrudos()
	if err != nil || datosCrudosNuevos.ComprobacionRef != resultadoCrudo.comprobacionRef {
		t.Fatal("el DTO externo compartio estado mutable con el resultado")
	}

	contextoCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := verificador.VerificarOrdenDespachoDocumentalV3(contextoCancelado, solicitud); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion no propagada: %v", err)
	}
	var verificadorNulo *verificadorOrdenDespachoDocumentalV3Prueba
	var puerto VerificadorOrdenDespachoDocumentalV3 = verificadorNulo
	if _, err := puerto.VerificarOrdenDespachoDocumentalV3(context.Background(), solicitud); err == nil {
		t.Fatal("receptor typed nil aceptado")
	}

	t.Run("MAC cercado alterado llega al verificador y falla", func(t *testing.T) {
		token := clonarTokenCercadoEjecucionDocumentalV3(solicitud.token)
		token.macAtestacion[0] ^= 0xff
		alterada, err := NuevaSolicitudComprobarOrdenDespachoDocumentalV3(
			clonarOrdenDespachoDocumentalV3Nominal(solicitud.orden), solicitud.vinculo, token,
		)
		if err != nil || alterada.Validar() != nil {
			t.Fatalf("la alteracion criptografica debia seguir siendo estructuralmente nominal: %v", err)
		}
		if _, err := verificador.VerificarOrdenDespachoDocumentalV3(context.Background(), alterada); err == nil {
			t.Fatal("MAC cercado alterado aceptado")
		}
	})

	for _, fase := range []string{"inicio", "reclamacion"} {
		t.Run("sobre "+fase+" alterado llega al verificador y falla", func(t *testing.T) {
			orden := clonarOrdenDespachoDocumentalV3Nominal(solicitud.orden)
			prueba := &orden.datos.AtestacionReclamacion
			if fase == "inicio" {
				prueba = &orden.datos.ReciboInicio.AtestacionInicio
			}
			prueba.sobreCriptografico[0] ^= 0xff
			prueba.huellaSobreSHA256 = huellaBytesFormatoDocumental(prueba.sobreCriptografico)
			if fase == "inicio" {
				recibo := ReciboInicioEfectoDocumentalV3Nominal{datos: &orden.datos.ReciboInicio}
				huellaRecibo, err := recibo.HuellaSHA256()
				if err != nil {
					t.Fatal(err)
				}
				orden.datos.HuellaReciboInicioSHA256 = huellaRecibo
				solicitudReclamacion := SolicitudReclamarOrdenDespachoDocumentalV3{
					ReclamacionRef: orden.datos.ReclamacionRef,
					InicioRef:      orden.datos.ReciboInicio.InicioRef,
					OutboxRef:      orden.datos.ReciboInicio.OutboxInicioRef,
					ConsumidorRef:  orden.datos.ConsumidorRef,
					ReclamadaEn:    orden.datos.ReclamadaEn,
					ExpiraEn:       orden.datos.ExpiraEn,
				}
				evidenciaRef, _ := orden.datos.AtestacionReclamacion.EvidenciaOperacionRef()
				mensaje, err := MensajeCanonicoAtestacionReclamacionDespachoDocumentalV3(
					recibo, solicitudReclamacion, orden.datos.VersionReclamacionCAS,
					orden.datos.AuditoriaReclamacionRef, evidenciaRef,
				)
				if err != nil {
					t.Fatal(err)
				}
				orden.datos.AtestacionReclamacion.mensajeCanonico = mensaje
				orden.datos.AtestacionReclamacion.huellaMensajeSHA256 = huellaBytesFormatoDocumental(mensaje)
				orden.datos.AtestacionReclamacion.sobreCriptografico = firmarAtestacionDocumentalV3Prueba(
					mensaje, orden.datos.AtestacionReclamacion.claveGestionadaRef,
					orden.datos.AtestacionReclamacion.revisionClaveGestionada,
				)
				orden.datos.AtestacionReclamacion.huellaSobreSHA256 = huellaBytesFormatoDocumental(
					orden.datos.AtestacionReclamacion.sobreCriptografico,
				)
			}
			alterada, err := NuevaSolicitudComprobarOrdenDespachoDocumentalV3(
				orden, solicitud.vinculo, solicitud.token,
			)
			if err != nil || alterada.Validar() != nil {
				t.Fatalf("el sobre alterado debia conservar forma nominal: %v", err)
			}
			if _, err := verificador.VerificarOrdenDespachoDocumentalV3(context.Background(), alterada); err == nil {
				t.Fatalf("sobre %s alterado aceptado", fase)
			}
		})
	}

	mutacionesEstructurales := map[string]func(*OrdenDespachoDocumentalV3Nominal, *VinculoEstableActivacionDocumentalV3){
		"referencia atestacion": func(o *OrdenDespachoDocumentalV3Nominal, _ *VinculoEstableActivacionDocumentalV3) {
			o.datos.AtestacionReclamacion.evidenciaOperacionRef = "atestacion:reclamacion:alterada"
		},
		"huella atestacion": func(o *OrdenDespachoDocumentalV3Nominal, _ *VinculoEstableActivacionDocumentalV3) {
			o.datos.AtestacionReclamacion.huellaSobreSHA256 = strings.Repeat("0", 64)
		},
		"clave": func(o *OrdenDespachoDocumentalV3Nominal, _ *VinculoEstableActivacionDocumentalV3) {
			o.datos.AtestacionReclamacion.claveGestionadaRef = "clave:cercado:alterada"
		},
		"audiencia": func(o *OrdenDespachoDocumentalV3Nominal, _ *VinculoEstableActivacionDocumentalV3) {
			o.datos.AtestacionReclamacion.audiencia = AudienciaAtestacionInicioEfectoV3
		},
		"vinculo": func(_ *OrdenDespachoDocumentalV3Nominal, v *VinculoEstableActivacionDocumentalV3) {
			v.ReservaRef = "reserva:documental:v3:alterada"
		},
		"secuencia": func(o *OrdenDespachoDocumentalV3Nominal, _ *VinculoEstableActivacionDocumentalV3) {
			o.datos.ReciboInicio.SecuenciaCercado++
		},
	}
	for nombre, mutar := range mutacionesEstructurales {
		t.Run(nombre, func(t *testing.T) {
			orden := clonarOrdenDespachoDocumentalV3Nominal(solicitud.orden)
			vinculo := clonarVinculoEstableActivacionDocumentalV3(solicitud.vinculo)
			mutar(&orden, &vinculo)
			alterada, err := NuevaSolicitudComprobarOrdenDespachoDocumentalV3(
				orden, vinculo, solicitud.token,
			)
			if err == nil {
				_, err = verificador.VerificarOrdenDespachoDocumentalV3(context.Background(), alterada)
			}
			if err == nil {
				t.Fatal("alteracion aceptada")
			}
		})
	}
}

func TestConsumidorOrdenDespachoDocumentalV3RechazaResultadoKMSDeOtraSolicitudAntesDelCAS(t *testing.T) {
	escenarioA := nuevoEscenarioEjecucionDocumentalV3Prueba(t)
	datosOrdenA, err := escenarioA.ordenDespachoConsumida.solicitud.orden.Datos()
	if err != nil {
		t.Fatal(err)
	}
	ordenB := ordenDespachoDocumentalV3ConsumidaNominalPrueba(
		t, escenarioA.vinculoActivacion, escenarioA.token,
		escenarioA.ordenDespachoConsumida.resultado.comprobadaEn,
		datosOrdenA.ExpiraEn, "cruce-kms-b",
	)
	if ordenB.resultado.ValidarPara(ordenB.solicitud) != nil ||
		escenarioA.ordenDespachoConsumida.resultado.ValidarPara(
			escenarioA.ordenDespachoConsumida.solicitud,
		) != nil {
		t.Fatal("los dos pares de control debian ser validos por separado")
	}

	consumidor := &consumidorOrdenDespachoDocumentalV3Prueba{
		sufijo: "cruce-kms", consumidaEn: escenarioA.ordenDespachoConsumida.estado.consumidaEn,
	}
	if _, err := consumidor.ReleerYConsumirOrdenDespachoDocumentalV3(
		context.Background(), escenarioA.ordenDespachoConsumida.solicitud, ordenB.resultado,
	); !errors.Is(err, ErrOrdenDespachoDocumentalV3Invalida) {
		t.Fatalf("resultado KMS de B no fue rechazado para A: %v", err)
	}
	if consumidor.intentosCAS != 0 || consumidor.estadosPersistidos != 0 {
		t.Fatalf(
			"la discordancia A/B alcanzo el CAS o persistio estado: intentos=%d estados=%d",
			consumidor.intentosCAS, consumidor.estadosPersistidos,
		)
	}
	if _, err := NuevoEstadoCrudoOrdenDespachoDocumentalV3(
		escenarioA.ordenDespachoConsumida.solicitud, ordenB.resultado,
		"estado:consumido:cruce-kms", "auditoria:consumo:cruce-kms",
		"consumo:despacho:cruce-kms", "outbox:consumo:cruce-kms", 3,
		escenarioA.ordenDespachoConsumida.estado.consumidaEn,
	); !errors.Is(err, ErrOrdenDespachoDocumentalV3Invalida) {
		t.Fatalf("el constructor de estado acepto el cruce A/B: %v", err)
	}

	contextoCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := consumidor.ReleerYConsumirOrdenDespachoDocumentalV3(
		contextoCancelado, escenarioA.ordenDespachoConsumida.solicitud,
		escenarioA.ordenDespachoConsumida.resultado,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("el consumidor no propago cancelacion: %v", err)
	}
	var consumidorNulo *consumidorOrdenDespachoDocumentalV3Prueba
	var puerto ConsumidorPrivadoOrdenDespachoDocumentalV3 = consumidorNulo
	if _, err := puerto.ReleerYConsumirOrdenDespachoDocumentalV3(
		context.Background(), escenarioA.ordenDespachoConsumida.solicitud,
		escenarioA.ordenDespachoConsumida.resultado,
	); err == nil {
		t.Fatal("receptor typed nil aceptado")
	}
}

func TestValoresSensiblesDocumentalesV3SeRedactanYBloqueanSerializacion(t *testing.T) {
	escenario := nuevoEscenarioEjecucionDocumentalV3Prueba(t)
	solicitudToken, err := NuevaSolicitudVerificacionTokenCercadoDocumentalV3(
		escenario.vinculoActivacion, escenario.token,
	)
	if err != nil {
		t.Fatal(err)
	}
	perfil, err := NuevoPerfilSelloEvidenciaHMACSHA256V3("clave:evidencia:v3")
	if err != nil {
		t.Fatal(err)
	}
	firma, err := NuevaSolicitudFirmaEvidenciaRenderizadoDocumentalV3(perfil, escenario.evidencia)
	if err != nil {
		t.Fatal(err)
	}
	solicitudEvidencia, err := NuevaSolicitudVerificacionEvidenciaDocumentalV3(firma, escenario.sello)
	if err != nil {
		t.Fatal(err)
	}
	datosSello, err := escenario.sello.Datos()
	if err != nil {
		t.Fatal(err)
	}
	orden := escenario.ordenDespachoConsumida
	datosManifiesto, _ := escenario.manifiesto.Datos()
	datosOrden, _ := orden.solicitud.orden.Datos()
	datosResultadoKMS, _ := orden.resultado.DatosCrudos()
	material, _ := orden.solicitud.MaterialCrudo()
	cercado, inicio, reclamacion, _ := material.Pruebas()
	vinculosMaterial, _ := material.Vinculos()
	consulta := ConsultaEjecucionDocumentalV3{
		IndiceIdempotenciaHMAC: escenario.preparar.IndiceIdempotenciaHMAC,
		HuellaSolicitudHMAC:    escenario.preparar.HuellaSolicitudHMAC,
	}
	instantanea := InstantaneaEjecucionDocumentalV3Nominal{
		IndiceIdempotenciaHMAC: escenario.preparar.IndiceIdempotenciaHMAC,
		HuellaSolicitudHMAC:    escenario.preparar.HuellaSolicitudHMAC,
	}
	valores := map[string]any{
		"datos manifiesto":      datosManifiesto,
		"manifiesto":            escenario.manifiesto,
		"solicitud preparar":    escenario.preparar,
		"consulta":              consulta,
		"instantanea":           instantanea,
		"token":                 escenario.token,
		"solicitud token":       solicitudToken,
		"metadatos token":       escenario.verificacionCercado,
		"solicitud KMS":         orden.solicitud,
		"resultado KMS":         orden.resultado,
		"datos resultado KMS":   datosResultadoKMS,
		"estado consumo":        orden.estado,
		"orden consumida":       orden,
		"datos orden":           datosOrden,
		"material KMS":          material,
		"prueba cercado":        cercado,
		"prueba inicio":         inicio,
		"prueba reclamacion":    reclamacion,
		"vinculos material":     vinculosMaterial,
		"datos evidencia":       escenario.evidencia,
		"perfil sello":          perfil,
		"solicitud firma":       firma,
		"datos sello":           datosSello,
		"sello":                 escenario.sello,
		"resultado renderizado": escenario.resultado,
		"solicitud evidencia":   solicitudEvidencia,
		"metadatos evidencia":   escenario.verificacion,
	}
	secretos := []string{
		escenario.token.valor, escenario.token.claveAtestacionRef,
		escenario.preparar.IndiceIdempotenciaHMAC, escenario.preparar.HuellaSolicitudHMAC,
		datosManifiesto.HuellaEntradaHMAC, orden.estado.consumoRef,
		datosSello.EvidenciaOperacionRef, escenario.resultado.ContenidoRef,
	}
	for nombre, valor := range valores {
		texto := fmt.Sprintf("%v|%+v|%#v", valor, valor, valor)
		for _, secreto := range secretos {
			if secreto != "" && strings.Contains(texto, secreto) {
				t.Fatalf("%s filtro %q por formato", nombre, secreto)
			}
		}
		resuelto := slog.Any("valor", valor).Value.Resolve()
		if resuelto.Kind() != slog.KindString {
			t.Fatalf("%s no se redujo a texto redactado en slog: %v", nombre, resuelto.Kind())
		}
		if _, err := json.Marshal(valor); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
			t.Fatalf("%s se serializo por JSON: %v", nombre, err)
		}
		serializadorTexto, ok := valor.(interface{ MarshalText() ([]byte, error) })
		if !ok {
			t.Fatalf("%s no bloquea MarshalText", nombre)
		}
		if _, err := serializadorTexto.MarshalText(); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
			t.Fatalf("%s se serializo como texto: %v", nombre, err)
		}
		serializadorBinario, ok := valor.(interface{ MarshalBinary() ([]byte, error) })
		if !ok {
			t.Fatalf("%s no bloquea MarshalBinary", nombre)
		}
		if _, err := serializadorBinario.MarshalBinary(); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
			t.Fatalf("%s se serializo como binario: %v", nombre, err)
		}
	}
}

func TestManifiestoEjecucionDocumentalV3EsCanonicoTripleEIdempotente(t *testing.T) {
	escenario := nuevoEscenarioEjecucionDocumentalV3Prueba(t)
	if err := escenario.manifiesto.Validar(); err != nil {
		t.Fatalf("manifiesto valido rechazado: %v", err)
	}
	datos, err := escenario.manifiesto.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if datos.LimiteEfectivoBytes != datos.ComponenteSemantico.MaximoBytes() ||
		datos.LimiteEfectivoBytes >= datos.ComponenteRender.MaximoBytes() ||
		datos.LimiteEfectivoBytes >= datos.ComponenteVerificador.MaximoBytes() {
		t.Fatal("el limite efectivo no quedo sujeto al menor de los tres componentes")
	}
	huella := datos.HuellaPlanSHA256
	datos.HuellaPlanSHA256 = strings.Repeat("f", 64)
	if escenario.manifiesto.Validar() != nil {
		t.Fatal("Datos altero el manifiesto opaco")
	}
	datosRestaurados, _ := escenario.manifiesto.Datos()
	restaurado := ManifiestoEjecucionDocumentalV3{datos: &datosRestaurados}
	if restaurado.datos == escenario.manifiesto.datos ||
		!manifiestosEjecucionDocumentalV3Coinciden(restaurado, escenario.manifiesto) ||
		huella != datosRestaurados.HuellaPlanSHA256 {
		t.Fatal("la igualdad semantica dependio de identidad de puntero")
	}

	preparacion := PreparacionEjecucionDocumentalV3Nominal{
		ReservaRef: "reserva:documental:v3:001", BorradorRef: datosRestaurados.BorradorRef,
		EfectoRef: datosRestaurados.EfectoRef, Estado: EstadoEjecucionDocumentalV3Preparada,
	}
	if err := preparacion.ValidarContra(escenario.preparar); err != nil {
		t.Fatalf("preparacion exacta rechazada: %v", err)
	}
	cruzada := preparacion
	cruzada.EfectoRef = "efecto:documental:v3:otro"
	if cruzada.ValidarContra(escenario.preparar) == nil {
		t.Fatal("preparacion acepto otra referencia de efecto")
	}

	base, _ := escenario.manifiesto.Datos()
	if _, err := NuevoManifiestoEjecucionDocumentalV3(
		base.Consulta, base.DescriptorPerfil, base.SituacionOperativa,
		base.ComponenteRender, base.ComponenteVerificador, base.ComponenteVerificador,
		base.BorradorRef, base.EfectoRef, base.HuellaEntradaHMAC, base.LimiteEfectivoBytes,
	); !errors.Is(err, ErrManifiestoEjecucionDocumentalV3Invalido) {
		t.Fatalf("componente semantico no independiente aceptado: %v", err)
	}
	if _, err := NuevoManifiestoEjecucionDocumentalV3(
		base.Consulta, base.DescriptorPerfil, base.SituacionOperativa,
		base.ComponenteRender, base.ComponenteVerificador, base.ComponenteSemantico,
		"dni:12345678z", base.EfectoRef, base.HuellaEntradaHMAC, base.LimiteEfectivoBytes,
	); !errors.Is(err, ErrManifiestoEjecucionDocumentalV3Invalido) {
		t.Fatalf("PII evidente aceptada como referencia: %v", err)
	}
	if _, err := NuevoManifiestoEjecucionDocumentalV3(
		base.Consulta, base.DescriptorPerfil, base.SituacionOperativa,
		base.ComponenteRender, base.ComponenteVerificador, base.ComponenteSemantico,
		base.BorradorRef, base.EfectoRef, base.HuellaEntradaHMAC,
		base.ComponenteSemantico.MaximoBytes()+1,
	); !errors.Is(err, ErrManifiestoEjecucionDocumentalV3Invalido) {
		t.Fatalf("techo semantico excedido: %v", err)
	}
}

func TestActivacionEjecucionDocumentalV3ExigeOrdenDurableV4Exacta(t *testing.T) {
	escenario := nuevoEscenarioEjecucionDocumentalV3Prueba(t)
	solicitud := SolicitudActivarEjecucionDocumentalV3{
		ReservaRef:               escenario.vinculoActivacion.ReservaRef,
		IndiceIdempotenciaHMAC:   escenario.vinculoActivacion.IndiceIdempotenciaHMAC,
		HuellaSolicitudHMAC:      escenario.vinculoActivacion.HuellaSolicitudHMAC,
		Manifiesto:               escenario.vinculoActivacion.Manifiesto,
		ConsumoDecision:          escenario.vinculoActivacion.ConsumoDecision,
		OrdenConsumoDurableV4Ref: escenario.vinculoActivacion.OrdenConsumoDurableV4Ref,
		ActivadaEn:               escenario.instante.Add(time.Second),
	}
	if err := solicitud.Validar(); err != nil {
		t.Fatalf("orden V4 exacta rechazada: %v", err)
	}
	activacion := ActivacionEjecucionDocumentalV3Nominal{Token: escenario.token}
	if err := activacion.ValidarContra(solicitud); err != nil {
		t.Fatalf("activacion exacta rechazada: %v", err)
	}
	reintento := solicitud
	reintento.ActivadaEn = solicitud.ActivadaEn.Add(time.Minute)
	if err := reintento.Validar(); err != nil {
		t.Fatalf("reintento estable con otro instante rechazado: %v", err)
	}
	vinculoOriginal, _ := solicitud.VinculoEstable()
	vinculoReintento, _ := reintento.VinculoEstable()
	if _, existe := reflect.TypeOf(vinculoOriginal).FieldByName("ActivadaEn"); existe {
		t.Fatal("el instante efimero entro en el vinculo estable")
	}
	huellaOriginal, _ := vinculoOriginal.HuellaSHA256()
	huellaReintento, _ := vinculoReintento.HuellaSHA256()
	if huellaOriginal != huellaReintento || escenario.token.ValidarPara(vinculoReintento) != nil {
		t.Fatal("ActivadaEn altero el vinculo estable o el token recuperable")
	}
	activacion.Repetida = true
	if err := activacion.ValidarContra(reintento); err != nil {
		t.Fatalf("el instante efimero impidio recuperar el mismo token: %v", err)
	}

	datos, err := escenario.manifiesto.Datos()
	if err != nil {
		t.Fatal(err)
	}
	manifiestoOtroPlan, err := NuevoManifiestoEjecucionDocumentalV3(
		datos.Consulta, datos.DescriptorPerfil, datos.SituacionOperativa,
		datos.ComponenteRender, datos.ComponenteVerificador, datos.ComponenteSemantico,
		"borrador:documental:v3:otro", datos.EfectoRef, datos.HuellaEntradaHMAC,
		datos.LimiteEfectivoBytes,
	)
	if err != nil {
		t.Fatal(err)
	}
	datosOtroPlan, _ := manifiestoOtroPlan.Datos()
	manifiestoOtroEfecto, err := NuevoManifiestoEjecucionDocumentalV3(
		datos.Consulta, datos.DescriptorPerfil, datos.SituacionOperativa,
		datos.ComponenteRender, datos.ComponenteVerificador, datos.ComponenteSemantico,
		datos.BorradorRef, "efecto:documental:v3:otro", datos.HuellaEntradaHMAC,
		datos.LimiteEfectivoBytes,
	)
	if err != nil {
		t.Fatal(err)
	}
	datosOtroEfecto, _ := manifiestoOtroEfecto.Datos()

	crucesValidos := []struct {
		nombre string
		mutar  func(*VinculoEstableActivacionDocumentalV3)
	}{
		{"reserva", func(v *VinculoEstableActivacionDocumentalV3) {
			v.ReservaRef = "reserva:documental:v3:otra"
		}},
		{"indice idempotencia HMAC", func(v *VinculoEstableActivacionDocumentalV3) {
			v.IndiceIdempotenciaHMAC = "hmac-sha256:indice-v3-otro:" + strings.Repeat("1", 64)
		}},
		{"huella solicitud HMAC", func(v *VinculoEstableActivacionDocumentalV3) {
			v.HuellaSolicitudHMAC = "hmac-sha256:solicitud-v3-otra:" + strings.Repeat("2", 64)
		}},
		{"manifiesto y plan", func(v *VinculoEstableActivacionDocumentalV3) {
			v.Manifiesto = manifiestoOtroPlan
			v.ConsumoDecision.HuellaPlanSHA256 = datosOtroPlan.HuellaPlanSHA256
		}},
		{"orden y efecto", func(v *VinculoEstableActivacionDocumentalV3) {
			v.Manifiesto = manifiestoOtroEfecto
			v.ConsumoDecision.EfectoRef = datosOtroEfecto.EfectoRef
			v.ConsumoDecision.HuellaPlanSHA256 = datosOtroEfecto.HuellaPlanSHA256
			v.OrdenConsumoDurableV4Ref = datosOtroEfecto.EfectoRef
		}},
		{"decision", func(v *VinculoEstableActivacionDocumentalV3) {
			v.ConsumoDecision.DecisionRef = "decision:documental:v3:otra"
		}},
		{"huella decision", func(v *VinculoEstableActivacionDocumentalV3) {
			v.ConsumoDecision.HuellaDecisionSHA256 = strings.Repeat("f", 64)
		}},
	}
	for _, caso := range crucesValidos {
		t.Run(caso.nombre, func(t *testing.T) {
			cruzada := escenario.vinculoActivacion
			caso.mutar(&cruzada)
			if cruzada.Validar() != nil {
				t.Fatal("el cruce de prueba debe ser otra intencion estructuralmente valida")
			}
			if escenario.token.ValidarPara(cruzada) == nil {
				t.Fatal("el token acepto una intencion estable distinta")
			}
			solicitudCruzada := SolicitudActivarEjecucionDocumentalV3{
				ReservaRef: cruzada.ReservaRef, IndiceIdempotenciaHMAC: cruzada.IndiceIdempotenciaHMAC,
				HuellaSolicitudHMAC: cruzada.HuellaSolicitudHMAC, Manifiesto: cruzada.Manifiesto,
				ConsumoDecision:          cruzada.ConsumoDecision,
				OrdenConsumoDurableV4Ref: cruzada.OrdenConsumoDurableV4Ref,
				ActivadaEn:               solicitud.ActivadaEn,
			}
			if solicitudCruzada.Validar() != nil || activacion.ValidarContra(solicitudCruzada) == nil {
				t.Fatal("la activacion acepto otra intencion estable valida")
			}
		})
	}
	for _, caso := range []struct {
		nombre string
		mutar  func(*SolicitudActivarEjecucionDocumentalV3)
	}{
		{"orden cero", func(s *SolicitudActivarEjecucionDocumentalV3) { s.OrdenConsumoDurableV4Ref = "" }},
		{"orden discordante", func(s *SolicitudActivarEjecucionDocumentalV3) {
			s.OrdenConsumoDurableV4Ref = "efecto:documental:v4:otro"
		}},
		{"orden con PII", func(s *SolicitudActivarEjecucionDocumentalV3) {
			s.OrdenConsumoDurableV4Ref = "dni:12345678z"
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			invalida := solicitud
			caso.mutar(&invalida)
			if invalida.Validar() == nil || activacion.ValidarContra(invalida) == nil {
				t.Fatal("una orden V4 invalida habilito la activacion")
			}
		})
	}
	if (SolicitudActivarEjecucionDocumentalV3{}).Validar() == nil {
		t.Fatal("el valor cero habilito una activacion")
	}
}

func TestSolicitudActivarEjecucionDocumentalV3SeRedactaYNoSeSerializa(t *testing.T) {
	escenario := nuevoEscenarioEjecucionDocumentalV3Prueba(t)
	datos, _ := escenario.manifiesto.Datos()
	solicitud := SolicitudActivarEjecucionDocumentalV3{
		ReservaRef:               "reserva:documental:v3:001",
		IndiceIdempotenciaHMAC:   escenario.preparar.IndiceIdempotenciaHMAC,
		HuellaSolicitudHMAC:      escenario.preparar.HuellaSolicitudHMAC,
		Manifiesto:               escenario.manifiesto,
		ConsumoDecision:          escenario.consumo,
		OrdenConsumoDurableV4Ref: datos.EfectoRef,
		ActivadaEn:               escenario.instante.Add(time.Second),
	}
	texto := fmt.Sprintf("%v|%+v|%#v", solicitud, solicitud, solicitud)
	vinculo, err := solicitud.VinculoEstable()
	if err != nil {
		t.Fatal(err)
	}
	texto += fmt.Sprintf("|%v|%+v|%#v", vinculo, vinculo, vinculo)
	for _, secreto := range []string{
		solicitud.ReservaRef, solicitud.OrdenConsumoDurableV4Ref,
		solicitud.IndiceIdempotenciaHMAC, solicitud.ConsumoDecision.DecisionRef,
	} {
		if strings.Contains(texto, secreto) {
			t.Fatalf("formato filtro material interno: %q", secreto)
		}
	}
	for nombre, valor := range map[string]any{"solicitud": solicitud, "vinculo": vinculo} {
		resuelto := slog.Any("valor", valor).Value.Resolve()
		if resuelto.Kind() != slog.KindString || strings.Contains(resuelto.String(), solicitud.ReservaRef) {
			t.Fatalf("slog filtro %s: %v", nombre, resuelto)
		}
	}
	if _, err := json.Marshal(solicitud); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("MarshalJSON solicitud: %v", err)
	}
	if _, err := solicitud.MarshalText(); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("MarshalText solicitud: %v", err)
	}
	if _, err := solicitud.MarshalBinary(); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("MarshalBinary solicitud: %v", err)
	}
	var restaurada SolicitudActivarEjecucionDocumentalV3
	if err := json.Unmarshal([]byte(`{}`), &restaurada); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("UnmarshalJSON solicitud: %v", err)
	}
	if err := restaurada.UnmarshalText([]byte("activacion")); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("UnmarshalText solicitud: %v", err)
	}
	if err := restaurada.UnmarshalBinary([]byte("activacion")); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("UnmarshalBinary solicitud: %v", err)
	}
	if _, err := json.Marshal(vinculo); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("MarshalJSON vinculo: %v", err)
	}
	if _, err := vinculo.MarshalText(); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("MarshalText vinculo: %v", err)
	}
}

func TestTokenCercadoV3ExigeVerificacionYNoSeSerializa(t *testing.T) {
	escenario := nuevoEscenarioEjecucionDocumentalV3Prueba(t)
	texto := escenario.token.String() + escenario.token.GoString()
	if strings.Contains(texto, "token:cercado:v3:001") || strings.Contains(texto, "aaaa") {
		t.Fatal("token filtrado por formato")
	}
	if _, err := json.Marshal(escenario.token); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("MarshalJSON token: %v", err)
	}
	if _, err := escenario.token.MarshalText(); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("MarshalText token: %v", err)
	}
	var restaurado TokenCercadoEjecucionDocumentalV3Nominal
	if err := json.Unmarshal([]byte(`{}`), &restaurado); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("UnmarshalJSON token: %v", err)
	}
	if err := restaurado.UnmarshalText([]byte("token")); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("UnmarshalText token: %v", err)
	}

	macEntrada := []byte(strings.Repeat("z", 32))
	tokenCopiado, err := NuevoTokenCercadoEjecucionDocumentalV3Nominal(
		"token:cercado:v3:copia", 42, escenario.vinculoActivacion,
		"clave:cercado:v3", 1, macEntrada, "evidencia:cercado:v3:copia",
	)
	if err != nil {
		t.Fatal(err)
	}
	macEntrada[0] ^= 0xff
	if tokenCopiado.macAtestacion[0] != 'z' {
		t.Fatal("el token retuvo un alias del MAC de entrada")
	}
	solicitudCopia, err := NuevaSolicitudVerificacionTokenCercadoDocumentalV3(
		escenario.vinculoActivacion, tokenCopiado,
	)
	if err != nil {
		t.Fatal(err)
	}
	mensajeCopia, _ := solicitudCopia.Mensaje()
	macCopia, _ := solicitudCopia.MAC()
	mensajeCopia[0] ^= 0xff
	macCopia[0] ^= 0xff
	if solicitudCopia.Validar() != nil || tokenCopiado.macAtestacion[0] != 'z' {
		t.Fatal("los accesores del verificador expusieron memoria interna")
	}
	huellaVinculoEstable, _ := escenario.vinculoActivacion.HuellaSHA256()
	mensajeAtestacion, _ := solicitudCopia.Mensaje()
	if !strings.Contains(string(mensajeAtestacion), huellaVinculoEstable) {
		t.Fatal("el mensaje MAC no comprometio la huella canonica del vinculo estable")
	}

	inicio := SolicitudIniciarEfectoDocumentalV3{
		VinculoActivacion: escenario.vinculoActivacion, Token: escenario.token,
		IniciadoEn: escenario.instante.Add(time.Second),
	}
	if err := inicio.Validar(); err != nil {
		t.Fatalf("inicio cercado valido: %v", err)
	}
	tipoInicio := reflect.TypeOf(inicio)
	for _, campo := range []string{"VerificacionCercado", "ComprobadoTCB", "MetadatosComprobacion"} {
		if _, existe := tipoInicio.FieldByName(campo); existe {
			t.Fatalf("SolicitudIniciar conserva autoridad autocertificable en %s", campo)
		}
	}
	sinToken := inicio
	sinToken.Token = TokenCercadoEjecucionDocumentalV3Nominal{}
	if sinToken.Validar() == nil {
		t.Fatal("un inicio sin token estructural exacto fue aceptado")
	}

	alterado := escenario.token
	alterado.macAtestacion = append([]byte(nil), escenario.token.macAtestacion...)
	alterado.macAtestacion[0] ^= 0xff
	if alterado.ValidarPara(inicio.VinculoActivacion) != nil {
		t.Fatal("la prueba necesita un sobre estructuralmente valido")
	}
	solicitudAlterada, err := NuevaSolicitudVerificacionTokenCercadoDocumentalV3(
		inicio.VinculoActivacion, alterado,
	)
	if err != nil {
		t.Fatal(err)
	}
	if escenario.verificacionCercado.ValidarPara(solicitudAlterada) == nil {
		t.Fatal("una verificacion anterior autorizo otro MAC")
	}

	vinculoCruzado := escenario.vinculoActivacion
	vinculoCruzado.ConsumoDecision.DecisionRef = "decision:documental:v3:otra"
	if vinculoCruzado.Validar() != nil || escenario.token.ValidarPara(vinculoCruzado) == nil {
		t.Fatal("token aceptado para otra DecisionRef")
	}
}

func TestEstadosEjecucionDocumentalV3SonCerrados(t *testing.T) {
	escenario := nuevoEscenarioEjecucionDocumentalV3Prueba(t)
	for _, caso := range []struct {
		desde, hasta EstadoEjecucionDocumentalV3
		admite       bool
	}{
		{EstadoEjecucionDocumentalV3Preparada, EstadoEjecucionDocumentalV3Activa, true},
		{EstadoEjecucionDocumentalV3Activa, EstadoEjecucionDocumentalV3EfectoIniciado, true},
		{EstadoEjecucionDocumentalV3EfectoIniciado, EstadoEjecucionDocumentalV3Indeterminada, true},
		{EstadoEjecucionDocumentalV3Indeterminada, EstadoEjecucionDocumentalV3Confirmada, true},
		{EstadoEjecucionDocumentalV3Confirmada, EstadoEjecucionDocumentalV3Activa, false},
		{"futuro", EstadoEjecucionDocumentalV3Confirmada, false},
	} {
		if caso.desde.PuedeTransicionarA(caso.hasta) != caso.admite {
			t.Fatalf("transicion %s -> %s", caso.desde, caso.hasta)
		}
	}
	base := InstantaneaEjecucionDocumentalV3Nominal{
		ReservaRef:             "reserva:documental:v3:001",
		IndiceIdempotenciaHMAC: escenario.preparar.IndiceIdempotenciaHMAC,
		HuellaSolicitudHMAC:    escenario.preparar.HuellaSolicitudHMAC,
		Manifiesto:             escenario.manifiesto, ActualizadaEn: escenario.instante,
	}
	preparada := base
	preparada.Estado = EstadoEjecucionDocumentalV3Preparada
	if preparada.Validar() != nil {
		t.Fatal("instantanea preparada valida rechazada")
	}
	abandonada := base
	abandonada.Estado = EstadoEjecucionDocumentalV3AbandonadaSinEfecto
	abandonada.EstadoOrigenAbandono = EstadoEjecucionDocumentalV3Preparada
	abandonada.MotivoAbandonoRef = "motivo:sin:efecto"
	if abandonada.Validar() != nil {
		t.Fatal("abandono desde preparada rechazado")
	}
	abandonada.SecuenciaCercado = 99
	if abandonada.Validar() == nil {
		t.Fatal("abandono acepto cercado arbitrario")
	}

	solicitudAbandonoPreparada := SolicitudAbandonarEjecucionDocumentalV3{
		ReservaRef: "reserva:documental:v3:001", Manifiesto: escenario.manifiesto,
		EstadoEsperado: EstadoEjecucionDocumentalV3Preparada,
		MotivoRef:      "motivo:sin:efecto", AbandonadaEn: escenario.instante.Add(time.Second),
	}
	if solicitudAbandonoPreparada.Validar() != nil {
		t.Fatal("abandono desde preparada valido rechazado")
	}
	solicitudAbandonoActiva := solicitudAbandonoPreparada
	solicitudAbandonoActiva.EstadoEsperado = EstadoEjecucionDocumentalV3Activa
	solicitudAbandonoActiva.ConsumoDecision = escenario.consumo
	if solicitudAbandonoActiva.Validar() != nil {
		t.Fatal("abandono desde activa valido rechazado")
	}
	consumoCruzado := solicitudAbandonoActiva
	consumoCruzado.ConsumoDecision.EfectoRef = "efecto:documental:v3:otro"
	if consumoCruzado.Validar() == nil {
		t.Fatal("abandono desde activa acepto un consumo cruzado")
	}
	abandonadaActiva := base
	abandonadaActiva.Estado = EstadoEjecucionDocumentalV3AbandonadaSinEfecto
	abandonadaActiva.EstadoOrigenAbandono = EstadoEjecucionDocumentalV3Activa
	abandonadaActiva.MotivoAbandonoRef = "motivo:sin:efecto"
	abandonadaActiva.SecuenciaCercado = escenario.token.Secuencia()
	abandonadaActiva.HuellaVinculoSHA256 = escenario.token.HuellaVinculoSHA256()
	abandonadaActiva.ConsumoDecision = escenario.consumo
	abandonadaActiva.OrdenConsumoDurableV4Ref = escenario.vinculoActivacion.OrdenConsumoDurableV4Ref
	if abandonadaActiva.Validar() != nil {
		t.Fatal("instantanea de abandono desde activa rechazada")
	}

	indeterminada := base
	indeterminada.Estado = EstadoEjecucionDocumentalV3Indeterminada
	indeterminada.SecuenciaCercado = escenario.token.Secuencia()
	indeterminada.HuellaVinculoSHA256 = escenario.token.HuellaVinculoSHA256()
	indeterminada.ConsumoDecision = escenario.consumo
	indeterminada.OrdenConsumoDurableV4Ref = escenario.vinculoActivacion.OrdenConsumoDurableV4Ref
	indeterminada.IncidenteRef = "incidente:documental:v3:001"
	if indeterminada.Validar() != nil {
		t.Fatal("instantanea indeterminada valida rechazada")
	}
	indeterminada.Resultado = escenario.resultado
	if indeterminada.Validar() == nil {
		t.Fatal("indeterminada acepto resultado asumido")
	}

	abandonadaIndeterminada := indeterminada
	abandonadaIndeterminada.Resultado = ResultadoEfectoRenderizadoDocumentalV3Crudo{}
	abandonadaIndeterminada.Estado = EstadoEjecucionDocumentalV3AbandonadaSinEfecto
	abandonadaIndeterminada.EstadoOrigenAbandono = EstadoEjecucionDocumentalV3Indeterminada
	abandonadaIndeterminada.MotivoAbandonoRef = "motivo:reconciliado:sin:efecto"
	abandonadaIndeterminada.ReconciliacionRef = "atestacion:reconciliacion:v3:sin:efecto"
	abandonadaIndeterminada.HuellaReconciliacionSHA256 = strings.Repeat("9", 64)
	if abandonadaIndeterminada.Validar() != nil {
		t.Fatal("abandono reconciliado no conservo incidente y evidencia negativa")
	}

	datosSello, _ := escenario.sello.Datos()
	confirmada := base
	confirmada.Estado = EstadoEjecucionDocumentalV3Confirmada
	confirmada.SecuenciaCercado = escenario.token.Secuencia()
	confirmada.HuellaVinculoSHA256 = escenario.token.HuellaVinculoSHA256()
	confirmada.ConsumoDecision = escenario.consumo
	confirmada.OrdenConsumoDurableV4Ref = escenario.vinculoActivacion.OrdenConsumoDurableV4Ref
	confirmada.Resultado = escenario.resultado
	confirmada.EvidenciaRef = datosSello.EvidenciaOperacionRef
	confirmada.HuellaEvidenciaSHA256 = datosSello.HuellaMensajeSHA256
	confirmada.ReconciliacionRef = "atestacion:reconciliacion:v3:001"
	confirmada.HuellaReconciliacionSHA256 = strings.Repeat("8", 64)
	if confirmada.Validar() != nil {
		t.Fatal("instantanea confirmada por reconciliacion exacta rechazada")
	}
	confirmada.HuellaReconciliacionSHA256 = ""
	if confirmada.Validar() == nil {
		t.Fatal("instantanea confirmada acepto reconciliacion parcial")
	}
}

func TestConfirmacionV3LigaTresRecibosCercadoYSelloRestaurable(t *testing.T) {
	escenario := nuevoEscenarioEjecucionDocumentalV3Prueba(t)
	if err := escenario.confirmacion.Validar(); err != nil {
		t.Fatalf("confirmacion valida rechazada: %v", err)
	}
	datos, _ := escenario.manifiesto.Datos()
	manifiestoRestaurado := ManifiestoEjecucionDocumentalV3{datos: &datos}
	restaurada := escenario.confirmacion
	restaurada.Manifiesto = manifiestoRestaurado
	restaurada.Evidencia.Manifiesto = manifiestoRestaurado
	if err := restaurada.Validar(); err != nil {
		t.Fatalf("confirmacion restaurada por valor rechazada: %v", err)
	}

	cruzada := escenario.confirmacion
	cruzada.Recibos.Semantico = escenario.recibos.Estructural
	if cruzada.Validar() == nil {
		t.Fatal("confirmacion acepto recibo semantico cruzado")
	}
	evidenciaAlterada := escenario.confirmacion
	evidenciaAlterada.Evidencia.Recibos.HuellaReciboSemanticoSHA256 = strings.Repeat("e", 64)
	if evidenciaAlterada.Validar() == nil {
		t.Fatal("confirmacion acepto huella de recibo manipulada")
	}

	datosSello, err := escenario.sello.Datos()
	if err != nil {
		t.Fatal(err)
	}
	original := append([]byte(nil), datosSello.Firma...)
	datosSello.Firma[0] ^= 0xff
	datosOtraLectura, _ := escenario.sello.Datos()
	if string(datosOtraLectura.Firma) != string(original) {
		t.Fatal("Datos expuso alias de la firma")
	}
	perfil, _ := NuevoPerfilSelloEvidenciaHMACSHA256V3("clave:evidencia:v3")
	firma, _ := NuevaSolicitudFirmaEvidenciaRenderizadoDocumentalV3(perfil, escenario.evidencia)
	solicitudManipulada, err := NuevaSolicitudVerificacionEvidenciaDocumentalV3DesdeDatos(
		firma, datosSello,
	)
	if err != nil {
		t.Fatal(err)
	}
	if escenario.verificacion.ValidarPara(solicitudManipulada) == nil {
		t.Fatal("resultado de verificacion reutilizado tras manipular la firma")
	}
	var selloCero SelloEvidenciaDocumentalV3Nominal
	if err := json.Unmarshal([]byte(`{}`), &selloCero); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("UnmarshalJSON sello: %v", err)
	}
	if err := selloCero.UnmarshalText([]byte("sello")); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("UnmarshalText sello: %v", err)
	}
	if _, err := json.Marshal(escenario.sello); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("MarshalJSON sello: %v", err)
	}
	if _, err := escenario.sello.MarshalText(); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("MarshalText sello: %v", err)
	}

	cronologia := escenario.evidencia
	cronologia.VerificadoCercadoEn = cronologia.GeneradoEn.Add(time.Second)
	if cronologia.Validar() == nil {
		t.Fatal("evidencia acepto cercado verificado despues de generar")
	}
}

func TestReconciliacionV3ExigeCOSEVerificadoYLigaConfirmacion(t *testing.T) {
	escenario := nuevoEscenarioEjecucionDocumentalV3Prueba(t)
	consulta := SolicitudConsultarEfectoDocumentalV3{
		ReservaRef: "reserva:documental:v3:001", Manifiesto: escenario.manifiesto,
		ConsumoDecision:        escenario.consumo,
		OrdenDespachoConsumida: escenario.ordenDespachoConsumida,
		SolicitadaEn:           escenario.instante.Add(4 * time.Second),
	}
	bytesSobre := []byte("cose-reconciliacion-aplicada-v3")
	sobre, err := NuevoSobreAtestacionReconciliacionDocumentalV3Crudo(bytesSobre)
	if err != nil {
		t.Fatal(err)
	}
	bytesSobre[0] ^= 0xff
	coseExpuesto, _ := sobre.COSESign1()
	coseExpuesto[0] ^= 0xff
	coseOtraLectura, _ := sobre.COSESign1()
	if string(coseOtraLectura) != "cose-reconciliacion-aplicada-v3" {
		t.Fatal("el sobre COSE no realizo copias defensivas")
	}
	if _, err := json.Marshal(sobre); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("MarshalJSON sobre COSE: %v", err)
	}
	huellaSobre, _ := sobre.HuellaSHA256()
	resultado := ResultadoConsultaEfectoDocumentalV3Crudo{
		ReservaRef: consulta.ReservaRef, EfectoRef: escenario.resultado.EfectoRef,
		SecuenciaCercado:    escenario.token.Secuencia(),
		HuellaVinculoSHA256: escenario.token.HuellaVinculoSHA256(),
		HuellaPlanSHA256:    escenario.consumo.HuellaPlanSHA256,
		Estado:              ResultadoReconciliacionDocumentalV3AplicadoExacto,
		Resultado:           escenario.resultado, AtestacionRef: "atestacion:reconciliacion:v3:001",
		HuellaAtestacionSHA256: huellaSobre, SobreAtestacion: sobre,
		ConsultadaEn: escenario.instante.Add(5 * time.Second),
	}
	textoResultado := fmt.Sprintf("%v|%+v|%#v", resultado, resultado, resultado)
	for _, secreto := range []string{
		resultado.ReservaRef, resultado.AtestacionRef, resultado.Resultado.ContenidoRef,
		"cose-reconciliacion-aplicada-v3",
	} {
		if strings.Contains(textoResultado, secreto) {
			t.Fatalf("el resultado de reconciliacion filtro referencia o COSE %q", secreto)
		}
	}
	if slog.Any("resultado", resultado).Value.Resolve().Kind() != slog.KindString {
		t.Fatal("el resultado de reconciliacion no se redacto en slog")
	}
	if _, err := json.Marshal(resultado); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("MarshalJSON resultado reconciliacion: %v", err)
	}
	if _, err := resultado.MarshalText(); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("MarshalText resultado reconciliacion: %v", err)
	}
	if _, err := resultado.MarshalBinary(); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("MarshalBinary resultado reconciliacion: %v", err)
	}
	solicitudVerificacion, err := NuevaSolicitudVerificacionReconciliacionDocumentalV3(consulta, resultado)
	if err != nil {
		t.Fatal(err)
	}
	verificacion, err := NuevosMetadatosComprobacionReconciliacionDocumentalV3Nominal(
		solicitudVerificacion, "verificacion:reconciliacion:v3:001",
		escenario.instante.Add(6*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion := confirmarReconciliacionEjecucionDocumentalV3Prueba(
		t, escenario, resultado, verificacion,
	)
	aplicar := SolicitudAplicarReconciliacionDocumentalV3{
		Consulta: consulta, ResultadoConsulta: resultado,
		TieneConfirmacion: true, Confirmacion: confirmacion,
	}
	if err := aplicar.Validar(); err != nil {
		t.Fatalf("indeterminada -> aplicada -> confirmada rechazada: %v", err)
	}
	tipoAplicar := reflect.TypeOf(aplicar)
	if _, existe := tipoAplicar.FieldByName("Verificacion"); existe {
		t.Fatal("la solicitud acepto metadatos publicos autocertificables como autoridad")
	}
	manipulada := resultado
	manipulada.SobreAtestacion.coseSign1 = append([]byte(nil), sobre.coseSign1...)
	manipulada.SobreAtestacion.coseSign1[0] ^= 0xff
	manipulada.SobreAtestacion.huella = huellaBytesFormatoDocumental(manipulada.SobreAtestacion.coseSign1)
	manipulada.HuellaAtestacionSHA256 = manipulada.SobreAtestacion.huella
	solicitudManipulada, err := NuevaSolicitudVerificacionReconciliacionDocumentalV3(consulta, manipulada)
	if err != nil {
		t.Fatal(err)
	}
	if verificacion.ValidarPara(solicitudManipulada) == nil {
		t.Fatal("verificacion de reconciliacion reutilizada con otro sobre")
	}
	cronologia := confirmacion.Evidencia
	cronologia.ReconciliacionConsultadaEn = cronologia.GeneradoEn.Add(-time.Second)
	if cronologia.Validar() == nil {
		t.Fatal("evidencia acepto una reconciliacion anterior a su generacion")
	}
}

func nuevoEscenarioEjecucionDocumentalV3Prueba(t *testing.T) escenarioEjecucionDocumentalV3Prueba {
	t.Helper()
	instante := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	identidad, _ := domain.NuevaIdentidadSintacticaDocumental("pdf")
	perfilRef, _ := domain.NuevaReferenciaPerfilDocumental("pdfa-4", 3)
	capacidades, _ := domain.NuevasCapacidadesPerfilFormatoDocumental(
		domain.CapacidadPerfilRenderizar, domain.CapacidadPerfilMetadatoInstitucional,
	)
	conformidad, err := domain.NuevaReferenciaConformidadDocumental(
		"conformidad:pdfa4:v3", 1, "esquema:pdfa4:v3", "dialecto:pdfa4:v3",
		"canonicalizacion:pdf:v3", "reglas:pdfa4:v3", strings.Repeat("1", 64),
		"politica:documental:v3", strings.Repeat("2", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	perfil, err := domain.NuevoPerfilFormatoDocumentalConforme(
		perfilRef, identidad, "application/pdf", "pdf", "binario",
		capacidades, conformidad, 4*1024*1024,
	)
	if err != nil {
		t.Fatal(err)
	}
	revision, _ := domain.NuevaRevisionCatalogoFormatosDocumentales(21, strings.Repeat("3", 64))
	consulta := ConsultaFormatoDocumental{
		Identidad: identidad, PerfilRef: perfilRef, DigestPerfilSHA256: perfil.DigestSHA256(),
		RevisionCatalogo: revision,
	}
	descriptor, err := NuevoDescriptorPerfilDocumental(
		"descriptor:pdfa4:v3", "publicacion:pdfa4:v3", perfil, revision,
	)
	if err != nil {
		t.Fatal(err)
	}
	situacion, _ := domain.NuevaSituacionOperativaPerfilDocumental(
		descriptor.PublicacionRef(), perfil, revision, 7, domain.EstadoPublicacionPerfilVigente,
	)
	render := descriptorComponenteEjecucionDocumentalV3Prueba(
		t, descriptor, domain.RolComponenteRenderizador, "motor:render:v3", "dominio:render:v3", '4', 2*1024*1024,
	)
	estructural := descriptorComponenteEjecucionDocumentalV3Prueba(
		t, descriptor, domain.RolComponenteValidadorEstructural, "motor:estructural:v3", "dominio:estructural:v3", '7', 2*1024*1024,
	)
	semantico := descriptorComponenteEjecucionDocumentalV3Prueba(
		t, descriptor, domain.RolComponenteVerificadorSemantico, "motor:semantico:v3", "dominio:semantico:v3", 'a', 1024*1024,
	)
	entrada := "hmac-sha256:entrada-v3:" + strings.Repeat("a", 64)
	manifiesto, err := NuevoManifiestoEjecucionDocumentalV3(
		consulta, descriptor, situacion, render, estructural, semantico,
		"borrador:documental:v3:001", "efecto:documental:v3:001", entrada, 1024*1024,
	)
	if err != nil {
		t.Fatal(err)
	}
	preparar := SolicitudPrepararEjecucionDocumentalV3{
		IndiceIdempotenciaHMAC: "hmac-sha256:indice-v3:" + strings.Repeat("b", 64),
		HuellaSolicitudHMAC:    "hmac-sha256:solicitud-v3:" + strings.Repeat("c", 64),
		Manifiesto:             manifiesto, SolicitadaEn: instante, ExpiraEn: instante.Add(10 * time.Minute),
	}
	datosManifiesto, _ := manifiesto.Datos()
	consumo := ConsumoDecisionEjecucionDocumentalV3{
		DecisionRef: "decision:documental:v3:001", EfectoRef: datosManifiesto.EfectoRef,
		EsquemaHuellaDecision: EsquemaHuellaDecisionAutorizacionReforzadaV1,
		HuellaDecisionSHA256:  strings.Repeat("d", 64), HuellaPlanSHA256: datosManifiesto.HuellaPlanSHA256,
	}
	activar := SolicitudActivarEjecucionDocumentalV3{
		ReservaRef:             "reserva:documental:v3:001",
		IndiceIdempotenciaHMAC: preparar.IndiceIdempotenciaHMAC,
		HuellaSolicitudHMAC:    preparar.HuellaSolicitudHMAC,
		Manifiesto:             manifiesto, ConsumoDecision: consumo,
		OrdenConsumoDurableV4Ref: datosManifiesto.EfectoRef,
		ActivadaEn:               instante.Add(250 * time.Millisecond),
	}
	vinculoActivacion, err := activar.VinculoEstable()
	if err != nil {
		t.Fatal(err)
	}
	token := nuevoTokenCercadoEjecucionDocumentalV3Prueba(
		t, "token:cercado:v3:001", 41, vinculoActivacion,
		"clave:cercado:v3", 1, "evidencia:cercado:v3:001",
	)
	solicitudCercado, _ := NuevaSolicitudVerificacionTokenCercadoDocumentalV3(
		vinculoActivacion, token,
	)
	verificacionCercado, err := NuevosMetadatosComprobacionTokenCercadoDocumentalV3Nominal(
		solicitudCercado, "verificacion:cercado:v3:001", instante.Add(500*time.Millisecond),
	)
	if err != nil {
		t.Fatal(err)
	}
	ordenDespachoConsumida := ordenDespachoDocumentalV3ConsumidaNominalPrueba(
		t, vinculoActivacion, token, instante.Add(2*time.Second),
		instante.Add(9*time.Minute), "escenario-v3",
	)
	resultado := ResultadoEfectoRenderizadoDocumentalV3Crudo{
		BorradorRef: datosManifiesto.BorradorRef, EfectoRef: datosManifiesto.EfectoRef,
		ContenidoRef: "objeto:documental:v3:001", ContenidoVersion: "version:v3:001",
		ConectorRef: "conector:almacen:v3", MIME: perfil.MIME(),
		HuellaSalidaSHA256: strings.Repeat("e", 64), TamanoSalida: 2048,
		EvidenciaOperacionRef: "evidencia:almacen:v3:001",
	}
	recibos := RecibosEjecucionDocumentalV3{
		Render: reciboEjecucionDocumentalV3Prueba(
			t, "render", OperacionRenderizadoDocumental, render, descriptor, situacion,
			ordenDespachoConsumida, resultado, instante, 1,
		),
		Estructural: reciboEjecucionDocumentalV3Prueba(
			t, "estructural", OperacionValidacionEstructuralDocumental, estructural,
			descriptor, situacion, ordenDespachoConsumida, resultado, instante, 2,
		),
		Semantico: reciboEjecucionDocumentalV3Prueba(
			t, "semantico", OperacionVerificacionSemanticaDocumental, semantico,
			descriptor, situacion, ordenDespachoConsumida, resultado, instante, 3,
		),
	}
	huellasRecibos, err := recibos.Huellas()
	if err != nil {
		t.Fatal(err)
	}
	evidencia := DatosEvidenciaRenderizadoDocumentalV3{
		Esquema: EsquemaEvidenciaRenderizadoV3, ReservaRef: "reserva:documental:v3:001",
		IndiceIdempotenciaHMAC: preparar.IndiceIdempotenciaHMAC,
		HuellaSolicitudHMAC:    preparar.HuellaSolicitudHMAC, Manifiesto: manifiesto,
		SecuenciaCercado: token.Secuencia(), HuellaVinculoSHA256: token.HuellaVinculoSHA256(),
		ClaveAtestacionCercadoRef:     token.claveAtestacionRef,
		HuellaMACCercadoSHA256:        huellaBytesFormatoDocumental(token.macAtestacion),
		EvidenciaAtestacionCercadoRef: token.evidenciaOperacionRef,
		VerificacionCercadoRef:        verificacionCercado.verificacionRef,
		VerificadoCercadoEn:           verificacionCercado.verificadaEn,
		ConsumoDecision:               consumo, Resultado: resultado, Recibos: huellasRecibos,
		GeneradoEn: instante.Add(2 * time.Second), ConfirmadoEn: instante.Add(3 * time.Second),
	}
	perfilSello, _ := NuevoPerfilSelloEvidenciaHMACSHA256V3("clave:evidencia:v3")
	firma, err := NuevaSolicitudFirmaEvidenciaRenderizadoDocumentalV3(perfilSello, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	sello, err := NuevoSelloEvidenciaDocumentalV3Nominal(
		firma, []byte(strings.Repeat("s", 32)), "evidencia:firma:v3:001", instante.Add(3*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitudVerificacion, _ := NuevaSolicitudVerificacionEvidenciaDocumentalV3(firma, sello)
	verificacion, err := NuevosMetadatosComprobacionEvidenciaDocumentalV3Nominal(
		solicitudVerificacion, "verificacion:evidencia:v3:001", instante.Add(4*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion := SolicitudConfirmarEjecucionDocumentalV3{
		ReservaRef: evidencia.ReservaRef, Manifiesto: manifiesto, ConsumoDecision: consumo,
		OrdenDespachoConsumida: ordenDespachoConsumida, Resultado: resultado,
		Recibos: recibos, Evidencia: evidencia, Sello: sello,
	}
	return escenarioEjecucionDocumentalV3Prueba{
		instante: instante, manifiesto: manifiesto, preparar: preparar, consumo: consumo,
		vinculoActivacion: vinculoActivacion,
		token:             token, verificacionCercado: verificacionCercado,
		ordenDespachoConsumida: ordenDespachoConsumida, resultado: resultado,
		recibos: recibos, evidencia: evidencia, sello: sello,
		verificacion: verificacion, confirmacion: confirmacion,
	}
}

func descriptorComponenteEjecucionDocumentalV3Prueba(
	t *testing.T,
	descriptor DescriptorPerfilDocumental,
	rol domain.RolComponenteDocumental,
	identificador, dominio string,
	semilla byte,
	maximo uint64,
) DescriptorComponenteDocumentalAtestado {
	t.Helper()
	huella := func(offset byte) string {
		alfabeto := "0123456789abcdef"
		return strings.Repeat(string(alfabeto[(int(semilla-'0')+int(offset))%len(alfabeto)]), 64)
	}
	consulta := consultaComponenteEjecucionDocumentalV3(descriptor, rol)
	componente, err := domain.NuevaReferenciaComponenteDocumental(
		rol, identificador, uint64(semilla), "homologacion:"+identificador,
		huella(0), huella(1),
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := NuevoDescriptorComponenteDocumentalAtestado(
		"descriptor:"+identificador, consulta, componente, dominio,
		"broker:"+identificador, "atestacion:"+identificador, huella(2), maximo,
	)
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}

func reciboEjecucionDocumentalV3Prueba(
	t *testing.T,
	nombre string,
	operacion OperacionComponenteDocumental,
	componente DescriptorComponenteDocumentalAtestado,
	descriptor DescriptorPerfilDocumental,
	situacion domain.SituacionOperativaPerfilDocumental,
	ordenDespachoConsumida OrdenDespachoDocumentalV3ConsumidaNominal,
	resultado ResultadoEfectoRenderizadoDocumentalV3Crudo,
	instante time.Time,
	semilla byte,
) ReciboEjecucionComponenteDocumentalNominal {
	t.Helper()
	vinculoActivacion, err := ordenDespachoConsumida.VinculoActivacion()
	if err != nil {
		t.Fatal(err)
	}
	manifiesto := vinculoActivacion.Manifiesto
	var reto [32]byte
	reto[0] = semilla
	huellaDocumento := resultado.HuellaSalidaSHA256
	tamano := resultado.TamanoSalida
	if operacion == OperacionRenderizadoDocumental {
		huellaDocumento, tamano = "", 0
	}
	compromiso, err := NuevoCompromisoEjecucionComponenteDocumental(
		"operacion:"+nombre+":v3", reto, operacion, descriptor, situacion, componente,
		ordenDespachoConsumida,
		resultado.BorradorRef, manifiesto.datos.HuellaEntradaHMAC, huellaDocumento,
		tamano, manifiesto.datos.LimiteEfectivoBytes, 5*time.Minute,
	)
	if err != nil {
		t.Fatal(err)
	}
	sobre, err := NuevoSobreReciboEjecucionDocumentalCrudo([]byte(strings.Repeat(string('k'+semilla), 32)))
	if err != nil {
		t.Fatal(err)
	}
	artefacto := componente.Componente().HuellaArtefactoSHA256()
	identidad, err := NuevaIdentidadEjecucionComponenteDocumental(
		"carga:"+nombre+":v3", "instancia:"+nombre+":v3", componente.DominioConfianzaRef(),
		"clave:"+nombre+":v3", strings.Repeat(string('1'+semilla), 64), artefacto,
	)
	if err != nil {
		t.Fatal(err)
	}
	resultadoRecibo := ResultadoRenderizadoDocumentalCorrecto
	if operacion == OperacionValidacionEstructuralDocumental {
		resultadoRecibo = ResultadoEstructuraDocumentalConforme
	} else if operacion == OperacionVerificacionSemanticaDocumental {
		resultadoRecibo = ResultadoSemanticaDocumentalEquivalente
	}
	recibo, err := NuevoReciboEjecucionComponenteDocumentalNominal(
		compromiso, sobre, "recibo:"+nombre+":v3", resultadoRecibo,
		resultado.HuellaSalidaSHA256, resultado.TamanoSalida, identidad,
		ordenDespachoConsumida.estado.consumidaEn.Add(time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	return recibo
}

func confirmarReconciliacionEjecucionDocumentalV3Prueba(
	t *testing.T,
	escenario escenarioEjecucionDocumentalV3Prueba,
	resultado ResultadoConsultaEfectoDocumentalV3Crudo,
	verificacion MetadatosComprobacionReconciliacionDocumentalV3Nominal,
) SolicitudConfirmarEjecucionDocumentalV3 {
	t.Helper()
	evidencia := escenario.evidencia
	evidencia.ReconciliacionRef = resultado.AtestacionRef
	evidencia.HuellaReconciliacionSHA256 = resultado.HuellaAtestacionSHA256
	evidencia.ReconciliacionConsultadaEn = resultado.ConsultadaEn
	evidencia.VerificacionReconciliacionRef = verificacion.verificacionRef
	evidencia.ReconciliacionVerificadaEn = verificacion.verificadaEn
	evidencia.ConfirmadoEn = verificacion.verificadaEn.Add(time.Second)
	perfil, _ := NuevoPerfilSelloEvidenciaHMACSHA256V3("clave:evidencia:v3")
	firma, err := NuevaSolicitudFirmaEvidenciaRenderizadoDocumentalV3(perfil, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	sello, err := NuevoSelloEvidenciaDocumentalV3Nominal(
		firma, []byte(strings.Repeat("r", 32)), "evidencia:firma:reconciliada:v3",
		evidencia.ConfirmadoEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion := escenario.confirmacion
	confirmacion.Evidencia = evidencia
	confirmacion.Sello = sello
	return confirmacion
}

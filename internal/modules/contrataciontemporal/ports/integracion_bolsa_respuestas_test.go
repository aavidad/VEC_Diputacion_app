package ports

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestContextoPeticionEsOpacoYSeRehidrataSoloTrasAutenticar(t *testing.T) {
	base := instanteBolsaPrueba()
	necesidad := referenciaBolsaPrueba("necesidad:temporal", "a")
	datos := datosContextoBolsaPrueba(
		base,
		"operacion:consultar:disponibilidad",
		necesidad,
		referenciaBolsaPrueba("accion:consultar:disponibilidad", "2"),
	)
	contexto := emitirContextoBolsaPrueba(t, datos)

	if _, err := json.Marshal(contexto); !errors.Is(err, ErrSerializacionCapacidadBolsa) {
		t.Fatalf("la capacidad opaca se serializó como DTO: %v", err)
	}
	var fabricado ContextoPeticionIntegracionBolsa
	if err := json.Unmarshal([]byte(`{"datos":{}}`), &fabricado); !errors.Is(
		err,
		ErrSerializacionCapacidadBolsa,
	) {
		t.Fatalf("la capacidad se fabricó desde JSON: %v", err)
	}

	registro, err := contexto.Registro()
	if err != nil {
		t.Fatalf("extraer registro durable: %v", err)
	}
	contenido, err := json.Marshal(registro)
	if err != nil {
		t.Fatalf("serializar registro firmado: %v", err)
	}
	var recuperado RegistroContextoPeticionIntegracionBolsa
	if err := json.Unmarshal(contenido, &recuperado); err != nil {
		t.Fatalf("recuperar registro firmado: %v", err)
	}
	autenticador, err := NuevoAutenticadorContextoPeticionIntegracionBolsa(
		autoridadPeticionBolsaPrueba,
		clavePeticionBolsaV1Prueba,
		nil,
		&verificadorHMACBolsaPrueba{
			claves: map[string][]byte{
				clavePeticionBolsaV1Prueba: secretoPeticionBolsaV1Prueba,
			},
		},
	)
	if err != nil {
		t.Fatalf("crear autenticador: %v", err)
	}
	rehidratado, err := autenticador.Reautenticar(
		context.Background(),
		recuperado,
		base.Add(time.Minute),
	)
	if err != nil || !registrosContextoIguales(contexto, rehidratado) {
		t.Fatalf("registro auténtico no rehidratado: %v", err)
	}

	alterado := recuperado
	alterado.Datos.ExpedienteRef = "expediente:otro"
	if _, err := autenticador.Reautenticar(
		context.Background(),
		alterado,
		base.Add(time.Minute),
	); !errors.Is(err, ErrPeticionIntegracionBolsaInvalida) {
		t.Fatalf("registro alterado rehidratado: %v", err)
	}
	if _, err := autenticador.Reautenticar(
		context.Background(),
		recuperado,
		datos.ValidaHasta,
	); !errors.Is(err, ErrPeticionIntegracionBolsaInvalida) {
		t.Fatalf("petición caducada rehidratada para uso en línea: %v", err)
	}
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := autenticador.Reautenticar(
		cancelado,
		recuperado,
		base.Add(time.Minute),
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("se perdió cancelación: %v", err)
	}
}

func TestCancelacionDuranteCriptografiaNoPromueveCapacidades(t *testing.T) {
	base := instanteBolsaPrueba()
	necesidad := referenciaBolsaPrueba("necesidad:temporal", "a")
	datos := datosContextoBolsaPrueba(
		base,
		"operacion:cancelacion:criptografia",
		necesidad,
		referenciaBolsaPrueba("accion:consultar:disponibilidad", "2"),
	)

	ctxSellado, cancelarSellado := context.WithCancel(context.Background())
	selladorCancelador := selladorPeticionBolsaPrueba()
	selladorCancelador.cancelar = cancelarSellado
	emisor, err := NuevoEmisorContextoPeticionIntegracionBolsa(
		autoridadPeticionBolsaPrueba,
		clavePeticionBolsaV1Prueba,
		selladorCancelador,
	)
	if err != nil {
		t.Fatalf("crear emisor: %v", err)
	}
	emitido, err := emisor.Emitir(ctxSellado, datos, base)
	if !errors.Is(err, context.Canceled) || emitido.datos != nil {
		t.Fatalf("sellado cancelado promovió contexto: contexto=%v err=%v", emitido, err)
	}

	contextoValido := emitirContextoBolsaPrueba(t, datos)
	registro, err := contextoValido.Registro()
	if err != nil {
		t.Fatalf("extraer registro: %v", err)
	}
	ctxVerificacion, cancelarVerificacion := context.WithCancel(context.Background())
	verificadorCancelador := &verificadorHMACBolsaPrueba{
		claves: map[string][]byte{
			clavePeticionBolsaV1Prueba: secretoPeticionBolsaV1Prueba,
		},
		cancelar: cancelarVerificacion,
	}
	autenticador, err := NuevoAutenticadorContextoPeticionIntegracionBolsa(
		autoridadPeticionBolsaPrueba,
		clavePeticionBolsaV1Prueba,
		nil,
		verificadorCancelador,
	)
	if err != nil {
		t.Fatalf("crear autenticador: %v", err)
	}
	rehidratado, err := autenticador.Reautenticar(
		ctxVerificacion,
		registro,
		base.Add(time.Minute),
	)
	if !errors.Is(err, context.Canceled) || rehidratado.datos != nil {
		t.Fatalf(
			"verificación cancelada promovió contexto: contexto=%v err=%v",
			rehidratado,
			err,
		)
	}

	solicitud := solicitudDisponibilidadPrueba(t, base)
	resultado := resultadoDisponibilidadPrueba(t, base)
	firmarDisponibilidadPrueba(
		t,
		selladorRespuestaBolsaPrueba(),
		solicitud,
		&resultado,
	)
	ctxEvidencia, cancelarEvidencia := context.WithCancel(context.Background())
	verificadorRespuesta, err := NuevoVerificadorEvidenciaIntegracionBolsa(
		autoridadRespuestaBolsaPrueba,
		claveRespuestaBolsaV1Prueba,
		nil,
		&verificadorHMACBolsaPrueba{
			claves: map[string][]byte{
				claveRespuestaBolsaV1Prueba: secretoRespuestaBolsaV1Prueba,
			},
			cancelar: cancelarEvidencia,
		},
	)
	if err != nil {
		t.Fatalf("crear verificador de respuesta: %v", err)
	}
	comprobante, _, err := verificadorRespuesta.VerificarDisponibilidad(
		ctxEvidencia,
		solicitud,
		resultado,
		base.Add(3*time.Minute),
	)
	if !errors.Is(err, context.Canceled) || comprobante.datos != nil {
		t.Fatalf(
			"verificación de evidencia cancelada promovió prueba: prueba=%v err=%v",
			comprobante,
			err,
		)
	}
}

func TestRespuestaDisponibilidadExigeFrescuraYAutenticacion(t *testing.T) {
	base := instanteBolsaPrueba()
	ahora := base.Add(3 * time.Minute)
	solicitud := solicitudDisponibilidadPrueba(t, base)
	resultado := resultadoDisponibilidadPrueba(t, base)
	sellador := selladorRespuestaBolsaPrueba()
	firmarDisponibilidadPrueba(t, sellador, solicitud, &resultado)
	verificador := verificadorRespuestaBolsaPrueba(claveRespuestaBolsaV1Prueba)

	comprobante, evidencia, err := verificador.VerificarDisponibilidad(
		context.Background(), solicitud, resultado, ahora,
	)
	if err != nil || comprobante.datos == nil || evidencia.Validar() != nil {
		t.Fatalf("respuesta auténtica no promovida: %v", err)
	}
	forjada := resultado
	forjada.Procedencia.Evidencia.SelloHMAC =
		selloNominalBolsaPrueba(claveRespuestaBolsaV1Prueba, "4")
	if _, _, err := verificador.VerificarDisponibilidad(
		context.Background(), solicitud, forjada, ahora,
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("sello nominal fabricado obtuvo comprobante: %v", err)
	}
	alterada := resultado
	alterada.CantidadDisponible++
	if _, _, err := verificador.VerificarDisponibilidad(
		context.Background(), solicitud, alterada, ahora,
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("respuesta alterada conservó autenticidad: %v", err)
	}
	if _, _, err := verificador.VerificarDisponibilidad(
		context.Background(),
		solicitud,
		resultado,
		resultado.Procedencia.Evidencia.ValidaHasta,
	); !errors.Is(err, ErrRespuestaBolsaNoConfiable) {
		t.Fatalf("respuesta volátil caducada aceptada: %v", err)
	}
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, _, err := verificador.VerificarDisponibilidad(
		cancelado, solicitud, resultado, ahora,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("se perdió cancelación del verificador: %v", err)
	}
}

func TestVerificadorFijaAutoridadYConservaSoloClavesRetenidas(t *testing.T) {
	base := instanteBolsaPrueba()
	ahora := base.Add(3 * time.Minute)
	comando := comandoOrdenPrueba(t, base)
	recibo := reciboOrdenPrueba(t, base)
	firmarOrdenPrueba(t, selladorRespuestaBolsaPrueba(), comando, &recibo)

	rotado := verificadorRespuestaBolsaPrueba(
		claveRespuestaBolsaV2Prueba,
		claveRespuestaBolsaV1Prueba,
	)
	if _, _, err := rotado.VerificarReciboOrden(
		context.Background(), comando, recibo, ahora,
	); err != nil {
		t.Fatalf("v1 retenida dejó de verificar tras activar v2: %v", err)
	}
	soloV2 := verificadorRespuestaBolsaPrueba(claveRespuestaBolsaV2Prueba)
	if _, _, err := soloV2.VerificarReciboOrden(
		context.Background(), comando, recibo, ahora,
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("v1 no retenida fue aceptada: %v", err)
	}

	otraAutoridad := recibo
	otraAutoridad.Procedencia.AutoridadRef = "autoridad:otra"
	firmarOrdenPrueba(t, selladorRespuestaBolsaPrueba(), comando, &otraAutoridad)
	espia := &verificadorHMACBolsaPrueba{
		claves: map[string][]byte{
			claveRespuestaBolsaV1Prueba: secretoRespuestaBolsaV1Prueba,
			claveRespuestaBolsaV2Prueba: secretoRespuestaBolsaV2Prueba,
		},
	}
	verificadorAutoridad, err := NuevoVerificadorEvidenciaIntegracionBolsa(
		autoridadRespuestaBolsaPrueba,
		claveRespuestaBolsaV2Prueba,
		[]string{claveRespuestaBolsaV1Prueba},
		espia,
	)
	if err != nil {
		t.Fatalf("crear verificador de autoridad: %v", err)
	}
	if _, _, err := verificadorAutoridad.VerificarReciboOrden(
		context.Background(), comando, otraAutoridad, ahora,
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("autoridad recibida amplió confianza local: %v", err)
	}
	if espia.usos.Load() != 0 {
		t.Fatal("la autoridad no esperada llegó al conector criptográfico")
	}

	tipo := reflect.TypeOf((*VerificadorHMACIntegracionBolsa)(nil)).Elem()
	if _, existe := tipo.MethodByName("SellarDatos"); existe {
		t.Fatal("el verificador expone capacidad de firma")
	}
	if tipo.NumMethod() != 1 || tipo.Method(0).Name != "VerificarDatos" {
		t.Fatalf("TCB de verificación demasiado amplio: %v", tipo)
	}
}

func TestAnilloRechazaGeneracionesDesordenadasDuplicadasOExcesivas(t *testing.T) {
	verificador := &verificadorHMACBolsaPrueba{
		claves: map[string][]byte{
			claveRespuestaBolsaV1Prueba: secretoRespuestaBolsaV1Prueba,
			claveRespuestaBolsaV2Prueba: secretoRespuestaBolsaV2Prueba,
		},
	}
	casos := []struct {
		nombre    string
		activa    string
		retenidas []string
	}{
		{
			nombre:    "generacion futura",
			activa:    claveRespuestaBolsaV1Prueba,
			retenidas: []string{claveRespuestaBolsaV2Prueba},
		},
		{
			nombre: "generacion duplicada",
			activa: claveRespuestaBolsaV2Prueba,
			retenidas: []string{
				claveRespuestaBolsaV1Prueba,
				claveRespuestaBolsaV1Prueba,
			},
		},
		{
			nombre: "retencion excesiva",
			activa: claveRespuestaBolsaV2Prueba,
			retenidas: []string{
				claveRespuestaBolsaV1Prueba,
				claveRespuestaBolsaV1Prueba,
				claveRespuestaBolsaV1Prueba,
				claveRespuestaBolsaV1Prueba,
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := NuevoVerificadorEvidenciaIntegracionBolsa(
				autoridadRespuestaBolsaPrueba,
				caso.activa,
				caso.retenidas,
				verificador,
			); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
				t.Fatalf("anillo inválido aceptado: %v", err)
			}
		})
	}
}

func TestEvidenciaDurableSeReautenticaTrasReinicioSinConfundirFrescura(t *testing.T) {
	base := instanteBolsaPrueba()
	ahora := base.Add(3 * time.Minute)
	comando := comandoOrdenPrueba(t, base)
	recibo := reciboOrdenPrueba(t, base)
	firmarOrdenPrueba(t, selladorRespuestaBolsaPrueba(), comando, &recibo)
	verificador := verificadorRespuestaBolsaPrueba(claveRespuestaBolsaV1Prueba)

	_, evidencia, err := verificador.VerificarReciboOrden(
		context.Background(), comando, recibo, ahora,
	)
	if err != nil {
		t.Fatalf("autenticar recibo inicial: %v", err)
	}
	contenido, err := json.Marshal(evidencia)
	if err != nil {
		t.Fatalf("persistir evidencia: %v", err)
	}
	var recuperada EvidenciaDurableIntegracionBolsa
	if err := json.Unmarshal(contenido, &recuperada); err != nil {
		t.Fatalf("recuperar evidencia: %v", err)
	}
	trasReinicio := verificadorRespuestaBolsaPrueba(claveRespuestaBolsaV1Prueba)
	instantePosterior := base.Add(24 * time.Hour)
	if _, _, err := trasReinicio.VerificarReciboOrden(
		context.Background(), comando, recibo, instantePosterior,
	); !errors.Is(err, ErrRespuestaBolsaNoConfiable) {
		t.Fatalf("transporte caducado se trató como fresco: %v", err)
	}
	if _, err := trasReinicio.reautenticarReciboOrden(
		context.Background(),
		comando,
		recibo,
		recuperada,
		instantePosterior,
	); err != nil {
		t.Fatalf("evidencia durable auténtica no sobrevivió al reinicio: %v", err)
	}

	alteraciones := []struct {
		nombre  string
		cambiar func(*EvidenciaDurableIntegracionBolsa)
	}{
		{"peticion", func(e *EvidenciaDurableIntegracionBolsa) {
			e.HuellaPeticionSHA256 = strings.Repeat("2", 64)
		}},
		{"respuesta", func(e *EvidenciaDurableIntegracionBolsa) {
			e.HuellaRespuestaSHA256 = strings.Repeat("3", 64)
		}},
		{"autoridad", func(e *EvidenciaDurableIntegracionBolsa) {
			e.AutoridadRef = "autoridad:otra"
		}},
		{"instante", func(e *EvidenciaDurableIntegracionBolsa) {
			e.EmitidaEn = e.EmitidaEn.Add(time.Microsecond)
		}},
	}
	for _, prueba := range alteraciones {
		t.Run(prueba.nombre, func(t *testing.T) {
			alterada := recuperada
			prueba.cambiar(&alterada)
			if _, err := trasReinicio.reautenticarReciboOrden(
				context.Background(),
				comando,
				recibo,
				alterada,
				instantePosterior,
			); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
				t.Fatalf("evidencia alterada aceptada: %v", err)
			}
		})
	}
	reciboAlterado := recibo
	reciboAlterado.TotalPosiciones--
	if _, err := trasReinicio.reautenticarReciboOrden(
		context.Background(),
		comando,
		reciboAlterado,
		recuperada,
		instantePosterior,
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("recibo alterado reautenticado: %v", err)
	}
}

func TestOrdenYLlamamientoQuedanLigadosDeFormaExacta(t *testing.T) {
	base := instanteBolsaPrueba()
	ahora := base.Add(3 * time.Minute)
	sellador := selladorRespuestaBolsaPrueba()
	comandoOrden := comandoOrdenPrueba(t, base)
	reciboOrden := reciboOrdenPrueba(t, base)
	firmarOrdenPrueba(t, sellador, comandoOrden, &reciboOrden)
	comprobante, _, err := verificadorRespuestaBolsaPrueba(
		claveRespuestaBolsaV1Prueba,
	).VerificarReciboOrden(context.Background(), comandoOrden, reciboOrden, ahora)
	if err != nil {
		t.Fatalf("autenticar orden: %v", err)
	}
	datosBase := datosContextoBolsaPrueba(
		base,
		"operacion:solicitar:llamamiento",
		reciboOrden.Orden,
		reciboOrden.AccionLlamamiento,
	)
	preparar := func(datos DatosContextoPeticionIntegracionBolsa) error {
		_, err := NuevoComandoSolicitarLlamamientoBolsa(
			PreparacionComandoSolicitarLlamamientoBolsa{
				Contexto:     emitirContextoBolsaPrueba(t, datos),
				ComandoOrden: comandoOrden, ReciboOrden: reciboOrden,
				ComprobanteOrden: comprobante, MaximaPosicionEvaluable: 50,
			},
			ahora,
		)
		return err
	}
	if err := preparar(datosBase); err != nil {
		t.Fatalf("vínculo exacto rechazado: %v", err)
	}
	if _, err := NuevoComandoSolicitarLlamamientoBolsa(
		PreparacionComandoSolicitarLlamamientoBolsa{
			Contexto:     emitirContextoBolsaPrueba(t, datosBase),
			ComandoOrden: comandoOrden, ReciboOrden: reciboOrden,
			ComprobanteOrden: comprobante, MaximaPosicionEvaluable: 50,
		},
		ahora.Add(time.Minute),
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("comprobante histórico se reutilizó para emitir: %v", err)
	}
	pruebas := []struct {
		nombre  string
		cambiar func(*DatosContextoPeticionIntegracionBolsa)
	}{
		{"organizacion", func(d *DatosContextoPeticionIntegracionBolsa) {
			d.OrganizacionRef = "organizacion:otra"
		}},
		{"expediente", func(d *DatosContextoPeticionIntegracionBolsa) {
			d.ExpedienteRef = "expediente:otro"
		}},
		{"version", func(d *DatosContextoPeticionIntegracionBolsa) {
			d.VersionExpediente++
		}},
		{"correlacion", func(d *DatosContextoPeticionIntegracionBolsa) {
			d.CorrelacionRef = "correlacion:otra"
		}},
		{"finalidad", func(d *DatosContextoPeticionIntegracionBolsa) {
			d.Finalidad = referenciaBolsaPrueba("finalidad:otra", "6")
		}},
		{"accion", func(d *DatosContextoPeticionIntegracionBolsa) {
			d.Accion = referenciaBolsaPrueba("accion:otra", "6")
		}},
		{"recurso", func(d *DatosContextoPeticionIntegracionBolsa) {
			d.Recurso = referenciaBolsaPrueba("orden:otra", "6")
		}},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			datos := datosBase
			prueba.cambiar(&datos)
			exigirErrorBolsa(t, preparar(datos), ErrPeticionIntegracionBolsaInvalida)
		})
	}

	preparacionSinPrueba := PreparacionComandoSolicitarLlamamientoBolsa{
		Contexto:     emitirContextoBolsaPrueba(t, datosBase),
		ComandoOrden: comandoOrden, ReciboOrden: reciboOrden,
		MaximaPosicionEvaluable: 50,
	}
	if _, err := NuevoComandoSolicitarLlamamientoBolsa(
		preparacionSinPrueba,
		ahora,
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("orden sin comprobante creó llamamiento: %v", err)
	}
}

func TestContratoEsNeutralAWebEscritorioCLIYMCP(t *testing.T) {
	tipos := []reflect.Type{
		reflect.TypeOf(DatosContextoPeticionIntegracionBolsa{}),
		reflect.TypeOf(SolicitudDisponibilidadBolsa{}),
		reflect.TypeOf(ResultadoDisponibilidadBolsa{}),
		reflect.TypeOf(ComandoPrepararOrdenBolsa{}),
		reflect.TypeOf(ReciboOrdenBolsa{}),
		reflect.TypeOf(DatosComandoSolicitarLlamamientoBolsa{}),
		reflect.TypeOf(ReciboSolicitudLlamamientoBolsa{}),
		reflect.TypeOf(EventoLlamamientoBolsa{}),
		reflect.TypeOf(ArtefactoProbatorioOrdenBolsa{}),
		reflect.TypeOf(ArtefactoProbatorioLlamamientoBolsa{}),
		reflect.TypeOf(ArtefactoProbatorioEventoBolsa{}),
	}
	prohibidos := []string{
		"dni", "nie", "nif", "nombre", "apellidos", "correo", "email",
		"telefono", "movil", "direccion", "cookie", "sesion", "cabecera",
		"remote_user", "http", "browser", "jwt",
	}
	for _, tipo := range tipos {
		comprobarCamposIntegracionBolsa(t, tipo, prohibidos)
	}
	tipoContexto := reflect.TypeOf((*context.Context)(nil)).Elem()
	for _, puerto := range []reflect.Type{
		reflect.TypeOf((*ConsultaDisponibilidadBolsa)(nil)).Elem(),
		reflect.TypeOf((*PreparadorOrdenBolsa)(nil)).Elem(),
		reflect.TypeOf((*GestorLlamamientosBolsa)(nil)).Elem(),
		reflect.TypeOf((*BandejaEventosLlamamientoBolsa)(nil)).Elem(),
	} {
		metodo := puerto.Method(0)
		if metodo.Type.NumIn() < 1 || metodo.Type.In(0) != tipoContexto {
			t.Fatalf("%s perdió context.Context neutral: %v", metodo.Name, metodo.Type)
		}
	}
}

func comprobarCamposIntegracionBolsa(t *testing.T, tipo reflect.Type, prohibidos []string) {
	t.Helper()
	for indice := 0; indice < tipo.NumField(); indice++ {
		campo := tipo.Field(indice)
		nombre := strings.ToLower(campo.Name + " " + campo.Tag.Get("json"))
		for _, prohibido := range prohibidos {
			if strings.Contains(nombre, prohibido) {
				t.Fatalf("%s expone campo prohibido %q", tipo.Name(), campo.Name)
			}
		}
		if campo.Type.Kind() == reflect.Struct &&
			campo.Type != reflect.TypeOf(time.Time{}) &&
			campo.Type != reflect.TypeOf(ContextoPeticionIntegracionBolsa{}) {
			comprobarCamposIntegracionBolsa(t, campo.Type, prohibidos)
		}
	}
}

func TestCapacidadesNoSonFabricablesNiSerializables(t *testing.T) {
	for _, valor := range []any{
		ComprobanteEvidenciaIntegracionBolsa{},
		ComandoSolicitarLlamamientoBolsa{},
		ComandoRegistrarEventoBolsa{},
		ContextoPeticionIntegracionBolsa{},
		EnlaceEventoLlamamientoBolsa{},
		OrdenProbatoriaRehidratadaBolsa{},
		LlamamientoProbatorioRehidratadoBolsa{},
		EventoProbatorioRehidratadoBolsa{},
	} {
		if _, err := json.Marshal(valor); !errors.Is(err, ErrSerializacionCapacidadBolsa) {
			t.Fatalf("%T se serializó: %v", valor, err)
		}
	}
	if _, err := NuevoVerificadorEvidenciaIntegracionBolsa(
		autoridadRespuestaBolsaPrueba,
		claveRespuestaBolsaV1Prueba,
		nil,
		(*verificadorHMACBolsaPrueba)(nil),
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("verificador aceptó dependencia tipada nula: %v", err)
	}
}

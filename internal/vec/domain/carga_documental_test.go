package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

var instanteCargaDocumental = time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)

func cargaDocumentalReservadaPrueba(t *testing.T) CargaDocumental {
	t.Helper()
	carga, err := NuevaCargaDocumental(
		"carga:bolsa:0123456789abcdef", "persona:0123456789abcdef", "solicitud:bolsa:0123456789abcdef",
		"bolsa", "solicitud", "operacion:carga:0123456789abcdef", "correlacion:0123456789abcdef",
		"aportar_documentacion_bolsa", "datos_personales", "application/pdf", 1024,
		strings.Repeat("a", 64), "hmac-sha256:idempotencia_v1:"+strings.Repeat("c", 64),
		"hmac-sha256:solicitud_v1:"+strings.Repeat("b", 64),
		instanteCargaDocumental, instanteCargaDocumental.Add(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("crear carga: %v", err)
	}
	return carga
}

func manifiestoPreparacionCargaDirectaPrueba(t *testing.T) (CargaDocumental, ManifiestoPreparacionCargaDirectaV1) {
	t.Helper()
	preparada, err := cargaDocumentalReservadaPrueba(t).Preparar(
		"hmac-sha256:sesion_v1:"+strings.Repeat("c", 64),
		"decision:preparar:0123456789abcdef",
		instanteCargaDocumental.Add(time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	manifiesto, err := NuevoManifiestoPreparacionCargaDirectaV1(
		preparada,
		ContextoManifiestoPreparacionCargaDirectaV1{
			CargaRef:                preparada.IndiceIdempotenciaHMAC,
			SujetoSeudonimoHMAC:     "hmac-sha256:seudonimo_v1:" + strings.Repeat("d", 64),
			HuellaRecursoBaseSHA256: strings.Repeat("2", 64),
			HuellaRecursoSHA256:     strings.Repeat("e", 64), ConectorAlmacenID: "almacen_s3_corporativo",
			EsquemaContexto: esquemaContextoAlmacenManifiestoV1,
			AccionNegocio:   accionPrepararCargaManifiestoV1, AccionTecnica: accionTecnicaPrepararManifiestoV1,
			PasoRef: pasoPrepararCargaManifiestoV1, EfectoRef: "efecto:carga:preparar:0123456789abcdef",
			HuellaPlanEfectoSHA256: strings.Repeat("f", 64),
			EsquemaHuellaDecision:  esquemaHuellaDecisionManifiestoV1,
			DecisionRef:            preparada.AutorizacionPreparacionRef,
			HuellaDecisionSHA256:   strings.Repeat("1", 64),
			ContextoVerificadoEn:   instanteCargaDocumental.Add(500 * time.Millisecond),
			DecisionValidaHasta:    preparada.ExpiraEn.Add(time.Minute),
		},
	)
	if err != nil {
		t.Fatalf("crear manifiesto: %v", err)
	}
	return preparada, manifiesto
}

func TestManifiestoPreparacionCargaDirectaEsCanonicoInmutableYLigado(t *testing.T) {
	preparada, manifiesto := manifiestoPreparacionCargaDirectaPrueba(t)
	if err := manifiesto.ValidarContraCarga(preparada); err != nil {
		t.Fatal(err)
	}
	datos, err := manifiesto.Datos()
	if err != nil || datos.HuellaManifiestoSHA256 == "" || datos.DecisionRef != preparada.AutorizacionPreparacionRef {
		t.Fatalf("datos del manifiesto: %#v, %v", datos, err)
	}
	huellaInicial := datos.HuellaManifiestoSHA256
	datos.MIME = "application/zip"
	datos.HuellaManifiestoSHA256 = strings.Repeat("0", 64)
	datosOtraLectura, err := manifiesto.Datos()
	if err != nil || datosOtraLectura.MIME != preparada.MIMEDeclarado ||
		datosOtraLectura.HuellaManifiestoSHA256 != huellaInicial {
		t.Fatal("una copia externa modifico el manifiesto opaco")
	}
	otra, err := NuevoManifiestoPreparacionCargaDirectaV1(preparada, ContextoManifiestoPreparacionCargaDirectaV1{
		CargaRef:                preparada.IndiceIdempotenciaHMAC,
		SujetoSeudonimoHMAC:     datosOtraLectura.SujetoSeudonimoHMAC,
		HuellaRecursoBaseSHA256: datosOtraLectura.HuellaRecursoBaseSHA256,
		HuellaRecursoSHA256:     datosOtraLectura.HuellaRecursoSHA256,
		ConectorAlmacenID:       datosOtraLectura.ConectorAlmacenID,
		EsquemaContexto:         datosOtraLectura.EsquemaContexto, AccionNegocio: datosOtraLectura.AccionNegocio,
		AccionTecnica: datosOtraLectura.AccionTecnica, PasoRef: datosOtraLectura.PasoRef,
		EfectoRef: datosOtraLectura.EfectoRef, HuellaPlanEfectoSHA256: datosOtraLectura.HuellaPlanEfectoSHA256,
		EsquemaHuellaDecision: datosOtraLectura.EsquemaHuellaDecision,
		DecisionRef:           datosOtraLectura.DecisionRef, HuellaDecisionSHA256: datosOtraLectura.HuellaDecisionSHA256,
		ContextoVerificadoEn: datosOtraLectura.ContextoVerificadoEn,
		DecisionValidaHasta:  datosOtraLectura.DecisionValidaHasta,
	})
	if err != nil {
		t.Fatal(err)
	}
	huellaOtra, _ := otra.HuellaSHA256()
	if huellaOtra != huellaInicial {
		t.Fatalf("la misma entrada no produjo la misma huella: %s != %s", huellaOtra, huellaInicial)
	}
}

func TestManifiestoPreparacionCargaDirectaSeRehidrataEstrictoDesdePersistencia(t *testing.T) {
	preparada, original := manifiestoPreparacionCargaDirectaPrueba(t)
	datos, err := original.Datos()
	if err != nil {
		t.Fatal(err)
	}
	persistidos, err := json.Marshal(datos)
	if err != nil {
		t.Fatal(err)
	}
	var leidos DatosManifiestoPreparacionCargaDirectaV1
	if err := json.Unmarshal(persistidos, &leidos); err != nil {
		t.Fatal(err)
	}
	rehidratado, err := RestaurarManifiestoPreparacionCargaDirectaV1(leidos)
	if err != nil || rehidratado.ValidarContraCarga(preparada) != nil {
		t.Fatalf("rehidratar manifiesto durable: %v", err)
	}
	huellaOriginal, _ := original.HuellaSHA256()
	huellaRehidratada, _ := rehidratado.HuellaSHA256()
	if huellaOriginal != huellaRehidratada {
		t.Fatalf("la persistencia cambio la huella: %s != %s", huellaOriginal, huellaRehidratada)
	}

	leidos.MIME = "application/zip"
	datosRehidratados, err := rehidratado.Datos()
	if err != nil || datosRehidratados.MIME != preparada.MIMEDeclarado {
		t.Fatal("la mutacion de la proyeccion persistida altero el valor opaco")
	}
	if _, err := RestaurarManifiestoPreparacionCargaDirectaV1(leidos); !errors.Is(err, ErrManifiestoPreparacionInvalido) {
		t.Fatalf("se rehidrato una fila alterada: %v", err)
	}
	leidos = datos
	leidos.HuellaRecursoBaseSHA256 = strings.Repeat("9", 64)
	if _, err := RestaurarManifiestoPreparacionCargaDirectaV1(leidos); !errors.Is(err, ErrManifiestoPreparacionInvalido) {
		t.Fatalf("se rehidrato una huella ABAC base alterada: %v", err)
	}

	var parcial DatosManifiestoPreparacionCargaDirectaV1
	if err := json.Unmarshal([]byte(`{"esquema":"vec.carga-directa.manifiesto-preparacion.v1"}`), &parcial); err != nil {
		t.Fatal(err)
	}
	if _, err := RestaurarManifiestoPreparacionCargaDirectaV1(parcial); !errors.Is(err, ErrManifiestoPreparacionInvalido) {
		t.Fatalf("se completaron datos parciales durante la rehidratacion: %v", err)
	}
}

func TestManifiestoPreparacionCargaDirectaFallaCerradoAnteCruceOTamper(t *testing.T) {
	preparada, manifiesto := manifiestoPreparacionCargaDirectaPrueba(t)
	cruzada := preparada
	cruzada.CorrelacionRef = "correlacion:otra:0123456789abcdef"
	if !errors.Is(manifiesto.ValidarContraCarga(cruzada), ErrManifiestoPreparacionInvalido) {
		t.Fatal("el manifiesto acepto otro agregado")
	}
	if !errors.Is((ManifiestoPreparacionCargaDirectaV1{}).Validar(), ErrManifiestoPreparacionInvalido) {
		t.Fatal("el valor cero del manifiesto fue aceptado")
	}
	_, err := NuevoManifiestoPreparacionCargaDirectaV1(preparada, ContextoManifiestoPreparacionCargaDirectaV1{
		CargaRef: preparada.ID,
	})
	if !errors.Is(err, ErrManifiestoPreparacionInvalido) {
		t.Fatalf("se aceptaron vinculos parciales o una carga distinta: %v", err)
	}
}

func TestCargaDocumentalSoloAvanzaConEvidenciaExacta(t *testing.T) {
	reservada := cargaDocumentalReservadaPrueba(t)
	preparada, err := reservada.Preparar(
		"hmac-sha256:sesion_v1:"+strings.Repeat("c", 64),
		"decision:preparar:0123456789abcdef",
		instanteCargaDocumental.Add(time.Second),
	)
	if err != nil || preparada.Estado != EstadoCargaDocumentalPreparada || preparada.Version != 2 {
		t.Fatalf("preparar: %#v, %v", preparada, err)
	}
	cuarentena := ContenidoCargaDocumental{
		ConectorID: "almacen:corporativo", Referencia: "objeto:cuarentena:0123456789abcdef", Version: "version:1",
		Zona: ZonaContenidoCargaCuarentena, MIME: reservada.MIMEDeclarado, Tamano: reservada.TamanoDeclarado,
		HuellaSHA256: reservada.HuellaDeclaradaSHA256, EvidenciaRef: "evidencia:almacen:0123456789abcdef",
		RegistradoEn: instanteCargaDocumental.Add(2 * time.Second),
	}
	recibida, err := preparada.RegistrarCuarentena(
		cuarentena, "decision:confirmar:0123456789abcdef", cuarentena.RegistradoEn,
	)
	if err != nil || recibida.Estado != EstadoCargaDocumentalCuarentena || recibida.Version != 3 {
		t.Fatalf("registrar cuarentena: %#v, %v", recibida, err)
	}
	analisis := AnalisisCargaDocumental{
		ObjetoReferencia: cuarentena.Referencia, ObjetoVersion: cuarentena.Version,
		HuellaObjetoSHA256: cuarentena.HuellaSHA256, ConectorAnalizadorID: "analizador:icap:corporativo",
		VersionConector: 1, Estado: EstadoAnalisisCargaLimpio, CodigoResultado: "limpio",
		EvidenciaRef: "evidencia:analisis:0123456789abcdef", HuellaEvidenciaSHA256: strings.Repeat("d", 64),
		CompletadoEn: instanteCargaDocumental.Add(3 * time.Second),
	}
	limpia, err := recibida.RegistrarAnalisis(
		analisis, "decision:analizar:0123456789abcdef", analisis.CompletadoEn,
	)
	if err != nil || limpia.Estado != EstadoCargaDocumentalAnalizadaLimpia || limpia.Version != 4 {
		t.Fatalf("registrar analisis: %#v, %v", limpia, err)
	}
	admitido := cuarentena
	admitido.Referencia = "objeto:admitido:0123456789abcdef"
	admitido.Version = "version:1"
	admitido.Zona = ZonaContenidoCargaAdmitida
	admitido.EvidenciaRef = "evidencia:promocion:0123456789abcdef"
	admitido.RegistradoEn = instanteCargaDocumental.Add(4 * time.Second)
	final, err := limpia.Admitir(admitido, "decision:promover:0123456789abcdef", admitido.RegistradoEn)
	if err != nil || final.Estado != EstadoCargaDocumentalAdmitida || final.Version != 5 || final.Validar() != nil {
		t.Fatalf("admitir: %#v, %v", final, err)
	}
	if huella, err := final.HuellaSHA256(); err != nil || len(huella) != 64 {
		t.Fatalf("huella del agregado: %q, %v", huella, err)
	}
}

func TestCargaDocumentalFallaCerradoAnteContenidoOAnalisisCruzado(t *testing.T) {
	reservada := cargaDocumentalReservadaPrueba(t)
	if _, err := reservada.Preparar("", "decision:preparar:uno", instanteCargaDocumental.Add(time.Second)); !errors.Is(err, ErrCargaDocumentalInvalida) {
		t.Fatalf("se esperaba rechazo de sesion sin vinculo HMAC: %v", err)
	}
	preparada, err := reservada.Preparar(
		"hmac-sha256:sesion_v1:"+strings.Repeat("c", 64), "decision:preparar:uno",
		instanteCargaDocumental.Add(time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	contenido := ContenidoCargaDocumental{
		ConectorID: "almacen:corporativo", Referencia: "objeto:cuarentena:uno", Version: "version:1",
		Zona: ZonaContenidoCargaCuarentena, MIME: reservada.MIMEDeclarado, Tamano: reservada.TamanoDeclarado + 1,
		HuellaSHA256: reservada.HuellaDeclaradaSHA256, EvidenciaRef: "evidencia:almacen:uno",
		RegistradoEn: instanteCargaDocumental.Add(2 * time.Second),
	}
	if _, err := preparada.RegistrarCuarentena(contenido, "decision:confirmar:uno", contenido.RegistradoEn); !errors.Is(err, ErrContenidoCargaNoCorresponde) {
		t.Fatalf("se acepto otro contenido: %v", err)
	}
	contenido.Tamano = reservada.TamanoDeclarado
	recibida, err := preparada.RegistrarCuarentena(contenido, "decision:confirmar:uno", contenido.RegistradoEn)
	if err != nil {
		t.Fatal(err)
	}
	analisis := AnalisisCargaDocumental{
		ObjetoReferencia: "objeto:cuarentena:otro", ObjetoVersion: contenido.Version,
		HuellaObjetoSHA256: contenido.HuellaSHA256, ConectorAnalizadorID: "analizador:icap:corporativo",
		VersionConector: 1, Estado: EstadoAnalisisCargaLimpio, CodigoResultado: "limpio",
		EvidenciaRef: "evidencia:analisis:uno", HuellaEvidenciaSHA256: strings.Repeat("d", 64),
		CompletadoEn: instanteCargaDocumental.Add(3 * time.Second),
	}
	if _, err := recibida.RegistrarAnalisis(analisis, "decision:analizar:uno", analisis.CompletadoEn); !errors.Is(err, ErrAnalisisCargaNoCorresponde) {
		t.Fatalf("se acepto el analisis de otro objeto: %v", err)
	}
}

func TestTransicionesCargaNoNormalizanVinculosNiAutorizaciones(t *testing.T) {
	reservada := cargaDocumentalReservadaPrueba(t)
	vinculo := "hmac-sha256:sesion_v1:" + strings.Repeat("c", 64)
	preparadaEn := instanteCargaDocumental.Add(time.Second)
	for _, entrada := range []struct {
		nombre       string
		vinculo      string
		autorizacion string
	}{
		{nombre: "vinculo", vinculo: " " + vinculo, autorizacion: "decision:preparar:uno"},
		{nombre: "autorizacion", vinculo: vinculo, autorizacion: "decision:preparar:uno "},
	} {
		t.Run("preparar_"+entrada.nombre, func(t *testing.T) {
			if _, err := reservada.Preparar(entrada.vinculo, entrada.autorizacion, preparadaEn); !errors.Is(err, ErrCargaDocumentalInvalida) {
				t.Fatalf("la preparacion normalizo %s: %v", entrada.nombre, err)
			}
		})
	}

	preparada, err := reservada.Preparar(vinculo, "decision:preparar:uno", preparadaEn)
	if err != nil {
		t.Fatalf("preparar carga valida: %v", err)
	}
	contenido := ContenidoCargaDocumental{
		ConectorID: "almacen:corporativo", Referencia: "objeto:cuarentena:uno", Version: "version:1",
		Zona: ZonaContenidoCargaCuarentena, MIME: reservada.MIMEDeclarado, Tamano: reservada.TamanoDeclarado,
		HuellaSHA256: reservada.HuellaDeclaradaSHA256, EvidenciaRef: "evidencia:almacen:uno",
		RegistradoEn: instanteCargaDocumental.Add(2 * time.Second),
	}
	if _, err := preparada.RegistrarCuarentena(contenido, " decision:confirmar:uno", contenido.RegistradoEn); !errors.Is(err, ErrCargaDocumentalInvalida) {
		t.Fatalf("la recepcion normalizo la autorizacion: %v", err)
	}
	recibida, err := preparada.RegistrarCuarentena(contenido, "decision:confirmar:uno", contenido.RegistradoEn)
	if err != nil {
		t.Fatalf("registrar cuarentena valida: %v", err)
	}

	analisis := AnalisisCargaDocumental{
		ObjetoReferencia: contenido.Referencia, ObjetoVersion: contenido.Version,
		HuellaObjetoSHA256: contenido.HuellaSHA256, ConectorAnalizadorID: "analizador:icap:corporativo",
		VersionConector: 1, Estado: EstadoAnalisisCargaLimpio, CodigoResultado: "limpio",
		EvidenciaRef: "evidencia:analisis:uno", HuellaEvidenciaSHA256: strings.Repeat("d", 64),
		CompletadoEn: instanteCargaDocumental.Add(3 * time.Second),
	}
	if _, err := recibida.RegistrarAnalisis(analisis, "decision:analizar:uno ", analisis.CompletadoEn); !errors.Is(err, ErrCargaDocumentalInvalida) {
		t.Fatalf("el analisis normalizo la autorizacion: %v", err)
	}
	limpia, err := recibida.RegistrarAnalisis(analisis, "decision:analizar:uno", analisis.CompletadoEn)
	if err != nil {
		t.Fatalf("registrar analisis valido: %v", err)
	}

	admitido := contenido
	admitido.Referencia = "objeto:admitido:uno"
	admitido.Zona = ZonaContenidoCargaAdmitida
	admitido.EvidenciaRef = "evidencia:promocion:uno"
	admitido.RegistradoEn = instanteCargaDocumental.Add(4 * time.Second)
	if _, err := limpia.Admitir(admitido, " decision:promover:uno", admitido.RegistradoEn); !errors.Is(err, ErrCargaDocumentalInvalida) {
		t.Fatalf("la promocion normalizo la autorizacion: %v", err)
	}
}

func TestResultadoNoLimpioQuedaRetenidoYNoPuedePromoverse(t *testing.T) {
	reservada := cargaDocumentalReservadaPrueba(t)
	preparada, _ := reservada.Preparar(
		"hmac-sha256:sesion_v1:"+strings.Repeat("c", 64), "decision:preparar:uno",
		instanteCargaDocumental.Add(time.Second),
	)
	contenido := ContenidoCargaDocumental{
		ConectorID: "almacen:corporativo", Referencia: "objeto:cuarentena:uno", Version: "version:1",
		Zona: ZonaContenidoCargaCuarentena, MIME: reservada.MIMEDeclarado, Tamano: reservada.TamanoDeclarado,
		HuellaSHA256: reservada.HuellaDeclaradaSHA256, EvidenciaRef: "evidencia:almacen:uno",
		RegistradoEn: instanteCargaDocumental.Add(2 * time.Second),
	}
	recibida, _ := preparada.RegistrarCuarentena(contenido, "decision:confirmar:uno", contenido.RegistradoEn)
	analisis := AnalisisCargaDocumental{
		ObjetoReferencia: contenido.Referencia, ObjetoVersion: contenido.Version,
		HuellaObjetoSHA256: contenido.HuellaSHA256, ConectorAnalizadorID: "analizador:icap:corporativo",
		VersionConector: 1, Estado: EstadoAnalisisCargaNoConcluyente, CodigoResultado: "motor_no_disponible",
		EvidenciaRef: "evidencia:analisis:uno", HuellaEvidenciaSHA256: strings.Repeat("d", 64),
		CompletadoEn: instanteCargaDocumental.Add(3 * time.Second),
	}
	retenida, err := recibida.RegistrarAnalisis(analisis, "decision:analizar:uno", analisis.CompletadoEn)
	if err != nil || retenida.Estado != EstadoCargaDocumentalRetenidaSeguridad {
		t.Fatalf("retener resultado no concluyente: %#v, %v", retenida, err)
	}
	admitido := contenido
	admitido.Referencia = "objeto:admitido:uno"
	admitido.Zona = ZonaContenidoCargaAdmitida
	if _, err := retenida.Admitir(admitido, "decision:promover:uno", instanteCargaDocumental.Add(4*time.Second)); !errors.Is(err, ErrContenidoCargaNoCorresponde) {
		t.Fatalf("se promovio un resultado no limpio: %v", err)
	}
}

func TestCargaDocumentalRechazaAutorizacionesAnticipadasYCronologiaNoEstricta(t *testing.T) {
	reservada := cargaDocumentalReservadaPrueba(t)
	anticipada := reservada
	anticipada.AutorizacionRecepcionRef = "decision:confirmar:anticipada"
	if !errors.Is(anticipada.Validar(), ErrCargaDocumentalInvalida) {
		t.Fatal("una reserva acepto una autorizacion de una transicion futura")
	}
	if _, err := reservada.Preparar(
		"hmac-sha256:sesion_v1:"+strings.Repeat("c", 64),
		"decision:preparar:uno",
		reservada.ActualizadaEn,
	); !errors.Is(err, ErrTransicionCargaNoPermitida) {
		t.Fatalf("se acepto preparar sin avance temporal estricto: %v", err)
	}

	preparada, err := reservada.Preparar(
		"hmac-sha256:sesion_v1:"+strings.Repeat("c", 64),
		"decision:preparar:uno",
		instanteCargaDocumental.Add(time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	anticipada = preparada
	anticipada.AutorizacionAnalisisRef = "decision:analizar:anticipada"
	if !errors.Is(anticipada.Validar(), ErrCargaDocumentalInvalida) {
		t.Fatal("una preparacion acepto una autorizacion de analisis anticipada")
	}

	contenido := ContenidoCargaDocumental{
		ConectorID: "almacen:corporativo", Referencia: "objeto:cuarentena:uno", Version: "version:1",
		Zona: ZonaContenidoCargaCuarentena, MIME: reservada.MIMEDeclarado, Tamano: reservada.TamanoDeclarado,
		HuellaSHA256: reservada.HuellaDeclaradaSHA256, EvidenciaRef: "evidencia:almacen:uno",
		RegistradoEn: preparada.ActualizadaEn,
	}
	if _, err := preparada.RegistrarCuarentena(
		contenido,
		"decision:confirmar:uno",
		preparada.ActualizadaEn.Add(time.Second),
	); !errors.Is(err, ErrContenidoCargaNoCorresponde) {
		t.Fatalf("se acepto evidencia de cuarentena no posterior a la preparacion: %v", err)
	}
	contenido.RegistradoEn = preparada.ActualizadaEn.Add(time.Second)
	if _, err := preparada.RegistrarCuarentena(
		contenido,
		"decision:confirmar:uno",
		preparada.ExpiraEn,
	); !errors.Is(err, ErrContenidoCargaNoCorresponde) {
		t.Fatalf("se confirmo una cuarentena al vencer la reserva: %v", err)
	}
}

func TestCargaDocumentalPromocionEsPosteriorAlAnalisis(t *testing.T) {
	reservada := cargaDocumentalReservadaPrueba(t)
	preparada, err := reservada.Preparar(
		"hmac-sha256:sesion_v1:"+strings.Repeat("c", 64),
		"decision:preparar:uno",
		instanteCargaDocumental.Add(time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	contenido := ContenidoCargaDocumental{
		ConectorID: "almacen:corporativo", Referencia: "objeto:cuarentena:uno", Version: "version:1",
		Zona: ZonaContenidoCargaCuarentena, MIME: reservada.MIMEDeclarado, Tamano: reservada.TamanoDeclarado,
		HuellaSHA256: reservada.HuellaDeclaradaSHA256, EvidenciaRef: "evidencia:almacen:uno",
		RegistradoEn: instanteCargaDocumental.Add(2 * time.Second),
	}
	cuarentena, err := preparada.RegistrarCuarentena(
		contenido,
		"decision:confirmar:uno",
		contenido.RegistradoEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	analisis := AnalisisCargaDocumental{
		ObjetoReferencia: contenido.Referencia, ObjetoVersion: contenido.Version,
		HuellaObjetoSHA256: contenido.HuellaSHA256, ConectorAnalizadorID: "analizador:icap:corporativo",
		VersionConector: 1, Estado: EstadoAnalisisCargaLimpio, CodigoResultado: "limpio",
		EvidenciaRef: "evidencia:analisis:uno", HuellaEvidenciaSHA256: strings.Repeat("d", 64),
		CompletadoEn: instanteCargaDocumental.Add(3 * time.Second),
	}
	limpia, err := cuarentena.RegistrarAnalisis(analisis, "decision:analizar:uno", analisis.CompletadoEn)
	if err != nil {
		t.Fatal(err)
	}
	admitido := contenido
	admitido.Referencia = "objeto:admitido:uno"
	admitido.Zona = ZonaContenidoCargaAdmitida
	admitido.EvidenciaRef = "evidencia:promocion:uno"
	admitido.RegistradoEn = analisis.CompletadoEn
	if _, err := limpia.Admitir(
		admitido,
		"decision:promover:uno",
		analisis.CompletadoEn.Add(time.Second),
	); !errors.Is(err, ErrContenidoCargaNoCorresponde) {
		t.Fatalf("se acepto evidencia de promocion simultanea al analisis: %v", err)
	}
	admitido.RegistradoEn = analisis.CompletadoEn.Add(time.Second)
	if _, err := limpia.Admitir(
		admitido,
		"decision:promover:uno",
		analisis.CompletadoEn,
	); !errors.Is(err, ErrContenidoCargaNoCorresponde) {
		t.Fatalf("se acepto una transicion de promocion no posterior al analisis: %v", err)
	}
}

package almacen

import (
	"strings"
	"testing"
	"time"
)

func TestTextoReferenciaYHuellaAplicanFormaCanonicaEstricta(t *testing.T) {
	t.Parallel()

	if !TextoSeguro("application/pdf", 255) || !ReferenciaOpacaValida("objeto:abc-123", 512) ||
		!SHA256HexadecimalValido(strings.Repeat("a", 64)) {
		t.Fatal("los valores canonicos validos fueron rechazados")
	}
	for nombre, valida := range map[string]bool{
		"texto con borde":        TextoSeguro(" pdf", 255),
		"texto con control":      TextoSeguro("pdf\n", 255),
		"texto sobre limite":     TextoSeguro("abc", 2),
		"referencia con espacio": ReferenciaOpacaValida("objeto uno", 512),
		"sha mayuscula":          SHA256HexadecimalValido(strings.Repeat("A", 64)),
		"sha no hexadecimal":     SHA256HexadecimalValido(strings.Repeat("z", 64)),
		"sha longitud distinta":  SHA256HexadecimalValido(strings.Repeat("a", 63)),
	} {
		if valida {
			t.Errorf("%s: se acepto un valor no canonico", nombre)
		}
	}
}

func TestDestinoYOrigenesCargaDirectaSonHTTPSExactos(t *testing.T) {
	t.Parallel()

	destino := "https://objetos.example.test/carga?firma=opaca"
	if !DestinoCargaDirectaValido(destino) || OrigenDestinoCargaDirecta(destino) != "https://objetos.example.test" {
		t.Fatal("se rechazo o proyecto mal un destino canonico")
	}
	for nombre, valor := range map[string]string{
		"http":           "http://objetos.example.test/carga",
		"credenciales":   "https://usuario@objetos.example.test/carga",
		"fragmento":      "https://objetos.example.test/carga#secreto",
		"host mixto":     "https://Objetos.example.test/carga",
		"consulta vacia": "https://objetos.example.test/carga?",
		"borde":          " https://objetos.example.test/carga",
	} {
		if DestinoCargaDirectaValido(valor) {
			t.Errorf("%s: se acepto el destino no canonico %q", nombre, valor)
		}
	}

	if !OrigenesCargaDirectaValidos([]string{"https://a.example.test", "https://b.example.test:8443"}) {
		t.Fatal("se rechazaron origenes canonicos distintos")
	}
	// Compatibilidad documentada: la declaracion historica admite la forma
	// HTTP, aunque ningun destino utilizable puede abandonar HTTPS.
	if !OrigenesCargaDirectaValidos([]string{"http://legado.example.test"}) ||
		DestinoCargaDirectaValido("http://legado.example.test/carga") {
		t.Fatal("cambio no coordinado en la compatibilidad historica de origenes")
	}
	for nombre, origenes := range map[string][]string{
		"ausentes":   nil,
		"duplicados": {"https://a.example.test", "https://a.example.test"},
		"con ruta":   {"https://a.example.test/carga"},
		"con barra":  {"https://a.example.test/"},
		"host mixto": {"https://A.example.test"},
	} {
		if OrigenesCargaDirectaValidos(origenes) {
			t.Errorf("%s: se acepto una lista de origenes no canonica", nombre)
		}
	}
}

func TestCabecerasCargaDirectaUsanListaPositivaSinDuplicados(t *testing.T) {
	t.Parallel()

	validas := []CabeceraCargaDirecta{
		{Nombre: "content-type", Valor: "application/pdf"},
		{Nombre: "x-amz-checksum-sha256", Valor: strings.Repeat("a", 64)},
	}
	if !CabecerasCargaDirectaValidas(validas) || !CabecerasCargaDirectaValidas(nil) {
		t.Fatal("se rechazaron cabeceras permitidas")
	}
	casos := map[string][]CabeceraCargaDirecta{
		"fuera de lista":     {{Nombre: "authorization", Valor: "opaco"}},
		"nombre no canonico": {{Nombre: "Content-Type", Valor: "application/pdf"}},
		"valor con control":  {{Nombre: "content-type", Valor: "application/pdf\r\nX: uno"}},
		"duplicada": {
			{Nombre: "content-type", Valor: "application/pdf"},
			{Nombre: "content-type", Valor: "text/plain"},
		},
	}
	demasiadas := make([]CabeceraCargaDirecta, MaximoCabecerasCargaDirecta+1)
	for indice := range demasiadas {
		demasiadas[indice] = CabeceraCargaDirecta{Nombre: "content-type", Valor: "application/pdf"}
	}
	casos["sobre limite"] = demasiadas
	for nombre, cabeceras := range casos {
		if CabecerasCargaDirectaValidas(cabeceras) {
			t.Errorf("%s: se aceptaron cabeceras no validas", nombre)
		}
	}
}

func TestInstruccionesCargaDirectaValidanConcesionCompleta(t *testing.T) {
	t.Parallel()

	emitida := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	base := DatosInstruccionesCargaDirecta{
		ConectorID: "almacen_s3_corporativo", SesionRef: "sesion:carga:abcdefghijkl",
		Metodo: MetodoCargaDirectaPUT, Destino: "https://objetos.example.test/carga?firma=opaca",
		Cabeceras: []CabeceraCargaDirecta{{Nombre: "content-type", Valor: "application/pdf"}},
		EmitidaEn: emitida, ExpiraEn: emitida.Add(DuracionMaximaInstruccionesCargaDirecta), TamanoMaximo: 1024,
	}
	if !InstruccionesCargaDirectaValidas(base) {
		t.Fatal("se rechazo una concesion canonica en el limite inclusivo")
	}

	casos := map[string]func(*DatosInstruccionesCargaDirecta){
		"metodo": func(d *DatosInstruccionesCargaDirecta) { d.Metodo = "PATCH" },
		"duracion": func(d *DatosInstruccionesCargaDirecta) {
			d.ExpiraEn = d.EmitidaEn.Add(DuracionMaximaInstruccionesCargaDirecta + time.Nanosecond)
		},
		"zona horaria": func(d *DatosInstruccionesCargaDirecta) { d.EmitidaEn = d.EmitidaEn.In(time.FixedZone("X", 0)) },
		"tamano":       func(d *DatosInstruccionesCargaDirecta) { d.TamanoMaximo = 0 },
		"destino":      func(d *DatosInstruccionesCargaDirecta) { d.Destino = "http://objetos.example.test" },
	}
	for nombre, alterar := range casos {
		copia := base
		copia.Cabeceras = append([]CabeceraCargaDirecta(nil), base.Cabeceras...)
		alterar(&copia)
		if InstruccionesCargaDirectaValidas(copia) {
			t.Errorf("%s: se acepto una concesion no valida", nombre)
		}
	}
}

func TestCapacidadesSatisfacenRequisitosYOrigenes(t *testing.T) {
	t.Parallel()

	capacidades := Capacidades{
		ConectorID: "almacen_s3_corporativo", EscrituraEnFlujo: true, ReferenciasOpacas: true,
		IntegridadSHA256: true, CargaDirectaTemporal: true, TamanoMaximoObjeto: 4096,
		OrigenesCargaDirecta: []string{"https://objetos.example.test"},
	}
	requisitos := Requisitos{
		EscrituraEnFlujo: true, ReferenciasOpacas: true, IntegridadSHA256: true,
		CargaDirectaTemporal: true, TamanoMinimoObjeto: 1024,
	}
	if !CapacidadesSatisfacen(capacidades, requisitos) {
		t.Fatal("se rechazaron capacidades suficientes")
	}
	sinIntegridad := capacidades
	sinIntegridad.IntegridadSHA256 = false
	if CapacidadesSatisfacen(sinIntegridad, requisitos) {
		t.Fatal("se aceptaron capacidades sin una garantia requerida")
	}
	origenNoCanonico := capacidades
	origenNoCanonico.OrigenesCargaDirecta = []string{"https://objetos.example.test/ruta"}
	if CapacidadesSatisfacen(origenNoCanonico, requisitos) {
		t.Fatal("se acepto carga directa con un origen no canonico")
	}
	requisitos.TamanoMinimoObjeto = -1
	if CapacidadesSatisfacen(capacidades, requisitos) {
		t.Fatal("se acepto un requisito de tamano negativo")
	}
}

func TestHuellaPreparacionCargaDirectaEsEstableYNoAmbigua(t *testing.T) {
	t.Parallel()

	datos := DatosHuellaPreparacionCargaDirecta{
		Esquema: "e", OperacionRef: "o", CorrelacionRef: "c", AutorizacionRef: "a",
		Finalidad: "f", Clasificacion: "l", AccionNegocio: "n", AccionTecnica: "t",
		CargaRef: "g", SujetoSeudonimoHMAC: "s", RecursoRef: "r", ModuloID: "m",
		HuellaSolicitudHMAC: "h", EfectoRef: "x", HuellaPlanEfectoSHA256: "p",
		PasoRef: "q", HuellaDecisionSHA256: "d", ClaveIdempotencia: "i",
		MIME: "application/pdf", Tamano: 42, HuellaSHA256: "z",
		ExpiraEn: time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC),
	}
	const esperada = "e9c053d6df967efef0bc20c9167eb649972fa6875da638e9b84dd14985d23fe6"
	if obtenida := HuellaPreparacionCargaDirecta(datos); obtenida != esperada {
		t.Fatalf("huella canonica distinta: %s", obtenida)
	}
	alterada := datos
	alterada.OperacionRef, alterada.CorrelacionRef = "oc", ""
	if HuellaPreparacionCargaDirecta(alterada) == esperada {
		t.Fatal("el prefijo de longitud no separo una concatenacion ambigua")
	}
}

func TestAccionesAlmacenMantienenListasCerradasPorUso(t *testing.T) {
	t.Parallel()

	acciones := []string{
		AccionEscribir, AccionLeer, AccionPrepararCargaDirecta, AccionConfirmarCargaDirecta,
		AccionAbandonarCargaDirecta, AccionPromover, AccionAplicarRetencion, AccionInmovilizar,
		AccionLevantarInmovilizacion, AccionEliminar, AccionAnalizarContenido,
	}
	for _, accion := range acciones {
		if !AccionOperacionValida(accion) {
			t.Errorf("se rechazo la accion declarada %q", accion)
		}
	}
	if AccionOperacionValida("*") || AccionOperacionValida("accion_futura") {
		t.Fatal("se acepto una accion no declarada")
	}
	if !AccionCreaObjeto(AccionEscribir) || !AccionCreaObjeto(AccionConfirmarCargaDirecta) ||
		!AccionCreaObjeto(AccionPromover) || AccionCreaObjeto(AccionLeer) {
		t.Fatal("la clasificacion de acciones creadoras cambio")
	}
	if !AccionResultadoValida(AccionAplicarRetencion) || AccionResultadoValida(AccionEliminar) {
		t.Fatal("la lista de acciones con resultado objeto cambio")
	}
}

func TestLigaduraExactaNoAdmiteSubconjuntosNiReordenacion(t *testing.T) {
	t.Parallel()

	esperada := []string{"operacion:1", "correlacion:1", "autorizacion:1"}
	if !LigaduraExacta(append([]string(nil), esperada...), esperada) {
		t.Fatal("se rechazo una ligadura exacta")
	}
	if LigaduraExacta(esperada[:2], esperada) ||
		LigaduraExacta([]string{"correlacion:1", "operacion:1", "autorizacion:1"}, esperada) ||
		LigaduraExacta([]string{"operacion:1", "correlacion:2", "autorizacion:1"}, esperada) {
		t.Fatal("se acepto una ligadura parcial, reordenada o alterada")
	}
}

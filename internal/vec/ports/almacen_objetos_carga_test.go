package ports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

var instanteCargaDirecta = time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

type emisorReciboCargaDirectaPrueba struct {
	emisiones int
}

type emisorReciboCargaDirectaNuloPrueba struct{}
type selladorVinculoSesionNuloPrueba struct{}
type gestorCargaDirectaNuloPrueba struct{}

func (*emisorReciboCargaDirectaNuloPrueba) EmitirReciboCargaDirecta(
	context.Context, SolicitudEmitirReciboCargaDirecta,
) (ReciboCargaDirecta, error) {
	panic("no debe invocarse una dependencia nula tipada")
}

func (*selladorVinculoSesionNuloPrueba) SellarVinculoSesionCarga(context.Context, string) (string, error) {
	panic("no debe invocarse una dependencia nula tipada")
}

func (*gestorCargaDirectaNuloPrueba) PrepararCargaDirecta(
	context.Context, SolicitudPrepararCargaDirecta,
) (InstruccionesCargaDirecta, error) {
	panic("no debe invocarse una dependencia nula tipada")
}

func (*gestorCargaDirectaNuloPrueba) ConfirmarCargaDirecta(
	context.Context, SolicitudConfirmarCargaDirecta,
) (ResultadoOperacionObjeto, error) {
	panic("no debe invocarse una dependencia nula tipada")
}

func (*gestorCargaDirectaNuloPrueba) AbandonarCargaDirecta(
	context.Context, ContextoOperacionAlmacen, string,
) error {
	panic("no debe invocarse una dependencia nula tipada")
}

type verificadorAtestacionReciboPrueba struct {
	contexto   ContextoOperacionAlmacen
	sesionRef  string
	atestacion string
}

type verificadorAtestacionReciboNuloPrueba struct{}

func (*verificadorAtestacionReciboNuloPrueba) VerificarAtestacionConsumoReciboCargaDirecta(
	context.Context,
	ContextoOperacionAlmacen,
	string,
	ComprobanteConsumoReciboCargaDirecta,
) error {
	// Si la fabrica invocase este metodo sobre el puntero nulo, una
	// implementacion permisiva podria convertir una dependencia ausente en
	// concesion. La fabrica debe rechazarla antes de la llamada.
	return nil
}

func (v verificadorAtestacionReciboPrueba) VerificarAtestacionConsumoReciboCargaDirecta(
	_ context.Context,
	contexto ContextoOperacionAlmacen,
	sesionRef string,
	comprobante ComprobanteConsumoReciboCargaDirecta,
) error {
	_, _, _, _, _, _, _, _, _, _, atestacion, err := comprobante.RevelarParaVerificacion()
	if err != nil || contexto != v.contexto || sesionRef != v.sesionRef || atestacion != v.atestacion {
		return ErrAtestacionReciboCargaDirectaNoValida
	}
	return nil
}

func (e *emisorReciboCargaDirectaPrueba) EmitirReciboCargaDirecta(
	_ context.Context,
	solicitud SolicitudEmitirReciboCargaDirecta,
) (ReciboCargaDirecta, error) {
	_, sesion, _, vinculo, err := solicitud.RevelarParaEmision()
	if err != nil {
		return ReciboCargaDirecta{}, err
	}
	e.emisiones++
	return NuevoReciboCargaDirecta("recibo:mac:v1:" + sesion + ":" + vinculo)
}

func contextoAlmacenVinculadoPrueba(
	t *testing.T,
	accion string,
	objetos ...ReferenciaObjetoAlmacen,
) ContextoOperacionAlmacen {
	t.Helper()
	var accionNegocio string
	var campos []string
	requiereObjeto := false
	var fabrica func() (ContextoOperacionAlmacen, error)

	decision, recurso, vinculos, instante := autorizacionAlmacenPrueba(
		t, AccionNegocioPrepararCargaDocumental,
		[]string{"clasificacion", "contenido", "huella_sha256", "mime", "tamano"}, false,
	)
	configurar := func(negocio string, permitidos []string, conObjeto bool) {
		accionNegocio, campos, requiereObjeto = negocio, permitidos, conObjeto
		decision, recurso, vinculos, instante = autorizacionAlmacenPrueba(t, negocio, permitidos, conObjeto)
	}
	switch accion {
	case AccionAlmacenPrepararCargaDirecta:
		fabrica = func() (ContextoOperacionAlmacen, error) {
			return NuevoContextoPrepararCargaDirectaAlmacen(decision, recurso, vinculos, instante)
		}
	case AccionAlmacenAbandonarCargaDirecta:
		fabrica = func() (ContextoOperacionAlmacen, error) {
			preparar, err := NuevoContextoPrepararCargaDirectaAlmacen(decision, recurso, vinculos, instante)
			if err != nil {
				return ContextoOperacionAlmacen{}, err
			}
			return preparar.DerivarPaso(PasoAlmacenAbandonarCargaDirecta)
		}
	case AccionAlmacenConfirmarCargaDirecta:
		configurar(AccionNegocioConfirmarCargaDocumental, []string{"contenido_cuarentena", "estado"}, false)
		fabrica = func() (ContextoOperacionAlmacen, error) {
			return NuevoContextoConfirmarCargaDirectaAlmacen(decision, recurso, vinculos, instante)
		}
	case AccionAlmacenEscribir:
		configurar(AccionNegocioCustodiarDecisionBaremacion,
			[]string{"documento_custodiado", "evidencia_custodia"}, false)
		fabrica = func() (ContextoOperacionAlmacen, error) {
			return NuevoContextoCustodiarDecisionBaremacionAlmacen(decision, recurso, vinculos, instante)
		}
	case AccionAlmacenPromover:
		configurar(AccionNegocioPromoverCargaDocumental, []string{"contenido_admitido", "estado"}, true)
		fabrica = func() (ContextoOperacionAlmacen, error) {
			return NuevoContextoPromoverCargaDocumentalAlmacen(decision, recurso, vinculos, instante)
		}
	case AccionAlmacenAplicarRetencion:
		configurar(AccionNegocioRetenerDocumentoFirmado,
			[]string{"documento_firmado.retencion", "evidencia_retencion"}, true)
		fabrica = func() (ContextoOperacionAlmacen, error) {
			return NuevoContextoRetenerDocumentoFirmadoAlmacen(decision, recurso, vinculos, instante)
		}
	default:
		t.Fatalf("la prueba intento acunar una capacidad no declarada: %s", accion)
	}
	_ = accionNegocio
	_ = campos
	if requiereObjeto && len(objetos) == 1 {
		vinculos.ObjetoVinculado = objetos[0]
		recurso.Atributos[AtributoAlmacenObjetoRef] = objetos[0].Referencia
		recurso.Atributos[AtributoAlmacenObjetoVersion] = objetos[0].Version
		huella, err := recurso.HuellaContextoAutorizacionSHA256()
		if err != nil {
			t.Fatalf("huella de recurso de prueba: %v", err)
		}
		decision.ContextoRecursoHuellaSHA256 = huella
	}
	contexto, err := fabrica()
	if err != nil {
		t.Fatalf("crear contexto de almacen de prueba: %v", err)
	}
	return contexto
}

func confirmacionCargaDirectaPrueba(
	t *testing.T,
	contexto ContextoOperacionAlmacen,
	sesionRef string,
) SolicitudConfirmarCargaDirecta {
	t.Helper()
	recibo, err := NuevoReciboCargaDirecta("recibo:mac:v1:0123456789abcdefghijkl")
	if err != nil {
		t.Fatal(err)
	}
	consumo := SolicitudConsumirReciboCargaDirecta{
		Contexto: contexto, SesionRef: sesionRef, Recibo: recibo,
		ValidaHasta: instanteCargaDirecta.Add(4 * time.Minute),
	}
	resultado := ResultadoConsumoReciboCargaDirecta{
		IndiceHMAC:               "hmac-sha256:indice_v1:" + strings.Repeat("3", 64),
		GrupoHMAC:                "hmac-sha256:grupo_v1:" + strings.Repeat("4", 64),
		VinculoHMAC:              "hmac-sha256:vinculo_v1:" + strings.Repeat("5", 64),
		EvidenciaConsumoRef:      "evidencia:consumo:recibo:uno",
		IntencionConfirmacionRef: "confirmacion:intencion:uno",
		HuellaIntencionHMAC:      "hmac-sha256:intencion_v1:" + strings.Repeat("7", 64),
		RegistradoEn:             instanteCargaDirecta,
		ConsumidoEn:              instanteCargaDirecta.Add(20 * time.Second),
		ExpiraEn:                 instanteCargaDirecta.Add(5 * time.Minute),
	}
	atestacion := "hmac-sha256:atestacion_v1:" + strings.Repeat("6", 64)
	comprobante, err := NuevoComprobanteConsumoReciboCargaDirecta(
		consumo, resultado, atestacion,
	)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion, err := NuevaSolicitudConfirmarCargaDirecta(
		context.Background(), contexto, sesionRef, comprobante,
		verificadorAtestacionReciboPrueba{contexto: contexto, sesionRef: sesionRef, atestacion: atestacion},
	)
	if err != nil {
		t.Fatal(err)
	}
	return confirmacion
}

func TestInstruccionesCargaDirectaSonOpacasAcotadasYCopianCabeceras(t *testing.T) {
	const secreto = "firma-super-secreta"
	cabeceras := []CabeceraCargaDirecta{
		{Nombre: "content-type", Valor: "application/pdf"},
		{Nombre: "x-checksum-sha256", Valor: strings.Repeat("a", 64)},
	}
	instrucciones, err := NuevasInstruccionesCargaDirecta(
		"almacen_s3_corporativo",
		"sesion:carga:0123456789abcdefghijkl",
		MetodoCargaDirectaPUT,
		"https://objetos.example.test/cuarentena/objeto-opaco?firma="+secreto,
		cabeceras,
		instanteCargaDirecta,
		instanteCargaDirecta.Add(5*time.Minute),
		1024,
	)
	if err != nil {
		t.Fatalf("crear instrucciones: %v", err)
	}
	cabeceras[0].Valor = "text/plain"
	if err := instrucciones.ValidarContra(CapacidadesAlmacenObjetos{
		ConectorID: "almacen_s3_corporativo", CargaDirectaTemporal: true,
		TamanoMaximoObjeto: 2048, OrigenesCargaDirecta: []string{"https://objetos.example.test"},
	}); err != nil {
		t.Fatalf("validar contra capacidades: %v", err)
	}

	sesion, metodo, destino, reveladas, expira, maximo, err := instrucciones.RevelarParaEntrega()
	if err != nil || sesion == "" || metodo != MetodoCargaDirectaPUT || !strings.Contains(destino, secreto) ||
		len(reveladas) != 2 || reveladas[0].Valor != "application/pdf" ||
		!expira.Equal(instanteCargaDirecta.Add(5*time.Minute)) || maximo != 1024 {
		t.Fatalf("revelacion inesperada: %q %q %q %#v %v %d %v", sesion, metodo, destino, reveladas, expira, maximo, err)
	}
	reveladas[0].Valor = "alterado"
	_, _, _, segundaLectura, _, _, _ := instrucciones.RevelarParaEntrega()
	if segundaLectura[0].Valor != "application/pdf" {
		t.Fatal("RevelarParaEntrega comparte la memoria interna")
	}

	for _, formato := range []string{"%s", "%v", "%+v", "%#v"} {
		salida := fmt.Sprintf(formato, instrucciones)
		if strings.Contains(salida, secreto) || !strings.Contains(salida, "CONFIDENCIALES") {
			t.Fatalf("formato %s filtra la concesion: %q", formato, salida)
		}
	}
	if _, err := json.Marshal(instrucciones); !errors.Is(err, ErrSerializacionCargaDirectaProhibida) {
		t.Fatalf("JSON debe fallar cerrado: %v", err)
	}
	if _, err := instrucciones.MarshalText(); !errors.Is(err, ErrSerializacionCargaDirectaProhibida) {
		t.Fatalf("texto debe fallar cerrado: %v", err)
	}
}

func TestInstruccionesCargaDirectaRechazanConcesionesPeligrosas(t *testing.T) {
	base := func() (string, MetodoCargaDirecta, []CabeceraCargaDirecta, time.Time, time.Time, int64) {
		return "https://objetos.example.test/carga?firma=opaca", MetodoCargaDirectaPUT,
			[]CabeceraCargaDirecta{{Nombre: "content-type", Valor: "application/pdf"}},
			instanteCargaDirecta, instanteCargaDirecta.Add(5 * time.Minute), 1024
	}
	pruebas := []struct {
		nombre string
		muta   func(*string, *MetodoCargaDirecta, *[]CabeceraCargaDirecta, *time.Time, *time.Time, *int64)
	}{
		{"http", func(destino *string, _ *MetodoCargaDirecta, _ *[]CabeceraCargaDirecta, _, _ *time.Time, _ *int64) {
			*destino = "http://objetos.example.test/carga"
		}},
		{"usuario en URL", func(destino *string, _ *MetodoCargaDirecta, _ *[]CabeceraCargaDirecta, _, _ *time.Time, _ *int64) {
			*destino = "https://usuario:clave@objetos.example.test/carga"
		}},
		{"fragmento", func(destino *string, _ *MetodoCargaDirecta, _ *[]CabeceraCargaDirecta, _, _ *time.Time, _ *int64) {
			*destino += "#secreto"
		}},
		{"metodo libre", func(_ *string, metodo *MetodoCargaDirecta, _ *[]CabeceraCargaDirecta, _, _ *time.Time, _ *int64) {
			*metodo = "PATCH"
		}},
		{"caducidad larga", func(_ *string, _ *MetodoCargaDirecta, _ *[]CabeceraCargaDirecta, _, expira *time.Time, _ *int64) {
			*expira = instanteCargaDirecta.Add(11 * time.Minute)
		}},
		{"emision no UTC", func(_ *string, _ *MetodoCargaDirecta, _ *[]CabeceraCargaDirecta, emitida, _ *time.Time, _ *int64) {
			*emitida = (*emitida).In(time.FixedZone("UTC+1", 60*60))
		}},
		{"caducidad no UTC", func(_ *string, _ *MetodoCargaDirecta, _ *[]CabeceraCargaDirecta, _, expira *time.Time, _ *int64) {
			*expira = (*expira).In(time.FixedZone("UTC+1", 60*60))
		}},
		{"sin tamano", func(_ *string, _ *MetodoCargaDirecta, _ *[]CabeceraCargaDirecta, _, _ *time.Time, maximo *int64) {
			*maximo = 0
		}},
		{"authorization", func(_ *string, _ *MetodoCargaDirecta, cabeceras *[]CabeceraCargaDirecta, _, _ *time.Time, _ *int64) {
			*cabeceras = []CabeceraCargaDirecta{{Nombre: "Authorization", Valor: "Bearer general"}}
		}},
		{"cookie", func(_ *string, _ *MetodoCargaDirecta, cabeceras *[]CabeceraCargaDirecta, _, _ *time.Time, _ *int64) {
			*cabeceras = []CabeceraCargaDirecta{{Nombre: "cookie", Valor: "sesion=general"}}
		}},
		{"cabecera repetida", func(_ *string, _ *MetodoCargaDirecta, cabeceras *[]CabeceraCargaDirecta, _, _ *time.Time, _ *int64) {
			*cabeceras = []CabeceraCargaDirecta{{Nombre: "content-type", Valor: "application/pdf"}, {Nombre: "Content-Type", Valor: "application/pdf"}}
		}},
		{"inyeccion de cabecera", func(_ *string, _ *MetodoCargaDirecta, cabeceras *[]CabeceraCargaDirecta, _, _ *time.Time, _ *int64) {
			*cabeceras = []CabeceraCargaDirecta{{Nombre: "x-checksum", Valor: "a\r\nOtra: valor"}}
		}},
		{"host", func(_ *string, _ *MetodoCargaDirecta, cabeceras *[]CabeceraCargaDirecta, _, _ *time.Time, _ *int64) {
			*cabeceras = []CabeceraCargaDirecta{{Nombre: "Host", Valor: "otro.example.test"}}
		}},
		{"content-length", func(_ *string, _ *MetodoCargaDirecta, cabeceras *[]CabeceraCargaDirecta, _, _ *time.Time, _ *int64) {
			*cabeceras = []CabeceraCargaDirecta{{Nombre: "Content-Length", Valor: "1"}}
		}},
		{"transfer-encoding", func(_ *string, _ *MetodoCargaDirecta, cabeceras *[]CabeceraCargaDirecta, _, _ *time.Time, _ *int64) {
			*cabeceras = []CabeceraCargaDirecta{{Nombre: "Transfer-Encoding", Valor: "chunked"}}
		}},
		{"connection", func(_ *string, _ *MetodoCargaDirecta, cabeceras *[]CabeceraCargaDirecta, _, _ *time.Time, _ *int64) {
			*cabeceras = []CabeceraCargaDirecta{{Nombre: "Connection", Valor: "keep-alive"}}
		}},
		{"forwarded", func(_ *string, _ *MetodoCargaDirecta, cabeceras *[]CabeceraCargaDirecta, _, _ *time.Time, _ *int64) {
			*cabeceras = []CabeceraCargaDirecta{{Nombre: "Forwarded", Valor: "for=192.0.2.1"}}
		}},
		{"x-forwarded-for", func(_ *string, _ *MetodoCargaDirecta, cabeceras *[]CabeceraCargaDirecta, _, _ *time.Time, _ *int64) {
			*cabeceras = []CabeceraCargaDirecta{{Nombre: "X-Forwarded-For", Valor: "192.0.2.1"}}
		}},
		{"proxy-connection", func(_ *string, _ *MetodoCargaDirecta, cabeceras *[]CabeceraCargaDirecta, _, _ *time.Time, _ *int64) {
			*cabeceras = []CabeceraCargaDirecta{{Nombre: "Proxy-Connection", Valor: "keep-alive"}}
		}},
		{"cabecera futura no declarada", func(_ *string, _ *MetodoCargaDirecta, cabeceras *[]CabeceraCargaDirecta, _, _ *time.Time, _ *int64) {
			*cabeceras = []CabeceraCargaDirecta{{Nombre: "X-Custom-Upload", Valor: "valor"}}
		}},
		{"valor de cabecera no canonico", func(_ *string, _ *MetodoCargaDirecta, cabeceras *[]CabeceraCargaDirecta, _, _ *time.Time, _ *int64) {
			*cabeceras = []CabeceraCargaDirecta{{Nombre: "content-type", Valor: "application/pdf "}}
		}},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			destino, metodo, cabeceras, emitida, expira, maximo := base()
			prueba.muta(&destino, &metodo, &cabeceras, &emitida, &expira, &maximo)
			_, err := NuevasInstruccionesCargaDirecta(
				"almacen_s3_corporativo", "sesion:carga:0123456789abcdefghijkl", metodo,
				destino, cabeceras, emitida, expira, maximo,
			)
			if !errors.Is(err, ErrInstruccionesCargaDirectaNoValidas) {
				t.Fatalf("se esperaba rechazo, recibido %v", err)
			}
		})
	}
}

func TestCargaDirectaExigeOrigenPublicadoYSolicitudesCompletas(t *testing.T) {
	instrucciones, err := NuevasInstruccionesCargaDirecta(
		"almacen_s3_corporativo", "sesion:carga:0123456789abcdefghijkl", MetodoCargaDirectaPOST,
		"https://objetos.example.test/carga", nil, instanteCargaDirecta,
		instanteCargaDirecta.Add(time.Minute), 1024,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, capacidades := range []CapacidadesAlmacenObjetos{
		{ConectorID: "otro", CargaDirectaTemporal: true, TamanoMaximoObjeto: 2048, OrigenesCargaDirecta: []string{"https://objetos.example.test"}},
		{ConectorID: "almacen_s3_corporativo", CargaDirectaTemporal: false, TamanoMaximoObjeto: 2048, OrigenesCargaDirecta: []string{"https://objetos.example.test"}},
		{ConectorID: "almacen_s3_corporativo", CargaDirectaTemporal: true, TamanoMaximoObjeto: 512, OrigenesCargaDirecta: []string{"https://objetos.example.test"}},
		{ConectorID: "almacen_s3_corporativo", CargaDirectaTemporal: true, TamanoMaximoObjeto: 2048, OrigenesCargaDirecta: []string{"https://otro.example.test"}},
		// Una coincidencia valida no sanea otra entrada no canonica: toda la
		// lista declarada por el conector debe superar la lista positiva.
		{ConectorID: "almacen_s3_corporativo", CargaDirectaTemporal: true, TamanoMaximoObjeto: 2048, OrigenesCargaDirecta: []string{"https://objetos.example.test", " https://otro.example.test"}},
		{ConectorID: "almacen_s3_corporativo", CargaDirectaTemporal: true, TamanoMaximoObjeto: 2048, OrigenesCargaDirecta: []string{"https://objetos.example.test", "https://otro.example.test/"}},
		{ConectorID: "almacen_s3_corporativo", CargaDirectaTemporal: true, TamanoMaximoObjeto: 2048, OrigenesCargaDirecta: []string{"https://objetos.example.test", "https://OBJETOS.example.test"}},
		{ConectorID: "almacen_s3_corporativo", CargaDirectaTemporal: true, TamanoMaximoObjeto: 2048, OrigenesCargaDirecta: []string{"https://objetos.example.test", "https://objetos.example.test"}},
	} {
		if !errors.Is(instrucciones.ValidarContra(capacidades), ErrInstruccionesCargaDirectaNoValidas) {
			t.Fatalf("capacidades peligrosas aceptadas: %#v", capacidades)
		}
	}

	contexto := contextoAlmacenVinculadoPrueba(t, AccionAlmacenPrepararCargaDirecta)
	preparar := SolicitudPrepararCargaDirecta{
		Contexto: contexto, ClaveIdempotencia: "idempotencia:carga:uno", MIME: "application/pdf",
		Tamano: 1024, HuellaSHA256: strings.Repeat("a", 64), ExpiraEn: instanteCargaDirecta.Add(time.Minute),
	}
	if err := preparar.Validar(); err != nil {
		t.Fatalf("preparacion valida: %v", err)
	}
	capacidadesValidas := CapacidadesAlmacenObjetos{
		ConectorID: "almacen_s3_corporativo", CargaDirectaTemporal: true,
		TamanoMaximoObjeto: 2048, OrigenesCargaDirecta: []string{"https://objetos.example.test"},
	}
	vinculadas, err := NuevasInstruccionesCargaDirectaParaSolicitud(
		preparar, capacidadesValidas.ConectorID, "sesion:carga:0123456789abcdefghijkl",
		MetodoCargaDirectaPUT, "https://objetos.example.test/carga?firma=opaca", nil, instanteCargaDirecta,
	)
	if err != nil || vinculadas.ValidarPara(preparar, capacidadesValidas) != nil {
		t.Fatalf("las instrucciones vinculadas deben corresponder: %v", err)
	}
	emisor := &emisorReciboCargaDirectaPrueba{}
	recibo, err := vinculadas.EmitirReciboConfirmacion(
		context.Background(), preparar, capacidadesValidas, emisor,
	)
	if err != nil || !recibo.Valido() || emisor.emisiones != 1 {
		t.Fatalf("recibo ligado a instrucciones: %v", err)
	}
	var emisorNulo *emisorReciboCargaDirectaNuloPrueba
	if _, err := vinculadas.EmitirReciboConfirmacion(
		context.Background(), preparar, capacidadesValidas, emisorNulo,
	); !errors.Is(err, ErrReciboCargaDirectaNoDisponible) {
		t.Fatalf("emisor nulo tipado aceptado: %v", err)
	}
	var selladorNulo *selladorVinculoSesionNuloPrueba
	if _, err := vinculadas.SellarVinculoSesion(context.Background(), selladorNulo); !errors.Is(err, ErrSelladoIdempotenciaCargaNoDisponible) {
		t.Fatalf("sellador nulo tipado aceptado: %v", err)
	}
	contextoAbandono, err := contexto.DerivarPaso(PasoAlmacenAbandonarCargaDirecta)
	if err != nil {
		t.Fatalf("derivar abandono declarado: %v", err)
	}
	var gestorNulo *gestorCargaDirectaNuloPrueba
	if err := vinculadas.Abandonar(context.Background(), gestorNulo, contextoAbandono); !errors.Is(err, ErrSesionCargaDirectaNoValida) {
		t.Fatalf("gestor nulo tipado aceptado: %v", err)
	}
	alterada := preparar
	alterada.HuellaSHA256 = strings.Repeat("b", 64)
	if !errors.Is(vinculadas.ValidarPara(alterada, capacidadesValidas), ErrInstruccionesCargaDirectaNoValidas) {
		t.Fatal("se aceptaron instrucciones emitidas para otra huella")
	}
	if _, err := vinculadas.EmitirReciboConfirmacion(
		context.Background(), alterada, capacidadesValidas, emisor,
	); !errors.Is(err, ErrReciboCargaDirectaNoDisponible) || emisor.emisiones != 1 {
		t.Fatal("se emitio un recibo para una solicitud cruzada")
	}
	sinVinculo, err := NuevasInstruccionesCargaDirecta(
		capacidadesValidas.ConectorID, "sesion:carga:sin-vinculo-abcdefghijkl", MetodoCargaDirectaPUT,
		"https://objetos.example.test/carga?firma=opaca", nil, instanteCargaDirecta,
		preparar.ExpiraEn, preparar.Tamano,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(sinVinculo.ValidarPara(preparar, capacidadesValidas), ErrInstruccionesCargaDirectaNoValidas) {
		t.Fatal("el caso de uso acepto una concesion no vinculada a su solicitud")
	}
	contextoConfirmacion := contextoAlmacenVinculadoPrueba(t, AccionAlmacenConfirmarCargaDirecta)
	confirmar := confirmacionCargaDirectaPrueba(
		t, contextoConfirmacion, "sesion:carga:0123456789abcdefghijkl",
	)
	if err := confirmar.Validar(); err != nil {
		t.Fatalf("confirmacion valida: %v", err)
	}
	preparar.HuellaSHA256 = "declarada-por-cliente"
	if !errors.Is(preparar.Validar(), ErrSolicitudAlmacenInvalida) {
		t.Fatal("se acepto una huella no canonica")
	}
	if !errors.Is((SolicitudConfirmarCargaDirecta{}).Validar(), ErrSolicitudAlmacenInvalida) {
		t.Fatal("se acepto confirmar sin sesion")
	}
	preparacionNoUTC := preparar
	preparacionNoUTC.HuellaSHA256 = strings.Repeat("a", 64)
	preparacionNoUTC.ExpiraEn = preparacionNoUTC.ExpiraEn.In(time.FixedZone("UTC+1", 60*60))
	if !errors.Is(preparacionNoUTC.Validar(), ErrSolicitudAlmacenInvalida) {
		t.Fatal("una caducidad no canonica fue convertida a UTC")
	}

	capacidades := CapacidadesAlmacenObjetos{
		ConectorID: "almacen_s3_corporativo", CargaDirectaTemporal: true,
		TamanoMaximoObjeto: 2048, OrigenesCargaDirecta: []string{"https://objetos.example.test/ruta"},
	}
	requisitos := RequisitosAlmacenObjetos{CargaDirectaTemporal: true, TamanoMinimoObjeto: 1}
	if !errors.Is(VerificarCapacidadesAlmacen(capacidades, requisitos), ErrCapacidadAlmacenNoDisponible) {
		t.Fatal("una capacidad con origen que incluye ruta fue aceptada")
	}
}

func TestContextoAlmacenDeniegaTodaAccionNoDeclarada(t *testing.T) {
	valido := contextoAlmacenVinculadoPrueba(t, AccionAlmacenEscribir)
	if err := valido.validarParaPaso(AccionAlmacenEscribir); err != nil {
		t.Fatalf("contexto valido: %v", err)
	}
	if err := valido.validarParaPaso(""); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("accion ausente aceptada: %v", err)
	}
	if err := valido.validarParaPaso("listar_todo"); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("accion libre aceptada: %v", err)
	}
	if err := valido.validarParaPaso(AccionAlmacenLeer); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatal("una autorizacion de escritura se reutilizo para leer")
	}
	var ausente ContextoOperacionAlmacen
	if err := ausente.validarParaPaso(AccionAlmacenEscribir); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("capacidad ausente aceptada: %v", err)
	}
	proyeccion, err := valido.Proyeccion()
	if err != nil {
		t.Fatal(err)
	}
	proyeccion.CargaRef = "carga:otra"
	segunda, err := valido.Proyeccion()
	if err != nil || segunda.CargaRef == proyeccion.CargaRef {
		t.Fatal("la proyeccion permitio mutar la capacidad")
	}
}

func TestSolicitudSeudonimizarNoExponeIdentificadorInterno(t *testing.T) {
	const sujeto = "persona:interna:0123456789abcdef"
	solicitud, err := NuevaSolicitudSeudonimizarSujetoAlmacen(sujeto, "almacen_documental")
	if err != nil {
		t.Fatal(err)
	}
	for _, formato := range []string{"%s", "%v", "%+v", "%#v"} {
		salida := fmt.Sprintf(formato, solicitud)
		if strings.Contains(salida, sujeto) || !strings.Contains(salida, "CONFIDENCIAL") {
			t.Fatalf("el formato %s filtro el sujeto: %q", formato, salida)
		}
	}
	revelado, ambito, err := solicitud.RevelarParaSellado()
	if err != nil || revelado != sujeto || ambito != "almacen_documental" {
		t.Fatalf("revelacion deliberada invalida: %q %q %v", revelado, ambito, err)
	}
}

func TestReciboCargaDirectaEsOpacoYNoPuedeCruzarse(t *testing.T) {
	const secreto = "recibo:mac:v1:secreto-0123456789abcdefghijkl"
	recibo, err := NuevoReciboCargaDirecta(secreto)
	if err != nil {
		t.Fatal(err)
	}
	for _, formato := range []string{"%s", "%v", "%+v", "%#v"} {
		salida := fmt.Sprintf(formato, recibo)
		if strings.Contains(salida, secreto) || !strings.Contains(salida, "CONFIDENCIAL") {
			t.Fatalf("el formato %s filtro el recibo: %q", formato, salida)
		}
	}
	if _, err := json.Marshal(recibo); !errors.Is(err, ErrSerializacionReciboCargaProhibida) {
		t.Fatalf("JSON debe fallar cerrado: %v", err)
	}

	contexto := contextoAlmacenVinculadoPrueba(t, AccionAlmacenConfirmarCargaDirecta)
	consumo := SolicitudConsumirReciboCargaDirecta{
		Contexto: contexto, SesionRef: "sesion:carga:0123456789abcdefghijkl", Recibo: recibo,
		ValidaHasta: instanteCargaDirecta.Add(4 * time.Minute),
	}
	confirmacion := confirmacionCargaDirectaPrueba(t, contexto, consumo.SesionRef)
	comprobante := confirmacion.comprobante
	_, _, _, _, _, _, _, _, _, _, atestacion, err := comprobante.RevelarParaVerificacion()
	if err != nil {
		t.Fatal(err)
	}
	verificador := verificadorAtestacionReciboPrueba{
		contexto: contexto, sesionRef: consumo.SesionRef, atestacion: atestacion,
	}
	var verificadorNulo *verificadorAtestacionReciboNuloPrueba
	if _, err := NuevaSolicitudConfirmarCargaDirecta(
		context.Background(), contexto, consumo.SesionRef, comprobante, verificadorNulo,
	); !errors.Is(err, ErrSolicitudAlmacenInvalida) {
		t.Fatalf("verificador nulo tipado aceptado: %v", err)
	}
	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := NuevaSolicitudConfirmarCargaDirecta(
		ctxCancelado, contexto, consumo.SesionRef, comprobante, verificador,
	); !errors.Is(err, ErrSolicitudAlmacenInvalida) {
		t.Fatalf("contexto cancelado creo una confirmacion: %v", err)
	}

	if _, err := NuevaSolicitudConfirmarCargaDirecta(
		context.Background(), contexto, "sesion:carga:otra-0123456789abcdef", comprobante, verificador,
	); err == nil {
		t.Fatal("se acepto un comprobante de otra sesion")
	}
	otroContexto := contextoAlmacenVinculadoPrueba(t, AccionAlmacenConfirmarCargaDirecta)
	if _, err := NuevaSolicitudConfirmarCargaDirecta(
		context.Background(), otroContexto, consumo.SesionRef, comprobante, verificador,
	); err == nil {
		t.Fatal("se acepto un comprobante con otra instancia de capacidad")
	}
	sinComprobante := SolicitudConfirmarCargaDirecta{}
	if !errors.Is(sinComprobante.Validar(), ErrSolicitudAlmacenInvalida) {
		t.Fatal("se confirmo una carga sin consumir recibo")
	}
}

func evidenciaAlmacenVinculadaPrueba(
	contexto ContextoOperacionAlmacen,
	objeto ReferenciaObjetoAlmacen,
	referencia, conectorID, fundamento string,
	realizadaEn time.Time,
) EvidenciaOperacionAlmacen {
	proyeccion, err := contexto.Proyeccion()
	if err != nil {
		return EvidenciaOperacionAlmacen{}
	}
	return EvidenciaOperacionAlmacen{
		Referencia: referencia, ConectorID: conectorID, EsquemaContexto: proyeccion.Esquema,
		AccionNegocio: proyeccion.AccionNegocio, Accion: proyeccion.AccionTecnica,
		EfectoRef: proyeccion.EfectoRef, HuellaPlanEfectoSHA256: proyeccion.HuellaPlanEfectoSHA256,
		HuellaManifiestoSHA256: proyeccion.HuellaManifiestoSHA256,
		HuellaPasoSHA256:       proyeccion.HuellaPasoSHA256,
		PasoRef:                proyeccion.PasoRef, HuellaDecisionSHA256: proyeccion.HuellaDecisionSHA256, Objeto: objeto,
		OperacionRef: proyeccion.OperacionRef, CorrelacionRef: proyeccion.CorrelacionRef,
		AutorizacionRef: proyeccion.AutorizacionRef, Finalidad: proyeccion.Finalidad,
		Clasificacion: proyeccion.Clasificacion, RealizadaEn: realizadaEn,
		CargaRef: proyeccion.CargaRef, SujetoSeudonimoHMAC: proyeccion.SujetoSeudonimoHMAC,
		RecursoRef: proyeccion.RecursoRef, ModuloID: proyeccion.ModuloID,
		HuellaSolicitudHMAC: proyeccion.HuellaSolicitudHMAC, FundamentoRef: fundamento,
	}
}

func TestEscrituraExigeAccionCapacidadesYRespuestaExactas(t *testing.T) {
	contexto := contextoAlmacenVinculadoPrueba(t, AccionAlmacenEscribir)
	solicitud := SolicitudEscribirObjeto{
		Contexto: contexto, ClaveIdempotencia: "idempotencia:escritura:uno", Zona: ZonaAlmacenAdmitida,
		MIME: "application/pdf", Tamano: 9, HuellaSHA256: strings.Repeat("a", 64),
		Contenido: strings.NewReader("contenido"),
	}
	referencia := ReferenciaObjetoAlmacen{Referencia: "objeto:escrito:uno", Version: "v1"}
	evidencia := evidenciaAlmacenVinculadaPrueba(
		contexto, referencia, "evidencia:escritura:uno", "almacen_s3_corporativo", "", instanteCargaDirecta,
	)
	resultado := ResultadoOperacionObjeto{
		Objeto: ObjetoAlmacenado{
			Objeto: referencia, ConectorID: evidencia.ConectorID, Zona: solicitud.Zona, MIME: solicitud.MIME,
			Tamano: solicitud.Tamano, HuellaSHA256: solicitud.HuellaSHA256,
			EvidenciaCreacionRef: evidencia.Referencia, AlmacenadoEn: evidencia.RealizadaEn,
		},
		Evidencia: evidencia,
	}
	capacidades := CapacidadesAlmacenObjetos{
		ConectorID: evidencia.ConectorID, EscrituraEnFlujo: true, ReferenciasOpacas: true,
		IntegridadSHA256: true, TamanoMaximoObjeto: 1024,
	}
	if err := resultado.ValidarEscritura(solicitud, capacidades); err != nil {
		t.Fatalf("escritura valida: %v", err)
	}
	sinIntegridad := capacidades
	sinIntegridad.IntegridadSHA256 = false
	if !errors.Is(resultado.ValidarEscritura(solicitud, sinIntegridad), ErrSolicitudAlmacenInvalida) {
		t.Fatal("se acepto un conector sin integridad declarada")
	}
	accionDistinta := resultado
	accionDistinta.Evidencia.Accion = AccionAlmacenPromover
	if !errors.Is(accionDistinta.ValidarEscritura(solicitud, capacidades), ErrSolicitudAlmacenInvalida) {
		t.Fatal("se acepto evidencia de otra operacion")
	}
	var lectorNulo *strings.Reader
	conLectorNulo := solicitud
	conLectorNulo.Contenido = lectorNulo
	if !errors.Is(conLectorNulo.Validar(), ErrSolicitudAlmacenInvalida) {
		t.Fatal("se acepto un lector tipado nulo")
	}
}

func TestResultadoCargaDirectaExigeCreacionCronologiaAccionYReciboExactos(t *testing.T) {
	contextoPreparar := contextoAlmacenVinculadoPrueba(t, AccionAlmacenPrepararCargaDirecta)
	preparacion := SolicitudPrepararCargaDirecta{
		Contexto: contextoPreparar, ClaveIdempotencia: "idempotencia:carga:uno", MIME: "application/pdf",
		Tamano: 1024, HuellaSHA256: strings.Repeat("a", 64), ExpiraEn: instanteCargaDirecta.Add(time.Minute),
	}
	contextoConfirmar := contextoAlmacenVinculadoPrueba(t, AccionAlmacenConfirmarCargaDirecta)
	confirmacion := confirmacionCargaDirectaPrueba(
		t, contextoConfirmar, "sesion:carga:0123456789abcdefghijkl",
	)
	referencia := ReferenciaObjetoAlmacen{Referencia: "objeto:cuarentena:uno", Version: "v1"}
	creadaEn := instanteCargaDirecta.Add(30 * time.Second)
	evidencia := evidenciaAlmacenVinculadaPrueba(
		contextoConfirmar, referencia, "evidencia:creacion:uno", "almacen_s3_corporativo",
		confirmacion.comprobante.intencionRef, creadaEn,
	)
	resultado := ResultadoOperacionObjeto{
		Objeto: ObjetoAlmacenado{
			Objeto: referencia, ConectorID: evidencia.ConectorID, Zona: ZonaAlmacenCuarentena,
			MIME: preparacion.MIME, Tamano: preparacion.Tamano, HuellaSHA256: preparacion.HuellaSHA256,
			EvidenciaCreacionRef: evidencia.Referencia, AlmacenadoEn: creadaEn,
		},
		Evidencia: evidencia,
	}
	capacidades := CapacidadesAlmacenObjetos{
		ConectorID: evidencia.ConectorID, CargaDirectaTemporal: true, TamanoMaximoObjeto: 4096,
		ReferenciasOpacas: true, IntegridadSHA256: true,
	}
	if err := resultado.ValidarCargaDirecta(preparacion, confirmacion, capacidades); err != nil {
		t.Fatalf("resultado valido: %v", err)
	}
	reintento := resultado
	reintento.Evidencia.Referencia = "evidencia:reintento:idempotente:uno"
	reintento.Evidencia.RealizadaEn = creadaEn.Add(time.Second)
	reintento.Evidencia.ReintentoIdempotente = true
	if err := reintento.ValidarCargaDirecta(preparacion, confirmacion, capacidades); err != nil {
		t.Fatalf("reintento inequivocamente marcado rechazado: %v", err)
	}
	sinMarca := reintento
	sinMarca.Evidencia.ReintentoIdempotente = false
	if !errors.Is(sinMarca.ValidarCargaDirecta(preparacion, confirmacion, capacidades), ErrSolicitudAlmacenInvalida) {
		t.Fatal("un reintento se hizo pasar por la creacion original")
	}

	pruebas := []struct {
		nombre string
		muta   func(*ResultadoOperacionObjeto)
	}{
		{"objeto eliminado", func(r *ResultadoOperacionObjeto) { r.Objeto.Eliminado = true }},
		{"accion distinta", func(r *ResultadoOperacionObjeto) { r.Evidencia.Accion = AccionAlmacenEscribir }},
		{"evidencia anterior al objeto", func(r *ResultadoOperacionObjeto) { r.Evidencia.RealizadaEn = creadaEn.Add(-time.Nanosecond) }},
		{"evidencia de creacion ambigua", func(r *ResultadoOperacionObjeto) { r.Objeto.EvidenciaCreacionRef = "evidencia:otra" }},
		{"sin prueba de recibo", func(r *ResultadoOperacionObjeto) { r.Evidencia.FundamentoRef = "" }},
		{"otro recurso", func(r *ResultadoOperacionObjeto) { r.Evidencia.RecursoRef = "solicitud:bolsa:otra" }},
		{"reintento presentado como creacion", func(r *ResultadoOperacionObjeto) { r.Evidencia.ReintentoIdempotente = true }},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			alterado := resultado
			prueba.muta(&alterado)
			if !errors.Is(alterado.ValidarCargaDirecta(preparacion, confirmacion, capacidades), ErrSolicitudAlmacenInvalida) {
				t.Fatal("se acepto un resultado ambiguo o cruzado")
			}
		})
	}
}

func TestPromocionRechazaOrigenOResultadoBloqueadosYConservaBytes(t *testing.T) {
	origen := ObjetoAlmacenado{
		Objeto:     ReferenciaObjetoAlmacen{Referencia: "objeto:cuarentena:uno", Version: "v1"},
		ConectorID: "almacen_s3_corporativo", Zona: ZonaAlmacenCuarentena, MIME: "application/pdf",
		Tamano: 1024, HuellaSHA256: strings.Repeat("a", 64), EvidenciaCreacionRef: "evidencia:origen:uno",
		AlmacenadoEn: instanteCargaDirecta,
	}
	contexto := contextoAlmacenVinculadoPrueba(t, AccionAlmacenPromover, origen.Objeto)
	solicitud := SolicitudPromoverObjeto{
		Contexto: contexto, ClaveIdempotencia: "idempotencia:promocion:uno", Origen: origen.Objeto,
		EvidenciaAnalisisRef: "evidencia:analisis:limpio:uno",
	}
	promovidoEn := instanteCargaDirecta.Add(time.Minute)
	referencia := ReferenciaObjetoAlmacen{Referencia: "objeto:admitido:uno", Version: "v1"}
	evidencia := evidenciaAlmacenVinculadaPrueba(
		contexto, referencia, "evidencia:promocion:uno", origen.ConectorID,
		solicitud.EvidenciaAnalisisRef, promovidoEn,
	)
	resultado := ResultadoOperacionObjeto{
		Objeto: ObjetoAlmacenado{
			Objeto: referencia, ConectorID: origen.ConectorID, Zona: ZonaAlmacenAdmitida,
			MIME: origen.MIME, Tamano: origen.Tamano, HuellaSHA256: origen.HuellaSHA256,
			EvidenciaCreacionRef: evidencia.Referencia, AlmacenadoEn: promovidoEn,
		},
		Evidencia: evidencia,
	}
	capacidades := CapacidadesAlmacenObjetos{
		ConectorID: origen.ConectorID, PromocionAtomica: true, PreservaObjetoOriginal: true,
		ReferenciasOpacas: true, IntegridadSHA256: true,
	}
	if err := resultado.ValidarPromocion(solicitud, origen, capacidades); err != nil {
		t.Fatalf("promocion valida: %v", err)
	}
	conRetencionAtomica := resultado
	conRetencionAtomica.Objeto.RetenidoHasta = promovidoEn.Add(time.Hour)
	capacidadesRetencionAtomica := capacidades
	capacidadesRetencionAtomica.RetencionAtomicaEnPromocion = true
	if err := conRetencionAtomica.ValidarPromocion(solicitud, origen, capacidadesRetencionAtomica); err != nil {
		t.Fatalf("promocion con WORM en el mismo efecto: %v", err)
	}
	pruebas := []struct {
		nombre string
		muta   func(*ObjetoAlmacenado, *ResultadoOperacionObjeto)
	}{
		{"origen inmovilizado", func(o *ObjetoAlmacenado, _ *ResultadoOperacionObjeto) { o.Inmovilizado = true }},
		{"origen eliminado", func(o *ObjetoAlmacenado, _ *ResultadoOperacionObjeto) { o.Eliminado = true }},
		{"resultado inmovilizado", func(_ *ObjetoAlmacenado, r *ResultadoOperacionObjeto) { r.Objeto.Inmovilizado = true }},
		{"resultado retenido", func(_ *ObjetoAlmacenado, r *ResultadoOperacionObjeto) {
			r.Objeto.RetenidoHasta = promovidoEn.Add(time.Hour)
		}},
		{"accion distinta", func(_ *ObjetoAlmacenado, r *ResultadoOperacionObjeto) { r.Evidencia.Accion = AccionAlmacenEscribir }},
		{"otra huella", func(_ *ObjetoAlmacenado, r *ResultadoOperacionObjeto) {
			r.Objeto.HuellaSHA256 = strings.Repeat("b", 64)
		}},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			otroOrigen, alterado := origen, resultado
			prueba.muta(&otroOrigen, &alterado)
			if !errors.Is(alterado.ValidarPromocion(solicitud, otroOrigen, capacidades), ErrSolicitudAlmacenInvalida) {
				t.Fatal("se acepto una promocion no permitida")
			}
		})
	}
}

func objetoAlmacenadoBasePrueba() ObjetoAlmacenado {
	return ObjetoAlmacenado{
		Objeto:     ReferenciaObjetoAlmacen{Referencia: "objeto:admitido:ciclo:uno", Version: "v1"},
		ConectorID: "almacen_s3_corporativo", Zona: ZonaAlmacenAdmitida, MIME: "application/pdf",
		Tamano: 1024, HuellaSHA256: strings.Repeat("a", 64), EvidenciaCreacionRef: "evidencia:creacion:ciclo",
		AlmacenadoEn: instanteCargaDirecta,
	}
}

func TestOperacionesPrivilegiadasTienenSolicitudesYRespuestasSeparadas(t *testing.T) {
	base := objetoAlmacenadoBasePrueba()

	contextoRetener := contextoAlmacenVinculadoPrueba(t, AccionAlmacenAplicarRetencion, base.Objeto)
	solicitudRetener := SolicitudRetenerObjeto{
		Contexto: contextoRetener, Objeto: base.Objeto, PoliticaRef: "politica:retencion:expediente:v1",
		Hasta: instanteCargaDirecta.Add(24 * time.Hour),
	}
	retenido := base
	retenido.RetenidoHasta = solicitudRetener.Hasta
	evidenciaRetencion := evidenciaAlmacenVinculadaPrueba(
		contextoRetener, base.Objeto, "evidencia:retencion:uno", base.ConectorID,
		solicitudRetener.PoliticaRef, instanteCargaDirecta.Add(time.Minute),
	)
	resultadoRetencion := ResultadoOperacionObjeto{Objeto: retenido, Evidencia: evidenciaRetencion}
	if err := resultadoRetencion.ValidarRetencion(solicitudRetener, base); err != nil {
		t.Fatalf("retencion valida: %v", err)
	}
	acortada := resultadoRetencion
	acortada.Objeto.RetenidoHasta = instanteCargaDirecta.Add(12 * time.Hour)
	solicitudAcortada := solicitudRetener
	solicitudAcortada.Hasta = acortada.Objeto.RetenidoHasta
	if !errors.Is(acortada.ValidarRetencion(solicitudAcortada, retenido), ErrSolicitudAlmacenInvalida) {
		t.Fatal("se permitio acortar una retencion vigente")
	}

	// No existe fabrica positiva para bloqueo legal ni para levantarlo. La
	// mera forma de la peticion no concede esas operaciones privilegiadas.
	solicitudInmovilizar := SolicitudInmovilizarObjeto{
		Objeto:        retenido.Objeto,
		AprobacionRef: "aprobacion:bloqueo-legal:uno", Motivo: "preservacion probatoria autorizada",
	}
	if !errors.Is(solicitudInmovilizar.Validar(), ErrSolicitudAlmacenInvalida) {
		t.Fatal("se inmovilizo sin una capacidad positiva especifica")
	}
	sinAprobacion := solicitudInmovilizar
	sinAprobacion.AprobacionRef = ""
	if !errors.Is(sinAprobacion.Validar(), ErrSolicitudAlmacenInvalida) {
		t.Fatal("se inmovilizo sin aprobacion explicita")
	}

	solicitudLevantar := SolicitudLevantarInmovilizacionObjeto{
		Objeto:        retenido.Objeto,
		AprobacionRef: "aprobacion:levantar-bloqueo:uno", Motivo: "fin del bloqueo autorizado",
	}
	if !errors.Is(solicitudLevantar.Validar(), ErrSolicitudAlmacenInvalida) {
		t.Fatal("se levanto un bloqueo sin una capacidad positiva especifica")
	}
}

func TestEliminacionPrivilegiadaRechazaBloqueoYRetencionVigente(t *testing.T) {
	base := objetoAlmacenadoBasePrueba()
	solicitud := SolicitudEliminarObjeto{
		Objeto: base.Objeto, AprobacionRef: "aprobacion:eliminar:uno",
		Motivo: "fin de conservacion aprobado",
	}
	if !errors.Is(solicitud.Validar(), ErrSolicitudAlmacenInvalida) {
		t.Fatal("se elimino sin una capacidad positiva especifica")
	}
	var contextoAusente ContextoOperacionAlmacen
	if err := contextoAusente.validarParaPaso(AccionAlmacenEliminar); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("la ausencia de autorizacion no denego: %v", err)
	}
}

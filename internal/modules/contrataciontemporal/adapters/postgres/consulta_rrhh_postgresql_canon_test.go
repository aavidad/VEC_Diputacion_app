package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestDecodificarContenidoCuadroRRHHPostgreSQLReproduceCanonGo(t *testing.T) {
	t.Parallel()
	generadaEn := instanteCanonRRHHPostgreSQLPrueba()
	pagina := ports.PaginaCuadroRRHH{
		GeneradaEn: generadaEn,
		Expedientes: []ports.ResumenExpedienteRRHH{
			resumenCanonRRHHPostgreSQLPrueba(2, generadaEn.Add(-time.Minute)),
			resumenCanonRRHHPostgreSQLPrueba(1, generadaEn.Add(-2*time.Minute)),
		},
	}
	pagina.Expedientes[1].ExpedienteRef = "expediente:rrhh:canon:01"
	pagina.Expedientes[1].NumeroVisible = "2026/CT-CANON-01"
	exportacion, err := pagina.ExportarContenidoCanonicoParaSQL()
	if err != nil {
		t.Fatal(err)
	}

	resultado, err := decodificarContenidoCuadroRRHHPostgreSQL(
		exportacion.BytesCanonicos(),
	)
	if err != nil {
		t.Fatalf("decodificar canon de cuadro: %v", err)
	}
	if !resultado.paginaSinRecibo.GeneradaEn.Equal(generadaEn) ||
		len(resultado.paginaSinRecibo.Expedientes) != 2 ||
		resultado.paginaSinRecibo.HayMas ||
		resultado.cursorHuella != ([32]byte{}) {
		t.Fatalf("contenido de cuadro inesperado: %#v", resultado)
	}
}

func TestDecodificarContenidoCuadroRRHHPostgreSQLConservaHuellaCursor(
	t *testing.T,
) {
	t.Parallel()
	materialCursor := bytes.Repeat([]byte{0x5a}, 32)
	cursor := base64.RawURLEncoding.EncodeToString(materialCursor)
	pagina := ports.PaginaCuadroRRHH{
		GeneradaEn: instanteCanonRRHHPostgreSQLPrueba(),
		Expedientes: []ports.ResumenExpedienteRRHH{
			resumenCanonRRHHPostgreSQLPrueba(
				1,
				instanteCanonRRHHPostgreSQLPrueba().Add(-time.Minute),
			),
		},
		HayMas:          true,
		CursorSiguiente: cursor,
	}
	exportacion, err := pagina.ExportarContenidoCanonicoParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := decodificarContenidoCuadroRRHHPostgreSQL(
		exportacion.BytesCanonicos(),
	)
	if err != nil {
		t.Fatalf("decodificar cuadro con cursor: %v", err)
	}
	esperada := sha256.Sum256(materialCursor)
	if !resultado.paginaSinRecibo.HayMas ||
		resultado.paginaSinRecibo.CursorSiguiente != "" ||
		resultado.cursorHuella != esperada {
		t.Fatal("la huella del cursor no se conservó sin revelar el cursor")
	}
}

func TestResumenesCuadroRRHHPostgreSQLRechazanDuplicadosYDesorden(
	t *testing.T,
) {
	t.Parallel()
	generadaEn := instanteCanonRRHHPostgreSQLPrueba()
	primero := resumenCanonRRHHPostgreSQLPrueba(
		2,
		generadaEn.Add(-time.Minute),
	)
	segundo := resumenCanonRRHHPostgreSQLPrueba(
		1,
		generadaEn.Add(-2*time.Minute),
	)
	if resumenesCuadroRRHHPostgreSQLValidos(
		[]ports.ResumenExpedienteRRHH{primero, primero},
		generadaEn,
	) {
		t.Fatal("se aceptó un expediente repetido")
	}
	segundo.ExpedienteRef = "expediente:rrhh:canon:segundo"
	segundo.NumeroVisible = "2026/CT-CANON-02"
	if resumenesCuadroRRHHPostgreSQLValidos(
		[]ports.ResumenExpedienteRRHH{segundo, primero},
		generadaEn,
	) {
		t.Fatal("se aceptó orden temporal ascendente")
	}
	primero.ActualizadoEn = generadaEn.Add(time.Microsecond)
	if resumenesCuadroRRHHPostgreSQLValidos(
		[]ports.ResumenExpedienteRRHH{primero},
		generadaEn,
	) {
		t.Fatal("se aceptó una versión posterior al corte")
	}
}

func TestDecodificarContenidoDetalleRRHHPostgreSQLAceptaTodosLosBloques(
	t *testing.T,
) {
	t.Parallel()
	for bloques := 0; bloques <= 3; bloques++ {
		bloques := bloques
		t.Run(fmt.Sprintf("%d_bloques", bloques), func(t *testing.T) {
			t.Parallel()
			entrada, generadaEn := entradaDetalleCanonRRHHPostgreSQLPrueba(
				t,
				bloques,
			)
			exportacion, err := entrada.ExportarContenidoCanonicoParaSQL(
				generadaEn,
			)
			if err != nil {
				t.Fatal(err)
			}
			resultado, err := decodificarContenidoDetalleRRHHPostgreSQL(
				exportacion.BytesCanonicos(),
			)
			if err != nil {
				t.Fatalf("decodificar detalle: %v", err)
			}
			if resultado.expedienteRef != "expediente:rrhh:canon" ||
				resultado.version != uint64(bloques+1) ||
				!resultado.actualizadoEn.Equal(generadaEn) {
				t.Fatalf("claves de detalle inesperadas: %#v", resultado)
			}
			reexportacion, err := resultado.entrada.
				ExportarContenidoCanonicoParaSQL(generadaEn)
			if err != nil ||
				!bytes.Equal(
					reexportacion.BytesCanonicos(),
					exportacion.BytesCanonicos(),
				) {
				t.Fatal("el detalle decodificado no reproduce el canon")
			}
		})
	}
}

func TestLectorCanonRRHHPostgreSQLRechazaEncuadresNoCanonicos(t *testing.T) {
	t.Parallel()
	casos := map[string][]byte{
		"sin_separador":        []byte("1x"),
		"longitud_vacia":       []byte(":x\n"),
		"cero_inicial":         []byte("01:x\n"),
		"signo":                []byte("+1:x\n"),
		"longitud_excesiva":    []byte("262145:x\n"),
		"valor_truncado":       []byte("2:x\n"),
		"sin_salto_final":      []byte("1:x"),
		"salto_final_distinto": []byte("1:x\r"),
	}
	for nombre, contenido := range casos {
		nombre, contenido := nombre, contenido
		t.Run(nombre, func(t *testing.T) {
			t.Parallel()
			lector := &lectorCanonRRHHPostgreSQL{contenido: contenido}
			if _, err := lector.marco(); err == nil {
				t.Fatal("encuadre no canónico aceptado")
			}
		})
	}
}

func TestLectorCanonRRHHPostgreSQLRechazaTiposNoCanonicos(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre string
		valor  []byte
		leer   func(*lectorCanonRRHHPostgreSQL) error
	}{
		{
			nombre: "utf8",
			valor:  encuadrarCanonRRHHPostgreSQLPrueba([]byte{0xff}),
			leer: func(l *lectorCanonRRHHPostgreSQL) error {
				_, err := l.texto()
				return err
			},
		},
		{
			nombre: "entero_cero_inicial",
			valor:  encuadrarCanonRRHHPostgreSQLPrueba([]byte("01")),
			leer: func(l *lectorCanonRRHHPostgreSQL) error {
				_, err := l.enteroSinSigno()
				return err
			},
		},
		{
			nombre: "entero_mayor_json_exacto",
			valor: encuadrarCanonRRHHPostgreSQLPrueba(
				[]byte("9007199254740992"),
			),
			leer: func(l *lectorCanonRRHHPostgreSQL) error {
				_, err := l.enteroSinSigno()
				return err
			},
		},
		{
			nombre: "entero_con_menos_cero",
			valor:  encuadrarCanonRRHHPostgreSQLPrueba([]byte("-0")),
			leer: func(l *lectorCanonRRHHPostgreSQL) error {
				_, err := l.enteroConSigno()
				return err
			},
		},
		{
			nombre: "booleano",
			valor:  encuadrarCanonRRHHPostgreSQLPrueba([]byte("true")),
			leer: func(l *lectorCanonRRHHPostgreSQL) error {
				_, err := l.booleano()
				return err
			},
		},
		{
			nombre: "instante_sin_microsegundos",
			valor: encuadrarCanonRRHHPostgreSQLPrueba(
				[]byte("2026-07-29T10:11:12Z"),
			),
			leer: func(l *lectorCanonRRHHPostgreSQL) error {
				_, err := l.instante()
				return err
			},
		},
		{
			nombre: "instante_con_desplazamiento",
			valor: encuadrarCanonRRHHPostgreSQLPrueba(
				[]byte("2026-07-29T12:11:12.000000+02:00"),
			),
			leer: func(l *lectorCanonRRHHPostgreSQL) error {
				_, err := l.instante()
				return err
			},
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			lector := &lectorCanonRRHHPostgreSQL{contenido: caso.valor}
			if err := caso.leer(lector); err == nil {
				t.Fatal("tipo no canónico aceptado")
			}
		})
	}
}

func TestDecodificadoresCanonRRHHPostgreSQLRechazanFronteras(t *testing.T) {
	t.Parallel()
	pagina := ports.PaginaCuadroRRHH{
		GeneradaEn: instanteCanonRRHHPostgreSQLPrueba(),
	}
	exportacion, err := pagina.ExportarContenidoCanonicoParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	valido := exportacion.BytesCanonicos()
	casos := map[string][]byte{
		"cabecera_ajena": append(
			[]byte("VEC-CT-CONTENIDO-AJENO-RRHH-V1\n"),
			valido[len(cabeceraContenidoCuadroRRHHPostgreSQL):]...,
		),
		"residuo": append(append([]byte(nil), valido...), 'x'),
		"mas_de_256_kib": bytes.Repeat(
			[]byte{'x'},
			ports.LimiteMaximoCanonResultadoRRHH+1,
		),
	}
	for nombre, canon := range casos {
		nombre, canon := nombre, canon
		t.Run(nombre, func(t *testing.T) {
			t.Parallel()
			if _, err := decodificarContenidoCuadroRRHHPostgreSQL(canon); err == nil {
				t.Fatal("frontera inválida aceptada")
			}
		})
	}
	if _, err := decodificarContenidoDetalleRRHHPostgreSQL(valido); err == nil {
		t.Fatal("el detalle aceptó la cabecera de cuadro")
	}
}

func TestDecodificadorDetalleRRHHPostgreSQLRechazaMascaraYCardinalidad(
	t *testing.T,
) {
	t.Parallel()
	entrada, generadaEn := entradaDetalleCanonRRHHPostgreSQLPrueba(t, 0)
	exportacion, err := entrada.ExportarContenidoCanonicoParaSQL(generadaEn)
	if err != nil {
		t.Fatal(err)
	}
	valido := exportacion.BytesCanonicos()

	// Tras cabecera, resumen (15) y solicitud (4), el siguiente marco es la
	// máscara. Se localiza recorriendo el contrato, no buscando texto.
	lector, err := nuevoLectorCanonRRHHPostgreSQL(
		valido,
		cabeceraContenidoDetalleRRHHPostgreSQL,
	)
	if err != nil {
		t.Fatal(err)
	}
	for indice := 0; indice < 19; indice++ {
		if _, err := lector.marco(); err != nil {
			t.Fatal(err)
		}
	}
	inicioMascara := lector.posicion
	if _, err := lector.marco(); err != nil {
		t.Fatal(err)
	}
	finMascara := lector.posicion
	mascaraInvalida := reemplazarTramoCanonRRHHPostgreSQLPrueba(
		valido,
		inicioMascara,
		finMascara,
		encuadrarCanonRRHHPostgreSQLPrueba([]byte("2")),
	)
	if _, err := decodificarContenidoDetalleRRHHPostgreSQL(
		mascaraInvalida,
	); err == nil {
		t.Fatal("máscara no incremental aceptada")
	}

	// Un total de hitos imposible se rechaza antes de reservar memoria.
	lector, _ = nuevoLectorCanonRRHHPostgreSQL(
		valido,
		cabeceraContenidoDetalleRRHHPostgreSQL,
	)
	for indice := 0; indice < 26; indice++ {
		if _, err := lector.marco(); err != nil {
			t.Fatal(err)
		}
	}
	inicioTotal := lector.posicion
	if _, err := lector.marco(); err != nil {
		t.Fatal(err)
	}
	finTotal := lector.posicion
	totalImposible := reemplazarTramoCanonRRHHPostgreSQLPrueba(
		valido,
		inicioTotal,
		finTotal,
		encuadrarCanonRRHHPostgreSQLPrueba([]byte("9007199254740991")),
	)
	if _, err := decodificarContenidoDetalleRRHHPostgreSQL(
		totalImposible,
	); err == nil {
		t.Fatal("cardinalidad imposible aceptada")
	}
}

func instanteCanonRRHHPostgreSQLPrueba() time.Time {
	return time.Date(2026, 7, 29, 10, 11, 12, 123456000, time.UTC)
}

func resumenCanonRRHHPostgreSQLPrueba(
	version uint64,
	actualizadoEn time.Time,
) ports.ResumenExpedienteRRHH {
	return ports.ResumenExpedienteRRHH{
		ExpedienteRef:   "expediente:rrhh:canon",
		OrganizacionRef: "organizacion:diputacion-granada",
		NumeroVisible:   "2026/CT-CANON",
		Version:         version,
		FlujoRef:        "flujo:rrhh:canon",
		FlujoVersion:    1,
		FlujoHuella:     strings.Repeat("a", 64),
		FaseClave:       "solicitud",
		EstadoClave:     domain.EstadoEnCurso,
		CentroRef:       "centro:rrhh:canon",
		CategoriaRef:    "categoria:rrhh:canon",
		CreadoEn:        actualizadoEn.Add(-time.Hour),
		ActualizadoEn:   actualizadoEn,
	}
}

func entradaDetalleCanonRRHHPostgreSQLPrueba(
	t *testing.T,
	bloques int,
) (ports.EntradaDetalleExpedienteRRHHMinimizada, time.Time) {
	t.Helper()
	ahora := instanteCanonRRHHPostgreSQLPrueba()
	version := uint64(bloques + 1)
	fases := []domain.ClaveFase{
		"solicitud", "gestion_bolsa", "asignacion_unidad", "unidad_gestora",
	}
	hitos := make([]ports.HitoExpedienteRRHH, int(version))
	for indice := range hitos {
		secuencia := uint64(indice + 1)
		faseOrigen := domain.ClaveFase("")
		estadoOrigen := domain.EstadoPendiente
		if indice > 0 {
			faseOrigen = fases[indice-1]
			estadoOrigen = domain.EstadoEnCurso
		}
		hitos[indice] = ports.HitoExpedienteRRHH{
			Secuencia: secuencia, VersionExpediente: secuencia,
			AccionClave: domain.ClaveCatalogo(fmt.Sprintf(
				"actuacion.canon.%d",
				secuencia,
			)),
			RealizadaEn: ahora.Add(time.Duration(indice-bloques) * time.Minute),
			FaseOrigen:  faseOrigen, FaseDestino: fases[indice],
			EstadoOrigen: estadoOrigen, EstadoDestino: domain.EstadoEnCurso,
		}
	}
	inicio := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	resumen := resumenCanonRRHHPostgreSQLPrueba(version, ahora)
	resumen.CreadoEn = hitos[0].RealizadaEn
	resumen.FaseClave = fases[bloques]
	solicitud := ports.SolicitudOperativaRRHH{
		GrupoSubgrupo: "C2", MotivoClave: "sustitucion",
		PeriodoInicio: inicio, PeriodoFin: inicio.AddDate(0, 1, 0),
	}

	var analisis *ports.AnalisisOperativoRRHH
	var refAnalisis ports.ReferenciaHitoAnalisisRRHH
	var cobertura *ports.CoberturaOperativaRRHH
	var refCobertura ports.ReferenciaHitoCoberturaRRHH
	var asignacion *ports.AsignacionOperativaRRHH
	var refAsignacion ports.ReferenciaHitoAsignacionRRHH
	var err error
	if bloques >= 1 {
		resumen.ModalidadClave = "interinidad"
		analisis = &ports.AnalisisOperativoRRHH{
			ModalidadClave: "interinidad", CategoriaRef: resumen.CategoriaRef,
			CausaClave: "sustitucion", PeriodoInicio: inicio,
			PeriodoFin:        inicio.AddDate(0, 1, 0),
			PorcentajeJornada: 10_000, ResultadoRC: domain.RCNoRequerida,
			CostePrevisto: &ports.ImporteOperativoRRHH{
				Centimos: 125_000,
				Moneda:   "EUR",
			},
			FuenteCosteRef: "fuente:coste:canon",
		}
		refAnalisis, err = ports.NuevaReferenciaHitoAnalisisRRHH(2)
		if err != nil {
			t.Fatal(err)
		}
	}
	if bloques >= 2 {
		cobertura = &ports.CoberturaOperativaRRHH{
			ViaClave: "bolsa", ProcedimientoRef: "procedimiento:rrhh:canon",
			BolsaRef: "bolsa:rrhh:canon",
			Comprobaciones: []ports.ComprobacionOperativaRRHH{{
				Clave: "disponibilidad", Resultado: domain.ComprobacionAfirmativa,
			}},
		}
		refCobertura, err = ports.NuevaReferenciaHitoCoberturaRRHH(3)
		if err != nil {
			t.Fatal(err)
		}
	}
	if bloques >= 3 {
		resumen.UnidadRef = "unidad:rrhh:canon"
		asignacion = &ports.AsignacionOperativaRRHH{
			UnidadRef:  resumen.UnidadRef,
			AsignadaEn: hitos[len(hitos)-1].RealizadaEn,
		}
		refAsignacion, err = ports.NuevaReferenciaHitoAsignacionRRHH(4)
		if err != nil {
			t.Fatal(err)
		}
	}
	entrada, err := ports.NuevaEntradaDetalleExpedienteRRHHMinimizada(
		resumen, solicitud,
		analisis, refAnalisis,
		cobertura, refCobertura,
		asignacion, refAsignacion,
		hitos,
	)
	if err != nil {
		t.Fatal(err)
	}
	return entrada, ahora
}

func encuadrarCanonRRHHPostgreSQLPrueba(valor []byte) []byte {
	resultado := []byte(strconvItoaCanonRRHHPostgreSQLPrueba(len(valor)))
	resultado = append(resultado, ':')
	resultado = append(resultado, valor...)
	return append(resultado, '\n')
}

func strconvItoaCanonRRHHPostgreSQLPrueba(valor int) string {
	return fmt.Sprintf("%d", valor)
}

func reemplazarTramoCanonRRHHPostgreSQLPrueba(
	origen []byte,
	inicio int,
	fin int,
	reemplazo []byte,
) []byte {
	resultado := make([]byte, 0, len(origen)-(fin-inicio)+len(reemplazo))
	resultado = append(resultado, origen[:inicio]...)
	resultado = append(resultado, reemplazo...)
	return append(resultado, origen[fin:]...)
}

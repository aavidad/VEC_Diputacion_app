package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

type filasRecuperacionResultadoCoberturaO405Prueba struct {
	contenidos          [][]byte
	longitudes          []int64
	err                 error
	indice              int
	cerrada             bool
	cierres             int
	retenida            []byte
	manifiestoRechazado bool
}

func (f *filasRecuperacionResultadoCoberturaO405Prueba) Close() {
	f.cerrada = true
	f.cierres++
}

func (f *filasRecuperacionResultadoCoberturaO405Prueba) Err() error {
	if !f.cerrada {
		return nil
	}
	return f.err
}

func (*filasRecuperacionResultadoCoberturaO405Prueba) CommandTag() pgconn.CommandTag {
	return pgconn.NewCommandTag("SELECT")
}

func (*filasRecuperacionResultadoCoberturaO405Prueba) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (f *filasRecuperacionResultadoCoberturaO405Prueba) Next() bool {
	if f.cerrada {
		return false
	}
	if f.indice+1 >= len(f.contenidos) {
		f.cerrada = true
		return false
	}
	f.indice++
	return true
}

func (f *filasRecuperacionResultadoCoberturaO405Prueba) Scan(
	destinos ...any,
) error {
	if f.indice < 0 || f.indice >= len(f.contenidos) {
		return errors.New("fila no posicionada")
	}
	if len(destinos) != 3 {
		return errors.New("columnas inesperadas")
	}
	destinoContenido, okContenido := destinos[0].(*[]byte)
	destinoLongitud, okLongitud := destinos[1].(*int64)
	destinoManifiesto, okManifiesto := destinos[2].(*bool)
	if !okContenido || !okLongitud || !okManifiesto {
		return errors.New("destinos inesperados")
	}
	*destinoContenido = append([]byte(nil), f.contenidos[f.indice]...)
	*destinoLongitud = f.longitudes[f.indice]
	*destinoManifiesto = !f.manifiestoRechazado
	f.retenida = *destinoContenido
	return nil
}

func (*filasRecuperacionResultadoCoberturaO405Prueba) Values() ([]any, error) {
	return nil, errors.New("no implementado")
}

func (*filasRecuperacionResultadoCoberturaO405Prueba) RawValues() [][]byte {
	return nil
}

func (*filasRecuperacionResultadoCoberturaO405Prueba) Conn() *pgx.Conn {
	return nil
}

type transaccionRecuperacionResultadoCoberturaO405Prueba struct {
	*transaccionPreparacionPrueba
	filas                  *filasRecuperacionResultadoCoberturaO405Prueba
	errConsulta            error
	consultas              int
	configuradaAntesDeLeer bool
	argumentos             int
	argumentoVivo          []byte
	argumentoCopia         []byte
	limite                 int64
	oidFuncion             uint32
}

func (t *transaccionRecuperacionResultadoCoberturaO405Prueba) oidFuncionRecuperacionCoberturaO405() uint32 {
	if t == nil || t.oidFuncion == 0 {
		return 405
	}
	return t.oidFuncion
}

func (*transaccionRecuperacionResultadoCoberturaO405Prueba) tlsEsperadoRecuperacionCoberturaO405() bool {
	return false
}

func (t *transaccionRecuperacionResultadoCoberturaO405Prueba) Query(
	_ context.Context,
	consulta string,
	argumentos ...any,
) (pgx.Rows, error) {
	t.consultas++
	t.consulta = consulta
	t.configuradaAntesDeLeer = t.configurada
	t.argumentos = len(argumentos)
	if len(argumentos) >= 14 {
		if contenido, ok := argumentos[13].([]byte); ok {
			t.argumentoVivo = contenido
			t.argumentoCopia = append([]byte(nil), contenido...)
		}
	}
	if len(argumentos) == 15 {
		t.limite, _ = argumentos[14].(int64)
	}
	return t.filas, t.errConsulta
}

func cargaRecuperacionResultadoCoberturaO405Prueba() consultaRecuperacionResultadoCoberturaO405 {
	return consultaRecuperacionResultadoCoberturaO405{
		Esquema:         esquemaConsultaRecuperacionResultadoCoberturaO405,
		OrganizacionRef: "organizacion_diputacion",
		ExpedienteRef:   "expediente_temporal_2026",
		AmbitosHMAC: []string{
			"hmac-sha256:vec.contratacion-temporal." +
				"cobertura-decision.ambito/v2:" + strings.Repeat("a", 64),
			"hmac-sha256:vec.contratacion-temporal." +
				"cobertura-decision.ambito/v1:" + strings.Repeat("b", 64),
		},
	}
}

func respuestaNoObservableRecuperacionResultadoCoberturaO405Prueba() []byte {
	return []byte(`{"esquema":"` +
		esquemaResultadoRecuperacionResultadoCoberturaO405 +
		`","estado":"no_observable","observada_en":"2026-07-26T10:02:00.123456789+00:00"}`)
}

func TestEjecutorRecuperacionResultadoCoberturaO405UsaUnaLecturaPrimariaExacta(
	t *testing.T,
) {
	t.Parallel()
	filas := &filasRecuperacionResultadoCoberturaO405Prueba{
		contenidos: [][]byte{
			respuestaNoObservableRecuperacionResultadoCoberturaO405Prueba(),
		},
		longitudes: []int64{
			int64(len(respuestaNoObservableRecuperacionResultadoCoberturaO405Prueba())),
		},
		indice: -1,
	}
	tx := &transaccionRecuperacionResultadoCoberturaO405Prueba{
		transaccionPreparacionPrueba: &transaccionPreparacionPrueba{},
		filas:                        filas,
	}
	iniciador := &iniciadorPreparacionPrueba{tx: tx}
	ejecutor, err :=
		nuevoEjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL(
			iniciador,
		)
	if err != nil {
		t.Fatal(err)
	}
	err = ejecutor.EjecutarLecturaResultadoHistoricoTCB(
		context.Background(),
		func(
			puerto cobertura.SesionLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
		) error {
			sesion, ok := puerto.(*sesionRecuperacionResultadoCoberturaO405)
			if !ok {
				return errors.New("sesion inesperada")
			}
			sesion.mu.Lock()
			sesion.usada = true
			sesion.mu.Unlock()
			resultado, errLectura := sesion.consultarResultadoHistoricoO405(
				context.Background(),
				cargaRecuperacionResultadoCoberturaO405Prueba(),
			)
			if errLectura != nil {
				return errLectura
			}
			if resultado.Encontrado || resultado.ObservadaEn.IsZero() {
				return errors.New("resultado inesperado")
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("ejecutar: %v", err)
	}
	if iniciador.inicios != 1 ||
		iniciador.opciones.IsoLevel != pgx.Serializable ||
		iniciador.opciones.AccessMode != pgx.ReadOnly {
		t.Fatalf(
			"transaccion incorrecta: begin=%d opciones=%+v",
			iniciador.inicios,
			iniciador.opciones,
		)
	}
	if !tx.configuradaAntesDeLeer || tx.consultas != 1 ||
		tx.argumentos != 15 ||
		tx.limite != maximoBytesResultadoRecuperacionResultadoCoberturaO405 ||
		tx.confirmaciones != 1 ||
		tx.reversiones != 1 {
		t.Fatalf(
			"ciclo incorrecto: configurada=%t consultas=%d argumentos=%d "+
				"limite=%d commit=%d rollback=%d",
			tx.configuradaAntesDeLeer,
			tx.consultas,
			tx.argumentos,
			tx.limite,
			tx.confirmaciones,
			tx.reversiones,
		)
	}
	if strings.Count(
		tx.consulta,
		nombreFuncionRecuperacionResultadoCoberturaO405,
	) != 1 ||
		!strings.Contains(tx.consulta, "to_regprocedure($1::text)") ||
		!strings.Contains(tx.consulta, "oid_funcion=$12::oid") ||
		!strings.Contains(tx.consulta, "tls_activo=$13::boolean") ||
		!strings.Contains(tx.consulta, "$14::jsonb") ||
		!strings.Contains(tx.consulta, "$15::bigint") ||
		!strings.Contains(tx.consulta, "pg_catalog.pg_stat_ssl") ||
		!strings.Contains(tx.consulta, "pg_catalog.pg_is_in_recovery") ||
		!strings.Contains(tx.consulta, "session_user::text,current_user::text") ||
		!strings.Contains(tx.consulta, "pg_catalog.pg_auth_members") ||
		!strings.Contains(tx.consulta, "pg_catalog.aclexplode") ||
		!strings.Contains(tx.consulta, "pg_catalog.pg_get_functiondef") ||
		!strings.Contains(tx.consulta, "pg_catalog.sha256") ||
		!strings.Contains(
			tx.consulta,
			"CASE WHEN manifiesto.acreditado THEN (",
		) ||
		!strings.Contains(tx.consulta, "octet_length") ||
		!strings.Contains(tx.consulta, "CASE") {
		t.Fatalf("SQL no es la llamada única esperada: %q", tx.consulta)
	}
	esperado, err := json.Marshal(
		cargaRecuperacionResultadoCoberturaO405Prueba(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if string(tx.argumentoCopia) != string(esperado) {
		t.Fatalf(
			"JSON no canónico:\nobtenido=%s\nesperado=%s",
			tx.argumentoCopia,
			esperado,
		)
	}
	if !bytesBorradosRecuperacionResultadoCoberturaO405Prueba(
		tx.argumentoVivo,
	) {
		t.Fatal("el argumento JSON conservó contenido tras la llamada")
	}
	if !bytesBorradosRecuperacionResultadoCoberturaO405Prueba(
		filas.retenida,
	) {
		t.Fatal("la respuesta []byte conservó contenido tras decodificarse")
	}
	if filas.cierres == 0 {
		t.Fatal("las filas no se cerraron")
	}
}

func TestConsultaAtomicaO405NoPublicaAnteDerivaViva(
	t *testing.T,
) {
	casos := []struct {
		nombre    string
		fragmento string
	}{
		{"TLS vivo", "pg_catalog.pg_stat_ssl"},
		{"primario", "pg_catalog.pg_is_in_recovery"},
		{"session user", "session_user::text,current_user::text"},
		{"membresía", "pg_catalog.pg_auth_members"},
		{"ACL y autoridad", "pg_catalog.aclexplode"},
		{"OID y firma", "oid_funcion=$12::oid"},
		{"prosrc", "procedimiento.prosrc"},
		{"definición canónica", "pg_catalog.pg_get_functiondef"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			respuesta :=
				respuestaNoObservableRecuperacionResultadoCoberturaO405Prueba()
			filas := &filasRecuperacionResultadoCoberturaO405Prueba{
				contenidos:          [][]byte{respuesta},
				longitudes:          []int64{int64(len(respuesta))},
				indice:              -1,
				manifiestoRechazado: true,
			}
			tx := &transaccionRecuperacionResultadoCoberturaO405Prueba{
				transaccionPreparacionPrueba: &transaccionPreparacionPrueba{},
				filas:                        filas,
				oidFuncion:                   405,
			}
			sesion := &sesionRecuperacionResultadoCoberturaO405{
				tx: tx, ctx: context.Background(), oidFuncion: 405,
			}
			resultado, err := sesion.consultarResultadoHistoricoO405(
				context.Background(),
				cargaRecuperacionResultadoCoberturaO405Prueba(),
			)
			if !errors.Is(
				err,
				cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
			) || resultado.Encontrado || !resultado.ObservadaEn.IsZero() ||
				tx.consultas != 1 {
				t.Fatalf(
					"deriva publicó respuesta: resultado=%+v consultas=%d err=%v",
					resultado,
					tx.consultas,
					err,
				)
			}
			if !strings.Contains(tx.consulta, caso.fragmento) {
				t.Fatalf("gate sin acreditación %q", caso.fragmento)
			}
		})
	}
}

func TestEjecutorRecuperacionResultadoCoberturaO405RevierteYSaneaPanic(
	t *testing.T,
) {
	t.Parallel()
	tx := &transaccionPreparacionPrueba{}
	ejecutor, err :=
		nuevoEjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL(
			&iniciadorPreparacionPrueba{tx: tx},
		)
	if err != nil {
		t.Fatal(err)
	}
	err = ejecutor.EjecutarLecturaResultadoHistoricoTCB(
		context.Background(),
		func(
			cobertura.SesionLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
		) error {
			panic("clave_idempotencia=PII-no-publicable")
		},
	)
	if !errors.Is(
		err,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
	) || strings.Contains(err.Error(), "PII") ||
		tx.confirmaciones != 0 || tx.reversiones != 1 {
		t.Fatalf(
			"panic no saneado: err=%v commit=%d rollback=%d",
			err,
			tx.confirmaciones,
			tx.reversiones,
		)
	}
}

func TestEjecutorRecuperacionResultadoCoberturaO405SaneaErrorCallback(
	t *testing.T,
) {
	t.Parallel()
	tx := &transaccionPreparacionPrueba{}
	ejecutor, err :=
		nuevoEjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL(
			&iniciadorPreparacionPrueba{tx: tx},
		)
	if err != nil {
		t.Fatal(err)
	}
	err = ejecutor.EjecutarLecturaResultadoHistoricoTCB(
		context.Background(),
		func(
			cobertura.SesionLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
		) error {
			return errors.New("expediente_ref=PII-no-publicable")
		},
	)
	if !errors.Is(
		err,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
	) || strings.Contains(err.Error(), "PII") ||
		tx.confirmaciones != 0 || tx.reversiones != 1 {
		t.Fatalf(
			"error no saneado: err=%v commit=%d rollback=%d",
			err,
			tx.confirmaciones,
			tx.reversiones,
		)
	}
}

func TestEjecutorRecuperacionResultadoCoberturaO405ClasificaErrorSQLSinReintento(
	t *testing.T,
) {
	t.Parallel()
	casos := []struct {
		codigo string
		quiere error
	}{
		{
			codigo: "23505",
			quiere: cobertura.ErrHistoriaResultadoOperacionDecisionCoberturaDivergente,
		},
		{
			codigo: "42501",
			quiere: cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
		},
		{
			codigo: "55000",
			quiere: cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
		},
		{
			codigo: "XX000",
			quiere: cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.codigo, func(t *testing.T) {
			t.Parallel()
			tx := &transaccionRecuperacionResultadoCoberturaO405Prueba{
				transaccionPreparacionPrueba: &transaccionPreparacionPrueba{},
				errConsulta: &pgconn.PgError{
					Code: caso.codigo, Message: "clave_idempotencia=PII",
				},
			}
			iniciador := &iniciadorPreparacionPrueba{
				tx: tx,
				transacciones: []pgx.Tx{
					tx,
					&transaccionPreparacionPrueba{},
				},
			}
			ejecutor, err :=
				nuevoEjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL(
					iniciador,
				)
			if err != nil {
				t.Fatal(err)
			}
			err = ejecutor.EjecutarLecturaResultadoHistoricoTCB(
				context.Background(),
				func(
					puerto cobertura.SesionLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
				) error {
					sesion :=
						puerto.(*sesionRecuperacionResultadoCoberturaO405)
					sesion.mu.Lock()
					sesion.usada = true
					sesion.mu.Unlock()
					_, errLectura := sesion.consultarResultadoHistoricoO405(
						context.Background(),
						cargaRecuperacionResultadoCoberturaO405Prueba(),
					)
					return errLectura
				},
			)
			if !errors.Is(err, caso.quiere) ||
				strings.Contains(err.Error(), "PII") ||
				iniciador.inicios != 1 || tx.consultas != 1 ||
				tx.confirmaciones != 0 || tx.reversiones != 1 {
				t.Fatalf(
					"clasificacion/ciclo: err=%v begin=%d query=%d "+
						"commit=%d rollback=%d",
					err,
					iniciador.inicios,
					tx.consultas,
					tx.confirmaciones,
					tx.reversiones,
				)
			}
		})
	}
}

func bytesBorradosRecuperacionResultadoCoberturaO405Prueba(
	contenido []byte,
) bool {
	if len(contenido) == 0 {
		return false
	}
	for _, valor := range contenido {
		if valor != 0 {
			return false
		}
	}
	return true
}

func TestSesionRecuperacionResultadoCoberturaO405EsDeUnSoloUsoYCierra(
	t *testing.T,
) {
	t.Parallel()
	guardia := nuevaGuardiaCicloDecisionCoberturaO404E()
	sesion := &sesionRecuperacionResultadoCoberturaO405{
		tx:      &transaccionPreparacionPrueba{},
		ctx:     context.Background(),
		guardia: guardia,
	}
	consultaInvalida :=
		cobertura.ConsultaLecturaResultadoHistoricoOperacionDecisionCoberturaTCB{}
	_, primero := sesion.LeerResultadoHistoricoTCB(
		context.Background(),
		consultaInvalida,
	)
	_, segundo := sesion.LeerResultadoHistoricoTCB(
		context.Background(),
		consultaInvalida,
	)
	if !errors.Is(
		primero,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
	) || !errors.Is(
		segundo,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
	) || !sesion.cerrar() {
		t.Fatalf("sesion no consumida: primero=%v segundo=%v", primero, segundo)
	}
	guardia.cerrar()
	if sesion.entrar() {
		t.Fatal("la sesión retenida volvió a abrirse")
	}
}

func TestConsultaRecuperacionResultadoCoberturaO405ExigeUnaFilaExacta(
	t *testing.T,
) {
	t.Parallel()
	respuesta :=
		respuestaNoObservableRecuperacionResultadoCoberturaO405Prueba()
	casos := []struct {
		nombre     string
		contenidos [][]byte
		longitudes []int64
		errFilas   error
	}{
		{nombre: "cero_filas"},
		{
			nombre:     "dos_filas",
			contenidos: [][]byte{respuesta, respuesta},
			longitudes: []int64{int64(len(respuesta)), int64(len(respuesta))},
		},
		{
			nombre:     "error_tardio",
			contenidos: [][]byte{respuesta},
			longitudes: []int64{int64(len(respuesta))},
			errFilas:   errors.New("cursor truncado con expediente_ref=PII"),
		},
		{
			nombre:     "sobredimension_pre_scan",
			contenidos: [][]byte{nil},
			longitudes: []int64{maximoBytesResultadoRecuperacionResultadoCoberturaO405 + 1},
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			filas := &filasRecuperacionResultadoCoberturaO405Prueba{
				contenidos: caso.contenidos,
				longitudes: caso.longitudes,
				err:        caso.errFilas,
				indice:     -1,
			}
			tx := &transaccionRecuperacionResultadoCoberturaO405Prueba{
				transaccionPreparacionPrueba: &transaccionPreparacionPrueba{},
				filas:                        filas,
			}
			sesion := &sesionRecuperacionResultadoCoberturaO405{
				tx:         tx,
				ctx:        context.Background(),
				oidFuncion: 405,
			}
			resultado, err := sesion.consultarResultadoHistoricoO405(
				context.Background(),
				cargaRecuperacionResultadoCoberturaO405Prueba(),
			)
			if !errors.Is(
				err,
				cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
			) || resultado.Encontrado ||
				!resultado.ObservadaEn.IsZero() ||
				strings.Contains(err.Error(), "PII") {
				t.Fatalf(
					"cardinalidad aceptada: resultado=%+v err=%v",
					resultado,
					err,
				)
			}
			if tx.consultas != 1 ||
				tx.limite !=
					maximoBytesResultadoRecuperacionResultadoCoberturaO405 ||
				filas.cierres == 0 {
				t.Fatalf(
					"consulta no acotada/cerrada: consultas=%d limite=%d cierres=%d",
					tx.consultas,
					tx.limite,
					filas.cierres,
				)
			}
			if caso.nombre == "sobredimension_pre_scan" &&
				len(filas.retenida) != 0 {
				t.Fatal("se materializó una respuesta remota sobredimensionada")
			}
		})
	}
}

func TestNormalizarErrorRecuperacionResultadoCoberturaO405PorSQLState(
	t *testing.T,
) {
	t.Parallel()
	casos := []struct {
		nombre string
		causa  error
		quiere error
	}{
		{
			nombre: "divergencia",
			causa: &pgconn.PgError{
				Code: "23505", Message: "expediente_ref=PII",
			},
			quiere: cobertura.ErrHistoriaResultadoOperacionDecisionCoberturaDivergente,
		},
		{
			nombre: "estado_no_confiable",
			causa: &pgconn.PgError{
				Code: "55000", Message: "clave_idempotencia=PII",
			},
			quiere: cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
		},
		{
			nombre: "acceso_tcb_no_confiable",
			causa: &pgconn.PgError{
				Code: "42501", Message: "perfil_ref=PII",
			},
			quiere: cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
		},
		{
			nombre: "no_analiza_mensaje",
			causa: &pgconn.PgError{
				Code: "XX000", Message: "23505 55000 expediente_ref=PII",
			},
			quiere: cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
		},
		{
			nombre: "cancelacion",
			causa:  context.Canceled,
			quiere: context.Canceled,
		},
		{
			nombre: "plazo",
			causa:  context.DeadlineExceeded,
			quiere: context.DeadlineExceeded,
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			obtenido := normalizarErrorRecuperacionResultadoCoberturaO405(
				context.Background(),
				caso.causa,
			)
			if !errors.Is(obtenido, caso.quiere) ||
				strings.Contains(obtenido.Error(), "PII") {
				t.Fatalf("error=%v; esperado=%v", obtenido, caso.quiere)
			}
		})
	}
}

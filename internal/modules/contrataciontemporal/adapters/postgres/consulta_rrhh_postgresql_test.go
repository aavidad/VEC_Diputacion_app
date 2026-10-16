package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type filaConsultaRRHHPrueba struct {
	err      error
	destinos int
	eventos  *[]string
	llenar   func([]any) error
}

func TestAnexoSelectoresRRHHVerificaDosSellosYFormaCerrada(t *testing.T) {
	filas := []ports.FirmaSelectorRecuperacionRRHH{{FirmaRef: "firma:historica:001",
		Selector: ports.SelectorRecuperacionFirmaRRHH{ExpedienteRef: "expediente:rrhh:001",
			VersionExpediente: 7, Documento: "resolucion", PasoOrden: 3,
			ClaveIdempotencia: "clave-firma-historica-000001", CatalogoHuella: strings.Repeat("a", 64),
			Via: ports.ViaFirmaCertificadoVEC}}}
	canon, err := json.Marshal(filas)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(canon)
	huella := hex.EncodeToString(h[:])
	base := salidaCierreConsultaRRHH{accesoRef: "acceso:rrhh:001", reciboSelloSHA256: strings.Repeat("b", 64)}
	filtro := ports.FiltroUnidadSelectoresRecuperacionRRHH{UnidadRef: "unidad:rrhh:001",
		PerfilAuxRef: "prf_firma_000001", VersionRolAuxRef: "rol:ct_cargo_jefatura_firma_dev:v1",
		AsignacionRef: "asignacion:rrhh:001:v1", AsignacionHuellaSHA256: strings.Repeat("c", 64)}
	recibo, err := json.Marshal(map[string]any{"esquema": "vec.ct.detalle.selectores-recuperacion.v1",
		"acceso_ref": base.accesoRef, "sello_detalle_sha256": base.reciboSelloSHA256,
		"decision_ref": "", "version_rol_ref": "rol:ct_cargo_jefatura_consulta_dev:v2",
		"asignacion_ref": filtro.AsignacionRef, "asignacion_huella_sha256": filtro.AsignacionHuellaSHA256,
		"perfil_aux_ref": filtro.PerfilAuxRef, "version_rol_aux_ref": filtro.VersionRolAuxRef,
		"unidad_ref": filtro.UnidadRef, "selectores_huella_sha256": huella, "selectores": filas})
	if err != nil {
		t.Fatal(err)
	}
	s := sha256.Sum256(recibo)
	sello := hex.EncodeToString(s[:])
	obtenidas, prueba, err := validarAnexoSelectoresRRHH(canon, huella, sello, recibo,
		base, ports.CapacidadConsultaRRHH{}, filtro)
	if err != nil || len(obtenidas) != 1 || prueba.SelloDetalleSHA256 != base.reciboSelloSHA256 ||
		prueba.SelloSelectoresSHA256 != sello {
		t.Fatalf("anexo exacto rechazado: %v", err)
	}
	mutado := append([]byte(nil), recibo...)
	mutado[len(mutado)-2] ^= 1
	if _, _, err := validarAnexoSelectoresRRHH(canon, huella, sello, mutado, base,
		ports.CapacidadConsultaRRHH{}, filtro); !errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable) {
		t.Fatal("recibo alterado admitido")
	}
	if _, _, err := validarAnexoSelectoresRRHH(canon, huella, sello, recibo, base,
		ports.CapacidadConsultaRRHH{}, ports.FiltroUnidadSelectoresRecuperacionRRHH{}); !errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable) {
		t.Fatal("filtro ausente reutilizó filas de otra unidad")
	}
}

func (f *filaConsultaRRHHPrueba) Scan(destinos ...any) error {
	*f.eventos = append(*f.eventos, "scan")
	f.destinos = len(destinos)
	if f.llenar != nil {
		return f.llenar(destinos)
	}
	return f.err
}

type transaccionConsultaRRHHPrueba struct {
	fila           *filaConsultaRRHHPrueba
	consulta       string
	argumentos     int
	valores        []any
	consultas      int
	confirmaciones int
	reversiones    int
	errCommit      error
	alRevertir     func()
	eventos        *[]string
}

func (t *transaccionConsultaRRHHPrueba) Begin(
	context.Context,
) (pgx.Tx, error) {
	return nil, errors.New("no usado")
}
func (t *transaccionConsultaRRHHPrueba) Commit(context.Context) error {
	*t.eventos = append(*t.eventos, "commit")
	t.confirmaciones++
	return t.errCommit
}
func (t *transaccionConsultaRRHHPrueba) Rollback(context.Context) error {
	*t.eventos = append(*t.eventos, "rollback")
	t.reversiones++
	if t.alRevertir != nil {
		t.alRevertir()
	}
	return nil
}
func (*transaccionConsultaRRHHPrueba) CopyFrom(
	context.Context,
	pgx.Identifier,
	[]string,
	pgx.CopyFromSource,
) (int64, error) {
	return 0, errors.New("no usado")
}
func (*transaccionConsultaRRHHPrueba) SendBatch(
	context.Context,
	*pgx.Batch,
) pgx.BatchResults {
	return nil
}
func (*transaccionConsultaRRHHPrueba) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}
func (*transaccionConsultaRRHHPrueba) Prepare(
	context.Context,
	string,
	string,
) (*pgconn.StatementDescription, error) {
	return nil, errors.New("no usado")
}
func (*transaccionConsultaRRHHPrueba) Exec(
	context.Context,
	string,
	...any,
) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("no usado")
}
func (*transaccionConsultaRRHHPrueba) Query(
	context.Context,
	string,
	...any,
) (pgx.Rows, error) {
	return nil, errors.New("no usado")
}
func (t *transaccionConsultaRRHHPrueba) QueryRow(
	_ context.Context,
	consulta string,
	argumentos ...any,
) pgx.Row {
	*t.eventos = append(*t.eventos, "query")
	t.consultas++
	t.consulta = consulta
	t.argumentos = len(argumentos)
	t.valores = argumentos
	return t.fila
}
func (*transaccionConsultaRRHHPrueba) Conn() *pgx.Conn { return nil }

type iniciadorConsultaRRHHPrueba struct {
	tx       pgx.Tx
	opciones pgx.TxOptions
	llamadas int
	err      error
}

func (i *iniciadorConsultaRRHHPrueba) BeginTx(
	_ context.Context,
	opciones pgx.TxOptions,
) (pgx.Tx, error) {
	i.llamadas++
	i.opciones = opciones
	return i.tx, i.err
}

func TestArgumentosSQLConsultaRRHHTienenValorYOrdenExactos(t *testing.T) {
	t.Parallel()
	cursor := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32))
	cuadro, err := ports.NuevaSolicitudCuadroRRHH(
		"texto_sentencia_04",
		domain.EstadoIncidencia,
		domain.ClaveFase("fase_sentencia_05"),
		76,
		cursor,
	)
	if err != nil {
		t.Fatal(err)
	}
	detalle, err := ports.NuevaSolicitudDetalleRRHH(
		"expediente:rrhh:sentencia:04",
		8_765,
	)
	if err != nil {
		t.Fatal(err)
	}
	material := argumentosMaterialConsultaRRHH{
		capacidadCanonica:     []byte("material-09"),
		decisionCanonica:      []byte("material-10"),
		motivoCanonico:        []byte("material-11"),
		contextoActorCanonico: []byte("material-12"),
		personaVersion:        13_013,
		perfilVersion:         14_014,
		payloadVECAD3:         []byte("material-15"),
		sobreCOSESign1:        []byte("material-16"),
		evidenciaVerificacion: []byte("material-17"),
		raizPublicaSPKI:       []byte("material-18"),
	}
	cuadroEsperado := []any{
		"organizacion-01",
		"clase-02",
		"ambito-03",
		"texto_sentencia_04",
		string(domain.EstadoIncidencia),
		"fase_sentencia_05",
		int16(76),
		cursor,
		[]byte("material-09"),
		[]byte("material-10"),
		[]byte("material-11"),
		[]byte("material-12"),
		int64(13_013),
		int64(14_014),
		[]byte("material-15"),
		[]byte("material-16"),
		[]byte("material-17"),
		[]byte("material-18"),
	}
	cuadroActual := argumentosSQLCuadroConsultaRRHH(
		"organizacion-01", "clase-02", "ambito-03", cuadro, material,
	)
	if !reflect.DeepEqual(cuadroActual, cuadroEsperado) {
		t.Fatalf("argumentos de cuadro fuera de contrato:\n%#v", cuadroActual)
	}

	detalleEsperado := []any{
		"organizacion-01",
		"clase-02",
		"ambito-03",
		"expediente:rrhh:sentencia:04",
		int64(8_765),
		[]byte("material-09"),
		[]byte("material-10"),
		[]byte("material-11"),
		[]byte("material-12"),
		int64(13_013),
		int64(14_014),
		[]byte("material-15"),
		[]byte("material-16"),
		[]byte("material-17"),
		[]byte("material-18"),
	}
	detalleActual := argumentosSQLDetalleConsultaRRHH(
		"organizacion-01", "clase-02", "ambito-03", detalle, material,
	)
	if !reflect.DeepEqual(detalleActual, detalleEsperado) {
		t.Fatalf("argumentos de detalle fuera de contrato:\n%#v", detalleActual)
	}
	resumenEsperado := make([]any, 0, 16)
	resumenEsperado = append(resumenEsperado, detalleEsperado[:5]...)
	resumenEsperado = append(resumenEsperado, "unidad:rrhh:001")
	resumenEsperado = append(resumenEsperado, detalleEsperado[5:]...)
	resumenActual := argumentosSQLResumenSeguimientoRRHH(
		"organizacion-01", "clase-02", "ambito-03", detalle,
		"unidad:rrhh:001", material,
	)
	if !reflect.DeepEqual(resumenActual, resumenEsperado) ||
		!strings.Contains(consultaResumenSeguimientoRRHHPostgreSQL,
			"vec_contratacion_temporal.consultar_resumen_seguimiento_rrhh_atestado_v1(") ||
		strings.Contains(consultaResumenSeguimientoRRHHPostgreSQL,
			"consultar_detalle_rrhh_atestado_v1(") ||
		strings.Count(consultaResumenSeguimientoRRHHPostgreSQL, "$16::bytea") != 1 {
		t.Fatalf("contrato SQL de resumen inesperado: %#v", resumenActual)
	}
}

func TestConsultaOriginalPropuestaRRHHTieneFachadaFijaYNoAdmiteOtroUso(t *testing.T) {
	if !strings.Contains(consultaOriginalPropuestaRRHHPostgreSQL,
		"vec_contratacion_temporal.consultar_original_propuesta_rrhh_atestado_v1(") ||
		strings.Contains(consultaOriginalPropuestaRRHHPostgreSQL,
			"consultar_detalle_rrhh_atestado_v1(") ||
		strings.Count(consultaOriginalPropuestaRRHHPostgreSQL, "$15::bytea") != 1 {
		t.Fatal("fachada histórica no fija o contrato de argumentos alterado")
	}
	sesion := &SesionConsultaRRHHPostgreSQL{
		pool:       &iniciadorConsultaRRHHPrueba{},
		analizador: analizadorCanonConsultaRRHHPostgreSQL{},
		modo:       modoConsultaDetalleRRHHOriginalPropuesta,
	}
	if _, err := sesion.ConsultarCuadroYRegistrar(context.Background(), ports.OrdenConsultaCuadroRRHH{}); !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) {
		t.Fatalf("sesión histórica admitió cuadro: %v", err)
	}
	if _, err := sesion.ConsultarDetalleYRegistrar(context.Background(), ports.OrdenConsultaDetalleRRHH{}); !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) {
		t.Fatalf("sesión histórica admitió una versión distinta de v7: %v", err)
	}
}

func TestEjecutarConsultaRRHHFlujoNominalCuadroYDetalle(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre     string
		consulta   string
		argumentos int
		destinos   []any
	}{
		{
			nombre: "cuadro", consulta: consultaCuadroRRHHPostgreSQL,
			argumentos: 18,
			destinos: destinosCuadroConsultaRRHH(
				&salidaCuadroConsultaRRHH{},
			),
		},
		{
			nombre: "detalle", consulta: consultaDetalleRRHHPostgreSQL,
			argumentos: 15,
			destinos: destinosDetalleConsultaRRHH(
				&salidaDetalleConsultaRRHH{},
			),
		},
		{
			nombre: "resumen_seguimiento", consulta: consultaResumenSeguimientoRRHHPostgreSQL,
			argumentos: 16,
			destinos:   make([]any, 8),
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			eventos := make([]string, 0, 5)
			fila := &filaConsultaRRHHPrueba{eventos: &eventos}
			tx := &transaccionConsultaRRHHPrueba{
				fila: fila, eventos: &eventos,
			}
			iniciador := &iniciadorConsultaRRHHPrueba{tx: tx}
			validaciones := 0
			resultado, err := ejecutarConsultaRRHHEnTransaccion(
				context.Background(),
				iniciador,
				caso.consulta,
				make([]any, caso.argumentos),
				caso.destinos,
				func() (string, error) {
					eventos = append(eventos, "validar")
					validaciones++
					return "validado", nil
				},
			)
			if err != nil || resultado != "validado" {
				t.Fatalf("flujo nominal: %q, %v", resultado, err)
			}
			if iniciador.llamadas != 1 || tx.consultas != 1 ||
				tx.confirmaciones != 1 || tx.reversiones != 1 ||
				validaciones != 1 {
				t.Fatalf(
					"flujo/reintento inesperado: begin=%d query=%d "+
						"commit=%d rollback=%d validar=%d",
					iniciador.llamadas, tx.consultas, tx.confirmaciones,
					tx.reversiones, validaciones,
				)
			}
			if iniciador.opciones.IsoLevel != pgx.Serializable ||
				iniciador.opciones.AccessMode != pgx.ReadWrite ||
				tx.consulta != caso.consulta ||
				tx.argumentos != caso.argumentos ||
				fila.destinos != len(caso.destinos) {
				t.Fatalf("contrato transaccional alterado")
			}
			esperados := []string{
				"query", "scan", "validar", "commit", "rollback",
			}
			if !reflect.DeepEqual(eventos, esperados) {
				t.Fatalf("secuencia transaccional = %v", eventos)
			}
		})
	}
}

func TestEjecutarConsultaRRHHFallaCerradoSinReintentos(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre       string
		errFila      error
		errValidar   error
		errCommit    error
		esperado     error
		commits      int
		validaciones int
		eventos      []string
	}{
		{
			nombre: "no_observable",
			errFila: &pgconn.PgError{
				Code: "42501", Message: "detalle privado",
			},
			esperado: ports.ErrConsultaRRHHNoObservable,
			eventos:  []string{"query", "scan", "rollback"},
		},
		{
			nombre:       "analizador_rechaza",
			errValidar:   ports.ErrResultadoConsultaRRHHNoConfiable,
			esperado:     ports.ErrResultadoConsultaRRHHNoConfiable,
			validaciones: 1,
			eventos:      []string{"query", "scan", "validar", "rollback"},
		},
		{
			nombre:    "commit_falla",
			errCommit: errors.New("confirmación sin respuesta"),
			esperado:  ports.ErrConsultaRRHHNoDisponible,
			commits:   1, validaciones: 1,
			eventos: []string{
				"query", "scan", "validar", "commit", "rollback",
			},
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			eventos := make([]string, 0, 5)
			fila := &filaConsultaRRHHPrueba{
				err: caso.errFila, eventos: &eventos,
			}
			tx := &transaccionConsultaRRHHPrueba{
				fila: fila, errCommit: caso.errCommit, eventos: &eventos,
			}
			iniciador := &iniciadorConsultaRRHHPrueba{tx: tx}
			validaciones := 0
			resultado, err := ejecutarConsultaRRHHEnTransaccion(
				context.Background(),
				iniciador,
				consultaCuadroRRHHPostgreSQL,
				make([]any, 18),
				destinosCuadroConsultaRRHH(&salidaCuadroConsultaRRHH{}),
				func() (string, error) {
					eventos = append(eventos, "validar")
					validaciones++
					return "no debe salir", caso.errValidar
				},
			)
			if resultado != "" || !errors.Is(err, caso.esperado) {
				t.Fatalf("resultado o error inseguro: %q, %v", resultado, err)
			}
			if iniciador.llamadas != 1 || tx.consultas != 1 ||
				tx.confirmaciones != caso.commits ||
				tx.reversiones != 1 ||
				validaciones != caso.validaciones {
				t.Fatalf(
					"hubo reintento/efecto: begin=%d query=%d "+
						"commit=%d rollback=%d validar=%d",
					iniciador.llamadas, tx.consultas, tx.confirmaciones,
					tx.reversiones, validaciones,
				)
			}
			if !reflect.DeepEqual(eventos, caso.eventos) {
				t.Fatalf("secuencia transaccional = %v", eventos)
			}
		})
	}
}

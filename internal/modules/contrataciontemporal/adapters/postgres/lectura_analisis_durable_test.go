package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type resultadoLecturaAnalisisDurableO3Prueba struct {
	contenido string
	huella    string
	err       error
}

type filasLecturaAnalisisDurableO3Prueba struct {
	pgx.Rows
	resultados []resultadoLecturaAnalisisDurableO3Prueba
	indice     int
	actual     int
	err        error
	cerradas   bool
	cancelar   context.CancelFunc
}

func (f *filasLecturaAnalisisDurableO3Prueba) Next() bool {
	if f.indice >= len(f.resultados) {
		return false
	}
	f.actual = f.indice
	f.indice++
	return true
}

func (f *filasLecturaAnalisisDurableO3Prueba) Scan(destinos ...any) error {
	if f.cancelar != nil {
		f.cancelar()
		f.cancelar = nil
	}
	resultado := f.resultados[f.actual]
	if resultado.err != nil {
		return resultado.err
	}
	if len(destinos) != 2 {
		return errors.New("contrato de escaneo inesperado")
	}
	contenido, okContenido := destinos[0].(*string)
	huella, okHuella := destinos[1].(*string)
	if !okContenido || !okHuella {
		return errors.New("destinos de escaneo inesperados")
	}
	*contenido = resultado.contenido
	*huella = resultado.huella
	return nil
}

func (f *filasLecturaAnalisisDurableO3Prueba) Err() error { return f.err }
func (f *filasLecturaAnalisisDurableO3Prueba) Close()     { f.cerradas = true }

type transaccionLecturaAnalisisDurableO3Prueba struct {
	pgx.Tx
	filas          pgx.Rows
	errConsulta    error
	errConfigurar  error
	errCommit      error
	consulta       string
	argumentos     []any
	configurada    bool
	confirmaciones int
	reversiones    int
}

func (t *transaccionLecturaAnalisisDurableO3Prueba) Exec(
	_ context.Context,
	_ string,
	_ ...any,
) (pgconn.CommandTag, error) {
	t.configurada = true
	return pgconn.CommandTag{}, t.errConfigurar
}

func (t *transaccionLecturaAnalisisDurableO3Prueba) Query(
	_ context.Context,
	consulta string,
	argumentos ...any,
) (pgx.Rows, error) {
	t.consulta = consulta
	t.argumentos = append([]any(nil), argumentos...)
	return t.filas, t.errConsulta
}

func (t *transaccionLecturaAnalisisDurableO3Prueba) Commit(
	context.Context,
) error {
	t.confirmaciones++
	return t.errCommit
}

func (t *transaccionLecturaAnalisisDurableO3Prueba) Rollback(
	context.Context,
) error {
	t.reversiones++
	return nil
}

type iniciadorLecturaAnalisisDurableO3Prueba struct {
	transacciones []pgx.Tx
	err           error
	inicios       int
	opciones      pgx.TxOptions
}

func (i *iniciadorLecturaAnalisisDurableO3Prueba) BeginTx(
	_ context.Context,
	opciones pgx.TxOptions,
) (pgx.Tx, error) {
	i.opciones = opciones
	if i.err != nil {
		return nil, i.err
	}
	if i.inicios >= len(i.transacciones) {
		return nil, errors.New("transacción de prueba ausente")
	}
	tx := i.transacciones[i.inicios]
	i.inicios++
	return tx, nil
}

func TestLectorExpedienteAnalisisDurableO3PostgreSQLLeeExacto(
	t *testing.T,
) {
	expediente, contenido, huella := expedienteAnalisisDurableO3PostgreSQLPrueba(t)
	filas := &filasLecturaAnalisisDurableO3Prueba{
		resultados: []resultadoLecturaAnalisisDurableO3Prueba{{
			contenido: contenido,
			huella:    huella,
		}},
	}
	tx := &transaccionLecturaAnalisisDurableO3Prueba{filas: filas}
	iniciador := &iniciadorLecturaAnalisisDurableO3Prueba{
		transacciones: []pgx.Tx{tx},
	}
	lector, err := nuevoLectorExpedienteAnalisisDurableO3PostgreSQL(iniciador)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := solicitudLecturaAnalisisDurableO3PostgreSQLPrueba(t, expediente)

	obtenido, err := lector.LeerExpedienteAnalisisDurableO3(
		context.Background(),
		solicitud,
	)
	if err != nil || obtenido.Validar() != nil ||
		obtenido.Referencia != expediente.Referencia ||
		obtenido.Version != expediente.Version ||
		iniciador.opciones.IsoLevel != pgx.Serializable ||
		iniciador.opciones.AccessMode != pgx.ReadOnly ||
		!tx.configurada || tx.confirmaciones != 1 || !filas.cerradas ||
		!strings.Contains(tx.consulta, funcionLeerAnalisisDurableO3) ||
		len(tx.argumentos) != 3 ||
		tx.argumentos[0] != expediente.OrganizacionRef ||
		tx.argumentos[1] != expediente.Referencia ||
		tx.argumentos[2] != expediente.Version {
		t.Fatalf(
			"lectura inesperada: expediente=%#v err=%v tx=%#v",
			obtenido,
			err,
			tx,
		)
	}
}

func TestLectorExpedienteAnalisisDurableO3PostgreSQLFallaCerrado(
	t *testing.T,
) {
	expediente, contenido, huella := expedienteAnalisisDurableO3PostgreSQLPrueba(t)
	duplicado := strings.Replace(
		contenido,
		`"referencia":"`+expediente.Referencia+`"`,
		`"referencia":"`+expediente.Referencia+`",`+
			`"referencia":"`+expediente.Referencia+`"`,
		1,
	)
	extra := strings.Replace(contenido, "{", `{"desconocido":true,`, 1)
	grande := `{"relleno":"` +
		strings.Repeat("x", maximoBytesExpedienteDurableO3) + `"}`
	casos := []struct {
		nombre     string
		resultados []resultadoLecturaAnalisisDurableO3Prueba
		errFilas   error
	}{
		{nombre: "ausente"},
		{nombre: "duplicada", resultados: []resultadoLecturaAnalisisDurableO3Prueba{{
			contenido: duplicado, huella: huella,
		}}},
		{nombre: "campo extra", resultados: []resultadoLecturaAnalisisDurableO3Prueba{{
			contenido: extra, huella: huella,
		}}},
		{nombre: "sobredimensionado", resultados: []resultadoLecturaAnalisisDurableO3Prueba{{
			contenido: grande, huella: huella,
		}}},
		{nombre: "huella SQL divergente", resultados: []resultadoLecturaAnalisisDurableO3Prueba{{
			contenido: contenido, huella: strings.Repeat("f", 64),
		}}},
		{nombre: "huella SQL cero", resultados: []resultadoLecturaAnalisisDurableO3Prueba{{
			contenido: contenido, huella: strings.Repeat("0", 64),
		}}},
		{nombre: "dos filas", resultados: []resultadoLecturaAnalisisDurableO3Prueba{
			{contenido: contenido, huella: huella},
			{contenido: contenido, huella: huella},
		}},
		{nombre: "error terminal", errFilas: errors.New("detalle privado")},
	}
	solicitud := solicitudLecturaAnalisisDurableO3PostgreSQLPrueba(t, expediente)
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			filas := &filasLecturaAnalisisDurableO3Prueba{
				resultados: caso.resultados,
				err:        caso.errFilas,
			}
			tx := &transaccionLecturaAnalisisDurableO3Prueba{filas: filas}
			lector, err := nuevoLectorExpedienteAnalisisDurableO3PostgreSQL(
				&iniciadorLecturaAnalisisDurableO3Prueba{
					transacciones: []pgx.Tx{tx},
				},
			)
			if err != nil {
				t.Fatal(err)
			}
			_, err = lector.LeerExpedienteAnalisisDurableO3(
				context.Background(),
				solicitud,
			)
			if !errors.Is(
				err,
				cobertura.ErrInstantaneaAnalisisDurableNoDisponible,
			) || strings.Contains(err.Error(), "detalle privado") ||
				tx.confirmaciones != 0 {
				t.Fatalf("fallo abierto: %v, commits=%d", err, tx.confirmaciones)
			}
		})
	}
}

func TestLectorExpedienteAnalisisDurableO3PostgreSQLCancelacionYReintento(
	t *testing.T,
) {
	expediente, contenido, huella := expedienteAnalisisDurableO3PostgreSQLPrueba(t)
	solicitud := solicitudLecturaAnalisisDurableO3PostgreSQLPrueba(t, expediente)
	t.Run("cancelacion antes de commit", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		filas := &filasLecturaAnalisisDurableO3Prueba{
			resultados: []resultadoLecturaAnalisisDurableO3Prueba{{
				contenido: contenido, huella: huella,
			}},
			cancelar: cancelar,
		}
		tx := &transaccionLecturaAnalisisDurableO3Prueba{filas: filas}
		lector, _ := nuevoLectorExpedienteAnalisisDurableO3PostgreSQL(
			&iniciadorLecturaAnalisisDurableO3Prueba{
				transacciones: []pgx.Tx{tx},
			},
		)
		_, err := lector.LeerExpedienteAnalisisDurableO3(ctx, solicitud)
		if !errors.Is(err, context.Canceled) || tx.confirmaciones != 0 {
			t.Fatalf("cancelación ignorada: %v", err)
		}
	})
	t.Run("serializacion", func(t *testing.T) {
		primera := &transaccionLecturaAnalisisDurableO3Prueba{
			errConsulta: &pgconn.PgError{Code: "40001"},
		}
		segunda := &transaccionLecturaAnalisisDurableO3Prueba{
			filas: &filasLecturaAnalisisDurableO3Prueba{
				resultados: []resultadoLecturaAnalisisDurableO3Prueba{{
					contenido: contenido, huella: huella,
				}},
			},
		}
		iniciador := &iniciadorLecturaAnalisisDurableO3Prueba{
			transacciones: []pgx.Tx{primera, segunda},
		}
		lector, _ := nuevoLectorExpedienteAnalisisDurableO3PostgreSQL(iniciador)
		if _, err := lector.LeerExpedienteAnalisisDurableO3(
			context.Background(),
			solicitud,
		); err != nil || iniciador.inicios != 2 ||
			segunda.confirmaciones != 1 {
			t.Fatalf("reintento inesperado: %v", err)
		}
	})
}

func TestLectorExpedienteAnalisisDurableO3PostgreSQLRechazaNulos(
	t *testing.T,
) {
	var poolNulo *iniciadorLecturaAnalisisDurableO3Prueba
	if _, err := nuevoLectorExpedienteAnalisisDurableO3PostgreSQL(poolNulo); !errors.Is(
		err,
		cobertura.ErrInstantaneaAnalisisDurableNoDisponible,
	) {
		t.Fatalf("pool nulo aceptado: %v", err)
	}
	var lector *LectorExpedienteAnalisisDurableO3PostgreSQL
	if _, err := lector.LeerExpedienteAnalisisDurableO3(
		context.Background(),
		cobertura.SolicitudInstantaneaAnalisisDurableO3{},
	); !errors.Is(
		err,
		cobertura.ErrSolicitudInstantaneaAnalisisDurableInvalida,
	) {
		t.Fatalf("receptor nulo aceptado: %v", err)
	}
}

func expedienteAnalisisDurableO3PostgreSQLPrueba(
	t *testing.T,
) (domain.Expediente, string, string) {
	t.Helper()
	expediente := expedienteInicialAnalisisPostgreSQLPrueba(t)
	instante := expediente.ActualizadoEn.Add(time.Minute)
	entrada := domain.VinculoEntradaRC{
		Referencia:   "entrada:rc-durable-o3-sintetica-001",
		HuellaSHA256: strings.Repeat("6", 64),
	}
	analisis := domain.AnalisisRRHH{
		ModalidadClave:    "modalidad.interinidad",
		CategoriaRef:      expediente.Solicitud.CategoriaRef,
		GrupoSubgrupo:     expediente.Solicitud.GrupoSubgrupo,
		CausaClave:        "causa.sustitucion",
		Periodo:           expediente.Solicitud.Periodo,
		PorcentajeJornada: domain.JornadaCompletaDiezmilesimas,
		EntradaRCEsperada: entrada,
		ValidacionRC: domain.ValidacionRC{
			Resultado:           domain.RCNoRequerida,
			EntradaRef:          entrada.Referencia,
			HuellaEntradaSHA256: entrada.HuellaSHA256,
			FuenteRef:           "fuente:rc-durable-o3-sintetica-001",
			ReciboRef:           "recibo:rc-durable-o3-sintetico-001",
			ValidadaEn:          instante.Add(-time.Second),
			Motivo:              "No requiere RC en este supuesto sintético.",
		},
	}
	var err error
	expediente, err = expediente.RegistrarAnalisis(
		expediente.Version,
		analisis,
		domain.DatosActuacion{
			AccionClave:   domain.ClaveCatalogo(ports.AccionRegistrarAnalisis),
			ActorRef:      "persona:tecnica-rrhh-sintetica-001",
			UnidadRef:     "unidad:rrhh-sintetica-001",
			ReciboRef:     "recibo:analisis-durable-o3-sintetico-001",
			RealizadaEn:   instante,
			FaseDestino:   expediente.FaseActual,
			EstadoDestino: expediente.EstadoActual,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := json.Marshal(expediente)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := ports.HuellaAnalisisRRHHRehidratadoO3(*expediente.Analisis)
	if err != nil {
		t.Fatal(err)
	}
	return expediente, string(contenido), huella
}

func solicitudLecturaAnalisisDurableO3PostgreSQLPrueba(
	t *testing.T,
	expediente domain.Expediente,
) cobertura.SolicitudInstantaneaAnalisisDurableO3 {
	t.Helper()
	solicitud, err := cobertura.NuevaSolicitudInstantaneaAnalisisDurableO3(
		expediente.OrganizacionRef,
		expediente.Referencia,
		expediente.Version,
	)
	if err != nil {
		t.Fatal(err)
	}
	return solicitud
}

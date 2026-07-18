package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
)

var instanteDiarioPostgreSQLPrueba = time.Date(2026, 7, 18, 10, 0, 0, 123_456_000, time.UTC)

type iniciadorDiarioPostgreSQLPrueba struct {
	tx       pgx.Tx
	opciones pgx.TxOptions
	inicios  int
}

func (i *iniciadorDiarioPostgreSQLPrueba) BeginTx(
	_ context.Context, opciones pgx.TxOptions,
) (pgx.Tx, error) {
	i.opciones = opciones
	i.inicios++
	return i.tx, nil
}

type transaccionDiarioPostgreSQLPrueba struct {
	pgx.Tx
	fila            pgx.Row
	consulta        string
	argumentos      []any
	configuraciones int
	confirmaciones  int
	reversiones     int
	errorCommit     error
	cerrada         bool
}

func (t *transaccionDiarioPostgreSQLPrueba) Exec(
	context.Context, string, ...any,
) (pgconn.CommandTag, error) {
	t.configuraciones++
	return pgconn.NewCommandTag("SELECT 1"), nil
}

func (t *transaccionDiarioPostgreSQLPrueba) QueryRow(
	_ context.Context, consulta string, argumentos ...any,
) pgx.Row {
	t.consulta = consulta
	t.argumentos = make([]any, len(argumentos))
	for indice, argumento := range argumentos {
		if contenido, ok := argumento.([]byte); ok {
			t.argumentos[indice] = append([]byte(nil), contenido...)
			continue
		}
		t.argumentos[indice] = argumento
	}
	return t.fila
}

func (t *transaccionDiarioPostgreSQLPrueba) Commit(context.Context) error {
	t.confirmaciones++
	if t.errorCommit != nil {
		return t.errorCommit
	}
	t.cerrada = true
	return nil
}

func (t *transaccionDiarioPostgreSQLPrueba) Rollback(context.Context) error {
	if t.cerrada {
		return pgx.ErrTxClosed
	}
	t.reversiones++
	t.cerrada = true
	return nil
}

type filaErrorDiarioPostgreSQLPrueba struct{ err error }

func (f filaErrorDiarioPostgreSQLPrueba) Scan(...any) error { return f.err }

// filaParcialDiarioPostgreSQLPrueba reproduce el comportamiento relevante de
// pgx cuando Scan alcanza una columna invalida: los destinos anteriores pueden
// quedar escritos aunque la llamada termine con error.
type filaParcialDiarioPostgreSQLPrueba struct {
	capturados *[][]byte
	err        error
}

func (f filaParcialDiarioPostgreSQLPrueba) Scan(destinos ...any) error {
	for indice, destino := range destinos {
		bytes, ok := destino.(*[]byte)
		if !ok {
			continue
		}
		*bytes = []byte("material-sensible-parcial-" + strconv.Itoa(indice))
		*f.capturados = append(*f.capturados, *bytes)
	}
	return f.err
}

type filaDiarioPostgreSQLPrueba struct{ valores []any }

func (f filaDiarioPostgreSQLPrueba) Scan(destinos ...any) error {
	if len(destinos) != len(f.valores) {
		return errors.New("cantidad de columnas inesperada")
	}
	for indice, valor := range f.valores {
		switch destino := destinos[indice].(type) {
		case *string:
			texto, valido := valor.(string)
			if !valido {
				return errors.New("texto invalido")
			}
			*destino = texto
		case *pgtype.Int8:
			if valor == nil {
				*destino = pgtype.Int8{}
				continue
			}
			numero, valido := valor.(int64)
			if !valido {
				return errors.New("entero invalido")
			}
			*destino = pgtype.Int8{Int64: numero, Valid: true}
		case *pgtype.Text:
			if valor == nil {
				*destino = pgtype.Text{}
				continue
			}
			texto, valido := valor.(string)
			if !valido {
				return errors.New("texto nulo invalido")
			}
			*destino = pgtype.Text{String: texto, Valid: true}
		case *pgtype.Timestamptz:
			if valor == nil {
				*destino = pgtype.Timestamptz{}
				continue
			}
			instante, valido := valor.(time.Time)
			if !valido {
				return errors.New("instante invalido")
			}
			*destino = pgtype.Timestamptz{Time: instante, Valid: true}
		case *[]byte:
			if valor == nil {
				*destino = nil
				continue
			}
			contenido, valido := valor.([]byte)
			if !valido {
				return errors.New("json invalido")
			}
			*destino = append([]byte(nil), contenido...)
		default:
			return errors.New("destino no soportado")
		}
	}
	return nil
}

func TestDiarioPostgreSQLConsultaAusenteEnTransaccionSerializableSoloLectura(t *testing.T) {
	tx := &transaccionDiarioPostgreSQLPrueba{fila: filaErrorDiarioPostgreSQLPrueba{pgx.ErrNoRows}}
	iniciador := &iniciadorDiarioPostgreSQLPrueba{tx: tx}
	diario, err := nuevoDiarioOperacionesBorradorPostgreSQL(iniciador)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := solicitudConsultaDiarioPostgreSQLPrueba(t)
	resultado, err := diario.ConsultarIdentidades(context.Background(), solicitud)
	if err != nil || len(resultado.Coincidencias) != 0 {
		t.Fatalf("ausencia no restaurada: resultado=%+v err=%v", resultado, err)
	}
	if iniciador.inicios != 1 || iniciador.opciones.IsoLevel != pgx.Serializable ||
		iniciador.opciones.AccessMode != pgx.ReadOnly || tx.configuraciones != 1 ||
		tx.confirmaciones != 1 || !strings.Contains(tx.consulta, funcionConsultarIdentidadesBorradorPostgreSQL) {
		t.Fatalf("frontera transaccional inesperada: opciones=%+v tx=%+v", iniciador.opciones, tx)
	}
}

func TestDiarioPostgreSQLBorraBuffersAunqueScanFalleTrasEscrituraParcial(t *testing.T) {
	reserva := solicitudReservaCompletaDiarioPostgreSQLPrueba(t)
	reclamacion := solicitudReclamacionDiarioPostgreSQLPrueba(t, reserva)
	identidad := reserva.Proyeccion.IdentidadPrimaria
	reconciliacion := gobiernoconvocatorias.SolicitudReconciliacionBorrador{
		IdentidadPrimaria: identidad,
		Control: gobiernoconvocatorias.ResultadoOperacionDiario{
			Estado: gobiernoconvocatorias.ResultadoDiarioReservado, Revision: 1, Cercado: 1,
			ArrendamientoIniciaEn: reserva.Proyeccion.ArrendamientoIniciaEn,
			ArrendamientoVenceEn:  reserva.Proyeccion.ArrendamientoVenceEn,
		},
		SolicitadaEn: reserva.Proyeccion.ArrendamientoIniciaEn.Add(time.Second),
	}
	if reconciliacion.Validar() != nil {
		t.Fatal("fixture de reconciliacion invalida")
	}

	casos := []struct {
		nombre        string
		minimoBuffers int
		ejecutar      func(*DiarioOperacionesBorradorPostgreSQL) error
	}{
		{
			nombre: "consulta", minimoBuffers: 3,
			ejecutar: func(diario *DiarioOperacionesBorradorPostgreSQL) error {
				_, err := diario.ConsultarIdentidades(context.Background(), solicitudConsultaDiarioPostgreSQLPrueba(t))
				return err
			},
		},
		{
			nombre: "reserva", minimoBuffers: 3,
			ejecutar: func(diario *DiarioOperacionesBorradorPostgreSQL) error {
				_, err := diario.ReservarDecision(context.Background(), reserva)
				return err
			},
		},
		{
			nombre: "reconciliacion", minimoBuffers: 1,
			ejecutar: func(diario *DiarioOperacionesBorradorPostgreSQL) error {
				_, err := diario.Reconciliar(context.Background(), reconciliacion)
				return err
			},
		},
		{
			nombre: "reclamacion", minimoBuffers: 2,
			ejecutar: func(diario *DiarioOperacionesBorradorPostgreSQL) error {
				_, err := diario.ReclamarDecision(context.Background(), reclamacion)
				return err
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			capturados := make([][]byte, 0, caso.minimoBuffers)
			fila := filaParcialDiarioPostgreSQLPrueba{
				capturados: &capturados, err: &pgconn.PgError{Code: "08006"},
			}
			tx := &transaccionDiarioPostgreSQLPrueba{fila: fila}
			diario, err := nuevoDiarioOperacionesBorradorPostgreSQL(
				&iniciadorDiarioPostgreSQLPrueba{tx: tx},
			)
			if err != nil {
				t.Fatal(err)
			}
			err = caso.ejecutar(diario)
			if !errors.Is(err, gobiernoconvocatorias.ErrOperacionBorradorIndeterminada) ||
				len(capturados) < caso.minimoBuffers {
				t.Fatalf("fallo parcial no reproducido: err=%v buffers=%d", err, len(capturados))
			}
			for indice, contenido := range capturados {
				for _, valor := range contenido {
					if valor != 0 {
						t.Fatalf("buffer %d no borrado: %q", indice, contenido)
					}
				}
			}
		})
	}
}

func TestDiarioPostgreSQLConsultaConflictoRestauraResolucionExacta(t *testing.T) {
	solicitud := solicitudConsultaDiarioPostgreSQLPrueba(t)
	identidadDTO, err := proyectarIdentidadDiarioPostgreSQL(solicitud.Identidades[0])
	if err != nil {
		t.Fatal(err)
	}
	identidades, _ := json.Marshal([]identidadDiarioPostgreSQL{identidadDTO})
	primaria, _ := json.Marshal(identidadDTO)
	valores := []any{
		"conflicto", int64(0), int64(0), nil, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		identidades, primaria,
	}
	tx := &transaccionDiarioPostgreSQLPrueba{fila: filaDiarioPostgreSQLPrueba{valores}}
	diario, _ := nuevoDiarioOperacionesBorradorPostgreSQL(&iniciadorDiarioPostgreSQLPrueba{tx: tx})
	resultado, err := diario.ConsultarIdentidades(context.Background(), solicitud)
	if err != nil || len(resultado.Coincidencias) != 1 ||
		resultado.Coincidencias[0].Resultado.Estado != gobiernoconvocatorias.ResultadoDiarioConflicto {
		t.Fatalf("conflicto no restaurado: resultado=%+v err=%v", resultado, err)
	}
	if !identidadesDiarioPostgreSQLIguales(
		resultado.Coincidencias[0].Resolucion.IdentidadPrimaria, solicitud.Identidades[0],
	) || tx.confirmaciones != 1 {
		t.Fatalf("resolucion o commit inesperados: %+v", resultado.Coincidencias[0].Resolucion)
	}
}

func TestDiarioPostgreSQLReconciliacionConservaLeaseRevisionYCercado(t *testing.T) {
	identidad, err := restaurarIdentidadDiarioPostgreSQL(identidadDiarioPostgreSQLPrueba(t))
	if err != nil {
		t.Fatal(err)
	}
	inicio := instanteDiarioPostgreSQLPrueba
	vence := inicio.Add(2 * time.Minute)
	solicitud := gobiernoconvocatorias.SolicitudReconciliacionBorrador{
		IdentidadPrimaria: identidad,
		Control: gobiernoconvocatorias.ResultadoOperacionDiario{
			Estado:   gobiernoconvocatorias.ResultadoDiarioReservado,
			Revision: 1, Cercado: 1,
			ArrendamientoIniciaEn: inicio, ArrendamientoVenceEn: vence,
		},
		SolicitadaEn: inicio.Add(time.Second),
	}
	if solicitud.Validar() != nil {
		t.Fatal("fixture de reconciliacion invalida")
	}
	valores := []any{
		"reservado", int64(1), int64(1), inicio, vence,
		nil, nil, inicio.Add(2 * time.Second), nil,
	}
	tx := &transaccionDiarioPostgreSQLPrueba{fila: filaDiarioPostgreSQLPrueba{valores}}
	iniciador := &iniciadorDiarioPostgreSQLPrueba{tx: tx}
	diario, _ := nuevoDiarioOperacionesBorradorPostgreSQL(iniciador)
	resultado, err := diario.Reconciliar(context.Background(), solicitud)
	if err != nil || resultado.Resultado.Revision != 1 || resultado.Resultado.Cercado != 1 ||
		!resultado.Resultado.ArrendamientoIniciaEn.Equal(inicio) ||
		!resultado.Resultado.ArrendamientoVenceEn.Equal(vence) {
		t.Fatalf("reconciliacion no restaurada: resultado=%+v err=%v", resultado, err)
	}
	if iniciador.opciones.IsoLevel != pgx.Serializable || iniciador.opciones.AccessMode != pgx.ReadWrite ||
		tx.confirmaciones != 1 || !strings.Contains(tx.consulta, funcionReconciliarBorradorPostgreSQL) {
		t.Fatalf("frontera de reconciliacion inesperada: opciones=%+v consulta=%q", iniciador.opciones, tx.consulta)
	}
}

func TestDiarioPostgreSQLReservarDecisionDistingueCreacionYOperacionEnCurso(t *testing.T) {
	solicitud := solicitudReservaCompletaDiarioPostgreSQLPrueba(t)
	identidades, primaria := resolucionJSONDiarioPostgreSQLPrueba(t, solicitud)
	casos := []struct {
		nombre string
		estado string
		inicio time.Time
		vence  time.Time
	}{
		{
			nombre: "reserva creada", estado: "reservado",
			inicio: solicitud.Proyeccion.ArrendamientoIniciaEn,
			vence:  solicitud.Proyeccion.ArrendamientoVenceEn,
		},
		{
			nombre: "reserva existente",
			estado: "en_curso",
			inicio: solicitud.Proyeccion.ArrendamientoIniciaEn.Add(-30 * time.Second),
			vence:  solicitud.Proyeccion.ArrendamientoVenceEn.Add(-30 * time.Second),
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			tx := &transaccionDiarioPostgreSQLPrueba{fila: filaDiarioPostgreSQLPrueba{[]any{
				caso.estado, int64(1), int64(1), caso.inicio, caso.vence, nil, identidades, primaria,
			}}}
			iniciador := &iniciadorDiarioPostgreSQLPrueba{tx: tx}
			diario, _ := nuevoDiarioOperacionesBorradorPostgreSQL(iniciador)
			resultado, err := diario.ReservarDecision(context.Background(), solicitud)
			if err != nil || string(resultado.Resultado.Estado) != caso.estado ||
				!resultado.Resultado.ArrendamientoIniciaEn.Equal(caso.inicio) ||
				!resultado.Resultado.ArrendamientoVenceEn.Equal(caso.vence) {
				t.Fatalf("resultado de reserva inesperado: %+v err=%v", resultado, err)
			}
			if iniciador.opciones.IsoLevel != pgx.Serializable || iniciador.opciones.AccessMode != pgx.ReadWrite ||
				tx.confirmaciones != 1 || tx.reversiones != 0 || len(tx.argumentos) != 6 ||
				!strings.Contains(tx.consulta, funcionReservarDecisionBorradorPostgreSQL) {
				t.Fatalf("frontera de reserva inesperada: opciones=%+v tx=%+v", iniciador.opciones, tx)
			}
			validarArgumentosReservaDiarioPostgreSQLPrueba(t, tx.argumentos)
		})
	}
}

func TestDiarioPostgreSQLReservarDecisionRevierteFilaManipuladaYCommitFallido(t *testing.T) {
	solicitud := solicitudReservaCompletaDiarioPostgreSQLPrueba(t)
	identidades, primaria := resolucionJSONDiarioPostgreSQLPrueba(t, solicitud)
	fila := func(inicio, vence time.Time) pgx.Row {
		return filaDiarioPostgreSQLPrueba{[]any{
			"reservado", int64(1), int64(1), inicio, vence, nil, identidades, primaria,
		}}
	}
	t.Run("lease manipulado", func(t *testing.T) {
		tx := &transaccionDiarioPostgreSQLPrueba{fila: fila(
			solicitud.Proyeccion.ArrendamientoIniciaEn.Add(time.Second),
			solicitud.Proyeccion.ArrendamientoVenceEn,
		)}
		diario, _ := nuevoDiarioOperacionesBorradorPostgreSQL(&iniciadorDiarioPostgreSQLPrueba{tx: tx})
		_, err := diario.ReservarDecision(context.Background(), solicitud)
		if !errors.Is(err, gobiernoconvocatorias.ErrResultadoBorradorInseguro) ||
			tx.confirmaciones != 0 || tx.reversiones != 1 {
			t.Fatalf("fila manipulada no fallo cerrada: err=%v tx=%+v", err, tx)
		}
	})
	t.Run("commit ambiguo", func(t *testing.T) {
		tx := &transaccionDiarioPostgreSQLPrueba{
			fila:        fila(solicitud.Proyeccion.ArrendamientoIniciaEn, solicitud.Proyeccion.ArrendamientoVenceEn),
			errorCommit: errors.New("conexion perdida durante commit"),
		}
		diario, _ := nuevoDiarioOperacionesBorradorPostgreSQL(&iniciadorDiarioPostgreSQLPrueba{tx: tx})
		_, err := diario.ReservarDecision(context.Background(), solicitud)
		if !errors.Is(err, gobiernoconvocatorias.ErrOperacionBorradorIndeterminada) ||
			tx.confirmaciones != 1 || tx.reversiones != 1 {
			t.Fatalf("commit ambiguo no quedo indeterminado: err=%v tx=%+v", err, tx)
		}
	})
}

func TestDiarioPostgreSQLReclamarDecisionExigeIncrementoEIdentidadExacta(t *testing.T) {
	reserva := solicitudReservaCompletaDiarioPostgreSQLPrueba(t)
	solicitud := solicitudReclamacionDiarioPostgreSQLPrueba(t, reserva)
	primariaDTO, _ := proyectarIdentidadDiarioPostgreSQL(solicitud.Nueva.Proyeccion.IdentidadPrimaria)
	primaria, _ := json.Marshal(primariaDTO)
	filaValida := func() pgx.Row {
		return filaDiarioPostgreSQLPrueba{[]any{
			"reservado", int64(3), int64(3), solicitud.Nueva.Proyeccion.ArrendamientoIniciaEn,
			solicitud.Nueva.Proyeccion.ArrendamientoVenceEn, nil, primaria,
		}}
	}
	t.Run("reclamacion creciente", func(t *testing.T) {
		tx := &transaccionDiarioPostgreSQLPrueba{fila: filaValida()}
		iniciador := &iniciadorDiarioPostgreSQLPrueba{tx: tx}
		diario, _ := nuevoDiarioOperacionesBorradorPostgreSQL(iniciador)
		resultado, err := diario.ReclamarDecision(context.Background(), solicitud)
		if err != nil || resultado.Revision != 3 || resultado.Cercado != 3 ||
			resultado.Estado != gobiernoconvocatorias.ResultadoDiarioReservado {
			t.Fatalf("reclamacion no restaurada: %+v err=%v", resultado, err)
		}
		if iniciador.opciones.IsoLevel != pgx.Serializable || iniciador.opciones.AccessMode != pgx.ReadWrite ||
			len(tx.argumentos) != 8 || tx.argumentos[0] != int64(2) || tx.argumentos[1] != int64(2) ||
			tx.confirmaciones != 1 || tx.reversiones != 0 ||
			!strings.Contains(tx.consulta, funcionReclamarReservaBorradorPostgreSQL) {
			t.Fatalf("frontera de reclamacion inesperada: opciones=%+v tx=%+v", iniciador.opciones, tx)
		}
		validarArgumentosReservaDiarioPostgreSQLPrueba(t, tx.argumentos[2:])
	})
	t.Run("revision no creciente", func(t *testing.T) {
		tx := &transaccionDiarioPostgreSQLPrueba{fila: filaDiarioPostgreSQLPrueba{[]any{
			"reservado", int64(2), int64(3), solicitud.Nueva.Proyeccion.ArrendamientoIniciaEn,
			solicitud.Nueva.Proyeccion.ArrendamientoVenceEn, nil, primaria,
		}}}
		diario, _ := nuevoDiarioOperacionesBorradorPostgreSQL(&iniciadorDiarioPostgreSQLPrueba{tx: tx})
		_, err := diario.ReclamarDecision(context.Background(), solicitud)
		if !errors.Is(err, gobiernoconvocatorias.ErrResultadoBorradorInseguro) || tx.reversiones != 1 {
			t.Fatalf("revision no creciente aceptada: err=%v tx=%+v", err, tx)
		}
	})
	t.Run("identidad ajena", func(t *testing.T) {
		ajenaDTO, _ := proyectarIdentidadDiarioPostgreSQL(solicitud.Nueva.IdentidadesConsulta[1])
		ajena, _ := json.Marshal(ajenaDTO)
		tx := &transaccionDiarioPostgreSQLPrueba{fila: filaDiarioPostgreSQLPrueba{[]any{
			"reservado", int64(3), int64(3), solicitud.Nueva.Proyeccion.ArrendamientoIniciaEn,
			solicitud.Nueva.Proyeccion.ArrendamientoVenceEn, nil, ajena,
		}}}
		diario, _ := nuevoDiarioOperacionesBorradorPostgreSQL(&iniciadorDiarioPostgreSQLPrueba{tx: tx})
		_, err := diario.ReclamarDecision(context.Background(), solicitud)
		if !errors.Is(err, gobiernoconvocatorias.ErrResultadoBorradorInseguro) || tx.reversiones != 1 {
			t.Fatalf("identidad ajena aceptada: err=%v tx=%+v", err, tx)
		}
	})
	t.Run("commit ambiguo", func(t *testing.T) {
		tx := &transaccionDiarioPostgreSQLPrueba{
			fila: filaValida(), errorCommit: errors.New("conexion perdida durante commit"),
		}
		diario, _ := nuevoDiarioOperacionesBorradorPostgreSQL(&iniciadorDiarioPostgreSQLPrueba{tx: tx})
		_, err := diario.ReclamarDecision(context.Background(), solicitud)
		if !errors.Is(err, gobiernoconvocatorias.ErrOperacionBorradorIndeterminada) ||
			tx.confirmaciones != 1 || tx.reversiones != 1 {
			t.Fatalf("commit ambiguo no quedo indeterminado: err=%v tx=%+v", err, tx)
		}
	})
}

func resolucionJSONDiarioPostgreSQLPrueba(
	t *testing.T, solicitud gobiernoconvocatorias.SolicitudReservaDecisionBorrador,
) ([]byte, []byte) {
	t.Helper()
	identidades := make([]identidadDiarioPostgreSQL, len(solicitud.IdentidadesConsulta))
	for indice, identidad := range solicitud.IdentidadesConsulta {
		var err error
		identidades[indice], err = proyectarIdentidadDiarioPostgreSQL(identidad)
		if err != nil {
			t.Fatal(err)
		}
	}
	primaria, err := proyectarIdentidadDiarioPostgreSQL(solicitud.Proyeccion.IdentidadPrimaria)
	if err != nil {
		t.Fatal(err)
	}
	identidadesJSON, err := json.Marshal(identidades)
	if err != nil {
		t.Fatal(err)
	}
	primariaJSON, err := json.Marshal(primaria)
	if err != nil {
		t.Fatal(err)
	}
	return identidadesJSON, primariaJSON
}

func solicitudReclamacionDiarioPostgreSQLPrueba(
	t *testing.T, reserva gobiernoconvocatorias.SolicitudReservaDecisionBorrador,
) gobiernoconvocatorias.SolicitudReclamacionDecisionBorrador {
	t.Helper()
	nueva := reserva
	nueva.SolicitadaEn = reserva.Proyeccion.ArrendamientoVenceEn
	nueva.Proyeccion.ArrendamientoIniciaEn = nueva.SolicitadaEn
	nueva.Proyeccion.ArrendamientoVenceEn = nueva.SolicitadaEn.Add(2 * time.Minute)
	if nueva.Validar() != nil {
		t.Fatal("fixture de nueva reserva para reclamacion invalida")
	}
	anterior := gobiernoconvocatorias.ResultadoOperacionDiario{
		Estado:   gobiernoconvocatorias.ResultadoDiarioNoAplicado,
		Revision: 2, Cercado: 2,
		ArrendamientoIniciaEn: reserva.Proyeccion.ArrendamientoIniciaEn,
		ArrendamientoVenceEn:  reserva.Proyeccion.ArrendamientoVenceEn,
	}
	solicitud := gobiernoconvocatorias.SolicitudReclamacionDecisionBorrador{
		ResolucionAnterior: gobiernoconvocatorias.ResolucionIdentidadBorrador{
			IdentidadesConsultadas: append(
				[]gobiernoconvocatorias.ProyeccionIdentidadOperacion(nil), reserva.IdentidadesConsulta...,
			),
			IdentidadPrimaria: reserva.Proyeccion.IdentidadPrimaria,
		},
		Reconciliacion: gobiernoconvocatorias.ResultadoReconciliacionBorrador{
			Resultado: anterior, ComprobadaEn: reserva.Proyeccion.ArrendamientoVenceEn,
			PruebaDesenlaceRef: "prueba:rollback:borrador:captura",
			HuellaPruebaSHA256: strings.Repeat("c", 64),
		},
		Nueva: nueva, SolicitadaEn: nueva.SolicitadaEn,
	}
	if solicitud.Validar() != nil {
		t.Fatal("fixture de reclamacion invalida")
	}
	return solicitud
}

func validarArgumentosReservaDiarioPostgreSQLPrueba(t *testing.T, argumentos []any) {
	t.Helper()
	if len(argumentos) != 6 {
		t.Fatalf("cantidad de argumentos de reserva: %d", len(argumentos))
	}
	fragmentos := []string{
		`"esquema":"vec.bolsa.convocatoria.reserva-decision.v2"`,
		`"principal_ref":`,
		`"esquema":"bolsa.convocatoria.intencion.v2"`,
		`"estado_gobierno":"borrador"`,
		`"decision_ref":`,
		`"ambitos":`,
	}
	for indice, fragmento := range fragmentos {
		contenido, valido := argumentos[indice].([]byte)
		if !valido || !json.Valid(contenido) || !strings.Contains(string(contenido), fragmento) {
			t.Fatalf("argumento %d fuera de orden o invalido: %q", indice+1, contenido)
		}
	}
}

func TestDiarioPostgreSQLJSONDeIdentidadEsCerradoYNoAmbiguo(t *testing.T) {
	identidad := identidadDiarioPostgreSQLPrueba(t)
	valida, err := json.Marshal(identidad)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := restaurarIdentidadDesdeJSONDiarioPostgreSQL(valida); err != nil {
		t.Fatalf("identidad valida rechazada: %v", err)
	}
	conDesconocida := append([]byte(`{"desconocida":true,`), valida[1:]...)
	duplicada := append([]byte(`{"localizador":{},`), valida[1:]...)
	for nombre, contenido := range map[string][]byte{
		"campo desconocido": conDesconocida,
		"clave duplicada":   duplicada,
		"contenido añadido": append(append([]byte(nil), valida...), []byte(` {}`)...),
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := restaurarIdentidadDesdeJSONDiarioPostgreSQL(contenido); !errors.Is(err, gobiernoconvocatorias.ErrResultadoBorradorInseguro) {
				t.Fatalf("json ambiguo aceptado: %v", err)
			}
		})
	}
}

func TestDiarioPostgreSQLReciboAntiguoSinEvidenciaKMSFallaCerrado(t *testing.T) {
	identidad := identidadDiarioPostgreSQLPrueba(t)
	contenido, err := json.Marshal(map[string]any{
		"esquema":    "bolsa.convocatoria.borrador.recibo.v2",
		"recibo_ref": "recibo:incompleto", "transaccion_ref": "transaccion:incompleta",
		"accion": "bolsa.convocatoria.borrador.crear", "identidad": identidad,
		"confirmada_en": instanteDiarioPostgreSQLPrueba.Format(formatoInstanteMicrosegundo),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := restaurarReciboBorradorPostgreSQL(contenido); !errors.Is(err, gobiernoconvocatorias.ErrResultadoBorradorInseguro) {
		t.Fatalf("recibo sin acreditacion aceptado: %v", err)
	}
}

func TestErrorDiarioPostgreSQLDistingueAmbiguedadContencionYManipulacion(t *testing.T) {
	casos := []struct {
		codigo string
		espera error
	}{
		{"21000", gobiernoconvocatorias.ErrConsultaIdempotenciaAmbigua},
		{"40001", gobiernoconvocatorias.ErrOperacionBorradorEnCurso},
		{"42501", gobiernoconvocatorias.ErrResultadoBorradorInseguro},
		{"08006", gobiernoconvocatorias.ErrOperacionBorradorIndeterminada},
	}
	for _, caso := range casos {
		err := errorDiarioPostgreSQL(context.Background(), &pgconn.PgError{Code: caso.codigo})
		if !errors.Is(err, caso.espera) {
			t.Fatalf("codigo %s: obtenido %v, esperado %v", caso.codigo, err, caso.espera)
		}
	}
}

func solicitudConsultaDiarioPostgreSQLPrueba(
	t *testing.T,
) gobiernoconvocatorias.SolicitudConsultaIdentidadesBorrador {
	t.Helper()
	identidad, err := restaurarIdentidadDiarioPostgreSQL(identidadDiarioPostgreSQLPrueba(t))
	if err != nil {
		t.Fatal(err)
	}
	solicitud := gobiernoconvocatorias.SolicitudConsultaIdentidadesBorrador{
		Identidades:  []gobiernoconvocatorias.ProyeccionIdentidadOperacion{identidad},
		SolicitadaEn: instanteDiarioPostgreSQLPrueba,
	}
	if (gobiernoconvocatorias.ResultadoConsultaIdentidadesBorrador{}).ValidarPara(solicitud) != nil {
		t.Fatal("fixture de consulta invalida")
	}
	return solicitud
}

func identidadDiarioPostgreSQLPrueba(t *testing.T) identidadDiarioPostgreSQL {
	t.Helper()
	return identidadDiarioPostgreSQL{
		Localizador: hmacDiarioPostgreSQL{
			VersionEsquema: 1, Dominio: "localizador",
			ClaveRef: "clave:hmac:convocatorias:localizador:prueba", GeneracionClave: 3,
			HMACSHA256: strings.Repeat("a", 64),
		},
		HuellaSolicitud: hmacDiarioPostgreSQL{
			VersionEsquema: 1, Dominio: "huella_solicitud",
			ClaveRef: "clave:hmac:convocatorias:huella:prueba", GeneracionClave: 3,
			HMACSHA256: strings.Repeat("b", 64),
		},
	}
}

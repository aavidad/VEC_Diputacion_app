package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type revalidadorConfirmacionPostgreSQLPrueba struct {
	identidad gobiernoconvocatorias.IdentidadAutoridadBorrador
}

func (r *revalidadorConfirmacionPostgreSQLPrueba) IdentidadAutoridadBorrador() gobiernoconvocatorias.IdentidadAutoridadBorrador {
	return r.identidad
}

func (*revalidadorConfirmacionPostgreSQLPrueba) RevalidarAtestacionKMS(
	context.Context,
	gobiernoconvocatorias.SolicitudRevalidacionAtestacionKMSBorrador,
) (gobiernoconvocatorias.ResultadoRevalidacionAtestacionKMSBorrador, error) {
	return gobiernoconvocatorias.ResultadoRevalidacionAtestacionKMSBorrador{}, errors.New("no invocable")
}

type verificadorCriptograficoPostgreSQLPrueba struct {
	identidad gobiernoconvocatorias.IdentidadAutoridadBorrador
}

func (v *verificadorCriptograficoPostgreSQLPrueba) IdentidadAutoridadBorrador() gobiernoconvocatorias.IdentidadAutoridadBorrador {
	return v.identidad
}

func (*verificadorCriptograficoPostgreSQLPrueba) VerificarEvidenciasRecibo(
	context.Context, gobiernoconvocatorias.ProyeccionReciboBorrador,
) error {
	return nil
}

type revalidadorBloqueadoPostgreSQLPrueba struct {
	identidad gobiernoconvocatorias.IdentidadAutoridadBorrador
	iniciado  chan struct{}
}

func (r *revalidadorBloqueadoPostgreSQLPrueba) IdentidadAutoridadBorrador() gobiernoconvocatorias.IdentidadAutoridadBorrador {
	return r.identidad
}

func (r *revalidadorBloqueadoPostgreSQLPrueba) RevalidarAtestacionKMS(
	ctx context.Context,
	_ gobiernoconvocatorias.SolicitudRevalidacionAtestacionKMSBorrador,
) (gobiernoconvocatorias.ResultadoRevalidacionAtestacionKMSBorrador, error) {
	select {
	case <-r.iniciado:
	default:
		close(r.iniciado)
	}
	<-ctx.Done()
	return gobiernoconvocatorias.ResultadoRevalidacionAtestacionKMSBorrador{}, ctx.Err()
}

type verificadorCriptograficoBloqueadoPostgreSQLPrueba struct {
	identidad   gobiernoconvocatorias.IdentidadAutoridadBorrador
	transaccion *transaccionDiarioPostgreSQLPrueba
	iniciado    chan struct{}
	vioCommit   bool
}

func (v *verificadorCriptograficoBloqueadoPostgreSQLPrueba) IdentidadAutoridadBorrador() gobiernoconvocatorias.IdentidadAutoridadBorrador {
	return v.identidad
}

func (v *verificadorCriptograficoBloqueadoPostgreSQLPrueba) VerificarEvidenciasRecibo(
	ctx context.Context, _ gobiernoconvocatorias.ProyeccionReciboBorrador,
) error {
	v.vioCommit = v.transaccion != nil && v.transaccion.cerrada && v.transaccion.confirmaciones == 1
	if v.iniciado != nil {
		select {
		case <-v.iniciado:
		default:
			close(v.iniciado)
		}
	}
	<-ctx.Done()
	return ctx.Err()
}

func identidadConfirmacionPostgreSQLPrueba(sufijo string) gobiernoconvocatorias.IdentidadAutoridadBorrador {
	identidad, _ := gobiernoconvocatorias.NuevaIdentidadAutoridadBorrador(
		"proveedor-"+sufijo, "instancia-"+sufijo, "credencial-"+sufijo, "rol-"+sufijo,
	)
	return identidad
}

func TestConstructoresConfirmacionPostgreSQLExigenAutoridadesYCredencialesSeparadas(t *testing.T) {
	pool := &iniciadorDiarioPostgreSQLPrueba{}
	confirmadorID := identidadConfirmacionPostgreSQLPrueba("confirmador")
	revalidadorID := identidadConfirmacionPostgreSQLPrueba("revalidador")
	revalidador := &revalidadorConfirmacionPostgreSQLPrueba{identidad: revalidadorID}
	verificadorID := identidadConfirmacionPostgreSQLPrueba("verificador-db")
	criptografiaID := identidadConfirmacionPostgreSQLPrueba("verificador-criptografico")
	verificador, err := nuevoVerificadorReciboBorradorPostgreSQL(
		pool, &verificadorCriptograficoPostgreSQLPrueba{identidad: criptografiaID}, verificadorID,
	)
	if err != nil || verificador.IdentidadAutoridadBorrador() != verificadorID {
		t.Fatalf("verificador separado rechazado: verificador=%v err=%v", verificador, err)
	}
	confirmador, err := nuevoConfirmadorAtomicoBorradorPostgreSQL(
		pool, revalidador, confirmadorID, verificador,
		esperarConfirmacionBorradorPostgreSQL,
	)
	if err != nil || confirmador == nil {
		t.Fatalf("confirmador separado rechazado: confirmador=%v err=%v", confirmador, err)
	}
	vinculoEsperado, errVinculo := gobiernoconvocatorias.NuevoVinculoVerificadorReciboBorrador(
		verificadorID, criptografiaID,
	)
	referenciaEsperada, errReferenciaEsperada := vinculoEsperado.ReferenciaParaAcreditacion()
	referenciaReal, errReferenciaReal := confirmador.VinculoVerificadorReciboBorrador().ReferenciaParaAcreditacion()
	if confirmador.IdentidadAutoridadBorrador() != confirmadorID ||
		errVinculo != nil || errReferenciaEsperada != nil || errReferenciaReal != nil ||
		referenciaReal != referenciaEsperada {
		t.Fatalf("confirmador separado rechazado: confirmador=%v err=%v", confirmador, err)
	}
	if _, err := nuevoConfirmadorAtomicoBorradorPostgreSQL(
		pool, &revalidadorConfirmacionPostgreSQLPrueba{identidad: confirmadorID},
		confirmadorID, verificador, esperarConfirmacionBorradorPostgreSQL,
	); !errors.Is(err, gobiernoconvocatorias.ErrServicioBorradoresInvalido) {
		t.Fatalf("misma credencial A/KMS aceptada: %v", err)
	}
	if _, err := nuevoVerificadorReciboBorradorPostgreSQL(
		pool, &verificadorCriptograficoPostgreSQLPrueba{identidad: verificadorID}, verificadorID,
	); !errors.Is(err, gobiernoconvocatorias.ErrServicioBorradoresInvalido) {
		t.Fatalf("misma credencial DB/cripto aceptada: %v", err)
	}
	for nombre, identidadKMS := range map[string]gobiernoconvocatorias.IdentidadAutoridadBorrador{
		"kms igual a db verificadora":         verificadorID,
		"kms igual a autoridad criptografica": criptografiaID,
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := nuevoConfirmadorAtomicoBorradorPostgreSQL(
				pool, &revalidadorConfirmacionPostgreSQLPrueba{identidad: identidadKMS},
				confirmadorID, verificador, esperarConfirmacionBorradorPostgreSQL,
			); !errors.Is(err, gobiernoconvocatorias.ErrServicioBorradoresInvalido) {
				t.Fatalf("autoridades KMS/verificador no separadas aceptadas: %v", err)
			}
		})
	}
}

func TestConstructoresConfirmacionPostgreSQLRechazanNulosTipados(t *testing.T) {
	pool := &iniciadorDiarioPostgreSQLPrueba{}
	confirmadorID := identidadConfirmacionPostgreSQLPrueba("confirmador-nulos")
	verificadorID := identidadConfirmacionPostgreSQLPrueba("verificador-nulos")
	criptografiaID := identidadConfirmacionPostgreSQLPrueba("cripto-nulos")
	verificador, err := nuevoVerificadorReciboBorradorPostgreSQL(
		pool, &verificadorCriptograficoPostgreSQLPrueba{identidad: criptografiaID}, verificadorID,
	)
	if err != nil {
		t.Fatal(err)
	}
	var revalidadorNulo *revalidadorConfirmacionPostgreSQLPrueba
	if _, err := nuevoConfirmadorAtomicoBorradorPostgreSQL(
		pool, revalidadorNulo, confirmadorID, verificador,
		esperarConfirmacionBorradorPostgreSQL,
	); !errors.Is(err, gobiernoconvocatorias.ErrServicioBorradoresInvalido) {
		t.Fatalf("revalidador KMS nulo tipado aceptado: %v", err)
	}
	var verificadorNulo *VerificadorReciboBorradorPostgreSQL
	if _, err := nuevoConfirmadorAtomicoBorradorPostgreSQL(
		pool,
		&revalidadorConfirmacionPostgreSQLPrueba{
			identidad: identidadConfirmacionPostgreSQLPrueba("kms-nulos"),
		},
		confirmadorID, verificadorNulo, esperarConfirmacionBorradorPostgreSQL,
	); !errors.Is(err, gobiernoconvocatorias.ErrServicioBorradoresInvalido) {
		t.Fatalf("verificador nulo tipado aceptado: %v", err)
	}
	var criptografiaNula *verificadorCriptograficoPostgreSQLPrueba
	if _, err := nuevoVerificadorReciboBorradorPostgreSQL(
		pool, criptografiaNula, verificadorID,
	); !errors.Is(err, gobiernoconvocatorias.ErrServicioBorradoresInvalido) {
		t.Fatalf("autoridad criptografica nula tipada aceptada: %v", err)
	}
}

func TestReciboDebeAcreditarElVinculoCompletoDelVerificadorPostgreSQL(t *testing.T) {
	persistenciaA := identidadConfirmacionPostgreSQLPrueba("db-vinculo-a")
	criptografiaA := identidadConfirmacionPostgreSQLPrueba("cripto-vinculo-a")
	vinculoA, err := gobiernoconvocatorias.NuevoVinculoVerificadorReciboBorrador(
		persistenciaA, criptografiaA,
	)
	if err != nil {
		t.Fatal(err)
	}
	referenciaA, err := vinculoA.ReferenciaParaAcreditacion()
	if err != nil {
		t.Fatal(err)
	}
	recibo := gobiernoconvocatorias.ProyeccionReciboBorrador{
		AcreditacionKMS: gobiernoconvocatorias.AcreditacionKMSConfirmacionBorrador{
			VerificadorRef: referenciaA,
		},
	}
	if !reciboAcreditaVinculoVerificadorPostgreSQL(vinculoA, recibo) {
		t.Fatal("recibo ligado al verificador exacto rechazado")
	}
	vinculoB, err := gobiernoconvocatorias.NuevoVinculoVerificadorReciboBorrador(
		identidadConfirmacionPostgreSQLPrueba("db-vinculo-b"),
		identidadConfirmacionPostgreSQLPrueba("cripto-vinculo-b"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if reciboAcreditaVinculoVerificadorPostgreSQL(vinculoB, recibo) {
		t.Fatal("recibo A aceptado por verificador B")
	}
	if reciboAcreditaVinculoVerificadorPostgreSQL(
		gobiernoconvocatorias.VinculoVerificadorReciboBorrador{}, recibo,
	) {
		t.Fatal("vinculo cero aceptado")
	}
}

func TestTransaccionesConfirmacionPostgreSQLSonSerializableYConfiguranSesion(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		modo   pgx.TxAccessMode
	}{
		{"confirmacion", pgx.ReadWrite},
		{"verificacion", pgx.ReadOnly},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			tx := &transaccionDiarioPostgreSQLPrueba{}
			pool := &iniciadorDiarioPostgreSQLPrueba{tx: tx}
			obtenida, err := iniciarTransaccionBorradorPostgreSQL(context.Background(), pool, caso.modo)
			if err != nil {
				t.Fatal(err)
			}
			if pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != caso.modo ||
				tx.configuraciones != 1 {
				t.Fatalf("frontera débil: opciones=%+v configuraciones=%d", pool.opciones, tx.configuraciones)
			}
			_ = obtenida.Rollback(context.Background())
		})
	}
}

func TestWrappersConfirmacionPostgreSQLInvocanSoloFuncionesPublicas(t *testing.T) {
	errFila := errors.New("fin de prueba")
	tx := &transaccionDiarioPostgreSQLPrueba{fila: filaErrorDiarioPostgreSQLPrueba{err: errFila}}
	carga := cargaConfirmacionBorradorPostgreSQL{
		Confirmacion: []byte(`{}`), Prueba: []byte(`{}`), Evidencia: []byte(`{}`),
		Decision: []byte(`{}`), Contexto: []byte(`{}`), Material: []byte(`{}`),
		Version: []byte(`{}`), AAD: []byte(`{}`), MaterialEnvuelto: []byte("0123456789abcdef"),
		Nonce: []byte("012345678901"), TextoCifrado: []byte("0123456789abcdef"),
	}
	if _, err := prepararConfirmacionBorradorPostgreSQL(context.Background(), tx, carga); !errors.Is(err, errFila) {
		t.Fatalf("error de fase A alterado: %v", err)
	}
	if !strings.Contains(tx.consulta, funcionPrepararConfirmacionBorradorPostgreSQL) ||
		!strings.Contains(tx.consulta, "preparada_en") || len(tx.argumentos) != 11 {
		t.Fatalf("fase A no usa la capacidad cerrada: consulta=%q args=%d", tx.consulta, len(tx.argumentos))
	}
	if _, err := ejecutarFaseBConfirmacionBorradorPostgreSQL(
		context.Background(), tx, "preparacion", []byte(`{}`), []byte(`{}`),
	); !errors.Is(err, errFila) {
		t.Fatalf("error de fase B alterado: %v", err)
	}
	if !strings.Contains(tx.consulta, funcionConfirmarBorradorPostgreSQL) || len(tx.argumentos) != 3 {
		t.Fatalf("fase B no usa la capacidad cerrada: consulta=%q args=%d", tx.consulta, len(tx.argumentos))
	}
	if _, err := releerReciboBorradorPostgreSQL(
		context.Background(), tx,
		gobiernoconvocatorias.ProyeccionReciboBorrador{ReciboRef: "recibo", TransaccionRef: "tx"},
		[]byte(`{}`),
	); !errors.Is(err, errFila) {
		t.Fatalf("error de verificacion alterado: %v", err)
	}
	if !strings.Contains(tx.consulta, funcionVerificarReciboBorradorPostgreSQL) ||
		!strings.Contains(tx.consulta, "sha256(convert_to($3::jsonb::text") || len(tx.argumentos) != 3 {
		t.Fatalf("relectura no calcula identidad JSONB en PostgreSQL: %q", tx.consulta)
	}
}

func TestVerificadorPostgreSQLCotejaVersionDurableConRecibo(t *testing.T) {
	recibo := gobiernoconvocatorias.ProyeccionReciboBorrador{
		EstadoPrincipal: puertosbolsa.ReferenciaEstadoVersionConvocatoria{
			Referencia: "convocatoria-001#3", Revision: 7,
		},
	}
	if !metadatosVersionReciboCoinciden("convocatoria-001", 3, 7, recibo) {
		t.Fatal("la version durable exacta fue rechazada")
	}
	for _, caso := range []struct {
		id        string
		secuencia uint64
		revision  uint64
	}{
		{"convocatoria-ajena", 3, 7},
		{"convocatoria-001", 4, 7},
		{"convocatoria-001", 3, 8},
	} {
		if metadatosVersionReciboCoinciden(caso.id, caso.secuencia, caso.revision, recibo) {
			t.Fatalf("metadatos SQL ajenos aceptados: %+v", caso)
		}
	}
}

func TestConfirmacionPostgreSQLDistingueRollbackProbadoYCommitAmbiguo(t *testing.T) {
	txRollback := &transaccionDiarioPostgreSQLPrueba{}
	resultado, err := cerrarPorRollbackBorradorPostgreSQL(
		context.Background(), txRollback, gobiernoconvocatorias.ErrResultadoBorradorInseguro,
	)
	if resultado.Estado != gobiernoconvocatorias.ResultadoDiarioNoAplicado ||
		!errors.Is(err, gobiernoconvocatorias.ErrResultadoBorradorInseguro) || txRollback.reversiones != 1 {
		t.Fatalf("rollback demostrado perdió semántica: resultado=%+v err=%v tx=%+v", resultado, err, txRollback)
	}
	txCommit := &transaccionDiarioPostgreSQLPrueba{errorCommit: errors.New("respuesta perdida")}
	resultado, err = confirmarCommitBorradorPostgreSQL(
		context.Background(), txCommit, resultadoNoAplicadoBorradorPostgreSQL(),
	)
	if resultado.Estado != gobiernoconvocatorias.ResultadoDiarioIndeterminado ||
		!errors.Is(err, gobiernoconvocatorias.ErrOperacionBorradorIndeterminada) {
		t.Fatalf("commit ambiguo se presentó como rollback: resultado=%+v err=%v", resultado, err)
	}
	txCommitRollback := &transaccionDiarioPostgreSQLPrueba{errorCommit: pgx.ErrTxCommitRollback}
	resultado, err = confirmarCommitBorradorPostgreSQL(
		context.Background(), txCommitRollback, resultadoIndeterminadoBorradorPostgreSQL(),
	)
	if resultado.Estado != gobiernoconvocatorias.ResultadoDiarioNoAplicado ||
		!errors.Is(err, gobiernoconvocatorias.ErrResultadoBorradorInseguro) {
		t.Fatalf("commit convertido en rollback no reconocido: resultado=%+v err=%v", resultado, err)
	}
}

func TestFilaNoAplicadaPostgreSQLSoloAceptaParCerradoYControlesDurables(t *testing.T) {
	fila := filaConfirmacionBorradorPostgreSQL{
		resultado: "conflicto_cas", estadoDiario: string(gobiernoconvocatorias.ResultadoDiarioNoAplicado),
		revision: pgtype.Int8{Int64: 2, Valid: true}, cercado: pgtype.Int8{Int64: 2, Valid: true},
		accion:       pgtype.Text{String: "accion", Valid: true},
		confirmadaEn: pgtype.Timestamptz{Time: time.Now().UTC().Truncate(time.Microsecond), Valid: true},
	}
	resultado, err := fila.restaurarSinPreparacion()
	if err != nil || resultado.Estado != gobiernoconvocatorias.ResultadoDiarioNoAplicado {
		t.Fatalf("no aplicado durable rechazado: resultado=%+v err=%v", resultado, err)
	}
	fila.resultado = "resultado-inventado"
	if _, err := fila.restaurarSinPreparacion(); !errors.Is(err, gobiernoconvocatorias.ErrResultadoBorradorInseguro) {
		t.Fatalf("resultado abierto aceptado: %v", err)
	}
	fila.resultado = "conflicto_cas"
	fila.transaccionRef = pgtype.Text{String: "metadato-prohibido", Valid: true}
	if _, err := fila.restaurarSinPreparacion(); !errors.Is(err, gobiernoconvocatorias.ErrResultadoBorradorInseguro) {
		t.Fatalf("metadato terminal espurio aceptado: %v", err)
	}
}

func TestConfirmacionPostgreSQLRespetaCancelacionAntesDeTocarPool(t *testing.T) {
	pool := &iniciadorDiarioPostgreSQLPrueba{}
	confirmadorID := identidadConfirmacionPostgreSQLPrueba("confirmador-cancelado")
	revalidador := &revalidadorConfirmacionPostgreSQLPrueba{
		identidad: identidadConfirmacionPostgreSQLPrueba("revalidador-cancelado"),
	}
	verificador, err := nuevoVerificadorReciboBorradorPostgreSQL(
		pool,
		&verificadorCriptograficoPostgreSQLPrueba{
			identidad: identidadConfirmacionPostgreSQLPrueba("cripto-cancelado"),
		},
		identidadConfirmacionPostgreSQLPrueba("verificador-cancelado"),
	)
	if err != nil {
		t.Fatal(err)
	}
	confirmador, err := nuevoConfirmadorAtomicoBorradorPostgreSQL(
		pool, revalidador, confirmadorID, verificador,
		esperarConfirmacionBorradorPostgreSQL,
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	resultado, err := confirmador.ConfirmarBorrador(
		ctx, gobiernoconvocatorias.SolicitudConfirmacionBorrador{},
	)
	if !errors.Is(err, context.Canceled) || resultado.Estado != gobiernoconvocatorias.ResultadoDiarioNoAplicado ||
		pool.inicios != 0 {
		t.Fatalf("cancelacion no fue fail-fast: resultado=%+v err=%v inicios=%d", resultado, err, pool.inicios)
	}
}

func TestRevalidacionKMSBloqueadaAgotaPlazoYPermiteRollback(t *testing.T) {
	preparadaEn := time.Now().UTC().Truncate(time.Microsecond)
	confirmadaEn := preparadaEn.Add(duracionMinimaVentanaKMSPostgreSQL)
	revalidador := &revalidadorBloqueadoPostgreSQLPrueba{
		identidad: identidadConfirmacionPostgreSQLPrueba("kms-bloqueado"),
		iniciado:  make(chan struct{}),
	}
	inicio := time.Now()
	_, err := revalidarAtestacionKMSConPlazoPostgreSQL(
		context.Background(), confirmadaEn, revalidador,
		gobiernoconvocatorias.SolicitudRevalidacionAtestacionKMSBorrador{SolicitadaEn: preparadaEn},
	)
	if !errors.Is(err, gobiernoconvocatorias.ErrOperacionBorradorEnCurso) ||
		time.Since(inicio) > time.Second {
		t.Fatalf("KMS bloqueado no quedo acotado: err=%v duracion=%s", err, time.Since(inicio))
	}
	tx := &transaccionDiarioPostgreSQLPrueba{}
	resultado, err := cerrarPorRollbackBorradorPostgreSQL(context.Background(), tx, err)
	if resultado.Estado != gobiernoconvocatorias.ResultadoDiarioNoAplicado ||
		!errors.Is(err, gobiernoconvocatorias.ErrOperacionBorradorEnCurso) || tx.reversiones != 1 {
		t.Fatalf("timeout KMS no libero la transaccion: resultado=%+v err=%v tx=%+v", resultado, err, tx)
	}
}

func TestVentanaRevalidacionKMSExigePreparacionRealSeparadaDelNotBefore(t *testing.T) {
	preparadaEn := time.Now().UTC().Truncate(time.Microsecond)
	if !ventanaRevalidacionKMSPostgreSQLValida(
		preparadaEn, preparadaEn.Add(duracionMinimaVentanaKMSPostgreSQL),
	) {
		t.Fatal("ventana minima gobernada rechazada")
	}
	for _, confirmadaEn := range []time.Time{
		preparadaEn,
		preparadaEn.Add(-time.Microsecond),
		preparadaEn.Add(duracionMinimaVentanaKMSPostgreSQL - time.Microsecond),
		preparadaEn.Add(duracionMaximaVentanaKMSPostgreSQL + time.Microsecond),
	} {
		if ventanaRevalidacionKMSPostgreSQLValida(preparadaEn, confirmadaEn) {
			t.Fatalf("instante preparado confundido con not-before: preparada=%s confirmada=%s", preparadaEn, confirmadaEn)
		}
	}
}

func TestRevalidacionKMSBloqueadaRespetaCancelacionSuperior(t *testing.T) {
	preparadaEn := time.Now().UTC().Truncate(time.Microsecond)
	revalidador := &revalidadorBloqueadoPostgreSQLPrueba{
		identidad: identidadConfirmacionPostgreSQLPrueba("kms-cancelado"),
		iniciado:  make(chan struct{}),
	}
	ctx, cancelar := context.WithCancel(context.Background())
	hecho := make(chan error, 1)
	go func() {
		_, err := revalidarAtestacionKMSConPlazoPostgreSQL(
			ctx, preparadaEn.Add(time.Second), revalidador,
			gobiernoconvocatorias.SolicitudRevalidacionAtestacionKMSBorrador{SolicitadaEn: preparadaEn},
		)
		hecho <- err
	}()
	<-revalidador.iniciado
	cancelar()
	if err := <-hecho; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion superior ocultada: %v", err)
	}
}

func TestVerificacionCriptograficaBloqueadaOcurreTrasCommitYConPlazo(t *testing.T) {
	tx := &transaccionDiarioPostgreSQLPrueba{}
	criptografia := &verificadorCriptograficoBloqueadoPostgreSQLPrueba{
		identidad:   identidadConfirmacionPostgreSQLPrueba("cripto-bloqueada"),
		transaccion: tx,
	}
	err := confirmarRelecturaYVerificarCriptografiaPostgreSQL(
		context.Background(), tx, criptografia,
		gobiernoconvocatorias.ProyeccionReciboBorrador{}, 20*time.Millisecond,
	)
	if !errors.Is(err, gobiernoconvocatorias.ErrOperacionBorradorEnCurso) ||
		!criptografia.vioCommit || tx.reversiones != 0 {
		t.Fatalf("cripto no quedo fuera del snapshot: err=%v vio_commit=%v tx=%+v", err, criptografia.vioCommit, tx)
	}
}

func TestVerificacionCriptograficaBloqueadaRespetaCancelacionSuperior(t *testing.T) {
	tx := &transaccionDiarioPostgreSQLPrueba{}
	criptografia := &verificadorCriptograficoBloqueadoPostgreSQLPrueba{
		identidad:   identidadConfirmacionPostgreSQLPrueba("cripto-cancelada"),
		transaccion: tx,
		iniciado:    make(chan struct{}),
	}
	ctx, cancelar := context.WithCancel(context.Background())
	hecho := make(chan error, 1)
	go func() {
		hecho <- confirmarRelecturaYVerificarCriptografiaPostgreSQL(
			ctx, tx, criptografia, gobiernoconvocatorias.ProyeccionReciboBorrador{}, time.Second,
		)
	}()
	<-criptografia.iniciado
	cancelar()
	if err := <-hecho; !errors.Is(err, context.Canceled) || !criptografia.vioCommit {
		t.Fatalf("cancelacion cripto no libero tras commit: err=%v vio_commit=%v", err, criptografia.vioCommit)
	}
}

func TestConfirmacionPostgreSQLInvalidaAntesDelPoolNoQuedaIndeterminada(t *testing.T) {
	resultado, err := (*ConfirmadorAtomicoBorradorPostgreSQL)(nil).ConfirmarBorrador(
		context.Background(), gobiernoconvocatorias.SolicitudConfirmacionBorrador{},
	)
	if resultado.Estado != gobiernoconvocatorias.ResultadoDiarioNoAplicado ||
		!errors.Is(err, gobiernoconvocatorias.ErrServicioBorradoresInvalido) {
		t.Fatalf("fallo previo a cualquier efecto marcado como incierto: resultado=%+v err=%v", resultado, err)
	}
}

func TestJSONPersistenciaConfirmacionPostgreSQLEsCerradoYSnakeCase(t *testing.T) {
	politica, err := json.Marshal(politicaCifradoBorradorPostgreSQL{})
	if err != nil {
		t.Fatal(err)
	}
	exigirClavesJSONConfirmacionPrueba(t, politica, []string{
		"accion", "arrendamiento_inicia_en", "arrendamiento_vence_en", "autoridad_ref",
		"catalogo_ref", "cercado", "decision_politica_ref", "emitida_en", "esquema",
		"estado", "huella_catalogo_sha256", "huella_decision_sha256",
		"huella_material_sha256", "identidad_primaria", "perfil", "revision",
		"revision_catalogo", "solicitada_en", "valida_hasta", "verificada_en",
		"version_decision_politica",
	})
	evidencia, err := json.Marshal(evidenciaPerfilBorradorPostgreSQL{})
	if err != nil {
		t.Fatal(err)
	}
	exigirClavesJSONConfirmacionPrueba(t, evidencia, []string{
		"accion", "arrendamiento_inicia_en", "arrendamiento_vence_en", "catalogo_ref",
		"cercado", "decision_politica_ref", "emitida_en", "esquema", "estado",
		"evidencia_ref", "huella_catalogo_sha256", "huella_decision_politica_sha256",
		"huella_evidencia_sha256", "huella_material_sha256", "identidad_primaria",
		"perfil", "revision", "revision_catalogo", "solicitud_resolucion_en",
		"valida_hasta", "verificada_en", "verificador_ref", "version_decision_politica",
		"version_evidencia",
	})
}

func exigirClavesJSONConfirmacionPrueba(t *testing.T, contenido []byte, esperadas []string) {
	t.Helper()
	var objeto map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &objeto); err != nil {
		t.Fatal(err)
	}
	if len(objeto) != len(esperadas) {
		t.Fatalf("objeto no cerrado: claves=%v esperadas=%v", objeto, esperadas)
	}
	for _, clave := range esperadas {
		if _, existe := objeto[clave]; !existe {
			t.Fatalf("falta clave canónica %q en %s", clave, contenido)
		}
	}
}

func TestEsperaConfirmacionPostgreSQLCancelaSinConsumirPlazo(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	inicio := time.Now()
	err := esperarConfirmacionBorradorPostgreSQL(ctx, inicio.Add(time.Minute))
	if !errors.Is(err, context.Canceled) || time.Since(inicio) > time.Second {
		t.Fatalf("espera no respetó contexto: err=%v duracion=%s", err, time.Since(inicio))
	}
}

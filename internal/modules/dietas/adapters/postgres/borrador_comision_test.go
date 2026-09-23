package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type filaBorradorPrueba struct {
	tx     *txBorradorPrueba
	salida []byte
	err    error
}

func (f filaBorradorPrueba) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(dest) == 1 {
		*dest[0].(*[]byte) = append([]byte(nil), f.salida...)
		return nil
	}
	return errors.New("scan inesperado")
}

type txBorradorPrueba struct {
	pgx.Tx
	consultas           []string
	argumentos          [][]any
	commits, rollbacks  int
	salida              []byte
	errSQL, errorCommit error
}

func (t *txBorradorPrueba) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	t.consultas = append(t.consultas, q)
	t.argumentos = append(t.argumentos, args)
	return filaBorradorPrueba{tx: t, salida: t.salida, err: t.errSQL}
}
func (t *txBorradorPrueba) Commit(context.Context) error   { t.commits++; return t.errorCommit }
func (t *txBorradorPrueba) Rollback(context.Context) error { t.rollbacks++; return nil }

type poolBorradorPrueba struct {
	tx       pgx.Tx
	opciones pgx.TxOptions
	inicios  int
	err      error
}

func (p *poolBorradorPrueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	p.inicios++
	p.opciones = o
	return p.tx, p.err
}

func identidadBorradorPrueba(t *testing.T) dietasports.IdentidadEfectivaBorrador {
	t.Helper()
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "contexto:prueba", strings.Repeat("c", 64), "dietas.borrador.propio.crear", "efecto:prueba", strings.Repeat("d", 64), "vec-interno-corporativo", time.Date(2026, 9, 20, 8, 0, 0, 0, time.UTC), time.Date(2026, 9, 20, 8, 0, 3, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	// SPKI Ed25519 público sintético; la prueba no crea ni conserva claves privadas.
	raiz, err := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	if err != nil {
		t.Fatal(err)
	}
	material, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	contexto := contextoRegistradoBorradorPrueba(t)
	if err := contexto.Validar(); err != nil {
		t.Fatalf("contexto registrado inválido: %v", err)
	}
	s := strings.TrimPrefix(contexto.Contexto.PersonaRef, "per_")
	relacion := dietasports.RelacionServicioAcreditada{RelacionRef: "rel_" + s, PersonaRef: contexto.Contexto.PersonaRef, EmpleadoRef: "emp_" + s, UnidadRef: "unidad:prueba", VigenteDesde: "2026-09-01", Version: 3, ProcedenciaActoRef: "acto:prueba", FuenteRef: "fuente:prueba", FuenteVersion: 4}
	return dietasports.IdentidadEfectivaBorrador{ContextoRegistrado: contexto, Relacion: relacion, Autorizacion: dietasports.AutorizacionBorradorDurable{Material: material, Accion: "dietas.borrador.propio.crear", RecursoRef: "dietas:borradores:propios", Finalidad: "crear_borrador_propio", Revalidacion: dietasports.RevalidacionRelacionPersonal{RelacionRef: relacion.RelacionRef, PersonaRef: relacion.PersonaRef, EmpleadoRef: relacion.EmpleadoRef, UnidadRef: relacion.UnidadRef, VigenteDesde: relacion.VigenteDesde, VigenteHasta: relacion.VigenteHasta, FechaReferencia: "2026-09-21", Version: 3, ProcedenciaActoRef: relacion.ProcedenciaActoRef, FuenteRef: relacion.FuenteRef, FuenteVersion: 4}}}
}

func identidadConsultaBorradorPrueba(t *testing.T, recurso string) dietasports.IdentidadEfectivaBorrador {
	i := identidadBorradorPrueba(t)
	i.Autorizacion.Accion = "dietas.borrador.propio.consultar"
	i.Autorizacion.RecursoRef = recurso
	i.Autorizacion.Finalidad = "consultar_borrador_propio"
	return i
}

func contextoRegistradoBorradorPrueba(t *testing.T) vecdomain.ResultadoContextoActorRegistradoV2 {
	t.Helper()
	ahora := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	s := strings.Repeat("a", 24)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + s, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	instantanea := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_" + s, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + s, PersonaVersion: 1, PerfilActivoRef: "prf_" + s, PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), Vinculos: []vecdomain.VinculoReferenciaContextoActor{{VinculoRef: "vin_" + s, Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: "emp_" + s, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}}}
	actor, err := vecdomain.NuevoContextoActor(cuenta, instantanea, ahora)
	if err != nil {
		t.Fatal(err)
	}
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	acreditacion := vecdomain.AcreditacionProcedenciaComponenteContextoActorV1{ProcedenciaRef: "prc_" + s, ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("a", 64), ProcedenciaAutoridad: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1}
	manifiesto := vecdomain.ManifiestoProcedenciaContextoActorV1{Esquema: vecdomain.EsquemaManifiestoProcedenciaContextoActorV1, AutoridadEfectiva: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, Cuenta: vecdomain.ProcedenciaCuentaContextoActorV1{CuentaRef: cuenta.CuentaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion}, Persona: vecdomain.ProcedenciaPersonaContextoActorV1{PersonaRef: actor.PersonaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion}, Perfil: vecdomain.ProcedenciaPerfilContextoActorV1{PerfilRef: actor.PerfilActivoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion}, Contexto: vecdomain.ProcedenciaVinculoContextoActorV1{VinculoRef: instantanea.VinculoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion}, Vinculos: []vecdomain.ProcedenciaVinculoReferenciaContextoActorV1{{VinculoRef: "vin_" + s, Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: "emp_" + s, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion}}}
	bytesManifiesto, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	huellaManifiesto, err := vecdomain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(bytesManifiesto)
	if err != nil {
		t.Fatal(err)
	}
	return vecdomain.ResultadoContextoActorRegistradoV2{RegistroContextoRef: "rca_" + s, Contexto: actor, RepresentacionCanonica: canon, HuellaSHA256: huella, ManifiestoProcedenciaCanonico: bytesManifiesto, ManifiestoProcedenciaHuellaSHA256: huellaManifiesto, AutoridadEfectiva: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, ResueltoEnAutoritativo: ahora}
}
func salidaBorradorPrueba(t *testing.T) []byte {
	return salidaBorradorConInstantePrueba(t, "2026-09-20T10:00:00.000000Z")
}
func salidaBorradorConInstantePrueba(t *testing.T, instante string) []byte {
	t.Helper()
	c, err := domain.NuevaComisionBorrador("dco_"+strings.Repeat("b", 22), "2026-09-21", "2026-09-21", "Visita técnica", "rel_"+strings.Repeat("a", 22), []string{"GR:001"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(map[string]any{"comision": c, "recibo": map[string]any{"referencia": "rcd_11111111-1111-1111-1111-111111111111", "version": 1, "registrado_en": instante, "repeticion": false}})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func salidaBorradorNoEncontradoPrueba(t *testing.T) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]string{"resultado": "no_encontrado"})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCrearBorradorUsaFuncionDietasNominalYConfirmaAntesDeExponer(t *testing.T) {
	tx := &txBorradorPrueba{salida: salidaBorradorPrueba(t)}
	pool := &poolBorradorPrueba{tx: tx}
	repo, err := nuevoRepositorioBorradorComisionPostgreSQL(pool)
	if err != nil {
		t.Fatal(err)
	}
	identidad := identidadBorradorPrueba(t)
	solicitud := dietasports.SolicitudCrearBorradorPropio{ClaveIdempotencia: "clave_idempotente_0001", FechaInicio: "2026-09-21", FechaFin: "2026-09-21", Motivo: "Visita técnica"}
	resultado, err := repo.CrearORecuperar(context.Background(), identidad, solicitud)
	if err != nil || resultado.Recibo.Referencia == "" || pool.inicios != 1 || pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != pgx.ReadWrite || tx.commits != 1 || len(tx.consultas) != 1 || tx.consultas[0] != crearORecuperarBorradorSQL {
		t.Fatalf("flujo durable alterado: err=%v tx=%#v", err, tx)
	}
	efecto, err := efectoCrearBorrador(identidad, solicitud)
	if err != nil || len(tx.argumentos[0]) != 11 || !bytes.Equal([]byte(tx.argumentos[0][0].(string)), efecto.Material) {
		t.Fatal("material o AD3 fuera de contrato")
	}
}
func TestCrearBorradorNoExponeReciboSiCommitEsIncierto(t *testing.T) {
	tx := &txBorradorPrueba{salida: salidaBorradorPrueba(t), errorCommit: errors.New("conexion perdida")}
	repo, _ := nuevoRepositorioBorradorComisionPostgreSQL(&poolBorradorPrueba{tx: tx})
	got, err := repo.CrearORecuperar(context.Background(), identidadBorradorPrueba(t), dietasports.SolicitudCrearBorradorPropio{ClaveIdempotencia: "clave_idempotente_0001", FechaInicio: "2026-09-21", FechaFin: "2026-09-21", Motivo: "Visita técnica"})
	if !errors.Is(err, dietasports.ErrResultadoBorradorIncierto) || got.Recibo.Referencia != "" || tx.commits != 1 {
		t.Fatalf("commit incierto expuso resultado: %#v %v", got, err)
	}
}

func TestCrearBorradorTraducePD002AConflictoIdempotencia(t *testing.T) {
	tx := &txBorradorPrueba{errSQL: &pgconn.PgError{Code: "PD002", Message: "detalle privado"}}
	repo, err := nuevoRepositorioBorradorComisionPostgreSQL(&poolBorradorPrueba{tx: tx})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.CrearORecuperar(context.Background(), identidadBorradorPrueba(t), dietasports.SolicitudCrearBorradorPropio{ClaveIdempotencia: "clave_idempotente_0001", FechaInicio: "2026-09-21", FechaFin: "2026-09-21", Motivo: "Visita técnica"})
	if !errors.Is(err, dietasports.ErrConflictoIdempotencia) || tx.commits != 0 {
		t.Fatalf("err=%v commits=%d", err, tx.commits)
	}
}

func TestLecturasUsanFuncionNominalYNoConfirmanResultadoMalformed(t *testing.T) {
	identidad := identidadConsultaBorradorPrueba(t, "dco_"+strings.Repeat("b", 22))
	salida := salidaBorradorPrueba(t)
	tx := &txBorradorPrueba{salida: salida}
	repo, _ := nuevoRepositorioBorradorComisionPostgreSQL(&poolBorradorPrueba{tx: tx})
	referencia := "dco_" + strings.Repeat("b", 22)
	if _, err := repo.ObtenerPropio(context.Background(), identidad, referencia); err != nil || len(tx.consultas) != 1 || tx.consultas[0] != consultarBorradoresSQL || tx.commits != 1 {
		t.Fatalf("detalle no usó la fachada nominal: %v %#v", err, tx)
	}
	efectoDetalle, err := efectoDetalleBorrador(identidad, referencia)
	if err != nil || !bytes.Equal([]byte(tx.argumentos[0][0].(string)), efectoDetalle.Material) {
		t.Fatal("detalle no transportó canon exacto")
	}
	lista, _ := json.Marshal(map[string]any{"items": []json.RawMessage{salida}, "siguiente_cursor": ""})
	tx = &txBorradorPrueba{salida: lista}
	identidad = identidadConsultaBorradorPrueba(t, "dietas:borradores:propios")
	repo, _ = nuevoRepositorioBorradorComisionPostgreSQL(&poolBorradorPrueba{tx: tx})
	pagina, err := repo.ListarPropios(context.Background(), identidad, dietasports.ConsultaBorradoresPropios{Limite: 1})
	if err != nil || len(pagina.Items) != 1 || len(tx.consultas) != 1 || tx.consultas[0] != consultarBorradoresSQL || tx.commits != 1 {
		t.Fatalf("lista no preservó fachada o resultado: %v %#v", err, pagina)
	}
	efectoLista, err := efectoListaBorrador(identidad, dietasports.ConsultaBorradoresPropios{Limite: 1})
	if err != nil || !bytes.Equal([]byte(tx.argumentos[0][0].(string)), efectoLista.Material) {
		t.Fatal("lista no transportó canon exacto")
	}
}

func TestDetalleNoEncontradoSeTraduceSoloTrasCommit(t *testing.T) {
	tx := &txBorradorPrueba{salida: salidaBorradorNoEncontradoPrueba(t)}
	repo, err := nuevoRepositorioBorradorComisionPostgreSQL(&poolBorradorPrueba{tx: tx})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.ObtenerPropio(context.Background(), identidadConsultaBorradorPrueba(t, "dco_"+strings.Repeat("b", 22)), "dco_"+strings.Repeat("b", 22))
	if !errors.Is(err, dietasports.ErrComisionNoEncontrada) || tx.commits != 1 || tx.rollbacks == 0 {
		t.Fatalf("resultado nominal no confirmado/traducido: err=%v tx=%#v", err, tx)
	}
}

func TestResultadoAceptaOffsetCeroYNormalizaUTC(t *testing.T) {
	resultado, err := decodificarResultado(salidaBorradorConInstantePrueba(t, "2026-09-20T10:00:00.000000+00:00"))
	if err != nil || resultado.Recibo.RegistradoEn.Location() != time.UTC || resultado.Recibo.RegistradoEn.Format("2006-01-02T15:04:05.000000Z") != "2026-09-20T10:00:00.000000Z" {
		t.Fatalf("offset UTC no normalizado: %#v, %v", resultado.Recibo, err)
	}
	if _, err := decodificarResultado(salidaBorradorConInstantePrueba(t, "2026-09-20T11:00:00.000000+01:00")); err == nil {
		t.Fatal("offset no UTC aceptado")
	}
	resultado, err = decodificarResultado(salidaBorradorConInstantePrueba(t, "2026-09-20T10:00:00+00:00"))
	if err != nil || resultado.Recibo.RegistradoEn.Location() != time.UTC {
		t.Fatalf("timestamp PostgreSQL UTC sin fracción rechazado: %#v, %v", resultado.Recibo, err)
	}
}

func TestOperacionesRechazanAutorizacionCruzadaSinAbrirTransaccion(t *testing.T) {
	type llamada func(*RepositorioBorradorComisionPostgreSQL, dietasports.IdentidadEfectivaBorrador) error
	crear := dietasports.SolicitudCrearBorradorPropio{ClaveIdempotencia: "clave_idempotente_0001", FechaInicio: "2026-09-21", FechaFin: "2026-09-21", Motivo: "Visita técnica"}
	referencia := "dco_" + strings.Repeat("b", 22)
	casos := []struct {
		nombre    string
		identidad dietasports.IdentidadEfectivaBorrador
		llamar    llamada
	}{
		{
			nombre:    "crear_con_autorizacion_de_consulta",
			identidad: identidadConsultaBorradorPrueba(t, "dietas:borradores:propios"),
			llamar: func(r *RepositorioBorradorComisionPostgreSQL, i dietasports.IdentidadEfectivaBorrador) error {
				_, err := r.CrearORecuperar(context.Background(), i, crear)
				return err
			},
		},
		{
			nombre: "crear_con_finalidad_de_consulta",
			identidad: func() dietasports.IdentidadEfectivaBorrador {
				i := identidadBorradorPrueba(t)
				i.Autorizacion.Finalidad = "consultar_borrador_propio"
				return i
			}(),
			llamar: func(r *RepositorioBorradorComisionPostgreSQL, i dietasports.IdentidadEfectivaBorrador) error {
				_, err := r.CrearORecuperar(context.Background(), i, crear)
				return err
			},
		},
		{
			nombre: "crear_con_recurso_de_detalle",
			identidad: func() dietasports.IdentidadEfectivaBorrador {
				i := identidadBorradorPrueba(t)
				i.Autorizacion.RecursoRef = referencia
				return i
			}(),
			llamar: func(r *RepositorioBorradorComisionPostgreSQL, i dietasports.IdentidadEfectivaBorrador) error {
				_, err := r.CrearORecuperar(context.Background(), i, crear)
				return err
			},
		},
		{
			nombre:    "detalle_con_recurso_de_lista",
			identidad: identidadConsultaBorradorPrueba(t, "dietas:borradores:propios"),
			llamar: func(r *RepositorioBorradorComisionPostgreSQL, i dietasports.IdentidadEfectivaBorrador) error {
				_, err := r.ObtenerPropio(context.Background(), i, referencia)
				return err
			},
		},
		{
			nombre:    "lista_con_recurso_de_detalle",
			identidad: identidadConsultaBorradorPrueba(t, referencia),
			llamar: func(r *RepositorioBorradorComisionPostgreSQL, i dietasports.IdentidadEfectivaBorrador) error {
				_, err := r.ListarPropios(context.Background(), i, dietasports.ConsultaBorradoresPropios{Limite: 1})
				return err
			},
		},
	}
	for _, tt := range casos {
		t.Run(tt.nombre, func(t *testing.T) {
			tx := &txBorradorPrueba{salida: salidaBorradorPrueba(t)}
			pool := &poolBorradorPrueba{tx: tx}
			repo, err := nuevoRepositorioBorradorComisionPostgreSQL(pool)
			if err != nil {
				t.Fatal(err)
			}
			if err := tt.llamar(repo, tt.identidad); !errors.Is(err, dietasports.ErrAccesoBorradorDenegado) {
				t.Fatalf("cruce no denegado: %v", err)
			}
			if pool.inicios != 0 || len(tx.consultas) != 0 {
				t.Fatalf("el cruce abrió transacción o ejecutó SQL: inicios=%d consultas=%d", pool.inicios, len(tx.consultas))
			}
		})
	}
}

func TestDecodificarPaginaRechazaCursorNoCanonico(t *testing.T) {
	for _, cursor := range []string{"dco_" + strings.Repeat("a", 21), "dco_" + strings.Repeat("a", 22) + ".", "dco_" + strings.Repeat("a", 21) + "\n"} {
		bruto, err := json.Marshal(map[string]any{"items": []any{}, "siguiente_cursor": cursor})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := decodificarPagina(bruto); err == nil {
			t.Fatalf("cursor no canónico aceptado: %q", cursor)
		}
	}
}

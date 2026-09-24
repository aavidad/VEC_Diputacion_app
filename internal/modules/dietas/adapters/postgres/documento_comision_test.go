package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/dietas/application"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

func identidadMutacionPrueba(t *testing.T, verbo, referencia string) dietasports.IdentidadEfectivaBorrador {
	t.Helper()
	i := identidadBorradorPrueba(t)
	i.Autorizacion.Accion = "dietas.borrador.propio." + verbo
	i.Autorizacion.RecursoRef = referencia
	i.Autorizacion.Finalidad = verbo + "_borrador_propio"
	return i
}

func identidadConsultaDocumentoPrueba(t *testing.T, recurso string) dietasports.IdentidadEfectivaBorrador {
	t.Helper()
	i := identidadBorradorPrueba(t)
	i.Autorizacion.Accion = "dietas.documento.propio.consultar"
	i.Autorizacion.RecursoRef = recurso
	i.Autorizacion.Finalidad = "consultar_documento_propio_dietas"
	return i
}

func salidaMutacionPrueba(t *testing.T, estado string, version uint64) []byte {
	t.Helper()
	var x map[string]any
	if err := json.Unmarshal(salidaBorradorPrueba(t), &x); err != nil {
		t.Fatal(err)
	}
	c := x["comision"].(map[string]any)
	x["resultado"] = "concedido"
	c["estado"] = estado
	c["version"] = version
	if version >= 2 {
		c["numero_documento"] = "VEC-D-2026-000001"
		c["fecha_apertura"] = "2026-09-21T08:00:00.000000Z"
	}
	r := x["recibo"].(map[string]any)
	r["version"] = version
	b, err := json.Marshal(x)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestBorrarDocumentoConfirmaSQLNominalYMaterialExacto(t *testing.T) {
	ref := "dco_" + strings.Repeat("b", 22)
	id := identidadMutacionPrueba(t, "borrar", ref)
	s := dietasports.SolicitudMutacionComisionPropia{Referencia: ref, ClaveIdempotencia: "clave_borrado_0001", VersionEsperada: 1, RelacionRef: id.Relacion.RelacionRef}
	tx := &txBorradorPrueba{salida: salidaMutacionPrueba(t, "eliminado", 2)}
	pool := &poolBorradorPrueba{tx: tx}
	repo, _ := nuevoRepositorioBorradorComisionPostgreSQL(pool)
	got, err := repo.BorrarPropio(context.Background(), id, s)
	if err != nil || got.Comision.Estado != "eliminado" || got.Comision.NumeroDocumento != "VEC-D-2026-000001" || got.Comision.FechaApertura != "2026-09-21T08:00:00.000000Z" || got.Recibo.Version != 2 || pool.opciones.IsoLevel != pgx.Serializable || tx.commits != 1 || len(tx.consultas) != 1 || tx.consultas[0] != mutarComisionPropiaSQL {
		t.Fatalf("mutación nominal: resultado=%#v error=%v commits=%d consultas=%v", got, err, tx.commits, tx.consultas)
	}
	op, err := application.NuevaSolicitudOperacionMutarBorrador(dietasports.OperacionBorrarBorrador, s)
	if err != nil {
		t.Fatal(err)
	}
	efecto, err := application.ConstruirEfectoAutorizacionBorrador(id.ContextoRegistrado, id.Relacion, id.Autorizacion.Revalidacion, op)
	if err != nil || len(tx.argumentos[0]) != 11 || !bytes.Equal([]byte(tx.argumentos[0][0].(string)), efecto.Material) {
		t.Fatal("preimagen AD3 o argumentos distintos")
	}
}

func TestLecturaDocumentoV2ExigeAccionYProyeccionNuevas(t *testing.T) {
	ref := "dco_" + strings.Repeat("b", 22)
	tx := &txBorradorPrueba{salida: salidaMutacionPrueba(t, "borrador", 1)}
	pool := &poolBorradorPrueba{tx: tx}
	repo, _ := nuevoRepositorioBorradorComisionPostgreSQL(pool)
	if _, err := repo.ObtenerDocumentoPropio(context.Background(), identidadConsultaBorradorPrueba(t, ref), ref); !errors.Is(err, dietasports.ErrAccesoBorradorDenegado) || pool.inicios != 0 {
		t.Fatalf("lectura v1 cruzó a documento v2: %v %d", err, pool.inicios)
	}
	got, err := repo.ObtenerDocumentoPropio(context.Background(), identidadConsultaDocumentoPrueba(t, ref), ref)
	if err != nil || got.Recibo.Version != 1 || len(tx.consultas) != 1 || tx.consultas[0] != consultarComisionesPropiasSQL || tx.commits != 1 {
		t.Fatalf("lectura v2: %#v %v", got, err)
	}
}

func TestPaginaDocumentoRechazaAusenciaDeArrayOcursor(t *testing.T) {
	for _, bruto := range []string{`{}`, `{"items":null,"siguiente_cursor":""}`, `{"items":[],"siguiente_cursor":null}`} {
		if _, err := decodificarPaginaDocumento([]byte(bruto)); err == nil {
			t.Fatalf("página SQL incompleta aceptada: %s", bruto)
		}
	}
}

func TestDecodificadorDocumentoConservaVehiculoNoYArrayRutasVacio(t *testing.T) {
	var x map[string]any
	if err := json.Unmarshal(salidaMutacionPrueba(t, "borrador", 2), &x); err != nil {
		t.Fatal(err)
	}
	c := x["comision"].(map[string]any)
	c["vehiculo_propio"] = false
	c["rutas"] = []any{}
	bruto, err := json.Marshal(x)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decodificarResultadoDocumento(bruto)
	if err != nil || got.Comision.VehiculoPropio == nil || *got.Comision.VehiculoPropio || got.Comision.Rutas == nil || len(*got.Comision.Rutas) != 0 {
		t.Fatalf("declaración D4 perdida: %#v %v", got.Comision, err)
	}
}

func TestBorrarDocumentoDeniegaCruceYNoExponeReciboConCommitIncierto(t *testing.T) {
	ref := "dco_" + strings.Repeat("b", 22)
	s := dietasports.SolicitudMutacionComisionPropia{Referencia: ref, ClaveIdempotencia: "clave_borrado_0001", VersionEsperada: 1}
	tx := &txBorradorPrueba{salida: salidaMutacionPrueba(t, "eliminado", 2)}
	pool := &poolBorradorPrueba{tx: tx}
	repo, _ := nuevoRepositorioBorradorComisionPostgreSQL(pool)
	if _, err := repo.BorrarPropio(context.Background(), identidadMutacionPrueba(t, "enviar", ref), s); !errors.Is(err, dietasports.ErrAccesoBorradorDenegado) || pool.inicios != 0 {
		t.Fatalf("cruce de acción abrió transacción: %v %d", err, pool.inicios)
	}
	tx.errorCommit = errors.New("conexión perdida")
	got, err := repo.BorrarPropio(context.Background(), identidadMutacionPrueba(t, "borrar", ref), s)
	if !errors.Is(err, dietasports.ErrResultadoBorradorIncierto) || got.Recibo.Referencia != "" || tx.commits != 1 {
		t.Fatalf("commit incierto expuso recibo: %#v %v", got, err)
	}
}

func TestMutacionTraduceConflictoVersionSinCommit(t *testing.T) {
	ref := "dco_" + strings.Repeat("b", 22)
	tx := &txBorradorPrueba{errSQL: &pgconn.PgError{Code: "PD005", Message: "dato privado"}}
	repo, _ := nuevoRepositorioBorradorComisionPostgreSQL(&poolBorradorPrueba{tx: tx})
	_, err := repo.EnviarPropio(context.Background(), identidadMutacionPrueba(t, "enviar", ref), dietasports.SolicitudMutacionComisionPropia{Referencia: ref, ClaveIdempotencia: "clave_envio_00001", VersionEsperada: 2})
	if !errors.Is(err, dietasports.ErrVersionComisionConflicto) || tx.commits != 0 {
		t.Fatalf("conflicto OCC: error=%v commits=%d", err, tx.commits)
	}
}

func TestRecuperarEdicionAntesDeRecalcularNoCreaYExigeConcesionNueva(t *testing.T) {
	ref := "dco_" + strings.Repeat("b", 22)
	id := identidadMutacionPrueba(t, "editar", ref)
	s := dietasports.SolicitudEditarComisionPropia{Referencia: ref, ClaveIdempotencia: "clave_edicion_0001", VersionEsperada: 1, RelacionRef: id.Relacion.RelacionRef, FechaInicio: "2026-09-21", FechaFin: "2026-09-21", HoraInicio: "09:00", HoraFin: "18:00", Motivo: "Visita técnica", CodigosRuta: []string{"GR:001", "GR:002"}, VersionTarifaAceptada: "provisional:rd462:20260923", TramosAceptados: []int{0}}
	ausente := &txBorradorPrueba{salida: []byte(`{"encontrado":false}`)}
	repo, _ := nuevoRepositorioBorradorComisionPostgreSQL(&poolBorradorPrueba{tx: ausente})
	got, encontrado, err := repo.RecuperarEdicionPorClave(context.Background(), id, s)
	if err != nil || encontrado || got.Recibo.Referencia != "" || ausente.commits != 1 || len(ausente.consultas) != 1 || ausente.consultas[0] != recuperarMutacionPorClaveSQL {
		t.Fatalf("preconsulta ausente: encontrado=%v err=%v consultas=%v", encontrado, err, ausente.consultas)
	}
	var presente map[string]any
	if err := json.Unmarshal(salidaMutacionPrueba(t, "borrador", 2), &presente); err != nil {
		t.Fatal(err)
	}
	presente["encontrado"] = true
	presente["recibo"].(map[string]any)["repeticion"] = true
	bruto, _ := json.Marshal(presente)
	tx := &txBorradorPrueba{salida: bruto}
	repo, _ = nuevoRepositorioBorradorComisionPostgreSQL(&poolBorradorPrueba{tx: tx})
	got, encontrado, err = repo.RecuperarEdicionPorClave(context.Background(), id, s)
	if err != nil || !encontrado || !got.Recibo.Repeticion || got.Recibo.Version != 2 || tx.commits != 1 {
		t.Fatalf("preconsulta presente: %#v encontrado=%v err=%v", got, encontrado, err)
	}
}

package auditoriaconsulta

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/vec/auditoria"
)

func TestRegistroCTRespetaFiltroYCursor(t *testing.T) {
	instante := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	f := auditoria.Filtro{Fuente: "ct", ExpedienteRef: "expediente:ct:1", Desde: instante.Add(-time.Hour),
		Hasta: instante.Add(time.Hour), Limite: 2}
	r := auditoria.Registro{ID: "ct:v:0000000000000002", Fuente: "ct", ModuloID: "contratacion_temporal",
		Accion: "cese_nombramiento_ct115", ActorRef: "actor:rrhh:1", Resultado: "en_curso",
		ExpedienteRef: f.ExpedienteRef, ReciboRef: "recibo:ct:2", OcurridoEn: instante,
		AntesSHA256: strings.Repeat("a", 64), DespuesSHA256: strings.Repeat("b", 64),
		Antes:   map[string]string{"fase": "nombramiento", "estado": "en_curso"},
		Despues: map[string]string{"fase": "nombramiento", "estado": "en_curso"}, DatosDisponibles: true}
	if !registroValido(r, f, nil) {
		t.Fatal("registro CT válido rechazado")
	}
	f.ActorRef = "actor:rrhh:2"
	if registroValido(r, f, nil) {
		t.Fatal("filtro de actor no aplicado")
	}
	f.ActorRef = ""
	f.Antes = auditoria.Posicion{OcurridoEn: instante, Fuente: "ct", ID: r.ID}
	if registroValido(r, f, nil) {
		t.Fatal("cursor repetido")
	}
	f.Antes = auditoria.Posicion{}
	masAntiguo := r
	masAntiguo.ID = "ct:v:0000000000000001"
	if !registroValido(masAntiguo, f, []auditoria.Registro{r}) {
		t.Fatal("orden descendente válido")
	}
	if registroValido(r, f, []auditoria.Registro{r}) {
		t.Fatal("duplicado de posición")
	}
}

func TestRegistroCTRechazaSalidaAjenayNoFiltraErrorSQL(t *testing.T) {
	instante := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	f := auditoria.Filtro{Fuente: "ct", ExpedienteRef: "expediente:ct:1", Desde: instante.Add(-time.Hour),
		Hasta: instante.Add(time.Hour), Limite: 2}
	r := auditoria.Registro{ID: "ct:v:0000000000000001", Fuente: "bolsa", ModuloID: "contratacion_temporal",
		Accion: "alta_o2", Resultado: "en_curso", ExpedienteRef: f.ExpedienteRef,
		OcurridoEn: instante, DespuesSHA256: strings.Repeat("b", 64),
		Despues: map[string]string{"fase": "inicio", "estado": "en_curso"}, DatosDisponibles: true}
	if registroValido(r, f, nil) {
		t.Fatal("fuente ajena aceptada")
	}
	r.Fuente = "ct"
	r.ExpedienteRef = "expediente:ajeno"
	if registroValido(r, f, nil) {
		t.Fatal("expediente ajeno aceptado")
	}
	r.ExpedienteRef = f.ExpedienteRef
	r.Despues["nombre"] = "persona"
	if registroValido(r, f, nil) {
		t.Fatal("campo personal inesperado aceptado")
	}
	if !errors.Is(normalizar(context.Background(), &pgconn.PgError{Code: "42501", Message: "privado"}), auditoria.ErrDenegada) {
		t.Fatal("denegación SQL no normalizada")
	}
	if !errors.Is(normalizar(context.Background(), errors.New("detalle privado")), auditoria.ErrNoDisponible) {
		t.Fatal("error técnico SQL revelado")
	}
}

type transaccionCierreCT struct {
	pgx.Tx
	err   error
	veces int
}

func (t *transaccionCierreCT) Rollback(ctx context.Context) error {
	t.veces++
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if _, ok := ctx.Deadline(); !ok {
		return errors.New("rollback sin plazo")
	}
	return t.err
}

func TestConsultaCTCierraAntesDeDevolverElResultado(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	pagina := auditoria.PaginaFuente{Registros: []auditoria.Registro{{ID: "fila"}}}
	fallo := auditoria.ErrDenegada
	tx := &transaccionCierreCT{}
	cerrarTransaccionConsulta(ctx, tx, &pagina, &fallo)
	if tx.veces != 1 || !errors.Is(fallo, auditoria.ErrDenegada) || len(pagina.Registros) != 1 {
		t.Fatalf("rollback válido alteró el resultado: veces=%d fallo=%v pagina=%+v", tx.veces, fallo, pagina)
	}
	tx.err = pgx.ErrTxClosed // COMMIT ya confirmó la lectura.
	fallo = nil
	cerrarTransaccionConsulta(ctx, tx, &pagina, &fallo)
	if fallo != nil || len(pagina.Registros) != 1 {
		t.Fatalf("transacción confirmada alterada: fallo=%v pagina=%+v", fallo, pagina)
	}
	tx.err = errors.New("rollback interrumpido")
	cerrarTransaccionConsulta(ctx, tx, &pagina, &fallo)
	if !errors.Is(fallo, auditoria.ErrNoDisponible) || len(pagina.Registros) != 0 {
		t.Fatalf("cierre fallido entregó la lectura: fallo=%v pagina=%+v", fallo, pagina)
	}
}

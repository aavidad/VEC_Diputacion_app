package auditoriaconsulta

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/vec/auditoria"
)

func TestFuenteDeniegaSinMaterialNiPool(t *testing.T) {
	if fuente, err := NuevaFuente(nil); fuente != nil || !errors.Is(err, auditoria.ErrNoDisponible) {
		t.Fatalf("pool ausente: fuente=%v error=%v", fuente, err)
	}
	var fuente Fuente
	if pagina, err := fuente.ConsultarAuditoria(context.Background(), auditoria.ConsultaAutorizada{}); len(pagina.Registros) != 0 || !errors.Is(err, auditoria.ErrNoDisponible) {
		t.Fatalf("consulta sin fuente disponible: pagina=%+v error=%v", pagina, err)
	}
}

type transaccionCierreBolsa struct {
	pgx.Tx
	err   error
	veces int
}

func (t *transaccionCierreBolsa) Rollback(ctx context.Context) error {
	t.veces++
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if _, ok := ctx.Deadline(); !ok {
		return errors.New("rollback sin plazo")
	}
	return t.err
}

func TestConsultaBolsaCierraAntesDeDevolverElResultado(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	pagina := auditoria.PaginaFuente{Registros: []auditoria.Registro{{ID: "fila"}}}
	fallo := auditoria.ErrDenegada
	tx := &transaccionCierreBolsa{}
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

func TestProyeccionHistorialMinimizaCampos(t *testing.T) {
	desde := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
	filtro := auditoria.Filtro{Fuente: "bolsa", ExpedienteRef: "participacion:prueba", Desde: desde, Hasta: desde.Add(time.Hour)}
	anterior, nuevo, campo := "no_disponible", "disponible", "situacion"
	fila := filaSQL{ID: "cambio:recibo:prueba:situacion", OcurridoEn: desde.Add(time.Minute),
		Accion: "valor:situacion", ActorRef: "per_sintetica", Resultado: "confirmado",
		ExpedienteRef: filtro.ExpedienteRef, ReciboRef: "recibo:prueba",
		Campo: &campo, Anterior: &anterior, Nuevo: &nuevo}
	r, ok := proyectar(fila, filtro)
	if !ok || !r.DatosDisponibles || r.Antes[campo] != anterior || r.Despues[campo] != nuevo ||
		r.Fuente != "bolsa" || r.ModuloID != "bolsa" || r.ExpedienteRef != filtro.ExpedienteRef {
		t.Fatalf("traza minimizada invalida: %+v, ok=%v", r, ok)
	}
	fila.Campo, fila.Anterior, fila.Nuevo = nil, nil, nil
	fila.ID, fila.Accion = "situacion:recibo:prueba", "reactivar"
	r, ok = proyectar(fila, filtro)
	if !ok || r.DatosDisponibles || len(r.Antes) != 0 || len(r.Despues) != 0 {
		t.Fatalf("operacion sin preimagen inventada: %+v, ok=%v", r, ok)
	}
}

func TestProyeccionRechazaFilaDeOtraParticipacion(t *testing.T) {
	desde := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
	filtro := auditoria.Filtro{Fuente: "bolsa", ExpedienteRef: "participacion:permitida", Desde: desde, Hasta: desde.Add(time.Hour)}
	fila := filaSQL{ID: "situacion:ajena", OcurridoEn: desde, Accion: "pausar",
		ActorRef: "per_sintetica", Resultado: "confirmado", ExpedienteRef: "participacion:ajena", ReciboRef: "recibo:ajeno"}
	if r, ok := proyectar(fila, filtro); ok || r.ID != "" {
		t.Fatalf("fila ajena expuesta: %+v", r)
	}
}

func TestProyeccionRechazaContactoEnClaro(t *testing.T) {
	desde := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
	filtro := auditoria.Filtro{Fuente: "bolsa", ExpedienteRef: "participacion:prueba", Desde: desde, Hasta: desde.Add(time.Hour)}
	campo, claro := "correo", "persona@example.invalid"
	fila := filaSQL{ID: "cambio:recibo:prueba:correo", OcurridoEn: desde.Add(time.Minute),
		Accion: "valor:correo", ActorRef: "per_sintetica", Resultado: "confirmado",
		ExpedienteRef: filtro.ExpedienteRef, ReciboRef: "recibo:prueba", Campo: &campo, Nuevo: &claro}
	if r, ok := proyectar(fila, filtro); ok || r.ID != "" {
		t.Fatalf("contacto en claro expuesto: %+v", r)
	}
	version := "version:2"
	fila.Nuevo = &version
	if r, ok := proyectar(fila, filtro); !ok || r.Despues[campo] != version {
		t.Fatalf("referencia cifrada rechazada: %+v, ok=%v", r, ok)
	}
}

func TestProyeccionRechazaMotivoLibre(t *testing.T) {
	desde := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
	filtro := auditoria.Filtro{Fuente: "bolsa", ExpedienteRef: "participacion:prueba", Desde: desde, Hasta: desde.Add(time.Hour)}
	libre := "Motivo con datos de una persona"
	fila := filaSQL{ID: "situacion:recibo:prueba", OcurridoEn: desde.Add(time.Minute),
		Accion: "pausar", ActorRef: "per_sintetica", Resultado: "confirmado",
		ExpedienteRef: filtro.ExpedienteRef, ReciboRef: "recibo:prueba", Motivo: &libre}
	if r, ok := proyectar(fila, filtro); ok || r.ID != "" {
		t.Fatalf("motivo libre expuesto: %+v", r)
	}
	reservado := "Motivo reservado en Bolsa"
	fila.Motivo = &reservado
	if r, ok := proyectar(fila, filtro); !ok || r.Motivo != reservado {
		t.Fatalf("motivo minimizado rechazado: %+v, ok=%v", r, ok)
	}
}

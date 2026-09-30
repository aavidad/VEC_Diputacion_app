package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/vec/ports"
)

func TestPreparacionUsoRPTObtieneHuellaPGSinAutorizarNiEscribir(t *testing.T) {
	m := materialReservaUsoRPTPrueba()
	terminal := ports.MaterialTerminalUsoCategoriaRPT{Reserva: m, TerminalReciboRef: "recibo:terminal:001",
		EvidenciaRef: "evidencia:ct:001", EvidenciaSHA256: strings.Repeat("e", 64)}
	for _, caso := range []struct {
		accion string
		claves int
		correr func(*GestorUsosCategoriaRPTPostgreSQL) (ports.PreparacionAutorizacionUsoCategoriaRPT, error)
	}{
		{accionReservarUsoRPT, 8, func(g *GestorUsosCategoriaRPTPostgreSQL) (ports.PreparacionAutorizacionUsoCategoriaRPT, error) {
			return g.PrepararReservaUsoCategoriaRPT(t.Context(), m)
		}},
		{accionConfirmarUsoRPT, 11, func(g *GestorUsosCategoriaRPTPostgreSQL) (ports.PreparacionAutorizacionUsoCategoriaRPT, error) {
			return g.PrepararConfirmacionUsoCategoriaRPT(t.Context(), terminal)
		}},
		{accionCancelarUsoRPT, 11, func(g *GestorUsosCategoriaRPTPostgreSQL) (ports.PreparacionAutorizacionUsoCategoriaRPT, error) {
			return g.PrepararCancelacionUsoCategoriaRPT(t.Context(), terminal)
		}},
	} {
		t.Run(caso.accion, func(t *testing.T) {
			// La huella devuelta por este doble sólo prueba el transporte de
			// la autoridad PG. No demuestra el canon real de PostgreSQL.
			tx := &transaccionLecturaRPTPrueba{huellaMaterial: strings.Repeat("b", 64)}
			inicio := &iniciadorUsoRPTPrueba{tx: tx}
			g, err := nuevoGestorUsosCategoriaRPTPostgreSQL(inicio, descriptorRPTPrueba, m.Consumidor)
			if err != nil {
				t.Fatal(err)
			}
			p, err := caso.correr(g)
			if err != nil || p.Accion != caso.accion || p.Finalidad != finalidadUsosRPT ||
				p.AudienciaConsumo != audienciaUsosRPT || p.Recurso.Referencia != m.UsoRef ||
				p.Recurso.ModuloID != descriptorRPTPrueba.ModuloID || p.Recurso.Tipo != tipoUsoRPT ||
				!reflect.DeepEqual(p.Recurso.Ambitos, map[string]string{"catalogo_id": m.Publicacion.CatalogoID,
					"modulo_id": descriptorRPTPrueba.ModuloID, "consumidor": m.Consumidor}) ||
				!reflect.DeepEqual(p.Recurso.Atributos, map[string]string{"material_sha256": tx.huellaMaterial}) ||
				tx.fachadas != 0 || tx.consultasCanon != 1 || !tx.configurada || !tx.confirmada ||
				inicio.opciones.AccessMode != pgx.ReadOnly {
				t.Fatalf("preparación no exacta o con efecto: %+v %v tx=%+v", p, err, tx)
			}
			var material map[string]any
			if json.Unmarshal([]byte(tx.materialCanonico), &material) != nil || len(material) != caso.claves {
				t.Fatalf("preparación omitió material: %s", tx.materialCanonico)
			}
		})
	}
}

func TestUsoRPTRechazaHuellaJSONGoYRecotejaPreparacion(t *testing.T) {
	m := materialReservaUsoRPTPrueba()
	tx := &transaccionLecturaRPTPrueba{huellaMaterial: strings.Repeat("b", 64)}
	g, _ := nuevoGestorUsosCategoriaRPTPostgreSQL(&iniciadorUsoRPTPrueba{tx: tx}, descriptorRPTPrueba, m.Consumidor)
	p, err := g.PrepararReservaUsoCategoriaRPT(t.Context(), m)
	if err != nil {
		t.Fatal(err)
	}
	sumaGo := sha256.Sum256([]byte(tx.materialCanonico))
	huellaGo := hex.EncodeToString(sumaGo[:])
	if huellaGo == p.Recurso.Atributos["material_sha256"] {
		t.Fatal("fixture no distingue JSON Go y huella PostgreSQL")
	}
	// Una solicitud V3 ligada a la preparación acepta el mismo material.
	s, a := autorizacionUsoRPTHuellaPrueba(t, p.Accion, p.AudienciaConsumo, m.UsoRef, p.Recurso.Atributos["material_sha256"])
	tx.respuesta = reciboUsoRPTPrueba(t, a, m, "reservado", m.ReservaReciboRef, "")
	if r, err := g.ReservarUsoCategoriaRPT(t.Context(), ports.OrdenReservaUsoCategoriaRPT{Material: m, Solicitud: s, Autorizacion: a}); err != nil || !r.Encontrado || !tx.confirmada || tx.materialCanonico != tx.argumentos[0] {
		t.Fatalf("material preparado no llegó intacto al efecto: %+v %v tx=%+v", r, err, tx)
	}
	tx.fachadas = 0
	s, a = autorizacionUsoRPTHuellaPrueba(t, p.Accion, p.AudienciaConsumo, m.UsoRef, huellaGo)
	tx.confirmada, tx.revertida = false, false
	if r, err := g.ReservarUsoCategoriaRPT(t.Context(), ports.OrdenReservaUsoCategoriaRPT{Material: m, Solicitud: s, Autorizacion: a}); !errors.Is(err, ports.ErrUsoCategoriaRPTDenegado) || r.Encontrado || tx.fachadas != 0 || tx.confirmada || !tx.revertida {
		t.Fatalf("huella JSON Go concedió efecto: %+v %v tx=%+v", r, err, tx)
	}
	// La preparación no es un permiso reusable: se vuelve a consultar PG.
	s, a = autorizacionUsoRPTHuellaPrueba(t, p.Accion, p.AudienciaConsumo, m.UsoRef, p.Recurso.Atributos["material_sha256"])
	tx.huellaMaterial = strings.Repeat("c", 64)
	if _, err := g.ReservarUsoCategoriaRPT(t.Context(), ports.OrdenReservaUsoCategoriaRPT{Material: m, Solicitud: s, Autorizacion: a}); !errors.Is(err, ports.ErrUsoCategoriaRPTDenegado) || tx.fachadas != 0 || tx.consultasCanon != 4 {
		t.Fatalf("huella preparada no se revalidó: %v tx=%+v", err, tx)
	}
}

func TestPreparacionUsoRPTFallaSinEntregarRecursoNiEfecto(t *testing.T) {
	m := materialReservaUsoRPTPrueba()
	for _, etapa := range []string{"configurar", "huella", "commit", "huella inválida"} {
		t.Run(etapa, func(t *testing.T) {
			tx := &transaccionFalloUsoRPT{transaccionLecturaRPTPrueba: &transaccionLecturaRPTPrueba{},
				etapa: etapa, fallo: errors.New("fallo sintético")}
			esperado := ports.ErrUsoCategoriaRPTNoDisponible
			if etapa == "huella inválida" {
				tx.huellaMaterial = "huella:no-confiable"
				esperado = ports.ErrUsoCategoriaRPTNoConfiable
			}
			g, _ := nuevoGestorUsosCategoriaRPTPostgreSQL(&iniciadorUsoRPTPrueba{tx: tx}, descriptorRPTPrueba, m.Consumidor)
			r, err := g.PrepararReservaUsoCategoriaRPT(t.Context(), m)
			if !errors.Is(err, esperado) || !reflect.DeepEqual(r, ports.PreparacionAutorizacionUsoCategoriaRPT{}) ||
				tx.fachadas != 0 || tx.confirmada || !tx.revertida {
				t.Fatalf("preparación fallida expuso recurso o efecto: %+v %v tx=%+v", r, err, tx)
			}
		})
	}
	inicio := &iniciadorUsoRPTPrueba{tx: &transaccionLecturaRPTPrueba{}}
	g, _ := nuevoGestorUsosCategoriaRPTPostgreSQL(inicio, descriptorRPTPrueba, m.Consumidor)
	if _, err := g.PrepararReservaUsoCategoriaRPT(nil, m); !errors.Is(err, ports.ErrUsoCategoriaRPTInvalido) || inicio.llamadas != 0 {
		t.Fatalf("preparación sin contexto llegó a SQL: %v", err)
	}
	m.Consumidor = "personal"
	if _, err := g.PrepararReservaUsoCategoriaRPT(t.Context(), m); !errors.Is(err, ports.ErrUsoCategoriaRPTInvalido) || inicio.llamadas != 0 {
		t.Fatalf("consumidor ajeno eligió preparación SQL: %v", err)
	}
}

type filaFalloUsoRPT struct{ err error }

func (f filaFalloUsoRPT) Scan(...any) error { return f.err }

type transaccionFalloUsoRPT struct {
	*transaccionLecturaRPTPrueba
	etapa       string
	fallo       error
	pendiente   bool
	durable     bool
	trasFachada func()
}

func (t *transaccionFalloUsoRPT) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if t.etapa == "configurar" {
		return pgconn.CommandTag{}, t.fallo
	}
	return t.transaccionLecturaRPTPrueba.Exec(ctx, sql, args...)
}

func (t *transaccionFalloUsoRPT) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if sql == consultaHuellaMaterialRPT {
		if t.etapa == "huella" {
			return filaFalloUsoRPT{t.fallo}
		}
	} else {
		if t.etapa == "fachada" {
			return filaFalloUsoRPT{t.fallo}
		}
		t.pendiente = true
		if t.trasFachada != nil {
			t.trasFachada()
		}
	}
	return t.transaccionLecturaRPTPrueba.QueryRow(ctx, sql, args...)
}

func (t *transaccionFalloUsoRPT) Commit(ctx context.Context) error {
	if t.etapa == "commit" {
		return t.fallo
	}
	t.durable = t.pendiente
	return t.transaccionLecturaRPTPrueba.Commit(ctx)
}

func (t *transaccionFalloUsoRPT) Rollback(ctx context.Context) error {
	t.pendiente = false
	return t.transaccionLecturaRPTPrueba.Rollback(ctx)
}

func TestUsoRPTFalloRevierteEfectoSimuladoSinEntregarRecibo(t *testing.T) {
	m := materialReservaUsoRPTPrueba()
	terminal := ports.MaterialTerminalUsoCategoriaRPT{Reserva: m, TerminalReciboRef: "recibo:terminal:001",
		EvidenciaRef: "evidencia:ct:001", EvidenciaSHA256: strings.Repeat("e", 64)}
	for _, accion := range []string{accionReservarUsoRPT, accionConfirmarUsoRPT, accionCancelarUsoRPT} {
		for _, etapa := range []string{"configurar", "huella", "fachada", "recibo", "cancelar", "commit"} {
			t.Run(accion+"/"+etapa, func(t *testing.T) {
				s, a := autorizacionUsoRPTPrueba(t, accion, audienciaUsosRPT, m.UsoRef)
				estado, reciboRef, terminalRef := "reservado", m.ReservaReciboRef, ""
				if accion != accionReservarUsoRPT {
					estado, reciboRef, terminalRef = "confirmado", terminal.TerminalReciboRef, terminal.TerminalReciboRef
					if accion == accionCancelarUsoRPT {
						estado = "cancelado"
					}
				}
				tx := &transaccionFalloUsoRPT{transaccionLecturaRPTPrueba: &transaccionLecturaRPTPrueba{
					respuesta: reciboUsoRPTPrueba(t, a, m, estado, reciboRef, terminalRef)},
					etapa: etapa, fallo: errors.New("fallo sintético")}
				esperado := ports.ErrUsoCategoriaRPTNoDisponible
				ctx, cancelar := context.WithCancel(t.Context())
				defer cancelar()
				if etapa == "recibo" {
					tx.respuesta = []byte(`{"recibo_ref":"recibo:ajeno"}`)
					esperado = ports.ErrUsoCategoriaRPTNoConfiable
				}
				if etapa == "cancelar" {
					tx.trasFachada = cancelar
					esperado = context.Canceled
				}
				g, _ := nuevoGestorUsosCategoriaRPTPostgreSQL(&iniciadorUsoRPTPrueba{tx: tx}, descriptorRPTPrueba, m.Consumidor)
				var r ports.ResultadoUsoCategoriaRPT
				var err error
				switch accion {
				case accionReservarUsoRPT:
					r, err = g.ReservarUsoCategoriaRPT(ctx, ports.OrdenReservaUsoCategoriaRPT{Material: m, Solicitud: s, Autorizacion: a})
				case accionConfirmarUsoRPT:
					r, err = g.ConfirmarUsoCategoriaRPT(ctx, ports.OrdenConfirmacionUsoCategoriaRPT{Material: terminal, Solicitud: s, Autorizacion: a})
				case accionCancelarUsoRPT:
					r, err = g.CancelarUsoCategoriaRPT(ctx, ports.OrdenCancelacionUsoCategoriaRPT{Material: terminal, Solicitud: s, Autorizacion: a})
				}
				if !errors.Is(err, esperado) || !reflect.DeepEqual(r, ports.ResultadoUsoCategoriaRPT{}) ||
					tx.durable || tx.pendiente || tx.confirmada || !tx.revertida {
					t.Fatalf("fallo expuso recibo o dejó efecto simulado: %+v %v tx=%+v", r, err, tx)
				}
			})
		}
	}
}

func TestUsoRPTTerminalContradictorioNoConfirma(t *testing.T) {
	m := ports.MaterialTerminalUsoCategoriaRPT{Reserva: materialReservaUsoRPTPrueba(),
		TerminalReciboRef: "recibo:terminal:001", EvidenciaRef: "evidencia:ct:001", EvidenciaSHA256: strings.Repeat("e", 64)}
	for _, confirmar := range []bool{true, false} {
		accion, estado := accionCancelarUsoRPT, "confirmado"
		if confirmar {
			accion, estado = accionConfirmarUsoRPT, "cancelado"
		}
		s, a := autorizacionUsoRPTPrueba(t, accion, audienciaUsosRPT, m.Reserva.UsoRef)
		tx := &transaccionFalloUsoRPT{transaccionLecturaRPTPrueba: &transaccionLecturaRPTPrueba{
			respuesta: reciboUsoRPTPrueba(t, a, m.Reserva, estado, m.TerminalReciboRef, m.TerminalReciboRef)}}
		g, _ := nuevoGestorUsosCategoriaRPTPostgreSQL(&iniciadorUsoRPTPrueba{tx: tx}, descriptorRPTPrueba, m.Reserva.Consumidor)
		var err error
		if confirmar {
			_, err = g.ConfirmarUsoCategoriaRPT(t.Context(), ports.OrdenConfirmacionUsoCategoriaRPT{Material: m, Solicitud: s, Autorizacion: a})
		} else {
			_, err = g.CancelarUsoCategoriaRPT(t.Context(), ports.OrdenCancelacionUsoCategoriaRPT{Material: m, Solicitud: s, Autorizacion: a})
		}
		if !errors.Is(err, ports.ErrUsoCategoriaRPTNoConfiable) || tx.durable || tx.pendiente || !tx.revertida {
			t.Fatalf("terminal contradictorio aceptado para %s: %v tx=%+v", accion, err, tx)
		}
	}
}

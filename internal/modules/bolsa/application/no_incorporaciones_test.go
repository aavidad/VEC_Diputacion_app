package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type buzonNoIncorporacionesPrueba struct {
	recibidos   []ports.EventoNoIncorporacionRecibido
	pendientes  []ports.PendienteNoIncorporacion
	reevaluados map[string]*ports.PlazosNoIncorporacion
}

func (b *buzonNoIncorporacionesPrueba) PendientesNoIncorporacion(_ context.Context, limite int) ([]ports.PendienteNoIncorporacion, error) {
	if len(b.pendientes) > limite {
		return b.pendientes[:limite], nil
	}
	return b.pendientes, nil
}

func (b *buzonNoIncorporacionesPrueba) ReevaluarNoIncorporacion(_ context.Context, ref string, plazos *ports.PlazosNoIncorporacion) (ports.ResultadoRegistroNoIncorporacion, error) {
	if b.reevaluados == nil {
		b.reevaluados = map[string]*ports.PlazosNoIncorporacion{}
	}
	b.reevaluados[ref] = plazos
	if plazos == nil {
		return ports.ResultadoRegistroNoIncorporacion{Reutilizado: true, Estado: "consecuencia_no_admitida"}, nil
	}
	return ports.ResultadoRegistroNoIncorporacion{Reutilizado: true, Estado: ports.EstadoNoIncorporacionAplicada}, nil
}

func (b *buzonNoIncorporacionesPrueba) CursorNoIncorporaciones(context.Context) (ports.CursorContratosParticipacion, bool, error) {
	return ports.CursorContratosParticipacion{}, false, nil
}

func (b *buzonNoIncorporacionesPrueba) RegistrarNoIncorporacion(_ context.Context, e ports.EventoNoIncorporacionRecibido) (ports.ResultadoRegistroNoIncorporacion, error) {
	b.recibidos = append(b.recibidos, e)
	return ports.ResultadoRegistroNoIncorporacion{Estado: "aplicada", ParticipacionRef: "participacion:1"}, nil
}

type catalogoNoIncorporacionPrueba struct {
	err   error
	clave string
	fecha time.Time
}

func (c *catalogoNoIncorporacionPrueba) ResolverSancion(_ context.Context, clave string, notificada time.Time) (ports.ResolucionCatalogoSancion, error) {
	c.clave, c.fecha = clave, notificada
	return ports.ResolucionCatalogoSancion{Consecuencia: ports.ConsecuenciaSancion{Clave: clave, Etiqueta: "Baja", Efecto: "excluir",
		ReglaRef: "vec.bolsa.reglas:3:" + clave, Huella: strings.Repeat("c", 64)},
		Recurso: ports.PlazoSancion{UltimoDia: "2026-10-20", ReglaRef: "vec.bolsa.reglas:3:b24.consecuencias", Huella: strings.Repeat("d", 64)}}, c.err
}

func eventoNoIncorporacionPrueba() (string, string) {
	origen := "evento:ct124:ni1"
	ref := sha256.Sum256([]byte("no_incorporacion\x1f" + origen))
	c := `{"tipo": "no_incorporacion", "esquema": "vec.contratacion-temporal.no-incorporacion-bolsa.v1", "actor_ref": "per_actor", "evento_ref": "evento:ct:no-incorporacion-bolsa:` +
		hex.EncodeToString(ref[:]) + `", "origen_ref": "` + origen + `", "ocurrido_en": "2026-09-21T09:00:00.000000Z", "motivo_clave": "no_presentado", "resuelta_por": "per_segunda", "expediente_ref": "expediente:ct:1", "resolucion_ref": "resolucion:rrhh:2026/0142", "llamamiento_ref": "llamamiento:1", "organizacion_ref": "organizacion:1", "resolucion_sha256": "` +
		strings.Repeat("b", 64) + `", "consecuencia_clave": "b24.sancion.baja_llamamiento_directo", "fecha_notificacion": "2026-09-20"}`
	h := sha256.Sum256([]byte(c))
	return c, hex.EncodeToString(h[:])
}

func TestRecepcionNoIncorporacionesCalculaLosPlazosConElCatalogo(t *testing.T) {
	buzon, catalogo := &buzonNoIncorporacionesPrueba{}, &catalogoNoIncorporacionPrueba{}
	s, err := NuevoServicioRecepcionNoIncorporaciones(buzon, catalogo)
	if err != nil {
		t.Fatal(err)
	}
	c, h := eventoNoIncorporacionPrueba()
	res, err := s.Recibir(context.Background(), []byte(c), h, time.Date(2026, 9, 21, 9, 0, 0, 0, time.UTC), 7)
	if err != nil || res.Estado != "aplicada" || len(buzon.recibidos) != 1 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	// La consecuencia no viaja: la resuelve la base con su política; el
	// relevo solo aporta las fechas del calendario y la regla del recurso.
	plazos := buzon.recibidos[0].Plazos
	if plazos == nil || plazos.RecursoVence != "2026-10-20" || plazos.RecursoReglaRef != "vec.bolsa.reglas:3:b24.consecuencias" ||
		plazos.RecursoReglaHuellaSHA256 != strings.Repeat("d", 64) || plazos.SuspensionHasta != nil ||
		catalogo.clave != "b24.sancion.baja_llamamiento_directo" || !catalogo.fecha.Equal(time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("plazos: %+v", plazos)
	}
	catalogo.err = dominiobolsa.ErrSancionParticipacionInvalida
	if _, err := s.Recibir(context.Background(), []byte(c), h, time.Now(), 7); err != nil || buzon.recibidos[1].Plazos != nil {
		t.Fatalf("clave desconocida: se conserva sin efecto: %v", err)
	}
	catalogo.err = ports.ErrSancionesNoConfiguradas
	if _, err := s.Recibir(context.Background(), []byte(c), h, time.Now(), 7); !errors.Is(err, ports.ErrSancionesNoConfiguradas) || len(buzon.recibidos) != 2 {
		t.Fatalf("catálogo no disponible: no se entrega nada: %v", err)
	}
	catalogo.err = nil
	if _, err := s.Recibir(context.Background(), []byte(strings.Replace(c, "per_segunda", "per_otra", 1)), h, time.Now(), 7); !errors.Is(err, dominiobolsa.ErrEventoContratoParticipacionInvalido) {
		t.Fatalf("huella de otro contenido: %v", err)
	}
	for _, cambio := range [][2]string{{`"tipo": "no_incorporacion"`, `"tipo": "incorporacion"`}, {`"fecha_notificacion": "2026-09-20"`, `"fecha_notificacion": "2026-02-30"`},
		{`"motivo_clave": "no_presentado"`, `"motivo_clave": "No"`}, {`, "actor_ref": "per_actor"`, ``}} {
		otro := strings.Replace(c, cambio[0], cambio[1], 1)
		suma := sha256.Sum256([]byte(otro))
		if _, err := s.Recibir(context.Background(), []byte(otro), hex.EncodeToString(suma[:]), time.Now(), 7); !errors.Is(err, dominiobolsa.ErrEventoContratoParticipacionInvalido) {
			t.Fatalf("%v: %v", cambio, err)
		}
	}
	if _, err := NuevoServicioRecepcionNoIncorporaciones(buzon, nil); err == nil {
		t.Fatal("sin catálogo no hay bandeja")
	}
}

// Lo pendiente se reevalúa con plazos recalculados; una clave desconocida va
// sin plazos y una indisponibilidad del catálogo detiene la pasada.
func TestRecepcionNoIncorporacionesReevaluaLosPendientes(t *testing.T) {
	buzon, catalogo := &buzonNoIncorporacionesPrueba{}, &catalogoNoIncorporacionPrueba{}
	s, err := NuevoServicioRecepcionNoIncorporaciones(buzon, catalogo)
	if err != nil {
		t.Fatal(err)
	}
	buzon.pendientes = []ports.PendienteNoIncorporacion{{EventoRef: "evento:1", ConsecuenciaClave: "b24.sancion.baja_llamamiento_directo", FechaNotificacion: "2026-09-20"}}
	aplicadas, err := s.ReevaluarPendientes(context.Background(), 10)
	if err != nil || aplicadas != 1 || buzon.reevaluados["evento:1"] == nil || buzon.reevaluados["evento:1"].RecursoVence != "2026-10-20" ||
		!catalogo.fecha.Equal(time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("reevaluación: %d %v %+v", aplicadas, err, buzon.reevaluados)
	}
	catalogo.err = dominiobolsa.ErrSancionParticipacionInvalida
	if aplicadas, err := s.ReevaluarPendientes(context.Background(), 10); err != nil || aplicadas != 0 || buzon.reevaluados["evento:1"] != nil {
		t.Fatalf("clave desconocida: %d %v", aplicadas, err)
	}
	catalogo.err = ports.ErrSancionesNoConfiguradas
	if _, err := s.ReevaluarPendientes(context.Background(), 10); !errors.Is(err, ports.ErrSancionesNoConfiguradas) {
		t.Fatalf("catálogo no disponible: %v", err)
	}
	catalogo.err = nil
	buzon.pendientes[0].FechaNotificacion = "2026-02-30"
	if _, err := s.ReevaluarPendientes(context.Background(), 10); err == nil {
		t.Fatal("fecha inválida aceptada")
	}
	for _, limite := range []int{0, ports.LimitePendientesNoIncorporacion + 1} {
		if _, err := s.ReevaluarPendientes(context.Background(), limite); err == nil {
			t.Fatalf("límite %d aceptado", limite)
		}
	}
}

// Tras una aceptación la persona no se incorpora: el siguiente llamamiento
// continúa desde el terminal de aceptación con el tramo contiguo posterior
// (el guardado exige además la no incorporación en la bandeja, Bolsa 000042).
func TestIntegracionLlamamientosDesarrolloSiguienteTrasAceptacionSinIncorporacion(t *testing.T) {
	s, _, _, _, p := aperturaAceptacionIntegracionPrueba(t)
	p.OperacionRef = "operacion:aceptacion"
	if _, err := s.AceptarLlamamiento(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	sig, err := s.SolicitarSiguienteLlamamiento(context.Background(), ports.PeticionSiguienteLlamamientoDesarrollo{
		OperacionRef: "operacion:siguiente", IntencionRef: "intencion:siguiente", TerminalOperacionRef: p.OperacionRef})
	if err != nil {
		t.Fatal(err)
	}
	if sig.Registro.Tipo != "propuesta" || sig.Registro.Propuesta.Continuacion == nil ||
		sig.Registro.Propuesta.Continuacion.TerminalOperacionRef != p.OperacionRef || sig.Registro.Propuesta.OrdenSeleccionado != 3 ||
		sig.Registro.EstadoLlamamiento != dominiobolsa.EstadoLlamamientoAbierto || sig.Registro.Accion() != ports.AccionAbrirSiguienteLlamamientoDesarrollo {
		t.Fatal("siguiente tras aceptación sin incorporación incorrecto")
	}
}

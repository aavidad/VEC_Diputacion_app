package application

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type relojCorreoPrueba struct{ ahora time.Time }

func (r relojCorreoPrueba) Ahora() time.Time { return r.ahora }

type autorizadorCorreoPrueba struct {
	mu       sync.Mutex
	llamadas int
	err      error
	caduca   bool
	vacia    bool
	cruzada  bool
}

func (a *autorizadorCorreoPrueba) AutorizarDespachoCorreoLlamamiento(_ context.Context, s ports.SolicitudDespacharCorreoLlamamiento) (ports.CapacidadDespachoCorreoLlamamiento, error) {
	a.mu.Lock()
	a.llamadas++
	a.mu.Unlock()
	if a.err != nil {
		return ports.CapacidadDespachoCorreoLlamamiento{}, a.err
	}
	vencimiento := instanteCorreoPrueba.Add(4 * time.Second)
	if a.caduca {
		vencimiento = instanteCorreoPrueba
		return ports.NuevaCapacidadDespachoCorreoLlamamiento(s, materialCorreoPrueba(s, vencimiento), vencimiento.Add(-time.Second))
	}
	if a.vacia {
		return ports.CapacidadDespachoCorreoLlamamiento{}, nil
	}
	if a.cruzada {
		otro := s
		otro.IntencionEnvioRef = "intencion-ajena"
		return ports.NuevaCapacidadDespachoCorreoLlamamiento(otro, materialCorreoPrueba(otro, vencimiento), instanteCorreoPrueba)
	}
	c, e := ports.NuevaCapacidadDespachoCorreoLlamamiento(s, materialCorreoPrueba(s, vencimiento), instanteCorreoPrueba)
	if e != nil {
		panic(e)
	}
	return c, nil
}

type resolutorCorreoPrueba struct {
	llamadas                  int
	err                       error
	ultimo                    ports.SolicitudDespacharCorreoLlamamiento
	doble, cero               bool
	concurrente, ignorarError bool
}

func (r *resolutorCorreoPrueba) ConDestinoCorreoLlamamiento(_ context.Context, s ports.SolicitudDespacharCorreoLlamamiento, c ports.CapacidadDespachoCorreoLlamamiento, ejecutar func(ports.DestinoCorreoLlamamiento) error) error {
	r.llamadas++
	if r.cero {
		return nil
	}
	r.ultimo = s
	if r.err != nil || c.ValidarPara(s, instanteCorreoPrueba) != nil {
		return errors.New("ligadura no autorizada")
	}
	d, e := ports.NuevoDestinoCorreoLlamamiento("sucesor@ejemplo.invalid")
	if e != nil {
		return e
	}
	if r.concurrente {
		var wg sync.WaitGroup
		for range 2 {
			wg.Add(1)
			go func() { defer wg.Done(); _ = ejecutar(d) }()
		}
		wg.Wait()
		return nil
	}
	if e = ejecutar(d); e != nil {
		return e
	}
	if r.doble {
		e = ejecutar(d)
		if !r.ignorarError {
			return e
		}
	}
	return nil
}

type registroCorreoPrueba struct {
	mu                   sync.Mutex
	reserva              ports.ReservaIntentoCorreoLlamamiento
	reservas, resultados int
	falloResultado       error
	estado               ports.EstadoCorreoLlamamiento
	contextoValido       bool
	bloquear             bool
}

func (r *registroCorreoPrueba) ConsultarIntentoCorreoLlamamiento(_ context.Context, _ ports.SolicitudDespacharCorreoLlamamiento, _ ports.CapacidadDespachoCorreoLlamamiento) (ports.ReservaIntentoCorreoLlamamiento, ports.EstadoCorreoLlamamiento, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.reserva, r.estado, nil
}

func (r *registroCorreoPrueba) ReservarIntentoCorreoLlamamiento(_ context.Context, solicitud ports.SolicitudDespacharCorreoLlamamiento, _ ports.CapacidadDespachoCorreoLlamamiento) (ports.ReservaIntentoCorreoLlamamiento, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reservas++
	if r.reserva.IntentoRef != "" {
		v := r.reserva
		v.YaReservado = true
		return v, nil
	}
	b, _ := json.Marshal(solicitud)
	r.reserva = ports.ReservaIntentoCorreoLlamamiento{IntentoRef: "intento-001", MessageID: "<intento-001@vec.invalid>", FechaOrigen: instanteCorreoPrueba, SolicitudHuella: fmt.Sprintf("%x", sha256.Sum256(b)), Estado: ports.CorreoLlamamientoIniciado}
	return r.reserva, nil
}
func (r *registroCorreoPrueba) RegistrarResultadoIntentoCorreoLlamamiento(ctx context.Context, _ ports.ReservaIntentoCorreoLlamamiento, estado ports.EstadoCorreoLlamamiento, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resultados++
	limite, ok := ctx.Deadline()
	r.contextoValido = ctx.Err() == nil && ok && time.Until(limite) > 0 && time.Until(limite) <= tiempoMaximoRegistroResultadoCorreoLlamamiento
	if !r.contextoValido {
		return errors.New("contexto de registro invalido")
	}
	if r.bloquear {
		<-ctx.Done()
		return ctx.Err()
	}
	r.estado = estado
	if r.falloResultado == nil {
		r.reserva.Estado = estado
	}
	return r.falloResultado
}

type transporteCorreoPrueba struct {
	mu       sync.Mutex
	envios   int
	cancelar func()
	estado   ports.EstadoCorreoLlamamiento
	ultimo   ports.MensajeCorreoLlamamiento
}

func (t *transporteCorreoPrueba) Enviar(_ context.Context, m ports.MensajeCorreoLlamamiento) ports.EstadoCorreoLlamamiento {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.envios++
	t.ultimo = m
	if t.cancelar != nil {
		t.cancelar()
	}
	return t.estado
}
func (t *transporteCorreoPrueba) total() int { t.mu.Lock(); defer t.mu.Unlock(); return t.envios }

var instanteCorreoPrueba = time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)

func solicitudCorreoPrueba() ports.SolicitudDespacharCorreoLlamamiento {
	return ports.SolicitudDespacharCorreoLlamamiento{OrganizacionRef: "org-001", ExpedienteRef: "exp-001", LlamamientoRef: "llamamiento-001", ComunicacionRef: "comunicacion-ct54", IntencionEnvioRef: "intencion-ct54"}
}
func materialCorreoPrueba(s ports.SolicitudDespacharCorreoLlamamiento, hasta time.Time) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	h := strings.Repeat("a", 64)
	r, e := ports.RecursoDespachoCorreoLlamamiento(s)
	if e != nil {
		panic(e)
	}
	huella, e := r.HuellaContextoAutorizacionSHA256()
	if e != nil {
		panic(e)
	}
	resumen, e := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("decision-001", h, h, "contexto-001", h, ports.AccionDespacharCorreoLlamamiento, s.IntencionEnvioRef, huella, ports.AudienciaDespachoCorreoLlamamientoV3, hasta.Add(-5*time.Second), hasta)
	if e != nil {
		panic(e)
	}
	k := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{7}, ed25519.SeedSize))
	raiz, e := x509.MarshalPKIXPublicKey(k.Public())
	if e != nil {
		panic(e)
	}
	m, e := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte{5}, vecports.TamanoMinimoCapacidadCanonicaV3), resumen, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("cose"), []byte("evidencia"), raiz)
	if e != nil {
		panic(e)
	}
	return m
}
func servicioCorreoPrueba(t *testing.T, a *autorizadorCorreoPrueba, r *resolutorCorreoPrueba, registro *registroCorreoPrueba, transporte *transporteCorreoPrueba) *ServicioCorreoLlamamiento {
	t.Helper()
	s, e := NuevoServicioCorreoLlamamiento(a, r, registro, transporte, relojCorreoPrueba{instanteCorreoPrueba})
	if e != nil {
		t.Fatal(e)
	}
	return s
}

func TestCorreoLlamamientoReservaUnaVezAnteConcurrenciaYReplay(t *testing.T) {
	a := &autorizadorCorreoPrueba{}
	r := &resolutorCorreoPrueba{}
	registro := &registroCorreoPrueba{}
	transporte := &transporteCorreoPrueba{estado: ports.CorreoLlamamientoAceptadoPorRelay}
	s := servicioCorreoPrueba(t, a, r, registro, transporte)
	var wg sync.WaitGroup
	errores := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := s.Despachar(context.Background(), solicitudCorreoPrueba())
			errores <- e
		}()
	}
	wg.Wait()
	close(errores)
	for e := range errores {
		if e != nil {
			t.Fatal(e)
		}
	}
	if transporte.total() != 1 || registro.resultados != 1 {
		t.Fatalf("envios=%d resultados=%d, quiere uno", transporte.total(), registro.resultados)
	}
}

func TestCorreoLlamamientoDeniegaAntesDeResolverOEnviar(t *testing.T) {
	for nombre, a := range map[string]*autorizadorCorreoPrueba{"denegada": {err: errors.New("denegada")}, "caducada": {caduca: true}, "material_vacio": {vacia: true}, "material_cruzado": {cruzada: true}} {
		t.Run(nombre, func(t *testing.T) {
			r := &resolutorCorreoPrueba{}
			registro := &registroCorreoPrueba{}
			transporte := &transporteCorreoPrueba{estado: ports.CorreoLlamamientoAceptadoPorRelay}
			_, e := servicioCorreoPrueba(t, a, r, registro, transporte).Despachar(context.Background(), solicitudCorreoPrueba())
			if !errors.Is(e, ErrCorreoLlamamientoDenegado) || r.llamadas != 0 || registro.reservas != 0 || transporte.total() != 0 {
				t.Fatalf("error=%v resolver=%d reservas=%d envios=%d", e, r.llamadas, registro.reservas, transporte.total())
			}
		})
	}
}

func TestCorreoLlamamientoBloqueaDestinoAjeno(t *testing.T) {
	a := &autorizadorCorreoPrueba{}
	r := &resolutorCorreoPrueba{err: errors.New("destino ajeno")}
	registro := &registroCorreoPrueba{}
	transporte := &transporteCorreoPrueba{estado: ports.CorreoLlamamientoAceptadoPorRelay}
	_, e := servicioCorreoPrueba(t, a, r, registro, transporte).Despachar(context.Background(), solicitudCorreoPrueba())
	if !errors.Is(e, ErrCorreoLlamamientoDenegado) || registro.reservas != 1 || registro.resultados != 1 || transporte.total() != 0 {
		t.Fatalf("destino ajeno: %v", e)
	}
}

func TestCorreoLlamamientoFalloAlGuardarNoReenvia(t *testing.T) {
	a := &autorizadorCorreoPrueba{}
	r := &resolutorCorreoPrueba{}
	registro := &registroCorreoPrueba{falloResultado: errors.New("caida")}
	transporte := &transporteCorreoPrueba{estado: ports.CorreoLlamamientoAceptadoPorRelay}
	s := servicioCorreoPrueba(t, a, r, registro, transporte)
	if _, e := s.Despachar(context.Background(), solicitudCorreoPrueba()); !errors.Is(e, ErrCorreoLlamamientoNoDisponible) {
		t.Fatal(e)
	}
	if _, e := s.Despachar(context.Background(), solicitudCorreoPrueba()); e != nil {
		t.Fatal(e)
	}
	if transporte.total() != 1 {
		t.Fatalf("reenvio ciego: %d", transporte.total())
	}
}

func TestCorreoLlamamientoRechazaCallbacksCeroODobleSinSegundoSMTP(t *testing.T) {
	for nombre, r := range map[string]*resolutorCorreoPrueba{"cero": {cero: true}, "doble": {doble: true}, "doble_error_ignorado": {doble: true, ignorarError: true}, "concurrente": {concurrente: true}} {
		t.Run(nombre, func(t *testing.T) {
			a := &autorizadorCorreoPrueba{}
			registro := &registroCorreoPrueba{}
			transporte := &transporteCorreoPrueba{estado: ports.CorreoLlamamientoAceptadoPorRelay}
			_, e := servicioCorreoPrueba(t, a, r, registro, transporte).Despachar(context.Background(), solicitudCorreoPrueba())
			if !errors.Is(e, ErrCorreoLlamamientoDenegado) || transporte.total() > 1 || registro.resultados != 1 {
				t.Fatalf("error=%v envios=%d resultados=%d", e, transporte.total(), registro.resultados)
			}
		})
	}
}

func TestCorreoLlamamientoEnumSMTPInvalidoSeVuelveIndeterminado(t *testing.T) {
	a := &autorizadorCorreoPrueba{}
	r := &resolutorCorreoPrueba{}
	registro := &registroCorreoPrueba{}
	transporte := &transporteCorreoPrueba{estado: 99}
	_, e := servicioCorreoPrueba(t, a, r, registro, transporte).Despachar(context.Background(), solicitudCorreoPrueba())
	if !errors.Is(e, ErrCorreoLlamamientoDenegado) || registro.estado != ports.CorreoLlamamientoIndeterminado {
		t.Fatalf("error=%v estado=%d", e, registro.estado)
	}
}

func TestCorreoLlamamientoRegistraTrasCancelacionYNoMutaCT54(t *testing.T) {
	a := &autorizadorCorreoPrueba{}
	r := &resolutorCorreoPrueba{}
	registro := &registroCorreoPrueba{}
	ctx, cancelar := context.WithCancel(context.Background())
	transporte := &transporteCorreoPrueba{estado: ports.CorreoLlamamientoIndeterminado, cancelar: cancelar}
	s := servicioCorreoPrueba(t, a, r, registro, transporte)
	solicitud := solicitudCorreoPrueba()
	original := solicitud
	_, e := s.Despachar(ctx, solicitud)
	if !errors.Is(e, context.Canceled) || !registro.contextoValido || registro.resultados != 1 || registro.estado != ports.CorreoLlamamientoIndeterminado || solicitud != original {
		t.Fatalf("error=%v resultados=%d estado=%d mutada=%v", e, registro.resultados, registro.estado, solicitud != original)
	}
	if got := fmt.Sprintf("%v", transporte.ultimo); got != "MensajeCorreoLlamamiento{redactado}" {
		t.Fatal(got)
	}
	b, e := json.Marshal(transporte.ultimo)
	if e != nil || string(b) != "{\"redactado\":true}" {
		t.Fatalf("json=%s err=%v", b, e)
	}
}

func TestCorreoLlamamientoCapacidadVigenteCaducidadYLigadura(t *testing.T) {
	s := solicitudCorreoPrueba()
	hasta := instanteCorreoPrueba.Add(time.Second)
	capacidad, err := ports.NuevaCapacidadDespachoCorreoLlamamiento(s, materialCorreoPrueba(s, hasta), instanteCorreoPrueba)
	if err != nil {
		t.Fatal(err)
	}
	if capacidad.ValidarPara(s, instanteCorreoPrueba) != nil {
		t.Fatal("rechaza vigente")
	}
	if capacidad.ValidarPara(s, hasta) == nil {
		t.Fatal("acepta caducada")
	}
	for _, campo := range []string{"org", "exp", "llam", "com", "int"} {
		otra := s
		switch campo {
		case "org":
			otra.OrganizacionRef = "org-ajena"
		case "exp":
			otra.ExpedienteRef = "exp-ajeno"
		case "llam":
			otra.LlamamientoRef = "llam-ajeno"
		case "com":
			otra.ComunicacionRef = "com-ajena"
		case "int":
			otra.IntencionEnvioRef = "int-ajena"
		}
		if capacidad.ValidarPara(otra, instanteCorreoPrueba) == nil {
			t.Fatalf("acepta cruce %s", campo)
		}
	}
}
func TestCorreoLlamamientoLimitaRegistroYConservaReserva(t *testing.T) {
	registro := &registroCorreoPrueba{bloquear: true}
	transporte := &transporteCorreoPrueba{estado: ports.CorreoLlamamientoAceptadoPorRelay}
	servicio := servicioCorreoPrueba(t, &autorizadorCorreoPrueba{}, &resolutorCorreoPrueba{}, registro, transporte)
	inicio := time.Now()
	reserva, err := servicio.Despachar(context.Background(), solicitudCorreoPrueba())
	if !errors.Is(err, ErrCorreoLlamamientoNoDisponible) || reserva.IntentoRef == "" || !registro.contextoValido {
		t.Fatalf("registro sin limite/reserva: %v", err)
	}
	if time.Since(inicio) < tiempoMaximoRegistroResultadoCorreoLlamamiento {
		t.Fatal("no ejercita timeout real")
	}
	if _, err = servicio.Despachar(context.Background(), solicitudCorreoPrueba()); err != nil || transporte.total() != 1 {
		t.Fatal("replay reenvia o falla", err)
	}
}

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
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"

	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/i18n"
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

type autorizadorResultadoCorreoPrueba struct {
	err      error
	llamadas int
	caduca   bool
}

func (a *autorizadorResultadoCorreoPrueba) AutorizarResultadoCorreoLlamamiento(_ context.Context, s ports.SolicitudRegistrarResultadoCorreoLlamamiento, auditoria ports.AuditoriaResultadoCorreoLlamamiento) (ports.CapacidadResultadoCorreoLlamamiento, error) {
	a.llamadas++
	if a.err != nil {
		return ports.CapacidadResultadoCorreoLlamamiento{}, a.err
	}
	if a.caduca {
		return ports.TransportarMaterialResultadoCorreoLlamamiento(materialResultadoCorreoPrueba(s, auditoria, instanteCorreoPrueba)), nil
	}
	return NuevaCapacidadResultadoCorreoLlamamiento(s, auditoria, materialResultadoCorreoPrueba(s, auditoria, instanteCorreoPrueba.Add(4*time.Second)), instanteCorreoPrueba)
}

type preparadorAuditoriaCorreoPrueba struct {
	err      error
	llamadas int
}

func (p *preparadorAuditoriaCorreoPrueba) PrepararAuditoriaResultadoCorreoLlamamiento(_ context.Context, s ports.SolicitudRegistrarResultadoCorreoLlamamiento, _ ports.CapacidadDespachoCorreoLlamamiento, ocurrido time.Time) (ports.AuditoriaResultadoCorreoLlamamiento, error) {
	p.llamadas++
	if p.err != nil {
		return ports.AuditoriaResultadoCorreoLlamamiento{}, p.err
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return ports.AuditoriaResultadoCorreoLlamamiento{}, err
	}
	return ctdomain.NuevaAuditoriaResultadoCorreoLlamamiento(ctdomain.DatosAuditoriaResultadoCorreoLlamamiento{
		ActorID: "hmac-sha256:prueba:" + strings.Repeat("a", 64), ActorProfile: "perfil-rrhh", VersionRolRef: "rol-version-rrhh-1", AuthMethod: "certificado", AuthAssurance: "alto", Correlacion: correlacion, Solicitud: s, OcurridoEn: ocurrido,
	})
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
		return NuevaCapacidadDespachoCorreoLlamamiento(s, materialCorreoPrueba(s, vencimiento), vencimiento.Add(-time.Second))
	}
	if a.vacia {
		return ports.CapacidadDespachoCorreoLlamamiento{}, nil
	}
	if a.cruzada {
		otro := s
		otro.IntencionEnvioRef = "intencion-ajena"
		return NuevaCapacidadDespachoCorreoLlamamiento(otro, materialCorreoPrueba(otro, vencimiento), instanteCorreoPrueba)
	}
	c, e := NuevaCapacidadDespachoCorreoLlamamiento(s, materialCorreoPrueba(s, vencimiento), instanteCorreoPrueba)
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
	if r.err != nil || ValidarCapacidadDespachoCorreoLlamamiento(c, s, instanteCorreoPrueba) != nil {
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
	finalizacion         ports.CapacidadFinalizacionIntentoCorreoLlamamiento
	modoFinalizacion     string
	reservas, resultados int
	falloResultado       error
	estado               ports.EstadoCorreoLlamamiento
	contextoValido       bool
	finalizacionValida   bool
	bloquear             bool
}

func (r *registroCorreoPrueba) ReservarIntentoCorreoLlamamiento(_ context.Context, solicitud ports.SolicitudDespacharCorreoLlamamiento, _ ports.CapacidadDespachoCorreoLlamamiento) (ports.ReservaIntentoCorreoLlamamiento, ports.CapacidadFinalizacionIntentoCorreoLlamamiento, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reservas++
	if r.reserva.IntentoRef != "" {
		v := r.reserva
		v.YaReservado = true
		if r.modoFinalizacion == "replay" {
			return v, r.finalizacion, nil
		}
		return v, ports.CapacidadFinalizacionIntentoCorreoLlamamiento{}, nil
	}
	b, _ := json.Marshal(solicitud)
	r.reserva = ports.ReservaIntentoCorreoLlamamiento{IntentoRef: "intento-001", MessageID: "<intento-001@vec.invalid>", FechaOrigen: instanteCorreoPrueba, SolicitudHuella: fmt.Sprintf("%x", sha256.Sum256(b)), Estado: ports.CorreoLlamamientoIniciado}
	var err error
	r.finalizacion, err = ctdomain.NuevaCapacidadFinalizacionIntentoCorreoLlamamiento(r.reserva, bytes.Repeat([]byte{1}, 32))
	if err != nil {
		return ports.ReservaIntentoCorreoLlamamiento{}, ports.CapacidadFinalizacionIntentoCorreoLlamamiento{}, err
	}
	if r.modoFinalizacion == "cero" {
		r.finalizacion = ports.CapacidadFinalizacionIntentoCorreoLlamamiento{}
	}
	if r.modoFinalizacion == "otro" || r.modoFinalizacion == "huella" {
		o := r.reserva
		if r.modoFinalizacion == "otro" {
			o.IntentoRef = "intento-otro"
		} else {
			o.SolicitudHuella = strings.Repeat("b", 64)
		}
		r.finalizacion, err = ctdomain.NuevaCapacidadFinalizacionIntentoCorreoLlamamiento(o, bytes.Repeat([]byte{1}, 32))
		if err != nil {
			return ports.ReservaIntentoCorreoLlamamiento{}, ports.CapacidadFinalizacionIntentoCorreoLlamamiento{}, err
		}
	}
	return r.reserva, r.finalizacion, nil
}
func (r *registroCorreoPrueba) RegistrarResultadoIntentoCorreoLlamamiento(ctx context.Context, solicitud ports.SolicitudRegistrarResultadoCorreoLlamamiento, finalizacion ports.CapacidadFinalizacionIntentoCorreoLlamamiento, auditoria ports.AuditoriaResultadoCorreoLlamamiento, resultado ports.CapacidadResultadoCorreoLlamamiento) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resultados++
	if ValidarCapacidadResultadoCorreoLlamamiento(resultado, solicitud, auditoria, instanteCorreoPrueba) != nil {
		return errors.New("capacidad resultado invalida")
	}
	reserva := r.reserva
	estado := solicitud.Estado
	if finalizacion.ValidarPara(reserva) != nil || !bytes.Equal(finalizacion.ExportarSecretoParaConsumidor(), r.finalizacion.ExportarSecretoParaConsumidor()) {
		return errors.New("capacidad final invalida")
	}
	r.finalizacionValida = true
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
	r, e := RecursoDespachoCorreoLlamamiento(s)
	if e != nil {
		panic(e)
	}
	huella, e := r.HuellaContextoAutorizacionSHA256()
	if e != nil {
		panic(e)
	}
	resumen, e := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("decision-001", h, h, "contexto-001", h, AccionDespacharCorreoLlamamiento, s.IntencionEnvioRef, huella, AudienciaDespachoCorreoLlamamientoV3, hasta.Add(-5*time.Second), hasta)
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

func materialResultadoCorreoPrueba(s ports.SolicitudRegistrarResultadoCorreoLlamamiento, auditoria ports.AuditoriaResultadoCorreoLlamamiento, hasta time.Time) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	h := strings.Repeat("a", 64)
	r, err := RecursoResultadoCorreoLlamamiento(s, auditoria)
	if err != nil {
		panic(err)
	}
	huella, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil {
		panic(err)
	}
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("decision-resultado-001", h, h, "contexto-001", h, AccionRegistrarResultadoCorreoLlamamiento, s.IntentoRef, huella, AudienciaResultadoCorreoLlamamientoV3, hasta.Add(-5*time.Second), hasta)
	if err != nil {
		panic(err)
	}
	k := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{8}, ed25519.SeedSize))
	raiz, err := x509.MarshalPKIXPublicKey(k.Public())
	if err != nil {
		panic(err)
	}
	m, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte{6}, vecports.TamanoMinimoCapacidadCanonicaV3), resumen, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("cose"), []byte("evidencia"), raiz)
	if err != nil {
		panic(err)
	}
	return m
}
func servicioCorreoPrueba(t *testing.T, a *autorizadorCorreoPrueba, r *resolutorCorreoPrueba, registro *registroCorreoPrueba, transporte *transporteCorreoPrueba) *ServicioCorreoLlamamiento {
	t.Helper()
	traductor, e := i18n.New(i18n.DefaultLocale, map[string]map[string]string{i18n.DefaultLocale: {
		claveAsuntoCorreoLlamamiento: "Asunto RRHH",
		claveCuerpoCorreoLlamamiento: "Cuerpo RRHH",
	}})
	if e != nil {
		t.Fatal(e)
	}
	s, e := NuevoServicioCorreoLlamamiento(a, &autorizadorResultadoCorreoPrueba{}, &preparadorAuditoriaCorreoPrueba{}, r, registro, transporte, relojCorreoPrueba{instanteCorreoPrueba}, traductor)
	if e != nil {
		t.Fatal(e)
	}
	return s
}

func TestCorreoLlamamientoObtieneTextoDelCatalogoComun(t *testing.T) {
	traductor, err := i18n.LoadDir("../../../../locales")
	if err != nil {
		t.Fatal(err)
	}
	destino, err := ports.NuevoDestinoCorreoLlamamiento("persona@ejemplo.invalid")
	if err != nil {
		t.Fatal(err)
	}
	r := ports.ReservaIntentoCorreoLlamamiento{MessageID: "<m@vec.invalid>", FechaOrigen: instanteCorreoPrueba}
	mensaje, err := destinoMensajeCorreo(destino, r, traductor)
	if err != nil {
		t.Fatal(err)
	}
	_, asunto, cuerpo := mensaje.Datos()
	if asunto != "Llamamiento de contratación temporal" || cuerpo != "Tiene disponible una comunicación de llamamiento de contratación temporal. Consulte su contenido a través del canal indicado por Recursos Humanos." {
		t.Fatalf("texto no traducido: %q / %q", asunto, cuerpo)
	}
}

func TestCorreoLlamamientoRechazaCatalogoSinTextoRequerido(t *testing.T) {
	traductor, err := i18n.New(i18n.DefaultLocale, map[string]map[string]string{i18n.DefaultLocale: {
		claveAsuntoCorreoLlamamiento: "Asunto del catálogo",
	}})
	if err != nil {
		t.Fatal(err)
	}
	destino, err := ports.NuevoDestinoCorreoLlamamiento("persona@ejemplo.invalid")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = destinoMensajeCorreo(destino, ports.ReservaIntentoCorreoLlamamiento{}, traductor); err != ErrServicioCorreoLlamamientoInvalido {
		t.Fatalf("error=%v", err)
	}
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
func TestCorreoLlamamientoNiegaCapacidadFinalizacionAntesSMTP(t *testing.T) {
	for _, m := range []string{"cero", "otro", "huella", "replay"} {
		t.Run(m, func(t *testing.T) {
			a := &autorizadorCorreoPrueba{}
			r := &resolutorCorreoPrueba{}
			g := &registroCorreoPrueba{modoFinalizacion: m}
			if m == "replay" {
				// El primer acceso construye una capacidad válida para la reserva
				// nueva; el servicio recibe luego un replay con ese secreto indebido.
				reserva, capacidad, err := g.ReservarIntentoCorreoLlamamiento(context.Background(), solicitudCorreoPrueba(), ports.CapacidadDespachoCorreoLlamamiento{})
				if err != nil || capacidad.ValidarPara(reserva) != nil {
					t.Fatal("el control de replay no tiene capacidad válida")
				}
			}
			tr := &transporteCorreoPrueba{estado: ports.CorreoLlamamientoAceptadoPorRelay}
			_, e := servicioCorreoPrueba(t, a, r, g, tr).Despachar(context.Background(), solicitudCorreoPrueba())
			if g.reserva.ValidarPara(solicitudCorreoPrueba()) != nil || (m != "cero" && g.finalizacion.EsCero()) {
				t.Fatal("el negativo no alcanzó la frontera con reserva y capacidad controladas")
			}
			if !errors.Is(e, ErrResultadoCorreoLlamamientoNoConfiable) || r.llamadas != 0 || tr.total() != 0 || g.resultados != 0 {
				t.Fatalf("error=%v resoluciones=%d envíos=%d resultados=%d", e, r.llamadas, tr.total(), g.resultados)
			}
		})
	}
}

func TestCorreoLlamamientoDevuelveReservaSinCapacidadFinalizacion(t *testing.T) {
	a := &autorizadorCorreoPrueba{}
	registro := &registroCorreoPrueba{}
	servicio := servicioCorreoPrueba(t, a, &resolutorCorreoPrueba{}, registro, &transporteCorreoPrueba{estado: ports.CorreoLlamamientoAceptadoPorRelay})
	reserva, err := servicio.Despachar(context.Background(), solicitudCorreoPrueba())
	if err != nil || !registro.finalizacionValida || a.llamadas != 1 {
		t.Fatal("el control positivo no finalizó con su capacidad")
	}
	b, err := json.Marshal(reserva)
	if err != nil {
		t.Fatal(err)
	}
	var campos map[string]json.RawMessage
	if err := json.Unmarshal(b, &campos); err != nil {
		t.Fatal(err)
	}
	publicos := []string{"IntentoRef", "MessageID", "FechaOrigen", "YaReservado", "SolicitudHuella", "Estado"}
	if len(campos) != len(publicos) {
		t.Fatal("el recibo contiene campos ajenos al contrato público")
	}
	for _, nombre := range publicos {
		if _, ok := campos[nombre]; !ok {
			t.Fatalf("falta el campo público %s", nombre)
		}
	}
	texto := strings.ToLower(fmt.Sprintf("%v %+v %#v", reserva, reserva, reserva))
	if strings.Contains(texto, "finalizacion") || strings.Contains(texto, "secreto") || strings.Contains(texto, fmt.Sprintf("%x", registro.finalizacion.ExportarSecretoParaConsumidor())) {
		t.Fatal("el recibo expone material de finalización al formatearse")
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
	if !errors.Is(e, context.Canceled) || !registro.contextoValido || !registro.finalizacionValida || a.llamadas != 1 || registro.resultados != 1 || registro.estado != ports.CorreoLlamamientoIndeterminado || solicitud != original {
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
	capacidad, err := NuevaCapacidadDespachoCorreoLlamamiento(s, materialCorreoPrueba(s, hasta), instanteCorreoPrueba)
	if err != nil {
		t.Fatal(err)
	}
	if ValidarCapacidadDespachoCorreoLlamamiento(capacidad, s, instanteCorreoPrueba) != nil {
		t.Fatal("rechaza vigente")
	}
	if ValidarCapacidadDespachoCorreoLlamamiento(capacidad, s, hasta) == nil {
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
		if ValidarCapacidadDespachoCorreoLlamamiento(capacidad, otra, instanteCorreoPrueba) == nil {
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

func TestServicioCorreoLlamamientoExigePreparadorAuditoriaAntesSMTP(t *testing.T) {
	traductor, err := i18n.New(i18n.DefaultLocale, map[string]map[string]string{i18n.DefaultLocale: {claveAsuntoCorreoLlamamiento: "Asunto", claveCuerpoCorreoLlamamiento: "Cuerpo"}})
	if err != nil {
		t.Fatal(err)
	}
	if servicio, err := NuevoServicioCorreoLlamamiento(&autorizadorCorreoPrueba{}, &autorizadorResultadoCorreoPrueba{}, nil, &resolutorCorreoPrueba{}, &registroCorreoPrueba{}, &transporteCorreoPrueba{}, relojCorreoPrueba{instanteCorreoPrueba}, traductor); err != ErrServicioCorreoLlamamientoInvalido || servicio != nil {
		t.Fatalf("constructor aceptó preparador ausente: servicio=%v err=%v", servicio, err)
	}
}

func TestCorreoLlamamientoNoAceptaCapacidadResultadoCaducadaTrasSMTP(t *testing.T) {
	traductor, err := i18n.New(i18n.DefaultLocale, map[string]map[string]string{i18n.DefaultLocale: {claveAsuntoCorreoLlamamiento: "Asunto", claveCuerpoCorreoLlamamiento: "Cuerpo"}})
	if err != nil {
		t.Fatal(err)
	}
	registro := &registroCorreoPrueba{}
	transporte := &transporteCorreoPrueba{estado: ports.CorreoLlamamientoAceptadoPorRelay}
	servicio, err := NuevoServicioCorreoLlamamiento(&autorizadorCorreoPrueba{}, &autorizadorResultadoCorreoPrueba{caduca: true}, &preparadorAuditoriaCorreoPrueba{}, &resolutorCorreoPrueba{}, registro, transporte, relojCorreoPrueba{instanteCorreoPrueba}, traductor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = servicio.Despachar(context.Background(), solicitudCorreoPrueba()); !errors.Is(err, ErrCorreoLlamamientoNoDisponible) || transporte.total() != 1 || registro.resultados != 0 {
		t.Fatalf("capacidad vencida finalizó o reenvió: err=%v envíos=%d resultados=%d", err, transporte.total(), registro.resultados)
	}
}

func TestRecursoResultadoCorreoLigaSolicitudYAuditoriaSeparadamente(t *testing.T) {
	despacho := solicitudCorreoPrueba()
	huella, err := despacho.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	reserva := ports.ReservaIntentoCorreoLlamamiento{IntentoRef: "intento-001", MessageID: "<m@vec.invalid>", FechaOrigen: instanteCorreoPrueba, SolicitudHuella: huella, Estado: ports.CorreoLlamamientoIniciado}
	resultado, err := ports.NuevaSolicitudRegistrarResultadoCorreoLlamamiento(despacho, reserva, ports.CorreoLlamamientoAceptadoPorRelay, ports.PlantillaCorreoLlamamientoV1)
	if err != nil {
		t.Fatal(err)
	}
	crear := func() ports.AuditoriaResultadoCorreoLlamamiento {
		correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), seguridadvec.GeneradorReferenciasCriptograficas{})
		if err != nil {
			t.Fatal(err)
		}
		a, e := ctdomain.NuevaAuditoriaResultadoCorreoLlamamiento(ctdomain.DatosAuditoriaResultadoCorreoLlamamiento{ActorID: "hmac-sha256:prueba:" + strings.Repeat("a", 64), ActorProfile: "perfil-rrhh", VersionRolRef: "rol-version-rrhh-1", AuthMethod: "certificado", AuthAssurance: "alto", Correlacion: correlacion, Solicitud: resultado, OcurridoEn: instanteCorreoPrueba})
		if e != nil {
			t.Fatal(e)
		}
		return a
	}
	primera, segunda := crear(), crear()
	recurso, err := RecursoResultadoCorreoLlamamiento(resultado, primera)
	if err != nil || recurso.Atributos["material_sha256"] == "" || recurso.Atributos["auditoria_sha256"] == "" {
		t.Fatalf("recurso=%#v err=%v", recurso, err)
	}
	otro, err := RecursoResultadoCorreoLlamamiento(resultado, segunda)
	if err != nil || recurso.Atributos["material_sha256"] != otro.Atributos["material_sha256"] || recurso.Atributos["auditoria_sha256"] == otro.Atributos["auditoria_sha256"] {
		t.Fatalf("huellas=%#v %#v", recurso.Atributos, otro.Atributos)
	}
}

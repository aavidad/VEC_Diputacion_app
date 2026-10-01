package registrocopias

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
	port "vec-diputacion-granada/internal/modules/administracion/ports/registrocopias"
)

type observadorFunc func(context.Context, port.Declaracion, port.SolicitudAbandono) (operacionescopias.ObservacionAbandono, error)

func (o observadorFunc) ConfirmarAbandono(ctx context.Context, d port.Declaracion, s port.SolicitudAbandono) (operacionescopias.ObservacionAbandono, error) {
	return o(ctx, d, s)
}
func pedidoAbandono() port.SolicitudAbandono {
	s := solicitud()
	return port.SolicitudAbandono{Operacion: s.Operacion, Clave: "abandono:fallo", VersionEsperada: 1, SolicitudSHA256: s.SHA256, Destino: s.Destino, FalloReferencia: "captura_fallida", FalloSHA256: strings.Repeat("f", 64)}
}
func observacionConfirmada(s port.SolicitudAbandono) operacionescopias.ObservacionAbandono {
	return operacionescopias.ObservacionAbandono{Operacion: s.Operacion, Destino: s.Destino, FalloReferencia: s.FalloReferencia, FalloSHA256: s.FalloSHA256, Lease: "lease:captura", EstadoEfecto: "inactivo", EstadoLease: "cancelada", EstadoPlataforma: "sin_efectos_pendientes"}
}
func abrirObservado(t *testing.T, cfg Config, o port.ObservadorAbandono) *Fichero {
	t.Helper()
	f, e := AbrirConObservadorAbandono(cfg, o)
	if e != nil {
		t.Fatal(e)
	}
	return f
}
func iniciar(t *testing.T, f *Fichero) port.Resultado {
	t.Helper()
	reservar(t, f)
	r, e := f.Aplicar(context.Background(), declaracion, solicitud().Operacion, operacionescopias.Comando{Clave: "captura:inicio", SolicitudSHA256: solicitud().SHA256, Accion: "iniciar_captura"})
	if e != nil {
		t.Fatal(e)
	}
	return r
}

func TestAbandonoDurableReintentoOtroConjuntoYReplay(t *testing.T) {
	cfg := pruebaConfig(t)
	calls := 0
	observer := observadorFunc(func(_ context.Context, d port.Declaracion, s port.SolicitudAbandono) (operacionescopias.ObservacionAbandono, error) {
		calls++
		if d != declaracion {
			t.Fatal("wrong actor")
		}
		return observacionConfirmada(s), nil
	})
	f := abrirObservado(t, cfg, observer)
	inicio := iniciar(t, f)
	pedido := pedidoAbandono()
	r, e := f.AbandonarCaptura(context.Background(), declaracion, pedido)
	if e != nil || r.Replay || r.Recibo.Estado != operacionescopias.AbandonadaDeclarada || r.Recibo.Version != 2 || r.Auditoria.Accion != "abandonar_captura" || r.Auditoria.Resultado != "registrado" {
		t.Fatal(e, r)
	}
	if len(r.Historia) != 2 || r.Historia[0].Comando.Accion != "iniciar_captura" || r.Historia[1].Comando.Evidencia != nil {
		t.Fatal("false trial/history")
	}
	q, e := abrir(t, cfg).Consultar(context.Background(), declaracion, pedido.Operacion)
	if e != nil || q.Recibo != r.Recibo || q.Recibo == inicio.Recibo {
		t.Fatal("reopen", e)
	}
	retry := solicitud()
	retry.Operacion, retry.Clave, retry.Conjunto = "op:reintento", "clave:reintento", "conjunto:reintento"
	if _, e := abrir(t, cfg).Reservar(context.Background(), declaracion, retry); e != nil {
		t.Fatal("destination still busy", e)
	}
	replay, e := abrirObservado(t, cfg, observer).AbandonarCaptura(context.Background(), declaracion, pedido)
	if e != nil || !replay.Replay || replay.Recibo != r.Recibo || len(replay.Historia) != 2 || calls != 2 {
		t.Fatal("replay/revalidation", e, calls)
	}
	pedido.FalloSHA256 = strings.Repeat("a", 64)
	if _, e := f.AbandonarCaptura(context.Background(), declaracion, pedido); !errors.Is(e, operacionescopias.ErrConflicto) {
		t.Fatal(e)
	}
}

func TestAbandonoDenegadoInciertoActivoOMantenimientoConservaDestino(t *testing.T) {
	for _, mode := range []string{"sin_observador", "denegado", "incierto", "activo", "lease_vigente", "restaurando", "datos_privados", "estado_privado", "verificador_activo", "verificador_incierto", "ventana_activa", "ventana_incierta"} {
		t.Run(mode, func(t *testing.T) {
			cfg := pruebaConfig(t)
			var observer port.ObservadorAbandono
			if mode != "sin_observador" {
				observer = observadorFunc(func(_ context.Context, _ port.Declaracion, s port.SolicitudAbandono) (operacionescopias.ObservacionAbandono, error) {
					a := observacionConfirmada(s)
					switch mode {
					case "denegado":
						return a, errors.New("secret-must-not-be-logged")
					case "incierto":
						a.EstadoEfecto = "incierto"
					case "activo":
						a.EstadoEfecto = "activo"
					case "lease_vigente":
						a.EstadoLease = "vigente"
					case "restaurando":
						a.EstadoPlataforma = "restaurando"
					case "datos_privados":
						a.Lease = "/private/not-allowed"
					case "estado_privado":
						a.EstadoEfecto = "secret-must-not-be-logged"
					case "verificador_activo":
						a.EstadoVerificador = "activo"
					case "verificador_incierto":
						a.EstadoVerificador = "incierto"
					case "ventana_activa":
						a.EstadoVentana = "activa"
					case "ventana_incierta":
						a.EstadoVentana = "incierta"
					}
					return a, nil
				})
			}
			f := abrirObservado(t, cfg, observer)
			before := iniciar(t, f)
			r, e := f.AbandonarCaptura(context.Background(), declaracion, pedidoAbandono())
			if e == nil || r.Recibo.Referencia != "" || r.Auditoria.Resultado == "registrado" {
				t.Fatal("uncertain release", e, r)
			}
			q, e := abrir(t, cfg).Consultar(context.Background(), declaracion, solicitud().Operacion)
			if e != nil || q.Recibo != before.Recibo || len(q.Historia) != 1 {
				t.Fatal("state changed", e)
			}
			s := solicitud()
			s.Operacion, s.Clave = "op:retry", "key:retry"
			if _, e := f.Reservar(context.Background(), declaracion, s); !errors.Is(e, port.ErrDestinoOcupado) {
				t.Fatal("released", e)
			}
			for _, frame := range leerTramas(t, cfg) {
				b, _ := json.Marshal(frame)
				if strings.Contains(string(b), "secret-must-not-be-logged") || strings.Contains(string(b), "/private/not-allowed") {
					t.Fatal("provider content leaked")
				}
				if frame.Registro.Evento != nil && frame.Registro.Evento.Comando.Accion == "abandonar_captura" {
					t.Fatal("false event")
				}
			}
		})
	}
}

func TestAbandonoCASVinculosYNoEntradaAplicarLibre(t *testing.T) {
	cfg := pruebaConfig(t)
	observer := observadorFunc(func(_ context.Context, _ port.Declaracion, s port.SolicitudAbandono) (operacionescopias.ObservacionAbandono, error) {
		return observacionConfirmada(s), nil
	})
	f := abrirObservado(t, cfg, observer)
	iniciar(t, f)
	for _, field := range []string{"version", "solicitud", "destino", "operacion"} {
		p := pedidoAbandono()
		want := operacionescopias.ErrVinculo
		switch field {
		case "version":
			p.VersionEsperada = 0
			want = operacionescopias.ErrVersion
		case "solicitud":
			p.SolicitudSHA256 = strings.Repeat("e", 64)
		case "destino":
			p.Destino = "destino:otro"
		case "operacion":
			p.Operacion = "op:otra"
			want = port.ErrNoExiste
		}
		if _, e := f.AbandonarCaptura(context.Background(), declaracion, p); !errors.Is(e, want) {
			t.Fatal(field, e)
		}
	}
	p := pedidoAbandono()
	a := observacionConfirmada(p)
	c := operacionescopias.Comando{Clave: p.Clave, VersionEsperada: p.VersionEsperada, SolicitudSHA256: p.SolicitudSHA256, Accion: "abandonar_captura", Abandono: &a}
	if _, e := f.Aplicar(context.Background(), declaracion, p.Operacion, c); !errors.Is(e, port.ErrEntrada) {
		t.Fatal("free abort", e)
	}
	if _, e := abrir(t, cfg).AbandonarCaptura(context.Background(), declaracion, p); !errors.Is(e, port.ErrAbandonoNoAutorizado) {
		t.Fatal("no authority", e)
	}
	q, e := f.Consultar(context.Background(), declaracion, p.Operacion)
	if e != nil || q.Recibo.Estado != operacionescopias.Capturando {
		t.Fatal(e)
	}
}

func TestAbandonoConcurrenciaUnaTransicionYRevocacionReplay(t *testing.T) {
	cfg := pruebaConfig(t)
	observer := observadorFunc(func(_ context.Context, _ port.Declaracion, s port.SolicitudAbandono) (operacionescopias.ObservacionAbandono, error) {
		return observacionConfirmada(s), nil
	})
	iniciar(t, abrirObservado(t, cfg, observer))
	var wg sync.WaitGroup
	out := make(chan port.Resultado, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := abrirObservado(t, cfg, observer).AbandonarCaptura(context.Background(), declaracion, pedidoAbandono())
			out <- r
			errs <- e
		}()
	}
	wg.Wait()
	a, b := <-out, <-out
	ea, eb := <-errs, <-errs
	if ea != nil || eb != nil || a.Replay == b.Replay || a.Recibo != b.Recibo || len(a.Historia) != 2 || len(b.Historia) != 2 {
		t.Fatal(a, b, ea, eb)
	}
	denied := observadorFunc(func(context.Context, port.Declaracion, port.SolicitudAbandono) (operacionescopias.ObservacionAbandono, error) {
		return operacionescopias.ObservacionAbandono{}, port.ErrAbandonoNoAutorizado
	})
	if _, e := abrirObservado(t, cfg, denied).AbandonarCaptura(context.Background(), declaracion, pedidoAbandono()); !errors.Is(e, port.ErrAbandonoNoAutorizado) {
		t.Fatal("historical key authorized replay", e)
	}
}

func TestAbandonoCanceladoTrasObservacionNoEscribe(t *testing.T) {
	cfg := pruebaConfig(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	observer := observadorFunc(func(_ context.Context, _ port.Declaracion, s port.SolicitudAbandono) (operacionescopias.ObservacionAbandono, error) {
		cancel()
		return observacionConfirmada(s), nil
	})
	f := abrirObservado(t, cfg, observer)
	before := iniciar(t, f)
	if _, e := f.AbandonarCaptura(ctx, declaracion, pedidoAbandono()); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	if len(leerTramas(t, cfg)) != 2 {
		t.Fatal("cancelled command appended")
	}
	q, e := abrir(t, cfg).Consultar(context.Background(), declaracion, solicitud().Operacion)
	if e != nil || q.Recibo != before.Recibo {
		t.Fatal("cancelled command changed state", e)
	}
}

func TestAbandonoPostCapturaExigeVentanaYVerificadorPropiosInactivos(t *testing.T) {
	for _, phase := range []string{"capturando_publicada", "capturada", "verificando"} {
		for _, mode := range []string{"sin_observador", "verificador_omitido", "verificador_activo", "verificador_incierto", "ventana_omitida", "ventana_activa", "ventana_incierta", "confirmado"} {
			t.Run(phase+"/"+mode, func(t *testing.T) {
				cfg := pruebaConfig(t)
				var provider port.ObservadorAbandono
				if mode != "sin_observador" {
					provider = observadorFunc(func(_ context.Context, _ port.Declaracion, p port.SolicitudAbandono) (operacionescopias.ObservacionAbandono, error) {
						a := observacionConfirmada(p)
						a.EstadoVerificador = "detenido"
						a.EstadoVentana = "inactiva"
						switch mode {
						case "verificador_omitido":
							a.EstadoVerificador = ""
						case "verificador_activo":
							a.EstadoVerificador = "activo"
						case "verificador_incierto":
							a.EstadoVerificador = "incierto"
						case "ventana_omitida":
							a.EstadoVentana = ""
						case "ventana_activa":
							a.EstadoVentana = "activa"
						case "ventana_incierta":
							a.EstadoVentana = "incierta"
						}
						return a, nil
					})
				}
				f := abrirObservado(t, cfg, provider)
				r := iniciar(t, f)
				s := solicitud()
				h := strings.Repeat("b", 64)
				var e error
				if phase != "capturando_publicada" {
					r, e = f.Aplicar(context.Background(), declaracion, s.Operacion, operacionescopias.Comando{Clave: "captura:confirmada", VersionEsperada: 1, SolicitudSHA256: s.SHA256, Accion: "confirmar_captura", ManifiestoSHA256: h})
					if e != nil {
						t.Fatal(e)
					}
				}
				if phase == "verificando" {
					r, e = f.Aplicar(context.Background(), declaracion, s.Operacion, operacionescopias.Comando{Clave: "verificacion:inicio", VersionEsperada: 2, SolicitudSHA256: s.SHA256, Accion: "iniciar_verificacion", ManifiestoSHA256: h, Ejecucion: "ensayo:propio"})
					if e != nil {
						t.Fatal(e)
					}
				}
				before := r
				p := pedidoAbandono()
				p.Clave, p.FalloReferencia, p.VersionEsperada = "abandono:verificacion", "verificacion_fallida", r.Recibo.Version
				abort, e := f.AbandonarCaptura(context.Background(), declaracion, p)
				if mode != "confirmado" {
					if e == nil || abort.Recibo.Referencia != "" {
						t.Fatal("released verification", mode, e)
					}
					q, e := abrir(t, cfg).Consultar(context.Background(), declaracion, s.Operacion)
					if e != nil || q.Recibo != before.Recibo || len(q.Historia) != len(before.Historia) {
						t.Fatal("history changed", e)
					}
					s.Operacion, s.Clave = "op:retry", "reserva:retry"
					if _, e := f.Reservar(context.Background(), declaracion, s); !errors.Is(e, port.ErrDestinoOcupado) {
						t.Fatal("destination released", e)
					}
					return
				}
				if e != nil || abort.Recibo.Estado != operacionescopias.AbandonadaDeclarada || abort.Recibo.Version != before.Recibo.Version+1 {
					t.Fatal("confirmed abort", e)
				}
				last := abort.Historia[len(abort.Historia)-1]
				if last.Comando.Evidencia != nil || last.Comando.ManifiestoSHA256 != "" || last.Comando.Ejecucion != "" || last.Comando.Abandono.EstadoVerificador != "detenido" || last.Comando.Abandono.EstadoVentana != "inactiva" {
					t.Fatal("fabricated evidence")
				}
				q, e := abrir(t, cfg).Consultar(context.Background(), declaracion, s.Operacion)
				if e != nil || q.Recibo != abort.Recibo {
					t.Fatal("reopen", e)
				}
				replay, e := abrirObservado(t, cfg, provider).AbandonarCaptura(context.Background(), declaracion, p)
				if e != nil || !replay.Replay || replay.Recibo != abort.Recibo || len(replay.Historia) != len(abort.Historia) {
					t.Fatal("replay", e)
				}
				s.Operacion, s.Clave, s.Conjunto = "op:retry", "reserva:retry", "conjunto:retry"
				if _, e := f.Reservar(context.Background(), declaracion, s); e != nil {
					t.Fatal("confirmed destination still busy", e)
				}
			})
		}
	}
}

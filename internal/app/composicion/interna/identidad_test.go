package interna

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/gob"
	"encoding/json"
	"errors"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const sanProxyIdentidadOfflinePrueba = "proxy-interno.identidad.test"

type relojIdentidadOfflinePrueba struct{ ahora time.Time }

func (r *relojIdentidadOfflinePrueba) Ahora() time.Time { return r.ahora }

type verificadorIdentidadOfflinePrueba struct {
	asercion httpseguridad.AsercionProxyIdentidad
	error    error
	llamadas atomic.Int32
}

func (v *verificadorIdentidadOfflinePrueba) Verificar(
	_ context.Context,
	_ []byte,
) (httpseguridad.AsercionProxyIdentidad, error) {
	v.llamadas.Add(1)
	if v.error != nil {
		return httpseguridad.AsercionProxyIdentidad{}, v.error
	}
	resultado := v.asercion
	resultado.Factores = append([]httpseguridad.FactorAutenticacion(nil), v.asercion.Factores...)
	return resultado, nil
}

type evaluadorIdentidadOfflinePrueba struct {
	resultado httpseguridad.ResultadoEvaluacionGarantia
	error     error
	llamadas  atomic.Int32
}

func (e *evaluadorIdentidadOfflinePrueba) Evaluar(
	_ context.Context,
	_ httpseguridad.EntradaEvaluacionGarantia,
) (httpseguridad.ResultadoEvaluacionGarantia, error) {
	e.llamadas.Add(1)
	return e.resultado, e.error
}

type registroIdentidadOfflinePrueba struct {
	errorAlta         error
	errorRevalidacion error
	adulterarAlta     func(*httpseguridad.ConfirmacionAltaSesion)
	altas             atomic.Int32
	revalidaciones    atomic.Int32
	confirmacion      httpseguridad.ConfirmacionAltaSesion
}

func (r *registroIdentidadOfflinePrueba) ConsumirAsercionYRegistrar(
	_ context.Context,
	alta httpseguridad.AltaSesionAtomica,
) (httpseguridad.ConfirmacionAltaSesion, error) {
	r.altas.Add(1)
	if r.errorAlta != nil {
		return httpseguridad.ConfirmacionAltaSesion{}, r.errorAlta
	}
	confirmacion := httpseguridad.ConfirmacionAltaSesion{
		AutenticacionRef:          "aut_0123456789abcdefghijkl",
		AsercionRef:               "ase_0123456789abcdefghijkl",
		SesionRef:                 "ses_0123456789abcdefghijkl",
		ControlSesionRef:          "cse_0123456789abcdefghijkl",
		ControlSesionRevision:     1,
		ControlSesionEstado:       httpseguridad.EstadoControlSesionActiva,
		ControlSesionHuellaSHA256: strings.Repeat("c", 64),
		CuentaRef:                 "cta_0123456789abcdefghijkl",
		CuentaOrdinariaRef:        "cta_0123456789abcdefghijkl",
		SesionRevalidadaEn:        alta.SesionEmitidaEn,
		SesionValidaHasta:         alta.AsercionExpiraEn,
		AltaConfirmada:            alta,
	}
	if r.adulterarAlta != nil {
		r.adulterarAlta(&confirmacion)
	}
	r.confirmacion = confirmacion
	return confirmacion, nil
}

func (r *registroIdentidadOfflinePrueba) ComprobarSesionYCuentaActivas(
	_ context.Context,
	consulta httpseguridad.ConsultaSesionActiva,
) error {
	r.revalidaciones.Add(1)
	if r.errorRevalidacion != nil {
		return r.errorRevalidacion
	}
	if consulta.Validar() != nil || r.confirmacion.Validar() != nil ||
		consulta.AutenticacionRef != r.confirmacion.AutenticacionRef ||
		consulta.SesionRef != r.confirmacion.SesionRef ||
		consulta.ControlSesionRef != r.confirmacion.ControlSesionRef {
		return errors.New("revalidacion no coincide")
	}
	return nil
}

type contextoIdentidadOfflineNulo struct{}

func (*contextoIdentidadOfflineNulo) Deadline() (time.Time, bool) { return time.Time{}, false }
func (*contextoIdentidadOfflineNulo) Done() <-chan struct{}       { return nil }
func (*contextoIdentidadOfflineNulo) Err() error                  { return nil }
func (*contextoIdentidadOfflineNulo) Value(any) any               { return nil }

type entornoIdentidadOfflinePrueba struct {
	configuracion httpseguridad.ConfiguracionSuperficie
	intercambio   *intercambioTLSIdentidadOfflinePrueba
	propietario   *ServidorInterno
	servicio      *httpseguridad.ServicioIdentidad
	fachada       *FachadaIdentidadOffline
	verificador   *verificadorIdentidadOfflinePrueba
	evaluador     *evaluadorIdentidadOfflinePrueba
	registro      *registroIdentidadOfflinePrueba
	canal         httpseguridad.CanalProxyAutenticado
	ahora         time.Time
}

type efectosIdentidadOfflinePrueba struct {
	verificaciones, evaluaciones, altas, revalidaciones int32
}

func (e *entornoIdentidadOfflinePrueba) efectos() efectosIdentidadOfflinePrueba {
	return efectosIdentidadOfflinePrueba{e.verificador.llamadas.Load(), e.evaluador.llamadas.Load(), e.registro.altas.Load(), e.registro.revalidaciones.Load()}
}

func (e *entornoIdentidadOfflinePrueba) ejecutarEnC4(
	t *testing.T,
	operacion func(context.Context),
) int {
	t.Helper()
	manejador := nuevoManejadorC4IdentidadOfflinePrueba(
		e.intercambio,
		e.propietario.token,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			operacion(r.Context())
			w.WriteHeader(http.StatusNoContent)
		}),
	)
	return ejecutarManejadorC4IdentidadOfflinePrueba(
		t, manejador, e.intercambio.servidor, &e.intercambio.estadoServidor,
	)
}

func (e *entornoIdentidadOfflinePrueba) autenticarEnC4(
	t *testing.T,
	asercion []byte,
) (httpseguridad.CapsulaIdentidadPeticion, error) {
	t.Helper()
	var capsula httpseguridad.CapsulaIdentidadPeticion
	var err error
	if codigo := e.ejecutarEnC4(t, func(ctx context.Context) {
		capsula, err = e.fachada.Autenticar(ctx, asercion)
	}); codigo != http.StatusNoContent {
		t.Fatalf("C4 no ejecuto identidad: %d", codigo)
	}
	return capsula, err
}

func TestFachadaIdentidadOfflineEmiteCapsulaRevalidadaYLigada(t *testing.T) {
	entorno := nuevoEntornoIdentidadOfflinePrueba(t)
	protegida := []byte("asercion-corporativa-protegida")

	capsula, err := entorno.autenticarEnC4(t, protegida)
	if err != nil {
		t.Fatalf("autenticar: %v", err)
	}
	efectos := entorno.efectos()
	if efectos != (efectosIdentidadOfflinePrueba{1, 2, 1, 1}) {
		t.Fatalf("secuencia incompleta: verificador=%d evaluador=%d altas=%d revalidaciones=%d",
			efectos.verificaciones, efectos.evaluaciones,
			efectos.altas, efectos.revalidaciones)
	}

	otro := nuevoEntornoIdentidadOfflinePrueba(t)
	if _, err = otro.servicio.VincularCapsulaIdentidadPeticion(
		context.Background(), capsula, otro.canal,
	); !errors.Is(err, httpseguridad.ErrSesionNoValida) {
		t.Fatalf("capsula cruzada entre instancias admitida: %v", err)
	}
	otroCanal, err := entorno.servicio.AutenticarCanalTLSMutuo(
		nuevoIntercambioTLSIdentidadOfflinePrueba(t, tls.VersionTLS13).estadoServidor,
	)
	if err != nil {
		t.Fatalf("segundo canal real: %v", err)
	}
	if _, err = entorno.servicio.VincularCapsulaIdentidadPeticion(
		context.Background(), capsula, otroCanal,
	); !errors.Is(err, httpseguridad.ErrSesionNoValida) {
		t.Fatalf("capsula cruzada entre canales admitida: %v", err)
	}

	ctx, err := entorno.servicio.VincularCapsulaIdentidadPeticion(
		context.Background(), capsula, entorno.canal,
	)
	if err != nil {
		t.Fatalf("vincular capsula: %v", err)
	}
	cuenta, auditoria, err := entorno.servicio.ExtraerCapsulaIdentidadPeticion(ctx)
	if err != nil || cuenta.Validar() != nil ||
		cuenta.Garantia != dominiovec.AuthAssuranceHigh ||
		auditoria.CanalVinculadoRef() != entorno.canal.ReferenciaVinculacion() {
		t.Fatalf("extraer identidad ligada: cuenta=%+v auditoria=%v error=%v", cuenta, auditoria, err)
	}
}

func TestFachadaIdentidadOfflineRechazaFronterasAdversas(t *testing.T) {
	casos := []struct {
		nombre               string
		mutar                func(*entornoIdentidadOfflinePrueba)
		altasEsperadas       int
		revalidacionesEspera int
	}{
		{"emisor", func(e *entornoIdentidadOfflinePrueba) { e.verificador.asercion.Emisor = "https://emisor.ajeno.test" }, 0, 0},
		{"audiencia", func(e *entornoIdentidadOfflinePrueba) { e.verificador.asercion.Audiencia = "vec-ajena" }, 0, 0},
		{"superficie", func(e *entornoIdentidadOfflinePrueba) {
			e.verificador.asercion.Superficie = httpseguridad.SuperficieExternaPersonal
		}, 0, 0},
		{"canal", func(e *entornoIdentidadOfflinePrueba) {
			e.verificador.asercion.CanalVinculadoRef = "tls-exportador:sha256:" + strings.Repeat("d", 64)
		}, 0, 0},
		{"factor", func(e *entornoIdentidadOfflinePrueba) {
			e.verificador.asercion.Factores = e.verificador.asercion.Factores[:1]
		}, 0, 0},
		{"grupo criptografico", func(e *entornoIdentidadOfflinePrueba) {
			e.verificador.asercion.Factores[1].GrupoCriptograficoRef = e.verificador.asercion.Factores[0].GrupoCriptograficoRef
		}, 0, 0},
		{"garantia", func(e *entornoIdentidadOfflinePrueba) {
			e.evaluador.resultado.Garantia = dominiovec.AuthAssuranceSubstantial
		}, 0, 0},
		{"alta durable", func(e *entornoIdentidadOfflinePrueba) { e.registro.errorAlta = errors.New("alta rechazada") }, 1, 0},
		{"confirmacion adulterada", func(e *entornoIdentidadOfflinePrueba) {
			e.registro.adulterarAlta = func(c *httpseguridad.ConfirmacionAltaSesion) { c.ControlSesionRevision = 2 }
		}, 1, 0},
		{"revalidacion", func(e *entornoIdentidadOfflinePrueba) { e.registro.errorRevalidacion = errors.New("sesion revocada") }, 1, 1},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			entorno := nuevoEntornoIdentidadOfflinePrueba(t)
			caso.mutar(entorno)
			capsula, err := entorno.autenticarEnC4(t, []byte("asercion-adversa"))
			if err == nil {
				t.Fatal("frontera adversa aceptada")
			}
			if entorno.registro.altas.Load() != int32(caso.altasEsperadas) ||
				entorno.registro.revalidaciones.Load() != int32(caso.revalidacionesEspera) {
				t.Fatalf("efectos inesperados: altas=%d revalidaciones=%d",
					entorno.registro.altas.Load(), entorno.registro.revalidaciones.Load())
			}
			if _, errVinculo := entorno.servicio.VincularCapsulaIdentidadPeticion(
				context.Background(), capsula, entorno.canal,
			); errVinculo == nil {
				t.Fatal("el rechazo devolvio una capsula utilizable")
			}
		})
	}
}

func TestFachadaIdentidadOfflineCierraNulosTLSYSuperficieAjena(t *testing.T) {
	var servicioNulo *httpseguridad.ServicioIdentidad
	if _, err := NuevaFachadaIdentidadOffline(nil, nil); !errors.Is(err, ErrIdentidadOfflineNoDisponible) {
		t.Fatalf("servicio nulo admitido: %v", err)
	}
	if _, err := NuevaFachadaIdentidadOffline(servicioNulo, nil); !errors.Is(err, ErrIdentidadOfflineNoDisponible) {
		t.Fatalf("servicio tipado nulo admitido: %v", err)
	}
	var fachadaNula *FachadaIdentidadOffline
	if _, err := fachadaNula.Autenticar(context.Background(), []byte("x")); !errors.Is(err, ErrIdentidadOfflineNoDisponible) {
		t.Fatalf("fachada nula admitida: %v", err)
	}
	entorno := nuevoEntornoIdentidadOfflinePrueba(t)
	var servidorNulo *ServidorInterno
	if _, err := NuevaFachadaIdentidadOffline(
		entorno.servicio, servidorNulo,
	); !errors.Is(err, ErrIdentidadOfflineNoDisponible) {
		t.Fatalf("propietario C4 tipado nulo admitido: %v", err)
	}
	servidorSinPropietario := &ServidorInterno{token: &tokenServidorInterno{marca: 1}}
	if _, err := NuevaFachadaIdentidadOffline(
		entorno.servicio, servidorSinPropietario,
	); !errors.Is(err, ErrIdentidadOfflineNoDisponible) {
		t.Fatalf("propietario C4 ajeno admitido: %v", err)
	}
	if _, err := entorno.fachada.Autenticar(nil, []byte("x")); !errors.Is(err, ErrIdentidadOfflineNoDisponible) {
		t.Fatalf("contexto nulo admitido: %v", err)
	}
	var contextoNulo *contextoIdentidadOfflineNulo
	if _, err := entorno.fachada.Autenticar(contextoNulo, []byte("x")); !errors.Is(err, ErrIdentidadOfflineNoDisponible) {
		t.Fatalf("contexto tipado nulo admitido: %v", err)
	}
	if _, err := entorno.fachada.Autenticar(context.Background(), []byte("x")); !errors.Is(err, httpseguridad.ErrCanalProxyNoAutenticado) {
		t.Fatalf("contexto sin capacidad C4 admitido: %v", err)
	}
	var capacidadNula *capacidadCanalTLSInterno
	ctxCapacidadNula := context.WithValue(
		context.Background(), claveContextoCanalTLSInterno{}, capacidadNula,
	)
	if _, err := entorno.fachada.Autenticar(ctxCapacidadNula, []byte("x")); !errors.Is(err, httpseguridad.ErrCanalProxyNoAutenticado) {
		t.Fatalf("capacidad C4 tipada nula admitida: %v", err)
	}
	if efectos := entorno.efectos(); efectos != (efectosIdentidadOfflinePrueba{}) {
		t.Fatalf("typed nil produjo efectos: %+v", efectos)
	}
	var errCancelacion, errValido, errTercero error
	var efectosCancelacion efectosIdentidadOfflinePrueba
	codigo := entorno.ejecutarEnC4(t, func(ctx context.Context) {
		ctxCancelado, cancelar := context.WithCancel(ctx)
		cancelar()
		_, errCancelacion = entorno.fachada.Autenticar(ctxCancelado, []byte("x"))
		efectosCancelacion = entorno.efectos()
		_, errValido = entorno.fachada.Autenticar(ctx, []byte("asercion-tras-cancelacion"))
		_, errTercero = entorno.fachada.Autenticar(ctx, []byte("asercion-tercera"))
	})
	if codigo != http.StatusNoContent || !errors.Is(errCancelacion, context.Canceled) || errValido != nil ||
		!errors.Is(errTercero, httpseguridad.ErrCanalProxyNoAutenticado) {
		t.Fatalf("cancelacion/consumo: codigo=%d cancelada=%v valida=%v tercera=%v",
			codigo, errCancelacion, errValido, errTercero)
	}
	if efectosCancelacion != (efectosIdentidadOfflinePrueba{}) ||
		entorno.efectos() != (efectosIdentidadOfflinePrueba{1, 2, 1, 1}) {
		t.Fatalf("ledger cancelacion/consumo: cancelada=%+v final=%+v",
			efectosCancelacion, entorno.efectos())
	}
	var protegidaNula []byte
	if _, err := entorno.autenticarEnC4(t, protegidaNula); !errors.Is(err, httpseguridad.ErrAsercionAusente) {
		t.Fatalf("asercion tipada nula admitida: %v", err)
	}
	configuracionExterna := entorno.configuracion
	configuracionExterna.Superficie = httpseguridad.SuperficieExternaPersonal
	configuracionExterna.ZonaRed = httpseguridad.ZonaRedPublica
	configuracionExterna.DireccionEscucha = "127.0.0.1:9443"
	configuracionExterna.RedesPermitidas = []string{"127.0.0.0/8"}
	configuracionExterna.MetodosAdmitidos = []httpseguridad.MetodoAutenticacion{httpseguridad.MetodoCertificado}
	configuracionExterna.FactoresRequeridos = nil
	configuracionExterna.MinimoFactoresVerificados = 1
	configuracionExterna.MinimoGruposCriptograficosDistintos = 1
	configuracionExterna.GarantiaMinima = dominiovec.AuthAssuranceSubstantial
	servicioExterno := debeServicioIdentidadOfflinePrueba(
		t, configuracionExterna, &verificadorIdentidadOfflinePrueba{},
		&evaluadorIdentidadOfflinePrueba{}, &registroIdentidadOfflinePrueba{}, entorno.ahora,
	)
	fachadaExterna, err := NuevaFachadaIdentidadOffline(servicioExterno, entorno.propietario)
	if err != nil {
		t.Fatalf("crear fachada externa para rechazo: %v", err)
	}
	entorno.ejecutarEnC4(t, func(ctx context.Context) {
		_, err = fachadaExterna.Autenticar(ctx, []byte("x"))
	})
	if !errors.Is(err, httpseguridad.ErrCanalProxyNoAutenticado) {
		t.Fatalf("superficie externa admitida: %v", err)
	}
}

func TestFachadaIdentidadOfflineRechazaEstadosNoAcreditadosSinEfectos(t *testing.T) {
	probarRechazo := func(
		t *testing.T,
		intercambio *intercambioTLSIdentidadOfflinePrueba,
		estado tls.ConnectionState,
	) {
		t.Helper()
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		manejador := nuevoManejadorC4IdentidadOfflinePrueba(
			intercambio,
			entorno.propietario.token,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = entorno.fachada.Autenticar(r.Context(), []byte("asercion-no-acreditada"))
				w.WriteHeader(http.StatusNoContent)
			}),
		)
		if codigo := ejecutarManejadorC4IdentidadOfflinePrueba(
			t, manejador, intercambio.servidor, &estado,
		); codigo != http.StatusBadRequest {
			t.Fatalf("estado no acreditado alcanzo C5: %d", codigo)
		}
		efectos := entorno.efectos()
		if efectos != (efectosIdentidadOfflinePrueba{}) {
			t.Fatalf(
				"estado no acreditado produjo efectos: verificador=%d evaluador=%d altas=%d revalidaciones=%d",
				efectos.verificaciones, efectos.evaluaciones,
				efectos.altas, efectos.revalidaciones,
			)
		}
	}

	t.Run("estado del lado cliente", func(t *testing.T) {
		intercambio := nuevoIntercambioTLSIdentidadOfflinePrueba(t, tls.VersionTLS13)
		probarRechazo(t, intercambio, intercambio.estadoCliente)
	})
	t.Run("exporter trasplantado entre handshakes", func(t *testing.T) {
		origen := nuevoIntercambioTLSIdentidadOfflinePrueba(t, tls.VersionTLS13)
		destino := nuevoIntercambioTLSIdentidadOfflinePrueba(t, tls.VersionTLS13)
		vinculoOrigen, errOrigen := origen.estadoServidor.ExportKeyingMaterial(
			etiquetaExportadorConexion, nil, 32,
		)
		vinculoDestino, errDestino := destino.estadoServidor.ExportKeyingMaterial(
			etiquetaExportadorConexion, nil, 32,
		)
		if errOrigen != nil || errDestino != nil || bytes.Equal(vinculoOrigen, vinculoDestino) {
			t.Fatalf("handshakes sin exporters distintos: (%v, %v)", errOrigen, errDestino)
		}
		hibrido := origen.estadoServidor
		hibrido.Version = destino.estadoServidor.Version
		hibrido.HandshakeComplete = destino.estadoServidor.HandshakeComplete
		hibrido.DidResume = destino.estadoServidor.DidResume
		hibrido.CipherSuite = destino.estadoServidor.CipherSuite
		hibrido.CurveID = destino.estadoServidor.CurveID
		hibrido.NegotiatedProtocol = destino.estadoServidor.NegotiatedProtocol
		hibrido.NegotiatedProtocolIsMutual = destino.estadoServidor.NegotiatedProtocolIsMutual
		hibrido.ServerName = destino.estadoServidor.ServerName
		hibrido.PeerCertificates = destino.estadoServidor.PeerCertificates
		hibrido.VerifiedChains = destino.estadoServidor.VerifiedChains
		hibrido.SignedCertificateTimestamps = destino.estadoServidor.SignedCertificateTimestamps
		hibrido.OCSPResponse = destino.estadoServidor.OCSPResponse
		hibrido.TLSUnique = destino.estadoServidor.TLSUnique
		hibrido.ECHAccepted = destino.estadoServidor.ECHAccepted
		probarRechazo(t, destino, hibrido)
	})
	t.Run("certificados trasplantados conservando exporter del destino", func(t *testing.T) {
		origen := nuevoIntercambioTLSIdentidadOfflinePrueba(t, tls.VersionTLS13)
		destino := nuevoIntercambioTLSIdentidadOfflinePrueba(t, tls.VersionTLS13)
		hibrido := destino.estadoServidor
		hibrido.PeerCertificates = origen.estadoServidor.PeerCertificates
		hibrido.VerifiedChains = origen.estadoServidor.VerifiedChains
		if hibrido.PeerCertificates[0].Equal(destino.estadoServidor.PeerCertificates[0]) {
			t.Fatal("handshakes sin certificados distintos")
		}
		probarRechazo(t, destino, hibrido)
	})
	t.Run("TLS 1.2 del lado servidor", func(t *testing.T) {
		intercambio := nuevoIntercambioTLSIdentidadOfflinePrueba(t, tls.VersionTLS12)
		probarRechazo(t, intercambio, intercambio.estadoServidor)
	})
}

func TestCapacidadC4IdentidadOfflineEsEfimeraYDeConsumoUnico(t *testing.T) {
	t.Run("segundo uso secuencial", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		var primerError, segundoError error
		var efectosTrasPrimero efectosIdentidadOfflinePrueba
		codigo := entorno.ejecutarEnC4(t, func(ctx context.Context) {
			_, primerError = entorno.fachada.Autenticar(ctx, []byte("asercion-primera"))
			efectosTrasPrimero = entorno.efectos()
			_, segundoError = entorno.fachada.Autenticar(ctx, []byte("asercion-segunda"))
		})
		if codigo != http.StatusNoContent || primerError != nil ||
			!errors.Is(segundoError, httpseguridad.ErrCanalProxyNoAutenticado) {
			t.Fatalf("replay secuencial: codigo=%d primero=%v segundo=%v",
				codigo, primerError, segundoError)
		}
		if efectosTrasPrimero != (efectosIdentidadOfflinePrueba{1, 2, 1, 1}) ||
			entorno.efectos() != efectosTrasPrimero {
			t.Fatalf("segundo uso produjo efectos: primero=%+v final=%+v",
				efectosTrasPrimero, entorno.efectos())
		}
	})

	t.Run("dos usos concurrentes con exactamente un ganador", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		errores := make([]error, 2)
		codigo := entorno.ejecutarEnC4(t, func(ctx context.Context) {
			inicio := make(chan struct{})
			var grupo sync.WaitGroup
			grupo.Add(len(errores))
			for indice := range errores {
				go func() {
					defer grupo.Done()
					<-inicio
					_, errores[indice] = entorno.fachada.Autenticar(
						ctx, []byte("asercion-concurrente"),
					)
				}()
			}
			close(inicio)
			grupo.Wait()
		})
		ganadores, rechazados := 0, 0
		for _, err := range errores {
			switch {
			case err == nil:
				ganadores++
			case errors.Is(err, httpseguridad.ErrCanalProxyNoAutenticado):
				rechazados++
			default:
				t.Fatalf("resultado concurrente inesperado: %v", err)
			}
		}
		if codigo != http.StatusNoContent || ganadores != 1 || rechazados != 1 ||
			entorno.efectos() != (efectosIdentidadOfflinePrueba{1, 2, 1, 1}) {
			t.Fatalf("consumo concurrente: codigo=%d ganadores=%d rechazos=%d efectos=%+v",
				codigo, ganadores, rechazados, entorno.efectos())
		}
	})

	t.Run("contexto capturado tras retorno", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		var capturado context.Context
		if codigo := entorno.ejecutarEnC4(t, func(ctx context.Context) {
			capturado = ctx
		}); codigo != http.StatusNoContent || capturado == nil {
			t.Fatalf("captura del contexto: codigo=%d contexto=%v", codigo, capturado)
		}
		_, err := entorno.fachada.Autenticar(capturado, []byte("asercion-tardia"))
		if !errors.Is(err, httpseguridad.ErrCanalProxyNoAutenticado) ||
			entorno.efectos() != (efectosIdentidadOfflinePrueba{}) {
			t.Fatalf("contexto capturado admitido: error=%v efectos=%+v", err, entorno.efectos())
		}
	})

	t.Run("trasplante entre propietarios", func(t *testing.T) {
		origen := nuevoEntornoIdentidadOfflinePrueba(t)
		destino := nuevoEntornoIdentidadOfflinePrueba(t)
		var errTrasplante error
		codigo := origen.ejecutarEnC4(t, func(ctx context.Context) {
			_, errTrasplante = destino.fachada.Autenticar(ctx, []byte("asercion-trasplantada"))
		})
		if codigo != http.StatusNoContent ||
			!errors.Is(errTrasplante, httpseguridad.ErrCanalProxyNoAutenticado) ||
			destino.efectos() != (efectosIdentidadOfflinePrueba{}) {
			t.Fatalf("trasplante admitido: codigo=%d error=%v efectos=%+v",
				codigo, errTrasplante, destino.efectos())
		}
	})
}

func TestCapsulaEmitidaPorFachadaIdentidadOfflineNoSeSerializa(t *testing.T) {
	entorno := nuevoEntornoIdentidadOfflinePrueba(t)
	capsula, err := entorno.autenticarEnC4(t, []byte("material-que-no-debe-serializarse"))
	if err != nil {
		t.Fatalf("autenticar: %v", err)
	}
	if _, err = json.Marshal(capsula); !errors.Is(err, httpseguridad.ErrIdentidadNoSerializable) {
		t.Fatalf("JSON admitido: %v", err)
	}
	if _, err = capsula.MarshalText(); !errors.Is(err, httpseguridad.ErrIdentidadNoSerializable) {
		t.Fatalf("texto admitido: %v", err)
	}
	if _, err = capsula.MarshalBinary(); !errors.Is(err, httpseguridad.ErrIdentidadNoSerializable) {
		t.Fatalf("binario admitido: %v", err)
	}
	var salida bytes.Buffer
	if err = gob.NewEncoder(&salida).Encode(capsula); !errors.Is(err, httpseguridad.ErrIdentidadNoSerializable) {
		t.Fatalf("gob admitido: %v", err)
	}
}

func nuevoEntornoIdentidadOfflinePrueba(
	t *testing.T,
) *entornoIdentidadOfflinePrueba {
	t.Helper()
	intercambio := nuevoIntercambioTLSIdentidadOfflinePrueba(t, tls.VersionTLS13)
	estadoTLS := intercambio.estadoServidor
	token := &tokenServidorInterno{marca: 1}
	propietario := &ServidorInterno{token: token}
	propietario.propietario = propietario
	ahora := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	configuracion := configuracionIdentidadOfflinePrueba()
	verificador := &verificadorIdentidadOfflinePrueba{}
	evaluador := &evaluadorIdentidadOfflinePrueba{resultado: httpseguridad.ResultadoEvaluacionGarantia{
		Garantia:       dominiovec.AuthAssuranceHigh,
		PoliticaRef:    "pga_0123456789abcdefghijkl",
		HuellaPolitica: "sha256:" + strings.Repeat("a", 64),
	}}
	registro := &registroIdentidadOfflinePrueba{}
	servicio := debeServicioIdentidadOfflinePrueba(
		t, configuracion, verificador, evaluador, registro, ahora,
	)
	canal, err := servicio.AutenticarCanalTLSMutuo(estadoTLS)
	if err != nil {
		t.Fatalf("autenticar canal de prueba: %v", err)
	}
	verificador.asercion = asercionIdentidadOfflinePrueba(ahora, configuracion, canal)
	fachada, err := NuevaFachadaIdentidadOffline(servicio, propietario)
	if err != nil {
		t.Fatalf("crear fachada: %v", err)
	}
	return &entornoIdentidadOfflinePrueba{
		configuracion: configuracion, intercambio: intercambio, propietario: propietario,
		servicio: servicio, fachada: fachada, verificador: verificador, evaluador: evaluador,
		registro: registro, canal: canal, ahora: ahora,
	}
}

func debeServicioIdentidadOfflinePrueba(
	t *testing.T,
	configuracion httpseguridad.ConfiguracionSuperficie,
	verificador httpseguridad.VerificadorAsercionProtegida,
	evaluador httpseguridad.EvaluadorGarantia,
	registro httpseguridad.RegistroSesiones,
	ahora time.Time,
) *httpseguridad.ServicioIdentidad {
	t.Helper()
	servicio, err := httpseguridad.NuevoServicioIdentidad(
		configuracion, verificador, evaluador, registro,
		&relojIdentidadOfflinePrueba{ahora: ahora},
	)
	if err != nil {
		t.Fatalf("crear servicio de identidad: %v", err)
	}
	return servicio
}

func configuracionIdentidadOfflinePrueba() httpseguridad.ConfiguracionSuperficie {
	return httpseguridad.ConfiguracionSuperficie{
		Superficie:                          httpseguridad.SuperficieInternaCorporativa,
		ZonaRed:                             httpseguridad.ZonaRedInterna,
		DireccionEscucha:                    "127.0.0.1:8443",
		Audiencia:                           "vec-interna",
		EmisorIdentidad:                     "https://idp.identidad.test",
		RedesPermitidas:                     []string{"127.0.0.0/8"},
		IdentidadesSANProxyPermitidas:       []string{"dns:" + sanProxyIdentidadOfflinePrueba},
		DuracionMaximaAsercion:              3 * time.Minute,
		EdadMaximaAutenticacion:             15 * time.Minute,
		ToleranciaReloj:                     20 * time.Second,
		MetodosAdmitidos:                    []httpseguridad.MetodoAutenticacion{httpseguridad.MetodoKerberos, httpseguridad.MetodoCertificado},
		FactoresRequeridos:                  []httpseguridad.MetodoAutenticacion{httpseguridad.MetodoKerberos, httpseguridad.MetodoCertificado},
		MinimoFactoresVerificados:           2,
		MinimoGruposCriptograficosDistintos: 2,
		GarantiaMinima:                      dominiovec.AuthAssuranceHigh,
	}
}

func asercionIdentidadOfflinePrueba(
	ahora time.Time,
	configuracion httpseguridad.ConfiguracionSuperficie,
	canal httpseguridad.CanalProxyAutenticado,
) httpseguridad.AsercionProxyIdentidad {
	emitida := ahora.Add(-time.Minute)
	return httpseguridad.AsercionProxyIdentidad{
		ID: "asercion-interna-001", Emisor: configuracion.EmisorIdentidad,
		Audiencia: configuracion.Audiencia, Superficie: configuracion.Superficie,
		SujetoID: "persona-interna-001",
		Cuenta: httpseguridad.CuentaAcceso{
			ID: "cuenta-interna-001", SujetoVinculadoID: "persona-interna-001",
		},
		SesionID: "sesion-interna-001", CanalVinculadoRef: canal.ReferenciaVinculacion(),
		AutenticacionVerificadaEn: emitida, EmitidaEn: emitida, NoAntesDe: emitida,
		ExpiraEn: ahora.Add(time.Minute), MetodoPrimario: httpseguridad.MetodoCertificado,
		ACRVerificado: "urn:vec:acr:alto",
		Factores: []httpseguridad.FactorAutenticacion{
			{
				Metodo: httpseguridad.MetodoKerberos, SujetoVinculadoID: "persona-interna-001",
				Principal: "cuenta-interna@IDENTIDAD.TEST", EvidenciaRef: "krb:ticket:001",
				GrupoCriptograficoRef: "grupo:kerberos:001", VerificadoEn: emitida,
			},
			{
				Metodo: httpseguridad.MetodoCertificado, SujetoVinculadoID: "persona-interna-001",
				CredencialRef: "cert:sha256:001", EvidenciaRef: "cert:validacion:001",
				GrupoCriptograficoRef: "grupo:certificado:001", VerificadoEn: emitida,
			},
		},
	}
}

type intercambioTLSIdentidadOfflinePrueba struct {
	servidor       *tls.Conn
	estadoServidor tls.ConnectionState
	estadoCliente  tls.ConnectionState
	raicesClientes *x509.CertPool
	nombreServidor string
}

func nuevoManejadorC4IdentidadOfflinePrueba(
	intercambio *intercambioTLSIdentidadOfflinePrueba,
	token *tokenServidorInterno,
	siguiente http.Handler,
) *manejadorInternoVerificado {
	return &manejadorInternoVerificado{
		siguiente: siguiente,
		token:     token,
		materialTLS: materialTLSAprobado{
			autoridadesClientes: intercambio.raicesClientes,
			nombreServidor:      intercambio.nombreServidor,
		},
	}
}

func ejecutarManejadorC4IdentidadOfflinePrueba(
	t *testing.T,
	manejador *manejadorInternoVerificado,
	conexion *tls.Conn,
	estado *tls.ConnectionState,
) int {
	t.Helper()
	peticion := httptest.NewRequest(http.MethodPost, "/api/vec/identidad", nil)
	peticion.TLS = estado
	peticion = peticion.WithContext(context.WithValue(
		peticion.Context(),
		claveContextoConexionTLS{},
		&posesionConexionTLS{token: manejador.token, conexion: conexion},
	))
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	return respuesta.Code
}

func nuevoIntercambioTLSIdentidadOfflinePrueba(
	t *testing.T,
	version uint16,
) *intercambioTLSIdentidadOfflinePrueba {
	t.Helper()
	ahora := time.Now()
	_, claveCA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generar CA: %v", err)
	}
	plantillaCA := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "CA identidad offline"},
		NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), IsCA: true,
		BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	derCA, err := x509.CreateCertificate(rand.Reader, plantillaCA, plantillaCA, claveCA.Public(), claveCA)
	if err != nil {
		t.Fatalf("crear CA: %v", err)
	}
	certificadoCA, err := x509.ParseCertificate(derCA)
	if err != nil {
		t.Fatalf("parsear CA: %v", err)
	}
	crearCertificado := func(serial int64, nombre string, uso x509.ExtKeyUsage) tls.Certificate {
		publica, privada, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generar clave: %v", err)
		}
		plantilla := &x509.Certificate{
			SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: nombre},
			DNSNames: []string{nombre}, NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour),
			KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{uso},
		}
		der, err := x509.CreateCertificate(rand.Reader, plantilla, certificadoCA, publica, claveCA)
		if err != nil {
			t.Fatalf("crear certificado: %v", err)
		}
		return tls.Certificate{Certificate: [][]byte{der, derCA}, PrivateKey: privada}
	}
	raices := x509.NewCertPool()
	raices.AddCert(certificadoCA)
	servidorConfig := &tls.Config{
		Certificates: []tls.Certificate{crearCertificado(2, "servidor.identidad.test", x509.ExtKeyUsageServerAuth)},
		ClientAuth:   tls.RequireAndVerifyClientCert, ClientCAs: raices,
		MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13,
	}
	clienteConfig := &tls.Config{
		RootCAs: raices, ServerName: "servidor.identidad.test",
		Certificates: []tls.Certificate{
			crearCertificado(3, sanProxyIdentidadOfflinePrueba, x509.ExtKeyUsageClientAuth),
		},
		MinVersion: version, MaxVersion: version,
		NextProtos: []string{protocoloALPNHTTPUno},
	}
	servidorConfig.MinVersion = version
	servidorConfig.MaxVersion = version
	servidorConfig.NextProtos = []string{protocoloALPNHTTPUno}
	parServidor, parCliente := net.Pipe()
	servidor := tls.Server(parServidor, servidorConfig)
	cliente := tls.Client(parCliente, clienteConfig)
	errores := make(chan error, 2)
	go func() { errores <- servidor.Handshake() }()
	go func() { errores <- cliente.Handshake() }()
	for range 2 {
		if err := <-errores; err != nil {
			_ = parServidor.Close()
			_ = parCliente.Close()
			t.Fatalf("handshake mTLS: %v", err)
		}
	}
	t.Cleanup(func() {
		_ = parServidor.Close()
		_ = parCliente.Close()
	})
	return &intercambioTLSIdentidadOfflinePrueba{
		servidor:       servidor,
		estadoServidor: servidor.ConnectionState(),
		estadoCliente:  cliente.ConnectionState(),
		raicesClientes: raices, nombreServidor: "servidor.identidad.test",
	}
}

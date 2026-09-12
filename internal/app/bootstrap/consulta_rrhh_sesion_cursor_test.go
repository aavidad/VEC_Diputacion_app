package bootstrap

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"errors"
	"math/big"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type consultorCursorRRHHFallaPrueba struct{ err error }

func (c consultorCursorRRHHFallaPrueba) Consultar(context.Context, ports.SolicitudCuadroRRHH) (ports.PaginaCuadroRRHH, error) {
	return ports.PaginaCuadroRRHH{}, c.err
}

func cursorSesionRRHHDesarrolloPrueba(relleno byte) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat(string([]byte{relleno}), 32)))
}

func vinculoCanalCursorRRHHDesarrolloPrueba(relleno byte) [32]byte {
	var vinculo [32]byte
	for indice := range vinculo {
		vinculo[indice] = relleno
	}
	return vinculo
}

type conexionesTLSCursorRRHHReales struct {
	nuevaConexion      func() tls.ConnectionState
	certificadoCliente *x509.Certificate
}

func nuevaConexionTLSCursorRRHHReal(t *testing.T) conexionesTLSCursorRRHHReales {
	t.Helper()
	ahora := time.Now()
	_, claveCA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	plantillaCA := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "CA cursor"},
		NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	derCA, err := x509.CreateCertificate(rand.Reader, plantillaCA, plantillaCA, claveCA.Public(), claveCA)
	if err != nil {
		t.Fatal(err)
	}
	ca, err := x509.ParseCertificate(derCA)
	if err != nil {
		t.Fatal(err)
	}
	certificado := func(serial int64, nombre string, uso x509.ExtKeyUsage) tls.Certificate {
		publica, privada, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		plantilla := &x509.Certificate{SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: nombre}, DNSNames: []string{nombre},
			NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{uso}}
		der, err := x509.CreateCertificate(rand.Reader, plantilla, ca, publica, claveCA)
		if err != nil {
			t.Fatal(err)
		}
		return tls.Certificate{Certificate: [][]byte{der, derCA}, PrivateKey: privada}
	}
	servidorCert, clienteCert := certificado(2, "servidor.cursor", x509.ExtKeyUsageServerAuth), certificado(3, "cliente.cursor", x509.ExtKeyUsageClientAuth)
	hojaCliente, err := x509.ParseCertificate(clienteCert.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	raices := x509.NewCertPool()
	raices.AddCert(ca)
	return conexionesTLSCursorRRHHReales{certificadoCliente: hojaCliente, nuevaConexion: func() tls.ConnectionState {
		servidorRed, clienteRed := net.Pipe()
		servidor := tls.Server(servidorRed, &tls.Config{Certificates: []tls.Certificate{servidorCert}, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: raices, MinVersion: tls.VersionTLS13})
		cliente := tls.Client(clienteRed, &tls.Config{Certificates: []tls.Certificate{clienteCert}, RootCAs: raices, ServerName: "servidor.cursor", MinVersion: tls.VersionTLS13})
		errores := make(chan error, 2)
		go func() { errores <- servidor.Handshake() }()
		go func() { errores <- cliente.Handshake() }()
		for indice := 0; indice < 2; indice++ {
			if err := <-errores; err != nil {
				t.Fatal(err)
			}
		}
		estado := servidor.ConnectionState()
		_ = servidor.Close()
		_ = cliente.Close()
		return estado
	}}
}

func TestVinculoCanalTLSCursorRRHHExigeConexionRealYDistingueConexiones(t *testing.T) {
	if _, ok := vinculoCanalTLSCursorRRHHDesarrollo(&tls.ConnectionState{HandshakeComplete: true, Version: tls.VersionTLS13}); ok {
		t.Fatal("un estado TLS fabricado produjo vínculo de conexión")
	}
	conexiones := nuevaConexionTLSCursorRRHHReal(t)
	primera, okPrimera := vinculoCanalTLSCursorRRHHDesarrollo(ptrEstadoTLSCursorRRHH(conexiones.nuevaConexion()))
	segunda, okSegunda := vinculoCanalTLSCursorRRHHDesarrollo(ptrEstadoTLSCursorRRHH(conexiones.nuevaConexion()))
	if !okPrimera || !okSegunda || primera == segunda {
		t.Fatal("dos conexiones reales con el mismo certificado no quedaron separadas")
	}
}

func ptrEstadoTLSCursorRRHH(estado tls.ConnectionState) *tls.ConnectionState { return &estado }

func prepararContinuidadCursorRRHHDesarrolloPrueba(t *testing.T) (
	*entornoSesionConsultaPrueba, *autoridadConsultasRRHHDesarrollo,
	*continuadorSesionCursorRRHHDesarrollo, context.Context, ports.ContextoAutorizacionAltaV3,
) {
	t.Helper()
	e := nuevaSesionConsultaPrueba(t)
	a := &autoridadConsultasRRHHDesarrollo{soporte: e.soporte, reloj: e.reloj, proveedor: e.p,
		clase: ports.AmbitoOrganizacionRRHH, ambitoRef: organizacionAltaContratacionTemporalDesarrollo}
	continuador, err := nuevoContinuadorSesionCursorRRHHDesarrollo(a, e.reloj)
	if err != nil {
		t.Fatal(err)
	}
	ctx := e.contexto()
	primero, err := e.p.ResolverContexto(ctx)
	if err != nil {
		t.Fatal(err)
	}
	capacidad := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	capacidad.consultaRRHH = &contextoConsultaRRHHPeticionDesarrollo{autoridad: a, contexto: primero}
	capacidad.vinculoCanalTLS = vinculoCanalCursorRRHHDesarrolloPrueba('1')
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	return e, a, continuador, ctx, primero
}

func segundaPeticionCursorRRHHDesarrolloPrueba(e *entornoSesionConsultaPrueba, vinculo [32]byte) context.Context {
	ctx := e.contexto()
	capacidad := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	capacidad.consultaRRHH = &contextoConsultaRRHHPeticionDesarrollo{}
	capacidad.vinculoCanalTLS = vinculo
	return context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
}

func TestContinuidadCursorRRHHConservaSesionYRevalidaEnSegundaPeticion(t *testing.T) {
	e, a, continuador, primeraPeticion, primero := prepararContinuidadCursorRRHHDesarrolloPrueba(t)
	cursor := cursorSesionRRHHDesarrolloPrueba('a')
	if err := continuador.recordar(primeraPeticion, ports.PaginaCuadroRRHH{HayMas: true, CursorSiguiente: cursor}); err != nil {
		t.Fatal(err)
	}
	segundaPeticion := segundaPeticionCursorRRHHDesarrolloPrueba(e, vinculoCanalCursorRRHHDesarrolloPrueba('1'))
	preparada, err := continuador.preparar(segundaPeticion, cursor)
	if err != nil {
		t.Fatal(err)
	}
	obtenido, err := a.contextoConsultaRRHHDesarrollo(preparada)
	if err != nil {
		t.Fatal(err)
	}
	datosPrimero, _ := primero.Vinculo.Datos()
	datosSegundo, _ := obtenido.Vinculo.Datos()
	if datosSegundo.SesionRef != datosPrimero.SesionRef || datosSegundo.AutenticacionRef != datosPrimero.AutenticacionRef ||
		e.revalidador.llamadas != 2 || e.resolutor.llamadas != 2 || len(e.registro.altas) != 1 {
		t.Fatal("la segunda petición no revalidó la sesión original o abrió otra")
	}
	if _, err := continuador.preparar(e.contexto(), cursor); !errors.Is(err, ports.ErrAutorizacionDenegada) {
		t.Fatal("un cursor reservado volvió a admitirse")
	}
}

func TestContinuidadCursorRRHHRechazaRevocacionCanalYCambioSesion(t *testing.T) {
	t.Run("revocacion", func(t *testing.T) {
		e, a, continuador, primera, _ := prepararContinuidadCursorRRHHDesarrolloPrueba(t)
		cursor := cursorSesionRRHHDesarrolloPrueba('b')
		if err := continuador.recordar(primera, ports.PaginaCuadroRRHH{HayMas: true, CursorSiguiente: cursor}); err != nil {
			t.Fatal(err)
		}
		e.revalidador.err = errors.New("sesión revocada")
		segunda, err := continuador.preparar(segundaPeticionCursorRRHHDesarrolloPrueba(e, vinculoCanalCursorRRHHDesarrolloPrueba('1')), cursor)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = a.contextoConsultaRRHHDesarrollo(segunda); !errors.Is(err, ports.ErrAutorizacionDenegada) {
			t.Fatal("una sesión revocada conservó continuidad")
		}
	})
	t.Run("conexion_TLS_distinta", func(t *testing.T) {
		e, a, continuador, primera, _ := prepararContinuidadCursorRRHHDesarrolloPrueba(t)
		cursor := cursorSesionRRHHDesarrolloPrueba('g')
		if err := continuador.recordar(primera, ports.PaginaCuadroRRHH{HayMas: true, CursorSiguiente: cursor}); err != nil {
			t.Fatal(err)
		}
		segunda, err := continuador.preparar(segundaPeticionCursorRRHHDesarrolloPrueba(e, vinculoCanalCursorRRHHDesarrolloPrueba('2')), cursor)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = a.contextoConsultaRRHHDesarrollo(segunda); !errors.Is(err, ports.ErrAutorizacionDenegada) {
			t.Fatal("otra conexión TLS reutilizó el cursor")
		}
	})
	t.Run("certificado_distinto", func(t *testing.T) {
		e, a, continuador, primera, _ := prepararContinuidadCursorRRHHDesarrolloPrueba(t)
		cursor := cursorSesionRRHHDesarrolloPrueba('c')
		if err := continuador.recordar(primera, ports.PaginaCuadroRRHH{HayMas: true, CursorSiguiente: cursor}); err != nil {
			t.Fatal(err)
		}
		otro := clonarPrincipalDesarrollo(e.principal)
		otro.Attributes["certificate_sha256"] = strings.Repeat("e", 64)
		ctx := contextoRutaCoberturaDesarrolloPrueba(e.soporte, otro, httpinterno.RutaConsultaCuadroRRHH)
		capacidad := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
		capacidad.certificadoVerificadoEn = e.reloj.Ahora().Add(-1)
		capacidad.certificadoValidoHasta = e.reloj.Ahora().Add(1)
		capacidad.consultaRRHH = &contextoConsultaRRHHPeticionDesarrollo{}
		capacidad.vinculoCanalTLS = vinculoCanalCursorRRHHDesarrolloPrueba('1')
		ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
		segunda, err := continuador.preparar(ctx, cursor)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = a.contextoConsultaRRHHDesarrollo(segunda); !errors.Is(err, ports.ErrAutorizacionDenegada) {
			t.Fatal("un certificado distinto reutilizó el cursor")
		}
	})
	t.Run("sesion_distinta", func(t *testing.T) {
		e, _, continuador, primera, _ := prepararContinuidadCursorRRHHDesarrolloPrueba(t)
		cursor := cursorSesionRRHHDesarrolloPrueba('d')
		if err := continuador.recordar(primera, ports.PaginaCuadroRRHH{HayMas: true, CursorSiguiente: cursor}); err != nil {
			t.Fatal(err)
		}
		e.revalidador.alterar = func(a *dominiovec.AutenticacionRevalidadaV1) { a.SesionRef = "ses_otra234567890abcdefghijkl" }
		segunda, err := continuador.preparar(segundaPeticionCursorRRHHDesarrolloPrueba(e, vinculoCanalCursorRRHHDesarrolloPrueba('1')), cursor)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := continuador.contextoContinuado(segunda); ok {
			t.Fatal("una revalidación de otra sesión reutilizó el cursor")
		}
		if e.revalidador.llamadas != 2 {
			t.Fatal("la sesión distinta se rechazó antes de la revalidación real")
		}
	})
}

func TestContinuidadCursorRRHHSeparaLectoresYContinuadores(t *testing.T) {
	e, primeraAutoridad, primero, primeraPeticion, _ := prepararContinuidadCursorRRHHDesarrolloPrueba(t)
	cursor := cursorSesionRRHHDesarrolloPrueba('j')
	if err := primero.recordar(primeraPeticion, ports.PaginaCuadroRRHH{HayMas: true, CursorSiguiente: cursor}); err != nil {
		t.Fatal(err)
	}
	segundaAutoridad := &autoridadConsultasRRHHDesarrollo{
		soporte: e.soporte, reloj: e.reloj, proveedor: e.p,
		clase: ports.AmbitoOrganizacionRRHH, ambitoRef: "organizacion:lector-distinto",
	}
	segundo, err := nuevoContinuadorSesionCursorRRHHDesarrollo(segundaAutoridad, e.reloj)
	if err != nil {
		t.Fatal(err)
	}
	ctx := segundaPeticionCursorRRHHDesarrolloPrueba(e, vinculoCanalCursorRRHHDesarrolloPrueba('1'))
	if _, err = segundo.preparar(ctx, cursor); !errors.Is(err, ports.ErrAutorizacionDenegada) {
		t.Fatal("un lector distinto recuperó la continuidad ajena")
	}
	if _, err = primero.preparar(ctx, cursor); err != nil {
		t.Fatal("el rechazo del lector distinto alteró el cursor de su dueño")
	}
	if primeraAutoridad == segundaAutoridad {
		t.Fatal("la prueba no separó las autoridades de lector")
	}
}

func TestContinuidadCursorRRHHCierraReservaAnteCancelacion(t *testing.T) {
	e, _, continuador, primera, _ := prepararContinuidadCursorRRHHDesarrolloPrueba(t)
	cursor := cursorSesionRRHHDesarrolloPrueba('e')
	if err := continuador.recordar(primera, ports.PaginaCuadroRRHH{HayMas: true, CursorSiguiente: cursor}); err != nil {
		t.Fatal(err)
	}
	cancelada, cancelar := context.WithCancel(e.contexto())
	cancelar()
	if _, err := continuador.preparar(cancelada, cursor); !errors.Is(err, ports.ErrAutorizacionDenegada) {
		t.Fatal("una petición cancelada reservó continuidad")
	}
	if _, err := continuador.preparar(e.contexto(), cursor); err != nil {
		t.Fatal("la cancelación previa consumió la continuidad")
	}
}

func TestContinuidadCursorRRHHNoPrometeReintentoTrasFalloAntesDeSQL(t *testing.T) {
	e, _, continuador, primera, _ := prepararContinuidadCursorRRHHDesarrolloPrueba(t)
	cursor := cursorSesionRRHHDesarrolloPrueba('f')
	if err := continuador.recordar(primera, ports.PaginaCuadroRRHH{HayMas: true, CursorSiguiente: cursor}); err != nil {
		t.Fatal(err)
	}
	delegado, err := nuevoConsultorCuadroConSesionCursorRRHHDesarrollo(
		consultorCursorRRHHFallaPrueba{err: errors.New("fallo antes de PostgreSQL")}, continuador,
	)
	if err != nil {
		t.Fatal(err)
	}
	segunda := segundaPeticionCursorRRHHDesarrolloPrueba(e, vinculoCanalCursorRRHHDesarrolloPrueba('1'))
	solicitud, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 2, cursor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = delegado.Consultar(segunda, solicitud); err == nil {
		t.Fatal("el doble debía fallar antes de PostgreSQL")
	}
	if _, err = continuador.preparar(e.contexto(), cursor); !errors.Is(err, ports.ErrAutorizacionDenegada) {
		t.Fatal("el seam permitió repetir un cursor tras reservarlo")
	}
}

func TestContinuidadCursorRRHHReservaConcurrenteYAcotaCache(t *testing.T) {
	e, _, continuador, primera, _ := prepararContinuidadCursorRRHHDesarrolloPrueba(t)
	cursor := cursorSesionRRHHDesarrolloPrueba('h')
	if err := continuador.recordar(primera, ports.PaginaCuadroRRHH{HayMas: true, CursorSiguiente: cursor}); err != nil {
		t.Fatal(err)
	}
	var grupo sync.WaitGroup
	exitos := make(chan struct{}, 2)
	for rango := 0; rango < 2; rango++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			if _, err := continuador.preparar(segundaPeticionCursorRRHHDesarrolloPrueba(e, vinculoCanalCursorRRHHDesarrolloPrueba('1')), cursor); err == nil {
				exitos <- struct{}{}
			}
		}()
	}
	grupo.Wait()
	close(exitos)
	if len(exitos) != 1 {
		t.Fatal("la reserva concurrente no consumió exactamente una continuidad")
	}
	for indice := 0; indice <= limiteContinuidadesCursorRRHHDesarrollo; indice++ {
		cursor := cursorSesionRRHHDesarrolloPrueba(byte(indice))
		if err := continuador.recordar(primera, ports.PaginaCuadroRRHH{HayMas: true, CursorSiguiente: cursor}); err != nil {
			t.Fatal(err)
		}
	}
	continuador.mu.Lock()
	longitud := len(continuador.entradas)
	continuador.mu.Unlock()
	if longitud > limiteContinuidadesCursorRRHHDesarrollo {
		t.Fatal("la caché de continuidad superó el límite")
	}
	caducado := cursorSesionRRHHDesarrolloPrueba('z')
	if err := continuador.recordar(primera, ports.PaginaCuadroRRHH{HayMas: true, CursorSiguiente: caducado}); err != nil {
		t.Fatal(err)
	}
	e.reloj.ahora = e.reloj.ahora.Add(2 * time.Minute)
	if _, err := continuador.preparar(segundaPeticionCursorRRHHDesarrolloPrueba(e, vinculoCanalCursorRRHHDesarrolloPrueba('1')), caducado); !errors.Is(err, ports.ErrAutorizacionDenegada) {
		t.Fatal("una continuidad caducada fue aceptada")
	}
}

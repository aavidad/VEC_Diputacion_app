package bootstrap

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/app/composicion/identidadordinaria"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

func estadoTLSRealFirmanteV2Prueba(t *testing.T) tls.ConnectionState {
	t.Helper()
	_, rutas := generarMaterialDesarrolloPrueba(t)
	certServidor, err := tls.LoadX509KeyPair(rutas.ServerCertificate, rutas.ServerPrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	certCliente, err := tls.LoadX509KeyPair(rutas.ClientCertificate, rutas.ClientPrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	caPEM, err := os.ReadFile(rutas.CACertificate)
	if err != nil {
		t.Fatal(err)
	}
	raices := x509.NewCertPool()
	if !raices.AppendCertsFromPEM(caPEM) {
		t.Fatal("CA de prueba inválida")
	}
	hojaServidor, err := x509.ParseCertificate(certServidor.Certificate[0])
	if err != nil || len(hojaServidor.DNSNames) == 0 {
		t.Fatal("certificado de servidor sin nombre", err)
	}
	ladoServidor, ladoCliente := net.Pipe()
	servidor := tls.Server(ladoServidor, &tls.Config{Certificates: []tls.Certificate{certServidor},
		ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: raices, MinVersion: tls.VersionTLS13})
	cliente := tls.Client(ladoCliente, &tls.Config{Certificates: []tls.Certificate{certCliente},
		RootCAs: raices, ServerName: hojaServidor.DNSNames[0], MinVersion: tls.VersionTLS13})
	defer servidor.Close()
	defer cliente.Close()
	errores := make(chan error, 2)
	go func() { errores <- servidor.Handshake() }()
	go func() { errores <- cliente.Handshake() }()
	for i := 0; i < 2; i++ {
		if err := <-errores; err != nil {
			t.Fatal("handshake de prueba", err)
		}
	}
	return servidor.ConnectionState()
}

func TestSesionFirmanteCertificadoLigaCanalTLSReal(t *testing.T) {
	primera := httptest.NewRequest("POST", httpinterno.RutaRegistroFirmaVec, nil)
	primera.TLS = new(tls.ConnectionState)
	*primera.TLS = estadoTLSRealFirmanteV2Prueba(t)
	uno, ok := canalSesionFirmanteV2(primera)
	if !ok || !huellaSHA256ValidaContratacionTemporalDesarrollo(uno) {
		t.Fatal("canal TLS real no derivado")
	}
	segunda := primera.Clone(primera.Context())
	segunda.TLS = new(tls.ConnectionState)
	*segunda.TLS = estadoTLSRealFirmanteV2Prueba(t)
	dos, ok := canalSesionFirmanteV2(segunda)
	if !ok || dos == uno {
		t.Fatal("otra conexión reutilizó el canal esperado")
	}
	segunda.TLS = &tls.ConnectionState{HandshakeComplete: true, Version: tls.VersionTLS13,
		PeerCertificates: primera.TLS.PeerCertificates, VerifiedChains: primera.TLS.VerifiedChains}
	if _, ok := canalSesionFirmanteV2(segunda); ok {
		t.Fatal("certificado sin exportador TLS admitido como canal")
	}
}

func TestSesionFirmanteCertificadoExigeFuenteComunYCotejosAUT56(t *testing.T) {
	e := nuevoEntornoFirmanteV2Prueba(t)
	if _, err := nuevaAutoridadSesionFirmanteV2Certificado(nil, e.reloj); !errors.Is(err, errSesionFirmanteV2NoDisponible) {
		t.Fatalf("fuente ausente: %v", err)
	}
	a, err := nuevaAutoridadSesionFirmanteV2Certificado(&identidadordinaria.FuenteCertificadoTemporal{}, e.reloj)
	if err != nil || a.garantia != core.AuthAssuranceSubstantial || !a.exigirCanalTLS {
		t.Fatalf("garantía temporal: %v", err)
	}
	f := a.fuente.(*fuenteCertificadoFirmaVecV2)
	ahora := e.reloj.Ahora()
	q := ports.SolicitudSesionFirmanteV2{
		CertificadoCanalSHA256: e.huella, CanalTLSVinculadoSHA256: strings.Repeat("b", 64),
		PersonaEsperadaRef: e.persona,
		CuentaEsperadaRef:  e.seleccion.CuentaRef, PerfilEsperadoRef: e.seleccion.PerfilActivoRef,
		RolEsperadoID: e.seleccion.RolID, CertificadoVerificadoEn: ahora,
		CertificadoTLSValidoHasta: ahora.Add(time.Minute),
	}
	if !solicitudCertificadoFirmaVecV2Valida(q, ahora) {
		t.Fatal("cotejos completos rechazados")
	}
	for nombre, cambiar := range map[string]func(*ports.SolicitudSesionFirmanteV2){
		"sin_certificado": func(q *ports.SolicitudSesionFirmanteV2) { q.CertificadoCanalSHA256 = "" },
		"sin_canal":       func(q *ports.SolicitudSesionFirmanteV2) { q.CanalTLSVinculadoSHA256 = "" },
		"sin_persona":     func(q *ports.SolicitudSesionFirmanteV2) { q.PersonaEsperadaRef = "" },
		"sin_cuenta":      func(q *ports.SolicitudSesionFirmanteV2) { q.CuentaEsperadaRef = "" },
		"sin_perfil":      func(q *ports.SolicitudSesionFirmanteV2) { q.PerfilEsperadoRef = "" },
		"sin_rol":         func(q *ports.SolicitudSesionFirmanteV2) { q.RolEsperadoID = "" },
		"sin_vigencia":    func(q *ports.SolicitudSesionFirmanteV2) { q.CertificadoTLSValidoHasta = ahora },
		"fecha_futura":    func(q *ports.SolicitudSesionFirmanteV2) { q.CertificadoVerificadoEn = ahora.Add(time.Second) },
	} {
		t.Run(nombre, func(t *testing.T) {
			mala := q
			cambiar(&mala)
			if _, err := f.AbrirSesionFirmanteV2(context.Background(), mala); !errors.Is(err, errSesionFirmanteV2Denegada) {
				t.Fatalf("cotejo ausente admitido: %v", err)
			}
		})
	}
	if _, err := f.AbrirSesionFirmanteV2(context.Background(), q); !errors.Is(err, errSesionFirmanteV2Denegada) {
		t.Fatalf("sin cápsula vinculada abrió sesión: %v", err)
	}
}

func TestSesionFirmanteCertificadoConservaCancelacionSinFiltrarCausa(t *testing.T) {
	e := nuevoEntornoFirmanteV2Prueba(t)
	a, err := nuevaAutoridadSesionFirmanteV2Certificado(&identidadordinaria.FuenteCertificadoTemporal{}, e.reloj)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	q := ports.SolicitudSesionFirmanteV2{CertificadoCanalSHA256: e.huella,
		CanalTLSVinculadoSHA256: strings.Repeat("b", 64),
		PersonaEsperadaRef:      e.persona, CuentaEsperadaRef: e.seleccion.CuentaRef,
		PerfilEsperadoRef: e.seleccion.PerfilActivoRef, RolEsperadoID: e.seleccion.RolID,
		CertificadoVerificadoEn: e.reloj.Ahora(), CertificadoTLSValidoHasta: e.reloj.Ahora().Add(time.Minute)}
	_, err = a.fuente.AbrirSesionFirmanteV2(ctx, q)
	if !errors.Is(err, errSesionFirmanteV2Denegada) || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación previa inesperada: %v", err)
	}
	opaco := falloSesionFirmanteV2(errors.New("material privado de la autoridad"))
	if !errors.Is(opaco, errSesionFirmanteV2Denegada) ||
		strings.Contains(fmt.Sprintf("%+v", opaco), "material privado") {
		t.Fatal("error externo filtró la causa")
	}
	if !errors.Is(falloSesionFirmanteV2(context.Canceled), context.Canceled) {
		t.Fatal("cancelación interna sin causa")
	}
	c, err := e.a.acreditar(e.r, e.seleccion, e.persona, e.huella)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := e.a.abrirConContexto(ctx, e.r, c); !errors.Is(err, context.Canceled) ||
		!errors.Is(err, errSesionFirmanteV2Denegada) || strings.Contains(fmt.Sprintf("%+v", err), "material privado") {
		t.Fatalf("apertura perdió cancelación u opacidad: %v", err)
	}
	if _, _, err := e.a.revalidarConContexto(ctx, e.r, c); !errors.Is(err, context.Canceled) ||
		!errors.Is(err, errSesionFirmanteV2Denegada) {
		t.Fatalf("revalidación perdió cancelación: %v", err)
	}
	r, err := prepararPeticionFirmaVecV2(e.r)
	if err != nil || r.URL.Path != httpinterno.RutaRegistroFirmaVec {
		t.Fatal(err)
	}
	f := &fuenteNominalFirmaVecV2{autoridad: e.a}
	ctxPeticion, cancelarPeticion := context.WithCancel(r.Context())
	cancelarPeticion()
	if _, err := f.RevalidarContextoActorFirmaV2(ctxPeticion); !errors.Is(err, context.Canceled) ||
		!errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
		t.Fatalf("fuente de emisor perdió cancelación: %v", err)
	}
}

type fuenteErrorFirmaVecV2Prueba struct {
	sesion ports.SesionFirmanteV2
	err    error
}

func (f *fuenteErrorFirmaVecV2Prueba) AbrirSesionFirmanteV2(context.Context,
	ports.SolicitudSesionFirmanteV2) (ports.SesionFirmanteV2, error) {
	return f.sesion, f.err
}

type sesionErrorFirmaVecV2Prueba struct {
	evidencia  ports.EvidenciaSesionFirmanteV2
	fallarTras int
	llamadas   int
}

func (s *sesionErrorFirmaVecV2Prueba) RevalidarSesionFirmanteV2(context.Context) (ports.EvidenciaSesionFirmanteV2, error) {
	s.llamadas++
	if s.llamadas > s.fallarTras {
		return ports.EvidenciaSesionFirmanteV2{}, context.Canceled
	}
	return s.evidencia, nil
}

func TestSesionFirmanteV2ConservaCancelacionDeFuenteDuranteAmbosUsos(t *testing.T) {
	e := nuevoEntornoFirmanteV2Prueba(t)
	proyectarEmpleadoFirmanteV2Prueba(t, &e)
	base, err := e.a.acreditar(e.r, e.seleccion, e.persona, e.huella)
	if err != nil {
		t.Fatal(err)
	}
	vinculo, resultado, err := e.a.abrir(e.r, base)
	if err != nil {
		t.Fatal(err)
	}
	fuente := &fuenteErrorFirmaVecV2Prueba{err: context.Canceled}
	a, err := nuevaAutoridadSesionFirmanteV2ConFuente(fuente, e.reloj, nil)
	if err != nil {
		t.Fatal(err)
	}
	c, err := a.acreditar(e.r, e.seleccion, e.persona, e.huella)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.abrir(e.r, c); !errors.Is(err, context.Canceled) ||
		!errors.Is(err, errSesionFirmanteV2Denegada) || strings.Contains(err.Error(), "canceled") {
		t.Fatalf("apertura perdió causa u opacidad: %v", err)
	}
	sesion := &sesionErrorFirmaVecV2Prueba{evidencia: ports.EvidenciaSesionFirmanteV2{
		Vinculo: vinculo, Resultado: resultado, CertificadoCanalSHA256: e.huella,
		CertificadoValidoHasta: e.r.TLS.VerifiedChains[0][0].NotAfter.UTC().Truncate(time.Microsecond)},
		fallarTras: 1}
	fuente.err, fuente.sesion = nil, sesion
	c, err = a.acreditar(e.r, e.seleccion, e.persona, e.huella)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.abrir(e.r, c); err != nil {
		t.Fatalf("primera revalidación: %v", err)
	}
	if _, _, err := a.revalidar(e.r, c); !errors.Is(err, context.Canceled) ||
		!errors.Is(err, errSesionFirmanteV2Denegada) || strings.Contains(err.Error(), "canceled") {
		t.Fatalf("revalidación perdió causa u opacidad: %v", err)
	}
}

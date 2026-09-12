package bootstrap

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/config"
)

func TestReservarEscuchasServidoresDesarrolloReservaAmbosAntesDeServir(t *testing.T) {
	rrhh := &http.Server{Addr: "127.0.0.1:0", Handler: http.NotFoundHandler()}
	administracion := &http.Server{Addr: "127.0.0.1:0", Handler: http.NotFoundHandler()}

	escuchas, err := reservarEscuchasServidoresDesarrollo(rrhh, administracion)
	if err != nil {
		t.Fatalf("reservar ambos listeners: %v", err)
	}
	t.Cleanup(escuchas.cerrar)
	if escuchas.rrhh == nil || escuchas.administracion == nil {
		t.Fatalf("listeners incompletos: %+v", escuchas)
	}
	if escuchas.rrhh.Addr().String() == escuchas.administracion.Addr().String() {
		t.Fatalf("los dos servidores comparten socket: %q", escuchas.rrhh.Addr())
	}
}

func TestServirServidoresDesarrolloLiberaRRHHYAdminSiNoReservaElSegundo(t *testing.T) {
	bloqueado, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = bloqueado.Close() })

	var cierres atomic.Int32
	err = servirServidoresDesarrollo(
		t.Context(),
		config.Config{},
		&http.Server{Addr: "127.0.0.1:0", Handler: http.NotFoundHandler()},
		&http.Server{Addr: bloqueado.Addr().String(), Handler: http.NotFoundHandler()},
		func() { cierres.Add(1) },
	)
	if err == nil {
		t.Fatal("la reserva del puerto ADMIN ocupado fue aceptada")
	}
	if cierres.Load() != 1 {
		t.Fatalf("cierre ADMIN=%d; se esperaba uno", cierres.Load())
	}
}

func TestServirServidoresDesarrolloCierraAmbosTrasCancelarContexto(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	direccionRRHH := direccionEfimeraPrueba(t)
	direccionAdministracion := direccionEfimeraPrueba(t)
	certificado, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancelar := context.WithCancel(t.Context())
	var cierres atomic.Int32
	resultado := make(chan error, 1)
	go func() {
		resultado <- servirServidoresDesarrollo(
			ctx,
			cfg,
			&http.Server{Addr: direccionRRHH, Handler: http.NotFoundHandler()},
			&http.Server{Addr: direccionAdministracion, Handler: http.NotFoundHandler(), TLSConfig: &tls.Config{Certificates: []tls.Certificate{certificado}, MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13}},
			func() { cierres.Add(1) },
		)
	}()
	esperarConexionDesarrolloPrueba(t, direccionRRHH)
	esperarConexionDesarrolloPrueba(t, direccionAdministracion)
	cancelar()
	if err := <-resultado; err != nil {
		t.Fatalf("cerrar ciclo conjunto: %v", err)
	}
	if cierres.Load() != 1 {
		t.Fatalf("cierre ADMIN=%d; se esperaba uno", cierres.Load())
	}
}

func direccionEfimeraPrueba(t *testing.T) string {
	t.Helper()
	escucha, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	direccion := escucha.Addr().String()
	if err := escucha.Close(); err != nil {
		t.Fatal(err)
	}
	return direccion
}

func esperarConexionDesarrolloPrueba(t *testing.T, direccion string) {
	t.Helper()
	limite := time.Now().Add(2 * time.Second)
	for time.Now().Before(limite) {
		conexion, err := net.DialTimeout("tcp", direccion, 25*time.Millisecond)
		if err == nil {
			_ = conexion.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("el listener no aceptó conexiones en %s", direccion)
}

func TestServeDesarrolloRechazaPresentacionAntesDeAbrirMaterialOPools(t *testing.T) {
	cfg := config.Config{ExecutionProfile: config.ExecutionProfileDevelopment, RRHHPresentationEnabled: true}
	err := ServeDesarrollo(t.Context(), cfg, io.Discard)
	if !errors.Is(err, ErrPresentacionRRHHEnComposicionNormal) {
		t.Fatalf("no conserva guarda previa: %v", err)
	}
}

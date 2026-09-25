package main

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestCierreEsperaPeticionYRechazaNuevas(t *testing.T) {
	escucha, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	iniciada := make(chan struct{})
	liberar := make(chan struct{})
	terminada := make(chan struct{})
	listenerCerrado := make(chan struct{})
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(iniciada)
		<-liberar
		_, _ = io.WriteString(w, "terminada")
		close(terminada)
	})}
	srv.RegisterOnShutdown(func() { close(listenerCerrado) })
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	cierreRecursos := make(chan bool, 1)
	servidorTerminado := make(chan error, 1)
	go func() {
		servidorTerminado <- ejecutarServidor(ctx, srv, func() error { return srv.Serve(escucha) }, func(context.Context) error {
			select {
			case <-terminada:
				cierreRecursos <- true
			default:
				cierreRecursos <- false
			}
			return nil
		}, time.Second)
	}()
	cliente := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}, Timeout: time.Second}
	respuesta := make(chan error, 1)
	go func() {
		r, e := cliente.Get("http://" + escucha.Addr().String() + "/lenta")
		if e != nil {
			respuesta <- e
			return
		}
		defer r.Body.Close()
		cuerpo, e := io.ReadAll(r.Body)
		if e == nil && string(cuerpo) != "terminada" {
			e = errors.New("respuesta incompleta")
		}
		respuesta <- e
	}()
	<-iniciada
	cancelar()
	<-listenerCerrado
	if _, err := cliente.Get("http://" + escucha.Addr().String() + "/nueva"); err == nil {
		t.Fatal("el listener aceptó una petición nueva")
	}
	select {
	case <-cierreRecursos:
		t.Fatal("los recursos cerraron con la petición activa")
	default:
	}
	close(liberar)
	if err := <-respuesta; err != nil {
		t.Fatal(err)
	}
	if err := <-servidorTerminado; err != nil {
		t.Fatal(err)
	}
	if !<-cierreRecursos {
		t.Fatal("los recursos cerraron antes de terminar la petición")
	}
}

func TestCierreDevuelveErrorAlVencerPlazo(t *testing.T) {
	escucha, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	iniciada := make(chan struct{})
	liberar := make(chan struct{})
	srv := &http.Server{Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(iniciada)
		<-liberar
	})}
	ctx, cancelar := context.WithCancel(context.Background())
	terminado := make(chan error, 1)
	go func() {
		terminado <- ejecutarServidor(ctx, srv, func() error { return srv.Serve(escucha) }, nil, 20*time.Millisecond)
	}()
	cliente := &http.Client{Timeout: time.Second}
	go func() {
		r, e := cliente.Get("http://" + escucha.Addr().String())
		if e == nil {
			r.Body.Close()
		}
	}()
	<-iniciada
	cancelar()
	if err := <-terminado; !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("plazo vencido: %v", err)
	}
	close(liberar)
}

func TestPlazoDeRecursosAcotado(t *testing.T) {
	liberar := make(chan struct{})
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancelar()
	err := cerrarRecursos(ctx, func(context.Context) error {
		<-liberar
		return nil
	})
	close(liberar)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("plazo de recursos: %v", err)
	}
}

func TestShutdownYRecursosCompartenPlazo(t *testing.T) {
	escucha, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	iniciada := make(chan struct{})
	liberar := make(chan struct{})
	srv := &http.Server{Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(iniciada)
		<-liberar
	})}
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	const plazo = 200 * time.Millisecond
	restanteCierre := make(chan time.Duration, 1)
	terminado := make(chan error, 1)
	go func() {
		terminado <- ejecutarServidor(ctx, srv, func() error { return srv.Serve(escucha) }, func(ctx context.Context) error {
			limite, ok := ctx.Deadline()
			if !ok {
				restanteCierre <- -1
			} else {
				restanteCierre <- time.Until(limite)
			}
			return nil
		}, plazo)
	}()
	cliente := &http.Client{Timeout: time.Second}
	go func() {
		r, e := cliente.Get("http://" + escucha.Addr().String())
		if e == nil {
			r.Body.Close()
		}
	}()
	<-iniciada
	inicio := time.Now()
	cancelar()
	err = <-terminado
	close(liberar)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown debe informar plazo vencido: %v", err)
	}
	if duracion := <-restanteCierre; duracion <= 0 || duracion > plazo/2 {
		t.Fatalf("tiempo restante del cierre = %s; debe reservarse dentro del plazo total", duracion)
	}
	if transcurrido := time.Since(inicio); transcurrido > plazo+100*time.Millisecond {
		t.Fatalf("cierre total = %s, plazo = %s", transcurrido, plazo)
	}
}

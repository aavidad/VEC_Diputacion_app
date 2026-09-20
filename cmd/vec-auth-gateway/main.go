package main

import (
	"context"
	"crypto/rand"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	composicion "vec-diputacion-granada/internal/app/composicion/gatewaypersonal"
	adaptador "vec-diputacion-granada/internal/vec/adapters/httpseguridad/gatewaypersonal"
)

func main() {
	cfg, err := composicion.CargarConfiguracion()
	if err != nil {
		log.Fatal("gateway personal: configuracion no disponible")
	}
	tlsConfig, crl, cuentas, err := cfg.TLS()
	if err != nil {
		log.Fatal("gateway personal: material TLS no disponible")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
	almacen, err := adaptador.Abrir(ctx, cfg.DSN)
	cancelar()
	if err != nil {
		log.Fatal("gateway personal: sesion durable no disponible")
	}
	defer almacen.CerrarPool()
	servicio, err := composicion.Nuevo(almacen, cfg.ClaveHMAC, rand.Reader, nil)
	if err != nil {
		log.Fatal("gateway personal: nucleo no disponible")
	}
	manejador := adaptador.NuevoHandler(servicio, cfg.Origen, cuentas, crl, nil)
	if err := manejador.ConfigurarActivos(cfg.RaizWeb); err != nil {
		log.Fatal("gateway personal: activos no disponibles")
	}
	servidor := &http.Server{Addr: cfg.Direccion, Handler: manejador.Rutas(), TLSConfig: tlsConfig, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	detener, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parar()
	terminado := make(chan error, 1)
	go func() { terminado <- servidor.ListenAndServeTLS("", "") }()
	select {
	case err := <-terminado:
		if err != nil && err != http.ErrServerClosed {
			log.Fatal("gateway personal: escucha no disponible")
		}
	case <-detener.Done():
		cierre, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = servidor.Shutdown(cierre)
	}
}

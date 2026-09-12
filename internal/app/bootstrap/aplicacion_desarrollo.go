package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"vec-diputacion-granada/config"
)

// ServeDesarrollo sirve conjuntamente las superficies RRHH y ADMIN del perfil
// de desarrollo. Reserva todos los sockets antes de admitir peticiones: una
// configuración ADMIN inválida o un puerto ocupado no deja RRHH expuesto.
//
// ADMIN es opcional mientras su configuración PostgreSQL esté completamente
// ausente. La factoría devuelve entonces servidor nil y un cierre inocuo.
func ServeDesarrollo(ctx context.Context, cfg config.Config, registro io.Writer) error {
	if ctx == nil {
		return errors.New("bootstrap: contexto de desarrollo ausente")
	}

	cfg = cfg.Normalize()
	if err := validarValoresConfiguracionConocidos(cfg); err != nil {
		return err
	}
	if err := rechazarSelectoresPresentacionEnComposicionNormal(cfg); err != nil {
		return err
	}
	if err := rechazarTLSDesarrolloEnProduccion(cfg); err != nil {
		return err
	}
	rrhh, composicion, err := NewHTTPServerDesarrolloWithConfig(cfg, registro)
	if err != nil {
		return err
	}
	// Shutdown ejecuta callbacks en goroutines. Invocar el mismo OnceFunc
	// al salir espera a que el cierre real de pools CT haya terminado.
	if composicion.cerrarContratacion != nil {
		defer composicion.cerrarContratacion()
	}
	administracion, cerrarAdministracion, err := NewHTTPServerAdministracionDesarrolloWithConfig(cfg, composicion)
	if err != nil {
		cerrarServidorDesarrollo(rrhh)
		return err
	}
	if cerrarAdministracion == nil {
		cerrarAdministracion = func() {}
	}
	return servirServidoresDesarrollo(ctx, cfg, rrhh, administracion, cerrarAdministracion)
}

type escuchasServidoresDesarrollo struct {
	rrhh           net.Listener
	administracion net.Listener
}

func reservarEscuchasServidoresDesarrollo(rrhh, administracion *http.Server) (escuchasServidoresDesarrollo, error) {
	if rrhh == nil || strings.TrimSpace(rrhh.Addr) == "" {
		return escuchasServidoresDesarrollo{}, errors.New("bootstrap: servidor RRHH de desarrollo ausente")
	}
	escuchaRRHH, err := net.Listen("tcp", rrhh.Addr)
	if err != nil {
		return escuchasServidoresDesarrollo{}, fmt.Errorf("reservar listener RRHH: %w", err)
	}
	resultado := escuchasServidoresDesarrollo{rrhh: escuchaRRHH}
	if administracion == nil {
		return resultado, nil
	}
	if strings.TrimSpace(administracion.Addr) == "" {
		escuchaRRHH.Close()
		return escuchasServidoresDesarrollo{}, errors.New("bootstrap: servidor ADMIN de desarrollo sin dirección")
	}
	escuchaAdministracion, err := net.Listen("tcp", administracion.Addr)
	if err != nil {
		escuchaRRHH.Close()
		return escuchasServidoresDesarrollo{}, fmt.Errorf("reservar listener ADMIN: %w", err)
	}
	resultado.administracion = escuchaAdministracion
	return resultado, nil
}

func (e escuchasServidoresDesarrollo) cerrar() {
	if e.rrhh != nil {
		_ = e.rrhh.Close()
	}
	if e.administracion != nil {
		_ = e.administracion.Close()
	}
}

func servirServidoresDesarrollo(ctx context.Context, cfg config.Config, rrhh, administracion *http.Server, cerrarAdministracion func()) error {
	escuchas, err := reservarEscuchasServidoresDesarrollo(rrhh, administracion)
	if err != nil {
		cerrarAdministracion()
		cerrarServidorDesarrollo(rrhh)
		return err
	}

	var cerrarUnaVez sync.Once
	cerrar := func() {
		cerrarUnaVez.Do(func() {
			apagarServidorDesarrollo(rrhh)
			apagarServidorDesarrollo(administracion)
			escuchas.cerrar()
			cerrarAdministracion()
		})
	}

	var servidoresEnCurso sync.WaitGroup
	defer servidoresEnCurso.Wait()
	defer cerrar()

	type resultadoServidor struct {
		nombre string
		err    error
	}
	resultados := make(chan resultadoServidor, 2)
	servir := func(nombre string, servidor *http.Server, escucha net.Listener, certificado, clave string) {
		if servidor == nil || escucha == nil {
			return
		}
		servidoresEnCurso.Add(1)
		go func() {
			defer servidoresEnCurso.Done()
			resultados <- resultadoServidor{nombre: nombre, err: servidor.ServeTLS(escucha, certificado, clave)}
		}()
	}
	servir("RRHH", rrhh, escuchas.rrhh, cfg.TLSCertFile, cfg.TLSKeyFile)
	// ADMIN carga su par de claves desde su configuración TLS propia. Pasar
	// los ficheros de RRHH aquí sustituiría esa identidad segregada.
	servir("ADMIN", administracion, escuchas.administracion, "", "")

	servidores := 1
	if administracion != nil {
		servidores++
	}
	for servidores > 0 {
		select {
		case <-ctx.Done():
			return nil
		case resultado := <-resultados:
			servidores--
			if resultado.err != nil && !errors.Is(resultado.err, http.ErrServerClosed) {
				return fmt.Errorf("servir %s desarrollo: %w", resultado.nombre, resultado.err)
			}
			if servidores > 0 {
				return fmt.Errorf("servidor %s de desarrollo terminó inesperadamente", resultado.nombre)
			}
		}
	}
	return nil
}

func cerrarServidorDesarrollo(servidor *http.Server) {
	apagarServidorDesarrollo(servidor)
}

func apagarServidorDesarrollo(servidor *http.Server) {
	if servidor == nil {
		return
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	if err := servidor.Shutdown(ctx); err != nil {
		_ = servidor.Close()
	}
}

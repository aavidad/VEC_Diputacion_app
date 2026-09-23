package interna

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

type extractorAsercionInternaPrueba struct{ llamadas int }

func (e *extractorAsercionInternaPrueba) ExtraerAsercionProtegida(*http.Request) ([]byte, error) {
	e.llamadas++
	return []byte("asercion de prueba"), nil
}

type recursoAdquiridoLecturaPrueba struct {
	propiedad atomic.Bool
	cierres   atomic.Int32
}

func (r *recursoAdquiridoLecturaPrueba) reclamarPropiedad() bool {
	return r.propiedad.CompareAndSwap(false, true)
}

func (r *recursoAdquiridoLecturaPrueba) cerrar() error {
	r.cierres.Add(1)
	return nil
}

func TestCargaParcialYCancellationCierranRecursosUnaVezSinListener(t *testing.T) {
	for _, caso := range []string{"fallo cargador", "cancelación posterior"} {
		t.Run(caso, func(t *testing.T) {
			reserva, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			direccion := reserva.Addr().String()
			_ = reserva.Close()
			cfg := configuracionInternaValidaPrueba()
			cfg.DireccionEscucha = direccion
			cfg.RedesPermitidas = []string{"127.0.0.0/8"}
			recurso := &recursoAdquiridoLecturaPrueba{}
			ctx, cancelar := context.WithCancel(context.Background())
			defer cancelar()
			aplicacion, err := nuevaAplicacionConCargador(ctx, cfg, func(context.Context, Configuracion) (proveedoresLecturaCT, error) {
				if caso == "cancelación posterior" {
					cancelar()
					return proveedoresLecturaCT{recursos: []recursoCerrableAplicacionInterna{recurso}}, nil
				}
				return proveedoresLecturaCT{recursos: []recursoCerrableAplicacionInterna{recurso}}, ErrDependenciasProductivasNoDisponibles
			})
			if aplicacion != nil || !errors.Is(err, ErrDependenciasProductivasNoDisponibles) || recurso.cierres.Load() != 1 {
				t.Fatalf("carga parcial = (%v, %v), cierres=%d", aplicacion, err, recurso.cierres.Load())
			}
			listener, err := net.Listen("tcp", direccion)
			if err != nil {
				t.Fatalf("se abrió listener antes de completar proveedores: %v", err)
			}
			_ = listener.Close()
		})
	}
}

func TestPuenteInternoNoDelegaSinSelloNiCanalC4(t *testing.T) {
	llamadasAPI := 0
	extractor := &extractorAsercionInternaPrueba{}
	puente := &puenteIdentidadCT{
		fuente: extractor,
		api:    http.HandlerFunc(func(http.ResponseWriter, *http.Request) { llamadasAPI++ }),
	}
	peticion := httptest.NewRequest(http.MethodPost, "/api/vec/contratacion-temporal/cuadro/consultas", nil)
	rec := httptest.NewRecorder()
	puente.ServeHTTP(rec, peticion)
	if rec.Code != http.StatusServiceUnavailable || extractor.llamadas != 0 || llamadasAPI != 0 {
		t.Fatalf("puente sin sello = %d, extractor=%d, API=%d", rec.Code, extractor.llamadas, llamadasAPI)
	}
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	servidor, err := construirServidorInternoPrueba(t, material.cfg, puente)
	if err != nil {
		t.Fatal(err)
	}
	f, err := NuevaFachadaIdentidadOffline(&httpseguridad.ServicioIdentidad{}, servidor)
	if err != nil || !puente.sellar(f) || puente.sellar(f) {
		t.Fatalf("sello único = %v", err)
	}
	rec = httptest.NewRecorder()
	puente.ServeHTTP(rec, peticion)
	if rec.Code != http.StatusUnauthorized || extractor.llamadas != 1 || llamadasAPI != 0 {
		t.Fatalf("sin canal C4 = %d, extractor=%d, API=%d", rec.Code, extractor.llamadas, llamadasAPI)
	}
}

func TestNuevaAplicacionLecturaCTRechazaProveedoresIncompletos(t *testing.T) {
	aplicacion, err := nuevaAplicacionLecturaCT(nil, configuracionInternaValidaPrueba(), proveedoresLecturaCT{})
	if aplicacion != nil || !errors.Is(err, ErrDependenciasProductivasNoDisponibles) {
		t.Fatalf("proveedores incompletos = (%v, %v)", aplicacion, err)
	}
}

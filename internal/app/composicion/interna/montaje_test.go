package interna

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

type extractorAsercionInternaPrueba struct{ llamadas int }

func (e *extractorAsercionInternaPrueba) ExtraerAsercionProtegida(*http.Request) ([]byte, error) {
	e.llamadas++
	return []byte("asercion de prueba"), nil
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

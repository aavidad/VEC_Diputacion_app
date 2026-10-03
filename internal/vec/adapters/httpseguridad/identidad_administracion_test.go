package httpseguridad

import (
	"errors"
	"testing"
	"time"
)

func TestTiemposAdministracionNoSuperanRetirada(t *testing.T) {
	ahora := time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
	cfg := configuracionAdministracionValida()
	cfg.PoliticaAdministracion = PoliticaAdministracionCertificadoTemporal
	cfg.RetiradaPoliticaAdministracionEn = ahora.Add(30 * time.Second)
	estado := estadoIdentidadSesion{autenticacionVerificadaEn: ahora, emitidaEn: ahora, noAntesDe: ahora, expiraEn: ahora.Add(20 * time.Second)}
	if err := validarTiempos(estado, cfg, ahora); err != nil {
		t.Fatal(err)
	}
	estado.expiraEn = cfg.RetiradaPoliticaAdministracionEn
	if !errors.Is(validarTiempos(estado, cfg, ahora), ErrAsercionNoValida) {
		t.Fatal("sesion hasta retirada admitida")
	}
	estado.expiraEn = ahora.Add(40 * time.Second)
	if !errors.Is(validarTiempos(estado, cfg, ahora), ErrAsercionNoValida) {
		t.Fatal("sesion posterior a retirada admitida")
	}
	if !errors.Is(validarTiempos(estado, cfg, cfg.RetiradaPoliticaAdministracionEn), ErrAsercionNoValida) {
		t.Fatal("politica retirada admitida")
	}
	if !errors.Is(validarTiempos(estado, cfg, cfg.RetiradaPoliticaAdministracionEn.Add(time.Second)), ErrAsercionNoValida) {
		t.Fatal("politica caducada admitida")
	}
}

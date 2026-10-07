package httpseguridad

import (
	"errors"
	"net/netip"
	"testing"
	"time"
)

func TestPoliticaRedAdministracionRetiradaDeniega(t *testing.T) {
	cfg := configuracionAdministracionValida()
	cfg.PoliticaAdministracion = PoliticaAdministracionCertificadoTemporal
	cfg.RetiradaPoliticaAdministracionEn = time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	cfg.RedesPermitidas = []string{"0.0.0.0/0"}
	politica, err := NuevaPoliticaRed(cfg)
	if err != nil {
		t.Fatal(err)
	}
	direccion := netip.MustParseAddr("192.0.2.8")
	if err := politica.Autorizar(direccion); err != nil {
		t.Fatal(err)
	}
	politica.retiradaEn = time.Now().UTC().Add(-time.Second)
	if !errors.Is(politica.Autorizar(direccion), ErrRedNoAutorizada) {
		t.Fatal("politica ADMIN retirada admitida")
	}
}

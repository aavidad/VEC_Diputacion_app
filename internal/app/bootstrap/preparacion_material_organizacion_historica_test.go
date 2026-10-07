package bootstrap

import (
	"bytes"
	"io"
	"path/filepath"
	"testing"
	"time"
)

func TestPrepararClaveOHCoincideConPublicadorExistente(t *testing.T) {
	cfg, rutas := generarMaterialDesarrolloPrueba(t)
	c, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC)
	m, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(c.derivadorIdempotencia, ahora)
	if err != nil {
		t.Fatal(err)
	}
	defer m.borrarCopiasEfimeras()
	cat, _, err := seleccionarMaterialOrganizacionHistorica(configuracionOrganizacionHistoricaPrueba(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var publicado []byte
	_, err = publicarMaterialOrganizacionHistoricaCon(m, cat, func(x *materialAtestacionContratacionTemporalDesarrollo) error {
		publicado = append([]byte(nil), x.claveHMAC...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer clear(publicado)
	k, err := DerivarClaveOrganizacionHistoricaDesdeMaterialDesarrollo(filepath.Dir(rutas.IdempotencyHMACConfig), ahora)
	if err != nil {
		t.Fatal(err)
	}
	defer k.Borrar()
	secreto := k.CopiarSecreto()
	defer clear(secreto)
	if !bytes.Equal(secreto, publicado) || k.DescriptorCapacidadOrganizacionHistoricaV3 != DescriptorCapacidadOrganizacionHistoricaV3Desarrollo() || k.Version != 0 || k.RevisionGobierno != 0 {
		t.Fatal("derivación OH distinta o versiones inventadas")
	}
}

func TestPrepararClaveOHRechazaMaterialAusente(t *testing.T) {
	for _, ruta := range []string{"", "relativo/idempotencia", "/ausente/idempotencia"} {
		k, err := DerivarClaveOrganizacionHistoricaDesdeMaterialDesarrollo(ruta, time.Now())
		if err == nil || len(k.CopiarSecreto()) != 0 {
			t.Fatal("material OH ausente admitido")
		}
	}
}

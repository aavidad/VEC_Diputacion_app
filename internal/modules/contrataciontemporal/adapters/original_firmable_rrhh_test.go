package adapters

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type catalogoTiposOriginalRRHHPrueba struct {
	publicados map[string]string
	reservados map[string]bool
}

func (c *catalogoTiposOriginalRRHHPrueba) TipoDocumentalRef(clave string) (string, error) {
	if referencia := c.publicados[clave]; referencia != "" {
		return referencia, nil
	}
	return "", errors.New("tipo no publicado")
}

func (c *catalogoTiposOriginalRRHHPrueba) CustodiaOriginalCTReservada(referencia string) bool {
	return c.reservados[referencia]
}

func catalogoOriginalRRHHPrueba() *catalogoTiposOriginalRRHHPrueba {
	c := &catalogoTiposOriginalRRHHPrueba{publicados: map[string]string{}, reservados: map[string]bool{}}
	for _, clave := range clavesOriginalFirmableRRHH {
		suma := sha256.Sum256([]byte(clave))
		referencia := "ref:" + hex.EncodeToString(suma[:])
		c.publicados[clave], c.reservados[referencia] = referencia, true
	}
	return c
}

func TestTiposOriginalFirmableRRHHExigeSeisTiposDistintosReservados(t *testing.T) {
	t.Parallel()
	c := catalogoOriginalRRHHPrueba()
	tipos, err := NuevosTiposOriginalFirmableRRHH(c)
	if err != nil {
		t.Fatal(err)
	}
	for tipo, clave := range clavesOriginalFirmableRRHH {
		obtenida, err := tipos.ResolverTipoOriginalRRHH(context.Background(), tipo)
		if err != nil || obtenida != c.publicados[clave] {
			t.Fatalf("tipo %s sin referencia gobernada", tipo)
		}
		paraCustodia, err := tipos.ResolverTipoOriginalCT(context.Background(), string(tipo))
		if err != nil || paraCustodia != obtenida {
			t.Fatalf("custodia no reutilizó el tipo gobernado %s", tipo)
		}
	}
	if _, err := tipos.ResolverTipoOriginalRRHH(context.Background(), ports.BorradorContratoLaboral); !errors.Is(err, ErrTiposOriginalFirmableRRHHNoDisponibles) {
		t.Fatal("se aceptó tipo CT ajeno a los seis originales")
	}
}

func TestTiposOriginalFirmableRRHHCierraCatalogoIncompletoOAlias(t *testing.T) {
	t.Parallel()
	clave := clavesOriginalFirmableRRHH[ports.BorradorResolucion]
	for _, mutar := range []func(*catalogoTiposOriginalRRHHPrueba){
		func(c *catalogoTiposOriginalRRHHPrueba) { delete(c.publicados, clave) },
		func(c *catalogoTiposOriginalRRHHPrueba) { c.reservados[c.publicados[clave]] = false },
		func(c *catalogoTiposOriginalRRHHPrueba) { c.publicados[clave] = "ref:invalid" },
		func(c *catalogoTiposOriginalRRHHPrueba) {
			c.publicados[clave] = c.publicados[clavesOriginalFirmableRRHH[ports.BorradorDiligencia]]
		},
	} {
		c := catalogoOriginalRRHHPrueba()
		mutar(c)
		if _, err := NuevosTiposOriginalFirmableRRHH(c); !errors.Is(err, ErrTiposOriginalFirmableRRHHNoDisponibles) {
			t.Fatal("se construyó la fuente con catálogo ambiguo")
		}
	}
}

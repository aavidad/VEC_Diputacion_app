package bootstrap

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func catalogoIncorporacionB2Prueba(t *testing.T) catalogoMaterialAutorizacionComunDesarrollo {
	t.Helper()
	c, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptoresMaterialIncorporacionB2())
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestPublicacionIncorporacionB2NueveAudienciasIdempotentes(t *testing.T) {
	m := materialRenovableCTPrueba(t, time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC))
	base := append([]byte(nil), m.claveHMAC...)
	baseID, baseAudiencia, baseHuella := m.claveHMACID, m.audienciaConsumo, m.claveHMACHuella
	gobierno := &publicadorGobiernoPrueba{versiones: map[string]uint64{}, siguiente: 40}
	catalogo := catalogoIncorporacionB2Prueba(t)
	for range 2 {
		if err := publicarMaterialIncorporacionB2ConDesarrollo(m, catalogo, gobierno.publicar); err != nil {
			t.Fatal(err)
		}
	}
	if gobierno.llamadas != 20 || len(gobierno.versiones) != 10 || gobierno.siguiente != 50 {
		t.Fatalf("publicación B2 repetida alteró versiones o cardinalidad: llamadas=%d claves=%d siguiente=%d", gobierno.llamadas, len(gobierno.versiones), gobierno.siguiente)
	}
	for _, secreto := range gobierno.secretos {
		if !bytes.Equal(secreto, make([]byte, len(secreto))) {
			t.Fatal("secreto derivado no borrado tras publicar")
		}
	}
	if !bytes.Equal(m.claveHMAC, base) || m.claveHMACID != baseID || m.audienciaConsumo != baseAudiencia || m.claveHMACHuella != baseHuella {
		t.Fatal("la publicación de incorporación alteró el material base")
	}
}

func TestPublicacionIncorporacionB2FallaCerrada(t *testing.T) {
	m := materialRenovableCTPrueba(t, time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC))
	completo := descriptoresMaterialIncorporacionB2()
	for _, alterar := range []func([]descriptorMaterialConsumidorV3Desarrollo) []descriptorMaterialConsumidorV3Desarrollo{
		func(d []descriptorMaterialConsumidorV3Desarrollo) []descriptorMaterialConsumidorV3Desarrollo {
			return d[:8]
		},
		func(d []descriptorMaterialConsumidorV3Desarrollo) []descriptorMaterialConsumidorV3Desarrollo {
			d[6].Dominio = "vec.incorporacion-b2.otro.capacidad-v3"
			return d
		},
	} {
		descriptores := append([]descriptorMaterialConsumidorV3Desarrollo(nil), completo...)
		catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(alterar(descriptores))
		if err != nil {
			t.Fatal(err)
		}
		gobierno := &publicadorGobiernoPrueba{versiones: map[string]uint64{}}
		if err := publicarMaterialIncorporacionB2ConDesarrollo(m, catalogo, gobierno.publicar); !errors.Is(err, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente) || gobierno.llamadas != 0 {
			t.Fatalf("catálogo incompleto o alterado llegó al publicador: %v, %d llamadas", err, gobierno.llamadas)
		}
	}
	if err := publicarMaterialIncorporacionB2ConDesarrollo(m, catalogoIncorporacionB2Prueba(t), nil); !errors.Is(err, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente) {
		t.Fatal("aceptó publicador ausente")
	}
	gobierno := &publicadorGobiernoPrueba{versiones: map[string]uint64{}, fallarEn: 6}
	var secretoFallido []byte
	publicar := func(derivado *materialAtestacionContratacionTemporalDesarrollo) error {
		if gobierno.llamadas == 5 {
			secretoFallido = derivado.claveHMAC
		}
		return gobierno.publicar(derivado)
	}
	if err := publicarMaterialIncorporacionB2ConDesarrollo(m, catalogoIncorporacionB2Prueba(t), publicar); !errors.Is(err, errPostgreSQLContratacionTemporalDesarrolloNoDisponible) || gobierno.llamadas != 6 {
		t.Fatalf("no propagó fallo de publicación: %v, %d llamadas", err, gobierno.llamadas)
	}
	if len(secretoFallido) == 0 || !bytes.Equal(secretoFallido, make([]byte, len(secretoFallido))) {
		t.Fatal("secreto de la publicación fallida retenido")
	}
	for _, secreto := range gobierno.secretos {
		if !bytes.Equal(secreto, make([]byte, len(secreto))) {
			t.Fatal("secreto anterior al fallo retenido")
		}
	}
}

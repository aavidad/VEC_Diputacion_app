package ports

import (
	"encoding/binary"
	"errors"
	"strconv"
	"testing"
)

func manifiestoSinSelloPrueba(t *testing.T) ManifiestoProbatorioBaremacion {
	t.Helper()
	manifiesto := manifiestoProbatorioValidoPrueba(t, contenidoDecisionValidoPrueba(t)).Clonar()
	manifiesto.HuellaManifiestoSHA256 = ""
	manifiesto.SelloManifiestoHMACSHA256 = ""
	return manifiesto
}

func TestManifiestoProbatorioV2RechazaDominioAusenteOAjeno(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*ManifiestoProbatorioBaremacion)
	}{
		{"sin esquema", func(m *ManifiestoProbatorioBaremacion) { m.Esquema = "" }},
		{"esquema ajeno", func(m *ManifiestoProbatorioBaremacion) { m.Esquema = "vec.otro.manifiesto" }},
		{"sin finalidad", func(m *ManifiestoProbatorioBaremacion) { m.Finalidad = "" }},
		{"finalidad ajena", func(m *ManifiestoProbatorioBaremacion) { m.Finalidad = "otra_finalidad" }},
		{"sin version", func(m *ManifiestoProbatorioBaremacion) { m.VersionEsquema = 0 }},
		{"version ajena", func(m *ManifiestoProbatorioBaremacion) { m.VersionEsquema = 1 }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			manifiesto := manifiestoSinSelloPrueba(t)
			caso.mutar(&manifiesto)
			if _, _, err := manifiesto.PrepararSellado(); !errors.Is(err, ErrSolicitudBaremacionInvalida) {
				t.Fatalf("el dominio ausente o ajeno fue admitido: %v", err)
			}
		})
	}
}

func TestManifiestoProbatorioV2AutenticaDominioComoPrimerosCampos(t *testing.T) {
	manifiesto := manifiestoSinSelloPrueba(t)
	_, carga, err := manifiesto.PrepararSellado()
	if err != nil {
		t.Fatalf("preparar manifiesto v2: %v", err)
	}
	material := carga.Revelar()
	leer := func() string {
		t.Helper()
		if len(material) < 8 {
			t.Fatal("material canónico truncado antes de la longitud")
		}
		longitud := binary.BigEndian.Uint64(material[:8])
		material = material[8:]
		if longitud > uint64(len(material)) {
			t.Fatal("material canónico truncado dentro de un campo")
		}
		valor := string(material[:longitud])
		material = material[longitud:]
		return valor
	}
	esperados := []string{
		EsquemaManifiestoProbatorioBaremacion,
		FinalidadManifiestoProbatorioBaremacion,
		strconv.Itoa(VersionManifiestoProbatorioBaremacion),
	}
	for indice, esperado := range esperados {
		if obtenido := leer(); obtenido != esperado {
			t.Fatalf("campo de dominio %d fuera de orden: obtenido=%q esperado=%q", indice, obtenido, esperado)
		}
	}
}

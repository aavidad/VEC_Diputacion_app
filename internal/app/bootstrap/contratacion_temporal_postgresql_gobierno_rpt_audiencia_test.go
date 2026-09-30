package bootstrap

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"testing"
	"time"
)

// Prueba pura de selección del publicador y su puntero. El doble no demuestra
// gobierno instalado, una transacción PostgreSQL ni un positivo de consumo V3.
func TestGobiernoRPTUsosMantieneListaNominalYReconocePuntero(t *testing.T) {
	const audiencia = "vec_catalogos_configurables.usos_categorias.v1"
	lista := audienciasConsumoGobiernoCTDesarrollo()
	if !audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(audiencia) ||
		!slices.Contains(lista, audienciaConsumoAltaContratacionTemporal) {
		t.Fatal("la lista no conserva CT y el consumidor RPT nominal")
	}
	for _, valor := range lista {
		if strings.Contains(valor, "*") {
			t.Fatal("la lista contiene un comodín")
		}
	}
	for _, caso := range []struct {
		audiencia string
		propia    bool
	}{
		{audiencia, true},
		{audienciaConsumoAltaContratacionTemporal, true},
		{"vec_catalogos_configurables.usos_categorias.v2", false},
		{"vec_catalogos_configurables.usos_categorias.*", false},
		{"vec_catalogos_configurables.lectura_categorias.v1", false},
	} {
		propio, err := gobiernoActualPostgreSQLContratacionTemporalDesarrolloEsPropio(
			context.Background(), &txGobiernoContinuidadPrueba{audienciaActual: caso.audiencia})
		if err != nil || propio != caso.propia ||
			audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(caso.audiencia) != caso.propia {
			t.Fatalf("selección nominal de %q: propio=%v, error=%v", caso.audiencia, propio, err)
		}
	}
}

func TestGobiernoRPTUsosDerivaDescriptorSeparadoDeCT(t *testing.T) {
	// Vector puramente sintético en memoria: no genera CA ni lee claves.
	base := materialAtestacionContratacionTemporalDesarrollo{
		claveHMAC: bytes.Repeat([]byte{7}, 32), claveHMACID: "clave:capacidad:ct:desarrollo:v1",
		claveHMACVersion: 1, claveHMACRevision: 1, emisorID: "emisor:rpt:fixture",
		validaDesde: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), validaHasta: time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC),
		claveHMACSecreto: strings.Repeat("a", 64), claveHMACHuella: strings.Repeat("b", 64), spkiHuella: strings.Repeat("c", 64),
	}
	defer base.borrarCopiasEfimeras()
	d := descriptorMaterialRPTUsosFixture()
	material, err := derivarMaterialConsumidorV3Desarrollo(base, d)
	if err != nil {
		t.Fatal(err)
	}
	defer borrarBytes(material.claveHMAC)
	if material.audienciaConsumo != audienciaUsosCategoriasRPTFixture ||
		material.claveHMACID == base.claveHMACID || !strings.HasPrefix(material.claveHMACID, d.Prefijo) ||
		material.claveHMACSecreto == base.claveHMACSecreto ||
		material.claveHMACHuella == base.claveHMACHuella || material.spkiHuella != base.spkiHuella {
		t.Fatal("el descriptor RPT no separa consumo y conserva la raíz común")
	}
}

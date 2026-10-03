package config

import (
	"testing"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestPlanCompetenciaFirmaV2NoConfiguraAutoridadPorDefecto(t *testing.T) {
	if fuente, err := PrepararPlanCompetenciaFirmaV2(nil, nil, ct.VersionPlanFirmaV2{}); err == nil || fuente != nil {
		t.Fatal("plan sin publicacion acreditada")
	}
}

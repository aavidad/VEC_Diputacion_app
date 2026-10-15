package ports

import (
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
)

func TestCriteriosAplicadosConservanLaPublicacionOriginal(t *testing.T) {
	original := &CriteriosOfrecimientoAplicados{
		VersionPoliticaOfertas: 2, HuellaPoliticaOfertasSHA256: strings.Repeat("a", 64), ModalidadOfrecida: "vacante", CausaOfrecida: "vacante",
		Configuracion: domain.CriteriosOfrecimiento{
			FuenteNormativa:       "Reglamento de bolsas, art. 9.4",
			ModalidadesAceptables: []string{"vacante", "programa"},
			Mejora: &domain.ReglaMejoraOfrecimiento{Habilitada: true, ModalidadesDestino: []string{"vacante"},
				CausasDestino:      []string{"vacante"},
				MaximoVacanteMeses: 36, ReduccionPorModalidadesPrevias: []string{"programa"},
				CondicionesActuales: []domain.CondicionRelacionActual{{Modalidad: "programa",
					MesesDuracionInicial: &domain.LimiteMeses{Meses: 9}}}},
		},
	}
	if err := original.Validar(); err != nil {
		t.Fatal(err)
	}
	copia := original.Clonar()
	if !copia.Iguales(original) {
		t.Fatal("el clon debe conservar todos los criterios")
	}
	copia.Configuracion.ModalidadesAceptables[0] = "programa"
	copia.Configuracion.Mejora.ModalidadesDestino[0] = "programa"
	copia.Configuracion.Mejora.CausasDestino[0] = "sustitucion"
	copia.Configuracion.Mejora.ReduccionPorModalidadesPrevias[0] = "sustitucion"
	copia.Configuracion.Mejora.CondicionesActuales[0].MesesDuracionInicial.Meses = 8
	if original.Configuracion.ModalidadesAceptables[0] != "vacante" ||
		original.Configuracion.Mejora.ModalidadesDestino[0] != "vacante" ||
		original.Configuracion.Mejora.CausasDestino[0] != "vacante" ||
		original.Configuracion.Mejora.ReduccionPorModalidadesPrevias[0] != "programa" ||
		original.Configuracion.Mejora.CondicionesActuales[0].MesesDuracionInicial.Meses != 9 || copia.Iguales(original) {
		t.Fatal("modificar el borrador no puede modificar el criterio original")
	}
	original.HuellaPoliticaOfertasSHA256 = strings.Repeat("Z", 64)
	if original.Validar() == nil {
		t.Fatal("una huella ajena al formato debe rechazarse")
	}
	var legado *CriteriosOfrecimientoAplicados
	if legado.Validar() != nil || legado.Clonar() != nil || !legado.Iguales(nil) || legado.Iguales(original) {
		t.Fatal("las lecturas históricas sin criterio conservan su ausencia")
	}
}

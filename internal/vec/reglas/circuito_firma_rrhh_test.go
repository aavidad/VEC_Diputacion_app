package reglas

import (
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/fichero"
)

// El catálogo confirmado conserva el anterior y lo consumen el mismo
// resolutor y la misma consulta de firmas, sin ampliar permisos.
func TestCircuitoFirmaRRHHVersionadoConservaRemisionYOrden(t *testing.T) {
	consulta, err := fichero.NuevaConsultaCatalogos("../../../data/demo/reglas/ct_circuito_firma.rrhh.v2.json")
	if err != nil {
		t.Fatal(err)
	}
	resolutor, err := NuevoResolutor(Configuracion{
		Consulta: consulta, Metadatos: consulta, CatalogoID: CatalogoCircuitoFirmaCT,
		ModuloID: ModuloContratacionTemporal, Reloj: relojFijo(time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)),
	})
	if err != nil {
		t.Fatal(err)
	}
	circuito, err := resolutor.CircuitoFirma(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if circuito.Version != 2 || len(circuito.Documentos) != 2 {
		t.Fatalf("circuito RRHH inesperado: %+v", circuito)
	}
	informe, resolucion := circuito.Documentos[0], circuito.Documentos[1]
	if informe.Documento != "informe_definitivo" || len(informe.Pasos) != 1 ||
		informe.Pasos[0].PerfilRef != "perfil:ct:jefatura_servicio_rrhh" ||
		informe.Pasos[0].Habilita != HabilitaRemisionIntervencion {
		t.Fatalf("la Jefatura debe firmar el informe para remitirlo: %+v", informe)
	}
	if resolucion.Documento != "resolucion" || len(resolucion.Pasos) != 2 ||
		resolucion.Pasos[0].PerfilRef != "perfil:ct:jefatura_servicio_rrhh" ||
		resolucion.Pasos[0].Accion != AccionFirmaVistoBueno ||
		resolucion.Pasos[1].PerfilRef != "perfil:ct:diputacion_delegada_rrhh" ||
		resolucion.Pasos[1].Accion != AccionFirmaFirma ||
		resolucion.Pasos[1].Habilita != HabilitaCierreCircuito {
		t.Fatalf("la resolución exige visto bueno y firma de la Diputada: %+v", resolucion)
	}
	anterior := resolutorReal(t, rutaCircuitoFirmaCTPrueba, CatalogoCircuitoFirmaCT, ModuloContratacionTemporal, nil)
	legado, err := anterior.CircuitoFirma(t.Context())
	if err != nil || legado.Version != 1 || legado.HuellaCatalogo == circuito.HuellaCatalogo ||
		len(legado.Documentos[0].Pasos) != 2 || len(legado.Documentos[1].Pasos) != 3 {
		t.Fatalf("el catálogo histórico debe permanecer distinto e intacto: %+v, %v", legado, err)
	}
}

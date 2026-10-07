package reglas

import (
	"bytes"
	"maps"
	"os"
	"path/filepath"
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
	if circuito.Version != 2 || len(circuito.Documentos) != 2 || circuito.PermiteMismaPersonaEnPasos ||
		circuito.HuellaCatalogo != "aab22b0b8fe45e15f7c592f71755cc3d9f35530ffe9c30bce8bd2e59bc8fef38" {
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
		len(resolucion.Pasos[0].PerfilesAlternativos) != 1 ||
		resolucion.Pasos[0].PerfilesAlternativos[0] != "perfil:ct:direccion_rrhh" ||
		resolucion.Pasos[0].Accion != AccionFirmaVistoBueno ||
		resolucion.Pasos[1].PerfilRef != "perfil:ct:diputacion_delegada_rrhh" ||
		resolucion.Pasos[1].Accion != AccionFirmaFirma ||
		resolucion.Pasos[1].Habilita != HabilitaCierreCircuito {
		t.Fatalf("la resolución exige visto bueno y firma de la Diputada: %+v", resolucion)
	}
	anterior := resolutorReal(t, rutaCircuitoFirmaCTPrueba, CatalogoCircuitoFirmaCT, ModuloContratacionTemporal, nil)
	legado, err := anterior.CircuitoFirma(t.Context())
	if err != nil || legado.Version != 1 || legado.PermiteMismaPersonaEnPasos || legado.HuellaCatalogo == circuito.HuellaCatalogo ||
		len(legado.Documentos[0].Pasos) != 2 || len(legado.Documentos[1].Pasos) != 3 {
		t.Fatalf("el catálogo histórico debe permanecer distinto e intacto: %+v, %v", legado, err)
	}
}

func TestCircuitoFirmaRRHHPoliticaMismaPersonaVersionada(t *testing.T) {
	consulta, err := fichero.NuevaConsultaCatalogos("../../../data/demo/reglas/ct_circuito_firma.rrhh.v2.json")
	if err != nil {
		t.Fatal(err)
	}
	r, err := NuevoResolutor(Configuracion{Consulta: consulta, Metadatos: consulta,
		CatalogoID: CatalogoCircuitoFirmaCT, ModuloID: ModuloContratacionTemporal,
		Reloj: relojFijo(time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC))})
	if err != nil {
		t.Fatal(err)
	}
	reglas, err := r.Reglas(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	base, err := r.CircuitoFirma(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	conValor := func(valor string) []Regla {
		copia := append([]Regla(nil), reglas...)
		for i := range copia {
			copia[i].Atributos = maps.Clone(reglas[i].Atributos)
			copia[i].Atributos["misma_persona_en_dos_pasos"] = valor
		}
		return copia
	}
	permitida, err := CircuitoFirmaDesdeReglas(conValor("true"))
	if err != nil || !permitida.PermiteMismaPersonaEnPasos || len(permitida.Documentos) != 2 {
		t.Fatalf("la opción explícita de la versión no se resolvió: %+v, %v", permitida, err)
	}
	contenido, err := os.ReadFile("../../../data/demo/reglas/ct_circuito_firma.rrhh.v2.json")
	if err != nil {
		t.Fatal(err)
	}
	declaracion := []byte(`"misma_persona_en_dos_pasos": "false"`)
	if bytes.Count(contenido, declaracion) != 3 {
		t.Fatal("la política de la versión debe constar en todos sus pasos")
	}
	ruta := filepath.Join(t.TempDir(), "catalogo.json")
	if err := os.WriteFile(ruta, bytes.ReplaceAll(contenido, declaracion, []byte(`"misma_persona_en_dos_pasos": "true"`)), 0600); err != nil {
		t.Fatal(err)
	}
	configurada, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		t.Fatal(err)
	}
	resolutorSi, err := NuevoResolutor(Configuracion{Consulta: configurada, Metadatos: configurada,
		CatalogoID: CatalogoCircuitoFirmaCT, ModuloID: ModuloContratacionTemporal,
		Reloj: relojFijo(time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC))})
	if err != nil {
		t.Fatal(err)
	}
	conSi, err := resolutorSi.CircuitoFirma(t.Context())
	if err != nil || !conSi.PermiteMismaPersonaEnPasos || conSi.HuellaCatalogo == base.HuellaCatalogo {
		t.Fatalf("una política distinta requiere otra huella de catálogo: %+v, %v", conSi, err)
	}
	for _, valor := range []string{"", "si", "1", "TRUE", "True"} {
		if _, err := CircuitoFirmaDesdeReglas(conValor(valor)); err == nil {
			t.Fatalf("aceptó valor no canónico %q", valor)
		}
	}
	parcial := conValor("true")
	delete(parcial[1].Atributos, "misma_persona_en_dos_pasos")
	if _, err := CircuitoFirmaDesdeReglas(parcial); err == nil {
		t.Fatal("aceptó catálogo con política presente solo en parte de los pasos")
	}
	mixta := conValor("true")
	mixta[1].Atributos["misma_persona_en_dos_pasos"] = "false"
	if _, err := CircuitoFirmaDesdeReglas(mixta); err == nil {
		t.Fatal("aceptó políticas contradictorias en un catálogo")
	}
}

func TestCircuitoFirmaRRHHRechazaAlternativasInvalidas(t *testing.T) {
	consulta, err := fichero.NuevaConsultaCatalogos("../../../data/demo/reglas/ct_circuito_firma.rrhh.v2.json")
	if err != nil {
		t.Fatal(err)
	}
	r, err := NuevoResolutor(Configuracion{Consulta: consulta, Metadatos: consulta,
		CatalogoID: CatalogoCircuitoFirmaCT, ModuloID: ModuloContratacionTemporal,
		Reloj: relojFijo(time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC))})
	if err != nil {
		t.Fatal(err)
	}
	reglas, err := r.Reglas(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, valor := range []string{
		"",                                 // un campo declarado no puede quedar sin opción
		"perfil:ct:jefatura_servicio_rrhh", // repite el perfil principal
		"perfil:ct:direccion_rrhh,perfil:ct:direccion_rrhh",
		"perfil:ct:direccion_rrhh,", "Dirección de RRHH",
	} {
		copia := append([]Regla(nil), reglas...)
		copia[1].Atributos = make(map[string]string, len(reglas[1].Atributos))
		for k, v := range reglas[1].Atributos {
			copia[1].Atributos[k] = v
		}
		copia[1].Atributos["perfiles_ref_alternativos"] = valor
		if _, err := CircuitoFirmaDesdeReglas(copia); err == nil {
			t.Fatalf("se aceptó alternativa inválida %q", valor)
		}
	}
}

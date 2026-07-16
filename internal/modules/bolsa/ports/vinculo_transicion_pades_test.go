package ports

import (
	"strings"
	"testing"
	"time"
)

func cadenaVinculosPAdESPrueba(
	t *testing.T,
) (SelloTiempoFirma, ValidacionFirmaServidor, ResultadoAumentoFirma, ValidacionFirmaServidor) {
	t.Helper()
	_, sello := solicitudSelloTiempoValidaAdversarial(t)
	validacionT := validacionFirmaValidaPrueba(
		sello.ArtefactoSellado,
		instantePuertosPrueba.Add(8*time.Minute),
		"vinculo-t",
		PerfilFirmaPAdESBaselineT,
	)
	artefactoLTA := sello.ArtefactoSellado
	artefactoLTA.DocumentoFirmadoRef += ":pades-lta"
	artefactoLTA.HuellaDocumentoSHA256 = huellaPruebaPuertos("4")
	politica := politicaFirmaValidaPrueba()
	aumento := ResultadoAumentoFirma{
		ArtefactoOrigen: sello.ArtefactoSellado, Artefacto: artefactoLTA,
		NivelAlcanzadoClave:            politica.NivelAumentoClave,
		PoliticaLongevidadRef:          politica.PoliticaLongevidadRef,
		PoliticaLongevidadVersion:      politica.PoliticaLongevidadVersion,
		HuellaPoliticaLongevidadSHA256: politica.HuellaPoliticaLongevidadSHA256,
		EvidenciaAumentoRef:            "evidencia-aumento-1",
		HuellaEvidenciaSHA256:          huellaPruebaPuertos("6"),
		AumentadaEn:                    instantePuertosPrueba.Add(9 * time.Minute),
	}
	validacionLTA := validacionFirmaValidaPrueba(
		artefactoLTA,
		instantePuertosPrueba.Add(10*time.Minute),
		"vinculo-lta",
		PerfilFirmaPAdESBaselineLTA,
	)
	if sello.Validar() != nil || validacionT.Validar() != nil || aumento.Validar() != nil || validacionLTA.Validar() != nil {
		t.Fatal("precondicion: cadena PAdES de prueba invalida")
	}
	return sello, validacionT, aumento, validacionLTA
}

func TestVinculosTransicionPAdESSonDeterministasYDerivanReferencia(t *testing.T) {
	sello, validacionT, aumento, validacionLTA := cadenaVinculosPAdESPrueba(t)
	vinculoT, err := NuevoVinculoRevisionSelladaPAdES(sello, validacionT)
	if err != nil {
		t.Fatalf("vinculo B-T: %v", err)
	}
	repetidoT, err := NuevoVinculoRevisionSelladaPAdES(sello, validacionT)
	if err != nil || repetidoT != vinculoT ||
		vinculoT.Referencia != prefijoVinculoRevisionSellada+vinculoT.HuellaSHA256 {
		t.Fatalf("vinculo B-T no determinista o referencia no derivada: %+v / %+v", vinculoT, repetidoT)
	}
	vinculoLTA, err := NuevoVinculoRevisionLongevaPAdES(sello, validacionT, aumento, validacionLTA)
	if err != nil {
		t.Fatalf("vinculo T-LTA: %v", err)
	}
	repetidoLTA, err := NuevoVinculoRevisionLongevaPAdES(sello, validacionT, aumento, validacionLTA)
	if err != nil || repetidoLTA != vinculoLTA ||
		vinculoLTA.Referencia != prefijoVinculoRevisionLongeva+vinculoLTA.HuellaSHA256 ||
		vinculoLTA == vinculoT {
		t.Fatalf("vinculo T-LTA no determinista o no separado por tipo: %+v / %+v", vinculoLTA, repetidoLTA)
	}
}

func TestVinculoRevisionSelladaComprometeTodoElMaterial(t *testing.T) {
	selloBase, validacionBase, _, _ := cadenaVinculosPAdESPrueba(t)
	original, err := NuevoVinculoRevisionSelladaPAdES(selloBase, validacionBase)
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre string
		mutar  func(*SelloTiempoFirma, *ValidacionFirmaServidor)
	}{
		{"origen", func(s *SelloTiempoFirma, _ *ValidacionFirmaServidor) {
			s.ArtefactoOrigen.DocumentoFirmadoRef += ":otro"
		}},
		{"destino", func(s *SelloTiempoFirma, v *ValidacionFirmaServidor) {
			s.ArtefactoSellado.DocumentoFirmadoRef += ":otro"
			v.Artefacto = s.ArtefactoSellado
		}},
		{"token", func(s *SelloTiempoFirma, v *ValidacionFirmaServidor) {
			s.HuellaSelloTiempoSHA256 = huellaPruebaPuertos("5")
			v.HuellaSelloTiempoVerificadaSHA256 = s.HuellaSelloTiempoSHA256
		}},
		{"politica", func(s *SelloTiempoFirma, _ *ValidacionFirmaServidor) { s.PoliticaSelloTiempoRef += ":otra" }},
		{"fecha", func(s *SelloTiempoFirma, _ *ValidacionFirmaServidor) { s.SelladoEn = s.SelladoEn.Add(time.Nanosecond) }},
		{"atestacion", func(_ *SelloTiempoFirma, v *ValidacionFirmaServidor) { v.ValidacionRef += ":otra" }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			sello, validacion := selloBase, validacionBase
			caso.mutar(&sello, &validacion)
			mutado, err := NuevoVinculoRevisionSelladaPAdES(sello, validacion)
			if err == nil && mutado == original {
				t.Fatal("la mutacion no cambio ni invalido el vinculo B-T")
			}
		})
	}
}

func TestVinculoRevisionLongevaComprometeTodoElMaterial(t *testing.T) {
	selloBase, validacionTBase, aumentoBase, validacionBase := cadenaVinculosPAdESPrueba(t)
	original, err := NuevoVinculoRevisionLongevaPAdES(selloBase, validacionTBase, aumentoBase, validacionBase)
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre string
		mutar  func(*SelloTiempoFirma, *ValidacionFirmaServidor, *ResultadoAumentoFirma, *ValidacionFirmaServidor)
	}{
		{"origen", func(_ *SelloTiempoFirma, _ *ValidacionFirmaServidor, a *ResultadoAumentoFirma, _ *ValidacionFirmaServidor) {
			a.ArtefactoOrigen.DocumentoFirmadoRef += ":otro"
		}},
		{"destino", func(_ *SelloTiempoFirma, _ *ValidacionFirmaServidor, a *ResultadoAumentoFirma, v *ValidacionFirmaServidor) {
			a.Artefacto.DocumentoFirmadoRef += ":otro"
			v.Artefacto = a.Artefacto
		}},
		{"evidencia", func(_ *SelloTiempoFirma, _ *ValidacionFirmaServidor, a *ResultadoAumentoFirma, v *ValidacionFirmaServidor) {
			a.HuellaEvidenciaSHA256 = huellaPruebaPuertos("5")
			v.HuellaAumentoLongevidadVerificadaSHA256 = a.HuellaEvidenciaSHA256
		}},
		{"politica", func(_ *SelloTiempoFirma, _ *ValidacionFirmaServidor, a *ResultadoAumentoFirma, _ *ValidacionFirmaServidor) {
			a.PoliticaLongevidadRef += ":otra"
		}},
		{"fecha", func(_ *SelloTiempoFirma, _ *ValidacionFirmaServidor, a *ResultadoAumentoFirma, _ *ValidacionFirmaServidor) {
			a.AumentadaEn = a.AumentadaEn.Add(time.Nanosecond)
		}},
		{"atestacion T", func(_ *SelloTiempoFirma, vT *ValidacionFirmaServidor, _ *ResultadoAumentoFirma, _ *ValidacionFirmaServidor) {
			vT.ValidacionRef += ":otra"
		}},
		{"atestacion LTA", func(_ *SelloTiempoFirma, _ *ValidacionFirmaServidor, _ *ResultadoAumentoFirma, v *ValidacionFirmaServidor) {
			v.ValidacionRef += ":otra"
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			sello, validacionT, aumento, validacion := selloBase, validacionTBase, aumentoBase, validacionBase
			caso.mutar(&sello, &validacionT, &aumento, &validacion)
			mutado, err := NuevoVinculoRevisionLongevaPAdES(sello, validacionT, aumento, validacion)
			if err == nil && mutado == original {
				t.Fatal("la mutacion no cambio ni invalido el vinculo T-LTA")
			}
		})
	}
}

func TestVinculoRevisionSelladaRechazaAtestacionDeOtroSello(t *testing.T) {
	sello, validacionT, _, _ := cadenaVinculosPAdESPrueba(t)
	validacionT.SelloTiempoVerificadoRef = "sello-tiempo-ajeno"
	validacionT.HuellaSelloTiempoVerificadaSHA256 = strings.Repeat("a", 64)
	if validacionT.Validar() != nil {
		t.Fatal("precondicion: la atestacion ajena debe ser valida aisladamente")
	}
	if _, err := NuevoVinculoRevisionSelladaPAdES(sello, validacionT); err == nil {
		t.Fatal("se enlazo una revision T con la atestacion de otro sello valido")
	}
}

func TestVinculoRevisionLongevaRechazaAtestacionDeOtroAumento(t *testing.T) {
	sello, validacionT, aumento, validacionLTA := cadenaVinculosPAdESPrueba(t)
	validacionLTA.AumentoLongevidadVerificadoRef = "evidencia-aumento-ajena"
	validacionLTA.HuellaAumentoLongevidadVerificadaSHA256 = strings.Repeat("a", 64)
	if validacionLTA.Validar() != nil {
		t.Fatal("precondicion: la atestacion ajena debe ser valida aisladamente")
	}
	if _, err := NuevoVinculoRevisionLongevaPAdES(sello, validacionT, aumento, validacionLTA); err == nil {
		t.Fatal("se enlazo una revision LTA con la atestacion de otro aumento valido")
	}
}

func TestVinculosPAdESCanonizanOrdenDeComprobaciones(t *testing.T) {
	sello, validacionT, aumento, validacionLTA := cadenaVinculosPAdESPrueba(t)
	originalT, err := NuevoVinculoRevisionSelladaPAdES(sello, validacionT)
	if err != nil {
		t.Fatal(err)
	}
	originalLTA, err := NuevoVinculoRevisionLongevaPAdES(sello, validacionT, aumento, validacionLTA)
	if err != nil {
		t.Fatal(err)
	}
	invertirComprobaciones := func(v *ValidacionFirmaServidor) {
		for izquierda, derecha := 0, len(v.Comprobaciones)-1; izquierda < derecha; izquierda, derecha = izquierda+1, derecha-1 {
			v.Comprobaciones[izquierda], v.Comprobaciones[derecha] = v.Comprobaciones[derecha], v.Comprobaciones[izquierda]
		}
	}
	invertirComprobaciones(&validacionT)
	invertirComprobaciones(&validacionLTA)
	reordenadoT, err := NuevoVinculoRevisionSelladaPAdES(sello, validacionT)
	if err != nil || reordenadoT != originalT {
		t.Fatalf("el orden de entrada altero el vinculo B-T: original=%+v reordenado=%+v err=%v", originalT, reordenadoT, err)
	}
	reordenadoLTA, err := NuevoVinculoRevisionLongevaPAdES(sello, validacionT, aumento, validacionLTA)
	if err != nil || reordenadoLTA != originalLTA {
		t.Fatalf("el orden de entrada altero el vinculo T-LTA: original=%+v reordenado=%+v err=%v", originalLTA, reordenadoLTA, err)
	}
}

func TestVinculoPAdESComprometeAtestacionCompletaYExigeApta(t *testing.T) {
	sello, validacionT, _, _ := cadenaVinculosPAdESPrueba(t)
	original, err := NuevoVinculoRevisionSelladaPAdES(sello, validacionT)
	if err != nil {
		t.Fatal(err)
	}
	mutada := validacionT
	mutada.Comprobaciones = append([]ComprobacionFirma(nil), validacionT.Comprobaciones...)
	mutada.Comprobaciones[0].EvidenciaRef += ":otra"
	vinculoMutado, err := NuevoVinculoRevisionSelladaPAdES(sello, mutada)
	if err != nil || vinculoMutado == original {
		t.Fatalf("la evidencia de comprobacion no quedo comprometida: original=%+v mutado=%+v err=%v", original, vinculoMutado, err)
	}

	estadoInvalido := validacionT
	estadoInvalido.Estado = EstadoValidacionFirmaInvalida
	if estadoInvalido.Validar() != nil {
		t.Fatal("precondicion: estado invalido es una respuesta estructuralmente valida")
	}
	if _, err := NuevoVinculoRevisionSelladaPAdES(sello, estadoInvalido); err == nil {
		t.Fatal("se creo un vinculo desde una atestacion no apta")
	}

	comprobacionFallida := validacionT
	comprobacionFallida.Comprobaciones = append([]ComprobacionFirma(nil), validacionT.Comprobaciones...)
	comprobacionFallida.Comprobaciones[0].Estado = EstadoComprobacionNoSuperada
	if comprobacionFallida.Validar() != nil {
		t.Fatal("precondicion: comprobacion no superada es estructuralmente valida")
	}
	if _, err := NuevoVinculoRevisionSelladaPAdES(sello, comprobacionFallida); err == nil {
		t.Fatal("se creo un vinculo desde una comprobacion no superada")
	}
}

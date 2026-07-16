package ports

import (
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

type cadenaManifiestoVinculoPAdESPrueba struct {
	contenido           dominiobolsa.ContenidoDecisionTecnica
	politica            PoliticaFirmaBaremacion
	artefacto           ArtefactoFirma
	validacionInicial   ValidacionFirmaServidor
	sello               SelloTiempoFirma
	validacionTrasSello ValidacionFirmaServidor
	aumento             ResultadoAumentoFirma
	validacionFinal     ValidacionFirmaServidor
	documento           DocumentoFirmadoCustodiado
	manifiesto          ManifiestoProbatorioBaremacion
}

func evidenciaManifiestoPorTipoPrueba(
	t *testing.T,
	manifiesto *ManifiestoProbatorioBaremacion,
	tipo TipoEvidenciaProbatoriaBaremacion,
) *EvidenciaProbatoriaBaremacion {
	t.Helper()
	var encontrada *EvidenciaProbatoriaBaremacion
	for indice := range manifiesto.Evidencias {
		if manifiesto.Evidencias[indice].Tipo != tipo {
			continue
		}
		if encontrada != nil {
			t.Fatalf("manifiesto con evidencia repetida de tipo %s", tipo)
		}
		encontrada = &manifiesto.Evidencias[indice]
	}
	if encontrada == nil {
		t.Fatalf("manifiesto sin evidencia de tipo %s", tipo)
	}
	return encontrada
}

func resecuenciarManifiestoPrueba(manifiesto *ManifiestoProbatorioBaremacion) {
	if manifiesto == nil {
		return
	}
	for indice := range manifiesto.Autorizaciones {
		manifiesto.Autorizaciones[indice].Secuencia = uint32(indice + 1)
	}
	for indice := range manifiesto.Evidencias {
		manifiesto.Evidencias[indice].Secuencia = uint32(indice + 1)
	}
}

func resellarManifiestoPrueba(
	manifiesto ManifiestoProbatorioBaremacion,
) (ManifiestoProbatorioBaremacion, error) {
	manifiesto.HuellaManifiestoSHA256 = ""
	manifiesto.SelloManifiestoHMACSHA256 = ""
	preparado, _, err := manifiesto.PrepararSellado()
	if err != nil {
		return ManifiestoProbatorioBaremacion{}, err
	}
	return preparado.IncorporarSello("hmac-sha256:manifiesto_1:" + huellaPruebaPuertos("3"))
}

func agregarEvidenciasVinculoSelloPrueba(
	t *testing.T,
	evidencias []EvidenciaProbatoriaBaremacion,
	sello SelloTiempoFirma,
	validacion *ValidacionFirmaServidor,
) []EvidenciaProbatoriaBaremacion {
	t.Helper()
	if validacion == nil {
		t.Fatal("sello de prueba sin validacion PAdES-T")
	}
	vinculo, err := NuevoVinculoRevisionSelladaPAdES(sello, *validacion)
	if err != nil {
		t.Fatalf("vinculo B-T de prueba: %v", err)
	}
	return append(evidencias,
		EvidenciaProbatoriaBaremacion{
			Tipo: EvidenciaSelloTiempoBaremacion, Referencia: sello.SelloTiempoRef,
			HuellaEvidenciaSHA256: sello.HuellaSelloTiempoSHA256,
		},
		EvidenciaProbatoriaBaremacion{
			Tipo:       EvidenciaVinculoRevisionSelladaBaremacion,
			Referencia: vinculo.Referencia, HuellaEvidenciaSHA256: vinculo.HuellaSHA256,
		},
	)
}

func agregarEvidenciaVinculoAumentoPrueba(
	t *testing.T,
	evidencias []EvidenciaProbatoriaBaremacion,
	sello SelloTiempoFirma,
	validacionT ValidacionFirmaServidor,
	aumento ResultadoAumentoFirma,
	validacion ValidacionFirmaServidor,
) []EvidenciaProbatoriaBaremacion {
	t.Helper()
	vinculo, err := NuevoVinculoRevisionLongevaPAdES(sello, validacionT, aumento, validacion)
	if err != nil {
		t.Fatalf("vinculo T-LTA de prueba: %v", err)
	}
	return append(evidencias, EvidenciaProbatoriaBaremacion{
		Tipo:       EvidenciaVinculoRevisionLongevaBaremacion,
		Referencia: vinculo.Referencia, HuellaEvidenciaSHA256: vinculo.HuellaSHA256,
	})
}

func nuevaCadenaManifiestoVinculoPAdESPrueba(t *testing.T) cadenaManifiestoVinculoPAdESPrueba {
	t.Helper()
	contenido := contenidoDecisionValidoPrueba(t)
	politica := politicaFirmaValidaPrueba()
	artefacto := artefactoFirmaValidoPrueba(t, contenido, politica, "vinculo-inicial")
	validacionInicial := validacionFirmaValidaPrueba(
		artefacto, instantePuertosPrueba.Add(6*time.Minute), "vinculo-inicial",
	)
	sello := selloValidoPrueba(artefacto, politica)
	validacionT := validacionFirmaValidaPrueba(
		sello.ArtefactoSellado, instantePuertosPrueba.Add(8*time.Minute),
		"vinculo-sellado", PerfilFirmaPAdESBaselineT,
	)
	artefactoLTA := sello.ArtefactoSellado
	artefactoLTA.DocumentoFirmadoRef += ":pades-lta"
	artefactoLTA.HuellaDocumentoSHA256 = huellaPruebaPuertos("4")
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
	validacionFinal := validacionFirmaValidaPrueba(
		artefactoLTA, instantePuertosPrueba.Add(10*time.Minute),
		"vinculo-final", PerfilFirmaPAdESBaselineLTA,
	)
	documento := documentoFirmadoCustodiadoPrueba(artefactoLTA)
	manifiesto := construirManifiestoProbatorioPrueba(
		t, contenido, politica, artefacto, validacionInicial, &sello, &validacionT,
		&aumento, validacionFinal, documento,
	)
	return cadenaManifiestoVinculoPAdESPrueba{
		contenido: contenido, politica: politica, artefacto: artefacto,
		validacionInicial: validacionInicial, sello: sello,
		validacionTrasSello: validacionT, aumento: aumento,
		validacionFinal: validacionFinal, documento: documento, manifiesto: manifiesto,
	}
}

func TestCoberturaManifiestoRechazaVinculosValidosMezclados(t *testing.T) {
	cadena := nuevaCadenaManifiestoVinculoPAdESPrueba(t)
	casos := []struct {
		nombre string
		tipo   TipoEvidenciaProbatoriaBaremacion
		ajeno  func() VinculoTransicionPAdES
	}{
		{
			nombre: "vinculo B-T de otra atestacion",
			tipo:   EvidenciaVinculoRevisionSelladaBaremacion,
			ajeno: func() VinculoTransicionPAdES {
				validacion := cadena.validacionTrasSello
				validacion.ValidacionRef += ":ajena"
				vinculo, err := NuevoVinculoRevisionSelladaPAdES(cadena.sello, validacion)
				if err != nil {
					t.Fatalf("vinculo B-T ajeno valido: %v", err)
				}
				return vinculo
			},
		},
		{
			nombre: "vinculo T-LTA de otra atestacion",
			tipo:   EvidenciaVinculoRevisionLongevaBaremacion,
			ajeno: func() VinculoTransicionPAdES {
				validacion := cadena.validacionFinal
				validacion.ValidacionRef += ":ajena"
				vinculo, err := NuevoVinculoRevisionLongevaPAdES(
					cadena.sello, cadena.validacionTrasSello, cadena.aumento, validacion,
				)
				if err != nil {
					t.Fatalf("vinculo T-LTA ajeno valido: %v", err)
				}
				return vinculo
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			mezclado := cadena.manifiesto.Clonar()
			ajeno := caso.ajeno()
			encontrado := false
			for indice := range mezclado.Evidencias {
				if mezclado.Evidencias[indice].Tipo == caso.tipo {
					mezclado.Evidencias[indice].Referencia = ajeno.Referencia
					mezclado.Evidencias[indice].HuellaEvidenciaSHA256 = ajeno.HuellaSHA256
					encontrado = true
					break
				}
			}
			if !encontrado {
				t.Fatal("manifiesto de prueba sin la hoja esperada")
			}
			mezclado, err := resellarManifiestoPrueba(mezclado)
			if err != nil {
				t.Fatalf("resellar manifiesto mezclado: %v", err)
			}
			if err := mezclado.validarCoberturaArtefactosFirmaPara(
				cadena.politica, cadena.artefacto, cadena.validacionInicial,
				&cadena.sello, &cadena.validacionTrasSello, &cadena.aumento,
				cadena.validacionFinal, cadena.documento,
			); err == nil {
				t.Fatal("un manifiesto HMAC con una transicion valida pero ajena supero la cobertura")
			}
		})
	}
}

func TestMismaEvidenciaValidacionFirmaIncluyeEvidenciasEmbebidas(t *testing.T) {
	_, validacionT, _, validacionLTA := cadenaVinculosPAdESPrueba(t)
	casos := []struct {
		nombre string
		base   ValidacionFirmaServidor
		mutar  func(*ValidacionFirmaServidor)
	}{
		{"referencia de sello", validacionT, func(v *ValidacionFirmaServidor) { v.SelloTiempoVerificadoRef += ":ajena" }},
		{"huella de sello", validacionT, func(v *ValidacionFirmaServidor) { v.HuellaSelloTiempoVerificadaSHA256 = huellaPruebaPuertos("a") }},
		{"referencia de aumento", validacionLTA, func(v *ValidacionFirmaServidor) { v.AumentoLongevidadVerificadoRef += ":ajena" }},
		{"huella de aumento", validacionLTA, func(v *ValidacionFirmaServidor) { v.HuellaAumentoLongevidadVerificadaSHA256 = huellaPruebaPuertos("a") }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			mutada := caso.base
			caso.mutar(&mutada)
			if mismaEvidenciaValidacionFirma(caso.base, mutada) {
				t.Fatal("dos atestaciones con distinta evidencia embebida se consideraron iguales")
			}
		})
	}
}

func TestCoberturaFirmaRechazaReferenciaVinculoNoDerivadaDeHuella(t *testing.T) {
	cadena := nuevaCadenaManifiestoVinculoPAdESPrueba(t)
	firmaBase, err := ConstituirFirmaDecisionConfiable(
		cadena.contenido, cadena.politica, cadena.artefacto, cadena.validacionInicial,
		&cadena.sello, &cadena.validacionTrasSello, &cadena.aumento,
		cadena.validacionFinal, cadena.documento, cadena.manifiesto,
	)
	if err != nil {
		t.Fatalf("firma base: %v", err)
	}
	version := ReferenciaVersionBaremacion{
		BaremacionMeritoRef: cadena.contenido.BaremacionMeritoRef,
		Numero:              cadena.contenido.VersionAnteriorBaremacion,
		HuellaEstadoSHA256:  cadena.contenido.HuellaEstadoAnteriorSHA256,
	}
	casos := []struct {
		nombre string
		tipo   TipoEvidenciaProbatoriaBaremacion
		mutar  func(*dominiobolsa.FirmaDecisionTecnica, string)
	}{
		{
			nombre: "B-T", tipo: EvidenciaVinculoRevisionSelladaBaremacion,
			mutar: func(f *dominiobolsa.FirmaDecisionTecnica, referencia string) {
				f.VinculoRevisionSelladaRef = referencia
			},
		},
		{
			nombre: "T-LTA", tipo: EvidenciaVinculoRevisionLongevaBaremacion,
			mutar: func(f *dominiobolsa.FirmaDecisionTecnica, referencia string) {
				f.VinculoRevisionLongevaRef = referencia
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			manifiesto := cadena.manifiesto.Clonar()
			evidencia := evidenciaManifiestoPorTipoPrueba(t, &manifiesto, caso.tipo)
			referenciaNoDerivada := string(caso.tipo) + ":" + huellaPruebaPuertos("a")
			evidencia.Referencia = referenciaNoDerivada
			manifiesto, err = resellarManifiestoPrueba(manifiesto)
			if err != nil {
				t.Fatalf("resellar manifiesto con referencia opaca: %v", err)
			}
			firma := firmaBase
			caso.mutar(&firma, referenciaNoDerivada)
			firma.ManifiestoProbatorioRef = manifiesto.Referencia
			firma.HuellaManifiestoProbatorioSHA256 = manifiesto.HuellaManifiestoSHA256
			firma.SelloManifiestoProbatorioHMACSHA256 = manifiesto.SelloManifiestoHMACSHA256
			if err := manifiesto.ValidarCoberturaFirmaPara(version, cadena.contenido, firma); err == nil {
				t.Fatal("una referencia de vinculo no derivada de su huella supero la cobertura")
			}
		})
	}
}

package ports

import (
	"testing"
	"time"
)

func solicitudSelloTiempoValidaAdversarial(
	t *testing.T,
) (SolicitudSellarTiempoFirma, SelloTiempoFirma) {
	t.Helper()
	contenido := contenidoDecisionValidoPrueba(t)
	politica := politicaFirmaValidaPrueba()
	artefacto := artefactoFirmaValidoPrueba(t, contenido, politica, "inicial")
	validacion := validacionFirmaValidaPrueba(
		artefacto,
		instantePuertosPrueba.Add(6*time.Minute),
		"inicial-adversarial",
		PerfilFirmaPAdESBaselineB,
	)
	solicitud := SolicitudSellarTiempoFirma{
		Contexto: contextoFirmaValido(
			AccionSellarTiempoDecisionBaremacion,
			artefacto.FirmaRef,
		),
		ClaveIdempotencia: "sellar-tiempo-adversarial-001",
		ArtefactoOrigen:   artefacto,
		ValidacionOrigen:  validacion,
		Politica:          politica,
		SolicitadaEn:      instantePuertosPrueba.Add(6*time.Minute + time.Second),
	}
	if err := solicitud.Validar(); err != nil {
		t.Fatalf("precondicion: solicitud de sello valida: %v", err)
	}
	sello := selloValidoPrueba(artefacto, politica)
	if err := sello.ValidarPara(solicitud); err != nil {
		t.Fatalf("precondicion: sello material valido: %v", err)
	}
	return solicitud, sello
}

func TestSelloTiempoPAdESTRechazaRevisionNoOp(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*SelloTiempoFirma)
	}{
		{"misma referencia", func(s *SelloTiempoFirma) {
			s.ArtefactoSellado.DocumentoFirmadoRef = s.ArtefactoOrigen.DocumentoFirmadoRef
		}},
		{"misma huella", func(s *SelloTiempoFirma) {
			s.ArtefactoSellado.HuellaDocumentoSHA256 = s.ArtefactoOrigen.HuellaDocumentoSHA256
		}},
		{"misma referencia y huella", func(s *SelloTiempoFirma) {
			s.ArtefactoSellado.DocumentoFirmadoRef = s.ArtefactoOrigen.DocumentoFirmadoRef
			s.ArtefactoSellado.HuellaDocumentoSHA256 = s.ArtefactoOrigen.HuellaDocumentoSHA256
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			solicitud, sello := solicitudSelloTiempoValidaAdversarial(t)
			caso.mutar(&sello)
			if err := sello.ValidarPara(solicitud); err == nil {
				t.Fatal("el sello sin una nueva revision fisica e inequivoca del PDF fue admitido")
			}
		})
	}
}

func TestSelloTiempoPAdESTRechazaCrucesEntreFirmas(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*SelloTiempoFirma)
	}{
		{
			nombre: "decision ajena",
			mutar: func(s *SelloTiempoFirma) {
				s.ArtefactoSellado.DecisionRef = "decision-ajena-001"
			},
		},
		{
			nombre: "firmante ajeno",
			mutar: func(s *SelloTiempoFirma) {
				s.ArtefactoSellado.FirmanteRef = "firmante-ajeno-001"
			},
		},
		{
			nombre: "politica de firma ajena",
			mutar: func(s *SelloTiempoFirma) {
				s.ArtefactoSellado.PoliticaFirmaRef = "politica-firma-ajena-001"
			},
		},
		{
			nombre: "firma criptografica ajena",
			mutar: func(s *SelloTiempoFirma) {
				s.ArtefactoSellado.FirmaRef = "firma-ajena-001"
			},
		},
		{
			nombre: "huella de firma criptografica ajena",
			mutar: func(s *SelloTiempoFirma) {
				s.ArtefactoSellado.HuellaFirmaSHA256 = huellaPruebaPuertos("0")
			},
		},
		{
			nombre: "politica de sello ajena",
			mutar: func(s *SelloTiempoFirma) {
				s.PoliticaSelloTiempoRef = "politica-sello-ajena-001"
			},
		},
	}

	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			solicitud, sello := solicitudSelloTiempoValidaAdversarial(t)
			caso.mutar(&sello)
			if err := sello.ValidarPara(solicitud); err == nil {
				t.Fatal("el sello cruzado con otra decision, identidad o politica fue admitido")
			}
		})
	}
}

func TestAumentoPAdESLTARechazaRevisionNoOp(t *testing.T) {
	_, sello := solicitudSelloTiempoValidaAdversarial(t)
	politica := politicaFirmaValidaPrueba()
	validacionT := validacionFirmaValidaPrueba(
		sello.ArtefactoSellado,
		instantePuertosPrueba.Add(8*time.Minute),
		"sellado-adversarial",
		PerfilFirmaPAdESBaselineT,
	)
	solicitud := SolicitudAumentarFirma{
		Contexto: contextoFirmaValido(
			AccionAumentarFirmaDecisionBaremacion,
			sello.ArtefactoSellado.FirmaRef,
		),
		ClaveIdempotencia: "aumentar-firma-adversarial-001",
		Artefacto:         sello.ArtefactoSellado,
		Validacion:        validacionT,
		SelloTiempo:       &sello,
		Politica:          politica,
		SolicitadaEn:      instantePuertosPrueba.Add(8*time.Minute + time.Second),
	}
	if err := solicitud.Validar(); err != nil {
		t.Fatalf("precondicion: solicitud de aumento valida: %v", err)
	}
	resultado := ResultadoAumentoFirma{
		ArtefactoOrigen:                sello.ArtefactoSellado,
		Artefacto:                      sello.ArtefactoSellado,
		NivelAlcanzadoClave:            politica.NivelAumentoClave,
		PoliticaLongevidadRef:          politica.PoliticaLongevidadRef,
		PoliticaLongevidadVersion:      politica.PoliticaLongevidadVersion,
		HuellaPoliticaLongevidadSHA256: politica.HuellaPoliticaLongevidadSHA256,
		EvidenciaAumentoRef:            "evidencia-aumento-adversarial-001",
		HuellaEvidenciaSHA256:          huellaPruebaPuertos("f"),
		AumentadaEn:                    instantePuertosPrueba.Add(9 * time.Minute),
	}

	if err := resultado.ValidarPara(solicitud); err == nil {
		t.Fatal("el aumento LTA que devolvio exactamente el mismo PDF fue admitido")
	}
}

func TestValidacionPAdESTRechazaPerfilBaselineB(t *testing.T) {
	_, sello := solicitudSelloTiempoValidaAdversarial(t)
	politica := politicaFirmaValidaPrueba()
	solicitud := SolicitudValidarFirmaServidor{
		Contexto: contextoFirmaValido(
			AccionValidarFirmaDecisionBaremacion,
			sello.ArtefactoSellado.FirmaRef,
		),
		Artefacto:                       sello.ArtefactoSellado,
		Politica:                        politica,
		FirmanteEsperadoRef:             sello.ArtefactoSellado.FirmanteRef,
		PerfilEsperadoClave:             sello.ArtefactoSellado.PerfilFirmanteClave,
		PerfilFirmaEsperadoClave:        PerfilFirmaPAdESBaselineT,
		SelloTiempoEsperadoRef:          sello.SelloTiempoRef,
		HuellaSelloTiempoEsperadaSHA256: sello.HuellaSelloTiempoSHA256,
		SolicitadaEn:                    instantePuertosPrueba.Add(7*time.Minute + time.Second),
	}
	if err := solicitud.Validar(); err != nil {
		t.Fatalf("precondicion: solicitud de validacion PAdES-T valida: %v", err)
	}
	validacionB := validacionFirmaValidaPrueba(
		sello.ArtefactoSellado,
		instantePuertosPrueba.Add(8*time.Minute),
		"perfil-b-adversarial",
		PerfilFirmaPAdESBaselineB,
	)
	if err := validacionB.Validar(); err != nil {
		t.Fatalf("precondicion: respuesta B estructuralmente valida: %v", err)
	}

	if err := validacionB.ValidarPara(solicitud); err == nil {
		t.Fatal("la respuesta que declaro perfil B fue admitida cuando se exigia PAdES-T")
	}
}

func TestSolicitudValidacionRechazaPerfilFueraDeLaProgresionDePolitica(t *testing.T) {
	_, sello := solicitudSelloTiempoValidaAdversarial(t)
	politica := politicaFirmaValidaPrueba()
	politica.PerfilFirmaClave = PerfilFirmaPAdESBaselineB
	politica.RequiereSelloTiempo = false
	politica.PoliticaSelloTiempoRef = ""
	politica.PoliticaSelloTiempoVersion = 0
	politica.HuellaPoliticaSelloTiempoSHA256 = ""
	politica.RequiereAumentoLongevidad = false
	politica.NivelAumentoClave = ""
	politica.PoliticaLongevidadRef = ""
	politica.PoliticaLongevidadVersion = 0
	politica.HuellaPoliticaLongevidadSHA256 = ""
	artefacto := sello.ArtefactoOrigen
	solicitud := SolicitudValidarFirmaServidor{
		Contexto:  contextoFirmaValido(AccionValidarFirmaDecisionBaremacion, artefacto.FirmaRef),
		Artefacto: artefacto, Politica: politica, FirmanteEsperadoRef: artefacto.FirmanteRef,
		PerfilEsperadoClave: artefacto.PerfilFirmanteClave, PerfilFirmaEsperadoClave: PerfilFirmaPAdESBaselineT,
		SolicitadaEn: instantePuertosPrueba.Add(7 * time.Minute),
	}
	if err := politica.Validar(); err != nil {
		t.Fatalf("precondicion: politica Baseline-B valida: %v", err)
	}
	if err := solicitud.Validar(); err == nil {
		t.Fatal("una politica Baseline-B permitio solicitar una atestacion PAdES-T")
	}
}

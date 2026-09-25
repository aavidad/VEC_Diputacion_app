package validadorautofirma

import (
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/documentos/ports"
)

var (
	huellaFirmadoPrueba  = strings.Repeat("1a", 32)
	huellaOriginalPrueba = strings.Repeat("2b", 32)
	huellaCertPrueba     = strings.Repeat("3c", 32)
)

func dictamenPrueba() *dictamenAutofirma {
	firmante := firmanteAutofirma{
		CertificadoHuellaSHA256: huellaCertPrueba,
		Cadena:                  aspectoAutofirma{"valida"}, Certificado: aspectoAutofirma{"vigente"},
		Revocacion: aspectoAutofirma{"vigente"}, SelloTiempo: aspectoAutofirma{"no_presente"},
	}
	return &dictamenAutofirma{
		Contrato: ContratoDictamen, Estado: "valida", Motivo: "verificada",
		Integridad: aspectoAutofirma{"valida"}, Cadena: aspectoAutofirma{"valida"},
		Certificado: aspectoAutofirma{"vigente"}, Revocacion: aspectoAutofirma{"vigente"},
		SelloTiempo: aspectoAutofirma{"no_presente"}, VinculoOriginal: aspectoAutofirma{"acreditado"},
		HuellaFirmadoSHA256: huellaFirmadoPrueba, HuellaOriginalSHA256: huellaOriginalPrueba,
		CertificadoHuellaSHA256: huellaCertPrueba, Firmantes: []firmanteAutofirma{firmante},
		Extensiones: extensionesDictamen{"desactivada", "desactivada"},
	}
}

func resultadoBase() ports.ResultadoVerificacionFirma {
	return ports.ResultadoVerificacionFirma{
		Estado: ports.EstadoVerificacionIndeterminada, HuellaOriginalSHA256: huellaOriginalPrueba,
		HuellaFirmadoSHA256: huellaFirmadoPrueba, SelloTiempoEstado: estadoNoInformado,
		RevocacionEstado: estadoNoInformado,
	}
}

func TestDictamenValidoSeTraduceAVerificada(t *testing.T) {
	r := traducir(resultadoBase(), dictamenPrueba())
	if r.Motivo != ports.MotivoFirmaVerificada || r.Resultado.FirmanteRef != "ref:"+huellaCertPrueba ||
		!r.Resultado.VinculoOriginal || r.Resultado.RevocacionEstado != "vigente" {
		t.Fatalf("dictamen valido: %+v", r)
	}
	// Las extensiones remotas activas no bloquean: son evidencia adicional.
	d := dictamenPrueba()
	d.Extensiones = extensionesDictamen{"activa", "activa"}
	if r := traducir(resultadoBase(), d); r.Motivo != ports.MotivoFirmaVerificada {
		t.Fatalf("extensiones activas: %+v", r)
	}
}

func TestDictamenMalFormadoNoEsInterpretable(t *testing.T) {
	mutaciones := map[string]func(*dictamenAutofirma){
		"contrato vacio":        func(d *dictamenAutofirma) { d.Contrato = "" },
		"contrato v2":           func(d *dictamenAutofirma) { d.Contrato = "autofirmav2.dictamen-verificacion.v2" },
		"estado desconocido":    func(d *dictamenAutofirma) { d.Estado = "VALIDA" },
		"motivo desconocido":    func(d *dictamenAutofirma) { d.Motivo = "otro" },
		"estado sin motivo":     func(d *dictamenAutofirma) { d.Estado = "indeterminada" },
		"integridad ajena":      func(d *dictamenAutofirma) { d.Integridad.Estado = "valid" },
		"revocacion ajena":      func(d *dictamenAutofirma) { d.Revocacion.Estado = "good" },
		"sello ajeno":           func(d *dictamenAutofirma) { d.SelloTiempo.Estado = "" },
		"vinculo ajeno":         func(d *dictamenAutofirma) { d.VinculoOriginal.Estado = "si" },
		"extension ajena":       func(d *dictamenAutofirma) { d.Extensiones.RevocacionRemota = "" },
		"firmante ajeno":        func(d *dictamenAutofirma) { d.Firmantes[0].Revocacion.Estado = "x" },
		"firmante sin huella":   func(d *dictamenAutofirma) { d.Firmantes[0].CertificadoHuellaSHA256 = "" },
		"huella cert distinta":  func(d *dictamenAutofirma) { d.CertificadoHuellaSHA256 = strings.Repeat("4d", 32) },
		"huella cert mayuscula": func(d *dictamenAutofirma) { d.CertificadoHuellaSHA256 = strings.ToUpper(huellaCertPrueba) },
		"huella cert ausente":   func(d *dictamenAutofirma) { d.CertificadoHuellaSHA256 = "" },
		"huella cert cero": func(d *dictamenAutofirma) {
			d.CertificadoHuellaSHA256 = strings.Repeat("0", 64)
			d.Firmantes[0].CertificadoHuellaSHA256 = d.CertificadoHuellaSHA256
		},
		"eco firmado distinto":  func(d *dictamenAutofirma) { d.HuellaFirmadoSHA256 = strings.Repeat("5e", 32) },
		"eco firmado ausente":   func(d *dictamenAutofirma) { d.HuellaFirmadoSHA256 = "" },
		"eco original distinto": func(d *dictamenAutofirma) { d.HuellaOriginalSHA256 = strings.Repeat("6f", 32) },
		"demasiados firmantes": func(d *dictamenAutofirma) {
			for len(d.Firmantes) <= maximoFirmantes {
				d.Firmantes = append(d.Firmantes, d.Firmantes[0])
			}
			d.CertificadoHuellaSHA256 = ""
		},
		"negativo incoherente": func(d *dictamenAutofirma) { d.Estado, d.Motivo = "no_valida", "confianza_no_valida" },
	}
	for nombre, mutar := range mutaciones {
		d := dictamenPrueba()
		mutar(d)
		r := traducir(resultadoBase(), d)
		if r.Motivo != ports.MotivoRespuestaNoInterpretable || r.Resultado.Estado != ports.EstadoVerificacionIndeterminada ||
			r.Resultado.RevocacionEstado != estadoNoInformado || r.Resultado.FirmanteRef != "" {
			t.Fatalf("%s: %+v", nombre, r)
		}
	}
	if r := traducir(resultadoBase(), nil); r.Motivo != ports.MotivoRespuestaNoInterpretable {
		t.Fatalf("sin dictamen: %+v", r)
	}
}

func TestVECMasEstrictaQueUnValidaDelValidador(t *testing.T) {
	casos := map[string]struct {
		mutar  func(*dictamenAutofirma)
		motivo ports.MotivoVerificacionFirma
	}{
		"original no aportado": {func(d *dictamenAutofirma) {
			d.VinculoOriginal.Estado, d.HuellaOriginalSHA256 = "no_aportado", ""
		}, ports.MotivoVinculoOriginalNoAcreditado},
		"sello no valido":       {func(d *dictamenAutofirma) { d.SelloTiempo.Estado = "no_valido" }, ports.MotivoSelloTiempoNoAcreditado},
		"certificado no compr.": {func(d *dictamenAutofirma) { d.Certificado.Estado = "no_comprobado" }, ports.MotivoCertificadoNoAcreditado},
		"sin firmantes": {func(d *dictamenAutofirma) {
			d.Firmantes, d.CertificadoHuellaSHA256 = nil, ""
		}, ports.MotivoFirmanteNoIdentificado},
	}
	for nombre, caso := range casos {
		d := dictamenPrueba()
		caso.mutar(d)
		r := traducir(resultadoBase(), d)
		if r.Motivo != caso.motivo || r.Resultado.Estado != ports.EstadoVerificacionIndeterminada {
			t.Fatalf("%s: %+v", nombre, r)
		}
	}
}

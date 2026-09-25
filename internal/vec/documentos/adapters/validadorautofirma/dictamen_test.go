package validadorautofirma

import (
	"encoding/json"
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
		Cadena:                  aspectoAutofirma{Estado: "valida"}, Certificado: aspectoAutofirma{Estado: "vigente"},
		Revocacion: aspectoAutofirma{Estado: "vigente"}, SelloTiempo: aspectoAutofirma{Estado: "no_presente"},
	}
	return &dictamenAutofirma{
		Contrato: ContratoDictamen, Estado: "valida", Motivo: "verificada",
		Integridad: aspectoAutofirma{Estado: "valida"}, Cadena: aspectoAutofirma{Estado: "valida"},
		Certificado: aspectoAutofirma{Estado: "vigente"}, Revocacion: aspectoAutofirma{Estado: "vigente"},
		SelloTiempo: aspectoAutofirma{Estado: "no_presente"}, VinculoOriginal: aspectoAutofirma{Estado: "acreditado"},
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
		// Coherencia entre firmantes y agregados: un agregado favorable no
		// puede ocultar el defecto de un firmante, ni ser peor que todos.
		"firmante revocado":        func(d *dictamenAutofirma) { d.Firmantes[0].Revocacion.Estado = "revocado" },
		"firmante no vigente":      func(d *dictamenAutofirma) { d.Firmantes[0].Certificado.Estado = "no_vigente" },
		"firmante uso no admitido": func(d *dictamenAutofirma) { d.Firmantes[0].Certificado.Estado = "uso_no_permitido" },
		"firmante cadena rota":     func(d *dictamenAutofirma) { d.Firmantes[0].Cadena.Estado = "no_valida" },
		"firmante sello no valido": func(d *dictamenAutofirma) { d.Firmantes[0].SelloTiempo.Estado = "no_valido" },
		"firmante revocacion no comprobada": func(d *dictamenAutofirma) {
			d.Firmantes[0].Revocacion.Estado = "no_comprobada"
		},
		"agregado peor que el firmante": func(d *dictamenAutofirma) {
			d.Revocacion.Estado, d.Estado, d.Motivo = "no_comprobada", "indeterminada", "revocacion_no_acreditada"
		},
		"segundo firmante revocado oculto": func(d *dictamenAutofirma) {
			otro := d.Firmantes[0]
			otro.CertificadoHuellaSHA256 = strings.Repeat("7a", 32)
			otro.Revocacion.Estado = "revocado"
			d.Firmantes = append(d.Firmantes, otro)
			d.CertificadoHuellaSHA256, d.Estado, d.Motivo = "", "indeterminada", "firmante_no_identificado"
		},
		"sin firmantes con agregados favorables": func(d *dictamenAutofirma) {
			d.Firmantes, d.CertificadoHuellaSHA256 = nil, ""
			d.Estado, d.Motivo = "indeterminada", "firmante_no_identificado"
		},
		"eco original ausente": func(d *dictamenAutofirma) { d.HuellaOriginalSHA256 = "" },
		"eco original sin vinculo aportado": func(d *dictamenAutofirma) {
			d.VinculoOriginal.Estado, d.Estado, d.Motivo = "no_aportado", "indeterminada", "vinculo_original_no_acreditado"
		},
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
		"sello no valido": {func(d *dictamenAutofirma) {
			d.SelloTiempo.Estado, d.Firmantes[0].SelloTiempo.Estado = "no_valido", "no_valido"
		}, ports.MotivoSelloTiempoNoAcreditado},
		"certificado no compr.": {func(d *dictamenAutofirma) {
			d.Certificado.Estado, d.Firmantes[0].Certificado.Estado = "no_comprobado", "no_comprobado"
		}, ports.MotivoCertificadoNoAcreditado},
		// Sin firmantes, los agregados son los estados por defecto del contrato.
		"sin firmantes": {func(d *dictamenAutofirma) {
			d.Firmantes, d.CertificadoHuellaSHA256 = nil, ""
			d.Cadena.Estado, d.Certificado.Estado = "no_comprobada", "no_comprobado"
			d.Revocacion.Estado, d.SelloTiempo.Estado = "no_comprobada", "no_presente"
		}, ports.MotivoFirmanteNoIdentificado},
		// Varios firmantes con agregado por el peor estado: coherente, pero sin
		// firmante unico no hay positivo.
		"varios firmantes coherentes": {func(d *dictamenAutofirma) {
			otro := d.Firmantes[0]
			otro.CertificadoHuellaSHA256 = strings.Repeat("7a", 32)
			otro.SelloTiempo.Estado = "valido"
			d.Firmantes, d.CertificadoHuellaSHA256 = append(d.Firmantes, otro), ""
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

// cuerpoPrueba serializa una respuesta completa con el dictamen valido y un
// campo heredado de primer nivel, que el adaptador ignora.
func cuerpoPrueba(t *testing.T) string {
	t.Helper()
	dictamen, err := json.Marshal(dictamenPrueba())
	if err != nil {
		t.Fatal(err)
	}
	return `{"valid":true,"dictamen":` + string(dictamen) + `}`
}

func TestDecodificacionEstrictaDeLaRespuesta(t *testing.T) {
	valido := cuerpoPrueba(t)
	d, ok := decodificarRespuesta([]byte(valido))
	if !ok || d == nil {
		t.Fatal("respuesta valida rechazada")
	}
	if r := traducir(resultadoBase(), d); r.Motivo != ports.MotivoFirmaVerificada {
		t.Fatalf("respuesta valida: %+v", r)
	}
	incluir := func(ancla, extra string) string {
		if !strings.Contains(valido, ancla) {
			t.Fatalf("ancla ausente: %s", ancla)
		}
		return strings.Replace(valido, ancla, ancla+extra, 1)
	}
	rechazadas := map[string]string{
		"clave ajena en dictamen":    incluir(`"dictamen":{`, `"veredictoExtra":"valida",`),
		"clave ajena en aspecto":     incluir(`"revocacion":{`, `"forzado":true,`),
		"clave ajena en firmante":    incluir(`"firmantes":[{`, `"aceptado":true,`),
		"clave duplicada":            incluir(`"dictamen":{`, `"estado":"no_valida",`),
		"clave duplicada plegada":    incluir(`"dictamen":{`, `"ESTADO":"no_valida",`),
		"dictamen duplicado":         incluir(`{"valid":true,`, `"dictamen":null,`),
		"duplicada en firmante":      incluir(`"firmantes":[{`, `"revocacion":{"estado":"revocado"},`),
		"dos objetos seguidos":       valido + valido,
		"datos sobrantes":            valido + ` 1`,
		"profundidad excesiva":       `{"valid":` + strings.Repeat("[", maximaProfundidad+1) + strings.Repeat("]", maximaProfundidad+1) + `}`,
		"json truncado":              valido[:len(valido)-1],
		"dictamen no objeto":         `{"dictamen":"valida"}`,
		"mayor que el limite":        `{"details":"` + strings.Repeat("x", maximaRespuesta) + `","dictamen":null}`,
		"valido mayor que el limite": incluir(`{"valid":true,`, `"details":"`+strings.Repeat("x", maximaRespuesta)+`",`),
	}
	for nombre, cuerpo := range rechazadas {
		if d, ok := decodificarRespuesta([]byte(cuerpo)); ok || d != nil {
			t.Fatalf("%s: aceptada", nombre)
		}
	}
	for _, sin := range []string{`{"valid":true}`, `{"dictamen":null}`} {
		d, ok := decodificarRespuesta([]byte(sin))
		if !ok || d != nil {
			t.Fatalf("%s: %v %v", sin, d, ok)
		}
		if r := traducir(resultadoBase(), d); r.Motivo != ports.MotivoRespuestaNoInterpretable {
			t.Fatalf("%s: %+v", sin, r)
		}
	}
}

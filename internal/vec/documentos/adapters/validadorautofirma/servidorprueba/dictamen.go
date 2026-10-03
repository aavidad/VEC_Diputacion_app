package servidorprueba

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// Aspecto reproduce `{estado, motivo?, fuente?, fecha?}` del contrato.
type Aspecto struct {
	Estado string `json:"estado"`
	Motivo string `json:"motivo,omitempty"`
	Fuente string `json:"fuente,omitempty"`
	Fecha  string `json:"fecha,omitempty"`
}

// Cambio reproduce la clasificación de cambios del contrato v2.
type Cambio struct {
	Estado          string   `json:"estado"`
	Detalle         []string `json:"detalle"`
	BytesNoFirmados *int     `json:"bytesNoFirmados,omitempty"`
}

// Firmante reproduce una firma del dictamen, con datos inventados.
type Firmante struct {
	Orden                           int     `json:"orden"`
	ByteRange                       [4]int  `json:"byteRange"`
	RevisionHuellaSHA256            string  `json:"revisionHuellaSHA256"`
	ContenidoFirmadoHuellaSHA256    string  `json:"contenidoFirmadoHuellaSHA256"`
	RevisionLongitud                int     `json:"revisionLongitud"`
	CubreDocumentoCompletoHastaAqui bool    `json:"cubreDocumentoCompletoHastaAqui"`
	Integridad                      Aspecto `json:"integridad"`
	CertificadoHuellaSHA256         string  `json:"certificadoHuellaSHA256"`
	Serie                           string  `json:"serie,omitempty"`
	Asunto                          string  `json:"asunto,omitempty"`
	Emisor                          string  `json:"emisor,omitempty"`
	Cadena                          Aspecto `json:"cadena"`
	Certificado                     Aspecto `json:"certificado"`
	Revocacion                      Aspecto `json:"revocacion"`
	SelloTiempo                     Aspecto `json:"selloTiempo"`
	TipoFirma                       string  `json:"tipoFirma"`
	NivelDocMDP                     *int    `json:"nivelDocMDP"`
	CambiosDesdeAnterior            Cambio  `json:"cambiosDesdeAnterior"`
}

// Dictamen reproduce `dictamen` de `autofirmav2.dictamen-verificacion.v2`.
type Dictamen struct {
	Contrato             string            `json:"contrato"`
	Estado               string            `json:"estado"`
	Motivo               string            `json:"motivo"`
	Formato              string            `json:"formato,omitempty"`
	ComprobadoEn         string            `json:"comprobadoEn"`
	Integridad           Aspecto           `json:"integridad"`
	Cadena               Aspecto           `json:"cadena"`
	Certificado          Aspecto           `json:"certificado"`
	Revocacion           Aspecto           `json:"revocacion"`
	SelloTiempo          Aspecto           `json:"selloTiempo"`
	VinculoOriginal      Aspecto           `json:"vinculoOriginal"`
	HuellaFirmadoSHA256  string            `json:"huellaFirmadoSHA256"`
	HuellaOriginalSHA256 string            `json:"huellaOriginalSHA256,omitempty"`
	Firmas               []Firmante        `json:"firmas"`
	CambiosPosteriores   Cambio            `json:"cambiosPosteriores"`
	Extensiones          map[string]string `json:"extensiones"`

	// fijado impide que recomponer sustituya un veredicto deliberadamente
	// incoherente de la prueba.
	fijado bool
}

func dictamenValido(p Peticion) *Dictamen {
	fecha := "2026-09-25T10:00:00Z"
	longitud := len(p.Firmado)
	primerTramo := min(len(p.Original), longitud)
	rango := [4]int{0, primerTramo, primerTramo, longitud - primerTramo}
	firmante := Firmante{
		Orden:                           1,
		ByteRange:                       rango,
		RevisionHuellaSHA256:            huella(p.Firmado),
		ContenidoFirmadoHuellaSHA256:    huellaByteRange(p.Firmado, rango),
		RevisionLongitud:                longitud,
		CubreDocumentoCompletoHastaAqui: true,
		Integridad:                      Aspecto{Estado: "valida"},
		CertificadoHuellaSHA256:         HuellaCertificadoSintetica, Serie: "07",
		Asunto: "CN=PERSONA SINTETICA", Emisor: "CN=CA SINTETICA",
		Cadena:               Aspecto{Estado: "valida", Fuente: "anclas_locales"},
		Certificado:          Aspecto{Estado: "vigente", Fecha: "2028-01-31T23:59:59Z"},
		Revocacion:           Aspecto{Estado: "vigente", Fuente: "crl_local", Fecha: fecha},
		SelloTiempo:          Aspecto{Estado: "no_presente"},
		TipoFirma:            "aprobacion",
		CambiosDesdeAnterior: Cambio{Estado: "permitidos", Detalle: []string{"firma_anadida"}},
	}
	d := &Dictamen{
		Contrato: ContratoDictamen, Formato: "PAdES", ComprobadoEn: fecha,
		Integridad:          Aspecto{Estado: "valida"},
		VinculoOriginal:     Aspecto{Estado: "acreditado", Fuente: "pades_revision"},
		HuellaFirmadoSHA256: huella(p.Firmado),
		Firmas:              []Firmante{firmante},
		CambiosPosteriores:  Cambio{Estado: "ninguno", Detalle: []string{}},
		Extensiones:         map[string]string{"revocacionRemota": "desactivada", "selloTiempoRemoto": "desactivada"},
	}
	if !p.OriginalFalta {
		d.HuellaOriginalSHA256 = huella(p.Original)
	} else {
		d.VinculoOriginal = Aspecto{Estado: "no_aportado"}
	}
	return d
}

func aplicar(esc Escenario, d *Dictamen, p Peticion) {
	f := &d.Firmas[0]
	switch esc {
	case ValidaConSello:
		f.SelloTiempo = Aspecto{Estado: "valido", Fuente: "rfc3161", Fecha: "2026-09-25T09:59:00Z"}
	case SelloNoComprobado:
		f.SelloTiempo = Aspecto{Estado: "no_comprobado", Motivo: "formato_sin_evaluacion_de_sello"}
	case SelloNoValido:
		f.SelloTiempo = Aspecto{Estado: "no_valido", Motivo: "sello_no_corresponde_a_la_firma"}
	case Revocado:
		f.Revocacion = Aspecto{Estado: "revocado", Fuente: "crl_local", Fecha: "2026-09-01T00:00:00Z"}
	case RevocacionNoComprobada:
		f.Revocacion = Aspecto{Estado: "no_comprobada", Motivo: "sin_crl_del_emisor"}
	case VinculoNoAcreditado:
		d.VinculoOriginal = Aspecto{Estado: "no_acreditado", Motivo: "original_no_es_revision_previa"}
	case VinculoNoAportado:
		d.HuellaOriginalSHA256 = ""
		d.VinculoOriginal = Aspecto{Estado: "no_aportado"}
	case ContratoDesconocido:
		d.Contrato = "autofirmav2.dictamen-verificacion.v1"
	case HuellaEcoDistinta:
		d.HuellaFirmadoSHA256 = strings.Repeat("cd", 32)
	case HuellaOriginalDistinta:
		d.HuellaOriginalSHA256 = strings.Repeat("ef", 32)
	case IntegridadRota:
		d.Integridad = Aspecto{Estado: "no_valida", Motivo: "firma_no_corresponde_al_contenido"}
		f.Integridad = d.Integridad
	case IntegridadParcial:
		d.Integridad = Aspecto{Estado: "parcial", Motivo: "contenido_no_cubierto_por_la_firma"}
		f.Integridad = d.Integridad
	case SinAnclas:
		f.Cadena = Aspecto{Estado: "no_comprobada", Motivo: "sin_cadena_hasta_ancla"}
	case VariosFirmantes:
		otro := *f
		primeraLongitud := f.RevisionLongitud - 1
		primerTramo := min(len(p.Original), primeraLongitud)
		f.ByteRange = [4]int{0, primerTramo, primerTramo, primeraLongitud - primerTramo}
		f.RevisionLongitud = primeraLongitud
		f.RevisionHuellaSHA256 = huella(p.Firmado[:primeraLongitud])
		f.ContenidoFirmadoHuellaSHA256 = huellaByteRange(p.Firmado[:primeraLongitud], f.ByteRange)
		otro.Orden = 2
		otro.CertificadoHuellaSHA256 = strings.Repeat("12", 32)
		d.Firmas = append(d.Firmas, otro)
	case ValidaIncoherente:
		f.Revocacion = Aspecto{Estado: "no_comprobada"}
		d.Estado, d.Motivo, d.fijado = "valida", "verificada", true
	case NegativaIncoherente:
		d.Estado, d.Motivo, d.fijado = "no_valida", "integridad_no_valida", true
	case EstadoDesconocido:
		f.Certificado = Aspecto{Estado: "quiza"}
	}
}

// recomponer agrega los aspectos por el peor estado y deriva el veredicto
// con la misma precedencia que GrxFirma.
func (d *Dictamen) recomponer() {
	peor := func(orden []string, valor func(Firmante) Aspecto) Aspecto {
		mejor, rango := Aspecto{}, len(orden)+1
		for _, f := range d.Firmas {
			a := valor(f)
			r := len(orden)
			for i, e := range orden {
				if e == a.Estado {
					r = i
				}
			}
			if r < rango {
				mejor, rango = a, r
			}
		}
		return mejor
	}
	d.Cadena = peor([]string{"no_valida", "no_comprobada", "valida"}, func(f Firmante) Aspecto { return f.Cadena })
	d.Certificado = peor([]string{"uso_no_permitido", "no_vigente", "no_comprobado", "vigente"}, func(f Firmante) Aspecto { return f.Certificado })
	d.Revocacion = peor([]string{"revocado", "no_comprobada", "vigente"}, func(f Firmante) Aspecto { return f.Revocacion })
	d.SelloTiempo = peor([]string{"no_valido", "no_comprobado", "no_presente", "valido"}, func(f Firmante) Aspecto { return f.SelloTiempo })
	if d.fijado {
		return
	}
	d.Estado, d.Motivo = "valida", "verificada"
	switch {
	case d.Integridad.Estado == "no_valida":
		d.Estado, d.Motivo = "no_valida", "integridad_no_valida"
	case d.Certificado.Estado == "no_vigente" || d.Certificado.Estado == "uso_no_permitido" || d.Revocacion.Estado == "revocado":
		d.Estado, d.Motivo = "no_valida", "certificado_no_valido"
	case d.Cadena.Estado == "no_valida":
		d.Estado, d.Motivo = "no_valida", "confianza_no_valida"
	case d.Integridad.Estado != "valida":
		d.Estado, d.Motivo = "indeterminada", "integridad_parcial"
	case d.Certificado.Estado != "vigente":
		d.Estado, d.Motivo = "indeterminada", "certificado_no_acreditado"
	case d.Cadena.Estado != "valida":
		d.Estado, d.Motivo = "indeterminada", "confianza_no_acreditada"
	case d.Revocacion.Estado != "vigente":
		d.Estado, d.Motivo = "indeterminada", "revocacion_no_acreditada"
	case d.SelloTiempo.Estado == "no_valido" || d.SelloTiempo.Estado == "no_comprobado":
		d.Estado, d.Motivo = "indeterminada", "sello_tiempo_no_acreditado"
	case d.VinculoOriginal.Estado != "acreditado":
		d.Estado, d.Motivo = "indeterminada", "vinculo_original_no_acreditado"
	}
}

func huella(b []byte) string {
	suma := sha256.Sum256(b)
	return hex.EncodeToString(suma[:])
}

func huellaByteRange(revision []byte, rango [4]int) string {
	h := sha256.New()
	_, _ = h.Write(revision[rango[0] : rango[0]+rango[1]])
	_, _ = h.Write(revision[rango[2] : rango[2]+rango[3]])
	return hex.EncodeToString(h.Sum(nil))
}

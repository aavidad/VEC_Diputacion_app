package validadorautofirma

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/vec/documentos/ports"
)

// aspectoV2 conserva el estado cerrado; descarta textos y datos del proveedor.
type aspectoV2 struct {
	Estado string `json:"estado"`
}

type cambioV2 struct {
	Estado  string   `json:"estado"`
	Detalle []string `json:"detalle"`
}

type cambiosPosterioresV2 struct {
	Estado          string   `json:"estado"`
	Detalle         []string `json:"detalle"`
	BytesNoFirmados *int64   `json:"bytesNoFirmados,omitempty"`
}

// firmaV2 identifica cada revision y su certificado, nunca una persona o cargo.
// La competencia y el orden exigido se comprueban en la autoridad de aplicacion.
// No conserva asunto, emisor, numero de serie ni nombre de campo del PDF.
type firmaV2 struct {
	Orden                           int       `json:"orden"`
	ByteRange                       [4]int64  `json:"byteRange"`
	RevisionHuellaSHA256            string    `json:"revisionHuellaSHA256"`
	ContenidoFirmadoHuellaSHA256    string    `json:"contenidoFirmadoHuellaSHA256"`
	RevisionLongitud                int64     `json:"revisionLongitud"`
	CubreDocumentoCompletoHastaAqui bool      `json:"cubreDocumentoCompletoHastaAqui"`
	CertificadoHuellaSHA256         string    `json:"certificadoHuellaSHA256,omitempty"`
	Integridad                      aspectoV2 `json:"integridad"`
	Cadena                          aspectoV2 `json:"cadena"`
	Certificado                     aspectoV2 `json:"certificado"`
	Revocacion                      aspectoV2 `json:"revocacion"`
	SelloTiempo                     aspectoV2 `json:"selloTiempo"`
	TipoFirma                       string    `json:"tipoFirma"`
	NivelDocMDP                     *int      `json:"nivelDocMDP"`
	CambiosDesdeAnterior            cambioV2  `json:"cambiosDesdeAnterior"`
}

// dictamenV2 es evidencia del proveedor ligada a los bytes de la solicitud.
// No acredita por si solo competencia, autorizacion, custodia o firma legal.
type dictamenV2 struct {
	Contrato             string               `json:"contrato"`
	Estado               string               `json:"estado"`
	Motivo               string               `json:"motivo"`
	Formato              string               `json:"formato"`
	ComprobadoEn         time.Time            `json:"comprobadoEn"`
	Integridad           aspectoV2            `json:"integridad"`
	Cadena               aspectoV2            `json:"cadena"`
	Certificado          aspectoV2            `json:"certificado"`
	Revocacion           aspectoV2            `json:"revocacion"`
	SelloTiempo          aspectoV2            `json:"selloTiempo"`
	VinculoOriginal      aspectoV2            `json:"vinculoOriginal"`
	HuellaFirmadoSHA256  string               `json:"huellaFirmadoSHA256"`
	HuellaOriginalSHA256 string               `json:"huellaOriginalSHA256,omitempty"`
	Firmas               []firmaV2            `json:"firmas"`
	CambiosPosteriores   cambiosPosterioresV2 `json:"cambiosPosteriores"`
}

// decodificarRespuestaV2 ignora la envoltura heredada y valida el dictamen
// contra el esquema publicado. Tambien exige claves exactas y no duplicadas.
func decodificarRespuestaV2(contenido []byte) (*dictamenV2, error) {
	if len(contenido) > maximaRespuesta || !utf8.Valid(contenido) {
		return nil, errEstructuraRespuesta
	}
	if err := validarJSONUnico(contenido); err != nil {
		return nil, err
	}
	valor, err := valorJSON(contenido)
	if err != nil {
		return nil, err
	}
	objeto, ok := valor.(map[string]any)
	if !ok {
		return nil, errEstructuraRespuesta
	}
	dictamen, existe := objeto["dictamen"]
	if !existe || dictamen == nil {
		return nil, errEstructuraRespuesta
	}
	if errEsquemaPublicadoV2 != nil {
		return nil, errEsquemaPublicadoV2
	}
	if esquemaPublicadoV2 == nil {
		return nil, errEstructuraRespuesta
	}
	if err := cumpleEsquemaV2(dictamen, esquemaPublicadoV2.Definiciones["dictamen"], 0); err != nil {
		return nil, err
	}
	var envoltura map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &envoltura); err != nil {
		return nil, err
	}
	var d dictamenV2
	if err := json.Unmarshal(envoltura["dictamen"], &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// VerificardictamenV2 permite consultar todas las revisiones sin reducirlas a
// un firmante. Una respuesta no interpretable nunca devuelve evidencia parcial.
// La composicion debe traducirla al puerto neutral de multifirma cuando exista.
func (c *Cliente) verificarDictamenV2(ctx context.Context, s ports.SolicitudVerificacionFirma) (dictamenV2, ports.MotivoVerificacionFirma, error) {
	if c == nil || c.http == nil || ctx == nil || s.Validar() != nil {
		return dictamenV2{}, "", ports.ErrVerificacionFirmaInvalida
	}
	d, motivo := c.llamar(ctx, peticionAutofirma{
		ContratoSolicitado: ContratoDictamen, Name: nombreDocumento,
		ContentBase64:  base64.StdEncoding.EncodeToString(s.ContenidoFirmado),
		OriginalBase64: base64.StdEncoding.EncodeToString(s.ContenidoOriginal),
	})
	if motivo != "" {
		return dictamenV2{}, motivo, nil
	}
	if !dictamenLigadoV2(d, s) {
		return dictamenV2{}, ports.MotivoRespuestaNoInterpretable, nil
	}
	c.observarDisponibilidad(ctx, true)
	return *d, "", nil
}

func huellaBytesV2(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// dictamenLigadoV2 verifica ecos, orden, limites y las DOS huellas por firma.
// No analiza PDF ni verifica CMS: esa autoridad sigue siendo GrxFirma.
func dictamenLigadoV2(d *dictamenV2, s ports.SolicitudVerificacionFirma) bool {
	if d == nil || d.Contrato != ContratoDictamen || d.Formato != "PAdES" ||
		(s.FormatoEsperado != "" && s.FormatoEsperado != d.Formato) ||
		d.HuellaFirmadoSHA256 != huellaBytesV2(s.ContenidoFirmado) ||
		(d.VinculoOriginal.Estado == "no_aportado") != (d.HuellaOriginalSHA256 == "") ||
		(d.HuellaOriginalSHA256 != "" && d.HuellaOriginalSHA256 != s.HuellaOriginalSHA256) {
		return false
	}
	var anterior int64
	for i, f := range d.Firmas {
		if f.Orden != i+1 || f.RevisionLongitud <= anterior || f.RevisionLongitud > int64(len(s.ContenidoFirmado)) {
			return false
		}
		anterior = f.RevisionLongitud
		b := f.ByteRange
		// Restas de valores ya acotados evitan desbordamientos incluso con
		// enteros hostiles. Cada segmento debe estar dentro de su revision.
		if b[0] < 0 || b[1] < 0 || b[2] < 0 || b[3] < 0 ||
			b[0] > anterior || b[1] > anterior-b[0] || b[2] < b[0]+b[1] ||
			b[2] > anterior || b[3] > anterior-b[2] {
			return false
		}
		if f.RevisionHuellaSHA256 != huellaBytesV2(s.ContenidoFirmado[:anterior]) {
			return false
		}
		h := sha256.New()
		_, _ = h.Write(s.ContenidoFirmado[b[0] : b[0]+b[1]])
		_, _ = h.Write(s.ContenidoFirmado[b[2] : b[2]+b[3]])
		if f.ContenidoFirmadoHuellaSHA256 != hex.EncodeToString(h.Sum(nil)) {
			return false
		}
		if f.CubreDocumentoCompletoHastaAqui && (b[0] != 0 || b[2]+b[3] != anterior) {
			return false
		}
		if f.TipoFirma == "certificacion" && (i != 0 || f.NivelDocMDP == nil) ||
			f.TipoFirma != "certificacion" && f.NivelDocMDP != nil {
			return false
		}
	}
	if d.CambiosPosteriores.BytesNoFirmados != nil &&
		*d.CambiosPosteriores.BytesNoFirmados != int64(len(s.ContenidoFirmado))-anterior {
		return false
	}
	if d.VinculoOriginal.Estado == "acreditado" && (len(d.Firmas) == 0 ||
		d.Firmas[0].RevisionLongitud <= int64(len(s.ContenidoOriginal)) ||
		d.Firmas[0].ByteRange[0] != 0 || d.Firmas[0].ByteRange[1] < int64(len(s.ContenidoOriginal)) ||
		!bytes.HasPrefix(s.ContenidoFirmado, s.ContenidoOriginal)) {
		return false
	}
	return coherenciaEstadoV2(d)
}

func cambioAdmisibleV2(estado string) bool { return estado == "ninguno" || estado == "permitidos" }

func coherenciaEstadoV2(d *dictamenV2) bool {
	positivo := d.Integridad.Estado == "valida" && d.Cadena.Estado == "valida" &&
		d.Certificado.Estado == "vigente" && d.Revocacion.Estado == "vigente" &&
		(d.SelloTiempo.Estado == "valido" || d.SelloTiempo.Estado == "no_presente") &&
		d.VinculoOriginal.Estado == "acreditado" && len(d.Firmas) > 0 && cambioAdmisibleV2(d.CambiosPosteriores.Estado)
	defecto := d.Integridad.Estado == "no_valida" || d.Cadena.Estado == "no_valida" ||
		d.Certificado.Estado == "no_vigente" || d.Certificado.Estado == "uso_no_permitido" ||
		d.Revocacion.Estado == "revocado" || d.CambiosPosteriores.Estado == "no_permitidos"
	for _, f := range d.Firmas {
		positivo = positivo && f.Integridad.Estado == "valida" && f.Cadena.Estado == "valida" &&
			f.Certificado.Estado == "vigente" && f.Revocacion.Estado == "vigente" &&
			(f.SelloTiempo.Estado == "valido" || f.SelloTiempo.Estado == "no_presente") &&
			f.CertificadoHuellaSHA256 != "" && f.CertificadoHuellaSHA256 != stringsCerosV2 &&
			f.CubreDocumentoCompletoHastaAqui && cambioAdmisibleV2(f.CambiosDesdeAnterior.Estado)
		defecto = defecto || f.Integridad.Estado == "no_valida" || f.Cadena.Estado == "no_valida" ||
			f.Certificado.Estado == "no_vigente" || f.Certificado.Estado == "uso_no_permitido" ||
			f.Revocacion.Estado == "revocado" || !f.CubreDocumentoCompletoHastaAqui || f.CambiosDesdeAnterior.Estado == "no_permitidos"
	}
	switch d.Estado {
	case "valida":
		return d.Motivo == "verificada" && positivo && !defecto
	case "no_valida":
		return defecto && motivoEstadoV2(d.Motivo) == ports.EstadoVerificacionNoValida
	case "indeterminada":
		return motivoEstadoV2(d.Motivo) == ports.EstadoVerificacionIndeterminada
	default:
		return false
	}
}

const stringsCerosV2 = "0000000000000000000000000000000000000000000000000000000000000000"

func motivoEstadoV2(m string) ports.EstadoVerificacionFirma {
	switch m {
	case "verificada":
		return ports.EstadoVerificacionValida
	case "integridad_no_valida", "certificado_no_valido", "confianza_no_valida", "cambios_no_permitidos", "byterange_no_cubre_revision_completa":
		return ports.EstadoVerificacionNoValida
	case "firmas_no_comprobadas", "cambios_no_comprobados", "limite_excedido", "pdf_no_comprobado":
		return ports.EstadoVerificacionIndeterminada
	default:
		return motivosDictamen[m].EstadoAsociado()
	}
}

func traducirV2(r ports.ResultadoVerificacionFirma, d *dictamenV2, s ports.SolicitudVerificacionFirma) ports.VerificacionFirmaMotivada {
	if !dictamenLigadoV2(d, s) {
		return motivar(r, ports.MotivoRespuestaNoInterpretable)
	}
	// La decision procede de dictamen.estado; motivo solo explica ese estado.
	if d.Estado != "valida" {
		motivo := motivosDictamen[d.Motivo]
		if motivo == "" {
			if d.Estado == "no_valida" {
				motivo = ports.MotivoIntegridadNoValida
			} else {
				motivo = ports.MotivoIntegridadParcial
			}
		}
		return motivar(r, motivo)
	}
	if len(d.Firmas) != 1 || d.Firmas[0].TipoFirma == "sello_tiempo_documento" {
		return motivar(r, ports.MotivoFirmanteNoIdentificado)
	}
	f := d.Firmas[0]
	r.Formato, r.VinculoOriginal = d.Formato, true
	r.CertificadoHuellaSHA256, r.FirmanteRef = f.CertificadoHuellaSHA256, "ref:"+f.CertificadoHuellaSHA256
	r.RevocacionEstado, r.SelloTiempoEstado = f.Revocacion.Estado, f.SelloTiempo.Estado
	return motivar(r, ports.MotivoFirmaVerificada)
}

package httpinterno

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/auditoriafirma"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmasv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	RutaDescargaFirmaR5V2            = "/api/vec/contratacion-temporal/firmas-documento/descargas-v2"
	EsquemaDescargaFirmaR5V2         = "vec.contratacion-temporal.descarga-firma-r5.v2"
	maximoRespuestaDescargaFirmaR5V2 = 2 << 20
)

type manejadorDescargaFirmaR5V2 struct {
	servicio  *consultafirmasv2.ServicioDescarga
	auditoria *manejadorConsultaFirmasR5V2
	garantia  FuenteGarantiaAutenticacionFirmaV2
}

func NuevoManejadorDescargaFirmaR5V2(servicio *consultafirmasv2.ServicioDescarga,
	intentos vecports.RegistradorIntentosAuditoria, fabrica auditoriafirma.FabricaOrden,
	garantia FuenteGarantiaAutenticacionFirmaV2) (http.Handler, error) {
	if servicio == nil || dependenciaNula(intentos) || dependenciaNula(fabrica) || dependenciaNula(garantia) {
		return nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	return &manejadorDescargaFirmaR5V2{servicio: servicio,
		auditoria: &manejadorConsultaFirmasR5V2{intentos: intentos, fabrica: fabrica}, garantia: garantia}, nil
}

type entradaDescargaFirmaR5V2 struct {
	ExpedienteRef    string `json:"expediente_ref"`
	VersionObservada uint64 `json:"version_observada"`
	Documento        string `json:"documento"`
	FirmaRef         string `json:"firma_ref"`
}

func (e *entradaDescargaFirmaR5V2) UnmarshalJSON(contenido []byte) error {
	if err := camposDescargaFirmaR5V2Exactos(contenido); err != nil {
		return err
	}
	type sinMetodo entradaDescargaFirmaR5V2
	return json.Unmarshal(contenido, (*sinMetodo)(e))
}

// La firma seleccionada es una referencia, nunca una terna de custodia.
// También se rechazan duplicados JSON, que json.Unmarshal suele sobrescribir.
func camposDescargaFirmaR5V2Exactos(contenido []byte) error {
	d := json.NewDecoder(bytes.NewReader(contenido))
	primero, err := d.Token()
	if err != nil {
		return fmt.Errorf("%w: %w", errEntradaConsultaRRHHInvalida, err)
	}
	if primero != json.Delim('{') {
		return errEntradaConsultaRRHHInvalida
	}
	esperados := map[string]bool{"expediente_ref": false, "version_observada": false,
		"documento": false, "firma_ref": false}
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return fmt.Errorf("%w: %w", errEntradaConsultaRRHHInvalida, err)
		}
		clave, ok := token.(string)
		if !ok {
			return errEntradaConsultaRRHHInvalida
		}
		visto, conocido := esperados[clave]
		if !conocido || visto {
			return errEntradaConsultaRRHHInvalida
		}
		var valor json.RawMessage
		if err := d.Decode(&valor); err != nil {
			return fmt.Errorf("%w: %w", errEntradaConsultaRRHHInvalida, err)
		}
		if string(valor) == "null" {
			return errEntradaConsultaRRHHInvalida
		}
		esperados[clave] = true
	}
	ultimo, err := d.Token()
	if err != nil {
		return fmt.Errorf("%w: %w", errEntradaConsultaRRHHInvalida, err)
	}
	if ultimo != json.Delim('}') {
		return errEntradaConsultaRRHHInvalida
	}
	var sobra any
	if err := d.Decode(&sobra); !errors.Is(err, io.EOF) {
		if err != nil {
			return fmt.Errorf("%w: %w", errEntradaConsultaRRHHInvalida, err)
		}
		return errEntradaConsultaRRHHInvalida
	}
	for _, visto := range esperados {
		if !visto {
			return errEntradaConsultaRRHHInvalida
		}
	}
	return nil
}

type descargaFirmaR5V2JSON struct {
	Esquema          string `json:"esquema"`
	PDFBase64        string `json:"pdf_base64"`
	ExpedienteRef    string `json:"expediente_ref"`
	VersionObservada uint64 `json:"version_observada"`
	Documento        string `json:"documento"`
	FirmaRef         string `json:"firma_ref"`
	ReciboRef        string `json:"recibo_ref"`
	DocumentoRef     string `json:"documento_ref"`
	DocumentoVersion uint64 `json:"documento_version"`
	HuellaSHA256     string `json:"huella_sha256"`
}

func (h *manejadorDescargaFirmaR5V2) fallo(w http.ResponseWriter, r *http.Request,
	recurso string, causa error, problema errorPublicoConsultaRRHH) {
	if h == nil || h.auditoria == nil {
		responderErrorPreflightFirma(w, r, causa, errorServicioConsultaRRHHNoDisponible)
		return
	}
	h.auditoria.responderFallo(w, r, recurso, causa, problema)
}

func (h *manejadorDescargaFirmaR5V2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.servicio == nil {
		h.fallo(w, r, "", nil, errorServicioConsultaRRHHNoDisponible)
		return
	}
	if !rutaConsultaRRHHExacta(r, RutaDescargaFirmaR5V2) {
		h.fallo(w, r, "", nil, errorRecursoConsultaRRHHNoEncontrado)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		h.fallo(w, r, "", nil, errorMetodoConsultaRRHHNoPermitido)
		return
	}
	if _, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(r.Context()); err != nil {
		h.fallo(w, r, "", err, errorServicioConsultaRRHHNoDisponible)
		return
	}
	if err := r.Context().Err(); err != nil {
		h.fallo(w, r, "", err, clasificarErrorConsultaRRHH(err))
		return
	}
	if problema := validarMetadatosConsultaRRHH(r, MaximoCuerpoConsultaDetalleRRHHBytes); problema != nil {
		h.fallo(w, r, "", nil, *problema)
		return
	}
	var entrada entradaDescargaFirmaR5V2
	if err := decodificarConsultaRRHH(w, r, MaximoCuerpoConsultaDetalleRRHHBytes, &entrada); err != nil {
		h.fallo(w, r, "", err, errorEntradaConsultaRRHH(err))
		return
	}
	q := consultafirmasv2.SolicitudDescarga{ExpedienteRef: entrada.ExpedienteRef,
		VersionObservada: entrada.VersionObservada, Documento: entrada.Documento, FirmaRef: entrada.FirmaRef}
	if q.Validar() != nil {
		h.fallo(w, r, "", nil, errorContenidoConsultaRRHHNoValido)
		return
	}
	garantia, err := garantiaAutenticacionFirmaV2(r.Context(), h.garantia)
	if err != nil || garantia != "sustancial" {
		h.fallo(w, r, q.ExpedienteRef, ports.ErrFirmaDocumentoDenegada, errorRecursoConsultaRRHHNoEncontrado)
		return
	}
	resultado, err := h.servicio.Descargar(r.Context(), q)
	if err != nil {
		h.fallo(w, r, q.ExpedienteRef, err, errorPreflightFirmaR5(err))
		return
	}
	defer clear(resultado.PDF)
	vista := descargaFirmaR5V2JSON{Esquema: EsquemaDescargaFirmaR5V2,
		PDFBase64:     base64.StdEncoding.EncodeToString(resultado.PDF),
		ExpedienteRef: resultado.ExpedienteRef, VersionObservada: resultado.VersionObservada,
		Documento: resultado.Documento, FirmaRef: resultado.FirmaRef, ReciboRef: resultado.ReciboRef,
		DocumentoRef: resultado.DocumentoRef, DocumentoVersion: resultado.DocumentoVersion,
		HuellaSHA256: resultado.HuellaSHA256}
	envoltorio := struct {
		Data descargaFirmaR5V2JSON `json:"data"`
	}{vista}
	contenido, err := json.Marshal(envoltorio)
	if err != nil || len(contenido) > maximoRespuestaDescargaFirmaR5V2 {
		h.fallo(w, r, q.ExpedienteRef, ports.ErrResultadoFirmaDocumentoInvalido, errorResultadoConsultaRRHHNoConfiable)
		return
	}
	responderJSONFirmaNominal(w, r, http.StatusOK, envoltorio, maximoRespuestaDescargaFirmaR5V2)
}

var _ http.Handler = (*manejadorDescargaFirmaR5V2)(nil)

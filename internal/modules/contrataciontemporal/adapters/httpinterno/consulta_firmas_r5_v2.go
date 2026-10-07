package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/auditoriafirma"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmasv2"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	RutaConsultaFirmasR5V2    = "/api/vec/contratacion-temporal/firmas-documento/consultas-v2"
	EsquemaConsultaFirmasR5V2 = "vec.contratacion-temporal.consulta-firmas-r5.v2"
)

// Esta composición exige fuente nominal auditada y emisor V2: no crea ninguna
// identidad ni concesión. Los fallos anteriores al lector requieren orden y
// acuse de auditoría común antes de devolver el error HTTP. Si aún no se conoce
// el recurso, la fábrica recibe "" y resuelve uno gobernado desde el contexto;
// nunca inventa actor, ámbito ni recurso. Sin ese contexto se devuelve 503.
// El lector queda envuelto para auditar errores sólo después de terminar la TX.
// Construir este handler no publica la ruta ni declara un recorrido acreditado.
func NuevoManejadorConsultaFirmasR5V2(fuente consultafirmasv2.FuenteContexto, autorizador ctports.AutorizadorConsultaFirmasR5V2,
	registro ctports.RegistroFirmasVerificadasV2, intentos vecports.RegistradorIntentosAuditoria,
	fabrica auditoriafirma.FabricaOrden) (http.Handler, error) {
	if dependenciaNula(fuente) || dependenciaNula(autorizador) || dependenciaNula(registro) || dependenciaNula(intentos) || dependenciaNula(fabrica) {
		return nil, ctports.ErrRegistroFirmaDocumentoNoDisponible
	}
	auditado, err := auditoriafirma.Nuevo(registro, intentos, fabrica)
	if err != nil {
		return nil, err
	}
	servicio, err := consultafirmasv2.Nuevo(fuente, autorizador, auditado)
	if err != nil {
		return nil, err
	}
	return &manejadorConsultaFirmasR5V2{servicio, intentos, fabrica}, nil
}

type manejadorConsultaFirmasR5V2 struct {
	servicio *consultafirmasv2.Servicio
	intentos vecports.RegistradorIntentosAuditoria
	fabrica  auditoriafirma.FabricaOrden
}

func (h *manejadorConsultaFirmasR5V2) responderFallo(w http.ResponseWriter, r *http.Request, recurso string, causa error, problema errorPublicoConsultaRRHH) {
	if !consultafirmasv2.CubiertoPorLector(causa) {
		if err := h.auditarFallo(r, recurso, causa, problema); err != nil {
			causa, problema = err, errorServicioConsultaRRHHNoDisponible
		}
	}
	responderErrorPreflightFirma(w, r, causa, problema)
}

func (h *manejadorConsultaFirmasR5V2) auditarFallo(r *http.Request, recurso string, causa error, problema errorPublicoConsultaRRHH) error {
	if h == nil || r == nil || dependenciaNula(h.intentos) || dependenciaNula(h.fabrica) {
		return ctports.ErrRegistroFirmaDocumentoNoDisponible
	}
	referenciaCorrelacion, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(r.Context())
	if err != nil {
		return ctports.ErrRegistroFirmaDocumentoNoDisponible
	}
	correlacion, err := referenciaCorrelacion.ValorCanonico()
	if err != nil {
		return ctports.ErrRegistroFirmaDocumentoNoDisponible
	}
	intento, err := vecports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return ctports.ErrRegistroFirmaDocumentoNoDisponible
	}
	resultado := vecdomain.ResultadoIntentoAuditoriaError
	if errors.Is(causa, ctports.ErrFirmaDocumentoDenegada) || errors.Is(causa, ctports.ErrAutorizacionDenegada) || problema.estado == http.StatusForbidden {
		resultado = vecdomain.ResultadoIntentoAuditoriaDenegado
	}
	ctx, cancelar := context.WithTimeout(context.WithoutCancel(r.Context()), plazoarranque.Ampliar(2*time.Second))
	defer cancelar()
	orden, err := h.fabrica.CrearOrdenIntentoFirma(ctx, intento, ctports.AccionConsultarFirmasR5V2, recurso, resultado)
	if err != nil {
		return ctports.ErrRegistroFirmaDocumentoNoDisponible
	}
	d, err := orden.Datos()
	if err != nil || d.IntentoRef != intento || d.Datos.Accion != ctports.AccionConsultarFirmasR5V2 ||
		d.Datos.ModuloID != ctports.ModuloContratacion || d.Datos.Resultado != resultado ||
		!ctdomain.ReferenciaOpacaValida(d.Datos.RecursoRef) ||
		(recurso != "" && d.Datos.RecursoRef != recurso) || d.Datos.CorrelacionRef != correlacion {
		return ctports.ErrRegistroFirmaDocumentoNoDisponible
	}
	acuse, err := h.intentos.AppendIntentoAuditoria(ctx, orden)
	if err != nil || acuse.ValidarPara(orden) != nil {
		return ctports.ErrRegistroFirmaDocumentoNoDisponible
	}
	return nil
}

type entradaConsultaFirmasR5V2 struct {
	ExpedienteRef     string `json:"expediente_ref"`
	VersionExpediente uint64 `json:"version_expediente"`
	Documento         string `json:"documento"`
	PasoOrden         int    `json:"paso_orden"`
	ClaveIdempotencia string `json:"clave_idempotencia"`
	CatalogoHuella    string `json:"catalogo_huella"`
	Via               string `json:"via"`
}

func (e *entradaConsultaFirmasR5V2) UnmarshalJSON(contenido []byte) error {
	if !camposFirmaNominalExactos(contenido, "expediente_ref", "version_expediente", "documento", "paso_orden", "clave_idempotencia", "catalogo_huella", "via") {
		return errEntradaConsultaRRHHInvalida
	}
	type sinMetodo entradaConsultaFirmasR5V2
	return json.Unmarshal(contenido, (*sinMetodo)(e))
}

func (h *manejadorConsultaFirmasR5V2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.servicio == nil {
		h.responderFallo(w, r, "", nil, errorServicioConsultaRRHHNoDisponible)
		return
	}
	if !rutaConsultaRRHHExacta(r, RutaConsultaFirmasR5V2) {
		h.responderFallo(w, r, "", nil, errorRecursoConsultaRRHHNoEncontrado)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		h.responderFallo(w, r, "", nil, errorMetodoConsultaRRHHNoPermitido)
		return
	}
	if err := r.Context().Err(); err != nil {
		h.responderFallo(w, r, "", err, clasificarErrorConsultaRRHH(err))
		return
	}
	if problema := validarMetadatosConsultaRRHH(r, MaximoCuerpoConsultaDetalleRRHHBytes); problema != nil {
		h.responderFallo(w, r, "", nil, *problema)
		return
	}
	var entrada entradaConsultaFirmasR5V2
	if err := decodificarConsultaRRHH(w, r, MaximoCuerpoConsultaDetalleRRHHBytes, &entrada); err != nil {
		h.responderFallo(w, r, "", err, errorEntradaConsultaRRHH(err))
		return
	}
	q := consultafirmasv2.Solicitud{ExpedienteRef: entrada.ExpedienteRef, VersionExpediente: entrada.VersionExpediente,
		Documento: entrada.Documento, PasoOrden: entrada.PasoOrden, ClaveIdempotencia: entrada.ClaveIdempotencia,
		CatalogoHuella: entrada.CatalogoHuella, Via: entrada.Via}
	if q.Validar() != nil {
		h.responderFallo(w, r, "", nil, errorContenidoConsultaRRHHNoValido)
		return
	}
	resultado, err := h.servicio.Consultar(r.Context(), q)
	if err != nil {
		h.responderFallo(w, r, q.ExpedienteRef, err, errorPreflightFirmaR5(err))
		return
	}
	envoltorio := struct {
		Data consultaFirmasR5V2JSON `json:"data"`
	}{proyectarConsultaFirmasR5V2(resultado)}
	// El helper común no audita su propio desbordamiento. Comprobar esta
	// proyección inmutable antes de invocarlo conserva el acuse del error.
	contenido, err := json.Marshal(envoltorio)
	if err != nil || len(contenido) > MaximoRespuestaConsultaRRHHBytes {
		h.responderFallo(w, r, q.ExpedienteRef, ctports.ErrResultadoFirmaDocumentoInvalido, errorResultadoConsultaRRHHNoConfiable)
		return
	}
	responderJSONFirmaNominal(w, r, http.StatusOK, envoltorio, MaximoRespuestaConsultaRRHHBytes)
}

type documentoConsultaFirmasV2JSON struct {
	DocumentoRef string `json:"documento_ref"`
	Version      uint64 `json:"version"`
	SHA256       string `json:"huella_sha256"`
}

type revisionConsultaFirmasV2JSON struct {
	OrdenFirma             int                           `json:"orden_firma"`
	FirmaAnteriorRef       string                        `json:"firma_anterior_ref"`
	ReciboAnteriorRef      string                        `json:"recibo_anterior_ref"`
	Entrada                documentoConsultaFirmasV2JSON `json:"entrada_documento"`
	EntradaLongitud        uint64                        `json:"entrada_longitud"`
	ByteRange              [4]uint64                     `json:"byte_range"`
	RevisionSHA256         string                        `json:"revision_sha256"`
	ContenidoFirmadoSHA256 string                        `json:"contenido_firmado_sha256"`
	RevisionLongitud       uint64                        `json:"revision_longitud"`
	EvidenciaFirmasSHA256  string                        `json:"evidencia_firmas_sha256"`
}

type firmaConsultaR5V2JSON struct {
	FirmaRef            string                        `json:"firma_ref"`
	ReciboRef           string                        `json:"recibo_ref"`
	RegistradaEn        string                        `json:"registrada_en"`
	Secuencia           int                           `json:"secuencia"`
	PasoOrden           int                           `json:"paso_orden"`
	PasoRef             string                        `json:"paso_ref"`
	VersionExpediente   uint64                        `json:"version_expediente"`
	Via                 string                        `json:"via"`
	Resultado           string                        `json:"resultado"`
	CatalogoRef         string                        `json:"catalogo_ref"`
	CatalogoHuella      string                        `json:"catalogo_huella"`
	Original            documentoConsultaFirmasV2JSON `json:"original"`
	DocumentoCustodiado *documentoCustodiadoJSON      `json:"documento_custodiado"`
	RevisionPDF         *revisionConsultaFirmasV2JSON `json:"revision_pdf"`
}

type consultaFirmasR5V2JSON struct {
	Esquema             string                  `json:"esquema"`
	ExpedienteRef       string                  `json:"expediente_ref"`
	VersionExpediente   uint64                  `json:"version_expediente"`
	Documento           string                  `json:"documento"`
	HistoriaRevision    uint64                  `json:"historia_revision"`
	HistoriaSHA256      string                  `json:"historia_sha256"`
	Firmas              []firmaConsultaR5V2JSON `json:"firmas"`
	Recuperacion        string                  `json:"recuperacion"`
	CamposNoDisponibles []string                `json:"campos_no_disponibles"`
	FirmaEficaz         bool                    `json:"firma_eficaz"`
}

func documentoConsultaV2(d consultafirmasv2.Documento) documentoConsultaFirmasV2JSON {
	return documentoConsultaFirmasV2JSON{d.Ref, d.Version, d.SHA256}
}

func proyectarConsultaFirmasR5V2(r consultafirmasv2.Resultado) consultaFirmasR5V2JSON {
	salida := consultaFirmasR5V2JSON{Esquema: EsquemaConsultaFirmasR5V2, ExpedienteRef: r.ExpedienteRef,
		VersionExpediente: r.VersionExpediente, Documento: r.Documento, HistoriaRevision: r.HistoriaRevision, HistoriaSHA256: r.HistoriaHuella,
		Firmas: make([]firmaConsultaR5V2JSON, 0, len(r.Firmas)), Recuperacion: "parcial",
		CamposNoDisponibles: []string{"material_root_sha256", "canon_nominal"}, FirmaEficaz: false}
	for _, f := range r.Firmas {
		fila := firmaConsultaR5V2JSON{FirmaRef: f.FirmaRef, ReciboRef: f.ReciboRef, RegistradaEn: f.RegistradaEn.UTC().Format(time.RFC3339Nano),
			Secuencia: f.Secuencia, PasoOrden: f.PasoOrden, PasoRef: f.PasoRef, VersionExpediente: f.VersionExpediente,
			Via: f.Via, Resultado: string(f.Resultado), CatalogoRef: f.CatalogoRef, CatalogoHuella: f.CatalogoHuella, Original: documentoConsultaV2(f.Original)}
		if f.Custodiado != nil {
			fila.DocumentoCustodiado = &documentoCustodiadoJSON{ExpedienteRef: r.ExpedienteDocumentalRef,
				DocumentoRef: f.Custodiado.Ref, Version: f.Custodiado.Version, HuellaSHA256: f.Custodiado.SHA256}
		}
		if v := f.RevisionPDF; v != nil {
			fila.RevisionPDF = &revisionConsultaFirmasV2JSON{OrdenFirma: v.OrdenFirma, FirmaAnteriorRef: v.FirmaAnteriorRef,
				ReciboAnteriorRef: v.ReciboAnteriorRef, Entrada: documentoConsultaV2(v.Entrada), EntradaLongitud: v.EntradaLongitud,
				ByteRange: v.ByteRange, RevisionSHA256: v.RevisionSHA256, ContenidoFirmadoSHA256: v.ContenidoFirmadoSHA256,
				RevisionLongitud: v.RevisionLongitud, EvidenciaFirmasSHA256: v.EvidenciaFirmasSHA256}
		}
		salida.Firmas = append(salida.Firmas, fila)
	}
	return salida
}

var _ http.Handler = (*manejadorConsultaFirmasR5V2)(nil)
var _ consultafirmasv2.Lector = (*auditoriafirma.Registro)(nil)

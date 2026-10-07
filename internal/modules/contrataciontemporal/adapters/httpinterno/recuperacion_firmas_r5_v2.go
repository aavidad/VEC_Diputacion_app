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
	RutaRecuperacionFirmasR5V2          = "/api/vec/contratacion-temporal/firmas-documento/recuperaciones-v2"
	EsquemaRecuperacionFirmasR5V2       = "vec.contratacion-temporal.recuperacion-firmas-r5.v2"
	maximoRespuestaRecuperacionFirmasV2 = 8 << 20
)

// La composición exige fuente nominal, decisión de 48 campos y auditoría común.
// Construir el handler no publica la ruta.
func NuevoManejadorRecuperacionFirmasR5V2(fuente consultafirmasv2.FuenteContexto, autorizador ctports.AutorizadorRecuperacionFirmasV2,
	lector ctports.LectorRecuperacionFirmasV2, intentos vecports.RegistradorIntentosAuditoria,
	fabrica auditoriafirma.FabricaOrden) (http.Handler, error) {
	if dependenciaNula(fuente) || dependenciaNula(autorizador) || dependenciaNula(lector) || dependenciaNula(intentos) || dependenciaNula(fabrica) {
		return nil, ctports.ErrRegistroFirmaDocumentoNoDisponible
	}
	auditado, err := auditoriafirma.NuevaRecuperacion(lector, intentos, fabrica)
	if err != nil {
		return nil, err
	}
	servicio, err := consultafirmasv2.NuevaRecuperacion(fuente, autorizador, auditado)
	if err != nil {
		return nil, err
	}
	return &manejadorRecuperacionFirmasR5V2{servicio: servicio, intentos: intentos, fabrica: fabrica}, nil
}

type manejadorRecuperacionFirmasR5V2 struct {
	servicio *consultafirmasv2.ServicioRecuperacion
	intentos vecports.RegistradorIntentosAuditoria
	fabrica  auditoriafirma.FabricaOrden
}

func (h *manejadorRecuperacionFirmasR5V2) responderFallo(w http.ResponseWriter, r *http.Request, recurso string, causa error, problema errorPublicoConsultaRRHH) {
	if !consultafirmasv2.CubiertoPorLector(causa) {
		if err := h.auditarFallo(r, recurso, causa, problema); err != nil {
			causa, problema = err, errorServicioConsultaRRHHNoDisponible
		}
	}
	responderErrorPreflightFirma(w, r, causa, problema)
}

func (h *manejadorRecuperacionFirmasR5V2) auditarFallo(r *http.Request, recurso string, causa error, problema errorPublicoConsultaRRHH) error {
	if h == nil || r == nil || dependenciaNula(h.intentos) || dependenciaNula(h.fabrica) {
		return ctports.ErrRegistroFirmaDocumentoNoDisponible
	}
	ref, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(r.Context())
	if err != nil {
		return ctports.ErrRegistroFirmaDocumentoNoDisponible
	}
	correlacion, err := ref.ValorCanonico()
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
	orden, err := h.fabrica.CrearOrdenIntentoFirma(ctx, intento, ctports.AccionRecuperarFirmasR5V2, recurso, resultado)
	if err != nil {
		return ctports.ErrRegistroFirmaDocumentoNoDisponible
	}
	d, err := orden.Datos()
	if err != nil || d.IntentoRef != intento || d.Datos.Accion != ctports.AccionRecuperarFirmasR5V2 ||
		d.Datos.ModuloID != ctports.ModuloContratacion || d.Datos.Resultado != resultado ||
		!ctdomain.ReferenciaOpacaValida(d.Datos.RecursoRef) || (recurso != "" && d.Datos.RecursoRef != recurso) ||
		d.Datos.CorrelacionRef != correlacion {
		return ctports.ErrRegistroFirmaDocumentoNoDisponible
	}
	acuse, err := h.intentos.AppendIntentoAuditoria(ctx, orden)
	if err != nil || acuse.ValidarPara(orden) != nil {
		return ctports.ErrRegistroFirmaDocumentoNoDisponible
	}
	return nil
}

func (h *manejadorRecuperacionFirmasR5V2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.servicio == nil {
		h.responderFallo(w, r, "", nil, errorServicioConsultaRRHHNoDisponible)
		return
	}
	if !rutaConsultaRRHHExacta(r, RutaRecuperacionFirmasR5V2) {
		h.responderFallo(w, r, "", nil, errorRecursoConsultaRRHHNoEncontrado)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		h.responderFallo(w, r, "", nil, errorMetodoConsultaRRHHNoPermitido)
		return
	}
	if _, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(r.Context()); err != nil {
		h.responderFallo(w, r, "", err, errorServicioConsultaRRHHNoDisponible)
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
	resultado, err := h.servicio.Recuperar(r.Context(), q)
	if err != nil {
		h.responderFallo(w, r, q.ExpedienteRef, err, errorPreflightFirmaR5(err))
		return
	}
	vista := proyectarRecuperacionFirmasR5V2(resultado)
	envoltorio := struct {
		Data recuperacionFirmasR5V2JSON `json:"data"`
	}{vista}
	contenido, err := json.Marshal(envoltorio)
	if err != nil || len(contenido) > maximoRespuestaRecuperacionFirmasV2 {
		h.responderFallo(w, r, q.ExpedienteRef, ctports.ErrResultadoFirmaDocumentoInvalido, errorResultadoConsultaRRHHNoConfiable)
		return
	}
	responderJSONFirmaNominal(w, r, http.StatusOK, envoltorio, maximoRespuestaRecuperacionFirmasV2)
}

func proyectarRecuperacionFirmasR5V2(resultado consultafirmasv2.ResultadoRecuperacion) recuperacionFirmasR5V2JSON {
	base := proyectarConsultaFirmasR5V2(resultado.Resultado)
	base.Esquema = EsquemaRecuperacionFirmasR5V2
	base.Recuperacion = "recuperada"
	base.CamposNoDisponibles = []string{}
	recuperaciones := make([]recuperacionFirmaR5V2JSON, 0, len(resultado.Recuperaciones))
	for _, v := range resultado.Recuperaciones {
		recuperaciones = append(recuperaciones, recuperacionFirmaR5V2JSON{
			FirmaRef: v.FirmaRef, MaterialRootSHA256: v.MaterialRootSHA256, CanonNominal: v.CanonNominal,
			CanonNominalSHA256: v.CanonNominalSHA256, CanonNominalRef: v.CanonNominalRef})
	}
	return recuperacionFirmasR5V2JSON{consultaFirmasR5V2JSON: base, Recuperaciones: recuperaciones}
}

type recuperacionFirmaR5V2JSON struct {
	FirmaRef           string `json:"firma_ref"`
	MaterialRootSHA256 string `json:"material_root_sha256"`
	CanonNominal       string `json:"canon_nominal"`
	CanonNominalSHA256 string `json:"canon_nominal_sha256"`
	CanonNominalRef    string `json:"canon_nominal_ref"`
}

type recuperacionFirmasR5V2JSON struct {
	consultaFirmasR5V2JSON
	Recuperaciones []recuperacionFirmaR5V2JSON `json:"recuperaciones"`
}

var _ http.Handler = (*manejadorRecuperacionFirmasR5V2)(nil)
var _ ctports.LectorRecuperacionFirmasV2 = (*auditoriafirma.Recuperacion)(nil)

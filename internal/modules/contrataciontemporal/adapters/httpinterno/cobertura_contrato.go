package httpinterno

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const MaximoCuerpoCoberturaBytes = 64 * 1024

var (
	errEntradaCoberturaInvalida       = errors.New("contratacion temporal http: entrada de cobertura invalida")
	errContenidoCoberturaNoValido     = errors.New("contratacion temporal http: contenido de cobertura no valido")
	errCuerpoCoberturaDemasiadoGrande = errors.New("contratacion temporal http: cuerpo de cobertura demasiado grande")
)

type propuestaCoberturaJSON struct {
	ExpedienteRef   string `json:"expediente_ref"`
	VersionEsperada uint64 `json:"version_esperada"`
}
type decisionCoberturaJSON struct {
	ExpedienteRef      string                                              `json:"expediente_ref"`
	VersionEsperada    uint64                                              `json:"version_esperada"`
	ClaveIdempotencia  string                                              `json:"clave_idempotencia"`
	IdentidadSemantica domain.IdentidadSemanticaPropuestaDecisionCobertura `json:"identidad_semantica"`
	ViaElegida         string                                              `json:"via_elegida"`
	MotivoClave        string                                              `json:"motivo_clave"`
	PredecesoraRef     string                                              `json:"predecesora_ref"`
	PredecesoraHuella  string                                              `json:"predecesora_huella"`
}

func validarMetadatosCobertura(r *http.Request) *errorPublicoCobertura {
	if r == nil || r.ContentLength > MaximoCuerpoCoberturaBytes || r.ContentLength == 0 || r.Body == nil || r.Body == http.NoBody || len(r.Trailer) != 0 || !transferenciaAltaPermitida(r.TransferEncoding) {
		problema := errorPeticionCoberturaNoValida
		return &problema
	}
	if !tipoContenidoJSON(r.Header) {
		problema := errorTipoCoberturaNoAdmitido
		return &problema
	}
	if !acceptCompatibleJSON(r.Header) {
		problema := errorRepresentacionCoberturaNoAceptable
		return &problema
	}
	if cabeceraCoberturaProhibida(r.Header) {
		problema := errorPeticionCoberturaNoPermitida
		return &problema
	}
	return nil
}

func cabeceraCoberturaProhibida(cabeceras http.Header) bool {
	for nombre := range cabeceras {
		minusculas := strings.ToLower(nombre)
		switch {
		case minusculas == "authorization", minusculas == "cookie", minusculas == "set-cookie", minusculas == "proxy-authorization", minusculas == "proxy-connection", minusculas == "forwarded", minusculas == "remote-user", minusculas == "x-remote-user", minusculas == "x-forwarded-user", minusculas == "idempotency-key", minusculas == "content-encoding", minusculas == "trailer", minusculas == "te", minusculas == "expect", minusculas == "x-http-method-override", cabeceraAutoridadLibreAlta(minusculas), strings.Contains(minusculas, "role"), strings.HasPrefix(minusculas, "x-auth-"), strings.HasPrefix(minusculas, "x-vec-"), strings.HasPrefix(minusculas, "x-forwarded-"), strings.HasPrefix(minusculas, "x-envoy-"):
			return true
		}
	}
	return false
}

// propuestaCoberturaDesdePeticion usa POST aunque no produzca efectos: así la
// referencia opaca y la versión no viajan en URL, query ni cabeceras.
func propuestaCoberturaDesdePeticion(w http.ResponseWriter, r *http.Request) (propuestaCoberturaJSON, error) {
	var entrada propuestaCoberturaJSON
	if err := decodificarCobertura(w, r, &entrada); err != nil {
		return propuestaCoberturaJSON{}, err
	}
	if !domain.ReferenciaOpacaValida(entrada.ExpedienteRef) || entrada.VersionEsperada == 0 || entrada.VersionEsperada >= cobertura.MaximoEnteroSeguroOperacionDecisionCobertura {
		return propuestaCoberturaJSON{}, errContenidoCoberturaNoValido
	}
	return entrada, nil
}

func decisionCoberturaDesdePeticion(w http.ResponseWriter, r *http.Request, rectificacion bool) (decisionCoberturaJSON, error) {
	var entrada decisionCoberturaJSON
	if err := decodificarCobertura(w, r, &entrada); err != nil {
		return decisionCoberturaJSON{}, err
	}
	if !domain.ReferenciaOpacaValida(entrada.ExpedienteRef) || entrada.VersionEsperada == 0 || entrada.VersionEsperada >= cobertura.MaximoEnteroSeguroOperacionDecisionCobertura || !ports.ClaveIdempotenciaValida(entrada.ClaveIdempotencia) || entrada.IdentidadSemantica.Validar() != nil || !domain.ClaveCatalogo(entrada.ViaElegida).Valida() || (entrada.MotivoClave != "" && !domain.ClaveCatalogo(entrada.MotivoClave).Valida()) {
		return decisionCoberturaJSON{}, errContenidoCoberturaNoValido
	}
	if rectificacion {
		if !domain.ReferenciaOpacaValida(entrada.PredecesoraRef) || !huellaCoberturaValida(entrada.PredecesoraHuella) || !domain.ClaveCatalogo(entrada.MotivoClave).Valida() {
			return decisionCoberturaJSON{}, errContenidoCoberturaNoValido
		}
	} else if entrada.PredecesoraRef != "" || entrada.PredecesoraHuella != "" {
		return decisionCoberturaJSON{}, errContenidoCoberturaNoValido
	}
	return entrada, nil
}

func (e decisionCoberturaJSON) solicitud(c ContextoCanalCobertura) application.SolicitudDecidirCobertura {
	return application.SolicitudDecidirCobertura{AutenticacionRef: c.AutenticacionRef, SesionRef: c.SesionRef, PerfilRef: c.PerfilRef, OrganizacionRef: c.OrganizacionRef, ExpedienteRef: e.ExpedienteRef, VersionEsperada: e.VersionEsperada, ClaveIdempotencia: e.ClaveIdempotencia, IdentidadSemantica: e.IdentidadSemantica, ViaElegida: domain.ClaveCatalogo(e.ViaElegida), MotivoClave: domain.ClaveCatalogo(e.MotivoClave)}
}
func (e decisionCoberturaJSON) rectificacion(c ContextoCanalCobertura) application.SolicitudRectificarCobertura {
	return application.SolicitudRectificarCobertura{AutenticacionRef: c.AutenticacionRef, SesionRef: c.SesionRef, PerfilRef: c.PerfilRef, OrganizacionRef: c.OrganizacionRef, ExpedienteRef: e.ExpedienteRef, VersionEsperada: e.VersionEsperada, ClaveIdempotencia: e.ClaveIdempotencia, IdentidadSemantica: e.IdentidadSemantica, ViaElegida: domain.ClaveCatalogo(e.ViaElegida), MotivoClave: domain.ClaveCatalogo(e.MotivoClave), PredecesoraRef: e.PredecesoraRef, PredecesoraHuella: e.PredecesoraHuella}
}

func decodificarCobertura(w http.ResponseWriter, r *http.Request, destino any) error {
	lector := http.MaxBytesReader(w, r.Body, MaximoCuerpoCoberturaBytes+1)
	contenido, err := io.ReadAll(lector)
	if err != nil {
		var limite *http.MaxBytesError
		if errors.As(err, &limite) {
			return errCuerpoCoberturaDemasiadoGrande
		}
		return errEntradaCoberturaInvalida
	}
	if len(contenido) == 0 || !utf8.Valid(contenido) {
		return errEntradaCoberturaInvalida
	}
	if len(contenido) > MaximoCuerpoCoberturaBytes {
		return errCuerpoCoberturaDemasiadoGrande
	}
	if err := validarJSONAltaSinDuplicados(contenido); err != nil {
		if errors.Is(err, errCuerpoAltaDemasiadoGrande) {
			return errCuerpoCoberturaDemasiadoGrande
		}
		return errEntradaCoberturaInvalida
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return errEntradaCoberturaInvalida
	}
	if err := decodificador.Decode(&struct{}{}); err != io.EOF {
		return errEntradaCoberturaInvalida
	}
	return nil
}

func huellaCoberturaValida(valor string) bool {
	if len(valor) != 64 || strings.Trim(valor, "0") == "" {
		return false
	}
	for _, caracter := range valor {
		if !strings.ContainsRune("0123456789abcdef", caracter) {
			return false
		}
	}
	return true
}

type envoltorioPropuestaCobertura struct {
	Data propuestaCoberturaSalidaJSON `json:"data"`
}
type propuestaCoberturaSalidaJSON struct {
	Esquema            string                          `json:"esquema"`
	Estado             string                          `json:"estado"`
	ViaRecomendada     string                          `json:"via_recomendada"`
	Evaluaciones       []evaluacionCoberturaJSON       `json:"evaluaciones"`
	IdentidadSemantica identidadSemanticaCoberturaJSON `json:"identidad_semantica"`
}
type evaluacionCoberturaJSON struct {
	ViaClave             string   `json:"via_clave"`
	Prioridad            uint16   `json:"prioridad"`
	Estado               string   `json:"estado"`
	ResultadosOmitidos   []string `json:"resultados_omitidos"`
	AusenciasBloqueantes []string `json:"ausencias_bloqueantes"`
	AusenciasAdmitidas   []string `json:"ausencias_admitidas"`
	NoHabilitantes       []string `json:"no_habilitantes"`
	Conflictos           []string `json:"conflictos"`
}
type identidadSemanticaCoberturaJSON struct {
	Referencia   string                    `json:"referencia"`
	HuellaSHA256 string                    `json:"huella_sha256"`
	Canon        dominioCanonCoberturaJSON `json:"canon"`
}
type dominioCanonCoberturaJSON struct {
	Dominio        string `json:"dominio"`
	VersionEsquema uint16 `json:"version_esquema"`
	Algoritmo      string `json:"algoritmo"`
}

func proyectarPropuestaCobertura(entrada application.PresentacionPropuestaCobertura) (propuestaCoberturaSalidaJSON, bool) {
	if entrada.IdentidadSemantica.Validar() != nil || !estadoPropuestaCoberturaValido(entrada.Estado) || !entrada.ViaRecomendada.Valida() || len(entrada.Evaluaciones) == 0 || len(entrada.Evaluaciones) > 64 {
		return propuestaCoberturaSalidaJSON{}, false
	}
	salida := propuestaCoberturaSalidaJSON{Esquema: "vec.contratacion-temporal.propuesta-cobertura.v1", Estado: string(entrada.Estado), ViaRecomendada: string(entrada.ViaRecomendada), IdentidadSemantica: identidadSemanticaCoberturaJSON{Referencia: entrada.IdentidadSemantica.Referencia, HuellaSHA256: entrada.IdentidadSemantica.HuellaSHA256, Canon: dominioCanonCoberturaJSON{Dominio: entrada.IdentidadSemantica.Canon.Dominio, VersionEsquema: entrada.IdentidadSemantica.Canon.VersionEsquema, Algoritmo: entrada.IdentidadSemantica.Canon.Algoritmo}}}
	for _, evaluacion := range entrada.Evaluaciones {
		if !evaluacion.ViaClave.Valida() || evaluacion.Prioridad == 0 || !estadoEvaluacionCoberturaValido(evaluacion.Estado) {
			return propuestaCoberturaSalidaJSON{}, false
		}
		resultadosOmitidos, ok := clavesCobertura(evaluacion.ResultadosOmitidos)
		if !ok {
			return propuestaCoberturaSalidaJSON{}, false
		}
		ausenciasBloqueantes, ok := clavesCobertura(evaluacion.AusenciasBloqueantes)
		if !ok {
			return propuestaCoberturaSalidaJSON{}, false
		}
		ausenciasAdmitidas, ok := clavesCobertura(evaluacion.AusenciasAdmitidas)
		if !ok {
			return propuestaCoberturaSalidaJSON{}, false
		}
		noHabilitantes, ok := clavesCobertura(evaluacion.NoHabilitantes)
		if !ok {
			return propuestaCoberturaSalidaJSON{}, false
		}
		conflictos, ok := clavesCobertura(evaluacion.Conflictos)
		if !ok {
			return propuestaCoberturaSalidaJSON{}, false
		}
		salida.Evaluaciones = append(salida.Evaluaciones, evaluacionCoberturaJSON{ViaClave: string(evaluacion.ViaClave), Prioridad: evaluacion.Prioridad, Estado: string(evaluacion.Estado), ResultadosOmitidos: resultadosOmitidos, AusenciasBloqueantes: ausenciasBloqueantes, AusenciasAdmitidas: ausenciasAdmitidas, NoHabilitantes: noHabilitantes, Conflictos: conflictos})
	}
	return salida, true
}

func estadoPropuestaCoberturaValido(estado domain.EstadoPropuestaDecisionCobertura) bool {
	return estado == domain.PropuestaCoberturaViable || estado == domain.PropuestaCoberturaIncompleta || estado == domain.PropuestaCoberturaConflictiva || estado == domain.PropuestaCoberturaSinVia
}

func estadoEvaluacionCoberturaValido(estado domain.EstadoEvaluacionViaCobertura) bool {
	return estado == domain.EvaluacionViaCoberturaViable || estado == domain.EvaluacionViaCoberturaIncompleta || estado == domain.EvaluacionViaCoberturaConflictiva || estado == domain.EvaluacionViaCoberturaNoViable
}

func clavesCobertura(entrada []domain.ClaveCatalogo) ([]string, bool) {
	salida := make([]string, len(entrada))
	for indice, clave := range entrada {
		if !clave.Valida() {
			return nil, false
		}
		salida[indice] = string(clave)
	}
	return salida, true
}

type envoltorioReciboCobertura struct {
	Data reciboCoberturaJSON `json:"data"`
}
type reciboCoberturaJSON struct {
	Esquema              string `json:"esquema"`
	ReciboRef            string `json:"recibo_ref"`
	Estado               string `json:"estado"`
	DecisionCoberturaRef string `json:"decision_cobertura_ref,omitempty"`
	VersionResultante    uint64 `json:"version_resultante,omitempty"`
	ConfirmadaEn         string `json:"confirmada_en"`
}

func proyectarReciboCobertura(entrada cobertura.ReciboOperacionDecisionCobertura) (reciboCoberturaJSON, bool) {
	if !domain.ReferenciaOpacaValida(entrada.ReciboRef) || !domain.InstanteUTCCanonico(entrada.ConfirmadaEn) || (entrada.Aplicada == nil) == (entrada.DenegadaVEC == nil) {
		return reciboCoberturaJSON{}, false
	}
	salida := reciboCoberturaJSON{Esquema: "vec.contratacion-temporal.recibo-cobertura.v1", ReciboRef: entrada.ReciboRef, ConfirmadaEn: entrada.ConfirmadaEn.UTC().Format(time.RFC3339Nano)}
	if aplicada, ok := entrada.ResultadoAplicado(); ok {
		if !domain.ReferenciaOpacaValida(aplicada.DecisionCoberturaRef) || aplicada.VersionResultante == 0 || aplicada.VersionResultante > cobertura.MaximoEnteroSeguroOperacionDecisionCobertura {
			return reciboCoberturaJSON{}, false
		}
		salida.Estado = "aplicada"
		salida.DecisionCoberturaRef = aplicada.DecisionCoberturaRef
		salida.VersionResultante = aplicada.VersionResultante
		return salida, true
	}
	if _, ok := entrada.ResultadoDenegadoVEC(); ok {
		salida.Estado = "denegada"
		return salida, true
	}
	return reciboCoberturaJSON{}, false
}

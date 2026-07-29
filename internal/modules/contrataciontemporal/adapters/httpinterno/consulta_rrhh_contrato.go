package httpinterno

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	MaximoCuerpoConsultaCuadroRRHHBytes  = 4 * 1024
	MaximoCuerpoConsultaDetalleRRHHBytes = 4 * 1024
	MaximoRespuestaConsultaRRHHBytes     = 256 * 1024

	esquemaConsultaCuadroRRHH  = "vec.contratacion-temporal.cuadro-rrhh.v1"
	esquemaConsultaDetalleRRHH = "vec.contratacion-temporal.detalle-rrhh.v1"
)

var (
	errEntradaConsultaRRHHInvalida       = errors.New("contratacion temporal http: entrada de consulta RRHH invalida")
	errContenidoConsultaRRHHNoValido     = errors.New("contratacion temporal http: contenido de consulta RRHH no valido")
	errCuerpoConsultaRRHHDemasiadoGrande = errors.New("contratacion temporal http: cuerpo de consulta RRHH demasiado grande")
)

type filtrosCuadroRRHHJSON struct {
	Texto       string `json:"texto"`
	EstadoClave string `json:"estado_clave"`
	FaseClave   string `json:"fase_clave"`
}

type paginacionCuadroRRHHJSON struct {
	Limite *uint16 `json:"limite"`
	Cursor string  `json:"cursor"`
}

type consultaCuadroRRHHJSON struct {
	Filtros    *filtrosCuadroRRHHJSON    `json:"filtros"`
	Paginacion *paginacionCuadroRRHHJSON `json:"paginacion"`
}

type consultaDetalleRRHHJSON struct {
	ExpedienteRef    string  `json:"expediente_ref"`
	VersionObservada *uint64 `json:"version_observada"`
}

func validarMetadatosConsultaRRHH(
	r *http.Request,
	maximo int64,
) *errorPublicoConsultaRRHH {
	if r != nil && r.ContentLength > maximo {
		problema := errorCuerpoConsultaRRHHDemasiadoGrande
		return &problema
	}
	if r == nil || r.ContentLength == 0 || r.Body == nil ||
		r.Body == http.NoBody || len(r.Trailer) != 0 ||
		!transferenciaAltaPermitida(r.TransferEncoding) {
		problema := errorPeticionConsultaRRHHNoValida
		return &problema
	}
	if !cabeceraJSONConsultaRRHHExacta(r.Header, "Content-Type") {
		problema := errorTipoConsultaRRHHNoAdmitido
		return &problema
	}
	if !cabeceraJSONConsultaRRHHExacta(r.Header, "Accept") {
		problema := errorRepresentacionConsultaRRHHNoAceptable
		return &problema
	}
	if cabeceraCoberturaProhibida(r.Header) {
		problema := errorPeticionConsultaRRHHNoPermitida
		return &problema
	}
	return nil
}

func cabeceraJSONConsultaRRHHExacta(
	cabeceras http.Header,
	nombre string,
) bool {
	valor, unico := cabeceraUnicaAlta(cabeceras, nombre)
	if !unico || strings.Contains(valor, ",") {
		return false
	}
	tipo, parametros, err := mime.ParseMediaType(valor)
	return err == nil && strings.EqualFold(tipo, "application/json") &&
		len(parametros) == 0
}

func solicitudCuadroRRHHDesdePeticion(
	w http.ResponseWriter,
	r *http.Request,
) (ports.SolicitudCuadroRRHH, error) {
	var entrada consultaCuadroRRHHJSON
	if err := decodificarConsultaRRHH(
		w, r, MaximoCuerpoConsultaCuadroRRHHBytes, &entrada,
	); err != nil {
		return ports.SolicitudCuadroRRHH{}, err
	}
	if entrada.Filtros == nil || entrada.Paginacion == nil ||
		entrada.Paginacion.Limite == nil {
		return ports.SolicitudCuadroRRHH{}, errContenidoConsultaRRHHNoValido
	}
	solicitud, err := ports.NuevaSolicitudCuadroRRHH(
		entrada.Filtros.Texto,
		domain.EstadoOperativo(entrada.Filtros.EstadoClave),
		domain.ClaveFase(entrada.Filtros.FaseClave),
		*entrada.Paginacion.Limite,
		entrada.Paginacion.Cursor,
	)
	if err != nil {
		return ports.SolicitudCuadroRRHH{}, errContenidoConsultaRRHHNoValido
	}
	return solicitud, nil
}

func solicitudDetalleRRHHDesdePeticion(
	w http.ResponseWriter,
	r *http.Request,
) (ports.SolicitudDetalleRRHH, error) {
	var entrada consultaDetalleRRHHJSON
	if err := decodificarConsultaRRHH(
		w, r, MaximoCuerpoConsultaDetalleRRHHBytes, &entrada,
	); err != nil {
		return ports.SolicitudDetalleRRHH{}, err
	}
	if entrada.VersionObservada == nil {
		return ports.SolicitudDetalleRRHH{}, errContenidoConsultaRRHHNoValido
	}
	solicitud, err := ports.NuevaSolicitudDetalleRRHH(
		entrada.ExpedienteRef,
		*entrada.VersionObservada,
	)
	if err != nil {
		return ports.SolicitudDetalleRRHH{}, errContenidoConsultaRRHHNoValido
	}
	return solicitud, nil
}

func decodificarConsultaRRHH(
	w http.ResponseWriter,
	r *http.Request,
	maximo int64,
	destino any,
) error {
	lector := http.MaxBytesReader(w, r.Body, maximo+1)
	contenido, err := io.ReadAll(lector)
	if err != nil {
		var limite *http.MaxBytesError
		if errors.As(err, &limite) {
			return errCuerpoConsultaRRHHDemasiadoGrande
		}
		return errEntradaConsultaRRHHInvalida
	}
	recortado := bytes.TrimSpace(contenido)
	if len(contenido) > int(maximo) {
		return errCuerpoConsultaRRHHDemasiadoGrande
	}
	if len(recortado) < 2 || recortado[0] != '{' ||
		recortado[len(recortado)-1] != '}' || !utf8.Valid(contenido) {
		return errEntradaConsultaRRHHInvalida
	}
	if err := validarJSONAltaSinDuplicados(contenido); err != nil {
		if errors.Is(err, errCuerpoAltaDemasiadoGrande) {
			return errCuerpoConsultaRRHHDemasiadoGrande
		}
		return errEntradaConsultaRRHHInvalida
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return errEntradaConsultaRRHHInvalida
	}
	if err := decodificador.Decode(&struct{}{}); err != io.EOF {
		return errEntradaConsultaRRHHInvalida
	}
	return nil
}

type envoltorioCuadroRRHH struct {
	Data paginaCuadroRRHHJSON `json:"data"`
}

type paginaCuadroRRHHJSON struct {
	Esquema         string            `json:"esquema"`
	GeneradaEn      string            `json:"generada_en"`
	Expedientes     []resumenRRHHJSON `json:"expedientes"`
	HayMas          bool              `json:"hay_mas"`
	CursorSiguiente string            `json:"cursor_siguiente,omitempty"`
}

type resumenRRHHJSON struct {
	ExpedienteRef  string `json:"expediente_ref"`
	NumeroVisible  string `json:"numero_visible"`
	Version        uint64 `json:"version"`
	FlujoRef       string `json:"flujo_ref"`
	FlujoVersion   uint64 `json:"flujo_version"`
	FlujoHuella    string `json:"flujo_huella_sha256"`
	FaseClave      string `json:"fase_clave"`
	EstadoClave    string `json:"estado_clave"`
	CentroRef      string `json:"centro_ref"`
	CategoriaRef   string `json:"categoria_ref"`
	ModalidadClave string `json:"modalidad_clave,omitempty"`
	UnidadRef      string `json:"unidad_ref,omitempty"`
	CreadoEn       string `json:"creado_en"`
	ActualizadoEn  string `json:"actualizado_en"`
}

func proyectarPaginaCuadroRRHH(
	entrada ports.PaginaCuadroRRHH,
) paginaCuadroRRHHJSON {
	salida := paginaCuadroRRHHJSON{
		Esquema:         esquemaConsultaCuadroRRHH,
		GeneradaEn:      instanteConsultaRRHH(entrada.GeneradaEn),
		Expedientes:     make([]resumenRRHHJSON, len(entrada.Expedientes)),
		HayMas:          entrada.HayMas,
		CursorSiguiente: entrada.CursorSiguiente,
	}
	for indice, resumen := range entrada.Expedientes {
		salida.Expedientes[indice] = proyectarResumenRRHH(resumen)
	}
	return salida
}

// paginaConsultaRRHHPublicable no suplanta la validación probatoria del caso
// de uso. Impide que una implementación defectuosa de la interfaz publique
// campos cero, referencias repetidas o una cardinalidad fuera de contrato.
func paginaConsultaRRHHPublicable(entrada ports.PaginaCuadroRRHH) bool {
	if !domain.InstanteUTCCanonico(entrada.GeneradaEn) ||
		len(entrada.Expedientes) > ports.LimiteMaximoCuadroRRHH ||
		entrada.HayMas != (entrada.CursorSiguiente != "") {
		return false
	}
	vistas := make(map[string]struct{}, len(entrada.Expedientes))
	for _, resumen := range entrada.Expedientes {
		if resumen.Validar() != nil ||
			resumen.ActualizadoEn.After(entrada.GeneradaEn) {
			return false
		}
		if _, repetida := vistas[resumen.ExpedienteRef]; repetida {
			return false
		}
		vistas[resumen.ExpedienteRef] = struct{}{}
	}
	return true
}

func proyectarResumenRRHH(entrada ports.ResumenExpedienteRRHH) resumenRRHHJSON {
	return resumenRRHHJSON{
		ExpedienteRef: entrada.ExpedienteRef, NumeroVisible: entrada.NumeroVisible,
		Version: entrada.Version, FlujoRef: entrada.FlujoRef,
		FlujoVersion: entrada.FlujoVersion, FlujoHuella: entrada.FlujoHuella,
		FaseClave: string(entrada.FaseClave), EstadoClave: string(entrada.EstadoClave),
		CentroRef: entrada.CentroRef, CategoriaRef: entrada.CategoriaRef,
		ModalidadClave: string(entrada.ModalidadClave), UnidadRef: entrada.UnidadRef,
		CreadoEn:      instanteConsultaRRHH(entrada.CreadoEn),
		ActualizadoEn: instanteConsultaRRHH(entrada.ActualizadoEn),
	}
}

func instanteConsultaRRHH(instante time.Time) string {
	return instante.UTC().Format(time.RFC3339Nano)
}

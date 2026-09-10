package httpinterno

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	RutaIncorporacionEjercicioV2 = "/api/vec/contratacion-temporal/incorporaciones-ejercicio"
	// 32 referencias de hasta 160 caracteres no caben en 4096 bytes.
	MaximoCuerpoIncorporacionEjercicioV2Bytes   = 8192
	MaximoConsultaIncorporacionEjercicioV2Bytes = 600
	EsquemaPreparacionIncorporacionEjercicioV2  = "vec.contratacion-temporal.incorporacion-ejercicio.preparacion.v2"
	EsquemaReciboIncorporacionEjercicioV2       = "vec.contratacion-temporal.incorporacion-ejercicio.recibo.v2"
)

var errPeticionIncorporacionEjercicioV2 = errors.New("contratacion temporal http: peticion de incorporacion invalida")

// EntradaIncorporacionEjercicioV2 es exclusivamente el wire del navegador.
// No contiene material, identidad, resultado Personal, raiz ni autoridad.
type EntradaIncorporacionEjercicioV2 = ports.IntencionIncorporacionAplicacionV2

func leerEntradaIncorporacionEjercicioV2(w http.ResponseWriter, r *http.Request) (EntradaIncorporacionEjercicioV2, error) {
	var cero EntradaIncorporacionEjercicioV2
	if r == nil || r.Body == nil || r.Body == http.NoBody || r.ContentLength == 0 ||
		r.ContentLength > MaximoCuerpoIncorporacionEjercicioV2Bytes || len(r.Trailer) != 0 ||
		!transferenciaAltaPermitida(r.TransferEncoding) || !tipoContenidoJSON(r.Header) {
		return cero, errPeticionIncorporacionEjercicioV2
	}
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaximoCuerpoIncorporacionEjercicioV2Bytes))
	if r.Context().Err() != nil {
		return cero, r.Context().Err()
	}
	if err != nil || len(b) == 0 || !utf8.Valid(b) || (r.ContentLength >= 0 && int64(len(b)) != r.ContentLength) ||
		validarJSONPropuestaFormalizacionSinDuplicados(b) != nil {
		return cero, errPeticionIncorporacionEjercicioV2
	}
	var campos map[string]json.RawMessage
	if json.Unmarshal(b, &campos) != nil || len(campos) != 7 {
		return cero, errPeticionIncorporacionEjercicioV2
	}
	for _, clave := range []string{"expediente_ref", "version_actual_expediente_observada", "solicitud_personal_ref", "motivo_clave", "documentos_refs", "confirma_revision_personal", "confirma_ejercicio_sintetico"} {
		if _, ok := campos[clave]; !ok {
			return cero, errPeticionIncorporacionEjercicioV2
		}
	}
	var entrada EntradaIncorporacionEjercicioV2
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&entrada) != nil || d.Decode(&struct{}{}) != io.EOF || entrada.DocumentosRefs == nil || len(entrada.DocumentosRefs) > 32 {
		return cero, errPeticionIncorporacionEjercicioV2
	}
	return entrada, nil
}

func leerConsultaIncorporacionEjercicioV2(r *http.Request) (string, error) {
	if r == nil || r.URL == nil || len(r.URL.RawQuery) > MaximoConsultaIncorporacionEjercicioV2Bytes ||
		r.ContentLength != 0 || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 {
		return "", errPeticionIncorporacionEjercicioV2
	}
	if r.Body != nil && r.Body != http.NoBody {
		b, err := io.ReadAll(io.LimitReader(r.Body, 1))
		if r.Context().Err() != nil {
			return "", r.Context().Err()
		}
		if err != nil || len(b) != 0 {
			return "", errPeticionIncorporacionEjercicioV2
		}
	}
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(q) != 1 || len(q["expediente_ref"]) != 1 || q.Get("expediente_ref") == "" {
		return "", errPeticionIncorporacionEjercicioV2
	}
	return q.Get("expediente_ref"), nil
}

func versionIncorporacionHTTPValida(v uint64, admiteCero bool) bool {
	return (admiteCero || v > 0) && v <= ports.MaximoEnteroSeguroOperacionAnalisis
}

// Defensas del wire, no prueba de persistencia. La procedencia y el cotejo del
// material original son obligaciones del servicio confiable, nunca del canal.
func reciboIncorporacionHTTPValido(r ports.ReciboIncorporacionAplicacionV2, expediente string) bool {
	if r.Esquema != EsquemaReciboIncorporacionEjercicioV2 || r.ExpedienteRef != expediente ||
		!r.EjercicioSintetico || r.FirmaOficial || r.EficaciaAdministrativa ||
		!domain.InstanteUTCCanonico(r.RegistradaEn) || r.Periodo.Validar() != nil ||
		!versionIncorporacionHTTPValida(r.VersionSolicitudPersonal, false) ||
		!versionIncorporacionHTTPValida(r.VersionActualExpediente, false) ||
		!versionIncorporacionHTTPValida(r.VersionSeguimientoAnterior, true) ||
		!versionIncorporacionHTTPValida(r.VersionSeguimientoResultante, false) ||
		r.VersionSeguimientoResultante != r.VersionSeguimientoAnterior+1 {
		return false
	}
	for _, ref := range []string{r.ExpedienteRef, r.SolicitudPersonalRef, r.RelacionRef, r.ReciboRef,
		r.ActuacionRef, r.SeguimientoRef, r.AuditoriaRef, r.OutboxRef} {
		if !domain.ReferenciaOpacaValida(ref) {
			return false
		}
	}
	return true
}

func proyeccionIncorporacionHTTPValida(p ports.ProyeccionIncorporacionAplicacionV2, expediente string) bool {
	if p.Esquema != EsquemaPreparacionIncorporacionEjercicioV2 || p.ExpedienteRef != expediente ||
		!domain.ReferenciaOpacaValida(p.ExpedienteRef) || !versionIncorporacionHTTPValida(p.VersionActualExpediente, false) ||
		(p.Preparacion == nil) == (p.Recibo == nil) {
		return false
	}
	if p.Recibo != nil {
		// La version del recibo es ORIGINAL, no se sustituye por la actual.
		return reciboIncorporacionHTTPValido(*p.Recibo, expediente) && p.Recibo.VersionActualExpediente <= p.VersionActualExpediente
	}
	v := p.Preparacion
	if !domain.ReferenciaOpacaValida(v.SolicitudPersonalRef) || !versionIncorporacionHTTPValida(v.VersionSolicitudPersonal, false) ||
		!versionIncorporacionHTTPValida(v.VersionSeguimientoEsperada, true) || v.Periodo.Validar() != nil ||
		len(v.Motivos) > 32 || len(v.DocumentosRefs) > 32 || (v.Disponible && len(v.Motivos) == 0) {
		return false
	}
	vistos := map[string]bool{}
	for _, motivo := range v.Motivos {
		m := string(motivo)
		if !motivo.Valida() || vistos[m] {
			return false
		}
		vistos[m] = true
	}
	vistos = map[string]bool{}
	for _, ref := range v.DocumentosRefs {
		if !domain.ReferenciaOpacaValida(ref) || vistos[ref] {
			return false
		}
		vistos[ref] = true
	}
	return true
}

func copiarProyeccionIncorporacionHTTP(p ports.ProyeccionIncorporacionAplicacionV2) ports.ProyeccionIncorporacionAplicacionV2 {
	p = p.Copia()
	if p.Preparacion != nil {
		p.Preparacion.Motivos = append([]domain.ClaveCatalogo{}, p.Preparacion.Motivos...)
		p.Preparacion.DocumentosRefs = append([]string{}, p.Preparacion.DocumentosRefs...)
	}
	return p
}

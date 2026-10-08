package interna

import (
	"context"
	"net/http"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const plazoRechazoSesionC4 = 2 * time.Second

func esRutaSesionC4(ruta string) bool {
	return ruta == domain.RutaSesionActual || ruta == domain.RutaInicioSesion
}

func metodoSesionC4(ruta string) string {
	if ruta == domain.RutaSesionActual {
		return http.MethodGet
	}
	if ruta == domain.RutaInicioSesion {
		return http.MethodPost
	}
	return ""
}

// El alias se registra antes de que otra fuente vea la petición. El acuse
// procede del registrador AD222 inyectado; no se atribuye actor aún.
func (p *puenteConsultaSeguimiento) AuditarRechazoSesionNoCanonica(ctx context.Context, ruta string) error {
	return p.registrarRechazoSesionC4(ctx, ruta, ports.MotivoFronteraIdentidadSolicitudInvalida)
}

func (p *puenteConsultaSeguimiento) registrarRechazoSesionC4(
	ctx context.Context, ruta string, motivo ports.MotivoFronteraIdentidadTecnica,
) error {
	if p == nil || interfazNulaIdentidadOffline(p.auditorSesion) || ctx == nil || metodoSesionC4(ruta) == "" {
		return ports.ErrFronteraIdentidadTecnicaNoDisponible
	}
	orden, err := ports.NuevaOrdenFronteraIdentidadTecnica(metodoSesionC4(ruta), ruta, motivo)
	if err != nil {
		return err
	}
	ctxAuditoria, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoRechazoSesionC4)
	defer cancelar()
	acuse, err := p.auditorSesion.RegistrarRechazoInicioSesion(ctxAuditoria, orden)
	if err != nil || acuse.CorrelacionRef != orden.CorrelacionRef() ||
		acuse.AuditoriaRef == "" || acuse.Secuencia == 0 ||
		acuse.MaterialSHA256 == "" || acuse.RegistradaEn.IsZero() {
		return ports.ErrFronteraIdentidadTecnicaNoDisponible
	}
	return nil
}

func (p *puenteConsultaSeguimiento) responderRechazoSesionC4(
	w http.ResponseWriter, ctx context.Context, ruta string,
	motivo ports.MotivoFronteraIdentidadTecnica, estado int,
) {
	if p.registrarRechazoSesionC4(ctx, ruta, motivo) != nil {
		responderPuenteSeguimiento(w, http.StatusServiceUnavailable)
		return
	}
	if estado == http.StatusMethodNotAllowed {
		w.Header().Set("Allow", metodoSesionC4(ruta))
	}
	responderPuenteSeguimiento(w, estado)
}

// atenderSesionC4 consume la capacidad TLS del servidor una sola vez. La
// lectura usa exclusivamente reanudación; sólo el POST explícito puede abrir.
func (p *puenteConsultaSeguimiento) atenderSesionC4(w http.ResponseWriter, r *http.Request) {
	ruta := r.URL.Path
	if p.presentacionCertificado == nil || interfazNulaIdentidadOffline(p.auditorSesion) || p.fachada.Load() == nil {
		responderPuenteSeguimiento(w, http.StatusServiceUnavailable)
		return
	}
	if r.URL.RawPath != "" || r.URL.Opaque != "" || r.URL.RawQuery != "" ||
		r.URL.EscapedPath() != ruta || r.URL.Fragment != "" || r.URL.RawFragment != "" ||
		r.URL.ForceQuery || r.URL.Scheme != "" || r.URL.Host != "" || r.URL.User != nil ||
		(r.RequestURI != "" && r.RequestURI != r.URL.RequestURI()) {
		p.responderRechazoSesionC4(w, r.Context(), ruta, ports.MotivoFronteraIdentidadSolicitudInvalida, http.StatusBadRequest)
		return
	}
	if r.Method != metodoSesionC4(ruta) {
		p.responderRechazoSesionC4(w, r.Context(), ruta, ports.MotivoFronteraIdentidadMetodoNoPermitido, http.StatusMethodNotAllowed)
		return
	}
	if r.Body != http.NoBody || r.ContentLength > 0 || len(r.TransferEncoding) != 0 ||
		!cabecerasSesionC4Validas(r.Header) {
		p.responderRechazoSesionC4(w, r.Context(), ruta, ports.MotivoFronteraIdentidadSolicitudInvalida, http.StatusBadRequest)
		return
	}
	if r.TLS == nil {
		p.responderRechazoSesionC4(w, r.Context(), ruta, ports.MotivoFronteraIdentidadCertificadoRequerido, http.StatusUnauthorized)
		return
	}
	preparada, err := httpseguridad.PrepararPeticionAsercionPasarela(r, 0)
	if err != nil {
		p.responderRechazoSesionC4(w, r.Context(), ruta, ports.MotivoFronteraIdentidadSolicitudInvalida, http.StatusBadRequest)
		return
	}
	defer preparada.Body.Close()
	asercion, err := p.extractor.ExtraerAsercionProtegida(preparada)
	if err != nil || len(asercion) == 0 {
		clear(asercion)
		p.responderRechazoSesionC4(w, preparada.Context(), ruta, ports.MotivoFronteraIdentidadAutenticacionRequerida, http.StatusUnauthorized)
		return
	}
	defer clear(asercion)
	capacidad, ok := preparada.Context().Value(claveContextoCanalTLSInterno{}).(*capacidadCanalTLSInterno)
	fachada := p.fachada.Load()
	if !ok || fachada == nil {
		p.responderRechazoSesionC4(w, preparada.Context(), ruta, ports.MotivoFronteraIdentidadCertificadoRequerido, http.StatusUnauthorized)
		return
	}
	estadoTLS, consumida := capacidad.consumir(fachada.propietario)
	if !consumida {
		p.responderRechazoSesionC4(w, preparada.Context(), ruta, ports.MotivoFronteraIdentidadCertificadoRequerido, http.StatusUnauthorized)
		return
	}
	ctx, canal, prueba, err := p.presentacionCertificado.AcreditarCertificadoActual(preparada.Context(), estadoTLS)
	if err != nil {
		p.responderRechazoSesionC4(w, preparada.Context(), ruta, ports.MotivoFronteraIdentidadAutenticacionRequerida, http.StatusUnauthorized)
		return
	}
	credencial, err := httpseguridad.NuevaCredencialProxy(asercion, canal)
	if err != nil {
		p.responderRechazoSesionC4(w, ctx, ruta, ports.MotivoFronteraIdentidadSolicitudInvalida, http.StatusBadRequest)
		return
	}
	var capsula httpseguridad.CapsulaPresentacionCertificado
	if ruta == domain.RutaInicioSesion {
		capsula, err = p.presentacionCertificado.IniciarYConsumirPresentacion(ctx, credencial, prueba)
	} else {
		capsula, err = p.presentacionCertificado.ReanudarYConsumirPresentacion(ctx, credencial, prueba)
	}
	if err != nil {
		p.responderRechazoSesionC4(w, ctx, ruta, ports.MotivoFronteraIdentidadServicioNoDisponible, http.StatusServiceUnavailable)
		return
	}
	if _, err := p.presentacionCertificado.VincularCapsulaPresentacion(ctx, capsula, canal); err != nil {
		p.responderRechazoSesionC4(w, ctx, ruta, ports.MotivoFronteraIdentidadRespuestaIncompatible, http.StatusServiceUnavailable)
		return
	}
	responderPuenteSeguimiento(w, http.StatusNoContent)
}

func cabecerasSesionC4Validas(h http.Header) bool {
	if !cabecerasCertificadoPersonalValidas(h) {
		return false
	}
	for nombre := range h {
		nombre = strings.ToLower(nombre)
		if nombre == "proxy-connection" || strings.HasPrefix(nombre, "x-proxy-") ||
			strings.HasPrefix(nombre, "x-original-") || strings.HasPrefix(nombre, "x-cert-") ||
			strings.HasPrefix(nombre, "x-tls-") || strings.HasPrefix(nombre, "x-vec-") ||
			strings.HasPrefix(nombre, "x-profile") ||
			strings.HasPrefix(nombre, "x-role") || strings.HasPrefix(nombre, "x-actor") ||
			strings.HasPrefix(nombre, "x-session") || strings.HasPrefix(nombre, "x-nonce") {
			return false
		}
	}
	return true
}

package httpapi

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	RutaProponerGobiernoCategoriaRPT  = "/api/vec/administracion/catalogos/rpt/gobierno/proponer"
	RutaAprobarGobiernoCategoriaRPT   = "/api/vec/administracion/catalogos/rpt/gobierno/aprobar"
	RutaConfirmarGobiernoCategoriaRPT = "/api/vec/administracion/catalogos/rpt/gobierno/confirmar"
	maximoCuerpoGobiernoCategoriaRPT  = 256 << 10
)

var ErrHandlerGobiernoCategoriaRPTInvalido = errors.New("vec http: frontera de gobierno RPT invalida")

// La fuente pertenece a la composicion ADMIN. Debe enlazar el certificado del
// handshake con actor, vinculo, contexto registrado, motivo y perfil publicados.
// La solicitud HTTP no puede publicar ni provisionar esas capacidades.
type FuenteCredencialesGobiernoCategoriaRPT interface {
	ResolverGobiernoCategoriaRPT(context.Context, *x509.Certificate) (application.CredencialesGobiernoCategoriaRPT, ports.DescriptorCatalogoRPT, string, error)
}

type AuditorDenegacionGobiernoCategoriaRPT interface {
	RegistrarDenegacionGobiernoCategoriaRPT(context.Context, string, string, string) error
}

type OperadorGobiernoCategoriaRPT interface {
	Proponer(context.Context, application.OrdenProponerGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error)
	Aprobar(context.Context, application.OrdenAvanzarGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error)
	Confirmar(context.Context, application.OrdenAvanzarGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error)
}

var _ OperadorGobiernoCategoriaRPT = (*application.ServicioGobiernoCategoriaRPT)(nil)

type handlerGobiernoCategoriaRPT struct {
	operador   OperadorGobiernoCategoriaRPT
	fuente     FuenteCredencialesGobiernoCategoriaRPT
	auditor    AuditorDenegacionGobiernoCategoriaRPT
	adminHost  string
	raices     *x509.CertPool
	descriptor ports.DescriptorCatalogoRPT
	perfilFijo string
}

// Construir estas rutas no las monta en el portal ordinario. #211 debe crear
// un listener ADMIN exclusivo con ClientAuth y entregar su CA privada.
func NuevasRutasGobiernoCategoriaRPT(op OperadorGobiernoCategoriaRPT, fuente FuenteCredencialesGobiernoCategoriaRPT, auditor AuditorDenegacionGobiernoCategoriaRPT, adminHost string, raices *x509.CertPool, descriptor ports.DescriptorCatalogoRPT, perfilFijo string) ([]RutaExacta, error) {
	if dependenciaRutaExactaNula(op) || dependenciaRutaExactaNula(fuente) || dependenciaRutaExactaNula(auditor) ||
		raices == nil || len(raices.Subjects()) == 0 || adminHost == "" || adminHost != strings.ToLower(adminHost) ||
		strings.ContainsAny(adminHost, ":/ \t\r\n") || !strings.Contains(adminHost, ".") ||
		descriptor.CatalogoID == "" || descriptor.ModuloID == "" || perfilFijo == "" {
		return nil, ErrHandlerGobiernoCategoriaRPTInvalido
	}
	h := &handlerGobiernoCategoriaRPT{op, fuente, auditor, adminHost, raices.Clone(), descriptor, perfilFijo}
	return []RutaExacta{
		{Ruta: RutaProponerGobiernoCategoriaRPT, Manejador: h},
		{Ruta: RutaAprobarGobiernoCategoriaRPT, Manejador: h},
		{Ruta: RutaConfirmarGobiernoCategoriaRPT, Manejador: h},
	}, nil
}

type entradaPreimagenGobiernoRPT struct {
	Version      int    `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
	Revision     int64  `json:"revision"`
	Estado       string `json:"estado"`
}

type entradaGobiernoRPT struct {
	PropuestaRef       string                                 `json:"propuesta_ref"`
	Accion             string                                 `json:"accion"`
	CatalogoID         string                                 `json:"catalogo_id"`
	ModuloID           string                                 `json:"modulo_id"`
	Version            int                                    `json:"version"`
	DocumentoCanonico  *string                                `json:"documento_canonico"`
	PreimagenesControl map[string]entradaPreimagenGobiernoRPT `json:"preimagenes_control"`
	CategoriaID        *string                                `json:"categoria_id"`
	RevisionEsperada   *int64                                 `json:"revision_esperada"`
	FuenteRef          string                                 `json:"fuente_ref"`
	HuellaSHA256       string                                 `json:"huella_sha256"` // only for approve/confirm
}

func (h *handlerGobiernoCategoriaRPT) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || r == nil || dependenciaRutaExactaNula(h.operador) || dependenciaRutaExactaNula(h.fuente) || dependenciaRutaExactaNula(h.auditor) {
		responderGobiernoRPT(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	if !peticionRutaExactaCanonica(r) || !rutaGobiernoRPTValida(r.URL.Path) {
		responderGobiernoRPT(w, http.StatusNotFound, "recurso_no_encontrado", nil)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		responderGobiernoRPT(w, http.StatusMethodNotAllowed, "metodo_no_permitido", nil)
		return
	}
	// El host y SNI deben ser ADMIN, y el certificado se verifica de nuevo
	// contra la CA dedicada. Cabeceras de identidad nunca son autoridad.
	if r.Host != h.adminHost || r.TLS == nil || r.TLS.ServerName != h.adminHost ||
		!r.TLS.HandshakeComplete || len(r.TLS.PeerCertificates) == 0 ||
		certificadoClienteTLSVerificado(r.TLS) == nil {
		h.denegar(w, r.Context(), r.URL.Path, http.StatusUnauthorized, "autenticacion_requerida")
		return
	}
	cert := r.TLS.PeerCertificates[0]
	if _, err := cert.Verify(x509.VerifyOptions{Roots: h.raices, Intermediates: intermediosGobiernoRPT(r.TLS.PeerCertificates[1:]), KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, CurrentTime: time.Now()}); err != nil {
		h.denegar(w, r.Context(), r.URL.Path, http.StatusUnauthorized, "autenticacion_requerida")
		return
	}
	cred, descriptor, perfil, err := h.fuente.ResolverGobiernoCategoriaRPT(r.Context(), cert)
	if err != nil {
		if errors.Is(err, ErrAutenticacionRutaExactaRequerida) || errors.Is(err, domain.ErrContextoActorNoResuelto) {
			h.denegar(w, r.Context(), r.URL.Path, http.StatusUnauthorized, "autenticacion_requerida")
		} else if errors.Is(err, ErrAccesoRutaExactaDenegado) || errors.Is(err, domain.ErrAutorizacionDenegada) || errors.Is(err, domain.ErrPermissionDenied) {
			h.denegar(w, r.Context(), r.URL.Path, http.StatusForbidden, "acceso_denegado")
		} else {
			responderGobiernoRPT(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		}
		return
	}
	if perfil != h.perfilFijo || cred.Actor.PerfilActivoRef != h.perfilFijo || descriptor != h.descriptor ||
		!cred.Actor.Principal.AuthAssurance.Cumple(domain.AuthAssuranceHigh) ||
		cred.Actor.Validar() != nil || cred.Vinculo.ValidarPara(cred.ResultadoContexto) != nil ||
		cred.ResultadoContexto.Validar() != nil || !domain.ReferenciaMotivoAutorizacionV2Valida(cred.Motivo) ||
		cred.Correlacion.Validar() != nil {
		h.denegar(w, r.Context(), r.URL.Path, http.StatusForbidden, "acceso_denegado")
		return
	}
	clave, entrada, err := leerEntradaGobiernoRPT(w, r)
	if err != nil || entrada.CatalogoID != h.descriptor.CatalogoID || entrada.ModuloID != h.descriptor.ModuloID {
		responderGobiernoRPT(w, http.StatusBadRequest, "peticion_no_valida", nil)
		return
	}
	var resultado ports.ResultadoGobiernoCategoriaRPT
	switch r.URL.Path {
	case RutaProponerGobiernoCategoriaRPT:
		if entrada.HuellaSHA256 != "" || entrada.Version < 1 || entrada.PreimagenesControl == nil || entrada.Accion == "" || entrada.FuenteRef == "" {
			responderGobiernoRPT(w, http.StatusBadRequest, "peticion_no_valida", nil)
			return
		}
		preimagenes := make(map[string]domain.PreimagenControlGobiernoCategoriaRPT, len(entrada.PreimagenesControl))
		for id, p := range entrada.PreimagenesControl {
			preimagenes[id] = domain.PreimagenControlGobiernoCategoriaRPT{Version: p.Version, HuellaSHA256: p.HuellaSHA256, Revision: p.Revision, Estado: p.Estado}
		}
		b := ports.BorradorPropuestaGobiernoCategoriaRPT{PropuestaRef: entrada.PropuestaRef, ReciboRef: clave,
			Contenido: domain.ContenidoGobiernoCategoriaRPT{Accion: entrada.Accion, CatalogoID: entrada.CatalogoID, ModuloID: entrada.ModuloID, Version: entrada.Version, DocumentoCanonico: entrada.DocumentoCanonico, PreimagenesControl: preimagenes, CategoriaID: entrada.CategoriaID, RevisionEsperada: entrada.RevisionEsperada, MotivoRef: cred.Motivo.Referencia(), FuenteRef: entrada.FuenteRef}}
		resultado, err = h.operador.Proponer(r.Context(), application.OrdenProponerGobiernoCategoriaRPT{Credenciales: cred, Borrador: b})
	case RutaAprobarGobiernoCategoriaRPT, RutaConfirmarGobiernoCategoriaRPT:
		if entrada.Accion != "" || entrada.Version != 0 || entrada.DocumentoCanonico != nil || entrada.PreimagenesControl != nil || entrada.CategoriaID != nil || entrada.RevisionEsperada == nil || entrada.FuenteRef != "" {
			responderGobiernoRPT(w, http.StatusBadRequest, "peticion_no_valida", nil)
			return
		}
		m := ports.MaterialAvanceGobiernoCategoriaRPT{PropuestaRef: entrada.PropuestaRef, HuellaSHA256: entrada.HuellaSHA256, ReciboRef: clave, RevisionEsperada: *entrada.RevisionEsperada, CatalogoID: entrada.CatalogoID, ModuloID: entrada.ModuloID}
		o := application.OrdenAvanzarGobiernoCategoriaRPT{Credenciales: cred, Material: m}
		if r.URL.Path == RutaAprobarGobiernoCategoriaRPT {
			resultado, err = h.operador.Aprobar(r.Context(), o)
		} else {
			resultado, err = h.operador.Confirmar(r.Context(), o)
		}
	}
	if err != nil {
		h.errorOperacion(w, r.Context(), r.URL.Path, err)
		return
	}
	if resultado.PropuestaRef != entrada.PropuestaRef || resultado.ReciboRef != clave || resultado.Evidencia.AuditoriaRef == "" || !resultado.Evidencia.ConsumoNuevo {
		responderGobiernoRPT(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	// ConsumoNuevo describe la autorización V3 de este acceso, no la creación
	// de un efecto nuevo. Tanto primer acto como replay devuelven 200.
	responderGobiernoRPT(w, http.StatusOK, "", map[string]any{"data": map[string]any{
		"propuesta_ref": resultado.PropuestaRef, "huella_sha256": resultado.HuellaSHA256,
		"revision": resultado.Revision, "estado": resultado.Estado, "recibo_ref": resultado.ReciboRef,
		"accion": resultado.Accion, "version": resultado.Version, "revision_categoria": resultado.RevisionCategoria,
		"evidencia": map[string]any{"decision_ref": resultado.Evidencia.DecisionRef,
			"efecto_ref": resultado.Evidencia.EfectoRef, "huella_efecto_sha256": resultado.Evidencia.HuellaEfectoSHA256,
			"consumo_huella_sha256": resultado.Evidencia.ConsumoHuellaSHA256,
			"auditoria_ref":         resultado.Evidencia.AuditoriaRef, "consumida_en": resultado.Evidencia.ConsumidaEn,
			"consumo_nuevo": resultado.Evidencia.ConsumoNuevo},
	}})
}

func rutaGobiernoRPTValida(ruta string) bool {
	return ruta == RutaProponerGobiernoCategoriaRPT || ruta == RutaAprobarGobiernoCategoriaRPT || ruta == RutaConfirmarGobiernoCategoriaRPT
}

func intermediosGobiernoRPT(certificados []*x509.Certificate) *x509.CertPool {
	p := x509.NewCertPool()
	for _, c := range certificados {
		if c != nil {
			p.AddCert(c)
		}
	}
	return p
}

func leerEntradaGobiernoRPT(w http.ResponseWriter, r *http.Request) (string, entradaGobiernoRPT, error) {
	var e entradaGobiernoRPT
	if r.URL.RawQuery != "" || r.Body == nil || r.Body == http.NoBody || r.ContentLength == 0 || r.ContentLength > maximoCuerpoGobiernoCategoriaRPT || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 ||
		cabeceraOrganizacionHistoricaPresente(r.Header, "Cookie") || cabeceraOrganizacionHistoricaPresente(r.Header, "Proxy-Authorization") || cabeceraOrganizacionHistoricaPresente(r.Header, "Content-Encoding") ||
		!cabeceraImportacionOrganizacionExacta(r.Header, "Content-Type", "application/json") {
		return "", e, ErrHandlerGobiernoCategoriaRPTInvalido
	}
	clave, ok := cabeceraImportacionOrganizacionUnica(r.Header, "Idempotency-Key")
	if !ok || !patronClaveHTTPImportacionOrganizacion.MatchString(clave) {
		return "", e, ErrHandlerGobiernoCategoriaRPTInvalido
	}
	cuerpo, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maximoCuerpoGobiernoCategoriaRPT+1))
	if err != nil || len(cuerpo) == 0 || len(cuerpo) > maximoCuerpoGobiernoCategoriaRPT || !bytes.HasPrefix(bytes.TrimSpace(cuerpo), []byte("{")) || validarJSONRegistroEmpleadoB2(cuerpo) != nil {
		return "", e, ErrHandlerGobiernoCategoriaRPTInvalido
	}
	var campos map[string]json.RawMessage
	if json.Unmarshal(cuerpo, &campos) != nil || campos == nil || !camposGobiernoRPTAdmitidos(r.URL.Path, campos) {
		return "", e, ErrHandlerGobiernoCategoriaRPTInvalido
	}
	d := json.NewDecoder(bytes.NewReader(cuerpo))
	d.DisallowUnknownFields()
	if d.Decode(&e) != nil || d.Decode(new(any)) != io.EOF {
		return "", entradaGobiernoRPT{}, ErrHandlerGobiernoCategoriaRPTInvalido
	}
	return clave, e, nil
}

func camposGobiernoRPTAdmitidos(ruta string, campos map[string]json.RawMessage) bool {
	permitidos := map[string]bool{"propuesta_ref": true, "catalogo_id": true, "modulo_id": true}
	if ruta == RutaProponerGobiernoCategoriaRPT {
		for _, campo := range []string{"accion", "version", "documento_canonico", "preimagenes_control", "categoria_id", "revision_esperada", "fuente_ref"} {
			permitidos[campo] = true
		}
	} else {
		permitidos["huella_sha256"] = true
		permitidos["revision_esperada"] = true
	}
	for campo := range campos {
		if !permitidos[campo] {
			return false
		}
	}
	return true
}

func (h *handlerGobiernoCategoriaRPT) denegar(w http.ResponseWriter, ctx context.Context, ruta string, estado int, codigo string) {
	correlacion := nuevaCorrelacionRutaExacta()
	ctxAudit, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoMaximoAuditoriaFronteraRutaExacta)
	defer cancelar()
	if h.auditor.RegistrarDenegacionGobiernoCategoriaRPT(ctxAudit, ruta, codigo, correlacion) != nil {
		responderGobiernoRPT(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	responderGobiernoRPT(w, estado, codigo, nil)
}

func (h *handlerGobiernoCategoriaRPT) errorOperacion(w http.ResponseWriter, ctx context.Context, ruta string, err error) {
	switch {
	case errors.Is(err, ports.ErrGobiernoCategoriaRPTDenegado):
		h.denegar(w, ctx, ruta, http.StatusForbidden, "acceso_denegado")
	case errors.Is(err, application.ErrOrdenGobiernoCategoriaRPTInvalida), errors.Is(err, ports.ErrGobiernoCategoriaRPTInvalido):
		responderGobiernoRPT(w, http.StatusBadRequest, "peticion_no_valida", nil)
	case errors.Is(err, ports.ErrGobiernoCategoriaRPTConflicto):
		responderGobiernoRPT(w, http.StatusConflict, "conflicto", nil)
	default:
		responderGobiernoRPT(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
	}
}

func responderGobiernoRPT(w http.ResponseWriter, estado int, codigo string, data map[string]any) {
	for _, k := range []string{"Set-Cookie", "Access-Control-Allow-Origin", "Access-Control-Allow-Credentials", "Location", "Retry-After"} {
		w.Header().Del(k)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; frame-ancestors 'none'")
	var payload any = data
	if codigo != "" {
		payload = map[string]any{"error": map[string]string{"codigo": codigo, "clave_i18n": "api.vec.catalogos.rpt.gobierno.error." + codigo}}
	}
	contenido, err := json.Marshal(payload)
	if err != nil || payload == nil {
		contenido = []byte("{}")
	}
	w.WriteHeader(estado)
	_, _ = w.Write(contenido)
}

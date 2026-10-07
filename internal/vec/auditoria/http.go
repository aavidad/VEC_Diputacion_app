package auditoria

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const maximoCuerpoConsulta = 8 * 1024

// FuenteConsulta es la unica seleccion de fuente que cruza hacia la frontera
// de identidad. El cliente aporta texto; el handler lo valida antes de usarlo.
type FuenteConsulta string

const (
	FuenteConsultaGeneral FuenteConsulta = ""
	FuenteConsultaCT      FuenteConsulta = "ct"
	FuenteConsultaBolsa   FuenteConsulta = "bolsa"
)

func fuenteConsultaDesdeTexto(valor string) (FuenteConsulta, bool) {
	switch FuenteConsulta(valor) {
	case FuenteConsultaCT, FuenteConsultaBolsa:
		return FuenteConsulta(valor), true
	default:
		return FuenteConsultaGeneral, false
	}
}

// IdentidadConsulta resuelve únicamente desde la frontera de identidad del
// servidor. El navegador nunca proporciona actor, vínculo, perfil o correlación.
type IdentidadConsulta interface {
	ResolverIdentidadConsulta(context.Context, *http.Request, FuenteConsulta) (IdentidadResuelta, error)
}

type IdentidadResuelta struct {
	Vinculo     vecdomain.VinculoAutenticacionActorV2
	Resultado   vecdomain.ResultadoContextoActorRegistradoV2
	Correlacion vecdomain.ReferenciaCorrelacionAutorizacionV2
}

type Manejador struct {
	servicio  *Servicio
	opciones  ProveedorOpciones
	identidad IdentidadConsulta
	ahora     func() time.Time
}

func NuevoManejador(servicio *Servicio, opciones ProveedorOpciones, identidad IdentidadConsulta) (*Manejador, error) {
	if servicio == nil || dependenciaNula(opciones) || dependenciaNula(identidad) {
		return nil, ErrNoDisponible
	}
	return &Manejador{servicio: servicio, opciones: opciones, identidad: identidad, ahora: time.Now}, nil
}

func (h *Manejador) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if h == nil || r == nil || h.servicio == nil || dependenciaNula(h.opciones) || dependenciaNula(h.identidad) {
		responderError(w, http.StatusServiceUnavailable)
		return
	}
	if r.URL.RawQuery != "" || r.URL.Fragment != "" || contieneCabeceraCookie(r.Header) {
		responderError(w, http.StatusForbidden)
		return
	}
	switch r.URL.Path {
	case RutaOpciones:
		if r.Method != http.MethodGet {
			responderError(w, http.StatusMethodNotAllowed)
			return
		}
		h.servirOpciones(w, r)
	case RutaConsulta:
		if r.Method != http.MethodPost {
			responderError(w, http.StatusMethodNotAllowed)
			return
		}
		h.servirConsulta(w, r)
	default:
		responderError(w, http.StatusNotFound)
	}
}

func contieneCabeceraCookie(cabeceras http.Header) bool {
	for nombre := range cabeceras {
		if strings.EqualFold(nombre, "Cookie") {
			return true
		}
	}
	return false
}

func (h *Manejador) resolverIdentidad(r *http.Request, fuente FuenteConsulta) (IdentidadResuelta, error) {
	identidad, err := h.identidad.ResolverIdentidadConsulta(r.Context(), r, fuente)
	if err != nil || identidad.Resultado.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.Resultado) != nil ||
		!identidad.Vinculo.VigenteEn(h.ahora().UTC().Truncate(time.Microsecond), identidad.Resultado) || identidad.Correlacion.Validar() != nil {
		return IdentidadResuelta{}, ErrDenegada
	}
	return identidad, nil
}

func (h *Manejador) servirOpciones(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength != 0 || len(r.TransferEncoding) != 0 {
		responderError(w, http.StatusBadRequest)
		return
	}
	if r.Body != nil {
		_ = r.Body.Close()
	}
	r.Body = http.NoBody
	r.GetBody = nil
	identidad, err := h.resolverIdentidad(r, FuenteConsultaGeneral)
	if err != nil {
		responderError(w, http.StatusForbidden)
		return
	}
	opciones, err := h.opciones.Actuales(r.Context())
	if err != nil {
		h.responderIntento(w, r, identidad, recursoIntentoOpcionesRRHH, http.StatusServiceUnavailable, nil)
		return
	}
	if r.Context().Err() != nil {
		h.responderIntento(w, r, identidad, recursoIntentoOpcionesRRHH, http.StatusServiceUnavailable, &opciones)
		return
	}
	responderJSON(w, http.StatusOK, opciones)
}

type cuerpoConsulta struct {
	Fuente        string `json:"fuente"`
	ExpedienteRef string `json:"expediente_ref"`
	ActorRef      string `json:"actor_ref"`
	Desde         string `json:"desde"`
	Hasta         string `json:"hasta"`
	Limite        uint16 `json:"limite"`
	Cursor        string `json:"cursor"`
	FinalidadRef  string `json:"finalidad_ref"`
	MotivoRef     string `json:"motivo_ref"`
}

// decodificarCuerpoConsulta recorre una sola vez el JSON, rechaza claves
// desconocidas o repetidas y nunca entrega el body al resolvedor de identidad.
func decodificarCuerpoConsulta(w http.ResponseWriter, r *http.Request) (cuerpoConsulta, error) {
	var c cuerpoConsulta
	tipo := r.Header.Get("Content-Type")
	if (tipo != "application/json" && tipo != "application/json; charset=utf-8") ||
		r.ContentLength > maximoCuerpoConsulta {
		return c, ErrDenegada
	}
	lector := http.MaxBytesReader(w, r.Body, maximoCuerpoConsulta)
	defer lector.Close()
	dec := json.NewDecoder(lector)
	inicio, err := dec.Token()
	if err != nil || inicio != json.Delim('{') {
		return c, ErrDenegada
	}
	vistas := make(map[string]bool, 9)
	for dec.More() {
		claveToken, err := dec.Token()
		clave, correcta := claveToken.(string)
		if err != nil || !correcta || vistas[clave] {
			return c, ErrDenegada
		}
		vistas[clave] = true
		var destino *string
		switch clave {
		case "fuente":
			destino = &c.Fuente
		case "expediente_ref":
			destino = &c.ExpedienteRef
		case "actor_ref":
			destino = &c.ActorRef
		case "desde":
			destino = &c.Desde
		case "hasta":
			destino = &c.Hasta
		case "cursor":
			destino = &c.Cursor
		case "finalidad_ref":
			destino = &c.FinalidadRef
		case "motivo_ref":
			destino = &c.MotivoRef
		case "limite":
			var limite *uint16
			if dec.Decode(&limite) != nil || limite == nil {
				return c, ErrDenegada
			}
			c.Limite = *limite
			continue
		default:
			return c, ErrDenegada
		}
		var valor *string
		if dec.Decode(&valor) != nil || valor == nil {
			return c, ErrDenegada
		}
		*destino = *valor
	}
	fin, err := dec.Token()
	if err != nil || fin != json.Delim('}') ||
		!vistas["fuente"] || !vistas["expediente_ref"] || !vistas["desde"] ||
		!vistas["hasta"] || !vistas["finalidad_ref"] || !vistas["motivo_ref"] {
		return c, ErrDenegada
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return c, ErrDenegada
	}
	r.Body = http.NoBody
	r.GetBody = nil
	r.ContentLength = 0
	r.TransferEncoding = nil
	return c, nil
}

func (h *Manejador) servirConsulta(w http.ResponseWriter, r *http.Request) {
	cuerpo, err := decodificarCuerpoConsulta(w, r)
	if err != nil {
		responderError(w, http.StatusBadRequest)
		return
	}
	fuente, valida := fuenteConsultaDesdeTexto(cuerpo.Fuente)
	if !valida {
		responderError(w, http.StatusBadRequest)
		return
	}
	if cuerpo.Limite == 0 {
		cuerpo.Limite = 50
	}
	identidad, err := h.resolverIdentidad(r, fuente)
	if err != nil {
		responderError(w, http.StatusForbidden)
		return
	}
	desde, errDesde := parsearInstante(cuerpo.Desde)
	hasta, errHasta := parsearInstante(cuerpo.Hasta)
	if errDesde != nil || errHasta != nil {
		h.responderIntento(w, r, identidad, recursoIntentoConsultaInvalida, http.StatusBadRequest, nil)
		return
	}
	f := Filtro{Fuente: string(fuente), ExpedienteRef: cuerpo.ExpedienteRef, ActorRef: cuerpo.ActorRef,
		Desde: desde, Hasta: hasta, Limite: cuerpo.Limite,
		FinalidadRef: cuerpo.FinalidadRef, MotivoRef: cuerpo.MotivoRef}
	if f.Validar() != nil {
		h.responderIntento(w, r, identidad, recursoIntentoConsultaInvalida, http.StatusForbidden, nil)
		return
	}
	opciones, err := h.opciones.Actuales(r.Context())
	if err != nil {
		h.responderIntento(w, r, identidad, recursoIntentoFiltro(f), http.StatusServiceUnavailable, nil)
		return
	}
	if cuerpo.FinalidadRef != opciones.FinalidadRef || cuerpo.MotivoRef != opciones.MotivoRef {
		h.responderIntento(w, r, identidad, recursoIntentoFiltro(f), http.StatusForbidden, &opciones)
		return
	}
	pagina, err := h.servicio.Consultar(r.Context(), Peticion{Filtro: f, Cursor: cuerpo.Cursor, Contexto: ContextoConsulta{
		Vinculo: identidad.Vinculo, Resultado: identidad.Resultado,
		Motivo: opciones.Motivo, Correlacion: identidad.Correlacion,
	}})
	if err != nil {
		if errors.Is(err, ErrDenegada) {
			h.responderIntento(w, r, identidad, recursoIntentoFiltro(f), http.StatusForbidden, &opciones)
		} else {
			h.responderIntento(w, r, identidad, recursoIntentoFiltro(f), http.StatusServiceUnavailable, &opciones)
		}
		return
	}
	responderJSON(w, http.StatusOK, pagina)
}

func recursoIntentoFiltro(f Filtro) string {
	if f.Validar() != nil {
		return recursoIntentoConsultaInvalida
	}
	if f.Fuente == "ct" {
		return recursoIntentoConsultaCT
	}
	return recursoIntentoConsultaBolsa
}

// El acuse procede de AD169, después del cierre de la lectura fallida. Una
// auditoría no confirmada se muestra como indisponibilidad y nunca como éxito.
func (h *Manejador) responderIntento(w http.ResponseWriter, r *http.Request,
	identidad IdentidadResuelta, recurso string, codigo int, opciones *Opciones) {
	if r.Context().Err() != nil {
		codigo = http.StatusServiceUnavailable
	}
	if h.servicio.intentos != nil {
		resultado := vecdomain.ResultadoIntentoAuditoriaError
		if codigo == http.StatusForbidden || codigo == http.StatusUnauthorized {
			resultado = vecdomain.ResultadoIntentoAuditoriaDenegado
		}
		acuse, err := h.servicio.registrarIntento(r.Context(), identidad, recurso, resultado, opciones)
		if err != nil {
			slog.Error("auditoria: intento nominal no confirmado", "estado_original", codigo,
				"causa", "registro_comun_no_disponible")
			responderError(w, http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("X-Audit-Ref", acuse.AuditoriaRef)
	}
	responderError(w, codigo)
}

func parsearInstante(s string) (time.Time, error) {
	if s == "" || !strings.HasSuffix(s, "Z") {
		return time.Time{}, ErrDenegada
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil || t.Nanosecond()%1000 != 0 {
		return time.Time{}, ErrDenegada
	}
	return t.UTC(), nil
}

func responderJSON(w http.ResponseWriter, codigo int, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		responderError(w, http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codigo)
	_, _ = w.Write(append(b, '\n'))
}

func responderError(w http.ResponseWriter, codigo int) {
	texto := "consulta no disponible"
	if codigo == http.StatusForbidden {
		texto = "consulta denegada"
	}
	if codigo == http.StatusBadRequest {
		texto = "solicitud invalida"
	}
	if codigo == http.StatusNotFound {
		texto = "ruta no encontrada"
	}
	if codigo == http.StatusMethodNotAllowed {
		texto = "metodo no admitido"
	}
	responderJSONSeguro(w, codigo, map[string]string{"error": texto})
}
func responderJSONSeguro(w http.ResponseWriter, codigo int, v any) {
	b := new(bytes.Buffer)
	_ = json.NewEncoder(b).Encode(v)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codigo)
	_, _ = w.Write(b.Bytes())
}

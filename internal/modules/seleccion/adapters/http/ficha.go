// Package http adapta la consulta exacta; no autentica ni concede permisos.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"time"

	bolsadomain "vec-diputacion-granada/internal/modules/bolsa/domain"
	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

const RutaFichaConvocatoria = "/api/vec/seleccion/convocatorias/ficha"

// ConfigFicha exige autoridades del montaje. ValidarFrontera comprueba canal,
// origen y audiencia; ResolverContexto deriva actor y correlación del servidor.
// Ambos deben registrar sus rechazos mediante la autoridad común de auditoría.
// Una composición de ensayo debe permanecer aislada y nunca exponerse como API
// institucional. El lector autorizado audita/consume V3 en su transacción.
type ConfigFicha struct {
	Lector           ports.LectorConvocatoriaExacta
	ResolverContexto func(*http.Request) (ports.SolicitudConsultaConvocatoria, error)
	ValidarFrontera  func(*http.Request) error
}

type fichaHandler struct{ config ConfigFicha }

func NuevaFichaHandler(c ConfigFicha) (http.Handler, error) {
	if c.Lector == nil || c.ResolverContexto == nil || c.ValidarFrontera == nil {
		return nil, ports.ErrConvocatoriaNoDisponible
	}
	return fichaHandler{config: c}, nil
}

func (h fichaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if r.URL.Path != RutaFichaConvocatoria || r.URL.RawQuery != "" || r.URL.RawPath != "" {
		escribirError(w, http.StatusNotFound, "no_encontrada")
		return
	}
	if err := h.config.ValidarFrontera(r); err != nil {
		escribirError(w, http.StatusForbidden, "acceso_denegado")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		escribirError(w, http.StatusMethodNotAllowed, "metodo_no_admitido")
		return
	}
	tipo, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || tipo != "application/json" || (params["charset"] != "" && params["charset"] != "utf-8") || len(params) > 1 || (len(params) == 1 && params["charset"] == "") {
		escribirError(w, http.StatusUnsupportedMediaType, "tipo_no_admitido")
		return
	}
	selector, err := leerSelector(http.MaxBytesReader(w, r.Body, 4096))
	if err != nil {
		escribirError(w, http.StatusBadRequest, "consulta_invalida")
		return
	}
	solicitud, err := h.config.ResolverContexto(r)
	if err != nil {
		escribirError(w, http.StatusForbidden, "acceso_denegado")
		return
	}
	// El cliente sólo aporta el selector; nunca reemplaza actor/correlación.
	solicitud.Selector = selector
	ctx, cancelar := context.WithTimeout(r.Context(), plazoarranque.Ampliar(10*time.Second))
	defer cancelar()
	lectura, err := application.ConsultarConvocatoria(ctx, solicitud, h.config.Lector)
	if err != nil {
		estado, codigo := http.StatusServiceUnavailable, "servicio_no_disponible"
		switch {
		case errors.Is(err, ports.ErrConsultaConvocatoriaDenegada):
			estado, codigo = http.StatusForbidden, "acceso_denegado"
		case errors.Is(err, ports.ErrConvocatoriaNoEncontrada):
			estado, codigo = http.StatusNotFound, "no_encontrada"
		case errors.Is(err, ports.ErrConsultaConvocatoriaInvalida):
			estado, codigo = http.StatusBadRequest, "consulta_invalida"
		}
		escribirError(w, estado, codigo)
		return
	}
	cuerpo, err := json.Marshal(respuestaFicha(lectura))
	if err != nil {
		escribirError(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	w.WriteHeader(http.StatusOK)
	// Un error de escritura cierra el transporte: no hay efectos que reintentar.
	if _, err := w.Write(cuerpo); err != nil {
		log.Print("seleccion_ficha_respuesta_no_entregada")
		return
	}
}

func leerSelector(cuerpo io.Reader) (bolsaports.SelectorVersionConvocatoriaExacta, error) {
	vacio := bolsaports.SelectorVersionConvocatoriaExacta{}
	dec := json.NewDecoder(cuerpo)
	inicio, err := dec.Token()
	if err != nil || inicio != json.Delim('{') {
		return vacio, ports.ErrConsultaConvocatoriaInvalida
	}
	s := vacio
	vistos := map[string]bool{}
	for dec.More() {
		token, err := dec.Token()
		if err != nil {
			return vacio, ports.ErrConsultaConvocatoriaInvalida
		}
		clave, ok := token.(string)
		if !ok || vistos[clave] {
			return vacio, ports.ErrConsultaConvocatoriaInvalida
		}
		vistos[clave] = true
		switch clave {
		case "convocatoria_id":
			err = dec.Decode(&s.ID)
		case "secuencia":
			err = dec.Decode(&s.Secuencia)
		default:
			return vacio, ports.ErrConsultaConvocatoriaInvalida
		}
		if err != nil {
			return vacio, ports.ErrConsultaConvocatoriaInvalida
		}
	}
	fin, err := dec.Token()
	if err != nil || fin != json.Delim('}') || len(vistos) != 2 || s.Validar() != nil {
		return vacio, ports.ErrConsultaConvocatoriaInvalida
	}
	if _, err := dec.Token(); err != io.EOF {
		return vacio, ports.ErrConsultaConvocatoriaInvalida
	}
	return s, nil
}

func escribirError(w http.ResponseWriter, estado int, codigo string) {
	cuerpo, err := json.Marshal(struct {
		Error string `json:"error"`
	}{codigo})
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(estado)
	if _, err := w.Write(cuerpo); err != nil {
		log.Print("seleccion_ficha_respuesta_no_entregada")
		return
	}
}

type evidenciaJSON struct {
	ReciboRef           string    `json:"recibo_ref"`
	DecisionRef         string    `json:"decision_ref"`
	ConsumoHuellaSHA256 string    `json:"consumo_huella_sha256"`
	AuditoriaRef        string    `json:"auditoria_ref"`
	CorrelacionRef      string    `json:"correlacion_ref"`
	ConsultadaEn        time.Time `json:"consultada_en"`
}

type fichaJSON struct {
	Ficha     ports.FichaConvocatoria `json:"ficha"`
	Evidencia evidenciaJSON           `json:"evidencia"`
}

func respuestaFicha(l ports.LecturaConvocatoria) fichaJSON {
	// Una lista vacía viaja como [] y nunca como null.
	if l.Ficha.Requisitos == nil {
		l.Ficha.Requisitos = []bolsadomain.RequisitoConvocatoria{}
	}
	e := l.Evidencia
	return fichaJSON{Ficha: l.Ficha, Evidencia: evidenciaJSON{
		ReciboRef: e.ReciboRef, DecisionRef: e.DecisionRef, ConsumoHuellaSHA256: e.ConsumoHuellaSHA256,
		AuditoriaRef: e.AuditoriaRef, CorrelacionRef: e.CorrelacionRef, ConsultadaEn: e.ConsultadaEn,
	}}
}

package http

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"mime"
	"net/http"
	"reflect"
	"time"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vec "vec-diputacion-granada/internal/vec/domain"
)

const (
	RutaGuardarPreparacionBases   = "/api/vec/seleccion/preparacion-bases/guardar"
	RutaConsultarPreparacionBases = "/api/vec/seleccion/preparacion-bases/consultar"
)

type ContextoPreparacionBases struct {
	Actor       vec.ContextoActor
	Correlacion vec.ReferenciaCorrelacionAutorizacionV2
	Ambito      bolsa.AmbitoOrganizativoConvocatoria
	// Callback de la composición con identidad capturada antes de leer JSON.
	RegistrarErrorEntrada func(context.Context, error) error
}

// Sigue la frontera de ConfigFicha: el montaje comprueba canal/origen y
// deriva contexto registrado. Ambas funciones auditan sus rechazos mediante
// la autoridad comun. Una cabecera o el JSON no resuelven estas capacidades.
type ConfigPreparacionBases struct {
	Preparador       ports.PreparadorBasesDurableV3
	ResolverContexto func(*http.Request) (ContextoPreparacionBases, error)
	ValidarFrontera  func(*http.Request) error
}

type preparacionBasesHandler struct{ config ConfigPreparacionBases }

func NuevaPreparacionBasesHandler(c ConfigPreparacionBases) (http.Handler, error) {
	if preparadorBasesNulo(c.Preparador) || c.ResolverContexto == nil || c.ValidarFrontera == nil {
		return nil, ports.ErrPreparacionBasesNoDisponible
	}
	return preparacionBasesHandler{c}, nil
}

func (h preparacionBasesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	guardar := r.URL.Path == RutaGuardarPreparacionBases
	if (!guardar && r.URL.Path != RutaConsultarPreparacionBases) || r.URL.RawQuery != "" || r.URL.RawPath != "" {
		escribirError(w, http.StatusNotFound, "no_encontrada")
		return
	}
	if err := h.config.ValidarFrontera(r); err != nil {
		responderErrorPreparacion(w, ports.ResultadoPreparacionBasesV3{}, err)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		escribirError(w, http.StatusMethodNotAllowed, "metodo_no_admitido")
		return
	}
	z, err := h.config.ResolverContexto(r)
	if err != nil {
		responderErrorPreparacion(w, ports.ResultadoPreparacionBasesV3{}, err)
		return
	}
	tipo, parametros, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || tipo != "application/json" || len(parametros) > 1 || (len(parametros) == 1 && parametros["charset"] != "utf-8") {
		h.entradaRechazada(w, r, z, http.StatusUnsupportedMediaType, "tipo_no_admitido")
		return
	}
	var entradaGuardar guardarPreparacionJSON
	var entradaConsultar consultarPreparacionJSON
	entrada := any(&entradaConsultar)
	if guardar {
		entrada = &entradaGuardar
	}
	if leerPreparacionJSON(http.MaxBytesReader(w, r.Body, 512*1024), entrada) != nil {
		h.entradaRechazada(w, r, z, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	ctx, cancelar := context.WithTimeout(r.Context(), plazoarranque.Ampliar(20*time.Second))
	defer cancelar()
	var resultado ports.ResultadoPreparacionBasesV3
	if guardar {
		resultado, err = h.config.Preparador.Guardar(ctx, ports.SolicitudGuardarPreparacionBasesV3{Actor: z.Actor, Correlacion: z.Correlacion, Ambito: z.Ambito, Esperada: entradaGuardar.Esperada, Material: entradaGuardar.Material, ClaveOperacion: entradaGuardar.ClaveOperacion})
	} else {
		resultado, err = h.config.Preparador.Consultar(ctx, ports.SolicitudConsultarPreparacionBasesV3{Actor: z.Actor, Correlacion: z.Correlacion, Ambito: z.Ambito, Selector: ports.SelectorConsultaPreparacionBases{Modo: entradaConsultar.Modo, Exacta: entradaConsultar.Exacta()}})
	}
	if err != nil {
		responderErrorPreparacion(w, resultado, err)
		return
	}
	respuesta, err := respuestaPreparacionBases(resultado)
	if err != nil {
		escribirError(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	estado := http.StatusOK
	if guardar && resultado.Estado == "guardada" {
		estado = http.StatusCreated
	}
	escribirPreparacionJSON(w, estado, respuesta)
}

func (h preparacionBasesHandler) entradaRechazada(w http.ResponseWriter, r *http.Request, z ContextoPreparacionBases, estado int, codigo string) {
	if z.RegistrarErrorEntrada == nil {
		escribirError(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	err := z.RegistrarErrorEntrada(r.Context(), ports.ErrPreparacionBasesInvalida)
	if !errors.Is(err, ports.ErrPreparacionBasesInvalida) || errors.Is(err, ports.ErrPreparacionBasesNoDisponible) {
		escribirError(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	escribirError(w, estado, codigo)
}

func responderErrorPreparacion(w http.ResponseWriter, r ports.ResultadoPreparacionBasesV3, err error) {
	estado, codigo := http.StatusServiceUnavailable, "servicio_no_disponible"
	switch {
	case errors.Is(err, ports.ErrPreparacionBasesNoDisponible), errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		// Un fallo tecnico no acredita una denegacion, incluso con error unido.
	case errors.Is(err, ports.ErrPreparacionBasesDenegada):
		estado, codigo = http.StatusForbidden, "acceso_denegado"
	case errors.Is(err, ports.ErrPreparacionBasesInvalida):
		estado, codigo = http.StatusBadRequest, "solicitud_invalida"
	case errors.Is(err, ports.ErrPreparacionBasesNoEncontrada):
		estado, codigo = http.StatusNotFound, "no_encontrada"
	case errors.Is(err, ports.ErrPreparacionBasesConflicto):
		estado, codigo = http.StatusConflict, "version_en_conflicto"
	case errors.Is(err, ports.ErrPreparacionBasesClaveReutilizada):
		estado, codigo = http.StatusConflict, "clave_reutilizada"
	}
	var acceso *accesoPreparacionJSON
	if (estado == http.StatusNotFound || estado == http.StatusConflict) && r.Acceso.DecisionRef != "" {
		a := accesoPreparacion(r.Acceso)
		acceso = &a
	}
	escribirPreparacionJSON(w, estado, struct {
		Error  string                 `json:"error"`
		Acceso *accesoPreparacionJSON `json:"acceso,omitempty"`
	}{codigo, acceso})
}

func preparadorBasesNulo(v ports.PreparadorBasesDurableV3) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return r.IsNil()
	}
	return false
}

func escribirPreparacionJSON(w http.ResponseWriter, estado int, cuerpo any) {
	b, err := json.Marshal(cuerpo)
	if err != nil {
		escribirError(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	w.WriteHeader(estado)
	if _, err := w.Write(b); err != nil {
		log.Print("seleccion_preparacion_respuesta_no_entregada")
	}
}

package httppersonal

import (
	"context"
	"net/http"

	"vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

const consultaPreferenciasModalidades = "tipo=modalidades"

type GuardadorPreferenciasModalidades interface {
	GuardarModalidades(context.Context, mibolsa.Orden, []string, uint64, string) (bolsa.ReciboPreferenciasModalidades, error)
}

type LectorPreferenciasModalidades interface {
	ConsultarPreferenciasModalidades(context.Context, mibolsa.Orden) (bolsa.EstadoPreferenciasModalidades, error)
}

type handlerPreferenciasModalidades struct {
	preparador Preparador
	guardador  GuardadorPreferenciasModalidades
	lector     LectorPreferenciasModalidades
}

// NuevoSelectorPreferenciasModalidades conserva la ruta y el contrato sin
// selector; solo la consulta canónica selecciona la capacidad de modalidades.
func NuevoSelectorPreferenciasModalidades(bolsas, modalidades http.Handler) (http.Handler, error) {
	if nula(bolsas) || nula(modalidades) {
		return nil, ErrDependenciaNoDisponible
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r != nil && r.URL != nil && r.URL.RawQuery == consultaPreferenciasModalidades && !r.URL.ForceQuery {
			modalidades.ServeHTTP(w, r)
			return
		}
		bolsas.ServeHTTP(w, r)
	}), nil
}

func NuevoHandlerPreferenciasModalidades(preparador Preparador, guardador GuardadorPreferenciasModalidades, lector LectorPreferenciasModalidades) (http.Handler, error) {
	if nula(preparador) || nula(guardador) || nula(lector) {
		return nil, ErrDependenciaNoDisponible
	}
	return &handlerPreferenciasModalidades{preparador: preparador, guardador: guardador, lector: lector}, nil
}

func (h *handlerPreferenciasModalidades) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || r.URL == nil || r.URL.Path != RutaPreferenciasBolsas || r.URL.RawPath != "" || r.URL.EscapedPath() != RutaPreferenciasBolsas {
		responder(w, http.StatusNotFound, errorRespuesta{"recurso_no_encontrado"})
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPut {
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPut)
		responder(w, http.StatusMethodNotAllowed, errorRespuesta{"metodo_no_permitido"})
		return
	}
	if r.Method == http.MethodGet {
		if r.URL.RawQuery != consultaPreferenciasModalidades || r.URL.ForceQuery || r.ContentLength != 0 || len(r.TransferEncoding) != 0 || !cuerpoAusente(r) || cabeceraProhibida(r.Header) {
			responder(w, http.StatusBadRequest, errorRespuesta{"peticion_no_permitida"})
			return
		}
		orden, err := h.preparador.PrepararMiBolsa(r)
		if err != nil {
			responderError(w, err)
			return
		}
		estado, err := h.lector.ConsultarPreferenciasModalidades(r.Context(), orden)
		if err != nil {
			responderErrorPreferenciasBolsas(w, err)
			return
		}
		type opcion struct {
			Clave  string                         `json:"clave"`
			Fuente core.ReferenciaEntradaCatalogo `json:"fuente"`
		}
		opciones := make([]opcion, 0, len(estado.ModalidadesDisponibles))
		for _, disponible := range estado.ModalidadesDisponibles {
			opciones = append(opciones, opcion{disponible.Clave, disponible.Fuente})
		}
		var fecha *string
		if estado.RegistradoEn != nil {
			valor := estado.RegistradoEn.UTC().Format("2006-01-02T15:04:05.000000Z")
			fecha = &valor
		}
		responder(w, http.StatusOK, struct {
			Version     uint64   `json:"version"`
			Guardadas   []string `json:"modalidades_guardadas"`
			Disponibles []opcion `json:"modalidades_disponibles"`
			Recibo      any      `json:"recibo_ref"`
			Fecha       *string  `json:"registrado_en"`
		}{estado.Version, estado.ModalidadesGuardadas, opciones, reciboRefOpcional(estado.ReciboRef), fecha})
		return
	}
	modalidades, version, clave, err := leerSolicitudPreferencias(r, consultaPreferenciasModalidades, "modalidades")
	if err != nil {
		responder(w, http.StatusBadRequest, errorRespuesta{"peticion_no_permitida"})
		return
	}
	orden, err := h.preparador.PrepararMiBolsa(r)
	if err != nil {
		responderError(w, err)
		return
	}
	recibo, err := h.guardador.GuardarModalidades(r.Context(), orden, modalidades, version, clave)
	if err != nil {
		responderErrorPreferenciasBolsas(w, err)
		return
	}
	codigo := http.StatusCreated
	if recibo.Reutilizado {
		codigo = http.StatusOK
	}
	responder(w, codigo, struct {
		Recibo      string   `json:"recibo_ref"`
		Version     uint64   `json:"version"`
		Modalidades []string `json:"modalidades"`
		Fecha       string   `json:"registrado_en"`
		Reutilizado bool     `json:"reutilizado"`
	}{recibo.ReciboRef, recibo.Version, recibo.Modalidades, recibo.RegistradoEn.UTC().Format("2006-01-02T15:04:05.000000Z"), recibo.Reutilizado})
}

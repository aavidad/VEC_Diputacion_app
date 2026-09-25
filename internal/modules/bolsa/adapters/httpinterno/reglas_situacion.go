package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

// RutaReglasSituacion sirve a la pantalla de RRHH las reglas del catálogo que
// afectan al cambio de situación: destinos admitidos, causas de baja y la
// propuesta de reposición. Es solo lectura; el cambio lo valida el servicio.
const RutaReglasSituacion = "/api/vec/bolsa/reglas-situacion"

const esquemaReglasSituacion = "vec.bolsa.rrhh.reglas_situacion.v1"

// HandlerReglasSituacion responde sin catálogo con «configuradas: false» y la
// tabla compilada, para que la pantalla siga como hasta ahora.
type HandlerReglasSituacion struct {
	reglas puertosbolsa.ConsultaReglasSituacion
	// transiciones da los destinos que admitirá el servicio, con la política
	// que publica la base. Nula: tabla compilada restringida por el catálogo.
	transiciones FuenteTransicionesSituacion
}

// FuenteTransicionesSituacion es el servicio de situación visto desde la
// lectura: por origen, los destinos que admitirá.
type FuenteTransicionesSituacion interface {
	TransicionesAdmitidas(ctx context.Context) (map[string][]string, error)
}

func NuevoHandlerReglasSituacion(reglas puertosbolsa.ConsultaReglasSituacion) (http.Handler, error) {
	return NuevoHandlerReglasSituacionConTransiciones(reglas, nil)
}

// NuevoHandlerReglasSituacionConTransiciones toma los destinos del servicio,
// de modo que la pantalla ofrece exactamente lo que se podrá registrar.
func NuevoHandlerReglasSituacionConTransiciones(reglas puertosbolsa.ConsultaReglasSituacion, transiciones FuenteTransicionesSituacion) (http.Handler, error) {
	if reglas == nil {
		return nil, errors.New("bolsa http interno: reglas de situacion no disponibles")
	}
	return &HandlerReglasSituacion{reglas: reglas, transiciones: transiciones}, nil
}

func (h *HandlerReglasSituacion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || r.URL == nil || r.URL.Path != RutaReglasSituacion || r.URL.RawPath != "" {
		responderSituacion(w, http.StatusNotFound, map[string]any{"error": map[string]string{"codigo": "recurso_no_encontrado"}})
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		responderSituacion(w, http.StatusMethodNotAllowed, map[string]any{"error": map[string]string{"codigo": "metodo_no_permitido"}})
		return
	}
	fin, modalidad, err := consultaReglasSituacion(r.URL.RawQuery)
	if err != nil || r.ContentLength > 0 {
		responderSituacion(w, http.StatusBadRequest, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	datos, err := h.datos(r.Context(), fin, modalidad)
	switch {
	case err == nil:
		responderSituacion(w, http.StatusOK, map[string]any{"data": datos})
	case errors.Is(err, puertosbolsa.ErrReposicionNoCalculable):
		responderSituacion(w, http.StatusBadRequest, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
	default:
		responderSituacion(w, http.StatusServiceUnavailable, map[string]any{"error": map[string]string{"codigo": "servicio_no_disponible"}})
	}
}

func (h *HandlerReglasSituacion) datos(ctx context.Context, fin time.Time, modalidad string) (map[string]any, error) {
	var transiciones map[string][]string
	var err error
	if h.transiciones != nil {
		transiciones, err = h.transiciones.TransicionesAdmitidas(ctx)
	} else {
		transiciones, err = TransicionesEfectivas(ctx, h.reglas)
	}
	if err != nil {
		return nil, err
	}
	datos := map[string]any{"esquema": esquemaReglasSituacion, "configuradas": h.reglas.Configurada(),
		"transiciones": transiciones, "causas_baja": []any{}, "reposicion": nil}
	if !h.reglas.Configurada() {
		return datos, nil
	}
	causas, err := h.reglas.CausasBaja(ctx)
	if err != nil {
		return nil, err
	}
	salidaCausas := make([]map[string]any, 0, len(causas))
	for _, causa := range causas {
		salidaCausas = append(salidaCausas, map[string]any{"codigo": causa.Codigo, "etiqueta": causa.Etiqueta, "procedencia": salidaProcedencia(causa.Procedencia)})
	}
	datos["causas_baja"] = salidaCausas
	modalidades, err := h.reglas.ModalidadesReposicion(ctx)
	if err != nil {
		return nil, err
	}
	salidaModalidades := make([]map[string]any, 0, len(modalidades))
	for _, m := range modalidades {
		salidaModalidades = append(salidaModalidades, map[string]any{"codigo": m.Codigo, "meses": m.Meses})
	}
	reposicion := map[string]any{"modalidades": salidaModalidades, "propuesta": nil}
	if !fin.IsZero() {
		propuesta, err := h.reglas.ProponerReposicion(ctx, fin, modalidad)
		if err != nil {
			return nil, err
		}
		reposicion["propuesta"] = map[string]any{
			"fecha_disponible":         propuesta.FechaDisponible.UTC().Format(time.RFC3339),
			"ultimo_dia_no_disponible": propuesta.UltimoDiaNoDisponible, "meses": propuesta.Meses,
			"procedencia": salidaProcedencia(propuesta.Procedencia),
		}
	}
	datos["reposicion"] = reposicion
	return datos, nil
}

// TransicionesEfectivas es la tabla compilada restringida por el catálogo:
// lo que el servicio admitirá en un cambio de situación.
func TransicionesEfectivas(ctx context.Context, reglas puertosbolsa.ReglasTransicionesSituacion) (map[string][]string, error) {
	resultado := make(map[string][]string, len(dominiobolsa.SituacionesParticipacion()))
	for _, origen := range dominiobolsa.SituacionesParticipacion() {
		destinos := dominiobolsa.DestinosSituacionParticipacion(origen)
		permitidos, configurada, err := reglas.DestinosSituacion(ctx, origen)
		if err != nil {
			return nil, err
		}
		if configurada {
			destinos = slices.DeleteFunc(destinos, func(d string) bool { return !slices.Contains(permitidos, d) })
		}
		resultado[origen] = destinos
	}
	return resultado, nil
}

func salidaProcedencia(p puertosbolsa.ProcedenciaRegla) map[string]any {
	return map[string]any{"clave": p.Clave, "referencia": p.Referencia, "articulo": p.Articulo, "norma": p.Norma, "ejemplo": p.Ejemplo}
}

// errConsultaReglasSituacion: la consulta no es válida (400).
var errConsultaReglasSituacion = errors.New("bolsa http interno: consulta de reglas de situacion no valida")

// consultaReglasSituacion admite, como mucho, fin_relacion (AAAA-MM-DD) y
// modalidad; la modalidad sin fecha no tiene sentido.
func consultaReglasSituacion(cruda string) (time.Time, string, error) {
	if cruda == "" {
		return time.Time{}, "", nil
	}
	if len(cruda) > 256 {
		return time.Time{}, "", errConsultaReglasSituacion
	}
	valores, err := url.ParseQuery(cruda)
	if err != nil {
		return time.Time{}, "", errors.Join(errConsultaReglasSituacion, err)
	}
	for clave, lista := range valores {
		if (clave != "fin_relacion" && clave != "modalidad") || len(lista) != 1 {
			return time.Time{}, "", errConsultaReglasSituacion
		}
	}
	texto := valores.Get("fin_relacion")
	modalidad := valores.Get("modalidad")
	if texto == "" || len(modalidad) > 64 {
		return time.Time{}, "", errConsultaReglasSituacion
	}
	dia, err := time.Parse(time.DateOnly, texto)
	if err != nil {
		return time.Time{}, "", errors.Join(errConsultaReglasSituacion, err)
	}
	if dia.Year() < 2000 || dia.Year() > 2100 {
		return time.Time{}, "", errConsultaReglasSituacion
	}
	// Mediodía UTC cae el mismo día civil en hora peninsular; el cómputo de
	// fecha a fecha solo usa ese día.
	return dia.Add(12 * time.Hour), modalidad, nil
}

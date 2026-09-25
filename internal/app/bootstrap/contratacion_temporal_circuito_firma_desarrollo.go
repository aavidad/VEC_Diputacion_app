package bootstrap

import (
	"encoding/json"
	"net/http"
	"strconv"

	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/reglas"
)

// La consulta del circuito de firma devuelve, por documento, los pasos del
// catálogo de ejemplo con su estado. Aún no existe registro de firmas, de modo
// que el estado es siempre el de un documento sin firmas: el primer paso
// pendiente y el resto en espera. No firma, no autoriza y no escribe nada.
const (
	rutaCircuitoFirmaContratacionTemporalDesarrollo    = "/api/vec/contratacion-temporal/circuito-firma"
	esquemaCircuitoFirmaContratacionTemporalDesarrollo = "vec.contratacion_temporal.circuito_firma.v1"
)

type pasoCircuitoFirmaDesarrollo struct {
	Orden       int    `json:"orden"`
	Cargo       string `json:"cargo"`
	PerfilRef   string `json:"perfil_ref"`
	Accion      string `json:"accion"`
	Condicion   string `json:"condicion"`
	Habilita    string `json:"habilita"`
	Devolucion  string `json:"devolucion"`
	Sustitucion string `json:"sustitucion"`
	Estado      string `json:"estado"`
	Referencia  string `json:"referencia"`
}

type documentoCircuitoFirmaDesarrollo struct {
	Documento string                        `json:"documento"`
	Etiqueta  string                        `json:"etiqueta"`
	Pasos     []pasoCircuitoFirmaDesarrollo `json:"pasos"`
}

type circuitoFirmaDesarrollo struct {
	Esquema      string                             `json:"esquema"`
	CatalogoRef  string                             `json:"catalogo_ref"`
	HuellaSHA256 string                             `json:"huella_sha256"`
	Ejemplo      bool                               `json:"ejemplo"`
	FirmaEficaz  bool                               `json:"firma_eficaz"`
	Documentos   []documentoCircuitoFirmaDesarrollo `json:"documentos"`
}

// nuevaRutaCircuitoFirmaContratacionTemporalDesarrollo compone la consulta.
// Sin catálogo la ruta existe y responde 503: el portal oculta el bloque.
func nuevaRutaCircuitoFirmaContratacionTemporalDesarrollo(resolutor *reglas.Resolutor) vechttp.RutaExacta {
	return vechttp.RutaExacta{
		Ruta:      rutaCircuitoFirmaContratacionTemporalDesarrollo,
		Manejador: manejadorCircuitoFirmaContratacionTemporalDesarrollo{resolutor: resolutor},
	}
}

type manejadorCircuitoFirmaContratacionTemporalDesarrollo struct {
	resolutor *reglas.Resolutor
}

func (m manejadorCircuitoFirmaContratacionTemporalDesarrollo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	prepararCabecerasCatalogosAltaContratacionTemporalDesarrollo(w)
	if r == nil || r.URL == nil {
		responderErrorCircuitoFirmaDesarrollo(w, r, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	if r.URL.Path != rutaCircuitoFirmaContratacionTemporalDesarrollo || r.URL.RawQuery != "" ||
		r.ContentLength != 0 || len(r.TransferEncoding) != 0 ||
		cabeceraCatalogosAltaContratacionTemporalDesarrolloProhibida(r.Header) {
		responderErrorCircuitoFirmaDesarrollo(w, r, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodHead)
		responderErrorCircuitoFirmaDesarrollo(w, r, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	circuito, err := m.resolutor.CircuitoFirma(r.Context())
	if err != nil {
		responderErrorCircuitoFirmaDesarrollo(w, r, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	contenido, err := json.Marshal(map[string]circuitoFirmaDesarrollo{"data": vistaCircuitoFirmaDesarrollo(circuito)})
	if err != nil {
		responderErrorCircuitoFirmaDesarrollo(w, r, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		_, _ = w.Write(contenido)
	}
}

func vistaCircuitoFirmaDesarrollo(circuito reglas.CircuitoFirma) circuitoFirmaDesarrollo {
	vista := circuitoFirmaDesarrollo{
		Esquema:      esquemaCircuitoFirmaContratacionTemporalDesarrollo,
		CatalogoRef:  circuito.CatalogoID + ":" + strconv.Itoa(circuito.Version),
		HuellaSHA256: circuito.HuellaCatalogo,
		Ejemplo:      circuito.PaqueteEjemplo,
		// Sin portafirmas corporativo ninguna firma tiene eficacia
		// administrativa; el campo existe para que el portal no lo suponga.
		FirmaEficaz: false,
		Documentos:  make([]documentoCircuitoFirmaDesarrollo, 0, len(circuito.Documentos)),
	}
	for _, documento := range circuito.Documentos {
		salida := documentoCircuitoFirmaDesarrollo{
			Documento: documento.Documento, Etiqueta: documento.Etiqueta,
			Pasos: make([]pasoCircuitoFirmaDesarrollo, 0, len(documento.Pasos)),
		}
		for _, paso := range documento.Pasos {
			salida.Pasos = append(salida.Pasos, pasoCircuitoFirmaDesarrollo{
				Orden: paso.Orden, Cargo: paso.Cargo, PerfilRef: paso.PerfilRef,
				Accion: string(paso.Accion), Condicion: string(paso.Condicion), Habilita: string(paso.Habilita),
				Devolucion: string(paso.Devolucion), Sustitucion: string(paso.Sustitucion),
				Estado: string(reglas.EstadoPasoSinFirmas(paso.Orden)), Referencia: paso.Referencia,
			})
		}
		vista.Documentos = append(vista.Documentos, salida)
	}
	return vista
}

func responderErrorCircuitoFirmaDesarrollo(w http.ResponseWriter, r *http.Request, estado int, codigo string) {
	contenido, err := json.Marshal(map[string]any{
		"error": map[string]string{
			"codigo":     codigo,
			"clave_i18n": "api.contratacion_temporal.circuito_firma.error." + codigo,
		},
	})
	if err != nil {
		contenido = []byte(`{"error":{"codigo":"servicio_no_disponible"}}`)
		estado = http.StatusServiceUnavailable
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(estado)
	if r == nil || r.Method != http.MethodHead {
		_, _ = w.Write(contenido)
	}
}

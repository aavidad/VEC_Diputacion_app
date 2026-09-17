package bootstrap

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"vec-diputacion-granada/config"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
)

const (
	rutaAreaPersonalBolsaDesarrollo   = "/api/vec/bolsa/area-personal"
	rutaDisponibilidadBolsaDesarrollo = "/api/vec/bolsa/mi-disponibilidad"
	candidaturaAreaPersonalDesarrollo = "candidatura:demo:0009" // tiene un llamamiento en el dataset sintético
)

type datasetAreaPersonalDesarrollo struct {
	Bolsas []struct {
		Referencia string `json:"bolsa_ref"`
		Categoria  string `json:"categoria"`
	} `json:"bolsas"`
	Candidaturas []fichaCandidaturaAreaPersonalDesarrollo `json:"candidaturas"`
	Llamamientos []struct {
		Referencia  string `json:"llamamiento_ref"`
		Candidatura string `json:"candidatura_ref"`
		Bolsa       string `json:"bolsa_ref"`
		Puesto      string `json:"puesto"`
		Plazo       string `json:"plazo_respuesta_hasta"`
		Resultado   string `json:"resultado"`
	} `json:"llamamientos"`
}

type fichaCandidaturaAreaPersonalDesarrollo struct {
	Referencia string  `json:"candidatura_ref"`
	Nombre     string  `json:"nombre_visible"`
	Bolsa      string  `json:"bolsa_ref"`
	Orden      int     `json:"orden"`
	Puntuacion float64 `json:"puntuacion"`
	Estado     string  `json:"estado"`
}

type areaPersonalBolsaDesarrollo struct {
	mu           sync.Mutex
	candidatura  fichaCandidaturaAreaPersonalDesarrollo
	bolsas       map[string]string
	llamamientos []map[string]string
	disponible   bool
}

func nuevaAreaPersonalBolsaDesarrollo(ruta string) (*areaPersonalBolsaDesarrollo, error) {
	if strings.TrimSpace(ruta) == "" {
		return nil, os.ErrNotExist
	}
	contenido, err := os.ReadFile(ruta)
	if err != nil || len(contenido) == 0 || len(contenido) > 2<<20 {
		return nil, os.ErrNotExist
	}
	var datos datasetAreaPersonalDesarrollo
	if json.Unmarshal(contenido, &datos) != nil {
		return nil, os.ErrNotExist
	}
	resultado := &areaPersonalBolsaDesarrollo{bolsas: map[string]string{}, disponible: true}
	for _, bolsa := range datos.Bolsas {
		if bolsa.Referencia != "" && bolsa.Categoria != "" {
			resultado.bolsas[bolsa.Referencia] = bolsa.Categoria
		}
	}
	for _, candidatura := range datos.Candidaturas {
		if candidatura.Referencia == candidaturaAreaPersonalDesarrollo {
			resultado.candidatura = candidatura
			resultado.disponible = candidatura.Estado == "Disponible"
			break
		}
	}
	if resultado.candidatura.Referencia == "" || resultado.bolsas[resultado.candidatura.Bolsa] == "" {
		return nil, os.ErrNotExist
	}
	for _, llamamiento := range datos.Llamamientos {
		if llamamiento.Candidatura == candidaturaAreaPersonalDesarrollo {
			resultado.llamamientos = append(resultado.llamamientos, map[string]string{"id": llamamiento.Referencia, "bolsa": resultado.bolsas[llamamiento.Bolsa], "puesto": llamamiento.Puesto, "plazo": llamamiento.Plazo, "estado": llamamiento.Resultado})
		}
	}
	return resultado, nil
}

func nuevasRutasAreaPersonalBolsaDesarrollo(cfg config.Config) ([]vechttp.RutaExacta, error) {
	area, err := nuevaAreaPersonalBolsaDesarrollo(cfg.BolsaDemoPath)
	if err != nil {
		return nil, err
	}
	return []vechttp.RutaExacta{{Ruta: rutaAreaPersonalBolsaDesarrollo, Manejador: area}, {Ruta: rutaDisponibilidadBolsaDesarrollo, Manejador: area}}, nil
}

func (a *areaPersonalBolsaDesarrollo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	if a == nil || r == nil || r.URL == nil || r.URL.RawQuery != "" || len(r.TransferEncoding) != 0 || cabeceraCatalogosAltaContratacionTemporalDesarrolloProhibida(r.Header) {
		responderAreaPersonalDesarrollo(w, http.StatusBadRequest, map[string]string{"codigo": "solicitud_invalida"})
		return
	}
	switch r.URL.Path {
	case rutaAreaPersonalBolsaDesarrollo:
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			responderAreaPersonalDesarrollo(w, http.StatusMethodNotAllowed, map[string]string{"codigo": "metodo_no_permitido"})
			return
		}
		if r.ContentLength != 0 {
			responderAreaPersonalDesarrollo(w, http.StatusBadRequest, map[string]string{"codigo": "solicitud_invalida"})
			return
		}
		a.mu.Lock()
		datos := a.panel()
		a.mu.Unlock()
		responderAreaPersonalDesarrollo(w, http.StatusOK, map[string]any{"data": datos}, r.Method == http.MethodHead)
	case rutaDisponibilidadBolsaDesarrollo:
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			responderAreaPersonalDesarrollo(w, http.StatusMethodNotAllowed, map[string]string{"codigo": "metodo_no_permitido"})
			return
		}
		var entrada struct {
			Data struct {
				Esquema      string `json:"esquema"`
				Accion       string `json:"accion"`
				Confirmacion bool   `json:"confirmacion"`
				Payload      struct {
					Disponible *bool `json:"disponible"`
				} `json:"payload"`
			} `json:"data"`
		}
		if r.ContentLength < 1 || r.ContentLength > 64<<10 || json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&entrada) != nil || entrada.Data.Esquema != "vec.bolsa.area-personal.accion.v1" || entrada.Data.Accion != "cambiar_disponibilidad" || !entrada.Data.Confirmacion || entrada.Data.Payload.Disponible == nil {
			responderAreaPersonalDesarrollo(w, http.StatusUnprocessableEntity, map[string]string{"codigo": "contenido_no_valido"})
			return
		}
		a.mu.Lock()
		a.disponible = *entrada.Data.Payload.Disponible
		disponible := a.disponible
		a.mu.Unlock()
		responderAreaPersonalDesarrollo(w, http.StatusOK, map[string]any{"data": map[string]any{"recibo": map[string]any{"esquema": "vec.bolsa.area-personal.recibo.v1", "presentacion": false, "referencia": "recibo:bolsa:demo:disponibilidad", "accion": "cambiar_disponibilidad", "objetivo": candidaturaAreaPersonalDesarrollo, "resultado": "confirmado", "actor": "RRHH demostración", "fecha": instanteAreaPersonalBolsaDesarrollo(), "advertencia": "Estado efímero de demostración; se restablece al reiniciar."}, "resultado": map[string]bool{"disponible": disponible}}})
	default:
		responderAreaPersonalDesarrollo(w, http.StatusNotFound, map[string]string{"codigo": "recurso_no_encontrado"})
	}
}

func (a *areaPersonalBolsaDesarrollo) panel() map[string]any {
	estado := "Disponible para llamamientos"
	if !a.disponible {
		estado = "No disponible (demostración)"
	}
	llamamientos := make([]map[string]string, len(a.llamamientos))
	copy(llamamientos, a.llamamientos)
	vacio := []map[string]string{}
	return map[string]any{"meta": map[string]any{"esquema": "vec.bolsa.area-personal.v1", "presentacion": false, "origen": "Dataset sintético de Bolsa; disponibilidad efímera del proceso", "generado_en": instanteAreaPersonalBolsaDesarrollo()}, "sesion": map[string]string{"persona_ref": candidaturaAreaPersonalDesarrollo, "nombre_visible": a.candidatura.Nombre, "iniciales": "CD", "metodo": "demostración sin identidad de candidato"}, "resumen": map[string]any{"acciones_pendientes": 0, "convocatorias_abiertas": 0, "solicitudes_activas": 0, "mensajes_no_leidos": 0, "puntuacion_provisional": a.candidatura.Puntuacion}, "perfil": map[string]string{"referencia": "perfil:demo:0009", "nombre_visible": a.candidatura.Nombre, "identificador_visible": "Candidatura sintética 0009", "correo": "candidatura.0009@ejemplo.test", "telefono": "No disponible en demostración", "domicilio": "No disponible en demostración", "estado_verificacion": "Demostración sin identidad de candidato"}, "plazos": vacio, "convocatorias": vacio, "meritos": vacio, "solicitudes": vacio, "baremo": vacio, "llamamientos": llamamientos, "subsanaciones": vacio, "alegaciones": vacio, "mensajes": vacio, "certificados": vacio, "documentos": vacio, "actividad": vacio, "ayuda": vacio, "disponibilidad": map[string]any{"disponible": a.disponible, "estado": estado}, "capacidades": map[string]bool{"cambiar_disponibilidad": true}}
}
func responderAreaPersonalDesarrollo(w http.ResponseWriter, estado int, valor any, soloCabecera ...bool) {
	b, _ := json.Marshal(valor)
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.WriteHeader(estado)
	if estado != http.StatusNoContent && (len(soloCabecera) == 0 || !soloCabecera[0]) {
		_, _ = w.Write(b)
	}
}

func instanteAreaPersonalBolsaDesarrollo() string {
	return time.Now().UTC().Truncate(time.Millisecond).Format("2006-01-02T15:04:05.000Z07:00")
}

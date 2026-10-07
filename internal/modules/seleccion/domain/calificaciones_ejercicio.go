package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
)

var ErrCalificacionesEjercicio = errors.New("seleccion.calificaciones_ejercicio.material_invalido")

const EsquemaCalificacionesEjercicio = "vec.seleccion.calificaciones-ejercicio.v1"

// NotaEjercicio conserva un dato aportado y su fuente opaca. La referencia de
// fuente no acredita corrección, custodia, autoría ni identidad de la persona.
type NotaEjercicio struct {
	SolicitudRef      string `json:"solicitud_ref"`
	PuntosMicropuntos *int64 `json:"puntos_micropuntos"`
	FuenteRef         string `json:"fuente_ref,omitempty"`
}

// MaterialCalificacionesEjercicio describe una revisión completa de un ejercicio.
// Configuracion se aporta desde el catálogo de bases, todavía sin cotejo aquí.
type MaterialCalificacionesEjercicio struct {
	Esquema           string          `json:"esquema"`
	ConvocatoriaRef   string          `json:"convocatoria_ref"`
	BasesVersion      int             `json:"bases_version"`
	BasesHuellaSHA256 string          `json:"bases_huella_sha256"`
	Configuracion     Configuracion   `json:"configuracion"`
	EjercicioRef      string          `json:"ejercicio_ref"`
	FaseRef           string          `json:"fase_ref"`
	Revision          int             `json:"revision"`
	AntecedenteSHA256 string          `json:"antecedente_sha256,omitempty"`
	Notas             []NotaEjercicio `json:"notas"`
}

type NotaEjercicioPreparada struct {
	NotaEjercicio
	Estado string `json:"estado"`
}

// RegistroCalificacionesPreparado fija la versión y huella del material local.
// La cadena de antecedentes se cotejará con el registro institucional.
type RegistroCalificacionesPreparado struct {
	Esquema              string                   `json:"esquema"`
	Estado               string                   `json:"estado"`
	ConvocatoriaRef      string                   `json:"convocatoria_ref"`
	BasesVersion         int                      `json:"bases_version"`
	BasesHuellaSHA256    string                   `json:"bases_huella_sha256"`
	ConfiguracionVersion int                      `json:"configuracion_version"`
	Configuracion        Configuracion            `json:"configuracion"`
	EjercicioRef         string                   `json:"ejercicio_ref"`
	FaseRef              string                   `json:"fase_ref"`
	Revision             int                      `json:"revision"`
	AntecedenteSHA256    string                   `json:"antecedente_sha256,omitempty"`
	Notas                []NotaEjercicioPreparada `json:"notas"`
	HuellaMaterialSHA256 string                   `json:"huella_material_sha256"`
	Pendientes           []string                 `json:"pendientes"`
	Aprobada             bool                     `json:"aprobada"`
	Publicada            bool                     `json:"publicada"`
}

// PrepararCalificacionesEjercicio valida las coordenadas y notas de una fase
// de prueba. Mínimos, ponderaciones y desempates pertenecen a Evaluar; no se
// decide aquí si alguien supera una fase ni se calcula una clasificación.
func PrepararCalificacionesEjercicio(m MaterialCalificacionesEjercicio) (RegistroCalificacionesPreparado, error) {
	if m.Esquema != EsquemaCalificacionesEjercicio || !referenciaValida(m.ConvocatoriaRef) ||
		m.BasesVersion < 1 || m.BasesVersion > 1_000_000 || !shaAdmision(m.BasesHuellaSHA256) ||
		m.Configuracion.Validar() != nil || !referenciaValida(m.EjercicioRef) || !referenciaValida(m.FaseRef) ||
		m.Revision < 1 || m.Revision > 1_000_000 || len(m.Notas) == 0 || len(m.Notas) > 128 ||
		(m.Revision == 1 && m.AntecedenteSHA256 != "") || (m.Revision > 1 && !shaAdmision(m.AntecedenteSHA256)) {
		return RegistroCalificacionesPreparado{}, ErrCalificacionesEjercicio
	}
	var fase *Fase
	for i := range m.Configuracion.Fases {
		if m.Configuracion.Fases[i].Referencia == m.FaseRef {
			fase = &m.Configuracion.Fases[i]
			break
		}
	}
	if fase == nil || fase.Tipo != "prueba" {
		return RegistroCalificacionesPreparado{}, ErrCalificacionesEjercicio
	}
	vistos := map[string]bool{}
	notas := make([]NotaEjercicioPreparada, 0, len(m.Notas))
	for _, n := range m.Notas {
		if !referenciaValida(n.SolicitudRef) || vistos[n.SolicitudRef] ||
			(n.PuntosMicropuntos == nil && n.FuenteRef != "") ||
			(n.PuntosMicropuntos != nil && (!referenciaValida(n.FuenteRef) || *n.PuntosMicropuntos < 0 || *n.PuntosMicropuntos > fase.MaximoMicropuntos)) {
			return RegistroCalificacionesPreparado{}, ErrCalificacionesEjercicio
		}
		vistos[n.SolicitudRef] = true
		estado := "pendiente"
		if n.PuntosMicropuntos != nil {
			estado = "aportada_para_revision"
		}
		puntos := copiarPuntos(n.PuntosMicropuntos)
		notas = append(notas, NotaEjercicioPreparada{NotaEjercicio{n.SolicitudRef, puntos, n.FuenteRef}, estado})
	}
	sort.Slice(notas, func(i, j int) bool { return notas[i].SolicitudRef < notas[j].SolicitudRef })
	configuracion := m.Configuracion
	configuracion.Fases = append([]Fase(nil), m.Configuracion.Fases...)
	for i := range configuracion.Fases {
		configuracion.Fases[i].MinimoMicropuntos = copiarPuntos(configuracion.Fases[i].MinimoMicropuntos)
	}
	configuracion.Desempates = append([]string(nil), m.Configuracion.Desempates...)
	out := RegistroCalificacionesPreparado{
		Esquema: m.Esquema, Estado: "borrador_pendiente_validacion", ConvocatoriaRef: m.ConvocatoriaRef,
		BasesVersion: m.BasesVersion, BasesHuellaSHA256: m.BasesHuellaSHA256,
		ConfiguracionVersion: m.Configuracion.Version, Configuracion: configuracion, EjercicioRef: m.EjercicioRef, FaseRef: m.FaseRef,
		Revision: m.Revision, AntecedenteSHA256: m.AntecedenteSHA256, Notas: notas,
		Pendientes: []string{"bases_y_fase_no_cotejadas", "admision_y_anonimato_no_cotejados", "fuentes_y_acta_no_cotejadas", "autor_y_aprobacion_pendientes", "reclamacion_y_publicacion_pendientes"},
	}
	if fase.MinimoMicropuntos == nil {
		out.Pendientes = append(out.Pendientes, "minimo_fase_pendiente")
	}
	if m.Revision > 1 {
		out.Pendientes = append(out.Pendientes, "antecedente_no_cotejado")
	}
	// La huella cubre sólo coordenadas y notas normalizadas, nunca mensajes.
	serial, err := json.Marshal(struct {
		Esquema              string                   `json:"esquema"`
		ConvocatoriaRef      string                   `json:"convocatoria_ref"`
		BasesVersion         int                      `json:"bases_version"`
		BasesHuellaSHA256    string                   `json:"bases_huella_sha256"`
		ConfiguracionVersion int                      `json:"configuracion_version"`
		Configuracion        Configuracion            `json:"configuracion"`
		EjercicioRef         string                   `json:"ejercicio_ref"`
		FaseRef              string                   `json:"fase_ref"`
		Revision             int                      `json:"revision"`
		AntecedenteSHA256    string                   `json:"antecedente_sha256,omitempty"`
		Notas                []NotaEjercicioPreparada `json:"notas"`
	}{out.Esquema, out.ConvocatoriaRef, out.BasesVersion, out.BasesHuellaSHA256, out.ConfiguracionVersion, out.Configuracion, out.EjercicioRef, out.FaseRef, out.Revision, out.AntecedenteSHA256, out.Notas})
	if err != nil {
		return RegistroCalificacionesPreparado{}, err
	}
	huella := sha256.Sum256(serial)
	out.HuellaMaterialSHA256 = hex.EncodeToString(huella[:])
	return out, nil
}

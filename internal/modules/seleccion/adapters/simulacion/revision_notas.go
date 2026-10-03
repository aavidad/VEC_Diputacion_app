package simulacion

import (
	"bytes"
	"encoding/json"
	"io"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

type SolicitudRevisionNotas struct {
	EjemploRef       string               `json:"ejemplo_ref"`
	Configuracion    domain.Configuracion `json:"configuracion"`
	NotasAntecedente []NotaPrueba         `json:"notas_antecedente,omitempty"`
	NotasPropuestas  []NotaPrueba         `json:"notas_propuestas,omitempty"`
}

type CambioNota struct {
	SolicitudRef         string `json:"solicitud_ref"`
	FaseRef              string `json:"fase_ref"`
	AnteriorMicropuntos  *int64 `json:"anterior_micropuntos"`
	PropuestaMicropuntos *int64 `json:"propuesta_micropuntos"`
}

type RevisionNotas struct {
	Esquema                 string               `json:"esquema"`
	Alcance                 string               `json:"alcance"`
	EjemploRef              string               `json:"ejemplo_ref"`
	Configuracion           domain.Configuracion `json:"configuracion"`
	NotasAntecedente        []NotaPrueba         `json:"notas_antecedente"`
	NotasPropuestas         []NotaPrueba         `json:"notas_propuestas"`
	Antecedente             domain.Resultado     `json:"antecedente"`
	Propuesta               domain.Resultado     `json:"propuesta"`
	HuellaAntecedenteSHA256 string               `json:"huella_antecedente_sha256"`
	Cambios                 []CambioNota         `json:"cambios"`
	Pendientes              []string             `json:"pendientes"`
}

// DecodificarRevisionNotas conserva los límites y claves exactas de S6a.
// Cada lista reutiliza su decodificador para distinguir ausencia de null.
func DecodificarRevisionNotas(datos []byte) (SolicitudRevisionNotas, error) {
	if len(datos) == 0 || len(datos) > MaximoBytes || !utf8.Valid(datos) {
		return SolicitudRevisionNotas{}, ErrSolicitud
	}
	d := json.NewDecoder(bytes.NewReader(datos))
	d.UseNumber()
	if tokens(d, 0) != nil {
		return SolicitudRevisionNotas{}, ErrSolicitud
	}
	if _, err := d.Token(); err != io.EOF {
		return SolicitudRevisionNotas{}, ErrSolicitud
	}
	var forma struct {
		EjemploRef       string          `json:"ejemplo_ref"`
		Configuracion    json.RawMessage `json:"configuracion"`
		NotasAntecedente json.RawMessage `json:"notas_antecedente"`
		NotasPropuestas  json.RawMessage `json:"notas_propuestas"`
	}
	d = json.NewDecoder(bytes.NewReader(datos))
	d.DisallowUnknownFields()
	if d.Decode(&forma) != nil {
		return SolicitudRevisionNotas{}, ErrSolicitud
	}
	decodificar := func(notas json.RawMessage) (Solicitud, error) {
		material, err := json.Marshal(struct {
			EjemploRef    string          `json:"ejemplo_ref"`
			Configuracion json.RawMessage `json:"configuracion"`
			NotasPrueba   json.RawMessage `json:"notas_prueba,omitempty"`
		}{forma.EjemploRef, forma.Configuracion, notas})
		if err != nil {
			return Solicitud{}, ErrSolicitud
		}
		return Decodificar(material)
	}
	a, err := decodificar(forma.NotasAntecedente)
	if err != nil {
		return SolicitudRevisionNotas{}, err
	}
	p, err := decodificar(forma.NotasPropuestas)
	if err != nil {
		return SolicitudRevisionNotas{}, err
	}
	return SolicitudRevisionNotas{forma.EjemploRef, a.Configuracion, a.NotasPrueba, p.NotasPrueba}, nil
}

// PrepararRevisionNotas calcula dos ensayos con la misma configuración. Las
// propuestas sustituyen únicamente sus pares, conservando las demás ediciones
// del antecedente. La huella identifica JSON Go del resultado, sin firma.
func PrepararRevisionNotas(s SolicitudRevisionNotas) (RevisionNotas, error) {
	material, err := json.Marshal(s)
	if err != nil {
		return RevisionNotas{}, ErrSolicitud
	}
	// Copia y valida también a los consumidores internos del DTO.
	s, err = DecodificarRevisionNotas(material)
	if err != nil {
		return RevisionNotas{}, err
	}
	base := Solicitud{EjemploRef: s.EjemploRef, Configuracion: s.Configuracion, NotasPrueba: s.NotasAntecedente}
	entrada, baremador, err := PrepararConNotas(base)
	if err != nil {
		return RevisionNotas{}, err
	}
	// Validar la lista propuesta por separado impide ocultar sus duplicados al
	// acumularla. Ninguna validación ni cálculo escribe en el ejemplo embebido.
	if _, _, err := PrepararConNotas(Solicitud{EjemploRef: s.EjemploRef, Configuracion: s.Configuracion, NotasPrueba: s.NotasPropuestas}); err != nil {
		return RevisionNotas{}, err
	}
	antecedente, err := application.Simular(s.Configuracion, entrada, baremador)
	if err != nil {
		return RevisionNotas{}, err
	}
	MarcarNotasEditadas(&antecedente, s.NotasAntecedente)
	anteriorJSON, err := json.Marshal(antecedente)
	if err != nil {
		return RevisionNotas{}, err
	}
	acumuladas := acumularNotas(s.NotasAntecedente, s.NotasPropuestas)
	base.NotasPrueba = acumuladas
	entrada, baremador, err = PrepararConNotas(base)
	if err != nil {
		return RevisionNotas{}, err
	}
	propuesta, err := application.Simular(s.Configuracion, entrada, baremador)
	if err != nil {
		return RevisionNotas{}, err
	}
	MarcarNotasEditadas(&propuesta, acumuladas)
	cambios := []CambioNota{}
	for i, solicitud := range antecedente.Solicitudes {
		for j, fase := range solicitud.Fases {
			nueva := propuesta.Solicitudes[i].Fases[j]
			if fase.Tipo == "prueba" && !mismosPuntos(fase.PuntosMicropuntos, nueva.PuntosMicropuntos) {
				cambios = append(cambios, CambioNota{solicitud.Referencia, fase.Referencia, fase.PuntosMicropuntos, nueva.PuntosMicropuntos})
			}
		}
	}
	return RevisionNotas{Esquema: "seleccion.revision_notas.v1", Alcance: "preparacion_sintetica", EjemploRef: s.EjemploRef,
		Configuracion: s.Configuracion, NotasAntecedente: append([]NotaPrueba{}, s.NotasAntecedente...), NotasPropuestas: append([]NotaPrueba{}, s.NotasPropuestas...),
		Antecedente: antecedente, Propuesta: propuesta, HuellaAntecedenteSHA256: huella(anteriorJSON), Cambios: cambios,
		Pendientes: []string{"revision_competente", "acto", "firma", "publicacion"}}, nil
}

func acumularNotas(antecedentes, propuestas []NotaPrueba) []NotaPrueba {
	r := append([]NotaPrueba{}, antecedentes...)
	posiciones := map[claveNota]int{}
	for i, nota := range r {
		posiciones[claveNota{nota.SolicitudRef, nota.FaseRef}] = i
	}
	for _, nota := range propuestas {
		clave := claveNota{nota.SolicitudRef, nota.FaseRef}
		if i, existe := posiciones[clave]; existe {
			r[i] = nota
		} else {
			posiciones[clave] = len(r)
			r = append(r, nota)
		}
	}
	return r
}

func mismosPuntos(a, b *int64) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

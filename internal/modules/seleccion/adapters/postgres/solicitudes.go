package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

func sqlConMaterial(prefijo string, primeros int) string {
	n := make([]any, 10)
	for i := range n {
		n[i] = primeros + 1 + i
	}
	return prefijo + fmt.Sprintf(parametrosMaterial, n...) + ")"
}

// GuardarBorrador guarda una versión nueva (o devuelve la repetición).
func (r *Repositorio) GuardarBorrador(ctx context.Context, c ports.ComandoGuardarBorrador) (ports.ResultadoGuardado, error) {
	if !materialValido(c.Material) {
		return ports.ResultadoGuardado{}, ports.ErrDatosNoValidos
	}
	// Una lista vacía se guarda como [] (nunca null).
	requisitos, e1 := json.Marshal(append([]domain.RequisitoDeclarado{}, c.Requisitos...))
	meritos, e2 := json.Marshal(append([]domain.MeritoDeclarado{}, c.Meritos...))
	if e1 != nil || e2 != nil {
		return ports.ResultadoGuardado{}, ports.ErrDatosNoValidos
	}
	args := []any{c.PersonaRef, c.ConvocatoriaRef, c.ConvocatoriaVersion, c.VersionEsperada, c.Clave, c.HuellaMaterial, c.SolicitudRefNueva, nulo(c.Turno),
		c.Sobre.ClaveRef, c.Sobre.Nonce, c.Sobre.Cifrado, nulo(c.DocumentoHuella), nulo(c.DocumentoParcial), c.DatosCompletos,
		string(requisitos), string(meritos), c.Puntuacion.Micropuntos()}
	args = append(args, argumentosMaterial(c.Material)...)
	var resultado ports.ResultadoGuardado
	var micropuntos int64
	err := r.ejecutar(ctx, sqlConMaterial(`SELECT reutilizada, solicitud_ref, version, puntuacion_micropuntos, datos_completos FROM vec_seleccion.guardar_borrador_propio_v1(
		$1::text,$2::text,$3::integer,$4::integer,$5::text,$6::text,$7::text,$8::text,$9::text,$10::bytea,$11::bytea,$12::text,$13::text,$14::boolean,$15::jsonb,$16::jsonb,$17::bigint,`, 17),
		args, &resultado.Reutilizada, &resultado.SolicitudRef, &resultado.Version, &micropuntos, &resultado.DatosCompletos)
	if err != nil {
		return ports.ResultadoGuardado{}, err
	}
	if resultado.Puntuacion, err = puntos(micropuntos); err != nil {
		return ports.ResultadoGuardado{}, err
	}
	return resultado, nil
}

// Presentar presenta la solicitud (o devuelve la repetición).
func (r *Repositorio) Presentar(ctx context.Context, c ports.ComandoPresentar) (ports.ResultadoPresentacion, error) {
	if !materialValido(c.Material) {
		return ports.ResultadoPresentacion{}, ports.ErrDatosNoValidos
	}
	args := append([]any{c.PersonaRef, c.SolicitudRef, c.VersionEsperada, c.Clave, c.HuellaMaterial, c.ReciboRef}, argumentosMaterial(c.Material)...)
	var resultado ports.ResultadoPresentacion
	var micropuntos int64
	err := r.ejecutar(ctx, sqlConMaterial(`SELECT reutilizada, solicitud_ref, numero_justificante, presentada_en, recibo_ref, puntuacion_micropuntos FROM vec_seleccion.presentar_solicitud_propia_v1(
		$1::text,$2::text,$3::integer,$4::text,$5::text,$6::text,`, 6),
		args, &resultado.Reutilizada, &resultado.SolicitudRef, &resultado.NumeroJustificante, &resultado.PresentadaEn, &resultado.ReciboRef, &micropuntos)
	if err != nil {
		return ports.ResultadoPresentacion{}, err
	}
	resultado.PresentadaEn = resultado.PresentadaEn.UTC()
	if resultado.Puntuacion, err = puntos(micropuntos); err != nil {
		return ports.ResultadoPresentacion{}, err
	}
	return resultado, nil
}

type resumenFila struct {
	SolicitudRef       string     `json:"solicitud_ref"`
	ConvocatoriaRef    string     `json:"convocatoria_ref"`
	ConvocatoriaTitulo string     `json:"convocatoria_titulo"`
	Estado             string     `json:"estado"`
	Version            int        `json:"version"`
	PresentadaEn       *time.Time `json:"presentada_en"`
	NumeroJustificante *string    `json:"numero_justificante"`
	Micropuntos        int64      `json:"puntuacion_micropuntos"`
}

// nulo lleva a NULL la cadena vacía (dato aún no aportado).
func nulo(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func texto(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func estado(e string) (domain.EstadoSolicitud, error) {
	switch domain.EstadoSolicitud(e) {
	case domain.EstadoBorrador, domain.EstadoPresentada:
		return domain.EstadoSolicitud(e), nil
	}
	return "", ports.ErrNoDisponible
}

// ListarPropias devuelve las solicitudes de la persona.
func (r *Repositorio) ListarPropias(ctx context.Context, persona string, m ports.MaterialConsumoV3) ([]ports.ResumenSolicitudPropia, error) {
	if !materialValido(m) {
		return nil, ports.ErrDatosNoValidos
	}
	var bruto []byte
	if err := r.ejecutar(ctx, sqlConMaterial(`SELECT vec_seleccion.listar_solicitudes_propias_v1($1::text,`, 1), append([]any{persona}, argumentosMaterial(m)...), &bruto); err != nil {
		return nil, err
	}
	var filas []resumenFila
	if json.Unmarshal(bruto, &filas) != nil {
		return nil, ports.ErrNoDisponible
	}
	resultado := make([]ports.ResumenSolicitudPropia, 0, len(filas))
	for _, f := range filas {
		e, err := estado(f.Estado)
		if err != nil {
			return nil, err
		}
		p, err := puntos(f.Micropuntos)
		if err != nil {
			return nil, err
		}
		fila := ports.ResumenSolicitudPropia{SolicitudRef: f.SolicitudRef, ConvocatoriaRef: f.ConvocatoriaRef, ConvocatoriaTitulo: f.ConvocatoriaTitulo,
			Estado: e, Version: f.Version, Puntuacion: p}
		if f.PresentadaEn != nil {
			t := f.PresentadaEn.UTC()
			fila.PresentadaEn = &t
		}
		if f.NumeroJustificante != nil {
			fila.NumeroJustificante = *f.NumeroJustificante
		}
		resultado = append(resultado, fila)
	}
	return resultado, nil
}

type versionFila struct {
	SolicitudRef        string                      `json:"solicitud_ref"`
	PersonaRef          string                      `json:"persona_ref"`
	ConvocatoriaRef     string                      `json:"convocatoria_ref"`
	ConvocatoriaVersion int                         `json:"convocatoria_version"`
	Version             int                         `json:"version"`
	Estado              string                      `json:"estado"`
	Turno               *string                     `json:"turno"`
	DatosCompletos      bool                        `json:"datos_completos"`
	ClaveRef            string                      `json:"datos_clave_ref"`
	Nonce               string                      `json:"datos_nonce"`
	Cifrado             string                      `json:"datos_cifrado"`
	Requisitos          []domain.RequisitoDeclarado `json:"requisitos"`
	Meritos             []domain.MeritoDeclarado    `json:"meritos"`
	Micropuntos         int64                       `json:"puntuacion_micropuntos"`
	ActualizadaEn       time.Time                   `json:"actualizada_en"`
}

// LeerBorrador devuelve la última versión propia en la convocatoria.
func (r *Repositorio) LeerBorrador(ctx context.Context, persona, convocatoria string, m ports.MaterialConsumoV3) (ports.VersionSolicitud, error) {
	if !materialValido(m) {
		return ports.VersionSolicitud{}, ports.ErrDatosNoValidos
	}
	var bruto []byte
	if err := r.ejecutar(ctx, sqlConMaterial(`SELECT vec_seleccion.leer_borrador_propio_v1($1::text,$2::text,`, 2), append([]any{persona, convocatoria}, argumentosMaterial(m)...), &bruto); err != nil {
		return ports.VersionSolicitud{}, err
	}
	if bruto == nil {
		return ports.VersionSolicitud{}, ports.ErrSinBorrador
	}
	var f versionFila
	if json.Unmarshal(bruto, &f) != nil {
		return ports.VersionSolicitud{}, ports.ErrNoDisponible
	}
	e, err := estado(f.Estado)
	if err != nil {
		return ports.VersionSolicitud{}, err
	}
	sobre, err := decodificarSobre(f.ClaveRef, f.Nonce, f.Cifrado)
	if err != nil {
		return ports.VersionSolicitud{}, err
	}
	p, err := puntos(f.Micropuntos)
	if err != nil {
		return ports.VersionSolicitud{}, err
	}
	return ports.VersionSolicitud{SolicitudRef: f.SolicitudRef, PersonaRef: f.PersonaRef, ConvocatoriaRef: f.ConvocatoriaRef,
		ConvocatoriaVersion: f.ConvocatoriaVersion, Version: f.Version, Estado: e, Turno: texto(f.Turno), DatosCompletos: f.DatosCompletos, Sobre: sobre,
		Requisitos: f.Requisitos, Meritos: f.Meritos, Puntuacion: p, ActualizadaEn: f.ActualizadaEn.UTC()}, nil
}

type filaRRHH struct {
	PresentacionID     int64     `json:"presentacion_id"`
	SolicitudRef       string    `json:"solicitud_ref"`
	PersonaRef         string    `json:"persona_ref"`
	ConvocatoriaRef    string    `json:"convocatoria_ref"`
	Version            int       `json:"version"`
	NumeroJustificante string    `json:"numero_justificante"`
	Turno              string    `json:"turno"`
	ClaveRef           string    `json:"datos_clave_ref"`
	Nonce              string    `json:"datos_nonce"`
	Cifrado            string    `json:"datos_cifrado"`
	DocumentoParcial   string    `json:"documento_parcial"`
	PresentadaEn       time.Time `json:"presentada_en"`
	Micropuntos        int64     `json:"puntuacion_micropuntos"`
}

func (f filaRRHH) aPuerto() (ports.FilaSolicitudRRHH, error) {
	sobre, err := decodificarSobre(f.ClaveRef, f.Nonce, f.Cifrado)
	if err != nil {
		return ports.FilaSolicitudRRHH{}, err
	}
	p, err := puntos(f.Micropuntos)
	if err != nil {
		return ports.FilaSolicitudRRHH{}, err
	}
	return ports.FilaSolicitudRRHH{PresentacionID: f.PresentacionID, SolicitudRef: f.SolicitudRef, PersonaRef: f.PersonaRef,
		ConvocatoriaRef: f.ConvocatoriaRef, Version: f.Version, NumeroJustificante: f.NumeroJustificante, Turno: f.Turno, Sobre: sobre,
		DocumentoParcial: f.DocumentoParcial, PresentadaEn: f.PresentadaEn.UTC(), Puntuacion: p}, nil
}

// ListarPresentadas devuelve una página del listado de RRHH (auditado).
func (r *Repositorio) ListarPresentadas(ctx context.Context, convocatoria string, cursor int64, limite int, m ports.MaterialConsumoV3) ([]ports.FilaSolicitudRRHH, error) {
	if !materialValido(m) {
		return nil, ports.ErrDatosNoValidos
	}
	var bruto []byte
	if err := r.ejecutar(ctx, sqlConMaterial(`SELECT vec_seleccion.listar_solicitudes_convocatoria_v1($1::text,$2::bigint,$3::integer,`, 3),
		append([]any{convocatoria, cursor, limite}, argumentosMaterial(m)...), &bruto); err != nil {
		return nil, err
	}
	var filas []filaRRHH
	if json.Unmarshal(bruto, &filas) != nil || len(filas) > limite {
		return nil, ports.ErrNoDisponible
	}
	resultado := make([]ports.FilaSolicitudRRHH, 0, len(filas))
	for _, f := range filas {
		fila, err := f.aPuerto()
		if err != nil {
			return nil, err
		}
		resultado = append(resultado, fila)
	}
	return resultado, nil
}

type fichaFila struct {
	filaRRHH
	ConvocatoriaTitulo      string                      `json:"convocatoria_titulo"`
	ConvocatoriaVersion     int                         `json:"convocatoria_version"`
	ConvocatoriaContenido   json.RawMessage             `json:"convocatoria_contenido"`
	ConvocatoriaAbreEn      time.Time                   `json:"convocatoria_abre_en"`
	ConvocatoriaCierraEn    time.Time                   `json:"convocatoria_cierra_en"`
	ConvocatoriaPublicadaEn time.Time                   `json:"convocatoria_publicada_en"`
	Requisitos              []domain.RequisitoDeclarado `json:"requisitos"`
	Meritos                 []domain.MeritoDeclarado    `json:"meritos"`
	Historia                []struct {
		Tipo    string    `json:"tipo"`
		Version int       `json:"version"`
		En      time.Time `json:"en"`
	} `json:"historia"`
}

// LeerFicha devuelve la ficha completa de una solicitud presentada (acceso
// auditado en la base).
func (r *Repositorio) LeerFicha(ctx context.Context, solicitud string, m ports.MaterialConsumoV3) (ports.FichaSolicitudRRHH, error) {
	if !materialValido(m) {
		return ports.FichaSolicitudRRHH{}, ports.ErrDatosNoValidos
	}
	var bruto []byte
	if err := r.ejecutar(ctx, sqlConMaterial(`SELECT vec_seleccion.leer_detalle_solicitud_v1($1::text,`, 1), append([]any{solicitud}, argumentosMaterial(m)...), &bruto); err != nil {
		return ports.FichaSolicitudRRHH{}, err
	}
	var f fichaFila
	if json.Unmarshal(bruto, &f) != nil {
		return ports.FichaSolicitudRRHH{}, ports.ErrNoDisponible
	}
	fila, err := f.aPuerto()
	if err != nil {
		return ports.FichaSolicitudRRHH{}, err
	}
	convocatoria, err := domain.ConvocatoriaDesdeContenido(f.ConvocatoriaRef, f.ConvocatoriaTitulo, f.ConvocatoriaAbreEn, f.ConvocatoriaCierraEn, f.ConvocatoriaPublicadaEn, f.ConvocatoriaContenido)
	if err != nil {
		return ports.FichaSolicitudRRHH{}, ports.ErrNoDisponible
	}
	ficha := ports.FichaSolicitudRRHH{Fila: fila, Convocatoria: domain.ConvocatoriaPublicada{Convocatoria: convocatoria, Version: f.ConvocatoriaVersion},
		Requisitos: f.Requisitos, Meritos: f.Meritos, Historia: make([]ports.EventoHistoria, 0, len(f.Historia))}
	for _, h := range f.Historia {
		ficha.Historia = append(ficha.Historia, ports.EventoHistoria{Tipo: h.Tipo, Version: h.Version, En: h.En.UTC()})
	}
	return ficha, nil
}

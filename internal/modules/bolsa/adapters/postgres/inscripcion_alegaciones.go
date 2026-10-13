package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
)

const consultaPresentarAlegacion = `SELECT vec_bolsa_llamamientos.presentar_alegacion_v1(
	$1::text,$2::jsonb,$3::bytea,$4::bytea,$5::bytea,$6::bytea,$7::bytea,
	$8::numeric,$9::numeric,$10::bytea,$11::bytea,$12::bytea,$13::bytea)`

const consultaLecturaAlegaciones = `SELECT vec_bolsa_llamamientos.consultar_alegaciones_v1(
	$1::text,$2::jsonb,$3::bytea,$4::bytea,$5::jsonb)`

type respuestaAlegacionSQL struct {
	Resultado      string                                           `json:"resultado"`
	Proyeccion     json.RawMessage                                  `json:"proyeccion"`
	SujetosB7      map[string]inscripcion.MetadatoNombreAlegacionB7 `json:"sujetos_b7"`
	AuditoriaRef   string                                           `json:"auditoria_ref"`
	CorrelacionRef string                                           `json:"correlacion_ref"`
	ConsultadaEn   time.Time                                        `json:"consultada_en"`
}

var _ inscripcion.RepositorioAlegaciones = (*RepositorioInscripcionesPostgreSQL)(nil)

func (r *RepositorioInscripcionesPostgreSQL) PresentarAlegacion(ctx context.Context, actor inscripcion.Actor,
	p inscripcion.PresentacionAlegacion) (inscripcion.ReciboAlegacion, error) {
	if r == nil || valorNulo(r.externo) || p.Validar() != nil || !actor.EscrituraValida() {
		return inscripcion.ReciboAlegacion{}, inscripcion.ErrSolicitudInvalida
	}
	captura, err := capturaEscrituraInscripcion(actor, inscripcion.AccionPresentarAlegacion,
		"vec_bolsa_llamamientos.alegacion.presentar.v1", "externa_personal")
	if err != nil {
		return inscripcion.ReciboAlegacion{}, err
	}
	material, huella, err := inscripcion.MaterialAlegacion(p)
	if err != nil {
		return inscripcion.ReciboAlegacion{}, err
	}
	recurso := actor.RecursoEscrituraCanonico
	hr := sha256.Sum256(recurso)
	ref, err := inscripcion.ReferenciaAlegacion(actor.PersonaRef, p.ClaveIdempotencia)
	resumen := actor.MaterialEscritura.ResumenCapacidad()
	if err != nil || resumen.EfectoRef() != ref || resumen.EfectoHuellaSHA256() != hex.EncodeToString(hr[:]) ||
		resumen.Operacion() != inscripcion.AccionPresentarAlegacion || huella == "" {
		return inscripcion.ReciboAlegacion{}, inscripcion.ErrAccesoDenegado
	}
	argumentos := append([]any{string(material), captura, recurso}, argumentosV3Inscripcion(actor)...)
	var recibo inscripcion.ReciboAlegacion
	_, err = transaccionInscripcion(ctx, r.externo, func(tx pgx.Tx) ([]byte, error) {
		var raw []byte
		err := tx.QueryRow(ctx, consultaPresentarAlegacion, argumentos...).Scan(&raw)
		return raw, err
	}, func(raw []byte) error {
		recibo = inscripcion.ReciboAlegacion{}
		if decodificarInscripcionEstricta(raw, &recibo) != nil || recibo.Alegacion.Validar() != nil ||
			recibo.AlegacionRef != ref || recibo.Objeto != p.Objeto || recibo.Fundamento != p.Fundamento ||
			recibo.PersonaResumen != "" {
			return inscripcion.ErrNoDisponible
		}
		return nil
	})
	if err != nil {
		return inscripcion.ReciboAlegacion{}, err
	}
	return recibo, nil
}

func consultarAlegaciones[T any](ctx context.Context, r *RepositorioInscripcionesPostgreSQL,
	actor inscripcion.Actor, accion, ref, tipo string, limite int, cursor string,
	validar func(T) error,
	adjuntar func(*T, map[string]inscripcion.MetadatoNombreAlegacionB7) error) (T, error) {
	var cero T
	if r == nil || validar == nil || limite < 1 || limite > 20 {
		return cero, inscripcion.ErrSolicitudInvalida
	}
	filtro := inscripcion.Filtro{Limite: limite, Cursor: cursor}
	id := actor.Idioma
	if id == "" {
		id = "es"
	}
	recurso, err := inscripcion.RecursoLectura(accion, actor.PersonaRef, id, filtro, ref)
	if err != nil {
		return cero, err
	}
	contexto, vinculo, captura, err := prepararCapturaLecturaInscripcion(actor, accion, recurso, filtro)
	if err != nil {
		return cero, err
	}
	lector, err := r.seleccionarLectorInscripcion(actor.Canal, accion)
	if err != nil {
		return cero, err
	}
	selector, err := json.Marshal(struct {
		Tipo   string `json:"tipo"`
		Ref    string `json:"ref"`
		Cursor string `json:"cursor"`
		Limite int    `json:"limite"`
	}{tipo, ref, cursor, limite})
	if err != nil {
		return cero, inscripcion.ErrSolicitudInvalida
	}
	var salida respuestaAlegacionSQL
	var proyeccion T
	_, err = transaccionInscripcion(ctx, lector, func(tx pgx.Tx) ([]byte, error) {
		var raw []byte
		err := tx.QueryRow(ctx, consultaLecturaAlegaciones, accion, selector, contexto, vinculo, captura).Scan(&raw)
		return raw, err
	}, func(raw []byte) error {
		salida, proyeccion = respuestaAlegacionSQL{}, cero
		if decodificarInscripcionEstricta(raw, &salida) != nil || salida.SujetosB7 == nil ||
			salida.AuditoriaRef == "" || salida.CorrelacionRef != actor.Lectura.CorrelacionRef || salida.ConsultadaEn.IsZero() {
			return inscripcion.ErrNoDisponible
		}
		if salida.Resultado == "no_encontrada" {
			if !bytes.Equal(bytes.TrimSpace(salida.Proyeccion), []byte("null")) || len(salida.SujetosB7) != 0 {
				return inscripcion.ErrNoDisponible
			}
			return nil
		}
		if salida.Resultado != "obtenida" || validarCamposProyeccionInscripcion(salida.Proyeccion, actor.Lectura.Campos) != nil ||
			decodificarInscripcionEstricta(salida.Proyeccion, &proyeccion) != nil || validar(proyeccion) != nil {
			return inscripcion.ErrNoDisponible
		}
		if adjuntar != nil {
			return adjuntar(&proyeccion, salida.SujetosB7)
		}
		if len(salida.SujetosB7) != 0 {
			return inscripcion.ErrNoDisponible
		}
		return nil
	})
	if err != nil {
		return cero, err
	}
	if salida.Resultado == "no_encontrada" {
		return cero, inscripcion.ErrNoEncontrada
	}
	return proyeccion, nil
}

func (r *RepositorioInscripcionesPostgreSQL) ObjetosAlegables(ctx context.Context, actor inscripcion.Actor,
	limite int, cursor string) (inscripcion.PaginaObjetosAlegacion, error) {
	return consultarAlegaciones(ctx, r, actor, inscripcion.AccionObjetosAlegacion, "objetos", "objetos", limite, cursor,
		func(p inscripcion.PaginaObjetosAlegacion) error {
			if p.Objetos == nil || len(p.Objetos) > limite || uint64(len(p.Objetos)) > p.Total {
				return inscripcion.ErrNoDisponible
			}
			for _, o := range p.Objetos {
				if o.Validar() != nil {
					return inscripcion.ErrNoDisponible
				}
			}
			return nil
		}, nil)
}

func (r *RepositorioInscripcionesPostgreSQL) AlegacionesPropias(ctx context.Context, actor inscripcion.Actor,
	limite int, cursor string) (inscripcion.PaginaAlegaciones, error) {
	return consultarAlegaciones(ctx, r, actor, inscripcion.AccionAlegacionesPropias, "propias", "propias", limite, cursor,
		func(p inscripcion.PaginaAlegaciones) error { return validarPaginaAlegaciones(p, limite, false) }, nil)
}

func (r *RepositorioInscripcionesPostgreSQL) AlegacionesRRHH(ctx context.Context, actor inscripcion.Actor,
	tipo, ref string, limite int, cursor string) (inscripcion.PaginaAlegaciones, error) {
	return consultarAlegaciones(ctx, r, actor, inscripcion.AccionAlegacionesRRHH, tipo+":"+ref, tipo, limite, cursor,
		func(p inscripcion.PaginaAlegaciones) error { return validarPaginaAlegaciones(p, limite, true) },
		adjuntarMetadatosB7Pagina)
}

func (r *RepositorioInscripcionesPostgreSQL) AlegacionRRHH(ctx context.Context, actor inscripcion.Actor,
	ref string) (inscripcion.Alegacion, error) {
	return consultarAlegaciones(ctx, r, actor, inscripcion.AccionAlegacionesRRHH, ref, "detalle", 1, "",
		func(a inscripcion.Alegacion) error { return a.Validar() }, adjuntarMetadatosB7Detalle)
}

func (r *RepositorioInscripcionesPostgreSQL) AmbitosAlegacionesRRHH(ctx context.Context, actor inscripcion.Actor,
	limite int, cursor string) (inscripcion.PaginaAmbitosAlegacionRRHH, error) {
	return consultarAlegaciones(ctx, r, actor, inscripcion.AccionAlegacionesRRHH, "ambitos", "ambitos", limite, cursor,
		func(p inscripcion.PaginaAmbitosAlegacionRRHH) error {
			if p.Ambitos == nil || len(p.Ambitos) > limite || uint64(len(p.Ambitos)) > p.Total {
				return inscripcion.ErrNoDisponible
			}
			for _, a := range p.Ambitos {
				if (a.Tipo != "convocatoria" && a.Tipo != "bolsa") || a.Referencia == "" || a.Titulo == "" {
					return inscripcion.ErrNoDisponible
				}
			}
			return nil
		}, nil)
}

func adjuntarMetadatosB7Pagina(p *inscripcion.PaginaAlegaciones,
	sujetos map[string]inscripcion.MetadatoNombreAlegacionB7) error {
	if p == nil || len(sujetos) > len(p.Alegaciones) {
		return inscripcion.ErrNoDisponible
	}
	for i := range p.Alegaciones {
		m, ok := sujetos[p.Alegaciones[i].AlegacionRef]
		if !ok {
			continue
		}
		if !metadatoNombreB7Valido(p.Alegaciones[i], m) {
			return inscripcion.ErrNoDisponible
		}
		p.Alegaciones[i].MetadatoNombreB7 = &m
		delete(sujetos, p.Alegaciones[i].AlegacionRef)
	}
	if len(sujetos) != 0 {
		return inscripcion.ErrNoDisponible
	}
	return nil
}

func adjuntarMetadatosB7Detalle(a *inscripcion.Alegacion,
	sujetos map[string]inscripcion.MetadatoNombreAlegacionB7) error {
	if a == nil || len(sujetos) > 1 {
		return inscripcion.ErrNoDisponible
	}
	m, ok := sujetos[a.AlegacionRef]
	if !ok {
		if len(sujetos) != 0 {
			return inscripcion.ErrNoDisponible
		}
		return nil
	}
	if !metadatoNombreB7Valido(*a, m) {
		return inscripcion.ErrNoDisponible
	}
	a.MetadatoNombreB7 = &m
	return nil
}

func metadatoNombreB7Valido(a inscripcion.Alegacion, m inscripcion.MetadatoNombreAlegacionB7) bool {
	if a.Objeto.Tipo != "acto_participacion" || m.ActaRef != a.Objeto.ActoRef ||
		m.ParticipacionRef != a.Objeto.ParticipacionRef || m.FilaNumero < 1 ||
		len(m.CategoriaRef) < 1 || len(m.CategoriaRef) > 512 || len(m.HuellaListadoSHA256) != 64 {
		return false
	}
	for _, c := range m.HuellaListadoSHA256 {
		if c < '0' || c > '9' && c < 'a' || c > 'f' {
			return false
		}
	}
	return true
}

func validarPaginaAlegaciones(p inscripcion.PaginaAlegaciones, limite int, rrhh bool) error {
	if p.Alegaciones == nil || len(p.Alegaciones) > limite || uint64(len(p.Alegaciones)) > p.Total {
		return inscripcion.ErrNoDisponible
	}
	for _, a := range p.Alegaciones {
		if a.Validar() != nil || !rrhh && a.PersonaResumen != "" {
			return inscripcion.ErrNoDisponible
		}
	}
	return nil
}

package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
)

const consultaLecturaInscripcion = `SELECT vec_bolsa_llamamientos.consultar_inscripcion_v1(
	$1::text,$2::jsonb,$3::bytea,$4::bytea,$5::jsonb)`

type sobreLecturaInscripcion struct {
	Resultado      string          `json:"resultado"`
	Proyeccion     json.RawMessage `json:"proyeccion"`
	AuditoriaRef   string          `json:"auditoria_ref"`
	CorrelacionRef string          `json:"correlacion_ref"`
	ConsultadaEn   time.Time       `json:"consultada_en"`
}

func consultarInscripcion[T any](ctx context.Context, r *RepositorioInscripcionesPostgreSQL,
	actor inscripcion.Actor, accion, recurso string, filtro inscripcion.Filtro,
	selector any, validar func(T) error) (T, error) {
	var cero T
	if r == nil || validar == nil {
		return cero, inscripcion.ErrNoDisponible
	}
	contexto, vinculo, captura, err := prepararCapturaLecturaInscripcion(actor, accion, recurso, filtro)
	if err != nil {
		return cero, err
	}
	lector, err := r.seleccionarLectorInscripcion(actor.Canal, accion)
	if err != nil {
		return cero, err
	}
	selectorJSON, err := json.Marshal(selector)
	if err != nil || len(selectorJSON) == 0 || len(selectorJSON) > 2048 {
		return cero, inscripcion.ErrSolicitudInvalida
	}
	var salida sobreLecturaInscripcion
	var proyeccion T
	_, err = transaccionInscripcion(ctx, lector, func(tx pgx.Tx) ([]byte, error) {
		var respuesta []byte
		err := tx.QueryRow(ctx, consultaLecturaInscripcion, accion, selectorJSON, contexto, vinculo, captura).Scan(&respuesta)
		return respuesta, err
	}, func(respuesta []byte) error {
		salida = sobreLecturaInscripcion{}
		proyeccion = cero
		if decodificarInscripcionEstricta(respuesta, &salida) != nil ||
			salida.AuditoriaRef == "" || salida.CorrelacionRef != actor.Lectura.CorrelacionRef ||
			salida.ConsultadaEn.IsZero() {
			return inscripcion.ErrNoDisponible
		}
		switch salida.Resultado {
		case "no_encontrada":
			if !bytes.Equal(bytes.TrimSpace(salida.Proyeccion), []byte("null")) {
				return inscripcion.ErrNoDisponible
			}
			return nil
		case "obtenida":
			if len(salida.Proyeccion) == 0 || bytes.Equal(bytes.TrimSpace(salida.Proyeccion), []byte("null")) ||
				validarCamposProyeccionInscripcion(salida.Proyeccion, actor.Lectura.Campos) != nil ||
				decodificarInscripcionEstricta(salida.Proyeccion, &proyeccion) != nil || validar(proyeccion) != nil {
				return inscripcion.ErrNoDisponible
			}
			return nil
		default:
			return inscripcion.ErrNoDisponible
		}
	})
	if err != nil {
		return cero, err
	}
	if salida.Resultado == "no_encontrada" {
		return cero, inscripcion.ErrNoEncontrada
	}
	return proyeccion, nil
}

// La decisión central concede hojas concretas. Cualquier hoja nueva en la
// proyección revierte también el asiento de auditoría de esta transacción.
func validarCamposProyeccionInscripcion(raw []byte, campos []string) error {
	if len(raw) == 0 || len(raw) > 2*1024*1024 || len(campos) == 0 {
		return inscripcion.ErrNoDisponible
	}
	permitidos := make(map[string]struct{}, len(campos))
	for _, campo := range campos {
		if campo == "" {
			return inscripcion.ErrNoDisponible
		}
		permitidos[campo] = struct{}{}
	}
	var documento any
	if json.Unmarshal(raw, &documento) != nil {
		return inscripcion.ErrNoDisponible
	}
	if _, ok := documento.(map[string]any); !ok {
		return inscripcion.ErrNoDisponible
	}
	prefijoPermitido := func(ruta string) bool {
		if ruta == "" {
			return true
		}
		for campo := range permitidos {
			if strings.HasPrefix(campo, ruta+".") || strings.HasPrefix(campo, ruta+"[]") || campo == ruta {
				return true
			}
		}
		return false
	}
	nodos := 0
	var visitar func(any, string, int) bool
	visitar = func(valor any, ruta string, profundidad int) bool {
		nodos++
		if nodos > 100000 || profundidad > 16 {
			return false
		}
		switch v := valor.(type) {
		case map[string]any:
			if len(v) == 0 && !prefijoPermitido(ruta) {
				return false
			}
			for clave, hijo := range v {
				siguiente := clave
				if ruta != "" {
					siguiente = ruta + "." + clave
				}
				if !visitar(hijo, siguiente, profundidad+1) {
					return false
				}
			}
			return true
		case []any:
			if len(v) == 0 && !prefijoPermitido(ruta) {
				return false
			}
			for _, hijo := range v {
				if !visitar(hijo, ruta+"[]", profundidad+1) {
					return false
				}
			}
			return true
		default:
			_, autorizado := permitidos[ruta]
			return autorizado
		}
	}
	if !visitar(documento, "", 0) {
		return inscripcion.ErrNoDisponible
	}
	return nil
}

func (r *RepositorioInscripcionesPostgreSQL) seleccionarLectorInscripcion(canal, accion string) (iniciadorTransacciones, error) {
	if r == nil {
		return nil, inscripcion.ErrNoDisponible
	}
	var lector iniciadorTransacciones
	switch accion {
	case inscripcion.AccionListarAbiertas, inscripcion.AccionDetalleAbierta,
		inscripcion.AccionListarPropias, inscripcion.AccionDetallePropia:
		if canal != "externa_personal" {
			return nil, inscripcion.ErrAccesoDenegado
		}
		lector = r.lectorExterno
	case inscripcion.AccionListarRRHH, inscripcion.AccionConvocatoriasRRHH, inscripcion.AccionDetalleRRHH, inscripcion.AccionMotivosRRHH:
		if canal != "interna_corporativa" {
			return nil, inscripcion.ErrAccesoDenegado
		}
		lector = r.lectorRRHH
	default:
		return nil, inscripcion.ErrAccesoDenegado
	}
	if valorNulo(lector) {
		return nil, inscripcion.ErrNoDisponible
	}
	return lector, nil
}

type selectorListaInscripcion struct {
	Limite          int    `json:"limite"`
	Cursor          string `json:"cursor"`
	Estado          string `json:"estado"`
	ConvocatoriaRef string `json:"convocatoria_ref"`
}

type selectorAbiertasInscripcion struct {
	Limite int    `json:"limite"`
	Cursor string `json:"cursor"`
}

type selectorDetalleInscripcion struct {
	SolicitudRef string `json:"solicitud_ref"`
}

type selectorConvocatoriaInscripcion struct {
	ConvocatoriaRef string `json:"convocatoria_ref"`
}

type selectorMotivosInscripcion struct {
	Decision string `json:"decision"`
}

func recursoInscripcionLectura(actor inscripcion.Actor, accion string, filtro inscripcion.Filtro, ref string) (string, error) {
	recurso, err := inscripcion.RecursoLectura(accion, actor.PersonaRef, idiomaInscripcion(actor.Idioma), filtro, ref)
	if err != nil || actor.Lectura == nil || actor.Lectura.RecursoRef != recurso {
		return "", inscripcion.ErrAccesoDenegado
	}
	return recurso, nil
}

func (r *RepositorioInscripcionesPostgreSQL) Abiertas(ctx context.Context, actor inscripcion.Actor, limite int, cursor string) (inscripcion.PaginaAbiertas, error) {
	filtro := inscripcion.Filtro{Limite: limite, Cursor: cursor}
	if filtro.Validar() != nil {
		return inscripcion.PaginaAbiertas{}, inscripcion.ErrSolicitudInvalida
	}
	recurso, err := recursoInscripcionLectura(actor, inscripcion.AccionListarAbiertas, filtro, "")
	if err != nil {
		return inscripcion.PaginaAbiertas{}, err
	}
	return consultarInscripcion(ctx, r, actor, inscripcion.AccionListarAbiertas, recurso, filtro,
		selectorAbiertasInscripcion{Limite: limite, Cursor: cursor}, func(p inscripcion.PaginaAbiertas) error {
			if len(p.Bolsas) > limite || uint64(len(p.Bolsas)) > p.Total {
				return inscripcion.ErrNoDisponible
			}
			for _, bolsa := range p.Bolsas {
				if validarBolsaAbiertaInscripcion(bolsa, false) != nil {
					return inscripcion.ErrNoDisponible
				}
			}
			return nil
		})
}

func (r *RepositorioInscripcionesPostgreSQL) DetalleAbierta(ctx context.Context, actor inscripcion.Actor, ref string) (inscripcion.BolsaAbierta, error) {
	recurso, err := recursoInscripcionLectura(actor, inscripcion.AccionDetalleAbierta, inscripcion.Filtro{}, ref)
	if err != nil {
		return inscripcion.BolsaAbierta{}, err
	}
	return consultarInscripcion(ctx, r, actor, inscripcion.AccionDetalleAbierta, recurso, inscripcion.Filtro{},
		selectorConvocatoriaInscripcion{ConvocatoriaRef: ref}, func(b inscripcion.BolsaAbierta) error {
			if b.ConvocatoriaRef != ref || validarBolsaAbiertaInscripcion(b, true) != nil {
				return inscripcion.ErrNoDisponible
			}
			return nil
		})
}

func validarBolsaAbiertaInscripcion(b inscripcion.BolsaAbierta, detalle bool) error {
	if b.ConvocatoriaRef == "" || b.Titulo == "" || b.NumeroCategorias < 1 || b.NumeroCategorias > 128 ||
		b.CatalogoVersion == 0 || b.PlazoInicio.IsZero() || !b.PlazoFin.After(b.PlazoInicio) ||
		b.RequisitosResumen == "" || (b.EstadoPropio == nil) != (b.SolicitudRef == nil) ||
		!b.PuedeIniciar && b.ImpedimentoEtiqueta == "" {
		return inscripcion.ErrNoDisponible
	}
	if detalle {
		if uint64(len(b.Categorias)) != b.NumeroCategorias {
			return inscripcion.ErrNoDisponible
		}
		vistas := make(map[string]struct{}, len(b.Categorias))
		for _, categoria := range b.Categorias {
			if categoria.CategoriaRef == "" || len(categoria.CategoriaRef) > 200 ||
				categoria.Categoria == "" || len(categoria.Categoria) > 2048 {
				return inscripcion.ErrNoDisponible
			}
			if _, repetida := vistas[categoria.CategoriaRef]; repetida {
				return inscripcion.ErrNoDisponible
			}
			vistas[categoria.CategoriaRef] = struct{}{}
		}
	} else if len(b.Categorias) != 0 {
		return inscripcion.ErrNoDisponible
	}
	return nil
}

func (r *RepositorioInscripcionesPostgreSQL) Propias(ctx context.Context, actor inscripcion.Actor, filtro inscripcion.Filtro) (inscripcion.Pagina, error) {
	if filtro.Validar() != nil {
		return inscripcion.Pagina{}, inscripcion.ErrSolicitudInvalida
	}
	recurso, err := recursoInscripcionLectura(actor, inscripcion.AccionListarPropias, filtro, "")
	if err != nil {
		return inscripcion.Pagina{}, err
	}
	return consultarInscripcion(ctx, r, actor, inscripcion.AccionListarPropias, recurso, filtro,
		selectorListaInscripcion{Limite: filtro.Limite, Cursor: filtro.Cursor, Estado: filtro.Estado,
			ConvocatoriaRef: filtro.ConvocatoriaRef}, validarPaginaInscripcion(filtro.Limite))
}

func (r *RepositorioInscripcionesPostgreSQL) Propia(ctx context.Context, actor inscripcion.Actor, ref string) (inscripcion.Solicitud, error) {
	recurso, err := recursoInscripcionLectura(actor, inscripcion.AccionDetallePropia, inscripcion.Filtro{}, ref)
	if err != nil {
		return inscripcion.Solicitud{}, err
	}
	return consultarInscripcion(ctx, r, actor, inscripcion.AccionDetallePropia, recurso, inscripcion.Filtro{},
		selectorDetalleInscripcion{SolicitudRef: ref}, validarSolicitudInscripcion(ref, false))
}

func (r *RepositorioInscripcionesPostgreSQL) PendientesRRHH(ctx context.Context, actor inscripcion.Actor, filtro inscripcion.Filtro) (inscripcion.Pagina, error) {
	if filtro.ConvocatoriaRef == "" || filtro.Validar() != nil {
		return inscripcion.Pagina{}, inscripcion.ErrSolicitudInvalida
	}
	recurso, err := recursoInscripcionLectura(actor, inscripcion.AccionListarRRHH, filtro, "")
	if err != nil {
		return inscripcion.Pagina{}, err
	}
	return consultarInscripcion(ctx, r, actor, inscripcion.AccionListarRRHH, recurso, filtro,
		selectorListaInscripcion{Limite: filtro.Limite, Cursor: filtro.Cursor, Estado: filtro.Estado,
			ConvocatoriaRef: filtro.ConvocatoriaRef}, func(p inscripcion.Pagina) error {
			if p.ConvocatoriaTitulo == "" || utf8.RuneCountInString(p.ConvocatoriaTitulo) > 180 {
				return inscripcion.ErrNoDisponible
			}
			return validarPaginaInscripcion(filtro.Limite)(p)
		})
}

func (r *RepositorioInscripcionesPostgreSQL) ConvocatoriasRRHH(ctx context.Context, actor inscripcion.Actor, limite int, cursor string) (inscripcion.PaginaConvocatoriasGestion, error) {
	filtro := inscripcion.Filtro{Limite: limite, Cursor: cursor}
	if limite < 1 || limite > 100 || cursor != "" && !inscripcion.ConvocatoriaRefValida(cursor) {
		return inscripcion.PaginaConvocatoriasGestion{}, inscripcion.ErrSolicitudInvalida
	}
	recurso, err := recursoInscripcionLectura(actor, inscripcion.AccionConvocatoriasRRHH, filtro, "")
	if err != nil {
		return inscripcion.PaginaConvocatoriasGestion{}, err
	}
	return consultarInscripcion(ctx, r, actor, inscripcion.AccionConvocatoriasRRHH, recurso, filtro,
		selectorAbiertasInscripcion{Limite: limite, Cursor: cursor}, func(p inscripcion.PaginaConvocatoriasGestion) error {
			if len(p.Convocatorias) > limite || uint64(len(p.Convocatorias)) > p.Total ||
				p.CursorSiguiente != nil && !inscripcion.ConvocatoriaRefValida(*p.CursorSiguiente) {
				return inscripcion.ErrNoDisponible
			}
			for _, convocatoria := range p.Convocatorias {
				if !inscripcion.ConvocatoriaRefValida(convocatoria.ConvocatoriaRef) ||
					convocatoria.Titulo == "" || utf8.RuneCountInString(convocatoria.Titulo) > 180 ||
					convocatoria.CategoriasResumen == "" || len(convocatoria.CategoriasResumen) > 2048 ||
					convocatoria.PlazoFin.IsZero() || convocatoria.EstadoPublicacion == "" {
					return inscripcion.ErrNoDisponible
				}
			}
			return nil
		})
}

func (r *RepositorioInscripcionesPostgreSQL) DetalleRRHH(ctx context.Context, actor inscripcion.Actor, ref string) (inscripcion.Solicitud, error) {
	recurso, err := recursoInscripcionLectura(actor, inscripcion.AccionDetalleRRHH, inscripcion.Filtro{}, ref)
	if err != nil {
		return inscripcion.Solicitud{}, err
	}
	return consultarInscripcion(ctx, r, actor, inscripcion.AccionDetalleRRHH, recurso, inscripcion.Filtro{},
		selectorDetalleInscripcion{SolicitudRef: ref}, validarSolicitudInscripcion(ref, true))
}

func (r *RepositorioInscripcionesPostgreSQL) MotivosRRHH(ctx context.Context, actor inscripcion.Actor, decision string) (inscripcion.CatalogoMotivos, error) {
	if decision != "admitir" && decision != "rechazar" {
		return inscripcion.CatalogoMotivos{}, inscripcion.ErrSolicitudInvalida
	}
	recurso, err := recursoInscripcionLectura(actor, inscripcion.AccionMotivosRRHH, inscripcion.Filtro{}, decision)
	if err != nil {
		return inscripcion.CatalogoMotivos{}, err
	}
	return consultarInscripcion(ctx, r, actor, inscripcion.AccionMotivosRRHH, recurso, inscripcion.Filtro{},
		selectorMotivosInscripcion{Decision: decision}, func(c inscripcion.CatalogoMotivos) error {
			if c.Version == 0 {
				return inscripcion.ErrNoDisponible
			}
			for _, motivo := range c.Motivos {
				if motivo.Codigo == "" || motivo.Etiqueta == "" {
					return inscripcion.ErrNoDisponible
				}
			}
			return nil
		})
}

func validarSolicitudInscripcion(ref string, detalleRRHH bool) func(inscripcion.Solicitud) error {
	return func(s inscripcion.Solicitud) error {
		if s.SolicitudRef != ref || s.Validar() != nil || s.DeclaracionRef == "" {
			return inscripcion.ErrNoDisponible
		}
		if detalleRRHH && (s.BasesRef == "" || s.CatalogoVersion == 0 || s.PlazoInicio == nil ||
			s.PlazoFin == nil || !s.PlazoFin.After(*s.PlazoInicio) || s.Requisitos == nil) {
			return inscripcion.ErrNoDisponible
		}
		if detalleRRHH {
			for _, requisito := range s.Requisitos {
				if requisito.Codigo == "" || requisito.Descripcion == "" || requisito.MotivoEtiqueta == "" ||
					(requisito.Estado != "cumple" && requisito.Estado != "no_cumple" && requisito.Estado != "pendiente") ||
					(requisito.HitoCumplimiento == nil) != (requisito.HitoEtiqueta == nil) {
					return inscripcion.ErrNoDisponible
				}
			}
		}
		return nil
	}
}

func validarPaginaInscripcion(limite int) func(inscripcion.Pagina) error {
	return func(p inscripcion.Pagina) error {
		if len(p.Solicitudes) > limite || uint64(len(p.Solicitudes)) > p.Total {
			return inscripcion.ErrNoDisponible
		}
		for _, s := range p.Solicitudes {
			if s.Validar() != nil || s.DeclaracionRef == "" {
				return inscripcion.ErrNoDisponible
			}
		}
		return nil
	}
}

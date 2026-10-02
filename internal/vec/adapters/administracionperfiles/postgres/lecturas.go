package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// FuenteLecturas proyecta exclusivamente resultados de fachadas nominales
// autorizadas. La transacción puede escribir consumo y auditoría, nunca actos.
type FuenteLecturas struct {
	autoridad *Autoridad
	fuente    ports.FuenteAutorizacion
}

var _ api.FuenteLecturas = (*FuenteLecturas)(nil)

func NuevaFuenteLecturas(autoridad *Autoridad, fuente ports.FuenteAutorizacion) (*FuenteLecturas, error) {
	if autoridad == nil || ausente(autoridad.pool) || ausente(autoridad.emisor) || ausente(autoridad.reloj) || ausente(fuente) {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return &FuenteLecturas{autoridad: autoridad, fuente: fuente}, nil
}

type lecturaJSON struct {
	Esquema         string `json:"esquema"`
	Consulta        string `json:"consulta"`
	OperacionRef    string `json:"operacion_ref"`
	ActorPersonaRef string `json:"actor_persona_ref"`
	ActorPerfilRef  string `json:"actor_perfil_ref"`
	PersonaRef      string `json:"persona_ref,omitempty"`
	PropuestaRef    string `json:"propuesta_ref,omitempty"`
	ReciboRef       string `json:"recibo_ref,omitempty"`
	Busqueda        string `json:"busqueda,omitempty"`
	Cursor          string `json:"cursor,omitempty"`
	Limite          int    `json:"limite"`
}

func (f *FuenteLecturas) ejecutarLectura(ctx context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles, x lecturaJSON, consulta string, validar func([]byte) error) error {
	if f == nil || f.autoridad == nil || ausente(f.fuente) {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := f.autoridad.disponible(ctx); err != nil {
		return err
	}
	if actor.Validar() != nil || evidencia.ValidarPara(actor) != nil {
		return domain.ErrAutorizacionDenegada
	}
	ahora := f.autoridad.reloj.Ahora()
	if !actor.Instantanea.VigenteEn(ahora) {
		return domain.ErrAutorizacionDenegada
	}
	instantanea, err := f.fuente.ObtenerInstantaneaAutorizacion(ctx, actor.PersonaRef, actor.PerfilActivoRef)
	if err != nil {
		return traducirErrorLectura(ctx, err)
	}
	if instantanea.Validar() != nil {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if instantanea.AsignacionPerfil.PrincipalID != actor.PersonaRef || instantanea.AsignacionPerfil.PerfilActivoRef != actor.PerfilActivoRef || !instantanea.AsignacionPerfil.VigenteEn(ahora) || instantanea.VersionRol.Estado != domain.EstadoVersionRolPublicada || ahora.Before(instantanea.VersionRol.PublicadaEn) || ahora.Before(instantanea.ControlVigenciaVersionRol.ActualizadoEn) || instantanea.ControlVigenciaVersionRol.Estado != domain.EstadoControlVigenciaVersionRolHabilitada {
		return domain.ErrAutorizacionDenegada
	}
	x.Esquema = "administracion_perfiles_lectura_v1"
	x.ActorPersonaRef = actor.PersonaRef
	x.ActorPerfilRef = actor.PerfilActivoRef
	x.Limite = limiteLecturasADMIN
	// La referencia nominal identifica la lectura, no otorga acceso. La capacidad
	// central queda ligada además a todos los filtros del material serializado.
	x.OperacionRef = actor.PerfilActivoRef
	switch {
	case x.PersonaRef != "":
		x.OperacionRef = x.PersonaRef
	case x.PropuestaRef != "":
		x.OperacionRef = x.PropuestaRef
	case x.ReciboRef != "":
		x.OperacionRef = x.ReciboRef
	}
	b, err := json.Marshal(x)
	if err != nil {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	accion := "administracion.perfiles.consultar"
	if x.Consulta == "listar_propuestas" || x.Consulta == "consultar_propuesta" {
		accion = "administracion.perfiles.historial.consultar"
	}
	if x.Consulta == "consultar_recibo" {
		accion = "administracion.perfiles.recibo.consultar"
	}
	efecto := Efecto{Accion: accion, Audiencia: "vec_autorizacion.administracion_perfiles.lectura." + x.Consulta + ".v1", Referencia: x.OperacionRef, Material: b}
	return traducirErrorLectura(ctx, f.autoridad.ejecutar(ctx, actor, evidencia, instantanea, efecto, consulta, validar))
}

func traducirErrorLectura(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, api.ErrRecursoNoEncontrado) {
		return api.ErrRecursoNoEncontrado
	}
	if errors.Is(err, domain.ErrAutorizacionDenegada) {
		return domain.ErrAutorizacionDenegada
	}
	if errors.Is(err, ports.ErrAsignacionPerfilNoEncontrada) || errors.Is(err, ports.ErrVersionRolNoEncontrada) {
		return domain.ErrAutorizacionDenegada
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501":
			return domain.ErrAutorizacionDenegada
		case "P0002":
			return api.ErrRecursoNoEncontrado
		}
	}
	return ports.ErrAutoridadAdministracionPerfilesNoDisponible
}

func (f *FuenteLecturas) Capacidades(ctx context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles) (api.Capacidades, error) {
	var x api.Capacidades
	err := f.ejecutarLectura(ctx, actor, evidencia, lecturaJSON{Consulta: "capacidades"}, capacidadesLecturaSQL, func(b []byte) error {
		if decodificar(b, &x) != nil || x.Version != "1" || x.ActorPersonaRef != actor.PersonaRef || x.Acciones == nil || len(x.Acciones) > limiteLecturasADMIN {
			return ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		vistas := make(map[string]bool, len(x.Acciones))
		for _, a := range x.Acciones {
			if !textoLectura(a, 256, false) || vistas[a] {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			vistas[a] = true
		}
		return nil
	})
	if err != nil {
		return api.Capacidades{}, err
	}
	return x, nil
}

func (f *FuenteLecturas) BuscarPersonas(ctx context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles, busqueda, cursor string) (api.PaginaPersonas, error) {
	var x api.PaginaPersonas
	if len(busqueda) < 2 || !textoLectura(busqueda, 80, false) || !textoLectura(cursor, 256, true) {
		return x, domain.ErrActoAdministracionPerfilesInvalido
	}
	err := f.ejecutarLectura(ctx, actor, evidencia, lecturaJSON{Consulta: "buscar_personas", Busqueda: busqueda, Cursor: cursor}, personasLecturaSQL, func(b []byte) error {
		if decodificar(b, &x) != nil || x.Personas == nil || len(x.Personas) > limiteLecturasADMIN || !textoLectura(x.SiguienteCursor, 256, true) || x.SiguienteCursor != "" && x.SiguienteCursor == cursor {
			return ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		vistas := make(map[string]bool, len(x.Personas))
		for _, p := range x.Personas {
			if !referenciaLectura(p.PersonaRef, "per_") || !textoLectura(p.Nombre, 512, false) || !textoLectura(p.UnidadNombre, 512, true) || !textoLectura(p.UnidadClaveI18N, 256, true) || vistas[p.PersonaRef] {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			vistas[p.PersonaRef] = true
		}
		return nil
	})
	if err != nil {
		return api.PaginaPersonas{}, err
	}
	return x, nil
}

func (f *FuenteLecturas) ConsultarPersona(ctx context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles, persona string) (api.FichaPersona, error) {
	var x api.FichaPersona
	if !referenciaLectura(persona, "per_") {
		return x, domain.ErrActoAdministracionPerfilesInvalido
	}
	err := f.ejecutarLectura(ctx, actor, evidencia, lecturaJSON{Consulta: "consultar_persona", PersonaRef: persona}, personaLecturaSQL, func(b []byte) error {
		if decodificar(b, &x) != nil || !validarFichaLectura(x, persona, f.autoridad.reloj.Ahora()) {
			return ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		return nil
	})
	if err != nil {
		return api.FichaPersona{}, err
	}
	return x, nil
}

func (f *FuenteLecturas) ListarRoles(ctx context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles) (api.Roles, error) {
	var x api.Roles
	err := f.ejecutarLectura(ctx, actor, evidencia, lecturaJSON{Consulta: "listar_roles"}, rolesLecturaSQL, func(b []byte) error {
		if decodificar(b, &x) != nil || x.Roles == nil || len(x.Roles) > limiteLecturasADMIN {
			return ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		vistas := make(map[string]bool, len(x.Roles))
		for _, r := range x.Roles {
			if !domain.RolVersionAdministracionPerfilesValido(r.VersionRef) || !domain.ClaseControlAdministracionPerfiles(r.Clase).Valida() || !textoLectura(r.ClaveI18N, 256, false) || !textoLectura(r.Etiqueta, 512, false) || !huellaLectura(r.HuellaSHA256) || vistas[r.VersionRef] {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			vistas[r.VersionRef] = true
		}
		return nil
	})
	if err != nil {
		return api.Roles{}, err
	}
	return x, nil
}

func (f *FuenteLecturas) ListarPropuestas(ctx context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles) (api.PaginaPropuestas, error) {
	var x api.PaginaPropuestas
	err := f.ejecutarLectura(ctx, actor, evidencia, lecturaJSON{Consulta: "listar_propuestas"}, propuestasLecturaSQL, func(b []byte) error {
		if decodificar(b, &x) != nil || x.Propuestas == nil || len(x.Propuestas) > limiteLecturasADMIN {
			return ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		vistas := make(map[string]bool, len(x.Propuestas))
		for _, p := range x.Propuestas {
			if !validarPropuestaLectura(p, actor, f.autoridad.reloj.Ahora()) || vistas[p.PropuestaRef] {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			vistas[p.PropuestaRef] = true
		}
		return nil
	})
	if err != nil {
		return api.PaginaPropuestas{}, err
	}
	return x, nil
}

func (f *FuenteLecturas) ConsultarPropuesta(ctx context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles, ref string) (api.Propuesta, error) {
	var x api.Propuesta
	if !domain.ReferenciaAdministracionPerfilesValida(ref, "propuesta_admin:") {
		return x, domain.ErrActoAdministracionPerfilesInvalido
	}
	err := f.ejecutarLectura(ctx, actor, evidencia, lecturaJSON{Consulta: "consultar_propuesta", PropuestaRef: ref}, propuestaLecturaSQL, func(b []byte) error {
		if decodificar(b, &x) != nil || x.PropuestaRef != ref || !validarPropuestaLectura(x, actor, f.autoridad.reloj.Ahora()) {
			return ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		return nil
	})
	if err != nil {
		return api.Propuesta{}, err
	}
	return x, nil
}

func (f *FuenteLecturas) ConsultarRecibo(ctx context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles, ref string) (domain.ReciboAdministracionPerfiles, error) {
	var x domain.ReciboAdministracionPerfiles
	if !domain.ReferenciaAdministracionPerfilesValida(ref, "recibo_admin:") {
		return x, domain.ErrActoAdministracionPerfilesInvalido
	}
	err := f.ejecutarLectura(ctx, actor, evidencia, lecturaJSON{Consulta: "consultar_recibo", ReciboRef: ref}, reciboLecturaSQL, func(b []byte) error {
		var dto reciboJSON
		if decodificar(b, &dto) != nil {
			return ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		x = dto.dominio()
		if x.Validar() != nil || x.ReciboRef != ref || !instantePersistible(x.ConfirmadoEn) {
			return ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		return nil
	})
	if err != nil {
		return domain.ReciboAdministracionPerfiles{}, err
	}
	return x, nil
}

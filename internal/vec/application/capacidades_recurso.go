package application

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type EstadoCapacidadRecurso string

const (
	CapacidadRecursoDisponible          EstadoCapacidadRecurso = "disponible"
	CapacidadRecursoNoAutorizado        EstadoCapacidadRecurso = "no_autorizado"
	CapacidadRecursoSinMontaje          EstadoCapacidadRecurso = "sin_montaje"
	CapacidadRecursoIndisponible        EstadoCapacidadRecurso = "indisponible"
	maximoRecursosCapacidadRecurso                             = 128
	maximoRutasMontadasCapacidadRecurso                        = 2048
)

type ResultadoCapacidadRecurso struct {
	Indice           int                    `json:"indice"`
	Estado           EstadoCapacidadRecurso `json:"estado"`
	CamposPermitidos []string               `json:"campos_permitidos"`
	Obligaciones     []string               `json:"obligaciones"`
}

type RevisionesCapacidadRecurso struct {
	Asignacion        int    `json:"asignacion"`
	ControlRol        uint64 `json:"control_rol"`
	CatalogoPoliticas uint64 `json:"catalogo_politicas"`
}

type ProyeccionCapacidadesRecurso struct {
	PerfilActivoRef string                      `json:"perfil_activo_ref"`
	Superficie      string                      `json:"superficie"`
	Resultados      []ResultadoCapacidadRecurso `json:"resultados"`
	Revisiones      RevisionesCapacidadRecurso  `json:"revisiones"`
	VigenteHasta    time.Time                   `json:"vigente_hasta"`
}

type ProyectorCapacidadesRecurso struct {
	fuente  ports.FuenteAutorizacion
	montaje ports.RegistroMontajeCapacidadesRecurso
	reloj   ports.Reloj
}

func NuevoProyectorCapacidadesRecurso(
	fuente ports.FuenteAutorizacion,
	montaje ports.RegistroMontajeCapacidadesRecurso,
	reloj ports.Reloj,
) (*ProyectorCapacidadesRecurso, error) {
	if dependenciaAutorizacionNula(fuente) || dependenciaAutorizacionNula(montaje) || dependenciaAutorizacionNula(reloj) {
		return nil, domain.ErrConfiguracionAccesoInvalida
	}
	return &ProyectorCapacidadesRecurso{fuente: fuente, montaje: montaje, reloj: reloj}, nil
}

// Proyectar consume recursos ya leídos y auditados por el módulo propietario.
// El resultado es información para navegación; ningún campo es una decisión
// consumible y la lectura posterior vuelve a autorizarse en su propia ruta.
func (p *ProyectorCapacidadesRecurso) Proyectar(
	ctx context.Context, lote ports.LoteRecursosLeidosCapacidad,
) (ProyeccionCapacidadesRecurso, error) {
	if p == nil || ctx == nil || dependenciaAutorizacionNula(p.fuente) ||
		dependenciaAutorizacionNula(p.montaje) || dependenciaAutorizacionNula(p.reloj) ||
		len(lote.Recursos) > maximoRecursosCapacidadRecurso {
		return ProyeccionCapacidadesRecurso{}, domain.ErrConfiguracionAccesoInvalida
	}
	if err := ctx.Err(); err != nil {
		return ProyeccionCapacidadesRecurso{}, err
	}
	resultado := ProyeccionCapacidadesRecurso{Resultados: make([]ResultadoCapacidadRecurso, len(lote.Recursos)), Superficie: lote.Superficie}
	for i := range resultado.Resultados {
		resultado.Resultados[i] = ResultadoCapacidadRecurso{Indice: i, Estado: CapacidadRecursoIndisponible,
			CamposPermitidos: []string{}, Obligaciones: []string{}}
	}
	if !identificadorCapacidadRecurso(lote.Superficie) || lote.Resultado.Validar() != nil ||
		lote.Vinculo.ValidarPara(lote.Resultado) != nil {
		return resultado, nil
	}
	datos, err := lote.Vinculo.Datos()
	if err != nil {
		return resultado, nil
	}
	resultado.PerfilActivoRef = datos.PerfilActivoRef
	ahora := p.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !lote.Vinculo.VigenteEn(ahora, lote.Resultado) {
		cambiarEstadosCapacidadRecurso(resultado.Resultados, CapacidadRecursoIndisponible, CapacidadRecursoNoAutorizado)
		return resultado, nil
	}
	vista, err := p.montaje.VistaMontada(ctx, lote.Superficie, datos.PerfilActivoRef)
	if cancelada := ctx.Err(); cancelada != nil {
		return ProyeccionCapacidadesRecurso{}, cancelada
	}
	if err != nil || !vistaMontajeCapacidadRecursoValida(vista, lote.Superficie, datos.PerfilActivoRef) {
		return resultado, nil
	}
	montadas := make(map[string]struct{}, len(vista.Rutas))
	for _, ruta := range vista.Rutas {
		montadas[claveMontajeCapacidadRecurso(ruta.RutaID, ruta.Metodo, ruta.ModuloID, ruta.TipoRecurso, ruta.Accion, ruta.Finalidad)] = struct{}{}
	}
	consultas := make([]domain.ConsultaCapacidadInformativaV3, 0, len(lote.Recursos))
	indices := make([]int, 0, len(lote.Recursos))
	for i, recurso := range lote.Recursos {
		if !recursoLeidoCapacidadValido(recurso) {
			continue
		}
		clave := claveMontajeCapacidadRecurso(recurso.RutaID, recurso.Metodo,
			recurso.Recurso.ModuloID, recurso.Recurso.Tipo, recurso.Accion, recurso.Finalidad)
		if _, existe := montadas[clave]; !existe {
			resultado.Resultados[i].Estado = CapacidadRecursoSinMontaje
			continue
		}
		consultas = append(consultas, domain.ConsultaCapacidadInformativaV3{
			Accion: recurso.Accion, Finalidad: recurso.Finalidad, Recurso: clonarRecursoCapacidad(recurso.Recurso),
		})
		indices = append(indices, i)
	}
	if len(consultas) == 0 {
		if err := ctx.Err(); err != nil {
			return ProyeccionCapacidadesRecurso{}, err
		}
		return resultado, nil
	}
	instantanea, err := p.fuente.ObtenerInstantaneaAutorizacion(ctx, datos.PrincipalID, datos.PerfilActivoRef)
	if cancelada := ctx.Err(); cancelada != nil {
		return ProyeccionCapacidadesRecurso{}, cancelada
	}
	if err != nil {
		if errors.Is(err, ports.ErrAsignacionPerfilNoEncontrada) && !errors.Is(err, ports.ErrFuenteAutorizacionNoDisponible) {
			cambiarEstadosIndiceCapacidadRecurso(resultado.Resultados, indices, CapacidadRecursoNoAutorizado)
		}
		return resultado, nil
	}
	ahora = p.reloj.Ahora().UTC().Truncate(time.Microsecond)
	evaluadas, limite, err := domain.EvaluarCapacidadesInformativasV3(lote.Vinculo, lote.Resultado, instantanea, consultas, ahora)
	if err != nil {
		if errors.Is(err, domain.ErrAutorizacionDenegada) {
			cambiarEstadosIndiceCapacidadRecurso(resultado.Resultados, indices, CapacidadRecursoNoAutorizado)
		}
		return resultado, nil
	}
	if err := ctx.Err(); err != nil {
		return ProyeccionCapacidadesRecurso{}, err
	}
	if !limite.After(p.reloj.Ahora().UTC()) {
		cambiarEstadosIndiceCapacidadRecurso(resultado.Resultados, indices, CapacidadRecursoNoAutorizado)
		return resultado, nil
	}
	for j, evaluada := range evaluadas {
		i := indices[j]
		resultado.Resultados[i].Estado = CapacidadRecursoNoAutorizado
		if evaluada.Concedida {
			resultado.Resultados[i].Estado = CapacidadRecursoDisponible
			resultado.Resultados[i].CamposPermitidos = append([]string{}, evaluada.CamposPermitidos...)
			resultado.Resultados[i].Obligaciones = append([]string{}, evaluada.Obligaciones...)
		}
	}
	resultado.Revisiones = RevisionesCapacidadRecurso{
		Asignacion:        instantanea.AsignacionPerfil.Version,
		ControlRol:        instantanea.ControlVigenciaVersionRol.Revision,
		CatalogoPoliticas: instantanea.RevisionCatalogoPoliticas,
	}
	resultado.VigenteHasta = limite
	return resultado, nil
}

func vistaMontajeCapacidadRecursoValida(v ports.VistaMontajeCapacidadesRecurso, superficie, perfil string) bool {
	if !v.Completa || v.Superficie != superficie || v.PerfilActivoRef != perfil ||
		len(v.Rutas) > maximoRutasMontadasCapacidadRecurso {
		return false
	}
	claves := make(map[string]struct{}, len(v.Rutas))
	for _, ruta := range v.Rutas {
		if !identificadorCapacidadRecurso(ruta.RutaID) || !metodoCapacidadRecursoValido(ruta.Metodo) ||
			!identificadorCapacidadRecurso(ruta.ModuloID) || !identificadorCapacidadRecurso(ruta.TipoRecurso) ||
			!identificadorCapacidadRecurso(ruta.Accion) || !identificadorCapacidadRecurso(ruta.Finalidad) {
			return false
		}
		clave := claveMontajeCapacidadRecurso(ruta.RutaID, ruta.Metodo, ruta.ModuloID, ruta.TipoRecurso, ruta.Accion, ruta.Finalidad)
		if _, existe := claves[clave]; existe {
			return false
		}
		claves[clave] = struct{}{}
	}
	return true
}

func recursoLeidoCapacidadValido(r ports.RecursoLeidoCapacidad) bool {
	return identificadorCapacidadRecurso(r.RutaID) && metodoCapacidadRecursoValido(r.Metodo) &&
		identificadorCapacidadRecurso(r.Accion) && identificadorCapacidadRecurso(r.Finalidad) &&
		r.Recurso.Validar() == nil
}

func identificadorCapacidadRecurso(v string) bool {
	return v != "" && len(v) <= 512 && v == strings.TrimSpace(v) && !strings.ContainsAny(v, "*\x00\r\n\t")
}

func metodoCapacidadRecursoValido(v string) bool {
	return v == http.MethodGet || v == http.MethodPost || v == http.MethodPut || v == http.MethodDelete || v == http.MethodPatch
}

func claveMontajeCapacidadRecurso(ruta, metodo, modulo, tipo, accion, finalidad string) string {
	return strings.Join([]string{ruta, metodo, modulo, tipo, accion, finalidad}, "\x00")
}

func cambiarEstadosCapacidadRecurso(resultados []ResultadoCapacidadRecurso, anterior, nuevo EstadoCapacidadRecurso) {
	for i := range resultados {
		if resultados[i].Estado == anterior {
			resultados[i].Estado = nuevo
		}
	}
}

func cambiarEstadosIndiceCapacidadRecurso(resultados []ResultadoCapacidadRecurso, indices []int, estado EstadoCapacidadRecurso) {
	for _, i := range indices {
		resultados[i].Estado = estado
	}
}

func clonarRecursoCapacidad(r domain.RecursoAutorizable) domain.RecursoAutorizable {
	copia := domain.RecursoAutorizable{Referencia: r.Referencia, ModuloID: r.ModuloID, Tipo: r.Tipo,
		Ambitos: make(map[string]string, len(r.Ambitos)), Atributos: make(map[string]string, len(r.Atributos))}
	for k, v := range r.Ambitos {
		copia.Ambitos[k] = v
	}
	for k, v := range r.Atributos {
		copia.Atributos[k] = v
	}
	return copia
}

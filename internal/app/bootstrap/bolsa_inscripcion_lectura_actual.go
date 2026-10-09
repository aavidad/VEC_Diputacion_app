package bootstrap

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
	postgresvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// DescriptorLecturaActualInscripcion fija el recurso y campos que el servidor
// somete a la concesión publicada. AmbitoPersonaClave liga expresamente una
// dimensión a la persona autenticada; AmbitosFijos nunca proceden de HTTP.
type DescriptorLecturaActualInscripcion struct {
	Accion, ModuloID, TipoRecurso, Finalidad string
	Campos                                   []string
	AmbitoPersonaClave                       string
	AmbitosFijos                             map[string]string
}

type ConfiguracionDecisorLecturaActualInscripcion struct {
	Reloj        interface{ Ahora() time.Time }
	Descriptores map[string]DescriptorLecturaActualInscripcion
}

type decisorLecturaActualInscripcion struct {
	externa, interna vecports.FuenteAutorizacion
	reloj            interface{ Ahora() time.Time }
	descriptores     map[string]DescriptorLecturaActualInscripcion
}

var _ DecisorLecturaActualInscripcionBolsa = (*decisorLecturaActualInscripcion)(nil)

// Cada pool debe usar el login de lectura nominal de su superficie. La
// construcción usa las fachadas PG distintas y nunca crea roles ni grants.
func NuevoDecisorLecturaActualInscripcionPostgreSQL(poolExterno, poolInterno *pgxpool.Pool, c ConfiguracionDecisorLecturaActualInscripcion) (DecisorLecturaActualInscripcionBolsa, error) {
	if poolExterno == nil || poolInterno == nil || poolExterno == poolInterno {
		return nil, inscripcion.ErrNoDisponible
	}
	externa, err := postgresvec.NuevoAlmacenAutorizacionExterna(poolExterno)
	if err != nil {
		return nil, inscripcion.ErrNoDisponible
	}
	interna, err := postgresvec.NuevoAlmacenAutorizacion(poolInterno)
	if err != nil {
		return nil, inscripcion.ErrNoDisponible
	}
	return nuevoDecisorLecturaActualInscripcionFuentes(externa, interna, c)
}

// Este seam privado permite probar denegaciones sin abrir conexiones. La
// composición real sólo expone el constructor PostgreSQL anterior.
func nuevoDecisorLecturaActualInscripcionFuentes(externa, interna vecports.FuenteAutorizacion, c ConfiguracionDecisorLecturaActualInscripcion) (*decisorLecturaActualInscripcion, error) {
	if nuloInscripcionBolsa(externa) || nuloInscripcionBolsa(interna) || nuloInscripcionBolsa(c.Reloj) || len(c.Descriptores) != 7 {
		return nil, inscripcion.ErrNoDisponible
	}
	clon := make(map[string]DescriptorLecturaActualInscripcion, 7)
	for accion, d := range c.Descriptores {
		if !accionLecturaInscripcion(accion) || d.Accion != accion || d.ModuloID == "" || d.TipoRecurso == "" ||
			d.Finalidad == "" || len(d.Campos) == 0 || d.AmbitoPersonaClave == "*" ||
			((accion == inscripcion.AccionListarPropias || accion == inscripcion.AccionDetallePropia) && d.AmbitoPersonaClave == "") {
			return nil, inscripcion.ErrNoDisponible
		}
		ambitos := maps.Clone(d.AmbitosFijos)
		if d.AmbitoPersonaClave != "" {
			if _, repetida := ambitos[d.AmbitoPersonaClave]; repetida {
				return nil, inscripcion.ErrNoDisponible
			}
			if ambitos == nil {
				ambitos = make(map[string]string, 1)
			}
			ambitos[d.AmbitoPersonaClave] = "persona_prueba"
		}
		if (vecdomain.RecursoAutorizable{Referencia: "recurso_prueba", ModuloID: d.ModuloID, Tipo: d.TipoRecurso, Ambitos: ambitos}).Validar() != nil {
			return nil, inscripcion.ErrNoDisponible
		}
		campos := slices.Clone(d.Campos)
		ordenados := slices.Clone(campos)
		slices.Sort(ordenados)
		if slices.Contains(ordenados, "*") {
			return nil, inscripcion.ErrNoDisponible
		}
		for i := range ordenados {
			if ordenados[i] == "" || i > 0 && ordenados[i] == ordenados[i-1] {
				return nil, inscripcion.ErrNoDisponible
			}
		}
		d.Campos, d.AmbitosFijos = campos, maps.Clone(d.AmbitosFijos)
		clon[accion] = d
	}
	return &decisorLecturaActualInscripcion{externa: externa, interna: interna, reloj: c.Reloj, descriptores: clon}, nil
}

func acreditacionInscripcionBolsaActual(s contextoSeguridadComunDesarrollo, a AcreditacionSesionInscripcionBolsa, ahora time.Time) bool {
	if s.Resultado.Validar() != nil || s.Vinculo.ValidarPara(s.Resultado) != nil || !s.Vinculo.VigenteEn(ahora, s.Resultado) ||
		!huellaCertificadoInscripcionValida(a.CertificadoHuellaSHA256) || a.VerificadaEn.IsZero() ||
		a.VerificadaEn.After(ahora) || !a.ValidaHasta.After(ahora) {
		return false
	}
	v, err := s.Vinculo.Datos()
	return err == nil && a.Canal == string(v.Superficie) &&
		a.PersonaRef == s.Resultado.Contexto.PersonaRef && a.PersonaRef == v.PrincipalID &&
		a.PerfilRef == s.Resultado.Contexto.PerfilActivoRef && a.PerfilRef == v.PerfilActivoRef &&
		a.CuentaRef == s.Resultado.Contexto.Instantanea.CuentaRef && a.CuentaRef == v.CuentaRef &&
		a.SesionRef == v.SesionRef && a.AutenticacionRef == v.AutenticacionRef &&
		!a.ValidaHasta.After(v.SesionValidaHasta)
}

func (d *decisorLecturaActualInscripcion) DecidirLecturaActual(ctx context.Context, s contextoSeguridadComunDesarrollo, a AcreditacionSesionInscripcionBolsa, accion, referencia string, filtro inscripcion.Filtro) (DecisionLecturaActualInscripcionBolsa, error) {
	vacia := DecisionLecturaActualInscripcionBolsa{}
	if d == nil || ctx == nil || ctx.Err() != nil || nuloInscripcionBolsa(d.reloj) ||
		!accionLecturaInscripcion(accion) || referencia == "" {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	descriptor, ok := d.descriptores[accion]
	if !ok || descriptor.Accion != accion || descriptor.Finalidad == "" || len(descriptor.Campos) == 0 {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	ahora := d.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !acreditacionInscripcionBolsaActual(s, a, ahora) {
		return vacia, inscripcion.ErrSesionAusente
	}
	v, err := s.Vinculo.Datos()
	if err != nil || !canalLecturaInscripcion(accion, a.Canal, v.Superficie) {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	var fuente vecports.FuenteAutorizacion
	switch v.Superficie {
	case vecdomain.SuperficieAutenticacionExternaPersonalV1:
		fuente = d.externa
	case vecdomain.SuperficieAutenticacionInternaCorporativaV1:
		fuente = d.interna
	default:
		return vacia, inscripcion.ErrAccesoDenegado
	}
	if nuloInscripcionBolsa(fuente) {
		return vacia, inscripcion.ErrNoDisponible
	}
	ambitos := maps.Clone(descriptor.AmbitosFijos)
	if descriptor.AmbitoPersonaClave != "" {
		if ambitos == nil {
			ambitos = make(map[string]string, 1)
		}
		ambitos[descriptor.AmbitoPersonaClave] = v.PrincipalID
	}
	recurso := vecdomain.RecursoAutorizable{Referencia: referencia, ModuloID: descriptor.ModuloID, Tipo: descriptor.TipoRecurso, Ambitos: ambitos}
	if recurso.Validar() != nil {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	// La fuente devuelve una instantánea completa y actual. Su implementación
	// PostgreSQL consulta exclusivamente la fachada propia de esta superficie.
	instantanea, err := fuente.ObtenerInstantaneaAutorizacion(ctx, v.PrincipalID, v.PerfilActivoRef)
	if err != nil || ctx.Err() != nil || instantanea.Validar() != nil ||
		instantanea.AsignacionPerfil.EmitidaPor == "identidad:desarrollo:no-autoritativa" ||
		instantanea.VersionRol.PublicadaPor == "seguridad:desarrollo:no-autoritativa" ||
		instantanea.ControlVigenciaVersionRol.ActualizadoPor == "seguridad:desarrollo:no-autoritativa" {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	// Un grant con campos o obligaciones ajenos al contrato de lectura no se
	// reduce silenciosamente. Las restricciones ABAC sólo pueden denegar.
	if !concesionLecturaInscripcionExacta(instantanea, descriptor) {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	ahora = d.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !acreditacionInscripcionBolsaActual(s, a, ahora) {
		return vacia, inscripcion.ErrSesionAusente
	}
	evaluacion, err := vecdomain.EvaluarLecturaActualAutorizacionV3(s.Vinculo, s.Resultado, instantanea,
		accion, recurso, descriptor.Finalidad, descriptor.Campos, ahora)
	if err != nil || !evaluacion.Permitido || len(evaluacion.Obligaciones) != 0 {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	camposEvaluados := slices.Clone(evaluacion.CamposPermitidos)
	camposEsperados := slices.Clone(descriptor.Campos)
	slices.Sort(camposEsperados)
	if !slices.Equal(camposEvaluados, camposEsperados) {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacia, inscripcion.ErrNoDisponible
	}
	correlacionRef, err := correlacion.ValorCanonico()
	if err != nil {
		return vacia, inscripcion.ErrNoDisponible
	}
	ahoraFinal := d.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if ctx.Err() != nil || !acreditacionInscripcionBolsaActual(s, a, ahoraFinal) || !evaluacion.ValidaHasta.After(ahoraFinal) {
		return vacia, inscripcion.ErrSesionAusente
	}
	hasta := evaluacion.ValidaHasta
	if limite := ahoraFinal.Add(30 * time.Second); hasta.After(limite) {
		hasta = limite
	}
	if hasta.After(a.ValidaHasta) {
		hasta = a.ValidaHasta
	}
	if !hasta.After(ahoraFinal) {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	if !revisionHuellaLecturaInscripcionCoincide(evaluacion.RevisionPermisos, evaluacion.HuellaInstantaneaSHA256) {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	return DecisionLecturaActualInscripcionBolsa{
		Concedida: true, PersonaRef: a.PersonaRef, PerfilRef: a.PerfilRef, CuentaRef: a.CuentaRef,
		SesionRef: a.SesionRef, AutenticacionRef: a.AutenticacionRef,
		CertificadoHuellaSHA256: a.CertificadoHuellaSHA256, Canal: a.Canal,
		Accion: accion, RecursoRef: referencia, Finalidad: descriptor.Finalidad, CorrelacionRef: correlacionRef,
		RevisionPermisos: evaluacion.RevisionPermisos, HuellaInstantaneaSHA256: evaluacion.HuellaInstantaneaSHA256,
		Campos: slices.Clone(descriptor.Campos), Filtro: filtro,
		EmitidaEn: evaluacion.EvaluadaEn, ValidaHasta: hasta,
	}, nil
}

func concesionLecturaInscripcionExacta(i vecdomain.InstantaneaAutorizacion, d DescriptorLecturaActualInscripcion) bool {
	coincidencias := 0
	esperados := slices.Clone(d.Campos)
	slices.Sort(esperados)
	for _, c := range i.VersionRol.Concesiones {
		if c.Accion != d.Accion {
			continue
		}
		if c.ModuloID != d.ModuloID || c.TipoRecurso != d.TipoRecurso || len(c.Finalidades) != 1 ||
			c.Finalidades[0] != d.Finalidad || len(c.Obligaciones) != 0 {
			return false
		}
		campos := slices.Clone(c.CamposPermitidos)
		slices.Sort(campos)
		if !slices.Equal(campos, esperados) {
			return false
		}
		coincidencias++
	}
	return coincidencias == 1
}

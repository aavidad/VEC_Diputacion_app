package bootstrap

import (
	"context"
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
	AmbitoVinculoClave                       string
	AmbitoVinculoTipo                        vecdomain.TipoReferenciaContextoActor
}

// Cada literal identifica una hoja JSON del contrato de respuesta; [] nombra
// un elemento de array y no actúa como comodín sobre otros campos. Cambiar la
// proyección obliga a publicar una nueva concesión y revisar esta tabla.
func camposLecturaInscripcionBolsa(accion string, rrhh bool) []string {
	requisitos := []string{"codigo", "descripcion", "obligatorio", "estado", "motivo_codigo", "motivo_etiqueta",
		"procedencia_ref", "hito_cumplimiento", "hito_etiqueta", "hito_fecha"}
	prefijar := func(prefijo string, hojas []string) []string {
		resultado := make([]string, 0, len(hojas))
		for _, hoja := range hojas {
			resultado = append(resultado, prefijo+hoja)
		}
		return resultado
	}
	bolsa := []string{"convocatoria_ref", "titulo", "categorias_resumen", "categorias[].categoria_ref",
		"categorias[].categoria", "plazo_inicio", "plazo_fin", "catalogo_version", "requisitos_resumen",
		"puede_iniciar", "impedimento_etiqueta", "estado_solicitud_propia", "solicitud_ref"}
	bolsa = append(bolsa, prefijar("requisitos[].", requisitos)...)
	solicitud := []string{"solicitud_ref", "recibo_ref", "convocatoria_ref", "categoria_ref", "bolsa_ref",
		"categoria", "declaracion_ref", "bases_ref", "catalogo_version", "plazo_inicio", "plazo_fin",
		"decision_ref", "estado", "version", "registrada_en", "decidida_en", "motivo_codigo",
		"motivo_etiqueta", "participacion_ref"}
	solicitud = append(solicitud, prefijar("requisitos[].", requisitos)...)
	if rrhh {
		solicitud = append(solicitud, "persona_resumen")
	}
	var campos []string
	switch accion {
	case inscripcion.AccionListarAbiertas:
		campos = append([]string{"total", "cursor_siguiente"}, prefijar("convocatorias[].", bolsa)...)
	case inscripcion.AccionDetalleAbierta:
		campos = bolsa
	case inscripcion.AccionListarPropias, inscripcion.AccionListarRRHH:
		campos = append([]string{"total", "cursor_siguiente"}, prefijar("solicitudes[].", solicitud)...)
	case inscripcion.AccionDetallePropia, inscripcion.AccionDetalleRRHH:
		campos = solicitud
	case inscripcion.AccionMotivosRRHH:
		campos = []string{"catalogo_version", "motivos[].codigo", "motivos[].etiqueta", "motivos[].obligatorio"}
	case accionListarConvocatoriasGestionRRHHInscripcion:
		campos = []string{"convocatorias[].convocatoria_ref", "convocatorias[].titulo", "convocatorias[].categorias_resumen",
			"convocatorias[].plazo_fin", "convocatorias[].estado_publicacion", "total", "cursor_siguiente"}
	default:
		return nil
	}
	slices.Sort(campos)
	return campos
}

// El dueño Bolsa obtiene sólo metadatos opacos de un recurso publicado o
// histórico antes de evaluar el GET RRHH. La lectura auditada posterior debe
// cotejar la misma fuente/version/huella dentro de su TX.
type FuenteAmbitoLecturaRRHHInscripcionBolsa interface {
	ResolverAmbitoLecturaRRHH(context.Context, contextoSeguridadComunDesarrollo, AcreditacionSesionInscripcionBolsa, string, string, inscripcion.Filtro) (AmbitoLecturaRRHHInscripcionBolsa, error)
}

type AmbitoLecturaRRHHInscripcionBolsa struct {
	RecursoRef, UnidadRef, AmbitoRef, FuenteRef string
	FuenteVersion                               uint64
	FuenteHuellaSHA256                          string
	ConjuntoRef, ConvocatoriaRef                string
}

type ConfiguracionDecisorLecturaActualInscripcion struct {
	Reloj        interface{ Ahora() time.Time }
	Descriptores map[ClaveOperacionInscripcionBolsa]DescriptorLecturaActualInscripcion
	AmbitoRRHH   FuenteAmbitoLecturaRRHHInscripcionBolsa
	RRHHNominal  []identidadConsultaRRHHDesarrollo
}

type decisorLecturaActualInscripcion struct {
	externa, interna vecports.FuenteAutorizacion
	reloj            interface{ Ahora() time.Time }
	descriptores     map[ClaveOperacionInscripcionBolsa]DescriptorLecturaActualInscripcion
	ambitoRRHH       FuenteAmbitoLecturaRRHHInscripcionBolsa
	rrhhNominal      []identidadRRHHInscripcionBolsa
}

var _ DecisorLecturaActualInscripcionBolsa = (*decisorLecturaActualInscripcion)(nil)
var _ FuenteActualEscrituraInscripcionBolsa = (*decisorLecturaActualInscripcion)(nil)

// Cada pool debe usar el login de lectura nominal de su superficie. La
// construcción usa las fachadas PG distintas y nunca crea roles ni grants.
func NuevoDecisorLecturaActualInscripcionPostgreSQL(poolExterno, poolInterno *pgxpool.Pool, c ConfiguracionDecisorLecturaActualInscripcion) (*decisorLecturaActualInscripcion, error) {
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
	if nuloInscripcionBolsa(externa) || nuloInscripcionBolsa(interna) || nuloInscripcionBolsa(c.Reloj) ||
		len(c.Descriptores) != 12 || !identidadesRRHHInscripcionValidas(c.RRHHNominal) {
		return nil, inscripcion.ErrNoDisponible
	}
	clon := make(map[ClaveOperacionInscripcionBolsa]DescriptorLecturaActualInscripcion, 12)
	for _, clave := range clavesLecturaInscripcionBolsa() {
		d, ok := c.Descriptores[clave]
		tipo, finalidad, esperado := tipoFinalidadInscripcionBolsa(clave)
		if !ok || !esperado || d.Accion != clave.Accion || d.ModuloID != "bolsa" ||
			d.TipoRecurso != tipo || d.Finalidad != finalidad || len(d.Campos) == 0 {
			return nil, inscripcion.ErrNoDisponible
		}
		if clave.Canal == "externa_personal" && (d.AmbitoVinculoClave != "candidato_ref" || d.AmbitoVinculoTipo != vecdomain.TipoReferenciaContextoActorCandidato) ||
			clave.Canal == "interna_corporativa" && !accionRRHHInscripcion(clave.Accion) &&
				(d.AmbitoVinculoClave != "empleado_ref" || d.AmbitoVinculoTipo != vecdomain.TipoReferenciaContextoActorEmpleado) ||
			accionRRHHInscripcion(clave.Accion) && (d.AmbitoVinculoClave != "" || d.AmbitoVinculoTipo != "") {
			return nil, inscripcion.ErrNoDisponible
		}
		campos := slices.Clone(d.Campos)
		ordenados := slices.Clone(campos)
		slices.Sort(ordenados)
		if !slices.Equal(ordenados, camposLecturaInscripcionBolsa(clave.Accion, accionRRHHInscripcion(clave.Accion))) {
			return nil, inscripcion.ErrNoDisponible
		}
		if slices.Contains(ordenados, "*") {
			return nil, inscripcion.ErrNoDisponible
		}
		for i := range ordenados {
			if ordenados[i] == "" || i > 0 && ordenados[i] == ordenados[i-1] {
				return nil, inscripcion.ErrNoDisponible
			}
		}
		d.Campos = campos
		clon[clave] = d
	}
	return &decisorLecturaActualInscripcion{externa: externa, interna: interna, reloj: c.Reloj, descriptores: clon,
		ambitoRRHH: c.AmbitoRRHH, rrhhNominal: copiarIdentidadesRRHHInscripcion(c.RRHHNominal)}, nil
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
	descriptor, ok := d.descriptores[ClaveOperacionInscripcionBolsa{accion, a.Canal}]
	if !ok || descriptor.Accion != accion || descriptor.Finalidad == "" || len(descriptor.Campos) == 0 {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	if accion == inscripcion.AccionListarRRHH && filtro.ConvocatoriaRef == "" {
		return vacia, inscripcion.ErrSolicitudInvalida
	}
	if accion == accionListarConvocatoriasGestionRRHHInscripcion &&
		(filtro.Validar() != nil || filtro.Estado != "" || filtro.ConvocatoriaRef != "") {
		return vacia, inscripcion.ErrSolicitudInvalida
	}
	ahora := d.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !acreditacionInscripcionBolsaActual(s, a, ahora) {
		return vacia, inscripcion.ErrSesionAusente
	}
	v, err := s.Vinculo.Datos()
	if err != nil || !canalLecturaInscripcion(accion, a.Canal, v.Superficie) {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	if v.Superficie == vecdomain.SuperficieAutenticacionInternaCorporativaV1 && !accionRRHHInscripcion(accion) &&
		rrhhNominalInscripcionEnLista(s, a, d.rrhhNominal) {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	fuente := d.fuenteParaSuperficie(v.Superficie)
	if nuloInscripcionBolsa(fuente) {
		return vacia, inscripcion.ErrNoDisponible
	}
	ambitos := map[string]string{}
	atributos := map[string]string{}
	if accionRRHHInscripcion(accion) {
		if !rrhhNominalInscripcionEnLista(s, a, d.rrhhNominal) || nuloInscripcionBolsa(d.ambitoRRHH) {
			return vacia, inscripcion.ErrAccesoDenegado
		}
		resuelto, err := d.ambitoRRHH.ResolverAmbitoLecturaRRHH(ctx, s, a, accion, referencia, filtro)
		if err != nil || resuelto.RecursoRef != referencia || resuelto.UnidadRef == "" || resuelto.AmbitoRef == "" ||
			resuelto.FuenteRef == "" || resuelto.FuenteVersion == 0 || !huellaCertificadoInscripcionValida(resuelto.FuenteHuellaSHA256) {
			return vacia, inscripcion.ErrAccesoDenegado
		}
		ambitos["unidad_ref"], ambitos["ambito_ref"] = resuelto.UnidadRef, resuelto.AmbitoRef
		if accion == accionListarConvocatoriasGestionRRHHInscripcion {
			if resuelto.ConjuntoRef == "" {
				return vacia, inscripcion.ErrAccesoDenegado
			}
			atributos["conjunto_ref"] = resuelto.ConjuntoRef
		}
		if accion == inscripcion.AccionListarRRHH {
			if resuelto.ConvocatoriaRef != filtro.ConvocatoriaRef {
				return vacia, inscripcion.ErrAccesoDenegado
			}
			atributos["convocatoria_ref"] = resuelto.ConvocatoriaRef
		}
	} else {
		ref, ok := vinculoUnicoInscripcion(s, descriptor.AmbitoVinculoTipo, ahora)
		if !ok {
			return vacia, inscripcion.ErrAccesoDenegado
		}
		ambitos[descriptor.AmbitoVinculoClave] = ref
	}
	recurso := vecdomain.RecursoAutorizable{Referencia: referencia, ModuloID: descriptor.ModuloID, Tipo: descriptor.TipoRecurso, Ambitos: ambitos, Atributos: atributos}
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

func (d *decisorLecturaActualInscripcion) fuenteParaSuperficie(superficie vecdomain.SuperficieAutenticacionActorV1) vecports.FuenteAutorizacion {
	if d == nil {
		return nil
	}
	switch superficie {
	case vecdomain.SuperficieAutenticacionExternaPersonalV1:
		return d.externa
	case vecdomain.SuperficieAutenticacionInternaCorporativaV1:
		return d.interna
	default:
		return nil
	}
}

func (d *decisorLecturaActualInscripcion) ObtenerInstantaneaEscrituraActual(ctx context.Context, s contextoSeguridadComunDesarrollo, a AcreditacionSesionInscripcionBolsa) (vecdomain.InstantaneaAutorizacion, error) {
	vacia := vecdomain.InstantaneaAutorizacion{}
	if d == nil || ctx == nil || ctx.Err() != nil || nuloInscripcionBolsa(d.reloj) {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	ahora := d.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !acreditacionInscripcionBolsaActual(s, a, ahora) {
		return vacia, inscripcion.ErrSesionAusente
	}
	v, err := s.Vinculo.Datos()
	if err != nil {
		return vacia, inscripcion.ErrSesionAusente
	}
	fuente := d.fuenteParaSuperficie(v.Superficie)
	if nuloInscripcionBolsa(fuente) {
		return vacia, inscripcion.ErrNoDisponible
	}
	i, err := fuente.ObtenerInstantaneaAutorizacion(ctx, v.PrincipalID, v.PerfilActivoRef)
	ahora = d.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if err != nil || ctx.Err() != nil || !acreditacionInscripcionBolsaActual(s, a, ahora) || i.Validar() != nil ||
		i.AsignacionPerfil.PrincipalID != v.PrincipalID || i.AsignacionPerfil.PerfilActivoRef != v.PerfilActivoRef ||
		!i.AsignacionPerfil.VigenteEn(ahora) || i.VersionRol.Estado != vecdomain.EstadoVersionRolPublicada ||
		i.ControlVigenciaVersionRol.Estado != vecdomain.EstadoControlVigenciaVersionRolHabilitada ||
		i.AsignacionPerfil.EmitidaPor == "identidad:desarrollo:no-autoritativa" ||
		i.VersionRol.PublicadaPor == "seguridad:desarrollo:no-autoritativa" ||
		i.ControlVigenciaVersionRol.ActualizadoPor == "seguridad:desarrollo:no-autoritativa" {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	return i, nil
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

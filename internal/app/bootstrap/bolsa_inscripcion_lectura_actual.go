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
	bolsa := []string{"convocatoria_ref", "titulo", "numero_categorias", "categorias[].categoria_ref",
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
	case inscripcion.AccionListarPropias:
		campos = append([]string{"total", "cursor_siguiente"}, prefijar("solicitudes[].", solicitud)...)
	case inscripcion.AccionListarRRHH:
		campos = append([]string{"total", "cursor_siguiente", "convocatoria_titulo"}, prefijar("solicitudes[].", solicitud)...)
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
// descriptoresLecturaActualInscripcionBolsa deriva del contrato compilado los
// descriptores de lectura de una superficie: tipo, finalidad, campos y vínculo
// que acota el ámbito. El permiso sigue saliendo de la concesión publicada.
func descriptoresLecturaActualInscripcionBolsa(superficie string) map[ClaveOperacionInscripcionBolsa]DescriptorLecturaActualInscripcion {
	claves := clavesLecturaInscripcionBolsa(superficie)
	resultado := make(map[ClaveOperacionInscripcionBolsa]DescriptorLecturaActualInscripcion, len(claves))
	for _, clave := range claves {
		tipo, finalidad, ok := tipoFinalidadInscripcionBolsa(clave)
		if !ok {
			return nil
		}
		d := DescriptorLecturaActualInscripcion{Accion: clave.Accion, ModuloID: "bolsa", TipoRecurso: tipo, Finalidad: finalidad,
			Campos: camposLecturaInscripcionBolsa(clave.Accion, accionRRHHInscripcion(clave.Accion))}
		if clave.Canal == superficieExternaInscripcionBolsa {
			d.AmbitoVinculoClave, d.AmbitoVinculoTipo = "candidato_ref", vecdomain.TipoReferenciaContextoActorCandidato
		}
		resultado[clave] = d
	}
	return resultado
}

type FuenteAmbitoLecturaRRHHInscripcionBolsa interface {
	ResolverAmbitoLecturaRRHH(context.Context, contextoSeguridadComunDesarrollo, AcreditacionSesionInscripcionBolsa, string, string, inscripcion.Filtro) (AmbitoLecturaRRHHInscripcionBolsa, error)
}

type AmbitoLecturaRRHHInscripcionBolsa struct {
	RecursoRef, ConvocatoriaRef string
	ConjuntoGestion             ConjuntoGestionRRHHInscripcionBolsa
	AmbitoSolicitud             *AmbitoSolicitudRRHHInscripcionBolsa
}

type ConjuntoGestionRRHHInscripcionBolsa struct {
	ConjuntoRef        string `json:"conjunto_ref"`
	UnidadRef          string `json:"unidad_ref"`
	AmbitoRef          string `json:"ambito_ref"`
	FuenteRef          string `json:"fuente_ref"`
	FuenteVersion      uint64 `json:"fuente_version"`
	FuenteHuellaSHA256 string `json:"fuente_sha256"`
}

type AmbitoSolicitudRRHHInscripcionBolsa struct {
	SolicitudRef       string `json:"solicitud_ref"`
	UnidadRef          string `json:"unidad_ref"`
	AmbitoRef          string `json:"ambito_ref"`
	FuenteRef          string `json:"fuente_ref"`
	FuenteVersion      uint64 `json:"fuente_version"`
	FuenteHuellaSHA256 string `json:"fuente_sha256"`
}

type ConfiguracionDecisorLecturaActualInscripcion struct {
	// Superficie fija qué fachada de autorización se consulta y qué
	// operaciones existen: externa_personal o interna_corporativa.
	Superficie   string
	Reloj        interface{ Ahora() time.Time }
	Descriptores map[ClaveOperacionInscripcionBolsa]DescriptorLecturaActualInscripcion
	AmbitoRRHH   FuenteAmbitoLecturaRRHHInscripcionBolsa
	RRHHNominal  []identidadConsultaRRHHDesarrollo
}

type decisorLecturaActualInscripcion struct {
	superficie   string
	fuente       vecports.FuenteAutorizacion
	reloj        interface{ Ahora() time.Time }
	descriptores map[ClaveOperacionInscripcionBolsa]DescriptorLecturaActualInscripcion
	ambitoRRHH   FuenteAmbitoLecturaRRHHInscripcionBolsa
	rrhhNominal  []identidadRRHHInscripcionBolsa
}

var _ DecisorLecturaActualInscripcionBolsa = (*decisorLecturaActualInscripcion)(nil)
var _ FuenteActualEscrituraInscripcionBolsa = (*decisorLecturaActualInscripcion)(nil)

// El pool debe usar el login de lectura nominal de su superficie: la fachada
// externa en el portal externo y la interna en vec-server. La construcción
// nunca crea roles ni grants.
func NuevoDecisorLecturaActualInscripcionPostgreSQL(pool *pgxpool.Pool, c ConfiguracionDecisorLecturaActualInscripcion) (*decisorLecturaActualInscripcion, error) {
	if pool == nil {
		return nil, inscripcion.ErrNoDisponible
	}
	var fuente vecports.FuenteAutorizacion
	var err error
	switch c.Superficie {
	case superficieExternaInscripcionBolsa:
		fuente, err = postgresvec.NuevoAlmacenAutorizacionExterna(pool)
	case superficieInternaInscripcionBolsa:
		fuente, err = postgresvec.NuevoAlmacenAutorizacion(pool)
	default:
		return nil, inscripcion.ErrNoDisponible
	}
	if err != nil {
		return nil, inscripcion.ErrNoDisponible
	}
	return nuevoDecisorLecturaActualInscripcionFuente(fuente, c)
}

// Este seam privado permite probar denegaciones sin abrir conexiones. La
// composición real sólo expone el constructor PostgreSQL anterior.
func nuevoDecisorLecturaActualInscripcionFuente(fuente vecports.FuenteAutorizacion, c ConfiguracionDecisorLecturaActualInscripcion) (*decisorLecturaActualInscripcion, error) {
	claves := clavesLecturaInscripcionBolsa(c.Superficie)
	interna := c.Superficie == superficieInternaInscripcionBolsa
	if len(claves) == 0 || nuloInscripcionBolsa(fuente) || nuloInscripcionBolsa(c.Reloj) || len(c.Descriptores) != len(claves) ||
		interna == nuloInscripcionBolsa(c.AmbitoRRHH) ||
		(interna && !identidadesRRHHInscripcionValidas(c.RRHHNominal)) || (!interna && len(c.RRHHNominal) != 0) {
		return nil, inscripcion.ErrNoDisponible
	}
	clon := make(map[ClaveOperacionInscripcionBolsa]DescriptorLecturaActualInscripcion, len(claves))
	for _, clave := range claves {
		d, ok := c.Descriptores[clave]
		tipo, finalidad, esperado := tipoFinalidadInscripcionBolsa(clave)
		if !ok || !esperado || d.Accion != clave.Accion || d.ModuloID != "bolsa" ||
			d.TipoRecurso != tipo || d.Finalidad != finalidad || len(d.Campos) == 0 || !slices.IsSorted(d.Campos) {
			return nil, inscripcion.ErrNoDisponible
		}
		if clave.Canal == superficieExternaInscripcionBolsa && (d.AmbitoVinculoClave != "candidato_ref" || d.AmbitoVinculoTipo != vecdomain.TipoReferenciaContextoActorCandidato) ||
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
	return &decisorLecturaActualInscripcion{superficie: c.Superficie, fuente: fuente, reloj: c.Reloj, descriptores: clon,
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
		(filtro.Validar() != nil || filtro.Estado != "" || filtro.ConvocatoriaRef != "" ||
			(filtro.Cursor != "" && !inscripcion.ConvocatoriaRefValida(filtro.Cursor))) {
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
	fuente := d.fuenteParaSuperficie(v.Superficie)
	if nuloInscripcionBolsa(fuente) {
		return vacia, inscripcion.ErrNoDisponible
	}
	ambitos := map[string]string{}
	atributos := map[string]string{}
	var conjuntoCaptura *ConjuntoGestionRRHHInscripcionBolsa
	var ambitoSolicitudCaptura *AmbitoSolicitudRRHHInscripcionBolsa
	if accionRRHHInscripcion(accion) {
		if !rrhhNominalInscripcionEnLista(s, a, d.rrhhNominal) || nuloInscripcionBolsa(d.ambitoRRHH) {
			return vacia, inscripcion.ErrAccesoDenegado
		}
		resuelto, err := d.ambitoRRHH.ResolverAmbitoLecturaRRHH(ctx, s, a, accion, referencia, filtro)
		conjunto := resuelto.ConjuntoGestion
		if err != nil || resuelto.RecursoRef != referencia || conjunto.ConjuntoRef == "" ||
			conjunto.UnidadRef == "" || conjunto.AmbitoRef == "" || conjunto.FuenteRef == "" ||
			conjunto.FuenteVersion == 0 || !huellaCertificadoInscripcionValida(conjunto.FuenteHuellaSHA256) {
			return vacia, inscripcion.ErrAccesoDenegado
		}
		ambitos["unidad_ref"], ambitos["ambito_ref"] = conjunto.UnidadRef, conjunto.AmbitoRef
		conjuntoCaptura = &ConjuntoGestionRRHHInscripcionBolsa{ConjuntoRef: conjunto.ConjuntoRef,
			UnidadRef: conjunto.UnidadRef, AmbitoRef: conjunto.AmbitoRef, FuenteRef: conjunto.FuenteRef,
			FuenteVersion: conjunto.FuenteVersion, FuenteHuellaSHA256: conjunto.FuenteHuellaSHA256}
		if accion == accionListarConvocatoriasGestionRRHHInscripcion {
			atributos["conjunto_ref"] = conjunto.ConjuntoRef
		}
		if accion == inscripcion.AccionListarRRHH {
			if resuelto.ConvocatoriaRef != filtro.ConvocatoriaRef {
				return vacia, inscripcion.ErrAccesoDenegado
			}
			atributos["convocatoria_ref"] = resuelto.ConvocatoriaRef
		}
		if accion == inscripcion.AccionDetalleRRHH {
			historico := resuelto.AmbitoSolicitud
			if historico == nil || historico.SolicitudRef != referencia || historico.UnidadRef != conjunto.UnidadRef ||
				historico.AmbitoRef != conjunto.AmbitoRef || historico.FuenteRef == "" || historico.FuenteVersion == 0 ||
				!huellaCertificadoInscripcionValida(historico.FuenteHuellaSHA256) {
				return vacia, inscripcion.ErrAccesoDenegado
			}
			ambitoSolicitudCaptura = &AmbitoSolicitudRRHHInscripcionBolsa{SolicitudRef: historico.SolicitudRef,
				UnidadRef: historico.UnidadRef, AmbitoRef: historico.AmbitoRef, FuenteRef: historico.FuenteRef,
				FuenteVersion: historico.FuenteVersion, FuenteHuellaSHA256: historico.FuenteHuellaSHA256}
		} else if resuelto.AmbitoSolicitud != nil {
			return vacia, inscripcion.ErrAccesoDenegado
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
		ConjuntoGestion: conjuntoCaptura, AmbitoSolicitud: ambitoSolicitudCaptura,
	}, nil
}

func (d *decisorLecturaActualInscripcion) fuenteParaSuperficie(superficie vecdomain.SuperficieAutenticacionActorV1) vecports.FuenteAutorizacion {
	if d == nil {
		return nil
	}
	if string(superficie) != d.superficie {
		return nil
	}
	return d.fuente
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

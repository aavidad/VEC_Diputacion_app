package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"time"

	bolsadomain "vec-diputacion-granada/internal/modules/bolsa/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// perfilPropioBolsaRRHH expone sólo la parte de la asignación vigente que
// necesita RRHH para entender su ámbito y las acciones de Bolsa publicadas.
// Cada operación sigue necesitando su autorización nominal independiente.
type perfilPropioBolsaRRHH struct {
	Esquema string `json:"esquema"`
	Rol     struct {
		Nombre  string `json:"nombre"`
		Version int    `json:"version"`
	} `json:"rol"`
	Ambito struct {
		UnidadRef string `json:"unidad_ref"`
		AmbitoRef string `json:"ambito_ref"`
	} `json:"ambito"`
	Acciones     []string                       `json:"acciones"`
	Canales      []bolsadomain.CanalLlamamiento `json:"canales"`
	ConsultadaEn string                         `json:"consultada_en"`
}

// ConsultarPerfilBolsaRRHH lee de la fuente nominal la versión vinculada al
// perfil de esta petición. No usa el perfil de desarrollo guardado en memoria.
// La ruta HTTP que la consuma debe registrar el intento y el resultado de esta
// lectura mediante la auditoría durable ya compuesta en la frontera RRHH.
func (m *manejadorParticipacionBolsaDesarrollo) ConsultarPerfilBolsaRRHH(ctx context.Context) (salida any, fallo error) {
	denegado := vecdomain.ErrAutorizacionDenegada
	rechazar := func(clave string, causas ...error) (any, error) {
		slog.Warn("bolsa_configuracion_denegada", "clave", clave, "esperado", true, "actual", false)
		return nil, errors.Join(append([]error{denegado}, causas...)...)
	}
	if m == nil || m.preparador == nil || m.preparador.soporte == nil ||
		m.registradorPerfil == nil || m.procesoPerfil == "" || ctx == nil || ctx.Err() != nil {
		return nil, vecports.ErrFuenteAutorizacionNoDisponible
	}
	p := m.preparador
	contexto, err := p.contextoRevalidado(ctx)
	if err != nil {
		return rechazar("sesion_nominal", err)
	}
	defer func() {
		resultado := vecdomain.ResultadoIntentoAuditoriaExito
		if fallo != nil || ctx.Err() != nil {
			resultado = vecdomain.ResultadoIntentoAuditoriaError
			if errors.Is(fallo, vecdomain.ErrAutorizacionDenegada) {
				resultado = vecdomain.ResultadoIntentoAuditoriaDenegado
			}
		}
		acuse, errAudit := m.registrarConsultaPerfilBolsaRRHH(ctx, contexto, resultado)
		if errAudit != nil {
			salida, fallo = nil, errors.Join(vecports.ErrIntentoAuditoriaNoDisponible, errAudit)
			return
		}
		if ctx.Err() != nil {
			salida, fallo = nil, ctx.Err()
			return
		}
		if perfil, ok := salida.(perfilPropioBolsaRRHH); ok {
			perfil.ConsultadaEn = acuse.RegistradaEn.UTC().Format(time.RFC3339Nano)
			salida = perfil
		}
	}()
	if p.fuenteRespuestas == nil {
		return nil, vecports.ErrFuenteAutorizacionNoDisponible
	}
	if p.soporte.unidadRef == "" || p.soporte.ambitoRef == "" {
		return rechazar("ambito_servidor")
	}
	datos, err := contexto.Vinculo.Datos()
	if err != nil || datos.Superficie != "interna_corporativa" || datos.CuentaPrivilegiada {
		return rechazar("canal_rrhh", err)
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	if !contexto.Vinculo.VigenteEn(ahora, contexto.Resultado) {
		return rechazar("sesion_vigente")
	}
	i, err := p.fuenteRespuestas.ObtenerInstantaneaAutorizacion(ctx, datos.PrincipalID, datos.PerfilActivoRef)
	if err != nil || ctx.Err() != nil {
		return nil, errors.Join(vecports.ErrFuenteAutorizacionNoDisponible, err, ctx.Err())
	}
	ahora = time.Now().UTC().Truncate(time.Microsecond)
	for _, guarda := range []struct {
		clave  string
		valida bool
	}{
		{"instantanea_valida", i.Validar() == nil},
		{"sesion_vigente", contexto.Vinculo.VigenteEn(ahora, contexto.Resultado)},
		{"persona_propia", i.AsignacionPerfil.PrincipalID == datos.PrincipalID},
		{"perfil_propio", i.AsignacionPerfil.PerfilActivoRef == datos.PerfilActivoRef},
		{"asignacion_vigente", i.AsignacionPerfil.VigenteEn(ahora)},
		{"rol_rrhh_bolsa", i.VersionRol.RolID == vecdomain.RolIDVersionarBolsaB1},
		{"rol_publicado", i.VersionRol.Estado == vecdomain.EstadoVersionRolPublicada && !i.VersionRol.PublicadaEn.After(ahora)},
		{"control_habilitado", i.ControlVigenciaVersionRol.Estado == vecdomain.EstadoControlVigenciaVersionRolHabilitada && !i.ControlVigenciaVersionRol.ActualizadoEn.After(ahora)},
		{"asignacion_autoritativa", i.AsignacionPerfil.EmitidaPor != "identidad:desarrollo:no-autoritativa"},
		{"rol_autoritativo", i.VersionRol.PublicadaPor != "seguridad:desarrollo:no-autoritativa"},
		{"control_autoritativo", i.ControlVigenciaVersionRol.ActualizadoPor != "seguridad:desarrollo:no-autoritativa"},
	} {
		if !guarda.valida {
			return rechazar(guarda.clave)
		}
	}

	// La asignación sólo debe contener las dos dimensiones privadas de esta
	// sesión, cada una con el valor exacto que resolverá la frontera HTTP.
	if len(i.AsignacionPerfil.Ambitos) != 2 {
		return rechazar("dimensiones_ambito")
	}
	valores := map[string]string{"unidad_ref": p.soporte.unidadRef, "ambito_ref": p.soporte.ambitoRef}
	for _, ambito := range i.AsignacionPerfil.Ambitos {
		valor, ok := valores[ambito.Clave]
		if !ok || len(ambito.Valores) != 1 || ambito.Valores[0] != valor {
			return rechazar("ambito_exacto")
		}
		delete(valores, ambito.Clave)
	}
	if len(valores) != 0 {
		return rechazar("ambito_completo")
	}
	recurso := vecdomain.RecursoAutorizable{Referencia: "bolsa:perfil:" + huellaAltaContratacionTemporalDesarrollo(datos.PerfilActivoRef), ModuloID: "bolsa", Tipo: "perfil_rrhh_bolsa", Ambitos: map[string]string{"unidad_ref": p.soporte.unidadRef, "ambito_ref": p.soporte.ambitoRef}, Atributos: map[string]string{"perfil_ref": datos.PerfilActivoRef}}
	evaluacion, err := vecdomain.EvaluarLecturaActualAutorizacionV3(contexto.Vinculo, contexto.Resultado, i, "bolsa.configuracion.consultar", recurso, "consulta_configuracion_bolsa_rrhh", camposConsultaPerfilBolsaRRHH(), ahora)
	if err != nil || !evaluacion.Permitido || !slices.Equal(evaluacion.CamposPermitidos, camposConsultaPerfilBolsaRRHH()) || !slices.Equal(evaluacion.Obligaciones, []string{"auditar"}) {
		return rechazar("concesion_configuracion", err)
	}
	acciones := make([]string, 0, len(i.VersionRol.Concesiones))
	for _, c := range i.VersionRol.Concesiones {
		if c.ModuloID == "bolsa" && c.Accion != "" {
			acciones = append(acciones, c.Accion)
		}
	}
	if len(acciones) == 0 || ctx.Err() != nil {
		return nil, denegado
	}
	slices.Sort(acciones)
	acciones = slices.Compact(acciones)
	respuesta := perfilPropioBolsaRRHH{Esquema: "vec.bolsa.rrhh.perfil.v1", Acciones: acciones, Canales: m.CanalesLlamamientoActivos(ctx)}
	if respuesta.Canales == nil {
		respuesta.Canales = []bolsadomain.CanalLlamamiento{}
	}
	respuesta.Rol.Nombre, respuesta.Rol.Version = i.VersionRol.Nombre, i.VersionRol.Version
	respuesta.Ambito.UnidadRef, respuesta.Ambito.AmbitoRef = p.soporte.unidadRef, p.soporte.ambitoRef
	return respuesta, nil
}

// El único efecto de esta consulta es su apunte nominal: se confirma antes
// de entregar la proyección y conserva incluso una cancelación del navegador.
func (m *manejadorParticipacionBolsaDesarrollo) registrarConsultaPerfilBolsaRRHH(ctx context.Context, actor contextoSeguridadComunDesarrollo, resultado vecdomain.ResultadoIntentoAuditoria) (vecports.AcuseIntentoAuditoria, error) {
	var vacio vecports.AcuseIntentoAuditoria
	protegido, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancelar()
	referencia, err := vecports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return vacio, err
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(protegido, m.preparador.generar)
	if err != nil {
		return vacio, err
	}
	canon, err := correlacion.ValorCanonico()
	if err != nil {
		return vacio, err
	}
	orden, err := vecports.NuevaOrdenIntentoAuditoria(referencia, actor.Resultado, actor.Vinculo, vecdomain.DatosIntentoAuditoria{
		Accion: "bolsa.configuracion.consultar", ModuloID: "bolsa", RecursoRef: "bolsa:perfil:" + huellaAltaContratacionTemporalDesarrollo(actor.Resultado.Contexto.PerfilActivoRef),
		FinalidadRef: "consulta_configuracion_bolsa_rrhh", Resultado: resultado, Motivo: motivoConsultarBorradorLlamamientoBolsaDesarrollo(),
		Proceso: m.procesoPerfil, Canal: "interna_corporativa", CorrelacionRef: canon,
	})
	if err != nil {
		return vacio, err
	}
	acuse, err := m.registradorPerfil.AppendIntentoAuditoria(protegido, orden)
	if err != nil {
		return vacio, err
	}
	if err := acuse.ValidarPara(orden); err != nil {
		return vacio, err
	}
	return acuse, nil
}

func camposConsultaPerfilBolsaRRHH() []string {
	return []string{"acciones[]", "ambito.ambito_ref", "ambito.unidad_ref", "canales[].activo", "canales[].acuse", "canales[].admite_respuesta", "canales[].al_emitir", "canales[].canal", "canales[].limite_caracteres", "canales[].modo", "canales[].requiere_alta_voluntaria", "canales[].resultados[]", "canales[].resultados_cierre[]", "canales[].seguimiento", "consultada_en", "rol.nombre", "rol.version"}
}

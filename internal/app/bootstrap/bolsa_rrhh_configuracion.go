package bootstrap

import (
	"context"
	"errors"
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
	if m == nil || m.preparador == nil || m.preparador.soporte == nil ||
		m.preparador.fuenteRespuestas == nil || m.registradorPerfil == nil || m.procesoPerfil == "" || ctx == nil || ctx.Err() != nil {
		return nil, vecports.ErrFuenteAutorizacionNoDisponible
	}
	p := m.preparador
	if p.soporte.unidadRef == "" || p.soporte.ambitoRef == "" {
		return nil, denegado
	}
	contexto, err := p.contextoRevalidado(ctx)
	if err != nil {
		return nil, denegado
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
			salida, fallo = nil, vecports.ErrIntentoAuditoriaNoDisponible
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
	datos, err := contexto.Vinculo.Datos()
	if err != nil || datos.Superficie != "interna_corporativa" || datos.CuentaPrivilegiada {
		return nil, denegado
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	if !contexto.Vinculo.VigenteEn(ahora, contexto.Resultado) {
		return nil, denegado
	}
	i, err := p.fuenteRespuestas.ObtenerInstantaneaAutorizacion(ctx, datos.PrincipalID, datos.PerfilActivoRef)
	if err != nil || ctx.Err() != nil {
		return nil, vecports.ErrFuenteAutorizacionNoDisponible
	}
	ahora = time.Now().UTC().Truncate(time.Microsecond)
	if i.Validar() != nil || !contexto.Vinculo.VigenteEn(ahora, contexto.Resultado) ||
		i.AsignacionPerfil.PrincipalID != datos.PrincipalID ||
		i.AsignacionPerfil.PerfilActivoRef != datos.PerfilActivoRef ||
		!i.AsignacionPerfil.VigenteEn(ahora) ||
		i.VersionRol.RolID != vecdomain.RolIDVersionarBolsaB1 ||
		i.VersionRol.Estado != vecdomain.EstadoVersionRolPublicada ||
		i.VersionRol.PublicadaEn.After(ahora) ||
		i.ControlVigenciaVersionRol.Estado != vecdomain.EstadoControlVigenciaVersionRolHabilitada ||
		i.ControlVigenciaVersionRol.ActualizadoEn.After(ahora) ||
		i.AsignacionPerfil.EmitidaPor == "identidad:desarrollo:no-autoritativa" ||
		i.VersionRol.PublicadaPor == "seguridad:desarrollo:no-autoritativa" ||
		i.ControlVigenciaVersionRol.ActualizadoPor == "seguridad:desarrollo:no-autoritativa" {
		return nil, denegado
	}
	// La asignación sólo debe contener las dos dimensiones privadas de esta
	// sesión, cada una con el valor exacto que resolverá la frontera HTTP.
	if len(i.AsignacionPerfil.Ambitos) != 2 {
		return nil, denegado
	}
	valores := map[string]string{"unidad_ref": p.soporte.unidadRef, "ambito_ref": p.soporte.ambitoRef}
	for _, ambito := range i.AsignacionPerfil.Ambitos {
		valor, ok := valores[ambito.Clave]
		if !ok || len(ambito.Valores) != 1 || ambito.Valores[0] != valor {
			return nil, denegado
		}
		delete(valores, ambito.Clave)
	}
	if len(valores) != 0 {
		return nil, denegado
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

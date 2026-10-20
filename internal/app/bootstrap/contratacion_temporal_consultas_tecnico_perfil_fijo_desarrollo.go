package bootstrap

import (
	"reflect"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	clavePerfilFijoConsultasTecnicoCTDesarrollo = "consultas_tecnico_rrhh"
	rolConsultasTecnicoCTDesarrollo             = "consulta_rrhh_tecnico_fijo_desarrollo"
)

// Las plantillas históricas y el perfil estable comparten la definición de
// acciones y ámbitos; la descarga sigue ligada solo al detalle.
func nuevasInstantaneasConsultasRRHHDesarrollo(principalID, perfilRef string, ahora time.Time,
	clase ports.ClaseAmbitoConsultaRRHH, ambitoRef string,
) (dominiovec.InstantaneaAutorizacion, dominiovec.InstantaneaAutorizacion, error) {
	nueva := func(rol, nombre, accion, finalidad, tipo string, adicionales ...dominiovec.ConcesionRol) (dominiovec.InstantaneaAutorizacion, error) {
		return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(principalID, perfilRef, ahora,
			rol, nombre, rol,
			append([]dominiovec.ConcesionRol{{Accion: accion, ModuloID: ports.ModuloContratacion,
				TipoRecurso: tipo, Finalidades: []string{finalidad}, GarantiaMinima: dominiovec.AuthAssuranceHigh}}, adicionales...),
			[]dominiovec.AmbitoPerfil{
				{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
				{Clave: "clase_ambito", Valores: []string{string(clase)}},
				{Clave: "ambito_ref", Valores: []string{ambitoRef}},
			})
	}
	cuadro, err := nueva("consulta_cuadro_rrhh_desarrollo", "Consulta de bandeja de desarrollo",
		ports.AccionConsultarCuadroRRHH, ports.FinalidadConsultarCuadroRRHH, ports.TipoRecursoCuadroRRHH)
	if err != nil {
		return dominiovec.InstantaneaAutorizacion{}, dominiovec.InstantaneaAutorizacion{}, err
	}
	detalle, err := nueva("consulta_detalle_rrhh_desarrollo", "Consulta de expediente de desarrollo",
		ports.AccionConsultarDetalleRRHH, ports.FinalidadConsultarDetalleRRHH, ports.TipoRecursoExpediente,
		dominiovec.ConcesionRol{Accion: ports.AccionDescargarBorradorRRHH, ModuloID: ports.ModuloContratacion,
			TipoRecurso: ports.TipoRecursoExpediente, Finalidades: []string{ports.FinalidadDescargarBorradorRRHH},
			GarantiaMinima: dominiovec.AuthAssuranceHigh})
	return cuadro, detalle, err
}

// El cuadro, el detalle y la descarga comparten una asignación estable de la
// técnica. Su cuenta y persona son las de Alta; la sesión y el perfil son
// propios. Las concesiones son exactamente la unión de las dos lecturas
// existentes: ninguna acción de escritura entra en este rol.
func componerPerfilFijoConsultasTecnicoCTDesarrollo(s *soporteAltaContratacionTemporalDesarrollo, ahora time.Time) error {
	if s == nil || s.lectorConsultasRRHH || s.principalID == "" || s.certificadoSHA256 == "" {
		return registrarFalloMontajeCTDesarrollo("consultas_tecnico_soporte", ports.ErrConsultaRRHHNoDisponible)
	}
	// El contexto registrado solo conserva la identidad mínima; el soporte
	// retiene la huella ya validada en la composición del técnico.
	principal := dominiovec.Principal{ID: s.principalID, Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo},
		AuthMethod: dominiovec.AuthMethodCertificate, AuthAssurance: dominiovec.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment,
			"certificate_sha256": s.certificadoSHA256}}
	if !principalContratacionTemporalDesarrolloValido(principal) {
		return registrarFalloMontajeCTDesarrollo("consultas_tecnico_principal", ports.ErrConsultaRRHHNoDisponible)
	}
	perfil, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, clavePerfilFijoConsultasTecnicoCTDesarrollo,
		[]string{httpinterno.RutaConsultaCuadroRRHH, httpinterno.RutaConsultaDetalleRRHH},
		func(principalID, perfilRef string) (dominiovec.InstantaneaAutorizacion, error) {
			cuadro, detalle, err := nuevasInstantaneasConsultasRRHHDesarrollo(principalID, perfilRef, ahora,
				ports.AmbitoOrganizacionRRHH, organizacionAltaContratacionTemporalDesarrollo)
			if err != nil || len(cuadro.VersionRol.Concesiones) != 1 || len(detalle.VersionRol.Concesiones) != 2 ||
				!reflect.DeepEqual(cuadro.AsignacionPerfil.Ambitos, detalle.AsignacionPerfil.Ambitos) {
				return dominiovec.InstantaneaAutorizacion{}, registrarFalloMontajeCTDesarrollo("consultas_tecnico_instantaneas", ports.ErrConsultaRRHHNoDisponible)
			}
			return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(principalID, perfilRef, ahora,
				rolConsultasTecnicoCTDesarrollo, "Consulta de bandeja y expediente de desarrollo", rolConsultasTecnicoCTDesarrollo,
				append(append([]dominiovec.ConcesionRol(nil), cuadro.VersionRol.Concesiones...), detalle.VersionRol.Concesiones...),
				cuadro.AsignacionPerfil.Ambitos)
		})
	if err != nil {
		return registrarFalloMontajeCTDesarrollo("consultas_tecnico_nuevo_perfil", err)
	}
	if err := s.registrarPerfilFijoCTDesarrollo(perfil); err != nil {
		return registrarFalloMontajeCTDesarrollo("consultas_tecnico_registrar_perfil", err)
	}
	return nil
}

func perfilFijoConsultasTecnicoCTDesarrollo(s *soporteAltaContratacionTemporalDesarrollo) *perfilFijoCTDesarrollo {
	if s == nil {
		return nil
	}
	for _, perfil := range s.perfilesFijosRegistrados() {
		if perfil.clave == clavePerfilFijoConsultasTecnicoCTDesarrollo &&
			perfil.atiende(httpinterno.RutaConsultaCuadroRRHH) && perfil.atiende(httpinterno.RutaConsultaDetalleRRHH) {
			return perfil
		}
	}
	return nil
}

func ligarAutoridadConsultasTecnicoPerfilFijoDesarrollo(
	a *autoridadConsultasRRHHDesarrollo, p *perfilFijoCTDesarrollo,
) error {
	if a == nil || a.soporte == nil || p == nil || p.clave != clavePerfilFijoConsultasTecnicoCTDesarrollo ||
		a.soporte.perfilFijoParaRuta(httpinterno.RutaConsultaCuadroRRHH) != p ||
		a.soporte.perfilFijoParaRuta(httpinterno.RutaConsultaDetalleRRHH) != p ||
		p.contextoEsperadoRegistrado.Validar() != nil {
		return registrarFalloMontajeCTDesarrollo("consultas_tecnico_ligar_perfil", ports.ErrConsultaRRHHNoDisponible)
	}
	proveedor, ok := p.sesionOperativa.(*proveedorSesionConsultaRRHHDesarrollo)
	if !ok || proveedor == nil {
		return registrarFalloMontajeCTDesarrollo("consultas_tecnico_ligar_sesion", ports.ErrConsultaRRHHNoDisponible)
	}
	a.perfilLectura = p
	return registrarFalloMontajeCTDesarrollo("consultas_tecnico_ligar_proveedor", a.configurarProveedorContextoConsultaRRHHDesarrollo(proveedor))
}

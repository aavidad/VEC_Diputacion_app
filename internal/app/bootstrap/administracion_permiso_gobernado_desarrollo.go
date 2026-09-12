package bootstrap

import (
	"context"
	"encoding/json"
	"time"

	adminmodule "vec-diputacion-granada/internal/modules/administracion"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Este adaptador reutiliza el PDP V3 y su registro CAS existente. No
// interpreta nombres de rol ni concede permisos desde el material nominal.
// El permiso de acceso no sustituye la V3 ligada a los bytes de la mutación.
type fuentePermisoAdministracionGobernadaV3 struct {
	autorizador vecports.AutorizadorSolicitudLigadaV3
	motivo      vecdomain.ReferenciaEntradaCatalogo
	referencias vecports.GeneradorReferenciasAutorizacionV2
	reloj       vecports.Reloj
}

func nuevaFuentePermisoAdministracionGobernadaV3(autorizador vecports.AutorizadorSolicitudLigadaV3, motivo vecdomain.ReferenciaEntradaCatalogo, referencias vecports.GeneradorReferenciasAutorizacionV2, reloj vecports.Reloj) (*fuentePermisoAdministracionGobernadaV3, error) {
	if dependenciaAdministracionNula(autorizador) || !vecdomain.ReferenciaMotivoAutorizacionV2Valida(motivo) ||
		dependenciaAdministracionNula(referencias) || dependenciaAdministracionNula(reloj) {
		return nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	return &fuentePermisoAdministracionGobernadaV3{autorizador: autorizador, motivo: motivo, referencias: referencias, reloj: reloj}, nil
}

func (f *fuentePermisoAdministracionGobernadaV3) ResolverPrincipalAdministracionCorreoV3(ctx context.Context, vinculo vecdomain.VinculoAutenticacionActorV2, resultado vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.Principal, error) {
	denegar := func() (vecdomain.Principal, error) {
		return vecdomain.Principal{}, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	if f == nil || dependenciaAdministracionNula(f.autorizador) || dependenciaAdministracionNula(f.referencias) || dependenciaAdministracionNula(f.reloj) ||
		!contextoPermisoAdministracionGobernadoValido(ctx, vinculo, resultado, f.reloj.Ahora()) {
		return denegar()
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, f.referencias)
	if err != nil {
		return denegar()
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: vinculo,
		ReferenciaMotivo:          f.motivo,
		Accion:                    adminmodule.PermissionIntegrationsManage,
		Recurso: vecdomain.RecursoAutorizable{
			Referencia: referenciaConfiguracionCorreoAdministracionV3,
			ModuloID:   adminmodule.ModuleID,
			Tipo:       tipoRecursoConfiguracionCorreoAdministracion,
			Ambitos:    map[string]string{"organizacion_ref": organizacionConfiguracionCorreoAdministracionV3},
		},
		Finalidad:   finalidadConfiguracionCorreoAdministracionV3,
		Correlacion: correlacion,
	})
	if err != nil {
		return denegar()
	}
	decision, confirmacion, err := f.autorizador.ExigirSolicitudLigadaV3(ctx, solicitud, resultado)
	if err != nil || decision.ValidarPara(solicitud) != nil {
		return denegar()
	}
	orden, err := vecports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(solicitud, decision, f.motivo, resultado)
	ahora := f.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if err != nil || confirmacion.ValidarPara(orden) != nil || !confirmacion.DentroDeVentanaEn(ahora) ||
		!contextoPermisoAdministracionGobernadoValido(ctx, vinculo, resultado, ahora) {
		return denegar()
	}
	// Proyección informativa para el caso de uso; las escrituras conservan
	// su consumo V3 separado, ligado al CAS y al sobre cifrado exactos.
	// El rol aquí es evidencia informativa de la decisión ya confirmada,
	// nunca una concesión inferida del certificado ni de datos del cliente.
	canonico, err := vecdomain.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	if err != nil {
		return denegar()
	}
	var evidencia struct {
		VersionRolRef string `json:"version_rol_ref"`
	}
	if json.Unmarshal(canonico, &evidencia) != nil || evidencia.VersionRolRef == "" {
		return denegar()
	}
	principal := clonarPrincipalDesarrollo(resultado.Contexto.Principal)
	principal.Roles = []string{evidencia.VersionRolRef}
	principal.Permissions = []string{adminmodule.PermissionIntegrationsManage}
	return principal, nil
}

func contextoPermisoAdministracionGobernadoValido(ctx context.Context, vinculo vecdomain.VinculoAutenticacionActorV2, resultado vecdomain.ResultadoContextoActorRegistradoV2, ahora time.Time) bool {
	capacidad, valida := capacidadAdministracionDesdeContexto(ctx)
	if !valida || resultado.Validar() != nil || vinculo.ValidarPara(resultado) != nil ||
		!resultado.Contexto.Instantanea.VigenteEn(ahora) || resultado.Contexto.PersonaRef != capacidad.identidad.personaRef ||
		resultado.Contexto.PerfilActivoRef != capacidad.identidad.perfilRef || resultado.Contexto.Instantanea.CuentaRef != capacidad.identidad.cuentaRef {
		return false
	}
	datos, err := vinculo.Datos()
	return err == nil && datos.CuentaPrivilegiada && datos.CuentaRef == capacidad.identidad.cuentaRef &&
		datos.CuentaOrdinariaRef == capacidad.identidad.cuentaOrdinariaRef && datos.Superficie == vecdomain.SuperficieAutenticacionAdministracionPrivilegiadaV1 &&
		datos.MetodoObservado == vecdomain.AuthMethodCertificate && datos.GarantiaObservada == vecdomain.AuthAssuranceHigh &&
		!ahora.Before(datos.SesionRevalidadaEn) && ahora.Before(datos.SesionValidaHasta)
}

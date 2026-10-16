package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// El acuse solo nace del selector AUT para esta petición. No es una concesión
// documental: D27 y el PDP vuelven a cotejar acción y ámbito del recurso.
type seleccionNominalJustificanteRRHH struct {
	perfil, auditoria, ruta, metodo string
	catalogo                        *identidadCatalogoFronterasComunDesarrollo
	holder                          *contextoOperacionCTDesarrollo
}

type cuentaNominalJustificanteRRHH struct {
	certificadoSHA256, sujeto, cuentaRef string
}

var auditoriaNominalJustificanteRRHH = regexp.MustCompile(`^aud_v3_b_[0-9a-f]{32}$`)
var codigoSelectorJustificanteRRHH = regexp.MustCompile(`^sqlstate_[0-9A-Z]{5}$`)
var errConfirmacionSesionNominalJustificanteRRHH = errors.New("justificante RRHH: confirmación de sesión divergente")
var errDependenciaSesionNominalJustificanteRRHH = errors.New("justificante RRHH: dependencia nominal ausente")
var errAcuseSelectorNominalJustificanteRRHH = errors.New("justificante RRHH: acuse nominal divergente")

type falloSesionNominalJustificanteRRHH struct {
	etapa string
	causa error
}

func (f falloSesionNominalJustificanteRRHH) Error() string {
	return "justificante RRHH: sesión nominal no disponible (" + f.etapa + ")"
}
func (f falloSesionNominalJustificanteRRHH) Unwrap() []error {
	if f.causa == nil {
		return []error{docports.ErrSubidaLocalNoDisponible}
	}
	return []error{docports.ErrSubidaLocalNoDisponible, f.causa}
}

func noDisponibleSesionNominalJustificanteRRHH(etapa string, err error) error {
	causa := "error_operacion"
	var pg interface{ SQLState() string }
	if errors.As(err, &pg) && codigoSelectorJustificanteRRHH.MatchString("sqlstate_"+pg.SQLState()) {
		causa = "sqlstate_" + pg.SQLState()
	}
	slog.Warn("justificante_renuncia_bolsa_rrhh_no_disponible", "etapa", etapa, "causa", causa)
	return falloSesionNominalJustificanteRRHH{etapa: etapa, causa: err}
}

type diagnosticoSelectorJustificanteRRHH interface {
	DiagnosticoSelectorPerfilJustificante() (string, string)
}

func registrarFalloSelectorJustificanteRRHH(err error) {
	etapa, causa := "selector_perfil", "fuente_autorizacion"
	var d diagnosticoSelectorJustificanteRRHH
	if errors.As(err, &d) {
		etapa, causa = d.DiagnosticoSelectorPerfilJustificante()
		switch etapa {
		case "dependencias", "transaccion", "configuracion", "consulta", "respuesta", "acuse", "confirmacion":
		default:
			etapa = "selector_perfil"
		}
		if causa != "error_operacion" && !codigoSelectorJustificanteRRHH.MatchString(causa) {
			causa = "fuente_autorizacion"
		}
	}
	slog.Warn("justificante_renuncia_bolsa_rrhh_no_disponible", "etapa", etapa, "causa", causa)
}

func perfilJustificanteAdmitidoPorFrontera(f fronteraSeguridadComunDesarrollo, z contextoSeguridadComunDesarrollo) bool {
	if f.ruta != bolsahttp.RutaJustificanteRenunciaRRHH || f.metodo != http.MethodGet {
		return z.seleccionJustificanteRRHH == nil &&
			f.descriptor.admitePerfil(z.Resultado.Contexto.PerfilActivoRef)
	}
	p := z.seleccionJustificanteRRHH
	return p != nil && f.ruta == bolsahttp.RutaJustificanteRenunciaRRHH &&
		f.metodo == http.MethodGet && f.descriptor.Clave == "bolsa-justificante-renuncia-rrhh" &&
		p.ruta == f.ruta && p.metodo == f.metodo && p.catalogo == f.catalogo.identidad &&
		p.holder != nil && p.holder == f.holder &&
		p.perfil == z.Resultado.Contexto.PerfilActivoRef &&
		auditoriaNominalJustificanteRRHH.MatchString(p.auditoria)
}

func (s sesionTecnicaJustificanteRenunciaRRHH) resolverNominal(ctx context.Context,
	c capacidadConsultaContratacionTemporalDesarrollo,
	f fronteraSeguridadComunDesarrollo,
) (contextoSeguridadComunDesarrollo, error) {
	vacia := contextoSeguridadComunDesarrollo{}
	usuarios := s.usuarios
	if usuarios == nil || usuarios.base == nil || usuarios.base.registro == nil ||
		usuarios.base.revalidador == nil || usuarios.base.contextos == nil || s.selector == nil {
		return vacia, noDisponibleSesionNominalJustificanteRRHH("dependencias", errDependenciaSesionNominalJustificanteRRHH)
	}
	if !principalNominalJustificanteRRHHValido(c.principal, s.cuentas) {
		return vacia, docports.ErrSubidaLocalDenegada
	}
	cuenta := s.cuentas[c.principal.Attributes["certificate_sha256"]]
	ahora := relojRutasDietas{}.Ahora()
	if ahora.Before(c.certificadoVerificadoEn) || !ahora.Before(c.certificadoValidoHasta) {
		return vacia, docports.ErrSubidaLocalDenegada
	}
	asercion, err := nonceRutasDietas()
	if err != nil {
		return vacia, noDisponibleSesionNominalJustificanteRRHH("nonce_asercion", err)
	}
	sesion, err := nonceRutasDietas()
	if err != nil {
		return vacia, noDisponibleSesionNominalJustificanteRRHH("nonce_sesion", err)
	}
	hasta := ahora.Add(2 * time.Minute)
	if limite := c.certificadoValidoHasta.UTC().Truncate(time.Microsecond); limite.Before(hasta) {
		hasta = limite
	}
	politica := "dev-certificado-mtls-v1;solo-sintetico;canal-privado-validado;garantia-alta-desarrollo;vigencia-120s;sin-kerberos;no-corporativa"
	alta := httpseguridad.AltaSesionAtomica{
		AsercionID: asercion, SesionID: sesion, SujetoID: cuenta.sujeto,
		CuentaID:         "desarrollo:" + cuenta.cuentaRef,
		Superficie:       httpseguridad.SuperficieInternaCorporativa,
		EspacioIdentidad: espacioIdentidadSesionDesarrollo,
		MetodoObservado:  core.AuthMethodCertificate, GarantiaObservada: core.AuthAssuranceHigh,
		AutenticacionVerificadaEn: c.certificadoVerificadoEn,
		SesionEmitidaEn:           ahora, AsercionExpiraEn: hasta,
		PoliticaGarantiaRef:          referenciaAltaContratacionTemporalDesarrollo("pga_", "dev-certificado-mtls-v1"),
		PoliticaGarantiaHuellaSHA256: huellaRutasDietas(politica),
		AutenticacionHuellaSHA256: huellaRutasDietas(usuarios.base.instancia + "|" + asercion + "|" + sesion + "|" +
			cuenta.certificadoSHA256 + "|" + cuenta.cuentaRef + "|" + bolsahttp.RutaJustificanteRenunciaRRHH + "|" + http.MethodGet + "|" + ahora.Format(time.RFC3339Nano)),
	}
	confirmacion, err := usuarios.base.registro.ConsumirAsercionYRegistrar(ctx, alta)
	if err != nil {
		return vacia, noDisponibleSesionNominalJustificanteRRHH("registro_sesion", err)
	}
	if err = confirmacion.ValidarPara(alta); err != nil || confirmacion.CuentaRef != cuenta.cuentaRef {
		if err == nil {
			err = errConfirmacionSesionNominalJustificanteRRHH
		}
		return vacia, noDisponibleSesionNominalJustificanteRRHH("confirmacion_sesion", err)
	}
	corr, err := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacia, noDisponibleSesionNominalJustificanteRRHH("correlacion", err)
	}
	correlacion, err := corr.ValorCanonico()
	if err != nil {
		return vacia, noDisponibleSesionNominalJustificanteRRHH("correlacion_valor", err)
	}
	seleccion, err := s.selector.SeleccionarPerfilJustificanteBolsaRRHH(ctx,
		vecports.SolicitudPerfilJustificanteBolsaRRHH{
			AutenticacionRef: confirmacion.AutenticacionRef, SesionRef: confirmacion.SesionRef,
			CuentaConfirmadaRef:   confirmacion.CuentaRef,
			ControlSesionRef:      confirmacion.ControlSesionRef,
			ControlSesionRevision: confirmacion.ControlSesionRevision,
			ControlSesionSHA256:   confirmacion.ControlSesionHuellaSHA256,
			AutenticacionSHA256:   alta.AutenticacionHuellaSHA256,
			CorrelacionRef:        correlacion,
		})
	if err != nil {
		if errors.Is(err, vecports.ErrFuenteAutorizacionNoDisponible) {
			registrarFalloSelectorJustificanteRRHH(err)
			return vacia, errors.Join(docports.ErrSubidaLocalNoDisponible, err)
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return vacia, noDisponibleSesionNominalJustificanteRRHH("selector_perfil", err)
		}
		return vacia, docports.ErrSubidaLocalDenegada
	}
	if seleccion.PerfilActivoRef == "" {
		return vacia, docports.ErrSubidaLocalDenegada
	}
	if !auditoriaNominalJustificanteRRHH.MatchString(seleccion.AuditoriaRef) {
		return vacia, noDisponibleSesionNominalJustificanteRRHH("acuse_selector", errAcuseSelectorNominalJustificanteRRHH)
	}
	revalidador := revalidadorSesionConsultaRRHHDesarrollo{
		delegado: usuarios.base.revalidador, alta: alta, confirmacion: confirmacion,
		reloj: usuarios.base.reloj, superficie: httpseguridad.SuperficieInternaCorporativa,
	}
	vinculo, actor, err := core.CrearVinculoAutenticacionActorV2ConResultado(ctx, revalidador,
		core.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: confirmacion.AutenticacionRef, SesionRef: confirmacion.SesionRef,
		}, usuarios.base.contextos, core.SolicitudContextoActor{
			Cuenta: core.CuentaAutenticadaContextoActor{
				CuentaRef: cuenta.cuentaRef, Metodo: core.AuthMethodCertificate,
				Garantia: core.AuthAssuranceHigh,
			}, PerfilActivoRef: seleccion.PerfilActivoRef,
		}, usuarios.base.reloj)
	if err != nil {
		if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(err, ctports.ErrConsultaRRHHNoDisponible) ||
			errors.Is(err, vecports.ErrFuenteContextoActorNoDisponible) ||
			errors.Is(err, vecports.ErrResolutorRegistroContextoActorNoDisponible) {
			return vacia, noDisponibleSesionNominalJustificanteRRHH("contexto_actor", err)
		}
		return vacia, docports.ErrSubidaLocalDenegada
	}
	// El sujeto del certificado identifica el alta de sesión; PersonaRef es la
	// identidad canónica del actor. El registro y el vínculo ya ligan ambos a
	// la misma cuenta, sesión y perfil sin exigir igualdad entre esos nombres.
	if actor.Validar() != nil || vinculo.ValidarPara(actor) != nil ||
		actor.Contexto.PerfilActivoRef != seleccion.PerfilActivoRef ||
		actor.Contexto.Instantanea.CuentaRef != cuenta.cuentaRef {
		return vacia, docports.ErrSubidaLocalDenegada
	}
	resultado := contextoSeguridadComunDesarrollo{Vinculo: vinculo, Resultado: actor,
		seleccionJustificanteRRHH: &seleccionNominalJustificanteRRHH{
			perfil: seleccion.PerfilActivoRef, auditoria: seleccion.AuditoriaRef,
			ruta: bolsahttp.RutaJustificanteRenunciaRRHH, metodo: http.MethodGet,
			catalogo: f.catalogo.identidad, holder: c.contextoOperacion,
		}}
	if !perfilJustificanteAdmitidoPorFrontera(f, resultado) {
		return vacia, docports.ErrSubidaLocalDenegada
	}
	return resultado, nil
}

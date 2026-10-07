package adminperfiles

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	h "vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type Proveedor struct {
	config h.ConfiguracionSuperficie
	deps   Dependencias
}

func Nuevo(config h.ConfiguracionSuperficie, deps Dependencias) (*Proveedor, error) {
	if config.Validar() != nil || config.Superficie != h.SuperficieAdministracionPrivilegiada ||
		!config.CertificadoClienteDirecto || !config.RequiereCuentaPrivilegiada ||
		config.PoliticaAdministracion != h.PoliticaAdministracionCertificadoTemporal ||
		nulo(deps.Cuentas) || nulo(deps.Registro) || nulo(deps.Revalidador) ||
		nulo(deps.Contextos) || nulo(deps.Autorizacion) || nulo(deps.Reloj) {
		return nil, api.ErrConfiguracionIncompleta
	}
	if _, ok := deps.Cuentas.(FuenteCuentasADMINConAcuse); !ok {
		return nil, api.ErrConfiguracionIncompleta
	}
	config.RedesPermitidas = append([]string(nil), config.RedesPermitidas...)
	config.HuellasProxyTLSPermitidas = append([]string(nil), config.HuellasProxyTLSPermitidas...)
	config.IdentidadesSANProxyPermitidas = append([]string(nil), config.IdentidadesSANProxyPermitidas...)
	config.MetodosAdmitidos = append([]h.MetodoAutenticacion(nil), config.MetodosAdmitidos...)
	config.FactoresRequeridos = append([]h.MetodoAutenticacion(nil), config.FactoresRequeridos...)
	return &Proveedor{config: config, deps: deps}, nil
}

// Resolver produce una sesión de una única petición con perfil fijo vigente.
// La observación ha de proceder de la frontera F, nunca de campos HTTP.
func (p *Proveedor) Resolver(ctx context.Context, r *http.Request, o ObservacionADMIN) (api.SesionConfiable, error) {
	var vacia api.SesionConfiable
	if p == nil || ctx == nil || ctx.Err() != nil || r == nil || r.TLS == nil ||
		!r.TLS.HandshakeComplete || r.TLS.DidResume || r.Host != o.Autoridad || nombreHostPeticion(o.Autoridad) != o.Host ||
		len(r.TLS.VerifiedChains) != 1 || len(r.TLS.VerifiedChains[0]) != 2 ||
		r.TLS.VerifiedChains[0][0] == nil || r.TLS.VerifiedChains[0][1] == nil ||
		o.Audiencia != p.config.Audiencia || !o.Valida(p.deps.Reloj.Ahora().UTC()) {
		return vacia, api.ErrAutenticacionRequerida
	}
	cert, ca := r.TLS.VerifiedChains[0][0], r.TLS.VerifiedChains[0][1]
	certHash, caHash := sha256.Sum256(cert.Raw), sha256.Sum256(ca.Raw)
	if hex.EncodeToString(certHash[:]) != o.CertificadoSHA256 ||
		hex.EncodeToString(caHash[:]) != o.CASHA256 {
		return vacia, api.ErrAutenticacionRequerida
	}
	ahora := p.deps.Reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !ahora.Before(o.AutenticacionVerificadaEn.Add(p.config.EdadMaximaAutenticacion)) ||
		!ahora.Before(p.config.RetiradaPoliticaAdministracionEn) || ahora.Before(cert.NotBefore) ||
		!ahora.Before(cert.NotAfter) || !o.CertificadoVigenteHasta.Equal(cert.NotAfter) {
		return vacia, api.ErrAutenticacionRequerida
	}
	puente := &asercionInterna{}
	if _, err := rand.Read(puente.clave[:]); err != nil {
		return vacia, api.ErrConfiguracionIncompleta
	}
	identidad, err := h.NuevoServicioIdentidad(p.config, puente, puente, p.deps.Registro, p.deps.Reloj)
	if err != nil {
		return vacia, api.ErrConfiguracionIncompleta
	}
	canal, err := identidad.AutenticarCanalTLSMutuo(*r.TLS)
	if err != nil {
		return vacia, api.ErrAutenticacionRequerida
	}
	cuenta, err := p.deps.Cuentas.ResolverCuentaADMIN(ctx, o)
	if err != nil {
		return vacia, errorAutoridad(err)
	}
	if !cuenta.Valida(ahora) {
		return vacia, api.ErrAccesoDenegado
	}
	hasta := ahora.Add(p.config.DuracionMaximaAsercion)
	for _, limite := range []time.Time{cuenta.VigenteHasta, o.CRLVigenteHasta, o.CertificadoVigenteHasta,
		o.AutenticacionVerificadaEn.Add(p.config.EdadMaximaAutenticacion), p.config.RetiradaPoliticaAdministracionEn} {
		if limite.Before(hasta) {
			hasta = limite
		}
	}
	if !ahora.Before(hasta) {
		return vacia, api.ErrAutenticacionRequerida
	}
	puente.preparar(cuenta, o, p.config, canal.ReferenciaVinculacion(), ahora, hasta)
	credencial, err := h.NuevaCredencialProxy(puente.clave[:], canal)
	if err != nil {
		return vacia, api.ErrConfiguracionIncompleta
	}
	sesion, err := identidad.Resolver(ctx, credencial)
	if err != nil {
		return vacia, api.ErrConfiguracionIncompleta
	}
	resuelta, auditoria, err := identidad.ProyectarCuentaAutenticada(ctx, sesion)
	if err != nil || resuelta.CuentaRef != cuenta.CuentaRef ||
		auditoria.CuentaOrdinariaRef() != cuenta.CuentaOrdinariaRef || !auditoria.CuentaPrivilegiada() ||
		auditoria.Superficie() != h.SuperficieAdministracionPrivilegiada {
		return vacia, api.ErrAccesoDenegado
	}
	conAcuse, ok := p.deps.Cuentas.(FuenteCuentasADMINConAcuse)
	if !ok {
		return vacia, api.ErrConfiguracionIncompleta
	}
	ligadura, err := conAcuse.VincularSesionADMINConAcuse(ctx, o, cuenta, ReferenciasSesionADMIN{
		AutenticacionRef: auditoria.AutenticacionRef(), SesionRef: auditoria.SesionRef(),
	})
	if err != nil {
		return vacia, errorAutoridad(err)
	}
	ctx, err = ContextoConVinculoSesionADMIN(ctx, ligadura)
	if err != nil {
		return vacia, api.ErrConfiguracionIncompleta
	}
	vinculo, resultado, err := domain.CrearVinculoAutenticacionActorV2ConResultado(ctx, p.deps.Revalidador,
		domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: auditoria.AutenticacionRef(), SesionRef: auditoria.SesionRef()},
		p.deps.Contextos, domain.SolicitudContextoActor{Cuenta: resuelta, PerfilActivoRef: cuenta.PerfilActivoRef}, p.deps.Reloj)
	if err != nil {
		return vacia, api.ErrConfiguracionIncompleta
	}
	datos, err := vinculo.Datos()
	if err != nil || !datos.CuentaPrivilegiada || datos.Superficie != domain.SuperficieAutenticacionAdministracionPrivilegiadaV1 ||
		datos.CuentaRef != cuenta.CuentaRef || datos.CuentaOrdinariaRef != cuenta.CuentaOrdinariaRef ||
		datos.PrincipalID != cuenta.PersonaRef || datos.PerfilActivoRef != cuenta.PerfilActivoRef ||
		datos.PoliticaGarantiaRef != cuenta.PoliticaGarantiaRef ||
		datos.PoliticaGarantiaHuellaSHA256 != cuenta.PoliticaGarantiaHuellaSHA256 ||
		!datos.AutenticacionVerificadaEn.Equal(o.AutenticacionVerificadaEn) {
		return vacia, api.ErrAccesoDenegado
	}
	instantanea, err := p.deps.Autorizacion.ObtenerInstantaneaAutorizacion(ctx, cuenta.PersonaRef, cuenta.PerfilActivoRef)
	if err != nil {
		return vacia, errorAutoridad(err)
	}
	final := p.deps.Reloj.Ahora().UTC().Truncate(time.Microsecond)
	if instantanea.Validar() != nil || instantanea.VersionRol.RolID != "administracion_perfiles" ||
		instantanea.VersionRol.Estado != domain.EstadoVersionRolPublicada ||
		instantanea.ControlVigenciaVersionRol.Estado != domain.EstadoControlVigenciaVersionRolHabilitada ||
		instantanea.AsignacionPerfil.PrincipalID != cuenta.PersonaRef ||
		instantanea.AsignacionPerfil.PerfilActivoRef != cuenta.PerfilActivoRef ||
		!instantanea.AsignacionPerfil.VigenteEn(final) || !vinculo.VigenteEn(final, resultado) ||
		!final.Before(hasta) {
		return vacia, api.ErrAccesoDenegado
	}
	actual, err := p.deps.Cuentas.ResolverCuentaADMIN(ctx, o)
	if err != nil {
		return vacia, errorAutoridad(err)
	}
	if actual != cuenta || !actual.Valida(final) {
		return vacia, api.ErrAccesoDenegado
	}
	if _, _, err = identidad.ProyectarCuentaAutenticada(ctx, sesion); err != nil {
		return vacia, api.ErrAccesoDenegado
	}
	correlacion, err := ports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil || ctx.Err() != nil {
		return vacia, api.ErrConfiguracionIncompleta
	}
	ref, err := correlacion.ValorCanonico()
	if err != nil {
		return vacia, api.ErrConfiguracionIncompleta
	}
	return api.SesionConfiable{Actor: resultado.Contexto,
		Evidencia:               domain.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: resultado, Vinculo: vinculo},
		InstantaneaAutorizacion: instantanea, CorrelacionRef: ref}, nil
}

func errorAutoridad(err error) error {
	if errors.Is(err, api.ErrAutenticacionRequerida) {
		return api.ErrAutenticacionRequerida
	}
	if errors.Is(err, api.ErrConflictoEstado) {
		return api.ErrConflictoEstado
	}
	if errors.Is(err, api.ErrConfiguracionIncompleta) {
		return api.ErrConfiguracionIncompleta
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "40001":
			return api.ErrConflictoEstado
		case "42501":
			return api.ErrAccesoDenegado
		}
	}
	if errors.Is(err, api.ErrAccesoDenegado) || errors.Is(err, domain.ErrAutorizacionDenegada) {
		return api.ErrAccesoDenegado
	}
	return api.ErrConfiguracionIncompleta
}

// nombreHostPeticion quita el puerto de la autoridad que la frontera ADMIN
// exigió en la cabecera Host. La observación lleva aparte el nombre (host_admin
// de la política). Una forma inválida devuelve "" y nunca coincide con una
// observación válida.
func nombreHostPeticion(autoridad string) string {
	if strings.ContainsAny(autoridad, "[]") {
		return ""
	}
	nombre, puerto, conPuerto := strings.Cut(autoridad, ":")
	if !conPuerto {
		return autoridad
	}
	if nombre == "" || puerto == "" || strings.Trim(puerto, "0123456789") != "" {
		return ""
	}
	return nombre
}

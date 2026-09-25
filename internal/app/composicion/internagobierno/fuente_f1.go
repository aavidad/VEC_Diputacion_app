// Package internagobierno compone el gobierno ya publicado del portal interno.
// Sus constructores no instalan SQL ni crean identidades o concesiones.
package internagobierno

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	domct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrGobiernoInternoNoDisponible = errors.New("composicion interna: gobierno no disponible")

// ContextoNominal es una selección privada de perfil y ámbito. No contiene
// persona ni permiso: F1 y V3 comprueban cada dato vivo por petición.
type ContextoNominal struct {
	PerfilActivoRef string
	OrganizacionRef string
	UnidadRef       string
}

type ConfiguracionFuenteF1 struct {
	Identidad   *httpseguridad.ServicioIdentidad
	Revalidador vecports.RevalidadorAutenticacionActorV1
	Resolutor   core.ResolutorContextoActorRegistradoV2
	// VinculoCorporativo es ContextoActor 000009: revalida en cada petición el
	// vínculo corporativo interna_corporativa/consulta_rrhh de la persona.
	VinculoCorporativo vecports.RevalidadorVinculoCorporativoRRHHV1
	Reloj              vecports.Reloj
	PorCuenta          map[string]ContextoNominal
	MotivoAlta         core.ReferenciaEntradaCatalogo
	MotivoLectura      core.ReferenciaEntradaCatalogo
	Politica           inc.PoliticaConsultaDesarrollo
}

type FuenteF1 struct {
	identidad     *httpseguridad.ServicioIdentidad
	revalidador   vecports.RevalidadorAutenticacionActorV1
	resolutor     core.ResolutorContextoActorRegistradoV2
	corporativo   vecports.RevalidadorVinculoCorporativoRRHHV1
	reloj         vecports.Reloj
	porCuenta     map[string]ContextoNominal
	motivoAlta    core.ReferenciaEntradaCatalogo
	motivoLectura core.ReferenciaEntradaCatalogo
	politica      inc.PoliticaConsultaDesarrollo
}

// NuevaFuenteF1 no conserva una segunda identidad. Lee solamente la cápsula
// autenticada por ServicioIdentidad y pasa su cuenta a la resolución F1 real.
func NuevaFuenteF1(c ConfiguracionFuenteF1) (*FuenteF1, error) {
	if c.Identidad == nil || c.Revalidador == nil || c.Resolutor == nil || c.VinculoCorporativo == nil ||
		c.Reloj == nil || len(c.PorCuenta) == 0 ||
		c.MotivoAlta.Validar() != nil || c.MotivoLectura.Validar() != nil ||
		!politicaConsultaValida(c.Politica, c.Reloj.Ahora()) {
		return nil, ErrGobiernoInternoNoDisponible
	}
	perfiles := make(map[string]ContextoNominal, len(c.PorCuenta))
	for cuenta, nominal := range c.PorCuenta {
		if (core.CuentaAutenticadaContextoActor{CuentaRef: cuenta, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceSubstantial}).Validar() != nil ||
			(core.SolicitudContextoActor{Cuenta: core.CuentaAutenticadaContextoActor{CuentaRef: cuenta, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceSubstantial}, PerfilActivoRef: nominal.PerfilActivoRef}).Validar() != nil ||
			!domct.ReferenciaOpacaValida(nominal.OrganizacionRef) ||
			!domct.UnidadSeguimientoValida(nominal.UnidadRef) {
			return nil, ErrGobiernoInternoNoDisponible
		}
		perfiles[cuenta] = nominal
	}
	return &FuenteF1{c.Identidad, c.Revalidador, c.Resolutor, c.VinculoCorporativo, c.Reloj, perfiles, c.MotivoAlta, c.MotivoLectura, c.Politica}, nil
}

func politicaConsultaValida(p inc.PoliticaConsultaDesarrollo, ahora time.Time) bool {
	if p.Tipo != httpseguridad.PoliticaInternaDesarrolloCertificadoPersonal ||
		len(p.Referencia) < 26 || len(p.Referencia) > 132 || !strings.HasPrefix(p.Referencia, "pga_") ||
		len(p.HuellaSHA256) != 64 || strings.Trim(p.HuellaSHA256, "0") == "" ||
		p.RetiradaEn.Location() != time.UTC || p.RetiradaEn.Nanosecond() != 0 || !ahora.Before(p.RetiradaEn) {
		return false
	}
	for _, c := range p.Referencia[4:] {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' && c != '-' {
			return false
		}
	}
	for _, c := range p.HuellaSHA256 {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// ResolverContexto devuelve el vínculo de persona y perfil efectivamente
// registrados por F1 para la cápsula de esta petición. Revalida en PostgreSQL
// cada vez; no conserva persona ni autorización en la fuente.
func (f *FuenteF1) ResolverContexto(ctx context.Context) (ct.ContextoAutorizacionAltaV3, error) {
	if f == nil || f.revalidador == nil || f.resolutor == nil || f.reloj == nil {
		return ct.ContextoAutorizacionAltaV3{}, ErrGobiernoInternoNoDisponible
	}
	p, err := f.PeticionVerificada(ctx)
	if err != nil {
		return ct.ContextoAutorizacionAltaV3{}, ErrGobiernoInternoNoDisponible
	}
	v, resultado, err := core.CrearVinculoAutenticacionActorV2ConResultado(ctx, f.revalidador, p.Autenticacion, f.resolutor, p.Contexto, f.reloj)
	if err != nil || ctx.Err() != nil {
		return ct.ContextoAutorizacionAltaV3{}, ErrGobiernoInternoNoDisponible
	}
	datos, err := v.Datos()
	if err != nil || datos.PoliticaGarantiaRef != f.politica.Referencia ||
		datos.PoliticaGarantiaHuellaSHA256 != f.politica.HuellaSHA256 ||
		!f.reloj.Ahora().Before(f.politica.RetiradaEn) {
		return ct.ContextoAutorizacionAltaV3{}, ErrGobiernoInternoNoDisponible
	}
	// La cuenta y el perfil pueden ser coherentes en F1 aunque el certificado
	// pertenezca a otra persona. Identidad coteja el sujeto autenticado con la
	// persona resuelta por F1 antes de entregar el contexto al detalle/V3.
	if err := f.identidad.ExigirSujetoPersonaCertificadoTemporal(ctx, datos.PrincipalID); err != nil {
		return ct.ContextoAutorizacionAltaV3{}, ErrGobiernoInternoNoDisponible
	}
	// El vínculo entregado es el segundo; si su versión cambió tras la
	// revalidación de PeticionVerificada, se revalida el que realmente sale.
	if err := f.exigirVinculoCorporativoVigente(ctx, datos); err != nil {
		return ct.ContextoAutorizacionAltaV3{}, ErrGobiernoInternoNoDisponible
	}
	return ct.ContextoAutorizacionAltaV3{Vinculo: v, Resultado: resultado}, nil
}

func (f *FuenteF1) PeticionVerificada(ctx context.Context) (inc.PeticionAutoridad, error) {
	var vacia inc.PeticionAutoridad
	if f == nil || f.identidad == nil || ctx == nil || ctx.Err() != nil {
		return vacia, ErrGobiernoInternoNoDisponible
	}
	cuenta, auditoria, err := f.identidad.ExtraerCapsulaIdentidadPeticion(ctx)
	if err != nil || cuenta.Validar() != nil || cuenta.Metodo != core.AuthMethodCertificate ||
		cuenta.Garantia != core.AuthAssuranceSubstantial || auditoria.CuentaRef() != cuenta.CuentaRef ||
		auditoria.CuentaPrivilegiada() || auditoria.MetodoObservado() != cuenta.Metodo ||
		auditoria.GarantiaObservada() != cuenta.Garantia ||
		auditoria.Superficie() != httpseguridad.SuperficieInternaCorporativa ||
		auditoria.ControlSesionEstado() != httpseguridad.EstadoControlSesionActiva ||
		auditoria.PoliticaGarantiaRef() != f.politica.Referencia ||
		auditoria.PoliticaGarantiaHuellaSHA256() != f.politica.HuellaSHA256 {
		return vacia, ErrGobiernoInternoNoDisponible
	}
	nominal, ok := f.porCuenta[cuenta.CuentaRef]
	if !ok || auditoria.AutenticacionRef() == "" || auditoria.SesionRef() == "" {
		return vacia, ErrGobiernoInternoNoDisponible
	}
	var nonce [32]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return vacia, ErrGobiernoInternoNoDisponible
	}
	solicitudAutenticacion := core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: auditoria.AutenticacionRef(), SesionRef: auditoria.SesionRef()}
	solicitudContexto := core.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: nominal.PerfilActivoRef}
	vinculo, _, err := core.CrearVinculoAutenticacionActorV2ConResultado(ctx, f.revalidador, solicitudAutenticacion, f.resolutor, solicitudContexto, f.reloj)
	if err != nil || ctx.Err() != nil {
		return vacia, ErrGobiernoInternoNoDisponible
	}
	datos, err := vinculo.Datos()
	if err != nil || !domct.ActorSeguimientoValido(datos.PrincipalID) ||
		datos.PoliticaGarantiaRef != f.politica.Referencia ||
		datos.PoliticaGarantiaHuellaSHA256 != f.politica.HuellaSHA256 ||
		!f.reloj.Ahora().Before(f.politica.RetiradaEn) {
		return vacia, ErrGobiernoInternoNoDisponible
	}
	// Sin este cotejo, un certificado personal A con cuenta F1 de B pasaría
	// las dos validaciones por separado y recibiría autoridad de B.
	if err := f.identidad.ExigirSujetoPersonaCertificadoTemporal(ctx, datos.PrincipalID); err != nil {
		return vacia, ErrGobiernoInternoNoDisponible
	}
	if err := f.exigirVinculoCorporativoVigente(ctx, datos); err != nil {
		return vacia, ErrGobiernoInternoNoDisponible
	}
	resultado := inc.PeticionAutoridad{
		Autenticacion: solicitudAutenticacion,
		Contexto:      solicitudContexto,
		PreparacionCT: ct.PreparacionSeguimientoConfirmacionIncorporacion{OrganizacionRef: nominal.OrganizacionRef, UnidadRef: nominal.UnidadRef, ActorRef: datos.PrincipalID, CorrelacionRef: "ref:" + hex.EncodeToString(nonce[:])},
		MotivoAlta:    f.motivoAlta, MotivoLectura: f.motivoLectura,
	}
	if resultado.Autenticacion.Validar() != nil || resultado.Contexto.Validar() != nil {
		return vacia, ErrGobiernoInternoNoDisponible
	}
	return resultado, nil
}

// exigirVinculoCorporativoVigente pregunta a ContextoActor, en cada petición y
// sin caché, si la cuenta, perfil, persona y vínculo de contexto que F1 acaba de
// acreditar conservan un vínculo corporativo actual, activo y vigente. Revocarlo,
// dejarlo caducar o ligarlo a otra versión corta la siguiente petición; la
// indisponibilidad también deniega. Cuesta una consulta autorizada por petición.
func (f *FuenteF1) exigirVinculoCorporativoVigente(ctx context.Context, datos core.DatosVinculoAutenticacionActorV2) error {
	if f == nil || f.corporativo == nil || ctx == nil || ctx.Err() != nil {
		return ErrGobiernoInternoNoDisponible
	}
	err := f.corporativo.RevalidarVinculoCorporativoRRHHV1(ctx, vecports.SolicitudRevalidacionVinculoCorporativoRRHHV1{
		CuentaRef: datos.CuentaRef, PerfilRef: datos.PerfilActivoRef, PersonaRef: datos.PrincipalID,
		VinculoContextoRef: datos.ContextoActorRef, VinculoContextoVersion: datos.ContextoActorVersion,
	})
	if err != nil || ctx.Err() != nil {
		return ErrGobiernoInternoNoDisponible
	}
	return nil
}

var _ inc.FuentePeticionAutoridad = (*FuenteF1)(nil)

package bootstrap

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sync/atomic"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	vigenciaSesionConsultaRRHHDesarrollo = 2 * time.Minute
	politicaSesionConsultaRRHHDesarrollo = "dev-certificado-mtls-v1;solo-sintetico;canal-privado-validado;garantia-alta-desarrollo;vigencia-120s;sin-kerberos;no-corporativa"
)

// Solo se compone con el soporte mTLS de desarrollo y puertos explícitos.
// La clave efímera protege cápsulas internas, no sustituye HSM/KMS ni una
// autoridad corporativa. PostgreSQL conserva la sesión y su huella, no esta clave.
type proveedorSesionConsultaRRHHDesarrollo struct {
	soporte     *soporteAltaContratacionTemporalDesarrollo
	registro    httpseguridad.RegistroSesiones
	revalidador puertosvec.RevalidadorAutenticacionActorV1
	reloj       ports.Reloj
	resolutor   dominiovec.ResolutorContextoActorRegistradoV2
	base        dominiovec.ResultadoContextoActorRegistradoV2
	clave       [sha256.Size]byte
}

type claveCapsulaSesionConsultaRRHHDesarrollo struct{}

// Tipo interno sin los serializadores redactados del DTO público: el sello
// debe comprometer todos los campos reales, incluidos nonces y vigencias.
type altaCapsulaSesionConsultaRRHHDesarrollo httpseguridad.AltaSesionAtomica

type evidenciaSesionConsultaRRHHDesarrollo struct {
	Esquema           string
	Ruta              string
	CertificadoSHA256 string
	PersonaRef        string
	PerfilRef         string
	Alta              altaCapsulaSesionConsultaRRHHDesarrollo
}

// Campos privados: no hay deserializador ni credencial transportable por HTTP.
// El nonce se liga a un contexto hijo de la petición autenticada.
type capsulaSesionConsultaRRHHDesarrollo struct {
	proveedor *proveedorSesionConsultaRRHHDesarrollo
	nonce     [32]byte
	evidencia evidenciaSesionConsultaRRHHDesarrollo
	mac       [sha256.Size]byte
	consumida atomic.Bool
}

func nuevoProveedorSesionConsultaRRHHDesarrollo(
	soporte *soporteAltaContratacionTemporalDesarrollo,
	registro httpseguridad.RegistroSesiones,
	revalidador puertosvec.RevalidadorAutenticacionActorV1,
	reloj ports.Reloj,
	resolutor dominiovec.ResolutorContextoActorRegistradoV2,
) (*proveedorSesionConsultaRRHHDesarrollo, error) {
	if soporte == nil || soporte.sello == nil || soporte.principalID == "" ||
		!huellaSHA256ValidaContratacionTemporalDesarrollo(soporte.certificadoSHA256) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(registro) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(revalidador) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(reloj) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(resolutor) {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	soporte.mu.Lock()
	base, err := soporte.contexto.Resultado.Clonar()
	soporte.mu.Unlock()
	if err != nil || base.Contexto.Principal.AuthMethod != dominiovec.AuthMethodCertificate ||
		base.Contexto.Principal.AuthAssurance != dominiovec.AuthAssuranceHigh ||
		!domain.InstanteUTCCanonico(reloj.Ahora()) {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	p := &proveedorSesionConsultaRRHHDesarrollo{
		soporte: soporte, registro: registro, revalidador: revalidador,
		reloj: reloj, resolutor: resolutor, base: base,
	}
	if _, err := rand.Read(p.clave[:]); err != nil {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	return p, nil
}

// El holder del llamador conserva resultado y error una sola vez por petición.
// No se cachea aquí otra sesión ni se modifica soporte.contexto.
func (p *proveedorSesionConsultaRRHHDesarrollo) ResolverContexto(
	ctx context.Context,
) (ports.ContextoAutorizacionAltaV3, error) {
	vacio := ports.ContextoAutorizacionAltaV3{}
	ctxCapsula, capsula, err := p.acreditarPeticion(ctx)
	if err != nil {
		return vacio, err
	}
	alta, confirmacion, err := p.registrarCapsula(ctxCapsula, capsula)
	if err != nil {
		return vacio, err
	}
	// El decorador invoca el puerto nominal real y coteja todos sus datos con
	// el alta confirmada; nunca devuelve la autenticación histórica del soporte.
	revalidador := revalidadorSesionConsultaRRHHDesarrollo{
		delegado: p.revalidador, alta: alta, confirmacion: confirmacion, reloj: p.reloj,
	}
	vinculo, resultado, err := dominiovec.CrearVinculoAutenticacionActorV2ConResultado(
		ctxCapsula, revalidador,
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: confirmacion.AutenticacionRef, SesionRef: confirmacion.SesionRef,
		},
		p.resolutor,
		dominiovec.SolicitudContextoActor{
			Cuenta: dominiovec.CuentaAutenticadaContextoActor{
				CuentaRef: p.base.Contexto.Instantanea.CuentaRef,
				Metodo:    dominiovec.AuthMethodCertificate, Garantia: dominiovec.AuthAssuranceHigh,
			},
			PerfilActivoRef: p.base.Contexto.PerfilActivoRef,
		},
		p.reloj,
	)
	if err != nil || !mismaIdentidadVersionadaSesionDesarrollo(p.base, resultado) {
		return vacio, ports.ErrConsultaRRHHNoDisponible
	}
	contexto := ports.ContextoAutorizacionAltaV3{Vinculo: vinculo, Resultado: resultado}
	if contexto.ValidarPara(ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: confirmacion.AutenticacionRef, SesionRef: confirmacion.SesionRef,
		PerfilRef: p.base.Contexto.PerfilActivoRef,
	}, p.reloj.Ahora()) != nil || ctxCapsula.Err() != nil {
		return vacio, ports.ErrConsultaRRHHNoDisponible
	}
	return contexto, nil
}

func (p *proveedorSesionConsultaRRHHDesarrollo) acreditarPeticion(
	ctx context.Context,
) (context.Context, *capsulaSesionConsultaRRHHDesarrollo, error) {
	if p == nil || p.soporte == nil || dependenciaEsNulaContratacionTemporalDesarrollo(p.reloj) {
		return nil, nil, ports.ErrConsultaRRHHNoDisponible
	}
	canal, valido := p.soporte.capacidadValida(ctx)
	if !valido || !rutaConsultaRRHHContratacionTemporalDesarrollo(canal.ruta) {
		return nil, nil, ports.ErrAutorizacionDenegada
	}
	ahora := p.reloj.Ahora()
	if !domain.InstanteUTCCanonico(ahora) || !p.base.Contexto.Instantanea.VigenteEn(ahora) {
		return nil, nil, ports.ErrConsultaRRHHNoDisponible
	}
	if !domain.InstanteUTCCanonico(canal.certificadoVerificadoEn) ||
		!domain.InstanteUTCCanonico(canal.certificadoValidoHasta) ||
		canal.certificadoVerificadoEn.Nanosecond()%1000 != 0 ||
		canal.certificadoValidoHasta.Nanosecond()%1000 != 0 ||
		canal.certificadoVerificadoEn.After(ahora) || !ahora.Before(canal.certificadoValidoHasta) {
		return nil, nil, ports.ErrAutorizacionDenegada
	}
	expiraEn := ahora.Add(vigenciaSesionConsultaRRHHDesarrollo)
	if canal.certificadoValidoHasta.Before(expiraEn) {
		expiraEn = canal.certificadoValidoHasta
	}
	for _, vinculo := range p.base.Contexto.Instantanea.Vinculos {
		if !vinculo.VigenteEn(ahora) {
			return nil, nil, ports.ErrConsultaRRHHNoDisponible
		}
	}
	c := &capsulaSesionConsultaRRHHDesarrollo{proveedor: p}
	var nonceSesion [32]byte
	if _, err := rand.Read(c.nonce[:]); err != nil {
		return nil, nil, ports.ErrConsultaRRHHNoDisponible
	}
	if _, err := rand.Read(nonceSesion[:]); err != nil || c.nonce == nonceSesion {
		return nil, nil, ports.ErrConsultaRRHHNoDisponible
	}
	// Esta política explícita solo acredita el certificado sintético admitido
	// por capacidadValida. No declara Kerberos ni cumplimiento corporativo.
	politica := sha256.Sum256([]byte(politicaSesionConsultaRRHHDesarrollo))
	c.evidencia = evidenciaSesionConsultaRRHHDesarrollo{
		Esquema: "vec.identidad.desarrollo.capsula-mtls.v1",
		Ruta:    canal.ruta, CertificadoSHA256: p.soporte.certificadoSHA256,
		PersonaRef: p.base.Contexto.PersonaRef, PerfilRef: p.base.Contexto.PerfilActivoRef,
		Alta: altaCapsulaSesionConsultaRRHHDesarrollo{
			AsercionID: hex.EncodeToString(c.nonce[:]), SesionID: hex.EncodeToString(nonceSesion[:]),
			SujetoID: p.soporte.principalID, CuentaID: "desarrollo:" + p.base.Contexto.Instantanea.CuentaRef,
			// Clase de ruta interna, no afirmación de identidad corporativa.
			Superficie:       httpseguridad.SuperficieInternaCorporativa,
			EspacioIdentidad: espacioIdentidadSesionDesarrollo,
			MetodoObservado:  dominiovec.AuthMethodCertificate, GarantiaObservada: dominiovec.AuthAssuranceHigh,
			AutenticacionVerificadaEn: canal.certificadoVerificadoEn, SesionEmitidaEn: ahora,
			AsercionExpiraEn:             expiraEn,
			PoliticaGarantiaRef:          referenciaAltaContratacionTemporalDesarrollo("pga_", "dev-certificado-mtls-v1"),
			PoliticaGarantiaHuellaSHA256: hex.EncodeToString(politica[:]),
		},
	}
	canon, err := json.Marshal(c.evidencia)
	if err != nil {
		return nil, nil, ports.ErrConsultaRRHHNoDisponible
	}
	defer borrarBytes(canon)
	mac := hmac.New(sha256.New, p.clave[:])
	_, _ = mac.Write(canon)
	copy(c.mac[:], mac.Sum(nil))
	return context.WithValue(ctx, claveCapsulaSesionConsultaRRHHDesarrollo{}, c.nonce), c, nil
}

func (p *proveedorSesionConsultaRRHHDesarrollo) registrarCapsula(
	ctx context.Context, c *capsulaSesionConsultaRRHHDesarrollo,
) (httpseguridad.AltaSesionAtomica, httpseguridad.ConfirmacionAltaSesion, error) {
	fallo := func() (httpseguridad.AltaSesionAtomica, httpseguridad.ConfirmacionAltaSesion, error) {
		return httpseguridad.AltaSesionAtomica{}, httpseguridad.ConfirmacionAltaSesion{}, ports.ErrAutorizacionDenegada
	}
	if p == nil || c == nil || c.proveedor != p || contextoInterfazNulo(ctx) || ctx.Err() != nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.registro) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.reloj) {
		return fallo()
	}
	nonce, presente := ctx.Value(claveCapsulaSesionConsultaRRHHDesarrollo{}).([32]byte)
	canal, valido := p.soporte.capacidadValida(ctx)
	ahora := p.reloj.Ahora()
	if !presente || nonce != c.nonce || !valido || canal.ruta != c.evidencia.Ruta ||
		canal.principal.ID != c.evidencia.Alta.SujetoID ||
		canal.principal.Attributes["certificate_sha256"] != c.evidencia.CertificadoSHA256 ||
		!domain.InstanteUTCCanonico(ahora) || ahora.Before(c.evidencia.Alta.SesionEmitidaEn) ||
		!domain.InstanteUTCCanonico(canal.certificadoVerificadoEn) ||
		!domain.InstanteUTCCanonico(canal.certificadoValidoHasta) ||
		canal.certificadoVerificadoEn.Nanosecond()%1000 != 0 ||
		canal.certificadoValidoHasta.Nanosecond()%1000 != 0 ||
		canal.certificadoVerificadoEn.After(ahora) || !ahora.Before(canal.certificadoValidoHasta) ||
		!c.evidencia.Alta.AutenticacionVerificadaEn.Equal(canal.certificadoVerificadoEn) ||
		c.evidencia.Alta.AsercionExpiraEn.After(canal.certificadoValidoHasta) ||
		c.evidencia.Alta.AsercionExpiraEn.After(c.evidencia.Alta.SesionEmitidaEn.Add(vigenciaSesionConsultaRRHHDesarrollo)) ||
		!ahora.Before(c.evidencia.Alta.AsercionExpiraEn) {
		return fallo()
	}
	canon, err := json.Marshal(c.evidencia)
	if err != nil {
		return fallo()
	}
	defer borrarBytes(canon)
	mac := hmac.New(sha256.New, p.clave[:])
	_, _ = mac.Write(canon)
	if !hmac.Equal(c.mac[:], mac.Sum(nil)) {
		return fallo()
	}
	huella := sha256.New()
	_, _ = huella.Write(canon)
	_, _ = huella.Write(c.mac[:])
	alta := httpseguridad.AltaSesionAtomica(c.evidencia.Alta)
	alta.AutenticacionHuellaSHA256 = hex.EncodeToString(huella.Sum(nil))
	if alta.Validar() != nil || !c.consumida.CompareAndSwap(false, true) {
		return fallo()
	}
	confirmacion, err := p.registro.ConsumirAsercionYRegistrar(ctx, alta)
	ahora = p.reloj.Ahora()
	if err != nil || ctx.Err() != nil || confirmacion.ValidarPara(alta) != nil ||
		confirmacion.CuentaRef != p.base.Contexto.Instantanea.CuentaRef ||
		!domain.InstanteUTCCanonico(ahora) || ahora.Before(confirmacion.SesionRevalidadaEn) ||
		!ahora.Before(confirmacion.SesionValidaHasta) {
		return httpseguridad.AltaSesionAtomica{}, httpseguridad.ConfirmacionAltaSesion{}, ports.ErrConsultaRRHHNoDisponible
	}
	return alta, confirmacion, nil
}

type revalidadorSesionConsultaRRHHDesarrollo struct {
	delegado     puertosvec.RevalidadorAutenticacionActorV1
	alta         httpseguridad.AltaSesionAtomica
	confirmacion httpseguridad.ConfirmacionAltaSesion
	reloj        ports.Reloj
}

func (r revalidadorSesionConsultaRRHHDesarrollo) RevalidarAutenticacionActorV1(
	ctx context.Context, solicitud dominiovec.SolicitudRevalidacionAutenticacionActorV1,
) (dominiovec.AutenticacionRevalidadaV1, error) {
	vacio := dominiovec.AutenticacionRevalidadaV1{}
	c, a := r.confirmacion, r.alta
	if dependenciaEsNulaContratacionTemporalDesarrollo(r.delegado) ||
		solicitud.AutenticacionRef != c.AutenticacionRef || solicitud.SesionRef != c.SesionRef {
		return vacio, ports.ErrConsultaRRHHNoDisponible
	}
	obtenida, err := r.delegado.RevalidarAutenticacionActorV1(ctx, solicitud)
	ahora := r.reloj.Ahora()
	if err != nil || obtenida.Validar() != nil || !domain.InstanteUTCCanonico(ahora) ||
		obtenida.SesionRevalidadaEn.Before(c.SesionRevalidadaEn) ||
		ahora.Before(obtenida.SesionRevalidadaEn) || !ahora.Before(obtenida.SesionValidaHasta) {
		return vacio, ports.ErrConsultaRRHHNoDisponible
	}
	esperada := dominiovec.AutenticacionRevalidadaV1{
		AutenticacionRef: c.AutenticacionRef, AutenticacionHuellaSHA256: a.AutenticacionHuellaSHA256,
		AsercionRef: c.AsercionRef, SesionRef: c.SesionRef, ControlSesionRef: c.ControlSesionRef,
		ControlSesionRevision: c.ControlSesionRevision, ControlSesionHuellaSHA256: c.ControlSesionHuellaSHA256,
		CuentaRef: c.CuentaRef, CuentaOrdinariaRef: c.CuentaOrdinariaRef,
		CuentaPrivilegiada: a.CuentaPrivilegiada, Superficie: dominiovec.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado: a.MetodoObservado, GarantiaObservada: a.GarantiaObservada,
		PoliticaGarantiaRef: a.PoliticaGarantiaRef, PoliticaGarantiaHuellaSHA256: a.PoliticaGarantiaHuellaSHA256,
		AutenticacionVerificadaEn: a.AutenticacionVerificadaEn, SesionEmitidaEn: a.SesionEmitidaEn,
		SesionValidaHasta: c.SesionValidaHasta, SesionRevalidadaEn: obtenida.SesionRevalidadaEn,
	}
	if obtenida != esperada {
		return vacio, ports.ErrConsultaRRHHNoDisponible
	}
	return obtenida, nil
}

func mismaIdentidadVersionadaSesionDesarrollo(
	base, actual dominiovec.ResultadoContextoActorRegistradoV2,
) bool {
	b, a := base.Contexto.Instantanea, actual.Contexto.Instantanea
	// El recibo y su ResueltoEn/huella pueden ser nuevos. Las coordenadas y
	// versiones de la identidad base no se sustituyen por otras al resolver.
	return actual.Validar() == nil && b.CuentaRef == a.CuentaRef && b.CuentaVersion == a.CuentaVersion &&
		b.PersonaRef == a.PersonaRef && b.PersonaVersion == a.PersonaVersion &&
		b.PerfilActivoRef == a.PerfilActivoRef && b.PerfilVersion == a.PerfilVersion &&
		b.VinculoRef == a.VinculoRef && b.VinculoVersion == a.VinculoVersion &&
		reflect.DeepEqual(b.Vinculos, a.Vinculos)
}

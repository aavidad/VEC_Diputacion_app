package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// Dobles explícitos de registro/revalidación/resolución. No acreditan SQL,
// autenticación corporativa ni la ejecución del recorrido en una base real.
type relojSesionConsultaPrueba struct{ ahora time.Time }

func (r *relojSesionConsultaPrueba) Ahora() time.Time { return r.ahora }

type registroSesionConsultaPrueba struct {
	reloj        *relojSesionConsultaPrueba
	cuenta       string
	altas        []httpseguridad.AltaSesionAtomica
	confirmacion httpseguridad.ConfirmacionAltaSesion
	despues      func()
}

func (r *registroSesionConsultaPrueba) ConsumirAsercionYRegistrar(
	_ context.Context, alta httpseguridad.AltaSesionAtomica,
) (httpseguridad.ConfirmacionAltaSesion, error) {
	r.altas = append(r.altas, alta)
	base := fmt.Sprint(len(r.altas))
	r.confirmacion = httpseguridad.ConfirmacionAltaSesion{
		AutenticacionRef:      referenciaAltaContratacionTemporalDesarrollo("aut_", "sesion-prueba"+base),
		AsercionRef:           referenciaAltaContratacionTemporalDesarrollo("ase_", alta.AsercionID),
		SesionRef:             referenciaAltaContratacionTemporalDesarrollo("ses_", alta.SesionID),
		ControlSesionRef:      referenciaAltaContratacionTemporalDesarrollo("cse_", "control-prueba"+base),
		ControlSesionRevision: 1, ControlSesionEstado: httpseguridad.EstadoControlSesionActiva,
		ControlSesionHuellaSHA256: strings.Repeat("b", 64),
		CuentaRef:                 r.cuenta, CuentaOrdinariaRef: r.cuenta,
		SesionRevalidadaEn: r.reloj.Ahora(), SesionValidaHasta: alta.AsercionExpiraEn,
		AltaConfirmada: alta,
	}
	if r.despues != nil {
		r.despues()
	}
	return r.confirmacion, nil
}

func (*registroSesionConsultaPrueba) ComprobarSesionYCuentaActivas(
	context.Context, httpseguridad.ConsultaSesionActiva,
) error {
	return errors.New("la prueba exige el revalidador nominal explícito")
}

type revalidadorSesionConsultaPrueba struct {
	registro *registroSesionConsultaPrueba
	llamadas int
	alterar  func(*dominiovec.AutenticacionRevalidadaV1)
	err      error
}

func (r *revalidadorSesionConsultaPrueba) RevalidarAutenticacionActorV1(
	_ context.Context, solicitud dominiovec.SolicitudRevalidacionAutenticacionActorV1,
) (dominiovec.AutenticacionRevalidadaV1, error) {
	r.llamadas++
	c := r.registro.confirmacion
	a := c.AltaConfirmada
	if r.err != nil || solicitud.AutenticacionRef != c.AutenticacionRef || solicitud.SesionRef != c.SesionRef {
		return dominiovec.AutenticacionRevalidadaV1{}, errors.New("revalidación denegada por el doble")
	}
	resultado := dominiovec.AutenticacionRevalidadaV1{
		AutenticacionRef: c.AutenticacionRef, AutenticacionHuellaSHA256: a.AutenticacionHuellaSHA256,
		AsercionRef: c.AsercionRef, SesionRef: c.SesionRef,
		ControlSesionRef: c.ControlSesionRef, ControlSesionRevision: c.ControlSesionRevision,
		ControlSesionHuellaSHA256: c.ControlSesionHuellaSHA256,
		CuentaRef:                 c.CuentaRef, CuentaOrdinariaRef: c.CuentaOrdinariaRef,
		Superficie:      dominiovec.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado: a.MetodoObservado, GarantiaObservada: a.GarantiaObservada,
		PoliticaGarantiaRef: a.PoliticaGarantiaRef, PoliticaGarantiaHuellaSHA256: a.PoliticaGarantiaHuellaSHA256,
		AutenticacionVerificadaEn: a.AutenticacionVerificadaEn, SesionEmitidaEn: a.SesionEmitidaEn,
		SesionRevalidadaEn: r.registro.reloj.Ahora(), SesionValidaHasta: c.SesionValidaHasta,
	}
	if r.alterar != nil {
		r.alterar(&resultado)
	}
	return resultado, nil
}

type resolutorSesionConsultaPrueba struct {
	base      dominiovec.ResultadoContextoActorRegistradoV2
	reloj     *relojSesionConsultaPrueba
	llamadas  int
	historico bool
}

func (r *resolutorSesionConsultaPrueba) ResolverContextoActorRegistradoV2(
	_ context.Context, solicitud dominiovec.SolicitudContextoActor,
) (dominiovec.ResultadoContextoActorRegistradoV2, error) {
	r.llamadas++
	resultado, err := r.base.Clonar()
	if err != nil || r.historico {
		return resultado, err
	}
	actor, err := dominiovec.NuevoContextoActor(solicitud.Cuenta, resultado.Contexto.Instantanea, r.reloj.Ahora())
	if err != nil {
		return dominiovec.ResultadoContextoActorRegistradoV2{}, err
	}
	resultado.Contexto = actor
	resultado.ResueltoEnAutoritativo = actor.ResueltoEn
	resultado.RegistroContextoRef = referenciaAltaContratacionTemporalDesarrollo("rca_", fmt.Sprint("contexto-nuevo-prueba", r.llamadas))
	resultado.RepresentacionCanonica, err = actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		return dominiovec.ResultadoContextoActorRegistradoV2{}, err
	}
	resultado.HuellaSHA256, err = actor.HuellaSHA256VinculadaV2()
	return resultado, err
}

type entornoSesionConsultaPrueba struct {
	p           *proveedorSesionConsultaRRHHDesarrollo
	soporte     *soporteAltaContratacionTemporalDesarrollo
	principal   dominiovec.Principal
	reloj       *relojSesionConsultaPrueba
	registro    *registroSesionConsultaPrueba
	revalidador *revalidadorSesionConsultaPrueba
	resolutor   *resolutorSesionConsultaPrueba
}

func nuevaSesionConsultaPrueba(t *testing.T) *entornoSesionConsultaPrueba {
	t.Helper()
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	reloj := &relojSesionConsultaPrueba{ahora: time.Now().UTC().Truncate(time.Microsecond)}
	registro := &registroSesionConsultaPrueba{reloj: reloj, cuenta: soporte.contexto.Resultado.Contexto.Instantanea.CuentaRef}
	revalidador := &revalidadorSesionConsultaPrueba{registro: registro}
	resolutor := &resolutorSesionConsultaPrueba{base: soporte.contexto.Resultado, reloj: reloj}
	p, err := nuevoProveedorSesionConsultaRRHHDesarrollo(soporte, registro, revalidador, reloj, resolutor)
	if err != nil {
		t.Fatal(err)
	}
	return &entornoSesionConsultaPrueba{p, soporte, principal, reloj, registro, revalidador, resolutor}
}

func (e *entornoSesionConsultaPrueba) contexto() context.Context {
	ctx := contextoRutaCoberturaDesarrolloPrueba(e.soporte, e.principal, httpinterno.RutaConsultaCuadroRRHH)
	canal := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	canal.certificadoVerificadoEn = e.reloj.Ahora().Add(-time.Second)
	canal.certificadoValidoHasta = e.reloj.Ahora().Add(5 * time.Minute)
	return context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, canal)
}

func TestSesionConsultaRRHHDesarrolloRegistroNuevoConIdentidadConservada(t *testing.T) {
	e := nuevaSesionConsultaPrueba(t)
	anterior, err := e.soporte.contexto.Resultado.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	primero, err := e.p.ResolverContexto(e.contexto())
	if err != nil {
		t.Fatal(err)
	}
	datos, err := primero.Vinculo.Datos()
	if err != nil || primero.Resultado.RegistroContextoRef == anterior.RegistroContextoRef ||
		primero.Resultado.HuellaSHA256 == anterior.HuellaSHA256 ||
		!primero.Resultado.ResueltoEnAutoritativo.Equal(e.reloj.Ahora()) ||
		datos.CuentaRef != anterior.Contexto.Instantanea.CuentaRef ||
		datos.PrincipalID != anterior.Contexto.PersonaRef || datos.PerfilActivoRef != anterior.Contexto.PerfilActivoRef ||
		datos.MetodoObservado != dominiovec.AuthMethodCertificate || !reflect.DeepEqual(anterior, e.soporte.contexto.Resultado) {
		t.Fatal("el contexto no es nuevo, ha cambiado de identidad o ha alterado el recibo anterior")
	}
	alta := e.registro.altas[0]
	if alta.EspacioIdentidad != espacioIdentidadSesionDesarrollo ||
		alta.CuentaID != "desarrollo:"+datos.CuentaRef || alta.SujetoID != e.soporte.principalID ||
		alta.CuentaOrdinariaID != "" || alta.CuentaPrivilegiada ||
		alta.AsercionID == alta.SesionID || alta.AsercionExpiraEn.Sub(alta.SesionEmitidaEn) != 2*time.Minute ||
		alta.AutenticacionVerificadaEn != e.reloj.Ahora().Add(-time.Second) ||
		alta.PoliticaGarantiaRef != referenciaAltaContratacionTemporalDesarrollo("pga_", "dev-certificado-mtls-v1") {
		t.Fatal("el alta no conserva el contrato breve y sintético pactado")
	}
	segundo, err := e.p.ResolverContexto(e.contexto())
	if err != nil || segundo.Vinculo.CoincideExactamenteCon(primero.Vinculo) ||
		e.registro.altas[1].AutenticacionHuellaSHA256 == alta.AutenticacionHuellaSHA256 ||
		e.registro.altas[1].AsercionID == alta.AsercionID || e.registro.altas[1].SesionID == alta.SesionID ||
		e.revalidador.llamadas != 2 || e.resolutor.llamadas != 2 {
		t.Fatal("se ha compartido una sesión/cápsula entre peticiones o eludido una autoridad")
	}
}

func TestSesionConsultaRRHHDesarrolloCapsulaLigadaYDeUnSoloUso(t *testing.T) {
	e := nuevaSesionConsultaPrueba(t)
	ctx, capsula, err := e.p.acreditarPeticion(e.contexto())
	if err != nil {
		t.Fatal(err)
	}
	serializado, err := json.Marshal(capsula)
	if err != nil || string(serializado) != "{}" {
		t.Fatal("la cápsula privada expone material al serializar")
	}
	if _, _, err := e.p.registrarCapsula(e.contexto(), capsula); err == nil {
		t.Fatal("la cápsula se aceptó en otra petición")
	}
	otro := nuevaSesionConsultaPrueba(t)
	if _, _, err := otro.p.registrarCapsula(ctx, capsula); err == nil {
		t.Fatal("otro proveedor aceptó la cápsula")
	}
	capsula.evidencia.Alta.SesionID += "x"
	if _, _, err := e.p.registrarCapsula(ctx, capsula); err == nil || len(e.registro.altas) != 0 {
		t.Fatal("evidencia adulterada llegó al registro")
	}
	capsula.evidencia.Alta.SesionID = strings.TrimSuffix(capsula.evidencia.Alta.SesionID, "x")
	if _, _, err := e.p.registrarCapsula(ctx, capsula); err != nil {
		t.Fatal(err)
	}
	if _, _, err := e.p.registrarCapsula(ctx, capsula); err == nil || len(e.registro.altas) != 1 {
		t.Fatal("se consumió dos veces la cápsula")
	}
}

func TestSesionConsultaRRHHDesarrolloRechazaSinCanal(t *testing.T) {
	e := nuevaSesionConsultaPrueba(t)
	cancelado, cancelar := context.WithCancel(e.contexto())
	cancelar()
	otroCertificado := clonarPrincipalDesarrollo(e.principal)
	otroCertificado.Attributes["certificate_sha256"] = strings.Repeat("e", 64)
	for _, ctx := range []context.Context{nil, context.Background(), cancelado,
		contextoRutaCoberturaDesarrolloPrueba(e.soporte, otroCertificado, httpinterno.RutaConsultaCuadroRRHH),
		contextoRutaCoberturaDesarrolloPrueba(e.soporte, e.principal, httpinterno.RutaAltaSolicitudes)} {
		if _, err := e.p.ResolverContexto(ctx); err == nil {
			t.Fatal("se admitió una petición sin el canal exacto de consultas")
		}
	}
	if len(e.registro.altas) != 0 || e.revalidador.llamadas != 0 || e.resolutor.llamadas != 0 {
		t.Fatal("un rechazo de canal alcanzó los puertos")
	}
}

func TestSesionConsultaRRHHDesarrolloRechazaCuentaIncorrecta(t *testing.T) {
	e := nuevaSesionConsultaPrueba(t)
	e.registro.cuenta = referenciaAltaContratacionTemporalDesarrollo("cta_", "otra-cuenta")
	if _, err := e.p.ResolverContexto(e.contexto()); err == nil ||
		e.revalidador.llamadas != 0 || e.resolutor.llamadas != 0 {
		t.Fatal("una cuenta distinta del recibo alcanzó la revalidación/resolución")
	}
}

func TestSesionConsultaRRHHDesarrolloCaducidadAntesYDespuesDelRegistro(t *testing.T) {
	e := nuevaSesionConsultaPrueba(t)
	ctx, capsula, err := e.p.acreditarPeticion(e.contexto())
	if err != nil {
		t.Fatal(err)
	}
	e.reloj.ahora = e.reloj.ahora.Add(2 * time.Minute)
	if _, _, err := e.p.registrarCapsula(ctx, capsula); err == nil || len(e.registro.altas) != 0 {
		t.Fatal("se registró una cápsula caducada")
	}
	e = nuevaSesionConsultaPrueba(t)
	e.registro.despues = func() { e.reloj.ahora = e.reloj.ahora.Add(2 * time.Minute) }
	if _, err := e.p.ResolverContexto(e.contexto()); err == nil || e.revalidador.llamadas != 0 {
		t.Fatal("se entregó una sesión caducada durante el registro")
	}
}

func TestSesionConsultaRRHHDesarrolloLimitaSesionAlCertificado(t *testing.T) {
	e := nuevaSesionConsultaPrueba(t)
	ctx := e.contexto()
	canal := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	canal.certificadoVerificadoEn = e.reloj.Ahora().Add(-3 * time.Second)
	canal.certificadoValidoHasta = e.reloj.Ahora().Add(30 * time.Second)
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, canal)
	if _, err := e.p.ResolverContexto(ctx); err != nil {
		t.Fatal(err)
	}
	alta := e.registro.altas[0]
	if alta.AutenticacionVerificadaEn != canal.certificadoVerificadoEn ||
		alta.SesionEmitidaEn != e.reloj.Ahora() || alta.AsercionExpiraEn != canal.certificadoValidoHasta {
		t.Fatal("se sustituyó el instante del canal o se amplió la vigencia del certificado")
	}
	ctxCapsula, capsula, err := e.p.acreditarPeticion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, tiempos := range [][2]time.Time{
		{},
		{e.reloj.Ahora().Add(time.Second), canal.certificadoValidoHasta},
		{canal.certificadoVerificadoEn, e.reloj.Ahora()},
	} {
		alterado := canal
		alterado.certificadoVerificadoEn, alterado.certificadoValidoHasta = tiempos[0], tiempos[1]
		ctxAlterado := context.WithValue(ctxCapsula, claveCapacidadConsultasContratacionTemporalDesarrollo{}, alterado)
		if _, _, err := e.p.acreditarPeticion(ctxAlterado); err == nil {
			t.Fatal("se acreditó un canal sin fechas válidas")
		}
		if _, _, err := e.p.registrarCapsula(ctxAlterado, capsula); err == nil {
			t.Fatal("el registro no recotejó las fechas del canal")
		}
	}
	e.reloj.ahora = canal.certificadoValidoHasta
	if _, _, err := e.p.registrarCapsula(ctxCapsula, capsula); err == nil || len(e.registro.altas) != 1 {
		t.Fatal("se registró otra sesión al caducar el certificado")
	}
	if _, err := e.p.ResolverContexto(ctx); err == nil || len(e.registro.altas) != 1 {
		t.Fatal("se emitió una sesión nueva desde un certificado vencido")
	}
}

func TestSesionConsultaRRHHDesarrolloExigeAutoridadesNominalesYContextoActual(t *testing.T) {
	t.Run("revalidacion_denegada", func(t *testing.T) {
		e := nuevaSesionConsultaPrueba(t)
		e.revalidador.err = errors.New("sesión revocada")
		if _, err := e.p.ResolverContexto(e.contexto()); err == nil || e.resolutor.llamadas != 0 {
			t.Fatal("se ignoró una denegación nominal")
		}
	})
	t.Run("revalidacion_de_otra_asercion", func(t *testing.T) {
		e := nuevaSesionConsultaPrueba(t)
		e.revalidador.alterar = func(a *dominiovec.AutenticacionRevalidadaV1) {
			a.AutenticacionHuellaSHA256 = strings.Repeat("f", 64)
		}
		if _, err := e.p.ResolverContexto(e.contexto()); err == nil || e.resolutor.llamadas != 0 {
			t.Fatal("se aceptó una revalidación ajena al recibo")
		}
	})
	t.Run("resultado_historico", func(t *testing.T) {
		e := nuevaSesionConsultaPrueba(t)
		e.resolutor.historico = true
		if _, err := e.p.ResolverContexto(e.contexto()); err == nil || e.resolutor.llamadas != 1 {
			t.Fatal("se rejuveneció un resultado histórico en lugar de resolver otro")
		}
	})
}

func TestSesionConsultaRRHHDesarrolloExigeDependenciasYVersiones(t *testing.T) {
	e := nuevaSesionConsultaPrueba(t)
	_, errRegistro := nuevoProveedorSesionConsultaRRHHDesarrollo(e.soporte, nil, e.revalidador, e.reloj, e.resolutor)
	_, errRevalidador := nuevoProveedorSesionConsultaRRHHDesarrollo(e.soporte, e.registro, nil, e.reloj, e.resolutor)
	_, errResolutor := nuevoProveedorSesionConsultaRRHHDesarrollo(e.soporte, e.registro, e.revalidador, e.reloj, nil)
	var relojNulo *relojSesionConsultaPrueba
	_, errReloj := nuevoProveedorSesionConsultaRRHHDesarrollo(e.soporte, e.registro, e.revalidador, relojNulo, e.resolutor)
	if errRegistro == nil || errRevalidador == nil || errResolutor == nil || errReloj == nil {
		t.Fatal("se permitió un puerto implícito o nulo")
	}
	cambiado, err := e.soporte.contexto.Resultado.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	cambiado.Contexto.Instantanea.CuentaVersion++
	if mismaIdentidadVersionadaSesionDesarrollo(e.soporte.contexto.Resultado, cambiado) {
		t.Fatal("se aceptó otra versión de identidad")
	}
}

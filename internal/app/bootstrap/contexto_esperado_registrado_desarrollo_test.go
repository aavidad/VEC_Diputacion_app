package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type resolutorContextoEsperadoPrueba func(context.Context, dominiovec.SolicitudContextoActor) (dominiovec.ResultadoContextoActorRegistradoV2, error)

func (f resolutorContextoEsperadoPrueba) ResolverContextoActorRegistradoV2(
	ctx context.Context, solicitud dominiovec.SolicitudContextoActor,
) (dominiovec.ResultadoContextoActorRegistradoV2, error) {
	return f(ctx, solicitud)
}

func TestContextoEsperadoRegistradoFallaCerradoSinDatosEnLog(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	const secreto = "secreto-certificado-dsn-persona"
	invalidado, err := soporte.contexto.Resultado.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	invalidado.HuellaSHA256 = secreto
	semillaInvalida := invalidado
	perfilCT130, err := nuevoContextoReincorporacionTitularDesarrollo(soporte, soporte.reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	if err := perfilCT130.Resultado.Validar(); err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre, etapa string
		resultado     dominiovec.ResultadoContextoActorRegistradoV2
		semilla       *dominiovec.ResultadoContextoActorRegistradoV2
		selloInvalido bool
		err           error
	}{
		{nombre: "semilla_invalida", etapa: "validar_semilla_contexto", semilla: &semillaInvalida},
		{nombre: "sello_soporte_invalido", etapa: "validar_soporte_contexto", selloInvalido: true},
		{nombre: "resolutor", etapa: "resolver_contexto_registrado", err: errors.New(secreto)},
		{nombre: "resultado_invalido", etapa: "validar_contexto_registrado", resultado: invalidado},
		{nombre: "perfil_ct130_distinto", etapa: "identidad_contexto_registrado", resultado: perfilCT130.Resultado},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			var salida bytes.Buffer
			anterior := log.Writer()
			log.SetOutput(&salida)
			t.Cleanup(func() { log.SetOutput(anterior) })
			resolutor := resolutorContextoEsperadoPrueba(func(context.Context, dominiovec.SolicitudContextoActor) (dominiovec.ResultadoContextoActorRegistradoV2, error) {
				return caso.resultado, caso.err
			})
			if caso.selloInvalido {
				anterior := soporte.certificadoSHA256
				soporte.certificadoSHA256 = secreto
				t.Cleanup(func() { soporte.certificadoSHA256 = anterior })
			}
			var resultado dominiovec.ResultadoContextoActorRegistradoV2
			var err error
			if caso.semilla != nil {
				resultado, err = contextoEsperadoRegistradoParaSemillaDesarrollo(context.Background(), resolutor, soporte, *caso.semilla)
			} else {
				resultado, err = contextoEsperadoRegistradoDesarrollo(context.Background(), resolutor, soporte)
			}
			if err != ports.ErrConsultaRRHHNoDisponible || !reflect.DeepEqual(resultado, dominiovec.ResultadoContextoActorRegistradoV2{}) {
				t.Fatalf("el fallo de %s no denegó con error público genérico: %v", caso.nombre, err)
			}
			if !strings.Contains(salida.String(), "etapa="+caso.etapa) || strings.Contains(salida.String(), secreto) {
				t.Fatalf("el diagnóstico de %s falta o contiene datos sensibles", caso.nombre)
			}
		})
	}
}

func resultadoConVinculoEmpleadoF1(t *testing.T, semilla dominiovec.ResultadoContextoActorRegistradoV2) dominiovec.ResultadoContextoActorRegistradoV2 {
	return resultadoConVersionVinculoEmpleadoF1(t, semilla, 1)
}

func resultadoConVersionVinculoEmpleadoF1(t *testing.T, semilla dominiovec.ResultadoContextoActorRegistradoV2, version uint64) dominiovec.ResultadoContextoActorRegistradoV2 {
	t.Helper()
	return resultadoConVinculoContextoPrueba(t, semilla, version, dominiovec.TipoReferenciaContextoActorEmpleado, "emp_", "f1-empleado")
}

// resultadoConVinculoContextoPrueba añade a la semilla un único vínculo
// activo del tipo pedido, con manifiesto y huellas recalculados.
func resultadoConVinculoContextoPrueba(t *testing.T, semilla dominiovec.ResultadoContextoActorRegistradoV2, version uint64,
	tipo dominiovec.TipoReferenciaContextoActor, prefijo, semillaRef string) dominiovec.ResultadoContextoActorRegistradoV2 {
	t.Helper()
	resultado, err := semilla.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	instantanea := resultado.Contexto.Instantanea
	vinculo := dominiovec.VinculoReferenciaContextoActor{
		VinculoRef: referenciaAltaContratacionTemporalDesarrollo("vin_", semillaRef+"-vinculo"),
		Version:    version, Tipo: tipo,
		Referencia:   referenciaAltaContratacionTemporalDesarrollo(prefijo, semillaRef),
		Estado:       dominiovec.EstadoVinculoContextoActorActivo,
		VigenteDesde: instantanea.VigenteDesde, VigenteHasta: instantanea.VigenteHasta,
	}
	instantanea.Vinculos = []dominiovec.VinculoReferenciaContextoActor{vinculo}
	actor, err := dominiovec.NuevoContextoActor(dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: instantanea.CuentaRef, Metodo: dominiovec.AuthMethodCertificate,
		Garantia: dominiovec.AuthAssuranceHigh,
	}, instantanea, resultado.ResueltoEnAutoritativo)
	if err != nil {
		t.Fatal(err)
	}
	resultado.Contexto = actor
	resultado.RepresentacionCanonica, err = actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	resultado.HuellaSHA256, err = actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	manifiesto, err := dominiovec.RehidratarManifiestoProcedenciaContextoActorV1(semilla.ManifiestoProcedenciaCanonico)
	if err != nil {
		t.Fatal(err)
	}
	manifiesto.Vinculos = []dominiovec.ProcedenciaVinculoReferenciaContextoActorV1{{
		VinculoRef: vinculo.VinculoRef, Version: vinculo.Version,
		Tipo: vinculo.Tipo, Referencia: vinculo.Referencia,
		AcreditacionProcedenciaComponenteContextoActorV1: manifiesto.Contexto.AcreditacionProcedenciaComponenteContextoActorV1,
	}}
	resultado.ManifiestoProcedenciaCanonico, err = manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	resultado.ManifiestoProcedenciaHuellaSHA256, err = dominiovec.HuellaSHA256ManifiestoProcedenciaContextoActorV1(resultado.ManifiestoProcedenciaCanonico)
	if err != nil || resultado.Validar() != nil {
		t.Fatalf("contexto de prueba inválido: %v", err)
	}
	return resultado
}

func TestContextoEsperadoRegistradoF1ConvivenciaYDeriva(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	semilla, err := soporte.contexto.Resultado.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	registrado := resultadoConVinculoEmpleadoF1(t, semilla)
	reloj := &relojSesionConsultaPrueba{ahora: time.Now().UTC().Truncate(time.Microsecond)}
	resolutor := &resolutorSesionConsultaPrueba{base: registrado, reloj: reloj, historico: true}
	esperado, err := contextoEsperadoRegistradoDesarrollo(context.Background(), resolutor, soporte)
	if err != nil || !mismoContextoEsperadoRegistradoDesarrollo(registrado, esperado) ||
		len(esperado.Contexto.Instantanea.Vinculos) != 1 ||
		len(soporte.contexto.Resultado.Contexto.Instantanea.Vinculos) != 0 {
		t.Fatalf("el esperado registrado no conserva convivencia sin alterar semilla: %v", err)
	}
	resolutor.historico = false
	fresco, err := resolutor.ResolverContextoActorRegistradoV2(context.Background(), dominiovec.SolicitudContextoActor{
		Cuenta: dominiovec.CuentaAutenticadaContextoActor{CuentaRef: registrado.Contexto.Instantanea.CuentaRef,
			Metodo: dominiovec.AuthMethodCertificate, Garantia: dominiovec.AuthAssuranceHigh},
		PerfilActivoRef: registrado.Contexto.PerfilActivoRef,
	})
	if err != nil || !mismoContextoEsperadoRegistradoDesarrollo(esperado, fresco) ||
		fresco.RegistroContextoRef == esperado.RegistroContextoRef {
		t.Fatalf("el recibo fresco no conserva exactamente la identidad: %v", err)
	}
	derivado, err := registrado.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	derivado.ManifiestoProcedenciaCanonico[0] ^= 1
	if mismoContextoEsperadoRegistradoDesarrollo(esperado, derivado) ||
		mismoContextoEsperadoRegistradoDesarrollo(semilla, fresco) {
		t.Fatal("se aceptó procedencia alterada o semilla histórica como esperado")
	}
	versionNueva := resultadoConVersionVinculoEmpleadoF1(t, semilla, 2)
	if mismoContextoEsperadoRegistradoDesarrollo(esperado, versionNueva) {
		t.Fatal("se aceptó una versión distinta del vínculo empleado")
	}
	revocado, err := registrado.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	revocado.Contexto.Instantanea.Vinculos[0].Estado = dominiovec.EstadoVinculoContextoActorRevocado
	resolutor.base = revocado
	resolutor.historico = true
	if _, err := contextoEsperadoRegistradoDesarrollo(context.Background(), resolutor, soporte); err == nil ||
		mismoContextoEsperadoRegistradoDesarrollo(esperado, revocado) {
		t.Fatal("se aceptó un vínculo revocado")
	}
	caducado, err := registrado.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	caducado.Contexto.Instantanea.Vinculos[0].VigenteHasta = reloj.Ahora().Add(-time.Second)
	resolutor.base = caducado
	if _, err := contextoEsperadoRegistradoDesarrollo(context.Background(), resolutor, soporte); err == nil {
		t.Fatal("se aceptó un vínculo caducado")
	}
}

func TestContextoOperativoF1EscrituraUsaUnReciboFrescoPorPeticion(t *testing.T) {
	e := nuevaSesionConsultaPrueba(t)
	registrado := resultadoConVinculoEmpleadoF1(t, e.soporte.contexto.Resultado)
	e.soporte.contextoEsperadoRegistrado = registrado
	e.resolutor.base = registrado
	proveedor, err := nuevoProveedorSesionConsultaRRHHDesarrollo(e.soporte, e.registro, e.revalidador, e.reloj, e.resolutor)
	if err != nil {
		t.Fatal(err)
	}
	e.soporte.sesionOperativa = proveedor
	contexto := func() context.Context {
		ctx := contextoRutaCoberturaDesarrolloPrueba(e.soporte, e.principal, httpinterno.RutaAltaSolicitudes)
		capacidad := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
		capacidad.certificadoVerificadoEn = e.reloj.Ahora().Add(-time.Second)
		capacidad.certificadoValidoHasta = e.reloj.Ahora().Add(5 * time.Minute)
		capacidad.contextoOperacion = &contextoOperacionCTDesarrollo{}
		return context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	}
	ctx := contexto()
	canal, err := e.soporte.ResolverContextoCanalAlta(ctx)
	if err != nil {
		t.Fatal(err)
	}
	operativo, err := e.soporte.ResolverContextoAutorizacionAltaV3(ctx, ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: canal.AutenticacionRef, SesionRef: canal.SesionRef, PerfilRef: canal.PerfilRef,
	})
	if err != nil || operativo.Resultado.RegistroContextoRef == e.soporte.contexto.Resultado.RegistroContextoRef ||
		len(operativo.Resultado.Contexto.Instantanea.Vinculos) != 1 ||
		len(e.registro.altas) != 1 || e.resolutor.llamadas != 1 ||
		!mismoContextoEsperadoRegistradoDesarrollo(registrado, operativo.Resultado) {
		t.Fatalf("la escritura no conservó una sesión operativa y vínculo registrado: %v", err)
	}
	// Otra petición resuelve de nuevo. Si la autoridad ya no devuelve el
	// esperado, el holder no acepta ni la semilla ni el recibo anterior.
	e.resolutor.base = e.soporte.contexto.Resultado
	if _, err := e.soporte.ResolverContextoCanalAlta(contexto()); err == nil || e.resolutor.llamadas != 2 {
		t.Fatal("se aceptó deriva de vínculos en una petición nueva")
	}
	if len(e.soporte.contexto.Resultado.Contexto.Instantanea.Vinculos) != 0 {
		t.Fatal("la semilla histórica fue modificada")
	}
}

func TestContextoOperativoReincorporacionSeleccionaPerfilYSesionDedicados(t *testing.T) {
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	base := soporte.contexto
	reincorporacion, err := nuevoContextoReincorporacionTitularDesarrollo(soporte, soporte.reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	baseVinculo, _ := base.Vinculo.Datos()
	reincVinculo, _ := reincorporacion.Vinculo.Datos()
	if baseVinculo.PerfilActivoRef == reincVinculo.PerfilActivoRef ||
		baseVinculo.SesionRef == reincVinculo.SesionRef ||
		base.Resultado.Contexto.Instantanea.VinculoRef == reincorporacion.Resultado.Contexto.Instantanea.VinculoRef ||
		base.Resultado.Contexto.Instantanea.CuentaRef != reincorporacion.Resultado.Contexto.Instantanea.CuentaRef ||
		base.Resultado.Contexto.PersonaRef != reincorporacion.Resultado.Contexto.PersonaRef {
		t.Fatal("reincorporación no tiene perfil, sesión y contexto propios sobre la misma persona")
	}
	soporte.reincorporacionTitular = &soporteSeguimientoCeseDesarrollo{
		contexto: reincorporacion, contextoEsperadoRegistrado: reincorporacion.Resultado,
		sesionOperativa: proveedorSesionOperativaCTPrueba{contexto: reincorporacion},
	}
	baseOperativo, err := soporte.contextoOperativoDesarrollo(contextoRutaCoberturaDesarrolloPrueba(soporte, principal, httpinterno.RutaAltaSolicitudes))
	if err != nil || baseOperativo.Resultado.Contexto.PerfilActivoRef != baseVinculo.PerfilActivoRef {
		t.Fatalf("perfil CT base alterado: %v", err)
	}
	for _, ruta := range []string{httpinterno.RutaReincorporacionesTitular, httpinterno.RutaCapacidadReincorporacionTitular} {
		operativo, err := soporte.contextoOperativoDesarrollo(contextoRutaCoberturaDesarrolloPrueba(soporte, principal, ruta))
		if err != nil || operativo.Resultado.Contexto.PerfilActivoRef != reincVinculo.PerfilActivoRef {
			t.Fatalf("ruta %s no usa perfil dedicado: %v", ruta, err)
		}
	}
	soporte.reincorporacionTitular.sesionOperativa = proveedorSesionOperativaCTPrueba{contexto: base}
	if _, err := soporte.contextoOperativoDesarrollo(contextoRutaCoberturaDesarrolloPrueba(soporte, principal, httpinterno.RutaReincorporacionesTitular)); err == nil {
		t.Fatal("sesión CT base fue aceptada para CT130")
	}
}

func TestSesionReincorporacionTitularRevalidaMTLSConPerfilPropio(t *testing.T) {
	e := nuevaSesionConsultaPrueba(t)
	basePerfil := e.soporte.contexto.Resultado.Contexto.PerfilActivoRef
	contextoCT130, err := nuevoContextoReincorporacionTitularDesarrollo(e.soporte, e.reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	perfilCT130 := contextoCT130.Resultado.Contexto.PerfilActivoRef
	fronteras, err := nuevoCatalogoFronterasComunDesarrollo(append(
		descriptoresFronterasContratacionTemporalDesarrollo(basePerfil, []string{basePerfil}),
		descriptoresFronterasReincorporacionTitularDesarrollo(perfilCT130)...))
	if err != nil {
		t.Fatal(err)
	}
	e.p.fronteras = fronteras
	e.soporte.sesionOperativa = e.p
	sesionCT130, err := nuevaSesionReincorporacionTitularDesarrollo(e.p, contextoCT130.Resultado)
	if err != nil {
		t.Fatal(err)
	}
	e.soporte.reincorporacionTitular = &soporteSeguimientoCeseDesarrollo{
		contexto: contextoCT130, contextoEsperadoRegistrado: contextoCT130.Resultado,
		sesionOperativa: sesionCT130,
	}
	e.resolutor.base = contextoCT130.Resultado
	ctx := contextoRutaCoberturaDesarrolloPrueba(e.soporte, e.principal, httpinterno.RutaReincorporacionesTitular)
	frontera, ok := fronteras.resolver(http.MethodPost, httpinterno.RutaReincorporacionesTitular)
	if !ok {
		t.Fatal("frontera CT130 ausente")
	}
	ctx = context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
		metodo: http.MethodPost, ruta: httpinterno.RutaReincorporacionesTitular,
		superficie: superficieInternaSeguridadComunDesarrollo, catalogo: fronteras, descriptor: frontera,
	})
	operativo, err := e.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := operativo.Vinculo.Datos()
	nominal, _ := contextoCT130.Vinculo.Datos()
	if err != nil || vinculo.PerfilActivoRef != perfilCT130 || vinculo.SesionRef == "" ||
		vinculo.SesionRef == nominal.SesionRef ||
		len(e.registro.altas) != 1 || e.revalidador.llamadas != 1 || e.resolutor.llamadas != 1 ||
		e.registro.altas[0].CuentaID != "desarrollo:"+contextoCT130.Resultado.Contexto.Instantanea.CuentaRef {
		t.Fatalf("sesión mTLS CT130 no revalidada con perfil propio: %v", err)
	}
	if _, err := e.p.ResolverContexto(ctx); err == nil {
		t.Fatal("proveedor CT base aceptó frontera exclusiva CT130")
	}
}

package bootstrap

import (
	"context"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func resultadoConVinculoEmpleadoF1(t *testing.T, semilla dominiovec.ResultadoContextoActorRegistradoV2) dominiovec.ResultadoContextoActorRegistradoV2 {
	return resultadoConVersionVinculoEmpleadoF1(t, semilla, 1)
}

func resultadoConVersionVinculoEmpleadoF1(t *testing.T, semilla dominiovec.ResultadoContextoActorRegistradoV2, version uint64) dominiovec.ResultadoContextoActorRegistradoV2 {
	t.Helper()
	resultado, err := semilla.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	instantanea := resultado.Contexto.Instantanea
	vinculo := dominiovec.VinculoReferenciaContextoActor{
		VinculoRef: referenciaAltaContratacionTemporalDesarrollo("vin_", "f1-empleado-vinculo"),
		Version:    version, Tipo: dominiovec.TipoReferenciaContextoActorEmpleado,
		Referencia:   referenciaAltaContratacionTemporalDesarrollo("emp_", "f1-empleado"),
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

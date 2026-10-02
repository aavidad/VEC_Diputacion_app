package composicion

import (
	"context"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type resolutorIntentoRPTVigenciaPrueba struct {
	identidad IdentidadRegistradaLectorRelacionRPT
	despues   func()
	llamadas  int
}

func (r *resolutorIntentoRPTVigenciaPrueba) ResolverIdentidadLectorRelacionRPT(context.Context) (IdentidadRegistradaLectorRelacionRPT, error) {
	r.llamadas++
	if r.despues != nil {
		r.despues()
	}
	return r.identidad, nil
}

type revalidadorIntentoRPTVigenciaPrueba struct {
	resultado vecdomain.AutenticacionRevalidadaV1
	llamadas  int
}

func (r *revalidadorIntentoRPTVigenciaPrueba) RevalidarAutenticacionActorV1(context.Context, vecdomain.SolicitudRevalidacionAutenticacionActorV1) (vecdomain.AutenticacionRevalidadaV1, error) {
	r.llamadas++
	return r.resultado, nil
}

type contextoRegistradoIntentoRPTVigenciaPrueba struct {
	resultado vecdomain.ResultadoContextoActorRegistradoV2
	llamadas  int
}

func (r *contextoRegistradoIntentoRPTVigenciaPrueba) ResolverContextoActorRegistradoV2(context.Context, vecdomain.SolicitudContextoActor) (vecdomain.ResultadoContextoActorRegistradoV2, error) {
	r.llamadas++
	return r.resultado, nil
}

type relojIntentoRPTVigenciaPrueba struct{ instante time.Time }

func (r relojIntentoRPTVigenciaPrueba) Ahora() time.Time { return r.instante }

// Las autoridades son fixtures unitarios; el vínculo opaco y el resultado
// ligado se crean mediante la fábrica real del dominio. No acredita una sesión
// emitida por infraestructura externa ni persistencia del registro de contexto.
func identidadIntentoRPTVigenciaPrueba(t *testing.T, instante time.Time, caduca string) IdentidadRegistradaLectorRelacionRPT {
	t.Helper()
	sufijo := "0123456789abcdefghijkl"
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + sufijo, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	snapshot := vecdomain.InstantaneaContextoActor{
		VinculoRef: "vca_" + sufijo, VinculoVersion: 5, CuentaRef: cuenta.CuentaRef, CuentaVersion: 7,
		PersonaRef: "per_" + sufijo, PersonaVersion: 3, PerfilActivoRef: "prf_" + sufijo, PerfilVersion: 4,
		Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: instante.Add(-time.Hour), VigenteHasta: instante.Add(30 * time.Minute),
		Vinculos: []vecdomain.VinculoReferenciaContextoActor{{VinculoRef: "pep_" + sufijo, Version: 2, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: "emp_" + sufijo, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: instante.Add(-time.Hour), VigenteHasta: instante.Add(30 * time.Minute)}},
	}
	if caduca == "contexto" {
		snapshot.VigenteHasta = instante.Add(time.Second)
	}
	if caduca == "enlace_empleado" {
		snapshot.Vinculos[0].VigenteHasta = instante.Add(time.Second)
	}
	actor, err := vecdomain.NuevoContextoActor(cuenta, snapshot, instante.Add(-2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	acreditacion := vecdomain.AcreditacionProcedenciaComponenteContextoActorV1{ProcedenciaRef: "prc_" + sufijo, ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("a", 64), ProcedenciaAutoridad: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1}
	manifiesto := vecdomain.ManifiestoProcedenciaContextoActorV1{
		Esquema: vecdomain.EsquemaManifiestoProcedenciaContextoActorV1, AutoridadEfectiva: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta:   vecdomain.ProcedenciaCuentaContextoActorV1{CuentaRef: actor.Instantanea.CuentaRef, Version: actor.Instantanea.CuentaVersion, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Persona:  vecdomain.ProcedenciaPersonaContextoActorV1{PersonaRef: actor.PersonaRef, Version: actor.Instantanea.PersonaVersion, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Perfil:   vecdomain.ProcedenciaPerfilContextoActorV1{PerfilRef: actor.PerfilActivoRef, Version: actor.Instantanea.PerfilVersion, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Contexto: vecdomain.ProcedenciaVinculoContextoActorV1{VinculoRef: actor.Instantanea.VinculoRef, Version: actor.Instantanea.VinculoVersion, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Vinculos: []vecdomain.ProcedenciaVinculoReferenciaContextoActorV1{{VinculoRef: actor.Instantanea.Vinculos[0].VinculoRef, Version: actor.Instantanea.Vinculos[0].Version, Tipo: actor.Instantanea.Vinculos[0].Tipo, Referencia: actor.Instantanea.Vinculos[0].Referencia, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion}},
	}
	canonManifiesto, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	huellaManifiesto, err := vecdomain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(canonManifiesto)
	if err != nil {
		t.Fatal(err)
	}
	resultado := vecdomain.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: "rca_0123456789abcdefghijklmn", Contexto: actor,
		RepresentacionCanonica: canon, HuellaSHA256: huella,
		ManifiestoProcedenciaCanonico: canonManifiesto, ManifiestoProcedenciaHuellaSHA256: huellaManifiesto,
		AutoridadEfectiva: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, ResueltoEnAutoritativo: actor.ResueltoEn,
	}
	if err := resultado.Validar(); err != nil {
		t.Fatal(err)
	}
	autenticacion := vecdomain.AutenticacionRevalidadaV1{
		AutenticacionRef: "aut_" + sufijo, AutenticacionHuellaSHA256: strings.Repeat("1", 64),
		AsercionRef: "ase_" + sufijo, SesionRef: "ses_" + sufijo, ControlSesionRef: "cse_" + sufijo,
		ControlSesionRevision: 7, ControlSesionHuellaSHA256: strings.Repeat("2", 64),
		CuentaRef: cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef,
		Superficie: vecdomain.SuperficieAutenticacionInternaCorporativaV1, MetodoObservado: cuenta.Metodo, GarantiaObservada: cuenta.Garantia,
		PoliticaGarantiaRef: "pga_" + sufijo, PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn: instante.Add(-5 * time.Minute), SesionEmitidaEn: instante.Add(-4 * time.Minute),
		SesionRevalidadaEn: instante.Add(-3 * time.Minute), SesionValidaHasta: instante.Add(10 * time.Minute),
	}
	if caduca == "sesion" {
		autenticacion.SesionValidaHasta = instante.Add(time.Second)
	}
	revalidador := &revalidadorIntentoRPTVigenciaPrueba{resultado: autenticacion}
	resolutor := &contextoRegistradoIntentoRPTVigenciaPrueba{resultado: resultado}
	vinculo, registrado, err := vecdomain.CrearVinculoAutenticacionActorV2ConResultado(context.Background(), revalidador,
		vecdomain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef},
		resolutor, vecdomain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: actor.PerfilActivoRef}, relojIntentoRPTVigenciaPrueba{instante: instante})
	if err != nil || vinculo.ValidarPara(registrado) != nil || !vinculo.VigenteEn(instante, registrado) {
		t.Fatalf("fábrica no produjo identidad válida en T0: %v", err)
	}
	if revalidador.llamadas != 1 || resolutor.llamadas != 1 {
		t.Fatal("la fábrica no cruzó ambas autoridades una vez")
	}
	return IdentidadRegistradaLectorRelacionRPT{Vinculo: vinculo, Resultado: registrado}
}

func TestIntentoRPTActorVigenteSeAtribuyeTrasResolver(t *testing.T) {
	t0 := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	identidad := identidadIntentoRPTVigenciaPrueba(t, t0, "")
	ahora := t0
	resolutor := &resolutorIntentoRPTVigenciaPrueba{identidad: identidad, despues: func() { ahora = t0.Add(time.Microsecond) }}
	destino := &destinoIntentosRPTPrueba{}
	registro, err := NuevoRegistroIntentosLectorRelacionRPT(resolutor, destino, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	intento := ports.IntentoLectorRelacionRPT{Actor: identidad.Resultado.Contexto, RelacionRef: "rel_" + strings.Repeat("r", 24), Motivo: "denegado"}
	if err := registro.RegistrarIntentoRelacionRPT(context.Background(), intento); err != nil {
		t.Fatal(err)
	}
	if destino.writes != 1 || resolutor.llamadas != 1 || destino.evento.ActorRef != identidad.Resultado.Contexto.PersonaRef || destino.evento.RelacionRef != intento.RelacionRef || destino.evento.Motivo != intento.Motivo || !strings.HasPrefix(destino.evento.CorrelacionRef, "correlacion_") {
		t.Fatalf("intento no atribuido a la identidad vigente: %+v", destino.evento)
	}
}

func TestIntentoRPTCaducidadDuranteResolucionConservaIntentoSinActor(t *testing.T) {
	for _, caduca := range []string{"sesion", "contexto", "enlace_empleado"} {
		t.Run(caduca, func(t *testing.T) {
			t0 := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
			identidad := identidadIntentoRPTVigenciaPrueba(t, t0, caduca)
			t1 := t0.Add(time.Second)
			// Sigue válida en forma y para la resolución anterior: el defecto
			// solo aparece al comprobar la vigencia después de resolver.
			if identidad.Resultado.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.Resultado) != nil || identidad.Vinculo.VigenteEn(t1, identidad.Resultado) {
				t.Fatal("fixture no distingue validez histórica de vigencia actual")
			}
			ahora := t0
			resolutor := &resolutorIntentoRPTVigenciaPrueba{identidad: identidad, despues: func() { ahora = t1 }}
			destino := &destinoIntentosRPTPrueba{}
			registro, err := NuevoRegistroIntentosLectorRelacionRPT(resolutor, destino, func() time.Time { return ahora })
			if err != nil {
				t.Fatal(err)
			}
			intento := ports.IntentoLectorRelacionRPT{Actor: identidad.Resultado.Contexto, RelacionRef: "rel_" + strings.Repeat("r", 24), Motivo: "denegado"}
			if err := registro.RegistrarIntentoRelacionRPT(context.Background(), intento); err != nil {
				t.Fatal(err)
			}
			if destino.writes != 1 || resolutor.llamadas != 1 || destino.evento.ActorRef != "" || destino.evento.RelacionRef != intento.RelacionRef || destino.evento.Motivo != intento.Motivo || !strings.HasPrefix(destino.evento.CorrelacionRef, "correlacion_") {
				t.Fatalf("identidad caducada atribuida o intento perdido: %+v", destino.evento)
			}
		})
	}
}

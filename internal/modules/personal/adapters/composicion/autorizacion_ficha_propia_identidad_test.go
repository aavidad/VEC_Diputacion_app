package composicion

import (
	"context"
	"strings"
	"testing"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type resolutorIntentoFichaPropiaVigenciaPrueba struct {
	identidad IdentidadRegistradaFichaPropia
	despues   func()
	llamadas  int
}

func (r *resolutorIntentoFichaPropiaVigenciaPrueba) ResolverIdentidadFichaPropia(context.Context) (IdentidadRegistradaFichaPropia, error) {
	r.llamadas++
	if r.despues != nil {
		r.despues()
	}
	return r.identidad, nil
}

type revalidadorIntentoFichaPropiaVigenciaPrueba struct {
	resultado vecdomain.AutenticacionRevalidadaV1
	llamadas  int
}

func (r *revalidadorIntentoFichaPropiaVigenciaPrueba) RevalidarAutenticacionActorV1(context.Context, vecdomain.SolicitudRevalidacionAutenticacionActorV1) (vecdomain.AutenticacionRevalidadaV1, error) {
	r.llamadas++
	return r.resultado, nil
}

type contextoRegistradoIntentoFichaPropiaVigenciaPrueba struct {
	resultado vecdomain.ResultadoContextoActorRegistradoV2
	llamadas  int
}

func (r *contextoRegistradoIntentoFichaPropiaVigenciaPrueba) ResolverContextoActorRegistradoV2(context.Context, vecdomain.SolicitudContextoActor) (vecdomain.ResultadoContextoActorRegistradoV2, error) {
	r.llamadas++
	return r.resultado, nil
}

type relojIntentoFichaPropiaVigenciaPrueba struct{ instante time.Time }

func (r relojIntentoFichaPropiaVigenciaPrueba) Ahora() time.Time { return r.instante }

// Las autoridades son fixtures unitarios; el vínculo opaco y el resultado
// ligado se crean mediante la fábrica real del dominio. No acredita una sesión
// emitida por infraestructura externa ni persistencia del registro de contexto.
func identidadIntentoFichaPropiaVigenciaPrueba(t *testing.T, instante time.Time, caduca string) IdentidadRegistradaFichaPropia {
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
	revalidador := &revalidadorIntentoFichaPropiaVigenciaPrueba{resultado: autenticacion}
	resolutor := &contextoRegistradoIntentoFichaPropiaVigenciaPrueba{resultado: resultado}
	vinculo, registrado, err := vecdomain.CrearVinculoAutenticacionActorV2ConResultado(context.Background(), revalidador,
		vecdomain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef},
		resolutor, vecdomain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: actor.PerfilActivoRef}, relojIntentoFichaPropiaVigenciaPrueba{instante: instante})
	if err != nil || vinculo.ValidarPara(registrado) != nil || !vinculo.VigenteEn(instante, registrado) {
		t.Fatalf("fábrica no produjo identidad válida en T0: %v", err)
	}
	if revalidador.llamadas != 1 || resolutor.llamadas != 1 {
		t.Fatal("la fábrica no cruzó ambas autoridades una vez")
	}
	return IdentidadRegistradaFichaPropia{Vinculo: vinculo, Resultado: registrado}
}

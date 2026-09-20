package bolsa

import (
	"context"
	"strings"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type emisorB11Prueba struct{ llamadas int }

func (e *emisorB11Prueba) EmitirMaterialAutorizacionAtestadaV3(context.Context, dominiovec.SolicitudAutorizacionLigadaV3, dominiovec.ResultadoContextoActorRegistradoV2) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	return dominiovec.DecisionAutorizacionLigadaV3{}, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, nil
}

type correladorB11Prueba struct{ llamadas int }

func (g *correladorB11Prueba) NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error) {
	g.llamadas++
	return "correlacion_11111111111111111111111111111111", nil
}

type revalidadorB11Prueba struct {
	autenticacion dominiovec.AutenticacionRevalidadaV1
}

func (r revalidadorB11Prueba) RevalidarAutenticacionActorV1(context.Context, dominiovec.SolicitudRevalidacionAutenticacionActorV1) (dominiovec.AutenticacionRevalidadaV1, error) {
	return r.autenticacion, nil
}

type resolutorB11Prueba struct {
	resultado dominiovec.ResultadoContextoActorRegistradoV2
}

func (r resolutorB11Prueba) ResolverContextoActorRegistradoV2(context.Context, dominiovec.SolicitudContextoActor) (dominiovec.ResultadoContextoActorRegistradoV2, error) {
	return r.resultado, nil
}

type relojB11Prueba struct{ ahora time.Time }

func (r relojB11Prueba) Ahora() time.Time { return r.ahora }

func parB11Prueba(t *testing.T) (dominiovec.VinculoAutenticacionActorV2, dominiovec.ResultadoContextoActorRegistradoV2) {
	t.Helper()
	ahora := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	cuenta := dominiovec.CuentaAutenticadaContextoActor{CuentaRef: "cta_0123456789abcdefghijkl", Metodo: dominiovec.AuthMethodCertificate, Garantia: dominiovec.AuthAssuranceHigh}
	instantanea := dominiovec.InstantaneaContextoActor{VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 3, CuentaRef: cuenta.CuentaRef, CuentaVersion: 4, PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 2, PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 5, Estado: dominiovec.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	actor, err := dominiovec.NuevoContextoActor(cuenta, instantanea, ahora.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	representacion, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	acreditacion := dominiovec.AcreditacionProcedenciaComponenteContextoActorV1{ProcedenciaRef: "prc_0123456789abcdefghijkl", ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("4", 64), ProcedenciaAutoridad: dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1}
	manifiesto := dominiovec.ManifiestoProcedenciaContextoActorV1{Esquema: dominiovec.EsquemaManifiestoProcedenciaContextoActorV1, AutoridadEfectiva: dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, Cuenta: dominiovec.ProcedenciaCuentaContextoActorV1{CuentaRef: cuenta.CuentaRef, Version: 4, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion}, Persona: dominiovec.ProcedenciaPersonaContextoActorV1{PersonaRef: instantanea.PersonaRef, Version: 2, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion}, Perfil: dominiovec.ProcedenciaPerfilContextoActorV1{PerfilRef: instantanea.PerfilActivoRef, Version: 5, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion}, Contexto: dominiovec.ProcedenciaVinculoContextoActorV1{VinculoRef: instantanea.VinculoRef, Version: 3, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion}, Vinculos: make([]dominiovec.ProcedenciaVinculoReferenciaContextoActorV1, 0)}
	canonico, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	huellaManifiesto, err := dominiovec.HuellaSHA256ManifiestoProcedenciaContextoActorV1(canonico)
	if err != nil {
		t.Fatal(err)
	}
	resultado := dominiovec.ResultadoContextoActorRegistradoV2{RegistroContextoRef: "rca_0123456789abcdefghijklmn", Contexto: actor, RepresentacionCanonica: representacion, HuellaSHA256: huella, ManifiestoProcedenciaCanonico: canonico, ManifiestoProcedenciaHuellaSHA256: huellaManifiesto, AutoridadEfectiva: dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, ResueltoEnAutoritativo: actor.ResueltoEn}
	autenticacion := dominiovec.AutenticacionRevalidadaV1{AutenticacionRef: "aut_0123456789abcdefghijkl", AutenticacionHuellaSHA256: strings.Repeat("1", 64), AsercionRef: "ase_0123456789abcdefghijkl", SesionRef: "ses_0123456789abcdefghijkl", ControlSesionRef: "cse_0123456789abcdefghijkl", ControlSesionRevision: 2, ControlSesionHuellaSHA256: strings.Repeat("2", 64), CuentaRef: cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef, Superficie: dominiovec.SuperficieAutenticacionExternaPersonalV1, MetodoObservado: cuenta.Metodo, GarantiaObservada: cuenta.Garantia, PoliticaGarantiaRef: "pga_0123456789abcdefghijkl", PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64), AutenticacionVerificadaEn: ahora.Add(-10 * time.Minute), SesionEmitidaEn: ahora.Add(-9 * time.Minute), SesionRevalidadaEn: ahora.Add(-3 * time.Minute), SesionValidaHasta: ahora.Add(20 * time.Minute)}
	vinculo, err := dominiovec.CrearVinculoAutenticacionActorV2(context.Background(), revalidadorB11Prueba{autenticacion}, dominiovec.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef}, resolutorB11Prueba{resultado}, dominiovec.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: instantanea.PerfilActivoRef}, relojB11Prueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	return vinculo, resultado
}

func motivoB11Prueba() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 2, CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_11111111111111111111111111111111"}
}

func recursoB11Prueba() dominiovec.RecursoAutorizable {
	return dominiovec.RecursoAutorizable{Referencia: "candidato:can_0123456789abcdef", ModuloID: puertosbolsa.ModuloParticipacionesPropiasBolsa, Tipo: puertosbolsa.TipoRecursoParticipacionesPropias, Ambitos: map[string]string{"candidato_ref": "can_0123456789abcdef"}, Atributos: map[string]string{"finalidad": puertosbolsa.FinalidadConsultarParticipacionesPropias}}
}

func TestNuevoAutorizadorParticipacionesPropiasV3DeniegaDependenciasNulasYMotivoNoGobernado(t *testing.T) {
	validoEmisor, validoCorrelador := &emisorB11Prueba{}, &correladorB11Prueba{}
	for nombre, caso := range map[string]struct {
		emisor     EmisorMaterialParticipacionesPropias
		correlador GeneradorCorrelacionParticipacionesPropias
		motivo     dominiovec.ReferenciaEntradaCatalogo
	}{
		"emisor_nulo":         {nil, validoCorrelador, motivoB11Prueba()},
		"correlador_nulo":     {validoEmisor, nil, motivoB11Prueba()},
		"motivo_no_gobernado": {validoEmisor, validoCorrelador, dominiovec.ReferenciaEntradaCatalogo{}},
	} {
		t.Run(nombre, func(t *testing.T) {
			if a, err := NuevoAutorizadorParticipacionesPropiasV3(caso.emisor, caso.correlador, caso.motivo); err == nil || a != nil {
				t.Fatal("constructor aceptó dependencia o motivo inválido")
			}
		})
	}
}

func TestAutorizadorParticipacionesPropiasV3DeniegaActorCruzadoAntesDeEmitir(t *testing.T) {
	emisor, correlador := &emisorB11Prueba{}, &correladorB11Prueba{}
	a, err := NuevoAutorizadorParticipacionesPropiasV3(emisor, correlador, motivoB11Prueba())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.AutorizarOperacion(context.Background(), dominiovec.VinculoAutenticacionActorV2{}, dominiovec.ResultadoContextoActorRegistradoV2{}, puertosbolsa.AccionConsultarParticipacionesPropias, recursoB11Prueba()); err == nil {
		t.Fatal("actor cruzado o no ligado alcanzó al emisor")
	}
	if emisor.llamadas != 0 || correlador.llamadas != 0 {
		t.Fatalf("se emitió o acuñó correlación antes de validar actor: emisor=%d correlador=%d", emisor.llamadas, correlador.llamadas)
	}
}

func TestAutorizadorParticipacionesPropiasV3EmiteUnaSolaVezAnteEmisorAdulterado(t *testing.T) {
	emisor, correlador := &emisorB11Prueba{}, &correladorB11Prueba{}
	a, err := NuevoAutorizadorParticipacionesPropiasV3(emisor, correlador, motivoB11Prueba())
	if err != nil {
		t.Fatal(err)
	}
	vinculo, resultado := parB11Prueba(t)
	if _, err = a.AutorizarOperacion(context.Background(), vinculo, resultado, puertosbolsa.AccionConsultarParticipacionesPropias, recursoB11Prueba()); err == nil {
		t.Fatal("emisor adulterado fue aceptado")
	}
	if emisor.llamadas != 1 || correlador.llamadas != 1 {
		t.Fatalf("emisión no única: emisor=%d correlador=%d", emisor.llamadas, correlador.llamadas)
	}
}

func TestContratoB11ExigeAccionRecursoYFinalidadExactos(t *testing.T) {
	for nombre, caso := range map[string]struct {
		accion string
		mutar  func(*dominiovec.RecursoAutorizable)
	}{
		"accion": {"bolsa.candidato.participaciones.modificar", func(*dominiovec.RecursoAutorizable) {}},
		"audiencia_no_es_finalidad": {puertosbolsa.AccionConsultarParticipacionesPropias, func(r *dominiovec.RecursoAutorizable) {
			r.Atributos["finalidad"] = puertosbolsa.AudienciaParticipacionesPropias
		}},
		"recurso_ajeno": {puertosbolsa.AccionConsultarParticipacionesPropias, func(r *dominiovec.RecursoAutorizable) { r.Referencia = "candidato:can_ajeno" }},
	} {
		t.Run(nombre, func(t *testing.T) {
			r := recursoB11Prueba()
			caso.mutar(&r)
			if contratoB11Valido(caso.accion, r) {
				t.Fatal("contrato B11 aceptó una acción, recurso o finalidad distinta")
			}
		})
	}
}

func TestMaterialB11ValidoDeniegaExportacionAdulterada(t *testing.T) {
	// Una exportación estructuralmente vacía no puede suplir una decisión,
	// confirmación ni una audiencia/efecto ligados al recurso B11.
	if materialB11Valido(puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, dominiovec.SolicitudAutorizacionLigadaV3{}, dominiovec.DecisionAutorizacionLigadaV3{}, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, dominiovec.ResultadoContextoActorRegistradoV2{}, puertosbolsa.AccionConsultarParticipacionesPropias, recursoB11Prueba()) {
		t.Fatal("exportación adulterada fue aceptada")
	}
}

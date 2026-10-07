package firmaemisorv2

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecapp "vec-diputacion-granada/internal/vec/application"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type autorizacionAdmisionPrueba struct{ snapshot vd.InstantaneaAutorizacion }

func (a autorizacionAdmisionPrueba) ObtenerInstantaneaAutorizacion(context.Context, string, string) (vd.InstantaneaAutorizacion, error) {
	return a.snapshot, nil
}

type emisorFalloAdmisionPrueba struct{ llamadas int }

func (e *emisorFalloAdmisionPrueba) EmitirMaterialAutorizacionAtestadaV3(context.Context,
	vd.SolicitudAutorizacionLigadaV3, vd.ResultadoContextoActorRegistradoV2,
) (vd.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vp.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	return vd.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, errors.New("fuente V3 de prueba caída")
}

func TestFirmaVecDesarrolloExigeGateYNoAdmiteHIGHComoExcepcion(t *testing.T) {
	base, fuente, emisor, _, _, ctx := escenario(t, ports.ViaFirmaCertificadoVEC)
	if _, err := NuevoEmisorFirmaVecDesarrollo(fuente, emisor, base.motivo, base.reloj,
		base.autorizacion, nil); !errors.Is(err, ports.ErrCompetenciaFirmanteNoDisponible) {
		t.Fatalf("constructor sin admisión: %v", err)
	}
	dev, err := NuevoEmisorFirmaVecDesarrollo(fuente, emisor, base.motivo, base.reloj,
		base.autorizacion, &vecapp.AdmisionGarantiaFirmaVecDesarrollo{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dev.ObtenerPerfilActivoOperadorFirmaV2(ctx); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || emisor.llamadas != 0 {
		t.Fatalf("HIGH no pertenece a la excepción DEV: %v", err)
	}
	if _, err := base.ObtenerPerfilActivoOperadorFirmaV2(ctx); err != nil {
		t.Fatalf("constructor corporativo HIGH cambió: %v", err)
	}
}

func TestFirmaVecDesarrolloSinPoliticaVigenteNoLlegaAlEmisorV3(t *testing.T) {
	base, fuente, emisor, material, _, ctx := escenario(t, ports.ViaFirmaCertificadoVEC)
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(emisor.reloj.Ahora(),
		fuente.base.Resultado.Contexto.PersonaRef, fuente.base.Resultado.Contexto.PerfilActivoRef,
		vd.AuthMethodCertificate, vd.AuthAssuranceSubstantial)
	if err != nil {
		t.Fatal(err)
	}
	fuente.base.Resultado, fuente.base.Vinculo = resultado, vinculo
	datos, err := vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	material.CuentaFirmanteRef = datos.CuentaRef
	recurso := recursoPrueba(t, material)
	dev, err := NuevoEmisorFirmaVecDesarrollo(fuente, emisor, base.motivo, base.reloj,
		base.autorizacion, &vecapp.AdmisionGarantiaFirmaVecDesarrollo{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dev.AutorizarMaterialFirmaVerificadaV2(ctx, material, recurso); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || emisor.llamadas != 0 {
		t.Fatalf("sin política aprobada produjo decisión V3: %v, llamadas=%d", err, emisor.llamadas)
	}
}

func TestFirmaVecDesarrolloAdmiteSinEmitirV3HastaCapturaFresca(t *testing.T) {
	base, fuente, emisorBase, material, _, ctx := escenario(t, ports.ViaFirmaCertificadoVEC)
	ahora := base.reloj.Ahora().UTC().Truncate(time.Microsecond)
	retirada := ahora.Add(10 * time.Minute)
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora,
		fuente.base.Resultado.Contexto.PersonaRef, fuente.base.Resultado.Contexto.PerfilActivoRef,
		vd.AuthMethodCertificate, vd.AuthAssuranceSubstantial)
	if err != nil {
		t.Fatal(err)
	}
	fuente.base.Resultado, fuente.base.Vinculo = resultado, vinculo
	datos, err := vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	material.CuentaFirmanteRef = datos.CuentaRef
	recurso := recursoPrueba(t, material)
	snapshot := instantaneaPrueba(t, datos.PrincipalID, datos.PerfilActivoRef,
		[]vd.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{material.OrganizacionRef}}}, ahora,
		vd.ConcesionRol{Accion: ports.AccionRegistrarFirmaVec, ModuloID: ports.ModuloContratacion,
			TipoRecurso: ports.TipoRecursoFirmaVec, Finalidades: []string{ports.FinalidadFirmaDocumento},
			GarantiaMinima: vd.AuthAssuranceSubstantial})
	rolSHA, err := snapshot.VersionRol.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	controlSHA, err := snapshot.ControlVigenciaVersionRol.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	gate, err := vecapp.NuevaAdmisionGarantiaFirmaVecDesarrollo(vecapp.ConfiguracionAdmisionGarantiaFirmaVecDesarrollo{
		EntornoConfiable: vecapp.EntornoAdmisionFirmaDesarrollo, Reloj: base.reloj,
		Politica: vecapp.PoliticaPrivadaAdmisionFirmaDesarrollo{
			Referencia: "politica:ct:firma_vec_dev:v1", HuellaSHA256: strings.Repeat("a", 64),
			Version: 1, RetiradaEn: retirada,
			PoliticaAutenticacionRef:    datos.PoliticaGarantiaRef,
			PoliticaAutenticacionSHA256: datos.PoliticaGarantiaHuellaSHA256,
			RolVersionRef:               snapshot.VersionRol.Referencia(), RolSHA256: rolSHA,
			ControlRevision: snapshot.ControlVigenciaVersionRol.Revision, ControlSHA256: controlSHA,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	emisor := &emisorFalloAdmisionPrueba{}
	dev, err := NuevoEmisorFirmaVecDesarrollo(fuente, emisor, base.motivo, base.reloj,
		autorizacionAdmisionPrueba{snapshot: snapshot}, gate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dev.AutorizarMaterialFirmaVerificadaV2(ctx, material, recurso); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || emisor.llamadas != 0 {
		t.Fatalf("admisión previa o rechazo de emisor V3: err=%v llamadas=%d", err, emisor.llamadas)
	}
	emisorBase.reloj.ahora = retirada
	if _, err := dev.AutorizarMaterialFirmaVerificadaV2(ctx, material, recurso); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || emisor.llamadas != 0 {
		t.Fatalf("política retirada alcanzó V3: err=%v llamadas=%d", err, emisor.llamadas)
	}
}

func TestFirmaVecDesarrolloNoExtiendeExcepcionAConsultaNiRecuperacion(t *testing.T) {
	a, fuente, emisor, consulta, ctx := escenarioConsulta(t, ports.ViaFirmaCertificadoVEC)
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(a.reloj.Ahora(),
		fuente.base.Resultado.Contexto.PersonaRef, fuente.base.Resultado.Contexto.PerfilActivoRef,
		vd.AuthMethodCertificate, vd.AuthAssuranceSubstantial)
	if err != nil {
		t.Fatal(err)
	}
	fuente.base.Resultado, fuente.base.Vinculo = resultado, vinculo
	a.admision = &vecapp.AdmisionGarantiaFirmaVecDesarrollo{}
	if _, err := a.ObtenerPerfilActivoOperadorFirmaV2(ctx); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
		t.Fatalf("getter de perfil DEV sin acto: %v", err)
	}
	if _, err := a.ObtenerAmbitosOperadorFirmaV2(ctx); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
		t.Fatalf("getter de ámbitos DEV sin acto: %v", err)
	}
	if _, err := a.AutorizarConsultaFirmasR5V2(ctx, consulta); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || emisor.base.llamadas != 0 {
		t.Fatalf("consulta con excepción DEV alcanzó V3: err=%v llamadas=%d", err, emisor.base.llamadas)
	}
	if _, err := a.AutorizarRecuperacionFirmasV2(ctx, consulta); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || emisor.base.llamadas != 0 {
		t.Fatalf("recuperación con excepción DEV alcanzó V3: err=%v llamadas=%d", err, emisor.base.llamadas)
	}
}

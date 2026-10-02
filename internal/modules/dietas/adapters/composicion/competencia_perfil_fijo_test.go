package composicion

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type fuentePerfilFijoPrueba struct {
	i               vecdomain.InstantaneaAutorizacion
	err             error
	persona, perfil string
	llamadas        int
}

func (f *fuentePerfilFijoPrueba) ObtenerInstantaneaAutorizacion(_ context.Context, persona, perfil string) (vecdomain.InstantaneaAutorizacion, error) {
	f.persona, f.perfil = persona, perfil
	f.llamadas++
	return f.i, f.err
}

func instantaneaPerfilDietas(t *testing.T, actor vecdomain.ContextoActor, rol string) vecdomain.InstantaneaAutorizacion {
	t.Helper()
	antes := actor.ResueltoEn.Add(-time.Hour)
	v := vecdomain.VersionRol{RolID: rol, Version: 1, Nombre: rol, Estado: vecdomain.EstadoVersionRolPublicada, PublicadaPor: "administrador:ejemplo", PublicadaEn: antes,
		Concesiones: []vecdomain.ConcesionRol{{Accion: "dietas.documento.revisar", ModuloID: "dietas", TipoRecurso: "documento_dietas", Finalidades: []string{"revisar_documento_dietas"}, GarantiaMinima: vecdomain.AuthAssuranceHigh}}}
	huella, err := vecdomain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	i := vecdomain.InstantaneaAutorizacion{
		AsignacionPerfil: vecdomain.AsignacionPerfil{AsignacionID: "asignacion:ejemplo", Version: int(actor.Instantanea.PerfilVersion), PerfilActivoRef: actor.PerfilActivoRef, PrincipalID: actor.PersonaRef, VersionRolRef: v.Referencia(), Estado: vecdomain.EstadoAsignacionPerfilActiva,
			Ambitos: []vecdomain.AmbitoPerfil{{Clave: "persona_ref", Valores: []string{actor.PersonaRef}}, {Clave: "unidad_ref", Valores: []string{"unidad:ejemplo"}}}, VigenteDesde: antes, VigenteHasta: actor.ResueltoEn.Add(time.Hour), EmitidaPor: "administrador:ejemplo", EmitidaEn: antes},
		VersionRol: v, ControlVigenciaVersionRol: vecdomain.ControlVigenciaVersionRol{VersionRolRef: v.Referencia(), Revision: 1, Estado: vecdomain.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: "administrador:ejemplo", ActualizadoEn: antes},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huella,
	}
	if i.Validar() != nil {
		t.Fatal("instantánea de prueba inválida")
	}
	return i
}

func TestCompetenciaPerfilFijoEtapaYUnidadCentrales(t *testing.T) {
	base, _ := identidadYMaterialR15(t)
	for rol, etapa := range map[string]domain.EtapaCircuito{"dietas_revision_administrativa": domain.EtapaRevision, "dietas_autorizacion": domain.EtapaAutorizacion, "dietas_liquidacion_rrhh": domain.EtapaLiquidacion, "dietas_fiscalizacion": domain.EtapaFiscalizacion} {
		t.Run(rol, func(t *testing.T) {
			fuente := &fuentePerfilFijoPrueba{i: instantaneaPerfilDietas(t, base.Contexto.Contexto, rol)}
			f, err := NuevaFuenteCompetenciaPerfilFijo(fuente, relojVinculoR15{base.Contexto.Contexto.ResueltoEn})
			if err != nil {
				t.Fatal(err)
			}
			estado, err := f.EstadoCompetencias(context.Background(), base.Contexto)
			if err != nil || estado.Fuente != dietasports.FuenteCompetenciaAcreditada || len(estado.Etapas) != 1 || estado.Etapas[0] != etapa {
				t.Fatalf("estado=%+v error=%v", estado, err)
			}
			unidad, err := f.UnidadCompetente(context.Background(), base.Contexto, etapa)
			if err != nil || unidad != "unidad:ejemplo" || fuente.persona != base.Contexto.Contexto.PersonaRef || fuente.perfil != base.Contexto.Contexto.PerfilActivoRef {
				t.Fatalf("unidad=%s error=%v", unidad, err)
			}
			if _, err := f.UnidadCompetente(context.Background(), base.Contexto, "otra"); !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) {
				t.Fatalf("otra etapa: %v", err)
			}
			// Cambiar la asignación tras consultar no reutiliza la competencia previa.
			fuente.i.AsignacionPerfil.VigenteHasta = base.Contexto.Contexto.ResueltoEn
			if _, err := f.UnidadCompetente(context.Background(), base.Contexto, etapa); !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) {
				t.Fatalf("vigencia retirada: %v", err)
			}
		})
	}
}

func TestCompetenciaPerfilFijoCierraPorAutoridadInvalida(t *testing.T) {
	base, _ := identidadYMaterialR15(t)
	for nombre, alterar := range map[string]func(*vecdomain.InstantaneaAutorizacion){
		"otra persona":         func(i *vecdomain.InstantaneaAutorizacion) { i.AsignacionPerfil.PrincipalID = "persona:ajena" },
		"otro perfil":          func(i *vecdomain.InstantaneaAutorizacion) { i.AsignacionPerfil.PerfilActivoRef = "perfil:ajeno" },
		"persona ambito ajena": func(i *vecdomain.InstantaneaAutorizacion) { i.AsignacionPerfil.Ambitos[0].Valores[0] = "persona:ajena" },
		"varias unidades": func(i *vecdomain.InstantaneaAutorizacion) {
			i.AsignacionPerfil.Ambitos[1].Valores = []string{"unidad:uno", "unidad:dos"}
		},
		"unidad ausente": func(i *vecdomain.InstantaneaAutorizacion) {
			i.AsignacionPerfil.Ambitos = i.AsignacionPerfil.Ambitos[:1]
		},
		"otro ambito": func(i *vecdomain.InstantaneaAutorizacion) { i.AsignacionPerfil.Ambitos[1].Clave = "departamento" },
		"control retirado": func(i *vecdomain.InstantaneaAutorizacion) {
			i.ControlVigenciaVersionRol.Estado = vecdomain.EstadoControlVigenciaVersionRolRetirada
			i.ControlVigenciaVersionRol.ActoRef = "acto:retirada"
			i.ControlVigenciaVersionRol.MotivoCodigo = "retirada"
		},
		"control futuro": func(i *vecdomain.InstantaneaAutorizacion) {
			i.ControlVigenciaVersionRol.ActualizadoEn = base.Contexto.Contexto.ResueltoEn.Add(time.Minute)
		},
		"asignacion futura": func(i *vecdomain.InstantaneaAutorizacion) {
			i.AsignacionPerfil.VigenteDesde = base.Contexto.Contexto.ResueltoEn.Add(time.Minute)
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			i := instantaneaPerfilDietas(t, base.Contexto.Contexto, "dietas_liquidacion_rrhh")
			alterar(&i)
			f, _ := NuevaFuenteCompetenciaPerfilFijo(&fuentePerfilFijoPrueba{i: i}, relojVinculoR15{base.Contexto.Contexto.ResueltoEn})
			if unidad, err := f.UnidadCompetente(context.Background(), base.Contexto, domain.EtapaLiquidacion); unidad != "" || !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) {
				t.Fatalf("unidad=%s error=%v", unidad, err)
			}
		})
	}
}

func TestCompetenciaPerfilFijoNoSumaPerfilesYRespetaCancelacion(t *testing.T) {
	base, _ := identidadYMaterialR15(t)
	fuente := &fuentePerfilFijoPrueba{i: instantaneaPerfilDietas(t, base.Contexto.Contexto, "empleado")}
	f, _ := NuevaFuenteCompetenciaPerfilFijo(fuente, relojVinculoR15{base.Contexto.Contexto.ResueltoEn})
	estado, err := f.EstadoCompetencias(context.Background(), base.Contexto)
	if err != nil || len(estado.Etapas) != 0 {
		t.Fatalf("perfil solicitante=%+v %v", estado, err)
	}
	if _, err := f.UnidadCompetente(context.Background(), base.Contexto, domain.EtapaLiquidacion); !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) {
		t.Fatal(err)
	}
	fuente.err = errors.New("fuente caída")
	if _, err := f.EstadoCompetencias(context.Background(), base.Contexto); !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) || !errors.Is(err, fuente.err) {
		t.Fatal(err)
	}
	previas := fuente.llamadas
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := f.EstadoCompetencias(ctx, base.Contexto); !errors.Is(err, context.Canceled) || fuente.llamadas != previas {
		t.Fatal("cancelación consultó la fuente")
	}
	if _, err := NuevaFuenteCompetenciaPerfilFijo(nil, relojVinculoR15{}); err == nil {
		t.Fatal("fuente nula aceptada")
	}
}

func TestCompetenciaVersionAsignacionNoEsVersionPerfilIdentidad(t *testing.T) {
	base, _ := identidadYMaterialR15(t)
	i := instantaneaPerfilDietas(t, base.Contexto.Contexto, "dietas_liquidacion_rrhh")
	i.AsignacionPerfil.Version = 4
	f, _ := NuevaFuenteCompetenciaPerfilFijo(&fuentePerfilFijoPrueba{i: i}, relojVinculoR15{base.Contexto.Contexto.ResueltoEn})
	unidad, err := f.UnidadCompetente(context.Background(), base.Contexto, domain.EtapaLiquidacion)
	if err != nil || unidad != "unidad:ejemplo" {
		t.Fatalf("asignación v4 / perfil v1: unidad=%s error=%v", unidad, err)
	}
}

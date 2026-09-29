package bootstrap

import (
	"context"
	"testing"

	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"

	"vec-diputacion-granada/config"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	usuariosports "vec-diputacion-granada/internal/modules/usuarios/ports"
)

func TestAudienciaAvisosMisCorreosEnGobiernoYSinColisiones(t *testing.T) {
	d := descriptorMaterialCorreoAvisosDesarrollo()
	if d.Audiencia != usuariosports.AudienciaCorreoAvisosLlamamientoInterna {
		t.Fatalf("audiencia: %s", d.Audiencia)
	}
	if !audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(d.Audiencia) {
		t.Fatal("la audiencia de avisos debe estar en el gobierno común")
	}
	todos := append(append(append(descriptoresMaterialPreferenciasUsuariosDesarrollo(), descriptoresMaterialCorreosUsuariosDesarrollo()...), descriptoresMaterialBorradorLlamamientoBolsaDesarrollo()...), d)
	if _, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(todos); err != nil {
		t.Fatal("la audiencia de avisos repite dominio, prefijo o audiencia")
	}
	if d.Audiencia == puertosbolsa.AudienciaEmitirLlamamiento {
		t.Fatal("la lectura no puede compartir audiencia con la emisión")
	}
}

func TestAvisosMisCorreosApagadoNoCompone(t *testing.T) {
	t.Setenv(envBolsaAvisosMisCorreosDesarrollo, "")
	previo := false
	cerrar, err := componerAvisosMisCorreosBolsaDesarrollo(context.Background(), config.Config{}, nil, nil, nil, nil, nil, func() { previo = true })
	if err != nil || cerrar == nil {
		t.Fatalf("apagado debe ser un no-op: %v", err)
	}
	if cerrar(); !previo {
		t.Fatal("apagado debe conservar el cierre anterior")
	}
	t.Setenv(envBolsaAvisosMisCorreosDesarrollo, "si")
	if cerrar, err := componerAvisosMisCorreosBolsaDesarrollo(context.Background(), config.Config{}, nil, nil, nil, nil, nil, nil); err == nil || cerrar == nil {
		t.Fatal("un valor distinto de true/false no arranca y el cierre nunca es nulo")
	}
}

func TestPuenteAvisosSinDependenciasNoLlamaAUsar(t *testing.T) {
	var p *puenteCorreoAvisosBolsaDesarrollo
	llamado := false
	if ok, _, err := p.ConCorreoAvisoPersona(context.Background(), puertosbolsa.SolicitudCorreoAvisoPersona{}, func(string) { llamado = true }); ok || err == nil || llamado {
		t.Fatal("puente nulo")
	}
	var proveedor *proveedorV3CorreoAvisosDesarrollo
	if _, err := proveedor.ProveerMaterialCorreoAvisos(context.Background(), usuariosports.MaterialCorreoAvisos{}, nil); err == nil {
		t.Fatal("proveedor nulo")
	}
	vacio := &proveedorV3CorreoAvisosDesarrollo{emisor: emisorAvisosNoLlamarPrueba{t}}
	if _, err := vacio.ProveerMaterialCorreoAvisos(context.Background(), usuariosports.MaterialCorreoAvisos{}, nil); err == nil {
		t.Fatal("sin identidad de quien emite no hay V3")
	}
}

type emisorAvisosNoLlamarPrueba struct{ t *testing.T }

func (e emisorAvisosNoLlamarPrueba) EmitirMaterialAutorizacionAtestadaV3(context.Context, core.SolicitudAutorizacionLigadaV3, core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.t.Fatal("no debe pedirse V3 sin identidad válida")
	return core.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, nil
}

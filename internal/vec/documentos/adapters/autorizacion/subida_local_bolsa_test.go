package autorizacion

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type emisorSubidaBolsaPrueba struct{}

func (emisorSubidaBolsaPrueba) EmitirMaterialAutorizacionAtestadaV3(context.Context,
	vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2,
) (vecdomain.DecisionAutorizacionLigadaV3,
	vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecdomain.DecisionAutorizacionLigadaV3{},
		vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, nil
}

type vinculoSubidaBolsaPrueba struct{}

func (vinculoSubidaBolsaPrueba) VinculoCargaLocalCertificado(context.Context,
	vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.VinculoAutenticacionActorV2, error) {
	return vecdomain.VinculoAutenticacionActorV2{}, nil
}

func TestAutoridadSubidaLocalBolsaMotivoYAccionesAislados(t *testing.T) {
	motivo := vecdomain.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion_justificante_renuncia_bolsa", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "motivo_" + strings.Repeat("b", 32),
	}
	emisor, vinculo := emisorSubidaBolsaPrueba{}, vinculoSubidaBolsaPrueba{}
	reloj := relojFijo{time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)}
	bolsa, err := NuevaAutoridadSubidaLocalBolsa(emisor, vinculo, motivo, reloj)
	if err != nil || bolsa.modulo != "bolsa" {
		t.Fatalf("autoridad Bolsa: %v", err)
	}
	if _, err := NuevaAutoridadSubidaLocal(emisor, vinculo, motivo, reloj); err == nil {
		t.Fatal("autoridad Méritos aceptó motivo Bolsa")
	}
	motivo.CatalogoID = "motivos_autorizacion_meritos_cartera"
	if _, err := NuevaAutoridadSubidaLocalBolsa(emisor, vinculo, motivo, reloj); err == nil {
		t.Fatal("autoridad Bolsa aceptó motivo Méritos")
	}
	meritos, err := NuevaAutoridadSubidaLocal(emisor, vinculo, motivo, reloj)
	if err != nil || meritos.modulo != "meritos" {
		t.Fatalf("autoridad Méritos alterada: %v", err)
	}
	casos := map[string]string{
		docports.AccionBolsaJustificanteRenunciaSubirLocal:     audienciaSubidaBolsa,
		docports.AccionBolsaJustificanteRenunciaDescargarLocal: audienciaDescargaPropiaBolsa,
		docports.AccionBolsaJustificanteRenunciaDescargarRRHH:  audienciaDescargaRRHHBolsa,
	}
	for accion, audiencia := range casos {
		modulo, tipo, finalidad, obtenida, ok := parametrosAccionLocal(accion)
		if !ok || modulo != "bolsa" || tipo != "justificante_renuncia" ||
			finalidad != "custodiar_justificante_renuncia_bolsa" || obtenida != audiencia {
			t.Fatalf("acción %s: audiencia %s", accion, obtenida)
		}
	}
	if _, _, _, _, ok := parametrosAccionLocal(docports.AccionDietasJustificanteSubirLocal); ok {
		t.Fatal("Dietas habilitada sin autoridad nominal")
	}
	actor, vinculoCert, err := pruebas.NuevoContextoRegistradoYVinculoV2(
		reloj.instante, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl",
		vecdomain.AuthMethodCertificate, vecdomain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	id := "ref:" + strings.Repeat("1", 64)
	corr := "correlacion_" + strings.Repeat("2", 32)
	pre, _ := json.Marshal(map[string]string{
		"accion": docports.AccionMeritosComprobanteSubirLocal, "carga_id": id,
		"persona_ref": actor.Contexto.PersonaRef, "correlacion_ref": corr,
	})
	if _, err := bolsa.AutorizarSubidaLocal(context.Background(), pre, actor, id, id,
		"custodiar_evidencia_meritos", corr); err == nil {
		t.Fatal("autoridad Bolsa emitió acción de Méritos")
	}
	pre, _ = json.Marshal(map[string]string{
		"accion": docports.AccionBolsaJustificanteRenunciaSubirLocal, "carga_id": id,
		"persona_ref": actor.Contexto.PersonaRef, "correlacion_ref": corr,
	})
	if _, err := meritos.AutorizarSubidaLocal(context.Background(), pre, actor, id, id,
		"custodiar_justificante_renuncia_bolsa", corr); err == nil {
		t.Fatal("autoridad Méritos emitió acción de Bolsa")
	}
	q := docports.SolicitudOriginalSubidaLocalRRHH{ActorRegistradoV2: actor, VinculoV2: vinculoCert,
		DocumentoID: id, HuellaPDFSHA256: strings.Repeat("a", 64), RecursoRef: id,
		PersonaTitularRef: "per_abcdefghijkl0123456789", BolsaRef: "bolsa_1",
		ParticipacionRef: "participacion_1", CorrelacionRef: corr}
	pre, err = docports.PreimagenConsultaSubidaLocalRRHH(q)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := meritos.AutorizarConsultaRRHH(context.Background(), pre, q); err == nil {
		t.Fatal("autoridad Méritos emitió lectura RRHH Bolsa")
	}
}

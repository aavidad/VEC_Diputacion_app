package controlrestauracionpg

import (
	"context"
	"strings"
	"testing"

	ordenpg "vec-diputacion-granada/internal/modules/administracion/adapters/ordenescopias/postgres"
	dominio "vec-diputacion-granada/internal/modules/administracion/domain/ordenescopias"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type materializadorPrueba struct{ llamadas int }

func (m *materializadorPrueba) MaterializarOrdenV3(
	context.Context,
	dominio.Orden,
	vecdomain.SolicitudAutorizacionLigadaV3,
	vecdomain.DecisionAutorizacionLigadaV3,
	vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
) (ordenpg.MaterialV3, error) {
	m.llamadas++
	return ordenpg.MaterialV3{}, nil
}

func TestProponerRechazaOrdenCeroAntesDeMaterializar(t *testing.T) {
	materializador := &materializadorPrueba{}
	registro := &Registro{material: materializador}
	if _, err := registro.Proponer(context.Background(), dominio.Orden{}, vecdomain.SolicitudAutorizacionLigadaV3{}, vecdomain.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}); !errorsIguales(err, ErrRegistro) {
		t.Fatalf("error=%v", err)
	}
	if materializador.llamadas != 0 {
		t.Fatalf("materializador llamado %d veces", materializador.llamadas)
	}
}

func TestResultadoDesdeSQLExigeFormaCerrada(t *testing.T) {
	valido := `{"orden":"orden:uno","plan_sha256":"` + strings.Repeat("a", 64) + `","persona_ref":"per_uno","version":1,"recibo":"recibo:uno","registrada_en":"2026-10-01T12:00:00.000000Z","replay":false}`
	resultado, err := resultadoDesdeSQL([]byte(valido))
	if err != nil || resultado.Orden != "orden:uno" || resultado.PlanSHA256 != strings.Repeat("a", 64) || resultado.RegistradaEn.Location().String() != "UTC" {
		t.Fatalf("resultado=%+v err=%v", resultado, err)
	}
	for _, bruto := range [][]byte{
		[]byte(`{"orden":"orden:uno","plan_sha256":"` + strings.Repeat("a", 64) + `","persona_ref":"per_uno","version":1,"recibo":"recibo:uno","registrada_en":"2026-10-01T12:00:00.000000Z","replay":false,"extra":true}`),
		[]byte(`{"orden":"orden:uno","orden":"orden:dos","plan_sha256":"` + strings.Repeat("a", 64) + `","persona_ref":"per_uno","version":1,"recibo":"recibo:uno","registrada_en":"2026-10-01T12:00:00.000000Z","replay":false}`),
		[]byte(`{"orden":"orden:uno","plan_sha256":"` + strings.Repeat("a", 64) + `","persona_ref":"per_uno","version":3,"recibo":"recibo:uno","registrada_en":"2026-10-01T12:00:00.000000Z","replay":false} x`),
		[]byte(`{"orden":"orden:uno","plan_sha256":"invalida","persona_ref":"per_uno","version":1,"recibo":"recibo:uno","registrada_en":"2026-10-01T12:00:00.000000Z","replay":false}`),
	} {
		if _, err := resultadoDesdeSQL(bruto); !errorsIguales(err, ErrRegistro) {
			t.Fatalf("respuesta invalida aceptada: %s; err=%v", bruto, err)
		}
	}
}

func TestContratoSQLDePropuestaYRevision(t *testing.T) {
	if accionProponer != "copias_restauracion_proponer" || finalidadProponer != "proponer_restauracion_copia" ||
		strings.Count(registrarPropuestaSQL, "$") != 11 {
		t.Fatalf("contrato de propuesta alterado")
	}
	if accionRevisar != "copias_restauracion_revisar" || finalidadRevisar != "revisar_restauracion_copia" ||
		strings.Count(registrarRevisionSQL, "$") != 12 {
		t.Fatalf("contrato de revision alterado")
	}
}

func TestRevisarCASRechazaVersionCeroAntesDeMaterializar(t *testing.T) {
	materializador := &materializadorPrueba{}
	registro := &Registro{material: materializador}
	if _, err := registro.RevisarCAS(context.Background(), dominio.Orden{}, 0, vecdomain.SolicitudAutorizacionLigadaV3{}, vecdomain.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}); !errorsIguales(err, ErrRegistro) {
		t.Fatalf("error=%v", err)
	}
	if materializador.llamadas != 0 {
		t.Fatalf("materializador llamado %d veces", materializador.llamadas)
	}
}

func errorsIguales(obtenido, esperado error) bool { return obtenido == esperado }

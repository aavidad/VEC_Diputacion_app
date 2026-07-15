package domain

import (
	"errors"
	"reflect"
	"testing"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

func TestConvocatoriaEsAliasDelAgregadoCanonicoBolsa(t *testing.T) {
	convocatoria, err := NewConvocatoria("conv-compat", "v1")
	if err != nil {
		t.Fatal(err)
	}
	var canonica dominiobolsa.Convocatoria = convocatoria
	if canonica.ID != convocatoria.ID || canonica.Estado != ProcedureStateBorrador {
		t.Fatalf("alias con perdida: %#v", canonica)
	}
}

func TestConvocatoriaVersionsRemainSameCall(t *testing.T) {
	call, err := NewConvocatoria(" conv-1 ", " v1 ")
	if err != nil {
		t.Fatalf("NewConvocatoria() error = %v", err)
	}
	next, err := call.NewVersion("v2")
	if err != nil {
		t.Fatalf("NewVersion() error = %v", err)
	}
	if call.ID != "conv-1" || call.Version != "v1" || call.Estado != ProcedureStateBorrador {
		t.Fatalf("call = %#v, want trimmed draft v1", call)
	}
	if next.ID != call.ID || next.Version != "v2" || next.Estado != ProcedureStateBorrador {
		t.Fatalf("next = %#v, want same id draft v2", next)
	}
	if _, err := call.NewVersion("v1"); !errors.Is(err, ErrProcedureInvalid) {
		t.Fatalf("NewVersion(duplicate) error = %v, want %v", err, ErrProcedureInvalid)
	}
}

func TestSolicitudStatesCoverAdministrativeCycle(t *testing.T) {
	state := SolicitudStateBorrador
	for _, next := range []SolicitudState{
		SolicitudStateInscrita,
		SolicitudStateSubsanacionRequerida,
		SolicitudStateSubsanada,
		SolicitudStateAdmitidaProvisional,
		SolicitudStateAlegacionPresentada,
		SolicitudStateAdmitidaDefinitiva,
	} {
		var err error
		state, err = state.Transition(next)
		if err != nil {
			t.Fatalf("Transition(%s) error = %v", next, err)
		}
	}
	if state != SolicitudStateAdmitidaDefinitiva {
		t.Fatalf("state = %s, want %s", state, SolicitudStateAdmitidaDefinitiva)
	}
	if _, err := SolicitudStateBorrador.Transition(SolicitudStateAdmitidaDefinitiva); !errors.Is(err, ErrProcedureTransition) {
		t.Fatalf("invalid solicitud transition error = %v, want %v", err, ErrProcedureTransition)
	}
	if _, err := SolicitudStateBorrador.Transition(SolicitudState("x")); !errors.Is(err, ErrProcedureInvalid) {
		t.Fatalf("invalid solicitud state error = %v, want %v", err, ErrProcedureInvalid)
	}
}

func TestBolsaStatesCoverAdministrativeCycle(t *testing.T) {
	state := BolsaStateSinConstituir
	for _, next := range []BolsaState{
		BolsaStateProvisional,
		BolsaStateEnAlegaciones,
		BolsaStateDefinitiva,
		BolsaStateAgotada,
		BolsaStateCerrada,
	} {
		var err error
		state, err = state.Transition(next)
		if err != nil {
			t.Fatalf("Transition(%s) error = %v", next, err)
		}
	}
	if _, err := BolsaStateSinConstituir.Transition(BolsaStateDefinitiva); !errors.Is(err, ErrProcedureTransition) {
		t.Fatalf("invalid bolsa transition error = %v, want %v", err, ErrProcedureTransition)
	}
}

func TestRankSolicitudesIsDeterministicAndExplainsOrder(t *testing.T) {
	ruleSet := mustBaremoRuleSet(t, []BaremoTieBreakRule{BaremoTieMayorExperiencia, BaremoTieLetraSorteo})
	input := []SolicitudRankingEntry{
		rankingEntry("sol-3", "cand-c", "Ruiz", 8, 7, 15),
		rankingEntry("sol-2", "cand-b", "Navas", 7, 8, 15),
		rankingEntry("sol-1", "cand-a", "Lopez", 7, 8, 15),
		rankingEntry("sol-4", "cand-d", "Perez", 5, 2, 7),
	}
	got, err := RankSolicitudes(input, ruleSet)
	if err != nil {
		t.Fatalf("RankSolicitudes() error = %v", err)
	}
	reversed := []SolicitudRankingEntry{input[3], input[2], input[1], input[0]}
	gotAgain, err := RankSolicitudes(reversed, ruleSet)
	if err != nil {
		t.Fatalf("RankSolicitudes(reversed) error = %v", err)
	}
	if !reflect.DeepEqual(got, gotAgain) {
		t.Fatalf("ranking should be deterministic\nfirst: %#v\nsecond: %#v", got, gotAgain)
	}

	wantIDs := []string{"cand-c", "cand-b", "cand-a", "cand-d"}
	for i, wantID := range wantIDs {
		if got[i].Position != i+1 || got[i].CandidateID != wantID {
			t.Fatalf("rank %d = %#v, want candidate %s", i+1, got[i], wantID)
		}
	}
	if got[1].PreviousOrderReason != "previous wins configured tie break" || len(got[1].PreviousTieDecisions) != 1 {
		t.Fatalf("second explanation = %#v, want experience tie break", got[1])
	}
	if got[2].PreviousOrderReason != "previous wins configured tie break" || len(got[2].PreviousTieDecisions) != 2 {
		t.Fatalf("third explanation = %#v, want lottery tie break", got[2])
	}
	if got[3].PreviousOrderReason != "previous has greater total points" {
		t.Fatalf("fourth reason = %q, want total points", got[3].PreviousOrderReason)
	}
}

func TestRankSolicitudesRejectsInvalidEntries(t *testing.T) {
	ruleSet := mustBaremoRuleSet(t, nil)
	_, err := RankSolicitudes([]SolicitudRankingEntry{
		rankingEntry("sol-1", "cand-a", "Lopez", 1, 1, 2),
		{SolicitudID: "sol-2", CandidateID: "cand-b", Estado: SolicitudStateExcluidaDefinitiva},
	}, ruleSet)
	if !errors.Is(err, ErrProcedureRanking) {
		t.Fatalf("RankSolicitudes() error = %v, want %v", err, ErrProcedureRanking)
	}
}

func rankingEntry(id, candidateID, sorteoKey string, experiencia, formacion, total float64) SolicitudRankingEntry {
	return SolicitudRankingEntry{
		SolicitudID: id,
		CandidateID: candidateID,
		Estado:      SolicitudStateAdmitidaDefinitiva,
		SorteoKey:   sorteoKey,
		Result: BaremoResult{
			TotalPoints: total,
			SectionPoints: map[BaremoSection]float64{
				BaremoSectionExperiencia: experiencia,
				BaremoSectionFormacion:   formacion,
			},
		},
	}
}

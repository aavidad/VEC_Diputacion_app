package repository

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
)

func TestMemoryRepositoriesSaveAndListDeterministically(t *testing.T) {
	ctx := context.Background()
	candidates, merits := NewRepositories()

	if err := candidates.Save(ctx, "call-1", candidateFixture("cand-2")); err != nil {
		t.Fatalf("Save(candidate cand-2) error = %v", err)
	}
	if err := candidates.Save(ctx, "call-1", candidateFixture("cand-1")); err != nil {
		t.Fatalf("Save(candidate cand-1) error = %v", err)
	}
	gotCandidate, gotCallID, err := candidates.GetByID(ctx, "cand-1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if gotCandidate.ID != "cand-1" || gotCallID != "call-1" {
		t.Fatalf("GetByID() = %q/%q, want cand-1/call-1", gotCandidate.ID, gotCallID)
	}

	gotCandidates, err := candidates.ListByCall(ctx, "call-1")
	if err != nil {
		t.Fatalf("ListByCall() error = %v", err)
	}
	if gotIDs := candidateIDs(gotCandidates); !reflect.DeepEqual(gotIDs, []string{"cand-1", "cand-2"}) {
		t.Fatalf("candidate IDs = %#v, want sorted cand-1/cand-2", gotIDs)
	}

	if err := merits.Save(ctx, "cand-1", meritFixture("m2")); err != nil {
		t.Fatalf("Save(merit m2) error = %v", err)
	}
	if err := merits.Save(ctx, "cand-1", meritFixture("m1")); err != nil {
		t.Fatalf("Save(merit m1) error = %v", err)
	}
	gotMerits, err := merits.ListByCandidate(ctx, "cand-1")
	if err != nil {
		t.Fatalf("ListByCandidate() error = %v", err)
	}
	if gotIDs := meritIDs(gotMerits); !reflect.DeepEqual(gotIDs, []string{"m1", "m2"}) {
		t.Fatalf("merit IDs = %#v, want sorted m1/m2", gotIDs)
	}
}

func TestMemoryCandidateRepositoryMovesCandidateBetweenCalls(t *testing.T) {
	ctx := context.Background()
	candidates := NewCandidateRepository(nil)

	if err := candidates.Save(ctx, "call-old", candidateFixture("cand-1")); err != nil {
		t.Fatalf("Save(old call) error = %v", err)
	}
	if err := candidates.Save(ctx, "call-new", candidateFixture("cand-1")); err != nil {
		t.Fatalf("Save(new call) error = %v", err)
	}

	oldCall, err := candidates.ListByCall(ctx, "call-old")
	if err != nil {
		t.Fatalf("ListByCall(old) error = %v", err)
	}
	if len(oldCall) != 0 {
		t.Fatalf("old call candidates = %d, want 0", len(oldCall))
	}
	newCall, err := candidates.ListByCall(ctx, "call-new")
	if err != nil {
		t.Fatalf("ListByCall(new) error = %v", err)
	}
	if gotIDs := candidateIDs(newCall); !reflect.DeepEqual(gotIDs, []string{"cand-1"}) {
		t.Fatalf("new call candidate IDs = %#v, want cand-1", gotIDs)
	}
}

func TestMemoryCandidateRepositoryNotFound(t *testing.T) {
	_, _, err := NewCandidateRepository(nil).GetByID(context.Background(), "missing")
	if !errors.Is(err, ports.ErrCandidateNotFound) {
		t.Fatalf("GetByID() error = %v, want %v", err, ports.ErrCandidateNotFound)
	}
}

func TestMemoryCandidateRepositoryRejectsImplicitOrWildcardCall(t *testing.T) {
	repository := NewCandidateRepository(nil)
	for _, callID := range []string{"", " call-1 ", "*", "call-*"} {
		if err := repository.Save(context.Background(), callID, candidateFixture("cand-1")); !errors.Is(err, ports.ErrCandidateCallInvalid) {
			t.Fatalf("Save(call=%q) error = %v, want %v", callID, err, ports.ErrCandidateCallInvalid)
		}
		if _, err := repository.ListByCall(context.Background(), callID); !errors.Is(err, ports.ErrCandidateCallInvalid) {
			t.Fatalf("ListByCall(call=%q) error = %v, want %v", callID, err, ports.ErrCandidateCallInvalid)
		}
	}
}

func TestMemoryRepositoriesSupportConcurrentSaveAndList(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	candidates := NewCandidateRepository(store)
	merits := NewMeritRepository(store)

	const total = 40
	var wg sync.WaitGroup
	errs := make(chan error, total*2)
	for i := 0; i < total; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			candidateID := fmt.Sprintf("cand-%02d", i)
			if err := candidates.Save(ctx, "call-1", candidateFixture(candidateID)); err != nil {
				errs <- err
				return
			}
			if err := merits.Save(ctx, candidateID, meritFixture(fmt.Sprintf("m-%02d", i))); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent save error = %v", err)
		}
	}

	gotCandidates, err := candidates.ListByCall(ctx, "call-1")
	if err != nil {
		t.Fatalf("ListByCall() error = %v", err)
	}
	if len(gotCandidates) != total {
		t.Fatalf("candidates = %d, want %d", len(gotCandidates), total)
	}
	for _, candidate := range gotCandidates {
		gotMerits, err := merits.ListByCandidate(ctx, candidate.ID)
		if err != nil {
			t.Fatalf("ListByCandidate(%q) error = %v", candidate.ID, err)
		}
		if len(gotMerits) != 1 {
			t.Fatalf("merits for %q = %d, want 1", candidate.ID, len(gotMerits))
		}
	}
}

func candidateFixture(id string) domain.Candidate {
	return domain.Candidate{
		ID:     id,
		DNI:    id + "-dni",
		Nombre: "Nombre " + id,
		Email:  id + "@example.test",
	}
}

func meritFixture(id string) domain.Merit {
	return domain.Merit{
		ID:     id,
		Tipo:   domain.MeritTypeFormacionCurso,
		Datos:  domain.MeritData{Horas: 20},
		Estado: domain.MeritStateValidado,
	}
}

func candidateIDs(candidates []domain.Candidate) []string {
	ids := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.ID)
	}
	return ids
}

func meritIDs(merits []domain.Merit) []string {
	ids := make([]string, 0, len(merits))
	for _, merit := range merits {
		ids = append(ids, merit.ID)
	}
	return ids
}

package bootstrap

import (
	"context"
	"errors"
	"sync"
	"testing"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// bolsasVigentesFalsas hace de Bolsa: la carga CONVOCA constituye una bolsa y
// a partir de ahí la consulta la da por vigente.
type bolsasVigentesFalsas struct {
	mu        sync.Mutex
	vigentes  map[string]bool
	consultas int
	fallo     error
}

func (f *bolsasVigentesFalsas) constituir(ref string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.vigentes[ref] = true
}

func (f *bolsasVigentesFalsas) BolsaConstituidaVigente(_ context.Context, ref string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.consultas++
	if f.fallo != nil {
		return false, f.fallo
	}
	return f.vigentes[ref], nil
}

func preparadorConBolsasVigentes(t *testing.T) (*preparadorBorradorLlamamientoDesarrollo, *bolsasVigentesFalsas, dominiovec.ContextoActor) {
	t.Helper()
	directorio, soporteCT, principal, ahora := fixtureSoporteSesionBorradorBolsa(t)
	escribirManifiestoIdentidadBorradorBolsa(t, directorio, principal, ahora, nil)
	soporte, err := nuevoSoporteSesionBorradorBolsaDesarrollo(directorio, soporteCT, ahora)
	if err != nil {
		t.Fatal(err)
	}
	vigentes := &bolsasVigentesFalsas{vigentes: map[string]bool{}}
	return &preparadorBorradorLlamamientoDesarrollo{soporte: soporte, vigentes: vigentes}, vigentes, soporte.soporteCanal.contexto.Resultado.Contexto
}

// Fallo 1 del recorrido del 10/10: la bolsa recién cargada desde CONVOCA no
// está en bolsas_ref. Antes de la carga se deniega; después, ver candidatos,
// abrir la ficha y emitir el llamamiento resuelven con la unidad y el ámbito
// del perfil, sin tocar el manifiesto ni reiniciar.
func TestBolsaCargadaDesdeConvocaAdmiteCandidatosFichaYEmision(t *testing.T) {
	preparador, bolsas, actor := preparadorConBolsasVigentes(t)
	const nueva = "bolsa:programador:nadclofbjbhdhoeopeljdcdkahcpmmdh"
	ctx := context.Background()
	if _, err := preparador.ResolverContextoContactosBolsa(ctx, actor, nueva); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("antes de la carga debía denegarse: %v", err)
	}
	bolsas.constituir(nueva)
	// Candidatos y emisión usan ResolverContextoContactosBolsa; la ficha y
	// los contactos de una persona, ResolverContextoSituacionParticipacion.
	candidatos, err := preparador.ResolverContextoContactosBolsa(ctx, actor, nueva)
	if err != nil || candidatos.UnidadRef != preparador.soporte.unidadRef || candidatos.AmbitoRef != preparador.soporte.ambitoRef {
		t.Fatalf("candidatos: %+v err=%v", candidatos, err)
	}
	ficha, err := preparador.ResolverContextoSituacionParticipacion(ctx, actor, nueva, "participacion:convoca:1")
	if err != nil || ficha != candidatos {
		t.Fatalf("ficha: %+v err=%v", ficha, err)
	}
	emision, err := preparador.ResolverContextoContactosBolsa(ctx, actor, nueva)
	if err != nil || emision != candidatos {
		t.Fatalf("emisión: %+v err=%v", emision, err)
	}
}

func TestAdmisionBolsaDeniegaLoQueBolsaNoTieneVigente(t *testing.T) {
	preparador, bolsas, actor := preparadorConBolsasVigentes(t)
	ctx := context.Background()
	if _, err := preparador.ResolverContextoSituacionParticipacion(ctx, actor, "bolsa:ajena", "participacion:x"); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("bolsa desconocida o extinguida no denegada: %v", err)
	}
	// La lista del manifiesto sigue valiendo sin preguntar a la base.
	antes := bolsas.consultas
	if _, err := preparador.ResolverContextoContactosBolsa(ctx, actor, "bolsa:b2:desarrollo"); err != nil {
		t.Fatalf("bolsa del manifiesto rechazada: %v", err)
	}
	if bolsas.consultas != antes {
		t.Fatal("la bolsa del manifiesto no debía consultar la base")
	}
	// Sin consulta compuesta solo vale el manifiesto.
	preparador.vigentes = nil
	bolsas.constituir("bolsa:nueva")
	if _, err := preparador.ResolverContextoContactosBolsa(ctx, actor, "bolsa:nueva"); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("sin consulta debía denegarse: %v", err)
	}
}

// Si la base no responde, el acto no sigue y tampoco se presenta como una
// denegación de permiso: es un servicio no disponible.
func TestAdmisionBolsaFallaCerradaSiLaBaseNoResponde(t *testing.T) {
	preparador, bolsas, actor := preparadorConBolsasVigentes(t)
	bolsas.fallo = errors.New("caída")
	_, err := preparador.ResolverContextoContactosBolsa(context.Background(), actor, "bolsa:nueva")
	if err == nil || errors.Is(err, dominiovec.ErrAutorizacionDenegada) || !errors.Is(err, errBorradorLlamamientoDesarrolloNoDisponible) {
		t.Fatalf("err=%v", err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := preparador.ResolverContextoContactosBolsa(ctx, actor, "bolsa:nueva"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelada: %v", err)
	}
}

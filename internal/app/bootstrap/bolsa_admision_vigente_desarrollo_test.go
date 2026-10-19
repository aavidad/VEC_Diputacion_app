package bootstrap

import (
	"context"
	"errors"
	"sync"
	"testing"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// bolsasVigentesFalsas hace de Bolsa: la carga CONVOCA constituye una bolsa y
// a partir de ahí la consulta la da por vigente, con el ámbito de la carga.
type bolsasVigentesFalsas struct {
	mu          sync.Mutex
	vigentes    map[string]bool
	ambitos     map[string]puertosbolsa.AmbitoBolsa
	consultas   int
	fallo       error
	falloAmbito error
}

func (f *bolsasVigentesFalsas) constituir(ref string) {
	f.constituirEn(ref, puertosbolsa.AmbitoBolsa{AmbitoRef: "ambito:desarrollo:bolsa", UnidadRef: "unidad:desarrollo:rrhh"})
}

func (f *bolsasVigentesFalsas) constituirEn(ref string, ambito puertosbolsa.AmbitoBolsa) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.vigentes[ref] = true
	f.declarar(ref, ambito)
}

func (f *bolsasVigentesFalsas) declarar(ref string, ambito puertosbolsa.AmbitoBolsa) {
	if f.ambitos == nil {
		f.ambitos = map[string]puertosbolsa.AmbitoBolsa{}
	}
	f.ambitos[ref] = ambito
}

func (f *bolsasVigentesFalsas) AmbitoBolsa(_ context.Context, ref string) (puertosbolsa.AmbitoBolsa, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.falloAmbito != nil {
		return puertosbolsa.AmbitoBolsa{}, f.falloAmbito
	}
	a, ok := f.ambitos[ref]
	if !ok {
		return puertosbolsa.AmbitoBolsa{}, puertosbolsa.ErrAmbitoBolsaNoDeclarado
	}
	return a, nil
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
	// La semilla de la copia declara las bolsas del manifiesto (B144).
	vigentes.declarar("bolsa:b2:desarrollo", puertosbolsa.AmbitoBolsa{AmbitoRef: soporte.ambitoRef, UnidadRef: soporte.unidadRef})
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
	// Sin consulta compuesta no hay ámbito y se deniega todo, también el
	// manifiesto.
	preparador.vigentes = nil
	bolsas.constituir("bolsa:nueva")
	for _, ref := range []string{"bolsa:nueva", "bolsa:b2:desarrollo"} {
		if _, err := preparador.ResolverContextoContactosBolsa(ctx, actor, ref); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
			t.Fatalf("sin consulta debía denegarse %s: %v", ref, err)
		}
	}
}

// F1-60: el recurso lleva la unidad y el ámbito de la bolsa, no los del
// perfil. Una bolsa de otra unidad resuelve con esa unidad (la decisión V3 la
// denegará) y una bolsa sin ámbito declarado se deniega aquí mismo, aunque esté
// vigente o figure en el manifiesto.
func TestContextoBolsaUsaElAmbitoGobernadoDeLaBolsa(t *testing.T) {
	preparador, bolsas, actor := preparadorConBolsasVigentes(t)
	ctx := context.Background()
	ajena := puertosbolsa.AmbitoBolsa{AmbitoRef: "ambito:otra:bolsa", UnidadRef: "unidad:otra:rrhh"}
	bolsas.constituirEn("bolsa:otra-unidad", ajena)
	resuelto, err := preparador.ResolverContextoSituacionParticipacion(ctx, actor, "bolsa:otra-unidad", "participacion:x")
	if err != nil || resuelto.UnidadRef != ajena.UnidadRef || resuelto.AmbitoRef != ajena.AmbitoRef {
		t.Fatalf("otra unidad: %+v err=%v", resuelto, err)
	}
	if resuelto.UnidadRef == preparador.soporte.unidadRef {
		t.Fatal("el recurso no puede tomar la unidad del perfil")
	}
	bolsas.mu.Lock()
	bolsas.vigentes["bolsa:sin-ambito"] = true
	bolsas.mu.Unlock()
	if _, err := preparador.ResolverContextoContactosBolsa(ctx, actor, "bolsa:sin-ambito"); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("bolsa vigente sin ámbito no denegada: %v", err)
	}
	preparador.soporte.bolsasRef["bolsa:manifiesto-sin-ambito"] = struct{}{}
	if _, err := preparador.ResolverContextoSituacionParticipacion(ctx, actor, "bolsa:manifiesto-sin-ambito", "participacion:x"); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("bolsa del manifiesto sin ámbito no denegada: %v", err)
	}
	bolsas.falloAmbito = errors.New("caída")
	if _, err := preparador.ResolverContextoContactosBolsa(ctx, actor, "bolsa:b2:desarrollo"); err == nil ||
		errors.Is(err, dominiovec.ErrAutorizacionDenegada) || !errors.Is(err, errBorradorLlamamientoDesarrolloNoDisponible) {
		t.Fatalf("lectura del ámbito caída: %v", err)
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

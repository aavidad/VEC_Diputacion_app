package ejecucioncopias

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	cs07 "vec-diputacion-granada/internal/modules/administracion/adapters/registrocopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	ej "vec-diputacion-granada/internal/modules/administracion/ports/ejecucioncopias"
)

type destinoNoUsado struct{}

func (destinoNoUsado) Publicar(context.Context, ej.Captura) (ej.Conjunto, error) {
	return ej.Conjunto{}, errRegistroConfiguracion
}
func (destinoNoUsado) CerrarVerificacion(context.Context, ej.Conjunto, copias.Verificacion) (ej.Conjunto, error) {
	return ej.Conjunto{}, errRegistroConfiguracion
}
func (destinoNoUsado) Recuperar(context.Context, string) (ej.Conjunto, error) {
	return ej.Conjunto{}, errRegistroConfiguracion
}

func TestRegistroReconciliaCS07TrasCaidaEntreDiarios(t *testing.T) {
	ctx := context.Background()
	base := t.TempDir()
	cs07Dir := filepath.Join(base, "cs07")
	exteriorDir := filepath.Join(base, "exterior")
	restaurada := filepath.Join(base, "restaurada")
	for _, dir := range []string{cs07Dir, exteriorDir, restaurada} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	journal, err := cs07.Abrir(cs07.Config{Directorio: cs07Dir, RaicesRestauradas: []string{restaurada}})
	if err != nil {
		t.Fatal(err)
	}
	config := ConfigRegistroCS07{Registro: journal, Destino: destinoNoUsado{}, DirectorioExterior: exteriorDir, RaicesRestauradas: []string{restaurada}}
	bridge, err := AbrirRegistroCS07(config)
	if err != nil {
		t.Fatal(err)
	}
	p := ej.Peticion{OperacionRef: "op-test", ActorRef: "actor-test", OrigenRef: "origen-test", DestinoRef: "destino-test", MotivoRef: "motivo-test", ConjuntoRef: "conjunto-test", PoliticaRef: "politica-test"}
	if _, err = bridge.Reservar(ctx, p); err != nil {
		t.Fatal(err)
	}
	evento := ej.EventoCopia{OperacionRef: p.OperacionRef, Transicion: "iniciar_captura", ConjuntoRef: p.ConjuntoRef}
	declarada := declaracion(p)
	actual, err := journal.Consultar(ctx, declarada, p.OperacionRef)
	if err != nil {
		t.Fatal(err)
	}
	comando, err := comandoCS07(evento, actual)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = journal.Aplicar(ctx, declarada, p.OperacionRef, comando); err != nil {
		t.Fatal(err)
	}
	// CS07 confirmó la transición; el proceso murió antes del append exterior.
	if err = bridge.AplicarCopia(ctx, evento); err != nil {
		t.Fatalf("reconciliación: %v", err)
	}
	if err = bridge.AplicarCopia(ctx, evento); err != nil {
		t.Fatalf("replay: %v", err)
	}
	bridge, err = AbrirRegistroCS07(config)
	if err != nil {
		t.Fatal(err)
	}
	op, err := bridge.Leer(ctx, p.OperacionRef)
	if err != nil || op.Estado != "capturando" || op.VersionRef != "1" {
		t.Fatalf("estado tras reabrir: %+v %v", op, err)
	}
	p.MotivoRef = "motivo-distinto"
	if _, err = bridge.Reservar(ctx, p); err == nil {
		t.Fatal("reserva semántica distinta aceptada")
	}
}

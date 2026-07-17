package domain

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

const referenciaCorrelacionNominalAutorizacionV2Prueba = "correlacion_0123456789abcdef0123456789abcdef"

type generadorCorrelacionAutorizacionV2Prueba struct {
	valor        string
	err          error
	invocaciones int
	despues      func()
}

func (g *generadorCorrelacionAutorizacionV2Prueba) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	g.invocaciones++
	if g.despues != nil {
		g.despues()
	}
	return g.valor, g.err
}

func referenciaCorrelacionAutorizacionV2ParaPrueba(
	t *testing.T,
	valor string,
) ReferenciaCorrelacionAutorizacionV2 {
	t.Helper()
	referencia, err := GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(),
		&generadorCorrelacionAutorizacionV2Prueba{valor: valor},
	)
	if err != nil {
		t.Fatalf("generar correlacion nominal de prueba: %v", err)
	}
	return referencia
}

func TestGenerarReferenciaCorrelacionAutorizacionV2AcunaUnaVezYReutiliza(t *testing.T) {
	generador := &generadorCorrelacionAutorizacionV2Prueba{valor: referenciaCorrelacionNominalAutorizacionV2Prueba}
	referencia, err := GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), generador)
	if err != nil || generador.invocaciones != 1 {
		t.Fatalf("generacion = %v, invocaciones=%d", err, generador.invocaciones)
	}
	primero, errPrimero := referencia.ValorCanonico()
	segundo, errSegundo := referencia.ValorCanonico()
	if referencia.Validar() != nil || errPrimero != nil || errSegundo != nil ||
		primero != referenciaCorrelacionNominalAutorizacionV2Prueba || segundo != primero ||
		generador.invocaciones != 1 {
		t.Fatalf("capacidad no reutilizable: %q/%q errores=%v/%v invocaciones=%d", primero, segundo, errPrimero, errSegundo, generador.invocaciones)
	}
}

func TestGenerarReferenciaCorrelacionAutorizacionV2FallaCerrado(t *testing.T) {
	errorGenerador := errors.New("entropia no disponible")
	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	ctxDurante, cancelarDurante := context.WithCancel(context.Background())
	var nuloTipado *generadorCorrelacionAutorizacionV2Prueba

	casos := []struct {
		nombre     string
		ctx        context.Context
		generador  generadorReferenciaCorrelacionAutorizacionV2
		esperar    error
		invocacion int
	}{
		{nombre: "contexto nulo", generador: &generadorCorrelacionAutorizacionV2Prueba{valor: referenciaCorrelacionNominalAutorizacionV2Prueba}, esperar: ErrReferenciaCorrelacionAutorizacionV2Invalida},
		{nombre: "generador nulo", ctx: context.Background(), esperar: ErrReferenciaCorrelacionAutorizacionV2Invalida},
		{nombre: "generador nulo tipado", ctx: context.Background(), generador: nuloTipado, esperar: ErrReferenciaCorrelacionAutorizacionV2Invalida},
		{nombre: "cancelado antes", ctx: ctxCancelado, generador: &generadorCorrelacionAutorizacionV2Prueba{valor: referenciaCorrelacionNominalAutorizacionV2Prueba}, esperar: context.Canceled},
		{nombre: "error generador", ctx: context.Background(), generador: &generadorCorrelacionAutorizacionV2Prueba{err: errorGenerador}, esperar: errorGenerador, invocacion: 1},
		{nombre: "valor vacio", ctx: context.Background(), generador: &generadorCorrelacionAutorizacionV2Prueba{}, esperar: ErrReferenciaCorrelacionAutorizacionV2Invalida, invocacion: 1},
		{nombre: "valor con datos", ctx: context.Background(), generador: &generadorCorrelacionAutorizacionV2Prueba{valor: "correlacion_dni_12345678z"}, esperar: ErrReferenciaCorrelacionAutorizacionV2Invalida, invocacion: 1},
		{nombre: "cancelado durante", ctx: ctxDurante, generador: &generadorCorrelacionAutorizacionV2Prueba{valor: referenciaCorrelacionNominalAutorizacionV2Prueba, despues: cancelarDurante}, esperar: context.Canceled, invocacion: 1},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			referencia, err := GenerarReferenciaCorrelacionAutorizacionV2(caso.ctx, caso.generador)
			if !errors.Is(err, ErrGeneracionReferenciaCorrelacionAutorizacionV2) || !errors.Is(err, caso.esperar) {
				t.Fatalf("error = %v", err)
			}
			if _, errValor := referencia.ValorCanonico(); !errors.Is(errValor, ErrReferenciaCorrelacionAutorizacionV2Invalida) {
				t.Fatalf("se devolvio capacidad valida: %v", errValor)
			}
			if !errors.Is(referencia.Validar(), ErrReferenciaCorrelacionAutorizacionV2Invalida) {
				t.Fatal("el valor cero no fallo cerrado")
			}
			if generador, ok := caso.generador.(*generadorCorrelacionAutorizacionV2Prueba); ok && generador != nil &&
				generador.invocaciones != caso.invocacion {
				t.Fatalf("invocaciones=%d; esperadas=%d", generador.invocaciones, caso.invocacion)
			}
		})
	}
}

func TestReferenciaCorrelacionAutorizacionV2BloqueaCodecsYRedacta(t *testing.T) {
	referencia, err := GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(),
		&generadorCorrelacionAutorizacionV2Prueba{valor: referenciaCorrelacionNominalAutorizacionV2Prueba},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(referencia); !errors.Is(err, ErrSerializacionReferenciaCorrelacionAutorizacionV2Prohibida) {
		t.Fatalf("JSON no bloqueado: %v", err)
	}
	if _, err := xml.Marshal(referencia); !errors.Is(err, ErrSerializacionReferenciaCorrelacionAutorizacionV2Prohibida) {
		t.Fatalf("XML no bloqueado: %v", err)
	}
	var destino bytes.Buffer
	if err := gob.NewEncoder(&destino).Encode(referencia); !errors.Is(err, ErrSerializacionReferenciaCorrelacionAutorizacionV2Prohibida) {
		t.Fatalf("Gob no bloqueado: %v", err)
	}
	for nombre, invocar := range map[string]func() error{
		"texto":   func() error { _, err := referencia.MarshalText(); return err },
		"binario": func() error { _, err := referencia.MarshalBinary(); return err },
		"CBOR":    func() error { _, err := referencia.MarshalCBOR(); return err },
		"YAML":    func() error { _, err := referencia.MarshalYAML(); return err },
	} {
		if err := invocar(); !errors.Is(err, ErrSerializacionReferenciaCorrelacionAutorizacionV2Prohibida) {
			t.Fatalf("%s no bloqueado: %v", nombre, err)
		}
	}
	for nombre, invocar := range map[string]func() error{
		"JSON":    func() error { return json.Unmarshal([]byte(`{}`), &ReferenciaCorrelacionAutorizacionV2{}) },
		"XML":     func() error { return xml.Unmarshal([]byte(`<referencia/>`), &ReferenciaCorrelacionAutorizacionV2{}) },
		"Gob":     func() error { return (&ReferenciaCorrelacionAutorizacionV2{}).GobDecode(nil) },
		"texto":   func() error { return (&ReferenciaCorrelacionAutorizacionV2{}).UnmarshalText(nil) },
		"binario": func() error { return (&ReferenciaCorrelacionAutorizacionV2{}).UnmarshalBinary(nil) },
		"CBOR":    func() error { return (&ReferenciaCorrelacionAutorizacionV2{}).UnmarshalCBOR(nil) },
		"YAML": func() error {
			return (&ReferenciaCorrelacionAutorizacionV2{}).UnmarshalYAML(func(any) error { return nil })
		},
	} {
		if err := invocar(); !errors.Is(err, ErrSerializacionReferenciaCorrelacionAutorizacionV2Prohibida) {
			t.Fatalf("decodificacion %s no bloqueada: %v", nombre, err)
		}
	}

	texto := fmt.Sprintf("%v %+v %#v %s %q", referencia, referencia, referencia, referencia, referencia)
	log := slog.AnyValue(referencia).Resolve().String()
	if strings.Contains(texto, referenciaCorrelacionNominalAutorizacionV2Prueba) ||
		strings.Contains(log, referenciaCorrelacionNominalAutorizacionV2Prueba) {
		t.Fatalf("valor filtrado por formato o log: %q / %q", texto, log)
	}
}

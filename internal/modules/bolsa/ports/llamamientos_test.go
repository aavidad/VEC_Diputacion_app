package ports

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestSolicitudProponerLlamamientoSoloExponeActorPerfilYReferencias(t *testing.T) {
	tipo := reflect.TypeOf(SolicitudProponerLlamamiento{})
	esperados := map[string]reflect.Type{
		"Actor": reflect.TypeOf(dominiovec.ContextoActor{}), "PerfilActivoRef": reflect.TypeOf(""),
		"AutenticacionRef": reflect.TypeOf(""), "SesionRef": reflect.TypeOf(""),
		"NecesidadRef": reflect.TypeOf(""), "CorrelacionRef": reflect.TypeOf(""),
	}
	if tipo.NumField() != len(esperados) {
		t.Fatalf("la solicitud incorporo datos declarables no autorizados: %d campos", tipo.NumField())
	}
	for indice := 0; indice < tipo.NumField(); indice++ {
		campo := tipo.Field(indice)
		esperado, existe := esperados[campo.Name]
		if !existe || campo.Type != esperado {
			t.Fatalf("campo externo no permitido: %s %v", campo.Name, campo.Type)
		}
	}
	for _, prohibido := range []string{"Entrada", "Evaluacion", "Listado", "Estado", "Criterio", "Rol", "Permiso"} {
		for indice := 0; indice < tipo.NumField(); indice++ {
			if strings.Contains(strings.ToLower(tipo.Field(indice).Name), strings.ToLower(prohibido)) {
				t.Fatalf("la solicitud permite declarar %q", tipo.Field(indice).Name)
			}
		}
	}
}

func TestSolicitudProponerLlamamientoNoPuedeUsarseComoDTODeCliente(t *testing.T) {
	var destino SolicitudProponerLlamamiento
	if err := json.Unmarshal([]byte(`{"perfil_activo_ref":"fabricado"}`), &destino); !errors.Is(err, ErrSerializacionSolicitudLlamamientoProhibida) {
		t.Fatalf("el cliente pudo reconstruir el comando interno: %v", err)
	}
	if _, err := json.Marshal(SolicitudProponerLlamamiento{}); !errors.Is(err, ErrSerializacionSolicitudLlamamientoProhibida) {
		t.Fatalf("el comando interno pudo filtrarse como JSON: %v", err)
	}
}

func TestReferenciaOpacaLlamamientoDeniegaPIIComodinesYTextoNoCanonico(t *testing.T) {
	validas := []string{"necesidad:01K0VS7P", "corr:01K0VS7Q", "prf_abcdefghijklmnopqrstuv"}
	for _, referencia := range validas {
		if !ReferenciaOpacaLlamamientoValida(referencia) {
			t.Fatalf("referencia opaca valida rechazada: %q", referencia)
		}
	}
	invalidas := []string{
		"", " necesidad:01 ", "necesidad:*", "sujeto:12345678Z", "sujeto:X1234567L",
		"sujeto:dni:opaco", "sujeto:pasaporte:opaco", "corr:\u202Eoculto", "corr:\nlinea",
		strings.Repeat("a", 513),
	}
	for _, referencia := range invalidas {
		if ReferenciaOpacaLlamamientoValida(referencia) {
			t.Fatalf("referencia peligrosa admitida: %q", referencia)
		}
	}
}

func TestSolicitudEvaluacionDeniegaValorCeroYTiempoNoCanonico(t *testing.T) {
	var cero SolicitudEvaluarParticipacionLlamamiento
	if err := cero.Validar(); err != ErrEvaluacionMotorNoConfiable {
		t.Fatalf("valor cero no denegado: %v", err)
	}
	cero.EvaluadaEn = time.Now()
	if err := cero.Validar(); err != ErrEvaluacionMotorNoConfiable {
		t.Fatalf("entrada parcial no denegada: %v", err)
	}
}

package ports_test

import (
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

const (
	valorCodigoCotejoSeparado = "2345-6789 abcdefgh jklmnpqrstuvwxyz"
	valorCodigoCotejoCanonico = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
)

func TestSecretoCodigoCotejoNormalizaSeparadoresYSoloSeRevelaExplicitamente(t *testing.T) {
	secreto, err := ports.NuevoSecretoCodigoCotejo(valorCodigoCotejoSeparado)
	if err != nil {
		t.Fatalf("NuevoSecretoCodigoCotejo() error = %v", err)
	}
	if err := secreto.Validar(); err != nil {
		t.Fatalf("SecretoCodigoCotejo.Validar() error = %v", err)
	}
	if obtenido := secreto.Revelar(); obtenido != valorCodigoCotejoCanonico {
		t.Fatalf("SecretoCodigoCotejo.Revelar() = %q, want %q", obtenido, valorCodigoCotejoCanonico)
	}
}

func TestSecretoCodigoCotejoRechazaValoresInvalidosYValorCero(t *testing.T) {
	for _, prueba := range []struct {
		nombre string
		valor  string
	}{
		{nombre: "vacio", valor: ""},
		{nombre: "demasiado corto", valor: "23456789ABCDEFGH"},
		{nombre: "cero ambiguo", valor: "23456789ABCDEFGHJKLMNPQRSTUV0"},
		{nombre: "uno ambiguo", valor: "23456789ABCDEFGHJKLMNPQRSTUV1"},
		{nombre: "i ambigua", valor: "23456789ABCDEFGHJKLMNPQRSTUVI"},
		{nombre: "o ambigua", valor: "23456789ABCDEFGHJKLMNPQRSTUVO"},
		{nombre: "separador no admitido", valor: "23456789ABCDEFGHJKLMNPQRSTUV/"},
		{nombre: "control", valor: "23456789ABCDEFGH\nJKLMNPQRSTUV"},
	} {
		t.Run(prueba.nombre, func(t *testing.T) {
			_, err := ports.NuevoSecretoCodigoCotejo(prueba.valor)
			if !errors.Is(err, ports.ErrMaterialCodigoCotejoInvalido) {
				t.Fatalf("NuevoSecretoCodigoCotejo(%q) error = %v, want %v", prueba.valor, err, ports.ErrMaterialCodigoCotejoInvalido)
			}
		})
	}

	var cero ports.SecretoCodigoCotejo
	if err := cero.Validar(); !errors.Is(err, ports.ErrMaterialCodigoCotejoInvalido) {
		t.Fatalf("valor cero Validar() error = %v, want %v", err, ports.ErrMaterialCodigoCotejoInvalido)
	}
}

func TestSecretoCodigoCotejoNoFiltraValorMedianteFormato(t *testing.T) {
	secreto, err := ports.NuevoSecretoCodigoCotejo(valorCodigoCotejoSeparado)
	if err != nil {
		t.Fatalf("NuevoSecretoCodigoCotejo() error = %v", err)
	}

	for _, formato := range []string{"%s", "%v", "%+v", "%#v"} {
		t.Run(formato, func(t *testing.T) {
			salida := fmt.Sprintf(formato, secreto)
			if strings.Contains(salida, valorCodigoCotejoCanonico) || strings.Contains(salida, valorCodigoCotejoSeparado) {
				t.Fatalf("fmt.Sprintf(%q) filtro el secreto: %q", formato, salida)
			}
			if strings.TrimSpace(salida) == "" {
				t.Fatalf("fmt.Sprintf(%q) no produjo marcador redactado", formato)
			}
		})
	}
}

func TestSecretoCodigoCotejoProhibeSerializacionJSONYTexto(t *testing.T) {
	secreto, err := ports.NuevoSecretoCodigoCotejo(valorCodigoCotejoSeparado)
	if err != nil {
		t.Fatalf("NuevoSecretoCodigoCotejo() error = %v", err)
	}

	if _, err := json.Marshal(secreto); !errors.Is(err, ports.ErrSerializacionCodigoCotejoProhibida) {
		t.Fatalf("json.Marshal() error = %v, want %v", err, ports.ErrSerializacionCodigoCotejoProhibida)
	}

	var serializadorTexto encoding.TextMarshaler = secreto
	if _, err := serializadorTexto.MarshalText(); !errors.Is(err, ports.ErrSerializacionCodigoCotejoProhibida) {
		t.Fatalf("encoding.TextMarshaler.MarshalText() error = %v, want %v", err, ports.ErrSerializacionCodigoCotejoProhibida)
	}
}

func TestContextosCustodiaCotejoSonOpacosIncompatiblesYFallaCerradoElValorCero(t *testing.T) {
	contextos := []any{
		ports.ContextoProtegerCodigoCotejo{},
		ports.ContextoRecuperarCodigoCotejo{},
		ports.ContextoEliminarCodigoCotejoHuerfano{},
	}
	for _, contexto := range contextos {
		tipo := reflect.TypeOf(contexto)
		for indice := 0; indice < tipo.NumField(); indice++ {
			if tipo.Field(indice).IsExported() {
				t.Fatalf("%s expone el campo forjable %s", tipo, tipo.Field(indice).Name)
			}
		}
		if _, err := json.Marshal(contexto); !errors.Is(err, ports.ErrSerializacionContextoCotejoProhibida) {
			t.Fatalf("json.Marshal(%s) error = %v", tipo, err)
		}
		serializador, ok := contexto.(encoding.TextMarshaler)
		if !ok {
			t.Fatalf("%s no implementa encoding.TextMarshaler", tipo)
		}
		if _, err := serializador.MarshalText(); !errors.Is(err, ports.ErrSerializacionContextoCotejoProhibida) {
			t.Fatalf("MarshalText(%s) error = %v", tipo, err)
		}
	}
	if err := (ports.SolicitudProtegerCodigoCotejo{}).ValidarEn(time.Now()); !errors.Is(err, ports.ErrContextoCustodiaCotejoInvalido) {
		t.Fatalf("solicitud cero proteger error = %v", err)
	}
	if err := (ports.SolicitudRecuperarCodigoCotejo{}).ValidarEn(time.Now()); !errors.Is(err, ports.ErrContextoCustodiaCotejoInvalido) {
		t.Fatalf("solicitud cero recuperar error = %v", err)
	}
	if err := (ports.SolicitudEliminarCodigoCotejoHuerfano{}).ValidarEn(time.Now()); !errors.Is(err, ports.ErrContextoCustodiaCotejoInvalido) {
		t.Fatalf("solicitud cero eliminar error = %v", err)
	}
}

func TestContextoProtegerCotejoExigeDecisionExactaYFijaLaSolicitud(t *testing.T) {
	ahora := time.Date(2026, time.July, 15, 9, 0, 0, 0, time.UTC)
	indice := "hmac-sha256:indice-cotejo-puerto:" + strings.Repeat("a", 64)
	recurso := recursoCustodiaCotejoPrueba(ports.AccionProtegerCodigoCotejo, map[string]string{
		"clave_idempotencia": "custodia-cotejo:codigo-prueba-001",
		"indice_codigo_hmac": indice,
	})
	decision := decisionCustodiaCotejoPrueba(t, ahora, ports.AccionProtegerCodigoCotejo, recurso)
	contexto, err := ports.NuevoContextoProtegerCodigoCotejo(
		decision, recurso, recurso.Referencia, "restringido",
		"custodia-cotejo:codigo-prueba-001", indice, ahora,
	)
	if err != nil {
		t.Fatalf("NuevoContextoProtegerCodigoCotejo() error = %v", err)
	}
	secreto, err := ports.NuevoSecretoCodigoCotejo(valorCodigoCotejoCanonico)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := ports.SolicitudProtegerCodigoCotejo{
		Contexto: contexto, ClaveIdempotencia: "custodia-cotejo:codigo-prueba-001",
		Secreto: secreto, IndiceCodigoHMAC: indice,
	}
	if err := solicitud.ValidarEn(ahora); err != nil {
		t.Fatalf("SolicitudProtegerCodigoCotejo.ValidarEn() error = %v", err)
	}
	mutada := solicitud
	mutada.ClaveIdempotencia = "custodia-cotejo:otra"
	if err := mutada.ValidarEn(ahora); !errors.Is(err, ports.ErrContextoCustodiaCotejoInvalido) {
		t.Fatalf("solicitud mutada error = %v", err)
	}

	decisionReserva := decisionCustodiaCotejoPrueba(t, ahora, "vec.documentos.cotejo.codigos.reservar", recurso)
	if _, err := ports.NuevoContextoProtegerCodigoCotejo(
		decisionReserva, recurso, recurso.Referencia, "restringido",
		"custodia-cotejo:codigo-prueba-001", indice, ahora,
	); !errors.Is(err, ports.ErrContextoCustodiaCotejoInvalido) {
		t.Fatalf("decision de reservar reinterpretada: error = %v", err)
	}
}

func TestContextoEliminarHuerfanoRechazaCuentaNoPrivilegiadaAunquePDPConceda(t *testing.T) {
	ahora := time.Date(2026, time.July, 15, 9, 0, 0, 0, time.UTC)
	recurso := recursoCustodiaCotejoPrueba(ports.AccionEliminarCodigoCotejoHuerfano, map[string]string{
		"proteccion_ref": "vault:cotejo:001", "evidencia_ref": "evidencia:cotejo:001",
		"motivo": "reserva no confirmada",
	})
	decision := decisionCustodiaCotejoPrueba(t, ahora, ports.AccionEliminarCodigoCotejoHuerfano, recurso)
	if _, err := ports.NuevoContextoEliminarCodigoCotejoHuerfano(
		decision, recurso, recurso.Referencia, "restringido", "vault:cotejo:001",
		"evidencia:cotejo:001", "reserva no confirmada", ahora,
	); !errors.Is(err, ports.ErrContextoCustodiaCotejoInvalido) {
		t.Fatalf("limpieza con cuenta ordinaria error = %v", err)
	}
}

func recursoCustodiaCotejoPrueba(accion string, especificos map[string]string) domain.RecursoAutorizable {
	atributos := map[string]string{
		"operacion_custodia": accion,
		"documento_ref":      "documento-cotejo-prueba:1",
		"politica_ref":       "politica_cotejo_prueba:1",
	}
	for clave, valor := range especificos {
		atributos[clave] = valor
	}
	return domain.RecursoAutorizable{
		Referencia: "cotejo:codigo-prueba-001", ModuloID: "bolsa", Tipo: "codigo_cotejo",
		Ambitos:   map[string]string{"clasificacion": "restringido", "expediente": "expediente-cotejo-prueba-001"},
		Atributos: atributos,
	}
}

func decisionCustodiaCotejoPrueba(
	t *testing.T,
	ahora time.Time,
	accion string,
	recurso domain.RecursoAutorizable,
) domain.DecisionAutorizacion {
	t.Helper()
	vinculo, err := pruebasvec.NuevoVinculoGenerico(ahora)
	if err != nil {
		t.Fatal(err)
	}
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	politicas := []string{"politica:acceso-cotejo:v1"}
	huellas := map[string]string{politicas[0]: strings.Repeat("d", 64)}
	huellaCatalogo, err := domain.HuellaEvidenciasCatalogoPoliticasAutorizacion(politicas, huellas)
	if err != nil {
		t.Fatal(err)
	}
	decision := domain.DecisionAutorizacion{
		DecisionRef: "decision:custodia-cotejo:" + accion, Concedida: true, Codigo: "concedida",
		PrincipalID: "per_0123456789abcdefghijkl", PerfilActivoRef: "prf_0123456789abcdefghijkl",
		Accion: accion, RecursoRef: recurso.Referencia, ModuloID: recurso.ModuloID,
		TipoRecurso: recurso.Tipo, ContextoRecursoHuellaSHA256: huellaRecurso,
		Finalidad: "custodiar_codigo_cotejo", CorrelacionRef: "correlacion-cotejo-prueba-001",
		VinculoAutenticacionActor: vinculo,
		AsignacionRef:             "asignacion:cotejo:v1", AsignacionHuellaSHA256: strings.Repeat("a", 64),
		VersionRolRef: "rol:cotejo:v1", VersionRolHuellaSHA256: strings.Repeat("b", 64),
		ControlVigenciaVersionRolRef: "rol:cotejo:v1", ControlVigenciaVersionRolRevision: 1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("c", 64),
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasRefs: politicas, PoliticasEvaluadasHuellasSHA256: huellas,
		PoliticasRefs: politicas, PoliticasHuellasSHA256: huellas,
		GarantiaMinima: domain.AuthAssuranceHigh, EmitidaEn: ahora, ValidaHasta: ahora.Add(5 * time.Minute),
	}
	if err := decision.ValidarEvidenciaInstantanea(); err != nil {
		t.Fatalf("decision de prueba invalida: %v", err)
	}
	return decision
}

package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/dietas/application"
	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

func TestCircuitoDeniegaIdentidadNoAcreditadaAntesDeSQL(t *testing.T) {
	tx := &txBorradorPrueba{}
	pool := &poolBorradorPrueba{tx: tx}
	repo, _ := nuevoRepositorioBorradorComisionPostgreSQL(pool)
	ref := "dco_" + strings.Repeat("b", 22)
	orden := dietasports.SolicitudDecisionCircuito{Referencia: ref, UnidadRef: "unidad:prueba", Etapa: domain.EtapaRevision, Decision: domain.DecisionAprobar, ClaveIdempotencia: "clave_revision_0001", VersionEsperada: 2}
	if _, err := repo.Decidir(context.Background(), dietasports.IdentidadEfectivaCircuito{}, orden); !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) {
		t.Fatalf("decisión sin identidad: %v", err)
	}
	lista := dietasports.ConsultaBandejaCircuito{Etapa: domain.EtapaRevision, UnidadRef: "unidad:prueba", Limite: 20}
	if _, err := repo.ListarPendientes(context.Background(), dietasports.IdentidadEfectivaCircuito{}, lista); !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) {
		t.Fatalf("bandeja sin identidad: %v", err)
	}
	if pool.inicios != 0 || len(tx.consultas) != 0 {
		t.Fatal("se abrió transacción sin identidad")
	}
}

func TestCircuitoTraduceConflictoYNoFiltraMensajeSQL(t *testing.T) {
	err := normalizarErrorCircuito(context.Background(), &pgconn.PgError{Code: "PD005", Message: "detalle privado"}, true)
	if !errors.Is(err, dietasports.ErrEstadoCircuitoConflicto) || strings.Contains(err.Error(), "privado") {
		t.Fatalf("error SQL expuesto: %v", err)
	}
}

// Reproduce la huella que calcula vec_dietas.cotejar_efecto_circuito_v2 (y la
// v1 para la bandeja) a partir de los bytes exactos del material, para que
// Go y SQL no diverjan: claves ordenadas y "etapa" solo en la lectura.
func TestEfectoCircuitoCoincideConElCotejoSQL(t *testing.T) {
	contexto := contextoRegistradoBorradorPrueba(t)
	ref := "dco_" + strings.Repeat("f", 22)
	casos := []dietasports.SolicitudOperacionCircuito{
		{Operacion: dietasports.OperacionDecidirCircuito, Decision: dietasports.SolicitudDecisionCircuito{Referencia: ref, UnidadRef: "unidad:prueba", Etapa: domain.EtapaAutorizacion, Decision: domain.DecisionDevolver, Motivo: "Falta justificante <&>", ClaveIdempotencia: "clave_autoriza_0001", VersionEsperada: 3}},
		{Operacion: dietasports.OperacionConsultarDocumentoCircuito, Documento: dietasports.SolicitudDocumentoCircuito{Referencia: ref, Etapa: domain.EtapaRevision, UnidadRef: "unidad:prueba"}},
		{Operacion: dietasports.OperacionListarBandeja, Consulta: dietasports.ConsultaBandejaCircuito{Etapa: domain.EtapaFiscalizacion, UnidadRef: "unidad:prueba", Limite: 20, FechaDesde: "2026-09-01"}},
	}
	for _, op := range casos {
		efecto, err := application.ConstruirEfectoAutorizacionCircuito(contexto, "unidad:prueba", op)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := json.Unmarshal(efecto.Material, &m); err != nil {
			t.Fatal(err)
		}
		if _, ok := m["asignacion"]; ok {
			t.Fatalf("%s: material con asignación del cliente", op.Operacion)
		}
		esquema := "vec.dietas.circuito-operacion.v2"
		if op.Operacion == dietasports.OperacionListarBandeja {
			esquema = "vec.dietas.circuito-operacion.v1"
		}
		if m["esquema"] != esquema {
			t.Fatalf("%s: esquema %v", op.Operacion, m["esquema"])
		}
		i := m["identidad"].(map[string]any)
		cadena := func(v any) string {
			b, _ := json.Marshal(fmt.Sprint(v))
			return string(b)
		}
		suma := sha256.Sum256(efecto.Material)
		atr := `"contexto_actor_ref":` + cadena(i["contexto_actor_ref"]) + `,"contexto_version":` + cadena(i["contexto_version"]) +
			`,"cuenta_ref":` + cadena(i["cuenta_ref"]) + `,"cuenta_version":` + cadena(i["cuenta_version"])
		if op.Operacion == dietasports.OperacionConsultarDocumentoCircuito {
			atr += `,"etapa":` + cadena(m["etapa"])
		}
		atr += `,"material_sha256":"` + hex.EncodeToString(suma[:]) + `","operacion":` + cadena(m["operacion"]) +
			`,"perfil_version":` + cadena(i["perfil_version"]) + `,"persona_version":` + cadena(i["persona_version"]) + `,"recurso_ref":` + cadena(m["recurso_ref"])
		sql := sha256.Sum256([]byte(`{"ambitos":{"persona_ref":` + cadena(i["persona_ref"]) + `,"unidad_ref":` + cadena(m["unidad_ref"]) + `},"atributos":{` + atr + `}}`))
		goHuella, err := efecto.Recurso.HuellaContextoAutorizacionSHA256()
		if err != nil || goHuella != hex.EncodeToString(sql[:]) {
			t.Fatalf("%s: huella Go %s distinta de la de SQL %x (%v)", op.Operacion, goHuella, sql, err)
		}
	}
}

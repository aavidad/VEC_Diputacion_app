package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"strings"
	"testing"
	"time"

	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func fixturePostimagenAnotacion(t *testing.T) (ct.OrdenConfirmarAnotacionAdministrativa, []byte) {
	t.Helper()
	e := expedienteAsignacionPostgreSQLPrueba(t)
	vinculo := dom.VinculoSeguimientoOriginal{SeguimientoRef: "ref:" + strings.Repeat("a", 64), VersionSeguimiento: 1, HuellaRaizSeguimientoSHA256: strings.Repeat("b", 64)}
	instante := e.ActualizadoEn.Add(time.Minute)
	n, err := e.RegistrarAnotacionAdministrativa(e.Version, vinculo, dom.DatosActuacion{AccionClave: dom.AccionRegistrarAnotacionAdministrativa, ActorRef: "persona:rrhh-sintetica", UnidadRef: "unidad:rrhh-sintetica", ReciboRef: "recibo:anotacion-sintetica", RealizadaEn: instante, FaseDestino: e.FaseActual, EstadoDestino: e.EstadoActual, Observaciones: "Anotación sintética de prueba."})
	if err != nil {
		t.Fatal(err)
	}
	original, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	// Reproduce los timestamps históricos escritos por PostgreSQL con seis
	// decimales; un marshal completo de Go los cambiaría aunque sean el mismo instante.
	original = bytes.ReplaceAll(original, []byte("00Z\""), []byte("00.000000Z\""))
	return ct.OrdenConfirmarAnotacionAdministrativa{Preparacion: ct.PreparacionAnotacionAdministrativa{Expediente: e, Estado: ct.PreparacionAnotacionAdministrativaPreparada}, ExpedienteSiguiente: n, InstanteEfecto: instante}, original
}
func TestAnotacionPostimagenPreservaJSONHistoricoYAnadeVinculo(t *testing.T) {
	o, original := fixturePostimagenAnotacion(t)
	b, err := postimagenAnotacion(original, o)
	if err != nil {
		t.Fatal(err)
	}
	var antes, despues map[string]json.RawMessage
	if json.Unmarshal(original, &antes) != nil || json.Unmarshal(b, &despues) != nil {
		t.Fatal("json inválido")
	}
	for k, v := range antes {
		if k == "version" || k == "actualizado_en" || k == "actuaciones" {
			continue
		}
		if !bytes.Equal(v, despues[k]) {
			t.Fatalf("se reescribe campo histórico %s", k)
		}
	}
	var aa, ad []json.RawMessage
	_ = json.Unmarshal(antes["actuaciones"], &aa)
	_ = json.Unmarshal(despues["actuaciones"], &ad)
	if len(ad) != len(aa)+1 {
		t.Fatal("no se añade exactamente una actuación")
	}
	for i := range aa {
		if !bytes.Equal(aa[i], ad[i]) {
			t.Fatalf("se reescribe actuación %d", i)
		}
	}
	var restaurado dom.Expediente
	if json.Unmarshal(b, &restaurado) != nil || restaurado.Validar() != nil {
		t.Fatal("postimagen no restaurable por dominio")
	}
	if restaurado.FaseActual != o.Preparacion.Expediente.FaseActual || restaurado.EstadoActual != o.Preparacion.Expediente.EstadoActual || restaurado.Actuaciones[len(aa)].SeguimientoOriginal == nil {
		t.Fatal("anotación cambia fase/estado o pierde vínculo")
	}
}
func TestAnotacionPostimagenRechazaPreimagenAjena(t *testing.T) {
	o, b := fixturePostimagenAnotacion(t)
	o.Preparacion.Expediente.Solicitud.Detalle = "Otro detalle"
	if _, err := postimagenAnotacion(b, o); err == nil {
		t.Fatal("acepta preimagen distinta")
	}
}
func TestAnotacionReplayNoReconstruyePostimagen(t *testing.T) {
	o, b := fixturePostimagenAnotacion(t)
	o.Preparacion.Estado = ct.PreparacionAnotacionAdministrativaConfirmada
	o.ExpedienteSiguiente = dom.Expediente{}
	got, e := postimagenAnotacion(b, o)
	if e != nil || !bytes.Equal(got, b) {
		t.Fatal("replay reconstruye historia")
	}
}
func TestAnotacionMaterialSQLNoTransportaClaveCruda(t *testing.T) {
	m := ct.MaterialAnotacionAdministrativa{ClaveIdempotencia: "b6968351-d744-42dd-9ff9-a909dfc0c116"}
	b, e := entradaPrepararAnotacion(m, sellosPrepararAltaV2{})
	if e != nil {
		t.Fatal(e)
	}
	if bytes.Contains(b, []byte(m.ClaveIdempotencia)) || bytes.Contains(b, []byte("clave_idempotencia")) {
		t.Fatal("clave cruda enviada a SQL")
	}
}
func TestAnotacionAdaptadoresRechazanDependenciasNulas(t *testing.T) {
	if _, e := NuevoPreparadorAnotacionAdministrativaPostgreSQL(nil); e == nil {
		t.Fatal("pool nil aceptado")
	}
	if _, e := NuevaTransaccionAnotacionesAdministrativasPostgreSQL(nil, nil); e == nil {
		t.Fatal("dependencias nil aceptadas")
	}
	var a *PreparadorAnotacionAdministrativaPostgreSQL
	if _, e := a.PrepararAnotacionAdministrativa(context.Background(), ct.SolicitudPrepararAnotacionAdministrativa{}); e == nil {
		t.Fatal("receptor nil aceptado")
	}
	var tx *TransaccionAnotacionesAdministrativasPostgreSQL
	if _, e := tx.ConfirmarAnotacionAdministrativa(context.Background(), ct.OrdenConfirmarAnotacionAdministrativa{}); e == nil {
		t.Fatal("TX nil aceptada")
	}
}

func TestAnotacionUnicodeMaximoConservaTexto(t *testing.T) {
	o, original := fixturePostimagenAnotacion(t)
	texto := strings.Repeat("界", 2000)
	o.ExpedienteSiguiente.Actuaciones[len(o.ExpedienteSiguiente.Actuaciones)-1].Observaciones = texto
	if o.ExpedienteSiguiente.Validar() != nil {
		t.Fatal("dominio rechaza máximo UTF8 válido")
	}
	b, e := postimagenAnotacion(original, o)
	if e != nil {
		t.Fatal(e)
	}
	var restaurado dom.Expediente
	if json.Unmarshal(b, &restaurado) != nil || restaurado.Actuaciones[len(restaurado.Actuaciones)-1].Observaciones != texto {
		t.Fatal("trunca texto Unicode")
	}
	a, e := json.Marshal(restaurado.Actuaciones[len(restaurado.Actuaciones)-1])
	if e != nil {
		t.Fatal(e)
	}
	if len(a) <= 4096 {
		t.Fatal("fixture no reproduce el límite de prueba SQL histórica")
	}
}
func TestAnotacionClasificaConflictoSinExponerFallosInternos(t *testing.T) {
	if !errors.Is(errorAnotacion(context.Background(), &pgconn.PgError{Code: "P0865"}), dom.ErrVersionEnConflicto) {
		t.Fatal("conflicto CAS no conserva error nominal")
	}
	for _, code := range []string{"P0862", "XX000", "42501"} {
		if errors.Is(errorAnotacion(context.Background(), &pgconn.PgError{Code: code}), dom.ErrVersionEnConflicto) {
			t.Fatalf("clasifica fallo interno %s como conflicto de versión", code)
		}
	}
}

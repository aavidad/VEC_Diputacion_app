package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func materialCentroPrueba() MaterialIncorporacionCentro {
	return MaterialIncorporacionCentro{Operacion: OperacionConfirmarIncorporacionCentro, ClaveIdempotencia: "77777777-7777-4777-8777-777777777777",
		OrganizacionRef: "organizacion:desarrollo:dipgra", Actor: domain.ActorPeticionCentro{ActorRef: "per_c1", PerfilRef: "prf_c1", CentroRef: "centro-520", PuestoRef: "puesto-1"},
		PeticionRef: "peticion:centro:1", ExpedienteRef: "expediente:ct:1", FechaIncorporacion: "2026-09-01",
		Documento: DocumentoIncorporacionCentro{Tipo: "toma_posesion", Referencia: "registro:centro-520:2026/1", SHA256: strings.Repeat("b", 64)},
		Regla:     ReglaDocumentoIncorporacion{Referencia: "vec.contratacion_temporal.reglas:1:c21.acreditacion_incorporacion", HuellaSHA256: strings.Repeat("a", 64), ModalidadClave: "sustitucion"}}
}

func TestMaterialIncorporacionCentroSeValidaYSeSerializaConLasClavesDeCT124(t *testing.T) {
	m := materialCentroPrueba()
	b, err := SerializarMaterialIncorporacionCentro(m)
	if err != nil {
		t.Fatal(err)
	}
	var claves map[string]json.RawMessage
	if json.Unmarshal(b, &claves) != nil || len(claves) != 9 {
		t.Fatalf("claves del material: %s", b)
	}
	for _, c := range []string{"operacion", "clave_idempotencia", "organizacion_ref", "actor", "peticion_ref", "expediente_ref", "fecha_incorporacion", "documento", "regla"} {
		if _, ok := claves[c]; !ok {
			t.Fatalf("falta %s", c)
		}
	}
	for nombre, mutar := range map[string]func(*MaterialIncorporacionCentro){
		"fecha imposible":    func(m *MaterialIncorporacionCentro) { m.FechaIncorporacion = "2026-02-30" },
		"huella nula":        func(m *MaterialIncorporacionCentro) { m.Documento.SHA256 = strings.Repeat("0", 64) },
		"tipo vacío":         func(m *MaterialIncorporacionCentro) { m.Documento.Tipo = "" },
		"clave no v4":        func(m *MaterialIncorporacionCentro) { m.ClaveIdempotencia = "77777777-7777-1777-8777-777777777777" },
		"regla larga":        func(m *MaterialIncorporacionCentro) { m.Regla.Referencia = strings.Repeat("r", 256) },
		"actor sin centro":   func(m *MaterialIncorporacionCentro) { m.Actor.CentroRef = "" },
		"operación ajena":    func(m *MaterialIncorporacionCentro) { m.Operacion = "cancelar" },
		"modalidad inválida": func(m *MaterialIncorporacionCentro) { m.Regla.ModalidadClave = "Vacante" },
	} {
		copia := materialCentroPrueba()
		mutar(&copia)
		if _, err := SerializarMaterialIncorporacionCentro(copia); err == nil {
			t.Fatalf("%s: se esperaba rechazo", nombre)
		}
	}
	// El recurso liga la huella exacta del material y los ámbitos que SQL recompone.
	r := RecursoIncorporacionCentro(m.ExpedienteRef, m.OrganizacionRef, m.Actor.CentroRef, b)
	h := sha256.Sum256(b)
	if r.Tipo != TipoRecursoIncorporacionCentro || r.Atributos["material_sha256"] != hex.EncodeToString(h[:]) ||
		r.Ambitos["centro_ref"] != "centro-520" || r.Ambitos["organizacion_ref"] != m.OrganizacionRef || len(r.Ambitos) != 2 || len(r.Atributos) != 1 {
		t.Fatalf("recurso: %+v", r)
	}
	huella, err := r.HuellaContextoAutorizacionSHA256()
	esperada := sha256.Sum256([]byte(`{"ambitos":{"centro_ref":"centro-520","organizacion_ref":"organizacion:desarrollo:dipgra"},"atributos":{"material_sha256":"` + hex.EncodeToString(h[:]) + `"}}`))
	if err != nil || huella != hex.EncodeToString(esperada[:]) {
		t.Fatalf("la huella de contexto no es la canónica que recompone CT124: %s %v", huella, err)
	}
}

func TestReciboYEstadoDeLaIncorporacionAcreditada(t *testing.T) {
	m := materialCentroPrueba()
	r := ReciboIncorporacionCentro{ReciboRef: "recibo:c:1", ConfirmacionRef: "confirmacion:c:1", PeticionRef: m.PeticionRef, ExpedienteRef: m.ExpedienteRef,
		FechaIncorporacion: m.FechaIncorporacion, DocumentoTipo: "toma_posesion", ActorRef: "per_c1", RegistradoEn: time.Now(), EstadoLocal: "replay_confirmado"}
	if r.ValidarPara(m) != nil {
		t.Fatal("recibo válido rechazado")
	}
	r.DocumentoTipo = "contrato_firmado"
	if r.ValidarPara(m) == nil {
		t.Fatal("recibo de otro documento aceptado")
	}
	e := EstadoIncorporacionAcreditada{GINPIX: &EstadoGINPIXConfirmado{Numero: "GX-1", ConfirmadaEn: "2027-02-30", ReciboRef: "recibo:g:1",
		RegistradaEn: time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC)}}
	if e.Valido() {
		t.Fatal("fecha de GINPIX imposible aceptada")
	}
	e.GINPIX.ConfirmadaEn = "2027-02-28"
	if !e.Valido() {
		t.Fatal("estado válido rechazado")
	}
}

// Una sustitución sin fecha de fin guarda en la petición causa_fin y
// politica_fin. La bandeja del centro las recibía como claves desconocidas,
// el adaptador estricto rechazaba la fila y la consulta respondía 503.
func TestBandejaCentroAdmitePeriodoSinFechaDeFin(t *testing.T) {
	fila := func(periodo string) []byte {
		return []byte(`{"peticion_ref":"peticion:centro:1","expediente_ref":"expediente:ct:1","numero_visible":"2026/91101",` +
			`"version":1,"fase":"solicitud","estado":"en_curso","modalidad_clave":null,"categoria_ref":"categoria:desarrollo:c2",` +
			`"periodo":` + periodo + `,"confirmacion":null}`)
	}
	decodificar := func(b []byte) (ExpedienteIncorporacionCentro, error) {
		var e ExpedienteIncorporacionCentro
		d := json.NewDecoder(bytes.NewReader(b))
		d.DisallowUnknownFields()
		return e, d.Decode(&e)
	}
	sinFin := `{"inicio":"2026-11-02T00:00:00Z","causa_fin":"reincorporacion_titular","politica_fin":{"causa_fin":"reincorporacion_titular",` +
		`"fecha_fin":"no_aplica","regla_ref":"vec.contratacion_temporal.reglas:3:c12.modalidad.sustitucion","catalogo_version":3,` +
		`"catalogo_huella_sha256":"99fe337dbb923df313480a3f6712a134eb3cadffc9bcdd0a673129ca748f2d39"}}`
	e, err := decodificar(fila(sinFin))
	if err != nil || !e.Valido() || e.Periodo == nil || e.Periodo.CausaFin != "reincorporacion_titular" || e.Periodo.Fin != "" {
		t.Fatalf("periodo sin fecha de fin rechazado: %+v %v", e.Periodo, err)
	}
	salida, err := json.Marshal(e.Periodo)
	if err != nil || string(salida) != `{"inicio":"2026-11-02T00:00:00Z","causa_fin":"reincorporacion_titular"}` {
		t.Fatalf("el centro no debe recibir la política del periodo: %s %v", salida, err)
	}
	e, err = decodificar(fila(`{"inicio":"2026-11-01T00:00:00Z","fin":"2026-12-31T00:00:00Z"}`))
	if err != nil || !e.Valido() || e.Periodo.Fin != "2026-12-31T00:00:00Z" || e.Periodo.CausaFin != "" {
		t.Fatalf("periodo con fecha de fin rechazado: %+v %v", e.Periodo, err)
	}
	for nombre, periodo := range map[string]string{
		"clave desconocida":        `{"inicio":"2026-11-02T00:00:00Z","fin":"2026-12-31T00:00:00Z","otra":1}`,
		"política incompleta":      `{"inicio":"2026-11-02T00:00:00Z","causa_fin":"reincorporacion_titular","politica_fin":{"fecha_fin":"no_aplica"}}`,
		"política con clave ajena": `{"inicio":"2026-11-02T00:00:00Z","causa_fin":"x1","politica_fin":{"otra":1}}`,
	} {
		if _, err := decodificar(fila(periodo)); err == nil {
			t.Fatalf("%s aceptada", nombre)
		}
	}
	if e, err := decodificar(fila(`{"inicio":"2026-11-02T00:00:00Z","causa_fin":"Causa Libre"}`)); err != nil || e.Valido() {
		t.Fatalf("causa de fin con forma inválida aceptada: %v", err)
	}
}

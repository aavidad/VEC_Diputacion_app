package ports

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func solicitudDerivacionPrueba(t *testing.T) (SolicitudDerivacionNombramiento, domain.Expediente, string) {
	t.Helper()
	b, err := os.ReadFile("../domain/testdata/expediente_nombramiento_v7.json")
	if err != nil {
		t.Fatal(err)
	}
	var padre domain.Expediente
	if err = json.Unmarshal(b, &padre); err != nil {
		t.Fatal(err)
	}
	padre.Actuaciones = padre.Actuaciones[:6]
	padre.Version = 6
	padre.FaseActual = domain.FaseFiscalizacion
	padre.ActualizadoEn = padre.Actuaciones[5].RealizadaEn
	padre.Solicitud.NumeroPersonas = 2
	padre.Solicitud.JornadaMinutos = 420
	padre.Solicitud.PuestoSolicitado = "puesto:administrativo:01"
	if err = padre.Validar(); err != nil {
		t.Fatalf("fixture padre: %v", err)
	}
	b, err = json.Marshal(padre)
	if err != nil {
		t.Fatal(err)
	}
	s := SolicitudDerivacionNombramiento{
		Esquema:         EsquemaSolicitudDerivacionNombramiento,
		OrganizacionRef: padre.OrganizacionRef, PadreRef: padre.Referencia, VersionPadre: padre.Version,
		HijoRef:   HijoRefDerivacionNombramiento(padre.OrganizacionRef, padre.Referencia, 1),
		OfertaRef: "oferta:" + strings.Repeat("a", 64), NumeroPlaza: 1, SecuenciaPlaza: 1,
		ParticipacionRef: "participacion:ct:sintetica:01", ReciboAdjudicacionRef: "recibo:adjudicacion:sintetica:01",
		ClaveIdempotencia: "a1234567-1234-4123-8123-123456789abc",
		UnidadRef:         padre.Asignacion.UnidadRef,
	}
	return s, padre, string(b)
}

func TestDerivacionCanonYResultadoExacto(t *testing.T) {
	s, padre, preimagen := solicitudDerivacionPrueba(t)
	canon, err := s.Canon()
	if err != nil {
		t.Fatal(err)
	}
	var campos map[string]any
	if err = json.Unmarshal(canon, &campos); err != nil || len(campos) != 12 ||
		campos["padre_canonico"] != nil || campos["hijo_ref"] != s.HijoRef {
		t.Fatalf("canon inesperado: %v", err)
	}
	instante := padre.ActualizadoEn.Add(time.Minute).Truncate(time.Second)
	r := ResultadoDerivacionNombramiento{
		Esquema: EsquemaResultadoDerivacionNombramiento, Estado: "confirmado", PadreRef: s.PadreRef,
		HijoRef: s.HijoRef, NumeroVisible: padre.NumeroVisible + "-P1", NumeroPlaza: 1,
		TotalPersonas: 2, Version: 6, AceptacionRef: "aceptacion:ct:sintetica:01",
		ReciboRef: "recibo:derivacion:sintetica:01", AuditoriaRef: "auditoria:ct:sintetica:01", ConfirmadaEn: instante,
	}
	r.Expediente, err = domain.NuevoExpedienteDerivado(domain.AltaExpedienteDerivado{
		Padre: padre, PreimagenPadreJSON: preimagen, Referencia: r.HijoRef, OfertaRef: s.OfertaRef, NumeroPlaza: 1,
		ActorRef: "actor:rrhh:sintetico:01", ReciboRef: r.ReciboRef, CreadaEn: instante,
	})
	if err != nil || r.ValidarPara(s) != nil {
		t.Fatalf("resultado válido: %v", err)
	}
	// PostgreSQL serializa timestamptz(6) con seis decimales incluso para un
	// segundo exacto; Go lo codifica sin fracción. Es el mismo instante.
	bruto, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	bruto = []byte(strings.ReplaceAll(string(bruto), instante.Format(time.RFC3339),
		instante.Format("2006-01-02T15:04:05")+".000000Z"))
	var desdeSQL ResultadoDerivacionNombramiento
	if json.Unmarshal(bruto, &desdeSQL) != nil || desdeSQL.ValidarPara(s) != nil {
		t.Fatal("se rechazó el mismo instante con seis decimales")
	}
	desdeSQL.Expediente.Solicitud.Periodo.Inicio = desdeSQL.Expediente.Solicitud.Periodo.Inicio.Add(time.Hour)
	if desdeSQL.ValidarPara(s) == nil {
		t.Fatal("se aceptó un período distinto del padre")
	}
	preimagenAlterada := r
	preimagenAlterada.Expediente = r.Expediente.Clonar()
	preimagenAlterada.Expediente.InstruccionCompartida.PreimagenPadreJSON += " "
	if preimagenAlterada.ValidarPara(s) == nil {
		t.Fatal("se aceptó una preimagen que no coincide con su huella")
	}
	periodoAlterado := r
	periodoAlterado.Expediente = r.Expediente.Clonar()
	periodoAlterado.Expediente.InstruccionCompartida.ExpedientePadre.Solicitud.Periodo.Inicio =
		periodoAlterado.Expediente.InstruccionCompartida.ExpedientePadre.Solicitud.Periodo.Inicio.Add(time.Hour)
	periodoAlterado.Expediente.Solicitud.Periodo.Inicio = periodoAlterado.Expediente.Solicitud.Periodo.Inicio.Add(time.Hour)
	if periodoAlterado.ValidarPara(s) == nil {
		t.Fatal("se aceptó padre e hijo alterados sin cambiar la preimagen SQL")
	}
	r.Estado = "replay_confirmado"
	if err = r.ValidarPara(s); err != nil {
		t.Fatalf("replay válido: %v", err)
	}
	r.Expediente.InstruccionCompartida.OfertaRef = "oferta:ct:ajena:01"
	if r.ValidarPara(s) == nil {
		t.Fatal("se aceptó un hijo ligado a otra oferta")
	}
	s.HijoRef = "expediente:nombramiento:ajeno"
	if _, err = s.Canon(); err == nil {
		t.Fatal("se aceptó hijo arbitrario")
	}
}

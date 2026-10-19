package bootstrap

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestContextoComprobacionProcedeDeEntradaPendienteYOrdenAuditada(t *testing.T) {
	orden := []byte(`{"ambito_ref":"ambito:granada","unidad_ref":"unidad:uno","correo":"dato-privado"}`)
	criterios := &puertosbolsa.CriteriosOfrecimientoAplicados{
		VersionPoliticaOfertas: 7, HuellaPoliticaOfertasSHA256: strings.Repeat("a", 64),
		ModalidadOfrecida: "sustitucion", CausaOfrecida: "baja",
	}
	registro := puertosbolsa.OrigenLlamamientoCT{
		BolsaRef: "bolsa:uno", CategoriaRef: "categoria:uno", VersionBolsa: 5,
		VersionInstantanea: 9, CriteriosOfrecimiento: criterios,
		Entradas: []puertosbolsa.EntradaOrigenLlamamientoCT{{OrdenActa: 2, ParticipacionRef: "participacion:real", CandidatoRef: "candidato:privado", Razon: "pendiente_comprobacion_rrhh"}},
	}
	_, inicial := primeraElegibleOrigenConstituidoCT(registro)
	var pendiente *ports.ErrorCriteriosOfrecimientoPendientesRRHH
	if !errors.As(inicial, &pendiente) {
		t.Fatalf("sin posición pendiente: %v", inicial)
	}
	origen := origenConstituidoVerificadoCT{registro: registro, ordenPermiso: orden,
		bolsa:       dominiobolsa.BolsaConstituida{BolsaRef: "bolsa:uno", Version: 5},
		instantanea: dominiobolsa.InstantaneaOrdenBolsa{Version: 9}}
	d := preparacionLlamamientoDesarrollo{categoria: "categoria:uno", necesidad: "necesidad:exp", unidad: "unidad:uno",
		expediente: ports.ExpedienteParaSeleccion{Fiscalizado: ctdomain.Expediente{Referencia: "expediente:uno", OrganizacionRef: "organizacion:uno", Version: 13}}}
	if err := enriquecerErrorCriteriosPendientesCT(pendiente, origen, d); !errors.Is(err, ports.ErrCriteriosOfrecimientoPendientesRRHH) {
		t.Fatalf("pausa perdida: %v", err)
	}
	c := pendiente.Comprobacion.Contexto
	if c == nil || c.ExpedienteRef != "expediente:uno" || c.NecesidadRef != "necesidad:exp" ||
		c.VersionExpediente != 13 || c.VersionBolsa != 5 || c.VersionInstantanea != 9 ||
		c.VersionPoliticaOfertas != 7 || c.ParticipacionRef != "participacion:real" ||
		c.AmbitoRef != "ambito:granada" || c.UnidadRef != "unidad:uno" || !c.ValidoPara(pendiente.Comprobacion) {
		t.Fatalf("contexto incorrecto: %+v", c)
	}
	if c.ParticipacionRef == registro.Entradas[0].CandidatoRef {
		t.Fatal("filtró candidato")
	}
	publicado, err := json.Marshal(pendiente.Comprobacion)
	if err != nil || strings.Contains(string(publicado), "candidato:privado") || strings.Contains(string(publicado), "dato-privado") {
		t.Fatalf("filtró datos privados: %s, %v", publicado, err)
	}

	// Las referencias reales de expediente son huellas SHA-256: algunas casan
	// por azar con el patrón de DNI y no deben perder la pausa ni el contexto.
	huella := "expediente:ct:e975732762ec" + strings.Repeat("0", 52)
	d.expediente.Fiscalizado.Referencia = huella
	pendiente.Comprobacion.Contexto = nil
	if err := enriquecerErrorCriteriosPendientesCT(pendiente, origen, d); !errors.Is(err, ports.ErrCriteriosOfrecimientoPendientesRRHH) ||
		pendiente.Comprobacion.Contexto == nil || pendiente.Comprobacion.Contexto.ExpedienteRef != huella {
		t.Fatalf("expediente con huella del sistema perdió el contexto: %v", err)
	}
	d.expediente.Fiscalizado.Referencia = "expediente:12345678Z"
	pendiente.Comprobacion.Contexto = nil
	if err := enriquecerErrorCriteriosPendientesCT(pendiente, origen, d); !errors.Is(err, ports.ErrRespuestaBolsaNoConfiable) {
		t.Fatalf("aceptó un documento en una referencia que no es huella: %v", err)
	}
	d.expediente.Fiscalizado.Referencia = "expediente:uno"

	for _, ordenInvalida := range []string{
		`{"ambito_ref":"ambito:ajeno","unidad_ref":"unidad:dos"}`,
		`{"ambito_ref":"dni:12345678Z","unidad_ref":"unidad:uno"}`,
		`{"unidad_ref":"unidad:uno"}`,
	} {
		origen.ordenPermiso = []byte(ordenInvalida)
		pendiente.Comprobacion.Contexto = nil
		if err := enriquecerErrorCriteriosPendientesCT(pendiente, origen, d); !errors.Is(err, ports.ErrRespuestaBolsaNoConfiable) || pendiente.Comprobacion.Contexto != nil {
			t.Fatalf("expuso contexto incoherente: %v", err)
		}
	}
}

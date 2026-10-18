package bootstrap

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

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
		expediente: ports.ExpedienteParaSeleccion{Fiscalizado: ctdomain.Expediente{Referencia: "expediente:uno", OrganizacionRef: "organizacion:uno", Version: 13,
			Analisis:   &ctdomain.AnalisisRRHH{CategoriaRef: "categoria:uno", ModalidadClave: "sustitucion", CausaClave: "baja"},
			Asignacion: &ctdomain.AsignacionUnidad{UnidadRef: "unidad:uno"}}}}
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
	if pendiente.Comprobacion.Estado != nil {
		t.Fatal("lectura histórica fabricó ausencia de acto")
	}
	if c.ParticipacionRef == registro.Entradas[0].CandidatoRef {
		t.Fatal("filtró candidato")
	}
	publicado, err := json.Marshal(pendiente.Comprobacion)
	if err != nil || strings.Contains(string(publicado), "candidato:privado") || strings.Contains(string(publicado), "dato-privado") {
		t.Fatalf("filtró datos privados: %s, %v", publicado, err)
	}

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
	origen.ordenPermiso = orden
	d.expediente.Fiscalizado.Analisis.CausaClave = "otra_causa"
	if err := enriquecerErrorCriteriosPendientesCT(pendiente, origen, d); !errors.Is(err, ports.ErrRespuestaBolsaNoConfiable) || pendiente.Comprobacion.Contexto != nil {
		t.Fatalf("mezcló causa fiscalizada con criterio de Bolsa: %v", err)
	}
}

func TestContextoComprobacionConservaAusenciaActoYUltimoActo(t *testing.T) {
	criterios := &puertosbolsa.CriteriosOfrecimientoAplicados{
		VersionPoliticaOfertas: 7, HuellaPoliticaOfertasSHA256: strings.Repeat("a", 64),
		ModalidadOfrecida: "sustitucion", CausaOfrecida: "baja",
	}
	registro := puertosbolsa.OrigenLlamamientoCT{BolsaRef: "bolsa:uno", CategoriaRef: "categoria:uno",
		VersionBolsa: 5, VersionInstantanea: 9, CriteriosOfrecimiento: criterios,
		Entradas: []puertosbolsa.EntradaOrigenLlamamientoCT{{OrdenActa: 2, ParticipacionRef: "participacion:real", Razon: "pendiente_comprobacion_rrhh"}}}
	origen := origenConstituidoVerificadoCT{registro: registro,
		ordenPermiso: []byte(`{"ambito_ref":"ambito:uno","unidad_ref":"unidad:uno"}`),
		bolsa:        dominiobolsa.BolsaConstituida{BolsaRef: "bolsa:uno", Version: 5},
		instantanea:  dominiobolsa.InstantaneaOrdenBolsa{Version: 9}}
	d := preparacionLlamamientoDesarrollo{categoria: "categoria:uno", necesidad: "necesidad:uno", unidad: "unidad:uno",
		expediente: ports.ExpedienteParaSeleccion{Fiscalizado: ctdomain.Expediente{Referencia: "expediente:uno", OrganizacionRef: "organizacion:uno", Version: 13,
			Analisis:   &ctdomain.AnalisisRRHH{CategoriaRef: "categoria:uno", ModalidadClave: "sustitucion", CausaClave: "baja"},
			Asignacion: &ctdomain.AsignacionUnidad{UnidadRef: "unidad:uno"}}}}
	proyectar := func(estado *puertosbolsa.EstadoComprobacionOfrecimiento) (*ports.ComprobacionOfrecimientoPendiente, error) {
		origen.registro.Entradas[0].ComprobacionOfrecimiento = estado
		_, err := primeraElegibleOrigenConstituidoCT(origen.registro)
		var pendiente *ports.ErrorCriteriosOfrecimientoPendientesRRHH
		if !errors.As(err, &pendiente) {
			return nil, err
		}
		err = enriquecerErrorCriteriosPendientesCT(pendiente, origen, d)
		return &pendiente.Comprobacion, err
	}
	ausencia, err := proyectar(&puertosbolsa.EstadoComprobacionOfrecimiento{VersionEsperada: 0})
	if !errors.Is(err, ports.ErrCriteriosOfrecimientoPendientesRRHH) || ausencia.Estado == nil {
		t.Fatalf("perdió ausencia acreditada: %v", err)
	}
	jsonAusencia, err := json.Marshal(ausencia.Estado)
	if err != nil || string(jsonAusencia) != `{"version_esperada":0}` {
		t.Fatalf("ausencia no exacta: %s %v", jsonAusencia, err)
	}
	vigente := false
	ref := "comprobacion:" + strings.Repeat("a", 64)
	acto := &puertosbolsa.EstadoComprobacionOfrecimiento{VersionEsperada: 3, ComprobacionRef: ref,
		ReciboRef: "recibo:" + ref, RegistradaEn: time.Date(2026, 10, 18, 8, 0, 0, 0, time.UTC),
		VersionPoliticaOfertas: 6, Resultado: "pendiente", MotivoClave: "antecedentes_incompletos", Vigente: &vigente}
	ultimo, err := proyectar(acto)
	if !errors.Is(err, ports.ErrCriteriosOfrecimientoPendientesRRHH) || ultimo.Estado == nil ||
		ultimo.Estado.VersionEsperada != 3 || ultimo.Estado.Vigente == nil || *ultimo.Estado.Vigente {
		t.Fatalf("perdió último acto anterior: %+v %v", ultimo, err)
	}
	actual := true
	acto.Vigente = &actual
	mal, err := proyectar(acto)
	if !errors.Is(err, ports.ErrRespuestaBolsaNoConfiable) || mal.Estado != nil {
		t.Fatalf("publicó acto vigente con política ajena: %+v %v", mal, err)
	}
}

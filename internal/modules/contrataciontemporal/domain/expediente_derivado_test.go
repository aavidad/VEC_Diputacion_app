package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func padreDerivacionPrueba(t *testing.T) Expediente {
	t.Helper()
	p := expedienteFiscalizablePrueba(t)
	p.Solicitud.JornadaMinutos = 420
	p.Solicitud.NumeroPersonas = 2
	p.Solicitud.PuestoSolicitado = "puesto:administrativo:01"
	instante := p.ActualizadoEn.Add(time.Minute)
	a := DatosActuacion{
		AccionClave: AccionRegistrarFiscalizacion,
		ActorRef:    "actor:intervencion:sintetico:01", UnidadRef: "unidad:intervencion:sintetica:01",
		ReciboRef: "recibo:fiscalizacion:padre:01", RealizadaEn: instante,
		FaseDestino: FaseFiscalizacion, EstadoDestino: EstadoEnCurso,
		DocumentosRef: []string{p.InformeJuridico.DocumentoRef},
	}
	fiscalizado, err := p.RegistrarFiscalizacion(p.Version, DatosRegistrarFiscalizacion{
		FiscalizacionRef: "fiscalizacion:padre:01", Resultado: FiscalizacionFavorable,
		UnidadFiscalizadoraRef: a.UnidadRef, FiscalizadaEn: instante,
	}, a)
	if err != nil {
		t.Fatal(err)
	}
	return fiscalizado
}

func altaDerivacionPrueba(p Expediente, plaza uint32) AltaExpedienteDerivado {
	preimagen, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	return AltaExpedienteDerivado{
		Padre: p, Referencia: "expediente:hijo:plaza:01", OfertaRef: "oferta:padre:01",
		NumeroPlaza: plaza, ActorRef: "actor:rrhh:derivacion:01",
		ReciboRef: "recibo:derivacion:plaza:01", CreadaEn: p.ActualizadoEn.Add(time.Minute),
		PreimagenPadreJSON: string(preimagen),
	}
}

func TestDerivacionConservaInstruccionYAvanzaAVersionSiete(t *testing.T) {
	p := padreDerivacionPrueba(t)
	h, err := NuevoExpedienteDerivado(altaDerivacionPrueba(p, 1))
	if err != nil || h.Validar() != nil {
		t.Fatalf("derivar: %v", err)
	}
	if h.Version != 6 || len(h.Actuaciones) != 1 || h.Actuaciones[0].Secuencia != 6 ||
		h.Actuaciones[0].AccionClave != AccionDerivarNombramiento ||
		h.Actuaciones[0].UnidadRef != p.Asignacion.UnidadRef ||
		h.Actuaciones[0].FaseDestino != FaseFiscalizacion ||
		len(h.Actuaciones[0].DocumentosRef) != 0 ||
		h.Actuaciones[0].ActorRef != "actor:rrhh:derivacion:01" ||
		h.Actuaciones[0].ReciboRef != "recibo:derivacion:plaza:01" ||
		h.InstruccionCompartida.ExpedientePadre.Actuaciones[5].ReciboRef != p.Actuaciones[5].ReciboRef ||
		h.Solicitud.NumeroPersonas != 1 {
		t.Fatalf("historia derivada incorrecta: %+v", h)
	}
	contenido, err := json.Marshal(h)
	var restaurado Expediente
	if err != nil || json.Unmarshal(contenido, &restaurado) != nil || restaurado.Validar() != nil {
		t.Fatal("la instrucción compartida no sobrevive a JSON")
	}
	a := DatosActuacion{
		AccionClave: "contratacion_temporal.nombramiento.proponer", ActorRef: "actor:rrhh:propuesta:01",
		UnidadRef: h.Asignacion.UnidadRef, ReciboRef: "recibo:propuesta:hijo:01",
		RealizadaEn: h.ActualizadoEn.Add(time.Minute),
		FaseDestino: FaseNombramiento, EstadoDestino: EstadoEnCurso,
	}
	v7, err := h.confirmarTransicion(a)
	if err != nil || v7.Validar() != nil || v7.Version != 7 || len(v7.Actuaciones) != 2 ||
		v7.Actuaciones[1].Secuencia != 7 || v7.Actuaciones[1].VersionExpediente != 7 {
		t.Fatalf("versión 7: %v %+v", err, v7.Actuaciones)
	}
}

func TestDerivacionRechazaPadreInvalidoYCiclo(t *testing.T) {
	p := padreDerivacionPrueba(t)
	casos := map[string]func(*AltaExpedienteDerivado){
		"sin fiscalización":       func(a *AltaExpedienteDerivado) { a.Padre.Fiscalizacion = nil },
		"plaza fuera de petición": func(a *AltaExpedienteDerivado) { a.NumeroPlaza = 3 },
		"recibo del padre":        func(a *AltaExpedienteDerivado) { a.ReciboRef = p.Actuaciones[5].ReciboRef },
		"mismo expediente":        func(a *AltaExpedienteDerivado) { a.Referencia = p.Referencia },
		"actor inválido":          func(a *AltaExpedienteDerivado) { a.ActorRef = "" },
		"instante anterior":       func(a *AltaExpedienteDerivado) { a.CreadaEn = p.ActualizadoEn.Add(-time.Minute) },
		"padre derivado":          func(a *AltaExpedienteDerivado) { a.Padre.InstruccionCompartida = &InstruccionCompartida{} },
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			a := altaDerivacionPrueba(p, 1)
			cambiar(&a)
			if _, err := NuevoExpedienteDerivado(a); err == nil {
				t.Fatal("se admitió la derivación")
			}
		})
	}
	// La guarda debe actuar antes de Validar incluso ante una referencia cíclica.
	a := altaDerivacionPrueba(p, 1)
	a.Padre.InstruccionCompartida = &InstruccionCompartida{}
	a.Padre.InstruccionCompartida.ExpedientePadre = a.Padre
	if _, err := NuevoExpedienteDerivado(a); err == nil {
		t.Fatal("se admitió un padre cíclico")
	}
}

func TestDerivacionRechazaInstruccionDivergenteYSeparaDosPlazas(t *testing.T) {
	p := padreDerivacionPrueba(t)
	h1, err := NuevoExpedienteDerivado(altaDerivacionPrueba(p, 1))
	if err != nil {
		t.Fatal(err)
	}
	a2 := altaDerivacionPrueba(p, 2)
	a2.Referencia = "expediente:hijo:plaza:02"
	a2.ReciboRef = "recibo:derivacion:plaza:02"
	h2, err := NuevoExpedienteDerivado(a2)
	if err != nil || h2.Validar() != nil || h1.InstruccionCompartida.NumeroPlaza != 1 ||
		h2.InstruccionCompartida.NumeroPlaza != 2 {
		t.Fatalf("segunda plaza: %v", err)
	}
	h2.InstruccionCompartida.ExpedientePadre.Actuaciones[5].ReciboRef = "recibo:alterado"
	if h2.Validar() == nil || h1.Validar() != nil || p.Validar() != nil {
		t.Fatal("una plaza mutó la instrucción de otra")
	}
	h1.Analisis.ModalidadClave = "otra_modalidad"
	if h1.Validar() == nil {
		t.Fatal("se admitió un análisis heredado divergente")
	}
}

func TestDerivacionAdmiteNumeroPadreMaximoYPlazaCien(t *testing.T) {
	p := padreDerivacionPrueba(t)
	p.NumeroVisible = "2026/" + strings.Repeat("X", 40)
	p.Solicitud.NumeroPersonas = 100
	if p.Validar() != nil {
		t.Fatal("padre válido de longitud máxima")
	}
	a := altaDerivacionPrueba(p, 100)
	h, err := NuevoExpedienteDerivado(a)
	if err != nil || h.Validar() != nil || len(h.NumeroVisible) != 50 {
		t.Fatalf("número derivado de 50 caracteres: %v", err)
	}
}

func TestExpedienteDerivadoValidaPreimagenExacta(t *testing.T) {
	p := padreDerivacionPrueba(t)
	a := altaDerivacionPrueba(p, 1)
	rawEquivalente := strings.ReplaceAll(a.PreimagenPadreJSON, "Z\"", ".000000Z\"")
	if rawEquivalente == a.PreimagenPadreJSON {
		t.Fatal("la preimagen de prueba no cambió")
	}
	a.PreimagenPadreJSON = rawEquivalente
	hijo, err := NuevoExpedienteDerivado(a)
	if err != nil || hijo.Validar() != nil ||
		hijo.InstruccionCompartida.HuellaPadreSHA256 != huellaPreimagenPadre(rawEquivalente) {
		t.Fatalf("representación temporal equivalente rechazada: %v", err)
	}
	hijo.InstruccionCompartida.HuellaPadreSHA256 = strings.Repeat("a", 64)
	if hijo.Validar() == nil {
		t.Fatal("se admitió una huella ajena a la preimagen")
	}
	hijo.InstruccionCompartida.PreimagenPadreJSON = strings.Replace(rawEquivalente, "2026-08-01", "2026-08-02", 1)
	hijo.InstruccionCompartida.HuellaPadreSHA256 = huellaPreimagenPadre(hijo.InstruccionCompartida.PreimagenPadreJSON)
	if hijo.Validar() == nil {
		t.Fatal("se admitió un periodo divergente con huella recalculada")
	}
}

func TestExpedienteDerivadoRechazaPreimagenDivergenteYCamposAjenos(t *testing.T) {
	p := padreDerivacionPrueba(t)
	casos := map[string]func(*AltaExpedienteDerivado){
		"ausente":          func(a *AltaExpedienteDerivado) { a.PreimagenPadreJSON = "" },
		"demasiado grande": func(a *AltaExpedienteDerivado) { a.PreimagenPadreJSON = strings.Repeat("x", 262145) },
		"utf8 inválido":    func(a *AltaExpedienteDerivado) { a.PreimagenPadreJSON = string([]byte{0xff, '{', '}'}) },
		"campo desconocido": func(a *AltaExpedienteDerivado) {
			a.PreimagenPadreJSON = strings.TrimSuffix(a.PreimagenPadreJSON, "}") + `,"campo_ajeno":1}`
		},
		"campo anidado desconocido": func(a *AltaExpedienteDerivado) {
			a.PreimagenPadreJSON = strings.Replace(a.PreimagenPadreJSON, `"solicitud":{`, `"solicitud":{"campo_ajeno":1,`, 1)
		},
		"periodo divergente": func(a *AltaExpedienteDerivado) {
			a.PreimagenPadreJSON = strings.Replace(a.PreimagenPadreJSON, "2026-08-01", "2026-08-02", 1)
		},
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			a := altaDerivacionPrueba(p, 1)
			alterar(&a)
			if _, err := NuevoExpedienteDerivado(a); err == nil {
				t.Fatal("se admitió una preimagen ajena")
			}
		})
	}
}

func TestExpedienteDerivadoPropagaFalloDeDecodificacion(t *testing.T) {
	a := altaDerivacionPrueba(padreDerivacionPrueba(t), 1)
	a.PreimagenPadreJSON = "{invalido}"
	var falloJSON *json.SyntaxError
	if _, err := NuevoExpedienteDerivado(a); !errors.Is(err, ErrExpedienteInvalido) || !errors.As(err, &falloJSON) {
		t.Fatalf("el fallo JSON debe llegar a quien valida: %v", err)
	}
	hijo, err := NuevoExpedienteDerivado(altaDerivacionPrueba(padreDerivacionPrueba(t), 1))
	if err != nil {
		t.Fatal(err)
	}
	hijo.InstruccionCompartida.PreimagenPadreJSON = "{invalido}"
	if err := hijo.Validar(); !errors.Is(err, ErrExpedienteInvalido) || !errors.As(err, &falloJSON) {
		t.Fatalf("el error de la instantánea debe propagarse: %v", err)
	}
}

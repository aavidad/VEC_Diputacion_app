package domain

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
)

func informeNuevoPrueba(t *testing.T, e Expediente, sufijo string) (InformeJuridicoEmitido, DatosActuacion) {
	t.Helper()
	datos := datosInformeJuridicoPrueba()
	datos.ExpedienteRef = e.Referencia
	datos.VersionEsperadaExpediente = e.Version
	borrador, err := NuevoBorradorInformeJuridico(datos)
	if err != nil {
		t.Fatal(err)
	}
	emitidoEn := e.ActualizadoEn.Add(time.Minute)
	informe := InformeJuridicoEmitido{
		Borrador: borrador.Estado(), InformeRef: "informe:juridico:nuevo:" + sufijo,
		DocumentoRef: "documento:informe:juridico:nuevo:" + sufijo, VersionDocumento: 1,
		HuellaDocumentoSHA256: cadena64("e"), EmitidoEn: emitidoEn,
	}
	retorno := ""
	if e.Fiscalizacion != nil && e.Fiscalizacion.Retorno != nil {
		retorno = e.Fiscalizacion.Retorno.RetornoRef
	}
	return informe, DatosActuacion{
		AccionClave: AccionEmitirInformeJuridico, ActorRef: "actor:juridico:sintetico:02",
		UnidadRef: e.Asignacion.UnidadRef, ReciboRef: "recibo:informe:nuevo:" + sufijo,
		RealizadaEn: emitidoEn, FaseDestino: FaseSubsanacionUnidad, EstadoDestino: EstadoIncidencia,
		DocumentosRef: []string{informe.DocumentoRef}, RetornoRef: retorno,
	}
}

func refiscalizarPrueba(e Expediente, resultado ResultadoFiscalizacion, sufijo, retornoNuevo string) (Expediente, error) {
	instante := e.ActualizadoEn.Add(time.Minute)
	fase, estado, observaciones := FaseFiscalizacion, EstadoEnCurso, ""
	if resultado == FiscalizacionDesfavorable {
		fase, estado, observaciones = FaseSubsanacionUnidad, EstadoIncidencia, "Nuevo reparo sintético."
	}
	return e.RegistrarFiscalizacion(e.Version,
		DatosRegistrarFiscalizacion{
			FiscalizacionRef: "fiscalizacion:reemision:" + sufijo, Resultado: resultado,
			UnidadFiscalizadoraRef: "unidad:intervencion:sintetica:01", Observaciones: observaciones,
			FiscalizadaEn: instante, RetornoRef: retornoNuevo,
		},
		DatosActuacion{
			AccionClave: AccionRegistrarFiscalizacion, ActorRef: "actor:intervencion:sintetico:01",
			UnidadRef: "unidad:intervencion:sintetica:01", ReciboRef: "recibo:fiscalizacion:reemision:" + sufijo,
			RealizadaEn: instante, FaseDestino: fase, EstadoDestino: estado, Observaciones: observaciones,
			DocumentosRef: []string{e.InformeJuridico.DocumentoRef}, RetornoRef: e.Fiscalizacion.Retorno.RetornoRef,
		},
	)
}

// Recorrido completo del dominio: desfavorable → subsanación → informe nuevo
// → nueva fiscalización favorable ligada al informe nuevo → sigue.
func TestReemisionInformeTrasSubsanacionRecorridoCompleto(t *testing.T) {
	antecedente := expedienteConSubsanacionParaRefiscalizacion(t)
	informeAnterior := *antecedente.InformeJuridico
	if !antecedente.RefiscalizacionEsperaInformeNuevo() || antecedente.InformeReemitidoTrasSubsanacion() {
		t.Fatal("tras subsanar debe esperarse el informe nuevo")
	}
	informe, actuacion := informeNuevoPrueba(t, antecedente, "01")
	conInforme, err := antecedente.ReemitirInformeJuridicoTrasSubsanacion(antecedente.Version, informe, actuacion)
	if err != nil {
		t.Fatalf("reemitir informe: %v", err)
	}
	if conInforme.Validar() != nil || conInforme.Version != antecedente.Version+1 ||
		conInforme.FaseActual != FaseSubsanacionUnidad || conInforme.EstadoActual != EstadoIncidencia ||
		!reflect.DeepEqual(conInforme.Fiscalizacion, antecedente.Fiscalizacion) ||
		conInforme.InformeJuridico.InformeRef != informe.InformeRef ||
		conInforme.InformeJuridico.Sustituye == nil ||
		conInforme.InformeJuridico.Sustituye.InformeRef != informeAnterior.InformeRef ||
		conInforme.InformeJuridico.Sustituye.RetornoRef != antecedente.Fiscalizacion.Retorno.RetornoRef ||
		!conInforme.InformeReemitidoTrasSubsanacion() || conInforme.RefiscalizacionEsperaInformeNuevo() {
		t.Fatalf("informe nuevo mal registrado: %#v", conInforme.InformeJuridico)
	}
	// Se conserva por JSON y se restaura válido.
	contenido, err := json.Marshal(conInforme)
	if err != nil {
		t.Fatal(err)
	}
	var restaurado Expediente
	if err := json.Unmarshal(contenido, &restaurado); err != nil || restaurado.Validar() != nil ||
		!reflect.DeepEqual(restaurado, conInforme) {
		t.Fatalf("el expediente con informe nuevo no se restaura: %v", err)
	}
	// Una segunda emisión para el mismo reparo se rechaza.
	otro, otraActuacion := informeNuevoPrueba(t, conInforme, "02")
	if _, err := conInforme.ReemitirInformeJuridicoTrasSubsanacion(conInforme.Version, otro, otraActuacion); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("segundo informe nuevo aceptado: %v", err)
	}
	favorable, err := refiscalizarPrueba(conInforme, FiscalizacionFavorable, "fav", "")
	if err != nil {
		t.Fatalf("nueva fiscalización con informe nuevo: %v", err)
	}
	if favorable.Validar() != nil || favorable.FaseActual != FaseFiscalizacion ||
		favorable.Fiscalizacion.InformeJuridicoRef != informe.InformeRef ||
		favorable.Fiscalizacion.DocumentoInformeRef != informe.DocumentoRef {
		t.Fatalf("la nueva fiscalización no cita el informe nuevo: %#v", favorable.Fiscalizacion)
	}
}

// Sin catálogo que lo exija, la nueva fiscalización con el mismo informe se
// conserva (conducta de siempre): el dominio no la bloquea.
func TestRefiscalizacionSinInformeNuevoSigueAdmitida(t *testing.T) {
	antecedente := expedienteConSubsanacionParaRefiscalizacion(t)
	if _, err := refiscalizarPrueba(antecedente, FiscalizacionFavorable, "sin", ""); err != nil {
		t.Fatalf("refiscalización con el mismo informe rechazada: %v", err)
	}
}

// Un nuevo reparo tras el informe nuevo abre otro retorno y admite otro
// informe nuevo después de su subsanación.
func TestReemisionTrasSegundoReparo(t *testing.T) {
	e := expedienteConSubsanacionParaRefiscalizacion(t)
	informe, actuacion := informeNuevoPrueba(t, e, "01")
	e, err := e.ReemitirInformeJuridicoTrasSubsanacion(e.Version, informe, actuacion)
	if err != nil {
		t.Fatal(err)
	}
	e, err = refiscalizarPrueba(e, FiscalizacionDesfavorable, "des2", "retorno:fiscalizacion:reemision:02")
	if err != nil {
		t.Fatalf("segundo reparo: %v", err)
	}
	if e.Validar() != nil || e.InformeReemitidoTrasSubsanacion() || e.PuedeReemitirInformeTrasSubsanacion() {
		t.Fatal("antes de subsanar el segundo reparo no cabe informe nuevo")
	}
	instante := e.ActualizadoEn.Add(time.Minute)
	e, err = e.RegistrarSubsanacionReparo(e.Version,
		DatosSubsanacionReparo{RetornoRef: e.Fiscalizacion.Retorno.RetornoRef, Observaciones: "Segunda corrección sintética."},
		DatosActuacion{AccionClave: AccionRegistrarSubsanacionReparo, ActorRef: "actor:unidad:sintetico:01",
			UnidadRef: e.Asignacion.UnidadRef, ReciboRef: "recibo:subsanacion:segunda", RealizadaEn: instante,
			FaseDestino: FaseSubsanacionUnidad, EstadoDestino: EstadoIncidencia, Observaciones: "Segunda corrección sintética.",
			RetornoRef: e.Fiscalizacion.Retorno.RetornoRef})
	if err != nil {
		t.Fatal(err)
	}
	if !e.RefiscalizacionEsperaInformeNuevo() {
		t.Fatal("tras la segunda subsanación debe esperarse otro informe nuevo")
	}
	tercero, actuacionTercero := informeNuevoPrueba(t, e, "03")
	if _, err := e.ReemitirInformeJuridicoTrasSubsanacion(e.Version, tercero, actuacionTercero); err != nil {
		t.Fatalf("tercer informe: %v", err)
	}
}

func TestReemisionInformeRechazaFormasAjenas(t *testing.T) {
	base := expedienteConSubsanacionParaRefiscalizacion(t)
	casos := map[string]func(*Expediente, *InformeJuridicoEmitido, *DatosActuacion){
		"sin subsanar": func(e *Expediente, _ *InformeJuridicoEmitido, _ *DatosActuacion) {
			*e = expedienteFiscalizablePrueba(t)
		},
		"otro retorno":    func(_ *Expediente, _ *InformeJuridicoEmitido, a *DatosActuacion) { a.RetornoRef = "retorno:ajeno:01" },
		"sale de la fase": func(_ *Expediente, _ *InformeJuridicoEmitido, a *DatosActuacion) { a.FaseDestino = FaseFiscalizacion },
		"mismo informe": func(e *Expediente, i *InformeJuridicoEmitido, a *DatosActuacion) {
			i.InformeRef = e.InformeJuridico.InformeRef
		},
		"mismo documento": func(e *Expediente, i *InformeJuridicoEmitido, a *DatosActuacion) {
			i.DocumentoRef = e.InformeJuridico.DocumentoRef
			a.DocumentosRef = []string{i.DocumentoRef}
		},
		"otra unidad": func(_ *Expediente, _ *InformeJuridicoEmitido, a *DatosActuacion) { a.UnidadRef = "unidad:ajena:01" },
		"sustitucion aportada": func(_ *Expediente, i *InformeJuridicoEmitido, _ *DatosActuacion) {
			i.Sustituye = &SustitucionInformeJuridico{InformeRef: "informe:x:01", DocumentoRef: "documento:x:01", RetornoRef: "retorno:x:01"}
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			e := base.Clonar()
			informe, actuacion := informeNuevoPrueba(t, e, "neg")
			mutar(&e, &informe, &actuacion)
			if _, err := e.ReemitirInformeJuridicoTrasSubsanacion(e.Version, informe, actuacion); !errors.Is(err, ErrTransicionInvalida) {
				t.Fatalf("aceptado: %v", err)
			}
		})
	}
	// El informe inicial no admite la forma del informe nuevo.
	fiscalizable := expedienteFiscalizablePrueba(t)
	if fiscalizable.InformeJuridico.Sustituye != nil || fiscalizable.RefiscalizacionEsperaInformeNuevo() {
		t.Fatal("el informe inicial no sustituye a ninguno")
	}
}

// La regla del expediente solo admite que la fiscalización cite un informe
// distinto del vigente si este lo sustituyó tras subsanar ese mismo reparo.
func TestValidarRechazaInformeSustituidoAdulterado(t *testing.T) {
	e := expedienteConSubsanacionParaRefiscalizacion(t)
	informe, actuacion := informeNuevoPrueba(t, e, "01")
	e, err := e.ReemitirInformeJuridicoTrasSubsanacion(e.Version, informe, actuacion)
	if err != nil {
		t.Fatal(err)
	}
	adulterado := e.Clonar()
	adulterado.InformeJuridico.Sustituye.InformeRef = "informe:ajeno:01"
	if adulterado.Validar() == nil {
		t.Fatal("se aceptó una sustitución que no cita el informe fiscalizado")
	}
	sinSustitucion := e.Clonar()
	sinSustitucion.InformeJuridico.Sustituye = nil
	if sinSustitucion.Validar() == nil {
		t.Fatal("se aceptó un informe distinto del fiscalizado sin sustitución")
	}
}

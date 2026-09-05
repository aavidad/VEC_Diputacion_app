package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type relojPuenteBolsaPrueba struct {
	instante time.Time
	paso     time.Duration
}

func (r *relojPuenteBolsaPrueba) Ahora() time.Time {
	t := r.instante
	r.instante = t.Add(r.paso)
	return t
}

// Dobles de unidad: fallan siempre al acceder al almacenamiento/permiso.
// No sustituyen la dinámica PostgreSQL ni acreditan persistencia.
type repositorioPuenteBolsaFallido struct{ llamadas int }

func (r *repositorioPuenteBolsaFallido) BuscarOperacion(context.Context, string) (puertosbolsa.RegistroLlamamientoDesarrollo, bool, error) {
	r.llamadas++
	return puertosbolsa.RegistroLlamamientoDesarrollo{}, false, errors.New("almacenamiento de prueba no disponible")
}
func (r *repositorioPuenteBolsaFallido) Guardar(context.Context, puertosbolsa.RegistroLlamamientoDesarrollo, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (puertosbolsa.ReciboLlamamientoDesarrollo, error) {
	r.llamadas++
	return puertosbolsa.ReciboLlamamientoDesarrollo{}, errors.New("no guardar en prueba")
}

type autorizadorPuenteBolsaDenegado struct{}

func (autorizadorPuenteBolsaDenegado) AutorizarOperacion(context.Context, string, dominiovec.RecursoAutorizable) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errors.New("permiso de prueba denegado")
}

func expedientePuenteBolsaPrueba(t *testing.T) ports.ExpedienteParaSeleccion {
	t.Helper()
	base := time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC)
	periodo := domain.PeriodoPrevisto{Inicio: time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC), Fin: time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC)}
	h := strings.Repeat("a", 64)
	a := func(accion string, fase domain.ClaveFase, paso time.Duration) domain.DatosActuacion {
		return domain.DatosActuacion{AccionClave: domain.ClaveCatalogo(accion), ActorRef: "actor:rrhh:sintetico", UnidadRef: "unidad:rrhh:sintetica",
			ReciboRef: "recibo:" + accion, RealizadaEn: base.Add(paso), FaseDestino: fase, EstadoDestino: domain.EstadoEnCurso}
	}
	e, err := domain.NuevoExpediente(domain.AltaExpediente{
		Referencia: "expediente:sintetico:puente", OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, NumeroVisible: "2026/123",
		Flujo: domain.ReferenciaFlujo{DefinicionRef: "flujo:sintetico", Version: 1, HuellaSHA256: h}, FaseInicial: "solicitud",
		Solicitud: domain.SolicitudCentro{CentroRef: "centro:sintetico", ContactoRef: "contacto:sintetico", CategoriaRef: "categoria:desarrollo:c2", GrupoSubgrupo: "C2",
			MotivoClave: "sustitucion", Detalle: "Solicitud sintética del puente.", Periodo: periodo},
		Actuacion: a("solicitud.registrada", "solicitud", 0),
	})
	if err != nil {
		t.Fatal("alta fixture:", err)
	}
	e, err = e.RegistrarAnalisis(1, domain.AnalisisRRHH{
		ModalidadClave: "sustitucion", CategoriaRef: "categoria:desarrollo:c2", GrupoSubgrupo: "C2", CausaClave: "incapacidad_temporal",
		Periodo: periodo, PorcentajeJornada: 10000,
		EntradaRCEsperada: domain.VinculoEntradaRC{Referencia: "entrada:rc:sintetica", HuellaSHA256: h},
		ValidacionRC: domain.ValidacionRC{Resultado: domain.RCNoRequerida, EntradaRef: "entrada:rc:sintetica", HuellaEntradaSHA256: h,
			FuenteRef: "fuente:rc:sintetica", ReciboRef: "recibo:rc:sintetica", ValidadaEn: base, Motivo: "Caso sintético sin RC."},
	}, a("analisis.validado", "gestion_bolsa", time.Minute))
	if err != nil {
		t.Fatal("analisis fixture:", err)
	}
	e, err = e.RegistrarViaCobertura(2, domain.DecisionViaCobertura{
		ViaClave: "bolsa_vigente", ProcedimientoRef: "procedimiento:sintetico",
		Comprobaciones: []domain.ComprobacionCobertura{{Clave: "existe_bolsa_vigente", Resultado: domain.ComprobacionAfirmativa, FuenteRef: "fuente:bolsa:sintetica", ReciboRef: "recibo:bolsa:sintetica", EvaluadaEn: base}},
		Motivacion:     "Bolsa de desarrollo.",
	}, a("cobertura.decidida", "asignacion_unidad", 2*time.Minute))
	if err != nil {
		t.Fatal("cobertura fixture:", err)
	}
	e, err = e.RegistrarAsignacion(3, domain.AsignacionUnidad{
		UnidadRef: unidadCoberturaContratacionTemporalDesarrollo, ResponsableRef: "responsable:sintetico", NotificacionRef: "notificacion:sintetica", AsignadaEn: base.Add(3 * time.Minute),
	}, a("unidad.asignada", "asignacion_unidad", 3*time.Minute))
	if err != nil {
		t.Fatal("asignacion fixture:", err)
	}
	borrador, err := domain.NuevoBorradorInformeJuridico(domain.DatosBorradorInformeJuridico{
		Canon: domain.CanonBorradorInformeJuridicoV1(), ExpedienteRef: e.Referencia, VersionEsperadaExpediente: 4,
		Plantilla:             domain.ReferenciaPlantillaInformeJuridico{PlantillaRef: "plantilla:sintetica", Version: 1, HuellaSHA256: h},
		ReferenciasNormativas: []domain.ReferenciaNormativaInformeJuridico{{NormaRef: "norma:sintetica", Version: 1, HuellaSHA256: h}},
		Anexos:                []domain.AnexoDocumentalInformeJuridico{{DocumentoRef: "documento:sintetico", VersionDocumento: 1, HuellaSHA256: h}},
	})
	if err != nil {
		t.Fatal("borrador fixture:", err)
	}
	ai := a(string(domain.AccionEmitirInformeJuridico), domain.FaseInformeJuridico, 4*time.Minute)
	ai.UnidadRef = e.Asignacion.UnidadRef
	ai.DocumentosRef = []string{"documento:informe:sintetico"}
	e, err = e.RegistrarInformeJuridico(4, domain.InformeJuridicoEmitido{
		Borrador: borrador.Estado(), InformeRef: "informe:sintetico", DocumentoRef: ai.DocumentosRef[0], VersionDocumento: 1, HuellaDocumentoSHA256: h, EmitidoEn: ai.RealizadaEn,
	}, ai)
	if err != nil {
		t.Fatal("informe fixture:", err)
	}
	af := a(string(domain.AccionRegistrarFiscalizacion), domain.FaseFiscalizacion, 5*time.Minute)
	af.DocumentosRef = ai.DocumentosRef
	e, err = e.RegistrarFiscalizacion(5, domain.DatosRegistrarFiscalizacion{
		FiscalizacionRef: "fiscalizacion:sintetica", Resultado: domain.FiscalizacionFavorable, UnidadFiscalizadoraRef: af.UnidadRef, FiscalizadaEn: af.RealizadaEn,
	}, af)
	if err != nil {
		t.Fatal("fiscalizacion fixture:", err)
	}
	return ports.ExpedienteParaSeleccion{Fiscalizado: e, VersionActual: 6}
}
func puenteBolsaPrueba(t *testing.T) (*puenteBolsaLlamamientoDesarrollo, context.Context, preparacionLlamamientoDesarrollo, *relojPuenteBolsaPrueba) {
	t.Helper()
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	reloj := &relojPuenteBolsaPrueba{instante: time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)}
	p := &puenteBolsaLlamamientoDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: soporte}, reloj: reloj,
		repositorio: &repositorioPuenteBolsaFallido{}, autorizador: autorizadorPuenteBolsaDenegado{}}
	if err := p.configurarFirmas(bytes.Repeat([]byte{0x5a}, 32)); err != nil {
		t.Fatal(err)
	}
	d, err := prepararReferenciasLlamamientoDesarrollo(expedientePuenteBolsaPrueba(t), "11111111-1111-4111-8111-111111111111")
	if err != nil {
		t.Fatal(err)
	}
	ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, httpinterno.RutaSeleccionLlamamiento)
	return p, context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{}, d), d, reloj
}

func TestPuenteBolsaLlamamientoDesarrolloReferenciasYFuenteEstables(t *testing.T) {
	p, ctx, d, reloj := puenteBolsaPrueba(t)
	f1, doc1, err := p.fuente(d)
	if err != nil {
		t.Fatal(err)
	}
	b1, firma1, err := f1.ExportarFuenteFirmada(ctx, d.necesidad)
	if err != nil {
		t.Fatal(err)
	}
	reloj.instante = reloj.instante.Add(24 * time.Hour)
	p2 := &puenteBolsaLlamamientoDesarrollo{reloj: reloj}
	if err := p2.configurarFirmas(bytes.Repeat([]byte{0x5a}, 32)); err != nil {
		t.Fatal(err)
	}
	f2, doc2, err := p2.fuente(d)
	if err != nil {
		t.Fatal(err)
	}
	b2, firma2, err := f2.ExportarFuenteFirmada(ctx, d.necesidad)
	if err != nil || !bytes.Equal(b1, b2) || !bytes.Equal(firma1, firma2) {
		t.Fatal("reinicio cambió la fuente firmada")
	}
	if doc1.Datos.Necesidad.UnidadRef != d.unidad || doc2.Datos.Necesidad.CategoriaRef != d.categoria ||
		doc2.Datos.Necesidad.CreadaEn != d.expediente.Fiscalizado.Fiscalizacion.FiscalizadaEn ||
		len(doc2.Datos.Entradas) != 3 {
		t.Fatal("fuente desligada del expediente")
	}
	otraClave, err := prepararReferenciasLlamamientoDesarrollo(d.expediente, "22222222-2222-4222-8222-222222222222")
	if err != nil || otraClave.necesidad != d.necesidad || otraClave.operacionOrden == d.operacionOrden ||
		otraClave.operacionPropuesta == d.operacionPropuesta {
		t.Fatal("derivaciones paralelas o colisión")
	}
	d.expediente.Fiscalizado.Analisis.CategoriaRef = "categoria:adulterada"
	if otraClave.expediente.Fiscalizado.Analisis.CategoriaRef == "categoria:adulterada" {
		t.Fatal("no se clonó expediente")
	}
	if _, err := prepararReferenciasLlamamientoDesarrollo(d.expediente, "clave-invalida"); err == nil {
		t.Fatal("clave inválida admitida")
	}
	reloj.instante = time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, _, err := p2.fuente(otraClave); err == nil {
		t.Fatal("fuente caducada admitida")
	}
}

func TestPuenteBolsaLlamamientoDesarrolloConsultaCanonicaMACYFalloSinRecibo(t *testing.T) {
	p, ctx, d, reloj := puenteBolsaPrueba(t)
	// El reloj avanza entre cada lectura: detecta comparación con el instante
	// anterior a emitir un contexto nuevo.
	reloj.paso = time.Microsecond
	q, err := p.PrepararConsultaDisponibilidad(ctx, d.clave)
	if err != nil {
		t.Fatal(err)
	}
	r, err := p.ConsultarDisponibilidad(ctx, q)
	if err != nil {
		t.Fatal("consulta:", err)
	}
	if _, _, err := p.Verificador().VerificarDisponibilidad(ctx, q, r, reloj.Ahora()); err != nil {
		t.Fatal("canon MAC:", err)
	}
	alterado := r
	alterado.CantidadDisponible = 1
	if _, _, err := p.Verificador().VerificarDisponibilidad(ctx, q, alterado, reloj.Ahora()); err == nil {
		t.Fatal("MAC no liga cantidad")
	}
	c, err := p.PrepararOrdenCompleto(ctx, d.clave, r)
	if err != nil {
		t.Fatal(err)
	}
	recibo, err := p.PrepararOrden(ctx, c)
	if err == nil || recibo != (ports.ReciboOrdenBolsa{}) {
		t.Fatal("fallo de repositorio fabricó éxito")
	}
	repo := p.repositorio.(*repositorioPuenteBolsaFallido)
	if repo.llamadas != 1 {
		t.Fatal("no se llegó al repositorio real del servicio")
	}
	registro, _ := q.Contexto.Registro()
	reinicio := &puenteBolsaLlamamientoDesarrollo{}
	if err := reinicio.configurarFirmas(bytes.Repeat([]byte{0x5a}, 32)); err != nil {
		t.Fatal(err)
	}
	if _, err := reinicio.AutenticadorContexto().Reautenticar(ctx, registro, reloj.Ahora()); err != nil {
		t.Fatal("reinicio MAC:", err)
	}
	otro := &puenteBolsaLlamamientoDesarrollo{}
	if err := otro.configurarFirmas(bytes.Repeat([]byte{0x6b}, 32)); err != nil {
		t.Fatal(err)
	}
	if _, err := otro.AutenticadorContexto().Reautenticar(ctx, registro, reloj.Ahora()); err == nil {
		t.Fatal("se admitió otra clave")
	}
	ctxCancelado, cancelar := context.WithCancel(ctx)
	cancelar()
	if _, err := p.PrepararConsultaDisponibilidad(ctxCancelado, d.clave); err == nil {
		t.Fatal("cancelación ignorada")
	}
}

func TestPuenteBolsaLlamamientoDesarrolloHistoricoSoloTerminal(t *testing.T) {
	p, ctx, d, _ := puenteBolsaPrueba(t)
	consulta, err := p.PrepararConsultaDisponibilidad(ctx, d.clave)
	if err != nil {
		t.Fatal(err)
	}
	original, _ := consulta.Contexto.Registro()
	d.expediente.VersionActual = 7
	ctx = context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{}, d)
	q, err := p.PrepararConsultaDisponibilidad(ctx, d.clave)
	if err != nil {
		t.Fatal("histórico bloqueó consulta terminal:", err)
	}
	nuevo, _ := q.Contexto.Registro()
	if original.Datos.Autorizacion != nuevo.Datos.Autorizacion || original.Datos.Recurso != nuevo.Datos.Recurso ||
		original.Datos.CorrelacionRef != nuevo.Datos.CorrelacionRef {
		t.Fatal("intención terminal inestable")
	}
	if _, err := p.ConsultarDisponibilidad(ctx, q); err == nil {
		t.Fatal("se inició operación histórica")
	}
	if _, err := p.PrepararOrden(ctx, ports.ComandoPrepararOrdenBolsa{}); err == nil {
		t.Fatal("escritura histórica")
	}
	if _, err := p.SolicitarLlamamiento(ctx, ports.ComandoSolicitarLlamamientoBolsa{}); err == nil {
		t.Fatal("apertura histórica")
	}
	if p.repositorio.(*repositorioPuenteBolsaFallido).llamadas != 0 {
		t.Fatal("se consultó efecto histórico")
	}
	if _, err := p.PrepararConsultaDisponibilidad(context.Background(), d.clave); err == nil {
		t.Fatal("contexto privado omitido")
	}
}

func TestPuenteBolsaLlamamientoDesarrolloRecibosConAgregadoYFechasReales(t *testing.T) {
	p, ctx, d, reloj := puenteBolsaPrueba(t)
	consulta, err := p.PrepararConsultaDisponibilidad(ctx, d.clave)
	if err != nil {
		t.Fatal(err)
	}
	disponible, err := p.ConsultarDisponibilidad(ctx, consulta)
	if err != nil {
		t.Fatal(err)
	}
	orden, err := p.PrepararOrdenCompleto(ctx, d.clave, disponible)
	if err != nil {
		t.Fatal(err)
	}
	cOrden, _ := orden.Contexto.DatosEn(reloj.Ahora())
	fuente, doc, err := p.fuente(d)
	if err != nil {
		t.Fatal(err)
	}
	b, firma, err := fuente.ExportarFuenteFirmada(ctx, d.necesidad)
	if err != nil {
		t.Fatal(err)
	}
	reloj.instante = reloj.instante.Add(time.Second)
	instantanea, err := dominiobolsa.NuevaInstantaneaOrdenBolsa(dominiobolsa.AltaInstantaneaOrdenBolsa{
		InstantaneaRef: "orden:fixture:unitaria", Version: 1, Bolsa: doc.Datos.Bolsa, ReferidaEn: reloj.Ahora(), GeneradaEn: reloj.Ahora(), Entradas: doc.Datos.Entradas,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Recibos de unidad construidos explícitamente desde el dominio. NO
	// acreditan efecto SQL; ejercitan únicamente la traducción y su firma.
	ro := puertosbolsa.ReciboLlamamientoDesarrollo{
		Registro: puertosbolsa.RegistroLlamamientoDesarrollo{
			Esquema: "vec.bolsa.integracion-llamamientos-desarrollo.v1", Tipo: "orden", OperacionRef: d.operacionOrden, NecesidadRef: d.necesidad,
			VersionNecesidad: 6, CategoriaRef: d.categoria, UnidadRef: d.unidad, Fuente: b, FirmaFuente: firma, Instantanea: instantanea,
		}, ReciboRef: "recibo:orden:unitario", AuditoriaRef: "auditoria:orden:unitaria", EventoRef: "evento:orden:unitario", ConfirmadaEn: reloj.Ahora(),
	}
	reciboOrden, err := p.firmarOrdenPersistida(ctx, orden, cOrden, doc, ro)
	if err != nil {
		t.Fatal("firma orden:", err)
	}
	if reciboOrden.OrdenRecuperada {
		t.Fatal("una orden nueva no es una recuperación")
	}
	_, _, err = p.Verificador().VerificarReciboOrden(ctx, orden, reciboOrden, reloj.Ahora())
	if err != nil {
		t.Fatal("verificación orden:", err)
	}
	cl, err := p.PrepararContextoLlamamiento(ctx, d.clave, reciboOrden)
	if err != nil {
		t.Fatal(err)
	}
	comp, evidencia, err := p.Verificador().VerificarReciboOrden(ctx, orden, reciboOrden, reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	artefacto, err := ports.NuevoArtefactoProbatorioOrdenBolsa(orden, reciboOrden, evidencia, comp)
	if err != nil {
		t.Fatal(err)
	}
	serializado, err := json.Marshal(artefacto)
	if err != nil || len(serializado) == 0 {
		t.Fatal("artefacto:", err)
	}
	comando, err := ports.NuevoComandoSolicitarLlamamientoBolsa(ports.PreparacionComandoSolicitarLlamamientoBolsa{
		Contexto: cl, ComandoOrden: orden, ReciboOrden: reciboOrden, ComprobanteOrden: comp, MaximaPosicionEvaluable: 3,
	}, reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	cLlamamiento, _ := cl.DatosEn(reloj.Ahora())
	evaluaciones := []dominiobolsa.EvaluacionParticipacionLlamamiento{}
	for _, entrada := range doc.Datos.Entradas[:2] {
		ev, err := fuente.EvaluarParticipacion(ctx, puertosbolsa.SolicitudEvaluarParticipacionLlamamiento{
			Necesidad: doc.Datos.Necesidad, InstantaneaRef: instantanea.InstantaneaRef, VersionInstantanea: instantanea.Version,
			HuellaInstantaneaSHA256: instantanea.HuellaContenidoSHA256, InstanteReferencia: instantanea.ReferidaEn,
			InstantaneaGeneradaEn: instantanea.GeneradaEn, Politica: doc.Datos.Politica, Entrada: entrada, EvaluadaEn: reloj.Ahora(),
		})
		if err != nil {
			t.Fatal(err)
		}
		evaluaciones = append(evaluaciones, ev)
	}
	propuesta, err := dominiobolsa.ProponerPrimerLlamamiento(dominiobolsa.OrdenProponerPrimerLlamamiento{
		PropuestaRef: "propuesta:fixture:unitaria", Bolsa: doc.Datos.Bolsa, Necesidad: doc.Datos.Necesidad,
		Instantanea: instantanea, Politica: doc.Datos.Politica, Evaluaciones: evaluaciones, GeneradaEn: reloj.Ahora(),
	})
	if err != nil {
		t.Fatal(err)
	}
	abierto, err := dominiobolsa.NuevoLlamamientoAbierto(dominiobolsa.DatosLlamamientoAbierto{
		LlamamientoRef: "llamamiento:fixture:unitario", BolsaRef: propuesta.BolsaRef, NecesidadRef: propuesta.NecesidadRef, PropuestaRef: propuesta.PropuestaRef, Version: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	rp := ro
	rp.Registro.Tipo = "propuesta"
	rp.Registro.OperacionRef = d.operacionPropuesta
	rp.Registro.OrdenOperacionRef = d.operacionOrden
	rp.Registro.Propuesta = &propuesta
	llamamiento := abierto.Datos()
	rp.Registro.Llamamiento = &llamamiento
	rp.Registro.EstadoLlamamiento = abierto.Estado()
	rp.ReciboRef = "recibo:llamamiento:unitario"
	rp.AuditoriaRef = "auditoria:llamamiento:unitaria"
	rp.EventoRef = "evento:llamamiento:unitario"
	recibo, err := p.firmarLlamamientoPersistido(ctx, comando, cLlamamiento, doc, rp)
	if err != nil {
		t.Fatal("firma llamamiento:", err)
	}
	if recibo.LlamamientoRef != llamamiento.LlamamientoRef || recibo.OrdenSeleccionado != 2 ||
		recibo.ConfirmadaEn != rp.ConfirmadaEn || !recibo.PropuestaGenerada {
		t.Fatal("recibo inventó agregado o fecha")
	}
	if _, _, err := p.Verificador().VerificarReciboLlamamiento(ctx, comando, recibo, reloj.Ahora()); err != nil {
		t.Fatal("verificación llamamiento:", err)
	}
	publico, _ := json.Marshal(recibo)
	if bytes.Contains(publico, []byte(propuesta.SujetoSeleccionadoRef)) || bytes.Contains(publico, []byte(propuesta.ParticipacionSeleccionadaRef)) {
		t.Fatal("filtró referencia privada")
	}
	sinAgregado := rp
	sinAgregado.Registro.Llamamiento = nil
	if r, err := p.firmarLlamamientoPersistido(ctx, comando, cLlamamiento, doc, sinAgregado); err == nil || r != (ports.ReciboSolicitudLlamamientoBolsa{}) {
		t.Fatal("firmó sin agregado abierto")
	}
	rp.ConfirmadaEn = cLlamamiento.SolicitadaEn.Add(-time.Microsecond)
	if r, err := p.firmarLlamamientoPersistido(ctx, comando, cLlamamiento, doc, rp); err == nil || r != (ports.ReciboSolicitudLlamamientoBolsa{}) {
		t.Fatal("replay retocó fecha antigua")
	}
	// Atestación de la misma orden tras una petición posterior. La fixture no
	// simula consumo SQL: comprueba la traducción que sigue al servicio real.
	reloj.instante = reloj.instante.Add(time.Minute)
	ordenNueva := orden
	ordenNueva.Contexto, err = p.contexto(ctx, d, d.operacionOrden, puertosbolsa.AccionPrepararOrdenDesarrollo, orden.Bolsa)
	if err != nil {
		t.Fatal(err)
	}
	contextoNuevo, err := ordenNueva.Contexto.DatosEn(reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	recuperada, err := p.firmarOrdenPersistida(ctx, ordenNueva, contextoNuevo, doc, ro)
	if err != nil || !recuperada.OrdenRecuperada || recuperada.ConfirmadaEn != ro.ConfirmadaEn ||
		recuperada.ReciboRef != ro.ReciboRef || recuperada.Orden != reciboOrden.Orden {
		t.Fatal("recuperar cambió el efecto o rechazó su fecha original:", err)
	}
	if _, _, err := p.Verificador().VerificarReciboOrden(ctx, ordenNueva, recuperada, reloj.Ahora()); err != nil {
		t.Fatal("atestación fresca:", err)
	}
	if _, _, err := p.Verificador().VerificarReciboOrden(ctx, ordenNueva, reciboOrden, reloj.Ahora()); err == nil {
		t.Fatal("la firma antigua se aceptó para la petición nueva")
	}
}

func TestPuenteBolsaLlamamientoDesarrolloConstructorCerrado(t *testing.T) {
	if p, err := nuevoPuenteBolsaLlamamientoDesarrollo(nil, nil, nil, nil, nil); err == nil || p != nil {
		t.Fatal("constructor incompleto")
	}
	p := &puenteBolsaLlamamientoDesarrollo{}
	if err := p.configurarFirmas([]byte("corta")); err == nil {
		t.Fatal("clave corta aceptada")
	}
}

package bootstrap

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	appbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
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

// Dobles exclusivos de unidad. No acreditan una evaluación de plazo, firmas
// V3 ni persistencia SQL; no se conectan a la composición de desarrollo.
type autorizadorAceptacionPuentePrueba struct {
	t        *testing.T
	puente   *puenteBolsaLlamamientoDesarrollo
	llamadas int
	denegar  bool
	accion   string
}

func (a *autorizadorAceptacionPuentePrueba) AutorizarOperacion(_ context.Context, accion string, recurso dominiovec.RecursoAutorizable) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.llamadas++
	a.accion = accion
	if a.denegar {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrAutorizacionDenegada
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		a.t.Fatal(err)
	}
	h := strings.Repeat("a", 64)
	ahora := a.puente.reloj.Ahora().UTC().Truncate(time.Microsecond)
	resumen, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:unidad", h, h, "contexto:unidad", h,
		accion, recurso.Referencia, huella, puertosbolsa.AudienciaIntegracionLlamamientoDesarrollo, ahora, ahora.Add(5*time.Second))
	if err != nil {
		a.t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(a.puente.privadaFuente.Public())
	if err != nil {
		a.t.Fatal(err)
	}
	material, err := puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3([]byte(strings.Repeat("x", 512)), resumen,
		[]byte("{}"), []byte("{}"), []byte("{}"), 1, 1, []byte("unidad"), []byte("unidad"), []byte("unidad"), spki)
	if err != nil {
		a.t.Fatal(err)
	}
	return material, nil
}

type repositorioAceptacionPuentePrueba struct {
	reloj     *relojPuenteBolsaPrueba
	filas     map[string]puertosbolsa.ReciboLlamamientoDesarrollo
	busquedas []string
	guardados int
	fallar    bool
}

func (r *repositorioAceptacionPuentePrueba) BuscarOperacion(_ context.Context, ref string) (puertosbolsa.RegistroLlamamientoDesarrollo, bool, error) {
	r.busquedas = append(r.busquedas, ref)
	fila, existe := r.filas[ref]
	return fila.Registro, existe, nil
}

func (r *repositorioAceptacionPuentePrueba) Guardar(_ context.Context, registro puertosbolsa.RegistroLlamamientoDesarrollo, _ puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (puertosbolsa.ReciboLlamamientoDesarrollo, error) {
	r.guardados++
	if r.fallar {
		return puertosbolsa.ReciboLlamamientoDesarrollo{}, ports.ErrIntegracionBolsaNoDisponible
	}
	if fila, existe := r.filas[registro.OperacionRef]; existe {
		return fila, nil
	}
	recibo := puertosbolsa.ReciboLlamamientoDesarrollo{Registro: registro, ReciboRef: "recibo:" + registro.OperacionRef,
		AuditoriaRef: "auditoria:" + registro.OperacionRef, EventoRef: "evento:" + registro.OperacionRef, ConfirmadaEn: r.reloj.Ahora()}
	r.filas[registro.OperacionRef] = recibo
	return recibo, nil
}

func escenarioAceptacionPuentePrueba(t *testing.T) (*puenteBolsaLlamamientoDesarrollo, context.Context, ports.SolicitudResolverLlamamiento,
	ports.ReciboSolicitudLlamamientoBolsa, puertosbolsa.ResolucionLlamamientoDesarrollo, *repositorioAceptacionPuentePrueba, *autorizadorAceptacionPuentePrueba) {
	t.Helper()
	p, ctx, d, reloj := puenteBolsaPrueba(t)
	fuente, doc, err := p.fuente(d)
	if err != nil {
		t.Fatal(err)
	}
	repo := &repositorioAceptacionPuentePrueba{reloj: reloj, filas: map[string]puertosbolsa.ReciboLlamamientoDesarrollo{}}
	a := &autorizadorAceptacionPuentePrueba{t: t, puente: p}
	servicio, err := appbolsa.NuevoServicioIntegracionLlamamientosDesarrollo(fuente, repo, a, reloj)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = servicio.PrepararOrden(ctx, puertosbolsa.PeticionLlamamientoDesarrollo{
		OperacionRef: d.operacionOrden, NecesidadRef: d.necesidad, MaximoPosiciones: 3,
	}); err != nil {
		t.Fatal(err)
	}
	r, err := servicio.SolicitarLlamamiento(ctx, puertosbolsa.PeticionLlamamientoDesarrollo{
		OperacionRef: d.operacionPropuesta, OrdenOperacionRef: d.operacionOrden, NecesidadRef: d.necesidad, MaximoPosiciones: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	n, bolsa, politica := referenciasFuentePuenteLlamamientoDesarrollo(doc)
	canon, err := r.Registro.Canonico()
	if err != nil {
		t.Fatal(err)
	}
	sello, err := p.seleccion.SellarDatos(ctx, []byte("seleccion-sintetica-unidad"))
	if err != nil {
		t.Fatal(err)
	}
	seudonimo, err := ports.NuevoSeudonimoSeleccionBolsa(sello)
	if err != nil {
		t.Fatal(err)
	}
	seleccion := ports.ReciboSolicitudLlamamientoBolsa{
		OperacionRef: d.operacionPropuesta, OrganizacionRef: d.expediente.Fiscalizado.OrganizacionRef,
		ExpedienteRef: d.expediente.Fiscalizado.Referencia, VersionExpediente: 6,
		Necesidad: n, Bolsa: bolsa, Politica: politica,
		Orden:             referenciaVersionadaPuenteLlamamientoDesarrollo(r.Registro.Instantanea.InstantaneaRef, r.Registro.Instantanea.Version, r.Registro.Instantanea.HuellaContenidoSHA256),
		Propuesta:         referenciaVersionadaPuenteLlamamientoDesarrollo(r.Registro.Propuesta.PropuestaRef, 1, r.Registro.Propuesta.HuellaContenidoSHA256),
		Resultado:         referenciaVersionadaPuenteLlamamientoDesarrollo(r.ReciboRef, 1, huellaPuenteLlamamientoDesarrollo(canon)),
		PropuestaGenerada: true, LlamamientoRef: r.Registro.Llamamiento.LlamamientoRef,
		SeleccionRef: seudonimo, OrdenSeleccionado: uint32(r.Registro.Propuesta.OrdenSeleccionado),
		ReciboRef: r.ReciboRef, AuditoriaRef: r.AuditoriaRef, EventoRef: r.EventoRef, ConfirmadaEn: r.ConfirmadaEn,
	}
	s := ports.SolicitudResolverLlamamiento{ClaveIdempotencia: "22222222-2222-4222-8222-222222222222",
		OrganizacionRef: seleccion.OrganizacionRef, ExpedienteRef: seleccion.ExpedienteRef, LlamamientoRef: seleccion.LlamamientoRef,
		ComunicacionRef: "comunicacion:sintetica", VersionEsperada: 2, Respuesta: ports.RespuestaLlamamientoAceptada, PruebaRespuestaRef: "justificante:unidad",
	}
	resolucion := puertosbolsa.ResolucionLlamamientoDesarrollo{AperturaOperacionRef: seleccion.OperacionRef,
		JustificanteRef: s.PruebaRespuestaRef, EvaluacionPlazoRef: "evaluacion:unidad", PoliticaRef: "politica:unidad",
		PoliticaVersion: 1, PoliticaSHA256: strings.Repeat("a", 64), VersionEsperada: 1,
	}
	capacidad, _ := p.alta.soporte.capacidadValida(ctx)
	capacidad.ruta = httpinterno.RutaResolucionComunicacionLlamamiento
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	p.repositorio = repo
	a = &autorizadorAceptacionPuentePrueba{t: t, puente: p}
	p.autorizadorAceptacion = a
	return p, ctx, s, seleccion, resolucion, repo, a
}

func TestPuenteBolsaLlamamientoDesarrolloAceptacionReutilizaAperturaYReplay(t *testing.T) {
	for _, caso := range []struct {
		tipo, prefijo, accion string
		estado                dominiobolsa.EstadoLlamamiento
		resolver              func(*puenteBolsaLlamamientoDesarrollo, context.Context, ports.SolicitudResolverLlamamiento, ports.ReciboSolicitudLlamamientoBolsa, puertosbolsa.ResolucionLlamamientoDesarrollo) (puertosbolsa.ReciboLlamamientoDesarrollo, error)
	}{
		{"aceptacion_rrhh", "operacion-aceptacion-rrhh", puertosbolsa.AccionAceptarLlamamientoRRHHDesarrollo, dominiobolsa.EstadoLlamamientoAceptado, (*puenteBolsaLlamamientoDesarrollo).AceptarRespuestaRRHH},
		{"renuncia_rrhh", "operacion-renuncia-rrhh", puertosbolsa.AccionRenunciarLlamamientoRRHHDesarrollo, dominiobolsa.EstadoLlamamientoRenunciado, (*puenteBolsaLlamamientoDesarrollo).RenunciarRespuestaRRHH},
	} {
		t.Run(caso.tipo, func(t *testing.T) {
			p, ctx, s, seleccion, resolucion, repo, a := escenarioAceptacionPuentePrueba(t)
			if caso.tipo == "renuncia_rrhh" {
				s.Respuesta = ports.RespuestaLlamamientoRenunciada
				p.autorizadorRenuncia, p.autorizadorAceptacion = a, autorizadorPuenteBolsaDenegado{}
			}
			canonOriginal, _ := repo.filas[seleccion.OperacionRef].Registro.Canonico()
			r, err := caso.resolver(p, ctx, s, seleccion, resolucion)
			if err != nil {
				t.Fatal(err)
			}
			esperada := referenciaPuenteLlamamientoDesarrollo(caso.prefijo, s.OrganizacionRef, s.ExpedienteRef, seleccion.OperacionRef, s.ClaveIdempotencia)
			if r.Registro.Tipo != caso.tipo || r.Registro.OperacionRef != esperada || r.Registro.Llamamiento.Version != 2 ||
				r.Registro.EstadoLlamamiento != caso.estado || r.Registro.Llamamiento.LlamamientoRef != seleccion.LlamamientoRef ||
				r.Registro.Resolucion.AperturaOperacionRef != seleccion.OperacionRef || a.llamadas != 1 ||
				a.accion != caso.accion || len(repo.filas) != 3 {
				t.Fatal("aceptación creó otra apertura o utilizó otro permiso")
			}
			p2 := *p
			repo.reloj.instante = repo.reloj.instante.Add(time.Minute)
			replay, err := caso.resolver(&p2, ctx, s, seleccion, resolucion)
			if err != nil || replay.ReciboRef != r.ReciboRef || replay.ConfirmadaEn != r.ConfirmadaEn ||
				replay.Registro.Resolucion.ResueltaEn != r.Registro.Resolucion.ResueltaEn || a.llamadas != 2 || len(repo.filas) != 3 || repo.guardados != 4 {
				t.Fatal("recuperación sin autorización nueva o con fecha/efectos distintos", err)
			}
			canonActual, _ := repo.filas[seleccion.OperacionRef].Registro.Canonico()
			if !bytes.Equal(canonOriginal, canonActual) || !resolucion.ResueltaEn.IsZero() {
				t.Fatal("mutó apertura o resolución de entrada")
			}
			resolucion.PoliticaVersion++
			if _, err = caso.resolver(p, ctx, s, seleccion, resolucion); err == nil || repo.guardados != 4 || a.llamadas != 2 {
				t.Fatal("replay divergente alcanzó autorización o persistencia")
			}
		})
	}
}

func TestPuenteBolsaLlamamientoDesarrolloRenunciaExigePermisoYRespuestaPropios(t *testing.T) {
	p, ctx, s, seleccion, resolucion, repo, a := escenarioAceptacionPuentePrueba(t)
	s.Respuesta = ports.RespuestaLlamamientoRenunciada
	for _, ausente := range []puertosbolsa.AutorizadorLlamamientoDesarrollo{nil, (*autorizadorAceptacionPuentePrueba)(nil)} {
		p.autorizadorRenuncia = ausente
		lecturas := len(repo.busquedas)
		if r, err := p.RenunciarRespuestaRRHH(ctx, s, seleccion, resolucion); !errors.Is(err, ports.ErrAutorizacionDenegada) || r.ReciboRef != "" || len(repo.busquedas) != lecturas || a.llamadas != 0 {
			t.Fatal("renuncia sin permiso propio consultó o reutilizó aceptación", err)
		}
	}
	p.autorizadorRenuncia = a
	s.Respuesta = ports.RespuestaLlamamientoAceptada
	if _, err := p.RenunciarRespuestaRRHH(ctx, s, seleccion, resolucion); !errors.Is(err, ports.ErrPeticionIntegracionBolsaInvalida) || a.llamadas != 0 {
		t.Fatal("método de renuncia admitió aceptación", err)
	}
	s.Respuesta = ports.RespuestaLlamamientoRenunciada
	if _, err := p.AceptarRespuestaRRHH(ctx, s, seleccion, resolucion); !errors.Is(err, ports.ErrPeticionIntegracionBolsaInvalida) || a.llamadas != 0 {
		t.Fatal("método de aceptación admitió renuncia", err)
	}
	seleccion.Orden.Version++
	if _, err := p.RenunciarRespuestaRRHH(ctx, s, seleccion, resolucion); err == nil || a.llamadas != 0 || repo.guardados != 2 {
		t.Fatal("renuncia desligada alcanzó permiso o efecto")
	}
	seleccion.Orden.Version--
	a.denegar = true
	if r, err := p.RenunciarRespuestaRRHH(ctx, s, seleccion, resolucion); !errors.Is(err, ports.ErrAutorizacionDenegada) || r.ReciboRef != "" || repo.guardados != 2 {
		t.Fatal("renuncia denegada produjo efecto", err)
	}
	a.denegar, repo.fallar = false, true
	if r, err := p.RenunciarRespuestaRRHH(ctx, s, seleccion, resolucion); err == nil || r.ReciboRef != "" || len(repo.filas) != 2 {
		t.Fatal("renuncia fallida produjo recibo", err)
	}
}

func TestPuenteBolsaLlamamientoDesarrolloAceptacionCotejaAntesDeAutorizar(t *testing.T) {
	p, ctx, s, seleccion, resolucion, repo, a := escenarioAceptacionPuentePrueba(t)
	for nombre, cambiar := range map[string]func(*ports.ReciboSolicitudLlamamientoBolsa){
		"apertura":     func(r *ports.ReciboSolicitudLlamamientoBolsa) { r.OperacionRef += "otra" },
		"organizacion": func(r *ports.ReciboSolicitudLlamamientoBolsa) { r.OrganizacionRef += "otra" },
		"expediente":   func(r *ports.ReciboSolicitudLlamamientoBolsa) { r.ExpedienteRef += "otro" },
		"llamamiento":  func(r *ports.ReciboSolicitudLlamamientoBolsa) { r.LlamamientoRef += "otro" },
		"necesidad":    func(r *ports.ReciboSolicitudLlamamientoBolsa) { r.Necesidad.Referencia += "otra" },
		"orden":        func(r *ports.ReciboSolicitudLlamamientoBolsa) { r.Orden.Version++ },
		"propuesta":    func(r *ports.ReciboSolicitudLlamamientoBolsa) { r.Propuesta.HuellaSHA256 = strings.Repeat("b", 64) },
		"canon":        func(r *ports.ReciboSolicitudLlamamientoBolsa) { r.Resultado.HuellaSHA256 = strings.Repeat("b", 64) },
	} {
		t.Run(nombre, func(t *testing.T) {
			otra := seleccion
			cambiar(&otra)
			r, err := p.AceptarRespuestaRRHH(ctx, s, otra, resolucion)
			if err == nil || r.ReciboRef != "" || a.llamadas != 0 || repo.guardados != 2 {
				t.Fatal("antecedente desligado alcanzó permiso/efecto", err)
			}
		})
	}
	capacidad, _ := p.alta.soporte.capacidadValida(ctx)
	capacidad.ruta = httpinterno.RutaSeleccionLlamamiento
	otroCtx := context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	if _, err := p.AceptarRespuestaRRHH(otroCtx, s, seleccion, resolucion); !errors.Is(err, ports.ErrAutorizacionDenegada) {
		t.Fatal("ruta selección admitida", err)
	}
	p.autorizadorAceptacion = (*autorizadorAceptacionPuentePrueba)(nil)
	lecturas := len(repo.busquedas)
	if _, err := p.AceptarRespuestaRRHH(ctx, s, seleccion, resolucion); !errors.Is(err, ports.ErrAutorizacionDenegada) || len(repo.busquedas) != lecturas {
		t.Fatal("sin permiso propio hubo lectura", err)
	}
}

func TestPuenteBolsaLlamamientoDesarrolloAceptacionNoFabricaExito(t *testing.T) {
	p, ctx, s, seleccion, resolucion, repo, a := escenarioAceptacionPuentePrueba(t)
	a.denegar = true
	if r, err := p.AceptarRespuestaRRHH(ctx, s, seleccion, resolucion); !errors.Is(err, ports.ErrAutorizacionDenegada) || r.ReciboRef != "" || repo.guardados != 2 {
		t.Fatal(err)
	}
	a.denegar, repo.fallar = false, true
	if r, err := p.AceptarRespuestaRRHH(ctx, s, seleccion, resolucion); err == nil || r.ReciboRef != "" || len(repo.filas) != 2 {
		t.Fatal("fallo de persistencia dio recibo", err)
	}
	ctx, cancelar := context.WithCancel(ctx)
	cancelar()
	if _, err := p.AceptarRespuestaRRHH(ctx, s, seleccion, resolucion); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

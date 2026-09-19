package bootstrap

import (
	"bytes"
	"context"
	"crypto/x509"
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

// Dobles exclusivos de unidad. No acreditan una evaluación de plazo, firmas
// V3 ni persistencia SQL; no se conectan a la composición de desarrollo.
func TestPuenteBolsaLlamamientoDesarrolloResolucionSucesorLigadoYReplay(t *testing.T) {
	p, ctx, l, repo, a := escenarioResolucionSucesorPrueba(t)
	seleccion, c := l.justificante.Seleccion, *l.justificante.Continuacion
	originales := map[string][]byte{}
	for ref, recibo := range repo.filas {
		originales[ref], _ = recibo.Registro.Canonico()
	}
	r := puertosbolsa.ResolucionLlamamientoDesarrollo{AperturaOperacionRef: c.ReciboBolsa.OperacionRef,
		JustificanteRef: l.solicitud.PruebaRespuestaRef, EvaluacionPlazoRef: l.local.EvaluacionPlazoRef,
		PoliticaRef: l.local.Politica.Referencia, PoliticaVersion: l.local.Politica.Version, PoliticaSHA256: l.local.Politica.HuellaSHA256, VersionEsperada: 1}
	for _, alterar := range []func(*aceptacionRevisadaDesarrollo){
		func(x *aceptacionRevisadaDesarrollo) { x.solicitud.ComunicacionRef += "otra" },
		func(x *aceptacionRevisadaDesarrollo) {
			x.justificante.Continuacion.ReciboBolsa.RegistroSHA256 = strings.Repeat("b", 64)
		},
		func(x *aceptacionRevisadaDesarrollo) {
			x.justificante.Continuacion.ReciboBolsa.TerminalOperacionRef += "otro"
		},
	} {
		otra, copia := l, c
		otra.justificante.Continuacion = &copia
		alterar(&otra)
		_, err := p.AceptarRespuestaRRHH(context.WithValue(ctx, claveAceptacionRevisadaDesarrollo{}, otra), l.solicitud, seleccion, r)
		if err == nil || a.llamadas != 0 || len(repo.filas) != len(originales) {
			t.Fatal("aceptó antecedente cruzado", err)
		}
	}
	sinContexto := context.WithValue(ctx, claveAceptacionRevisadaDesarrollo{}, aceptacionRevisadaDesarrollo{})
	if _, err := p.AceptarRespuestaRRHH(sinContexto, l.solicitud, seleccion, r); err == nil || a.llamadas != 0 {
		t.Fatal("sin contexto privado")
	}
	var primero puertosbolsa.ReciboLlamamientoDesarrollo
	for i := 1; i <= 2; i++ {
		actual, err := p.AceptarRespuestaRRHH(ctx, l.solicitud, seleccion, r)
		if err != nil || actual.Registro.OperacionRef != operacionAceptacionManualDesarrollo(l) ||
			actual.Registro.Resolucion.AperturaOperacionRef != c.ReciboBolsa.OperacionRef ||
			actual.Registro.Llamamiento.LlamamientoRef != c.ReciboBolsa.LlamamientoRef || actual.Registro.Llamamiento.Version != 2 ||
			a.llamadas != i || a.accion != puertosbolsa.AccionAceptarLlamamientoRRHHDesarrollo || len(repo.filas) != len(originales)+1 {
			t.Fatal("resolución sucesora no ligada o duplicada", err)
		}
		if i == 1 {
			primero = actual
		} else if actual.ReciboRef != primero.ReciboRef || actual.ConfirmadaEn != primero.ConfirmadaEn {
			t.Fatal("replay cambió recibo")
		}
		repo.reloj.instante = repo.reloj.instante.Add(time.Second)
	}
	for ref, canon := range originales {
		actual, err := repo.filas[ref].Registro.Canonico()
		if err != nil || !bytes.Equal(canon, actual) {
			t.Fatal("antecedente alterado", ref, err)
		}
	}
	if seleccion != l.justificante.Seleccion || *l.justificante.Continuacion != c {
		t.Fatal("selección o continuación transformada")
	}
}

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

func TestPuenteBolsaLlamamientoDesarrolloAceptaReciboSeleccionPosteriorSinCambiarBolsa(t *testing.T) {
	p, ctx, s, seleccion, resolucion, repo, _ := escenarioAceptacionPuentePrueba(t)
	seleccion.VersionExpediente = 7
	r, err := p.AceptarRespuestaRRHH(ctx, s, seleccion, resolucion)
	if err != nil || r.Registro.Llamamiento.Version != 2 ||
		repo.filas[seleccion.OperacionRef].Registro.VersionNecesidad != 6 {
		t.Fatal("la respuesta posterior alteró la apertura o la necesidad de Bolsa", err)
	}
}

func TestPuenteBolsaLlamamientoDesarrolloRechazaVersionSeleccionFueraDeRango(t *testing.T) {
	p, ctx, s, seleccion, resolucion, repo, _ := escenarioAceptacionPuentePrueba(t)
	seleccion.VersionExpediente = ^uint64(0)
	if _, err := p.AceptarRespuestaRRHH(ctx, s, seleccion, resolucion); !errors.Is(err, ports.ErrPeticionIntegracionBolsaInvalida) || len(repo.filas) != 2 {
		t.Fatal("la versión CT fuera de rango alcanzó Bolsa", err)
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

func TestPuenteBolsaLlamamientoDesarrolloVersionTrasSubsanacion(t *testing.T) {
	p, ctx, legado, _ := puenteBolsaPrueba(t)
	e := expedienteTrasSubsanacionPrueba(t, legado.expediente.Fiscalizado)
	nuevo, err := prepararReferenciasLlamamientoDesarrollo(ports.ExpedienteParaSeleccion{Fiscalizado: e, VersionActual: 8}, legado.clave)
	if err != nil {
		t.Fatal(err)
	}
	ctx = context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{}, nuevo)
	q, err := p.PrepararConsultaDisponibilidad(ctx, nuevo.clave)
	if err != nil {
		t.Fatal(err)
	}
	registro, err := q.Contexto.Registro()
	if err != nil || registro.Datos.VersionExpediente != 8 {
		t.Fatal("la intención pierde la versión CT", err)
	}
	_, fuente, err := p.fuente(nuevo)
	if err != nil || fuente.Datos.Necesidad.Version != 6 || nuevo.necesidad != legado.necesidad {
		t.Fatal("se mezclaron versiones CT y Bolsa", err)
	}

	resultado, err := p.ConsultarDisponibilidad(ctx, q)
	if err != nil || resultado.VersionExpediente != 8 {
		t.Fatal("disponibilidad N vigente denegada", err)
	}
	if _, _, err = p.Verificador().VerificarDisponibilidad(ctx, q, resultado, p.reloj.Ahora()); err != nil {
		t.Fatal("respuesta N no autentica", err)
	}
	orden, err := p.PrepararOrdenCompleto(ctx, nuevo.clave, resultado)
	if err != nil {
		t.Fatal(err)
	}
	terminal, err := ports.NuevaConsultaTerminalAutorizada(nuevo.clave, q.Contexto, p.reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	reserva, err := ports.NuevaSolicitudReservaEjecucionSeleccionLlamamiento(terminal, orden, resultado.CantidadDisponible, p.reloj.Ahora())
	if err != nil || !reservaReanudacionLigadaAPreparacionDesarrollo(nuevo, reserva) {
		t.Fatal("reanudación N ligada denegada", err)
	}
	cruzado := context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{}, legado)
	if _, err = p.ConsultarDisponibilidad(cruzado, q); err == nil {
		t.Fatal("contexto N admitido con preparación v6")
	}
	nuevo.expediente.VersionActual = 9
	if reservaReanudacionLigadaAPreparacionDesarrollo(nuevo, reserva) {
		t.Fatal("reanudación con cabeza posterior admitida")
	}
	ctx = context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{}, nuevo)
	if _, err = p.ConsultarDisponibilidad(ctx, q); err == nil {
		t.Fatal("cabeza posterior inició efecto")
	}
	if p.repositorio.(*repositorioPuenteBolsaFallido).llamadas != 0 {
		t.Fatal("se tocó Bolsa tras cambiar cabeza")
	}
}

func expedienteTrasSubsanacionPrueba(t *testing.T, inicial domain.Expediente) domain.Expediente {
	t.Helper()
	e := inicial.Clonar()
	anterior := e.Actuaciones[4]
	e.Actuaciones = e.Actuaciones[:5]
	e.Version, e.FaseActual, e.EstadoActual, e.Fiscalizacion = 5, domain.FaseInformeJuridico, domain.EstadoEnCurso, nil
	e.ActualizadoEn = anterior.RealizadaEn
	instante := anterior.RealizadaEn.Add(time.Minute)
	actuar := func(accion domain.ClaveCatalogo, fase domain.ClaveFase, estado domain.EstadoOperativo, recibo string) domain.DatosActuacion {
		return domain.DatosActuacion{AccionClave: accion, ActorRef: "actor:rrhh:sintetico", UnidadRef: e.Asignacion.UnidadRef, ReciboRef: recibo, RealizadaEn: instante, FaseDestino: fase, EstadoDestino: estado}
	}
	a := actuar(domain.AccionRegistrarFiscalizacion, domain.FaseSubsanacionUnidad, domain.EstadoIncidencia, "recibo:reparo:version")
	a.DocumentosRef = []string{e.InformeJuridico.DocumentoRef}
	a.Observaciones = "Reparo sintético."
	var err error
	e, err = e.RegistrarFiscalizacion(5, domain.DatosRegistrarFiscalizacion{FiscalizacionRef: "fiscalizacion:reparo:version", Resultado: domain.FiscalizacionDesfavorable, UnidadFiscalizadoraRef: a.UnidadRef, FiscalizadaEn: instante, Observaciones: a.Observaciones, RetornoRef: "retorno:version:uno"}, a)
	if err != nil {
		t.Fatal(err)
	}
	instante = instante.Add(time.Minute)
	a = actuar(domain.AccionRegistrarSubsanacionReparo, domain.FaseSubsanacionUnidad, domain.EstadoIncidencia, "recibo:subsanacion:version")
	a.RetornoRef = "retorno:version:uno"
	a.Observaciones = "Corrección sintética."
	e, err = e.RegistrarSubsanacionReparo(6, domain.DatosSubsanacionReparo{RetornoRef: a.RetornoRef, Observaciones: a.Observaciones}, a)
	if err != nil {
		t.Fatal(err)
	}
	instante = instante.Add(time.Minute)
	a = actuar(domain.AccionRegistrarFiscalizacion, domain.FaseFiscalizacion, domain.EstadoEnCurso, "recibo:favorable:version")
	a.RetornoRef = "retorno:version:uno"
	a.DocumentosRef = []string{e.InformeJuridico.DocumentoRef}
	e, err = e.RegistrarFiscalizacion(7, domain.DatosRegistrarFiscalizacion{FiscalizacionRef: "fiscalizacion:nueva:version", Resultado: domain.FiscalizacionFavorable, UnidadFiscalizadoraRef: a.UnidadRef, FiscalizadaEn: instante}, a)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

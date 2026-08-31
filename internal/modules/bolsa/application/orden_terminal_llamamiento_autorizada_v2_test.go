package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	correlacionOrdenTerminalPrueba = "correlacion_0123456789abcdef0123456789abcdef"
	motivoOrdenTerminalPrueba      = "motivo_0123456789abcdef0123456789abcdef"
)

type generadorCorrelacionOrdenTerminalPrueba string

func (g generadorCorrelacionOrdenTerminalPrueba) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return string(g), nil
}

type autorizadorOrdenTerminalPrueba struct {
	ahora    time.Time
	garantia dominiovec.AuthAssurance
	mutar    func(*dominiovec.DecisionAutorizacion, dominiovec.DatosSolicitudAutorizacionLigadaV2)
	observar func(dominiovec.DatosSolicitudAutorizacionLigadaV2)
	despues  func()
	err      error
	llamadas int
	decision dominiovec.DecisionAutorizacion
}

func (a *autorizadorOrdenTerminalPrueba) ExigirSolicitudLigadaV2(
	ctx context.Context,
	solicitud dominiovec.SolicitudAutorizacionLigadaV2,
) (dominiovec.DecisionAutorizacion, error) {
	a.llamadas++
	if a.err != nil {
		return dominiovec.DecisionAutorizacion{}, a.err
	}
	if err := ctx.Err(); err != nil {
		return dominiovec.DecisionAutorizacion{}, err
	}
	datos, err := solicitud.Datos()
	if err != nil {
		return dominiovec.DecisionAutorizacion{}, err
	}
	correlacion, err := datos.Correlacion.ValorCanonico()
	if err != nil {
		return dominiovec.DecisionAutorizacion{}, err
	}
	huellaSolicitud, err := dominiovec.HuellaSHA256SolicitudAutorizacionV2(solicitud)
	if err != nil {
		return dominiovec.DecisionAutorizacion{}, err
	}
	huellaMotivo, err := dominiovec.HuellaSHA256MotivoAutorizacionV2(datos.ReferenciaMotivo)
	if err != nil {
		return dominiovec.DecisionAutorizacion{}, err
	}
	proyeccion := dominiovec.SolicitudAutorizacion{
		Principal: datos.ContextoActor.Principal, PerfilActivoRef: datos.ContextoActor.PerfilActivoRef,
		ContextoActor: datos.ContextoActor, VinculoAutenticacionActor: datos.VinculoAutenticacionActor,
		ReferenciaMotivo: datos.ReferenciaMotivo, Accion: datos.Accion, Recurso: datos.Recurso,
		Finalidad: datos.Finalidad, CorrelacionRef: correlacion, Motivo: datos.ReferenciaMotivo.EntradaClave,
	}
	decision := completarDecisionAutorizacionPrueba(proyeccion, dominiovec.DecisionAutorizacion{
		DecisionRef: fmt.Sprintf("decision:terminal:%03d", a.llamadas), Concedida: true,
		Codigo: "concedida", PrincipalID: datos.ContextoActor.Principal.ID,
		PerfilActivoRef: datos.ContextoActor.PerfilActivoRef, Accion: datos.Accion,
		RecursoRef: datos.Recurso.Referencia, Finalidad: datos.Finalidad,
		CorrelacionRef: correlacion, VinculoAutenticacionActor: datos.VinculoAutenticacionActor,
		GarantiaMinima: a.garantia, EmitidaEn: a.ahora.Add(-time.Second),
		ValidaHasta: a.ahora.Add(time.Minute),
	})
	decision.EsquemaHuellaSolicitud = dominiovec.EsquemaHuellaSolicitudAutorizacionV2
	decision.SolicitudHuellaSHA256 = huellaSolicitud
	decision.EsquemaHuellaMotivo = dominiovec.EsquemaHuellaMotivoAutorizacionV2
	decision.MotivoHuellaSHA256 = huellaMotivo
	if a.observar != nil {
		a.observar(datos)
	}
	if a.mutar != nil {
		a.mutar(&decision, datos)
	}
	a.decision = decision
	if a.despues != nil {
		a.despues()
	}
	return decision, nil
}

type exigidorReplayOrdenTerminalPrueba struct {
	base      aplicacionvec.ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
	evidencia puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
	llamadas  int
}

func (e *exigidorReplayOrdenTerminalPrueba) ExigirEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(
	ctx context.Context,
	actor dominiovec.ContextoActor,
	vinculo dominiovec.VinculoAutenticacionActorV1,
	recurso dominiovec.RecursoAutorizable,
	correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2,
	motivo dominiovec.ReferenciaEntradaCatalogo,
	politica aplicacionvec.PoliticaUsoDecisionAutorizacion,
) (puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2, error) {
	e.llamadas++
	if e.llamadas == 1 {
		var err error
		e.evidencia, err = e.base.ExigirEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(
			ctx, actor, vinculo, recurso, correlacion, motivo, politica,
		)
		return e.evidencia, err
	}
	return e.evidencia, nil
}

type escenarioOrdenTerminalPrueba struct {
	actor       dominiovec.ContextoActor
	vinculo     dominiovec.VinculoAutenticacionActorV1
	llamamiento dominiobolsa.LlamamientoAbierto
	version     uint64
	terminal    dominiobolsa.TerminalLlamamiento
	correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2
	motivo      dominiovec.ReferenciaEntradaCatalogo
	autorizador *autorizadorOrdenTerminalPrueba
	fachada     *aplicacionvec.FachadaUsoDecisionAutorizacionSolicitudLigadaV2
	emisor      *EmisorOrdenTerminalLlamamientoAutorizadaV2
}

func nuevoEscenarioOrdenTerminalPrueba(
	t *testing.T,
	estado dominiobolsa.EstadoLlamamiento,
) escenarioOrdenTerminalPrueba {
	t.Helper()
	superficie := dominiovec.SuperficieAutenticacionExternaPersonalV1
	garantia := dominiovec.AuthAssuranceSubstantial
	if estado == dominiobolsa.EstadoLlamamientoExpirado {
		superficie, garantia = dominiovec.SuperficieAutenticacionInternaCorporativaV1,
			dominiovec.AuthAssuranceHigh
	}
	actor, vinculo := nuevoContextoYVinculoPanelPrueba(
		t, dominiovec.AuthMethodCertificate, garantia, superficie,
	)
	llamamiento, err := dominiobolsa.NuevoLlamamientoAbierto(
		dominiobolsa.DatosLlamamientoAbierto{
			LlamamientoRef: "llamamiento:01K3PRECAP", BolsaRef: "bolsa:01K3PRECAP",
			NecesidadRef: "necesidad:01K3PRECAP", PropuestaRef: "propuesta:01K3PRECAP",
			Version: 7,
		},
	)
	if err != nil {
		t.Fatalf("crear agregado B2: %v", err)
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(), generadorCorrelacionOrdenTerminalPrueba(correlacionOrdenTerminalPrueba),
	)
	if err != nil {
		t.Fatalf("crear correlacion V2: %v", err)
	}
	autorizador := &autorizadorOrdenTerminalPrueba{
		ahora: instantePanelInternoPrueba, garantia: garantia,
	}
	fachada, err := aplicacionvec.NuevaFachadaUsoDecisionAutorizacionSolicitudLigadaV2(
		autorizador, relojPanelInternoPrueba{ahora: instantePanelInternoPrueba},
	)
	if err != nil {
		t.Fatalf("crear fachada V2: %v", err)
	}
	emisor, err := NuevoEmisorOrdenTerminalLlamamientoAutorizadaV2(fachada)
	if err != nil {
		t.Fatalf("crear emisor PRE-CAP: %v", err)
	}
	return escenarioOrdenTerminalPrueba{
		actor: actor, vinculo: vinculo, llamamiento: llamamiento,
		version: llamamiento.Datos().Version,
		terminal: dominiobolsa.TerminalLlamamiento{
			Estado: estado, OperacionRef: "operacion:01K3PRECAP",
		},
		correlacion: correlacion,
		motivo: dominiovec.ReferenciaEntradaCatalogo{
			CatalogoID: "motivos_autorizacion_llamamiento", CatalogoVersion: 3,
			CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: motivoOrdenTerminalPrueba,
		},
		autorizador: autorizador, fachada: fachada, emisor: emisor,
	}
}

func (e escenarioOrdenTerminalPrueba) emitir(
	ctx context.Context,
) (OrdenTerminalLlamamientoAutorizadaV2, error) {
	return e.emisor.Emitir(
		ctx, e.actor, e.vinculo, e.llamamiento, e.version, e.terminal,
		e.correlacion, e.motivo,
	)
}

func TestOrdenTerminalLlamamientoEmiteLasTresPoliticasYRecursoExactos(t *testing.T) {
	casos := []struct {
		estado, accion, finalidad string
	}{
		{"aceptacion", accionAceptarLlamamiento, finalidadAceptarLlamamiento},
		{"renuncia", accionRenunciarLlamamiento, finalidadRenunciarLlamamiento},
		{"expiracion_gobernada", accionExpirarLlamamiento, finalidadExpirarLlamamiento},
	}
	for _, caso := range casos {
		t.Run(caso.estado, func(t *testing.T) {
			escenario := nuevoEscenarioOrdenTerminalPrueba(t, dominiobolsa.EstadoLlamamiento(caso.estado))
			var solicitud dominiovec.DatosSolicitudAutorizacionLigadaV2
			escenario.autorizador.observar = func(datos dominiovec.DatosSolicitudAutorizacionLigadaV2) {
				solicitud = datos
			}
			orden, err := escenario.emitir(context.Background())
			if err != nil || orden.Validar() != nil || escenario.autorizador.llamadas != 1 {
				t.Fatalf("emision nominal: orden=%v llamadas=%d error=%v", orden, escenario.autorizador.llamadas, err)
			}
			recurso := solicitud.Recurso
			esperadosAmbitos := map[string]string{
				"bolsa_ref": "bolsa:01K3PRECAP", "necesidad_ref": "necesidad:01K3PRECAP",
				"propuesta_ref": "propuesta:01K3PRECAP",
			}
			esperadosAtributos := map[string]string{
				"version_esperada": "7", "estado_terminal": caso.estado,
				"operacion_ref": "operacion:01K3PRECAP",
			}
			if solicitud.Accion != caso.accion || solicitud.Finalidad != caso.finalidad ||
				recurso.Referencia != "llamamiento:01K3PRECAP" || recurso.ModuloID != "bolsa" ||
				recurso.Tipo != "llamamiento_abierto" || !reflect.DeepEqual(recurso.Ambitos, esperadosAmbitos) ||
				!reflect.DeepEqual(recurso.Atributos, esperadosAtributos) ||
				len(escenario.autorizador.decision.CamposPermitidos) != 0 ||
				len(escenario.autorizador.decision.Obligaciones) != 0 {
				t.Fatalf("solicitud no literal: accion=%q finalidad=%q recurso=%+v", solicitud.Accion, solicitud.Finalidad, recurso)
			}
			preimagen := fmt.Sprintf(
				`{"ambitos":{"bolsa_ref":"bolsa:01K3PRECAP","necesidad_ref":"necesidad:01K3PRECAP","propuesta_ref":"propuesta:01K3PRECAP"},"atributos":{"estado_terminal":"%s","operacion_ref":"operacion:01K3PRECAP","version_esperada":"7"}}`,
				caso.estado,
			)
			suma := sha256.Sum256([]byte(preimagen))
			huella, err := recurso.HuellaContextoAutorizacionSHA256()
			if err != nil || huella != hex.EncodeToString(suma[:]) ||
				escenario.autorizador.decision.ContextoRecursoHuellaSHA256 != huella {
				t.Fatalf("huella de contexto distinta: %q / %q / %v", huella, hex.EncodeToString(suma[:]), err)
			}
			proyeccion, err := orden.ReacreditarYProyectar()
			agregado, version, terminal, errDatos := proyeccion.Datos()
			if err != nil || errDatos != nil || agregado != escenario.llamamiento ||
				version != escenario.version || terminal != escenario.terminal || agregado.EsTerminal() {
				t.Fatalf("proyeccion B3 incoherente: version=%d terminal=%+v errores=%v/%v", version, terminal, err, errDatos)
			}
		})
	}
}

func TestOrdenTerminalLlamamientoFallaCerradoAntesDelExigidor(t *testing.T) {
	var fachadaNula *aplicacionvec.FachadaUsoDecisionAutorizacionSolicitudLigadaV2
	for nombre, exigidor := range map[string]aplicacionvec.ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{
		"nulo": nil, "nulo_tipado": fachadaNula,
	} {
		t.Run(nombre, func(t *testing.T) {
			if emisor, err := NuevoEmisorOrdenTerminalLlamamientoAutorizadaV2(exigidor); emisor != nil ||
				err != ErrOrdenTerminalLlamamientoInvalida {
				t.Fatalf("dependencia nula aceptada: emisor=%v error=%v", emisor, err)
			}
		})
	}
	base := nuevoEscenarioOrdenTerminalPrueba(t, dominiobolsa.EstadoLlamamientoAceptado)
	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	correlacionRef, _ := base.correlacion.ValorCanonico()
	casos := []struct {
		nombre string
		ctx    context.Context
		mutar  func(*escenarioOrdenTerminalPrueba)
	}{
		{"contexto_nulo", nil, nil}, {"contexto_cancelado", ctxCancelado, nil},
		{"actor_cero", context.Background(), func(e *escenarioOrdenTerminalPrueba) { e.actor = dominiovec.ContextoActor{} }},
		{"vinculo_cero", context.Background(), func(e *escenarioOrdenTerminalPrueba) { e.vinculo = dominiovec.VinculoAutenticacionActorV1{} }},
		{"agregado_cero", context.Background(), func(e *escenarioOrdenTerminalPrueba) { e.llamamiento = dominiobolsa.LlamamientoAbierto{} }},
		{"version_cero", context.Background(), func(e *escenarioOrdenTerminalPrueba) { e.version = 0 }},
		{"version_divergente", context.Background(), func(e *escenarioOrdenTerminalPrueba) { e.version++ }},
		{"terminal_cero", context.Background(), func(e *escenarioOrdenTerminalPrueba) { e.terminal = dominiobolsa.TerminalLlamamiento{} }},
		{"correlacion_cero", context.Background(), func(e *escenarioOrdenTerminalPrueba) {
			e.correlacion = dominiovec.ReferenciaCorrelacionAutorizacionV2{}
		}},
		{"motivo_cero", context.Background(), func(e *escenarioOrdenTerminalPrueba) { e.motivo = dominiovec.ReferenciaEntradaCatalogo{} }},
		{"operacion_es_correlacion", context.Background(), func(e *escenarioOrdenTerminalPrueba) { e.terminal.OperacionRef = correlacionRef }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			escenario := base
			escenario.autorizador = &autorizadorOrdenTerminalPrueba{ahora: base.autorizador.ahora, garantia: base.autorizador.garantia}
			fachada, _ := aplicacionvec.NuevaFachadaUsoDecisionAutorizacionSolicitudLigadaV2(
				escenario.autorizador, relojPanelInternoPrueba{ahora: instantePanelInternoPrueba},
			)
			escenario.emisor, _ = NuevoEmisorOrdenTerminalLlamamientoAutorizadaV2(fachada)
			if caso.mutar != nil {
				caso.mutar(&escenario)
			}
			orden, err := escenario.emitir(caso.ctx)
			if err != ErrOrdenTerminalLlamamientoInvalida || orden.Validar() == nil ||
				escenario.autorizador.llamadas != 0 {
				t.Fatalf("precondicion aceptada: orden=%v llamadas=%d error=%v", orden, escenario.autorizador.llamadas, err)
			}
		})
	}
	var emisorNulo *EmisorOrdenTerminalLlamamientoAutorizadaV2
	base.emisor = emisorNulo
	if orden, err := base.emitir(context.Background()); err != ErrOrdenTerminalLlamamientoInvalida || orden.Validar() == nil {
		t.Fatalf("receptor nulo aceptado: orden=%v error=%v", orden, err)
	}
}

func TestOrdenTerminalLlamamientoRechazaPerfilesCruzados(t *testing.T) {
	casos := []struct {
		nombre     string
		estado     dominiobolsa.EstadoLlamamiento
		superficie dominiovec.SuperficieAutenticacionActorV1
		garantia   dominiovec.AuthAssurance
	}{
		{"aceptacion_interna", dominiobolsa.EstadoLlamamientoAceptado, dominiovec.SuperficieAutenticacionInternaCorporativaV1, dominiovec.AuthAssuranceHigh},
		{"renuncia_interna", dominiobolsa.EstadoLlamamientoRenunciado, dominiovec.SuperficieAutenticacionInternaCorporativaV1, dominiovec.AuthAssuranceHigh},
		{"expiracion_externa", dominiobolsa.EstadoLlamamientoExpirado, dominiovec.SuperficieAutenticacionExternaPersonalV1, dominiovec.AuthAssuranceHigh},
		{"expiracion_sustancial", dominiobolsa.EstadoLlamamientoExpirado, dominiovec.SuperficieAutenticacionInternaCorporativaV1, dominiovec.AuthAssuranceSubstantial},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			escenario := nuevoEscenarioOrdenTerminalPrueba(t, caso.estado)
			escenario.actor, escenario.vinculo = nuevoContextoYVinculoPanelPrueba(
				t, dominiovec.AuthMethodCertificate, caso.garantia, caso.superficie,
			)
			orden, err := escenario.emitir(context.Background())
			if err != ErrOrdenTerminalLlamamientoInvalida || orden.Validar() == nil ||
				escenario.autorizador.llamadas != 0 {
				t.Fatalf("perfil cruzado aceptado: orden=%v llamadas=%d error=%v", orden, escenario.autorizador.llamadas, err)
			}
		})
	}
}

func TestOrdenTerminalLlamamientoRechazaTodaDivergenciaDeDecisionV2(t *testing.T) {
	huella := strings.Repeat("8", 64)
	correlacionAlternativa := "correlacion_fedcba9876543210fedcba9876543210"
	mutarContexto := func(clave, valor string, ambito bool) func(*dominiovec.DecisionAutorizacion, dominiovec.DatosSolicitudAutorizacionLigadaV2) {
		return func(decision *dominiovec.DecisionAutorizacion, datos dominiovec.DatosSolicitudAutorizacionLigadaV2) {
			recurso := datos.Recurso
			recurso.Ambitos = clonarMapaPrueba(recurso.Ambitos)
			recurso.Atributos = clonarMapaPrueba(recurso.Atributos)
			if ambito {
				recurso.Ambitos[clave] = valor
			} else {
				recurso.Atributos[clave] = valor
			}
			decision.ContextoRecursoHuellaSHA256, _ = recurso.HuellaContextoAutorizacionSHA256()
		}
	}
	casos := []struct {
		nombre string
		mutar  func(*dominiovec.DecisionAutorizacion, dominiovec.DatosSolicitudAutorizacionLigadaV2)
	}{
		{"denegada", func(d *dominiovec.DecisionAutorizacion, _ dominiovec.DatosSolicitudAutorizacionLigadaV2) {
			d.Concedida, d.Codigo = false, "denegada"
		}},
		{"caducada", func(d *dominiovec.DecisionAutorizacion, _ dominiovec.DatosSolicitudAutorizacionLigadaV2) {
			d.ValidaHasta = instantePanelInternoPrueba
		}},
		{"V1", func(d *dominiovec.DecisionAutorizacion, _ dominiovec.DatosSolicitudAutorizacionLigadaV2) {
			d.EsquemaHuellaSolicitud, d.SolicitudHuellaSHA256, d.EsquemaHuellaMotivo, d.MotivoHuellaSHA256 = "", "", "", ""
		}},
		{"referencia", func(d *dominiovec.DecisionAutorizacion, _ dominiovec.DatosSolicitudAutorizacionLigadaV2) {
			d.RecursoRef = "llamamiento:01K3OTRO"
		}},
		{"modulo", func(d *dominiovec.DecisionAutorizacion, _ dominiovec.DatosSolicitudAutorizacionLigadaV2) {
			d.ModuloID = "personal"
		}},
		{"tipo", func(d *dominiovec.DecisionAutorizacion, _ dominiovec.DatosSolicitudAutorizacionLigadaV2) {
			d.TipoRecurso = "llamamiento"
		}},
		{"ambito_bolsa", mutarContexto("bolsa_ref", "bolsa:01K3OTRA", true)},
		{"ambito_necesidad", mutarContexto("necesidad_ref", "necesidad:01K3OTRA", true)},
		{"ambito_propuesta", mutarContexto("propuesta_ref", "propuesta:01K3OTRA", true)},
		{"atributo_version", mutarContexto("version_esperada", "8", false)},
		{"atributo_terminal", mutarContexto("estado_terminal", "renuncia", false)},
		{"atributo_operacion", mutarContexto("operacion_ref", "operacion:01K3OTRA", false)},
		{"accion", func(d *dominiovec.DecisionAutorizacion, _ dominiovec.DatosSolicitudAutorizacionLigadaV2) {
			d.Accion = accionRenunciarLlamamiento
		}},
		{"finalidad", func(d *dominiovec.DecisionAutorizacion, _ dominiovec.DatosSolicitudAutorizacionLigadaV2) {
			d.Finalidad = finalidadRenunciarLlamamiento
		}},
		{"perfil", func(d *dominiovec.DecisionAutorizacion, _ dominiovec.DatosSolicitudAutorizacionLigadaV2) {
			d.PerfilActivoRef = "prf_fedcba9876543210fedcba"
		}},
		{"huella_contexto", func(d *dominiovec.DecisionAutorizacion, _ dominiovec.DatosSolicitudAutorizacionLigadaV2) {
			d.ContextoRecursoHuellaSHA256 = huella
		}},
		{"huella_solicitud", func(d *dominiovec.DecisionAutorizacion, _ dominiovec.DatosSolicitudAutorizacionLigadaV2) {
			d.SolicitudHuellaSHA256 = huella
		}},
		{"correlacion", func(d *dominiovec.DecisionAutorizacion, _ dominiovec.DatosSolicitudAutorizacionLigadaV2) {
			d.CorrelacionRef = correlacionAlternativa
		}},
		{"huella_motivo", func(d *dominiovec.DecisionAutorizacion, _ dominiovec.DatosSolicitudAutorizacionLigadaV2) {
			d.MotivoHuellaSHA256 = huella
		}},
		{"campos", func(d *dominiovec.DecisionAutorizacion, _ dominiovec.DatosSolicitudAutorizacionLigadaV2) {
			d.CamposPermitidos = []string{"estado"}
		}},
		{"obligaciones", func(d *dominiovec.DecisionAutorizacion, _ dominiovec.DatosSolicitudAutorizacionLigadaV2) {
			d.Obligaciones = []string{"segundo_control"}
		}},
		{"comodin", func(d *dominiovec.DecisionAutorizacion, _ dominiovec.DatosSolicitudAutorizacionLigadaV2) {
			d.PoliticasRefs = []string{"politica:*"}
			d.PoliticasHuellasSHA256 = map[string]string{"politica:*": huella}
		}},
		{"vinculo", func(d *dominiovec.DecisionAutorizacion, _ dominiovec.DatosSolicitudAutorizacionLigadaV2) {
			d.VinculoAutenticacionActor = dominiovec.VinculoAutenticacionActorV1{}
		}},
		{"garantia", func(d *dominiovec.DecisionAutorizacion, _ dominiovec.DatosSolicitudAutorizacionLigadaV2) {
			d.GarantiaMinima = dominiovec.AuthAssuranceHigh
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			escenario := nuevoEscenarioOrdenTerminalPrueba(t, dominiobolsa.EstadoLlamamientoAceptado)
			escenario.autorizador.mutar = caso.mutar
			orden, err := escenario.emitir(context.Background())
			if err != ErrOrdenTerminalLlamamientoInvalida || orden.Validar() == nil ||
				escenario.autorizador.llamadas != 1 {
				t.Fatalf("decision divergente aceptada: orden=%v llamadas=%d error=%v", orden, escenario.autorizador.llamadas, err)
			}
		})
	}
}

func TestOrdenTerminalLlamamientoReduceErroresYCancelaTrasElPDP(t *testing.T) {
	for _, preparar := range []func(*escenarioOrdenTerminalPrueba){
		func(e *escenarioOrdenTerminalPrueba) {
			e.autorizador.err = errors.New("proveedor secreto operacion:filtrada")
		},
		func(e *escenarioOrdenTerminalPrueba) {
			ctx, cancelar := context.WithCancel(context.Background())
			e.autorizador.despues = cancelar
			orden, err := e.emitir(ctx)
			if err != ErrOrdenTerminalLlamamientoInvalida || orden.Validar() == nil {
				t.Fatalf("cancelacion posterior aceptada: orden=%v error=%v", orden, err)
			}
		},
	} {
		escenario := nuevoEscenarioOrdenTerminalPrueba(t, dominiobolsa.EstadoLlamamientoAceptado)
		preparar(&escenario)
		if escenario.autorizador.err != nil {
			orden, err := escenario.emitir(context.Background())
			if err != ErrOrdenTerminalLlamamientoInvalida || orden.Validar() == nil ||
				strings.Contains(err.Error(), "secreto") {
				t.Fatalf("error privado filtrado: orden=%v error=%v", orden, err)
			}
		}
	}
}

func TestOrdenTerminalLlamamientoReplayCopiasYValoresCorruptos(t *testing.T) {
	escenario := nuevoEscenarioOrdenTerminalPrueba(t, dominiobolsa.EstadoLlamamientoRenunciado)
	replay := &exigidorReplayOrdenTerminalPrueba{base: escenario.fachada}
	escenario.emisor, _ = NuevoEmisorOrdenTerminalLlamamientoAutorizadaV2(replay)
	primera, err := escenario.emitir(context.Background())
	segunda, errReplay := escenario.emitir(context.Background())
	p1, errP1 := primera.ReacreditarYProyectar()
	p2, errP2 := segunda.ReacreditarYProyectar()
	a1, v1, t1, errD1 := p1.Datos()
	a2, v2, t2, errD2 := p2.Datos()
	if err != nil || errReplay != nil || errP1 != nil || errP2 != nil || errD1 != nil || errD2 != nil ||
		a1 != a2 || v1 != v2 || t1 != t2 {
		t.Fatalf("replay exacto divergente: %v %v %v %v %v %v", err, errReplay, errP1, errP2, errD1, errD2)
	}
	escenario.terminal.OperacionRef = "operacion:01K3DIVERGENTE"
	if orden, err := escenario.emitir(context.Background()); err != ErrOrdenTerminalLlamamientoInvalida || orden.Validar() == nil {
		t.Fatalf("replay divergente aceptado: orden=%v error=%v", orden, err)
	}
	entrada := nuevoEscenarioOrdenTerminalPrueba(t, dominiobolsa.EstadoLlamamientoAceptado)
	orden, err := entrada.emitir(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	entrada.terminal.OperacionRef = "operacion:mutada-despues"
	proyeccion, _ := orden.ReacreditarYProyectar()
	_, _, terminal, err := proyeccion.Datos()
	if err != nil || terminal.OperacionRef == entrada.terminal.OperacionRef {
		t.Fatalf("la orden comparte la entrada mutable: terminal=%+v error=%v", terminal, err)
	}
	corrupta := orden
	datosCorruptos := *orden.datos
	datosCorruptos.huellaSolicitud = strings.Repeat("9", 64)
	corrupta.datos = &datosCorruptos
	if corrupta.Validar() != ErrOrdenTerminalLlamamientoInvalida {
		t.Fatal("orden corrupta validada")
	}
	proyeccionCorrupta := proyeccion
	proyeccionCorrupta.versionEsperada++
	if _, _, _, err := proyeccionCorrupta.Datos(); err != ErrOrdenTerminalLlamamientoInvalida {
		t.Fatalf("proyeccion corrupta validada: %v", err)
	}
	var cero OrdenTerminalLlamamientoAutorizadaV2
	var proyeccionCero ProyeccionOrdenTerminalLlamamientoAutorizadaV2
	if cero.Validar() != ErrOrdenTerminalLlamamientoInvalida {
		t.Fatal("orden cero valida")
	}
	if _, err := cero.ReacreditarYProyectar(); err != ErrOrdenTerminalLlamamientoInvalida {
		t.Fatalf("orden cero proyectable: %v", err)
	}
	if _, _, _, err := proyeccionCero.Datos(); err != ErrOrdenTerminalLlamamientoInvalida {
		t.Fatalf("proyeccion cero valida: %v", err)
	}
}

func TestOrdenTerminalLlamamientoBloqueaCodecsYRedactaSiempre(t *testing.T) {
	escenario := nuevoEscenarioOrdenTerminalPrueba(t, dominiobolsa.EstadoLlamamientoExpirado)
	orden, err := escenario.emitir(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	proyeccion, _ := orden.ReacreditarYProyectar()
	corrupta := orden
	corrupta.datos = nil
	valores := []any{orden, proyeccion, OrdenTerminalLlamamientoAutorizadaV2{}, corrupta}
	for indice, valor := range valores {
		if _, err := json.Marshal(valor); !errors.Is(err, ErrSerializacionOrdenTerminalLlamamientoProhibida) {
			t.Fatalf("%d JSON: %v", indice, err)
		}
		if _, err := xml.Marshal(valor); !errors.Is(err, ErrSerializacionOrdenTerminalLlamamientoProhibida) {
			t.Fatalf("%d XML: %v", indice, err)
		}
		var destino bytes.Buffer
		if err := gob.NewEncoder(&destino).Encode(valor); !errors.Is(err, ErrSerializacionOrdenTerminalLlamamientoProhibida) {
			t.Fatalf("%d Gob: %v", indice, err)
		}
		for nombre, codificar := range map[string]func() error{
			"texto": func() error { _, err := valor.(interface{ MarshalText() ([]byte, error) }).MarshalText(); return err },
			"binario": func() error {
				_, err := valor.(interface{ MarshalBinary() ([]byte, error) }).MarshalBinary()
				return err
			},
			"CBOR": func() error { _, err := valor.(interface{ MarshalCBOR() ([]byte, error) }).MarshalCBOR(); return err },
			"YAML": func() error { _, err := valor.(interface{ MarshalYAML() (any, error) }).MarshalYAML(); return err },
		} {
			if err := codificar(); err != ErrSerializacionOrdenTerminalLlamamientoProhibida {
				t.Fatalf("%d %s: %v", indice, nombre, err)
			}
		}
		for _, formato := range []string{"%s", "%q", "%v", "%+v", "%#v", "%x"} {
			if salida := fmt.Sprintf(formato, valor); salida != etiquetaOrdenTerminalLlamamiento {
				t.Fatalf("%d formato %s filtro %q", indice, formato, salida)
			}
		}
		if valor.(fmt.Stringer).String() != etiquetaOrdenTerminalLlamamiento ||
			valor.(fmt.GoStringer).GoString() != etiquetaOrdenTerminalLlamamiento ||
			valor.(slog.LogValuer).LogValue().String() != etiquetaOrdenTerminalLlamamiento {
			t.Fatalf("%d redaccion inconsistente", indice)
		}
	}
	for nombre, destino := range map[string]any{
		"orden":      &OrdenTerminalLlamamientoAutorizadaV2{},
		"proyeccion": &ProyeccionOrdenTerminalLlamamientoAutorizadaV2{},
	} {
		if err := json.Unmarshal([]byte(`{}`), destino); !errors.Is(err, ErrSerializacionOrdenTerminalLlamamientoProhibida) {
			t.Fatalf("%s decode JSON: %v", nombre, err)
		}
		if err := xml.Unmarshal([]byte(`<x/>`), destino); !errors.Is(err, ErrSerializacionOrdenTerminalLlamamientoProhibida) {
			t.Fatalf("%s decode XML: %v", nombre, err)
		}
		if err := destino.(interface{ UnmarshalText([]byte) error }).UnmarshalText(nil); err != ErrSerializacionOrdenTerminalLlamamientoProhibida {
			t.Fatalf("%s decode texto: %v", nombre, err)
		}
		if err := destino.(interface{ UnmarshalBinary([]byte) error }).UnmarshalBinary(nil); err != ErrSerializacionOrdenTerminalLlamamientoProhibida {
			t.Fatalf("%s decode binario: %v", nombre, err)
		}
		if err := destino.(interface{ GobDecode([]byte) error }).GobDecode(nil); err != ErrSerializacionOrdenTerminalLlamamientoProhibida {
			t.Fatalf("%s decode Gob: %v", nombre, err)
		}
		if err := destino.(interface{ UnmarshalCBOR([]byte) error }).UnmarshalCBOR(nil); err != ErrSerializacionOrdenTerminalLlamamientoProhibida {
			t.Fatalf("%s decode CBOR: %v", nombre, err)
		}
		if err := destino.(interface{ UnmarshalYAML(func(any) error) error }).UnmarshalYAML(nil); err != ErrSerializacionOrdenTerminalLlamamientoProhibida {
			t.Fatalf("%s decode YAML: %v", nombre, err)
		}
	}
}

func TestOrdenTerminalLlamamientoSuperficieMinimaSinAtajosB2(t *testing.T) {
	contenido, err := os.ReadFile("orden_terminal_llamamiento_autorizada_v2.go")
	if err != nil {
		t.Fatal(err)
	}
	texto := string(contenido)
	for _, prohibido := range []string{
		"TransicionarATerminal", "time.Now", "NuevaEvidenciaUsoDecisionAutorizacion",
		"SolicitudAutorizacion,", "Autorizador ", "EvidenciaUsoDecisionAutorizacion ",
		"persona", "sucesor", "persistencia", "auditoria", "outbox", "COMMIT",
	} {
		if strings.Contains(texto, prohibido) {
			t.Fatalf("produccion contiene atajo prohibido %q", prohibido)
		}
	}
	tipoOrden := reflect.TypeOf(OrdenTerminalLlamamientoAutorizadaV2{})
	tipoProyeccion := reflect.TypeOf(ProyeccionOrdenTerminalLlamamientoAutorizadaV2{})
	for _, tipo := range []reflect.Type{tipoOrden, tipoProyeccion} {
		for indice := 0; indice < tipo.NumField(); indice++ {
			if tipo.Field(indice).IsExported() {
				t.Fatalf("%s expone campo %s", tipo.Name(), tipo.Field(indice).Name)
			}
		}
	}
	for indice := 0; indice < tipoOrden.NumMethod(); indice++ {
		nombre := strings.ToLower(tipoOrden.Method(indice).Name)
		for _, prohibido := range []string{"actor", "vinculo", "evidencia", "decision", "pdp"} {
			if strings.Contains(nombre, prohibido) {
				t.Fatalf("orden expone getter %q", nombre)
			}
		}
	}
	tipoEmisor := reflect.TypeOf((*EmisorOrdenTerminalLlamamientoAutorizadaV2)(nil))
	metodo, existe := tipoEmisor.MethodByName("Emitir")
	if !existe {
		t.Fatal("falta API Emitir")
	}
	for indice := 1; indice < metodo.Type.NumIn(); indice++ {
		if metodo.Type.In(indice).Kind() == reflect.String {
			t.Fatalf("Emitir acepta selector libre en argumento %d", indice)
		}
	}
}

func clonarMapaPrueba(origen map[string]string) map[string]string {
	clon := make(map[string]string, len(origen))
	for clave, valor := range origen {
		clon[clave] = valor
	}
	return clon
}

var _ puertosvec.AutorizadorSolicitudLigadaV2 = (*autorizadorOrdenTerminalPrueba)(nil)
var _ aplicacionvec.ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2 = (*exigidorReplayOrdenTerminalPrueba)(nil)

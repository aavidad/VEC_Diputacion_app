package gobiernoreglasbaremo

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestPlanCubreTransicionesYLigaActorMotivoYAlcance(t *testing.T) {
	borrador := borradorPrueba(t)
	publicada, aprobacion := publicadaConEvidenciaPrueba(t, borrador)
	activa, dependencias := activaConEvidenciaPrueba(t, publicada)
	sustituida, autoridadSustituir := terminalConEvidenciaPrueba(
		t, activa, reglas.AccionSustituirReglasBaremo,
	)
	retirada, autoridadRetirar := terminalConEvidenciaPrueba(
		t, activa, reglas.AccionRetirarReglasBaremo,
	)
	descartada, autoridadDescartar := terminalConEvidenciaPrueba(
		t, borrador, reglas.AccionDescartarReglasBaremo,
	)
	vinculoPublicar, err := NuevoVinculoEvidenciaPublicacionReglasBaremoV2(aprobacion)
	debeSinError(t, err)
	vinculoActivar, err := NuevoVinculoEvidenciaActivacionReglasBaremoV2(dependencias)
	debeSinError(t, err)
	vinculoSustituir, err := NuevoVinculoEvidenciaTerminalReglasBaremoV2(autoridadSustituir)
	debeSinError(t, err)
	vinculoRetirar, err := NuevoVinculoEvidenciaTerminalReglasBaremoV2(autoridadRetirar)
	debeSinError(t, err)
	vinculoDescartar, err := NuevoVinculoEvidenciaTerminalReglasBaremoV2(autoridadDescartar)
	debeSinError(t, err)

	casBorrador := vinculoPrueba(t, borrador)
	casPublicada := vinculoPrueba(t, publicada)
	casActiva := vinculoPrueba(t, activa)
	type caso struct {
		operacion OperacionGobiernoReglasBaremoV2
		cas       *reglas.VinculoEstadoReglasBaremo
		resultado reglas.VersionGobernadaReglasBaremo
		evidencia *VinculoEvidenciaTransicionReglasBaremoV2
	}
	casos := []caso{
		{OperacionAltaBorrador, nil, borrador, nil},
		{OperacionPublicar, &casBorrador, publicada, &vinculoPublicar},
		{OperacionActivar, &casPublicada, activa, &vinculoActivar},
		{OperacionSustituir, &casActiva, sustituida, &vinculoSustituir},
		{OperacionRetirar, &casActiva, retirada, &vinculoRetirar},
		{OperacionDescartar, &casBorrador, descartada, &vinculoDescartar},
	}
	for _, caso := range casos {
		instante, err := caso.resultado.InstanteUltimaActuacion()
		debeSinError(t, err)
		entrada := datosNuevoPlanPrueba(t, caso.operacion, caso.cas, caso.resultado, caso.evidencia)
		plan, err := NuevoPlanCambioReglasBaremoV2(entrada)
		if err != nil {
			t.Fatalf("operacion %d: %v", caso.operacion, err)
		}
		datos, err := plan.Datos()
		if err != nil || datos.Operacion != caso.operacion ||
			datos.TieneCAS != (caso.cas != nil) ||
			datos.TieneVinculoEvidencia != (caso.evidencia != nil) ||
			datos.PrincipalRef != actorPlanPrueba ||
			!datos.InstanteTransicion.Equal(instante) ||
			!dominiovec.ReferenciaMotivoAutorizacionV2Valida(datos.ReferenciaMotivo) ||
			!componentesExactos(datos.Componentes) {
			t.Fatalf("proyeccion incorrecta: %#v, %v", datos, err)
		}
		canonicoRestaurado, err := datos.VersionResultado.RepresentacionCanonica()
		if err != nil || !bytes.Equal(canonicoRestaurado, datos.VersionCanonica) {
			t.Fatalf("restauracion de dominio divergente: %v", err)
		}
		contrato, err := plan.ContratoAutorizacionV2()
		debeSinError(t, err)
		recurso, err := contrato.Recurso()
		debeSinError(t, err)
		identidad := conjuntoPrueba(t).Identidad()
		if recurso.Ambitos[ambitoConvocatoriaRef] != identidad.ConvocatoriaRef() ||
			recurso.Ambitos[ambitoExpedienteRef] != identidad.ExpedienteRef() {
			t.Fatalf("alcance no derivado de identidad: %#v", recurso.Ambitos)
		}
	}
}

func TestPlanCanonicoComprometeContratoSinInventarRecibo(t *testing.T) {
	primero, segundo := planAltaPrueba(t), planAltaPrueba(t)
	canonico, err := primero.RepresentacionCanonica()
	debeSinError(t, err)
	otro, err := segundo.RepresentacionCanonica()
	if err != nil || !bytes.Equal(canonico, otro) {
		t.Fatalf("plan no determinista: %v", err)
	}
	huella, err := primero.HuellaSHA256()
	suma := sha256.Sum256(canonico)
	if err != nil || huella != hex.EncodeToString(suma[:]) {
		t.Fatalf("huella incorrecta: %q, %v", huella, err)
	}
	var material materialPlanCambioV2
	debeSinError(t, json.Unmarshal(canonico, &material))
	esperados := []string{
		"contenido", "version", "puntero_cas", "vinculo_evidencia",
		"vec", "auditoria", "outbox", "recibo",
	}
	requisitos := []string{
		"alcance_resuelto_servidor",
		"cotejo_evidencia_verificador_confiable",
		"decision_vec_v2_consumible",
		"consumo_atestado_vec_ad2_mismo_commit",
		"commit_serializable_atomico",
		"reloj_autoritativo_frescura_cotejada",
		"recibo_durable_reconciliable",
	}
	motivoEsperado, _ := dominiovec.RepresentacionCanonicaMotivoAutorizacionV2(
		referenciaMotivoVersionPrueba(t, borradorPrueba(t)),
	)
	if material.Esquema != esquemaPlanCambioV2 ||
		material.Operacion != "alta_borrador" ||
		material.InstanteTransicion != "2026-07-17T08:30:00.123456Z" ||
		material.CASEsperado != nil || material.VinculoEvidencia != nil ||
		material.PrincipalRef != actorPlanPrueba ||
		!bytes.Equal(material.MotivoCanonico, motivoEsperado) ||
		!huellaSHA256Valida(material.HuellaMotivoSHA256) ||
		material.Accion != "bolsa.reglas_baremo.borrador.crear" ||
		material.ModuloID != moduloBolsaGobiernoReglas ||
		material.TipoRecurso != tipoRecursoReglasGobernadas ||
		material.PerfilProteccion != perfilProteccionReglas ||
		material.ConvocatoriaRef != referenciaConvocatoriaPlanPrueba ||
		material.ExpedienteRef != referenciaExpedientePlanPrueba ||
		material.Finalidad != finalidadGobiernoReglas ||
		material.RecursoRef != "reglas-baremo:"+material.VinculoResultado.HuellaEstadoSHA256 ||
		!cadenasIguales(material.RequisitosEjecucion, requisitos) ||
		!cadenasIguales(material.Componentes, esperados) ||
		bytes.Contains(canonico, []byte(`"recibo_ref"`)) ||
		bytes.Contains(canonico, []byte(`"efectuar_en"`)) {
		t.Fatalf("material canonico incompleto o deshonesto: %#v", material)
	}
}

func TestPlanRechazaEvidenciaValidaNoIncorporada(t *testing.T) {
	borrador := borradorPrueba(t)
	publicada, aprobacion := publicadaConEvidenciaPrueba(t, borrador)
	cas := vinculoPrueba(t, borrador)
	vinculoCorrecto, err := NuevoVinculoEvidenciaPublicacionReglasBaremoV2(aprobacion)
	debeSinError(t, err)
	otra := reemplazarAtestacionAprobacionPrueba(
		t, aprobacion, referenciaTecnicaPrueba(t, prefijoAtestacionV2, "aprobacion-distinta"),
	)
	vinculoDistinto, err := NuevoVinculoEvidenciaPublicacionReglasBaremoV2(otra)
	debeSinError(t, err)

	base := datosNuevoPlanPrueba(t, OperacionPublicar, &cas, publicada, &vinculoCorrecto)
	_, err = NuevoPlanCambioReglasBaremoV2(base)
	debeSinError(t, err)
	base.VinculoEvidencia = &vinculoDistinto
	if _, err := NuevoPlanCambioReglasBaremoV2(base); !errors.Is(err, ErrPlanCambioInvalido) {
		t.Fatalf("evidencia B aceptada para version A: %v", err)
	}
}

func TestPlanRechazaMutacionDeCadaCompromiso(t *testing.T) {
	borrador := borradorPrueba(t)
	publicada, aprobacion := publicadaConEvidenciaPrueba(t, borrador)
	cas := vinculoPrueba(t, borrador)
	vinculo, err := NuevoVinculoEvidenciaPublicacionReglasBaremoV2(aprobacion)
	debeSinError(t, err)
	base, err := NuevoPlanCambioReglasBaremoV2(
		datosNuevoPlanPrueba(t, OperacionPublicar, &cas, publicada, &vinculo),
	)
	debeSinError(t, err)
	casos := []struct {
		nombre string
		mutar  func(*PlanCambioReglasBaremoV2)
	}{
		{"operacion", func(p *PlanCambioReglasBaremoV2) { p.datos.operacion = OperacionRetirar }},
		{"intencion", func(p *PlanCambioReglasBaremoV2) { p.datos.intencion = IntencionGobiernoReglasBaremoV2{} }},
		{"cas", func(p *PlanCambioReglasBaremoV2) { p.datos.cas = reglas.VinculoEstadoReglasBaremo{} }},
		{"version", func(p *PlanCambioReglasBaremoV2) { p.datos.versionCanonica[0] ^= 1 }},
		{"huella version", func(p *PlanCambioReglasBaremoV2) { p.datos.huellaVersionSHA256 = strings.Repeat("0", 64) }},
		{"vinculo resultado", func(p *PlanCambioReglasBaremoV2) { p.datos.vinculoResultado = reglas.VinculoEstadoReglasBaremo{} }},
		{"evidencia", func(p *PlanCambioReglasBaremoV2) {
			p.datos.vinculoEvidencia = VinculoEvidenciaTransicionReglasBaremoV2{}
		}},
		{"principal", func(p *PlanCambioReglasBaremoV2) { p.datos.principalRef = "per_ffffffffffffffffffffffffffffffff" }},
		{"motivo referencia", func(p *PlanCambioReglasBaremoV2) {
			p.datos.referenciaMotivo.EntradaClave = "motivo_ffffffffffffffffffffffffffffffff"
		}},
		{"motivo canonico", func(p *PlanCambioReglasBaremoV2) { p.datos.motivoCanonico[0] ^= 1 }},
		{"huella motivo", func(p *PlanCambioReglasBaremoV2) { p.datos.huellaMotivoSHA256 = strings.Repeat("f", 64) }},
		{"correlacion", func(p *PlanCambioReglasBaremoV2) {
			p.datos.correlacion = dominiovec.ReferenciaCorrelacionAutorizacionV2{}
		}},
		{"instante", func(p *PlanCambioReglasBaremoV2) {
			p.datos.instanteTransicion = p.datos.instanteTransicion.Add(time.Nanosecond)
		}},
		{"componentes", func(p *PlanCambioReglasBaremoV2) { p.datos.componentes[0] = ComponenteOutbox }},
		{"representacion", func(p *PlanCambioReglasBaremoV2) { p.datos.representacionCanonica[0] ^= 1 }},
		{"representacion sobredimensionada", func(p *PlanCambioReglasBaremoV2) {
			p.datos.representacionCanonica = make([]byte, maximoBytesPlanCambioV2+1)
		}},
		{"huella plan", func(p *PlanCambioReglasBaremoV2) { p.datos.huellaPlanSHA256 = strings.Repeat("f", 64) }},
	}
	for _, caso := range casos {
		corrupto := clonarPlanPrueba(base)
		caso.mutar(&corrupto)
		if _, err := corrupto.RepresentacionCanonica(); !errors.Is(err, ErrPlanCambioInvalido) {
			t.Fatalf("%s: mutacion aceptada: %v", caso.nombre, err)
		}
		if _, err := corrupto.Datos(); !errors.Is(err, ErrPlanCambioInvalido) {
			t.Fatalf("%s: Datos acepto mutacion: %v", caso.nombre, err)
		}
	}
}

func TestPlanRechazaActorMotivoEInstantesNoExactos(t *testing.T) {
	borrador := borradorPrueba(t)
	base := datosNuevoPlanPrueba(t, OperacionAltaBorrador, nil, borrador, nil)
	casos := []struct {
		nombre string
		mutar  func(*DatosNuevoPlanCambioReglasBaremoV2)
	}{
		{"operacion cero", func(d *DatosNuevoPlanCambioReglasBaremoV2) { d.Operacion = 0 }},
		{"consulta", func(d *DatosNuevoPlanCambioReglasBaremoV2) { d.Operacion = OperacionConsultaExacta }},
		{"intencion cero", func(d *DatosNuevoPlanCambioReglasBaremoV2) { d.Intencion = IntencionGobiernoReglasBaremoV2{} }},
		{"actor cero", func(d *DatosNuevoPlanCambioReglasBaremoV2) { d.ContextoActor = dominiovec.ContextoActor{} }},
		{"otro actor valido", func(d *DatosNuevoPlanCambioReglasBaremoV2) {
			d.ContextoActor = contextoActorConPrincipalPrueba(t, "per_ffffffffffffffffffffffffffffffff")
		}},
		{"motivo cero", func(d *DatosNuevoPlanCambioReglasBaremoV2) {
			d.ReferenciaMotivo = dominiovec.ReferenciaEntradaCatalogo{}
		}},
		{"otro motivo valido", func(d *DatosNuevoPlanCambioReglasBaremoV2) {
			d.ReferenciaMotivo.EntradaClave = "motivo_ffffffffffffffffffffffffffffffff"
		}},
		{"correlacion cero", func(d *DatosNuevoPlanCambioReglasBaremoV2) {
			d.Correlacion = dominiovec.ReferenciaCorrelacionAutorizacionV2{}
		}},
		{"instante cero", func(d *DatosNuevoPlanCambioReglasBaremoV2) { d.InstanteTransicion = time.Time{} }},
		{"zona no UTC", func(d *DatosNuevoPlanCambioReglasBaremoV2) {
			d.InstanteTransicion = d.InstanteTransicion.In(time.FixedZone("CET", 3600))
		}},
		{"submicrosegundo", func(d *DatosNuevoPlanCambioReglasBaremoV2) {
			d.InstanteTransicion = d.InstanteTransicion.Add(time.Nanosecond)
		}},
	}
	for _, caso := range casos {
		datos := base
		caso.mutar(&datos)
		if _, err := NuevoPlanCambioReglasBaremoV2(datos); !errors.Is(err, ErrPlanCambioInvalido) {
			t.Fatalf("%s aceptado: %v", caso.nombre, err)
		}
	}

	for _, referencia := range []reglas.ReferenciaVersionada{
		{},
		referenciaPrueba(t, "intencion:libre", 2),
		referenciaPrueba(t, "dni:12345678z", 2),
	} {
		if _, err := NuevaIntencionGobiernoReglasBaremoV2(referencia); !errors.Is(err, ErrPlanCambioInvalido) {
			t.Fatalf("intencion libre aceptada: %#v, %v", referencia, err)
		}
	}
	_, aprobacion := publicadaConEvidenciaPrueba(t, borrador)
	aprobacionConDNI := reemplazarAtestacionAprobacionPrueba(
		t, aprobacion, referenciaPrueba(t, "dni:12345678z", 2),
	)
	if _, err := NuevoVinculoEvidenciaPublicacionReglasBaremoV2(aprobacionConDNI); !errors.Is(err, ErrPlanCambioInvalido) {
		t.Fatalf("referencia personal aceptada como evidencia V2: %v", err)
	}
}

func TestPlanHaceCopiasYNoConservaContextoActor(t *testing.T) {
	plan := planAltaPrueba(t)
	datos, err := plan.Datos()
	debeSinError(t, err)
	canonicoAntes, _ := plan.RepresentacionCanonica()
	datos.VersionCanonica[0] ^= 1
	datos.MotivoCanonico[0] ^= 1
	datos.Componentes[0] = ComponenteOutbox
	canonicoDespues, err := plan.RepresentacionCanonica()
	if err != nil || !bytes.Equal(canonicoAntes, canonicoDespues) {
		t.Fatalf("una proyeccion modifico el plan: %v", err)
	}
	tipoContexto := reflect.TypeOf(dominiovec.ContextoActor{})
	tipoInterno := reflect.TypeOf(datosPlanCambioReglasBaremoV2{})
	for indice := 0; indice < tipoInterno.NumField(); indice++ {
		if tipoInterno.Field(indice).Type == tipoContexto {
			t.Fatal("el plan conserva indebidamente el ContextoActor completo")
		}
	}
}

func TestTiposOpacosNoFiltranEnJSONNiFormato(t *testing.T) {
	plan := planAltaPrueba(t)
	datos, _ := plan.Datos()
	contrato, _ := plan.ContratoAutorizacionV2()
	borrador := borradorPrueba(t)
	conjunto, _ := borrador.Conjunto()
	descriptor, _ := nuevoDescriptorConsultaExactaReglasBaremoV2(
		vinculoPrueba(t, borrador), conjunto.Identidad(),
	)
	consulta, _ := nuevaConsultaExactaReglasBaremoV2(descriptor)
	_, aprobacion := publicadaConEvidenciaPrueba(t, borrador)
	vinculo, err := NuevoVinculoEvidenciaPublicacionReglasBaremoV2(aprobacion)
	debeSinError(t, err)
	entrada := datosNuevoPlanPrueba(t, OperacionAltaBorrador, nil, borrador, nil)
	valores := []any{
		plan, datos, contrato, descriptor, consulta, entrada, entrada.Intencion, vinculo,
	}
	for _, valor := range valores {
		serializado, err := json.Marshal(valor)
		if !errors.Is(err, ErrSerializacionProhibida) || serializado != nil {
			t.Fatalf("JSON permitido para %T: %q, %v", valor, serializado, err)
		}
		texto := fmt.Sprintf("%v %#v", valor, valor)
		if strings.Contains(texto, "intencion:reglas") ||
			strings.Contains(texto, "correlacion_") || strings.Contains(texto, actorPlanPrueba) {
			t.Fatalf("formato revelador para %T: %s", valor, texto)
		}
	}
}

func datosNuevoPlanPrueba(
	t *testing.T,
	operacion OperacionGobiernoReglasBaremoV2,
	cas *reglas.VinculoEstadoReglasBaremo,
	version reglas.VersionGobernadaReglasBaremo,
	vinculo *VinculoEvidenciaTransicionReglasBaremoV2,
) DatosNuevoPlanCambioReglasBaremoV2 {
	t.Helper()
	instante, err := version.InstanteUltimaActuacion()
	debeSinError(t, err)
	return DatosNuevoPlanCambioReglasBaremoV2{
		Operacion:          operacion,
		Intencion:          intencionPrueba(t),
		CASEsperado:        cas,
		VersionResultado:   version,
		VinculoEvidencia:   vinculo,
		ContextoActor:      contextoActorPrueba(t),
		ReferenciaMotivo:   referenciaMotivoVersionPrueba(t, version),
		Correlacion:        correlacionPrueba(t),
		InstanteTransicion: instante,
	}
}

func reemplazarAtestacionAprobacionPrueba(
	t *testing.T,
	original reglas.AtestacionAprobacionFirmadaReglasBaremo,
	atestacion reglas.ReferenciaVersionada,
) reglas.AtestacionAprobacionFirmadaReglasBaremo {
	t.Helper()
	resultado, err := reglas.NuevaAtestacionAprobacionFirmadaReglasBaremo(
		reglas.DatosAtestacionAprobacionFirmadaReglasBaremo{
			Atestacion:    atestacion,
			Vinculo:       original.Vinculo(),
			Firma:         original.Firma(),
			PoliticaFirma: original.PoliticaFirma(),
			Firmantes:     original.Firmantes(),
			FirmadaEn:     original.FirmadaEn(),
			VerificadaEn:  original.VerificadaEn(),
			ValidaHasta:   original.ValidaHasta(),
		},
	)
	debeSinError(t, err)
	return resultado
}

func cadenasIguales(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for indice := range a {
		if a[indice] != b[indice] {
			return false
		}
	}
	return true
}

func clonarPlanPrueba(origen PlanCambioReglasBaremoV2) PlanCambioReglasBaremoV2 {
	datos := *origen.datos
	datos.versionCanonica = append([]byte(nil), origen.datos.versionCanonica...)
	datos.motivoCanonico = append([]byte(nil), origen.datos.motivoCanonico...)
	datos.componentes = append(
		[]ComponenteEscrituraReglasBaremoV2(nil), origen.datos.componentes...,
	)
	datos.representacionCanonica = append(
		[]byte(nil), origen.datos.representacionCanonica...,
	)
	return PlanCambioReglasBaremoV2{datos: &datos}
}

func debeSinError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

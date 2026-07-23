package ports

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestContextoIntegracionBolsaExigeVigenciaHuellaYLimites(t *testing.T) {
	instante := instanteIntegracionBolsaPrueba()
	contexto := contextoIntegracionBolsaPrueba(instante)
	if err := contexto.ValidarEn(instante.Add(time.Minute)); err != nil {
		t.Fatalf("contexto válido rechazado: %v", err)
	}

	pruebas := []struct {
		nombre  string
		cambiar func(*ContextoPeticionIntegracionBolsa)
	}{
		{"operación vacía", func(c *ContextoPeticionIntegracionBolsa) { c.OperacionRef = "" }},
		{"versión cero", func(c *ContextoPeticionIntegracionBolsa) { c.VersionExpediente = 0 }},
		{"contrato ajeno", func(c *ContextoPeticionIntegracionBolsa) { c.ContratoVersion++ }},
		{"finalidad sin huella", func(c *ContextoPeticionIntegracionBolsa) { c.Finalidad.HuellaSHA256 = "" }},
		{"HMAC cero", func(c *ContextoPeticionIntegracionBolsa) { c.HuellaPeticionHMAC = strings.Repeat("0", 64) }},
		{"tiempo local", func(c *ContextoPeticionIntegracionBolsa) {
			c.SolicitadaEn = c.SolicitadaEn.In(time.FixedZone("local", 3600))
		}},
		{"ventana invertida", func(c *ContextoPeticionIntegracionBolsa) { c.ValidaHasta = c.SolicitadaEn }},
		{"ventana excesiva", func(c *ContextoPeticionIntegracionBolsa) {
			c.ValidaHasta = c.SolicitadaEn.Add(VigenciaMaximaPeticionIntegracionBolsa + time.Microsecond)
		}},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			alterado := contexto
			prueba.cambiar(&alterado)
			if err := alterado.ValidarEn(instante.Add(time.Minute)); !errors.Is(err, ErrPeticionIntegracionBolsaInvalida) {
				t.Fatalf("contexto alterado aceptado: %v", err)
			}
		})
	}
	if err := contexto.ValidarEn(contexto.ValidaHasta); !errors.Is(err, ErrPeticionIntegracionBolsaInvalida) {
		t.Fatalf("contexto caducado aceptado: %v", err)
	}

	solicitud := solicitudDisponibilidadBolsaPrueba(instante)
	solicitud.MaximoResultados = MaximoElementosIntegracionBolsa + 1
	if err := solicitud.ValidarEn(instante.Add(time.Minute)); !errors.Is(err, ErrLimiteIntegracionBolsaExcedido) {
		t.Fatalf("límite excesivo aceptado: %v", err)
	}
}

func TestResultadoDisponibilidadBolsaDistingueNegativoDeFallo(t *testing.T) {
	instante := instanteIntegracionBolsaPrueba()
	solicitud := solicitudDisponibilidadBolsaPrueba(instante)
	negativo := resultadoDisponibilidadBasePrueba(solicitud, instante)
	negativo.CantidadExacta = true
	if err := negativo.ValidarPara(solicitud); err != nil {
		t.Fatalf("ausencia autoritativa rechazada: %v", err)
	}

	positivo := negativo
	positivo.BolsaEncontrada = true
	positivo.Bolsa = referenciaIntegracionBolsaPrueba("bolsa:vigente", "b")
	positivo.Disponible = true
	positivo.CantidadDisponible = 7
	if err := positivo.ValidarPara(solicitud); err != nil {
		t.Fatalf("disponibilidad confirmada rechazada: %v", err)
	}

	pruebas := []struct {
		nombre  string
		cambiar func(*ResultadoDisponibilidadBolsa)
	}{
		{"operación ajena", func(r *ResultadoDisponibilidadBolsa) { r.OperacionRef = "operacion:otra" }},
		{"organización ajena", func(r *ResultadoDisponibilidadBolsa) { r.OrganizacionRef = "organizacion:otra" }},
		{"versión de expediente ajena", func(r *ResultadoDisponibilidadBolsa) { r.VersionExpediente++ }},
		{"categoría ajena", func(r *ResultadoDisponibilidadBolsa) { r.CategoriaRef = "categoria:otra" }},
		{"necesidad adulterada", func(r *ResultadoDisponibilidadBolsa) { r.Necesidad.Version++ }},
		{"procedencia caducada", func(r *ResultadoDisponibilidadBolsa) { r.Procedencia.EmitidaEn = solicitud.Contexto.ValidaHasta }},
		{"cantidad superior al límite", func(r *ResultadoDisponibilidadBolsa) { r.CantidadDisponible = solicitud.MaximoResultados + 1 }},
		{"positivo sin bolsa", func(r *ResultadoDisponibilidadBolsa) {
			r.BolsaEncontrada = false
			r.Bolsa = ReferenciaVersionadaIntegracionBolsa{}
		}},
		{"negativo truncado", func(r *ResultadoDisponibilidadBolsa) {
			r.Disponible = false
			r.CantidadDisponible = 0
			r.CantidadExacta = false
		}},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			alterado := positivo
			prueba.cambiar(&alterado)
			if err := alterado.ValidarPara(solicitud); !errors.Is(err, ErrRespuestaBolsaNoConfiable) {
				t.Fatalf("respuesta alterada aceptada: %v", err)
			}
		})
	}

	var cero ResultadoDisponibilidadBolsa
	if err := cero.ValidarPara(solicitud); !errors.Is(err, ErrRespuestaBolsaNoConfiable) {
		t.Fatalf("respuesta cero se convirtió en ausencia: %v", err)
	}
}

func TestOrdenBolsaSoloCruzaInstantaneaOpacaCompleta(t *testing.T) {
	instante := instanteIntegracionBolsaPrueba()
	solicitud := SolicitudOrdenBolsa{
		Contexto:         contextoIntegracionBolsaPrueba(instante),
		Necesidad:        referenciaIntegracionBolsaPrueba("necesidad:temporal", "a"),
		Bolsa:            referenciaIntegracionBolsaPrueba("bolsa:vigente", "b"),
		Politica:         referenciaIntegracionBolsaPrueba("politica:llamamiento", "c"),
		MaximoPosiciones: 200,
	}
	if err := solicitud.ValidarEn(instante.Add(time.Minute)); err != nil {
		t.Fatalf("solicitud de orden válida rechazada: %v", err)
	}
	resultado := ResultadoOrdenBolsa{
		OperacionRef: solicitud.Contexto.OperacionRef, ExpedienteRef: solicitud.Contexto.ExpedienteRef,
		OrganizacionRef: solicitud.Contexto.OrganizacionRef, VersionExpediente: solicitud.Contexto.VersionExpediente,
		CorrelacionRef: solicitud.Contexto.CorrelacionRef, Necesidad: solicitud.Necesidad,
		Bolsa: solicitud.Bolsa, Politica: solicitud.Politica,
		Resultado:     referenciaIntegracionBolsaPrueba("resultado:orden", "d"),
		OrdenGenerado: true, Orden: referenciaIntegracionBolsaPrueba("orden:instantanea", "e"),
		TotalPosiciones: 82, Procedencia: procedenciaIntegracionBolsaPrueba(instante.Add(time.Minute)),
	}
	if err := resultado.ValidarPara(solicitud); err != nil {
		t.Fatalf("orden válida rechazada: %v", err)
	}

	negativo := resultado
	negativo.OrdenGenerado = false
	negativo.Orden = ReferenciaVersionadaIntegracionBolsa{}
	negativo.TotalPosiciones = 0
	if err := negativo.ValidarPara(solicitud); err != nil {
		t.Fatalf("resultado negativo explícito rechazado: %v", err)
	}

	alterado := resultado
	alterado.Politica.Version++
	if err := alterado.ValidarPara(solicitud); !errors.Is(err, ErrRespuestaBolsaNoConfiable) {
		t.Fatalf("orden ligada a otra política aceptada: %v", err)
	}
	alterado = resultado
	alterado.TotalPosiciones = solicitud.MaximoPosiciones + 1
	if err := alterado.ValidarPara(solicitud); !errors.Is(err, ErrRespuestaBolsaNoConfiable) {
		t.Fatalf("orden que excede el límite aceptada: %v", err)
	}
}

func TestSolicitudYResultadoLlamamientoQuedanLigados(t *testing.T) {
	instante := instanteIntegracionBolsaPrueba()
	comando := comandoLlamamientoBolsaPrueba(instante)
	if err := comando.ValidarEn(instante.Add(time.Minute)); err != nil {
		t.Fatalf("comando válido rechazado: %v", err)
	}
	resultado := ResultadoSolicitudLlamamientoBolsa{
		OperacionRef: comando.Contexto.OperacionRef, ExpedienteRef: comando.Contexto.ExpedienteRef,
		OrganizacionRef: comando.Contexto.OrganizacionRef, VersionExpediente: comando.Contexto.VersionExpediente,
		CorrelacionRef: comando.Contexto.CorrelacionRef, Necesidad: comando.Necesidad,
		Bolsa: comando.Bolsa, Orden: comando.Orden, Politica: comando.Politica,
		Resultado:         referenciaIntegracionBolsaPrueba("resultado:propuesta", "d"),
		PropuestaGenerada: true, Propuesta: referenciaIntegracionBolsaPrueba("propuesta:primera", "e"),
		LlamamientoRef: "llamamiento:primero", SeleccionRef: "seleccion:opaca",
		OrdenSeleccionado: 4, Procedencia: procedenciaIntegracionBolsaPrueba(instante.Add(time.Minute)),
	}
	if err := resultado.ValidarPara(comando); err != nil {
		t.Fatalf("propuesta válida rechazada: %v", err)
	}

	sinPersonaElegible := resultado
	sinPersonaElegible.PropuestaGenerada = false
	sinPersonaElegible.Propuesta = ReferenciaVersionadaIntegracionBolsa{}
	sinPersonaElegible.LlamamientoRef = ""
	sinPersonaElegible.SeleccionRef = ""
	sinPersonaElegible.OrdenSeleccionado = 0
	if err := sinPersonaElegible.ValidarPara(comando); err != nil {
		t.Fatalf("resultado negativo explícito rechazado: %v", err)
	}

	alterado := resultado
	alterado.Orden.HuellaSHA256 = strings.Repeat("f", 64)
	if err := alterado.ValidarPara(comando); !errors.Is(err, ErrRespuestaBolsaNoConfiable) {
		t.Fatalf("propuesta de otro orden aceptada: %v", err)
	}
	alterado = resultado
	alterado.PropuestaGenerada = false
	if err := alterado.ValidarPara(comando); !errors.Is(err, ErrRespuestaBolsaNoConfiable) {
		t.Fatalf("resultado contradictorio aceptado: %v", err)
	}
	alterado = resultado
	alterado.OrdenSeleccionado = MaximoElementosIntegracionBolsa + 1
	if err := alterado.ValidarPara(comando); !errors.Is(err, ErrRespuestaBolsaNoConfiable) {
		t.Fatalf("posición fuera del límite aceptada: %v", err)
	}
}

func TestEventoBolsaEsVersionadoMinimizadoYSuAcuseIdempotente(t *testing.T) {
	instante := instanteIntegracionBolsaPrueba()
	evento := eventoLlamamientoBolsaPrueba(instante)
	if err := evento.Validar(); err != nil {
		t.Fatalf("evento válido rechazado: %v", err)
	}
	acuse := AcuseEventoLlamamientoBolsa{
		EventoRef: evento.EventoRef, Secuencia: evento.Secuencia, Aplicado: true,
		VersionResultante: evento.VersionExpedienteEsperada + 1,
		ActuacionRef:      "actuacion:llamamiento", AuditoriaRef: "auditoria:llamamiento",
		RegistradoEn: instante.Add(3 * time.Minute),
	}
	if err := acuse.ValidarPara(evento); err != nil {
		t.Fatalf("acuse válido rechazado: %v", err)
	}
	repetido := acuse
	repetido.Aplicado = false
	repetido.YaRegistrado = true
	if err := repetido.ValidarPara(evento); err != nil {
		t.Fatalf("acuse idempotente rechazado: %v", err)
	}

	alterado := evento
	alterado.HuellaCargaSHA256 = strings.Repeat("f", 64)
	if err := alterado.Validar(); !errors.Is(err, ErrEventoBolsaInvalido) {
		t.Fatalf("evento con carga no ligada aceptado: %v", err)
	}
	alterado = evento
	alterado.Estado.Version = 0
	if err := alterado.Validar(); !errors.Is(err, ErrEventoBolsaInvalido) {
		t.Fatalf("estado no gobernado aceptado: %v", err)
	}
	acuseIncoherente := acuse
	acuseIncoherente.YaRegistrado = true
	if err := acuseIncoherente.ValidarPara(evento); !errors.Is(err, ErrAcuseEventoBolsaNoConfiable) {
		t.Fatalf("acuse con doble resultado aceptado: %v", err)
	}
}

func TestContratoIntegracionBolsaNoExponePIINiAutoridadDeInterfaz(t *testing.T) {
	tipos := []reflect.Type{
		reflect.TypeOf(SolicitudDisponibilidadBolsa{}),
		reflect.TypeOf(ResultadoDisponibilidadBolsa{}),
		reflect.TypeOf(SolicitudOrdenBolsa{}),
		reflect.TypeOf(ResultadoOrdenBolsa{}),
		reflect.TypeOf(ComandoSolicitarLlamamientoBolsa{}),
		reflect.TypeOf(ResultadoSolicitudLlamamientoBolsa{}),
		reflect.TypeOf(EventoLlamamientoBolsa{}),
	}
	prohibidos := []string{
		"dni", "nie", "nif", "nombre", "apellidos", "correo", "email",
		"telefono", "movil", "direccion", "cookie", "sesion", "cabecera",
		"rol", "permiso", "actor", "perfil",
	}
	for _, tipo := range tipos {
		comprobarCamposSinPII(t, tipo, prohibidos)
	}

	contenido, err := json.Marshal(eventoLlamamientoBolsaPrueba(instanteIntegracionBolsaPrueba()))
	if err != nil {
		t.Fatalf("evento no serializable por un adaptador: %v", err)
	}
	texto := strings.ToLower(string(contenido))
	for _, prohibido := range prohibidos {
		if strings.Contains(texto, prohibido) {
			t.Fatalf("evento incorpora campo prohibido %q: %s", prohibido, texto)
		}
	}
}

func TestPuertosIntegracionBolsaConservanContextoCancelable(t *testing.T) {
	tipoContexto := reflect.TypeOf((*context.Context)(nil)).Elem()
	puertos := []reflect.Type{
		reflect.TypeOf((*ConsultaDisponibilidadBolsa)(nil)).Elem(),
		reflect.TypeOf((*ConsultaOrdenBolsa)(nil)).Elem(),
		reflect.TypeOf((*GestorLlamamientosBolsa)(nil)).Elem(),
		reflect.TypeOf((*BandejaEventosLlamamientoBolsa)(nil)).Elem(),
	}
	for _, puerto := range puertos {
		if puerto.NumMethod() != 1 {
			t.Fatalf("%v debe exponer una capacidad mínima", puerto)
		}
		metodo := puerto.Method(0)
		if metodo.Type.NumIn() < 1 || metodo.Type.In(0) != tipoContexto {
			t.Fatalf("%s perdió context.Context: %v", metodo.Name, metodo.Type)
		}
	}
}

func comprobarCamposSinPII(t *testing.T, tipo reflect.Type, prohibidos []string) {
	t.Helper()
	for indice := 0; indice < tipo.NumField(); indice++ {
		campo := tipo.Field(indice)
		nombre := strings.ToLower(campo.Name + " " + campo.Tag.Get("json"))
		for _, prohibido := range prohibidos {
			if strings.Contains(nombre, prohibido) {
				t.Fatalf("%s expone campo prohibido %q", tipo.Name(), campo.Name)
			}
		}
		if campo.Type.Kind() == reflect.Struct && campo.Type != reflect.TypeOf(time.Time{}) {
			comprobarCamposSinPII(t, campo.Type, prohibidos)
		}
	}
}

func instanteIntegracionBolsaPrueba() time.Time {
	return time.Date(2026, 7, 23, 8, 0, 0, 0, time.UTC)
}

func referenciaIntegracionBolsaPrueba(referencia string, caracter string) ReferenciaVersionadaIntegracionBolsa {
	return ReferenciaVersionadaIntegracionBolsa{
		Referencia: referencia, Version: 3, HuellaSHA256: strings.Repeat(caracter, 64),
	}
}

func contextoIntegracionBolsaPrueba(instante time.Time) ContextoPeticionIntegracionBolsa {
	return ContextoPeticionIntegracionBolsa{
		OperacionRef: "operacion:bolsa:temporal", OrganizacionRef: "organizacion:diputacion",
		ExpedienteRef:     "expediente:temporal",
		VersionExpediente: 5, CorrelacionRef: "correlacion:temporal",
		ContratoVersion:    VersionContratoIntegracionBolsa,
		Finalidad:          referenciaIntegracionBolsaPrueba("finalidad:cobertura", "f"),
		HuellaPeticionHMAC: strings.Repeat("9", 64),
		SolicitadaEn:       instante, ValidaHasta: instante.Add(5 * time.Minute),
	}
}

func procedenciaIntegracionBolsaPrueba(instante time.Time) ProcedenciaIntegracionBolsa {
	return ProcedenciaIntegracionBolsa{
		AutoridadRef: "autoridad:bolsa", RespuestaRef: "respuesta:bolsa",
		ContratoVersion: VersionContratoIntegracionBolsa,
		Fuente:          referenciaIntegracionBolsaPrueba("fuente:bolsa", "8"),
		EvidenciaRef:    "evidencia:bolsa", EmitidaEn: instante,
	}
}

func solicitudDisponibilidadBolsaPrueba(instante time.Time) SolicitudDisponibilidadBolsa {
	return SolicitudDisponibilidadBolsa{
		Contexto:     contextoIntegracionBolsaPrueba(instante),
		Necesidad:    referenciaIntegracionBolsaPrueba("necesidad:temporal", "a"),
		CategoriaRef: "categoria:auxiliar", MaximoResultados: 100,
	}
}

func resultadoDisponibilidadBasePrueba(
	solicitud SolicitudDisponibilidadBolsa,
	instante time.Time,
) ResultadoDisponibilidadBolsa {
	return ResultadoDisponibilidadBolsa{
		OperacionRef:      solicitud.Contexto.OperacionRef,
		OrganizacionRef:   solicitud.Contexto.OrganizacionRef,
		ExpedienteRef:     solicitud.Contexto.ExpedienteRef,
		VersionExpediente: solicitud.Contexto.VersionExpediente,
		CorrelacionRef:    solicitud.Contexto.CorrelacionRef, Necesidad: solicitud.Necesidad,
		CategoriaRef: solicitud.CategoriaRef,
		Resultado:    referenciaIntegracionBolsaPrueba("resultado:disponibilidad", "d"),
		Procedencia:  procedenciaIntegracionBolsaPrueba(instante.Add(time.Minute)),
	}
}

func comandoLlamamientoBolsaPrueba(instante time.Time) ComandoSolicitarLlamamientoBolsa {
	return ComandoSolicitarLlamamientoBolsa{
		Contexto:  contextoIntegracionBolsaPrueba(instante),
		Necesidad: referenciaIntegracionBolsaPrueba("necesidad:temporal", "a"),
		Bolsa:     referenciaIntegracionBolsaPrueba("bolsa:vigente", "b"),
		Orden:     referenciaIntegracionBolsaPrueba("orden:instantanea", "c"),
		Politica:  referenciaIntegracionBolsaPrueba("politica:llamamiento", "7"),
	}
}

func eventoLlamamientoBolsaPrueba(instante time.Time) EventoLlamamientoBolsa {
	procedencia := procedenciaIntegracionBolsaPrueba(instante.Add(2 * time.Minute))
	return EventoLlamamientoBolsa{
		EventoRef: "evento:llamamiento", Secuencia: 2,
		OrganizacionRef: "organizacion:diputacion",
		ExpedienteRef:   "expediente:temporal", CorrelacionRef: "correlacion:temporal",
		VersionExpedienteEsperada: 5,
		Necesidad:                 referenciaIntegracionBolsaPrueba("necesidad:temporal", "a"),
		Bolsa:                     referenciaIntegracionBolsaPrueba("bolsa:vigente", "b"),
		Orden:                     referenciaIntegracionBolsaPrueba("orden:instantanea", "c"),
		Politica:                  referenciaIntegracionBolsaPrueba("politica:llamamiento", "7"),
		Propuesta:                 referenciaIntegracionBolsaPrueba("propuesta:primera", "e"),
		LlamamientoRef:            "llamamiento:primero", SeleccionRef: "seleccion:opaca",
		Tipo:              referenciaIntegracionBolsaPrueba("evento_tipo:estado_llamamiento", "5"),
		Estado:            referenciaIntegracionBolsaPrueba("estado:llamamiento", "6"),
		HuellaCargaSHA256: procedencia.Fuente.HuellaSHA256,
		OcurridoEn:        instante.Add(time.Minute), PublicadoEn: instante.Add(2 * time.Minute),
		Procedencia: procedencia,
	}
}

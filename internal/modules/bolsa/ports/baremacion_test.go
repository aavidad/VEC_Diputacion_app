package ports

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

var (
	_ RepositorioBaremaciones              = repositorioBaremacionesContrato{}
	_ FuenteDatosBaremacion                = fuenteDatosBaremacionContrato{}
	_ CalculadorOficialBaremacion          = calculadorOficialContrato{}
	_ CatalogoPoliticasFirmaBaremacion     = catalogoPoliticasFirmaContrato{}
	_ CodificadorCanonicoDecision          = codificadorCanonicoContrato{}
	_ FirmadorInteractivo                  = firmadorInteractivoContrato{}
	_ ValidadorFirmaServidor               = validadorFirmaContrato{}
	_ SelladorTiempoFirma                  = selladorTiempoContrato{}
	_ AumentadorFirmaLongeva               = aumentadorFirmaContrato{}
	_ ArchivoEvidenciasFirmaBaremacion     = archivoFirmaContrato{}
	_ SelladorSolicitudBaremacion          = selladorSolicitudContrato{}
	_ GeneradorReferenciasOpacasBaremacion = generadorReferenciasContrato{}
	_ Reloj                                = relojContrato{}
)

var instantePuertosPrueba = time.Date(2026, time.July, 15, 9, 0, 0, 0, time.UTC)

const (
	principalBaremacionPuertosPrueba = "per_0123456789abcdefghijkl"
	perfilBaremacionPuertosPrueba    = "prf_0123456789abcdefghijkl"
)

func TestCargaProtegidaNoSeFiltraYCopiaLosBytes(t *testing.T) {
	original := []byte("contenido-administrativo-sensible")
	carga, err := NuevaCargaProtegida(original)
	if err != nil {
		t.Fatalf("NuevaCargaProtegida: %v", err)
	}
	original[0] = 'X'
	revelada := carga.Revelar()
	if string(revelada) != "contenido-administrativo-sensible" {
		t.Fatalf("constructor comparte memoria: %q", revelada)
	}
	revelada[0] = 'Y'
	if string(carga.Revelar()) != "contenido-administrativo-sensible" {
		t.Fatal("Revelar comparte memoria interna")
	}
	for _, formato := range []string{"%s", "%q", "%v", "%+v", "%#v"} {
		salida := fmt.Sprintf(formato, carga)
		if strings.Contains(salida, "contenido-administrativo-sensible") {
			t.Fatalf("carga filtrada con %s: %q", formato, salida)
		}
	}
	if _, err := json.Marshal(carga); !errors.Is(err, ErrSerializacionCargaProtegidaProhibida) {
		t.Fatalf("MarshalJSON debe cerrar: %v", err)
	}
	if _, err := carga.MarshalText(); !errors.Is(err, ErrSerializacionCargaProtegidaProhibida) {
		t.Fatalf("MarshalText debe cerrar: %v", err)
	}
	if err := (CargaProtegida{}).Validar(); !errors.Is(err, ErrCargaProtegidaInvalida) {
		t.Fatalf("valor cero admitido: %v", err)
	}
}

func TestTokenReservaSoloBase64URLCanonicoYNoSeFiltra(t *testing.T) {
	secreto := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdefghijklmn"))
	token, err := NuevoTokenReservaBaremacion(secreto)
	if err != nil {
		t.Fatalf("NuevoTokenReservaBaremacion: %v", err)
	}
	if err := token.Validar(); err != nil || token.Revelar() != secreto {
		t.Fatalf("token valido no recuperable: %v", err)
	}
	for _, formato := range []string{"%s", "%q", "%v", "%+v", "%#v"} {
		if salida := fmt.Sprintf(formato, token); strings.Contains(salida, secreto) {
			t.Fatalf("token filtrado con %s: %q", formato, salida)
		}
	}
	if _, err := json.Marshal(token); !errors.Is(err, ErrSerializacionTokenReservaProhibida) {
		t.Fatalf("MarshalJSON debe cerrar: %v", err)
	}
	invalidos := []string{
		"token con espacios", strings.Repeat("a", 31), strings.Repeat("a", 129),
		secreto + "=", "á" + secreto, "abc$" + secreto,
	}
	for _, invalido := range invalidos {
		if _, err := NuevoTokenReservaBaremacion(invalido); !errors.Is(err, ErrTokenReservaBaremacionInvalido) {
			t.Fatalf("token ambiguo admitido %q: %v", invalido, err)
		}
	}
}

func TestContextoEsObligatorioEnLecturasReservasYFirma(t *testing.T) {
	contexto := contextoOperacionValido(AccionConsultarBaremacionVigente, "baremacion-1")
	if err := contexto.Validar(); err != nil {
		t.Fatalf("contexto valido: %v", err)
	}
	if err := (ContextoOperacionBaremacion{}).Validar(); err == nil {
		t.Fatal("el valor cero de la capacidad concede acceso")
	}
	if _, err := json.Marshal(contexto); !errors.Is(err, ErrSerializacionAutorizacionProhibida) {
		t.Fatalf("la capacidad se puede serializar: %v", err)
	}
	decision := decisionAutorizacionValida(AccionConsultarBaremacionVigente, "baremacion-1")
	decision.Concedida = false
	if _, err := nuevaAutorizacionPrueba(decision); !errors.Is(err, ErrAutorizacionBaremacionInvalida) {
		t.Fatalf("decision denegada crea capacidad: %v", err)
	}
	decision = decisionAutorizacionValida(AccionConsultarBaremacionVigente, "baremacion-1")
	decision.CamposPermitidos = nil
	if _, err := nuevaAutorizacionPrueba(decision); !errors.Is(err, ErrAutorizacionBaremacionInvalida) {
		t.Fatalf("decision sin lista positiva de campos crea capacidad: %v", err)
	}
	if err := (SolicitudObtenerBaremacionVigente{Contexto: contexto, BaremacionMeritoRef: "baremacion-1"}).Validar(); err != nil {
		t.Fatalf("lectura vigente valida: %v", err)
	}
	contextoVersion := contextoOperacionValido(AccionConsultarVersionBaremacion, "baremacion-1")
	if err := (SolicitudObtenerVersionBaremacion{Contexto: contextoVersion, BaremacionMeritoRef: "baremacion-1", Numero: 1}).Validar(); err != nil {
		t.Fatalf("lectura historica valida: %v", err)
	}
	if err := (SolicitudObtenerVersionBaremacion{Contexto: contextoVersion, BaremacionMeritoRef: "baremacion-1"}).Validar(); err == nil {
		t.Fatal("lectura historica sin version admitida")
	}
	if err := (SolicitudObtenerVersionBaremacion{Contexto: contexto, BaremacionMeritoRef: "baremacion-1", Numero: 1}).Validar(); err == nil {
		t.Fatal("una autorizacion de lectura vigente se reutiliza para leer historial")
	}
	tiposConContexto := []reflect.Type{
		reflect.TypeOf(SolicitudReservarCambioBaremacion{}), reflect.TypeOf(SolicitudConfirmarCambioBaremacion{}),
		reflect.TypeOf(SolicitudAbandonarReservaBaremacion{}), reflect.TypeOf(SolicitudObtenerBaremacionVigente{}),
		reflect.TypeOf(SolicitudObtenerVersionBaremacion{}), reflect.TypeOf(SolicitudObtenerCriterioBaremacion{}),
		reflect.TypeOf(SolicitudObtenerEvidenciaBaremacion{}), reflect.TypeOf(SolicitudObtenerRepresentacionBaremacion{}),
		reflect.TypeOf(SolicitudCalcularPuntuacionOficial{}), reflect.TypeOf(SolicitudRecuperarCalculoOficial{}),
		reflect.TypeOf(SolicitudObtenerPoliticaFirma{}), reflect.TypeOf(SolicitudCodificarDecisionCanonica{}),
		reflect.TypeOf(SolicitudCustodiarDocumentoFirmable{}), reflect.TypeOf(SolicitudPrepararFirmaInteractiva{}),
		reflect.TypeOf(SolicitudConsultarFirmaInteractiva{}), reflect.TypeOf(SolicitudValidarFirmaServidor{}),
		reflect.TypeOf(SolicitudSellarTiempoFirma{}), reflect.TypeOf(SolicitudAumentarFirma{}),
		reflect.TypeOf(SolicitudRecuperarArtefactoFirma{}), reflect.TypeOf(SolicitudRecuperarValidacionFirma{}),
		reflect.TypeOf(SolicitudRecuperarSelloTiempo{}), reflect.TypeOf(SolicitudRecuperarAumentoFirma{}),
	}
	for _, tipo := range tiposConContexto {
		campo, existe := tipo.FieldByName("Contexto")
		if !existe || (campo.Type != reflect.TypeOf(ContextoOperacionBaremacion{}) &&
			campo.Type != reflect.TypeOf(ContextoOperacionFirma{})) {
			t.Fatalf("%s no exige contexto de autorizacion/finalidad/sujeto", tipo.Name())
		}
	}
}

func TestContextoTransportaEvidenciaYExigeVinculoV1Exacto(t *testing.T) {
	decisionReserva := decisionAutorizacionValida(AccionReservarAltaBaremacion, "baremacion-1")
	contextoReserva, err := nuevaAutorizacionPrueba(decisionReserva)
	if err != nil {
		t.Fatal(err)
	}
	evidencia, err := contextoReserva.EvidenciaUsoAutorizacion()
	if err != nil || evidencia.ValidarEn(instantePuertosPrueba) != nil {
		t.Fatalf("evidencia opaca no utilizable por el adaptador: %v", err)
	}
	datos, err := evidencia.Datos()
	if err != nil || datos.Decision.DecisionRef != contextoReserva.Proyeccion().AutorizacionRef {
		t.Fatalf("evidencia no ligada a la decision exacta: %+v / %v", datos, err)
	}
	if _, err := json.Marshal(evidencia); !errors.Is(err, puertosvec.ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
		t.Fatalf("la evidencia interna se puede serializar: %v", err)
	}

	decisionConfirmacion := decisionAutorizacionValida(AccionConfirmarAltaBaremacion, "baremacion-1")
	decisionConfirmacion.VinculoAutenticacionActor = decisionReserva.VinculoAutenticacionActor
	contextoConfirmacion, err := nuevaAutorizacionPrueba(decisionConfirmacion)
	if err != nil {
		t.Fatal(err)
	}
	if !contextoReserva.MismoVinculoAutenticacionQue(contextoConfirmacion) {
		t.Fatal("dos acciones de la misma sesion V1 no conservaron el vinculo exacto")
	}
	if contextoReserva.CoincideExactamenteCon(contextoConfirmacion) {
		t.Fatal("reservar y confirmar compartieron una decision de autorizacion")
	}

	decisionOtraSesion := decisionConfirmacion
	decisionOtraSesion.VinculoAutenticacionActor, err = pruebasvec.NuevoVinculoGenerico(
		instantePuertosPrueba.Add(time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	contextoOtraSesion, err := nuevaAutorizacionPrueba(decisionOtraSesion)
	if err != nil {
		t.Fatal(err)
	}
	if contextoReserva.MismoVinculoAutenticacionQue(contextoOtraSesion) {
		t.Fatal("una sesion o control V1 distinto se trato como el mismo vinculo")
	}
	partesConfirmacion, err := partesCanonicasAutorizacion(contextoConfirmacion)
	if err != nil {
		t.Fatal(err)
	}
	partesOtraSesion, err := partesCanonicasAutorizacion(contextoOtraSesion)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(partesConfirmacion, partesOtraSesion) {
		t.Fatal("el sello canonico no cubre la sesion y la decision V1 completas")
	}
	if !contextoReserva.CoincideExactamenteCon(contextoReserva) {
		t.Fatal("la misma capacidad valida no coincide consigo misma")
	}
	contextoReevaluado, err := NuevaAutorizacionOperacionBaremacion(
		decisionReserva, vinculoBaremacionPuertosPrueba(decisionReserva, "sujeto-001"),
		instantePuertosPrueba.Add(time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !contextoReserva.MismoVinculoAutenticacionQue(contextoReevaluado) ||
		contextoReserva.CoincideExactamenteCon(contextoReevaluado) {
		t.Fatal("una reevaluacion temporal se confundio con el mismo contexto exacto")
	}
}

func TestAutorizacionBaremacionFallaCerradaPorRecursoObligacionCamposYGarantia(t *testing.T) {
	tipo := reflect.TypeOf(ContextoOperacionBaremacion{})
	for indice := 0; indice < tipo.NumField(); indice++ {
		if tipo.Field(indice).IsExported() {
			t.Fatalf("la capacidad expone un campo fabricable: %s", tipo.Field(indice).Name)
		}
	}
	decision := decisionAutorizacionValida(AccionConsultarBaremacionVigente, "baremacion-1")
	decision.Obligaciones = []string{"registrar_justificacion_adicional"}
	if _, err := nuevaAutorizacionPrueba(decision); !errors.Is(err, ErrAutorizacionBaremacionInvalida) {
		t.Fatalf("obligacion no implementada ignorada: %v", err)
	}
	decision = decisionAutorizacionValida(AccionConsultarBaremacionVigente, "baremacion-1")
	decision.CamposPermitidos = append(decision.CamposPermitidos, "dato_nuevo")
	if _, err := nuevaAutorizacionPrueba(decision); !errors.Is(err, ErrAutorizacionBaremacionInvalida) {
		t.Fatalf("campo adicional amplio silenciosamente la operacion: %v", err)
	}
	decision = decisionAutorizacionValida(AccionConsultarBaremacionVigente, "baremacion-1")
	decision.ContextoRecursoHuellaSHA256 = huellaPruebaPuertos("9")
	if _, err := nuevaAutorizacionPrueba(decision); !errors.Is(err, ErrAutorizacionBaremacionInvalida) {
		t.Fatalf("huella de contexto ajena aceptada: %v", err)
	}
	decision = decisionAutorizacionValida(AccionConsultarBaremacionVigente, "baremacion-1")
	vinculoOtroSujeto := vinculoBaremacionPuertosPrueba(decision, "sujeto-002")
	if _, err := NuevaAutorizacionOperacionBaremacion(decision, vinculoOtroSujeto, instantePuertosPrueba); !errors.Is(err, ErrAutorizacionBaremacionInvalida) {
		t.Fatalf("huella de otro sujeto aceptada: %v", err)
	}
	decision = decisionAutorizacionValida(AccionConsultarBaremacionVigente, "baremacion-1")
	vinculoGarantiaInsuficiente := vinculoBaremacionPuertosPrueba(decision, "sujeto-001")
	vinculoGarantiaInsuficiente.Garantia = dominiovec.AuthAssuranceSubstantial
	if _, err := NuevaAutorizacionOperacionBaremacion(decision, vinculoGarantiaInsuficiente, instantePuertosPrueba); !errors.Is(err, ErrAutorizacionBaremacionInvalida) {
		t.Fatalf("garantia insuficiente admitida: %v", err)
	}
	decision = decisionAutorizacionValida(AccionConsultarBaremacionVigente, "baremacion-1")
	vinculoSesionCruzada := vinculoBaremacionPuertosPrueba(decision, "sujeto-001")
	vinculoSesionCruzada.SesionRef = "ses_otra_sesion_abcdefghijkl"
	if _, err := NuevaAutorizacionOperacionBaremacion(
		decision, vinculoSesionCruzada, instantePuertosPrueba,
	); !errors.Is(err, ErrAutorizacionBaremacionInvalida) {
		t.Fatalf("sesion distinta del vinculo V1 admitida: %v", err)
	}
	decision = decisionAutorizacionValida(AccionConsultarBaremacionVigente, "baremacion-1")
	vinculoSinCapacidad := vinculoBaremacionPuertosPrueba(decision, "sujeto-001")
	vinculoSinCapacidad.VinculoAutenticacionActor = dominiovec.VinculoAutenticacionActorV1{}
	if _, err := NuevaAutorizacionOperacionBaremacion(
		decision, vinculoSinCapacidad, instantePuertosPrueba,
	); !errors.Is(err, ErrAutorizacionBaremacionInvalida) {
		t.Fatalf("capacidad V1 ausente admitida: %v", err)
	}
	decision = decisionAutorizacionValida(AccionConsultarBaremacionVigente, "baremacion-1")
	if _, err := NuevaAutorizacionOperacionBaremacion(
		decision, vinculoBaremacionPuertosPrueba(decision, "sujeto-001"), decision.ValidaHasta,
	); !errors.Is(err, ErrAutorizacionBaremacionInvalida) {
		t.Fatalf("decision expirada admitida: %v", err)
	}
	contexto := contextoOperacionValido(AccionConsultarBaremacionVigente, "baremacion-1")
	proyeccion := contexto.Proyeccion()
	proyeccion.RecursoRef = "baremacion-2"
	proyeccion.CamposPermitidos[0] = "campo_fabricado"
	if contexto.Proyeccion().RecursoRef != "baremacion-1" || contexto.Proyeccion().CamposPermitidos[0] == "campo_fabricado" {
		t.Fatal("la proyeccion comparte el estado interno de la capacidad")
	}
	if err := (SolicitudObtenerBaremacionVigente{
		Contexto: contexto, BaremacionMeritoRef: "baremacion-2",
	}).Validar(); err == nil {
		t.Fatal("una capacidad para otro recurso fue reutilizada")
	}
}

func TestAutorizacionesAlmacenExigenRecursoEnriquecidoEvaluadoPorPDP(t *testing.T) {
	casos := []struct {
		nombre        string
		accion        AccionOperacionBaremacion
		recursoRef    string
		accionTecnica string
	}{
		{"custodia canonica", AccionCustodiarDecisionBaremacion, "decision-001", puertosvec.AccionAlmacenEscribir},
		{"custodia firmada", AccionCustodiarDocumentoFirmadoBaremacion, "documento-firmado-001", puertosvec.AccionAlmacenEscribir},
		{"retencion exacta", AccionRetenerDocumentoFirmadoBaremacion, "documento-firmado-001", puertosvec.AccionAlmacenAplicarRetencion},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			vinculos := vinculosAlmacenBaremacionPrueba(caso.accion)
			decision, recurso := decisionYRecursoAlmacenBaremacionValidos(caso.accion, caso.recursoRef, vinculos)
			vinculo := vinculoBaremacionPuertosPrueba(decision, "sujeto-001")
			if _, err := NuevaAutorizacionOperacionBaremacion(
				decision, vinculo, instantePuertosPrueba,
			); !errors.Is(err, ErrAutorizacionBaremacionInvalida) {
				t.Fatalf("la fabrica sin recurso enriquecido admitio %s: %v", caso.accion, err)
			}
			contexto, err := NuevaAutorizacionOperacionAlmacenBaremacion(
				decision, recurso, vinculo, instantePuertosPrueba,
			)
			if err != nil || contexto.Validar() != nil {
				t.Fatalf("recurso enriquecido exacto rechazado: %v", err)
			}
			// El llamador conserva su mapa, pero nunca comparte el que quedo
			// comprometido dentro de la capacidad.
			recurso.Atributos[puertosvec.AtributoAlmacenEfectoRef] = "efecto:alterado"
			if contexto.Validar() != nil {
				t.Fatal("la capacidad compartio los atributos mutables del recurso")
			}

			contextoAlmacen, err := crearContextoAlmacenBaremacionPrueba(contexto, caso.accion, vinculos)
			if err != nil {
				t.Fatalf("puente especifico rechazado: %v", err)
			}
			proyeccion, err := contextoAlmacen.Proyeccion()
			if err != nil || proyeccion.AccionNegocio != string(caso.accion) ||
				proyeccion.AccionTecnica != caso.accionTecnica ||
				proyeccion.EfectoRef != vinculos.EfectoRef ||
				proyeccion.ObjetoVinculado != vinculos.ObjetoVinculado {
				t.Fatalf("capacidad tecnica no ligada exactamente: %+v / %v", proyeccion, err)
			}
			vinculosAlterados := vinculos
			vinculosAlterados.EfectoRef = "efecto:distinto:001"
			if _, err := crearContextoAlmacenBaremacionPrueba(
				contexto, caso.accion, vinculosAlterados,
			); !errors.Is(err, ErrAutorizacionBaremacionInvalida) {
				t.Fatalf("se reconstruyo un efecto no evaluado: %v", err)
			}
		})
	}
}

func TestPuentesAlmacenNoCruzanAccionesNiObjetoRetenido(t *testing.T) {
	vinculosCustodia := vinculosAlmacenBaremacionPrueba(AccionCustodiarDocumentoFirmadoBaremacion)
	contextoCustodia := contextoOperacionAlmacenBaremacionValido(
		AccionCustodiarDocumentoFirmadoBaremacion, "documento-firmado-001", vinculosCustodia,
	)
	if _, err := contextoCustodia.CrearContextoAlmacenCustodiarDecision(vinculosCustodia); !errors.Is(err, ErrAutorizacionBaremacionInvalida) {
		t.Fatalf("custodia firmada se reutilizo como custodia canonica: %v", err)
	}
	if _, err := contextoCustodia.CrearContextoAlmacenRetenerDocumentoFirmado(vinculosCustodia); !errors.Is(err, ErrAutorizacionBaremacionInvalida) {
		t.Fatalf("custodia se reutilizo como retencion: %v", err)
	}

	vinculosRetencion := vinculosAlmacenBaremacionPrueba(AccionRetenerDocumentoFirmadoBaremacion)
	contextoRetencion := contextoOperacionAlmacenBaremacionValido(
		AccionRetenerDocumentoFirmadoBaremacion, "documento-firmado-001", vinculosRetencion,
	)
	vinculosOtroObjeto := vinculosRetencion
	vinculosOtroObjeto.ObjetoVinculado.Version = "version-firmada-002"
	if _, err := contextoRetencion.CrearContextoAlmacenRetenerDocumentoFirmado(
		vinculosOtroObjeto,
	); !errors.Is(err, ErrAutorizacionBaremacionInvalida) {
		t.Fatalf("retencion acepto otra version del objeto: %v", err)
	}
	if _, err := contextoRetencion.CrearContextoAlmacenCustodiarDocumentoFirmado(
		vinculosRetencion,
	); !errors.Is(err, ErrAutorizacionBaremacionInvalida) {
		t.Fatalf("retencion se reutilizo para escribir: %v", err)
	}
}

func TestAutorizacionAlmacenDeniegaAtributosIncompletosYVigenciaAmpliada(t *testing.T) {
	vinculos := vinculosAlmacenBaremacionPrueba(AccionCustodiarDecisionBaremacion)
	decision, recurso := decisionYRecursoAlmacenBaremacionValidos(
		AccionCustodiarDecisionBaremacion, "decision-001", vinculos,
	)
	vinculo := vinculoBaremacionPuertosPrueba(decision, "sujeto-001")

	recursoIncompleto := clonarRecursoAutorizableBaremacion(recurso)
	delete(recursoIncompleto.Atributos, puertosvec.AtributoAlmacenEfectoRef)
	huella, err := recursoIncompleto.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	decisionIncompleta := decision
	decisionIncompleta.ContextoRecursoHuellaSHA256 = huella
	if _, err := NuevaAutorizacionOperacionAlmacenBaremacion(
		decisionIncompleta, recursoIncompleto, vinculo, instantePuertosPrueba,
	); !errors.Is(err, ErrAutorizacionBaremacionInvalida) {
		t.Fatalf("recurso sin efecto exacto admitido: %v", err)
	}

	decisionAmplia := decision
	decisionAmplia.ValidaHasta = instantePuertosPrueba.Add(2 * VentanaMaximaUsoAutorizacionBaremacion)
	if _, err := NuevaAutorizacionOperacionAlmacenBaremacion(
		decisionAmplia, recurso, vinculo, instantePuertosPrueba,
	); !errors.Is(err, ErrAutorizacionBaremacionInvalida) {
		t.Fatalf("el puente ampliaria la vigencia recortada de Bolsa: %v", err)
	}

	decisionOrdinaria := decisionAutorizacionValida(AccionConsultarBaremacionVigente, "baremacion-1")
	recursoOrdinario := dominiovec.RecursoAutorizable{
		Referencia: "baremacion-1", ModuloID: "bolsa", Tipo: string(ClaseRecursoBaremacion),
		Ambitos: map[string]string{"sujeto_ref": "sujeto-001"},
	}
	if _, err := NuevaAutorizacionOperacionAlmacenBaremacion(
		decisionOrdinaria, recursoOrdinario,
		vinculoBaremacionPuertosPrueba(decisionOrdinaria, "sujeto-001"), instantePuertosPrueba,
	); !errors.Is(err, ErrAutorizacionBaremacionInvalida) {
		t.Fatalf("la variante de almacen admitio una lectura ordinaria: %v", err)
	}
}

func TestAccionesDeDecisionNoHeredanPermisosEntreTransiciones(t *testing.T) {
	casos := []struct {
		clase  dominiobolsa.ClaseDecisionTecnica
		accion AccionOperacionBaremacion
	}{
		{dominiobolsa.ClaseDecisionInicial, AccionAdoptarDecisionInicialBaremacion},
		{dominiobolsa.ClaseDecisionRectificacion, AccionRectificarDecisionBaremacion},
		{dominiobolsa.ClaseDecisionRevocacion, AccionRevocarDecisionBaremacion},
		{dominiobolsa.ClaseDecisionRehabilitacion, AccionRehabilitarDecisionBaremacion},
	}
	for _, caso := range casos {
		accion, existe := AccionAdopcionParaClase(caso.clase)
		if !existe || accion != caso.accion {
			t.Fatalf("mapeo no exacto para %s: %s, %v", caso.clase, accion, existe)
		}
		contexto := contextoOperacionValido(caso.accion, "baremacion-1")
		for _, otra := range casos {
			err := contexto.ValidarPara(otra.accion, ClaseRecursoBaremacion, "baremacion-1")
			if otra.accion == caso.accion && err != nil {
				t.Fatalf("accion exacta %s denegada: %v", caso.accion, err)
			}
			if otra.accion != caso.accion && !errors.Is(err, ErrAutorizacionBaremacionInvalida) {
				t.Fatalf("%s heredo permiso de %s", otra.accion, caso.accion)
			}
		}
	}
	if accion, existe := AccionAdopcionParaClase(dominiobolsa.ClaseDecisionTecnica("desconocida")); existe || accion != "" {
		t.Fatalf("clase desconocida obtuvo permiso: %s", accion)
	}
}

func TestAutorizacionExpiraEnElMinimoDeSesionDecisionYVentanaBreve(t *testing.T) {
	decision := decisionAutorizacionValida(AccionConsultarBaremacionVigente, "baremacion-1")
	contexto, err := NuevaAutorizacionOperacionBaremacion(
		decision, vinculoBaremacionPuertosPrueba(decision, "sujeto-001"), instantePuertosPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	limite := instantePuertosPrueba.Add(VentanaMaximaUsoAutorizacionBaremacion)
	if !contexto.Proyeccion().ValidaHasta.Equal(limite) {
		t.Fatalf("vigencia no acotada a ventana breve: %v", contexto.Proyeccion().ValidaHasta)
	}
	if err := contexto.ValidarVigentePara(
		AccionConsultarBaremacionVigente, ClaseRecursoBaremacion, "baremacion-1", limite,
	); !errors.Is(err, ErrAutorizacionBaremacionInvalida) {
		t.Fatalf("PEP admitio capacidad en su limite de expiracion: %v", err)
	}
}

func TestReservaExigeHMACSemanticaOCCYVentanaAcotada(t *testing.T) {
	solicitud := solicitudReservaAltaValida()
	if err := solicitud.Validar(); err != nil {
		t.Fatalf("reserva valida: %v", err)
	}
	solicitud.ExpiraEn = solicitud.SolicitadaEn.Add(VentanaMaximaReservaBaremacion + time.Nanosecond)
	if err := solicitud.Validar(); err == nil {
		t.Fatal("ventana de reserva sin limite")
	}
	solicitud = solicitudReservaAltaValida()
	solicitud.HuellaSolicitudHMAC = huellaPruebaPuertos("a")
	if err := solicitud.Validar(); err == nil {
		t.Fatal("SHA sin clave admitido como huella de idempotencia")
	}
	solicitud = solicitudReservaAltaValida()
	solicitud.VersionEsperada = &ReferenciaVersionBaremacion{BaremacionMeritoRef: solicitud.BaremacionMeritoRef, Numero: 1, HuellaEstadoSHA256: huellaPruebaPuertos("a")}
	if err := solicitud.Validar(); err == nil {
		t.Fatal("alta con version esperada admitida")
	}
	solicitud = solicitudReservaAltaValida()
	solicitud.Clase = ClaseCambioIncorporarDecision
	if err := solicitud.Validar(); err == nil {
		t.Fatal("incorporacion sin OCC admitida")
	}
}

func TestRespuestaReservaNoPuedeCruzarseEntreSolicitudes(t *testing.T) {
	solicitud := solicitudReservaAltaValida()
	token, err := NuevoTokenReservaBaremacion(base64.RawURLEncoding.EncodeToString([]byte("respuesta-reserva-exacta-32bytes")))
	if err != nil {
		t.Fatal(err)
	}
	respuesta := ReservaCambioBaremacion{
		Token: token, BaremacionMeritoRef: solicitud.BaremacionMeritoRef, Clase: solicitud.Clase,
		HuellaSolicitudHMAC: solicitud.HuellaSolicitudHMAC, ExpiraEn: solicitud.ExpiraEn,
	}
	if err := respuesta.ValidarPara(solicitud); err != nil {
		t.Fatalf("respuesta exacta denegada: %v", err)
	}
	otra := solicitud
	otra.BaremacionMeritoRef = "baremacion-2"
	otra.Contexto = contextoOperacionValido(AccionReservarAltaBaremacion, otra.BaremacionMeritoRef)
	otra.ClaveIdempotencia = "alta-baremacion-2"
	if err := respuesta.ValidarPara(otra); !errors.Is(err, ErrReservaBaremacionNoValida) {
		t.Fatalf("respuesta cruzada admitida: %v", err)
	}
}

func TestVersionYCopiasDefensivasConservanHuellaExacta(t *testing.T) {
	baremacion := baremacionValidaPrueba(t)
	huella, err := baremacion.HuellaEstadoSHA256()
	if err != nil {
		t.Fatalf("huella: %v", err)
	}
	version := VersionBaremacion{
		Referencia: ReferenciaVersionBaremacion{BaremacionMeritoRef: baremacion.ID, Numero: 1, HuellaEstadoSHA256: huella},
		Agregado:   baremacion, ConfirmadaEn: instantePuertosPrueba.Add(time.Minute),
	}
	if err := version.Validar(); err != nil {
		t.Fatalf("version valida: %v", err)
	}
	clon, err := version.Clonar()
	if err != nil {
		t.Fatalf("clonar: %v", err)
	}
	clon.Agregado.EvidenciasIniciales[0].Referencia.DocumentoRef = "alterado"
	if version.Agregado.EvidenciasIniciales[0].Referencia.DocumentoRef == "alterado" {
		t.Fatal("VersionBaremacion.Clonar comparte evidencias")
	}
	version.Referencia.Numero = 2
	if err := version.Validar(); err == nil {
		t.Fatal("version de repositorio no coincide con cardinalidad del historial")
	}
}

func TestConfirmacionEsTipadaSinAuditoriaEventosONiMapasLibres(t *testing.T) {
	tipo := reflect.TypeOf(SolicitudConfirmarCambioBaremacion{})
	if _, existe := tipo.FieldByName("Auditoria"); existe {
		t.Fatal("confirmacion vuelve a aceptar AuditEntry libre")
	}
	if _, existe := tipo.FieldByName("EventosOutbox"); existe {
		t.Fatal("confirmacion vuelve a aceptar eventos libres")
	}
	if _, existe := tipo.FieldByName("HuellaSolicitudHMAC"); !existe {
		t.Fatal("confirmacion no queda ligada criptograficamente a la reserva")
	}
	tipos := []reflect.Type{
		tipo, reflect.TypeOf(TrazabilidadCambioBaremacion{}), reflect.TypeOf(EvidenciaTransaccionBaremacion{}),
		reflect.TypeOf(SolicitudCalcularPuntuacionOficial{}), reflect.TypeOf(PoliticaFirmaBaremacion{}),
		reflect.TypeOf(ArtefactoFirma{}), reflect.TypeOf(ValidacionFirmaServidor{}),
	}
	for _, actual := range tipos {
		for i := 0; i < actual.NumField(); i++ {
			if actual.Field(i).Type.Kind() == reflect.Map {
				t.Fatalf("%s.%s declara mapa libre", actual.Name(), actual.Field(i).Name)
			}
		}
	}
	baremacion := baremacionValidaPrueba(t)
	token, err := NuevoTokenReservaBaremacion(base64.RawURLEncoding.EncodeToString([]byte("abcdefghijklmnopqrstuvwx")))
	if err != nil {
		t.Fatal(err)
	}
	solicitud := SolicitudConfirmarCambioBaremacion{
		Contexto: contextoOperacionValido(AccionConfirmarAltaBaremacion, baremacion.ID), Token: token, Clase: ClaseCambioAltaBaremacion,
		HuellaSolicitudHMAC: "hmac-sha256:reserva_1:" + huellaPruebaPuertos("a"), Agregado: baremacion,
		Trazabilidad: TrazabilidadCambioBaremacion{MotivoClave: "alta_merito", Motivo: "Alta del merito calculado oficialmente."},
		ConfirmadaEn: instantePuertosPrueba.Add(time.Minute),
	}
	if err := solicitud.Validar(); err != nil {
		t.Fatalf("confirmacion tipada valida: %v", err)
	}
	clon, err := solicitud.Clonar()
	if err != nil {
		t.Fatalf("clonar confirmacion: %v", err)
	}
	clon.Agregado.CalculoInicial.Evidencias[0].Referencia.DocumentoRef = "alterado"
	if solicitud.Agregado.CalculoInicial.Evidencias[0].Referencia.DocumentoRef == "alterado" {
		t.Fatal("confirmacion clonada comparte evidencias")
	}
	solicitud.Agregado.Decisiones = []dominiobolsa.DecisionTecnica{{}}
	if err := solicitud.Validar(); err == nil {
		t.Fatal("alta que importa un historial admitida")
	}
}

func TestCalculoOficialNoAceptaPuntosCalculadosDelCliente(t *testing.T) {
	tipoSolicitud := reflect.TypeOf(SolicitudCalcularPuntuacionOficial{})
	if _, existe := tipoSolicitud.FieldByName("PuntosCalculados"); existe {
		t.Fatal("el puerto acepta puntos calculados del cliente")
	}
	for _, tipoDominio := range []reflect.Type{
		reflect.TypeOf(dominiobolsa.AltaMeritoBaremable{}),
		reflect.TypeOf(dominiobolsa.PropuestaDecisionTecnica{}),
	} {
		if _, existe := tipoDominio.FieldByName("PuntosCalculados"); existe {
			t.Fatalf("%s acepta puntos calculados sueltos", tipoDominio.Name())
		}
	}
	criterio := criterioValidoPrueba()
	if criterio.ProcesoRef == "" || criterio.ReglaCalculo.Validar() != nil {
		t.Fatalf("criterio no liga proceso y regla: %+v", criterio)
	}
	resultado := ResultadoCalculoOficial{
		Calculo: calculoValidoPrueba(4_250_000, "inicial"), EvidenciaGobiernoRef: "evidencia-gobierno-calculo-1",
		HuellaEvidenciaSHA256: huellaPruebaPuertos("7"),
	}
	if err := resultado.Validar(); err != nil {
		t.Fatalf("resultado confiable valido: %v", err)
	}
	resultado.Calculo.Regla.Version++
	if err := resultado.Validar(); err == nil {
		t.Fatal("resultado con regla distinta del criterio admitido")
	}
}

func TestCalculoOficialValidaEvidenciasConfiablesExactasYCopiaDefensiva(t *testing.T) {
	evidencia := evidenciaConfiableValidaPrueba(t)
	solicitud := SolicitudCalcularPuntuacionOficial{
		Contexto: contextoOperacionValido(AccionCalcularPuntuacionBaremacion, "baremacion-001"), BaremacionMeritoRef: "baremacion-001",
		ProcesoRef: "proceso-selectivo-2026-017", SolicitudRef: "solicitud-001", SujetoRef: "sujeto-001",
		Criterio: criterioValidoPrueba(), Evidencias: []EvidenciaBaremacionConfiable{evidencia},
		PuntosDeclarados: 5_000_000, SolicitadaEn: instantePuertosPrueba.Add(-2 * time.Minute),
	}
	if err := solicitud.Validar(); err != nil {
		t.Fatalf("solicitud oficial valida: %v", err)
	}
	resultado := ResultadoCalculoOficial{
		Calculo: calculoValidoPrueba(4_250_000, "inicial"), EvidenciaGobiernoRef: "evidencia-gobierno-calculo-1",
		HuellaEvidenciaSHA256: huellaPruebaPuertos("7"),
	}
	if err := resultado.ValidarPara(solicitud); err != nil {
		t.Fatalf("resultado no enlazado a entradas confiables: %v", err)
	}
	clonSolicitud, err := solicitud.Clonar()
	if err != nil {
		t.Fatalf("clonar solicitud: %v", err)
	}
	clonSolicitud.Evidencias[0].Documento.Relaciones[0].Referencia = "alterada"
	if solicitud.Evidencias[0].Documento.Relaciones[0].Referencia == "alterada" {
		t.Fatal("SolicitudCalcularPuntuacionOficial.Clonar comparte relaciones")
	}
	clonResultado, err := resultado.Clonar()
	if err != nil {
		t.Fatalf("clonar resultado: %v", err)
	}
	clonResultado.Calculo.Evidencias[0].Referencia.DocumentoRef = "alterado"
	if resultado.Calculo.Evidencias[0].Referencia.DocumentoRef == "alterado" {
		t.Fatal("ResultadoCalculoOficial.Clonar comparte evidencias")
	}
	resultado.Calculo.Evidencias[0].Referencia.HuellaSHA256 = huellaPruebaPuertos("0")
	if err := resultado.ValidarPara(solicitud); !errors.Is(err, ErrCalculoOficialNoReproducible) {
		t.Fatalf("resultado calculado con otra evidencia admitido: %v", err)
	}
}

func TestRepresentacionConfiableExigeDisponibilidadYAntivirusApto(t *testing.T) {
	base := RepresentacionBaremacionConfiable{
		Representacion: dominiovec.RepresentacionDocumento{
			ID:        "representacion-001",
			Documento: dominiovec.ReferenciaDocumento{ID: "documento-001", Version: 1},
			Tipo:      dominiovec.TipoRepresentacionVisualizacion, Formato: dominiovec.FormatoDocumentoPDF,
			MIME: "application/pdf", NombreFichero: "merito.pdf", Tamano: 1024,
			HuellaContenidoSHA256: huellaPruebaPuertos("b"),
			HuellaFuenteHMAC:      "hmac-sha256:documentos_1:" + huellaPruebaPuertos("3"),
			ReferenciaContenido:   "objeto:merito:001",
			EstadoTecnico:         dominiovec.EstadoRepresentacionDisponible,
			EstadoAntivirus:       dominiovec.EstadoAntivirusLimpio,
			GeneradaPor:           "sistema-documental", GeneradaEn: instantePuertosPrueba.Add(-time.Minute),
		},
		EvidenciaConsultaRef:  "evidencia-consulta-representacion-001",
		HuellaEvidenciaSHA256: huellaPruebaPuertos("8"), ConsultadaEn: instantePuertosPrueba,
	}
	if err := base.Validar(); err != nil {
		t.Fatalf("representacion disponible y limpia rechazada: %v", err)
	}
	noAplicable := base
	noAplicable.Representacion.EstadoAntivirus = dominiovec.EstadoAntivirusNoAplica
	if err := noAplicable.Validar(); err != nil {
		t.Fatalf("representacion generada internamente rechazada: %v", err)
	}

	for _, estado := range []dominiovec.EstadoRepresentacionDocumento{
		dominiovec.EstadoRepresentacionPendiente,
		dominiovec.EstadoRepresentacionCuarentena,
		dominiovec.EstadoRepresentacionRechazada,
		dominiovec.EstadoRepresentacionRetirada,
	} {
		caso := base
		caso.Representacion.EstadoTecnico = estado
		if err := caso.Validar(); !errors.Is(err, ErrRepresentacionBaremacionNoConfiable) {
			t.Errorf("estado tecnico %q admitido: %v", estado, err)
		}
	}
	for _, estado := range []dominiovec.EstadoAntivirusDocumento{
		dominiovec.EstadoAntivirusPendiente,
		dominiovec.EstadoAntivirusRechazado,
		dominiovec.EstadoAntivirusError,
	} {
		caso := base
		caso.Representacion.EstadoAntivirus = estado
		if err := caso.Validar(); !errors.Is(err, ErrRepresentacionBaremacionNoConfiable) {
			t.Errorf("estado antivirus %q admitido: %v", estado, err)
		}
	}
}

type lectorCierreTipadoNuloPrueba struct{}

func (*lectorCierreTipadoNuloPrueba) Read([]byte) (int, error) { return 0, nil }
func (*lectorCierreTipadoNuloPrueba) Close() error             { return nil }

func TestBinarioFirmadoRecuperadoRechazaContenidoNuloTipado(t *testing.T) {
	solicitud := SolicitudRecuperarBinarioFirmado{
		Contexto: ContextoOperacionFirma{
			ContextoOperacionBaremacion: contextoOperacionValido(
				AccionRecuperarBinarioFirmadoBaremacion,
				"documento-firmado-001",
			),
			OperacionRef: "operacion-recuperar-binario-001",
		},
		DocumentoFirmadoRef:   "documento-firmado-001",
		HuellaDocumentoSHA256: huellaPruebaPuertos("f"),
		LimiteBytes:           4096,
	}
	if err := solicitud.Validar(); err != nil {
		t.Fatalf("solicitud de prueba invalida: %v", err)
	}
	var contenido *lectorCierreTipadoNuloPrueba
	binario := BinarioFirmadoRecuperado{
		DocumentoFirmadoRef:   solicitud.DocumentoFirmadoRef,
		HuellaDocumentoSHA256: solicitud.HuellaDocumentoSHA256,
		MIME:                  "application/pdf", Tamano: 1024, Contenido: contenido,
		EvidenciaRecuperacionRef: "evidencia-recuperacion-001",
		HuellaEvidenciaSHA256:    huellaPruebaPuertos("e"), RecuperadoEn: instantePuertosPrueba,
	}
	if err := binario.ValidarPara(solicitud); !errors.Is(err, ErrEvidenciaFirmaNoEncontrada) {
		t.Fatalf("contenido nulo tipado admitido: %v", err)
	}
}

func TestPoliticaExigeFirmaInteractivaYValidacionServidorConCapasExactas(t *testing.T) {
	politica := politicaFirmaValidaPrueba()
	if err := politica.Validar(); err != nil {
		t.Fatalf("politica valida: %v", err)
	}
	politica.RequiereFirmaInteractiva = false
	if !errors.Is(politica.Validar(), ErrPoliticaFirmaInsegura) {
		t.Fatal("politica sin firma interactiva admitida")
	}
	politica = politicaFirmaValidaPrueba()
	politica.RequiereValidacionServidor = false
	if !errors.Is(politica.Validar(), ErrPoliticaFirmaInsegura) {
		t.Fatal("politica sin validacion servidor admitida")
	}
	politica = politicaFirmaValidaPrueba()
	politica.HuellaPoliticaSelloTiempoSHA256 = ""
	if !errors.Is(politica.Validar(), ErrPoliticaFirmaInsegura) {
		t.Fatal("sello sin politica exacta admitido")
	}
	politica = politicaFirmaValidaPrueba()
	politica.PoliticaLongevidadVersion = 0
	if !errors.Is(politica.Validar(), ErrPoliticaFirmaInsegura) {
		t.Fatal("longevidad sin version de politica admitida")
	}
	politica = politicaFirmaValidaPrueba()
	solicitud := SolicitudObtenerPoliticaFirma{
		Contexto: contextoOperacionValido(AccionConsultarPoliticaFirmaBaremacion, politica.Referencia), Referencia: politica.Referencia, Version: politica.Version,
		HuellaEsperadaSHA256: politica.HuellaSHA256, VigenteEn: instantePuertosPrueba,
	}
	if err := politica.ValidarPara(solicitud); err != nil {
		t.Fatalf("respuesta de politica confiable: %v", err)
	}
	solicitud.Version++
	if !errors.Is(politica.ValidarPara(solicitud), ErrPoliticaFirmaNoVigente) {
		t.Fatal("catalogo devolvio otra version de politica sin ser detectada")
	}
}

func TestCodificacionCustodiaYFirmaQuedanEnlazadasPorReferenciaYHuella(t *testing.T) {
	contenido := contenidoDecisionValidoPrueba(t)
	politica := politicaFirmaValidaPrueba()
	codificacion := codificacionValidaPrueba(t, contenido, politica)
	if codificacion.AutorizacionDecisionRef == codificacion.AutorizacionCodificacionRef {
		t.Fatal("adopcion y codificacion reutilizan la misma autorizacion")
	}
	solicitudCodificacion := SolicitudCodificarDecisionCanonica{
		Contexto: contextoOperacionValido(AccionCodificarDecisionBaremacion, contenido.ID),
		AutorizacionDecision: contextoOperacionValido(
			AccionAdoptarDecisionInicialBaremacion, contenido.BaremacionMeritoRef,
		),
		Contenido: contenido, Politica: politica,
	}
	solicitudConfundida := solicitudCodificacion
	solicitudConfundida.Contexto = solicitudCodificacion.AutorizacionDecision
	if err := solicitudConfundida.Validar(); err == nil {
		t.Fatal("la autorizacion para adoptar una decision sirvio para codificarla")
	}
	solicitudCustodia := solicitudCustodiaValidaPrueba(codificacion)
	escritura, err := solicitudCustodia.PrepararEscritura()
	proyeccionEscritura, errProyeccion := escritura.Contexto.Proyeccion()
	if err != nil || escritura.Validar() != nil || escritura.HuellaSHA256 != codificacion.HuellaDocumentoSHA256 ||
		errProyeccion != nil ||
		proyeccionEscritura.AutorizacionRef != solicitudCustodia.Contexto.Proyeccion().AutorizacionRef ||
		proyeccionEscritura.CorrelacionRef != solicitudCustodia.Contexto.Proyeccion().CorrelacionRef {
		t.Fatalf("puente al almacen VEC invalido: %+v / %v", escritura, err)
	}
	resultadoAlmacen := resultadoAlmacenValidoPrueba(t, solicitudCustodia)
	custodia, err := NuevoDocumentoFirmableCustodiado(solicitudCustodia, resultadoAlmacen)
	if err != nil {
		t.Fatalf("custodia valida: %v", err)
	}
	if err := custodia.ValidarPara(solicitudCustodia); err != nil {
		t.Fatalf("enlace codificacion-custodia: %v", err)
	}
	if custodia.AutorizacionDecisionRef == custodia.AutorizacionCodificacionRef ||
		custodia.AutorizacionDecisionRef == custodia.AutorizacionCustodiaRef ||
		custodia.AutorizacionCodificacionRef == custodia.AutorizacionCustodiaRef {
		t.Fatal("la cadena documental reutiliza una autorizacion entre operaciones")
	}
	alterado := resultadoAlmacen
	alterado.Objeto.HuellaSHA256 = huellaPruebaPuertos("9")
	if _, err := NuevoDocumentoFirmableCustodiado(solicitudCustodia, alterado); !errors.Is(err, ErrCustodiaDocumentoFirmableInvalida) {
		t.Fatalf("se firmarian bytes distintos de los codificados: %v", err)
	}
	solicitud := solicitudFirmaInteractivaValidaPrueba(custodia, politica)
	if autorizacionFirma := solicitud.Contexto.Proyeccion().AutorizacionRef; autorizacionFirma == custodia.AutorizacionDecisionRef ||
		autorizacionFirma == custodia.AutorizacionCodificacionRef || autorizacionFirma == custodia.AutorizacionCustodiaRef {
		t.Fatal("la firma reutiliza una autorizacion documental anterior")
	}
	if err := solicitud.Validar(); err != nil {
		t.Fatalf("solicitud firma valida: %v", err)
	}
	solicitud.Documento.HuellaDocumentoSHA256 = huellaPruebaPuertos("8")
	if err := solicitud.Validar(); err == nil {
		t.Fatal("firma admite custodia con huella alterada")
	}
}

func TestSesionFirmaTieneVentanaMaximaYRespuestaConEstadoCoherente(t *testing.T) {
	contenido := contenidoDecisionValidoPrueba(t)
	politica := politicaFirmaValidaPrueba()
	codificacion := codificacionValidaPrueba(t, contenido, politica)
	solicitudCustodia := solicitudCustodiaValidaPrueba(codificacion)
	custodia, err := NuevoDocumentoFirmableCustodiado(
		solicitudCustodia, resultadoAlmacenValidoPrueba(t, solicitudCustodia),
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := solicitudFirmaInteractivaValidaPrueba(custodia, politica)
	sesion := sesionFirmaValidaPrueba(t, solicitud)
	if err := sesion.ValidarPara(solicitud); err != nil {
		t.Fatalf("sesion valida: %v", err)
	}
	sesion.ExpiraEn = sesion.PreparadaEn.Add(VentanaMaximaSesionFirmaInteractiva + time.Nanosecond)
	if err := sesion.Validar(); err == nil {
		t.Fatal("sesion sin ventana maxima")
	}
	consulta := ConsultaFirmaInteractiva{
		SesionRef: "sesion-firma-1", Estado: EstadoSesionFirmaPendiente,
		EvidenciaConsultaRef: "evidencia-consulta-1", HuellaEvidenciaSHA256: huellaPruebaPuertos("a"),
		ConsultadaEn: instantePuertosPrueba.Add(3 * time.Minute),
	}
	if err := consulta.Validar(); err != nil {
		t.Fatalf("consulta pendiente valida: %v", err)
	}
	artefacto := artefactoFirmaValidoPrueba(t, contenido, politica, "inicial")
	consulta.Artefacto = &artefacto
	if err := consulta.Validar(); err == nil {
		t.Fatal("estado pendiente con artefacto admitido")
	}
	consulta.Estado = EstadoSesionFirmaCompletada
	consultaSolicitud := SolicitudConsultarFirmaInteractiva{
		Contexto: contextoFirmaValido(AccionConsultarFirmaDecisionBaremacion, consulta.SesionRef), SesionRef: consulta.SesionRef, Documento: custodia,
		HuellaContenidoSHA256: custodia.HuellaContenidoSHA256, PoliticaFirmaRef: politica.Referencia,
		PoliticaFirmaVersion: politica.Version, HuellaPoliticaSHA256: politica.HuellaSHA256,
		FirmanteRef: principalBaremacionPuertosPrueba, PerfilFirmanteClave: perfilBaremacionPuertosPrueba,
	}
	if err := consulta.ValidarPara(consultaSolicitud); err != nil {
		t.Fatalf("consulta completada no enlazada a custodia/politica: %v", err)
	}
	sesion = sesionFirmaValidaPrueba(t, solicitud)
	if err := artefacto.ValidarPara(solicitud, sesion); err != nil {
		t.Fatalf("artefacto no enlazado a sesion/custodia: %v", err)
	}
}

func TestValidacionServidorCardinalidadEstadosYCopiaDefensiva(t *testing.T) {
	contenido := contenidoDecisionValidoPrueba(t)
	politica := politicaFirmaValidaPrueba()
	artefacto := artefactoFirmaValidoPrueba(t, contenido, politica, "inicial")
	validacion := validacionFirmaValidaPrueba(artefacto, instantePuertosPrueba.Add(6*time.Minute), "inicial")
	if !validacion.AptaParaPerfil(politica, PerfilFirmaPAdESBaselineB) {
		t.Fatalf("validacion valida no apta: %+v", validacion)
	}
	clon, err := validacion.Clonar()
	if err != nil {
		t.Fatalf("clonar validacion: %v", err)
	}
	clon.Comprobaciones[0].Clave = "alterada"
	if validacion.Comprobaciones[0].Clave == "alterada" {
		t.Fatal("clon comparte comprobaciones")
	}
	validacion.Comprobaciones = append(validacion.Comprobaciones, validacion.Comprobaciones[0])
	if err := validacion.Validar(); err == nil {
		t.Fatal("comprobacion duplicada admitida")
	}
	validacion = validacionFirmaValidaPrueba(artefacto, instantePuertosPrueba.Add(6*time.Minute), "inicial")
	validacion.Comprobaciones[0].Estado = EstadoComprobacionIndeterminada
	if validacion.AptaParaPerfil(politica, PerfilFirmaPAdESBaselineB) {
		t.Fatal("validacion indeterminada admitida para decision")
	}
}

func TestPoliticaFirmaDeniegaChecklistIncompletaDesconocidaOExpirada(t *testing.T) {
	politica := politicaFirmaValidaPrueba()
	politica.ComprobacionesObligatorias = politica.ComprobacionesObligatorias[:len(politica.ComprobacionesObligatorias)-1]
	if err := politica.Validar(); err == nil {
		t.Fatal("politica con comprobacion obligatoria ausente admitida")
	}
	politica = politicaFirmaValidaPrueba()
	politica.ComprobacionesObligatorias[0] = "comprobacion_desconocida"
	if err := politica.Validar(); err == nil {
		t.Fatal("politica con comprobacion desconocida admitida")
	}
	politica = politicaFirmaValidaPrueba()
	if politica.VigenteEn(politica.VigenteHasta) || politica.VigenteEn(politica.VigenteHasta.Add(time.Nanosecond)) {
		t.Fatal("politica expirada admitida en el instante de firma")
	}
}

func TestManifiestoProbatorioFallaCerradoAnteOmisionesReordenacionYAccionDesconocida(t *testing.T) {
	contenido := contenidoDecisionValidoPrueba(t)
	manifiesto := manifiestoProbatorioValidoPrueba(t, contenido)
	clon := manifiesto.Clonar()
	clon.Autorizaciones[0].Secuencia = 2
	if err := clon.Validar(); err == nil {
		t.Fatal("manifiesto reordenado admitido")
	}
	clon = manifiesto.Clonar()
	clon.Autorizaciones[0].Accion = AccionOperacionBaremacion("accion_desconocida")
	if err := clon.Validar(); err == nil {
		t.Fatal("accion desconocida admitida en manifiesto")
	}
	clon = manifiesto.Clonar()
	clon.Evidencias = nil
	if err := clon.Validar(); err == nil {
		t.Fatal("manifiesto sin evidencias admitido")
	}
	clon = manifiesto.Clonar()
	clon.Autorizaciones[0].AutorizacionRef = "alterada"
	if manifiesto.Autorizaciones[0].AutorizacionRef == "alterada" {
		t.Fatal("clon del manifiesto comparte autorizaciones")
	}
}

func TestManifiestoProbatorioMatrizRechazaOmitirCadaAutorizacionYEvidencia(t *testing.T) {
	contenido := contenidoDecisionValidoPrueba(t)
	base := manifiestoProbatorioValidoPrueba(t, contenido)
	for indice := range base.Autorizaciones {
		indice := indice
		t.Run(fmt.Sprintf("sin_autorizacion_%02d_%s", indice+1, base.Autorizaciones[indice].Accion), func(t *testing.T) {
			adulterado := base.Clonar()
			adulterado.Autorizaciones = append(
				adulterado.Autorizaciones[:indice:indice], adulterado.Autorizaciones[indice+1:]...,
			)
			resecuenciarManifiestoPrueba(&adulterado)
			if _, err := resellarManifiestoPrueba(adulterado); err == nil {
				t.Fatal("la omision individual de una autorizacion admitio un manifiesto resellado")
			}
		})
	}
	for indice := range base.Evidencias {
		indice := indice
		t.Run(fmt.Sprintf("sin_evidencia_%02d_%s", indice+1, base.Evidencias[indice].Tipo), func(t *testing.T) {
			adulterado := base.Clonar()
			adulterado.Evidencias = append(
				adulterado.Evidencias[:indice:indice], adulterado.Evidencias[indice+1:]...,
			)
			resecuenciarManifiestoPrueba(&adulterado)
			if _, err := resellarManifiestoPrueba(adulterado); err == nil {
				t.Fatal("la omision individual de una evidencia admitio un manifiesto resellado")
			}
		})
	}
}

func TestManifiestoProbatorioMatrizRechazaExtrasOrdenYReferenciasCruzadas(t *testing.T) {
	contenido := contenidoDecisionValidoPrueba(t)
	base := manifiestoProbatorioValidoPrueba(t, contenido)
	version := ReferenciaVersionBaremacion{
		BaremacionMeritoRef: base.BaremacionMeritoRef,
		Numero:              base.VersionBase,
		HuellaEstadoSHA256:  base.HuellaVersionBaseSHA256,
	}
	if err := base.ValidarPara(version, contenido); err != nil {
		t.Fatalf("base de la matriz no valida: %v", err)
	}

	casosEstructurales := []struct {
		nombre string
		mutar  func(*ManifiestoProbatorioBaremacion)
	}{
		{
			nombre: "autorizacion extra conocida",
			mutar: func(m *ManifiestoProbatorioBaremacion) {
				m.Autorizaciones = append(m.Autorizaciones, AutorizacionProbatoriaBaremacion{
					Accion: AccionConfirmarDecisionBaremacion, ClaseRecurso: ClaseRecursoBaremacion,
					RecursoRef: m.BaremacionMeritoRef, AutorizacionRef: "autorizacion-extra-manifiesto",
				})
			},
		},
		{
			nombre: "evidencia extra conocida",
			mutar: func(m *ManifiestoProbatorioBaremacion) {
				m.Evidencias = append(m.Evidencias, EvidenciaProbatoriaBaremacion{
					Tipo: EvidenciaRetencionFirmadoBaremacion, Referencia: "evidencia-extra-manifiesto",
					HuellaEvidenciaSHA256: huellaPruebaPuertos("0"),
				})
			},
		},
		{
			nombre: "orden de autorizaciones alterado",
			mutar: func(m *ManifiestoProbatorioBaremacion) {
				m.Autorizaciones[0], m.Autorizaciones[1] = m.Autorizaciones[1], m.Autorizaciones[0]
			},
		},
		{
			nombre: "orden de evidencias alterado",
			mutar: func(m *ManifiestoProbatorioBaremacion) {
				documento := evidenciaManifiestoPorTipoPrueba(t, m, EvidenciaDocumentoMeritoBaremacion)
				representacion := evidenciaManifiestoPorTipoPrueba(t, m, EvidenciaRepresentacionBaremacion)
				*documento, *representacion = *representacion, *documento
			},
		},
		{
			nombre: "documento y representacion cruzados",
			mutar: func(m *ManifiestoProbatorioBaremacion) {
				m.Autorizaciones[3].RecursoRef, m.Autorizaciones[4].RecursoRef =
					m.Autorizaciones[4].RecursoRef, m.Autorizaciones[3].RecursoRef
			},
		},
	}
	for _, caso := range casosEstructurales {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			adulterado := base.Clonar()
			caso.mutar(&adulterado)
			resecuenciarManifiestoPrueba(&adulterado)
			if sellado, err := resellarManifiestoPrueba(adulterado); err == nil &&
				sellado.ValidarPara(version, contenido) == nil {
				t.Fatal("la alteracion estructural admitio cobertura valida")
			}
		})
	}

	casosConcretos := []struct {
		nombre string
		mutar  func(*ManifiestoProbatorioBaremacion)
	}{
		{
			nombre: "calculo de otra operacion",
			mutar:  func(m *ManifiestoProbatorioBaremacion) { m.Autorizaciones[1].RecursoRef = "calculo-ajeno-001" },
		},
		{
			nombre: "autorizacion de otra adopcion",
			mutar: func(m *ManifiestoProbatorioBaremacion) {
				m.Autorizaciones[5].AutorizacionRef = "autorizacion-adopcion-ajena"
			},
		},
		{
			nombre: "huella de otro criterio",
			mutar: func(m *ManifiestoProbatorioBaremacion) {
				evidenciaManifiestoPorTipoPrueba(t, m, EvidenciaCriterioPublicadoBaremacion).HuellaEvidenciaSHA256 = huellaPruebaPuertos("0")
			},
		},
		{
			nombre: "huella de otro contenido",
			mutar: func(m *ManifiestoProbatorioBaremacion) {
				evidenciaManifiestoPorTipoPrueba(t, m, EvidenciaContenidoDecisionBaremacion).HuellaEvidenciaSHA256 = huellaPruebaPuertos("0")
			},
		},
	}
	for _, caso := range casosConcretos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			adulterado := base.Clonar()
			caso.mutar(&adulterado)
			sellado, err := resellarManifiestoPrueba(adulterado)
			if err != nil {
				t.Fatalf("la alteracion concreta debia conservar una estructura resellable: %v", err)
			}
			if err := sellado.ValidarPara(version, contenido); err == nil {
				t.Fatal("la referencia o huella cruzada se acepto para otra decision")
			}
		})
	}
}

func TestManifiestoProbatorioMatrizLigaCadaCapaConLaFirmaConcreta(t *testing.T) {
	contenido := contenidoDecisionValidoPrueba(t)
	politica := politicaFirmaValidaPrueba()
	artefacto := artefactoFirmaValidoPrueba(t, contenido, politica, "inicial")
	validacionInicial := validacionFirmaValidaPrueba(artefacto, instantePuertosPrueba.Add(6*time.Minute), "inicial")
	sello := selloValidoPrueba(artefacto, politica)
	validacionTrasSello := validacionFirmaValidaPrueba(
		sello.ArtefactoSellado, instantePuertosPrueba.Add(7*time.Minute), "sellado", PerfilFirmaPAdESBaselineT,
	)
	artefactoFinal := artefactoFirmaValidoPrueba(t, contenido, politica, "longevo")
	artefactoFinal.FirmaRef = artefacto.FirmaRef
	artefactoFinal.HuellaFirmaSHA256 = artefacto.HuellaFirmaSHA256
	artefactoFinal.SesionFirmaRef = artefacto.SesionFirmaRef
	artefactoFinal.EvidenciaFirmaInteractivaRef = artefacto.EvidenciaFirmaInteractivaRef
	artefactoFinal.HuellaEvidenciaInteractivaSHA256 = artefacto.HuellaEvidenciaInteractivaSHA256
	artefactoFinal.FirmadaEn = artefacto.FirmadaEn
	aumento := ResultadoAumentoFirma{
		ArtefactoOrigen: sello.ArtefactoSellado, Artefacto: artefactoFinal,
		NivelAlcanzadoClave:   politica.NivelAumentoClave,
		PoliticaLongevidadRef: politica.PoliticaLongevidadRef, PoliticaLongevidadVersion: politica.PoliticaLongevidadVersion,
		HuellaPoliticaLongevidadSHA256: politica.HuellaPoliticaLongevidadSHA256,
		EvidenciaAumentoRef:            "evidencia-aumento-1", HuellaEvidenciaSHA256: huellaPruebaPuertos("6"),
		AumentadaEn: instantePuertosPrueba.Add(8 * time.Minute),
	}
	validacionFinal := validacionFirmaValidaPrueba(
		artefactoFinal, instantePuertosPrueba.Add(9*time.Minute), "final", PerfilFirmaPAdESBaselineLTA,
	)
	documento := documentoFirmadoCustodiadoPrueba(artefactoFinal)
	base := construirManifiestoProbatorioPrueba(
		t, contenido, politica, artefacto, validacionInicial, &sello, &validacionTrasSello,
		&aumento, validacionFinal, documento,
	)
	if _, err := ConstituirFirmaDecisionConfiable(
		contenido, politica, artefacto, validacionInicial, &sello, &validacionTrasSello,
		&aumento, validacionFinal, documento, base,
	); err != nil {
		t.Fatalf("firma base de la matriz: %v", err)
	}
	casos := []struct {
		nombre string
		mutar  func(*ManifiestoProbatorioBaremacion)
	}{
		{"politica ajena", func(m *ManifiestoProbatorioBaremacion) { m.Autorizaciones[6].RecursoRef = "politica-ajena-001" }},
		{"aprobacion de politica ajena", func(m *ManifiestoProbatorioBaremacion) {
			evidencia := evidenciaManifiestoPorTipoPrueba(t, m, EvidenciaPoliticaFirmaBaremacion)
			evidencia.Referencia = "aprobacion-politica-ajena"
			evidencia.HuellaEvidenciaSHA256 = huellaPruebaPuertos("0")
		}},
		{"sesion ajena", func(m *ManifiestoProbatorioBaremacion) { m.Autorizaciones[10].RecursoRef = "sesion-ajena-001" }},
		{"documento canonico ajeno", func(m *ManifiestoProbatorioBaremacion) {
			evidenciaManifiestoPorTipoPrueba(t, m, EvidenciaDocumentoCanonicoBaremacion).HuellaEvidenciaSHA256 = huellaPruebaPuertos("0")
			evidenciaManifiestoPorTipoPrueba(t, m, EvidenciaCustodiaFirmableBaremacion).HuellaEvidenciaSHA256 = huellaPruebaPuertos("0")
		}},
		{"validacion inicial ajena", func(m *ManifiestoProbatorioBaremacion) {
			evidenciaManifiestoPorTipoPrueba(t, m, EvidenciaValidacionInicialBaremacion).Referencia = "validacion-inicial-ajena"
		}},
		{"sello ajeno", func(m *ManifiestoProbatorioBaremacion) {
			evidenciaManifiestoPorTipoPrueba(t, m, EvidenciaSelloTiempoBaremacion).Referencia = "sello-tiempo-ajeno"
		}},
		{"aumento ajeno", func(m *ManifiestoProbatorioBaremacion) {
			evidenciaManifiestoPorTipoPrueba(t, m, EvidenciaAumentoLongevidadBaremacion).Referencia = "aumento-ajeno"
		}},
		{"documento firmado ajeno", func(m *ManifiestoProbatorioBaremacion) {
			for _, indice := range []int{16, 17, 18} {
				m.Autorizaciones[indice].RecursoRef = "documento-firmado-ajeno"
			}
		}},
		{"recuperacion ajena", func(m *ManifiestoProbatorioBaremacion) {
			evidenciaManifiestoPorTipoPrueba(t, m, EvidenciaRecuperacionFirmadoBaremacion).Referencia = "recuperacion-ajena"
		}},
		{"custodia ajena", func(m *ManifiestoProbatorioBaremacion) {
			evidenciaManifiestoPorTipoPrueba(t, m, EvidenciaCustodiaFirmadoBaremacion).Referencia = "custodia-ajena"
		}},
		{"retencion ajena", func(m *ManifiestoProbatorioBaremacion) {
			evidenciaManifiestoPorTipoPrueba(t, m, EvidenciaRetencionFirmadoBaremacion).Referencia = "retencion-ajena"
		}},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			adulterado := base.Clonar()
			caso.mutar(&adulterado)
			sellado, err := resellarManifiestoPrueba(adulterado)
			if err != nil {
				t.Fatalf("estructura de la capa adulterada: %v", err)
			}
			if _, err := ConstituirFirmaDecisionConfiable(
				contenido, politica, artefacto, validacionInicial, &sello, &validacionTrasSello, &aumento,
				validacionFinal, documento, sellado,
			); err == nil {
				t.Fatal("la capa de otra firma se acepto como cobertura de la decision")
			}
		})
	}
}

func TestEnsambladorConservaPoliticasSelloLongevidadYValidacionFinal(t *testing.T) {
	contenido := contenidoDecisionValidoPrueba(t)
	politica := politicaFirmaValidaPrueba()
	artefacto := artefactoFirmaValidoPrueba(t, contenido, politica, "inicial")
	validacionInicial := validacionFirmaValidaPrueba(artefacto, instantePuertosPrueba.Add(6*time.Minute), "inicial")
	sello := selloValidoPrueba(artefacto, politica)
	validacionTrasSello := validacionFirmaValidaPrueba(
		sello.ArtefactoSellado, instantePuertosPrueba.Add(7*time.Minute), "sellado", PerfilFirmaPAdESBaselineT,
	)
	artefactoFinal := artefactoFirmaValidoPrueba(t, contenido, politica, "longevo")
	artefactoFinal.FirmaRef = artefacto.FirmaRef
	artefactoFinal.HuellaFirmaSHA256 = artefacto.HuellaFirmaSHA256
	artefactoFinal.SesionFirmaRef = artefacto.SesionFirmaRef
	artefactoFinal.EvidenciaFirmaInteractivaRef = artefacto.EvidenciaFirmaInteractivaRef
	artefactoFinal.HuellaEvidenciaInteractivaSHA256 = artefacto.HuellaEvidenciaInteractivaSHA256
	artefactoFinal.FirmadaEn = artefacto.FirmadaEn
	aumento := ResultadoAumentoFirma{
		ArtefactoOrigen: sello.ArtefactoSellado, Artefacto: artefactoFinal,
		NivelAlcanzadoClave:            politica.NivelAumentoClave,
		PoliticaLongevidadRef:          politica.PoliticaLongevidadRef,
		PoliticaLongevidadVersion:      politica.PoliticaLongevidadVersion,
		HuellaPoliticaLongevidadSHA256: politica.HuellaPoliticaLongevidadSHA256,
		EvidenciaAumentoRef:            "evidencia-aumento-1", HuellaEvidenciaSHA256: huellaPruebaPuertos("6"),
		AumentadaEn: instantePuertosPrueba.Add(8 * time.Minute),
	}
	validacionFinal := validacionFirmaValidaPrueba(
		artefactoFinal, instantePuertosPrueba.Add(9*time.Minute), "final", PerfilFirmaPAdESBaselineLTA,
	)
	custodiaFinal := documentoFirmadoCustodiadoPrueba(artefactoFinal)
	manifiesto := manifiestoProbatorioValidoPrueba(t, contenido)
	firma, err := ConstituirFirmaDecisionConfiable(
		contenido, politica, artefacto, validacionInicial, &sello, &validacionTrasSello,
		&aumento, validacionFinal, custodiaFinal, manifiesto,
	)
	if err != nil {
		t.Fatalf("ensamblar firma: %v", err)
	}
	decision, err := dominiobolsa.ConstituirDecisionFirmada(contenido, firma)
	if err != nil || decision.Validar() != nil {
		t.Fatalf("firma ensamblada no constituye decision: %v", err)
	}
	if firma.PoliticaFirmaVersion != politica.Version ||
		firma.HuellaPoliticaSelloTiempoSHA256 != politica.HuellaPoliticaSelloTiempoSHA256 ||
		firma.HuellaPoliticaLongevidadSHA256 != politica.HuellaPoliticaLongevidadSHA256 ||
		firma.ValidacionFirmaRef != validacionFinal.ValidacionRef ||
		firma.DocumentoFirmadoRef != artefactoFinal.DocumentoFirmadoRef {
		t.Fatalf("no se conservaron capas exactas: %+v", firma)
	}
	validacionFinal.Estado = EstadoValidacionFirmaIndeterminada
	if _, err := ConstituirFirmaDecisionConfiable(
		contenido, politica, artefacto, validacionInicial, &sello, &validacionTrasSello,
		&aumento, validacionFinal, custodiaFinal, manifiesto,
	); !errors.Is(err, ErrFirmaServidorNoValida) {
		t.Fatalf("aumento no revalidado admitido: %v", err)
	}
}

func TestRecuperacionHistoricaExigeReferenciaYHuellaExactas(t *testing.T) {
	contenido := contenidoDecisionValidoPrueba(t)
	politica := politicaFirmaValidaPrueba()
	artefacto := artefactoFirmaValidoPrueba(t, contenido, politica, "inicial")
	validacion := validacionFirmaValidaPrueba(artefacto, instantePuertosPrueba.Add(6*time.Minute), "inicial")
	sello := selloValidoPrueba(artefacto, politica)
	artefactoLongevo := sello.ArtefactoSellado
	artefactoLongevo.DocumentoFirmadoRef += ":pades-lta"
	artefactoLongevo.HuellaDocumentoSHA256 = huellaPruebaPuertos("f")
	aumento := ResultadoAumentoFirma{
		ArtefactoOrigen: sello.ArtefactoSellado, Artefacto: artefactoLongevo,
		NivelAlcanzadoClave:            politica.NivelAumentoClave,
		PoliticaLongevidadRef:          politica.PoliticaLongevidadRef,
		PoliticaLongevidadVersion:      politica.PoliticaLongevidadVersion,
		HuellaPoliticaLongevidadSHA256: politica.HuellaPoliticaLongevidadSHA256,
		EvidenciaAumentoRef:            "evidencia-aumento-1", HuellaEvidenciaSHA256: huellaPruebaPuertos("6"),
		AumentadaEn: instantePuertosPrueba.Add(8 * time.Minute),
	}
	if err := artefacto.ValidarRecuperacion(SolicitudRecuperarArtefactoFirma{
		Contexto: contextoFirmaValido(AccionRecuperarArtefactoFirmaBaremacion, artefacto.FirmaRef),
		FirmaRef: artefacto.FirmaRef, HuellaFirmaSHA256: artefacto.HuellaFirmaSHA256,
		DocumentoFirmadoRef: artefacto.DocumentoFirmadoRef, HuellaDocumentoSHA256: artefacto.HuellaDocumentoSHA256,
	}); err != nil {
		t.Fatalf("recuperar artefacto: %v", err)
	}
	if err := validacion.ValidarRecuperacion(SolicitudRecuperarValidacionFirma{
		Contexto: contextoFirmaValido(AccionRecuperarValidacionFirmaBaremacion, validacion.ValidacionRef), ValidacionRef: validacion.ValidacionRef,
		HuellaValidacionSHA256: validacion.HuellaValidacionSHA256,
	}); err != nil {
		t.Fatalf("recuperar validacion: %v", err)
	}
	if err := sello.ValidarRecuperacion(SolicitudRecuperarSelloTiempo{
		Contexto:       contextoFirmaValido(AccionRecuperarSelloTiempoFirmaBaremacion, sello.SelloTiempoRef),
		SelloTiempoRef: sello.SelloTiempoRef, HuellaSelloSHA256: sello.HuellaSelloTiempoSHA256,
	}); err != nil {
		t.Fatalf("recuperar sello: %v", err)
	}
	if err := aumento.ValidarRecuperacion(SolicitudRecuperarAumentoFirma{
		Contexto: contextoFirmaValido(AccionRecuperarAumentoFirmaBaremacion, aumento.EvidenciaAumentoRef), EvidenciaAumentoRef: aumento.EvidenciaAumentoRef,
		HuellaAumentoSHA256: aumento.HuellaEvidenciaSHA256,
	}); err != nil {
		t.Fatalf("recuperar aumento: %v", err)
	}
	solicitudErronea := SolicitudRecuperarValidacionFirma{
		Contexto:      contextoFirmaValido(AccionRecuperarValidacionFirmaBaremacion, validacion.ValidacionRef),
		ValidacionRef: validacion.ValidacionRef, HuellaValidacionSHA256: huellaPruebaPuertos("0"),
	}
	if !errors.Is(validacion.ValidarRecuperacion(solicitudErronea), ErrEvidenciaFirmaNoEncontrada) {
		t.Fatal("recuperacion acepto una huella historica distinta")
	}
}

func TestEstadosTecnicosSonCerrados(t *testing.T) {
	for _, estado := range []EstadoSesionFirmaInteractiva{
		EstadoSesionFirmaPreparada, EstadoSesionFirmaPendiente, EstadoSesionFirmaCompletada,
		EstadoSesionFirmaRechazada, EstadoSesionFirmaCancelada, EstadoSesionFirmaExpirada, EstadoSesionFirmaFallida,
	} {
		if !estado.Valido() {
			t.Fatalf("estado de sesion rechazado: %q", estado)
		}
	}
	if EstadoSesionFirmaInteractiva("proveedor_x").Valido() || EstadoValidacionFirma("aceptable").Valido() ||
		EstadoComprobacionFirma("casi").Valido() || ClaseCambioBaremacion("sobrescribir").Valida() {
		t.Fatal("estado abierto o dependiente de proveedor admitido")
	}
}

func contextoOperacionValido(accion AccionOperacionBaremacion, recursoRef string) ContextoOperacionBaremacion {
	contexto, err := nuevaAutorizacionPrueba(decisionAutorizacionValida(accion, recursoRef))
	if err != nil {
		panic(err)
	}
	return contexto
}

func contextoOperacionAlmacenBaremacionValido(
	accion AccionOperacionBaremacion,
	recursoRef string,
	vinculos puertosvec.VinculosOperacionAlmacen,
) ContextoOperacionBaremacion {
	decision, recurso := decisionYRecursoAlmacenBaremacionValidos(accion, recursoRef, vinculos)
	contexto, err := NuevaAutorizacionOperacionAlmacenBaremacion(
		decision, recurso, vinculoBaremacionPuertosPrueba(decision, "sujeto-001"), instantePuertosPrueba,
	)
	if err != nil {
		panic(err)
	}
	return contexto
}

func vinculosAlmacenBaremacionPrueba(
	accion AccionOperacionBaremacion,
) puertosvec.VinculosOperacionAlmacen {
	vinculos := puertosvec.VinculosOperacionAlmacen{
		OperacionRef:        "operacion:almacen:baremacion:001",
		CargaRef:            "carga:almacen:baremacion:001",
		Clasificacion:       "datos_personales_alta",
		SujetoSeudonimoHMAC: "hmac-sha256:sujeto_1:" + huellaPruebaPuertos("1"),
		HuellaSolicitudHMAC: "hmac-sha256:almacen_1:" + huellaPruebaPuertos("2"),
		EfectoRef:           "efecto:almacen:baremacion:001",
	}
	if accion == AccionRetenerDocumentoFirmadoBaremacion {
		vinculos.ObjetoVinculado = puertosvec.ReferenciaObjetoAlmacen{
			Referencia: "objeto:firmado:001", Version: "version-firmada-001",
		}
	}
	return vinculos
}

func crearContextoAlmacenBaremacionPrueba(
	contexto ContextoOperacionBaremacion,
	accion AccionOperacionBaremacion,
	vinculos puertosvec.VinculosOperacionAlmacen,
) (puertosvec.ContextoOperacionAlmacen, error) {
	switch accion {
	case AccionCustodiarDecisionBaremacion:
		return contexto.CrearContextoAlmacenCustodiarDecision(vinculos)
	case AccionCustodiarDocumentoFirmadoBaremacion:
		return contexto.CrearContextoAlmacenCustodiarDocumentoFirmado(vinculos)
	case AccionRetenerDocumentoFirmadoBaremacion:
		return contexto.CrearContextoAlmacenRetenerDocumentoFirmado(vinculos)
	default:
		return puertosvec.ContextoOperacionAlmacen{}, ErrAutorizacionBaremacionInvalida
	}
}

func decisionYRecursoAlmacenBaremacionValidos(
	accion AccionOperacionBaremacion,
	recursoRef string,
	vinculos puertosvec.VinculosOperacionAlmacen,
) (dominiovec.DecisionAutorizacion, dominiovec.RecursoAutorizable) {
	decision := decisionAutorizacionValida(accion, recursoRef)
	decision.ValidaHasta = instantePuertosPrueba.Add(VentanaMaximaUsoAutorizacionBaremacion)
	atributos := map[string]string{
		puertosvec.AtributoAlmacenOperacionRef:        vinculos.OperacionRef,
		puertosvec.AtributoAlmacenCargaRef:            vinculos.CargaRef,
		puertosvec.AtributoAlmacenClasificacion:       vinculos.Clasificacion,
		puertosvec.AtributoAlmacenSujetoSeudonimoHMAC: vinculos.SujetoSeudonimoHMAC,
		puertosvec.AtributoAlmacenHuellaSolicitudHMAC: vinculos.HuellaSolicitudHMAC,
		puertosvec.AtributoAlmacenEfectoRef:           vinculos.EfectoRef,
	}
	if accion == AccionRetenerDocumentoFirmadoBaremacion {
		atributos[puertosvec.AtributoAlmacenObjetoRef] = vinculos.ObjetoVinculado.Referencia
		atributos[puertosvec.AtributoAlmacenObjetoVersion] = vinculos.ObjetoVinculado.Version
	}
	recurso := dominiovec.RecursoAutorizable{
		Referencia: recursoRef, ModuloID: "bolsa", Tipo: string(especificacionesAccionBaremacion[accion].clase),
		Ambitos: map[string]string{"sujeto_ref": "sujeto-001"}, Atributos: atributos,
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		panic(err)
	}
	decision.ContextoRecursoHuellaSHA256 = huella
	return decision, recurso
}

func contextoFirmaValido(accion AccionOperacionBaremacion, recursoRef string) ContextoOperacionFirma {
	return ContextoOperacionFirma{
		ContextoOperacionBaremacion: contextoOperacionValido(accion, recursoRef), OperacionRef: "operacion-firma-1",
	}
}

func decisionAutorizacionValida(accion AccionOperacionBaremacion, recursoRef string) dominiovec.DecisionAutorizacion {
	campos, existe := CamposRequeridosOperacionBaremacion(accion)
	if !existe {
		panic("accion de prueba desconocida")
	}
	especificacion := especificacionesAccionBaremacion[accion]
	recurso := dominiovec.RecursoAutorizable{
		Referencia: recursoRef, ModuloID: "bolsa", Tipo: string(especificacion.clase),
		Ambitos: map[string]string{"sujeto_ref": "sujeto-001"},
	}
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		panic(err)
	}
	huellaCatalogo, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		panic(err)
	}
	vinculo, err := pruebasvec.NuevoVinculoGenerico(instantePuertosPrueba)
	if err != nil {
		panic(err)
	}
	return dominiovec.DecisionAutorizacion{
		DecisionRef: referenciaAutorizacionPrueba(accion), Concedida: true, Codigo: "concedida",
		PrincipalID: principalBaremacionPuertosPrueba, PerfilActivoRef: perfilBaremacionPuertosPrueba,
		Accion: string(accion), RecursoRef: recursoRef, Finalidad: "baremacion_proceso_selectivo",
		ModuloID: "bolsa", TipoRecurso: string(especificacion.clase), ContextoRecursoHuellaSHA256: huellaContexto,
		CorrelacionRef: "correlacion-1", AsignacionRef: "asignacion-tecnico-v1",
		VinculoAutenticacionActor: vinculo,
		AsignacionHuellaSHA256:    huellaPruebaPuertos("1"), VersionRolRef: "rol-tecnico-v1",
		VersionRolHuellaSHA256: huellaPruebaPuertos("2"), GarantiaMinima: dominiovec.AuthAssuranceHigh,
		ControlVigenciaVersionRolRef:      "rol-tecnico-v1",
		ControlVigenciaVersionRolRevision: 1, ControlVigenciaVersionRolHuellaSHA256: huellaPruebaPuertos("3"),
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasHuellasSHA256: map[string]string{},
		CamposPermitidos:                campos, EmitidaEn: instantePuertosPrueba.Add(-time.Minute),
		ValidaHasta: instantePuertosPrueba.Add(4 * time.Minute),
	}
}

func referenciaAutorizacionPrueba(accion AccionOperacionBaremacion) string {
	return "autorizacion-" + strings.ReplaceAll(string(accion), ".", "-")
}

func nuevaAutorizacionPrueba(decision dominiovec.DecisionAutorizacion) (ContextoOperacionBaremacion, error) {
	return NuevaAutorizacionOperacionBaremacion(
		decision, vinculoBaremacionPuertosPrueba(decision, "sujeto-001"), instantePuertosPrueba,
	)
}

func vinculoBaremacionPuertosPrueba(
	decision dominiovec.DecisionAutorizacion,
	sujetoRef string,
) VinculoAutenticacionBaremacion {
	datos, err := decision.VinculoAutenticacionActor.Datos()
	if err != nil {
		panic(err)
	}
	return VinculoAutenticacionBaremacion{
		SujetoRef: sujetoRef, Metodo: datos.MetodoObservado, Garantia: datos.GarantiaObservada,
		AutenticacionRef: datos.AutenticacionRef, SesionRef: datos.SesionRef,
		SesionEmitidaEn: datos.SesionEmitidaEn, SesionValidaHasta: datos.SesionValidaHasta,
		VinculoAutenticacionActor: decision.VinculoAutenticacionActor,
	}
}

func solicitudReservaAltaValida() SolicitudReservarCambioBaremacion {
	return SolicitudReservarCambioBaremacion{
		Contexto: contextoOperacionValido(AccionReservarAltaBaremacion, "baremacion-1"), Clase: ClaseCambioAltaBaremacion, ClaveIdempotencia: "alta-baremacion-1",
		BaremacionMeritoRef: "baremacion-1", HuellaSolicitudHMAC: "hmac-sha256:reserva_1:" + huellaPruebaPuertos("a"),
		SolicitadaEn: instantePuertosPrueba, ExpiraEn: instantePuertosPrueba.Add(5 * time.Minute),
	}
}

func criterioValidoPrueba() dominiobolsa.ReferenciaCriterio {
	return dominiobolsa.ReferenciaCriterio{
		ProcesoRef: "proceso-selectivo-2026-017", Clave: "experiencia.entidad_publica.grupo_c1", Version: 7,
		HuellaSHA256: huellaPruebaPuertos("a"), PuntosMaximos: 10 * dominiobolsa.UnidadesPorPunto,
		ReglaCalculo: dominiobolsa.ReferenciaReglaCalculo{
			Clave: "experiencia_publica_dias", Version: 3, HuellaSHA256: huellaPruebaPuertos("9"),
		},
	}
}

func evidenciaConfiableValidaPrueba(t *testing.T) EvidenciaBaremacionConfiable {
	t.Helper()
	documento := dominiovec.DocumentoLogico{
		ID: "documento-001", Version: 1, Revision: 1,
		Plantilla: dominiovec.ReferenciaPlantillaDocumento{ID: "plantilla-baremo", Version: 7, HuellaSHA256: huellaPruebaPuertos("1")},
		ModuloID:  "bolsa", TipoDocumental: "merito", Clasificacion: "datos_personales_alta",
		Relaciones: []dominiovec.RelacionDocumento{
			{Tipo: dominiovec.TipoRelacionPersona, Referencia: "sujeto-001", Rol: "interesada"},
			{Tipo: dominiovec.TipoRelacionExpediente, Referencia: "solicitud-001", Rol: "principal"},
		},
		Estado:           dominiovec.EstadoDocumentoLogicoBorrador,
		HuellaDatosHMAC:  "hmac-sha256:documentos_1:" + huellaPruebaPuertos("2"),
		HuellaFuenteHMAC: "hmac-sha256:documentos_1:" + huellaPruebaPuertos("3"),
		CreadoPor:        "persona-interna-tecnica-17", CreadoEn: instantePuertosPrueba.Add(-time.Hour),
		CorrelacionRef: "correlacion-1", Motivo: "Evidencia aportada para baremacion.",
		ENI: dominiovec.MetadatosENI{
			Identificador: "documento-001", Organo: "DIPUTACION-GRANADA", Origen: "ciudadano",
			EstadoElaboracion: "original", TipoDocumental: "merito", FechaCaptura: instantePuertosPrueba.Add(-time.Hour),
		},
	}
	if err := documento.Validar(); err != nil {
		t.Fatalf("documento logico de prueba: %v", err)
	}
	evidencia := EvidenciaBaremacionConfiable{
		Evidencia: dominiobolsa.EvidenciaMerito{Referencia: dominiobolsa.ReferenciaEvidencia{
			DocumentoRef: "documento-001", VersionDocumento: 1, RepresentacionRef: "representacion-001",
			HuellaSHA256: huellaPruebaPuertos("b"),
		}},
		Documento: documento, VerificacionPertenenciaRef: "verificacion-pertenencia-1",
		HuellaVerificacionSHA256: huellaPruebaPuertos("4"), VerificadaEn: instantePuertosPrueba.Add(-time.Minute),
	}
	if err := evidencia.Validar(); err != nil {
		t.Fatalf("evidencia confiable de prueba: %v", err)
	}
	return evidencia
}

func calculoValidoPrueba(puntos dominiobolsa.Puntos, sufijo string) dominiobolsa.CalculoOficialBaremacion {
	criterio := criterioValidoPrueba()
	return dominiobolsa.CalculoOficialBaremacion{
		CalculoRef: "calculo-oficial-" + sufijo, ProcesoRef: criterio.ProcesoRef, SolicitudRef: "solicitud-001",
		SujetoRef: "sujeto-001", BaremacionMeritoRef: "baremacion-001", Criterio: criterio,
		Regla: criterio.ReglaCalculo, Evidencias: []dominiobolsa.EvidenciaMerito{{Referencia: dominiobolsa.ReferenciaEvidencia{
			DocumentoRef: "documento-001", VersionDocumento: 1, RepresentacionRef: "representacion-001",
			HuellaSHA256: huellaPruebaPuertos("b"),
		}}}, EntradaRef: "entrada-calculo-" + sufijo,
		HuellaEntradaSHA256: huellaPruebaPuertos("b"), PuntosCalculados: puntos,
		DesgloseRef: "desglose-calculo-" + sufijo, HuellaDesgloseSHA256: huellaPruebaPuertos("c"),
		ResultadoRef: "resultado-calculo-" + sufijo, HuellaResultadoSHA256: huellaPruebaPuertos("d"),
		MotorCalculoRef: "motor-baremo-oficial", VersionMotorCalculo: "motor-v2.1.0",
		EvidenciaEjecucionRef: "ejecucion-calculo-" + sufijo, HuellaEjecucionSHA256: huellaPruebaPuertos("e"),
		CalculadoEn: instantePuertosPrueba.Add(-time.Minute),
	}
}

func baremacionValidaPrueba(t *testing.T) dominiobolsa.BaremacionMerito {
	t.Helper()
	baremacion, err := dominiobolsa.NuevaBaremacionMerito(dominiobolsa.AltaMeritoBaremable{
		ID: "baremacion-001", ProcesoRef: "proceso-selectivo-2026-017", SolicitudRef: "solicitud-001",
		SujetoRef: "sujeto-001", Criterio: criterioValidoPrueba(),
		EvidenciasIniciales: []dominiobolsa.EvidenciaMerito{{Referencia: dominiobolsa.ReferenciaEvidencia{
			DocumentoRef: "documento-001", VersionDocumento: 1, RepresentacionRef: "representacion-001",
			HuellaSHA256: huellaPruebaPuertos("b"),
		}}},
		PuntosDeclarados: 5_000_000, CalculoOficial: calculoValidoPrueba(4_250_000, "inicial"),
		CreadaEn: instantePuertosPrueba,
	})
	if err != nil {
		t.Fatalf("crear baremacion: %v", err)
	}
	return baremacion
}

func contenidoDecisionValidoPrueba(t *testing.T) dominiobolsa.ContenidoDecisionTecnica {
	t.Helper()
	baremacion := baremacionValidaPrueba(t)
	autorizacionDecision := contextoOperacionValido(
		AccionAdoptarDecisionInicialBaremacion, baremacion.ID,
	).Proyeccion()
	contenido, err := baremacion.PrepararDecisionInicial(dominiobolsa.PropuestaDecisionTecnica{
		ID: "decision-001", CalculoOficial: calculoValidoPrueba(4_250_000, "inicial"), PuntosReconocidos: 4_000_000,
		Resultado: dominiobolsa.ResultadoAceptado, DecisorRef: principalBaremacionPuertosPrueba,
		PerfilDecisorClave: perfilBaremacionPuertosPrueba, ValoracionesEvidencia: []dominiobolsa.ValoracionEvidencia{{
			Evidencia: baremacion.EvidenciasIniciales[0], Estado: dominiobolsa.EstadoEvidenciaApta,
			ResultadoSubsanacion: dominiobolsa.ResultadoSubsanacionNoAplica,
			MotivoClave:          "documento_valido", Motivo: "Documento autentico y suficiente.",
		}},
		MotivoClave: "valoracion_inicial", Motivo: "Valoracion conforme al criterio publicado.",
		FuentesNormativasRefs: []string{"norma-baremo-v7"}, AutorizacionRef: autorizacionDecision.AutorizacionRef,
		FinalidadClave: "baremacion_proceso_selectivo", CorrelacionRef: "correlacion-1",
		DecididaEn: instantePuertosPrueba.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("preparar contenido: %v", err)
	}
	return contenido
}

func politicaFirmaValidaPrueba() PoliticaFirmaBaremacion {
	return PoliticaFirmaBaremacion{
		Referencia: "politica-firma-baremacion", Version: 4, HuellaSHA256: huellaPruebaPuertos("1"),
		FormatoFirmaClave: FormatoFirmaPDFCanonico, PerfilFirmaClave: PerfilFirmaPAdESBaselineLTA,
		AlgoritmoHuellaClave: AlgoritmoHuellaFirmaSHA256, ComprobacionesObligatorias: ComprobacionesFirmaObligatorias(),
		RequiereFirmaInteractiva: true, RequiereValidacionServidor: true, RequiereSelloTiempo: true,
		PoliticaSelloTiempoRef: "politica-sello-tsa", PoliticaSelloTiempoVersion: 2,
		HuellaPoliticaSelloTiempoSHA256: huellaPruebaPuertos("2"), RequiereAumentoLongevidad: true,
		NivelAumentoClave: "pades_lta", PoliticaLongevidadRef: "politica-longevidad-lta",
		PoliticaLongevidadVersion: 3, HuellaPoliticaLongevidadSHA256: huellaPruebaPuertos("3"),
		AprobacionRef: "aprobacion-politica-4", HuellaAprobacionSHA256: huellaPruebaPuertos("4"),
		VigenteDesde: instantePuertosPrueba.Add(-time.Hour), VigenteHasta: instantePuertosPrueba.Add(24 * time.Hour),
	}
}

func codificacionValidaPrueba(t *testing.T, contenido dominiobolsa.ContenidoDecisionTecnica, politica PoliticaFirmaBaremacion) CodificacionCanonicaDecision {
	t.Helper()
	carga, err := NuevaCargaProtegida([]byte("%PDF-contenido-canonico-decision"))
	if err != nil {
		t.Fatal(err)
	}
	huella, err := contenido.HuellaContenidoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	codificacion := CodificacionCanonicaDecision{
		Carga: carga, ProcesoRef: contenido.ProcesoRef, SolicitudRef: contenido.SolicitudRef,
		SujetoRef: contenido.SujetoRef, BaremacionMeritoRef: contenido.BaremacionMeritoRef,
		DecisionRef: contenido.ID, VersionBaremacion: contenido.VersionBaremacion,
		PrincipalRef: contenido.DecisorRef, PerfilActorClave: contenido.PerfilDecisorClave,
		AutorizacionDecisionRef:     contenido.AutorizacionRef,
		AutorizacionCodificacionRef: referenciaAutorizacionPrueba(AccionCodificarDecisionBaremacion),
		FinalidadClave:              contenido.FinalidadClave,
		CorrelacionRef:              contenido.CorrelacionRef,
		FormatoClave:                politica.FormatoFirmaClave, MIME: "application/pdf",
		HuellaContenidoSHA256: huella, HuellaDocumentoSHA256: huellaPruebaPuertos("5"),
		VersionCodificador: "codificador-pdf-v3.2.1",
	}
	solicitud := SolicitudCodificarDecisionCanonica{
		Contexto: contextoOperacionValido(AccionCodificarDecisionBaremacion, contenido.ID),
		AutorizacionDecision: contextoOperacionValido(
			AccionAdoptarDecisionInicialBaremacion, contenido.BaremacionMeritoRef,
		),
		Contenido: contenido, Politica: politica,
	}
	if err := codificacion.ValidarPara(solicitud); err != nil {
		t.Fatalf("codificacion valida: %v", err)
	}
	return codificacion
}

func resultadoAlmacenValidoPrueba(
	t *testing.T,
	solicitudCustodia SolicitudCustodiarDocumentoFirmable,
) puertosvec.ResultadoOperacionObjeto {
	t.Helper()
	c := solicitudCustodia.Codificacion
	solicitudEscritura, err := solicitudCustodia.PrepararEscritura()
	if err != nil {
		t.Fatalf("preparar escritura de prueba: %v", err)
	}
	contexto, err := solicitudEscritura.Contexto.Proyeccion()
	if err != nil {
		t.Fatalf("proyectar contexto de escritura: %v", err)
	}
	objeto := puertosvec.ReferenciaObjetoAlmacen{Referencia: "objeto-firmable-001", Version: "version-001"}
	almacenado := puertosvec.ObjetoAlmacenado{
		Objeto: objeto, ConectorID: "almacen-s3-local", Zona: puertosvec.ZonaAlmacenAdmitida, MIME: c.MIME,
		Tamano: int64(c.Carga.Tamano()), HuellaSHA256: c.HuellaDocumentoSHA256,
		EvidenciaCreacionRef: "evidencia-custodia-001", AlmacenadoEn: instantePuertosPrueba.Add(10 * time.Second),
	}
	evidencia := puertosvec.EvidenciaOperacionAlmacen{
		Referencia: "evidencia-custodia-001", ConectorID: almacenado.ConectorID,
		EsquemaContexto: contexto.Esquema, AccionNegocio: contexto.AccionNegocio,
		Accion: contexto.AccionTecnica, EfectoRef: contexto.EfectoRef,
		HuellaPlanEfectoSHA256: contexto.HuellaPlanEfectoSHA256, PasoRef: contexto.PasoRef,
		HuellaDecisionSHA256: contexto.HuellaDecisionSHA256,
		Objeto:               objeto, OperacionRef: contexto.OperacionRef, CorrelacionRef: contexto.CorrelacionRef,
		AutorizacionRef: contexto.AutorizacionRef, Finalidad: contexto.Finalidad,
		Clasificacion: contexto.Clasificacion, RealizadaEn: almacenado.AlmacenadoEn,
		CargaRef: contexto.CargaRef, SujetoSeudonimoHMAC: contexto.SujetoSeudonimoHMAC,
		RecursoRef: contexto.RecursoRef, ModuloID: contexto.ModuloID,
		HuellaSolicitudHMAC: contexto.HuellaSolicitudHMAC,
	}
	return puertosvec.ResultadoOperacionObjeto{Objeto: almacenado, Evidencia: evidencia}
}

func solicitudCustodiaValidaPrueba(c CodificacionCanonicaDecision) SolicitudCustodiarDocumentoFirmable {
	solicitud := SolicitudCustodiarDocumentoFirmable{
		OperacionRef:      "operacion-custodia-001",
		ClaveIdempotencia: "custodia-decision-001", CargaRef: "carga-decision-001",
		SujetoSeudonimoHMAC: "hmac-sha256:sujeto_1:" + huellaPruebaPuertos("1"),
		HuellaAlmacenHMAC:   "hmac-sha256:almacen_1:" + huellaPruebaPuertos("2"),
		EfectoRef:           "efecto:custodia:decision:001",
		ProcesoRef:          "proceso-selectivo-2026-017",
		SolicitudRef:        "solicitud-001", BaremacionMeritoRef: "baremacion-001", DecisionRef: "decision-001",
		ClasificacionClave: "datos_personales_alta", Codificacion: c,
	}
	vinculos := puertosvec.VinculosOperacionAlmacen{
		OperacionRef: solicitud.OperacionRef, CargaRef: solicitud.CargaRef,
		Clasificacion: solicitud.ClasificacionClave, SujetoSeudonimoHMAC: solicitud.SujetoSeudonimoHMAC,
		HuellaSolicitudHMAC: solicitud.HuellaAlmacenHMAC, EfectoRef: solicitud.EfectoRef,
	}
	solicitud.Contexto = contextoOperacionAlmacenBaremacionValido(
		AccionCustodiarDecisionBaremacion, solicitud.DecisionRef, vinculos,
	)
	return solicitud
}

func solicitudFirmaInteractivaValidaPrueba(custodia DocumentoFirmableCustodiado, politica PoliticaFirmaBaremacion) SolicitudPrepararFirmaInteractiva {
	return SolicitudPrepararFirmaInteractiva{
		Contexto: contextoFirmaValido(AccionPrepararFirmaDecisionBaremacion, "decision-001"), ClaveIdempotencia: "firma-decision-001",
		ProcesoRef: "proceso-selectivo-2026-017", SolicitudRef: "solicitud-001",
		BaremacionMeritoRef: "baremacion-001", DecisionRef: "decision-001", Documento: custodia,
		FirmanteRef: principalBaremacionPuertosPrueba, PerfilFirmanteClave: perfilBaremacionPuertosPrueba, Politica: politica,
		SolicitadaEn: instantePuertosPrueba.Add(3 * time.Minute), ExpiraEn: instantePuertosPrueba.Add(13 * time.Minute),
	}
}

func sesionFirmaValidaPrueba(t *testing.T, solicitud SolicitudPrepararFirmaInteractiva) SesionFirmaInteractiva {
	t.Helper()
	carga, err := NuevaCargaProtegida([]byte("autofirma://lanzamiento-opaco"))
	if err != nil {
		t.Fatal(err)
	}
	return SesionFirmaInteractiva{
		SesionRef: "sesion-firma-1", Estado: EstadoSesionFirmaPreparada, CargaLanzamiento: carga,
		MIMELanzamiento: "application/octet-stream", Documento: solicitud.Documento,
		PoliticaFirmaRef: solicitud.Politica.Referencia, PoliticaFirmaVersion: solicitud.Politica.Version,
		HuellaPoliticaSHA256: solicitud.Politica.HuellaSHA256, EvidenciaPreparacionRef: "evidencia-preparacion-1",
		HuellaEvidenciaSHA256: huellaPruebaPuertos("6"), PreparadaEn: solicitud.SolicitadaEn,
		ExpiraEn: solicitud.ExpiraEn,
	}
}

func manifiestoProbatorioValidoPrueba(t *testing.T, contenido dominiobolsa.ContenidoDecisionTecnica) ManifiestoProbatorioBaremacion {
	t.Helper()
	politica := politicaFirmaValidaPrueba()
	artefacto := artefactoFirmaValidoPrueba(t, contenido, politica, "inicial")
	validacionInicial := validacionFirmaValidaPrueba(
		artefacto, instantePuertosPrueba.Add(6*time.Minute), "inicial",
	)
	sello := selloValidoPrueba(artefacto, politica)
	validacionTrasSello := validacionFirmaValidaPrueba(
		sello.ArtefactoSellado, instantePuertosPrueba.Add(7*time.Minute), "sellado", PerfilFirmaPAdESBaselineT,
	)
	artefactoFinal := artefactoFirmaValidoPrueba(t, contenido, politica, "longevo")
	artefactoFinal.FirmaRef = artefacto.FirmaRef
	artefactoFinal.HuellaFirmaSHA256 = artefacto.HuellaFirmaSHA256
	artefactoFinal.SesionFirmaRef = artefacto.SesionFirmaRef
	artefactoFinal.EvidenciaFirmaInteractivaRef = artefacto.EvidenciaFirmaInteractivaRef
	artefactoFinal.HuellaEvidenciaInteractivaSHA256 = artefacto.HuellaEvidenciaInteractivaSHA256
	artefactoFinal.FirmadaEn = artefacto.FirmadaEn
	aumento := ResultadoAumentoFirma{
		ArtefactoOrigen: sello.ArtefactoSellado, Artefacto: artefactoFinal,
		NivelAlcanzadoClave:            politica.NivelAumentoClave,
		PoliticaLongevidadRef:          politica.PoliticaLongevidadRef,
		PoliticaLongevidadVersion:      politica.PoliticaLongevidadVersion,
		HuellaPoliticaLongevidadSHA256: politica.HuellaPoliticaLongevidadSHA256,
		EvidenciaAumentoRef:            "evidencia-aumento-1", HuellaEvidenciaSHA256: huellaPruebaPuertos("6"),
		AumentadaEn: instantePuertosPrueba.Add(8 * time.Minute),
	}
	validacionFinal := validacionFirmaValidaPrueba(
		artefactoFinal, instantePuertosPrueba.Add(9*time.Minute), "final", PerfilFirmaPAdESBaselineLTA,
	)
	return construirManifiestoProbatorioPrueba(
		t, contenido, politica, artefacto, validacionInicial, &sello, &validacionTrasSello, &aumento,
		validacionFinal, documentoFirmadoCustodiadoPrueba(artefactoFinal),
	)
}

func construirManifiestoProbatorioPrueba(
	t *testing.T,
	contenido dominiobolsa.ContenidoDecisionTecnica,
	politica PoliticaFirmaBaremacion,
	artefacto ArtefactoFirma,
	validacionInicial ValidacionFirmaServidor,
	sello *SelloTiempoFirma,
	validacionTrasSello *ValidacionFirmaServidor,
	aumento *ResultadoAumentoFirma,
	validacionFinal ValidacionFirmaServidor,
	documentoFirmado DocumentoFirmadoCustodiado,
) ManifiestoProbatorioBaremacion {
	t.Helper()
	autorizaciones := make([]AutorizacionProbatoriaBaremacion, 0, 32)
	agregarAutorizacion := func(accion AccionOperacionBaremacion, recurso, referencia string) {
		clase, existe := ClaseRecursoRequeridaOperacionBaremacion(accion)
		if !existe {
			t.Fatalf("accion de manifiesto sin clase: %s", accion)
		}
		if referencia == "" {
			referencia = fmt.Sprintf("autorizacion-manifiesto-%02d", len(autorizaciones)+1)
		}
		autorizaciones = append(autorizaciones, AutorizacionProbatoriaBaremacion{
			Secuencia: uint32(len(autorizaciones) + 1), Accion: accion,
			ClaseRecurso: clase, RecursoRef: recurso, AutorizacionRef: referencia,
		})
	}
	agregarAutorizacion(AccionConsultarBaremacionVigente, contenido.BaremacionMeritoRef, "")
	agregarAutorizacion(AccionRecuperarCalculoBaremacion, contenido.CalculoOficial.CalculoRef, "")
	agregarAutorizacion(AccionConsultarCriterioBaremacion, contenido.Criterio.ProcesoRef, "")
	evidenciasMerito, err := contenido.CalculoOficial.EvidenciasCanonicas()
	if err != nil {
		t.Fatalf("evidencias canonicas de prueba: %v", err)
	}
	for _, evidencia := range evidenciasMerito {
		agregarAutorizacion(AccionConsultarEvidenciaBaremacion, evidencia.Referencia.DocumentoRef, "")
		agregarAutorizacion(AccionConsultarRepresentacionBaremacion, evidencia.Referencia.RepresentacionRef, "")
	}
	accionAdopcion, existe := AccionAdopcionParaClase(contenido.Clase)
	if !existe {
		t.Fatal("clase de decision de prueba sin accion")
	}
	agregarAutorizacion(accionAdopcion, contenido.BaremacionMeritoRef, contenido.AutorizacionRef)
	agregarAutorizacion(AccionConsultarPoliticaFirmaBaremacion, politica.Referencia, "")
	agregarAutorizacion(AccionCodificarDecisionBaremacion, contenido.ID, "")
	agregarAutorizacion(AccionCustodiarDecisionBaremacion, contenido.ID, "")
	agregarAutorizacion(AccionPrepararFirmaDecisionBaremacion, contenido.ID, "")
	agregarAutorizacion(AccionConsultarFirmaDecisionBaremacion, artefacto.SesionFirmaRef, "")
	agregarAutorizacion(AccionValidarFirmaDecisionBaremacion, artefacto.FirmaRef, "")
	if sello != nil {
		agregarAutorizacion(AccionSellarTiempoDecisionBaremacion, artefacto.FirmaRef, "")
	}
	artefactoFinal := artefacto
	if sello != nil {
		artefactoFinal = sello.ArtefactoSellado
	}
	if aumento != nil {
		agregarAutorizacion(AccionValidarFirmaDecisionBaremacion, sello.ArtefactoSellado.FirmaRef, "")
		agregarAutorizacion(AccionAumentarFirmaDecisionBaremacion, sello.ArtefactoSellado.FirmaRef, "")
		artefactoFinal = aumento.Artefacto
	}
	if sello != nil {
		agregarAutorizacion(AccionValidarFirmaDecisionBaremacion, artefactoFinal.FirmaRef, "")
	}
	agregarAutorizacion(AccionRecuperarBinarioFirmadoBaremacion, artefactoFinal.DocumentoFirmadoRef, "")
	agregarAutorizacion(AccionCustodiarDocumentoFirmadoBaremacion, artefactoFinal.DocumentoFirmadoRef, "")
	agregarAutorizacion(AccionRetenerDocumentoFirmadoBaremacion, artefactoFinal.DocumentoFirmadoRef, "")
	agregarAutorizacion(AccionReservarDecisionBaremacion, contenido.BaremacionMeritoRef, "")
	agregarAutorizacion(AccionConfirmarDecisionBaremacion, contenido.BaremacionMeritoRef, "")

	huellaContenido, err := contenido.HuellaContenidoSHA256()
	if err != nil {
		t.Fatalf("huella del contenido de prueba: %v", err)
	}
	evidencias := []EvidenciaProbatoriaBaremacion{
		{Tipo: EvidenciaEstadoBaseBaremacion, Referencia: contenido.BaremacionMeritoRef, HuellaEvidenciaSHA256: contenido.HuellaEstadoAnteriorSHA256},
		{Tipo: EvidenciaCalculoOficialBaremacion, Referencia: "evidencia-gobierno-calculo-1", HuellaEvidenciaSHA256: huellaPruebaPuertos("f")},
		{Tipo: EvidenciaCriterioPublicadoBaremacion, Referencia: contenido.Criterio.ProcesoRef, HuellaEvidenciaSHA256: contenido.Criterio.HuellaSHA256},
	}
	for _, evidencia := range evidenciasMerito {
		evidencias = append(evidencias,
			EvidenciaProbatoriaBaremacion{Tipo: EvidenciaDocumentoMeritoBaremacion, Referencia: evidencia.Referencia.DocumentoRef, HuellaEvidenciaSHA256: evidencia.Referencia.HuellaSHA256},
			EvidenciaProbatoriaBaremacion{Tipo: EvidenciaRepresentacionBaremacion, Referencia: evidencia.Referencia.RepresentacionRef, HuellaEvidenciaSHA256: evidencia.Referencia.HuellaSHA256},
		)
	}
	evidencias = append(evidencias,
		EvidenciaProbatoriaBaremacion{Tipo: EvidenciaContenidoDecisionBaremacion, Referencia: contenido.ID, HuellaEvidenciaSHA256: huellaContenido},
		EvidenciaProbatoriaBaremacion{Tipo: EvidenciaPoliticaFirmaBaremacion, Referencia: politica.AprobacionRef, HuellaEvidenciaSHA256: politica.HuellaAprobacionSHA256},
		EvidenciaProbatoriaBaremacion{Tipo: EvidenciaDocumentoCanonicoBaremacion, Referencia: contenido.ID, HuellaEvidenciaSHA256: artefacto.HuellaDocumentoFirmableSHA256},
		EvidenciaProbatoriaBaremacion{Tipo: EvidenciaCustodiaFirmableBaremacion, Referencia: artefacto.EvidenciaCustodiaRef, HuellaEvidenciaSHA256: artefacto.HuellaDocumentoFirmableSHA256},
		EvidenciaProbatoriaBaremacion{Tipo: EvidenciaPreparacionFirmaBaremacion, Referencia: "evidencia-preparacion-1", HuellaEvidenciaSHA256: huellaPruebaPuertos("6")},
		EvidenciaProbatoriaBaremacion{Tipo: EvidenciaConsultaFirmaBaremacion, Referencia: "evidencia-consulta-firma-1", HuellaEvidenciaSHA256: huellaPruebaPuertos("7")},
		EvidenciaProbatoriaBaremacion{Tipo: EvidenciaValidacionInicialBaremacion, Referencia: validacionInicial.ValidacionRef, HuellaEvidenciaSHA256: validacionInicial.HuellaValidacionSHA256},
	)
	if sello != nil {
		evidencias = agregarEvidenciasVinculoSelloPrueba(t, evidencias, *sello, validacionTrasSello)
	}
	if aumento != nil {
		if validacionTrasSello == nil {
			t.Fatal("aumento de prueba sin validacion PAdES-T")
		}
		evidencias = append(evidencias, EvidenciaProbatoriaBaremacion{
			Tipo: EvidenciaValidacionDocumentoSelladoBaremacion, Referencia: validacionTrasSello.ValidacionRef,
			HuellaEvidenciaSHA256: validacionTrasSello.HuellaValidacionSHA256,
		})
		evidencias = append(evidencias, EvidenciaProbatoriaBaremacion{
			Tipo: EvidenciaAumentoLongevidadBaremacion, Referencia: aumento.EvidenciaAumentoRef,
			HuellaEvidenciaSHA256: aumento.HuellaEvidenciaSHA256,
		})
		evidencias = agregarEvidenciaVinculoAumentoPrueba(
			t, evidencias, *sello, *validacionTrasSello, *aumento, validacionFinal,
		)
	}
	evidencias = append(evidencias,
		EvidenciaProbatoriaBaremacion{Tipo: EvidenciaValidacionFinalBaremacion, Referencia: validacionFinal.ValidacionRef, HuellaEvidenciaSHA256: validacionFinal.HuellaValidacionSHA256},
		EvidenciaProbatoriaBaremacion{Tipo: EvidenciaRecuperacionFirmadoBaremacion, Referencia: documentoFirmado.EvidenciaRecuperacionRef, HuellaEvidenciaSHA256: documentoFirmado.HuellaEvidenciaRecuperacionSHA256},
		EvidenciaProbatoriaBaremacion{Tipo: EvidenciaCustodiaFirmadoBaremacion, Referencia: documentoFirmado.EvidenciaEscritura.Referencia, HuellaEvidenciaSHA256: documentoFirmado.HuellaDocumentoSHA256},
		EvidenciaProbatoriaBaremacion{Tipo: EvidenciaRetencionFirmadoBaremacion, Referencia: documentoFirmado.EvidenciaRetencion.Referencia, HuellaEvidenciaSHA256: documentoFirmado.HuellaDocumentoSHA256},
	)
	for indice := range evidencias {
		evidencias[indice].Secuencia = uint32(indice + 1)
	}
	manifiesto := ManifiestoProbatorioBaremacion{
		Esquema:        EsquemaManifiestoProbatorioBaremacion,
		Finalidad:      FinalidadManifiestoProbatorioBaremacion,
		VersionEsquema: VersionManifiestoProbatorioBaremacion,
		Referencia:     "manifiesto-probatorio-1", ProcesoRef: contenido.ProcesoRef,
		SolicitudRef: contenido.SolicitudRef, SujetoRef: contenido.SujetoRef,
		BaremacionMeritoRef: contenido.BaremacionMeritoRef, DecisionRef: contenido.ID,
		VersionBase: contenido.VersionAnteriorBaremacion, HuellaVersionBaseSHA256: contenido.HuellaEstadoAnteriorSHA256,
		Autorizaciones: autorizaciones, Evidencias: evidencias,
		CreadoEn: validacionFinal.ValidadaEn.Add(time.Minute).UTC(),
	}
	preparado, _, err := manifiesto.PrepararSellado()
	if err != nil {
		t.Fatalf("preparar manifiesto: %v", err)
	}
	resultado, err := preparado.IncorporarSello("hmac-sha256:manifiesto_1:" + huellaPruebaPuertos("3"))
	if err != nil {
		t.Fatalf("sellar manifiesto: %v", err)
	}
	return resultado
}

func selloValidoPrueba(artefacto ArtefactoFirma, politica PoliticaFirmaBaremacion) SelloTiempoFirma {
	artefactoSellado := artefacto
	artefactoSellado.DocumentoFirmadoRef += ":pades-t"
	artefactoSellado.HuellaDocumentoSHA256 = huellaPruebaPuertos("c")
	return SelloTiempoFirma{
		SelloTiempoRef: "sello-tiempo-1", HuellaSelloTiempoSHA256: huellaPruebaPuertos("d"),
		ArtefactoOrigen: artefacto, ArtefactoSellado: artefactoSellado,
		PoliticaSelloTiempoRef:          politica.PoliticaSelloTiempoRef,
		PoliticaSelloTiempoVersion:      politica.PoliticaSelloTiempoVersion,
		HuellaPoliticaSelloTiempoSHA256: politica.HuellaPoliticaSelloTiempoSHA256,
		ValidacionSelloRef:              "validacion-sello-1", HuellaValidacionSHA256: huellaPruebaPuertos("e"),
		SelladoEn: instantePuertosPrueba.Add(7 * time.Minute),
	}
}

func huellaPruebaPuertos(caracter string) string { return strings.Repeat(caracter, 64) }

type repositorioBaremacionesContrato struct{}

func (repositorioBaremacionesContrato) ReservarCambio(context.Context, SolicitudReservarCambioBaremacion) (ReservaCambioBaremacion, error) {
	return ReservaCambioBaremacion{}, nil
}
func (repositorioBaremacionesContrato) ConfirmarCambio(context.Context, SolicitudConfirmarCambioBaremacion) (ResultadoConfirmarCambioBaremacion, error) {
	return ResultadoConfirmarCambioBaremacion{}, nil
}
func (repositorioBaremacionesContrato) AbandonarReserva(context.Context, SolicitudAbandonarReservaBaremacion) error {
	return nil
}
func (repositorioBaremacionesContrato) ObtenerVersionVigente(context.Context, SolicitudObtenerBaremacionVigente) (VersionBaremacion, error) {
	return VersionBaremacion{}, nil
}
func (repositorioBaremacionesContrato) ObtenerVersion(context.Context, SolicitudObtenerVersionBaremacion) (VersionBaremacion, error) {
	return VersionBaremacion{}, nil
}
func (repositorioBaremacionesContrato) ObtenerEvidenciaTransaccion(context.Context, SolicitudObtenerEvidenciaTransaccionBaremacion) (EvidenciaTransaccionBaremacionRecuperada, error) {
	return EvidenciaTransaccionBaremacionRecuperada{}, nil
}

type fuenteDatosBaremacionContrato struct{}

func (fuenteDatosBaremacionContrato) ObtenerCriterio(context.Context, SolicitudObtenerCriterioBaremacion) (CriterioBaremacionConfiable, error) {
	return CriterioBaremacionConfiable{}, nil
}
func (fuenteDatosBaremacionContrato) ObtenerEvidencia(context.Context, SolicitudObtenerEvidenciaBaremacion) (EvidenciaBaremacionConfiable, error) {
	return EvidenciaBaremacionConfiable{}, nil
}
func (fuenteDatosBaremacionContrato) ObtenerRepresentacion(context.Context, SolicitudObtenerRepresentacionBaremacion) (RepresentacionBaremacionConfiable, error) {
	return RepresentacionBaremacionConfiable{}, nil
}

type calculadorOficialContrato struct{}

func (calculadorOficialContrato) CalcularPuntuacionOficial(context.Context, SolicitudCalcularPuntuacionOficial) (ResultadoCalculoOficial, error) {
	return ResultadoCalculoOficial{}, nil
}
func (calculadorOficialContrato) RecuperarCalculoOficial(context.Context, SolicitudRecuperarCalculoOficial) (ResultadoCalculoOficial, error) {
	return ResultadoCalculoOficial{}, nil
}

type catalogoPoliticasFirmaContrato struct{}

func (catalogoPoliticasFirmaContrato) ObtenerPoliticaFirma(context.Context, SolicitudObtenerPoliticaFirma) (PoliticaFirmaBaremacion, error) {
	return PoliticaFirmaBaremacion{}, nil
}

type codificadorCanonicoContrato struct{}

func (codificadorCanonicoContrato) CodificarDecision(context.Context, SolicitudCodificarDecisionCanonica) (CodificacionCanonicaDecision, error) {
	return CodificacionCanonicaDecision{}, nil
}

type firmadorInteractivoContrato struct{}

func (firmadorInteractivoContrato) PrepararFirmaInteractiva(context.Context, SolicitudPrepararFirmaInteractiva) (SesionFirmaInteractiva, error) {
	return SesionFirmaInteractiva{}, nil
}
func (firmadorInteractivoContrato) ConsultarFirmaInteractiva(context.Context, SolicitudConsultarFirmaInteractiva) (ConsultaFirmaInteractiva, error) {
	return ConsultaFirmaInteractiva{}, nil
}

type validadorFirmaContrato struct{}

func (validadorFirmaContrato) ValidarFirmaServidor(context.Context, SolicitudValidarFirmaServidor) (ValidacionFirmaServidor, error) {
	return ValidacionFirmaServidor{}, nil
}

type selladorTiempoContrato struct{}

func (selladorTiempoContrato) SellarTiempoFirma(context.Context, SolicitudSellarTiempoFirma) (SelloTiempoFirma, error) {
	return SelloTiempoFirma{}, nil
}

type aumentadorFirmaContrato struct{}

func (aumentadorFirmaContrato) AumentarFirma(context.Context, SolicitudAumentarFirma) (ResultadoAumentoFirma, error) {
	return ResultadoAumentoFirma{}, nil
}

type archivoFirmaContrato struct{}

func (archivoFirmaContrato) RecuperarArtefactoFirma(context.Context, SolicitudRecuperarArtefactoFirma) (ArtefactoFirma, error) {
	return ArtefactoFirma{}, nil
}
func (archivoFirmaContrato) RecuperarValidacionFirma(context.Context, SolicitudRecuperarValidacionFirma) (ValidacionFirmaServidor, error) {
	return ValidacionFirmaServidor{}, nil
}
func (archivoFirmaContrato) RecuperarSelloTiempo(context.Context, SolicitudRecuperarSelloTiempo) (SelloTiempoFirma, error) {
	return SelloTiempoFirma{}, nil
}
func (archivoFirmaContrato) RecuperarAumentoFirma(context.Context, SolicitudRecuperarAumentoFirma) (ResultadoAumentoFirma, error) {
	return ResultadoAumentoFirma{}, nil
}

type selladorSolicitudContrato struct{}

func (selladorSolicitudContrato) SellarSolicitudBaremacion(context.Context, CargaProtegida) (string, error) {
	return "hmac-sha256:prueba:" + huellaPruebaPuertos("f"), nil
}

type generadorReferenciasContrato struct{}

func (generadorReferenciasContrato) NuevoIDBaremacion() (string, error) {
	return "baremacion-prueba", nil
}
func (generadorReferenciasContrato) NuevoIDDecisionTecnica() (string, error) {
	return "decision-prueba", nil
}
func (generadorReferenciasContrato) NuevaReferenciaManifiestoProbatorio() (string, error) {
	return "manifiesto-probatorio-prueba", nil
}
func (generadorReferenciasContrato) NuevaReferenciaCorrelacion() (string, error) {
	return "correlacion-prueba", nil
}
func (generadorReferenciasContrato) NuevaReferenciaEfectoAlmacen() (string, error) {
	return "efecto-almacen-prueba", nil
}

type relojContrato struct{}

func (relojContrato) Ahora() time.Time { return time.Unix(0, 0).UTC() }

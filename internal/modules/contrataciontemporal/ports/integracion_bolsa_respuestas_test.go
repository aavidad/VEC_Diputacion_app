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

func TestRespuestaDisponibilidadExigeFrescuraYAutenticacionTCB(t *testing.T) {
	base := instanteBolsaPrueba()
	ahora := base.Add(3 * time.Minute)
	solicitud := solicitudDisponibilidadPrueba(base)
	resultado := resultadoDisponibilidadPrueba(base)
	sellador := &selladorHMACBolsaPrueba{clave: []byte("clave-prueba-respuesta-bolsa")}
	firmarDisponibilidadPrueba(t, sellador, solicitud, &resultado)

	if err := resultado.ValidarParaEn(solicitud, ahora); err != nil {
		t.Fatalf("respuesta nominal válida rechazada: %v", err)
	}
	verificador, err := NuevoVerificadorEvidenciaIntegracionBolsa(sellador)
	if err != nil {
		t.Fatalf("crear verificador: %v", err)
	}
	comprobante, err := verificador.VerificarDisponibilidad(
		context.Background(), solicitud, resultado, ahora,
	)
	if err != nil || comprobante.datos == nil {
		t.Fatalf("respuesta auténtica no promovida: comprobante=%v err=%v", comprobante.datos, err)
	}

	if err := resultado.ValidarParaEn(solicitud, resultado.Procedencia.Evidencia.ValidaHasta); !errors.Is(err, ErrRespuestaBolsaNoConfiable) {
		t.Fatalf("consulta volátil caducada aceptada: %v", err)
	}

	forjada := resultado
	forjada.Procedencia.Evidencia.SelloHMAC =
		selloNominalBolsaPrueba(dominioSelloRespuestaBolsa, "4")
	if err := forjada.ValidarParaEn(solicitud, ahora); err != nil {
		t.Fatalf("la forma nominal no debía fingir autenticidad ni invalidar estructura: %v", err)
	}
	if _, err := verificador.VerificarDisponibilidad(
		context.Background(), solicitud, forjada, ahora,
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("sello nominal fabricado obtuvo comprobante: %v", err)
	}

	alterada := resultado
	alterada.CantidadDisponible++
	if err := alterada.ValidarParaEn(solicitud, ahora); err != nil {
		t.Fatalf("la alteración estructuralmente válida debía llegar al cotejo HMAC: %v", err)
	}
	if _, err := verificador.VerificarDisponibilidad(
		context.Background(), solicitud, alterada, ahora,
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("respuesta alterada conservó autenticidad: %v", err)
	}

	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := verificador.VerificarDisponibilidad(
		cancelado, solicitud, resultado, ahora,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("se perdió la cancelación del conector TCB: %v", err)
	}
}

func TestSellosBolsaUsanSobreComunVersionadoSinConfundirSintaxisYAutenticidad(t *testing.T) {
	base := instanteBolsaPrueba()
	contexto := contextoBolsaPrueba(base)
	if !SelloHMACSHA256Valido(contexto.SelloPeticionHMAC) ||
		!selloHMACBolsaValido(contexto.SelloPeticionHMAC, dominioSelloPeticionBolsa) {
		t.Fatal("sello común versionado válido rechazado")
	}
	for _, sello := range []string{
		strings.Repeat("a", 64),
		selloNominalBolsaPrueba(dominioSelloRespuestaBolsa, "a"),
		"hmac-sha256:" + dominioSelloPeticionBolsa + "/v01:" + strings.Repeat("a", 64),
		"hmac-sha256:" + dominioSelloPeticionBolsa + "/v1:" + strings.Repeat("0", 64),
	} {
		alterado := contexto
		alterado.SelloPeticionHMAC = sello
		if alterado.ValidarEn(base.Add(time.Minute)) == nil {
			t.Fatalf("sello de petición inválido aceptado: %q", sello)
		}
	}
}

func TestEvidenciaBolsaLigaContextoFinalidadTiemposYAutoridad(t *testing.T) {
	base := instanteBolsaPrueba()
	ahora := base.Add(3 * time.Minute)
	solicitud := solicitudDisponibilidadPrueba(base)
	resultado := resultadoDisponibilidadPrueba(base)
	sellador := &selladorHMACBolsaPrueba{clave: []byte("otra-clave-prueba-bolsa")}
	firmarDisponibilidadPrueba(t, sellador, solicitud, &resultado)
	verificador, _ := NuevoVerificadorEvidenciaIntegracionBolsa(sellador)

	pruebas := []struct {
		nombre  string
		cambiar func(*SolicitudDisponibilidadBolsa, *ResultadoDisponibilidadBolsa)
	}{
		{"organización", func(s *SolicitudDisponibilidadBolsa, r *ResultadoDisponibilidadBolsa) {
			s.Contexto.OrganizacionRef = "organizacion:otra"
			r.OrganizacionRef = s.Contexto.OrganizacionRef
		}},
		{"expediente", func(s *SolicitudDisponibilidadBolsa, r *ResultadoDisponibilidadBolsa) {
			s.Contexto.ExpedienteRef = "expediente:otro"
			r.ExpedienteRef = s.Contexto.ExpedienteRef
		}},
		{"finalidad", func(s *SolicitudDisponibilidadBolsa, _ *ResultadoDisponibilidadBolsa) {
			s.Contexto.Finalidad = referenciaBolsaPrueba("finalidad:otra", "1")
		}},
		{"correlación", func(s *SolicitudDisponibilidadBolsa, r *ResultadoDisponibilidadBolsa) {
			s.Contexto.CorrelacionRef = "correlacion:otra"
			r.CorrelacionRef = s.Contexto.CorrelacionRef
		}},
		{"tiempo", func(s *SolicitudDisponibilidadBolsa, _ *ResultadoDisponibilidadBolsa) {
			s.Contexto.SolicitadaEn = s.Contexto.SolicitadaEn.Add(time.Microsecond)
		}},
		{"autoridad", func(_ *SolicitudDisponibilidadBolsa, r *ResultadoDisponibilidadBolsa) {
			r.Procedencia.AutoridadRef = "autoridad:otra"
		}},
		{"respuesta", func(_ *SolicitudDisponibilidadBolsa, r *ResultadoDisponibilidadBolsa) {
			r.Procedencia.RespuestaRef = "respuesta:otra"
		}},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			solicitudAlterada := solicitud
			resultadoAlterado := resultado
			prueba.cambiar(&solicitudAlterada, &resultadoAlterado)
			if resultadoAlterado.ValidarParaEn(solicitudAlterada, ahora) != nil {
				return
			}
			if _, err := verificador.VerificarDisponibilidad(
				context.Background(), solicitudAlterada, resultadoAlterado, ahora,
			); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
				t.Fatalf("alteración %s no rompió el vínculo canónico: %v", prueba.nombre, err)
			}
		})
	}
}

func TestRecibosDurablesOrdenYLlamamientoQuedanAutenticados(t *testing.T) {
	base := instanteBolsaPrueba()
	ahora := base.Add(3 * time.Minute)
	sellador := &selladorHMACBolsaPrueba{clave: []byte("clave-recibos-bolsa")}
	verificador, _ := NuevoVerificadorEvidenciaIntegracionBolsa(sellador)

	comandoOrden := comandoOrdenPrueba(base)
	reciboOrden := reciboOrdenPrueba(base)
	firmarOrdenPrueba(t, sellador, comandoOrden, &reciboOrden)
	if reciboOrden.ValidarParaEn(comandoOrden, ahora) != nil {
		t.Fatal("recibo de orden nominal válido rechazado")
	}
	if comprobante, err := verificador.VerificarReciboOrden(
		context.Background(), comandoOrden, reciboOrden, ahora,
	); err != nil || comprobante.datos == nil {
		t.Fatalf("recibo de orden no autenticado: %v", err)
	}
	if err := reciboOrden.ValidarParaEn(
		comandoOrden, comandoOrden.Contexto.ValidaHasta,
	); !errors.Is(err, ErrRespuestaBolsaNoConfiable) {
		t.Fatalf("respuesta de transporte caducada aceptada: %v", err)
	}
	if err := reciboOrden.ValidarDurablePara(comandoOrden); err != nil {
		t.Fatalf("recibo durable autenticado perdió validez estructural: %v", err)
	}

	comandoLlamamiento := comandoLlamamientoPrueba(t, base, sellador)
	reciboLlamamiento := reciboLlamamientoPrueba(t, comandoLlamamiento, base)
	firmarLlamamientoPrueba(t, sellador, comandoLlamamiento, &reciboLlamamiento)
	if reciboLlamamiento.ValidarParaEn(comandoLlamamiento, ahora) != nil {
		t.Fatal("recibo de llamamiento nominal válido rechazado")
	}
	if comprobante, err := verificador.VerificarReciboLlamamiento(
		context.Background(), comandoLlamamiento, reciboLlamamiento, ahora,
	); err != nil || comprobante.datos == nil {
		t.Fatalf("recibo de llamamiento no autenticado: %v", err)
	}
	if err := reciboLlamamiento.ValidarDurablePara(comandoLlamamiento); err != nil {
		t.Fatalf("recibo durable de llamamiento inválido: %v", err)
	}

	alterado := reciboLlamamiento
	datosComando, err := comandoLlamamiento.DatosEn(ahora)
	if err != nil {
		t.Fatalf("leer comando probado: %v", err)
	}
	alterado.OrdenSeleccionado = datosComando.TotalPosicionesOrden + 1
	if err := alterado.ValidarParaEn(comandoLlamamiento, ahora); !errors.Is(err, ErrRespuestaBolsaNoConfiable) {
		t.Fatalf("posición fuera del total aceptada: %v", err)
	}
	alterado = reciboLlamamiento
	alterado.OrdenSeleccionado = datosComando.MaximaPosicionEvaluable + 1
	if err := alterado.ValidarParaEn(comandoLlamamiento, ahora); !errors.Is(err, ErrRespuestaBolsaNoConfiable) {
		t.Fatalf("posición fuera del límite aceptada: %v", err)
	}
	alterado = reciboLlamamiento
	alterado.RetencionSeleccion = ReferenciaVersionadaIntegracionBolsa{}
	if err := alterado.ValidarParaEn(comandoLlamamiento, ahora); !errors.Is(err, ErrRespuestaBolsaNoConfiable) {
		t.Fatalf("selección seudonimizada sin retención gobernada aceptada: %v", err)
	}
}

func TestComandoLlamamientoDerivaTotalDeOrdenAutenticada(t *testing.T) {
	base := instanteBolsaPrueba()
	ahora := base.Add(3 * time.Minute)
	sellador := &selladorHMACBolsaPrueba{clave: []byte("clave-orden-autoritativa")}
	comandoOrden := comandoOrdenPrueba(base)
	reciboOrden := reciboOrdenPrueba(base)
	firmarOrdenPrueba(t, sellador, comandoOrden, &reciboOrden)
	contexto := contextoBolsaPrueba(base)
	contexto.OperacionRef = "operacion:llamar:desde-orden"

	preparacion := PreparacionComandoSolicitarLlamamientoBolsa{
		Contexto: contexto, ComandoOrden: comandoOrden, ReciboOrden: reciboOrden,
		MaximaPosicionEvaluable: 50,
	}
	if _, err := NuevoComandoSolicitarLlamamientoBolsa(
		preparacion, ahora,
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("orden nominal sin comprobante creó comando: %v", err)
	}

	verificador, _ := NuevoVerificadorEvidenciaIntegracionBolsa(sellador)
	comprobante, err := verificador.VerificarReciboOrden(
		context.Background(), comandoOrden, reciboOrden, ahora,
	)
	if err != nil {
		t.Fatalf("autenticar orden: %v", err)
	}
	preparacion.ComprobanteOrden = comprobante
	comando, err := NuevoComandoSolicitarLlamamientoBolsa(preparacion, ahora)
	if err != nil {
		t.Fatalf("orden comprobada no creó comando: %v", err)
	}
	datos, err := comando.DatosEn(ahora)
	if err != nil || datos.TotalPosicionesOrden != reciboOrden.TotalPosiciones ||
		datos.HuellaReciboOrden != huellaBytesBolsa(materialReciboOrdenBolsa(comandoOrden, reciboOrden)) {
		t.Fatalf("total/huella no derivados del recibo: datos=%+v err=%v", datos, err)
	}

	alterada := preparacion
	alterada.ReciboOrden.TotalPosiciones++
	if _, err := NuevoComandoSolicitarLlamamientoBolsa(
		alterada, ahora,
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("total alterado conservó comprobante: %v", err)
	}
	fueraDeOrden := preparacion
	fueraDeOrden.MaximaPosicionEvaluable = reciboOrden.TotalPosiciones + 1
	if _, err := NuevoComandoSolicitarLlamamientoBolsa(
		fueraDeOrden, ahora,
	); !errors.Is(err, ErrPeticionIntegracionBolsaInvalida) {
		t.Fatalf("límite superior al total aceptado: %v", err)
	}
}

func TestContratoNoDependeDeCookiesWebNiExponeIdentificadoresDirectos(t *testing.T) {
	tipos := []reflect.Type{
		reflect.TypeOf(SolicitudDisponibilidadBolsa{}),
		reflect.TypeOf(ResultadoDisponibilidadBolsa{}),
		reflect.TypeOf(ComandoPrepararOrdenBolsa{}),
		reflect.TypeOf(ReciboOrdenBolsa{}),
		reflect.TypeOf(ComandoSolicitarLlamamientoBolsa{}),
		reflect.TypeOf(DatosComandoSolicitarLlamamientoBolsa{}),
		reflect.TypeOf(ReciboSolicitudLlamamientoBolsa{}),
		reflect.TypeOf(EventoLlamamientoBolsa{}),
	}
	prohibidos := []string{
		"dni", "nie", "nif", "nombre", "apellidos", "correo", "email",
		"telefono", "movil", "direccion", "cookie", "sesion", "cabecera",
		"rol", "permiso", "actor", "perfil",
	}
	for _, tipo := range tipos {
		comprobarCamposIntegracionBolsa(t, tipo, prohibidos)
	}
	tipoRecibo := reflect.TypeOf(ReciboSolicitudLlamamientoBolsa{})
	if _, existe := tipoRecibo.FieldByName("SeleccionRef"); !existe {
		t.Fatal("se perdió la referencia seudonimizada necesaria")
	}
	if _, existe := tipoRecibo.FieldByName("RetencionSeleccion"); !existe {
		t.Fatal("la selección personal quedó sin política de retención")
	}
}

func TestPuertosBolsaConservanContextoCancelableYNoUnaSesionWeb(t *testing.T) {
	tipoContexto := reflect.TypeOf((*context.Context)(nil)).Elem()
	puertos := []reflect.Type{
		reflect.TypeOf((*ConsultaDisponibilidadBolsa)(nil)).Elem(),
		reflect.TypeOf((*PreparadorOrdenBolsa)(nil)).Elem(),
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

func comprobarCamposIntegracionBolsa(t *testing.T, tipo reflect.Type, prohibidos []string) {
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
			comprobarCamposIntegracionBolsa(t, campo.Type, prohibidos)
		}
	}
}

func TestComprobanteNoEsFabricableNiSerializablePorEntrada(t *testing.T) {
	tipo := reflect.TypeOf(ComprobanteEvidenciaIntegracionBolsa{})
	for indice := 0; indice < tipo.NumField(); indice++ {
		if tipo.Field(indice).IsExported() {
			t.Fatalf("comprobante expone campo fabricable: %s", tipo.Field(indice).Name)
		}
	}
	if _, err := NuevoVerificadorEvidenciaIntegracionBolsa(
		(*selladorHMACBolsaPrueba)(nil),
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("verificador aceptó dependencia tipada nula: %v", err)
	}
	if _, err := json.Marshal(ComprobanteEvidenciaIntegracionBolsa{}); !errors.Is(err, ErrSerializacionCapacidadBolsa) {
		t.Fatalf("comprobante se serializó: %v", err)
	}
	var comprobante ComprobanteEvidenciaIntegracionBolsa
	if err := json.Unmarshal([]byte(`{}`), &comprobante); !errors.Is(err, ErrSerializacionCapacidadBolsa) {
		t.Fatalf("comprobante se reconstruyó desde transporte: %v", err)
	}
	if _, err := json.Marshal(ComandoSolicitarLlamamientoBolsa{}); !errors.Is(err, ErrSerializacionCapacidadBolsa) {
		t.Fatalf("comando interno se serializó: %v", err)
	}
}

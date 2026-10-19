package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/catalogoalta"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestEntregaRatificadaRecuperaSnapshotOriginalTrasCambiarPublicacion(t *testing.T) {
	escenario, fuente := escenarioNecesidadRegistroPrueba(t)
	servicio, _ := construirServicioRegistro(t, escenario)
	if err := servicio.ConfigurarFuenteNecesidadesAlta(fuente); err != nil {
		t.Fatal(err)
	}
	entrada := *escenario.solicitud.NecesidadEntrada
	sellada, err := fuente.catalogo.SellarDatos(entrada)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := escenario.solicitud.Solicitud
	solicitud.Periodo, solicitud.Necesidad = sellada.Periodo, &sellada
	configuracion := domain.ConfiguracionPeticionCentro{Referencia: "config:centro:001", Version: 1,
		Solicitante: domain.ActorPeticionCentro{ActorRef: "actor:centro:uno", PerfilRef: "perfil:centro:uno", CentroRef: solicitud.CentroRef, PuestoRef: "puesto:centro:uno"},
		Ratificador: domain.ActorPeticionCentro{ActorRef: "actor:centro:dos", PerfilRef: "perfil:centro:dos", CentroRef: solicitud.CentroRef, PuestoRef: "puesto:centro:dos"}}
	peticion, err := domain.NuevaPeticionCentro("peticion:centro:historica", configuracion, solicitud, escenario.instante)
	if err != nil {
		t.Fatal(err)
	}
	peticion, err = peticion.Ratificar(configuracion.Ratificador, 1, "Revisión de segunda identidad", escenario.instante.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := escenario.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	const idAlta = "018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b"
	const ambito = "hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	e := ports.EntregaPeticionCentro{Peticion: peticion.Datos(), EstadoEntrega: "preparada", ClaveAlta: idAlta,
		AmbitoAltaHMAC: ambito, ActorRef: vinculo.PrincipalID, PerfilRef: vinculo.PerfilActivoRef}
	ctx, err := ConNecesidadRatificadaPeticionCentro(t.Context(), escenario.solicitud.OrganizacionRef, e)
	if err != nil {
		t.Fatal(err)
	}
	ambitos, err := ports.NuevaColeccionSellosHMAC(
		selloHMACRegistroPrueba("vec.contratacion-temporal.ambito-idempotencia/v2", "c"), []string{ambito})
	if err != nil {
		t.Fatal(err)
	}
	entrada.Periodo = sellada.Periodo
	fuente.catalogo = domain.CatalogoNecesidadesAlta{} // La publicación activa ya cambió.
	recuperada, err := servicio.prepararNecesidadAlta(ctx, entrada, ambitos,
		escenario.solicitud.OrganizacionRef, e.ActorRef, e.PerfilRef)
	if err != nil || !reflect.DeepEqual(recuperada, sellada) || fuente.resoluciones != 0 {
		t.Fatalf("snapshot ratificada sustituida: error=%v, resoluciones=%d", err, fuente.resoluciones)
	}
	entrada.Campos = map[string]string{"puesto_codigo": "otro"}
	if _, err := servicio.prepararNecesidadAlta(ctx, entrada, ambitos,
		escenario.solicitud.OrganizacionRef, e.ActorRef, e.PerfilRef); !errors.Is(err, ports.ErrClaveIdempotenciaUsada) {
		t.Fatalf("intención alterada admitida: %v", err)
	}
	entrada = *escenario.solicitud.NecesidadEntrada
	entrada.Periodo = sellada.Periodo
	if _, err := servicio.prepararNecesidadAlta(ctx, entrada, ambitos,
		"organizacion:otra", e.ActorRef, e.PerfilRef); !errors.Is(err, ports.ErrClaveIdempotenciaUsada) {
		t.Fatalf("marca de otra organización admitida: %v", err)
	}
}

type fuenteNecesidadRegistroPrueba struct {
	catalogo     domain.CatalogoNecesidadesAlta
	resultado    ports.ResultadoInstantaneaNecesidadAlta
	resoluciones int
}

type verificadorPuestoRPTPrueba struct {
	llamadas int
	existe   bool
}

func (v *verificadorPuestoRPTPrueba) VerificarPuestoRPTAlta(
	_ context.Context, s ports.SolicitudVerificarPuestoRPTAlta,
) (ports.PuestoRPTAltaVerificado, error) {
	v.llamadas++
	return ports.PuestoRPTAltaVerificado{
		CatalogoRef:          s.CatalogoRef,
		CatalogoHuellaSHA256: s.CatalogoHuellaSHA256, PuestoCodigo: s.PuestoCodigo,
		ExisteEnPublicacion: v.existe,
	}, nil
}

func (f *fuenteNecesidadRegistroPrueba) ResolverCatalogoNecesidadesAlta(
	_ context.Context, ref string, version uint64, huella string,
) (domain.CatalogoNecesidadesAlta, error) {
	f.resoluciones++
	if ref != f.catalogo.Referencia || version != f.catalogo.Version || huella != f.catalogo.HuellaSHA256 {
		return domain.CatalogoNecesidadesAlta{}, ports.ErrFuenteNecesidadesAltaNoDisponible
	}
	return f.catalogo, nil
}

func (f *fuenteNecesidadRegistroPrueba) RecuperarInstantaneaNecesidadConfirmada(
	_ context.Context, consulta ports.ConsultaInstantaneaNecesidadConfirmada,
) (ports.ResultadoInstantaneaNecesidadAlta, error) {
	if consulta.Validar() != nil {
		return ports.ResultadoInstantaneaNecesidadAlta{}, ports.ErrFuenteNecesidadesAltaNoDisponible
	}
	return f.resultado, nil
}

func escenarioNecesidadRegistroPrueba(t *testing.T) (escenarioRegistro, *fuenteNecesidadRegistroPrueba) {
	t.Helper()
	base := nuevoEscenarioRegistro(t)
	c, err := catalogoalta.CargarNecesidades("")
	if err != nil {
		t.Fatal(err)
	}
	base.solicitud.EsquemaAlta = EsquemaAltaNecesidadV1
	base.solicitud.Solicitud.MotivoClave = "vacante"
	base.solicitud.NecesidadEntrada = &domain.DatosNecesidadAlta{
		Esquema: "vec.ct.necesidad_alta.v1", CatalogoRef: c.Referencia,
		CatalogoVersion: c.Version, CatalogoHuellaSHA256: c.HuellaSHA256,
		CausaClave: "vacante", Periodo: base.solicitud.Solicitud.Periodo, JornadaMinutos: 2250,
		Campos: map[string]string{"numero_personas": "2", "plaza_codigo": "1201", "puesto_codigo": "3388", "organica_codigo": "100",
			"rpt_catalogo_ref":           "rpt:dipgra:2026",
			"rpt_catalogo_huella_sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"funcional_codigo":           "200", "proyecto_gasto_codigo": "300", "porcentaje_financiacion": "100"},
	}
	f := &fuenteNecesidadRegistroPrueba{catalogo: c, resultado: ports.ResultadoInstantaneaNecesidadAlta{Estado: ports.InstantaneaNecesidadAusente}}
	return base, f
}

func TestAltaNecesidadSellaAntesDeHuellasYUsaMismoServicio(t *testing.T) {
	escenario, fuente := escenarioNecesidadRegistroPrueba(t)
	servicio, dobles := construirServicioRegistro(t, escenario)
	if err := servicio.ConfigurarFuenteNecesidadesAlta(fuente); err != nil {
		t.Fatal(err)
	}
	verificador := &verificadorPuestoRPTPrueba{existe: true}
	if err := servicio.ConfigurarVerificadorPuestoRPTAlta(verificador); err != nil {
		t.Fatal(err)
	}
	verificada := false
	dobles.huellas.antes = func(m *ports.MaterialHuellaAlta) {
		n := m.Solicitud.Necesidad
		verificada = n != nil && n.CatalogoHuellaSHA256 == fuente.catalogo.HuellaSHA256 &&
			bytes.Equal(n.CatalogoInstantanea, fuente.catalogo.ContenidoCanonico)
	}
	recibo, err := servicio.Registrar(context.Background(), escenario.solicitud)
	if err != nil || recibo != escenario.recibo || !verificada || fuente.resoluciones != 1 || verificador.llamadas != 1 || dobles.transaccion.llamadas != 1 {
		t.Fatalf("alta v3 no selló antes del efecto: recibo=%+v err=%v", recibo, err)
	}
}

func TestAltaNecesidadReproduceSnapshotConfirmadaYDeniegaCruceLegado(t *testing.T) {
	escenario, fuente := escenarioNecesidadRegistroPrueba(t)
	servicio, dobles := construirServicioRegistro(t, escenario)
	if err := servicio.ConfigurarFuenteNecesidadesAlta(fuente); err != nil {
		t.Fatal(err)
	}
	fuente.resultado = ports.ResultadoInstantaneaNecesidadAlta{
		Estado:      ports.InstantaneaNecesidadConfirmadaV3,
		Instantanea: append([]byte(nil), fuente.catalogo.ContenidoCanonico...),
	}
	if _, err := servicio.Registrar(context.Background(), escenario.solicitud); err != nil || fuente.resoluciones != 0 {
		t.Fatalf("replay consultó catálogo vigente: %v", err)
	}
	escenario.solicitud.NecesidadEntrada.CatalogoVersion++
	if _, err := servicio.Registrar(context.Background(), escenario.solicitud); !errors.Is(err, ports.ErrClaveIdempotenciaUsada) {
		t.Fatalf("misma clave con otra versión de necesidad no devolvió 409: %v", err)
	}
	escenario.solicitud.NecesidadEntrada.CatalogoVersion--
	fuente.resultado = ports.ResultadoInstantaneaNecesidadAlta{Estado: ports.InstantaneaNecesidadLegadoV2}
	if _, err := servicio.Registrar(context.Background(), escenario.solicitud); !errors.Is(err, ports.ErrClaveIdempotenciaUsada) {
		t.Fatalf("cruce E2/E3 no colisionó: %v", err)
	}
	if dobles.transaccion.llamadas != 1 {
		t.Fatal("se intentó confirmar el cruce legado")
	}
	fuente.resultado = ports.ResultadoInstantaneaNecesidadAlta{Estado: ports.InstantaneaNecesidadConfirmadaV3, Instantanea: []byte("adulterada")}
	if _, err := servicio.Registrar(context.Background(), escenario.solicitud); !errors.Is(err, ErrResultadoRegistroNoConfiable) {
		t.Fatalf("snapshot corrupta no falló cerrada: %v", err)
	}
}

func TestAltaNuevaConPuestoEsperaFuenteRPTExacta(t *testing.T) {
	escenario, fuente := escenarioNecesidadRegistroPrueba(t)
	servicio, dobles := construirServicioRegistro(t, escenario)
	if err := servicio.ConfigurarFuenteNecesidadesAlta(fuente); err != nil {
		t.Fatal(err)
	}
	if _, err := servicio.Registrar(context.Background(), escenario.solicitud); !errors.Is(err, ErrServicioRegistroInvalido) {
		t.Fatalf("alta sin fuente T no falló cerrada: %v", err)
	}
	if dobles.transaccion.llamadas != 0 {
		t.Fatal("se confirmó sin verificar puesto")
	}
	verificador := &verificadorPuestoRPTPrueba{existe: false}
	if err := servicio.ConfigurarVerificadorPuestoRPTAlta(verificador); err != nil {
		t.Fatal(err)
	}
	if _, err := servicio.Registrar(context.Background(), escenario.solicitud); !errors.Is(err, ErrSolicitudRegistroInvalida) {
		t.Fatalf("puesto ausente en publicación admitido: %v", err)
	}
	if dobles.transaccion.llamadas != 0 || verificador.llamadas != 1 {
		t.Fatal("fuente T no bloqueó efecto")
	}
}

func TestProgramaTemporalV3EsNecesidadSinModalidadPrecargada(t *testing.T) {
	escenario, fuente := escenarioNecesidadRegistroPrueba(t)
	escenario.solicitud.Solicitud.MotivoClave = "programa_temporal"
	escenario.solicitud.NecesidadEntrada.CausaClave = "programa_temporal"
	escenario.solicitud.NecesidadEntrada.Campos = map[string]string{
		"organica_codigo": "100", "funcional_codigo": "200", "proyecto_gasto_codigo": "300",
		"porcentaje_financiacion": "100", "programa_denominacion": "Programa temporal de refuerzo",
		"numero_personas": "2",
		"programa_fin":    "2026-10-31", "proyecto_codigo": "P01",
		"financiacion_ref": "financiacion:opaca:1", "rc_ref": "rc:opaca:1",
	}
	servicio, dobles := construirServicioRegistro(t, escenario)
	if err := servicio.ConfigurarFuenteNecesidadesAlta(fuente); err != nil {
		t.Fatal(err)
	}
	if _, err := servicio.Registrar(context.Background(), escenario.solicitud); err != nil {
		t.Fatal(err)
	}
	evidencia, err := dobles.transaccion.orden.Datos()
	if err != nil || evidencia.Expediente.Analisis != nil ||
		evidencia.Expediente.Solicitud.Necesidad == nil ||
		evidencia.Expediente.Solicitud.Necesidad.CausaClave != "programa_temporal" {
		t.Fatalf("se dedujo modalidad de la causa del programa: %v", err)
	}
}

func TestMarcaPrivadaNecesidadExigeMaterialExacto(t *testing.T) {
	escenario, fuente := escenarioNecesidadRegistroPrueba(t)
	sellada, err := fuente.catalogo.SellarDatos(*escenario.solicitud.NecesidadEntrada)
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := contextoNecesidadAltaValidada(context.Background(), escenario.solicitud.OrganizacionRef, &sellada)
	if err != nil || !NecesidadAltaValidadaPara(ctx, escenario.solicitud.OrganizacionRef, &sellada) {
		t.Fatal("material sellado no reconocido", err)
	}
	alterada, err := sellada.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	alterada.Campos["puesto_codigo"] = "9999"
	if NecesidadAltaValidadaPara(ctx, escenario.solicitud.OrganizacionRef, &alterada) ||
		NecesidadAltaValidadaPara(ctx, "organizacion:ajena", &sellada) {
		t.Fatal("la marca privada se reutilizó con otros datos")
	}
}

func TestAltaNecesidadV2PropagaCantidadSelladaAntesDeHuella(t *testing.T) {
	escenario, fuente := escenarioNecesidadRegistroPrueba(t)
	escenario.solicitud.EsquemaAlta = EsquemaAltaNecesidadV2
	servicio, dobles := construirServicioRegistro(t, escenario)
	if err := servicio.ConfigurarFuenteNecesidadesAlta(fuente); err != nil {
		t.Fatal(err)
	}
	if err := servicio.ConfigurarVerificadorPuestoRPTAlta(&verificadorPuestoRPTPrueba{existe: true}); err != nil {
		t.Fatal(err)
	}
	var primera []byte
	dobles.huellas.antes = func(m *ports.MaterialHuellaAlta) {
		if m.Solicitud.NumeroPersonas != 2 || m.Solicitud.JornadaMinutos != 0 ||
			m.Solicitud.PuestoSolicitado != "" || m.Solicitud.Validar() != nil || m.Solicitud.Necesidad == nil {
			t.Fatal("la cantidad plural no salió de la necesidad sellada")
		}
		primera, _ = json.Marshal(m.Solicitud)
	}
	recibo, err := servicio.Registrar(context.Background(), escenario.solicitud)
	if err != nil || recibo != escenario.recibo || len(primera) == 0 || dobles.transaccion.llamadas != 1 ||
		dobles.huellasEfecto.expediente.Solicitud.NumeroPersonas != 2 {
		t.Fatalf("alta plural v2: recibo=%+v err=%v", recibo, err)
	}
	fuente.resultado = ports.ResultadoInstantaneaNecesidadAlta{
		Estado:      ports.InstantaneaNecesidadConfirmadaV3,
		Instantanea: append([]byte(nil), fuente.catalogo.ContenidoCanonico...)}
	fuente.catalogo = domain.CatalogoNecesidadesAlta{}
	dobles.huellas.antes = func(m *ports.MaterialHuellaAlta) {
		actual, e := json.Marshal(m.Solicitud)
		if e != nil || !bytes.Equal(actual, primera) {
			t.Fatal("replay v2 cambió la solicitud original")
		}
	}
	recibo2, err := servicio.Registrar(context.Background(), escenario.solicitud)
	if err != nil || recibo2 != recibo || fuente.resoluciones != 1 {
		t.Fatalf("replay plural cambió catálogo o recibo: %+v %v", recibo2, err)
	}
}

func TestAltaNecesidadV1PluralHistoricaConservaBytesYRecibo(t *testing.T) {
	escenario, fuente := escenarioNecesidadRegistroPrueba(t)
	servicio, dobles := construirServicioRegistro(t, escenario)
	if err := servicio.ConfigurarFuenteNecesidadesAlta(fuente); err != nil {
		t.Fatal(err)
	}
	if err := servicio.ConfigurarVerificadorPuestoRPTAlta(&verificadorPuestoRPTPrueba{existe: true}); err != nil {
		t.Fatal(err)
	}
	var primera []byte
	dobles.huellas.antes = func(m *ports.MaterialHuellaAlta) {
		if m.Solicitud.NumeroPersonas != 0 || m.Solicitud.JornadaMinutos != 0 || m.Solicitud.PuestoSolicitado != "" {
			t.Fatal("v1 cambió la forma histórica de la solicitud")
		}
		primera, _ = json.Marshal(m.Solicitud)
	}
	recibo, err := servicio.Registrar(context.Background(), escenario.solicitud)
	if err != nil || recibo != escenario.recibo || len(primera) == 0 {
		t.Fatalf("v1 original: %v", err)
	}
	fuente.resultado = ports.ResultadoInstantaneaNecesidadAlta{Estado: ports.InstantaneaNecesidadConfirmadaV3,
		Instantanea: append([]byte(nil), fuente.catalogo.ContenidoCanonico...)}
	fuente.catalogo = domain.CatalogoNecesidadesAlta{}
	dobles.huellas.antes = func(m *ports.MaterialHuellaAlta) {
		actual, e := json.Marshal(m.Solicitud)
		if e != nil || !bytes.Equal(actual, primera) {
			t.Fatal("replay v1 cambió HMAC original")
		}
	}
	recibo2, err := servicio.Registrar(context.Background(), escenario.solicitud)
	if err != nil || recibo2 != recibo || fuente.resoluciones != 1 {
		t.Fatalf("replay v1 alterado: %v", err)
	}
}

func TestCantidadV2DivergenteSeRechaza(t *testing.T) {
	escenario, fuente := escenarioNecesidadRegistroPrueba(t)
	escenario.solicitud.EsquemaAlta = EsquemaAltaNecesidadV2
	escenario.solicitud.Solicitud.JornadaMinutos = 2250
	escenario.solicitud.Solicitud.NumeroPersonas = 3
	escenario.solicitud.Solicitud.PuestoSolicitado = "Puesto publicado"
	servicio, dobles := construirServicioRegistro(t, escenario)
	if err := servicio.ConfigurarFuenteNecesidadesAlta(fuente); err != nil {
		t.Fatal(err)
	}
	if err := servicio.ConfigurarVerificadorPuestoRPTAlta(&verificadorPuestoRPTPrueba{existe: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := servicio.Registrar(context.Background(), escenario.solicitud); !errors.Is(err, ErrSolicitudRegistroInvalida) || dobles.transaccion.llamadas != 0 {
		t.Fatalf("N del centro distinta a la sellada llegó al efecto: %v", err)
	}
	escenario.solicitud.Solicitud.NumeroPersonas = 2
	escenario.solicitud.Solicitud.JornadaMinutos = 1800
	if _, err := servicio.Registrar(context.Background(), escenario.solicitud); !errors.Is(err, ErrSolicitudRegistroInvalida) || dobles.transaccion.llamadas != 0 {
		t.Fatalf("jornada distinta a la sellada llegó al efecto: %v", err)
	}
}

func TestSolicitudCentroCantidadSolaExigeNecesidadSellada(t *testing.T) {
	escenario, fuente := escenarioNecesidadRegistroPrueba(t)
	sellada, err := fuente.catalogo.SellarDatos(*escenario.solicitud.NecesidadEntrada)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := escenario.solicitud.Solicitud
	solicitud.Periodo, solicitud.Necesidad = sellada.Periodo, &sellada
	solicitud.NumeroPersonas = 2
	if err := solicitud.Validar(); err != nil {
		t.Fatalf("N sola sellada inválida: %v", err)
	}
	solicitud.NumeroPersonas = 3
	if solicitud.Validar() == nil {
		t.Fatal("N ajena a la necesidad sellada admitida")
	}
	solicitud.NumeroPersonas = 0
	if err := solicitud.Validar(); err != nil {
		t.Fatalf("ausencia V3 histórica invalidada: %v", err)
	}
	solicitud.Necesidad = nil
	solicitud.NumeroPersonas = 2
	if solicitud.Validar() == nil {
		t.Fatal("N sola sin necesidad sellada admitida")
	}
}

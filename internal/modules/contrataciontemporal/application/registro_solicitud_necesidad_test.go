package application

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/catalogoalta"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

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

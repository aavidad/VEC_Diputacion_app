package ports_test

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestMaterialConsultaRRHHEsRedactadoYNoSerializable(t *testing.T) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	autoridad, contexto := autoridadYContextoPuertosRRHH(t, ahora)
	solicitud, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 50, "")
	if err != nil {
		t.Fatal(err)
	}
	capacidad := capacidadCuadroPuertosRRHH(
		t, autoridad, contexto, solicitud, ahora,
	)
	orden, err := ports.NuevaOrdenConsultaCuadroRRHH(
		contexto, capacidad, solicitud, ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitudDetalle, err := ports.NuevaSolicitudDetalleRRHH(
		"expediente:rrhh:001", 1,
	)
	if err != nil {
		t.Fatal(err)
	}
	capacidadDetalle := capacidadDetallePuertosRRHH(
		t, autoridad, contexto, solicitudDetalle, ahora,
	)
	ordenDetalle, err := ports.NuevaOrdenConsultaDetalleRRHH(
		contexto, capacidadDetalle, solicitudDetalle, ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	for nombre, valor := range map[string]any{
		"contexto": contexto, "solicitud": solicitud,
		"solicitud_detalle": solicitudDetalle,
		"capacidad":         capacidad,
		"orden":             orden,
		"orden_detalle":     ordenDetalle,
	} {
		if contenido, err := json.Marshal(valor); !errors.Is(
			err, ports.ErrMaterialConsultaRRHHSensible,
		) || contenido != nil {
			t.Fatalf("%s serializable: %q, %v", nombre, contenido, err)
		}
		representacion := fmt.Sprintf("%v %#v", valor, valor)
		if strings.Contains(representacion, contexto.ActorRef()) ||
			strings.Contains(representacion, capacidad.DecisionRef()) {
			t.Fatalf("%s filtra material: %s", nombre, representacion)
		}
	}
}

func TestCapacidadConsultaRRHHNoPuedeCruzarseNiUsarseCaducada(t *testing.T) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	autoridad, contexto := autoridadYContextoPuertosRRHH(t, ahora)
	cuadro, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 10, "")
	if err != nil {
		t.Fatal(err)
	}
	detalle, err := ports.NuevaSolicitudDetalleRRHH("expediente:rrhh:001", 1)
	if err != nil {
		t.Fatal(err)
	}
	capacidad := capacidadCuadroPuertosRRHH(
		t, autoridad, contexto, cuadro, ahora,
	)
	if _, err = ports.NuevaOrdenConsultaDetalleRRHH(
		contexto, capacidad, detalle, ahora,
	); !errors.Is(err, ports.ErrOrdenConsultaRRHHInvalida) {
		t.Fatalf("capacidad de cuadro aceptada para detalle: %v", err)
	}
	otroCuadro, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 11, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ports.NuevaOrdenConsultaCuadroRRHH(
		contexto, capacidad, otroCuadro, ahora,
	); !errors.Is(err, ports.ErrOrdenConsultaRRHHInvalida) {
		t.Fatalf("capacidad aceptada para otra consulta funcional: %v", err)
	}
	if _, err = ports.NuevaOrdenConsultaCuadroRRHH(
		contexto, capacidad, cuadro, capacidad.ValidaHasta(),
	); !errors.Is(err, ports.ErrOrdenConsultaRRHHInvalida) {
		t.Fatalf("capacidad aceptada en su instante de caducidad: %v", err)
	}
}

func TestMaterialConsultaRRHHCierraAudienciaYTipoDeRecurso(t *testing.T) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	autoridad, contexto := autoridadYContextoPuertosRRHH(t, ahora)
	cuadro, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 10, "")
	if err != nil {
		t.Fatal(err)
	}
	detalle, err := ports.NuevaSolicitudDetalleRRHH(
		"expediente:rrhh:001", 1,
	)
	if err != nil {
		t.Fatal(err)
	}
	for nombre, crear := range map[string]func() error{
		"audiencia_detalle_en_cuadro": func() error {
			_, err := materialConsultaRRHHPrueba(
				t, autoridad, contexto, cuadro, ports.SolicitudDetalleRRHH{},
				ports.AccionConsultarCuadroRRHH,
				ports.FinalidadConsultarCuadroRRHH,
				ports.AudienciaConsumoConsultaDetalleRRHHV3, ahora,
			)
			return err
		},
		"audiencia_cuadro_en_detalle": func() error {
			_, err := materialConsultaRRHHPrueba(
				t, autoridad, contexto, ports.SolicitudCuadroRRHH{}, detalle,
				ports.AccionConsultarDetalleRRHH,
				ports.FinalidadConsultarDetalleRRHH,
				ports.AudienciaConsumoConsultaCuadroRRHHV3, ahora,
			)
			return err
		},
		"tipo_detalle_en_cuadro": func() error {
			_, err := materialConsultaRRHHPruebaAlterado(
				t, autoridad, contexto, cuadro, ports.SolicitudDetalleRRHH{},
				ports.AccionConsultarCuadroRRHH,
				ports.FinalidadConsultarCuadroRRHH,
				ports.AudienciaConsumoConsultaCuadroRRHHV3, ahora,
				func(r *dominiovec.RecursoAutorizable) {
					r.Tipo = ports.TipoRecursoExpediente
				},
			)
			return err
		},
		"tipo_cuadro_en_detalle": func() error {
			_, err := materialConsultaRRHHPruebaAlterado(
				t, autoridad, contexto, ports.SolicitudCuadroRRHH{}, detalle,
				ports.AccionConsultarDetalleRRHH,
				ports.FinalidadConsultarDetalleRRHH,
				ports.AudienciaConsumoConsultaDetalleRRHHV3, ahora,
				func(r *dominiovec.RecursoAutorizable) {
					r.Tipo = ports.TipoRecursoCuadroRRHH
				},
			)
			return err
		},
		"referencia_expediente_en_cuadro": func() error {
			_, err := materialConsultaRRHHPruebaAlterado(
				t, autoridad, contexto, cuadro, ports.SolicitudDetalleRRHH{},
				ports.AccionConsultarCuadroRRHH,
				ports.FinalidadConsultarCuadroRRHH,
				ports.AudienciaConsumoConsultaCuadroRRHHV3, ahora,
				func(r *dominiovec.RecursoAutorizable) {
					r.Referencia = "expediente:rrhh:otro"
				},
			)
			return err
		},
	} {
		if err := crear(); !errors.Is(
			err, ports.ErrCapacidadConsultaRRHHInvalida,
		) {
			t.Fatalf("%s aceptado: %v", nombre, err)
		}
	}
	materialHuellaAjena, err := materialConsultaRRHHPruebaAlterado(
		t, autoridad, contexto, cuadro, ports.SolicitudDetalleRRHH{},
		ports.AccionConsultarCuadroRRHH,
		ports.FinalidadConsultarCuadroRRHH,
		ports.AudienciaConsumoConsultaCuadroRRHHV3, ahora,
		func(r *dominiovec.RecursoAutorizable) {
			r.Atributos["consulta_huella_sha256"] = strings.Repeat("b", 64)
		},
	)
	if err != nil {
		t.Fatalf("material con huella declarada válida: %v", err)
	}
	if _, err := ports.NuevaCapacidadConsultaCuadroRRHH(
		contexto, materialHuellaAjena, cuadro, ahora,
	); !errors.Is(err, ports.ErrCapacidadConsultaRRHHInvalida) {
		t.Fatalf("material ligado a otra huella aceptado: %v", err)
	}
}

func TestCapacidadConsultaRRHHCaducaEnCincoSegundos(t *testing.T) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	autoridad, contexto := autoridadYContextoPuertosRRHH(t, ahora)
	cuadro, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 10, "")
	if err != nil {
		t.Fatal(err)
	}
	capacidad := capacidadCuadroPuertosRRHH(
		t, autoridad, contexto, cuadro, ahora,
	)
	if capacidad.ValidaHasta().Sub(capacidad.ValidaDesde()) !=
		ports.DuracionMaximaCapacidadConsultaRRHH {
		t.Fatalf(
			"vigencia inesperada: %s",
			capacidad.ValidaHasta().Sub(capacidad.ValidaDesde()),
		)
	}
	if _, err := ports.NuevaOrdenConsultaCuadroRRHH(
		contexto, capacidad, cuadro, capacidad.ValidaHasta(),
	); !errors.Is(err, ports.ErrOrdenConsultaRRHHInvalida) {
		t.Fatalf("capacidad aceptada al caducar: %v", err)
	}
}

func TestReciboLecturaRRHHRevalidaCapacidadAlRegistrar(t *testing.T) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	autoridad, contexto := autoridadYContextoPuertosRRHH(t, ahora)
	solicitud, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 10, "")
	if err != nil {
		t.Fatal(err)
	}
	capacidad := capacidadCuadroPuertosRRHH(
		t, autoridad, contexto, solicitud, ahora,
	)
	if _, err = ports.NuevoReciboLecturaRRHH(
		"lectura:rrhh:001", "auditoria:rrhh:001",
		contexto, capacidad, "", 0, 0, capacidad.ValidaHasta(),
	); !errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable) {
		t.Fatalf("recibo registrado con capacidad caducada: %v", err)
	}
}

func TestReciboLecturaRRHHBloqueaCodecsFmtYSlog(t *testing.T) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	autoridad, contexto := autoridadYContextoPuertosRRHH(t, ahora)
	solicitud, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 10, "")
	if err != nil {
		t.Fatal(err)
	}
	capacidad := capacidadCuadroPuertosRRHH(
		t, autoridad, contexto, solicitud, ahora,
	)
	recibo, err := ports.NuevoReciboLecturaRRHH(
		"lectura:rrhh:001", "auditoria:rrhh:001",
		contexto, capacidad, "", 0, 0, ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	for nombre, err := range map[string]error{
		"json": func() error { _, err := json.Marshal(recibo); return err }(),
		"xml":  func() error { _, err := xml.Marshal(recibo); return err }(),
		"texto": func() error {
			_, err := recibo.MarshalText()
			return err
		}(),
		"binario": func() error {
			_, err := recibo.MarshalBinary()
			return err
		}(),
		"gob_directo": func() error {
			_, err := recibo.GobEncode()
			return err
		}(),
		"cbor": func() error {
			_, err := recibo.MarshalCBOR()
			return err
		}(),
		"yaml": func() error {
			_, err := recibo.MarshalYAML()
			return err
		}(),
	} {
		if !errors.Is(err, ports.ErrMaterialConsultaRRHHSensible) {
			t.Fatalf("%s no quedó bloqueado: %v", nombre, err)
		}
	}
	var gobSalida bytes.Buffer
	if err := gob.NewEncoder(&gobSalida).Encode(recibo); !errors.Is(
		err, ports.ErrMaterialConsultaRRHHSensible,
	) {
		t.Fatalf("gob no quedó bloqueado: %v", err)
	}
	var bitacora bytes.Buffer
	slog.New(slog.NewJSONHandler(&bitacora, nil)).Info(
		"recibo", "valor", recibo,
	)
	formato := fmt.Sprintf("%+v %#v", recibo, recibo)
	if !strings.Contains(formato, "MATERIAL-CONSULTA-RRHH-OPACO") ||
		!strings.Contains(bitacora.String(), "MATERIAL-CONSULTA-RRHH-OPACO") {
		t.Fatalf(
			"recibo sin redacción contractual: fmt=%q slog=%q",
			formato, bitacora.String(),
		)
	}
	for _, sensible := range []string{
		capacidad.DecisionRef(), capacidad.CorrelacionRef(),
		contexto.SesionRef(),
	} {
		if strings.Contains(formato, sensible) ||
			strings.Contains(bitacora.String(), sensible) {
			t.Fatalf("recibo filtró %q", sensible)
		}
	}
}

func TestExportacionSQLSoloExisteEnOrdenLigada(t *testing.T) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	autoridad, contexto := autoridadYContextoPuertosRRHH(t, ahora)
	solicitud, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 10, "")
	if err != nil {
		t.Fatal(err)
	}
	material, err := materialConsultaRRHHPrueba(
		t, autoridad, contexto, solicitud, ports.SolicitudDetalleRRHH{},
		ports.AccionConsultarCuadroRRHH, ports.FinalidadConsultarCuadroRRHH,
		ports.AudienciaConsumoConsultaCuadroRRHHV3, ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	capacidad, err := ports.NuevaCapacidadConsultaCuadroRRHH(
		contexto, material, solicitud, ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	orden, err := ports.NuevaOrdenConsultaCuadroRRHH(
		contexto, capacidad, solicitud, ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	for nombre, valor := range map[string]any{
		"material": material, "capacidad": capacidad,
	} {
		if _, existe := reflect.TypeOf(valor).MethodByName(
			"ExportacionParaSQL",
		); existe {
			t.Fatalf("%s expone un atajo de exportación SQL", nombre)
		}
	}
	exportacion, err := orden.ExportacionParaSQL()
	if err != nil || exportacion.ValidarEstructura() != nil {
		t.Fatalf("orden nominal no exporta conjunto válido: %v", err)
	}
}

func TestDTOCuadroRRHHEsSerializableYRechazaHuellaCentinela(t *testing.T) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	resumen := resumenPuertosRRHH(ahora)
	contenido, err := json.Marshal(resumen)
	if err != nil || !strings.Contains(string(contenido), `"expediente_ref"`) {
		t.Fatalf("DTO no serializable: %q, %v", contenido, err)
	}
	resumen.FlujoHuella = strings.Repeat("0", 64)
	if resumen.Validar() == nil {
		t.Fatal("se aceptó la huella centinela")
	}
}

func resumenPuertosRRHH(ahora time.Time) ports.ResumenExpedienteRRHH {
	return ports.ResumenExpedienteRRHH{
		ExpedienteRef: "expediente:rrhh:001", NumeroVisible: "2026/CT-001",
		OrganizacionRef: "organizacion:diputacion-granada",
		Version:         1, FlujoRef: "flujo:rrhh:001", FlujoVersion: 1,
		FlujoHuella: strings.Repeat("a", 64), FaseClave: "solicitud",
		EstadoClave: domain.EstadoEnCurso, CentroRef: "centro:rrhh:001",
		CategoriaRef: "categoria:rrhh:001", CreadoEn: ahora,
		ActualizadoEn: ahora,
	}
}

func instantePuertosRRHH() time.Time {
	return time.Date(2026, 7, 26, 8, 0, 0, 0, time.UTC)
}

package ports_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestSolicitudCuadroRRHHValidaLimitesYCursorSinPanico(t *testing.T) {
	t.Parallel()
	for _, longitud := range []int{32, 1000, 1001, 2048} {
		longitud := longitud
		t.Run(fmt.Sprintf("cursor_%d", longitud), func(t *testing.T) {
			t.Parallel()
			if _, err := ports.NuevaSolicitudCuadroRRHH(
				"2026/CT", domain.EstadoEnCurso, "solicitud",
				100, strings.Repeat("a", longitud),
			); err != nil {
				t.Fatalf("cursor válido de %d bytes: %v", longitud, err)
			}
		})
	}
	for _, cursor := range []string{
		strings.Repeat("a", 31),
		strings.Repeat("a", 2049),
		strings.Repeat("a", 31) + "=",
	} {
		if _, err := ports.NuevaSolicitudCuadroRRHH(
			"", "", "", 1, cursor,
		); !errors.Is(err, ports.ErrSolicitudConsultaRRHHInvalida) {
			t.Fatalf("cursor hostil aceptado: longitud=%d error=%v", len(cursor), err)
		}
	}
	for _, limite := range []uint16{0, 101} {
		if _, err := ports.NuevaSolicitudCuadroRRHH(
			"", "", "", limite, "",
		); !errors.Is(err, ports.ErrSolicitudConsultaRRHHInvalida) {
			t.Fatalf("límite %d aceptado: %v", limite, err)
		}
	}
}

func TestMaterialConsultaRRHHEsRedactadoYNoSerializable(t *testing.T) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	contexto := contextoPuertosRRHH(t, ahora)
	solicitud, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 50, "")
	if err != nil {
		t.Fatal(err)
	}
	capacidad, err := ports.NuevaCapacidadConsultaRRHH(
		"decision:rrhh:001", "correlacion:rrhh:001", "motivo:rrhh:001",
		contexto, ports.AmbitoOrganizacionRRHH,
		contexto.OrganizacionRef(), ports.AccionConsultarCuadroRRHH,
		ports.FinalidadConsultarCuadroRRHH, "",
		ahora, ahora.Add(time.Minute),
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
	solicitudDetalle, err := ports.NuevaSolicitudDetalleRRHH(
		"expediente:rrhh:001", 1,
	)
	if err != nil {
		t.Fatal(err)
	}
	capacidadDetalle, err := ports.NuevaCapacidadConsultaRRHH(
		"decision:rrhh:002", "correlacion:rrhh:002", "motivo:rrhh:002",
		contexto, ports.AmbitoOrganizacionRRHH,
		contexto.OrganizacionRef(), ports.AccionConsultarDetalleRRHH,
		ports.FinalidadConsultarDetalleRRHH, solicitudDetalle.ExpedienteRef(),
		ahora, ahora.Add(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
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
	contexto := contextoPuertosRRHH(t, ahora)
	cuadro, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 10, "")
	if err != nil {
		t.Fatal(err)
	}
	detalle, err := ports.NuevaSolicitudDetalleRRHH("expediente:rrhh:001", 1)
	if err != nil {
		t.Fatal(err)
	}
	capacidad, err := ports.NuevaCapacidadConsultaRRHH(
		"decision:rrhh:001", "correlacion:rrhh:001", "motivo:rrhh:001",
		contexto, ports.AmbitoOrganizacionRRHH, contexto.OrganizacionRef(),
		ports.AccionConsultarCuadroRRHH, ports.FinalidadConsultarCuadroRRHH, "",
		ahora, ahora.Add(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ports.NuevaOrdenConsultaDetalleRRHH(
		contexto, capacidad, detalle, ahora,
	); !errors.Is(err, ports.ErrOrdenConsultaRRHHInvalida) {
		t.Fatalf("capacidad de cuadro aceptada para detalle: %v", err)
	}
	if _, err = ports.NuevaOrdenConsultaCuadroRRHH(
		contexto, capacidad, cuadro, capacidad.ValidaHasta(),
	); !errors.Is(err, ports.ErrOrdenConsultaRRHHInvalida) {
		t.Fatalf("capacidad aceptada en su instante de caducidad: %v", err)
	}
}

func TestCapacidadConsultaRRHHLimitaDuracionYAmbitoOrganizacion(t *testing.T) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	contexto, err := ports.NuevoContextoConsultaRRHH(
		"autenticacion:rrhh:001", "sesion:rrhh:001",
		"actor:rrhh:001", "perfil:rrhh:001",
		"organizacion:diputacion-granada",
		ahora, ahora.Add(10*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	crear := func(duracion time.Duration, ambitoRef string) error {
		_, err := ports.NuevaCapacidadConsultaRRHH(
			"decision:rrhh:001", "correlacion:rrhh:001", "motivo:rrhh:001",
			contexto, ports.AmbitoOrganizacionRRHH, ambitoRef,
			ports.AccionConsultarCuadroRRHH,
			ports.FinalidadConsultarCuadroRRHH, "",
			ahora, ahora.Add(duracion),
		)
		return err
	}
	if err := crear(
		ports.DuracionMaximaCapacidadConsultaRRHH,
		contexto.OrganizacionRef(),
	); err != nil {
		t.Fatalf("borde de cinco minutos rechazado: %v", err)
	}
	for nombre, err := range map[string]error{
		"excede_duracion": crear(
			ports.DuracionMaximaCapacidadConsultaRRHH+time.Microsecond,
			contexto.OrganizacionRef(),
		),
		"ambito_organizacion_ajeno": crear(
			time.Minute, "organizacion:otra",
		),
	} {
		if !errors.Is(err, ports.ErrCapacidadConsultaRRHHInvalida) {
			t.Fatalf("%s aceptado: %v", nombre, err)
		}
	}
}

func TestReciboLecturaRRHHRevalidaCapacidadAlRegistrar(t *testing.T) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	contexto, err := ports.NuevoContextoConsultaRRHH(
		"autenticacion:rrhh:001", "sesion:rrhh:001",
		"actor:rrhh:001", "perfil:rrhh:001",
		"organizacion:diputacion-granada",
		ahora, ahora.Add(10*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	capacidad, err := ports.NuevaCapacidadConsultaRRHH(
		"decision:rrhh:001", "correlacion:rrhh:001", "motivo:rrhh:001",
		contexto, ports.AmbitoOrganizacionRRHH,
		contexto.OrganizacionRef(), ports.AccionConsultarCuadroRRHH,
		ports.FinalidadConsultarCuadroRRHH, "",
		ahora, ahora.Add(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ports.NuevoReciboLecturaRRHH(
		"lectura:rrhh:001", "auditoria:rrhh:001",
		contexto, capacidad, "", 0, 0, capacidad.ValidaHasta(),
	); !errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable) {
		t.Fatalf("recibo registrado con capacidad caducada: %v", err)
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

func contextoPuertosRRHH(
	t *testing.T,
	ahora time.Time,
) ports.ContextoConsultaRRHH {
	t.Helper()
	contexto, err := ports.NuevoContextoConsultaRRHH(
		"autenticacion:rrhh:001", "sesion:rrhh:001",
		"actor:rrhh:001", "perfil:rrhh:001",
		"organizacion:diputacion-granada",
		ahora, ahora.Add(2*time.Minute),
	)
	if err != nil {
		t.Fatalf("contexto: %v", err)
	}
	return contexto
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

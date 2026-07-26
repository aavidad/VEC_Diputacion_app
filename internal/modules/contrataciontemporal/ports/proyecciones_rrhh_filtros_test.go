package ports_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestContextoConsultaRRHHConservaControlSesionYVersionPerfilSinFiltrarlos(
	t *testing.T,
) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	autoridad, contexto := autoridadYContextoPuertosRRHH(t, ahora)
	vinculo, err := autoridad.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if contexto.ControlSesionRef() != vinculo.ControlSesionRef ||
		contexto.ControlSesionRevision() != vinculo.ControlSesionRevision ||
		contexto.ControlSesionHuellaSHA256() !=
			vinculo.ControlSesionHuellaSHA256 ||
		contexto.PerfilVersion() !=
			autoridad.Resultado.Contexto.Instantanea.PerfilVersion {
		t.Fatal("el contexto no conservó exactamente la evidencia de sesión y perfil")
	}
	contenido, err := json.Marshal(contexto)
	if contenido != nil || !errors.Is(err, ports.ErrMaterialConsultaRRHHSensible) {
		t.Fatalf("contexto serializable: %q, %v", contenido, err)
	}
	var bitacora bytes.Buffer
	slog.New(slog.NewJSONHandler(&bitacora, nil)).Info(
		"contexto", "valor", contexto,
	)
	representaciones := []string{
		fmt.Sprintf("%v", contexto),
		fmt.Sprintf("%#v", contexto),
		bitacora.String(),
	}
	for _, representacion := range representaciones {
		for _, sensible := range []string{
			contexto.ControlSesionRef(),
			contexto.ControlSesionHuellaSHA256(),
			contexto.PerfilRef(),
		} {
			if strings.Contains(representacion, sensible) {
				t.Fatalf("el contexto filtró %q en %q", sensible, representacion)
			}
		}
	}
}

func TestHuellasCuadroRRHHSeparanFamiliaEstableDeConsultaExacta(
	t *testing.T,
) {
	t.Parallel()
	const (
		huellaFiltrosEsperada = "a52a2e97b7e5ad15c558d84348d8de2d629f0ed9234e586b72b23776ca63bc4d"
		huellaPagina1Esperada = "c145a15fa8ac964f779d805c159540e9cf71bb55340e78af2a7460c4f67b6e08"
		huellaPagina2Esperada = "9c13a2c2a09e6ae7257f2d22cc1dda0566369ea57cf5bc25f30b703833c0a4a8"
	)
	cursor := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0xff}, 32))
	pagina1, err := ports.NuevaSolicitudCuadroRRHH(
		"ÁREA_Ñ 2026/CT", domain.EstadoEnCurso, "solicitud", 37, "",
	)
	if err != nil {
		t.Fatal(err)
	}
	pagina2, err := ports.NuevaSolicitudCuadroRRHH(
		"ÁREA_Ñ 2026/CT", domain.EstadoEnCurso, "solicitud", 37, cursor,
	)
	if err != nil {
		t.Fatal(err)
	}
	filtros1, err := pagina1.FiltrosHuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	filtros2, err := pagina2.FiltrosHuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	consulta1, err := pagina1.HuellaCanonicaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	consulta2, err := pagina2.HuellaCanonicaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	if filtros1 != huellaFiltrosEsperada || filtros2 != huellaFiltrosEsperada {
		t.Fatalf("golden de filtros inestable: página1=%s página2=%s", filtros1, filtros2)
	}
	if consulta1 != huellaPagina1Esperada || consulta2 != huellaPagina2Esperada ||
		consulta1 == consulta2 {
		t.Fatalf(
			"golden de consulta exacta incorrecto: página1=%s página2=%s",
			consulta1, consulta2,
		)
	}

	ahora := instantePuertosRRHH()
	autoridad, contexto := autoridadYContextoPuertosRRHH(t, ahora)
	capacidad1 := capacidadCuadroPuertosRRHH(
		t, autoridad, contexto, pagina1, ahora,
	)
	orden1, err := ports.NuevaOrdenConsultaCuadroRRHH(
		contexto, capacidad1, pagina1, ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	capacidad2 := capacidadCuadroPuertosRRHH(
		t, autoridad, contexto, pagina2, ahora,
	)
	orden2, err := ports.NuevaOrdenConsultaCuadroRRHH(
		contexto, capacidad2, pagina2, ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	if orden1.FiltrosHuellaSHA256() != huellaFiltrosEsperada ||
		orden2.FiltrosHuellaSHA256() != huellaFiltrosEsperada ||
		orden1.ConsultaHuellaSHA256() != huellaPagina1Esperada ||
		orden2.ConsultaHuellaSHA256() != huellaPagina2Esperada {
		t.Fatal("las órdenes no conservaron ambas huellas canónicas")
	}
}

func TestHuellaFiltrosCuadroRRHHMutaConCadaFiltroPeroNoConCursor(t *testing.T) {
	t.Parallel()
	cursor := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0xa5}, 32))
	base, err := ports.NuevaSolicitudCuadroRRHH(
		"Área_Ñ", domain.EstadoEnCurso, "solicitud", 20, "",
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaBase, err := base.FiltrosHuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	conCursor, err := ports.NuevaSolicitudCuadroRRHH(
		"Área_Ñ", domain.EstadoEnCurso, "solicitud", 20, cursor,
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaConCursor, err := conCursor.FiltrosHuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	if huellaConCursor != huellaBase {
		t.Fatal("el cursor alteró la identidad de la familia de filtros")
	}

	casos := []struct {
		nombre string
		texto  string
		estado domain.EstadoOperativo
		fase   domain.ClaveFase
		limite uint16
	}{
		{"texto", "Área_Ó", domain.EstadoEnCurso, "solicitud", 20},
		{"estado", "Área_Ñ", domain.EstadoCompletado, "solicitud", 20},
		{"fase", "Área_Ñ", domain.EstadoEnCurso, "revision", 20},
		{"limite", "Área_Ñ", domain.EstadoEnCurso, "solicitud", 21},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			mutada, err := ports.NuevaSolicitudCuadroRRHH(
				caso.texto, caso.estado, caso.fase, caso.limite, "",
			)
			if err != nil {
				t.Fatal(err)
			}
			huella, err := mutada.FiltrosHuellaSHA256()
			if err != nil {
				t.Fatal(err)
			}
			if huella == huellaBase {
				t.Fatalf("%s no alteró la huella de filtros", caso.nombre)
			}
		})
	}
	if _, err := ports.NuevaSolicitudCuadroRRHH(
		"Área 100%", domain.EstadoEnCurso, "solicitud", 20, "",
	); !errors.Is(err, ports.ErrSolicitudConsultaRRHHInvalida) {
		t.Fatalf("se aceptó %% fuera del vocabulario permitido: %v", err)
	}
}

func TestMaterialConsultaRRHHRechazaOtroControlDeSesion(t *testing.T) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	_, contexto := autoridadYContextoPuertosRRHH(t, ahora)
	otraAutoridad := autoridadContextoPuertosRRHHPersonalizada(
		t, ahora, "a", "a", nil,
		func(autenticacion *dominiovec.AutenticacionRevalidadaV1) {
			autenticacion.ControlSesionRef = "cse_otra0123456789abcdefghijkl"
			autenticacion.ControlSesionRevision++
			autenticacion.ControlSesionHuellaSHA256 = strings.Repeat("6", 64)
		},
	)
	solicitud, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 20, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := materialConsultaRRHHPrueba(
		t, otraAutoridad, contexto, solicitud, ports.SolicitudDetalleRRHH{},
		ports.AccionConsultarCuadroRRHH, ports.FinalidadConsultarCuadroRRHH,
		ports.AudienciaConsumoConsultaCuadroRRHHV3, ahora,
	); !errors.Is(err, ports.ErrCapacidadConsultaRRHHInvalida) {
		t.Fatalf("se aceptó material ligado a otro control de sesión: %v", err)
	}
}

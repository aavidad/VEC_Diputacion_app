package ports_test

import (
	"bytes"
	"crypto/sha256"
	"encoding"
	"encoding/base64"
	"encoding/gob"
	"encoding/hex"
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

type canonRRHHParaSQL interface {
	Dominio() string
	Version() uint16
	BytesCanonicos() []byte
	HuellaSHA256() string
	fmt.Stringer
	fmt.GoStringer
	slog.LogValuer
}

func TestExportacionesCanonicasSQLConservanVectoresDorados(t *testing.T) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	autoridad, contexto := autoridadYContextoPuertosRRHH(t, ahora)

	solicitudVacia, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 25, "")
	if err != nil {
		t.Fatal(err)
	}
	ordenVacia := ordenCuadroCanonSQL(
		t, autoridad, contexto, solicitudVacia, ahora, nil,
	)
	consultaVacia, err := ordenVacia.ExportarConsultaCanonicaParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	comprobarVectorCanonSQL(
		t,
		consultaVacia,
		`{"dominio":"vec.contratacion_temporal.consulta_rrhh.cuadro.v1","version":1,"texto":"","estado_clave":"","fase_clave":"","limite":25,"cursor":""}`,
		"b76cd3ff068efd5761f1854b1ea321d36cccb109c1126a45bf77ab0d15fe53f2",
	)

	solicitudUnicode, err := ports.NuevaSolicitudCuadroRRHH(
		"Área_Ñ", domain.EstadoEnCurso, "solicitud", 20, "",
	)
	if err != nil {
		t.Fatal(err)
	}
	ordenUnicode := ordenCuadroCanonSQL(
		t, autoridad, contexto, solicitudUnicode, ahora, nil,
	)
	familiaUnicode, err := ordenUnicode.ExportarFamiliaCanonicaParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	comprobarVectorCanonSQL(
		t,
		familiaUnicode,
		`{"dominio":"vec.contratacion_temporal.filtros_rrhh.cuadro.v1","version":1,"texto":"Área_Ñ","estado_clave":"en_curso","fase_clave":"solicitud","limite":20}`,
		"fe7c99036d23c08241d81921efcd18a4c3aba32a1ae59e4a7bd0808382037f1b",
	)

	solicitudDetalle, err := ports.NuevaSolicitudDetalleRRHH(
		"expediente:contratacion:001", 7,
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
	detalle, err := ordenDetalle.ExportarConsultaCanonicaParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	comprobarVectorCanonSQL(
		t,
		detalle,
		`{"dominio":"vec.contratacion_temporal.consulta_rrhh.detalle.v1","version":1,"expediente_ref":"expediente:contratacion:001","version_observada":7}`,
		"8b8bf70873d81c8a750f935c7e6cf47af5c852639d5e4ca1ce2ba94d092d3c1a",
	)

	cursor := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0xff}, 32))
	solicitudCursor, err := ports.NuevaSolicitudCuadroRRHH(
		"ÁREA_Ñ 2026/CT", domain.EstadoEnCurso, "solicitud", 37, cursor,
	)
	if err != nil {
		t.Fatal(err)
	}
	ordenCursor := ordenCuadroCanonSQL(
		t, autoridad, contexto, solicitudCursor, ahora, nil,
	)
	consultaCursor, err := ordenCursor.ExportarConsultaCanonicaParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	comprobarVectorCanonSQL(
		t,
		consultaCursor,
		fmt.Sprintf(
			`{"dominio":"vec.contratacion_temporal.consulta_rrhh.cuadro.v1","version":1,"texto":"ÁREA_Ñ 2026/CT","estado_clave":"en_curso","fase_clave":"solicitud","limite":37,"cursor":%q}`,
			cursor,
		),
		"9c13a2c2a09e6ae7257f2d22cc1dda0566369ea57cf5bc25f30b703833c0a4a8",
	)
}

func TestExportacionCanonicaAlcanceSQLCubreAmbitosGobernados(t *testing.T) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	autoridad, contexto := autoridadYContextoPuertosRRHH(t, ahora)
	solicitud, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 25, "")
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre, clase, referencia, canon, huella string
	}{
		{
			nombre: "organizacion", clase: string(ports.AmbitoOrganizacionRRHH),
			referencia: "organizacion:diputacion-granada",
			canon:      `{"dominio":"vec.contratacion_temporal.alcance_rrhh.v1","version":1,"organizacion_ref":"organizacion:diputacion-granada","clase_ambito":"organizacion","ambito_ref":"organizacion:diputacion-granada"}`,
			huella:     "cc4772d9abe85886f10f5ee98e304ae190929de3602e69c8281bcd8e3ea7fd4b",
		},
		{
			nombre: "centro", clase: string(ports.AmbitoCentroRRHH),
			referencia: "centro:rrhh:001",
			canon:      `{"dominio":"vec.contratacion_temporal.alcance_rrhh.v1","version":1,"organizacion_ref":"organizacion:diputacion-granada","clase_ambito":"centro","ambito_ref":"centro:rrhh:001"}`,
			huella:     "f864ac24878bf8cd93701254a8a64f27dfaf4702e1b44f6ee0ddf30177abe7d9",
		},
		{
			nombre: "unidad", clase: string(ports.AmbitoUnidadGestionRRHH),
			referencia: "unidad:seleccion:001",
			canon:      `{"dominio":"vec.contratacion_temporal.alcance_rrhh.v1","version":1,"organizacion_ref":"organizacion:diputacion-granada","clase_ambito":"unidad_gestion","ambito_ref":"unidad:seleccion:001"}`,
			huella:     "0e9140f36357f299b8f2606ff07e1e74e12cdb3cfe97839da8ad372ba733b6d0",
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			orden := ordenCuadroCanonSQL(
				t, autoridad, contexto, solicitud, ahora,
				func(recurso *dominiovec.RecursoAutorizable) {
					recurso.Ambitos["clase_ambito"] = caso.clase
					recurso.Ambitos["ambito_ref"] = caso.referencia
					recurso.Referencia = caso.referencia
				},
			)
			alcance, err := orden.ExportarAlcanceCanonicoParaSQL()
			if err != nil {
				t.Fatal(err)
			}
			comprobarVectorCanonSQL(t, alcance, caso.canon, caso.huella)
		})
	}
}

func TestExportacionesCanonicasSQLSonNominalesDefensivasYMutablesSoloPorEntrada(
	t *testing.T,
) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	autoridad, contexto := autoridadYContextoPuertosRRHH(t, ahora)
	cursor := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0xa5}, 32))
	sinCursor, err := ports.NuevaSolicitudCuadroRRHH(
		"Área_Ñ", domain.EstadoEnCurso, "solicitud", 20, "",
	)
	if err != nil {
		t.Fatal(err)
	}
	conCursor, err := ports.NuevaSolicitudCuadroRRHH(
		"Área_Ñ", domain.EstadoEnCurso, "solicitud", 20, cursor,
	)
	if err != nil {
		t.Fatal(err)
	}
	ordenSinCursor := ordenCuadroCanonSQL(
		t, autoridad, contexto, sinCursor, ahora, nil,
	)
	ordenConCursor := ordenCuadroCanonSQL(
		t, autoridad, contexto, conCursor, ahora, nil,
	)
	consultaSinCursor, err := ordenSinCursor.ExportarConsultaCanonicaParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	consultaConCursor, err := ordenConCursor.ExportarConsultaCanonicaParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	familiaSinCursor, err := ordenSinCursor.ExportarFamiliaCanonicaParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	familiaConCursor, err := ordenConCursor.ExportarFamiliaCanonicaParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	alcance, err := ordenSinCursor.ExportarAlcanceCanonicoParaSQL()
	if err != nil {
		t.Fatal(err)
	}

	if consultaSinCursor.HuellaSHA256() == consultaConCursor.HuellaSHA256() {
		t.Fatal("el cursor no mutó la consulta exacta")
	}
	if familiaSinCursor.HuellaSHA256() != familiaConCursor.HuellaSHA256() ||
		!bytes.Equal(
			familiaSinCursor.BytesCanonicos(),
			familiaConCursor.BytesCanonicos(),
		) {
		t.Fatal("el cursor mutó la familia de filtros")
	}
	primera := consultaConCursor.BytesCanonicos()
	original := bytes.Clone(primera)
	clear(primera)
	if !bytes.Equal(consultaConCursor.BytesCanonicos(), original) {
		t.Fatal("la persona consumidora pudo mutar el canon conservado")
	}

	tipos := []reflect.Type{
		reflect.TypeOf(consultaSinCursor),
		reflect.TypeOf(familiaSinCursor),
		reflect.TypeOf(alcance),
	}
	solicitudDetalle, err := ports.NuevaSolicitudDetalleRRHH(
		"expediente:contratacion:001", 7,
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
	detalle, err := ordenDetalle.ExportarConsultaCanonicaParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	tipos = append(tipos, reflect.TypeOf(detalle))
	for izquierda := range tipos {
		for derecha := izquierda + 1; derecha < len(tipos); derecha++ {
			if tipos[izquierda] == tipos[derecha] {
				t.Fatalf(
					"dos canones distintos comparten tipo nominal: %v",
					tipos[izquierda],
				)
			}
		}
	}
	dominios := map[string]struct{}{}
	for _, exportacion := range []canonRRHHParaSQL{
		consultaSinCursor, familiaSinCursor, detalle, alcance,
	} {
		if _, repetido := dominios[exportacion.Dominio()]; repetido {
			t.Fatalf("dominio canónico reutilizado: %s", exportacion.Dominio())
		}
		dominios[exportacion.Dominio()] = struct{}{}
		if exportacion.Version() != 1 {
			t.Fatalf("versión canónica inesperada: %d", exportacion.Version())
		}
	}
}

func TestExportacionesCanonicasSQLNoFiltranPIINiSeSerializan(t *testing.T) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	autoridad, contexto := autoridadYContextoPuertosRRHH(t, ahora)
	solicitud, err := ports.NuevaSolicitudCuadroRRHH(
		"Área_Ñ", domain.EstadoEnCurso, "solicitud", 20, "",
	)
	if err != nil {
		t.Fatal(err)
	}
	orden := ordenCuadroCanonSQL(
		t, autoridad, contexto, solicitud, ahora, nil,
	)
	consulta, err := orden.ExportarConsultaCanonicaParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	familia, err := orden.ExportarFamiliaCanonicaParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	alcance, err := orden.ExportarAlcanceCanonicoParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	solicitudDetalle, err := ports.NuevaSolicitudDetalleRRHH(
		"expediente:contratacion:001", 7,
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
	detalle, err := ordenDetalle.ExportarConsultaCanonicaParaSQL()
	if err != nil {
		t.Fatal(err)
	}

	sensibles := []string{
		contexto.AutenticacionRef(), contexto.SesionRef(),
		contexto.ControlSesionRef(), contexto.ControlSesionHuellaSHA256(),
		contexto.ActorRef(), contexto.PerfilRef(),
		orden.Capacidad().DecisionRef(),
		orden.Capacidad().CapacidadHuellaSHA256(),
		orden.Capacidad().MaterialHuellaSHA256(),
	}
	for nombre, exportacion := range map[string]canonRRHHParaSQL{
		"consulta": consulta, "familia": familia,
		"detalle": detalle, "alcance": alcance,
	} {
		canon := string(exportacion.BytesCanonicos())
		for _, sensible := range sensibles {
			if strings.Contains(canon, sensible) {
				t.Fatalf("%s filtró material sensible en el canon", nombre)
			}
		}
		comprobarOpacidadCanonSQL(t, nombre, exportacion, sensibles)
	}

	var campos map[string]any
	if err := json.Unmarshal(alcance.BytesCanonicos(), &campos); err != nil {
		t.Fatal(err)
	}
	permitidos := map[string]bool{
		"dominio": true, "version": true, "organizacion_ref": true,
		"clase_ambito": true, "ambito_ref": true,
	}
	for campo := range campos {
		if !permitidos[campo] {
			t.Fatalf("alcance filtró un campo no permitido: %s", campo)
		}
	}
	if len(campos) != len(permitidos) {
		t.Fatalf("alcance incompleto: %#v", campos)
	}
}

func comprobarVectorCanonSQL(
	t *testing.T,
	exportacion canonRRHHParaSQL,
	canonEsperado, huellaEsperada string,
) {
	t.Helper()
	if canon := string(exportacion.BytesCanonicos()); canon != canonEsperado {
		t.Fatalf("canon inestable:\nobtenido: %s\nesperado: %s", canon, canonEsperado)
	}
	suma := sha256.Sum256([]byte(canonEsperado))
	if calculada := hex.EncodeToString(suma[:]); calculada != huellaEsperada {
		t.Fatalf("vector de prueba incoherente: %s", calculada)
	}
	if exportacion.HuellaSHA256() != huellaEsperada {
		t.Fatalf(
			"huella inestable: obtenida=%s esperada=%s",
			exportacion.HuellaSHA256(), huellaEsperada,
		)
	}
}

func comprobarOpacidadCanonSQL(
	t *testing.T,
	nombre string,
	exportacion canonRRHHParaSQL,
	sensibles []string,
) {
	t.Helper()
	if contenido, err := json.Marshal(exportacion); contenido != nil ||
		!errors.Is(err, ports.ErrMaterialConsultaRRHHSensible) {
		t.Fatalf("%s serializable como JSON: %q, %v", nombre, contenido, err)
	}
	if contenido, err := xml.Marshal(exportacion); len(contenido) != 0 ||
		!errors.Is(err, ports.ErrMaterialConsultaRRHHSensible) {
		t.Fatalf("%s serializable como XML: %q, %v", nombre, contenido, err)
	}
	texto, correcto := exportacion.(encoding.TextMarshaler)
	if !correcto {
		t.Fatalf("%s no bloquea MarshalText", nombre)
	}
	if contenido, err := texto.MarshalText(); contenido != nil ||
		!errors.Is(err, ports.ErrMaterialConsultaRRHHSensible) {
		t.Fatalf("%s serializable como texto: %q, %v", nombre, contenido, err)
	}
	binario, correcto := exportacion.(encoding.BinaryMarshaler)
	if !correcto {
		t.Fatalf("%s no bloquea MarshalBinary", nombre)
	}
	if contenido, err := binario.MarshalBinary(); contenido != nil ||
		!errors.Is(err, ports.ErrMaterialConsultaRRHHSensible) {
		t.Fatalf("%s serializable como binario: %q, %v", nombre, contenido, err)
	}
	var gobSalida bytes.Buffer
	if err := gob.NewEncoder(&gobSalida).Encode(exportacion); !errors.Is(
		err, ports.ErrMaterialConsultaRRHHSensible,
	) {
		t.Fatalf("%s serializable como gob: %v", nombre, err)
	}
	var bitacora bytes.Buffer
	slog.New(slog.NewJSONHandler(&bitacora, nil)).Info(
		"canon", "valor", exportacion,
	)
	representaciones := []string{
		fmt.Sprintf("%v", exportacion),
		fmt.Sprintf("%#v", exportacion),
		bitacora.String(),
	}
	for _, representacion := range representaciones {
		for _, sensible := range sensibles {
			if strings.Contains(representacion, sensible) {
				t.Fatalf("%s filtró %q en %q", nombre, sensible, representacion)
			}
		}
	}
}

func ordenCuadroCanonSQL(
	t *testing.T,
	autoridad ports.ContextoAutorizacionAltaV3,
	contexto ports.ContextoConsultaRRHH,
	solicitud ports.SolicitudCuadroRRHH,
	instante time.Time,
	mutarRecurso func(*dominiovec.RecursoAutorizable),
) ports.OrdenConsultaCuadroRRHH {
	t.Helper()
	material, err := materialConsultaRRHHPruebaAlterado(
		t, autoridad, contexto, solicitud, ports.SolicitudDetalleRRHH{},
		ports.AccionConsultarCuadroRRHH, ports.FinalidadConsultarCuadroRRHH,
		ports.AudienciaConsumoConsultaCuadroRRHHV3, instante, mutarRecurso,
	)
	if err != nil {
		t.Fatal(err)
	}
	capacidad, err := ports.NuevaCapacidadConsultaCuadroRRHH(
		contexto, material, solicitud, instante,
	)
	if err != nil {
		t.Fatal(err)
	}
	orden, err := ports.NuevaOrdenConsultaCuadroRRHH(
		contexto, capacidad, solicitud, instante,
	)
	if err != nil {
		t.Fatal(err)
	}
	return orden
}

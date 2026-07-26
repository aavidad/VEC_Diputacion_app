package cobertura

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestIdentidadOperacionDecisionCoberturaEsOpacaEnTodasLasFronteras(
	t *testing.T,
) {
	identidad := identidadRectificacionDecisionCoberturaPrueba()
	if _, err := json.Marshal(identidad); !errors.Is(
		err, ErrSerializacionOperacionDecisionCoberturaProhibida,
	) {
		t.Fatalf("identidad serializable como JSON: %v", err)
	}

	formateada := fmt.Sprintf("%v %#v", identidad, identidad)
	redaccionEsperada := redaccionOperacionDecisionCobertura + " " +
		redaccionOperacionDecisionCobertura
	if formateada != redaccionEsperada {
		t.Fatalf("identidad sin redacción estable: %q", formateada)
	}

	var registro bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&registro, nil))
	logger.Info("prueba", "identidad", identidad)
	salida := registro.String()
	for _, secreto := range []string{
		identidad.claveIdempotencia,
		identidad.actorRef,
		identidad.perfilRef,
		identidad.predecesoraRef,
		identidad.predecesoraHuella,
		string(identidad.motivo.ClaveI18n),
	} {
		if strings.Contains(salida, secreto) {
			t.Fatalf("slog filtró material opaco %q en %q", secreto, salida)
		}
	}
	if !strings.Contains(salida, redaccionOperacionDecisionCobertura) {
		t.Fatalf("slog no aplicó la redacción: %q", salida)
	}

	tipo := reflect.TypeOf(identidad)
	for indice := 0; indice < tipo.NumField(); indice++ {
		campo := tipo.Field(indice)
		if campo.IsExported() {
			t.Fatalf("identidad expuso el campo %s a reflexión", campo.Name)
		}
	}
	for indice := 0; indice < tipo.NumMethod(); indice++ {
		nombre := strings.ToLower(tipo.Method(indice).Name)
		for _, prohibido := range []string{
			"identidad", "actor", "perfil", "motivo", "predecesora", "clave",
		} {
			if strings.Contains(nombre, prohibido) {
				t.Fatalf("identidad expuso el método sensible %s", nombre)
			}
		}
	}
}

func TestConsultaOperacionDecisionCoberturaNoExponeIdentidad(t *testing.T) {
	consulta, _ := solicitudReservaOperacionDecisionCoberturaPrueba(
		t, identidadOperacionDecisionCoberturaPrueba(),
	)
	tipo := reflect.TypeOf(consulta)
	if _, existe := tipo.MethodByName("Identidad"); existe {
		t.Fatal("la consulta conserva un getter público de identidad")
	}
	for indice := 0; indice < tipo.NumField(); indice++ {
		if tipo.Field(indice).IsExported() {
			t.Fatalf(
				"la consulta expuso el campo %s",
				tipo.Field(indice).Name,
			)
		}
	}
}

func TestProyeccionPersistenciaOperacionDecisionCoberturaEsMinima(
	t *testing.T,
) {
	identidad := identidadRectificacionDecisionCoberturaPrueba()
	_, solicitud := solicitudReservaOperacionDecisionCoberturaPrueba(
		t, identidad,
	)
	datos, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if datos.OrganizacionRef != identidad.organizacionRef ||
		datos.ExpedienteRef != identidad.expedienteRef ||
		datos.VersionExpediente != identidad.versionExpediente ||
		datos.AmbitoIdempotenciaHMAC == "" ||
		datos.HuellaSemanticaHMAC == "" ||
		!huellaSHA256OperacionDecisionCoberturaValida(
			datos.TokenPropietarioSHA256,
		) {
		t.Fatalf("proyección mínima incoherente: %v", datos)
	}
	if _, err := json.Marshal(datos); !errors.Is(
		err, ErrSerializacionOperacionDecisionCoberturaProhibida,
	) {
		t.Fatalf("proyección serializable fuera del adaptador: %v", err)
	}

	tipo := reflect.TypeOf(datos)
	for indice := 0; indice < tipo.NumField(); indice++ {
		nombre := strings.ToLower(tipo.Field(indice).Name)
		for _, prohibido := range []string{
			"clave", "actor", "perfil", "motivo", "predecesora", "identidad",
		} {
			if strings.Contains(nombre, prohibido) {
				t.Fatalf(
					"persistencia recibió el campo prohibido %s",
					tipo.Field(indice).Name,
				)
			}
		}
	}
}

func TestCapacidadVECOperacionDecisionCoberturaConstruyeIdentidadValida(
	t *testing.T,
) {
	base := identidadOperacionDecisionCoberturaPrueba()
	contexto, solicitudContexto :=
		contextoAutorizacionOperacionDecisionCoberturaPrueba(t)
	identidad, err := NuevaIdentidadOperacionDecisionCobertura(
		base.claveIdempotencia,
		base.tipo,
		base.organizacionRef,
		base.expedienteRef,
		base.versionExpediente,
		contexto,
		solicitudContexto,
		instanteOperacionDecisionCoberturaPrueba,
		base.accion,
		base.viaElegida,
		base.identidadSemantica,
		base.motivo,
		base.predecesoraRef,
		base.predecesoraHuella,
	)
	if err != nil || identidad.Validar() != nil {
		t.Fatalf("identidad del orquestador rechazada: %v", err)
	}
	datosVinculo, err := contexto.Vinculo.Datos()
	if err != nil || identidad.actorRef != datosVinculo.PrincipalID ||
		identidad.perfilRef != datosVinculo.PerfilActivoRef {
		t.Fatal("actor o perfil no se derivaron de la capacidad VEC")
	}

	adulterado := contexto
	adulterado.Resultado.HuellaSHA256 = strings.Repeat("8", 64)
	if _, err := NuevaIdentidadOperacionDecisionCobertura(
		base.claveIdempotencia,
		base.tipo,
		base.organizacionRef,
		base.expedienteRef,
		base.versionExpediente,
		adulterado,
		solicitudContexto,
		instanteOperacionDecisionCoberturaPrueba,
		base.accion,
		base.viaElegida,
		base.identidadSemantica,
		base.motivo,
		base.predecesoraRef,
		base.predecesoraHuella,
	); err == nil {
		t.Fatal("identidad aceptada con resultado VEC adulterado")
	}
	if _, err := NuevaIdentidadOperacionDecisionCobertura(
		base.claveIdempotencia,
		base.tipo,
		base.organizacionRef,
		base.expedienteRef,
		base.versionExpediente,
		ports.ContextoAutorizacionAltaV3{},
		solicitudContexto,
		instanteOperacionDecisionCoberturaPrueba,
		base.accion,
		base.viaElegida,
		base.identidadSemantica,
		base.motivo,
		base.predecesoraRef,
		base.predecesoraHuella,
	); err == nil {
		t.Fatal("identidad aceptada con capacidad VEC cero")
	}
}

func TestSuperficieOperacionDecisionCoberturaNoTieneConstructorActorDesdeTexto(
	t *testing.T,
) {
	_, ficheroPrueba, _, correcto := runtime.Caller(0)
	if !correcto {
		t.Fatal("no se localizó el fichero de prueba")
	}
	ruta := filepath.Join(
		filepath.Dir(ficheroPrueba),
		"operacion_decision_idempotencia_canon.go",
	)
	archivo, err := parser.ParseFile(token.NewFileSet(), ruta, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, declaracion := range archivo.Decls {
		funcion, esFuncion := declaracion.(*ast.FuncDecl)
		if esFuncion && strings.Contains(
			funcion.Name.Name,
			"ContextoServidorOperacionDecisionCobertura",
		) {
			t.Fatalf(
				"permanece el constructor raw %s",
				funcion.Name.Name,
			)
		}
	}
}

func TestVersionCatalogoOperacionDecisionCoberturaRespetaEnteroSeguro(
	t *testing.T,
) {
	if strconv.IntSize < 64 {
		t.Skip("la arquitectura no puede representar el límite uint53")
	}
	datos := identidadRectificacionDecisionCoberturaPrueba()
	datos.motivo.ReferenciaCatalogo.CatalogoVersion =
		int(MaximoEnteroSeguroOperacionDecisionCobertura)
	if datos.Validar() != nil {
		t.Fatal("se rechazó el máximo entero interoperable")
	}
	if _, err := NuevasPreimagenesOperacionDecisionCobertura(datos); err != nil {
		t.Fatalf("no se canonizó el máximo interoperable: %v", err)
	}

	datos.motivo.ReferenciaCatalogo.CatalogoVersion =
		int(MaximoEnteroSeguroOperacionDecisionCobertura) + 1
	if datos.Validar() == nil {
		t.Fatal("se aceptó una versión de catálogo superior a 2^53-1")
	}
	if _, err := NuevasPreimagenesOperacionDecisionCobertura(datos); !errors.Is(
		err, ErrOperacionDecisionCoberturaIdempotenteInvalida,
	) {
		t.Fatalf("el canon aceptó una versión no interoperable: %v", err)
	}
}

func TestDenegacionVECNoTieneCamposDeActuacionNiEfectoC2(t *testing.T) {
	tipo := reflect.TypeOf(ResultadoDenegadoVECOperacionDecisionCobertura{})
	if tipo.NumField() != 0 {
		t.Fatal("la rama denegada conserva campos de actuación o efecto C2")
	}

	_, solicitud := solicitudReservaOperacionDecisionCoberturaPrueba(
		t, identidadOperacionDecisionCoberturaPrueba(),
	)
	recibo := reciboDenegadoVECOperacionDecisionCoberturaPrueba(t, solicitud)
	if recibo.Aplicada != nil || recibo.DenegadaVEC == nil {
		t.Fatal("la denegación de prueba contiene efecto C2")
	}
}

func TestCodigoVECOperacionDecisionCoberturaNoSeSustituyePorCatalogoLocal(
	t *testing.T,
) {
	_, solicitud := solicitudReservaOperacionDecisionCoberturaPrueba(
		t, identidadOperacionDecisionCoberturaPrueba(),
	)
	recibo := reciboOperacionDecisionCoberturaPrueba(t, solicitud)
	recibo.CodigoProbatorioVEC = string(domain.ClaveCatalogo("aceptada_por_rrhh"))
	if recibo.ValidarPara(solicitud.consulta) == nil {
		t.Fatal("se aceptó un código local inventado como resultado VEC V3")
	}
}

func TestReciboOperacionDecisionCoberturaNoPrecedeObservacionDurable(
	t *testing.T,
) {
	_, solicitud := solicitudReservaOperacionDecisionCoberturaPrueba(
		t, identidadOperacionDecisionCoberturaPrueba(),
	)
	reserva := datosPropiedadOperacionDecisionCoberturaPrueba(t, solicitud)
	recibo := reciboOperacionDecisionCoberturaPrueba(t, solicitud)

	recibo.ConfirmadaEn = reserva.ObservadaEnDB
	if err := recibo.ValidarParaReservaCongelada(
		solicitud,
		reserva,
	); err != nil {
		t.Fatalf("se rechazó el límite temporal inclusivo: %v", err)
	}

	recibo.ConfirmadaEn = reserva.ObservadaEnDB.Add(-time.Microsecond)
	if err := recibo.ValidarParaReservaCongelada(
		solicitud,
		reserva,
	); !errors.Is(err, ErrOperacionDecisionCoberturaIdempotenteInvalida) {
		t.Fatalf("se aceptó un recibo anterior a la observación durable: %v", err)
	}
}

package ports

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"runtime"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type preparadorSolicitudesFuentesAnalisisO3Doble struct {
	solicitudes SolicitudesFuentesAnalisisO3
	err         error
	llamadas    int
}

func (d *preparadorSolicitudesFuentesAnalisisO3Doble) PrepararSolicitudesFuentesAnalisisO3(
	_ context.Context,
	_ SolicitudPrepararArtefactoAnalisis,
) (SolicitudesFuentesAnalisisO3, error) {
	d.llamadas++
	return d.solicitudes, d.err
}

// capacidadPrepararArtefactoAnalisisO3Prueba replica en este paquete sólo la
// secuencia necesaria para probar los invariantes privados del tipo opaco. La
// capacidad productiva pertenece a application para preservar la arquitectura.
type capacidadPrepararArtefactoAnalisisO3Prueba struct {
	solicitudes PreparadorSolicitudesFuentesAnalisisO3
	fuenteRC    FuentePresupuestaria
	calculador  CalculadorCostePersonal
	verificador VerificadorRespuestaFuenteAnalisis
	publicador  VerificadorPublicacionMotivoFuenteAnalisis
	confianza   ConfianzaAutoridadesFuenteAnalisis
	reloj       RelojFuenteAnalisis
}

func (c *capacidadPrepararArtefactoAnalisisO3Prueba) PrepararArtefactoAnalisis(
	ctx context.Context,
	solicitud SolicitudPrepararArtefactoAnalisis,
) (ArtefactoAnalisisPreparado, error) {
	if c == nil || ctx == nil || solicitud.Validar() != nil {
		return ArtefactoAnalisisPreparado{},
			ErrSolicitudArtefactoAnalisisInvalida
	}
	if err := ctx.Err(); err != nil {
		return ArtefactoAnalisisPreparado{},
			errors.Join(ErrArtefactoAnalisisNoDisponible, err)
	}
	operacion, cancelar := context.WithTimeout(
		ctx,
		TiempoMaximoFuenteAnalisis,
	)
	defer cancelar()
	solicitudes, err := c.solicitudes.PrepararSolicitudesFuentesAnalisisO3(
		operacion,
		solicitud,
	)
	if err != nil || solicitudes.ValidarPara(solicitud) != nil {
		return ArtefactoAnalisisPreparado{},
			ErrArtefactoAnalisisNoConfiable
	}
	rc, err := VerificarValidacionRCConFuenteO3(
		operacion,
		c.fuenteRC,
		c.verificador,
		c.publicador,
		c.confianza,
		c.reloj,
		solicitudes.ValidacionRC,
	)
	if err != nil {
		return ArtefactoAnalisisPreparado{}, err
	}
	var coste EvidenciaCalculoCosteVerificadaO3
	if solicitudes.CalculoCoste != nil {
		coste, err = VerificarCalculoCosteConFuenteO3(
			operacion,
			c.calculador,
			c.verificador,
			c.confianza,
			c.reloj,
			*solicitudes.CalculoCoste,
		)
		if err != nil {
			return ArtefactoAnalisisPreparado{}, err
		}
	}
	comprobadaEn := c.reloj.Ahora()
	if err := operacion.Err(); err != nil {
		return ArtefactoAnalisisPreparado{},
			errors.Join(ErrArtefactoAnalisisNoDisponible, err)
	}
	if rc.ValidarEn(comprobadaEn) != nil ||
		coste.ValidarEn(comprobadaEn) != nil ||
		RevalidarEvidenciasFuenteAnalisisO3(
			operacion,
			c.fuenteRC,
			c.calculador,
			c.verificador,
			c.publicador,
			c.confianza,
			rc,
			coste,
			comprobadaEn,
		) != nil {
		return ArtefactoAnalisisPreparado{},
			ErrArtefactoAnalisisNoConfiable
	}
	preparadoEn := c.reloj.Ahora()
	return NuevoArtefactoAnalisisVerificadoO3(
		solicitud,
		rc,
		coste,
		preparadoEn,
	)
}

type fuenteRCConCredencialCambiantePrueba struct {
	presentador presentadorAutoridadConfiguradoPrueba
	resultado   ResultadoValidacionRC
	presentadas int
}

func (f *fuenteRCConCredencialCambiantePrueba) PresentarAutoridadFuenteAnalisis(
	ctx context.Context,
	desafio DesafioAutoridadFuenteAnalisis,
) (PresentacionAutoridadFuenteAnalisis, error) {
	f.presentadas++
	presentador := f.presentador
	if f.presentadas > 1 {
		presentador.datos.Serie++
	}
	return presentador.PresentarAutoridadFuenteAnalisis(ctx, desafio)
}

func (f *fuenteRCConCredencialCambiantePrueba) ValidarRC(
	context.Context,
	SolicitudValidarRC,
) (ResultadoValidacionRC, error) {
	return f.resultado, nil
}

func TestPuenteArtefactoAnalisisRechazaCambioDeCredencialEnRevalidacion(
	t *testing.T,
) {
	solicitud, capacidad := capacidadArtefactoAnalisisPrueba(t)
	solicitudes, err := capacidad.solicitudes.
		PrepararSolicitudesFuentesAnalisisO3(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := capacidad.fuenteRC.ValidarRC(
		context.Background(),
		solicitudes.ValidacionRC,
	)
	if err != nil {
		t.Fatal(err)
	}
	capacidad.fuenteRC = &fuenteRCConCredencialCambiantePrueba{
		presentador: nuevoPresentadorAutoridadConfiguradoPrueba(
			RolFuentePresupuestaria,
			"fuente_presupuesto_0123456789",
			"backend_presupuesto_0123456789",
		),
		resultado: resultado,
	}
	if _, err := capacidad.PrepararArtefactoAnalisis(
		context.Background(),
		solicitud,
	); !errors.Is(err, ErrArtefactoAnalisisNoConfiable) {
		t.Fatalf("credencial cambiada aceptada: %v", err)
	}
}

func TestPuenteArtefactoAnalisisRechazaCatalogoAdulteradoAntesDeConsumir(
	t *testing.T,
) {
	solicitud, capacidad := capacidadArtefactoAnalisisPrueba(t)
	solicitudes, err := capacidad.solicitudes.
		PrepararSolicitudesFuentesAnalisisO3(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	inicio := instanteFuenteAnalisisPrueba()
	validacion := validacionRCNegativaPrueba(
		t,
		solicitudes.ValidacionRC,
		inicio.Add(time.Second),
	)
	metadatos := metadatosRespuestaPrueba(
		validacion.FuenteRef,
		validacion.ReciboRef,
		inicio,
	)
	resultado := resultadoRCFirmadoPrueba(
		t,
		solicitudes.ValidacionRC,
		validacion,
		motivoFuenteAnalisisPrueba(t),
		metadatos,
	)
	capacidad.fuenteRC = fuentePresupuestariaDoble(func(
		context.Context,
		SolicitudValidarRC,
	) (ResultadoValidacionRC, error) {
		return resultado, nil
	})
	capacidad.publicador = verificadorPublicacionMotivoDoble(func(
		_ context.Context,
		peticion SolicitudVerificarPublicacionMotivoFuenteAnalisis,
	) (ConfirmacionPublicacionMotivoFuenteAnalisis, error) {
		confirmacion, err := NuevaConfirmacionPublicacionMotivoFuenteAnalisis(
			peticion,
			"publicador_catalogo_motivos_012345",
			"publicacion_catalogo_motivos_rc_012345",
			"recibo_verificacion_catalogo_012345",
			inicio.Add(2500*time.Millisecond),
		)
		if err != nil {
			return ConfirmacionPublicacionMotivoFuenteAnalisis{}, err
		}
		ultimo := "0"
		if confirmacion.datos.HuellaSolicitudSHA256[63] == '0' {
			ultimo = "1"
		}
		confirmacion.datos.HuellaSolicitudSHA256 =
			confirmacion.datos.HuellaSolicitudSHA256[:63] + ultimo
		return confirmacion, nil
	})
	if _, err := capacidad.PrepararArtefactoAnalisis(
		context.Background(),
		solicitud,
	); !errors.Is(err, ErrVerificacionFuenteAnalisisNoDisponible) {
		t.Fatalf("catálogo adulterado aceptado: %v", err)
	}
}

func TestArtefactoAnalisisRechazaUsoCuandoLaFuenteHaCaducado(
	t *testing.T,
) {
	solicitud, capacidad := capacidadArtefactoAnalisisPrueba(t)
	artefacto, err := capacidad.PrepararArtefactoAnalisis(
		context.Background(),
		solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := artefacto.ValidarVigenciaEn(
		solicitud,
		instanteFuenteAnalisisPrueba().Add(24*time.Hour),
	); !errors.Is(err, ErrArtefactoAnalisisNoConfiable) {
		t.Fatalf("respuesta caducada aceptada: %v", err)
	}
}

func TestArtefactoAnalisisRechazaCualquierAdulteracionDeLaPrueba(
	t *testing.T,
) {
	solicitud, capacidad := capacidadArtefactoAnalisisPrueba(t)
	artefacto := prepararArtefactoAnalisisPrueba(
		t,
		capacidad,
		solicitud,
	)
	casos := []struct {
		nombre    string
		modificar func(*ArtefactoAnalisisPreparado)
	}{
		{"organizacion", func(a *ArtefactoAnalisisPreparado) {
			a.datos.OrganizacionRef = "organizacion_adulterada_012345"
		}},
		{"huella", func(a *ArtefactoAnalisisPreparado) {
			a.datos.HuellaRespuestaRC = a.datos.HuellaRespuestaCoste
		}},
		{"atestacion", func(a *ArtefactoAnalisisPreparado) {
			a.datos.SelloRespuestaRCHMAC += "a"
		}},
		{"confirmacion", func(a *ArtefactoAnalisisPreparado) {
			a.datos.ConfirmadaRCEn = a.datos.ConfirmadaRCEn.Add(
				time.Microsecond,
			)
		}},
		{"credencial", func(a *ArtefactoAnalisisPreparado) {
			a.datos.AutoridadFuenteRC.Generacion++
		}},
		{"orden_consumo_pendiente", func(a *ArtefactoAnalisisPreparado) {
			a.pruebas.ordenConjunto.datos.huellaSHA256 =
				string(make([]byte, 64))
		}},
		{"coste", func(a *ArtefactoAnalisisPreparado) {
			a.datos.CostePrevisto.Centimos++
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			adulterado := clonarArtefactoAnalisisPrueba(artefacto)
			caso.modificar(&adulterado)
			if _, err := adulterado.DatosPara(solicitud); !errors.Is(
				err,
				ErrArtefactoAnalisisNoConfiable,
			) {
				t.Fatalf("prueba adulterada aceptada: %v", err)
			}
		})
	}
}

func TestArtefactoAnalisisNoTieneConstructorNominalExportado(
	t *testing.T,
) {
	tipo := reflect.TypeOf(ArtefactoAnalisisPreparado{})
	if campo, existe := tipo.FieldByName("datos"); !existe ||
		campo.IsExported() {
		t.Fatal("la autoridad nominal quedó expuesta")
	}
	nominal := ArtefactoAnalisisPreparado{}
	if _, err := nominal.DatosPara(
		SolicitudPrepararArtefactoAnalisis{},
	); !errors.Is(err, ErrArtefactoAnalisisNoConfiable) {
		t.Fatalf("artefacto nominal aceptado: %v", err)
	}
	_, fichero, _, correcto := runtime.Caller(0)
	if !correcto {
		t.Fatal("no se pudo resolver el fichero de prueba")
	}
	archivo, err := parser.ParseFile(
		token.NewFileSet(),
		fichero[:len(fichero)-len("_test.go")]+".go",
		nil,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, declaracion := range archivo.Decls {
		funcion, esFuncion := declaracion.(*ast.FuncDecl)
		if esFuncion &&
			funcion.Name.Name == "NuevoArtefactoAnalisisPreparado" {
			t.Fatal("reapareció el constructor nominal público")
		}
	}
}

func TestArtefactoAnalisisBloqueaTodosLosCodecs(t *testing.T) {
	solicitud, capacidad := capacidadArtefactoAnalisisPrueba(t)
	artefacto := prepararArtefactoAnalisisPrueba(
		t,
		capacidad,
		solicitud,
	)
	var err error
	comprobar := func(nombre string, err error) {
		t.Helper()
		if !errors.Is(err, ErrSerializacionOperacionAnalisisProhibida) {
			t.Fatalf("%s no quedó bloqueado: %v", nombre, err)
		}
	}
	_, err = json.Marshal(artefacto)
	comprobar("json", err)
	_, err = xml.Marshal(artefacto)
	comprobar("xml", err)
	_, err = artefacto.MarshalText()
	comprobar("text", err)
	_, err = artefacto.MarshalBinary()
	comprobar("binary", err)
	var destino bytes.Buffer
	comprobar("gob", gob.NewEncoder(&destino).Encode(artefacto))
	_, err = artefacto.MarshalCBOR()
	comprobar("cbor", err)
	_, err = artefacto.MarshalYAML()
	comprobar("yaml", err)

	var reconstruido ArtefactoAnalisisPreparado
	comprobar("json_decode", json.Unmarshal([]byte(`{}`), &reconstruido))
	comprobar("xml_decode", xml.Unmarshal([]byte(`<a/>`), &reconstruido))
	comprobar("text_decode", reconstruido.UnmarshalText(nil))
	comprobar("binary_decode", reconstruido.UnmarshalBinary(nil))
	comprobar("gob_decode", reconstruido.GobDecode(nil))
	comprobar("cbor_decode", reconstruido.UnmarshalCBOR(nil))
	comprobar("yaml_decode", reconstruido.UnmarshalYAML(func(any) error {
		return nil
	}))

	pruebas, err := artefacto.PruebasParaO3(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	_, err = json.Marshal(pruebas)
	comprobar("pruebas_json", err)
}

func TestPuenteArtefactoAnalisisRespetaContextoCancelado(
	t *testing.T,
) {
	solicitud, capacidad := capacidadArtefactoAnalisisPrueba(t)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := capacidad.PrepararArtefactoAnalisis(
		ctx,
		solicitud,
	); !errors.Is(err, ErrArtefactoAnalisisNoDisponible) ||
		!errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación no conservada: %v", err)
	}
}

func TestPreimagenSemanticaLigaTodaLaEvidenciaDelArtefacto(t *testing.T) {
	solicitud, capacidad := capacidadArtefactoAnalisisPrueba(t)
	artefacto, err := capacidad.PrepararArtefactoAnalisis(
		context.Background(),
		solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	base := DatosPreimagenesOperacionAnalisis{
		ClaveIdempotencia:  "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
		Operacion:          OperacionRegistrarAnalisis,
		ActorRef:           "actor_sintetico_preimagen_001",
		PerfilRef:          "perfil_sintetico_preimagen_001",
		SolicitudArtefacto: solicitud,
		Artefacto:          artefacto,
	}
	preimagen, err := NuevasPreimagenesOperacionAnalisis(base)
	if err != nil {
		t.Fatal(err)
	}
	bytesPrimera, _ := preimagen.BytesSemantica()
	otra := clonarArtefactoAnalisisPrueba(artefacto)
	otra.datos.AutoridadFuenteRC.Serie++
	otra.datos.ArtefactoHuellaSHA256 = ""
	huella, err := huellaArtefactoAnalisisO3(*otra.datos)
	if err != nil {
		t.Fatal(err)
	}
	otra.datos.ArtefactoHuellaSHA256 = huella
	// La prueba privada original impide usar esta proyección nominal coherente.
	if _, err := otra.DatosPara(solicitud); !errors.Is(
		err,
		ErrArtefactoAnalisisNoConfiable,
	) {
		t.Fatalf("proyección nominal autoconsistente aceptada: %v", err)
	}
	otra = clonarArtefactoAnalisisPrueba(artefacto)
	otra.datos.PreparadoEn = otra.datos.PreparadoEn.Add(time.Microsecond)
	otra.pruebas.ordenConjunto.datos.huellaSHA256 =
		otra.datos.ArtefactoHuellaSHA256
	// Una mutación directa no puede producir una segunda preimagen válida.
	base.Artefacto = otra
	if _, err := NuevasPreimagenesOperacionAnalisis(base); !errors.Is(
		err,
		ErrOperacionAnalisisInvalida,
	) {
		t.Fatalf("evidencia adulterada llegó a la preimagen: %v", err)
	}
	base.Artefacto = artefacto
	repetida, err := NuevasPreimagenesOperacionAnalisis(base)
	if err != nil {
		t.Fatal(err)
	}
	bytesSegunda, _ := repetida.BytesSemantica()
	if !bytes.Equal(bytesPrimera, bytesSegunda) {
		t.Fatal("la misma evidencia no produjo preimagen determinista")
	}
}

func capacidadArtefactoAnalisisPrueba(
	t *testing.T,
) (
	SolicitudPrepararArtefactoAnalisis,
	*capacidadPrepararArtefactoAnalisisO3Prueba,
) {
	t.Helper()
	inicio := instanteFuenteAnalisisPrueba()
	solicitudRC := solicitudValidarRCPrueba(t, inicio)
	solicitudCoste := solicitudCalcularCostePrueba(t, inicio)
	validacion := validacionRCPrueba(
		t,
		solicitudRC,
		inicio.Add(time.Second),
	)
	metadatosRC := metadatosRespuestaPrueba(
		validacion.FuenteRef,
		validacion.ReciboRef,
		inicio,
	)
	resultadoRC := resultadoRCFirmadoPrueba(
		t,
		solicitudRC,
		validacion,
		MotivoFuenteAnalisis{},
		metadatosRC,
	)
	metadatosCoste := metadatosRespuestaPrueba(
		"tabla_retributiva_2026_v3",
		"recibo_coste_0123456789",
		inicio,
	)
	resultadoCoste := resultadoCosteFirmadoPrueba(
		t,
		solicitudCoste,
		metadatosCoste,
	)
	funcionales := DatosFuncionalesOperacionAnalisis{
		ModalidadClave:    "sustitucion",
		CategoriaRef:      "categoria_trabajo_social",
		GrupoSubgrupo:     "A2",
		CausaClave:        "incapacidad_temporal",
		Periodo:           preparacionCalcularCostePrueba().Periodo,
		PorcentajeJornada: domain.JornadaCompletaDiezmilesimas,
		EntradaRC:         preparacionValidarRCPrueba().Entrada,
	}
	solicitud := SolicitudPrepararArtefactoAnalisis{
		ArtefactoRef:      "artefacto_analisis_prueba_012345",
		OrganizacionRef:   organizacionAutoridadPrueba,
		ExpedienteRef:     "expediente_temporal_0123456789",
		VersionExpediente: 2,
		DatosFuncionales:  funcionales,
		SolicitadaEn:      inicio.Add(-time.Second),
	}
	solicitudes := &preparadorSolicitudesFuentesAnalisisO3Doble{
		solicitudes: SolicitudesFuentesAnalisisO3{
			ValidacionRC: solicitudRC,
			CalculoCoste: &solicitudCoste,
		},
	}
	fuente := fuentePresupuestariaDoble(func(
		context.Context,
		SolicitudValidarRC,
	) (ResultadoValidacionRC, error) {
		return resultadoRC, nil
	})
	calculador := calculadorCosteDoble(func(
		context.Context,
		SolicitudCalcularCoste,
	) (ResultadoCalculoCoste, error) {
		return resultadoCoste, nil
	})
	capacidad := &capacidadPrepararArtefactoAnalisisO3Prueba{
		solicitudes: solicitudes,
		fuenteRC:    fuente,
		calculador:  calculador,
		verificador: verificadorRespuestaHMACPrueba(
			inicio.Add(2500 * time.Millisecond),
		),
		publicador: verificadorPublicacionNoInvocablePrueba(t),
		confianza:  confianzaAutoridadesPrueba(t),
		reloj:      relojFijoFuenteAnalisis(inicio.Add(4 * time.Second)),
	}
	return solicitud, capacidad
}

func clonarArtefactoAnalisisPrueba(
	origen ArtefactoAnalisisPreparado,
) ArtefactoAnalisisPreparado {
	datos := clonarDatosArtefactoAnalisis(*origen.datos)
	pruebas := *origen.pruebas
	datosOrden, _ := origen.pruebas.ordenConjunto.Datos()
	pruebas.ordenConjunto = ordenConsumoConjuntoDesdeDatosO3(datosOrden)
	return ArtefactoAnalisisPreparado{
		datos:   &datos,
		pruebas: &pruebas,
	}
}

func prepararArtefactoAnalisisPrueba(
	t *testing.T,
	capacidad *capacidadPrepararArtefactoAnalisisO3Prueba,
	solicitud SolicitudPrepararArtefactoAnalisis,
) ArtefactoAnalisisPreparado {
	t.Helper()
	artefacto, err := capacidad.PrepararArtefactoAnalisis(
		context.Background(),
		solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	return artefacto
}

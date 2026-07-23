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

type consumoMemoriaFuenteAnalisisO3 struct {
	instante time.Time
	recibos  map[string]ReciboConsumoRespuestaFuenteAnalisis
	huellas  map[string]string
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

type relojMutableFuenteAnalisisO3Prueba struct{ instante time.Time }

func (r *relojMutableFuenteAnalisisO3Prueba) Ahora() time.Time {
	return r.instante
}

type consumidorQueAvanzaRelojFuenteAnalisisO3Prueba struct {
	delegado ConsumidorRespuestaFuenteAnalisis
	reloj    *relojMutableFuenteAnalisisO3Prueba
	destino  time.Time
}

func (c consumidorQueAvanzaRelojFuenteAnalisisO3Prueba) ConsumirRespuestaFuenteAnalisis(
	ctx context.Context,
	orden OrdenConsumoRespuestaFuenteAnalisis,
) (ReciboConsumoRespuestaFuenteAnalisis, error) {
	recibo, err := c.delegado.ConsumirRespuestaFuenteAnalisis(ctx, orden)
	if err == nil {
		c.reloj.instante = c.destino
	}
	return recibo, err
}

func (c *consumoMemoriaFuenteAnalisisO3) ConsumirRespuestaFuenteAnalisis(
	_ context.Context,
	orden OrdenConsumoRespuestaFuenteAnalisis,
) (ReciboConsumoRespuestaFuenteAnalisis, error) {
	datos, err := orden.Datos()
	if err != nil {
		return ReciboConsumoRespuestaFuenteAnalisis{}, err
	}
	clave := datos.Atestacion.Metadatos.AutoridadRef + ":" +
		datos.Atestacion.Metadatos.ReciboRef
	if anterior, existe := c.huellas[clave]; existe {
		if anterior != datos.HuellaRespuestaSHA256 {
			return ReciboConsumoRespuestaFuenteAnalisis{},
				ErrRespuestaFuenteAnalisisYaConsumida
		}
		return c.recibos[clave], nil
	}
	recibo, err := NuevoReciboConsumoRespuestaFuenteAnalisis(
		orden,
		"consumo_"+string(datos.Tipo)+"_0123456789",
		c.instante,
	)
	if err != nil {
		return ReciboConsumoRespuestaFuenteAnalisis{}, err
	}
	c.huellas[clave] = datos.HuellaRespuestaSHA256
	c.recibos[clave] = recibo
	return recibo, nil
}

func TestPuenteArtefactoAnalisisIntegraEvidenciaO3VerificadaYConsumida(
	t *testing.T,
) {
	solicitud, capacidad, consumidor := capacidadArtefactoAnalisisPrueba(t)
	artefacto, err := capacidad.PrepararArtefactoAnalisis(
		context.Background(),
		solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := artefacto.DatosPara(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	analisis, err := DerivarAnalisisDesdeArtefacto(solicitud, artefacto)
	if err != nil {
		t.Fatal(err)
	}
	pruebas, err := artefacto.PruebasParaO3(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if len(consumidor.recibos) != 2 ||
		datos.ArtefactoHuellaSHA256 == "" ||
		datos.ConsumoRCRef == "" ||
		datos.ConsumoCosteRef == "" ||
		pruebas.ReciboConsumoCoste == nil ||
		analisis.ValidacionRC.FuenteRef != datos.FuenteRCRef ||
		analisis.CostePrevisto == datos.CostePrevisto ||
		analisis.CostePrevisto.Centimos != datos.CostePrevisto.Centimos {
		t.Fatal("el puente perdió evidencia verificada o consumo durable")
	}
	for _, nombre := range []string{
		"ResultadoRC", "ConfirmacionRC", "OrdenConsumoRC",
		"ReciboConsumoRC", "Credencial", "Raiz", "Autoridad",
	} {
		if _, existe := reflect.TypeOf(
			SolicitudPrepararArtefactoAnalisis{},
		).FieldByName(nombre); existe {
			t.Fatalf("el DTO acepta autoridad mediante %s", nombre)
		}
	}
}

func TestPuenteArtefactoAnalisisRepeticionExactaEsIdempotente(
	t *testing.T,
) {
	solicitud, capacidad, consumidor := capacidadArtefactoAnalisisPrueba(t)
	primero, err := capacidad.PrepararArtefactoAnalisis(
		context.Background(),
		solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := capacidad.PrepararArtefactoAnalisis(
		context.Background(),
		solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	datosPrimero, _ := primero.DatosPara(solicitud)
	datosSegundo, _ := segundo.DatosPara(solicitud)
	if datosPrimero.ArtefactoHuellaSHA256 !=
		datosSegundo.ArtefactoHuellaSHA256 ||
		datosPrimero.ConsumoRCRef != datosSegundo.ConsumoRCRef ||
		datosPrimero.ConsumoCosteRef != datosSegundo.ConsumoCosteRef ||
		len(consumidor.recibos) != 2 {
		t.Fatal("la repetición exacta no reutilizó los consumos")
	}
}

func TestPuenteArtefactoAnalisisRechazaCambioDeCredencialEnRevalidacion(
	t *testing.T,
) {
	solicitud, capacidad, consumidor := capacidadArtefactoAnalisisPrueba(t)
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
	if len(consumidor.recibos) != 0 {
		t.Fatal("la credencial cambiante alcanzó el consumo durable")
	}
}

func TestPuenteArtefactoAnalisisRechazaCatalogoAdulteradoAntesDeConsumir(
	t *testing.T,
) {
	solicitud, capacidad, consumidor := capacidadArtefactoAnalisisPrueba(t)
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
	if len(consumidor.recibos) != 0 {
		t.Fatal("el catálogo adulterado alcanzó el consumo durable")
	}
}

func TestPuenteArtefactoAnalisisRechazaConflictoDeRespuestaYaConsumida(
	t *testing.T,
) {
	solicitud, capacidad, _ := capacidadArtefactoAnalisisPrueba(t)
	if _, err := capacidad.PrepararArtefactoAnalisis(
		context.Background(),
		solicitud,
	); err != nil {
		t.Fatal(err)
	}
	solicitudes, err := capacidad.solicitudes.
		PrepararSolicitudesFuentesAnalisisO3(context.Background(), solicitud)
	if err != nil || solicitudes.CalculoCoste == nil {
		t.Fatal("solicitud de coste ausente")
	}
	inicio := instanteFuenteAnalisisPrueba()
	metadatos := metadatosRespuestaPrueba(
		"tabla_retributiva_2026_v3",
		"recibo_coste_0123456789",
		inicio,
	)
	importe := domain.Importe{Centimos: 3_148_026, Moneda: "EUR"}
	preimagen, err := NuevaPreimagenRespuestaCalculoCoste(
		*solicitudes.CalculoCoste,
		metadatos.AutoridadRef,
		metadatos.ReciboRef,
		importe,
		metadatos.EmitidaEn.Add(-time.Second),
		metadatos,
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := NuevoResultadoCalculoCoste(
		*solicitudes.CalculoCoste,
		metadatos.AutoridadRef,
		metadatos.ReciboRef,
		importe,
		metadatos.EmitidaEn.Add(-time.Second),
		atestacionRespuestaPrueba(t, preimagen, metadatos),
	)
	if err != nil {
		t.Fatal(err)
	}
	capacidad.calculador = calculadorCosteDoble(func(
		context.Context,
		SolicitudCalcularCoste,
	) (ResultadoCalculoCoste, error) {
		return resultado, nil
	})
	if _, err := capacidad.PrepararArtefactoAnalisis(
		context.Background(),
		solicitud,
	); !errors.Is(err, ErrRespuestaFuenteAnalisisYaConsumida) {
		t.Fatalf("conflicto de replay aceptado: %v", err)
	}
}

func TestPuenteArtefactoAnalisisRevalidaVigenciaDespuesDelConsumo(
	t *testing.T,
) {
	solicitud, capacidad, consumidor := capacidadArtefactoAnalisisPrueba(t)
	inicio := instanteFuenteAnalisisPrueba()
	reloj := &relojMutableFuenteAnalisisO3Prueba{
		instante: inicio.Add(4 * time.Second),
	}
	capacidad.reloj = reloj
	capacidad.consumidor = consumidorQueAvanzaRelojFuenteAnalisisO3Prueba{
		delegado: consumidor,
		reloj:    reloj,
		destino:  inicio.Add(5 * time.Second),
	}
	if _, err := capacidad.PrepararArtefactoAnalisis(
		context.Background(),
		solicitud,
	); !errors.Is(err, ErrArtefactoAnalisisNoConfiable) {
		t.Fatalf("respuesta caducada tras consumo aceptada: %v", err)
	}
}

func TestArtefactoAnalisisRechazaCualquierAdulteracionDeLaPrueba(
	t *testing.T,
) {
	solicitud, capacidad, _ := capacidadArtefactoAnalisisPrueba(t)
	artefacto, err := capacidad.PrepararArtefactoAnalisis(
		context.Background(),
		solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
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
		{"consumo", func(a *ArtefactoAnalisisPreparado) {
			a.datos.ConsumoRCRef = "consumo_adulterado_012345"
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
	solicitud, capacidad, _ := capacidadArtefactoAnalisisPrueba(t)
	artefacto, err := capacidad.PrepararArtefactoAnalisis(
		context.Background(),
		solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
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

func TestPuenteArtefactoAnalisisRechazaNilTipadoYContextoCancelado(
	t *testing.T,
) {
	solicitud, _, _ := capacidadArtefactoAnalisisPrueba(t)
	var fuente *fuentePresupuestariaDoble
	if _, err := NuevaCapacidadPrepararArtefactoAnalisisO3(
		&preparadorSolicitudesFuentesAnalisisO3Doble{},
		fuente,
		calculadorCosteDoble(nil),
		verificadorRespuestaDoble(nil),
		verificadorPublicacionMotivoDoble(nil),
		&consumoMemoriaFuenteAnalisisO3{},
		ConfianzaAutoridadesFuenteAnalisis{},
		relojFijoFuenteAnalisis(solicitud.SolicitadaEn),
	); !errors.Is(err, ErrSolicitudArtefactoAnalisisInvalida) {
		t.Fatalf("nil tipado aceptado: %v", err)
	}
	_, capacidad, _ := capacidadArtefactoAnalisisPrueba(t)
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
	solicitud, capacidad, _ := capacidadArtefactoAnalisisPrueba(t)
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
	otra.pruebas.reciboRC.ConsumidaEn =
		otra.pruebas.reciboRC.ConsumidaEn.Add(time.Microsecond)
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
	*CapacidadPrepararArtefactoAnalisisO3,
	*consumoMemoriaFuenteAnalisisO3,
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
	consumidor := &consumoMemoriaFuenteAnalisisO3{
		instante: inicio.Add(3 * time.Second),
		recibos:  map[string]ReciboConsumoRespuestaFuenteAnalisis{},
		huellas:  map[string]string{},
	}
	capacidad, err := NuevaCapacidadPrepararArtefactoAnalisisO3(
		solicitudes,
		fuente,
		calculador,
		verificadorRespuestaHMACPrueba(
			inicio.Add(2500*time.Millisecond),
		),
		verificadorPublicacionNoInvocablePrueba(t),
		consumidor,
		confianzaAutoridadesPrueba(t),
		relojFijoFuenteAnalisis(inicio.Add(4*time.Second)),
	)
	if err != nil {
		t.Fatal(err)
	}
	return solicitud, capacidad, consumidor
}

func clonarArtefactoAnalisisPrueba(
	origen ArtefactoAnalisisPreparado,
) ArtefactoAnalisisPreparado {
	datos := clonarDatosArtefactoAnalisis(*origen.datos)
	pruebas := *origen.pruebas
	if origen.pruebas.reciboCoste != nil {
		recibo := *origen.pruebas.reciboCoste
		pruebas.reciboCoste = &recibo
	}
	return ArtefactoAnalisisPreparado{
		datos:   &datos,
		pruebas: &pruebas,
	}
}

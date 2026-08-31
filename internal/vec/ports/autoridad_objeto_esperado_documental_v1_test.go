package ports

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

type escenarioAutoridadObjetoEsperadoDocumentalV1Prueba struct {
	materializacion escenarioMaterializacionDocumentalV4
	perfil          PerfilCapacidadesAlmacenMaterialV2
	plan            SeleccionPlanMaterialAlmacenV2
	recibo          ReciboEscrituraObjetoMaterialV2
	criptografia    *criptografiaMaterialV2Prueba
	registro        *registroAutoritativoMaterialV2Prueba
}

func nuevoEscenarioAutoridadObjetoEsperadoDocumentalV1Prueba(
	t *testing.T,
) escenarioAutoridadObjetoEsperadoDocumentalV1Prueba {
	t.Helper()
	materializacion := escenarioMaterializacionDocumentalV4Prueba(t)
	criptografia := &criptografiaMaterialV2Prueba{
		clave:        []byte("clave-exclusiva-autoridad-objeto-v1-prueba"),
		claveRef:     "clave:atestacion:objeto-esperado:v1",
		claveVersion: 11,
	}
	registro := nuevoRegistroAutoritativoMaterialV2Prueba()
	const perfilRef = "perfil:capacidades:objeto-esperado:v1"
	registro.perfiles[perfilRef] = perfilPublicadoMaterialV2Prueba{
		version: 13, capacidades: materializacion.capacidades,
	}
	perfil, err := NuevoPerfilCapacidadesAlmacenMaterialV2(
		context.Background(), perfilRef, 13, materializacion.capacidades,
		criptografia, criptografia, registro,
	)
	if err != nil {
		t.Fatalf("crear perfil material atestado: %v", err)
	}
	plan, err := NuevaSeleccionPlanMaterialAlmacenV2(
		"plan:material:objeto-esperado:v1", 17,
	)
	if err != nil {
		t.Fatalf("seleccionar plan material: %v", err)
	}
	proyeccion, err := materializacion.solicitud.Contexto.Proyeccion()
	if err != nil {
		t.Fatalf("proyectar contexto V4: %v", err)
	}
	registro.planes[plan.referencia] = planPublicadoMaterialV2Prueba{
		version: plan.version, huella: strings.Repeat("9", 64),
		conectorLogicoID: materializacion.capacidades.ConectorID,
		moduloID:         proyeccion.ModuloID,
		accionNegocio:    proyeccion.AccionNegocio,
		accionTecnica:    proyeccion.AccionTecnica,
		recursoRef:       proyeccion.RecursoRef,
		operacionRef:     proyeccion.OperacionRef,
		cargaRef:         proyeccion.CargaRef,
		efectoRef:        proyeccion.EfectoRef,
		clasificacion:    proyeccion.Clasificacion,
	}
	escenario := escenarioAutoridadObjetoEsperadoDocumentalV1Prueba{
		materializacion: materializacion,
		perfil:          perfil,
		plan:            plan,
		criptografia:    criptografia,
		registro:        registro,
	}
	escenario.recibo = escenario.crearRecibo(t, materializacion.resultado)
	return escenario
}

func (e escenarioAutoridadObjetoEsperadoDocumentalV1Prueba) crearRecibo(
	t *testing.T,
	resultado ResultadoOperacionObjeto,
) ReciboEscrituraObjetoMaterialV2 {
	t.Helper()
	recibo, err := NuevoReciboEscrituraObjetoMaterialV2(
		context.Background(), e.materializacion.solicitud, resultado,
		e.materializacion.capacidades, e.perfil, e.registro, e.plan,
		e.registro, e.registro, e.registro, e.criptografia, e.criptografia,
	)
	if err != nil {
		t.Fatalf("crear recibo material V2: %v", err)
	}
	return recibo
}

func (e escenarioAutoridadObjetoEsperadoDocumentalV1Prueba) crearAutoridad(
	t *testing.T,
) AutoridadObjetoEsperadoDocumentalV1 {
	t.Helper()
	if !reciboMaterialV2CotejaDeclaracionV4(e.recibo, e.materializacion.declaracion) {
		t.Fatal("precondicion: el recibo material no coteja la declaracion V4")
	}
	if err := e.recibo.VerificarAtestacion(context.Background(), e.criptografia); err != nil {
		t.Fatalf("precondicion: atestacion material no verificable: %v", err)
	}
	reciboSinReferencia := e.recibo
	reciboSinReferencia.referenciaDurableOriginal = ""
	identidad, err := reciboSinReferencia.canonicoIdentidadDurable()
	if err != nil {
		t.Fatalf("precondicion: identidad durable invalida: %v", err)
	}
	solicitud, err := nuevaSolicitudReservarReferenciaReciboMaterialV2(identidad)
	if err != nil {
		t.Fatalf("precondicion: solicitud de referencia invalida: %v", err)
	}
	resultado, err := NuevoResultadoReferenciaReciboMaterialV2(
		solicitud, e.recibo.referenciaDurableOriginal,
	)
	if err != nil {
		t.Fatalf("precondicion: resultado de referencia invalido: %v", err)
	}
	if err := e.registro.VerificarReferenciaReciboMaterialV2(
		context.Background(), solicitud, resultado,
	); err != nil {
		t.Fatalf("precondicion: referencia durable no verificable: %v", err)
	}
	if err := verificarReciboAutoridadObjetoEsperadoDocumentalV1(
		context.Background(), e.recibo, e.registro, e.criptografia,
	); err != nil {
		t.Fatalf("precondicion: recibo material no verificable: %v", err)
	}
	autoridad, err := NuevaAutoridadObjetoEsperadoDocumentalV1(
		context.Background(), e.materializacion.declaracion, e.recibo,
		e.registro, e.criptografia,
	)
	if err != nil {
		t.Fatalf("crear autoridad de objeto esperado: %v", err)
	}
	return autoridad
}

func TestAutoridadObjetoEsperadoDocumentalV1SellaReciboMaterialExacto(t *testing.T) {
	escenario := nuevoEscenarioAutoridadObjetoEsperadoDocumentalV1Prueba(t)
	autoridad := escenario.crearAutoridad(t)
	if err := autoridad.Validar(); err != nil {
		t.Fatalf("autoridad valida rechazada: %v", err)
	}
	proyeccion, err := autoridad.PrepararRegistro(
		context.Background(), escenario.registro, escenario.criptografia,
	)
	if err != nil {
		t.Fatalf("preparar registro: %v", err)
	}
	contexto, err := escenario.materializacion.solicitud.Contexto.Proyeccion()
	if err != nil {
		t.Fatal(err)
	}
	huellaRecibo, err := escenario.recibo.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	canonico, err := escenario.recibo.BytesCanonicos()
	if err != nil {
		t.Fatal(err)
	}
	if proyeccion.Esquema != EsquemaAutoridadObjetoEsperadoDocumentalV1 ||
		proyeccion.Version != VersionAutoridadObjetoEsperadoDocumentalV1 ||
		proyeccion.ReciboMaterialRef != escenario.recibo.referenciaDurableOriginal ||
		proyeccion.HuellaReciboMaterialSHA256 != hex.EncodeToString(huellaRecibo[:]) ||
		proyeccion.Objeto != escenario.materializacion.resultado.Objeto.Objeto ||
		proyeccion.ConectorID != escenario.materializacion.resultado.Objeto.ConectorID ||
		proyeccion.EfectoRef != contexto.EfectoRef ||
		proyeccion.HuellaPlanEfectoSHA256 != contexto.HuellaPlanEfectoSHA256 ||
		proyeccion.HuellaManifiestoSHA256 != contexto.HuellaManifiestoSHA256 ||
		proyeccion.PasoRef != contexto.PasoRef ||
		proyeccion.HuellaPasoSHA256 != contexto.HuellaPasoSHA256 ||
		!bytes.Equal(proyeccion.ReciboMaterialCanonico, canonico) {
		t.Fatal("la proyeccion no conserva el cruce exacto V4/recibo material")
	}
	atestacion := proyeccion.AtestacionReciboMaterial
	if atestacion.Algoritmo != AlgoritmoAtestacionMaterialHMACSHA256 ||
		atestacion.ClaveRef != escenario.criptografia.claveRef ||
		atestacion.ClaveVersion != escenario.criptografia.claveVersion ||
		atestacion.Dominio != dominioAtestacionReciboEscrituraMaterialV2 ||
		len(atestacion.Codigo) != sha256.Size {
		t.Fatal("la proyeccion no entrega material verificable del recibo")
	}
}

func TestAutoridadObjetoEsperadoDocumentalV1EntregaCopiasYPermaneceOpaca(t *testing.T) {
	escenario := nuevoEscenarioAutoridadObjetoEsperadoDocumentalV1Prueba(t)
	autoridad := escenario.crearAutoridad(t)
	primera, err := autoridad.PrepararRegistro(
		context.Background(), escenario.registro, escenario.criptografia,
	)
	if err != nil {
		t.Fatal(err)
	}
	canonicoOriginal := append([]byte(nil), primera.ReciboMaterialCanonico...)
	codigoOriginal := append([]byte(nil), primera.AtestacionReciboMaterial.Codigo...)
	primera.ReciboMaterialCanonico[0] ^= 1
	primera.AtestacionReciboMaterial.Codigo[0] ^= 1
	segunda, err := autoridad.PrepararRegistro(
		context.Background(), escenario.registro, escenario.criptografia,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(segunda.ReciboMaterialCanonico, canonicoOriginal) ||
		!bytes.Equal(segunda.AtestacionReciboMaterial.Codigo, codigoOriginal) {
		t.Fatal("el consumidor pudo mutar el material sellado")
	}
	for nombre, valor := range map[string]any{
		"autoridad":  autoridad,
		"proyeccion": segunda,
	} {
		if _, err := json.Marshal(valor); !errors.Is(
			err, ErrSerializacionAutoridadObjetoEsperadoDocumentalV1Prohibida,
		) {
			t.Fatalf("%s admitio JSON generico: %v", nombre, err)
		}
		texto := fmt.Sprintf("%+v", valor)
		if texto != textoAutoridadObjetoEsperadoDocumentalV1Redactado ||
			strings.Contains(texto, escenario.materializacion.resultado.Objeto.Objeto.Referencia) {
			t.Fatalf("%s se filtro por formato: %q", nombre, texto)
		}
	}
	var bitacora bytes.Buffer
	slog.New(slog.NewTextHandler(&bitacora, nil)).Info("prueba", "autoridad", autoridad)
	if strings.Contains(bitacora.String(), escenario.materializacion.resultado.Objeto.Objeto.Referencia) {
		t.Fatal("la autoridad filtro la referencia en logging")
	}
}

func TestAutoridadObjetoEsperadoDocumentalV1RechazaEvidenciaSinReciboYObjetoAjeno(t *testing.T) {
	escenario := nuevoEscenarioAutoridadObjetoEsperadoDocumentalV1Prueba(t)
	if huellaEvidenciaOperacionAlmacenDocumental(
		escenario.materializacion.resultado.Evidencia,
	) == "" {
		t.Fatal("precondicion: la evidencia autocontenida debe tener huella")
	}
	if _, err := NuevaAutoridadObjetoEsperadoDocumentalV1(
		context.Background(), escenario.materializacion.declaracion,
		ReciboEscrituraObjetoMaterialV2{}, escenario.registro, escenario.criptografia,
	); !errors.Is(err, ErrAutoridadObjetoEsperadoDocumentalV1NoValida) {
		t.Fatalf("la evidencia V4 sin recibo autenticado creo autoridad: %v", err)
	}

	resultadoAjeno := escenario.materializacion.resultado
	resultadoAjeno.Objeto.Objeto = ReferenciaObjetoAlmacen{
		Referencia: "objeto:documental:v4:ajeno", Version: "version:documental:v4:ajena",
	}
	resultadoAjeno.Evidencia = evidenciaAlmacenVinculadaPrueba(
		escenario.materializacion.solicitud.Contexto, resultadoAjeno.Objeto.Objeto,
		"evidencia:almacen:v4:ajena", resultadoAjeno.Objeto.ConectorID, "",
		resultadoAjeno.Objeto.AlmacenadoEn,
	)
	resultadoAjeno.Objeto.EvidenciaCreacionRef = resultadoAjeno.Evidencia.Referencia
	reciboAjeno := escenario.crearRecibo(t, resultadoAjeno)
	if _, err := NuevaAutoridadObjetoEsperadoDocumentalV1(
		context.Background(), escenario.materializacion.declaracion, reciboAjeno,
		escenario.registro, escenario.criptografia,
	); !errors.Is(err, ErrAutoridadObjetoEsperadoDocumentalV1NoValida) {
		t.Fatalf("se acepto recibo material de otra pareja objeto/version: %v", err)
	}
}

func TestAutoridadObjetoEsperadoDocumentalV1FallaCerradaEnDependenciasYCancelacion(t *testing.T) {
	escenario := nuevoEscenarioAutoridadObjetoEsperadoDocumentalV1Prueba(t)
	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	var verificadorNulo *criptografiaMaterialV2Prueba
	casos := []struct {
		nombre     string
		ctx        context.Context
		referencia VerificadorReferenciaReciboMaterialV2
		atestacion VerificadorAtestacionMaterialAlmacenV2
	}{
		{"contexto nulo", nil, escenario.registro, escenario.criptografia},
		{"contexto cancelado", ctxCancelado, escenario.registro, escenario.criptografia},
		{"referencia nula tipada", context.Background(), (*registroAutoritativoMaterialV2Prueba)(nil), escenario.criptografia},
		{"atestacion nula tipada", context.Background(), escenario.registro, verificadorNulo},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := NuevaAutoridadObjetoEsperadoDocumentalV1(
				caso.ctx, escenario.materializacion.declaracion, escenario.recibo,
				caso.referencia, caso.atestacion,
			); !errors.Is(err, ErrAutoridadObjetoEsperadoDocumentalV1NoValida) {
				t.Fatalf("dependencia invalida admitida: %v", err)
			}
		})
	}
}

func TestAutoridadObjetoEsperadoDocumentalV1ReverificaAntesDelRegistro(t *testing.T) {
	escenario := nuevoEscenarioAutoridadObjetoEsperadoDocumentalV1Prueba(t)
	autoridad := escenario.crearAutoridad(t)
	escenario.criptografia.rechazar = true
	if _, err := autoridad.PrepararRegistro(
		context.Background(), escenario.registro, escenario.criptografia,
	); !errors.Is(err, ErrAutoridadObjetoEsperadoDocumentalV1NoValida) {
		t.Fatalf("se tolero atestacion revocada: %v", err)
	}
	escenario.criptografia.rechazar = false
	escenario.registro.rechazarReferencia = true
	if _, err := autoridad.PrepararRegistro(
		context.Background(), escenario.registro, escenario.criptografia,
	); !errors.Is(err, ErrAutoridadObjetoEsperadoDocumentalV1NoValida) {
		t.Fatalf("se tolero referencia durable revocada: %v", err)
	}
	escenario.registro.rechazarReferencia = false
	if _, err := autoridad.PrepararRegistro(
		context.Background(), escenario.registro, escenario.criptografia,
	); err != nil {
		t.Fatalf("la autoridad no se recupero con autoridades validas: %v", err)
	}
}

func TestAutoridadObjetoEsperadoDocumentalV1DetectaAlteracionInterna(t *testing.T) {
	escenario := nuevoEscenarioAutoridadObjetoEsperadoDocumentalV1Prueba(t)
	autoridad := escenario.crearAutoridad(t)
	mutaciones := map[string]func(*datosAutoridadObjetoEsperadoDocumentalV1){
		"objeto": func(d *datosAutoridadObjetoEsperadoDocumentalV1) {
			d.objeto.Version = "version:documental:v4:alterada"
		},
		"recibo": func(d *datosAutoridadObjetoEsperadoDocumentalV1) {
			d.recibo.atestacion.codigo[0] ^= 1
		},
		"referencia durable": func(d *datosAutoridadObjetoEsperadoDocumentalV1) {
			d.reciboRef += ":alterada"
		},
		"conector": func(d *datosAutoridadObjetoEsperadoDocumentalV1) {
			d.conectorID += ":alterado"
		},
	}
	for nombre, mutar := range mutaciones {
		t.Run(nombre, func(t *testing.T) {
			copia := autoridad
			datos := *autoridad.datos
			datos.recibo = clonarReciboAutoridadObjetoEsperadoDocumentalV1(datos.recibo)
			copia.datos = &datos
			mutar(copia.datos)
			_, err := copia.PrepararRegistro(
				context.Background(), escenario.registro, escenario.criptografia,
			)
			if !errors.Is(err, ErrAutoridadObjetoEsperadoDocumentalV1NoValida) {
				t.Fatal("la alteracion interna alcanzo la frontera de registro")
			}
		})
	}
}

func TestAutoridadObjetoEsperadoDocumentalV1PreparacionConcurrente(t *testing.T) {
	escenario := nuevoEscenarioAutoridadObjetoEsperadoDocumentalV1Prueba(t)
	autoridad := escenario.crearAutoridad(t)
	const repeticiones = 24
	errores := make(chan error, repeticiones)
	var grupo sync.WaitGroup
	grupo.Add(repeticiones)
	for i := 0; i < repeticiones; i++ {
		go func() {
			defer grupo.Done()
			proyeccion, err := autoridad.PrepararRegistro(
				context.Background(), escenario.registro, escenario.criptografia,
			)
			if err == nil && proyeccion.Objeto != escenario.materializacion.resultado.Objeto.Objeto {
				err = errors.New("pareja de objeto inestable")
			}
			errores <- err
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatalf("preparacion concurrente: %v", err)
		}
	}
}

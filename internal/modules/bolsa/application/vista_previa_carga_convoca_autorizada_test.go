package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type consumidorVistaCargaPrueba struct {
	llamadas   int
	orden      ports.OrdenVistaPreviaCargaConvoca
	material   puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
	err        error
	acuseAjeno bool
}

func (c *consumidorVistaCargaPrueba) ConsumirVistaPreviaCargaConvoca(_ context.Context,
	orden ports.OrdenVistaPreviaCargaConvoca, material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
) (ports.AcuseVistaPreviaCargaConvoca, error) {
	c.llamadas++
	c.orden, c.material = orden, material
	if c.err != nil {
		return ports.AcuseVistaPreviaCargaConvoca{}, c.err
	}
	huella := sha256.Sum256(orden.ContextoRecursoCanonico)
	if c.acuseAjeno {
		huella = sha256.Sum256([]byte("otra pagina"))
	}
	return ports.AcuseVistaPreviaCargaConvoca{DecisionRef: material.ResumenCapacidad().DecisionRef(),
		ActaRef: orden.ActaRef, HuellaContextoSHA256: hex.EncodeToString(huella[:]),
		AuditoriaRef: "aud_v3_0123456789abcdef0123456789abcdef",
		ConsumidaEn:  instanteBorradorLlamamientoPrueba.UTC().Truncate(time.Microsecond)}, nil
}

func escenarioVistaCargaAutorizada(t *testing.T) (*ServicioVistaPreviaCargaConvocaAutorizada,
	ports.SolicitudVistaPreviaCargaConvoca, *autorizadorCargaPrueba, *consumidorVistaCargaPrueba, *lectorContadoCarga) {
	t.Helper()
	e := nuevoEscenarioCargaConvoca(t)
	lector := &lectorContadoCarga{}
	vista, err := NuevoPrevisualizadorCargaConvoca(lector)
	if err != nil {
		t.Fatal(err)
	}
	consumidor := &consumidorVistaCargaPrueba{}
	servicio, err := NuevoServicioVistaPreviaCargaConvocaAutorizada(vista, e.servicio.contexto, e.autorizador, consumidor)
	if err != nil {
		t.Fatal(err)
	}
	q := ports.SolicitudVistaPreviaCargaConvoca{Vinculo: e.solicitud.Vinculo,
		ResultadoContexto: e.solicitud.ResultadoContexto, Correlacion: e.solicitud.Correlacion,
		MotivoAutorizacion: e.solicitud.MotivoAutorizacion, CategoriaRef: e.solicitud.CategoriaRef,
		NombreFichero: e.solicitud.NombreFichero, Contenido: e.solicitud.Contenido,
		Pagina: ports.PaginaVistaPreviaCargaConvoca{Filtro: "con_avisos", Limite: 1, Desplazamiento: 1}}
	return servicio, q, e.autorizador, consumidor, lector
}

func TestVistaPreviaCargaConvocaPreparaPaginaAntesDeConsumirAD218(t *testing.T) {
	servicio, q, autorizador, consumidor, lector := escenarioVistaCargaAutorizada(t)
	preparada, err := servicio.Preparar(context.Background(), q)
	if err != nil || lector.llamadas != 1 || len(autorizador.solicitudes) != 0 || consumidor.llamadas != 0 {
		t.Fatalf("preparación consumió antes de respuesta: %v lector=%d", err, lector.llamadas)
	}
	if preparada.TotalFiltrado != 2 || preparada.Vista.FilasLeidas != 12 ||
		preparada.Vista.Aceptadas != 11 || preparada.Vista.Rechazadas != 1 ||
		len(preparada.Vista.Filas) != 1 || preparada.Vista.Filas[0].Numero != 9 {
		t.Fatalf("página/cifras inesperadas: %+v", preparada)
	}
	acuse, err := servicio.Consumir(context.Background(), preparada)
	if err != nil || acuse.ValidarPara(consumidor.orden, consumidor.material) != nil ||
		consumidor.llamadas != 1 || len(autorizador.solicitudes) != 1 || lector.llamadas != 1 {
		t.Fatalf("consumo/acuse incorrecto: %v %+v", err, acuse)
	}
	suma := sha256.Sum256(q.Contenido)
	expected := "acta:importacion-convoca:"
	if !strings.HasPrefix(consumidor.orden.ActaRef, expected) || consumidor.orden.HuellaFicheroSHA256 != hex.EncodeToString(suma[:]) ||
		consumidor.orden.CategoriaRef != q.CategoriaRef || consumidor.orden.ActorRef != q.ResultadoContexto.Contexto.Principal.ID {
		t.Fatalf("recurso no ligado a fichero/categoría/actor: %+v", consumidor.orden)
	}
	canon := `{"ambitos":{"ambito_ref":"ambito:bolsa","unidad_ref":"unidad:seleccion"},` +
		`"atributos":{"desplazamiento":"1","esquema":"vec.bolsa.rrhh.carga_convoca.vista_previa.v1",` +
		`"fase":"vista_previa","filtro":"con_avisos","limite":"1"}}`
	if string(consumidor.orden.ContextoRecursoCanonico) != canon {
		t.Fatalf("contexto raw distinto de V3/B93: %s", consumidor.orden.ContextoRecursoCanonico)
	}
	datos := autorizador.solicitudes[0]
	if datos.Accion != ports.AccionConfirmarCargaConvoca || datos.Finalidad != ports.FinalidadConfirmarCargaConvoca ||
		datos.Recurso.Referencia != consumidor.orden.ActaRef || datos.Recurso.Atributos["fase"] != "vista_previa" {
		t.Fatalf("decisión no ligada a página real: %+v", datos)
	}
}

func TestVistaPreviaCargaConvocaDenegacionYFalloNoConsumen(t *testing.T) {
	servicio, q, autorizador, consumidor, _ := escenarioVistaCargaAutorizada(t)
	preparada, err := servicio.Preparar(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	autorizador.base.err = puertosvec.ErrDenegacionExplicitaAutorizacionLigadaV3
	if _, err := servicio.Consumir(context.Background(), preparada); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || consumidor.llamadas != 0 {
		t.Fatalf("denegación emitió consulta: %v", err)
	}
	autorizador.base.err = nil
	consumidor.err = errors.New("detalle de base con dato sintético")
	if _, err := servicio.Consumir(context.Background(), preparada); !errors.Is(err, ports.ErrVistaPreviaCargaConvocaNoDisponible) ||
		strings.Contains(err.Error(), "dato sintético") || consumidor.llamadas != 1 {
		t.Fatalf("fallo SQL no minimizado: %v", err)
	}
}

func TestVistaPreviaCargaConvocaRechazaAcuseDeOtraPagina(t *testing.T) {
	servicio, q, _, consumidor, _ := escenarioVistaCargaAutorizada(t)
	preparada, err := servicio.Preparar(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	consumidor.acuseAjeno = true
	if _, err := servicio.Consumir(context.Background(), preparada); !errors.Is(err, ports.ErrVistaPreviaCargaConvocaNoDisponible) || consumidor.llamadas != 1 {
		t.Fatalf("acuse de otra página aceptado: %v", err)
	}
}

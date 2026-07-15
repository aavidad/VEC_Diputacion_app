package application

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type catalogoPerfilesEjecucionV2Prueba struct {
	mu         sync.Mutex
	respuestas [][]ports.DescriptorPerfilDocumental
	error      error
	llamadas   int
}

func (c *catalogoPerfilesEjecucionV2Prueba) BuscarDescriptoresPerfilDocumental(
	_ context.Context,
	_ ports.ConsultaFormatoDocumental,
) ([]ports.DescriptorPerfilDocumental, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	respuesta := respuestaSecuencialEjecucionV2(c.respuestas, c.llamadas)
	c.llamadas++
	return respuesta, c.error
}

func (c *catalogoPerfilesEjecucionV2Prueba) numeroLlamadas() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.llamadas
}

type situacionesEjecucionV2Prueba struct {
	mu         sync.Mutex
	respuestas [][]domain.SituacionOperativaPerfilDocumental
	error      error
	llamadas   int
}

func (s *situacionesEjecucionV2Prueba) BuscarSituacionesOperativasActuales(
	_ context.Context,
	_ ports.ConsultaSituacionOperativaActual,
) ([]domain.SituacionOperativaPerfilDocumental, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	respuesta := respuestaSecuencialEjecucionV2(s.respuestas, s.llamadas)
	s.llamadas++
	return respuesta, s.error
}

func (s *situacionesEjecucionV2Prueba) numeroLlamadas() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.llamadas
}

type componentesEjecucionV2Prueba struct {
	mu         sync.Mutex
	respuestas map[domain.RolComponenteDocumental][][]ports.DescriptorComponenteDocumentalAtestado
	errores    map[domain.RolComponenteDocumental]error
	llamadas   map[domain.RolComponenteDocumental]int
}

func (c *componentesEjecucionV2Prueba) BuscarComponentesDocumentalesAtestados(
	_ context.Context,
	consulta ports.ConsultaComponenteDocumentalAtestado,
) ([]ports.DescriptorComponenteDocumentalAtestado, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	indice := c.llamadas[consulta.Rol]
	c.llamadas[consulta.Rol] = indice + 1
	return respuestaSecuencialEjecucionV2(c.respuestas[consulta.Rol], indice),
		c.errores[consulta.Rol]
}

func (c *componentesEjecucionV2Prueba) numeroLlamadas(
	rol domain.RolComponenteDocumental,
) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.llamadas[rol]
}

type renderizadorEjecucionV2Prueba struct {
	mu                 sync.Mutex
	salida             []byte
	error              error
	excederLimite      bool
	ignorarErrorWriter bool
	llamadas           int
	limites            []uint64
	ultimaEntrada      domain.ContenidoDocumento
}

func (r *renderizadorEjecucionV2Prueba) Renderizar(
	_ context.Context,
	_ ports.DescriptorComponenteDocumentalAtestado,
	_ domain.PerfilFormatoDocumental,
	contenido domain.ContenidoDocumento,
	limite uint64,
	destino io.Writer,
) error {
	r.mu.Lock()
	r.llamadas++
	r.limites = append(r.limites, limite)
	r.ultimaEntrada = clonarContenidoNeutralEjecucionV2(contenido)
	exceder := r.excederLimite
	ignorar := r.ignorarErrorWriter
	salida := append([]byte(nil), r.salida...)
	errConfigurado := r.error
	r.mu.Unlock()

	if errConfigurado != nil {
		return errConfigurado
	}
	if exceder {
		_, err := destino.Write(bytes.Repeat([]byte{'x'}, int(limite)+1))
		if ignorar {
			return nil
		}
		return err
	}
	_, err := destino.Write(salida)
	return err
}

func (r *renderizadorEjecucionV2Prueba) numeroLlamadas() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.llamadas
}

func (r *renderizadorEjecucionV2Prueba) ultimoLimite() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.limites) == 0 {
		return 0
	}
	return r.limites[len(r.limites)-1]
}

type verificadorEjecucionV2Prueba struct {
	mu       sync.Mutex
	error    error
	mutar    bool
	llamadas int
	limites  []uint64
	retenido []byte
}

func (v *verificadorEjecucionV2Prueba) ValidarConformidad(
	_ context.Context,
	_ ports.DescriptorComponenteDocumentalAtestado,
	_ domain.PerfilFormatoDocumental,
	contenido []byte,
	limite uint64,
) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.llamadas++
	v.limites = append(v.limites, limite)
	v.retenido = contenido
	if v.mutar && len(contenido) > 0 {
		contenido[0] ^= 0xff
	}
	return v.error
}

func (v *verificadorEjecucionV2Prueba) mutarRetenido() {
	v.mu.Lock()
	defer v.mu.Unlock()
	if len(v.retenido) > 0 {
		v.retenido[0] ^= 0xff
	}
}

func (v *verificadorEjecucionV2Prueba) numeroLlamadas() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.llamadas
}

func (v *verificadorEjecucionV2Prueba) ultimoLimite() uint64 {
	v.mu.Lock()
	defer v.mu.Unlock()
	if len(v.limites) == 0 {
		return 0
	}
	return v.limites[len(v.limites)-1]
}

type generadorReferenciaEjecucionV2Prueba struct {
	mu         sync.Mutex
	referencia string
	error      error
	llamadas   int
}

func (g *generadorReferenciaEjecucionV2Prueba) NuevaReferenciaBorradorDocumental(
	context.Context,
) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.llamadas++
	return g.referencia, g.error
}

func (g *generadorReferenciaEjecucionV2Prueba) numeroLlamadas() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.llamadas
}

type selladorDatosEjecucionV2Prueba struct {
	mu            sync.Mutex
	error         error
	sello         string
	llamadas      int
	ultimaEntrada []byte
}

func (s *selladorDatosEjecucionV2Prueba) SellarDatos(
	_ context.Context,
	datos []byte,
) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.llamadas++
	s.ultimaEntrada = append([]byte(nil), datos...)
	if s.error != nil {
		return "", s.error
	}
	if s.sello != "" {
		return s.sello, nil
	}
	calculador := hmac.New(sha256.New, []byte("clave-hmac-prueba-secreta"))
	_, _ = calculador.Write(datos)
	return "hmac-sha256:clave-prueba:" + hex.EncodeToString(calculador.Sum(nil)), nil
}

func (s *selladorDatosEjecucionV2Prueba) numeroLlamadas() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.llamadas
}

type relojEjecucionV2Prueba struct{ ahora time.Time }

func (r *relojEjecucionV2Prueba) Ahora() time.Time { return r.ahora }

type escenarioEjecucionFormatoV2Prueba struct {
	perfil      domain.PerfilFormatoDocumental
	revision    domain.RevisionCatalogoFormatosDocumentales
	consulta    ports.ConsultaFormatoDocumental
	descriptor  ports.DescriptorPerfilDocumental
	vigente     domain.SituacionOperativaPerfilDocumental
	revocada    domain.SituacionOperativaPerfilDocumental
	render      ports.DescriptorComponenteDocumentalAtestado
	verificador ports.DescriptorComponenteDocumentalAtestado

	catalogo            *catalogoPerfilesEjecucionV2Prueba
	situaciones         *situacionesEjecucionV2Prueba
	componentes         *componentesEjecucionV2Prueba
	ejecutorRender      *renderizadorEjecucionV2Prueba
	ejecutorVerificador *verificadorEjecucionV2Prueba
	generadorReferencia *generadorReferenciaEjecucionV2Prueba
	sellador            *selladorDatosEjecucionV2Prueba
	reloj               *relojEjecucionV2Prueba
	servicio            *ServicioEjecucionFormatoDocumentalV2
	techoInstitucional  uint64
}

func TestEjecucionFormatoDocumentalV2ExitoEvidenciaRestaurableYCopias(t *testing.T) {
	escenario := nuevoEscenarioEjecucionFormatoV2Prueba(t, 4096, 3072, 2048, 1024)
	contenidoNeutral := domain.ContenidoDocumento{
		Titulo:   "Expediente de 12345678Z",
		Parrafos: []string{"Acuerdo primero", "Acuerdo segundo"},
	}
	borrador, err := escenario.servicio.RenderizarBorrador(
		context.Background(), escenario.consulta, contenidoNeutral,
	)
	if err != nil || borrador.Validar() != nil {
		t.Fatalf("renderizado gobernado rechazado: %#v / %v", borrador, err)
	}
	referencia, errRef := borrador.Referencia()
	contenido1, errContenido1 := borrador.Contenido()
	evidencia, errEvidencia := borrador.Evidencia()
	datos, errDatos := evidencia.Datos()
	huellaEvidencia, errHuella := evidencia.HuellaSHA256()
	restaurada, errRestaurar := RestaurarEvidenciaRenderizadoDocumentalV2(datos)
	huellaRestaurada, errHuellaRestaurada := restaurada.HuellaSHA256()
	if errRef != nil || errContenido1 != nil || errEvidencia != nil || errDatos != nil ||
		errHuella != nil || errRestaurar != nil || errHuellaRestaurada != nil ||
		referencia != "borrador:prueba:0001" || !bytes.Equal(contenido1, escenario.ejecutorRender.salida) ||
		datos.Consulta != escenario.consulta || datos.DescriptorPerfil != escenario.descriptor ||
		datos.SituacionOperativa != escenario.vigente || datos.ComponenteRender != escenario.render ||
		datos.ComponenteVerificador != escenario.verificador ||
		datos.TechoInstitucionalBytes != escenario.techoInstitucional ||
		datos.LimiteEfectivoBytes != 1024 || datos.TamanoSalida != uint64(len(contenido1)) ||
		huellaRestaurada != huellaEvidencia || !strings.HasPrefix(datos.HuellaEntradaHMAC, "hmac-sha256:") ||
		strings.Contains(datos.HuellaEntradaHMAC, "12345678") || escenario.sellador.numeroLlamadas() != 1 {
		t.Fatalf("evidencia o resultado incompleto: %#v / %v / %v", datos, err, errRestaurar)
	}

	contenido1[0] ^= 0xff
	contenido2, err := borrador.Contenido()
	if err != nil || !bytes.Equal(contenido2, escenario.ejecutorRender.salida) {
		t.Fatal("mutar la copia de contenido altero el borrador autoritativo")
	}
	escenario.ejecutorVerificador.mutarRetenido()
	contenido3, err := borrador.Contenido()
	if err != nil || !bytes.Equal(contenido3, escenario.ejecutorRender.salida) || borrador.Validar() != nil {
		t.Fatal("el verificador retuvo y altero el borrador despues de validarlo")
	}

	alterados := datos
	alterados.TamanoSalida++
	if _, err := RestaurarEvidenciaRenderizadoDocumentalV2(alterados); !errors.Is(err, ErrEjecucionFormatoDocumentalCerrada) {
		t.Fatalf("evidencia manipulada restaurada: %v", err)
	}
	if posterior, _ := evidencia.HuellaSHA256(); posterior != huellaEvidencia {
		t.Fatal("mutar Datos altero la evidencia original")
	}
}

func TestEjecucionFormatoDocumentalV2RechazaHistoricoVigenteSiActualRevocada(t *testing.T) {
	escenario := nuevoEscenarioEjecucionFormatoV2Prueba(t, 1024, 1024, 1024, 1024)
	if !escenario.revocada.EsSucesoraDe(escenario.vigente) {
		t.Fatal("escenario operativo no representa una revocacion valida")
	}
	escenario.situaciones.respuestas = [][]domain.SituacionOperativaPerfilDocumental{{escenario.revocada}}
	borrador, err := escenario.servicio.RenderizarBorrador(
		context.Background(), escenario.consulta, domain.ContenidoDocumento{Titulo: "acto"},
	)
	if !errors.Is(err, ErrEjecucionFormatoDocumentalCerrada) || borrador.Validar() == nil ||
		escenario.ejecutorRender.numeroLlamadas() != 0 {
		t.Fatalf("una revision historica vigente autorizo frente a la proyeccion revocada: %#v / %v", borrador, err)
	}
}

func TestEjecucionFormatoDocumentalV2RechazaRevocacionEntrePreYPostEfecto(t *testing.T) {
	escenario := nuevoEscenarioEjecucionFormatoV2Prueba(t, 1024, 1024, 1024, 1024)
	escenario.situaciones.respuestas = [][]domain.SituacionOperativaPerfilDocumental{
		{escenario.vigente}, {escenario.vigente}, {escenario.vigente}, {escenario.revocada},
	}
	borrador, err := escenario.servicio.RenderizarBorrador(
		context.Background(), escenario.consulta, domain.ContenidoDocumento{Titulo: "acto"},
	)
	if !errors.Is(err, ErrEjecucionFormatoDocumentalCerrada) || borrador.Validar() == nil ||
		escenario.ejecutorRender.numeroLlamadas() != 1 ||
		escenario.ejecutorVerificador.numeroLlamadas() != 1 || escenario.situaciones.numeroLlamadas() != 4 {
		t.Fatalf("revocacion TOCTOU no cerro el efecto: %#v / %v", borrador, err)
	}
}

func TestEjecucionFormatoDocumentalV2CierraCardinalidadCeroODos(t *testing.T) {
	tipos := []string{
		"descriptor cero", "descriptor dos", "situacion cero", "situacion dos",
		"renderizador cero", "renderizador dos", "verificador cero", "verificador dos",
	}
	for _, tipo := range tipos {
		t.Run(tipo, func(t *testing.T) {
			escenario := nuevoEscenarioEjecucionFormatoV2Prueba(t, 1024, 1024, 1024, 1024)
			switch tipo {
			case "descriptor cero":
				escenario.catalogo.respuestas = [][]ports.DescriptorPerfilDocumental{{}}
			case "descriptor dos":
				escenario.catalogo.respuestas = [][]ports.DescriptorPerfilDocumental{{escenario.descriptor, escenario.descriptor}}
			case "situacion cero":
				escenario.situaciones.respuestas = [][]domain.SituacionOperativaPerfilDocumental{{}}
			case "situacion dos":
				escenario.situaciones.respuestas = [][]domain.SituacionOperativaPerfilDocumental{{escenario.vigente, escenario.vigente}}
			case "renderizador cero":
				escenario.componentes.respuestas[domain.RolComponenteRenderizador] = [][]ports.DescriptorComponenteDocumentalAtestado{{}}
			case "renderizador dos":
				escenario.componentes.respuestas[domain.RolComponenteRenderizador] = [][]ports.DescriptorComponenteDocumentalAtestado{{escenario.render, escenario.render}}
			case "verificador cero":
				escenario.componentes.respuestas[domain.RolComponenteVerificador] = [][]ports.DescriptorComponenteDocumentalAtestado{{}}
			case "verificador dos":
				escenario.componentes.respuestas[domain.RolComponenteVerificador] = [][]ports.DescriptorComponenteDocumentalAtestado{{escenario.verificador, escenario.verificador}}
			}
			borrador, err := escenario.servicio.RenderizarBorrador(
				context.Background(), escenario.consulta, domain.ContenidoDocumento{Titulo: "acto"},
			)
			if !errors.Is(err, ErrEjecucionFormatoDocumentalCerrada) || borrador.Validar() == nil ||
				escenario.ejecutorRender.numeroLlamadas() != 0 {
				t.Fatalf("cardinalidad no cerrada: %#v / %v", borrador, err)
			}
		})
	}
}

func TestEjecucionFormatoDocumentalV2ExigeComponentesIndependientes(t *testing.T) {
	t.Run("mismo dominio de confianza", func(t *testing.T) {
		escenario := nuevoEscenarioEjecucionFormatoV2Prueba(t, 1024, 1024, 1024, 1024)
		noIndependiente := descriptorComponenteEjecucionV2Prueba(
			t, escenario.descriptor, domain.RolComponenteVerificador,
			"verificador-no-independiente", 1, escenario.render.DominioConfianzaRef(),
			1024, 'd', 'e', 'f',
		)
		escenario.componentes.respuestas[domain.RolComponenteVerificador] =
			[][]ports.DescriptorComponenteDocumentalAtestado{{noIndependiente}}
		borrador, err := escenario.servicio.RenderizarBorrador(
			context.Background(), escenario.consulta, domain.ContenidoDocumento{Titulo: "acto"},
		)
		if !errors.Is(err, ErrEjecucionFormatoDocumentalCerrada) || borrador.Validar() == nil ||
			escenario.ejecutorRender.numeroLlamadas() != 0 {
			t.Fatalf("mismo trust-domain aceptado: %#v / %v", borrador, err)
		}
	})

	t.Run("mismo artefacto renombrado", func(t *testing.T) {
		escenario := nuevoEscenarioEjecucionFormatoV2Prueba(t, 1024, 1024, 1024, 1024)
		consulta := consultaComponenteDesdeDescriptorV2(
			escenario.descriptor, domain.RolComponenteVerificador,
		)
		componente, err := domain.NuevaReferenciaComponenteDocumental(
			domain.RolComponenteVerificador, "verificador-copia", 1,
			"homologacion:verificador-copia:1", strings.Repeat("d", 64),
			escenario.render.Componente().HuellaArtefactoSHA256(),
		)
		if err != nil {
			t.Fatal(err)
		}
		noIndependiente, err := ports.NuevoDescriptorComponenteDocumentalAtestado(
			"descriptor-componente:verificador-copia", consulta, componente,
			"dominio:verificador-copia", "broker:formatos",
			"atestacion:verificador-copia:1", strings.Repeat("f", 64), 1024,
		)
		if err != nil {
			t.Fatal(err)
		}
		escenario.componentes.respuestas[domain.RolComponenteVerificador] =
			[][]ports.DescriptorComponenteDocumentalAtestado{{noIndependiente}}
		borrador, err := escenario.servicio.RenderizarBorrador(
			context.Background(), escenario.consulta, domain.ContenidoDocumento{Titulo: "acto"},
		)
		if !errors.Is(err, ErrEjecucionFormatoDocumentalCerrada) || borrador.Validar() == nil {
			t.Fatalf("artefacto renombrado aceptado: %#v / %v", borrador, err)
		}
	})
}

func TestEjecucionFormatoDocumentalV2AplicaElMenorTecho(t *testing.T) {
	casos := []struct {
		nombre                                     string
		perfil, render, verificador, institucional uint64
		esperado                                   uint64
	}{
		{"institucional", 100, 90, 80, 7, 7},
		{"perfil", 6, 90, 80, 100, 6},
		{"renderizador", 100, 5, 80, 90, 5},
		{"verificador", 100, 90, 4, 80, 4},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			escenario := nuevoEscenarioEjecucionFormatoV2Prueba(
				t, caso.perfil, caso.render, caso.verificador, caso.institucional,
			)
			escenario.ejecutorRender.salida = []byte("ok")
			borrador, err := escenario.servicio.RenderizarBorrador(
				context.Background(), escenario.consulta, domain.ContenidoDocumento{Titulo: "a"},
			)
			evidencia, errEvidencia := borrador.Evidencia()
			datos, errDatos := evidencia.Datos()
			if err != nil || errEvidencia != nil || errDatos != nil ||
				datos.LimiteEfectivoBytes != caso.esperado ||
				escenario.ejecutorRender.ultimoLimite() != caso.esperado ||
				escenario.ejecutorVerificador.ultimoLimite() != caso.esperado {
				t.Fatalf("limite efectivo incorrecto: %#v / %v / %v", datos, err, errDatos)
			}
		})
	}
}

func TestEjecucionFormatoDocumentalV2WriterExcedidoNoDevuelveParcial(t *testing.T) {
	escenario := nuevoEscenarioEjecucionFormatoV2Prueba(t, 8, 8, 8, 8)
	escenario.ejecutorRender.excederLimite = true
	escenario.ejecutorRender.ignorarErrorWriter = true
	borrador, err := escenario.servicio.RenderizarBorrador(
		context.Background(), escenario.consulta, domain.ContenidoDocumento{Titulo: "a"},
	)
	if !errors.Is(err, ErrEjecucionFormatoDocumentalCerrada) || borrador.Validar() == nil ||
		escenario.ejecutorRender.numeroLlamadas() != 1 ||
		escenario.ejecutorVerificador.numeroLlamadas() != 0 {
		t.Fatalf("writer excedido produjo salida parcial: %#v / %v", borrador, err)
	}
}

func TestEjecucionFormatoDocumentalV2EntradaExcesivaCortaAntesDelEjecutor(t *testing.T) {
	escenario := nuevoEscenarioEjecucionFormatoV2Prueba(t, 8, 8, 8, 8)
	borrador, err := escenario.servicio.RenderizarBorrador(
		context.Background(), escenario.consulta,
		domain.ContenidoDocumento{Titulo: "123456789", Parrafos: []string{"no-copiar"}},
	)
	if !errors.Is(err, ErrEjecucionFormatoDocumentalCerrada) || borrador.Validar() == nil ||
		escenario.ejecutorRender.numeroLlamadas() != 0 ||
		escenario.ejecutorVerificador.numeroLlamadas() != 0 ||
		escenario.generadorReferencia.numeroLlamadas() != 0 || escenario.sellador.numeroLlamadas() != 0 {
		t.Fatalf("entrada excesiva alcanzo un efecto: %#v / %v", borrador, err)
	}
}

func TestEjecucionFormatoDocumentalV2AislaMutacionYRetencionDelVerificador(t *testing.T) {
	t.Run("mutacion durante verificacion cierra", func(t *testing.T) {
		escenario := nuevoEscenarioEjecucionFormatoV2Prueba(t, 1024, 1024, 1024, 1024)
		escenario.ejecutorVerificador.mutar = true
		borrador, err := escenario.servicio.RenderizarBorrador(
			context.Background(), escenario.consulta, domain.ContenidoDocumento{Titulo: "acto"},
		)
		if !errors.Is(err, ErrEjecucionFormatoDocumentalCerrada) || borrador.Validar() == nil ||
			escenario.ejecutorVerificador.numeroLlamadas() != 1 {
			t.Fatalf("mutacion del verificador no se detecto: %#v / %v", borrador, err)
		}
	})

	t.Run("retencion posterior no altera autoritativo", func(t *testing.T) {
		escenario := nuevoEscenarioEjecucionFormatoV2Prueba(t, 1024, 1024, 1024, 1024)
		borrador, err := escenario.servicio.RenderizarBorrador(
			context.Background(), escenario.consulta, domain.ContenidoDocumento{Titulo: "acto"},
		)
		if err != nil {
			t.Fatal(err)
		}
		antes, _ := borrador.Contenido()
		escenario.ejecutorVerificador.mutarRetenido()
		despues, err := borrador.Contenido()
		if err != nil || !bytes.Equal(antes, despues) || borrador.Validar() != nil {
			t.Fatal("buffer retenido compartia el contenido autoritativo")
		}
	})
}

func TestEjecucionFormatoDocumentalV2RechazaCambioDeComponentePostEfecto(t *testing.T) {
	t.Run("antes del render", func(t *testing.T) {
		escenario := nuevoEscenarioEjecucionFormatoV2Prueba(t, 1024, 1024, 1024, 1024)
		renderCambiado := descriptorComponenteEjecucionV2Prueba(
			t, escenario.descriptor, domain.RolComponenteRenderizador,
			"renderizador-pdfa-nuevo", 2, "dominio:renderizador:nuevo",
			1024, '0', '1', '2',
		)
		escenario.componentes.respuestas[domain.RolComponenteRenderizador] =
			[][]ports.DescriptorComponenteDocumentalAtestado{{escenario.render}, {renderCambiado}}
		borrador, err := escenario.servicio.RenderizarBorrador(
			context.Background(), escenario.consulta, domain.ContenidoDocumento{Titulo: "acto"},
		)
		if !errors.Is(err, ErrEjecucionFormatoDocumentalCerrada) || borrador.Validar() == nil ||
			escenario.ejecutorRender.numeroLlamadas() != 0 ||
			escenario.componentes.numeroLlamadas(domain.RolComponenteRenderizador) != 2 {
			t.Fatalf("cambio de componente pre-efecto no cerro: %#v / %v", borrador, err)
		}
	})

	t.Run("despues de verificar", func(t *testing.T) {
		escenario := nuevoEscenarioEjecucionFormatoV2Prueba(t, 1024, 1024, 1024, 1024)
		renderCambiado := descriptorComponenteEjecucionV2Prueba(
			t, escenario.descriptor, domain.RolComponenteRenderizador,
			"renderizador-pdfa-nuevo", 2, "dominio:renderizador:nuevo",
			1024, '0', '1', '2',
		)
		escenario.componentes.respuestas[domain.RolComponenteRenderizador] =
			[][]ports.DescriptorComponenteDocumentalAtestado{
				{escenario.render}, {escenario.render}, {renderCambiado},
			}
		borrador, err := escenario.servicio.RenderizarBorrador(
			context.Background(), escenario.consulta, domain.ContenidoDocumento{Titulo: "acto"},
		)
		if !errors.Is(err, ErrEjecucionFormatoDocumentalCerrada) || borrador.Validar() == nil ||
			escenario.ejecutorRender.numeroLlamadas() != 1 ||
			escenario.ejecutorVerificador.numeroLlamadas() != 1 ||
			escenario.componentes.numeroLlamadas(domain.RolComponenteRenderizador) != 3 {
			t.Fatalf("cambio de componente post-efecto no cerro: %#v / %v", borrador, err)
		}
	})

	t.Run("verificador cambia antes de recibir la salida", func(t *testing.T) {
		escenario := nuevoEscenarioEjecucionFormatoV2Prueba(t, 1024, 1024, 1024, 1024)
		verificadorCambiado := descriptorComponenteEjecucionV2Prueba(
			t, escenario.descriptor, domain.RolComponenteVerificador,
			"verificador-pdfa-nuevo", 2, "dominio:verificador:nuevo",
			1024, '0', '1', '2',
		)
		escenario.componentes.respuestas[domain.RolComponenteVerificador] =
			[][]ports.DescriptorComponenteDocumentalAtestado{
				{escenario.verificador}, {escenario.verificador}, {verificadorCambiado},
			}
		borrador, err := escenario.servicio.RenderizarBorrador(
			context.Background(), escenario.consulta, domain.ContenidoDocumento{Titulo: "acto"},
		)
		if !errors.Is(err, ErrEjecucionFormatoDocumentalCerrada) || borrador.Validar() == nil ||
			escenario.ejecutorRender.numeroLlamadas() != 1 ||
			escenario.ejecutorVerificador.numeroLlamadas() != 0 ||
			escenario.componentes.numeroLlamadas(domain.RolComponenteVerificador) != 3 {
			t.Fatalf("la salida se entrego a un verificador cambiado: %#v / %v", borrador, err)
		}
	})
}

func TestEjecucionFormatoDocumentalV2FalloOSelloMalformadoCortanAntesDelEfecto(t *testing.T) {
	casos := []struct {
		nombre string
		sello  string
		err    error
	}{
		{"fallo del sellador", "", errors.New("kms no disponible")},
		{"sha sin clave", strings.Repeat("a", 64), nil},
		{"id de clave hostil", "hmac-sha256:clave*:" + strings.Repeat("a", 64), nil},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			escenario := nuevoEscenarioEjecucionFormatoV2Prueba(t, 1024, 1024, 1024, 1024)
			escenario.sellador.sello = caso.sello
			escenario.sellador.error = caso.err
			borrador, err := escenario.servicio.RenderizarBorrador(
				context.Background(), escenario.consulta, domain.ContenidoDocumento{Titulo: "acto"},
			)
			if !errors.Is(err, ErrEjecucionFormatoDocumentalCerrada) || borrador.Validar() == nil ||
				escenario.sellador.numeroLlamadas() != 1 ||
				escenario.generadorReferencia.numeroLlamadas() != 0 ||
				escenario.ejecutorRender.numeroLlamadas() != 0 {
				t.Fatalf("sello invalido alcanzo un efecto: %#v / %v", borrador, err)
			}
		})
	}
}

func TestEjecucionFormatoDocumentalV2ConstructorRechazaNilTipadoYOpcionCero(t *testing.T) {
	escenario := nuevoEscenarioEjecucionFormatoV2Prueba(t, 1024, 1024, 1024, 1024)
	opciones := OpcionesEjecucionFormatoDocumentalV2{TechoInstitucionalBytes: 1024}
	casos := []struct {
		nombre string
		crear  func() (*ServicioEjecucionFormatoDocumentalV2, error)
	}{
		{"catalogo", func() (*ServicioEjecucionFormatoDocumentalV2, error) {
			var valor *catalogoPerfilesEjecucionV2Prueba
			return NuevoServicioEjecucionFormatoDocumentalV2(valor, escenario.situaciones, escenario.componentes, escenario.ejecutorRender, escenario.ejecutorVerificador, escenario.generadorReferencia, escenario.sellador, escenario.reloj, opciones)
		}},
		{"situaciones", func() (*ServicioEjecucionFormatoDocumentalV2, error) {
			var valor *situacionesEjecucionV2Prueba
			return NuevoServicioEjecucionFormatoDocumentalV2(escenario.catalogo, valor, escenario.componentes, escenario.ejecutorRender, escenario.ejecutorVerificador, escenario.generadorReferencia, escenario.sellador, escenario.reloj, opciones)
		}},
		{"componentes", func() (*ServicioEjecucionFormatoDocumentalV2, error) {
			var valor *componentesEjecucionV2Prueba
			return NuevoServicioEjecucionFormatoDocumentalV2(escenario.catalogo, escenario.situaciones, valor, escenario.ejecutorRender, escenario.ejecutorVerificador, escenario.generadorReferencia, escenario.sellador, escenario.reloj, opciones)
		}},
		{"renderizador", func() (*ServicioEjecucionFormatoDocumentalV2, error) {
			var valor *renderizadorEjecucionV2Prueba
			return NuevoServicioEjecucionFormatoDocumentalV2(escenario.catalogo, escenario.situaciones, escenario.componentes, valor, escenario.ejecutorVerificador, escenario.generadorReferencia, escenario.sellador, escenario.reloj, opciones)
		}},
		{"verificador", func() (*ServicioEjecucionFormatoDocumentalV2, error) {
			var valor *verificadorEjecucionV2Prueba
			return NuevoServicioEjecucionFormatoDocumentalV2(escenario.catalogo, escenario.situaciones, escenario.componentes, escenario.ejecutorRender, valor, escenario.generadorReferencia, escenario.sellador, escenario.reloj, opciones)
		}},
		{"generador", func() (*ServicioEjecucionFormatoDocumentalV2, error) {
			var valor *generadorReferenciaEjecucionV2Prueba
			return NuevoServicioEjecucionFormatoDocumentalV2(escenario.catalogo, escenario.situaciones, escenario.componentes, escenario.ejecutorRender, escenario.ejecutorVerificador, valor, escenario.sellador, escenario.reloj, opciones)
		}},
		{"sellador", func() (*ServicioEjecucionFormatoDocumentalV2, error) {
			var valor *selladorDatosEjecucionV2Prueba
			return NuevoServicioEjecucionFormatoDocumentalV2(escenario.catalogo, escenario.situaciones, escenario.componentes, escenario.ejecutorRender, escenario.ejecutorVerificador, escenario.generadorReferencia, valor, escenario.reloj, opciones)
		}},
		{"reloj", func() (*ServicioEjecucionFormatoDocumentalV2, error) {
			var valor *relojEjecucionV2Prueba
			return NuevoServicioEjecucionFormatoDocumentalV2(escenario.catalogo, escenario.situaciones, escenario.componentes, escenario.ejecutorRender, escenario.ejecutorVerificador, escenario.generadorReferencia, escenario.sellador, valor, opciones)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			servicio, err := caso.crear()
			if servicio != nil || !errors.Is(err, ErrEjecucionFormatoDocumentalCerrada) {
				t.Fatalf("nil tipado obtuvo capacidad: %#v / %v", servicio, err)
			}
		})
	}
	if servicio, err := NuevoServicioEjecucionFormatoDocumentalV2(
		escenario.catalogo, escenario.situaciones, escenario.componentes,
		escenario.ejecutorRender, escenario.ejecutorVerificador,
		escenario.generadorReferencia, escenario.sellador, escenario.reloj,
		OpcionesEjecucionFormatoDocumentalV2{},
	); servicio != nil || !errors.Is(err, ErrEjecucionFormatoDocumentalCerrada) {
		t.Fatalf("opcion cero obtuvo capacidad: %#v / %v", servicio, err)
	}
	if servicio, err := NuevoServicioEjecucionFormatoDocumentalV2(
		escenario.catalogo, escenario.situaciones, escenario.componentes,
		escenario.ejecutorRender, escenario.ejecutorVerificador,
		escenario.generadorReferencia, escenario.sellador, escenario.reloj,
		OpcionesEjecucionFormatoDocumentalV2{TechoInstitucionalBytes: techoAbsolutoEjecucionDocumentalV2 + 1},
	); servicio != nil || !errors.Is(err, ErrEjecucionFormatoDocumentalCerrada) {
		t.Fatalf("techo superior al corte en memoria aceptado: %#v / %v", servicio, err)
	}
}

func TestEjecucionFormatoDocumentalV2RechazaReferenciaHostil(t *testing.T) {
	for _, referencia := range []string{
		"https://externo.example/borrador/1", "borrador:*", "Borrador:1",
		"borrador:../../dni", "borrador:1\ninyectado",
	} {
		t.Run(referencia, func(t *testing.T) {
			escenario := nuevoEscenarioEjecucionFormatoV2Prueba(t, 1024, 1024, 1024, 1024)
			escenario.generadorReferencia.referencia = referencia
			borrador, err := escenario.servicio.RenderizarBorrador(
				context.Background(), escenario.consulta, domain.ContenidoDocumento{Titulo: "acto"},
			)
			if !errors.Is(err, ErrEjecucionFormatoDocumentalCerrada) || borrador.Validar() == nil ||
				escenario.ejecutorRender.numeroLlamadas() != 0 {
				t.Fatalf("referencia hostil aceptada: %q / %#v / %v", referencia, borrador, err)
			}
		})
	}
}

func nuevoEscenarioEjecucionFormatoV2Prueba(
	t *testing.T,
	limitePerfil, limiteRender, limiteVerificador, techoInstitucional uint64,
) *escenarioEjecucionFormatoV2Prueba {
	t.Helper()
	identidad, err := domain.NuevaIdentidadSintacticaDocumental("pdf")
	if err != nil {
		t.Fatal(err)
	}
	perfilRef, err := domain.NuevaReferenciaPerfilDocumental("pdfa-4", 2)
	if err != nil {
		t.Fatal(err)
	}
	capacidades, err := domain.NuevasCapacidadesPerfilFormatoDocumental(
		domain.CapacidadPerfilRenderizar, domain.CapacidadPerfilMetadatoInstitucional,
	)
	if err != nil {
		t.Fatal(err)
	}
	conformidad, err := domain.NuevaReferenciaConformidadDocumental(
		"conformidad:pdfa4", 1, "esquema:pdfa4:1", "dialecto:pdfa4",
		"canonicalizacion:pdf:1", "reglas:pdfa4:1", strings.Repeat("a", 64),
		"politica:documental:1", strings.Repeat("b", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	perfil, err := domain.NuevoPerfilFormatoDocumentalConforme(
		perfilRef, identidad, "application/pdf", "pdf", "binario",
		capacidades, conformidad, limitePerfil,
	)
	if err != nil {
		t.Fatal(err)
	}
	revision, err := domain.NuevaRevisionCatalogoFormatosDocumentales(21, strings.Repeat("c", 64))
	if err != nil {
		t.Fatal(err)
	}
	descriptor, err := ports.NuevoDescriptorPerfilDocumental(
		"descriptor:pdfa4:2", "publicacion:pdfa4:2", perfil, revision,
	)
	if err != nil {
		t.Fatal(err)
	}
	consulta := ports.ConsultaFormatoDocumental{
		Identidad: identidad, PerfilRef: perfilRef,
		DigestPerfilSHA256: perfil.DigestSHA256(), RevisionCatalogo: revision,
	}
	vigente, err := domain.NuevaSituacionOperativaPerfilDocumental(
		descriptor.PublicacionRef(), perfil, revision, 1, domain.EstadoPublicacionPerfilVigente,
	)
	if err != nil {
		t.Fatal(err)
	}
	revocada, err := domain.NuevaSituacionOperativaPerfilDocumental(
		descriptor.PublicacionRef(), perfil, revision, 2, domain.EstadoPublicacionPerfilRevocada,
	)
	if err != nil {
		t.Fatal(err)
	}
	render := descriptorComponenteEjecucionV2Prueba(
		t, descriptor, domain.RolComponenteRenderizador, "renderizador-pdfa", 1,
		"dominio:renderizador", limiteRender, 'a', 'b', 'c',
	)
	verificador := descriptorComponenteEjecucionV2Prueba(
		t, descriptor, domain.RolComponenteVerificador, "verificador-pdfa", 1,
		"dominio:verificador", limiteVerificador, 'd', 'e', 'f',
	)
	catalogo := &catalogoPerfilesEjecucionV2Prueba{
		respuestas: [][]ports.DescriptorPerfilDocumental{{descriptor}},
	}
	situaciones := &situacionesEjecucionV2Prueba{
		respuestas: [][]domain.SituacionOperativaPerfilDocumental{{vigente}},
	}
	componentes := &componentesEjecucionV2Prueba{
		respuestas: map[domain.RolComponenteDocumental][][]ports.DescriptorComponenteDocumentalAtestado{
			domain.RolComponenteRenderizador: {{render}},
			domain.RolComponenteVerificador:  {{verificador}},
		},
		errores:  make(map[domain.RolComponenteDocumental]error),
		llamadas: make(map[domain.RolComponenteDocumental]int),
	}
	ejecutorRender := &renderizadorEjecucionV2Prueba{salida: []byte("%PDF-2.0\nborrador gobernado")}
	ejecutorVerificador := &verificadorEjecucionV2Prueba{}
	generador := &generadorReferenciaEjecucionV2Prueba{referencia: "borrador:prueba:0001"}
	sellador := &selladorDatosEjecucionV2Prueba{}
	reloj := &relojEjecucionV2Prueba{
		ahora: time.Date(2026, time.July, 15, 14, 30, 0, 0, time.UTC),
	}
	servicio, err := NuevoServicioEjecucionFormatoDocumentalV2(
		catalogo, situaciones, componentes, ejecutorRender, ejecutorVerificador,
		generador, sellador, reloj,
		OpcionesEjecucionFormatoDocumentalV2{TechoInstitucionalBytes: techoInstitucional},
	)
	if err != nil {
		t.Fatal(err)
	}
	return &escenarioEjecucionFormatoV2Prueba{
		perfil: perfil, revision: revision, consulta: consulta, descriptor: descriptor,
		vigente: vigente, revocada: revocada, render: render, verificador: verificador,
		catalogo: catalogo, situaciones: situaciones, componentes: componentes,
		ejecutorRender: ejecutorRender, ejecutorVerificador: ejecutorVerificador,
		generadorReferencia: generador, sellador: sellador, reloj: reloj,
		servicio: servicio, techoInstitucional: techoInstitucional,
	}
}

func descriptorComponenteEjecucionV2Prueba(
	t *testing.T,
	descriptor ports.DescriptorPerfilDocumental,
	rol domain.RolComponenteDocumental,
	identificador string,
	version uint64,
	dominioConfianza string,
	maximoBytes uint64,
	huellaHomologacion, huellaArtefacto, huellaAtestacion byte,
) ports.DescriptorComponenteDocumentalAtestado {
	t.Helper()
	consulta := consultaComponenteDesdeDescriptorV2(descriptor, rol)
	componente, err := domain.NuevaReferenciaComponenteDocumental(
		rol, identificador, version,
		"homologacion:"+identificador+":"+strconv.FormatUint(version, 10),
		strings.Repeat(string(huellaHomologacion), 64),
		strings.Repeat(string(huellaArtefacto), 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := ports.NuevoDescriptorComponenteDocumentalAtestado(
		"descriptor-componente:"+identificador, consulta, componente, dominioConfianza,
		"broker:formatos", "atestacion:"+identificador+":"+strconv.FormatUint(version, 10),
		strings.Repeat(string(huellaAtestacion), 64), maximoBytes,
	)
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}

func respuestaSecuencialEjecucionV2[T any](respuestas [][]T, indice int) []T {
	if len(respuestas) == 0 {
		return nil
	}
	if indice >= len(respuestas) {
		indice = len(respuestas) - 1
	}
	return append([]T(nil), respuestas[indice]...)
}

func clonarContenidoNeutralEjecucionV2(
	contenido domain.ContenidoDocumento,
) domain.ContenidoDocumento {
	return domain.ContenidoDocumento{
		Titulo: contenido.Titulo, Parrafos: append([]string(nil), contenido.Parrafos...),
	}
}

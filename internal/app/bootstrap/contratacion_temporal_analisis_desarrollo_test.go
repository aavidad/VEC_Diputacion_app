package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type consultaCatalogoRectificacionAnalisisPrueba struct {
	resultado vecports.ResultadoConsultaCatalogosAcotada
	err       error
}

func (c consultaCatalogoRectificacionAnalisisPrueba) ObtenerCatalogoAcotado(
	context.Context,
	string,
	int,
	vecports.LimitesConsultaCatalogosAcotada,
) (vecports.ResultadoConsultaCatalogoAcotado, error) {
	return vecports.ResultadoConsultaCatalogoAcotado{}, vecports.ErrCatalogoNoEncontrado
}

func (c consultaCatalogoRectificacionAnalisisPrueba) ListarVersionesCatalogoAcotado(
	context.Context,
	string,
	vecports.LimitesConsultaCatalogosAcotada,
) (vecports.ResultadoConsultaCatalogosAcotada, error) {
	return c.resultado, c.err
}

func catalogoMotivosRectificacionAnalisisPrueba(t *testing.T) vecdomain.CatalogoConfigurable {
	t.Helper()
	creadoEn := time.Date(2026, 9, 13, 9, 0, 0, 0, time.UTC)
	borrador := vecdomain.CatalogoConfigurable{
		ID: "motivos_rectificacion_analisis", Version: 1, Revision: 1,
		ModuloID: "contratacion_temporal", Nombre: "Motivos de rectificación",
		FuenteRef: "fuente_rrhh_sintetica_001", MotivoCreacion: "Alta de prueba.",
		Entradas: []vecdomain.EntradaCatalogoConfigurable{{
			Clave: "rectificacion_coste", Etiqueta: "Ajuste de coste", Orden: 1,
			VigenteDesde: creadoEn,
			Atributos: map[string]string{
				"clave_i18n": "contratacion_temporal.analisis.rectificacion.ajuste_coste",
			},
		}},
		Estado:    vecdomain.EstadoCatalogoBorrador,
		CreadoPor: "gestor_catalogo_001", CreadoEn: creadoEn,
	}
	publicado, err := borrador.Publicar(
		"revisor_catalogo_001", "aprobacion_catalogo_001", "Publicación de prueba.", creadoEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	return publicado
}

func TestRutaConfiguracionAnalisisDesarrolloPublicaContratoCerrado(t *testing.T) {
	ruta, err := nuevaRutaConfiguracionAnalisisContratacionTemporalDesarrollo()
	if err != nil {
		t.Fatal(err)
	}
	if ruta.Ruta != rutaConfiguracionAnalisisContratacionTemporalDesarrollo ||
		ruta.Manejador == nil {
		t.Fatalf("ruta inesperada: %+v", ruta)
	}
	peticion := httptest.NewRequest(http.MethodGet, ruta.Ruta, nil)
	respuesta := httptest.NewRecorder()
	ruta.Manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusOK {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
	if respuesta.Header().Get("Set-Cookie") != "" ||
		respuesta.Header().Get("Cache-Control") != "no-store, no-transform" {
		t.Fatalf("cabeceras inseguras: %+v", respuesta.Header())
	}
	var contenido struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(respuesta.Body.Bytes(), &contenido); err != nil {
		t.Fatal(err)
	}
	campos := []string{
		"esquema",
		"artefacto_ref",
		"modalidades",
		"categorias",
		"causas",
		"entradas_rc",
		"motivos_rectificacion",
	}
	if len(contenido.Data) != len(campos) {
		t.Fatalf("contrato abierto o incompleto: %v", contenido.Data)
	}
	for _, campo := range campos {
		if _, existe := contenido.Data[campo]; !existe {
			t.Fatalf("falta %s", campo)
		}
	}
	var configuracion configuracionAnalisisContratacionTemporalDesarrollo
	if err := json.Unmarshal(respuesta.Body.Bytes(), &struct {
		Data *configuracionAnalisisContratacionTemporalDesarrollo `json:"data"`
	}{Data: &configuracion}); err != nil {
		t.Fatal(err)
	}
	if configuracion.Esquema != esquemaConfiguracionAnalisisContratacionTemporal ||
		configuracion.ArtefactoRef != artefactoAnalisisContratacionTemporalDesarrollo ||
		len(configuracion.Modalidades) != 5 ||
		len(configuracion.Categorias) != 1 ||
		len(configuracion.Categorias[0].GruposSubgrupos) != 1 ||
		len(configuracion.Causas) != 1 ||
		len(configuracion.EntradasRC) != 1 ||
		len(configuracion.MotivosRectificacion) != 0 {
		t.Fatalf("configuracion inesperada: %+v", configuracion)
	}
	esperadas := []string{
		"sustitucion",
		"vacante",
		"acumulacion_tareas",
		"programa",
		"relevo",
	}
	for indice, esperada := range esperadas {
		if configuracion.Modalidades[indice].Clave != esperada {
			t.Fatalf("modalidad[%d]=%q", indice, configuracion.Modalidades[indice].Clave)
		}
	}
	entrada := configuracion.EntradasRC[0]
	if entrada.Referencia != entradaRCAnalisisContratacionTemporalDesarrollo ||
		entrada.HuellaSHA256 != huellaEntradaRCAnalisisContratacionTemporalDesarrollo ||
		entrada.HuellaSHA256 == strings.Repeat("0", 64) {
		t.Fatalf("entrada RC inesperada: %+v", entrada)
	}

	peticionHEAD := httptest.NewRequest(http.MethodHead, ruta.Ruta, nil)
	respuestaHEAD := httptest.NewRecorder()
	ruta.Manejador.ServeHTTP(respuestaHEAD, peticionHEAD)
	if respuestaHEAD.Code != http.StatusOK || respuestaHEAD.Body.Len() != 0 ||
		respuestaHEAD.Header().Get("Content-Length") !=
			respuesta.Header().Get("Content-Length") {
		t.Fatalf("HEAD incoherente: estado=%d cabeceras=%v cuerpo=%q",
			respuestaHEAD.Code,
			respuestaHEAD.Header(),
			respuestaHEAD.Body.String(),
		)
	}
}

func TestRutaConfiguracionAnalisisDesarrolloFallaCerrada(t *testing.T) {
	ruta, err := nuevaRutaConfiguracionAnalisisContratacionTemporalDesarrollo()
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre string
		crear  func() *http.Request
		estado int
	}{
		{
			nombre: "metodo",
			crear: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, ruta.Ruta, nil)
			},
			estado: http.StatusMethodNotAllowed,
		},
		{
			nombre: "consulta",
			crear: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, ruta.Ruta+"?perfil=rrhh", nil)
			},
			estado: http.StatusBadRequest,
		},
		{
			nombre: "cuerpo",
			crear: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, ruta.Ruta, strings.NewReader("{}"))
			},
			estado: http.StatusBadRequest,
		},
		{
			nombre: "identidad libre",
			crear: func() *http.Request {
				peticion := httptest.NewRequest(http.MethodGet, ruta.Ruta, nil)
				peticion.Header.Set("X-User", "rrhh")
				return peticion
			},
			estado: http.StatusBadRequest,
		},
		{
			nombre: "cookie",
			crear: func() *http.Request {
				peticion := httptest.NewRequest(http.MethodGet, ruta.Ruta, nil)
				peticion.Header.Set("Cookie", "sesion=no")
				return peticion
			},
			estado: http.StatusBadRequest,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			respuesta := httptest.NewRecorder()
			ruta.Manejador.ServeHTTP(respuesta, caso.crear())
			if respuesta.Code != caso.estado {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
			}
			if respuesta.Header().Get("Set-Cookie") != "" {
				t.Fatal("la respuesta emitio una cookie")
			}
			var contenido map[string]json.RawMessage
			if err := json.Unmarshal(respuesta.Body.Bytes(), &contenido); err != nil {
				t.Fatal(err)
			}
			if _, existe := contenido["data"]; existe {
				t.Fatal("una solicitud denegada recibio datos")
			}
		})
	}
}

func TestRutaConfiguracionAnalisisDesarrolloCargaSoloMotivosPublicadosVigentes(t *testing.T) {
	catalogo := catalogoMotivosRectificacionAnalisisPrueba(t)
	fuente := nuevaFuenteMotivosRectificacionAnalisisDesarrollo(
		consultaCatalogoRectificacionAnalisisPrueba{
			resultado: vecports.ResultadoConsultaCatalogosAcotada{
				Catalogos: []vecdomain.CatalogoConfigurable{catalogo},
			},
		},
		catalogo.ID,
		catalogo.ModuloID,
		relojFijoAltaContratacionTemporalDesarrollo{ahora: catalogo.PublicadoEn},
	)
	ruta, err := nuevaRutaConfiguracionAnalisisContratacionTemporalDesarrolloConMotivos(fuente)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	ruta.Manejador.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, ruta.Ruta, nil))
	if respuesta.Code != http.StatusOK {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
	var contenido struct {
		Data configuracionAnalisisContratacionTemporalDesarrollo `json:"data"`
	}
	if err := json.Unmarshal(respuesta.Body.Bytes(), &contenido); err != nil {
		t.Fatal(err)
	}
	if len(contenido.Data.MotivosRectificacion) != 1 ||
		contenido.Data.MotivosRectificacion[0].Clave != "rectificacion_coste" ||
		contenido.Data.MotivosRectificacion[0].Etiqueta != "Ajuste de coste" {
		t.Fatalf("motivos inesperados: %+v", contenido.Data.MotivosRectificacion)
	}
}

func TestFuenteMotivosRectificacionAnalisisFallaCerrada(t *testing.T) {
	catalogo := catalogoMotivosRectificacionAnalisisPrueba(t)
	catalogo.Entradas[0].Atributos["clave_i18n"] = "motivo_sin_espacio_i18n"
	casos := []struct {
		nombre    string
		resultado vecports.ResultadoConsultaCatalogosAcotada
	}{
		{"truncada", vecports.ResultadoConsultaCatalogosAcotada{Truncado: true}},
		{"atributo_i18n_ajeno", vecports.ResultadoConsultaCatalogosAcotada{Catalogos: []vecdomain.CatalogoConfigurable{catalogo}}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			fuente := nuevaFuenteMotivosRectificacionAnalisisDesarrollo(
				consultaCatalogoRectificacionAnalisisPrueba{resultado: caso.resultado},
				"motivos_rectificacion_analisis", "contratacion_temporal",
				relojFijoAltaContratacionTemporalDesarrollo{ahora: catalogo.PublicadoEn},
			)
			if opciones := fuente.opciones(t.Context()); len(opciones) != 0 {
				t.Fatalf("la fuente no confiable expuso motivos: %+v", opciones)
			}
		})
	}
}

func TestPoliticaAnalisisDesarrolloSoloAdmiteRegistroConfigurado(t *testing.T) {
	instante := time.Now().UTC().Truncate(time.Microsecond)
	solicitud := ports.SolicitudResolverPoliticaOperacionAnalisis{
		Operacion:         ports.OperacionRegistrarAnalisis,
		OrganizacionRef:   organizacionAltaContratacionTemporalDesarrollo,
		ExpedienteRef:     "expediente:ct:desarrollo:analisis:001",
		VersionExpediente: 1,
		Flujo: domain.ReferenciaFlujo{
			DefinicionRef: "flujo:ct:desarrollo",
			Version:       1,
			HuellaSHA256:  huellaAltaContratacionTemporalDesarrollo("flujo"),
		},
		FasePrevia:            "solicitud",
		EstadoPrevio:          domain.EstadoEnCurso,
		ActorRef:              "principal:desarrollo:rrhh",
		PerfilRef:             "perfil:desarrollo:rrhh",
		ArtefactoRef:          artefactoAnalisisContratacionTemporalDesarrollo,
		ArtefactoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("artefacto"),
		Instante:              instante,
	}
	politica, err := (resolutorPoliticaOperacionAnalisisDesarrollo{}).
		ResolverPoliticaOperacionAnalisis(t.Context(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if politica.ValidarPara(solicitud) != nil ||
		politica.Accion != domain.ClaveCatalogo(ports.AccionRegistrarAnalisis) ||
		politica.Finalidad != finalidadAnalisisContratacionTemporalDesarrollo ||
		politica.MotivoAutorizacion != referenciaMotivoAutorizacionAnalisisDesarrollo("registro") {
		t.Fatalf("politica inesperada: %+v", politica)
	}
	for _, caso := range []struct {
		nombre  string
		alterar func(*ports.SolicitudResolverPoliticaOperacionAnalisis)
	}{
		{
			nombre: "fase ajena",
			alterar: func(s *ports.SolicitudResolverPoliticaOperacionAnalisis) {
				s.FasePrevia = "cobertura"
			},
		},
		{
			nombre: "estado ajeno",
			alterar: func(s *ports.SolicitudResolverPoliticaOperacionAnalisis) {
				s.EstadoPrevio = domain.EstadoPendiente
			},
		},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			alterada := solicitud
			caso.alterar(&alterada)
			if _, err := (resolutorPoliticaOperacionAnalisisDesarrollo{}).
				ResolverPoliticaOperacionAnalisis(t.Context(), alterada); err == nil {
				t.Fatal("la politica acepto coordenadas fuera del paso real")
			}
		})
	}
	solicitud.Operacion = ports.OperacionRectificarAnalisis
	solicitud.ActorAnalisisAnteriorRef = "principal:desarrollo:anterior"
	solicitud.MotivoRectificacionClave = "correccion"
	if _, err := (resolutorPoliticaOperacionAnalisisDesarrollo{}).
		ResolverPoliticaOperacionAnalisis(t.Context(), solicitud); err == nil {
		t.Fatal("la rectificacion sin motivo configurado quedo abierta")
	}
}

func TestConfiguracionAnalisisAnunciaSubsanacionSoloCompuesta(t *testing.T) {
	for _, disponible := range []bool{false, true} {
		ruta, err := nuevaRutaConfiguracionAnalisisConSubsanacionDesarrollo(disponible)
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		ruta.Manejador.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta.Ruta, nil))
		var respuesta struct {
			Data map[string]json.RawMessage `json:"data"`
		}
		if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &respuesta) != nil {
			t.Fatal("configuración no disponible")
		}
		valor, existe := respuesta.Data["subsanacion_disponible"]
		if disponible && (!existe || string(valor) != "true") {
			t.Fatal("composición no anunciada")
		}
		if !disponible && existe {
			t.Fatal("contrato anterior alterado sin subsanación")
		}
	}
}

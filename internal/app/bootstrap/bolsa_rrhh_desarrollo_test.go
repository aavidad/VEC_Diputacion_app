package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	constitucionapp "vec-diputacion-granada/internal/modules/bolsa/application/constitucion"
	importacionapp "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	bolsadominio "vec-diputacion-granada/internal/modules/bolsa/domain"
	importaciondominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type repositorioBolsasRRHHPrueba struct {
	ports.RepositorioConstitucion
	vigentes []ports.ConstitucionVigente
	entradas []ports.EntradaConstitucion
}

func (r repositorioBolsasRRHHPrueba) ListarVigentes(context.Context) ([]ports.ConstitucionVigente, error) {
	return r.vigentes, nil
}

func (r repositorioBolsasRRHHPrueba) Entradas(context.Context, string, uint64) ([]ports.EntradaConstitucion, error) {
	return r.entradas, nil
}

type recuperadorBolsasRRHHPrueba struct {
	lote   importaciondominio.LoteValidado
	existe bool
}

type situacionesBolsasRRHHPrueba struct{ situacion ports.SituacionParticipacion }

func (situacionesBolsasRRHHPrueba) ParticipacionPerteneceABolsa(context.Context, string, string) (bool, error) {
	return true, nil
}
func (r situacionesBolsasRRHHPrueba) SituacionVigente(context.Context, string) (ports.SituacionParticipacion, error) {
	return r.situacion, nil
}
func (situacionesBolsasRRHHPrueba) BuscarRegistroSituacion(context.Context, string, string) (ports.RegistroSituacionParticipacion, error) {
	return ports.RegistroSituacionParticipacion{}, ports.ErrSituacionParticipacionNoEncontrada
}
func (situacionesBolsasRRHHPrueba) RegistrarSituacion(context.Context, ports.ComandoCambiarSituacionParticipacion) (ports.RegistroSituacionParticipacion, error) {
	return ports.RegistroSituacionParticipacion{}, ports.ErrSituacionParticipacionNoDisponible
}

func (r recuperadorBolsasRRHHPrueba) RecuperarLote(context.Context, string, string) (importaciondominio.LoteValidado, importacionapp.EstadoImportacion, bool, error) {
	return r.lote, importacionapp.EstadoImportacion{}, r.existe, nil
}

var _ constitucionapp.Recuperador = recuperadorBolsasRRHHPrueba{}

func datosBolsasRRHHPrueba() datasetBolsasRRHHDesarrollo {
	datos := datasetBolsasRRHHDesarrollo{GeneradoEn: "2026-09-20T10:00:00Z"}
	datos.Bolsas = append(datos.Bolsas, struct {
		Referencia   string  `json:"bolsa_ref"`
		CategoriaRef string  `json:"categoria_ref"`
		Categoria    string  `json:"categoria"`
		TipoLista    string  `json:"tipo_lista"`
		VigenteDesde string  `json:"vigente_desde"`
		VigenteHasta *string `json:"vigente_hasta"`
	}{Referencia: "bolsa:constituida:administrativo", CategoriaRef: "categoria:rpt:administrativo", Categoria: "Administrativo", TipoLista: "definitiva", VigenteDesde: datos.GeneradoEn})
	for _, candidatura := range []struct {
		referencia, nombre, documento string
		orden                         int
	}{
		{"participacion:001", "Candidatura Uno", "***0001**", 1},
		{"participacion:002", "Candidatura Dos", "***0002**", 2},
	} {
		datos.Candidaturas = append(datos.Candidaturas, struct {
			Referencia  string  `json:"candidatura_ref"`
			BolsaRef    string  `json:"bolsa_ref"`
			Orden       int     `json:"orden"`
			Nombre      string  `json:"nombre_visible"`
			Documento   string  `json:"documento_enmascarado"`
			Estado      string  `json:"estado_clave"`
			EstadoDesde string  `json:"estado_desde"`
			Disponible  *string `json:"disponible_desde"`
		}{Referencia: candidatura.referencia, BolsaRef: "bolsa:constituida:administrativo", Orden: candidatura.orden, Nombre: candidatura.nombre, Documento: candidatura.documento, Estado: "disponible", EstadoDesde: datos.GeneradoEn})
	}
	return datos
}

func manejadorBolsasRRHHPrueba() *bolsasRRHHDesarrollo {
	datos := datosBolsasRRHHPrueba()
	return nuevoManejadorBolsasRRHHDesarrollo(func(context.Context) (datasetBolsasRRHHDesarrollo, error) { return datos, nil })
}

func TestBolsasRRHHDesarrolloExponeContratoCerradoYPaginaCandidatos(t *testing.T) {
	manejador := manejadorBolsasRRHHPrueba()
	lista := httptest.NewRecorder()
	manejador.ServeHTTP(lista, httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo, nil))
	if lista.Code != http.StatusOK || strings.Contains(strings.ToLower(lista.Body.String()), "correo") || strings.Contains(strings.ToLower(lista.Body.String()), "telefono") {
		t.Fatalf("lista C20: status=%d body=%s", lista.Code, lista.Body.String())
	}
	var salida struct {
		Data struct {
			Esquema string `json:"esquema"`
			Bolsas  []struct {
				Referencia string         `json:"bolsa_ref"`
				Estados    map[string]int `json:"por_estado"`
			} `json:"bolsas"`
		} `json:"data"`
	}
	if err := json.Unmarshal(lista.Body.Bytes(), &salida); err != nil || salida.Data.Esquema != "vec.bolsa.rrhh.bolsas.v1" || len(salida.Data.Bolsas) != 1 || len(salida.Data.Bolsas[0].Estados) != 7 {
		t.Fatalf("contrato bolsas: %#v err=%v", salida, err)
	}
	candidatos := httptest.NewRecorder()
	manejador.ServeHTTP(candidatos, httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+"/bolsa:constituida:administrativo/candidatos?estado=disponible&limite=1", nil))
	if candidatos.Code != http.StatusOK {
		t.Fatalf("candidatos=%d body=%s", candidatos.Code, candidatos.Body.String())
	}
	var pagina struct {
		Data struct {
			Esquema    string `json:"esquema"`
			Candidatos []struct {
				Documento string `json:"documento_enmascarado"`
				Estado    string `json:"estado_clave"`
			} `json:"candidatos"`
			HayMas bool    `json:"hay_mas"`
			Cursor *string `json:"cursor_siguiente"`
		} `json:"data"`
	}
	if err := json.Unmarshal(candidatos.Body.Bytes(), &pagina); err != nil || pagina.Data.Esquema != "vec.bolsa.rrhh.candidatos.v1" || len(pagina.Data.Candidatos) != 1 || pagina.Data.Candidatos[0].Estado != "disponible" || !strings.HasPrefix(pagina.Data.Candidatos[0].Documento, "***") || !pagina.Data.HayMas || pagina.Data.Cursor == nil {
		t.Fatalf("contrato candidatos: %#v err=%v", pagina, err)
	}
	if !esRutaContratacionTemporalDesarrollo(httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+"/bolsa:constituida:administrativo/candidatos", nil)) {
		t.Fatal("la colección C20 no pasa por el guardián mTLS")
	}
}

func TestBolsasRRHHDesarrolloNoDependeDelDatasetDemoYFallaCerrado(t *testing.T) {
	rutas, colecciones, err := nuevasRutasBolsasRRHHDesarrollo(config.Config{BolsaDemoPath: "/no-debe-leerse/bolsa-demo.json"})
	if err != nil || len(rutas) != 1 || len(colecciones) != 1 {
		t.Fatalf("rutas RRHH: exactas=%d colecciones=%d error=%v", len(rutas), len(colecciones), err)
	}
	w := httptest.NewRecorder()
	rutas[0].Manejador.ServeHTTP(w, httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo, nil))
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), `"codigo":"servicio_no_disponible"`) {
		t.Fatalf("sin fuente durable: status=%d body=%s", w.Code, w.Body.String())
	}
	fallo := nuevoManejadorBolsasRRHHDesarrollo(func(context.Context) (datasetBolsasRRHHDesarrollo, error) {
		return datasetBolsasRRHHDesarrollo{}, errors.New("staging no disponible")
	})
	w = httptest.NewRecorder()
	fallo.ServeHTTP(w, httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+"/bolsa:constituida:administrativo/candidatos", nil))
	if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), "demo") {
		t.Fatalf("fallo de lectura durable: status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestFuenteConstituidaRRHHNoSirveEntradasSinFilaProtegida(t *testing.T) {
	vigente := ports.ConstitucionVigente{
		CategoriaRef: "categoria:rpt:administrativo",
		Bolsa: bolsadominio.BolsaConstituida{
			BolsaRef: "bolsa:constituida:administrativo", HuellaListadoSHA256: "huella:listado",
		},
		Instantanea: bolsadominio.InstantaneaOrdenBolsa{InstantaneaRef: "instantanea:administrativo", Version: 1},
	}
	for _, caso := range []struct {
		nombre string
		existe bool
	}{
		{nombre: "lote ausente", existe: false},
		{nombre: "fila ausente", existe: true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			fuente := &fuenteConstituidaRRHHDesarrollo{
				repositorio: repositorioBolsasRRHHPrueba{vigentes: []ports.ConstitucionVigente{vigente}, entradas: []ports.EntradaConstitucion{{Orden: 1, ParticipacionRef: "participacion:001", FilaNumero: 2}}},
				situaciones: situacionesBolsasRRHHPrueba{situacion: ports.SituacionParticipacion{ParticipacionRef: "participacion:001", Situacion: "disponible", Desde: time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)}},
				recuperador: recuperadorBolsasRRHHPrueba{existe: caso.existe},
				ahora:       func() time.Time { return time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC) },
			}
			if _, err := fuente.cargar(context.Background()); !errors.Is(err, ErrComposicionDesarrolloIncompleta) {
				t.Fatalf("error=%v; se esperaba fallo cerrado", err)
			}
		})
	}
}

func TestBolsasRRHHDesarrolloRechazaConsultaNoCanonica(t *testing.T) {
	manejador := manejadorBolsasRRHHPrueba()
	for _, caso := range []*http.Request{
		httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+"?limite=1", nil),
		httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+"/bolsa:constituida:administrativo/candidatos?estado=inventado", nil),
		httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+"/bolsa:constituida:administrativo/candidatos?limite=101", nil),
	} {
		w := httptest.NewRecorder()
		manejador.ServeHTTP(w, caso)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status=%d para %s", w.Code, caso.URL.String())
		}
	}
}

func TestBolsasRRHHDesarrolloDelegaB2ConIdempotencia(t *testing.T) {
	manejador := manejadorBolsasRRHHPrueba()
	llamadas := 0
	manejador.mutar = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		llamadas++
		if r.Header.Get("Idempotency-Key") != "b2-prueba-0001" {
			t.Fatal("la frontera perdió la clave de idempotencia")
		}
		w.WriteHeader(http.StatusCreated)
	})
	peticion := httptest.NewRequest(
		http.MethodPost,
		rutaBolsasRRHHDesarrollo+"/bolsa:01/candidatos/participacion:01/situacion",
		strings.NewReader(`{"situacion":"no_disponible"}`),
	)
	peticion.Header.Set("Idempotency-Key", "b2-prueba-0001")
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusCreated || llamadas != 1 {
		t.Fatalf("B2 no delegada: status=%d llamadas=%d", respuesta.Code, llamadas)
	}

	peticion = httptest.NewRequest(
		http.MethodPost,
		rutaBolsasRRHHDesarrollo+"/bolsa:01/candidatos/participacion:01/situacion",
		strings.NewReader(`{"situacion":"no_disponible"}`),
	)
	peticion.Header.Set("Idempotency-Key", "b2-prueba-0002")
	peticion.Header.Set("X-Actor", "inyectado")
	respuesta = httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusBadRequest || llamadas != 1 {
		t.Fatalf("cabecera de autoridad aceptada: status=%d llamadas=%d", respuesta.Code, llamadas)
	}
}

package ginpixapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const secretoSinteticoPrueba = "Bearer SECRETO-SINTETICO-NO-REGISTRAR"

type transporteFalso struct {
	mu       sync.Mutex
	llamadas int
	funcion  func(int, *http.Request) (*http.Response, error)
}

func (t *transporteFalso) RoundTrip(peticion *http.Request) (*http.Response, error) {
	t.mu.Lock()
	t.llamadas++
	llamada := t.llamadas
	funcion := t.funcion
	t.mu.Unlock()
	return funcion(llamada, peticion)
}

func (t *transporteFalso) total() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.llamadas
}

type autenticadorFalso struct {
	err   error
	mutar func(*http.Request)
	total atomic.Int32
}

func (a *autenticadorFalso) Autorizar(ctx context.Context, peticion *http.Request) error {
	a.total.Add(1)
	if err := ctx.Err(); err != nil {
		return err
	}
	peticion.Header.Set("Authorization", secretoSinteticoPrueba)
	if a.mutar != nil {
		a.mutar(peticion)
	}
	return a.err
}

type cuerpoRespuestaFalso struct {
	*bytes.Reader
	cerrados  *atomic.Int32
	errCierre error
}

func (c *cuerpoRespuestaFalso) Close() error {
	c.cerrados.Add(1)
	return c.errCierre
}

type cuerpoLecturaCancelada struct {
	cancelar context.CancelFunc
	cerrados *atomic.Int32
}

func (c *cuerpoLecturaCancelada) Read([]byte) (int, error) {
	c.cancelar()
	return 0, context.Canceled
}

func (c *cuerpoLecturaCancelada) Close() error {
	c.cerrados.Add(1)
	return nil
}

type autenticadorHastaCancelacion struct{ total atomic.Int32 }

func (a *autenticadorHastaCancelacion) Autorizar(
	ctx context.Context,
	_ *http.Request,
) error {
	a.total.Add(1)
	<-ctx.Done()
	return ctx.Err()
}

func TestAdaptadorEnviaReproduceYConsultaConReciboExacto(t *testing.T) {
	preparacion, _ := preparacionAPIGINPIXPrueba(t)
	cuerpoPreparado, _ := preparacion.Cuerpo()
	var cuerpos [][]byte
	var cabeceras []http.Header
	var cerrados atomic.Int32
	transporte := &transporteFalso{funcion: func(llamada int, peticion *http.Request) (*http.Response, error) {
		contenido, err := io.ReadAll(peticion.Body)
		if err != nil {
			t.Fatalf("leer petición falsa: %v", err)
		}
		cuerpos = append(cuerpos, append([]byte(nil), contenido...))
		cabeceras = append(cabeceras, peticion.Header.Clone())
		estado := http.StatusOK
		if llamada == 1 {
			estado = http.StatusCreated
		}
		return respuestaValidaPrueba(t, preparacion, estado, &cerrados), nil
	}}
	autenticador := &autenticadorFalso{}
	adaptador := nuevoAdaptadorPrueba(t, transporte, autenticador,
		politicaConfiguradaPrueba(time.Second, nil, 32*1024))

	primero, err := adaptador.Enviar(context.Background(), preparacion)
	if err != nil {
		t.Fatalf("enviar operación: %v", err)
	}
	replay, err := adaptador.Enviar(context.Background(), preparacion)
	if err != nil {
		t.Fatalf("replay exacto: %v", err)
	}
	consultado, err := adaptador.Consultar(context.Background(), preparacion)
	if err != nil {
		t.Fatalf("consultar operación: %v", err)
	}
	datosPrimero, _ := primero.Datos()
	datosReplay, _ := replay.Datos()
	datosConsulta, _ := consultado.Datos()
	if datosPrimero != datosReplay || datosPrimero != datosConsulta {
		t.Fatalf("recibos divergentes: %#v / %#v / %#v", datosPrimero, datosReplay, datosConsulta)
	}
	if transporte.total() != 3 || autenticador.total.Load() != 3 || cerrados.Load() != 3 {
		t.Fatalf("conteos inesperados: transporte=%d auth=%d cierres=%d",
			transporte.total(), autenticador.total.Load(), cerrados.Load())
	}
	if !bytes.Equal(cuerpos[0], cuerpoPreparado) || !bytes.Equal(cuerpos[1], cuerpoPreparado) {
		t.Fatal("envío o replay no conservó los bytes preparados")
	}
	if bytes.Contains(cuerpos[2], []byte(datoPersonalSinteticoPrueba)) ||
		!bytes.Contains(cuerpos[2], []byte("idempotencia_ginpix_api_0001")) {
		t.Fatal("consulta no minimizada")
	}
	for _, cabecera := range cabeceras {
		if cabecera.Get("Authorization") != secretoSinteticoPrueba ||
			cabecera.Get("Idempotency-Key") != "idempotencia_ginpix_api_0001" ||
			cabecera.Get("X-Correlation-ID") != "correlacion_ginpix_api_0001" {
			t.Fatal("cabeceras técnicas incompletas")
		}
	}
}

func TestAdaptadorCancelacionAntesDuranteYDespuesDelRecibo(t *testing.T) {
	preparacion, _ := preparacionAPIGINPIXPrueba(t)

	t.Run("antes", func(t *testing.T) {
		transporte := &transporteFalso{funcion: func(_ int, _ *http.Request) (*http.Response, error) {
			t.Fatal("se invocó transporte con contexto ya cancelado")
			return nil, nil
		}}
		adaptador := nuevoAdaptadorPrueba(t, transporte, &autenticadorFalso{}, politicaPrueba())
		ctx, cancelar := context.WithCancel(context.Background())
		cancelar()
		if _, err := adaptador.Enviar(ctx, preparacion); !errors.Is(err, context.Canceled) ||
			errors.Is(err, ErrOperacionAPIGINPIXIndeterminada) || transporte.total() != 0 {
			t.Fatalf("cancelación previa mal clasificada: %v / %d", err, transporte.total())
		}
	})

	t.Run("durante", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		transporte := &transporteFalso{funcion: func(_ int, _ *http.Request) (*http.Response, error) {
			cancelar()
			return nil, context.Canceled
		}}
		adaptador := nuevoAdaptadorPrueba(t, transporte, &autenticadorFalso{}, politicaPrueba())
		_, err := adaptador.Enviar(ctx, preparacion)
		if !errors.Is(err, context.Canceled) || !errors.Is(err, ErrOperacionAPIGINPIXIndeterminada) {
			t.Fatalf("cancelación durante efecto no quedó indeterminada: %v", err)
		}
	})

	t.Run("durante autenticacion antes de emitir", func(t *testing.T) {
		transporte := &transporteFalso{funcion: func(_ int, _ *http.Request) (*http.Response, error) {
			t.Fatal("la autenticacion cancelada alcanzo el transporte")
			return nil, nil
		}}
		autenticador := &autenticadorHastaCancelacion{}
		adaptador := nuevoAdaptadorPrueba(t, transporte, autenticador,
			politicaConfiguradaPrueba(20*time.Millisecond, nil, 32*1024))
		_, err := adaptador.Enviar(context.Background(), preparacion)
		if !errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(err, ErrOperacionAPIGINPIXIndeterminada) ||
			transporte.total() != 0 || autenticador.total.Load() != 1 {
			t.Fatalf("cancelacion durante autenticacion mal clasificada: %v", err)
		}
	})

	t.Run("durante lectura tras emitir", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		var cerrados atomic.Int32
		transporte := &transporteFalso{funcion: func(_ int, _ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode:    http.StatusOK,
				Header:        http.Header{"Content-Type": []string{"application/json"}},
				Body:          &cuerpoLecturaCancelada{cancelar: cancelar, cerrados: &cerrados},
				ContentLength: -1,
			}, nil
		}}
		adaptador := nuevoAdaptadorPrueba(t, transporte, &autenticadorFalso{}, politicaPrueba())
		_, err := adaptador.Enviar(ctx, preparacion)
		if !errors.Is(err, context.Canceled) ||
			!errors.Is(err, ErrOperacionAPIGINPIXIndeterminada) || cerrados.Load() != 1 {
			t.Fatalf("cancelacion durante lectura mal clasificada: %v / cierres=%d", err, cerrados.Load())
		}
	})

	t.Run("despues de recibo completo", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		var cerrados atomic.Int32
		transporte := &transporteFalso{funcion: func(_ int, _ *http.Request) (*http.Response, error) {
			cancelar()
			return respuestaValidaPrueba(t, preparacion, http.StatusOK, &cerrados), nil
		}}
		adaptador := nuevoAdaptadorPrueba(t, transporte, &autenticadorFalso{}, politicaPrueba())
		if _, err := adaptador.Enviar(ctx, preparacion); err != nil || cerrados.Load() != 1 {
			t.Fatalf("recibo completo no prevaleció: %v / cierres=%d", err, cerrados.Load())
		}
	})
}

func TestAdaptadorTimeoutTrasEmitirQuedaIndeterminado(t *testing.T) {
	preparacion, _ := preparacionAPIGINPIXPrueba(t)
	transporte := &transporteFalso{funcion: func(_ int, peticion *http.Request) (*http.Response, error) {
		<-peticion.Context().Done()
		return nil, peticion.Context().Err()
	}}
	adaptador := nuevoAdaptadorPrueba(t, transporte, &autenticadorFalso{},
		politicaConfiguradaPrueba(time.Second, nil, 32*1024))
	_, err := adaptador.Enviar(context.Background(), preparacion)
	if !errors.Is(err, context.DeadlineExceeded) ||
		!errors.Is(err, ErrOperacionAPIGINPIXIndeterminada) || transporte.total() != 1 {
		t.Fatalf("timeout mal clasificado: %v / llamadas=%d", err, transporte.total())
	}
}

func TestAdaptadorNoInventaCodigosReintentablesOTerminales(t *testing.T) {
	preparacion, _ := preparacionAPIGINPIXPrueba(t)
	casos := []struct {
		nombre   string
		consulta bool
		codigo   int
		esperado error
	}{
		{"envio 408", false, http.StatusRequestTimeout, ErrOperacionAPIGINPIXIndeterminada},
		{"envio 409", false, http.StatusConflict, ErrOperacionAPIGINPIXIndeterminada},
		{"envio 429", false, http.StatusTooManyRequests, ErrOperacionAPIGINPIXIndeterminada},
		{"envio 500", false, http.StatusInternalServerError, ErrOperacionAPIGINPIXIndeterminada},
		{"envio 202 ambiguo", false, http.StatusAccepted, ErrOperacionAPIGINPIXIndeterminada},
		{"consulta 404", true, http.StatusNotFound, ErrConsultaAPIGINPIXNoDisponible},
		{"consulta 503", true, http.StatusServiceUnavailable, ErrConsultaAPIGINPIXNoDisponible},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			var cerrados atomic.Int32
			autenticador := &autenticadorFalso{}
			transporte := &transporteFalso{funcion: func(_ int, _ *http.Request) (*http.Response, error) {
				return respuestaSimplePrueba(caso.codigo, "text/plain", []byte("detalle no confiable"), &cerrados), nil
			}}
			adaptador := nuevoAdaptadorPrueba(t, transporte, autenticador, politicaPrueba())
			var err error
			if caso.consulta {
				_, err = adaptador.Consultar(context.Background(), preparacion)
			} else {
				_, err = adaptador.Enviar(context.Background(), preparacion)
			}
			if !errors.Is(err, caso.esperado) || transporte.total() != 1 ||
				cerrados.Load() != 1 || autenticador.total.Load() != 1 {
				t.Fatalf("clasificación=%v llamadas=%d cierres=%d", err, transporte.total(), cerrados.Load())
			}
		})
	}
}

func TestAdaptadorDeniegaRespuestasExcesivasMalformadasODesligadas(t *testing.T) {
	preparacion, _ := preparacionAPIGINPIXPrueba(t)
	metadatos, _ := preparacion.Metadatos()
	datosValidos := datosReciboExternoPrueba(metadatos)
	casos := map[string]func(*atomic.Int32) (*http.Response, error){
		"cuerpo excesivo": func(c *atomic.Int32) (*http.Response, error) {
			return respuestaSimplePrueba(http.StatusOK, "application/json", bytes.Repeat([]byte("x"), 2049), c), nil
		},
		"JSON malformado": func(c *atomic.Int32) (*http.Response, error) {
			return respuestaSimplePrueba(http.StatusOK, "application/json", []byte(`{"esquema":`), c), nil
		},
		"JSON duplicado": func(c *atomic.Int32) (*http.Response, error) {
			contenido, _ := json.Marshal(datosValidos)
			contenido = append(contenido[:len(contenido)-1], []byte(
				`,"esquema":"`+EsquemaReciboAPIGINPIXV1+`"}`,
			)...)
			return respuestaSimplePrueba(http.StatusOK, "application/json", contenido, c), nil
		},
		"JSON con cola": func(c *atomic.Int32) (*http.Response, error) {
			contenido, _ := json.Marshal(datosValidos)
			contenido = append(contenido, []byte("\n{}")...)
			return respuestaSimplePrueba(http.StatusOK, "application/json", contenido, c), nil
		},
		"Content-Type": func(c *atomic.Int32) (*http.Response, error) {
			contenido, _ := json.Marshal(datosValidos)
			return respuestaSimplePrueba(http.StatusOK, "text/plain", contenido, c), nil
		},
		"ligadura alterada": func(c *atomic.Int32) (*http.Response, error) {
			datos := datosValidos
			datos.IdempotenciaRef = "idempotencia_ginpix_api_alterada"
			contenido, _ := json.Marshal(datos)
			return respuestaSimplePrueba(http.StatusOK, "application/json", contenido, c), nil
		},
		"respuesta parcial": func(c *atomic.Int32) (*http.Response, error) {
			datos := datosValidos
			datos.EvidenciaExternaRef = ""
			contenido, _ := json.Marshal(datos)
			return respuestaSimplePrueba(http.StatusOK, "application/json", contenido, c), nil
		},
		"campo desconocido": func(c *atomic.Int32) (*http.Response, error) {
			contenido, _ := json.Marshal(datosValidos)
			contenido = append(contenido[:len(contenido)-1], []byte(`,"estado_inventado":"exito"}`)...)
			return respuestaSimplePrueba(http.StatusOK, "application/json", contenido, c), nil
		},
		"Set-Cookie": func(c *atomic.Int32) (*http.Response, error) {
			respuesta := respuestaValidaPrueba(t, preparacion, http.StatusOK, c)
			respuesta.Header.Add("Set-Cookie", "sesion="+secretoSinteticoPrueba)
			return respuesta, nil
		},
		"fallo de Close": func(c *atomic.Int32) (*http.Response, error) {
			respuesta := respuestaValidaPrueba(t, preparacion, http.StatusOK, c)
			respuesta.Body.(*cuerpoRespuestaFalso).errCierre = errors.New("cierre opaco")
			return respuesta, nil
		},
		"resultado mas error": func(c *atomic.Int32) (*http.Response, error) {
			return respuestaValidaPrueba(t, preparacion, http.StatusOK, c),
				errors.New("fallo " + secretoSinteticoPrueba)
		},
	}
	for nombre, respuesta := range casos {
		t.Run(nombre, func(t *testing.T) {
			var cerrados atomic.Int32
			transporte := &transporteFalso{funcion: func(_ int, _ *http.Request) (*http.Response, error) {
				return respuesta(&cerrados)
			}}
			adaptador := nuevoAdaptadorPrueba(t, transporte, &autenticadorFalso{},
				politicaConfiguradaPrueba(time.Second, nil, 2048))
			_, err := adaptador.Enviar(context.Background(), preparacion)
			if !errors.Is(err, ErrOperacionAPIGINPIXIndeterminada) || cerrados.Load() != 1 ||
				strings.Contains(err.Error(), secretoSinteticoPrueba) ||
				strings.Contains(err.Error(), datoPersonalSinteticoPrueba) {
				t.Fatalf("respuesta ambigua no saneada: %v / cierres=%d", err, cerrados.Load())
			}
		})
	}
}

func TestAdaptadorReciboDeniegaCadaLigaduraAlterada(t *testing.T) {
	preparacion, _ := preparacionAPIGINPIXPrueba(t)
	metadatos, _ := preparacion.Metadatos()
	otraHuella := strings.Repeat("b", 64)
	alteraciones := map[string]func(*DatosReciboExterno){
		"version OCC":   func(d *DatosReciboExterno) { d.VersionExpediente++ },
		"expediente":    func(d *DatosReciboExterno) { d.ExpedienteRef = "expediente_api_otro" },
		"incorporacion": func(d *DatosReciboExterno) { d.IncorporacionRef = "incorporacion_api_otra" },
		"correlacion":   func(d *DatosReciboExterno) { d.CorrelacionRef = "correlacion_api_otra" },
		"idempotencia":  func(d *DatosReciboExterno) { d.IdempotenciaRef = "idempotencia_api_otra" },
		"huella modelo": func(d *DatosReciboExterno) { d.ModeloHuellaSHA256 = otraHuella },
		"mapeo":         func(d *DatosReciboExterno) { d.MapeoRef = "mapeo_api_otro" },
		"version mapeo": func(d *DatosReciboExterno) { d.MapeoVersion++ },
		"huella mapeo":  func(d *DatosReciboExterno) { d.MapeoHuellaSHA256 = otraHuella },
		"huella carga":  func(d *DatosReciboExterno) { d.CargaHuellaSHA256 = otraHuella },
		"huella cuerpo": func(d *DatosReciboExterno) { d.CuerpoHuellaSHA256 = otraHuella },
		"recibo incorporacion": func(d *DatosReciboExterno) {
			d.ReciboIncorporacionRef = "recibo_incorporacion_api_otro"
		},
		"resultado Personal": func(d *DatosReciboExterno) {
			d.ResultadoPersonalRef = "resultado_personal_api_otro"
		},
		"recibo Personal": func(d *DatosReciboExterno) { d.ReciboPersonalRef = "recibo_personal_api_otro" },
	}
	for nombre, alterar := range alteraciones {
		t.Run(nombre, func(t *testing.T) {
			datos := datosReciboExternoPrueba(metadatos)
			alterar(&datos)
			contenido, _ := json.Marshal(datos)
			var cerrados atomic.Int32
			transporte := &transporteFalso{funcion: func(_ int, _ *http.Request) (*http.Response, error) {
				return respuestaSimplePrueba(http.StatusOK, "application/json", contenido, &cerrados), nil
			}}
			adaptador := nuevoAdaptadorPrueba(t, transporte, &autenticadorFalso{}, politicaPrueba())
			if _, err := adaptador.Enviar(context.Background(), preparacion); !errors.Is(err, ErrOperacionAPIGINPIXIndeterminada) ||
				cerrados.Load() != 1 {
				t.Fatalf("ligadura alterada aceptada: %v", err)
			}
		})
	}
}

func TestPoliticaSellaCanonExactoYRechazaEsperasAntesDelTransporte(t *testing.T) {
	base := politicaPrueba()
	mutaciones := map[string]func(*Politica){
		"referencia": func(p *Politica) { p.Referencia = "politica_http_ginpix_api_otra" },
		"version":    func(p *Politica) { p.Version++ },
		"timeout":    func(p *Politica) { p.TiempoMaximo += time.Nanosecond },
		"limite":     func(p *Politica) { p.MaximoBytesRespuesta++ },
		"huella":     func(p *Politica) { p.HuellaSHA256 = strings.Repeat("a", 64) },
	}
	transporte := &transporteFalso{funcion: func(_ int, _ *http.Request) (*http.Response, error) {
		t.Fatal("una politica sin sello exacto alcanzo el transporte")
		return nil, nil
	}}
	autenticador := &autenticadorFalso{}
	for nombre, mutar := range mutaciones {
		t.Run(nombre, func(t *testing.T) {
			adulterada := base
			mutar(&adulterada)
			candidato, err := Nuevo(Configuracion{
				URLEnvio:    "https://ginpix.invalid/operaciones/enviar",
				URLConsulta: "https://ginpix.invalid/operaciones/consultar",
				Politica:    adulterada,
			}, transporte, autenticador)
			if candidato != nil || !errors.Is(err, ErrConfiguracionAPIGINPIXInvalida) {
				t.Fatalf("politica adulterada aceptada: %#v / %v", adulterada, err)
			}
		})
	}

	conEspera, err := SellarPolitica(Politica{
		Referencia: base.Referencia, Version: base.Version, TiempoMaximo: base.TiempoMaximo,
		EsperasReintento: []time.Duration{0}, MaximoBytesRespuesta: base.MaximoBytesRespuesta,
	})
	if err != nil || conEspera.HuellaSHA256 == base.HuellaSHA256 {
		t.Fatalf("la espera no quedo sellada en el canon: %#v / %v", conEspera, err)
	}
	candidato, err := Nuevo(Configuracion{
		URLEnvio:    "https://ginpix.invalid/operaciones/enviar",
		URLConsulta: "https://ginpix.invalid/operaciones/consultar",
		Politica:    conEspera,
	}, transporte, autenticador)
	if candidato != nil || !errors.Is(err, ErrConfiguracionAPIGINPIXInvalida) ||
		transporte.total() != 0 || autenticador.total.Load() != 0 {
		t.Fatalf("politica con espera no se rechazo antes del transporte: %v", err)
	}
}

func TestAdaptadorDeniegaAutenticacionYConfiguracionSinFiltrarSecretos(t *testing.T) {
	preparacion, _ := preparacionAPIGINPIXPrueba(t)
	transporte := &transporteFalso{funcion: func(_ int, _ *http.Request) (*http.Response, error) {
		return nil, errors.New("transporte " + secretoSinteticoPrueba + datoPersonalSinteticoPrueba)
	}}
	autenticador := &autenticadorFalso{err: errors.New("auth " + secretoSinteticoPrueba)}
	adaptador := nuevoAdaptadorPrueba(t, transporte, autenticador, politicaPrueba())
	_, err := adaptador.Enviar(context.Background(), preparacion)
	if !errors.Is(err, ErrAutenticacionAPIGINPIXFallida) || transporte.total() != 0 ||
		strings.Contains(err.Error(), secretoSinteticoPrueba) {
		t.Fatalf("fallo de autenticación filtrado o emitido: %v / %d", err, transporte.total())
	}

	autenticador.err = nil
	_, err = adaptador.Enviar(context.Background(), preparacion)
	if !errors.Is(err, ErrOperacionAPIGINPIXIndeterminada) ||
		strings.Contains(err.Error(), secretoSinteticoPrueba) ||
		strings.Contains(err.Error(), datoPersonalSinteticoPrueba) {
		t.Fatalf("fallo de transporte filtró detalles: %v", err)
	}
	transporteNoInvocable := &transporteFalso{funcion: func(_ int, _ *http.Request) (*http.Response, error) {
		t.Fatal("una autenticación que añadió Cookie alcanzó el transporte")
		return nil, nil
	}}
	autenticadorHostil := &autenticadorFalso{mutar: func(peticion *http.Request) {
		peticion.Header.Set("Cookie", "sesion="+secretoSinteticoPrueba)
	}}
	adaptadorHostil := nuevoAdaptadorPrueba(t, transporteNoInvocable, autenticadorHostil, politicaPrueba())
	if _, err := adaptadorHostil.Enviar(context.Background(), preparacion); !errors.Is(err, ErrAutenticacionAPIGINPIXFallida) ||
		transporteNoInvocable.total() != 0 || strings.Contains(err.Error(), secretoSinteticoPrueba) {
		t.Fatalf("mutación de autenticación no denegada: %v", err)
	}

	configuracionesInvalidas := []Configuracion{
		{},
		{URLEnvio: "http://ginpix.invalid/envio", URLConsulta: "https://ginpix.invalid/consulta", Politica: politicaPrueba()},
		{URLEnvio: "https://usuario:clave@ginpix.invalid/envio", URLConsulta: "https://ginpix.invalid/consulta", Politica: politicaPrueba()},
		{URLEnvio: "https://ginpix.invalid/envio", URLConsulta: "https://otro.invalid/consulta", Politica: politicaPrueba()},
	}
	for _, configuracion := range configuracionesInvalidas {
		if candidato, err := Nuevo(configuracion, transporte, autenticador); candidato != nil ||
			!errors.Is(err, ErrConfiguracionAPIGINPIXInvalida) {
			t.Fatalf("configuración insegura aceptada: %#v / %v", configuracion, err)
		}
	}
}

func nuevoAdaptadorPrueba(
	t *testing.T,
	transporte http.RoundTripper,
	autenticador ProveedorAutenticacionOpaca,
	politica Politica,
) *Adaptador {
	t.Helper()
	adaptador, err := Nuevo(Configuracion{
		URLEnvio:    "https://ginpix.invalid/operaciones/enviar",
		URLConsulta: "https://ginpix.invalid/operaciones/consultar",
		Politica:    politica,
	}, transporte, autenticador)
	if err != nil {
		t.Fatalf("crear adaptador sintético: %v", err)
	}
	return adaptador
}

func politicaPrueba() Politica {
	return politicaConfiguradaPrueba(time.Second, nil, 32*1024)
}

func politicaConfiguradaPrueba(
	tiempo time.Duration,
	esperas []time.Duration,
	maximo int64,
) Politica {
	politica, err := SellarPolitica(Politica{
		Referencia: "politica_http_ginpix_api_0001", Version: 2,
		TiempoMaximo:     tiempo,
		EsperasReintento: esperas, MaximoBytesRespuesta: maximo,
	})
	if err != nil {
		panic(err)
	}
	return politica
}

func respuestaValidaPrueba(
	t *testing.T,
	preparacion Preparacion,
	estado int,
	cerrados *atomic.Int32,
) *http.Response {
	t.Helper()
	metadatos, err := preparacion.Metadatos()
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := json.Marshal(datosReciboExternoPrueba(metadatos))
	if err != nil {
		t.Fatal(err)
	}
	return respuestaSimplePrueba(estado, "application/json; charset=utf-8", contenido, cerrados)
}

func datosReciboExternoPrueba(m MetadatosOperacion) DatosReciboExterno {
	return DatosReciboExterno{
		Esquema: EsquemaReciboAPIGINPIXV1, Version: VersionReciboAPIGINPIXV1,
		ReciboExternoRef:             "recibo_externo_ginpix_api_0001",
		EvidenciaExternaRef:          "evidencia_externa_ginpix_api_0001",
		EvidenciaExternaHuellaSHA256: strings.Repeat("e", 64),
		VersionExpediente:            m.VersionExpediente, ExpedienteRef: m.ExpedienteRef,
		IncorporacionRef: m.IncorporacionRef, CorrelacionRef: m.CorrelacionRef,
		IdempotenciaRef: m.IdempotenciaRef, ModeloHuellaSHA256: m.ModeloHuellaSHA256,
		MapeoRef: m.MapeoRef, MapeoVersion: m.MapeoVersion,
		MapeoHuellaSHA256: m.MapeoHuellaSHA256, CargaHuellaSHA256: m.CargaHuellaSHA256,
		CuerpoHuellaSHA256:     m.CuerpoHuellaSHA256,
		ReciboIncorporacionRef: m.ReciboIncorporacionRef,
		ResultadoPersonalRef:   m.ResultadoPersonalRef, ReciboPersonalRef: m.ReciboPersonalRef,
	}
}

func respuestaSimplePrueba(
	estado int,
	contentType string,
	contenido []byte,
	cerrados *atomic.Int32,
) *http.Response {
	if contenido == nil {
		contenido = []byte("sin resultado")
	}
	cuerpo := &cuerpoRespuestaFalso{Reader: bytes.NewReader(contenido), cerrados: cerrados}
	return &http.Response{
		StatusCode: estado, Header: http.Header{"Content-Type": []string{contentType}},
		Body: cuerpo, ContentLength: int64(len(contenido)),
	}
}

package application

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestOperacionAnalisisReplayExigeIdentidadSemanticaCompleta(
	t *testing.T,
) {
	const marcaBase = "-identidad-completa-sintetica"
	casos := []struct {
		nombre    string
		modificar func(
			*testing.T,
			*escenarioOperacionAnalisisSaneado,
			*dependenciasOperacionAnalisisSaneado,
			*SolicitudRegistrarAnalisis,
		)
	}{
		{"actor", func(t *testing.T, e *escenarioOperacionAnalisisSaneado, d *dependenciasOperacionAnalisisSaneado, s *SolicitudRegistrarAnalisis) {
			d.contextos.contexto = contextoAutorizacionAltaV3PruebaConMarcas(
				t, e.instante, "-actor-distinto-sintetico", marcaBase,
			)
			vinculo, err := d.contextos.contexto.Vinculo.Datos()
			if err != nil {
				t.Fatal(err)
			}
			s.AutenticacionRef = vinculo.AutenticacionRef
			s.SesionRef = vinculo.SesionRef
			s.PerfilRef = vinculo.PerfilActivoRef
		}},
		{"perfil", func(t *testing.T, e *escenarioOperacionAnalisisSaneado, d *dependenciasOperacionAnalisisSaneado, s *SolicitudRegistrarAnalisis) {
			d.contextos.contexto = contextoAutorizacionAltaV3PruebaConMarcas(
				t, e.instante, marcaBase, "-perfil-distinto-sintetico",
			)
			vinculo, err := d.contextos.contexto.Vinculo.Datos()
			if err != nil {
				t.Fatal(err)
			}
			s.AutenticacionRef = vinculo.AutenticacionRef
			s.SesionRef = vinculo.SesionRef
			s.PerfilRef = vinculo.PerfilActivoRef
		}},
		{"uuid_reintento", func(_ *testing.T, _ *escenarioOperacionAnalisisSaneado, _ *dependenciasOperacionAnalisisSaneado, s *SolicitudRegistrarAnalisis) {
			s.ClaveIdempotencia = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
		}},
		{"organizacion", func(_ *testing.T, _ *escenarioOperacionAnalisisSaneado, _ *dependenciasOperacionAnalisisSaneado, s *SolicitudRegistrarAnalisis) {
			s.OrganizacionRef = "organizacion:distinta-sintetica-001"
		}},
		{"expediente", func(_ *testing.T, _ *escenarioOperacionAnalisisSaneado, _ *dependenciasOperacionAnalisisSaneado, s *SolicitudRegistrarAnalisis) {
			s.ExpedienteRef = "expediente:distinto-sintetico-001"
		}},
		{"version", func(_ *testing.T, _ *escenarioOperacionAnalisisSaneado, _ *dependenciasOperacionAnalisisSaneado, s *SolicitudRegistrarAnalisis) {
			s.VersionEsperada++
		}},
		{"artefacto", func(_ *testing.T, _ *escenarioOperacionAnalisisSaneado, _ *dependenciasOperacionAnalisisSaneado, s *SolicitudRegistrarAnalisis) {
			s.ArtefactoRef = "artefacto:distinto-sintetico-001"
		}},
		{"modalidad", func(_ *testing.T, _ *escenarioOperacionAnalisisSaneado, _ *dependenciasOperacionAnalisisSaneado, s *SolicitudRegistrarAnalisis) {
			s.DatosFuncionales.ModalidadClave = "modalidad.distinta"
		}},
		{"categoria", func(_ *testing.T, _ *escenarioOperacionAnalisisSaneado, _ *dependenciasOperacionAnalisisSaneado, s *SolicitudRegistrarAnalisis) {
			s.DatosFuncionales.CategoriaRef = "categoria:distinta-sintetica-001"
		}},
		{"grupo_subgrupo", func(_ *testing.T, _ *escenarioOperacionAnalisisSaneado, _ *dependenciasOperacionAnalisisSaneado, s *SolicitudRegistrarAnalisis) {
			s.DatosFuncionales.GrupoSubgrupo = "A1"
		}},
		{"causa", func(_ *testing.T, _ *escenarioOperacionAnalisisSaneado, _ *dependenciasOperacionAnalisisSaneado, s *SolicitudRegistrarAnalisis) {
			s.DatosFuncionales.CausaClave = "causa.distinta"
		}},
		{"periodo_inicio", func(_ *testing.T, _ *escenarioOperacionAnalisisSaneado, _ *dependenciasOperacionAnalisisSaneado, s *SolicitudRegistrarAnalisis) {
			s.DatosFuncionales.Periodo.Inicio =
				s.DatosFuncionales.Periodo.Inicio.Add(24 * time.Hour)
		}},
		{"periodo_fin", func(_ *testing.T, _ *escenarioOperacionAnalisisSaneado, _ *dependenciasOperacionAnalisisSaneado, s *SolicitudRegistrarAnalisis) {
			s.DatosFuncionales.Periodo.Fin =
				s.DatosFuncionales.Periodo.Fin.Add(-24 * time.Hour)
		}},
		{"jornada", func(_ *testing.T, _ *escenarioOperacionAnalisisSaneado, _ *dependenciasOperacionAnalisisSaneado, s *SolicitudRegistrarAnalisis) {
			s.DatosFuncionales.PorcentajeJornada = 5_000
		}},
		{"entrada_rc_ref", func(_ *testing.T, _ *escenarioOperacionAnalisisSaneado, _ *dependenciasOperacionAnalisisSaneado, s *SolicitudRegistrarAnalisis) {
			s.DatosFuncionales.EntradaRC.Referencia =
				"entrada:rc-distinta-sintetica-001"
		}},
		{"entrada_rc_huella", func(_ *testing.T, _ *escenarioOperacionAnalisisSaneado, _ *dependenciasOperacionAnalisisSaneado, s *SolicitudRegistrarAnalisis) {
			s.DatosFuncionales.EntradaRC.HuellaSHA256 =
				strings.Repeat("7", 64)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			escenario := nuevoEscenarioOperacionAnalisisSaneado(
				t,
				ports.OperacionRegistrarAnalisis,
				marcaBase,
			)
			servicio, d :=
				construirServicioOperacionAnalisisSaneado(t, escenario)
			confirmado, err := servicio.Registrar(
				context.Background(),
				escenario.registrar,
			)
			if err != nil {
				t.Fatal(err)
			}
			d.preparaciones.consultaConfirmada = &confirmado
			cambiada := escenario.registrar
			caso.modificar(t, &escenario, d, &cambiada)

			recibo, err := servicio.Registrar(
				context.Background(),
				cambiada,
			)
			if recibo != (ports.ReciboOperacionAnalisis{}) ||
				!errors.Is(
					err,
					ErrResultadoOperacionAnalisisNoConfiable,
				) ||
				d.artefactos.llamadas != 1 ||
				d.transaccion.commits != 1 {
				t.Fatalf(
					"el recibo cruzó %s: %#v, %v",
					caso.nombre,
					recibo,
					err,
				)
			}
		})
	}
}

func TestOperacionAnalisisReplayExigeMotivoRectificacionExacto(
	t *testing.T,
) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRectificarAnalisis,
		"-motivo-replay-sintetico",
	)
	servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
	confirmado, err := servicio.Rectificar(
		context.Background(),
		escenario.rectificar,
	)
	if err != nil {
		t.Fatal(err)
	}
	d.preparaciones.consultaConfirmada = &confirmado
	cambiada := escenario.rectificar
	cambiada.MotivoRectificacionClave =
		"contratacion_temporal.analisis.rectificacion.otro_motivo"

	recibo, err := servicio.Rectificar(context.Background(), cambiada)
	if recibo != (ports.ReciboOperacionAnalisis{}) ||
		!errors.Is(err, ErrResultadoOperacionAnalisisNoConfiable) ||
		d.artefactos.llamadas != 1 || d.transaccion.commits != 1 {
		t.Fatalf("el recibo cruzó el motivo: %#v, %v", recibo, err)
	}
}

func TestOperacionAnalisisReplayRechazaReciboVacioOManipulado(
	t *testing.T,
) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRegistrarAnalisis,
		"-recibo-consulta-manipulado-sintetico",
	)
	servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
	confirmado, err := servicio.Registrar(
		context.Background(),
		escenario.registrar,
	)
	if err != nil {
		t.Fatal(err)
	}
	casos := map[string]ports.ReciboOperacionAnalisis{
		"vacio": {},
		"huella_vacia": func() ports.ReciboOperacionAnalisis {
			recibo := confirmado
			recibo.HuellaConsultaHMAC = ""
			return recibo
		}(),
		"huella_ajena_valida": func() ports.ReciboOperacionAnalisis {
			recibo := confirmado
			recibo.HuellaConsultaHMAC =
				"hmac-sha256:vec.contratacion-temporal.analisis.huella-semantica/v2:" +
					strings.Repeat("a", 64)
			return recibo
		}(),
	}
	for nombre, reciboForzado := range casos {
		t.Run(nombre, func(t *testing.T) {
			d.preparaciones.consultaConfirmada = &reciboForzado
			recibo, err := servicio.Registrar(
				context.Background(),
				escenario.registrar,
			)
			if recibo != (ports.ReciboOperacionAnalisis{}) ||
				!errors.Is(
					err,
					ErrResultadoOperacionAnalisisNoConfiable,
				) {
				t.Fatalf("recibo aceptado: %#v, %v", recibo, err)
			}
		})
	}
}

func TestOperacionAnalisisReplayConfirmadoPrevaleceTrasCancelacion(
	t *testing.T,
) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRegistrarAnalisis,
		"-replay-cancelado-sintetico",
	)
	servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
	confirmado, err := servicio.Registrar(
		context.Background(),
		escenario.registrar,
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	d.preparaciones.consultaConfirmada = &confirmado
	d.preparaciones.despuesConsulta = cancelar

	recibo, err := servicio.Registrar(ctx, escenario.registrar)
	if err != nil || recibo != confirmado ||
		d.artefactos.llamadas != 1 || d.transaccion.commits != 1 {
		t.Fatalf("el commit confirmado no prevaleció: %#v, %v", recibo, err)
	}
}

func TestIdentidadReplayEsSeguraEnConcurrencia(t *testing.T) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRegistrarAnalisis,
		"-replay-concurrente-sintetico",
	)
	servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
	recibo, err := servicio.Registrar(
		context.Background(),
		escenario.registrar,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := d.preparaciones.solicitudConsulta
	if _, expuesta := reflect.TypeOf(solicitud).MethodByName(
		"ParActivo",
	); expuesta {
		t.Fatal("el adaptador puede extraer la huella semántica de consulta")
	}
	if _, err := json.Marshal(solicitud); !errors.Is(
		err,
		ports.ErrSerializacionOperacionAnalisisProhibida,
	) {
		t.Fatalf("la identidad sellada se expuso como DTO: %v", err)
	}
	const trabajadores = 64
	errores := make(chan error, trabajadores)
	var grupo sync.WaitGroup
	for indice := 0; indice < trabajadores; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			errores <- recibo.ValidarParaConsulta(solicitud)
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatalf("validación concurrente inestable: %v", err)
		}
	}
}

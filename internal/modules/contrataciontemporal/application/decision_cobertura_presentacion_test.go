package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"sync"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestProponerCoberturaPresentaVistaDinamicaSinEfectos(
	t *testing.T,
) {
	escenario := nuevoEscenarioPresentacionCobertura(
		t,
		viasPresentacionCoberturaPrueba(3),
	)
	presentacion, err := escenario.servicio.Proponer(
		context.Background(),
		escenario.solicitud,
	)
	if err != nil {
		t.Fatalf(
			"%v (contextos=%d, accesos=%d, analisis=%d, gobierno=%d, consultas=%d)",
			err,
			escenario.contextos.total(),
			escenario.accesos.total(),
			escenario.analisis.total(),
			escenario.gobierno.total(),
			escenario.global.generador.llamadas(),
		)
	}
	if presentacion.Estado != domain.PropuestaCoberturaViable ||
		presentacion.ViaRecomendada != "via_global_01" ||
		len(presentacion.Evaluaciones) != 3 ||
		presentacion.IdentidadSemantica.Validar() != nil {
		t.Fatalf("presentación dinámica incompleta: %+v", presentacion)
	}
	for indice, evaluacion := range presentacion.Evaluaciones {
		esperada := domain.ClaveCatalogo(fmt.Sprintf(
			"via_global_%02d",
			indice+1,
		))
		if evaluacion.ViaClave != esperada {
			t.Fatalf("la vía %d fue compilada o reordenada: %s", indice, evaluacion.ViaClave)
		}
	}
	if escenario.contextos.total() != 1 ||
		escenario.accesos.total() != 2 ||
		escenario.analisis.total() != 1 ||
		escenario.gobierno.total() != 1 {
		t.Fatal("el caso de uso no respetó la secuencia de autoridades")
	}
	exigirCeroConsumoPreparacionGlobal(t, escenario.global)
	exigirServicioPresentacionSinPuertosMutantes(t)
	exigirSolicitudPresentacionMinimizada(t)
	exigirVistaPresentacionMinimizada(t)
}

func TestProponerCoberturaAutenticadoSinPermisoNoRevelaExistencia(
	t *testing.T,
) {
	escenario := nuevoEscenarioPresentacionCobertura(
		t,
		viasPresentacionCoberturaPrueba(2),
	)
	escenario.accesos.err = ErrPresentacionPropuestaCoberturaDenegada
	_, err := escenario.servicio.Proponer(
		context.Background(),
		escenario.solicitud,
	)
	if !errors.Is(err, ErrPresentacionPropuestaCoberturaDenegada) {
		t.Fatalf("la denegación de lectura no cerró el acceso: %v", err)
	}
	if escenario.contextos.total() != 1 ||
		escenario.accesos.total() != 1 ||
		escenario.analisis.total() != 0 ||
		escenario.gobierno.total() != 0 ||
		escenario.global.generador.llamadas() != 0 {
		t.Fatal("se reveló o consultó información después de denegar")
	}
	exigirCeroConsumoPreparacionGlobal(t, escenario.global)
}

func TestProponerCoberturaReautorizaJustoAntesDeRevelar(t *testing.T) {
	escenario := nuevoEscenarioPresentacionCobertura(
		t,
		viasPresentacionCoberturaPrueba(2),
	)
	escenario.accesos.errores = map[int]error{
		2: ErrPresentacionPropuestaCoberturaDenegada,
	}
	presentacion, err := escenario.servicio.Proponer(
		context.Background(),
		escenario.solicitud,
	)
	if !errors.Is(err, ErrPresentacionPropuestaCoberturaDenegada) ||
		!reflect.DeepEqual(presentacion, PresentacionPropuestaCobertura{}) {
		t.Fatalf("la revocación final dejó revelar la vista: %+v, %v", presentacion, err)
	}
	if escenario.accesos.total() != 2 ||
		escenario.analisis.total() != 1 ||
		escenario.gobierno.total() != 1 ||
		escenario.global.generador.llamadas() != 2 {
		t.Fatal("la reautorización no ocupó la frontera final esperada")
	}
	exigirCeroConsumoPreparacionGlobal(t, escenario.global)
}

func TestProponerCoberturaRepresentaAusenciaYConflictoSinListasFijas(
	t *testing.T,
) {
	t.Run("ausencia acreditada", func(t *testing.T) {
		escenario := nuevoEscenarioPresentacionCobertura(
			t,
			viasPresentacionCoberturaPrueba(1),
		)
		configurarResultadosPresentacionCobertura(
			t,
			escenario,
			func(_ ports.SolicitudConsultarCobertura) domain.ResultadoComprobacion {
				return domain.ComprobacionNoConsta
			},
		)
		presentacion, err := escenario.servicio.Proponer(
			context.Background(),
			escenario.solicitud,
		)
		if err != nil {
			t.Fatal(err)
		}
		if presentacion.Estado != domain.PropuestaCoberturaViable ||
			len(presentacion.Evaluaciones) != 1 ||
			len(presentacion.Evaluaciones[0].AusenciasAdmitidas) != 1 {
			t.Fatalf("se confundió ausencia acreditada: %+v", presentacion)
		}
	})

	t.Run("resultados conflictivos", func(t *testing.T) {
		vias := viasPresentacionCoberturaPrueba(2)
		vias[1].Comprobaciones[0].Clave = vias[0].Comprobaciones[0].Clave
		escenario := nuevoEscenarioPresentacionCobertura(t, vias)
		configurarResultadosPresentacionCobertura(
			t,
			escenario,
			func(s ports.SolicitudConsultarCobertura) domain.ResultadoComprobacion {
				if s.ViaClave == vias[0].Clave {
					return domain.ComprobacionAfirmativa
				}
				return domain.ComprobacionNegativa
			},
		)
		presentacion, err := escenario.servicio.Proponer(
			context.Background(),
			escenario.solicitud,
		)
		if err != nil {
			t.Fatal(err)
		}
		if presentacion.Estado != domain.PropuestaCoberturaConflictiva ||
			presentacion.ViaRecomendada != "" {
			t.Fatalf("el conflicto no quedó visible y bloqueado: %+v", presentacion)
		}
	})
}

func TestProponerCoberturaRechazaEntradaAntesDeConsultarAutoridades(
	t *testing.T,
) {
	base := nuevoEscenarioPresentacionCobertura(
		t,
		viasPresentacionCoberturaPrueba(1),
	)
	casos := map[string]func(*SolicitudProponerCobertura){
		"autenticación": func(s *SolicitudProponerCobertura) {
			s.AutenticacionRef = ""
		},
		"sesión": func(s *SolicitudProponerCobertura) { s.SesionRef = "" },
		"perfil": func(s *SolicitudProponerCobertura) { s.PerfilRef = "" },
		"organización": func(s *SolicitudProponerCobertura) {
			s.OrganizacionRef = ""
		},
		"expediente": func(s *SolicitudProponerCobertura) {
			s.ExpedienteRef = ""
		},
		"versión cero": func(s *SolicitudProponerCobertura) {
			s.VersionEsperada = 0
		},
		"versión 2^53": func(s *SolicitudProponerCobertura) {
			s.VersionEsperada = coberturaMaximoEnteroPresentacion()
		},
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			antes := base.contextos.total()
			solicitud := base.solicitud
			alterar(&solicitud)
			_, err := base.servicio.Proponer(context.Background(), solicitud)
			if !errors.Is(err, ErrSolicitudProponerCoberturaInvalida) ||
				base.contextos.total() != antes {
				t.Fatalf("entrada inválida alcanzó autoridades: %v", err)
			}
		})
	}
}

func TestProponerCoberturaSaneaErroresPrivadosYPriorizaCancelacion(
	t *testing.T,
) {
	t.Run("error privado de contexto", func(t *testing.T) {
		escenario := nuevoEscenarioPresentacionCobertura(
			t,
			viasPresentacionCoberturaPrueba(1),
		)
		privado := errors.New("ldap privado: persona 12345678Z")
		escenario.contextos.err = privado
		_, err := escenario.servicio.Proponer(
			context.Background(),
			escenario.solicitud,
		)
		if !errors.Is(
			err,
			ErrPresentacionPropuestaCoberturaNoDisponible,
		) || errors.Is(err, privado) ||
			strings.Contains(err.Error(), "12345678Z") {
			t.Fatalf("se filtró error privado: %v", err)
		}
	})

	t.Run("cancelación durante lectura", func(t *testing.T) {
		escenario := nuevoEscenarioPresentacionCobertura(
			t,
			viasPresentacionCoberturaPrueba(1),
		)
		ctx, cancelar := context.WithCancel(context.Background())
		escenario.analisis.cancelar = cancelar
		_, err := escenario.servicio.Proponer(ctx, escenario.solicitud)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("la cancelación no prevaleció: %v", err)
		}
		if escenario.gobierno.total() != 0 ||
			escenario.global.generador.llamadas() != 0 {
			t.Fatal("se continuó después de cancelar la lectura")
		}
	})
}

func TestProponerCoberturaEntregaCopiasYRedactaSolicitud(t *testing.T) {
	escenario := nuevoEscenarioPresentacionCobertura(
		t,
		viasPresentacionCoberturaPrueba(2),
	)
	primera, err := escenario.servicio.Proponer(
		context.Background(),
		escenario.solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	primera.Evaluaciones[0].ViaClave = "via_mutada_por_canal"
	primera.Evaluaciones[0].Conflictos = append(
		primera.Evaluaciones[0].Conflictos,
		"conflicto_mutado_por_canal",
	)
	segunda, err := escenario.servicio.Proponer(
		context.Background(),
		escenario.solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	if segunda.Evaluaciones[0].ViaClave != "via_global_01" ||
		len(segunda.Evaluaciones[0].Conflictos) != 0 {
		t.Fatal("la vista compartió memoria mutable con el canal")
	}
	texto := fmt.Sprintf(
		"%v|%+v|%#v",
		escenario.solicitud,
		escenario.solicitud,
		escenario.solicitud,
	)
	if strings.Contains(texto, escenario.solicitud.AutenticacionRef) ||
		strings.Contains(texto, escenario.solicitud.SesionRef) ||
		texto != redaccionSolicitudProponerCobertura+"|"+
			redaccionSolicitudProponerCobertura+"|"+
			redaccionSolicitudProponerCobertura {
		t.Fatalf("la solicitud no quedó redactada: %s", texto)
	}
	valorLog := escenario.solicitud.LogValue()
	if valorLog.Kind() != slog.KindString ||
		valorLog.String() != redaccionSolicitudProponerCobertura {
		t.Fatal("LogValue expuso la solicitud")
	}
}

func TestProponerCoberturaEsConcurrenteYDeterminista(t *testing.T) {
	escenario := nuevoEscenarioPresentacionCobertura(
		t,
		viasPresentacionCoberturaPrueba(3),
	)
	const total = 16
	const comprobacionesPorPropuesta = 3
	llegadas := make(chan struct{}, total*comprobacionesPorPropuesta)
	continuar := make(chan struct{})
	escenario.global.antes = func(
		ctx context.Context,
		_ ports.SolicitudConsultarCobertura,
	) error {
		llegadas <- struct{}{}
		select {
		case <-continuar:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	resultados := make(chan PresentacionPropuestaCobertura, total)
	errores := make(chan error, total)
	var grupo sync.WaitGroup
	for indice := 0; indice < total; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			resultado, err := escenario.servicio.Proponer(
				context.Background(),
				escenario.solicitud,
			)
			resultados <- resultado
			errores <- err
		}()
	}
	for indice := 0; indice < total*comprobacionesPorPropuesta; indice++ {
		<-llegadas
	}
	close(continuar)
	grupo.Wait()
	close(resultados)
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatalf("presentación concurrente falló: %v", err)
		}
	}
	var identidad domain.IdentidadSemanticaPropuestaDecisionCobertura
	for resultado := range resultados {
		if identidad == (domain.IdentidadSemanticaPropuestaDecisionCobertura{}) {
			identidad = resultado.IdentidadSemantica
			continue
		}
		if !identidad.CoincideExactamente(resultado.IdentidadSemantica) {
			t.Fatal("los metadatos transitorios contaminaron la semántica")
		}
	}
	if escenario.accesos.total() != total*2 {
		t.Fatal("alguna revelación evitó la doble autorización")
	}
	exigirCeroConsumoPreparacionGlobal(t, escenario.global)
}

func TestServicioPresentacionCoberturaRechazaNulosTipados(t *testing.T) {
	escenario := nuevoEscenarioPresentacionCobertura(
		t,
		viasPresentacionCoberturaPrueba(1),
	)
	var acceso *autorizadorPresentacionPrueba
	if _, err := NuevoServicioPresentacionPropuestaCobertura(
		escenario.contextos,
		acceso,
		escenario.analisis,
		escenario.reloj,
		escenario.gobierno,
		escenario.global.preparador,
	); !errors.Is(
		err,
		ErrServicioPresentacionPropuestaCoberturaInvalido,
	) {
		t.Fatalf("se aceptó dependencia nula tipada: %v", err)
	}
	var servicio *ServicioPresentacionPropuestaCobertura
	if _, err := servicio.Proponer(
		context.Background(),
		escenario.solicitud,
	); !errors.Is(
		err,
		ErrServicioPresentacionPropuestaCoberturaInvalido,
	) {
		t.Fatalf("receptor nil no falló cerrado: %v", err)
	}
	if _, err := escenario.servicio.Proponer(
		nil,
		escenario.solicitud,
	); !errors.Is(
		err,
		ErrServicioPresentacionPropuestaCoberturaInvalido,
	) {
		t.Fatalf("contexto nil no falló cerrado: %v", err)
	}
}

func exigirServicioPresentacionSinPuertosMutantes(t *testing.T) {
	t.Helper()
	tipo := reflect.TypeOf(ServicioPresentacionPropuestaCobertura{})
	if tipo.NumField() != 6 {
		t.Fatalf("aparecieron dependencias no revisadas: %d", tipo.NumField())
	}
	for indice := 0; indice < tipo.NumField(); indice++ {
		nombre := strings.ToLower(tipo.Field(indice).Name)
		for _, prohibido := range []string{
			"reserva", "consumo", "transaccion", "vec", "registro",
			"escrit", "confirm",
		} {
			if strings.Contains(nombre, prohibido) {
				t.Fatalf("Proponer recibió puerto mutante %q", nombre)
			}
		}
	}
}

func exigirVistaPresentacionMinimizada(t *testing.T) {
	t.Helper()
	tipo := reflect.TypeOf(PresentacionPropuestaCobertura{})
	permitidos := map[string]bool{
		"Estado": true, "ViaRecomendada": true,
		"Evaluaciones": true, "IdentidadSemantica": true,
	}
	if tipo.NumField() != len(permitidos) {
		t.Fatalf("la vista amplió su superficie: %d campos", tipo.NumField())
	}
	for indice := 0; indice < tipo.NumField(); indice++ {
		if !permitidos[tipo.Field(indice).Name] {
			t.Fatalf("la vista filtró material no autorizado: %s", tipo.Field(indice).Name)
		}
	}
}

func exigirSolicitudPresentacionMinimizada(t *testing.T) {
	t.Helper()
	tipo := reflect.TypeOf(SolicitudProponerCobertura{})
	permitidos := map[string]bool{
		"AutenticacionRef": true,
		"SesionRef":        true,
		"PerfilRef":        true,
		"OrganizacionRef":  true,
		"ExpedienteRef":    true,
		"VersionEsperada":  true,
	}
	if tipo.NumField() != len(permitidos) {
		t.Fatalf("la solicitud amplió su superficie: %d campos", tipo.NumField())
	}
	for indice := 0; indice < tipo.NumField(); indice++ {
		campo := tipo.Field(indice)
		if !permitidos[campo.Name] ||
			strings.Contains(strings.ToLower(campo.Name), "cookie") {
			t.Fatalf("el canal incorporó autoridad libre: %s", campo.Name)
		}
	}
}

func coberturaMaximoEnteroPresentacion() uint64 {
	return 1 << 53
}

package seguridad

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	materialAmbitoAsignacionV1Dorado = `{"esquema":"vec.contratacion-temporal.asignacion.ambito.v1","clave_idempotencia":"12345678-1234-4abc-8def-1234567890ab","organizacion_ref":"organizacion:dipgra:asignacion-001","actor_ref":"persona:tecnica:asignacion-001","perfil_ref":"perfil:tecnico:asignacion-001"}`
	selloAmbitoAsignacionV1Dorado    = "hmac-sha256:vec.contratacion-temporal.asignacion.ambito/v1:cf86a71bf12954a1fff4ee05623a6eb368e7680fdaf45772171a49c5f621c8e7"

	materialPeticionAsignacionV1Dorado = `{"esquema":"vec.contratacion-temporal.asignacion.peticion.v1","operacion":"reasignar","organizacion_ref":"organizacion:dipgra:asignacion-001","expediente_ref":"expediente:contratacion:asignacion-001","version_expediente":7,"actor_ref":"persona:tecnica:asignacion-001","perfil_ref":"perfil:tecnico:asignacion-001","unidad_ref":"unidad:seleccion:asignacion-001","responsable_ref":"persona:responsable:asignacion-001","motivo_reasignacion_clave":"necesidad_servicio","observaciones":"Cambio motivado por necesidad del servicio."}`
	selloPeticionAsignacionV1Dorado    = "hmac-sha256:vec.contratacion-temporal.asignacion.peticion/v1:e461c2ed2b6a8d124e6c1ff175d9a4dc68901ccc95612088a5ef4dc7bd7f7174"
)

type selladorAsignacionPrueba struct {
	mu                  sync.Mutex
	clave               []byte
	referencia          string
	referenciaRespuesta string
	err                 error
	cancelarTrasSellar  func()
	llamadas            int
	material            []byte
	materialPrestado    []byte
}

func (s *selladorAsignacionPrueba) SellarDatos(
	ctx context.Context,
	material []byte,
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.llamadas++
	s.material = append([]byte(nil), material...)
	s.materialPrestado = material
	if s.err != nil {
		return "", s.err
	}
	referencia := s.referencia
	if s.referenciaRespuesta != "" {
		referencia = s.referenciaRespuesta
	}
	mac := hmac.New(sha256.New, s.clave)
	_, _ = mac.Write(material)
	sello := "hmac-sha256:" + referencia + ":" +
		hex.EncodeToString(mac.Sum(nil))
	if s.cancelarTrasSellar != nil {
		s.cancelarTrasSellar()
	}
	return sello, nil
}

func (s *selladorAsignacionPrueba) estado() (int, []byte, []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.llamadas,
		append([]byte(nil), s.material...),
		append([]byte(nil), s.materialPrestado...)
}

type escenarioSellosAsignacion struct {
	autoridad *AutoridadSellosAsignacionHMAC
	ambitos   []*selladorAsignacionPrueba
	huellas   []*selladorAsignacionPrueba
}

func nuevaAutoridadSellosAsignacionPrueba(
	t *testing.T,
	generaciones []uint32,
) escenarioSellosAsignacion {
	t.Helper()
	if len(generaciones) == 0 {
		t.Fatal("faltan generaciones de prueba")
	}
	ambitos := make([]ConfiguracionSelladorHMAC, 0, len(generaciones))
	huellas := make([]ConfiguracionSelladorHMAC, 0, len(generaciones))
	selladoresAmbito := make([]*selladorAsignacionPrueba, 0, len(generaciones))
	selladoresHuella := make([]*selladorAsignacionPrueba, 0, len(generaciones))
	for _, generacion := range generaciones {
		configuracionAmbito, selladorAmbito := configuracionAsignacionPrueba(
			t,
			ports.DominioAmbitoIdempotenciaAsignacion,
			generacion,
		)
		configuracionHuella, selladorHuella := configuracionAsignacionPrueba(
			t,
			ports.DominioHuellaPeticionAsignacion,
			generacion,
		)
		ambitos = append(ambitos, configuracionAmbito)
		huellas = append(huellas, configuracionHuella)
		selladoresAmbito = append(selladoresAmbito, selladorAmbito)
		selladoresHuella = append(selladoresHuella, selladorHuella)
	}
	autoridad, err := NuevaAutoridadSellosAsignacionHMAC(
		ambitos[0],
		ambitos[1:],
		huellas[0],
		huellas[1:],
	)
	if err != nil {
		t.Fatal(err)
	}
	return escenarioSellosAsignacion{
		autoridad: autoridad,
		ambitos:   selladoresAmbito,
		huellas:   selladoresHuella,
	}
}

func configuracionAsignacionPrueba(
	t *testing.T,
	dominio string,
	generacion uint32,
) (ConfiguracionSelladorHMAC, *selladorAsignacionPrueba) {
	t.Helper()
	referencia := dominio + "/v" + strconv.FormatUint(uint64(generacion), 10)
	sellador := &selladorAsignacionPrueba{
		clave:      []byte("material-hmac-sintetico-" + referencia),
		referencia: referencia,
	}
	configuracion, err := NuevaConfiguracionSelladorHMAC(referencia, sellador)
	if err != nil {
		t.Fatal(err)
	}
	return configuracion, sellador
}

func solicitudAmbitoAsignacionPrueba() ports.SolicitudSellarAmbitoIdempotencia {
	return ports.SolicitudSellarAmbitoIdempotencia{
		ClaveIdempotencia: "12345678-1234-4abc-8def-1234567890ab",
		OrganizacionRef:   "organizacion:dipgra:asignacion-001",
		ActorRef:          "persona:tecnica:asignacion-001",
		PerfilRef:         "perfil:tecnico:asignacion-001",
	}
}

func materialReasignacionHMACPrueba() ports.MaterialHuellaAsignacion {
	return ports.MaterialHuellaAsignacion{
		Operacion:               ports.OperacionRegistrarReasignacion,
		OrganizacionRef:         "organizacion:dipgra:asignacion-001",
		ExpedienteRef:           "expediente:contratacion:asignacion-001",
		VersionExpediente:       7,
		ActorRef:                "persona:tecnica:asignacion-001",
		PerfilRef:               "perfil:tecnico:asignacion-001",
		UnidadRef:               "unidad:seleccion:asignacion-001",
		ResponsableRef:          "persona:responsable:asignacion-001",
		MotivoReasignacionClave: "necesidad_servicio",
		Observaciones:           "Cambio motivado por necesidad del servicio.",
	}
}

func TestAutoridadSellosAsignacionHMACFijaABIV1ConVectoresDorados(
	t *testing.T,
) {
	if esquemaAmbitoAsignacionV1 == ports.DominioAmbitoIdempotenciaAsignacion ||
		esquemaAmbitoAsignacionV1 == ports.DominioAmbitoIdempotenciaAsignacion+"/v1" ||
		esquemaPeticionAsignacionV1 == ports.DominioHuellaPeticionAsignacion ||
		esquemaPeticionAsignacionV1 == ports.DominioHuellaPeticionAsignacion+"/v1" {
		t.Fatal("el esquema ABI se confundió con dominio o generación de clave")
	}
	escenario := nuevaAutoridadSellosAsignacionPrueba(t, []uint32{1})
	ambitos, err := escenario.autoridad.SellarAmbitoAsignacion(
		context.Background(),
		solicitudAmbitoAsignacionPrueba(),
	)
	if err != nil {
		t.Fatal(err)
	}
	huellas, err := escenario.autoridad.DerivarHuellaAsignacion(
		context.Background(),
		materialReasignacionHMACPrueba(),
	)
	if err != nil {
		t.Fatal(err)
	}
	datosAmbito, err := ambitos.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datosHuella, err := huellas.Datos()
	if err != nil {
		t.Fatal(err)
	}
	_, materialAmbito, _ := escenario.ambitos[0].estado()
	_, materialHuella, _ := escenario.huellas[0].estado()
	if !slices.Equal(materialAmbito, []byte(materialAmbitoAsignacionV1Dorado)) ||
		datosAmbito.Activo.Valor != selloAmbitoAsignacionV1Dorado {
		t.Fatalf(
			"vector de ámbito V1 alterado sin elevar esquema: material=%q sello=%q",
			materialAmbito,
			datosAmbito.Activo.Valor,
		)
	}
	if !slices.Equal(materialHuella, []byte(materialPeticionAsignacionV1Dorado)) ||
		datosHuella.Activo.Valor != selloPeticionAsignacionV1Dorado {
		t.Fatalf(
			"vector de petición V1 alterado sin elevar esquema: material=%q sello=%q",
			materialHuella,
			datosHuella.Activo.Valor,
		)
	}
}

func TestAutoridadSellosAsignacionHMACConservaRotacionOrdenYDominios(
	t *testing.T,
) {
	escenario := nuevaAutoridadSellosAsignacionPrueba(t, []uint32{3, 2, 1})
	solicitud := solicitudAmbitoAsignacionPrueba()
	material := materialReasignacionHMACPrueba()

	ambitos, err := escenario.autoridad.SellarAmbitoAsignacion(
		context.Background(),
		solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	huellas, err := escenario.autoridad.DerivarHuellaAsignacion(
		context.Background(),
		material,
	)
	if err != nil {
		t.Fatal(err)
	}
	generacionesAmbito, err := ambitos.Generaciones()
	if err != nil {
		t.Fatal(err)
	}
	generacionesHuella, err := huellas.Generaciones()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(generacionesAmbito, []uint32{3, 2, 1}) ||
		!slices.Equal(generacionesAmbito, generacionesHuella) ||
		ambitos.ValidarDominio(ports.DominioAmbitoIdempotenciaAsignacion) != nil ||
		huellas.ValidarDominio(ports.DominioHuellaPeticionAsignacion) != nil {
		t.Fatalf(
			"rotación o dominio divergente: %v / %v",
			generacionesAmbito,
			generacionesHuella,
		)
	}
	if _, _, err := ports.ParActivoColeccionesHMAC(
		ambitos,
		ports.DominioAmbitoIdempotenciaAsignacion,
		huellas,
		ports.DominioHuellaPeticionAsignacion,
	); err != nil {
		t.Fatalf("par generacional separado: %v", err)
	}

	datosAmbito, _ := ambitos.Datos()
	datosHuella, _ := huellas.Datos()
	resultado := datosAmbito.Activo.Valor + datosHuella.Activo.Valor
	for _, sensible := range []string{
		solicitud.ClaveIdempotencia,
		solicitud.OrganizacionRef,
		material.ExpedienteRef,
		material.ResponsableRef,
		material.Observaciones,
	} {
		if strings.Contains(resultado, sensible) {
			t.Fatalf("el resultado expuso material sensible %q", sensible)
		}
	}
	_, materialAmbito, prestadoAmbito := escenario.ambitos[0].estado()
	_, materialHuella, prestadoHuella := escenario.huellas[0].estado()
	if !strings.Contains(
		string(materialAmbito),
		`"esquema":"`+esquemaAmbitoAsignacionV1+`"`,
	) || strings.Contains(
		string(materialAmbito),
		esquemaPeticionAsignacionV1,
	) || !strings.Contains(
		string(materialHuella),
		`"esquema":"`+esquemaPeticionAsignacionV1+`"`,
	) || strings.Contains(
		string(materialHuella),
		esquemaAmbitoAsignacionV1,
	) {
		t.Fatalf("preimágenes de dominio mezcladas: %q / %q", materialAmbito, materialHuella)
	}
	if !contenidoBorrado(prestadoAmbito) || !contenidoBorrado(prestadoHuella) {
		t.Fatal("la preimagen sobrevivió fuera de la invocación HMAC")
	}

	_, _ = escenario.autoridad.SellarAmbitoAsignacion(context.Background(), solicitud)
	_, _ = escenario.autoridad.DerivarHuellaAsignacion(context.Background(), material)
	for _, sellador := range append(escenario.ambitos, escenario.huellas...) {
		llamadas, _, _ := sellador.estado()
		if llamadas != 2 {
			t.Fatalf("se detectó caché o generación omitida: %d llamadas", llamadas)
		}
	}
}

func TestAutoridadSellosAsignacionHMACDistingueAsignacionYReasignacion(
	t *testing.T,
) {
	escenario := nuevaAutoridadSellosAsignacionPrueba(t, []uint32{1})
	reasignacion := materialReasignacionHMACPrueba()
	asignacion := reasignacion
	asignacion.Operacion = ports.OperacionRegistrarAsignacion
	asignacion.MotivoReasignacionClave = ""
	asignacion.Observaciones = ""

	selloAsignacion := derivarHuellaActiva(t, escenario.autoridad, asignacion)
	selloReasignacion := derivarHuellaActiva(t, escenario.autoridad, reasignacion)
	if hmac.Equal([]byte(selloAsignacion), []byte(selloReasignacion)) {
		t.Fatal("asignación y reasignación colisionaron")
	}
}

func TestAutoridadSellosAsignacionHMACTodaCoordenadaCambiaLaHuella(
	t *testing.T,
) {
	escenario := nuevaAutoridadSellosAsignacionPrueba(t, []uint32{2, 1})
	base := materialReasignacionHMACPrueba()
	selloBase := derivarHuellaActiva(t, escenario.autoridad, base)
	casos := map[string]func(*ports.MaterialHuellaAsignacion){
		"operacion": func(m *ports.MaterialHuellaAsignacion) {
			m.Operacion = ports.OperacionRegistrarAsignacion
			m.MotivoReasignacionClave = ""
			m.Observaciones = ""
		},
		"organizacion": func(m *ports.MaterialHuellaAsignacion) {
			m.OrganizacionRef = "organizacion:dipgra:asignacion-002"
		},
		"expediente": func(m *ports.MaterialHuellaAsignacion) {
			m.ExpedienteRef = "expediente:contratacion:asignacion-002"
		},
		"version": func(m *ports.MaterialHuellaAsignacion) {
			m.VersionExpediente++
		},
		"actor": func(m *ports.MaterialHuellaAsignacion) {
			m.ActorRef = "persona:tecnica:asignacion-002"
		},
		"perfil": func(m *ports.MaterialHuellaAsignacion) {
			m.PerfilRef = "perfil:tecnico:asignacion-002"
		},
		"unidad": func(m *ports.MaterialHuellaAsignacion) {
			m.UnidadRef = "unidad:seleccion:asignacion-002"
		},
		"responsable": func(m *ports.MaterialHuellaAsignacion) {
			m.ResponsableRef = "persona:responsable:asignacion-002"
		},
		"motivo": func(m *ports.MaterialHuellaAsignacion) {
			m.MotivoReasignacionClave = domain.ClaveCatalogo("cambio_organizativo")
		},
		"observaciones": func(m *ports.MaterialHuellaAsignacion) {
			m.Observaciones = "Cambio motivado por reorganización del servicio."
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			alterado := base
			mutar(&alterado)
			sello := derivarHuellaActiva(t, escenario.autoridad, alterado)
			if hmac.Equal([]byte(selloBase), []byte(sello)) {
				t.Fatalf("la coordenada %s no alteró la huella", nombre)
			}
		})
	}
}

func TestAutoridadSellosAsignacionHMACFallaCerradaEnConfiguracionInvalida(
	t *testing.T,
) {
	ambitoV3, _ := configuracionAsignacionPrueba(
		t,
		ports.DominioAmbitoIdempotenciaAsignacion,
		3,
	)
	ambitoV2, _ := configuracionAsignacionPrueba(
		t,
		ports.DominioAmbitoIdempotenciaAsignacion,
		2,
	)
	huellaV3, _ := configuracionAsignacionPrueba(
		t,
		ports.DominioHuellaPeticionAsignacion,
		3,
	)
	huellaV1, _ := configuracionAsignacionPrueba(
		t,
		ports.DominioHuellaPeticionAsignacion,
		1,
	)
	if autoridad, err := NuevaAutoridadSellosAsignacionHMAC(
		ambitoV3,
		[]ConfiguracionSelladorHMAC{ambitoV2},
		huellaV3,
		[]ConfiguracionSelladorHMAC{huellaV1},
	); autoridad != nil || !errors.Is(err, ErrSelladoAltaNoDisponible) {
		t.Fatalf("historias divergentes aceptadas: %#v / %v", autoridad, err)
	}

	var selladorNulo *selladorAsignacionPrueba
	huellaNula := ConfiguracionSelladorHMAC{
		referenciaClave: ports.DominioHuellaPeticionAsignacion + "/v3",
		sellador:        selladorNulo,
	}
	if autoridad, err := NuevaAutoridadSellosAsignacionHMAC(
		ambitoV3,
		nil,
		huellaNula,
		nil,
	); autoridad != nil || !errors.Is(err, ErrSelladoAltaNoDisponible) {
		t.Fatalf("nil tipado aceptado: %#v / %v", autoridad, err)
	}

	cruzada, _ := configuracionAsignacionPrueba(
		t,
		"vec.contratacion-temporal.huella-peticion",
		3,
	)
	if autoridad, err := NuevaAutoridadSellosAsignacionHMAC(
		ambitoV3,
		nil,
		cruzada,
		nil,
	); autoridad != nil || !errors.Is(err, ErrSelladoAltaNoDisponible) {
		t.Fatalf("dominio ajeno aceptado: %#v / %v", autoridad, err)
	}
}

func TestAutoridadSellosAsignacionHMACRechazaSelloDeDominioIncorrecto(
	t *testing.T,
) {
	escenario := nuevaAutoridadSellosAsignacionPrueba(t, []uint32{1})
	escenario.ambitos[0].referenciaRespuesta =
		ports.DominioHuellaPeticionAsignacion + "/v1"
	if coleccion, err := escenario.autoridad.SellarAmbitoAsignacion(
		context.Background(),
		solicitudAmbitoAsignacionPrueba(),
	); coleccion != (ports.ColeccionSellosHMAC{}) ||
		!errors.Is(err, ErrSelladoAltaNoDisponible) {
		t.Fatalf("sello cruzado de ámbito aceptado: %#v / %v", coleccion, err)
	}

	escenario = nuevaAutoridadSellosAsignacionPrueba(t, []uint32{1})
	escenario.huellas[0].referenciaRespuesta =
		ports.DominioAmbitoIdempotenciaAsignacion + "/v1"
	if coleccion, err := escenario.autoridad.DerivarHuellaAsignacion(
		context.Background(),
		materialReasignacionHMACPrueba(),
	); coleccion != (ports.ColeccionSellosHMAC{}) ||
		!errors.Is(err, ErrSelladoAltaNoDisponible) {
		t.Fatalf("sello cruzado de huella aceptado: %#v / %v", coleccion, err)
	}
}

func TestAutoridadSellosAsignacionHMACCancelaYNuncaFiltraMaterial(
	t *testing.T,
) {
	escenario := nuevaAutoridadSellosAsignacionPrueba(t, []uint32{1})
	solicitud := solicitudAmbitoAsignacionPrueba()
	material := materialReasignacionHMACPrueba()
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := escenario.autoridad.SellarAmbitoAsignacion(
		ctx,
		solicitud,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación de ámbito perdida: %v", err)
	}
	if _, err := escenario.autoridad.DerivarHuellaAsignacion(
		ctx,
		material,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación de huella perdida: %v", err)
	}

	var autoridadNula *AutoridadSellosAsignacionHMAC
	if _, err := autoridadNula.SellarAmbitoAsignacion(
		context.Background(),
		solicitud,
	); !errors.Is(err, ErrSelladoAltaNoDisponible) {
		t.Fatalf("receptor nil aceptado: %v", err)
	}
	if _, err := escenario.autoridad.DerivarHuellaAsignacion(
		nil,
		material,
	); !errors.Is(err, ErrSelladoAltaNoDisponible) {
		t.Fatalf("contexto nil aceptado: %v", err)
	}

	sensible := solicitud.ClaveIdempotencia
	escenario.ambitos[0].err = errors.New("fallo privado: " + sensible)
	_, err := escenario.autoridad.SellarAmbitoAsignacion(
		context.Background(),
		solicitud,
	)
	if !errors.Is(err, ErrSelladoAltaNoDisponible) ||
		strings.Contains(err.Error(), sensible) {
		t.Fatalf("el error filtró material sensible: %v", err)
	}
}

func TestAutoridadSellosAsignacionHMACCancelaDentroDelUltimoConector(
	t *testing.T,
) {
	t.Run("ambito", func(t *testing.T) {
		escenario := nuevaAutoridadSellosAsignacionPrueba(t, []uint32{1})
		ctx, cancelar := context.WithCancel(context.Background())
		escenario.ambitos[0].cancelarTrasSellar = cancelar
		coleccion, err := escenario.autoridad.SellarAmbitoAsignacion(
			ctx,
			solicitudAmbitoAsignacionPrueba(),
		)
		if coleccion != (ports.ColeccionSellosHMAC{}) ||
			!errors.Is(err, context.Canceled) {
			t.Fatalf("cancelación tardía de ámbito aceptada: %#v / %v", coleccion, err)
		}
		_, _, prestado := escenario.ambitos[0].estado()
		if !contenidoBorrado(prestado) {
			t.Fatal("la preimagen de ámbito no se borró tras cancelar")
		}
	})

	t.Run("peticion", func(t *testing.T) {
		escenario := nuevaAutoridadSellosAsignacionPrueba(t, []uint32{1})
		ctx, cancelar := context.WithCancel(context.Background())
		escenario.huellas[0].cancelarTrasSellar = cancelar
		coleccion, err := escenario.autoridad.DerivarHuellaAsignacion(
			ctx,
			materialReasignacionHMACPrueba(),
		)
		if coleccion != (ports.ColeccionSellosHMAC{}) ||
			!errors.Is(err, context.Canceled) {
			t.Fatalf("cancelación tardía de petición aceptada: %#v / %v", coleccion, err)
		}
		_, _, prestado := escenario.huellas[0].estado()
		if !contenidoBorrado(prestado) {
			t.Fatal("la preimagen de petición no se borró tras cancelar")
		}
	})
}

func TestAutoridadSellosAsignacionHMACEsConcurrenteSinEstadoMutable(
	t *testing.T,
) {
	escenario := nuevaAutoridadSellosAsignacionPrueba(t, []uint32{3, 2, 1})
	solicitud := solicitudAmbitoAsignacionPrueba()
	material := materialReasignacionHMACPrueba()
	ambitoEsperado := sellarAmbitoActivo(t, escenario.autoridad, solicitud)
	huellaEsperada := derivarHuellaActiva(t, escenario.autoridad, material)

	const concurrentes = 32
	errores := make(chan error, concurrentes*2)
	var grupo sync.WaitGroup
	for indice := 0; indice < concurrentes; indice++ {
		grupo.Add(2)
		go func() {
			defer grupo.Done()
			coleccion, err := escenario.autoridad.SellarAmbitoAsignacion(
				context.Background(),
				solicitud,
			)
			if err != nil {
				errores <- err
				return
			}
			datos, err := coleccion.Datos()
			if err != nil || datos.Activo.Valor != ambitoEsperado {
				errores <- fmt.Errorf("ámbito concurrente divergente: %v", err)
			}
		}()
		go func() {
			defer grupo.Done()
			coleccion, err := escenario.autoridad.DerivarHuellaAsignacion(
				context.Background(),
				material,
			)
			if err != nil {
				errores <- err
				return
			}
			datos, err := coleccion.Datos()
			if err != nil || datos.Activo.Valor != huellaEsperada {
				errores <- fmt.Errorf("huella concurrente divergente: %v", err)
			}
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		t.Error(err)
	}
	for _, sellador := range append(escenario.ambitos, escenario.huellas...) {
		llamadas, _, prestado := sellador.estado()
		if llamadas != concurrentes+1 || !contenidoBorrado(prestado) {
			t.Fatalf("estado concurrente inesperado: llamadas=%d", llamadas)
		}
	}
}

func derivarHuellaActiva(
	t *testing.T,
	autoridad *AutoridadSellosAsignacionHMAC,
	material ports.MaterialHuellaAsignacion,
) string {
	t.Helper()
	coleccion, err := autoridad.DerivarHuellaAsignacion(
		context.Background(),
		material,
	)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := coleccion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	return datos.Activo.Valor
}

func sellarAmbitoActivo(
	t *testing.T,
	autoridad *AutoridadSellosAsignacionHMAC,
	solicitud ports.SolicitudSellarAmbitoIdempotencia,
) string {
	t.Helper()
	coleccion, err := autoridad.SellarAmbitoAsignacion(
		context.Background(),
		solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := coleccion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	return datos.Activo.Valor
}

func contenidoBorrado(contenido []byte) bool {
	if len(contenido) == 0 {
		return false
	}
	for _, valor := range contenido {
		if valor != 0 {
			return false
		}
	}
	return true
}

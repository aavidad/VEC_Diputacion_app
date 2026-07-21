package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestServicioContextoActorProductivoResuelveYRegistraEnUnaLlamadaExacta(t *testing.T) {
	solicitadoEn := instanteServicioContextoActorPrueba()
	resueltoEnDB := solicitadoEn.Add(2 * time.Millisecond)
	comprobadoEn := resueltoEnDB.Add(time.Millisecond)
	solicitud := solicitudServicioContextoActorPrueba()
	generador := nuevoGeneradorOperacionContextoActorV2Prueba()
	resolutor := &resolutorRegistroContextoActorV2Prueba{
		resultado: confirmacionRegistroContextoActorV2Prueba(
			t, contextoActorServicioPrueba(t, resueltoEnDB, solicitud), generador.ref,
		),
	}
	servicio, err := NuevoServicioContextoActorProductivoV2(
		resolutor, generador,
		&relojSecuenciaContextoActorPrueba{instantes: []time.Time{solicitadoEn, comprobadoEn}},
	)
	if err != nil {
		t.Fatalf("crear servicio productivo: %v", err)
	}

	resultado, err := servicio.ResolverRegistrado(context.Background(), solicitud)
	if err != nil || resultado.Contexto.Validar() != nil {
		t.Fatalf("resolver y registrar: resultado=%#v error=%v", resultado, err)
	}
	llamadas, recibida := resolutor.observacion()
	if llamadas != 1 || recibida.Contexto != solicitud ||
		recibida.OperacionRef != generador.ref || !recibida.SolicitadoEn.Equal(solicitadoEn) ||
		resultado.ValidarParaProductiva(recibida) != nil ||
		resultado.RegistroContextoRef == "" || len(resultado.RepresentacionCanonica) == 0 ||
		len(resultado.ManifiestoProcedenciaCanonico) == 0 ||
		resultado.AutoridadEfectiva != domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1 ||
		resultado.Contexto.Instantanea.CuentaRef != solicitud.Cuenta.CuentaRef ||
		resultado.Contexto.PerfilActivoRef != solicitud.PerfilActivoRef ||
		resultado.Contexto.Principal.AuthMethod != solicitud.Cuenta.Metodo ||
		resultado.Contexto.Principal.AuthAssurance != solicitud.Cuenta.Garantia ||
		!resultado.Contexto.ResueltoEn.Equal(resueltoEnDB) {
		t.Fatalf("contrato productivo no exacto: llamadas=%d solicitud=%#v resultado=%#v",
			llamadas, recibida, resultado)
	}
	extraido, err := servicio.Resolver(context.Background(), solicitud)
	if extraido.Validar() == nil ||
		!errors.Is(err, ports.ErrResolutorRegistroContextoActorNoDisponible) {
		t.Fatalf("la API heredada perdio el recibo productivo: actor=%#v err=%v", extraido, err)
	}
}

func TestServicioContextoActorProductivoRechazaAutoridadNoAutoritativa(t *testing.T) {
	instante := instanteServicioContextoActorPrueba()
	solicitud := solicitudServicioContextoActorPrueba()
	generador := nuevoGeneradorOperacionContextoActorV2Prueba()
	confirmacion := confirmacionRegistroContextoActorV2Prueba(
		t, contextoActorServicioPrueba(t, instante, solicitud), generador.ref,
	)
	confirmacion.ManifiestoProcedenciaCanonico = []byte(strings.ReplaceAll(
		string(confirmacion.ManifiestoProcedenciaCanonico),
		string(domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1),
		string(domain.AutoridadProcedenciaContextoActorNoAutoritativaV1),
	))
	huella, err := domain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(
		confirmacion.ManifiestoProcedenciaCanonico,
	)
	if err != nil {
		t.Fatalf("fixture no autoritativa invalida: %v", err)
	}
	confirmacion.ManifiestoProcedenciaHuellaSHA256 = huella
	confirmacion.AutoridadEfectiva = domain.AutoridadProcedenciaContextoActorNoAutoritativaV1
	solicitudRegistro := ports.SolicitudResolucionRegistroContextoActorV2{
		OperacionRef: generador.ref, Contexto: solicitud, SolicitadoEn: instante,
	}
	if confirmacion.ValidarPara(solicitudRegistro) != nil ||
		confirmacion.ValidarParaProductiva(solicitudRegistro) == nil {
		t.Fatal("el contrato no distinguio evidencia estructural de autoridad productiva")
	}
	resolutor := &resolutorRegistroContextoActorV2Prueba{resultado: confirmacion}
	servicio, err := NuevoServicioContextoActorProductivoV2(
		resolutor, generador,
		&relojSecuenciaContextoActorPrueba{instantes: []time.Time{instante, instante}},
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := servicio.ResolverRegistrado(context.Background(), solicitud)
	if resultado.Contexto.Validar() == nil || !errors.Is(err, domain.ErrContextoActorNoResuelto) {
		t.Fatalf("servicio productivo entrego no_autoritativa: %#v %v", resultado, err)
	}
}

func TestServicioContextoActorProductivoAcotaTiempoDBEntreExtremosLocales(t *testing.T) {
	solicitadoEn := instanteServicioContextoActorPrueba()
	solicitud := solicitudServicioContextoActorPrueba()
	casos := []struct {
		nombre       string
		resueltoEnDB time.Time
		comprobadoEn time.Time
	}{
		{"DB anterior a solicitud", solicitadoEn.Add(-time.Microsecond), solicitadoEn},
		{"DB posterior a comprobacion", solicitadoEn.Add(2 * time.Millisecond), solicitadoEn.Add(time.Millisecond)},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			generador := nuevoGeneradorOperacionContextoActorV2Prueba()
			actor := contextoActorServicioPrueba(t, caso.resueltoEnDB, solicitud)
			resolutor := &resolutorRegistroContextoActorV2Prueba{resultado: confirmacionRegistroContextoActorV2Prueba(
				t, actor, generador.ref,
			)}
			reloj := &relojSecuenciaContextoActorPrueba{instantes: []time.Time{solicitadoEn, caso.comprobadoEn}}
			servicio, err := NuevoServicioContextoActorProductivoV2(resolutor, generador, reloj)
			if err != nil {
				t.Fatalf("crear servicio: %v", err)
			}
			resultado, err := servicio.ResolverRegistrado(context.Background(), solicitud)
			if resultado.Contexto.Validar() == nil || !errors.Is(err, domain.ErrContextoActorNoResuelto) {
				t.Fatalf("tiempo DB fuera de extremos aceptado: resultado=%#v error=%v", resultado, err)
			}
		})
	}
}

func TestServicioContextoActorProductivoRechazaCapacidadNoExactaSinSegundoIntento(t *testing.T) {
	instante := instanteServicioContextoActorPrueba()
	solicitud := solicitudServicioContextoActorPrueba()
	base := contextoActorServicioPrueba(t, instante, solicitud)
	casos := []struct {
		nombre string
		mutar  func(contextoActorPruebaMutador) domain.ContextoActor
	}{
		{"cuenta distinta", func(m contextoActorPruebaMutador) domain.ContextoActor {
			m.solicitud.Cuenta.CuentaRef = referenciaServicioContextoActorPrueba("cta_", "x")
			m.instantanea.CuentaRef = m.solicitud.Cuenta.CuentaRef
			return m.crear(t)
		}},
		{"perfil distinto", func(m contextoActorPruebaMutador) domain.ContextoActor {
			m.solicitud.PerfilActivoRef = referenciaServicioContextoActorPrueba("prf_", "x")
			m.instantanea.PerfilActivoRef = m.solicitud.PerfilActivoRef
			return m.crear(t)
		}},
		{"metodo distinto", func(m contextoActorPruebaMutador) domain.ContextoActor {
			m.solicitud.Cuenta.Metodo = domain.AuthMethodSSO
			return m.crear(t)
		}},
		{"garantia distinta", func(m contextoActorPruebaMutador) domain.ContextoActor {
			m.solicitud.Cuenta.Garantia = domain.AuthAssuranceSubstantial
			return m.crear(t)
		}},
		{"capacidad invalida", func(m contextoActorPruebaMutador) domain.ContextoActor {
			resultado := m.crear(t)
			resultado.Principal.Roles = []string{"administrador"}
			return resultado
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			mutador := contextoActorPruebaMutador{
				solicitud: solicitud,
				instantanea: func() domain.InstantaneaContextoActor {
					copia, err := base.Instantanea.ClonarCanonica()
					if err != nil {
						t.Fatalf("clonar instantanea: %v", err)
					}
					return copia
				}(),
				instante: instante,
			}
			generador := nuevoGeneradorOperacionContextoActorV2Prueba()
			actor := caso.mutar(mutador)
			confirmacion := confirmacionRegistroContextoActorV2Prueba(t, base, generador.ref)
			if actor.Validar() == nil {
				confirmacion = confirmacionRegistroContextoActorV2Prueba(t, actor, generador.ref)
			} else {
				confirmacion.Contexto = actor
			}
			resolutor := &resolutorRegistroContextoActorV2Prueba{resultado: confirmacion}
			servicio, err := NuevoServicioContextoActorProductivoV2(
				resolutor, generador, &relojContextoActorPrueba{ahora: instante},
			)
			if err != nil {
				t.Fatalf("crear servicio: %v", err)
			}
			resultado, err := servicio.ResolverRegistrado(context.Background(), solicitud)
			llamadas, _ := resolutor.observacion()
			if !errors.Is(err, domain.ErrContextoActorNoResuelto) || resultado.Contexto.Validar() == nil || llamadas != 1 {
				t.Fatalf("capacidad no exacta aceptada o reintentada: resultado=%#v error=%v llamadas=%d",
					resultado, err, llamadas)
			}
		})
	}
}

func TestServicioContextoActorProductivoRechazaReciboDurableAlterado(t *testing.T) {
	instante := instanteServicioContextoActorPrueba()
	solicitud := solicitudServicioContextoActorPrueba()
	generadorBase := nuevoGeneradorOperacionContextoActorV2Prueba()
	base := confirmacionRegistroContextoActorV2Prueba(
		t, contextoActorServicioPrueba(t, instante, solicitud), generadorBase.ref,
	)
	casos := []struct {
		nombre string
		mutar  func(*ports.ConfirmacionRegistroContextoActorV2)
	}{
		{"operacion ajena", func(c *ports.ConfirmacionRegistroContextoActorV2) {
			c.OperacionRef = referenciaServicioContextoActorPrueba("oca_", "x")
		}},
		{"registro no canonico", func(c *ports.ConfirmacionRegistroContextoActorV2) {
			c.RegistroContextoRef = "rca_corto"
		}},
		{"preimagen alterada", func(c *ports.ConfirmacionRegistroContextoActorV2) {
			c.RepresentacionCanonica[0] = '['
		}},
		{"huella alterada", func(c *ports.ConfirmacionRegistroContextoActorV2) {
			c.HuellaSHA256 = strings.Repeat("0", 64)
		}},
		{"tiempo DB no ligado", func(c *ports.ConfirmacionRegistroContextoActorV2) {
			c.ResueltoEnAutoritativo = c.ResueltoEnAutoritativo.Add(time.Microsecond)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			generador := nuevoGeneradorOperacionContextoActorV2Prueba()
			confirmacion := base
			confirmacion.RepresentacionCanonica = append([]byte(nil), base.RepresentacionCanonica...)
			caso.mutar(&confirmacion)
			resolutor := &resolutorRegistroContextoActorV2Prueba{resultado: confirmacion}
			servicio, err := NuevoServicioContextoActorProductivoV2(
				resolutor, generador, &relojContextoActorPrueba{ahora: instante},
			)
			if err != nil {
				t.Fatalf("crear servicio: %v", err)
			}
			resultado, err := servicio.ResolverRegistrado(context.Background(), solicitud)
			llamadas, _ := resolutor.observacion()
			if resultado.Contexto.Validar() == nil || llamadas != 1 ||
				!errors.Is(err, domain.ErrContextoActorNoResuelto) {
				t.Fatalf("recibo alterado aceptado: resultado=%#v llamadas=%d error=%v",
					resultado, llamadas, err)
			}
		})
	}
}

func TestServicioContextoActorProductivoRevalidaTrasEsperasAntesDeEntregar(t *testing.T) {
	instante := instanteServicioContextoActorPrueba()
	solicitud := solicitudServicioContextoActorPrueba()
	instantaneaBreve := instantaneaServicioContextoActorPrueba(instante, solicitud)
	instantaneaBreve.VigenteHasta = instante.Add(time.Second)
	for indice := range instantaneaBreve.Vinculos {
		instantaneaBreve.Vinculos[indice].VigenteHasta = instante.Add(time.Second)
	}
	actorBreve, err := domain.NuevoContextoActor(solicitud.Cuenta, instantaneaBreve, instante)
	if err != nil {
		t.Fatalf("crear capacidad de vigencia breve: %v", err)
	}
	actorVigente := contextoActorServicioPrueba(t, instante, solicitud)

	casos := []struct {
		nombre     string
		resultado  domain.ContextoActor
		comprobado time.Time
	}{
		{"caduca mientras espera bloqueos", actorBreve, instante.Add(time.Second)},
		{"respuesta pierde frescura", actorVigente, instante.Add(ports.VentanaMaximaFrescuraContextoActorV2 + time.Microsecond)},
		{"reloj retrocede", actorVigente, instante.Add(-time.Microsecond)},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			generador := nuevoGeneradorOperacionContextoActorV2Prueba()
			resolutor := &resolutorRegistroContextoActorV2Prueba{resultado: confirmacionRegistroContextoActorV2Prueba(
				t, caso.resultado, generador.ref,
			)}
			reloj := &relojSecuenciaContextoActorPrueba{instantes: []time.Time{instante, caso.comprobado}}
			servicio, err := NuevoServicioContextoActorProductivoV2(resolutor, generador, reloj)
			if err != nil {
				t.Fatalf("crear servicio: %v", err)
			}
			resultado, err := servicio.ResolverRegistrado(context.Background(), solicitud)
			llamadas, _ := resolutor.observacion()
			if !errors.Is(err, domain.ErrContextoActorNoResuelto) || resultado.Contexto.Validar() == nil ||
				llamadas != 1 || reloj.llamadas != 2 {
				t.Fatalf("se entrego capacidad no fresca: resultado=%#v error=%v resoluciones=%d relojes=%d",
					resultado, err, llamadas, reloj.llamadas)
			}
		})
	}
}

func TestServicioContextoActorProductivoSaneaFalloYNoUsaFuenteHeredada(t *testing.T) {
	const detalleSensible = "fila=vca_secreta dsn=postgres://secreto@interno"
	instante := instanteServicioContextoActorPrueba()
	resolutor := &resolutorRegistroContextoActorV2Prueba{error: errors.New(detalleSensible)}
	generador := nuevoGeneradorOperacionContextoActorV2Prueba()
	servicio, err := NuevoServicioContextoActorProductivoV2(
		resolutor, generador, &relojContextoActorPrueba{ahora: instante},
	)
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}

	resultado, err := servicio.ResolverRegistrado(context.Background(), solicitudServicioContextoActorPrueba())
	llamadas, _ := resolutor.observacion()
	if resultado.Contexto.Validar() == nil || llamadas != 1 ||
		!errors.Is(err, domain.ErrContextoActorNoResuelto) ||
		!errors.Is(err, ports.ErrResolutorRegistroContextoActorNoDisponible) ||
		strings.Contains(err.Error(), detalleSensible) {
		t.Fatalf("fallo productivo no cerrado: resultado=%#v llamadas=%d error=%v", resultado, llamadas, err)
	}

	// Una composicion imposible con ambas dependencias queda cerrada; nunca cae
	// a la fuente heredada si el resolutor productivo falla.
	fuente := &fuenteContextoActorPrueba{instantaneas: []domain.InstantaneaContextoActor{
		instantaneaServicioContextoActorPrueba(instante, solicitudServicioContextoActorPrueba()),
	}}
	servicio.fuente = fuente
	if _, err = servicio.ResolverRegistrado(context.Background(), solicitudServicioContextoActorPrueba()); !errors.Is(err, ports.ErrResolutorRegistroContextoActorNoDisponible) || fuente.numeroLlamadas() != 0 {
		t.Fatalf("se uso respaldo heredado: error=%v llamadas_fuente=%d", err, fuente.numeroLlamadas())
	}
}

func TestNuevoServicioContextoActorProductivoRechazaDependenciasNulasTipadas(t *testing.T) {
	var resolutorNulo *resolutorRegistroContextoActorV2Prueba
	var generadorNulo *generadorOperacionContextoActorV2Prueba
	var relojNulo *relojContextoActorPrueba
	validoResolutor := &resolutorRegistroContextoActorV2Prueba{}
	validoGenerador := nuevoGeneradorOperacionContextoActorV2Prueba()
	validoReloj := &relojContextoActorPrueba{ahora: instanteServicioContextoActorPrueba()}
	casos := []struct {
		nombre    string
		resolutor ports.ResolutorRegistroContextoActorV2
		generador ports.GeneradorOperacionContextoActorV2
		reloj     ports.Reloj
	}{
		{"resolutor nil", nil, validoGenerador, validoReloj},
		{"resolutor nil tipado", resolutorNulo, validoGenerador, validoReloj},
		{"generador nil", validoResolutor, nil, validoReloj},
		{"generador nil tipado", validoResolutor, generadorNulo, validoReloj},
		{"reloj nil", validoResolutor, validoGenerador, nil},
		{"reloj nil tipado", validoResolutor, validoGenerador, relojNulo},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			servicio, err := NuevoServicioContextoActorProductivoV2(
				caso.resolutor, caso.generador, caso.reloj,
			)
			if servicio != nil || !errors.Is(err, domain.ErrContextoActorInvalido) {
				t.Fatalf("dependencia nula aceptada: servicio=%#v error=%v", servicio, err)
			}
		})
	}
}

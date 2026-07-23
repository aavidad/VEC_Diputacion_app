package ports

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestCoberturaRechazaCadaCoordenadaAlteradaAunqueEsteResellada(
	t *testing.T,
) {
	casos := []struct {
		nombre  string
		alterar func(*DatosResultadoConsultaCobertura)
	}{
		{"petición", func(d *DatosResultadoConsultaCobertura) {
			d.PeticionRef = "peticion_cobertura_ajena_0123"
		}},
		{"huella petición", func(d *DatosResultadoConsultaCobertura) {
			d.HuellaPeticionSHA256 = strings.Repeat("c", 64)
		}},
		{"organización", func(d *DatosResultadoConsultaCobertura) {
			d.OrganizacionRef = "organizacion_ajena_0123456789"
		}},
		{"expediente", func(d *DatosResultadoConsultaCobertura) {
			d.ExpedienteRef = "expediente_ajeno_0123456789"
		}},
		{"versión expediente", func(d *DatosResultadoConsultaCobertura) {
			d.VersionExpediente++
		}},
		{"referencia catálogo", func(d *DatosResultadoConsultaCobertura) {
			d.Catalogo.Referencia = "catalogo_cobertura_ajeno_01"
		}},
		{"versión catálogo", func(d *DatosResultadoConsultaCobertura) {
			d.Catalogo.Version++
		}},
		{"huella catálogo", func(d *DatosResultadoConsultaCobertura) {
			d.Catalogo.HuellaSHA256 = strings.Repeat("b", 64)
		}},
		{"vía", func(d *DatosResultadoConsultaCobertura) {
			d.ViaClave = "otra_via_dinamica"
		}},
		{"procedencia", func(d *DatosResultadoConsultaCobertura) {
			d.ProcedenciaClave = "otra_procedencia"
		}},
		{"categoría", func(d *DatosResultadoConsultaCobertura) {
			d.CategoriaRef = "categoria_ajena_0123456789"
		}},
		{"periodo", func(d *DatosResultadoConsultaCobertura) {
			d.Periodo.Fin = d.Periodo.Fin.AddDate(0, 0, 1)
		}},
		{"comprobación", func(d *DatosResultadoConsultaCobertura) {
			d.Comprobacion.Clave = "otra_comprobacion"
		}},
		{"detalle libre", func(d *DatosResultadoConsultaCobertura) {
			d.Comprobacion.Detalle = "DNI 12345678Z"
		}},
		{"definición fuente", func(d *DatosResultadoConsultaCobertura) {
			d.DefinicionFuenteRef = "definicion_fuente_ajena_0123"
		}},
		{"evaluación anterior", func(d *DatosResultadoConsultaCobertura) {
			d.Comprobacion.EvaluadaEn = d.Comprobacion.EvaluadaEn.Add(-2 * time.Second)
		}},
		{"evaluación futura", func(d *DatosResultadoConsultaCobertura) {
			d.Comprobacion.EvaluadaEn = d.Comprobacion.EvaluadaEn.Add(2 * time.Second)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			entorno := nuevoEntornoCoberturaPrueba(t)
			entorno.fuente.consultar = func(
				_ context.Context,
				solicitud SolicitudConsultarCobertura,
			) (ResultadoConsultaCobertura, error) {
				datos := datosResultadoCoberturaPrueba(solicitud)
				caso.alterar(&datos)
				return resultadoCoberturaConDatosFirmadoPrueba(
					t,
					solicitud,
					datos,
				), nil
			}
			if _, err := entorno.consultar(context.Background()); !errors.Is(err, ErrResultadoFuenteCoberturaNoConfiable) {
				t.Fatalf("coordenada alterada aceptada: %v", err)
			}
		})
	}
}

func TestCoberturaRechazaFirmaYPreimagenAlteradas(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		alterar func(*ResultadoConsultaCobertura)
	}{
		{"sello", func(r *ResultadoConsultaCobertura) {
			ultimo := "0"
			if strings.HasSuffix(r.atestacion.SelloHMAC, "0") {
				ultimo = "1"
			}
			r.atestacion.SelloHMAC = r.atestacion.SelloHMAC[:len(
				r.atestacion.SelloHMAC,
			)-1] + ultimo
		}},
		{"preimagen", func(r *ResultadoConsultaCobertura) {
			r.preimagen.contenido[0] ^= 1
		}},
		{"metadatos", func(r *ResultadoConsultaCobertura) {
			r.atestacion.Metadatos.AutoridadRef = "autoridad_ajena_0123456789"
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			entorno := nuevoEntornoCoberturaPrueba(t)
			entorno.fuente.consultar = func(
				context.Context,
				SolicitudConsultarCobertura,
			) (ResultadoConsultaCobertura, error) {
				resultado := resultadoCoberturaFirmadoPrueba(
					t,
					entorno.solicitud,
				)
				caso.alterar(&resultado)
				return resultado, nil
			}
			if _, err := entorno.consultar(context.Background()); !errors.Is(err, ErrResultadoFuenteCoberturaNoConfiable) {
				t.Fatalf("alteración aceptada: %v", err)
			}
		})
	}
}

func TestCoberturaRechazaCredencialesManipuladas(t *testing.T) {
	casos := []struct {
		nombre  string
		alterar func(*presentadorAutoridadConfiguradoPrueba)
	}{
		{"raíz", func(p *presentadorAutoridadConfiguradoPrueba) {
			p.datos.RaizClaveID = "raiz_institucional_ajena_012345"
		}},
		{"rol", func(p *presentadorAutoridadConfiguradoPrueba) {
			p.datos.Rol = RolVerificadorCobertura
		}},
		{"audiencia", func(p *presentadorAutoridadConfiguradoPrueba) {
			p.datos.Audiencia = "audiencia_externa_no_admitida"
		}},
		{"organización", func(p *presentadorAutoridadConfiguradoPrueba) {
			p.datos.OrganizacionRef = "organizacion_ajena_0123456789"
		}},
		{"firma credencial", func(p *presentadorAutoridadConfiguradoPrueba) {
			p.alterar = func(c *CredencialAutoridadFuenteAnalisis) {
				c.firma[0] ^= 1
			}
		}},
		{"prueba de otro desafío", func(p *presentadorAutoridadConfiguradoPrueba) {
			p.reusarPrueba = make([]byte, 64)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			entorno := nuevoEntornoCoberturaPrueba(t)
			caso.alterar(&entorno.fuente.presentador)
			if _, err := entorno.consultar(context.Background()); !errors.Is(err, ErrResultadoFuenteCoberturaNoConfiable) {
				t.Fatalf("credencial manipulada aceptada: %v", err)
			}
		})
	}
}

func TestCoberturaNoAdmitePruebaDePosesionDeOtroDesafio(t *testing.T) {
	entorno := nuevoEntornoCoberturaPrueba(t)
	material, err := canonPeticionCobertura(entorno.solicitud)
	if err != nil {
		t.Fatal(err)
	}
	primero, err := nuevoDesafioAutoridadFuenteAnalisis(
		material,
		organizacionAutoridadPrueba,
		audienciaAutoridadPrueba,
		RolFuenteCobertura,
	)
	if err != nil {
		t.Fatal(err)
	}
	presentacion, err := presentacionAutoridadPrueba(
		RolFuenteCobertura,
		"fuente_cobertura_bolsa_012345",
		"backend_fuente_cobertura_012345",
		primero,
	)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := nuevoDesafioAutoridadFuenteAnalisis(
		material,
		organizacionAutoridadPrueba,
		audienciaAutoridadPrueba,
		RolFuenteCobertura,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entorno.confianza.verificarPresentacion(
		presentacion,
		segundo,
		RolFuenteCobertura,
		entorno.reloj.ahora,
	); err == nil {
		t.Fatal("una prueba de posesión se reutilizó en otro desafío")
	}
}

func TestCoberturaExigeTresLimitesDeConfianzaDistintos(t *testing.T) {
	entorno := nuevoEntornoCoberturaPrueba(t)
	entorno.verificador.presentador.datos.BackendRef =
		entorno.fuente.presentador.datos.BackendRef
	if _, err := entorno.consultar(context.Background()); !errors.Is(err, ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("wrappers del mismo backend aceptados: %v", err)
	}

	entorno = nuevoEntornoCoberturaPrueba(t)
	entorno.publicador.presentador.datos.ClavePruebaEd25519 =
		append(
			[]byte(nil),
			entorno.fuente.presentador.datos.ClavePruebaEd25519...,
		)
	if _, err := entorno.consultar(context.Background()); !errors.Is(err, ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("clave de posesión repetida aceptada: %v", err)
	}

	entorno = nuevoEntornoCoberturaPrueba(t)
	entorno.publicador.presentador.datos.AutoridadRef =
		entorno.fuente.presentador.datos.AutoridadRef
	if _, err := entorno.consultar(context.Background()); !errors.Is(err, ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("autoridad repetida aceptada: %v", err)
	}
}

func TestCoberturaRestauraCatalogoYPruebaPertenenciaExacta(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		alterar func(*DatosConfirmacionPublicacionCobertura)
	}{
		{"publicación adulterada", func(d *DatosConfirmacionPublicacionCobertura) {
			d.Publicacion.Vias[0].Clave = "via_adulterada"
		}},
		{"publicador suplantado", func(d *DatosConfirmacionPublicacionCobertura) {
			d.PublicadorRef = "publicador_suplantado_012345"
		}},
		{"catálogo expirado", func(d *DatosConfirmacionPublicacionCobertura) {
			d.Publicacion.Vigencia.Hasta = d.VerificadaEn
			catalogo, _ := domain.PublicarCatalogoViasCobertura(
				domain.BorradorCatalogoViasCobertura{
					Referencia:     d.Publicacion.Referencia,
					Version:        d.Publicacion.Version,
					PublicadoEn:    d.Publicacion.PublicadoEn,
					Vigencia:       d.Publicacion.Vigencia,
					ProcedenciaRef: d.Publicacion.ProcedenciaRef,
					Vias:           d.Publicacion.Vias,
				},
			)
			d.Publicacion = catalogo.Publicacion()
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			entorno := nuevoEntornoCoberturaPrueba(t)
			publicar := entorno.publicador.publicar
			entorno.publicador.publicar = func(
				ctx context.Context,
				solicitud SolicitudConsultarCobertura,
			) (ConfirmacionPublicacionCobertura, error) {
				confirmacion, err := publicar(ctx, solicitud)
				if err == nil {
					caso.alterar(confirmacion.datos)
				}
				return confirmacion, err
			}
			if _, err := entorno.consultar(context.Background()); !errors.Is(err, ErrResultadoFuenteCoberturaNoConfiable) {
				t.Fatalf("catálogo manipulado aceptado: %v", err)
			}
		})
	}
}

func TestCoberturaRechazaComprobacionAusenteDelCatalogo(t *testing.T) {
	entorno := nuevoEntornoCoberturaPrueba(t)
	entorno.solicitud.Comprobacion = domain.ComprobacionExigibleCobertura{
		Clave:       "comprobacion_no_publicada",
		Orden:       2,
		Obligatoria: true,
		Procedencia: domain.ProcedenciaComprobacionCobertura{
			Clave:               "fuente_no_publicada",
			DefinicionFuenteRef: "conector_no_publicado_012345",
		},
	}
	if entorno.solicitud.Validar() != nil {
		t.Fatal("el caso adversarial no es estructuralmente válido")
	}
	if _, err := entorno.consultar(context.Background()); !errors.Is(err, ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("comprobación no publicada aceptada: %v", err)
	}
}

func TestCoberturaRechazaConfirmacionDeVerificadorSuplantada(t *testing.T) {
	entorno := nuevoEntornoCoberturaPrueba(t)
	verificar := entorno.verificador.verificar
	entorno.verificador.verificar = func(
		ctx context.Context,
		solicitud SolicitudVerificarRespuestaCobertura,
	) (ConfirmacionRespuestaCobertura, error) {
		confirmacion, err := verificar(ctx, solicitud)
		if err == nil {
			confirmacion.datos.VerificadorRef =
				"verificador_suplantado_012345"
		}
		return confirmacion, err
	}
	if _, err := entorno.consultar(context.Background()); !errors.Is(err, ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("confirmación suplantada aceptada: %v", err)
	}
}

func TestCoberturaReplayConcurrenteEsUnicoYConflictoEsVisible(t *testing.T) {
	entorno := nuevoEntornoCoberturaPrueba(t)
	const paralelismo = 24
	var grupo sync.WaitGroup
	errores := make(chan error, paralelismo)
	for range paralelismo {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			_, err := entorno.consultar(context.Background())
			errores <- err
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatalf("replay concurrente falló: %v", err)
		}
	}
	if len(entorno.consumidor.registros) != 1 {
		t.Fatalf("más de un efecto durable: %d", len(entorno.consumidor.registros))
	}

	entorno.fuente.consultar = func(
		_ context.Context,
		solicitud SolicitudConsultarCobertura,
	) (ResultadoConsultaCobertura, error) {
		datos := datosResultadoCoberturaPrueba(solicitud)
		datos.Comprobacion.Resultado = domain.ComprobacionNegativa
		return resultadoCoberturaConDatosFirmadoPrueba(
			t,
			solicitud,
			datos,
		), nil
	}
	if _, err := entorno.consultar(context.Background()); !errors.Is(err, ErrRespuestaCoberturaYaConsumida) {
		t.Fatalf("conflicto durable no detectado: %v", err)
	}
}

func TestCoberturaExpiraEnLimiteExclusivoSinReconsumir(t *testing.T) {
	entorno := nuevoEntornoCoberturaPrueba(t)
	if _, err := entorno.consultar(context.Background()); err != nil {
		t.Fatal(err)
	}
	entorno.reloj.ahora = entorno.solicitud.SolicitadaEn.Add(5 * time.Second)
	if _, err := entorno.consultar(context.Background()); !errors.Is(err, ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("respuesta expirada aceptada: %v", err)
	}
	if len(entorno.consumidor.registros) != 1 {
		t.Fatal("una respuesta expirada alcanzó el consumidor")
	}
}

func TestCoberturaDaPrioridadACancelacionSobreFalloPrivado(t *testing.T) {
	entorno := nuevoEntornoCoberturaPrueba(t)
	ctx, cancelar := context.WithCancel(context.Background())
	privado := &errorPrivadoCobertura{detalle: "backend privado"}
	entorno.fuente.consultar = func(
		context.Context,
		SolicitudConsultarCobertura,
	) (ResultadoConsultaCobertura, error) {
		cancelar()
		return ResultadoConsultaCobertura{}, privado
	}
	_, err := entorno.consultar(ctx)
	if !errors.Is(err, context.Canceled) ||
		!errors.Is(err, ErrFuenteCoberturaNoDisponible) ||
		errors.Is(err, privado) {
		t.Fatalf("cancelación no prioritaria o filtrada: %v", err)
	}
}

func TestCoberturaLimitesYMinimizacion(t *testing.T) {
	entorno := nuevoEntornoCoberturaPrueba(t)
	enLimite := entorno.solicitud
	enLimite.VersionExpediente = maximoEnteroSeguroFuenteAnalisis
	enLimite.Periodo.Fin = enLimite.Periodo.Inicio.AddDate(100, 0, 0)
	if err := enLimite.Validar(); err != nil {
		t.Fatalf("se rechazó el límite positivo exacto: %v", err)
	}
	for _, modificar := range []func(*SolicitudConsultarCobertura){
		func(s *SolicitudConsultarCobertura) {
			s.VersionExpediente = maximoEnteroSeguroFuenteAnalisis + 1
		},
		func(s *SolicitudConsultarCobertura) {
			s.Periodo.Fin = s.Periodo.Inicio.AddDate(100, 0, 1)
		},
	} {
		solicitud := entorno.solicitud
		modificar(&solicitud)
		if solicitud.Validar() == nil {
			t.Fatal("límite numérico o temporal aceptado")
		}
	}
	for _, duracion := range []time.Duration{
		0,
		-1,
		TiempoMaximoFuenteCobertura + time.Nanosecond,
	} {
		if _, err := ConsultarCoberturaConFuente(
			context.Background(),
			entorno.fuente,
			entorno.verificador,
			entorno.publicador,
			entorno.consumidor,
			entorno.confianza,
			entorno.reloj,
			entorno.solicitud,
			duracion,
		); !errors.Is(err, ErrPeticionFuenteCoberturaInvalida) {
			t.Fatalf("timeout inválido aceptado: %v", err)
		}
	}
	metadatos := MetadatosAtestacionRespuestaCobertura{
		AutoridadRef: "fuente_cobertura_bolsa_012345",
		Generacion:   1,
		ReciboRef:    "recibo_limite_cobertura_012345",
		EmitidaEn:    entorno.solicitud.SolicitadaEn,
		ValidaHasta: entorno.solicitud.SolicitadaEn.Add(
			VigenciaMaximaRespuestaCobertura,
		),
	}
	if metadatos.Validar() != nil {
		t.Fatal("se rechazó la vida exacta de cinco segundos")
	}
	metadatos.ValidaHasta = metadatos.ValidaHasta.Add(time.Microsecond)
	if metadatos.Validar() == nil {
		t.Fatal("se aceptó una vida superior a cinco segundos")
	}
	tipo := reflect.TypeOf(SolicitudConsultarCobertura{})
	permitidos := map[string]struct{}{
		"PeticionRef":       {},
		"OrganizacionRef":   {},
		"ExpedienteRef":     {},
		"VersionExpediente": {},
		"Catalogo":          {},
		"ViaClave":          {},
		"Comprobacion":      {},
		"CategoriaRef":      {},
		"Periodo":           {},
		"SolicitadaEn":      {},
	}
	if tipo.NumField() != len(permitidos) {
		t.Fatalf("superficie de petición ampliada: %d", tipo.NumField())
	}
	for indice := range tipo.NumField() {
		if _, existe := permitidos[tipo.Field(indice).Name]; !existe {
			t.Fatalf("campo con posible PII: %s", tipo.Field(indice).Name)
		}
	}
}

package ports

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const claveRespuestaCoberturaPrueba = "clave-tcb-respuesta-cobertura"

type relojCoberturaDoble struct {
	ahora time.Time
}

func (r relojCoberturaDoble) Ahora() time.Time { return r.ahora }

type fuenteCoberturaDoble struct {
	presentador presentadorAutoridadConfiguradoPrueba
	consultar   func(
		context.Context,
		SolicitudConsultarCobertura,
	) (ResultadoConsultaCobertura, error)
}

func (f fuenteCoberturaDoble) PresentarAutoridadFuenteAnalisis(
	ctx context.Context,
	desafio DesafioAutoridadFuenteAnalisis,
) (PresentacionAutoridadFuenteAnalisis, error) {
	return f.presentador.PresentarAutoridadFuenteAnalisis(ctx, desafio)
}

func (f fuenteCoberturaDoble) ConsultarCobertura(
	ctx context.Context,
	solicitud SolicitudConsultarCobertura,
) (ResultadoConsultaCobertura, error) {
	return f.consultar(ctx, solicitud)
}

type verificadorCoberturaDoble struct {
	presentador presentadorAutoridadConfiguradoPrueba
	verificar   func(
		context.Context,
		SolicitudVerificarRespuestaCobertura,
	) (ConfirmacionRespuestaCobertura, error)
}

func (v verificadorCoberturaDoble) PresentarAutoridadFuenteAnalisis(
	ctx context.Context,
	desafio DesafioAutoridadFuenteAnalisis,
) (PresentacionAutoridadFuenteAnalisis, error) {
	return v.presentador.PresentarAutoridadFuenteAnalisis(ctx, desafio)
}

func (v verificadorCoberturaDoble) VerificarRespuestaCobertura(
	ctx context.Context,
	solicitud SolicitudVerificarRespuestaCobertura,
) (ConfirmacionRespuestaCobertura, error) {
	return v.verificar(ctx, solicitud)
}

type publicadorCoberturaDoble struct {
	presentador presentadorAutoridadConfiguradoPrueba
	publicar    func(
		context.Context,
		SolicitudConsultarCobertura,
	) (ConfirmacionPublicacionCobertura, error)
}

func (p publicadorCoberturaDoble) PresentarAutoridadFuenteAnalisis(
	ctx context.Context,
	desafio DesafioAutoridadFuenteAnalisis,
) (PresentacionAutoridadFuenteAnalisis, error) {
	return p.presentador.PresentarAutoridadFuenteAnalisis(ctx, desafio)
}

func (p publicadorCoberturaDoble) ConsultarPublicacionCobertura(
	ctx context.Context,
	solicitud SolicitudConsultarCobertura,
) (ConfirmacionPublicacionCobertura, error) {
	return p.publicar(ctx, solicitud)
}

type consumidorCoberturaDurablePrueba struct {
	mu        sync.Mutex
	registros map[string]registroConsumoCoberturaPrueba
	ahora     time.Time
}

type registroConsumoCoberturaPrueba struct {
	huella string
	recibo ReciboConsumoCobertura
}

func (c *consumidorCoberturaDurablePrueba) ConsumirCobertura(
	_ context.Context,
	orden OrdenConsumoCobertura,
) (ReciboConsumoCobertura, error) {
	datos, err := orden.Datos()
	if err != nil {
		return ReciboConsumoCobertura{}, err
	}
	clave := datos.AutoridadRef + ":" +
		time.Duration(datos.Generacion).String() + ":" +
		datos.ReciboRespuestaRef
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.registros == nil {
		c.registros = make(map[string]registroConsumoCoberturaPrueba)
	}
	if anterior, existe := c.registros[clave]; existe {
		if anterior.huella != datos.HuellaRespuestaSHA256 {
			return ReciboConsumoCobertura{},
				ErrRespuestaCoberturaYaConsumida
		}
		return anterior.recibo, nil
	}
	recibo, err := NuevoReciboConsumoCobertura(
		orden,
		"consumo_cobertura_0123456789",
		c.ahora,
	)
	if err != nil {
		return ReciboConsumoCobertura{}, err
	}
	c.registros[clave] = registroConsumoCoberturaPrueba{
		huella: datos.HuellaRespuestaSHA256,
		recibo: recibo,
	}
	return recibo, nil
}

type entornoCoberturaPrueba struct {
	solicitud   SolicitudConsultarCobertura
	fuente      fuenteCoberturaDoble
	verificador verificadorCoberturaDoble
	publicador  publicadorCoberturaDoble
	consumidor  *consumidorCoberturaDurablePrueba
	confianza   ConfianzaAutoridadesFuenteAnalisis
	reloj       relojCoberturaDoble
}

func nuevoEntornoCoberturaPrueba(t *testing.T) entornoCoberturaPrueba {
	t.Helper()
	inicio := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	solicitud, catalogo := solicitudYCatalogoCoberturaPrueba(t, inicio)
	ahora := inicio.Add(2 * time.Second)
	return entornoCoberturaPrueba{
		solicitud: solicitud,
		fuente: fuenteCoberturaDoble{
			presentador: nuevoPresentadorAutoridadConfiguradoPrueba(
				RolFuenteCobertura,
				"fuente_cobertura_bolsa_012345",
				"backend_fuente_cobertura_012345",
			),
			consultar: func(
				_ context.Context,
				recibida SolicitudConsultarCobertura,
			) (ResultadoConsultaCobertura, error) {
				return resultadoCoberturaFirmadoPrueba(t, recibida), nil
			},
		},
		verificador: verificadorCoberturaHMACPrueba(ahora),
		publicador: publicadorCoberturaDoble{
			presentador: nuevoPresentadorAutoridadConfiguradoPrueba(
				RolPublicadorCatalogoCobertura,
				"publicador_catalogo_cobertura_01",
				"backend_publicador_cobertura_01",
			),
			publicar: func(
				context.Context,
				SolicitudConsultarCobertura,
			) (ConfirmacionPublicacionCobertura, error) {
				return NuevaConfirmacionPublicacionCobertura(
					"publicador_catalogo_cobertura_01",
					catalogo.Publicacion(),
					ahora,
				)
			},
		},
		consumidor: &consumidorCoberturaDurablePrueba{ahora: ahora},
		confianza:  confianzaAutoridadesPrueba(t),
		reloj:      relojCoberturaDoble{ahora: ahora},
	}
}

func (e entornoCoberturaPrueba) consultar(
	ctx context.Context,
) (domain.ComprobacionCobertura, error) {
	return ConsultarCoberturaConFuente(
		ctx,
		e.fuente,
		e.verificador,
		e.publicador,
		e.consumidor,
		e.confianza,
		e.reloj,
		e.solicitud,
		time.Second,
	)
}

func TestConsultarCoberturaConFuenteCompletaEfectoDurable(t *testing.T) {
	entorno := nuevoEntornoCoberturaPrueba(t)
	resultado, err := entorno.consultar(context.Background())
	if err != nil || resultado.Validar() != nil ||
		resultado.Resultado != domain.ComprobacionAfirmativa ||
		resultado.FuenteRef != "fuente_cobertura_bolsa_012345" {
		t.Fatalf("consulta completa rechazada: %#v, %v", resultado, err)
	}
	repetido, err := entorno.consultar(context.Background())
	if err != nil || repetido != resultado {
		t.Fatalf("replay exacto no idempotente: %#v, %v", repetido, err)
	}
	if len(entorno.consumidor.registros) != 1 {
		t.Fatalf("se duplicó el efecto durable: %d", len(entorno.consumidor.registros))
	}
}

func TestFuenteCoberturaNoFiltraCausaPrivada(t *testing.T) {
	entorno := nuevoEntornoCoberturaPrueba(t)
	privado := &errorPrivadoCobertura{
		detalle: "dsn=secreto dni=12345678Z",
	}
	entorno.fuente.consultar = func(
		context.Context,
		SolicitudConsultarCobertura,
	) (ResultadoConsultaCobertura, error) {
		return ResultadoConsultaCobertura{}, privado
	}
	_, err := entorno.consultar(context.Background())
	var filtrado *errorPrivadoCobertura
	if !errors.Is(err, ErrFuenteCoberturaNoDisponible) ||
		errors.Is(err, privado) || errors.As(err, &filtrado) ||
		err.Error() != ErrFuenteCoberturaNoDisponible.Error() {
		t.Fatalf("se filtró la causa privada: %v", err)
	}
}

type errorPrivadoCobertura struct {
	detalle string
}

func (e *errorPrivadoCobertura) Error() string { return e.detalle }

func solicitudYCatalogoCoberturaPrueba(
	t *testing.T,
	inicio time.Time,
) (SolicitudConsultarCobertura, domain.CatalogoViasCobertura) {
	t.Helper()
	comprobacion := domain.ComprobacionExigibleCobertura{
		Clave:       "existe_bolsa_vigente",
		Orden:       1,
		Obligatoria: true,
		Procedencia: domain.ProcedenciaComprobacionCobertura{
			Clave:               "bolsa",
			DefinicionFuenteRef: "fuente_definicion_bolsa_v3",
		},
	}
	catalogo, err := domain.PublicarCatalogoViasCobertura(
		domain.BorradorCatalogoViasCobertura{
			Referencia:  "catalogo_cobertura_general",
			Version:     7,
			PublicadoEn: inicio.AddDate(0, 0, -1),
			Vigencia: domain.VigenciaCatalogoCobertura{
				Desde: inicio.AddDate(0, 0, -1),
				Hasta: inicio.AddDate(0, 1, 0),
			},
			ProcedenciaRef: "procedimiento_gobierno_catalogo_01",
			Vias: []domain.DefinicionViaCobertura{{
				Clave: "bolsa_vigente",
				Orden: 1,
				Comprobaciones: []domain.ComprobacionExigibleCobertura{
					comprobacion,
				},
			}},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return SolicitudConsultarCobertura{
		PeticionRef:       "peticion_cobertura_0123456789",
		OrganizacionRef:   organizacionAutoridadPrueba,
		ExpedienteRef:     "expediente_temporal_0123456789",
		VersionExpediente: 3,
		Catalogo:          catalogo.Identidad(),
		ViaClave:          "bolsa_vigente",
		Comprobacion:      comprobacion,
		CategoriaRef:      "categoria_trabajo_social",
		Periodo: domain.PeriodoPrevisto{
			Inicio: time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC),
			Fin:    time.Date(2027, 7, 23, 0, 0, 0, 0, time.UTC),
		},
		SolicitadaEn: inicio,
	}, catalogo
}

func resultadoCoberturaFirmadoPrueba(
	t *testing.T,
	solicitud SolicitudConsultarCobertura,
) ResultadoConsultaCobertura {
	t.Helper()
	datos := datosResultadoCoberturaPrueba(solicitud)
	return resultadoCoberturaConDatosFirmadoPrueba(t, solicitud, datos)
}

func resultadoCoberturaConDatosFirmadoPrueba(
	t *testing.T,
	solicitud SolicitudConsultarCobertura,
	datos DatosResultadoConsultaCobertura,
) ResultadoConsultaCobertura {
	t.Helper()
	metadatos := MetadatosAtestacionRespuestaCobertura{
		AutoridadRef: "fuente_cobertura_bolsa_012345",
		Generacion:   7,
		ReciboRef:    datos.Comprobacion.ReciboRef,
		EmitidaEn:    solicitud.SolicitadaEn.Add(time.Second),
		ValidaHasta:  solicitud.SolicitadaEn.Add(5 * time.Second),
	}
	preimagen, err := NuevaPreimagenRespuestaCobertura(datos, metadatos)
	if err != nil {
		t.Fatal(err)
	}
	atestacion := atestacionCoberturaPrueba(t, preimagen, metadatos)
	resultado, err := NuevoResultadoConsultaCobertura(datos, atestacion)
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}

func datosResultadoCoberturaPrueba(
	solicitud SolicitudConsultarCobertura,
) DatosResultadoConsultaCobertura {
	huellaPeticion, _ := huellaPeticionCobertura(solicitud)
	return DatosResultadoConsultaCobertura{
		PeticionRef:          solicitud.PeticionRef,
		HuellaPeticionSHA256: huellaPeticion,
		OrganizacionRef:      solicitud.OrganizacionRef,
		ExpedienteRef:        solicitud.ExpedienteRef,
		VersionExpediente:    solicitud.VersionExpediente,
		Catalogo:             solicitud.Catalogo,
		ViaClave:             solicitud.ViaClave,
		ProcedenciaClave:     solicitud.Comprobacion.Procedencia.Clave,
		CategoriaRef:         solicitud.CategoriaRef,
		Periodo:              solicitud.Periodo,
		Comprobacion: domain.ComprobacionCobertura{
			Clave:      solicitud.Comprobacion.Clave,
			Resultado:  domain.ComprobacionAfirmativa,
			FuenteRef:  "fuente_cobertura_bolsa_012345",
			ReciboRef:  "recibo_consulta_bolsa_012345",
			EvaluadaEn: solicitud.SolicitadaEn.Add(time.Second),
		},
		DefinicionFuenteRef: solicitud.Comprobacion.Procedencia.DefinicionFuenteRef,
	}
}

func atestacionCoberturaPrueba(
	t *testing.T,
	preimagen PreimagenRespuestaCobertura,
	metadatos MetadatosAtestacionRespuestaCobertura,
) AtestacionRespuestaCobertura {
	t.Helper()
	contenido, err := preimagen.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, []byte(claveRespuestaCoberturaPrueba))
	_, _ = mac.Write(contenido)
	sello := "hmac-sha256:" + dominioSelloRespuestaCobertura +
		"7:" + hex.EncodeToString(mac.Sum(nil))
	atestacion, err := NuevaAtestacionRespuestaCobertura(metadatos, sello)
	if err != nil {
		t.Fatal(err)
	}
	return atestacion
}

func verificadorCoberturaHMACPrueba(
	verificadaEn time.Time,
) verificadorCoberturaDoble {
	return verificadorCoberturaDoble{
		presentador: nuevoPresentadorAutoridadConfiguradoPrueba(
			RolVerificadorCobertura,
			"verificador_cobertura_tcb_012345",
			"backend_verificador_cobertura_01",
		),
		verificar: func(
			_ context.Context,
			solicitud SolicitudVerificarRespuestaCobertura,
		) (ConfirmacionRespuestaCobertura, error) {
			preimagen, atestacion, err := solicitud.Material()
			if err != nil {
				return ConfirmacionRespuestaCobertura{}, err
			}
			contenido, _ := preimagen.Bytes()
			mac := hmac.New(
				sha256.New,
				[]byte(claveRespuestaCoberturaPrueba),
			)
			_, _ = mac.Write(contenido)
			esperado := "hmac-sha256:" +
				dominioSelloRespuestaCobertura +
				"7:" + hex.EncodeToString(mac.Sum(nil))
			if !hmac.Equal(
				[]byte(esperado),
				[]byte(atestacion.SelloHMAC),
			) {
				return ConfirmacionRespuestaCobertura{},
					ErrResultadoFuenteCoberturaNoConfiable
			}
			return NuevaConfirmacionRespuestaCobertura(
				solicitud,
				"verificador_cobertura_tcb_012345",
				verificadaEn,
			)
		},
	}
}

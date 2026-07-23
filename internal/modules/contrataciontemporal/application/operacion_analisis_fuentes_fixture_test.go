package application

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	organizacionFuentesAplicacionPrueba = "organizacion:sintetica-001"
	audienciaFuentesAplicacionPrueba    = "audiencia_fuentes_sintetica_001"
	raizFuentesAplicacionPrueba         = "raiz_fuentes_sintetica_001"
)

type relojFuentesAplicacionPrueba struct{ instante time.Time }

func (r relojFuentesAplicacionPrueba) Ahora() time.Time { return r.instante }

type generadorFuentesAplicacionPrueba struct{}

func (generadorFuentesAplicacionPrueba) NuevaReferenciaPeticionFuenteAnalisis(
	_ context.Context,
	tipo ports.TipoPeticionFuenteAnalisis,
) (string, error) {
	switch tipo {
	case ports.TipoPeticionValidacionRC:
		return "pet_rc_sintetica_0123456789abcdef", nil
	case ports.TipoPeticionCalculoCoste:
		return "pet_coste_sintetica_0123456789abcd", nil
	default:
		return "", errors.New("tipo-sintetico-desconocido")
	}
}

type selladorFuentesAplicacionPrueba struct{}

func (selladorFuentesAplicacionPrueba) SellarPeticionFuenteAnalisis(
	_ context.Context,
	preimagen ports.PreimagenPeticionFuenteAnalisis,
) (string, error) {
	contenido, err := preimagen.Bytes()
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, []byte("material-sintetico-solicitud"))
	_, _ = mac.Write(contenido)
	return "hmac-sha256:fuente-analisis-v1:" +
		hex.EncodeToString(mac.Sum(nil)), nil
}

type preparadorSolicitudesFuentesAplicacionPrueba struct {
	reloj relojFuentesAplicacionPrueba
}

func (p preparadorSolicitudesFuentesAplicacionPrueba) PrepararSolicitudesFuentesAnalisisO3(
	ctx context.Context,
	solicitud ports.SolicitudPrepararArtefactoAnalisis,
) (ports.SolicitudesFuentesAnalisisO3, error) {
	fecha := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	solicitudRC, err := ports.NuevaSolicitudValidarRC(
		ctx,
		generadorFuentesAplicacionPrueba{},
		selladorFuentesAplicacionPrueba{},
		p.reloj,
		ports.PreparacionSolicitudValidarRC{
			OrganizacionRef:   solicitud.OrganizacionRef,
			ExpedienteRef:     solicitud.ExpedienteRef,
			VersionExpediente: solicitud.VersionExpediente,
			Entrada:           solicitud.DatosFuncionales.EntradaRC,
			Declaracion: domain.DeclaracionRC{
				Existe: true, Numero: "rc_sintetica_012345",
				Fecha: fecha,
				Importe: domain.Importe{
					Centimos: 5_000_000,
					Moneda:   "EUR",
				},
				DocumentoRef: "documento_rc_sintetico_012345",
			},
		},
	)
	if err != nil {
		return ports.SolicitudesFuentesAnalisisO3{}, err
	}
	solicitudCoste, err := ports.NuevaSolicitudCalcularCoste(
		ctx,
		generadorFuentesAplicacionPrueba{},
		selladorFuentesAplicacionPrueba{},
		p.reloj,
		ports.PreparacionSolicitudCalcularCoste{
			OrganizacionRef:   solicitud.OrganizacionRef,
			ExpedienteRef:     solicitud.ExpedienteRef,
			VersionExpediente: solicitud.VersionExpediente,
			CategoriaRef:      solicitud.DatosFuncionales.CategoriaRef,
			GrupoSubgrupo:     solicitud.DatosFuncionales.GrupoSubgrupo,
			ModalidadClave:    solicitud.DatosFuncionales.ModalidadClave,
			CausaClave:        solicitud.DatosFuncionales.CausaClave,
			Periodo:           solicitud.DatosFuncionales.Periodo,
			Jornada:           solicitud.DatosFuncionales.PorcentajeJornada,
		},
	)
	if err != nil {
		return ports.SolicitudesFuentesAnalisisO3{}, err
	}
	return ports.SolicitudesFuentesAnalisisO3{
		ValidacionRC: solicitudRC,
		CalculoCoste: &solicitudCoste,
	}, nil
}

type autoridadFuentesAplicacionPrueba struct {
	raizPrivada ed25519.PrivateKey
	prueba      ed25519.PrivateKey
	datos       ports.DatosCredencialAutoridadFuenteAnalisis
}

func (a autoridadFuentesAplicacionPrueba) presentar(
	desafio ports.DesafioAutoridadFuenteAnalisis,
) (ports.PresentacionAutoridadFuenteAnalisis, error) {
	documento, err := canonCredencialFuentesAplicacionPrueba(a.datos)
	if err != nil {
		return ports.PresentacionAutoridadFuenteAnalisis{}, err
	}
	credencial, err := ports.NuevaCredencialAutoridadFuenteAnalisis(
		a.datos,
		ed25519.Sign(a.raizPrivada, documento),
	)
	if err != nil {
		return ports.PresentacionAutoridadFuenteAnalisis{}, err
	}
	material, err := desafio.Bytes()
	if err != nil {
		return ports.PresentacionAutoridadFuenteAnalisis{}, err
	}
	return ports.NuevaPresentacionAutoridadFuenteAnalisis(
		credencial,
		ed25519.Sign(a.prueba, material),
	)
}

type fuenteRCAplicacionPrueba struct {
	autoridad autoridadFuentesAplicacionPrueba
	instante  time.Time
	claveHMAC []byte
}

func (f fuenteRCAplicacionPrueba) PresentarAutoridadFuenteAnalisis(
	_ context.Context,
	desafio ports.DesafioAutoridadFuenteAnalisis,
) (ports.PresentacionAutoridadFuenteAnalisis, error) {
	return f.autoridad.presentar(desafio)
}

func (f fuenteRCAplicacionPrueba) ValidarRC(
	_ context.Context,
	solicitud ports.SolicitudValidarRC,
) (ports.ResultadoValidacionRC, error) {
	datos, err := solicitud.Datos()
	if err != nil {
		return ports.ResultadoValidacionRC{}, err
	}
	fecha := datos.Declaracion.Fecha
	importe := datos.Declaracion.Importe
	validacion := domain.ValidacionRC{
		Resultado:           domain.RCValidada,
		EntradaRef:          datos.Entrada.Referencia,
		HuellaEntradaSHA256: datos.Entrada.HuellaSHA256,
		FuenteRef:           f.autoridad.datos.AutoridadRef,
		ReciboRef:           "recibo_rc_sintetico_012345",
		ValidadaEn:          f.instante,
		FechaRC:             &fecha,
		Numero:              datos.Declaracion.Numero,
		Importe:             &importe,
		DocumentoRef:        datos.Declaracion.DocumentoRef,
	}
	metadatos := ports.MetadatosAtestacionRespuestaFuenteAnalisis{
		AutoridadRef: f.autoridad.datos.AutoridadRef,
		Generacion:   7,
		ReciboRef:    validacion.ReciboRef,
		EmitidaEn:    f.instante,
		ValidaHasta:  f.instante.Add(ports.VigenciaMaximaRespuestaFuenteAnalisis),
	}
	preimagen, err := ports.NuevaPreimagenRespuestaValidacionRC(
		solicitud,
		validacion,
		ports.MotivoFuenteAnalisis{},
		metadatos,
	)
	if err != nil {
		return ports.ResultadoValidacionRC{}, err
	}
	atestacion, err := atestacionFuentesAplicacionPrueba(
		preimagen,
		metadatos,
		f.claveHMAC,
	)
	if err != nil {
		return ports.ResultadoValidacionRC{}, err
	}
	return ports.NuevoResultadoValidacionRC(
		solicitud,
		validacion,
		ports.MotivoFuenteAnalisis{},
		atestacion,
	)
}

type calculadorCosteAplicacionPrueba struct {
	autoridad autoridadFuentesAplicacionPrueba
	instante  time.Time
	claveHMAC []byte
}

func (c calculadorCosteAplicacionPrueba) PresentarAutoridadFuenteAnalisis(
	_ context.Context,
	desafio ports.DesafioAutoridadFuenteAnalisis,
) (ports.PresentacionAutoridadFuenteAnalisis, error) {
	return c.autoridad.presentar(desafio)
}

func (c calculadorCosteAplicacionPrueba) CalcularCoste(
	_ context.Context,
	solicitud ports.SolicitudCalcularCoste,
) (ports.ResultadoCalculoCoste, error) {
	metadatos := ports.MetadatosAtestacionRespuestaFuenteAnalisis{
		AutoridadRef: c.autoridad.datos.AutoridadRef,
		Generacion:   7,
		ReciboRef:    "recibo_coste_sintetico_012345",
		EmitidaEn:    c.instante,
		ValidaHasta:  c.instante.Add(ports.VigenciaMaximaRespuestaFuenteAnalisis),
	}
	importe := domain.Importe{Centimos: 4_000_000, Moneda: "EUR"}
	preimagen, err := ports.NuevaPreimagenRespuestaCalculoCoste(
		solicitud,
		c.autoridad.datos.AutoridadRef,
		metadatos.ReciboRef,
		importe,
		c.instante,
		metadatos,
	)
	if err != nil {
		return ports.ResultadoCalculoCoste{}, err
	}
	atestacion, err := atestacionFuentesAplicacionPrueba(
		preimagen,
		metadatos,
		c.claveHMAC,
	)
	if err != nil {
		return ports.ResultadoCalculoCoste{}, err
	}
	return ports.NuevoResultadoCalculoCoste(
		solicitud,
		c.autoridad.datos.AutoridadRef,
		metadatos.ReciboRef,
		importe,
		c.instante,
		atestacion,
	)
}

type verificadorFuentesAplicacionPrueba struct {
	autoridad autoridadFuentesAplicacionPrueba
	instante  time.Time
	claveHMAC []byte
}

func (v verificadorFuentesAplicacionPrueba) PresentarAutoridadFuenteAnalisis(
	_ context.Context,
	desafio ports.DesafioAutoridadFuenteAnalisis,
) (ports.PresentacionAutoridadFuenteAnalisis, error) {
	return v.autoridad.presentar(desafio)
}

func (v verificadorFuentesAplicacionPrueba) VerificarRespuestaFuenteAnalisis(
	_ context.Context,
	solicitud ports.SolicitudVerificarRespuestaFuenteAnalisis,
) (ports.ConfirmacionRespuestaFuenteAnalisis, error) {
	preimagen, atestacion, err := solicitud.Material()
	if err != nil {
		return ports.ConfirmacionRespuestaFuenteAnalisis{}, err
	}
	contenido, err := preimagen.Bytes()
	if err != nil {
		return ports.ConfirmacionRespuestaFuenteAnalisis{}, err
	}
	mac := hmac.New(sha256.New, v.claveHMAC)
	_, _ = mac.Write(contenido)
	esperado := "hmac-sha256:fuente-analisis-respuesta/v7:" +
		hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(esperado), []byte(atestacion.SelloHMAC)) {
		return ports.ConfirmacionRespuestaFuenteAnalisis{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	return ports.NuevaConfirmacionRespuestaFuenteAnalisis(
		solicitud,
		v.autoridad.datos.AutoridadRef,
		v.instante,
	)
}

type publicadorFuentesAplicacionPrueba struct {
	autoridad autoridadFuentesAplicacionPrueba
}

func (p publicadorFuentesAplicacionPrueba) PresentarAutoridadFuenteAnalisis(
	_ context.Context,
	desafio ports.DesafioAutoridadFuenteAnalisis,
) (ports.PresentacionAutoridadFuenteAnalisis, error) {
	return p.autoridad.presentar(desafio)
}

func (publicadorFuentesAplicacionPrueba) VerificarPublicacionMotivoFuenteAnalisis(
	context.Context,
	ports.SolicitudVerificarPublicacionMotivoFuenteAnalisis,
) (ports.ConfirmacionPublicacionMotivoFuenteAnalisis, error) {
	return ports.ConfirmacionPublicacionMotivoFuenteAnalisis{},
		errors.New("motivo-sintetico-no-esperado")
}

func nuevoPreparadorArtefactoAnalisisO3AplicacionPrueba(
	t *testing.T,
	instante time.Time,
) ports.PreparadorArtefactoAnalisisO3 {
	t.Helper()
	publicaRaiz, privadaRaiz, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	nuevaAutoridad := func(
		rol ports.RolAutoridadFuenteAnalisis,
		autoridadRef string,
		backendRef string,
	) autoridadFuentesAplicacionPrueba {
		publica, privada, errClave := ed25519.GenerateKey(rand.Reader)
		if errClave != nil {
			t.Fatal(errClave)
		}
		return autoridadFuentesAplicacionPrueba{
			raizPrivada: privadaRaiz,
			prueba:      privada,
			datos: ports.DatosCredencialAutoridadFuenteAnalisis{
				RaizClaveID:        raizFuentesAplicacionPrueba,
				AutoridadRef:       autoridadRef,
				BackendRef:         backendRef,
				OrganizacionRef:    organizacionFuentesAplicacionPrueba,
				Audiencia:          audienciaFuentesAplicacionPrueba,
				Rol:                rol,
				Serie:              1,
				Generacion:         1,
				ClavePruebaEd25519: publica,
				EmitidaEn: instante.Add(
					-24 * time.Hour,
				),
				ValidaHasta: instante.Add(24 * time.Hour),
			},
		}
	}
	fuente := fuenteRCAplicacionPrueba{
		autoridad: nuevaAutoridad(
			ports.RolFuentePresupuestaria,
			"fuente_rc_sintetica_012345",
			"backend_rc_sintetico_012345",
		),
		instante: instante,
	}
	calculador := calculadorCosteAplicacionPrueba{
		autoridad: nuevaAutoridad(
			ports.RolCalculadorCoste,
			"fuente_coste_sintetica_012345",
			"backend_coste_sintetico_012345",
		),
		instante: instante,
	}
	verificador := verificadorFuentesAplicacionPrueba{
		autoridad: nuevaAutoridad(
			ports.RolVerificadorRespuesta,
			"verificador_sintetico_012345",
			"backend_verificador_sintetico_012345",
		),
		instante: instante,
	}
	publicador := publicadorFuentesAplicacionPrueba{
		autoridad: nuevaAutoridad(
			ports.RolPublicadorCatalogo,
			"publicador_sintetico_012345",
			"backend_publicador_sintetico_012345",
		),
	}
	claveHMAC := sha256.Sum256([]byte("material-sintetico-respuesta"))
	fuente.claveHMAC = claveHMAC[:]
	calculador.claveHMAC = claveHMAC[:]
	verificador.claveHMAC = claveHMAC[:]
	confianza, err := ports.NuevaConfianzaAutoridadesFuenteAnalisis(
		organizacionFuentesAplicacionPrueba,
		audienciaFuentesAplicacionPrueba,
		[]ports.RaizConfianzaAutoridadFuenteAnalisis{{
			ClaveID:             raizFuentesAplicacionPrueba,
			ClavePublicaEd25519: publicaRaiz,
			Estado:              ports.RaizAutoridadActiva,
			ValidaDesde:         instante.Add(-48 * time.Hour),
			ValidaHasta:         instante.Add(48 * time.Hour),
			UltimaEmisionPermitida: instante.Add(
				24 * time.Hour,
			),
		}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	capacidad, err := NuevaCapacidadPrepararArtefactoAnalisisO3ParaComposicionInterna(
		preparadorSolicitudesFuentesAplicacionPrueba{
			reloj: relojFuentesAplicacionPrueba{instante: instante},
		},
		fuente,
		calculador,
		verificador,
		publicador,
		confianza,
		relojFuentesAplicacionPrueba{instante: instante},
	)
	if err != nil {
		t.Fatal(err)
	}
	return capacidad
}

func atestacionFuentesAplicacionPrueba(
	preimagen ports.PreimagenRespuestaFuenteAnalisis,
	metadatos ports.MetadatosAtestacionRespuestaFuenteAnalisis,
	clave []byte,
) (ports.AtestacionRespuestaFuenteAnalisis, error) {
	contenido, err := preimagen.Bytes()
	if err != nil {
		return ports.AtestacionRespuestaFuenteAnalisis{}, err
	}
	mac := hmac.New(sha256.New, clave)
	_, _ = mac.Write(contenido)
	return ports.NuevaAtestacionRespuestaFuenteAnalisis(
		metadatos,
		"hmac-sha256:fuente-analisis-respuesta/v7:"+
			hex.EncodeToString(mac.Sum(nil)),
	)
}

func canonCredencialFuentesAplicacionPrueba(
	datos ports.DatosCredencialAutoridadFuenteAnalisis,
) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	escribirTextoCredencialAplicacionPrueba(
		buffer,
		"VEC-CT-CREDENCIAL-AUTORIDAD-FUENTE-ANALISIS-V1",
	)
	for _, valor := range []string{
		datos.RaizClaveID,
		datos.AutoridadRef,
		datos.BackendRef,
		datos.OrganizacionRef,
		datos.Audiencia,
		string(datos.Rol),
	} {
		escribirTextoCredencialAplicacionPrueba(buffer, valor)
	}
	var entero [8]byte
	binary.BigEndian.PutUint64(entero[:], datos.Serie)
	_, _ = buffer.Write(entero[:])
	var generacion [4]byte
	binary.BigEndian.PutUint32(generacion[:], datos.Generacion)
	_, _ = buffer.Write(generacion[:])
	_, _ = buffer.Write(datos.ClavePruebaEd25519)
	binary.BigEndian.PutUint64(
		entero[:],
		uint64(datos.EmitidaEn.UnixMicro()),
	)
	_, _ = buffer.Write(entero[:])
	binary.BigEndian.PutUint64(
		entero[:],
		uint64(datos.ValidaHasta.UnixMicro()),
	)
	_, _ = buffer.Write(entero[:])
	return buffer.Bytes(), nil
}

func escribirTextoCredencialAplicacionPrueba(
	buffer *bytes.Buffer,
	valor string,
) {
	var longitud [4]byte
	binary.BigEndian.PutUint32(longitud[:], uint32(len(valor)))
	_, _ = buffer.Write(longitud[:])
	_, _ = buffer.WriteString(valor)
}

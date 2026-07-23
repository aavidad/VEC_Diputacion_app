package application

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func identidadCoberturaAplicacionPrueba(
	t interfazPruebaCobertura,
	autoridadRef string,
	backendRef string,
	clave ed25519.PrivateKey,
	rol ports.RolAutoridadFuenteAnalisis,
) ports.IdentidadAutoridadFuenteAnalisis {
	t.Helper()
	identidad, err := ports.NuevaIdentidadAutoridadFuenteAnalisis(
		autoridadRef,
		backendRef,
		clave.Public().(ed25519.PublicKey),
		rol,
	)
	if err != nil {
		t.Fatal(err)
	}
	return identidad
}

type interfazPruebaCobertura interface {
	Helper()
	Fatal(...any)
}

func solicitudCatalogoCoberturaAplicacionPrueba(
	t interfazPruebaCobertura,
	inicio time.Time,
) (ports.SolicitudConsultarCobertura, domain.CatalogoViasCobertura) {
	t.Helper()
	comprobacion := domain.ComprobacionExigibleCobertura{
		Clave: "existe_bolsa_vigente", Orden: 1, Obligatoria: true,
		Procedencia: domain.ProcedenciaComprobacionCobertura{
			Clave:               "bolsa",
			DefinicionFuenteRef: "fuente_definicion_bolsa_v3",
		},
	}
	catalogo, err := domain.PublicarCatalogoViasCobertura(
		domain.BorradorCatalogoViasCobertura{
			Referencia:  "catalogo_cobertura_general",
			Version:     7,
			PublicadoEn: inicio.Add(-time.Hour),
			Vigencia: domain.VigenciaCatalogoCobertura{
				Desde: inicio.Add(-time.Hour),
				Hasta: inicio.Add(time.Hour),
			},
			ProcedenciaRef: "procedimiento_gobierno_catalogo_01",
			Vias: []domain.DefinicionViaCobertura{{
				Clave: "bolsa_vigente", Orden: 1,
				Comprobaciones: []domain.ComprobacionExigibleCobertura{
					comprobacion,
				},
			}},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return ports.SolicitudConsultarCobertura{
		PeticionRef:       "peticion_cobertura_0123456789",
		OrganizacionRef:   organizacionCoberturaPrueba,
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

func resultadoCoberturaAplicacionPrueba(
	t interfazPruebaCobertura,
	solicitud ports.SolicitudConsultarCobertura,
	alterar func(*ports.DatosResultadoConsultaCobertura),
) ports.ResultadoConsultaCobertura {
	t.Helper()
	material, err := solicitud.MaterialCanonico()
	if err != nil {
		t.Fatal(err)
	}
	huella := sha256.Sum256(material)
	datos := ports.DatosResultadoConsultaCobertura{
		PeticionRef:          solicitud.PeticionRef,
		HuellaPeticionSHA256: hex.EncodeToString(huella[:]),
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
	if alterar != nil {
		alterar(&datos)
	}
	metadatos := ports.MetadatosAtestacionRespuestaCobertura{
		AutoridadRef: datos.Comprobacion.FuenteRef,
		Generacion:   7,
		ReciboRef:    datos.Comprobacion.ReciboRef,
		EmitidaEn:    solicitud.SolicitadaEn.Add(time.Second),
		ValidaHasta:  solicitud.SolicitadaEn.Add(5 * time.Second),
	}
	preimagen, err := ports.NuevaPreimagenRespuestaCobertura(
		datos,
		metadatos,
	)
	if err != nil {
		t.Fatal(err)
	}
	contenido, _ := preimagen.Bytes()
	mac := hmac.New(sha256.New, []byte(claveHMACCoberturaPrueba))
	_, _ = mac.Write(contenido)
	sello := "hmac-sha256:fuente-cobertura-respuesta/v7:" +
		hex.EncodeToString(mac.Sum(nil))
	atestacion, err := ports.NuevaAtestacionRespuestaCobertura(
		metadatos,
		sello,
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := ports.NuevoResultadoConsultaCobertura(
		datos,
		atestacion,
	)
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}

func verificarRespuestaCoberturaAplicacionPrueba(
	solicitud ports.SolicitudVerificarRespuestaCobertura,
	verificadorRef string,
	clave ed25519.PrivateKey,
	verificadaEn time.Time,
) (ports.ConfirmacionRespuestaCobertura, error) {
	preimagen, atestacion, err := solicitud.Material()
	if err != nil {
		return ports.ConfirmacionRespuestaCobertura{}, err
	}
	contenido, _ := preimagen.Bytes()
	mac := hmac.New(sha256.New, []byte(claveHMACCoberturaPrueba))
	_, _ = mac.Write(contenido)
	esperado := "hmac-sha256:fuente-cobertura-respuesta/v7:" +
		hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(esperado), []byte(atestacion.SelloHMAC)) {
		return ports.ConfirmacionRespuestaCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	preimagenConfirmacion, err :=
		ports.NuevaPreimagenConfirmacionRespuestaCobertura(
			solicitud,
			verificadorRef,
			verificadaEn,
		)
	if err != nil {
		return ports.ConfirmacionRespuestaCobertura{}, err
	}
	materialConfirmacion, _ := preimagenConfirmacion.Bytes()
	return ports.NuevaConfirmacionRespuestaCobertura(
		solicitud,
		verificadorRef,
		verificadaEn,
		ed25519.Sign(clave, materialConfirmacion),
	)
}

type errorPrivadoCoberturaAplicacionPrueba struct {
	detalle string
}

func (e *errorPrivadoCoberturaAplicacionPrueba) Error() string {
	return e.detalle
}

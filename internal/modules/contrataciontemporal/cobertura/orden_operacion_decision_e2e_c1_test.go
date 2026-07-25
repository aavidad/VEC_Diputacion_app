package cobertura_test

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	organizacionOrdenC3 = "organizacion_diputacion_granada"
	audienciaOrdenC3    = "servicio_contratacion_temporal"
	claveHMACOrdenC3    = "clave-prueba-orden-c3-cobertura"
)

type relojFuenteOrdenC3 struct{ ahora time.Time }

func (r *relojFuenteOrdenC3) Ahora() time.Time { return r.ahora }

type autenticadorFuenteOrdenC3 struct {
	identidades map[ports.RolAutoridadFuenteAnalisis]ports.IdentidadAutoridadFuenteAnalisis
}

func (*autenticadorFuenteOrdenC3) OrganizacionAutoridadFuenteAnalisis() string {
	return organizacionOrdenC3
}

func (*autenticadorFuenteOrdenC3) AudienciaAutoridadFuenteAnalisis() string {
	return audienciaOrdenC3
}

func (a *autenticadorFuenteOrdenC3) VerificarEvidenciaPublicaAutoridadFuenteAnalisis(
	evidencia ports.EvidenciaPublicaAutoridadFuenteAnalisis,
) (ports.IdentidadAutoridadFuenteAnalisis, error) {
	_, _, rol, _, err := evidencia.Datos()
	if err != nil {
		return ports.IdentidadAutoridadFuenteAnalisis{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	identidad, existe := a.identidades[rol]
	if !existe || identidad.Rol() != rol {
		return ports.IdentidadAutoridadFuenteAnalisis{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	return identidad, nil
}

type fuenteOrdenC3 struct {
	identidad ports.IdentidadAutoridadFuenteAnalisis
	ahora     time.Time
}

func (f *fuenteOrdenC3) PresentarAutoridadFuenteAnalisis(
	context.Context,
	ports.DesafioAutoridadFuenteAnalisis,
) (ports.PresentacionAutoridadFuenteAnalisis, error) {
	return presentacionFuenteOrdenC3(f.identidad)
}

func (f *fuenteOrdenC3) ConsultarCobertura(
	_ context.Context,
	solicitud ports.SolicitudConsultarCobertura,
) (ports.ResultadoConsultaCobertura, error) {
	material, err := solicitud.MaterialCanonico()
	if err != nil {
		return ports.ResultadoConsultaCobertura{}, err
	}
	huellaPeticion := sha256.Sum256(material)
	datos := ports.DatosResultadoConsultaCobertura{
		PeticionRef:          solicitud.PeticionRef,
		HuellaPeticionSHA256: hex.EncodeToString(huellaPeticion[:]),
		OrganizacionRef:      solicitud.OrganizacionRef,
		ExpedienteRef:        solicitud.ExpedienteRef,
		VersionExpediente:    solicitud.VersionExpediente,
		Catalogo:             solicitud.Catalogo,
		ViaClave:             solicitud.ViaClave,
		ProcedenciaClave:     solicitud.Comprobacion.Procedencia.Clave,
		CategoriaRef:         solicitud.CategoriaRef,
		Periodo:              solicitud.Periodo,
		Comprobacion: domain.ComprobacionCobertura{
			Clave: solicitud.Comprobacion.Clave,
			Resultado: domain.
				ComprobacionAfirmativa,
			FuenteRef:  f.identidad.AutoridadRef(),
			ReciboRef:  "recibo_respuesta_cobertura_orden_c3_01",
			EvaluadaEn: f.ahora,
		},
		DefinicionFuenteRef: solicitud.Comprobacion.Procedencia.
			DefinicionFuenteRef,
	}
	metadatos := ports.MetadatosAtestacionRespuestaCobertura{
		AutoridadRef: datos.Comprobacion.FuenteRef,
		Generacion:   1,
		ReciboRef:    datos.Comprobacion.ReciboRef,
		EmitidaEn:    f.ahora,
		ValidaHasta:  f.ahora.Add(4 * time.Second),
	}
	preimagen, err := ports.NuevaPreimagenRespuestaCobertura(datos, metadatos)
	if err != nil {
		return ports.ResultadoConsultaCobertura{}, err
	}
	contenido, _ := preimagen.Bytes()
	mac := hmac.New(sha256.New, []byte(claveHMACOrdenC3))
	_, _ = mac.Write(contenido)
	atestacion, err := ports.NuevaAtestacionRespuestaCobertura(
		metadatos,
		"hmac-sha256:fuente-cobertura-respuesta/v1:"+
			hex.EncodeToString(mac.Sum(nil)),
	)
	if err != nil {
		return ports.ResultadoConsultaCobertura{}, err
	}
	return ports.NuevoResultadoConsultaCobertura(datos, atestacion)
}

type verificadorFuenteOrdenC3 struct {
	identidad ports.IdentidadAutoridadFuenteAnalisis
	clave     ed25519.PrivateKey
	ahora     time.Time
}

func (v *verificadorFuenteOrdenC3) PresentarAutoridadFuenteAnalisis(
	context.Context,
	ports.DesafioAutoridadFuenteAnalisis,
) (ports.PresentacionAutoridadFuenteAnalisis, error) {
	return presentacionFuenteOrdenC3(v.identidad)
}

func (v *verificadorFuenteOrdenC3) VerificarRespuestaCobertura(
	_ context.Context,
	solicitud ports.SolicitudVerificarRespuestaCobertura,
) (ports.ConfirmacionRespuestaCobertura, error) {
	preimagen, atestacion, err := solicitud.Material()
	if err != nil {
		return ports.ConfirmacionRespuestaCobertura{}, err
	}
	contenido, _ := preimagen.Bytes()
	mac := hmac.New(sha256.New, []byte(claveHMACOrdenC3))
	_, _ = mac.Write(contenido)
	esperado := "hmac-sha256:fuente-cobertura-respuesta/v1:" +
		hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(esperado), []byte(atestacion.SelloHMAC)) {
		return ports.ConfirmacionRespuestaCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	aFirmar, err := ports.NuevaPreimagenConfirmacionRespuestaCobertura(
		solicitud,
		v.identidad.AutoridadRef(),
		v.ahora,
	)
	if err != nil {
		return ports.ConfirmacionRespuestaCobertura{}, err
	}
	material, _ := aFirmar.Bytes()
	return ports.NuevaConfirmacionRespuestaCobertura(
		solicitud,
		v.identidad.AutoridadRef(),
		v.ahora,
		ed25519.Sign(v.clave, material),
	)
}

type publicadorFuenteOrdenC3 struct {
	identidad ports.IdentidadAutoridadFuenteAnalisis
	catalogo  domain.CatalogoViasCobertura
	ahora     time.Time
}

func (p *publicadorFuenteOrdenC3) PresentarAutoridadFuenteAnalisis(
	context.Context,
	ports.DesafioAutoridadFuenteAnalisis,
) (ports.PresentacionAutoridadFuenteAnalisis, error) {
	return presentacionFuenteOrdenC3(p.identidad)
}

func (p *publicadorFuenteOrdenC3) ConsultarPublicacionCobertura(
	context.Context,
	ports.SolicitudConsultarCobertura,
) (ports.ConfirmacionPublicacionCobertura, error) {
	return ports.NuevaConfirmacionPublicacionCobertura(
		p.identidad.AutoridadRef(),
		p.catalogo.Publicacion(),
		p.ahora,
	)
}

func prepararC1OrdenC3(
	t *testing.T,
	expediente domain.Expediente,
	analisisRef string,
	analisisHuella string,
	catalogo domain.CatalogoViasCobertura,
	politica domain.PoliticaDecisionCobertura,
	preparadaEn time.Time,
) cobertura.PreparacionConjuntosViasCobertura {
	t.Helper()
	claves := map[ports.RolAutoridadFuenteAnalisis]ed25519.PrivateKey{
		ports.RolFuenteCobertura: claveFuenteOrdenC3("fuente"),
		ports.RolVerificadorCobertura: claveFuenteOrdenC3(
			"verificador",
		),
		ports.RolPublicadorCatalogoCobertura: claveFuenteOrdenC3(
			"publicador",
		),
	}
	identidades := map[ports.RolAutoridadFuenteAnalisis]ports.IdentidadAutoridadFuenteAnalisis{}
	for rol, clave := range claves {
		identidades[rol] = identidadFuenteOrdenC3(t, rol, clave)
	}
	fuente := &fuenteOrdenC3{
		identidad: identidades[ports.RolFuenteCobertura],
		ahora:     preparadaEn,
	}
	verificador := &verificadorFuenteOrdenC3{
		identidad: identidades[ports.RolVerificadorCobertura],
		clave:     claves[ports.RolVerificadorCobertura],
		ahora:     preparadaEn,
	}
	publicador := &publicadorFuenteOrdenC3{
		identidad: identidades[ports.RolPublicadorCatalogoCobertura],
		catalogo:  catalogo,
		ahora:     preparadaEn,
	}
	preparador, err := application.NuevoPreparadorConsultaCobertura(
		fuente,
		verificador,
		publicador,
		&autenticadorFuenteOrdenC3{identidades: identidades},
		&relojFuenteOrdenC3{ahora: preparadaEn},
		time.Second,
	)
	if err != nil {
		t.Fatalf("crear preparador C1: %v", err)
	}
	via := catalogo.Vias()[0]
	solicitud := ports.SolicitudConsultarCobertura{
		PeticionRef:       "peticion_cobertura_orden_c3_01",
		OrganizacionRef:   expediente.OrganizacionRef,
		ExpedienteRef:     expediente.Referencia,
		VersionExpediente: expediente.Version,
		Catalogo:          catalogo.Identidad(),
		ViaClave:          via.Clave,
		Comprobacion:      via.Comprobaciones[0],
		CategoriaRef:      expediente.Solicitud.CategoriaRef,
		Periodo:           expediente.Solicitud.Periodo,
		SolicitadaEn:      preparadaEn,
	}
	evidencia, err := preparador.ConsultarConEvidencia(
		context.Background(),
		solicitud,
	)
	if err != nil {
		t.Fatalf("consultar evidencia C1: %v", err)
	}
	finalidadClave, finalidadRef := politica.Finalidad()
	conjunto, err := cobertura.NuevoConjuntoEvidenciasCobertura(
		cobertura.CoordenadasConjuntoEvidencias{
			OrganizacionRef:   expediente.OrganizacionRef,
			ExpedienteRef:     expediente.Referencia,
			VersionExpediente: expediente.Version,
			Catalogo:          catalogo.Identidad(),
			Politica:          politica.Identidad(),
			FinalidadClave:    finalidadClave,
			FinalidadRef:      finalidadRef,
			ViaClave:          via.Clave,
			CategoriaRef:      expediente.Solicitud.CategoriaRef,
			Periodo:           expediente.Solicitud.Periodo,
		},
		catalogo,
		politica,
		[]cobertura.EvidenciaConsultaCobertura{evidencia},
		preparadaEn,
	)
	if err != nil {
		t.Fatalf("crear conjunto C1: %v", err)
	}
	preparacion, err := cobertura.PrepararConjuntosViasCobertura(
		cobertura.DatosPrepararConjuntosViasCobertura{
			AnalisisRef: analisisRef, AnalisisHuellaSHA256: analisisHuella,
			Catalogo: catalogo, Politica: politica,
			Conjuntos:   []cobertura.ConjuntoEvidenciasCobertura{conjunto},
			PreparadaEn: preparadaEn,
		},
	)
	if err != nil {
		t.Fatalf("preparar conjuntos C1: %v", err)
	}
	return preparacion
}

func claveFuenteOrdenC3(etiqueta string) ed25519.PrivateKey {
	semilla := sha256.Sum256([]byte("VEC-CT-ORDEN-C3:" + etiqueta))
	return ed25519.NewKeyFromSeed(semilla[:])
}

func identidadFuenteOrdenC3(
	t *testing.T,
	rol ports.RolAutoridadFuenteAnalisis,
	clave ed25519.PrivateKey,
) ports.IdentidadAutoridadFuenteAnalisis {
	t.Helper()
	autoridad := "autoridad_fuente_orden_c3_01"
	backend := "fuente_definicion_bolsa_orden_c3"
	switch rol {
	case ports.RolVerificadorCobertura:
		autoridad = "autoridad_verificador_orden_c3_01"
		backend = "backend_verificador_orden_c3_01"
	case ports.RolPublicadorCatalogoCobertura:
		autoridad = "autoridad_publicador_orden_c3_01"
		backend = "backend_publicador_orden_c3_01"
	}
	identidad, err := ports.NuevaIdentidadAutoridadFuenteAnalisis(
		autoridad,
		backend,
		clave.Public().(ed25519.PublicKey),
		rol,
	)
	if err != nil {
		t.Fatal(err)
	}
	return identidad
}

func presentacionFuenteOrdenC3(
	identidad ports.IdentidadAutoridadFuenteAnalisis,
) (ports.PresentacionAutoridadFuenteAnalisis, error) {
	credencial, err := ports.NuevaCredencialAutoridadFuenteAnalisis(
		ports.DatosCredencialAutoridadFuenteAnalisis{
			RaizClaveID:        "raiz_credencial_orden_c3_01",
			AutoridadRef:       identidad.AutoridadRef(),
			BackendRef:         identidad.BackendRef(),
			OrganizacionRef:    organizacionOrdenC3,
			Audiencia:          audienciaOrdenC3,
			Rol:                identidad.Rol(),
			Serie:              1,
			Generacion:         1,
			ClavePruebaEd25519: identidad.ClavePruebaEd25519(),
			EmitidaEn:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			ValidaHasta:        time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		make([]byte, ed25519.SignatureSize),
	)
	if err != nil {
		return ports.PresentacionAutoridadFuenteAnalisis{}, err
	}
	return ports.NuevaPresentacionAutoridadFuenteAnalisis(
		credencial,
		make([]byte, ed25519.SignatureSize),
	)
}

func catalogoYPoliticaOrdenC3(
	t *testing.T,
	base time.Time,
) (
	domain.CatalogoViasCobertura,
	domain.PoliticaDecisionCobertura,
) {
	t.Helper()
	comprobacion := domain.ComprobacionExigibleCobertura{
		Clave: "existe_bolsa_vigente", Orden: 1, Obligatoria: true,
		Procedencia: domain.ProcedenciaComprobacionCobertura{
			Clave:               "bolsa",
			DefinicionFuenteRef: "fuente_definicion_bolsa_orden_c3",
		},
	}
	catalogo, err := domain.PublicarCatalogoViasCobertura(
		domain.BorradorCatalogoViasCobertura{
			Referencia:  "catalogo_cobertura_orden_c3",
			Version:     1,
			PublicadoEn: base.Add(-time.Hour),
			Vigencia: domain.VigenciaCatalogoCobertura{
				Desde: base.Add(-time.Hour),
				Hasta: base.Add(time.Hour),
			},
			ProcedenciaRef: "gobierno_catalogo_cobertura_orden_c3",
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
	politica, err := domain.PublicarPoliticaDecisionCobertura(
		domain.BorradorPoliticaDecisionCobertura{
			Referencia:      "politica_decision_cobertura_orden_c3",
			Version:         1,
			Catalogo:        catalogo.Identidad(),
			OrganizacionRef: organizacionOrdenC3,
			FinalidadClave:  "decidir_via_cobertura",
			FinalidadRef:    "finalidad_decidir_via_cobertura_orden_c3",
			PublicadaEn:     base.Add(-time.Hour),
			Vigencia: domain.VigenciaCatalogoCobertura{
				Desde: base.Add(-time.Hour),
				Hasta: base.Add(time.Hour),
			},
			ProcedenciaRef: "gobierno_politica_cobertura_orden_c3",
			Vias: []domain.ReglaViaDecisionCobertura{{
				ViaClave: "bolsa_vigente", Prioridad: 1,
				Comprobaciones: []domain.ReglaComprobacionDecisionCobertura{{
					Clave: comprobacion.Clave,
					ResultadosHabilitantes: []domain.ResultadoComprobacion{
						domain.ComprobacionAfirmativa,
					},
					TratamientoAusencia: domain.AusenciaCoberturaAdmitida,
				}},
			}},
		},
		catalogo,
	)
	if err != nil {
		t.Fatal(err)
	}
	return catalogo, politica
}

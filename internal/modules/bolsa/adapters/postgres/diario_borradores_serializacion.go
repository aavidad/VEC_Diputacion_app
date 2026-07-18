package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"time"

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const esquemaReservaDecisionBorradorPostgreSQLV2 = "vec.bolsa.convocatoria.reserva-decision.v2"

type hmacDiarioPostgreSQL struct {
	VersionEsquema  uint16 `json:"version_esquema"`
	Dominio         string `json:"dominio"`
	ClaveRef        string `json:"clave_ref"`
	GeneracionClave uint32 `json:"generacion_clave"`
	HMACSHA256      string `json:"hmac_sha256"`
}

type identidadDiarioPostgreSQL struct {
	Localizador     hmacDiarioPostgreSQL `json:"localizador"`
	HuellaSolicitud hmacDiarioPostgreSQL `json:"huella_solicitud"`
}

type atestacionPDPDiarioPostgreSQL struct {
	DecisionRef            string `json:"decision_ref"`
	AtestacionRef          string `json:"atestacion_ref"`
	Version                uint32 `json:"version"`
	Estado                 string `json:"estado"`
	HuellaAtestacionSHA256 string `json:"huella_atestacion_sha256"`
	VerificadorRef         string `json:"verificador_ref"`
	VerificadaEn           string `json:"verificada_en"`
}

type decisionDiarioPostgreSQL struct {
	EsquemaHuella                         string                        `json:"esquema_huella"`
	DecisionRef                           string                        `json:"decision_ref"`
	HuellaDecisionSHA256                  string                        `json:"huella_decision_sha256"`
	Accion                                string                        `json:"accion"`
	RecursoRef                            string                        `json:"recurso_ref"`
	ModuloID                              string                        `json:"modulo_id"`
	TipoRecurso                           string                        `json:"tipo_recurso"`
	ContextoRecursoHuellaSHA256           string                        `json:"contexto_recurso_huella_sha256"`
	Finalidad                             string                        `json:"finalidad"`
	AsignacionRef                         string                        `json:"asignacion_ref"`
	AsignacionHuellaSHA256                string                        `json:"asignacion_huella_sha256"`
	VersionRolRef                         string                        `json:"version_rol_ref"`
	VersionRolHuellaSHA256                string                        `json:"version_rol_huella_sha256"`
	ControlVigenciaVersionRolRef          string                        `json:"control_vigencia_version_rol_ref"`
	ControlVigenciaVersionRolRevision     uint64                        `json:"control_vigencia_version_rol_revision"`
	ControlVigenciaVersionRolHuellaSHA256 string                        `json:"control_vigencia_version_rol_huella_sha256"`
	RevisionCatalogoPoliticas             uint64                        `json:"revision_catalogo_politicas"`
	CatalogoPoliticasHuellaSHA256         string                        `json:"catalogo_politicas_huella_sha256"`
	EmitidaEn                             string                        `json:"emitida_en"`
	VerificadaEn                          string                        `json:"verificada_en"`
	ValidaHasta                           string                        `json:"valida_hasta"`
	AtestacionPDP                         atestacionPDPDiarioPostgreSQL `json:"atestacion_pdp"`
}

type reservaDecisionBorradorPostgreSQL struct {
	Esquema                     string                      `json:"esquema"`
	Identidad                   identidadDiarioPostgreSQL   `json:"identidad"`
	IdentidadesConsulta         []identidadDiarioPostgreSQL `json:"identidades_consulta"`
	Accion                      string                      `json:"accion"`
	Decision                    decisionDiarioPostgreSQL    `json:"decision"`
	ArrendamientoIniciaEn       string                      `json:"arrendamiento_inicia_en"`
	ArrendamientoVenceEn        string                      `json:"arrendamiento_vence_en"`
	HuellaMaterialSHA256        string                      `json:"huella_material_sha256"`
	ContextoRecursoHuellaSHA256 string                      `json:"contexto_recurso_huella_sha256"`
	RecursoRef                  string                      `json:"recurso_ref"`
	SolicitadaEn                string                      `json:"solicitada_en"`
}

type pruebaDecisionBorradorPostgreSQL struct {
	EsquemaHuella        string `json:"esquema_huella"`
	DecisionRef          string `json:"decision_ref"`
	HuellaDecisionSHA256 string `json:"huella_decision_sha256"`
	VerificadaEn         string `json:"verificada_en"`
	PrincipalRef         string `json:"principal_ref"`
}

type contextoRecursoBorradorPostgreSQL struct {
	Ambitos   map[string]string `json:"ambitos"`
	Atributos map[string]string `json:"atributos"`
}

type cargaReservaBorradorPostgreSQL struct {
	Reserva, Prueba, Material, Version, Decision, Contexto []byte
}

func serializarConsultaIdentidadesBorradorPostgreSQL(
	solicitud gobiernoconvocatorias.SolicitudConsultaIdentidadesBorrador,
) ([]byte, error) {
	if (gobiernoconvocatorias.ResultadoConsultaIdentidadesBorrador{}).ValidarPara(solicitud) != nil {
		return nil, gobiernoconvocatorias.ErrReservaBorradorInvalida
	}
	identidades := make([]identidadDiarioPostgreSQL, len(solicitud.Identidades))
	for indice, identidad := range solicitud.Identidades {
		convertida, err := proyectarIdentidadDiarioPostgreSQL(identidad)
		if err != nil {
			return nil, err
		}
		identidades[indice] = convertida
	}
	contenido, err := json.Marshal(identidades)
	if err != nil {
		return nil, gobiernoconvocatorias.ErrReservaBorradorInvalida
	}
	return contenido, nil
}

func serializarReservaBorradorPostgreSQL(
	solicitud gobiernoconvocatorias.SolicitudReservaDecisionBorrador,
) (cargaReservaBorradorPostgreSQL, error) {
	if solicitud.Validar() != nil {
		return cargaReservaBorradorPostgreSQL{}, gobiernoconvocatorias.ErrReservaBorradorInvalida
	}
	return serializarReservaValidadaBorradorPostgreSQL(solicitud)
}

// La reclamacion valida Nueva con la regla nominal validar(false): la primaria
// durable puede no ocupar la primera posicion de la ventana rotada. El llamador
// debe haber validado antes SolicitudReclamacionDecisionBorrador completa.
func serializarReservaValidadaBorradorPostgreSQL(
	solicitud gobiernoconvocatorias.SolicitudReservaDecisionBorrador,
) (cargaReservaBorradorPostgreSQL, error) {
	identidad, err := proyectarIdentidadDiarioPostgreSQL(solicitud.Proyeccion.IdentidadPrimaria)
	if err != nil {
		return cargaReservaBorradorPostgreSQL{}, err
	}
	identidades := make([]identidadDiarioPostgreSQL, len(solicitud.IdentidadesConsulta))
	for indice, candidata := range solicitud.IdentidadesConsulta {
		identidades[indice], err = proyectarIdentidadDiarioPostgreSQL(candidata)
		if err != nil {
			return cargaReservaBorradorPostgreSQL{}, err
		}
	}
	material, err := json.Marshal(solicitud.Material)
	if err != nil {
		return cargaReservaBorradorPostgreSQL{}, gobiernoconvocatorias.ErrReservaBorradorInvalida
	}
	huellaMaterial, err := solicitud.Material.HuellaSHA256()
	if err != nil || huellaBytesDiarioPostgreSQL(material) != huellaMaterial {
		borrarBytesDiarioPostgreSQL(material)
		return cargaReservaBorradorPostgreSQL{}, gobiernoconvocatorias.ErrReservaBorradorInvalida
	}
	version, err := solicitud.Version.RepresentacionCanonica()
	if err != nil {
		borrarBytesDiarioPostgreSQL(material)
		return cargaReservaBorradorPostgreSQL{}, gobiernoconvocatorias.ErrReservaBorradorInvalida
	}
	datosDecision, err := solicitud.Concesion.Evidencia.Datos()
	if err != nil {
		borrarBytesDiarioPostgreSQL(material, version)
		return cargaReservaBorradorPostgreSQL{}, gobiernoconvocatorias.ErrReservaBorradorInvalida
	}
	decisionCanonica, err := datosDecision.RepresentacionCanonica()
	if err != nil || huellaBytesDiarioPostgreSQL(decisionCanonica) != datosDecision.HuellaDecisionSHA256 {
		borrarBytesDiarioPostgreSQL(material, version, decisionCanonica)
		return cargaReservaBorradorPostgreSQL{}, gobiernoconvocatorias.ErrReservaBorradorInvalida
	}
	contexto, err := serializarContextoRecursoBorradorPostgreSQL(solicitud.Recurso)
	if err != nil {
		borrarBytesDiarioPostgreSQL(material, version, decisionCanonica)
		return cargaReservaBorradorPostgreSQL{}, err
	}
	decision := proyectarDecisionDiarioPostgreSQL(solicitud.Proyeccion.Decision)
	reserva, err := json.Marshal(reservaDecisionBorradorPostgreSQL{
		Esquema: esquemaReservaDecisionBorradorPostgreSQLV2, Identidad: identidad,
		IdentidadesConsulta: identidades, Accion: solicitud.Proyeccion.Accion, Decision: decision,
		ArrendamientoIniciaEn:       solicitud.Proyeccion.ArrendamientoIniciaEn.UTC().Format(formatoInstanteMicrosegundo),
		ArrendamientoVenceEn:        solicitud.Proyeccion.ArrendamientoVenceEn.UTC().Format(formatoInstanteMicrosegundo),
		HuellaMaterialSHA256:        huellaMaterial,
		ContextoRecursoHuellaSHA256: solicitud.Proyeccion.Decision.ContextoRecursoHuellaSHA256,
		RecursoRef:                  solicitud.Recurso.Referencia,
		SolicitadaEn:                solicitud.SolicitadaEn.UTC().Format(formatoInstanteMicrosegundo),
	})
	if err != nil {
		borrarBytesDiarioPostgreSQL(material, version, decisionCanonica, contexto)
		return cargaReservaBorradorPostgreSQL{}, gobiernoconvocatorias.ErrReservaBorradorInvalida
	}
	prueba, err := json.Marshal(pruebaDecisionBorradorPostgreSQL{
		EsquemaHuella: datosDecision.EsquemaHuella, DecisionRef: datosDecision.Decision.DecisionRef,
		HuellaDecisionSHA256: datosDecision.HuellaDecisionSHA256,
		VerificadaEn:         datosDecision.VerificadaEn.UTC().Format(formatoInstanteMicrosegundo),
		PrincipalRef:         datosDecision.Decision.PrincipalID,
	})
	if err != nil {
		borrarBytesDiarioPostgreSQL(reserva, material, version, decisionCanonica, contexto)
		return cargaReservaBorradorPostgreSQL{}, gobiernoconvocatorias.ErrReservaBorradorInvalida
	}
	return cargaReservaBorradorPostgreSQL{
		Reserva: reserva, Prueba: prueba, Material: material, Version: version,
		Decision: decisionCanonica, Contexto: contexto,
	}, nil
}

func (c cargaReservaBorradorPostgreSQL) borrar() {
	borrarBytesDiarioPostgreSQL(c.Reserva, c.Prueba, c.Material, c.Version, c.Decision, c.Contexto)
}

func proyectarIdentidadDiarioPostgreSQL(
	identidad gobiernoconvocatorias.ProyeccionIdentidadOperacion,
) (identidadDiarioPostgreSQL, error) {
	if identidad.Validar() != nil {
		return identidadDiarioPostgreSQL{}, gobiernoconvocatorias.ErrReservaBorradorInvalida
	}
	convertir := func(h gobiernoconvocatorias.ProyeccionHMACDiario) hmacDiarioPostgreSQL {
		return hmacDiarioPostgreSQL{
			VersionEsquema: h.VersionEsquema, Dominio: h.Dominio, ClaveRef: h.ClaveRef,
			GeneracionClave: h.GeneracionClave, HMACSHA256: h.ValorHMACSHA256,
		}
	}
	return identidadDiarioPostgreSQL{
		Localizador:     convertir(identidad.Localizador),
		HuellaSolicitud: convertir(identidad.HuellaSolicitud),
	}, nil
}

func restaurarIdentidadDiarioPostgreSQL(
	identidad identidadDiarioPostgreSQL,
) (gobiernoconvocatorias.ProyeccionIdentidadOperacion, error) {
	convertir := func(h hmacDiarioPostgreSQL) gobiernoconvocatorias.ProyeccionHMACDiario {
		return gobiernoconvocatorias.ProyeccionHMACDiario{
			VersionEsquema: h.VersionEsquema, Dominio: h.Dominio, ClaveRef: h.ClaveRef,
			GeneracionClave: h.GeneracionClave, ValorHMACSHA256: h.HMACSHA256,
		}
	}
	resultado := gobiernoconvocatorias.ProyeccionIdentidadOperacion{
		Localizador:     convertir(identidad.Localizador),
		HuellaSolicitud: convertir(identidad.HuellaSolicitud),
	}
	if resultado.Validar() != nil {
		return gobiernoconvocatorias.ProyeccionIdentidadOperacion{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return resultado, nil
}

func proyectarDecisionDiarioPostgreSQL(
	decision gobiernoconvocatorias.ProyeccionDecisionDiario,
) decisionDiarioPostgreSQL {
	a := decision.AtestacionPDP
	return decisionDiarioPostgreSQL{
		EsquemaHuella: decision.EsquemaHuella, DecisionRef: decision.DecisionRef,
		HuellaDecisionSHA256: decision.HuellaDecisionSHA256, Accion: decision.Accion,
		RecursoRef: decision.RecursoRef, ModuloID: decision.ModuloID, TipoRecurso: decision.TipoRecurso,
		ContextoRecursoHuellaSHA256: decision.ContextoRecursoHuellaSHA256, Finalidad: decision.Finalidad,
		AsignacionRef: decision.AsignacionRef, AsignacionHuellaSHA256: decision.AsignacionHuellaSHA256,
		VersionRolRef: decision.VersionRolRef, VersionRolHuellaSHA256: decision.VersionRolHuellaSHA256,
		ControlVigenciaVersionRolRef:          decision.ControlVigenciaVersionRolRef,
		ControlVigenciaVersionRolRevision:     decision.ControlVigenciaVersionRolRevision,
		ControlVigenciaVersionRolHuellaSHA256: decision.ControlVigenciaVersionRolHuellaSHA256,
		RevisionCatalogoPoliticas:             decision.RevisionCatalogoPoliticas,
		CatalogoPoliticasHuellaSHA256:         decision.CatalogoPoliticasHuellaSHA256,
		EmitidaEn:                             decision.EmitidaEn.UTC().Format(formatoInstanteMicrosegundo),
		VerificadaEn:                          decision.VerificadaEn.UTC().Format(formatoInstanteMicrosegundo),
		ValidaHasta:                           decision.ValidaHasta.UTC().Format(formatoInstanteMicrosegundo),
		AtestacionPDP: atestacionPDPDiarioPostgreSQL{
			DecisionRef: a.DecisionRef, AtestacionRef: a.AtestacionRef, Version: a.VersionAtestacion,
			Estado: a.EstadoAtestacion, HuellaAtestacionSHA256: a.HuellaAtestacionSHA256,
			VerificadorRef: a.VerificadorRef,
			VerificadaEn:   a.VerificadaEn.UTC().Format(formatoInstanteMicrosegundo),
		},
	}
}

func restaurarDecisionDiarioPostgreSQL(
	decision decisionDiarioPostgreSQL,
) (gobiernoconvocatorias.ProyeccionDecisionDiario, error) {
	emitida, errEmitida := instanteJSONDiarioPostgreSQL(decision.EmitidaEn)
	verificada, errVerificada := instanteJSONDiarioPostgreSQL(decision.VerificadaEn)
	validaHasta, errValida := instanteJSONDiarioPostgreSQL(decision.ValidaHasta)
	atestacionVerificada, errAtestacion := instanteJSONDiarioPostgreSQL(decision.AtestacionPDP.VerificadaEn)
	if errors.Join(errEmitida, errVerificada, errValida, errAtestacion) != nil {
		return gobiernoconvocatorias.ProyeccionDecisionDiario{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	resultado := gobiernoconvocatorias.ProyeccionDecisionDiario{
		EsquemaHuella: decision.EsquemaHuella, DecisionRef: decision.DecisionRef,
		HuellaDecisionSHA256: decision.HuellaDecisionSHA256, Accion: decision.Accion,
		RecursoRef: decision.RecursoRef, ModuloID: decision.ModuloID, TipoRecurso: decision.TipoRecurso,
		ContextoRecursoHuellaSHA256: decision.ContextoRecursoHuellaSHA256, Finalidad: decision.Finalidad,
		AsignacionRef: decision.AsignacionRef, AsignacionHuellaSHA256: decision.AsignacionHuellaSHA256,
		VersionRolRef: decision.VersionRolRef, VersionRolHuellaSHA256: decision.VersionRolHuellaSHA256,
		ControlVigenciaVersionRolRef:          decision.ControlVigenciaVersionRolRef,
		ControlVigenciaVersionRolRevision:     decision.ControlVigenciaVersionRolRevision,
		ControlVigenciaVersionRolHuellaSHA256: decision.ControlVigenciaVersionRolHuellaSHA256,
		RevisionCatalogoPoliticas:             decision.RevisionCatalogoPoliticas,
		CatalogoPoliticasHuellaSHA256:         decision.CatalogoPoliticasHuellaSHA256,
		EmitidaEn:                             emitida, VerificadaEn: verificada, ValidaHasta: validaHasta,
		AtestacionPDP: gobiernoconvocatorias.ProyeccionAtestacionPDP{
			DecisionRef:            decision.AtestacionPDP.DecisionRef,
			AtestacionRef:          decision.AtestacionPDP.AtestacionRef,
			VersionAtestacion:      decision.AtestacionPDP.Version,
			EstadoAtestacion:       decision.AtestacionPDP.Estado,
			HuellaAtestacionSHA256: decision.AtestacionPDP.HuellaAtestacionSHA256,
			VerificadorRef:         decision.AtestacionPDP.VerificadorRef, VerificadaEn: atestacionVerificada,
		},
	}
	if resultado.Validar() != nil {
		return gobiernoconvocatorias.ProyeccionDecisionDiario{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return resultado, nil
}

func serializarContextoRecursoBorradorPostgreSQL(recurso dominiovec.RecursoAutorizable) ([]byte, error) {
	if recurso.Validar() != nil {
		return nil, gobiernoconvocatorias.ErrReservaBorradorInvalida
	}
	clonar := func(origen map[string]string) map[string]string {
		copia := make(map[string]string, len(origen))
		for clave, valor := range origen {
			copia[clave] = valor
		}
		return copia
	}
	contenido, err := json.Marshal(contextoRecursoBorradorPostgreSQL{
		Ambitos: clonar(recurso.Ambitos), Atributos: clonar(recurso.Atributos),
	})
	if err != nil {
		return nil, gobiernoconvocatorias.ErrReservaBorradorInvalida
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || huellaBytesDiarioPostgreSQL(contenido) != huella {
		borrarBytesDiarioPostgreSQL(contenido)
		return nil, gobiernoconvocatorias.ErrReservaBorradorInvalida
	}
	return contenido, nil
}

func decodificarJSONCerradoDiarioPostgreSQL(contenido []byte, destino any) error {
	if len(contenido) == 0 || len(contenido) > 4<<20 || destino == nil || jsonDiarioPostgreSQLNoAmbiguo(contenido) != nil {
		return gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	decodificador.UseNumber()
	if err := decodificador.Decode(destino); err != nil {
		return gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	var resto any
	if err := decodificador.Decode(&resto); !errors.Is(err, io.EOF) {
		return gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return nil
}

func jsonDiarioPostgreSQLNoAmbiguo(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.UseNumber()
	if err := consumirValorJSONDiarioPostgreSQL(decodificador, 0); err != nil {
		return err
	}
	if _, err := decodificador.Token(); !errors.Is(err, io.EOF) {
		return gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return nil
}

func consumirValorJSONDiarioPostgreSQL(decodificador *json.Decoder, profundidad int) error {
	if profundidad > 32 {
		return gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	token, err := decodificador.Token()
	if err != nil {
		return gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	delimitador, compuesto := token.(json.Delim)
	if !compuesto {
		return nil
	}
	switch delimitador {
	case '{':
		claves := make(map[string]struct{})
		for decodificador.More() {
			tokenClave, err := decodificador.Token()
			clave, valida := tokenClave.(string)
			if err != nil || !valida {
				return gobiernoconvocatorias.ErrResultadoBorradorInseguro
			}
			if _, duplicada := claves[clave]; duplicada {
				return gobiernoconvocatorias.ErrResultadoBorradorInseguro
			}
			claves[clave] = struct{}{}
			if err := consumirValorJSONDiarioPostgreSQL(decodificador, profundidad+1); err != nil {
				return err
			}
		}
		cierre, err := decodificador.Token()
		if err != nil || cierre != json.Delim('}') {
			return gobiernoconvocatorias.ErrResultadoBorradorInseguro
		}
	case '[':
		for decodificador.More() {
			if err := consumirValorJSONDiarioPostgreSQL(decodificador, profundidad+1); err != nil {
				return err
			}
		}
		cierre, err := decodificador.Token()
		if err != nil || cierre != json.Delim(']') {
			return gobiernoconvocatorias.ErrResultadoBorradorInseguro
		}
	default:
		return gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return nil
}

func instanteJSONDiarioPostgreSQL(valor string) (time.Time, error) {
	instante, err := time.Parse(formatoInstanteMicrosegundo, valor)
	if err != nil || instante.Location() != time.UTC || instante.Nanosecond()%1_000 != 0 ||
		instante.Format(formatoInstanteMicrosegundo) != valor {
		return time.Time{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return instante, nil
}

func huellaBytesDiarioPostgreSQL(contenido []byte) string {
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:])
}

func borrarBytesDiarioPostgreSQL(valores ...[]byte) {
	for _, valor := range valores {
		for indice := range valor {
			valor[indice] = 0
		}
	}
}

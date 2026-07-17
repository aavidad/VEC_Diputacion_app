package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"
)

const (
	// EsquemaHuellaSolicitudAutorizacionV2 identifica un documento canonico
	// cerrado. La huella resultante acredita solo integridad estructural: no es
	// una firma ni demuestra que la solicitud proceda del PDP o del registro.
	EsquemaHuellaSolicitudAutorizacionV2 = "vec.autorizacion.solicitud.v2.efectiva-minimizada"
	// EsquemaHuellaMotivoAutorizacionV2 compromete una referencia completa a
	// una entrada de catalogo: identificador, version, huella y clave. No
	// constituye una firma ni demuestra por si sola que el catalogo exista,
	// este publicado o proceda de una frontera confiable.
	EsquemaHuellaMotivoAutorizacionV2      = "vec.autorizacion.motivo.v2.referencia-opaca-catalogada"
	formatoInstanteSolicitudAutorizacionV2 = "2006-01-02T15:04:05.000000Z"
)

type motivoAutorizacionCanonicoV2 struct {
	Esquema    string                                 `json:"esquema"`
	Referencia referenciaMotivoAutorizacionCanonicaV2 `json:"referencia"`
}

type referenciaMotivoAutorizacionCanonicaV2 struct {
	CatalogoID           string `json:"catalogo_id"`
	CatalogoVersion      int    `json:"catalogo_version"`
	CatalogoHuellaSHA256 string `json:"catalogo_huella_sha256"`
	EntradaClave         string `json:"entrada_clave"`
}

type entradaSolicitudAutorizacionCanonicaV2 struct {
	Clave string `json:"clave"`
	Valor string `json:"valor"`
}

type principalSolicitudAutorizacionCanonicoV2 struct {
	ID string `json:"id"`
}

type recursoSolicitudAutorizacionCanonicoV2 struct {
	Referencia string                                   `json:"referencia"`
	ModuloID   string                                   `json:"modulo_id"`
	Tipo       string                                   `json:"tipo"`
	Ambitos    []entradaSolicitudAutorizacionCanonicaV2 `json:"ambitos"`
	Atributos  []entradaSolicitudAutorizacionCanonicaV2 `json:"atributos"`
}

// contextoActorSolicitudAutorizacionCanonicoV2 no copia el documento vivo del
// actor. VinculoAutenticacionActorV1 ya fija su referencia, version y huella
// canonica completa, y ValidarPara comprueba esa huella cuando el contexto esta
// presente en la frontera confiable.
type contextoActorSolicitudAutorizacionCanonicoV2 struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type vinculoSolicitudAutorizacionCanonicoV2 struct {
	BloqueVersion                uint16                         `json:"bloque_version"`
	AutenticacionRef             string                         `json:"autenticacion_ref"`
	AutenticacionHuellaSHA256    string                         `json:"autenticacion_huella_sha256"`
	AsercionRef                  string                         `json:"asercion_ref"`
	SesionRef                    string                         `json:"sesion_ref"`
	ControlSesionRef             string                         `json:"control_sesion_ref"`
	ControlSesionRevision        uint64                         `json:"control_sesion_revision"`
	ControlSesionHuellaSHA256    string                         `json:"control_sesion_huella_sha256"`
	CuentaRef                    string                         `json:"cuenta_ref"`
	CuentaOrdinariaRef           string                         `json:"cuenta_ordinaria_ref"`
	PrincipalID                  string                         `json:"principal_id"`
	PerfilActivoRef              string                         `json:"perfil_activo_ref"`
	CuentaPrivilegiada           bool                           `json:"cuenta_privilegiada"`
	Superficie                   SuperficieAutenticacionActorV1 `json:"superficie"`
	MetodoObservado              AuthMethod                     `json:"metodo_observado"`
	GarantiaObservada            AuthAssurance                  `json:"garantia_observada"`
	PoliticaGarantiaRef          string                         `json:"politica_garantia_ref"`
	PoliticaGarantiaHuellaSHA256 string                         `json:"politica_garantia_huella_sha256"`
	AutenticacionVerificadaEn    string                         `json:"autenticacion_verificada_en"`
	SesionEmitidaEn              string                         `json:"sesion_emitida_en"`
	SesionValidaHasta            string                         `json:"sesion_valida_hasta"`
	SesionRevalidadaEn           string                         `json:"sesion_revalidada_en"`
	ContextoActorRef             string                         `json:"contexto_actor_ref"`
	ContextoActorVersion         uint64                         `json:"contexto_actor_version"`
	ContextoActorHuellaSHA256    string                         `json:"contexto_actor_huella_sha256"`
}

// solicitudAutorizacionCanonicaV2 es un DTO congelado. No debe sustituirse por
// json.Marshal(SolicitudAutorizacion): sus tipos vivos pueden crecer e incluir
// datos que nunca deben copiarse a logs, decisiones o evidencias.
type solicitudAutorizacionCanonicaV2 struct {
	Esquema          string                                       `json:"esquema"`
	Principal        principalSolicitudAutorizacionCanonicoV2     `json:"principal"`
	PerfilActivoRef  string                                       `json:"perfil_activo_ref"`
	ContextoActor    contextoActorSolicitudAutorizacionCanonicoV2 `json:"contexto_actor"`
	Vinculo          vinculoSolicitudAutorizacionCanonicoV2       `json:"vinculo_autenticacion_actor"`
	Accion           string                                       `json:"accion"`
	Recurso          recursoSolicitudAutorizacionCanonicoV2       `json:"recurso"`
	Finalidad        string                                       `json:"finalidad"`
	CorrelacionRef   string                                       `json:"correlacion_ref"`
	ReferenciaMotivo referenciaMotivoAutorizacionCanonicaV2       `json:"referencia_motivo"`
}

// representacionCanonicaSolicitudAutorizacionV2 queda deliberadamente privada:
// contiene referencias internas de la solicitud efectiva. Solo su huella puede
// cruzar la frontera del dominio o persistirse en una decision. Nombre, correo,
// roles, permisos y atributos declarados del principal se excluyen por
// construccion: no intervienen en el PDP V2 y hashearlos no los anonimizaria.
func representacionCanonicaSolicitudAutorizacionV2(s SolicitudAutorizacionLigadaV2) ([]byte, error) {
	datos, err := s.Datos()
	if err != nil {
		return nil, ErrSolicitudAutorizacionLigadaV2Invalida
	}
	datosVinculo, err := datos.VinculoAutenticacionActor.Datos()
	if err != nil {
		return nil, ErrSolicitudAutorizacionLigadaV2Invalida
	}

	documento := solicitudAutorizacionCanonicaV2{
		Esquema:         EsquemaHuellaSolicitudAutorizacionV2,
		Principal:       principalSolicitudAutorizacionCanonicoV2{ID: datosVinculo.PrincipalID},
		PerfilActivoRef: datosVinculo.PerfilActivoRef,
		ContextoActor: contextoActorSolicitudAutorizacionCanonicoV2{
			Referencia: datosVinculo.ContextoActorRef, Version: datosVinculo.ContextoActorVersion,
			HuellaSHA256: datosVinculo.ContextoActorHuellaSHA256,
		},
		Vinculo: vinculoSolicitudAutorizacionCanonicoV2DesdeDatos(datosVinculo),
		Accion:  datos.Accion,
		Recurso: recursoSolicitudAutorizacionCanonicoV2{
			Referencia: datos.Recurso.Referencia, ModuloID: datos.Recurso.ModuloID, Tipo: datos.Recurso.Tipo,
			Ambitos:   entradasSolicitudAutorizacionCanonicasV2(datos.Recurso.Ambitos),
			Atributos: entradasSolicitudAutorizacionCanonicasV2(datos.Recurso.Atributos),
		},
		Finalidad: datos.Finalidad, CorrelacionRef: datos.CorrelacionRef,
		ReferenciaMotivo: referenciaMotivoAutorizacionCanonicaV2Desde(datos.ReferenciaMotivo),
	}
	contenido, err := json.Marshal(documento)
	if err != nil {
		return nil, ErrSolicitudAutorizacionInvalida
	}
	return contenido, nil
}

func HuellaSHA256SolicitudAutorizacionV2(s SolicitudAutorizacionLigadaV2) (string, error) {
	contenido, err := representacionCanonicaSolicitudAutorizacionV2(s)
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func contextoActorSolicitudAutorizacionAusenteV2(c ContextoActor) bool {
	p := c.Principal
	i := c.Instantanea
	return p.ID == "" && p.DisplayName == "" && p.Email == "" &&
		len(p.Roles) == 0 && len(p.Permissions) == 0 && p.AuthMethod == "" &&
		p.AuthAssurance == "" && len(p.Attributes) == 0 &&
		c.PerfilActivoRef == "" && c.PersonaRef == "" && c.ResueltoEn.IsZero() &&
		i.VinculoRef == "" && i.VinculoVersion == 0 && i.CuentaRef == "" &&
		i.PersonaRef == "" && i.PersonaVersion == 0 && i.PerfilActivoRef == "" &&
		i.PerfilVersion == 0 && i.Estado == "" && i.VigenteDesde.IsZero() &&
		i.VigenteHasta.IsZero() && len(i.Vinculos) == 0
}

// HuellaSHA256MotivoAutorizacionV2 calcula el compromiso que puede cotejar un
// adaptador durable. No recibe ni persiste texto libre: compromete la referencia
// integra a una entrada catalogada ya resuelta por una frontera confiable.
func HuellaSHA256MotivoAutorizacionV2(referencia ReferenciaEntradaCatalogo) (string, error) {
	if !ReferenciaMotivoAutorizacionV2Valida(referencia) {
		return "", ErrSolicitudAutorizacionInvalida
	}
	contenido, err := json.Marshal(motivoAutorizacionCanonicoV2{
		Esquema:    EsquemaHuellaMotivoAutorizacionV2,
		Referencia: referenciaMotivoAutorizacionCanonicaV2Desde(referencia),
	})
	if err != nil {
		return "", ErrSolicitudAutorizacionInvalida
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func referenciaMotivoAutorizacionCanonicaV2Desde(
	referencia ReferenciaEntradaCatalogo,
) referenciaMotivoAutorizacionCanonicaV2 {
	return referenciaMotivoAutorizacionCanonicaV2{
		CatalogoID: referencia.CatalogoID, CatalogoVersion: referencia.CatalogoVersion,
		CatalogoHuellaSHA256: referencia.CatalogoHuellaSHA256,
		EntradaClave:         referencia.EntradaClave,
	}
}

func entradasSolicitudAutorizacionCanonicasV2(valores map[string]string) []entradaSolicitudAutorizacionCanonicaV2 {
	claves := make([]string, 0, len(valores))
	for clave := range valores {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	resultado := make([]entradaSolicitudAutorizacionCanonicaV2, 0, len(claves))
	for _, clave := range claves {
		resultado = append(resultado, entradaSolicitudAutorizacionCanonicaV2{Clave: clave, Valor: valores[clave]})
	}
	return resultado
}

func vinculoSolicitudAutorizacionCanonicoV2DesdeDatos(
	v DatosVinculoAutenticacionActorV1,
) vinculoSolicitudAutorizacionCanonicoV2 {
	return vinculoSolicitudAutorizacionCanonicoV2{
		BloqueVersion: v.BloqueVersion, AutenticacionRef: v.AutenticacionRef,
		AutenticacionHuellaSHA256: v.AutenticacionHuellaSHA256, AsercionRef: v.AsercionRef,
		SesionRef: v.SesionRef, ControlSesionRef: v.ControlSesionRef,
		ControlSesionRevision: v.ControlSesionRevision, ControlSesionHuellaSHA256: v.ControlSesionHuellaSHA256,
		CuentaRef: v.CuentaRef, CuentaOrdinariaRef: v.CuentaOrdinariaRef,
		PrincipalID: v.PrincipalID, PerfilActivoRef: v.PerfilActivoRef,
		CuentaPrivilegiada: v.CuentaPrivilegiada, Superficie: v.Superficie,
		MetodoObservado: v.MetodoObservado, GarantiaObservada: v.GarantiaObservada,
		PoliticaGarantiaRef:          v.PoliticaGarantiaRef,
		PoliticaGarantiaHuellaSHA256: v.PoliticaGarantiaHuellaSHA256,
		AutenticacionVerificadaEn:    instanteSolicitudAutorizacionV2(v.AutenticacionVerificadaEn),
		SesionEmitidaEn:              instanteSolicitudAutorizacionV2(v.SesionEmitidaEn),
		SesionValidaHasta:            instanteSolicitudAutorizacionV2(v.SesionValidaHasta),
		SesionRevalidadaEn:           instanteSolicitudAutorizacionV2(v.SesionRevalidadaEn),
		ContextoActorRef:             v.ContextoActorRef, ContextoActorVersion: v.ContextoActorVersion,
		ContextoActorHuellaSHA256: v.ContextoActorHuellaSHA256,
	}
}

func instanteSolicitudAutorizacionV2(instante time.Time) string {
	return instante.UTC().Format(formatoInstanteSolicitudAutorizacionV2)
}
